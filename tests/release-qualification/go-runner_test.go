package main

import "testing"

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
