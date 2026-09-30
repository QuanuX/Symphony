package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/snvstate"
	"github.com/spf13/cobra"
)

const featureSNVAdministration = "ssfv:symphony:qxctl-snv-administration"
const snvStateResultProtocol = "symphony.qxctl.snv-state-result.v1"
const snvEvidenceResultProtocol = "symphony.qxctl.snv-evidence-result.v1"
const snvProposalProtocol = "symphony.qxctl.snv-state-proposal.v1"

type snvOptions struct {
	prefix, version, input, owner, operation, stateRoot, topsID, viewID, operationID, mode, baseline, expectedHead string
	offset, limit, authorizationOffset, authorizationLimit                                                         int64
	expectedJournal                                                                                                string
}
type snvLeafSpec struct{ path, owner, native, inputProtocol, outputProtocol, interaction string }

var snvLeaves = []snvLeafSpec{
	{"observe", "local-observer", "observe", "symphony.snv.local-observe-input.v1", "symphony.snv.local-observe.v1", "inspect"},
	{"identity.validate", "sniv", "identity_validate", "symphony.sniv.identity-validate-input.v1", "symphony.sniv.identity-validate.v1", "validate"},
	{"resources.validate", "snrv", "resources_validate", "symphony.snrv.resources-validate-input.v1", "symphony.snrv.resources-validate.v1", "validate"},
	{"clusters.validate", "sciv", "sciv_validate", "symphony.snv.sciv.evidence.v1", "symphony.snv.sciv.result.v1", "validate"},
	{"names.validate", "scnv", "names_validate", "scnv.names.v1", "scnv.names-result.v1", "validate"},
	{"names.resolve", "scnv", "names_resolve", "scnv.resolve.v1", "scnv.resolve-result.v1", "query"},
	{"inspect", "snv", "snv_inspect", "symphony.snv.inspect-input.v1", "symphony.snv.inspect.v1", "inspect"},
	{"evidence.prepare", "snv", "snv_evidence_plan", "symphony.snv.evidence-plan-input.v1", snvEvidenceResultProtocol, "propose"},
	{"evidence.commit", "snv", "snv_evidence_plan", "", snvEvidenceResultProtocol, "invoke"},
	{"evidence.status", "snv", "", "", snvEvidenceResultProtocol, "inspect"},
	{"state.plan", "snv", "snv_state_plan", "symphony.snv.state-plan-input.v1", snvProposalProtocol, "propose"},
	{"state.apply", "snv", "snv_state_reduce", snvProposalProtocol, snvStateResultProtocol, "apply"},
	{"state.status", "snv", "", "", snvStateResultProtocol, "inspect"},
	{"state.recover", "snv", "snv_state_reduce", "", snvStateResultProtocol, "recover"},
	{"schema", "", "", "", "symphony.qxctl.snv-schema.v1", "discover"},
	{"template", "", "", "", "symphony.qxctl.snv-template.v1", "discover"},
}

