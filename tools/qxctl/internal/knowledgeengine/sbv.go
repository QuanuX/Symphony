package knowledgeengine

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var SBVOperations = []string{"capabilities", "run", "compose", "result_inspect", "result_query", "evaluate", "catalogue", "compose_joint", "economics", "book", "liquidity", "allocation_economics"}
var sbvSHA = regexp.MustCompile(`^[0-9a-f]{64}$`)

func sbvSpec() engineSpec {
	return engineSpec{label: "SBV", moduleID: "sbv-engine", vectorID: "sbv", engineID: "symphony-sbv", componentKind: "vector_engine", processProtocol: processProtocol, operationTimeoutByVersion: map[string]time.Duration{"0.6.0-dev": 5 * time.Minute}}
}
func InspectSBV(prefix, version string) (Installation, error) {
	s := sbvSpec()
	if prefix == "" || sbvAdministrationInterfaceAdmission[version] == nil {
		return Installation{}, fmt.Errorf("SBV exact release required")
	}
	e, err := InspectReceiptV2EntryPoint(prefix, version, ReceiptV2EntryPointSpec{Label: s.label, ComponentID: s.moduleID, ComponentKind: s.componentKind, ModuleID: s.moduleID, PackageID: s.moduleID, VectorID: &s.vectorID, EngineID: &s.engineID, EntryPointID: s.engineID, EntryPointKind: "executable", EntryPointRelativePath: filepath.ToSlash(filepath.Join("libexec", "symphony", s.moduleID, version, s.engineID)), RequiredProtocols: []string{processProtocol}})
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
	case "run", "compose", "evaluate", "compose_joint", "economics", "book", "liquidity", "allocation_economics", "result_inspect":
		target := "output_path"
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
		query, _ := SCVCanonical(map[string]any{"snapshot": p["expected_sha256"], "pointer": p["pointer"], "limit": p["limit"]})
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

// The only open transport object is the explicitly named user extension.
// Native validation still enforces portable strings/booleans/null/containers.
func sbvInputShape(shape, p map[string]any) bool {
	if ext, present := p["extensions"]; present {
		if _, ok := ext.(map[string]any); !ok {
			return false
		}
		props, ok := shape["properties"].(map[string]any)
		if !ok {
			return false
		}
		if _, ok = props["extensions"]; !ok {
			return false
		}
		props["extensions"] = map[string]any{"const": ext}
	}
	return sqvTransportShape(shape, p, 0)
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
