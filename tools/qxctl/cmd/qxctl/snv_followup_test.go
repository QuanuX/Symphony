package main

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/snvstate"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/ssiagclient"
)

func TestSNVFollowupErrorHelper(t *testing.T) {
	if os.Getenv("QXCTL_SNV_FOLLOWUP_ERROR_HELPER") != "1" {
		return
	}
	root, err := newRootCommand()
	if err != nil {
		os.Exit(125)
	}
	args := []string{"snv", "names", "validate", "--json"}
	if os.Getenv("QXCTL_SNV_FOLLOWUP_LEGACY") == "1" {
		args = []string{"scv", "schema", "--json"}
	}
	command, _, err := root.Find(args[:len(args)-1])
	if err != nil {
		os.Exit(125)
	}
	failure := &knowledgeengine.ProcessError{Code: os.Getenv("QXCTL_SNV_FOLLOWUP_CODE"), Message: "private-marker\x1b[31m"}
	var selected error = failure
	if os.Getenv("QXCTL_SNV_FOLLOWUP_BOUNDARY") == "1" {
		selected = snvstate.Refusal(failure.Code, errors.New(failure.Message))
	}
	os.Exit(finishCommandError(root, command, args, snvSafeError(selected)))
}

func TestSNVFollowupTypedErrors(t *testing.T) {
	cases := []struct {
		code     string
		status   int
		admitted bool
	}{
		{"scnv.expected_evidence_conflict", 4, true}, {"snv.snapshot_conflict", 4, true},
		{"reference_conflict", 4, true}, {"scnv.capacity_exceeded", 2, true},
		{"invalid_input", 2, true}, {"snv.deadline_exceeded", 3, true},
		{"snv.state_conflict", 4, true}, {"snv.intent_conflict", 4, true}, {"snv.capacity_exceeded", 2, true}, {"snv.authority_denied", 5, true}, {"snv.authority_unavailable", 3, true}, {"snv.authority_expired", 5, true}, {"snv.recovery_required", 6, true}, {"snv.installation_drift", 4, true}, {"snv.candidate_drift", 4, true}, {"snv.unsafe_state", 2, true}, {"snv.state_busy", 3, true},
		{"private-marker", 1, false}, {"scnv.private_marker", 1, false},
	}
	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestSNVFollowupErrorHelper$")
			cmd.Env = append(os.Environ(), "QXCTL_SNV_FOLLOWUP_ERROR_HELPER=1", "QXCTL_SNV_FOLLOWUP_CODE="+tc.code)
			if strings.HasPrefix(tc.code, "snv.") && tc.code != "snv.snapshot_conflict" && tc.code != "snv.deadline_exceeded" {
				cmd.Env = append(cmd.Env, "QXCTL_SNV_FOLLOWUP_BOUNDARY=1")
			}
			output, err := cmd.CombinedOutput()
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ExitCode() != tc.status {
				t.Fatalf("exit got %v: %s", err, output)
			}
			var envelope cliErrorEnvelope
			if json.Unmarshal(output, &envelope) != nil || envelope.ExitCode != tc.status || envelope.Protocol != cliErrorProtocol {
				t.Fatalf("invalid refusal: %s", output)
			}
			if strings.Contains(string(output), "private-marker") || strings.Contains(string(output), "private_marker") || strings.Contains(string(output), "\\u001b") {
				t.Fatalf("caller diagnostics leaked: %s", output)
			}
			if tc.admitted && strings.HasPrefix(tc.code, "snv.") && envelope.Error.Code != "operation_refused" {
				t.Fatal("local SNV refusal falsely attributed to native engine", string(output))
			}
			if tc.admitted {
				if envelope.Error.EngineCode == nil || *envelope.Error.EngineCode != tc.code {
					t.Fatalf("category lost: %s", output)
				}
			} else if envelope.Error.EngineCode != nil {
				t.Fatal("unknown code admitted")
			}
		})
	}
	// Domain category admission must not change legacy SCV exit behavior.
	cmd := exec.Command(os.Args[0], "-test.run=^TestSNVFollowupErrorHelper$")
	cmd.Env = append(os.Environ(), "QXCTL_SNV_FOLLOWUP_ERROR_HELPER=1", "QXCTL_SNV_FOLLOWUP_CODE=scnv.expected_evidence_conflict", "QXCTL_SNV_FOLLOWUP_LEGACY=1")
	output, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 1 {
		t.Fatalf("legacy status changed: %v %s", err, output)
	}
}

