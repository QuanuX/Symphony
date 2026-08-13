package foundationlifecycle

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/config"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/lifecycle"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/packageinstall"
	ssiagpaths "github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/paths"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/supervision"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/version"
)

type Engine struct {
	now       func() time.Time
	mutate    func(Plan) error
	observeFn func(string, ssiagpaths.Scope, string, *Attempt) (Observation, error)
	evidence  func(ssiagpaths.Scope) (invocationEvidence, error)
}

type invocationEvidence struct {
	Binary                string
	BinaryDigest          string
	InstallEvidenceDigest string
	ReceiptDigest         *string
	Version               string
	State                 string
	Legacy                bool
	Verified              bool
}

func New() *Engine {
	engine := &Engine{now: func() time.Time { return time.Now().UTC() }}
	engine.mutate = engine.mutatePlan
	engine.observeFn = engine.observe
	engine.evidence = engine.invocationEvidence
	return engine
}

func (engine *Engine) VerifyInvocation(scope ssiagpaths.Scope) error {
	executable, err := canonicalExecutable()
	if err != nil {
		return err
	}
	evidence, err := engine.evidence(scope)
	if err != nil {
		return err
	}
	if !evidence.Verified {
		return fmt.Errorf("foundation lifecycle installation evidence is drifted")
	}
	if filepath.Clean(evidence.Binary) == executable {
		return nil
	}
	return fmt.Errorf("foundation lifecycle adapter must be invoked from the exact installed SSIAG binary")
}

func (engine *Engine) InstalledScope() (ssiagpaths.Scope, error) {
	executable, err := canonicalExecutable()
	if err != nil {
		return "", err
	}
	if _, current, inspectErr := packageinstall.InspectExecutable(executable); inspectErr != nil {
		return "", inspectErr
	} else if current {
		return ssiagpaths.ScopeUser, nil
	}
	for _, scope := range []ssiagpaths.Scope{ssiagpaths.ScopeUser, ssiagpaths.ScopeSystem} {
		layout, layoutErr := ssiagpaths.ResolveInstall(scope)
		if layoutErr == nil && filepath.Clean(layout.Binary) == executable {
			if err := engine.VerifyInvocation(scope); err != nil {
				return "", err
			}
			return scope, nil
		}
	}
	return "", fmt.Errorf("foundation lifecycle adapter is not running from a canonical SSIAG installation")
}

func (engine *Engine) Descriptor(scope ssiagpaths.Scope) (AdapterDescriptor, error) {
	evidence, err := engine.evidence(scope)
	if err != nil {
		return AdapterDescriptor{}, err
	}
	if !evidence.Verified {
		return AdapterDescriptor{}, fmt.Errorf("foundation lifecycle installation evidence is drifted")
	}
	return engine.descriptorFor(evidence.Binary, evidence.BinaryDigest, evidence.InstallEvidenceDigest, version.Version)
}

// invocationEvidence is the single installation identity resolver used by
// description, observation, attempt binding, and mutation. Receipt-v2 wins
// only when the running executable is its exact owned entry point; otherwise
// the legacy fixed-path installation is verified for backwards compatibility.
func (engine *Engine) invocationEvidence(scope ssiagpaths.Scope) (invocationEvidence, error) {
	executable, err := canonicalExecutable()
	if err != nil {
		return invocationEvidence{}, err
	}
	if receipt, current, inspectErr := packageinstall.InspectExecutable(executable); inspectErr != nil {
		return invocationEvidence{}, inspectErr
	} else if current {
		receiptDigest := receipt.ReceiptDigest
		return invocationEvidence{Binary: receipt.Binary, BinaryDigest: receipt.BinaryDigest, InstallEvidenceDigest: receipt.ReceiptDigest, ReceiptDigest: &receiptDigest, Version: version.Version, State: "installed", Legacy: false, Verified: true}, nil
	}
	install, err := lifecycle.VerifyInstalled(scope)
	if err != nil {
		layout, layoutErr := ssiagpaths.ResolveInstall(scope)
		if layoutErr != nil || !pathExists(layout.Binary) || !pathExists(layout.InstallManifest) {
			return invocationEvidence{}, err
		}
		binaryDigest, binaryErr := digestFile(layout.Binary)
		installDigest, installErr := digestFile(layout.InstallManifest)
		if binaryErr != nil || installErr != nil {
			return invocationEvidence{}, err
		}
		return invocationEvidence{Binary: layout.Binary, BinaryDigest: binaryDigest, InstallEvidenceDigest: installDigest, Version: version.Version, State: "drifted", Legacy: true}, nil
	}
	installDigest, err := digestFile(mustInstallLayout(scope).InstallManifest)
	if err != nil {
		return invocationEvidence{}, err
	}
	return invocationEvidence{Binary: install.Binary, BinaryDigest: "sha256:" + install.BinarySHA256, InstallEvidenceDigest: installDigest, Version: version.Version, State: "legacy", Legacy: true, Verified: true}, nil
}

