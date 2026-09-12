package knowledgeengine

// ValidateSCVBundleText protects the original raw spelling at new bundle CLI
// boundaries before encoding/json could replace invalid UTF-16 escapes. It is
// syntax/Unicode validation only; semantic owner validation remains separate.
// Ordinary legacy readers and canonicalization conventions are unchanged.
func ValidateSCVBundleText(raw []byte) error {
	if err := validateJSONObject(raw, maxRequestBytes); err != nil {
		return err
	}
	return ValidateSCVBundleUnicode(raw)
}

// ValidateSCVBundleUnicode rejects raw unpaired Unicode escapes only. The caller
// must first apply its own explicit bounded JSON profile. This permits the exact
// retained bundle-record envelope to keep its established aggregate value limit
// while independently validating its original native input/result text.
func ValidateSCVBundleUnicode(raw []byte) error {
	return scvBundleUnicode(raw)
}