func newSNVCommand() *cobra.Command {
	root := structural("snv", errUsageOnly)
	groups := map[string]*cobra.Command{}
	for _, spec := range snvLeaves {
		parts := strings.Split(spec.path, ".")
		parent := root
		if len(parts) == 2 {
			parent = groups[parts[0]]
			if parent == nil {
				parent = structural(parts[0], errUsageOnly)
				groups[parts[0]] = parent
				root.AddCommand(parent)
			}
		}
		parent.AddCommand(newSNVLeaf(parts[len(parts)-1], spec))
	}
	return root
}
func newSNVLeaf(name string, spec snvLeafSpec) *cobra.Command {
	o := snvOptions{owner: spec.owner, mode: "projection", limit: 128, authorizationLimit: 16}
	c := &cobra.Command{Use: name, Short: "Operate exact independently installed SNV evidence", Args: usageOnlyArgs, RunE: func(*cobra.Command, []string) error {
		if e := runSNV(spec, o); e != nil {
			return snvSafeError(e)
		}
		return nil
	}}
	c.Flags().StringVar(&o.prefix, "prefix", "", "exact owner installation prefix")
	c.Flags().StringVar(&o.version, "version", "", "exact owner release; no implicit upgrade")
	c.Flags().Bool("json", false, "structured evidence and machine-readable failures")
	_ = c.MarkFlagRequired("prefix")
	_ = c.MarkFlagRequired("version")
	if spec.path == "schema" || spec.path == "template" {
		c.Flags().StringVar(&o.owner, "owner", "", "exact owner: sniv, snrv, sciv, scnv, snv, local-observer")
		c.Flags().StringVar(&o.operation, "operation", "", "exact advertised native operation")
		_ = c.MarkFlagRequired("owner")
		_ = c.MarkFlagRequired("operation")
	} else {
		if spec.path == "inspect" || spec.path == "state.apply" || spec.path == "state.plan" || spec.path == "evidence.prepare" || strings.HasSuffix(spec.path, ".validate") || spec.path == "names.resolve" || spec.path == "observe" {
			c.Flags().StringVar(&o.input, "input", "", "bounded no-follow public JSON input")
		}
		if spec.path == "state.plan" || spec.path == "evidence.prepare" || strings.HasSuffix(spec.path, ".validate") || spec.path == "names.resolve" || spec.path == "observe" {
			_ = c.MarkFlagRequired("input")
		}
		if strings.HasPrefix(spec.path, "state.") || strings.HasPrefix(spec.path, "evidence.") || spec.path == "inspect" {
			c.Flags().StringVar(&o.stateRoot, "state-root", "", "explicit private local state root")
			if spec.path != "inspect" {
				_ = c.MarkFlagRequired("state-root")
			}
		}
		if strings.HasPrefix(spec.path, "state.") || spec.path == "inspect" {
			c.Flags().StringVar(&o.topsID, "tops-id", "", "exact TOPS identity")
			c.Flags().StringVar(&o.viewID, "view-id", "", "caller-selected named view")
			if spec.path != "inspect" {
				_ = c.MarkFlagRequired("tops-id")
				_ = c.MarkFlagRequired("view-id")
			}
		}
		if spec.path == "evidence.commit" || spec.path == "evidence.status" || spec.path == "state.apply" || spec.path == "state.status" || spec.path == "state.recover" {
			c.Flags().StringVar(&o.operationID, "operation-id", "", "exact retained operation identity")
			if spec.path == "evidence.commit" || spec.path == "state.recover" {
				_ = c.MarkFlagRequired("operation-id")
			}
		}
		if spec.path == "state.status" {
			c.Flags().Int64Var(&o.authorizationOffset, "authorization-offset", 0, "prior authorization history offset for an exact attempt")
			c.Flags().Int64Var(&o.authorizationLimit, "authorization-limit", 16, "prior authorization page limit,1..128")
			c.Flags().StringVar(&o.expectedJournal, "expected-journal-digest", "", "exact captured journal digest; required for later authorization pages")
		}
		if spec.path == "inspect" {
			c.Flags().StringVar(&o.mode, "mode", "projection", "named-view projection, replay, diff, history, export_manifest or export_chunk")
			c.Flags().StringVar(&o.baseline, "baseline-input", "", "explicit baseline bundle for named-view diff")
			c.Flags().StringVar(&o.expectedHead, "expected-head-digest", "", "exact captured head digest; required for named-view export chunks and later history pages")
			c.Flags().Int64Var(&o.offset, "offset", 0, "snapshot-bound history offset or export chunk index")
			c.Flags().Int64Var(&o.limit, "limit", 128, "bounded page limit")
		}
	}
	c.SetFlagErrorFunc(func(*cobra.Command, error) error { return errUsageOnly })
	s := commandSpec("snv."+spec.path, featureSNVAdministration, spec.interaction)
	if spec.owner != "" {
		feature := "ssfv:symphony:" + spec.owner + "-engine"
		if spec.owner == "local-observer" {
			feature = "ssfv:symphony:snv-local-observer"
		}
		s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: feature, Interaction: spec.interaction})
	}
	if spec.native != "" {
		backendOwner := spec.owner
		if spec.owner == "local-observer" {
			backendOwner = "snv-local-observer"
		}
		s.BackendOperationIDs = []string{"engop:symphony:" + backendOwner + "." + strings.ReplaceAll(spec.native, "_", "-")}
	}
	if spec.path == "schema" || spec.path == "template" {
		s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:snv-local-observer", Interaction: "discover"})
	}
	if spec.path == "clusters.validate" {
		s.BackendOperationIDs = append(s.BackendOperationIDs, "engop:symphony:sciv.sciv-transition")
	}
	if spec.inputProtocol != "" {
		s.InputProtocols = []string{spec.inputProtocol}
	}
	s.OutputProtocols = []string{spec.outputProtocol}
	s.ResultValidationProtocols = s.OutputProtocols
	if spec.path == "clusters.validate" {
		s.InputProtocols = append(s.InputProtocols, "symphony.snv.sciv.transition.v1")
		s.OutputProtocols = append(s.OutputProtocols, "symphony.snv.sciv.transition-result.v1")
		s.ResultValidationProtocols = s.OutputProtocols
		s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:sciv-engine", Interaction: "propose"})
	}
	if spec.path == "identity.validate" || spec.path == "resources.validate" {
		s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:" + spec.owner + "-engine", Interaction: "propose"})
	}
	if spec.path == "clusters.validate" || spec.path == "names.validate" || spec.path == "names.resolve" {
		s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:" + spec.owner + "-engine", Interaction: "inspect"})
	}
	if spec.path == "names.validate" {
		s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:scnv-engine", Interaction: "propose"})
	}
	if spec.path == "inspect" {
		s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:snv-engine", Interaction: "query"})
		s.OutputProtocols = append(s.OutputProtocols, "symphony.qxctl.snv-view-inspect.v1")
		s.ResultValidationProtocols = s.OutputProtocols
	}
	if spec.path == "evidence.prepare" || spec.path == "evidence.commit" {
		s.Mutability = "evidence_only"
		s.RecoveryCommandID = stringPointer("qxcmd:symphony:snv.evidence.commit")
	}
	if spec.path == "state.plan" {
		s.Mutability = "proposal_only"
	}
	if spec.path == "state.apply" || spec.path == "state.recover" {
		s.Mutability = "permission_backed_mutation"
		s.AuthorityMode = "target_host_permission"
		s.RecoveryCommandID = stringPointer("qxcmd:symphony:snv.state.recover")
		s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: backendFeatureSSIAG, Interaction: "invoke"})
	}
	commandregistry.Attach(c, s)
	return c
}
func snvSafeError(e error) error {
	if errors.Is(e, errUsageOnly) {
		return errUsageOnly
	}
	var boundary *snvstate.BoundaryError
	if errors.As(e, &boundary) && safeSCVEngineCode(boundary.Code) != nil {
		// A publication guard retains its specific typed cause when known.
		if boundary.Code == "snv.recovery_required" && boundary.Cause != nil {
			var process *knowledgeengine.ProcessError
			if errors.As(boundary.Cause, &process) && safeSCVEngineCode(process.Code) != nil {
				return &knowledgeengine.ProcessError{Code: process.Code, Message: "SNV operation refused; inspect exact retained attempt"}
			}
			var inner *snvstate.BoundaryError
			if errors.As(boundary.Cause, &inner) && safeSCVEngineCode(inner.Code) != nil {
				return &knowledgeengine.ProcessError{Code: inner.Code, Message: "SNV operation refused; inspect exact retained attempt"}
			}
		}
		return &knowledgeengine.ProcessError{Code: boundary.Code, Message: "SNV operation refused; inspect exact retained attempt"}
	}
	var pe *knowledgeengine.ProcessError
	if errors.As(e, &pe) && safeSCVEngineCode(pe.Code) != nil {
		return &knowledgeengine.ProcessError{Code: pe.Code, Message: "SNV operation refused"}
	}
	return fmt.Errorf("SNV command refused; inspect exact installation, public input, retained state and authority")
}
func snvInvoke(o snvOptions, owner, operation string, input any) (json.RawMessage, error) {
	cwd, e := os.Getwd()
	if e != nil {
		return nil, e
	}
	raw, e := knowledgeengine.SCVCanonical(input)
	if e != nil {
		return nil, e
	}
	r, e := knowledgeengine.InvokeSNV(context.Background(), o.prefix, o.version, cwd, owner, operation, raw)
	return r.Result, e
}
func snvSeal(m map[string]any) (json.RawMessage, error) {
	// Embedded structs and RawMessages must use the same sorted object
	// representation that a reader reconstructs from the public JSON capsule.
	canonical, e := snvValueCanonical(m)
	if e != nil {
		return nil, e
	}
	object, e := snvObject(canonical)
	if e != nil {
		return nil, e
	}
	digest, e := knowledgeengine.SCVDigest(object)
	if e != nil {
		return nil, e
	}
	object["digest"] = digest
	raw, e := knowledgeengine.SCVCanonical(object)
	if e == nil && len(raw) > 4*1024*1024 {
		return nil, fmt.Errorf("SNV command result exceeds bound")
	}
	return raw, e
}
func snvObject(raw json.RawMessage) (map[string]any, error) {
	return knowledgeengine.ParseSNVObject(raw)
}
func snvSame(a, b any) bool {
	x, e := snvValueCanonical(a)
	if e != nil {
		return false
	}
	y, e := snvValueCanonical(b)
	return e == nil && bytes.Equal(x, y)
}
func snvValueCanonical(value any) ([]byte, error) {
	raw, e := json.Marshal(value)
	if e != nil {
		return nil, e
	}
	var object any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if e = d.Decode(&object); e != nil {
		return nil, e
	}
	return knowledgeengine.SCVCanonical(object)
}
func snvBundle(root, digest string) (json.RawMessage, error) {
	store, e := snvstate.NewEvidence(root)
	if e != nil {
		return nil, e
	}
	var raw json.RawMessage
	e = store.WithRead(func(tx *snvstate.Transaction) error {
		var ok bool
		raw, ok = tx.Bundle(digest)
		if !ok {
			return snvstate.Refusal("snv.evidence_missing", nil)
		}
		return nil
	})
	return raw, e
}
func snvProveBundle(o snvOptions, input json.RawMessage) error {
	m, e := snvObject(input)
	if e != nil {
		return e
	}
	if m["change_kind"] == "unselect" {
		if m["bundle"] != nil {
			return fmt.Errorf("unselect must not carry a bundle")
		}
		return nil
	}
	hash, e := knowledgeengine.SCVDigest(m["bundle"])
	if e != nil {
		return e
	}
	retained, e := snvBundle(o.stateRoot, hash)
	if e != nil {
		return e
	}
	if !snvSame(retained, m["bundle"]) {
		return snvstate.Refusal("snv.candidate_drift", nil)
	}
	return nil
}

