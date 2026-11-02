package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

const testCommit = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const testCategory = "/language:c-cpp/os:ubuntu-24.04"

func sarifFixture(t *testing.T, root, name, raw string) string {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func requireError(t *testing.T, err error, contains string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), contains) {
		t.Fatalf("want refusal containing %q, got %v", contains, err)
	}
}

func TestMissingAnalysisCannotBeACleanScan(t *testing.T) {
	_, err := summarize(t.TempDir(), testCommit, testCategory)
	requireError(t, err, "absent")
}

func TestZeroResultsKeepsDiagnosticsAndNoClearance(t *testing.T) {
	root := t.TempDir()
	raw := `{"version":"2.1.0","runs":[{"results":[],"invocations":[{"toolExecutionNotifications":[{"level":"warning","message":{"text":"missing header"}}],"toolConfigurationNotifications":[{"message":{"text":"configuration diagnostic"}}],"properties":{"unknownOriginalField":true}}],"properties":{"metricResults":[{"value":575}],"exactLargeInteger":9007199254740993}}]}`
	sarifFixture(t, root, "cpp.sarif", raw)
	report, err := summarize(root, testCommit, testCategory)
	if err != nil {
		t.Fatal(err)
	}
	file, run := report.OriginalSARIF[0], report.OriginalSARIF[0].Runs[0]
	if report.Clearance != "not_asserted" || run.Coverage != "not_asserted" || run.ProvenancePresent || run.ResultCount != 0 {
		t.Fatal("missing provenance and zero results must not become clearance or coverage claims")
	}
	if run.NotificationCount != 2 || !bytes.Contains(run.Notifications[0], []byte("missing header")) || !bytes.Contains(run.Notifications[1], []byte("configuration diagnostic")) {
		t.Fatal("original execution/configuration diagnostics were not retained")
	}
	if !bytes.Contains(run.Properties, []byte(`"value":575`)) || !bytes.Contains(run.Properties, []byte(`9007199254740993`)) || !bytes.Contains(run.Invocations[0], []byte("unknownOriginalField")) {
		t.Fatal("original metrics or invocation properties were changed")
	}
	digest := sha256.Sum256([]byte(raw))
	if file.SHA256 != hex.EncodeToString(digest[:]) || file.Bytes != len(raw) {
		t.Fatal("summary must bind exact original bytes")
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(encoded, []byte(`"version_control_provenance":[]`)) || !bytes.Contains(encoded, []byte(`9007199254740993`)) {
		t.Fatal("JSON output changed absent-array semantics or rounded an original metric")
	}
}

func TestOtherCategoryCannotReplaceSelectedEvidence(t *testing.T) {
	root := t.TempDir()
	sarifFixture(t, root, "cpp.sarif", `{"version":"2.1.0","runs":[{"automationDetails":{"id":"/language:c-cpp/os:macos-15-intel/"}}]}`)
	_, err := summarize(root, testCommit, testCategory)
	requireError(t, err, "category differs")
}

func TestOtherSourceRevisionIsRejected(t *testing.T) {
	root := t.TempDir()
	sarifFixture(t, root, "cpp.sarif", `{"version":"2.1.0","runs":[{"versionControlProvenance":[{"revisionId":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}]}]}`)
	_, err := summarize(root, testCommit, testCategory)
	requireError(t, err, "provenance differs")
}

func TestExactBindingAndDeterministicFileOrder(t *testing.T) {
	root := filepath.Join(t.TempDir(), "literal[1]")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	raw := `{"version":"2.1.0","runs":[{"automationDetails":{"id":"` + testCategory + `/"},"versionControlProvenance":[{"revisionId":"` + testCommit + `","repositoryUri":"https://github.com/QuanuX/Symphony"}],"tool":{"driver":{"version":"2.27.3"}}}]}`
	sarifFixture(t, root, "z.sarif", raw)
	sarifFixture(t, root, "a.sarif", raw)
	report, err := summarize(root, testCommit, testCategory)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.OriginalSARIF) != 2 || report.OriginalSARIF[0].File != "a.sarif" || report.OriginalSARIF[1].File != "z.sarif" || !report.OriginalSARIF[0].Runs[0].ProvenancePresent {
		t.Fatal("exact source/category evidence or deterministic file ordering changed")
	}
}

func TestInvalidSourceAndDocumentsAreRejected(t *testing.T) {
	_, err := summarize(t.TempDir(), "main", testCategory)
	requireError(t, err, "exact source commit")
	for _, raw := range []string{`{`, `{"version":"2.0.0","runs":[{}]}`, `{"version":"2.1.0","runs":[]}`} {
		root := t.TempDir()
		sarifFixture(t, root, "cpp.sarif", raw)
		_, err = summarize(root, testCommit, testCategory)
		requireError(t, err, "invalid")
	}
}

func TestNonregularAndOversizeInputsAreRejected(t *testing.T) {
	for _, kind := range []string{"symlink", "directory", "fifo", "oversize"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "cpp.sarif")
			var err error
			switch kind {
			case "symlink":
				target := sarifFixture(t, root, "original.json", `{"version":"2.1.0","runs":[{}]}`)
				err = os.Symlink(target, path)
			case "directory":
				err = os.Mkdir(path, 0700)
			case "fifo":
				err = syscall.Mkfifo(path, 0600)
			case "oversize":
				sarifFixture(t, root, "cpp.sarif", "")
				err = os.Truncate(path, maxSARIFBytes+1)
			}
			if err != nil {
				t.Fatal(err)
			}
			_, err = summarize(root, testCommit, testCategory)
			requireError(t, err, "bounded regular file")
		})
	}
}

func TestMalformedNullEvidenceIsRejected(t *testing.T) {
	for _, run := range []string{
		`null`, `{"automationDetails":null}`,
		`{"versionControlProvenance":null}`, `{"versionControlProvenance":[null]}`,
		`{"results":null}`, `{"artifacts":null}`,
		`{"invocations":null}`, `{"invocations":[null]}`,
		`{"invocations":[{"toolExecutionNotifications":null}]}`,
		`{"invocations":[{"toolConfigurationNotifications":null}]}`,
	} {
		t.Run(run, func(t *testing.T) {
			root := t.TempDir()
			sarifFixture(t, root, "cpp.sarif", `{"version":"2.1.0","runs":[`+run+`]}`)
			_, err := summarize(root, testCommit, testCategory)
			requireError(t, err, "invalid SARIF")
		})
	}
}

func TestCommandPreservesSchemaAndRequiresMetadata(t *testing.T) {
	root := t.TempDir()
	sarifFixture(t, root, "cpp.sarif", `{"version":"2.1.0","runs":[{}]}`)
	output := filepath.Join(root, "output", "summary.json")
	args := []string{"--sarif-root", root, "--output", output, "--commit", testCommit, "--category", testCategory,
		"--ref", "refs/pull/179/merge", "--language", "c-cpp", "--runner", "ubuntu-24.04", "--build-mode", "none"}
	var stdout, stderr bytes.Buffer
	if err := collect(args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	var report evidence
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatal(err)
	}
	if report.FormatVersion != 1 || report.SourceCommit != testCommit || report.Category != testCategory || report.Ref != "refs/pull/179/merge" || report.Language != "c-cpp" || report.Runner != "ubuntu-24.04" || report.BuildMode != "none" || report.Status != "evidence_collected" {
		t.Fatal("CLI output lost its exact source/job metadata")
	}
	requireError(t, collect(args[:len(args)-2], &stdout, &stderr), "--build-mode is required")
}
