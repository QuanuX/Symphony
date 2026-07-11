package repository

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindRoot(t *testing.T) {
	tempDir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("failed to eval symlinks: %v", err)
	}

	readme := filepath.Join(tempDir, "README.md")
	intent := filepath.Join(tempDir, "INTENT.md")
	modules := filepath.Join(tempDir, "modules")

	if err := os.WriteFile(readme, []byte("readme"), 0644); err != nil {
		t.Fatalf("failed to write README.md: %v", err)
	}

	if err := os.WriteFile(intent, []byte("intent"), 0644); err != nil {
		t.Fatalf("failed to write INTENT.md: %v", err)
	}

	if err := os.MkdirAll(modules, 0755); err != nil {
		t.Fatalf("failed to create modules dir: %v", err)
	}

	t.Run("FindRoot succeeds from repository root", func(t *testing.T) {
		root, err := FindRoot(tempDir)
		if err != nil {
			t.Fatalf("FindRoot failed: %v", err)
		}
		if root != tempDir {
			t.Errorf("expected %q, got %q", tempDir, root)
		}
	})

	nested := filepath.Join(tempDir, "a", "b", "c")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}

	t.Run("FindRoot succeeds from nested directories under a valid repository root", func(t *testing.T) {
		root, err := FindRoot(nested)
		if err != nil {
			t.Fatalf("FindRoot failed: %v", err)
		}
		if root != tempDir {
			t.Errorf("expected %q, got %q", tempDir, root)
		}
	})

	toolDir := filepath.Join(tempDir, "tools", "qxctl")
	if err := os.MkdirAll(toolDir, 0755); err != nil {
		t.Fatalf("failed to create tool dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(toolDir, "INTENT.md"), []byte("tool intent"), 0644); err != nil {
		t.Fatalf("failed to write tool INTENT.md: %v", err)
	}

	t.Run("FindRoot succeeds from tools/qxctl inside a valid repository root", func(t *testing.T) {
		root, err := FindRoot(toolDir)
		if err != nil {
			t.Fatalf("FindRoot failed: %v", err)
		}
		if root != tempDir {
			t.Errorf("expected %q, got %q", tempDir, root)
		}
	})
	
	t.Run("FindRoot succeeds with current working directory inside tools/qxctl", func(t *testing.T) {
		orig, err := os.Getwd()
		if err != nil {
			t.Fatalf("Getwd failed: %v", err)
		}
		if err := os.Chdir(toolDir); err != nil {
			t.Fatalf("Chdir failed: %v", err)
		}
		defer os.Chdir(orig)
		
		wd, _ := os.Getwd()
		root, err := FindRoot(wd)
		if err != nil {
			t.Fatalf("FindRoot failed: %v", err)
		}
		if root != tempDir {
			t.Errorf("expected %q, got %q", tempDir, root)
		}
	})
}

func TestFindRootNotFound(t *testing.T) {
	t.Run("FindRoot fails outside a repository root", func(t *testing.T) {
		tempDir, _ := filepath.EvalSymlinks(t.TempDir())
		_, err := FindRoot(tempDir)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("FindRoot fails when README.md and INTENT.md exist but modules/ is absent", func(t *testing.T) {
		tempDir, _ := filepath.EvalSymlinks(t.TempDir())
		os.WriteFile(filepath.Join(tempDir, "README.md"), []byte("readme"), 0644)
		os.WriteFile(filepath.Join(tempDir, "INTENT.md"), []byte("intent"), 0644)
		_, err := FindRoot(tempDir)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("FindRoot does not treat tools/qxctl as a repository root merely because qxctl has INTENT.md", func(t *testing.T) {
		tempDir, _ := filepath.EvalSymlinks(t.TempDir())
		toolDir := filepath.Join(tempDir, "tools", "qxctl")
		os.MkdirAll(toolDir, 0755)
		os.WriteFile(filepath.Join(toolDir, "INTENT.md"), []byte("intent"), 0644)
		
		_, err := FindRoot(toolDir)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}
