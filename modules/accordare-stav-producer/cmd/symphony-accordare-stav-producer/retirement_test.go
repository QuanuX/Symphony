package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/packageinstall"
	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/paths"
	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/version"
)

// The installed test executable exercises the actual native command handlers
// in a disposable process. No host service manager or live provider is used.
func TestNativeRetirementCommandChild(t *testing.T) {
	encoded := os.Getenv("ACCORDare_RETIREMENT_TEST_COMMAND")
	if encoded == "" {
		return
	}
	var args []string
	if err := json.Unmarshal([]byte(encoded), &args); err != nil {
		t.Fatal(err)
	}
	if err := run(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func nativeTestCommand(binary string, args ...string) *exec.Cmd {
	data, _ := json.Marshal(args)
	cmd := exec.Command(binary, "-test.run=^TestNativeRetirementCommandChild$")
	cmd.Env = append(os.Environ(), "ACCORDare_RETIREMENT_TEST_COMMAND="+string(data))
	return cmd
}

func TestNativeServiceRetirementAdmissionAndRecovery(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("system reclamation remains unqualified")
	}
	root, err := os.MkdirTemp("/tmp", "a")
	if err != nil {
		t.Fatal(err)
	}
	runtimeBase := root // Keep the disposable Unix socket within sockaddr_un's bound.
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(root) })
	for name, suffix := range map[string]string{"HOME": "home", "XDG_CONFIG_HOME": "config", "XDG_STATE_HOME": "state", "XDG_RUNTIME_DIR": "runtime"} {
		t.Setenv(name, filepath.Join(root, suffix))
	}
	t.Setenv("XDG_RUNTIME_DIR", runtimeBase)
	source, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	installed, err := packageinstall.Install(source, filepath.Join(root, "prefix"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	tops := "11111111-1111-4111-8111-111111111111"
	uid, gid := uint64(os.Geteuid()), uint64(os.Getegid())
	stav := stavprotocol.AppendAuthorityConfig{Authentication: stavprotocol.AppendAuthorityAuthentication{Authority: stavprotocol.AuthorityGrant{UID: uid, GID: gid, Subject: stavprotocol.SafeReference{ID: "retirement-test-stav", Kind: "symphony.identity.service"}}, Mechanism: "kernel-peer-credentials", Producers: []stavprotocol.ProducerGrant{}, Readers: []stavprotocol.ReaderGrant{}}, Ledger: stavprotocol.AppendAuthorityLedger{Durability: "fsync-before-receipt", MaxBytes: 1 << 20, Path: filepath.Join(root, "ledger"), Recovery: "preserve-incomplete-tail", Retention: "preserve_all", Rotation: "disabled"}, Listen: stavprotocol.AppendAuthorityListen{Address: filepath.Join(root, "absent-stav.sock"), Network: "unix"}, Mode: "user", Schema: stavprotocol.SchemaAppendAuthorityConfig, TOPSID: tops}
	data, err := stavprotocol.EncodeAppendAuthorityConfig(stav)
	if err != nil {
		t.Fatal(err)
	}
	stavPath := filepath.Join(root, "stav.json")
	if err := os.WriteFile(stavPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	enrollArgs := []string{"enroll", "--scope", "user", "--tops-id", tops, "--stav-config", stavPath}
	if output, err := nativeTestCommand(installed.Binary, enrollArgs...).CombinedOutput(); err != nil {
		t.Fatalf("native enrollment: %v %s", err, output)
	}
	layout, err := paths.Resolve(paths.ScopeUser, tops)
	if err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(root, "service.log")
	log, err := os.Create(logPath)
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	service := nativeTestCommand(installed.Binary, "serve", "--scope", "user", "--tops-id", tops)
	service.Stdout, service.Stderr = log, log
	if err := service.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- service.Wait() }()
	stopped := false
	stop := func() {
		if stopped {
			return
		}
		stopped = true
		_ = service.Process.Signal(os.Interrupt)
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("native service exit: %v", err)
			}
		case <-time.After(5 * time.Second):
			_ = service.Process.Kill()
			<-done
			t.Error("native service did not drain")
		}
	}
	t.Cleanup(stop)
	deadline := time.Now().Add(3 * time.Second)
	for {
		if info, err := os.Lstat(layout.Socket); err == nil && info.Mode()&os.ModeSocket != 0 {
			break
		}
		select {
		case err := <-done:
			stopped = true
			output, _ := os.ReadFile(logPath)
			t.Fatalf("native startup: %v %s", err, output)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("native service did not become ready")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := packageinstall.Uninstall(installed.Prefix, version.Version); err == nil {
		t.Fatal("live native process allowed reclamation")
	}
	for _, args := range [][]string{enrollArgs, {"serve", "--scope", "user", "--tops-id", tops}, {"unenroll", "--scope", "user", "--tops-id", tops}, {"supervisor", "install", "--scope", "user", "--tops-id", tops, "--force", "--no-start"}} {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		command := nativeTestCommand(installed.Binary, args...)
		// These refusals complete immediately; the context bounds the test if a
		// regression accidentally starts another native service.
		bounded := exec.CommandContext(ctx, command.Path, command.Args[1:]...)
		bounded.Env = command.Env
		output, err := bounded.CombinedOutput()
		cancel()
		if err == nil {
			t.Fatalf("retiring package admitted %v: %s", args, output)
		}
		if !strings.Contains(string(output), "package") {
			t.Fatalf("unexpected admission failure for %v: %s", args, output)
		}
	}
	if _, err := os.Stat(layout.ConfigFile); err != nil {
		t.Fatal("live unenrollment removed recovery configuration")
	}
	stop()
	if output, err := nativeTestCommand(installed.Binary, "unenroll", "--scope", "user", "--tops-id", tops).CombinedOutput(); err != nil {
		t.Fatalf("retiring maintenance could not remove empty enrollment: %v %s", err, output)
	}
	if _, err := packageinstall.Uninstall(installed.Prefix, version.Version); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{installed.Binary, installed.Receipt} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("retired package file remains: %s %v", path, err)
		}
	}
}
