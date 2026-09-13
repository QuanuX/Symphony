package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
)

func graphIndexCLIRaw(t *testing.T, v any) json.RawMessage {
	t.Helper()
	raw, err := knowledgeengine.SCVCanonical(v)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
func graphIndexCLISeal(t *testing.T, v map[string]any) map[string]any {
	t.Helper()
	delete(v, "digest")
	normalized := graphIndexCLIMap(t, graphIndexCLIRaw(t, v))
	for k, value := range normalized {
		v[k] = value
	}
	d, err := knowledgeengine.SCVDigest(v)
	if err != nil {
		t.Fatal(err)
	}
	v["digest"] = d
	return v
}
func graphIndexCLIMap(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var m map[string]any
	d := json.NewDecoder(strings.NewReader(string(raw)))
	d.UseNumber()
	if err := d.Decode(&m); err != nil {
		t.Fatal(err)
	}
	return m
}
func graphIndexCLIInstallation(connector bool) knowledgeengine.Installation {
	role, module, engine, version := "scv", "scv-engine", "symphony-scv", "0.10.0-dev"
	if connector {
		role = "scv-graph-duckdb-connector"
		module = role
		engine = "symphony-" + role
		version = "0.1.0-dev"
	}
	prefix := "/private/graph-index-fixture/" + module
	return knowledgeengine.Installation{Role: role, ModuleID: module, EngineID: engine, Version: version, Prefix: prefix, ReceiptPath: filepath.Join(prefix, "share/symphony/receipts", module, version, "install-receipt.json"), ReceiptDigest: "sha256:" + strings.Repeat("a", 64), ReceiptProtocol: "symphony.knowledge.install-receipt.v2", ExecutablePath: filepath.Join(prefix, "libexec/symphony", module, version, engine), ExecutableDigest: "sha256:" + strings.Repeat("b", 64)}
}

type graphIndexCLIModel struct {
	intent              map[string]any
	state               string
	commits, ownerCalls int
	times               []string
	ownerMissing        bool
	changed             bool
	oversized           bool
}

// This model tests coordinator ordering and binding, not DuckDB transactions or
// native graph semantics. Installed-process coverage lives in verify_installed.py.
func graphIndexCLIModelRunner(t *testing.T) (*graphIndexRunner, *graphIndexCLIModel, map[string]any) {
	owner, connector := graphIndexCLIInstallation(false), graphIndexCLIInstallation(true)
	m := &graphIndexCLIModel{}
	r := &graphIndexRunner{options: graphIndexOptions{backend: "duckdb", connectorPrefix: connector.Prefix, connectorVersion: connector.Version, namespace: "fixture", operationID: "operation-1", scv: scvOptions{domain: owner.Role, prefix: owner.Prefix, version: owner.Version, topsID: "00000000-0000-4000-8000-000000000099"}}, connector: connector}
	r.inspectOwner = func(domain, prefix, version string) (knowledgeengine.Installation, error) {
		if m.ownerMissing {
			return knowledgeengine.Installation{}, errors.New("owner unavailable")
		}
		if domain != owner.Role || prefix != owner.Prefix || version != owner.Version {
			t.Fatal("coordinator selected a different original owner")
		}
		current := owner
		if m.changed {
			current.ExecutableDigest = "sha256:" + strings.Repeat("c", 64)
		}
		return current, nil
	}
	r.owner = func(inst knowledgeengine.Installation, graph json.RawMessage, queryTime string) (json.RawMessage, error) {
		m.ownerCalls++
		m.times = append(m.times, queryTime)
		if inst != owner {
			t.Fatal("owner was not pinned")
		}
		if m.oversized {
			values := make([]any, 32760)
			for i := range values {
				values[i] = false
			}
			raw := graphIndexCLIRaw(t, map[string]any{"values": values})
			if err := knowledgeengine.ValidateJSONObject(raw, 4<<20); err != nil {
				t.Fatalf("individual bounded fixture invalid: %v", err)
			}
			return raw, nil
		}
		g := graphIndexCLIMap(t, graph)
		return graphIndexCLIRaw(t, graphIndexCLISeal(t, map[string]any{"protocol": "symphony.scv.query-result.v1", "graph_digest": g["digest"], "query_time": queryTime, "fixture": "owner validation result"})), nil
	}
	counts := map[string]any{"claims": 0, "nodes": 0, "edges": 0}
	pd, _ := knowledgeengine.SCVDigest(map[string]any{"claims": []any{}, "nodes": []any{}, "edges": []any{}})
	r.invoke = func(op string, input map[string]any) (json.RawMessage, error) {
		var out map[string]any
		switch op {
		case "prepare":
			snapshot := graphIndexCLISeal(t, map[string]any{"protocol": "symphony.scv.graph-index-snapshot.v1", "backend": "duckdb", "mapping_version": "1", "tops_id": input["tops_id"], "namespace": input["namespace"], "graph": input["graph"], "owner": input["owner"], "connector": input["connector"]})
			candidate := graphIndexCLISeal(t, map[string]any{"protocol": "symphony.scv.graph-index-intent.v1", "operation_id": input["operation_id"], "snapshot": snapshot, "validation_query_time": input["query_time"]})
			if m.intent != nil && m.intent["digest"] != candidate["digest"] {
				return nil, errors.New("operation collision")
			}
			m.intent = candidate
			if m.state == "" {
				m.state = "prepared"
			}
		case "commit":
			if m.intent == nil || input["expected_intent_digest"] != m.intent["digest"] {
				return nil, errors.New("stale intent")
			}
			m.commits++
			m.state = "committed"
		case "status":
			if m.intent == nil {
				return nil, errors.New("missing operation")
			}
		case "query", "export":
			if m.state != "committed" {
				return nil, errors.New("unpublished snapshot")
			}
		default:
			return nil, errors.New("unsupported mock operation")
		}
		snapshot := m.intent["snapshot"].(map[string]any)
		if op == "query" {
			out = map[string]any{"protocol": "symphony.scv.graph-index-query.v1", "backend": "duckdb", "input": input, "snapshot": snapshot, "projection_digest": pd, "counts": counts, "rows": []any{}, "matched_count": 0, "next_cursor": nil}
		} else if op == "export" {
			out = map[string]any{"protocol": "symphony.scv.graph-index-export.v1", "backend": "duckdb", "snapshot": snapshot, "projection_digest": pd, "counts": counts}
		} else {
			out = map[string]any{"protocol": "symphony.scv.graph-index-status.v1", "backend": "duckdb", "intent": m.intent, "state": m.state, "snapshot_digest": snapshot["digest"], "projection_digest": pd, "counts": counts, "index_verified": m.state == "committed"}
		}
		raw := graphIndexCLIRaw(t, graphIndexCLISeal(t, out))
		if err := knowledgeengine.ValidateSCVGraphIndexResult(op, graphIndexCLIRaw(t, input), raw); err != nil {
			t.Fatalf("invalid mock boundary for %s: %v", op, err)
		}
		return raw, nil
	}
	h := "sha256:" + strings.Repeat("d", 64)
	graph := graphIndexCLISeal(t, map[string]any{"protocol": "symphony.scv.graph.v1", "domain": "scv", "selection_policy": map[string]any{"policy_id": "fixture", "max_age_seconds": nil, "allowed_statement_kinds": []any{}, "partial_capture": "exclude"}, "knowledge_digests": []any{h}, "interpretations": []any{map[string]any{"knowledge_digest": h, "domain": "scv", "interpreter_version": "fixture"}}, "captures": []any{}, "claims": []any{}, "native_nodes": []any{}, "native_edges": []any{}, "limitations": []any{}})
	return r, m, map[string]any{"operation_id": "operation-1", "graph": graph, "query_time": "2026-09-13T00:00:00Z"}
}
func TestSCVGraphIndexImportReturnsOwnerEvaluation(t *testing.T) {
	r, m, input := graphIndexCLIModelRunner(t)
	raw, err := r.execute("import", input)
	if err != nil {
		t.Fatal(err)
	}
	out := graphIndexCLIMap(t, raw)
	if out["owner_evaluation"] == nil || m.ownerCalls != 1 || m.commits != 1 {
		t.Fatalf("import dropped semantic result or commit ordering: %s", raw)
	}
	if out["owner_evaluation"].(map[string]any)["query_time"] != input["query_time"] {
		t.Fatal("query time changed")
	}
	native := out["connector_result"].(map[string]any)
	if native["state"] != "committed" {
		t.Fatal("did not publish committed response")
	}
}
func TestSCVGraphIndexInterruptedImportAndExactRecovery(t *testing.T) {
	r, m, input := graphIndexCLIModelRunner(t)
	r.afterPrepare = func() error { return errors.New("interrupted after durable prepare") }
	if _, err := r.execute("import", input); err == nil || m.state != "prepared" || m.commits != 0 {
		t.Fatal("interruption crossed commit")
	}
	exactIntent := string(graphIndexCLIRaw(t, m.intent))
	m.ownerMissing = true
	calls := m.ownerCalls
	status, err := r.execute("status", nil)
	if err != nil {
		t.Fatal(err)
	}
	if graphIndexCLIMap(t, status)["owner_evaluation"] != nil || m.ownerCalls != calls {
		t.Fatal("status executed owner or implied semantics")
	}
	if _, err := r.execute("recover", nil); err == nil || m.commits != 0 {
		t.Fatal("recovery silently replaced missing owner")
	}
	if string(graphIndexCLIRaw(t, m.intent)) != exactIntent {
		t.Fatal("failed recovery changed intent")
	}
	m.ownerMissing = false
	r.options.scv.domain = "schv"
	r.options.scv.prefix = "/another"
	r.options.scv.version = "0.1.0-dev"
	recovered, err := r.execute("recover", nil)
	if err != nil {
		t.Fatal(err)
	}
	if graphIndexCLIMap(t, recovered)["owner_evaluation"] == nil || m.commits != 1 {
		t.Fatal("recovery failed pinned owner")
	}
	replay, err := r.execute("recover", nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(replay) != string(recovered) || string(graphIndexCLIRaw(t, m.intent)) != exactIntent {
		t.Fatal("completed retry changed retained evidence")
	}
}
func TestSCVGraphIndexOwnerDriftAndEnvelopeFailBeforeCommit(t *testing.T) {
	for _, test := range []string{"owner_drift", "oversized_evaluation"} {
		t.Run(test, func(t *testing.T) {
			r, m, input := graphIndexCLIModelRunner(t)
			if test == "owner_drift" {
				r.afterPrepare = func() error { m.changed = true; return nil }
			} else {
				m.oversized = true
			}
			if _, err := r.execute("import", input); err == nil || m.state != "prepared" || m.commits != 0 {
				t.Fatalf("failure crossed commit: %v state=%s commits=%d", err, m.state, m.commits)
			}
		})
	}
}
func TestSCVGraphIndexQueryTimeAndMissingOwnerSeparation(t *testing.T) {
	r, m, input := graphIndexCLIModelRunner(t)
	if _, err := r.execute("import", input); err != nil {
		t.Fatal(err)
	}
	snapshot := m.intent["snapshot"].(map[string]any)
	for _, action := range []string{"query", "export"} {
		t.Run(action, func(t *testing.T) {
			in := map[string]any{"snapshot_digest": snapshot["digest"], "query_time": "2027-09-13T00:00:00Z"}
			if action == "query" {
				in["kind"] = "claims"
				in["filters"] = map[string]any{}
				in["cursor"] = nil
				in["limit"] = 128
			}
			raw, err := r.execute(action, in)
			if err != nil {
				t.Fatal(err)
			}
			out := graphIndexCLIMap(t, raw)
			if out["owner_evaluation"].(map[string]any)["query_time"] != in["query_time"] {
				t.Fatal("query time not passed to exact owner")
			}
			if m.intent["validation_query_time"] != input["query_time"] {
				t.Fatal("query overwrote retained import time")
			}
			m.ownerMissing = true
			if _, err := r.execute(action, in); err == nil {
				t.Fatal("semantic retrieval succeeded with owner absent")
			}
			m.ownerMissing = false
		})
	}
}
func TestSCVGraphIndexClosedInputAndRawUnicode(t *testing.T) {
	r, m, input := graphIndexCLIModelRunner(t)
	input["sql"] = "SELECT *"
	if _, err := r.execute("import", input); err == nil || m.ownerCalls != 0 || m.intent != nil {
		t.Fatal("unknown caller input reached owner or persistence")
	}
	for _, tc := range []struct {
		name, raw string
		valid     bool
	}{{"lone", `{"text":"\ud800"}`, false}, {"replacement", `{"text":"�"}`, true}, {"pair", `{"text":"\ud83d\ude00"}`, true}, {"duplicate", `{"text":1,"text":2}`, false}} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "input.json")
			if err := os.WriteFile(path, []byte(tc.raw), 0600); err != nil {
				t.Fatal(err)
			}
			o := graphIndexOptions{scv: scvOptions{input: path}}
			_, err := graphIndexInput(o)
			if (err == nil) != tc.valid {
				t.Fatalf("raw validation: %v", err)
			}
		})
	}
}
func TestSCVGraphIndexAgenticCommandBindings(t *testing.T) {
	group := newSCVGraphIndexCommand()
	if len(group.Commands()) != 6 {
		t.Fatal("graph index leaves missing")
	}
	for _, c := range group.Commands() {
		spec, err := commandregistry.Spec(c)
		if err != nil {
			t.Fatal(err)
		}
		if c.Name() != "inspect" && spec.Mutability != "evidence_only" {
			t.Fatal("physical recovery misclassified")
		}
		if c.Name() == "import" || c.Name() == "recover" || c.Name() == "query" || c.Name() == "export" {
			for _, domain := range knowledgeengine.SCVDomains() {
				want := "engop:symphony:" + domain + ".graph.query"
				found := false
				for _, op := range spec.BackendOperationIDs {
					if op == want {
						found = true
					}
				}
				if !found {
					t.Fatalf("%s omitted %s", c.Name(), want)
				}
			}
		}
		if c.Name() == "recover" {
			want := commandregistry.FeatureBinding{FeatureID: graphIndexFeature, Interaction: "recover"}
			found := false
			for _, b := range spec.FeatureBindings {
				if reflect.DeepEqual(b, want) {
					found = true
				}
			}
			if !found {
				t.Fatal("connector recovery interaction missing")
			}
		}
		if c.Name() == "import" {
			for _, name := range []string{"domain", "prefix", "version", "connector-prefix", "connector-version", "index-root", "tops-id", "namespace"} {
				if len(c.Flags().Lookup(name).Annotations["cobra_annotation_bash_completion_one_required_flag"]) == 0 {
					t.Fatalf("%s must be explicitly selected", name)
				}
			}
		}
	}
}

func TestSCVGraphIndexRejectsUnusedWorkingDirectory(t *testing.T) {
	group := newSCVGraphIndexCommand()
	for _, command := range group.Commands() {
		if command.Flags().Lookup("repo") != nil {
			t.Fatalf("%s exposes an ignored working-directory flag", command.Name())
		}
	}
	group.SetArgs([]string{"import", "--repo", "/unselected-working-directory"})
	if err := group.Execute(); err == nil {
		t.Fatal("ignored --repo flag was accepted")
	}
	_, err := newGraphIndexRunner("import", graphIndexOptions{scv: scvOptions{repository: "/unselected-working-directory"}})
	if err == nil || !strings.Contains(err.Error(), "--repo") {
		t.Fatalf("programmatic ignored repository reached installation/storage lookup: %v", err)
	}
	for _, action := range []string{"inspect", "status", "recover"} {
		r, model, _ := graphIndexCLIModelRunner(t)
		r.options.scv.input = "/unused-input.json"
		if _, err := r.execute(action, nil); err == nil || model.intent != nil || model.ownerCalls != 0 {
			t.Fatalf("%s ignored input or invoked backend", action)
		}
	}
}
