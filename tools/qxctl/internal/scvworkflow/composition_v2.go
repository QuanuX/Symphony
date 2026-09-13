package scvworkflow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/scvcorpus"
)

// CompositionRunV2 is local evidence bookkeeping, not a selected authority head.
// Its separate protocol leaves the earlier workflow journal unchanged.
type CompositionRunV2 struct {
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

// CompositionPlanV2 explicitly admits bundle transport. The unchanged v1
// planner validates the shared orchestration fields without reinterpreting v1.
func CompositionPlanV2(input map[string]any) ([]string, error) {
	if err := exactCompositionFields(input, "protocol", "transport", "operation_id", "pack_evaluations", "pack_evaluation_refs", "interpretation_refs", "knowledge_refs", "composition", "prior_composition_ref"); err != nil {
		return nil, err
	}
	if input["protocol"] != "symphony.qxctl.scv-composition-workflow-request.v2" || input["transport"] != "bundle" {
		return nil, fmt.Errorf("explicit v2 bundle workflow request required")
	}
	legacy := make(map[string]any, 7)
	for k, v := range input {
		if k != "protocol" && k != "transport" {
			legacy[k] = v
		}
	}
	return CompositionPlan(legacy)
}
func NewCompositionRunV2(store Store, installed knowledgeengine.Installation, request json.RawMessage) (CompositionRunV2, error) {
	r := CompositionRunV2{Protocol: "symphony.qxctl.scv-composition-workflow-run.v2", Root: store.Root, TOPSID: store.TOPSID, Installation: installed, Request: request, Stages: map[string]Checkpoint{}}
	if !knowledgeengine.SCVSupports(installed.Version, "composition_bundle_workflow") {
		return r, fmt.Errorf("composition journal requires an exact supporting release")
	}
	if err := knowledgeengine.ValidateSCVBundleText(request); err != nil {
		return r, err
	}
	input, err := Decode(request)
	if err != nil {
		return r, err
	}
	if _, err = CompositionPlanV2(input); err != nil {
		return r, err
	}
	r.OperationID = input["operation_id"].(string)
	if _, err = New(store.Root, store.TOPSID); err != nil {
		return r, err
	}
	intent := map[string]any{"protocol": "symphony.qxctl.scv-composition-workflow-intent.v2", "operation_id": r.OperationID, "root": r.Root, "tops_id": r.TOPSID, "installation": installed, "request": request}
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
func ReadCompositionRunV2(raw []byte, store Store, operation string) (CompositionRunV2, error) {
	var r CompositionRunV2
	if err := knowledgeengine.ValidateJSONObject(raw, MaxBytes); err != nil {
		return r, err
	}
	if err := knowledgeengine.ValidateSCVBundleUnicode(raw); err != nil {
		return r, err
	}
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
	if r.Protocol != "symphony.qxctl.scv-composition-workflow-run.v2" || r.OperationID != operation || r.Root != store.Root || r.TOPSID != store.TOPSID || r.Stages == nil || len(r.Stages) > 18 {
		return r, fmt.Errorf("invalid composition workflow journal identity")
	}
	expected, err := NewCompositionRunV2(store, r.Installation, r.Request)
	if err != nil || r.IntentDigest != expected.IntentDigest || expected.OperationID != operation {
		return r, fmt.Errorf("composition workflow intent mismatch")
	}
	input, err := Decode(r.Request)
	if err != nil {
		return r, err
	}
	plan, err := CompositionPlanV2(input)
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
