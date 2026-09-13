package knowledgeengine

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
)

const SCVGraphIndexConnectorVersion = "0.1.0-dev"

var scvGraphIndexConnectorSpec = engineSpec{
	label: "scv-graph-duckdb-connector", moduleID: "scv-graph-duckdb-connector",
	engineID: "symphony-scv-graph-duckdb-connector", componentKind: "adapter",
	vectorID: "scv", processProtocol: processProtocol,
}

// InspectSCVGraphIndexConnector selects one explicit optional backend and exact
// receipt-owned executable. It never discovers or chooses a default database.
func InspectSCVGraphIndexConnector(backend, prefix, version string) (Installation, error) {
	if backend != "duckdb" || version != SCVGraphIndexConnectorVersion || prefix == "" {
		return Installation{}, fmt.Errorf("graph index requires explicit duckdb backend, connector prefix and exact supported version")
	}
	s := scvGraphIndexConnectorSpec
	entry, err := InspectReceiptV2EntryPoint(prefix, version, ReceiptV2EntryPointSpec{
		Label: s.label, ComponentID: s.moduleID, ComponentKind: s.componentKind,
		ModuleID: s.moduleID, PackageID: s.moduleID, VectorID: &s.vectorID, EngineID: &s.engineID,
		EntryPointID: s.engineID, EntryPointKind: "executable",
		EntryPointRelativePath: filepath.ToSlash(filepath.Join("libexec", "symphony", s.moduleID, version, s.engineID)),
		RequiredProtocols:      []string{processProtocol},
	})
	if err != nil {
		return Installation{}, err
	}
	return Installation{Role: s.moduleID, ModuleID: s.moduleID, EngineID: s.engineID, Version: version,
		Prefix: entry.Prefix, ReceiptPath: entry.ReceiptPath, ReceiptDigest: entry.ReceiptDigest,
		ReceiptProtocol: receiptProtocolV2, ExecutablePath: entry.ExecutablePath, ExecutableDigest: entry.ExecutableDigest}, nil
}

func InvokeSCVGraphIndexConnector(ctx context.Context, backend, prefix, version, cwd, operation string, payload []byte) (Response, error) {
	switch operation {
	case "inspect", "prepare", "commit", "status", "query", "export":
	default:
		return Response{}, fmt.Errorf("unsupported graph connector operation")
	}
	if err := ValidateSCVBundleText(payload); err != nil {
		return Response{}, err
	}
	installed, err := InspectSCVGraphIndexConnector(backend, prefix, version)
	if err != nil {
		return Response{}, err
	}
	response, err := invokeResolved(ctx, scvGraphIndexConnectorSpec, installed.ExecutablePath, version, cwd, operation, payload)
	if err != nil {
		return response, err
	}
	if err := ValidateSCVGraphIndexResult(operation, payload, response.Result); err != nil {
		return Response{}, err
	}
	if operation != "inspect" {
		ref, err := SCVGraphIndexReference(response.Result)
		if err != nil || ref.Connector != installed {
			return Response{}, fmt.Errorf("graph index snapshot binds another connector installation")
		}
	}
	after, err := InspectSCVGraphIndexConnector(backend, prefix, version)
	if err != nil || after != installed {
		return Response{}, fmt.Errorf("graph connector installation changed during invocation")
	}
	return response, nil
}

var graphIndexNamespace = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
var graphIndexOperation = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)

type GraphIndexReference struct {
	Owner, Connector Installation
	Graph            json.RawMessage
	SnapshotDigest   string
	IntentDigest     string
	QueryTime        string
	State            string
}

