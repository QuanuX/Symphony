package validation

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

const testTOPSID = "123e4567-e89b-42d3-a456-426614174000"

func TestTOPSIDAcceptsCanonicalRFC9562Versions(t *testing.T) {
	for _, value := range []string{
		"123e4567-e89b-12d3-a456-426614174000",
		"018f0c3a-7b2d-7e11-8c12-0242ac120002",
		"123e4567-e89b-82d3-b456-426614174000",
	} {
		if !validTOPSID(value) {
			t.Fatalf("canonical RFC UUID was rejected: %s", value)
		}
	}
	for _, value := range []string{
		"00000000-0000-0000-0000-000000000000",
		"123e4567-e89b-02d3-a456-426614174000",
		"123e4567-e89b-92d3-a456-426614174000",
		"018F0C3A-7B2D-7E11-8C12-0242AC120002",
		"018f0c3a-7b2d-7e11-7c12-0242ac120002",
	} {
		if validTOPSID(value) {
			t.Fatalf("noncanonical or unsupported UUID was accepted: %s", value)
		}
	}
}

func TestPolicyStoreCASPersistenceAndRemoval(t *testing.T) {
	store, err := NewStore(t.TempDir(), testTOPSID)
	if err != nil {
		t.Fatal(err)
	}
	config := PolicyConfig{
		ProfileID: "guarded", DefaultWarningDisposition: "review",
		HistoricalPresentation: "count", NewPresentation: "summary",
		Rules: []RulePolicy{
			{RuleID: "z.rule", Disposition: "require", Presentation: "full"},
			{RuleID: "a.rule", Disposition: "record", Presentation: "summary"},
		},
	}
	policy, changed, err := store.SetPolicy(config, "absent", time.Date(2026, 8, 11, 14, 15, 16, 999, time.UTC))
	if err != nil || !changed {
		t.Fatalf("set policy: changed=%t err=%v", changed, err)
	}
	if policy.Generation != 1 || policy.UpdatedAt != "2026-08-11T14:15:16Z" || policy.Rules[0].RuleID != "a.rule" {
		t.Fatalf("policy was not normalized: %+v", policy)
	}
	snapshot, err := store.Policy("guarded")
	if err != nil || !snapshot.Exists || snapshot.Policy.PolicyDigest != policy.PolicyDigest {
		t.Fatalf("policy snapshot mismatch: %+v %v", snapshot, err)
	}
	list, err := store.ListPolicies()
	if err != nil || len(list.Policies) != 1 || list.Policies[0].ProfileID != "guarded" {
		t.Fatalf("policy list mismatch: %+v %v", list, err)
	}
	retried, changed, err := store.SetPolicy(config, "absent", time.Now())
	if err != nil || changed || retried.PolicyDigest != policy.PolicyDigest {
		t.Fatalf("initial semantic retry did not converge: changed=%t err=%v", changed, err)
	}
	conflicting := config
	conflicting.DefaultWarningDisposition = "require"
	if _, _, err := store.SetPolicy(conflicting, "absent", time.Now()); err == nil || !strings.Contains(err.Error(), "compare-and-swap") {
		t.Fatalf("stale policy mutation was accepted: %v", err)
	}
	policy, changed, err = store.SetPolicy(config, policy.PolicyDigest, time.Now())
	if err != nil || changed || policy.Generation != 1 {
		t.Fatalf("semantic retry was not idempotent: changed=%t policy=%+v err=%v", changed, policy, err)
	}
	changed, err = store.RemovePolicy("guarded", policy.PolicyDigest)
	if err != nil || !changed {
		t.Fatalf("remove policy: changed=%t err=%v", changed, err)
	}
	changed, err = store.RemovePolicy("guarded", policy.PolicyDigest)
	if err != nil || changed {
		t.Fatalf("remove policy retry: changed=%t err=%v", changed, err)
	}
}

