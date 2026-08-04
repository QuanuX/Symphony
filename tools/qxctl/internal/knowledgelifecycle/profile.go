package knowledgelifecycle

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgebinding"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
)

const (
	ProfileInputProtocol = "symphony.knowledge.lifecycle-profile-input.v1"
	ProfileProtocol      = "symphony.knowledge.lifecycle-profile.v1"
	DesiredProtocol      = "symphony.knowledge.lifecycle-desired-state.v1"
	ObservationProtocol  = "symphony.knowledge.lifecycle-observation.v1"
	maxProfileBytes      = 1 << 20
	maxProfiles          = 256
)

type PackageIdentity struct {
	PackageID       string `json:"package_id"`
	Version         string `json:"version"`
	ReceiptProtocol string `json:"receipt_protocol"`
	ReceiptDigest   string `json:"receipt_digest"`
}

type Docking struct {
	Disposition string  `json:"disposition"`
	ReceptorID  *string `json:"receptor_id"`
}

type Dependency struct {
	TargetComponentID string `json:"target_component_id"`
	Condition         string `json:"condition"`
	Critical          bool   `json:"critical"`
}

type Compatibility struct {
	RequiredCapabilities []string `json:"required_capabilities"`
	PlatformRequirements []string `json:"platform_requirements"`
}

type Extension struct {
	ExtensionID      string `json:"extension_id"`
	ExtensionVersion string `json:"extension_version"`
	Critical         bool   `json:"critical"`
	Payload          any    `json:"payload"`
	PayloadDigest    string `json:"payload_digest"`
}

type DesiredComponent struct {
	ComponentID     string           `json:"component_id"`
	ComponentKind   string           `json:"component_kind"`
	ModuleID        string           `json:"module_id"`
	VectorID        *string          `json:"vector_id"`
	EngineID        *string          `json:"engine_id"`
	Presence        string           `json:"presence"`
	SelectedPackage *PackageIdentity `json:"selected_package"`
	Required        bool             `json:"required"`
	InstallScope    string           `json:"install_scope"`
	InstallRoot     string           `json:"install_root"`
	Activation      string           `json:"activation"`
	Docking         Docking          `json:"docking"`
	Dependencies    []Dependency     `json:"dependencies"`
	Compatibility   Compatibility    `json:"compatibility"`
	Extensions      []Extension      `json:"extensions"`
}

type ProfileInput struct {
	Protocol        string             `json:"protocol"`
	FormatVersion   uint64             `json:"format_version"`
	ProfileID       string             `json:"profile_id"`
	TOPSID          string             `json:"tops_id"`
	ConfiguredRoots []string           `json:"configured_roots"`
	BootMode        string             `json:"boot_mode"`
	Components      []DesiredComponent `json:"components"`
	Extensions      []Extension        `json:"extensions"`
}

type DesiredState struct {
	Protocol                   string             `json:"protocol"`
	FormatVersion              uint64             `json:"format_version"`
	ProfileID                  string             `json:"profile_id"`
	TOPSID                     string             `json:"tops_id"`
	Generation                 uint64             `json:"generation"`
	PreviousDesiredStateDigest *string            `json:"previous_desired_state_digest"`
	Components                 []DesiredComponent `json:"components"`
	Extensions                 []Extension        `json:"extensions"`
	Canonical                  bool               `json:"canonical"`
	DesiredStateDigest         string             `json:"desired_state_digest"`
}

type Profile struct {
	Protocol              string       `json:"protocol"`
	FormatVersion         uint64       `json:"format_version"`
	ProfileID             string       `json:"profile_id"`
	TOPSID                string       `json:"tops_id"`
	Generation            uint64       `json:"generation"`
	PreviousProfileDigest *string      `json:"previous_profile_digest"`
	ConfiguredRoots       []string     `json:"configured_roots"`
	BootMode              string       `json:"boot_mode"`
	DesiredState          DesiredState `json:"desired_state"`
	Canonical             bool         `json:"canonical"`
	ProfileDigest         string       `json:"profile_digest"`
}

type Snapshot struct {
	Exists  bool    `json:"exists"`
	Profile Profile `json:"profile"`
}

