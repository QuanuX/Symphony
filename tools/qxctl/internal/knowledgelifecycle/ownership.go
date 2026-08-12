package knowledgelifecycle

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sort"
	"time"
)

const (
	OwnershipProtocol      = "symphony.knowledge.lifecycle-root-ownership.v1"
	OwnershipFenceProtocol = "symphony.knowledge.lifecycle-root-ownership-fence.v1"
	packageMutationLock    = ".symphony-lifecycle.lock"
	ownershipRegistryFile  = ".symphony-lifecycle-ownership-v1.json"
	ownershipFenceModule   = "symphony-root-ownership"
	ownershipFenceVersion  = "1"
	ownershipFenceRelative = "share/symphony/receipts/symphony-root-ownership/1/install-receipt.json"
	ownershipFenceDocument = "{\"canonical\":false,\"format_version\":1,\"protocol\":\"symphony.knowledge.lifecycle-root-ownership-fence.v1\",\"purpose\":\"require_ownership_aware_lifecycle_mutation\"}\n"
	maxOwnershipBytes      = 4 << 20
	maxOwnershipClaims     = 16384
)

func isOwnershipFenceCandidate(candidate receiptCandidate) bool {
	return candidate.module == ownershipFenceModule && candidate.version == ownershipFenceVersion &&
		candidate.relativePath == ownershipFenceRelative
}

func validOwnershipFence(data []byte) bool {
	return string(data) == ownershipFenceDocument
}

type OwnershipClaim struct {
	ClaimID         string `json:"claim_id"`
	ClaimKind       string `json:"claim_kind"`
	ControlDomainID string `json:"control_domain_id"`
	TOPSID          string `json:"tops_id"`
	ProfileID       string `json:"profile_id"`
	ComponentID     string `json:"component_id"`
	ReceiptDigest   string `json:"receipt_digest"`
	Disposition     string `json:"disposition"`
}

type OwnershipRegistry struct {
	Protocol                        string           `json:"protocol"`
	FormatVersion                   uint64           `json:"format_version"`
	InstallRoot                     string           `json:"install_root"`
	EnforcementState                string           `json:"enforcement_state"`
	Generation                      uint64           `json:"generation"`
	PreviousOwnershipRegistryDigest *string          `json:"previous_ownership_registry_digest"`
	Claims                          []OwnershipClaim `json:"claims"`
	ObservedReceiptDigests          []string         `json:"observed_receipt_digests"`
	ReleasedReceiptDigests          []string         `json:"released_receipt_digests"`
	UpdatedAt                       string           `json:"updated_at"`
	Canonical                       bool             `json:"canonical"`
	OwnershipRegistryDigest         string           `json:"ownership_registry_digest"`
}

type OwnershipSnapshot struct {
	Exists   bool               `json:"exists"`
	Registry *OwnershipRegistry `json:"registry"`
}

type OwnershipResult struct {
	Protocol  string            `json:"protocol"`
	Operation string            `json:"operation"`
	Changed   bool              `json:"changed"`
	Snapshot  OwnershipSnapshot `json:"snapshot"`
	Canonical bool              `json:"canonical"`
	Digest    string            `json:"result_digest"`
}

type OwnershipReconciliation struct {
	Protocol  string            `json:"protocol"`
	TOPSID    string            `json:"tops_id"`
	ProfileID string            `json:"profile_id"`
	Results   []OwnershipResult `json:"results"`
	Changed   bool              `json:"changed"`
	Canonical bool              `json:"canonical"`
	Digest    string            `json:"reconciliation_digest"`
}

type OwnershipStore struct {
	installRoot     string
	controlDomainID string
	topsID          string
	profileID       string
	now             func() time.Time
}

