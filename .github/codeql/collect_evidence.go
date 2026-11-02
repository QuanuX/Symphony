// Collect original CodeQL observations without asserting security clearance.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
)

const maxSARIFBytes int64 = 256 * 1024 * 1024

type sarifRun struct {
	Tool        json.RawMessage   `json:"tool"`
	Automation  json.RawMessage   `json:"automationDetails"`
	Provenance  []json.RawMessage `json:"versionControlProvenance"`
	Results     []json.RawMessage `json:"results"`
	Artifacts   []json.RawMessage `json:"artifacts"`
	Invocations []json.RawMessage `json:"invocations"`
	Properties  json.RawMessage   `json:"properties"`
}

type runEvidence struct {
	Tool              json.RawMessage   `json:"tool"`
	Automation        json.RawMessage   `json:"automation_details"`
	Provenance        []json.RawMessage `json:"version_control_provenance"`
	ProvenancePresent bool              `json:"provenance_present_in_original_sarif"`
	ResultCount       int               `json:"result_count"`
	ArtifactCount     int               `json:"artifact_count"`
	Invocations       []json.RawMessage `json:"invocations"`
	NotificationCount int               `json:"diagnostic_notification_count"`
	Notifications     []json.RawMessage `json:"diagnostic_notifications"`
	Properties        json.RawMessage   `json:"properties"`
	Coverage          string            `json:"coverage_completeness"`
}

type fileEvidence struct {
	File   string        `json:"file"`
	Bytes  int           `json:"bytes"`
	SHA256 string        `json:"sha256"`
	Runs   []runEvidence `json:"runs"`
}

type evidence struct {
	FormatVersion int            `json:"format_version"`
	SourceCommit  string         `json:"source_commit"`
	Category      string         `json:"category"`
	Status        string         `json:"status"`
	Clearance     string         `json:"security_clearance"`
	OriginalSARIF []fileEvidence `json:"original_sarif"`
	Limits        []string       `json:"limits"`
	Ref           string         `json:"ref,omitempty"`
	Language      string         `json:"language,omitempty"`
	Runner        string         `json:"runner,omitempty"`
	BuildMode     string         `json:"build_mode,omitempty"`
}

