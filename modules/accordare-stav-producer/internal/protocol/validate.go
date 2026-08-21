package protocol

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
)

var ssiagKindPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)

const (
	maxSubmissionBytes = 1 << 20
	resultProtocol     = "symphony.knowledge.named-version-result.v1"
)

type decisionSubject struct {
	Authority string `json:"authority"`
	ID        string `json:"id"`
	Kind      string `json:"kind"`
}

type decisionTarget struct {
	Audience  string `json:"audience"`
	Operation string `json:"operation"`
	Resource  string `json:"resource"`
	Scope     string `json:"scope"`
}

type capability struct {
	AuthorityBasis string          `json:"authority_basis"`
	BindingDigest  string          `json:"binding_digest"`
	CanonicalApply bool            `json:"canonical_apply"`
	CapabilityID   string          `json:"capability_id"`
	ConfigDigest   string          `json:"config_digest"`
	CorrelationID  string          `json:"correlation_id"`
	ExpiresAt      time.Time       `json:"expires_at"`
	GrantID        string          `json:"grant_id"`
	IssuedAt       time.Time       `json:"issued_at"`
	PolicyDigest   string          `json:"policy_digest"`
	Protocol       string          `json:"protocol"`
	RequestID      string          `json:"request_id"`
	Subject        decisionSubject `json:"subject"`
	Target         decisionTarget  `json:"target"`
	TOPSID         string          `json:"tops_id"`
	Transferable   bool            `json:"transferable"`
}

type authorizationDecision struct {
	AuthorityBasis  *string         `json:"authority_basis"`
	CallerClassUsed bool            `json:"caller_class_used"`
	CanonicalApply  bool            `json:"canonical_apply"`
	Capability      *capability     `json:"capability"`
	ConfigDigest    string          `json:"config_digest"`
	CorrelationID   string          `json:"correlation_id"`
	DecidedAt       time.Time       `json:"decided_at"`
	DecisionID      string          `json:"decision_id"`
	Effect          string          `json:"effect"`
	ExpiresAt       *time.Time      `json:"expires_at"`
	PolicyDigest    string          `json:"policy_digest"`
	ReasonCode      string          `json:"reason_code"`
	RequestID       string          `json:"request_id"`
	Schema          string          `json:"schema"`
	Subject         decisionSubject `json:"subject"`
	Target          decisionTarget  `json:"target"`
	TOPSID          string          `json:"tops_id"`
}

type command struct {
	Alias                  *string               `json:"alias"`
	AuthorizationDecision  authorizationDecision `json:"authorization_decision"`
	Client                 json.RawMessage       `json:"client"`
	ExpectedRegistryDigest *string               `json:"expected_registry_digest"`
	NamedVersion           json.RawMessage       `json:"named_version"`
	Operation              string                `json:"operation"`
	OperationID            *string               `json:"operation_id"`
	PreparedOperationID    *string               `json:"prepared_operation_id"`
	ProposalDigest         *string               `json:"proposal_digest"`
	Protocol               string                `json:"protocol"`
	SAVEngine              json.RawMessage       `json:"sav_engine"`
	Selector               json.RawMessage       `json:"selector"`
	StateRoot              string                `json:"state_root"`
	TOPSID                 string                `json:"tops_id"`
	ValidationResult       json.RawMessage       `json:"validation_result"`
}

type namedVersionResult struct {
	AliasCount            uint64          `json:"alias_count"`
	Artifact              json.RawMessage `json:"artifact"`
	Canonical             bool            `json:"canonical"`
	CanonicalApplyEnabled bool            `json:"canonical_apply_enabled"`
	Changed               bool            `json:"changed"`
	Compatibility         json.RawMessage `json:"compatibility"`
	FormatVersion         uint64          `json:"format_version"`
	Operation             string          `json:"operation"`
	ProposalDigest        *string         `json:"proposal_digest"`
	ReadOnly              bool            `json:"read_only"`
	RegistryPresent       bool            `json:"registry_present"`
	RepairActions         []string        `json:"repair_actions"`
	Recovered             bool            `json:"recovered"`
	RegistryDigest        *string         `json:"registry_digest"`
	ResultDigest          string          `json:"result_digest"`
	SelectedAlias         *string         `json:"selected_alias"`
	STAVAppendEnabled     bool            `json:"stav_append_enabled"`
	Protocol              string          `json:"protocol"`
	VersionCount          uint64          `json:"version_count"`
}

