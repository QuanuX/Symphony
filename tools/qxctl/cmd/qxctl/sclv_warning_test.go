package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/validation"
)

func TestSCLVWarningAcknowledgementPreservesEvidenceAndLocalScope(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	opts := validationOptions{topsID: "11111111-1111-4111-8111-111111111111", stateRoot: t.TempDir(), warningStateID: "default"}
	store, err := validation.NewStore(opts.stateRoot, opts.topsID)
	if err != nil {
		t.Fatal(err)
	}
	raw := sclvWarningFixture(t, "SCLV-CHG-TEST-OLD")
	state, _, err := store.SyncWarningState("default", "absent", raw, now)
	if err != nil {
		t.Fatal(err)
	}
	listed := captureStdout(t, func() error { return runSCLVWarning("list", opts) })
	if !strings.Contains(listed, "count=2") || !strings.Contains(listed, `record="SCLV-CHG-TEST-OLD" path="retired.md"`) || strings.Contains(listed, "rule=unrelated.warning") {
		t.Fatalf("incorrect selection: %s", listed)
	}
	opts.subjectID, opts.expectedStateDigest, opts.rationale = raw.Evidence.Findings[0].SubjectID, state.StateDigest, "Intentional manual retirement"
	captureStdout(t, func() error {
		root, err := newRootCommand()
		if err != nil {
			return err
		}
		root.SetArgs([]string{"sclv", "warning", "acknowledge", "--tops-id", opts.topsID, "--state-root", opts.stateRoot, "--subject-id", opts.subjectID, "--expected-state-digest", opts.expectedStateDigest, "--rationale", opts.rationale})
		return root.Execute()
	})
	accepted, err := store.WarningState("default")
	if err != nil {
		t.Fatal(err)
	}
	check := func(evidence validation.Result, state *validation.WarningState, count int) {
		t.Helper()
		p, err := validation.EvaluateWithWarningState(evidence, nil, nil, state, validation.DisplayFilter{}, now, false)
		if err != nil || len(p.Displayed) != count || !reflect.DeepEqual(p.Result.Evidence, evidence.Evidence) {
			t.Fatalf("wrong actionability or changed evidence: count=%d want=%d err=%v", len(p.Displayed), count, err)
		}
	}
	check(raw, &accepted.State, 2)
	check(raw, nil, 3) // A clean delivery state carries no workstation acceptance.
	check(sclvWarningFixture(t, "SCLV-CHG-TEST-NEW"), &accepted.State, 3)
	captureStdout(t, func() error { return runSCLVWarning("acknowledge", opts) })
	opts.subjectID = raw.Evidence.Findings[2].SubjectID
	if err := runSCLVWarning("acknowledge", opts); err == nil || !strings.Contains(err.Error(), "not a supported SCLV") {
		t.Fatalf("unrelated selection accepted: %v", err)
	}
	opts.subjectID = raw.Evidence.Findings[1].SubjectID
	if err := runSCLVWarning("acknowledge", opts); err == nil || !strings.Contains(err.Error(), "compare-and-swap") {
		t.Fatalf("stale write accepted: %v", err)
	}
	unchanged, err := store.WarningState("default")
	if err != nil || unchanged.State.StateDigest != accepted.State.StateDigest {
		t.Fatalf("retry or rejected write changed state: %v", err)
	}
	opts.subjectID, opts.expectedStateDigest, opts.rationale = raw.Evidence.Findings[0].SubjectID, accepted.State.StateDigest, "Revisit retirement"
	captureStdout(t, func() error { return runSCLVWarning("reopen", opts) })
	reopened, err := store.WarningState("default")
	if err != nil {
		t.Fatal(err)
	}
	check(raw, &reopened.State, 3)
	if len(reopened.State.Transitions) != len(state.Transitions)+2 {
		t.Fatal("accept/reopen history lost")
	}
}

