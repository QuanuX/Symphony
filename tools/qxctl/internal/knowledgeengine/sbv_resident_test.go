package knowledgeengine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSBVResidentControlValidation(t *testing.T) {
	p := map[string]any{"directory": "/private/tmp/host", "instance_id": strings.Repeat("a", 32), "job_id": nil, "after_sequence": "0", "limit": "1"}
	job := map[string]any{"job_id": "job-1", "sequence": "1", "request_sha256": strings.Repeat("a", 64), "operation": "run", "state": "completed", "requested_workers": "18446744073709551615", "cancellation_requested": false, "error_code": "", "output": map[string]any{"output_path": "/private/tmp/result.json"}}
	base := map[string]any{"protocol": "symphony.sbv.dataset-jobs.v1", "engine_version": SBVAdministrationInterfaceVersion, "directory": p["directory"], "instance_id": p["instance_id"], "jobs": []any{job}, "next_after_sequence": "1", "highest_sequence": "1", "more": false}
	validate := func(op string, r map[string]any, want bool) {
		t.Helper()
		raw, _ := json.Marshal(r)
		if err := validateSBVResult(op, p, raw); (err == nil) != want {
			t.Fatalf("%s accepted=%v want=%v: %v", op, err == nil, want, err)
		}
	}
	validate("dataset_jobs", base, true)
	for _, change := range []func(map[string]any){
		func(m map[string]any) { m["more"] = true },
		func(m map[string]any) { m["next_after_sequence"] = "0" },
		func(m map[string]any) { m["highest_sequence"] = "18446744073709551616" },
		func(m map[string]any) { m["instance_id"] = strings.Repeat("b", 32) },
		func(m map[string]any) { m["extra"] = true },
		func(m map[string]any) { m["jobs"].([]any)[0].(map[string]any)["state"] = "invented" },
		func(m map[string]any) {
			m["jobs"].([]any)[0].(map[string]any)["output"] = map[string]any{"output_path": "../escape"}
		},
		func(m map[string]any) { m["jobs"].([]any)[0].(map[string]any)["state"] = "cancelled" },
	} {
		raw, _ := json.Marshal(base)
		var m map[string]any
		_ = json.Unmarshal(raw, &m)
		change(m)
		validate("dataset_jobs", m, false)
	}
	p["job_id"] = "job-1"
	cancel := map[string]any{"protocol": "symphony.sbv.dataset-cancel.v1", "engine_version": SBVAdministrationInterfaceVersion, "directory": p["directory"], "instance_id": p["instance_id"], "job": job}
	validate("dataset_cancel", cancel, true)
	job["state"] = "running"
	validate("dataset_cancel", cancel, false)
	job["cancellation_requested"] = true
	validate("dataset_cancel", cancel, true)
	job["state"] = "cancelled"
	job["error_code"] = "sbv.cancelled"
	validate("dataset_cancel", cancel, true)
	p["job_id"] = "wrong"
	validate("dataset_cancel", cancel, false)
	stop := map[string]any{"protocol": "symphony.sbv.dataset-stop.v1", "engine_version": SBVAdministrationInterfaceVersion, "directory": p["directory"], "instance_id": p["instance_id"], "state": "termination_requested", "active_jobs": "1", "active_workers": "65", "results_may_be_incomplete": true, "automatic_retry": false, "rollback_performed": false}
	validate("dataset_stop", stop, true)
	stop["rollback_performed"] = true
	validate("dataset_stop", stop, false)
}

func TestSBVInstalledResidentControls(t *testing.T) {
	prefix, source := os.Getenv("SYMPHONY_SBV_TEST_PREFIX"), os.Getenv("SYMPHONY_SBV_PARTITIONED_SOURCE_REQUEST")
	if prefix == "" || source == "" {
		t.Skip("installed SBV/source fixture not selected")
	}
	raw, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	var run map[string]any
	if err = json.Unmarshal(raw, &run); err != nil {
		t.Fatal(err)
	}
	scratch := os.Getenv("SYMPHONY_TEST_TMPDIR")
	if scratch == "" {
		scratch = "/tmp"
	}
	dir, err := os.MkdirTemp(scratch, "sbv-go-control-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	ref := map[string]any{"directory": dir, "instance_id": strings.Repeat("a", 32)}
	input := func(op string) map[string]any {
		return map[string]any{"protocol": "symphony.sbv." + strings.ReplaceAll(op, "_", "-") + "-input.v1", "directory": ref["directory"], "instance_id": ref["instance_id"]}
	}
	invoke := func(op string, p map[string]any) (map[string]any, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		raw, _ := json.Marshal(p)
		r, e := InvokeSBV(ctx, prefix, SBVAdministrationInterfaceVersion, dir, op, raw)
		if e != nil {
			return nil, e
		}
		var m map[string]any
		e = json.Unmarshal(r.Result, &m)
		return m, e
	}
	call := func(op string, p map[string]any) map[string]any {
		t.Helper()
		r, e := invoke(op, p)
		if e != nil {
			t.Fatal(op, e)
		}
		return r
	}
	load := input("dataset_load")
	for _, key := range []string{"source_path", "source_sha256", "dataset"} {
		load[key] = run[key]
	}
	load["residency"] = "pageable"
	load["memory_budget_bytes"] = nil
	load["max_concurrent_jobs"] = nil
	load["worker_budget"] = nil
	load["idle_timeout_ms"] = "0"
	call("dataset_load", load)
	defer func() { _, _ = invoke("dataset_stop", input("dataset_stop")) }()
	config := input("dataset_configure")
	config["expected_generation"] = "0"
	config["max_concurrent_jobs"] = nil
	config["worker_budget"] = "18446744073709551615"
	config["idle_timeout_ms"] = "0"
	config["on_active_over_limit"] = "reject"
	if call("dataset_configure", config)["configuration_generation"] != "1" {
		t.Fatal("generation")
	}
	if _, e := invoke("dataset_configure", config); e == nil {
		t.Fatal("stale configure admitted")
	}
	delete(run, "output_path")
	delete(run, "output")
	run["workers"] = "1"
	task := input("dataset_execute")
	task["job_id"] = "cli-parity"
	task["operation"] = "run"
	task["request"] = run
	task["output_path"] = filepath.Join(dir, "result.json")
	call("dataset_execute", task)
	jobs := input("dataset_jobs")
	jobs["job_id"] = nil
	jobs["after_sequence"] = "0"
	jobs["limit"] = "1"
	page := call("dataset_jobs", jobs)
	if len(page["jobs"].([]any)) != 1 {
		t.Fatal("job listing")
	}
	cancel := input("dataset_cancel")
	cancel["job_id"] = "cli-parity"
	if call("dataset_cancel", cancel)["job"].(map[string]any)["state"] != "completed" {
		t.Fatal("terminal cancellation changed outcome")
	}
	call("dataset_stop", input("dataset_stop"))
}
