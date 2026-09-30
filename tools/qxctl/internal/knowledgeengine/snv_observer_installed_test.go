package knowledgeengine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSNVObserverInstalledConsumer(t *testing.T) {
	prefix := os.Getenv("SNV_OBSERVER_TEST_PREFIX")
	if prefix == "" {
		t.Skip("SNV_OBSERVER_TEST_PREFIX selects the independently installed optional collector")
	}
	if _, err := InspectSNV(prefix, "latest", "local-observer"); err == nil {
		t.Fatal("Floating collector release accepted")
	}
	for _, template := range []bool{false, true} {
		_, raw, err := SNVResource(prefix, "0.1.0-dev", "local-observer", "observe", template)
		if err != nil || !json.Valid(raw) {
			t.Fatal("Exact collector resource refused", err)
		}
	}
	input := snvReadObject(t, filepath.Join(snvRepository(t), "modules/snv-local-observer/tests/fixtures/observe.json"))
	response, err := InvokeSNV(context.Background(), prefix, "0.1.0-dev", snvRepository(t), "local-observer", "observe", snvEncode(t, input))
	if err != nil {
		t.Fatal("Installed finite collector refused", err)
	}
	result, err := ParseSNVObject(response.Result)
	if err != nil || result["acquisition_route"] != "native_fixed_sources" ||
		result["physical_inventory_complete"] != false || result["allocation_verified"] != false {
		t.Fatal("Collector result exceeded its acquisition or coverage contract", err)
	}
}
