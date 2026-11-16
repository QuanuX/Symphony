//go:build darwin || linux

package packageretirement

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"golang.org/x/sys/unix"
)

const ownershipRegistryPath = ".symphony-lifecycle-ownership-v1.json"
const ownershipFencePath = "share/symphony/receipts/symphony-root-ownership/1/install-receipt.json"
const ownershipFenceBytes = "{\"canonical\":false,\"format_version\":1,\"protocol\":\"symphony.knowledge.lifecycle-root-ownership-fence.v1\",\"purpose\":\"require_ownership_aware_lifecycle_mutation\"}\n"

// Delegation owns duplicates of the inherited open file descriptions. Close
// never unlocks them: the qxctl parent retains the same lock until child reap.
type Delegation struct {
	proof      DispatchProof
	root, lock *os.File
}

func (d *Delegation) Close() error {
	if d == nil {
		return nil
	}
	var err error
	for _, f := range []**os.File{&d.lock, &d.root} {
		if *f != nil {
			err = errors.Join(err, (*f).Close())
			*f = nil
		}
	}
	return err
}
func duplicateDispatchFD(fd int, name string) (*os.File, error) {
	if fd < 3 {
		return nil, fmt.Errorf("dispatch requires explicit inherited descriptors")
	}
	copy, err := unix.Dup(fd)
	if err != nil {
		return nil, err
	}
	unix.CloseOnExec(copy)
	return os.NewFile(uintptr(copy), name), nil
}

