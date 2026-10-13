package sovremote

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func encoded(t *testing.T, v any) []byte {
	t.Helper()
	b, e := json.Marshal(v)
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func selection(t *testing.T) Selection {
	t.Helper()
	v, e := Template("plan")
	if e != nil {
		t.Fatal(e)
	}
	return v.(Selection)
}
func ptr(s string) *string { return &s }
func TestPlanSelectionsAndUnboundedResources(t *testing.T) {
	s := selection(t)
	s.Resources.MemoryBytes = ptr("18446744073709551616000000000000")
	s.Resources.Threads = ptr("999999999999999999999")
	s.Resources.DeadlineUnixMS = ptr("99999999999999999999999999")
	s.Resources.Devices = []string{"user:device:choice"}
	s.Extensions["user-model"] = "anything <chosen>"
	for _, arrangement := range []string{"controller", "target", "staged"} {
		for _, family := range []string{"ipc", "scoped_cli"} {
			t.Run(arrangement+"/"+family, func(t *testing.T) {
				s.Transport.Arrangement = arrangement
				s.Transport.Family = family
				p, e := MakePlan(encoded(t, s))
				if e != nil {
					t.Fatal(e)
				}
				if p.RemoteOperational || p.AuthorityVerified || p.ResourcesReserved || p.AutomaticRetry || p.CheckpointResumeSupported || p.Status != "planned_not_dispatched" {
					t.Fatal(p)
				}
				if *p.Selection.Resources.MemoryBytes != *s.Resources.MemoryBytes {
					t.Fatal("clamped selection")
				}
				pretty, e := json.MarshalIndent(s, "", "  ")
				if e != nil {
					t.Fatal(e)
				}
				q, e := MakePlan(pretty)
				if e != nil || q.PlanSHA256 != p.PlanSHA256 {
					t.Fatal("unstable plan")
				}
			})
		}
	}
	s = selection(t)
	p, e := MakePlan(encoded(t, s))
	if e != nil {
		t.Fatal(e)
	}
	if p.Selection.Resources.Threads != nil || p.Selection.Resources.MemoryBytes != nil || p.Selection.Resources.DeadlineUnixMS != nil {
		t.Fatal("implicit resource limit")
	}
}
func TestPlanBindsEverySelection(t *testing.T) {
	s := selection(t)
	p, e := MakePlan(encoded(t, s))
	if e != nil {
		t.Fatal(e)
	}
	mutations := map[string]func(*Selection){
		"operation": func(s *Selection) { s.OperationID = "other" }, "target": func(s *Selection) { s.Target.Node = "other" }, "target_digest": func(s *Selection) { s.Target.IdentitySHA256 = strings.Repeat("a", 64) },
		"adapter": func(s *Selection) { s.Transport.AdapterSHA256 = strings.Repeat("a", 64) }, "endpoint": func(s *Selection) { s.Transport.EndpointRef = "other" }, "arrangement": func(s *Selection) { s.Transport.Arrangement = "target" },
		"runtime": func(s *Selection) { s.Runtime.QXCTLSHA256 = strings.Repeat("a", 64) }, "job": func(s *Selection) { s.Job.PlanSHA256 = strings.Repeat("a", 64) }, "memory": func(s *Selection) { s.Resources.MemoryBytes = ptr("1") },
		"deadline": func(s *Selection) { s.Resources.DeadlineUnixMS = ptr("1") }, "authority": func(s *Selection) { s.Authority.PolicySHA256 = ptr(strings.Repeat("a", 64)) }, "extension": func(s *Selection) { s.Extensions["new"] = "choice" },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			s := selection(t)
			mutate(&s)
			q, e := MakePlan(encoded(t, s))
			if e != nil || q.PlanSHA256 == p.PlanSHA256 {
				t.Fatalf("unbound selection %v", e)
			}
		})
	}
}
func TestPlanClosedInputs(t *testing.T) {
	good := encoded(t, selection(t))
	mutations := map[string]func(map[string]any){
		"missing": func(m map[string]any) { delete(m, "extensions") }, "extra": func(m map[string]any) { m["password"] = "rejected" }, "case_alias": func(m map[string]any) { m["Protocol"] = m["protocol"]; delete(m, "protocol") },
		"null_object": func(m map[string]any) { m["target"] = nil }, "null_extension_value": func(m map[string]any) { m["extensions"].(map[string]any)["x"] = nil },
		"missing_null": func(m map[string]any) { delete(m["resources"].(map[string]any), "threads") }, "number": func(m map[string]any) { m["resources"].(map[string]any)["memory_bytes"] = 123 },
		"zero": func(m map[string]any) { m["resources"].(map[string]any)["threads"] = "0" }, "leading_zero": func(m map[string]any) { m["resources"].(map[string]any)["threads"] = "01" },
		"negative_deadline": func(m map[string]any) { m["resources"].(map[string]any)["deadline_unix_ms"] = "-1" }, "nil_devices": func(m map[string]any) { m["resources"].(map[string]any)["device_refs"] = nil },
		"duplicate_device": func(m map[string]any) { m["resources"].(map[string]any)["device_refs"] = []string{"a", "a"} }, "wrong_version": func(m map[string]any) { m["runtime"].(map[string]any)["sbv_version"] = "latest" },
		"adapter_upgrade": func(m map[string]any) { m["transport"].(map[string]any)["adapter_version"] = "^1" }, "bad_sha": func(m map[string]any) { m["target"].(map[string]any)["identity_sha256"] = "none" },
		"relative_job": func(m map[string]any) { m["job"].(map[string]any)["directory"] = "relative" }, "aliased_job": func(m map[string]any) { m["job"].(map[string]any)["directory"] = "/one/../two" },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			var m map[string]any
			_ = json.Unmarshal(good, &m)
			mutate(m)
			if _, e := MakePlan(encoded(t, m)); e == nil {
				t.Fatal("invalid input accepted")
			}
		})
	}
	for name, raw := range map[string][]byte{"duplicate": append([]byte(`{"protocol":"duplicate",`), good[1:]...), "trailing": append(append([]byte{}, good...), []byte(` {}`)...), "invalid_utf8": append([]byte(`{"x":"`), 0xff)} {
		t.Run(name, func(t *testing.T) {
			if _, e := MakePlan(raw); e == nil {
				t.Fatal("malformed input accepted")
			}
		})
	}
}
func observation(s Selection) map[string]any {
	d := strings.Repeat("a", 64)
	ref := func(name, digest string) any {
		return map[string]any{"path": s.Job.Directory + "/" + name, "expected_sha256": digest, "pointer": ""}
	}
	return map[string]any{"protocol": "symphony.sbv.job-inspect.v1", "engine_version": s.Runtime.SBVVersion, "directory": s.Job.Directory, "job_id": s.Job.ID, "plan_sha256": s.Job.PlanSHA256, "generation": "0", "phase": "queued", "cancellation_requested": false, "cancellation_observed": false, "execution_lock_held": false, "admitted": false, "automatic_retry": false, "result_receipt": nil, "error_code": nil, "plan_reference": ref("plan.json", s.Job.PlanSHA256), "state_reference": ref("state-"+d+".json", d), "outcome_reference": nil, "claim_reference": nil, "cancellation_reference": nil}
}
func TestLostResponseAndNativeCrashPrefixes(t *testing.T) {
	s := selection(t)
	p, _ := build(s)
	in := AssessmentInput{Protocol: AssessmentProtocol, Selection: s, ExpectedPlanSHA256: p.PlanSHA256, TransportOutcome: "response_lost", Observation: json.RawMessage("null")}
	a, e := Assess(encoded(t, in))
	if e != nil || a.RecommendedAction != "inspect_same_job" {
		t.Fatalf("%+v %v", a, e)
	}
	in.TransportOutcome = "not_dispatched"
	a, e = Assess(encoded(t, in))
	if e != nil || a.RecommendedAction != "qualify_target_and_authority" {
		t.Fatal(a, e)
	}
	for _, tc := range []struct {
		name, phase               string
		admitted, locked, outcome bool
		action                    string
	}{
		{"queued", "queued", false, false, false, "qualify_target_and_authority"},
		{"claim_before_head", "queued", true, false, false, "reconcile_same_job"},
		{"active", "running", true, true, false, "observe_same_job"},
		{"lost_worker", "running", true, false, false, "reconcile_same_job"},
		{"unknown", "unknown", true, false, false, "reconcile_same_job"},
		{"completed", "completed", true, false, true, "inspect_bound_result"},
		{"partial", "completed_partial", true, false, true, "inspect_bound_result"},
		{"failed", "failed", true, false, true, "reconcile_same_job"},
		{"cancelled_before_admission", "cancelled", false, false, false, "reconcile_same_job"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o := observation(s)
			o["phase"] = tc.phase
			o["admitted"] = tc.admitted
			o["execution_lock_held"] = tc.locked
			ref := func(name string) any {
				return map[string]any{"path": s.Job.Directory + "/" + name, "expected_sha256": strings.Repeat("a", 64), "pointer": ""}
			}
			if tc.admitted {
				o["claim_reference"] = ref("claim.json")
			}
			if tc.outcome {
				o["outcome_reference"] = ref("outcome.json")
			}
			if tc.phase == "completed" || tc.phase == "completed_partial" {
				status := "completed"
				if tc.phase == "completed_partial" {
					status = "partial"
				}
				o["result_receipt"] = map[string]any{"protocol": "symphony.sbv.run.v1", "status": status}
			}
			in.TransportOutcome = "response_received"
			in.ObservedTargetSHA256 = &s.Target.IdentitySHA256
			in.Observation = encoded(t, o)
			a, e := Assess(encoded(t, in))
			if e != nil || a.RecommendedAction != tc.action || a.AutomaticRetry || a.AuthorityVerified || a.EvidenceStatus != "supplied_unverified" {
				t.Fatal(a, e)
			}
			if !bytes.Equal(a.Observation, in.Observation) {
				t.Fatal("discarded native evidence")
			}
			in.TransportOutcome = "response_lost"
			a, e = Assess(encoded(t, in))
			if e != nil || a.RecommendedAction != "inspect_same_job" {
				t.Fatal("stale observation settled transport loss", e)
			}
		})
	}
}
func TestAssessmentRejectsDrift(t *testing.T) {
	s := selection(t)
	p, _ := build(s)
	for _, key := range []string{"job_id", "plan_sha256", "directory", "engine_version", "phase", "generation"} {
		t.Run(key, func(t *testing.T) {
			o := observation(s)
			o[key] = "wrong"
			in := AssessmentInput{AssessmentProtocol, s, p.PlanSHA256, "response_received", &s.Target.IdentitySHA256, encoded(t, o)}
			if _, e := Assess(encoded(t, in)); e == nil {
				t.Fatal("drift admitted")
			}
		})
	}
	for _, key := range []string{"plan", "target", "missing_observation", "not_dispatched", "missing_target"} {
		t.Run(key, func(t *testing.T) {
			in := AssessmentInput{AssessmentProtocol, s, p.PlanSHA256, "response_received", &s.Target.IdentitySHA256, encoded(t, observation(s))}
			switch key {
			case "plan":
				in.ExpectedPlanSHA256 = strings.Repeat("a", 64)
			case "target":
				in.ObservedTargetSHA256 = ptr(strings.Repeat("b", 64))
			case "missing_observation":
				in.Observation = json.RawMessage("null")
			case "not_dispatched":
				in.TransportOutcome = "not_dispatched"
			case "missing_target":
				in.ObservedTargetSHA256 = nil
			}
			if _, e := Assess(encoded(t, in)); e == nil {
				t.Fatal("drift admitted")
			}
		})
	}
}
func TestResourcesMatchRuntimeShapes(t *testing.T) {
	for _, op := range []string{"plan", "assess"} {
		t.Run(op, func(t *testing.T) {
			v, e := Template(op)
			if e != nil {
				t.Fatal(e)
			}
			if op == "plan" {
				_, e = MakePlan(encoded(t, v))
			} else {
				_, e = Assess(encoded(t, v))
			}
			if e != nil {
				t.Fatal(e)
			}
			s, e := Schema(op)
			if e != nil {
				t.Fatal(e)
			}
			if len(encoded(t, s)) < 100 {
				t.Fatal("empty schema")
			}
		})
	}
	if _, e := Template("execute"); e == nil {
		t.Fatal("operational template advertised")
	}
	if _, e := Schema("execute"); e == nil {
		t.Fatal("operational schema advertised")
	}
}
