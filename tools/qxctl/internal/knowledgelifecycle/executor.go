package knowledgelifecycle

import (
	"encoding/json"
	"fmt"
	"runtime"
	"sort"
)

type PlannedAction struct {
	ActionID                string    `json:"action_id"`
	ComponentID             string    `json:"component_id"`
	Kind                    string    `json:"kind"`
	Direction               string    `json:"direction"`
	PrerequisiteActionIDs   []string  `json:"prerequisite_action_ids"`
	InverseActionID         *string   `json:"inverse_action_id"`
	ExpectedBeforeDigest    *string   `json:"expected_before_digest"`
	TargetStateDigest       string    `json:"target_state_digest"`
	TargetReceptorID        *string   `json:"target_receptor_id"`
	ExpectedArtifactDigests []string  `json:"expected_artifact_digests"`
	ExpectedEvidence        []string  `json:"expected_evidence"`
	Disposition             string    `json:"disposition"`
	Blockers                []Blocker `json:"blockers"`
}

type Blocker struct {
	Class       string  `json:"class"`
	ComponentID string  `json:"component_id"`
	ActionID    *string `json:"action_id"`
	Retryable   bool    `json:"retryable"`
	Detail      string  `json:"detail"`
}

type ExecutionResult struct {
	Protocol              string   `json:"protocol"`
	ActionID              string   `json:"action_id"`
	ComponentID           string   `json:"component_id"`
	Kind                  string   `json:"kind"`
	Outcome               string   `json:"outcome"`
	BlockerClass          *string  `json:"blocker_class"`
	BeforeEvidenceDigests []string `json:"before_evidence_digests"`
	AfterEvidenceDigests  []string `json:"after_evidence_digests"`
	Detail                string   `json:"detail"`
	Canonical             bool     `json:"canonical"`
	EvidenceDigest        string   `json:"evidence_digest"`
}

type DockingComponentEvidence struct {
	ComponentID      string
	ModuleID         string
	VectorID         string
	EngineID         string
	ReceiptDigest    string
	ExecutableDigest string
}

type DockingAdapter interface {
	ExecuteDocking(action PlannedAction, component DockingComponentEvidence) (outcome string, detail string, evidence []string, err error)
}

type Executor struct {
	stateRoot   string
	topsID      string
	profileID   string
	sourceRoots []string
	docking     DockingAdapter
}

func (e *Executor) SetDockingAdapter(adapter DockingAdapter) {
	e.docking = adapter
}

func NewExecutor(stateRoot, topsID, profileID string, sourceRoots []string) (*Executor, error) {
	runtimeStore, err := NewRuntimeStore(stateRoot, topsID, profileID)
	if err != nil {
		return nil, err
	}
	roots := append([]string(nil), sourceRoots...)
	sort.Strings(roots)
	for index, root := range roots {
		if !safeAbsolutePath(root) || index > 0 && root == roots[index-1] {
			return nil, fmt.Errorf("staged package roots are invalid or duplicated")
		}
	}
	return &Executor{
		stateRoot: runtimeStore.stateRoot, topsID: topsID, profileID: profileID, sourceRoots: roots,
	}, nil
}

func DecodePlannedAction(value any) (PlannedAction, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return PlannedAction{}, err
	}
	var action PlannedAction
	if err := decodeExact(data, &action); err != nil {
		return PlannedAction{}, fmt.Errorf("decode lifecycle action: %w", err)
	}
	if !safeToken(action.ActionID, 256) || !safeToken(action.ComponentID, 256) ||
		!oneOf(action.Kind, "install", "uninstall", "select", "deselect", "activate", "deactivate", "dock", "undock", "verify", "preserve", "report") ||
		!oneOf(action.Direction, "forward", "inverse", "neutral") || !taggedDigest(action.TargetStateDigest) ||
		len(action.ExpectedArtifactDigests) > 4096 || len(action.ExpectedEvidence) > 128 || len(action.Blockers) > 64 {
		return PlannedAction{}, fmt.Errorf("lifecycle action identity is invalid")
	}
	return action, nil
}

