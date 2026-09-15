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
	return shvKernelInterfaceAdmission[version] != nil
}

var shvEngineSpec = engineSpec{label: "SHV", moduleID: "shv-engine", engineID: "symphony-shv", componentKind: "vector_engine", vectorID: "shv", processProtocol: processProtocol}
var shvAdapterSpec = engineSpec{label: "shv-graph-adapter", moduleID: "shv-graph-adapter", engineID: "symphony-shv-graph-adapter", componentKind: "adapter", vectorID: "shv", processProtocol: processProtocol}
var shvOutputs = shvKernelInterfaceOutputs

func SHVResultProtocol(op string, adapter bool) (string, bool) {
	if adapter {
		p, ok := shvGraphAdapterInterfaceOutputs[op]
		return p, ok
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
	if prefix == "" || (adapter && shvGraphAdapterInterfaceAdmission[version] == nil) || (!adapter && !shvKernelVersion(version)) {
		return Installation{}, fmt.Errorf("SHV requires an explicit prefix and a supported exact owner version")
	}
	s := shvEngineSpec
	if adapter {
		s = shvAdapterSpec
	}
	e, err := InspectReceiptV2EntryPoint(prefix, version, ReceiptV2EntryPointSpec{Label: s.label, ComponentID: s.moduleID, ComponentKind: s.componentKind, ModuleID: s.moduleID, PackageID: s.moduleID, VectorID: &s.vectorID, EngineID: &s.engineID, EntryPointID: s.engineID, EntryPointKind: "executable", EntryPointRelativePath: filepath.ToSlash(filepath.Join("libexec", "symphony", s.moduleID, version, s.engineID)), RequiredProtocols: []string{processProtocol}})
	if err != nil {
		return Installation{}, err
	}
	inst := Installation{Role: s.moduleID, ModuleID: s.moduleID, EngineID: s.engineID, Version: version, Prefix: e.Prefix, ReceiptPath: e.ReceiptPath, ReceiptDigest: e.ReceiptDigest, ReceiptProtocol: receiptProtocolV2, ExecutablePath: e.ExecutablePath, ExecutableDigest: e.ExecutableDigest}
	if !adapter && version == SHVKernelInterfaceVersion {
		if err := verifySHVOwnerInterface(inst, shvKernelInterfaceDigest); err != nil {
			return Installation{}, err
		}
	}
	if adapter && version == SHVGraphAdapterInterfaceVersion {
		if err := verifySHVOwnerInterface(inst, shvGraphAdapterInterfaceDigest); err != nil {
			return Installation{}, err
		}
	}
	return inst, nil
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
