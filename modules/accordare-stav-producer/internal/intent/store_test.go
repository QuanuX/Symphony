package intent

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	accordareprotocol "github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/protocol"
)

func TestStoreMakesExactRetryIdempotentAndConflictVisible(t *testing.T) {
	store, err := Open(t.TempDir() + "/intents")
	if err != nil {
		t.Fatal(err)
	}
	value := testIntent()
	if err := store.Put(value); err != nil {
		t.Fatal(err)
	}
	retry := value
	retry.PreparedAt = retry.PreparedAt.Add(time.Minute)
	if err := store.Put(retry); err != nil {
		t.Fatalf("exact retry was not idempotent: %v", err)
	}
	conflict := value
	var command map[string]any
	if err := json.Unmarshal(conflict.Submission.Command, &command); err != nil {
		t.Fatal(err)
	}
	command["operation_id"] = "different-operation"
	conflict.Submission.Command, _ = json.Marshal(command)
	if err := store.Put(conflict); err == nil {
		t.Fatal("conflicting evidence reused an intent identity")
	}
	reopened, err := Open(t.TempDir() + "/unused")
	if err != nil || reopened == nil {
		t.Fatal(err)
	}
	if count, err := store.Count(); err != nil || count != 1 {
		t.Fatalf("durable intent count=%d err=%v", count, err)
	}
	if err := store.Remove(value.IntentID); err != nil {
		t.Fatal(err)
	}
}

func testIntent() Intent {
	command, _ := json.Marshal(map[string]any{
		"authorization_decision": nil,
		"operation_id":           "operation-1",
	})
	return Intent{
		IntentID:   "11111111-1111-4111-8111-111111111111",
		Peer:       stavprotocol.SafeReference{ID: "owner.test", Kind: "owner"},
		PreparedAt: time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC), Schema: schema,
		Submission: accordareprotocol.Submission{Command: command, Coordinator: accordareprotocol.InstallationEvidence{ComponentID: "knowledge-session-coordinator", ExecutableDigest: "sha256:" + strings.Repeat("1", 64), ReceiptDigest: "sha256:" + strings.Repeat("2", 64), Version: "0.1.0"}, Operation: "named_version_prepare", Schema: accordareprotocol.SchemaSubmission, TOPSID: "22222222-2222-4222-8222-222222222222"},
	}
}