func graphIndexFields(m map[string]any, keys ...string) bool {
	if m == nil || len(m) != len(keys) {
		return false
	}
	for _, key := range keys {
		if _, ok := m[key]; !ok {
			return false
		}
	}
	return true
}
func graphIndexText(v any, max int) (string, bool) {
	s, ok := v.(string)
	if !ok || s == "" || len(s) > max {
		return "", false
	}
	for _, c := range s {
		if c == 0 {
			return "", false
		}
	}
	return s, true
}
func graphIndexTime(v any) bool {
	s, ok := v.(string)
	if !ok || len(s) != 20 {
		return false
	}
	t, err := time.Parse("2006-01-02T15:04:05Z", s)
	return err == nil && t.Format("2006-01-02T15:04:05Z") == s
}
func graphIndexScope(m map[string]any) bool {
	tops, ok := m["tops_id"].(string)
	if !ok || stavprotocol.ValidateTOPSID(tops) != nil {
		return false
	}
	ns, ok := m["namespace"].(string)
	return ok && graphIndexNamespace.MatchString(ns)
}
func graphIndexInstallation(value any, connector bool) (Installation, error) {
	m, ok := value.(map[string]any)
	if !ok || !graphIndexFields(m, "Role", "ModuleID", "EngineID", "Version", "Prefix", "ReceiptPath", "ReceiptDigest", "ReceiptProtocol", "ExecutablePath", "ExecutableDigest") {
		return Installation{}, fmt.Errorf("graph index installation has invalid exact fields")
	}
	for _, v := range m {
		if _, ok := graphIndexText(v, 4096); !ok {
			return Installation{}, fmt.Errorf("graph index installation has invalid text")
		}
	}
	raw, err := SCVCanonical(m)
	if err != nil {
		return Installation{}, err
	}
	var inst Installation
	if json.Unmarshal(raw, &inst) != nil || inst.ReceiptProtocol != receiptProtocolV2 || !taggedSHA256(inst.ReceiptDigest) || !taggedSHA256(inst.ExecutableDigest) {
		return Installation{}, fmt.Errorf("graph index installation identity is invalid")
	}
	for _, path := range []string{inst.Prefix, inst.ReceiptPath, inst.ExecutablePath} {
		if !filepath.IsAbs(path) || filepath.Clean(path) != path {
			return Installation{}, fmt.Errorf("graph index installation paths must be exact absolute paths")
		}
	}
	if connector {
		s := scvGraphIndexConnectorSpec
		if inst.Role != s.moduleID || inst.ModuleID != s.moduleID || inst.EngineID != s.engineID || inst.Version != SCVGraphIndexConnectorVersion {
			return Installation{}, fmt.Errorf("graph index connector identity mismatch")
		}
	} else if !SCVDomainSupported(inst.Version, inst.Role) || inst.ModuleID != inst.Role+"-engine" || inst.EngineID != "symphony-"+inst.Role {
		return Installation{}, fmt.Errorf("graph index SCV owner identity mismatch")
	}
	if inst.ReceiptPath != filepath.Join(inst.Prefix, "share/symphony/receipts", inst.ModuleID, inst.Version, "install-receipt.json") || inst.ExecutablePath != filepath.Join(inst.Prefix, "libexec/symphony", inst.ModuleID, inst.Version, inst.EngineID) {
		return Installation{}, fmt.Errorf("graph index installation path binding mismatch")
	}
	return inst, nil
}

func graphIndexSnapshot(value any) (map[string]any, map[string]any, GraphIndexReference, error) {
	var ref GraphIndexReference
	snapshot, ok := value.(map[string]any)
	if !ok || !graphIndexFields(snapshot, "protocol", "backend", "mapping_version", "tops_id", "namespace", "graph", "owner", "connector", "digest") || snapshot["protocol"] != "symphony.scv.graph-index-snapshot.v1" || snapshot["backend"] != "duckdb" || snapshot["mapping_version"] != "1" || !graphIndexScope(snapshot) {
		return nil, nil, ref, fmt.Errorf("invalid graph index snapshot")
	}
	if err := scvSeal(snapshot, "digest"); err != nil {
		return nil, nil, ref, err
	}
	owner, err := graphIndexInstallation(snapshot["owner"], false)
	if err != nil {
		return nil, nil, ref, err
	}
	connector, err := graphIndexInstallation(snapshot["connector"], true)
	if err != nil {
		return nil, nil, ref, err
	}
	graph, ok := snapshot["graph"].(map[string]any)
	if !ok || !graphIndexFields(graph, "protocol", "domain", "selection_policy", "knowledge_digests", "interpretations", "captures", "claims", "native_nodes", "native_edges", "limitations", "digest") || graph["protocol"] != "symphony.scv.graph.v1" || graph["domain"] != owner.Role {
		return nil, nil, ref, fmt.Errorf("graph index snapshot has invalid owner graph")
	}
	if err := scvSeal(graph, "digest"); err != nil {
		return nil, nil, ref, err
	}
	for name, bound := range map[string]int{"captures": 16, "claims": 128, "native_nodes": 512, "native_edges": 1024, "knowledge_digests": 16, "interpretations": 16} {
		items, ok := graph[name].([]any)
		if !ok || len(items) > bound {
			return nil, nil, ref, fmt.Errorf("graph index graph exceeds %s bound", name)
		}
	}
	projection, err := graphIndexProjection(graph)
	if err != nil {
		return nil, nil, ref, err
	}
	graphRaw, err := SCVCanonical(graph)
	if err != nil {
		return nil, nil, ref, err
	}
	ref = GraphIndexReference{Owner: owner, Connector: connector, Graph: graphRaw, SnapshotDigest: snapshot["digest"].(string)}
	return snapshot, projection, ref, nil
}

