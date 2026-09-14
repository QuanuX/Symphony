package knowledgeengine

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

const SHVVersion = "0.1.0-dev"
const SHVTableVersion = "0.2.0-dev"
const SHVDocumentVersion = "0.3.0-dev"

func shvKernelVersion(version string) bool {
	return version == SHVVersion || version == SHVTableVersion || version == SHVDocumentVersion
}

var shvEngineSpec = engineSpec{label: "SHV", moduleID: "shv-engine", engineID: "symphony-shv", componentKind: "vector_engine", vectorID: "shv", processProtocol: processProtocol}
var shvAdapterSpec = engineSpec{label: "shv-graph-adapter", moduleID: "shv-graph-adapter", engineID: "symphony-shv-graph-adapter", componentKind: "adapter", vectorID: "shv", processProtocol: processProtocol}
var shvOutputs = map[string]string{"inspect": "symphony.knowledge.engine-descriptor.v2", "coverage_default": "symphony.shv.coverage-profile.v1", "coverage_plan": "symphony.shv.coverage-result.v1", "catalogue_build": "symphony.shv.catalogue.v1", "catalogue_query": "symphony.shv.query-result.v1", "evaluate": "symphony.shv.evaluation.v1", "graph_project": "symphony.graph.exchange.v1", "graph_validate": "symphony.shv.graph-validation.v1"}

func SHVResultProtocol(op string, adapter bool) (string, bool) {
	if adapter {
		if op == "inspect" {
			return shvOutputs[op], true
		}
		if op == "roundtrip" || op == "query" {
			return "symphony.graph.adapter-result.v1", true
		}
		return "", false
	}
	p, ok := shvOutputs[op]
	return p, ok
}
func SHVInputProtocol(op string, adapter bool) string {
	prefix := "symphony.shv."
	if adapter {
		prefix = "symphony.graph.adapter-"
	}
	return prefix + strings.ReplaceAll(op, "_", "-") + "-input.v1"
}
func inspectSHV(prefix, version string, adapter bool) (Installation, error) {
	if prefix == "" || (adapter && version != SHVVersion) || (!adapter && !shvKernelVersion(version)) {
		return Installation{}, fmt.Errorf("SHV requires an explicit prefix and supported exact version (kernel %s, %s or %s; adapter %s)", SHVVersion, SHVTableVersion, SHVDocumentVersion, SHVVersion)
	}
	s := shvEngineSpec
	if adapter {
		s = shvAdapterSpec
	}
	e, err := InspectReceiptV2EntryPoint(prefix, version, ReceiptV2EntryPointSpec{Label: s.label, ComponentID: s.moduleID, ComponentKind: s.componentKind, ModuleID: s.moduleID, PackageID: s.moduleID, VectorID: &s.vectorID, EngineID: &s.engineID, EntryPointID: s.engineID, EntryPointKind: "executable", EntryPointRelativePath: filepath.ToSlash(filepath.Join("libexec", "symphony", s.moduleID, version, s.engineID)), RequiredProtocols: []string{processProtocol}})
	if err != nil {
		return Installation{}, err
	}
	return Installation{Role: s.moduleID, ModuleID: s.moduleID, EngineID: s.engineID, Version: version, Prefix: e.Prefix, ReceiptPath: e.ReceiptPath, ReceiptDigest: e.ReceiptDigest, ReceiptProtocol: receiptProtocolV2, ExecutablePath: e.ExecutablePath, ExecutableDigest: e.ExecutableDigest}, nil
}
func InspectSHV(prefix, version string) (Installation, error) {
	return inspectSHV(prefix, version, false)
}
func InspectSHVGraphAdapter(prefix, version string) (Installation, error) {
	return inspectSHV(prefix, version, true)
}
func InvokeSHV(ctx context.Context, prefix, version, cwd, op string, payload []byte) (Response, error) {
	return invokeSHV(ctx, prefix, version, cwd, op, payload, false)
}
func InvokeSHVGraphAdapter(ctx context.Context, prefix, version, cwd, op string, payload []byte) (Response, error) {
	return invokeSHV(ctx, prefix, version, cwd, op, payload, true)
}
func invokeSHV(ctx context.Context, prefix, version, cwd, op string, payload []byte, adapter bool) (Response, error) {
	if _, ok := SHVResultProtocol(op, adapter); !ok {
		return Response{}, fmt.Errorf("unsupported SHV operation")
	}
	if err := ValidateSCVBundleText(payload); err != nil {
		return Response{}, err
	}
	if err := validateJSONObject(payload, maxRequestBytes); err != nil {
		return Response{}, err
	}
	inst, err := inspectSHV(prefix, version, adapter)
	if err != nil {
		return Response{}, err
	}
	s := shvEngineSpec
	if adapter {
		s = shvAdapterSpec
	}
	r, err := invokeResolved(ctx, s, inst.ExecutablePath, version, cwd, op, payload)
	if err != nil {
		return r, err
	}
	if err = ValidateSHVResultVersion(op, payload, r.Result, adapter, version); err != nil {
		return Response{}, err
	}
	after, err := inspectSHV(prefix, version, adapter)
	if err != nil || after != inst {
		return Response{}, fmt.Errorf("SHV installation changed during invocation")
	}
	return r, nil
}

// SHVResource reads only compiled, receipt-owned schema/template resources.
// It does not discover protocol definitions from a source checkout.
func SHVResource(prefix, version string, adapter, templates bool) (Installation, json.RawMessage, error) {
	inst, err := inspectSHV(prefix, version, adapter)
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
	name := "shv"
	if adapter {
		name = "graph-adapter"
	}
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
			after, e := inspectSHV(prefix, version, adapter)
			if e != nil || after != inst {
				return inst, nil, fmt.Errorf("SHV resource installation changed")
			}
			return inst, data, nil
		}
	}
	return inst, nil, fmt.Errorf("SHV receipt lacks compiled resource %s", path)
}
