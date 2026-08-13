package validation

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

const (
	maxWarningStates      = 256
	maxWarningSubjects    = 4096
	maxWarningOccurrences = 16384
	maxWarningTransitions = 32768
)

func (s *Store) WarningState(stateID string) (WarningStateSnapshot, error) {
	if !safeToken(stateID) {
		return WarningStateSnapshot{}, fmt.Errorf("warning state ID has invalid syntax")
	}
	var snapshot WarningStateSnapshot
	err := s.withStateLock("warnings", false, func(directory *os.File) error {
		data, exists, err := readStateFile(directory, "warning-state", stateID)
		if err != nil || !exists {
			snapshot.Exists = exists
			return err
		}
		state, err := decodeWarningState(data)
		if err != nil {
			return err
		}
		if state.StateID != stateID || state.TOPSID != s.topsID {
			return fmt.Errorf("validation warning state storage identity mismatch")
		}
		snapshot = WarningStateSnapshot{Exists: true, State: state}
		return nil
	})
	return snapshot, err
}

func (s *Store) ListWarningStates() (WarningStateList, error) {
	result := WarningStateList{Canonical: false, States: []WarningStateSummary{}, TOPSID: s.topsID}
	err := s.withStateLock("warnings", false, func(directory *os.File) error {
		files, err := listStateFiles(directory, "warning-state", maxWarningStates)
		if err != nil {
			return err
		}
		for _, file := range files {
			state, err := decodeWarningState(file.data)
			if err != nil {
				return err
			}
			if state.TOPSID != s.topsID {
				return fmt.Errorf("validation warning state TOPS identity mismatch")
			}
			summary := WarningStateSummary{
				Generation: state.Generation, StateDigest: state.StateDigest, StateID: state.StateID,
			}
			for _, subject := range state.Subjects {
				switch subject.Classification {
				case "open":
					summary.Open++
				case "accepted":
					summary.Accepted++
				case "resolved":
					summary.Resolved++
				case "superseded":
					summary.Superseded++
				}
				if subject.Muted {
					summary.Muted++
				}
			}
			result.States = append(result.States, summary)
		}
		sort.Slice(result.States, func(i, j int) bool { return result.States[i].StateID < result.States[j].StateID })
		return nil
	})
	return result, err
}

// SyncWarningState observes every warning in an already complete raw result.
// It never changes, removes, or reclassifies the embedded validator evidence.
func (s *Store) SyncWarningState(
	stateID, expected string, result Result, now time.Time,
) (WarningState, bool, error) {
	if !safeToken(stateID) || (expected != "absent" && !taggedDigest(expected)) {
		return WarningState{}, false, fmt.Errorf("warning state ID or expected digest is invalid")
	}
	if result.Protocol != ResultProtocol || result.FormatVersion != 1 || result.Evaluation != nil ||
		!taggedDigest(result.Evidence.EvidenceDigest) || !taggedDigest(result.Evidence.RepositoryIdentityDigest) ||
		result.Evidence.ValidatorID != "symphony-validator" || !safeVersion(result.Evidence.ValidatorVersion) {
		return WarningState{}, false, fmt.Errorf("raw validation result identity is invalid")
	}
	encodedResult, err := canonicalJSON(result)
	if err != nil {
		return WarningState{}, false, err
	}
	result, err = decodeResult(encodedResult)
	if err != nil {
		return WarningState{}, false, fmt.Errorf("raw validation result failed immutable evidence verification: %w", err)
	}
	observedAt := now.UTC().Truncate(time.Second)
	var state WarningState
	changed := false
	err = s.withStateLock("warnings", true, func(directory *os.File) error {
		data, exists, err := readStateFile(directory, "warning-state", stateID)
		if err != nil {
			return err
		}
		var prior WarningState
		actual := "absent"
		if exists {
			prior, err = decodeWarningState(data)
			if err != nil {
				return err
			}
			if prior.StateID != stateID || prior.TOPSID != s.topsID {
				return fmt.Errorf("validation warning state storage identity mismatch")
			}
			actual = prior.StateDigest
			if prior.RepositoryIdentityDigest != result.Evidence.RepositoryIdentityDigest ||
				prior.ValidatorID != result.Evidence.ValidatorID || prior.ValidatorVersion != result.Evidence.ValidatorVersion {
				return fmt.Errorf("validation warning state repository or validator identity mismatch")
			}
			if prior.LastEvidenceDigest == result.Evidence.EvidenceDigest && !hasExpiredAcceptance(prior, observedAt) {
				if !mutationRetryMatches(prior, expected) {
					return fmt.Errorf("validation warning state compare-and-swap conflict: expected %s, actual %s", expected, actual)
				}
				state = prior
				return nil
			}
		}
		if actual != expected {
			return fmt.Errorf("validation warning state compare-and-swap conflict: expected %s, actual %s", expected, actual)
		}
		state, err = nextWarningState(prior, exists, stateID, s.topsID, result, observedAt)
		if err != nil {
			return err
		}
		encoded, err := canonicalJSON(state)
		if err != nil {
			return err
		}
		if err := writeStateFile(directory, "warning-state", stateID, encoded); err != nil {
			return err
		}
		changed = true
		return nil
	})
	return state, changed, err
}