func NewOwnershipStore(installRoot, stateRoot, topsID, profileID string) (*OwnershipStore, error) {
	root := filepath.Clean(installRoot)
	if !safeAbsolutePath(root) || root == "/" || !validTOPSID(topsID) || !safeToken(profileID, 256) {
		return nil, fmt.Errorf("shared-root ownership identity is invalid")
	}
	canonicalState, err := canonicalStateRoot(stateRoot)
	if err != nil {
		return nil, err
	}
	domain := sha256.Sum256([]byte("symphony.knowledge.lifecycle.control-domain.v1\n" + canonicalState + "\n" + topsID + "\n" + profileID))
	return &OwnershipStore{
		installRoot: root, controlDomainID: "sha256:" + hex.EncodeToString(domain[:]),
		topsID: topsID, profileID: profileID,
		now: func() time.Time { return time.Now().UTC() },
	}, nil
}

func (s *OwnershipStore) Snapshot() (OwnershipSnapshot, error) {
	return s.snapshotOwnership()
}

func (s *OwnershipStore) HasProfileClaims() (bool, error) {
	claims, err := s.ProfileClaims()
	return len(claims) != 0, err
}

func (s *OwnershipStore) ProfileClaims() ([]OwnershipClaim, error) {
	snapshot, err := s.Snapshot()
	if err != nil || !snapshot.Exists || snapshot.Registry == nil {
		return nil, err
	}
	claims := make([]OwnershipClaim, 0)
	for _, claim := range snapshot.Registry.Claims {
		if s.ownsClaim(claim) {
			claims = append(claims, claim)
		}
	}
	return claims, nil
}

func (e *Executor) ReconcileOwnership(desired DesiredState, observed Observation) (OwnershipReconciliation, error) {
	if desired.TOPSID != e.topsID || desired.ProfileID != e.profileID ||
		observed.TOPSID != e.topsID || observed.ProfileID != e.profileID {
		return OwnershipReconciliation{}, fmt.Errorf("ownership reconciliation identity is invalid")
	}
	roots := make(map[string]struct{})
	for _, component := range desired.Components {
		if safeAbsolutePath(component.InstallRoot) {
			roots[filepath.Clean(component.InstallRoot)] = struct{}{}
		}
	}
	for _, component := range observed.Components {
		for _, installed := range component.Packages {
			if safeAbsolutePath(installed.InstallRoot) {
				roots[filepath.Clean(installed.InstallRoot)] = struct{}{}
			}
		}
	}
	ordered := make([]string, 0, len(roots))
	for root := range roots {
		ordered = append(ordered, root)
	}
	if len(ordered) > 64 {
		return OwnershipReconciliation{}, fmt.Errorf("ownership reconciliation exceeds 64 installation roots")
	}
	sort.Strings(ordered)
	result := OwnershipReconciliation{
		Protocol: "symphony.knowledge.lifecycle-root-ownership-reconciliation.v1",
		TOPSID:   e.topsID, ProfileID: e.profileID, Results: make([]OwnershipResult, 0, len(ordered)), Canonical: false,
	}
	for _, root := range ordered {
		store, err := NewOwnershipStore(root, e.stateRoot, e.topsID, e.profileID)
		if err != nil {
			return OwnershipReconciliation{}, err
		}
		item, err := store.Reconcile(desired, observed)
		if err != nil {
			return OwnershipReconciliation{}, fmt.Errorf("reconcile ownership root %s: %w", root, err)
		}
		result.Results = append(result.Results, item)
		result.Changed = result.Changed || item.Changed
	}
	value, err := objectWithout(mustJSON(result), "reconciliation_digest")
	if err != nil {
		return OwnershipReconciliation{}, err
	}
	result.Digest, err = digestValue(value)
	return result, err
}

