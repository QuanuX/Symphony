package knowledgeengine

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

// SCVDomains is the finite installed surface of this release, not a provider
// admission rule. Every invocation selects one exact domain and version.
func SCVDomains() []string {
	return []string{"scv", "schv", "scev", "schv-aws", "schv-azure", "schv-do", "schv-gcp", "scev-cf"}
}

func scvSpec(domain string) (engineSpec, error) {
	for _, allowed := range SCVDomains() {
		if domain == allowed {
			return engineSpec{label: domain, moduleID: domain + "-engine", engineID: "symphony-" + domain,
				componentKind: "vector_engine", vectorID: domain, processProtocol: processProtocol,
				requiredReceptors: []string{"symphony.maestro.knowledge-engine.v1"}}, nil
		}
	}
	return engineSpec{}, fmt.Errorf("unsupported SCV domain %q", domain)
}

// InspectSCVDomain deliberately accepts receipt v2 only. Legacy receipt and
// engine-binding role vocabularies remain unchanged.
func InspectSCVDomain(domain, prefix, version string) (Installation, error) {
	spec, err := scvSpec(domain)
	if err != nil {
		return Installation{}, err
	}
	entry, err := InspectReceiptV2EntryPoint(prefix, version, ReceiptV2EntryPointSpec{
		Label: domain, ComponentID: spec.moduleID, ComponentKind: spec.componentKind,
		ModuleID: spec.moduleID, PackageID: spec.moduleID, VectorID: &spec.vectorID, EngineID: &spec.engineID,
		EntryPointID: spec.engineID, EntryPointKind: "executable",
		EntryPointRelativePath: filepath.ToSlash(filepath.Join("libexec", "symphony", spec.moduleID, version, spec.engineID)),
		RequiredProtocols:      []string{processProtocol}, RequiredReceptors: spec.requiredReceptors,
	})
	if err != nil {
		return Installation{}, err
	}
	return Installation{Role: domain, ModuleID: spec.moduleID, EngineID: spec.engineID, Version: version,
		Prefix: entry.Prefix, ReceiptPath: entry.ReceiptPath, ReceiptDigest: entry.ReceiptDigest,
		ReceiptProtocol: receiptProtocolV2, ExecutablePath: entry.ExecutablePath, ExecutableDigest: entry.ExecutableDigest}, nil
}

func InvokeSCVDomain(ctx context.Context, domain, prefix, version, cwd, operation string, payload []byte) (Response, error) {
	spec, err := scvSpec(domain)
	if err != nil {
		return Response{}, err
	}
	if _, ok := SCVResultProtocol(operation); !ok {
		return Response{}, fmt.Errorf("unsupported SCV operation")
	}
	if err := validateJSONObject(payload, maxRequestBytes); err != nil {
		return Response{}, err
	}
	installed, err := InspectSCVDomain(domain, prefix, version)
	if err != nil {
		return Response{}, err
	}
	response, err := invokeResolved(ctx, spec, installed.ExecutablePath, version, cwd, operation, payload)
	if err != nil {
		return response, err
	}
	if err := ValidateSCVResult(operation, payload, response.Result); err != nil {
		return Response{}, err
	}
	value, _ := scvObject(response.Result)
	if operation == "inspect" {
		if value["module_id"] != spec.moduleID || value["engine_id"] != spec.engineID || value["vector_id"] != domain || value["engine_version"] != version {
			return Response{}, fmt.Errorf("SCV descriptor installation identity mismatch")
		}
		operations, ok := value["operations"].([]any)
		if !ok || len(operations) != 13 {
			return Response{}, fmt.Errorf("SCV descriptor operation set mismatch")
		}
		seen := map[string]bool{}
		for _, entry := range operations {
			item, ok := entry.(map[string]any)
			if !ok {
				return Response{}, fmt.Errorf("invalid SCV descriptor operation")
			}
			name, _ := item["operation_name"].(string)
			if _, ok := SCVResultProtocol(name); !ok || seen[name] {
				return Response{}, fmt.Errorf("SCV descriptor operation identity mismatch")
			}
			seen[name] = true
			if item["engine_operation_id"] != "engop:symphony:"+domain+"."+strings.ReplaceAll(name, "_", ".") {
				return Response{}, fmt.Errorf("SCV descriptor backend identity mismatch")
			}
		}
	} else if _, hasDomain := value["domain"]; hasDomain && value["domain"] != domain {
		return Response{}, fmt.Errorf("SCV result domain mismatch")
	}
	return response, nil
}

