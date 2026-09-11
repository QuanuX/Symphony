package scvworkflow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/scvcorpus"
)

// CompositionRun is local evidence bookkeeping, not a selected authority head.
// Its separate protocol leaves the earlier workflow journal unchanged.
type CompositionRun struct {
	Protocol     string                       `json:"protocol"`
	OperationID  string                       `json:"operation_id"`
	Root         string                       `json:"root"`
	TOPSID       string                       `json:"tops_id"`
	Installation knowledgeengine.Installation `json:"installation"`
	Request      json.RawMessage              `json:"request"`
	IntentDigest string                       `json:"intent_digest"`
	Stages       map[string]Checkpoint        `json:"stages"`
	Complete     bool                         `json:"complete"`
	Digest       string                       `json:"digest"`
}

func exactCompositionFields(m map[string]any, keys ...string) error {
	if len(m) != len(keys) {
		return fmt.Errorf("unexpected composition workflow fields")
	}
	for _, k := range keys {
		if _, ok := m[k]; !ok {
			return fmt.Errorf("missing composition workflow field %s", k)
		}
	}
	return nil
}
func CompositionReferences(v any) ([]string, error) {
	list, ok := v.([]any)
	if !ok || len(list) > 16 {
		return nil, fmt.Errorf("composition workflow references exceed 16")
	}
	refs := []string{}
	seen := map[string]bool{}
	for _, v := range list {
		s, ok := v.(string)
		if !ok || !scvcorpus.ValidDigest(s) || seen[s] {
			return nil, fmt.Errorf("invalid or duplicate composition workflow reference")
		}
		seen[s] = true
		refs = append(refs, s)
	}
	return refs, nil
}