type VerifiedSubmission struct {
	Actor           stavprotocol.SafeReference
	Candidate       stavprotocol.Candidate
	CandidateDigest string
}

type VerifiedIntent struct {
	IntentID   string
	Peer       stavprotocol.SafeReference
	Submission Submission
}

func SubmissionIntentID(raw Submission) (string, error) {
	commandBytes, err := canonicalObject(raw.Command)
	if err != nil {
		return "", fmt.Errorf("command evidence is invalid")
	}
	var identity struct {
		Operation   string  `json:"operation"`
		OperationID *string `json:"operation_id"`
		TOPSID      string  `json:"tops_id"`
	}
	if err := json.Unmarshal(commandBytes, &identity); err != nil || identity.OperationID == nil || !token(*identity.OperationID) || identity.Operation != raw.Operation || identity.TOPSID != raw.TOPSID {
		return "", fmt.Errorf("command identity is invalid")
	}
	return stableUUID(raw.TOPSID + "\n" + raw.Operation + "\n" + *identity.OperationID + "\nintent"), nil
}

// CommandsMatchIntent compares the complete coordinator command except for the
// SSIAG decision. Retry must carry fresh authorization, while every mutation
// semantic and client capability remains exact.
func CommandsMatchIntent(left, right json.RawMessage) bool {
	leftCanonical, leftErr := canonicalIntentCommand(left)
	rightCanonical, rightErr := canonicalIntentCommand(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftCanonical, rightCanonical)
}

func canonicalIntentCommand(raw json.RawMessage) ([]byte, error) {
	canonical, err := canonicalObject(raw)
	if err != nil {
		return nil, err
	}
	var value map[string]json.RawMessage
	if err := json.Unmarshal(canonical, &value); err != nil {
		return nil, err
	}
	if _, present := value["authorization_decision"]; !present {
		return nil, fmt.Errorf("command lacks authorization evidence")
	}
	value["authorization_decision"] = json.RawMessage("null")
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return canonicalJSON(encoded)
}

// VerifyIntent authenticates and canonicalizes the exact command before the
// coordinator is allowed to mutate. The returned identity is deterministic so
// an exact command retry observes the same durable intent.
func VerifyIntent(raw Submission, peerSubject stavprotocol.SafeReference, now time.Time) (VerifiedIntent, error) {
	if raw.Schema != SchemaSubmission || raw.Operation == "" || raw.TOPSID == "" || len(raw.Command) == 0 || !nullRaw(raw.Result) || raw.Outcome != nil || raw.ReasonCode != nil {
		return VerifiedIntent{}, fmt.Errorf("intent identity is incomplete")
	}
	if err := stavprotocol.ValidateTOPSID(raw.TOPSID); err != nil {
		return VerifiedIntent{}, err
	}
	if raw.Coordinator.ComponentID != "knowledge-session-coordinator" || !taggedDigest(raw.Coordinator.ReceiptDigest) || !taggedDigest(raw.Coordinator.ExecutableDigest) || !token(raw.Coordinator.Version) {
		return VerifiedIntent{}, fmt.Errorf("coordinator evidence is invalid")
	}
	commandBytes, err := canonicalObject(raw.Command)
	if err != nil || len(commandBytes) > maxSubmissionBytes {
		return VerifiedIntent{}, fmt.Errorf("command evidence is invalid")
	}
	var cmd command
	if err := exactDecode(commandBytes, &cmd, []string{
		"alias", "authorization_decision", "client", "expected_registry_digest", "named_version",
		"operation", "operation_id", "prepared_operation_id", "proposal_digest", "protocol", "sav_engine",
		"selector", "state_root", "tops_id", "validation_result",
	}); err != nil {
		return VerifiedIntent{}, fmt.Errorf("command field set is invalid: %w", err)
	}
	if cmd.Protocol != "symphony.knowledge.named-version-command.v1" || cmd.Operation != raw.Operation || cmd.TOPSID != raw.TOPSID || cmd.OperationID == nil || !token(*cmd.OperationID) || !supportedOperation(raw.Operation) {
		return VerifiedIntent{}, fmt.Errorf("command identity is invalid")
	}
	if err := validateDecision(cmd.AuthorizationDecision, raw.TOPSID, raw.Operation, namedVersionResource(commandBytes), peerSubject, now); err != nil {
		return VerifiedIntent{}, err
	}
	raw.Command = append(json.RawMessage(nil), commandBytes...)
	intentID, _ := SubmissionIntentID(raw)
	return VerifiedIntent{
		IntentID: intentID,
		Peer:     peerSubject, Submission: raw,
	}, nil
}