func TestSNVFollowupHeadPinning(t *testing.T) {
	digest := "sha256:" + strings.Repeat("a", 64)
	head := map[string]any{"digest": digest}
	for _, options := range []snvOptions{{mode: "export_chunk"}, {mode: "history", offset: 1}} {
		if !errors.Is(snvInspectHeadBinding(options, head), errUsageOnly) {
			t.Fatal("continuation admitted without captured revision")
		}
		options.expectedHead = digest
		if err := snvInspectHeadBinding(options, head); err != nil {
			t.Fatal(err)
		}
		options.expectedHead = "sha256:" + strings.Repeat("b", 64)
		var failure *knowledgeengine.ProcessError
		if !errors.As(snvInspectHeadBinding(options, head), &failure) || failure.Code != "snv.snapshot_conflict" {
			t.Fatal("stale captured revision admitted")
		}
	}
	if err := snvInspectHeadBinding(snvOptions{mode: "history", offset: 0}, head); err != nil {
		t.Fatal("first page requires unnecessary pin")
	}
	if err := snvInspectHeadBinding(snvOptions{mode: "projection", expectedHead: digest}, head); err != nil {
		t.Fatal(err)
	}
	if err := runSNVInspect(snvOptions{input: "private-marker", expectedHead: digest}); !errors.Is(err, errUsageOnly) {
		t.Fatal("supplied input accepted named-view revision flag")
	}
	root, err := newRootCommand()
	if err != nil {
		t.Fatal(err)
	}
	command, _, err := root.Find([]string{"snv", "inspect"})
	if err != nil {
		t.Fatal(err)
	}
	if command.Flags().Lookup("expected-head-digest") == nil {
		t.Fatal("snapshot pin missing from command flags")
	}
	out, status := invokeCLI(t, "snv", "inspect", "--help")
	if status != 0 || !strings.Contains(out, "--expected-head-digest") {
		t.Fatalf("pinning missing from help: %d %s", status, out)
	}
}

