// Package commandregistry binds stable qxctl command identities to the Cobra
// commands that implement them and emits the observed machine manifest.
package commandregistry

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	Protocol      = "symphony.qxctl.command-registry.v1"
	FormatVersion = 1

	roleAnnotation              = "symphony.qxctl.command-role"
	specAnnotation              = "symphony.qxctl.command-spec"
	structuralHandlerAnnotation = "symphony.qxctl.structural-handler"
	retiredHandlerAnnotation    = "symphony.qxctl.retired-handler"
	roleExecutable              = "executable"
	roleRetired                 = "retired"
	roleStructural              = "structural"
	roleProhibited              = "prohibited"
	roleInternal                = "internal"
	maxObservedExecutableSize   = 512 << 20
	registryDigestAlgorithmName = "sha256"
)

// FeatureBinding identifies the exact registered feature interaction served by
// a command. It does not confer runtime availability or authorization.
type FeatureBinding struct {
	FeatureID   string `json:"feature_id"`
	Interaction string `json:"interaction"`
}

// CommandSpec is the stable semantic portion of a public executable command.
// Grammar, aliases, visibility, and JSON support are deliberately derived from
// the same attached Cobra command rather than copied into another registry.
type CommandSpec struct {
	CommandID                 string
	Status                    string
	IntroducedIn              string
	DeprecatedIn              *string
	ReplacementIDs            []string
	FeatureBindings           []FeatureBinding
	InfrastructurePurpose     *string
	BackendOperationIDs       []string
	Mutability                string
	AuthorityMode             string
	TargetScope               string
	InputProtocols            []string
	OutputProtocols           []string
	ResultValidationProtocols []string
	RecoveryCommandID         *string
	Noninteractive            bool
	JSONOutput                *bool
}

// CommandRecord is the deterministic wire projection of an attached command.
type CommandRecord struct {
	CommandID                 string           `json:"command_id"`
	Status                    string           `json:"status"`
	IntroducedIn              string           `json:"introduced_in"`
	DeprecatedIn              *string          `json:"deprecated_in"`
	ReplacementIDs            []string         `json:"replacement_ids"`
	Grammar                   *string          `json:"grammar"`
	Aliases                   []string         `json:"aliases"`
	Visibility                string           `json:"visibility"`
	FeatureBindings           []FeatureBinding `json:"feature_bindings"`
	InfrastructurePurpose     *string          `json:"infrastructure_purpose"`
	BackendOperationIDs       []string         `json:"backend_operation_ids"`
	Mutability                string           `json:"mutability"`
	AuthorityMode             string           `json:"authority_mode"`
	TargetScope               string           `json:"target_scope"`
	InputProtocols            []string         `json:"input_protocols"`
	OutputProtocols           []string         `json:"output_protocols"`
	ResultValidationProtocols []string         `json:"result_validation_protocols"`
	RecoveryCommandID         *string          `json:"recovery_command_id"`
	Noninteractive            bool             `json:"noninteractive"`
	JSONOutput                bool             `json:"json_output"`
}

// Manifest is the complete observed qxctl command registry.
type Manifest struct {
	Protocol         string          `json:"protocol"`
	FormatVersion    int             `json:"format_version"`
	RegistryKind     string          `json:"registry_kind"`
	ClientID         string          `json:"client_id"`
	ClientVersion    *string         `json:"client_version"`
	ClientTrust      string          `json:"client_trust"`
	ExecutableDigest *string         `json:"executable_digest"`
	ReceiptDigest    *string         `json:"receipt_digest"`
	Commands         []CommandRecord `json:"commands"`
	RegistryDigest   string          `json:"registry_digest,omitempty"`
}

// Identity binds the observed registry to one exact executable and optional
// installation receipt. A nil receipt denotes an unreceipted client.
type Identity struct {
	ClientVersion    string
	ExecutableDigest string
	ReceiptDigest    *string
}

// VerifyDigest recomputes the manifest registry digest using the canonical
// omission rule and rejects a mismatch.
func VerifyDigest(manifest Manifest) error {
	provided := manifest.RegistryDigest
	if !validTaggedDigest(provided) {
		return fmt.Errorf("registry digest must be a tagged SHA-256 digest")
	}
	manifest.RegistryDigest = ""
	canonical, err := canonicalJSON(manifest)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(canonical)
	want := registryDigestAlgorithmName + ":" + hex.EncodeToString(hash[:])
	if provided != want {
		return fmt.Errorf("registry digest mismatch")
	}
	return nil
}

