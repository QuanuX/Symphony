package knowledgeengine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSNVCLIWrapperSchemas(t *testing.T) {
	raw, err := os.ReadFile("../../../../modules/snv-engine/schemas/v1/admin.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	document, err := ParseSNVObject(raw)
	if err != nil {
		t.Fatal(err)
	}
	schemas, okay := document["cli_results"].(map[string]any)
	if !okay || len(schemas) != 6 {
		t.Fatal("six discoverable CLI wrapper schemas required")
	}
	for protocol, shape := range schemas {
		object, okay := shape.(map[string]any)
		if !okay || object["$id"] != protocol || !strings.HasPrefix(protocol, "symphony.qxctl.snv-") {
			t.Fatal("wrapper schema identity mismatch", protocol)
		}
		if object["additionalProperties"] != false {
			t.Fatal("wrapper schema admits unknown top-level keys", protocol)
		}
	}
	evidence := os.Getenv("SYMPHONY_SNV_WRAPPER_TEST_EVIDENCE")
	if evidence == "" {
		t.Skip("set SYMPHONY_SNV_WRAPPER_TEST_EVIDENCE to preserved independent real CLI outputs")
	}
	entries, err := os.ReadDir(evidence)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]int{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(evidence, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		value, err := ParseSNVObject(raw)
		if err != nil {
			continue
		}
		protocol, _ := value["protocol"].(string)
		shape, found := schemas[protocol]
		if !found {
			continue
		}
		schema := shape.(map[string]any)
		if !snvTransportShape(schema, value, schema, 0) {
			t.Fatalf("preserved actual %s violates discoverable wrapper %s", entry.Name(), protocol)
		}
		seen[protocol]++
		copyValue := func() map[string]any {
			b, e := json.Marshal(value)
			if e != nil {
				t.Fatal(e)
			}
			v, e := ParseSNVObject(b)
			if e != nil {
				t.Fatal(e)
			}
			return v
		}
		broken := copyValue()
		delete(broken, "protocol")
		if snvTransportShape(schema, broken, schema, 0) {
			t.Fatal("missing protocol admitted", entry.Name())
		}
		broken = copyValue()
		broken["protocol"] = "private-marker"
		if snvTransportShape(schema, broken, schema, 0) {
			t.Fatal("wrong wrapper protocol admitted", entry.Name())
		}
		broken = copyValue()
		broken["unexpected"] = true
		if snvTransportShape(schema, broken, schema, 0) {
			t.Fatal("unknown wrapper key admitted", entry.Name())
		}
		if _, found := value["digest"]; found {
			broken = copyValue()
			broken["digest"] = "sha256:wrong"
			if snvTransportShape(schema, broken, schema, 0) {
				t.Fatal("malformed wrapper digest admitted", entry.Name())
			}
		}
		if _, found := value["owner_result"].(map[string]any); found {
			broken = copyValue()
			broken["owner_result"].(map[string]any)["unexpected"] = true
			if snvTransportShape(schema, broken, schema, 0) {
				t.Fatal("unknown native result key admitted", entry.Name())
			}
		}
		if _, found := value["installation"].(map[string]any); found {
			broken = copyValue()
			broken["installation"].(map[string]any)["ReceiptDigest"] = "bad"
			if snvTransportShape(schema, broken, schema, 0) {
				t.Fatal("malformed installation seal admitted", entry.Name())
			}
		}
	}
	for protocol := range schemas {
		if seen[protocol] == 0 {
			t.Fatal("missing independent actual wrapper fixture", protocol)
		}
	}
	t.Logf("validated preserved independent CLI positives by protocol: %v", seen)
}