func SCVResultProtocol(operation string) (string, bool) {
	protocol, ok := map[string]string{
		"inspect": "symphony.knowledge.engine-descriptor.v2", "provider_onboard": "symphony.scv.provider.v1",
		"source_plan": "symphony.scv.source-plan.v1", "source_apply": "symphony.scv.source-transition.v1",
		"source_status": "symphony.scv.source-status.v1", "capture_import": "symphony.scv.capture.v1",
		"capture_compare": "symphony.scv.capture-diff.v1", "knowledge_interpret": "symphony.scv.knowledge.v1",
		"graph_build": "symphony.scv.graph.v1", "graph_query": "symphony.scv.query-result.v1",
		"graph_evaluate": "symphony.scv.evaluate-result.v1", "graph_diff": "symphony.scv.diff-result.v1", "graph_explain": "symphony.scv.explain-result.v1",
	}[operation]
	return protocol, ok
}

// SCVCanonical encodes the same sorted, UTF-8 JSON subset as the C++ owner.
// Decode with UseNumber: exact integral evidence must not pass through float64.
func SCVCanonical(value any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	// encoding/json escapes these Unicode line separators even with EscapeHTML
	// disabled; nlohmann JSON emits their UTF-8 representation.
	data := bytes.TrimSuffix(buffer.Bytes(), []byte("\n"))
	var output bytes.Buffer
	for i := 0; i < len(data); i++ {
		if data[i] == '\\' && i+1 < len(data) {
			if i+6 <= len(data) && (string(data[i:i+6]) == `\u2028` || string(data[i:i+6]) == `\u2029`) {
				if data[i+5] == '8' {
					output.WriteString("\u2028")
				} else {
					output.WriteString("\u2029")
				}
				i += 5
				continue
			}
			output.WriteByte(data[i])
			i++
			output.WriteByte(data[i])
			continue
		}
		output.WriteByte(data[i])
	}
	return output.Bytes(), nil
}

func SCVDigest(value any) (string, error) {
	data, err := SCVCanonical(value)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(hash[:]), nil
}

func scvObject(raw []byte) (map[string]any, error) {
	if err := validateJSONObject(raw, maxResponseBytes); err != nil {
		return nil, err
	}
	var value map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
}

func scvSeal(value map[string]any, field string) error {
	expected, ok := value[field].(string)
	if !ok || !taggedSHA256(expected) {
		return fmt.Errorf("SCV result lacks %s", field)
	}
	body := make(map[string]any, len(value))
	for key, item := range value {
		if key != field {
			body[key] = item
		}
	}
	actual, err := SCVDigest(body)
	if err != nil || actual != expected {
		return fmt.Errorf("SCV result %s mismatch", field)
	}
	return nil
}

func scvEqual(a, b any) bool {
	x, ex := SCVCanonical(a)
	y, ey := SCVCanonical(b)
	return ex == nil && ey == nil && bytes.Equal(x, y)
}

