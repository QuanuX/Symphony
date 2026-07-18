package client

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/config"
	ssiagpaths "github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/paths"
)

const testTOPSID = "018f0c3a-7b2d-7e11-8c12-0242ac120002"

func TestClientNewForTOPS(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_RUNTIME_DIR", filepath.Join(home, "runtime"))

	layout, err := ssiagpaths.ResolveInstance(ssiagpaths.ScopeUser, testTOPSID)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Dir(layout.ConfigFile), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(layout.Socket), 0700); err != nil {
		t.Fatal(err)
	}

	// 1. Config lacking service identity
	cfg := config.Config{
		Schema: "symphony.ssiag.config.v1",
		Mode:   "user",
		TOPS:   config.TOPSConfig{ID: testTOPSID, Name: "Test TOPS"},
		Listen: config.ListenConfig{Network: "unix", Address: layout.Socket},
	}
	writeConfig(t, layout.ConfigFile, cfg)

	_, err = NewForTOPS(ssiagpaths.ScopeUser, testTOPSID, 2*time.Second)
	if err == nil {
		t.Fatal("expected failure for config lacking service identity")
	}

	// 2. Valid config with a canonical service identity.
	currentUID := uint32(os.Geteuid())
	currentGID := uint32(os.Getegid())
	cfg.Authentication = &config.AuthenticationConfig{
		Mechanism: "unix_peer_credentials",
		Service: &config.SubjectConfig{
			ID: config.ServiceSubjectID, Kind: config.ServiceSubjectKind, UID: &currentUID, GID: &currentGID,
		},
		Subjects: []config.SubjectConfig{},
	}
	writeConfig(t, layout.ConfigFile, cfg)

	// Now try creating client
	c, err := NewForTOPS(ssiagpaths.ScopeUser, testTOPSID, 2*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil client")
	}

	// 3. Unsafe config permissions (group writable)
	if err := os.Chmod(layout.ConfigFile, 0620); err != nil {
		t.Fatal(err)
	}
	_, err = NewForTOPS(ssiagpaths.ScopeUser, testTOPSID, 2*time.Second)
	if err == nil {
		t.Fatal("expected failure for group-writable configuration file")
	}
	_ = os.Chmod(layout.ConfigFile, 0600) // restore

	// 4. Config is symlink
	if err := os.Remove(layout.ConfigFile); err != nil {
		t.Fatal(err)
	}
	targetFile := filepath.Join(home, "real-config.json")
	writeConfig(t, targetFile, cfg)
	if err := os.Symlink(targetFile, layout.ConfigFile); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	_, err = NewForTOPS(ssiagpaths.ScopeUser, testTOPSID, 2*time.Second)
	if err == nil {
		t.Fatal("expected failure for symlinked configuration file")
	}
}

func TestPeerVerificationFailsBeforeHTTPBytes(t *testing.T) {
	_, layout, _ := clientFixture(t)
	listener, err := net.Listen("unix", layout.Socket)
	if err != nil {
		t.Skipf("Unix sockets unavailable: %v", err)
	}
	defer listener.Close()

	readResult := make(chan int, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			readResult <- -1
			return
		}
		defer conn.Close()
		_ = conn.SetReadDeadline(time.Now().Add(time.Second))
		buffer := make([]byte, 1)
		n, _ := conn.Read(buffer)
		readResult <- n
	}()

	client, err := newForTOPS(ssiagpaths.ScopeUser, testTOPSID, time.Second, func(net.Conn, uint32, uint32) error {
		return errors.New("deliberate peer mismatch")
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Status(context.Background())
	if err == nil || !strings.Contains(err.Error(), "deliberate peer mismatch") {
		t.Fatalf("expected peer mismatch, got %v", err)
	}
	if n := <-readResult; n != 0 {
		t.Fatalf("server received %d HTTP bytes before peer verification", n)
	}
}

func TestSocketOverrideRetainsPeerVerification(t *testing.T) {
	home, _, _ := clientFixture(t)
	override := filepath.Join(home, "override.sock")
	listener, err := net.Listen("unix", override)
	if err != nil {
		t.Skipf("Unix sockets unavailable: %v", err)
	}
	defer listener.Close()
	t.Setenv("SYMPHONY_SSIAG_SOCKET", override)

	done := make(chan struct{})
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr == nil {
			_ = conn.Close()
		}
		close(done)
	}()
	called := false
	client, err := newForTOPS(ssiagpaths.ScopeUser, testTOPSID, time.Second, func(_ net.Conn, uid, gid uint32) error {
		called = true
		if uid != uint32(os.Geteuid()) || gid != uint32(os.Getegid()) {
			t.Fatalf("override changed expected identity to uid=%d gid=%d", uid, gid)
		}
		return errors.New("verification sentinel")
	})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = client.Status(context.Background())
	<-done
	if !called {
		t.Fatal("socket override bypassed peer verification")
	}
}

func clientFixture(t *testing.T) (string, ssiagpaths.InstanceLayout, config.Config) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_RUNTIME_DIR", filepath.Join(home, "runtime"))
	layout, err := ssiagpaths.ResolveInstance(ssiagpaths.ScopeUser, testTOPSID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(layout.ConfigFile), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(layout.Socket), 0o700); err != nil {
		t.Fatal(err)
	}
	uid, gid := uint32(os.Geteuid()), uint32(os.Getegid())
	cfg := config.Config{
		Schema: "symphony.ssiag.config.v1", Mode: "user",
		TOPS:   config.TOPSConfig{ID: testTOPSID, Name: "Test TOPS"},
		Listen: config.ListenConfig{Network: "unix", Address: layout.Socket},
		Authentication: &config.AuthenticationConfig{
			Mechanism: "unix_peer_credentials",
			Service:   &config.SubjectConfig{ID: config.ServiceSubjectID, Kind: config.ServiceSubjectKind, UID: &uid, GID: &gid},
			Subjects:  []config.SubjectConfig{},
		},
		Providers: []config.ProviderConfig{},
	}
	writeConfig(t, layout.ConfigFile, cfg)
	return home, layout, cfg
}

func writeConfig(t *testing.T, path string, cfg config.Config) {
	t.Helper()
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}
