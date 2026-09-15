package knowledgeengine

import "fmt"

// Installed metadata is evidence of exact compiled admission, never a dynamic loader.
func verifySHVProfileInterface(inst Installation) error {
	receiptPath := "share/symphony/receipts/" + inst.ModuleID + "/" + inst.Version + "/install-receipt.json"
	raw, err := readTrustedNoFollowRelative(inst.Prefix, receiptPath, maxReceiptBytes)
	if err != nil {
		return err
	}
	var receipt receiptV2
	if decodeExact(raw, &receipt) != nil || receipt.ReceiptDigest != inst.ReceiptDigest {
		return fmt.Errorf("profile interface receipt changed")
	}
	path := "share/symphony/contracts/" + inst.ModuleID + "/" + inst.Version + "/OWNER-INTERFACE.json"
	for _, file := range receipt.Files {
		if file.Path != path || file.Kind != "regular" {
			continue
		}
		data, err := readTrustedNoFollowRelative(inst.Prefix, path, maxRequestBytes)
		if err != nil {
			return err
		}
		if uint64(len(data)) != file.Size || digestBytes(data) != file.Digest {
			return fmt.Errorf("profile interface owned bytes changed")
		}
		value, err := shvObject(data)
		if err != nil {
			return err
		}
		canonical, err := SCVCanonical(value)
		if err != nil {
			return err
		}
		if digestBytes(canonical) != shvProfileInterfaceDigest {
			return fmt.Errorf("profile interface differs from compiled admission")
		}
		after, err := readTrustedNoFollowRelative(inst.Prefix, receiptPath, maxReceiptBytes)
		if err != nil || string(after) != string(raw) {
			return fmt.Errorf("profile interface receipt changed during inspection")
		}
		return nil
	}
	return fmt.Errorf("profile interface is not receipt owned")
}
