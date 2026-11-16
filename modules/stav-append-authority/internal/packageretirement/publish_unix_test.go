//go:build darwin || linux

package packageretirement

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func publicationDigest(b []byte) string { return fmt.Sprintf("sha256:%x", sha256.Sum256(b)) }
func publicationFixture(t *testing.T) (string, Identity, *Lease, []byte, []byte) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	payload, receipt := []byte("native owner bytes"), []byte("immutable receipt bytes")
	id := Identity{Owner: "publication-test", Version: "1", ReceiptDigest: publicationDigest(receipt), BinaryDigest: publicationDigest(payload), BinaryRelative: "libexec/owner", ReceiptRelative: "share/receipt.json", BinarySize: uint64(len(payload))}
	lease, err := PrepareInstall(root, id, true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { lease.Close() })
	return root, id, lease, payload, receipt
}

func TestPublicationDurableBytesAndNoImplicitRepair(t *testing.T) {
	root, id, lease, payload, receipt := publicationFixture(t)
	for _, f := range []struct {
		path string
		data []byte
		mode os.FileMode
	}{{id.BinaryRelative, payload, 0o755}, {id.ReceiptRelative, receipt, 0o644}} {
		changed, err := lease.PublishFile(f.path, bytes.NewReader(f.data), publicationDigest(f.data), uint64(len(f.data)), f.mode)
		if err != nil || !changed {
			t.Fatalf("publish: changed=%v err=%v", changed, err)
		}
	}
	if err := lease.Activate(); err != nil {
		t.Fatal(err)
	}
	if changed, err := lease.PublishFile(id.BinaryRelative, nil, id.BinaryDigest, id.BinarySize, 0o755); err != nil || changed {
		t.Fatalf("exact retry: %v %v", changed, err)
	}
	if err := os.Remove(filepath.Join(root, id.BinaryRelative)); err != nil {
		t.Fatal(err)
	}
	if _, err := lease.PublishFile(id.BinaryRelative, bytes.NewReader(payload), id.BinaryDigest, id.BinarySize, 0o755); err == nil {
		t.Fatal("silently repaired an open package")
	}
}

type publicationHook struct {
	reader io.Reader
	hook   func()
}

func (r *publicationHook) Read(b []byte) (int, error) {
	if r.hook != nil {
		f := r.hook
		r.hook = nil
		f()
	}
	return r.reader.Read(b)
}

func TestPublicationRefusesReplacementAndConflictingFile(t *testing.T) {
	for _, kind := range []string{"root replaced", "destination created", "wrong bytes"} {
		t.Run(kind, func(t *testing.T) {
			root, id, lease, payload, _ := publicationFixture(t)
			input := io.Reader(bytes.NewReader(payload))
			destination := filepath.Join(root, id.BinaryRelative)
			switch kind {
			case "root replaced":
				input = &publicationHook{reader: input, hook: func() {
					if err := os.Rename(root, root+"-moved"); err != nil {
						t.Fatal(err)
					}
					t.Cleanup(func() { os.RemoveAll(root + "-moved") })
					if err := os.Mkdir(root, 0o700); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(root, "sentinel"), []byte("untouched"), 0o600); err != nil {
						t.Fatal(err)
					}
				}}
			case "destination created":
				input = &publicationHook{reader: input, hook: func() {
					if err := os.WriteFile(destination, []byte("administrator"), 0o600); err != nil {
						t.Fatal(err)
					}
				}}
			case "wrong bytes":
				input = bytes.NewReader([]byte("unreviewed content"))
			}
			if _, err := lease.PublishFile(id.BinaryRelative, input, id.BinaryDigest, id.BinarySize, 0o755); err == nil {
				t.Fatal("unsafe publication succeeded")
			}
			if kind == "destination created" {
				data, err := os.ReadFile(destination)
				if err != nil || string(data) != "administrator" {
					t.Fatal("administrator file changed", err)
				}
			} else if _, err := os.Lstat(destination); !os.IsNotExist(err) {
				t.Fatal("publication escaped or accepted changed source", err)
			}
			if kind == "root replaced" {
				data, err := os.ReadFile(filepath.Join(root, "sentinel"))
				if err != nil || string(data) != "untouched" {
					t.Fatal("replacement root changed", err)
				}
			}
		})
	}
}

func TestDistinctPrefixesKeepIndependentUse(t *testing.T) {
	a, id, first, payload, receipt := publicationFixture(t)
	b, other, second, _, _ := publicationFixture(t)
	if id != other {
		t.Fatal("fixture must use exactly the same package identity")
	}
	for _, lease := range []*Lease{first, second} {
		if _, err := lease.PublishFile(id.BinaryRelative, bytes.NewReader(payload), id.BinaryDigest, id.BinarySize, 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := lease.PublishFile(id.ReceiptRelative, bytes.NewReader(receipt), publicationDigest(receipt), uint64(len(receipt)), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := lease.Activate(); err != nil {
			t.Fatal(err)
		}
		lease.Close()
	}
	use, err := AcquireUse(a, id)
	if err != nil {
		t.Fatal(err)
	}
	defer use.Close()
	retiring, err := BeginRetirement(b, id)
	if err != nil {
		t.Fatal("other installation use blocked retirement", err)
	}
	defer retiring.Close()
	if err := retiring.RemoveFile(id.BinaryRelative, id.BinaryDigest, id.BinarySize); err != nil {
		t.Fatal(err)
	}
	if err := retiring.RemoveFile(id.ReceiptRelative, publicationDigest(receipt), uint64(len(receipt))); err != nil {
		t.Fatal(err)
	}
	if err := retiring.FinishRetirement(); err != nil {
		t.Fatal(err)
	}
	again, err := AcquireUse(a, id)
	if err != nil {
		t.Fatal("other prefix retirement closed this installation", err)
	}
	again.Close()
	if err := use.VerifyFile(id.BinaryRelative, id.BinaryDigest, id.BinarySize); err != nil {
		t.Fatal("other prefix retirement changed this payload", err)
	}
}