// ReadDispatch accepts either no fd flags, or all three explicit descriptors.
// Input assertions are checked against actual locked files and exact claims.
func ReadDispatch(rootFD, lockFD, proofFD int, operation, componentID, requestedVersion, prefix string) (*Delegation, error) {
	if rootFD == -1 && lockFD == -1 && proofFD == -1 {
		return nil, nil
	}
	if rootFD < 3 || lockFD < 3 || proofFD < 3 || rootFD == lockFD || rootFD == proofFD || lockFD == proofFD {
		return nil, fmt.Errorf("lifecycle descriptor flags must be supplied together and distinct")
	}
	proofFile, err := duplicateDispatchFD(proofFD, "lifecycle dispatch proof")
	if err != nil {
		return nil, err
	}
	defer proofFile.Close()
	raw, err := io.ReadAll(io.LimitReader(proofFile, 65537))
	if err != nil || len(raw) > 65536 {
		return nil, fmt.Errorf("dispatch proof is unavailable or exceeds its bound")
	}
	var proof DispatchProof
	if err := strictDispatchDecode(raw, &proof); err != nil {
		return nil, err
	}
	if proof.Protocol != DispatchCapability || proof.FormatVersion != 1 || proof.Operation != operation || (operation != "install" && operation != "uninstall") || proof.ComponentID != componentID || proof.Version != requestedVersion || proof.InstallRoot != prefix || !filepath.IsAbs(prefix) || filepath.Clean(prefix) != prefix || prefix == "/" || !validSegment(componentID) || !validSegment(requestedVersion) || !validDigest(proof.ReceiptDigest) || !validDigest(proof.BinaryDigest) || !validDigest(proof.ControlDomainID) || stavprotocol.ValidateTOPSID(proof.TOPSID) != nil || !dispatchToken(proof.ProfileID) || !validDigest(proof.OwnershipRegistryDigest) || proof.OwnershipGeneration == 0 || proof.OwnershipGeneration > 9007199254740991 || len(proof.Nonce) != 32 || strings.ToLower(proof.Nonce) != proof.Nonce {
		return nil, fmt.Errorf("dispatch proof does not match exact native operation")
	}
	if _, err := hex.DecodeString(proof.Nonce); err != nil {
		return nil, fmt.Errorf("invalid dispatch nonce")
	}
	d := &Delegation{proof: proof}
	d.root, err = duplicateDispatchFD(rootFD, "lifecycle root")
	if err != nil {
		return nil, err
	}
	d.lock, err = duplicateDispatchFD(lockFD, "lifecycle root lock")
	if err != nil {
		d.Close()
		return nil, err
	}
	// Original inherited descriptors must not escape through any later exec.
	unix.CloseOnExec(rootFD)
	unix.CloseOnExec(lockFD)
	unix.CloseOnExec(proofFD)
	if err = d.validate(); err != nil {
		d.Close()
		return nil, err
	}
	return d, nil
}
func dispatchToken(s string) bool {
	if len(s) == 0 || len(s) > 256 {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("._:-", r)) {
			return false
		}
	}
	return true
}
func (d *Delegation) validateIdentity(prefix string, id Identity, operation string) error {
	if d == nil {
		return nil
	}
	p := d.proof
	if prefix != p.InstallRoot || id.Owner != p.ComponentID || id.Version != p.Version || id.ReceiptDigest != p.ReceiptDigest || id.BinaryDigest != p.BinaryDigest || operation != p.Operation {
		return fmt.Errorf("native package identity differs from delegated operation")
	}
	return d.validate()
}
func (d *Delegation) validate() error {
	if d == nil || d.root == nil || d.lock == nil {
		return fmt.Errorf("dispatch custody is closed")
	}
	current, st, err := openRoot(d.proof.InstallRoot)
	if err != nil {
		return err
	}
	defer current.Close()
	if !sameFile(current, d.root) || uint64(st.Dev) != d.proof.RootDevice || st.Ino != d.proof.RootInode {
		return fmt.Errorf("delegated root identity changed")
	}
	g := &Lease{prefix: d.proof.InstallRoot, root: current, rootStat: st}
	fresh, err := g.openFile(current, ".symphony-lifecycle.lock", false)
	if err != nil {
		return err
	}
	defer fresh.Close()
	var lockStat unix.Stat_t
	if unix.Fstat(int(d.lock.Fd()), &lockStat) != nil || !sameFile(fresh, d.lock) || uint64(lockStat.Dev) != d.proof.LockDevice || lockStat.Ino != d.proof.LockInode {
		return fmt.Errorf("delegated root lock identity differs")
	}
	// A fresh description must be excluded, while the inherited description
	// must already share exclusive custody. Neither operation drops that lock.
	if err = unix.Flock(int(fresh.Fd()), unix.LOCK_EX|unix.LOCK_NB); err == nil {
		_ = unix.Flock(int(fresh.Fd()), unix.LOCK_UN)
		return fmt.Errorf("delegated root lock was not held")
	} else if !errors.Is(err, unix.EWOULDBLOCK) && !errors.Is(err, unix.EAGAIN) {
		return err
	}
	if err = unix.Flock(int(d.lock.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		return fmt.Errorf("inherited descriptor does not own the root lock: %w", err)
	}
	fence, err := readDispatchFile(g, ownershipFencePath, 4096)
	if err != nil || string(fence) != ownershipFenceBytes {
		return fmt.Errorf("shared-root compatibility fence differs")
	}
	raw, err := readDispatchFile(g, ownershipRegistryPath, 4<<20)
	if err != nil {
		return err
	}
	return validateDelegatedRegistry(raw, d.proof)
}
func readDispatchFile(g *Lease, relative string, limit int64) ([]byte, error) {
	parent := g.root
	if filepath.Dir(relative) != "." {
		d, err := g.openDir(filepath.ToSlash(filepath.Dir(relative)), false)
		if err != nil {
			return nil, err
		}
		defer d.Close()
		parent = d
	}
	f, err := g.openFile(parent, filepath.Base(relative), false)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil || int64(len(b)) > limit {
		return nil, fmt.Errorf("delegated ownership evidence exceeds its bound")
	}
	return b, nil
}
func (g *Lease) adoptDelegation(d *Delegation) error {
	if err := d.validate(); err != nil {
		return err
	}
	if !sameFile(g.root, d.root) {
		return fmt.Errorf("native root differs from delegated root")
	}
	fd, err := unix.Dup(int(d.lock.Fd()))
	if err != nil {
		return err
	}
	unix.CloseOnExec(fd)
	g.mutation = os.NewFile(uintptr(fd), "delegated lifecycle root lock")
	return nil
}

type dispatchClaim struct {
	ClaimID         string `json:"claim_id"`
	ClaimKind       string `json:"claim_kind"`
	ControlDomainID string `json:"control_domain_id"`
	TOPSID          string `json:"tops_id"`
	ProfileID       string `json:"profile_id"`
	ComponentID     string `json:"component_id"`
	ReceiptDigest   string `json:"receipt_digest"`
	Disposition     string `json:"disposition"`
}
type dispatchRegistry struct {
	Protocol         string          `json:"protocol"`
	FormatVersion    uint64          `json:"format_version"`
	InstallRoot      string          `json:"install_root"`
	EnforcementState string          `json:"enforcement_state"`
	Generation       uint64          `json:"generation"`
	Previous         *string         `json:"previous_ownership_registry_digest"`
	Claims           []dispatchClaim `json:"claims"`
	Observed         []string        `json:"observed_receipt_digests"`
	Released         []string        `json:"released_receipt_digests"`
	UpdatedAt        string          `json:"updated_at"`
	Canonical        bool            `json:"canonical"`
	Digest           string          `json:"ownership_registry_digest"`
}

func dispatchDigest(b []byte) string {
	h := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(h[:])
}
func validDigestSet(set []string) bool {
	if set == nil || len(set) > 4096 {
		return false
	}
	for i, s := range set {
		if !validDigest(s) || (i > 0 && set[i-1] >= s) {
			return false
		}
	}
	return true
}
func containsDigest(set []string, s string) bool {
	for _, v := range set {
		if v == s {
			return true
		}
	}
	return false
}
func validateDelegatedRegistry(raw []byte, p DispatchProof) error {
	var r dispatchRegistry
	if err := strictDispatchDecode(raw, &r); err != nil {
		return err
	}
	if r.Protocol != "symphony.knowledge.lifecycle-root-ownership.v1" || r.FormatVersion != 1 || r.InstallRoot != p.InstallRoot || r.EnforcementState != "enforced" || r.Generation != p.OwnershipGeneration || r.Canonical || r.Digest != p.OwnershipRegistryDigest || r.Claims == nil || len(r.Claims) > 16384 || !validDigestSet(r.Observed) || !validDigestSet(r.Released) {
		return fmt.Errorf("delegated ownership registry does not match enforced evidence")
	}
	timestamp, err := time.Parse("2006-01-02T15:04:05Z", r.UpdatedAt)
	if err != nil || timestamp.Format("2006-01-02T15:04:05Z") != r.UpdatedAt {
		return fmt.Errorf("invalid ownership timestamp")
	}
	if (r.Generation == 1) != (r.Previous == nil) || (r.Previous != nil && !validDigest(*r.Previous)) {
		return fmt.Errorf("invalid ownership predecessor")
	}
	for _, v := range r.Released {
		if !containsDigest(r.Observed, v) {
			return fmt.Errorf("release lacks observed receipt")
		}
	}
	var obj map[string]any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&obj); err != nil {
		return err
	}
	delete(obj, "ownership_registry_digest")
	canonical, err := json.Marshal(obj)
	if err != nil || dispatchDigest(canonical) != r.Digest {
		return fmt.Errorf("ownership registry digest mismatch")
	}
	allowed := p.Operation == "uninstall" && containsDigest(r.Released, p.ReceiptDigest)
	coordinates := map[string]bool{}
	for i, c := range r.Claims {
		coordinate := c.ClaimKind + "\n" + c.ControlDomainID + "\n" + c.TOPSID + "\n" + c.ProfileID + "\n" + c.ComponentID + "\n" + c.ReceiptDigest
		if !validDigest(c.ClaimID) || !validDigest(c.ReceiptDigest) || c.ClaimID != dispatchDigest([]byte(coordinate+"\n"+c.Disposition)) || (i > 0 && r.Claims[i-1].ClaimID >= c.ClaimID) || coordinates[coordinate] {
			return fmt.Errorf("invalid or duplicate ownership claim")
		}
		coordinates[coordinate] = true
		if c.ClaimKind == "profile" {
			if !validDigest(c.ControlDomainID) || stavprotocol.ValidateTOPSID(c.TOPSID) != nil || !dispatchToken(c.ProfileID) || !dispatchToken(c.ComponentID) || (c.Disposition != "retained" && c.Disposition != "retiring") {
				return fmt.Errorf("invalid profile ownership claim")
			}
		} else if c.ClaimKind != "legacy" || c.ControlDomainID != "not_applicable" || c.TOPSID != "not_applicable" || c.ProfileID != "not_applicable" || c.ComponentID != "not_applicable" || c.Disposition != "legacy_preserve" {
			return fmt.Errorf("unknown ownership claim")
		}
		if c.ReceiptDigest != p.ReceiptDigest {
			continue
		}
		own := c.ClaimKind == "profile" && c.ControlDomainID == p.ControlDomainID && c.TOPSID == p.TOPSID && c.ProfileID == p.ProfileID && c.ComponentID == p.ComponentID
		if p.Operation == "install" {
			if own && c.Disposition == "retained" {
				allowed = true
			}
			continue
		}
		if c.ClaimKind != "profile" || c.Disposition != "retiring" {
			return fmt.Errorf("receipt remains retained by another ownership claim")
		}
		if own {
			allowed = true
		}
	}
	if !allowed {
		return fmt.Errorf("exact delegated retained or release claim is absent")
	}
	return nil
}

