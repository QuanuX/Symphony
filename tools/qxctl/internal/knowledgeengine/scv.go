package knowledgeengine

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
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
	if !SCVOperationSupported(version, operation) {
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
		expectedCount := 13
		if version == "0.2.0-dev" {
			expectedCount = 17
		} else if version == "0.3.0-dev" {
			expectedCount = 20
		} else if version == "0.4.0-dev" {
			expectedCount = 21
		}
		if !ok || len(operations) != expectedCount {
			return Response{}, fmt.Errorf("SCV descriptor operation set mismatch")
		}
		seen := map[string]bool{}
		for _, entry := range operations {
			item, ok := entry.(map[string]any)
			if !ok {
				return Response{}, fmt.Errorf("invalid SCV descriptor operation")
			}
			name, _ := item["operation_name"].(string)
			if !SCVOperationSupported(version, name) || seen[name] {
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
		"capture_index": "symphony.scv.capture-index.v1", "corpus_build": "symphony.scv.corpus.v1",
		"corpus_query": "symphony.scv.corpus-query.v1", "corpus_diff": "symphony.scv.corpus-diff.v1",
		"profile_prepare":     "symphony.scv.interpretation-profile.v1",
		"provider_interpret":  "symphony.scv.provider-interpretation.v1",
		"connection_evaluate": "symphony.scv.connection-evaluation.v1",
		"connection_reassess": "symphony.scv.connection-reassessment.v1",
	}[operation]
	return protocol, ok
}

// SCVOperationSupported preserves each exact package's finite operation set.
// Future versions require an explicit consumer update, never a latest alias.
func SCVOperationSupported(version, operation string) bool {
	if version != "0.1.0-dev" && version != "0.2.0-dev" && version != "0.3.0-dev" && version != "0.4.0-dev" {
		return false
	}
	if _, ok := SCVResultProtocol(operation); !ok {
		return false
	}
	if operation == "profile_prepare" {
		return version == "0.4.0-dev"
	}
	if operation == "provider_interpret" || strings.HasPrefix(operation, "connection_") {
		return version == "0.3.0-dev" || version == "0.4.0-dev"
	}
	return version != "0.1.0-dev" || (operation != "capture_index" && !strings.HasPrefix(operation, "corpus_"))
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
	if operation == "profile_prepare" {
		return validateSCVProfilePreparation(payload, value)
	}
	if operation == "provider_interpret" || strings.HasPrefix(operation, "connection_") {
		return validateSCVInterpretationResult(operation, payload, value)
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
		"capture_index":       {"protocol", "domain", "source_id", "provider_id", "family_id", "source_digest", "source_generation", "locator_id", "requested_uri", "capture_digest", "body_digest", "byte_size", "observed_at", "upstream_revision", "media_type", "completeness", "issues", "digest"},
		"corpus_build":        {"protocol", "domain", "corpus_id", "generation", "parent_digest", "snapshot_time", "members", "coverage", "digest"},
		"corpus_query":        {"protocol", "domain", "corpus_digest", "query_time", "selection", "max_age_seconds", "members", "coverage", "digest"},
		"corpus_diff":         {"protocol", "domain", "before_digest", "after_digest", "added_member_ids", "removed_member_ids", "changes", "affected_member_ids", "digest"},
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
	if strings.HasPrefix(operation, "corpus_") || operation == "capture_index" {
		if err := validateSCVCorpusResult(operation, payload, value); err != nil {
			return err
		}
	}
	return scvSeal(value, "digest")
}

func validateSCVCorpusResult(operation string, input, value map[string]any) error {
	switch operation {
	case "capture_index":
		capture, ok := input["capture"].(map[string]any)
		if !ok {
			return fmt.Errorf("capture index missing input capture")
		}
		source, ok := capture["source"].(map[string]any)
		if !ok {
			return fmt.Errorf("capture index missing source")
		}
		for _, key := range []string{"source_id", "provider_id", "family_id"} {
			if !scvEqual(value[key], source[key]) {
				return fmt.Errorf("capture index changed %s", key)
			}
		}
		for _, key := range []string{"locator_id", "body_digest", "byte_size", "observed_at", "upstream_revision", "media_type", "completeness", "issues"} {
			if !scvEqual(value[key], capture[key]) {
				return fmt.Errorf("capture index changed %s", key)
			}
		}
		if !scvEqual(value["source_digest"], source["digest"]) || !scvEqual(value["source_generation"], source["generation"]) || !scvEqual(value["capture_digest"], capture["digest"]) {
			return fmt.Errorf("capture index revision mismatch")
		}
		locators, ok := source["locators"].([]any)
		if !ok {
			return fmt.Errorf("capture index source locators missing")
		}
		found := false
		for _, raw := range locators {
			locator, ok := raw.(map[string]any)
			if ok && scvEqual(locator["locator_id"], capture["locator_id"]) {
				found = scvEqual(value["requested_uri"], locator["uri"])
			}
		}
		if !found {
			return fmt.Errorf("capture index changed requested URI")
		}
	case "corpus_build":
		if !scvEqual(value["corpus_id"], input["corpus_id"]) || !scvEqual(value["snapshot_time"], input["snapshot_time"]) {
			return fmt.Errorf("corpus snapshot changed request")
		}
		var parent any
		generation := int64(1)
		previousMembers := map[string]map[string]any{}
		if input["previous"] != nil {
			previous, old, err := scvCorpusSnapshot(input["previous"])
			if err != nil {
				return err
			}
			if !scvEqual(previous["corpus_id"], input["corpus_id"]) || !scvEqual(previous["domain"], value["domain"]) {
				return fmt.Errorf("corpus predecessor owner/identity mismatch")
			}
			n, err := scvCorpusInteger(previous["generation"], 1, 9007199254740990)
			if err != nil {
				return err
			}
			parent, generation, previousMembers = previous["digest"], n+1, old
		}
		if !scvEqual(value["parent_digest"], parent) || !scvEqual(value["generation"], generation) {
			return fmt.Errorf("corpus predecessor mismatch")
		}
		attempts, ok := input["attempts"].([]any)
		if !ok || len(attempts) > 128 {
			return fmt.Errorf("invalid corpus attempts")
		}
		expected := map[string]map[string]any{}
		for _, raw := range attempts {
			item, ok := raw.(map[string]any)
			if !ok || !scvCorpusFields(item, "member_id", "capture") {
				return fmt.Errorf("invalid corpus attempt fields")
			}
			id, err := scvCorpusID(item["member_id"])
			if err != nil || expected[id] != nil {
				return fmt.Errorf("invalid or duplicate corpus attempt identity")
			}
			index, err := scvCorpusIndex(item["capture"])
			if err != nil {
				return err
			}
			var last any
			if previous, exists := previousMembers[id]; exists {
				if !scvEqual(scvCorpusTuple(previous["latest_attempt"].(map[string]any)), scvCorpusTuple(index)) {
					return fmt.Errorf("corpus member identity rebound")
				}
				last = previous["last_complete"]
			}
			if index["completeness"] == "complete" {
				last = index
			}
			expected[id] = map[string]any{"member_id": id, "latest_attempt": index, "last_complete": last}
		}
		actual, err := scvCorpusMembers(value["members"])
		if err != nil {
			return err
		}
		if !scvEqual(actual, expected) || !scvEqual(value["coverage"], scvCorpusCoverage(expected)) {
			return fmt.Errorf("corpus changed attempt, retained complete evidence or coverage")
		}
	case "corpus_query":
		corpus, available, err := scvCorpusSnapshot(input["corpus"])
		if err != nil {
			return err
		}
		if !scvEqual(value["corpus_digest"], corpus["digest"]) || !scvEqual(value["domain"], corpus["domain"]) {
			return fmt.Errorf("corpus query revision or owner mismatch")
		}
		for _, key := range []string{"query_time", "selection", "max_age_seconds"} {
			if !scvEqual(value[key], input[key]) {
				return fmt.Errorf("corpus query changed %s", key)
			}
		}
		selection, ok := input["selection"].(string)
		if !ok || (selection != "latest_attempt" && selection != "last_complete") {
			return fmt.Errorf("invalid corpus query selection")
		}
		queryTime, err := scvCorpusTime(input["query_time"])
		if err != nil {
			return err
		}
		var maximumAge int64
		if input["max_age_seconds"] != nil {
			maximumAge, err = scvCorpusInteger(input["max_age_seconds"], 0, 3155760000)
			if err != nil {
				return err
			}
		}
		ids, ok := input["member_ids"].([]any)
		if !ok || len(ids) > 128 {
			return fmt.Errorf("invalid requested corpus members")
		}
		requested := map[string]map[string]any{}
		for _, raw := range ids {
			id, err := scvCorpusID(raw)
			if err != nil || available[id] == nil || requested[id] != nil {
				return fmt.Errorf("unknown or duplicate requested corpus member")
			}
			requested[id] = available[id]
		}
		if len(ids) == 0 {
			requested = available
		}
		members, ok := value["members"].([]any)
		if !ok || len(members) != len(requested) {
			return fmt.Errorf("corpus query changed requested member count")
		}
		lastID := ""
		for _, raw := range members {
			member, ok := raw.(map[string]any)
			if !ok || !scvCorpusFields(member, "member_id", "latest_attempt_digest", "selected", "status", "freshness", "source_revision_matches_latest", "reasons") {
				return fmt.Errorf("invalid corpus query member fields")
			}
			id, err := scvCorpusID(member["member_id"])
			if err != nil || id <= lastID || requested[id] == nil {
				return fmt.Errorf("corpus query changed requested member identity/order")
			}
			lastID = id
			original := requested[id]
			latest := original["latest_attempt"].(map[string]any)
			selected := original[selection]
			if !scvEqual(member["latest_attempt_digest"], latest["capture_digest"]) || !scvEqual(member["selected"], selected) {
				return fmt.Errorf("corpus query substituted selected evidence")
			}
			status, freshness := "unavailable", "not_selected"
			var sameSource any
			if selected != nil {
				index := selected.(map[string]any)
				status = index["completeness"].(string)
				sameSource = scvEqual(index["source_digest"], latest["source_digest"])
				observed, err := scvCorpusTime(index["observed_at"])
				if err != nil {
					return err
				}
				freshness = "current"
				if observed > queryTime {
					freshness = "future"
				} else if input["max_age_seconds"] != nil && queryTime-observed > maximumAge {
					freshness = "expired"
				}
			}
			if member["status"] != status || member["freshness"] != freshness || !scvEqual(member["source_revision_matches_latest"], sameSource) {
				return fmt.Errorf("corpus query changed acquisition status, freshness or source attribution")
			}
			reasons, ok := member["reasons"].([]any)
			if !ok || len(reasons) > 64 {
				return fmt.Errorf("invalid corpus query reasons")
			}
			for _, raw := range reasons {
				reason, ok := raw.(string)
				if !ok || len(reason) == 0 || len(reason) > 4096 {
					return fmt.Errorf("invalid corpus query reason")
				}
			}
		}
		if !scvEqual(value["coverage"], scvCorpusCoverage(requested)) {
			return fmt.Errorf("corpus query changed requested coverage")
		}
	case "corpus_diff":
		before, old, err := scvCorpusSnapshot(input["before"])
		if err != nil {
			return err
		}
		after, current, err := scvCorpusSnapshot(input["after"])
		if err != nil {
			return err
		}
		if !scvEqual(before["corpus_id"], after["corpus_id"]) || !scvEqual(before["domain"], after["domain"]) ||
			!scvEqual(value["domain"], before["domain"]) || !scvEqual(value["before_digest"], before["digest"]) || !scvEqual(value["after_digest"], after["digest"]) {
			return fmt.Errorf("corpus diff changed owner or revision binding")
		}
		added, removed, changes := []any{}, []any{}, []any{}
		affected := map[string]map[string]any{}
		for _, id := range scvCorpusKeys(old) {
			if current[id] == nil {
				removed = append(removed, id)
				affected[id] = old[id]
			}
		}
		for _, id := range scvCorpusKeys(current) {
			member, prior := current[id], old[id]
			if prior == nil {
				added = append(added, id)
				affected[id] = member
				continue
			}
			a, b := prior["latest_attempt"].(map[string]any), member["latest_attempt"].(map[string]any)
			if !scvEqual(scvCorpusTuple(a), scvCorpusTuple(b)) {
				return fmt.Errorf("corpus diff member identity rebound")
			}
			body := !scvEqual(a["body_digest"], b["body_digest"]) || !scvEqual(a["byte_size"], b["byte_size"])
			source := !scvEqual(a["source_digest"], b["source_digest"])
			observation := !scvEqual(a["capture_digest"], b["capture_digest"]) || !scvEqual(a["domain"], b["domain"])
			coverage := !scvEqual(a["completeness"], b["completeness"]) || !scvEqual(a["issues"], b["issues"])
			last := !scvEqual(prior["last_complete"], member["last_complete"])
			if body || source || observation || coverage || last {
				changes = append(changes, map[string]any{"member_id": id, "body_changed": body, "source_changed": source,
					"observation_changed": observation, "coverage_changed": coverage, "last_complete_changed": last})
				affected[id] = member
			}
		}
		if !scvEqual(value["added_member_ids"], added) || !scvEqual(value["removed_member_ids"], removed) ||
			!scvEqual(value["changes"], changes) || !scvEqual(value["affected_member_ids"], scvCorpusKeys(affected)) {
			return fmt.Errorf("corpus diff changed exact member/change attribution")
		}
	}
	return nil
}

// These helpers bind observable results to exact submitted evidence. They do
// not interpret provider prose, grant authority or choose a different corpus.
func scvCorpusFields(value map[string]any, fields ...string) bool {
	if len(value) != len(fields) {
		return false
	}
	for _, key := range fields {
		if _, ok := value[key]; !ok {
			return false
		}
	}
	return true
}
func scvCorpusID(value any) (string, error) {
	id, ok := value.(string)
	if !ok || len(id) == 0 || len(id) > 128 {
		return "", fmt.Errorf("invalid corpus identity")
	}
	for _, c := range []byte(id) {
		if c < 32 || c == 127 {
			return "", fmt.Errorf("control character in corpus identity")
		}
	}
	return id, nil
}
func scvCorpusInteger(value any, minimum, maximum int64) (int64, error) {
	number, ok := value.(json.Number)
	if !ok {
		return 0, fmt.Errorf("expected corpus integer")
	}
	n, err := number.Int64()
	if err != nil || n < minimum || n > maximum {
		return 0, fmt.Errorf("corpus integer outside bounds")
	}
	return n, nil
}
func scvCorpusTime(value any) (int64, error) {
	text, ok := value.(string)
	if !ok || len(text) != 20 {
		return 0, fmt.Errorf("invalid corpus UTC observation")
	}
	timestamp, err := time.Parse("2006-01-02T15:04:05Z", text)
	if err != nil || timestamp.Format("2006-01-02T15:04:05Z") != text {
		return 0, fmt.Errorf("invalid corpus UTC observation")
	}
	return timestamp.Unix(), nil
}
func scvCorpusIndex(raw any) (map[string]any, error) {
	index, ok := raw.(map[string]any)
	if !ok || !scvCorpusFields(index, "protocol", "domain", "source_id", "provider_id", "family_id", "source_digest", "source_generation",
		"locator_id", "requested_uri", "capture_digest", "body_digest", "byte_size", "observed_at", "upstream_revision", "media_type", "completeness", "issues", "digest") ||
		index["protocol"] != "symphony.scv.capture-index.v1" {
		return nil, fmt.Errorf("invalid corpus index fields")
	}
	for _, field := range []string{"capture_digest", "source_digest", "body_digest"} {
		digest, ok := index[field].(string)
		if !ok || !taggedSHA256(digest) {
			return nil, fmt.Errorf("invalid corpus index reference")
		}
	}
	completeness, ok := index["completeness"].(string)
	if !ok || (completeness != "complete" && completeness != "partial" && completeness != "failed") {
		return nil, fmt.Errorf("invalid corpus index completeness")
	}
	if err := scvSeal(index, "digest"); err != nil {
		return nil, err
	}
	return index, nil
}
func scvCorpusTuple(index map[string]any) any {
	return []any{index["family_id"], index["provider_id"], index["source_id"], index["locator_id"]}
}
func scvCorpusMembers(raw any) (map[string]map[string]any, error) {
	values, ok := raw.([]any)
	if !ok || len(values) > 128 {
		return nil, fmt.Errorf("invalid corpus member count")
	}
	result := map[string]map[string]any{}
	lastID := ""
	for _, raw := range values {
		member, ok := raw.(map[string]any)
		if !ok || !scvCorpusFields(member, "member_id", "latest_attempt", "last_complete") {
			return nil, fmt.Errorf("invalid corpus member fields")
		}
		id, err := scvCorpusID(member["member_id"])
		if err != nil || id <= lastID {
			return nil, fmt.Errorf("corpus member identities not unique/sorted")
		}
		lastID = id
		latest, err := scvCorpusIndex(member["latest_attempt"])
		if err != nil {
			return nil, err
		}
		if member["last_complete"] != nil {
			last, err := scvCorpusIndex(member["last_complete"])
			if err != nil {
				return nil, err
			}
			if last["completeness"] != "complete" || !scvEqual(scvCorpusTuple(last), scvCorpusTuple(latest)) {
				return nil, fmt.Errorf("invalid retained complete corpus index")
			}
		}
		if latest["completeness"] == "complete" && !scvEqual(member["last_complete"], latest) {
			return nil, fmt.Errorf("complete latest attempt differs from last complete")
		}
		result[id] = member
	}
	return result, nil
}
func scvCorpusSnapshot(raw any) (map[string]any, map[string]map[string]any, error) {
	corpus, ok := raw.(map[string]any)
	if !ok || !scvCorpusFields(corpus, "protocol", "domain", "corpus_id", "generation", "parent_digest", "snapshot_time", "members", "coverage", "digest") ||
		corpus["protocol"] != "symphony.scv.corpus.v1" {
		return nil, nil, fmt.Errorf("invalid corpus snapshot fields")
	}
	if err := scvSeal(corpus, "digest"); err != nil {
		return nil, nil, err
	}
	members, err := scvCorpusMembers(corpus["members"])
	if err != nil {
		return nil, nil, err
	}
	if !scvEqual(corpus["coverage"], scvCorpusCoverage(members)) {
		return nil, nil, fmt.Errorf("invalid corpus coverage attribution")
	}
	return corpus, members, nil
}
func scvCorpusCoverage(members map[string]map[string]any) map[string]any {
	complete, partial, failed, retained := 0, 0, 0, 0
	for _, member := range members {
		latest := member["latest_attempt"].(map[string]any)
		switch latest["completeness"] {
		case "complete":
			complete++
		case "partial":
			partial++
		case "failed":
			failed++
		}
		if latest["completeness"] != "complete" && member["last_complete"] != nil {
			retained++
		}
	}
	return map[string]any{"requested_members": len(members), "complete_attempts": complete, "partial_attempts": partial,
		"failed_attempts": failed, "retained_complete_members": retained}
}
func scvCorpusKeys(members map[string]map[string]any) []string {
	ids := make([]string, 0, len(members))
	for id := range members {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
