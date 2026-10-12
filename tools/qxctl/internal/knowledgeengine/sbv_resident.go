package knowledgeengine

import (
	"fmt"
	"regexp"
	"strconv"
)

var sbvJobID = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,128}$`)

func sbvResidentUint(v any) (uint64, bool) {
	s, ok := v.(string)
	n, err := strconv.ParseUint(s, 10, 64)
	return n, ok && err == nil && strconv.FormatUint(n, 10) == s
}

func sbvResidentJob(v any) (map[string]any, bool) {
	j, ok := v.(map[string]any)
	if !ok || !sbvRecoveryFields(j, "job_id", "sequence", "request_sha256", "operation", "state", "requested_workers", "cancellation_requested", "error_code", "output") {
		return nil, false
	}
	id, ok := j["job_id"].(string)
	hash, hashed := j["request_sha256"].(string)
	cancelled, flagged := j["cancellation_requested"].(bool)
	code, coded := j["error_code"].(string)
	if !ok || !sbvJobID.MatchString(id) || !hashed || !sbvSHA.MatchString(hash) || !flagged || !coded {
		return nil, false
	}
	for _, k := range []string{"sequence", "requested_workers"} {
		if n, ok := sbvResidentUint(j[k]); !ok || n == 0 {
			return nil, false
		}
	}
	switch j["operation"] {
	case "run", "evaluate", "book", "generate_census":
	default:
		return nil, false
	}
	switch j["state"] {
	case "running", "completed":
		if code != "" {
			return nil, false
		}
	case "cancelled":
		if !cancelled || code != "sbv.cancelled" {
			return nil, false
		}
	case "recovery_required":
		if code != "sbv.dataset_execution_incomplete" {
			return nil, false
		}
	case "failed":
		if code == "" {
			return nil, false
		}
	default:
		return nil, false
	}
	output, ok := j["output"].(map[string]any)
	if !ok || len(output) != 1 {
		return nil, false
	}
	if value, present := output["output_path"]; present {
		if !sbvRecoveryPath(value) {
			return nil, false
		}
	} else if !sbvPartitionedOutput(output["output"]) {
		return nil, false
	}
	return j, true
}

func sbvResidentControlResult(op string, p, m map[string]any) error {
	bad := func() error { return fmt.Errorf("SBV resident control contract mismatch") }
	if m["engine_version"] != SBVAdministrationInterfaceVersion || m["directory"] != p["directory"] || m["instance_id"] != p["instance_id"] {
		return bad()
	}
	switch op {
	case "dataset_stop":
		if !sbvRecoveryFields(m, "protocol", "engine_version", "directory", "instance_id", "state", "active_jobs", "active_workers", "results_may_be_incomplete", "automatic_retry", "rollback_performed") || m["state"] != "termination_requested" || m["results_may_be_incomplete"] != true || m["automatic_retry"] != false || m["rollback_performed"] != false {
			return bad()
		}
		for _, k := range []string{"active_jobs", "active_workers"} {
			if _, ok := sbvResidentUint(m[k]); !ok {
				return bad()
			}
		}
	case "dataset_cancel":
		j, ok := sbvResidentJob(m["job"])
		if !ok || !sbvRecoveryFields(m, "protocol", "engine_version", "directory", "instance_id", "job") || j["job_id"] != p["job_id"] || (j["state"] == "running" && j["cancellation_requested"] != true) {
			return bad()
		}
	case "dataset_jobs":
		if !sbvRecoveryFields(m, "protocol", "engine_version", "directory", "instance_id", "jobs", "next_after_sequence", "highest_sequence", "more") {
			return bad()
		}
		jobs, ok := m["jobs"].([]any)
		after, aok := sbvResidentUint(p["after_sequence"])
		limit, lok := sbvResidentUint(p["limit"])
		next, nok := sbvResidentUint(m["next_after_sequence"])
		highest, hok := sbvResidentUint(m["highest_sequence"])
		more, mok := m["more"].(bool)
		if !ok || !aok || !lok || !nok || !hok || !mok || limit == 0 || uint64(len(jobs)) > limit || (more && len(jobs) == 0) {
			return bad()
		}
		seen := map[string]bool{}
		for _, value := range jobs {
			j, valid := sbvResidentJob(value)
			if !valid {
				return bad()
			}
			seq, _ := sbvResidentUint(j["sequence"])
			id := j["job_id"].(string)
			if seen[id] || seq <= after || seq > highest {
				return bad()
			}
			seen[id] = true
			after = seq
			if p["job_id"] != nil && j["job_id"] != p["job_id"] {
				return bad()
			}
		}
		if next != after || (more && next >= highest) || (p["job_id"] != nil && (len(jobs) != 1 || more)) {
			return bad()
		}
		if p["job_id"] == nil && !more && highest > next {
			return bad()
		}
	default:
		return bad()
	}
	return nil
}