// CompositionPlan validates orchestration structure only. The selected native
// owners still validate exact pack and composition semantics before completion.
func CompositionPlan(input map[string]any) ([]string, error) {
	if err := exactCompositionFields(input, "operation_id", "pack_evaluations", "pack_evaluation_refs", "interpretation_refs", "knowledge_refs", "composition", "prior_composition_ref"); err != nil {
		return nil, err
	}
	id, ok := input["operation_id"].(string)
	if !ok || !scvcorpus.ValidID(id) {
		return nil, fmt.Errorf("invalid composition workflow operation ID")
	}
	packs, ok := input["pack_evaluations"].([]any)
	if !ok || len(packs) > 16 {
		return nil, fmt.Errorf("pack_evaluations requires at most 16 entries")
	}
	retained, err := CompositionReferences(input["pack_evaluation_refs"])
	if err != nil {
		return nil, err
	}
	if len(packs)+len(retained) > 16 {
		return nil, fmt.Errorf("new and retained pack evaluations exceed 16")
	}
	totalEvidence := len(packs) + len(retained)
	for _, k := range []string{"interpretation_refs", "knowledge_refs"} {
		refs, err := CompositionReferences(input[k])
		if err != nil {
			return nil, err
		}
		totalEvidence += len(refs)
	}
	if totalEvidence > 16 {
		return nil, fmt.Errorf("combined composition evidence exceeds 16 artifacts")
	}
	if input["prior_composition_ref"] != nil {
		s, ok := input["prior_composition_ref"].(string)
		if !ok || !scvcorpus.ValidDigest(s) {
			return nil, fmt.Errorf("invalid prior composition reference")
		}
	}
	composition, ok := input["composition"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing explicit composition request")
	}
	if err = exactCompositionFields(composition, "query_time", "requirements", "slots", "allowed_guarantee_changes", "counterfactuals", "bounds"); err != nil {
		return nil, err
	}
	if _, ok = composition["query_time"].(string); !ok {
		return nil, fmt.Errorf("composition query time must be explicit")
	}
	for _, k := range []string{"requirements", "slots", "allowed_guarantee_changes", "counterfactuals"} {
		if _, ok = composition[k].([]any); !ok {
			return nil, fmt.Errorf("invalid composition array %s", k)
		}
	}
	if _, ok = composition["bounds"].(map[string]any); !ok {
		return nil, fmt.Errorf("missing composition search bounds")
	}
	stages := []string{}
	seen := map[string]bool{}
	for _, raw := range packs {
		p, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid pack evaluation entry")
		}
		if err = exactCompositionFields(p, "evaluation_id", "pack_ref", "evaluation"); err != nil {
			return nil, err
		}
		id, ok := p["evaluation_id"].(string)
		if !ok || !scvcorpus.ValidID(id) || seen[id] {
			return nil, fmt.Errorf("invalid or duplicate pack evaluation ID")
		}
		seen[id] = true
		ref, ok := p["pack_ref"].(string)
		if !ok || !scvcorpus.ValidDigest(ref) {
			return nil, fmt.Errorf("invalid pack reference")
		}
		e, ok := p["evaluation"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("missing pack evaluation input")
		}
		if err = exactCompositionFields(e, "captures", "bindings", "selection_policy", "fixtures"); err != nil {
			return nil, err
		}
		for _, k := range []string{"captures", "bindings", "fixtures"} {
			a, ok := e[k].([]any)
			if !ok || len(a) > 16 {
				return nil, fmt.Errorf("pack evaluation %s exceeds 16", k)
			}
		}
		if _, ok = e["selection_policy"].(map[string]any); !ok {
			return nil, fmt.Errorf("pack evaluation requires explicit policy")
		}
		stages = append(stages, "pack:"+id)
	}
	stages = append(stages, "explore")
	if input["prior_composition_ref"] != nil {
		stages = append(stages, "reassess")
	}
	return stages, nil
}
func NewCompositionRun(store Store, installed knowledgeengine.Installation, request json.RawMessage) (CompositionRun, error) {
	r := CompositionRun{Protocol: "symphony.qxctl.scv-composition-workflow-run.v1", Root: store.Root, TOPSID: store.TOPSID, Installation: installed, Request: request, Stages: map[string]Checkpoint{}}
	if !knowledgeengine.SCVSupports(installed.Version, "composition_workflow") {
		return r, fmt.Errorf("composition journal requires an exact supporting release")
	}
	if err := knowledgeengine.ValidateJSONObject(request, MaxInputBytes); err != nil {
		return r, err
	}
	input, err := Decode(request)
	if err != nil {
		return r, err
	}
	if _, err = CompositionPlan(input); err != nil {
		return r, err
	}
	r.OperationID = input["operation_id"].(string)
	if _, err = New(store.Root, store.TOPSID); err != nil {
		return r, err
	}
	intent := map[string]any{"protocol": "symphony.qxctl.scv-composition-workflow-intent.v1", "operation_id": r.OperationID, "root": r.Root, "tops_id": r.TOPSID, "installation": installed, "request": request}
	raw, err := Canonical(intent)
	if err != nil {
		return r, err
	}
	v, err := Decode(raw)
	if err != nil {
		return r, err
	}
	r.IntentDigest, err = knowledgeengine.SCVDigest(v)
	return r, err
}
func ReadCompositionRun(raw []byte, store Store, operation string) (CompositionRun, error) {
	var r CompositionRun
	if _, err := Digest(raw); err != nil {
		return r, err
	}
	object, err := Decode(raw)
	if err != nil {
		return r, err
	}
	if err = exactCompositionFields(object, "protocol", "operation_id", "root", "tops_id", "installation", "request", "intent_digest", "stages", "complete", "digest"); err != nil {
		return r, err
	}
	stages, ok := object["stages"].(map[string]any)
	if !ok {
		return r, fmt.Errorf("invalid composition stage object")
	}
	for _, raw := range stages {
		checkpoint, ok := raw.(map[string]any)
		if !ok {
			return r, fmt.Errorf("invalid composition checkpoint object")
		}
		if err = exactCompositionFields(checkpoint, "input_ref", "result_ref"); err != nil {
			return r, err
		}
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&r); err != nil {
		return r, err
	}
	// encoding/json accepts null for scalar zero values and case-insensitive
	// struct field names. Require exact semantic round-trip before retaining the
	// original seal so normalization cannot change the object under its digest.
	normalized, err := Canonical(r)
	if err != nil || !Same(raw, normalized) {
		return r, fmt.Errorf("composition journal fields do not retain their exact typed representation")
	}
	if r.Protocol != "symphony.qxctl.scv-composition-workflow-run.v1" || r.OperationID != operation || r.Root != store.Root || r.TOPSID != store.TOPSID || r.Stages == nil || len(r.Stages) > 18 {
		return r, fmt.Errorf("invalid composition workflow journal identity")
	}
	expected, err := NewCompositionRun(store, r.Installation, r.Request)
	if err != nil || r.IntentDigest != expected.IntentDigest || expected.OperationID != operation {
		return r, fmt.Errorf("composition workflow intent mismatch")
	}
	input, err := Decode(r.Request)
	if err != nil {
		return r, err
	}
	plan, err := CompositionPlan(input)
	if err != nil {
		return r, err
	}
	seen := 0
	pending := false
	for _, name := range plan {
		c, ok := r.Stages[name]
		if !ok {
			pending = true
			continue
		}
		seen++
		if pending || !scvcorpus.ValidDigest(c.InputRef) || (c.ResultRef != "" && !scvcorpus.ValidDigest(c.ResultRef)) {
			return r, fmt.Errorf("invalid or out-of-order composition checkpoint")
		}
		if c.ResultRef == "" {
			pending = true
		}
	}
	if seen != len(r.Stages) || r.Complete && (pending || seen != len(plan)) {
		return r, fmt.Errorf("unknown or incomplete composition stages")
	}
	return r, nil
}