func (s *OwnershipStore) Reconcile(desired DesiredState, observed Observation) (OwnershipResult, error) {
	if desired.TOPSID != s.topsID || desired.ProfileID != s.profileID ||
		observed.TOPSID != s.topsID || observed.ProfileID != s.profileID {
		return OwnershipResult{}, fmt.Errorf("ownership reconciliation identity does not match the selected profile")
	}
	return s.mutateOwnership(true, "reconcile", "", func(registry *OwnershipRegistry, exists bool) (bool, error) {
		observedByComponent := make(map[string][]string)
		observedReceipts := make([]string, 0)
		for _, component := range observed.Components {
			for _, installed := range component.Packages {
				if filepath.Clean(installed.InstallRoot) != s.installRoot || !taggedDigest(installed.ReceiptDigest) {
					continue
				}
				observedByComponent[component.ComponentID] = append(observedByComponent[component.ComponentID], installed.ReceiptDigest)
				observedReceipts = append(observedReceipts, installed.ReceiptDigest)
			}
		}
		observedReceipts = uniqueSortedDigests(observedReceipts)

		if !exists {
			*registry = OwnershipRegistry{
				Protocol: OwnershipProtocol, FormatVersion: 1, InstallRoot: s.installRoot,
				EnforcementState: "enforced", Claims: make([]OwnershipClaim, 0),
				ObservedReceiptDigests: make([]string, 0), ReleasedReceiptDigests: make([]string, 0),
				Canonical: false,
			}
			if len(observedReceipts) != 0 {
				registry.EnforcementState = "adoption_required"
				for _, receipt := range observedReceipts {
					registry.Claims = append(registry.Claims, legacyOwnershipClaim(receipt))
				}
			}
		}

		previousState := registry.EnforcementState
		previousClaims := append([]OwnershipClaim(nil), registry.Claims...)
		previousObserved := append([]string(nil), registry.ObservedReceiptDigests...)
		previousReleased := append([]string(nil), registry.ReleasedReceiptDigests...)
		ownedComponents := make(map[string]struct{})
		retained := make(map[string]struct{})
		nextClaims := make([]OwnershipClaim, 0, len(registry.Claims)+len(desired.Components))
		for _, claim := range registry.Claims {
			if s.ownsClaim(claim) {
				ownedComponents[claim.ComponentID] = struct{}{}
				continue
			}
			nextClaims = append(nextClaims, claim)
		}
		for _, component := range desired.Components {
			if filepath.Clean(component.InstallRoot) != s.installRoot {
				continue
			}
			ownedComponents[component.ComponentID] = struct{}{}
			if component.Presence == "present" && component.SelectedPackage != nil {
				claim := s.profileClaim(component.ComponentID, component.SelectedPackage.ReceiptDigest, "retained")
				nextClaims = append(nextClaims, claim)
				retained[component.SelectedPackage.ReceiptDigest] = struct{}{}
			}
		}
		for componentID := range ownedComponents {
			for _, receipt := range observedByComponent[componentID] {
				if _, keep := retained[receipt]; keep {
					continue
				}
				nextClaims = append(nextClaims, s.profileClaim(componentID, receipt, "retiring"))
			}
		}
		registry.Claims = uniqueSortedClaims(nextClaims)
		registry.ObservedReceiptDigests = observedReceipts
		registry.ReleasedReceiptDigests = retainObservedDigests(registry.ReleasedReceiptDigests, observedReceipts)
		// A complete scan that no longer contains a receipt proves that a
		// retiring or legacy-preserve claim has no physical package left to
		// protect. Pruning those claims makes recovery converge if package
		// removal became durable before the registry update. Retained claims
		// remain, because they express desired presence and must expose the
		// missing package to their owning control domain.
		observedSet := make(map[string]struct{}, len(observedReceipts))
		for _, receipt := range observedReceipts {
			observedSet[receipt] = struct{}{}
		}
		presentClaims := make([]OwnershipClaim, 0, len(registry.Claims))
		for _, claim := range registry.Claims {
			_, present := observedSet[claim.ReceiptDigest]
			if !present && (claim.ClaimKind == "legacy" || claim.Disposition == "retiring") {
				continue
			}
			presentClaims = append(presentClaims, claim)
		}
		registry.Claims = presentClaims
		for receipt := range retained {
			registry.ReleasedReceiptDigests = removeDigest(registry.ReleasedReceiptDigests, receipt)
		}

		claimed := make(map[string]struct{})
		for _, claim := range registry.Claims {
			claimed[claim.ReceiptDigest] = struct{}{}
		}
		for _, receipt := range observedReceipts {
			if _, found := claimed[receipt]; found || containsString(registry.ReleasedReceiptDigests, receipt) {
				continue
			}
			registry.Claims = append(registry.Claims, legacyOwnershipClaim(receipt))
			registry.EnforcementState = "adoption_required"
		}
		registry.Claims = uniqueSortedClaims(registry.Claims)
		return !exists || previousState != registry.EnforcementState ||
			!ownershipClaimsEqual(previousClaims, registry.Claims) ||
			!sameStringSet(previousObserved, registry.ObservedReceiptDigests) ||
			!sameStringSet(previousReleased, registry.ReleasedReceiptDigests), nil
	})
}

