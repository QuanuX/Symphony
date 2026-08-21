package supervision

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/paths"
)

func TestRenderPinsExactBinaryAndHasNoCrossServiceDependency(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native supervisor is intentionally unsupported")
	}
	binary := filepath.Join(t.TempDir(), "symphony-accordare-stav-producer")
	spec := Spec{Scope: paths.ScopeUser, TOPSID: "11111111-1111-4111-8111-111111111111", Binary: binary, UID: uint32(os.Geteuid()), GID: uint32(os.Getegid())}
	record, content, err := Render(spec)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	for _, required := range []string{binary, spec.TOPSID, "serve", "--supervised", "--scope", "user"} {
		if !strings.Contains(text, required) {
			t.Fatalf("descriptor omits %q", required)
		}
	}
	for _, forbidden := range []string{"symphony-stav@", "symphony-ssiag", "Requires=", "Wants="} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("descriptor gained cross-service dependency %q", forbidden)
		}
	}
	if record.Digest == "" {
		t.Fatal("descriptor lacks a content digest")
	}
}

func TestEnsureDirectoryRejectsSymlinkComponents(t *testing.T) {
	root := t.TempDir()
	target := t.TempDir()
	alias := filepath.Join(root, "alias")
	if err := os.Symlink(target, alias); err != nil {
		t.Fatal(err)
	}
	if err := ensureDirectory(filepath.Join(alias, "nested")); err == nil {
		t.Fatal("symlinked supervisor directory component was accepted")
	}
}
