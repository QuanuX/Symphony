package knowledgeengine

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
)

const SHVStoreVersion = "0.1.0-dev"
const SHVStoreInventoryVersion = "0.2.0-dev"
const SHVStoreTransferVersion = "0.3.0-dev"

var shvStoreSpec = engineSpec{label: "shv-graph-duckdb-connector", moduleID: "shv-graph-duckdb-connector", engineID: "symphony-shv-graph-duckdb-connector", componentKind: "adapter", vectorID: "shv", processProtocol: processProtocol}
var shvStoreOutputs = shvStoreInterfaceOutputs

func SHVStoreResultProtocol(op string) (string, bool) { v, ok := shvStoreOutputs[op]; return v, ok }
func SHVStoreInputProtocol(op string) string          { return "symphony.shv.graph-store-" + op + "-input.v1" }
func InspectSHVStore(prefix, version string) (Installation, error) {
	if prefix == "" || shvStoreInterfaceAdmission[version] == nil {
		return Installation{}, fmt.Errorf("SHV requires an explicit prefix and a supported exact owner version")
	}
	s := shvStoreSpec
	e, err := InspectReceiptV2EntryPoint(prefix, version, ReceiptV2EntryPointSpec{Label: s.label, ComponentID: s.moduleID, ComponentKind: s.componentKind, ModuleID: s.moduleID, PackageID: s.moduleID, VectorID: &s.vectorID, EngineID: &s.engineID, EntryPointID: s.engineID, EntryPointKind: "executable", EntryPointRelativePath: filepath.ToSlash(filepath.Join("libexec", "symphony", s.moduleID, version, s.engineID)), RequiredProtocols: []string{processProtocol}})
	if err != nil {
		return Installation{}, err
	}
	inst := Installation{Role: s.moduleID, ModuleID: s.moduleID, EngineID: s.engineID, Version: version, Prefix: e.Prefix, ReceiptPath: e.ReceiptPath, ReceiptDigest: e.ReceiptDigest, ReceiptProtocol: receiptProtocolV2, ExecutablePath: e.ExecutablePath, ExecutableDigest: e.ExecutableDigest}
	if version == SHVStoreInterfaceVersion {
		if err := verifySHVOwnerInterface(inst, shvStoreInterfaceDigest); err != nil {
			return Installation{}, err
		}
	}
	return inst, nil
}
func SHVStoreResource(prefix, version string, templates bool) (Installation, json.RawMessage, error) {
	inst, err := InspectSHVStore(prefix, version)
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
	name := "graph-store"
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
			after, e := InspectSHVStore(prefix, version)
			if e != nil || after != inst {
				return inst, nil, fmt.Errorf("SHV resource installation changed")
			}
			return inst, data, nil
		}
	}
	return inst, nil, fmt.Errorf("SHV receipt lacks compiled resource %s", path)
}

func InvokeSHVStore(ctx context.Context, prefix, version, cwd, op string, payload []byte) (Response, error) {
	if _, ok := SHVStoreResultProtocol(op); !ok {
		return Response{}, fmt.Errorf("unsupported graph store operation")
	}
	p, e := shvObject(payload)
	if e != nil {
		return Response{}, e
	}
	if e = shvStoreInputVersion(op, p, version); e != nil {
		return Response{}, e
	}
	inst, e := InspectSHVStore(prefix, version)
	if e != nil {
		return Response{}, e
	}
	if op == "transfer_plan" {
		raw, _ := json.Marshal(inst)
		selected, _ := shvObject(raw)
		if !scvEqual(p["source_connector"], selected) {
			return Response{}, shvFail()
		}
	}
	if op == "prepare" {
		raw, _ := json.Marshal(inst)
		selected, _ := shvObject(raw)
		if !scvEqual(p["connector"], selected) {
			return Response{}, shvFail()
		}
	}
	response, e := invokeResolved(ctx, shvStoreSpec, inst.ExecutablePath, version, cwd, op, payload)
	if e != nil {
		return response, e
	}
	if e = ValidateSHVStoreResultVersion(op, payload, response.Result, version); e != nil {
		return Response{}, e
	}
	if op != "inspect" && op != "inventory" && op != "transfer_plan" {
		r, _ := shvObject(response.Result)
		snap := shvMap(r["snapshot"])
		if snap == nil {
			snap = shvMap(shvMap(r["intent"])["snapshot"])
		}
		raw, _ := json.Marshal(inst)
		selected, _ := shvObject(raw)
		if !scvEqual(snap["connector"], selected) {
			return Response{}, shvFail()
		}
	}
	after, e := InspectSHVStore(prefix, version)
	if e != nil || after != inst {
		return Response{}, fmt.Errorf("graph store installation changed")
	}
	return response, nil
}