func (e *Executor) AvailableArtifactDigests() ([]string, error) {
	digests := make([]string, 0)
	seen := make(map[string]struct{})
	for _, root := range e.sourceRoots {
		candidates, err := scanReceiptRoot(root)
		if err != nil {
			return nil, fmt.Errorf("scan staged package root %s: %w", root, err)
		}
		for _, candidate := range candidates {
			if !candidate.readable {
				continue
			}
			protocol, err := receiptProtocol(candidate.data)
			if err != nil || protocol != "symphony.knowledge.install-receipt.v2" {
				continue
			}
			evidence, err := observeV2(candidate, Platform{OS: platformOS(), Architecture: platformArchitecture(), KernelABI: kernelABI()})
			if err != nil {
				continue
			}
			if _, duplicate := seen[evidence.packageState.ReceiptDigest]; !duplicate {
				seen[evidence.packageState.ReceiptDigest] = struct{}{}
				digests = append(digests, evidence.packageState.ReceiptDigest)
			}
		}
	}
	sort.Strings(digests)
	return digests, nil
}

func (e *Executor) Execute(action PlannedAction, desired DesiredState, observed Observation) ExecutionResult {
	result := ExecutionResult{
		Protocol: "symphony.knowledge.lifecycle-execution-evidence.v1",
		ActionID: action.ActionID, ComponentID: action.ComponentID, Kind: action.Kind,
		Outcome: "failed", BeforeEvidenceDigests: make([]string, 0),
		AfterEvidenceDigests: make([]string, 0), Canonical: false,
	}
	if action.ExpectedBeforeDigest != nil {
		result.BeforeEvidenceDigests = append(result.BeforeEvidenceDigests, *action.ExpectedBeforeDigest)
	}
	if err := e.execute(&result, action, desired, observed); err != nil {
		class := classifyExecutionError(err)
		result.BlockerClass = &class
		result.Detail = err.Error()
	} else if result.Detail == "" {
		result.Outcome = "committed"
		result.Detail = "exact lifecycle action completed and requires verified re-observation"
	}
	result.EvidenceDigest = ""
	value, err := objectWithout(mustJSON(result), "evidence_digest")
	if err == nil {
		result.EvidenceDigest, _ = digestValue(value)
	}
	return result
}