// Attach binds one stable command specification to the exact Cobra command
// that implements it. Validation remains deferred until the complete tree is
// assembled so command paths and flags are final.
func Attach(command *cobra.Command, spec CommandSpec) *cobra.Command {
	if command.Annotations == nil {
		command.Annotations = make(map[string]string)
	}
	command.Annotations[roleAnnotation] = roleExecutable
	encoded, err := json.Marshal(spec)
	if err != nil {
		command.Annotations[specAnnotation] = "!invalid:" + err.Error()
	} else {
		command.Annotations[specAnnotation] = string(encoded)
	}
	return command
}

// Spec returns the semantic specification already attached to command.
func Spec(command *cobra.Command) (CommandSpec, error) {
	if command == nil || (command.Annotations[roleAnnotation] != roleExecutable && command.Annotations[roleAnnotation] != roleRetired) {
		return CommandSpec{}, fmt.Errorf("command is not a registered command identity")
	}
	return decodeSpec(command.Annotations[specAnnotation])
}

// Retired constructs a hidden, fail-closed compatibility tombstone. The old
// path remains capable only of returning a deterministic retirement diagnostic;
// its manifest record retains the stable ID with null grammar so no caller can
// mistake the path for supported executable grammar or reuse the identity.
func Retired(use string, spec CommandSpec) *cobra.Command {
	command := &cobra.Command{Use: use, Hidden: true, DisableFlagParsing: true}
	command.RunE = retiredHandler(spec.CommandID, spec.ReplacementIDs)
	command.Annotations = make(map[string]string)
	command.Annotations[roleAnnotation] = roleRetired
	command.Annotations[retiredHandlerAnnotation] = strconv.FormatUint(uint64(reflect.ValueOf(command.RunE).Pointer()), 16)
	encoded, err := json.Marshal(spec)
	if err != nil {
		command.Annotations[specAnnotation] = "!invalid:" + err.Error()
	} else {
		command.Annotations[specAnnotation] = string(encoded)
	}
	return command
}

// Structural constructs a namespace-only command. Its handler identity is
// checked during parity validation, so replacing it with executable behavior
// without assigning a CommandSpec fails closed.
func Structural(use string, args cobra.PositionalArgs, result error) *cobra.Command {
	command := &cobra.Command{Use: use, Args: args, RunE: structuralHandler(result)}
	if command.Annotations == nil {
		command.Annotations = make(map[string]string)
	}
	command.Annotations[roleAnnotation] = roleStructural
	command.Annotations[structuralHandlerAnnotation] = strconv.FormatUint(uint64(reflect.ValueOf(command.RunE).Pointer()), 16)
	return command
}

// Prohibited marks an intentionally unavailable hidden grammar reservation.
// It remains executable only to provide its fail-closed diagnostic and never
// satisfies feature-administration coverage.
func Prohibited(command *cobra.Command) *cobra.Command {
	if command.Annotations == nil {
		command.Annotations = make(map[string]string)
	}
	command.Hidden = true
	command.Annotations[roleAnnotation] = roleProhibited
	return command
}

// Internal marks hidden Cobra plumbing that is neither public grammar nor an
// architectural prohibition.
func Internal(command *cobra.Command) *cobra.Command {
	if command.Annotations == nil {
		command.Annotations = make(map[string]string)
	}
	command.Hidden = true
	command.Annotations[roleAnnotation] = roleInternal
	return command
}

func structuralHandler(result error) func(*cobra.Command, []string) error {
	return func(*cobra.Command, []string) error { return result }
}

func retiredHandler(commandID string, replacements []string) func(*cobra.Command, []string) error {
	stableReplacements := sortedCopy(replacements)
	return func(*cobra.Command, []string) error {
		if len(stableReplacements) == 0 {
			return fmt.Errorf("command %s is retired and has no replacement", commandID)
		}
		return fmt.Errorf("command %s is retired; replacement IDs: %s", commandID, strings.Join(stableReplacements, ","))
	}
}

// Validate checks one complete Cobra tree for registration parity without
// reading prose, help output, configuration files, or the environment.
func Validate(root *cobra.Command) error {
	_, err := commandRecords(root)
	return err
}

