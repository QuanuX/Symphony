package knowledgelifecycle

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	HostIntegrationProtocol = "symphony.knowledge.lifecycle-host-integration.v1"
	HostResultProtocol      = "symphony.knowledge.lifecycle-host-integration-result.v1"
	maxHostDescriptorBytes  = 1 << 20
)

type HostExecutor struct {
	Digest string `json:"digest"`
	Path   string `json:"path"`
}

type HostIntegration struct {
	Protocol                  string         `json:"protocol"`
	FormatVersion             uint64         `json:"format_version"`
	TOPSID                    string         `json:"tops_id"`
	ProfileID                 string         `json:"profile_id"`
	Scope                     string         `json:"scope"`
	Manager                   string         `json:"manager"`
	State                     string         `json:"state"`
	DesiredEnabled            bool           `json:"desired_enabled"`
	BootPolicy                string         `json:"boot_policy"`
	RecoveryMode              string         `json:"recovery_mode"`
	StateRoot                 string         `json:"state_root"`
	RepositoryRoot            string         `json:"repository_root"`
	IntegrationRoot           string         `json:"integration_root"`
	UnitName                  string         `json:"unit_name"`
	UnitDigest                string         `json:"unit_digest"`
	ActiveExecutor            HostExecutor   `json:"active_executor"`
	FallbackExecutors         []HostExecutor `json:"fallback_executors"`
	Generation                uint64         `json:"generation"`
	PreviousIntegrationDigest *string        `json:"previous_integration_digest"`
	UpdatedAt                 string         `json:"updated_at"`
	Canonical                 bool           `json:"canonical"`
	IntegrationDigest         string         `json:"integration_digest"`
}

type HostSnapshot struct {
	Exists      bool            `json:"exists"`
	Integration HostIntegration `json:"integration"`
}

type HostDesired struct {
	ProfileID       string
	RepositoryRoot  string
	IntegrationRoot string
	DesiredEnabled  bool
	RecoveryMode    string
	Executor        HostExecutor
	Fallbacks       []HostExecutor
	UnitDigest      string
	State           string
}

func (s *Store) PrepareHostDesired(desired HostDesired) (HostDesired, []byte, error) {
	desired.UnitDigest = "sha256:" + strings.Repeat("0", 64)
	if err := validateHostDesired(s.topsID, desired); err != nil {
		return desired, nil, err
	}
	preview := HostIntegration{
		Protocol: HostIntegrationProtocol, FormatVersion: 1, TOPSID: s.topsID,
		ProfileID: desired.ProfileID, Scope: "system", Manager: "systemd", State: desired.State,
		DesiredEnabled: desired.DesiredEnabled, BootPolicy: "report_only", RecoveryMode: desired.RecoveryMode,
		StateRoot: s.stateRoot, RepositoryRoot: desired.RepositoryRoot,
		IntegrationRoot: desired.IntegrationRoot, UnitName: mustHostUnitName(s.topsID, desired.ProfileID),
		UnitDigest: desired.UnitDigest, ActiveExecutor: desired.Executor,
		FallbackExecutors: canonicalExecutors(desired.Fallbacks, desired.Executor.Digest),
		Generation:        1, UpdatedAt: time.Unix(0, 0).UTC().Format(time.RFC3339),
		IntegrationDigest: "sha256:" + strings.Repeat("0", 64),
	}
	content, err := RenderHostSystemdUnit(preview)
	if err != nil {
		return desired, nil, err
	}
	desired.UnitDigest = DigestHostUnit(content)
	return desired, content, nil
}

func HostUnitName(topsID, profileID string) (string, error) {
	if !validTOPSID(topsID) || !safeToken(profileID, 256) {
		return "", fmt.Errorf("host integration identity is invalid")
	}
	digest := sha256.Sum256([]byte("symphony-qxctl-lifecycle\n" + topsID + "\n" + profileID))
	return "symphony-qxctl-lifecycle@" + topsID + "-" + hex.EncodeToString(digest[:]) + ".service", nil
}

func HostProfileKey(profileID string) string {
	digest := sha256.Sum256([]byte("host-integration\n" + profileID))
	return hex.EncodeToString(digest[:])
}

