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
	"unicode/utf8"
)

// SQVAdministrationOperation maps one declared action to its semantic owner.
// The shared code handles receipts, JSON transport and result contracts only.
type SQVAdministrationOperation struct {
	Operation, Domain, Leaf, ModuleID, VectorID, EngineID, InputProtocol, OutputProtocol, BackendID string
	admission                                                                                       map[string]map[string]bool
	declaration                                                                                     string
	resources                                                                                       map[string]string
}

var SQVAdministrationOperations = []SQVAdministrationOperation{
	{"metadata_validate", "metadata", "validate", "sqmv-metadata-engine", "sqmv", "symphony-sqmv-metadata", "symphony.sqmv.metadata-validate-input.v1", "symphony.sqmv.metadata-validate.v1", "engop:symphony:sqmv-metadata.validate", sqmvMetadataInterfaceAdmission, sqmvMetadataInterfaceDigest, sqmvMetadataInterfaceResources},
	{"metadata_inspect", "metadata", "inspect", "sqmv-metadata-engine", "sqmv", "symphony-sqmv-metadata", "symphony.sqmv.metadata-inspect-input.v1", "symphony.sqmv.metadata-inspect.v1", "engop:symphony:sqmv-metadata.inspect", sqmvMetadataInterfaceAdmission, sqmvMetadataInterfaceDigest, sqmvMetadataInterfaceResources},
	{"flow_validate", "flow", "validate", "sqfv-flow-engine", "sqfv", "symphony-sqfv-flow", "symphony.sqfv.flow-validate-input.v1", "symphony.sqfv.flow-validate.v1", "engop:symphony:sqfv-flow.validate", sqfvFlowInterfaceAdmission, sqfvFlowInterfaceDigest, sqfvFlowInterfaceResources},
	{"conversion_validate", "conversion", "validate", "sqtv-conversion-engine", "sqtv", "symphony-sqtv-conversion", "symphony.sqtv.conversion-validate-input.v1", "symphony.sqtv.conversion-validate.v1", "engop:symphony:sqtv-conversion.validate", sqtvConversionInterfaceAdmission, sqtvConversionInterfaceDigest, sqtvConversionInterfaceResources},
	{"store_inspect", "store", "inspect", "sqpv-inspection-engine", "sqpv", "symphony-sqpv-inspection", "symphony.sqpv.store-inspect-input.v1", "symphony.sqpv.store-inspect.v1", "engop:symphony:sqpv-inspection.inspect", sqpvInspectionInterfaceAdmission, sqpvInspectionInterfaceDigest, sqpvInspectionInterfaceResources},
	{"checkpoint_inspect", "checkpoint", "inspect", "sqdv-checkpoint-engine", "sqdv", "symphony-sqdv-checkpoint", "symphony.sqdv.checkpoint-inspect-input.v1", "symphony.sqdv.checkpoint-inspect.v1", "engop:symphony:sqdv-checkpoint.inspect", sqdvCheckpointInterfaceAdmission, sqdvCheckpointInterfaceDigest, sqdvCheckpointInterfaceResources},
	{"attempts_inspect", "acquisition", "attempts", "sqav-attempt-engine", "sqav", "symphony-sqav-attempt", "symphony.sqav.attempts-inspect-input.v1", "symphony.sqav.attempts-inspect.v1", "engop:symphony:sqav-attempt.inspect", sqavAttemptInterfaceAdmission, sqavAttemptInterfaceDigest, sqavAttemptInterfaceResources},
}

