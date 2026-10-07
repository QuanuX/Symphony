package knowledgeengine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSBVInstalledEconomics(t *testing.T) {
	prefix, fixture := os.Getenv("SYMPHONY_SBV_TEST_PREFIX"), os.Getenv("SYMPHONY_SBV_ECONOMICS_FIXTURE")
	if prefix == "" || fixture == "" {
		t.Skip("set exact installation and native-emitted economics fixture")
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
	request["output_path"] = filepath.Join(dir, "economics.json")
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	invoke := func(op string, p map[string]any) (Response, error) {
		b, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		return InvokeSBV(context.Background(), prefix, "0.21.0-dev", cwd, op, b)
	}
	response, err := invoke("economics", request)
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
	schema, err := SBVSchema(prefix, "0.21.0-dev", "economics")
	if err != nil {
		t.Fatal(err)
	}
	contract, err := sqavObject(schema, maxRequestBytes)
	if err != nil {
		t.Fatal(err)
	}
	defs := contract["$defs"].(map[string]any)
	sections := artifact["sections"].(map[string]any)
	for _, pair := range [][2]string{{"economics", "economic_outcome"}, {"studies", "economic_study"}} {
		rows := sections[pair[0]].(map[string]any)["data"].([]any)
		for _, row := range rows {
			if !sqvTransportShape(defs[pair[1]], row, 0) {
				t.Fatalf("installed %s schema rejects native output", pair[1])
			}
		}
	}
	query := map[string]any{"protocol": "symphony.sbv.result-query-input.v1", "path": request["output_path"], "expected_sha256": receipt["content_sha256"], "pointer": "/sections/economics/data/0/atoms/2/weight/numerator", "limit": "1", "cursor": ""}
	response, err = invoke("result_query", query)
	if err != nil {
		t.Fatal(err)
	}
	page, err := sqavObject(response.Result, maxResponseBytes)
	if err != nil {
		t.Fatal(err)
	}
	if page["nodes"].([]any)[0].(map[string]any)["value"] != "2" {
		t.Fatal("nonexecution mass not queryable")
	}
	if _, err = invoke("economics", request); err == nil {
		t.Fatal("existing result overwritten")
	}
	request["output_path"] = filepath.Join(dir, "bad.json")
	request["expected_sha256"] = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if _, err = invoke("economics", request); err == nil {
		t.Fatal("changed source digest admitted")
	}
	if _, err = os.Stat(request["output_path"].(string)); !os.IsNotExist(err) {
		t.Fatal("invalid request published output")
	}
	response, err = invoke("catalogue", map[string]any{"protocol": "symphony.sbv.catalogue-input.v1"})
	if err != nil {
		t.Fatal(err)
	}
	cat, err := sqavObject(response.Result, maxResponseBytes)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range cat["studies"].([]any) {
		card := v.(map[string]any)
		if descriptor, ok := card["descriptor"]; ok && !sqvTransportShape(defs["study_descriptor"], descriptor, 0) {
			t.Fatal("study descriptor mismatch")
		}
	}
	if len(cat["transforms"].([]any)) != 2 || len(cat["studies"].([]any)) != 21 {
		t.Fatal("discovery incomplete")
	}
}
