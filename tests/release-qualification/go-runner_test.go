package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInstalledSuitesRetainIndependentEvidence(t *testing.T) {
	repo, work, prefix := t.TempDir(), t.TempDir(), t.TempDir()
	for name, contents := range map[string]string{
		"fixture/go.mod": "module example/qualification\n\ngo 1.26.0\n",
		"fixture/first/fixture_test.go": `package first
import ("os"; "testing")
func TestFirst(t *testing.T) {
    if os.Getenv("QUALIFICATION_FIXTURE_PREFIX") == "" { t.Fatal("missing installed prefix") }
}
func TestSecond(t *testing.T) {}
`,
		"fixture/second/fixture_test.go": `package second
import "testing"
func TestThird(t *testing.T) {}
`,
	} {
		path := filepath.Join(repo, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
	}
	selections := []suite{
		{Group: "quantitative", Module: "fixture", Package: "./first", Tests: []string{"TestFirst"}, PrefixEnvironment: []string{"QUALIFICATION_FIXTURE_PREFIX"}},
		{Group: "quantitative", Module: "fixture", Package: "./second", Tests: []string{"TestThird"}},
		{Group: "quantitative", Module: "fixture", Package: "./first", Tests: []string{"TestSecond"}},
	}
	for _, selected := range selections {
		if err := run(repo, work, "installed", prefix, module{Path: selected.Module, Packages: []string{selected.Package}}, []suite{selected}, nil); err != nil {
			t.Fatalf("suite %s %v: %v", selected.Package, selected.Tests, err)
		}
	}
	entries, err := os.ReadDir(work)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3*len(selections) {
		t.Fatalf("expected independent event, stderr and summary files for every suite; found %d", len(entries))
	}
	retained := map[string][]byte{}
	passed := map[string]bool{}
	for _, entry := range entries {
		raw, err := os.ReadFile(filepath.Join(work, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		retained[entry.Name()] = raw
		if !strings.HasSuffix(entry.Name(), "-summary.json") {
			continue
		}
		var s summary
		if err := json.Unmarshal(raw, &s); err != nil {
			t.Fatal(err)
		}
		if s.Passed != 1 || s.Skipped != 0 || s.Error != "" || len(s.Required) != 1 || len(s.Results) != 1 {
			t.Fatalf("incorrect suite evidence: %+v", s)
		}
		key := s.Results[0].Package + "/" + s.Results[0].Test
		if key != s.Required[0] || passed[key] {
			t.Fatalf("missing or repeated suite evidence: %s", key)
		}
		passed[key] = true
	}
	for _, selected := range selections {
		key := packageName("example/qualification", selected.Package) + "/" + selected.Tests[0]
		if !passed[key] {
			t.Fatalf("suite evidence missing: %s", key)
		}
	}
	// Reusing the same selection must still refuse to overwrite retained evidence.
	selected := selections[0]
	if err := run(repo, work, "installed", prefix, module{Path: selected.Module, Packages: []string{selected.Package}}, []suite{selected}, nil); !os.IsExist(err) {
		t.Fatalf("repeated selection must fail with existing evidence: %v", err)
	}
	for name, before := range retained {
		after, err := os.ReadFile(filepath.Join(work, name))
		if err != nil || !bytes.Equal(before, after) {
			t.Fatalf("retained evidence changed: %s (%v)", name, err)
		}
	}
}

func TestSelectedTestMustActuallyPass(t *testing.T) {
	required := []suite{{Module: "tools/qxctl", Package: "./internal/example", Tests: []string{"TestRequired"}}}
	if _, err := evaluate("tools/qxctl", "example/qxctl", "installed", []event{{Action: "pass", Package: "example/qxctl/internal/example", Test: "TestDifferent"}}, nil, required); err == nil {
		t.Fatal("missing selected test accepted")
	}
}
func TestOptionalSkipsRemainDistinctFromCoverage(t *testing.T) {
	events := []event{{Action: "pass", Package: "example/qxctl", Test: "TestReal/child"}, {Action: "pass", Package: "example/qxctl", Test: "TestReal"}, {Action: "skip", Package: "example/qxctl", Test: "TestHistorical"}}
	optional := []optionalTest{{Module: "tools/qxctl", Package: ".", Test: "TestHistorical", Classification: "retained_fixture"}}
	s, err := evaluate("tools/qxctl", "example/qxctl", "full", events, optional, nil)
	if err != nil || s.Passed != 1 || s.Subtests != 1 || s.Skipped != 1 {
		t.Fatal(s, err)
	}
	if _, err = evaluate("tools/qxctl", "example/qxctl", "installed", events, optional, nil); err == nil {
		t.Fatal("selected installed skip accepted")
	}
	if _, err = evaluate("tools/qxctl", "example/qxctl", "full", events, nil, nil); err == nil {
		t.Fatal("unclassified skip accepted")
	}
}

func TestInstalledQualificationIsRequired(t *testing.T) {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test source unavailable")
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(source), "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m manifest
	if err = json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	for _, selected := range []struct{ pkg, name, group, prefix string }{
		{"./internal/knowledgeengine", "TestSRRAPIVInstalledNative", "quantitative", "SRRAPIV_TEST_PREFIX"},
		{"./internal/knowledgeengine", "TestSRRAPIVInstalledResidentLifecycle", "quantitative", "SRRAPIV_TEST_PREFIX"},
		{"./cmd/qxctl", "TestSRRAPIVInstalledCLI", "quantitative", "SRRAPIV_TEST_PREFIX"},
		{"./internal/sovcloudflare", "TestCloudflareInstalledReceiptProcess", "cloud", "SYMPHONY_CLOUDFLARE_TEST_PREFIX"},
	} {
		pkg := packageName("example/qxctl", selected.pkg)
		events := []event{{Action: "pass", Package: pkg, Test: "TestUnit"}, {Action: "skip", Package: pkg, Test: selected.name}}
		if _, err := evaluate("tools/qxctl", "example/qxctl", "full", events, m.Optional, nil); err != nil {
			t.Fatalf("source-only skip not classified for %s: %v", selected.name, err)
		}
		var required []suite
		for _, s := range m.Installed {
			if s.Group != selected.group || s.Module != "tools/qxctl" || s.Package != selected.pkg {
				continue
			}
			for _, name := range s.Tests {
				if name == selected.name {
					foundPrefix := false
					for _, variable := range s.PrefixEnvironment {
						foundPrefix = foundPrefix || variable == selected.prefix
					}
					if !foundPrefix {
						t.Fatalf("%s lacks its exact installed prefix binding", selected.name)
					}
					required = append(required, suite{Module: s.Module, Package: s.Package, Tests: []string{name}})
				}
			}
		}
		if len(required) != 1 {
			t.Fatalf("%s must have exactly one installed selection", selected.name)
		}
		if _, err := evaluate("tools/qxctl", "example/qxctl", "installed", events[:1], m.Optional, required); err == nil {
			t.Fatal("missing installed test accepted")
		}
		if _, err := evaluate("tools/qxctl", "example/qxctl", "installed", events, m.Optional, required); err == nil {
			t.Fatal("installed skip accepted")
		}
		events[1].Action = "pass"
		if _, err := evaluate("tools/qxctl", "example/qxctl", "installed", events, m.Optional, required); err != nil {
			t.Fatal(err)
		}
	}
}
