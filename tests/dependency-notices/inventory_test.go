package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInventoryRejectsTraversalAndSymlink(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "LICENSE")
	if e := os.WriteFile(outside, []byte("outside"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := os.Symlink(outside, filepath.Join(root, "LICENSE")); e != nil {
		t.Fatal(e)
	}
	for _, name := range []string{"../LICENSE", "LICENSE", "/absolute"} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("unsafe source admitted")
				}
			}()
			checked(root, name)
		})
	}
}
func TestGoModuleEscapingAndLegalMaterial(t *testing.T) {
	if escapeModule("github.com/QuanuX/Symphony") != "github.com/!quanu!x/!symphony" {
		t.Fatal("module identity changed")
	}
	root := t.TempDir()
	for _, name := range []string{"LICENSE.MIT", "NOTICE.txt", "COPYING", "ordinary.go"} {
		if e := os.WriteFile(filepath.Join(root, name), nil, 0600); e != nil {
			t.Fatal(e)
		}
	}
	if len(legalFiles(root)) != 3 {
		t.Fatal("legal material omitted")
	}
}

func TestReceiptNoticeTamperingIsRejected(t *testing.T) {
	oldRepo, oldPrefix, oldOutput := repo, prefix, output
	defer func() { repo, prefix, output = oldRepo, oldPrefix, oldOutput }()
	repo = "../.."
	// Named-file go test executes in the caller's working directory.
	if _, err := os.Stat("cmake/notices/sources.json"); err == nil {
		repo = "."
	}
	prefix = t.TempDir()
	output = t.TempDir()
	rel := "share/licenses/fixture/LICENSE"
	p := filepath.Join(prefix, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		t.Fatal(err)
	}
	good := []byte("original notice")
	if err := os.WriteFile(p, good, 0600); err != nil {
		t.Fatal(err)
	}
	rp := filepath.Join(prefix, "share/symphony/receipts/fixture/1/install-receipt.json")
	if err := os.MkdirAll(filepath.Dir(rp), 0700); err != nil {
		t.Fatal(err)
	}
	data := encode(object{"protocol": "symphony.knowledge.install-receipt.v2", "component_id": "fixture", "version": "1", "files": []any{object{"path": rel, "kind": "regular", "digest": "sha256:" + hash(good), "size": len(good)}}})
	if err := os.WriteFile(rp, data, 0600); err != nil {
		t.Fatal(err)
	}
	if len(receiptInventory()) != 1 {
		t.Fatal("valid notice receipt omitted")
	}
	if err := os.WriteFile(p, []byte("altered notice"), 0600); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("altered receipt-owned notice admitted")
		}
	}()
	receiptInventory()
}
