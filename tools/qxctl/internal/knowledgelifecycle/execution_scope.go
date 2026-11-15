package knowledgelifecycle

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
)

const ExecutionScopeProtocol = "symphony.knowledge.lifecycle-execution-scope.v1"
const maxExecutionScopeBytes = 4 << 20

type ScopeEffect struct {
	ComponentID             string   `json:"component_id"`
	Kind                    string   `json:"kind"`
	TargetStateDigest       string   `json:"target_state_digest"`
	TargetReceptorID        *string  `json:"target_receptor_id"`
	ExpectedArtifactDigests []string `json:"expected_artifact_digests"`
	InstallRoot             string   `json:"install_root"`
}

type ScopeOwnershipEffect struct {
	InstallRoot   string `json:"install_root"`
	ComponentID   string `json:"component_id"`
	ReceiptDigest string `json:"receipt_digest"`
	Disposition   string `json:"disposition"`
}

type ExecutionScope struct {
	Protocol                     string                 `json:"protocol"`
	FormatVersion                uint64                 `json:"format_version"`
	ScopeID                      string                 `json:"scope_id"`
	TOPSID                       string                 `json:"tops_id"`
	ProfileID                    string                 `json:"profile_id"`
	ProfileDigest                string                 `json:"profile_digest"`
	DesiredStateDigest           string                 `json:"desired_state_digest"`
	SourceReportJournalDigest    string                 `json:"source_report_journal_digest"`
	InitialPlanDigest            string                 `json:"initial_plan_digest"`
	InitialStableInventoryDigest string                 `json:"initial_stable_inventory_digest"`
	ExpectedApplyJournalDigest   string                 `json:"expected_apply_journal_digest"`
	ExpectedAppliedStateDigest   string                 `json:"expected_applied_state_digest"`
	Components                   []string               `json:"components"`
	Effects                      []ScopeEffect          `json:"effects"`
	OwnershipEffects             []ScopeOwnershipEffect `json:"ownership_effects"`
	Canonical                    bool                   `json:"canonical"`
	ScopeDigest                  string                 `json:"scope_digest"`
}

// scopeJSON follows the native JSON serialization: sorted object keys, UTF-8
// strings and no HTML escaping. Go's U+2028/U+2029 JavaScript escapes are decoded
// only when they are actual JSON escapes, never when the backslash is literal.
func scopeJSON(value any) ([]byte, error) {
	initial, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var normalized any
	decoder := json.NewDecoder(bytes.NewReader(initial))
	decoder.UseNumber()
	if err := decoder.Decode(&normalized); err != nil {
		return nil, err
	}
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(normalized); err != nil {
		return nil, err
	}
	raw := bytes.TrimSuffix(buffer.Bytes(), []byte("\n"))
	out := make([]byte, 0, len(raw))
	for i := 0; i < len(raw); i++ {
		if raw[i] == '\\' && i+1 < len(raw) {
			if i+6 <= len(raw) && (string(raw[i:i+6]) == "\\u2028" || string(raw[i:i+6]) == "\\u2029") {
				if raw[i+5] == '8' {
					out = append(out, []byte("\u2028")...)
				} else {
					out = append(out, []byte("\u2029")...)
				}
				i += 5
				continue
			}
			out = append(out, raw[i], raw[i+1])
			i++
			continue
		}
		out = append(out, raw[i])
	}
	return out, nil
}

// CanonicalScopeJSON encodes values with the native coordinator canonical JSON rules.
func CanonicalScopeJSON(value any) ([]byte, error) { return scopeJSON(value) }

func scopeValueDigest(value any) (string, error) {
	data, err := scopeJSON(value)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func ExecutionScopeDigest(scope ExecutionScope) (string, error) {
	value, err := objectWithout(mustJSON(scope), "scope_digest")
	if err != nil {
		return "", err
	}
	return scopeValueDigest(value)
}

func DecodeExecutionScope(data []byte) (ExecutionScope, error) {
	if err := knowledgeengine.ValidateJSONObject(data, maxExecutionScopeBytes); err != nil {
		return ExecutionScope{}, err
	}
	var scope ExecutionScope
	if err := decodeExact(data, &scope); err != nil {
		return scope, err
	}
	// Struct decoding alone cannot distinguish missing fields from zero values.
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return scope, err
	}
	if len(fields) != 17 || !bytes.Equal(bytes.TrimSpace(fields["canonical"]), []byte("false")) {
		return scope, fmt.Errorf("execution scope field set is invalid")
	}
	for field, count := range map[string]int{"effects": 6, "ownership_effects": 4} {
		var entries []map[string]json.RawMessage
		if err := json.Unmarshal(fields[field], &entries); err != nil {
			return scope, err
		}
		for _, entry := range entries {
			if len(entry) != count {
				return scope, fmt.Errorf("execution scope nested field set is invalid")
			}
		}
	}
	if err := scope.Validate(); err != nil {
		return scope, err
	}
	return scope, nil
}

