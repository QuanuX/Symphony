package stavproducer

import (
	"context"
	"strings"
	"testing"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
)

const testTOPSID = "123e4567-e89b-42d3-a456-426614174000"

type fakeTransport struct {
	request stavprotocol.LocalRequest
	reject  bool
}

func (f *fakeTransport) Do(_ context.Context, request stavprotocol.LocalRequest) (stavprotocol.LocalResponse, error) {
	f.request = request
	if f.reject {
		return stavprotocol.LocalResponse{
			Disposition: stavprotocol.LocalDispositionRejected,
			Operation:   request.Operation,
			ReasonCode:  stavprotocol.ReasonResponseUnauthorizedPeer,
			RequestID:   request.RequestID,
			Schema:      stavprotocol.SchemaLocalResponse,
			TOPSID:      request.TOPSID,
		}, nil
	}
	digest, _ := stavprotocol.CandidateDigest(*request.Candidate)
	receipt := stavprotocol.Receipt{
		CandidateDigest: digest,
		Commit: stavprotocol.CommitResult{
			EventDigest: "sha256:4236aee922a67725aa5b90e22e88bfcf0aa510875f03777b82e326a1ffa5eef2",
			EventID:     "b90e1205-1b3b-4e47-9b91-1cd624cd87cd",
			Sequence:    1,
			State:       "committed",
			Timestamp:   "2026-07-16T12:00:00.000000001Z",
		},
		Disposition: "committed",
		ReasonCode:  stavprotocol.ReasonReceiptCommitted,
		RequestID:   request.RequestID,
		Schema:      stavprotocol.SchemaReceipt,
		TOPSID:      request.TOPSID,
	}
	return stavprotocol.LocalResponse{
		Disposition: stavprotocol.LocalDispositionSucceeded,
		Operation:   request.Operation,
		ReasonCode:  stavprotocol.ReasonResponseSucceeded,
		Receipt:     &receipt,
		RequestID:   request.RequestID,
		Schema:      stavprotocol.SchemaLocalResponse,
		TOPSID:      request.TOPSID,
	}, nil
}

func validRecord(kind Kind, outcome string) Record {
	return Record{
		Kind:          kind,
		RequestID:     "684921d8-a8b5-49da-872b-568eb6a6dc03",
		CorrelationID: "0190c7df-6df2-7f2b-9f4f-8e0c33f5f287",
		Actor:         stavprotocol.SafeReference{ID: "operator", Kind: "symphony.identity.operator"},
		Authentication: stavprotocol.Authentication{
			MethodID: "symphony.ssiag.local-peer",
			State:    "identified",
		},
		Target:         stavprotocol.SafeReference{ID: "policy-one", Kind: "symphony.ssiag.policy"},
		Outcome:        outcome,
		Configuration:  stavprotocol.Configuration{ReasonCode: "symphony.stav.configuration.not-applicable", State: "not_applicable"},
		TROG:           stavprotocol.TROG{ReasonCode: "symphony.stav.trog.not-applicable", State: "not_applicable"},
		Classification: "administrative_metadata",
	}
}

func TestSubmitBuildsClosedSafeCandidate(t *testing.T) {
	transport := &fakeTransport{}
	producer, err := New(testTOPSID, transport)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := producer.Submit(context.Background(), validRecord(PolicyDecision, "denied"))
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Disposition != "committed" {
		t.Fatalf("unexpected receipt: %#v", receipt)
	}
	candidate := transport.request.Candidate
	if candidate.Operation.EventClass != "symphony.ssiag.policy.decision" || candidate.Operation.OperationID != "symphony.ssiag.authorize" || candidate.Result.IntentID != "symphony.ssiag.policy.evaluate" || candidate.Result.ReasonCode != "symphony.ssiag.policy.denied" {
		t.Fatalf("producer vocabulary drifted: %#v", candidate)
	}
}

func TestSubmitFailsClosedOnUnknownOutcomeOrRejection(t *testing.T) {
	transport := &fakeTransport{}
	producer, err := New(testTOPSID, transport)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := producer.Submit(context.Background(), validRecord(PolicyDecision, "succeeded")); err == nil || !strings.Contains(err.Error(), "unsupported outcome") {
		t.Fatalf("unexpected outcome error: %v", err)
	}
	transport.reject = true
	if _, err := producer.Submit(context.Background(), validRecord(PolicyDecision, "denied")); err == nil || !strings.Contains(err.Error(), "append rejected") {
		t.Fatalf("unexpected rejection error: %v", err)
	}
}
