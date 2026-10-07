package knowledgeengine

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var SBVOperations = []string{"capabilities", "run", "compose", "result_inspect", "result_query", "evaluate", "catalogue", "compose_joint", "economics", "book", "liquidity", "allocation_economics", "result_select", "backend_plan", "live_plan", "analyze", "compare", "resample", "experiment", "split", "fit", "predict", "source_retain", "source_export", "generate_census", "provider_inspect", "compose_economics", "research_history", "dataset_load", "dataset_inspect", "dataset_execute", "dataset_release"}
var sbvSHA = regexp.MustCompile(`^[0-9a-f]{64}$`)

func sbvSpec() engineSpec {
	return engineSpec{label: "SBV", moduleID: "sbv-engine", vectorID: "sbv", engineID: "symphony-sbv", componentKind: "vector_engine", processProtocol: processProtocolV2}
}
func InspectSBV(prefix, version string) (Installation, error) {
	s := sbvSpec()
	if prefix == "" || sbvAdministrationInterfaceAdmission[version] == nil {
		return Installation{}, fmt.Errorf("SBV exact release required")
	}
	e, err := InspectReceiptV2EntryPoint(prefix, version, ReceiptV2EntryPointSpec{Label: s.label, ComponentID: s.moduleID, ComponentKind: s.componentKind, ModuleID: s.moduleID, PackageID: s.moduleID, VectorID: &s.vectorID, EngineID: &s.engineID, EntryPointID: s.engineID, EntryPointKind: "executable", EntryPointRelativePath: filepath.ToSlash(filepath.Join("libexec", "symphony", s.moduleID, version, s.engineID)), RequiredProtocols: []string{processProtocolV2}})
	if err != nil {
		return Installation{}, err
	}
	inst := Installation{Role: s.moduleID, ModuleID: s.moduleID, EngineID: s.engineID, Version: version, Prefix: e.Prefix, ReceiptPath: e.ReceiptPath, ReceiptDigest: e.ReceiptDigest, ReceiptProtocol: receiptProtocolV2, ExecutablePath: e.ExecutablePath, ExecutableDigest: e.ExecutableDigest}
	if err = verifyOwnerInterface(inst, sbvAdministrationInterfaceDigest); err != nil {
		return Installation{}, err
	}
	return inst, nil
}
func SBVResource(prefix, version, operation string, templates bool) (Installation, json.RawMessage, error) {
	if !sbvAdministrationInterfaceAdmission[version][operation] {
		return Installation{}, nil, fmt.Errorf("SBV operation is not admitted")
	}
	inst, err := InspectSBV(prefix, version)
	if err != nil {
		return inst, nil, err
	}
	name := "admin.schema.json"
	if templates {
		name = "admin.templates.json"
	}
	raw, err := snvOwnedResource(inst, "share/symphony/schemas/sbv-engine/"+version+"/"+name, sbvAdministrationInterfaceResources[name])
	if err != nil {
		return inst, nil, err
	}
	m, err := sqavObject(raw, maxRequestBytes)
	if err != nil {
		return inst, nil, err
	}
	key := "requests"
	if templates {
		key = "templates"
	}
	variants, ok := m[key].(map[string]any)
	if !ok || variants[operation] == nil {
		return inst, nil, fmt.Errorf("SBV resource variant missing")
	}
	out, err := SCVCanonical(variants[operation])
	return inst, out, err
}
func InvokeSBV(ctx context.Context, prefix, version, cwd, operation string, payload []byte) (Response, error) {
	p, err := sqavObject(payload, maxRequestBytes)
	if err != nil {
		return Response{}, err
	}
	inst, schema, err := SBVResource(prefix, version, operation, false)
	if err != nil {
		return Response{}, err
	}
	shape, err := sqavObject(schema, maxRequestBytes)
	if err != nil || !sbvInputShape(shape, p) {
		return Response{}, fmt.Errorf("SBV input shape rejected")
	}
	if operation == "dataset_execute" {
		child, ok := p["request"].(map[string]any)
		if !ok {
			return Response{}, fmt.Errorf("SBV resident child request required")
		}
		if _, present := child["output_path"]; present {
			return Response{}, fmt.Errorf("SBV resident child must omit output_path")
		}
		childOp, _ := p["operation"].(string)
		_, childSchema, e := SBVResource(prefix, version, childOp, false)
		if e != nil {
			return Response{}, e
		}
		childShape, e := sqavObject(childSchema, maxRequestBytes)
		if e != nil {
			return Response{}, e
		}
		expanded := make(map[string]any, len(child)+1)
		for k, v := range child {
			expanded[k] = v
		}
		expanded["output_path"] = p["output_path"]
		if !sbvInputShape(childShape, expanded) {
			return Response{}, fmt.Errorf("SBV resident child shape rejected")
		}
	}
	canonical, err := SCVCanonical(p)
	if err != nil {
		return Response{}, err
	}
	r, invokeErr := invokeResolved(ctx, sbvSpec(), inst.ExecutablePath, version, cwd, operation, canonical)
	after, err := InspectSBV(prefix, version)
	if err != nil || after != inst {
		return Response{}, fmt.Errorf("SBV installation changed during invocation; inspect destination before retrying writes")
	}
	if invokeErr != nil {
		return r, invokeErr
	}
	if err = validateSBVResult(operation, p, r.Result); err != nil {
		return Response{}, err
	}
	return r, nil
}
func validateSBVResult(op string, p map[string]any, raw []byte) error {
	if handled, err := ValidateSBVRecovery(op, p, raw); handled || err != nil {
		return err
	}
	m, err := sqavObject(raw, maxResponseBytes)
	if err != nil {
		return err
	}
	bad := func() error { return fmt.Errorf("SBV result contract mismatch") }
	if m["protocol"] != sbvAdministrationInterfaceOutputs[op] {
		return bad()
	}
	digest := func(k string) bool { s, ok := m[k].(string); return ok && sbvSHA.MatchString(s) }
	number := func(k string) (uint64, bool) {
		s, ok := m[k].(string)
		n, e := strconv.ParseUint(s, 10, 64)
		return n, ok && e == nil && strconv.FormatUint(n, 10) == s
	}
	switch op {
	case "dataset_load", "dataset_inspect", "dataset_release":
		if m["engine_version"] != SBVAdministrationInterfaceVersion || m["directory"] != p["directory"] || m["instance_id"] != p["instance_id"] || !digest("source_sha256") {
			return bad()
		}
		state := "ready"
		if op == "dataset_release" {
			state = "released"
		}
		if m["state"] != state {
			return bad()
		}
		for _, k := range []string{"events", "decoded_bytes", "load_buffer_bytes", "source_reads", "decode_passes", "active_jobs", "active_workers", "completed_jobs", "failed_jobs", "max_concurrent_jobs", "worker_budget", "idle_timeout_ms"} {
			if _, ok := number(k); !ok {
				return bad()
			}
		}
		if m["memory_budget_bytes"] != nil {
			if n, ok := number("memory_budget_bytes"); !ok || n == 0 {
				return bad()
			}
		}
		limits, ok := m["dataset_limits"].(map[string]any)
		if !ok || len(limits) != 3 {
			return bad()
		}
		for _, k := range []string{"max_source_bytes", "max_source_events", "max_metadata_bytes"} {
			v, present := limits[k]
			if !present {
				return bad()
			}
			if v != nil {
				raw, ok := v.(string)
				n, e := strconv.ParseUint(raw, 10, 64)
				if !ok || e != nil || n == 0 || strconv.FormatUint(n, 10) != raw {
					return bad()
				}
			}
		}
		if m["source_reads"] != "1" || m["decode_passes"] != "1" || (m["residency"] != "pageable" && m["residency"] != "locked") {
			return bad()
		}
		if delivery, present := m["source_delivery"]; present {
			identity, err := sbvRetainedDeliveryIdentity(delivery)
			if err != nil {
				return bad()
			}
			for _, k := range []string{"source_path", "source_sha256", "dataset"} {
				if m[k] != identity[k] {
					return bad()
				}
			}
			if m["load_buffer_scope"] != "original byte span plus decoded event allocation; excludes owner capture/batch/store/delivery allocations" {
				return bad()
			}
			if op == "dataset_load" {
				selected, ok := p["retained_source"].(map[string]any)
				captured := delivery.(map[string]any)
				if !ok || !reflect.DeepEqual(selected["reference"], captured["reference"]) || !reflect.DeepEqual(selected["delivery"], captured["choices"]) {
					return bad()
				}
			}
		} else if _, selected := p["retained_source"]; selected || m["load_buffer_scope"] != nil {
			return bad()
		}
		if op == "dataset_load" {
			wantLimits, present := p["dataset_limits"]
			if !present {
				wantLimits = map[string]any{"max_source_bytes": nil, "max_source_events": nil, "max_metadata_bytes": nil}
			}
			if !reflect.DeepEqual(m["dataset_limits"], wantLimits) {
				return bad()
			}
			fields := []string{"memory_budget_bytes", "residency", "max_concurrent_jobs", "worker_budget", "idle_timeout_ms"}
			if _, retained := p["retained_source"]; !retained {
				fields = append(fields, "source_path", "source_sha256", "dataset")
			}
			for _, k := range fields {
				if m[k] != p[k] {
					return bad()
				}
			}
		}
	case "capabilities":
		if m["engine_version"] != SBVAdministrationInterfaceVersion {
			return bad()
		}
		for _, k := range []string{"cpu", "cuda", "tensor", "live"} {
			x, ok := m[k].(map[string]any)
			if !ok {
				return bad()
			}
			a, ok := x["available"].(bool)
			if !ok || a != (k == "cpu") {
				return bad()
			}
		}
	case "run", "compose", "evaluate", "compose_joint", "economics", "book", "liquidity", "allocation_economics", "analyze", "compare", "resample", "experiment", "split", "fit", "predict", "source_retain", "source_export", "generate_census", "compose_economics", "research_history", "dataset_execute", "result_inspect":
		target := "output_path"
		if op == "source_export" {
			target = "receipt_path"
		}
		if op == "result_inspect" {
			target = "path"
		}
		n, ok := number("bytes")
		if m["path"] != p[target] || !digest("content_sha256") || !digest("file_sha256") || !ok || n == 0 || n > 128<<20 {
			return bad()
		}
		if op == "result_inspect" && p["expected_sha256"] != "" && m["content_sha256"] != p["expected_sha256"] {
			return bad()
		}
		if m["status"] != "completed" && m["status"] != "partial" {
			return bad()
		}
	case "provider_inspect":
		selection, ok := p["provider"].(map[string]any)
		if !ok || len(m) != 4 || m["engine_version"] != SBVAdministrationInterfaceVersion || !reflect.DeepEqual(m["extensions"], p["extensions"]) {
			return bad()
		}
		evidence, ok := m["provider"].(map[string]any)
		if !ok || len(evidence) != 15 || evidence["protocol"] != "symphony.sbv.native-provider-evidence.v1" || evidence["abi_version"] != "1" {
			return bad()
		}
		for _, key := range []string{"id", "version", "role", "concurrency"} {
			if evidence[key] != selection[key] {
				return bad()
			}
		}
		canonical, err := sbvNativeCanonical(selection)
		if err != nil || evidence["selection_sha256"] != strings.TrimPrefix(digestBytes(canonical), "sha256:") {
			return bad()
		}
		descriptor, ok := evidence["descriptor_json"].(string)
		if !ok || !json.Valid([]byte(descriptor)) || evidence["descriptor_sha256"] != strings.TrimPrefix(digestBytes([]byte(descriptor)), "sha256:") {
			return bad()
		}
		// Descriptor JSON may contain ordinary JSON Schema numbers; its exact
		// bytes are retained as a string by the numeric-string result contract.
		var declared map[string]any
		if json.Unmarshal([]byte(descriptor), &declared) != nil || declared["protocol"] != "symphony.sbv.native-provider-descriptor.v1" {
			return bad()
		}
		for _, key := range []string{"id", "version", "reproducibility", "cancellation"} {
			v, ok := evidence[key].(string)
			if !ok || v == "" || declared[key] != v {
				return bad()
			}
		}
		configuration, ok := evidence["configuration_validation"].(string)
		loading, present := evidence["loading"].(map[string]any)
		if !ok || configuration == "" || !present || len(loading) != 6 || loading["isolation"] != false || loading["symbol"] != "symphony_sbv_provider_api_v1" {
			return bad()
		}
		for _, key := range []string{"profile", "visibility", "dependency_resolution", "trust"} {
			if v, ok := loading[key].(string); !ok || v == "" {
				return bad()
			}
		}
		verifyFile := func(got, want any) bool {
			actual, ok := got.(map[string]any)
			selected, present := want.(map[string]any)
			if !ok || !present || len(actual) != 3 || actual["path"] != selected["path"] || actual["expected_sha256"] != selected["expected_sha256"] {
				return false
			}
			bytes, ok := actual["bytes"].(string)
			n, e := strconv.ParseUint(bytes, 10, 64)
			return ok && e == nil && strconv.FormatUint(n, 10) == bytes
		}
		if !verifyFile(evidence["library"], selection["library"]) {
			return bad()
		}
		dependencies, ok := evidence["dependencies"].(map[string]any)
		selectedDependencies, present := selection["dependencies"].(map[string]any)
		if !ok || !present || len(dependencies) != 3 || dependencies["capture"] != selectedDependencies["capture"] || dependencies["description"] != selectedDependencies["description"] {
			return bad()
		}
		actualFiles, ok := dependencies["artifacts"].([]any)
		selectedFiles, present := selectedDependencies["artifacts"].([]any)
		if !ok || !present || len(actualFiles) != len(selectedFiles) {
			return bad()
		}
		for i := range actualFiles {
			if !verifyFile(actualFiles[i], selectedFiles[i]) {
				return bad()
			}
		}
	case "catalogue":
		if m["engine_version"] != SBVAdministrationInterfaceVersion {
			return bad()
		}
		for _, key := range []string{"models", "studies", "transforms"} {
			items, ok := m[key].([]any)
			if !ok || len(items) == 0 {
				return bad()
			}
			for _, item := range items {
				card, ok := item.(map[string]any)
				if !ok {
					return bad()
				}
				for _, field := range []string{"id", "version"} {
					v, ok := card[field].(string)
					if !ok || v == "" {
						return bad()
					}
				}
			}
		}
	case "backend_plan", "live_plan":
		if m["engine_version"] != SBVAdministrationInterfaceVersion {
			return bad()
		}
		if op == "live_plan" && m["can_activate"] != false {
			return bad()
		}
	case "result_select":
		if m["content_sha256"] != p["expected_sha256"] || m["pointer"] != p["pointer"] || !digest("query_sha256") {
			return bad()
		}
		offset, a := number("offset")
		total, b := number("matched_rows")
		rows, c := m["rows"].([]any)
		complete, d := m["complete"].(bool)
		cursor, e := m["next_cursor"].(string)
		limit, _ := strconv.ParseUint(p["limit"].(string), 10, 64)
		if !a || !b || !c || !d || !e || offset > total || uint64(len(rows)) > limit || uint64(len(rows)) > total-offset || complete != (offset+uint64(len(rows)) == total) || (complete && cursor != "") || (!complete && len(rows) == 0) {
			return bad()
		}
		definition := make(map[string]any, len(p))
		for k, v := range p {
			if k != "cursor" {
				definition[k] = v
			}
		}
		encoded, err := sbvNativeCanonical(definition)
		if err != nil {
			return bad()
		}
		hash := strings.TrimPrefix(digestBytes(encoded), "sha256:")
		expectedOffset := uint64(0)
		inputCursor := p["cursor"].(string)
		if inputCursor != "" {
			if !strings.HasPrefix(inputCursor, hash+":") {
				return bad()
			}
			expectedOffset, err = strconv.ParseUint(strings.TrimPrefix(inputCursor, hash+":"), 10, 64)
			if err != nil {
				return bad()
			}
		}
		sourceRows, ok := number("source_rows")
		if !ok || total > sourceRows || m["query_sha256"] != hash || offset != expectedOffset || (!complete && cursor != hash+":"+strconv.FormatUint(offset+uint64(len(rows)), 10)) {
			return bad()
		}
		seen := map[uint64]bool{}
		for _, item := range rows {
			row, ok := item.(map[string]any)
			if !ok || len(row) != 3 {
				return bad()
			}
			rawIndex, ok := row["source_index"].(string)
			if !ok {
				return bad()
			}
			index, err := strconv.ParseUint(rawIndex, 10, 64)
			if err != nil || strconv.FormatUint(index, 10) != rawIndex || index >= sourceRows || seen[index] || row["pointer"] != p["pointer"].(string)+"/"+rawIndex {
				return bad()
			}
			seen[index] = true
			if _, ok = row["value"]; !ok {
				return bad()
			}
		}

	case "result_query":
		if m["content_sha256"] != p["expected_sha256"] || m["pointer"] != p["pointer"] || !digest("content_sha256") || !digest("query_sha256") {
			return bad()
		}
		offset, a := number("offset")
		total, b := number("total")
		nodes, c := m["nodes"].([]any)
		complete, d := m["complete"].(bool)
		cursor, e := m["next_cursor"].(string)
		limit, _ := strconv.ParseUint(p["limit"].(string), 10, 64)
		if !a || !b || !c || !d || !e || offset > total || uint64(len(nodes)) > limit || uint64(len(nodes)) > total-offset || complete != (offset+uint64(len(nodes)) == total) {
			return bad()
		}
		query, _ := sbvNativeCanonical(map[string]any{"snapshot": p["expected_sha256"], "pointer": p["pointer"], "limit": p["limit"]})
		hash := strings.TrimPrefix(digestBytes(query), "sha256:")
		if m["query_sha256"] != hash {
			return bad()
		}
		wantOffset := uint64(0)
		inCursor := p["cursor"].(string)
		if inCursor != "" {
			if !strings.HasPrefix(inCursor, hash+":") {
				return bad()
			}
			var err error
			wantOffset, err = strconv.ParseUint(strings.TrimPrefix(inCursor, hash+":"), 10, 64)
			if err != nil {
				return bad()
			}
		}
		if offset != wantOffset || (!complete && (len(nodes) == 0 || cursor != hash+":"+strconv.FormatUint(offset+uint64(len(nodes)), 10))) || (complete && cursor != "") {
			return bad()
		}
		for _, node := range nodes {
			x, ok := node.(map[string]any)
			if !ok {
				return bad()
			}
			if _, ok = x["pointer"].(string); !ok {
				return bad()
			}
			if _, ok = x["type"].(string); !ok {
				return bad()
			}
			if _, ok = x["children"].(string); !ok {
				return bad()
			}
		}
	default:
		return bad()
	}
	return nil
}

