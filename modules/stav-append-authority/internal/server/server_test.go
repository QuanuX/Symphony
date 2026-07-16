package server_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	stavclient "github.com/QuanuX/Symphony/modules/stav-append-authority/client"
	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/server"
)

const testTOPSID = "123e4567-e89b-42d3-a456-426614174000"

func testConfig(t *testing.T, grantProducer bool) stavprotocol.AppendAuthorityConfig {
	t.Helper()
	root, err := os.MkdirTemp("/tmp", "stav-server-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	uid, gid := uint64(os.Geteuid()), uint64(os.Getegid())
	producers := []stavprotocol.ProducerGrant{}
	if grantProducer {
		producers = append(producers, stavprotocol.ProducerGrant{
			GID: gid,
			Permissions: []stavprotocol.PeerPermission{{
				EventClass:  "symphony.ssiag.authentication.decision",
				OperationID: "symphony.ssiag.authenticate",
			}},
			Producer: stavprotocol.SafeReference{ID: "ssiag", Kind: "symphony.stav.producer"},
			Subject:  stavprotocol.SafeReference{ID: "ssiag-service", Kind: "symphony.identity.service"},
			UID:      uid,
		})
	}
	return stavprotocol.AppendAuthorityConfig{
		Authentication: stavprotocol.AppendAuthorityAuthentication{
			Authority: stavprotocol.AuthorityGrant{GID: gid, Subject: stavprotocol.SafeReference{ID: "stav-authority", Kind: "symphony.identity.service"}, UID: uid},
			Mechanism: "kernel-peer-credentials",
			Producers: producers,
			Readers: []stavprotocol.ReaderGrant{{
				Classifications: []string{"administrative_metadata"},
				GID:             gid,
				Subject:         stavprotocol.SafeReference{ID: "operator", Kind: "symphony.identity.operator"},
				UID:             uid,
			}},
		},
		Ledger: stavprotocol.AppendAuthorityLedger{
			Durability: "fsync-before-receipt",
			MaxBytes:   1_048_576,
			Path:       filepath.Join(root, "state", "ledger-v1.stavlog"),
			Recovery:   "preserve-incomplete-tail",
			Retention:  "preserve_all",
			Rotation:   "disabled",
		},
		Listen: stavprotocol.AppendAuthorityListen{Address: filepath.Join(root, "run", "append.sock"), Network: "unix"},
		Mode:   "user",
		Schema: stavprotocol.SchemaAppendAuthorityConfig,
		TOPSID: testTOPSID,
	}
}

func testCandidate() stavprotocol.Candidate {
	return stavprotocol.Candidate{
		Actor: stavprotocol.CandidateActor{
			Authentication: stavprotocol.Authentication{MethodID: "symphony.ssiag.fixture", State: "identified"},
			Principal:      stavprotocol.SafeReference{ID: "operator", Kind: "symphony.identity.operator"},
		},
		Configuration: stavprotocol.Configuration{ReasonCode: "symphony.stav.configuration.not-applicable", State: "not_applicable"},
		Correlation: stavprotocol.Correlation{
			CorrelationID: "0190c7df-6df2-7f2b-9f4f-8e0c33f5f287",
			RequestID:     "b90e1205-1b3b-4e47-9b91-1cd624cd87cd",
		},
		Operation: stavprotocol.Operation{
			EventClass:  "symphony.ssiag.authentication.decision",
			OperationID: "symphony.ssiag.authenticate",
			Target:      stavprotocol.SafeReference{ID: "provider", Kind: "symphony.ssiag.provider"},
		},
		Redaction: stavprotocol.Redaction{Classification: "administrative_metadata"},
		Result: stavprotocol.Result{
			IntentID:   "symphony.ssiag.authentication.requested",
			Outcome:    "allowed",
			ReasonCode: "symphony.ssiag.authentication.allowed",
		},
		Schema:   stavprotocol.SchemaCandidate,
		Topology: stavprotocol.Topology{TOPSID: testTOPSID, TROG: stavprotocol.TROG{ReasonCode: "symphony.stav.trog.not-applicable", State: "not_applicable"}},
	}
}

func runServer(t *testing.T, cfg stavprotocol.AppendAuthorityConfig) (*stavclient.Client, context.CancelFunc) {
	t.Helper()
	service, err := server.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = service.Close() })
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- service.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("server shutdown: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Error("server did not stop")
		}
	})
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); {
		if info, err := os.Lstat(cfg.Listen.Address); err == nil && info.Mode()&os.ModeSocket != 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	client, err := stavclient.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return client, cancel
}

func TestMutuallyAuthenticatedAppendAndRead(t *testing.T) {
	cfg := testConfig(t, true)
	client, _ := runServer(t, cfg)
	candidate := testCandidate()
	appendRequest := stavprotocol.LocalRequest{
		Candidate: &candidate,
		Operation: stavprotocol.LocalOperationAppend,
		RequestID: candidate.Correlation.RequestID,
		Schema:    stavprotocol.SchemaLocalRequest,
		TOPSID:    testTOPSID,
	}
	response, err := client.Do(context.Background(), appendRequest)
	if err != nil {
		t.Fatal(err)
	}
	if response.Receipt == nil || response.Receipt.Disposition != "committed" || response.Receipt.Commit.Sequence != 1 {
		t.Fatalf("unexpected append response: %#v", response)
	}
	statusID := "684921d8-a8b5-49da-872b-568eb6a6dc03"
	statusResponse, err := client.Do(context.Background(), stavprotocol.LocalRequest{
		Operation: stavprotocol.LocalOperationStatus,
		RequestID: statusID,
		Schema:    stavprotocol.SchemaLocalRequest,
		TOPSID:    testTOPSID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if statusResponse.Status == nil || statusResponse.Status.Events != 1 || !statusResponse.Status.Ready {
		t.Fatalf("unexpected status response: %#v", statusResponse)
	}
}

func TestUngrantableProducerIsRejected(t *testing.T) {
	cfg := testConfig(t, false)
	client, _ := runServer(t, cfg)
	candidate := testCandidate()
	response, err := client.Do(context.Background(), stavprotocol.LocalRequest{
		Candidate: &candidate,
		Operation: stavprotocol.LocalOperationAppend,
		RequestID: candidate.Correlation.RequestID,
		Schema:    stavprotocol.SchemaLocalRequest,
		TOPSID:    testTOPSID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Disposition != stavprotocol.LocalDispositionRejected || response.ReasonCode != stavprotocol.ReasonResponseUnauthorizedPeer {
		t.Fatalf("unexpected rejection: %#v", response)
	}
}

func TestClientRejectsWrongAuthorityIdentity(t *testing.T) {
	cfg := testConfig(t, true)
	_, _ = runServer(t, cfg)
	wrong := cfg
	wrong.Authentication.Authority.UID++
	client, err := stavclient.New(wrong)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Do(context.Background(), stavprotocol.LocalRequest{
		Operation: stavprotocol.LocalOperationStatus,
		RequestID: "684921d8-a8b5-49da-872b-568eb6a6dc03",
		Schema:    stavprotocol.SchemaLocalRequest,
		TOPSID:    testTOPSID,
	})
	if err == nil {
		t.Fatal("client accepted the wrong authority identity")
	}
}
