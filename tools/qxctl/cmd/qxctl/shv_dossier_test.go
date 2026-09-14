package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSHVDossierAdmission(t *testing.T) {
	row := map[string]any{"id": "r", "subject_id": "source-id", "manufacturer": "OEM", "model": "Topic", "hardware_class": "custom", "locator": nil, "evidence_id": nil}
	inv := map[string]any{"profile": nil, "roster": []any{row}, "evidence": []any{}}
	a := map[string]any{"id": "a", "roster_id": "r", "relation": "caller-defined", "citation_digests": []string{}}
	input := map[string]any{"component": map[string]any{"id": "component", "label": "Caller label"}, "inventory": inv, "associations": []any{a}}
	if _, _, e := dossierAdmission(jobMarshal(input)); e != nil {
		t.Fatal(e)
	}
	a["roster_id"] = "unlisted"
	if _, _, e := dossierAdmission(jobMarshal(input)); e == nil {
		t.Fatal("unlisted member admitted")
	}
	a["roster_id"] = "r"
	a["citation_digests"] = []string{"arbitrary"}
	if _, _, e := dossierAdmission(jobMarshal(input)); e == nil {
		t.Fatal("invalid citation admitted")
	}
	a["citation_digests"] = []string{}
	input["associations"] = []any{a, a}
	if _, _, e := dossierAdmission(jobMarshal(input)); e == nil {
		t.Fatal("duplicate association admitted")
	}
}
func TestSHVDossierCitationBindsWholeAssertion(t *testing.T) {
	a := map[string]any{"predicate": "identifier", "qualifier": "documented", "source_id": "source-a", "value": "MI300X"}
	h, e := dossierAssertionDigest(jobMarshal(a))
	if e != nil {
		t.Fatal(e)
	}
	for _, key := range []string{"predicate", "qualifier", "source_id", "value"} {
		prior := a[key]
		a[key] = "changed"
		other, e := dossierAssertionDigest(jobMarshal(a))
		if e != nil || other == h {
			t.Fatal("citation lost", key, e)
		}
		a[key] = prior
	}
}
func TestSHVDossierErrors(t *testing.T) {
	for _, op := range []string{"run", "graph"} {
		out, status := invokeCLI(t, "shv", "dossier", op, "--json")
		var d map[string]any
		if status == 0 || json.Unmarshal([]byte(out), &d) != nil || d["protocol"] != cliErrorProtocol {
			t.Fatal(status, out)
		}
	}
	root, e := newRootCommand()
	if e != nil {
		t.Fatal(e)
	}
	if scvJSONRequested(root, []string{"shv", "dossier", "run", "--input", "--json"}) || scvJSONRequested(root, []string{"shv", "dossier", "run", "--json=false"}) {
		t.Fatal("false JSON intent")
	}
}

func TestSHVDossierBundleRereadIdentity(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.json")
	original, e := sealSHVActivation(map[string]any{"protocol": "test", "value": "retained"})
	if e != nil {
		t.Fatal(e)
	}
	var obj map[string]json.RawMessage
	_ = json.Unmarshal(original, &obj)
	endpoint := map[string]any{"bundle_path": path, "source_root": dir, "state_root": dir, "tops_id": "test", "source_id": "test", "source_prefix": dir, "source_version": "0.1.0-dev", "kernel_prefix": dir, "kernel_version": "0.2.0-dev"}
	evidence := []map[string]json.RawMessage{{"evidence_id": jobMarshal("e"), "endpoint": jobMarshal(endpoint)}}
	observations := []map[string]json.RawMessage{{"evidence_id": jobMarshal("e"), "bundle_digest": obj["digest"]}}
	if e = os.WriteFile(path, original, 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = dossierBundles(evidence, observations); e != nil {
		t.Fatal("unchanged serialized bundle rejected", e)
	}
	replacement, _ := sealSHVActivation(map[string]any{"protocol": "test", "value": "replacement"})
	if e = os.WriteFile(path, replacement, 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = dossierBundles(evidence, observations); e == nil {
		t.Fatal("resealed changed bundle admitted")
	}
	obj["value"] = jobMarshal("forged")
	if e = os.WriteFile(path, jobMarshal(obj), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = dossierBundles(evidence, observations); e == nil {
		t.Fatal("forged unchanged digest admitted")
	}
}
