package sovremote

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
)

// Template contains deliberately synthetic references, not a discovered host.
func Template(operation string) (any, error) {
	d := strings.Repeat("0", 64)
	s := Selection{Protocol: PlanProtocol, OperationID: "example-operation", Target: Target{"example-tops", "example-node", "example-habitat", d}, Transport: Transport{"controller", "ipc", "example-adapter", "0.1.0", d, "example-endpoint-reference"}, Runtime: Runtime{"0.1.0-dev", d, knowledgeengine.SBVAdministrationInterfaceVersion, d}, Job: Job{"/example/private-job", "example-job", d}, Resources: Resources{Devices: []string{}}, Authority: Authority{Operation: "example-operation", Resource: "example-resource", Audience: "example-audience", Scope: "example-scope"}, Extensions: map[string]string{}}
	if operation == "plan" {
		return s, nil
	}
	if operation == "assess" {
		p, err := build(s)
		if err != nil {
			return nil, err
		}
		return AssessmentInput{Protocol: AssessmentProtocol, Selection: s, ExpectedPlanSHA256: p.PlanSHA256, TransportOutcome: "response_lost", Observation: json.RawMessage("null")}, nil
	}
	if operation == "invoke" {
		p, err := build(s)
		if err != nil {
			return nil, err
		}
		raw, _ := json.Marshal(map[string]any{"protocol": "symphony.sbv.job-inspect-input.v1", "directory": s.Job.Directory, "job_id": s.Job.ID, "expected_plan_sha256": s.Job.PlanSHA256})
		return Invocation{Protocol: "symphony.sov.remote-invoke-input.v1", Selection: s, ExpectedPlanSHA256: p.PlanSHA256, RequestID: "00000000-0000-4000-8000-000000000001", CorrelationID: "00000000-0000-4000-8000-000000000002", Operation: "job_inspect", Request: raw, Adapter: Executable{"/example/sov-stdio-adapter", d}, Endpoint: Endpoint{Executable{"/example/qxctl", d}, "/example/private-sov/configuration.json", d}}, nil
	}
	if operation == "target_configure" {
		return ConfigureInput{Protocol: "symphony.sov.target-configure-input.v1", Path: "/example/private-sov/configuration.json", Configuration: Configuration{Protocol: "symphony.sov.target-configuration.v1", Target: s.Target, Job: s.Job, Runtime: s.Runtime, SBVPrefix: "/example/sbv-prefix", AuthorityMode: "owner_filesystem", Authority: s.Authority, AllowedArtifacts: []Artifact{}}}, nil
	}
	return nil, fmt.Errorf("unknown SOV remote operation")
}

