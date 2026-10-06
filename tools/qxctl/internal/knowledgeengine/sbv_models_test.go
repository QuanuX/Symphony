package knowledgeengine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSBVInstalledModels(t *testing.T) {
	prefix, fixture := os.Getenv("SYMPHONY_SBV_TEST_PREFIX"), os.Getenv("SYMPHONY_SBV_MODEL_FIXTURE")
	if prefix == "" || fixture == "" {
		t.Skip("set exact installed prefix and emitted offline model fixture")
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
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	request["output_path"] = filepath.Join(dir, "observed.json")
	invoke := func(p map[string]any) (Response, error) {
		b, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		return InvokeSBV(context.Background(), prefix, "0.13.0-dev", cwd, "evaluate", b)
	}
	if _, err = invoke(request); err != nil {
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
	sections := artifact["sections"].(map[string]any)
	rows := sections["execution"].(map[string]any)["data"].([]any)
	schema, err := SBVSchema(prefix, "0.13.0-dev", "evaluate")
	if err != nil {
		t.Fatal(err)
	}
	contract, err := sqavObject(schema, maxRequestBytes)
	if err != nil {
		t.Fatal(err)
	}
	definition := contract["$defs"].(map[string]any)["model_outcome"]
	for _, row := range rows {
		if !sqvTransportShape(definition, row, 0) {
			t.Fatal("native outcome does not match installed discoverable schema")
		}
	}
	if len(rows) != 2 || rows[0].(map[string]any)["status"] != "available" || rows[1].(map[string]any)["status"] != "unavailable" {
		t.Fatal("fixture model behavior")
	}
	census := sections["summary"].(map[string]any)["data"].(map[string]any)["census_sha256"]
	producer := map[string]any{"id": "external-test", "version": "1", "artifact_sha256": "", "reproducibility": "nondeterministic"}
	rat := func(n, d string) any { return map[string]any{"numerator": n, "denominator": d} }
	var supplied []any
	for _, row := range rows {
		supplied = append(supplied, map[string]any{"signal_id": row.(map[string]any)["signal_id"], "evidence_reference": "user://scenario",
			"execution_probability": map[string]any{"status": "unavailable", "reason": "signed measure"},
			"support":               []any{map[string]any{"price_nanos": "99000000000", "weight": rat("-1", "2")}, map[string]any{"price_nanos": "101000000000", "weight": rat("1", "2")}}})
	}
	parameters := map[string]any{"producer": producer, "measure": "signed_coefficient", "conditioning": "scenario", "calibration_reference": "", "outcomes": supplied}
	request["model"] = map[string]any{"protocol": "symphony.sbv.model-selection.v1", "id": "external_outcomes", "version": "1", "horizon_ns": "1100", "parameters": parameters}
	request["output_path"] = filepath.Join(dir, "external.json")
	if _, err = invoke(request); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(request["output_path"].(string))
	if err != nil {
		t.Fatal(err)
	}
	artifact, err = sqavObject(data, 128<<20)
	if err != nil {
		t.Fatal(err)
	}
	sections = artifact["sections"].(map[string]any)
	if sections["summary"].(map[string]any)["data"].(map[string]any)["census_sha256"] != census {
		t.Fatal("model changed census identity")
	}
	for _, row := range sections["execution"].(map[string]any)["data"].([]any) {
		if !sqvTransportShape(definition, row, 0) {
			t.Fatal("external outcome schema mismatch")
		}
	}
	parameters["measure"] = "probability"
	request["output_path"] = filepath.Join(dir, "invalid.json")
	if _, err = invoke(request); err == nil {
		t.Fatal("signed weights admitted as probability")
	}
	if _, err = os.Stat(request["output_path"].(string)); !os.IsNotExist(err) {
		t.Fatal("invalid model published an artifact")
	}
}
