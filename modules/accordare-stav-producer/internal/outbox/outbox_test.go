package outbox

import (
	"strings"
	"testing"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
)

func TestOutboxDurablyReopensAndRejectsConflictingIdentity(t *testing.T) {
	directory := t.TempDir() + "/outbox"
	store, err := Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	candidate := validCandidate()
	digest, err := stavprotocol.CandidateDigest(candidate)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put(candidate, digest); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	items, err := reopened.List()
	if err != nil || len(items) != 1 || items[0].CandidateDigest != digest {
		t.Fatalf("durable outbox entry was not recovered: items=%+v err=%v", items, err)
	}
	conflict := candidate
	conflict.Result.ReasonCode = "symphony.sav.named-version.prepare.failed"
	conflict.Result.Outcome = "failed"
	conflictDigest, _ := stavprotocol.CandidateDigest(conflict)
	if err := reopened.Put(conflict, conflictDigest); err == nil {
		t.Fatal("same request identity accepted different evidence")
	}
	if err := reopened.Remove(candidate.Correlation.RequestID); err != nil {
		t.Fatal(err)
	}
	if count, err := reopened.Count(); err != nil || count != 0 {
		t.Fatalf("outbox cleanup failed: count=%d err=%v", count, err)
	}
}

func validCandidate() stavprotocol.Candidate {
	digest := "sha256:" + strings.Repeat("1", 64)
	return stavprotocol.Candidate{
		Actor:         stavprotocol.CandidateActor{Authentication: stavprotocol.Authentication{MethodID: "symphony.ssiag.local-peer", State: "identified"}, Principal: stavprotocol.SafeReference{ID: "owner.test", Kind: "symphony.identity.owner"}},
		Configuration: stavprotocol.Configuration{PreviousDigest: digest, NewDigest: "sha256:" + strings.Repeat("2", 64), State: "digests"},
		Correlation:   stavprotocol.Correlation{RequestID: "11111111-1111-4111-8111-111111111111", CorrelationID: "22222222-2222-4222-8222-222222222222"},
		Operation:     stavprotocol.Operation{EventClass: "symphony.sav.named-version.lifecycle", OperationID: "symphony.sav.named-version.prepare", Target: stavprotocol.SafeReference{ID: digest, Kind: "symphony.sav.named-version-proposal"}},
		Redaction:     stavprotocol.Redaction{Classification: "administrative_metadata"}, Result: stavprotocol.Result{IntentID: "symphony.sav.named-version.prepare", Outcome: "succeeded", ReasonCode: "symphony.sav.named-version.prepare.succeeded"},
		Schema: stavprotocol.SchemaCandidate, Topology: stavprotocol.Topology{TOPSID: "33333333-3333-4333-8333-333333333333", TROG: stavprotocol.TROG{ReasonCode: "symphony.stav.trog.not-applicable", State: "not_applicable"}},
	}
}