// Match qxctl's existing null-capable canonical map digest, while rejecting
// duplicates and unknown fields before a typed proof or registry is trusted.
func strictDispatchDecode(raw []byte, out any) error {
	if !utf8.Valid(raw) {
		return fmt.Errorf("invalid UTF-8 ownership evidence")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var walk func(int) error
	walk = func(depth int) error {
		if depth > 32 {
			return fmt.Errorf("ownership JSON is too deep")
		}
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		d, ok := tok.(json.Delim)
		if !ok {
			return nil
		}
		switch d {
		case '{':
			seen := map[string]bool{}
			for dec.More() {
				key, err := dec.Token()
				if err != nil {
					return err
				}
				name, ok := key.(string)
				if !ok || seen[name] {
					return fmt.Errorf("duplicate ownership JSON key")
				}
				seen[name] = true
				if err := walk(depth + 1); err != nil {
					return err
				}
			}
		case '[':
			for dec.More() {
				if err := walk(depth + 1); err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("invalid JSON delimiter")
		}
		_, err = dec.Token()
		return err
	}
	if err := walk(0); err != nil {
		return err
	}
	if _, err := dec.Token(); err != io.EOF {
		return fmt.Errorf("ownership JSON has trailing values")
	}
	// Every top-level field is mandatory, including false and null values;
	// accepting omitted defaults would describe a different wire contract.
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	var required []string
	switch out.(type) {
	case *DispatchProof:
		required = []string{"protocol", "format_version", "operation", "component_id", "version", "install_root", "receipt_digest", "binary_digest", "control_domain_id", "tops_id", "profile_id", "ownership_registry_digest", "ownership_generation", "root_device", "root_inode", "lock_device", "lock_inode", "nonce"}
	case *dispatchRegistry:
		required = []string{"protocol", "format_version", "install_root", "enforcement_state", "generation", "previous_ownership_registry_digest", "claims", "observed_receipt_digests", "released_receipt_digests", "updated_at", "canonical", "ownership_registry_digest"}
	}
	if len(fields) != len(required) {
		return fmt.Errorf("ownership evidence has unexpected fields")
	}
	if _, registry := out.(*dispatchRegistry); registry {
		var claims []map[string]json.RawMessage
		if err := json.Unmarshal(fields["claims"], &claims); err != nil {
			return err
		}
		claimKeys := []string{"claim_id", "claim_kind", "control_domain_id", "tops_id", "profile_id", "component_id", "receipt_digest", "disposition"}
		for _, claim := range claims {
			if len(claim) != len(claimKeys) {
				return fmt.Errorf("ownership claim has unexpected fields")
			}
			for _, key := range claimKeys {
				value, ok := claim[key]
				if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
					return fmt.Errorf("ownership claim lacks non-null %s", key)
				}
			}
		}
	}
	for _, key := range required {
		value, ok := fields[key]
		if !ok {
			return fmt.Errorf("ownership evidence lacks %s", key)
		}
		if key != "previous_ownership_registry_digest" && bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("ownership evidence has null %s", key)
		}
	}
	typed := json.NewDecoder(bytes.NewReader(raw))
	typed.DisallowUnknownFields()
	if err := typed.Decode(out); err != nil {
		return err
	}
	return nil
}