func nextWarningState(
	prior WarningState, exists bool, stateID, topsID string, result Result, now time.Time,
) (WarningState, error) {
	stamp := now.Format("2006-01-02T15:04:05Z")
	state := prior
	if !exists {
		state = WarningState{
			Canonical: false, FormatVersion: 1, Generation: 1, Protocol: WarningStateProtocol,
			RepositoryIdentityDigest: result.Evidence.RepositoryIdentityDigest,
			StateID:                  stateID, Subjects: []WarningSubject{}, TOPSID: topsID,
			Transitions: []WarningTransition{}, UpdatedAt: stamp,
			ValidatorID: result.Evidence.ValidatorID, ValidatorVersion: result.Evidence.ValidatorVersion,
		}
	} else {
		if state.Generation >= 9007199254740991 {
			return WarningState{}, fmt.Errorf("validation warning state generation is exhausted")
		}
		state.Generation++
		state.PreviousStateDigest = stringPointer(prior.StateDigest)
		state.UpdatedAt = stamp
	}
	state.StateDigest = ""
	state.LastEvidenceDigest = result.Evidence.EvidenceDigest

	bySubject := make(map[string]int, len(state.Subjects))
	for index := range state.Subjects {
		bySubject[state.Subjects[index].SubjectID] = index
		for occurrenceIndex := range state.Subjects[index].Occurrences {
			state.Subjects[index].Occurrences[occurrenceIndex].Active = false
		}
	}
	newSubjects := []string{}
	knownNewOccurrences := []string{}
	unchangedOccurrences := []string{}
	observedOccurrences := []string{}
	reopenedSubjects := []string{}
	currentSubjects := map[string]bool{}
	findings := make([]Finding, 0)
	for _, finding := range result.Evidence.Findings {
		if finding.Category == "warning" {
			findings = append(findings, finding)
		}
	}
	sort.Slice(findings, func(i, j int) bool { return findings[i].OccurrenceID < findings[j].OccurrenceID })
	for _, finding := range findings {
		observedOccurrences = append(observedOccurrences, finding.OccurrenceID)
		currentSubjects[finding.SubjectID] = true
		index, known := bySubject[finding.SubjectID]
		if !known {
			state.Subjects = append(state.Subjects, WarningSubject{
				Classification: "open", FirstObservedAt: stamp, LastObservedAt: stamp,
				OccurrenceIDs: []string{}, Occurrences: []WarningOccurrence{}, RuleID: finding.RuleID,
				SubjectID: finding.SubjectID,
			})
			index = len(state.Subjects) - 1
			bySubject[finding.SubjectID] = index
			newSubjects = append(newSubjects, finding.SubjectID)
			if err := appendWarningTransition(&state, &state.Subjects[index], "detected", nil, &finding.OccurrenceID, "", "", stamp); err != nil {
				return WarningState{}, err
			}
		}
		subject := &state.Subjects[index]
		if subject.RuleID != finding.RuleID {
			return WarningState{}, fmt.Errorf("stable warning subject changed rule identity")
		}
		if subject.Classification == "resolved" {
			from := subject.Classification
			subject.Classification = "open"
			subject.Rationale, subject.ValidUntil, subject.SupersededBySubjectID = nil, nil, nil
			reopenedSubjects = append(reopenedSubjects, subject.SubjectID)
			if err := appendWarningTransition(&state, subject, "recurred", &from, &finding.OccurrenceID, "", "", stamp); err != nil {
				return WarningState{}, err
			}
		}
		if subject.Classification == "accepted" && subject.ValidUntil != nil && !now.Before(mustParseUTC(*subject.ValidUntil)) {
			from := subject.Classification
			subject.Classification = "open"
			subject.Rationale, subject.ValidUntil = nil, nil
			if err := appendWarningTransition(&state, subject, "acceptance_expired", &from, &finding.OccurrenceID, "", "", stamp); err != nil {
				return WarningState{}, err
			}
		}
		subject.LastObservedAt = stamp
		occurrenceIndex := sort.Search(len(subject.Occurrences), func(i int) bool {
			return subject.Occurrences[i].OccurrenceID >= finding.OccurrenceID
		})
		if occurrenceIndex == len(subject.Occurrences) || subject.Occurrences[occurrenceIndex].OccurrenceID != finding.OccurrenceID {
			occurrence := WarningOccurrence{
				Active: true, EvidenceDigests: []string{result.Evidence.EvidenceDigest}, Finding: finding,
				FirstObservedAt: stamp, LastObservedAt: stamp, OccurrenceID: finding.OccurrenceID,
			}
			subject.Occurrences = append(subject.Occurrences, WarningOccurrence{})
			copy(subject.Occurrences[occurrenceIndex+1:], subject.Occurrences[occurrenceIndex:])
			subject.Occurrences[occurrenceIndex] = occurrence
			subject.OccurrenceIDs = insertSorted(subject.OccurrenceIDs, finding.OccurrenceID)
			if known {
				knownNewOccurrences = append(knownNewOccurrences, finding.OccurrenceID)
				if err := appendWarningTransition(&state, subject, "occurrence_observed", nil, &finding.OccurrenceID, "", "", stamp); err != nil {
					return WarningState{}, err
				}
			}
		} else {
			occurrence := &subject.Occurrences[occurrenceIndex]
			occurrence.Active = true
			occurrence.LastObservedAt = stamp
			occurrence.Finding = finding
			if !containsString(occurrence.EvidenceDigests, result.Evidence.EvidenceDigest) {
				occurrence.EvidenceDigests = append(occurrence.EvidenceDigests, result.Evidence.EvidenceDigest)
			}
			unchangedOccurrences = append(unchangedOccurrences, finding.OccurrenceID)
		}
	}

	resolvedSubjects := []string{}
	for index := range state.Subjects {
		subject := &state.Subjects[index]
		if currentSubjects[subject.SubjectID] || subject.Classification == "resolved" || subject.Classification == "superseded" {
			continue
		}
		from := subject.Classification
		subject.Classification = "resolved"
		subject.Rationale, subject.ValidUntil, subject.SupersededBySubjectID = nil, nil, nil
		resolvedSubjects = append(resolvedSubjects, subject.SubjectID)
		if err := appendWarningTransition(&state, subject, "not_observed", &from, nil, "", "", stamp); err != nil {
			return WarningState{}, err
		}
	}
	sort.Slice(state.Subjects, func(i, j int) bool { return state.Subjects[i].SubjectID < state.Subjects[j].SubjectID })
	if len(state.Subjects) > maxWarningSubjects || totalWarningOccurrences(state.Subjects) > maxWarningOccurrences ||
		len(state.Transitions) > maxWarningTransitions {
		return WarningState{}, fmt.Errorf("validation warning lifecycle exceeds its bounded history")
	}
	state.LastSync = WarningSync{
		EvidenceDigest:               result.Evidence.EvidenceDigest,
		KnownSubjectNewOccurrenceIDs: sortedStrings(knownNewOccurrences), NewSubjectIDs: sortedStrings(newSubjects),
		ObservedOccurrenceIDs: sortedStrings(observedOccurrences), ReopenedSubjectIDs: sortedStrings(reopenedSubjects),
		ResolvedSubjectIDs: sortedStrings(resolvedSubjects), UnchangedOccurrenceIDs: sortedStrings(unchangedOccurrences),
	}
	basis, err := objectWithout(state.LastSync, "sync_digest")
	if err != nil {
		return WarningState{}, err
	}
	state.LastSync.SyncDigest, err = digestValue(basis)
	if err != nil {
		return WarningState{}, err
	}
	if err := setWarningStateDigest(&state); err != nil {
		return WarningState{}, err
	}
	return state, nil
}

