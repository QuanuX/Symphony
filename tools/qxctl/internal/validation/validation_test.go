package validation

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
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

func sampleResult(findings []Finding) Result {
	result := Result{Protocol: ResultProtocol, FormatVersion: 1, ResultDigest: digest('7')}
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
	return result
}

func warning(marker, rule, detail string) Finding {
	hash := sha256.Sum256([]byte(marker))
	identity := "sha256:" + hex.EncodeToString(hash[:])
	return Finding{
		Category: "warning", RuleID: rule, Detail: detail, Scope: "historical",
		Attributes:   map[string]string{"path": strings.TrimPrefix(detail, "path=")},
		OccurrenceID: identity, SubjectID: identity,
	}
}

func digest(marker byte) string { return "sha256:" + strings.Repeat(string(marker), 64) }
