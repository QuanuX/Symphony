//go:build darwin || linux

package sshprofile

import (
	c "github.com/QuanuX/Symphony/tools/qxctl/internal/sshprofilewire"
	"runtime"
	"strings"
)

func (d *directory) inventory(prefix string) ([]string, error) {
	names, e := d.names()
	if e != nil {
		return nil, e
	}
	result := []string{}
	for _, name := range names {
		child, e := d.child(name)
		if e == nil {
			files, err := child.inventory(prefix + name + "/")
			child.close()
			if err != nil {
				return nil, err
			}
			result = append(result, files...)
		} else {
			// The subsequent digest/open checks reject links, devices and pipes.
			result = append(result, prefix+name)
		}
	}
	return result, nil
}
func verifyPackage(root *directory, ref c.PackageRef, role, platform string) error {
	id, e := root.child(ref.PackageID)
	if e != nil {
		return errTrust
	}
	defer id.close()
	d, e := id.child(ref.Version)
	if e != nil {
		return errTrust
	}
	defer d.close()
	raw, e := d.read("manifest.json", c.MaxDocumentBytes)
	var m c.PackageManifest
	if e != nil || c.Decode(raw, &m) != nil || m.Protocol != c.PackageProtocol || m.PackageID != ref.PackageID || m.Version != ref.Version || m.Role != role || m.Platform != platform || m.Digest != ref.ManifestDigest || m.Digest != c.Digest(c.PackageProtocol, m) || m.ReceiptDigest != ref.ReceiptDigest || !c.ValidDigest(m.QualificationRef) || !c.ValidPath(m.Executable) || !c.ValidPath(m.ReceiptPath) || !c.ValidPath(m.QualificationPath) || m.QualificationPath == m.ReceiptPath || m.QualificationPath == m.Executable || m.ReceiptPath == m.Executable || len(m.Files) == 0 {
		return errTrust
	}
	seen := map[string]bool{"manifest.json": true}
	foundExecutable, foundReceipt, foundQualification := false, false, false
	for _, f := range m.Files {
		if !c.ValidPath(f.Path) || seen[f.Path] || !c.ValidDigest(f.Digest) || !c.Decimal(f.Size, ^uint64(0), true) {
			return errTrust
		}
		seen[f.Path] = true
		digest, size, executable, e := d.digestFile(f.Path)
		if e != nil || digest != f.Digest || size != f.Size || executable != f.Executable {
			return errTrust
		}
		if f.Path == m.Executable {
			foundExecutable = executable && digest == ref.ExecutableDigest
		}
		if f.Path == m.QualificationPath {
			foundQualification = !executable && digest == m.QualificationRef && size != "0"
		}
		if f.Path == m.ReceiptPath {
			foundReceipt = !executable && digest == ref.ReceiptDigest
		}
	}
	if !foundExecutable || !foundReceipt || !foundQualification {
		return errTrust
	}
	files, e := d.inventory("")
	if e != nil || len(files) != len(seen) {
		return errTrust
	}
	for _, p := range files {
		if !seen[p] {
			return errTrust
		}
	}
	// Validate the complete existing receipt-v2, including its intrinsic digest
	// and exact file/entry-point coverage. Qualification evidence is opaque
	// administrator-reviewed content pinned by bytes, not a live-test assertion.
	parts := strings.Split(m.ReceiptPath, "/")
	receiptDir := d
	for _, part := range parts[:len(parts)-1] {
		n, err := receiptDir.child(part)
		if receiptDir != d {
			receiptDir.close()
		}
		if err != nil {
			return errTrust
		}
		receiptDir = n
	}
	if receiptDir != d {
		defer receiptDir.close()
	}
	receipt, e := receiptDir.read(parts[len(parts)-1], c.MaxDocumentBytes)
	if e != nil || verifyReceipt(receipt, m) != nil {
		return errTrust
	}
	return nil
}
func (s *Store) verifyTrust(snapshot c.Snapshot) error {
	if snapshot.Profile.State != "enabled" {
		return nil
	}
	config := snapshot.Profile.Configuration
	if config.Platform != runtime.GOOS+"-"+runtime.GOARCH || s.packages == nil {
		return errTrust
	}
	for _, p := range []struct {
		ref  c.PackageRef
		role string
	}{{config.Adapter, "carrier"}, {config.SSH, "ssh"}, {config.Helper, "proxy"}} {
		if verifyPackage(s.packages, p.ref, p.role, config.Platform) != nil {
			return errTrust
		}
	}
	return nil
}