func (s ExecutionScope) Validate() error {
	if s.Protocol != ExecutionScopeProtocol || s.FormatVersion != 1 || s.Canonical || !safeToken(s.ScopeID, 256) || !safeToken(s.TOPSID, 256) || !safeToken(s.ProfileID, 256) {
		return fmt.Errorf("execution scope identity is invalid")
	}
	for _, value := range []string{s.ProfileDigest, s.DesiredStateDigest, s.SourceReportJournalDigest, s.InitialPlanDigest, s.InitialStableInventoryDigest, s.ScopeDigest} {
		if !taggedDigest(value) {
			return fmt.Errorf("execution scope digest is invalid")
		}
	}
	for _, value := range []string{s.ExpectedApplyJournalDigest, s.ExpectedAppliedStateDigest} {
		if value != "absent" && !taggedDigest(value) {
			return fmt.Errorf("execution scope expected state is invalid")
		}
	}
	if len(s.Components) == 0 || len(s.Components) > 4096 || len(s.Effects) == 0 || len(s.Effects) > 4096 || s.OwnershipEffects == nil || len(s.OwnershipEffects) > 4096 {
		return fmt.Errorf("execution scope collections are invalid")
	}
	for i, id := range s.Components {
		if !safeToken(id, 256) || i > 0 && id <= s.Components[i-1] {
			return fmt.Errorf("execution scope components must be sorted and unique")
		}
	}
	roots := map[string]bool{}
	previous := ""
	packageKinds := map[string]string{}
	effectKinds := map[string]map[string]bool{}
	for _, effect := range s.Effects {
		if !s.Contains(effect.ComponentID) || !oneOf(effect.Kind, "install", "uninstall", "select", "deselect", "activate", "deactivate", "dock", "undock") || !taggedDigest(effect.TargetStateDigest) || !safeAbsolutePath(effect.InstallRoot) || effect.InstallRoot == "/" || effect.ExpectedArtifactDigests == nil || len(effect.ExpectedArtifactDigests) > 4096 {
			return fmt.Errorf("execution scope effect is invalid")
		}
		if effectKinds[effect.ComponentID] == nil {
			effectKinds[effect.ComponentID] = map[string]bool{}
		}
		if effectKinds[effect.ComponentID][effect.Kind] {
			return fmt.Errorf("scope duplicates a component effect kind")
		}
		effectKinds[effect.ComponentID][effect.Kind] = true
		if effectKinds[effect.ComponentID]["select"] && effectKinds[effect.ComponentID]["deselect"] {
			return fmt.Errorf("scope cannot both select and deselect one component")
		}
		if effect.Kind == "install" || effect.Kind == "uninstall" {
			if len(effect.ExpectedArtifactDigests) == 0 {
				return fmt.Errorf("package scope effect requires exact receipt digests")
			}
			if previousKind, ok := packageKinds[effect.ComponentID]; ok && previousKind != effect.Kind {
				return fmt.Errorf("scope cannot both install and uninstall one component")
			}
			packageKinds[effect.ComponentID] = effect.Kind
		}
		if (effect.Kind == "dock") != (effect.TargetReceptorID != nil) || effect.TargetReceptorID != nil && !safeToken(*effect.TargetReceptorID, 256) {
			return fmt.Errorf("execution scope receptor is invalid")
		}
		for i, digest := range effect.ExpectedArtifactDigests {
			if !taggedDigest(digest) || i > 0 && digest <= effect.ExpectedArtifactDigests[i-1] {
				return fmt.Errorf("execution scope artifacts must be sorted and unique")
			}
		}
		encoded, _ := scopeJSON(effect)
		key := string(encoded)
		if key <= previous {
			return fmt.Errorf("execution scope effects must be sorted and unique")
		}
		previous = key
		roots[effect.InstallRoot] = true
	}
	previous = ""
	for _, effect := range s.OwnershipEffects {
		if !s.Contains(effect.ComponentID) || !safeAbsolutePath(effect.InstallRoot) || effect.InstallRoot == "/" || !taggedDigest(effect.ReceiptDigest) || !oneOf(effect.Disposition, "retained", "retiring", "release") {
			return fmt.Errorf("execution scope ownership effect is invalid")
		}
		encoded, _ := scopeJSON(effect)
		key := string(encoded)
		if key <= previous {
			return fmt.Errorf("execution scope ownership effects must be sorted and unique")
		}
		previous = key
		roots[effect.InstallRoot] = true
	}
	if len(roots) > 64 {
		return fmt.Errorf("execution scope exceeds installation root bound")
	}
	digest, err := ExecutionScopeDigest(s)
	if err != nil || digest != s.ScopeDigest {
		return fmt.Errorf("execution scope digest mismatch")
	}
	return nil
}

