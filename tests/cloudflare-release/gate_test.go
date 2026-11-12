package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalEvidenceCannotCloseLiveGate(t *testing.T) {
	dir := t.TempDir()
	raw := []byte(`{"Action":"start","Package":"p"}
{"Action":"run","Package":"p","Test":"TestOne"}
{"Action":"pass","Package":"p","Test":"TestOne"}
{"Action":"pass","Package":"p"}`)
	os.WriteFile(filepath.Join(dir, "go.jsonl"), raw, 0600)
	h := sha256.Sum256(raw)
	m := manifest{Version: 1, Suites: []suite{{ID: "go", Format: "go-json", Required: []string{"TestOne"}}}, Open: []string{"LIVE-01"}}
	in := selection{Revision: strings.Repeat("a", 40), Platform: "darwin-amd64", Artifacts: []artifact{{Suite: "go", Path: "go.jsonl", SHA256: hex.EncodeToString(h[:])}}}
	r, e := evaluate(m, in, dir)
	if e != nil || r.Local != "passed" || r.Milestone != "open" || r.Live != "not_evaluated" {
		t.Fatal(r, e)
	}
	in.Artifacts[0].SHA256 = strings.Repeat("0", 64)
	if _, e = evaluate(m, in, dir); e == nil {
		t.Fatal("tampered evidence accepted")
	}
	in.Artifacts = nil
	r, e = evaluate(m, in, dir)
	if e != nil || r.Local != "incomplete" {
		t.Fatal(r, e)
	}
}
func TestMissingSkippedFailedAndTruncatedCoverage(t *testing.T) {
	s := suite{ID: "native", Format: "ctest-junit", Required: []string{"one", "two"}}
	for _, tc := range []struct{ body, want string }{{`<testsuite><testcase name="one" status="run"/></testsuite>`, "incomplete"}, {`<testsuite><testcase name="one" status="run"/><testcase name="two" status="notrun"><skipped/></testcase></testsuite>`, "incomplete"}, {`<testsuite><testcase name="one" status="run"><failure/></testcase><testcase name="two" status="run"/></testsuite>`, "failed"}} {
		r, e := parseSuite(s, []byte(tc.body))
		if e != nil || r.Status != tc.want {
			t.Fatal(r, e)
		}
	}
	for _, raw := range []string{``, `{"Action":"start","Package":"p"}`, `{"Action":"pass","Package":"p"}`, `{"Action":"start","Package":"p"}` + "\n" + `garbage`} {
		if _, e := parseSuite(suite{Format: "go-json"}, []byte(raw)); e == nil {
			t.Fatal("invalid Go stream accepted", raw)
		}
	}
}
func TestEvidenceEscapeAndSymlinkRefused(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "real"), []byte("x"), 0600)
	os.Symlink(filepath.Join(dir, "real"), filepath.Join(dir, "link"))
	for _, p := range []string{"../outside", filepath.Join(dir, "real"), "link"} {
		if _, e := evidence(dir, p); e == nil {
			t.Fatal("unsafe path accepted", p)
		}
	}
}
func TestDuplicateAndUnknownSuiteRefused(t *testing.T) {
	m := manifest{Version: 1, Suites: []suite{{ID: "one", Format: "go-json", Required: []string{"TestOne"}}}, Open: []string{"LIVE-01"}}
	in := selection{Revision: strings.Repeat("a", 40), Platform: "linux-amd64", Artifacts: []artifact{{Suite: "unknown"}}}
	if _, e := evaluate(m, in, t.TempDir()); e == nil {
		t.Fatal("unknown suite accepted")
	}
	in.Artifacts = []artifact{{Suite: "one"}, {Suite: "one"}}
	if _, e := evaluate(m, in, t.TempDir()); e == nil {
		t.Fatal("duplicate suite accepted")
	}
}

func TestInventoryAndSupportMatrixMatchOwner(t *testing.T) {
	var m manifest
	raw, err := os.ReadFile("manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = strict(raw, &m); err != nil || len(m.Suites) != 7 {
		t.Fatal(m, err)
	}
	raw, err = os.ReadFile("../../knowledge/sov/cloudflare/owner.json")
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(raw)
	var owner struct {
		Profiles []struct {
			Operation string `json:"operation"`
		} `json:"resource_profiles"`
	}
	if err = json.Unmarshal(raw, &owner); err != nil {
		t.Fatal(err)
	}
	raw, err = os.ReadFile("support-matrix.json")
	if err != nil {
		t.Fatal(err)
	}
	var matrix struct {
		Owner    string `json:"owner_sha256"`
		Profiles []struct {
			Operation string `json:"operation"`
			Live      string `json:"live"`
		} `json:"resource_operations"`
		Mandatory []struct {
			ID string `json:"id"`
		} `json:"mandatory_requirements"`
	}
	if err = json.Unmarshal(raw, &matrix); err != nil {
		t.Fatal(err)
	}
	if matrix.Owner != hex.EncodeToString(digest[:]) || len(matrix.Profiles) != len(owner.Profiles) || len(matrix.Mandatory) != 7 {
		t.Fatal("support matrix drift")
	}
	seen := map[string]bool{}
	for _, p := range matrix.Profiles {
		if seen[p.Operation] || p.Live != "unqualified" {
			t.Fatal("invalid qualification", p)
		}
		seen[p.Operation] = true
	}
	for _, p := range owner.Profiles {
		if !seen[p.Operation] {
			t.Fatal("omitted operation", p.Operation)
		}
	}
	for i, p := range matrix.Mandatory {
		if p.ID != fmt.Sprintf("M1-%02d", i+1) {
			t.Fatal("mandatory requirement omitted")
		}
	}
}
