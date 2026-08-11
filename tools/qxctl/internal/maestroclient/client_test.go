package maestroclient

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestResourceAndComponentEvidenceAreDeterministic(t *testing.T) {
	receipt := "sha256:" + strings.Repeat("a", 64)
	executable := "sha256:" + strings.Repeat("b", 64)
	resource := Resource(
		"123e4567-e89b-42d3-a456-426614174000", "receptor-a", "dock",
		"ssfv-engine", receipt, "absent",
	)
	if resource != "symphony.maestro.docking:acf1d7a7665091e1a107ee93d6c7c13440a3932c8138d136b02bc2cc5b672813" {
		t.Fatalf("Maestro authorization resource drifted: %s", resource)
	}
	evidence, err := NewComponentEvidence(
		"ssfv-engine", "ssfv-engine", "ssfv", "symphony-ssfv", receipt, executable,
	)
	if err != nil {
		t.Fatal(err)
	}
	if evidence.EvidenceDigest != "sha256:5f569d8f6a9a24cffb392f04cf7853bc921aeebd83169da97cf7ffdc209958b6" ||
		evidence.ComponentKind != "vector_engine" || evidence.ReceptorKind != ReceptorKind {
		t.Fatalf("Maestro component evidence drifted: %+v", evidence)
	}
}

func TestDecodedRegistryRejectsIdentityAndDigestDrift(t *testing.T) {
	digest := "sha256:" + strings.Repeat("c", 64)
	registry := Registry{
		Protocol: "symphony.maestro.docking-presence-registry.v1", FormatVersion: 1,
		TOPSID: "123e4567-e89b-42d3-a456-426614174000", ReceptorID: "receptor-a",
		ReceptorKind: ReceptorKind, Generation: 1, Components: []Presence{}, Extensions: []any{},
		Recovery:  Recovery{State: "clean", Disposition: "not_applicable", Detail: "clean"},
		UpdatedAt: "2026-08-11T00:00:00Z", RegistryDigest: digest,
	}
	raw, err := json.Marshal(registry)
	if err != nil {
		t.Fatal(err)
	}
	result := Result{
		TOPSID: registry.TOPSID, ReceptorID: registry.ReceptorID, RegistryPresent: true,
		Registry: raw, RegistryDigest: &digest,
	}
	decoded, err := result.DecodedRegistry()
	if err != nil || decoded == nil || decoded.RegistryDigest != digest {
		t.Fatalf("valid Maestro registry was rejected: %+v err=%v", decoded, err)
	}
	result.RegistryDigest = nil
	if _, err := result.DecodedRegistry(); err == nil {
		t.Fatal("Maestro registry without its result digest was accepted")
	}
	result.RegistryPresent = false
	result.Registry = json.RawMessage("null")
	result.RegistryDigest = nil
	if decoded, err := result.DecodedRegistry(); err != nil || decoded != nil {
		t.Fatalf("absent registry did not remain absent: %+v err=%v", decoded, err)
	}
}
