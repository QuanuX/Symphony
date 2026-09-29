package knowledgeengine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSQAVResultBinding(t *testing.T) {
	payload := []byte(`{"protocol":"symphony.sqav.request-validation-input.v1","adapter":"fred","selection":{},"limits":null}`)
	p, _ := sqavObject(payload, 32768)
	canonical, _ := SCVCanonical(p)
	base := map[string]any{"protocol": "symphony.sqav.request-validation.v1", "adapter": "fred", "adapter_id": "sqav-fred-cpp", "adapter_version": "0.1.0-dev", "request_digest": digestBytes(canonical), "plan_reference": "source-request-v1-" + strings.Repeat("a", 64), "validation_scope": "native_request_only", "provider_observation": "not_performed"}
	seal := func(m map[string]any) []byte {
		delete(m, "result_digest")
		b, _ := SCVCanonical(m)
		m["result_digest"] = digestBytes(b)
		b, _ = SCVCanonical(m)
		return b
	}
	if err := ValidateSQAVRequestResult(payload, seal(base)); err != nil {
		t.Fatal(err)
	}
	for k, v := range map[string]any{"adapter": "databento_reference", "adapter_id": "other", "adapter_version": "latest", "request_digest": "sha256:" + strings.Repeat("b", 64), "plan_reference": "source-request-v1-xyz", "validation_scope": "acquired", "provider_observation": "complete", "extra": true, "protocol": "wrong"} {
		clone := map[string]any{}
		for a, b := range base {
			clone[a] = b
		}
		clone[k] = v
		if ValidateSQAVRequestResult(payload, seal(clone)) == nil {
			t.Fatal("accepted forged " + k)
		}
	}
	for _, bad := range []string{`{"adapter":"fred","adapter":"fred"}`, `{"a":1.2}`, `{"a":9007199254740992}`, `{"a":"\ud800"}`, `{"a":"` + strings.Repeat("x", 32768) + `"}`} {
		if _, err := sqavObject([]byte(bad), 32768); err == nil {
			t.Fatal("invalid transport accepted")
		}
	}
	zero, _ := sqavObject([]byte(`{"offset":-0}`), 32768)
	b, _ := SCVCanonical(zero)
	if string(b) != `{"offset":0}` {
		t.Fatal("integer identity drift")
	}
}
func TestSQAVInstalled(t *testing.T) {
	prefix := os.Getenv("SQAV_REQUEST_TEST_PREFIX")
	if prefix == "" {
		t.Skip("set SQAV_REQUEST_TEST_PREFIX to a verified native installation")
	}
	root, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, adapter := range []string{"fred", "databento_historical", "databento_reference"} {
		t.Run(adapter, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(root, "modules/sqav-request-engine/tests/fixtures", adapter+".json"))
			if err != nil {
				t.Fatal(err)
			}
			r, err := InvokeSQAVRequest(context.Background(), prefix, SQAVRequestInterfaceVersion, t.TempDir(), raw)
			if err != nil {
				t.Fatal(err)
			}
			if ValidateSQAVRequestResult(raw, r.Result) != nil {
				t.Fatal("native correspondence")
			}
			for _, templates := range []bool{false, true} {
				inst, data, err := SQAVRequestResource(prefix, SQAVRequestInterfaceVersion, adapter, templates)
				if err != nil || inst.Version != SQAVRequestInterfaceVersion || !json.Valid(data) {
					t.Fatal(inst, err)
				}
			}
		})
	}
	if _, err := InspectSQAVRequest(prefix, "latest"); err == nil {
		t.Fatal("floating release admitted")
	}
	if _, _, err := SQAVRequestResource(prefix, SQAVRequestInterfaceVersion, "unknown", false); err == nil {
		t.Fatal("unknown adapter admitted")
	}
}