// Export uses the exact bytes independently hashed by the native validator.
// Financial values and extension fields are transported, never recomputed.
func ReadSBVExport(path string, inspection json.RawMessage) (json.RawMessage, error) {
	m, err := sqavObject(inspection, maxResponseBytes)
	if err != nil {
		return nil, err
	}
	if m["protocol"] != "symphony.sbv.result-inspect.v1" || m["path"] != path || !filepath.IsAbs(path) {
		return nil, fmt.Errorf("SBV export inspection mismatch")
	}
	raw, err := readTrustedNoFollowRelative("/", strings.TrimPrefix(path, "/"), 128<<20)
	if err != nil {
		return nil, err
	}
	if digestBytes(raw) != "sha256:"+fmt.Sprint(m["file_sha256"]) || strconv.Itoa(len(raw)) != m["bytes"] {
		return nil, fmt.Errorf("SBV result changed after native inspection")
	}
	return raw, nil
}

// SBV owns these richer request schemas; other vectors retain their existing
// transport subset. This checks shape only, not native owner side effects.
func sbvInputShape(shape, p map[string]any) bool { return sbvSchemaShape(shape, p, 0) }

func sbvSchemaShape(schema, value any, depth int) bool {
	s, ok := schema.(map[string]any)
	if !ok || depth > 64 {
		return false
	}
	base := make(map[string]any, len(s))
	for key, rule := range s {
		base[key] = rule
	}
	for _, kind := range []string{"allOf", "anyOf", "oneOf"} {
		if raw, present := s[kind]; present {
			terms, ok := raw.([]any)
			if !ok || len(terms) == 0 {
				return false
			}
			matched := 0
			for _, term := range terms {
				if sbvSchemaShape(term, value, depth+1) {
					matched++
				}
			}
			if (kind == "allOf" && matched != len(terms)) || (kind == "anyOf" && matched == 0) || (kind == "oneOf" && matched != 1) {
				return false
			}
			delete(base, kind)
		}
	}
	if term, present := s["not"]; present {
		if sbvSchemaShape(term, value, depth+1) {
			return false
		}
		delete(base, "not")
	}
	if predicate, present := s["if"]; present {
		branch := "else"
		if sbvSchemaShape(predicate, value, depth+1) {
			branch = "then"
		}
		if rule, present := s[branch]; present && !sbvSchemaShape(rule, value, depth+1) {
			return false
		}
	}
	for _, key := range []string{"if", "then", "else"} {
		delete(base, key)
	}
	if contains, present := s["contains"]; present {
		// JSON Schema applies contains only to arrays; an explicit type remains
		// checked below. Each independent allOf condition keeps its own count.
		if rows, array := value.([]any); array {
			matched := int64(0)
			for _, row := range rows {
				if sbvSchemaShape(contains, row, depth+1) {
					matched++
				}
			}
			minimum, maximum := int64(1), int64(len(rows))
			if raw, present := s["minContains"]; present {
				var ok bool
				minimum, ok = raw.(int64)
				if !ok || minimum < 0 {
					return false
				}
			}
			if raw, present := s["maxContains"]; present {
				var ok bool
				maximum, ok = raw.(int64)
				if !ok || maximum < 0 {
					return false
				}
			}
			if matched < minimum || matched > maximum {
				return false
			}
		}
	}
	for _, key := range []string{"contains", "minContains", "maxContains"} {
		delete(base, key)
	}
	if values, object := value.(map[string]any); object {
		// Object keywords apply even without an explicit type (conditional schema).
		props, _ := s["properties"].(map[string]any)
		bound := make(map[string]any, len(values))
		for key, item := range values {
			if rule, present := props[key]; present {
				if !sbvSchemaShape(rule, item, depth+1) {
					return false
				}
			} else if additional, present := s["additionalProperties"]; present {
				if permitted, boolean := additional.(bool); boolean {
					if !permitted {
						return false
					}
				} else if !sbvSchemaShape(additional, item, depth+1) {
					return false
				}
			}
			bound[key] = map[string]any{"const": item}
		}
		base["properties"] = bound
		base["additionalProperties"] = false
		if _, present := base["required"]; !present {
			base["required"] = []any{}
		}
		if _, present := base["type"]; !present {
			base["type"] = "object"
		}
	} else {
		// These constraints are inapplicable to nonobjects; explicit type remains.
		delete(base, "properties")
		delete(base, "required")
		delete(base, "additionalProperties")
	}
	if rows, array := value.([]any); array {
		if rule, present := s["items"]; present {
			for _, row := range rows {
				if !sbvSchemaShape(rule, row, depth+1) {
					return false
				}
			}
		}
		base["items"] = map[string]any{}
		if _, present := base["type"]; !present {
			base["type"] = "array"
		}
	} else {
		delete(base, "items")
		delete(base, "minItems")
		delete(base, "maxItems")
	}
	if _, text := value.(string); text {
		// Conjunctive constraints need not repeat a type declaration.
		if _, present := base["type"]; !present {
			base["type"] = "string"
		}
	} else {
		delete(base, "minLength")
		delete(base, "maxLength")
		delete(base, "pattern")
		delete(base, "format")
	}
	return sqvTransportShape(base, value, depth)
}

