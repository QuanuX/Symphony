package knowledgeengine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"
)

const SRRAPIVInputProtocol = "symphony.srrapiv.request.v1"
const SRRAPIVOutputProtocol = "symphony.srrapiv.result.v1"
const SRRAPIVModuleID = "srrapiv-runtime-cpp"
const SRRAPIVEngineID = "symphony-srrapiv"
const SRRAPIVVersion = "0.1.0-dev"
const srrapivJSONValues = 65536

// SRRAPIVOperation describes control transport, never SDK or trading semantics.
type SRRAPIVOperation struct {
	Operation, Domain, Leaf, Interaction string
	Mutating, Runtime                    bool
}

var SRRAPIVOperations = []SRRAPIVOperation{
	{"catalog_inspect", "catalog", "inspect", "inspect", false, false},
	{"api_describe", "api", "describe", "discover", false, false},
	{"binding_validate", "binding", "validate", "validate", false, false},
	{"binding_compare", "binding", "compare", "inspect", false, false},
	{"coverage_assess", "coverage", "assess", "inspect", false, false},
	{"runtime_start", "runtime", "start", "lifecycle", true, true},
	{"runtime_status", "runtime", "status", "inspect", false, true},
	{"runtime_invoke", "runtime", "invoke", "invoke", true, true},
	{"runtime_events", "runtime", "events", "inspect", false, true},
	{"runtime_stop", "runtime", "stop", "lifecycle", true, true},
	{"runtime_attempts", "runtime", "attempts", "recover", false, true},
}

var srrapivSpec = engineSpec{
	label: SRRAPIVModuleID, moduleID: SRRAPIVModuleID, engineID: SRRAPIVEngineID,
	componentKind: "vector_engine", vectorID: "srrapiv", processProtocol: processProtocol,
	operationTimeoutByVersion:   map[string]time.Duration{SRRAPIVVersion: 30 * time.Second},
	responseJSONValuesByVersion: map[string]int{SRRAPIVVersion: srrapivJSONValues},
}

func srrapivInvalid() error {
	return fmt.Errorf("SRRAPIV request or result violates its exact installed contract")
}

// ReadSRRAPIVPayload uses the native owner's declared value budget and shared
// no-follow file transport. The payload and process envelope remain byte-bound.
func ReadSRRAPIVPayload(path string) ([]byte, error) {
	if path == "" {
		return nil, srrapivInvalid()
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, srrapivInvalid()
	}
	raw, err := ReadRepositoryJSONObject(filepath.Dir(abs), filepath.Base(abs), maxRequestBytes, srrapivJSONValues)
	if err != nil {
		return nil, srrapivInvalid()
	}
	return raw, nil
}