func TestBaselinePersistenceAndEvaluationDelta(t *testing.T) {
	store, err := NewStore(t.TempDir(), testTOPSID)
	if err != nil {
		t.Fatal(err)
	}
	old := sampleResult([]Finding{
		warning("one", "rule.one", "path=a.md"),
		warning("two", "rule.two", "path=b.md"),
	})
	baseline, changed, err := store.CreateBaseline("known", "absent", old, time.Date(2026, 8, 11, 1, 2, 3, 0, time.UTC))
	if err != nil || !changed {
		t.Fatalf("create baseline: changed=%t err=%v", changed, err)
	}
	snapshot, err := store.Baseline("known")
	if err != nil || !snapshot.Exists || snapshot.Baseline.BaselineDigest != baseline.BaselineDigest {
		t.Fatalf("baseline snapshot mismatch: %+v %v", snapshot, err)
	}
	current := sampleResult([]Finding{
		warning("two", "rule.two", "path=b.md"),
		warning("three", "rule.three", "path=c.md"),
	})
	policy := Policy{
		ProfileID: "guarded", PolicyDigest: digest('9'), TOPSID: testTOPSID,
		DefaultWarningDisposition: "review", HistoricalPresentation: "count", NewPresentation: "full",
		Rules: []RulePolicy{{RuleID: "rule.three", Disposition: "require", Presentation: "full"}},
	}
	projection, err := Evaluate(current, &policy, &snapshot.Baseline, DisplayFilter{})
	if err != nil {
		t.Fatal(err)
	}
	evaluation := projection.Result.Evaluation
	if evaluation.Outcome != "failed" || len(evaluation.NewWarningOccurrenceIDs) != 1 ||
		len(evaluation.UnchangedWarningOccurrenceIDs) != 1 || len(evaluation.ResolvedWarningOccurrenceIDs) != 1 ||
		len(evaluation.RequiredRuleIDs) != 1 || len(projection.Displayed) != 1 {
		t.Fatalf("unexpected delta evaluation: %+v displayed=%+v", evaluation, projection.Displayed)
	}
	if _, err := Evaluate(current, &policy, &Baseline{
		RepositoryIdentityDigest: digest('8'), ValidatorID: "symphony-validator", ValidatorVersion: "0.1.0-dev",
	}, DisplayFilter{}); err == nil || !strings.Contains(err.Error(), "repository identity mismatch") {
		t.Fatalf("incompatible baseline was accepted: %v", err)
	}
}

func TestValidationStateRejectsSymlinkRoot(t *testing.T) {
	root := t.TempDir()
	link := filepath.Join(t.TempDir(), "state")
	if err := os.Symlink(root, link); err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(link, testTOPSID)
	if err == nil {
		_, err = store.ListPolicies()
	}
	if err == nil {
		t.Fatal("symlink state root was accepted")
	}
}