func (e *Executor) execute(result *ExecutionResult, action PlannedAction, desired DesiredState, observed Observation) error {
	component, wanted := findDesiredComponent(desired, action.ComponentID)
	observedComponent, present := findObservedComponent(observed, action.ComponentID)
	switch action.Kind {
	case "install":
		if !wanted || component.SelectedPackage == nil {
			return fmt.Errorf("integrity_fatal: install action lacks exact desired package identity")
		}
		if !containsString(action.ExpectedArtifactDigests, component.SelectedPackage.ReceiptDigest) {
			return fmt.Errorf("integrity_fatal: install action does not bind the desired receipt")
		}
		if present {
			if installed, ok := findObservedPackage(observedComponent, component.SelectedPackage.ReceiptDigest); ok &&
				installed.Integrity == "valid" {
				result.Outcome = "already_applied"
				result.AfterEvidenceDigests = append(result.AfterEvidenceDigests, installed.ReceiptDigest)
				result.Detail = "exact receipt-v2 package is already installed"
				return nil
			}
		}
		if !actionBeforeMatches(action, observedComponent, present) {
			return fmt.Errorf("observation_retryable: install target changed after action preparation")
		}
		source, receipt, err := e.stagedReceipt(*component.SelectedPackage)
		if err != nil {
			return err
		}
		if err := installReceiptV2(source.root, component.InstallRoot, source.relativePath, receipt); err != nil {
			return err
		}
		result.AfterEvidenceDigests = append(result.AfterEvidenceDigests, receipt.ReceiptDigest)
	case "uninstall":
		if !wanted || component.InstallRoot == "" {
			return fmt.Errorf("integrity_fatal: uninstall action lacks an exact desired install root")
		}
		if len(action.ExpectedArtifactDigests) == 0 {
			return fmt.Errorf("integrity_fatal: uninstall action lacks exact receipt evidence")
		}
		matched := false
		observedTarget := false
		for _, digest := range action.ExpectedArtifactDigests {
			identity := PackageIdentity{ReceiptProtocol: "symphony.knowledge.install-receipt.v2", ReceiptDigest: digest}
			if present {
				if installed, ok := findObservedPackage(observedComponent, digest); ok {
					observedTarget = true
					identity.PackageID = installed.PackageID
					identity.Version = installed.Version
				}
			}
			source, receipt, err := e.stagedReceipt(identity)
			if err != nil {
				return fmt.Errorf("rollback proof unavailable for %s: %w", digest, err)
			}
			if receipt.ComponentID != action.ComponentID {
				continue
			}
			matched = true
			if source.root == component.InstallRoot {
				return fmt.Errorf("integrity_fatal: rollback source must be separate from the installed package root")
			}
			if err := uninstallReceiptV2(component.InstallRoot, source.root, source.relativePath, receipt); err != nil {
				return err
			}
			result.AfterEvidenceDigests = append(result.AfterEvidenceDigests, digest)
		}
		if !matched {
			return fmt.Errorf("dependency_wait: exact staged rollback proof is unavailable")
		}
		if !observedTarget {
			result.Outcome = "already_applied"
			result.Detail = "all receipt-v2 package targets are already absent"
		}
	case "select", "deselect", "activate", "deactivate":
		if action.ComponentID == "knowledge-session-coordinator" && (action.Kind == "select" || action.Kind == "deselect") {
			return fmt.Errorf("compatibility_blocked: the active coordinator requires a separately verified handoff")
		}
		if runtimeActionAlreadyApplied(action.Kind, component, wanted, observedComponent, present) {
			result.Outcome = "already_applied"
			result.Detail = "protected lifecycle runtime state already reflects the requested action"
			return nil
		}
		if !actionBeforeMatches(action, observedComponent, present) {
			return fmt.Errorf("observation_retryable: lifecycle runtime target changed after action preparation")
		}
		if err := e.mutateRuntime(action, component, wanted, observedComponent, present); err != nil {
			return err
		}
	case "dock", "undock":
		if e.docking == nil {
			return fmt.Errorf("compatibility_blocked: exact Maestro receptor execution is unavailable")
		}
		if !present {
			return fmt.Errorf("observation_retryable: docking requires an observed component")
		}
		targetObserved := action.Kind == "undock" && observedComponent.Docking != "docked"
		if action.Kind == "dock" {
			targetObserved = wanted && component.SelectedPackage != nil &&
				observedComponent.SelectedPackageDigest != nil &&
				*observedComponent.SelectedPackageDigest == component.SelectedPackage.ReceiptDigest &&
				observedComponent.Docking == "docked" && action.TargetReceptorID != nil &&
				observedComponent.ReceptorID != nil && *observedComponent.ReceptorID == *action.TargetReceptorID
		}
		if targetObserved {
			result.Outcome = "already_applied"
			result.AfterEvidenceDigests = append(result.AfterEvidenceDigests, observedComponent.ObservationDigest)
			result.Detail = "authenticated Maestro observation already reflects the requested docking target"
			return nil
		}
		if !actionBeforeMatches(action, observedComponent, true) {
			return fmt.Errorf("observation_retryable: docking target changed after action preparation")
		}
		installRoot := ""
		receiptDigest := ""
		if action.Kind == "dock" {
			if !wanted || component.SelectedPackage == nil ||
				observedComponent.SelectedPackageDigest == nil ||
				*observedComponent.SelectedPackageDigest != component.SelectedPackage.ReceiptDigest {
				return fmt.Errorf("observation_retryable: dock requires the exact desired selected package")
			}
			installRoot = component.InstallRoot
			receiptDigest = component.SelectedPackage.ReceiptDigest
		} else {
			if observedComponent.SelectedPackageDigest == nil {
				return fmt.Errorf("critical_state_unknown: live Maestro presence lacks a selected package")
			}
			selected, ok := findObservedPackage(observedComponent, *observedComponent.SelectedPackageDigest)
			if !ok || selected.Integrity != "valid" || !selected.EntryPointsValidated {
				return fmt.Errorf("integrity_fatal: live Maestro presence lacks exact valid installed receipt evidence")
			}
			installRoot = selected.InstallRoot
			receiptDigest = selected.ReceiptDigest
		}
		if !containsString(action.ExpectedArtifactDigests, receiptDigest) {
			return fmt.Errorf("integrity_fatal: docking action does not bind the active receipt evidence")
		}
		evidence, err := e.resolveDockingComponent(action.ComponentID, installRoot, receiptDigest)
		if err != nil {
			return err
		}
		outcome, detail, digests, err := e.docking.ExecuteDocking(action, evidence)
		if err != nil {
			return err
		}
		if outcome != "committed" && outcome != "already_applied" {
			return fmt.Errorf("critical_state_unknown: Maestro returned unsupported outcome %q", outcome)
		}
		result.Outcome = outcome
		result.Detail = detail
		result.AfterEvidenceDigests = append(result.AfterEvidenceDigests, digests...)
	case "verify", "preserve", "report":
		result.Outcome = "already_applied"
		result.Detail = "non-mutating lifecycle evidence requires no host action"
	default:
		return fmt.Errorf("critical_state_unknown: unsupported lifecycle action kind")
	}
	return nil
}