func (s *OwnershipStore) Adopt(expected string) (OwnershipResult, error) {
	return s.mutateOwnership(false, "adopt", expected, func(registry *OwnershipRegistry, exists bool) (bool, error) {
		if !exists || registry.EnforcementState != "adoption_required" {
			return false, fmt.Errorf("ownership adoption requires an adoption_required registry")
		}
		retained := make(map[string]struct{})
		for _, claim := range registry.Claims {
			if claim.ClaimKind == "profile" && claim.Disposition == "retained" {
				retained[claim.ReceiptDigest] = struct{}{}
			}
		}
		next := make([]OwnershipClaim, 0, len(registry.Claims))
		for _, claim := range registry.Claims {
			if claim.ClaimKind == "legacy" {
				if _, mapped := retained[claim.ReceiptDigest]; mapped {
					continue
				}
			}
			next = append(next, claim)
		}
		registry.Claims = next
		registry.EnforcementState = "enforced"
		return true, nil
	})
}

func (s *OwnershipStore) ReleaseLegacy(receiptDigest, expected string) (OwnershipResult, error) {
	if !taggedDigest(receiptDigest) {
		return OwnershipResult{}, fmt.Errorf("legacy release requires an exact receipt digest")
	}
	return s.mutateOwnership(false, "release", expected, func(registry *OwnershipRegistry, exists bool) (bool, error) {
		if !exists || registry.EnforcementState != "enforced" {
			return false, fmt.Errorf("legacy release requires an enforced ownership registry")
		}
		found := false
		next := make([]OwnershipClaim, 0, len(registry.Claims))
		for _, claim := range registry.Claims {
			if claim.ClaimKind == "legacy" && claim.ReceiptDigest == receiptDigest {
				found = true
				continue
			}
			next = append(next, claim)
		}
		if !found {
			return false, fmt.Errorf("exact legacy-preserve claim is absent")
		}
		registry.Claims = next
		registry.ReleasedReceiptDigests = uniqueSortedDigests(append(registry.ReleasedReceiptDigests, receiptDigest))
		return true, nil
	})
}

func (s *OwnershipStore) profileClaim(componentID, receiptDigest, disposition string) OwnershipClaim {
	claim := OwnershipClaim{
		ClaimKind: "profile", ControlDomainID: s.controlDomainID, TOPSID: s.topsID,
		ProfileID: s.profileID, ComponentID: componentID, ReceiptDigest: receiptDigest,
		Disposition: disposition,
	}
	claim.ClaimID = ownershipClaimID(claim)
	return claim
}

func (s *OwnershipStore) ownsClaim(claim OwnershipClaim) bool {
	return claim.ClaimKind == "profile" && claim.ControlDomainID == s.controlDomainID &&
		claim.TOPSID == s.topsID && claim.ProfileID == s.profileID
}

func legacyOwnershipClaim(receiptDigest string) OwnershipClaim {
	claim := OwnershipClaim{
		ClaimKind: "legacy", ControlDomainID: "not_applicable", TOPSID: "not_applicable",
		ProfileID: "not_applicable", ComponentID: "not_applicable",
		ReceiptDigest: receiptDigest, Disposition: "legacy_preserve",
	}
	claim.ClaimID = ownershipClaimID(claim)
	return claim
}

func ownershipClaimID(claim OwnershipClaim) string {
	value := claim.ClaimKind + "\n" + claim.ControlDomainID + "\n" + claim.TOPSID + "\n" +
		claim.ProfileID + "\n" + claim.ComponentID + "\n" + claim.ReceiptDigest + "\n" + claim.Disposition
	digest := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(digest[:])
}

