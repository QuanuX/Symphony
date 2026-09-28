//go:build darwin || linux

package credential

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	stav "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/identity"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/model"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/peerauth"
	"golang.org/x/sys/unix"
)

func recoveryAuthorization(in DispatchIntent) model.AuthorizationRequest {
	a := in.Authorization
	a.RequestID, _ = stav.GenerateUUIDv4()
	a.RequestedAt = time.Now().UTC()
	a.RequestedExpiresAt = a.RequestedAt.Add(20 * time.Second)
	return a
}
func unitJournal(t *testing.T) (*Journal, journalIdentity, UseBinding) {
	t.Helper()
	d, in, _, _, _ := dispatchFixture(t)
	peer := peerauth.Peer{Mapped: true, Subject: identity.Subject{ID: "fixture-consumer", Kind: "symphony.identity.service", Authority: peerauth.Mechanism}, Credentials: peerauth.Credentials{UID: uint32(os.Geteuid()), GID: uint32(os.Getegid())}}
	key := journalKey(dispatchTOPS, peer, in)
	binding, now := useFixture()
	binding.RequestID = key.RequestID
	binding.SubjectKind = key.Subject.Kind
	binding.SubjectAuthority = key.Subject.Authority
	binding.IssuedAt = now
	binding.Deadline = now.Add(10 * time.Second)
	return d.journal, key, binding
}
func journalPath(j *Journal, id string) string {
	return filepath.Join(j.root, "credential-requests", j.topsID, id+".json")
}
func TestJournalDuplicateAcrossInstancesAndConflicts(t *testing.T) {
	d, in, p, _, _ := dispatchFixture(t)
	ctx := dispatchPeer(t, true)
	a, err := d.Prepare(in)
	if err != nil {
		t.Fatal(err)
	}
	if a.Run(ctx) != DispatchDelivered {
		t.Fatal("first dispatch failed")
	}
	journal, err := NewJournal(d.journal.root, dispatchTOPS)
	if err != nil {
		t.Fatal(err)
	}
	other, err := NewDispatcher(d.policy, d.audit, p, journal)
	if err != nil {
		t.Fatal(err)
	}
	a, _ = other.Prepare(in)
	if a.Run(ctx) != DispatchRecorded || p.executions.Load() != 1 {
		t.Fatal("new dispatcher replayed request")
	}
	changed := in
	changed.Reference.Version = "generation-2"
	a, _ = other.Prepare(changed)
	if a.Run(ctx) != DispatchConflict || p.executions.Load() != 1 {
		t.Fatal("changed request was admitted")
	}
	recovered, err := other.Recover(ctx, in, recoveryAuthorization(in))
	if err != nil || recovered.Outcome != DispatchDelivered {
		t.Fatal("durable outcome not retained", err)
	}
	if _, err = other.Recover(ctx, changed, recoveryAuthorization(changed)); err == nil {
		t.Fatal("changed intent recovered original")
	}
	if _, err = other.Recover(context.Background(), in, recoveryAuthorization(in)); err == nil {
		t.Fatal("unauthenticated recovery")
	}
	if _, err = other.Recover(ctx, in, in.Authorization); err == nil {
		t.Fatal("recovery reused audit request identity")
	}
}
func TestJournalConcurrentInstancesExecuteOnce(t *testing.T) {
	d, in, p, _, _ := dispatchFixture(t)
	ctx := dispatchPeer(t, true)
	var delivered atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 24; i++ {
		wg.Go(func() {
			j, _ := NewJournal(d.journal.root, dispatchTOPS)
			other, _ := NewDispatcher(d.policy, d.audit, p, j)
			a, err := other.Prepare(in)
			if err != nil {
				t.Error(err)
				return
			}
			switch a.Run(ctx) {
			case DispatchDelivered:
				delivered.Add(1)
			case DispatchUnavailable, DispatchRecorded:
			default:
				t.Error("unexpected concurrent journal result")
			}
		})
	}
	wg.Wait()
	if delivered.Load() != 1 || p.executions.Load() != 1 {
		t.Fatal("duplicate cross-instance delivery")
	}
}
func TestJournalRecoveryPreservesIntentAndUncertainty(t *testing.T) {
	for _, armed := range []bool{false, true} {
		t.Run(map[bool]string{false: "intent", true: "armed"}[armed], func(t *testing.T) {
			j, key, binding := unitJournal(t)
			a, status := j.begin(context.Background(), key)
			if a == nil {
				t.Fatal(status)
			}
			if armed && a.arm(binding) != nil {
				t.Fatal("arm failed")
			}
			if _, err := j.recover(context.Background(), key); err == nil {
				t.Fatal("recovered an active request")
			}
			a.file.close()
			if again, status := j.begin(context.Background(), key); again != nil || status != DispatchRecoveryRequired {
				t.Fatal("abandoned request replayed")
			}
			// Historical binding timestamps are deliberately old and remain recoverable.
			result, err := j.recover(context.Background(), key)
			want := DispatchNotDispatched
			if armed {
				want = DispatchIndeterminate
			}
			if err != nil || result.State != "closed" || result.Outcome != want {
				t.Fatal("wrong recovered state", err, result)
			}
			again, err := j.recover(context.Background(), key)
			if err != nil || again != result {
				t.Fatal("recovery was not idempotent")
			}
			if tx, status := j.begin(context.Background(), key); tx != nil || status != DispatchRecorded {
				t.Fatal("closed tombstone lost")
			}
		})
	}
}