// ValidateSCVResult verifies the consumer boundary. Domain semantics remain in
// C++; this checks result identity, exact source/capture fields, seals and the
// correspondence to the submitted request before printing or persisting it.
func ValidateSCVResult(operation string, input, raw []byte) error {
	value, err := scvObject(raw)
	if err != nil {
		return err
	}
	payload, err := scvObject(input)
	if err != nil {
		return err
	}
	expected, ok := SCVResultProtocol(operation)
	if !ok || value["protocol"] != expected {
		return fmt.Errorf("SCV result protocol does not match operation")
	}
	if operation == "inspect" {
		if err := requireExactFields(raw, []string{"protocol", "format_version", "module_id", "engine_id", "vector_id", "engine_version", "process_protocols", "contract_versions", "operations", "limits", "supported_scopes", "language", "thermal_path", "canonical_apply_enabled", "session_mutation_enabled", "network_listener", "descriptor_digest"}); err != nil {
			return err
		}
		if value["canonical_apply_enabled"] != false || value["session_mutation_enabled"] != false || value["network_listener"] != false {
			return fmt.Errorf("SCV descriptor broadens authority")
		}
		return scvSeal(value, "descriptor_digest")
	}
	exact := map[string][]string{
		"source_plan":         {"protocol", "operation_id", "expected_state_digest", "change_kind", "reason", "source", "plan_digest"},
		"source_apply":        {"protocol", "operation_id", "expected_state_digest", "state", "state_digest"},
		"source_status":       {"protocol", "source", "state_digest"},
		"capture_import":      {"protocol", "source", "locator_id", "resolved_uri", "redirects", "observed_at", "upstream_revision", "media_type", "body", "body_digest", "byte_size", "completeness", "issues", "digest"},
		"capture_compare":     {"protocol", "before_digest", "after_digest", "change_kind", "source_identity_preserved", "body_changed", "configuration_changed", "coverage_before", "coverage_after", "requires_reinterpretation", "digest"},
		"provider_onboard":    {"protocol", "provider_id", "family_id", "display_name", "sources", "disposition", "interpretation_scope", "digest"},
		"knowledge_interpret": {"protocol", "domain", "interpreter_version", "selection_policy", "captures", "claims", "native_nodes", "native_edges", "limitations", "digest"},
		"graph_build":         {"protocol", "domain", "selection_policy", "knowledge_digests", "interpretations", "captures", "claims", "native_nodes", "native_edges", "limitations", "digest"},
		"graph_query":         {"protocol", "domain", "graph_digest", "query_time", "selection_policy", "findings", "conflicts", "evidence_origins", "limitations", "coverage", "query_selection", "digest"},
		"graph_evaluate":      {"protocol", "domain", "graph_digest", "query_time", "selection_policy", "findings", "conflicts", "evidence_origins", "limitations", "coverage", "query_selection", "digest"},
		"graph_explain":       {"protocol", "domain", "graph_digest", "query_time", "selection_policy", "findings", "conflicts", "evidence_origins", "limitations", "coverage", "query_selection", "claim_id", "dependency_closure", "digest"},
		"graph_diff":          {"protocol", "domain", "before_digest", "after_digest", "query_time", "changes", "affected_claim_ids", "before_evaluation", "after_evaluation", "limitations", "digest"},
	}
	if names, ok := exact[operation]; ok {
		if operation == "graph_query" || operation == "graph_evaluate" || operation == "graph_explain" {
			findings, ok := value["findings"].([]any)
			if !ok {
				return fmt.Errorf("graph findings must be an array")
			}
			absence, present := value["absence"]
			if len(findings) == 0 {
				if !present || absence != "no matching claim in the bounded selected corpus; no availability conclusion" {
					return fmt.Errorf("empty graph selection lacks bounded absence qualification")
				}
				names = append(names, "absence")
			} else if present {
				return fmt.Errorf("nonempty graph selection has unexpected absence qualification")
			}
		}
		if len(value) != len(names) {
			return fmt.Errorf("unexpected SCV result fields")
		}
		for _, name := range names {
			if _, ok := value[name]; !ok {
				return fmt.Errorf("missing SCV result field %s", name)
			}
		}
	}
	if operation == "source_plan" {
		if err := scvSeal(value, "plan_digest"); err != nil {
			return err
		}
		if !scvEqual(value["operation_id"], payload["operation_id"]) || !scvEqual(value["reason"], payload["reason"]) {
			return fmt.Errorf("source plan does not bind request")
		}
		source, ok := value["source"].(map[string]any)
		if !ok {
			return fmt.Errorf("source plan lacks successor")
		}
		desired := map[string]any{}
		for key, item := range source {
			if key != "protocol" && key != "generation" && key != "predecessor_digest" && key != "digest" {
				desired[key] = item
			}
		}
		if !scvEqual(desired, payload["desired"]) {
			return fmt.Errorf("source plan changed desired fields")
		}
		var previous any
		if current, ok := payload["current"].(map[string]any); ok {
			previous = current["digest"]
		}
		if !scvEqual(value["expected_state_digest"], previous) {
			return fmt.Errorf("source plan changed expected state")
		}
	}
	if operation == "source_status" || operation == "source_apply" || operation == "source_plan" {
		key := "source"
		if operation == "source_apply" {
			key = "state"
		}
		source := value[key]
		if source != nil {
			object, ok := source.(map[string]any)
			if !ok || object["protocol"] != "symphony.scv.source.v1" {
				return fmt.Errorf("invalid source result")
			}
			if len(object) != 12 {
				return fmt.Errorf("source result has unexpected fields")
			}
			if err := scvSeal(object, "digest"); err != nil {
				return err
			}
			if operation != "source_plan" && !scvEqual(value["state_digest"], object["digest"]) {
				return fmt.Errorf("source state digest mismatch")
			}
		} else if operation != "source_status" || value["state_digest"] != nil {
			return fmt.Errorf("unexpected absent source")
		}
		if operation == "source_status" && !scvEqual(source, payload["source"]) {
			return fmt.Errorf("source status changed supplied state")
		}
		if operation == "source_apply" {
			plan, ok := payload["plan"].(map[string]any)
			if !ok || !scvEqual(value["state"], plan["source"]) || !scvEqual(value["operation_id"], plan["operation_id"]) || !scvEqual(value["expected_state_digest"], plan["expected_state_digest"]) {
				return fmt.Errorf("source transition does not bind exact plan")
			}
		}
		return nil
	}
	if operation == "capture_import" {
		for key, item := range payload {
			if !scvEqual(item, value[key]) {
				return fmt.Errorf("capture changed submitted %s", key)
			}
		}
		body, ok := value["body"].(string)
		if !ok || len(body) > 65536 {
			return fmt.Errorf("capture body exceeds bounds")
		}
		hash := sha256.Sum256([]byte(body))
		if value["body_digest"] != "sha256:"+hex.EncodeToString(hash[:]) || value["byte_size"] != json.Number(fmt.Sprint(len(body))) {
			return fmt.Errorf("capture body identity mismatch")
		}
	}
	if operation == "graph_query" || operation == "graph_evaluate" || operation == "graph_explain" {
		graph, ok := payload["graph"].(map[string]any)
		if !ok || !scvEqual(value["graph_digest"], graph["digest"]) || !scvEqual(value["query_time"], payload["query_time"]) || !scvEqual(value["selection_policy"], graph["selection_policy"]) {
			return fmt.Errorf("graph result changed request binding")
		}
		if operation == "graph_explain" && !scvEqual(value["claim_id"], payload["claim_id"]) {
			return fmt.Errorf("explanation changed requested claim")
		}
		if operation != "graph_explain" {
			selection := map[string]any{}
			for key, item := range payload {
				if key != "graph" {
					selection[key] = item
				}
			}
			if !scvEqual(selection, value["query_selection"]) {
				return fmt.Errorf("query selection mismatch")
			}
		}
	}
	if operation == "graph_diff" {
		before, bok := payload["before"].(map[string]any)
		after, aok := payload["after"].(map[string]any)
		if !bok || !aok || !scvEqual(value["before_digest"], before["digest"]) || !scvEqual(value["after_digest"], after["digest"]) || !scvEqual(value["query_time"], payload["query_time"]) {
			return fmt.Errorf("graph difference changed request binding")
		}
	}
	return scvSeal(value, "digest")
}