func graphIndexProjection(graph map[string]any) (map[string]any, error) {
	projection := map[string]any{}
	for kind, field := range map[string]string{"claims": "claims", "nodes": "native_nodes", "edges": "native_edges"} {
		items, ok := graph[field].([]any)
		if !ok {
			return nil, fmt.Errorf("invalid graph index collection")
		}
		rows := make([]any, 0, len(items))
		seen := map[string]bool{}
		for _, item := range items {
			value, ok := item.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("graph index row value must be an object")
			}

			if kind == "claims" {
				for _, field := range []string{"subject", "predicate"} {
					if _, ok := graphIndexText(value[field], 512); !ok {
						return nil, fmt.Errorf("invalid indexed claim text")
					}
				}
				if !graphIndexScopeFilter(value["scope"]) {
					return nil, fmt.Errorf("invalid indexed claim scope")
				}
			}
			if kind == "nodes" {
				if _, ok := graphIndexText(value["kind"], 512); !ok {
					return nil, fmt.Errorf("invalid indexed node kind")
				}
				if d, ok := value["capture_digest"].(string); !ok || !taggedSHA256(d) {
					return nil, fmt.Errorf("invalid indexed capture identity")
				}
			}
			var key string
			if kind == "edges" {
				if !graphIndexFields(value, "from", "relation", "to") {
					return nil, fmt.Errorf("invalid graph index edge")
				}
				for _, v := range value {
					if _, ok := graphIndexText(v, 512); !ok {
						return nil, fmt.Errorf("invalid graph index edge field")
					}
				}
				var err error
				key, err = SCVDigest(value)
				if err != nil {
					return nil, err
				}
			} else {
				id := "claim_id"
				if kind == "nodes" {
					id = "node_id"
				}
				var ok bool
				key, ok = graphIndexText(value[id], 512)
				if !ok {
					return nil, fmt.Errorf("invalid graph index row key")
				}
			}
			if seen[key] {
				return nil, fmt.Errorf("duplicate graph index row key")
			}
			seen[key] = true
			rows = append(rows, map[string]any{"key": key, "value": value})
		}
		sort.Slice(rows, func(i, j int) bool {
			return rows[i].(map[string]any)["key"].(string) < rows[j].(map[string]any)["key"].(string)
		})
		projection[kind] = rows
	}
	return projection, nil
}

func graphIndexInventory(result, projection map[string]any) error {
	expected, err := SCVDigest(projection)
	if err != nil {
		return err
	}
	counts := map[string]any{}
	for _, kind := range []string{"claims", "nodes", "edges"} {
		counts[kind] = len(projection[kind].([]any))
	}
	if result["projection_digest"] != expected || !scvEqual(result["counts"], counts) {
		return fmt.Errorf("graph index inventory projection or count mismatch")
	}
	return nil
}

// SCVGraphIndexReference extracts only a mechanically validated retained graph.
// The caller still re-inspects and invokes its exact SCV owner when required.
func SCVGraphIndexReference(raw []byte) (GraphIndexReference, error) {
	value, err := scvObject(raw)
	if err != nil {
		return GraphIndexReference{}, err
	}
	if err = ValidateSCVBundleUnicode(raw); err != nil {
		return GraphIndexReference{}, err
	}
	snapshot := value["snapshot"]
	intent, _ := value["intent"].(map[string]any)
	if intent != nil {
		snapshot = intent["snapshot"]
	}
	_, _, ref, err := graphIndexSnapshot(snapshot)
	if err != nil {
		return ref, err
	}
	if intent != nil {
		ref.IntentDigest, _ = intent["digest"].(string)
		ref.QueryTime, _ = intent["validation_query_time"].(string)
		ref.State, _ = value["state"].(string)
	}
	return ref, nil
}

