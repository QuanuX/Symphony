package server

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	accordareclient "github.com/QuanuX/Symphony/modules/accordare-stav-producer/client"
	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/config"
	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/intent"
	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/outbox"
	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/producer"
)

const socketTestTOPS = "11111111-1111-4111-8111-111111111111"

type unavailableSTAV struct{}

func (unavailableSTAV) Do(context.Context, stavprotocol.LocalRequest) (stavprotocol.LocalResponse, error) {
	return stavprotocol.LocalResponse{}, fmt.Errorf("STAV unavailable in socket test")
}

type drainingSTAV struct{ entered, release chan struct{} }

func (transport drainingSTAV) Do(ctx context.Context, _ stavprotocol.LocalRequest) (stavprotocol.LocalResponse, error) {
	close(transport.entered)
	<-transport.release
	return stavprotocol.LocalResponse{}, ctx.Err()
}

func TestShutdownWaitsForStartupReconciliation(t *testing.T) {
	root, err := os.MkdirTemp("/tmp", "accordare-drain-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)
	cfg := config.Config{Identity: config.Identity{UID: uint64(os.Geteuid()), GID: uint64(os.Getegid()), Subject: stavprotocol.SafeReference{ID: "accordare-test-service", Kind: "symphony.identity.service"}}, Listen: config.Listen{Address: filepath.Join(root, "runtime", "submit.sock"), Network: "unix"}, Mode: "user", Schema: config.Schema, STAVConfig: filepath.Join(root, "stav.json"), TOPSID: socketTestTOPS, Submitters: []config.Identity{{UID: uint64(os.Geteuid()), GID: uint64(os.Getegid()), Subject: stavprotocol.SafeReference{ID: "qxctl-test", Kind: "owner"}}}, VocabularyDigest: config.VocabularyDigest}
	store, err := outbox.Open(filepath.Join(root, "outbox"))
	if err != nil {
		t.Fatal(err)
	}
	intents, err := intent.Open(filepath.Join(root, "intents"))
	if err != nil {
		t.Fatal(err)
	}
	digest := "sha256:" + strings.Repeat("1", 64)
	candidate := stavprotocol.Candidate{Actor: stavprotocol.CandidateActor{Authentication: stavprotocol.Authentication{MethodID: "symphony.ssiag.local-peer", State: "identified"}, Principal: stavprotocol.SafeReference{ID: "owner.test", Kind: "symphony.identity.owner"}}, Configuration: stavprotocol.Configuration{PreviousDigest: digest, NewDigest: "sha256:" + strings.Repeat("2", 64), State: "digests"}, Correlation: stavprotocol.Correlation{RequestID: "11111111-1111-4111-8111-111111111111", CorrelationID: "22222222-2222-4222-8222-222222222222"}, Operation: stavprotocol.Operation{EventClass: "symphony.sav.named-version.lifecycle", OperationID: "symphony.sav.named-version.prepare", Target: stavprotocol.SafeReference{ID: digest, Kind: "symphony.sav.named-version-proposal"}}, Redaction: stavprotocol.Redaction{Classification: "administrative_metadata"}, Result: stavprotocol.Result{IntentID: "symphony.sav.named-version.prepare", Outcome: "succeeded", ReasonCode: "symphony.sav.named-version.prepare.succeeded"}, Schema: stavprotocol.SchemaCandidate, Topology: stavprotocol.Topology{TOPSID: socketTestTOPS, TROG: stavprotocol.TROG{ReasonCode: "symphony.stav.trog.not-applicable", State: "not_applicable"}}}
	candidateDigest, err := stavprotocol.CandidateDigest(candidate)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put(candidate, candidateDigest); err != nil {
		t.Fatal(err)
	}
	transport := drainingSTAV{entered: make(chan struct{}), release: make(chan struct{})}
	runtimeProducer, err := producer.New(socketTestTOPS, intents, store, transport)
	if err != nil {
		t.Fatal(err)
	}
	service, err := New(cfg, runtimeProducer)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- service.Run(ctx) }()
	select {
	case <-transport.entered:
	case <-time.After(2 * time.Second):
		close(transport.release)
		t.Fatal("startup reconciliation did not reach transport")
	}
	cancel()
	select {
	case err := <-done:
		close(transport.release)
		t.Fatalf("server released ownership before reconciliation drained: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	close(transport.release)
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not finish after reconciliation drained")
	}
	if count, err := store.Count(); err != nil || count != 1 {
		t.Fatalf("pending evidence changed during shutdown: %d %v", count, err)
	}
}

func TestAuthenticatedSocketStatusAndReconcile(t *testing.T) {
	root, err := os.MkdirTemp("/tmp", "accordare-socket-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	cfg := config.Config{
		Identity: config.Identity{UID: uint64(os.Geteuid()), GID: uint64(os.Getegid()), Subject: stavprotocol.SafeReference{ID: "accordare-test-service", Kind: "symphony.identity.service"}},
		Listen:   config.Listen{Address: filepath.Join(root, "runtime", "submit.sock"), Network: "unix"},
		Mode:     "user", Schema: config.Schema, STAVConfig: filepath.Join(root, "stav.json"), TOPSID: socketTestTOPS,
		Submitters:       []config.Identity{{UID: uint64(os.Geteuid()), GID: uint64(os.Getegid()), Subject: stavprotocol.SafeReference{ID: "qxctl-test", Kind: "owner"}}},
		VocabularyDigest: config.VocabularyDigest,
	}
	store, err := outbox.Open(filepath.Join(root, "outbox"))
	if err != nil {
		t.Fatal(err)
	}
	intentStore, err := intent.Open(filepath.Join(root, "intents"))
	if err != nil {
		t.Fatal(err)
	}
	runtimeProducer, err := producer.New(socketTestTOPS, intentStore, store, unavailableSTAV{})
	if err != nil {
		t.Fatal(err)
	}
	service, err := New(cfg, runtimeProducer)
	if err != nil {
		t.Fatal(err)
	}
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
			t.Error("server did not shut down")
		}
	})

	deadline := time.Now().Add(2 * time.Second)
	for {
		if info, statErr := os.Lstat(cfg.Listen.Address); statErr == nil && info.Mode()&os.ModeSocket != 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("producer socket did not become ready")
		}
		time.Sleep(5 * time.Millisecond)
	}

	producerClient, err := accordareclient.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	callCtx, callCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer callCancel()
	status, err := producerClient.Status(callCtx, "22222222-2222-4222-8222-222222222222", socketTestTOPS)
	if err != nil {
		t.Fatal(err)
	}
	if status.Disposition != "succeeded" || status.Pending != 0 {
		t.Fatalf("unexpected status: %+v", status)
	}
	reconciled, err := producerClient.Reconcile(callCtx, "33333333-3333-4333-8333-333333333333", socketTestTOPS)
	if err != nil {
		t.Fatal(err)
	}
	if reconciled.Disposition != "succeeded" || reconciled.Pending != 0 {
		t.Fatalf("unexpected reconcile result: %+v", reconciled)
	}
}
