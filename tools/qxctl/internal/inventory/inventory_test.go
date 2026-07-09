package inventory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/modules"
)

func TestSnapshot_Success(t *testing.T) {
	tempDir := t.TempDir()

	for _, mod := range modules.CanonicalModules {
		modPath := filepath.Join(tempDir, "modules", mod)
		if err := os.MkdirAll(modPath, 0755); err != nil {
			t.Fatalf("failed to create module dir: %v", err)
		}

		for _, file := range modules.ExpectedFiles {
			filePath := filepath.Join(modPath, file)
			if err := os.WriteFile(filePath, []byte("# Title\nContent"), 0644); err != nil {
				t.Fatalf("failed to write contract file: %v", err)
			}
		}
	}

	output, err := Snapshot(tempDir)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if len(output) < 3 {
		t.Fatalf("expected output lines, got %d", len(output))
	}

	if output[0] != "inventory: schema qxctl.runtime_inventory.v1" {
		t.Errorf("expected schema line, got %q", output[0])
	}

	lastLine := output[len(output)-1]
	if lastLine != "inventory: checks passed" {
		t.Errorf("expected last line to be success, got %q", lastLine)
	}
}

func TestSnapshotJSON_Success(t *testing.T) {
	tempDir := t.TempDir()

	for _, mod := range modules.CanonicalModules {
		modPath := filepath.Join(tempDir, "modules", mod)
		if err := os.MkdirAll(modPath, 0755); err != nil {
			t.Fatalf("failed to create module dir: %v", err)
		}

		for _, file := range modules.ExpectedFiles {
			filePath := filepath.Join(modPath, file)
			if err := os.WriteFile(filePath, []byte("# Title\nContent"), 0644); err != nil {
				t.Fatalf("failed to write contract file: %v", err)
			}
		}
	}

	outputBytes, err := SnapshotJSON(tempDir)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	output := string(outputBytes)
	if !strings.Contains(output, `"schema": "qxctl.runtime_inventory.v1"`) {
		t.Errorf("expected JSON to contain correct schema, got %s", output)
	}
	if !strings.Contains(output, `"go_baseline": "go1.26.5"`) {
		t.Errorf("expected JSON to contain go_baseline")
	}
	if !strings.Contains(output, `"module_count": 3`) {
		t.Errorf("expected JSON to contain module_count 3")
	}
	if !strings.Contains(output, `"contract_count": 12`) {
		t.Errorf("expected JSON to contain contract_count 12")
	}
	if !strings.Contains(output, `"module": "node-troll"`) {
		t.Errorf("expected JSON to contain module node-troll")
	}
	if !strings.Contains(output, `"path": "modules/node-troll/INTENT.md"`) {
		t.Errorf("expected JSON to contain relative paths without absolute prefix")
	}
	if !strings.Contains(output, `"path": "modules/node-troll"`) {
		t.Errorf("expected JSON to contain relative module path")
	}

	// Validate JSON is well-formed
	var data RuntimeInventory
	if err := json.Unmarshal(outputBytes, &data); err != nil {
		t.Fatalf("failed to unmarshal generated JSON: %v", err)
	}
}

func TestSnapshot_MissingModule(t *testing.T) {
	tempDir := t.TempDir()

	_, err := Snapshot(tempDir)
	if err == nil {
		t.Fatal("expected error for missing module, got nil")
	}
}

func TestSnapshot_MissingContract(t *testing.T) {
	tempDir := t.TempDir()

	modPath := filepath.Join(tempDir, "modules", modules.CanonicalModules[0])
	if err := os.MkdirAll(modPath, 0755); err != nil {
		t.Fatalf("failed to create module dir: %v", err)
	}

	_, err := Snapshot(tempDir)
	if err == nil {
		t.Fatal("expected error for missing contract, got nil")
	}
}

func TestSnapshot_EmptyContract(t *testing.T) {
	tempDir := t.TempDir()

	for _, mod := range modules.CanonicalModules {
		modPath := filepath.Join(tempDir, "modules", mod)
		if err := os.MkdirAll(modPath, 0755); err != nil {
			t.Fatalf("failed to create module dir: %v", err)
		}

		for _, file := range modules.ExpectedFiles {
			filePath := filepath.Join(modPath, file)
			if err := os.WriteFile(filePath, []byte(""), 0644); err != nil {
				t.Fatalf("failed to write contract file: %v", err)
			}
		}
	}

	_, err := Snapshot(tempDir)
	if err == nil {
		t.Fatal("expected error for empty contract, got nil")
	}
}

func TestSnapshot_MissingH1(t *testing.T) {
	tempDir := t.TempDir()

	for _, mod := range modules.CanonicalModules {
		modPath := filepath.Join(tempDir, "modules", mod)
		if err := os.MkdirAll(modPath, 0755); err != nil {
			t.Fatalf("failed to create module dir: %v", err)
		}

		for _, file := range modules.ExpectedFiles {
			filePath := filepath.Join(modPath, file)
			if err := os.WriteFile(filePath, []byte("Content without H1\n"), 0644); err != nil {
				t.Fatalf("failed to write contract file: %v", err)
			}
		}
	}

	_, err := Snapshot(tempDir)
	if err == nil {
		t.Fatal("expected error for missing H1, got nil")
	}
}
