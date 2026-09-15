package knowledgeengine

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

const SHVProfileVersion = "0.1.0-dev"
const SHVProfileDiagnosticsVersion = "0.2.0-dev"

var shvProfileSpec = engineSpec{label: "shv-profile-engine", moduleID: "shv-profile-engine", engineID: "symphony-shv-profile", componentKind: "vector_engine", vectorID: "shv", processProtocol: processProtocol}

func SHVProfileResultProtocol(op string) (string, bool) {
	v, ok := shvProfileOutputs[op]
	return v, ok
}
func SHVProfileInputProtocol(op string) string {
	return "symphony.shv." + strings.ReplaceAll(op, "_", "-") + "-input.v1"
}
func InspectSHVProfile(prefix, version string) (Installation, error) {
	if prefix == "" || shvProfileAdmission[version] == nil {
		return Installation{}, fmt.Errorf("profile engine requires an explicit prefix and a supported exact version")
	}
	s := shvProfileSpec
	e, err := InspectReceiptV2EntryPoint(prefix, version, ReceiptV2EntryPointSpec{Label: s.label, ComponentID: s.moduleID, ComponentKind: s.componentKind, ModuleID: s.moduleID, PackageID: s.moduleID, VectorID: &s.vectorID, EngineID: &s.engineID, EntryPointID: s.engineID, EntryPointKind: "executable", EntryPointRelativePath: filepath.ToSlash(filepath.Join("libexec", "symphony", s.moduleID, version, s.engineID)), RequiredProtocols: []string{processProtocol}})
	if err != nil {
		return Installation{}, err
	}
	inst := Installation{Role: s.moduleID, ModuleID: s.moduleID, EngineID: s.engineID, Version: version, Prefix: e.Prefix, ReceiptPath: e.ReceiptPath, ReceiptDigest: e.ReceiptDigest, ReceiptProtocol: receiptProtocolV2, ExecutablePath: e.ExecutablePath, ExecutableDigest: e.ExecutableDigest}
	if version == SHVProfileInterfaceVersion {
		if err := verifySHVProfileInterface(inst); err != nil {
			return Installation{}, err
		}
	}
	return inst, nil
}
func SHVProfileResource(prefix, version string, templates bool) (Installation, json.RawMessage, error) {
	inst, err := InspectSHVProfile(prefix, version)
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
	name := "profile"
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
			after, e := InspectSHVProfile(prefix, version)
			if e != nil || after != inst {
				return inst, nil, fmt.Errorf("SHV resource installation changed")
			}
			return inst, data, nil
		}
	}
	return inst, nil, fmt.Errorf("SHV receipt lacks compiled resource %s", path)
}

func InvokeSHVProfile(ctx context.Context, prefix, version, cwd, op string, payload []byte) (Response, error) {
	if !shvProfileAdmission[version][op] {
		return Response{}, shvFail()
	}
	if _, e := shvObject(payload); e != nil {
		return Response{}, e
	}
	inst, e := InspectSHVProfile(prefix, version)
	if e != nil {
		return Response{}, e
	}
	r, e := invokeResolved(ctx, shvProfileSpec, inst.ExecutablePath, version, cwd, op, payload)
	if e != nil {
		return r, e
	}
	if e = ValidateSHVProfileResultVersion(op, payload, r.Result, version); e != nil {
		return Response{}, e
	}
	after, e := InspectSHVProfile(prefix, version)
	if e != nil || after != inst {
		return Response{}, fmt.Errorf("profile owner installation changed")
	}
	return r, nil
}