func (engine *Engine) descriptorFor(binaryPath, binaryDigest, installDigest, adapterVersion string) (AdapterDescriptor, error) {
	managers := []string{"external"}
	if runtime.GOOS == "darwin" {
		managers = append([]string{"launchd"}, managers...)
	}
	if runtime.GOOS == "linux" {
		managers = append([]string{"systemd"}, managers...)
	}
	descriptor := AdapterDescriptor{
		Protocol: AdapterProtocol, FormatVersion: 1, Component: "ssiag", AdapterVersion: adapterVersion,
		BinaryPath: binaryPath, BinaryDigest: binaryDigest, InstallEvidenceDigest: installDigest,
		Operations: []string{
			"engop:symphony:ssiag.enrollment.observe", "engop:symphony:ssiag.enrollment.plan", "engop:symphony:ssiag.enrollment.apply", "engop:symphony:ssiag.enrollment.apply-status", "engop:symphony:ssiag.enrollment.recover",
			"engop:symphony:ssiag.supervisor.observe", "engop:symphony:ssiag.supervisor.plan", "engop:symphony:ssiag.supervisor.apply", "engop:symphony:ssiag.supervisor.apply-status", "engop:symphony:ssiag.supervisor.recover",
		},
		SupportedScopes: []string{"user", "system"}, SupportedManagers: managers,
		Compatibility:         Compatibility{ConfigReadMajors: []uint64{1}, ConfigWriteMajor: 1, RuntimeReadMajors: []uint64{1}, RuntimeWriteMajor: 1, StateReadMajors: []uint64{1}, StateWriteMajor: 1, RollbackReadable: true},
		Limits:                Limits{RequestBytes: maxRequestBytes, ResponseBytes: maxResponseBytes, DeadlineMS: 60000, JSONDepth: maxJSONDepth, JSONValues: maxJSONValues},
		CanonicalApplyEnabled: false, NetworkListener: false,
	}
	var err error
	descriptor.DescriptorDigest, err = digestWithout(descriptor, "descriptor_digest")
	return descriptor, err
}

// DirectApply is the qxctl-free human CLI bridge. It uses the same plan,
// attempt, expected-state, mutation, verification, and recovery engine as the
// machine adapter. An interrupted direct operation recovers its exact active
// attempt before accepting new intent.
func (engine *Engine) DirectApply(surface string, scope ssiagpaths.Scope, topsID string, intent Intent) (Result, error) {
	install, err := ssiagpaths.ResolveInstall(scope)
	if err != nil {
		return Result{}, err
	}
	store, err := openAttemptStore(install.LifecycleDir, topsID, surface, false)
	if err != nil {
		return Result{}, err
	}
	attempt, present, err := store.read()
	_ = store.close()
	if err != nil {
		return Result{}, err
	}
	now := engine.stscNow()
	deadlineTime, _ := time.Parse(time.RFC3339, now)
	deadline := deadlineTime.Add(time.Minute).Format(time.RFC3339)
	if present && attempt.Phase != "closed" {
		operationID := attempt.OperationID
		result, executeErr := engine.Execute(Command{Protocol: CommandProtocol, FormatVersion: 1, Operation: "recover", Component: "ssiag", Surface: surface, Scope: string(scope), TOPSID: topsID, OperationID: &operationID, Discover: true, RequestedAt: now, DeadlineAt: deadline})
		return result, resultError(result, executeErr)
	}
	observed, err := engine.Execute(Command{Protocol: CommandProtocol, FormatVersion: 1, Operation: "observe", Component: "ssiag", Surface: surface, Scope: string(scope), TOPSID: topsID, RequestedAt: now, DeadlineAt: deadline})
	if err != nil {
		return Result{}, err
	}
	operationID := directOperationID(surface, topsID, intent, observed.Observation.StableStateDigest)
	requestID, err := randomUUID()
	if err != nil {
		return Result{}, err
	}
	correlationID, err := randomUUID()
	if err != nil {
		return Result{}, err
	}
	expected := observed.Observation.StableStateDigest
	planned, err := engine.Execute(Command{Protocol: CommandProtocol, FormatVersion: 1, Operation: "plan", Component: "ssiag", Surface: surface, Scope: string(scope), TOPSID: topsID, OperationID: &operationID, RequestID: &requestID, CorrelationID: &correlationID, ExpectedStateDigest: &expected, Intent: &intent, RequestedAt: now, DeadlineAt: deadline})
	if err != nil || planned.Plan == nil {
		return planned, resultError(planned, err)
	}
	absent := "absent"
	result, err := engine.Execute(Command{Protocol: CommandProtocol, FormatVersion: 1, Operation: "apply", Component: "ssiag", Surface: surface, Scope: string(scope), TOPSID: topsID, OperationID: &operationID, ExpectedAttemptDigest: &absent, Plan: planned.Plan, RequestedAt: now, DeadlineAt: deadline})
	return result, resultError(result, err)
}

