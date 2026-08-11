package validation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
)

const (
	maxValidatorOutput = 16 * 1024 * 1024
	maxValidatorStderr = 64 * 1024
	maxValidatorValues = 2 * 1024 * 1024
	validatorTimeout   = 30 * time.Second
)

type boundedBuffer struct {
	bytes.Buffer
	limit int
}

func (buffer *boundedBuffer) Write(data []byte) (int, error) {
	if buffer.Len()+len(data) > buffer.limit {
		return 0, fmt.Errorf("process output exceeds %d bytes", buffer.limit)
	}
	return buffer.Buffer.Write(data)
}

func Run(ctx context.Context, prefix, version, repositoryRoot string) (Result, error) {
	installation, err := knowledgeengine.InspectValidatorInstallation(prefix, version)
	if err != nil {
		return Result{}, fmt.Errorf("validator installation is unavailable: %w", err)
	}
	repositoryRoot, err = canonicalRepository(repositoryRoot)
	if err != nil {
		return Result{}, err
	}
	deadlineContext, cancel := context.WithTimeout(ctx, validatorTimeout)
	defer cancel()
	command := exec.CommandContext(
		deadlineContext, installation.ExecutablePath,
		"check", "--repo", repositoryRoot, "--json")
	command.Env = []string{}
	command.Dir = repositoryRoot
	stdout := &boundedBuffer{limit: maxValidatorOutput}
	stderr := &boundedBuffer{limit: maxValidatorStderr}
	command.Stdout = stdout
	command.Stderr = stderr
	runErr := command.Run()
	if deadlineContext.Err() != nil {
		return Result{}, fmt.Errorf("validator exceeded its hard process deadline")
	}
	exitCode := 0
	if runErr != nil {
		var exitError *exec.ExitError
		if !errors.As(runErr, &exitError) {
			return Result{}, fmt.Errorf("execute validator: %w", runErr)
		}
		exitCode = exitError.ExitCode()
	}
	result, err := decodeResult(stdout.Bytes())
	if err != nil {
		return Result{}, err
	}
	if result.Evidence.ExitCode != exitCode {
		return Result{}, fmt.Errorf("validator process status does not match structured evidence")
	}
	if result.Evidence.ValidatorVersion != version || result.Evidence.RepositoryIdentityDigest != repositoryIdentity(repositoryRoot) {
		return Result{}, fmt.Errorf("validator result identity mismatch")
	}
	if stderr.Len() != 0 {
		return Result{}, fmt.Errorf("validator emitted unexpected diagnostics: %s", strings.TrimSpace(stderr.String()))
	}
	return result, nil
}

func canonicalRepository(path string) (string, error) {
	if path == "" {
		path = "."
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve repository root: %w", err)
	}
	requested, err := os.Lstat(absolute)
	if err != nil || requested.Mode()&os.ModeSymlink != 0 || !requested.IsDir() {
		return "", fmt.Errorf("repository root must identify a no-follow directory")
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("canonicalize repository root: %w", err)
	}
	actual, err := os.Lstat(resolved)
	if err != nil || actual.Mode()&os.ModeSymlink != 0 || !actual.IsDir() || !os.SameFile(requested, actual) {
		return "", fmt.Errorf("repository root changed during canonicalization")
	}
	return filepath.Clean(resolved), nil
}

func repositoryIdentity(repositoryRoot string) string {
	digest := sha256.Sum256([]byte(filepath.ToSlash(repositoryRoot)))
	return "sha256:" + hex.EncodeToString(digest[:])
}

