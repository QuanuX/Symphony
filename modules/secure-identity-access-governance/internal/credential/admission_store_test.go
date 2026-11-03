//go:build darwin || linux

package credential

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/peerauth"
)

func durableAdmission(t *testing.T) (*AdmissionRegistry, string, *dispatchProvider, context.Context, DispatchIntent, AdmissionGrant) {
	t.Helper()
	_, _, in, native, ctx, grant := admissionFixture(t)
	root := t.TempDir()
	if err := os.Chmod(root, 0700); err != nil {
		t.Fatal(err)
	}
	r, err := OpenAdmissionRegistry(root, dispatchTOPS, fixtureDigest, native)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close() })
	return r, root, native, ctx, in, grant
}
func admissionState(root string) string {
	return filepath.Join(root, "credential-admission", dispatchTOPS, "active.json")
}
func TestAdmissionStoreRestartRevocationAndAuthority(t *testing.T) {
	r, root, n, ctx, in, g := durableAdmission(t)
	if rev, s := r.Replace(ctx, 1, []AdmissionGrant{g}); rev != 2 || s != AdmissionUpdated {
		t.Fatal(rev, s)
	}
	if _, e := OpenAdmissionRegistry(root, dispatchTOPS, fixtureDigest, n); e == nil {
		t.Fatal("second owner acquired state")
	}
	pin := admissionAcquire(t, r, ctx, in)
	if r.Close() != AdmissionBusy {
		t.Fatal("closed with active native pin")
	}
	pin.Release()
	if r.Close() != AdmissionUpdated || r.Revision() != 0 {
		t.Fatal("close")
	}
	r2, e := OpenAdmissionRegistry(root, dispatchTOPS, fixtureDigest, n)
	if e != nil {
		t.Fatal(e)
	}
	defer r2.Close()
	if r2.Revision() != 2 {
		t.Fatal("revision reset")
	}
	pin = admissionAcquire(t, r2, ctx, in)
	pin.Release()
	if _, s := r2.Replace(ctx, 1, nil); s != AdmissionConflict {
		t.Fatal("stale revision")
	}
	if rev, s := r2.Replace(ctx, 2, nil); rev != 3 || s != AdmissionUpdated {
		t.Fatal(rev, s)
	}
	r2.Close()
	before, _ := os.ReadFile(admissionState(root))
	if _, e := OpenAdmissionRegistry(root, dispatchTOPS, "sha256:"+string(bytes.Repeat([]byte("f"), 64)), n); e == nil {
		t.Fatal("authority drift admitted")
	}
	after, _ := os.ReadFile(admissionState(root))
	if !bytes.Equal(before, after) {
		t.Fatal("authority drift changed store")
	}
	r3, e := OpenAdmissionRegistry(root, dispatchTOPS, fixtureDigest, n)
	if e != nil {
		t.Fatal(e)
	}
	defer r3.Close()
	peer, _ := peerauth.PeerFromContext(ctx)
	if r3.Revision() != 3 {
		t.Fatal("revocation revision lost")
	}
	if p, e := r3.Pin(ctx, peer, in); e == nil || p != nil {
		t.Fatal("revoked grant restored")
	}
}
func TestAdmissionStoreWriteUncertaintyFencesOldGrants(t *testing.T) {
	r, root, n, ctx, in, g := durableAdmission(t)
	if _, s := r.Replace(ctx, 1, []AdmissionGrant{g}); s != AdmissionUpdated {
		t.Fatal(s)
	}
	persist := r.persist
	r.persist = func(rev uint64, grants map[admissionKey]AdmissionGrant) error {
		if e := persist(rev, grants); e != nil {
			return e
		}
		return errors.New("simulated lost durable acknowledgement")
	}
	if _, s := r.Replace(ctx, 2, nil); s != AdmissionUnavailable {
		t.Fatal(s)
	}
	peer, _ := peerauth.PeerFromContext(ctx)
	if p, e := r.Pin(ctx, peer, in); e == nil || p != nil {
		t.Fatal("uncertainty reused old grants")
	}
	if _, s := r.Replace(ctx, 2, []AdmissionGrant{g}); s != AdmissionUnavailable {
		t.Fatal("uncertain state accepted update")
	}
	r.Close()
	recovered, e := OpenAdmissionRegistry(root, dispatchTOPS, fixtureDigest, n)
	if e != nil {
		t.Fatal(e)
	}
	defer recovered.Close()
	if recovered.Revision() != 3 {
		t.Fatal("did not recover actual durable state")
	}
	if p, e := recovered.Pin(ctx, peer, in); e == nil || p != nil {
		t.Fatal("durable revocation lost")
	}
}
func TestAdmissionStoreRefusalsPreserveBytes(t *testing.T) {
	r, root, _, ctx, _, g := durableAdmission(t)
	before, _ := os.ReadFile(admissionState(root))
	for _, grants := range [][]AdmissionGrant{make([]AdmissionGrant, MaxAdmissionGrants+1), {g, g}} {
		if _, s := r.Replace(ctx, 1, grants); s != AdmissionInvalid {
			t.Fatal(s)
		}
	}
	g.UID = ^uint32(0)
	if _, s := r.Replace(ctx, 1, []AdmissionGrant{g}); s != AdmissionInvalid {
		t.Fatal("UID sentinel")
	}
	after, _ := os.ReadFile(admissionState(root))
	if !bytes.Equal(before, after) {
		t.Fatal("refusal changed persisted state")
	}
}
func TestAdmissionStoreExpiredRecoveryNeverPinsNative(t *testing.T) {
	r, root, n, ctx, in, g := durableAdmission(t)
	if _, s := r.Replace(ctx, 1, []AdmissionGrant{g}); s != AdmissionUpdated {
		t.Fatal(s)
	}
	r.Close()
	other, e := OpenAdmissionRegistry(root, dispatchTOPS, fixtureDigest, n)
	if e != nil {
		t.Fatal(e)
	}
	defer other.Close()
	other.now = func() time.Time { return g.ExpiresAt }
	peer, _ := peerauth.PeerFromContext(ctx)
	before := n.pins.Load()
	if p, e := other.Pin(ctx, peer, in); e == nil || p != nil || n.pins.Load() != before {
		t.Fatal("expired persisted lease reached native")
	}
}
func TestAdmissionStoreDamagedStateFailsClosed(t *testing.T) {
	for _, mode := range []string{"missing", "digest", "unknown", "duplicate", "symlink", "permissions", "missing-lock"} {
		t.Run(mode, func(t *testing.T) {
			r, root, n, _, _, _ := durableAdmission(t)
			r.Close()
			p := admissionState(root)
			data, e := os.ReadFile(p)
			if e != nil {
				t.Fatal(e)
			}
			switch mode {
			case "missing":
				e = os.Remove(p)
			case "digest":
				data = bytes.Replace(data, []byte(`"Revision":1`), []byte(`"Revision":2`), 1)
				e = os.WriteFile(p, data, 0600)
			case "unknown":
				data = bytes.Replace(data, []byte("{"), []byte(`{"Unknown":false,`), 1)
				e = os.WriteFile(p, data, 0600)
			case "duplicate":
				data = bytes.Replace(data, []byte("{"), []byte(`{"Revision":1,`), 1)
				e = os.WriteFile(p, data, 0600)
			case "symlink":
				e = os.Rename(p, p+".original")
				if e == nil {
					e = os.Symlink(p+".original", p)
				}
			case "permissions":
				e = os.Chmod(p, 0640)
			case "missing-lock":
				e = os.Remove(filepath.Join(filepath.Dir(p), "active.lock"))
			}
			if e != nil {
				t.Fatal(e)
			}
			if got, e := OpenAdmissionRegistry(root, dispatchTOPS, fixtureDigest, n); e == nil {
				got.Close()
				t.Fatal("damaged state accepted")
			}
		})
	}
}
func TestAdmissionStoreProcessCrashReleasesLockAndRetainsRevision(t *testing.T) {
	if root := os.Getenv("SYMPHONY_ADMISSION_CRASH_ROOT"); root != "" {
		r, e := OpenAdmissionRegistry(root, dispatchTOPS, fixtureDigest, &dispatchProvider{})
		if e != nil {
			os.Exit(91)
		}
		if _, s := r.Replace(context.Background(), 1, nil); s != AdmissionUpdated {
			os.Exit(92)
		}
		os.Stdout.WriteString("ready\n")
		select {}
	}
	root := t.TempDir()
	if e := os.Chmod(root, 0700); e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestAdmissionStoreProcessCrashReleasesLockAndRetainsRevision$")
	cmd.Env = append(os.Environ(), "SYMPHONY_ADMISSION_CRASH_ROOT="+root)
	pipe, e := cmd.StdoutPipe()
	if e != nil {
		t.Fatal(e)
	}
	if e = cmd.Start(); e != nil {
		t.Fatal(e)
	}
	defer func() { cmd.Process.Kill(); cmd.Wait() }()
	line, e := bufio.NewReader(pipe).ReadString('\n')
	if e != nil || line != "ready\n" {
		t.Fatal("child did not persist", e)
	}
	if _, e = OpenAdmissionRegistry(root, dispatchTOPS, fixtureDigest, &dispatchProvider{}); e == nil {
		t.Fatal("live child lock bypassed")
	}
	if e = cmd.Process.Kill(); e != nil {
		t.Fatal(e)
	}
	cmd.Wait()
	r, e := OpenAdmissionRegistry(root, dispatchTOPS, fixtureDigest, &dispatchProvider{})
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	if r.Revision() != 2 {
		t.Fatal("crash reset revision")
	}
}

