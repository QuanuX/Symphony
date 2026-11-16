package knowledgelifecycle

import (
	"fmt"
	"strings"
)

const nativeOwnerDispatchCapability = "symphony.package-owner-dispatch.v1"

// These are existing native package owners, not an executable plugin registry.
// Retained controllers are explicitly selected by the caller and must match
// the exact reviewed receipt before any command can run.
func nativeOwnerBinary(component string) string {
	switch component {
	case "secure-identity-access-governance":
		return "symphony-ssiag"
	case "stav-append-authority":
		return "symphony-stav-append-authority"
	case "accordare-stav-producer":
		return "symphony-accordare-stav-producer"
	}
	return ""
}

func isNativeServiceOwner(component string) bool { return nativeOwnerBinary(component) != "" }

// ValidateNativeOwners performs syntax validation before a CLI acquires a state
// lease. Content, custody and receipt identity are checked at dispatch time.
func ValidateNativeOwners(bindings []string) error {
	_, err := parseNativeOwners(bindings)
	return err
}

func parseNativeOwners(bindings []string) (map[string][]string, error) {
	if len(bindings) > 64 {
		return nil, fmt.Errorf("native owner binding count exceeds its bound")
	}
	owners := make(map[string][]string)
	seen := make(map[string]bool)
	for _, binding := range bindings {
		component, path, ok := strings.Cut(binding, "=")
		if !ok || !isNativeServiceOwner(component) || !safeAbsolutePath(path) || path == "/" ||
			strings.ContainsAny(path, "\x00\r\n") || seen[path] {
			return nil, fmt.Errorf("native owner must be a known component and unique absolute retained executable path")
		}
		seen[path] = true
		owners[component] = append(owners[component], path)
	}
	return owners, nil
}

func (e *Executor) SetNativeOwners(bindings []string) error {
	owners, err := parseNativeOwners(bindings)
	if err == nil {
		e.nativeOwners = owners
	}
	return err
}

func nativeOwnerArguments(operation, component, root, version string) []string {
	args := []string{operation, "--prefix", root, "--version", version}
	if component == "secure-identity-access-governance" {
		args = append([]string{"package"}, args...)
	}
	if component == "stav-append-authority" {
		args = append(args, "--scope", "user")
	}
	return append(args, "--lifecycle-root-fd", "3", "--lifecycle-lock-fd", "4", "--lifecycle-proof-fd", "5")
}

type nativeDispatchProof struct {
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