func (s *Store) MutateWarning(mutation WarningMutation, now time.Time) (WarningState, bool, error) {
	if !safeToken(mutation.StateID) || !taggedDigest(mutation.ExpectedStateDigest) || !taggedDigest(mutation.SubjectID) {
		return WarningState{}, false, fmt.Errorf("warning mutation identity or expected digest is invalid")
	}
	stamp := now.UTC().Truncate(time.Second).Format("2006-01-02T15:04:05Z")
	var state WarningState
	changed := false
	err := s.withStateLock("warnings", true, func(directory *os.File) error {
		data, exists, err := readStateFile(directory, "warning-state", mutation.StateID)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("validation warning state %q is absent", mutation.StateID)
		}
		state, err = decodeWarningState(data)
		if err != nil {
			return err
		}
		if state.TOPSID != s.topsID || state.StateID != mutation.StateID {
			return fmt.Errorf("validation warning state storage identity mismatch")
		}
		index := sort.Search(len(state.Subjects), func(i int) bool { return state.Subjects[i].SubjectID >= mutation.SubjectID })
		if index == len(state.Subjects) || state.Subjects[index].SubjectID != mutation.SubjectID {
			return fmt.Errorf("warning subject %q is unknown", mutation.SubjectID)
		}
		subject := &state.Subjects[index]
		semanticNoop, err := validateAndApplyWarningMutation(&state, subject, mutation, now, stamp)
		if err != nil {
			return err
		}
		if semanticNoop {
			if mutation.ExpectedStateDigest != state.StateDigest &&
				!warningMutationRetryMatches(state, mutation, mutation.ExpectedStateDigest) {
				return fmt.Errorf("validation warning state compare-and-swap conflict")
			}
			return nil
		}
		if state.StateDigest != mutation.ExpectedStateDigest {
			return fmt.Errorf("validation warning state compare-and-swap conflict: expected %s, actual %s", mutation.ExpectedStateDigest, state.StateDigest)
		}
		if state.Generation >= 9007199254740991 {
			return fmt.Errorf("validation warning state generation is exhausted")
		}
		previous := state.StateDigest
		state.Generation++
		state.PreviousStateDigest = &previous
		state.UpdatedAt = stamp
		state.StateDigest = ""
		if len(state.Transitions) > maxWarningTransitions {
			return fmt.Errorf("validation warning transition history exceeds its bound")
		}
		if err := setWarningStateDigest(&state); err != nil {
			return err
		}
		encoded, err := canonicalJSON(state)
		if err != nil {
			return err
		}
		if err := writeStateFile(directory, "warning-state", mutation.StateID, encoded); err != nil {
			return err
		}
		changed = true
		return nil
	})
	return state, changed, err
}

