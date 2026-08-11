package validation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgebinding"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
)

const (
	maxStateBytes = 4 * 1024 * 1024
	maxProfiles   = 256
	maxBaselines  = 256
)

type Store struct {
	stateRoot string
	topsID    string
}

func NewStore(stateRoot, topsID string) (*Store, error) {
	if !validTOPSID(topsID) {
		return nil, fmt.Errorf("TOPS ID must be a canonical non-nil lowercase RFC UUID")
	}
	if stateRoot == "" {
		var err error
		stateRoot, err = knowledgebinding.DefaultStateRoot()
		if err != nil {
			return nil, err
		}
	}
	canonical, err := canonicalStateRoot(stateRoot)
	if err != nil {
		return nil, err
	}
	return &Store{stateRoot: canonical, topsID: topsID}, nil
}

func (s *Store) Policy(profileID string) (PolicySnapshot, error) {
	if !safeToken(profileID) {
		return PolicySnapshot{}, fmt.Errorf("profile ID has invalid syntax")
	}
	var snapshot PolicySnapshot
	err := s.withStateLock("profiles", false, func(directory *os.File) error {
		data, exists, err := readStateFile(directory, "profile", profileID)
		if err != nil || !exists {
			snapshot.Exists = exists
			return err
		}
		policy, err := decodePolicy(data)
		if err != nil {
			return err
		}
		if policy.ProfileID != profileID || policy.TOPSID != s.topsID {
			return fmt.Errorf("validation policy storage identity mismatch")
		}
		snapshot = PolicySnapshot{Exists: true, Policy: policy}
		return nil
	})
	return snapshot, err
}

func (s *Store) ListPolicies() (PolicyList, error) {
	result := PolicyList{Canonical: false, Policies: []PolicySummary{}, TOPSID: s.topsID}
	err := s.withStateLock("profiles", false, func(directory *os.File) error {
		files, err := listStateFiles(directory, "profile", maxProfiles)
		if err != nil {
			return err
		}
		for _, file := range files {
			policy, err := decodePolicy(file.data)
			if err != nil {
				return err
			}
			if policy.TOPSID != s.topsID {
				return fmt.Errorf("validation policy TOPS identity mismatch")
			}
			result.Policies = append(result.Policies, PolicySummary{
				DefaultWarningDisposition: policy.DefaultWarningDisposition,
				Generation:                policy.Generation, PolicyDigest: policy.PolicyDigest,
				ProfileID: policy.ProfileID, RuleCount: len(policy.Rules),
			})
		}
		sort.Slice(result.Policies, func(i, j int) bool {
			return result.Policies[i].ProfileID < result.Policies[j].ProfileID
		})
		return nil
	})
	return result, err
}

