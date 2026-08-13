package validation

import (
	"fmt"
	"sort"
	"time"
)

type DisplayFilter struct {
	Classification string
	Delta          string
	Path           string
	RecordID       string
	RuleID         string
	SubjectID      string
}

func Evaluate(raw Result, policy *Policy, baseline *Baseline, filter DisplayFilter) (Projection, error) {
	return EvaluateWithWarningState(raw, policy, baseline, nil, filter, time.Time{}, true)
}

// EvaluateWithWarningState keeps raw evidence and v1 policy evaluation exact,
// while limiting ordinary presentation to actionable open warning subjects.
// Debug presentation may inspect accepted, superseded, muted, and resolved
// history without changing detector execution or its result.
func EvaluateWithWarningState(
	raw Result, policy *Policy, baseline *Baseline, warningState *WarningState,
	filter DisplayFilter, now time.Time, debug bool,
) (Projection, error) {
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
	if filter.Classification != "" && filter.Classification != "open" && filter.Classification != "accepted" &&
		filter.Classification != "resolved" && filter.Classification != "superseded" && filter.Classification != "muted" {
		return Projection{}, fmt.Errorf("debug classification must be open, accepted, resolved, superseded, or muted")
	}
	if warningState != nil {
		if warningState.RepositoryIdentityDigest != raw.Evidence.RepositoryIdentityDigest {
			return Projection{}, fmt.Errorf("validation warning state repository identity mismatch")
		}
		if warningState.ValidatorID != raw.Evidence.ValidatorID || warningState.ValidatorVersion != raw.Evidence.ValidatorVersion {
			return Projection{}, fmt.Errorf("validation warning state validator identity mismatch")
		}
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
	classifications := map[string]string{}
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
			classification, muted := currentWarningClassification(warningState, finding.SubjectID, now)
			classifications[finding.OccurrenceID] = classification
			disposition := defaultDisposition
			if rule, ok := rules[finding.RuleID]; ok {
				disposition = rule.Disposition
				presentation = rule.Presentation
			} else if unchangedSet[finding.OccurrenceID] {
				presentation = historicalPresentation
			} else {
				presentation = newPresentation
			}
			// Muting is presentation-only and therefore cannot alter a policy
			// gate. Accepted/superseded/resolved subjects are nonactionable;
			// expired acceptance is projected as open immediately.
			if classification == "open" && newSet[finding.OccurrenceID] {
				switch disposition {
				case "review":
					reviewRules[finding.RuleID] = true
				case "require":
					requiredRules[finding.RuleID] = true
				}
			}
			if !debug && (classification != "open" || muted) {
				continue
			}
			if debug && !matchesLifecycleFilter(classification, muted, finding.SubjectID, filter) {
				continue
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
	historical := historicalWarningEvidence(warningState, filter, debug)
	return Projection{
		Result: projected, RawJSON: encoded, Displayed: displayed, Historical: historical,
		WarningClassifications: classifications,
	}, nil
}

func currentWarningClassification(state *WarningState, subjectID string, now time.Time) (string, bool) {
	if state == nil {
		return "open", false
	}
	index := sort.Search(len(state.Subjects), func(i int) bool { return state.Subjects[i].SubjectID >= subjectID })
	if index == len(state.Subjects) || state.Subjects[index].SubjectID != subjectID {
		return "open", false
	}
	subject := state.Subjects[index]
	classification := subject.Classification
	if classification == "accepted" && subject.ValidUntil != nil && !now.IsZero() &&
		!now.UTC().Before(mustParseUTC(*subject.ValidUntil)) {
		classification = "open"
	}
	return classification, subject.Muted
}

func matchesLifecycleFilter(classification string, muted bool, subjectID string, filter DisplayFilter) bool {
	if filter.SubjectID != "" && filter.SubjectID != subjectID {
		return false
	}
	if filter.Classification == "muted" {
		return muted
	}
	return filter.Classification == "" || filter.Classification == classification
}

func historicalWarningEvidence(state *WarningState, filter DisplayFilter, debug bool) []WarningOccurrence {
	if !debug || state == nil || filter.Classification != "resolved" && filter.Delta != "resolved" {
		return []WarningOccurrence{}
	}
	result := []WarningOccurrence{}
	for _, subject := range state.Subjects {
		if subject.Classification != "resolved" || filter.SubjectID != "" && filter.SubjectID != subject.SubjectID {
			continue
		}
		for _, occurrence := range subject.Occurrences {
			finding := occurrence.Finding
			if filter.RuleID != "" && finding.RuleID != filter.RuleID ||
				filter.RecordID != "" && finding.Attributes["record_id"] != filter.RecordID ||
				filter.Path != "" && finding.Attributes["path"] != filter.Path {
				continue
			}
			result = append(result, occurrence)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].OccurrenceID < result[j].OccurrenceID })
	return result
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
