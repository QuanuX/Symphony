package enrollment

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/paths"
)

func pendingTestLayout(t *testing.T) paths.Layout {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	layout := paths.Layout{ConfigDir: filepath.Join(root, "config"), StateDir: filepath.Join(root, "state"), RuntimeDir: filepath.Join(root, "runtime")}
	layout.ConfigFile = filepath.Join(layout.ConfigDir, "config.json")
	layout.IntentDir = filepath.Join(layout.StateDir, "intents-v1")
	layout.OutboxDir = filepath.Join(layout.StateDir, "outbox-v1")
	layout.Socket = filepath.Join(layout.RuntimeDir, "submit.sock")
	for _, dir := range []string{layout.ConfigDir, layout.IntentDir, layout.OutboxDir, layout.RuntimeDir} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(layout.ConfigFile, []byte("retained recovery configuration\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return layout
}

func TestUnenrollRetainsConfigurationForPendingWork(t *testing.T) {
	for _, purge := range []bool{false, true} {
		for _, store := range []string{"intents", "outbox"} {
			t.Run(store+map[bool]string{false: "-retain", true: "-purge"}[purge], func(t *testing.T) {
				layout := pendingTestLayout(t)
				dir := layout.IntentDir
				if store == "outbox" {
					dir = layout.OutboxDir
				}
				pending := filepath.Join(dir, "pending.json")
				if err := os.WriteFile(pending, []byte("unresolved evidence\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				if changed, err := Unenroll(layout, purge); err == nil || changed {
					t.Fatalf("pending work allowed configuration removal: changed=%v err=%v", changed, err)
				}
				for _, path := range []string{layout.ConfigFile, pending} {
					if data, err := os.ReadFile(path); err != nil || len(data) == 0 {
						t.Fatalf("recovery evidence changed: %s %v", path, err)
					}
				}
			})
		}
	}
}

func TestUnenrollRefusesLinkedPendingStateAndPreservesUnrelatedData(t *testing.T) {
	layout := pendingTestLayout(t)
	if err := os.Remove(layout.OutboxDir); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), layout.OutboxDir); err != nil {
		t.Fatal(err)
	}
	if changed, err := Unenroll(layout, false); err == nil || changed {
		t.Fatalf("linked state was treated as empty: %v %v", changed, err)
	}
	if _, err := os.Stat(layout.ConfigFile); err != nil {
		t.Fatal("configuration was lost")
	}
	if err := os.Remove(layout.OutboxDir); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(layout.StateDir, "user-note")
	if err := os.WriteFile(sentinel, []byte("user data"), 0o600); err != nil {
		t.Fatal(err)
	}
	if changed, err := Unenroll(layout, false); err != nil || !changed {
		t.Fatalf("empty state could not be unenrolled: %v %v", changed, err)
	}
	if data, err := os.ReadFile(sentinel); err != nil || string(data) != "user data" {
		t.Fatal("unrelated state changed")
	}
}