func validateAndApplyWarningMutation(
	state *WarningState, subject *WarningSubject, mutation WarningMutation, now time.Time, stamp string,
) (bool, error) {
	if mutation.Muted != nil {
		if mutation.Classification != "" || mutation.ValidUntil != "" || mutation.SupersededBySubjectID != "" {
			return false, fmt.Errorf("presentation mutation cannot also change classification")
		}
		if strings.TrimSpace(mutation.Rationale) == "" || len(mutation.Rationale) > 4096 {
			return false, fmt.Errorf("warning presentation mutation requires a bounded rationale")
		}
		if subject.Muted == *mutation.Muted {
			return true, nil
		}
		subject.Muted = *mutation.Muted
		operation := "muted"
		if !*mutation.Muted {
			operation = "unmuted"
		}
		return false, appendWarningTransition(state, subject, operation, nil, nil, mutation.Rationale, "", stamp)
	}
	if mutation.Classification != "open" && mutation.Classification != "accepted" && mutation.Classification != "superseded" {
		return false, fmt.Errorf("administrative classification must be open, accepted, or superseded")
	}
	if subject.Classification == "resolved" {
		return false, fmt.Errorf("resolved warning subjects reopen only through complete detector evidence")
	}
	if strings.TrimSpace(mutation.Rationale) == "" || len(mutation.Rationale) > 4096 {
		return false, fmt.Errorf("warning classification mutation requires a bounded rationale")
	}
	if mutation.Classification == "accepted" {
		if mutation.SupersededBySubjectID != "" {
			return false, fmt.Errorf("accepted warning cannot name a superseding subject")
		}
		var expiry *string
		if mutation.ValidUntil != "" {
			if !exactUTCSeconds(mutation.ValidUntil) || !mustParseUTC(mutation.ValidUntil).After(now.UTC()) {
				return false, fmt.Errorf("warning acceptance expiry must be a future STSC whole-second UTC timestamp")
			}
			expiry = stringPointer(mutation.ValidUntil)
		}
		if subject.Classification == "accepted" && pointerStringEqual(subject.ValidUntil, expiry) &&
			subject.Rationale != nil && *subject.Rationale == mutation.Rationale {
			return true, nil
		}
		from := subject.Classification
		subject.Classification = "accepted"
		subject.Rationale = stringPointer(mutation.Rationale)
		subject.ValidUntil = expiry
		subject.SupersededBySubjectID = nil
		return false, appendWarningTransition(state, subject, "accepted", &from, nil, mutation.Rationale, mutation.ValidUntil, stamp)
	}
	if mutation.Classification == "superseded" {
		if !taggedDigest(mutation.SupersededBySubjectID) || mutation.SupersededBySubjectID == subject.SubjectID {
			return false, fmt.Errorf("superseded warning requires a distinct known subject ID")
		}
		targetIndex := sort.Search(len(state.Subjects), func(i int) bool {
			return state.Subjects[i].SubjectID >= mutation.SupersededBySubjectID
		})
		if targetIndex == len(state.Subjects) || state.Subjects[targetIndex].SubjectID != mutation.SupersededBySubjectID {
			return false, fmt.Errorf("superseding warning subject is unknown")
		}
		if warningSupersessionCycle(*state, subject.SubjectID, mutation.SupersededBySubjectID) {
			return false, fmt.Errorf("warning supersession would create a cycle")
		}
		if subject.Classification == "superseded" && subject.SupersededBySubjectID != nil &&
			*subject.SupersededBySubjectID == mutation.SupersededBySubjectID && subject.Rationale != nil &&
			*subject.Rationale == mutation.Rationale {
			return true, nil
		}
		from := subject.Classification
		subject.Classification = "superseded"
		subject.Rationale = stringPointer(mutation.Rationale)
		subject.SupersededBySubjectID = stringPointer(mutation.SupersededBySubjectID)
		subject.ValidUntil = nil
		return false, appendWarningTransition(state, subject, "superseded", &from, nil, mutation.Rationale, "", stamp)
	}
	if mutation.ValidUntil != "" || mutation.SupersededBySubjectID != "" {
		return false, fmt.Errorf("open warning cannot carry acceptance or supersession fields")
	}
	if subject.Classification == "open" && subject.Rationale == nil {
		return true, nil
	}
	from := subject.Classification
	subject.Classification = "open"
	subject.Rationale, subject.ValidUntil, subject.SupersededBySubjectID = nil, nil, nil
	return false, appendWarningTransition(state, subject, "reopened", &from, nil, mutation.Rationale, "", stamp)
}