func (s *OwnershipStore) mutateOwnership(
	create bool,
	operation, expected string,
	mutation func(*OwnershipRegistry, bool) (bool, error),
) (OwnershipResult, error) {
	var result OwnershipResult
	err := s.withOwnershipLock(create, func(rootFD int, exists bool, registry OwnershipRegistry) error {
		if expected != "" {
			if expected != "absent" && !taggedDigest(expected) {
				return fmt.Errorf("expected ownership registry must be absent or an exact tagged digest")
			}
			if (expected == "absent") != !exists || exists && expected != registry.OwnershipRegistryDigest {
				return fmt.Errorf("ownership registry compare-and-swap mismatch")
			}
		}
		before := registry
		changed, err := mutation(&registry, exists)
		if err != nil {
			return err
		}
		if !changed {
			result = ownershipResult(operation, false, ownershipSnapshot(exists, registry))
			return nil
		}
		if exists {
			if registry.Generation >= 9007199254740991 {
				return fmt.Errorf("ownership registry generation is exhausted")
			}
			registry.Generation++
			registry.PreviousOwnershipRegistryDigest = stringPointer(before.OwnershipRegistryDigest)
		} else {
			registry.Generation = 1
			registry.PreviousOwnershipRegistryDigest = nil
		}
		registry.UpdatedAt = s.now().UTC().Truncate(time.Second).Format(time.RFC3339)
		if err := finalizeOwnershipRegistry(&registry); err != nil {
			return err
		}
		if err := writeOwnershipRegistryAt(rootFD, registry); err != nil {
			return err
		}
		result = ownershipResult(operation, true, ownershipSnapshot(true, registry))
		return nil
	})
	return result, err
}

func ownershipSnapshot(exists bool, registry OwnershipRegistry) OwnershipSnapshot {
	if !exists {
		return OwnershipSnapshot{Exists: false, Registry: nil}
	}
	copy := registry
	return OwnershipSnapshot{Exists: true, Registry: &copy}
}

func ownershipResult(operation string, changed bool, snapshot OwnershipSnapshot) OwnershipResult {
	result := OwnershipResult{
		Protocol: "symphony.knowledge.lifecycle-root-ownership-result.v1", Operation: operation,
		Changed: changed, Snapshot: snapshot, Canonical: false,
	}
	value, err := objectWithout(mustJSON(result), "result_digest")
	if err == nil {
		result.Digest, _ = digestValue(value)
	}
	return result
}

func OwnershipView(operation string, snapshot OwnershipSnapshot) OwnershipResult {
	return ownershipResult(operation, false, snapshot)
}

func finalizeOwnershipRegistry(registry *OwnershipRegistry) error {
	registry.Claims = uniqueSortedClaims(registry.Claims)
	registry.ObservedReceiptDigests = uniqueSortedDigests(registry.ObservedReceiptDigests)
	registry.ReleasedReceiptDigests = uniqueSortedDigests(registry.ReleasedReceiptDigests)
	registry.OwnershipRegistryDigest = ""
	value, err := objectWithout(mustJSON(*registry), "ownership_registry_digest")
	if err != nil {
		return err
	}
	registry.OwnershipRegistryDigest, err = digestValue(value)
	return err
}

