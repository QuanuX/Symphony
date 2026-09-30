package knowledgeengine

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

const snvJSONValues = 262144

// SNVOperation identifies a native operation. Domain and Leaf describe its
// administrative route; typed cluster transitions share the validate leaf.
type SNVOperation struct {
	Domain, Leaf, Owner, Operation, BackendID, InputProtocol, OutputProtocol string
}

var SNVOperations = []SNVOperation{
	{"", "observe", "local-observer", "observe", "engop:symphony:snv-local-observer.observe", "symphony.snv.local-observe-input.v1", "symphony.snv.local-observe.v1"},
	{"identity", "validate", "sniv", "identity_validate", "engop:symphony:sniv.identity-validate", "symphony.sniv.identity-validate-input.v1", "symphony.sniv.identity-validate.v1"},
	{"resources", "validate", "snrv", "resources_validate", "engop:symphony:snrv.resources-validate", "symphony.snrv.resources-validate-input.v1", "symphony.snrv.resources-validate.v1"},
	{"clusters", "validate", "sciv", "sciv_validate", "engop:symphony:sciv.sciv-validate", "symphony.snv.sciv.evidence.v1", "symphony.snv.sciv.result.v1"},
	{"clusters", "validate", "sciv", "sciv_transition", "engop:symphony:sciv.sciv-transition", "symphony.snv.sciv.transition.v1", "symphony.snv.sciv.transition-result.v1"},
	{"names", "validate", "scnv", "names_validate", "engop:symphony:scnv.names-validate", "scnv.names.v1", "scnv.names-result.v1"},
	{"names", "resolve", "scnv", "names_resolve", "engop:symphony:scnv.names-resolve", "scnv.resolve.v1", "scnv.resolve-result.v1"},
	{"", "inspect", "snv", "snv_inspect", "engop:symphony:snv.snv-inspect", "symphony.snv.inspect-input.v1", "symphony.snv.inspect.v1"},
	{"evidence", "prepare", "snv", "snv_evidence_plan", "engop:symphony:snv.snv-evidence-plan", "symphony.snv.evidence-plan-input.v1", "symphony.snv.evidence-plan.v1"},
	{"state", "plan", "snv", "snv_state_plan", "engop:symphony:snv.snv-state-plan", "symphony.snv.state-plan-input.v1", "symphony.snv.state-plan.v1"},
	{"state", "apply", "snv", "snv_state_reduce", "engop:symphony:snv.snv-state-reduce", "symphony.snv.state-reduce-input.v1", "symphony.snv.state-transition.v1"},
}

type snvInterface struct {
	admission   map[string]map[string]bool
	outputs     map[string]string
	declaration string
	resources   map[string]string
}

func snvInterfaceFor(owner string) (snvInterface, error) {
	switch owner {
	case "local-observer":
		return snvInterface{snvObserverAdministrationInterfaceAdmission, snvObserverAdministrationInterfaceOutputs, snvObserverAdministrationInterfaceDigest, snvObserverAdministrationInterfaceResources}, nil
	case "sniv":
		return snvInterface{snivAdministrationInterfaceAdmission, snivAdministrationInterfaceOutputs, snivAdministrationInterfaceDigest, snivAdministrationInterfaceResources}, nil
	case "snrv":
		return snvInterface{snrvAdministrationInterfaceAdmission, snrvAdministrationInterfaceOutputs, snrvAdministrationInterfaceDigest, snrvAdministrationInterfaceResources}, nil
	case "sciv":
		return snvInterface{scivAdministrationInterfaceAdmission, scivAdministrationInterfaceOutputs, scivAdministrationInterfaceDigest, scivAdministrationInterfaceResources}, nil
	case "scnv":
		return snvInterface{scnvAdministrationInterfaceAdmission, scnvAdministrationInterfaceOutputs, scnvAdministrationInterfaceDigest, scnvAdministrationInterfaceResources}, nil
	case "snv":
		return snvInterface{snvAdministrationInterfaceAdmission, snvAdministrationInterfaceOutputs, snvAdministrationInterfaceDigest, snvAdministrationInterfaceResources}, nil
	}
	return snvInterface{}, fmt.Errorf("Unsupported SNV owner %q", owner)
}

