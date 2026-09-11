package knowledgeengine

import (
	"fmt"
	"path/filepath"
)

// SCVOwnerInterface inspects the selected receipt-owned declaration without a
// checkout or a native invocation. Its meaning is the compiled interface
// inventory; it does not admit new handlers or execute declaration content.
func SCVOwnerInterface(domain, prefix, version string) (map[string]any, error) {
	if !SCVSupports(version, "interface") {
		return nil, fmt.Errorf("selected exact SCV release does not expose an owner interface resource")
	}
	installed, err := InspectSCVDomain(domain, prefix, version)
	if err != nil {
		return nil, err
	}
	relativeReceipt := filepath.ToSlash(filepath.Join("share", "symphony", "receipts", installed.ModuleID, version, "install-receipt.json"))
	raw, err := readTrustedNoFollowRelative(installed.Prefix, relativeReceipt, maxReceiptBytes)
	if err != nil {
		return nil, err
	}
	var receipt receiptV2
	if err := decodeExact(raw, &receipt); err != nil || receipt.ReceiptDigest != installed.ReceiptDigest {
		return nil, fmt.Errorf("selected owner interface receipt changed")
	}
	relative := filepath.ToSlash(filepath.Join("share", "symphony", "contracts", installed.ModuleID, version, "OWNER-INTERFACE.json"))
	var owned *receiptV2File
	for i := range receipt.Files {
		if receipt.Files[i].Path == relative {
			owned = &receipt.Files[i]
		}
	}
	if owned == nil || owned.Kind != "regular" {
		return nil, fmt.Errorf("owner interface is not a receipt-owned regular resource")
	}
	data, err := readTrustedNoFollowRelative(installed.Prefix, relative, maxRequestBytes)
	if err != nil {
		return nil, err
	}
	if uint64(len(data)) != owned.Size || digestBytes(data) != owned.Digest {
		return nil, fmt.Errorf("owner interface resource differs from its receipt")
	}
	definition, err := scvObject(data)
	if err != nil {
		return nil, err
	}
	digest, err := scvValidateOwnerInterface(version, definition)
	if err != nil {
		return nil, err
	}
	after, err := InspectSCVDomain(domain, prefix, version)
	if err != nil || after != installed {
		return nil, fmt.Errorf("owner interface installation changed during inspection")
	}
	return map[string]any{
		"protocol": "symphony.qxctl.scv-owner-interface.v1", "domain": domain, "engine_version": version,
		"installation": installed, "interface": definition, "interface_digest": owned.Digest,
		"definition_digest": digest, "receipt_validation": "exact_owned_file_verified", "scope": "interface_metadata_only",
	}, nil
}

func scvValidateOwnerInterface(version string, definition map[string]any) (string, error) {
	if !SCVSupports(version, "interface") || !scvCorpusFields(definition, "protocol", "current_release", "domains", "releases", "operations", "schema_catalog") ||
		definition["protocol"] != "symphony.scv.owner-interface.v1" || definition["current_release"] != version {
		return "", fmt.Errorf("owner interface identity does not match selected exact release")
	}
	actual, err := SCVDigest(definition)
	if err != nil || actual != scvInterfaceDefinitionDigests[version] {
		return "", fmt.Errorf("owner interface definition differs from compiled interface admission")
	}
	return actual, nil
}