func sqvAdminInvalid() error {
	return fmt.Errorf("SQV administration request or result violates its exact contract")
}
func sqvAdminOperation(operation string) (SQVAdministrationOperation, error) {
	for _, op := range SQVAdministrationOperations {
		if op.Operation == operation {
			return op, nil
		}
	}
	return SQVAdministrationOperation{}, sqvAdminInvalid()
}
func (op SQVAdministrationOperation) spec() engineSpec {
	return engineSpec{label: op.ModuleID, moduleID: op.ModuleID, engineID: op.EngineID, componentKind: "vector_engine", vectorID: op.VectorID, processProtocol: processProtocol}
}
func InspectSQVAdministration(prefix, version, operation string) (Installation, error) {
	op, err := sqvAdminOperation(operation)
	if err != nil || prefix == "" || !op.admission[version][operation] {
		return Installation{}, sqvAdminInvalid()
	}
	s := op.spec()
	e, err := InspectReceiptV2EntryPoint(prefix, version, ReceiptV2EntryPointSpec{Label: s.label, ComponentID: s.moduleID, ComponentKind: s.componentKind, ModuleID: s.moduleID, PackageID: s.moduleID, VectorID: &s.vectorID, EngineID: &s.engineID, EntryPointID: s.engineID, EntryPointKind: "executable", EntryPointRelativePath: filepath.ToSlash(filepath.Join("libexec", "symphony", s.moduleID, version, s.engineID)), RequiredProtocols: []string{processProtocol}})
	if err != nil {
		return Installation{}, err
	}
	inst := Installation{Role: s.moduleID, ModuleID: s.moduleID, EngineID: s.engineID, Version: version, Prefix: e.Prefix, ReceiptPath: e.ReceiptPath, ReceiptDigest: e.ReceiptDigest, ReceiptProtocol: receiptProtocolV2, ExecutablePath: e.ExecutablePath, ExecutableDigest: e.ExecutableDigest}
	if err = verifyOwnerInterface(inst, op.declaration); err != nil {
		return Installation{}, err
	}
	return inst, nil
}
func SQVAdministrationResource(prefix, version, operation string, templates bool) (Installation, json.RawMessage, error) {
	op, err := sqvAdminOperation(operation)
	if err != nil {
		return Installation{}, nil, err
	}
	inst, err := InspectSQVAdministration(prefix, version, operation)
	if err != nil {
		return inst, nil, err
	}
	name := "admin.schema.json"
	if templates {
		name = "admin.templates.json"
	}
	path := "share/symphony/schemas/" + inst.ModuleID + "/" + version + "/" + name
	receiptBytes, err := readTrustedNoFollowRelative(inst.Prefix, "share/symphony/receipts/"+inst.ModuleID+"/"+version+"/install-receipt.json", maxReceiptBytes)
	var receipt receiptV2
	if err != nil || decodeExact(receiptBytes, &receipt) != nil || receipt.ReceiptDigest != inst.ReceiptDigest {
		return Installation{}, nil, sqvAdminInvalid()
	}
	owned := false
	for _, f := range receipt.Files {
		if f.Path == path && f.Kind == "regular" && f.Digest == op.resources[name] {
			owned = true
		}
	}
	if !owned {
		return Installation{}, nil, sqvAdminInvalid()
	}
	data, err := readTrustedNoFollowRelative(inst.Prefix, path, maxRequestBytes)
	if err != nil || digestBytes(data) != op.resources[name] {
		return Installation{}, nil, sqvAdminInvalid()
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
	if !ok || variants[operation] == nil {
		return Installation{}, nil, sqvAdminInvalid()
	}
	selected := variants[operation]
	if !templates {
		results, ok := m["results"].(map[string]any)
		if !ok || results[operation] == nil {
			return Installation{}, nil, sqvAdminInvalid()
		}
		selected = map[string]any{"input": selected, "result": results[operation]}
	}
	out, err := SCVCanonical(selected)
	if err != nil {
		return Installation{}, nil, err
	}
	after, err := InspectSQVAdministration(prefix, version, operation)
	if err != nil || after != inst {
		return Installation{}, nil, sqvAdminInvalid()
	}
	return inst, out, nil
}

// This intentionally small JSON Schema transport subset admits only the
// keywords used by receipt-owned, compiled-digest SQV resources. It neither
// resolves external schemas nor implements native vector constraints.
func sqvTransportShape(schema, value any, depth int) bool {
	s, ok := schema.(map[string]any)
	if !ok || depth > 64 {
		return false
	}
	for k := range s {
		switch k {
		case "type", "const", "enum", "anyOf", "properties", "required", "additionalProperties", "pattern", "maxLength", "minLength", "items", "minItems", "maxItems", "description", "format":
		default:
			return false
		}
	}
	if c, ok := s["const"]; ok && !reflect.DeepEqual(c, value) {
		return false
	}
	if variants, ok := s["anyOf"]; ok {
		a, ok := variants.([]any)
		if !ok || len(a) == 0 {
			return false
		}
		matched := false
		for _, v := range a {
			if sqvTransportShape(v, value, depth+1) {
				matched = true
			}
		}
		if !matched {
			return false
		}
	}
	if options, ok := s["enum"]; ok {
		a, ok := options.([]any)
		if !ok {
			return false
		}
		found := false
		for _, v := range a {
			if reflect.DeepEqual(v, value) {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	count := func(k string, n int) bool {
		v, ok := s[k]
		if !ok {
			return true
		}
		limit, ok := v.(int64)
		if !ok || limit < 0 {
			return false
		}
		if strings.HasPrefix(k, "min") {
			return int64(n) >= limit
		}
		return int64(n) <= limit
	}
	switch s["type"] {
	case nil:
	case "null":
		if value != nil {
			return false
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return false
		}
	case "string":
		str, ok := value.(string)
		if !ok || !count("minLength", utf8.RuneCountInString(str)) || !count("maxLength", utf8.RuneCountInString(str)) {
			return false
		}
		if pattern, ok := s["pattern"]; ok {
			p, ok := pattern.(string)
			if !ok {
				return false
			}
			r, err := regexp.Compile(p)
			if err != nil || !r.MatchString(str) {
				return false
			}
		}
		if f, ok := s["format"]; ok {
			if f != "uint64-decimal" {
				return false
			}
			n, err := strconv.ParseUint(str, 10, 64)
			if err != nil || strconv.FormatUint(n, 10) != str {
				return false
			}
		}
	case "array":
		a, ok := value.([]any)
		if !ok || !count("minItems", len(a)) || !count("maxItems", len(a)) {
			return false
		}
		for _, v := range a {
			if !sqvTransportShape(s["items"], v, depth+1) {
				return false
			}
		}
	case "object":
		m, ok := value.(map[string]any)
		if !ok || s["additionalProperties"] != false {
			return false
		}
		props, ok := s["properties"].(map[string]any)
		if !ok {
			return false
		}
		req, ok := s["required"].([]any)
		if !ok {
			return false
		}
		for _, k := range req {
			key, ok := k.(string)
			if !ok {
				return false
			}
			if _, ok := m[key]; !ok {
				return false
			}
		}
		for k, v := range m {
			rule, ok := props[k]
			if !ok || !sqvTransportShape(rule, v, depth+1) {
				return false
			}
		}
	default:
		return false
	}
	return true
}
func ValidateSQVAdministrationResult(operation string, payload, result, contract []byte) error {
	op, err := sqvAdminOperation(operation)
	if err != nil {
		return err
	}
	p, err := sqavObject(payload, 262144)
	if err != nil {
		return err
	}
	r, err := sqavObject(result, 450000)
	if err != nil {
		return err
	}
	s, err := sqavObject(contract, maxRequestBytes)
	if err != nil {
		return err
	}
	if !sqvTransportShape(s["input"], p, 0) || !sqvTransportShape(s["result"], r, 0) || r["operation"] != operation || r["owner"] != op.VectorID || r["protocol"] != op.OutputProtocol || r["persistent_mutation"] != false || r["provider_observation"] != "not_performed" {
		return sqvAdminInvalid()
	}
	canonical, err := SCVCanonical(p)
	if err != nil || r["request_digest"] != digestBytes(canonical) {
		return sqvAdminInvalid()
	}
	digest := r["result_digest"]
	delete(r, "result_digest")
	canonical, err = SCVCanonical(r)
	if err != nil || digest != digestBytes(canonical) {
		return sqvAdminInvalid()
	}
	return nil
}
func InvokeSQVAdministration(ctx context.Context, prefix, version, cwd, operation string, payload []byte) (Response, error) {
	op, err := sqvAdminOperation(operation)
	if err != nil {
		return Response{}, err
	}
	p, err := sqavObject(payload, 262144)
	if err != nil {
		return Response{}, err
	}
	payload, err = SCVCanonical(p)
	if err != nil {
		return Response{}, err
	}
	inst, contract, err := SQVAdministrationResource(prefix, version, operation, false)
	if err != nil {
		return Response{}, err
	}
	schema, err := sqavObject(contract, maxRequestBytes)
	if err != nil || !sqvTransportShape(schema["input"], p, 0) {
		return Response{}, sqvAdminInvalid()
	}
	r, invokeErr := invokeResolved(ctx, op.spec(), inst.ExecutablePath, version, cwd, operation, payload)
	after, err := InspectSQVAdministration(prefix, version, operation)
	if err != nil || after != inst {
		return Response{}, sqvAdminInvalid()
	}
	if invokeErr != nil {
		return Response{}, invokeErr
	}
	if err = ValidateSQVAdministrationResult(operation, payload, r.Result, contract); err != nil {
		return Response{}, err
	}
	return r, nil
}
