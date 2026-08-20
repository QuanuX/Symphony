#include "sev.hpp"
#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include <cassert>
#include <functional>
#include <iostream>

namespace sev = symphony::knowledge::sev;
namespace engine = symphony::knowledge::engine;

namespace {
engine::Request request(std::string operation, engine::Json payload) {
    return {"test", "test", std::move(operation), sev::engine_id, 1, std::move(payload)};
}
engine::Json current() {
    return {{"snapshot_digest", "sha256:0000000000000000000000000000000000000000000000000000000000000000"},
        {"coverage_state", "complete"}, {"sources", engine::Json::array()}};
}

std::string digest_for(const std::string& value) {
    return engine::tagged_sha256(value);
}

void finalize(engine::Json& value, const char* field) {
    value.erase(field);
    value[field] = engine::tagged_sha256(value.dump());
}

engine::Json action(const std::string& id, const std::string& target) {
    engine::Json value{{"action_id", id}, {"operation_id", "qxcmd:symphony:test"},
        {"disposition", "extend_command_surface"}, {"target_id", target},
        {"expected_state_digest", digest_for("expected:" + id)},
        {"authorization_operation", "symphony.sev.external.apply"}, {"audit_required", true},
        {"success_predicate", {{"kind", "always"}, {"source_ids", engine::Json::array()},
            {"pointer", nullptr}, {"expected_json", nullptr}, {"expected_digest", nullptr}}},
        {"recovery_operation_id", "qxcmd:symphony:test.recover"},
        {"execution_class", "proposal_only"}, {"thermal_restriction", "freezing_only"},
        {"action_digest", nullptr}};
    auto copy = value; copy.erase("action_digest");
    value["action_digest"] = engine::tagged_sha256(copy.dump());
    return value;
}

engine::Json impact(const engine::Json& evolution_case) {
    engine::Json value{{"protocol", "symphony.sev.impact-result.v1"},
        {"case_digest", evolution_case.at("case_digest")},
        {"source_snapshot_digest", evolution_case.at("source_snapshot_digest")},
        {"coverage_state", "complete"}, {"affected", engine::Json::array()},
        {"unresolved", engine::Json::array()}, {"conflicts", engine::Json::array()},
        {"complete", true}, {"read_only", true}, {"noncanonical", true},
        {"impact_digest", digest_for("impact")}};
    return value;
}

void require_error(const std::string& code, const std::function<void()>& function) {
    bool rejected = false;
    try { function(); }
    catch (const engine::Error& error) { rejected = error.code() == code; }
    assert(rejected);
}
}