// BindCommittedAuditReceipt is the typed handoff for the separately owned
// SSIAG/STAV reconciliation endpoint. The caller must first validate a
// committed receipt bound to this exact lifecycle attempt; this method only
// performs the protected compare-and-swap binding and never manufactures one.
func (engine *Engine) BindCommittedAuditReceipt(scope ssiagpaths.Scope, topsID, surface, expectedAttemptDigest, receiptDigest string) (Attempt, error) {
	if !validDigest(expectedAttemptDigest) || !validDigest(receiptDigest) {
		return Attempt{}, fmt.Errorf("SSIAG lifecycle audit reconciliation digest is invalid")
	}
	install, err := ssiagpaths.ResolveInstall(scope)
	if err != nil {
		return Attempt{}, err
	}
	store, err := openAttemptStore(install.LifecycleDir, topsID, surface, true)
	if err != nil {
		return Attempt{}, err
	}
	defer store.close()
	attempt, present, err := store.read()
	if err != nil || !present {
		return Attempt{}, fmt.Errorf("SSIAG lifecycle audit reconciliation attempt is absent")
	}
	if attempt.AttemptDigest != expectedAttemptDigest || attempt.Phase != "closed" || attempt.AuditState != "audit_deferred" || attempt.AuditReceiptDigest != nil {
		return Attempt{}, fmt.Errorf("SSIAG lifecycle audit reconciliation compare-and-swap mismatch")
	}
	predecessor := attempt.AttemptDigest
	attempt.PredecessorDigest = &predecessor
	attempt.AuditState = "reconciled"
	attempt.AuditReceiptDigest = &receiptDigest
	attempt.UpdatedAt = engine.stscNow()
	if err := store.write(&attempt); err != nil {
		return Attempt{}, err
	}
	return attempt, nil
}

func (engine *Engine) Execute(command Command) (Result, error) {
	deadline, err := time.Parse(time.RFC3339, command.DeadlineAt)
	if err != nil || !deadline.After(engine.now()) {
		return Result{}, fmt.Errorf("foundation lifecycle command deadline has expired")
	}
	requested, _ := time.Parse(time.RFC3339, command.RequestedAt)
	if requested.After(engine.now().Add(time.Minute)) {
		return Result{}, fmt.Errorf("foundation lifecycle command requested_at is implausibly in the future")
	}
	scope, err := ssiagpaths.ParseScope(command.Scope)
	if err != nil {
		return Result{}, err
	}
	install := mustInstallLayout(scope)
	exclusive := command.Operation == "apply" || command.Operation == "recover"
	store, err := openAttemptStore(install.LifecycleDir, command.TOPSID, command.Surface, exclusive)
	if err != nil {
		return Result{}, err
	}
	defer store.close()
	attempt, present, err := store.read()
	if err != nil {
		return Result{}, err
	}
	var current *Attempt
	if present {
		current = &attempt
	}
	observation, err := engine.observeFn(command.Surface, scope, command.TOPSID, current)
	if err != nil {
		return Result{}, err
	}
	switch command.Operation {
	case "observe":
		return engine.finish(Result{Operation: "observe", Component: "ssiag", Surface: command.Surface, Scope: command.Scope, TOPSID: command.TOPSID, Disposition: "observed", Observation: observation, AuditState: "not_applicable", ReadOnly: true})
	case "apply_status":
		result := Result{Operation: "apply_status", Component: "ssiag", Surface: command.Surface, Scope: command.Scope, TOPSID: command.TOPSID, OperationID: command.OperationID, Disposition: "observed", Observation: observation, AuditState: "not_applicable", ReadOnly: true}
		if present {
			result.AttemptDigest = stringPointer(attempt.AttemptDigest)
			result.AuditState = publicAuditState(attempt.AuditState)
			result.AuditReceiptDigest = attempt.AuditReceiptDigest
			result.StartedAt = stringPointer(attempt.StartedAt)
			result.DesiredState = stringPointer(attempt.DesiredState)
			result.ReconciliationRequired = attempt.AuditState == "audit_deferred"
		}
		return engine.finish(result)
	case "plan":
		return engine.plan(command, observation)
	case "apply":
		return engine.apply(command, observation, current, store, false)
	case "recover":
		return engine.recover(command, observation, current, store)
	default:
		return Result{}, fmt.Errorf("unsupported foundation lifecycle operation")
	}
}