func (s *Store) HostSnapshot(profileID string) (HostSnapshot, error) {
	if !safeToken(profileID, 256) {
		return HostSnapshot{}, fmt.Errorf("profile ID has invalid syntax")
	}
	var snapshot HostSnapshot
	err := s.withProfileLock(false, func(directory *os.File) error {
		data, exists, err := readStateFile(directory, hostFileName(profileID), maxHostDescriptorBytes, "lifecycle host integration")
		if err != nil || !exists {
			snapshot.Exists = exists
			return err
		}
		record, err := decodeHostIntegration(data)
		if err != nil {
			return err
		}
		if record.TOPSID != s.topsID || record.ProfileID != profileID {
			return fmt.Errorf("lifecycle host integration storage identity mismatch")
		}
		snapshot = HostSnapshot{Exists: true, Integration: record}
		return nil
	})
	return snapshot, err
}

func (s *Store) CommitHost(desired HostDesired, expected string, now time.Time) (HostIntegration, bool, error) {
	if !safeToken(desired.ProfileID, 256) || (expected != "absent" && !taggedDigest(expected)) {
		return HostIntegration{}, false, fmt.Errorf("host integration compare-and-swap input is invalid")
	}
	if err := validateHostDesired(s.topsID, desired); err != nil {
		return HostIntegration{}, false, err
	}
	var committed HostIntegration
	changed := false
	err := s.withProfileLock(true, func(directory *os.File) error {
		_, profileExists, err := readStateFile(directory, profileFileName(desired.ProfileID), maxProfileBytes, "lifecycle profile")
		if err != nil {
			return err
		}
		if !profileExists {
			return fmt.Errorf("lifecycle profile %q is absent", desired.ProfileID)
		}
		data, exists, err := readStateFile(directory, hostFileName(desired.ProfileID), maxHostDescriptorBytes, "lifecycle host integration")
		if err != nil {
			return err
		}
		actual := "absent"
		var previous *HostIntegration
		if exists {
			decoded, err := decodeHostIntegration(data)
			if err != nil {
				return err
			}
			actual = decoded.IntegrationDigest
			previous = &decoded
		}
		if actual != expected {
			return fmt.Errorf("host integration compare-and-swap mismatch: current state is %s", actual)
		}
		generation := uint64(1)
		var previousDigest *string
		if previous != nil {
			generation = previous.Generation + 1
			value := previous.IntegrationDigest
			previousDigest = &value
		}
		committed = HostIntegration{
			Protocol: HostIntegrationProtocol, FormatVersion: 1, TOPSID: s.topsID,
			ProfileID: desired.ProfileID, Scope: "system", Manager: "systemd", State: desired.State,
			DesiredEnabled: desired.DesiredEnabled, BootPolicy: "report_only", RecoveryMode: desired.RecoveryMode,
			StateRoot: s.stateRoot, RepositoryRoot: desired.RepositoryRoot,
			IntegrationRoot: desired.IntegrationRoot, UnitName: mustHostUnitName(s.topsID, desired.ProfileID),
			UnitDigest: desired.UnitDigest, ActiveExecutor: desired.Executor,
			FallbackExecutors: canonicalExecutors(desired.Fallbacks, desired.Executor.Digest),
			Generation:        generation, PreviousIntegrationDigest: previousDigest,
			UpdatedAt: now.UTC().Truncate(time.Second).Format(time.RFC3339), Canonical: false,
		}
		digest, err := digestHostIntegration(committed)
		if err != nil {
			return err
		}
		committed.IntegrationDigest = digest
		encoded, err := json.Marshal(committed)
		if err != nil {
			return err
		}
		if previous != nil && hostOperationallyEqual(*previous, committed) {
			committed = *previous
			return nil
		}
		if err := writeStateFile(directory, hostFileName(desired.ProfileID), append(encoded, '\n'), "lifecycle host integration"); err != nil {
			return err
		}
		changed = true
		return nil
	})
	return committed, changed, err
}