func TestAdmissionStoreCapacitySurvivesRestart(t *testing.T) {
	r, root, n, ctx, _, g := durableAdmission(t)
	grants := make([]AdmissionGrant, MaxAdmissionGrants)
	for i := range grants {
		grants[i] = g
		grants[i].LeaseID = fmt.Sprintf("lease-%03d", i)
		grants[i].Target.Resource = fmt.Sprintf("resource-%03d", i)
	}
	if rev, status := r.Replace(ctx, 1, grants); rev != 2 || status != AdmissionUpdated {
		t.Fatal(rev, status)
	}
	before, e := os.ReadFile(admissionState(root))
	if e != nil || len(before) <= maxJournalRecordBytes {
		t.Fatal("larger admission record was not exercised", e)
	}
	if _, status := r.Replace(ctx, 2, append(grants, g)); status != AdmissionInvalid {
		t.Fatal(status)
	}
	after, _ := os.ReadFile(admissionState(root))
	if !bytes.Equal(before, after) {
		t.Fatal("capacity refusal altered state")
	}
	r.Close()
	other, e := OpenAdmissionRegistry(root, dispatchTOPS, fixtureDigest, n)
	if e != nil {
		t.Fatal(e)
	}
	defer other.Close()
	if other.Revision() != 2 || len(other.grants) != MaxAdmissionGrants {
		t.Fatal("capacity state lost")
	}
}
