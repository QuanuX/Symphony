package profile

import (
	"os"
	"strings"
	"testing"
)

func TestSSHProfileIndependentGolden(t *testing.T) {
	raw, err := os.ReadFile("../testdata/endpoint-v2.json")
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("../testdata/endpoint-v2.sha256")
	if err != nil {
		t.Fatal(err)
	}
	var endpoint Endpoint
	if Decode(raw, &endpoint) != nil || Validate(endpoint) != nil {
		t.Fatal("invalid shared vector")
	}
	if got := Digest(endpoint); got != strings.TrimSpace(string(want)) {
		t.Fatal("independent Python SHA-256 vector differs", got)
	}
}