func (s *Store) RemoveHost(profileID, expected string) error {
	if !safeToken(profileID, 256) || !taggedDigest(expected) {
		return fmt.Errorf("host integration removal compare-and-swap input is invalid")
	}
	return s.withProfileLock(true, func(directory *os.File) error {
		data, exists, err := readStateFile(directory, hostFileName(profileID), maxHostDescriptorBytes, "lifecycle host integration")
		if err != nil {
			return err
		}
		if !exists {
			return nil
		}
		record, err := decodeHostIntegration(data)
		if err != nil {
			return err
		}
		if record.IntegrationDigest != expected {
			return fmt.Errorf("host integration compare-and-swap mismatch: current state is %s", record.IntegrationDigest)
		}
		return removeStateFile(directory, hostFileName(profileID), "lifecycle host integration")
	})
}

func RenderHostSystemdUnit(record HostIntegration) ([]byte, error) {
	if err := validateHostIntegration(record); err != nil {
		return nil, err
	}
	arguments := []string{
		record.ActiveExecutor.Path, "knowledge", "lifecycle", "host", "run",
		"--scope", "system", "--tops-id", record.TOPSID, "--profile-id", record.ProfileID,
		"--state-root", record.StateRoot,
	}
	quoted := make([]string, len(arguments))
	for index, argument := range arguments {
		value, err := systemdQuote(argument)
		if err != nil {
			return nil, err
		}
		quoted[index] = value
	}
	content := "[Unit]\n" +
		"Description=Symphony qxctl report-only lifecycle convergence for TOPS " + record.TOPSID + "\n" +
		"After=local-fs.target symphony-ssiag@" + record.TOPSID + ".service symphony-stav@" + record.TOPSID + ".service\n" +
		"Wants=symphony-ssiag@" + record.TOPSID + ".service symphony-stav@" + record.TOPSID + ".service\n" +
		"StartLimitIntervalSec=300\nStartLimitBurst=5\n\n[Service]\n" +
		"Type=oneshot\nExecStart=" + strings.Join(quoted, " ") + "\n" +
		"RemainAfterExit=yes\nRestart=on-failure\nRestartSec=15s\nTimeoutStartSec=5min\n" +
		"UMask=0077\nNoNewPrivileges=true\nPrivateTmp=true\n\n[Install]\nWantedBy=multi-user.target\n"
	return []byte(content), nil
}

func DigestHostUnit(content []byte) string {
	digest := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func decodeHostIntegration(data []byte) (HostIntegration, error) {
	var record HostIntegration
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&record); err != nil {
		return record, fmt.Errorf("decode lifecycle host integration: %w", err)
	}
	if err := validateHostIntegration(record); err != nil {
		return record, err
	}
	expected, err := digestHostIntegration(record)
	if err != nil || expected != record.IntegrationDigest {
		return record, fmt.Errorf("lifecycle host integration digest mismatch")
	}
	return record, nil
}

func validateHostDesired(topsID string, desired HostDesired) error {
	if !validTOPSID(topsID) || !safeToken(desired.ProfileID, 256) ||
		(desired.State != "installed" && desired.State != "retiring") ||
		(desired.RecoveryMode != "strict" && desired.RecoveryMode != "discover") ||
		!safeHostPath(desired.RepositoryRoot) || !safeHostPath(desired.IntegrationRoot) ||
		!taggedDigest(desired.UnitDigest) || !validHostExecutor(desired.Executor, desired.IntegrationRoot) || len(desired.Fallbacks) > 8 {
		return fmt.Errorf("lifecycle host integration desired state is invalid")
	}
	for _, executor := range desired.Fallbacks {
		if !validHostExecutor(executor, desired.IntegrationRoot) {
			return fmt.Errorf("lifecycle host fallback executor is invalid")
		}
	}
	return nil
}