type snvAuthorizationPage struct {
	offset, limit   int64
	expectedJournal string
}

func snvAttemptSummary(a snvstate.Attempt, page snvAuthorizationPage) (map[string]any, error) {
	count := int64(len(a.PriorAuthorizations))
	if page.offset < 0 || page.offset > count || page.limit < 1 || page.limit > 128 {
		return nil, snvstate.Refusal("snv.invalid_input", nil)
	}
	end := page.offset + page.limit
	if end > count {
		end = count
	}
	history := append([]json.RawMessage{}, a.PriorAuthorizations[page.offset:end]...)
	var next any
	if end < count {
		next = end
	}
	return map[string]any{"kind": a.Kind, "operation_id": a.OperationID, "intent_digest": a.IntentDigest, "status": a.Status, "installation": a.Installation, "correlation_id": a.CorrelationID, "authorization": a.Authorization, "prior_authorizations": history, "prior_authorization_count": count, "prior_authorization_offset": page.offset, "prior_authorization_limit": page.limit, "next_prior_authorization_offset": next}, nil
}
func snvStoreResult(op string, tx *snvstate.Transaction, id string, owner json.RawMessage, options ...snvAuthorizationPage) (json.RawMessage, error) {
	doc := tx.Snapshot()
	page := snvAuthorizationPage{limit: 16}
	if len(options) > 0 {
		page = options[0]
	}
	if page.offset > 0 && page.expectedJournal == "" {
		return nil, errUsageOnly
	}
	if page.expectedJournal != "" && page.expectedJournal != doc.Digest {
		return nil, &knowledgeengine.ProcessError{Code: "snv.snapshot_conflict", Message: "SNV history snapshot differs"}
	}
	var attempt any
	if id != "" {
		a, ok := tx.Attempt(id)
		if !ok {
			return nil, snvstate.Refusal("snv.evidence_missing", nil)
		}
		summary, e := snvAttemptSummary(a, page)
		if e != nil {
			return nil, e
		}
		attempt = summary
	}
	hashes := []string{}
	for hash := range doc.Bundles {
		hashes = append(hashes, hash)
	}
	sort.Strings(hashes)
	resultProtocol := snvStateResultProtocol
	if doc.Kind == "evidence" {
		resultProtocol = snvEvidenceResultProtocol
	}
	var journal any
	if doc.Digest != "" {
		journal = doc.Digest
	}
	return snvSeal(map[string]any{"protocol": resultProtocol, "operation": op, "tops_id": doc.TOPSID, "view_id": doc.ViewID, "state_digest": doc.StateDigest, "journal_digest": journal, "head": doc.Head, "attempt": attempt, "operation_count": len(doc.Operations), "bundle_digests": hashes, "owner_result": owner, "authorization_audit": "ssiag_policy_decision_only", "head_write_stav_receipt": nil, "canonical_apply_enabled": false})
}

