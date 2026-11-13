package main

import (
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/packageinstall"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/version"
	"os"
	"path/filepath"
	"testing"
)

func installFixture(source, prefix string) (string, string, error) {
	r, err := packageinstall.Install(source, prefix, version.Version)
	return r.Binary, r.Receipt, err
}
func removeFixture(source, prefix string) error {
	_, err := packageinstall.Uninstall(prefix, version.Version)
	return err
}
func fixture(t *testing.T) (string, string, string, string) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	t.Setenv("XDG_RUNTIME_DIR", filepath.Join(root, "runtime"))
	prefix := filepath.Join(root, "prefix")
	if err = os.Mkdir(prefix, 0700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "fixture")
	if err = os.WriteFile(source, []byte("fixture bytes; never executed\n"), 0700); err != nil {
		t.Fatal(err)
	}
	binary, receipt, err := installFixture(source, prefix)
	if err != nil {
		t.Fatal(err)
	}
	return source, prefix, binary, receipt
}
func TestReadOnlyOwnerAndRemoval(t *testing.T) {
	source, prefix, binary, _ := fixture(t)
	sentinel := filepath.Join(prefix, "user-data")
	if err := os.WriteFile(sentinel, []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	r, err := inspect(binary)
	if err != nil {
		t.Fatal(err)
	}
	if r.Version != version.Version || r.Component != component || r.Provider || r.FreshHost || r.Release || r.Publication || r.Scope != "read_only_owner_receipt" {
		t.Fatalf("incorrect scope: %+v", r)
	}
	if err = removeFixture(source, prefix); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(sentinel)
	if err != nil || string(b) != "preserve" {
		t.Fatal("unrelated state changed")
	}
	if _, err = inspect(binary); err == nil {
		t.Fatal("removed executable accepted")
	}
	for _, name := range []string{"config", "state", "runtime", "Library"} {
		if _, err = os.Stat(filepath.Join(os.Getenv("HOME"), name)); !os.IsNotExist(err) {
			t.Fatalf("inspection/install created operational state: %s", name)
		}
	}
}
func TestOwnerRefusals(t *testing.T) {
	for _, name := range []string{"payload", "receipt", "missing", "binary-link", "receipt-link", "writable-binary", "writable-receipt", "special-mode", "hardlink", "relative", "wrong-version", "parent-link"} {
		t.Run(name, func(t *testing.T) {
			_, prefix, binary, receipt := fixture(t)
			switch name {
			case "payload":
				if err := os.WriteFile(binary, []byte("changed"), 0700); err != nil {
					t.Fatal(err)
				}
			case "receipt":
				if err := os.WriteFile(receipt, []byte("{}"), 0600); err != nil {
					t.Fatal(err)
				}
			case "missing":
				if err := os.Remove(receipt); err != nil {
					t.Fatal(err)
				}
			case "binary-link", "receipt-link":
				target := binary
				if name == "receipt-link" {
					target = receipt
				}
				if err := os.Rename(target, target+".real"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target+".real", target); err != nil {
					t.Fatal(err)
				}
			case "writable-binary":
				if err := os.Chmod(binary, 0777); err != nil {
					t.Fatal(err)
				}
			case "writable-receipt":
				if err := os.Chmod(receipt, 0666); err != nil {
					t.Fatal(err)
				}
			case "special-mode":
				// Some hosts clear setuid/setgid on ordinary-user fixture files.
				// Sticky mode remains observable and exercises special-mode refusal.
				if err := os.Chmod(binary, 0700|os.ModeSticky); err != nil {
					t.Fatal(err)
				}
			case "hardlink":
				if err := os.Link(binary, binary+".link"); err != nil {
					t.Fatal(err)
				}
			case "relative":
				binary = "relative/" + filepath.Base(binary)
			case "wrong-version":
				binary = filepath.Join(prefix, "libexec", "symphony", component, "other", filepath.Base(binary))
			case "parent-link":
				if err := os.Symlink(prefix, prefix+"-link"); err != nil {
					t.Fatal(err)
				}
				rel, _ := filepath.Rel(prefix, binary)
				binary = filepath.Join(prefix+"-link", rel)
			}
			if _, err := inspect(binary); err == nil {
				t.Fatal("unsafe installation accepted")
			}
		})
	}
}