func TestWarningLifecyclePreservesSubjectsOccurrencesAndActionability(t *testing.T) {
	store, err := NewStore(t.TempDir(), testTOPSID)
	if err != nil {
		t.Fatal(err)
	}
	first := warningWithSubject("one-v1", "subject-one", "rule.one", "path=a.md")
	second := warningWithSubject("two-v1", "subject-two", "rule.two", "path=b.md")
	raw := sampleResult([]Finding{first, second})
	state, changed, err := store.SyncWarningState("default", "absent", raw, time.Date(2026, 8, 13, 1, 0, 0, 0, time.UTC))
	if err != nil || !changed || state.Generation != 1 || len(state.LastSync.NewSubjectIDs) != 2 {
		t.Fatalf("initial warning sync: changed=%t state=%+v err=%v", changed, state, err)
	}
	accepted, changed, err := store.MutateWarning(WarningMutation{
		Classification: "accepted", ExpectedStateDigest: state.StateDigest,
		Rationale: "tracked by the owning module", StateID: "default", SubjectID: first.SubjectID,
		ValidUntil: "2026-08-14T01:00:00Z",
	}, time.Date(2026, 8, 13, 2, 0, 0, 0, time.UTC))
	if err != nil || !changed {
		t.Fatalf("accept warning: changed=%t err=%v", changed, err)
	}
	projection, err := EvaluateWithWarningState(raw, nil, nil, &accepted, DisplayFilter{},
		time.Date(2026, 8, 13, 3, 0, 0, 0, time.UTC), false)
	if err != nil || len(projection.Displayed) != 1 || projection.Displayed[0].SubjectID != second.SubjectID {
		t.Fatalf("ordinary output did not contain only actionable open warnings: displayed=%+v err=%v", projection.Displayed, err)
	}
	debug, err := EvaluateWithWarningState(raw, nil, nil, &accepted,
		DisplayFilter{Classification: "accepted"}, time.Date(2026, 8, 13, 3, 0, 0, 0, time.UTC), true)
	if err != nil || len(debug.Displayed) != 1 || debug.Displayed[0].SubjectID != first.SubjectID {
		t.Fatalf("accepted warning was not debug-queryable: %+v err=%v", debug.Displayed, err)
	}

	firstV2 := warningWithSubject("one-v2", "subject-one", "rule.one", "path=a-renamed.md")
	nextRaw := sampleResult([]Finding{firstV2})
	next, changed, err := store.SyncWarningState("default", accepted.StateDigest, nextRaw,
		time.Date(2026, 8, 13, 4, 0, 0, 0, time.UTC))
	if err != nil || !changed {
		t.Fatalf("second warning sync: changed=%t err=%v", changed, err)
	}
	if len(next.LastSync.NewSubjectIDs) != 0 || len(next.LastSync.KnownSubjectNewOccurrenceIDs) != 1 ||
		next.LastSync.KnownSubjectNewOccurrenceIDs[0] != firstV2.OccurrenceID || len(next.LastSync.ResolvedSubjectIDs) != 1 {
		t.Fatalf("subject-aware delta is incorrect: %+v", next.LastSync)
	}
	resolvedDebug, err := EvaluateWithWarningState(nextRaw, nil, nil, &next,
		DisplayFilter{Classification: "resolved", SubjectID: second.SubjectID},
		time.Date(2026, 8, 13, 4, 0, 0, 0, time.UTC), true)
	if err != nil || len(resolvedDebug.Historical) != 1 || resolvedDebug.Historical[0].Finding.Detail != second.Detail ||
		len(resolvedDebug.Historical[0].EvidenceDigests) != 1 {
		t.Fatalf("resolved warning evidence is not explainable: %+v err=%v", resolvedDebug.Historical, err)
	}
	snapshot, err := store.WarningState("default")
	if err != nil || !snapshot.Exists || snapshot.State.StateDigest != next.StateDigest {
		t.Fatalf("warning state did not round-trip: %+v err=%v", snapshot, err)
	}
	if _, _, err := store.MutateWarning(WarningMutation{
		Classification: "open", ExpectedStateDigest: accepted.StateDigest,
		Rationale: "stale writer", StateID: "default", SubjectID: first.SubjectID,
	}, time.Now()); err == nil || !strings.Contains(err.Error(), "compare-and-swap") {
		t.Fatalf("stale warning lifecycle mutation was accepted: %v", err)
	}
}

