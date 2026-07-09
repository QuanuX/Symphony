package status

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func createValidTestRepo(t *testing.T) string {
	t.Helper()
	tempDir := t.TempDir()

	modules := []string{"node-troll", "bus-troll", "hotpath-runtime"}
	files := []string{"INTENT.md", "MANIFEST.md", "INSTALL.md", "SKILL.md"}

	for _, mod := range modules {
		modPath := filepath.Join(tempDir, "modules", mod)
		if err := os.MkdirAll(modPath, 0755); err != nil {
			t.Fatalf("failed to create module dir: %v", err)
		}
		for _, f := range files {
			filePath := filepath.Join(modPath, f)
			content := "# " + f + "\n\nvalid content\n"
			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				t.Fatalf("failed to write contract file: %v", err)
			}
		}
	}
	return tempDir
}

func TestReport_ValidRepo(t *testing.T) {
	repoPath := createValidTestRepo(t)

	output, err := Report(repoPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedLines := []string{
		"status: schema qxctl.status.v1",
		"status: repository ok",
		"status: modules ok",
		"status: contracts ok",
		"status: inventory ok",
	}

	for _, expected := range expectedLines {
		found := false
		for _, line := range output {
			if line == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected output line %q not found", expected)
		}
	}

	foundDigest := false
	for _, line := range output {
		if strings.HasPrefix(line, "status: digest sha256 ") {
			foundDigest = true
			digest := strings.TrimPrefix(line, "status: digest sha256 ")
			if len(digest) != 64 {
				t.Errorf("expected 64 character digest, got %d", len(digest))
			}
			break
		}
	}
	if !foundDigest {
		t.Errorf("expected digest line not found")
	}

	foundPassed := false
	for _, line := range output {
		if line == "status: checks passed" {
			foundPassed = true
			break
		}
	}
	if !foundPassed {
		t.Errorf("expected 'checks passed' line not found")
	}
}

func TestReportJSON_ValidRepo(t *testing.T) {
	repoPath := createValidTestRepo(t)

	jsonBytes, err := ReportJSON(repoPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var status AdministrativeStatus
	if err := json.Unmarshal(jsonBytes, &status); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if status.Schema != "qxctl.status.v1" {
		t.Errorf("expected schema qxctl.status.v1, got %q", status.Schema)
	}

	if status.GoBaseline != "go1.26.5" {
		t.Errorf("expected go_baseline go1.26.5, got %q", status.GoBaseline)
	}

	if !status.Modules.Ok || status.Modules.Count != 3 {
		t.Errorf("expected modules ok=true count=3, got ok=%v count=%d", status.Modules.Ok, status.Modules.Count)
	}

	if !status.Contracts.Ok || status.Contracts.Count != 12 {
		t.Errorf("expected contracts ok=true count=12, got ok=%v count=%d", status.Contracts.Ok, status.Contracts.Count)
	}

	if !status.Digest.Ok || status.Digest.Algorithm != "sha256" || len(status.Digest.Value) != 64 {
		t.Errorf("invalid digest structure: %+v", status.Digest)
	}
}

func TestReport_MissingModule(t *testing.T) {
	repoPath := createValidTestRepo(t)
	// Remove a module directory
	os.RemoveAll(filepath.Join(repoPath, "modules", "node-troll"))

	_, err := Report(repoPath)
	if err == nil {
		t.Fatal("expected error for missing module, got nil")
	}
	if !strings.Contains(err.Error(), "missing module directory") {
		t.Errorf("expected missing module directory error, got: %v", err)
	}
}

func TestReport_MissingContract(t *testing.T) {
	repoPath := createValidTestRepo(t)
	// Remove a contract file
	os.Remove(filepath.Join(repoPath, "modules", "node-troll", "INTENT.md"))

	_, err := Report(repoPath)
	if err == nil {
		t.Fatal("expected error for missing contract, got nil")
	}
	if !strings.Contains(err.Error(), "missing contract file") {
		t.Errorf("expected missing contract error, got: %v", err)
	}
}

func TestReport_EmptyContract(t *testing.T) {
	repoPath := createValidTestRepo(t)
	// Empty a contract file
	os.WriteFile(filepath.Join(repoPath, "modules", "node-troll", "INTENT.md"), []byte(""), 0644)

	_, err := Report(repoPath)
	if err == nil {
		t.Fatal("expected error for empty contract, got nil")
	}
	if !strings.Contains(err.Error(), "invalid contract file") {
		t.Errorf("expected invalid contract error, got: %v", err)
	}
}

func TestReport_ContractLacksH1(t *testing.T) {
	repoPath := createValidTestRepo(t)
	// Modify a contract file to lack an H1
	os.WriteFile(filepath.Join(repoPath, "modules", "node-troll", "INTENT.md"), []byte("invalid content\nno h1 here"), 0644)

	_, err := Report(repoPath)
	if err == nil {
		t.Fatal("expected error for contract lacking H1, got nil")
	}
	if !strings.Contains(err.Error(), "invalid contract file") {
		t.Errorf("expected invalid contract error, got: %v", err)
	}
}

func TestReport_NoAbsolutePathsOrTimestamps(t *testing.T) {
	repoPath := createValidTestRepo(t)

	output, _ := Report(repoPath)
	for _, line := range output {
		if strings.Contains(line, repoPath) {
			t.Errorf("output contains absolute path: %s", line)
		}
		if strings.Contains(line, ":") && !strings.Contains(line, "status:") && !strings.Contains(line, "qxctl.status.v1") {
			// Basic heuristic to check for timestamp formats (e.g. 12:34:56)
			// A true timestamp check might be more rigorous, but since we know we only output "status: ...", this is a good sanity check
			if strings.Count(line, ":") > 1 {
				t.Errorf("output might contain a timestamp: %s", line)
			}
		}
	}

	jsonBytes, _ := ReportJSON(repoPath)
	jsonStr := string(jsonBytes)
	if strings.Contains(jsonStr, repoPath) {
		t.Errorf("JSON output contains absolute path: %s", jsonStr)
	}
}
