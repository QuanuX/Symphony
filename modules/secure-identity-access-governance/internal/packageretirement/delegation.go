package packageretirement

// DispatchCapability names this bounded native-owner handoff, not a generic
// executable plugin or permission to bypass native retirement checks.
const DispatchCapability = "symphony.package-owner-dispatch.v1"

type DispatchProof struct {
	Protocol                string `json:"protocol"`
	FormatVersion           uint64 `json:"format_version"`
	Operation               string `json:"operation"`
	ComponentID             string `json:"component_id"`
	Version                 string `json:"version"`
	InstallRoot             string `json:"install_root"`
	ReceiptDigest           string `json:"receipt_digest"`
	BinaryDigest            string `json:"binary_digest"`
	ControlDomainID         string `json:"control_domain_id"`
	TOPSID                  string `json:"tops_id"`
	ProfileID               string `json:"profile_id"`
	OwnershipRegistryDigest string `json:"ownership_registry_digest"`
	OwnershipGeneration     uint64 `json:"ownership_generation"`
	RootDevice              uint64 `json:"root_device"`
	RootInode               uint64 `json:"root_inode"`
	LockDevice              uint64 `json:"lock_device"`
	LockInode               uint64 `json:"lock_inode"`
	Nonce                   string `json:"nonce"`
}
