package knowledgebinding

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
)

const (
	ProtocolV1 = "symphony.knowledge.engine-binding-registry.v1"
	ProtocolV2 = "symphony.knowledge.engine-binding-registry.v2"

	// Protocol is the protocol emitted by new registries. ProtocolV1 remains a
	// separate, immutable read-compatibility surface.
	Protocol  = ProtocolV2
	Scope     = "user"
	ProfileID = "default"

	CompatibilityAbsent                    = "absent"
	CompatibilityCurrent                   = "current"
	CompatibilityV1                        = "v1_compatible"
	CompatibilityLegacyV1MigrationRequired = "legacy_v1_migration_required"
	CompatibilityUnsupportedRolePreserved  = "unsupported_role_preserved"

	maxV2Bindings = 256
)

type roleIdentity struct {
	moduleID string
	engineID string
}

var supportedRoles = map[string]roleIdentity{
	"coordinator": {moduleID: "knowledge-session-coordinator", engineID: "symphony-knowledge-session"},
	"sacv":        {moduleID: "sacv-engine", engineID: "symphony-sacv"},
	"sclv":        {moduleID: "sclv-engine", engineID: "symphony-sclv"},
	"skvi":        {moduleID: "skvi-engine", engineID: "symphony-skvi"},
	"sodv":        {moduleID: "sodv-engine", engineID: "symphony-sodv"},
	"sav":         {moduleID: "sav-engine", engineID: "symphony-sav"},
	"sev":         {moduleID: "sev-engine", engineID: "symphony-sev"},
	"ssfv":        {moduleID: "ssfv-engine", engineID: "symphony-ssfv"},
}

var v1Roles = map[string]roleIdentity{
	"coordinator": {moduleID: "knowledge-session-coordinator", engineID: "symphony-knowledge-session"},
	"sacv":        {moduleID: "sacv-engine", engineID: "symphony-sacv"},
	"sclv":        {moduleID: "sclv-engine", engineID: "symphony-sclv"},
	"skvi":        {moduleID: "skvi-engine", engineID: "symphony-skvi"},
	"sodv":        {moduleID: "sodv-engine", engineID: "symphony-sodv"},
	"ssfv":        {moduleID: "ssfv-engine", engineID: "symphony-ssfv"},
}

// legacyV1Roles is deliberately frozen independently of supportedRoles. It is
// the only nonconforming v1 shape emitted by affected historical qxctl builds;
// adding a future v2 role must never widen this migration adapter.
var legacyV1Roles = map[string]roleIdentity{
	"coordinator": {moduleID: "knowledge-session-coordinator", engineID: "symphony-knowledge-session"},
	"sacv":        {moduleID: "sacv-engine", engineID: "symphony-sacv"},
	"sclv":        {moduleID: "sclv-engine", engineID: "symphony-sclv"},
	"skvi":        {moduleID: "skvi-engine", engineID: "symphony-skvi"},
	"sodv":        {moduleID: "sodv-engine", engineID: "symphony-sodv"},
	"sav":         {moduleID: "sav-engine", engineID: "symphony-sav"},
	"sev":         {moduleID: "sev-engine", engineID: "symphony-sev"},
	"ssfv":        {moduleID: "ssfv-engine", engineID: "symphony-ssfv"},
}

type Binding struct {
	Role             string  `json:"role"`
	ModuleID         string  `json:"module_id"`
	EngineID         string  `json:"engine_id"`
	Version          string  `json:"version"`
	Prefix           string  `json:"prefix"`
	ReceiptPath      string  `json:"receipt_path"`
	ReceiptDigest    string  `json:"receipt_digest"`
	ExecutablePath   string  `json:"executable_path"`
	ExecutableDigest string  `json:"executable_digest"`
	State            string  `json:"state"`
	DefaultReceptor  *string `json:"default_receptor"`
}