func ValidateSCVGraphIndexResult(operation string, input, raw []byte) error {
	if err := ValidateSCVBundleText(input); err != nil {
		return err
	}
	if err := validateJSONObject(raw, maxResponseBytes); err != nil {
		return err
	}
	if err := ValidateSCVBundleUnicode(raw); err != nil {
		return err
	}
	payload, err := scvObject(input)
	if err != nil {
		return err
	}
	result, err := scvObject(raw)
	if err != nil {
		return err
	}
	if operation == "inspect" {
		return graphIndexDescriptor(payload, result)
	}
	if !graphIndexScope(payload) || result["backend"] != "duckdb" {
		return fmt.Errorf("graph index namespace/backend mismatch")
	}
	if err = scvSeal(result, "digest"); err != nil {
		return err
	}
	var snapshotValue any
	if operation == "prepare" || operation == "commit" || operation == "status" {
		if !graphIndexFields(result, "protocol", "backend", "intent", "state", "snapshot_digest", "projection_digest", "counts", "index_verified", "digest") || result["protocol"] != "symphony.scv.graph-index-status.v1" {
			return fmt.Errorf("invalid graph index status fields")
		}
		intent, ok := result["intent"].(map[string]any)
		if !ok || !graphIndexFields(intent, "protocol", "operation_id", "snapshot", "validation_query_time", "digest") || intent["protocol"] != "symphony.scv.graph-index-intent.v1" || !graphIndexTime(intent["validation_query_time"]) {
			return fmt.Errorf("invalid graph index intent")
		}
		id, ok := intent["operation_id"].(string)
		if !ok || !graphIndexOperation.MatchString(id) || payload["operation_id"] != id {
			return fmt.Errorf("graph index operation binding mismatch")
		}
		if err = scvSeal(intent, "digest"); err != nil {
			return err
		}
		snapshotValue = intent["snapshot"]
		state, ok := result["state"].(string)
		if !ok || (state != "prepared" && state != "committed") || result["index_verified"] != (state == "committed") {
			return fmt.Errorf("graph index status overstates verification")
		}
		switch operation {
		case "prepare":
			if !graphIndexFields(payload, "tops_id", "namespace", "operation_id", "graph", "owner", "connector", "query_time") || !scvEqual(payload["query_time"], intent["validation_query_time"]) {
				return fmt.Errorf("graph index preparation input mismatch")
			}
			m, ok := snapshotValue.(map[string]any)
			if !ok {
				return fmt.Errorf("missing graph index snapshot")
			}
			for _, key := range []string{"graph", "owner", "connector"} {
				if !scvEqual(payload[key], m[key]) {
					return fmt.Errorf("graph index preparation changed %s", key)
				}
			}
		case "commit":
			if !graphIndexFields(payload, "tops_id", "namespace", "operation_id", "expected_intent_digest") || payload["expected_intent_digest"] != intent["digest"] || state != "committed" {
				return fmt.Errorf("graph index commit changed expected intent")
			}
		case "status":
			if !graphIndexFields(payload, "tops_id", "namespace", "operation_id") {
				return fmt.Errorf("invalid graph index status input")
			}
		}
	} else if operation == "query" {
		if !graphIndexFields(result, "protocol", "backend", "input", "snapshot", "projection_digest", "counts", "rows", "matched_count", "next_cursor", "digest") || result["protocol"] != "symphony.scv.graph-index-query.v1" || !scvEqual(result["input"], payload) {
			return fmt.Errorf("graph index query input/result mismatch")
		}
		snapshotValue = result["snapshot"]
	} else if operation == "export" {
		if !graphIndexFields(payload, "tops_id", "namespace", "snapshot_digest") || !graphIndexFields(result, "protocol", "backend", "snapshot", "projection_digest", "counts", "digest") || result["protocol"] != "symphony.scv.graph-index-export.v1" {
			return fmt.Errorf("invalid graph index export")
		}
		snapshotValue = result["snapshot"]
	} else {
		return fmt.Errorf("unsupported graph index result")
	}
	snapshot, projection, _, err := graphIndexSnapshot(snapshotValue)
	if err != nil {
		return err
	}
	if payload["tops_id"] != snapshot["tops_id"] || payload["namespace"] != snapshot["namespace"] {
		return fmt.Errorf("graph index snapshot changed namespace")
	}
	if operation == "query" || operation == "export" {
		if payload["snapshot_digest"] != snapshot["digest"] {
			return fmt.Errorf("graph index snapshot reference mismatch")
		}
	} else if result["snapshot_digest"] != snapshot["digest"] {
		return fmt.Errorf("graph index status snapshot mismatch")
	}
	if err = graphIndexInventory(result, projection); err != nil {
		return err
	}
	if operation == "query" {
		return graphIndexQuery(payload, result, projection)
	}
	return nil
}