// VerifyCompletion binds a terminal coordinator result to the exact durable
// pre-mutation intent. Completion carries a freshly validated authorization
// proof; all mutation semantics, client capabilities, coordinator installation,
// peer, operation, and TOPS identity remain exact.
func VerifyCompletion(prepared Submission, result Submission, peerSubject stavprotocol.SafeReference, preparedAt, now time.Time) (VerifiedSubmission, string, error) {
	if prepared.Schema != result.Schema || prepared.Operation != result.Operation || prepared.TOPSID != result.TOPSID || prepared.Coordinator != result.Coordinator {
		return VerifiedSubmission{}, "", fmt.Errorf("completion does not bind the prepared intent")
	}
	left, err := canonicalObject(prepared.Command)
	if err != nil {
		return VerifiedSubmission{}, "", err
	}
	if !CommandsMatchIntent(left, result.Command) {
		return VerifiedSubmission{}, "", fmt.Errorf("completion command differs from prepared intent")
	}
	verified, err := VerifySubmission(result, peerSubject, now)
	if err != nil {
		return VerifiedSubmission{}, "", err
	}
	intent, err := VerifyIntent(prepared, peerSubject, preparedAt)
	if err != nil {
		return VerifiedSubmission{}, "", err
	}
	return verified, intent.IntentID, nil
}

