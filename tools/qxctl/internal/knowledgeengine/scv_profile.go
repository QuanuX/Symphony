package knowledgeengine

import "fmt"

// Preparation retains exactly authored fields. The owner validates rules;
// the consumer independently rejects a result that substitutes any input.
func validateSCVProfilePreparation(input, result map[string]any) error {
	if !scvCorpusFields(input, "profile") {
		return fmt.Errorf("profile preparation input must contain only profile")
	}
	draft, ok := input["profile"].(map[string]any)
	keys := []string{"protocol", "profile_id", "profile_version", "provider_id", "source_id", "locator_id", "media_types", "authored_by", "rationale", "rules"}
	if !ok || !scvCorpusFields(draft, keys...) || !scvCorpusFields(result, append(keys, "digest")...) ||
		draft["protocol"] != "symphony.scv.interpretation-profile.v1" {
		return fmt.Errorf("invalid profile draft or prepared profile shape")
	}
	for _, key := range keys {
		if !scvEqual(draft[key], result[key]) {
			return fmt.Errorf("profile preparation changed authored %s", key)
		}
	}
	for _, key := range []string{"profile_id", "profile_version", "provider_id", "source_id", "locator_id", "authored_by", "rationale"} {
		value, ok := result[key].(string)
		if !ok || value == "" {
			return fmt.Errorf("invalid prepared profile identity %s", key)
		}
	}
	rules, ok := result["rules"].([]any)
	if !ok || len(rules) > 128 {
		return fmt.Errorf("invalid prepared profile rules")
	}
	return scvSeal(result, "digest")
}