type ProfileSummary struct {
	ProfileID      string `json:"profile_id"`
	TOPSID         string `json:"tops_id"`
	Generation     uint64 `json:"generation"`
	BootMode       string `json:"boot_mode"`
	ComponentCount int    `json:"component_count"`
	ProfileDigest  string `json:"profile_digest"`
}

type ListResult struct {
	Schema     string           `json:"schema"`
	TOPSID     string           `json:"tops_id"`
	Profiles   []ProfileSummary `json:"profiles"`
	Canonical  bool             `json:"canonical"`
	ListDigest string           `json:"list_digest"`
}

type Store struct {
	stateRoot string
	topsID    string
}

func NewStore(stateRoot, topsID string) (*Store, error) {
	if !validTOPSID(topsID) {
		return nil, fmt.Errorf("TOPS ID must be a canonical non-nil lowercase RFC UUID")
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
	return &Store{stateRoot: canonical, topsID: topsID}, nil
}

func (s *Store) StateRoot() string { return s.stateRoot }

func DecodeProfileInput(data []byte) (ProfileInput, error) {
	if err := knowledgeengine.ValidateJSONObject(data, maxProfileBytes); err != nil {
		return ProfileInput{}, fmt.Errorf("lifecycle profile input violates bounded JSON rules: %w", err)
	}
	var input ProfileInput
	if err := decodeExact(data, &input); err != nil {
		return ProfileInput{}, fmt.Errorf("decode lifecycle profile input: %w", err)
	}
	if err := validateInput(&input); err != nil {
		return ProfileInput{}, err
	}
	return input, nil
}

func ProfileInputDigest(input ProfileInput) (string, error) {
	if err := validateInput(&input); err != nil {
		return "", err
	}
	return digestValue(input)
}

func (s *Store) Snapshot(profileID string) (Snapshot, error) {
	if !safeToken(profileID, 256) {
		return Snapshot{}, fmt.Errorf("profile ID has invalid syntax")
	}
	var snapshot Snapshot
	err := s.withProfileLock(false, func(directory *os.File) error {
		data, exists, err := readProfileFile(directory, profileID)
		if err != nil || !exists {
			snapshot.Exists = exists
			return err
		}
		profile, err := decodeProfile(data)
		if err != nil {
			return err
		}
		if profile.ProfileID != profileID || profile.TOPSID != s.topsID {
			return fmt.Errorf("lifecycle profile storage identity mismatch")
		}
		snapshot = Snapshot{Exists: true, Profile: profile}
		return nil
	})
	return snapshot, err
}

func (s *Store) List() (ListResult, error) {
	result := ListResult{
		Schema: "qxctl.knowledge.lifecycle-profile-list.v1",
		TOPSID: s.topsID, Profiles: make([]ProfileSummary, 0), Canonical: false,
	}
	err := s.withProfileLock(false, func(directory *os.File) error {
		files, err := listProfileFiles(directory)
		if err != nil {
			return err
		}
		if len(files) > maxProfiles {
			return fmt.Errorf("lifecycle profile count exceeds %d", maxProfiles)
		}
		for _, file := range files {
			profile, err := decodeProfile(file.data)
			if err != nil {
				return err
			}
			if profile.TOPSID != s.topsID || file.name != profileFileName(profile.ProfileID) {
				return fmt.Errorf("lifecycle profile storage identity mismatch")
			}
			result.Profiles = append(result.Profiles, ProfileSummary{
				ProfileID: profile.ProfileID, TOPSID: profile.TOPSID,
				Generation: profile.Generation, BootMode: profile.BootMode,
				ComponentCount: len(profile.DesiredState.Components), ProfileDigest: profile.ProfileDigest,
			})
		}
		return nil
	})
	if err != nil {
		return ListResult{}, err
	}
	sort.Slice(result.Profiles, func(i, j int) bool { return result.Profiles[i].ProfileID < result.Profiles[j].ProfileID })
	digest, err := digestValue(map[string]any{
		"schema": result.Schema, "tops_id": result.TOPSID,
		"profiles": result.Profiles, "canonical": result.Canonical,
	})
	if err != nil {
		return ListResult{}, err
	}
	result.ListDigest = digest
	return result, nil
}

func (s *Store) Set(input ProfileInput, expected string) (Profile, bool, error) {
	if input.TOPSID != s.topsID {
		return Profile{}, false, fmt.Errorf("profile input TOPS does not match the selected TOPS")
	}
	if err := validateExpected(expected); err != nil {
		return Profile{}, false, err
	}
	var result Profile
	changed := false
	err := s.withProfileLock(true, func(directory *os.File) error {
		data, exists, err := readProfileFile(directory, input.ProfileID)
		if err != nil {
			return err
		}
		var current Profile
		if exists {
			current, err = decodeProfile(data)
			if err != nil {
				return err
			}
			if current.TOPSID != s.topsID || current.ProfileID != input.ProfileID {
				return fmt.Errorf("lifecycle profile storage identity mismatch")
			}
			if sameIntent(current, input) {
				result = current
				return nil
			}
		}
		if err := requireExpected(current, exists, expected); err != nil {
			return err
		}
		next, err := buildProfile(input, current, exists)
		if err != nil {
			return err
		}
		encoded, err := encodeProfile(next)
		if err != nil {
			return err
		}
		if err := writeProfileFile(directory, input.ProfileID, encoded); err != nil {
			return err
		}
		result = next
		changed = true
		return nil
	})
	return result, changed, err
}

func (s *Store) Remove(profileID, expected string) (bool, error) {
	if !safeToken(profileID, 256) {
		return false, fmt.Errorf("profile ID has invalid syntax")
	}
	if !taggedDigest(expected) {
		return false, fmt.Errorf("--expected-profile-digest must be an exact tagged SHA-256 digest")
	}
	changed := false
	err := s.withProfileLock(true, func(directory *os.File) error {
		data, exists, err := readProfileFile(directory, profileID)
		if err != nil {
			return err
		}
		if !exists {
			return nil
		}
		current, err := decodeProfile(data)
		if err != nil {
			return err
		}
		if current.TOPSID != s.topsID || current.ProfileID != profileID {
			return fmt.Errorf("lifecycle profile storage identity mismatch")
		}
		if err := requireExpected(current, true, expected); err != nil {
			return err
		}
		if err := removeProfileFile(directory, profileID); err != nil {
			return err
		}
		changed = true
		return nil
	})
	return changed, err
}

func buildProfile(input ProfileInput, current Profile, exists bool) (Profile, error) {
	generation := uint64(1)
	var previousProfile, previousDesired *string
	if exists {
		if current.Generation >= 9007199254740991 {
			return Profile{}, fmt.Errorf("lifecycle profile generation is exhausted")
		}
		generation = current.Generation + 1
		previousProfile = stringPointer(current.ProfileDigest)
		previousDesired = stringPointer(current.DesiredState.DesiredStateDigest)
	}
	desired := DesiredState{
		Protocol: DesiredProtocol, FormatVersion: 1, ProfileID: input.ProfileID,
		TOPSID: input.TOPSID, Generation: generation,
		PreviousDesiredStateDigest: previousDesired,
		Components:                 cloneComponents(input.Components), Extensions: cloneExtensions(input.Extensions),
		Canonical: false,
	}
	normalizeDesired(&desired)
	digest, err := desiredDigest(desired)
	if err != nil {
		return Profile{}, err
	}
	desired.DesiredStateDigest = digest
	profile := Profile{
		Protocol: ProfileProtocol, FormatVersion: 1, ProfileID: input.ProfileID,
		TOPSID: input.TOPSID, Generation: generation, PreviousProfileDigest: previousProfile,
		ConfiguredRoots: append([]string(nil), input.ConfiguredRoots...), BootMode: input.BootMode,
		DesiredState: desired, Canonical: false,
	}
	sort.Strings(profile.ConfiguredRoots)
	profile.ProfileDigest, err = profileDigest(profile)
	if err != nil {
		return Profile{}, err
	}
	if err := validateProfile(profile); err != nil {
		return Profile{}, err
	}
	return profile, nil
}

func decodeProfile(data []byte) (Profile, error) {
	if err := knowledgeengine.ValidateJSONObject(data, maxProfileBytes); err != nil {
		return Profile{}, fmt.Errorf("stored lifecycle profile violates bounded JSON rules: %w", err)
	}
	var profile Profile
	if err := decodeExact(data, &profile); err != nil {
		return Profile{}, fmt.Errorf("decode stored lifecycle profile: %w", err)
	}
	if err := validateProfile(profile); err != nil {
		return Profile{}, err
	}
	return profile, nil
}

func validateProfile(profile Profile) error {
	if profile.Protocol != ProfileProtocol || profile.FormatVersion != 1 || profile.Canonical ||
		!safeToken(profile.ProfileID, 256) || !validTOPSID(profile.TOPSID) ||
		profile.Generation == 0 || profile.Generation > 9007199254740991 ||
		!taggedDigest(profile.ProfileDigest) {
		return fmt.Errorf("lifecycle profile identity or generation is invalid")
	}
	if profile.Generation == 1 && profile.PreviousProfileDigest != nil {
		return fmt.Errorf("first lifecycle profile generation has a previous digest")
	}
	if profile.Generation > 1 && (profile.PreviousProfileDigest == nil || !taggedDigest(*profile.PreviousProfileDigest)) {
		return fmt.Errorf("later lifecycle profile generation lacks a previous digest")
	}
	input := ProfileInput{
		Protocol: ProfileInputProtocol, FormatVersion: 1, ProfileID: profile.ProfileID,
		TOPSID: profile.TOPSID, ConfiguredRoots: append([]string(nil), profile.ConfiguredRoots...),
		BootMode: profile.BootMode, Components: cloneComponents(profile.DesiredState.Components),
		Extensions: cloneExtensions(profile.DesiredState.Extensions),
	}
	if err := validateInput(&input); err != nil {
		return err
	}
	desired := profile.DesiredState
	if desired.Protocol != DesiredProtocol || desired.FormatVersion != 1 || desired.Canonical ||
		desired.ProfileID != profile.ProfileID || desired.TOPSID != profile.TOPSID ||
		desired.Generation != profile.Generation || !taggedDigest(desired.DesiredStateDigest) {
		return fmt.Errorf("embedded desired-state identity is invalid")
	}
	if profile.Generation == 1 && desired.PreviousDesiredStateDigest != nil {
		return fmt.Errorf("first desired-state generation has a previous digest")
	}
	if profile.Generation > 1 && (desired.PreviousDesiredStateDigest == nil || !taggedDigest(*desired.PreviousDesiredStateDigest)) {
		return fmt.Errorf("later desired-state generation lacks a previous digest")
	}
	expectedDesired, err := desiredDigest(desired)
	if err != nil || expectedDesired != desired.DesiredStateDigest {
		return fmt.Errorf("desired-state digest mismatch")
	}
	expectedProfile, err := profileDigest(profile)
	if err != nil || expectedProfile != profile.ProfileDigest {
		return fmt.Errorf("lifecycle profile digest mismatch")
	}
	return nil
}

func validateInput(input *ProfileInput) error {
	if input.Protocol != ProfileInputProtocol || input.FormatVersion != 1 ||
		!safeToken(input.ProfileID, 256) || !validTOPSID(input.TOPSID) ||
		(input.BootMode != "report" && input.BootMode != "apply-compatible") ||
		input.Components == nil || input.Extensions == nil {
		return fmt.Errorf("lifecycle profile input identity or boot mode is invalid")
	}
	if len(input.ConfiguredRoots) == 0 || len(input.ConfiguredRoots) > 64 || len(input.Components) > 4096 || len(input.Extensions) > 64 {
		return fmt.Errorf("lifecycle profile input exceeds a collection bound")
	}
	rootSet := make(map[string]struct{}, len(input.ConfiguredRoots))
	for index, root := range input.ConfiguredRoots {
		if !safeAbsolutePath(root) {
			return fmt.Errorf("configured root %d is not a safe absolute path", index)
		}
		if _, duplicate := rootSet[root]; duplicate {
			return fmt.Errorf("configured roots contain a duplicate")
		}
		rootSet[root] = struct{}{}
	}
	componentIDs := make(map[string]struct{}, len(input.Components))
	for index := range input.Components {
		component := &input.Components[index]
		if err := validateComponent(component, rootSet); err != nil {
			return fmt.Errorf("component %d: %w", index, err)
		}
		if _, duplicate := componentIDs[component.ComponentID]; duplicate {
			return fmt.Errorf("desired component identity is duplicated")
		}
		componentIDs[component.ComponentID] = struct{}{}
	}
	if err := validateExtensions(input.Extensions); err != nil {
		return fmt.Errorf("profile extensions: %w", err)
	}
	normalizeInput(input)
	return nil
}

func validateComponent(component *DesiredComponent, roots map[string]struct{}) error {
	if !safeToken(component.ComponentID, 256) || !safeToken(component.ModuleID, 256) ||
		!oneOf(component.ComponentKind, "coordinator", "vector_engine", "module", "adapter", "ui", "service") ||
		!oneOf(component.Presence, "present", "absent") ||
		!oneOf(component.InstallScope, "prefix", "user", "system", "tops") ||
		!oneOf(component.Activation, "inactive", "active", "unmanaged") ||
		!safeAbsolutePath(component.InstallRoot) {
		return fmt.Errorf("identity, lifecycle, scope, or install root is invalid")
	}
	if _, configured := roots[component.InstallRoot]; !configured {
		return fmt.Errorf("install root is outside the configured root set")
	}
	for _, optional := range []*string{component.VectorID, component.EngineID} {
		if optional != nil && !safeToken(*optional, 256) {
			return fmt.Errorf("optional identity token is invalid")
		}
	}
	if !oneOf(component.Docking.Disposition, "undocked", "docked", "unmanaged") ||
		(component.Docking.Disposition == "docked") != (component.Docking.ReceptorID != nil) ||
		(component.Docking.ReceptorID != nil && !safeToken(*component.Docking.ReceptorID, 256)) {
		return fmt.Errorf("docking state is invalid")
	}
	if component.Presence == "present" {
		if component.SelectedPackage == nil {
			return fmt.Errorf("present component requires an exact selected package")
		}
	} else if component.SelectedPackage != nil || component.Activation == "active" || component.Docking.Disposition == "docked" {
		return fmt.Errorf("absent component carries active package state")
	}
	if component.SelectedPackage != nil {
		selected := component.SelectedPackage
		if !safeToken(selected.PackageID, 256) || !safeVersion(selected.Version) ||
			!oneOf(selected.ReceiptProtocol, "symphony.knowledge.install-receipt.v1", "symphony.knowledge.install-receipt.v2") ||
			!taggedDigest(selected.ReceiptDigest) {
			return fmt.Errorf("selected package identity is invalid")
		}
	}
	if len(component.Dependencies) > 256 || len(component.Compatibility.RequiredCapabilities) > 128 ||
		len(component.Compatibility.PlatformRequirements) > 128 || len(component.Extensions) > 64 ||
		component.Dependencies == nil || component.Compatibility.RequiredCapabilities == nil ||
		component.Compatibility.PlatformRequirements == nil || component.Extensions == nil {
		return fmt.Errorf("component collection bound is exceeded")
	}
	dependencySet := make(map[string]struct{}, len(component.Dependencies))
	for _, dependency := range component.Dependencies {
		if !safeToken(dependency.TargetComponentID, 256) || dependency.TargetComponentID == component.ComponentID ||
			!oneOf(dependency.Condition, "present", "absent", "installed", "active", "inactive", "docked", "undocked", "compatible") {
			return fmt.Errorf("dependency is invalid")
		}
		key := dependency.TargetComponentID + "\x00" + dependency.Condition
		if _, duplicate := dependencySet[key]; duplicate {
			return fmt.Errorf("dependency is duplicated")
		}
		dependencySet[key] = struct{}{}
	}
	if err := validateTokenSet(component.Compatibility.RequiredCapabilities, 128); err != nil {
		return fmt.Errorf("required capabilities: %w", err)
	}
	if err := validateTokenSet(component.Compatibility.PlatformRequirements, 128); err != nil {
		return fmt.Errorf("platform requirements: %w", err)
	}
	if err := validateExtensions(component.Extensions); err != nil {
		return err
	}
	normalizeComponent(component)
	return nil
}

func validateExtensions(extensions []Extension) error {
	seen := make(map[string]struct{}, len(extensions))
	for index := range extensions {
		extension := &extensions[index]
		if !safeToken(extension.ExtensionID, 256) || !safeVersion(extension.ExtensionVersion) || !taggedDigest(extension.PayloadDigest) {
			return fmt.Errorf("extension identity or digest is invalid")
		}
		key := extension.ExtensionID + "\x00" + extension.ExtensionVersion
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf("extension identity is duplicated")
		}
		seen[key] = struct{}{}
		digest, err := digestValue(extension.Payload)
		if err != nil || digest != extension.PayloadDigest {
			return fmt.Errorf("extension %d payload digest mismatch", index)
		}
	}
	return nil
}

