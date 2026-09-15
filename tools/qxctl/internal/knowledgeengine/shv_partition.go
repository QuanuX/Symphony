package knowledgeengine

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

const SHVPartitionVersion = "0.1.0-dev"
const SHVDocumentPartitionVersion = "0.2.0-dev"

var shvPartitionSpec = engineSpec{label: "shv-partition-engine", moduleID: "shv-partition-engine", engineID: "symphony-shv-partition", componentKind: "vector_engine", vectorID: "shv", processProtocol: processProtocol}
var shvPartitionOutputs = shvPartitionInterfaceOutputs

func SHVPartitionResultProtocol(op string) (string, bool) {
	p, ok := shvPartitionOutputs[op]
	return p, ok
}
func SHVPartitionInputProtocol(op string) string {
	if op == "partition_build" {
		op = "build"
	}
	return "symphony.shv.partition-" + strings.ReplaceAll(op, "_", "-") + "-input.v1"
}

func InvokeSHVPartition(ctx context.Context, prefix, version, cwd, op string, payload []byte) (Response, error) {
	if _, ok := SHVPartitionResultProtocol(op); !ok {
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
		if _, e = shvPartitionExpected(op, p); e != nil {
			return Response{}, e
		}
	} else if len(p) != 0 {
		return Response{}, shvFail()
	}
	inst, e := InspectSHVPartition(prefix, version)
	if e != nil {
		return Response{}, e
	}
	r, e := invokeResolved(ctx, shvPartitionSpec, inst.ExecutablePath, version, cwd, op, payload)
	if e != nil {
		return r, e
	}
	if e = ValidateSHVPartitionResultVersion(op, payload, r.Result, version); e != nil {
		return Response{}, e
	}
	after, e := InspectSHVPartition(prefix, version)
	if e != nil || after != inst {
		return Response{}, fmt.Errorf("SHV partition installation changed during invocation")
	}
	return r, nil
}
func InspectSHVPartition(prefix, version string) (Installation, error) {
	if prefix == "" || shvPartitionInterfaceAdmission[version] == nil {
		return Installation{}, fmt.Errorf("SHV requires an explicit prefix and a supported exact owner version")
	}
	s := shvPartitionSpec
	e, err := InspectReceiptV2EntryPoint(prefix, version, ReceiptV2EntryPointSpec{Label: s.label, ComponentID: s.moduleID, ComponentKind: s.componentKind, ModuleID: s.moduleID, PackageID: s.moduleID, VectorID: &s.vectorID, EngineID: &s.engineID, EntryPointID: s.engineID, EntryPointKind: "executable", EntryPointRelativePath: filepath.ToSlash(filepath.Join("libexec", "symphony", s.moduleID, version, s.engineID)), RequiredProtocols: []string{processProtocol}})
	if err != nil {
		return Installation{}, err
	}
	inst := Installation{Role: s.moduleID, ModuleID: s.moduleID, EngineID: s.engineID, Version: version, Prefix: e.Prefix, ReceiptPath: e.ReceiptPath, ReceiptDigest: e.ReceiptDigest, ReceiptProtocol: receiptProtocolV2, ExecutablePath: e.ExecutablePath, ExecutableDigest: e.ExecutableDigest}
	if version == SHVPartitionInterfaceVersion {
		if err := verifySHVOwnerInterface(inst, shvPartitionInterfaceDigest); err != nil {
			return Installation{}, err
		}
	}
	return inst, nil
}
func SHVPartitionResource(prefix, version string, templates bool) (Installation, json.RawMessage, error) {
	inst, err := InspectSHVPartition(prefix, version)
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
	name := "partition"
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
			after, e := InspectSHVPartition(prefix, version)
			if e != nil || after != inst {
				return inst, nil, fmt.Errorf("SHV resource installation changed")
			}
			return inst, data, nil
		}
	}
	return inst, nil, fmt.Errorf("SHV receipt lacks compiled resource %s", path)
}
