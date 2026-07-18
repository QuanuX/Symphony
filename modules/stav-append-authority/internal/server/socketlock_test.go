//go:build darwin || linux

package server

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSocketLeaseIsExclusiveAndPersistent(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "append.sock")
	first, err := acquireSocketLease(socket)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := acquireSocketLease(socket); err == nil {
		t.Fatal("second process acquired the same STAV socket lifecycle lease")
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	second, err := acquireSocketLease(socket)
	if err != nil {
		t.Fatalf("released STAV socket lifecycle lease was not reusable: %v", err)
	}
	if err := second.Close(); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Lstat(socket + ".lock"); err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		t.Fatalf("unsafe persistent STAV socket lock: info=%v error=%v", info, err)
	}
}

func TestSocketLeaseRejectsSymlink(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "append.sock")
	target := filepath.Join(t.TempDir(), "target")
	if err := os.WriteFile(target, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, socket+".lock"); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := acquireSocketLease(socket); err == nil {
		t.Fatal("STAV socket lease followed a symbolic link")
	}
}
