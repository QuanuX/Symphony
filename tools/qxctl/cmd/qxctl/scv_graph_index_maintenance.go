package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/spf13/cobra"
)

func newSCVGraphIndexMaintenanceCommand(action string) *cobra.Command {
	o := graphIndexOptions{}
	c := &cobra.Command{Use: action, Args: usageOnlyArgs, RunE: func(*cobra.Command, []string) error {
		if o.connectorVersion != knowledgeengine.SCVGraphIndexPlanningVersion {
			return fmt.Errorf("inventory/planning requires exact connector 0.2.0-dev")
		}
		input, err := graphIndexInput(o)
		if err != nil {
			return err
		}
		r, err := newGraphIndexRunner(action, o)
		if err != nil {
			return err
		}
		result, err := runGraphIndexMaintenance(r, action, input, observeGraphIndexTarget)
		if err != nil {
			return err
		}
		return outputSCV(o.scv, result)
	}}
	scvRetainedReadFlags(c, &o.scv)
	c.Flags().StringVar(&o.backend, "backend", "duckdb", "explicit selected SQL index backend")
	c.Flags().StringVar(&o.connectorPrefix, "connector-prefix", "", "exact source connector installation")
	c.Flags().StringVar(&o.connectorVersion, "connector-version", "", "exact source reader version, 0.2.0-dev")
	c.Flags().StringVar(&o.root, "index-root", "", "existing private source index root")
	c.Flags().StringVar(&o.scv.topsID, "tops-id", "", "exact caller TOPS UUID")
	c.Flags().StringVar(&o.namespace, "namespace", "", "explicit inventory namespace")
	for _, flag := range []string{"connector-prefix", "connector-version", "index-root", "tops-id", "namespace"} {
		_ = c.MarkFlagRequired(flag)
	}
	s := commandSpec("scv.graph-index."+action, featureSCVAdministration, "query")
	s.Mutability = "evidence_only"
	s.InputProtocols = []string{"symphony.qxctl.scv-graph-index-" + action + "-input.v1"}
	s.OutputProtocols = []string{"symphony.qxctl.scv-graph-index-maintenance-result.v1"}
	s.ResultValidationProtocols = s.OutputProtocols
	s.BackendOperationIDs = []string{"engop:symphony:scv.graph-index.inventory"}
	s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: graphIndexFeature, Interaction: "query"})
	if action == "transfer-plan" {
		s.BackendOperationIDs = append(s.BackendOperationIDs, "engop:symphony:scv.graph-index.transfer.plan", "engop:symphony:scv.graph-index.inspect")
		s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: graphIndexFeature, Interaction: "inspect"})
		for _, domain := range knowledgeengine.SCVDomains() {
			s.BackendOperationIDs = append(s.BackendOperationIDs, "engop:symphony:"+domain+".graph.query")
			s.FeatureBindings = append(s.FeatureBindings, commandregistry.FeatureBinding{FeatureID: "ssfv:symphony:" + domain + "-engine", Interaction: "query"})
		}
	}
	commandregistry.Attach(c, s)
	return c
}

type graphIndexTargetObserver func(map[string]any) (map[string]any, error)

