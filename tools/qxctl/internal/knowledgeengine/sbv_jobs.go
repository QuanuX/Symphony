package knowledgeengine

import (
	"fmt"
	"strconv"
	"strings"
)

// ValidateSBVJobObservation checks the native job-inspect contract and immutable
// job binding. It does not authenticate a remote origin or prove freshness.
func ValidateSBVJobObservation(raw []byte, directory, jobID, planSHA256 string) error {
	return validateSBVJobResult("job_inspect", map[string]any{"directory": directory, "job_id": jobID, "expected_plan_sha256": planSHA256}, raw)
}

// ValidateSBVOperationResult validates a selected remote native response using
// the same owner checks as direct invocation. This authenticates no transport.
func ValidateSBVOperationResult(operation string, request, result []byte) (err error) {
	// Direct invocation first validates installed owner schemas. At this external
	// response boundary both request and reply are untrusted. Never let a type
	// assertion in an owner validator turn a malformed pair into a process panic.
	defer func() {
		if recover() != nil {
			err = fmt.Errorf("invalid remote SBV request/result shape")
		}
	}()
	switch operation {
	case "job_inspect", "job_execute", "job_cancel", "job_reconcile", "result_inspect", "result_query", "bundle_inspect", "bundle_query", "bundle_verify":
	default:
		return fmt.Errorf("unsupported remote SBV result operation")
	}
	p, err := sqavObject(request, maxRequestBytes)
	if err != nil {
		return err
	}
	if p["protocol"] != "symphony.sbv."+strings.ReplaceAll(operation, "_", "-")+"-input.v1" {
		return fmt.Errorf("remote SBV request protocol mismatch")
	}
	if operation == "result_query" {
		for _, key := range []string{"path", "expected_sha256", "pointer", "limit", "cursor"} {
			if _, ok := p[key].(string); !ok {
				return fmt.Errorf("remote SBV query field missing or invalid")
			}
		}
	}
	return validateSBVResult(operation, p, result)
}

// Journal references are immutable native result snapshots. The job plan binds
// the full child selection; a status receipt does not assert remote authority.
func validateSBVJobResult(op string, p map[string]any, raw []byte) error {
	m, err := sqavObject(raw, maxResponseBytes)
	bad := func() error { return fmt.Errorf("SBV durable job result mismatch") }
	if err != nil || len(m) != 19 || m["protocol"] != sbvAdministrationInterfaceOutputs[op] || m["engine_version"] != SBVAdministrationInterfaceVersion || m["directory"] != p["directory"] || m["job_id"] != p["job_id"] || m["automatic_retry"] != false {
		return bad()
	}
	for _, key := range []string{"protocol", "engine_version", "directory", "job_id", "plan_sha256", "generation", "phase", "cancellation_requested", "cancellation_observed", "execution_lock_held", "admitted", "automatic_retry", "result_receipt", "error_code", "plan_reference", "state_reference", "outcome_reference", "claim_reference", "cancellation_reference"} {
		if _, exists := m[key]; !exists {
			return bad()
		}
	}
	digest, ok := m["plan_sha256"].(string)
	if !ok || !sbvSHA.MatchString(digest) || (op != "job_create" && digest != p["expected_plan_sha256"]) {
		return bad()
	}
	generation, ok := m["generation"].(string)
	n, e := strconv.ParseUint(generation, 10, 64)
	if !ok || e != nil || strconv.FormatUint(n, 10) != generation {
		return bad()
	}
	for _, k := range []string{"cancellation_requested", "cancellation_observed", "execution_lock_held", "admitted"} {
		if _, ok := m[k].(bool); !ok {
			return bad()
		}
	}
	phase, ok := m["phase"].(string)
	if !ok {
		return bad()
	}
	switch phase {
	case "queued", "running", "completed", "completed_partial", "cancelled", "failed", "unknown":
	default:
		return bad()
	}
	// A crash after claim fsync but before state replacement legitimately leaves
	// a queued recorded phase with admitted=true. Reconciliation must stay usable.
	if phase != "queued" && phase != "cancelled" && m["admitted"] != true {
		return bad()
	}
	directory, ok := m["directory"].(string)
	if !ok {
		return bad()
	}
	ref := func(v any, name string, expected string) bool {
		r, ok := v.(map[string]any)
		if !ok || len(r) != 3 || r["path"] != directory+"/"+name || r["pointer"] != "" {
			return false
		}
		d, ok := r["expected_sha256"].(string)
		return ok && sbvSHA.MatchString(d) && (expected == "" || d == expected)
	}
	if !ref(m["plan_reference"], "plan.json", digest) {
		return bad()
	}
	state, ok := m["state_reference"].(map[string]any)
	if !ok {
		return bad()
	}
	sd, ok := state["expected_sha256"].(string)
	if !ok || !ref(state, "state-"+sd+".json", sd) {
		return bad()
	}
	for key, name := range map[string]string{"claim_reference": "claim.json", "cancellation_reference": "cancel.json"} {
		if m[key] != nil && !ref(m[key], name, "") {
			return bad()
		}
	}
	if (m["claim_reference"] != nil) != m["admitted"].(bool) || (m["cancellation_reference"] != nil) != m["cancellation_requested"].(bool) {
		return bad()
	}
	if m["outcome_reference"] != nil && !ref(m["outcome_reference"], "outcome.json", "") {
		return bad()
	}
	if (phase == "completed" || phase == "completed_partial" || phase == "failed") && m["outcome_reference"] == nil {
		return bad()
	}
	if m["error_code"] != nil {
		if s, ok := m["error_code"].(string); !ok || s == "" {
			return bad()
		}
	}
	if m["result_receipt"] != nil {
		r, ok := m["result_receipt"].(map[string]any)
		if !ok || m["outcome_reference"] == nil {
			return bad()
		}
		protocol, ok := r["protocol"].(string)
		if !ok || !strings.HasPrefix(protocol, "symphony.sbv.") {
			return bad()
		}
		if op == "job_create" && protocol != "symphony.sbv."+strings.ReplaceAll(fmt.Sprint(p["operation"]), "_", "-")+".v1" {
			return bad()
		}
		if phase == "completed" && r["status"] != "completed" {
			return bad()
		}
		if phase == "completed_partial" && r["status"] != "partial" {
			return bad()
		}
	} else if phase == "completed" || phase == "completed_partial" {
		return bad()
	}
	return nil
}
