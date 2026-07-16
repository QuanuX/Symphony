package provider

import (
	"testing"

	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/config"
)

func TestDeclaredProviderIsNotReady(t *testing.T) {
	registry, err := New([]config.ProviderConfig{{
		Name:         "native",
		Kind:         "native-keyring",
		Enabled:      true,
		Capabilities: []string{"store", "retrieve"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	got := registry.Descriptors()
	if len(got) != 1 || got[0].Status != "declared" {
		t.Fatalf("unexpected descriptors: %+v", got)
	}
}
