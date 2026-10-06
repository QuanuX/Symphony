package knowledgeengine

import (
	"encoding/json"
	"testing"
)

func TestSBVQueryCorrespondence(t *testing.T) {
	p := map[string]any{"expected_sha256": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", "pointer": "/a", "limit": "2", "cursor": ""}
	query, _ := SCVCanonical(map[string]any{"snapshot": p["expected_sha256"], "pointer": p["pointer"], "limit": p["limit"]})
	hash := digestBytes(query)[7:]
	r := map[string]any{"protocol": "symphony.sbv.result-query.v1", "content_sha256": p["expected_sha256"], "pointer": "/a", "query_sha256": hash, "offset": "0", "total": "1", "complete": true, "next_cursor": "", "nodes": []any{map[string]any{"pointer": "/a/0", "type": "string", "value": "9007199254740993", "children": "0"}}}
	raw, _ := json.Marshal(r)
	if err := validateSBVResult("result_query", p, raw); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"pointer", "query_sha256", "content_sha256", "offset", "total", "next_cursor"} {
		bad := map[string]any{}
		for k, v := range r {
			bad[k] = v
		}
		bad[key] = "wrong"
		b, _ := json.Marshal(bad)
		if validateSBVResult("result_query", p, b) == nil {
			t.Fatal("accepted changed " + key)
		}
	}
}
func TestSBVExactReleaseAndCapabilities(t *testing.T) {
	if _, err := InspectSBV("/private/tmp", "0.1.0-dev"); err == nil {
		t.Fatal("implicit release upgrade")
	}
	p := map[string]any{}
	if validateSBVResult("capabilities", p, []byte(`{"protocol":"symphony.sbv.capabilities.v1","engine_version":"0.9.0-dev","cpu":{"available":true},"cuda":{"available":true},"tensor":{"available":false},"live":{"available":false}}`)) == nil {
		t.Fatal("unexpected accelerator capability accepted")
	}
}