func (s *Store) SetPolicy(config PolicyConfig, expected string, now time.Time) (Policy, bool, error) {
	if err := validatePolicyConfig(config); err != nil {
		return Policy{}, false, err
	}
	if expected != "absent" && !taggedDigest(expected) {
		return Policy{}, false, fmt.Errorf("expected policy digest must be absent or tagged SHA-256")
	}
	var result Policy
	changed := false
	err := s.withStateLock("profiles", true, func(directory *os.File) error {
		data, exists, err := readStateFile(directory, "profile", config.ProfileID)
		if err != nil {
			return err
		}
		var prior Policy
		if exists {
			prior, err = decodePolicy(data)
			if err != nil {
				return err
			}
		}
		actual := "absent"
		if exists {
			actual = prior.PolicyDigest
		}
		rules := append([]RulePolicy(nil), config.Rules...)
		sort.Slice(rules, func(i, j int) bool { return rules[i].RuleID < rules[j].RuleID })
		if exists && prior.DefaultWarningDisposition == config.DefaultWarningDisposition &&
			prior.HistoricalPresentation == config.HistoricalPresentation &&
			prior.NewPresentation == config.NewPresentation && policiesEqual(prior.Rules, rules) {
			retryMatches := expected == actual || prior.Generation == 1 && expected == "absent" ||
				prior.PreviousPolicyDigest != nil && *prior.PreviousPolicyDigest == expected
			if !retryMatches {
				return fmt.Errorf("validation policy compare-and-swap conflict: expected %s, actual %s", expected, actual)
			}
			result = prior
			return nil
		}
		if actual != expected {
			return fmt.Errorf("validation policy compare-and-swap conflict: expected %s, actual %s", expected, actual)
		}
		generation := uint64(1)
		var previous *string
		if exists {
			if prior.Generation >= 9007199254740991 {
				return fmt.Errorf("validation policy generation is exhausted")
			}
			generation = prior.Generation + 1
			previous = stringPointer(prior.PolicyDigest)
		}
		result = Policy{
			Protocol: PolicyProtocol, FormatVersion: 1, ProfileID: config.ProfileID,
			TOPSID: s.topsID, Generation: generation, PreviousPolicyDigest: previous,
			UpdatedAt:                 now.UTC().Truncate(time.Second).Format("2006-01-02T15:04:05Z"),
			DefaultWarningDisposition: config.DefaultWarningDisposition,
			HistoricalPresentation:    config.HistoricalPresentation,
			NewPresentation:           config.NewPresentation, Rules: rules, Canonical: false,
		}
		basis, err := objectWithout(result, "policy_digest")
		if err != nil {
			return err
		}
		result.PolicyDigest, err = digestValue(basis)
		if err != nil {
			return err
		}
		encoded, err := canonicalJSON(result)
		if err != nil {
			return err
		}
		if err := writeStateFile(directory, "profile", config.ProfileID, encoded); err != nil {
			return err
		}
		changed = true
		return nil
	})
	return result, changed, err
}

func (s *Store) RemovePolicy(profileID, expected string) (bool, error) {
	if !safeToken(profileID) || !taggedDigest(expected) {
		return false, fmt.Errorf("profile ID or expected policy digest is invalid")
	}
	changed := false
	err := s.withStateLock("profiles", true, func(directory *os.File) error {
		data, exists, err := readStateFile(directory, "profile", profileID)
		if err != nil {
			return err
		}
		if !exists {
			return nil
		}
		policy, err := decodePolicy(data)
		if err != nil {
			return err
		}
		if policy.PolicyDigest != expected {
			return fmt.Errorf("validation policy compare-and-swap conflict")
		}
		if err := removeStateFile(directory, "profile", profileID); err != nil {
			return err
		}
		changed = true
		return nil
	})
	return changed, err
}

func (s *Store) Baseline(baselineID string) (BaselineSnapshot, error) {
	if !safeToken(baselineID) {
		return BaselineSnapshot{}, fmt.Errorf("baseline ID has invalid syntax")
	}
	var snapshot BaselineSnapshot
	err := s.withStateLock("baselines", false, func(directory *os.File) error {
		data, exists, err := readStateFile(directory, "baseline", baselineID)
		if err != nil || !exists {
			snapshot.Exists = exists
			return err
		}
		baseline, err := decodeBaseline(data)
		if err != nil {
			return err
		}
		if baseline.BaselineID != baselineID || baseline.TOPSID != s.topsID {
			return fmt.Errorf("validation baseline storage identity mismatch")
		}
		snapshot = BaselineSnapshot{Exists: true, Baseline: baseline}
		return nil
	})
	return snapshot, err
}