func appendWarningTransition(
	state *WarningState, subject *WarningSubject, operation string, from *string, occurrence *string,
	rationale, validUntil, stamp string,
) error {
	transition := WarningTransition{
		At: stamp, FromClassification: from, Operation: operation, OccurrenceID: occurrence,
		Sequence: uint64(len(state.Transitions) + 1), SubjectID: subject.SubjectID,
		SupersededBySubjectID: subject.SupersededBySubjectID, ToClassification: subject.Classification,
		ValidUntil: subject.ValidUntil,
	}
	if rationale != "" {
		transition.Rationale = stringPointer(rationale)
	}
	if validUntil != "" {
		transition.ValidUntil = stringPointer(validUntil)
	}
	if len(state.Transitions) != 0 {
		transition.PreviousTransitionDigest = stringPointer(state.Transitions[len(state.Transitions)-1].TransitionDigest)
	}
	basis, err := objectWithout(transition, "transition_digest")
	if err != nil {
		return err
	}
	transition.TransitionDigest, err = digestValue(basis)
	if err != nil {
		return err
	}
	state.Transitions = append(state.Transitions, transition)
	return nil
}

func decodeWarningState(data []byte) (WarningState, error) {
	var state WarningState
	if err := decodeExact(data, &state); err != nil {
		return WarningState{}, fmt.Errorf("decode validation warning state: %w", err)
	}
	if state.Protocol != WarningStateProtocol || state.FormatVersion != 1 || state.Canonical ||
		!safeToken(state.StateID) || !validTOPSID(state.TOPSID) || state.Generation == 0 ||
		state.Generation > 9007199254740991 || state.ValidatorID != "symphony-validator" ||
		!safeVersion(state.ValidatorVersion) || !exactUTCSeconds(state.UpdatedAt) ||
		!taggedDigest(state.RepositoryIdentityDigest) || !taggedDigest(state.LastEvidenceDigest) ||
		!taggedDigest(state.StateDigest) || len(state.Subjects) > maxWarningSubjects ||
		len(state.Transitions) > maxWarningTransitions {
		return WarningState{}, fmt.Errorf("validation warning state identity or bound is invalid")
	}
	if state.Generation == 1 && state.PreviousStateDigest != nil ||
		state.Generation > 1 && (state.PreviousStateDigest == nil || !taggedDigest(*state.PreviousStateDigest)) {
		return WarningState{}, fmt.Errorf("validation warning state lineage is invalid")
	}
	if state.LastSync.EvidenceDigest != state.LastEvidenceDigest || !taggedDigest(state.LastSync.SyncDigest) {
		return WarningState{}, fmt.Errorf("validation warning sync identity is invalid")
	}
	for _, collection := range [][]string{
		state.LastSync.KnownSubjectNewOccurrenceIDs, state.LastSync.NewSubjectIDs,
		state.LastSync.ObservedOccurrenceIDs, state.LastSync.ReopenedSubjectIDs,
		state.LastSync.ResolvedSubjectIDs, state.LastSync.UnchangedOccurrenceIDs,
	} {
		if !validSortedDigests(collection) {
			return WarningState{}, fmt.Errorf("validation warning sync collection is invalid")
		}
	}
	syncBasis, err := objectWithout(state.LastSync, "sync_digest")
	if err != nil {
		return WarningState{}, err
	}
	expectedSync, err := digestValue(syncBasis)
	if err != nil || expectedSync != state.LastSync.SyncDigest {
		return WarningState{}, fmt.Errorf("validation warning sync digest mismatch")
	}
	priorSubject := ""
	occurrenceCount := 0
	observed := makeSet(state.LastSync.ObservedOccurrenceIDs)
	knownSubjects := make(map[string]bool, len(state.Subjects))
	lastClassification := make(map[string]string, len(state.Subjects))
	lastMuted := make(map[string]bool, len(state.Subjects))
	for _, subject := range state.Subjects {
		if !taggedDigest(subject.SubjectID) || !safeToken(subject.RuleID) ||
			!validWarningClassification(subject.Classification) || !exactUTCSeconds(subject.FirstObservedAt) ||
			!exactUTCSeconds(subject.LastObservedAt) || priorSubject >= subject.SubjectID && priorSubject != "" ||
			mustParseUTC(subject.LastObservedAt).Before(mustParseUTC(subject.FirstObservedAt)) ||
			!validSortedDigests(subject.OccurrenceIDs) || len(subject.OccurrenceIDs) != len(subject.Occurrences) {
			return WarningState{}, fmt.Errorf("validation warning subject identity or ordering is invalid")
		}
		knownSubjects[subject.SubjectID] = true
		if subject.Classification == "accepted" {
			if !validRationale(subject.Rationale) || subject.SupersededBySubjectID != nil ||
				subject.ValidUntil != nil && !exactUTCSeconds(*subject.ValidUntil) {
				return WarningState{}, fmt.Errorf("accepted warning state is invalid")
			}
		} else if subject.Classification == "superseded" {
			if !validRationale(subject.Rationale) || subject.SupersededBySubjectID == nil ||
				!taggedDigest(*subject.SupersededBySubjectID) || subject.ValidUntil != nil {
				return WarningState{}, fmt.Errorf("superseded warning state is invalid")
			}
		} else if subject.Rationale != nil || subject.ValidUntil != nil || subject.SupersededBySubjectID != nil {
			return WarningState{}, fmt.Errorf("open or resolved warning carries administrative classification fields")
		}
		for index, occurrence := range subject.Occurrences {
			occurrenceCount++
			if occurrence.OccurrenceID != subject.OccurrenceIDs[index] || occurrence.Finding.Category != "warning" ||
				occurrence.Finding.OccurrenceID != occurrence.OccurrenceID ||
				occurrence.Finding.SubjectID != subject.SubjectID || occurrence.Finding.RuleID != subject.RuleID ||
				occurrence.Finding.OccurrenceID != findingDigest(occurrence.Finding, false) ||
				occurrence.Finding.SubjectID != findingDigest(occurrence.Finding, true) ||
				!exactUTCSeconds(occurrence.FirstObservedAt) || !exactUTCSeconds(occurrence.LastObservedAt) ||
				mustParseUTC(occurrence.LastObservedAt).Before(mustParseUTC(occurrence.FirstObservedAt)) ||
				occurrence.Active != observed[occurrence.OccurrenceID] ||
				!validDigestSequence(occurrence.EvidenceDigests) {
				return WarningState{}, fmt.Errorf("validation warning occurrence history is invalid")
			}
		}
		priorSubject = subject.SubjectID
	}
	if occurrenceCount > maxWarningOccurrences {
		return WarningState{}, fmt.Errorf("validation warning occurrence history exceeds its bound")
	}
	var previous *string
	for index, transition := range state.Transitions {
		if transition.Sequence != uint64(index+1) || !taggedDigest(transition.SubjectID) ||
			!exactUTCSeconds(transition.At) || !validWarningClassification(transition.ToClassification) ||
			!validOptionalClassification(transition.FromClassification) || !validOptionalDigest(transition.OccurrenceID) ||
			!validOptionalDigest(transition.SupersededBySubjectID) || !validOptionalTimestamp(transition.ValidUntil) ||
			!validOptionalRationale(transition.Rationale) || !validWarningOperation(transition.Operation) ||
			!taggedDigest(transition.TransitionDigest) || !pointerStringEqual(transition.PreviousTransitionDigest, previous) {
			return WarningState{}, fmt.Errorf("validation warning transition chain is invalid")
		}
		if transition.PreviousTransitionDigest != nil && !taggedDigest(*transition.PreviousTransitionDigest) {
			return WarningState{}, fmt.Errorf("validation warning transition predecessor is invalid")
		}
		basis, err := objectWithout(transition, "transition_digest")
		if err != nil {
			return WarningState{}, err
		}
		expected, err := digestValue(basis)
		if err != nil || expected != transition.TransitionDigest {
			return WarningState{}, fmt.Errorf("validation warning transition digest mismatch")
		}
		previous = stringPointer(transition.TransitionDigest)
		lastClassification[transition.SubjectID] = transition.ToClassification
		switch transition.Operation {
		case "muted":
			lastMuted[transition.SubjectID] = true
		case "unmuted":
			lastMuted[transition.SubjectID] = false
		}
	}
	for _, subject := range state.Subjects {
		if lastClassification[subject.SubjectID] != subject.Classification || lastMuted[subject.SubjectID] != subject.Muted {
			return WarningState{}, fmt.Errorf("validation warning subject does not match its transition history")
		}
		if subject.SupersededBySubjectID != nil && !knownSubjects[*subject.SupersededBySubjectID] {
			return WarningState{}, fmt.Errorf("validation warning supersession target is unknown")
		}
	}
	basis, err := objectWithout(state, "state_digest")
	if err != nil {
		return WarningState{}, err
	}
	expected, err := digestValue(basis)
	if err != nil || expected != state.StateDigest {
		return WarningState{}, fmt.Errorf("validation warning state digest mismatch")
	}
	return state, nil
}