func (engine *Engine) plan(command Command, observation Observation) (Result, error) {
	if command.ExpectedStateDigest == nil || *command.ExpectedStateDigest != observation.StableStateDigest {
		return engine.blocked(command, observation, nil, "state.compare-and-swap", "expected state does not match the current SSIAG lifecycle state")
	}
	created := engine.stscNow()
	createdTime, _ := time.Parse(time.RFC3339, created)
	plan := Plan{
		Protocol: PlanProtocol, FormatVersion: 1, Component: "ssiag", Surface: command.Surface, Scope: command.Scope, TOPSID: command.TOPSID,
		OperationID: *command.OperationID, RequestID: *command.RequestID, CorrelationID: *command.CorrelationID, ExpectedStateDigest: observation.StableStateDigest,
		DesiredState: command.Intent.DesiredState, TOPSName: command.Intent.TOPSName, ServiceUID: command.Intent.ServiceUID, ServiceGID: command.Intent.ServiceGID,
		AuthorityUID: nil, AuthorityGID: nil, AuditMode: command.Intent.AuditMode, CreatedAt: created, ExpiresAt: createdTime.Add(time.Duration(command.Intent.TTLSeconds) * time.Second).Format(time.RFC3339),
	}
	var err error
	plan.PlanDigest, err = digestWithout(plan, "plan_digest")
	if err != nil {
		return Result{}, err
	}
	return engine.finish(Result{Operation: "plan", Component: "ssiag", Surface: command.Surface, Scope: command.Scope, TOPSID: command.TOPSID, OperationID: command.OperationID, Disposition: "planned", DesiredState: stringPointer(plan.DesiredState), Observation: observation, Plan: &plan, AuditState: "not_applicable", ReadOnly: true})
}

func (engine *Engine) apply(command Command, observation Observation, current *Attempt, store *attemptStore, recovered bool) (Result, error) {
	plan := *command.Plan
	if plan.Component != "ssiag" || plan.Surface != command.Surface || plan.Scope != command.Scope || plan.TOPSID != command.TOPSID || plan.OperationID != *command.OperationID {
		return engine.blocked(command, observation, &plan, "plan.binding-mismatch", "plan does not match the selected SSIAG lifecycle target")
	}
	expires, _ := time.Parse(time.RFC3339, plan.ExpiresAt)
	if !engine.now().Before(expires) {
		return engine.blocked(command, observation, &plan, "plan.expired", "SSIAG lifecycle plan has expired")
	}
	if current != nil {
		if current.OperationID == plan.OperationID && current.PlanDigest != plan.PlanDigest {
			return engine.blocked(command, observation, &plan, "operation.reuse-conflict", "operation identity was already bound to different SSIAG lifecycle evidence")
		}
		if current.OperationID == plan.OperationID && current.PlanDigest == plan.PlanDigest && current.Phase == "closed" {
			return engine.appliedResult(command, observation, &plan, current, false, true, recovered, "already_applied")
		}
		if current.Phase == "closed" && current.AuditState == "audit_deferred" {
			result, err := engine.blocked(command, observation, &plan, "audit.reconciliation-required", "the previous SSIAG lifecycle outcome must be reconciled before this surface accepts another mutation")
			if err != nil {
				return Result{}, err
			}
			result.AttemptDigest = stringPointer(current.AttemptDigest)
			result.AuditState = "audit_deferred"
			result.ReconciliationRequired = true
			result.StartedAt = stringPointer(current.StartedAt)
			return engine.finish(result)
		}
		if current.Phase != "closed" {
			if command.ExpectedAttemptDigest == nil || *command.ExpectedAttemptDigest != current.AttemptDigest || current.OperationID != plan.OperationID || current.PlanDigest != plan.PlanDigest {
				return engine.blocked(command, observation, &plan, "attempt.compare-and-swap", "an exact SSIAG lifecycle attempt must be recovered before another mutation")
			}
			return engine.resume(command, observation, &plan, current, store, recovered)
		}
	}
	if command.ExpectedAttemptDigest == nil || *command.ExpectedAttemptDigest != "absent" {
		return engine.blocked(command, observation, &plan, "attempt.compare-and-swap", "new SSIAG lifecycle apply requires expected attempt state absent")
	}
	if observation.StableStateDigest != plan.ExpectedStateDigest {
		if engine.desiredProven(plan, observation) {
			return engine.appliedResult(command, observation, &plan, nil, false, true, recovered, "already_applied")
		}
		return engine.blocked(command, observation, &plan, "state.compare-and-swap", "SSIAG lifecycle state changed after plan creation")
	}
	if engine.desiredProven(plan, observation) {
		return engine.appliedResult(command, observation, &plan, nil, false, true, recovered, "already_applied")
	}
	if plan.AuditMode == "ordinary" {
		return engine.blocked(command, observation, &plan, "audit.unavailable", "ordinary SSIAG foundational lifecycle mutation requires a committed STAV receipt before external mutation")
	}
	if plan.Surface == "supervisor" && observation.Supervisor.ManagerState == "manager_unavailable" {
		return engine.blocked(command, observation, &plan, "manager.unavailable", "native SSIAG supervisor manager is unavailable")
	}
	installation := observation.Installation
	if installation.BinaryDigest == nil || installation.InstallEvidenceDigest == nil || installation.State != "legacy" && installation.State != "installed" {
		return engine.blocked(command, observation, &plan, "installation.unavailable", "exact installed SSIAG evidence is required")
	}
	now := engine.stscNow()
	attempt := &Attempt{
		Protocol: AttemptProtocol, FormatVersion: 1, Component: "ssiag", Surface: command.Surface, Scope: command.Scope, TOPSID: command.TOPSID,
		OperationID: plan.OperationID, RequestID: plan.RequestID, CorrelationID: plan.CorrelationID, Phase: "prepared", PlanDigest: plan.PlanDigest,
		PriorStateDigest: plan.ExpectedStateDigest, DesiredState: plan.DesiredState, BinaryDigest: *installation.BinaryDigest, InstallEvidenceDigest: *installation.InstallEvidenceDigest,
		AuditState: auditAttemptState(plan.AuditMode), StartedAt: now, UpdatedAt: now,
	}
	if err := store.writePlan(plan); err != nil {
		return Result{}, err
	}
	if err := store.write(attempt); err != nil {
		return Result{}, err
	}
	return engine.resume(command, observation, &plan, attempt, store, recovered)
}

