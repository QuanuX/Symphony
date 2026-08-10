package knowledgelifecycle

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgebinding"
)

const (
	RuntimeProtocol = "symphony.knowledge.lifecycle-runtime-state.v1"
	maxRuntimeBytes = 2 << 20
)

type RuntimeComponent struct {
	ComponentID           string  `json:"component_id"`
	SelectedReceiptDigest *string `json:"selected_receipt_digest"`
	Activation            string  `json:"activation"`
	Docking               string  `json:"docking"`
	ReceptorID            *string `json:"receptor_id"`
	ComponentStateDigest  string  `json:"component_state_digest"`
}

type RuntimeState struct {
	Protocol                   string             `json:"protocol"`
	FormatVersion              uint64             `json:"format_version"`
	ProfileID                  string             `json:"profile_id"`
	TOPSID                     string             `json:"tops_id"`
	Generation                 uint64             `json:"generation"`
	PreviousRuntimeStateDigest *string            `json:"previous_runtime_state_digest"`
	Components                 []RuntimeComponent `json:"components"`
	UpdatedAt                  string             `json:"updated_at"`
	Canonical                  bool               `json:"canonical"`
	RuntimeStateDigest         string             `json:"runtime_state_digest"`
}

type RuntimeSnapshot struct {
	Exists bool
	State  RuntimeState
}

type RuntimeStore struct {
	stateRoot string
	topsID    string
	profileID string
	now       func() time.Time
}

func NewRuntimeStore(stateRoot, topsID, profileID string) (*RuntimeStore, error) {
	if !validTOPSID(topsID) {
		return nil, fmt.Errorf("TOPS ID must be a canonical non-nil lowercase RFC UUID")
	}
	if !safeToken(profileID, 256) {
		return nil, fmt.Errorf("profile ID has invalid syntax")
	}
	if stateRoot == "" {
		var err error
		stateRoot, err = knowledgebinding.DefaultStateRoot()
		if err != nil {
			return nil, err
		}
	}
	canonical, err := canonicalStateRoot(stateRoot)
	if err != nil {
		return nil, err
	}
	return &RuntimeStore{
		stateRoot: canonical, topsID: topsID, profileID: profileID,
		now: func() time.Time { return time.Now().UTC() },
	}, nil
}

func (s *RuntimeStore) Snapshot() (RuntimeSnapshot, error) {
	var snapshot RuntimeSnapshot
	err := s.withRuntimeLock(false, false, func(directory runtimeDirectory) error {
		data, exists, err := readRuntimeFile(directory)
		if err != nil || !exists {
			snapshot.Exists = exists
			return err
		}
		state, err := decodeRuntimeState(data)
		if err != nil {
			return err
		}
		if state.TOPSID != s.topsID || state.ProfileID != s.profileID {
			return fmt.Errorf("lifecycle runtime state identity mismatch")
		}
		snapshot = RuntimeSnapshot{Exists: true, State: state}
		return nil
	})
	return snapshot, err
}

