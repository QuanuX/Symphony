//go:build darwin || linux

package snvstate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

func TestSNVActualDifferentDirectoryOwner(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("non-root test caller required for the read-only root-owned fixture")
	}
	fd, err := unix.Open("/usr/bin", unix.O_RDONLY|unix.O_CLOEXEC|unix.O_DIRECTORY|unix.O_NOFOLLOW, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(fd)
	var actual unix.Stat_t
	if err := unix.Fstat(fd, &actual); err != nil {
		t.Fatal(err)
	}
	// With strict=false, directory type and non-writable permissions satisfy
	// every condition except ownership. This observes actual kernel metadata;
	// it neither mocks Fstat nor changes ownership or system directory bytes.
	if actual.Mode&unix.S_IFMT != unix.S_IFDIR || actual.Mode&0022 != 0 || actual.Uid == uint32(os.Geteuid()) {
		t.Fatal("read-only fixture does not isolate the actual owner mismatch")
	}
	if err := privateDirectory(fd, false); err == nil {
		t.Fatal("actual different-owner directory accepted")
	}
	t.Logf("actual Fstat uid=%d, caller uid=%d, directory mode=%#o: different owner refused", actual.Uid, os.Geteuid(), actual.Mode&0777)
}

func TestSNVHostileTemporaryFilesPreserveAcknowledgedJournal(t *testing.T) {
	for _, kind := range []string{"fifo", "symlink", "hardlink"} {
		t.Run(kind, func(t *testing.T) {
			s, err := NewEvidence(testRoot(t))
			if err != nil {
				t.Fatal(err)
			}
			acknowledged := evidenceFixture(t, "acknowledged")
			if err := s.WithLock(func(tx *Transaction) error {
				if _, err := tx.Prepare(acknowledged); err != nil {
					return err
				}
				return tx.CommitEvidence(acknowledged.OperationID, func() error { return nil })
			}); err != nil {
				t.Fatal(err)
			}
			directory := filepath.Join(s.Root, "symphony", "qxctl", "snv", "evidence-v1")
			journal := filepath.Join(directory, "state.json")
			before, err := os.ReadFile(journal)
			if err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(s.Root, "unrelated-target")
			if err := os.WriteFile(target, []byte("acknowledged unrelated target bytes"), 0600); err != nil {
				t.Fatal(err)
			}
			targetBefore, _ := os.ReadFile(target)
			temporary := filepath.Join(directory, ".state-"+strings.Repeat("b", 32)+".tmp")
			switch kind {
			case "fifo":
				err = unix.Mkfifo(temporary, 0600)
			case "symlink":
				err = os.Symlink(target, temporary)
			case "hardlink":
				err = os.Link(target, temporary)
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := s.WithLock(func(tx *Transaction) error {
				_, err := tx.Prepare(evidenceFixture(t, "must-refuse"))
				return err
			}); err == nil {
				t.Fatal("hostile matching temporary file accepted", kind)
			} else {
				t.Logf("actual matching-name %s refused at writer cleanup: %v", kind, err)
			}
			after, err := os.ReadFile(journal)
			if err != nil || string(after) != string(before) {
				t.Fatal("hostile temporary file refusal changed acknowledged journal", kind, err)
			}
			targetAfter, err := os.ReadFile(target)
			if err != nil || string(targetAfter) != string(targetBefore) {
				t.Fatal("hostile temporary file refusal changed unrelated target", kind, err)
			}
			if _, err := os.Lstat(temporary); err != nil {
				t.Fatal("unsafe temporary entry was silently removed", kind, err)
			}
			if err := s.WithRead(func(tx *Transaction) error {
				if len(tx.Snapshot().Operations) != 1 || len(tx.Snapshot().Bundles) != 1 || string(tx.Current()) != "null" {
					t.Fatal("refusal lost acknowledged retention or selected a head")
				}
				return nil
			}); err != nil {
				t.Fatal("observation could not read unchanged acknowledged journal", err)
			}
		})
	}
}