func VerifySubmission(raw Submission, peerSubject stavprotocol.SafeReference, now time.Time) (VerifiedSubmission, error) {
	if raw.Schema != SchemaSubmission || raw.Operation == "" || raw.TOPSID == "" || len(raw.Command) == 0 || raw.Outcome == nil || raw.ReasonCode == nil {
		return VerifiedSubmission{}, fmt.Errorf("submission identity is incomplete")
	}
	if err := stavprotocol.ValidateTOPSID(raw.TOPSID); err != nil {
		return VerifiedSubmission{}, err
	}
	if raw.Coordinator.ComponentID != "knowledge-session-coordinator" || !taggedDigest(raw.Coordinator.ReceiptDigest) ||
		!taggedDigest(raw.Coordinator.ExecutableDigest) || !token(raw.Coordinator.Version) {
		return VerifiedSubmission{}, fmt.Errorf("coordinator evidence is invalid")
	}
	commandBytes, err := canonicalObject(raw.Command)
	if err != nil || len(commandBytes) > maxSubmissionBytes {
		return VerifiedSubmission{}, fmt.Errorf("command evidence is invalid")
	}
	var cmd command
	if err := exactDecode(commandBytes, &cmd, []string{
		"alias", "authorization_decision", "client", "expected_registry_digest", "named_version",
		"operation", "operation_id", "prepared_operation_id", "proposal_digest", "protocol", "sav_engine",
		"selector", "state_root", "tops_id", "validation_result",
	}); err != nil {
		return VerifiedSubmission{}, fmt.Errorf("command field set is invalid: %w", err)
	}
	if cmd.Protocol != "symphony.knowledge.named-version-command.v1" || cmd.Operation != raw.Operation || cmd.TOPSID != raw.TOPSID || cmd.OperationID == nil || !token(*cmd.OperationID) {
		return VerifiedSubmission{}, fmt.Errorf("command identity is invalid")
	}
	if !supportedOperation(raw.Operation) {
		return VerifiedSubmission{}, fmt.Errorf("operation is outside the Accordare vocabulary")
	}
	if err := validateDecision(cmd.AuthorizationDecision, raw.TOPSID, raw.Operation, namedVersionResource(commandBytes), peerSubject, now); err != nil {
		return VerifiedSubmission{}, err
	}
	operationID := strings.Replace(raw.Operation, "named_version_", "symphony.sav.named-version.", 1)
	if (*raw.Outcome != "succeeded" && *raw.Outcome != "failed" && *raw.Outcome != "unavailable") || *raw.ReasonCode != operationID+"."+*raw.Outcome {
		return VerifiedSubmission{}, fmt.Errorf("terminal outcome is outside the Accordare vocabulary")
	}
	stavActor, err := NormalizeSSIAGSubject(peerSubject)
	if err != nil {
		return VerifiedSubmission{}, err
	}
	if *raw.Outcome != "succeeded" {
		if !nullRaw(raw.Result) {
			return VerifiedSubmission{}, fmt.Errorf("non-success terminal evidence cannot contain a result payload")
		}
		candidate, err := buildTerminalCandidate(raw, cmd, stavActor, *raw.Outcome)
		if err != nil {
			return VerifiedSubmission{}, err
		}
		digest, err := stavprotocol.CandidateDigest(candidate)
		if err != nil {
			return VerifiedSubmission{}, err
		}
		return VerifiedSubmission{Actor: peerSubject, Candidate: candidate, CandidateDigest: digest}, nil
	}
	if nullRaw(raw.Result) {
		return VerifiedSubmission{}, fmt.Errorf("successful completion lacks result evidence")
	}
	resultBytes, err := canonicalObject(raw.Result)
	if err != nil || len(resultBytes) > maxSubmissionBytes {
		return VerifiedSubmission{}, fmt.Errorf("result evidence is invalid")
	}
	var result namedVersionResult
	if err := exactDecode(resultBytes, &result, []string{
		"alias_count", "artifact", "canonical", "canonical_apply_enabled", "changed", "compatibility",
		"format_version", "operation", "proposal_digest", "read_only", "recovered", "registry_digest",
		"registry_present", "repair_actions", "result_digest", "selected_alias", "stav_append_enabled", "protocol", "version_count",
	}); err != nil {
		return VerifiedSubmission{}, fmt.Errorf("result field set is invalid: %w", err)
	}
	if result.Protocol != resultProtocol || result.FormatVersion != 1 || result.Operation != raw.Operation ||
		result.ReadOnly || result.Canonical || result.CanonicalApplyEnabled || result.STAVAppendEnabled || !taggedDigest(result.ResultDigest) {
		return VerifiedSubmission{}, fmt.Errorf("result identity is invalid")
	}
	if digest, err := digestWithout(resultBytes, "result_digest"); err != nil || digest != result.ResultDigest {
		return VerifiedSubmission{}, fmt.Errorf("result digest is invalid")
	}
	candidate, err := buildCandidate(raw, cmd, result, stavActor)
	if err != nil {
		return VerifiedSubmission{}, err
	}
	digest, err := stavprotocol.CandidateDigest(candidate)
	if err != nil {
		return VerifiedSubmission{}, err
	}
	return VerifiedSubmission{Actor: peerSubject, Candidate: candidate, CandidateDigest: digest}, nil
}