func (s *Store) CreateBaseline(baselineID, expected string, result Result, now time.Time) (Baseline, bool, error) {
	if !safeToken(baselineID) || (expected != "absent" && !taggedDigest(expected)) {
		return Baseline{}, false, fmt.Errorf("baseline ID or expected digest is invalid")
	}
	occurrences, subjects := warningIDs(result)
	baseline := Baseline{
		Protocol: BaselineProtocol, FormatVersion: 1, BaselineID: baselineID, TOPSID: s.topsID,
		RepositoryIdentityDigest: result.Evidence.RepositoryIdentityDigest,
		ValidatorID:              result.Evidence.ValidatorID, ValidatorVersion: result.Evidence.ValidatorVersion,
		CreatedAt:      now.UTC().Truncate(time.Second).Format("2006-01-02T15:04:05Z"),
		EvidenceDigest: result.Evidence.EvidenceDigest, WarningOccurrenceIDs: occurrences,
		WarningSubjectIDs: subjects, Canonical: false,
	}
	basis, err := objectWithout(baseline, "baseline_digest")
	if err != nil {
		return Baseline{}, false, err
	}
	baseline.BaselineDigest, err = digestValue(basis)
	if err != nil {
		return Baseline{}, false, err
	}
	changed := false
	err = s.withStateLock("baselines", true, func(directory *os.File) error {
		data, exists, err := readStateFile(directory, "baseline", baselineID)
		if err != nil {
			return err
		}
		actual := "absent"
		if exists {
			prior, err := decodeBaseline(data)
			if err != nil {
				return err
			}
			actual = prior.BaselineDigest
			if prior.EvidenceDigest == baseline.EvidenceDigest && prior.RepositoryIdentityDigest == baseline.RepositoryIdentityDigest &&
				prior.ValidatorVersion == baseline.ValidatorVersion && sameStrings(prior.WarningOccurrenceIDs, baseline.WarningOccurrenceIDs) {
				baseline = prior
				if actual != expected && expected != "absent" {
					return fmt.Errorf("validation baseline compare-and-swap conflict")
				}
				return nil
			}
		}
		if actual != expected {
			return fmt.Errorf("validation baseline compare-and-swap conflict: expected %s, actual %s", expected, actual)
		}
		encoded, err := canonicalJSON(baseline)
		if err != nil {
			return err
		}
		if err := writeStateFile(directory, "baseline", baselineID, encoded); err != nil {
			return err
		}
		changed = true
		return nil
	})
	return baseline, changed, err
}

func (s *Store) RemoveBaseline(baselineID, expected string) (bool, error) {
	if !safeToken(baselineID) || !taggedDigest(expected) {
		return false, fmt.Errorf("baseline ID or expected baseline digest is invalid")
	}
	changed := false
	err := s.withStateLock("baselines", true, func(directory *os.File) error {
		data, exists, err := readStateFile(directory, "baseline", baselineID)
		if err != nil {
			return err
		}
		if !exists {
			return nil
		}
		baseline, err := decodeBaseline(data)
		if err != nil {
			return err
		}
		if baseline.BaselineDigest != expected {
			return fmt.Errorf("validation baseline compare-and-swap conflict")
		}
		if err := removeStateFile(directory, "baseline", baselineID); err != nil {
			return err
		}
		changed = true
		return nil
	})
	return changed, err
}

func decodePolicy(data []byte) (Policy, error) {
	var policy Policy
	if err := decodeExact(data, &policy); err != nil {
		return Policy{}, fmt.Errorf("decode validation policy: %w", err)
	}
	if policy.Protocol != PolicyProtocol || policy.FormatVersion != 1 || policy.Canonical ||
		!safeToken(policy.ProfileID) || !validTOPSID(policy.TOPSID) || policy.Generation == 0 ||
		policy.Generation > 9007199254740991 ||
		!exactUTCSeconds(policy.UpdatedAt) || !validDisposition(policy.DefaultWarningDisposition) ||
		!validPresentation(policy.HistoricalPresentation) ||
		(policy.NewPresentation != "full" && policy.NewPresentation != "summary") ||
		!taggedDigest(policy.PolicyDigest) || len(policy.Rules) > 2048 {
		return Policy{}, fmt.Errorf("validation policy identity or bound is invalid")
	}
	if policy.Generation == 1 && policy.PreviousPolicyDigest != nil ||
		policy.Generation > 1 && (policy.PreviousPolicyDigest == nil || !taggedDigest(*policy.PreviousPolicyDigest)) {
		return Policy{}, fmt.Errorf("validation policy lineage is invalid")
	}
	prior := ""
	for _, rule := range policy.Rules {
		if !safeToken(rule.RuleID) || !validDisposition(rule.Disposition) || !validPresentation(rule.Presentation) ||
			(prior != "" && prior >= rule.RuleID) {
			return Policy{}, fmt.Errorf("validation policy rule ordering or value is invalid")
		}
		prior = rule.RuleID
	}
	basis, err := objectWithout(policy, "policy_digest")
	if err != nil {
		return Policy{}, err
	}
	expected, err := digestValue(basis)
	if err != nil || expected != policy.PolicyDigest {
		return Policy{}, fmt.Errorf("validation policy digest mismatch")
	}
	return policy, nil
}