func snvEngineSpec(owner string) engineSpec {
	if owner == "local-observer" {
		return engineSpec{label: "SNV-local-observer", moduleID: "snv-local-observer", engineID: "symphony-snv-local-observer", componentKind: "module", processProtocol: processProtocol, responseJSONValuesByVersion: map[string]int{"0.1.0-dev": snvJSONValues}}
	}
	return engineSpec{label: strings.ToUpper(owner), moduleID: owner + "-engine", engineID: "symphony-" + owner, componentKind: "vector_engine", vectorID: owner, processProtocol: processProtocol, responseJSONValuesByVersion: map[string]int{"0.1.0-dev": snvJSONValues}}
}

// ParseSNVObject applies the admitted transport profile and rejects spelling
// that encoding/json would silently replace. It performs no domain reduction.
func ParseSNVObject(raw []byte) (map[string]any, error) { return snvObject(raw, maxResponseBytes) }
func snvObject(raw []byte, maximum int64) (map[string]any, error) {
	if err := validateJSONObjectWithValueLimit(raw, maximum, snvJSONValues); err != nil {
		return nil, err
	}
	if err := ValidateSCVBundleUnicode(raw); err != nil {
		return nil, err
	}
	var object map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&object); err != nil {
		return nil, err
	}
	var normalize func(any) any
	normalize = func(v any) any {
		switch x := v.(type) {
		case json.Number:
			n, _ := x.Int64()
			return n
		case map[string]any:
			for k, item := range x {
				x[k] = normalize(item)
			}
		case []any:
			for i, item := range x {
				x[i] = normalize(item)
			}
		}
		return v
	}
	normalize(object)
	return object, nil
}

func snvOperationFor(owner, operation string) (SNVOperation, error) {
	for _, op := range SNVOperations {
		if op.Owner == owner && op.Operation == operation {
			return op, nil
		}
	}
	return SNVOperation{}, fmt.Errorf("Unsupported SNV owner operation %s/%s", owner, operation)
}

// InspectSNV validates a complete, exact, inactive receipt-owned installation.
// Child installations and docking are independent of parent installation.
func InspectSNV(prefix, version, owner string) (Installation, error) {
	compiled, err := snvInterfaceFor(owner)
	if err != nil {
		return Installation{}, err
	}
	if prefix == "" || len(compiled.admission[version]) == 0 {
		return Installation{}, fmt.Errorf("Unsupported exact SNV release")
	}
	spec := snvEngineSpec(owner)
	var vectorID, engineID *string
	if owner != "local-observer" {
		vectorID, engineID = &spec.vectorID, &spec.engineID
	}
	e, err := InspectReceiptV2EntryPoint(prefix, version, ReceiptV2EntryPointSpec{Label: spec.label, ComponentID: spec.moduleID, ComponentKind: spec.componentKind, ModuleID: spec.moduleID, PackageID: spec.moduleID, VectorID: vectorID, EngineID: engineID, EntryPointID: spec.engineID, EntryPointKind: "executable", EntryPointRelativePath: filepath.ToSlash(filepath.Join("libexec", "symphony", spec.moduleID, version, spec.engineID)), RequiredProtocols: []string{processProtocol}})
	if err != nil {
		return Installation{}, err
	}
	inst := Installation{Role: spec.moduleID, ModuleID: spec.moduleID, EngineID: spec.engineID, Version: version, Prefix: e.Prefix, ReceiptPath: e.ReceiptPath, ReceiptDigest: e.ReceiptDigest, ReceiptProtocol: receiptProtocolV2, ExecutablePath: e.ExecutablePath, ExecutableDigest: e.ExecutableDigest}
	path := "share/symphony/" + inst.ModuleID + "/" + version + "/OWNER-INTERFACE.json"
	data, err := snvOwnedResource(inst, path, "")
	if err != nil {
		return Installation{}, err
	}
	declaration, err := snvObject(data, maxRequestBytes)
	if err != nil {
		return Installation{}, err
	}
	canonical, err := SCVCanonical(declaration)
	if err != nil || digestBytes(canonical) != compiled.declaration {
		return Installation{}, fmt.Errorf("SNV interface differs from compiled admission")
	}
	if declaration["module_id"] != inst.ModuleID || declaration["engine_id"] != inst.EngineID || declaration["current_version"] != version {
		return Installation{}, fmt.Errorf("SNV interface identity mismatch")
	}
	for name, digest := range compiled.resources {
		if _, err := snvOwnedResource(inst, "share/symphony/"+inst.ModuleID+"/"+version+"/schemas/v1/"+name, digest); err != nil {
			return Installation{}, err
		}
	}
	return inst, nil
}