int main() {
    const auto descriptor = sev::descriptor();
    assert(descriptor.at("protocol") == engine::descriptor_protocol_v2);
    assert(descriptor.at("separate_scsev_registry") == false);
    engine::Json input{{"protocol", "symphony.sev.case-open-input.v1"},
        {"case_id", "sevcase:symphony:test"}, {"case_kind", "planned_change"},
        {"source_current", current()}, {"target", {{"version", "v2"}}},
        {"created_at", "2026-08-20T12:00:00Z"}};
    const auto opened = sev::handle_request(request("case_open", input));
    assert(opened.at("state") == "open");
    assert(opened.at("canonical") == false);
    const auto status = sev::handle_request(request("case_status", {{"case", opened}}));
    assert(status.at("case_id") == "sevcase:symphony:test");

    auto first = action("action:01", "surface:qxctl");
    auto second = action("action:02", "surface:manifest");
    engine::Json plan_input{{"case", opened}, {"impact", impact(opened)},
        {"actions", engine::Json::array({first, second})},
        {"edges", engine::Json::array({{{"source", "action:01"}, {"target", "action:02"},
            {"kind", "hard_safety"}}})}, {"blockers", engine::Json::array()}};
    const auto plan = sev::handle_request(request("disposition_plan", plan_input));
    assert(plan.at("state") == "ready" && plan.at("ready_action_ids") == engine::Json::array({"action:01"}));

    auto observed = current(); observed["coverage_state"] = "complete";
    const auto recalculated = sev::handle_request(request("case_recalculate", {
        {"case", opened}, {"plan", plan}, {"observed_current", observed},
        {"completed_action_ids", engine::Json::array({"action:01"})},
        {"failed_action_ids", engine::Json::array()}, {"updated_at", "2026-08-20T12:00:01Z"}}));
    assert(recalculated.at("state") == "ready" &&
           recalculated.at("ready_action_ids") == engine::Json::array({"action:02"}));
    auto successor_plan_input = plan_input;
    successor_plan_input["case"] = recalculated;
    successor_plan_input["impact"] = impact(recalculated);
    const auto successor_plan = sev::handle_request(request("disposition_plan", successor_plan_input));
    const auto converged = sev::handle_request(request("case_recalculate", {
        {"case", recalculated}, {"plan", successor_plan}, {"observed_current", observed},
        {"completed_action_ids", engine::Json::array({"action:01", "action:02"})},
        {"failed_action_ids", engine::Json::array()}, {"updated_at", "2026-08-20T12:00:02Z"}}));
    assert(converged.at("state") == "converged" && converged.at("ready_action_ids").empty());

    const auto blocked = sev::handle_request(request("case_recalculate", {
        {"case", opened}, {"plan", plan}, {"observed_current", observed},
        {"completed_action_ids", engine::Json::array()},
        {"failed_action_ids", engine::Json::array({"action:01"})}, {"updated_at", "2026-08-20T12:00:01Z"}}));
    assert(blocked.at("state") == "blocked" &&
           blocked.at("blocker_ids") == engine::Json::array({"action:01", "action:02"}));

    auto cycle_input = plan_input;
    cycle_input["edges"].push_back({{"source", "action:02"}, {"target", "action:01"},
                                     {"kind", "hard_safety"}});
    require_error("sev.dependency_cycle", [&] {
        static_cast<void>(sev::handle_request(request("disposition_plan", cycle_input)));
    });
    auto invalid_action = first;
    invalid_action["disposition"] = "invented_disposition";
    auto copy = invalid_action; copy.erase("action_digest");
    invalid_action["action_digest"] = engine::tagged_sha256(copy.dump());
    auto invalid_plan = plan_input; invalid_plan["actions"] = engine::Json::array({invalid_action, second});
    require_error("sev.disposition", [&] {
        static_cast<void>(sev::handle_request(request("disposition_plan", invalid_plan)));
    });

    engine::Json policy{{"protocol", "symphony.sev.watch-policy.v1"},
        {"policy_id", "sevwatch:symphony:test"}, {"tops_id", "tops:test"}, {"enabled", true},
        {"session_boundary", "authentication_to_logout_or_reauthentication"},
        {"source_scopes", engine::Json::array({"scope:knowledge"})},
        {"event_classes", engine::Json::array({"event:contract-change"})},
        {"debounce_ms", 1000U}, {"coalescing_limit", 16U}, {"thermal_restriction", "freezing_only"},
        {"export_enabled", false}, {"generation", 1U}, {"previous_policy_digest", nullptr},
        {"updated_at", "2026-08-20T12:00:00Z"}, {"canonical", false}, {"policy_digest", nullptr}};
    finalize(policy, "policy_digest");
    const auto policy_result = sev::handle_request(request("watch_policy_check", {{"policy", policy}}));
    assert(policy_result.at("ambient_mutation") == false && policy_result.at("thermal_path") == "freezing");
    engine::Json event{{"event_id", "event:01"}, {"event_class", "event:contract-change"},
        {"source_scope", "scope:knowledge"}, {"occurred_at", "2026-08-20T12:00:01Z"},
        {"event_digest", nullptr}};
    finalize(event, "event_digest");
    const auto coalesced = sev::handle_request(request("trigger_coalesce", {
        {"policy", policy}, {"events", engine::Json::array({event})}}));
    assert(coalesced.at("event_count") == 1U && coalesced.at("case_open_authorized") == false);

    engine::Json item{{"item_id", "item:01"}, {"disclosure_class", "internal"},
        {"content_digest", nullptr}, {"payload", {{"finding", "new capability"}}}};
    item["content_digest"] = engine::tagged_sha256(item.at("payload").dump());
    engine::Json bundle{{"protocol", "symphony.sev.novelty-bundle.v1"},
        {"bundle_id", "sevnovelty:symphony:test"}, {"case_digest", opened.at("case_digest")},
        {"source_snapshot_digest", opened.at("source_snapshot_digest")}, {"items", engine::Json::array({item})},
        {"redactions", engine::Json::array()}, {"approval_reference", "ssiag:decision:test"},
        {"offline_capable", true}, {"network_transfer", false}, {"created_at", "2026-08-20T12:00:02Z"},
        {"canonical", false}, {"bundle_digest", nullptr}};
    finalize(bundle, "bundle_digest");
    const auto novelty = sev::handle_request(request("novelty_bundle_check", {{"bundle", bundle}}));
    assert(novelty.at("state") == "valid_offline_projection" && novelty.at("export_authorized") == false);
    auto forbidden = bundle;
    forbidden["items"][0]["payload"] = {{"token", "not-allowed"}};
    forbidden["items"][0]["content_digest"] = engine::tagged_sha256(forbidden["items"][0]["payload"].dump());
    finalize(forbidden, "bundle_digest");
    require_error("sev.novelty_payload", [&] {
        static_cast<void>(sev::handle_request(request("novelty_bundle_check", {{"bundle", forbidden}})));
    });

    require_error("operation.unsupported", [&] {
        static_cast<void>(sev::handle_request(request("apply", {})));
    });
    std::cout << "SEV engine tests passed\n";
}