// Build constructs and digest-binds the observed manifest. RegistryDigest is a
// tagged SHA-256 of compact recursively/lexicographically key-sorted UTF-8 JSON
// for the complete object with registry_digest omitted.
func Build(root *cobra.Command, identity Identity) (Manifest, error) {
	commands, err := commandRecords(root)
	if err != nil {
		return Manifest{}, err
	}
	if !validTaggedDigest(identity.ExecutableDigest) {
		return Manifest{}, fmt.Errorf("executable digest must be a tagged SHA-256 digest")
	}
	clientTrust := "unreceipted"
	if identity.ReceiptDigest != nil {
		if !validTaggedDigest(*identity.ReceiptDigest) {
			return Manifest{}, fmt.Errorf("receipt digest must be a tagged SHA-256 digest")
		}
		clientTrust = "receipted"
	}
	manifest := Manifest{
		Protocol: Protocol, FormatVersion: FormatVersion, RegistryKind: "observed",
		ClientID: "qxctl", ClientVersion: stringPointer(identity.ClientVersion), ClientTrust: clientTrust,
		ExecutableDigest: stringPointer(identity.ExecutableDigest), ReceiptDigest: identity.ReceiptDigest,
		Commands: commands,
	}
	if !validVersion(identity.ClientVersion) {
		return Manifest{}, fmt.Errorf("client version is invalid")
	}
	canonical, err := canonicalJSON(manifest)
	if err != nil {
		return Manifest{}, fmt.Errorf("canonicalize command registry: %w", err)
	}
	digest := sha256.Sum256(canonical)
	manifest.RegistryDigest = registryDigestAlgorithmName + ":" + hex.EncodeToString(digest[:])
	return manifest, nil
}

// BuildExpected constructs the client-independent registry packaged for
// engine-first design evaluation when no qxctl executable is installed.
func BuildExpected(root *cobra.Command) (Manifest, error) {
	commands, err := commandRecords(root)
	if err != nil {
		return Manifest{}, err
	}
	manifest := Manifest{
		Protocol: Protocol, FormatVersion: FormatVersion, RegistryKind: "expected",
		ClientID: "qxctl", ClientTrust: "unreceipted", Commands: commands,
	}
	canonical, err := canonicalJSON(manifest)
	if err != nil {
		return Manifest{}, fmt.Errorf("canonicalize expected command registry: %w", err)
	}
	digest := sha256.Sum256(canonical)
	manifest.RegistryDigest = registryDigestAlgorithmName + ":" + hex.EncodeToString(digest[:])
	return manifest, nil
}

func canonicalJSON(value any) ([]byte, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	var tree any
	if err := decoder.Decode(&tree); err != nil {
		return nil, err
	}
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(tree); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buffer.Bytes(), []byte("\n")), nil
}

// Marshal emits deterministic indented JSON for bounded machine consumption.
// Digest verification canonicalizes the decoded object and is independent of
// presentation whitespace.
func Marshal(manifest Manifest) ([]byte, error) {
	return json.MarshalIndent(manifest, "", "  ")
}

// DigestExecutable hashes the exact running executable with an explicit size
// ceiling. It accepts only a regular file and never consults ambient config.
func DigestExecutable(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open qxctl executable: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("stat qxctl executable: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > maxObservedExecutableSize {
		return "", fmt.Errorf("qxctl executable must be a regular file no larger than %d bytes", maxObservedExecutableSize)
	}
	hash := sha256.New()
	written, err := io.Copy(hash, io.LimitReader(file, maxObservedExecutableSize+1))
	if err != nil {
		return "", fmt.Errorf("hash qxctl executable: %w", err)
	}
	if written != info.Size() {
		return "", fmt.Errorf("qxctl executable changed while hashing")
	}
	return registryDigestAlgorithmName + ":" + hex.EncodeToString(hash.Sum(nil)), nil
}

