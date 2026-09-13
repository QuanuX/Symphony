package knowledgeengine

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

const SHVSourceVersion = "0.1.0-dev"

var shvSourceSpec = engineSpec{label: "shv-source-engine", moduleID: "shv-source-engine", engineID: "symphony-shv-source", componentKind: "vector_engine", vectorID: "shv", processProtocol: processProtocol}
var shvSourceOutputs = map[string]string{"inspect": "symphony.knowledge.engine-descriptor.v2", "source_plan": "symphony.shv.source-plan.v1", "source_reduce": "symphony.shv.source-transition.v1", "source_status": "symphony.shv.source-status.v1", "capture_import": "symphony.shv.source-capture.v1", "capture_compare": "symphony.shv.capture-comparison.v1", "graph_project": "symphony.graph.exchange.v1", "graph_validate": "symphony.shv.source-graph-validation.v1"}

func SHVSourceResultProtocol(op string) (string, bool) { p, ok := shvSourceOutputs[op]; return p, ok }
func SHVSourceInputProtocol(op string) string {
	if op == "inspect" || op == "graph_project" || op == "graph_validate" {
		op = "source_" + op
	}
	return "symphony.shv." + strings.ReplaceAll(op, "_", "-") + "-input.v1"
}
func InvokeSHVSource(ctx context.Context, prefix, version, cwd, op string, payload []byte) (Response, error) {
	if _, ok := SHVSourceResultProtocol(op); !ok {
		return Response{}, shvFail()
	}
	p, e := shvObject(payload)
	if e != nil {
		return Response{}, e
	}
	if e = validateJSONObject(payload, maxRequestBytes); e != nil {
		return Response{}, e
	}
	// Validate and rederive before launch as well as after, including actual retained bytes.
	if op != "inspect" {
		if _, e = shvLifecycleExpected(op, p); e != nil {
			return Response{}, e
		}
	} else if len(p) != 0 {
		return Response{}, shvFail()
	}
	inst, e := InspectSHVSource(prefix, version)
	if e != nil {
		return Response{}, e
	}
	r, e := invokeResolved(ctx, shvSourceSpec, inst.ExecutablePath, version, cwd, op, payload)
	if e != nil {
		return r, e
	}
	if e = ValidateSHVSourceResult(op, payload, r.Result); e != nil {
		return Response{}, e
	}
	after, e := InspectSHVSource(prefix, version)
	if e != nil || after != inst {
		return Response{}, fmt.Errorf("SHV source installation changed during invocation")
	}
	return r, nil
}
func InspectSHVSource(prefix, version string) (Installation, error) {
	if prefix == "" || version != SHVSourceVersion {
		return Installation{}, fmt.Errorf("SHV requires explicit prefix and exact supported version %s", SHVSourceVersion)
	}
	s := shvSourceSpec
	e, err := InspectReceiptV2EntryPoint(prefix, version, ReceiptV2EntryPointSpec{Label: s.label, ComponentID: s.moduleID, ComponentKind: s.componentKind, ModuleID: s.moduleID, PackageID: s.moduleID, VectorID: &s.vectorID, EngineID: &s.engineID, EntryPointID: s.engineID, EntryPointKind: "executable", EntryPointRelativePath: filepath.ToSlash(filepath.Join("libexec", "symphony", s.moduleID, version, s.engineID)), RequiredProtocols: []string{processProtocol}})
	if err != nil {
		return Installation{}, err
	}
	return Installation{Role: s.moduleID, ModuleID: s.moduleID, EngineID: s.engineID, Version: version, Prefix: e.Prefix, ReceiptPath: e.ReceiptPath, ReceiptDigest: e.ReceiptDigest, ReceiptProtocol: receiptProtocolV2, ExecutablePath: e.ExecutablePath, ExecutableDigest: e.ExecutableDigest}, nil
}
func SHVSourceResource(prefix, version string, templates bool) (Installation, json.RawMessage, error) {
	inst, err := InspectSHVSource(prefix, version)
	if err != nil {
		return inst, nil, err
	}
	raw, err := readTrustedNoFollowRelative(inst.Prefix, "share/symphony/receipts/"+inst.ModuleID+"/"+version+"/install-receipt.json", maxReceiptBytes)
	if err != nil {
		return inst, nil, err
	}
	var receipt receiptV2
	if decodeExact(raw, &receipt) != nil || receipt.ReceiptDigest != inst.ReceiptDigest {
		return inst, nil, fmt.Errorf("SHV resource receipt changed")
	}
	name := "source"
	if templates {
		name += ".templates.json"
	} else {
		name += ".schema.json"
	}
	path := "share/symphony/schemas/" + inst.ModuleID + "/" + version + "/" + name
	for _, f := range receipt.Files {
		if f.Path == path && f.Kind == "regular" {
			data, e := readTrustedNoFollowRelative(inst.Prefix, path, maxRequestBytes)
			if e != nil {
				return inst, nil, e
			}
			if uint64(len(data)) != f.Size || digestBytes(data) != f.Digest {
				return inst, nil, fmt.Errorf("SHV receipt-owned resource changed")
			}
			if _, e = shvObject(data); e != nil {
				return inst, nil, e
			}
			after, e := InspectSHVSource(prefix, version)
			if e != nil || after != inst {
				return inst, nil, fmt.Errorf("SHV resource installation changed")
			}
			return inst, data, nil
		}
	}
	return inst, nil, fmt.Errorf("SHV receipt lacks compiled resource %s", path)
}