func snvOwnedResource(inst Installation, path, expectedDigest string) ([]byte, error) {
	receiptPath := "share/symphony/receipts/" + inst.ModuleID + "/" + inst.Version + "/install-receipt.json"
	raw, err := readTrustedNoFollowRelative(inst.Prefix, receiptPath, maxReceiptBytes)
	if err != nil {
		return nil, err
	}
	var receipt receiptV2
	if decodeExact(raw, &receipt) != nil || receipt.ReceiptDigest != inst.ReceiptDigest {
		return nil, fmt.Errorf("SNV receipt changed")
	}
	for _, file := range receipt.Files {
		if file.Path != path || file.Kind != "regular" {
			continue
		}
		data, err := readTrustedNoFollowRelative(inst.Prefix, path, maxRequestBytes)
		if err != nil {
			return nil, err
		}
		if uint64(len(data)) != file.Size || digestBytes(data) != file.Digest || (expectedDigest != "" && file.Digest != expectedDigest) {
			return nil, fmt.Errorf("SNV resource differs from exact owned bytes")
		}
		after, err := readTrustedNoFollowRelative(inst.Prefix, receiptPath, maxReceiptBytes)
		if err != nil || !bytes.Equal(after, raw) {
			return nil, fmt.Errorf("SNV receipt changed during resource inspection")
		}
		return data, nil
	}
	return nil, fmt.Errorf("SNV resource is not receipt owned")
}

// SNVResource returns the selected exact installed template or input/result
// schemas. Local definition roots are kept explicit for standard schema users.
func SNVResource(prefix, version, owner, operation string, templates bool) (Installation, json.RawMessage, error) {
	if _, err := snvOperationFor(owner, operation); err != nil {
		return Installation{}, nil, err
	}
	compiled, err := snvInterfaceFor(owner)
	if err != nil || !compiled.admission[version][operation] {
		return Installation{}, nil, fmt.Errorf("SNV operation release is incompatible")
	}
	inst, err := InspectSNV(prefix, version, owner)
	if err != nil {
		return Installation{}, nil, err
	}
	read := func(name string) (map[string]any, error) {
		expected, okay := compiled.resources[name]
		if !okay {
			return nil, fmt.Errorf("SNV schema filename is not compiled")
		}
		raw, e := snvOwnedResource(inst, "share/symphony/"+inst.ModuleID+"/"+version+"/schemas/v1/"+name, expected)
		if e != nil {
			return nil, e
		}
		return snvObject(raw, maxRequestBytes)
	}
	var selected any
	if templates {
		name := "admin.templates.json"
		if owner == "sciv" {
			if operation == "sciv_validate" {
				name = "sciv-evidence.template.json"
			} else {
				name = "sciv-transition.template.json"
			}
		}
		object, e := read(name)
		if e != nil {
			return Installation{}, nil, e
		}
		if owner == "sciv" {
			selected = object
		} else {
			variants, okay := object["templates"].(map[string]any)
			if !okay || variants[operation] == nil {
				return Installation{}, nil, fmt.Errorf("SNV template operation is absent")
			}
			selected = variants[operation]
			if owner == "snv" {
				if workflows, present := object["workflows"]; present {
					selected = map[string]any{"input": selected, "workflows": workflows}
				}
			}
		}
	} else if owner == "sciv" || owner == "scnv" {
		request, result := "", ""
		switch operation {
		case "sciv_validate":
			request, result = "sciv-evidence.schema.json", "sciv-result.schema.json"
		case "sciv_transition":
			request, result = "sciv-transition.schema.json", "sciv-transition-result.schema.json"
		case "names_validate":
			request, result = "names.schema.json", "names-result.schema.json"
		case "names_resolve":
			request, result = "resolve.schema.json", "resolve-result.schema.json"
		}
		input, e := read(request)
		if e != nil {
			return Installation{}, nil, e
		}
		output, e := read(result)
		if e != nil {
			return Installation{}, nil, e
		}
		selected = map[string]any{"input": input, "result": output}
	} else {
		object, e := read("admin.schema.json")
		if e != nil {
			return Installation{}, nil, e
		}
		requests, requestOkay := object["requests"].(map[string]any)
		results, resultOkay := object["results"].(map[string]any)
		if !requestOkay || !resultOkay || requests[operation] == nil || results[operation] == nil {
			return Installation{}, nil, fmt.Errorf("SNV schema operation is absent")
		}
		value := map[string]any{"input": requests[operation], "result": results[operation]}
		if defs, present := object["$defs"]; present {
			value["$defs"] = defs
		}
		if owner == "snv" {
			if cli, present := object["cli_inputs"]; present {
				value["cli_inputs"] = cli
			}
			if cli, present := object["cli_results"]; present {
				value["cli_results"] = cli
			}
		}
		selected = value
	}
	data, err := SCVCanonical(selected)
	if err != nil {
		return Installation{}, nil, err
	}
	after, err := InspectSNV(prefix, version, owner)
	if err != nil || after != inst {
		return Installation{}, nil, fmt.Errorf("SNV installation changed during resource selection")
	}
	return inst, data, nil
}

