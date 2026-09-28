package knowledgeengine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

const SQAVRequestInputProtocol = "symphony.sqav.request-validation-input.v1"
const sqavRequestPayloadBytes = 32768

var sqavRequestSpec = engineSpec{label: "sqav-request-engine", moduleID: "sqav-request-engine", engineID: "symphony-sqav-request", componentKind: "vector_engine", vectorID: "sqav", processProtocol: processProtocol}

func SQAVRequestAdapter(adapter string) bool {
	switch adapter {
	case "fred", "databento_historical", "databento_reference":
		return true
	}
	return false
}
func sqavInvalid() error {
	return fmt.Errorf("SQAV request or result does not match the admitted contract")
}

// Only JSON transport mechanics live here. Native Plan::create owns selections.
func sqavObject(raw []byte, bound int64) (map[string]any, error) {
	if err := validateJSONObject(raw, bound); err != nil {
		return nil, sqavInvalid()
	}
	if err := ValidateSCVBundleUnicode(raw); err != nil {
		return nil, sqavInvalid()
	}
	var m map[string]any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if d.Decode(&m) != nil {
		return nil, sqavInvalid()
	}
	// Native integral JSON normalizes -0; keep exact integer arithmetic in Go.
	var normalize func(any) any
	normalize = func(v any) any {
		switch x := v.(type) {
		case json.Number:
			n, _ := x.Int64()
			return n
		case map[string]any:
			for k, a := range x {
				x[k] = normalize(a)
			}
		case []any:
			for i, a := range x {
				x[i] = normalize(a)
			}
		}
		return v
	}
	normalize(m)
	return m, nil
}
func InspectSQAVRequest(prefix, version string) (Installation, error) {
	if prefix == "" || !sqavRequestInterfaceAdmission[version]["request_validate"] {
		return Installation{}, sqavInvalid()
	}
	s := sqavRequestSpec
	e, err := InspectReceiptV2EntryPoint(prefix, version, ReceiptV2EntryPointSpec{Label: s.label, ComponentID: s.moduleID, ComponentKind: s.componentKind, ModuleID: s.moduleID, PackageID: s.moduleID, VectorID: &s.vectorID, EngineID: &s.engineID, EntryPointID: s.engineID, EntryPointKind: "executable", EntryPointRelativePath: filepath.ToSlash(filepath.Join("libexec", "symphony", s.moduleID, version, s.engineID)), RequiredProtocols: []string{processProtocol}})
	if err != nil {
		return Installation{}, err
	}
	inst := Installation{Role: s.moduleID, ModuleID: s.moduleID, EngineID: s.engineID, Version: version, Prefix: e.Prefix, ReceiptPath: e.ReceiptPath, ReceiptDigest: e.ReceiptDigest, ReceiptProtocol: receiptProtocolV2, ExecutablePath: e.ExecutablePath, ExecutableDigest: e.ExecutableDigest}
	if err = verifyOwnerInterface(inst, sqavRequestInterfaceDigest); err != nil {
		return Installation{}, err
	}
	return inst, nil
}
func SQAVRequestResource(prefix, version, adapter string, templates bool) (Installation, json.RawMessage, error) {
	if !SQAVRequestAdapter(adapter) {
		return Installation{}, nil, sqavInvalid()
	}
	inst, err := InspectSQAVRequest(prefix, version)
	if err != nil {
		return inst, nil, err
	}
	name := "request.schema.json"
	if templates {
		name = "request.templates.json"
	}
	path := "share/symphony/schemas/" + inst.ModuleID + "/" + version + "/" + name
	receiptBytes, err := readTrustedNoFollowRelative(inst.Prefix, "share/symphony/receipts/"+inst.ModuleID+"/"+version+"/install-receipt.json", maxReceiptBytes)
	var receipt receiptV2
	if err != nil || decodeExact(receiptBytes, &receipt) != nil || receipt.ReceiptDigest != inst.ReceiptDigest {
		return Installation{}, nil, sqavInvalid()
	}
	owned := false
	for _, file := range receipt.Files {
		if file.Path == path && file.Kind == "regular" && file.Digest == sqavRequestInterfaceResources[name] {
			owned = true
		}
	}
	if !owned {
		return Installation{}, nil, sqavInvalid()
	}
	data, err := readTrustedNoFollowRelative(inst.Prefix, path, maxRequestBytes)
	if err != nil || digestBytes(data) != sqavRequestInterfaceResources[name] {
		return Installation{}, nil, sqavInvalid()
	}
	m, err := sqavObject(data, maxRequestBytes)
	if err != nil {
		return Installation{}, nil, err
	}
	key := "requests"
	if templates {
		key = "templates"
	}
	variants, ok := m[key].(map[string]any)
	if !ok || variants[adapter] == nil {
		return Installation{}, nil, sqavInvalid()
	}
	selected := variants[adapter]
	if !templates {
		selected = map[string]any{"input": selected, "result": m["result"]}
	}
	out, err := SCVCanonical(selected)
	if err != nil {
		return Installation{}, nil, err
	}
	after, err := InspectSQAVRequest(prefix, version)
	if err != nil || after != inst {
		return Installation{}, nil, sqavInvalid()
	}
	return inst, out, nil
}
func ValidateSQAVRequestResult(payload, result []byte) error {
	p, err := sqavObject(payload, sqavRequestPayloadBytes)
	if err != nil {
		return err
	}
	if requireExactFields(result, []string{"protocol", "adapter", "adapter_id", "adapter_version", "request_digest", "plan_reference", "validation_scope", "provider_observation", "result_digest"}) != nil {
		return sqavInvalid()
	}
	r, err := sqavObject(result, 65536)
	if err != nil {
		return err
	}
	adapter, ok := p["adapter"].(string)
	if !ok || !SQAVRequestAdapter(adapter) {
		return sqavInvalid()
	}
	id, release, refPrefix := "", "0.1.0-dev", "source-request-v1-"
	switch adapter {
	case "fred":
		id = "sqav-fred-cpp"
	case "databento_historical":
		id = "sqav-databento-dbn-cpp"
		release = "0.5.0-dev"
		refPrefix = "sqdh1-sha256-"
	case "databento_reference":
		id = "sqav-databento-reference-cpp"
	}
	canonical, err := SCVCanonical(p)
	if err != nil {
		return err
	}
	ref, ok := r["plan_reference"].(string)
	if !ok || !strings.HasPrefix(ref, refPrefix) || !taggedDigest("sha256:"+strings.TrimPrefix(ref, refPrefix)) || r["protocol"] != sqavRequestInterfaceOutputs["request_validate"] || r["adapter"] != adapter || r["adapter_id"] != id || r["adapter_version"] != release || r["request_digest"] != digestBytes(canonical) || r["validation_scope"] != "native_request_only" || r["provider_observation"] != "not_performed" {
		return sqavInvalid()
	}
	digest := r["result_digest"]
	delete(r, "result_digest")
	canonical, err = SCVCanonical(r)
	if err != nil || digest != digestBytes(canonical) {
		return sqavInvalid()
	}
	return nil
}
func InvokeSQAVRequest(ctx context.Context, prefix, version, cwd string, payload []byte) (Response, error) {
	p, err := sqavObject(payload, sqavRequestPayloadBytes)
	if err != nil {
		return Response{}, err
	}
	// Canonicalize the transport once, retaining native integer/string identity.
	payload, err = SCVCanonical(p)
	if err != nil {
		return Response{}, err
	}
	inst, err := InspectSQAVRequest(prefix, version)
	if err != nil {
		return Response{}, err
	}
	r, invokeErr := invokeResolved(ctx, sqavRequestSpec, inst.ExecutablePath, version, cwd, "request_validate", payload)
	after, err := InspectSQAVRequest(prefix, version)
	if err != nil || after != inst {
		return Response{}, sqavInvalid()
	}
	if invokeErr != nil {
		return Response{}, invokeErr
	}
	if err = ValidateSQAVRequestResult(payload, r.Result); err != nil {
		return Response{}, err
	}
	return r, nil
}