func validateTokenSet(values []string, maximum int) error {
	if len(values) > maximum {
		return fmt.Errorf("collection exceeds %d entries", maximum)
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if !safeToken(value, 256) {
			return fmt.Errorf("token has invalid syntax")
		}
		if _, duplicate := seen[value]; duplicate {
			return fmt.Errorf("token is duplicated")
		}
		seen[value] = struct{}{}
	}
	return nil
}

func normalizeInput(input *ProfileInput) {
	sort.Strings(input.ConfiguredRoots)
	for index := range input.Components {
		normalizeComponent(&input.Components[index])
	}
	sort.Slice(input.Components, func(i, j int) bool { return canonicalLess(input.Components[i], input.Components[j]) })
	normalizeExtensions(input.Extensions)
}

func normalizeDesired(desired *DesiredState) {
	for index := range desired.Components {
		normalizeComponent(&desired.Components[index])
	}
	sort.Slice(desired.Components, func(i, j int) bool { return canonicalLess(desired.Components[i], desired.Components[j]) })
	normalizeExtensions(desired.Extensions)
}

func normalizeComponent(component *DesiredComponent) {
	sort.Slice(component.Dependencies, func(i, j int) bool { return canonicalLess(component.Dependencies[i], component.Dependencies[j]) })
	sort.Strings(component.Compatibility.RequiredCapabilities)
	sort.Strings(component.Compatibility.PlatformRequirements)
	normalizeExtensions(component.Extensions)
}