func (s *RuntimeStore) Mutate(
	componentID, action string,
	selectedReceipt *string,
	expected string,
) (RuntimeState, bool, error) {
	if !safeToken(componentID, 256) || !oneOf(action, "select", "deselect", "activate", "deactivate") {
		return RuntimeState{}, false, fmt.Errorf("lifecycle runtime mutation identity is invalid")
	}
	if selectedReceipt != nil && !taggedDigest(*selectedReceipt) {
		return RuntimeState{}, false, fmt.Errorf("selected receipt digest is invalid")
	}
	if expected != "absent" && !taggedDigest(expected) {
		return RuntimeState{}, false, fmt.Errorf("expected runtime state must be absent or an exact tagged SHA-256 digest")
	}
	var result RuntimeState
	changed := false
	err := s.withRuntimeLock(true, true, func(directory runtimeDirectory) error {
		data, exists, err := readRuntimeFile(directory)
		if err != nil {
			return err
		}
		var current RuntimeState
		if exists {
			current, err = decodeRuntimeState(data)
			if err != nil {
				return err
			}
			if current.TOPSID != s.topsID || current.ProfileID != s.profileID {
				return fmt.Errorf("lifecycle runtime state identity mismatch")
			}
		}
		if (expected == "absent") != !exists || (exists && expected != current.RuntimeStateDigest) {
			return fmt.Errorf("lifecycle runtime state compare-and-swap mismatch")
		}
		next := current
		if !exists {
			next = RuntimeState{
				Protocol: RuntimeProtocol, FormatVersion: 1, ProfileID: s.profileID,
				TOPSID: s.topsID, Generation: 1, Components: make([]RuntimeComponent, 0), Canonical: false,
			}
		} else {
			next.Generation++
			next.PreviousRuntimeStateDigest = stringPointer(current.RuntimeStateDigest)
		}
		index := -1
		for i := range next.Components {
			if next.Components[i].ComponentID == componentID {
				index = i
				break
			}
		}
		component := RuntimeComponent{
			ComponentID: componentID, Activation: "inactive", Docking: "undocked", ReceptorID: nil,
		}
		if index >= 0 {
			component = next.Components[index]
		}
		switch action {
		case "select":
			if selectedReceipt == nil {
				return fmt.Errorf("select requires an exact receipt digest")
			}
			component.SelectedReceiptDigest = cloneString(selectedReceipt)
		case "deselect":
			if component.Activation == "active" {
				return fmt.Errorf("cannot deselect an active lifecycle component")
			}
			component.SelectedReceiptDigest = nil
		case "activate":
			if component.SelectedReceiptDigest == nil {
				return fmt.Errorf("cannot activate an unselected lifecycle component")
			}
			component.Activation = "active"
		case "deactivate":
			component.Activation = "inactive"
		}
		if index >= 0 && runtimeComponentEquivalent(next.Components[index], component) {
			result = current
			return nil
		}
		if !exists {
			next.Generation = 1
			next.PreviousRuntimeStateDigest = nil
		}
		if index < 0 {
			next.Components = append(next.Components, component)
		} else {
			next.Components[index] = component
		}
		if component.SelectedReceiptDigest == nil && component.Activation == "inactive" {
			next.Components = removeEmptyRuntimeComponent(next.Components, componentID)
		}
		next.UpdatedAt = s.now().UTC().Truncate(time.Second).Format(time.RFC3339)
		if err := finalizeRuntimeState(&next); err != nil {
			return err
		}
		encoded, err := json.Marshal(next)
		if err != nil {
			return err
		}
		encoded = append(encoded, '\n')
		if err := writeRuntimeFile(directory, encoded); err != nil {
			return err
		}
		result, changed = next, true
		return nil
	})
	return result, changed, err
}