func validateOwnershipRegistry(registry OwnershipRegistry, installRoot string) error {
	if registry.Protocol != OwnershipProtocol || registry.FormatVersion != 1 || registry.InstallRoot != installRoot ||
		!oneOf(registry.EnforcementState, "adoption_required", "enforced") || registry.Generation == 0 ||
		registry.Generation > 9007199254740991 || registry.Canonical || !validSTSCSeconds(registry.UpdatedAt) ||
		!taggedDigest(registry.OwnershipRegistryDigest) || len(registry.Claims) > maxOwnershipClaims ||
		len(registry.ObservedReceiptDigests) > 4096 || len(registry.ReleasedReceiptDigests) > 4096 {
		return fmt.Errorf("ownership registry identity is invalid")
	}
	if (registry.Generation == 1) != (registry.PreviousOwnershipRegistryDigest == nil) ||
		registry.PreviousOwnershipRegistryDigest != nil && !taggedDigest(*registry.PreviousOwnershipRegistryDigest) {
		return fmt.Errorf("ownership registry predecessor is invalid")
	}
	if !sortedUniqueDigests(registry.ObservedReceiptDigests) || !sortedUniqueDigests(registry.ReleasedReceiptDigests) {
		return fmt.Errorf("ownership receipt sets are invalid")
	}
	seen := make(map[string]struct{}, len(registry.Claims))
	seenCoordinates := make(map[string]struct{}, len(registry.Claims))
	for index, claim := range registry.Claims {
		if !taggedDigest(claim.ClaimID) || !taggedDigest(claim.ReceiptDigest) ||
			!oneOf(claim.ClaimKind, "profile", "legacy") ||
			claim.ClaimKind == "profile" && (!taggedDigest(claim.ControlDomainID) || !validTOPSID(claim.TOPSID) ||
				!safeToken(claim.ProfileID, 256) || !safeToken(claim.ComponentID, 256) || !oneOf(claim.Disposition, "retained", "retiring")) ||
			claim.ClaimKind == "legacy" && (claim.ControlDomainID != "not_applicable" || claim.TOPSID != "not_applicable" ||
				claim.ProfileID != "not_applicable" || claim.ComponentID != "not_applicable" || claim.Disposition != "legacy_preserve") ||
			ownershipClaimID(claim) != claim.ClaimID {
			return fmt.Errorf("ownership claim is invalid")
		}
		if _, duplicate := seen[claim.ClaimID]; duplicate {
			return fmt.Errorf("ownership claim is duplicated")
		}
		if index > 0 && claim.ClaimID <= registry.Claims[index-1].ClaimID {
			return fmt.Errorf("ownership claims are not in canonical order")
		}
		seen[claim.ClaimID] = struct{}{}
		coordinate := claim.ClaimKind + "\n" + claim.ControlDomainID + "\n" + claim.TOPSID + "\n" +
			claim.ProfileID + "\n" + claim.ComponentID + "\n" + claim.ReceiptDigest
		if _, duplicate := seenCoordinates[coordinate]; duplicate {
			return fmt.Errorf("ownership claim coordinate is duplicated")
		}
		seenCoordinates[coordinate] = struct{}{}
	}
	for _, released := range registry.ReleasedReceiptDigests {
		if !containsString(registry.ObservedReceiptDigests, released) {
			return fmt.Errorf("released ownership receipt is not observed")
		}
	}
	copy := registry
	copy.OwnershipRegistryDigest = ""
	value, err := objectWithout(mustJSON(copy), "ownership_registry_digest")
	if err != nil {
		return err
	}
	digest, err := digestValue(value)
	if err != nil || digest != registry.OwnershipRegistryDigest {
		return fmt.Errorf("ownership registry digest mismatch")
	}
	return nil
}

func uniqueSortedClaims(values []OwnershipClaim) []OwnershipClaim {
	byID := make(map[string]OwnershipClaim, len(values))
	for _, value := range values {
		byID[value.ClaimID] = value
	}
	result := make([]OwnershipClaim, 0, len(byID))
	for _, value := range byID {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ClaimID < result[j].ClaimID })
	return result
}

func uniqueSortedDigests(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if taggedDigest(value) {
			seen[value] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func sortedUniqueDigests(values []string) bool {
	for index, value := range values {
		if !taggedDigest(value) || index > 0 && value <= values[index-1] {
			return false
		}
	}
	return true
}

func retainObservedDigests(values, observed []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if containsString(observed, value) {
			result = append(result, value)
		}
	}
	return uniqueSortedDigests(result)
}

func removeDigest(values []string, removed string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != removed {
			result = append(result, value)
		}
	}
	return result
}

func ownershipClaimsEqual(left, right []OwnershipClaim) bool {
	left = uniqueSortedClaims(left)
	right = uniqueSortedClaims(right)
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func sameStringSet(left, right []string) bool {
	left = uniqueSortedDigests(left)
	right = uniqueSortedDigests(right)
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