func InvokeSNV(ctx context.Context, prefix, version, cwd, owner, operation string, payload []byte) (Response, error) {
	if operation == "descriptor" {
		return invokeSNVDescriptor(ctx, prefix, version, cwd, owner, payload)
	}
	op, err := snvOperationFor(owner, operation)
	if err != nil {
		return Response{}, err
	}
	object, err := snvObject(payload, maxRequestBytes)
	if err != nil {
		return Response{}, err
	}
	if object["protocol"] != op.InputProtocol {
		return Response{}, fmt.Errorf("SNV operation input protocol mismatch")
	}
	canonical, err := SCVCanonical(object)
	if err != nil {
		return Response{}, err
	}
	inst, contract, err := SNVResource(prefix, version, owner, operation, false)
	if err != nil {
		return Response{}, err
	}
	schema, err := snvObject(contract, maxRequestBytes)
	if err != nil || !snvTransportShape(schema["input"], object, schema, 0) {
		return Response{}, fmt.Errorf("SNV input violates exact transport schema")
	}
	response, invokeErr := invokeResolved(ctx, snvEngineSpec(owner), inst.ExecutablePath, version, cwd, operation, canonical)
	after, err := InspectSNV(prefix, version, owner)
	if err != nil || after != inst {
		return Response{}, fmt.Errorf("SNV installation changed during native invocation")
	}
	if invokeErr != nil {
		return Response{}, invokeErr
	}
	if err := ValidateSNVResult(owner, operation, canonical, response.Result, contract); err != nil {
		return Response{}, err
	}
	return response, nil
}