func commandRecords(root *cobra.Command) ([]CommandRecord, error) {
	if root == nil {
		return nil, fmt.Errorf("command root is required")
	}
	seenIDs := make(map[string]string)
	seenGrammar := make(map[string]string)
	records := make([]CommandRecord, 0)
	var visit func(*cobra.Command) error
	visit = func(command *cobra.Command) error {
		role := command.Annotations[roleAnnotation]
		path := command.CommandPath()
		switch role {
		case roleExecutable:
			if command.Run == nil && command.RunE == nil {
				return fmt.Errorf("registered command %q has no executable handler", path)
			}
			spec, err := decodeSpec(command.Annotations[specAnnotation])
			if err != nil {
				return fmt.Errorf("registered command %q: %w", path, err)
			}
			record, err := project(command, spec)
			if err != nil {
				return fmt.Errorf("registered command %q: %w", path, err)
			}
			if record.Status == "retired" || record.Grammar == nil {
				return fmt.Errorf("registered command %q must use the retirement tombstone factory", path)
			}
			if prior, ok := seenIDs[record.CommandID]; ok {
				return fmt.Errorf("duplicate command ID %q on %q and %q", record.CommandID, prior, path)
			}
			if prior, ok := seenGrammar[*record.Grammar]; ok {
				return fmt.Errorf("duplicate command grammar %q on IDs %q and %q", *record.Grammar, prior, record.CommandID)
			}
			seenIDs[record.CommandID] = path
			seenGrammar[*record.Grammar] = record.CommandID
			records = append(records, record)
		case roleRetired:
			if !command.Hidden || command.Run != nil || command.RunE == nil {
				return fmt.Errorf("retired command %q must remain a hidden fail-closed tombstone", path)
			}
			actual := strconv.FormatUint(uint64(reflect.ValueOf(command.RunE).Pointer()), 16)
			if actual != command.Annotations[retiredHandlerAnnotation] {
				return fmt.Errorf("retired command %q has a handler outside the tombstone factory", path)
			}
			spec, err := decodeSpec(command.Annotations[specAnnotation])
			if err != nil {
				return fmt.Errorf("retired command %q: %w", path, err)
			}
			record, err := project(command, spec)
			if err != nil {
				return fmt.Errorf("retired command %q: %w", path, err)
			}
			if record.Status != "retired" || record.Grammar != nil {
				return fmt.Errorf("retired command %q did not project a retirement tombstone", path)
			}
			if prior, ok := seenIDs[record.CommandID]; ok {
				return fmt.Errorf("duplicate command ID %q on %q and %q", record.CommandID, prior, path)
			}
			seenIDs[record.CommandID] = path
			records = append(records, record)
		case roleStructural:
			actual := strconv.FormatUint(uint64(reflect.ValueOf(command.RunE).Pointer()), 16)
			if actual != command.Annotations[structuralHandlerAnnotation] {
				return fmt.Errorf("structural command %q has an executable handler outside the structural factory", path)
			}
			if command != root && len(command.Commands()) == 0 {
				return fmt.Errorf("structural command %q has no subcommands", path)
			}
		case roleProhibited:
			if !command.Hidden || (command.Run == nil && command.RunE == nil) {
				return fmt.Errorf("prohibited command %q must remain hidden and fail closed", path)
			}
		case roleInternal:
			if !command.Hidden {
				return fmt.Errorf("internal command %q must remain hidden", path)
			}
		case "":
			return fmt.Errorf("command %q has no explicit command-registry role", path)
		default:
			return fmt.Errorf("command %q has unknown command-registry role %q", path, role)
		}
		for _, child := range command.Commands() {
			if err := visit(child); err != nil {
				return err
			}
		}
		return nil
	}
	if err := visit(root); err != nil {
		return nil, err
	}
	if len(records) > 1024 {
		return nil, fmt.Errorf("command registry exceeds 1024 records")
	}
	sort.Slice(records, func(i, j int) bool { return records[i].CommandID < records[j].CommandID })
	return records, nil
}

func decodeSpec(value string) (CommandSpec, error) {
	var spec CommandSpec
	if value == "" {
		return spec, fmt.Errorf("command specification is absent")
	}
	if strings.HasPrefix(value, "!invalid:") {
		return spec, fmt.Errorf("command specification could not be attached: %s", strings.TrimPrefix(value, "!invalid:"))
	}
	if err := json.Unmarshal([]byte(value), &spec); err != nil {
		return spec, fmt.Errorf("decode command specification: %w", err)
	}
	return spec, nil
}