func normalizeExtensions(extensions []Extension) {
	sort.Slice(extensions, func(i, j int) bool { return canonicalLess(extensions[i], extensions[j]) })
}

func desiredDigest(desired DesiredState) (string, error) {
	desired.DesiredStateDigest = ""
	normalizeDesired(&desired)
	value, err := objectWithout(mustJSON(desired), "desired_state_digest")
	if err != nil {
		return "", err
	}
	return digestValue(value)
}

func profileDigest(profile Profile) (string, error) {
	profile.ProfileDigest = ""
	sort.Strings(profile.ConfiguredRoots)
	normalizeDesired(&profile.DesiredState)
	value, err := objectWithout(mustJSON(profile), "profile_digest")
	if err != nil {
		return "", err
	}
	return digestValue(value)
}

func sameIntent(current Profile, input ProfileInput) bool {
	currentIntent := ProfileInput{
		Protocol: ProfileInputProtocol, FormatVersion: 1, ProfileID: current.ProfileID,
		TOPSID: current.TOPSID, ConfiguredRoots: append([]string(nil), current.ConfiguredRoots...),
		BootMode: current.BootMode, Components: cloneComponents(current.DesiredState.Components),
		Extensions: cloneExtensions(current.DesiredState.Extensions),
	}
	normalizeInput(&currentIntent)
	normalizeInput(&input)
	left, leftErr := marshalCanonical(currentIntent)
	right, rightErr := marshalCanonical(input)
	return leftErr == nil && rightErr == nil && bytes.Equal(left, right)
}

