package modules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestList_Success(t *testing.T) {
	tempDir := t.TempDir()

	for _, mod := range CanonicalModules {
		modPath := filepath.Join(tempDir, "modules", mod)
		if err := os.MkdirAll(modPath, 0755); err != nil {
			t.Fatalf("failed to create module dir: %v", err)
		}
	}

	output, err := List(tempDir)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	lastLine := output[len(output)-1]
	if lastLine != "modules: checks passed" {
		t.Errorf("expected last line to be success, got %q", lastLine)
	}

	if !strings.Contains(output[1], "node-troll") {
		t.Errorf("expected node-troll to be listed in index 1")
	}
	if !strings.Contains(output[2], "bus-troll") {
		t.Errorf("expected bus-troll to be listed in index 2")
	}
	if !strings.Contains(output[3], "hotpath-runtime") {
		t.Errorf("expected hotpath-runtime to be listed in index 3")
	}
}

func TestList_MissingModule(t *testing.T) {
	tempDir := t.TempDir()
	// Leave tempDir empty so modules are missing

	output, err := List(tempDir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "missing required module") {
		t.Errorf("expected missing module error, got: %v", err)
	}

	lastLine := output[len(output)-1]
	if !strings.Contains(lastLine, "missing module") {
		t.Errorf("expected last output line to indicate missing module, got %q", lastLine)
	}
}

func TestInspect_Success(t *testing.T) {
	tempDir := t.TempDir()

	mod := CanonicalModules[0]
	modPath := filepath.Join(tempDir, "modules", mod)
	if err := os.MkdirAll(modPath, 0755); err != nil {
		t.Fatalf("failed to create module dir: %v", err)
	}

	for _, file := range ExpectedFiles {
		filePath := filepath.Join(modPath, file)
		if err := os.WriteFile(filePath, []byte("# Title\nContent"), 0644); err != nil {
			t.Fatalf("failed to write contract file: %v", err)
		}
	}

	output, err := Inspect(tempDir, mod)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	lastLine := output[len(output)-1]
	if lastLine != "inspection: checks passed" {
		t.Errorf("expected last line to be success, got %q", lastLine)
	}

	hasTitle := false
	for _, line := range output {
		if strings.HasPrefix(line, "title: INTENT.md # Title") {
			hasTitle = true
		}
	}

	if !hasTitle {
		t.Errorf("expected to extract H1 title, output: %v", output)
	}
}

func TestInspect_UnknownModule(t *testing.T) {
	tempDir := t.TempDir()

	_, err := Inspect(tempDir, "unknown-module")
	if err == nil {
		t.Fatal("expected error for unknown module, got nil")
	}

	if !strings.Contains(err.Error(), "unknown module") {
		t.Errorf("expected unknown module error, got: %v", err)
	}
}

func TestInspect_MissingModuleDir(t *testing.T) {
	tempDir := t.TempDir()

	mod := CanonicalModules[0]

	_, err := Inspect(tempDir, mod)
	if err == nil {
		t.Fatal("expected error for missing module dir, got nil")
	}

	if !strings.Contains(err.Error(), "missing module directory") {
		t.Errorf("expected missing module directory error, got: %v", err)
	}
}

func TestInspect_MissingContractFile(t *testing.T) {
	tempDir := t.TempDir()

	mod := CanonicalModules[0]
	modPath := filepath.Join(tempDir, "modules", mod)
	if err := os.MkdirAll(modPath, 0755); err != nil {
		t.Fatalf("failed to create module dir: %v", err)
	}

	output, err := Inspect(tempDir, mod)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "missing contract file") {
		t.Errorf("expected missing contract file error, got: %v", err)
	}

	lastLine := output[len(output)-1]
	if !strings.Contains(lastLine, "missing") {
		t.Errorf("expected last output line to indicate missing, got %q", lastLine)
	}
}