func decodeRuntimeState(data []byte) (RuntimeState, error) {
	if err := validateBoundedRuntimeJSON(data); err != nil {
		return RuntimeState{}, err
	}
	var state RuntimeState
	if err := decodeExact(data, &state); err != nil {
		return RuntimeState{}, fmt.Errorf("decode lifecycle runtime state: %w", err)
	}
	if state.Protocol != RuntimeProtocol || state.FormatVersion != 1 || !safeToken(state.ProfileID, 256) ||
		!validTOPSID(state.TOPSID) || state.Generation == 0 || state.Generation > 9007199254740991 || state.Canonical ||
		!validSTSCSeconds(state.UpdatedAt) || len(state.Components) > 4096 || !taggedDigest(state.RuntimeStateDigest) {
		return RuntimeState{}, fmt.Errorf("lifecycle runtime state identity is invalid")
	}
	if (state.Generation == 1) != (state.PreviousRuntimeStateDigest == nil) ||
		(state.PreviousRuntimeStateDigest != nil && !taggedDigest(*state.PreviousRuntimeStateDigest)) {
		return RuntimeState{}, fmt.Errorf("lifecycle runtime predecessor is invalid")
	}
	seen := make(map[string]struct{}, len(state.Components))
	for _, component := range state.Components {
		if !safeToken(component.ComponentID, 256) || component.Activation != "inactive" && component.Activation != "active" ||
			component.Docking != "undocked" || component.ReceptorID != nil ||
			component.SelectedReceiptDigest != nil && !taggedDigest(*component.SelectedReceiptDigest) ||
			component.Activation == "active" && component.SelectedReceiptDigest == nil ||
			!taggedDigest(component.ComponentStateDigest) {
			return RuntimeState{}, fmt.Errorf("lifecycle runtime component is invalid")
		}
		if _, duplicate := seen[component.ComponentID]; duplicate {
			return RuntimeState{}, fmt.Errorf("lifecycle runtime component is duplicated")
		}
		seen[component.ComponentID] = struct{}{}
		copy := component
		copy.ComponentStateDigest = ""
		value, err := objectWithout(mustJSON(copy), "component_state_digest")
		if err != nil {
			return RuntimeState{}, err
		}
		digest, err := digestValue(value)
		if err != nil || digest != component.ComponentStateDigest {
			return RuntimeState{}, fmt.Errorf("lifecycle runtime component digest mismatch")
		}
	}
	sort.Slice(state.Components, func(i, j int) bool { return canonicalLess(state.Components[i], state.Components[j]) })
	copy := state
	copy.RuntimeStateDigest = ""
	value, err := objectWithout(mustJSON(copy), "runtime_state_digest")
	if err != nil {
		return RuntimeState{}, err
	}
	digest, err := digestValue(value)
	if err != nil || digest != state.RuntimeStateDigest {
		return RuntimeState{}, fmt.Errorf("lifecycle runtime state digest mismatch")
	}
	return state, nil
}

func finalizeRuntimeState(state *RuntimeState) error {
	sort.Slice(state.Components, func(i, j int) bool { return canonicalLess(state.Components[i], state.Components[j]) })
	for index := range state.Components {
		component := state.Components[index]
		component.ComponentStateDigest = ""
		value, err := objectWithout(mustJSON(component), "component_state_digest")
		if err != nil {
			return err
		}
		component.ComponentStateDigest, err = digestValue(value)
		if err != nil {
			return err
		}
		state.Components[index] = component
	}
	state.RuntimeStateDigest = ""
	value, err := objectWithout(mustJSON(*state), "runtime_state_digest")
	if err != nil {
		return err
	}
	state.RuntimeStateDigest, err = digestValue(value)
	return err
}

func runtimeComponentEquivalent(left, right RuntimeComponent) bool {
	return left.ComponentID == right.ComponentID && equalStringPointer(left.SelectedReceiptDigest, right.SelectedReceiptDigest) &&
		left.Activation == right.Activation && left.Docking == right.Docking && equalStringPointer(left.ReceptorID, right.ReceptorID)
}

func removeEmptyRuntimeComponent(components []RuntimeComponent, id string) []RuntimeComponent {
	result := make([]RuntimeComponent, 0, len(components))
	for _, component := range components {
		if component.ComponentID != id {
			result = append(result, component)
		}
	}
	return result
}

func validateBoundedRuntimeJSON(data []byte) error {
	if len(data) == 0 || len(data) > maxRuntimeBytes {
		return fmt.Errorf("lifecycle runtime state exceeds its bound")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return fmt.Errorf("lifecycle runtime state is invalid JSON: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("lifecycle runtime state contains trailing JSON")
	}
	return nil
}

func runtimeProfileFile(profileID string) string {
	digest := sha256.Sum256([]byte("profile:" + profileID))
	return "runtime-" + hex.EncodeToString(digest[:]) + ".json"
}

func validSTSCSeconds(value string) bool {
	parsed, err := time.Parse(time.RFC3339, value)
	return err == nil && parsed.Nanosecond() == 0 && parsed.UTC().Format(time.RFC3339) == value
}
