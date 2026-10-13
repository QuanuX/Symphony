// Package sovremote implements the read-only SOV remote handoff contract.
// It does not dispatch, authenticate a target, reserve resources, or grant access.
package sovremote

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"path"
	"reflect"
	"regexp"
	"strings"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
)

const PlanProtocol = "symphony.sov.remote-plan-input.v1"
const AssessmentProtocol = "symphony.sov.remote-assess-input.v1"

// MaxDocumentBytes bounds a control document, never the planned dataset or RAM.
const MaxDocumentBytes int64 = 8 << 20

type Target struct {
	TOPS           string `json:"tops_id"`
	Node           string `json:"node_id"`
	Habitat        string `json:"habitat_id"`
	IdentitySHA256 string `json:"identity_sha256"`
}
type Transport struct {
	Arrangement    string `json:"arrangement"`
	Family         string `json:"family"`
	AdapterID      string `json:"adapter_id"`
	AdapterVersion string `json:"adapter_version"`
	AdapterSHA256  string `json:"adapter_sha256"`
	EndpointRef    string `json:"endpoint_ref"`
}
type Runtime struct {
	QXCTLVersion          string `json:"qxctl_version"`
	QXCTLSHA256           string `json:"qxctl_sha256"`
	SBVVersion            string `json:"sbv_version"`
	SBVInstallationSHA256 string `json:"sbv_installation_sha256"`
}
type Job struct {
	Directory  string `json:"directory"`
	ID         string `json:"job_id"`
	PlanSHA256 string `json:"plan_sha256"`
}
type Resources struct {
	ObservationSHA256 *string  `json:"observation_sha256"`
	Threads           *string  `json:"threads"`
	MemoryBytes       *string  `json:"memory_bytes"`
	Devices           []string `json:"device_refs"`
	DeadlineUnixMS    *string  `json:"deadline_unix_ms"`
}
type Authority struct {
	Operation    string  `json:"operation"`
	Resource     string  `json:"resource"`
	Audience     string  `json:"audience"`
	Scope        string  `json:"scope"`
	PolicySHA256 *string `json:"policy_sha256"`
}
type Selection struct {
	Protocol    string            `json:"protocol"`
	OperationID string            `json:"operation_id"`
	Target      Target            `json:"target"`
	Transport   Transport         `json:"transport"`
	Runtime     Runtime           `json:"runtime"`
	Job         Job               `json:"job"`
	Resources   Resources         `json:"resources"`
	Authority   Authority         `json:"authority"`
	Extensions  map[string]string `json:"extensions"`
}
type Plan struct {
	Protocol                  string    `json:"protocol"`
	PlanSHA256                string    `json:"plan_sha256"`
	Selection                 Selection `json:"selection"`
	Status                    string    `json:"status"`
	RemoteOperational         bool      `json:"remote_operational"`
	AuthorityVerified         bool      `json:"authority_verified"`
	ResourcesReserved         bool      `json:"resources_reserved"`
	AutomaticRetry            bool      `json:"automatic_retry"`
	CheckpointResumeSupported bool      `json:"checkpoint_resume_supported"`
	Requirements              []string  `json:"requirements"`
}
type AssessmentInput struct {
	Protocol             string          `json:"protocol"`
	Selection            Selection       `json:"selection"`
	ExpectedPlanSHA256   string          `json:"expected_plan_sha256"`
	TransportOutcome     string          `json:"transport_outcome"`
	ObservedTargetSHA256 *string         `json:"observed_target_sha256"`
	Observation          json.RawMessage `json:"observation"`
}
type Assessment struct {
	Protocol          string          `json:"protocol"`
	PlanSHA256        string          `json:"plan_sha256"`
	EvidenceStatus    string          `json:"evidence_status"`
	TransportOutcome  string          `json:"transport_outcome"`
	AutomaticRetry    bool            `json:"automatic_retry"`
	AuthorityVerified bool            `json:"authority_verified"`
	RecommendedAction string          `json:"recommended_action"`
	Observation       json.RawMessage `json:"observation"`
}

