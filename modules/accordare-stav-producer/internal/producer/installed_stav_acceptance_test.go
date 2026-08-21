package producer

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/intent"
	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/outbox"
	stavclient "github.com/QuanuX/Symphony/modules/stav-append-authority/client"
)

func TestInstalledSTAVAcceptance(t *testing.T) {
	binary := os.Getenv("SYMPHONY_STAV_ACCEPTANCE_BINARY")
	if binary == "" {
		t.Skip("set SYMPHONY_STAV_ACCEPTANCE_BINARY to an exact built STAV executable")
	}
	root, err := os.MkdirTemp("/tmp", "accordare-stav-acceptance-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	uid, gid := uint64(os.Geteuid()), uint64(os.Getegid())
	candidate := producerCandidate()
	cfg := stavprotocol.AppendAuthorityConfig{
		Authentication: stavprotocol.AppendAuthorityAuthentication{
			Authority: stavprotocol.AuthorityGrant{UID: uid, GID: gid, Subject: stavprotocol.SafeReference{ID: "stav-acceptance", Kind: "symphony.identity.service"}}, Mechanism: "kernel-peer-credentials",
			Producers: []stavprotocol.ProducerGrant{{UID: uid, GID: gid, Subject: stavprotocol.SafeReference{ID: "accordare-acceptance", Kind: "symphony.identity.service"}, Producer: stavprotocol.SafeReference{ID: "symphony.accordare.stav-producer", Kind: "symphony.stav.producer"}, Permissions: []stavprotocol.PeerPermission{{EventClass: candidate.Operation.EventClass, OperationID: candidate.Operation.OperationID}}}},
			Readers:   []stavprotocol.ReaderGrant{},
		},
		Ledger: stavprotocol.AppendAuthorityLedger{Durability: "fsync-before-receipt", MaxBytes: 1 << 20, Path: filepath.Join(root, "state", "ledger-v1.stavlog"), Recovery: "preserve-incomplete-tail", Retention: "preserve_all", Rotation: "disabled"},
		Listen: stavprotocol.AppendAuthorityListen{Address: filepath.Join(root, "runtime", "append.sock"), Network: "unix"}, Mode: "user", Schema: stavprotocol.SchemaAppendAuthorityConfig, TOPSID: candidate.Topology.TOPSID,
	}
	data, err := stavprotocol.EncodeAppendAuthorityConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(root, "append-authority.json")
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(binary, "serve", "--scope", "user", "--tops-id", cfg.TOPSID, "--config", configPath)
	var processOutput bytes.Buffer
	command.Stdout, command.Stderr = &processOutput, &processOutput
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = command.Process.Signal(os.Interrupt); _, _ = command.Process.Wait() })
	deadline := time.Now().Add(3 * time.Second)
	for {
		if info, err := os.Lstat(cfg.Listen.Address); err == nil && info.Mode()&os.ModeSocket != 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("installed STAV did not become ready: %s", processOutput.String())
		}
		time.Sleep(10 * time.Millisecond)
	}
	transport, err := stavclient.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	intentStore, err := intent.Open(filepath.Join(root, "intents"))
	if err != nil {
		t.Fatal(err)
	}
	outboxStore, err := outbox.Open(filepath.Join(root, "outbox"))
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := New(cfg.TOPSID, intentStore, outboxStore, transport)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := stavprotocol.CandidateDigest(candidate)
	if err != nil {
		t.Fatal(err)
	}
	receipt, pending, err := runtime.Submit(context.Background(), candidate, digest)
	if err != nil || pending || receipt.Disposition != "committed" || receipt.CandidateDigest != digest {
		t.Fatalf("real STAV acceptance failed: receipt=%+v pending=%t err=%v", receipt, pending, err)
	}
}
