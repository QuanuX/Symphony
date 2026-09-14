package knowledgeengine

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

const SHVPublicationVersion = "0.1.0-dev"
const SHVPublicationTransferVersion = "0.2.0-dev"

var shvPublicationSpec = engineSpec{label: "shv-publication-engine", moduleID: "shv-publication-engine", engineID: "symphony-shv-publication", componentKind: "vector_engine", vectorID: "shv", processProtocol: processProtocol}
var shvPublicationOutputs = map[string]string{"inspect": "symphony.knowledge.engine-descriptor.v2", "publication_plan": "symphony.shv.publication-plan.v1", "publication_reduce": "symphony.shv.publication-transition.v1", "publication_status": "symphony.shv.publication-status.v1"}

func SHVPublicationResultProtocol(op string) (string, bool) {
	v, ok := shvPublicationOutputs[op]
	return v, ok
}
func SHVPublicationInputProtocol(op string) string {
	return "symphony.shv." + strings.ReplaceAll(op, "_", "-") + "-input.v1"
}
func InspectSHVPublication(prefix, version string) (Installation, error) {
	if prefix == "" || (version != SHVPublicationVersion && version != SHVPublicationTransferVersion) {
		return Installation{}, fmt.Errorf("publication engine requires an explicit prefix and exact version %s or %s", SHVPublicationVersion, SHVPublicationTransferVersion)
	}
	s := shvPublicationSpec
	e, err := InspectReceiptV2EntryPoint(prefix, version, ReceiptV2EntryPointSpec{Label: s.label, ComponentID: s.moduleID, ComponentKind: s.componentKind, ModuleID: s.moduleID, PackageID: s.moduleID, VectorID: &s.vectorID, EngineID: &s.engineID, EntryPointID: s.engineID, EntryPointKind: "executable", EntryPointRelativePath: filepath.ToSlash(filepath.Join("libexec", "symphony", s.moduleID, version, s.engineID)), RequiredProtocols: []string{processProtocol}})
	if err != nil {
		return Installation{}, err
	}
	return Installation{Role: s.moduleID, ModuleID: s.moduleID, EngineID: s.engineID, Version: version, Prefix: e.Prefix, ReceiptPath: e.ReceiptPath, ReceiptDigest: e.ReceiptDigest, ReceiptProtocol: receiptProtocolV2, ExecutablePath: e.ExecutablePath, ExecutableDigest: e.ExecutableDigest}, nil
}
func SHVPublicationResource(prefix, version string, templates bool) (Installation, json.RawMessage, error) {
	inst, err := InspectSHVPublication(prefix, version)
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
	name := "publication"
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
			after, e := InspectSHVPublication(prefix, version)
			if e != nil || after != inst {
				return inst, nil, fmt.Errorf("SHV resource installation changed")
			}
			return inst, data, nil
		}
	}
	return inst, nil, fmt.Errorf("SHV receipt lacks compiled resource %s", path)
}

func InvokeSHVPublication(ctx context.Context, prefix, version, cwd, op string, payload []byte) (Response, error) {
	if _, ok := SHVPublicationResultProtocol(op); !ok {
		return Response{}, shvFail()
	}
	if _, e := shvObject(payload); e != nil {
		return Response{}, e
	}
	inst, e := InspectSHVPublication(prefix, version)
	if e != nil {
		return Response{}, e
	}
	r, e := invokeResolved(ctx, shvPublicationSpec, inst.ExecutablePath, version, cwd, op, payload)
	if e != nil {
		return r, e
	}
	if e = ValidateSHVPublicationResultVersion(op, payload, r.Result, version); e != nil {
		return Response{}, e
	}
	after, e := InspectSHVPublication(prefix, version)
	if e != nil || after != inst {
		return Response{}, fmt.Errorf("publication owner installation changed")
	}
	return r, nil
}
