#include "coordinator.hpp"
#include "lifecycle.hpp"

#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/protocol.hpp"

#include <algorithm>
#include <array>
#include <filesystem>
#include <fstream>
#include <iostream>
#include <stdexcept>
#include <string>
#include <sys/stat.h>
#include <unistd.h>
#include <vector>

namespace fs = std::filesystem;
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

const std::vector<std::string> journal_capabilities = {
    "atomic-head-v1",
    "dual-slot-journal-v1",
    "dynamic-replanning-v1",
    "expected-state-cas-v1",
    "idempotent-operation-v1",
    "opaque-extension-preservation-v1",
    "recovery-forward-v1",
    "report-only-v1",
};

const std::vector<std::string> apply_capabilities = {
    "action-attempt-journal-v2", "applied-state-v1", "dynamic-replanning-v1",
    "expected-state-cas-v1", "external-action-adapter-v1", "forward-inverse-v1",
    "opaque-extension-preservation-v1", "per-action-authorization-v1",
    "recovery-forward-v1", "verified-observation-commit-v1",
};

class TemporaryDirectory final {
public:
    TemporaryDirectory() {
        std::string pattern = (fs::canonical(fs::temp_directory_path()) /
            "symphony-lifecycle-journal-test-XXXXXX").string();
        pattern.push_back('\0');
        char* result = ::mkdtemp(pattern.data());
        if (result == nullptr) throw std::runtime_error("mkdtemp failed");
        path_ = result;
    }
    ~TemporaryDirectory() {
        std::error_code ignored;
        fs::remove_all(path_, ignored);
    }
    TemporaryDirectory(const TemporaryDirectory&) = delete;
    TemporaryDirectory& operator=(const TemporaryDirectory&) = delete;
    [[nodiscard]] const fs::path& path() const { return path_; }
private:
    fs::path path_;
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

std::string lifecycle_resource(const std::string& evidence) {
    return "symphony.knowledge.lifecycle:" +
        engine::sha256_hex("tops-test\ndefault\n" + evidence);
}

engine::Json lifecycle_authorization(
    const std::string& operation,
    const std::string& evidence,
    const std::string& suffix) {
    const auto request_id = "lifecycle-request-" + suffix;
    const auto correlation_id = "lifecycle-correlation-" + suffix;
    const auto target = engine::Json{
        {"operation", "symphony.knowledge.lifecycle." + operation},
        {"resource", lifecycle_resource(evidence)},
        {"audience", "qxctl"},
        {"scope", "tops:tops-test"},
    };
    const auto subject = engine::Json{
        {"id", "owner.primary"}, {"kind", "symphony.identity.owner"},
        {"authority", "unix_peer_credentials"},
    };
    engine::Json capability{
        {"protocol", "symphony.ssiag.capability.v1"}, {"capability_id", "pending"},
        {"subject", subject}, {"tops_id", "tops-test"}, {"target", target},
        {"authority_basis", "host_owner"}, {"grant_id", "lifecycle-" + suffix},
        {"request_id", request_id}, {"correlation_id", correlation_id},
        {"issued_at", "2020-01-01T00:00:00Z"}, {"expires_at", "2099-01-01T00:00:00Z"},
        {"policy_digest", "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
        {"config_digest", "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
        {"binding_digest", "pending"}, {"transferable", false}, {"canonical_apply", false},
    };
    const auto values = std::array<std::string, 19>{
        capability.at("protocol"), subject.at("id"), subject.at("kind"), subject.at("authority"),
        "tops-test", target.at("operation"), target.at("resource"), target.at("audience"), target.at("scope"),
        capability.at("authority_basis"), capability.at("grant_id"), request_id, correlation_id,
        capability.at("issued_at"), capability.at("expires_at"), capability.at("policy_digest"),
        capability.at("config_digest"), "transferable=false", "canonical_apply=false",
    };
    std::string joined;
    for (std::size_t index = 0; index < values.size(); ++index) {
        if (index != 0U) joined.push_back('\n');
        joined += values[index];
    }
    capability["binding_digest"] = engine::tagged_sha256(joined);
    capability["capability_id"] =
        "ssiag-capability:" + capability.at("binding_digest").get<std::string>().substr(7);
    return engine::Json{
        {"schema", "symphony.ssiag.authorization-decision.v1"},
        {"decision_id", "ssiag-decision:" + engine::sha256_hex(operation + suffix)},
        {"request_id", request_id}, {"correlation_id", correlation_id}, {"tops_id", "tops-test"},
        {"subject", subject}, {"target", target}, {"effect", "allow"},
        {"reason_code", "symphony.ssiag.policy.exact-grant"}, {"authority_basis", "host_owner"},
        {"capability", capability}, {"policy_digest", capability.at("policy_digest")},
        {"config_digest", capability.at("config_digest")}, {"decided_at", capability.at("issued_at")},
        {"expires_at", capability.at("expires_at")}, {"caller_class_used", false},
        {"canonical_apply", false},
    };
}

engine::Json journal_client(bool full = true) {
    auto values = journal_capabilities;
    if (!full) values.pop_back();
    return engine::Json{
        {"client_id", "qxctl"}, {"client_version", "dev"},
        {"process_protocols", engine::Json::array({engine::process_protocol_v1})},
        {"journal_read_versions", engine::Json::array({1})},
        {"journal_write_versions", engine::Json::array({1})},
        {"capabilities", values},
    };
}

engine::Request lifecycle_boot_request(
    const fs::path& state_root,
    const engine::Json& desired,
    const engine::Json& observed,
    const std::string& operation_id,
    const std::string& expected,
    const std::string& suffix,
    const std::string& mode = "report",
    engine::Json prior_applied = nullptr) {
    const auto stable = session::lifecycle_stable_inventory_digest(
        observed, engine::unix_time_ms() + 60000);
    const auto profile_digest = engine::tagged_sha256("profile:" + desired.at("desired_state_digest").get<std::string>());
    return engine::Request{
        "boot-" + suffix, "boot-correlation-" + suffix, "lifecycle_boot", session::engine_id,
        engine::unix_time_ms() + 60000,
        engine::Json{
            {"protocol", "symphony.knowledge.lifecycle-boot-command.v1"}, {"operation", "lifecycle_boot"},
            {"state_root", fs::absolute(state_root).lexically_normal().string()}, {"operation_id", operation_id},
            {"expected_journal_digest", expected}, {"profile_id", "default"}, {"tops_id", "tops-test"},
            {"profile_digest", profile_digest}, {"stable_inventory_digest", stable}, {"mode", mode},
            {"desired_state", desired}, {"observation", observed},
            {"prior_applied_state_digest", std::move(prior_applied)},
            {"authorization_decision", lifecycle_authorization(
                "boot", profile_digest + "\n" +
                    desired.at("desired_state_digest").get<std::string>() + "\n" + mode + "\n" + stable,
                suffix)},
            {"planner_client", client()}, {"journal_client", journal_client()},
        },
    };
}

engine::Json lifecycle_boot(
    const fs::path& state_root,
    const engine::Json& desired,
    const engine::Json& observed,
    const std::string& operation_id,
    const std::string& expected,
    const std::string& suffix,
    const std::string& mode = "report",
    engine::Json prior_applied = nullptr) {
    return session::handle_request(lifecycle_boot_request(
        state_root, desired, observed, operation_id, expected, suffix, mode, std::move(prior_applied)));
}

engine::Json lifecycle_boot_state(
    const fs::path& state_root,
    const std::string& operation,
    const engine::Json& operation_id,
    const engine::Json& expected,
    const std::string& suffix,
    bool full = true) {
    const auto evidence = operation == "lifecycle_boot_status" ? "status" : expected.get<std::string>();
    const auto permission = operation == "lifecycle_boot_status" ? "boot.status" : "boot.recover";
    return session::handle_request(engine::Request{
        "state-" + suffix, "state-correlation-" + suffix, operation, session::engine_id,
        engine::unix_time_ms() + 60000,
        engine::Json{
            {"protocol", "symphony.knowledge.lifecycle-boot-command.v1"}, {"operation", operation},
            {"state_root", fs::absolute(state_root).lexically_normal().string()}, {"operation_id", operation_id},
            {"expected_journal_digest", expected}, {"profile_id", "default"}, {"tops_id", "tops-test"},
            {"authorization_decision", lifecycle_authorization(permission, evidence, suffix)},
            {"journal_client", journal_client(full)},
        },
    });
}

fs::path lifecycle_stream_path(const fs::path& state_root) {
    return state_root / "symphony" / "knowledge-session-coordinator" / "lifecycle" / "v1" /
        "tops" / engine::sha256_hex("tops:tops-test") / "profiles" /
        engine::sha256_hex("profile:default");
}

fs::path lifecycle_apply_stream_path(const fs::path& state_root) {
    return state_root / "symphony" / "knowledge-session-coordinator" / "lifecycle" / "v2" /
        "tops" / engine::sha256_hex("tops:tops-test") / "profiles" /
        engine::sha256_hex("profile:default");
}

engine::Json apply_client(bool full = true) {
    auto values = apply_capabilities;
    if (!full) values.pop_back();
    return engine::Json{
        {"client_id", "qxctl"}, {"client_version", "dev"},
        {"process_protocols", engine::Json::array({engine::process_protocol_v1})},
        {"journal_read_versions", engine::Json::array({1, 2})},
        {"journal_write_versions", engine::Json::array({2})},
        {"capabilities", values},
    };
}

engine::Json lifecycle_apply_mutation(
    const fs::path& state_root,
    const std::string& operation,
    const engine::Json& desired,
    const engine::Json& observed,
    const std::string& source_journal_digest,
    const std::string& expected_journal,
    const std::string& expected_applied,
    const engine::Json& action_id,
    const engine::Json& outcome,
    const engine::Json& blocker_class,
    const engine::Json& execution_digest,
    const std::string& suffix,
    engine::Json prior_applied = nullptr,
    engine::Json available = engine::Json::array()) {
    const auto stable = session::lifecycle_stable_inventory_digest(
        observed, engine::unix_time_ms() + 60000);
    const auto profile_digest = engine::tagged_sha256("profile:" + desired.at("desired_state_digest").get<std::string>());
    const auto phase = operation == "lifecycle_apply_prepare" ? "prepare" :
        (operation == "lifecycle_apply_close" ? "close" : "finalize");
    const auto action_evidence = action_id.is_string() ? action_id.get<std::string>() : "converged";
    const auto evidence = profile_digest + "\n" + desired.at("desired_state_digest").get<std::string>() +
        "\n" + stable + "\n" + source_journal_digest + "\n" + expected_journal + "\n" + action_evidence;
    return session::handle_request(engine::Request{
        "apply-" + suffix, "apply-correlation-" + suffix, operation, session::engine_id,
        engine::unix_time_ms() + 60000,
        engine::Json{
            {"protocol", "symphony.knowledge.lifecycle-apply-command.v1"}, {"operation", operation},
            {"state_root", fs::absolute(state_root).lexically_normal().string()},
            {"operation_id", "apply-operation-" + suffix}, {"expected_journal_digest", expected_journal},
            {"expected_applied_state_digest", expected_applied},
            {"source_report_journal_digest", source_journal_digest}, {"profile_id", "default"},
            {"tops_id", "tops-test"}, {"profile_digest", profile_digest}, {"stable_inventory_digest", stable},
            {"desired_state", desired}, {"observation", observed},
            {"prior_applied_state_digest", std::move(prior_applied)}, {"action_id", action_id},
            {"available_artifact_digests", std::move(available)}, {"outcome", outcome},
            {"blocker_class", blocker_class}, {"execution_evidence_digest", execution_digest},
            {"authorization_decision", lifecycle_authorization("apply." + std::string(phase), evidence, suffix)},
            {"planner_client", client()}, {"journal_client", apply_client()},
        },
    });
}

engine::Json lifecycle_apply_state(
    const fs::path& state_root,
    const std::string& operation,
    const engine::Json& operation_id,
    const engine::Json& expected,
    const std::string& suffix,
    bool full = true) {
    const auto status = operation == "lifecycle_apply_status";
    const auto evidence = status ? "status" : expected.get<std::string>();
    return session::handle_request(engine::Request{
        "apply-state-" + suffix, "apply-state-correlation-" + suffix, operation, session::engine_id,
        engine::unix_time_ms() + 60000,
        engine::Json{
            {"protocol", "symphony.knowledge.lifecycle-apply-command.v1"}, {"operation", operation},
            {"state_root", fs::absolute(state_root).lexically_normal().string()}, {"operation_id", operation_id},
            {"expected_journal_digest", expected}, {"profile_id", "default"}, {"tops_id", "tops-test"},
            {"authorization_decision", lifecycle_authorization(
                status ? "apply.status" : "apply.recover", evidence, suffix)},
            {"journal_client", apply_client(full)},
        },
    });
}

engine::Json read_json(const fs::path& path) {
    std::ifstream input(path, std::ios::binary);
    if (!input) throw std::runtime_error("could not read lifecycle test JSON");
    return engine::Json::parse(input);
}

void write_json(const fs::path& path, const engine::Json& value) {
    std::ofstream output(path, std::ios::binary | std::ios::trunc);
    if (!output) throw std::runtime_error("could not write lifecycle test JSON");
    output << value.dump() << '\n';
    output.close();
    if (::chmod(path.c_str(), 0600) != 0) throw std::runtime_error("could not protect lifecycle test JSON");
}

void write_private_text(const fs::path& path, const std::string& value) {
    std::ofstream output(path, std::ios::binary | std::ios::trunc);
    if (!output) throw std::runtime_error("could not write lifecycle test text");
    output << value;
    output.close();
    if (::chmod(path.c_str(), 0600) != 0) throw std::runtime_error("could not protect lifecycle test text");
}

void finalize_test_digest(engine::Json& value, const char* field) {
    value.erase(field);
    value[field] = engine::tagged_sha256(value.dump());
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
    require(inspect.at("lifecycle_journal").at("persistence_enabled") == true,
            "inspect omitted lifecycle journal persistence");
    require(inspect.at("lifecycle_journal").at("action_execution_enabled") == false &&
            inspect.at("lifecycle_journal").at("external_action_coordination_enabled") == true &&
            inspect.at("lifecycle_journal").at("host_action_execution_enabled") == false &&
            inspect.at("lifecycle_journal").at("applied_state_persistence_enabled") == true &&
            inspect.at("lifecycle_journal").at("apply_authorized") == false,
            "lifecycle apply capability or external-authorization boundary drifted");
    const auto close = std::find_if(
        descriptor.at("operations").begin(), descriptor.at("operations").end(), [](const engine::Json& operation) {
            return operation.at("name") == "lifecycle_apply_close";
        });
    require(close != descriptor.at("operations").end(), "descriptor omitted lifecycle_apply_close");
}

void test_apply_journal_prepare_resume_finalize_close_and_recovery() {
    TemporaryDirectory state;
    const auto package = receipt("apply-action");
    const auto desired_active = desired_state(engine::Json::array({
        desired_component("apply-action", package, "active"),
    }));
    const auto observed_inactive = observation(engine::Json::array({
        observed_component("apply-action", package, "inactive"),
    }));
    const auto report = lifecycle_boot(
        state.path(), desired_active, observed_inactive, "apply-source", "absent", "apply-source",
        "apply-compatible");
    const auto source_digest = report.at("journal_digest").get<std::string>();
    const auto planned = plan(desired_active, observed_inactive);
    const auto action_id = action(planned, "apply-action", "activate").at("action_id");

    const auto prepared = lifecycle_apply_mutation(
        state.path(), "lifecycle_apply_prepare", desired_active, observed_inactive,
        source_digest, "absent", "absent", action_id, nullptr, nullptr, nullptr, "prepare");
    require(prepared.at("journal").at("state") == "acting" && prepared.at("action") ==
            prepared.at("journal").at("active_action") && prepared.at("changed") == true,
            "apply prepare did not durably select one exact action");
    const auto prepared_digest = prepared.at("journal_digest").get<std::string>();
    const auto status = lifecycle_apply_state(
        state.path(), "lifecycle_apply_status", nullptr, nullptr, "prepared-status");
    require(status.at("journal_digest") == prepared_digest &&
            status.at("action") == status.at("journal").at("active_action") &&
            status.at("action").at("action_id") == action_id,
            "apply status did not preserve the prepared action");

    const auto replay = lifecycle_apply_mutation(
        state.path(), "lifecycle_apply_prepare", desired_active, observed_inactive,
        source_digest, prepared_digest, "absent", action_id, nullptr, nullptr, nullptr, "prepare");
    require(replay.at("changed") == false && replay.at("journal_digest") == prepared_digest,
            "idempotent apply prepare replay changed state");

    const auto mismatched_desired = desired_state(engine::Json::array({
        desired_component("apply-action", package, "inactive"),
    }));
    require_error([&] {
        static_cast<void>(lifecycle_apply_mutation(
            state.path(), "lifecycle_apply_finalize", mismatched_desired, observed_inactive,
            source_digest, prepared_digest, "absent", action_id, "committed", nullptr,
            receipt("mismatched-profile-evidence"), "mismatched-profile"));
    }, "lifecycle_apply.prepared_evidence_mismatch");
    require_error([&] {
        static_cast<void>(lifecycle_apply_mutation(
            state.path(), "lifecycle_apply_finalize", desired_active, observed_inactive,
            receipt("different-source-journal"), prepared_digest, "absent", action_id, "committed", nullptr,
            receipt("mismatched-source-evidence"), "mismatched-source"));
    }, "lifecycle_apply.prepared_evidence_mismatch");
    require_error([&] {
        static_cast<void>(lifecycle_apply_mutation(
            state.path(), "lifecycle_apply_finalize", desired_active, observed_inactive,
            source_digest, prepared_digest, "absent", action_id, "committed", nullptr,
            receipt("mismatched-prior-evidence"), "mismatched-prior", receipt("unexpected-prior")));
    }, "lifecycle_apply.command_invalid");
    require_error([&] {
        static_cast<void>(lifecycle_apply_mutation(
            state.path(), "lifecycle_apply_finalize", desired_active, observation(engine::Json::array()),
            source_digest, prepared_digest, "absent", action_id, "committed", nullptr,
            receipt("unverified-action-evidence"), "unverified-action"));
    }, "lifecycle_apply.verification_failed");

    const auto apply_directory = lifecycle_apply_stream_path(state.path());
    write_json(apply_directory / "head.json", engine::Json{{"broken", true}});
    require_error([&] {
        static_cast<void>(lifecycle_apply_state(
            state.path(), "lifecycle_apply_status", nullptr, nullptr, "damaged-apply-status"));
    }, "lifecycle_journal.field_set");
    const auto recovered = lifecycle_apply_state(
        state.path(), "lifecycle_apply_recover", "apply-recovery", "discover", "apply-recovery");
    require(recovered.at("recovered") == true && recovered.at("journal").at("state") == "acting" &&
            recovered.at("action").at("action_id") == action_id,
            "apply recovery did not preserve the in-flight exact action");
    auto recovered_digest = recovered.at("journal_digest").get<std::string>();

    const auto failed_attempt = lifecycle_apply_mutation(
        state.path(), "lifecycle_apply_finalize", desired_active, observed_inactive,
        source_digest, recovered_digest, "absent", action_id, "failed", "observation_retryable",
        receipt("retryable-execution-evidence"), "retryable-finalize");
    const auto retried_prepare = lifecycle_apply_mutation(
        state.path(), "lifecycle_apply_prepare", desired_active, observed_inactive,
        source_digest, failed_attempt.at("journal_digest"), "absent", action_id,
        nullptr, nullptr, nullptr, "retryable-prepare");
    const auto recovered_head = read_json(apply_directory / "head.json");
    const auto damaged_slot = recovered_head.at("active_slot").get<int>();
    write_private_text(apply_directory / ("journal." + std::to_string(damaged_slot) + ".json"), "{malformed");
    require_error([&] {
        static_cast<void>(lifecycle_apply_state(
            state.path(), "lifecycle_apply_status", nullptr, nullptr, "malformed-apply-status"));
    }, "lifecycle_apply.head_slot_mismatch");
    const auto slot_recovered = lifecycle_apply_state(
        state.path(), "lifecycle_apply_recover", "apply-slot-recovery", "discover", "apply-slot-recovery");
    require(slot_recovered.at("recovered") == true && slot_recovered.at("journal").at("state") == "blocked" &&
            slot_recovered.at("action").is_null(),
            "apply recovery did not replace a malformed selected slot from its valid predecessor");
    const auto resumed_after_slot_recovery = lifecycle_apply_mutation(
        state.path(), "lifecycle_apply_prepare", desired_active, observed_inactive,
        source_digest, slot_recovered.at("journal_digest"), "absent", action_id,
        nullptr, nullptr, nullptr, "slot-recovery-resume");
    require(resumed_after_slot_recovery.at("journal").at("state") == "acting" &&
            resumed_after_slot_recovery.at("action").at("action_id") == action_id,
            "recovered predecessor could not resume the exact lifecycle action");
    recovered_digest = resumed_after_slot_recovery.at("journal_digest").get<std::string>();
    require(retried_prepare.at("journal_digest") != recovered_digest,
            "malformed selected slot was not replaced by a forward recovery chain");

    const auto observed_active = observation(engine::Json::array({
        observed_component("apply-action", package, "active"),
    }));
    const auto evidence = receipt("execution-evidence");
    const auto finalized = lifecycle_apply_mutation(
        state.path(), "lifecycle_apply_finalize", desired_active, observed_active,
        source_digest, recovered_digest, "absent", action_id, "committed", nullptr, evidence, "finalize");
    require(finalized.at("journal").at("state") == "closed" && finalized.at("applied_state").is_object() &&
            finalized.at("journal").at("applied_state_digest") ==
                finalized.at("applied_state").at("applied_state_digest"),
            "verified lifecycle action did not close with applied evidence");
    const auto finalized_digest = finalized.at("journal_digest").get<std::string>();
    const auto applied_digest = finalized.at("applied_state").at("applied_state_digest").get<std::string>();
    const auto applied_path = lifecycle_apply_stream_path(state.path()) /
        ("applied." + applied_digest.substr(7) + ".json");
    require(fs::exists(applied_path), "applied evidence was not content-addressed beside the journal");
    require_error([&] {
        static_cast<void>(lifecycle_apply_mutation(
            state.path(), "lifecycle_apply_prepare", desired_active, observed_active,
            source_digest, prepared_digest, applied_digest, action_id, nullptr, nullptr, nullptr, "stale",
            applied_digest));
    }, "lifecycle_apply.expected_state_mismatch");

    const auto desired_next = desired_state(engine::Json::array({
        desired_component("apply-action", package, "inactive"),
    }));
    const auto next_report = lifecycle_boot(
        state.path(), desired_next, observed_active, "next-source", source_digest,
        "next-source", "apply-compatible", applied_digest);
    const auto next_plan = plan(desired_next, observed_active);
    const auto next_action = action(next_plan, "apply-action", "deactivate").at("action_id");
    const auto next_prepared = lifecycle_apply_mutation(
        state.path(), "lifecycle_apply_prepare", desired_next, observed_active,
        next_report.at("journal_digest"), finalized_digest, applied_digest, next_action,
        nullptr, nullptr, nullptr, "next-prepare", applied_digest);
    const auto next_status = lifecycle_apply_state(
        state.path(), "lifecycle_apply_status", nullptr, nullptr, "next-status");
    require(next_status.at("journal_digest") == next_prepared.at("journal_digest") &&
            next_status.at("applied_state").at("applied_state_digest") == applied_digest,
            "new transaction could not retain and read prior applied evidence while acting");

    TemporaryDirectory converged_state;
    const auto desired_inactive = desired_state(engine::Json::array({
        desired_component("already-converged", receipt("already-converged"), "inactive"),
    }));
    const auto observed_converged = observation(engine::Json::array({
        observed_component("already-converged", receipt("already-converged"), "inactive"),
    }));
    const auto converged_report = lifecycle_boot(
        converged_state.path(), desired_inactive, observed_converged,
        "converged-source", "absent", "converged-source", "apply-compatible");
    const auto closed = lifecycle_apply_mutation(
        converged_state.path(), "lifecycle_apply_close", desired_inactive, observed_converged,
        converged_report.at("journal_digest"), "absent", "absent", nullptr, nullptr, nullptr, nullptr,
        "close");
    require(closed.at("journal").at("state") == "closed" && closed.at("changed") == true,
            "already-converged lifecycle state could not commit applied evidence");
    const auto closed_digest = closed.at("journal_digest").get<std::string>();
    const auto closed_applied = closed.at("applied_state").at("applied_state_digest").get<std::string>();
    const auto close_retry = lifecycle_apply_mutation(
        converged_state.path(), "lifecycle_apply_close", desired_inactive, observed_converged,
        converged_report.at("journal_digest"), closed_digest, closed_applied, nullptr, nullptr, nullptr, nullptr,
        "close-retry", closed_applied);
    require(close_retry.at("changed") == false && close_retry.at("journal_digest") == closed_digest,
            "semantic close retry churned applied state");

    auto incompatible = apply_client(false);
    static_cast<void>(incompatible);
    require(finalized_digest != prepared_digest, "finalize did not advance the apply journal");
}

void test_apply_prepare_accepts_only_exact_staged_install_evidence() {
    TemporaryDirectory state;
    const auto package = receipt("staged-install");
    const auto desired = desired_state(engine::Json::array({
        desired_component("staged-install", package, "inactive"),
    }));
    const auto observed = observation(engine::Json::array());
    const auto report = lifecycle_boot(
        state.path(), desired, observed, "staged-install-source", "absent",
        "staged-install-source", "apply-compatible");
    const auto planned = plan(desired, observed);
    const auto& install = action(planned, "staged-install", "install");
    require(install.at("disposition") == "blocked" && install.at("blockers").size() == 1U,
            "absent package did not produce the isolated staged-install blocker");
    const auto prepared = lifecycle_apply_mutation(
        state.path(), "lifecycle_apply_prepare", desired, observed,
        report.at("journal_digest"), "absent", "absent", install.at("action_id"),
        nullptr, nullptr, nullptr, "staged-install-prepare", nullptr,
        engine::Json::array({package}));
    require(prepared.at("journal").at("state") == "acting" &&
            prepared.at("action").at("kind") == "install",
            "exact staged receipt did not satisfy only the package-absence blocker");
}

void test_durable_boot_journal_replanning_and_recovery() {
    TemporaryDirectory state;
    TemporaryDirectory unsafe_state;
    if (::chmod(unsafe_state.path().c_str(), 0777) != 0) {
        throw std::runtime_error("could not create unsafe lifecycle state fixture");
    }
    require_error([&] {
        static_cast<void>(lifecycle_boot_state(
            unsafe_state.path(), "lifecycle_boot_status", nullptr, nullptr, "unsafe-status"));
    }, "lifecycle_journal.state_directory_unsafe");
    const auto absent_root = state.path() / "absent-state-root";
    const auto absent = lifecycle_boot_state(
        absent_root, "lifecycle_boot_status", nullptr, nullptr, "absent-status");
    require(absent.at("journal_present") == false && absent.at("read_only") == true &&
            !fs::exists(absent_root),
            "read-only lifecycle status created an absent state root");
    TemporaryDirectory partial_state;
    const auto partial_stream = lifecycle_stream_path(partial_state.path());
    fs::create_directories(partial_stream);
    const auto partial = lifecycle_boot_state(
        partial_state.path(), "lifecycle_boot_status", nullptr, nullptr, "partial-status");
    require(partial.at("journal_present") == false && !fs::exists(partial_stream / ".lock"),
            "read-only lifecycle status created a missing stream lock");
    const auto symlink_target = state.path() / "symlink-target";
    const auto symlink_root = state.path() / "symlink-root";
    fs::create_directories(symlink_target);
    fs::create_directory_symlink(symlink_target, symlink_root);
    require_error([&] {
        static_cast<void>(lifecycle_boot_state(
            symlink_root, "lifecycle_boot_status", nullptr, nullptr, "symlink-status"));
    }, "lifecycle_journal.state_directory_open_failed");
    const auto package = receipt("durable-a");
    const auto desired_inactive = desired_state(engine::Json::array({
        desired_component("durable-a", package, "inactive"),
    }));
    const auto observed_inactive = observation(engine::Json::array({
        observed_component("durable-a", package, "inactive"),
    }));
    auto wrong_inventory = lifecycle_boot_request(
        state.path(), desired_inactive, observed_inactive,
        "boot-operation-wrong-inventory", "absent", "wrong-inventory");
    wrong_inventory.payload["stable_inventory_digest"] = receipt("wrong-inventory");
    require_error([&] {
        static_cast<void>(session::handle_request(wrong_inventory));
    }, "lifecycle_journal.inventory_digest_mismatch");
    auto wrong_target = lifecycle_boot_request(
        state.path(), desired_inactive, observed_inactive,
        "boot-operation-wrong-target", "absent", "wrong-target");
    wrong_target.payload["authorization_decision"]["target"]["resource"] =
        lifecycle_resource("wrong-target");
    require_error([&] {
        static_cast<void>(session::handle_request(wrong_target));
    }, "session.authorization_target_mismatch");
    const auto first = lifecycle_boot(
        state.path(), desired_inactive, observed_inactive, "boot-operation-1", "absent", "one");
    require(first.at("changed") == true && first.at("journal").at("state") == "verified",
            "converged lifecycle boot did not commit verified evidence");
    require(first.at("journal").at("generation") == 1 &&
            first.at("journal").at("current_plan_revision") == 1 &&
            first.at("apply_authorized") == false,
            "initial lifecycle journal identity or apply boundary drifted");
    const auto first_digest = first.at("journal_digest").get<std::string>();
    const auto transaction = first.at("journal").at("transaction_id");
    const auto directory = lifecycle_stream_path(state.path());
    const auto linked_lock = state.path() / "linked-lock";
    fs::create_hard_link(directory / ".lock", linked_lock);
    require_error([&] {
        static_cast<void>(lifecycle_boot_state(
            state.path(), "lifecycle_boot_status", nullptr, nullptr, "linked-lock-status"));
    }, "lifecycle_journal.lock_unsafe");
    fs::remove(linked_lock);

    const auto replay = lifecycle_boot(
        state.path(), desired_inactive, observed_inactive, "boot-operation-1", "absent", "replay");
    require(replay.at("changed") == false && replay.at("journal_digest") == first_digest,
            "idempotent lifecycle boot replay changed state");
    auto conflicting_profile = lifecycle_boot_request(
        state.path(), desired_inactive, observed_inactive,
        "boot-operation-1", "absent", "conflicting-profile");
    const auto replacement_profile_digest = receipt("replacement-profile");
    const auto stable_inventory_digest =
        conflicting_profile.payload.at("stable_inventory_digest").get<std::string>();
    conflicting_profile.payload["profile_digest"] = replacement_profile_digest;
    conflicting_profile.payload["authorization_decision"] = lifecycle_authorization(
        "boot", replacement_profile_digest + "\n" +
            desired_inactive.at("desired_state_digest").get<std::string>() + "\nreport\n" +
            stable_inventory_digest,
        "conflicting-profile");
    require_error([&] {
        static_cast<void>(session::handle_request(conflicting_profile));
    }, "lifecycle_journal.operation_conflict");

    const auto status = lifecycle_boot_state(
        state.path(), "lifecycle_boot_status", nullptr, nullptr, "status");
    require(status.at("read_only") == true && status.at("journal_digest") == first_digest,
            "lifecycle boot status was not a read-only exact snapshot");

    auto timestamp_only = observed_inactive;
    timestamp_only["observed_at"] = "2026-08-04T16:00:01Z";
    finalize_observation(timestamp_only);
    require(timestamp_only.at("observation_digest") != observed_inactive.at("observation_digest"),
            "timestamp-only lifecycle fixture did not change its document digest");
    const auto rescanned = lifecycle_boot(
        state.path(), desired_inactive, timestamp_only, "boot-operation-rescan", first_digest, "rescan");
    require(rescanned.at("changed") == false && rescanned.at("journal_digest") == first_digest &&
            rescanned.at("journal").at("current_observation_digest") ==
                observed_inactive.at("observation_digest"),
            "timestamp-only lifecycle rescan created durable journal churn");

    const auto desired_active = desired_state(engine::Json::array({
        desired_component("durable-a", package, "active"),
    }));
    const auto second = lifecycle_boot(
        state.path(), desired_active, observed_inactive, "boot-operation-2", first_digest, "two");
    require(second.at("changed") == true && second.at("journal").at("state") == "open" &&
            second.at("journal").at("transaction_id") == transaction &&
            second.at("journal").at("current_plan_revision") == 2 &&
            second.at("journal").at("replan_count") == 1 &&
            second.at("plan").at("previous_plan_digest") == first.at("journal").at("current_plan_digest"),
            "changed desired evidence did not create a linked in-transaction plan revision");
    const auto second_digest = second.at("journal_digest").get<std::string>();

    const auto observed_active = observation(engine::Json::array({
        observed_component("durable-a", package, "active"),
    }));
    const auto healed = lifecycle_boot(
        state.path(), desired_active, observed_active, "boot-operation-3", second_digest, "three");
    require(healed.at("journal").at("state") == "verified" &&
            healed.at("journal").at("transaction_id") == transaction &&
            healed.at("journal").at("current_plan_revision") == 3 &&
            healed.at("journal").at("checkpoints").size() == 3,
            "verified evidence did not heal the durable transaction forward");
    const auto healed_digest = healed.at("journal_digest").get<std::string>();

    require_error([&] {
        static_cast<void>(lifecycle_boot(
            state.path(), desired_active, observed_active, "boot-operation-stale", second_digest, "stale"));
    }, "lifecycle_journal.expected_state_mismatch");

    write_json(directory / "head.json", engine::Json{{"broken", true}});
    require_error([&] {
        static_cast<void>(lifecycle_boot_state(
            state.path(), "lifecycle_boot_status", nullptr, nullptr, "broken-status"));
    }, "lifecycle_journal.field_set");
    const auto recovered = lifecycle_boot_state(
        state.path(), "lifecycle_boot_recover", "boot-recover-1", "discover", "recover");
    require(recovered.at("changed") == true && recovered.at("recovered") == true &&
            recovered.at("journal").at("previous_journal_digest") == healed_digest &&
            recovered.at("journal").at("recovery").at("state") == "recovered",
            "lifecycle discovery recovery did not preserve and advance verified evidence");

    const auto head = read_json(directory / "head.json");
    const auto active_slot = head.at("active_slot").get<int>();
    const auto active = read_json(directory / ("journal." + std::to_string(active_slot) + ".json"));
    auto divergent = active;
    divergent["journal_id"] = "lifecycle-journal:divergent";
    finalize_test_digest(divergent, "journal_digest");
    write_json(directory / ("journal." + std::to_string(1 - active_slot) + ".json"), divergent);
    require_error([&] {
        static_cast<void>(lifecycle_boot_state(
            state.path(), "lifecycle_boot_status", nullptr, nullptr, "divergent-status"));
    }, "lifecycle_journal.recovery_ambiguous");

    require_error([&] {
        static_cast<void>(lifecycle_boot_state(
            state.path(), "lifecycle_boot_recover", "boot-recover-read-only", "discover",
            "recover-read-only", false));
    }, "lifecycle_journal.compatibility_required");

    TemporaryDirectory critical_state;
    const auto critical_first = lifecycle_boot(
        critical_state.path(), desired_inactive, observed_inactive,
        "boot-operation-critical", "absent", "critical");
    const auto critical_directory = lifecycle_stream_path(critical_state.path());
    const auto critical_head = read_json(critical_directory / "head.json");
    const auto critical_slot = critical_head.at("active_slot").get<int>();
    auto critical_journal = critical_first.at("journal");
    const engine::Json extension_payload{{"future_state", "requires-compatible-reader"}};
    critical_journal["extensions"].push_back(engine::Json{
        {"extension_id", "future-critical-v1"}, {"extension_version", "1.0.0"},
        {"critical", true}, {"payload", extension_payload},
        {"payload_digest", engine::tagged_sha256(extension_payload.dump())},
    });
    finalize_test_digest(critical_journal, "journal_digest");
    write_json(
        critical_directory / ("journal." + std::to_string(critical_slot) + ".json"),
        critical_journal);
    require_error([&] {
        static_cast<void>(lifecycle_boot_state(
            critical_state.path(), "lifecycle_boot_recover", "boot-recover-critical", "discover",
            "recover-critical"));
    }, "lifecycle_journal.compatibility_required");
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
    require(safe_undock.at("expected_artifact_digests") == engine::Json::array({old_receipt}),
            "inverse undock did not bind the currently observed receipt");
    require(safe_dock.at("expected_artifact_digests") == engine::Json::array({new_receipt}),
            "forward dock did not bind the desired replacement receipt");
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

    auto impossible_time = observed;
    impossible_time["observed_at"] = "2026-02-31T16:00:00Z";
    finalize_observation(impossible_time);
    require_error([&] { static_cast<void>(plan(desired, impossible_time)); }, "lifecycle.invalid_field");

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

void test_observation_timestamp_does_not_restart_transaction() {
    const auto component_receipt = receipt("stable-inventory");
    const auto desired = desired_state(engine::Json::array({
        desired_component("stable-inventory", component_receipt),
    }));
    const auto first_observation = observation(engine::Json::array({
        observed_component("stable-inventory", component_receipt),
    }));
    auto later_observation = first_observation;
    later_observation["observed_at"] = "2026-08-04T16:01:00Z";
    finalize_observation(later_observation);
    require(first_observation.at("observation_digest") != later_observation.at("observation_digest"),
            "fresh observation timestamp did not change document evidence");

    const auto first = plan(desired, first_observation);
    const auto later = plan(desired, later_observation);
    require(first.at("transaction_id") == later.at("transaction_id"),
            "timestamp-only observation refresh restarted the lifecycle transaction");
    require(first.at("observation_key") == later.at("observation_key"),
            "timestamp-only observation refresh changed the stable observation key");
    require(action(first, "stable-inventory", "activate").at("action_id") ==
            action(later, "stable-inventory", "activate").at("action_id"),
            "timestamp-only refresh changed semantic action identity");
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
        test_apply_journal_prepare_resume_finalize_close_and_recovery();
        test_apply_prepare_accepts_only_exact_staged_install_evidence();
        test_durable_boot_journal_replanning_and_recovery();
        test_dependency_ready_set_and_replanning();
        test_cycle_isolation();
        test_dependency_criticality_and_component_capabilities();
        test_two_way_selection_and_determinism();
        test_receptor_switch_and_safe_package_sequence();
        test_compatibility_integrity_and_digest_fail_closed();
        test_observation_timestamp_does_not_restart_transaction();
        test_bounded_scale_plan();
        std::cout << "knowledge lifecycle planner tests passed\n";
        return 0;
    } catch (const std::exception& error) {
        std::cerr << "test failure: " << error.what() << '\n';
        return 1;
    }
}
