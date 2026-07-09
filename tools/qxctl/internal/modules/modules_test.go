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
}

func TestInspect_UnknownModule(t *testing.T) {
	tempDir := t.TempDir()

	_, err := Inspect(tempDir, "unknown-module")
	if err == nil {
		t.Fatal("expected error for unknown module, got nil")
	}
}

func TestInspect_MissingModuleDir(t *testing.T) {
	tempDir := t.TempDir()

	mod := CanonicalModules[0]

	_, err := Inspect(tempDir, mod)
	if err == nil {
		t.Fatal("expected error for missing module dir, got nil")
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

	lastLine := output[len(output)-1]
	if !strings.Contains(lastLine, "missing") {
		t.Errorf("expected last output line to indicate missing, got %q", lastLine)
	}
}

func TestCheck_Success(t *testing.T) {
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

	output, err := Check(tempDir, mod)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	lastLine := output[len(output)-1]
	if lastLine != "module check: checks passed" {
		t.Errorf("expected last line to be success, got %q", lastLine)
	}
}

func TestCheck_UnknownModule(t *testing.T) {
	tempDir := t.TempDir()

	_, err := Check(tempDir, "unknown-module")
	if err == nil {
		t.Fatal("expected error for unknown module, got nil")
	}
}

func TestCheck_MissingModuleDir(t *testing.T) {
	tempDir := t.TempDir()

	mod := CanonicalModules[0]

	_, err := Check(tempDir, mod)
	if err == nil {
		t.Fatal("expected error for missing module dir, got nil")
	}
}

func TestCheck_MissingContractFile(t *testing.T) {
	tempDir := t.TempDir()

	mod := CanonicalModules[0]
	modPath := filepath.Join(tempDir, "modules", mod)
	if err := os.MkdirAll(modPath, 0755); err != nil {
		t.Fatalf("failed to create module dir: %v", err)
	}

	_, err := Check(tempDir, mod)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCheck_EmptyContractFile(t *testing.T) {
	tempDir := t.TempDir()

	mod := CanonicalModules[0]
	modPath := filepath.Join(tempDir, "modules", mod)
	if err := os.MkdirAll(modPath, 0755); err != nil {
		t.Fatalf("failed to create module dir: %v", err)
	}

	for _, file := range ExpectedFiles {
		filePath := filepath.Join(modPath, file)
		if err := os.WriteFile(filePath, []byte(""), 0644); err != nil {
			t.Fatalf("failed to write contract file: %v", err)
		}
	}

	_, err := Check(tempDir, mod)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCheck_MissingH1(t *testing.T) {
	tempDir := t.TempDir()

	mod := CanonicalModules[0]
	modPath := filepath.Join(tempDir, "modules", mod)
	if err := os.MkdirAll(modPath, 0755); err != nil {
		t.Fatalf("failed to create module dir: %v", err)
	}

	for _, file := range ExpectedFiles {
		filePath := filepath.Join(modPath, file)
		if err := os.WriteFile(filePath, []byte("No title here\nContent"), 0644); err != nil {
			t.Fatalf("failed to write contract file: %v", err)
		}
	}

	_, err := Check(tempDir, mod)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCheckAll_Success(t *testing.T) {
	tempDir := t.TempDir()

	for _, mod := range CanonicalModules {
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
	}

	output, err := CheckAll(tempDir)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	lastLine := output[len(output)-1]
	if lastLine != "modules check: checks passed" {
		t.Errorf("expected last line to be success, got %q", lastLine)
	}

	if !strings.Contains(output[0], "node-troll") {
		t.Errorf("expected node-troll to be listed in index 0")
	}
	if !strings.Contains(output[1], "bus-troll") {
		t.Errorf("expected bus-troll to be listed in index 1")
	}
	if !strings.Contains(output[2], "hotpath-runtime") {
		t.Errorf("expected hotpath-runtime to be listed in index 2")
	}
}

func TestCheckAll_Failure(t *testing.T) {
	tempDir := t.TempDir()

	// Only populate one module
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

	_, err := CheckAll(tempDir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestMetadata_Success(t *testing.T) {
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

	output, err := Metadata(tempDir, mod)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	lastLine := output[len(output)-1]
	if lastLine != "module metadata: checks passed" {
		t.Errorf("expected last line to be success, got %q", lastLine)
	}
}

func TestMetadataAll_Success(t *testing.T) {
	tempDir := t.TempDir()

	for _, mod := range CanonicalModules {
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
	}

	output, err := MetadataAll(tempDir)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	lastLine := output[len(output)-1]
	if lastLine != "modules metadata: checks passed" {
		t.Errorf("expected last line to be success, got %q", lastLine)
	}
}

func TestMetadataJSON_Success(t *testing.T) {
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

	outputBytes, err := MetadataJSON(tempDir, mod)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	output := string(outputBytes)
	if !strings.Contains(output, `"schema": "qxctl.contract_metadata.v1"`) {
		t.Errorf("expected JSON to contain correct schema, got %s", output)
	}
	if !strings.Contains(output, `"module": "node-troll"`) {
		t.Errorf("expected JSON to contain module node-troll")
	}
	if !strings.Contains(output, `"path": "modules/node-troll/INTENT.md"`) {
		t.Errorf("expected JSON to contain relative paths without absolute prefix")
	}
}

func TestMetadataAllJSON_Success(t *testing.T) {
	tempDir := t.TempDir()

	for _, mod := range CanonicalModules {
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
	}

	outputBytes, err := MetadataAllJSON(tempDir)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	output := string(outputBytes)
	if !strings.Contains(output, `"schema": "qxctl.modules_contract_metadata.v1"`) {
		t.Errorf("expected JSON to contain correct schema")
	}
	if !strings.Contains(output, `"module": "node-troll"`) {
		t.Errorf("expected JSON to contain module node-troll")
	}
	if !strings.Contains(output, `"module": "bus-troll"`) {
		t.Errorf("expected JSON to contain module bus-troll")
	}
	if !strings.Contains(output, `"module": "hotpath-runtime"`) {
		t.Errorf("expected JSON to contain module hotpath-runtime")
	}
	if !strings.Contains(output, `"path": "modules/node-troll/INTENT.md"`) {
		t.Errorf("expected JSON to contain relative paths without absolute prefix")
	}
}

func TestMetadataJSON_UnknownModule(t *testing.T) {
	tempDir := t.TempDir()

	_, err := MetadataJSON(tempDir, "unknown-module")
	if err == nil {
		t.Fatal("expected error for unknown module, got nil")
	}
}