func defaultObject(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{}`)
	}
	return raw
}

func nonnilArray(raw []json.RawMessage) []json.RawMessage {
	if raw == nil {
		return []json.RawMessage{}
	}
	return raw
}

func requireObject(raw json.RawMessage, nonnullFields ...string) error {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return errors.New("invalid SARIF: expected a non-null object")
	}
	for _, name := range nonnullFields {
		if bytes.Equal(bytes.TrimSpace(object[name]), []byte("null")) {
			return fmt.Errorf("invalid SARIF: %s must not be null", name)
		}
	}
	return nil
}

func readSARIF(path string) ([]byte, error) {
	before, err := os.Lstat(path)
	if err != nil || !before.Mode().IsRegular() || before.Size() > maxSARIFBytes {
		return nil, errors.New("SARIF must be a bounded regular file")
	}
	// The collector runs on the declared macOS/Linux jobs. Refuse symlink
	// substitution and avoid blocking if a nonregular file replaces the input.
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, fmt.Errorf("open SARIF: %w", err)
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(before, opened) || opened.Size() > maxSARIFBytes {
		return nil, errors.New("SARIF must be a bounded regular file")
	}
	raw, err := io.ReadAll(io.LimitReader(f, maxSARIFBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > maxSARIFBytes {
		return nil, errors.New("SARIF must be a bounded regular file")
	}
	return raw, nil
}

func summarize(root, commit, category string) (evidence, error) {
	var result evidence
	if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(commit) {
		return result, errors.New("an exact source commit is required")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return result, err
	}
	// ReadDir sorts by filename and treats metacharacters in root literally.
	paths := []string{}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".sarif") {
			paths = append(paths, filepath.Join(root, entry.Name()))
		}
	}
	if len(paths) == 0 {
		return result, errors.New("original analyzer SARIF is absent")
	}
	result = evidence{FormatVersion: 1, SourceCommit: commit, Category: category,
		Status: "evidence_collected", Clearance: "not_asserted", OriginalSARIF: []fileEvidence{},
		Limits: []string{
			"A successful analysis is not zero vulnerabilities.",
			"No missing diagnostic or metric is interpreted as complete coverage.",
			"PR filtering does not close an unchanged default-branch alert.",
			"C++26 is outside the published CodeQL language support matrix.",
		}}
	for _, path := range paths {
		raw, err := readSARIF(path)
		if err != nil {
			return evidence{}, err
		}
		var document struct {
			Version string            `json:"version"`
			Runs    []json.RawMessage `json:"runs"`
		}
		if err := json.Unmarshal(raw, &document); err != nil {
			return evidence{}, fmt.Errorf("invalid SARIF document: %w", err)
		}
		if document.Version != "2.1.0" || len(document.Runs) == 0 {
			return evidence{}, errors.New("invalid or empty SARIF document")
		}
		digest := sha256.Sum256(raw)
		file := fileEvidence{File: filepath.Base(path), Bytes: len(raw), SHA256: hex.EncodeToString(digest[:]), Runs: []runEvidence{}}
		for _, rawRun := range document.Runs {
			if err := requireObject(rawRun, "versionControlProvenance", "results", "artifacts", "invocations"); err != nil {
				return evidence{}, err
			}
			var run sarifRun
			if err := json.Unmarshal(rawRun, &run); err != nil {
				return evidence{}, fmt.Errorf("invalid SARIF run: %w", err)
			}
			var automation struct {
				ID string `json:"id"`
			}
			if err := requireObject(defaultObject(run.Automation)); err != nil {
				return evidence{}, err
			}
			if err := json.Unmarshal(defaultObject(run.Automation), &automation); err != nil {
				return evidence{}, fmt.Errorf("invalid SARIF automation details: %w", err)
			}
			if automation.ID != "" && strings.TrimRight(automation.ID, "/") != strings.TrimRight(category, "/") {
				return evidence{}, errors.New("SARIF category differs from the selected job category")
			}
			for _, rawProvenance := range run.Provenance {
				if err := requireObject(rawProvenance); err != nil {
					return evidence{}, err
				}
				var provenance struct {
					Revision *string `json:"revisionId"`
				}
				if err := json.Unmarshal(rawProvenance, &provenance); err != nil {
					return evidence{}, fmt.Errorf("invalid SARIF source provenance: %w", err)
				}
				if provenance.Revision != nil && *provenance.Revision != commit {
					return evidence{}, errors.New("SARIF provenance differs from the checked-out commit")
				}
			}
			notifications := []json.RawMessage{}
			for _, rawInvocation := range run.Invocations {
				if err := requireObject(rawInvocation, "toolExecutionNotifications", "toolConfigurationNotifications"); err != nil {
					return evidence{}, err
				}
				var invocation struct {
					Execution     []json.RawMessage `json:"toolExecutionNotifications"`
					Configuration []json.RawMessage `json:"toolConfigurationNotifications"`
				}
				if err := json.Unmarshal(rawInvocation, &invocation); err != nil {
					return evidence{}, fmt.Errorf("invalid SARIF invocation: %w", err)
				}
				notifications = append(notifications, invocation.Execution...)
				notifications = append(notifications, invocation.Configuration...)
			}
			file.Runs = append(file.Runs, runEvidence{Tool: defaultObject(run.Tool), Automation: defaultObject(run.Automation),
				Provenance: nonnilArray(run.Provenance), ProvenancePresent: len(run.Provenance) != 0,
				ResultCount: len(run.Results), ArtifactCount: len(run.Artifacts), Invocations: nonnilArray(run.Invocations),
				NotificationCount: len(notifications), Notifications: notifications, Properties: defaultObject(run.Properties), Coverage: "not_asserted"})
		}
		result.OriginalSARIF = append(result.OriginalSARIF, file)
	}
	return result, nil
}

func collect(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("collect_evidence", flag.ContinueOnError)
	flags.SetOutput(stderr)
	values := map[string]*string{}
	for _, name := range []string{"sarif-root", "output", "commit", "category", "ref", "language", "runner", "build-mode"} {
		values[name] = flags.String(name, "", "required: "+name)
	}
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	for name, value := range values {
		if *value == "" {
			return fmt.Errorf("--%s is required", name)
		}
	}
	result, err := summarize(*values["sarif-root"], *values["commit"], *values["category"])
	if err != nil {
		return err
	}
	result.Ref, result.Language = *values["ref"], *values["language"]
	result.Runner, result.BuildMode = *values["runner"], *values["build-mode"]
	raw, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	output := *values["output"]
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(output, append(raw, '\n'), 0644); err != nil {
		return err
	}
	fmt.Fprintln(stdout, "Collected original CodeQL evidence; security and full coverage are not asserted")
	return nil
}

func main() {
	if err := collect(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
