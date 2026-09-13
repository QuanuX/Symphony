package knowledgeengine

import (
	"encoding/json"
	"fmt"
)

// SCVGraphIndexSchema reads the exact receipt-owned connector schema. It does
// not substitute a newer release or require a source checkout.
func SCVGraphIndexSchema(prefix, version string) (Installation, json.RawMessage, error) {
	inst, err := InspectSCVGraphIndexConnector("duckdb", prefix, version)
	if err != nil {
		return inst, nil, err
	}
	receiptPath := "share/symphony/receipts/" + inst.ModuleID + "/" + version + "/install-receipt.json"
	raw, err := readTrustedNoFollowRelative(inst.Prefix, receiptPath, maxReceiptBytes)
	if err != nil {
		return inst, nil, err
	}
	var receipt receiptV2
	if err = decodeExact(raw, &receipt); err != nil || receipt.ReceiptDigest != inst.ReceiptDigest {
		return inst, nil, fmt.Errorf("schema receipt changed")
	}
	path := "share/symphony/schemas/" + inst.ModuleID + "/" + version + "/graph-index.schema.json"
	for _, f := range receipt.Files {
		if f.Path == path && f.Kind == "regular" {
			data, err := readTrustedNoFollowRelative(inst.Prefix, path, maxRequestBytes)
			if err != nil {
				return inst, nil, err
			}
			if uint64(len(data)) != f.Size || digestBytes(data) != f.Digest {
				return inst, nil, fmt.Errorf("owned graph schema changed")
			}
			if _, err = scvObject(data); err != nil {
				return inst, nil, err
			}
			return inst, json.RawMessage(data), nil
		}
	}
	return inst, nil, fmt.Errorf("selected receipt lacks graph index schema")
}