func requireExpected(current Profile, exists bool, expected string) error {
	if expected == "absent" {
		if exists {
			return fmt.Errorf("lifecycle profile expected absent but current digest is %s", current.ProfileDigest)
		}
		return nil
	}
	if !exists {
		return fmt.Errorf("lifecycle profile expected %s but is absent", expected)
	}
	if current.ProfileDigest != expected {
		return fmt.Errorf("lifecycle profile expected %s but current digest is %s", expected, current.ProfileDigest)
	}
	return nil
}

func validateExpected(value string) error {
	if value != "absent" && !taggedDigest(value) {
		return fmt.Errorf("--expected-profile-digest must be absent or an exact tagged SHA-256 digest")
	}
	return nil
}

func encodeProfile(profile Profile) ([]byte, error) {
	encoded, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode lifecycle profile: %w", err)
	}
	if len(encoded)+1 > maxProfileBytes {
		return nil, fmt.Errorf("encoded lifecycle profile exceeds %d bytes", maxProfileBytes)
	}
	return append(encoded, '\n'), nil
}

func canonicalStateRoot(root string) (string, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) == string(os.PathSeparator) {
		return "", fmt.Errorf("lifecycle state root must be an absolute descendant path")
	}
	clean := filepath.Clean(root)
	if info, err := os.Lstat(clean); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return "", fmt.Errorf("lifecycle state root must be a no-follow directory")
		}
		resolved, err := filepath.EvalSymlinks(clean)
		if err != nil {
			return "", err
		}
		return filepath.Clean(resolved), nil
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("inspect lifecycle state root: %w", err)
	}
	return clean, nil
}

