#include "coordinator.hpp"
#include "lifecycle.hpp"

#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/protocol.hpp"

#include <algorithm>
#include <iostream>
#include <stdexcept>
#include <string>
#include <vector>

namespace session = symphony::knowledge::session;
namespace engine = symphony::knowledge::engine;

namespace {

const std::vector<std::string> capabilities = {
    "dependency-ready-set-v1",
    "deterministic-action-id-v1",
    "forward-inverse-v1",
    "localized-blocker-isolation-v1",
    "ordered-safety-phases-v1",
    "receipt-v1-adapter",
    "receipt-v2",
    "report-only-v1",
    "unknown-critical-block-v1",
};

void require(bool condition, const std::string& message) {
    if (!condition) throw std::runtime_error(message);
}

template <typename Function>
void require_error(Function&& function, const std::string& code) {
    try {
        function();
    } catch (const engine::Error& error) {
        require(error.code() == code, "expected " + code + ", got " + error.code());
        return;
    }
    throw std::runtime_error("expected Error with code " + code);
}

void sort_json(engine::Json& value) {
    std::sort(value.begin(), value.end(), [](const engine::Json& left, const engine::Json& right) {
        return left.dump() < right.dump();
    });
}

std::string receipt(std::string_view identity) {
    return engine::tagged_sha256("receipt:" + std::string(identity));
}

engine::Json dependency(std::string target, std::string condition = "active", bool critical = true) {
    return engine::Json{
        {"target_component_id", std::move(target)},
        {"condition", std::move(condition)},
        {"critical", critical},
    };
}

engine::Json desired_component(
    std::string id,
    std::string selected,
    std::string activation = "active",
    engine::Json dependencies = engine::Json::array(),
    std::string presence = "present",
    std::string receipt_protocol = "symphony.knowledge.install-receipt.v2") {
    engine::Json package = presence == "present" ? engine::Json{
        {"package_id", id + "-package"},
        {"version", "1.0.0"},
        {"receipt_protocol", std::move(receipt_protocol)},
        {"receipt_digest", std::move(selected)},
    } : engine::Json(nullptr);
    return engine::Json{
        {"component_id", id},
        {"component_kind", "module"},
        {"module_id", id},
        {"vector_id", nullptr},
        {"engine_id", nullptr},
        {"presence", presence},
        {"selected_package", std::move(package)},
        {"required", true},
        {"install_scope", "prefix"},
        {"install_root", "/opt/symphony"},
        {"activation", presence == "present" ? activation : "inactive"},
        {"docking", engine::Json{{"disposition", "undocked"}, {"receptor_id", nullptr}}},
        {"dependencies", std::move(dependencies)},
        {"compatibility", engine::Json{
            {"required_capabilities", engine::Json::array()},
            {"platform_requirements", engine::Json::array()},
        }},
        {"extensions", engine::Json::array()},
    };
}

void finalize_desired(engine::Json& desired) {
    for (auto& component : desired.at("components")) {
        sort_json(component.at("dependencies"));
        sort_json(component.at("compatibility").at("required_capabilities"));
        sort_json(component.at("compatibility").at("platform_requirements"));
        sort_json(component.at("extensions"));
    }
    sort_json(desired.at("components"));
    sort_json(desired.at("extensions"));
    auto basis = desired;
    basis.erase("desired_state_digest");
    desired["desired_state_digest"] = engine::tagged_sha256(basis.dump());
}

engine::Json desired_state(engine::Json components) {
    engine::Json desired{
        {"protocol", "symphony.knowledge.lifecycle-desired-state.v1"},
        {"format_version", 1},
        {"profile_id", "default"},
        {"tops_id", "tops-test"},
        {"generation", 1},
        {"previous_desired_state_digest", nullptr},
        {"components", std::move(components)},
        {"extensions", engine::Json::array()},
        {"canonical", false},
        {"desired_state_digest", nullptr},
    };
    finalize_desired(desired);
    return desired;
}

engine::Json observed_component(
    std::string id,
    std::string selected,
    std::string activation = "inactive",
    std::string version = "1.0.0",
    std::string protocol = "symphony.knowledge.install-receipt.v2",
    std::string integrity = "valid",
    bool entry_points_validated = true) {
    engine::Json component{
        {"component_id", id},
        {"component_kind", "module"},
        {"module_id", id},
        {"vector_id", nullptr},
        {"engine_id", nullptr},
        {"packages", engine::Json::array({engine::Json{
            {"package_id", id + "-package"},
            {"version", std::move(version)},
            {"install_root", "/opt/symphony"},
            {"receipt_protocol", std::move(protocol)},
            {"receipt_digest", selected},
            {"integrity", std::move(integrity)},
            {"entry_points_validated", entry_points_validated},
        }})},
        {"selected_package_digest", selected},
        {"activation", std::move(activation)},
        {"docking", "undocked"},
        {"receptor_id", nullptr},
        {"capabilities", engine::Json::array()},
        {"platform_compatibility", "compatible"},
        {"observation_digest", nullptr},
    };
    auto basis = component;
    basis.erase("observation_digest");
    sort_json(basis.at("packages"));
    sort_json(basis.at("capabilities"));
    component["observation_digest"] = engine::tagged_sha256(basis.dump());
    return component;
}

void set_desired_docking(engine::Json& component, std::string disposition, engine::Json receptor) {
    component["docking"] = engine::Json{
        {"disposition", std::move(disposition)},
        {"receptor_id", std::move(receptor)},
    };
}

void require_desired_capability(engine::Json& component, std::string capability) {
    component.at("compatibility").at("required_capabilities").push_back(std::move(capability));
}

void finalize_observed_component(engine::Json& component) {
    auto basis = component;
    basis.erase("observation_digest");
    sort_json(basis.at("packages"));
    sort_json(basis.at("capabilities"));
    component["observation_digest"] = engine::tagged_sha256(basis.dump());
}

void set_observed_docking(engine::Json& component, std::string disposition, engine::Json receptor) {
    component["docking"] = std::move(disposition);
    component["receptor_id"] = std::move(receptor);
    finalize_observed_component(component);
}

void add_observed_capability(engine::Json& component, std::string capability) {
    component.at("capabilities").push_back(std::move(capability));
    sort_json(component.at("capabilities"));
    finalize_observed_component(component);
}

void add_observed_package(engine::Json& component, std::string digest, std::string version) {
    auto package = component.at("packages").at(0);
    package["receipt_digest"] = std::move(digest);
    package["version"] = std::move(version);
    component.at("packages").push_back(std::move(package));
    sort_json(component.at("packages"));
    finalize_observed_component(component);
}

void finalize_observation(engine::Json& observation) {
    auto& platform = observation.at("platform");
    auto platform_basis = platform;
    platform_basis.erase("compatibility_digest");
    sort_json(platform_basis.at("provider_availability"));
    platform["compatibility_digest"] = engine::tagged_sha256(platform_basis.dump());
    sort_json(observation.at("configured_roots"));
    sort_json(observation.at("components"));
    sort_json(observation.at("unknown_packages"));
    auto basis = observation;
    basis.erase("observation_digest");
    sort_json(basis.at("configured_roots"));
    sort_json(basis.at("platform").at("provider_availability"));
    sort_json(basis.at("components"));
    sort_json(basis.at("unknown_packages"));
    observation["observation_digest"] = engine::tagged_sha256(basis.dump());
}

engine::Json observation(engine::Json components) {
    engine::Json observed{
        {"protocol", "symphony.knowledge.lifecycle-observation.v1"},
        {"format_version", 1},
        {"profile_id", "default"},
        {"tops_id", "tops-test"},
        {"configured_roots", engine::Json::array({"/opt/symphony"})},
        {"platform", engine::Json{
            {"os", "linux"},
            {"kernel_abi", "linux-test"},
            {"architecture", "x86_64"},
            {"qxctl_identity", engine::Json{
                {"component_id", "qxctl"},
                {"version", "dev"},
                {"executable_digest", receipt("qxctl")},
            }},
            {"coordinator_identity", engine::Json{
                {"component_id", "symphony-knowledge-session"},
                {"version", "0.1.0-dev"},
                {"executable_digest", receipt("coordinator")},
            }},
            {"provider_availability", engine::Json::array()},
            {"compatibility_digest", nullptr},
        }},
        {"binding_registry_digest", nullptr},
        {"components", std::move(components)},
        {"unknown_packages", engine::Json::array()},
        {"observed_at", "2026-08-04T16:00:00Z"},
        {"canonical", false},
        {"observation_digest", nullptr},
    };
    finalize_observation(observed);
    return observed;
}

engine::Json client(bool full = true) {
    auto values = capabilities;
    if (!full) values.pop_back();
    return engine::Json{
        {"client_id", "qxctl"},
        {"client_version", "dev"},
        {"process_protocols", engine::Json::array({engine::process_protocol_v1})},
        {"desired_state_read_versions", engine::Json::array({1})},
        {"observation_read_versions", engine::Json::array({1})},
        {"plan_read_versions", engine::Json::array({1})},
        {"applied_state_read_versions", engine::Json::array({1})},
        {"receipt_read_versions", engine::Json::array({1, 2})},
        {"capabilities", values},
    };
}

engine::Json plan(const engine::Json& desired, const engine::Json& observed, engine::Json caller = client()) {
    return session::handle_request(engine::Request{
        "lifecycle-request", "lifecycle-correlation", "lifecycle_plan", session::engine_id,
        engine::unix_time_ms() + 60000,
        engine::Json{
            {"protocol", "symphony.knowledge.lifecycle-plan-command.v1"},
            {"operation", "lifecycle_plan"},
            {"desired_state", desired},
            {"observation", observed},
            {"prior_applied_state_digest", nullptr},
            {"client", std::move(caller)},
        },
    });
}

const engine::Json& action(const engine::Json& result, const std::string& component, const std::string& kind) {
    const auto found = std::find_if(result.at("actions").begin(), result.at("actions").end(), [&](const engine::Json& value) {
        return value.at("component_id") == component && value.at("kind") == kind;
    });
    if (found == result.at("actions").end()) {
        throw std::runtime_error("expected lifecycle action was absent: " + component + "/" + kind +
            " actions=" + result.at("actions").dump());
    }
    return *found;
}

bool contains_id(const engine::Json& values, const engine::Json& id) {
    return std::find(values.begin(), values.end(), id) != values.end();
}

void test_descriptor() {
    const auto descriptor = session::descriptor();
    const auto found = std::find_if(descriptor.at("operations").begin(), descriptor.at("operations").end(), [](const engine::Json& operation) {
        return operation.at("name") == "lifecycle_plan";
    });
    require(found != descriptor.at("operations").end(), "descriptor omitted lifecycle_plan");
    require(found->at("availability") == "implemented_report_only", "lifecycle operation availability drift");
    const auto inspect = session::handle_request(engine::Request{
        "inspect", "inspect", "inspect", session::engine_id,
        engine::unix_time_ms() + 60000, engine::Json::object(),
    });
    require(inspect.at("lifecycle").at("dynamic_replanning") == true, "inspect omitted dynamic replanning");
    require(inspect.at("lifecycle").at("action_execution_enabled") == false, "inspect enabled lifecycle actions");
}

void test_dependency_ready_set_and_replanning() {
    const auto a_receipt = receipt("a");
    const auto b_receipt = receipt("b");
    const auto d_receipt = receipt("d");
    const auto desired = desired_state(engine::Json::array({
        desired_component("a", a_receipt, "active", engine::Json::array({dependency("b")})),
        desired_component("b", b_receipt),
        desired_component("d", d_receipt),
    }));

    const auto first_observation = observation(engine::Json::array({
        observed_component("a", a_receipt),
        observed_component("d", d_receipt),
    }));
    const auto first = plan(desired, first_observation);
    const auto& install_b = action(first, "b", "install");
    const auto& activate_a_first = action(first, "a", "activate");
    const auto& activate_d = action(first, "d", "activate");
    require(install_b.at("disposition") == "blocked", "missing dependency package was not blocked");
    require(activate_a_first.at("disposition") == "waiting", "dependent action did not wait");
    require(activate_d.at("disposition") == "ready", "unrelated ready action was stalled");
    const auto reordered = plan(
        desired_state(engine::Json::array({
            desired_component("d", d_receipt),
            desired_component("b", b_receipt),
            desired_component("a", a_receipt, "active", engine::Json::array({dependency("b")})),
        })),
        observation(engine::Json::array({
            observed_component("d", d_receipt), observed_component("a", a_receipt),
        })));
    require(first == reordered, "input array order changed the normalized lifecycle plan");

    const auto second_observation = observation(engine::Json::array({
        observed_component("a", a_receipt),
        observed_component("b", b_receipt),
        observed_component("d", d_receipt, "active"),
    }));
    const auto second = plan(desired, second_observation);
    const auto& activate_b = action(second, "b", "activate");
    const auto& activate_a_second = action(second, "a", "activate");
    require(activate_b.at("disposition") == "ready", "newly observed dependency did not become ready");
    require(activate_a_second.at("disposition") == "waiting", "dependent action ignored new prerequisite");
    require(contains_id(activate_a_second.at("prerequisite_action_ids"), activate_b.at("action_id")),
            "dependent action did not bind dependency action");

    const auto third_observation = observation(engine::Json::array({
        observed_component("a", a_receipt),
        observed_component("b", b_receipt, "active"),
        observed_component("d", d_receipt, "active"),
    }));
    const auto third = plan(desired, third_observation);
    require(action(third, "a", "activate").at("disposition") == "ready",
            "dependent action did not self-heal to ready after evidence changed");
    require(third.at("scheduler").at("algorithm") == "dependency_ready_set_v1", "scheduler drift");
}

void test_cycle_isolation() {
    const auto a_receipt = receipt("cycle-a");
    const auto b_receipt = receipt("cycle-b");
    const auto d_receipt = receipt("cycle-d");
    const auto desired = desired_state(engine::Json::array({
        desired_component("a", a_receipt, "active", engine::Json::array({dependency("b")})),
        desired_component("b", b_receipt, "active", engine::Json::array({dependency("a")})),
        desired_component("d", d_receipt),
    }));
    const auto observed = observation(engine::Json::array({
        observed_component("a", a_receipt), observed_component("b", b_receipt),
        observed_component("d", d_receipt),
    }));
    const auto result = plan(desired, observed);
    require(action(result, "a", "activate").at("disposition") == "blocked", "cycle action a was not blocked");
    require(action(result, "b", "activate").at("disposition") == "blocked", "cycle action b was not blocked");
    require(action(result, "d", "activate").at("disposition") == "ready", "cycle stalled unrelated component");
    require(action(result, "a", "activate").at("blockers").at(0).at("class") == "cycle_detected",
            "cycle blocker was not explicit");
}

void test_dependency_criticality_and_component_capabilities() {
    const auto a_receipt = receipt("criticality-a");
    const auto b_receipt = receipt("criticality-b");

    const auto advisory_desired = desired_state(engine::Json::array({
        desired_component("a", a_receipt, "active",
            engine::Json::array({dependency("optional-service", "active", false)})),
    }));
    const auto advisory = plan(
        advisory_desired,
        observation(engine::Json::array({observed_component("a", a_receipt)})));
    require(action(advisory, "a", "activate").at("disposition") == "ready",
            "noncritical dependency incorrectly blocked convergence");
    require(advisory.at("advisories").size() == 1U,
            "unsatisfied noncritical dependency was not reported");
    require(advisory.at("advisories").at(0).at("target_component_id") == "optional-service",
            "noncritical dependency advisory lost its target identity");

    const auto contradictory = desired_state(engine::Json::array({
        desired_component("a", a_receipt, "active", engine::Json::array({dependency("b")})),
        desired_component("b", b_receipt, "inactive"),
    }));
    const auto contradicted = plan(
        contradictory,
        observation(engine::Json::array({
            observed_component("a", a_receipt),
            observed_component("b", b_receipt, "inactive"),
        })));
    require(action(contradicted, "a", "activate").at("disposition") == "blocked",
            "contradictory hard dependency was presented as healable ordering");

    auto capability_desired = desired_component("capable", a_receipt, "active");
    require_desired_capability(capability_desired, "feature-x");
    const auto missing = plan(
        desired_state(engine::Json::array({capability_desired})),
        observation(engine::Json::array({observed_component("capable", a_receipt)})));
    require(action(missing, "capable", "activate").at("disposition") == "blocked",
            "missing required component capability did not localize a compatibility blocker");

    const auto replacement_receipt = receipt("capability-replacement");
    auto replacement_desired = desired_component("capable", replacement_receipt, "active");
    require_desired_capability(replacement_desired, "feature-x");
    auto old_selected = observed_component("capable", a_receipt, "active");
    add_observed_package(old_selected, replacement_receipt, "2.0.0");
    const auto replacement = plan(
        desired_state(engine::Json::array({replacement_desired})),
        observation(engine::Json::array({old_selected})));
    require(action(replacement, "capable", "select").at("disposition") != "blocked",
            "old selected package capabilities incorrectly blocked exact package replacement");

    auto capable_observed = observed_component("capable", a_receipt);
    add_observed_capability(capable_observed, "feature-x");
    const auto satisfied = plan(
        desired_state(engine::Json::array({capability_desired})),
        observation(engine::Json::array({capable_observed})));
    require(action(satisfied, "capable", "activate").at("disposition") == "ready",
            "verified component capability did not release its localized blocker");
}

void test_two_way_selection_and_determinism() {
    const auto old_receipt = receipt("version-1");
    const auto new_receipt = receipt("version-2");
    auto observed_new = observed_component("module", new_receipt, "active", "99.0.0");
    add_observed_package(observed_new, old_receipt, "0.0.1");
    const auto toward_old = plan(
        desired_state(engine::Json::array({desired_component("module", old_receipt)})),
        observation(engine::Json::array({observed_new})));
    const auto& rollback = action(toward_old, "module", "select");
    require(rollback.at("direction") == "forward", "desired rollback was not a first-class forward convergence action");
    require(!rollback.at("inverse_action_id").is_null(), "selection action omitted inverse relationship");
    const auto repeated = plan(
        desired_state(engine::Json::array({desired_component("module", old_receipt)})),
        observation(engine::Json::array({observed_new})));
    require(toward_old == repeated, "identical lifecycle evidence produced a different plan");

    auto observed_old = observed_component("module", old_receipt, "active", "0.0.1");
    add_observed_package(observed_old, new_receipt, "99.0.0");
    const auto toward_new = plan(
        desired_state(engine::Json::array({desired_component("module", new_receipt)})),
        observation(engine::Json::array({observed_old})));
    require(action(toward_new, "module", "select").at("direction") == "forward",
            "desired upgrade direction depended on semantic version recency");
}

void test_receptor_switch_and_safe_package_sequence() {
    const auto old_receipt = receipt("docked-old");
    const auto new_receipt = receipt("docked-new");

    auto desired_receptor = desired_component("docked", old_receipt, "active");
    set_desired_docking(desired_receptor, "docked", "maestro-receptor-b");
    auto observed_receptor = observed_component("docked", old_receipt, "active");
    set_observed_docking(observed_receptor, "docked", "maestro-receptor-a");
    const auto switched = plan(
        desired_state(engine::Json::array({desired_receptor})),
        observation(engine::Json::array({observed_receptor})));
    const auto& undock = action(switched, "docked", "undock");
    const auto& dock = action(switched, "docked", "dock");
    require(dock.at("target_receptor_id") == "maestro-receptor-b",
            "dock action did not bind the exact desired receptor");
    require(contains_id(dock.at("prerequisite_action_ids"), undock.at("action_id")),
            "receptor switch did not undock before docking the replacement");
    require(undock.at("target_receptor_id").is_null(), "undock action carried a receptor target");
    require(dock.at("target_state_digest").is_string(), "dock action omitted its target-state digest");

    auto desired_package = desired_component("docked", new_receipt, "active");
    set_desired_docking(desired_package, "docked", "maestro-receptor-a");
    auto observed_package = observed_component("docked", old_receipt, "active");
    add_observed_package(observed_package, new_receipt, "2.0.0");
    set_observed_docking(observed_package, "docked", "maestro-receptor-a");
    const auto reselection = plan(
        desired_state(engine::Json::array({desired_package})),
        observation(engine::Json::array({observed_package})));
    const auto& safe_undock = action(reselection, "docked", "undock");
    const auto& deactivate = action(reselection, "docked", "deactivate");
    const auto& select = action(reselection, "docked", "select");
    const auto& activate = action(reselection, "docked", "activate");
    const auto& safe_dock = action(reselection, "docked", "dock");
    require(contains_id(deactivate.at("prerequisite_action_ids"), safe_undock.at("action_id")),
            "package transition did not deactivate after undocking");
    require(contains_id(select.at("prerequisite_action_ids"), deactivate.at("action_id")),
            "package transition selected before deactivation");
    require(contains_id(activate.at("prerequisite_action_ids"), select.at("action_id")),
            "package transition activated before selection");
    require(contains_id(safe_dock.at("prerequisite_action_ids"), activate.at("action_id")),
            "package transition docked before activation completed");
    require(safe_dock.at("target_receptor_id") == "maestro-receptor-a",
            "package transition did not restore the exact receptor");
}

void test_compatibility_integrity_and_digest_fail_closed() {
    const auto component_receipt = receipt("guarded");
    const auto desired = desired_state(engine::Json::array({desired_component("guarded", component_receipt)}));
    const auto observed = observation(engine::Json::array({observed_component("guarded", component_receipt)}));
    const auto blocked = plan(desired, observed, client(false));
    require(blocked.at("compatibility").at("mode") == "blocked", "missing capability did not block compatibility");
    require(blocked.at("actions").empty(), "compatibility-blocked plan emitted actions");
    require(blocked.at("fatal_blockers").at(0).at("class") == "compatibility_blocked",
            "compatibility blocker was not explicit");

    auto unsupported_receipt_client = client();
    unsupported_receipt_client["receipt_read_versions"] = engine::Json::array({3});
    const auto unsupported_receipt = plan(desired, observed, unsupported_receipt_client);
    require(unsupported_receipt.at("compatibility").at("mode") == "blocked",
            "unsupported receipt-reader overlap did not block compatibility");
    require(unsupported_receipt.at("compatibility").at("receipt_versions").empty(),
            "blocked compatibility invented a supported receipt version");
    require(unsupported_receipt.at("actions").empty(),
            "receipt-incompatible plan emitted actions");

    const auto v1_receipt = receipt("legacy-v1");
    auto v1_client = client();
    v1_client["receipt_read_versions"] = engine::Json::array({1});
    auto& v1_capabilities = v1_client.at("capabilities");
    v1_capabilities.erase(std::find(v1_capabilities.begin(), v1_capabilities.end(), "receipt-v2"));
    const auto v1_plan = plan(
        desired_state(engine::Json::array({desired_component(
            "legacy", v1_receipt, "active", engine::Json::array(), "present",
            "symphony.knowledge.install-receipt.v1")})),
        observation(engine::Json::array({observed_component(
            "legacy", v1_receipt, "inactive", "1.0.0",
            "symphony.knowledge.install-receipt.v1")})),
        v1_client);
    require(v1_plan.at("compatibility").at("mode") == "full",
            "v1-only evidence unnecessarily required the v2 receipt reader");
    require(action(v1_plan, "legacy", "activate").at("disposition") == "ready",
            "legacy v1-compatible plan did not remain operational");

    const auto invalid_integrity = observation(engine::Json::array({
        observed_component("guarded", component_receipt, "inactive", "1.0.0",
            "symphony.knowledge.install-receipt.v2", "invalid"),
    }));
    const auto integrity = plan(desired, invalid_integrity);
    require(integrity.at("fatal_blockers").at(0).at("class") == "integrity_fatal",
            "invalid receipt integrity was not fatal");

    auto damaged = desired;
    damaged["desired_state_digest"] = receipt("wrong");
    require_error([&] { static_cast<void>(plan(damaged, observed)); }, "lifecycle.digest_mismatch");

    auto malformed_receptor = observed;
    malformed_receptor.at("components").at(0)["docking"] = "docked";
    finalize_observation(malformed_receptor);
    require_error([&] { static_cast<void>(plan(desired, malformed_receptor)); }, "lifecycle.invalid_field");

    auto unknown_field = desired;
    unknown_field["unexpected"] = true;
    require_error([&] { static_cast<void>(plan(unknown_field, observed)); }, "lifecycle.field_set");

    require_error([&] {
        static_cast<void>(session::handle_lifecycle_plan(engine::Request{
            "expired", "expired", "lifecycle_plan", session::engine_id,
            engine::unix_time_ms() - 1,
            engine::Json{
                {"protocol", "symphony.knowledge.lifecycle-plan-command.v1"},
                {"operation", "lifecycle_plan"},
                {"desired_state", desired},
                {"observation", observed},
                {"prior_applied_state_digest", nullptr},
                {"client", client()},
            },
        }));
    }, "request.deadline_expired");
}

void test_bounded_scale_plan() {
    engine::Json desired_components = engine::Json::array();
    engine::Json observed_components = engine::Json::array();
    constexpr std::size_t component_count = 512U;
    for (std::size_t index = 0; index < component_count; ++index) {
        const auto id = "scale-" + std::to_string(index);
        const auto component_receipt = receipt(id);
        desired_components.push_back(desired_component(id, component_receipt));
        observed_components.push_back(observed_component(id, component_receipt));
    }
    const auto result = plan(
        desired_state(std::move(desired_components)),
        observation(std::move(observed_components)));
    require(result.at("actions").size() == component_count,
            "bounded scale plan omitted component actions");
    require(result.at("ready_action_ids").size() == component_count,
            "bounded scale plan did not expose every independent action as ready");
    require(result.dump().size() <= 4U * 1024U * 1024U,
            "bounded scale plan exceeded the process response ceiling");
}

}

int main() {
    try {
        test_descriptor();
        test_dependency_ready_set_and_replanning();
        test_cycle_isolation();
        test_dependency_criticality_and_component_capabilities();
        test_two_way_selection_and_determinism();
        test_receptor_switch_and_safe_package_sequence();
        test_compatibility_integrity_and_digest_fail_closed();
        test_bounded_scale_plan();
        std::cout << "knowledge lifecycle planner tests passed\n";
        return 0;
    } catch (const std::exception& error) {
        std::cerr << "test failure: " << error.what() << '\n';
        return 1;
    }
}
