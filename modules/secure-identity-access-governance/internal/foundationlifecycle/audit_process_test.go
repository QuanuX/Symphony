//go:build darwin || linux

package foundationlifecycle

import (
	"context"
	"errors"
	stav "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/config"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/foundationaudit"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/lifecycle"
	ssiagpaths "github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/paths"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/provider"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/server"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/stavproducer"
	stavclient "github.com/QuanuX/Symphony/modules/stav-append-authority/client"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

type loseFoundationOutcomeAcknowledgement struct {
	client  *stavclient.Client
	dropped atomic.Bool
}

func (a *loseFoundationOutcomeAcknowledgement) Do(ctx context.Context, request stav.LocalRequest) (stav.LocalResponse, error) {
	response, err := a.client.Do(ctx, request)
	if err == nil && !a.dropped.Load() && request.Candidate.Operation.EventClass == "symphony.ssiag.foundation.outcome" {
		a.dropped.Store(true)
		return stav.LocalResponse{}, errors.New("fixture lost acknowledgement after durable append")
	}
	return response, err
}

// Real STAV process, SSIAG server/closed producer, and native enrollment owner.
// All installation/configuration lives in the private fixture; no OS supervisor
// or credential provider is activated.
func TestFoundationLifecycleRealSTAVBridgeRestartReconcilesLostReply(t *testing.T) {
	root, err := os.MkdirTemp("/tmp", "l13-audit-")
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
		Schema: stav.SchemaAppendAuthorityConfig, TOPSID: testTOPSID, Mode: "user",
		Listen: stav.AppendAuthorityListen{Network: "unix", Address: filepath.Join(root, "run", "s")},
		Ledger: stav.AppendAuthorityLedger{Path: filepath.Join(root, "state", "ledger"), MaxBytes: 1048576, Durability: "fsync-before-receipt", Recovery: "preserve-incomplete-tail", Retention: "preserve_all", Rotation: "disabled"},
		Authentication: stav.AppendAuthorityAuthentication{
			Mechanism: "kernel-peer-credentials",
			Authority: stav.AuthorityGrant{UID: uid, GID: gid, Subject: stav.SafeReference{ID: "fixture-stav", Kind: "symphony.identity.service"}},
			Producers: []stav.ProducerGrant{{UID: uid, GID: gid, Subject: stav.SafeReference{ID: "fixture-ssiag", Kind: "symphony.identity.service"}, Producer: stav.SafeReference{ID: "fixture-ssiag", Kind: "symphony.stav.producer"}, Permissions: []stav.PeerPermission{
				{EventClass: "symphony.ssiag.foundation.intent", OperationID: "symphony.ssiag.foundation.prepare"},
				{EventClass: "symphony.ssiag.foundation.outcome", OperationID: "symphony.ssiag.foundation.observe"},
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
		command := exec.Command(binary, "serve", "--tops-id", testTOPSID, "--config", configPath)
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
			r, err := client.Do(ctx, stav.LocalRequest{Schema: stav.SchemaLocalRequest, TOPSID: testTOPSID, RequestID: id, Operation: stav.LocalOperationStatus})
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
	engine, _ := setupEngine(t)
	if _, err := lifecycle.Enroll(ssiagpaths.ScopeUser, testTOPSID, "Bootstrap fixture", nil, nil); err != nil {
		t.Fatal(err)
	}
	layout, err := ssiagpaths.ResolveInstance(ssiagpaths.ScopeUser, testTOPSID)
	if err != nil {
		t.Fatal(err)
	}
	serviceConfig, err := config.LoadTrusted(layout.ConfigFile, ssiagpaths.ScopeUser)
	if err != nil {
		t.Fatal(err)
	}
	socket := filepath.Join(root, "ssiag.sock")
	serviceConfig.Listen.Address = socket
	t.Setenv("SYMPHONY_SSIAG_SOCKET", socket)
	loss := &loseFoundationOutcomeAcknowledgement{client: client}
	producer, err := stavproducer.New(testTOPSID, loss)
	if err != nil {
		t.Fatal(err)
	}
	registry, _ := provider.New(nil)
	instance, err := server.NewWithAudit(serviceConfig, registry, producer)
	if err != nil {
		t.Fatal(err)
	}
	serviceCtx, cancelService := context.WithCancel(context.Background())
	serviceDone := make(chan error, 1)
	go func() { serviceDone <- instance.Run(serviceCtx) }()
	t.Cleanup(func() {
		cancelService()
		select {
		case err := <-serviceDone:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(5 * time.Second):
			t.Error("fixture SSIAG shutdown timeout")
		}
	})
	ready := false
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
		conn, err := net.DialTimeout("unix", socket, 50*time.Millisecond)
		if err == nil {
			conn.Close()
			ready = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !ready {
		t.Fatal("SSIAG test bridge did not listen")
	}
	command, _ := planEnrollment(t, engine, "enrolled")
	command.Plan.AuditMode = "ordinary"
	command.Plan.PlanDigest, _ = digestWithout(*command.Plan, "plan_digest")
	result, err := engine.Execute(command)
	if err != nil {
		t.Fatal(err)
	}
	if result.Disposition != "applied" || !result.Changed || result.AuditState != "audit_deferred" || result.AuditReceiptDigest != nil || !loss.dropped.Load() {
		t.Fatalf("lost real append response was misreported: %+v", result)
	}
	stored, err := foundationaudit.Load(filepath.Join(mustInstallLayout(ssiagpaths.ScopeUser).LifecycleDir, testTOPSID, "enrollment", "audit"), *command.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Intent == nil || stored.Intent.Receipt.Commit.Sequence != 1 || stored.Completion != nil || stored.Outcome == nil {
		t.Fatal("intent and pending outcome are not independently durable")
	}
	outcomeID := stored.Outcome.RequestID
	stop()
	start()
	restarted := New()
	restarted.now = engine.now
	restarted.mutate = func(Plan) error { t.Fatal("real receipt reconciliation reran enrollment"); return nil }
	command.Operation = "recover"
	command.Plan = nil
	command.ExpectedAttemptDigest = result.AttemptDigest
	recovered, err := restarted.Execute(command)
	if err != nil {
		t.Fatal(err)
	}
	if !recovered.Recovered || recovered.AuditState != "reconciled" || recovered.AuditReceiptDigest == nil || recovered.ReconciliationRequired {
		t.Fatalf("real restarted receipt not reconciled: %+v", recovered)
	}
	stored, err = foundationaudit.Load(filepath.Join(mustInstallLayout(ssiagpaths.ScopeUser).LifecycleDir, testTOPSID, "enrollment", "audit"), *command.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Completion == nil || stored.Completion.Receipt.Commit.Sequence != 2 || stored.Completion.Receipt.RequestID != outcomeID {
		t.Fatal("recovery changed or duplicated the real outcome")
	}
	id, _ := stav.GenerateUUIDv4()
	status, err := client.Do(context.Background(), stav.LocalRequest{Schema: stav.SchemaLocalRequest, TOPSID: testTOPSID, RequestID: id, Operation: stav.LocalOperationStatus})
	if err != nil || status.Status == nil || status.Status.Events != 2 {
		t.Fatalf("real ledger did not contain exactly intent/outcome: %+v %v", status, err)
	}
}