func (s ExecutionScope) Contains(component string) bool {
	i := sort.SearchStrings(s.Components, component)
	return i < len(s.Components) && s.Components[i] == component
}

func (s ExecutionScope) ValidateDesired(profileDigest string, desired DesiredState, sourceDigest string) error {
	if err := s.Validate(); err != nil {
		return err
	}
	if s.ProfileID != desired.ProfileID || s.TOPSID != desired.TOPSID || s.ProfileDigest != profileDigest || s.DesiredStateDigest != desired.DesiredStateDigest || s.SourceReportJournalDigest != sourceDigest {
		return fmt.Errorf("execution scope does not bind the complete desired profile and source journal")
	}
	for _, id := range s.Components {
		if _, ok := findDesiredComponent(desired, id); !ok {
			return fmt.Errorf("execution scope component %s is absent from complete desired state", id)
		}
	}
	for _, effect := range s.Effects {
		component, ok := findDesiredComponent(desired, effect.ComponentID)
		if !ok || component.InstallRoot != effect.InstallRoot {
			return fmt.Errorf("execution scope effect root differs from desired state")
		}
		if effect.Kind == "install" && (component.Presence != "present" || component.SelectedPackage == nil || len(effect.ExpectedArtifactDigests) != 1 || effect.ExpectedArtifactDigests[0] != component.SelectedPackage.ReceiptDigest) {
			return fmt.Errorf("scope install must bind the exact desired receipt")
		}
		if effect.Kind == "uninstall" && component.Presence != "absent" {
			return fmt.Errorf("scope uninstall requires desired absence")
		}
		if effect.Kind == "activate" && component.Activation != "active" {
			return fmt.Errorf("scope activation contradicts desired state")
		}
		if effect.Kind == "dock" && (component.Docking.Disposition == "undocked" || component.Docking.Disposition == "docked" && !equalStringPointer(component.Docking.ReceptorID, effect.TargetReceptorID)) {
			return fmt.Errorf("scope docking contradicts desired state")
		}

	}
	for _, effect := range s.OwnershipEffects {
		component, _ := findDesiredComponent(desired, effect.ComponentID)
		if component.InstallRoot != effect.InstallRoot {
			return fmt.Errorf("execution scope ownership root differs from desired state")
		}
	}
	return nil
}

func (s ExecutionScope) AllowsAction(action PlannedAction, desired DesiredState) bool {
	component, ok := findDesiredComponent(desired, action.ComponentID)
	if !ok || !s.Contains(action.ComponentID) {
		return false
	}
	wanted := ScopeEffect{ComponentID: action.ComponentID, Kind: action.Kind, TargetStateDigest: action.TargetStateDigest, TargetReceptorID: action.TargetReceptorID, ExpectedArtifactDigests: action.ExpectedArtifactDigests, InstallRoot: component.InstallRoot}
	encoded, _ := scopeJSON(wanted)
	for _, effect := range s.Effects {
		candidate, _ := scopeJSON(effect)
		if bytes.Equal(encoded, candidate) {
			return true
		}
	}
	return false
}

func (s ExecutionScope) AllowsOwnership(root, component, receipt, disposition string) bool {
	for _, effect := range s.OwnershipEffects {
		if effect.InstallRoot == root && effect.ComponentID == component && effect.ReceiptDigest == receipt && effect.Disposition == disposition {
			return true
		}
	}
	return false
}

