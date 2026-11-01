//go:build darwin || linux

package credential

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	stav "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/stavproducer"
	stavclient "github.com/QuanuX/Symphony/modules/stav-append-authority/client"
)

type loseOutcomeAcknowledgement struct {
	client  *stavclient.Client
	dropped bool
}

func (a *loseOutcomeAcknowledgement) Do(ctx context.Context, request stav.LocalRequest) (stav.LocalResponse, error) {
	response, err := a.client.Do(ctx, request)
	if err == nil && !a.dropped && request.Candidate.Operation.EventClass == "symphony.ssiag.provider.operation" {
		a.dropped = true
		return stav.LocalResponse{}, errors.New("fixture lost acknowledgement after durable append")
	}
	return response, err
}

// Real append-authority process, kernel peer checks, fsync receipt and restart
// idempotency; only credential delivery is a safe synthetic callback.
func TestOutcomeRealSTAVRestartRecoversLostAcknowledgement(t *testing.T) {
	root, err := os.MkdirTemp("", "l14-audit-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(root) })
	buildCtx, cancelBuild := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancelBuild()
	binary := filepath.Join(root, "stav")
	// Resolve the declared dependency: the source workspace selects its local
	// authority, while an independent module test selects the pinned release.
	// Do not require an unrecorded sibling checkout to run this module's tests.
	build := exec.CommandContext(buildCtx, "go", "build", "-mod=readonly", "-o", binary,
		"github.com/QuanuX/Symphony/modules/stav-append-authority/cmd/symphony-stav-append-authority")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build current STAV: %v\n%s", err, output)
	}
	uid, gid := uint64(os.Geteuid()), uint64(os.Getegid())
	cfg := stav.AppendAuthorityConfig{
		Schema: stav.SchemaAppendAuthorityConfig, TOPSID: dispatchTOPS, Mode: "user",
		Listen: stav.AppendAuthorityListen{Network: "unix", Address: filepath.Join(root, "run", "s")},
		Ledger: stav.AppendAuthorityLedger{Path: filepath.Join(root, "state", "ledger"), MaxBytes: 1048576, Durability: "fsync-before-receipt", Recovery: "preserve-incomplete-tail", Retention: "preserve_all", Rotation: "disabled"},
		Authentication: stav.AppendAuthorityAuthentication{
			Mechanism: "kernel-peer-credentials",
			Authority: stav.AuthorityGrant{UID: uid, GID: gid, Subject: stav.SafeReference{ID: "fixture-stav", Kind: "symphony.identity.service"}},
			Producers: []stav.ProducerGrant{{UID: uid, GID: gid, Subject: stav.SafeReference{ID: "fixture-ssiag", Kind: "symphony.identity.service"}, Producer: stav.SafeReference{ID: "fixture-ssiag", Kind: "symphony.stav.producer"}, Permissions: []stav.PeerPermission{
				{EventClass: "symphony.ssiag.policy.decision", OperationID: "symphony.ssiag.authorize"},
				{EventClass: "symphony.ssiag.provider.operation", OperationID: "symphony.ssiag.provider.execute"},
			}}},
			Readers: []stav.ReaderGrant{{UID: uid, GID: gid, Subject: stav.SafeReference{ID: "fixture-reader", Kind: "symphony.identity.operator"}, Classifications: []string{"administrative_metadata"}}},
		},
	}
	encoded, err := stav.EncodeAppendAuthorityConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(root, "config.json")
	if err := os.WriteFile(configPath, encoded, 0600); err != nil {
		t.Fatal(err)
	}
	client, err := stavclient.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	start := func() func() {
		command := exec.Command(binary, "serve", "--tops-id", dispatchTOPS, "--config", configPath)
		if err := command.Start(); err != nil {
			t.Fatal(err)
		}
		done := make(chan error, 1)
		go func() { done <- command.Wait() }()
		stopped := false
		stop := func() {
			if stopped {
				return
			}
			stopped = true
			_ = command.Process.Signal(syscall.SIGTERM)
			select {
			case err := <-done:
				if err != nil {
					t.Errorf("STAV shutdown: %v", err)
				}
			case <-time.After(10 * time.Second):
				_ = command.Process.Kill()
				<-done
				t.Error("STAV shutdown timeout")
			}
		}
		t.Cleanup(stop)
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			id, _ := stav.GenerateUUIDv4()
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			r, err := client.Do(ctx, stav.LocalRequest{Schema: stav.SchemaLocalRequest, TOPSID: dispatchTOPS, RequestID: id, Operation: stav.LocalOperationStatus})
			cancel()
			if err == nil && r.Status != nil && r.Status.Ready {
				return stop
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatal("current STAV process did not become ready")
		return stop
	}
	stop := start()
	d, in, p, _, _ := dispatchFixture(t)
	loss := &loseOutcomeAcknowledgement{client: client}
	d.audit, err = stavproducer.New(dispatchTOPS, loss)
	if err != nil {
		t.Fatal(err)
	}
	ctx := dispatchPeer(t, true)
	attempt, _ := d.Prepare(in)
	if got := attempt.Run(ctx); got != DispatchRecoveryRequired || !loss.dropped {
		t.Fatalf("lost acknowledged outcome: %s", got)
	}
	pending := readOutcomeRecord(t, d, in)
	if pending.Audit.Receipt != nil || pending.Outcome != DispatchDelivered {
		t.Fatal("lost ack became committed locally")
	}
	stop()
	start()
	d.audit, err = stavproducer.New(dispatchTOPS, client)
	if err != nil {
		t.Fatal(err)
	}
	result, err := d.Recover(ctx, in, recoveryAuthorization(in))
	if err != nil || result.AuditState != "committed" || result.Outcome != DispatchDelivered || p.executions.Load() != 1 {
		t.Fatal(result, err)
	}
	committed := readOutcomeRecord(t, d, in)
	if committed.Audit.RequestID != pending.Audit.RequestID || committed.Audit.Receipt.Commit.Sequence != 2 {
		t.Fatal("restart did not return original provider receipt")
	}
	id, _ := stav.GenerateUUIDv4()
	status, err := client.Do(ctx, stav.LocalRequest{Schema: stav.SchemaLocalRequest, TOPSID: dispatchTOPS, RequestID: id, Operation: stav.LocalOperationStatus})
	if err != nil || status.Status == nil || status.Status.Events != 3 {
		t.Fatal("duplicate outcome append after restart", err)
	}
}