func (e *Executor) resolveDockingComponent(componentID, installRoot, receiptDigest string) (DockingComponentEvidence, error) {
	if componentID == "" || installRoot == "" || receiptDigest == "" {
		return DockingComponentEvidence{}, fmt.Errorf("integrity_fatal: docking lacks exact package identity")
	}
	candidates, err := scanReceiptRoot(installRoot)
	if err != nil {
		return DockingComponentEvidence{}, err
	}
	for _, candidate := range candidates {
		var receipt receiptV2
		if !candidate.readable || decodeExact(candidate.data, &receipt) != nil ||
			validateReceiptV2(candidate, receipt) != nil ||
			receipt.ReceiptDigest != receiptDigest {
			continue
		}
		if receipt.ComponentID != componentID || receipt.ComponentKind != "vector_engine" ||
			receipt.VectorID == nil || receipt.EngineID == nil ||
			!containsString(receipt.CompatibleReceptors, "symphony.maestro.knowledge-engine.v1") {
			return DockingComponentEvidence{}, fmt.Errorf("compatibility_blocked: selected receipt does not declare the Maestro vector-engine receptor")
		}
		executableDigest := ""
		for _, entry := range receipt.EntryPoints {
			if !containsString(entry.Protocols, "symphony.knowledge.engine-process.v1") {
				continue
			}
			for _, file := range receipt.Files {
				if file.Path == entry.Path && file.Kind == "executable" {
					if executableDigest != "" && executableDigest != file.Digest {
						return DockingComponentEvidence{}, fmt.Errorf("integrity_fatal: receipt declares ambiguous Maestro process entrypoints")
					}
					executableDigest = file.Digest
				}
			}
		}
		if executableDigest == "" {
			return DockingComponentEvidence{}, fmt.Errorf("compatibility_blocked: receipt lacks a validated process entrypoint for Maestro")
		}
		return DockingComponentEvidence{
			ComponentID: receipt.ComponentID, ModuleID: receipt.ModuleID, VectorID: *receipt.VectorID,
			EngineID: *receipt.EngineID, ReceiptDigest: receipt.ReceiptDigest,
			ExecutableDigest: executableDigest,
		}, nil
	}
	return DockingComponentEvidence{}, fmt.Errorf("integrity_fatal: exact installed receipt-v2 docking evidence is unavailable")
}

func actionBeforeMatches(action PlannedAction, observed ObservedComponent, present bool) bool {
	if action.ExpectedBeforeDigest == nil {
		return !present
	}
	return present && observed.ObservationDigest == *action.ExpectedBeforeDigest
}

func (e *Executor) mutateRuntime(
	action PlannedAction,
	desired DesiredComponent,
	wanted bool,
	observed ObservedComponent,
	present bool,
) error {
	runtimeStore, err := NewRuntimeStore(e.stateRoot, e.topsID, e.profileID)
	if err != nil {
		return err
	}
	snapshot, err := runtimeStore.Snapshot()
	if err != nil {
		return err
	}
	if !runtimeSnapshotMatchesObservation(snapshot, action.ComponentID, observed, present) {
		return fmt.Errorf("observation_retryable: protected lifecycle runtime state changed after observation")
	}
	expected := "absent"
	if snapshot.Exists {
		expected = snapshot.State.RuntimeStateDigest
	}
	var selected *string
	if action.Kind == "select" {
		if !wanted || desired.SelectedPackage == nil || !present {
			return fmt.Errorf("observation_retryable: selected package is not available")
		}
		if _, ok := findObservedPackage(observed, desired.SelectedPackage.ReceiptDigest); !ok {
			return fmt.Errorf("observation_retryable: exact selected receipt is not observed")
		}
		selected = stringPointer(desired.SelectedPackage.ReceiptDigest)
	}
	_, _, err = runtimeStore.Mutate(action.ComponentID, action.Kind, selected, expected)
	return err
}

