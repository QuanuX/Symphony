#include "coordinator.hpp"

#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/protocol.hpp"

#include <array>
#include <filesystem>
#include <fstream>
#include <iostream>
#include <stdexcept>
#include <string>
#include <sys/stat.h>
#include <unistd.h>

namespace fs = std::filesystem;
namespace session = symphony::knowledge::session;
namespace engine = symphony::knowledge::engine;

namespace {

constexpr const char* tops_id = "018f0c3a-7b2d-7e11-8c12-0242ac120002";
constexpr const char* subject_id = "owner.primary";
constexpr const char* session_digest =
    "sha256:1111111111111111111111111111111111111111111111111111111111111111";
constexpr const char* binding_digest =
    "sha256:2222222222222222222222222222222222222222222222222222222222222222";

class TemporaryDirectory final {
public:
    TemporaryDirectory() {
        std::string pattern = (fs::canonical(fs::temp_directory_path()) /
                               "symphony-ssfv-maintenance-test-XXXXXX").string();
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

class CurrentDirectory final {
public:
    explicit CurrentDirectory(const fs::path& path) : prior_(fs::current_path()) {
        fs::current_path(path);
    }
    ~CurrentDirectory() {
        std::error_code ignored;
        fs::current_path(prior_, ignored);
    }
    CurrentDirectory(const CurrentDirectory&) = delete;
    CurrentDirectory& operator=(const CurrentDirectory&) = delete;
private:
    fs::path prior_;
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

void finalize(engine::Json& value, const char* field) {
    value.erase(field);
    value[field] = engine::tagged_sha256(value.dump());
}

engine::Request request(const std::string& operation, engine::Json payload) {
    return engine::Request{
        "request-1", "correlation-1", operation, session::engine_id,
        engine::unix_time_ms() + 60000, std::move(payload),
    };
}

engine::Json engine_evidence(const std::string& version) {
    engine::Json value{
        {"module_id", "ssfv-engine"}, {"engine_id", "symphony-ssfv"},
        {"vector_id", "ssfv"}, {"version", version},
        {"receipt_digest", "sha256:3333333333333333333333333333333333333333333333333333333333333333"},
        {"executable_digest", "sha256:4444444444444444444444444444444444444444444444444444444444444444"},
    };
    finalize(value, "evidence_digest");
    return value;
}

engine::Json semantic_snapshot(const std::string& version, const std::string& source_digit) {
    engine::Json value{
        {"protocol", "symphony.ssfv.semantic-snapshot.v1"},
        {"module_id", "ssfv-engine"}, {"engine_id", "symphony-ssfv"},
        {"engine_version", version}, {"vector_id", "ssfv"},
        {"contract_digest", "sha256:5555555555555555555555555555555555555555555555555555555555555555"},
        {"namespace_digest", "sha256:6666666666666666666666666666666666666666666666666666666666666666"},
        {"registry_digest", "sha256:7777777777777777777777777777777777777777777777777777777777777777"},
        {"source_digest", "sha256:" + std::string(64U, source_digit.front())},
        {"feature_files", engine::Json::array()}, {"records", engine::Json::array()},
    };
    finalize(value, "snapshot_digest");
    return value;
}

engine::Json maestro_not_configured() {
    engine::Json value{
        {"availability", "not_configured"},
        {"reason", "Maestro inventory observation was not configured for this checkpoint"},
        {"observation", nullptr},
    };
    finalize(value, "evidence_digest");
    return value;
}

engine::Json diff_result(const engine::Json& baseline, const engine::Json& current,
                         const std::string& state = "identical") {
    engine::Json value{
        {"protocol", "symphony.ssfv.diff-result.v2"},
        {"baseline_digest", baseline.at("snapshot_digest")},
        {"current_digest", current.at("snapshot_digest")}, {"state", state},
        {"added_feature_ids", engine::Json::array()},
        {"changed_feature_ids", engine::Json::array()},
        {"removed_feature_ids", engine::Json::array()},
        {"uncovered_paths", engine::Json::array()}, {"stale_references", engine::Json::array()},
        {"semantic_candidates", engine::Json::array()},
        {"summary", engine::Json{{"added", 0}, {"changed", 0}, {"removed", 0},
            {"uncovered", 0}, {"stale", 0}, {"review_required", 0}}},
        {"read_only", true}, {"noncanonical", true},
    };
    finalize(value, "result_digest");
    return value;
}

std::string maintenance_resource(const fs::path& repository, const std::string& operation,
                                 const std::string& expected, const std::string& session,
                                 const std::string& snapshot, const std::string& inventory) {
    return "symphony.knowledge.ssfv-maintenance:" + engine::sha256_hex(
        std::string(tops_id) + "\n" + fs::canonical(repository).string() + "\n" + operation +
        "\n" + expected + "\n" + session + "\n" + snapshot + "\n" + inventory);
}

engine::Json authorization(const std::string& operation, const std::string& resource,
                           const std::string& suffix) {
    const auto request_id = "request-" + suffix;
    const auto correlation_id = "correlation-" + suffix;
    const auto target = engine::Json{
        {"operation", "symphony.knowledge." + operation}, {"resource", resource},
        {"audience", "qxctl"}, {"scope", "tops:" + std::string(tops_id)},
    };
    const auto subject = engine::Json{
        {"id", subject_id}, {"kind", "symphony.identity.owner"},
        {"authority", "unix_peer_credentials"},
    };
    engine::Json capability{
        {"protocol", "symphony.ssiag.capability.v1"}, {"capability_id", "pending"},
        {"subject", subject}, {"tops_id", tops_id}, {"target", target},
        {"authority_basis", "host_owner"}, {"grant_id", "ssfv-maintenance-" + suffix},
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
    capability["capability_id"] =
        "ssiag-capability:" + capability.at("binding_digest").get<std::string>().substr(7U);
    return engine::Json{
        {"schema", "symphony.ssiag.authorization-decision.v1"},
        {"decision_id", "ssiag-decision:" + engine::sha256_hex(operation + suffix)},
        {"request_id", request_id}, {"correlation_id", correlation_id}, {"tops_id", tops_id},
        {"subject", subject}, {"target", target}, {"effect", "allow"},
        {"reason_code", "symphony.ssiag.policy.exact-grant"}, {"authority_basis", "host_owner"},
        {"capability", capability}, {"policy_digest", capability.at("policy_digest")},
        {"config_digest", capability.at("config_digest")}, {"decided_at", "2020-01-01T00:00:00Z"},
        {"expires_at", capability.at("expires_at")}, {"caller_class_used", false},
        {"canonical_apply", false},
    };
}

engine::Json client() {
    return engine::Json{
        {"client_id", "qxctl"}, {"client_version", "qxctl-dev"},
        {"process_protocols", engine::Json::array({engine::process_protocol_v1})},
        {"journal_read_versions", engine::Json::array({1})},
        {"journal_write_versions", engine::Json::array({1})},
        {"capabilities", engine::Json::array({
            "atomic-head-v1", "content-addressed-ssfv-baseline-v1", "dual-slot-journal-v1",
            "expected-state-cas-v1", "idempotent-operation-v1", "maestro-inventory-lineage-v1",
            "opaque-extension-preservation-v1", "recovery-forward-v1",
            "ssiag-capability-binding-v1",
        })},
    };
}

engine::Json invoke(const fs::path& repository, const fs::path& state,
                    const std::string& operation, const engine::Json& operation_id,
                    const engine::Json& expected, const engine::Json& selected_engine,
                    const engine::Json& snapshot, const engine::Json& diff,
                    const engine::Json& inventory, const std::string& suffix) {
    const bool status = operation == "ssfv_maintenance_status";
    const bool recover = operation == "ssfv_maintenance_recover";
    const auto expected_resource = status ? "status" : expected.get<std::string>();
    const auto session_resource = status ? "none" : session_digest;
    const auto snapshot_resource = status || recover ? "none" :
        snapshot.at("snapshot_digest").get<std::string>();
    const auto inventory_resource = status || recover ? "none" :
        inventory.at("evidence_digest").get<std::string>();
    const auto resource = maintenance_resource(repository, operation, expected_resource,
        session_resource, snapshot_resource, inventory_resource);
    return session::handle_request(request(operation, engine::Json{
        {"protocol", "symphony.knowledge.ssfv-maintenance-command.v1"},
        {"operation", operation}, {"state_root", fs::canonical(state).string()},
        {"tops_id", tops_id}, {"operation_id", operation_id},
        {"expected_journal_digest", expected},
        {"repository_root", fs::canonical(repository).string()},
        {"session_journal_digest", status ? engine::Json(nullptr) : engine::Json(session_digest)},
        {"binding_registry_digest", status || recover ? engine::Json(nullptr) : engine::Json(binding_digest)},
        {"engine", status || recover ? engine::Json(nullptr) : selected_engine},
        {"semantic_snapshot", status || recover ? engine::Json(nullptr) : snapshot},
        {"diff_result", status || recover ? engine::Json(nullptr) : diff},
        {"maestro_inventory", status || recover ? engine::Json(nullptr) : inventory},
        {"authorization_decision", authorization(operation, resource, suffix)},
        {"client", client()},
    }));
}

void test_maintenance_upgrade_cas_replay_and_recovery() {
    TemporaryDirectory repository;
    TemporaryDirectory state;
    CurrentDirectory current(repository.path());
    const auto old_engine = engine_evidence("0.1.0-dev");
    const auto old_snapshot = semantic_snapshot("0.1.0-dev", "8");
    const auto inventory = maestro_not_configured();

    const auto absent = invoke(repository.path(), state.path(), "ssfv_maintenance_status",
        nullptr, nullptr, nullptr, nullptr, nullptr, nullptr, "status-empty");
    require(absent.at("journal_present") == false && absent.at("read_only") == true,
            "absent maintenance status was not read-only");
    require(!fs::exists(state.path() / "symphony"), "absent status created state");

    const auto begin = invoke(repository.path(), state.path(), "ssfv_maintenance_begin",
        "maintenance-begin", "absent", old_engine, old_snapshot, nullptr, inventory, "begin");
    require(begin.at("changed") == true && begin.at("effective_state") == "open",
            "maintenance begin did not commit");
    require(begin.at("journal").at("baseline_engine").at("version") == "0.1.0-dev",
            "baseline engine identity was not persisted");
    const auto begin_digest = begin.at("journal_digest").get<std::string>();

    const auto replay = invoke(repository.path(), state.path(), "ssfv_maintenance_begin",
        "maintenance-begin", "absent", old_engine, old_snapshot, nullptr, inventory, "begin-replay");
    require(replay.at("changed") == false && replay.at("journal_digest") == begin_digest,
            "idempotent maintenance begin replay changed state");

    const auto new_engine = engine_evidence("0.2.0-dev");
    const auto new_snapshot = semantic_snapshot("0.2.0-dev", "8");
    const auto difference = diff_result(old_snapshot, new_snapshot);
    const auto checkpoint = invoke(repository.path(), state.path(), "ssfv_maintenance_checkpoint",
        "maintenance-checkpoint", begin_digest, new_engine, new_snapshot, difference, inventory, "checkpoint");
    require(checkpoint.at("changed") == true &&
                checkpoint.at("journal").at("baseline_engine").at("version") == "0.1.0-dev" &&
                checkpoint.at("journal").at("current_engine").at("version") == "0.2.0-dev",
            "out-of-order SSFV engine upgrade did not preserve baseline lineage");
    const auto checkpoint_digest = checkpoint.at("journal_digest").get<std::string>();

    require_error([&] {
        static_cast<void>(invoke(repository.path(), state.path(), "ssfv_maintenance_checkpoint",
            "maintenance-stale", begin_digest, new_engine, new_snapshot, difference, inventory, "stale"));
    }, "ssfv_maintenance.stale_expected_state");

    const auto healthy = invoke(repository.path(), state.path(), "ssfv_maintenance_status",
        nullptr, nullptr, nullptr, nullptr, nullptr, nullptr, "status-healthy");
    require(healthy.at("journal_digest") == checkpoint_digest && healthy.at("review_state") == "current",
            "maintenance status did not read upgraded journal");

    const auto context = engine::tagged_sha256(std::string(tops_id) + "\n" + subject_id + "\n" +
        fs::canonical(repository.path()).string());
    const auto directory = state.path() / "symphony" / "knowledge-session-coordinator" /
        "ssfv-maintenance" / "v1" / "contexts" / context.substr(7U);
    {
        std::ofstream output(directory / "head.json", std::ios::binary | std::ios::trunc);
        output << "{broken\n";
    }
    if (::chmod((directory / "head.json").c_str(), 0600) != 0) {
        throw std::runtime_error("could not protect corrupt head fixture");
    }
    require_error([&] {
        static_cast<void>(invoke(repository.path(), state.path(), "ssfv_maintenance_status",
            nullptr, nullptr, nullptr, nullptr, nullptr, nullptr, "status-broken"));
    }, "ssfv_maintenance.state_json_invalid");
    const auto recovered = invoke(repository.path(), state.path(), "ssfv_maintenance_recover",
        "maintenance-recover", "discover", nullptr, nullptr, nullptr, nullptr, "recover");
    require(recovered.at("changed") == true && recovered.at("recovered") == true &&
                recovered.at("journal").at("previous_journal_digest") == checkpoint_digest,
            "maintenance recovery did not roll the selected chain forward");
}

} // namespace

int main() {
    try {
        test_maintenance_upgrade_cas_replay_and_recovery();
        std::cout << "SSFV maintenance tests passed\n";
        return 0;
    } catch (const std::exception& error) {
        std::cerr << "test failure: " << error.what() << '\n';
        return 1;
    }
}