func (engine *Engine) resume(command Command, prior Observation, plan *Plan, attempt *Attempt, store *attemptStore, recovered bool) (Result, error) {
	if engine.desiredProven(*plan, prior) {
		return engine.closeAttempt(command, prior, plan, attempt, store, false, true, recovered)
	}
	predecessor, predecessorErr := engine.observeFn(command.Surface, parseScope(command.Scope), command.TOPSID, nil)
	if predecessorErr != nil {
		return Result{}, predecessorErr
	}
	if predecessor.StableStateDigest != attempt.PriorStateDigest {
		return engine.markRecoveryRequired(command, prior, plan, attempt, store, "state.partial-or-divergent", "current SSIAG lifecycle state proves neither predecessor nor desired state")
	}
	if err := engine.advance(store, attempt, "mutating"); err != nil {
		return Result{}, err
	}
	if err := engine.mutate(*plan); err != nil {
		post, observeErr := engine.observeFn(command.Surface, parseScope(command.Scope), command.TOPSID, attempt)
		if observeErr == nil && engine.desiredProven(*plan, post) {
			return engine.closeAttempt(command, post, plan, attempt, store, true, false, recovered)
		}
		if observeErr != nil {
			return Result{}, errors.Join(err, observeErr)
		}
		return engine.markRecoveryRequired(command, post, plan, attempt, store, "mutation.interrupted", err.Error())
	}
	if err := engine.advance(store, attempt, "observing"); err != nil {
		return Result{}, err
	}
	post, err := engine.observeFn(command.Surface, parseScope(command.Scope), command.TOPSID, attempt)
	if err != nil {
		return Result{}, err
	}
	if !engine.desiredProven(*plan, post) {
		return engine.markRecoveryRequired(command, post, plan, attempt, store, "mutation.unverified", "native mutation completed without proving the requested SSIAG lifecycle state")
	}
	return engine.closeAttempt(command, post, plan, attempt, store, true, false, recovered)
}

func (engine *Engine) recover(command Command, observation Observation, current *Attempt, store *attemptStore) (Result, error) {
	if current == nil {
		return engine.blocked(command, observation, nil, "recovery.absent", "SSIAG lifecycle recovery evidence is absent")
	}
	if !command.Discover && (command.ExpectedAttemptDigest == nil || *command.ExpectedAttemptDigest != current.AttemptDigest) {
		return engine.blocked(command, observation, nil, "attempt.compare-and-swap", "SSIAG lifecycle recovery attempt digest does not match")
	}
	if current.OperationID != *command.OperationID {
		return engine.blocked(command, observation, nil, "operation.binding-mismatch", "recovery operation does not own the current SSIAG lifecycle attempt")
	}
	plan, err := store.readPlan()
	if err != nil || plan.PlanDigest != current.PlanDigest || plan.OperationID != current.OperationID {
		return engine.blocked(command, observation, nil, "recovery.plan-unavailable", "exact SSIAG lifecycle recovery plan is missing or invalid")
	}
	return engine.resume(command, observation, &plan, current, store, true)
}