func graphIndexScopeFilter(value any) bool {
	m, ok := value.(map[string]any)
	if !ok || len(m) > 16 {
		return false
	}
	for k, v := range m {
		if len(k) == 0 || len(k) > 128 {
			return false
		}
		if s, ok := v.(string); !ok || len(s) > 512 {
			return false
		}
	}
	return true
}

func graphIndexQuery(input, result, projection map[string]any) error {
	if !graphIndexFields(input, "tops_id", "namespace", "snapshot_digest", "kind", "filters", "cursor", "limit") {
		return fmt.Errorf("invalid graph index query fields")
	}
	kind, ok := input["kind"].(string)
	if !ok {
		return fmt.Errorf("invalid graph index kind")
	}
	fields, ok := map[string][]string{"claims": {"claim_id", "subject", "predicate", "scope"}, "nodes": {"node_id", "kind", "capture_digest"}, "edges": {"from", "relation", "to"}}[kind]
	if !ok {
		return fmt.Errorf("unsupported graph index kind")
	}
	filters, ok := input["filters"].(map[string]any)
	if !ok {
		return fmt.Errorf("graph index filters must be object")
	}
	allowed := map[string]bool{}
	for _, f := range fields {
		allowed[f] = true
	}
	for key, value := range filters {
		if !allowed[key] {
			return fmt.Errorf("unknown graph index filter")
		}
		if key == "scope" {
			if !graphIndexScopeFilter(value) {
				return fmt.Errorf("invalid graph scope filter")
			}
		} else if _, ok := graphIndexText(value, 65536); !ok {
			return fmt.Errorf("invalid graph index filter text")
		}
	}
	limit, ok := input["limit"].(json.Number)
	if !ok {
		return fmt.Errorf("graph index limit must be integer")
	}
	n, err := limit.Int64()
	if err != nil || n < 1 || n > 128 {
		return fmt.Errorf("graph index limit outside bounds")
	}
	query := map[string]any{}
	for _, key := range []string{"tops_id", "namespace", "snapshot_digest", "kind", "filters"} {
		query[key] = input[key]
	}
	queryDigest, err := SCVDigest(query)
	if err != nil {
		return err
	}
	matches := []any{}
	for _, item := range projection[kind].([]any) {
		row := item.(map[string]any)
		value := row["value"].(map[string]any)
		match := true
		for key, v := range filters {
			if !scvEqual(v, value[key]) {
				match = false
				break
			}
		}
		if match {
			matches = append(matches, item)
		}
	}
	start := 0
	if input["cursor"] != nil {
		cursor, ok := input["cursor"].(map[string]any)
		if !ok || !graphIndexFields(cursor, "query_digest", "after_key") || cursor["query_digest"] != queryDigest {
			return fmt.Errorf("graph index cursor does not bind query")
		}
		key, ok := graphIndexText(cursor["after_key"], 512)
		if !ok {
			return fmt.Errorf("invalid graph index cursor key")
		}
		found := false
		for i, item := range matches {
			if item.(map[string]any)["key"] == key {
				start = i + 1
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("graph index cursor is not a matching row")
		}
	}
	end := start + int(n)
	if end > len(matches) {
		end = len(matches)
	}
	rows := matches[start:end]
	var next any
	if end < len(matches) {
		next = map[string]any{"query_digest": queryDigest, "after_key": rows[len(rows)-1].(map[string]any)["key"]}
	}
	if !scvEqual(result["rows"], rows) || !scvEqual(result["matched_count"], len(matches)) || !scvEqual(result["next_cursor"], next) {
		return fmt.Errorf("graph index query rows/count/cursor differ from exact graph projection")
	}
	return nil
}

func graphIndexDescriptor(input, result map[string]any) error {
	if len(input) != 0 {
		return fmt.Errorf("connector inspect payload must be empty")
	}
	raw, err := SCVCanonical(result)
	if err != nil {
		return err
	}
	if err = ValidateSCVResult("inspect", []byte(`{}`), raw); err != nil {
		return err
	}
	s := scvGraphIndexConnectorSpec
	if result["module_id"] != s.moduleID || result["engine_id"] != s.engineID || result["vector_id"] != "scv" || result["engine_version"] != SCVGraphIndexConnectorVersion {
		return fmt.Errorf("graph connector descriptor identity mismatch")
	}

	for key, expected := range map[string]any{
		"format_version": 2, "process_protocols": []string{processProtocol}, "supported_scopes": []string{"tops"},
		"language": "C++26", "thermal_path": "freezing",
		"limits": map[string]any{"request_bytes": maxRequestBytes, "response_bytes": maxResponseBytes, "json_depth": maxJSONDepth, "json_values": maxJSONValues, "path_bytes": 4096, "snapshot_files": 1024, "snapshot_file_bytes": 4 << 20, "deadline_ahead_ms": 300000},
	} {
		if !scvEqual(result[key], expected) {
			return fmt.Errorf("graph connector descriptor changed %s", key)
		}
	}
	ops, ok := result["operations"].([]any)
	if !ok || len(ops) != 6 {
		return fmt.Errorf("graph connector descriptor operation count mismatch")
	}
	seen := map[string]bool{}
	for _, item := range ops {
		m, ok := item.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid graph connector operation descriptor")
		}
		name, ok := m["operation_name"].(string)
		if !ok || seen[name] {
			return fmt.Errorf("duplicate graph connector operation")
		}
		switch name {
		case "inspect", "prepare", "commit", "status", "query", "export":
		default:
			return fmt.Errorf("unknown graph connector operation")
		}
		if m["engine_operation_id"] != "engop:symphony:scv.graph-index."+name {
			return fmt.Errorf("graph connector backend operation identity mismatch")
		}

		if !graphIndexFields(m, "engine_operation_id", "operation_name", "availability", "feature_ids", "administrative_interactions", "administration_disposition", "input_protocol", "output_protocol", "mutability", "idempotency", "expected_state_required", "authorization_requirement", "recovery_operation_id", "direct_invocation", "thermal_path") {
			return fmt.Errorf("graph connector operation has missing or unknown fields")
		}
		interactions := map[string][]string{"inspect": {"inspect"}, "prepare": {"invoke"}, "commit": {"invoke", "recover"}, "status": {"inspect", "recover"}, "query": {"query"}, "export": {"query"}}[name]
		output := "symphony.scv.graph-index-status.v1"
		if name == "inspect" {
			output = result["protocol"].(string)
		} else if name == "query" {
			output = "symphony.scv.graph-index-query.v1"
		} else if name == "export" {
			output = "symphony.scv.graph-index-export.v1"
		}
		mutability := "evidence_only"
		if name == "inspect" {
			mutability = "read_only"
		}
		for key, expected := range map[string]any{"availability": "implemented", "feature_ids": []string{"ssfv:symphony:scv-graph-duckdb-connector"}, "administrative_interactions": interactions, "administration_disposition": "qxctl_required", "input_protocol": "symphony.scv.graph-index-" + name + "-input.v1", "output_protocol": output, "mutability": mutability, "idempotency": "idempotent", "authorization_requirement": "none", "recovery_operation_id": nil, "direct_invocation": "supported", "thermal_path": "freezing"} {
			if !scvEqual(m[key], expected) {
				return fmt.Errorf("graph connector operation changed %s", key)
			}
		}
		if !scvEqual(m["expected_state_required"], name == "commit") {
			return fmt.Errorf("graph connector expected-state declaration mismatch")
		}
		seen[name] = true
	}
	return nil
}