func profileFileName(profileID string) string {
	digest := sha256.Sum256([]byte("lifecycle-profile:" + profileID))
	return "profile-" + hex.EncodeToString(digest[:]) + ".json"
}

func decodeExact(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("JSON contains trailing data")
		}
		return err
	}
	return nil
}

func marshalCanonical(value any) ([]byte, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var normalized any
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	if err := decoder.Decode(&normalized); err != nil {
		return nil, err
	}
	return json.Marshal(normalized)
}

func digestValue(value any) (string, error) {
	encoded, err := marshalCanonical(value)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func objectWithout(data []byte, field string) (map[string]any, error) {
	var object map[string]any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&object); err != nil {
		return nil, err
	}
	delete(object, field)
	return object, nil
}

func mustJSON(value any) []byte {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}

func canonicalLess(left, right any) bool {
	l, _ := marshalCanonical(left)
	r, _ := marshalCanonical(right)
	return bytes.Compare(l, r) < 0
}

func cloneComponents(value []DesiredComponent) []DesiredComponent {
	encoded := mustJSON(value)
	var result []DesiredComponent
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	if err := decoder.Decode(&result); err != nil {
		panic(err)
	}
	return result
}

func cloneExtensions(value []Extension) []Extension {
	encoded := mustJSON(value)
	var result []Extension
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	if err := decoder.Decode(&result); err != nil {
		panic(err)
	}
	return result
}

