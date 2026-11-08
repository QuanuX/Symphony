//go:build darwin || linux

package store

import (
	"context"
	"errors"
	stav "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	c "github.com/QuanuX/Symphony/tools/sov-transport-profile-admin/contract"
	a "github.com/QuanuX/Symphony/tools/sov-transport-profile-admin/internal/transportaudit"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fixture(t *testing.T) (*Store, c.Plan, map[string]a.Request) {
	t.Helper()
	root := t.TempDir()
	for _, d := range []string{"profiles", "history", "attempts"} {
		if e := os.Mkdir(filepath.Join(root, d), 0755); e != nil {
			t.Fatal(e)
		}
	}
	if e := os.WriteFile(filepath.Join(root, "LOCK"), nil, 0644); e != nil {
		t.Fatal(e)
	}
	f, e := os.Open(root)
	if e != nil {
		t.Fatal(e)
	}
	d := &directory{file: f, owner: uint32(os.Geteuid())}
	s := &Store{dir: d, scope: c.Scope{HostID: "11111111-1111-4111-8111-111111111111", TOPSID: "22222222-2222-4222-8222-222222222222"}, administrator: "sha256:" + strings.Repeat("b", 64), now: func() time.Time { return time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC) }}
	s.profiles, e = d.child("profiles")
	if e != nil {
		t.Fatal(e)
	}
	s.history, e = d.child("history")
	if e != nil {
		t.Fatal(e)
	}
	s.attempts, e = d.child("attempts")
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(s.Close)
	audits := map[string]a.Request{}
	s.audit = func(_ context.Context, r a.Request) (a.Response, error) {
		if previous, ok := audits[r.RequestID]; ok && !c.Same(previous, r) {
			return a.Response{}, errors.New("candidate changed")
		}
		audits[r.RequestID] = r
		candidate, e := a.Candidate(r, 0)
		if e != nil {
			return a.Response{}, e
		}
		digest, e := stav.CandidateDigest(candidate)
		if e != nil {
			return a.Response{}, e
		}
		receipt := stav.Receipt{Schema: stav.SchemaReceipt, TOPSID: r.TOPSID, RequestID: r.RequestID, CandidateDigest: digest, Disposition: "committed", ReasonCode: stav.ReasonReceiptCommitted, Commit: stav.CommitResult{EventID: "55555555-5555-4555-8555-555555555555", EventDigest: "sha256:" + strings.Repeat("c", 64), Sequence: 1, State: "committed", Timestamp: "2026-11-07T12:00:00.000000000Z"}}
		return a.Response{Protocol: a.Protocol, RequestDigest: a.Hash(r), Receipt: receipt}, nil
	}
	digest := "sha256:" + strings.Repeat("a", 64)
	pkg := c.PackageRef{PackageID: "66666666-6666-4666-8666-666666666666", Version: "0.1.0-dev", ManifestDigest: digest, ReceiptDigest: digest, ExecutableDigest: digest}
	config := c.Configuration{Name: "example", TransportKind: "cloudflare-access", Platform: "darwin-amd64", ExecutionProfile: "cloudflare-access-user-session.v1", Adapter: pkg, SSH: pkg, Helper: pkg, Applications: []string{"example.invalid"}, Endpoints: []c.EndpointConstraint{{Host: "ssh.example.invalid", Port: "22", User: "example", HostKeyDigest: digest}}, AuthenticationMode: "user_session", SessionRef: "session:example", Limits: c.Limits{ControlBytes: "8388608", CapturedBytes: "1048576", CleanupSeconds: "5", CallerTimeout: true}, Compatibility: []string{"symphony.sov.ssh-endpoint.v2", "symphony.sov.ssh-invoke-input.v2", "symphony.sov.ssh-carrier-result.v2"}}
	principals := []c.Principal{{UID: "501", GID: "20"}}
	plan, e := c.MakePlan(s.scope, "33333333-3333-4333-8333-333333333333", "44444444-4444-4444-8444-444444444444", s.administrator, nil, c.Change{Action: "create", Configuration: &config, Permitted: &principals}, s.now())
	if e != nil {
		t.Fatal(e)
	}
	return s, plan, audits
}
func TestPublicationFaultsRequireExplicitRecovery(t *testing.T) {
	for _, stage := range []string{"intent_prepared", "audit_intent_committed", "state_published", "audit_outcome_committed", "outcome_archived"} {
		t.Run(stage, func(t *testing.T) {
			s, plan, audits := fixture(t)
			s.fault = func(at string) error {
				if at == stage {
					return errors.New("simulated interruption")
				}
				return nil
			}
			if s.apply(context.Background(), plan) == nil {
				t.Fatal("fault not observed")
			}
			p, e := s.readPending("pending.json", s.dir)
			if e != nil || p == nil {
				t.Fatal("durable pending intent missing", e)
			}
			if e = s.apply(context.Background(), plan); !errors.Is(e, errPending) {
				t.Fatal("uncertain attempt silently replayed", e)
			}
			r := c.Recovery{OperationID: plan.OperationID, PlanDigest: plan.Digest, ExpectedPendingDigest: p.Digest, Action: "complete"}
			stale := r
			stale.ExpectedPendingDigest = "sha256:" + strings.Repeat("f", 64)
			if s.recover(context.Background(), stale) == nil {
				t.Fatal("stale recovery accepted")
			}
			s.fault = nil
			if stage == "outcome_archived" {
				abort := r
				abort.Action = "abort"
				if s.recover(context.Background(), abort) == nil {
					t.Fatal("abort acknowledged an already committed publication")
				}
			}
			if e = s.recover(context.Background(), r); e != nil {
				t.Fatal(e)
			}
			current, e := s.current(plan.ProfileID)
			if e != nil || !c.Same(current, &plan.Desired) {
				t.Fatal("coherent snapshot not published", e)
			}
			pending, e := s.readPending("pending.json", s.dir)
			if e != nil || pending != nil {
				t.Fatal("recovery not settled")
			}
			if len(audits) != 2 {
				t.Fatal("audit request identity changed across recovery", len(audits))
			}
			if s.apply(context.Background(), plan) == nil {
				t.Fatal("settled operation silently replayed")
			}
		})
	}
}
func TestAuditMissingOrForgedNeverPublishes(t *testing.T) {
	for _, forge := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing", true: "forged"}[forge], func(t *testing.T) {
			s, plan, _ := fixture(t)
			original := s.audit
			s.audit = nil
			if forge {
				s.audit = func(ctx context.Context, r a.Request) (a.Response, error) {
					v, e := original(ctx, r)
					v.Receipt.CandidateDigest = "sha256:" + strings.Repeat("f", 64)
					return v, e
				}
			}
			if s.apply(context.Background(), plan) == nil {
				t.Fatal("uncommitted audit accepted")
			}
			current, e := s.current(plan.ProfileID)
			if e != nil || current != nil {
				t.Fatal("state published without audit")
			}
			p, e := s.readPending("pending.json", s.dir)
			if e != nil || p == nil || p.Stage != "prepared" {
				t.Fatal("required recovery evidence lost")
			}
			s.audit = original
			if e = s.recover(context.Background(), c.Recovery{OperationID: plan.OperationID, PlanDigest: plan.Digest, ExpectedPendingDigest: p.Digest, Action: "abort"}); e != nil {
				t.Fatal(e)
			}
			current, e = s.current(plan.ProfileID)
			if e != nil || current != nil {
				t.Fatal("abort changed profile")
			}
		})
	}
}
func TestProtectedPathsLockAndKernelAuthority(t *testing.T) {
	s, plan, _ := fixture(t)
	root := s.dir.file.Name()
	if _, e := openProtected(root); e == nil {
		t.Fatal("writable temporary ancestor accepted as protected production root")
	}
	if e := os.WriteFile(filepath.Join(root, "owned"), []byte("safe"), 0644); e != nil {
		t.Fatal(e)
	}
	if e := os.Link(filepath.Join(root, "owned"), filepath.Join(root, "hardlink")); e != nil {
		t.Fatal(e)
	}
	if _, e := s.dir.read("owned", 100); e == nil {
		t.Fatal("hard-linked root file accepted")
	}
	if e := os.Symlink("owned", filepath.Join(root, "link")); e != nil {
		t.Fatal(e)
	}
	if _, e := s.dir.read("link", 100); e == nil {
		t.Fatal("symlink accepted")
	}
	unlock, e := s.dir.lock(true)
	if e != nil {
		t.Fatal(e)
	}
	if close, e := s.dir.lock(true); e == nil {
		close()
		t.Fatal("concurrent exclusive lock accepted")
	}
	unlock()
	if os.Geteuid() != 0 {
		t.Setenv("SUDO_UID", "0")
		t.Setenv("SUDO_USER", "root")
		r := s.Handle(context.Background(), c.Request{Protocol: c.RequestProtocol, RequestID: "77777777-7777-4777-8777-777777777777", Operation: "apply", Scope: s.scope, Plan: &plan})
		if r.Completed || r.Code != "transport_profile.root_required" {
			t.Fatal("environment substituted for kernel root", r)
		}
	}
	if e := os.Chmod(filepath.Join(root, "profiles"), 0777); e != nil {
		t.Fatal(e)
	}
	if d, e := s.dir.child("profiles"); e == nil {
		d.close()
		t.Fatal("writable directory accepted")
	}
}