// Bound the complete emitted wrapper before writing any success bytes. A write
// that already committed remains observable through its exact status/retry.
func snvPrintJSON(value any) error {
	encoded, e := json.MarshalIndent(value, "", "  ")
	if e != nil {
		return e
	}
	if len(encoded)+1 > 4*1024*1024 {
		return snvstate.Refusal("snv.capacity_exceeded", nil)
	}
	fmt.Println(string(encoded))
	return nil
}
func runSNV(spec snvLeafSpec, o snvOptions) error {
	if spec.path == "schema" || spec.path == "template" {
		inst, raw, e := knowledgeengine.SNVResource(o.prefix, o.version, o.owner, o.operation, spec.path == "template")
		if e != nil {
			return e
		}
		return snvPrintJSON(map[string]any{"protocol": "symphony.qxctl.snv-" + spec.path + ".v1", "owner": o.owner, "operation": o.operation, "installation": inst, spec.path: raw})
	}
	inst, e := knowledgeengine.InspectSNV(o.prefix, o.version, o.owner)
	if e != nil {
		return e
	}
	if strings.HasPrefix(spec.path, "evidence.") {
		return runSNVEvidence(spec, o, inst)
	}
	if strings.HasPrefix(spec.path, "state.") {
		return runSNVState(spec, o, inst)
	}
	if spec.path == "inspect" {
		return runSNVInspect(o)
	}
	raw, e := knowledgeengine.ReadPayload(o.input)
	if e != nil {
		return e
	}
	operation := spec.native
	if spec.path == "clusters.validate" {
		body, e := snvObject(raw)
		if e != nil {
			return e
		}
		if body["protocol"] == "symphony.snv.sciv.transition.v1" {
			operation = "sciv_transition"
		}
	}
	r, e := snvInvoke(o, spec.owner, operation, json.RawMessage(raw))
	if e != nil {
		return e
	}
	return snvPrintJSON(r)
}