func (engine *Engine) mutatePlan(plan Plan) error {
	scope := parseScope(plan.Scope)
	evidence, err := engine.evidence(scope)
	if err != nil {
		return err
	}
	if !evidence.Verified {
		return fmt.Errorf("SSIAG installation evidence is drifted")
	}
	switch plan.Surface {
	case "enrollment":
		if plan.DesiredState == "enrolled" {
			if evidence.Legacy {
				_, err = lifecycle.Enroll(scope, plan.TOPSID, dereference(plan.TOPSName), plan.ServiceUID, plan.ServiceGID)
			} else {
				_, err = lifecycle.EnrollVerifiedPackage(scope, plan.TOPSID, dereference(plan.TOPSName), plan.ServiceUID, plan.ServiceGID, evidence.Binary, evidence.BinaryDigest)
			}
			return err
		}
		_, err = lifecycle.Unenroll(scope, plan.TOPSID, false)
		return err
	case "supervisor":
		layout, err := ssiagpaths.ResolveInstance(scope, plan.TOPSID)
		if err != nil {
			return err
		}
		cfg, err := config.LoadTrusted(layout.ConfigFile, scope)
		if err != nil {
			return err
		}
		spec, err := supervision.SpecFromConfig(scope, plan.TOPSID, evidence.Binary, cfg)
		if err != nil {
			return err
		}
		status, err := supervision.ObserveOffline(scope, plan.TOPSID, &spec)
		if err != nil {
			return err
		}
		if !status.ManagerAvailable {
			return fmt.Errorf("%w: %s", supervision.ErrManagerUnavailable, status.Manager)
		}
		if plan.DesiredState == "absent_stopped" {
			_, err := supervision.Uninstall(spec, false, true)
			return err
		}
		record, err := supervision.Install(spec, false)
		if err != nil {
			return err
		}
		if plan.DesiredState == "native_running" {
			return supervision.Start(record)
		}
		return supervision.Stop(record)
	}
	return fmt.Errorf("unsupported SSIAG lifecycle surface")
}

func (engine *Engine) observe(surface string, scope ssiagpaths.Scope, topsID string, attempt *Attempt) (Observation, error) {
	evidence, err := engine.evidence(scope)
	if err != nil {
		return Observation{}, err
	}
	instance, err := ssiagpaths.ResolveInstance(scope, topsID)
	if err != nil {
		return Observation{}, err
	}
	observation := Observation{
		Protocol: ObservationProtocol, FormatVersion: 1, Component: "ssiag", Surface: surface, Scope: string(scope), TOPSID: topsID,
		Installation: InstallationObservation{State: evidence.State, BinaryPath: stringPointer(evidence.Binary), BinaryDigest: stringPointer(evidence.BinaryDigest), InstallEvidenceDigest: stringPointer(evidence.InstallEvidenceDigest), ReceiptDigest: evidence.ReceiptDigest, Legacy: evidence.Legacy},
		Enrollment:   EnrollmentObservation{State: "unenrolled", DataPreserved: pathExists(instance.ConfigDir) || pathExists(instance.StateDir)},
		Supervisor:   SupervisorObservation{ManagerState: "unknown", DescriptorState: "absent", Enablement: "unknown", ProcessState: "unknown", EndpointState: endpointState(instance.Socket)},
		ObservedAt:   engine.stscNow(),
	}
	if record, present, inspectErr := lifecycle.InspectEnrollment(scope, topsID); inspectErr == nil && present {
		observation.Enrollment.State = "enrolled"
		observation.Enrollment.RecordPath = stringPointer(instance.EnrollmentManifest)
		observation.Enrollment.ConfigPath = stringPointer(record.ConfigFile)
		if digest, digestErr := digestFile(instance.EnrollmentManifest); digestErr == nil {
			observation.Enrollment.RecordDigest = &digest
		}
		if digest, digestErr := digestFile(record.ConfigFile); digestErr == nil {
			observation.Enrollment.ConfigDigest = &digest
		}
		if cfg, cfgErr := config.LoadTrusted(record.ConfigFile, scope); cfgErr == nil && cfg.Authentication != nil && cfg.Authentication.Service != nil {
			observation.Enrollment.UID, observation.Enrollment.GID = cfg.Authentication.Service.UID, cfg.Authentication.Service.GID
		} else if cfgErr != nil {
			observation.Enrollment.State = "drifted"
		}
	} else if inspectErr != nil {
		observation.Enrollment.State = "unsafe"
	}
	var spec *supervision.Spec
	if observation.Enrollment.State == "enrolled" && observation.Installation.BinaryPath != nil {
		if cfg, cfgErr := config.LoadTrusted(instance.ConfigFile, scope); cfgErr == nil {
			if value, specErr := supervision.SpecFromConfig(scope, topsID, *observation.Installation.BinaryPath, cfg); specErr == nil {
				spec = &value
			}
		}
	}
	offline, err := supervision.ObserveOffline(scope, topsID, spec)
	if err != nil {
		return Observation{}, err
	}
	observation.Supervisor.Manager = stringPointer(offline.Manager)
	if offline.ManagerAvailable {
		observation.Supervisor.ManagerState = "available"
	} else {
		observation.Supervisor.ManagerState = "manager_unavailable"
	}
	observation.Supervisor.DescriptorPath = stringPointer(offline.Descriptor)
	switch offline.DescriptorState {
	case "absent":
		observation.Supervisor.DescriptorState = "absent"
	case "matching", "present":
		observation.Supervisor.DescriptorState = "installed"
	case "drifted":
		observation.Supervisor.DescriptorState = "drifted"
	default:
		observation.Supervisor.DescriptorState = "unsafe"
	}
	if offline.DescriptorHash != "" {
		digest := "sha256:" + offline.DescriptorHash
		observation.Supervisor.DescriptorDigest = &digest
	}
	if observation.Supervisor.EndpointState == "ready" {
		observation.Supervisor.ProcessState = "running"
	} else if observation.Supervisor.EndpointState == "absent" || observation.Supervisor.EndpointState == "stale" {
		observation.Supervisor.ProcessState = "stopped"
	}
	if observation.Supervisor.DescriptorState == "absent" {
		observation.Supervisor.Enablement = "not_applicable"
	} else {
		observation.Supervisor.Enablement = "unknown"
	}
	if attempt != nil && attempt.Phase != "closed" {
		observation.RecoveryRequired = attempt.Phase == "recovery_required"
		observation.ActiveAttemptDigest = stringPointer(attempt.AttemptDigest)
	}
	observation.StableStateDigest, err = digestWithoutFields(observation, "observed_at", "stable_state_digest", "observation_digest")
	if err != nil {
		return Observation{}, err
	}
	observation.ObservationDigest, err = digestWithout(observation, "observation_digest")
	return observation, err
}

