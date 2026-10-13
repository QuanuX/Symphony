package knowledgeengine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSBVJobResponseBoundary(t *testing.T) {
	digest := strings.Repeat("a", 64)
	ref := func(name string) any {
		return map[string]any{"path": "/job/" + name, "expected_sha256": digest, "pointer": ""}
	}
	p := map[string]any{"directory": "/job", "job_id": "test", "expected_plan_sha256": digest}
	good := map[string]any{"protocol": "symphony.sbv.job-inspect.v1", "engine_version": SBVAdministrationInterfaceVersion, "directory": "/job", "job_id": "test", "plan_sha256": digest, "generation": "0", "phase": "queued", "cancellation_requested": false, "cancellation_observed": false, "execution_lock_held": false, "admitted": false, "automatic_retry": false, "result_receipt": nil, "error_code": nil, "plan_reference": ref("plan.json"), "state_reference": ref("state-" + digest + ".json"), "outcome_reference": nil, "claim_reference": nil, "cancellation_reference": nil}
	encode := func(v any) []byte {
		b, e := json.Marshal(v)
		if e != nil {
			t.Fatal(e)
		}
		return b
	}
	if e := validateSBVResult("job_inspect", p, encode(good)); e != nil {
		t.Fatal(e)
	}
	t.Run("claim_before_state", func(t *testing.T) {
		image := map[string]any{}
		for k, v := range good {
			image[k] = v
		}
		image["admitted"] = true
		image["claim_reference"] = ref("claim.json")
		if e := validateSBVResult("job_inspect", p, encode(image)); e != nil {
			t.Fatal(e)
		}
	})
	t.Run("closed_fields", func(t *testing.T) {
		image := map[string]any{}
		for k, v := range good {
			image[k] = v
		}
		delete(image, "error_code")
		image["unknown"] = nil
		if validateSBVResult("job_inspect", p, encode(image)) == nil {
			t.Fatal("missing field accepted")
		}
	})
	t.Run("exact_phase", func(t *testing.T) {
		image := map[string]any{}
		for k, v := range good {
			image[k] = v
		}
		image["phase"] = "queued|running"
		if validateSBVResult("job_inspect", p, encode(image)) == nil {
			t.Fatal("combined phase accepted")
		}
	})
	cases := map[string]any{"generation": "01", "phase": "invented", "job_id": "other", "plan_sha256": strings.Repeat("b", 64), "engine_version": "latest", "automatic_retry": true, "admitted": true, "cancellation_requested": "false", "state_reference": ref("state-other.json"), "plan_reference": ref("other.json"), "outcome_reference": ref("other.json"), "error_code": false, "result_receipt": map[string]any{"status": "completed"}}
	for field, value := range cases {
		t.Run(field, func(t *testing.T) {
			bad := map[string]any{}
			for k, v := range good {
				bad[k] = v
			}
			bad[field] = value
			if validateSBVResult("job_inspect", p, encode(bad)) == nil {
				t.Fatal("invalid response accepted")
			}
		})
	}
}

func TestSBVInstalledDurableJobs(t *testing.T) {
	prefix := os.Getenv("SYMPHONY_SBV_TEST_PREFIX")
	if prefix == "" {
		t.Skip("select exact installed SBV prefix")
	}
	root, e := filepath.EvalSymlinks(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	directory := filepath.Join(root, "job")
	if e = os.Mkdir(directory, 0700); e != nil {
		t.Fatal(e)
	}
	_, raw, e := SBVTemplateVariant(prefix, SBVAdministrationInterfaceVersion, "job_create", "")
	if e != nil {
		t.Fatal(e)
	}
	var create map[string]any
	if e = json.Unmarshal(raw, &create); e != nil {
		t.Fatal(e)
	}
	create["directory"] = directory
	create["request"].(map[string]any)["output_path"] = filepath.Join(root, "result.json")
	invoke := func(op string, p map[string]any) map[string]any {
		p["protocol"] = "symphony.sbv." + strings.ReplaceAll(op, "_", "-") + "-input.v1"
		b, _ := json.Marshal(p)
		r, e := InvokeSBV(context.Background(), prefix, SBVAdministrationInterfaceVersion, root, op, b)
		if e != nil {
			t.Fatal(op, e)
		}
		var got map[string]any
		if e = json.Unmarshal(r.Result, &got); e != nil {
			t.Fatal(e)
		}
		schema, e := SBVSchema(prefix, SBVAdministrationInterfaceVersion, op)
		if e != nil {
			t.Fatal(e)
		}
		shape, e := sbvDecodeOwnedResource(schema)
		if e != nil || !sbvSchemaShape(shape["output"], got, 0) {
			t.Fatal(op, "output schema", e)
		}
		return got
	}
	queued := invoke("job_create", create)
	address := map[string]any{"directory": directory, "job_id": create["job_id"], "expected_plan_sha256": queued["plan_sha256"]}
	complete := invoke("job_execute", address)
	if complete["phase"] != "completed" {
		t.Fatal(complete)
	}
	inspect := invoke("job_inspect", address)
	if !reflect.DeepEqual(complete["result_receipt"], inspect["result_receipt"]) {
		t.Fatal("receipt changed")
	}
	duplicate := invoke("job_execute", address)
	if !reflect.DeepEqual(complete, duplicate) {
		t.Fatal("duplicate changed state")
	}
	address["expected_generation"] = complete["generation"]
	reconciled := invoke("job_reconcile", address)
	if reconciled["generation"] != complete["generation"] {
		t.Fatal("reconciliation repeated mutation")
	}
	address["request_id"] = "too-late"
	address["reason"] = "completion already won"
	late := invoke("job_cancel", address)
	if late["phase"] != "completed" || late["cancellation_requested"] != false {
		t.Fatal("late cancel changed completion")
	}
	r := complete["result_receipt"].(map[string]any)
	result := invoke("result_inspect", map[string]any{"path": r["path"], "expected_sha256": r["content_sha256"]})
	if result["content_sha256"] != r["content_sha256"] {
		t.Fatal("result retargeted")
	}
	// Reconstruct the durable prefix after admission but before state/outcome
	// publication. An output at the chosen path is not authoritative completion.
	initial, e := os.ReadFile(queued["state_reference"].(map[string]any)["path"].(string))
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(directory, "state.json"), initial, 0600); e != nil {
		t.Fatal(e)
	}
	if e = os.Remove(filepath.Join(directory, "outcome.json")); e != nil {
		t.Fatal(e)
	}
	address = map[string]any{"directory": directory, "job_id": create["job_id"], "expected_plan_sha256": queued["plan_sha256"]}
	orphan := invoke("job_inspect", address)
	if orphan["phase"] != "queued" || orphan["admitted"] != true {
		t.Fatal("claim prefix hidden")
	}
	address["expected_generation"] = orphan["generation"]
	unknown := invoke("job_reconcile", address)
	if unknown["phase"] != "unknown" || unknown["result_receipt"] != nil {
		t.Fatal("unproven result adopted")
	}
	delete(address, "expected_generation")
	unchanged := invoke("job_execute", address)
	if unchanged["phase"] != "unknown" || unchanged["generation"] != unknown["generation"] {
		t.Fatal("uncertain attempt repeated")
	}
}
