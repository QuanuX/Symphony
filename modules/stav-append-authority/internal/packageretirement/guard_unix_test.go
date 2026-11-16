//go:build darwin || linux

package packageretirement

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testHash(b []byte) string { h := sha256.Sum256(b); return "sha256:" + hex.EncodeToString(h[:]) }
func fixture(t *testing.T) (string, Identity) {
	t.Helper()
	prefix, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	binary := []byte("owned executable fixture\n")
	receipt := []byte("exact receipt fixture\n")
	id := Identity{Owner: "fixture-owner", Version: "1", ReceiptDigest: testHash(receipt), BinaryDigest: testHash(binary), BinarySize: uint64(len(binary)), BinaryRelative: "libexec/fixture/1/server", ReceiptRelative: "share/symphony/receipts/fixture/1/install-receipt.json"}
	g, err := PrepareInstall(prefix, id, true)
	if err != nil {
		t.Fatal(err)
	}
	for path, data := range map[string][]byte{id.BinaryRelative: binary, id.ReceiptRelative: receipt} {
		absolute := filepath.Join(prefix, path)
		if err := os.MkdirAll(filepath.Dir(absolute), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(absolute, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := g.Activate(); err != nil {
		t.Fatal(err)
	}
	if err := g.Close(); err != nil {
		t.Fatal(err)
	}
	return prefix, id
}

// This helper is a real subprocess, including actual OS file-lock lifetime.
func TestRetirementChild(t *testing.T) {
	mode := os.Getenv("SYMPHONY_RETIREMENT_TEST_CHILD")
	if mode == "" {
		return
	}
	var id Identity
	if err := json.Unmarshal([]byte(os.Getenv("SYMPHONY_RETIREMENT_TEST_ID")), &id); err != nil {
		t.Fatal(err)
	}
	prefix := os.Getenv("SYMPHONY_RETIREMENT_TEST_ROOT")
	var g *Lease
	var err error
	if mode == "use" {
		g, err = AcquireUse(prefix, id)
	} else {
		g, err = BeginRetirement(prefix, id)
	}
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	if err := g.VerifyFile(id.BinaryRelative, id.BinaryDigest, id.BinarySize); err != nil {
		t.Fatal(err)
	}
	fmt.Println("lease-held")
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
}
func child(t *testing.T, prefix string, id Identity, mode string) *exec.Cmd {
	t.Helper()
	body, _ := json.Marshal(id)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(executable, "-test.run=^TestRetirementChild$")
	cmd.Env = append(os.Environ(), "SYMPHONY_RETIREMENT_TEST_CHILD="+mode, "SYMPHONY_RETIREMENT_TEST_ROOT="+prefix, "SYMPHONY_RETIREMENT_TEST_ID="+string(body))
	in, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = in.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	})
	ready := make(chan string, 1)
	go func() { line, _ := bufio.NewReader(out).ReadString('\n'); ready <- line }()
	select {
	case line := <-ready:
		if line != "lease-held\n" {
			t.Fatalf("child did not hold lease: %q", line)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("child lease barrier timed out")
	}
	return cmd
}
func TestLiveChildClosesAdmissionUntilTerminalRelease(t *testing.T) {
	prefix, id := fixture(t)
	cmd := child(t, prefix, id, "use")
	if g, err := BeginRetirement(prefix, id); err == nil {
		g.Close()
		t.Fatal("retirement reclaimed a live child")
	}
	if _, phase, err := Inspect(prefix, id.Owner, id.Version); err != nil || phase != "retiring" {
		t.Fatalf("durable closed admission missing: %s %v", phase, err)
	}
	if g, err := AcquireUse(prefix, id); err == nil {
		g.Close()
		t.Fatal("new use admitted during retirement")
	}
	if _, err := os.Stat(filepath.Join(prefix, id.BinaryRelative)); err != nil {
		t.Fatal("busy refusal changed payload", err)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	cmd.Process = nil
	g, err := BeginRetirement(prefix, id)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	if err := g.RemoveFile(id.BinaryRelative, id.BinaryDigest, id.BinarySize); err != nil {
		t.Fatal(err)
	}
	data, err := g.ReadFile(id.ReceiptRelative, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if err := g.RemoveFile(id.ReceiptRelative, testHash(data), uint64(len(data))); err != nil {
		t.Fatal(err)
	}
	if err := g.FinishRetirement(); err != nil {
		t.Fatal(err)
	}
	if err := g.Close(); err != nil {
		t.Fatal(err)
	}
	if _, phase, err := Inspect(prefix, id.Owner, id.Version); err != nil || phase != "retired" {
		t.Fatalf("tombstone missing: %s %v", phase, err)
	}
	if g, err := PrepareInstall(prefix, id, true); err == nil {
		g.Close()
		t.Fatal("retired epoch reopened")
	}
}
func TestControllerCrashPreservesClosedAdmission(t *testing.T) {
	prefix, id := fixture(t)
	cmd := child(t, prefix, id, "retire")
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	cmd.Process = nil
	if g, err := AcquireUse(prefix, id); err == nil {
		g.Close()
		t.Fatal("controller death reopened admission")
	}
	g, err := BeginRetirement(prefix, id)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	if g.State() != "retiring" {
		t.Fatal("recovery lost retirement intent")
	}
	if err := g.VerifyFile(id.BinaryRelative, id.BinaryDigest, id.BinarySize); err != nil {
		t.Fatal(err)
	}
}
func TestReplacedUseLockCannotBypassLiveConsumer(t *testing.T) {
	prefix, id := fixture(t)
	use, err := AcquireUse(prefix, id)
	if err != nil {
		t.Fatal(err)
	}
	defer use.Close()
	lock := filepath.Join(prefix, guardPath(id.Owner, id.Version), "use.lock")
	if err := os.Remove(lock); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lock, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if g, err := BeginRetirement(prefix, id); err == nil {
		g.Close()
		t.Fatal("replacement lock bypassed held use")
	}
	if _, err := os.Stat(filepath.Join(prefix, id.BinaryRelative)); err != nil {
		t.Fatal(err)
	}
}
func TestUnsafeGuardAndRootRefuse(t *testing.T) {
	for _, kind := range []string{"symlink", "hardlink", "state", "root"} {
		t.Run(kind, func(t *testing.T) {
			prefix, id := fixture(t)
			lock := filepath.Join(prefix, guardPath(id.Owner, id.Version), "use.lock")
			switch kind {
			case "symlink":
				target := filepath.Join(prefix, "other")
				if err := os.WriteFile(target, nil, 0o644); err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(lock); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, lock); err != nil {
					t.Fatal(err)
				}
			case "hardlink":
				if err := os.Link(lock, filepath.Join(prefix, "other")); err != nil {
					t.Fatal(err)
				}
			case "state":
				if err := os.WriteFile(filepath.Join(prefix, guardPath(id.Owner, id.Version), "state.json"), []byte("{}\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			case "root":
				moved := prefix + "-moved"
				if err := os.Rename(prefix, moved); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = os.RemoveAll(moved) })
				if err := os.Mkdir(prefix, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			if g, err := AcquireUse(prefix, id); err == nil {
				g.Close()
				t.Fatal("unsafe package guard admitted use")
			}
			if g, err := BeginRetirement(prefix, id); err == nil {
				g.Close()
				t.Fatal("unsafe package guard admitted reclamation")
			}
		})
	}
}
func TestSharedRootMarkersRefuseNativeMutation(t *testing.T) {
	for _, marker := range []string{".symphony-lifecycle-ownership-v1.json", "share/symphony/receipts/symphony-root-ownership/1/install-receipt.json"} {
		t.Run(filepath.Base(filepath.Dir(marker)), func(t *testing.T) {
			prefix, id := fixture(t)
			path := filepath.Join(prefix, marker)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, nil, 0o600); err != nil {
				t.Fatal(err)
			}
			if g, err := BeginRetirement(prefix, id); err == nil {
				g.Close()
				t.Fatal("native removal bypassed shared-root marker")
			} else if !strings.Contains(err.Error(), "qxctl-administered") {
				t.Fatal(err)
			}
			if _, phase, err := Inspect(prefix, id.Owner, id.Version); err != nil || phase != "open" {
				t.Fatalf("refusal mutated admission: %s %v", phase, err)
			}
		})
	}
}
func TestConcurrentAdmissionAndRetirementHaveOneWinner(t *testing.T) {
	for range 12 {
		prefix, id := fixture(t)
		start := make(chan struct{})
		used := make(chan *Lease, 1)
		retired := make(chan *Lease, 1)
		go func() { <-start; g, _ := AcquireUse(prefix, id); used <- g }()
		go func() { <-start; g, _ := BeginRetirement(prefix, id); retired <- g }()
		close(start)
		a, b := <-used, <-retired
		if a != nil && b != nil {
			a.Close()
			b.Close()
			t.Fatal("start and reclamation both obtained conflicting leases")
		}
		if a != nil {
			a.Close()
		}
		if b != nil {
			b.Close()
		}
	}
}
func TestFreshGuardCannotAdoptExistingBytes(t *testing.T) {
	prefix, id := fixture(t)
	if err := os.RemoveAll(filepath.Join(prefix, ".symphony-package-retirement")); err != nil {
		t.Fatal(err)
	}
	if g, err := PrepareInstall(prefix, id, true); err == nil {
		g.Close()
		t.Fatal("existing package was adopted without quiescence")
	}
}

func TestClosedAdmissionAllowsBoundedRecoveryButNotStarts(t *testing.T) {
	prefix, id := fixture(t)
	retirement, err := BeginRetirement(prefix, id)
	if err != nil {
		t.Fatal(err)
	}
	if recovery, err := AcquireRecoveryUse(prefix, id); err == nil {
		recovery.Close()
		t.Fatal("recovery overlapped exclusive reclamation")
	}
	if err := retirement.Close(); err != nil {
		t.Fatal(err)
	}
	recovery, err := AcquireRecoveryUse(prefix, id)
	if err != nil {
		t.Fatal(err)
	}
	if start, err := AcquireUse(prefix, id); err == nil {
		start.Close()
		t.Fatal("recovery reopened start admission")
	}
	if reclaim, err := BeginRetirement(prefix, id); err == nil {
		reclaim.Close()
		t.Fatal("reclamation overlapped active recovery")
	}
	if err := recovery.Close(); err != nil {
		t.Fatal(err)
	}
	reclaim, err := BeginRetirement(prefix, id)
	if err != nil {
		t.Fatal(err)
	}
	defer reclaim.Close()
}