func decodeBaseline(data []byte) (Baseline, error) {
	var baseline Baseline
	if err := decodeExact(data, &baseline); err != nil {
		return Baseline{}, fmt.Errorf("decode validation baseline: %w", err)
	}
	if baseline.Protocol != BaselineProtocol || baseline.FormatVersion != 1 || baseline.Canonical ||
		!safeToken(baseline.BaselineID) || !validTOPSID(baseline.TOPSID) ||
		baseline.ValidatorID != "symphony-validator" || !safeVersion(baseline.ValidatorVersion) ||
		!exactUTCSeconds(baseline.CreatedAt) || !taggedDigest(baseline.RepositoryIdentityDigest) ||
		!taggedDigest(baseline.EvidenceDigest) || !taggedDigest(baseline.BaselineDigest) {
		return Baseline{}, fmt.Errorf("validation baseline identity is invalid")
	}
	if len(baseline.WarningOccurrenceIDs) > 16384 || len(baseline.WarningSubjectIDs) > 16384 {
		return Baseline{}, fmt.Errorf("validation baseline warning collection exceeds its bound")
	}
	occurrences, err := uniqueSorted(baseline.WarningOccurrenceIDs)
	if err != nil || !sameStrings(occurrences, baseline.WarningOccurrenceIDs) {
		return Baseline{}, fmt.Errorf("validation baseline occurrence collection is invalid")
	}
	subjects, err := uniqueSorted(baseline.WarningSubjectIDs)
	if err != nil || !sameStrings(subjects, baseline.WarningSubjectIDs) {
		return Baseline{}, fmt.Errorf("validation baseline subject collection is invalid")
	}
	basis, err := objectWithout(baseline, "baseline_digest")
	if err != nil {
		return Baseline{}, err
	}
	expected, err := digestValue(basis)
	if err != nil || expected != baseline.BaselineDigest {
		return Baseline{}, fmt.Errorf("validation baseline digest mismatch")
	}
	return baseline, nil
}

func decodeExact(data []byte, target any) error {
	if err := knowledgeengine.ValidateJSONObject(data, maxStateBytes); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if decoder.Decode(&struct{}{}) == nil {
		return fmt.Errorf("trailing JSON value")
	}
	return nil
}

func validatePolicyConfig(config PolicyConfig) error {
	if !safeToken(config.ProfileID) || !validDisposition(config.DefaultWarningDisposition) ||
		!validPresentation(config.HistoricalPresentation) ||
		(config.NewPresentation != "full" && config.NewPresentation != "summary") || len(config.Rules) > 2048 {
		return fmt.Errorf("validation policy configuration is invalid")
	}
	seen := map[string]bool{}
	for _, rule := range config.Rules {
		if !safeToken(rule.RuleID) || !validDisposition(rule.Disposition) ||
			!validPresentation(rule.Presentation) || seen[rule.RuleID] {
			return fmt.Errorf("validation policy rule is invalid or duplicated")
		}
		seen[rule.RuleID] = true
	}
	return nil
}

func validDisposition(value string) bool {
	return value == "record" || value == "review" || value == "require"
}
func validPresentation(value string) bool {
	return value == "full" || value == "summary" || value == "count"
}
func policiesEqual(left, right []RulePolicy) bool {
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
