package main

import (
	"encoding/json"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/scvtransfer"
	"path/filepath"
	"strings"
)

type transferRunner struct {
	source, target, reader *graphIndexRunner
	intent                 scvtransfer.Intent
	plan                   map[string]any
	hook                   func(string) error
}

func newTransferRunner(source *graphIndexRunner, input map[string]any) (*transferRunner, error) {
	if !graphIndexExact(input, "plan", "expected_plan_digest") {
		return nil, fmt.Errorf("transfer requires plan and expected_plan_digest")
	}
	raw, err := knowledgeengine.SCVCanonical(input["plan"])
	if err != nil {
		return nil, err
	}
	plan, err := scvtransfer.Decode(raw)
	if err != nil {
		return nil, err
	}
	pin, ok := plan["input"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("native transfer plan required")
	}
	in, _ := knowledgeengine.SCVCanonical(pin)
	if err = knowledgeengine.ValidateSCVGraphIndexResult("transfer_plan", in, raw); err != nil {
		return nil, err
	}
	if plan["digest"] != input["expected_plan_digest"] || plan["disposition"] != "ready" {
		return nil, fmt.Errorf("exact ready plan digest required")
	}
	planned, _ := knowledgeengine.SCVCanonical(pin["source_connector"])
	var plannedSource knowledgeengine.Installation
	if err = json.Unmarshal(planned, &plannedSource); err != nil {
		return nil, err
	}
	if source.connector != plannedSource || pin["tops_id"] != source.options.scv.topsID || pin["namespace"] != source.options.namespace {
		return nil, fmt.Errorf("plan source installation or scope differs")
	}
	targetRoot := pin["target_root"].(string)
	// Nested roots make destination writes source filesystem changes and complicate
	// recovery ownership. This tool's copy boundary requires separate peer roots.
	a, b := source.options.root, targetRoot
	if a == b || withinRoot(a, b) || withinRoot(b, a) {
		return nil, fmt.Errorf("source and target roots must be separate")
	}
	targetRaw, _ := knowledgeengine.SCVCanonical(pin["target_connector"])
	var installation knowledgeengine.Installation
	if err = json.Unmarshal(targetRaw, &installation); err != nil {
		return nil, err
	}
	o := source.options
	o.root = targetRoot
	o.connectorPrefix = installation.Prefix
	o.connectorVersion = installation.Version
	target, err := newGraphIndexRunner("transfer", o)
	if err != nil {
		return nil, err
	}
	if target.connector != installation {
		return nil, fmt.Errorf("exact target installation differs")
	}
	o.connectorPrefix = source.connector.Prefix
	o.connectorVersion = source.connector.Version
	reader, err := newGraphIndexRunner("inventory", o)
	if err != nil {
		return nil, err
	}
	intent := scvtransfer.Intent{Protocol: scvtransfer.Protocol, SourceRoot: a, TargetRoot: b, Plan: raw}
	encoded, err := scvtransfer.Seal(intent)
	if err != nil {
		return nil, err
	}
	intent, err = scvtransfer.ReadIntent(encoded)
	if err != nil {
		return nil, err
	}
	return &transferRunner{source: source, target: target, reader: reader, intent: intent, plan: plan, hook: scvTransferBarrier}, nil
}
func withinRoot(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	return err == nil && rel != ".." && !filepath.IsAbs(rel) && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
func (t *transferRunner) verifySource() error {
	pin := t.plan["input"].(map[string]any)
	raw, err := maintenanceInvoke(t.source, "transfer_plan", pin)
	if err != nil {
		return err
	}
	actual, _ := scvtransfer.Decode(raw)
	if actual["digest"] != t.plan["digest"] {
		return fmt.Errorf("source plan changed")
	}
	return nil
}
func (t *transferRunner) owners() error {
	for _, v := range t.plan["selected"].([]any) {
		s := v.(map[string]any)["source"]
		raw, _ := knowledgeengine.SCVCanonical(s)
		ref, err := knowledgeengine.SCVGraphIndexReference(raw)
		if err != nil {
			return err
		}
		if _, err = t.source.validateOwner(ref.Owner, ref.Graph, ref.QueryTime); err != nil {
			return err
		}
	}
	return nil
}
func (t *transferRunner) targetRecord(item map[string]any) (map[string]any, error) {
	raw, _ := knowledgeengine.SCVCanonical(item["source"])
	source, _ := scvtransfer.Decode(raw)
	intent := source["intent"].(map[string]any)
	snapshot := intent["snapshot"].(map[string]any)
	snapshot["connector"] = t.plan["input"].(map[string]any)["target_connector"]
	raw, err := scvtransfer.Seal(snapshot)
	if err != nil {
		return nil, err
	}
	snapshot, _ = scvtransfer.Decode(raw)
	intent["snapshot"] = snapshot
	raw, err = scvtransfer.Seal(intent)
	if err != nil {
		return nil, err
	}
	intent, _ = scvtransfer.Decode(raw)
	if intent["digest"] != item["target_intent_digest"] || snapshot["digest"] != item["target_snapshot_digest"] {
		return nil, fmt.Errorf("planned target identity differs")
	}
	source["intent"] = intent
	source["snapshot_digest"] = snapshot["digest"]
	raw, err = scvtransfer.Seal(source)
	if err != nil {
		return nil, err
	}
	return scvtransfer.Decode(raw)
}
func (t *transferRunner) auditTarget(j *scvtransfer.Journal) (map[string]any, error) {
	exists, err := j.CheckRoot()
	if err != nil {
		return nil, err
	}
	phases := map[string]string{}
	for _, e := range j.Events {
		if e.OperationID != "" {
			phases[e.OperationID] = e.Phase
		}
	}
	expected := map[string]map[string]any{}
	for _, v := range t.plan["selected"].([]any) {
		s, err := t.targetRecord(v.(map[string]any))
		if err != nil {
			return nil, err
		}
		id := s["intent"].(map[string]any)["operation_id"].(string)
		expected[id] = s
	}
	for _, e := range j.Events {
		if e.ResultDigest != "" {
			want := expected[e.OperationID]
			want["state"] = e.Phase
			want["index_verified"] = e.Phase == "committed"
			raw, err := scvtransfer.Seal(want)
			if err != nil {
				return nil, err
			}
			v, _ := scvtransfer.Decode(raw)
			if v["digest"] != e.ResultDigest {
				return nil, fmt.Errorf("recorded target acknowledgement differs")
			}
		}
	}
	if !exists {
		for _, p := range phases {
			if p != "prepare_pending" {
				return nil, fmt.Errorf("acknowledged destination database missing")
			}
		}
		return nil, nil
	}
	input := map[string]any{"tops_id": t.source.options.scv.topsID, "namespace": t.source.options.namespace, "expected_revision": nil, "cursor": nil, "limit": 16}
	raw, err := maintenanceInvoke(t.reader, "inventory", input)
	if err != nil {
		return nil, err
	}
	inventory, _ := scvtransfer.Decode(raw)
	manifest := inventory["manifest"].(map[string]any)
	entries := manifest["entries"].([]any)
	global := manifest["global_counts"].(map[string]any)
	if fmt.Sprint(global["intents"]) != fmt.Sprint(len(entries)) || inventory["next_cursor"] != nil {
		return nil, fmt.Errorf("unexpected global destination records")
	}
	actual := map[string]bool{}
	for _, v := range inventory["records"].([]any) {
		s := v.(map[string]any)
		id := s["intent"].(map[string]any)["operation_id"].(string)
		want, ok := expected[id]
		if !ok || phases[id] == "" {
			return nil, fmt.Errorf("unexpected destination operation")
		}
		phase, state := phases[id], s["state"].(string)
		if (phase == "prepare_pending" || phase == "prepared") && state != "prepared" {
			return nil, fmt.Errorf("destination committed before selected commit stage")
		}
		if phase == "committed" && state != "committed" {
			return nil, fmt.Errorf("destination lost committed state")
		}
		want["state"] = state
		want["index_verified"] = state == "committed"
		sealed, err := scvtransfer.Seal(want)
		if err != nil {
			return nil, err
		}
		w, _ := scvtransfer.Decode(sealed)
		if w["digest"] != s["digest"] {
			return nil, fmt.Errorf("destination record or projection differs")
		}
		actual[id] = true
	}
	for id, p := range phases {
		if p != "prepare_pending" && !actual[id] {
			return nil, fmt.Errorf("acknowledged destination operation missing")
		}
	}
	return manifest, nil
}
func (t *transferRunner) work(j *scvtransfer.Journal) (map[string]any, error) {
	if _, err := maintenanceInvoke(t.target, "inspect", map[string]any{}); err != nil {
		return nil, err
	}
	if err := t.verifySource(); err != nil {
		return nil, err
	}
	if err := t.owners(); err != nil {
		return nil, err
	}
	manifest, err := t.auditTarget(j)
	if err != nil {
		return nil, err
	}
	for {
		phase, id := scvtransfer.Next(j.Intent, j.Events)
		if phase == "" {
			return manifest, nil
		}
		if phase == "prepare_pending" || phase == "commit_pending" {
			if err = j.Append(phase, id, ""); err != nil {
				return manifest, err
			}
			if err = t.hook(phase); err != nil {
				return manifest, err
			}
			continue
		}
		// Check exact source, native owners and target again before each destination
		// mutation. Journal pending stages precede all prepare/commit calls.
		if err = t.verifySource(); err != nil {
			return manifest, err
		}
		if err = t.owners(); err != nil {
			return manifest, err
		}
		if manifest, err = t.auditTarget(j); err != nil {
			return manifest, err
		}
		if phase == "complete" {
			if _, err = maintenanceInvoke(t.target, "inspect", map[string]any{}); err != nil {
				return manifest, err
			}
			if err = j.Append("complete", "", ""); err != nil {
				return manifest, err
			}
			if err = t.hook("complete"); err != nil {
				return manifest, err
			}
			return manifest, nil
		}
		var item map[string]any
		for _, v := range t.plan["selected"].([]any) {
			s := v.(map[string]any)
			if s["source"].(map[string]any)["intent"].(map[string]any)["operation_id"] == id {
				item = s
				break
			}
		}
		want, err := t.targetRecord(item)
		if err != nil {
			return manifest, err
		}
		intent := want["intent"].(map[string]any)
		snap := intent["snapshot"].(map[string]any)
		payload := map[string]any{"tops_id": t.source.options.scv.topsID, "namespace": t.source.options.namespace, "operation_id": id}
		op := "prepare"
		if phase == "prepared" {
			payload["graph"] = snap["graph"]
			payload["query_time"] = intent["validation_query_time"]
			payload["owner"] = snap["owner"]
			payload["connector"] = snap["connector"]
		} else {
			op = "commit"
			payload["expected_intent_digest"] = intent["digest"]
		}
		raw, err := maintenanceInvoke(t.target, op, payload)
		if err != nil {
			return manifest, err
		}
		result, _ := scvtransfer.Decode(raw)
		if result["intent"].(map[string]any)["digest"] != intent["digest"] || result["state"] != map[string]string{"prepared": "prepared", "committed": "committed"}[phase] {
			return manifest, fmt.Errorf("destination response differs from exact plan")
		}
		if err = t.hook("after_" + op); err != nil {
			return manifest, err
		}
		if err = j.Append(phase, id, result["digest"].(string)); err != nil {
			return manifest, err
		}
		if err = t.hook(phase); err != nil {
			return manifest, err
		}
	}
}
func (t *transferRunner) execute(create bool) (json.RawMessage, error) {
	var output json.RawMessage
	// Initial validation must happen before reserving even an empty destination.
	if err := t.verifySource(); err != nil {
		return nil, err
	}
	if err := t.owners(); err != nil {
		return nil, err
	}
	err := scvtransfer.With(t.intent.TargetRoot, create, &t.intent, func(j *scvtransfer.Journal) error {
		manifest, workErr := t.work(j)
		return transferOutput(j, manifest, workErr, true, &output)
	})
	return output, err
}
func transferOutput(j *scvtransfer.Journal, manifest map[string]any, problem error, revalidated bool, out *json.RawMessage) error {
	phase, id := scvtransfer.Next(j.Intent, j.Events)
	status := "incomplete"
	if phase == "" && problem == nil {
		status = "complete"
	}
	if !revalidated {
		status = "recorded_" + status
	}
	var reason any
	if problem != nil {
		reason = problem.Error()
	}
	events := j.Events
	if events == nil {
		events = []scvtransfer.Event{}
	}
	r := map[string]any{"protocol": "symphony.qxctl.scv-index-transfer-result.v1", "transfer_digest": j.Intent.Digest, "source_root": j.Intent.SourceRoot, "target_root": j.Intent.TargetRoot, "events": events, "next_phase": phase, "next_operation_id": id, "status": status, "problem": reason, "target_manifest": manifest}
	raw, err := scvtransfer.Seal(r)
	*out = raw
	return err
}
