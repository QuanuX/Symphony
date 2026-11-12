package transportsession

import (
	profile "github.com/QuanuX/Symphony/tools/qxctl/internal/sshprofile"
	"reflect"
	"strings"
)

// Shape projection is discovery only; current profile, identity, plan and clock
// checks in the native owner and protected carrier remain normative.
func Schema() any {
	return map[string]any{"protocol": "symphony.sov.transport-session.schemas.v1", "$schema": "https://json-schema.org/draft/2020-12/schema", "semantic_contract": "knowledge/sov/CLOUDFLARE-ACCESS-SESSIONS.md", "control_document_max_bytes": 65536, "request": shape(reflect.TypeOf(Request{})), "native_request": shape(reflect.TypeOf(NativeRequest{})), "result": shape(reflect.TypeOf(Result{}))}
}
func shape(t reflect.Type) map[string]any {
	if t.Kind() == reflect.Pointer {
		return map[string]any{"anyOf": []any{shape(t.Elem()), map[string]any{"type": "null"}}}
	}
	switch t.Kind() {
	case reflect.Struct:
		p := map[string]any{}
		required := []string{}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			key := strings.Split(f.Tag.Get("json"), ",")[0]
			p[key] = shape(f.Type)
			required = append(required, key)
		}
		protocol := map[reflect.Type]string{reflect.TypeOf(Request{}): RequestProtocol, reflect.TypeOf(NativeRequest{}): NativeProtocol, reflect.TypeOf(Result{}): ResultProtocol, reflect.TypeOf(profile.Endpoint{}): profile.EndpointProtocol}[t]
		if protocol != "" {
			p["protocol"] = map[string]any{"const": protocol}
		}
		if t == reflect.TypeOf(Request{}) || t == reflect.TypeOf(NativeRequest{}) {
			p["operation"] = map[string]any{"enum": Operations}
		}
		if t == reflect.TypeOf(Result{}) {
			p["provider_authorization"] = map[string]any{"const": "not_proven"}
		}
		return map[string]any{"type": "object", "properties": p, "required": required, "additionalProperties": false}
	case reflect.Slice:
		return map[string]any{"type": "array", "items": shape(t.Elem())}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int64, reflect.Uint64:
		return map[string]any{"type": "integer"}
	default:
		return map[string]any{"type": "string"}
	}
}
func Template(endpoint profile.Endpoint) Request {
	return Request{RequestProtocol, "77777777-7777-4777-8777-777777777777", "inspect", endpoint, nil, nil, "1"}
}
