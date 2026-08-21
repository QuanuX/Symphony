package producer

import (
	"context"
	"fmt"
	"strings"
	"testing"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/outbox"
)

type retryTransport struct{ fail bool }

func (transport *retryTransport) Do(_ context.Context, request stavprotocol.LocalRequest) (stavprotocol.LocalResponse, error) {
	if transport.fail {
		return stavprotocol.LocalResponse{}, fmt.Errorf("STAV unavailable")
	}
	digest, _ := stavprotocol.CandidateDigest(*request.Candidate)
	receipt := stavprotocol.Receipt{Schema: stavprotocol.SchemaReceipt, TOPSID: request.TOPSID, RequestID: request.RequestID, CandidateDigest: digest, Disposition: "committed", ReasonCode: stavprotocol.ReasonReceiptCommitted, Commit: stavprotocol.CommitResult{State: "committed", EventID: "44444444-4444-4444-8444-444444444444", EventDigest: "sha256:" + strings.Repeat("4", 64), Sequence: 1, Timestamp: "2026-08-21T12:00:00.000000000Z"}}
	return stavprotocol.LocalResponse{Schema: stavprotocol.SchemaLocalResponse, TOPSID: request.TOPSID, RequestID: request.RequestID, Operation: request.Operation, Disposition: stavprotocol.LocalDispositionSucceeded, ReasonCode: stavprotocol.ReasonResponseSucceeded, Receipt: &receipt}, nil
}

func TestSubmitPersistsBeforeUnavailableSTAVAndReconciles(t *testing.T) {
	store, err := outbox.Open(t.TempDir() + "/outbox")
	if err != nil {
		t.Fatal(err)
	}
	transport := &retryTransport{fail: true}
	runtime, err := New("33333333-3333-4333-8333-333333333333", store, transport)
	if err != nil {
		t.Fatal(err)
	}
	candidate := producerCandidate()
	digest, _ := stavprotocol.CandidateDigest(candidate)
	if _, pending, err := runtime.Submit(context.Background(), candidate, digest); err == nil || !pending {
		t.Fatalf("unavailable STAV did not produce durable pending state: pending=%t err=%v", pending, err)
	}
	if count, _ := store.Count(); count != 1 {
		t.Fatalf("pending candidate was not durable: count=%d", count)
	}
	transport.fail = false
	committed, remaining, err := runtime.Reconcile(context.Background())
	if err != nil || committed != 1 || remaining != 0 {
		t.Fatalf("reconciliation failed: committed=%d remaining=%d err=%v", committed, remaining, err)
	}
}

func producerCandidate() stavprotocol.Candidate {
	return stavprotocol.Candidate{
		Actor:         stavprotocol.CandidateActor{Authentication: stavprotocol.Authentication{MethodID: "symphony.ssiag.local-peer", State: "identified"}, Principal: stavprotocol.SafeReference{ID: "owner.test", Kind: "symphony.identity.owner"}},
		Configuration: stavprotocol.Configuration{PreviousDigest: "sha256:" + strings.Repeat("1", 64), NewDigest: "sha256:" + strings.Repeat("2", 64), State: "digests"},
		Correlation:   stavprotocol.Correlation{RequestID: "11111111-1111-4111-8111-111111111111", CorrelationID: "22222222-2222-4222-8222-222222222222"},
		Operation:     stavprotocol.Operation{EventClass: "symphony.sav.named-version.lifecycle", OperationID: "symphony.sav.named-version.prepare", Target: stavprotocol.SafeReference{ID: "proposal", Kind: "symphony.sav.named-version-proposal"}},
		Redaction:     stavprotocol.Redaction{Classification: "administrative_metadata"}, Result: stavprotocol.Result{IntentID: "symphony.sav.named-version.prepare", Outcome: "succeeded", ReasonCode: "symphony.sav.named-version.prepare.succeeded"}, Schema: stavprotocol.SchemaCandidate,
		Topology: stavprotocol.Topology{TOPSID: "33333333-3333-4333-8333-333333333333", TROG: stavprotocol.TROG{ReasonCode: "symphony.stav.trog.not-applicable", State: "not_applicable"}},
	}
}
