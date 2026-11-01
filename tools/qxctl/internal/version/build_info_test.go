package version

import (
	"encoding/json"
	"runtime"
	"strings"
	"testing"
)

func TestBuildInfoUnstamped(t *testing.T) {
	info, err := describe(sourceStamp{}, strings.Repeat("a", 64))
	if err != nil || info.SourceState != "unavailable" || info.Version != "0.1.0-dev" ||
		info.GoVersion != runtime.Version() || info.GOOS != runtime.GOOS || info.GOARCH != runtime.GOARCH {
		t.Fatalf("unexpected identity: %+v, %v", info, err)
	}
	raw, err := json.Marshal(info)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"source_commit", "source_tree", "source_date_epoch", "dependency_map_sha256", "build_recipe"} {
		value, exists := decoded[field]
		if !exists || value != nil {
			t.Fatalf("unstamped %s must be present and null", field)
		}
	}
}

func TestBuildInfoCompleteStampAndRefusals(t *testing.T) {
	valid := sourceStamp{strings.Repeat("a", 40), strings.Repeat("b", 40), "1793510400", strings.Repeat("c", 64), ReproducibleBuildRecipe}
	digest := strings.Repeat("d", 64)
	info, err := describe(valid, digest)
	if err != nil || info.SourceState != "stamped" || *info.SourceCommit != valid.commit ||
		*info.SourceTree != valid.tree || *info.SourceDateEpoch != valid.epoch ||
		*info.DependencyMapSHA256 != valid.dependencyMap || *info.BuildRecipe != valid.recipe || info.ExecutableSHA256 != digest {
		t.Fatalf("stamp was not preserved: %+v, %v", info, err)
	}
	for name, change := range map[string]func(*sourceStamp){
		"missing_commit":     func(s *sourceStamp) { s.commit = "" },
		"missing_tree":       func(s *sourceStamp) { s.tree = "" },
		"missing_epoch":      func(s *sourceStamp) { s.epoch = "" },
		"missing_map":        func(s *sourceStamp) { s.dependencyMap = "" },
		"missing_recipe":     func(s *sourceStamp) { s.recipe = "" },
		"uppercase_commit":   func(s *sourceStamp) { s.commit = strings.Repeat("A", 40) },
		"short_tree":         func(s *sourceStamp) { s.tree = "abcd" },
		"nonhex_map":         func(s *sourceStamp) { s.dependencyMap = strings.Repeat("g", 64) },
		"negative_epoch":     func(s *sourceStamp) { s.epoch = "-1" },
		"leading_zero_epoch": func(s *sourceStamp) { s.epoch = "01" },
		"plus_epoch":         func(s *sourceStamp) { s.epoch = "+1" },
		"overflow_epoch":     func(s *sourceStamp) { s.epoch = "9223372036854775808" },
		"unknown_recipe":     func(s *sourceStamp) { s.recipe = "other" },
	} {
		t.Run(name, func(t *testing.T) {
			bad := valid
			change(&bad)
			got, err := describe(bad, digest)
			if err == nil || got.SourceState != "invalid" {
				t.Fatalf("accepted malformed stamp: %+v, %v", got, err)
			}
		})
	}
	valid.epoch = "0"
	if _, err := describe(valid, digest); err != nil {
		t.Fatalf("zero epoch refused: %v", err)
	}
	for _, bad := range []string{"", "sha256:" + digest, strings.Repeat("D", 64), digest[:63]} {
		if _, err := describe(valid, bad); err == nil {
			t.Fatalf("accepted invalid executable digest %q", bad)
		}
	}
}
