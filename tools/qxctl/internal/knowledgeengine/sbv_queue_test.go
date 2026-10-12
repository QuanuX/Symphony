package knowledgeengine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSBVInstalledQueueScenarios(t *testing.T) {
	prefix, fixture := os.Getenv("SYMPHONY_SBV_TEST_PREFIX"), os.Getenv("SYMPHONY_SBV_QUEUE_FIXTURE")
	if prefix == "" || fixture == "" {
		t.Skip("select exact installation and native queue fixture")
	}
	raw, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	var request map[string]any
	if err = json.Unmarshal(raw, &request); err != nil {
		t.Fatal(err)
	}
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	request["output_path"] = filepath.Join(dir, "queue.json")
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	invoke := func(op string, p map[string]any) (Response, error) {
		b, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		return InvokeSBV(context.Background(), prefix, SBVAdministrationInterfaceVersion, cwd, op, b)
	}
	response, err := invoke("queue_scenarios", request)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := sqavObject(response.Result, maxResponseBytes)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(request["output_path"].(string))
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := sqavObject(data, 128<<20)
	if err != nil {
		t.Fatal(err)
	}
	if receipt["content_sha256"] != artifact["content_sha256"] {
		t.Fatal("receipt identity mismatch")
	}
	schema, err := SBVSchema(prefix, SBVAdministrationInterfaceVersion, "queue_scenarios")
	if err != nil {
		t.Fatal(err)
	}
	contract, err := sqavObject(schema, maxRequestBytes)
	if err != nil {
		t.Fatal(err)
	}
	defs := contract["$defs"].(map[string]any)
	sections := artifact["sections"].(map[string]any)
	for _, pair := range [][2]string{{"queue_scenarios", "queue_scenario"}, {"execution", "queue_summary"}} {
		for _, row := range sections[pair[0]].(map[string]any)["data"].([]any) {
			if !sqvTransportShape(defs[pair[1]], row, 0) {
				t.Fatalf("native %s fails typed schema", pair[1])
			}
		}
	}
	request["output_path"] = filepath.Join(dir, "invalid.json")
	request["model"].(map[string]any)["priority"] = "unknown"
	if _, err = invoke("queue_scenarios", request); err == nil {
		t.Fatal("unknown queue priority admitted")
	}
	if _, err = os.Stat(request["output_path"].(string)); !os.IsNotExist(err) {
		t.Fatal("invalid request created an artifact")
	}
}