func TestSNVFollowupBindingRoleGrammar(t *testing.T) {
	root, err := newRootCommand()
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := commandregistry.BuildExpected(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, operation := range []string{"inspect", "bind", "unbind", "list", "doctor", "migrate"} {
		id := "qxcmd:symphony:knowledge.engines." + operation
		found := false
		for _, record := range manifest.Commands {
			if record.CommandID != id {
				continue
			}
			found = true
			required := operation == "inspect" || operation == "bind" || operation == "unbind"
			if record.Grammar == nil || strings.Contains(*record.Grammar, "<role>") != required {
				t.Fatalf("role grammar contradicts required Args for %s: %v", id, record.Grammar)
			}
		}
		if !found {
			t.Fatal("binding route absent", id)
		}
	}
	for _, operation := range []string{"inspect", "bind", "unbind"} {
		output, status := invokeCLI(t, "knowledge", "engines", operation, "--help")
		if status != 0 || !strings.Contains(output, "<role>") {
			t.Fatalf("required role not discoverable: %d %s", status, output)
		}
	}
}

func TestSNVFollowupAuthorityClassification(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	tops := "00000000-0000-4000-8000-000000000001"
	request := ssiagclient.AuthorizationRequest{RequestID: "00000000-0000-4000-8000-000000000002", CorrelationID: "00000000-0000-4000-8000-000000000003", Operation: "symphony.snv.view.select", Resource: snvstate.Resource(tops, "test-view"), Audience: "qxctl", Scope: "tops:" + tops, RequestedAt: now, RequestedExpiresAt: now.Add(time.Minute)}
	hash := "sha256:" + strings.Repeat("a", 64)
	decision := ssiagclient.AuthorizationDecision{Schema: "symphony.ssiag.authorization-decision.v1", DecisionID: "test-denied", RequestID: request.RequestID, CorrelationID: request.CorrelationID, TOPSID: tops, Subject: ssiagclient.DecisionSubject{ID: "host-501", Kind: "host", Authority: "unix_peer_credentials"}, Target: ssiagclient.DecisionTarget{Operation: request.Operation, Resource: request.Resource, Audience: request.Audience, Scope: request.Scope}, Effect: "deny", ReasonCode: "symphony.ssiag.policy.no-grant", PolicyDigest: hash, ConfigDigest: hash, DecidedAt: now}
	classify := func(d ssiagclient.AuthorizationDecision, want string) {
		t.Helper()
		var e *snvstate.BoundaryError
		if !errors.As(snvSelectionDecision(d, request, tops), &e) || e.Code != want {
			t.Fatalf("expected %s got %v", want, e)
		}
	}
	classify(decision, "snv.authority_denied")
	changed := decision
	changed.Target.Resource = "private-marker"
	classify(changed, "snv.authority_unavailable")
	changed = decision
	changed.CorrelationID = "private-marker"
	classify(changed, "snv.authority_unavailable")
	changed = decision
	changed.Subject.Authority = "claimed-authority"
	classify(changed, "snv.authority_unavailable")
	changed = decision
	changed.Effect = "allow"
	classify(changed, "snv.authority_unavailable")
	changed = decision
	changed.DecidedAt = now.Add(-time.Hour)
	classify(changed, "snv.authority_unavailable")
	err := snvSafeError(snvstate.Refusal("snv.recovery_required", snvstate.Refusal("snv.authority_denied", errors.New("private-marker"))))
	var pe *knowledgeengine.ProcessError
	if !errors.As(err, &pe) || pe.Code != "snv.authority_denied" {
		t.Fatal("known guard refusal lost")
	}
	err = snvSafeError(snvstate.Refusal("snv.recovery_required", errors.New("authorization denied private-marker")))
	if !errors.As(err, &pe) || pe.Code != "snv.recovery_required" {
		t.Fatal("opaque failure was classified as denial")
	}
}

func TestSNVFollowupAuthorizationHistoryPaging(t *testing.T) {
	root := os.Getenv("SYMPHONY_SNV_AUTH_HISTORY_FIXTURE")
	if root == "" {
		t.Skip("set exact disposable2000-record history fixture")
	}
	store, err := snvstate.NewView(root, "00000000-0000-4000-8000-000000000001", "test-view")
	if err != nil {
		t.Fatal(err)
	}
	var snapshot string
	read := func(page snvAuthorizationPage) (map[string]any, error) {
		var output json.RawMessage
		e := store.WithRead(func(tx *snvstate.Transaction) error {
			var e error
			output, e = snvStoreResult("state.status", tx, "history", nil, page)
			return e
		})
		if e != nil {
			return nil, e
		}
		return snvObject(output)
	}
	first, err := read(snvAuthorizationPage{limit: 16})
	if err != nil {
		t.Fatal(err)
	}
	snapshot = first["journal_digest"].(string)
	a := first["attempt"].(map[string]any)
	if a["prior_authorization_count"] != int64(2000) || len(a["prior_authorizations"].([]any)) != 16 || a["next_prior_authorization_offset"] != int64(16) {
		t.Fatal("default status lost history count or paging", a)
	}
	if _, err := read(snvAuthorizationPage{offset: 16, limit: 7}); !errors.Is(err, errUsageOnly) {
		t.Fatal("unbound continuation admitted")
	}
	if _, err := read(snvAuthorizationPage{offset: 16, limit: 7, expectedJournal: "sha256:" + strings.Repeat("b", 64)}); err == nil {
		t.Fatal("another history revision admitted")
	}
	next, err := read(snvAuthorizationPage{offset: 16, limit: 7, expectedJournal: snapshot})
	if err != nil {
		t.Fatal(err)
	}
	a = next["attempt"].(map[string]any)
	records := a["prior_authorizations"].([]any)
	if len(records) != 7 || a["prior_authorization_offset"] != int64(16) || a["next_prior_authorization_offset"] != int64(23) {
		t.Fatal("history continuation omitted page metadata")
	}
	if err := store.WithRead(func(tx *snvstate.Transaction) error {
		original, _ := tx.Attempt("history")
		for i, r := range records {
			if !snvSame(r, original.PriorAuthorizations[16+i]) {
				t.Fatal("paging changed retained policy evidence")
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	last, err := read(snvAuthorizationPage{offset: 1999, limit: 128, expectedJournal: snapshot})
	if err != nil {
		t.Fatal(err)
	}
	a = last["attempt"].(map[string]any)
	if len(a["prior_authorizations"].([]any)) != 1 || a["next_prior_authorization_offset"] != nil {
		t.Fatal("last page completeness wrong")
	}
	// The same snapshot remains intact; reading pages cannot prune history.
	if err := store.WithRead(func(tx *snvstate.Transaction) error {
		original, _ := tx.Attempt("history")
		if len(original.PriorAuthorizations) != 2000 || tx.Snapshot().Digest != snapshot {
			t.Fatal("history observation mutated acknowledged data")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	rootCommand, err := newRootCommand()
	if err != nil {
		t.Fatal(err)
	}
	command, _, err := rootCommand.Find([]string{"snv", "state", "status"})
	if err != nil {
		t.Fatal(err)
	}
	for _, flag := range []string{"authorization-offset", "authorization-limit", "expected-journal-digest"} {
		if command.Flags().Lookup(flag) == nil {
			t.Fatal("page flag undiscoverable", flag)
		}
	}
}
func TestSNVFollowupOutputBound(t *testing.T) {
	// Refusal happens before fmt.Println, so no partial successful JSON document
	// can precede the structured error. This tests the full pretty output budget.
	var refusal *snvstate.BoundaryError
	if !errors.As(snvPrintJSON(map[string]any{"bounded": strings.Repeat("x", 4*1024*1024)}), &refusal) || refusal.Code != "snv.capacity_exceeded" {
		t.Fatal("oversized success was emitted")
	}
}