func TestWarningLifecycleMuteExpiryAndSupersessionCycle(t *testing.T) {
	store, err := NewStore(t.TempDir(), testTOPSID)
	if err != nil {
		t.Fatal(err)
	}
	first := warningWithSubject("one", "subject-one", "rule.one", "path=a.md")
	second := warningWithSubject("two", "subject-two", "rule.two", "path=b.md")
	state, _, err := store.SyncWarningState("guarded", "absent", sampleResult([]Finding{first, second}),
		time.Date(2026, 8, 13, 1, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	muted := true
	state, changed, err := store.MutateWarning(WarningMutation{
		ExpectedStateDigest: state.StateDigest, Muted: &muted, Rationale: "reduce ordinary presentation only",
		StateID: "guarded", SubjectID: first.SubjectID,
	}, time.Date(2026, 8, 13, 2, 0, 0, 0, time.UTC))
	if err != nil || !changed {
		t.Fatalf("mute warning: changed=%t err=%v", changed, err)
	}
	requirePolicy := &Policy{
		ProfileID: "guarded", PolicyDigest: digest('8'), TOPSID: testTOPSID,
		DefaultWarningDisposition: "require", HistoricalPresentation: "full", NewPresentation: "full",
	}
	projection, err := EvaluateWithWarningState(sampleResult([]Finding{first, second}), requirePolicy, nil, &state,
		DisplayFilter{}, time.Date(2026, 8, 13, 3, 0, 0, 0, time.UTC), false)
	if err != nil || len(projection.Displayed) != 1 || projection.Displayed[0].SubjectID != second.SubjectID ||
		projection.Result.Evaluation.Outcome != "failed" || len(projection.Result.Evaluation.RequiredRuleIDs) != 1 {
		t.Fatalf("mute changed more than presentation: displayed=%+v err=%v", projection.Displayed, err)
	}
	debug, err := EvaluateWithWarningState(sampleResult([]Finding{first, second}), nil, nil, &state,
		DisplayFilter{Classification: "muted"}, time.Date(2026, 8, 13, 3, 0, 0, 0, time.UTC), true)
	if err != nil || len(debug.Displayed) != 1 || debug.Displayed[0].SubjectID != first.SubjectID {
		t.Fatalf("muted warning disappeared from debug evidence: %+v err=%v", debug.Displayed, err)
	}
	state, _, err = store.MutateWarning(WarningMutation{
		Classification: "superseded", ExpectedStateDigest: state.StateDigest, Rationale: "same defect, stable successor",
		StateID: "guarded", SubjectID: first.SubjectID, SupersededBySubjectID: second.SubjectID,
	}, time.Date(2026, 8, 13, 4, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.MutateWarning(WarningMutation{
		Classification: "superseded", ExpectedStateDigest: state.StateDigest, Rationale: "would make a cycle",
		StateID: "guarded", SubjectID: second.SubjectID, SupersededBySubjectID: first.SubjectID,
	}, time.Date(2026, 8, 13, 5, 0, 0, 0, time.UTC)); err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("supersession cycle was accepted: %v", err)
	}
	if _, _, err := store.MutateWarning(WarningMutation{
		Classification: "accepted", ExpectedStateDigest: state.StateDigest, Rationale: "expired already",
		StateID: "guarded", SubjectID: second.SubjectID, ValidUntil: "2026-08-13T04:59:59Z",
	}, time.Date(2026, 8, 13, 5, 0, 0, 0, time.UTC)); err == nil || !strings.Contains(err.Error(), "future STSC") {
		t.Fatalf("past acceptance expiry was accepted: %v", err)
	}
}

func TestWarningAcceptanceExpiryIsImmediatelyActionableAndDurablySynchronized(t *testing.T) {
	store, err := NewStore(t.TempDir(), testTOPSID)
	if err != nil {
		t.Fatal(err)
	}
	finding := warningWithSubject("occurrence", "subject", "rule.expiry", "path=expiry.md")
	raw := sampleResult([]Finding{finding})
	state, _, err := store.SyncWarningState("expiry", "absent", raw,
		time.Date(2026, 8, 13, 1, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	state, _, err = store.MutateWarning(WarningMutation{
		Classification: "accepted", ExpectedStateDigest: state.StateDigest, Rationale: "bounded exception",
		StateID: "expiry", SubjectID: finding.SubjectID, ValidUntil: "2026-08-13T03:00:00Z",
	}, time.Date(2026, 8, 13, 2, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	projection, err := EvaluateWithWarningState(raw, nil, nil, &state, DisplayFilter{},
		time.Date(2026, 8, 13, 3, 0, 0, 0, time.UTC), false)
	if err != nil || len(projection.Displayed) != 1 ||
		projection.WarningClassifications[finding.OccurrenceID] != "open" {
		t.Fatalf("expired acceptance did not become immediately actionable: %+v err=%v", projection, err)
	}
	state, changed, err := store.SyncWarningState("expiry", state.StateDigest, raw,
		time.Date(2026, 8, 13, 3, 0, 1, 0, time.UTC))
	if err != nil || !changed || state.Subjects[0].Classification != "open" ||
		state.Transitions[len(state.Transitions)-1].Operation != "acceptance_expired" {
		t.Fatalf("expired acceptance was not durably synchronized: changed=%t state=%+v err=%v", changed, state, err)
	}
}

func TestWarningLifecycleConcurrentCASHasOneWinnerAndValidState(t *testing.T) {
	store, err := NewStore(t.TempDir(), testTOPSID)
	if err != nil {
		t.Fatal(err)
	}
	first := warningWithSubject("first", "subject-a", "ignored", "path=a.md")
	second := warningWithSubject("second", "subject-b", "ignored", "path=b.md")
	state, _, err := store.SyncWarningState("concurrent", "absent", sampleResult([]Finding{first, second}),
		time.Date(2026, 8, 13, 1, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	var group sync.WaitGroup
	for index, subjectID := range []string{first.SubjectID, second.SubjectID} {
		group.Add(1)
		go func(index int, subjectID string) {
			defer group.Done()
			<-start
			_, _, err := store.MutateWarning(WarningMutation{
				Classification: "accepted", ExpectedStateDigest: state.StateDigest,
				Rationale: fmt.Sprintf("concurrent writer %d", index), StateID: "concurrent", SubjectID: subjectID,
			}, time.Date(2026, 8, 13, 2, 0, 0, 0, time.UTC))
			results <- err
		}(index, subjectID)
	}
	close(start)
	group.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
			continue
		}
		if !strings.Contains(err.Error(), "compare-and-swap") && !strings.Contains(err.Error(), "busy") {
			t.Fatalf("unexpected concurrent mutation error: %v", err)
		}
	}
	if successes != 1 {
		t.Fatalf("concurrent exact-CAS success count = %d, want 1", successes)
	}
	snapshot, err := store.WarningState("concurrent")
	if err != nil || !snapshot.Exists || snapshot.State.Generation != 2 {
		t.Fatalf("concurrent state did not remain valid: %+v err=%v", snapshot, err)
	}
	accepted := 0
	for _, subject := range snapshot.State.Subjects {
		if subject.Classification == "accepted" {
			accepted++
		}
	}
	if accepted != 1 {
		t.Fatalf("accepted subjects after concurrent CAS = %d, want 1", accepted)
	}
}

func TestRunRootSummaryUsesExactInstalledProjectionAndExposesExit25(t *testing.T) {
	repository := t.TempDir()
	canonicalRepositoryPath, err := filepath.EvalSymlinks(repository)
	if err != nil {
		t.Fatal(err)
	}
	summary := sampleRootSummary(t)
	encoded, err := canonicalJSON(summary)
	if err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\n" +
		"if [ \"$#\" -ne 4 ] || [ \"$1\" != \"root-summary\" ] || [ \"$2\" != \"--repo\" ] || [ \"$3\" != \"" + canonicalRepositoryPath + "\" ] || [ \"$4\" != \"--json\" ]; then exit 23; fi\n" +
		"printf '%s\\n' '" + string(encoded) + "'\n"
	prefix := installValidationFixture(t, script)
	observed, err := RunRootSummary(t.Context(), prefix, "0.1.0-dev", repository)
	if err != nil || observed.SummaryDigest != summary.SummaryDigest || observed.QXCTL.RegisteredCommands != 144 {
		t.Fatalf("exact root-summary invocation failed: observed=%+v err=%v", observed, err)
	}

	failingPrefix := installValidationFixture(t, "#!/bin/sh\necho 'root_summary.commands reason=missing_commands' >&2\nexit 25\n")
	_, err = RunRootSummary(t.Context(), failingPrefix, "0.1.0-dev", repository)
	var exitFailure *ValidatorExitError
	if !errors.As(err, &exitFailure) || exitFailure.ExitCode != 25 ||
		!strings.Contains(exitFailure.Diagnostics, "root_summary.commands") {
		t.Fatalf("root-summary exit 25 was not preserved: %#v", err)
	}
}

func TestDecodeRootSummaryRejectsUnknownFieldsAndDigestDrift(t *testing.T) {
	summary := sampleRootSummary(t)
	encoded, err := canonicalJSON(summary)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeRootSummary(encoded); err != nil {
		t.Fatalf("valid root summary was rejected: %v", err)
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatal(err)
	}
	object["unexpected"] = true
	unknown, _ := json.Marshal(object)
	if _, err := decodeRootSummary(unknown); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown root-summary field was accepted: %v", err)
	}
	delete(object, "unexpected")
	object["summary_digest"] = digest('9')
	tampered, _ := json.Marshal(object)
	if _, err := decodeRootSummary(tampered); err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("root-summary digest drift was accepted: %v", err)
	}
}

func sampleRootSummary(t *testing.T) RootSummary {
	t.Helper()
	summary := RootSummary{
		FeatureAdministration: RootSummaryAdministration{
			Exemptions: 1, Expectations: 12, Prohibited: 2, Required: 9, Unreviewed: 0,
		},
		FormatVersion: 1, Protocol: RootSummaryProtocol,
		PublishedSourceVersions: []RootSummaryPublication{{
			Coordinate: "github.com/QuanuX/Symphony/tools/qxctl", Revision: strings.Repeat("a", 40),
			Tag: "tools/qxctl/v0.1.0", Version: "v0.1.0",
		}},
		QXCTL: RootSummaryQXCTL{RegisteredCommands: 144},
		SSFV: RootSummarySSFV{
			CatalogState: "partial", NestedFeatures: 8, RegisteredFeatures: 10,
			RegisteredOwnerFeatures: []string{"ssfv:symphony:qxctl", "ssfv:symphony:symphony-validator"},
			RegisteredOwnerScopes:   2,
		},
	}
	basis, err := objectWithout(summary, "summary_digest")
	if err != nil {
		t.Fatal(err)
	}
	summary.SummaryDigest, err = digestValue(basis)
	if err != nil {
		t.Fatal(err)
	}
	return summary
}

func installValidationFixture(t *testing.T, executable string) string {
	t.Helper()
	prefix := t.TempDir()
	version := "0.1.0-dev"
	base := "share/doc/symphony/symphony-validator/" + version + "/"
	license := "share/licenses/symphony-validator/" + version + "/"
	paths := []string{
		"libexec/symphony/symphony-validator/" + version + "/symphony-validator",
		"share/symphony/receipts/symphony-validator/" + version + "/install-receipt.json",
		base + "INTENT.md", base + "MANIFEST.md", base + "INSTALL.md", base + "SKILL.md", base + "SPEC.md",
		license + "LICENSE-AGPL-3.0", license + "nlohmann-json-LICENSE.MIT",
	}
	for _, relative := range paths {
		path := filepath.Join(prefix, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if strings.HasSuffix(relative, "/symphony-validator") {
			if err := os.WriteFile(path, []byte(executable), 0o755); err != nil {
				t.Fatal(err)
			}
		} else if !strings.HasSuffix(relative, "/install-receipt.json") {
			if err := os.WriteFile(path, []byte("fixture\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	receipt := map[string]any{
		"protocol": "symphony.knowledge.install-receipt.v1", "module_id": "symphony-validator",
		"version": version, "install_scope": "prefix", "prefix_mode": "installation_prefix",
		"state": "installed_undocked", "active": false, "default_receptor": nil, "files": paths,
	}
	data, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(prefix, "share/symphony/receipts/symphony-validator", version, "install-receipt.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	return prefix
}

func sampleResult(findings []Finding) Result {
	result := Result{Protocol: ResultProtocol, FormatVersion: 1}
	result.Evidence = Evidence{
		ValidatorID: "symphony-validator", ValidatorVersion: "0.1.0-dev",
		RepositoryIdentityDigest: digest('1'), EvidenceDigest: digest('2'), Outcome: "pass",
		Findings: findings,
	}
	for _, finding := range findings {
		result.Evidence.Summary.Total++
		switch finding.Category {
		case "warning":
			result.Evidence.Summary.Warning++
		case "pass":
			result.Evidence.Summary.Pass++
		case "violation":
			result.Evidence.Summary.Violation++
		default:
			result.Evidence.Summary.Other++
		}
	}
	evidenceBasis, _ := objectWithout(result.Evidence, "evidence_digest")
	result.Evidence.EvidenceDigest, _ = digestValue(evidenceBasis)
	resultBasis, _ := objectWithout(result, "result_digest")
	result.ResultDigest, _ = digestValue(resultBasis)
	return result
}

func warning(marker, rule, detail string) Finding {
	_ = marker
	finding := Finding{
		Category: "warning", RuleID: rule, Detail: detail, Scope: "historical",
		Attributes: map[string]string{"path": strings.TrimPrefix(detail, "path=")},
	}
	finding.OccurrenceID = findingDigest(finding, false)
	finding.SubjectID = findingDigest(finding, true)
	return finding
}

func warningWithSubject(marker, subjectMarker, rule, detail string) Finding {
	_ = marker
	_ = rule
	finding := Finding{
		Category: "warning", RuleID: "sclv.affected_surface.unindexed", Detail: detail, Scope: "historical",
		Attributes: map[string]string{"path": subjectMarker},
	}
	finding.OccurrenceID = findingDigest(finding, false)
	finding.SubjectID = findingDigest(finding, true)
	return finding
}

func digest(marker byte) string { return "sha256:" + strings.Repeat(string(marker), 64) }
