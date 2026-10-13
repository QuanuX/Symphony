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
	default:
		return nil, fmt.Errorf("unknown SOV remote operation")
	}
	return map[string]any{"$schema": "https://json-schema.org/draft/2020-12/schema", "request": shape(reflect.TypeOf(in), ""), "result": shape(reflect.TypeOf(out), ""), "semantic_contract": "knowledge/sov/REMOTE-HANDOFF.md", "control_document_max_bytes": MaxDocumentBytes}, nil
}
func shape(t reflect.Type, name string) map[string]any {
	if t == reflect.TypeOf(json.RawMessage{}) {
		return map[string]any{"type": []string{"object", "null"}, "description": "Null or the complete native symphony.sbv.job-inspect.v1 status for the selected exact SBV version; validated by the SBV owner contract, never authenticated by this planner."}
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
		protocol := map[reflect.Type]string{reflect.TypeOf(Selection{}): PlanProtocol, reflect.TypeOf(Plan{}): "symphony.sov.remote-plan.v1", reflect.TypeOf(AssessmentInput{}): AssessmentProtocol, reflect.TypeOf(Assessment{}): "symphony.sov.remote-assess.v1"}[t]
		if protocol != "" {
			properties["protocol"] = map[string]any{"const": protocol}
		}
	case reflect.String:
		s["type"] = "string"
		if name != "" {
			s["minLength"] = 1
		}
		switch {
		case strings.HasSuffix(name, "sha256"):
			s["pattern"] = sha.String()
		case name == "threads" || name == "memory_bytes" || name == "deadline_unix_ms":
			s["pattern"] = positive.String()
		case name == "job_id":
			s["pattern"] = jobID.String()
		}
		switch name {
		case "sbv_version":
			s["const"] = knowledgeengine.SBVAdministrationInterfaceVersion
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
		s = map[string]any{"const": false}
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