// ValidateSNVResult checks representation, exact schemas, protocol identities
// and correspondence. Native domain decisions stay with the C++ owner.
func ValidateSNVResult(owner, operation string, payload, result, contract []byte) error {
	op, err := snvOperationFor(owner, operation)
	if err != nil {
		return err
	}
	input, err := snvObject(payload, maxRequestBytes)
	if err != nil {
		return err
	}
	output, err := snvObject(result, maxResponseBytes)
	if err != nil {
		return err
	}
	shape, err := snvObject(contract, maxRequestBytes)
	if err != nil {
		return err
	}
	if !snvTransportShape(shape["input"], input, shape, 0) || !snvTransportShape(shape["result"], output, shape, 0) || output["protocol"] != op.OutputProtocol {
		return fmt.Errorf("SNV native result violates exact transport contract")
	}
	canonical, err := SCVCanonical(input)
	if err != nil {
		return err
	}
	sourceDigest := digestBytes(canonical)
	if owner != "snv" {
		if output["owner"] != owner || output["owner_version"] != "0.1.0-dev" || output["source_digest"] != sourceDigest {
			return fmt.Errorf("SNV owner result identity or source binding mismatch")
		}
		if subjects, okay := output["subject_ids"].([]any); !okay || !snvUniqueTexts(subjects) || !snvSuppliedSubjects(input, subjects) {
			return fmt.Errorf("SNV owner subjects are invalid")
		}
	} else {
		if value, present := output["source_digest"]; present && value != sourceDigest {
			return fmt.Errorf("SNV parent source binding mismatch")
		}
		if value, present := output["input_digest"]; present && value != sourceDigest {
			return fmt.Errorf("SNV parent plan input binding mismatch")
		}
		if ownerVersion, present := output["owner_version"]; present && ownerVersion != "0.1.0-dev" {
			return fmt.Errorf("SNV parent version mismatch")
		}
		if operationID, present := input["operation_id"]; present && output["operation_id"] != operationID {
			return fmt.Errorf("SNV operation correlation mismatch")
		}
		if operation == "snv_state_reduce" {
			plan, okay := input["plan"].(map[string]any)
			if !okay || output["head"] == nil || !reflect.DeepEqual(output["head"], plan["head"]) || output["plan_digest"] != plan["digest"] || output["operation_id"] != plan["operation_id"] || output["expected_state_digest"] != plan["expected_state_digest"] || output["effect"] != "proposed_only" {
				return fmt.Errorf("SNV transition differs from supplied plan")
			}
		}
	}
	if owner == "local-observer" {
		if output["acquisition_route"] != "native_fixed_sources" {
			return fmt.Errorf("SNV finite observation acquisition route mismatch")
		}
		if output["observation_id"] != input["observation_id"] || output["node_ref"] != input["node_ref"] || output["profile"] != input["profile"] {
			return fmt.Errorf("SNV observation request binding mismatch")
		}
		subjects := []any{}
		if input["node_ref"] != nil {
			subjects = append(subjects, input["node_ref"])
		}
		if !reflect.DeepEqual(output["subject_ids"], subjects) {
			return fmt.Errorf("SNV observation subject binding mismatch")
		}
		requested, ok := input["fields"].([]any)
		observed, observedOK := output["fields"].([]any)
		if !ok || !observedOK || len(requested) != len(observed) {
			return fmt.Errorf("SNV observation field census mismatch")
		}
		wanted := map[any]bool{}
		for _, field := range requested {
			text, ok := field.(string)
			if !ok || wanted[text] {
				return fmt.Errorf("SNV observation request field mismatch")
			}
			wanted[text] = true
		}
		for _, value := range observed {
			field, ok := value.(map[string]any)
			if !ok {
				return fmt.Errorf("SNV observation field shape mismatch")
			}
			key, ok := field["field"].(string)
			if !ok || !wanted[key] {
				return fmt.Errorf("SNV observation returned unrequested field")
			}
			delete(wanted, key)
		}
		if len(wanted) != 0 {
			return fmt.Errorf("SNV observation fields incomplete")
		}
	}
	if owner == "snv" {
		if err := snvParentCorrespondence(operation, input, output); err != nil {
			return err
		}
	}
	return snvVerifySeals(output, 0)
}
func snvUniqueTexts(values []any) bool {
	seen := map[string]bool{}
	for _, value := range values {
		text, okay := value.(string)
		if !okay || text == "" || seen[text] {
			return false
		}
		seen[text] = true
	}
	return true
}
func snvVerifySeals(value any, depth int) error {
	if depth > maxJSONDepth {
		return fmt.Errorf("SNV seal depth exceeded")
	}
	switch object := value.(type) {
	case map[string]any:
		sealField := ""
		if protocol, okay := object["protocol"].(string); okay {
			switch protocol {
			case "symphony.snv.head.v1", "symphony.snv.state-plan.v1", "symphony.snv.state-transition.v1", "symphony.snv.evidence-plan.v1", "symphony.snv.export-manifest.v1":
				sealField = "digest"
			case "scnv.resolution-binding.v1":
				sealField = "binding_digest"
			}
		}
		if sealField != "" {
			expected, present := object[sealField]
			if !present {
				return fmt.Errorf("SNV sealed object digest is missing")
			}
			body := make(map[string]any, len(object)-1)
			for k, v := range object {
				if k != sealField {
					body[k] = v
				}
			}
			raw, err := SCVCanonical(body)
			if err != nil || expected != digestBytes(raw) {
				return fmt.Errorf("SNV sealed result digest mismatch")
			}
		}
		keys := make([]string, 0, len(object))
		for key := range object {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if err := snvVerifySeals(object[key], depth+1); err != nil {
				return err
			}
		}
	case []any:
		for _, item := range object {
			if err := snvVerifySeals(item, depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}

func snvSuppliedSubjects(input any, subjects []any) bool {
	supplied := map[string]bool{}
	var collect func(any)
	collect = func(value any) {
		switch item := value.(type) {
		case string:
			supplied[item] = true
		case map[string]any:
			for _, v := range item {
				collect(v)
			}
		case []any:
			for _, v := range item {
				collect(v)
			}
		}
	}
	collect(input)
	for _, subject := range subjects {
		text, okay := subject.(string)
		if !okay || !supplied[text] {
			return false
		}
	}
	return true
}
func snvParentCorrespondence(operation string, input, output map[string]any) error {
	if operation == "snv_inspect" || operation == "snv_evidence_plan" {
		bundle, okay := input["bundle"].(map[string]any)
		if operation == "snv_evidence_plan" {
			bundle, okay = output["bundle"].(map[string]any)
		}
		if !okay {
			return fmt.Errorf("SNV composed result is missing its bundle")
		}
		encoded, err := SCVCanonical(bundle)
		if err != nil || output["bundle_digest"] != digestBytes(encoded) {
			return fmt.Errorf("SNV bundle binding mismatch")
		}
		replay := output
		if operation == "snv_evidence_plan" {
			var okay bool
			replay, okay = output["replay"].(map[string]any)
			if !okay {
				return fmt.Errorf("SNV evidence plan lacks replay")
			}
		}
		if parentOwner, present := replay["owner"]; present && parentOwner != "symphony-snv" {
			return fmt.Errorf("SNV replay owner mismatch")
		}
		if manifest, present := output["export_manifest"]; present {
			if err := snvExportCorrespondence(encoded, manifest, nil); err != nil {
				return err
			}
		}
		if chunk, present := output["export_chunk"]; present {
			if err := snvExportCorrespondence(encoded, nil, chunk); err != nil {
				return err
			}
		}
		if replay["bundle_digest"] != output["bundle_digest"] || replay["bundle_id"] != bundle["bundle_id"] {
			return fmt.Errorf("SNV replay bundle correlation mismatch")
		}
		if views, present := replay["views"]; present {
			owners, okay := views.(map[string]any)
			if !okay {
				return fmt.Errorf("SNV replay views are invalid")
			}
			artifacts, okay := bundle["artifacts"].([]any)
			if !okay {
				return fmt.Errorf("SNV supplied artifacts are invalid")
			}
			expected := map[string]map[string]any{}
			for _, value := range artifacts {
				artifact, okay := value.(map[string]any)
				if !okay {
					return fmt.Errorf("SNV artifact is invalid")
				}
				owner, okay := artifact["owner"].(string)
				if !okay {
					return fmt.Errorf("SNV artifact owner missing")
				}
				expected[owner] = artifact
			}
			if len(owners) != len(expected) {
				return fmt.Errorf("SNV replay owner census mismatch")
			}
			byOwner, okay := replay["subject_ids"].(map[string]any)
			if !okay || len(byOwner) != len(owners) {
				return fmt.Errorf("SNV replay subject owner census mismatch")
			}
			for owner, value := range owners {
				view, okay := value.(map[string]any)
				artifact, present := expected[owner]
				if !okay || !present {
					return fmt.Errorf("SNV replay unexpected owner")
				}
				source, okay := artifact["source_utf8"].(string)
				if !okay || artifact["source_digest"] != digestBytes([]byte(source)) {
					return fmt.Errorf("SNV artifact bytes binding mismatch")
				}
				original, err := snvObject([]byte(source), maxRequestBytes)
				if err != nil {
					return err
				}
				canonical, err := SCVCanonical(original)
				if err != nil || view["owner"] != owner || view["owner_version"] != artifact["owner_version"] || view["source_digest"] != digestBytes(canonical) {
					return fmt.Errorf("SNV replay owner source correspondence mismatch")
				}
				subjects, okay := view["subject_ids"].([]any)
				if !okay || !snvUniqueTexts(subjects) || !snvSuppliedSubjects(original, subjects) {
					return fmt.Errorf("SNV replay subjects differ from supplied evidence")
				}
				projected, okay := byOwner[owner].([]any)
				if !okay || !snvUniqueTexts(projected) || !snvSuppliedSubjects(original, projected) {
					return fmt.Errorf("SNV replay subject projection mismatch")
				}
				for _, subject := range subjects {
					found := false
					for _, supplied := range projected {
						found = found || subject == supplied
					}
					if !found {
						return fmt.Errorf("SNV replay projection omits an owner subject")
					}
				}
			}
		}
	}
	if operation == "snv_state_plan" {
		head, okay := output["head"].(map[string]any)
		if !okay || !reflect.DeepEqual(head["view"], input["view"]) || head["operation_id"] != input["operation_id"] || head["previous_digest"] != input["expected_state_digest"] || output["change_kind"] != input["change_kind"] || output["expected_state_digest"] != input["expected_state_digest"] {
			return fmt.Errorf("SNV plan/head correspondence mismatch")
		}
	}
	return nil
}

func snvExportCorrespondence(bundleBytes []byte, manifest, chunk any) error {
	if manifest != nil {
		m, okay := manifest.(map[string]any)
		if !okay || m["bundle_digest"] != digestBytes(bundleBytes) || m["byte_count"] != int64(len(bundleBytes)) || m["chunk_size"] != int64(16384) || m["encoding"] != "hex" {
			return fmt.Errorf("SNV export manifest byte correspondence mismatch")
		}
		records, okay := m["chunks"].([]any)
		if !okay || len(records) != (len(bundleBytes)+16383)/16384 {
			return fmt.Errorf("SNV export manifest chunk census mismatch")
		}
		for index, value := range records {
			record, okay := value.(map[string]any)
			begin := index * 16384
			end := begin + 16384
			if end > len(bundleBytes) {
				end = len(bundleBytes)
			}
			if !okay || len(record) != 3 || record["index"] != int64(index) || record["byte_count"] != int64(end-begin) || record["digest"] != digestBytes(bundleBytes[begin:end]) {
				return fmt.Errorf("SNV export manifest chunk bytes mismatch")
			}
		}
	}
	if chunk != nil {
		record, okay := chunk.(map[string]any)
		if !okay {
			return fmt.Errorf("SNV export chunk is invalid")
		}
		index, okay := record["index"].(int64)
		if !okay || index < 0 || index > int64((len(bundleBytes)+16383)/16384-1) {
			return fmt.Errorf("SNV export chunk index invalid")
		}
		begin := int(index) * 16384
		end := begin + 16384
		if end > len(bundleBytes) {
			end = len(bundleBytes)
		}
		text, okay := record["hex"].(string)
		if !okay {
			return fmt.Errorf("SNV export chunk bytes absent")
		}
		raw, err := hex.DecodeString(text)
		if err != nil || !bytes.Equal(raw, bundleBytes[begin:end]) || record["bundle_digest"] != digestBytes(bundleBytes) || record["byte_count"] != int64(len(raw)) || record["digest"] != digestBytes(raw) {
			return fmt.Errorf("SNV export chunk byte correspondence mismatch")
		}
	}
	return nil
}
