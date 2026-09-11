package knowledgeengine

import (
	"context"
	"os"
	"strings"
	"testing"
)

func profileDraft() map[string]any {
	return map[string]any{"protocol": "symphony.scv.interpretation-profile.v1", "profile_id": "fixture-profile", "profile_version": "1", "provider_id": "cf", "source_id": "docs", "locator_id": "main", "media_types": []any{"text/markdown"}, "authored_by": "test fixture", "rationale": "Empty fixture mapping asserts no facts", "rules": []any{}}
}
func TestSCVProfilePreparationConsumerRejectsSubstitution(t *testing.T) {
	draft := profileDraft()
	input := map[string]any{"profile": draft}
	result := scvTestSeal(t, scvTestClone(t, draft))
	if err := validateSCVProfilePreparation(input, result); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"profile_id", "rationale", "source_id", "provider_id", "profile_version"} {
		changed := scvTestClone(t, result)
		changed[key] = "substituted"
		changed = scvTestSeal(t, changed)
		if err := validateSCVProfilePreparation(input, changed); err == nil {
			t.Fatal("accepted resealed substitution", key)
		}
	}
	if SCVOperationSupported("0.3.0-dev", "profile_prepare") || !SCVOperationSupported("0.4.0-dev", "profile_prepare") {
		t.Fatal("profile preparation version admission changed")
	}
	if !SCVOperationSupported("0.4.0-dev", "connection_reassess") {
		t.Fatal("prior operation missing from .4")
	}
}
func TestInstalledSCVProfilePreparation(t *testing.T) {
	prefix := os.Getenv("SYMPHONY_SCV_AGENT_PREFIX")
	if prefix == "" {
		t.Skip("requires exact .4 packages")
	}
	for _, domain := range SCVDomains() {
		t.Run(domain, func(t *testing.T) {
			draft := profileDraft()
			if strings.HasPrefix(domain, "schv-") {
				draft["provider_id"] = strings.TrimPrefix(domain, "schv-")
			}
			input, _ := SCVCanonical(map[string]any{"profile": draft})
			result, err := InvokeSCVDomain(context.Background(), domain, prefix, "0.4.0-dev", t.TempDir(), "profile_prepare", input)
			if err != nil {
				t.Fatal(err)
			}
			if err = ValidateSCVResult("profile_prepare", input, result.Result); err != nil {
				t.Fatal(err)
			}
		})
	}
}
