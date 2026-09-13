package scvtransfer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func fixture(t *testing.T) Intent {
	t.Helper()
	raw, err := os.ReadFile("../../../../modules/scv-graph-duckdb-connector/tests/fixtures/maintenance-wire.json")
	if err != nil {
		t.Fatal(err)
	}
	all, err := Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	p := all["plan"].(map[string]any)["result"].(map[string]any)
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	p["input"].(map[string]any)["target_root"] = root
	plan, err := Seal(p)
	if err != nil {
		t.Fatal(err)
	}
	i := Intent{Protocol: Protocol, SourceRoot: "/source", TargetRoot: root, Plan: plan}
	raw, err = Seal(i)
	if err != nil {
		t.Fatal(err)
	}
	i, err = ReadIntent(raw)
	if err != nil {
		t.Fatal(err)
	}
	return i
}
func TestTransferJournalAdjacentRecovery(t *testing.T) {
	i := fixture(t)
	n := 0
	for {
		done := false
		err := With(i.TargetRoot, true, &i, func(j *Journal) error {
			phase, id := Next(i, j.Events)
			if phase == "" {
				done = true
				return nil
			}
			result := ""
			if phase == "prepared" || phase == "committed" {
				result = "sha256:" + string(makeHex())
			}
			n++
			return j.Append(phase, id, result)
		})
		if err != nil {
			t.Fatal(err)
		}
		if done {
			break
		}
	}
	if n < 5 {
		t.Fatal(n)
	}
	if err := With(i.TargetRoot, false, nil, func(j *Journal) error {
		if p, _ := Next(i, j.Events); p != "" {
			t.Fatal(p)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
func makeHex() []byte {
	b := make([]byte, 64)
	for n := range b {
		b[n] = 'a'
	}
	return b
}
func TestTransferJournalConflictAndStrictFields(t *testing.T) {
	i := fixture(t)
	if err := With(i.TargetRoot, true, &i, func(j *Journal) error { return nil }); err != nil {
		t.Fatal(err)
	}
	other := i
	other.SourceRoot = "/other"
	raw, _ := Seal(other)
	other, _ = ReadIntent(raw)
	if With(i.TargetRoot, true, &other, func(j *Journal) error { t.Fatal("conflict entered"); return nil }) == nil {
		t.Fatal("accepted conflicting intent")
	}
	m, _ := Decode(raw)
	m["unknown"] = true
	raw, _ = Seal(m)
	if _, err := ReadIntent(raw); err == nil {
		t.Fatal("unknown journal fields")
	}
}
func TestTransferJournalScratchAndGap(t *testing.T) {
	i := fixture(t)
	if err := With(i.TargetRoot, true, &i, func(j *Journal) error { return nil }); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(i.TargetRoot, Directory)
	if err := os.WriteFile(filepath.Join(dir, "pending.json"), []byte("interrupted"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := With(i.TargetRoot, true, &i, func(j *Journal) error { phase, id := Next(i, j.Events); return j.Append(phase, id, "") }); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "event-00.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "event-02.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if With(i.TargetRoot, false, nil, func(j *Journal) error { return nil }) == nil {
		t.Fatal("accepted nonadjacent progress")
	}
}
func TestTransferJournalUnsafeStorage(t *testing.T) {
	for _, kind := range []string{"symlink", "hardlink", "mode", "foreign"} {
		t.Run(kind, func(t *testing.T) {
			i := fixture(t)
			if kind == "foreign" {
				os.WriteFile(filepath.Join(i.TargetRoot, "caller"), []byte("keep"), 0o600)
				if With(i.TargetRoot, true, &i, func(j *Journal) error { return nil }) == nil {
					t.Fatal("foreign destination")
				}
				if _, err := os.Stat(filepath.Join(i.TargetRoot, Directory)); !os.IsNotExist(err) {
					t.Fatal("created journal in foreign root")
				}
				return
			}
			if err := With(i.TargetRoot, true, &i, func(j *Journal) error { return nil }); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(i.TargetRoot, Directory, "intent.json")
			switch kind {
			case "mode":
				os.Chmod(path, 0o644)
			case "hardlink":
				os.Link(path, filepath.Join(t.TempDir(), "linked"))
			case "symlink":
				saved := filepath.Join(t.TempDir(), "saved")
				os.Rename(path, saved)
				os.Symlink(saved, path)
			}
			if With(i.TargetRoot, false, nil, func(j *Journal) error { return nil }) == nil {
				t.Fatal("unsafe storage accepted")
			}
		})
	}
}
func TestTransferJournalPhaseValidation(t *testing.T) {
	i := fixture(t)
	for _, phase := range []string{"committed", "complete", "unknown"} {
		if err := With(i.TargetRoot, true, &i, func(j *Journal) error { return j.Append(phase, "", "") }); err == nil {
			t.Fatal("phase accepted", phase)
		}
	}
	raw, _ := Seal(Event{Protocol: "symphony.qxctl.scv-index-transfer-event.v1", TransferDigest: i.Digest, Sequence: 0, Previous: i.Digest, Phase: "prepare_pending", OperationID: "x"})
	var m map[string]any
	json.Unmarshal(raw, &m)
	m["sequence"] = nil
	raw, _ = Seal(m)
	if _, err := ReadEvent(raw); err == nil {
		t.Fatal("null sequence accepted")
	}
}