// NormalizeSSIAGSubject maps SSIAG's compact subject kinds into STAV's
// registered-identifier namespace without changing the opaque subject ID.
// Already namespaced kinds remain exact. The mapping is closed and mechanical;
// it never infers caller class or authority.
func NormalizeSSIAGSubject(subject stavprotocol.SafeReference) (stavprotocol.SafeReference, error) {
	kind := subject.Kind
	if !strings.Contains(kind, ".") {
		if !ssiagKindPattern.MatchString(kind) {
			return stavprotocol.SafeReference{}, fmt.Errorf("SSIAG subject kind is invalid")
		}
		kind = "symphony.identity." + kind
	}
	normalized := stavprotocol.SafeReference{ID: subject.ID, Kind: kind}
	probe := stavprotocol.Candidate{
		Actor:         stavprotocol.CandidateActor{Authentication: stavprotocol.Authentication{MethodID: "symphony.ssiag.local-peer", State: "identified"}, Principal: normalized},
		Configuration: stavprotocol.Configuration{PreviousDigest: taggedSHA256([]byte("subject-validation")), NewDigest: taggedSHA256([]byte("subject-validation")), State: "digests"},
		Correlation:   stavprotocol.Correlation{RequestID: "00000000-0000-4000-8000-000000000000", CorrelationID: "00000000-0000-4000-8000-000000000000"},
		Operation:     stavprotocol.Operation{EventClass: "symphony.validation.subject", OperationID: "symphony.validation.subject", Target: stavprotocol.SafeReference{ID: "subject-validation", Kind: "symphony.validation.subject"}},
		Redaction:     stavprotocol.Redaction{Classification: "administrative_metadata"}, Result: stavprotocol.Result{IntentID: "symphony.validation.subject", Outcome: "succeeded", ReasonCode: "symphony.validation.subject.succeeded"},
		Schema: stavprotocol.SchemaCandidate, Topology: stavprotocol.Topology{TOPSID: "00000000-0000-4000-8000-000000000000", TROG: stavprotocol.TROG{ReasonCode: "symphony.stav.trog.not-applicable", State: "not_applicable"}},
	}
	if err := probe.Validate(); err != nil {
		return stavprotocol.SafeReference{}, err
	}
	return normalized, nil
}

func buildTerminalCandidate(submission Submission, cmd command, actor stavprotocol.SafeReference, outcome string) (stavprotocol.Candidate, error) {
	absent := taggedSHA256([]byte("symphony.sav.named-version-registry.absent.v1"))
	unobserved := taggedSHA256([]byte("symphony.sav.named-version-registry.previous-unobserved.v1"))
	previous := expectedOrSentinel(cmd.ExpectedRegistryDigest, absent, unobserved)
	targetID, targetKind := previous, "symphony.sav.named-version-registry"
	switch submission.Operation {
	case "named_version_prepare":
		var value struct {
			NamedVersionDigest string `json:"named_version_digest"`
		}
		if err := exactNamedVersionDigest(cmd.NamedVersion, &value); err != nil || !taggedDigest(value.NamedVersionDigest) {
			return stavprotocol.Candidate{}, fmt.Errorf("prepare intent lacks a Named Version digest")
		}
		targetID, targetKind = value.NamedVersionDigest, "symphony.sav.named-version-proposal"
	case "named_version_seal":
		if cmd.ProposalDigest == nil || !taggedDigest(*cmd.ProposalDigest) {
			return stavprotocol.Candidate{}, fmt.Errorf("seal intent lacks a proposal digest")
		}
		targetID, targetKind = *cmd.ProposalDigest, "symphony.sav.named-version"
	case "named_version_alias":
		var selector struct {
			Kind  string `json:"kind"`
			Value string `json:"value"`
		}
		if err := json.Unmarshal(cmd.Selector, &selector); err != nil || selector.Kind != "digest" || !taggedDigest(selector.Value) {
			return stavprotocol.Candidate{}, fmt.Errorf("alias intent lacks an exact target")
		}
		targetID = selector.Value
	}
	requestID := stableUUID(submission.TOPSID + "\n" + submission.Operation + "\n" + *cmd.OperationID + "\nrequest")
	correlationID := stableUUID(submission.TOPSID + "\n" + submission.Operation + "\n" + *cmd.OperationID + "\ncorrelation")
	operationID := strings.Replace(submission.Operation, "named_version_", "symphony.sav.named-version.", 1)
	candidate := stavprotocol.Candidate{
		Actor:         stavprotocol.CandidateActor{Authentication: stavprotocol.Authentication{MethodID: "symphony.ssiag.local-peer", State: "identified"}, Principal: actor},
		Configuration: stavprotocol.Configuration{PreviousDigest: previous, NewDigest: previous, State: "digests"},
		Correlation:   stavprotocol.Correlation{RequestID: requestID, CorrelationID: correlationID},
		Operation:     stavprotocol.Operation{EventClass: "symphony.sav.named-version.lifecycle", OperationID: operationID, Target: stavprotocol.SafeReference{ID: targetID, Kind: targetKind}},
		Redaction:     stavprotocol.Redaction{Classification: "administrative_metadata"},
		Result:        stavprotocol.Result{IntentID: operationID, Outcome: outcome, ReasonCode: operationID + "." + outcome},
		Schema:        stavprotocol.SchemaCandidate,
		Topology:      stavprotocol.Topology{TOPSID: submission.TOPSID, TROG: stavprotocol.TROG{ReasonCode: "symphony.stav.trog.not-applicable", State: "not_applicable"}},
	}
	if err := candidate.Validate(); err != nil {
		return stavprotocol.Candidate{}, fmt.Errorf("derived terminal candidate is invalid: %w", err)
	}
	return candidate, nil
}

