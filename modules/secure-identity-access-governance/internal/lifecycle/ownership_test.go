package lifecycle

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	ssiagpaths "github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/paths"
)

func TestEnrollmentRejectsReservedOwnershipBeforeInstallationOrMutation(t *testing.T) {
	for _, ids := range [][2]uint32{{math.MaxUint32, 20}, {501, math.MaxUint32}} {
		_, err := Enroll(ssiagpaths.ScopeSystem, testTOPSID, "Desk", &ids[0], &ids[1])
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

func TestOwnedDirectoryPreservesRepresentableIdentityAtChown(t *testing.T) {
	captureOnly := errors.New("ownership arguments captured without changing ownership")
	for _, value := range []uint32{0, math.MaxInt32, math.MaxInt32 + 1, math.MaxUint32 - 1} {
		for _, ids := range [][2]uint32{{value, 20}, {501, value}} {
			path := filepath.Join(t.TempDir(), "capture-only-chown")
			called := false
			err := ensureOwnedPrivateDirectoryWithChown(path, ids[0], ids[1], func(_ string, uid, gid int) error {
				called = true
				if uid < 0 || gid < 0 || uint64(uid) != uint64(ids[0]) || uint64(gid) != uint64(ids[1]) {
					t.Errorf("ownership IDs changed during conversion: want %v, got %d/%d", ids, uid, gid)
				}
				return captureOnly
			})
			if uint64(ids[0]) > math.MaxInt || uint64(ids[1]) > math.MaxInt {
				if err == nil || called || !strings.Contains(err.Error(), "cannot be represented") {
					t.Fatalf("unrepresentable identity %v reached chown: called=%t error=%v", ids, called, err)
				}
				if _, err := os.Lstat(path); !os.IsNotExist(err) {
					t.Fatalf("unrepresentable identity created a directory: %v", err)
				}
			} else if !called || !errors.Is(err, captureOnly) {
				t.Fatalf("representable identity %v did not reach capture-only chown: called=%t error=%v", ids, called, err)
			}
		}
	}
}
