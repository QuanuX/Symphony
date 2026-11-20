//go:build linux && symphony_node_keyring_fixture

// This test-only executable exercises the installed node package using disposable
// service accounts. It has no third-party dependencies and is never installed.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"math"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const collection = "/org/freedesktop/secrets/collection/login"

func require(ok bool, what string) {
	if !ok {
		panic(what)
	}
}

func checked[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}
	return value
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func randomBytes(n int) []byte {
	b := make([]byte, n)
	_, err := rand.Read(b)
	must(err)
	return b
}

func uuid() string {
	b := randomBytes(16)
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:])
}

type account struct {
	uid, gid   uint32
	name, home string
}

func parseFixtureID(raw string) (uint32, error) {
	id, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return 0, err
	}
	// The fixture drops privileges to a disposable account. Exclude root,
	// chown's leave-unchanged sentinel, and IDs that os.Chown cannot represent.
	if id == 0 || id >= math.MaxUint32 || id > math.MaxInt || strconv.FormatUint(id, 10) != raw {
		return 0, fmt.Errorf("invalid disposable account ID %q", raw)
	}
	return uint32(id), nil
}

func disposableNode() {
	_, err := os.Stat("/.dockerenv")
	require(err == nil && os.Geteuid() == 0, "use a fresh root container with --network none")
	active := 0
	for _, device := range checked(net.Interfaces()) {
		if device.Flags&net.FlagUp != 0 {
			require(device.Name == "lo", "fixture requires external networking disabled")
			active++
		}
	}
	require(active == 1, "fixture requires only loopback")
	routes := checked(os.ReadFile("/proc/net/route"))
	require(len(strings.Split(strings.TrimSpace(string(routes)), "\n")) == 1, "unexpected IPv4 routes")
}

func addAccount(name string) account {
	nothing := process{}
	nothing.success(nil, "useradd", "--create-home", "--shell", "/bin/sh", "--password", "*", name)
	u := checked(user.Lookup(name))
	return account{checked(parseFixtureID(u.Uid)), checked(parseFixtureID(u.Gid)), name, u.HomeDir}
}

func (a account) credentials() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Credential: &syscall.Credential{
		Uid: a.uid, Gid: a.gid, Groups: []uint32{a.gid},
	}}
}

func (a account) mkdir(path string) {
	must(os.Mkdir(path, 0700))
	a.chown(path)
}

func (a account) chown(path string) {
	must(os.Chown(path, int(a.uid), int(a.gid)))
}

func (a account) environment() []string {
	return []string{"HOME=" + a.home, "PATH=/usr/bin:/bin", "USER=" + a.name, "LOGNAME=" + a.name}
}

type process struct {
	env       []string
	identity  *syscall.SysProcAttr
	protected [][]byte
}

type result struct {
	code           int
	stdout, stderr string
}

func (p process) command(ctx context.Context, input []byte, args []string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Env = p.env
	cmd.SysProcAttr = p.identity
	// os/exec provides a pipe, including for empty input.
	cmd.Stdin = bytes.NewReader(input)
	cmd.WaitDelay = 2 * time.Second
	return cmd
}

func (p process) run(input []byte, args ...string) result {
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()
	cmd := p.command(ctx, input, args)
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	err := cmd.Run()
	if ctx.Err() != nil {
		panic("fixture command exceeded deadline: " + filepath.Base(args[0]))
	}
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			panic(err)
		}
	}
	for _, secret := range p.protected {
		require(len(secret) > 0 && !bytes.Contains(out.Bytes(), secret) && !bytes.Contains(errOut.Bytes(), secret), "credential appeared in process output")
	}
	status := cmd.ProcessState.Sys().(syscall.WaitStatus)
	code := status.ExitStatus()
	if status.Signaled() {
		code = -int(status.Signal())
	}
	return result{code, out.String(), errOut.String()}
}

func (p process) success(input []byte, args ...string) result {
	r := p.run(input, args...)
	require(r.code == 0, fmt.Sprintf("%s exited %d", filepath.Base(args[0]), r.code))
	return r
}

type daemon struct {
	cmd  *exec.Cmd
	done chan struct{}
}

func (p process) start(input []byte, args ...string) *daemon {
	cmd := p.command(context.Background(), input, args)
	// Background services never relay credential-bearing output to the test log.
	must(cmd.Start())
	d := &daemon{cmd, make(chan struct{})}
	go func() { _ = cmd.Wait(); close(d.done) }()
	return d
}

func (d *daemon) stop() {
	select {
	case <-d.done:
		return
	default:
	}
	_ = d.cmd.Process.Signal(syscall.SIGTERM)
	select {
	case <-d.done:
	case <-time.After(5 * time.Second):
		_ = d.cmd.Process.Kill()
		<-d.done
		panic("fixture daemon failed to stop")
	}
}

func waitFor(probe func() bool, daemons ...*daemon) {
	until := time.Now().Add(10 * time.Second)
	for time.Now().Before(until) {
		for _, d := range daemons {
			select {
			case <-d.done:
				panic("fixture daemon exited before readiness")
			default:
			}
		}
		if probe() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	panic("fixture readiness timed out")
}

func isSocket(path string) bool {
	s, err := os.Lstat(path)
	return err == nil && s.Mode()&os.ModeSocket != 0
}

func frame(request, tops, ref string, value []byte) []byte {
	header := fmt.Sprintf("SNKI1 %s %s %s %d\n", request, tops, ref, len(value))
	return append([]byte(header), value...)
}

func generation(r result, request string) string {
	parts := strings.Split(strings.TrimSuffix(r.stdout, "\n"), " ")
	require(r.code == 0 && len(parts) == 3 && parts[0] == "STORED" && parts[1] == request, "missing import receipt")
	id := parts[2]
	require(len(id) == 36, "invalid generation")
	for i, c := range id {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			require(c == '-', "invalid UUID separator")
		} else {
			require(c >= '0' && c <= '9' || c >= 'a' && c <= 'f', "invalid UUID digit")
		}
	}
	require(r.stdout == "STORED "+request+" "+id+"\n", "invalid receipt terminator")
	return id
}

func binding(path string, a account, bus, tops, ref, gen string) {
	body := fmt.Sprintf("SYMPHONY-NODE-KEYRING-1\nuid=%d\nbus_socket=%s\ncollection=%s\ntops=%s\nreference=%s\ngeneration=%s\nmaximum_bytes=64\n", a.uid, bus, collection, tops, ref, gen)
	must(os.WriteFile(path, []byte(body), 0644))
	must(os.Chmod(path, 0644))
}

func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Fprintln(os.Stderr, "fixture failed:", err)
			os.Exit(1)
		}
	}()
	switch {
	case len(os.Args) == 3 && os.Args[1] == "import":
		importChecks(os.Args[2])
	case len(os.Args) == 4 && os.Args[1] == "ssh":
		sshChecks(os.Args[2], os.Args[3])
	case len(os.Args) == 5 && os.Args[1] == "restart":
		restartChecks(os.Args[2], os.Args[3], os.Args[4])
	default:
		panic("expected import RECEIVER, ssh PREFIX CONSUMER, or restart LAUNCHER RECEIVER CONSUMER")
	}
}