func validateHostIntegration(record HostIntegration) error {
	if record.Protocol != HostIntegrationProtocol || record.FormatVersion != 1 || !validTOPSID(record.TOPSID) ||
		!safeToken(record.ProfileID, 256) || record.Scope != "system" || record.Manager != "systemd" ||
		(record.State != "installed" && record.State != "retiring") || record.BootPolicy != "report_only" ||
		(record.RecoveryMode != "strict" && record.RecoveryMode != "discover") || record.Canonical ||
		!safeHostPath(record.StateRoot) || !safeHostPath(record.RepositoryRoot) || !safeHostPath(record.IntegrationRoot) ||
		!safeHostUnitName(record.UnitName) || record.UnitName != mustHostUnitName(record.TOPSID, record.ProfileID) ||
		!taggedDigest(record.UnitDigest) || !validHostExecutor(record.ActiveExecutor, record.IntegrationRoot) ||
		record.Generation == 0 || record.Generation > 9007199254740991 ||
		(record.PreviousIntegrationDigest != nil && !taggedDigest(*record.PreviousIntegrationDigest)) ||
		!validSecondUTC(record.UpdatedAt) || !taggedDigest(record.IntegrationDigest) || len(record.FallbackExecutors) > 8 {
		return fmt.Errorf("lifecycle host integration contract is invalid")
	}
	seen := map[string]bool{record.ActiveExecutor.Digest: true}
	for _, executor := range record.FallbackExecutors {
		if !validHostExecutor(executor, record.IntegrationRoot) || seen[executor.Digest] {
			return fmt.Errorf("lifecycle host executor candidates are invalid")
		}
		seen[executor.Digest] = true
	}
	return nil
}

func digestHostIntegration(record HostIntegration) (string, error) {
	encoded, err := json.Marshal(record)
	if err != nil {
		return "", err
	}
	object, err := objectWithout(encoded, "integration_digest")
	if err != nil {
		return "", err
	}
	return digestValue(object)
}

func hostOperationallyEqual(left, right HostIntegration) bool {
	left.Generation, right.Generation = 0, 0
	left.PreviousIntegrationDigest, right.PreviousIntegrationDigest = nil, nil
	left.UpdatedAt, right.UpdatedAt = "", ""
	left.IntegrationDigest, right.IntegrationDigest = "", ""
	return fmt.Sprintf("%#v", left) == fmt.Sprintf("%#v", right)
}

func canonicalExecutors(values []HostExecutor, active string) []HostExecutor {
	seen := make(map[string]bool)
	result := make([]HostExecutor, 0, len(values))
	for _, value := range values {
		if value.Digest != active && !seen[value.Digest] {
			seen[value.Digest] = true
			result = append(result, value)
			if len(result) == 8 {
				break
			}
		}
	}
	return result
}

func validHostExecutor(value HostExecutor, integrationRoot string) bool {
	if !taggedDigest(value.Digest) || !safeHostPath(value.Path) || !safeHostPath(integrationRoot) ||
		filepath.Base(value.Path) != "qxctl" {
		return false
	}
	digestDirectory := filepath.Base(filepath.Dir(value.Path))
	executorsDirectory := filepath.Dir(filepath.Dir(value.Path))
	return digestDirectory == strings.TrimPrefix(value.Digest, "sha256:") &&
		filepath.Base(executorsDirectory) == "executors" && filepath.Dir(executorsDirectory) == integrationRoot
}

func safeHostPath(value string) bool {
	return filepath.IsAbs(value) && filepath.Clean(value) == value && value != string(filepath.Separator) &&
		!strings.ContainsAny(value, "\x00\n\r%$")
}

func systemdQuote(value string) (string, error) {
	if value == "" || strings.ContainsAny(value, "\x00\n\r%$") {
		return "", fmt.Errorf("systemd argument contains an unsafe character")
	}
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(value) + `"`, nil
}

func safeHostUnitName(value string) bool {
	if len(value) == 0 || len(value) > 256 || !strings.HasPrefix(value, "symphony-qxctl-lifecycle@") ||
		!strings.HasSuffix(value, ".service") {
		return false
	}
	for _, character := range value {
		if (character >= 'a' && character <= 'z') || (character >= '0' && character <= '9') ||
			strings.ContainsRune("._@-", character) {
			continue
		}
		return false
	}
	return true
}

func validSecondUTC(value string) bool {
	parsed, err := time.Parse(time.RFC3339, value)
	return err == nil && parsed.Location() == time.UTC && parsed.Nanosecond() == 0 && parsed.Format(time.RFC3339) == value
}

func hostFileName(profileID string) string { return ".host-" + HostProfileKey(profileID) + ".json" }

func mustHostUnitName(topsID, profileID string) string {
	value, _ := HostUnitName(topsID, profileID)
	return value
}
