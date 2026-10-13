//go:build darwin || linux

// The optional SSH carrier performs one explicit authenticated exchange.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/QuanuX/Symphony/tools/sov-ssh-adapter/contract"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Stdin, os.Stdout))
}
func sshOptions(e contract.Endpoint, known, identity string) []string {
	// All paths consumed by the SSH config parser are quoted independently of
	// remote POSIX-shell quoting. No ambient user/system configuration is loaded.
	q := func(s string) string { return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"` }
	a := []string{"-F", "none", "-S", "none", "-T", "-a", "-x", "-l", e.User, "-p", e.Port}
	for _, v := range []string{"BatchMode=yes", "IdentitiesOnly=yes", "PreferredAuthentications=publickey", "PasswordAuthentication=no", "KbdInteractiveAuthentication=no", "GSSAPIAuthentication=no", "HostbasedAuthentication=no", "PubkeyAuthentication=yes", "PKCS11Provider=none", "CertificateFile=none", "IdentityAgent=" + q(e.AgentSocket), "IdentityFile=" + q(identity), "StrictHostKeyChecking=yes", "HostKeyAlias=sov-selected-target", "UserKnownHostsFile=" + q(known), "GlobalKnownHostsFile=/dev/null", "KnownHostsCommand=none", "VerifyHostKeyDNS=no", "UpdateHostKeys=no", "CheckHostIP=no", "ClearAllForwardings=yes", "ForwardAgent=no", "ForwardX11=no", "PermitLocalCommand=no", "ControlMaster=no", "ProxyCommand=none", "ProxyJump=none", "RemoteCommand=none", "RequestTTY=no", "ConnectionAttempts=1", "ServerAliveInterval=0", "TCPKeepAlive=no", "LogLevel=ERROR"} {
		a = append(a, "-o", v)
	}
	if e.ConnectTimeoutSeconds != nil {
		a = append(a, "-o", "ConnectTimeout="+*e.ConnectTimeoutSeconds)
	}
	return append(a, e.Host)
}

type boundedBuffer struct{ bytes.Buffer }

// A system-signed macOS client may refuse relocation. This explicit profile
// trusts the host administrator's root-owned, non-writable, no-symlink tree.
// It detects drift by digest; it does not claim byte-copy isolation from root.
func protectedSystemPath(path string) error {
	for p := path; ; p = filepath.Dir(p) {
		st, err := os.Lstat(p)
		if err != nil {
			return err
		}
		s, ok := st.Sys().(*syscall.Stat_t)
		if !ok || s.Uid != 0 || st.Mode().Perm()&0022 != 0 || st.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("SSH system path is not protected")
		}
		if p == path {
			if !st.Mode().IsRegular() {
				return fmt.Errorf("SSH client is not regular")
			}
		} else if !st.IsDir() {
			return fmt.Errorf("SSH ancestor is not a directory")
		}
		if p == "/" {
			return nil
		}
	}
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	if int64(b.Len()+len(p)) > contract.MaxDocumentBytes {
		return 0, fmt.Errorf("reply too large")
	}
	return b.Buffer.Write(p)
}
func run(ctx context.Context, args []string, input io.Reader, output io.Writer) int {
	if len(args) != 0 {
		return 2
	}
	raw, err := io.ReadAll(io.LimitReader(input, contract.MaxDocumentBytes+1))
	if err != nil {
		return 2
	}
	in, err := contract.DecodeInput(raw)
	if err != nil {
		return 2
	}
	remote, invocationSHA, err := contract.TargetCommand(in)
	if err != nil {
		return 2
	}
	dir, err := os.MkdirTemp("", "sov-ssh-")
	if err != nil {
		return 2
	}
	defer os.RemoveAll(dir)
	client := filepath.Join(dir, "ssh")
	if in.Endpoint.SSHExecution == "protected_system_path" {
		if protectedSystemPath(in.Endpoint.SSH.Path) != nil {
			return 2
		}
		client = in.Endpoint.SSH.Path
	}
	src, err := os.OpenFile(in.Endpoint.SSH.Path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return 2
	}
	defer src.Close()
	st, err := src.Stat()
	if err != nil || !st.Mode().IsRegular() {
		return 2
	}
	h := sha256.New()
	if in.Endpoint.SSHExecution == "stage_copy" {
		dst, e := os.OpenFile(client, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0700)
		if e != nil {
			return 2
		}
		_, err = io.Copy(io.MultiWriter(dst, h), src)
		closeErr := dst.Close()
		if err != nil || closeErr != nil {
			return 2
		}
	} else {
		_, err = io.Copy(h, src)
	}
	if err != nil || fmt.Sprintf("%x", h.Sum(nil)) != in.Endpoint.SSH.SHA256 {
		return 2
	}
	known, identity := filepath.Join(dir, "known_hosts"), filepath.Join(dir, "identity.pub")
	if os.WriteFile(known, []byte("sov-selected-target "+in.Endpoint.HostPublicKey+"\n"), 0600) != nil || os.WriteFile(identity, []byte(in.Endpoint.ClientPublicKey+"\n"), 0600) != nil {
		return 2
	}
	argv := append(sshOptions(in.Endpoint, known, identity), remote)
	cmd := exec.CommandContext(ctx, client, argv...)
	cmd.Env = []string{"PATH=/usr/bin:/bin", "LANG=C", "LC_ALL=C"}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.WaitDelay = 250 * time.Millisecond
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM) }
	defer func() {
		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
	}()
	cmd.Stdin = bytes.NewReader(in.Invocation)
	cmd.Stderr = io.Discard
	var reply boundedBuffer
	cmd.Stdout = &reply
	if cmd.Run() != nil || contract.CheckJSON(reply.Bytes()) != nil {
		return 1
	}
	host, _ := contract.PublicKeyDigest(in.Endpoint.HostPublicKey)
	key, _ := contract.PublicKeyDigest(in.Endpoint.ClientPublicKey)
	result := contract.Reply{Protocol: contract.ReplyProtocol, EndpointSHA256: contract.EndpointDigest(in.Endpoint), InvocationSHA256: invocationSHA, HostKeySHA256: host, ClientKeySHA256: key, TargetResult: json.RawMessage(reply.Bytes())}
	if json.NewEncoder(output).Encode(result) != nil {
		return 1
	}
	return 0
}