// Continuation reads bind the captured immutable head rather than reusing an
// offset against whatever view happens to be selected on the next invocation.
func snvInspectHeadBinding(o snvOptions, head map[string]any) error {
	if (o.mode == "export_chunk" || (o.mode == "history" && o.offset > 0)) && o.expectedHead == "" {
		return errUsageOnly
	}
	if o.expectedHead != "" && head["digest"] != o.expectedHead {
		return &knowledgeengine.ProcessError{Code: "snv.snapshot_conflict", Message: "SNV captured head differs from expected revision"}
	}
	return nil
}
func runSNVInspect(o snvOptions) error {
	if o.input != "" {
		if o.topsID != "" || o.viewID != "" || o.stateRoot != "" || o.expectedHead != "" {
			return errUsageOnly
		}
		raw, e := knowledgeengine.ReadPayload(o.input)
		if e != nil {
			return e
		}
		r, e := snvInvoke(o, "snv", "snv_inspect", json.RawMessage(raw))
		if e != nil {
			return e
		}
		return snvPrintJSON(r)
	}
	store, e := snvstate.NewView(o.stateRoot, o.topsID, o.viewID)
	if e != nil {
		return e
	}
	var head json.RawMessage
	e = store.WithRead(func(tx *snvstate.Transaction) error { head = tx.Current(); return nil })
	if e != nil {
		return e
	}
	h, e := snvObject(head)
	if e != nil {
		return snvstate.Refusal("snv.recovery_required", e)
	}
	if h["tombstone"] == true {
		return snvstate.Refusal("snv.evidence_missing", nil)
	}
	if e := snvInspectHeadBinding(o, h); e != nil {
		return e
	}
	hash, ok := h["bundle_digest"].(string)
	if !ok {
		return snvstate.Refusal("snv.recovery_required", nil)
	}
	bundle, e := snvBundle(o.stateRoot, hash)
	if e != nil {
		return e
	}
	var baseline json.RawMessage = json.RawMessage("null")
	if o.baseline != "" {
		baseline, e = knowledgeengine.ReadPayload(o.baseline)
		if e != nil {
			return e
		}
	}
	r, e := snvInvoke(o, "snv", "snv_inspect", map[string]any{"protocol": "symphony.snv.inspect-input.v1", "mode": o.mode, "bundle": bundle, "baseline": baseline, "offset": o.offset, "limit": o.limit})
	if e != nil {
		return e
	}
	output, e := snvSeal(map[string]any{"protocol": "symphony.qxctl.snv-view-inspect.v1", "captured_head": head, "owner_result": r})
	if e != nil {
		return e
	}
	return snvPrintJSON(output)
}
func runSNVEvidence(spec snvLeafSpec, o snvOptions, inst knowledgeengine.Installation) error {
	store, e := snvstate.NewEvidence(o.stateRoot)
	if e != nil {
		return e
	}
	var input, plan json.RawMessage
	if spec.path == "evidence.prepare" {
		input, e = knowledgeengine.ReadPayload(o.input)
		if e != nil {
			return e
		}
		plan, e = snvInvoke(o, "snv", "snv_evidence_plan", input)
		if e != nil {
			return e
		}
	}
	var output json.RawMessage
	withStore := store.WithLock
	if spec.path == "evidence.status" {
		withStore = store.WithRead
	}
	e = withStore(func(tx *snvstate.Transaction) error {
		id := o.operationID
		if spec.path == "evidence.prepare" {
			a, e := snvstate.NewAttempt("evidence", input, plan, json.RawMessage("null"), inst)
			if e != nil {
				return e
			}
			if _, e = tx.Prepare(a); e != nil {
				return e
			}
			id = a.OperationID
		}
		if spec.path == "evidence.commit" {
			a, ok := tx.Attempt(id)
			if !ok || a.Installation != inst {
				return fmt.Errorf("SNV retention requires exact original installation and intent")
			}
			guard := func() error {
				current, e := knowledgeengine.InspectSNV(o.prefix, o.version, "snv")
				if e != nil || current != inst {
					return fmt.Errorf("SNV owner changed")
				}
				replay, e := snvInvoke(o, "snv", "snv_evidence_plan", a.Input)
				if e != nil || !snvSame(replay, a.Plan) {
					return fmt.Errorf("SNV retained input replay differs")
				}
				return nil
			}
			if e := guard(); e != nil {
				return e
			}
			if e := tx.CommitEvidence(id, guard); e != nil {
				return e
			}
			plan = a.Plan
		}
		var e error
		output, e = snvStoreResult(spec.path, tx, id, plan)
		return e
	})
	if e != nil {
		return e
	}
	return snvPrintJSON(output)
}
