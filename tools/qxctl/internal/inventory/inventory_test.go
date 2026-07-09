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

func TestDigest_Success(t *testing.T) {
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

	output, err := Digest(tempDir)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if len(output) != 5 {
		t.Fatalf("expected 5 output lines, got %d", len(output))
	}

	if output[0] != "inventory digest: schema qxctl.runtime_inventory_digest.v1" {
		t.Errorf("expected schema line, got %q", output[0])
	}
	if output[1] != "inventory digest: source_schema qxctl.runtime_inventory.v1" {
		t.Errorf("expected source_schema line, got %q", output[1])
	}
	if output[2] != "inventory digest: algorithm sha256" {
		t.Errorf("expected algorithm line, got %q", output[2])
	}

	digestLine := output[3]
	if !strings.HasPrefix(digestLine, "inventory digest: ") {
		t.Errorf("expected digest line prefix, got %q", digestLine)
	}
	digest := strings.TrimPrefix(digestLine, "inventory digest: ")
	if len(digest) != 64 {
		t.Errorf("expected 64 character digest, got %d", len(digest))
	}

	if output[4] != "inventory digest: checks passed" {
		t.Errorf("expected last line to be success, got %q", output[4])
	}
}

func TestDigestJSON_Success(t *testing.T) {
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

	outputBytes, err := DigestJSON(tempDir)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	var data RuntimeInventoryDigest
	if err := json.Unmarshal(outputBytes, &data); err != nil {
		t.Fatalf("failed to unmarshal generated JSON: %v", err)
	}

	if data.Schema != "qxctl.runtime_inventory_digest.v1" {
		t.Errorf("expected correct schema, got %q", data.Schema)
	}
	if data.SourceSchema != "qxctl.runtime_inventory.v1" {
		t.Errorf("expected correct source_schema, got %q", data.SourceSchema)
	}
	if data.Algorithm != "sha256" {
		t.Errorf("expected correct algorithm, got %q", data.Algorithm)
	}
	if len(data.Digest) != 64 {
		t.Errorf("expected 64 character digest, got %d", len(data.Digest))
	}
}

func TestDigest_Determinism(t *testing.T) {
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

	output1, err := Digest(tempDir)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	output2, err := Digest(tempDir)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if output1[3] != output2[3] {
		t.Errorf("expected deterministic digest, got %q and %q", output1[3], output2[3])
	}

	// Change a file and verify digest changes
	modPath := filepath.Join(tempDir, "modules", modules.CanonicalModules[0])
	filePath := filepath.Join(modPath, modules.ExpectedFiles[0])
	if err := os.WriteFile(filePath, []byte("# Title\nContent Changed"), 0644); err != nil {
		t.Fatalf("failed to write contract file: %v", err)
	}

	output3, err := Digest(tempDir)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if output1[3] == output3[3] {
		t.Errorf("expected digest to change, but got %q", output3[3])
	}
}

func TestDigest_FailsWhenSnapshotFails(t *testing.T) {
	tempDir := t.TempDir()

	_, err := Digest(tempDir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	_, err = DigestJSON(tempDir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