type failJournalFile struct {
	journalFile
	writes int
	at     int
	after  bool
}

func (f *failJournalFile) write(data []byte) error {
	f.writes++
	if f.writes >= f.at {
		if f.after {
			if err := f.journalFile.write(data); err != nil {
				return err
			}
		}
		return errors.New("injected journal persistence failure")
	}
	return f.journalFile.write(data)
}
func TestJournalPersistenceFailuresNeverAuthorizeReplay(t *testing.T) {
	for _, at := range []int{1, 2, 3} {
		for _, after := range []bool{false, true} {
			name := []string{"", "intent", "armed", "outcome"}[at]
			if after {
				name += "-after-write"
			}
			t.Run(name, func(t *testing.T) {
				d, in, p, _, _ := dispatchFixture(t)
				ctx := dispatchPeer(t, true)
				d.journal.open = func(j *Journal, id string, create bool) (journalFile, error) {
					f, err := openJournalFile(j, id, create)
					if err != nil {
						return nil, err
					}
					return &failJournalFile{journalFile: f, at: at, after: after}, nil
				}
				a, _ := d.Prepare(in)
				got := a.Run(ctx)
				want := DispatchUnavailable
				if at == 3 {
					want = DispatchIndeterminate
				}
				if got != want {
					t.Fatalf("got %s want %s", got, want)
				}
				if (at < 3 && p.executions.Load() != 0) || (at == 3 && p.executions.Load() != 1) {
					t.Fatal("execution crossed failed durability boundary")
				}
				d.journal.open = openJournalFile
				if at == 1 && !after {
					return
				} // No durable claim and no audit/provider work occurred.
				before := p.executions.Load()
				a, _ = d.Prepare(in)
				got = a.Run(ctx)
				if got != DispatchRecoveryRequired && got != DispatchRecorded {
					t.Fatal("failed commit became a new request")
				}
				_, err := d.Recover(ctx, in, recoveryAuthorization(in))
				if err != nil {
					t.Fatal(err)
				}
				if p.executions.Load() != before {
					t.Fatal("recovery dispatched provider")
				}
			})
		}
	}
}
func TestJournalRejectsUnsafeFilesAndPaths(t *testing.T) {
	for _, attack := range []string{"state-missing", "lock-missing", "state-symlink", "state-hardlink", "state-mode", "lock-symlink", "lock-hardlink", "lock-fifo", "directory-symlink", "directory-mode", "oversize", "truncated", "unknown-field", "duplicate-field", "digest", "missing-field"} {
		t.Run(attack, func(t *testing.T) {
			j, key, _ := unitJournal(t)
			a, status := j.begin(context.Background(), key)
			if a == nil {
				t.Fatal(status)
			}
			a.file.close()
			p := journalPath(j, key.RequestID)
			lock := filepath.Join(filepath.Dir(p), key.RequestID+".lock")
			must := func(err error) {
				if err != nil {
					t.Fatal(err)
				}
			}
			data, err := os.ReadFile(p)
			must(err)
			switch attack {
			case "state-missing":
				must(os.Remove(p))
			case "lock-missing":
				must(os.Remove(lock))
			case "state-symlink":
				must(os.Rename(p, p+".kept"))
				must(os.Symlink(p+".kept", p))
			case "state-hardlink":
				must(os.Link(p, p+".linked"))
			case "state-mode":
				must(os.Chmod(p, 0644))
			case "lock-symlink":
				must(os.Rename(lock, lock+".kept"))
				must(os.Symlink(lock+".kept", lock))
			case "lock-hardlink":
				must(os.Link(lock, lock+".linked"))
			case "lock-fifo":
				must(os.Remove(lock))
				must(unix.Mkfifo(lock, 0600))
			case "directory-symlink":
				dir := filepath.Dir(p)
				must(os.Rename(dir, dir+".kept"))
				must(os.Symlink(dir+".kept", dir))
			case "directory-mode":
				must(os.Chmod(filepath.Dir(p), 0755))
			case "oversize":
				must(os.WriteFile(p, bytes.Repeat([]byte("x"), maxJournalRecordBytes+1), 0600))
			case "truncated":
				must(os.WriteFile(p, data[:len(data)/2], 0600))
			case "unknown-field":
				must(os.WriteFile(p, append([]byte("{\"unknown\":true,"), data[1:]...), 0600))
			case "duplicate-field":
				must(os.WriteFile(p, append([]byte("{\"Phase\":\"intent\","), data[1:]...), 0600))
			case "digest":
				data = bytes.Replace(data, []byte("sha256:"), []byte("sha257:"), 1)
				must(os.WriteFile(p, data, 0600))
			case "missing-field":
				data = bytes.Replace(data, []byte("\"Binding\":null,"), nil, 1)
				must(os.WriteFile(p, data, 0600))
			}
			if a, status := j.begin(context.Background(), key); a != nil || status != DispatchUnavailable {
				if a != nil {
					a.file.close()
				}
				t.Fatal("unsafe journal accepted")
			}
		})
	}
}
func TestJournalNamespaceAndIdentityIsolation(t *testing.T) {
	j, key, _ := unitJournal(t)
	a, status := j.begin(context.Background(), key)
	if a == nil {
		t.Fatal(status)
	}
	a.file.close()
	for _, edit := range []func(*journalIdentity){func(k *journalIdentity) { k.UID++ }, func(k *journalIdentity) { k.GID++ }, func(k *journalIdentity) { k.Subject.ID = "other" }, func(k *journalIdentity) { k.IntentDigest = fixtureDigest }} {
		other := key
		edit(&other)
		if a, status := j.begin(context.Background(), other); a != nil || status != DispatchConflict {
			t.Fatal("different identity inherited request")
		}
	}
	otherID := "018f0c3a-7b2d-7e11-8c12-0242ac120003"
	other, _ := NewJournal(j.root, otherID)
	otherKey := key
	otherKey.TOPSID = otherID
	a, status = other.begin(context.Background(), otherKey)
	if a == nil {
		t.Fatal(status)
	}
	a.file.close()
	for _, root := range []string{"relative", "/", j.root + "/../other"} {
		if _, err := NewJournal(root, dispatchTOPS); err == nil {
			t.Fatal("unsafe root accepted")
		}
	}
}

