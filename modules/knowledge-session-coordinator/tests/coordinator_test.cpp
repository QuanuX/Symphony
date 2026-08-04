#include "coordinator.hpp"

#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/protocol.hpp"

#include <algorithm>
#include <array>
#include <fcntl.h>
#include <filesystem>
#include <fstream>
#include <iostream>
#include <stdexcept>
#include <string>
#include <sys/file.h>
#include <sys/stat.h>
#include <unistd.h>

namespace fs = std::filesystem;
namespace session = symphony::knowledge::session;
namespace engine = symphony::knowledge::engine;

namespace {

class TemporaryDirectory final {
public:
    TemporaryDirectory() {
        std::string pattern = (fs::canonical(fs::temp_directory_path()) / "symphony-session-test-XXXXXX").string();
        pattern.push_back('\0');
        char* result = ::mkdtemp(pattern.data());
        if (result == nullptr) {
            throw std::runtime_error("mkdtemp failed");
        }
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

class CurrentDirectory final {
public:
    explicit CurrentDirectory(const fs::path& path) : previous_(fs::current_path()) {
        fs::current_path(path);
    }
    ~CurrentDirectory() {
        std::error_code ignored;
        fs::current_path(previous_, ignored);
    }
    CurrentDirectory(const CurrentDirectory&) = delete;
    CurrentDirectory& operator=(const CurrentDirectory&) = delete;

private:
    fs::path previous_;
};

void require(bool condition, const std::string& message) {
    if (!condition) {
        throw std::runtime_error(message);
    }
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

engine::Request request(std::string operation, engine::Json payload) {
    return engine::Request{
        "request-1",
        "correlation-1",
        std::move(operation),
        session::engine_id,
        engine::unix_time_ms() + 60000,
        std::move(payload),
    };
}

engine::Json reconciliation_payload(
    const fs::path& state_root,
    const std::string& operation,
    const engine::Json& operation_id,
    const engine::Json& expected,
    const engine::Json& paths,
    std::string engine_version = "0.1.0-dev",
    bool full_capabilities = true) {
    auto capabilities = engine::Json::array({
        "atomic-head-v1",
        "content-snapshot-v1",
        "discovery-recovery-v1",
        "dual-slot-journal-v1",
        "expected-state-cas-v1",
        "idempotent-operation-v1",
        "nonblocking-lock-v1",
        "opaque-extension-preservation-v1",
        "recovery-forward-v1",
    });
    if (!full_capabilities) {
        capabilities = engine::Json::array({"content-snapshot-v1"});
    }
    return engine::Json{
        {"protocol", "symphony.knowledge.reconciliation-command.v1"},
        {"operation", operation},
        {"state_root", fs::canonical(state_root).string()},
        {"operation_id", operation_id},
        {"expected_journal_digest", expected},
        {"paths", paths},
        {"binding_registry_digest",
         "sha256:2222222222222222222222222222222222222222222222222222222222222222"},
        {"engine_inventory", engine::Json::array({
            engine::Json{
                {"role", "coordinator"},
                {"module_id", "knowledge-session-coordinator"},
                {"engine_id", "symphony-knowledge-session"},
                {"version", std::move(engine_version)},
                {"receipt_digest",
                 "sha256:0000000000000000000000000000000000000000000000000000000000000000"},
                {"executable_digest",
                 "sha256:1111111111111111111111111111111111111111111111111111111111111111"},
            },
        })},
        {"client", engine::Json{
            {"client_id", "qxctl"},
            {"client_version", "qxctl-dev"},
            {"process_protocols", engine::Json::array({engine::process_protocol_v1})},
            {"journal_read_versions", engine::Json::array({1})},
            {"journal_write_versions", engine::Json::array({1})},
            {"capabilities", capabilities},
        }},
    };
}

engine::Json reconcile(
    const fs::path& state_root,
    const std::string& operation,
    const engine::Json& operation_id,
    const engine::Json& expected,
    const engine::Json& paths = engine::Json::array(),
    std::string engine_version = "0.1.0-dev",
    bool full_capabilities = true) {
    return session::handle_request(request(
        operation,
        reconciliation_payload(
            state_root,
            operation,
            operation_id,
            expected,
            paths,
            std::move(engine_version),
            full_capabilities)));
}

engine::Json authorization_decision(const std::string& operation, const std::string& suffix = "1") {
    const std::string tops_id = "018f0c3a-7b2d-7e11-8c12-0242ac120002";
    const auto request_id = "request-" + suffix;
    const auto correlation_id = "correlation-" + suffix;
    const auto target = engine::Json{
        {"operation", "symphony.knowledge.session." + operation},
        {"resource", "symphony.knowledge.repository:" + engine::sha256_hex(
            "repository-root:" + fs::canonical(fs::current_path()).string())}, {"audience", "qxctl"},
        {"scope", "tops:" + tops_id},
    };
    const auto subject = engine::Json{
        {"id", "owner.primary"}, {"kind", "symphony.identity.owner"},
        {"authority", "unix_peer_credentials"},
    };
    engine::Json capability{
        {"protocol", "symphony.ssiag.capability.v1"}, {"capability_id", "pending"},
        {"subject", subject}, {"tops_id", tops_id}, {"target", target},
        {"authority_basis", "host_owner"}, {"grant_id", "knowledge-session-" + operation},
        {"request_id", request_id}, {"correlation_id", correlation_id},
        {"issued_at", "2020-01-01T00:00:00Z"}, {"expires_at", "2099-01-01T00:00:00Z"},
        {"policy_digest", "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
        {"config_digest", "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
        {"binding_digest", "pending"}, {"transferable", false}, {"canonical_apply", false},
    };
    const auto values = std::array<std::string, 19>{
        capability.at("protocol"), subject.at("id"), subject.at("kind"), subject.at("authority"),
        tops_id, target.at("operation"), target.at("resource"), target.at("audience"), target.at("scope"),
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
    capability["capability_id"] = "ssiag-capability:" + capability.at("binding_digest").get<std::string>().substr(7);
    return engine::Json{
        {"schema", "symphony.ssiag.authorization-decision.v1"},
        {"decision_id", "ssiag-decision:" + engine::sha256_hex(operation + suffix)},
        {"request_id", request_id}, {"correlation_id", correlation_id}, {"tops_id", tops_id},
        {"subject", subject}, {"target", target}, {"effect", "allow"},
        {"reason_code", "symphony.ssiag.policy.exact-grant"}, {"authority_basis", "host_owner"},
        {"capability", capability}, {"policy_digest", capability.at("policy_digest")},
        {"config_digest", capability.at("config_digest")}, {"decided_at", "2020-01-01T00:00:00Z"},
        {"expires_at", capability.at("expires_at")}, {"caller_class_used", false}, {"canonical_apply", false},
    };
}

engine::Json authenticated_session(
    const fs::path& state_root,
    const std::string& operation,
    const engine::Json& operation_id,
    const engine::Json& expected,
    const engine::Json& contexts = engine::Json::array(),
    const std::string& suffix = "1",
    bool full_capabilities = true) {
    const auto qualified = "session_" + operation;
    auto capabilities = engine::Json::array({
        "atomic-head-v1", "authority-epoch-v1", "discovery-recovery-v1",
        "dual-slot-journal-v1", "expected-state-cas-v1", "idempotent-operation-v1",
        "nonblocking-lock-v1", "opaque-extension-preservation-v1",
        "recovery-forward-v1", "ssiag-capability-binding-v1",
    });
    if (!full_capabilities) capabilities.erase(capabilities.begin());
    return session::handle_request(request(qualified, engine::Json{
        {"protocol", "symphony.knowledge.session-command.v1"}, {"operation", qualified},
        {"state_root", fs::canonical(state_root).string()}, {"operation_id", operation_id},
        {"expected_journal_digest", expected}, {"repository_root", fs::canonical(fs::current_path()).string()},
        {"context_refs", contexts}, {"authorization_decision", authorization_decision(operation, suffix)},
        {"client", engine::Json{
            {"client_id", "qxctl"}, {"client_version", "qxctl-dev"},
            {"process_protocols", engine::Json::array({engine::process_protocol_v1})},
            {"journal_read_versions", engine::Json::array({1})}, {"journal_write_versions", engine::Json::array({1})},
            {"capabilities", capabilities},
        }},
    }));
}

engine::Json read_json(const fs::path& path) {
    std::ifstream input(path, std::ios::binary);
    if (!input) {
        throw std::runtime_error("could not read test JSON");
    }
    return engine::Json::parse(input);
}

void write_json(const fs::path& path, const engine::Json& value) {
    std::ofstream output(path, std::ios::binary | std::ios::trunc);
    if (!output) {
        throw std::runtime_error("could not write test JSON");
    }
    output << value.dump() << '\n';
    output.close();
    if (::chmod(path.c_str(), 0600) != 0) {
        throw std::runtime_error("could not protect test JSON");
    }
}

void finalize_test_digest(engine::Json& value, const char* field) {
    value.erase(field);
    value[field] = engine::tagged_sha256(value.dump());
}

fs::path context_path(const fs::path& state, const fs::path& repository) {
    const auto worktree =
        engine::sha256_hex("worktree-root:" + fs::canonical(repository).string());
    return state / "symphony" / "knowledge-session-coordinator" /
           "reconciliation" / "v1" / "contexts" / worktree;
}

void test_descriptor_and_inspect() {
    const auto descriptor = session::descriptor();
    require(descriptor.at("engine_id") == session::engine_id, "descriptor engine mismatch");
    require(descriptor.at("canonical_apply_enabled") == false, "apply must remain disabled");
    require(descriptor.at("session_mutation_enabled") == true, "session mutation must be enabled");
    require(descriptor.at("network_listener") == false, "network listener must remain disabled");

    const auto result = session::handle_request(request("inspect", engine::Json::object()));
    require(result.at("readiness") == "authenticated_session_foundation", "inspect readiness mismatch");
    require(result.at("reconciliation").at("two_way_procedural_compatibility") == true,
            "reconciliation compatibility declaration missing");
    require(result.at("authenticated_session").at("two_way_procedural_compatibility") == true,
            "authenticated session compatibility declaration missing");
    require(result.at("maestro_docking_enabled") == false, "docking must remain disabled");

    require_error([&] {
        static_cast<void>(session::handle_request(request("inspect", engine::Json{{"extra", true}})));
    }, "payload.field_set");
}

void test_check() {
    TemporaryDirectory temporary;
    {
        std::ofstream output(temporary.path() / "INTENT.md", std::ios::binary);
        output << "intent\n";
    }
    CurrentDirectory current(temporary.path());
    const auto first = session::handle_request(request("check", engine::Json{
        {"paths", engine::Json::array({"INTENT.md"})},
        {"expected_snapshot_digest", nullptr},
    }));
    const auto digest = first.at("snapshot").at("digest").get<std::string>();
    require(first.at("expected_snapshot_matches").is_null(), "unexpected expected-state result");
    require(first.at("read_only") == true, "check must be read-only");

    const auto second = session::handle_request(request("check", engine::Json{
        {"paths", engine::Json::array({"INTENT.md"})},
        {"expected_snapshot_digest", digest},
    }));
    require(second.at("expected_snapshot_matches") == true, "expected digest did not match");
    const auto mismatch = session::handle_request(request("check", engine::Json{
        {"paths", engine::Json::array({"INTENT.md"})},
        {"expected_snapshot_digest", "sha256:0000000000000000000000000000000000000000000000000000000000000000"},
    }));
    require(mismatch.at("expected_snapshot_matches") == false, "stale expected digest was accepted");

    require_error([&] {
        static_cast<void>(session::handle_request(request("check", engine::Json{
            {"paths", engine::Json::array({"../INTENT.md"})},
            {"expected_snapshot_digest", nullptr},
        })));
    }, "path.unsafe");
    require_error([&] {
        static_cast<void>(session::handle_request(request("check", engine::Json{
            {"paths", engine::Json::array({"INTENT.md"})},
            {"expected_snapshot_digest", "sha256:gggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggg"},
        })));
    }, "payload.invalid_expected_digest");
}

void test_reserved_operations() {
    for (const std::string operation : {"apply"}) {
        require_error([&] {
            static_cast<void>(session::handle_request(request(operation, engine::Json::object())));
        }, "operation.unsupported");
    }
}

void test_authenticated_session_lifecycle_and_recovery() {
    TemporaryDirectory repository;
    TemporaryDirectory state;
    CurrentDirectory current(repository.path());

    const auto begin = authenticated_session(
        state.path(), "begin", "session-begin-1", "absent",
        engine::Json::array({"reconcile:context-one"}), "begin-1");
    require(begin.at("changed") == true && begin.at("effective_state") == "open",
            "authenticated session begin did not commit");
    require(begin.at("journal").at("contexts").size() == 1,
            "session begin did not attach the reconciliation context");
    const auto begin_digest = begin.at("journal_digest").get<std::string>();

    const auto replay = authenticated_session(
        state.path(), "begin", "session-begin-1", "absent",
        engine::Json::array({"reconcile:context-one"}), "begin-2");
    require(replay.at("changed") == false && replay.at("journal_digest") == begin_digest,
            "authenticated session begin replay changed state");
    require_error([&] {
        static_cast<void>(authenticated_session(
            state.path(), "begin", "session-begin-1", "absent",
            engine::Json::array({"reconcile:different-context"}), "begin-drift"));
    }, "session.operation_reused");

    const auto checkpoint = authenticated_session(
        state.path(), "checkpoint", "session-checkpoint-1", begin_digest,
        engine::Json::array({"reconcile:context-two"}), "checkpoint-1");
    require(checkpoint.at("journal").at("contexts").size() == 2,
            "session checkpoint did not attach a second context");
    const auto checkpoint_digest = checkpoint.at("journal_digest").get<std::string>();
    require_error([&] {
        static_cast<void>(authenticated_session(
            state.path(), "checkpoint", "session-checkpoint-stale", begin_digest,
            engine::Json::array(), "checkpoint-stale"));
    }, "session.expected_state_mismatch");

    const auto status = authenticated_session(
        state.path(), "status", nullptr, nullptr, engine::Json::array(), "status-1");
    require(status.at("read_only") == true && status.at("effective_state") == "open",
            "authenticated session status was not read-only and open");

    const auto close = authenticated_session(
        state.path(), "close", "session-close-1", checkpoint_digest,
        engine::Json::array(), "close-1");
    require(close.at("journal").at("state") == "closed" &&
                close.at("journal").at("close_reason") == "logout",
            "authenticated session close did not record logout");
    const auto closed_digest = close.at("journal_digest").get<std::string>();

    const auto second = authenticated_session(
        state.path(), "begin", "session-begin-2", closed_digest,
        engine::Json::array(), "begin-3");
    require(second.at("journal").at("generation") > close.at("journal").at("generation") &&
                second.at("journal").at("previous_journal_digest") == closed_digest,
            "new authority epoch did not link the closed predecessor");

    const auto second_digest = second.at("journal_digest").get<std::string>();
    const auto key = engine::sha256_hex(
        "session-key:018f0c3a-7b2d-7e11-8c12-0242ac120002|owner.primary|" +
        fs::canonical(repository.path()).string());
    const auto directory = state.path() / "symphony" / "knowledge-session-coordinator" /
        "sessions" / "v1" / "epochs" / key;
    write_json(directory / "head.json", engine::Json{{"broken", true}});
    require_error([&] {
        static_cast<void>(authenticated_session(
            state.path(), "status", nullptr, nullptr, engine::Json::array(), "status-broken"));
    }, "session.field_set");
    const auto recovered = authenticated_session(
        state.path(), "recover", "session-recover-1", "discover",
        engine::Json::array(), "recover-1");
    require(recovered.at("changed") == true && recovered.at("recovered") == true &&
                recovered.at("journal").at("previous_journal_digest") == second_digest,
            "authenticated session discovery recovery did not roll forward");

    require_error([&] {
        static_cast<void>(authenticated_session(
            state.path(), "recover", "session-recover-read-only", recovered.at("journal_digest"),
            engine::Json::array(), "recover-read-only", false));
    }, "session.compatibility_required");

    auto denied = authorization_decision("checkpoint", "denied");
    denied["effect"] = "deny";
    denied["capability"] = nullptr;
    denied["authority_basis"] = nullptr;
    denied["expires_at"] = nullptr;
    auto payload = engine::Json{
        {"protocol", "symphony.knowledge.session-command.v1"}, {"operation", "session_checkpoint"},
        {"state_root", fs::canonical(state.path()).string()}, {"operation_id", "denied-op"},
        {"expected_journal_digest", recovered.at("journal_digest")},
        {"repository_root", fs::canonical(repository.path()).string()}, {"context_refs", engine::Json::array()},
        {"authorization_decision", denied},
        {"client", engine::Json{
            {"client_id", "qxctl"}, {"client_version", "qxctl-dev"},
            {"process_protocols", engine::Json::array({engine::process_protocol_v1})},
            {"journal_read_versions", engine::Json::array({1})}, {"journal_write_versions", engine::Json::array({1})},
            {"capabilities", engine::Json::array({
                "atomic-head-v1", "authority-epoch-v1", "dual-slot-journal-v1",
                "expected-state-cas-v1", "idempotent-operation-v1",
                "opaque-extension-preservation-v1", "recovery-forward-v1",
                "ssiag-capability-binding-v1",
            })},
        }},
    };
    require_error([&] {
        static_cast<void>(session::handle_request(request("session_checkpoint", payload)));
    }, "session.authorization_denied");

    payload["authorization_decision"] = authorization_decision("checkpoint", "malformed-client");
    payload["client"]["journal_read_versions"] = engine::Json::array({"1"});
    require_error([&] {
        static_cast<void>(session::handle_request(request("session_checkpoint", payload)));
    }, "session.invalid_field");
}

void test_authenticated_session_slot_safety() {
    TemporaryDirectory repository;
    TemporaryDirectory state;
    CurrentDirectory current(repository.path());

    const auto begin = authenticated_session(
        state.path(), "begin", "slot-begin", "absent", engine::Json::array(), "slot-begin");
    const auto key = engine::sha256_hex(
        "session-key:018f0c3a-7b2d-7e11-8c12-0242ac120002|owner.primary|" +
        fs::canonical(repository.path()).string());
    const auto directory = state.path() / "symphony" / "knowledge-session-coordinator" /
        "sessions" / "v1" / "epochs" / key;
    auto head = read_json(directory / "head.json");
    const auto active_slot = head.at("active_slot").get<int>();
    const auto active_path = directory / ("journal." + std::to_string(active_slot) + ".json");
    const auto inactive_path = directory / ("journal." + std::to_string(1 - active_slot) + ".json");
    auto active = read_json(active_path);
    auto linked = active;
    linked["generation"] = active.at("generation").get<std::uint64_t>() + 1U;
    linked["previous_journal_digest"] = active.at("journal_digest");
    finalize_test_digest(linked, "journal_digest");
    write_json(inactive_path, linked);

    require_error([&] {
        static_cast<void>(authenticated_session(
            state.path(), "status", nullptr, nullptr, engine::Json::array(), "slot-status"));
    }, "session.recovery_required");
    const auto recovered = authenticated_session(
        state.path(), "recover", "slot-recover", "discover", engine::Json::array(), "slot-recover");
    require(recovered.at("recovered") == true && recovered.at("changed") == true,
            "linked durable successor was not recovered forward");
    require(recovered.at("journal").at("recovery").at("disposition") ==
                "adopted_linked_successor",
            "linked session successor recovery disposition mismatch");

    head = read_json(directory / "head.json");
    const auto recovered_slot = head.at("active_slot").get<int>();
    const auto recovered_path = directory / ("journal." + std::to_string(recovered_slot) + ".json");
    const auto other_path = directory / ("journal." + std::to_string(1 - recovered_slot) + ".json");
    active = read_json(recovered_path);
    auto divergent = active;
    divergent["session_id"] = "session:divergent";
    finalize_test_digest(divergent, "journal_digest");
    write_json(other_path, divergent);
    require_error([&] {
        static_cast<void>(authenticated_session(
            state.path(), "status", nullptr, nullptr, engine::Json::array(), "divergent-status"));
    }, "session.recovery_ambiguous");

    auto critical = active;
    auto extension = engine::Json{
        {"extension_id", "future-critical"}, {"extension_version", "v2"},
        {"critical", true}, {"payload", engine::Json{{"future", true}}},
        {"payload_digest", nullptr},
    };
    extension["payload_digest"] = engine::tagged_sha256(extension.at("payload").dump());
    critical["extensions"].push_back(extension);
    finalize_test_digest(critical, "journal_digest");
    write_json(other_path, critical);
    require_error([&] {
        static_cast<void>(authenticated_session(
            state.path(), "status", nullptr, nullptr, engine::Json::array(), "critical-status"));
    }, "session.compatibility_required");

    require(begin.at("canonical") == false, "slot safety fixture changed canonical state");
}

void test_authenticated_session_empty_recovery_cas() {
    TemporaryDirectory repository;
    TemporaryDirectory state;
    CurrentDirectory current(repository.path());
    const auto status = authenticated_session(
        state.path(), "status", nullptr, nullptr, engine::Json::array(), "empty-status");
    require(status.at("journal_present") == false && status.at("read_only") == true,
            "absent authenticated session status was not read-only");
    require(!fs::exists(state.path() / "symphony"),
            "absent authenticated session status created operational state");
    const std::string absent_digest =
        "sha256:0000000000000000000000000000000000000000000000000000000000000000";
    require_error([&] {
        static_cast<void>(authenticated_session(
            state.path(), "recover", "empty-exact-recover", absent_digest,
            engine::Json::array(), "empty-exact-recover"));
    }, "session.expected_state_mismatch");
    const auto discovered = authenticated_session(
        state.path(), "recover", "empty-discovery-recover", "discover",
        engine::Json::array(), "empty-discovery-recover");
    require(discovered.at("journal_present") == false && discovered.at("changed") == false,
            "empty discovery recovery did not remain a non-mutating absent result");
}

void test_reconciliation_lifecycle_and_compatibility() {
    TemporaryDirectory repository;
    TemporaryDirectory state;
    {
        std::ofstream output(repository.path() / "INTENT.md", std::ios::binary);
        output << "first\n";
    }
    CurrentDirectory current(repository.path());

    const auto compatibility = reconcile(
        state.path(), "compatibility", nullptr, nullptr);
    require(compatibility.at("compatibility").at("mode") == "full",
            "full compatibility was not negotiated");
    require(compatibility.at("journal_present") == false,
            "compatibility unexpectedly created a journal");

    const auto begin = reconcile(
        state.path(), "begin", "begin-1", "absent",
        engine::Json::array({"INTENT.md"}));
    require(begin.at("changed") == true, "begin did not commit");
    require(begin.at("journal").at("generation") == 1, "begin generation mismatch");
    const auto begin_digest = begin.at("journal_digest").get<std::string>();

    const auto replay = reconcile(
        state.path(), "begin", "begin-1", "absent",
        engine::Json::array({"INTENT.md"}));
    require(replay.at("changed") == false && replay.at("journal_digest") == begin_digest,
            "idempotent begin replay changed state");

    {
        std::ofstream output(repository.path() / "INTENT.md", std::ios::binary);
        output << "second\n";
    }
    const auto checkpoint = reconcile(
        state.path(), "checkpoint", "checkpoint-1", begin_digest,
        engine::Json::array(), "0.2.0-dev");
    require(checkpoint.at("changed") == true, "checkpoint did not commit");
    require(checkpoint.at("journal").at("checkpoints").back().at("changed_file_count") == 1,
            "content drift was not recorded");
    require(checkpoint.at("journal").at("engine_inventory").at("entries")[0].at("version") ==
                "0.2.0-dev",
            "out-of-order engine upgrade inventory was not recorded");
    const auto checkpoint_digest = checkpoint.at("journal_digest").get<std::string>();

    require_error([&] {
        static_cast<void>(reconcile(
            state.path(), "checkpoint", "checkpoint-stale", begin_digest));
    }, "reconcile.expected_state_mismatch");

    const auto read_only = reconcile(
        state.path(), "status", nullptr, nullptr,
        engine::Json::array(), "0.2.0-dev", false);
    require(read_only.at("compatibility").at("mode") == "read_only",
            "missing capabilities did not negotiate read-only mode");
    require_error([&] {
        static_cast<void>(reconcile(
            state.path(), "checkpoint", "checkpoint-incompatible", checkpoint_digest,
            engine::Json::array(), "0.2.0-dev", false));
    }, "reconcile.compatibility_required");

    const auto close = reconcile(
        state.path(), "close", "close-1", checkpoint_digest,
        engine::Json::array(), "0.2.0-dev");
    require(close.at("journal").at("state") == "closed", "close did not persist");
    const auto closed_digest = close.at("journal_digest").get<std::string>();

    const auto healthy = reconcile(
        state.path(), "recover", "recover-healthy", closed_digest,
        engine::Json::array(), "0.2.0-dev");
    require(healthy.at("changed") == false && healthy.at("recovered") == false,
            "healthy recovery was not a no-op");
}

void test_reconciliation_discovery_recovery_and_replay() {
    TemporaryDirectory repository;
    TemporaryDirectory state;
    {
        std::ofstream output(repository.path() / "INTENT.md", std::ios::binary);
        output << "recovery\n";
    }
    CurrentDirectory current(repository.path());
    const auto begin = reconcile(
        state.path(), "begin", "begin-recovery", "absent",
        engine::Json::array({"INTENT.md"}));
    const auto begin_digest = begin.at("journal_digest").get<std::string>();
    const auto checkpoint = reconcile(
        state.path(), "checkpoint", "checkpoint-recovery", begin_digest);
    const auto checkpoint_digest = checkpoint.at("journal_digest").get<std::string>();

    const auto context = context_path(state.path(), repository.path());
    {
        std::ofstream output(context / ".head.tmp-crash", std::ios::binary);
        output << "interrupted head\n";
    }
    fs::permissions(
        context / ".head.tmp-crash",
        fs::perms::owner_read | fs::perms::owner_write,
        fs::perm_options::replace);
    {
        std::ofstream output(context / "head.json", std::ios::binary);
        output << "{broken\n";
    }
    require_error([&] {
        static_cast<void>(reconcile(state.path(), "status", nullptr, nullptr));
    }, "reconcile.state_json_invalid");

    const auto recovered = reconcile(
        state.path(), "recover", "recover-discovery", "discover");
    require(recovered.at("changed") == true && recovered.at("recovered") == true,
            "discovery recovery did not repair the journal");
    require(recovered.at("journal").at("previous_journal_digest") == checkpoint_digest,
            "recovery did not preserve the selected predecessor");
    require(!fs::exists(context / ".head.tmp-crash"),
            "recovery did not remove the stale atomic-head temporary");

    const auto replay = reconcile(
        state.path(), "recover", "recover-discovery", "discover");
    require(replay.at("changed") == false,
            "replayed discovery recovery committed a second checkpoint");
}

void test_reconciliation_extension_and_filesystem_safety() {
    TemporaryDirectory repository;
    TemporaryDirectory state;
    {
        std::ofstream output(repository.path() / "INTENT.md", std::ios::binary);
        output << "extension\n";
    }
    CurrentDirectory current(repository.path());
    const auto begin = reconcile(
        state.path(), "begin", "begin-extension", "absent",
        engine::Json::array({"INTENT.md"}));
    const auto context = context_path(state.path(), repository.path());
    auto head = read_json(context / "head.json");
    const auto slot = head.at("active_slot").get<int>();
    const auto journal_path = context / ("journal." + std::to_string(slot) + ".json");
    auto journal = read_json(journal_path);
    engine::Json extension{
        {"extension_id", "future-compatible-evidence"},
        {"extension_version", "2.0.0"},
        {"critical", false},
        {"payload", engine::Json{{"future_field", "preserve-verbatim"}}},
    };
    extension["payload_digest"] = engine::tagged_sha256(extension.at("payload").dump());
    journal["extensions"].push_back(extension);
    finalize_test_digest(journal, "journal_digest");
    head["journal_digest"] = journal.at("journal_digest");
    finalize_test_digest(head, "head_digest");
    write_json(journal_path, journal);
    write_json(context / "head.json", head);

    const auto extended_digest = journal.at("journal_digest").get<std::string>();
    const auto checkpoint = reconcile(
        state.path(), "checkpoint", "checkpoint-extension", extended_digest);
    require(checkpoint.at("journal").at("extensions").size() == 1,
            "noncritical future extension was not preserved");
    require(checkpoint.at("journal").at("extensions")[0] == extension,
            "noncritical future extension changed during the write");

    head = read_json(context / "head.json");
    const auto critical_slot = head.at("active_slot").get<int>();
    const auto critical_path =
        context / ("journal." + std::to_string(critical_slot) + ".json");
    journal = read_json(critical_path);
    journal["extensions"][0]["critical"] = true;
    finalize_test_digest(journal, "journal_digest");
    head["journal_digest"] = journal.at("journal_digest");
    finalize_test_digest(head, "head_digest");
    write_json(critical_path, journal);
    write_json(context / "head.json", head);
    require_error([&] {
        static_cast<void>(reconcile(state.path(), "status", nullptr, nullptr));
    }, "reconcile.critical_extension_unknown");

    TemporaryDirectory unsafe_state;
    const auto link = unsafe_state.path().parent_path() /
                      ("symphony-state-link-" + std::to_string(::getpid()));
    std::error_code ignored;
    fs::remove(link, ignored);
    if (::symlink(unsafe_state.path().c_str(), link.c_str()) != 0) {
        throw std::runtime_error("could not create state-root symlink");
    }
    auto unsafe_payload = reconciliation_payload(
        unsafe_state.path(), "status", nullptr, nullptr, engine::Json::array());
    unsafe_payload["state_root"] = link.string();
    require_error([&] {
        static_cast<void>(session::handle_request(request("status", unsafe_payload)));
    }, "reconcile.state_open_failed");
    fs::remove(link, ignored);
}

void test_reconciliation_lock_and_worktree_isolation() {
    TemporaryDirectory repository_one;
    TemporaryDirectory repository_two;
    TemporaryDirectory state;
    for (const auto& repository : {repository_one.path(), repository_two.path()}) {
        std::ofstream output(repository / "INTENT.md", std::ios::binary);
        output << repository.string() << '\n';
    }
    engine::Json first;
    {
        CurrentDirectory current(repository_one.path());
        first = reconcile(
            state.path(), "begin", "begin-one", "absent",
            engine::Json::array({"INTENT.md"}));
        const auto lock_path =
            context_path(state.path(), repository_one.path()) / "journal.lock";
        const int lock = ::open(lock_path.c_str(), O_RDWR | O_CLOEXEC);
        if (lock < 0 || ::flock(lock, LOCK_EX | LOCK_NB) != 0) {
            if (lock >= 0) ::close(lock);
            throw std::runtime_error("could not acquire contention test lock");
        }
        require_error([&] {
            static_cast<void>(reconcile(state.path(), "status", nullptr, nullptr));
        }, "reconcile.busy");
        static_cast<void>(::flock(lock, LOCK_UN));
        ::close(lock);
    }
    engine::Json second;
    {
        CurrentDirectory current(repository_two.path());
        second = reconcile(
            state.path(), "begin", "begin-two", "absent",
            engine::Json::array({"INTENT.md"}));
    }
    require(first.at("journal").at("context_id") != second.at("journal").at("context_id"),
            "separate worktrees shared a reconciliation context");
    require(
        context_path(state.path(), repository_one.path()) !=
            context_path(state.path(), repository_two.path()),
        "separate worktrees shared a state path");
}

void test_reconciliation_refuses_future_and_ambiguous_state() {
    {
        TemporaryDirectory repository;
        TemporaryDirectory state;
        std::ofstream(repository.path() / "INTENT.md", std::ios::binary) << "future\n";
        CurrentDirectory current(repository.path());
        static_cast<void>(reconcile(
            state.path(), "begin", "begin-future", "absent",
            engine::Json::array({"INTENT.md"})));
        const auto context = context_path(state.path(), repository.path());
        auto head = read_json(context / "head.json");
        const auto slot = head.at("active_slot").get<int>();
        const auto journal_path =
            context / ("journal." + std::to_string(slot) + ".json");
        auto journal = read_json(journal_path);
        journal["protocol"] = "symphony.knowledge.reconciliation-journal.v2";
        journal["format_version"] = 2;
        finalize_test_digest(journal, "journal_digest");
        head["journal_digest"] = journal.at("journal_digest");
        finalize_test_digest(head, "head_digest");
        write_json(journal_path, journal);
        write_json(context / "head.json", head);
        const auto preserved = read_json(journal_path);
        require_error([&] {
            static_cast<void>(reconcile(state.path(), "status", nullptr, nullptr));
        }, "reconcile.journal_incompatible");
        require_error([&] {
            static_cast<void>(reconcile(
                state.path(), "recover", "recover-future", "discover"));
        }, "reconcile.compatibility_required");
        require(read_json(journal_path) == preserved,
                "future journal was changed by incompatible recovery");
    }
    {
        TemporaryDirectory repository;
        TemporaryDirectory state;
        std::ofstream(repository.path() / "INTENT.md", std::ios::binary) << "ambiguous\n";
        CurrentDirectory current(repository.path());
        static_cast<void>(reconcile(
            state.path(), "begin", "begin-ambiguous", "absent",
            engine::Json::array({"INTENT.md"})));
        const auto context = context_path(state.path(), repository.path());
        auto head = read_json(context / "head.json");
        const auto active = head.at("active_slot").get<int>();
        const auto other = 1 - active;
        auto divergent = read_json(
            context / ("journal." + std::to_string(active) + ".json"));
        divergent["journal_id"] =
            "journal:" + engine::sha256_hex("equally-ranked-divergent-journal");
        finalize_test_digest(divergent, "journal_digest");
        const auto other_path =
            context / ("journal." + std::to_string(other) + ".json");
        write_json(other_path, divergent);
        fs::permissions(
            other_path,
            fs::perms::owner_read | fs::perms::owner_write,
            fs::perm_options::replace);
        fs::remove(context / "head.json");
        require_error([&] {
            static_cast<void>(reconcile(
                state.path(), "recover", "recover-ambiguous", "discover"));
        }, "reconcile.recovery_ambiguous");
        require(fs::exists(other_path),
                "ambiguous recovery removed evidence");
    }
}

void test_reconciliation_adopts_interrupted_successor() {
    TemporaryDirectory repository;
    TemporaryDirectory state;
    std::ofstream(repository.path() / "INTENT.md", std::ios::binary) << "successor\n";
    CurrentDirectory current(repository.path());
    const auto begin = reconcile(
        state.path(), "begin", "begin-successor", "absent",
        engine::Json::array({"INTENT.md"}));
    const auto begin_digest = begin.at("journal_digest").get<std::string>();
    const auto context = context_path(state.path(), repository.path());
    const auto prior_head = read_json(context / "head.json");
    static_cast<void>(reconcile(
        state.path(), "checkpoint", "checkpoint-successor", begin_digest));
    write_json(context / "head.json", prior_head);

    require_error([&] {
        static_cast<void>(reconcile(state.path(), "status", nullptr, nullptr));
    }, "reconcile.recovery_required");
    const auto recovered = reconcile(
        state.path(), "recover", "recover-successor", begin_digest);
    require(recovered.at("changed") == true && recovered.at("recovered") == true,
            "linked successor was not recovered");
    require(recovered.at("journal").at("recovery").at("disposition") ==
                "adopted_linked_successor",
            "linked successor recovery disposition mismatch");
    const auto replay = reconcile(
        state.path(), "recover", "recover-successor", begin_digest);
    require(replay.at("changed") == false,
            "linked successor recovery replay changed state");
}

}

int main() {
    try {
        test_descriptor_and_inspect();
        test_check();
        test_reserved_operations();
        test_authenticated_session_lifecycle_and_recovery();
        test_authenticated_session_slot_safety();
        test_authenticated_session_empty_recovery_cas();
        test_reconciliation_lifecycle_and_compatibility();
        test_reconciliation_discovery_recovery_and_replay();
        test_reconciliation_extension_and_filesystem_safety();
        test_reconciliation_lock_and_worktree_isolation();
        test_reconciliation_refuses_future_and_ambiguous_state();
        test_reconciliation_adopts_interrupted_successor();
        std::cout << "knowledge session coordinator tests passed\n";
        return 0;
    } catch (const std::exception& error) {
        std::cerr << "test failure: " << error.what() << '\n';
        return 1;
    }
}