// Schema describes the complete control shape from the actual typed contract.
// Cross-field digest, native observation and identity checks remain normative.
func Schema(operation string) (any, error) {
	var in, out any
	switch operation {
	case "plan":
		in, out = Selection{}, Plan{}
	case "assess":
		in, out = AssessmentInput{}, Assessment{}
	case "invoke":
		in, out = Invocation{}, InvocationResult{}
	case "target_configure":
		in, out = ConfigureInput{}, ConfigurationResult{}
	default:
		return nil, fmt.Errorf("unknown SOV remote operation")
	}
	return map[string]any{"$schema": "https://json-schema.org/draft/2020-12/schema", "request": shape(reflect.TypeOf(in), ""), "result": shape(reflect.TypeOf(out), ""), "semantic_contract": "knowledge/sov/REMOTE-HANDOFF.md", "control_document_max_bytes": MaxDocumentBytes}, nil
}
func shape(t reflect.Type, name string) map[string]any {
	if t == reflect.TypeOf(json.RawMessage{}) {
		return map[string]any{"type": []string{"object", "null"}, "description": "Complete native request/result for the selected operation and exact SBV version; native owner schemas and cross-field validation are normative. Assessment observation is null or job-inspect.v1. Invocation requests cannot be null."}
	}
	if t.Kind() == reflect.Pointer {
		return map[string]any{"anyOf": []any{shape(t.Elem(), name), map[string]any{"type": "null"}}}
	}
	s := map[string]any{}
	switch t.Kind() {
	case reflect.Struct:
		properties := map[string]any{}
		required := []string{}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			key := f.Tag.Get("json")
			properties[key] = shape(f.Type, key)
			required = append(required, key)
		}
		s = map[string]any{"type": "object", "additionalProperties": false, "properties": properties, "required": required}
		protocol := map[reflect.Type]string{reflect.TypeOf(Selection{}): PlanProtocol, reflect.TypeOf(Plan{}): "symphony.sov.remote-plan.v1", reflect.TypeOf(AssessmentInput{}): AssessmentProtocol, reflect.TypeOf(Assessment{}): "symphony.sov.remote-assess.v1", reflect.TypeOf(Invocation{}): "symphony.sov.remote-invoke-input.v1", reflect.TypeOf(InvocationResult{}): "symphony.sov.remote-invoke.v1", reflect.TypeOf(TargetResult{}): "symphony.sov.target-result.v1", reflect.TypeOf(Configuration{}): "symphony.sov.target-configuration.v1", reflect.TypeOf(ConfigureInput{}): "symphony.sov.target-configure-input.v1", reflect.TypeOf(ConfigurationResult{}): "symphony.sov.target-configuration-result.v1"}[t]
		if protocol != "" {
			properties["protocol"] = map[string]any{"const": protocol}
		}
		if t == reflect.TypeOf(TargetResult{}) {
			properties["status"] = map[string]any{"enum": []string{"completed", "native_error"}}
		}
		if t == reflect.TypeOf(Invocation{}) || t == reflect.TypeOf(TargetResult{}) {
			properties["operation"] = map[string]any{"enum": []string{"job_inspect", "job_execute", "job_cancel", "job_reconcile", "result_inspect", "result_query", "bundle_inspect", "bundle_query", "bundle_verify"}}
		}
		if t == reflect.TypeOf(Invocation{}) {
			properties["request"] = map[string]any{"type": "object"}
		}
	case reflect.String:
		s["type"] = "string"
		if name != "" {
			s["minLength"] = 1
		}
		switch {
		case strings.HasSuffix(name, "sha256"):
			s["pattern"] = sha.String()
		case name == "threads" || name == "memory_bytes" || name == "deadline_unix_ms" || name == "authorization_window_seconds":
			s["pattern"] = positive.String()
		case name == "job_id":
			s["pattern"] = jobID.String()
		}
		switch name {
		case "sbv_version":
			s["const"] = knowledgeengine.SBVAdministrationInterfaceVersion
		case "authority_mode":
			s["enum"] = []string{"owner_filesystem", "ssiag"}
		case "origin_verification":
			s["const"] = "selected_adapter_assertion"
		case "request_id", "correlation_id":
			s["pattern"] = uuid.String()
		case "arrangement":
			s["enum"] = []string{"controller", "target", "staged"}
		case "family":
			s["enum"] = []string{"ipc", "scoped_cli"}
		case "transport_outcome":
			s["enum"] = []string{"not_dispatched", "response_lost", "response_received"}
		case "status":
			s["const"] = "planned_not_dispatched"
		case "evidence_status":
			s["const"] = "supplied_unverified"
		case "recommended_action":
			s["enum"] = []string{"qualify_target_and_authority", "inspect_same_job", "observe_same_job", "inspect_bound_result", "reconcile_same_job"}
		}
	case reflect.Bool:
		s = map[string]any{"type": "boolean"}
		if name != "dispatch_started" {
			s = map[string]any{"const": false}
		}
	case reflect.Slice:
		s = map[string]any{"type": "array", "items": shape(t.Elem(), "")}
		if name == "device_refs" {
			s["uniqueItems"] = true
			s["items"] = map[string]any{"type": "string", "minLength": 1}
		}
	case reflect.Map:
		s = map[string]any{"type": "object", "additionalProperties": shape(t.Elem(), "")}
	}
	return s
}
