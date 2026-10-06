package knowledgeengine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Optional exact-prefix integration, run at the SBV increment gate. It performs
// no network access and uses only user-supplied mathematical scenarios.
func TestSBVInstalledBoundary(t *testing.T) {
	prefix := os.Getenv("SYMPHONY_SBV_TEST_PREFIX")
	if prefix == "" {
		t.Skip("set exact installed SBV prefix")
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	invoke := func(op string, p any) (Response, error) {
		raw, e := json.Marshal(p)
		if e != nil {
			t.Fatal(e)
		}
		return InvokeSBV(context.Background(), prefix, "0.10.0-dev", cwd, op, raw)
	}
	for _, op := range SBVOperations {
		if _, _, e := SBVResource(prefix, "0.10.0-dev", op, true); e != nil {
			t.Fatal(op, e)
		}
		if _, _, e := SBVResource(prefix, "0.10.0-dev", op, false); e != nil {
			t.Fatal(op, e)
		}
	}
	if _, e := invoke("capabilities", map[string]any{"protocol": "symphony.sbv.capabilities-input.v1"}); e != nil {
		t.Fatal(e)
	}
	if _, e := invoke("catalogue", map[string]any{"protocol": "symphony.sbv.catalogue-input.v1"}); e != nil {
		t.Fatal(e)
	}
	ratio := func(n, d string) any { return map[string]any{"numerator": n, "denominator": d} }
	p := map[string]any{"protocol": "symphony.sbv.compose-input.v1", "returns": []any{ratio("-1", "100"), ratio("1", "100")}, "weights": []any{ratio("1", "2"), ratio("1", "2")}, "steps": "2", "dependence": "independent", "output_path": filepath.Join(dir, "result.json"), "extensions": map[string]any{"private-study": map[string]any{"status": "supplied_not_computed", "value": "9223372036854775807"}}}
	r, e := invoke("compose", p)
	if e != nil {
		t.Fatal(e)
	}
	var creation map[string]any
	if json.Unmarshal(r.Result, &creation) != nil {
		t.Fatal("creation JSON")
	}
	if _, e = invoke("compose", p); e == nil {
		t.Fatal("overwrite admitted")
	}
	inspect := map[string]any{"protocol": "symphony.sbv.result-inspect-input.v1", "path": p["output_path"], "expected_sha256": creation["content_sha256"]}
	r, e = invoke("result_inspect", inspect)
	if e != nil {
		t.Fatal(e)
	}
	raw, e := ReadSBVExport(p["output_path"].(string), r.Result)
	if e != nil {
		t.Fatal(e)
	}
	if !json.Valid(raw) {
		t.Fatal("export invalid")
	}
	q := map[string]any{"protocol": "symphony.sbv.result-query-input.v1", "path": p["output_path"], "expected_sha256": creation["content_sha256"], "pointer": "/sections/distributions/data/paths", "limit": "1", "cursor": ""}
	seen := map[string]bool{}
	for {
		page, e := invoke("result_query", q)
		if e != nil {
			t.Fatal(e)
		}
		var m map[string]any
		json.Unmarshal(page.Result, &m)
		for _, node := range m["nodes"].([]any) {
			pointer := node.(map[string]any)["pointer"].(string)
			if seen[pointer] {
				t.Fatal("duplicate cursor node")
			}
			seen[pointer] = true
		}
		if m["complete"] == true {
			break
		}
		q["cursor"] = m["next_cursor"]
	}
	if len(seen) != 4 {
		t.Fatal("lost composed path")
	}
	q["pointer"] = "/sections/signals"
	if _, e = invoke("result_query", q); e == nil {
		t.Fatal("cursor changed query accepted")
	}
	inspect["expected_sha256"] = ""
	if _, e = invoke("result_inspect", inspect); e != nil {
		t.Fatal("cannot inspect uncertain creation", e)
	}
	if err = os.WriteFile(p["output_path"].(string), append(raw, ' '), 0600); err != nil {
		t.Fatal(err)
	}
	if _, e = ReadSBVExport(p["output_path"].(string), r.Result); e == nil {
		t.Fatal("post-inspection bytes changed undetected")
	}
	joint := map[string]any{"protocol": "symphony.sbv.compose-joint-input.v1", "output_path": filepath.Join(dir, "joint.json"), "extensions": map[string]any{}, "paths": []any{
		map[string]any{"path_id": "mixed-a", "returns": []any{ratio("-1", "100"), ratio("1", "100")}, "probability": ratio("1", "2")},
		map[string]any{"path_id": "mixed-b", "returns": []any{ratio("1", "100"), ratio("-1", "100")}, "probability": ratio("1", "2")},
	}}
	if _, e = invoke("compose_joint", joint); e != nil {
		t.Fatal(e)
	}
	if _, e = InspectSBV(prefix, "0.1.0-dev"); e == nil {
		t.Fatal("new installation silently admitted as the old release")
	}
}
