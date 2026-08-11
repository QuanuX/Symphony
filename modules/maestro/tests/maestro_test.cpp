#include "maestro.hpp"

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
#include <unistd.h>
#include <vector>

namespace fs = std::filesystem;
namespace engine = symphony::knowledge::engine;
namespace maestro = symphony::maestro;

namespace {

constexpr const char* tops_id = "018f0c3a-7b2d-7e11-8c12-0242ac120002";
constexpr const char* receptor_id = "maestro-primary";

class TemporaryDirectory final {
public:
    TemporaryDirectory() {
        const auto value = (fs::canonical(fs::temp_directory_path()) /
                            "symphony-maestro-test.XXXXXX").string();
        std::vector<char> pattern(value.begin(), value.end());
        pattern.push_back('\0');
        if (::mkdtemp(pattern.data()) == nullptr) throw std::runtime_error("mkdtemp failed");
        path_ = pattern.data();
    }
    ~TemporaryDirectory() { std::error_code error; fs::remove_all(path_, error); }
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
        require(error.code() == code, "unexpected error: " + error.code());
        return;
    }
    throw std::runtime_error("expected error: " + code);
}

std::string resource(const std::string& operation, const std::string& component_id,
                     const std::string& receipt_digest, const std::string& expected) {
    return "symphony.maestro.docking:" + engine::sha256_hex(
        std::string(tops_id) + "\n" + receptor_id + "\n" + operation + "\n" +
        component_id + "\n" + receipt_digest + "\n" + expected);
}