func observeGraphIndexTarget(target map[string]any) (map[string]any, error) {
	if !graphIndexExact(target, "prefix", "version", "root") {
		return nil, fmt.Errorf("target requires exact prefix/version/root")
	}
	prefix, ok := target["prefix"].(string)
	if !ok {
		return nil, fmt.Errorf("target prefix required")
	}
	version, ok := target["version"].(string)
	if !ok {
		return nil, fmt.Errorf("target version required")
	}
	root, ok := target["root"].(string)
	if !ok || !filepath.IsAbs(root) || filepath.Clean(root) != root || root == "/" {
		return nil, fmt.Errorf("target root must be clean absolute")
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil || resolved != root {
		return nil, fmt.Errorf("target root must exist without symlinks")
	}
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0o700 {
		return nil, fmt.Errorf("target root must be private 0700 directory")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(os.Geteuid()) {
		return nil, fmt.Errorf("target root ownership differs")
	}
	inst, err := knowledgeengine.InspectSCVGraphIndexConnector("duckdb", prefix, version)
	if err != nil {
		return nil, err
	}
	if _, err = knowledgeengine.InvokeSCVGraphIndexConnector(context.Background(), "duckdb", prefix, version, root, "inspect", []byte(`{}`)); err != nil {
		return nil, err
	}
	directory, err := os.Open(root)
	if err != nil {
		return nil, err
	}
	defer directory.Close()
	entries, err := directory.ReadDir(1)
	if err != nil && err != io.EOF {
		return nil, err
	}
	state := "empty"
	if len(entries) > 0 {
		state = "not_empty"
	}
	return map[string]any{"installation": inst, "root": root, "state": state}, nil
}
func maintenanceInvoke(r *graphIndexRunner, op string, payload map[string]any) (json.RawMessage, error) {
	raw, err := r.invoke(op, payload)
	if err != nil {
		return nil, err
	}
	input, err := knowledgeengine.SCVCanonical(payload)
	if err != nil {
		return nil, err
	}
	if err = knowledgeengine.ValidateSCVGraphIndexResult(op, input, raw); err != nil {
		return nil, err
	}
	return raw, nil
}
func maintenanceMap(raw []byte) map[string]any {
	var v map[string]any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	_ = d.Decode(&v)
	return v
}
func runGraphIndexMaintenance(r *graphIndexRunner, action string, input map[string]any, observe graphIndexTargetObserver) (json.RawMessage, error) {
	if r.connector.Version != knowledgeengine.SCVGraphIndexPlanningVersion {
		return nil, fmt.Errorf("maintenance requires selected 0.2.0-dev reader")
	}
	payload := map[string]any{"tops_id": r.options.scv.topsID, "namespace": r.options.namespace}
	var native json.RawMessage
	var target any
	owners := []any{}
	blockers := []any{}
	disposition := "observed"
	if action == "inventory" {
		if !graphIndexExact(input, "expected_revision", "cursor", "limit") {
			return nil, fmt.Errorf("inventory requires expected_revision/cursor/limit")
		}
		for key, value := range input {
			payload[key] = value
		}
		var err error
		native, err = maintenanceInvoke(r, "inventory", payload)
		if err != nil {
			return nil, err
		}
	} else if action == "transfer-plan" {
		if !graphIndexExact(input, "expected_revision", "operation_ids", "target", "capacity") {
			return nil, fmt.Errorf("transfer-plan requires expected_revision/operation_ids/target/capacity")
		}
		choice, ok := input["target"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("target object required")
		}
		observed, err := observe(choice)
		if err != nil {
			return nil, err
		}
		target = observed
		payload["expected_revision"] = input["expected_revision"]
		payload["operation_ids"] = input["operation_ids"]
		payload["capacity"] = input["capacity"]
		payload["source_connector"] = r.connector
		payload["target_connector"] = observed["installation"]
		payload["target_root"] = observed["root"]
		native, err = maintenanceInvoke(r, "transfer_plan", payload)
		if err != nil {
			return nil, err
		}
		plan := maintenanceMap(native)
		for _, code := range plan["blockers"].([]any) {
			blockers = append(blockers, map[string]any{"code": code, "operation_id": nil})
		}
		if observed["state"] != "empty" {
			blockers = append(blockers, map[string]any{"code": "target_not_empty", "operation_id": nil})
		}
		for _, item := range plan["selected"].([]any) {
			source := item.(map[string]any)["source"]
			raw, err := knowledgeengine.SCVCanonical(source)
			if err != nil {
				return nil, err
			}
			ref, err := knowledgeengine.SCVGraphIndexReference(raw)
			if err != nil {
				return nil, err
			}
			id := source.(map[string]any)["intent"].(map[string]any)["operation_id"]
			evaluation, ownerErr := r.validateOwner(ref.Owner, ref.Graph, ref.QueryTime)
			record := map[string]any{"operation_id": id, "outcome": "validated", "evaluation": evaluation, "reason": nil}
			if ownerErr != nil {
				reason := ownerErr.Error()
				if len(reason) > 1024 {
					reason = "exact owner validation unavailable"
				}
				record["outcome"] = "unavailable"
				record["evaluation"] = nil
				record["reason"] = reason
				blockers = append(blockers, map[string]any{"code": "owner_validation_unavailable", "operation_id": id})
			}
			owners = append(owners, record)
		}
		// Reobserve source revision after all semantic work. This is planning-time
		// evidence, never a reservation or an atomic transaction across both roots.
		if _, err = maintenanceInvoke(r, "inventory", map[string]any{"tops_id": payload["tops_id"], "namespace": payload["namespace"], "expected_revision": payload["expected_revision"], "cursor": nil, "limit": 1}); err != nil {
			return nil, err
		}
		after, err := observe(choice)
		if err != nil {
			return nil, err
		}
		a, _ := knowledgeengine.SCVCanonical(observed)
		b, _ := knowledgeengine.SCVCanonical(after)
		if string(a) != string(b) {
			return nil, fmt.Errorf("target observation changed during planning")
		}
		disposition = "ready"
		if len(blockers) > 0 {
			disposition = "blocked"
		}
	} else {
		return nil, fmt.Errorf("unsupported maintenance action")
	}
	result := map[string]any{"protocol": "symphony.qxctl.scv-graph-index-maintenance-result.v1", "operation": action, "source_connector": r.connector, "connector_result": native, "target_observation": target, "owner_evaluations": owners, "blockers": blockers, "disposition": disposition}
	encoded, err := knowledgeengine.SCVCanonical(result)
	if err != nil {
		return nil, err
	}
	if err = knowledgeengine.ValidateJSONObject(encoded, 4<<20); err != nil {
		return nil, err
	}
	// Installation structs have declaration order; normalize every nested value
	// before sealing so independent consumers hash sorted JSON object keys.
	result = maintenanceMap(encoded)
	digest, err := knowledgeengine.SCVDigest(result)
	if err != nil {
		return nil, err
	}
	result["digest"] = digest
	raw, err := knowledgeengine.SCVCanonical(result)
	if err != nil {
		return nil, err
	}
	if err = knowledgeengine.ValidateJSONObject(raw, 4<<20); err != nil {
		return nil, err
	}
	return raw, knowledgeengine.ValidateSCVBundleUnicode(raw)
}
