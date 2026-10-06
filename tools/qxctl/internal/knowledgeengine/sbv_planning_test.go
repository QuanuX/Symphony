package knowledgeengine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSBVInstalledPlanning(t *testing.T) {
	prefix, fixtures := os.Getenv("SYMPHONY_SBV_TEST_PREFIX"), os.Getenv("SYMPHONY_SBV_PLANNING_FIXTURES")
	if prefix == "" || fixtures == "" {
		t.Skip("set exact prefix and planning fixture directory")
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]string{{"backend_plan", "backend-request.json"}, {"live_plan", "live-request.json"}, {"result_select", "selection-request.json"}} {
		raw, err := os.ReadFile(filepath.Join(fixtures, pair[1]))
		if err != nil {
			t.Fatal(err)
		}
		response, err := InvokeSBV(context.Background(), prefix, "0.16.0-dev", cwd, pair[0], raw)
		if err != nil {
			t.Fatalf("%s: %v", pair[0], err)
		}
		result, err := sqavObject(response.Result, maxResponseBytes)
		if err != nil {
			t.Fatal(err)
		}
		schema, err := SBVSchema(prefix, "0.16.0-dev", pair[0])
		if err != nil {
			t.Fatal(err)
		}
		contract, err := sqavObject(schema, maxRequestBytes)
		if err != nil {
			t.Fatal(err)
		}
		if !sqvTransportShape(contract["output"], result, 0) {
			t.Fatalf("%s native result violates installed schema", pair[0])
		}
		switch pair[0] {
		case "backend_plan":
			if result["status"] != "unavailable" || result["applied"] != nil {
				t.Fatal("invented GPU execution")
			}
		case "live_plan":
			if result["can_activate"] != false || result["provider_requests"] != "0" {
				t.Fatal("live activation permitted")
			}
		case "result_select":
			rows := result["rows"].([]any)
			if rows[0].(map[string]any)["source_index"] != "0" || rows[1].(map[string]any)["source_index"] != "2" {
				t.Fatal("typed ordering changed")
			}
			var request map[string]any
			if err = json.Unmarshal(raw, &request); err != nil {
				t.Fatal(err)
			}
			request["columns"].([]any)[0].(map[string]any)["name"] = "separator\u2028 and literal\\u2028"
			unicodeRaw, _ := json.Marshal(request)
			unicodeResponse, err := InvokeSBV(context.Background(), prefix, "0.16.0-dev", cwd, pair[0], unicodeRaw)
			if err != nil {
				t.Fatalf("Unicode query identity: %v", err)
			}
			unicodeResult, err := sqavObject(unicodeResponse.Result, maxResponseBytes)
			if err != nil {
				t.Fatal(err)
			}
			request["cursor"] = unicodeResult["next_cursor"]
			request["limit"] = "1"
			bad, _ := json.Marshal(request)
			if _, err = InvokeSBV(context.Background(), prefix, "0.16.0-dev", cwd, pair[0], bad); err == nil {
				t.Fatal("cursor rebound to changed query")
			}
		}
	}
}

func TestSBVNativeCanonicalUnicode(t *testing.T) {
	value := map[string]any{"a": "\u2028", "b": "\\u2028", "c": "\\\u2029"}
	got, err := sbvNativeCanonical(value)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"a":"` + string(rune(0x2028)) + `","b":"\\u2028","c":"\\` + string(rune(0x2029)) + `"}`
	if string(got) != want {
		t.Fatalf("native canonical mismatch: %q != %q", got, want)
	}
}