var sha = regexp.MustCompile(`^[a-f0-9]{64}$`)
var positive = regexp.MustCompile(`^[1-9][0-9]*$`)
var jobID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,80}$`)

// Every field, including explicit null/false/empty values, is required. Compare
// decoded struct shape as well as DisallowUnknownFields: Go alone otherwise
// accepts omitted nulls, case-insensitive member aliases, and duplicate keys.
func decode(raw []byte, dst any) error {
	if err := knowledgeengine.ValidateJSONObject(raw, MaxDocumentBytes); err != nil {
		return fmt.Errorf("invalid SOV control JSON")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(dst); err != nil {
		return fmt.Errorf("invalid SOV control shape")
	}
	b, err := json.Marshal(dst)
	if err != nil {
		return err
	}
	var input, shape any
	_ = json.Unmarshal(raw, &input)
	_ = json.Unmarshal(b, &shape)
	if !sameShape(input, shape) {
		return fmt.Errorf("missing or invalid SOV control member")
	}
	return nil
}
func sameShape(a, b any) bool {
	x, ok := b.(map[string]any)
	if !ok {
		return reflect.TypeOf(a) == reflect.TypeOf(b)
	}
	y, ok := a.(map[string]any)
	if !ok || len(x) != len(y) {
		return false
	}
	for k, v := range x {
		w, exists := y[k]
		if !exists || !sameShape(w, v) {
			return false
		}
	}
	return true
}
func named(s string) bool { return strings.TrimSpace(s) != "" && !strings.ContainsRune(s, '\x00') }
func exactVersion(s string) bool {
	return named(s) && s != "latest" && s != "*" && !strings.ContainsAny(s, "<>^~ ")
}

// qxctl currently reports this development identity; its mandatory executable
// digest selects exact bytes even though the display version is not SemVer.
func exactQXCTLVersion(s string) bool { return s == "qxctl dev" || exactVersion(s) }
func nullablePositive(s *string) bool { return s == nil || positive.MatchString(*s) }
func nullableSHA(s *string) bool      { return s == nil || sha.MatchString(*s) }
func validate(s Selection) error {
	bad := func() error { return fmt.Errorf("invalid SOV remote selection") }
	if s.Protocol != PlanProtocol || !named(s.OperationID) || !named(s.Target.TOPS) || !named(s.Target.Node) || !named(s.Target.Habitat) || !sha.MatchString(s.Target.IdentitySHA256) {
		return bad()
	}
	if s.Transport.Arrangement != "controller" && s.Transport.Arrangement != "target" && s.Transport.Arrangement != "staged" {
		return bad()
	}
	if s.Transport.Family != "ipc" && s.Transport.Family != "scoped_cli" {
		return bad()
	}
	if !named(s.Transport.AdapterID) || !exactVersion(s.Transport.AdapterVersion) || !sha.MatchString(s.Transport.AdapterSHA256) || !named(s.Transport.EndpointRef) {
		return bad()
	}
	if !exactQXCTLVersion(s.Runtime.QXCTLVersion) || !sha.MatchString(s.Runtime.QXCTLSHA256) || s.Runtime.SBVVersion != knowledgeengine.SBVAdministrationInterfaceVersion || !sha.MatchString(s.Runtime.SBVInstallationSHA256) {
		return bad()
	}
	if !path.IsAbs(s.Job.Directory) || path.Clean(s.Job.Directory) != s.Job.Directory || s.Job.Directory == "/" || !named(s.Job.Directory) || !jobID.MatchString(s.Job.ID) || !sha.MatchString(s.Job.PlanSHA256) {
		return bad()
	}
	if !nullablePositive(s.Resources.Threads) || !nullablePositive(s.Resources.MemoryBytes) || !nullablePositive(s.Resources.DeadlineUnixMS) || !nullableSHA(s.Resources.ObservationSHA256) || s.Resources.Devices == nil || s.Extensions == nil {
		return bad()
	}
	seen := map[string]bool{}
	for _, device := range s.Resources.Devices {
		if !named(device) || seen[device] {
			return bad()
		}
		seen[device] = true
	}
	if !named(s.Authority.Operation) || !named(s.Authority.Resource) || !named(s.Authority.Audience) || !named(s.Authority.Scope) || !nullableSHA(s.Authority.PolicySHA256) {
		return bad()
	}
	return nil
}
func build(s Selection) (Plan, error) {
	if err := validate(s); err != nil {
		return Plan{}, err
	}
	// Normalize to maps so every object's keys are sorted, including struct
	// fields. Quantities are strings; no float conversion participates here.
	// This versioned Go encoding is not a claim of RFC 8785 canonicalization.
	b, err := json.Marshal(s)
	if err != nil {
		return Plan{}, err
	}
	var canonical map[string]any
	if err = json.Unmarshal(b, &canonical); err != nil {
		return Plan{}, err
	}
	b, err = json.Marshal(canonical)
	if err != nil {
		return Plan{}, err
	}
	d := sha256.Sum256(append([]byte("symphony.sov.remote-plan.v1\x00"), b...))
	return Plan{Protocol: "symphony.sov.remote-plan.v1", PlanSHA256: fmt.Sprintf("%x", d), Selection: s, Status: "planned_not_dispatched", Requirements: []string{
		"Verify exact target identity, installed qxctl/SBV and independently installed adapter before dispatch.",
		"Authenticate and authorize at the target; obtain a fresh target-local SSIAG decision where protected authority applies. A controller decision is not a transferable grant.",
		"Verify the immutable native job plan and selected input/output identities on the target. Placement requests do not rewrite that plan.",
		"Report requested, granted and applied resources; observations reserve nothing. No deadline or resource ceiling is implied by null.",
		"After transport loss, inspect/reconcile this exact job and plan before any further action. Never infer not-admitted from a timeout or missing response.",
		"Cancellation is a separate generation-checked request; transport closure does not prove worker termination. Native checkpoint/resume is unavailable in this profile.",
		"Preserve native result digests, access projections, completeness and snapshot-bound cursors through transfer; retain typed unavailable evidence.",
	}}, nil
}
func MakePlan(raw []byte) (Plan, error) {
	var s Selection
	if err := decode(raw, &s); err != nil {
		return Plan{}, err
	}
	return build(s)
}

// Assess classifies supplied observations for a replacement controller. It is
// deliberately not evidence authentication, current-state discovery or dispatch.
func Assess(raw []byte) (Assessment, error) {
	var in AssessmentInput
	if err := decode(raw, &in); err != nil {
		return Assessment{}, err
	}
	p, err := build(in.Selection)
	if err != nil {
		return Assessment{}, err
	}
	bad := func() (Assessment, error) { return Assessment{}, fmt.Errorf("SOV observation binding mismatch") }
	if in.Protocol != AssessmentProtocol || in.ExpectedPlanSHA256 != p.PlanSHA256 {
		return bad()
	}
	if in.TransportOutcome != "not_dispatched" && in.TransportOutcome != "response_lost" && in.TransportOutcome != "response_received" {
		return bad()
	}
	a := Assessment{Protocol: "symphony.sov.remote-assess.v1", PlanSHA256: p.PlanSHA256, EvidenceStatus: "supplied_unverified", TransportOutcome: in.TransportOutcome, RecommendedAction: "inspect_same_job", Observation: in.Observation}
	if reflect.DeepEqual(bytes.TrimSpace(in.Observation), []byte("null")) {
		if in.ObservedTargetSHA256 != nil || in.TransportOutcome == "response_received" {
			return bad()
		}
		if in.TransportOutcome == "not_dispatched" {
			a.RecommendedAction = "qualify_target_and_authority"
		}
		return a, nil
	}
	if in.ObservedTargetSHA256 == nil || *in.ObservedTargetSHA256 != in.Selection.Target.IdentitySHA256 || in.TransportOutcome == "not_dispatched" {
		return bad()
	}
	j := in.Selection.Job
	if err := knowledgeengine.ValidateSBVJobObservation(in.Observation, j.Directory, j.ID, j.PlanSHA256); err != nil {
		return bad()
	}
	var o map[string]any
	_ = json.Unmarshal(in.Observation, &o)
	// A supplied prior observation cannot settle a subsequently lost response.
	if in.TransportOutcome == "response_lost" {
		return a, nil
	}
	switch {
	case o["execution_lock_held"] == true:
		a.RecommendedAction = "observe_same_job"
	case o["phase"] == "completed" || o["phase"] == "completed_partial":
		a.RecommendedAction = "inspect_bound_result"
	case o["phase"] == "queued" && o["admitted"] == false && o["cancellation_requested"] == false:
		a.RecommendedAction = "qualify_target_and_authority"
	default:
		a.RecommendedAction = "reconcile_same_job"
	}
	return a, nil
}
