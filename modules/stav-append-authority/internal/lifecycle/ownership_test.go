package lifecycle

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	stavpaths "github.com/QuanuX/Symphony/modules/stav-append-authority/internal/paths"
)

func TestEnrollmentRejectsReservedOwnershipBeforeInstallationOrMutation(t *testing.T) {
	for _, ids := range [][2]uint64{{math.MaxUint32, 20}, {501, math.MaxUint32}, {math.MaxUint32 + 1, 20}, {501, math.MaxUint32 + 1}} {
		_, err := EnrollWithExecutable(stavpaths.ScopeSystem, enrollmentTOPSID, ids[0], ids[1], filepath.Join(t.TempDir(), "absent"))
		if err == nil || !strings.Contains(err.Error(), "is reserved") {
			t.Fatalf("identity %v reached installation or privilege handling: %v", ids, err)
		}
	}
}

func TestOwnedDirectoryRejectsReservedIdentityBeforeMutation(t *testing.T) {
	for _, ids := range [][2]uint32{{math.MaxUint32, 20}, {501, math.MaxUint32}} {
		path := filepath.Join(t.TempDir(), "must-remain-absent")
		called := false
		err := ensureOwnedPrivateDirectoryWithChown(path, ids[0], ids[1], func(string, int, int) error { called = true; return nil })
		if err == nil || called {
			t.Fatalf("reserved identity reached ownership mutation: called=%t error=%v", called, err)
		}
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("reserved identity created a directory: %v", err)
		}
	}
}

func TestOwnedDirectoryVerifiesOwnershipAfterChown(t *testing.T) {
	uid, gid := uint32(os.Geteuid()), uint32(os.Getegid())
	for _, ids := range [][2]uint32{{uid ^ 1, gid}, {uid, gid ^ 1}} {
		path := filepath.Join(t.TempDir(), "no-op-chown")
		called := false
		err := ensureOwnedPrivateDirectoryWithChown(path, ids[0], ids[1], func(string, int, int) error { called = true; return nil })
		if err == nil || !called || !strings.Contains(err.Error(), "does not match requested UID/GID") {
			t.Fatalf("successful no-op chown was not detected: called=%t error=%v", called, err)
		}
	}
	path := filepath.Join(t.TempDir(), "verified")
	for range 2 {
		if err := ensureOwnedPrivateDirectory(path, uid, gid); err != nil {
			t.Fatal(err)
		}
	}
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	stat := info.Sys().(*syscall.Stat_t)
	if stat.Uid != uid || stat.Gid != gid || info.Mode().Perm() != 0o700 {
		t.Fatalf("directory postcondition not satisfied: uid=%d gid=%d mode=%v", stat.Uid, stat.Gid, info.Mode())
	}
}
