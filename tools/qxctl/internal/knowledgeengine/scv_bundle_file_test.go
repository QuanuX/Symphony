package knowledgeengine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func bundleFileFixture(t *testing.T, raw []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "input.json")
	if err := os.WriteFile(p, raw, 0600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestSCVBundleFileReaderAcceptsExplicitLargeInputOnly(t *testing.T) {
	shared := map[string]any{"body": strings.Repeat("x", 65000)}
	items := []any{}
	for i := 0; i < 20; i++ {
		items = append(items, shared)
	}
	child := bundleTestRaw(t, map[string]any{"items": items})
	raw := bundleTestRaw(t, map[string]any{"operation": "composition_explore", "input": json.RawMessage(child)})
	if len(raw) <= 1<<20 || len(raw) > 4<<20 {
		t.Fatal("fixture must be between1MiB and4MiB")
	}
	path := bundleFileFixture(t, raw)
	read, err := ReadSCVBundlePackPayload(path)
	if err != nil || string(read) != string(raw) {
		t.Fatal("explicit bounded reader lost input", err)
	}
	if _, err = ReadPayload(path); err == nil {
		t.Fatal("legacy reader accepted an oversized request")
	}
	bundle, err := SCVBundleEncode(child)
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle) >= 1<<20 {
		t.Fatal("shared fixture did not fit new transport")
	}
	decoded, err := SCVBundleDecode(bundle)
	if err != nil || string(decoded) != string(child) {
		t.Fatal("large logical child changed", err)
	}
}

func TestSCVBundleFileReaderRejectsUnsafeOrInvalidFiles(t *testing.T) {
	for name, raw := range map[string][]byte{"oversize": []byte(strings.Repeat(" ", 4<<20) + "{}"), "duplicate": []byte(`{"operation":"composition_explore","input":{},"input":{}}`), "malformed": []byte(`{"input":`), "root_array": []byte(`[]`), "floating": []byte(`{"input":{"x":1.5}}`)} {
		t.Run(name, func(t *testing.T) {
			if _, err := ReadSCVBundlePackPayload(bundleFileFixture(t, raw)); err == nil {
				t.Fatal("invalid local packing file accepted")
			}
		})
	}
	t.Run("symlink", func(t *testing.T) {
		path := bundleFileFixture(t, []byte(`{"operation":"composition_explore","input":{}}`))
		link := filepath.Join(filepath.Dir(path), "link.json")
		if err := os.Symlink(path, link); err != nil {
			t.Fatal(err)
		}
		if _, err := ReadSCVBundlePackPayload(link); err == nil {
			t.Fatal("leaf symlink accepted")
		}
	})
	t.Run("directory", func(t *testing.T) {
		if _, err := ReadSCVBundlePackPayload(t.TempDir()); err == nil {
			t.Fatal("directory accepted")
		}
	})
	t.Run("empty", func(t *testing.T) {
		if _, err := ReadSCVBundlePackPayload(""); err == nil {
			t.Fatal("implicit input accepted")
		}
	})
}

func TestSCVBundlePackEnvelopeDoesNotWidenLogicalValueBound(t *testing.T) {
	// root object + key + outer array =3 values;255 shared arrays of128
	// values each plus125 scalar items bring the logical child to exactly32768.
	shared := []any{}
	for i := 0; i < 127; i++ {
		shared = append(shared, i)
	}
	items := []any{}
	for i := 0; i < 255; i++ {
		items = append(items, shared)
	}
	for i := 0; i < 125; i++ {
		items = append(items, i)
	}
	for _, extra := range []bool{false, true} {
		name := "at_boundary"
		if extra {
			name = "over_boundary"
		}
		t.Run(name, func(t *testing.T) {
			selected := append([]any{}, items...)
			if extra {
				selected = append(selected, 0)
			}
			child := bundleTestRaw(t, map[string]any{"items": selected})
			raw := bundleTestRaw(t, map[string]any{"operation": "composition_explore", "input": json.RawMessage(child)})
			path := bundleFileFixture(t, raw)
			if _, err := ReadSCVBundlePackPayload(path); err != nil {
				t.Fatal("small explicit envelope allowance was not available", err)
			}
			if _, err := ReadPayload(path); err == nil {
				t.Fatal("legacy default value ceiling widened")
			}
			encoded, err := SCVBundleEncode(child)
			if extra {
				if err == nil {
					t.Fatal("packing envelope allowance leaked into logical child")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := SCVBundleDecode(encoded)
			if err != nil || string(decoded) != string(child) {
				t.Fatal("exact logical value boundary failed", err)
			}
		})
	}
}
