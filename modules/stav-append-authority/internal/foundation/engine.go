package foundation

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/config"
	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/lifecycle"
	stavpaths "github.com/QuanuX/Symphony/modules/stav-append-authority/internal/paths"
	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/supervision"
	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/version"
)

const (
	maxRequestBytes  = 1 << 20
	maxResponseBytes = 1 << 20
	maxDeadline      = 60 * time.Second
)

var tokenPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,256}$`)
var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
var digestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
var afterPreparedHook func() error
var currentExecutable = os.Executable

func DecodeCommand(reader io.Reader) (Command, error) {
	limited := io.LimitReader(reader, maxRequestBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return Command{}, err
	}
	if len(data) == 0 || len(data) > maxRequestBytes {
		return Command{}, fmt.Errorf("foundation lifecycle command exceeds bounded input")
	}
	if err := validateJSONBounds(data); err != nil {
		return Command{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var command Command
	if err := decoder.Decode(&command); err != nil {
		return Command{}, fmt.Errorf("decode foundation lifecycle command: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return Command{}, fmt.Errorf("foundation lifecycle command must contain exactly one JSON value")
	}
	return command, nil
}

func validateJSONBounds(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	values := 0
	var walk func(any, int) error
	walk = func(value any, depth int) error {
		if depth > 32 {
			return fmt.Errorf("foundation lifecycle JSON exceeds maximum depth")
		}
		values++
		if values > 4096 {
			return fmt.Errorf("foundation lifecycle JSON exceeds maximum value count")
		}
		switch current := value.(type) {
		case map[string]any:
			for _, child := range current {
				if err := walk(child, depth+1); err != nil {
					return err
				}
			}
		case []any:
			for _, child := range current {
				if err := walk(child, depth+1); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return walk(value, 1)
}

func EncodeBounded(writer io.Writer, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if len(data)+1 > maxResponseBytes {
		return fmt.Errorf("foundation lifecycle response exceeds bounded output")
	}
	_, err = writer.Write(append(data, '\n'))
	return err
}

func Describe() (AdapterDescriptor, error) {
	executable, err := currentExecutable()
	if err != nil {
		return AdapterDescriptor{}, err
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return AdapterDescriptor{}, err
	}
	var record lifecycle.InstallRecord
	var evidenceDigest string
	matched := false
	for _, scope := range []stavpaths.Scope{stavpaths.ScopeUser, stavpaths.ScopeSystem} {
		_, candidate, digest, verifyErr := lifecycle.VerifyExecutable(executable, scope)
		if verifyErr == nil {
			record, evidenceDigest, matched = candidate, digest, true
			break
		}
	}
	if !matched {
		return AdapterDescriptor{}, fmt.Errorf("foundation lifecycle descriptor requires the exact installed executable")
	}
	descriptor := AdapterDescriptor{
		Protocol: AdapterProtocol, FormatVersion: 1, Component: "stav", AdapterVersion: version.Version,
		BinaryPath: executable, BinaryDigest: "sha256:" + record.BinarySHA256, InstallEvidenceDigest: evidenceDigest,
		Operations: []string{
			"engop:symphony:stav.enrollment.observe", "engop:symphony:stav.enrollment.plan", "engop:symphony:stav.enrollment.apply", "engop:symphony:stav.enrollment.apply-status", "engop:symphony:stav.enrollment.recover",
			"engop:symphony:stav.supervisor.observe", "engop:symphony:stav.supervisor.plan", "engop:symphony:stav.supervisor.apply", "engop:symphony:stav.supervisor.apply-status", "engop:symphony:stav.supervisor.recover",
		},
		SupportedScopes: []string{"user", "system"}, SupportedManagers: []string{"launchd", "systemd", "external"},
		Compatibility:         AdapterCompatibility{ConfigReadMajors: []int{1}, ConfigWriteMajor: 1, RuntimeReadMajors: []int{1}, RuntimeWriteMajor: 1, StateReadMajors: []int{1}, StateWriteMajor: 1, RollbackReadable: true},
		Limits:                AdapterLimits{RequestBytes: maxRequestBytes, ResponseBytes: maxResponseBytes, DeadlineMS: int(maxDeadline / time.Millisecond), JSONDepth: 32, JSONValues: 4096},
		CanonicalApplyEnabled: false, NetworkListener: false,
	}
	descriptor.DescriptorDigest = digestWithout(descriptor, "descriptor_digest")
	return descriptor, nil
}

func Execute(command Command) (Result, error) {
	started := canonicalNow()
	if err := validateCommand(command); err != nil {
		return Result{}, err
	}
	scope, _ := parseScope(command.Scope)
	observation, observeErr := Observe(scope, command.TOPSID, command.Surface)
	if observeErr != nil {
		return Result{}, observeErr
	}
	base := Result{Protocol: ResultProtocol, FormatVersion: 1, Operation: command.Operation, Component: "stav", Surface: command.Surface, Scope: command.Scope, TOPSID: command.TOPSID, OperationID: command.OperationID, Observation: observation, AuditState: "not_applicable", StartedAt: &started, CompletedAt: canonicalNow(), Canonical: false}
	switch command.Operation {
	case "observe":
		base.Disposition, base.ReadOnly, base.StartedAt = "observed", true, nil
	case "plan":
		base.ReadOnly, base.StartedAt = true, nil
		plan, err := createPlan(command, observation)
		if err != nil {
			return failedResult(base, "invalid_plan", err, false), err
		}
		base.Disposition, base.Plan, base.DesiredState = "planned", &plan, &plan.DesiredState
	case "apply":
		return apply(command, base)
	case "apply_status":
		return applyStatus(command, base)
	case "recover":
		return recoverAttempt(command, base)
	}
	finalizeResult(&base)
	return base, nil
}

// DirectApply keeps the human CLI on the same plan/apply transaction engine.
func DirectApply(surface string, scope stavpaths.Scope, topsID, desiredState string, authorityUID, authorityGID *uint32, auditDeferred bool) (Result, error) {
	if !auditDeferred {
		return Result{}, fmt.Errorf("direct foundational mutation requires explicit --audit-deferred until a closed audit receipt is available")
	}
	observation, err := Observe(scope, topsID, surface)
	if err != nil {
		return Result{}, err
	}
	operationID := "direct:" + randomUUID()
	requestID, correlationID := randomUUID(), randomUUID()
	requested := time.Now().UTC().Truncate(time.Second)
	deadline := requested.Add(maxDeadline)
	command := Command{Protocol: CommandProtocol, FormatVersion: 1, Operation: "plan", Component: "stav", Surface: surface, Scope: string(scope), TOPSID: topsID, OperationID: &operationID, RequestID: &requestID, CorrelationID: &correlationID, ExpectedStateDigest: &observation.StableStateDigest, Intent: &Intent{DesiredState: desiredState, AuthorityUID: authorityUID, AuthorityGID: authorityGID, AuditMode: "audit_deferred", TTLSeconds: 60}, RequestedAt: formatTimestamp(requested), DeadlineAt: formatTimestamp(deadline)}
	planned, err := Execute(command)
	if err != nil {
		return planned, err
	}
	requested = time.Now().UTC().Truncate(time.Second)
	deadline = requested.Add(maxDeadline)
	command.Operation, command.Intent, command.Plan = "apply", nil, planned.Plan
	absent := "absent"
	command.ExpectedAttemptDigest = &absent
	command.RequestedAt, command.DeadlineAt = formatTimestamp(requested), formatTimestamp(deadline)
	return Execute(command)
}

func randomUUID() string {
	var data [16]byte
	if _, err := rand.Read(data[:]); err != nil {
		panic(err)
	}
	data[6] = (data[6] & 0x0f) | 0x40
	data[8] = (data[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", data[0:4], data[4:6], data[6:8], data[8:10], data[10:16])
}

func validateCommand(command Command) error {
	if command.Protocol != CommandProtocol || command.FormatVersion != 1 || command.Component != "stav" {
		return fmt.Errorf("unsupported foundation lifecycle command protocol or component")
	}
	if command.Operation != "observe" && command.Operation != "plan" && command.Operation != "apply" && command.Operation != "apply_status" && command.Operation != "recover" {
		return fmt.Errorf("unsupported foundation lifecycle operation")
	}
	if command.Surface != "enrollment" && command.Surface != "supervisor" {
		return fmt.Errorf("unsupported STAV foundation lifecycle surface")
	}
	if _, err := parseScope(command.Scope); err != nil {
		return err
	}
	if !uuidPattern.MatchString(command.TOPSID) || stavpaths.ValidateTOPSID(command.TOPSID) != nil {
		return fmt.Errorf("invalid TOPS UUID")
	}
	requested, err := parseTimestamp(command.RequestedAt)
	if err != nil {
		return fmt.Errorf("invalid requested_at")
	}
	deadline, err := parseTimestamp(command.DeadlineAt)
	if err != nil || !deadline.After(requested) || deadline.Sub(requested) > maxDeadline || !deadline.After(time.Now().UTC()) {
		return fmt.Errorf("invalid or expired foundation lifecycle deadline")
	}
	if command.Operation == "plan" && command.Intent == nil {
		return fmt.Errorf("plan requires intent")
	}
	if command.Operation == "apply" && command.Plan == nil {
		return fmt.Errorf("apply requires plan")
	}
	if (command.Operation == "apply" || command.Operation == "recover" || command.Operation == "plan") && !validToken(command.OperationID) {
		return fmt.Errorf("operation requires a valid operation_id")
	}
	return nil
}

func createPlan(command Command, observation Observation) (Plan, error) {
	intent := command.Intent
	if command.ExpectedStateDigest == nil || *command.ExpectedStateDigest != observation.StableStateDigest {
		return Plan{}, fmt.Errorf("expected state digest does not match current STAV state")
	}
	if command.RequestID == nil || command.CorrelationID == nil || !uuidPattern.MatchString(*command.RequestID) || !uuidPattern.MatchString(*command.CorrelationID) {
		return Plan{}, fmt.Errorf("plan requires request and correlation UUIDs")
	}
	if err := validateIntent(command.Surface, command.Scope, intent); err != nil {
		return Plan{}, err
	}
	created := time.Now().UTC().Truncate(time.Second)
	var uid, gid *uint32
	if command.Surface == "enrollment" && intent.DesiredState == "enrolled" {
		uid, gid = intent.AuthorityUID, intent.AuthorityGID
		if command.Scope == "user" {
			actualUID, actualGID := uint32(os.Geteuid()), uint32(os.Getegid())
			uid, gid = &actualUID, &actualGID
		}
	}
	plan := Plan{Protocol: PlanProtocol, FormatVersion: 1, Component: "stav", Surface: command.Surface, Scope: command.Scope, TOPSID: command.TOPSID, OperationID: *command.OperationID, RequestID: *command.RequestID, CorrelationID: *command.CorrelationID, ExpectedStateDigest: *command.ExpectedStateDigest, DesiredState: intent.DesiredState, AuthorityUID: uid, AuthorityGID: gid, AuditMode: intent.AuditMode, CreatedAt: formatTimestamp(created), ExpiresAt: formatTimestamp(created.Add(time.Duration(intent.TTLSeconds) * time.Second))}
	plan.PlanDigest = digestWithout(plan, "plan_digest")
	return plan, nil
}

func validateIntent(surface, scope string, intent *Intent) error {
	if intent == nil || intent.TTLSeconds < 1 || intent.TTLSeconds > 600 || (intent.AuditMode != "ordinary" && intent.AuditMode != "audit_deferred") {
		return fmt.Errorf("invalid lifecycle intent")
	}
	if intent.TOPSName != nil || intent.ServiceUID != nil || intent.ServiceGID != nil {
		return fmt.Errorf("STAV intent must not contain SSIAG fields")
	}
	if surface == "enrollment" && intent.DesiredState != "enrolled" && intent.DesiredState != "unenrolled_preserved" {
		return fmt.Errorf("invalid STAV enrollment desired state; purge is unavailable through machine v1")
	}
	if surface == "supervisor" && intent.DesiredState != "native_running" && intent.DesiredState != "native_installed_stopped" && intent.DesiredState != "absent_stopped" {
		return fmt.Errorf("invalid STAV supervisor desired state")
	}
	needsAuthorityIdentity := surface == "enrollment" && intent.DesiredState == "enrolled"
	if !needsAuthorityIdentity && (intent.AuthorityUID != nil || intent.AuthorityGID != nil) {
		return fmt.Errorf("authority UID/GID apply only to enrolled STAV enrollment")
	}
	if needsAuthorityIdentity && scope == "user" && (intent.AuthorityUID != nil || intent.AuthorityGID != nil) {
		return fmt.Errorf("user scope derives authority UID/GID from kernel credentials")
	}
	if needsAuthorityIdentity && scope == "system" && (intent.AuthorityUID == nil || intent.AuthorityGID == nil) {
		return fmt.Errorf("system STAV lifecycle requires explicit authority UID/GID")
	}
	return nil
}

func Observe(scope stavpaths.Scope, topsID, surface string) (Observation, error) {
	layout, err := stavpaths.ResolveInstance(scope, topsID)
	if err != nil {
		return Observation{}, err
	}
	observation := Observation{Protocol: ObservationProtocol, FormatVersion: 1, Component: "stav", Surface: surface, Scope: string(scope), TOPSID: topsID, Installation: Installation{State: "absent"}, Enrollment: Enrollment{State: "unenrolled"}, Supervisor: Supervisor{ManagerState: "unknown", DescriptorState: "unknown", Enablement: "unknown", ProcessState: "unknown", EndpointState: endpointState(layout.Socket)}, ObservedAt: canonicalNow()}
	install, installRecord, evidenceDigest, installVerifyErr := currentInstall(scope)
	if info, statErr := os.Lstat(install.Binary); statErr == nil && info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0 {
		observation.Installation.BinaryPath = stringPointer(install.Binary)
		if installVerifyErr == nil {
			observation.Installation.State = "installed"
			observation.Installation.BinaryDigest = stringPointer("sha256:" + installRecord.BinarySHA256)
			observation.Installation.InstallEvidenceDigest = &evidenceDigest
			observation.Installation.ReceiptDigest = &installRecord.ReceiptDigest
		} else if _, evidenceErr := os.Lstat(install.Manifest); os.IsNotExist(evidenceErr) {
			if legacyRecord, legacyEvidenceDigest, legacyErr := lifecycle.InspectLegacyInstalled(scope); legacyErr == nil {
				observation.Installation.State, observation.Installation.Legacy = "legacy", true
				observation.Installation.BinaryPath = stringPointer(legacyRecord.Binary)
				observation.Installation.BinaryDigest = stringPointer("sha256:" + legacyRecord.BinarySHA256)
				observation.Installation.InstallEvidenceDigest = &legacyEvidenceDigest
			} else {
				observation.Installation.State = "drifted"
			}
		} else {
			observation.Installation.State = "drifted"
		}
	} else if statErr != nil && !os.IsNotExist(statErr) {
		observation.Installation.State = "unsupported"
	}
	inspection, inspectErr := lifecycle.InspectEnrollment(scope, topsID)
	if inspectErr != nil {
		return Observation{}, inspectErr
	}
	observation.Enrollment.State, observation.Enrollment.DataPreserved = inspection.State, inspection.DataPreserved
	if inspection.State != "unenrolled" || inspection.RecordDigest != "" {
		observation.Enrollment.RecordPath = stringPointer(layout.EnrollmentFile)
		observation.Enrollment.ConfigPath = stringPointer(layout.ConfigFile)
	}
	if inspection.RecordDigest != "" {
		observation.Enrollment.RecordDigest = &inspection.RecordDigest
	}
	if inspection.ConfigDigest != "" {
		observation.Enrollment.ConfigDigest = &inspection.ConfigDigest
	}
	observation.Enrollment.UID, observation.Enrollment.GID = inspection.AuthorityUID, inspection.AuthorityGID
	if inspection.State == "enrolled" && observation.Installation.BinaryPath != nil {
		cfg, cfgErr := config.Load(layout.ConfigFile)
		if cfgErr == nil && config.ValidateLayout(cfg, layout) == nil {
			spec, specErr := supervision.SpecFromConfig(scope, topsID, *observation.Installation.BinaryPath, cfg)
			if specErr == nil {
				supervisor, supervisorErr := supervision.Observe(spec)
				if supervisorErr == nil {
					observation.Supervisor.Manager = &supervisor.Record.Manager
					observation.Supervisor.ManagerState, observation.Supervisor.DescriptorState = supervisor.ManagerState, supervisor.DescriptorState
					observation.Supervisor.DescriptorPath = &supervisor.Record.Descriptor
					if supervisor.DescriptorDigest != "" {
						observation.Supervisor.DescriptorDigest = &supervisor.DescriptorDigest
					}
					observation.Supervisor.Enablement, observation.Supervisor.ProcessState = supervisor.Enablement, supervisor.ProcessState
				}
			}
		}
	}
	if active, data, activeErr := readAttempt(layout.ActiveAttempt); activeErr == nil && active != nil {
		observation.ActiveAttemptDigest = &active.AttemptDigest
		observation.RecoveryRequired = active.Phase != "closed"
	} else if activeErr != nil {
		digest := digestBytes(data)
		observation.ActiveAttemptDigest, observation.RecoveryRequired = &digest, true
	}
	observation.StableStateDigest = digestWithout(observation, "observed_at", "stable_state_digest", "observation_digest")
	observation.ObservationDigest = digestWithout(observation, "observation_digest")
	return observation, nil
}

func currentInstall(scope stavpaths.Scope) (stavpaths.InstallLayout, lifecycle.InstallRecord, string, error) {
	executable, err := currentExecutable()
	if err != nil {
		return stavpaths.InstallLayout{}, lifecycle.InstallRecord{}, "", err
	}
	layout, record, digest, exactErr := lifecycle.VerifyExecutable(executable, scope)
	if exactErr == nil {
		return layout, record, digest, nil
	}
	resolved, resolveErr := filepath.EvalSymlinks(filepath.Clean(executable))
	immutableSuffix := filepath.Join("libexec", "symphony", "stav-append-authority", version.Version, stavpaths.BinaryName)
	if resolveErr == nil && strings.HasSuffix(resolved, string(filepath.Separator)+immutableSuffix) {
		return stavpaths.InstallLayout{}, lifecycle.InstallRecord{}, "", exactErr
	}
	layout, err = stavpaths.ResolveInstall(scope)
	if err != nil {
		return stavpaths.InstallLayout{}, lifecycle.InstallRecord{}, "", exactErr
	}
	record, digest, err = lifecycle.VerifyInstalled(scope)
	if err != nil {
		return stavpaths.InstallLayout{}, lifecycle.InstallRecord{}, "", exactErr
	}
	executableDigest, err := digestRegularFile(executable)
	if err != nil || executableDigest != "sha256:"+record.BinarySHA256 {
		return stavpaths.InstallLayout{}, lifecycle.InstallRecord{}, "", exactErr
	}
	return layout, record, digest, nil
}

func digestRegularFile(path string) (string, error) {
	resolved, err := filepath.EvalSymlinks(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(resolved)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("executable is unsafe")
	}
	return digestBytes(data), nil
}

func endpointState(path string) string {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return "absent"
	}
	if err != nil {
		return "indeterminate"
	}
	if info.Mode()&os.ModeSocket == 0 {
		return "foreign"
	}
	connection, err := net.DialTimeout("unix", path, 100*time.Millisecond)
	if err == nil {
		_ = connection.Close()
		return "ready"
	}
	if errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.ENOENT) {
		return "stale"
	}
	return "indeterminate"
}

func validToken(value *string) bool { return value != nil && tokenPattern.MatchString(*value) }
func parseTimestamp(value string) (time.Time, error) {
	return time.Parse("2006-01-02T15:04:05Z", value)
}
func formatTimestamp(value time.Time) string {
	return value.UTC().Truncate(time.Second).Format("2006-01-02T15:04:05Z")
}
func canonicalNow() string               { return formatTimestamp(time.Now()) }
func stringPointer(value string) *string { return &value }

func digestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func digestJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var canonical any
	if err := decoder.Decode(&canonical); err != nil {
		panic(err)
	}
	data, err = json.Marshal(canonical)
	if err != nil {
		panic(err)
	}
	return digestBytes(data)
}

func digestWithout(value any, keys ...string) string {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var object map[string]any
	if err := decoder.Decode(&object); err != nil {
		panic(err)
	}
	for _, key := range keys {
		delete(object, key)
	}
	return digestJSON(object)
}

func finalizeResult(result *Result) {
	result.CompletedAt = canonicalNow()
	result.ResultDigest = digestWithout(*result, "result_digest")
}

func failedResult(result Result, code string, err error, blocked bool) Result {
	result.Disposition = "failed"
	if blocked {
		result.Disposition = "blocked"
	}
	message := err.Error()
	if len(message) > 4096 {
		message = message[:4096]
	}
	result.Error = &ResultError{Code: code, Message: message}
	// A result can never obscure recovery evidence already present in its
	// observation. This also gives callers one exact CAS identity even when the
	// protected attempt document cannot be decoded and only its byte digest is
	// safe to report.
	result.RecoveryRequired = result.Observation.RecoveryRequired
	if result.RecoveryRequired {
		result.AttemptDigest = result.Observation.ActiveAttemptDigest
	}
	finalizeResult(&result)
	return result
}

func managerForPlatform() string {
	if runtime.GOOS == "darwin" {
		return "launchd"
	}
	if runtime.GOOS == "linux" {
		return "systemd"
	}
	return "external"
}

func desiredSatisfied(observation Observation, desired string) bool {
	switch desired {
	case "enrolled":
		return observation.Enrollment.State == "enrolled"
	case "unenrolled_preserved":
		return observation.Enrollment.State == "unenrolled"
	case "native_running":
		return observation.Supervisor.DescriptorState == "installed" && observation.Supervisor.ProcessState == "running"
	case "native_installed_stopped":
		return observation.Supervisor.DescriptorState == "installed" && observation.Supervisor.ProcessState == "stopped"
	case "absent_stopped":
		return observation.Supervisor.DescriptorState == "absent" && observation.Supervisor.ProcessState != "running"
	}
	return false
}

func sanitizeCode(err error) string {
	text := strings.ToLower(err.Error())
	switch {
	case strings.Contains(text, "expected") || strings.Contains(text, "differs") || strings.Contains(text, "drift"):
		return "state_conflict"
	case strings.Contains(text, "manager") || strings.Contains(text, "systemctl") || strings.Contains(text, "launchctl"):
		return "manager_unavailable"
	case strings.Contains(text, "audit"):
		return "audit_unavailable"
	case strings.Contains(text, "recovery"):
		return "recovery_required"
	default:
		return "operation_failed"
	}
}
