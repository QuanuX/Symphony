package contracts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerify_Success(t *testing.T) {
	tempDir := t.TempDir()

	for _, mod := range ExpectedModules {
		modPath := filepath.Join(tempDir, mod)
		if err := os.MkdirAll(modPath, 0755); err != nil {
			t.Fatalf("failed to create module dir: %v", err)
		}
		for _, file := range ExpectedFiles {
			filePath := filepath.Join(modPath, file)
			if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
				t.Fatalf("failed to write contract file: %v", err)
			}
		}
	}

	output, err := Verify(tempDir)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	lastLine := output[len(output)-1]
	if lastLine != "contracts: checks passed" {
		t.Errorf("expected last line to be success, got %q", lastLine)
	}
}

func TestVerify_MissingModule(t *testing.T) {
	tempDir := t.TempDir()
	// Leave tempDir empty so first module is missing

	output, err := Verify(tempDir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "missing module") {
		t.Errorf("expected missing module error, got: %v", err)
	}

	lastLine := output[len(output)-1]
	if !strings.Contains(lastLine, "missing module") {
		t.Errorf("expected last output line to indicate missing module, got %q", lastLine)
	}
}

func TestVerify_MissingFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create modules but leave files empty
	for _, mod := range ExpectedModules {
		modPath := filepath.Join(tempDir, mod)
		if err := os.MkdirAll(modPath, 0755); err != nil {
			t.Fatalf("failed to create module dir: %v", err)
		}
	}

	output, err := Verify(tempDir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "missing file") {
		t.Errorf("expected missing file error, got: %v", err)
	}

	lastLine := output[len(output)-1]
	if !strings.Contains(lastLine, "missing file") {
		t.Errorf("expected last output line to indicate missing file, got %q", lastLine)
	}
}