func TestJournalCrashWorker(t *testing.T) {
	stage := os.Getenv("SQV16_CRASH_STAGE")
	if stage == "" {
		return
	}
	root := os.Getenv("SQV16_CRASH_ROOT")
	data, err := os.ReadFile(filepath.Join(root, "fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	var in DispatchIntent
	if json.Unmarshal(data, &in) != nil {
		t.Fatal("fixture decode")
	}
	d, _, p, a, _ := dispatchFixture(t)
	d.journal, err = NewJournal(root, dispatchTOPS)
	if err != nil {
		t.Fatal(err)
	}
	pause := func() { os.Stdout.Write([]byte("READY\n")); select {} }
	if stage == "intent" {
		a.after = pause
	}
	p.execute = func(context.Context, UseBinding) DispatchStatus {
		if stage == "armed" {
			pause()
		}
		f, err := os.OpenFile(filepath.Join(root, "effect"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = f.Write([]byte("x")); err != nil {
			t.Fatal(err)
		}
		if f.Sync() != nil {
			t.Fatal("effect sync")
		}
		f.Close()
		if stage == "effect" {
			pause()
		}
		return DispatchDelivered
	}
	attempt, err := d.Prepare(in)
	if err != nil {
		t.Fatal(err)
	}
	if attempt.Run(dispatchPeer(t, true)) != DispatchDelivered {
		t.Fatal("worker dispatch failed")
	}
	if stage == "closed" {
		pause()
	}
	t.Fatal("unknown worker stage")
}
func TestJournalSIGKILLRecoveryNeverRedelivers(t *testing.T) {
	for _, stage := range []string{"intent", "armed", "effect", "closed"} {
		t.Run(stage, func(t *testing.T) {
			d, in, p, _, _ := dispatchFixture(t)
			root := d.journal.root
			data, _ := json.Marshal(in)
			if err := os.WriteFile(filepath.Join(root, "fixture.json"), data, 0600); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(os.Args[0], "-test.run=^TestJournalCrashWorker$")
			cmd.Env = append(os.Environ(), "SQV16_CRASH_STAGE="+stage, "SQV16_CRASH_ROOT="+root)
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			if err = cmd.Start(); err != nil {
				t.Fatal(err)
			}
			defer func() {
				if cmd.ProcessState == nil {
					cmd.Process.Kill()
					cmd.Wait()
				}
			}()
			ready := make(chan string, 1)
			go func() { line, _ := bufio.NewReader(stdout).ReadString('\n'); ready <- line }()
			select {
			case line := <-ready:
				if line != "READY\n" {
					t.Fatal("worker did not reach checkpoint")
				}
			case <-time.After(10 * time.Second):
				t.Fatal("worker checkpoint timeout")
			}
			ctx := dispatchPeer(t, true)
			if stage != "closed" {
				if _, err := d.Recover(ctx, in, recoveryAuthorization(in)); err == nil {
					t.Fatal("recovered a live process")
				}
			}
			if err = cmd.Process.Kill(); err != nil {
				t.Fatal(err)
			}
			err = cmd.Wait()
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ProcessState.Sys().(syscall.WaitStatus).Signal() != syscall.SIGKILL {
				t.Fatal("worker was not killed at checkpoint")
			}
			result, err := d.Recover(ctx, in, recoveryAuthorization(in))
			if err != nil {
				t.Fatal(err)
			}
			want := DispatchIndeterminate
			if stage == "intent" {
				want = DispatchNotDispatched
			}
			if stage == "closed" {
				want = DispatchDelivered
			}
			if result.Outcome != want || p.executions.Load() != 0 || p.pins.Load() != 0 {
				t.Fatal("unsafe crash recovery", result)
			}
			effect, _ := os.ReadFile(filepath.Join(root, "effect"))
			expected := 0
			if stage == "effect" || stage == "closed" {
				expected = 1
			}
			if len(effect) != expected {
				t.Fatal("effect count changed")
			}
			again, _ := d.Prepare(in)
			if again.Run(ctx) != DispatchRecorded || p.executions.Load() != 0 {
				t.Fatal("recovered request replayed")
			}
		})
	}
}
