package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

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

func TestSRRAPIVInstalledQualificationIsRequired(t *testing.T) {
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
	for _, selected := range []struct{ pkg, name string }{
		{"./internal/knowledgeengine", "TestSRRAPIVInstalledNative"},
		{"./internal/knowledgeengine", "TestSRRAPIVInstalledResidentLifecycle"},
		{"./cmd/qxctl", "TestSRRAPIVInstalledCLI"},
	} {
		pkg := packageName("example/qxctl", selected.pkg)
		events := []event{{Action: "pass", Package: pkg, Test: "TestUnit"}, {Action: "skip", Package: pkg, Test: selected.name}}
		if _, err := evaluate("tools/qxctl", "example/qxctl", "full", events, m.Optional, nil); err != nil {
			t.Fatalf("source-only skip not classified for %s: %v", selected.name, err)
		}
		var required []suite
		for _, s := range m.Installed {
			if s.Group != "quantitative" || s.Module != "tools/qxctl" || s.Package != selected.pkg {
				continue
			}
			for _, name := range s.Tests {
				if name == selected.name {
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
