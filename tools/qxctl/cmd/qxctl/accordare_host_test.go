package main

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSAVProjectionBindsCanonicalPayload(t *testing.T) {
	payload := map[string]any{"z": json.Number("2"), "a": map[string]any{"b": true}}
	projection := savProjection(
		"host:test", "knowledge", "knowledge/SPEC.md", "fixture.v1",
		"derived_evidence", payload,
		"sha256:1111111111111111111111111111111111111111111111111111111111111111",
		time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC),
	)
	if projection.ContentDigest != digestCanonicalJSON(payload) {
		t.Fatal("source projection content digest does not bind canonical JSON")
	}
	if projection.ObservedAt != "2026-08-20T12:00:00Z" || projection.Freshness != "current" {
		t.Fatal("source projection did not use STSC whole-second current evidence")
	}
}

func TestSAVCurrentExposesHostAssemblyFlags(t *testing.T) {
	command := newSAVCommand()
	current, _, err := command.Find([]string{"current"})
	if err != nil {
		t.Fatalf("find sav current: %v", err)
	}
	for _, name := range []string{
		"host", "tops-id", "state-root", "maestro-prefix", "maestro-version", "scope", "ttl",
	} {
		if current.Flags().Lookup(name) == nil {
			t.Fatalf("sav current lacks --%s", name)
		}
	}
}