// SBVSchema exposes both sides of an operation and the exact recursive output
// definitions. result_export exposes the portable artifact and stream contracts.
func SBVSchema(prefix, version, operation string) (json.RawMessage, error) {
	inst, err := InspectSBV(prefix, version)
	if err != nil {
		return nil, err
	}
	read := func(name string) (map[string]any, error) {
		raw, e := snvOwnedResource(inst, "share/symphony/schemas/sbv-engine/"+version+"/"+name, sbvAdministrationInterfaceResources[name])
		if e != nil {
			return nil, e
		}
		return sqavObject(raw, maxRequestBytes)
	}
	if operation == "result_export" {
		result, e := read("result.schema.json")
		if e != nil {
			return nil, e
		}
		stream, e := read("pointer-stream.schema.json")
		if e != nil {
			return nil, e
		}
		return SCVCanonical(map[string]any{"result": result, "ndjson_record": stream})
	}
	if !sbvAdministrationInterfaceAdmission[version][operation] {
		return nil, fmt.Errorf("SBV schema operation unavailable")
	}
	all, err := read("admin.schema.json")
	if err != nil {
		return nil, err
	}
	requests, ok := all["requests"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("SBV request schemas missing")
	}
	results, ok := all["results"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("SBV result schemas missing")
	}
	return SCVCanonical(map[string]any{"input": requests[operation], "output": results[operation], "$defs": all["$defs"]})
}