func setWarningStateDigest(state *WarningState) error {
	basis, err := objectWithout(*state, "state_digest")
	if err != nil {
		return err
	}
	state.StateDigest, err = digestValue(basis)
	return err
}

func mutationRetryMatches(state WarningState, expected string) bool {
	return expected == state.StateDigest || state.Generation == 1 && expected == "absent" ||
		state.PreviousStateDigest != nil && *state.PreviousStateDigest == expected
}

func hasExpiredAcceptance(state WarningState, now time.Time) bool {
	for _, subject := range state.Subjects {
		if subject.Classification == "accepted" && subject.ValidUntil != nil &&
			!now.UTC().Before(mustParseUTC(*subject.ValidUntil)) {
			return true
		}
	}
	return false
}

func warningMutationRetryMatches(state WarningState, mutation WarningMutation, expected string) bool {
	if state.PreviousStateDigest == nil || *state.PreviousStateDigest != expected || len(state.Transitions) == 0 {
		return false
	}
	last := state.Transitions[len(state.Transitions)-1]
	if last.SubjectID != mutation.SubjectID || last.Rationale == nil || *last.Rationale != mutation.Rationale {
		return false
	}
	if mutation.Muted != nil {
		return *mutation.Muted && last.Operation == "muted" || !*mutation.Muted && last.Operation == "unmuted"
	}
	switch mutation.Classification {
	case "accepted":
		return last.Operation == "accepted" && pointerValue(last.ValidUntil) == mutation.ValidUntil
	case "open":
		return last.Operation == "reopened"
	case "superseded":
		return last.Operation == "superseded" && pointerValue(last.SupersededBySubjectID) == mutation.SupersededBySubjectID
	default:
		return false
	}
}

func pointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func warningSupersessionCycle(state WarningState, source, target string) bool {
	seen := map[string]bool{source: true}
	for target != "" {
		if seen[target] {
			return true
		}
		seen[target] = true
		index := sort.Search(len(state.Subjects), func(i int) bool { return state.Subjects[i].SubjectID >= target })
		if index == len(state.Subjects) || state.Subjects[index].SubjectID != target ||
			state.Subjects[index].SupersededBySubjectID == nil {
			return false
		}
		target = *state.Subjects[index].SupersededBySubjectID
	}
	return false
}

func validWarningClassification(value string) bool {
	return value == "open" || value == "accepted" || value == "resolved" || value == "superseded"
}

func validWarningOperation(value string) bool {
	switch value {
	case "detected", "occurrence_observed", "not_observed", "recurred", "acceptance_expired",
		"accepted", "reopened", "superseded", "muted", "unmuted":
		return true
	default:
		return false
	}
}

func validOptionalClassification(value *string) bool {
	return value == nil || validWarningClassification(*value)
}

func validOptionalDigest(value *string) bool {
	return value == nil || taggedDigest(*value)
}

func validOptionalTimestamp(value *string) bool {
	return value == nil || exactUTCSeconds(*value)
}

func validRationale(value *string) bool {
	return value != nil && strings.TrimSpace(*value) != "" && len(*value) <= 4096
}

func validOptionalRationale(value *string) bool {
	return value == nil || validRationale(value)
}

func validSortedDigests(values []string) bool {
	for index, value := range values {
		if !taggedDigest(value) || index > 0 && values[index-1] >= value {
			return false
		}
	}
	return true
}

func validDigestSequence(values []string) bool {
	seen := map[string]bool{}
	for _, value := range values {
		if !taggedDigest(value) || seen[value] {
			return false
		}
		seen[value] = true
	}
	return len(values) != 0
}

func totalWarningOccurrences(subjects []WarningSubject) int {
	total := 0
	for _, subject := range subjects {
		total += len(subject.Occurrences)
	}
	return total
}

func sortedStrings(values []string) []string {
	result := make([]string, len(values))
	copy(result, values)
	sort.Strings(result)
	return result
}

func insertSorted(values []string, value string) []string {
	index := sort.SearchStrings(values, value)
	values = append(values, "")
	copy(values[index+1:], values[index:])
	values[index] = value
	return values
}

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func pointerStringEqual(left, right *string) bool {
	return left == nil && right == nil || left != nil && right != nil && *left == *right
}

func mustParseUTC(value string) time.Time {
	parsed, _ := time.Parse("2006-01-02T15:04:05Z", value)
	return parsed
}
