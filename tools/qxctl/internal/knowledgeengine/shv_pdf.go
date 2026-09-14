package knowledgeengine

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

const SHVPDFVersion = "0.1.0-dev"

var shvPDFSpec = engineSpec{label: "shv-pdf-adapter", moduleID: "shv-pdf-adapter", engineID: "symphony-shv-pdf", componentKind: "adapter", vectorID: "shv", processProtocol: processProtocol}
var shvPDFOutputs = map[string]string{"inspect": "symphony.knowledge.engine-descriptor.v2", "extract": "symphony.shv.pdf-extraction.v1"}

func SHVPDFResultProtocol(op string) (string, bool) {
	p, ok := shvPDFOutputs[op]
	return p, ok
}
func SHVPDFInputProtocol(op string) string {
	return "symphony.shv.pdf-" + strings.ReplaceAll(op, "_", "-") + "-input.v1"
}

func InvokeSHVPDF(ctx context.Context, prefix, version, cwd, op string, payload []byte) (Response, error) {
	if _, ok := SHVPDFResultProtocol(op); !ok {
		return Response{}, shvFail()
	}
	p, e := shvObject(payload)
	if e != nil {
		return Response{}, e
	}
	if e = validateJSONObject(payload, maxRequestBytes); e != nil {
		return Response{}, e
	}
	if op == "extract" {
		if e = validatePDFInput(p); e != nil {
			return Response{}, e
		}
	} else if len(p) != 0 {
		return Response{}, shvFail()
	}
	inst, e := InspectSHVPDF(prefix, version)
	if e != nil {
		return Response{}, e
	}
	r, e := invokeResolved(ctx, shvPDFSpec, inst.ExecutablePath, version, cwd, op, payload)
	if e != nil {
		return r, e
	}
	if e = ValidateSHVPDFResult(op, payload, r.Result); e != nil {
		return Response{}, e
	}
	after, e := InspectSHVPDF(prefix, version)
	if e != nil || after != inst {
		return Response{}, fmt.Errorf("SHV PDF installation changed during invocation")
	}
	return r, nil
}
func InspectSHVPDF(prefix, version string) (Installation, error) {
	if prefix == "" || version != SHVPDFVersion {
		return Installation{}, fmt.Errorf("SHV requires explicit prefix and exact supported version %s", SHVPDFVersion)
	}
	s := shvPDFSpec
	e, err := InspectReceiptV2EntryPoint(prefix, version, ReceiptV2EntryPointSpec{Label: s.label, ComponentID: s.moduleID, ComponentKind: s.componentKind, ModuleID: s.moduleID, PackageID: s.moduleID, VectorID: &s.vectorID, EngineID: &s.engineID, EntryPointID: s.engineID, EntryPointKind: "executable", EntryPointRelativePath: filepath.ToSlash(filepath.Join("libexec", "symphony", s.moduleID, version, s.engineID)), RequiredProtocols: []string{processProtocol}})
	if err != nil {
		return Installation{}, err
	}
	return Installation{Role: s.moduleID, ModuleID: s.moduleID, EngineID: s.engineID, Version: version, Prefix: e.Prefix, ReceiptPath: e.ReceiptPath, ReceiptDigest: e.ReceiptDigest, ReceiptProtocol: receiptProtocolV2, ExecutablePath: e.ExecutablePath, ExecutableDigest: e.ExecutableDigest}, nil
}
func SHVPDFResource(prefix, version string, templates bool) (Installation, json.RawMessage, error) {
	inst, err := InspectSHVPDF(prefix, version)
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
	name := "pdf"
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
			after, e := InspectSHVPDF(prefix, version)
			if e != nil || after != inst {
				return inst, nil, fmt.Errorf("SHV resource installation changed")
			}
			return inst, data, nil
		}
	}
	return inst, nil, fmt.Errorf("SHV receipt lacks compiled resource %s", path)
}
