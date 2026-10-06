package knowledgeengine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSBVInstalledBook(t *testing.T) {
	prefix, fixture := os.Getenv("SYMPHONY_SBV_TEST_PREFIX"), os.Getenv("SYMPHONY_SBV_BOOK_FIXTURE")
	if prefix == "" || fixture == "" {
		t.Skip("set exact installation and native-emitted book fixture")
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
	request["output_path"] = filepath.Join(dir, "book.json")
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	invoke := func(op string, p map[string]any) (Response, error) {
		b, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		return InvokeSBV(context.Background(), prefix, "0.13.0-dev", cwd, op, b)
	}
	response, err := invoke("book", request)
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
	schema, err := SBVSchema(prefix, "0.13.0-dev", "book")
	if err != nil {
		t.Fatal(err)
	}
	contract, err := sqavObject(schema, maxRequestBytes)
	if err != nil {
		t.Fatal(err)
	}
	defs := contract["$defs"].(map[string]any)
	sections := artifact["sections"].(map[string]any)
	rows := sections["book_frames"].(map[string]any)["data"].([]any)
	for _, row := range rows {
		if !sqvTransportShape(defs["book_frame"], row, 0) {
			t.Fatal("native frame fails installed schema")
		}
	}
	checkpoint := sections["book_checkpoint"].(map[string]any)["data"]
	if !sqvTransportShape(defs["book_checkpoint"], checkpoint, 0) {
		t.Fatal("native checkpoint fails installed schema")
	}
	query := map[string]any{"protocol": "symphony.sbv.result-query-input.v1", "path": request["output_path"], "expected_sha256": receipt["content_sha256"], "pointer": "/sections/book_frames/data/6/data/bids/0/size", "limit": "1", "cursor": ""}
	response, err = invoke("result_query", query)
	if err != nil {
		t.Fatal(err)
	}
	page, err := sqavObject(response.Result, maxResponseBytes)
	if err != nil {
		t.Fatal(err)
	}
	if page["nodes"].([]any)[0].(map[string]any)["value"] != "7" {
		t.Fatal("partial cancel quantity inaccessible")
	}
	if _, err = invoke("book", request); err == nil {
		t.Fatal("replaced existing result")
	}
	request["output_path"] = filepath.Join(dir, "bad.json")
	request["census_result"].(map[string]any)["expected_sha256"] = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if _, err = invoke("book", request); err == nil {
		t.Fatal("wrong census admitted")
	}
	if _, err = os.Stat(request["output_path"].(string)); !os.IsNotExist(err) {
		t.Fatal("invalid output exists")
	}
	request["unknown"] = true
	if _, err = invoke("book", request); err == nil {
		t.Fatal("unknown field admitted")
	}
}
