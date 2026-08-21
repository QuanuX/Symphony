package server

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	accordareclient "github.com/QuanuX/Symphony/modules/accordare-stav-producer/client"
	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/config"
	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/outbox"
	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/producer"
)

const socketTestTOPS = "11111111-1111-4111-8111-111111111111"

type unavailableSTAV struct{}

func (unavailableSTAV) Do(context.Context, stavprotocol.LocalRequest) (stavprotocol.LocalResponse, error) {
	return stavprotocol.LocalResponse{}, fmt.Errorf("STAV unavailable in socket test")
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
	runtimeProducer, err := producer.New(socketTestTOPS, store, unavailableSTAV{})
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
