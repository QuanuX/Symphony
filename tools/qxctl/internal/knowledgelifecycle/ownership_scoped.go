package knowledgelifecycle

import "fmt"

// reconcileScoped changes only reviewed claim coordinates. A complete inventory
// is still used to detect unknown packages, but never authorizes their adoption,
// another profile's cleanup, or compatibility-fence repair.
func (s *OwnershipStore) reconcileScoped(desired DesiredState, observed Observation) (OwnershipResult, error) {
	scope := s.executionScope
	if scope == nil {
		return OwnershipResult{}, fmt.Errorf("execution scope is required")
	}
	if err := scope.ValidateDesired(scope.ProfileDigest, desired, scope.SourceReportJournalDigest); err != nil {
		return OwnershipResult{}, err
	}
	return s.mutateOwnership(true, "reconcile", "", func(registry *OwnershipRegistry, exists bool) (bool, error) {
		receipts := make([]string, 0)
		byComponent := map[string][]string{}
		for _, component := range observed.Components {
			for _, installed := range component.Packages {
				if installed.InstallRoot == s.installRoot {
					if !taggedDigest(installed.ReceiptDigest) {
						return false, fmt.Errorf("invalid observed receipt")
					}
					receipts = append(receipts, installed.ReceiptDigest)
					byComponent[component.ComponentID] = append(byComponent[component.ComponentID], installed.ReceiptDigest)
				}
			}
		}
		receipts = uniqueSortedDigests(receipts)
		if !exists {
			if len(receipts) != 0 {
				return false, fmt.Errorf("scoped reconciliation cannot adopt an existing installation")
			}
			*registry = OwnershipRegistry{Protocol: OwnershipProtocol, FormatVersion: 1, InstallRoot: s.installRoot, EnforcementState: "enforced", Claims: []OwnershipClaim{}, ObservedReceiptDigests: []string{}, ReleasedReceiptDigests: []string{}}
		}
		if registry.EnforcementState != "enforced" {
			return false, fmt.Errorf("scoped reconciliation requires enforced ownership; explicit adoption is separate")
		}
		known := map[string]bool{}
		for _, claim := range registry.Claims {
			known[claim.ReceiptDigest] = true
		}
		for _, receipt := range registry.ReleasedReceiptDigests {
			known[receipt] = true
		}
		for _, receipt := range receipts {
			if !known[receipt] {
				return false, fmt.Errorf("observed receipt without existing ownership requires explicit adoption outside scope")
			}
		}
		beforeClaims := append([]OwnershipClaim(nil), registry.Claims...)
		beforeObserved := append([]string(nil), registry.ObservedReceiptDigests...)
		beforeReleased := append([]string(nil), registry.ReleasedReceiptDigests...)
		next := make([]OwnershipClaim, 0, len(registry.Claims))
		wanted := map[string]OwnershipClaim{}
		for _, component := range desired.Components {
			if !scope.Contains(component.ComponentID) || component.InstallRoot != s.installRoot {
				continue
			}
			if component.Presence == "present" && component.SelectedPackage != nil {
				c := s.profileClaim(component.ComponentID, component.SelectedPackage.ReceiptDigest, "retained")
				wanted[component.ComponentID+"\n"+c.ReceiptDigest] = c
			}
			for _, receipt := range byComponent[component.ComponentID] {
				key := component.ComponentID + "\n" + receipt
				if _, ok := wanted[key]; !ok {
					wanted[key] = s.profileClaim(component.ComponentID, receipt, "retiring")
				}
			}
		}
		for _, claim := range registry.Claims {
			if !s.ownsClaim(claim) || !scope.Contains(claim.ComponentID) {
				next = append(next, claim)
				continue
			}
			key := claim.ComponentID + "\n" + claim.ReceiptDigest
			target, ok := wanted[key]
			if ok {
				if target.ClaimID != claim.ClaimID && !scope.AllowsOwnership(s.installRoot, claim.ComponentID, claim.ReceiptDigest, target.Disposition) {
					return false, fmt.Errorf("claim transition exceeds execution scope")
				}
				next = append(next, target)
				delete(wanted, key)
				continue
			}
			if scope.AllowsOwnership(s.installRoot, claim.ComponentID, claim.ReceiptDigest, "release") && !containsString(receipts, claim.ReceiptDigest) {
				continue
			}
			next = append(next, claim)
		}
		for _, claim := range wanted {
			if !scope.AllowsOwnership(s.installRoot, claim.ComponentID, claim.ReceiptDigest, claim.Disposition) {
				return false, fmt.Errorf("new claim exceeds execution scope")
			}
			next = append(next, claim)
		}
		registry.Claims = uniqueSortedClaims(next)
		claimed := map[string]bool{}
		for _, claim := range registry.Claims {
			claimed[claim.ReceiptDigest] = true
			if s.ownsClaim(claim) && scope.Contains(claim.ComponentID) && claim.Disposition == "retained" && scope.AllowsOwnership(s.installRoot, claim.ComponentID, claim.ReceiptDigest, "retained") {
				registry.ReleasedReceiptDigests = removeDigest(registry.ReleasedReceiptDigests, claim.ReceiptDigest)
			}
		}
		for _, receipt := range receipts {
			if !claimed[receipt] && !containsString(registry.ReleasedReceiptDigests, receipt) {
				return false, fmt.Errorf("unknown observed receipt requires separate ownership review")
			}
		}
		registry.ObservedReceiptDigests = receipts
		return !exists || !ownershipClaimsEqual(beforeClaims, registry.Claims) || !sameStringSet(beforeObserved, receipts) || !sameStringSet(beforeReleased, registry.ReleasedReceiptDigests), nil
	})
}
