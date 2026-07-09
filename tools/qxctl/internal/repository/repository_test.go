package repository

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindRoot(t *testing.T) {
	tempDir := t.TempDir()

	readme := filepath.Join(tempDir, "README.md")
	intent := filepath.Join(tempDir, "INTENT.md")

	if err := os.WriteFile(readme, []byte("readme"), 0644); err != nil {
		t.Fatalf("failed to write README.md: %v", err)
	}

	if err := os.WriteFile(intent, []byte("intent"), 0644); err != nil {
		t.Fatalf("failed to write INTENT.md: %v", err)
	}

	nested := filepath.Join(tempDir, "a", "b", "c")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}

	root, err := FindRoot(nested)
	if err != nil {
		t.Fatalf("FindRoot failed: %v", err)
	}

	if root != tempDir {
		t.Errorf("expected %q, got %q", tempDir, root)
	}
}

func TestFindRootNotFound(t *testing.T) {
	tempDir := t.TempDir()

	nested := filepath.Join(tempDir, "a", "b", "c")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}

	_, err := FindRoot(nested)
	if err == nil {
		t.Error("expected error, got nil")
	}
}
