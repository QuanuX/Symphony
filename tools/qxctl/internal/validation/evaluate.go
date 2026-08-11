package validation

import (
	"fmt"
	"sort"
)

type DisplayFilter struct {
	Delta    string
	Path     string
	RecordID string
	RuleID   string
}

func Evaluate(raw Result, policy *Policy, baseline *Baseline, filter DisplayFilter) (Projection, error) {
	if policy != nil && policy.TOPSID == "" {
		return Projection{}, fmt.Errorf("validation policy identity is incomplete")
	}
	if baseline != nil {
		if baseline.RepositoryIdentityDigest != raw.Evidence.RepositoryIdentityDigest {
			return Projection{}, fmt.Errorf("validation baseline repository identity mismatch")
		}
		if baseline.ValidatorID != raw.Evidence.ValidatorID || baseline.ValidatorVersion != raw.Evidence.ValidatorVersion {
			return Projection{}, fmt.Errorf("validation baseline validator identity mismatch")
		}
	}
	if filter.Delta != "" && filter.Delta != "new" && filter.Delta != "unchanged" && filter.Delta != "resolved" {
		return Projection{}, fmt.Errorf("debug delta must be new, unchanged, or resolved")
	}
	baselineSet := map[string]bool{}
	if baseline != nil {
		for _, id := range baseline.WarningOccurrenceIDs {
			baselineSet[id] = true
		}
	}
	currentSet := map[string]bool{}
	newIDs := []string{}
	unchangedIDs := []string{}
	for _, finding := range raw.Evidence.Findings {
		if finding.Category != "warning" {
			continue
		}
		currentSet[finding.OccurrenceID] = true
		if baselineSet[finding.OccurrenceID] {
			unchangedIDs = append(unchangedIDs, finding.OccurrenceID)
		} else {
			newIDs = append(newIDs, finding.OccurrenceID)
		}
	}
	resolvedIDs := []string{}
	for id := range baselineSet {
		if !currentSet[id] {
			resolvedIDs = append(resolvedIDs, id)
		}
	}
	sort.Strings(newIDs)
	sort.Strings(unchangedIDs)
	sort.Strings(resolvedIDs)
	newSet := makeSet(newIDs)
	unchangedSet := makeSet(unchangedIDs)

	defaultDisposition := "record"
	historicalPresentation := "summary"
	newPresentation := "full"
	var profileID, policyDigest *string
	rules := map[string]RulePolicy{}
	if policy != nil {
		defaultDisposition = policy.DefaultWarningDisposition
		historicalPresentation = policy.HistoricalPresentation
		newPresentation = policy.NewPresentation
		profileID = stringPointer(policy.ProfileID)
		policyDigest = stringPointer(policy.PolicyDigest)
		for _, rule := range policy.Rules {
			rules[rule.RuleID] = rule
		}
	}
	var baselineID, baselineDigest *string
	if baseline != nil {
		baselineID = stringPointer(baseline.BaselineID)
		baselineDigest = stringPointer(baseline.BaselineDigest)
	}
	reviewRules := map[string]bool{}
	requiredRules := map[string]bool{}
	displayed := []Finding{}
	displayedIDs := []string{}
	summarySeen := map[string]bool{}
	for _, finding := range raw.Evidence.Findings {
		if finding.Category == "pass" && filter == (DisplayFilter{}) {
			continue
		}
		if !matchesFilter(finding, filter, newSet, unchangedSet) {
			continue
		}
		presentation := "full"
		if finding.Category == "warning" {
			disposition := defaultDisposition
			if rule, ok := rules[finding.RuleID]; ok {
				disposition = rule.Disposition
				presentation = rule.Presentation
			} else if unchangedSet[finding.OccurrenceID] {
				presentation = historicalPresentation
			} else {
				presentation = newPresentation
			}
			if newSet[finding.OccurrenceID] {
				switch disposition {
				case "review":
					reviewRules[finding.RuleID] = true
				case "require":
					requiredRules[finding.RuleID] = true
				}
			}
		}
		if presentation == "count" {
			continue
		}
		if presentation == "summary" {
			key := finding.RuleID + "\n" + finding.SubjectID
			if summarySeen[key] {
				continue
			}
			summarySeen[key] = true
		}
		displayed = append(displayed, finding)
		displayedIDs = append(displayedIDs, finding.OccurrenceID)
	}
	review := sortedKeys(reviewRules)
	required := sortedKeys(requiredRules)
	outcome := "pass"
	if raw.Evidence.Summary.Violation != 0 || len(required) != 0 {
		outcome = "failed"
	} else if len(review) != 0 {
		outcome = "review_required"
	}
	evaluation := Evaluation{
		BaselineDigest: baselineDigest, BaselineID: baselineID,
		DisplayedOccurrenceIDs: displayedIDs, NewWarningOccurrenceIDs: newIDs,
		Outcome: outcome, PolicyDigest: policyDigest, ProfileID: profileID,
		RequiredRuleIDs: required, ResolvedWarningOccurrenceIDs: resolvedIDs,
		ReviewRuleIDs: review, UnchangedWarningOccurrenceIDs: unchangedIDs,
	}
	basis, err := objectWithout(evaluation, "evaluation_digest")
	if err != nil {
		return Projection{}, err
	}
	evaluation.EvaluationDigest, err = digestValue(basis)
	if err != nil {
		return Projection{}, err
	}
	projected := raw
	projected.Evaluation = &evaluation
	resultBasis, err := objectWithout(projected, "result_digest")
	if err != nil {
		return Projection{}, err
	}
	projected.ResultDigest, err = digestValue(resultBasis)
	if err != nil {
		return Projection{}, err
	}
	encoded, err := canonicalJSON(projected)
	if err != nil {
		return Projection{}, err
	}
	return Projection{Result: projected, RawJSON: encoded, Displayed: displayed}, nil
}

func matchesFilter(finding Finding, filter DisplayFilter, newSet, unchangedSet map[string]bool) bool {
	if filter.RuleID != "" && finding.RuleID != filter.RuleID {
		return false
	}
	if filter.RecordID != "" && finding.Attributes["record_id"] != filter.RecordID {
		return false
	}
	if filter.Path != "" && finding.Attributes["path"] != filter.Path {
		return false
	}
	switch filter.Delta {
	case "new":
		return newSet[finding.OccurrenceID]
	case "unchanged":
		return unchangedSet[finding.OccurrenceID]
	case "resolved":
		return false // resolved identities have no current finding body
	default:
		return true
	}
}

func makeSet(values []string) map[string]bool {
	result := make(map[string]bool, len(values))
	for _, value := range values {
		result[value] = true
	}
	return result
}

func sortedKeys(values map[string]bool) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