func runtimeSnapshotMatchesObservation(
	snapshot RuntimeSnapshot,
	componentID string,
	observed ObservedComponent,
	present bool,
) bool {
	var runtimeComponent *RuntimeComponent
	if snapshot.Exists {
		for index := range snapshot.State.Components {
			if snapshot.State.Components[index].ComponentID == componentID {
				runtimeComponent = &snapshot.State.Components[index]
				break
			}
		}
	}
	if runtimeComponent == nil {
		return !present || observed.SelectedPackageDigest == nil && observed.Activation == "inactive"
	}
	return present && equalStringPointer(runtimeComponent.SelectedReceiptDigest, observed.SelectedPackageDigest) &&
		runtimeComponent.Activation == observed.Activation && observed.Docking == "undocked" && observed.ReceptorID == nil
}

func (e *Executor) stagedReceipt(identity PackageIdentity) (receiptCandidate, receiptV2, error) {
	if identity.ReceiptProtocol != "symphony.knowledge.install-receipt.v2" {
		return receiptCandidate{}, receiptV2{}, fmt.Errorf("compatibility_blocked: generic package actions require an immutable v2 receipt")
	}
	for _, root := range e.sourceRoots {
		candidates, err := scanReceiptRoot(root)
		if err != nil {
			return receiptCandidate{}, receiptV2{}, err
		}
		for _, candidate := range candidates {
			var receipt receiptV2
			if !candidate.readable || decodeExact(candidate.data, &receipt) != nil || validateReceiptV2(candidate, receipt) != nil {
				continue
			}
			packageMatches := identity.PackageID == "" || receipt.PackageID == identity.PackageID
			versionMatches := identity.Version == "" || receipt.Version == identity.Version
			if packageMatches && versionMatches && receipt.ReceiptDigest == identity.ReceiptDigest {
				return candidate, receipt, nil
			}
		}
	}
	return receiptCandidate{}, receiptV2{}, fmt.Errorf("dependency_wait: no exact trusted staged v2 package matches %s", identity.ReceiptDigest)
}

func findDesiredComponent(desired DesiredState, id string) (DesiredComponent, bool) {
	for _, component := range desired.Components {
		if component.ComponentID == id {
			return component, true
		}
	}
	return DesiredComponent{}, false
}

func findObservedComponent(observation Observation, id string) (ObservedComponent, bool) {
	for _, component := range observation.Components {
		if component.ComponentID == id {
			return component, true
		}
	}
	return ObservedComponent{}, false
}

func findObservedPackage(component ObservedComponent, digest string) (ObservedPackage, bool) {
	for _, installed := range component.Packages {
		if installed.ReceiptDigest == digest {
			return installed, true
		}
	}
	return ObservedPackage{}, false
}

func runtimeActionAlreadyApplied(
	action string,
	desired DesiredComponent,
	wanted bool,
	observed ObservedComponent,
	present bool,
) bool {
	switch action {
	case "select":
		return wanted && desired.SelectedPackage != nil && present && observed.SelectedPackageDigest != nil &&
			*observed.SelectedPackageDigest == desired.SelectedPackage.ReceiptDigest
	case "deselect":
		return !present || observed.SelectedPackageDigest == nil
	case "activate":
		return present && observed.Activation == "active"
	case "deactivate":
		return !present || observed.Activation == "inactive"
	default:
		return false
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func classifyExecutionError(err error) string {
	text := err.Error()
	for _, class := range []string{
		"dependency_wait", "observation_retryable", "compatibility_blocked",
		"authorization_denied", "integrity_fatal", "critical_state_unknown", "cycle_detected",
	} {
		if len(text) > len(class) && text[:len(class)] == class {
			return class
		}
	}
	return "integrity_fatal"
}

func platformOS() string {
	if runtime.GOOS == "darwin" {
		return "macos"
	}
	return runtime.GOOS
}

func platformArchitecture() string { return runtime.GOARCH }
