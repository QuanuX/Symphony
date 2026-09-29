package knowledgeengine

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

func TestSQVTransportShape(t *testing.T) {
	schema := []byte(`{"type":"object","additionalProperties":false,"required":["count","bytes"],"properties":{"count":{"type":"string","format":"uint64-decimal"},"bytes":{"type":"string","pattern":"^([0-9a-f]{2})*$","maxLength":8}}}`)
	s, _ := sqavObject(schema, 4096)
	good := map[string]any{"count": "18446744073709551615", "bytes": "00ff"}
	if !sqvTransportShape(s, good, 0) {
		t.Fatal("uint64/opaque bytes rejected")
	}
	for _, mutation := range []map[string]any{{"count": "18446744073709551616", "bytes": "00"}, {"count": "01", "bytes": "00"}, {"count": "0", "bytes": "ffgg"}, {"count": "1", "bytes": "0000000000"}, {"count": "1", "bytes": "00", "extra": true}, {"count": "1"}} {
		if sqvTransportShape(s, mutation, 0) {
			t.Fatal("invalid shape accepted")
		}
	}
	s["$ref"] = "https://untrusted.invalid/schema"
	if sqvTransportShape(s, good, 0) {
		t.Fatal("unsupported external schema accepted")
	}
}
func adminResult(t *testing.T, prefix, operation string, p map[string]any) map[string]any {
	t.Helper()
	raw, err := SCVCanonical(p)
	if err != nil {
		t.Fatal(err)
	}
	r, err := InvokeSQVAdministration(context.Background(), prefix, "0.1.0-dev", t.TempDir(), operation, raw)
	if err != nil {
		t.Fatal(operation, err)
	}
	result, err := sqavObject(r.Result, 450000)
	if err != nil {
		t.Fatal(err)
	}
	return result
}
func adminRefused(t *testing.T, prefix, operation string, p map[string]any, code string) {
	t.Helper()
	raw, _ := SCVCanonical(p)
	_, err := InvokeSQVAdministration(context.Background(), prefix, "0.1.0-dev", t.TempDir(), operation, raw)
	if err == nil {
		t.Fatal("refusal expected", operation)
	}
	if code != "" {
		var pe *ProcessError
		if !errors.As(err, &pe) || pe.Code != code {
			t.Fatalf("expected %s, got %v", code, err)
		}
	}
}
func adminFixture(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	p, err := sqavObject(raw, 262144)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
func adminClone(t *testing.T, p map[string]any) map[string]any {
	t.Helper()
	b, _ := SCVCanonical(p)
	m, err := sqavObject(b, 262144)
	if err != nil {
		t.Fatal(err)
	}
	return m
}
func adminCopyStore(t *testing.T, source string) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(root, 0700); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(source)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		b, err := os.ReadFile(filepath.Join(source, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(root, entry.Name()), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}
func adminTree(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relative, _ := filepath.Rel(root, path)
		out[relative] = info.Mode().String() + "|" + info.ModTime().UTC().String()
		if info.Mode().IsRegular() {
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			out[relative] += "|" + digestBytes(b)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}
func TestSQVAdministrationInstalled(t *testing.T) {
	prefix, fixtures := os.Getenv("SQV_ADMIN_TEST_PREFIX"), os.Getenv("SQV_ADMIN_FIXTURES")
	if prefix == "" || fixtures == "" {
		t.Skip("set SQV_ADMIN_TEST_PREFIX and SQV_ADMIN_FIXTURES for installed integration")
	}
	root, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, op := range SQVAdministrationOperations {
		for _, templates := range []bool{false, true} {
			inst, data, err := SQVAdministrationResource(prefix, "0.1.0-dev", op.Operation, templates)
			if err != nil || inst.ModuleID != op.ModuleID || !json.Valid(data) {
				t.Fatal(op.Operation, err)
			}
		}
		if _, err := InspectSQVAdministration(prefix, "latest", op.Operation); err == nil {
			t.Fatal("floating release")
		}
	}
	for _, op := range []string{"metadata_validate", "flow_validate", "conversion_validate"} {
		operation, _ := sqvAdminOperation(op)
		p := adminFixture(t, filepath.Join(root, "modules", operation.ModuleID, "tests/fixtures", op+".json"))
		result := adminResult(t, prefix, op, p)
		_, contract, err := SQVAdministrationResource(prefix, "0.1.0-dev", op, false)
		if err != nil {
			t.Fatal(err)
		}
		payload, _ := SCVCanonical(p)
		for k, v := range map[string]any{"owner": "sqv", "operation": "other", "persistent_mutation": true, "provider_observation": "performed", "request_digest": "sha256:" + strings.Repeat("0", 64), "extra": true} {
			forged := adminClone(t, result)
			forged[k] = v
			delete(forged, "result_digest")
			b, _ := SCVCanonical(forged)
			forged["result_digest"] = digestBytes(b)
			b, _ = SCVCanonical(forged)
			if ValidateSQVAdministrationResult(op, payload, b, contract) == nil {
				t.Fatal("accepted forged", k)
			}
		}
		forged := adminClone(t, result)
		forged["data"].(map[string]any)["extra"] = true
		delete(forged, "result_digest")
		b, _ := SCVCanonical(forged)
		forged["result_digest"] = digestBytes(b)
		b, _ = SCVCanonical(forged)
		if ValidateSQVAdministrationResult(op, payload, b, contract) == nil {
			t.Fatal("unknown result data")
		}
		if op == "metadata_validate" {
			d := result["data"].(map[string]any)
			inspect := map[string]any{"protocol": "symphony.sqmv.metadata-inspect-input.v1", "encoded_hex_chunks": d["encoded_hex_chunks"], "expected_reference": d["metadata_reference"], "limits": p["limits"], "projection": "all"}
			inspected := adminResult(t, prefix, "metadata_inspect", inspect)
			if !reflect.DeepEqual(inspected["data"], d) {
				t.Fatal("metadata roundtrip")
			}
		}
	}
	cases := []struct{ name, op string }{{"store", "store_inspect"}, {"empty", "store_inspect"}, {"exhausted", "store_inspect"}, {"checkpoint", "checkpoint_inspect"}, {"attempts", "attempts_inspect"}}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			p := adminFixture(t, filepath.Join(fixtures, test.name+".json"))
			selection := p["store"].(map[string]any)
			root := selection["root"].(string)
			before := adminTree(t, root)
			result := adminResult(t, prefix, test.op, p)
			if !reflect.DeepEqual(before, adminTree(t, root)) {
				t.Fatal("read-only observation mutated store")
			}
			d := result["data"].(map[string]any)
			switch test.name {
			case "store":
				if d["committed_batches"] != "1" || d["next_sequence"] != "8" {
					t.Fatal(d)
				}
			case "empty":
				if d["committed_batches"] != "0" || d["next_sequence"] != "7" {
					t.Fatal(d)
				}
			case "exhausted":
				if d["next_sequence"] != "18446744073709551615" || d["sequence_exhausted"] != true {
					t.Fatal(d)
				}
			case "checkpoint":
				if d["latest_persisted"].(map[string]any)["next_sequence"] != "8" || d["remote_commit"] != "not_established" {
					t.Fatal(d)
				}
			case "attempts":
				if d["charged_ceiling_nano_usd"] != "2015646000" || d["remaining_nano_usd"] != "2984354000" || d["unresolved"] != "1" {
					t.Fatal(d)
				}
				entry := d["entries"].([]any)[1].(map[string]any)
				if entry["persisted_outcome"] != "reserved" || entry["restart_outcome"] != "indeterminate" {
					t.Fatal(entry)
				}
				page := adminClone(t, p)
				page["offset"] = "1"
				page["limit"] = "1"
				if len(adminResult(t, prefix, test.op, page)["data"].(map[string]any)["entries"].([]any)) != 1 {
					t.Fatal("pagination")
				}
			}
			// All observer owners must refuse held writer locks, preserving the files.
			f, err := os.OpenFile(filepath.Join(root, "writer.lock"), os.O_RDWR, 0)
			if err != nil {
				t.Fatal(err)
			}
			if unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB) != nil {
				t.Fatal("fixture lock")
			}
			adminRefused(t, prefix, test.op, p, "sqv.store.busy")
			f.Close()
			if !reflect.DeepEqual(before, adminTree(t, root)) {
				t.Fatal("busy observation mutated store")
			}
			// An unfinished record never authorizes checkpoint or budget projection.
			copied := adminCopyStore(t, root)
			dirty := adminClone(t, p)
			dirty["store"].(map[string]any)["root"] = copied
			if err := os.WriteFile(filepath.Join(copied, "stage-commit"), []byte("partial"), 0600); err != nil {
				t.Fatal(err)
			}
			before = adminTree(t, copied)
			if test.op == "store_inspect" {
				d := adminResult(t, prefix, test.op, dirty)["data"].(map[string]any)
				if d["state"] != "recovery_required" || d["tail_validated"] != false {
					t.Fatal(d)
				}
			} else {
				adminRefused(t, prefix, test.op, dirty, "sqv.store.recovery_required")
			}
			if !reflect.DeepEqual(before, adminTree(t, copied)) {
				t.Fatal("dirty observation repaired files")
			}
		})
	}
	for _, name := range []string{"checkpoint-invalid-baseline", "attempts-orphan-terminal", "attempts-overspend", "attempts-invalid-ordinal", "attempts-invalid-request", "attempts-invalid-time"} {
		t.Run(name, func(t *testing.T) {
			p := adminFixture(t, filepath.Join(fixtures, name+".json"))
			op := "attempts_inspect"
			if strings.HasPrefix(name, "checkpoint") {
				op = "checkpoint_inspect"
			}
			root := p["store"].(map[string]any)["root"].(string)
			before := adminTree(t, root)
			// Storage integrity is valid. Only the semantic owner may reject this record.
			storage := adminClone(t, p)
			delete(storage, "offset")
			delete(storage, "limit")
			storage["protocol"] = "symphony.sqpv.store-inspect-input.v1"
			_ = adminResult(t, prefix, "store_inspect", storage)
			adminRefused(t, prefix, op, p, "sqv.admin.invalid")
			if !reflect.DeepEqual(before, adminTree(t, root)) {
				t.Fatal("semantic refusal mutated store")
			}
		})
	}
	for _, name := range []string{"checkpoint-empty", "attempts-empty"} {
		t.Run(name, func(t *testing.T) {
			p := adminFixture(t, filepath.Join(fixtures, name+".json"))
			op := "attempts_inspect"
			if name == "checkpoint-empty" {
				op = "checkpoint_inspect"
			}
			root := p["store"].(map[string]any)["root"].(string)
			before := adminTree(t, root)
			d := adminResult(t, prefix, op, p)["data"].(map[string]any)
			if name == "checkpoint-empty" {
				if d["checkpoint_state"] != "baseline_not_persisted" || d["latest_persisted"] != nil {
					t.Fatal(d)
				}
			} else {
				if d["attempts"] != "0" || d["charged_ceiling_nano_usd"] != "15646000" {
					t.Fatal(d)
				}
			}
			if !reflect.DeepEqual(before, adminTree(t, root)) {
				t.Fatal("empty observation synthesized persistence")
			}
		})
	}

	original := adminFixture(t, filepath.Join(fixtures, "store.json"))
	for _, kind := range []string{"head", "frame", "commit", "unknown", "symlink", "hardlink", "permissions", "missing_lock", "limit", "generation", "reference", "fifo"} {
		t.Run("reject_"+kind, func(t *testing.T) {
			p := adminClone(t, original)
			selection := p["store"].(map[string]any)
			root := adminCopyStore(t, selection["root"].(string))
			selection["root"] = root
			_ = adminResult(t, prefix, "store_inspect", p) // Prove this copy is admissible before damaging it.
			switch kind {
			case "head", "frame", "commit":
				name := "head"
				if kind != "head" {
					name = kind + "-00000000000000000007"
				}
				path := filepath.Join(root, name)
				b, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				b[len(b)/2] ^= 1
				if err = os.WriteFile(path, b, 0600); err != nil {
					t.Fatal(err)
				}
			case "unknown":
				if err = os.WriteFile(filepath.Join(root, "unknown"), []byte("x"), 0600); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				path := filepath.Join(root, "head")
				if err = os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err = os.Symlink(filepath.Join(fixtures, "store", "head"), path); err != nil {
					t.Fatal(err)
				}
			case "hardlink":
				if err = os.Link(filepath.Join(root, "head"), filepath.Join(root, "stage-head")); err != nil {
					t.Fatal(err)
				}
			case "permissions":
				if err = os.Chmod(filepath.Join(root, "head"), 0644); err != nil {
					t.Fatal(err)
				}
			case "missing_lock":
				if err = os.Remove(filepath.Join(root, "writer.lock")); err != nil {
					t.Fatal(err)
				}
			case "limit":
				selection["max_read_bytes"] = "1"
			case "generation":
				selection["expected_store_generation"] = strings.Repeat("00", 16)
			case "reference":
				selection["expected_metadata_reference"] = "sqmv1-sha256-" + strings.Repeat("0", 64)
			case "fifo":
				if err = unix.Mkfifo(filepath.Join(root, "stage-head"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			before := adminTree(t, root)
			adminRefused(t, prefix, "store_inspect", p, "")
			if !reflect.DeepEqual(before, adminTree(t, root)) {
				t.Fatal("refusal mutated files")
			}
		})
	}
}