engine::Json authorization(const std::string& operation, const std::string& target_resource,
                           const std::string& suffix) {
    const auto request_id = "request-" + suffix;
    const auto correlation_id = "correlation-" + suffix;
    const auto target = engine::Json{
        {"operation", "symphony.maestro.docking." + operation},
        {"resource", target_resource}, {"audience", "qxctl"},
        {"scope", "tops:" + std::string(tops_id)},
    };
    const auto subject = engine::Json{
        {"id", "owner.primary"}, {"kind", "symphony.identity.owner"},
        {"authority", "unix_peer_credentials"},
    };
    engine::Json capability{
        {"protocol", "symphony.ssiag.capability.v1"}, {"capability_id", "pending"},
        {"subject", subject}, {"tops_id", tops_id}, {"target", target},
        {"authority_basis", "host_owner"}, {"grant_id", "maestro-" + operation},
        {"request_id", request_id}, {"correlation_id", correlation_id},
        {"issued_at", "2020-01-01T00:00:00Z"}, {"expires_at", "2099-01-01T00:00:00Z"},
        {"policy_digest", "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
        {"config_digest", "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
        {"binding_digest", "pending"}, {"transferable", false}, {"canonical_apply", false},
    };
    const std::array<std::string, 19> values = {
        capability.at("protocol"), subject.at("id"), subject.at("kind"), subject.at("authority"),
        tops_id, target.at("operation"), target.at("resource"), target.at("audience"), target.at("scope"),
        capability.at("authority_basis"), capability.at("grant_id"), request_id, correlation_id,
        capability.at("issued_at"), capability.at("expires_at"), capability.at("policy_digest"),
        capability.at("config_digest"), "transferable=false", "canonical_apply=false",
    };
    std::string binding_input;
    for (std::size_t index = 0; index < values.size(); ++index) {
        if (index != 0U) binding_input.push_back('\n');
        binding_input += values[index];
    }
    capability["binding_digest"] = engine::tagged_sha256(binding_input);
    capability["capability_id"] =
        "ssiag-capability:" + capability.at("binding_digest").get<std::string>().substr(7U);
    return engine::Json{
        {"schema", "symphony.ssiag.authorization-decision.v1"},
        {"decision_id", "ssiag-decision:" + engine::sha256_hex(operation + suffix)},
        {"request_id", request_id}, {"correlation_id", correlation_id}, {"tops_id", tops_id},
        {"subject", subject}, {"target", target}, {"effect", "allow"},
        {"reason_code", "symphony.ssiag.policy.exact-grant"}, {"authority_basis", "host_owner"},
        {"capability", capability}, {"policy_digest", capability.at("policy_digest")},
        {"config_digest", capability.at("config_digest")}, {"decided_at", capability.at("issued_at")},
        {"expires_at", capability.at("expires_at")}, {"caller_class_used", false},
        {"canonical_apply", false},
    };
}

engine::Json client(bool full = true) {
    auto capabilities = engine::Json::array({
        "atomic-head-v1", "dual-slot-presence-v1", "exact-receipt-binding-v1",
        "expected-state-cas-v1", "idempotent-operation-v1", "recovery-forward-v1",
        "ssiag-capability-binding-v1",
    });
    if (!full) capabilities.erase(capabilities.end() - 1);
    return engine::Json{
        {"client_id", "qxctl"}, {"client_version", "qxctl-dev"},
        {"process_protocols", engine::Json::array({engine::process_protocol_v1})},
        {"presence_read_versions", engine::Json::array({1})},
        {"presence_write_versions", engine::Json::array({1})}, {"capabilities", capabilities},
    };
}

engine::Json component() {
    engine::Json value{
        {"component_id", "skvi-engine"}, {"component_kind", "vector_engine"},
        {"module_id", "skvi-engine"}, {"vector_id", "skvi"}, {"engine_id", "symphony-skvi"},
        {"receipt_digest", "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
        {"executable_digest", "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
        {"receptor_kind", "symphony.maestro.knowledge-engine.v1"},
    };
    value["evidence_digest"] = engine::tagged_sha256(value.dump());
    return value;
}

engine::Request command(const std::string& operation, const fs::path& root,
                        engine::Json operation_id, engine::Json expected,
                        engine::Json component_value, engine::Json decision,
                        bool full_client = true) {
    return engine::Request{
        "request-" + operation, "correlation-" + operation, operation, maestro::engine_id,
        engine::unix_time_ms() + 60000,
        engine::Json{
            {"protocol", "symphony.maestro.knowledge-engine-docking.v1"}, {"format_version", 1},
            {"operation", operation}, {"state_root", operation == "inspect" ? engine::Json(nullptr) : engine::Json(root.string())},
            {"tops_id", tops_id}, {"receptor_id", receptor_id}, {"operation_id", std::move(operation_id)},
            {"expected_registry_digest", std::move(expected)}, {"component", std::move(component_value)},
            {"authorization_decision", std::move(decision)}, {"client", client(full_client)},
        },
    };
}

engine::Json inspect() {
    return maestro::handle_request(command("inspect", {}, nullptr, nullptr, nullptr, nullptr));
}

engine::Json maestro_status(const fs::path& root, const std::string& filter = "skvi-engine") {
    auto selected = filter == "all" ? engine::Json(nullptr) : engine::Json{{"component_id", filter}};
    return maestro::handle_request(command(
        "status", root, nullptr, nullptr, selected,
        authorization("status", resource("status", filter, "none", "status"), "status")));
}

engine::Json mutate(const std::string& operation, const fs::path& root, const std::string& expected,
                    const std::string& suffix, bool full_client = true) {
    const auto evidence = component();
    return maestro::handle_request(command(
        operation, root, "operation-" + suffix, expected, evidence,
        authorization(operation, resource(operation, "skvi-engine",
            evidence.at("receipt_digest"), expected), suffix), full_client));
}

void test_descriptor_and_status_are_bounded() {
    TemporaryDirectory root;
    const auto description = inspect();
    require(description.at("descriptor").at("receptor_kind") == "symphony.maestro.knowledge-engine.v1",
            "descriptor receptor kind mismatch");
    require(description.at("execution_enabled") == false, "execution must remain disabled");
    require(!fs::exists(root.path() / "symphony"), "inspect must not touch state");
    const auto absent = maestro_status(root.path());
    require(absent.at("registry_present") == false, "status should report absent state");
    require(!fs::exists(root.path() / "symphony"), "absent status must not create state");
}

void test_dock_idempotency_cas_and_undock() {
    TemporaryDirectory root;
    const auto docked = mutate("dock", root.path(), "absent", "dock-1");
    require(docked.at("changed") == true && docked.at("presence_present") == true,
            "dock did not persist presence");
    const auto digest = docked.at("registry_digest").get<std::string>();
    const auto retry = mutate("dock", root.path(), "absent", "dock-retry");
    require(retry.at("outcome") == "already_applied" && retry.at("changed") == false,
            "semantic retry should be idempotent");
    require_error([&] { static_cast<void>(mutate("undock", root.path(), "absent", "stale")); },
                  "maestro.stale_expected_state");
    const auto undocked = mutate("undock", root.path(), digest, "undock-1");
    require(undocked.at("changed") == true && undocked.at("presence_present") == false,
            "undock did not persist absence");
    const auto observed = maestro_status(root.path());
    require(observed.at("presence").at("disposition") == "undocked", "status lost tombstone");
}

void test_authorization_compatibility_and_component_fail_closed() {
    TemporaryDirectory root;
    auto denied = command("dock", root.path(), "operation-denied", "absent", component(),
        authorization("dock", resource("dock", "skvi-engine",
            component().at("receipt_digest"), "absent"), "denied"));
    denied.payload["authorization_decision"]["caller_class_used"] = true;
    require_error([&] { static_cast<void>(maestro::handle_request(denied)); },
                  "maestro.authorization_denied");
    require_error([&] { static_cast<void>(mutate("dock", root.path(), "absent", "compat", false)); },
                  "maestro.compatibility_required");
    auto unsupported = component();
    unsupported["component_kind"] = "runtime";
    unsupported["evidence_digest"] = engine::tagged_sha256(unsupported.dump());
    auto request = command("dock", root.path(), "operation-unsupported", "absent", unsupported,
        authorization("dock", resource("dock", "skvi-engine",
            unsupported.at("receipt_digest"), "absent"), "unsupported"));
    require_error([&] { static_cast<void>(maestro::handle_request(request)); },
                  "maestro.component_invalid");

    const auto docked = mutate("dock", root.path(), "absent", "identity-dock");
    auto drifted = component();
    drifted["receipt_digest"] =
        "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc";
    drifted.erase("evidence_digest");
    drifted["evidence_digest"] = engine::tagged_sha256(drifted.dump());
    const auto expected = docked.at("registry_digest").get<std::string>();
    auto mismatched = command("undock", root.path(), "operation-mismatch", expected, drifted,
        authorization("undock", resource("undock", "skvi-engine",
            drifted.at("receipt_digest"), expected), "mismatch"));
    require_error([&] { static_cast<void>(maestro::handle_request(mismatched)); },
                  "maestro.component_mismatch");
}

void test_forward_recovery_and_scope_isolation() {
    TemporaryDirectory root;
    const auto first = mutate("dock", root.path(), "absent", "recovery-dock");
    const auto second = mutate("undock", root.path(), first.at("registry_digest"), "recovery-undock");
    const auto presence_dir = root.path() / "symphony" / "maestro" / "docking" / "v1" / "tops" /
        engine::sha256_hex(std::string("tops:") + tops_id) / "receptors" /
        engine::sha256_hex(std::string("receptor:") + receptor_id);
    fs::remove(presence_dir / "head.json");
    require_error([&] { static_cast<void>(maestro_status(root.path())); }, "maestro.head_missing");
    const auto recovery_resource = resource("recover", "all", "none", "discover");
    const auto recovered = maestro::handle_request(command(
        "recover", root.path(), "operation-recover", "discover", nullptr,
        authorization("recover", recovery_resource, "recover")));
    require(recovered.at("recovered") == true && recovered.at("changed") == true,
            "recover did not commit forward state");
    require(recovered.at("registry").at("generation").get<std::uint64_t>() ==
            second.at("registry").at("generation").get<std::uint64_t>() + 1U,
            "recover must advance generation");
    require(maestro_status(root.path()).at("registry_digest") == recovered.at("registry_digest"),
            "recovered head is not readable");
}

void test_inactive_slot_damage_requires_and_accepts_recovery() {
    TemporaryDirectory root;
    const auto first = mutate("dock", root.path(), "absent", "slot-dock");
    const auto second = mutate("undock", root.path(), first.at("registry_digest"), "slot-undock");
    const auto presence_dir = root.path() / "symphony" / "maestro" / "docking" / "v1" / "tops" /
        engine::sha256_hex(std::string("tops:") + tops_id) / "receptors" /
        engine::sha256_hex(std::string("receptor:") + receptor_id);
    fs::remove(presence_dir / "registry.0.json");
    require_error([&] { static_cast<void>(maestro_status(root.path())); }, "maestro.recovery_required");
    const auto recovery_resource = resource("recover", "all", "none", "discover");
    const auto recovered = maestro::handle_request(command(
        "recover", root.path(), "operation-slot-recover", "discover", nullptr,
        authorization("recover", recovery_resource, "slot-recover")));
    require(recovered.at("recovered") == true &&
            recovered.at("registry").at("generation").get<std::uint64_t>() ==
                second.at("registry").at("generation").get<std::uint64_t>() + 1U,
            "inactive slot recovery did not commit a forward healing generation");
    require(maestro_status(root.path()).at("registry_digest") == recovered.at("registry_digest"),
            "healed inactive slot state is not readable");
}

void test_future_head_is_never_recovered_as_corruption() {
    TemporaryDirectory root;
    static_cast<void>(mutate("dock", root.path(), "absent", "future-head-dock"));
    const auto presence_dir = root.path() / "symphony" / "maestro" / "docking" / "v1" / "tops" /
        engine::sha256_hex(std::string("tops:") + tops_id) / "receptors" /
        engine::sha256_hex(std::string("receptor:") + receptor_id);
    {
        std::ofstream head(presence_dir / "head.json", std::ios::trunc);
        require(head.good(), "could not create future head fixture");
        head << "{\"protocol\":\"symphony.maestro.docking-presence-head.v2\",\"format_version\":2}\n";
    }
    require_error([&] { static_cast<void>(maestro_status(root.path())); },
                  "maestro.compatibility_required");
    const auto recovery_resource = resource("recover", "all", "none", "discover");
    auto recovery = command("recover", root.path(), "operation-future-head", "discover", nullptr,
        authorization("recover", recovery_resource, "future-head"));
    require_error([&] { static_cast<void>(maestro::handle_request(recovery)); },
                  "maestro.compatibility_required");
}

void test_symlink_state_root_is_rejected() {
    TemporaryDirectory parent;
    TemporaryDirectory target;
    const auto link = parent.path() / "state-link";
    fs::create_directory_symlink(target.path(), link);
    require_error([&] { static_cast<void>(maestro_status(link)); }, "maestro.state_directory_open_failed");
}

} // namespace

int main() {
    try {
        test_descriptor_and_status_are_bounded();
        test_dock_idempotency_cas_and_undock();
        test_authorization_compatibility_and_component_fail_closed();
        test_forward_recovery_and_scope_isolation();
        test_inactive_slot_damage_requires_and_accepts_recovery();
        test_future_head_is_never_recovered_as_corruption();
        test_symlink_state_root_is_rejected();
        std::cout << "maestro tests passed\n";
        return 0;
    } catch (const std::exception& error) {
        std::cerr << error.what() << '\n';
        return 1;
    }
}