func buildCandidate(submission Submission, cmd command, result namedVersionResult, actor stavprotocol.SafeReference) (stavprotocol.Candidate, error) {
	var previous, next, targetID, targetKind string
	absent := taggedSHA256([]byte("symphony.sav.named-version-registry.absent.v1"))
	unobserved := taggedSHA256([]byte("symphony.sav.named-version-registry.previous-unobserved.v1"))
	switch submission.Operation {
	case "named_version_prepare":
		if result.ProposalDigest == nil || !taggedDigest(*result.ProposalDigest) {
			return stavprotocol.Candidate{}, fmt.Errorf("prepare result lacks a proposal digest")
		}
		var version struct {
			NamedVersionDigest string `json:"named_version_digest"`
		}
		if err := exactNamedVersionDigest(cmd.NamedVersion, &version); err != nil || !taggedDigest(version.NamedVersionDigest) {
			return stavprotocol.Candidate{}, fmt.Errorf("prepare command lacks a valid Named Version digest")
		}
		previous, next, targetID, targetKind = version.NamedVersionDigest, *result.ProposalDigest, *result.ProposalDigest, "symphony.sav.named-version-proposal"
	case "named_version_seal":
		if result.RegistryDigest == nil || !taggedDigest(*result.RegistryDigest) {
			return stavprotocol.Candidate{}, fmt.Errorf("seal result lacks a registry digest")
		}
		previous = expectedOrAbsent(cmd.ExpectedRegistryDigest, absent)
		next = *result.RegistryDigest
		var artifact struct {
			NamedVersionDigest string `json:"named_version_digest"`
		}
		if err := exactNamedVersionDigest(result.Artifact, &artifact); err != nil || !taggedDigest(artifact.NamedVersionDigest) {
			return stavprotocol.Candidate{}, fmt.Errorf("seal result lacks a valid artifact digest")
		}
		targetID, targetKind = artifact.NamedVersionDigest, "symphony.sav.named-version"
	case "named_version_alias", "named_version_recover":
		if result.RegistryDigest == nil || !taggedDigest(*result.RegistryDigest) {
			return stavprotocol.Candidate{}, fmt.Errorf("registry result lacks a digest")
		}
		previous, next = expectedOrSentinel(cmd.ExpectedRegistryDigest, absent, unobserved), *result.RegistryDigest
		targetID, targetKind = *result.RegistryDigest, "symphony.sav.named-version-registry"
	}
	requestID := stableUUID(submission.TOPSID + "\n" + submission.Operation + "\n" + *cmd.OperationID + "\nrequest")
	correlationID := stableUUID(submission.TOPSID + "\n" + submission.Operation + "\n" + *cmd.OperationID + "\ncorrelation")
	operationID := strings.Replace(submission.Operation, "named_version_", "symphony.sav.named-version.", 1)
	candidate := stavprotocol.Candidate{
		Actor:         stavprotocol.CandidateActor{Authentication: stavprotocol.Authentication{MethodID: "symphony.ssiag.local-peer", State: "identified"}, Principal: actor},
		Configuration: stavprotocol.Configuration{PreviousDigest: previous, NewDigest: next, State: "digests"},
		Correlation:   stavprotocol.Correlation{RequestID: requestID, CorrelationID: correlationID},
		Operation:     stavprotocol.Operation{EventClass: "symphony.sav.named-version.lifecycle", OperationID: operationID, Target: stavprotocol.SafeReference{ID: targetID, Kind: targetKind}},
		Redaction:     stavprotocol.Redaction{Classification: "administrative_metadata"},
		Result:        stavprotocol.Result{IntentID: operationID, Outcome: "succeeded", ReasonCode: operationID + ".succeeded"},
		Schema:        stavprotocol.SchemaCandidate,
		Topology:      stavprotocol.Topology{TOPSID: submission.TOPSID, TROG: stavprotocol.TROG{ReasonCode: "symphony.stav.trog.not-applicable", State: "not_applicable"}},
	}
	if err := candidate.Validate(); err != nil {
		return stavprotocol.Candidate{}, fmt.Errorf("derived candidate is invalid: %w", err)
	}
	return candidate, nil
}

