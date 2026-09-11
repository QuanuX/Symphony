package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/scvcorpus"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/scvworkflow"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func workflowStore(t *testing.T) scvworkflow.Store {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s, err := scvworkflow.New(root, ssiagTestTOPSID)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func workflowValue(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	v, err := scvworkflow.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func workflowClone(t *testing.T, v map[string]any) map[string]any {
	t.Helper()
	raw, err := scvworkflow.Canonical(v)
	if err != nil {
		t.Fatal(err)
	}
	return workflowValue(t, raw)
}
func mockWorkflow(t *testing.T) (*workflowRunner, scvworkflow.Store, map[string]any) {
	t.Helper()
	s := workflowStore(t)
	inst := knowledgeengine.Installation{Role: "scv", Version: "0.4.0-dev", Prefix: s.Root, ExecutableDigest: "fixture-executable", ReceiptDigest: "fixture-receipt"}
	r := &workflowRunner{installation: inst}
	r.inspect = func(role, prefix, version string) (knowledgeengine.Installation, error) {
		v := inst
		v.Role = role
		v.Prefix = prefix
		v.Version = version
		return v, nil
	}
	r.owner = func(owner knowledgeengine.Installation, op string, input any) (json.RawMessage, error) {
		current, _ := r.inspect(owner.Role, owner.Prefix, owner.Version)
		if current != owner {
			return nil, fmt.Errorf("exact installation changed")
		}
		protocol, ok := knowledgeengine.SCVResultProtocol(op)
		if !ok {
			return nil, fmt.Errorf("unknown mock operation")
		}
		// This fixture exercises bookkeeping only; installed-owner tests below
		// exercise provider semantics and consumer validation without this stub.
		return scvworkflow.Seal(map[string]any{"protocol": protocol, "domain": owner.Role, "input": input})
	}
	raw, err := r.artifact(s, "import", map[string]any{"operation": "knowledge_interpret", "input": map[string]any{"fixture": "caller knowledge"}, "result": nil})
	if err != nil {
		t.Fatal(err)
	}
	ref := workflowValue(t, raw)["digest"]
	request := map[string]any{"operation_id": "run", "corpus": nil, "profile_bindings": []any{}, "interpretation_refs": []any{}, "knowledge_refs": []any{ref}, "selection_policy": nil, "query_time": "2026-09-10T12:00:00Z", "connections": []any{}, "prior_evaluation_ref": nil}
	return r, s, request
}
func TestSCVWorkflowResumesEveryDurableBoundaryWithoutReplacingIntent(t *testing.T) {
	for _, boundary := range []string{"prepared", "evaluate_input", "evaluate_artifact", "evaluate_result", "complete"} {
		t.Run(boundary, func(t *testing.T) {
			r, s, request := mockWorkflow(t)
			interrupted := false
			r.afterCheckpoint = func(name string) error {
				if name == boundary && !interrupted {
					interrupted = true
					return errors.New("injected crash after durable " + name)
				}
				return nil
			}
			if _, err := r.execute(s, "run", request, ""); err == nil || !interrupted {
				t.Fatal("boundary did not interrupt", err)
			}
			status, err := r.execute(s, "status", map[string]any{"operation_id": "run"}, "")
			if err != nil {
				t.Fatal(err)
			}
			if workflowValue(t, status)["validation"] != "sealed_checkpoints_only" {
				t.Fatal("status implied native replay")
			}
			r.afterCheckpoint = nil
			recovered, err := r.execute(s, "recover", map[string]any{"operation_id": "run"}, "")
			if err != nil {
				t.Fatal(err)
			}
			again, err := r.execute(s, "run", request, "")
			if err != nil {
				t.Fatal(err)
			}
			if !scvworkflow.Same(recovered, again) {
				t.Fatal("lost-response replay changed retained result")
			}
			changed := workflowClone(t, request)
			changed["query_time"] = "2026-09-10T12:00:01Z"
			if _, err = r.execute(s, "run", changed, ""); err == nil {
				t.Fatal("same operation ID accepted changed query time")
			}
			r.installation.ExecutableDigest = "replacement"
			if _, err = r.execute(s, "recover", map[string]any{"operation_id": "run"}, ""); err == nil {
				t.Fatal("recovery accepted changed executable")
			}
		})
	}
}
func workflowRunPath(s scvworkflow.Store, id string) string {
	d, _ := knowledgeengine.SCVDigest(map[string]any{"operation_id": id})
	return filepath.Join(s.Root, "symphony", "qxctl", "scv", "workflows-v1", s.TOPSID, "runs", strings.TrimPrefix(d, "sha256:"), "run.json")
}
func TestSCVWorkflowRejectsResealedCheckpointSubstitutionAndCorruptPayload(t *testing.T) {
	r, s, request := mockWorkflow(t)
	if _, err := r.execute(s, "run", request, ""); err != nil {
		t.Fatal(err)
	}
	path := workflowRunPath(s, "run")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	run, err := scvworkflow.ReadRun(original, "run")
	if err != nil {
		t.Fatal(err)
	}
	var replacement string
	if err = s.With("", true, func(session *scvworkflow.Session, _ *scvworkflow.Run) error {
		raw, err := scvworkflow.Seal(map[string]any{"protocol": "symphony.qxctl.scv-workflow-payload.v1", "input": map[string]any{"unrelated": "input"}})
		if err != nil {
			return err
		}
		replacement, err = session.Put("payloads", raw)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	c := run.Stages["evaluate"]
	c.InputRef = replacement
	run.Stages["evaluate"] = c
	changed, err := scvworkflow.Seal(run)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, changed, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err = r.execute(s, "status", map[string]any{"operation_id": "run"}, ""); err == nil {
		t.Fatal("status accepted checkpoint/input mismatch")
	}
	if _, err = r.execute(s, "recover", map[string]any{"operation_id": "run"}, ""); err == nil {
		t.Fatal("recovery accepted checkpoint/input mismatch")
	}
	if err = os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	run, _ = scvworkflow.ReadRun(original, "run")
	payloadPath := filepath.Join(s.Root, "symphony", "qxctl", "scv", "workflows-v1", s.TOPSID, "payloads", strings.TrimPrefix(run.Stages["evaluate"].InputRef, "sha256:")+".json")
	if err = os.WriteFile(payloadPath, []byte(`{"corrupt":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err = r.execute(s, "recover", map[string]any{"operation_id": "run"}, ""); err == nil {
		t.Fatal("corrupt retained input accepted")
	}
}
func TestSCVWorkflowReconstructsPendingStageBeforeReplay(t *testing.T) {
	r, s, request := mockWorkflow(t)
	r.afterCheckpoint = func(name string) error {
		if name == "evaluate_input" {
			return errors.New("stop before owner")
		}
		return nil
	}
	if _, err := r.execute(s, "run", request, ""); err == nil {
		t.Fatal("did not stop")
	}
	r.afterCheckpoint = nil
	path := workflowRunPath(s, "run")
	raw, _ := os.ReadFile(path)
	run, err := scvworkflow.ReadRun(raw, "run")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.With("", true, func(session *scvworkflow.Session, _ *scvworkflow.Run) error {
		payload, err := scvworkflow.Seal(map[string]any{"protocol": "symphony.qxctl.scv-workflow-payload.v1", "input": map[string]any{"unrelated": "pending input"}})
		if err != nil {
			return err
		}
		ref, err := session.Put("payloads", payload)
		run.Stages["evaluate"] = scvworkflow.Checkpoint{InputRef: ref}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	changed, _ := scvworkflow.Seal(run)
	if err = os.WriteFile(path, changed, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err = r.execute(s, "recover", map[string]any{"operation_id": "run"}, ""); err == nil || !strings.Contains(err.Error(), "input differs") {
		t.Fatal("pending stage was not reconstructed from pinned request", err)
	}
}

func TestSCVWorkflowStatusRejectsMatchedUnrelatedCheckpointPair(t *testing.T) {
	r, s, request := mockWorkflow(t)
	if _, err := r.execute(s, "run", request, ""); err != nil {
		t.Fatal(err)
	}
	other := workflowClone(t, request)
	other["operation_id"] = "other"
	other["query_time"] = "2026-09-10T12:00:01Z"
	if _, err := r.execute(s, "run", other, ""); err != nil {
		t.Fatal(err)
	}
	firstRaw, err := os.ReadFile(workflowRunPath(s, "run"))
	if err != nil {
		t.Fatal(err)
	}
	secondRaw, err := os.ReadFile(workflowRunPath(s, "other"))
	if err != nil {
		t.Fatal(err)
	}
	first, err := scvworkflow.ReadRun(firstRaw, "run")
	if err != nil {
		t.Fatal(err)
	}
	second, err := scvworkflow.ReadRun(secondRaw, "other")
	if err != nil {
		t.Fatal(err)
	}
	first.Stages["evaluate"] = second.Stages["evaluate"]
	changed, err := scvworkflow.Seal(first)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(workflowRunPath(s, "run"), changed, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, op := range []string{"status", "recover"} {
		if _, err = r.execute(s, op, map[string]any{"operation_id": "run"}, ""); err == nil || !strings.Contains(err.Error(), "input differs") {
			t.Fatalf("%s accepted valid but unrelated input/result pair: %v", op, err)
		}
	}
}

func TestSCVWorkflowRejectsIgnoredPolicyAndRelativeCorpusRoot(t *testing.T) {
	r, s, request := mockWorkflow(t)
	request["selection_policy"] = map[string]any{"policy_id": "otherwise ignored"}
	if _, err := r.execute(s, "run", request, ""); err == nil {
		t.Fatal("unused caller policy silently ignored")
	}
	request["selection_policy"] = nil
	if _, err := r.execute(s, "run", request, "relative/root"); err == nil {
		t.Fatal("relative corpus root pinned")
	}
	if _, err := os.Stat(workflowRunPath(s, "run")); !os.IsNotExist(err) {
		t.Fatal("invalid initial request created a run journal")
	}
}
func TestSCVArtifactsRejectSuppliedResultMismatchAndReplayOriginalOwner(t *testing.T) {
	r, s, _ := mockWorkflow(t)
	foreign := r.installation
	foreign.Role = "schv-gcp"
	foreign.Version = "0.3.0-dev"
	r.installation = foreign
	raw, err := r.artifact(s, "import", map[string]any{"operation": "knowledge_interpret", "input": map[string]any{"fixture": "foreign owner"}, "result": nil})
	if err != nil {
		t.Fatal(err)
	}
	record, err := scvworkflow.ReadRecord(raw)
	if err != nil {
		t.Fatal(err)
	}
	r.installation.Role = "scv"
	r.installation.Version = "0.4.0-dev"
	show, err := r.artifact(s, "show", map[string]any{"record_digest": record.Digest})
	if err != nil {
		t.Fatal(err)
	}
	if !scvworkflow.Same(raw, show) {
		t.Fatal("show replaced original owner provenance")
	}
	if _, err = r.artifact(s, "import", map[string]any{"operation": "knowledge_interpret", "input": map[string]any{"fixture": "different"}, "result": json.RawMessage(record.Artifact)}); err == nil {
		t.Fatal("import accepted result from different invocation")
	}
	originalInspect := r.inspect
	r.inspect = func(role, prefix, version string) (knowledgeengine.Installation, error) {
		inst, err := originalInspect(role, prefix, version)
		if role == "schv-gcp" {
			inst.ReceiptDigest = "changed"
		}
		return inst, err
	}
	if _, err = r.artifact(s, "show", map[string]any{"record_digest": record.Digest}); err == nil {
		t.Fatal("retained artifact accepted changed original owner receipt")
	}
	if _, err = r.artifact(s, "list", map[string]any{"after_digest": nil, "limit": json.Number("128")}); err != nil {
		t.Fatal("metadata-only listing required live owner", err)
	}
}

func installedWorkflowFixture(t *testing.T) (*workflowRunner, scvworkflow.Store, *corpusRunner, scvcorpus.Store, map[string]any) {
	t.Helper()
	prefix := os.Getenv("SYMPHONY_SCV_WORKFLOW_PREFIX")
	if prefix == "" {
		t.Skip("requires exact installed 0.4.0-dev SCV owners")
	}
	s := workflowStore(t)
	options := scvOptions{domain: "scv", prefix: prefix, version: "0.4.0-dev", topsID: ssiagTestTOPSID, repository: s.Root}
	r, err := newWorkflowRunner(options, true)
	if err != nil {
		t.Fatal(err)
	}
	cr, err := newCorpusRunner(options)
	if err != nil {
		t.Fatal(err)
	}
	cr.now = func() time.Time { return time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC) }
	cs, err := scvcorpus.New(s.Root, ssiagTestTOPSID, "scv")
	if err != nil {
		t.Fatal(err)
	}
	source := corpusSource(t, cr, "profile-doc")
	capture := corpusCapture(t, cr, source, "Fixture service. Port: 7844 UDP.", "complete")
	snapshot := executeCorpus(t, cr, cs, "import", corpusRequest("fixture", "caller-corpus", nil, map[string]any{"member_id": "docs", "capture": capture}))
	scope := map[string]any{"service": "synthetic-fixture"}
	draft := map[string]any{"protocol": "symphony.scv.interpretation-profile.v1", "profile_id": "fixture-profile", "profile_version": "1", "provider_id": "aws", "source_id": "profile-doc", "locator_id": "docs", "media_types": []any{"text/plain"}, "authored_by": "unit test author", "rationale": "Synthetic fixture, no provider truth", "rules": []any{map[string]any{"rule_id": "port", "claim_id": "port", "subject": "endpoint", "predicate": "requires-port", "scope": scope, "statement_kind": "requirement", "dependencies": []any{}, "context": []any{"Fixture service."}, "extractor": map[string]any{"kind": "delimited", "prefix": "Port: ", "suffix": " UDP.", "type": "integer", "unit": "port"}}}}
	profile, err := r.artifact(s, "import", map[string]any{"operation": "profile_prepare", "input": map[string]any{"profile": draft}, "result": nil})
	if err != nil {
		t.Fatal(err)
	}
	profileRef := workflowValue(t, profile)["digest"]
	policy := map[string]any{"policy_id": "caller-policy", "max_age_seconds": 7200, "allowed_statement_kinds": []any{"requirement"}, "partial_capture": "exclude"}
	checks := []any{map[string]any{"check_id": "port", "importance": "required", "left": map[string]any{"claim_id": "port", "subject": "endpoint", "scope": scope}, "operator": "eq", "right": map[string]any{"kind": "literal", "value": map[string]any{"type": "integer", "value": 7844, "unit": "port"}}}}
	request := map[string]any{"operation_id": "first", "corpus": map[string]any{"domain": "scv", "snapshot_digest": snapshot["snapshot_digest"], "member_ids": []any{"docs"}, "selection": "latest_attempt", "max_age_seconds": nil}, "profile_bindings": []any{map[string]any{"profile_ref": profileRef, "member_id": "docs"}}, "interpretation_refs": []any{}, "knowledge_refs": []any{}, "selection_policy": policy, "query_time": "2026-09-10T12:00:01Z", "connections": []any{map[string]any{"connection_id": "caller-connection", "from_subject": "endpoint", "to_subject": "consumer", "checks": checks}}, "prior_evaluation_ref": nil}
	return r, s, cr, cs, request
}
func executeWorkflow(t *testing.T, r *workflowRunner, s scvworkflow.Store, operation string, input map[string]any, corpusRoot string) map[string]any {
	t.Helper()
	raw, err := r.execute(s, operation, input, corpusRoot)
	if err != nil {
		t.Fatal(err)
	}
	return workflowValue(t, raw)
}
func retainedArtifact(t *testing.T, r *workflowRunner, s scvworkflow.Store, ref any) map[string]any {
	t.Helper()
	raw, err := r.artifact(s, "show", map[string]any{"record_digest": ref})
	if err != nil {
		t.Fatal(err)
	}
	return workflowValue(t, raw)["artifact"].(map[string]any)
}
func TestInstalledSCVWorkflowNativeRecoveryRefreshReassessmentAndComposition(t *testing.T) {
	r, s, cr, cs, request := installedWorkflowFixture(t)
	interrupted := false
	r.afterCheckpoint = func(name string) error {
		if name == "interpret_result" && !interrupted {
			interrupted = true
			return errors.New("interrupt after native interpretation")
		}
		return nil
	}
	if _, err := r.execute(s, "run", request, cs.Root); err == nil || !interrupted {
		t.Fatal("native interpretation checkpoint did not interrupt", err)
	}
	r.afterCheckpoint = nil
	first := executeWorkflow(t, r, s, "recover", map[string]any{"operation_id": "first"}, "")
	before := retainedArtifact(t, r, s, first["evaluation_ref"])
	if before["connections"].([]any)[0].(map[string]any)["status"] != "satisfied" {
		t.Fatal("native connection should be satisfied")
	}
	replay := executeWorkflow(t, r, s, "run", request, cs.Root)
	if !scvcorpus.Same(first, replay) {
		t.Fatal("completed native replay changed result")
	}
	changed := workflowClone(t, request)
	changed["query_time"] = "2026-09-10T12:00:02Z"
	if _, err := r.execute(s, "run", changed, cs.Root); err == nil {
		t.Fatal("same native operation ID accepted new time")
	}
	source := corpusSource(t, cr, "profile-doc")
	capture := corpusCapture(t, cr, source, "Fixture service. Port: 7845 UDP.", "complete")
	snapshot := executeCorpus(t, cr, cs, "import", corpusRequest("refresh", "caller-corpus", request["corpus"].(map[string]any)["snapshot_digest"], map[string]any{"member_id": "docs", "capture": capture}))
	secondRequest := workflowClone(t, request)
	secondRequest["operation_id"] = "changed"
	secondRequest["corpus"].(map[string]any)["snapshot_digest"] = snapshot["snapshot_digest"]
	secondRequest["prior_evaluation_ref"] = first["evaluation_ref"]
	second := executeWorkflow(t, r, s, "run", secondRequest, cs.Root)
	after := retainedArtifact(t, r, s, second["evaluation_ref"])
	if after["connections"].([]any)[0].(map[string]any)["status"] != "contradicted" {
		t.Fatal("changed source requirement not propagated")
	}
	reassessment := retainedArtifact(t, r, s, second["reassessment_ref"])
	axes := reassessment["change_axes"].(map[string]any)
	if axes["captures"] != true || axes["profiles"] != false || axes["query_time"] != false {
		t.Fatal("refresh changed unselected axes")
	}
	// Reuse a provider-owned interpretation without forcing a new corpus scan.
	run := first["run"].(map[string]any)
	interpretationRef := run["stages"].(map[string]any)["interpret"].(map[string]any)["result_ref"]
	compose := workflowClone(t, request)
	compose["operation_id"] = "compose"
	compose["corpus"] = nil
	compose["selection_policy"] = nil
	compose["profile_bindings"] = []any{}
	compose["interpretation_refs"] = []any{interpretationRef}
	composed := executeWorkflow(t, r, s, "run", compose, "")
	if retainedArtifact(t, r, s, composed["evaluation_ref"])["digest"] != before["digest"] {
		t.Fatal("retained interpretation composition changed native evaluation")
	}
	foreignOptions := r.options
	foreignOptions.domain = "schv-aws"
	foreign, err := newWorkflowRunner(foreignOptions, true)
	if err != nil {
		t.Fatal(err)
	}
	initialRecord, err := r.artifact(s, "show", map[string]any{"record_digest": interpretationRef})
	if err != nil {
		t.Fatal(err)
	}
	providerRecord, err := foreign.artifact(s, "import", map[string]any{"operation": "provider_interpret", "input": workflowValue(t, initialRecord)["input"], "result": nil})
	if err != nil {
		t.Fatal(err)
	}
	providerValue := workflowValue(t, providerRecord)
	if providerValue["installation"].(map[string]any)["Role"] != "schv-aws" {
		t.Fatal("provider owner provenance missing")
	}
	compose["operation_id"] = "provider-compose"
	compose["interpretation_refs"] = []any{providerValue["digest"]}
	providerComposed := executeWorkflow(t, r, s, "run", compose, "")
	if !scvcorpus.Same(retainedArtifact(t, r, s, providerComposed["evaluation_ref"])["connections"], before["connections"]) {
		t.Fatal("independently installed provider composition changed connection checks")
	}
	compose["operation_id"] = "caller-selected-later-time"
	compose["query_time"] = "2026-09-11T12:00:00Z"
	stale := executeWorkflow(t, r, s, "run", compose, "")
	if retainedArtifact(t, r, s, stale["evaluation_ref"])["connections"].([]any)[0].(map[string]any)["status"] != "unresolved" {
		t.Fatal("expired evidence was not unresolved at caller-selected later time")
	}
	for _, namespace := range []string{"sources-v1", "graphs-v1"} {
		if _, err := os.Stat(filepath.Join(s.Root, "symphony", "qxctl", "scv", namespace)); !os.IsNotExist(err) {
			t.Fatal("workflow touched selected authority head")
		}
	}
}

func TestInstalledSCVCorpusMetadataQueryExceedsExportBound(t *testing.T) {
	r, s := corpusFixture(t)
	members := []any{}
	for i := 0; i < 17; i++ {
		id := fmt.Sprintf("doc-%02d", i)
		source := corpusSource(t, r, id)
		members = append(members, map[string]any{"member_id": id, "capture": corpusCapture(t, r, source, "small synthetic fixture", "complete")})
	}
	snapshot := executeCorpus(t, r, s, "import", corpusRequest("query-inventory", "inventory", nil, members...))
	input := map[string]any{"snapshot_digest": snapshot["snapshot_digest"], "member_ids": []any{}, "selection": "latest_attempt", "query_time": "2026-09-10T12:00:00Z", "max_age_seconds": nil}
	query := executeCorpus(t, r, s, "query", input)
	if query["protocol"] != "symphony.scv.corpus-query.v1" || len(query["members"].([]any)) != 17 {
		t.Fatal("metadata query did not expose 17 members")
	}
	if _, err := r.execute(s, "export", input); err == nil || !strings.Contains(err.Error(), "16 captures") {
		t.Fatal("capture export bound changed", err)
	}
}