func taggedDigest(value string) bool {
	if len(value) != 71 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	for _, character := range value[7:] {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}

func safeToken(value string, maximum int) bool {
	if value == "" || len(value) > maximum {
		return false
	}
	for _, character := range value {
		if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') || strings.ContainsRune("._:-", character) {
			continue
		}
		return false
	}
	return true
}

func safeVersion(value string) bool {
	if value == "" || len(value) > 64 || value == "." || value == ".." {
		return false
	}
	for _, character := range value {
		if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') || strings.ContainsRune(".+-", character) {
			continue
		}
		return false
	}
	return true
}

func safeAbsolutePath(value string) bool {
	if value == "" || len(value) > 4096 || !filepath.IsAbs(value) || filepath.Clean(value) != value ||
		strings.Contains(value, "\\") || (len(value) > 1 && strings.HasSuffix(value, "/")) {
		return false
	}
	if value == string(os.PathSeparator) {
		return true
	}
	for _, part := range strings.Split(strings.TrimPrefix(value, "/"), "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
		for _, character := range part {
			if character < 0x20 || character == 0x7f {
				return false
			}
		}
	}
	return true
}

func validTOPSID(value string) bool {
	if len(value) != 36 || strings.ToLower(value) != value || value == "00000000-0000-0000-0000-000000000000" {
		return false
	}
	for index, character := range value {
		switch index {
		case 8, 13, 18, 23:
			if character != '-' {
				return false
			}
		default:
			if !((character >= '0' && character <= '9') || (character >= 'a' && character <= 'f')) {
				return false
			}
		}
	}
	return value[14] >= '1' && value[14] <= '8' && strings.Contains("89ab", value[19:20])
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func stringPointer(value string) *string { return &value }

// Timestamp is injectable by tests that build observation evidence.
var Timestamp = func() time.Time { return time.Now().UTC().Truncate(time.Second) }