func TestSCLVWarningSelectionFailsClosed(t *testing.T) {
	f := sclvWarningFixture(t, "SCLV-CHG-TEST").Evidence.Findings[0]
	for _, category := range []string{"violation", "pass", "other"} {
		altered := f
		altered.Category = category
		if isSCLVHistoricalWarning(validation.WarningSubject{RuleID: f.RuleID, Occurrences: []validation.WarningOccurrence{{Finding: altered}}}) {
			t.Fatalf("selected %s", category)
		}
	}
	if isSCLVHistoricalWarning(validation.WarningSubject{RuleID: f.RuleID}) {
		t.Fatal("empty subject selected")
	}
	f.Attributes = map[string]string{"path": "retired.md"}
	if isSCLVHistoricalWarning(validation.WarningSubject{RuleID: f.RuleID, Occurrences: []validation.WarningOccurrence{{Finding: f}}}) {
		t.Fatal("missing record identity accepted")
	}
}

func TestSCLVWarningCommandRegistryContracts(t *testing.T) {
	root, err := newRootCommand()
	if err != nil {
		t.Fatal(err)
	}
	m, err := commandregistry.BuildExpected(root)
	if err != nil {
		t.Fatal(err)
	}
	seen := 0
	for _, r := range m.Commands {
		if !strings.HasPrefix(r.CommandID, "qxcmd:symphony:sclv.warning.") {
			continue
		}
		seen++
		mutation := strings.HasSuffix(r.CommandID, ".acknowledge") || strings.HasSuffix(r.CommandID, ".reopen")
		interaction := "inspect"
		if strings.HasSuffix(r.CommandID, ".list") {
			interaction = "discover"
		}
		if mutation {
			interaction = "configure"
		}
		if !containsFeatureBinding(r.FeatureBindings, commandregistry.FeatureBinding{FeatureID: featureValidation, Interaction: interaction}) {
			t.Fatalf("wrong owner: %+v", r)
		}
		if interaction != "discover" && (!reflect.DeepEqual(r.OutputProtocols, []string{validation.WarningStateProtocol}) || !reflect.DeepEqual(r.ResultValidationProtocols, []string{validation.WarningStateProtocol})) {
			t.Fatalf("wrong protocol: %+v", r)
		}
		if mutation {
			if r.Mutability != "permission_backed_mutation" || r.AuthorityMode != "target_host_permission" || r.RecoveryCommandID == nil || *r.RecoveryCommandID != "qxcmd:symphony:sclv.warning.reopen" {
				t.Fatalf("wrong authority/recovery: %+v", r)
			}
		} else if r.Mutability != "read_only" {
			t.Fatalf("query claims mutation: %+v", r)
		}
	}
	if seen != 4 {
		t.Fatalf("found %d SCLV warning commands", seen)
	}
}

// Synthetic protocol evidence, independent of workstation logs and incidents.
func sclvWarningFixture(t *testing.T, record string) validation.Result {
	t.Helper()
	findings := make([]validation.Finding, 0, 3)
	for _, rule := range []string{"sclv_reference.historical_path_absent", "sclv_skvi_reference.historical", "unrelated.warning"} {
		f := validation.Finding{Category: "warning", Scope: "active", RuleID: rule, Detail: "record_id=" + record + " path=retired.md", Attributes: map[string]string{"record_id": record, "path": "retired.md"}}
		f.SubjectID = fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(f.Category+"\n"+rule+"\n"+f.Scope+"\n"+f.Detail)))
		f.OccurrenceID = f.SubjectID
		findings = append(findings, f)
	}
	result := validation.Result{Protocol: validation.ResultProtocol, FormatVersion: 1, Evidence: validation.Evidence{ValidatorID: "symphony-validator", ValidatorVersion: "0.1.0-dev", RepositoryIdentityDigest: commandManifestTestDigest, Outcome: "pass", Findings: findings, Summary: validation.Summary{Total: 3, Warning: 3}}}
	digestWithout := func(value any, field string) string {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		var object map[string]any
		if err := json.Unmarshal(data, &object); err != nil {
			t.Fatal(err)
		}
		delete(object, field)
		data, err = json.Marshal(object)
		if err != nil {
			t.Fatal(err)
		}
		return fmt.Sprintf("sha256:%x", sha256.Sum256(data))
	}
	result.Evidence.EvidenceDigest = digestWithout(result.Evidence, "evidence_digest")
	result.ResultDigest = digestWithout(result, "result_digest")
	return result
}
