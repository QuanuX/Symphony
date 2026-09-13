package knowledgeengine

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSCVGraphIndexConnectorReceiptBoundary(t *testing.T) {
	prefix := t.TempDir()
	spec := scvGraphIndexConnectorSpec
	spec.expectedFiles = func(version string) map[string]struct{} {
		return map[string]struct{}{
			filepath.ToSlash(filepath.Join("libexec", "symphony", spec.moduleID, version, spec.engineID)):        {},
			filepath.ToSlash(filepath.Join("share", "symphony", "contracts", spec.moduleID, version, "SPEC.md")): {},
		}
	}
	receiptPath, receipt := createInstalledV2Fixture(t, spec, prefix, SCVGraphIndexConnectorVersion)
	selected, err := InspectSCVGraphIndexConnector("duckdb", prefix, SCVGraphIndexConnectorVersion)
	if err != nil {
		t.Fatal(err)
	}
	if selected.ReceiptDigest != receipt.ReceiptDigest || selected.ModuleID != spec.moduleID || selected.Role != spec.moduleID {
		t.Fatal("connector identity was not pinned to the complete exact receipt")
	}
	for _, choice := range [][3]string{{"", prefix, SCVGraphIndexConnectorVersion}, {"other", prefix, SCVGraphIndexConnectorVersion}, {"duckdb", "", SCVGraphIndexConnectorVersion}, {"duckdb", prefix, "latest"}} {
		if _, err := InspectSCVGraphIndexConnector(choice[0], choice[1], choice[2]); err == nil {
			t.Fatalf("implicit or unsupported selection accepted: %v", choice)
		}
	}
	receipt.ComponentKind = "vector_engine"
	writeReceiptV2Fixture(t, receiptPath, &receipt)
	if _, err := InspectSCVGraphIndexConnector("duckdb", prefix, SCVGraphIndexConnectorVersion); err == nil {
		t.Fatal("connector receipt silently accepted another component kind")
	}
	receipt.ComponentKind = spec.componentKind
	writeReceiptV2Fixture(t, receiptPath, &receipt)
	document := filepath.Join(prefix, "share", "symphony", "contracts", spec.moduleID, SCVGraphIndexConnectorVersion, "SPEC.md")
	if err := os.WriteFile(document, []byte("substituted contract"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := InspectSCVGraphIndexConnector("duckdb", prefix, SCVGraphIndexConnectorVersion); err == nil {
		t.Fatal("connector inspection omitted a receipt-owned non-executable file")
	}
}

func TestSCVGraphIndexRejectsRawUnicodeBeforeInstallation(t *testing.T) {
	_, err := InvokeSCVGraphIndexConnector(context.Background(), "duckdb", "/absent-connector", SCVGraphIndexConnectorVersion, "/", "prepare", []byte(`{"data":"\ud800"}`))
	if err == nil || strings.Contains(err.Error(), "receipt") || strings.Contains(err.Error(), "installation") {
		t.Fatalf("malformed raw Unicode was not rejected before installation lookup: %v", err)
	}
}

// These fixtures exercise structural index-consumer integrity. They deliberately
// do not stand in for the separate native SCV semantic-owner acceptance tests.
func graphIndexTestRaw(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := SCVCanonical(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
func graphIndexTestSeal(t *testing.T, value map[string]any) map[string]any {
	t.Helper()
	delete(value, "digest")
	normalized, err := scvObject(graphIndexTestRaw(t, value))
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range normalized {
		value[k] = v
	}
	d, err := SCVDigest(value)
	if err != nil {
		t.Fatal(err)
	}
	value["digest"] = d
	return value
}
func graphIndexTestClone(t *testing.T, value map[string]any) map[string]any {
	t.Helper()
	m, err := scvObject(graphIndexTestRaw(t, value))
	if err != nil {
		t.Fatal(err)
	}
	return m
}
func graphIndexTestInstallation(connector bool) Installation {
	role, module, engine, version := "scv", "scv-engine", "symphony-scv", "0.10.0-dev"
	if connector {
		role = scvGraphIndexConnectorSpec.moduleID
		module = role
		engine = scvGraphIndexConnectorSpec.engineID
		version = SCVGraphIndexConnectorVersion
	}
	prefix := "/private/graph-index-test/" + module
	return Installation{Role: role, ModuleID: module, EngineID: engine, Version: version, Prefix: prefix, ReceiptPath: filepath.Join(prefix, "share/symphony/receipts", module, version, "install-receipt.json"), ReceiptDigest: "sha256:" + strings.Repeat("a", 64), ReceiptProtocol: receiptProtocolV2, ExecutablePath: filepath.Join(prefix, "libexec/symphony", module, version, engine), ExecutableDigest: "sha256:" + strings.Repeat("b", 64)}
}
func graphIndexTestSnapshot(t *testing.T) map[string]any {
	h := "sha256:" + strings.Repeat("c", 64)
	claim := func(id, subject string, scope map[string]any) any {
		return map[string]any{"claim_id": id, "subject": subject, "predicate": "available", "scope": scope, "statement_kind": "user_assertion", "value": map[string]any{"type": "boolean", "value": true, "unit": nil}, "evidence": []any{}, "dependencies": []any{}, "valid_from": nil, "valid_until": nil}
	}
	a, b := "native:"+strings.Repeat("a", 64), "native:"+strings.Repeat("b", 64)
	graph := graphIndexTestSeal(t, map[string]any{"protocol": "symphony.scv.graph.v1", "domain": "scv", "selection_policy": map[string]any{"policy_id": "fixture", "max_age_seconds": nil, "allowed_statement_kinds": []any{"user_assertion"}, "partial_capture": "exclude"}, "knowledge_digests": []any{h}, "interpretations": []any{map[string]any{"knowledge_digest": h, "domain": "scv", "interpreter_version": "fixture"}}, "captures": []any{}, "claims": []any{claim("é", "same", map[string]any{"region": "a"}), claim("a\t", "same", map[string]any{"region": "a"}), claim("z", "other", map[string]any{})}, "native_nodes": []any{map[string]any{"node_id": a, "kind": "source", "capture_digest": h, "pointer": "", "attributes": map[string]any{}}, map[string]any{"node_id": b, "kind": "section", "capture_digest": h, "pointer": "/section", "attributes": map[string]any{"title": "Fixture"}}}, "native_edges": []any{map[string]any{"from": a, "relation": "contains", "to": b}}, "limitations": []any{"Structural fixture only"}})
	return graphIndexTestClone(t, graphIndexTestSeal(t, map[string]any{"protocol": "symphony.scv.graph-index-snapshot.v1", "backend": "duckdb", "mapping_version": "1", "tops_id": "00000000-0000-4000-8000-000000000099", "namespace": "fixture", "graph": graph, "owner": graphIndexTestInstallation(false), "connector": graphIndexTestInstallation(true)}))
}
func graphIndexTestStatus(t *testing.T, snapshot map[string]any, state string) map[string]any {
	_, projection, _, err := graphIndexSnapshot(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	pd, err := SCVDigest(projection)
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]any{}
	for _, k := range []string{"claims", "nodes", "edges"} {
		counts[k] = len(projection[k].([]any))
	}
	intent := graphIndexTestSeal(t, map[string]any{"protocol": "symphony.scv.graph-index-intent.v1", "operation_id": "operation-1", "snapshot": snapshot, "validation_query_time": "2026-09-13T00:00:00Z"})
	return graphIndexTestSeal(t, map[string]any{"protocol": "symphony.scv.graph-index-status.v1", "backend": "duckdb", "intent": intent, "state": state, "snapshot_digest": snapshot["digest"], "projection_digest": pd, "counts": counts, "index_verified": state == "committed"})
}
func graphIndexTestQuery(t *testing.T, snapshot map[string]any, kind string, filters map[string]any, cursor any, limit int) (map[string]any, map[string]any) {
	input := map[string]any{"tops_id": snapshot["tops_id"], "namespace": snapshot["namespace"], "snapshot_digest": snapshot["digest"], "kind": kind, "filters": filters, "cursor": cursor, "limit": limit}
	_, projection, _, err := graphIndexSnapshot(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	q := map[string]any{}
	for _, k := range []string{"tops_id", "namespace", "snapshot_digest", "kind", "filters"} {
		q[k] = input[k]
	}
	qd, _ := SCVDigest(q)
	all := []any{}
	for _, v := range projection[kind].([]any) {
		r := v.(map[string]any)
		value := r["value"].(map[string]any)
		match := true
		for k, w := range filters {
			if !scvEqual(value[k], w) {
				match = false
			}
		}
		if match {
			all = append(all, r)
		}
	}
	start := 0
	if cursor != nil {
		for i, v := range all {
			if v.(map[string]any)["key"] == cursor.(map[string]any)["after_key"] {
				start = i + 1
			}
		}
	}
	end := start + limit
	if end > len(all) {
		end = len(all)
	}
	var next any
	if end < len(all) {
		next = map[string]any{"query_digest": qd, "after_key": all[end-1].(map[string]any)["key"]}
	}
	status := graphIndexTestStatus(t, snapshot, "committed")
	result := graphIndexTestSeal(t, map[string]any{"protocol": "symphony.scv.graph-index-query.v1", "backend": "duckdb", "input": input, "snapshot": snapshot, "projection_digest": status["projection_digest"], "counts": status["counts"], "rows": all[start:end], "matched_count": len(all), "next_cursor": next})
	return input, result
}
func TestSCVGraphIndexConsumerRejectsResealedProjectionAndBinding(t *testing.T) {
	snapshot := graphIndexTestSnapshot(t)
	input, result := graphIndexTestQuery(t, snapshot, "claims", map[string]any{"subject": "same", "scope": map[string]any{"region": "a"}}, nil, 1)
	if err := ValidateSCVGraphIndexResult("query", graphIndexTestRaw(t, input), graphIndexTestRaw(t, result)); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(map[string]any){
		"row_value": func(r map[string]any) {
			r["rows"].([]any)[0].(map[string]any)["value"].(map[string]any)["subject"] = "forged"
		},
		"count":            func(r map[string]any) { r["matched_count"] = 999 },
		"inventory_count":  func(r map[string]any) { r["counts"].(map[string]any)["claims"] = 1 },
		"inventory_digest": func(r map[string]any) { r["projection_digest"] = "sha256:" + strings.Repeat("0", 64) },
		"cursor":           func(r map[string]any) { r["next_cursor"] = nil },
		"unknown":          func(r map[string]any) { r["approved"] = true },
		"namespace": func(r map[string]any) {
			s := r["snapshot"].(map[string]any)
			s["namespace"] = "another"
			graphIndexTestSeal(t, s)
		},
		"owner_role": func(r map[string]any) {
			s := r["snapshot"].(map[string]any)
			s["owner"].(map[string]any)["Role"] = "schv"
			graphIndexTestSeal(t, s)
		},
		"connector_version": func(r map[string]any) {
			s := r["snapshot"].(map[string]any)
			s["connector"].(map[string]any)["Version"] = "latest"
			graphIndexTestSeal(t, s)
		},
	} {
		t.Run(name, func(t *testing.T) {
			r := graphIndexTestClone(t, result)
			mutate(r)
			graphIndexTestSeal(t, r)
			if err := ValidateSCVGraphIndexResult("query", graphIndexTestRaw(t, input), graphIndexTestRaw(t, r)); err == nil {
				t.Fatal("accepted resealed substitution")
			}
		})
	}
}
func TestSCVGraphIndexExactFiltersCursorAndOrdering(t *testing.T) {
	snapshot := graphIndexTestSnapshot(t)
	input, first := graphIndexTestQuery(t, snapshot, "claims", map[string]any{"subject": "same"}, nil, 1)
	if err := ValidateSCVGraphIndexResult("query", graphIndexTestRaw(t, input), graphIndexTestRaw(t, first)); err != nil {
		t.Fatal(err)
	}
	if first["rows"].([]any)[0].(map[string]any)["key"] != "a\t" {
		t.Fatal("UTF8 byte ordering or permitted control text changed")
	}
	next, second := graphIndexTestQuery(t, snapshot, "claims", map[string]any{"subject": "same"}, first["next_cursor"], 1)
	if err := ValidateSCVGraphIndexResult("query", graphIndexTestRaw(t, next), graphIndexTestRaw(t, second)); err != nil {
		t.Fatal(err)
	}
	if second["next_cursor"] != nil || second["rows"].([]any)[0].(map[string]any)["key"] != "é" {
		t.Fatal("incorrect final page")
	}
	for _, kind := range []string{"nodes", "edges"} {
		in, out := graphIndexTestQuery(t, snapshot, kind, map[string]any{}, nil, 128)
		if err := ValidateSCVGraphIndexResult("query", graphIndexTestRaw(t, in), graphIndexTestRaw(t, out)); err != nil {
			t.Fatal(err)
		}
	}
	for name, alter := range map[string]func(map[string]any){"foreign_cursor": func(i map[string]any) { i["filters"] = map[string]any{"subject": "other"} }, "missing_cursor_key": func(i map[string]any) { i["cursor"].(map[string]any)["after_key"] = "absent" }, "limit_zero": func(i map[string]any) { i["limit"] = 0 }, "limit_129": func(i map[string]any) { i["limit"] = 129 }, "filter_sql": func(i map[string]any) { i["filters"].(map[string]any)["sql"] = "SELECT *" }} {
		t.Run(name, func(t *testing.T) {
			in := graphIndexTestClone(t, next)
			alter(in)
			out := graphIndexTestClone(t, second)
			out["input"] = in
			graphIndexTestSeal(t, out)
			if err := ValidateSCVGraphIndexResult("query", graphIndexTestRaw(t, in), graphIndexTestRaw(t, out)); err == nil {
				t.Fatal("accepted invalid cursor/filter/bound")
			}
		})
	}
}
func TestSCVGraphIndexPreparedCommitAndUnicodeBoundaries(t *testing.T) {
	snapshot := graphIndexTestSnapshot(t)
	prepared := graphIndexTestStatus(t, snapshot, "prepared")
	committed := graphIndexTestStatus(t, snapshot, "committed")
	intent := prepared["intent"].(map[string]any)
	prepare := map[string]any{"tops_id": snapshot["tops_id"], "namespace": snapshot["namespace"], "operation_id": "operation-1", "graph": snapshot["graph"], "owner": snapshot["owner"], "connector": snapshot["connector"], "query_time": intent["validation_query_time"]}
	status := map[string]any{"tops_id": snapshot["tops_id"], "namespace": snapshot["namespace"], "operation_id": "operation-1"}
	commit := graphIndexTestClone(t, status)
	commit["expected_intent_digest"] = intent["digest"]
	for _, tc := range []struct {
		op      string
		in, out map[string]any
	}{{"prepare", prepare, prepared}, {"prepare", prepare, committed}, {"status", status, prepared}, {"status", status, committed}, {"commit", commit, committed}} {
		if err := ValidateSCVGraphIndexResult(tc.op, graphIndexTestRaw(t, tc.in), graphIndexTestRaw(t, tc.out)); err != nil {
			t.Fatalf("%s: %v", tc.op, err)
		}
	}
	stale := graphIndexTestClone(t, commit)
	stale["expected_intent_digest"] = "sha256:" + strings.Repeat("0", 64)
	if ValidateSCVGraphIndexResult("commit", graphIndexTestRaw(t, stale), graphIndexTestRaw(t, committed)) == nil {
		t.Fatal("stale commit accepted")
	}
	forged := graphIndexTestClone(t, prepared)
	forged["index_verified"] = true
	graphIndexTestSeal(t, forged)
	if ValidateSCVGraphIndexResult("status", graphIndexTestRaw(t, status), graphIndexTestRaw(t, forged)) == nil {
		t.Fatal("prepared state claimed committed verification")
	}
	duplicate := graphIndexTestClone(t, snapshot)
	g := duplicate["graph"].(map[string]any)
	g["claims"] = append(g["claims"].([]any), g["claims"].([]any)[0])
	graphIndexTestSeal(t, g)
	graphIndexTestSeal(t, duplicate)
	if _, _, _, err := graphIndexSnapshot(duplicate); err == nil {
		t.Fatal("duplicate row key accepted")
	}
	for _, raw := range []string{`{"tops_id":"\ud800"}`, `{"tops_id":"x","tops_id":"y"}`, `{"n":9007199254740992}`} {
		if ValidateSCVGraphIndexResult("status", []byte(raw), graphIndexTestRaw(t, prepared)) == nil {
			t.Fatal("invalid raw input accepted")
		}
	}
}

func TestSCVGraphIndexIndexedFieldShapeAndOpaqueScope(t *testing.T) {
	for name, mutate := range map[string]func(map[string]any){
		"subject":    func(g map[string]any) { g["claims"].([]any)[0].(map[string]any)["subject"] = true },
		"scope":      func(g map[string]any) { g["claims"].([]any)[0].(map[string]any)["scope"] = "not-object" },
		"capture_id": func(g map[string]any) { g["native_nodes"].([]any)[0].(map[string]any)["capture_digest"] = "not-sha256" },
		"edge_extra": func(g map[string]any) { g["native_edges"].([]any)[0].(map[string]any)["semantic_proof"] = true },
	} {
		t.Run(name, func(t *testing.T) {
			s := graphIndexTestSnapshot(t)
			g := s["graph"].(map[string]any)
			mutate(g)
			graphIndexTestSeal(t, g)
			graphIndexTestSeal(t, s)
			if _, _, _, err := graphIndexSnapshot(s); err == nil {
				t.Fatal("malformed indexed field accepted")
			}
		})
	}
	s := graphIndexTestSnapshot(t)
	g := s["graph"].(map[string]any)
	scope := map[string]any{"empty": "", "opaque\x00": "\x00"}
	g["claims"].([]any)[0].(map[string]any)["scope"] = scope
	graphIndexTestSeal(t, g)
	graphIndexTestSeal(t, s)
	in, out := graphIndexTestQuery(t, s, "claims", map[string]any{"scope": scope}, nil, 128)
	if err := ValidateSCVGraphIndexResult("query", graphIndexTestRaw(t, in), graphIndexTestRaw(t, out)); err != nil {
		t.Fatal(err)
	}
	if !scvEqual(out["matched_count"], 1) {
		t.Fatal("scope equality was reinterpreted")
	}
}
func TestSCVGraphIndexDescriptorRejectsResealedAuthorityDrift(t *testing.T) {
	operations := []any{}
	for _, name := range []string{"inspect", "prepare", "commit", "status", "query", "export"} {
		output := "symphony.scv.graph-index-status.v1"
		mutability := "evidence_only"
		if name == "inspect" {
			output = "symphony.knowledge.engine-descriptor.v2"
			mutability = "read_only"
		} else if name == "query" {
			output = "symphony.scv.graph-index-query.v1"
		} else if name == "export" {
			output = "symphony.scv.graph-index-export.v1"
		}
		interactions := map[string][]string{"inspect": {"inspect"}, "prepare": {"invoke"}, "commit": {"invoke", "recover"}, "status": {"inspect", "recover"}, "query": {"query"}, "export": {"query"}}[name]
		operations = append(operations, map[string]any{"engine_operation_id": "engop:symphony:scv.graph-index." + name, "operation_name": name, "availability": "implemented", "feature_ids": []string{"ssfv:symphony:scv-graph-duckdb-connector"}, "administrative_interactions": interactions, "administration_disposition": "qxctl_required", "input_protocol": "symphony.scv.graph-index-" + name + "-input.v1", "output_protocol": output, "mutability": mutability, "idempotency": "idempotent", "expected_state_required": name == "commit", "authorization_requirement": "none", "recovery_operation_id": nil, "direct_invocation": "supported", "thermal_path": "freezing"})
	}
	descriptor := map[string]any{"protocol": "symphony.knowledge.engine-descriptor.v2", "format_version": 2, "module_id": "scv-graph-duckdb-connector", "engine_id": "symphony-scv-graph-duckdb-connector", "vector_id": "scv", "engine_version": "0.1.0-dev", "process_protocols": []string{processProtocol}, "contract_versions": []string{"knowledge/SPEC.md@v1", "scv-graph-duckdb-connector/SPEC.md@v1", "scv-graph-duckdb-connector/DUCKDB-PROVENANCE.json@1.5.5"}, "operations": operations, "limits": map[string]any{"request_bytes": maxRequestBytes, "response_bytes": maxResponseBytes, "json_depth": maxJSONDepth, "json_values": maxJSONValues, "path_bytes": 4096, "snapshot_files": 1024, "snapshot_file_bytes": 4 << 20, "deadline_ahead_ms": 300000}, "supported_scopes": []string{"tops"}, "language": "C++26", "thermal_path": "freezing", "canonical_apply_enabled": false, "session_mutation_enabled": false, "network_listener": false}
	seal := func(m map[string]any) {
		delete(m, "descriptor_digest")
		d, err := SCVDigest(m)
		if err != nil {
			t.Fatal(err)
		}
		m["descriptor_digest"] = d
	}
	seal(descriptor)
	if err := ValidateSCVGraphIndexResult("inspect", []byte(`{}`), graphIndexTestRaw(t, descriptor)); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(map[string]any){"network": func(m map[string]any) { m["network_listener"] = true }, "bounds": func(m map[string]any) { m["limits"].(map[string]any)["request_bytes"] = 8 << 20 }, "operation_id": func(m map[string]any) {
		m["operations"].([]any)[0].(map[string]any)["engine_operation_id"] = "engop:symphony:other.inspect"
	}, "interaction": func(m map[string]any) {
		m["operations"].([]any)[1].(map[string]any)["administrative_interactions"] = []string{"configure"}
	}, "unknown": func(m map[string]any) { m["operations"].([]any)[0].(map[string]any)["authorize_user"] = true }} {
		t.Run(name, func(t *testing.T) {
			m := graphIndexTestClone(t, descriptor)
			mutate(m)
			seal(m)
			if ValidateSCVGraphIndexResult("inspect", []byte(`{}`), graphIndexTestRaw(t, m)) == nil {
				t.Fatal("accepted resealed descriptor drift")
			}
		})
	}
}
