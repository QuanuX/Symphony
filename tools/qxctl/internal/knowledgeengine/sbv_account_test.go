package knowledgeengine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSBVInstalledAccountSimulation(t *testing.T) {
	prefix, fixture := os.Getenv("SYMPHONY_SBV_TEST_PREFIX"), os.Getenv("SYMPHONY_SBV_ACCOUNT_FIXTURE")
	if prefix == "" || fixture == "" {
		t.Skip("select exact installation and account fixture directory")
	}
	raw, err := os.ReadFile(filepath.Join(fixture, "request.json"))
	if err != nil {
		t.Fatal(err)
	}
	var p map[string]any
	if err = json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	p["output_path"] = filepath.Join(dir, "account.json")
	invoke := func(op string, request any) (Response, error) {
		b, e := json.Marshal(request)
		if e != nil {
			t.Fatal(e)
		}
		return InvokeSBV(context.Background(), prefix, SBVAdministrationInterfaceVersion, cwd, op, b)
	}
	response, err := invoke("account_simulate", p)
	if err != nil {
		t.Fatal(err)
	}
	var receipt map[string]any
	if err = json.Unmarshal(response.Result, &receipt); err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile(p["output_path"].(string))
	if err != nil {
		t.Fatal(err)
	}
	expected, err := os.ReadFile(filepath.Join(fixture, "expected.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got, want map[string]any
	if json.Unmarshal(actual, &got) != nil || json.Unmarshal(expected, &want) != nil {
		t.Fatal("invalid result")
	}
	if receipt["content_sha256"] != got["content_sha256"] {
		t.Fatal("receipt identity")
	}
	gs, ws := got["sections"].(map[string]any), want["sections"].(map[string]any)
	for _, name := range []string{"summary", "execution", "signals", "snapshots", "terminal_state"} {
		if !reflect.DeepEqual(gs[name], ws[name]) {
			t.Fatalf("native/qxctl science differs: %s", name)
		}
	}
	schema, err := SBVSchema(prefix, SBVAdministrationInterfaceVersion, "account_simulate")
	if err != nil {
		t.Fatal(err)
	}
	contract, err := sbvDecodeOwnedResource(schema)
	if err != nil {
		t.Fatal(err)
	}
	defs := contract["$defs"].(map[string]any)
	if !sbvSchemaShape(defs["account_state"], gs["terminal_state"].(map[string]any)["data"], 0) {
		t.Fatal("typed account state schema")
	}
	for _, variant := range []string{"", "feedback", "partitioned", "shared_liquidity", "target_position", "position_split", "native_feedback", "linear_variation"} {
		if _, _, err := SBVTemplateVariant(prefix, SBVAdministrationInterfaceVersion, "account_simulate", variant); err != nil {
			t.Fatal(err)
		}
	}
	if _, selected := p["feedback"]; selected {
		if !reflect.DeepEqual(got, want) {
			t.Fatal("complete native/qxctl feedback payload differs")
		}
		provider := gs["feedback_provider"].(map[string]any)["data"].(map[string]any)
		var descriptor map[string]any
		if json.Unmarshal([]byte(provider["descriptor_json"].(string)), &descriptor) != nil {
			t.Fatal("feedback descriptor JSON")
		}
		if !sbvSchemaShape(defs["native_feedback_descriptor"], descriptor, 0) {
			t.Fatal("feedback descriptor schema")
		}
		if _, err := invoke("provider_inspect", map[string]any{"protocol": "symphony.sbv.provider-inspect-input.v1", "provider": p["feedback"].(map[string]any)["provider"], "extensions": map[string]any{}}); err != nil {
			t.Fatal(err)
		}
		for _, value := range gs["execution"].(map[string]any)["data"].([]any) {
			row := value.(map[string]any)
			if row["feedback"] == nil {
				continue
			}
			feedback := row["feedback"].(map[string]any)
			if !sbvSchemaShape(defs["account_feedback_call"], feedback["call"], 0) {
				t.Fatal("feedback call schema")
			}
			if feedback["status"] == "0" && !sbvSchemaShape(defs["feedback_decision"], feedback["decision"], 0) {
				t.Fatal("feedback decision schema")
			}
		}
		if _, _, err := SBVTemplateVariant(prefix, SBVAdministrationInterfaceVersion, "provider_inspect", "feedback"); err != nil {
			t.Fatal(err)
		}
	}
	// Every retained row remains accessible through the standard exact result reader.
	_, err = invoke("result_inspect", map[string]any{"protocol": "symphony.sbv.result-inspect-input.v1", "path": p["output_path"], "expected_sha256": got["content_sha256"]})
	if err != nil {
		t.Fatal(err)
	}
	p["output_path"] = filepath.Join(dir, "invalid.json")
	p["mode"] = "implicit_feedback"
	if _, err = invoke("account_simulate", p); err == nil {
		t.Fatal("unknown mode accepted")
	}
	if _, err = os.Stat(p["output_path"].(string)); !os.IsNotExist(err) {
		t.Fatal("invalid result published")
	}
}