// Native nlohmann JSON uses UTF-8 for U+2028/U+2029. Go's encoder escapes
// these even with HTML escaping off. Match the native hash representation,
// while preserving literal backslash-u text and all other escaped pairs.
func sbvNativeCanonical(value any) ([]byte, error) {
	raw, err := SCVCanonical(value)
	if err != nil {
		return nil, err
	}
	out := make([]byte, 0, len(raw))
	for i := 0; i < len(raw); {
		if raw[i] == '\\' && i+1 < len(raw) {
			if i+6 <= len(raw) && (string(raw[i:i+6]) == "\\u2028" || string(raw[i:i+6]) == "\\u2029") {
				if raw[i+5] == '8' {
					out = append(out, []byte(" ")...)
				} else {
					out = append(out, []byte(" ")...)
				}
				i += 6
				continue
			}
			out = append(out, raw[i], raw[i+1])
			i += 2
			continue
		}
		out = append(out, raw[i])
		i++
	}
	return out, nil
}

// Correspondence checks for the native load's captured delivery. The engine
// validates the complete owner contract and bytes; Go does not reopen storage
// or treat this retained evidence as producer authentication.
func sbvRetainedDeliveryIdentity(value any) (map[string]any, error) {
	bad := func() (map[string]any, error) {
		return nil, fmt.Errorf("SBV retained delivery correspondence mismatch")
	}
	delivery, ok := value.(map[string]any)
	if !ok || delivery["protocol"] != "symphony.sbv.source-delivery.v1" || delivery["origin"] != "retained" || delivery["exactly_once_claimed"] != false || delivery["destination_commit_claimed"] != false {
		return bad()
	}
	reference, rok := delivery["reference"].(map[string]any)
	choices, cok := delivery["choices"].(map[string]any)
	source, sok := delivery["source"].(map[string]any)
	if !rok || !cok || !sok || len(reference) != 3 || source["protocol"] != "symphony.sbv.retained-source.v1" || source["owner_profile"] != "sqv-local-retained-capture.v1" || choices["profile"] != "retained_before_delivery" || choices["checkpoint"] != nil || delivery["acknowledgement"] != choices["processed_ack"] {
		return bad()
	}
	if delivery["acknowledgement"] != "none" && delivery["acknowledgement"] != "dataset_admitted" {
		return bad()
	}
	expected, ok := reference["expected_sha256"].(string)
	path, pok := reference["path"].(string)
	_, ptrOK := reference["pointer"].(string)
	if !ok || !sbvSHA.MatchString(expected) || !pok || !filepath.IsAbs(path) || !ptrOK {
		return bad()
	}
	original, ok := source["original"].(map[string]any)
	if !ok {
		return bad()
	}
	hash, hok := original["source_sha256"].(string)
	originalPath, pok := original["source_path"].(string)
	dataset, dok := original["dataset"].(string)
	if !hok || !sbvSHA.MatchString(hash) || !pok || !filepath.IsAbs(originalPath) || !dok || dataset == "" {
		return bad()
	}
	return map[string]any{"source_path": originalPath, "source_sha256": hash, "dataset": dataset}, nil
}
