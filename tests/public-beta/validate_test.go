package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func dossierRepo(t *testing.T) string {
	t.Helper()
	b, e := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if e != nil {
		t.Fatal(e)
	}
	return strings.TrimSpace(string(b))
}
func copyDossier(t *testing.T, repo string) string {
	t.Helper()
	folder := t.TempDir()
	entries, e := os.ReadDir(filepath.Join(repo, "tests/public-beta"))
	if e != nil {
		t.Fatal(e)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		b, e := os.ReadFile(filepath.Join(repo, "tests/public-beta", entry.Name()))
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(folder, entry.Name()), b, 0600); e != nil {
			t.Fatal(e)
		}
	}
	return folder
}
func changeJSON(t *testing.T, folder, name string, change func(object)) {
	t.Helper()
	path := filepath.Join(folder, name)
	var m object
	d := json.NewDecoder(strings.NewReader(string(read(path))))
	d.UseNumber()
	if e := d.Decode(&m); e != nil {
		t.Fatal(e)
	}
	change(m)
	b, e := json.MarshalIndent(m, "", "  ")
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(path, b, 0600); e != nil {
		t.Fatal(e)
	}
}
func TestHistoricalDossier(t *testing.T) {
	repo := dossierRepo(t)
	out, e := validate(repo, filepath.Join(repo, "tests/public-beta"), true)
	if e != nil {
		t.Fatal(e)
	}
	for key, want := range map[string]any{"status": "valid_historical_preparation_only", "current_product_source_verified": false, "component_roots": 73, "native_components": 60, "capability_groups": 26, "core_cells": 36, "paper_criteria": 58, "cloudflare_operations": 23, "scope_approved": false, "release_acceptance_complete": false, "publication_authorized": false} {
		if !eq(out[key], want) {
			t.Errorf("%s: got %v, want %v", key, out[key], want)
		}
	}
}
func TestHistoricalDossierRefusals(t *testing.T) {
	repo := dossierRepo(t)
	cases := []struct {
		name, want string
		mutate     func(*testing.T, string)
	}{
		{"missing-core-cell", "36 core profile/scenario", func(t *testing.T, f string) {
			changeJSON(t, f, "acceptance-mapping.json", func(m object) { m["core_cells"] = list(m["core_cells"])[1:] })
		}},
		{"duplicate-criterion", "duplicate mapped criterion", func(t *testing.T, f string) {
			changeJSON(t, f, "acceptance-mapping.json", func(m object) { rows := list(m["paper_criteria"]); rows[1] = rows[0] })
		}},
		{"publication-authority", "must not assert publication_authorized", func(t *testing.T, f string) {
			changeJSON(t, f, "dossier.json", func(m object) { m["publication_authorized"] = true })
		}},
		{"stale-fingerprint", "historical fingerprint mismatch", func(t *testing.T, f string) {
			changeJSON(t, f, "dossier.json", func(m object) {
				for k := range obj(m["source_fingerprints"]) {
					obj(m["source_fingerprints"])[k] = strings.Repeat("0", 64)
					break
				}
			})
		}},
		{"missing-capability-root", "capability roots missing or unknown", func(t *testing.T, f string) {
			changeJSON(t, f, "capability-groups.json", func(m object) { r := obj(list(m["groups"])[0]); r["component_roots"] = list(r["component_roots"])[1:] })
		}},
		{"unadmitted-runtime", "unadmitted Cloudflare runtime", func(t *testing.T, f string) {
			changeJSON(t, f, "acceptance-mapping.json", func(m object) { obj(list(m["cloudflare_operation_cells"])[0])["runtime_platform"] = "linux-arm64" })
		}},
		{"invented-artifact", "fabricated candidate artifact", func(t *testing.T, f string) {
			changeJSON(t, f, "package-plan.json", func(m object) {
				obj(obj(list(m["package_recipes"])[0])["observed_artifacts"])["archive_sha256"] = strings.Repeat("1", 64)
			})
		}},
		{"changed-native-version", "native version/SDK export drift", func(t *testing.T, f string) {
			changeJSON(t, f, "package-plan.json", func(m object) {
				for _, v := range list(m["package_recipes"]) {
					r := obj(v)
					if r["kind"] == "native_receipt_unit" {
						r["proposed_package_version"] = "99.0.0"
						break
					}
				}
			})
		}},
		{"missing-dependency", "delivery set omits native dependency closure", func(t *testing.T, f string) {
			changeJSON(t, f, "package-plan.json", func(m object) {
				for _, v := range list(m["package_recipes"]) {
					r := obj(v)
					if r["kind"] == "delivery_set_not_single_native_archive" {
						var kept []any
						for _, id := range list(r["content_recipe_ids"]) {
							if id != "PKG-NATIVE-knowledge-vector-engine-cpp" {
								kept = append(kept, id)
							}
						}
						r["content_recipe_ids"] = kept
						break
					}
				}
			})
		}},
		{"duplicate-json-key", "duplicate JSON key", func(t *testing.T, f string) {
			path := filepath.Join(f, "dossier.json")
			b := read(path)
			b = append([]byte(`{"format_version":1,`), b[1:]...)
			if e := os.WriteFile(path, b, 0600); e != nil {
				t.Fatal(e)
			}
		}},
		{"rehash-altered-inventory", "inventory projection drift", func(t *testing.T, f string) {
			changeJSON(t, f, "source-inventory.json", func(m object) { obj(list(m["component_roots"])[0])["path"] = "modules/invented" })
			changeJSON(t, f, "dossier.json", func(m object) {
				obj(m["source_inventory"])["sha256"] = sha(read(filepath.Join(f, "source-inventory.json")))
			})
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			folder := copyDossier(t, repo)
			tc.mutate(t, folder)
			_, e := validate(repo, folder, true)
			if e == nil || !strings.Contains(e.Error(), tc.want) {
				t.Fatalf("got %v, want refusal containing %q", e, tc.want)
			}
		})
	}
}
func TestCurrentSourceDriftRefused(t *testing.T) {
	repo := dossierRepo(t)
	_, e := validate(repo, filepath.Join(repo, "tests/public-beta"), false)
	if e == nil || (!strings.Contains(e.Error(), "working source drift") && !strings.Contains(e.Error(), "source outside dossier changed")) {
		t.Fatalf("expected current-source drift refusal, got %v", e)
	}
}
func TestZeroCapabilityOperationsRefused(t *testing.T) {
	repo := dossierRepo(t)
	folder := copyDossier(t, repo)
	changeJSON(t, folder, "capability-groups.json", func(m object) { obj(list(m["groups"])[0])["operations"] = json.Number("0") })
	_, err := validate(repo, folder, true)
	if err == nil || !strings.Contains(err.Error(), "capability group lacks operations") {
		t.Fatalf("numeric zero bypassed nonempty requirement: %v", err)
	}
}
func TestStrictJSONRefusals(t *testing.T) {
	for _, input := range []string{`{"text":"` + string([]byte{0xff}) + `"}`, `{"a":{"x":1,"x":2}}`, `{"a":1} {"b":2}`, `{"missing":`, strings.Repeat("[", 66) + "0" + strings.Repeat("]", 66)} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("malformed JSON accepted: %.60s", input)
				}
			}()
			parseJSON([]byte(input))
		}()
	}
}