type Registry struct {
	Protocol                 string    `json:"protocol"`
	FormatVersion            uint64    `json:"format_version,omitempty"`
	Scope                    string    `json:"scope"`
	ProfileID                string    `json:"profile_id"`
	Generation               uint64    `json:"generation"`
	UpdatedAt                string    `json:"updated_at"`
	PreviousRegistryDigest   *string   `json:"previous_registry_digest"`
	PreviousRegistryProtocol string    `json:"previous_registry_protocol,omitempty"`
	Bindings                 []Binding `json:"bindings"`
	Canonical                bool      `json:"canonical"`
	RegistryDigest           string    `json:"registry_digest"`
}

type Snapshot struct {
	Exists        bool     `json:"exists"`
	Compatibility string   `json:"compatibility"`
	Registry      Registry `json:"registry"`
}

type DoctorResult struct {
	Role    string `json:"role"`
	Healthy bool   `json:"healthy"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type DoctorReport struct {
	Protocol       string         `json:"protocol"`
	RegistryExists bool           `json:"registry_exists"`
	RegistryDigest *string        `json:"registry_digest"`
	Healthy        bool           `json:"healthy"`
	Results        []DoctorResult `json:"results"`
}

type Store struct {
	stateRoot string
	now       func() time.Time
}

// StateRoot is the canonical root used by this store and by bound
// user-scope operational state.
func (s *Store) StateRoot() string {
	return s.stateRoot
}

func NewStore(stateRoot string) (*Store, error) {
	if stateRoot == "" {
		var err error
		stateRoot, err = DefaultStateRoot()
		if err != nil {
			return nil, err
		}
	}
	if !filepath.IsAbs(stateRoot) {
		return nil, fmt.Errorf("knowledge binding state root must be absolute")
	}
	canonicalRoot, err := canonicalStateRoot(stateRoot)
	if err != nil {
		return nil, err
	}
	return &Store{
		stateRoot: canonicalRoot,
		now:       func() time.Time { return time.Now().UTC() },
	}, nil
}

func canonicalStateRoot(root string) (string, error) {
	clean := filepath.Clean(root)
	if filepath.Dir(clean) == clean {
		return "", fmt.Errorf("knowledge binding state root must be a descendant directory")
	}
	if info, err := os.Lstat(clean); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("knowledge binding state root must not be a symbolic link")
		}
		if !info.IsDir() {
			return "", fmt.Errorf("knowledge binding state root must be a directory")
		}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("inspect knowledge binding state root: %w", err)
	}
	existing := clean
	missing := make([]string, 0)
	for {
		info, err := os.Lstat(existing)
		if err == nil {
			if !info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
				return "", fmt.Errorf("knowledge binding state ancestor is not a directory")
			}
			break
		}
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("inspect knowledge binding state ancestor: %w", err)
		}
		parent := filepath.Dir(existing)
		if parent == existing {
			return "", fmt.Errorf("knowledge binding state root has no existing ancestor")
		}
		missing = append(missing, filepath.Base(existing))
		existing = parent
	}
	resolved, err := filepath.EvalSymlinks(existing)
	if err != nil {
		return "", fmt.Errorf("canonicalize knowledge binding state ancestor: %w", err)
	}
	for index := len(missing) - 1; index >= 0; index-- {
		resolved = filepath.Join(resolved, missing[index])
	}
	return filepath.Clean(resolved), nil
}

func DefaultStateRoot() (string, error) {
	if configured := os.Getenv("XDG_STATE_HOME"); configured != "" {
		if !filepath.IsAbs(configured) {
			return "", fmt.Errorf("XDG_STATE_HOME must be absolute")
		}
		return filepath.Clean(configured), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user state home: %w", err)
	}
	return filepath.Join(home, ".local", "state"), nil
}

func Roles() []string {
	roles := make([]string, 0, len(supportedRoles))
	for role := range supportedRoles {
		roles = append(roles, role)
	}
	sort.Strings(roles)
	return roles
}

func (s *Store) Snapshot() (Snapshot, error) {
	snapshot, err := s.AdministrativeSnapshot()
	if err != nil || !snapshot.Exists {
		return snapshot, err
	}
	if err := requireOperationalCompatibility(snapshot.Compatibility); err != nil {
		return Snapshot{}, err
	}
	return snapshot, nil
}

// AdministrativeSnapshot reads every structurally valid supported registry
// representation. It is used by list, inspect, doctor, and explicit migration;
// operational consumers use Snapshot so a partially understood registry cannot
// be acted upon.
func (s *Store) AdministrativeSnapshot() (Snapshot, error) {
	var snapshot Snapshot
	err := s.withStateLock(false, func(directory *os.File) error {
		registry, exists, compatibility, err := readRegistry(directory)
		if err != nil {
			return err
		}
		snapshot = Snapshot{Exists: exists, Compatibility: compatibility, Registry: registry}
		return nil
	})
	return snapshot, err
}

func (s *Store) Bind(role, prefix, version, expected string) (Registry, bool, error) {
	if _, ok := supportedRoles[role]; !ok {
		return Registry{}, false, fmt.Errorf("unsupported knowledge engine role %q", role)
	}
	if !validExpectedDigest(expected) {
		return Registry{}, false, fmt.Errorf("--expected-registry-digest must be absent or an exact tagged SHA-256 digest")
	}
	var result Registry
	changed := false
	err := s.withStateLock(true, func(directory *os.File) error {
		current, exists, compatibility, err := readRegistry(directory)
		if err != nil {
			return err
		}
		if err := requireExpected(current, exists, expected); err != nil {
			return err
		}
		if exists {
			if err := requireCurrentWritableCompatibility(compatibility); err != nil {
				return err
			}
		}
		installed, err := knowledgeengine.InspectInstallation(role, prefix, version)
		if err != nil {
			return fmt.Errorf("inspect exact knowledge engine installation: %w", err)
		}
		nextBinding := Binding{
			Role:             installed.Role,
			ModuleID:         installed.ModuleID,
			EngineID:         installed.EngineID,
			Version:          installed.Version,
			Prefix:           installed.Prefix,
			ReceiptPath:      installed.ReceiptPath,
			ReceiptDigest:    installed.ReceiptDigest,
			ExecutablePath:   installed.ExecutablePath,
			ExecutableDigest: installed.ExecutableDigest,
			State:            "bound_undocked",
			DefaultReceptor:  nil,
		}
		if exists {
			for _, binding := range current.Bindings {
				if binding.Role == role && binding == nextBinding {
					result = current
					return nil
				}
			}
		}
		next := nextRegistry(current, exists, s.now())
		replaced := false
		for index := range next.Bindings {
			if next.Bindings[index].Role == role {
				next.Bindings[index] = nextBinding
				replaced = true
				break
			}
		}
		if !replaced {
			next.Bindings = append(next.Bindings, nextBinding)
		}
		sort.Slice(next.Bindings, func(i, j int) bool {
			return next.Bindings[i].Role < next.Bindings[j].Role
		})
		if err := finalizeRegistry(&next); err != nil {
			return err
		}
		encoded, err := encodeRegistry(next)
		if err != nil {
			return err
		}
		if err := writeRegistry(directory, encoded); err != nil {
			return err
		}
		result = next
		changed = true
		return nil
	})
	return result, changed, err
}

func (s *Store) Unbind(role, expected string) (Registry, bool, error) {
	if _, ok := supportedRoles[role]; !ok {
		return Registry{}, false, fmt.Errorf("unsupported knowledge engine role %q", role)
	}
	if !validExpectedDigest(expected) {
		return Registry{}, false, fmt.Errorf("--expected-registry-digest must be absent or an exact tagged SHA-256 digest")
	}
	var result Registry
	changed := false
	err := s.withStateLock(true, func(directory *os.File) error {
		current, exists, compatibility, err := readRegistry(directory)
		if err != nil {
			return err
		}
		if err := requireExpected(current, exists, expected); err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("knowledge engine binding registry is absent")
		}
		if err := requireCurrentWritableCompatibility(compatibility); err != nil {
			return err
		}
		nextBindings := make([]Binding, 0, len(current.Bindings))
		for _, binding := range current.Bindings {
			if binding.Role == role {
				changed = true
				continue
			}
			nextBindings = append(nextBindings, binding)
		}
		if !changed {
			result = current
			return nil
		}
		next := nextRegistry(current, true, s.now())
		next.Bindings = nextBindings
		if err := finalizeRegistry(&next); err != nil {
			return err
		}
		encoded, err := encodeRegistry(next)
		if err != nil {
			return err
		}
		if err := writeRegistry(directory, encoded); err != nil {
			return err
		}
		result = next
		return nil
	})
	return result, changed, err
}

// Migrate rewrites one exact v1 generation as a distinct v2 generation. It
// preserves every binding byte value represented by the model and never
// inspects installations or selects a version.
func (s *Store) Migrate(expected string) (Registry, bool, error) {
	if !taggedDigest(expected) {
		return Registry{}, false, fmt.Errorf("--expected-registry-digest must be an exact tagged SHA-256 digest")
	}
	var result Registry
	changed := false
	err := s.withStateLock(true, func(directory *os.File) error {
		current, exists, compatibility, err := readRegistry(directory)
		if err != nil {
			return err
		}
		if err := requireExpected(current, exists, expected); err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("knowledge engine binding registry is absent")
		}
		switch compatibility {
		case CompatibilityCurrent:
			result = current
			return nil
		case CompatibilityV1, CompatibilityLegacyV1MigrationRequired:
			// Continue with the explicit write-forward generation below.
		default:
			return requireOperationalCompatibility(compatibility)
		}
		next := nextRegistry(current, true, s.now())
		if err := finalizeRegistry(&next); err != nil {
			return err
		}
		encoded, err := encodeRegistry(next)
		if err != nil {
			return err
		}
		if err := writeRegistry(directory, encoded); err != nil {
			return err
		}
		result = next
		changed = true
		return nil
	})
	return result, changed, err
}

func (s *Store) Doctor() (DoctorReport, error) {
	snapshot, err := s.AdministrativeSnapshot()
	if err != nil {
		return DoctorReport{}, err
	}
	report := DoctorReport{
		Protocol:       "symphony.knowledge.engine-binding-doctor.v1",
		RegistryExists: snapshot.Exists,
		Healthy:        true,
		Results:        make([]DoctorResult, 0),
	}
	if !snapshot.Exists {
		report.Healthy = false
		report.Results = append(report.Results, DoctorResult{
			Role: "registry", Healthy: false, Code: "binding.registry_absent",
			Message: "user-default knowledge engine binding registry is absent",
		})
		return report, nil
	}
	report.RegistryDigest = &snapshot.Registry.RegistryDigest
	if compatibilityErr := requireOperationalCompatibility(snapshot.Compatibility); compatibilityErr != nil {
		report.Healthy = false
		report.Results = append(report.Results, DoctorResult{
			Role: "registry", Healthy: false, Code: "binding.registry_incompatible",
			Message: compatibilityErr.Error(),
		})
	}
	for _, binding := range snapshot.Registry.Bindings {
		if _, supported := supportedRoles[binding.Role]; !supported {
			report.Healthy = false
			report.Results = append(report.Results, DoctorResult{
				Role: binding.Role, Healthy: false, Code: "binding.role_unsupported",
				Message: "binding role is preserved but unsupported by this qxctl version",
			})
			continue
		}
		installed, inspectErr := knowledgeengine.InspectInstallation(
			binding.Role, binding.Prefix, binding.Version)
		result := DoctorResult{Role: binding.Role, Healthy: true, Code: "binding.healthy", Message: "exact installation matches binding"}
		if inspectErr != nil {
			result.Healthy = false
			result.Code = "binding.installation_invalid"
			result.Message = inspectErr.Error()
		} else if installed.ModuleID != binding.ModuleID || installed.EngineID != binding.EngineID ||
			installed.ReceiptPath != binding.ReceiptPath || installed.ReceiptDigest != binding.ReceiptDigest ||
			installed.ExecutablePath != binding.ExecutablePath ||
			installed.ExecutableDigest != binding.ExecutableDigest {
			result.Healthy = false
			result.Code = "binding.content_mismatch"
			result.Message = "installed receipt or executable no longer matches the bound content digest"
		}
		if !result.Healthy {
			report.Healthy = false
		}
		report.Results = append(report.Results, result)
	}
	return report, nil
}

func nextRegistry(current Registry, exists bool, now time.Time) Registry {
	next := Registry{
		Protocol:                 ProtocolV2,
		FormatVersion:            2,
		Scope:                    Scope,
		ProfileID:                ProfileID,
		Generation:               1,
		UpdatedAt:                now.UTC().Truncate(time.Second).Format(time.RFC3339),
		PreviousRegistryProtocol: "absent",
		Bindings:                 make([]Binding, 0),
		Canonical:                false,
	}
	if exists {
		previous := current.RegistryDigest
		next.Generation = current.Generation + 1
		next.PreviousRegistryDigest = &previous
		next.PreviousRegistryProtocol = current.Protocol
		next.Bindings = append(next.Bindings, current.Bindings...)
	}
	return next
}

func requireExpected(current Registry, exists bool, expected string) error {
	if expected == "absent" {
		if exists {
			return fmt.Errorf("binding registry expected absent but current digest is %s", current.RegistryDigest)
		}
		return nil
	}
	if !exists {
		return fmt.Errorf("binding registry expected %s but is absent", expected)
	}
	if current.RegistryDigest != expected {
		return fmt.Errorf("binding registry expected %s but current digest is %s", expected, current.RegistryDigest)
	}
	return nil
}

func validExpectedDigest(value string) bool {
	return value == "absent" || taggedDigest(value)
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

func finalizeRegistry(registry *Registry) error {
	digest, err := calculateDigest(*registry)
	if err != nil {
		return err
	}
	registry.RegistryDigest = digest
	return validateRegistry(*registry)
}

func calculateDigest(registry Registry) (string, error) {
	bindings := make([]any, 0, len(registry.Bindings))
	for _, binding := range registry.Bindings {
		bindings = append(bindings, map[string]any{
			"role":              binding.Role,
			"module_id":         binding.ModuleID,
			"engine_id":         binding.EngineID,
			"version":           binding.Version,
			"prefix":            binding.Prefix,
			"receipt_path":      binding.ReceiptPath,
			"receipt_digest":    binding.ReceiptDigest,
			"executable_path":   binding.ExecutablePath,
			"executable_digest": binding.ExecutableDigest,
			"state":             binding.State,
			"default_receptor":  binding.DefaultReceptor,
		})
	}
	input := map[string]any{
		"protocol":                 registry.Protocol,
		"scope":                    registry.Scope,
		"profile_id":               registry.ProfileID,
		"generation":               registry.Generation,
		"updated_at":               registry.UpdatedAt,
		"previous_registry_digest": registry.PreviousRegistryDigest,
		"bindings":                 bindings,
		"canonical":                registry.Canonical,
	}
	switch registry.Protocol {
	case ProtocolV1:
		// The immutable v1 digest preimage never contained format_version.
	case ProtocolV2:
		input["format_version"] = registry.FormatVersion
		input["previous_registry_protocol"] = registry.PreviousRegistryProtocol
	default:
		return "", fmt.Errorf("unsupported binding registry protocol %q", registry.Protocol)
	}
	encoded, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("encode binding registry digest input: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func encodeRegistry(registry Registry) ([]byte, error) {
	encoded, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode binding registry: %w", err)
	}
	return append(encoded, '\n'), nil
}

func readRegistry(directory *os.File) (Registry, bool, string, error) {
	data, exists, err := readRegistryFile(directory)
	if err != nil || !exists {
		return Registry{}, exists, CompatibilityAbsent, err
	}
	if len(data) == 0 || len(data) > 1024*1024 || !json.Valid(data) {
		return Registry{}, false, "", fmt.Errorf("binding registry is empty, oversized, or invalid JSON")
	}
	if err := knowledgeengine.ValidateJSONObject(data, 1024*1024); err != nil {
		return Registry{}, false, "", fmt.Errorf("binding registry violates bounded JSON rules: %w", err)
	}
	var registry Registry
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&registry); err != nil {
		return Registry{}, false, "", fmt.Errorf("decode binding registry: %w", err)
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return Registry{}, false, "", fmt.Errorf("binding registry contains trailing data")
		}
		return Registry{}, false, "", fmt.Errorf("decode binding registry trailing data: %w", err)
	}
	if err := validateVersionedFieldPresence(data, registry.Protocol); err != nil {
		return Registry{}, false, "", err
	}
	compatibility, err := registryCompatibility(registry)
	if err != nil {
		return Registry{}, false, "", err
	}
	return registry, true, compatibility, nil
}

func validateVersionedFieldPresence(data []byte, protocol string) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return fmt.Errorf("decode binding registry fields: %w", err)
	}
	_, hasFormatVersion := fields["format_version"]
	_, hasPreviousProtocol := fields["previous_registry_protocol"]
	switch protocol {
	case ProtocolV1:
		if hasFormatVersion || hasPreviousProtocol {
			return fmt.Errorf("binding registry v1 contains v2-only fields")
		}
	case ProtocolV2:
		if !hasFormatVersion || !hasPreviousProtocol {
			return fmt.Errorf("binding registry v2 lacks required version or lineage fields")
		}
	default:
		return fmt.Errorf("binding registry protocol is unsupported")
	}
	return nil
}

func validateRegistry(registry Registry) error {
	compatibility, err := registryCompatibility(registry)
	if err == nil && compatibility == CompatibilityLegacyV1MigrationRequired {
		return fmt.Errorf("binding registry is a recognized legacy nonconforming v1 representation")
	}
	return err
}

func registryCompatibility(registry Registry) (string, error) {
	if registry.Scope != Scope || registry.ProfileID != ProfileID ||
		registry.Generation == 0 || registry.Generation > 9007199254740991 ||
		registry.Canonical || registry.UpdatedAt == "" ||
		!taggedDigest(registry.RegistryDigest) {
		return "", fmt.Errorf("binding registry identity or lifecycle fields are invalid")
	}
	if (registry.Protocol == ProtocolV1 && registry.FormatVersion != 0) ||
		(registry.Protocol == ProtocolV2 && registry.FormatVersion != 2) ||
		(registry.Protocol != ProtocolV1 && registry.Protocol != ProtocolV2) {
		return "", fmt.Errorf("binding registry protocol or format version is unsupported")
	}
	if registry.Protocol == ProtocolV1 && registry.PreviousRegistryProtocol != "" {
		return "", fmt.Errorf("binding registry v1 contains a v2 lineage field")
	}
	if _, err := time.Parse(time.RFC3339Nano, registry.UpdatedAt); err != nil ||
		!strings.HasSuffix(registry.UpdatedAt, "Z") {
		return "", fmt.Errorf("binding registry timestamp is invalid")
	}
	if registry.Generation == 1 && registry.PreviousRegistryDigest != nil {
		return "", fmt.Errorf("first binding registry generation has a previous digest")
	}
	if registry.Generation > 1 &&
		(registry.PreviousRegistryDigest == nil || !taggedDigest(*registry.PreviousRegistryDigest)) {
		return "", fmt.Errorf("later binding registry generation lacks a previous digest")
	}
	if registry.Protocol == ProtocolV2 {
		if (registry.Generation == 1 && registry.PreviousRegistryProtocol != "absent") ||
			(registry.Generation > 1 && registry.PreviousRegistryProtocol != ProtocolV1 &&
				registry.PreviousRegistryProtocol != ProtocolV2) {
			return "", fmt.Errorf("binding registry v2 has invalid predecessor protocol evidence")
		}
	}

	maxBindings := maxV2Bindings
	allowUnknownRoles := registry.Protocol == ProtocolV2
	compatibility := CompatibilityCurrent
	if registry.Protocol == ProtocolV1 {
		maxBindings = len(legacyV1Roles)
		compatibility = CompatibilityV1
	}
	seen := make(map[string]struct{}, len(registry.Bindings))
	if len(registry.Bindings) > maxBindings {
		return "", fmt.Errorf("binding registry exceeds the role bound")
	}
	previousRole := ""
	for _, binding := range registry.Bindings {
		identity, ok := supportedRoles[binding.Role]
		if registry.Protocol == ProtocolV1 {
			identity, ok = legacyV1Roles[binding.Role]
		}
		if !ok && (!allowUnknownRoles || !validToken(binding.Role)) {
			return "", fmt.Errorf("binding registry contains unsupported role %q", binding.Role)
		}
		if _, duplicate := seen[binding.Role]; duplicate || binding.Role <= previousRole {
			return "", fmt.Errorf("binding registry roles are duplicate or unsorted")
		}
		seen[binding.Role] = struct{}{}
		previousRole = binding.Role
		if (ok && (binding.ModuleID != identity.moduleID || binding.EngineID != identity.engineID)) ||
			(!ok && (!validToken(binding.ModuleID) || !validToken(binding.EngineID))) ||
			!knowledgeengine.ValidVersion(binding.Version) ||
			!safeAbsolutePath(binding.Prefix) || !safeAbsolutePath(binding.ReceiptPath) ||
			!safeAbsolutePath(binding.ExecutablePath) || !pathWithin(binding.Prefix, binding.ReceiptPath) ||
			!pathWithin(binding.Prefix, binding.ExecutablePath) || !taggedDigest(binding.ReceiptDigest) ||
			!taggedDigest(binding.ExecutableDigest) || binding.State != "bound_undocked" ||
			binding.DefaultReceptor != nil {
			return "", fmt.Errorf("binding %q has invalid identity, path, digest, or lifecycle state", binding.Role)
		}
		if registry.Protocol == ProtocolV1 {
			if _, canonicalV1Role := v1Roles[binding.Role]; !canonicalV1Role {
				compatibility = CompatibilityLegacyV1MigrationRequired
			}
		} else if !ok {
			compatibility = CompatibilityUnsupportedRolePreserved
		}
	}
	calculated, err := calculateDigest(registry)
	if err != nil {
		return "", err
	}
	if calculated != registry.RegistryDigest {
		return "", fmt.Errorf("binding registry digest mismatch")
	}
	return compatibility, nil
}

func requireCurrentWritableCompatibility(compatibility string) error {
	if compatibility == CompatibilityCurrent {
		return nil
	}
	if compatibility == CompatibilityV1 || compatibility == CompatibilityLegacyV1MigrationRequired {
		return fmt.Errorf("binding registry requires explicit `qxctl knowledge engines migrate --expected-registry-digest DIGEST` before mutation")
	}
	return requireOperationalCompatibility(compatibility)
}

func requireOperationalCompatibility(compatibility string) error {
	switch compatibility {
	case CompatibilityCurrent, CompatibilityV1:
		return nil
	case CompatibilityLegacyV1MigrationRequired:
		return fmt.Errorf("legacy nonconforming v1 binding registry requires explicit migration")
	case CompatibilityUnsupportedRolePreserved:
		return fmt.Errorf("binding registry contains a preserved role unsupported by this qxctl version")
	default:
		return fmt.Errorf("binding registry compatibility is unsupported")
	}
}

func validToken(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for _, character := range value {
		if (character < 'a' || character > 'z') && (character < 'A' || character > 'Z') &&
			(character < '0' || character > '9') && character != '.' && character != '_' &&
			character != ':' && character != '-' {
			return false
		}
	}
	return true
}

func safeAbsolutePath(value string) bool {
	if value == "" || len(value) > 4096 || !filepath.IsAbs(value) || filepath.Clean(value) != value ||
		strings.ContainsRune(value, '\x00') || strings.Contains(value, "\\") {
		return false
	}
	for _, character := range value {
		if character < 0x20 || character == 0x7f {
			return false
		}
	}
	return true
}

func pathWithin(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != "." && relative != ".." &&
		!strings.HasPrefix(relative, ".."+string(os.PathSeparator)) &&
		!filepath.IsAbs(relative)
}