func decodeResult(data []byte) (Result, error) {
	if err := knowledgeengine.ValidateJSONObjectWithValueLimit(data, maxValidatorOutput, maxValidatorValues); err != nil {
		return Result{}, fmt.Errorf("invalid validator JSON result: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var result Result
	if err := decoder.Decode(&result); err != nil {
		return Result{}, fmt.Errorf("decode validator result: %w", err)
	}
	if result.Protocol != ResultProtocol || result.FormatVersion != 1 || result.Evaluation != nil ||
		result.Evidence.ValidatorID != "symphony-validator" || !safeVersion(result.Evidence.ValidatorVersion) ||
		!taggedDigest(result.ResultDigest) || !taggedDigest(result.Evidence.EvidenceDigest) ||
		!taggedDigest(result.Evidence.RepositoryIdentityDigest) ||
		(result.Evidence.Outcome != "pass" && result.Evidence.Outcome != "violation") ||
		result.Evidence.ExitCode < 0 || result.Evidence.ExitCode > 255 || len(result.Evidence.Findings) > 16384 {
		return Result{}, fmt.Errorf("validator result identity, category, or bound is invalid")
	}
	if result.Evidence.Outcome == "pass" && result.Evidence.ExitCode != 0 ||
		result.Evidence.Outcome == "violation" && result.Evidence.ExitCode == 0 {
		return Result{}, fmt.Errorf("validator result outcome and exit status disagree")
	}
	if err := validateFindings(result.Evidence); err != nil {
		return Result{}, err
	}
	evidenceBasis, err := objectWithout(result.Evidence, "evidence_digest")
	if err != nil {
		return Result{}, err
	}
	expectedEvidence, err := digestValue(evidenceBasis)
	if err != nil || expectedEvidence != result.Evidence.EvidenceDigest {
		return Result{}, fmt.Errorf("validator evidence digest mismatch")
	}
	resultBasis, err := objectWithout(result, "result_digest")
	if err != nil {
		return Result{}, err
	}
	expectedResult, err := digestValue(resultBasis)
	if err != nil || expectedResult != result.ResultDigest {
		return Result{}, fmt.Errorf("validator result digest mismatch")
	}
	return result, nil
}

func validateFindings(evidence Evidence) error {
	counts := Summary{}
	allowedCategories := map[string]bool{
		"pass": true, "warning": true, "violation": true, "deferred": true,
		"absent": true, "stale": true, "unknown": true, "blocked": true,
	}
	allowedScopes := map[string]bool{"active": true, "historical": true, "system": true}
	for _, finding := range evidence.Findings {
		if !allowedCategories[finding.Category] || !allowedScopes[finding.Scope] ||
			!safeToken(finding.RuleID) || !taggedDigest(finding.OccurrenceID) || !taggedDigest(finding.SubjectID) ||
			len(finding.Detail) > 65536 || len(finding.Attributes) > 32 {
			return fmt.Errorf("validator finding identity or bound is invalid")
		}
		for key, value := range finding.Attributes {
			if !safeToken(key) || len(value) > 4096 {
				return fmt.Errorf("validator finding attribute is invalid")
			}
		}
		expectedOccurrence := findingDigest(finding, false)
		expectedSubject := findingDigest(finding, true)
		if finding.OccurrenceID != expectedOccurrence || finding.SubjectID != expectedSubject {
			return fmt.Errorf("validator finding digest mismatch")
		}
		switch finding.Category {
		case "pass":
			counts.Pass++
		case "warning":
			counts.Warning++
		case "violation":
			counts.Violation++
		default:
			counts.Other++
		}
		counts.Total++
	}
	if counts != evidence.Summary {
		return fmt.Errorf("validator finding summary mismatch")
	}
	return nil
}

func findingDigest(finding Finding, subject bool) string {
	basis := finding.Category + "\n" + finding.RuleID + "\n" + finding.Scope + "\n" + finding.Detail
	if subject && finding.RuleID == "sclv.affected_surface.unindexed" {
		if path := finding.Attributes["path"]; path != "" {
			basis = finding.RuleID + "\npath=" + path
		}
	}
	digest := sha256.Sum256([]byte(basis))
	return "sha256:" + hex.EncodeToString(digest[:])
}

func warningIDs(result Result) ([]string, []string) {
	occurrences := make([]string, 0)
	subjects := make([]string, 0)
	for _, finding := range result.Evidence.Findings {
		if finding.Category == "warning" {
			occurrences = append(occurrences, finding.OccurrenceID)
			subjects = append(subjects, finding.SubjectID)
		}
	}
	sort.Strings(occurrences)
	sort.Strings(subjects)
	subjects = compactStrings(subjects)
	return occurrences, subjects
}

func compactStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	result := values[:1]
	for _, value := range values[1:] {
		if value != result[len(result)-1] {
			result = append(result, value)
		}
	}
	return result
}
