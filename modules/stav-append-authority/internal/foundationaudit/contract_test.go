package foundationaudit

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	stav "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
)

func testRequest() Request {
	return Request{Schema: Schema, Component: "ssiag", Surface: "supervisor", Scope: "user", TOPSID: "123e4567-e89b-42d3-a456-426614174000", OperationID: "lifecycle:test", RequestID: "123e4567-e89b-42d3-a456-426614174001", CorrelationID: "123e4567-e89b-42d3-a456-426614174002", PlanDigest: "sha256:" + strings.Repeat("a", 64), PriorStateDigest: "sha256:" + strings.Repeat("b", 64), DesiredState: "native_installed_stopped", AuditMode: "ordinary", Phase: "intent"}
}
func committed(t *testing.T, r Request) Response {
	t.Helper()
	candidate, err := Candidate(r, uint32(os.Geteuid()))
	if err != nil {
		t.Fatal(err)
	}
	digest, err := stav.CandidateDigest(candidate)
	if err != nil {
		t.Fatal(err)
	}
	return Response{Schema: Schema, Phase: r.Phase, RequestDigest: Hash(r), Receipt: stav.Receipt{Schema: stav.SchemaReceipt, TOPSID: r.TOPSID, RequestID: r.RequestID, CandidateDigest: digest, Disposition: "committed", ReasonCode: stav.ReasonReceiptCommitted, Commit: stav.CommitResult{State: "committed", EventID: "123e4567-e89b-42d3-a456-426614174003", EventDigest: "sha256:" + strings.Repeat("c", 64), Sequence: 1, Timestamp: "2026-11-01T12:00:00.000000000Z"}}}
}
func TestReceiptBindsFullIntentAndKernelActor(t *testing.T) {
	r := testRequest()
	response := committed(t, r)
	if err := response.Validate(r, uint32(os.Geteuid())); err != nil {
		t.Fatal(err)
	}
	cases := map[string]func(*Request){
		"target":    func(v *Request) { v.Component = "stav" },
		"plan":      func(v *Request) { v.PlanDigest = "sha256:" + strings.Repeat("d", 64) },
		"prior":     func(v *Request) { v.PriorStateDigest = "sha256:" + strings.Repeat("d", 64) },
		"operation": func(v *Request) { v.OperationID = "another" },
		"desired":   func(v *Request) { v.DesiredState = "native_running" },
		"phase": func(v *Request) {
			v.Phase = "outcome"
			v.ObservedAt = "2026-11-01T12:00:00Z"
			v.ObservedStateDigest = v.PriorStateDigest
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			other := r
			change(&other)
			if response.Validate(other, uint32(os.Geteuid())) == nil {
				t.Fatal("receipt accepted different request")
			}
		})
	}
	if response.Validate(r, uint32(os.Geteuid())+1) == nil {
		t.Fatal("receipt accepted another kernel actor")
	}
	response.Receipt.Commit.Sequence = 0
	if response.Validate(r, uint32(os.Geteuid())) == nil {
		t.Fatal("digest-only/invalid receipt accepted")
	}
}
func TestIntentCannotClaimCompletionAndDecodeIsClosed(t *testing.T) {
	r := testRequest()
	r.ObservedAt = "2026-11-01T12:00:00Z"
	if r.Validate() == nil {
		t.Fatal("intent claimed completed execution")
	}
	r = testRequest()
	r.Phase = "outcome"
	if r.Validate() == nil {
		t.Fatal("outcome without observation accepted")
	}
	for _, data := range []string{`{"schema":"x","schema":"y"}`, `{"actor":"forged"}`} {
		var v Request
		if Decode([]byte(data), &v) == nil {
			t.Fatal("open/duplicate JSON accepted")
		}
	}
}
func TestJournalLostReplyRestartAndOutcomeReconciliation(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "audit")
	r := testRequest()
	j, err := Open(directory, r)
	if err != nil {
		t.Fatal(err)
	}
	calls, commits := 0, 0
	receipts := map[string]Response{}
	submit := func(_ context.Context, request Request) (Response, error) {
		calls++
		persisted, e := Load(directory, r.OperationID)
		if e != nil {
			t.Fatal(e)
		}
		if request.Phase == "outcome" && persisted.Outcome == nil {
			t.Fatal("outcome sent before durable observation")
		}
		response, exists := receipts[request.RequestID]
		if !exists {
			response = committed(t, request)
			receipts[request.RequestID] = response
			commits++
		}
		if calls == 1 {
			return Response{}, errors.New("reply lost after commit")
		}
		return response, nil
	}
	if err = j.CommitIntent(context.Background(), submit); err == nil {
		t.Fatal("lost reply treated as committed")
	}
	j, err = Load(directory, r.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	if j.Intent != nil || j.Completion != nil {
		t.Fatal("unconfirmed receipt persisted")
	}
	if err = j.CommitIntent(context.Background(), submit); err != nil {
		t.Fatal(err)
	}
	if commits != 1 || calls != 2 || j.Completion != nil {
		t.Fatal("retry duplicated event or intent certified outcome")
	}
	if err = j.Observe(r.PriorStateDigest, "2026-11-01T12:00:00Z"); err != nil {
		t.Fatal(err)
	}
	if err = j.Reconcile(context.Background(), nil); err == nil {
		t.Fatal("unavailable audit accepted")
	}
	j, err = Load(directory, r.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	original := Hash(*j.Outcome)
	if err = j.Observe("sha256:"+strings.Repeat("e", 64), "2026-11-02T12:00:00Z"); err != nil {
		t.Fatal(err)
	}
	if Hash(*j.Outcome) != original {
		t.Fatal("restart overwrote original stopped observation")
	}
	if err = j.Reconcile(context.Background(), submit); err != nil {
		t.Fatal(err)
	}
	if commits != 2 || j.Outcome.RequestID == r.RequestID {
		t.Fatal("outcome idempotency not separate")
	}
	if err = j.Reconcile(context.Background(), submit); err != nil {
		t.Fatal(err)
	}
	if calls != 3 {
		t.Fatal("confirmed reconciliation sent again")
	}
	other := r
	other.DesiredState = "native_running"
	if _, err = Open(directory, other); err == nil {
		t.Fatal("operation identity rebound")
	}
}
func TestJournalRejectsReceiptMismatchAndUnsafeFiles(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "audit")
	r := testRequest()
	j, err := Open(directory, r)
	if err != nil {
		t.Fatal(err)
	}
	submit := func(_ context.Context, request Request) (Response, error) {
		v := committed(t, request)
		v.Receipt.CandidateDigest = "sha256:" + strings.Repeat("e", 64)
		return v, nil
	}
	if err = j.CommitIntent(context.Background(), submit); err == nil {
		t.Fatal("wrong receipt accepted")
	}
	loaded, err := Load(directory, r.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Intent != nil {
		t.Fatal("wrong receipt stored")
	}
	path := journalPath(directory, r.OperationID)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(t.TempDir(), "foreign")
	if err = os.WriteFile(foreign, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink(foreign, path); err != nil {
		t.Fatal(err)
	}
	if _, err = Load(directory, r.OperationID); err == nil {
		t.Fatal("symlink journal accepted")
	}
}

func TestJournalRejectsFIFOWithoutWaitingForWriter(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "audit")
	r := testRequest()
	j, err := Open(directory, r)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(j.path); err != nil {
		t.Fatal(err)
	}
	if err = syscall.Mkfifo(j.path, 0600); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, err := Load(directory, r.OperationID); done <- err }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("FIFO journal accepted")
		}
	case <-time.After(time.Second):
		t.Fatal("FIFO journal blocked before regular-file validation")
	}
}