func validateDecision(decision authorizationDecision, topsID, operation, resource string, peer stavprotocol.SafeReference, now time.Time) error {
	wantOperation := "symphony.knowledge." + operation
	if decision.Schema != "symphony.ssiag.authorization-decision.v1" || decision.Effect != "allow" ||
		decision.ReasonCode != "symphony.ssiag.policy.exact-grant" || decision.TOPSID != topsID ||
		decision.Target.Operation != wantOperation || decision.Target.Resource != resource ||
		decision.Target.Audience != "qxctl" || decision.Target.Scope != "tops:"+topsID ||
		decision.Subject.ID != peer.ID || decision.Subject.Kind != peer.Kind || decision.Subject.Authority != "unix_peer_credentials" ||
		decision.CallerClassUsed || decision.CanonicalApply || decision.AuthorityBasis == nil || decision.Capability == nil || decision.ExpiresAt == nil {
		return fmt.Errorf("SSIAG decision does not bind the submitted operation and kernel peer")
	}
	capability := decision.Capability
	basis := *decision.AuthorityBasis
	if capability.Protocol != "symphony.ssiag.capability.v1" || capability.TOPSID != topsID ||
		capability.Subject != decision.Subject || capability.Target != decision.Target || capability.RequestID != decision.RequestID ||
		capability.CorrelationID != decision.CorrelationID || capability.AuthorityBasis != basis || capability.PolicyDigest != decision.PolicyDigest ||
		capability.ConfigDigest != decision.ConfigDigest || capability.Transferable || capability.CanonicalApply ||
		!taggedDigest(capability.PolicyDigest) || !taggedDigest(capability.ConfigDigest) || !taggedDigest(capability.BindingDigest) ||
		capabilityBinding(*capability) != capability.BindingDigest || capability.CapabilityID != "ssiag-capability:"+strings.TrimPrefix(capability.BindingDigest, "sha256:") ||
		!capability.IssuedAt.Equal(decision.DecidedAt) || !capability.ExpiresAt.Equal(*decision.ExpiresAt) ||
		capability.IssuedAt.After(now.Add(30*time.Second)) || !capability.ExpiresAt.After(now) || !capability.ExpiresAt.After(capability.IssuedAt) ||
		(basis != "host_owner" && basis != "granted_permission") {
		return fmt.Errorf("SSIAG capability evidence is inconsistent or expired")
	}
	return nil
}

