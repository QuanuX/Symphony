package knowledgeengine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSBVInstalledAllocationEconomics(t *testing.T) {
	prefix, fixture := os.Getenv("SYMPHONY_SBV_TEST_PREFIX"), os.Getenv("SYMPHONY_SBV_ALLOCATION_FIXTURE")
	if prefix == "" || fixture == "" {
		t.Skip("set exact installation and native-emitted allocation economics fixture")
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
	request["output_path"] = filepath.Join(dir, "allocation.json")
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	invoke := func(op string, p map[string]any) (Response, error) {
		b, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		return InvokeSBV(context.Background(), prefix, "0.23.0-dev", cwd, op, b)
	}
	response, err := invoke("allocation_economics", request)
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
	schema, err := SBVSchema(prefix, "0.23.0-dev", "allocation_economics")
	if err != nil {
		t.Fatal(err)
	}
	contract, err := sqavObject(schema, maxRequestBytes)
	if err != nil {
		t.Fatal(err)
	}
	defs := contract["$defs"].(map[string]any)
	sections := artifact["sections"].(map[string]any)
	for _, pair := range [][2]string{{"economics", "allocation_economic_outcome"}, {"studies", "allocation_economic_study"}} {
		for _, row := range sections[pair[0]].(map[string]any)["data"].([]any) {
			if !sqvTransportShape(defs[pair[1]], row, 0) {
				t.Fatalf("native %s fails schema", pair[1])
			}
		}
	}
	query := map[string]any{"protocol": "symphony.sbv.result-query-input.v1", "path": request["output_path"], "expected_sha256": receipt["content_sha256"], "pointer": "/sections/economics/data/0/conditional/value/net_markout/numerator", "limit": "1", "cursor": ""}
	response, err = invoke("result_query", query)
	if err != nil {
		t.Fatal(err)
	}
	page, err := sqavObject(response.Result, maxResponseBytes)
	if err != nil {
		t.Fatal(err)
	}
	if page["nodes"].([]any)[0].(map[string]any)["value"] != "129" {
		t.Fatal("conditional net markout inaccessible")
	}
	if _, err = invoke("allocation_economics", request); err == nil {
		t.Fatal("replaced existing result")
	}
	request["output_path"] = filepath.Join(dir, "bad.json")
	request["expected_sha256"] = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if _, err = invoke("allocation_economics", request); err == nil {
		t.Fatal("mismatched allocation accepted")
	}
	if _, err = os.Stat(request["output_path"].(string)); !os.IsNotExist(err) {
		t.Fatal("bad result exists")
	}
	response, err = invoke("catalogue", map[string]any{"protocol": "symphony.sbv.catalogue-input.v1"})
	if err != nil {
		t.Fatal(err)
	}
	catalogue, err := sqavObject(response.Result, maxResponseBytes)
	if err != nil {
		t.Fatal(err)
	}
	catalogueSchema, err := SBVSchema(prefix, "0.23.0-dev", "catalogue")
	if err != nil {
		t.Fatal(err)
	}
	cs, err := sqavObject(catalogueSchema, maxRequestBytes)
	if err != nil {
		t.Fatal(err)
	}
	if !sqvTransportShape(cs["output"], catalogue, 0) {
		t.Fatal("catalogue schema mismatch")
	}
	if len(catalogue["models"].([]any)) != 7 || len(catalogue["studies"].([]any)) != 21 {
		t.Fatal("catalogue incomplete")
	}
}