func srrapivObject(raw []byte, bound int64) (map[string]any, error) {
	if validateJSONObjectWithValueLimit(raw, bound, srrapivJSONValues) != nil || ValidateSCVBundleUnicode(raw) != nil {
		return nil, srrapivInvalid()
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var object map[string]any
	if decoder.Decode(&object) != nil {
		return nil, srrapivInvalid()
	}
	var normalize func(any) any
	normalize = func(value any) any {
		switch v := value.(type) {
		case json.Number:
			n, _ := v.Int64()
			return n
		case map[string]any:
			for key, element := range v {
				v[key] = normalize(element)
			}
		case []any:
			for i, element := range v {
				v[i] = normalize(element)
			}
		}
		return value
	}
	normalize(object)
	return object, nil
}

func srrapivOperation(operation string) (SRRAPIVOperation, error) {
	for _, op := range SRRAPIVOperations {
		if op.Operation == operation {
			return op, nil
		}
	}
	return SRRAPIVOperation{}, srrapivInvalid()
}

func InspectSRRAPIV(prefix, version, operation string) (Installation, error) {
	if _, err := srrapivOperation(operation); err != nil || prefix == "" || version != SRRAPIVVersion {
		return Installation{}, srrapivInvalid()
	}
	s := srrapivSpec
	e, err := InspectReceiptV2EntryPoint(prefix, version, ReceiptV2EntryPointSpec{
		Label: s.label, ComponentID: s.moduleID, ComponentKind: s.componentKind,
		ModuleID: s.moduleID, PackageID: s.moduleID, VectorID: &s.vectorID, EngineID: &s.engineID,
		EntryPointID: s.engineID, EntryPointKind: "executable",
		EntryPointRelativePath: filepath.ToSlash(filepath.Join("libexec", "symphony", s.moduleID, version, s.engineID)),
		RequiredProtocols:      []string{processProtocol},
	})
	if err != nil {
		return Installation{}, err
	}
	inst := Installation{Role: s.moduleID, ModuleID: s.moduleID, EngineID: s.engineID, Version: version,
		Prefix: e.Prefix, ReceiptPath: e.ReceiptPath, ReceiptDigest: e.ReceiptDigest, ReceiptProtocol: receiptProtocolV2,
		ExecutablePath: e.ExecutablePath, ExecutableDigest: e.ExecutableDigest}
	if err = verifyOwnerInterface(inst, srrapivInterfaceDigest); err != nil {
		return Installation{}, err
	}
	return inst, nil
}

func SRRAPIVResource(prefix, version, operation string, templates bool) (Installation, json.RawMessage, error) {
	inst, err := InspectSRRAPIV(prefix, version, operation)
	if err != nil {
		return Installation{}, nil, err
	}
	name := "srrapiv.schema.json"
	if templates {
		name = "srrapiv.templates.json"
	}
	path := "share/symphony/schemas/" + inst.ModuleID + "/" + version + "/" + name
	receiptBytes, err := readTrustedNoFollowRelative(inst.Prefix, "share/symphony/receipts/"+inst.ModuleID+"/"+version+"/install-receipt.json", maxReceiptBytes)
	var receipt receiptV2
	if err != nil || decodeExact(receiptBytes, &receipt) != nil || receipt.ReceiptDigest != inst.ReceiptDigest {
		return Installation{}, nil, srrapivInvalid()
	}
	owned := false
	for _, f := range receipt.Files {
		if f.Path == path && f.Kind == "regular" && f.Digest == srrapivInterfaceResources[name] {
			owned = true
		}
	}
	if !owned {
		return Installation{}, nil, srrapivInvalid()
	}
	data, err := readTrustedNoFollowRelative(inst.Prefix, path, maxRequestBytes)
	if err != nil || digestBytes(data) != srrapivInterfaceResources[name] {
		return Installation{}, nil, srrapivInvalid()
	}
	m, err := srrapivObject(data, maxRequestBytes)
	if err != nil {
		return Installation{}, nil, srrapivInvalid()
	}
	key := "requests"
	if templates {
		key = "templates"
	}
	variants, ok := m[key].(map[string]any)
	if !ok || variants[operation] == nil {
		return Installation{}, nil, srrapivInvalid()
	}
	selected := variants[operation]
	if !templates {
		results, ok := m["results"].(map[string]any)
		if !ok || results[operation] == nil {
			return Installation{}, nil, srrapivInvalid()
		}
		selected = map[string]any{"input": selected, "result": results[operation]}
	}
	out, err := SCVCanonical(selected)
	if err != nil {
		return Installation{}, nil, srrapivInvalid()
	}
	after, err := InspectSRRAPIV(prefix, version, operation)
	if err != nil || after != inst {
		return Installation{}, nil, srrapivInvalid()
	}
	return inst, out, nil
}

// ValidateSRRAPIVResult checks transport and correspondence. Domain acceptance,
// authorization, SDK dispatch and provider observations belong to native C++.
func ValidateSRRAPIVResult(operation, version string, payload, result, contract []byte) error {
	if _, err := srrapivOperation(operation); err != nil || version != SRRAPIVVersion {
		return srrapivInvalid()
	}
	p, err := srrapivObject(payload, maxRequestBytes)
	if err != nil {
		return srrapivInvalid()
	}
	r, err := srrapivObject(result, maxResponseBytes)
	if err != nil {
		return srrapivInvalid()
	}
	s, err := srrapivObject(contract, maxRequestBytes)
	if err != nil || !srrapivTransportShape(s["input"], p, 0) || !srrapivTransportShape(s["result"], r, 0) ||
		p["protocol"] != SRRAPIVInputProtocol || len(r) != 6 || r["protocol"] != SRRAPIVOutputProtocol ||
		r["operation"] != operation || r["engine_version"] != version {
		return srrapivInvalid()
	}
	if _, ok := r["result"].(map[string]any); !ok {
		return srrapivInvalid()
	}
	canonical, err := SCVCanonical(p)
	if err != nil || r["request_digest"] != digestBytes(canonical) {
		return srrapivInvalid()
	}
	digest := r["result_digest"]
	delete(r, "result_digest")
	canonical, err = SCVCanonical(r)
	if err != nil || digest != digestBytes(canonical) {
		return srrapivInvalid()
	}
	return nil
}

func InvokeSRRAPIV(ctx context.Context, prefix, version, cwd, operation string, payload []byte) (Response, error) {
	if _, err := srrapivOperation(operation); err != nil {
		return Response{}, err
	}
	p, err := srrapivObject(payload, maxRequestBytes)
	if err != nil || p["protocol"] != SRRAPIVInputProtocol {
		return Response{}, srrapivInvalid()
	}
	payload, err = SCVCanonical(p)
	if err != nil {
		return Response{}, srrapivInvalid()
	}
	inst, contract, err := SRRAPIVResource(prefix, version, operation, false)
	if err != nil {
		return Response{}, err
	}
	schema, err := srrapivObject(contract, maxRequestBytes)
	if err != nil || !srrapivTransportShape(schema["input"], p, 0) {
		return Response{}, srrapivInvalid()
	}
	response, invokeErr := invokeResolved(ctx, srrapivSpec, inst.ExecutablePath, version, cwd, operation, payload)
	after, err := InspectSRRAPIV(prefix, version, operation)
	if err != nil || after != inst {
		return Response{}, srrapivInvalid()
	}
	if invokeErr != nil {
		return Response{}, invokeErr
	}
	if err = ValidateSRRAPIVResult(operation, version, payload, response.Result, contract); err != nil {
		return Response{}, err
	}
	return response, nil
}