func capabilityBinding(value capability) string {
	joined := strings.Join([]string{
		value.Protocol, value.Subject.ID, value.Subject.Kind, value.Subject.Authority,
		value.TOPSID, value.Target.Operation, value.Target.Resource, value.Target.Audience, value.Target.Scope,
		value.AuthorityBasis, value.GrantID, value.RequestID, value.CorrelationID,
		value.IssuedAt.UTC().Format(time.RFC3339), value.ExpiresAt.UTC().Format(time.RFC3339),
		value.PolicyDigest, value.ConfigDigest, "transferable=false", "canonical_apply=false",
	}, "\n")
	return taggedSHA256([]byte(joined))
}

func namedVersionResource(commandBytes []byte) string {
	var value map[string]any
	_ = json.Unmarshal(commandBytes, &value)
	var versionDigest any
	if version, ok := value["named_version"].(map[string]any); ok {
		versionDigest = version["named_version_digest"]
	}
	normalized := map[string]any{
		"tops_id": value["tops_id"], "operation": value["operation"],
		"expected_registry_digest": value["expected_registry_digest"], "named_version_digest": versionDigest,
		"prepared_operation_id": value["prepared_operation_id"], "proposal_digest": value["proposal_digest"],
		"alias": value["alias"], "selector": value["selector"],
	}
	encoded, _ := json.Marshal(normalized)
	digest := sha256.Sum256(encoded)
	return "symphony.knowledge.named-version:" + hex.EncodeToString(digest[:])
}

func stableUUID(seed string) string {
	digest := sha256.Sum256([]byte(seed))
	value := digest[:16]
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", value[0:4], value[4:6], value[6:8], value[8:10], value[10:16])
}

func expectedOrAbsent(value *string, absent string) string {
	return expectedOrSentinel(value, absent, absent)
}

func expectedOrSentinel(value *string, absent, unobserved string) string {
	if value == nil || *value == "absent" {
		return absent
	}
	if *value == "discover" {
		return unobserved
	}
	return *value
}

func exactNamedVersionDigest(raw []byte, target any) error {
	canonical, err := canonicalObject(raw)
	if err != nil {
		return err
	}
	return json.Unmarshal(canonical, target)
}

func digestWithout(raw []byte, field string) (string, error) {
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", err
	}
	delete(value, field)
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	canonical, err := canonicalJSON(encoded)
	if err != nil {
		return "", err
	}
	return taggedSHA256(canonical), nil
}

func exactDecode(raw []byte, target any, fields []string) error {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || len(object) != len(fields) {
		return fmt.Errorf("unexpected field count")
	}
	for _, field := range fields {
		if _, ok := object[field]; !ok {
			return fmt.Errorf("missing field %s", field)
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return fmt.Errorf("trailing JSON")
	}
	return nil
}

func canonicalObject(raw []byte) ([]byte, error) {
	canonical, err := canonicalJSON(raw)
	if err != nil {
		return nil, err
	}
	if len(canonical) == 0 || canonical[0] != '{' {
		return nil, fmt.Errorf("JSON object required")
	}
	return canonical, nil
}

func supportedOperation(value string) bool {
	switch value {
	case "named_version_prepare", "named_version_seal", "named_version_alias", "named_version_recover":
		return true
	default:
		return false
	}
}

func token(value string) bool {
	if value == "" || len(value) > 256 || strings.TrimSpace(value) != value {
		return false
	}
	for _, c := range value {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || strings.ContainsRune("._:+-", c) {
			continue
		}
		return false
	}
	return true
}

func taggedDigest(value string) bool {
	if len(value) != 71 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

func taggedSHA256(value []byte) string {
	digest := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(digest[:])
}
