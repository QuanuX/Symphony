#include "sav.hpp"

#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/limits.hpp"
#include "symphony/knowledge/engine/operation.hpp"
#include "symphony/knowledge/engine/temporal.hpp"

#include <algorithm>
#include <array>
#include <map>
#include <set>
#include <string>
#include <string_view>
#include <utility>
#include <vector>

namespace symphony::knowledge::sav {
namespace engine = symphony::knowledge::engine;

namespace {

constexpr std::size_t max_sources = 256U;
constexpr std::size_t max_references = 1024U;
constexpr std::size_t max_rule_sources = 64U;

[[noreturn]] void invalid(std::string code, std::string message) {
    throw engine::Error(std::move(code), std::move(message), 3);
}

void exact_fields(const engine::Json& value, std::initializer_list<std::string_view> fields,
                  std::string_view context) {
    if (!value.is_object()) invalid("sav.type", std::string(context) + " must be an object");
    std::set<std::string> expected;
    for (const auto field : fields) expected.emplace(field);
    if (value.size() != expected.size()) invalid("sav.fields", std::string(context) + " has an unexpected field set");
    for (const auto& field : expected) {
        if (!value.contains(field)) invalid("sav.fields", std::string(context) + " is missing " + field);
    }
}

const std::string& string_field(const engine::Json& value, std::string_view field,
                                std::size_t maximum = 4096U) {
    const auto key = std::string(field);
    if (!value.at(key).is_string()) invalid("sav.type", key + " must be a string");
    const auto& result = value.at(key).get_ref<const std::string&>();
    if (result.empty() || result.size() > maximum) invalid("sav.bounds", key + " is outside its bound");
    return result;
}

bool tagged_digest(std::string_view value) {
    if (value.size() != 71U || !value.starts_with("sha256:")) return false;
    return std::all_of(value.begin() + 7, value.end(), [](const unsigned char character) {
        return (character >= '0' && character <= '9') || (character >= 'a' && character <= 'f');
    });
}

bool safe_identity(std::string_view value) {
    return !value.empty() && value.size() <= 256U &&
           std::all_of(value.begin(), value.end(), [](const unsigned char c) {
               return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
                      (c >= '0' && c <= '9') || c == '.' || c == '_' || c == ':' ||
                      c == '/' || c == '+' || c == '-';
           });
}

void require_identity(const engine::Json& value, std::string_view field) {
    if (!safe_identity(string_field(value, field, 256U))) invalid("sav.identity", std::string(field) + " is invalid");
}

void require_digest(const engine::Json& value, std::string_view field) {
    if (!tagged_digest(string_field(value, field, 71U))) invalid("sav.digest", std::string(field) + " is invalid");
}

std::string digest_without(engine::Json value, std::string_view field) {
    value.erase(std::string(field));
    return engine::tagged_sha256(value.dump());
}

std::vector<std::string> string_array(const engine::Json& value, std::string_view field,
                                      std::size_t maximum, bool nonempty = false) {
    const auto& array = value.at(std::string(field));
    if (!array.is_array() || array.size() > maximum || (nonempty && array.empty())) {
        invalid("sav.bounds", std::string(field) + " must be a bounded array");
    }
    std::vector<std::string> result;
    for (const auto& entry : array) {
        if (!entry.is_string() || !safe_identity(entry.get_ref<const std::string&>())) {
            invalid("sav.identity", std::string(field) + " contains an invalid identity");
        }
        result.push_back(entry.get<std::string>());
    }
    if (!std::is_sorted(result.begin(), result.end()) ||
        std::adjacent_find(result.begin(), result.end()) != result.end()) {
        invalid("sav.order", std::string(field) + " must be sorted and unique");
    }
    return result;
}

void validate_source(const engine::Json& source) {
    exact_fields(source, {"source_id", "owner_vector", "owner_contract", "protocol", "authority_role",
        "collection_state", "content_digest", "observation_digest", "observed_at", "freshness", "payload"}, "source");
    require_identity(source, "source_id");
    require_identity(source, "owner_vector");
    require_identity(source, "protocol");
    string_field(source, "owner_contract");
    string_field(source, "authority_role", 64U);
    string_field(source, "collection_state", 32U);
    require_digest(source, "content_digest");
    if (!source.at("observation_digest").is_null()) require_digest(source, "observation_digest");
    if (!source.at("observed_at").is_null() &&
        (!source.at("observed_at").is_string() || !engine::is_utc_seconds(source.at("observed_at").get<std::string>()))) {
        invalid("sav.timestamp", "observed_at must be null or STSC whole-second UTC");
    }
    string_field(source, "freshness", 32U);
    if (source.at("content_digest").get<std::string>() != engine::tagged_sha256(source.at("payload").dump())) {
        invalid("sav.source_digest", "source content_digest does not bind payload");
    }
}

void validate_rule(const engine::Json& rule) {
    exact_fields(rule, {"kind", "source_ids", "pointer", "expected_json", "expected_digest"}, "rule");
    const auto kind = string_field(rule, "kind", 64U);
    const auto ids = string_array(rule, "source_ids", max_rule_sources);
    const auto none = kind == "always" || kind == "never";
    if (none && !ids.empty()) invalid("sav.rule", "constant rules cannot name sources");
    if (kind == "source_available" && ids.empty()) invalid("sav.rule", "source_available needs a source");
    if ((kind == "source_content_digest_equals" || kind == "source_payload_pointer_equals") && ids.size() != 1U) {
        invalid("sav.rule", "comparison rules require exactly one source");
    }
    const bool known = none || kind == "source_available" || kind == "source_content_digest_equals" ||
                       kind == "source_payload_pointer_equals";
    if (!known) invalid("sav.rule", "unknown rule kind");
    if (kind == "source_content_digest_equals") {
        if (!rule.at("expected_digest").is_string() || !tagged_digest(rule.at("expected_digest").get<std::string>()) ||
            !rule.at("pointer").is_null() || !rule.at("expected_json").is_null()) {
            invalid("sav.rule", "digest rule parameters are invalid");
        }
    } else if (kind == "source_payload_pointer_equals") {
        if (!rule.at("pointer").is_string() || !rule.at("expected_json").is_string() ||
            !rule.at("expected_digest").is_null()) invalid("sav.rule", "pointer rule parameters are invalid");
        try { static_cast<void>(engine::Json::parse(rule.at("expected_json").get<std::string>())); }
        catch (const std::exception&) { invalid("sav.rule", "expected_json is not JSON"); }
    } else if (!rule.at("pointer").is_null() || !rule.at("expected_json").is_null() ||
               !rule.at("expected_digest").is_null()) {
        invalid("sav.rule", "unused rule parameters must be null");
    }
}

void validate_reference(const engine::Json& reference) {
    exact_fields(reference, {"protocol", "reference_id", "relationship_id", "owner_vector", "owner_contract",
        "subject_id", "object_id", "relationship_type", "applicability", "applicability_rule",
        "evidence_protocols", "evaluation_rule", "evaluation_version", "violation_severity",
        "exception_eligible", "exception_bounds", "thermal_restriction", "source_digest", "record_digest"}, "reference");
    if (string_field(reference, "protocol") != "symphony.sav.accord-reference.v1") invalid("sav.protocol", "reference protocol is unsupported");
    for (const auto field : {"reference_id", "relationship_id", "owner_vector", "subject_id", "object_id"}) require_identity(reference, field);
    string_field(reference, "owner_contract");
    string_field(reference, "relationship_type", 64U);
    string_field(reference, "applicability", 32U);
    validate_rule(reference.at("applicability_rule"));
    static_cast<void>(string_array(reference, "evidence_protocols", 32U, true));
    validate_rule(reference.at("evaluation_rule"));
    require_identity(reference, "evaluation_version");
    string_field(reference, "violation_severity", 32U);
    if (!reference.at("exception_eligible").is_boolean()) invalid("sav.type", "exception_eligible must be boolean");
    if (!reference.at("exception_bounds").is_null() && !reference.at("exception_bounds").is_string()) invalid("sav.type", "exception_bounds must be null or string");
    string_field(reference, "thermal_restriction", 32U);
    require_digest(reference, "source_digest");
    require_digest(reference, "record_digest");
    if (digest_without(reference, "record_digest") != reference.at("record_digest").get<std::string>()) {
        invalid("sav.reference_digest", "record_digest does not bind the reference");
    }
}

enum class RuleState { pass, fail, unknown };

RuleState evaluate_rule(const engine::Json& rule, const std::map<std::string, engine::Json>& sources) {
    const auto kind = rule.at("kind").get<std::string>();
    if (kind == "always") return RuleState::pass;
    if (kind == "never") return RuleState::fail;
    for (const auto& id_value : rule.at("source_ids")) {
        const auto id = id_value.get<std::string>();
        const auto found = sources.find(id);
        if (found == sources.end() || found->second.at("collection_state") != "available") return RuleState::unknown;
    }
    if (kind == "source_available") return RuleState::pass;
    const auto& source = sources.at(rule.at("source_ids").at(0).get<std::string>());
    if (kind == "source_content_digest_equals") {
        return source.at("content_digest") == rule.at("expected_digest") ? RuleState::pass : RuleState::fail;
    }
    try {
        const engine::Json::json_pointer pointer(rule.at("pointer").get<std::string>());
        const auto expected = engine::Json::parse(rule.at("expected_json").get<std::string>());
        if (!source.at("payload").contains(pointer)) return RuleState::unknown;
        return source.at("payload").at(pointer) == expected ? RuleState::pass : RuleState::fail;
    } catch (const std::exception&) {
        return RuleState::unknown;
    }
}

engine::Json inspect(const engine::Json& payload) {
    exact_fields(payload, {}, "inspect payload");
    engine::Json result{{"protocol", "symphony.sav.inspect-result.v1"}, {"module_id", module_id},
        {"engine_id", engine_id}, {"engine_version", engine_version}, {"vector_id", vector_id},
        {"source_discovery", false}, {"canonical_mutation", false}, {"caller_neutral", true},
        {"thermal_path", "freezing"}, {"read_only", true}};
    result["result_digest"] = engine::tagged_sha256(result.dump());
    return result;
}

engine::Json reference_check(const engine::Json& payload) {
    exact_fields(payload, {"references"}, "reference_check payload");
    const auto& references = payload.at("references");
    if (!references.is_array() || references.size() > max_references) invalid("sav.bounds", "references exceed bound");
    std::string previous;
    auto digests = engine::Json::array();
    for (const auto& reference : references) {
        validate_reference(reference);
        const auto id = reference.at("reference_id").get<std::string>();
        if (!previous.empty() && id <= previous) invalid("sav.order", "references must be sorted and unique");
        previous = id;
        digests.push_back(reference.at("record_digest"));
    }
    engine::Json result{{"protocol", "symphony.sav.reference-check-result.v1"},
        {"state", "valid"}, {"reference_count", references.size()},
        {"reference_set_digest", engine::tagged_sha256(digests.dump())}, {"read_only", true}};
    result["result_digest"] = engine::tagged_sha256(result.dump());
    return result;
}

engine::Json current_resolve(const engine::Json& input) {
    exact_fields(input, {"protocol", "tops_id", "operation_id", "snapshot_started_at", "snapshot_completed_at",
        "named_version_id", "named_version_digest", "declared_scope", "required_source_ids", "sources"}, "CURRENT input");
    if (string_field(input, "protocol") != "symphony.sav.current-resolution-input.v1") invalid("sav.protocol", "CURRENT input protocol is unsupported");
    require_identity(input, "tops_id");
    require_identity(input, "operation_id");
    const auto started = string_field(input, "snapshot_started_at", 20U);
    const auto completed = string_field(input, "snapshot_completed_at", 20U);
    if (!engine::is_utc_seconds(started) || !engine::is_utc_seconds(completed) || completed < started) invalid("sav.timestamp", "snapshot timestamps are invalid");
    if (input.at("named_version_id").is_null() != input.at("named_version_digest").is_null()) invalid("sav.named_version", "named version ID and digest must be present together");
    if (!input.at("named_version_id").is_null()) {
        if (!safe_identity(input.at("named_version_id").get<std::string>()) || !tagged_digest(input.at("named_version_digest").get<std::string>())) invalid("sav.named_version", "named version binding is invalid");
    }
    const auto scope = string_array(input, "declared_scope", max_sources, true);
    const auto required = string_array(input, "required_source_ids", max_sources);
    const auto& sources = input.at("sources");
    if (!sources.is_array() || sources.size() > max_sources) invalid("sav.bounds", "sources exceed bound");
    std::map<std::string, engine::Json> indexed;
    for (const auto& source : sources) {
        validate_source(source);
        const auto id = source.at("source_id").get<std::string>();
        if (!indexed.emplace(id, source).second) invalid("sav.order", "source IDs must be unique");
    }
    auto sorted_sources = engine::Json::array();
    for (const auto& [id, source] : indexed) { static_cast<void>(id); sorted_sources.push_back(source); }
    auto unresolved = engine::Json::array();
    for (const auto& id : required) {
        const auto found = indexed.find(id);
        if (found == indexed.end()) {
            unresolved.push_back({{"source_id", id}, {"reason", "absent"}, {"detail", "required source was not supplied"}});
            continue;
        }
        const auto state = found->second.at("collection_state").get<std::string>();
        const auto freshness = found->second.at("freshness").get<std::string>();
        if (state != "available" || freshness == "stale") {
            const auto reason = freshness == "stale" ? "stale" : state;
            unresolved.push_back({{"source_id", id}, {"reason", reason}, {"detail", "required source is not usable"}});
        }
    }
    const auto coverage = unresolved.empty() ? "complete" : (indexed.empty() ? "unknown" : "partial");
    engine::Json result{{"protocol", "symphony.sav.current-snapshot.v1"}, {"tops_id", input.at("tops_id")},
        {"operation_id", input.at("operation_id")}, {"snapshot_started_at", input.at("snapshot_started_at")},
        {"snapshot_completed_at", input.at("snapshot_completed_at")}, {"named_version_id", input.at("named_version_id")},
        {"named_version_digest", input.at("named_version_digest")}, {"declared_scope", scope},
        {"scope_digest", engine::tagged_sha256(engine::Json(scope).dump())}, {"coverage_state", coverage},
        {"sources", std::move(sorted_sources)}, {"unresolved_sources", std::move(unresolved)},
        {"snapshot_digest", nullptr}, {"observation_digest", nullptr}, {"canonical", false}, {"derived", true}, {"read_only", true}};
    auto stable = result;
    stable.erase("snapshot_started_at"); stable.erase("snapshot_completed_at"); stable.erase("observation_digest"); stable.erase("snapshot_digest");
    result["snapshot_digest"] = engine::tagged_sha256(stable.dump());
    result["observation_digest"] = digest_without(result, "observation_digest");
    return result;
}

engine::Json evaluate(const engine::Json& payload) {
    exact_fields(payload, {"protocol", "current", "references"}, "evaluation input");
    if (string_field(payload, "protocol") != "symphony.sav.evaluation-input.v1") invalid("sav.protocol", "evaluation protocol is unsupported");
    const auto& current = payload.at("current");
    require_digest(current, "snapshot_digest");
    std::map<std::string, engine::Json> sources;
    for (const auto& source : current.at("sources")) {
        validate_source(source);
        sources.emplace(source.at("source_id").get<std::string>(), source);
    }
    const auto& references = payload.at("references");
    if (!references.is_array() || references.size() > max_references) invalid("sav.bounds", "references exceed bound");
    auto findings = engine::Json::array();
    auto reference_digests = engine::Json::array();
    std::array<std::size_t, 4U> counts{};
    bool unresolved = current.at("coverage_state") != "complete";
    bool stale = false;
    bool blocking = false;
    bool attunement = false;
    std::string previous;
    for (const auto& reference : references) {
        validate_reference(reference);
        const auto id = reference.at("reference_id").get<std::string>();
        if (!previous.empty() && id <= previous) invalid("sav.order", "references must be sorted and unique");
        previous = id;
        reference_digests.push_back(reference.at("record_digest"));
        const auto app = evaluate_rule(reference.at("applicability_rule"), sources);
        RuleState decision = RuleState::unknown;
        std::string applicability = "unknown";
        std::string state = "unknown";
        std::string reason = "sav.evidence.unknown";
        std::string detail = "applicability or evidence could not be established";
        if (reference.at("applicability") == "not_applicable" || app == RuleState::fail) {
            applicability = "not_applicable"; state = "not_applicable"; reason = "sav.reference.not_applicable";
            detail = "the closed applicability rule evaluated false"; ++counts[3];
        } else if (app == RuleState::pass) {
            applicability = "applicable";
            decision = evaluate_rule(reference.at("evaluation_rule"), sources);
            if (decision == RuleState::pass) { state = "in_accord"; reason = "sav.relationship.satisfied"; detail = "the closed evaluation rule passed"; ++counts[0]; }
            else if (decision == RuleState::fail) { state = "out_of_accord"; reason = "sav.relationship.violated"; detail = "the closed evaluation rule failed"; ++counts[1]; attunement = true; blocking = blocking || reference.at("violation_severity") == "blocking"; }
            else { ++counts[2]; unresolved = true; }
        } else { ++counts[2]; unresolved = true; }
        auto evidence_ids = engine::Json::array();
        auto evidence_digests = engine::Json::array();
        std::set<std::string> ids;
        for (const auto& rule_name : {"applicability_rule", "evaluation_rule"}) {
            for (const auto& value : reference.at(rule_name).at("source_ids")) ids.insert(value.get<std::string>());
        }
        for (const auto& source_id : ids) {
            evidence_ids.push_back(source_id);
            const auto found = sources.find(source_id);
            if (found != sources.end()) {
                evidence_digests.push_back(found->second.at("content_digest"));
                stale = stale || found->second.at("freshness") == "stale";
            }
        }
        engine::Json finding{{"reference_id", id}, {"relationship_id", reference.at("relationship_id")},
            {"subject_id", reference.at("subject_id")}, {"object_id", reference.at("object_id")},
            {"owner_contract", reference.at("owner_contract")}, {"applicability", applicability}, {"state", state},
            {"reason_code", reason}, {"detail", detail}, {"evidence_source_ids", std::move(evidence_ids)},
            {"evidence_digests", std::move(evidence_digests)}, {"sev_eligible", state == "out_of_accord"},
            {"finding_digest", nullptr}};
        finding["finding_digest"] = digest_without(finding, "finding_digest");
        findings.push_back(std::move(finding));
    }
    const bool contradiction = counts[1] > 0U;
    engine::Json result{{"protocol", "symphony.sav.evaluation-result.v1"},
        {"snapshot_digest", current.at("snapshot_digest")},
        {"reference_set_digest", engine::tagged_sha256(reference_digests.dump())},
        {"reference_resolution", stale ? "stale" : (unresolved ? "reference_unresolved" : "resolved")},
        {"composition_accord", contradiction ? "cacophonous" : (unresolved ? "indeterminate" : "in_accord")},
        {"transition_readiness", blocking ? "blocked" : (attunement ? "attunement_required" : (unresolved ? "not_evaluated" : "ready"))},
        {"findings", std::move(findings)}, {"summary", {{"in_accord", counts[0]}, {"out_of_accord", counts[1]},
            {"unknown", counts[2]}, {"not_applicable", counts[3]}}}, {"read_only", true}, {"noncanonical", true},
        {"result_digest", nullptr}};
    result["result_digest"] = digest_without(result, "result_digest");
    return result;
}

engine::Json diff(const engine::Json& payload) {
    exact_fields(payload, {"left", "right"}, "diff payload");
    require_digest(payload.at("left"), "snapshot_digest");
    require_digest(payload.at("right"), "snapshot_digest");
    std::map<std::string, std::string> left, right;
    for (const auto& s : payload.at("left").at("sources")) left.emplace(s.at("source_id"), s.at("content_digest"));
    for (const auto& s : payload.at("right").at("sources")) right.emplace(s.at("source_id"), s.at("content_digest"));
    auto changes = engine::Json::array();
    std::set<std::string> ids;
    for (const auto& [id, digest] : left) { static_cast<void>(digest); ids.insert(id); }
    for (const auto& [id, digest] : right) { static_cast<void>(digest); ids.insert(id); }
    for (const auto& id : ids) {
        const auto l = left.find(id); const auto r = right.find(id);
        if (l == left.end()) changes.push_back({{"source_id", id}, {"change", "added"}, {"before", nullptr}, {"after", r->second}});
        else if (r == right.end()) changes.push_back({{"source_id", id}, {"change", "removed"}, {"before", l->second}, {"after", nullptr}});
        else if (l->second != r->second) changes.push_back({{"source_id", id}, {"change", "changed"}, {"before", l->second}, {"after", r->second}});
    }
    engine::Json result{{"protocol", "symphony.sav.diff-result.v1"}, {"left_snapshot_digest", payload.at("left").at("snapshot_digest")},
        {"right_snapshot_digest", payload.at("right").at("snapshot_digest")}, {"changes", std::move(changes)}, {"read_only", true}, {"result_digest", nullptr}};
    result["result_digest"] = digest_without(result, "result_digest"); return result;
}

engine::Json explain(const engine::Json& payload) {
    exact_fields(payload, {"evaluation", "reference_id"}, "explain payload");
    const auto id = string_field(payload, "reference_id", 256U);
    for (const auto& finding : payload.at("evaluation").at("findings")) {
        if (finding.at("reference_id") == id) {
            engine::Json result{{"protocol", "symphony.sav.explain-result.v1"}, {"finding", finding},
                {"snapshot_digest", payload.at("evaluation").at("snapshot_digest")}, {"read_only", true}, {"result_digest", nullptr}};
            result["result_digest"] = digest_without(result, "result_digest"); return result;
        }
    }
    invalid("sav.reference_absent", "reference was not found in the evaluation");
}

engine::Json project_graph(const engine::Json& payload) {
    exact_fields(payload, {"current", "evaluation", "references"}, "graph payload");
    std::map<std::string, engine::Json> nodes;
    for (const auto& source : payload.at("current").at("sources")) {
        const auto id = source.at("source_id").get<std::string>();
        nodes.emplace(id, engine::Json{{"id", id}, {"kind", "source"}, {"digest", source.at("content_digest")}});
    }
    std::map<std::string, std::string> states;
    if (!payload.at("evaluation").is_null()) for (const auto& finding : payload.at("evaluation").at("findings")) states.emplace(finding.at("relationship_id"), finding.at("state"));
    auto edges = engine::Json::array();
    for (const auto& reference : payload.at("references")) {
        validate_reference(reference);
        for (const auto field : {"subject_id", "object_id"}) {
            const auto id = reference.at(field).get<std::string>();
            if (!nodes.contains(id)) nodes.emplace(id, engine::Json{{"id", id}, {"kind", "identity"}, {"digest", engine::tagged_sha256(id)}});
        }
        const auto relation_id = reference.at("relationship_id").get<std::string>();
        engine::Json edge{{"id", relation_id}, {"source", reference.at("subject_id")}, {"target", reference.at("object_id")},
            {"type", reference.at("relationship_type")}, {"state", states.contains(relation_id) ? states.at(relation_id) : "not_evaluated"}, {"digest", nullptr}};
        edge["digest"] = digest_without(edge, "digest"); edges.push_back(std::move(edge));
    }
    auto node_array = engine::Json::array(); for (auto& [id, node] : nodes) { static_cast<void>(id); node_array.push_back(std::move(node)); }
    engine::Json result{{"protocol", "symphony.sav.graph-projection.v1"}, {"snapshot_digest", payload.at("current").at("snapshot_digest")},
        {"evaluation_digest", payload.at("evaluation").is_null() ? engine::Json(nullptr) : payload.at("evaluation").at("result_digest")},
        {"nodes", std::move(node_array)}, {"edges", std::move(edges)}, {"noncanonical", true}, {"rebuildable", true}, {"projection_digest", nullptr}};
    result["projection_digest"] = digest_without(result, "projection_digest"); return result;
}

engine::Json compatibility(const engine::Json& payload) {
    exact_fields(payload, {"reader_versions", "writer_version"}, "compatibility payload");
    const auto readers = string_array(payload, "reader_versions", 32U, true);
    const auto writer = string_field(payload, "writer_version", 64U);
    const bool supported = writer == "v1" && std::find(readers.begin(), readers.end(), "v1") != readers.end();
    engine::Json result{{"protocol", "symphony.sav.compatibility-result.v1"}, {"writer_version", writer},
        {"reader_versions", readers}, {"compatible", supported}, {"reason", supported ? "exact_v1_overlap" : "no_supported_overlap"},
        {"read_only", true}, {"result_digest", nullptr}};
    result["result_digest"] = digest_without(result, "result_digest"); return result;
}

const std::vector<engine::OperationSpec>& operations() {
    static const std::vector<engine::OperationSpec> specs{
        {"engop:symphony:sav.inspect", "inspect", "implemented", false, true, {"ssfv:symphony:sav-engine"}, {"inspect"}, "not_applicable", {}, "symphony.sav.inspect-result.v1", "read_only", "idempotent", false, "none", "engop:symphony:sav.inspect", "supported", "freezing"},
        {"engop:symphony:sav.reference-check", "reference_check", "implemented", false, true, {"ssfv:symphony:sav-engine"}, {"validate"}, "not_applicable", {}, "symphony.sav.reference-check-result.v1", "read_only", "idempotent", false, "none", "engop:symphony:sav.reference-check", "supported", "freezing"},
        {"engop:symphony:sav.current-resolve", "current_resolve", "implemented", false, true, {"ssfv:symphony:sav-engine"}, {"query"}, "system_orchestrated", "symphony.sav.current-resolution-input.v1", "symphony.sav.current-snapshot.v1", "read_only", "idempotent", false, "none", "engop:symphony:sav.current-resolve", "supported", "freezing"},
        {"engop:symphony:sav.evaluate", "evaluate", "implemented", false, true, {"ssfv:symphony:sav-engine"}, {"validate"}, "system_orchestrated", "symphony.sav.evaluation-input.v1", "symphony.sav.evaluation-result.v1", "read_only", "idempotent", false, "none", "engop:symphony:sav.evaluate", "supported", "freezing"},
        {"engop:symphony:sav.diff", "diff", "implemented", false, true, {"ssfv:symphony:sav-engine"}, {"query"}, "not_applicable", {}, "symphony.sav.diff-result.v1", "read_only", "idempotent", false, "none", "engop:symphony:sav.diff", "supported", "freezing"},
        {"engop:symphony:sav.explain", "explain", "implemented", false, true, {"ssfv:symphony:sav-engine"}, {"query"}, "not_applicable", {}, "symphony.sav.explain-result.v1", "read_only", "idempotent", false, "none", "engop:symphony:sav.explain", "supported", "freezing"},
        {"engop:symphony:sav.project-graph", "project_graph", "implemented", false, true, {"ssfv:symphony:sav-engine"}, {"query"}, "not_applicable", {}, "symphony.sav.graph-projection.v1", "read_only", "idempotent", false, "none", "engop:symphony:sav.project-graph", "supported", "freezing"},
        {"engop:symphony:sav.compatibility", "compatibility", "implemented", false, true, {"ssfv:symphony:sav-engine"}, {"validate"}, "not_applicable", {}, "symphony.sav.compatibility-result.v1", "read_only", "idempotent", false, "none", "engop:symphony:sav.compatibility", "supported", "freezing"},
    };
    return specs;
}

}

engine::Json descriptor() {
    const auto& specs = operations();
    engine::validate_operation_specs(specs);
    return engine::Json{{"protocol", engine::descriptor_protocol_v2}, {"module_id", module_id},
        {"engine_id", engine_id}, {"vector_id", vector_id}, {"engine_version", engine_version},
        {"process_protocols", engine::Json::array({engine::process_protocol_v1})},
        {"contract_versions", engine::Json::array({"knowledge/SPEC.md@v1", "knowledge/sav/SPEC.md@v1"})},
        {"operations", engine::legacy_operation_descriptors(specs)},
        {"administration_operations", engine::administration_operation_descriptors(specs)},
        {"operation_registry_digest", engine::operation_registry_digest(specs)},
        {"supported_scopes", engine::Json::array({"user", "host"})}, {"language", "C++26"},
        {"thermal_path", "freezing"}, {"source_discovery", false}, {"network_listener", false},
        {"canonical_apply_enabled", false}, {"session_mutation_enabled", false},
        {"install_state", "installed_undocked"}, {"default_receptor", "receptor:symphony:knowledge.sav"}};
}

engine::Json handle_request(const engine::Request& request) {
    if (request.operation == "inspect") return inspect(request.payload);
    if (request.operation == "reference_check") return reference_check(request.payload);
    if (request.operation == "current_resolve") return current_resolve(request.payload);
    if (request.operation == "evaluate") return evaluate(request.payload);
    if (request.operation == "diff") return diff(request.payload);
    if (request.operation == "explain") return explain(request.payload);
    if (request.operation == "project_graph") return project_graph(request.payload);
    if (request.operation == "compatibility") return compatibility(request.payload);
    throw engine::Error("operation.unsupported", "operation is reserved or unsupported", 4);
}

}
