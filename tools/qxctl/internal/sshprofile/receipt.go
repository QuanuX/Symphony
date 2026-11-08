//go:build darwin || linux

package sshprofile

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	c "github.com/QuanuX/Symphony/tools/qxctl/internal/sshprofilewire"
	"strconv"
	"strings"
)

// Independent reader of the existing knowledge install-receipt.v2 contract.
// Its intrinsic digest has no domain prefix; transport manifest digests do.
type receiptV2File struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Size   uint64 `json:"size"`
	Digest string `json:"digest"`
}

type receiptV2EntryPoint struct {
	EntryPointID string   `json:"entry_point_id"`
	Kind         string   `json:"kind"`
	Path         string   `json:"path"`
	Protocols    []string `json:"protocols"`
}

type receiptV2Platform struct {
	OS           string  `json:"os"`
	Architecture string  `json:"architecture"`
	KernelABI    *string `json:"kernel_abi"`
	Critical     bool    `json:"critical"`
}

type receiptV2 struct {
	Protocol             string                `json:"protocol"`
	FormatVersion        uint64                `json:"format_version"`
	ComponentID          string                `json:"component_id"`
	ComponentKind        string                `json:"component_kind"`
	ModuleID             string                `json:"module_id"`
	VectorID             *string               `json:"vector_id"`
	EngineID             *string               `json:"engine_id"`
	PackageID            string                `json:"package_id"`
	Version              string                `json:"version"`
	InstallScope         string                `json:"install_scope"`
	PrefixMode           string                `json:"prefix_mode"`
	Files                []receiptV2File       `json:"files"`
	EntryPoints          []receiptV2EntryPoint `json:"entry_points"`
	ProvidesCapabilities []string              `json:"provides_capabilities"`
	RequiresCapabilities []string              `json:"requires_capabilities"`
	CompatibleReceptors  []string              `json:"compatible_receptors"`
	PlatformRequirements []receiptV2Platform   `json:"platform_requirements"`
	ReceiptDigest        string                `json:"receipt_digest"`
}

func verifyReceipt(raw []byte, m c.PackageManifest) error {
	var r receiptV2
	if c.Decode(raw, &r) != nil || r.Protocol != "symphony.knowledge.install-receipt.v2" || r.FormatVersion != 2 ||
		r.ComponentID != m.ReceiptComponentID || r.ModuleID != r.ComponentID || r.PackageID != r.ComponentID || r.ComponentID == "" ||
		r.Version != m.ReceiptVersion || r.Version == "" || r.InstallScope != "prefix" || r.PrefixMode != "installation_prefix" ||
		r.Files == nil || r.EntryPoints == nil || r.ProvidesCapabilities == nil || r.RequiresCapabilities == nil || r.CompatibleReceptors == nil || r.PlatformRequirements == nil {
		return errTrust
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(raw, &object) != nil {
		return errTrust
	}
	delete(object, "receipt_digest")
	canonical, err := c.Canonical(object)
	if err != nil {
		return errTrust
	}
	hash := sha256.Sum256(canonical)
	if r.ReceiptDigest != "sha256:"+hex.EncodeToString(hash[:]) {
		return errTrust
	}
	expected := map[string]c.PackageFile{}
	for _, f := range m.Files {
		if f.Path != m.ReceiptPath && f.Path != m.QualificationPath {
			expected[f.Path] = f
		}
	}
	if len(r.Files) != len(expected) || len(expected) == 0 {
		return errTrust
	}
	seen := map[string]bool{}
	for _, f := range r.Files {
		want, ok := expected[f.Path]
		if !ok || seen[f.Path] || f.Digest != want.Digest || strconv.FormatUint(f.Size, 10) != want.Size ||
			(f.Kind != "executable" && f.Kind != "regular") || (f.Kind == "executable") != want.Executable {
			return errTrust
		}
		seen[f.Path] = true
	}
	matched := false
	entries := map[string]bool{}
	for _, e := range r.EntryPoints {
		f, ok := expected[e.Path]
		if !ok || entries[e.EntryPointID] || e.EntryPointID == "" || e.Protocols == nil {
			return errTrust
		}
		entries[e.EntryPointID] = true
		if e.EntryPointID == m.ReceiptEntryPointID && e.Path == m.Executable && f.Executable && (e.Kind == "executable" || e.Kind == "adapter") {
			if m.Role == "carrier" {
				for _, protocol := range []string{"symphony.sov.ssh-endpoint.v2", "symphony.sov.ssh-invoke-input.v2", "symphony.sov.ssh-carrier-result.v2"} {
					found := false
					for _, admitted := range e.Protocols {
						found = found || admitted == protocol
					}
					if !found {
						return errTrust
					}
				}
			}
			matched = true
		}
	}
	if !matched {
		return errTrust
	}
	platform := strings.Split(m.Platform, "-")
	if len(platform) != 2 {
		return errTrust
	}
	if platform[0] == "darwin" {
		platform[0] = "macos"
	}
	critical := false
	for _, p := range r.PlatformRequirements {
		if p.Critical {
			if p.OS != platform[0] || p.Architecture != platform[1] || p.KernelABI != nil {
				return errTrust
			}
			critical = true
		}
	}
	if !critical {
		return errTrust
	}
	return nil
}