// EffectsComplete reports only the final goals of the reviewed effects. A
// broader profile may still require selection, activation, or other work.
func (s ExecutionScope) EffectsComplete(desired DesiredState, observation Observation, fatal int) bool {
	if fatal != 0 {
		return false
	}
	kinds := map[string]map[string]bool{}
	for _, effect := range s.Effects {
		if kinds[effect.ComponentID] == nil {
			kinds[effect.ComponentID] = map[string]bool{}
		}
		kinds[effect.ComponentID][effect.Kind] = true
	}
	for _, effect := range s.Effects {
		component, present := findObservedComponent(observation, effect.ComponentID)
		wanted, exists := findDesiredComponent(desired, effect.ComponentID)
		if !exists {
			return false
		}
		switch effect.Kind {
		case "install":
			if !present || len(effect.ExpectedArtifactDigests) == 0 {
				return false
			}
			for _, digest := range effect.ExpectedArtifactDigests {
				found := false
				for _, installed := range component.Packages {
					found = found || installed.ReceiptDigest == digest && installed.InstallRoot == effect.InstallRoot && installed.Integrity == "valid" && installed.EntryPointsValidated
				}
				if !found {
					return false
				}
			}
		case "uninstall":
			if present {
				return false
			}
		case "select":
			if !present || wanted.SelectedPackage == nil || component.SelectedPackageDigest == nil || *component.SelectedPackageDigest != wanted.SelectedPackage.ReceiptDigest {
				return false
			}
		case "deselect":
			if !kinds[effect.ComponentID]["select"] && present && component.SelectedPackageDigest != nil {
				return false
			}
		case "activate":
			if !present || component.Activation != "active" {
				return false
			}
		case "deactivate":
			if !kinds[effect.ComponentID]["activate"] && present && component.Activation != "inactive" {
				return false
			}
		case "dock":
			if !present || component.Docking != "docked" || effect.TargetReceptorID == nil || component.ReceptorID == nil || *component.ReceptorID != *effect.TargetReceptorID {
				return false
			}
		case "undock":
			if !kinds[effect.ComponentID]["dock"] && present && (component.Docking != "undocked" || component.ReceptorID != nil) {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func (s ExecutionScope) PreservedComponentsDigest(observation Observation) (string, error) {
	values := make([]ObservedComponent, 0)
	for _, component := range observation.Components {
		if !s.Contains(component.ComponentID) {
			values = append(values, component)
		}
	}
	sort.Slice(values, func(i, j int) bool {
		a, _ := scopeJSON(values[i])
		b, _ := scopeJSON(values[j])
		return strings.Compare(string(a), string(b)) < 0
	})
	return scopeValueDigest(values)
}

// ActionPreservedComponentsDigest binds every observation field except the
// exact action's effects. It is used before resuming an interrupted external
// action as well as by the native coordinator when finalizing its evidence.
func ActionPreservedComponentsDigest(observation Observation, action PlannedAction) (string, error) {
	values := make([]any, 0, len(observation.Components))
	for _, component := range observation.Components {
		if component.ComponentID != action.ComponentID {
			values = append(values, component)
			continue
		}
		raw, err := scopeJSON(component)
		if err != nil {
			return "", err
		}
		var projected map[string]any
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		if err := decoder.Decode(&projected); err != nil {
			return "", err
		}
		delete(projected, "observation_digest")
		switch action.Kind {
		case "activate", "deactivate":
			delete(projected, "activation")
		case "dock", "undock":
			delete(projected, "docking")
			delete(projected, "receptor_id")
		case "select", "deselect":
			delete(projected, "selected_package_digest")
			delete(projected, "capabilities")
			delete(projected, "platform_compatibility")
		case "install", "uninstall":
			retained := make([]any, 0)
			for _, installed := range component.Packages {
				if !containsString(action.ExpectedArtifactDigests, installed.ReceiptDigest) {
					retained = append(retained, installed)
				}
			}
			if len(retained) == 0 {
				if component.Activation != "inactive" || component.SelectedPackageDigest != nil || component.Docking != "undocked" || component.ReceptorID != nil {
					return "", fmt.Errorf("package action cannot change or conceal runtime selection, activation, or docking")
				}
				continue
			}
			sortScopeValues(retained)
			projected["packages"] = retained
		default:
			return "", fmt.Errorf("unsupported scoped action projection")
		}
		if packages, ok := projected["packages"].([]any); ok {
			sortScopeValues(packages)
		}
		if capabilities, ok := projected["capabilities"].([]any); ok {
			sortScopeValues(capabilities)
		}
		values = append(values, projected)
	}
	envelopeBytes, err := scopeJSON(observation)
	if err != nil {
		return "", err
	}
	var envelope map[string]any
	decoder := json.NewDecoder(bytes.NewReader(envelopeBytes))
	decoder.UseNumber()
	if err := decoder.Decode(&envelope); err != nil {
		return "", err
	}
	delete(envelope, "observation_digest")
	delete(envelope, "observed_at")
	envelope["components"] = values
	if action.Kind == "select" || action.Kind == "deselect" {
		delete(envelope, "binding_registry_digest")
	}
	if action.ComponentID == "knowledge-session-coordinator" && action.Kind == "select" {
		if platform, ok := envelope["platform"].(map[string]any); ok {
			delete(platform, "coordinator_identity")
			delete(platform, "compatibility_digest")
		}
	}
	// Normalize struct-valued retained components as JSON objects before sorting.
	normalizedBytes, err := scopeJSON(envelope)
	if err != nil {
		return "", err
	}
	decoder = json.NewDecoder(bytes.NewReader(normalizedBytes))
	decoder.UseNumber()
	if err := decoder.Decode(&envelope); err != nil {
		return "", err
	}
	for _, field := range []string{"configured_roots", "unknown_packages"} {
		if array, ok := envelope[field].([]any); ok {
			sortScopeValues(array)
		}
	}
	if platform, ok := envelope["platform"].(map[string]any); ok {
		if providers, ok := platform["provider_availability"].([]any); ok {
			sortScopeValues(providers)
		}
	}
	if components, ok := envelope["components"].([]any); ok {
		for _, value := range components {
			if component, ok := value.(map[string]any); ok {
				for _, field := range []string{"packages", "capabilities"} {
					if array, ok := component[field].([]any); ok {
						sortScopeValues(array)
					}
				}
			}
		}
		sortScopeValues(components)
	}
	return scopeValueDigest(envelope)
}
func sortScopeValues(values []any) {
	sort.Slice(values, func(i, j int) bool {
		a, _ := scopeJSON(values[i])
		b, _ := scopeJSON(values[j])
		return bytes.Compare(a, b) < 0
	})
}

func (s ExecutionScope) ValidateInitial(desired DesiredState, observation Observation, planDigest, stableDigest, expectedJournal, expectedApplied string) error {
	if s.InitialPlanDigest != planDigest || s.InitialStableInventoryDigest != stableDigest || s.ExpectedApplyJournalDigest != expectedJournal || s.ExpectedAppliedStateDigest != expectedApplied {
		return fmt.Errorf("scope initial plan, inventory or compare-and-swap evidence is stale")
	}
	for _, id := range s.Components {
		component, ok := findDesiredComponent(desired, id)
		if !ok {
			return fmt.Errorf("scope component is absent from complete desired state")
		}
		receipts := map[string]bool{}
		if component.SelectedPackage != nil {
			receipts[component.SelectedPackage.ReceiptDigest] = true
		}
		if observed, present := findObservedComponent(observation, id); present {
			for _, installed := range observed.Packages {
				if installed.InstallRoot == component.InstallRoot {
					receipts[installed.ReceiptDigest] = true
				}
			}
		}
		hasEffect := false
		for _, effect := range s.Effects {
			if effect.ComponentID == id {
				hasEffect = true
				if effect.Kind == "dock" && component.Docking.Disposition == "unmanaged" {
					observed, present := findObservedComponent(observation, id)
					if !present || observed.Docking != "docked" || !equalStringPointer(observed.ReceptorID, effect.TargetReceptorID) {
						return fmt.Errorf("unmanaged docking scope must restore its initial observed receptor")
					}
				}
				for _, digest := range effect.ExpectedArtifactDigests {
					if !receipts[digest] {
						return fmt.Errorf("scope action names an unobserved receipt")
					}
				}
			}
		}
		if !hasEffect {
			return fmt.Errorf("scope component lacks explicit effect")
		}
		for _, effect := range s.OwnershipEffects {
			if effect.ComponentID == id && !receipts[effect.ReceiptDigest] {
				return fmt.Errorf("scope ownership names an unobserved receipt")
			}
		}
	}
	return nil
}

// InitialScopePlanDigest retains the complete reviewed plan semantics while
// replacing only its volatile observation timestamp binding with the separately
// validated stable inventory digest. The original native plan remains evidence.
func InitialScopePlanDigest(plan any, stableDigest string) (string, error) {
	if !taggedDigest(stableDigest) {
		return "", fmt.Errorf("invalid stable inventory digest")
	}
	raw, err := scopeJSON(plan)
	if err != nil {
		return "", err
	}
	var object map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&object); err != nil {
		return "", err
	}
	if object["protocol"] != "symphony.knowledge.lifecycle-plan.v1" {
		return "", fmt.Errorf("invalid initial scope plan protocol")
	}
	delete(object, "plan_digest")
	object["observation_digest"] = stableDigest
	return scopeValueDigest(object)
}
