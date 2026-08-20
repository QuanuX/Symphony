#include "sev.hpp"

#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/operation.hpp"
#include "symphony/knowledge/engine/temporal.hpp"

#include <algorithm>
#include <array>
#include <map>
#include <queue>
#include <set>
#include <string>
#include <string_view>
#include <utility>
#include <vector>

namespace symphony::knowledge::sev {
namespace engine = symphony::knowledge::engine;

namespace {

constexpr std::size_t max_items = 1024U;
constexpr std::size_t max_edges = 8192U;

[[noreturn]] void invalid(std::string code, std::string message) {
    throw engine::Error(std::move(code), std::move(message), 3);
}

void exact_fields(const engine::Json& value, std::initializer_list<std::string_view> fields,
                  std::string_view context) {
    if (!value.is_object()) invalid("sev.type", std::string(context) + " must be an object");
    std::set<std::string> expected;
    for (const auto field : fields) expected.emplace(field);
    if (value.size() != expected.size()) invalid("sev.fields", std::string(context) + " has an unexpected field set");
    for (const auto& field : expected) if (!value.contains(field)) invalid("sev.fields", std::string(context) + " is missing " + field);
}

const std::string& text(const engine::Json& value, std::string_view field, std::size_t max = 4096U) {
    const auto key = std::string(field);
    if (!value.at(key).is_string()) invalid("sev.type", key + " must be a string");
    const auto& result = value.at(key).get_ref<const std::string&>();
    if (result.empty() || result.size() > max) invalid("sev.bounds", key + " is outside its bound");
    return result;
}

bool safe_id(std::string_view value) {
    return !value.empty() && value.size() <= 256U && std::all_of(value.begin(), value.end(), [](const unsigned char c) {
        return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
               c == '.' || c == '_' || c == ':' || c == '/' || c == '+' || c == '-';
    });
}

bool digest(std::string_view value) {
    return value.size() == 71U && value.starts_with("sha256:") &&
           std::all_of(value.begin() + 7, value.end(), [](const unsigned char c) {
               return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f');
           });
}

void require_id(const engine::Json& value, std::string_view field) {
    if (!safe_id(text(value, field, 256U))) invalid("sev.identity", std::string(field) + " is invalid");
}

void require_digest(const engine::Json& value, std::string_view field) {
    if (!digest(text(value, field, 71U))) invalid("sev.digest", std::string(field) + " is invalid");
}

std::string digest_without(engine::Json value, std::string_view field) {
    value.erase(std::string(field));
    return engine::tagged_sha256(value.dump());
}

std::vector<std::string> ids(const engine::Json& value, std::string_view field,
                             std::size_t maximum = max_items) {
    const auto& array = value.at(std::string(field));
    if (!array.is_array() || array.size() > maximum) invalid("sev.bounds", std::string(field) + " exceeds bound");
    std::vector<std::string> result;
    for (const auto& item : array) {
        if (!item.is_string() || !safe_id(item.get_ref<const std::string&>())) invalid("sev.identity", std::string(field) + " contains an invalid ID");
        result.push_back(item.get<std::string>());
    }
    if (!std::is_sorted(result.begin(), result.end()) || std::adjacent_find(result.begin(), result.end()) != result.end()) {
        invalid("sev.order", std::string(field) + " must be sorted and unique");
    }
    return result;
}

void valid_case(const engine::Json& value) {
    exact_fields(value, {"protocol", "case_id", "case_kind", "semantic_fingerprint", "source_snapshot_digest",
        "target_digest", "state", "generation", "predecessor_digest", "affected_surfaces", "disposition_ids",
        "ready_action_ids", "blocker_ids", "opened_at", "updated_at", "canonical", "read_only", "case_digest"}, "case");
    if (text(value, "protocol") != "symphony.sev.evolution-case.v1") invalid("sev.protocol", "case protocol is unsupported");
    require_id(value, "case_id"); require_digest(value, "semantic_fingerprint");
    require_digest(value, "source_snapshot_digest"); require_digest(value, "target_digest");
    const auto kind = text(value, "case_kind", 32U); const auto state = text(value, "state", 64U);
    if (kind != "planned_change" && kind != "encountered_novelty") invalid("sev.case_kind", "case kind is invalid");
    static const std::set<std::string> states = {
        "open", "ready", "blocked", "reobservation_required", "recalculation_required",
        "attunement_required", "converged", "closed", "abandoned"
    };
    if (!states.contains(state)) invalid("sev.case_state", "case state is invalid");
    if (!value.at("generation").is_number_unsigned() || value.at("generation").get<std::uint64_t>() == 0U) invalid("sev.generation", "case generation is invalid");
    if (!value.at("predecessor_digest").is_null()) require_digest(value, "predecessor_digest");
    static_cast<void>(ids(value, "affected_surfaces")); static_cast<void>(ids(value, "disposition_ids"));
    static_cast<void>(ids(value, "ready_action_ids")); static_cast<void>(ids(value, "blocker_ids"));
    const auto opened = text(value, "opened_at", 20U); const auto updated = text(value, "updated_at", 20U);
    if (!engine::is_utc_seconds(opened) || !engine::is_utc_seconds(updated) || updated < opened) invalid("sev.timestamp", "case timestamps are invalid");
    if (value.at("canonical") != false || value.at("read_only") != true) invalid("sev.authority", "case authority flags are invalid");
    require_digest(value, "case_digest");
    if (digest_without(value, "case_digest") != value.at("case_digest").get<std::string>()) invalid("sev.case_digest", "case digest mismatch");
}

void valid_rule(const engine::Json& rule) {
    exact_fields(rule, {"kind", "source_ids", "pointer", "expected_json", "expected_digest"}, "success predicate");
    const auto kind = text(rule, "kind", 64U); const auto source_ids = ids(rule, "source_ids", 64U);
    if ((kind == "always" || kind == "never") && !source_ids.empty()) invalid("sev.rule", "constant predicate names a source");
    if (kind == "source_available" && source_ids.empty()) invalid("sev.rule", "source_available needs evidence");
    if ((kind == "source_content_digest_equals" || kind == "source_payload_pointer_equals") && source_ids.size() != 1U) invalid("sev.rule", "comparison predicate needs one source");
    const bool known = kind == "always" || kind == "never" || kind == "source_available" ||
        kind == "source_content_digest_equals" || kind == "source_payload_pointer_equals";
    if (!known) invalid("sev.rule", "predicate kind is unsupported");
    if (kind == "source_content_digest_equals") {
        if (!rule.at("expected_digest").is_string() || !digest(rule.at("expected_digest").get<std::string>()) ||
            !rule.at("pointer").is_null() || !rule.at("expected_json").is_null()) invalid("sev.rule", "digest predicate parameters are invalid");
    } else if (kind == "source_payload_pointer_equals") {
        if (!rule.at("pointer").is_string() || !rule.at("expected_json").is_string() || !rule.at("expected_digest").is_null()) invalid("sev.rule", "pointer predicate parameters are invalid");
        try { static_cast<void>(engine::Json::parse(rule.at("expected_json").get<std::string>())); }
        catch (const std::exception&) { invalid("sev.rule", "expected_json is invalid"); }
    } else if (!rule.at("pointer").is_null() || !rule.at("expected_json").is_null() || !rule.at("expected_digest").is_null()) {
        invalid("sev.rule", "unused predicate parameters must be null");
    }
}

enum class RuleState { pass, fail, unknown };
RuleState evaluate_rule(const engine::Json& rule, const engine::Json& current) {
    const auto kind = rule.at("kind").get<std::string>();
    if (kind == "always") return RuleState::pass;
    if (kind == "never") return RuleState::fail;
    std::map<std::string, engine::Json> sources;
    for (const auto& source : current.at("sources")) sources.emplace(source.at("source_id"), source);
    for (const auto& id : rule.at("source_ids")) {
        const auto found = sources.find(id.get<std::string>());
        if (found == sources.end() || found->second.at("collection_state") != "available") return RuleState::unknown;
    }
    if (kind == "source_available") return RuleState::pass;
    const auto& source = sources.at(rule.at("source_ids").at(0).get<std::string>());
    if (kind == "source_content_digest_equals") return source.at("content_digest") == rule.at("expected_digest") ? RuleState::pass : RuleState::fail;
    try {
        const engine::Json::json_pointer pointer(rule.at("pointer").get<std::string>());
        if (!source.at("payload").contains(pointer)) return RuleState::unknown;
        return source.at("payload").at(pointer) == engine::Json::parse(rule.at("expected_json").get<std::string>()) ? RuleState::pass : RuleState::fail;
    } catch (const std::exception&) { return RuleState::unknown; }
}

void valid_action(const engine::Json& action) {
    exact_fields(action, {"action_id", "operation_id", "disposition", "target_id", "expected_state_digest",
        "authorization_operation", "audit_required", "success_predicate", "recovery_operation_id",
        "execution_class", "thermal_restriction", "action_digest"}, "action");
    for (const auto field : {"action_id", "operation_id", "target_id", "authorization_operation"}) require_id(action, field);
    const auto disposition = text(action, "disposition", 64U);
    static const std::set<std::string> dispositions = {
        "accept_as_compatible", "attune_configuration", "extend_contract",
        "extend_command_surface", "install_exact_component", "select_exact_component",
        "dock_exact_component", "undock_exact_component", "replace_with_successor",
        "preserve_observed_state", "defer_with_blocker", "reject_incompatible", "retire_identity"
    };
    if (!dispositions.contains(disposition)) invalid("sev.disposition", "action disposition is unsupported");
    require_digest(action, "expected_state_digest");
    if (!action.at("audit_required").is_boolean()) invalid("sev.type", "audit_required must be boolean");
    valid_rule(action.at("success_predicate"));
    if (!action.at("recovery_operation_id").is_null()) require_id(action, "recovery_operation_id");
    text(action, "execution_class", 32U);
    if (text(action, "thermal_restriction", 32U) != "freezing_only") invalid("sev.thermal", "action is not freezing-only");
    require_digest(action, "action_digest");
    if (digest_without(action, "action_digest") != action.at("action_digest").get<std::string>()) invalid("sev.action_digest", "action digest mismatch");
}

engine::Json inspect(const engine::Json& payload) {
    exact_fields(payload, {}, "inspect payload");
    engine::Json result{{"protocol", "symphony.sev.inspect-result.v1"}, {"module_id", module_id},
        {"engine_id", engine_id}, {"engine_version", engine_version}, {"vector_id", vector_id},
        {"scsev_profile", true}, {"separate_scsev_registry", false}, {"canonical_mutation", false},
        {"caller_neutral", true}, {"thermal_path", "freezing"}, {"read_only", true}};
    result["result_digest"] = engine::tagged_sha256(result.dump()); return result;
}

engine::Json case_open(const engine::Json& input) {
    exact_fields(input, {"protocol", "case_id", "case_kind", "source_current", "target", "created_at"}, "case-open input");
    if (text(input, "protocol") != "symphony.sev.case-open-input.v1") invalid("sev.protocol", "case-open protocol is unsupported");
    require_id(input, "case_id");
    const auto kind = text(input, "case_kind", 32U);
    if (kind != "planned_change" && kind != "encountered_novelty") invalid("sev.case_kind", "case kind is invalid");
    const auto created = text(input, "created_at", 20U);
    if (!engine::is_utc_seconds(created)) invalid("sev.timestamp", "created_at is not STSC whole-second UTC");
    const auto& current = input.at("source_current"); require_digest(current, "snapshot_digest");
    const auto target_digest = engine::tagged_sha256(input.at("target").dump());
    engine::Json fingerprint{{"case_kind", kind}, {"source_snapshot_digest", current.at("snapshot_digest")}, {"target_digest", target_digest}};
    engine::Json result{{"protocol", "symphony.sev.evolution-case.v1"}, {"case_id", input.at("case_id")},
        {"case_kind", kind}, {"semantic_fingerprint", engine::tagged_sha256(fingerprint.dump())},
        {"source_snapshot_digest", current.at("snapshot_digest")}, {"target_digest", target_digest}, {"state", "open"},
        {"generation", 1U}, {"predecessor_digest", nullptr}, {"affected_surfaces", engine::Json::array()},
        {"disposition_ids", engine::Json::array()}, {"ready_action_ids", engine::Json::array()},
        {"blocker_ids", engine::Json::array()}, {"opened_at", created}, {"updated_at", created},
        {"canonical", false}, {"read_only", true}, {"case_digest", nullptr}};
    result["case_digest"] = digest_without(result, "case_digest"); return result;
}

void valid_impact_item(const engine::Json& item) {
    exact_fields(item, {"id", "kind", "reason", "evidence_digests"}, "impact item");
    require_id(item, "id"); require_id(item, "kind"); text(item, "reason", 65536U);
    const auto& evidence = item.at("evidence_digests");
    if (!evidence.is_array() || evidence.size() > 64U) invalid("sev.bounds", "impact evidence exceeds bound");
    std::string previous;
    for (const auto& value : evidence) {
        if (!value.is_string() || !digest(value.get<std::string>())) invalid("sev.digest", "impact evidence digest is invalid");
        const auto current = value.get<std::string>(); if (!previous.empty() && current <= previous) invalid("sev.order", "impact evidence must be sorted and unique"); previous = current;
    }
}

engine::Json impact_assess(const engine::Json& payload) {
    exact_fields(payload, {"case", "coverage_state", "affected", "unresolved", "conflicts"}, "impact payload");
    valid_case(payload.at("case"));
    const auto coverage = text(payload, "coverage_state", 32U);
    if (coverage != "complete" && coverage != "partial" && coverage != "unknown") invalid("sev.coverage", "coverage state is invalid");
    for (const auto field : {"affected", "unresolved", "conflicts"}) {
        const auto& items = payload.at(field); if (!items.is_array() || items.size() > max_items) invalid("sev.bounds", std::string(field) + " exceeds bound");
        std::string previous; for (const auto& item : items) { valid_impact_item(item); const auto id = item.at("id").get<std::string>(); if (!previous.empty() && id <= previous) invalid("sev.order", std::string(field) + " must be sorted and unique"); previous = id; }
    }
    const bool complete = coverage == "complete" && payload.at("unresolved").empty();
    engine::Json result{{"protocol", "symphony.sev.impact-result.v1"}, {"case_digest", payload.at("case").at("case_digest")},
        {"source_snapshot_digest", payload.at("case").at("source_snapshot_digest")}, {"coverage_state", coverage},
        {"affected", payload.at("affected")}, {"unresolved", payload.at("unresolved")}, {"conflicts", payload.at("conflicts")},
        {"complete", complete}, {"read_only", true}, {"noncanonical", true}, {"impact_digest", nullptr}};
    result["impact_digest"] = digest_without(result, "impact_digest"); return result;
}

engine::Json disposition_plan(const engine::Json& payload) {
    exact_fields(payload, {"case", "impact", "actions", "edges", "blockers"}, "plan payload");
    valid_case(payload.at("case")); require_digest(payload.at("impact"), "impact_digest");
    const auto& actions = payload.at("actions"); const auto& edges = payload.at("edges"); const auto& blockers = payload.at("blockers");
    if (!actions.is_array() || actions.size() > max_items || !edges.is_array() || edges.size() > max_edges ||
        !blockers.is_array() || blockers.size() > max_items) invalid("sev.bounds", "plan collection exceeds bound");
    std::map<std::string, engine::Json> action_map; std::map<std::string, std::set<std::string>> successors;
    std::map<std::string, std::size_t> indegree;
    std::string previous_action;
    for (const auto& action : actions) {
        valid_action(action); const auto id = action.at("action_id").get<std::string>();
        if ((!previous_action.empty() && id <= previous_action) || !action_map.emplace(id, action).second) {
            invalid("sev.order", "actions must be sorted by unique action ID");
        }
        previous_action = id; indegree[id] = 0U;
    }
    std::set<std::string> edge_keys;
    std::string previous_edge;
    for (const auto& edge : edges) {
        exact_fields(edge, {"source", "target", "kind"}, "dependency edge"); require_id(edge, "source"); require_id(edge, "target");
        const auto source = edge.at("source").get<std::string>(); const auto target = edge.at("target").get<std::string>();
        if (!action_map.contains(source) || !action_map.contains(target) || source == target) invalid("sev.edge", "dependency edge references an invalid action");
        const auto kind = text(edge, "kind", 32U); if (kind != "hard_safety" && kind != "semantic_dependency") invalid("sev.edge", "dependency kind is invalid");
        const auto edge_key = source + "\n" + target + "\n" + kind;
        if ((!previous_edge.empty() && edge_key <= previous_edge) || !edge_keys.insert(source + "\n" + target).second) {
            invalid("sev.order", "dependency edges must be sorted and unique");
        }
        previous_edge = edge_key;
        successors[source].insert(target); ++indegree[target];
    }
    auto queue = std::queue<std::string>(); auto degree = indegree;
    for (const auto& [id, count] : degree) if (count == 0U) queue.push(id);
    std::size_t visited = 0U; while (!queue.empty()) { const auto id = queue.front(); queue.pop(); ++visited; for (const auto& next : successors[id]) if (--degree[next] == 0U) queue.push(next); }
    if (visited != action_map.size()) invalid("sev.dependency_cycle", "plan dependency graph contains a cycle");
    std::set<std::string> blocked;
    std::string previous_blocker;
    for (const auto& blocker : blockers) {
        exact_fields(blocker, {"blocker_id", "action_id", "reason", "detail"}, "blocker"); require_id(blocker, "blocker_id"); require_id(blocker, "action_id"); require_id(blocker, "reason"); text(blocker, "detail", 65536U);
        const auto blocker_id = blocker.at("blocker_id").get<std::string>();
        if (!previous_blocker.empty() && blocker_id <= previous_blocker) invalid("sev.order", "blockers must be sorted and unique");
        previous_blocker = blocker_id;
        const auto action = blocker.at("action_id").get<std::string>(); if (!action_map.contains(action)) invalid("sev.blocker", "blocker references an absent action"); blocked.insert(action);
    }
    std::queue<std::string> blocked_queue; for (const auto& id : blocked) blocked_queue.push(id);
    while (!blocked_queue.empty()) { const auto id = blocked_queue.front(); blocked_queue.pop(); for (const auto& next : successors[id]) if (blocked.insert(next).second) blocked_queue.push(next); }
    auto ready = engine::Json::array(); for (const auto& [id, count] : indegree) if (count == 0U && !blocked.contains(id)) ready.push_back(id);
    const auto state = action_map.empty() ? "converged" : (!ready.empty() ? "ready" : (!blocked.empty() ? "blocked" : "attunement_required"));
    engine::Json result{{"protocol", "symphony.sev.disposition-plan.v1"}, {"case_digest", payload.at("case").at("case_digest")},
        {"impact_digest", payload.at("impact").at("impact_digest")}, {"actions", actions}, {"edges", edges},
        {"ready_action_ids", std::move(ready)}, {"blockers", blockers}, {"state", state},
        {"canonical", false}, {"read_only", true}, {"apply_authorized", false}, {"plan_digest", nullptr}};
    result["plan_digest"] = digest_without(result, "plan_digest"); return result;
}

engine::Json transition_verify(const engine::Json& input) {
    exact_fields(input, {"protocol", "case", "plan", "action_id", "execution_evidence_digest", "observed_current"}, "verification input");
    if (text(input, "protocol") != "symphony.sev.transition-verification-input.v1") invalid("sev.protocol", "verification protocol is unsupported");
    valid_case(input.at("case")); require_digest(input.at("plan"), "plan_digest"); require_id(input, "action_id"); require_digest(input, "execution_evidence_digest");
    const auto& current = input.at("observed_current"); require_digest(current, "snapshot_digest");
    const engine::Json* action = nullptr; for (const auto& candidate : input.at("plan").at("actions")) if (candidate.at("action_id") == input.at("action_id")) action = &candidate;
    if (action == nullptr) invalid("sev.action_absent", "attempted action is absent from the plan");
    valid_action(*action);
    const auto rule = current.at("coverage_state") == "complete" ? evaluate_rule(action->at("success_predicate"), current) : RuleState::unknown;
    const auto outcome = rule == RuleState::pass ? "proven_complete" : (rule == RuleState::fail ? "proven_failed" : "indeterminate");
    engine::Json result{{"protocol", "symphony.sev.transition-verification-result.v1"}, {"case_digest", input.at("case").at("case_digest")},
        {"plan_digest", input.at("plan").at("plan_digest")}, {"action_id", input.at("action_id")},
        {"execution_evidence_digest", input.at("execution_evidence_digest")}, {"observed_snapshot_digest", current.at("snapshot_digest")},
        {"outcome", outcome}, {"recalculation_required", true},
        {"detail", rule == RuleState::pass ? "success predicate is proven by the complete observed CURRENT" :
            (rule == RuleState::fail ? "success predicate is contradicted by the complete observed CURRENT" : "evidence does not prove or contradict the success predicate")},
        {"read_only", true}, {"result_digest", nullptr}};
    result["result_digest"] = digest_without(result, "result_digest"); return result;
}

engine::Json case_recalculate(const engine::Json& payload) {
    exact_fields(payload, {"case", "plan", "observed_current", "completed_action_ids", "failed_action_ids", "updated_at"}, "recalculation payload");
    valid_case(payload.at("case")); require_digest(payload.at("observed_current"), "snapshot_digest");
    exact_fields(payload.at("plan"), {"protocol", "case_digest", "impact_digest", "actions", "edges",
        "ready_action_ids", "blockers", "state", "canonical", "read_only", "apply_authorized", "plan_digest"}, "recalculation plan");
    if (payload.at("plan").at("protocol") != "symphony.sev.disposition-plan.v1" ||
        payload.at("plan").at("canonical") != false || payload.at("plan").at("read_only") != true ||
        payload.at("plan").at("apply_authorized") != false || !payload.at("plan").at("actions").is_array() ||
        !payload.at("plan").at("edges").is_array() || !payload.at("plan").at("blockers").is_array()) {
        invalid("sev.plan", "recalculation plan authority or collection fields are invalid");
    }
    require_digest(payload.at("plan"), "plan_digest");
    if (digest_without(payload.at("plan"), "plan_digest") != payload.at("plan").at("plan_digest").get<std::string>() ||
        payload.at("plan").at("case_digest") != payload.at("case").at("case_digest")) {
        invalid("sev.plan_digest", "recalculation plan does not bind the supplied case");
    }
    const auto completed = ids(payload, "completed_action_ids"); const auto failed = ids(payload, "failed_action_ids");
    std::set<std::string> completed_set(completed.begin(), completed.end());
    std::set<std::string> failed_set(failed.begin(), failed.end());
    for (const auto& id : completed_set) if (failed_set.contains(id)) invalid("sev.attempt_state", "one action cannot be completed and failed");
    std::map<std::string, engine::Json> actions;
    std::map<std::string, std::set<std::string>> predecessors;
    std::map<std::string, std::set<std::string>> successors;
    std::string previous_action;
    for (const auto& action : payload.at("plan").at("actions")) {
        valid_action(action);
        const auto id = action.at("action_id").get<std::string>();
        if (!previous_action.empty() && id <= previous_action) invalid("sev.order", "plan actions are not canonical");
        previous_action = id; actions.emplace(id, action);
    }
    for (const auto& id : completed_set) if (!actions.contains(id)) invalid("sev.action_absent", "completed action is absent from plan");
    for (const auto& id : failed_set) if (!actions.contains(id)) invalid("sev.action_absent", "failed action is absent from plan");
    for (const auto& edge : payload.at("plan").at("edges")) {
        const auto source = edge.at("source").get<std::string>();
        const auto target = edge.at("target").get<std::string>();
        if (!actions.contains(source) || !actions.contains(target)) invalid("sev.edge", "plan edge references an absent action");
        predecessors[target].insert(source); successors[source].insert(target);
    }
    std::set<std::string> blocked = failed_set;
    for (const auto& blocker : payload.at("plan").at("blockers")) blocked.insert(blocker.at("action_id").get<std::string>());
    std::queue<std::string> blocked_queue;
    for (const auto& id : blocked) blocked_queue.push(id);
    while (!blocked_queue.empty()) {
        const auto id = blocked_queue.front(); blocked_queue.pop();
        for (const auto& successor : successors[id]) if (blocked.insert(successor).second) blocked_queue.push(successor);
    }
    auto ready = engine::Json::array();
    auto blocker_ids = engine::Json::array();
    auto disposition_ids = engine::Json::array();
    auto surfaces = engine::Json::array();
    std::set<std::string> surface_set;
    for (const auto& [id, action] : actions) {
        disposition_ids.push_back(id);
        surface_set.insert(action.at("target_id").get<std::string>());
        if (blocked.contains(id)) blocker_ids.push_back(id);
        if (completed_set.contains(id) || blocked.contains(id)) continue;
        const auto& required = predecessors[id];
        if (std::all_of(required.begin(), required.end(), [&](const auto& predecessor) { return completed_set.contains(predecessor); })) {
            ready.push_back(id);
        }
    }
    for (const auto& surface : surface_set) surfaces.push_back(surface);
    const auto updated = text(payload, "updated_at", 20U); if (!engine::is_utc_seconds(updated) || updated < payload.at("case").at("updated_at")) invalid("sev.timestamp", "recalculation time is invalid");
    auto result = payload.at("case"); result["generation"] = result.at("generation").get<std::uint64_t>() + 1U;
    result["predecessor_digest"] = payload.at("case").at("case_digest"); result["source_snapshot_digest"] = payload.at("observed_current").at("snapshot_digest");
    result["updated_at"] = updated; result["ready_action_ids"] = std::move(ready);
    result["blocker_ids"] = std::move(blocker_ids); result["disposition_ids"] = std::move(disposition_ids);
    result["affected_surfaces"] = std::move(surfaces);
    const bool coverage_complete = payload.at("observed_current").contains("coverage_state") &&
        payload.at("observed_current").at("coverage_state") == "complete";
    const bool converged = completed_set.size() == actions.size() && blocked.empty();
    result["state"] = !coverage_complete ? "reobservation_required" :
        (converged ? "converged" : (!result.at("ready_action_ids").empty() ? "ready" :
            (!blocked.empty() ? "blocked" : "recalculation_required")));
    result["case_digest"] = nullptr; result["case_digest"] = digest_without(result, "case_digest"); return result;
}

engine::Json case_status(const engine::Json& payload) {
    exact_fields(payload, {"case"}, "status payload"); valid_case(payload.at("case"));
    engine::Json result{{"protocol", "symphony.sev.case-status-result.v1"}, {"case_id", payload.at("case").at("case_id")},
        {"case_digest", payload.at("case").at("case_digest")}, {"generation", payload.at("case").at("generation")},
        {"state", payload.at("case").at("state")}, {"ready_action_ids", payload.at("case").at("ready_action_ids")},
        {"blocker_ids", payload.at("case").at("blocker_ids")}, {"read_only", true}, {"result_digest", nullptr}};
    result["result_digest"] = digest_without(result, "result_digest"); return result;
}

engine::Json case_recover(const engine::Json& payload) {
    exact_fields(payload, {"case", "plan", "attempt_states"}, "recovery payload"); valid_case(payload.at("case"));
    if (!payload.at("plan").is_null()) require_digest(payload.at("plan"), "plan_digest");
    if (!payload.at("attempt_states").is_array() || payload.at("attempt_states").size() > max_items) invalid("sev.bounds", "attempt states exceed bound");
    const auto state = payload.at("case").at("state").get<std::string>();
    const auto advice = state == "reobservation_required" || state == "recalculation_required" ? "reobserve_then_recalculate" :
        (state == "blocked" ? "inspect_blockers_then_recover_external_action" : (state == "converged" ? "propose_close" : "resume_from_ready_set"));
    engine::Json result{{"protocol", "symphony.sev.case-recovery-advice.v1"}, {"case_digest", payload.at("case").at("case_digest")},
        {"advice", advice}, {"mutated", false}, {"read_only", true}, {"result_digest", nullptr}};
    result["result_digest"] = digest_without(result, "result_digest"); return result;
}

engine::Json case_close(const engine::Json& payload) {
    exact_fields(payload, {"case", "abandonment_reason", "authority_evidence_digest"}, "close payload"); valid_case(payload.at("case"));
    const auto state = payload.at("case").at("state").get<std::string>();
    const bool converged = state == "converged"; const bool abandonment = !payload.at("abandonment_reason").is_null();
    if (!converged && !abandonment) invalid("sev.close_ineligible", "case is neither converged nor explicitly abandoned");
    if (abandonment && !payload.at("abandonment_reason").is_string()) invalid("sev.type", "abandonment reason is invalid");
    if (!payload.at("authority_evidence_digest").is_null()) require_digest(payload, "authority_evidence_digest");
    engine::Json result{{"protocol", "symphony.sev.case-close-proposal.v1"}, {"case_digest", payload.at("case").at("case_digest")},
        {"proposed_state", converged ? "closed" : "abandoned"}, {"abandonment_reason", payload.at("abandonment_reason")},
        {"authority_evidence_digest", payload.at("authority_evidence_digest")}, {"apply_authorized", false},
        {"read_only", true}, {"proposal_digest", nullptr}};
    result["proposal_digest"] = digest_without(result, "proposal_digest"); return result;
}

const std::array<std::string_view, 14U> consequence_families{
    "command_identity", "grammar", "feature_binding", "engine_operation_binding", "authority",
    "protocols", "headless_behavior", "mutation_recovery", "manifest_parity", "tests",
    "invariant_evidence", "knowledge_documentation", "thermal_noninterference", "compatibility_lifecycle"};

engine::Json command_surface_assess(const engine::Json& payload) {
    exact_fields(payload, {"source_snapshot_digest", "proposal_digest", "change_kind", "consequences"}, "SCSEV payload");
    require_digest(payload, "source_snapshot_digest"); require_digest(payload, "proposal_digest");
    const auto kind = text(payload, "change_kind", 32U);
    if (kind != "add" && kind != "change" && kind != "deprecate" && kind != "replace" && kind != "retire") invalid("sev.change_kind", "command change kind is invalid");
    const auto& consequences = payload.at("consequences");
    if (!consequences.is_array() || consequences.size() != consequence_families.size()) invalid("sev.scsev_coverage", "all fourteen consequence families are required");
    auto missing = engine::Json::array();
    for (std::size_t index = 0; index < consequence_families.size(); ++index) {
        const auto& consequence = consequences.at(index);
        exact_fields(consequence, {"family", "state", "detail", "evidence_digests"}, "SCSEV consequence");
        if (text(consequence, "family", 64U) != consequence_families[index]) invalid("sev.scsev_order", "consequence families must use canonical order");
        const auto state = text(consequence, "state", 32U);
        if (state != "satisfied" && state != "missing" && state != "prohibited" && state != "not_applicable" && state != "unresolved") invalid("sev.scsev_state", "consequence state is invalid");
        text(consequence, "detail", 65536U);
        const auto& evidence = consequence.at("evidence_digests"); if (!evidence.is_array() || evidence.size() > 64U) invalid("sev.bounds", "consequence evidence exceeds bound");
        for (const auto& value : evidence) if (!value.is_string() || !digest(value.get<std::string>())) invalid("sev.digest", "consequence evidence digest is invalid");
        if (state == "missing" || state == "prohibited" || state == "unresolved") missing.push_back(consequence);
    }
    std::string state = "complete";
    if (!missing.empty()) {
        state = "unresolved";
        for (const auto& item : missing) {
            if (item.at("family") == "command_identity" || item.at("family") == "feature_binding") state = "semantic_registration_required";
            else if (state == "unresolved" && (item.at("family") == "grammar" || item.at("family") == "engine_operation_binding" || item.at("family") == "manifest_parity")) state = "administration_unintegrated";
            if (item.at("state") == "prohibited") state = "blocked_incompatible";
        }
        if ((kind == "deprecate" || kind == "retire") && state == "unresolved") state = "retirement_incomplete";
    }
    engine::Json result{{"protocol", "symphony.sev.command-surface-assessment.v1"},
        {"source_snapshot_digest", payload.at("source_snapshot_digest")}, {"proposal_digest", payload.at("proposal_digest")},
        {"state", state}, {"consequences", consequences}, {"missing", std::move(missing)},
        {"invented_identity", false}, {"invented_grammar", false}, {"apply_authorized", false},
        {"read_only", true}, {"assessment_digest", nullptr}};
    result["assessment_digest"] = digest_without(result, "assessment_digest"); return result;
}

bool contains_prohibited_novelty_key(const engine::Json& value) {
    static const std::set<std::string> prohibited = {
        "credential", "credentials", "token", "tokens", "proof", "proofs", "assertion",
        "assertions", "raw_token", "provider_payload", "secret", "secrets", "private_key"
    };
    if (value.is_object()) {
        for (const auto& [key, child] : value.items()) {
            if (prohibited.contains(key) || contains_prohibited_novelty_key(child)) return true;
        }
    } else if (value.is_array()) {
        for (const auto& child : value) if (contains_prohibited_novelty_key(child)) return true;
    }
    return false;
}

engine::Json novelty_bundle_check(const engine::Json& payload) {
    exact_fields(payload, {"bundle"}, "novelty bundle check payload");
    const auto& bundle = payload.at("bundle");
    exact_fields(bundle, {"protocol", "bundle_id", "case_digest", "source_snapshot_digest", "items",
        "redactions", "approval_reference", "offline_capable", "network_transfer", "created_at",
        "canonical", "bundle_digest"}, "novelty bundle");
    if (text(bundle, "protocol") != "symphony.sev.novelty-bundle.v1") invalid("sev.protocol", "novelty bundle protocol is unsupported");
    require_id(bundle, "bundle_id"); require_digest(bundle, "case_digest"); require_digest(bundle, "source_snapshot_digest");
    if (!bundle.at("bundle_id").get<std::string>().starts_with("sevnovelty:")) invalid("sev.bundle_id", "novelty bundle identity must use sevnovelty namespace");
    require_id(bundle, "approval_reference");
    if (bundle.at("offline_capable") != true || bundle.at("network_transfer") != false || bundle.at("canonical") != false ||
        !engine::is_utc_seconds(text(bundle, "created_at", 20U))) invalid("sev.novelty_authority", "novelty bundle authority or time is invalid");
    if (!bundle.at("items").is_array() || bundle.at("items").size() > 4096U ||
        !bundle.at("redactions").is_array() || bundle.at("redactions").size() > 4096U) invalid("sev.bounds", "novelty bundle exceeds bound");
    std::set<std::string> item_ids; std::string previous_item;
    for (const auto& item : bundle.at("items")) {
        exact_fields(item, {"item_id", "disclosure_class", "content_digest", "payload"}, "novelty item");
        require_id(item, "item_id"); const auto id = item.at("item_id").get<std::string>();
        if (!previous_item.empty() && id <= previous_item) invalid("sev.order", "novelty items must be sorted and unique");
        previous_item = id; item_ids.insert(id);
        const auto disclosure = text(item, "disclosure_class", 32U);
        if (disclosure != "public" && disclosure != "internal" && disclosure != "restricted") invalid("sev.novelty_disclosure", "novelty disclosure class is invalid");
        require_digest(item, "content_digest");
        if (engine::tagged_sha256(item.at("payload").dump()) != item.at("content_digest").get<std::string>() ||
            contains_prohibited_novelty_key(item.at("payload"))) invalid("sev.novelty_payload", "novelty item is unbound or contains prohibited material");
    }
    std::string previous_redaction;
    for (const auto& redaction : bundle.at("redactions")) {
        exact_fields(redaction, {"item_id", "json_pointer", "reason"}, "novelty redaction");
        require_id(redaction, "item_id");
        const auto key = redaction.at("item_id").get<std::string>() + "\n" + text(redaction, "json_pointer", 4096U);
        if (!item_ids.contains(redaction.at("item_id").get<std::string>()) ||
            (!previous_redaction.empty() && key <= previous_redaction)) invalid("sev.novelty_redaction", "redactions must reference items and be sorted");
        previous_redaction = key; text(redaction, "reason", 4096U);
    }
    require_digest(bundle, "bundle_digest");
    if (digest_without(bundle, "bundle_digest") != bundle.at("bundle_digest").get<std::string>()) invalid("sev.novelty_digest", "novelty bundle digest mismatch");
    engine::Json result{{"protocol", "symphony.sev.novelty-bundle-check-result.v1"},
        {"bundle_id", bundle.at("bundle_id")}, {"bundle_digest", bundle.at("bundle_digest")},
        {"state", "valid_offline_projection"}, {"item_count", bundle.at("items").size()},
        {"redaction_count", bundle.at("redactions").size()}, {"network_transfer", false},
        {"export_authorized", false}, {"read_only", true}, {"result_digest", nullptr}};
    result["result_digest"] = digest_without(result, "result_digest"); return result;
}

void validate_watch_policy(const engine::Json& policy) {
    exact_fields(policy, {"protocol", "policy_id", "tops_id", "enabled", "session_boundary", "source_scopes",
        "event_classes", "debounce_ms", "coalescing_limit", "thermal_restriction", "export_enabled",
        "generation", "previous_policy_digest", "updated_at", "canonical", "policy_digest"}, "watch policy");
    if (text(policy, "protocol") != "symphony.sev.watch-policy.v1") invalid("sev.protocol", "watch policy protocol is unsupported");
    require_id(policy, "policy_id"); require_id(policy, "tops_id");
    if (!policy.at("policy_id").get<std::string>().starts_with("sevwatch:")) invalid("sev.policy_id", "watch policy identity must use sevwatch namespace");
    if (!policy.at("enabled").is_boolean() || policy.at("export_enabled") != false || policy.at("canonical") != false) invalid("sev.watch_authority", "watch policy authority flags are invalid");
    const auto boundary = text(policy, "session_boundary", 64U);
    if (boundary != "authentication_to_logout_or_reauthentication" && boundary != "explicit_qxctl_policy") invalid("sev.watch_boundary", "watch session boundary is invalid");
    static_cast<void>(ids(policy, "source_scopes", 256U)); static_cast<void>(ids(policy, "event_classes", 256U));
    if (!policy.at("debounce_ms").is_number_unsigned() || policy.at("debounce_ms").get<std::uint64_t>() < 100U ||
        policy.at("debounce_ms").get<std::uint64_t>() > 3600000U ||
        !policy.at("coalescing_limit").is_number_unsigned() || policy.at("coalescing_limit").get<std::uint64_t>() == 0U ||
        policy.at("coalescing_limit").get<std::uint64_t>() > 4096U ||
        text(policy, "thermal_restriction", 32U) != "freezing_only" ||
        !policy.at("generation").is_number_unsigned() || policy.at("generation").get<std::uint64_t>() == 0U ||
        !engine::is_utc_seconds(text(policy, "updated_at", 20U))) invalid("sev.watch_policy", "watch policy bounds or time are invalid");
    if (!policy.at("previous_policy_digest").is_null()) require_digest(policy, "previous_policy_digest");
    if (policy.at("generation") == 1U && !policy.at("previous_policy_digest").is_null()) invalid("sev.watch_lineage", "first watch generation has a predecessor");
    if (policy.at("generation").get<std::uint64_t>() > 1U && policy.at("previous_policy_digest").is_null()) invalid("sev.watch_lineage", "later watch generation lacks a predecessor");
    require_digest(policy, "policy_digest");
    if (digest_without(policy, "policy_digest") != policy.at("policy_digest").get<std::string>()) invalid("sev.watch_digest", "watch policy digest mismatch");
}

engine::Json watch_policy_check(const engine::Json& payload) {
    exact_fields(payload, {"policy"}, "watch policy check payload"); validate_watch_policy(payload.at("policy"));
    engine::Json result{{"protocol", "symphony.sev.watch-policy-check-result.v1"},
        {"policy_id", payload.at("policy").at("policy_id")}, {"policy_digest", payload.at("policy").at("policy_digest")},
        {"enabled", payload.at("policy").at("enabled")}, {"default_disabled", payload.at("policy").at("generation") == 1U && payload.at("policy").at("enabled") == false},
        {"thermal_path", "freezing"}, {"ambient_mutation", false}, {"read_only", true}, {"result_digest", nullptr}};
    result["result_digest"] = digest_without(result, "result_digest"); return result;
}

engine::Json trigger_coalesce(const engine::Json& payload) {
    exact_fields(payload, {"policy", "events"}, "trigger coalescing payload"); validate_watch_policy(payload.at("policy"));
    const auto& policy = payload.at("policy");
    if (policy.at("enabled") != true) invalid("sev.watch_disabled", "disabled watch policy cannot coalesce events");
    const auto& events = payload.at("events");
    if (!events.is_array() || events.empty() || events.size() > policy.at("coalescing_limit").get<std::size_t>()) invalid("sev.watch_events", "watch events exceed policy bound");
    std::string previous; auto digests = engine::Json::array(); std::string first_time, last_time;
    for (const auto& event : events) {
        exact_fields(event, {"event_id", "event_class", "source_scope", "occurred_at", "event_digest"}, "watch event");
        require_id(event, "event_id"); require_id(event, "event_class"); require_id(event, "source_scope");
        const auto occurred = text(event, "occurred_at", 20U);
        if (!engine::is_utc_seconds(occurred)) invalid("sev.timestamp", "watch event time is invalid");
        const auto key = occurred + "\n" + event.at("event_id").get<std::string>();
        if (!previous.empty() && key <= previous) invalid("sev.order", "watch events must be sorted and unique"); previous = key;
        require_digest(event, "event_digest");
        if (digest_without(event, "event_digest") != event.at("event_digest").get<std::string>()) invalid("sev.watch_event_digest", "watch event digest mismatch");
        if (first_time.empty()) first_time = occurred; last_time = occurred; digests.push_back(event.at("event_digest"));
    }
    engine::Json result{{"protocol", "symphony.sev.trigger-coalescing-result.v1"},
        {"policy_digest", policy.at("policy_digest")}, {"first_event_at", first_time}, {"last_event_at", last_time},
        {"event_count", events.size()}, {"event_set_digest", engine::tagged_sha256(digests.dump())},
        {"proposed_case_kind", "encountered_novelty"}, {"case_open_authorized", false},
        {"debounced", true}, {"read_only", true}, {"result_digest", nullptr}};
    result["result_digest"] = digest_without(result, "result_digest"); return result;
}

engine::Json evolution_session_bind(const engine::Json& payload) {
    exact_fields(payload, {"case", "lifecycle_profile_id", "lifecycle_profile_digest",
        "source_report_journal_digest", "desired_state_digest", "direction", "created_at"},
        "evolution session binding payload");
    valid_case(payload.at("case"));
    require_id(payload, "lifecycle_profile_id");
    for (const auto* field : {"lifecycle_profile_digest", "source_report_journal_digest", "desired_state_digest"}) require_digest(payload, field);
    const auto direction = text(payload, "direction", 16U);
    if (direction != "forward" && direction != "reverse") invalid("sev.session_direction", "evolution session direction must be forward or reverse");
    const auto created = text(payload, "created_at", 20U);
    if (!engine::is_utc_seconds(created) || created < payload.at("case").at("updated_at")) invalid("sev.timestamp", "evolution session binding time is invalid");
    engine::Json binding{{"protocol", "symphony.sev.evolution-session-binding.v1"},
        {"case_digest", payload.at("case").at("case_digest")},
        {"source_snapshot_digest", payload.at("case").at("source_snapshot_digest")},
        {"target_digest", payload.at("case").at("target_digest")},
        {"lifecycle_profile_id", payload.at("lifecycle_profile_id")},
        {"lifecycle_profile_digest", payload.at("lifecycle_profile_digest")},
        {"source_report_journal_digest", payload.at("source_report_journal_digest")},
        {"desired_state_digest", payload.at("desired_state_digest")}, {"direction", direction},
        {"created_at", created}, {"canonical", false}, {"read_only", true}, {"apply_authorized", false}, {"binding_digest", nullptr}};
    binding["binding_digest"] = digest_without(binding, "binding_digest");
    return binding;
}

engine::Json project_graph(const engine::Json& payload) {
    exact_fields(payload, {"case", "plan"}, "graph payload"); valid_case(payload.at("case"));
    auto nodes = engine::Json::array();
    engine::Json case_node{{"id", payload.at("case").at("case_id")}, {"kind", "evolution_case"},
        {"state", payload.at("case").at("state")}, {"digest", payload.at("case").at("case_digest")}};
    nodes.push_back(std::move(case_node)); auto edges = engine::Json::array(); engine::Json plan_digest = nullptr;
    if (!payload.at("plan").is_null()) {
        require_digest(payload.at("plan"), "plan_digest"); plan_digest = payload.at("plan").at("plan_digest");
        for (const auto& action : payload.at("plan").at("actions")) {
            valid_action(action); nodes.push_back({{"id", action.at("action_id")}, {"kind", "action"},
                {"state", action.at("execution_class")}, {"digest", action.at("action_digest")}});
        }
        edges = payload.at("plan").at("edges");
    }
    engine::Json result{{"protocol", "symphony.sev.graph-projection.v1"}, {"case_digest", payload.at("case").at("case_digest")},
        {"plan_digest", plan_digest}, {"nodes", std::move(nodes)}, {"edges", std::move(edges)},
        {"noncanonical", true}, {"rebuildable", true}, {"projection_digest", nullptr}};
    result["projection_digest"] = digest_without(result, "projection_digest"); return result;
}

engine::Json compatibility(const engine::Json& payload) {
    exact_fields(payload, {"reader_versions", "writer_version"}, "compatibility payload");
    const auto readers = ids(payload, "reader_versions", 32U); const auto writer = text(payload, "writer_version", 64U);
    const bool supported = writer == "v1" && std::find(readers.begin(), readers.end(), "v1") != readers.end();
    engine::Json result{{"protocol", "symphony.sev.compatibility-result.v1"}, {"writer_version", writer},
        {"reader_versions", readers}, {"compatible", supported}, {"unknown_critical_preserved", !supported},
        {"reason", supported ? "exact_v1_overlap" : "no_supported_overlap"}, {"read_only", true}, {"result_digest", nullptr}};
    result["result_digest"] = digest_without(result, "result_digest"); return result;
}

engine::OperationSpec operation(std::string id, std::string name, std::string interaction,
                                std::string feature, std::optional<std::string> input,
                                std::optional<std::string> output, std::string disposition = "system_orchestrated") {
    static_cast<void>(feature);
    return {std::move(id), std::move(name), "implemented", false, true, {"ssfv:symphony:sev-engine"},
        {std::move(interaction)}, std::move(disposition), std::move(input), std::move(output),
        "read_only", "idempotent", false, "none", "engop:symphony:sev.inspect", "supported", "freezing"};
}

const std::vector<engine::OperationSpec>& operations() {
    static const std::vector<engine::OperationSpec> specs{
        operation("engop:symphony:sev.inspect", "inspect", "inspect", "ssfv:symphony:sev-engine", {}, "symphony.sev.inspect-result.v1", "not_applicable"),
        operation("engop:symphony:sev.case.open", "case_open", "propose", "ssfv:symphony:sev-engine.case", "symphony.sev.case-open-input.v1", "symphony.sev.evolution-case.v1"),
        operation("engop:symphony:sev.impact.assess", "impact_assess", "query", "ssfv:symphony:sev-engine.impact", {}, "symphony.sev.impact-result.v1"),
        operation("engop:symphony:sev.disposition.plan", "disposition_plan", "propose", "ssfv:symphony:sev-engine.disposition", {}, "symphony.sev.disposition-plan.v1"),
        operation("engop:symphony:sev.transition.verify", "transition_verify", "validate", "ssfv:symphony:sev-engine.verification", "symphony.sev.transition-verification-input.v1", "symphony.sev.transition-verification-result.v1"),
        operation("engop:symphony:sev.case.recalculate", "case_recalculate", "recover", "ssfv:symphony:sev-engine.recalculation", {}, "symphony.sev.evolution-case.v1"),
        operation("engop:symphony:sev.case.status", "case_status", "query", "ssfv:symphony:sev-engine.case", {}, "symphony.sev.case-status-result.v1"),
        operation("engop:symphony:sev.case.recover", "case_recover", "recover", "ssfv:symphony:sev-engine.recovery", {}, "symphony.sev.case-recovery-advice.v1"),
        operation("engop:symphony:sev.case.close", "case_close", "propose", "ssfv:symphony:sev-engine.closure", {}, "symphony.sev.case-close-proposal.v1"),
        operation("engop:symphony:sev.command-surface.assess", "command_surface_assess", "validate", "ssfv:symphony:sev-engine.scsev", {}, "symphony.sev.command-surface-assessment.v1"),
        operation("engop:symphony:sev.novelty-bundle.check", "novelty_bundle_check", "validate", "ssfv:symphony:sev-engine", "symphony.sev.novelty-bundle.v1", "symphony.sev.novelty-bundle-check-result.v1"),
        operation("engop:symphony:sev.watch-policy.check", "watch_policy_check", "validate", "ssfv:symphony:sev-engine", "symphony.sev.watch-policy.v1", "symphony.sev.watch-policy-check-result.v1"),
        operation("engop:symphony:sev.trigger.coalesce", "trigger_coalesce", "propose", "ssfv:symphony:sev-engine", {}, "symphony.sev.trigger-coalescing-result.v1"),
        operation("engop:symphony:sev.session.bind", "evolution_session_bind", "propose", "ssfv:symphony:sev-engine", {}, "symphony.sev.evolution-session-binding.v1"),
        operation("engop:symphony:sev.graph.project", "project_graph", "query", "ssfv:symphony:sev-engine.graph", {}, "symphony.sev.graph-projection.v1", "not_applicable"),
        operation("engop:symphony:sev.compatibility", "compatibility", "validate", "ssfv:symphony:sev-engine.compatibility", {}, "symphony.sev.compatibility-result.v1", "not_applicable"),
    };
    return specs;
}

}

engine::Json descriptor() {
    const auto& specs = operations(); engine::validate_operation_specs(specs);
    return engine::Json{{"protocol", engine::descriptor_protocol_v2}, {"module_id", module_id},
        {"engine_id", engine_id}, {"vector_id", vector_id}, {"engine_version", engine_version},
        {"process_protocols", engine::Json::array({engine::process_protocol_v1})},
        {"contract_versions", engine::Json::array({"knowledge/SPEC.md@v1", "knowledge/sev/SPEC.md@v1"})},
        {"operations", engine::legacy_operation_descriptors(specs)},
        {"administration_operations", engine::administration_operation_descriptors(specs)},
        {"operation_registry_digest", engine::operation_registry_digest(specs)},
        {"supported_scopes", engine::Json::array({"user", "host"})}, {"language", "C++26"},
        {"thermal_path", "freezing"}, {"scsev_profile", true}, {"separate_scsev_registry", false},
        {"network_listener", false}, {"canonical_apply_enabled", false}, {"session_mutation_enabled", false},
        {"install_state", "installed_undocked"}, {"default_receptor", "receptor:symphony:knowledge.sev"}};
}

engine::Json handle_request(const engine::Request& request) {
    if (request.operation == "inspect") return inspect(request.payload);
    if (request.operation == "case_open") return case_open(request.payload);
    if (request.operation == "impact_assess") return impact_assess(request.payload);
    if (request.operation == "disposition_plan") return disposition_plan(request.payload);
    if (request.operation == "transition_verify") return transition_verify(request.payload);
    if (request.operation == "case_recalculate") return case_recalculate(request.payload);
    if (request.operation == "case_status") return case_status(request.payload);
    if (request.operation == "case_recover") return case_recover(request.payload);
    if (request.operation == "case_close") return case_close(request.payload);
    if (request.operation == "command_surface_assess") return command_surface_assess(request.payload);
    if (request.operation == "novelty_bundle_check") return novelty_bundle_check(request.payload);
    if (request.operation == "watch_policy_check") return watch_policy_check(request.payload);
    if (request.operation == "trigger_coalesce") return trigger_coalesce(request.payload);
    if (request.operation == "evolution_session_bind") return evolution_session_bind(request.payload);
    if (request.operation == "project_graph") return project_graph(request.payload);
    if (request.operation == "compatibility") return compatibility(request.payload);
    throw engine::Error("operation.unsupported", "operation is reserved or unsupported", 4);
}

}