func (engine *Engine) desiredProven(plan Plan, observation Observation) bool {
	switch plan.DesiredState {
	case "enrolled":
		if observation.Enrollment.State != "enrolled" {
			return false
		}
		layout, err := ssiagpaths.ResolveInstance(parseScope(plan.Scope), plan.TOPSID)
		if err != nil {
			return false
		}
		cfg, err := config.LoadTrusted(layout.ConfigFile, parseScope(plan.Scope))
		if err != nil || plan.TOPSName == nil || cfg.TOPS.Name != *plan.TOPSName {
			return false
		}
		if plan.ServiceUID != nil && (cfg.Authentication == nil || cfg.Authentication.Service == nil || cfg.Authentication.Service.UID == nil || *cfg.Authentication.Service.UID != *plan.ServiceUID || *cfg.Authentication.Service.GID != *plan.ServiceGID) {
			return false
		}
		return true
	case "unenrolled_preserved":
		return observation.Enrollment.State == "unenrolled" && observation.Enrollment.DataPreserved
	case "native_running":
		return observation.Supervisor.DescriptorState == "installed" && observation.Supervisor.EndpointState == "ready"
	case "native_installed_stopped":
		return observation.Supervisor.DescriptorState == "installed" && observation.Supervisor.ProcessState == "stopped"
	case "absent_stopped":
		return observation.Supervisor.DescriptorState == "absent" && observation.Supervisor.ProcessState == "stopped"
	default:
		return false
	}
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

func (engine *Engine) closeAttempt(command Command, observation Observation, plan *Plan, attempt *Attempt, store *attemptStore, changed, replayed, recovered bool) (Result, error) {
	if err := engine.advance(store, attempt, "mutation_closed"); err != nil {
		return Result{}, err
	}
	completed := engine.stscNow()
	attempt.PredecessorDigest = stringPointer(attempt.AttemptDigest)
	attempt.Phase = "closed"
	attempt.UpdatedAt = completed
	attempt.CompletedAt = &completed
	if err := store.write(attempt); err != nil {
		return Result{}, err
	}
	observation, err := engine.observeFn(command.Surface, parseScope(command.Scope), command.TOPSID, attempt)
	if err != nil {
		return Result{}, err
	}
	return engine.appliedResult(command, observation, plan, attempt, changed, replayed, recovered, map[bool]string{true: "recovered", false: "applied"}[recovered])
}

func (engine *Engine) markRecoveryRequired(command Command, observation Observation, plan *Plan, attempt *Attempt, store *attemptStore, code, message string) (Result, error) {
	if err := engine.advance(store, attempt, "recovery_required"); err != nil {
		return Result{}, err
	}
	observation, observeErr := engine.observeFn(command.Surface, parseScope(command.Scope), command.TOPSID, attempt)
	if observeErr != nil {
		return Result{}, observeErr
	}
	result, err := engine.finish(Result{Operation: command.Operation, Component: "ssiag", Surface: command.Surface, Scope: command.Scope, TOPSID: command.TOPSID, OperationID: command.OperationID, Disposition: "failed", DesiredState: stringPointer(plan.DesiredState), Observation: observation, Plan: plan, RecoveryRequired: true, ReconciliationRequired: plan.AuditMode == "audit_deferred", AttemptDigest: stringPointer(attempt.AttemptDigest), AuditState: publicAuditState(attempt.AuditState), StartedAt: stringPointer(attempt.StartedAt), Error: &ErrorResult{Code: code, Message: message}})
	return result, err
}

func (engine *Engine) advance(store *attemptStore, attempt *Attempt, phase string) error {
	predecessor := attempt.AttemptDigest
	attempt.PredecessorDigest = &predecessor
	attempt.Phase = phase
	attempt.UpdatedAt = engine.stscNow()
	return store.write(attempt)
}

func (engine *Engine) appliedResult(command Command, observation Observation, plan *Plan, attempt *Attempt, changed, replayed, recovered bool, disposition string) (Result, error) {
	result := Result{Operation: command.Operation, Component: "ssiag", Surface: command.Surface, Scope: command.Scope, TOPSID: command.TOPSID, OperationID: command.OperationID, Disposition: disposition, DesiredState: stringPointer(plan.DesiredState), Observation: observation, Plan: plan, Changed: changed, Replayed: replayed, Recovered: recovered, AuditState: "not_applicable"}
	if attempt != nil {
		result.AttemptDigest = stringPointer(attempt.AttemptDigest)
		result.AuditState = publicAuditState(attempt.AuditState)
		result.AuditReceiptDigest = attempt.AuditReceiptDigest
		result.StartedAt = stringPointer(attempt.StartedAt)
		result.ReconciliationRequired = attempt.AuditState == "audit_deferred"
	}
	return engine.finish(result)
}

func (engine *Engine) blocked(command Command, observation Observation, plan *Plan, code, message string) (Result, error) {
	return engine.finish(Result{Operation: command.Operation, Component: "ssiag", Surface: command.Surface, Scope: command.Scope, TOPSID: command.TOPSID, OperationID: command.OperationID, Disposition: "blocked", Observation: observation, Plan: plan, RecoveryRequired: observation.RecoveryRequired, AttemptDigest: observation.ActiveAttemptDigest, AuditState: "not_applicable", Error: &ErrorResult{Code: code, Message: message}, ReadOnly: command.Operation != "apply" && command.Operation != "recover"})
}

func (engine *Engine) finish(result Result) (Result, error) {
	result.Protocol = ResultProtocol
	result.FormatVersion = 1
	result.Canonical = false
	result.CompletedAt = engine.stscNow()
	var err error
	result.ResultDigest, err = digestWithout(result, "result_digest")
	return result, err
}

func (engine *Engine) stscNow() string {
	return engine.now().UTC().Truncate(time.Second).Format(time.RFC3339)
}
func parseScope(value string) ssiagpaths.Scope {
	scope, _ := ssiagpaths.ParseScope(value)
	return scope
}
func stringPointer(value string) *string { copy := value; return &copy }
func dereference(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
func pathExists(path string) bool { _, err := os.Lstat(path); return err == nil }
func publicAuditState(value string) string {
	if value == "pending" {
		return "not_applicable"
	}
	return value
}
func auditAttemptState(mode string) string {
	return "audit_deferred"
}
func auditPlanMode(state string) string {
	if state == "audit_deferred" {
		return "audit_deferred"
	}
	return "ordinary"
}
func mustInstallLayout(scope ssiagpaths.Scope) ssiagpaths.InstallLayout {
	layout, _ := ssiagpaths.ResolveInstall(scope)
	return layout
}

func directOperationID(surface, topsID string, intent Intent, stateDigest string) string {
	basis, _ := digestValue(struct {
		Surface string `json:"surface"`
		TOPSID  string `json:"tops_id"`
		Intent  Intent `json:"intent"`
		State   string `json:"state"`
	}{surface, topsID, intent, stateDigest})
	sum := sha256.Sum256([]byte(basis))
	return "ssiag-direct:" + hex.EncodeToString(sum[:16])
}

func randomUUID() (string, error) {
	var data [16]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", err
	}
	data[6] = data[6]&0x0f | 0x40
	data[8] = data[8]&0x3f | 0x80
	value := hex.EncodeToString(data[:])
	return value[0:8] + "-" + value[8:12] + "-" + value[12:16] + "-" + value[16:20] + "-" + value[20:32], nil
}

func resultError(result Result, err error) error {
	if err != nil {
		return err
	}
	if result.Error != nil {
		return fmt.Errorf("%s: %s", result.Error.Code, result.Error.Message)
	}
	return nil
}

// AssertNativePurgeSafe binds native-only purge to the same protected attempt
// state. Purge remains intentionally absent from the machine/qxctl v1 intent.
func (engine *Engine) AssertNativePurgeSafe(scope ssiagpaths.Scope, topsID string) error {
	install, err := ssiagpaths.ResolveInstall(scope)
	if err != nil {
		return err
	}
	store, err := openAttemptStore(install.LifecycleDir, topsID, "enrollment", false)
	if err != nil {
		return err
	}
	defer store.close()
	attempt, present, err := store.read()
	if err != nil {
		return err
	}
	if present && attempt.Phase != "closed" {
		return fmt.Errorf("refusing SSIAG purge while lifecycle recovery is required")
	}
	status, err := supervision.ObserveOffline(scope, topsID, nil)
	if err != nil {
		return err
	}
	if status.DescriptorState != "absent" {
		return fmt.Errorf("refusing SSIAG purge while supervisor descriptor is present")
	}
	return nil
}

var _ = filepath.Clean
var _ = strings.TrimSpace
