package stavprotocol

import "testing"

func TestGenerateUUIDv4(t *testing.T) {
	id, err := GenerateUUIDv4()
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateEventUUID(id); err != nil {
		t.Fatalf("generated %q: %v", id, err)
	}
}

func TestRequestUUIDVersions(t *testing.T) {
	for _, id := range []string{
		"85cece8e-1201-4c64-95e4-1ce9020a4b26",
		"018f0f7a-6a5d-7cc4-8a76-2c6a879f9f10",
	} {
		if err := ValidateRequestUUID(id); err != nil {
			t.Fatalf("valid ID %q: %v", id, err)
		}
	}
	for _, id := range []string{
		"85CECE8E-1201-4C64-95E4-1CE9020A4B26",
		"00000000-0000-0000-0000-000000000000",
	} {
		if err := ValidateRequestUUID(id); err == nil {
			t.Fatalf("expected invalid ID %q", id)
		}
	}
}
