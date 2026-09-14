package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/shvtransfer"
)

type shvTransferRunner struct {
	source, target, reader *shvStoreRunner
	intent                 shvtransfer.Intent
	plan                   map[string]any
	hook                   func(string) error
}

func newSHVTransferRunner(source *shvStoreRunner, input map[string]any) (*shvTransferRunner, error) {
	if !graphIndexExact(input, "plan", "expected_plan_digest") {
		return nil, fmt.Errorf("transfer requires plan and expected_plan_digest")
	}
	raw, err := knowledgeengine.SCVCanonical(input["plan"])
	if err != nil {
		return nil, err
	}
	plan, err := shvtransfer.Decode(raw)
	if err != nil {
		return nil, err
	}
	pin, ok := plan["input"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("native transfer plan required")
	}
	in, _ := knowledgeengine.SCVCanonical(pin)
	if err = knowledgeengine.ValidateSHVStoreResultVersion("transfer_plan", in, raw, knowledgeengine.SHVStoreTransferVersion); err != nil {
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
	if source.connector != plannedSource {
		return nil, fmt.Errorf("plan source installation or scope differs")
	}
	targetRoot := pin["target_root"].(string)
	// Nested roots make destination writes source filesystem changes and complicate
	// recovery ownership. This tool's copy boundary requires separate peer roots.
	a, b := source.root, targetRoot
	if a == b || withinRoot(a, b) || withinRoot(b, a) {
		return nil, fmt.Errorf("source and target roots must be separate")
	}
	targetRaw, _ := knowledgeengine.SCVCanonical(pin["target_connector"])
	var installation knowledgeengine.Installation
	if err = json.Unmarshal(targetRaw, &installation); err != nil {
		return nil, err
	}
	target, err := newSHVStoreRunner(targetRoot, installation.Prefix, installation.Version)
	if err != nil {
		return nil, err
	}
	if target.connector != installation {
		return nil, fmt.Errorf("exact target installation differs")
	}
	reader, err := newSHVStoreRunner(targetRoot, source.connector.Prefix, source.connector.Version)
	if err != nil {
		return nil, err
	}
	intent := shvtransfer.Intent{Protocol: shvtransfer.Protocol, SourceRoot: a, TargetRoot: b, Plan: raw}
	encoded, err := shvtransfer.Seal(intent)
	if err != nil {
		return nil, err
	}
	intent, err = shvtransfer.ReadIntent(encoded)
	if err != nil {
		return nil, err
	}
	return &shvTransferRunner{source: source, target: target, reader: reader, intent: intent, plan: plan, hook: shvTransferBarrier}, nil
}
func (t *shvTransferRunner) verifySource() error {
	pin := t.plan["input"].(map[string]any)
	raw, err := shvTransferInvoke(t.source, "transfer_plan", pin)
	if err != nil {
		return err
	}
	actual, _ := shvtransfer.Decode(raw)
	if actual["digest"] != t.plan["digest"] {
		return fmt.Errorf("source plan changed")
	}
	return nil
}

// Transfer verifies structural evidence. Source-document and kernel replay remain
// the publication owner's separate mandatory validation boundary.
func (t *shvTransferRunner) owners() error {
	for _, runner := range []*shvStoreRunner{t.source, t.target} {
		i, e := knowledgeengine.InspectSHVStore(runner.connector.Prefix, runner.connector.Version)
		if e != nil {
			return e
		}
		if i != runner.connector {
			return fmt.Errorf("transfer installation changed")
		}
	}
	return nil
}
func (t *shvTransferRunner) targetRecord(item map[string]any) (map[string]any, error) {
	raw, _ := knowledgeengine.SCVCanonical(item["source"])
	source, _ := shvtransfer.Decode(raw)
	intent := source["intent"].(map[string]any)
	snapshot := intent["snapshot"].(map[string]any)
	snapshot["connector"] = t.plan["input"].(map[string]any)["target_connector"]
	raw, err := shvtransfer.Seal(snapshot)
	if err != nil {
		return nil, err
	}
	snapshot, _ = shvtransfer.Decode(raw)
	intent["snapshot"] = snapshot
	raw, err = shvtransfer.Seal(intent)
	if err != nil {
		return nil, err
	}
	intent, _ = shvtransfer.Decode(raw)
	if intent["digest"] != item["target_intent_digest"] || snapshot["digest"] != item["target_snapshot_digest"] {
		return nil, fmt.Errorf("planned target identity differs")
	}
	source["intent"] = intent
	source["snapshot_digest"] = snapshot["digest"]
	raw, err = shvtransfer.Seal(source)
	if err != nil {
		return nil, err
	}
	return shvtransfer.Decode(raw)
}
func (t *shvTransferRunner) auditTarget(j *shvtransfer.Journal) (map[string]any, error) {
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
			raw, err := shvtransfer.Seal(want)
			if err != nil {
				return nil, err
			}
			v, _ := shvtransfer.Decode(raw)
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
	input := map[string]any{"tops_id": t.plan["input"].(map[string]any)["tops_id"], "namespace": t.plan["input"].(map[string]any)["namespace"], "expected_revision": nil, "cursor": nil, "limit": 16}
	raw, err := shvTransferInvoke(t.reader, "inventory", input)
	if err != nil {
		return nil, err
	}
	inventory, _ := shvtransfer.Decode(raw)
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
		sealed, err := shvtransfer.Seal(want)
		if err != nil {
			return nil, err
		}
		w, _ := shvtransfer.Decode(sealed)
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
func (t *shvTransferRunner) work(j *shvtransfer.Journal) (map[string]any, error) {
	if _, err := shvTransferInvoke(t.target, "inspect", map[string]any{}); err != nil {
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
		phase, id := shvtransfer.Next(j.Intent, j.Events)
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
		// Check exact source, exact installations and target again before each destination
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
			if _, err = shvTransferInvoke(t.target, "inspect", map[string]any{}); err != nil {
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
		payload := map[string]any{"tops_id": t.plan["input"].(map[string]any)["tops_id"], "namespace": t.plan["input"].(map[string]any)["namespace"], "operation_id": id}
		op := "prepare"
		if phase == "prepared" {
			payload["graph"] = snap["graph"]
			payload["connector"] = snap["connector"]
		} else {
			op = "commit"
			payload["expected_intent_digest"] = intent["digest"]
		}
		raw, err := shvTransferInvoke(t.target, op, payload)
		if err != nil {
			return manifest, err
		}
		result, _ := shvtransfer.Decode(raw)
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
func (t *shvTransferRunner) execute(create bool) (json.RawMessage, error) {
	var output json.RawMessage
	// Initial validation must happen before reserving even an empty destination.
	if err := t.verifySource(); err != nil {
		return nil, err
	}
	if err := t.owners(); err != nil {
		return nil, err
	}
	err := shvtransfer.With(t.intent.TargetRoot, create, &t.intent, func(j *shvtransfer.Journal) error {
		manifest, workErr := t.work(j)
		return shvTransferOutput(j, manifest, workErr, true, &output)
	})
	return output, err
}
func shvTransferOutput(j *shvtransfer.Journal, manifest map[string]any, problem error, revalidated bool, out *json.RawMessage) error {
	phase, id := shvtransfer.Next(j.Intent, j.Events)
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
		events = []shvtransfer.Event{}
	}
	r := map[string]any{"protocol": "symphony.qxctl.shv-store-transfer-result.v1", "transfer_digest": j.Intent.Digest, "source_root": j.Intent.SourceRoot, "target_root": j.Intent.TargetRoot, "events": events, "next_phase": phase, "next_operation_id": id, "status": status, "problem": reason, "target_manifest": manifest}
	raw, err := shvtransfer.Seal(r)
	*out = raw
	return err
}

type shvStoreRunner struct {
	root      string
	connector knowledgeengine.Installation
}

func newSHVStoreRunner(root, prefix, version string) (*shvStoreRunner, error) {
	i, e := knowledgeengine.InspectSHVStore(prefix, version)
	if e != nil {
		return nil, e
	}
	return &shvStoreRunner{root: root, connector: i}, nil
}
func shvTransferInvoke(r *shvStoreRunner, op string, p map[string]any) (json.RawMessage, error) {
	raw, e := knowledgeengine.SCVCanonical(p)
	if e != nil {
		return nil, e
	}
	out, e := knowledgeengine.InvokeSHVStore(context.Background(), r.connector.Prefix, r.connector.Version, r.root, op, raw)
	return out.Result, e
}