func project(command *cobra.Command, spec CommandSpec) (CommandRecord, error) {
	if !validCommandID(spec.CommandID) {
		return CommandRecord{}, fmt.Errorf("invalid command ID %q", spec.CommandID)
	}
	if spec.Status == "" || spec.IntroducedIn == "" || spec.Mutability == "" ||
		spec.AuthorityMode == "" || spec.TargetScope == "" {
		return CommandRecord{}, fmt.Errorf("command specification has an empty required field")
	}
	if !oneOf(spec.Status, "experimental", "stable", "deprecated", "retired") ||
		!oneOf(spec.Mutability, "read_only", "evidence_only", "proposal_only", "permission_backed_mutation", "prohibited") ||
		!oneOf(spec.AuthorityMode, "none", "target_host_permission", "ssiag") ||
		!oneOf(spec.TargetScope, "local", "target_host") {
		return CommandRecord{}, fmt.Errorf("command specification contains an unknown enum value")
	}
	if !spec.Noninteractive {
		return CommandRecord{}, fmt.Errorf("public qxctl commands must be explicitly noninteractive")
	}
	if !validVersion(spec.IntroducedIn) ||
		(spec.DeprecatedIn != nil && !validVersion(*spec.DeprecatedIn)) {
		return CommandRecord{}, fmt.Errorf("command specification contains an invalid version")
	}
	if (spec.Status == "deprecated" || spec.Status == "retired") != (spec.DeprecatedIn != nil) {
		return CommandRecord{}, fmt.Errorf("deprecated and retired commands require deprecated_in; other statuses prohibit it")
	}
	if spec.Status == "deprecated" && len(spec.ReplacementIDs) == 0 {
		return CommandRecord{}, fmt.Errorf("deprecated command requires a replacement ID")
	}
	if spec.Status == "retired" && (!command.Hidden || spec.Mutability != "prohibited" || len(command.Aliases) != 0) {
		return CommandRecord{}, fmt.Errorf("retired command must be hidden, prohibited, and alias-free")
	}
	if spec.ReplacementIDs == nil || spec.FeatureBindings == nil || spec.BackendOperationIDs == nil ||
		spec.InputProtocols == nil || spec.OutputProtocols == nil || spec.ResultValidationProtocols == nil {
		return CommandRecord{}, fmt.Errorf("command specification arrays must be explicit, not null")
	}
	if spec.Mutability == "permission_backed_mutation" && spec.RecoveryCommandID == nil {
		return CommandRecord{}, fmt.Errorf("permission-backed mutation requires a recovery command ID")
	}
	if (len(spec.FeatureBindings) == 0) == (spec.InfrastructurePurpose == nil) {
		return CommandRecord{}, fmt.Errorf("command must exclusively bind feature interactions or declare an infrastructure purpose")
	}
	if spec.InfrastructurePurpose != nil && (len(*spec.InfrastructurePurpose) == 0 || len(*spec.InfrastructurePurpose) > 4096) {
		return CommandRecord{}, fmt.Errorf("infrastructure purpose is invalid")
	}
	if len(spec.ReplacementIDs) > 32 || !uniqueValid(spec.ReplacementIDs, validCommandID) ||
		len(spec.FeatureBindings) > 256 || !uniqueFeatureBindings(spec.FeatureBindings) ||
		len(spec.BackendOperationIDs) > 256 || !uniqueValid(spec.BackendOperationIDs, validOperationID) ||
		len(spec.InputProtocols) > 64 || !uniqueValid(spec.InputProtocols, validToken) ||
		len(spec.OutputProtocols) > 64 || !uniqueValid(spec.OutputProtocols, validToken) ||
		len(spec.ResultValidationProtocols) > 64 || !uniqueValid(spec.ResultValidationProtocols, validToken) {
		return CommandRecord{}, fmt.Errorf("command specification contains an invalid, duplicate, or oversized semantic array")
	}
	for _, binding := range spec.FeatureBindings {
		if !validFeatureID(binding.FeatureID) || !oneOf(binding.Interaction, "discover", "inspect", "query", "validate", "configure", "propose", "invoke", "apply", "lifecycle", "recover") {
			return CommandRecord{}, fmt.Errorf("feature bindings require feature_id and interaction")
		}
	}
	aliases := append([]string{}, command.Aliases...)
	if len(aliases) > 32 || !uniqueValid(aliases, func(value string) bool { return value != "" && len(value) <= 1024 }) {
		return CommandRecord{}, fmt.Errorf("command aliases are invalid, duplicated, or oversized")
	}
	sort.Strings(aliases)
	visibility := "public"
	if command.Hidden {
		visibility = "hidden"
	}
	jsonOutput := command.Flags().Lookup("json") != nil
	if spec.JSONOutput != nil {
		jsonOutput = *spec.JSONOutput
	}
	var commandGrammar *string
	if spec.Status != "retired" {
		value := grammar(command)
		if value == "" || len(value) > 1024 {
			return CommandRecord{}, fmt.Errorf("command grammar is empty or exceeds 1024 bytes")
		}
		commandGrammar = &value
	}
	if spec.RecoveryCommandID != nil && !validCommandID(*spec.RecoveryCommandID) {
		return CommandRecord{}, fmt.Errorf("recovery command ID is invalid")
	}
	return CommandRecord{
		CommandID: spec.CommandID, Status: spec.Status, IntroducedIn: spec.IntroducedIn,
		DeprecatedIn: spec.DeprecatedIn, ReplacementIDs: sortedCopy(spec.ReplacementIDs),
		Grammar: commandGrammar, Aliases: aliases, Visibility: visibility,
		FeatureBindings:       sortedFeatureBindings(spec.FeatureBindings),
		InfrastructurePurpose: spec.InfrastructurePurpose,
		BackendOperationIDs:   sortedCopy(spec.BackendOperationIDs), Mutability: spec.Mutability,
		AuthorityMode: spec.AuthorityMode, TargetScope: spec.TargetScope,
		InputProtocols: sortedCopy(spec.InputProtocols), OutputProtocols: sortedCopy(spec.OutputProtocols),
		ResultValidationProtocols: sortedCopy(spec.ResultValidationProtocols),
		RecoveryCommandID:         spec.RecoveryCommandID, Noninteractive: spec.Noninteractive,
		JSONOutput: jsonOutput,
	}, nil
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func grammar(command *cobra.Command) string {
	parts := []string{command.CommandPath()}
	useFields := strings.Fields(command.Use)
	if len(useFields) > 1 {
		parts = append(parts, strings.Join(useFields[1:], " "))
	}
	flags := make([]*pflag.Flag, 0)
	command.NonInheritedFlags().VisitAll(func(flag *pflag.Flag) { flags = append(flags, flag) })
	sort.Slice(flags, func(i, j int) bool { return flags[i].Name < flags[j].Name })
	for _, flag := range flags {
		if flag.Hidden || flag.Name == "help" {
			continue
		}
		fragment := "[--" + flag.Name
		if flag.NoOptDefVal == "" {
			fragment += " " + flagPlaceholder(flag.Value.Type())
		}
		parts = append(parts, fragment+"]")
	}
	return strings.Join(parts, " ")
}

func flagPlaceholder(flagType string) string {
	switch flagType {
	case "duration":
		return "DURATION"
	case "uint64":
		return "UINT64"
	case "stringArray", "stringSlice":
		return "VALUE..."
	default:
		return "VALUE"
	}
}

func sortedFeatureBindings(values []FeatureBinding) []FeatureBinding {
	result := append([]FeatureBinding(nil), values...)
	sort.Slice(result, func(i, j int) bool {
		if result[i].FeatureID != result[j].FeatureID {
			return result[i].FeatureID < result[j].FeatureID
		}
		return result[i].Interaction < result[j].Interaction
	})
	return result
}

func sortedCopy(values []string) []string {
	result := append([]string{}, values...)
	sort.Strings(result)
	return result
}

func stringPointer(value string) *string { return &value }

var (
	commandIDPattern   = regexp.MustCompile(`^qxcmd:[a-z][a-z0-9-]{0,62}:[a-z][a-z0-9]*(?:[.-][a-z0-9]+)*$`)
	operationIDPattern = regexp.MustCompile(`^engop:[a-z][a-z0-9-]{0,62}:[a-z][a-z0-9]*(?:[.-][a-z0-9]+)*$`)
	featureIDPattern   = regexp.MustCompile(`^ssfv:[a-z][a-z0-9-]{0,62}:[a-z][a-z0-9]*(?:[.-][a-z0-9]+)*$`)
	versionPattern     = regexp.MustCompile(`^[0-9A-Za-z.+-]+$`)
	tokenPattern       = regexp.MustCompile(`^[A-Za-z0-9._:-]+$`)
)

func validCommandID(value string) bool {
	return len(value) <= 256 && commandIDPattern.MatchString(value)
}
func validOperationID(value string) bool {
	return len(value) <= 256 && operationIDPattern.MatchString(value)
}
func validFeatureID(value string) bool {
	return len(value) <= 256 && featureIDPattern.MatchString(value)
}
func validVersion(value string) bool {
	return len(value) >= 1 && len(value) <= 64 && versionPattern.MatchString(value)
}
func validToken(value string) bool {
	return len(value) >= 1 && len(value) <= 256 && tokenPattern.MatchString(value)
}

func uniqueValid(values []string, valid func(string) bool) bool {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if !valid(value) {
			return false
		}
		if _, duplicate := seen[value]; duplicate {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}

func uniqueFeatureBindings(values []FeatureBinding) bool {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		key := value.FeatureID + "\x00" + value.Interaction
		if _, duplicate := seen[key]; duplicate {
			return false
		}
		seen[key] = struct{}{}
	}
	return true
}

func validTaggedDigest(value string) bool {
	if len(value) != len("sha256:")+64 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}
