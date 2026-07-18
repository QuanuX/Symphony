//go:build integration && (darwin || linux)

package server

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/config"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/provider"
)

func TestDistinctOSIdentityCannotStartSSIAG(t *testing.T) {
	if os.Getenv("SYMPHONY_SSIAG_IDENTITY_HELPER") == "1" {
		expectedUID, _ := strconv.ParseUint(os.Getenv("SYMPHONY_SSIAG_EXPECTED_UID"), 10, 32)
		expectedGID, _ := strconv.ParseUint(os.Getenv("SYMPHONY_SSIAG_EXPECTED_GID"), 10, 32)
		uid, gid := uint32(expectedUID), uint32(expectedGID)
		cfg := config.Config{
			Schema: "symphony.ssiag.config.v1", Mode: "development",
			TOPS:           config.TOPSConfig{ID: testTOPSID, Name: "Distinct identity test"},
			Listen:         config.ListenConfig{Network: "unix", Address: os.Getenv("SYMPHONY_SSIAG_TEST_SOCKET")},
			Authentication: serviceAuthentication(uid, gid), Providers: []config.ProviderConfig{},
		}
		registry, _ := provider.New(nil)
		service, err := New(cfg, registry)
		if err == nil {
			err = service.Run(context.Background())
		}
		if err == nil || !strings.Contains(err.Error(), "process identity mismatch") {
			fmt.Fprintf(os.Stderr, "expected process identity mismatch, got %v\n", err)
			os.Exit(2)
		}
		return
	}
	if os.Geteuid() != 0 {
		t.Skip("distinct-identity integration test requires administrator execution")
	}
	nobody, err := user.Lookup("nobody")
	if err != nil {
		t.Skipf("nobody account is unavailable: %v", err)
	}
	uid, err := strconv.ParseUint(nobody.Uid, 10, 32)
	if err != nil {
		t.Fatal(err)
	}
	gid, err := strconv.ParseUint(nobody.Gid, 10, 32)
	if err != nil {
		t.Fatal(err)
	}
	root, err := os.MkdirTemp("/tmp", "symphony-ssiag-distinct-identity-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)
	if err := os.Chmod(root, 0o755); err != nil {
		t.Fatal(err)
	}
	source, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	copyPath := filepath.Join(root, "ssiag-integration-test")
	if err := copyExecutable(source, copyPath); err != nil {
		t.Fatal(err)
	}
	socket := filepath.Join(root, "ssiag.sock")
	command := exec.Command(copyPath, "-test.run=TestDistinctOSIdentityCannotStartSSIAG")
	command.Env = append(os.Environ(),
		"SYMPHONY_SSIAG_IDENTITY_HELPER=1",
		fmt.Sprintf("SYMPHONY_SSIAG_EXPECTED_UID=%d", os.Geteuid()),
		fmt.Sprintf("SYMPHONY_SSIAG_EXPECTED_GID=%d", os.Getegid()),
		"SYMPHONY_SSIAG_TEST_SOCKET="+socket,
	)
	command.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid)}}
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("distinct identity helper: %v\n%s", err, output)
	}
	if _, err := os.Lstat(socket); !os.IsNotExist(err) {
		t.Fatalf("identity-mismatched process changed the socket path: %v", err)
	}
}

func copyExecutable(source, destination string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
