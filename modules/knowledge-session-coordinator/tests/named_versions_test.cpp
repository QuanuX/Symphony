#include "coordinator.hpp"

#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/protocol.hpp"

#include <array>
#include <filesystem>
#include <iostream>
#include <stdexcept>
#include <string>
#include <unistd.h>

namespace fs = std::filesystem;
namespace session = symphony::knowledge::session;
namespace engine = symphony::knowledge::engine;

namespace {

class TemporaryDirectory final {
public:
    TemporaryDirectory() {
        std::string pattern = (fs::canonical(fs::temp_directory_path()) / "symphony-named-version-test-XXXXXX").string();
        pattern.push_back('\0');
        char* result = ::mkdtemp(pattern.data());
        if (result == nullptr) throw std::runtime_error("mkdtemp failed");
        path_ = result;
    }
    ~TemporaryDirectory() {
        std::error_code ignored;
        fs::remove_all(path_, ignored);
    }
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

void digest(engine::Json& value, const char* field) {
    value.erase(field);
    value[field] = engine::tagged_sha256(value.dump());
}

engine::Json artifact(const std::string& id = "savver:symphony:baseline.v1",
                      const engine::Json& predecessor = nullptr) {
    engine::Json value{
        {"protocol", "symphony.sav.named-version.v1"}, {"named_version_id", id},
        {"alias", "Symphony Baseline"}, {"predecessor_digest", predecessor},
        {"component_requirements", engine::Json::array()}, {"contract_requirements", engine::Json::array()},
        {"accord_reference_ids", engine::Json::array()}, {"required_traits", engine::Json::array()},
        {"extension_points", engine::Json::array()},
        {"platform_bounds", engine::Json::array({"platform:symphony:darwin"})},
        {"thermal_restriction", "freezing_only"}, {"sealed_at", "2026-08-21T08:30:00Z"},
        {"composition_authority_reference", "authority:symphony:architect"},
        {"sodv_publication_reference", nullptr}, {"named_version_digest", nullptr},
    };
    digest(value, "named_version_digest");
    return value;
}

engine::Json validation(const engine::Json& value) {
    engine::Json result{
        {"protocol", "symphony.sav.named-version-validation-result.v1"},
        {"named_version_id", value.at("named_version_id")},
        {"named_version_digest", value.at("named_version_digest")},
        {"state", "valid_immutable_envelope"}, {"read_only", true},
        {"seal_authorized", false}, {"result_digest", nullptr},
    };
    digest(result, "result_digest");
    return result;
}

engine::Json engine_evidence() {
    engine::Json result{
        {"module_id", "sav-engine"}, {"engine_id", "symphony-sav"}, {"vector_id", "sav"},
        {"version", "0.1.0-dev"},
        {"receipt_digest", "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
        {"executable_digest", "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
        {"evidence_digest", nullptr},
    };
    digest(result, "evidence_digest");
    return result;
}

engine::Json client() {
    return engine::Json{
        {"client_id", "qxctl"}, {"client_version", "qxctl-dev"},
        {"process_protocols", engine::Json::array({engine::process_protocol_v1})},
        {"registry_read_versions", engine::Json::array({1})},
        {"registry_write_versions", engine::Json::array({1})},
        {"capabilities", engine::Json::array({
            "atomic-head-v1", "content-addressed-named-version-v1", "dual-slot-registry-v1",
            "expected-state-cas-v1", "idempotent-operation-v1", "immutable-object-v1",
            "opaque-extension-preservation-v1", "recovery-forward-v1", "ssiag-capability-binding-v1",
        })},
    };
}

std::string resource(const engine::Json& payload) {
    const engine::Json normalized{
        {"tops_id", payload.at("tops_id")}, {"operation", payload.at("operation")},
        {"expected_registry_digest", payload.at("expected_registry_digest")},
        {"named_version_digest", payload.at("named_version").is_object() ?
            payload.at("named_version").at("named_version_digest") : engine::Json(nullptr)},
        {"prepared_operation_id", payload.at("prepared_operation_id")},
        {"proposal_digest", payload.at("proposal_digest")}, {"alias", payload.at("alias")},
        {"selector", payload.at("selector")},
    };
    return "symphony.knowledge.named-version:" + engine::sha256_hex(normalized.dump());
}

engine::Json authorization(const std::string& operation, const std::string& resource_value,
                           const std::string& suffix) {
    const std::string tops_id = "018f0c3a-7b2d-7e11-8c12-0242ac120002";
    const auto request_id = "request-" + suffix;
    const auto correlation_id = "correlation-" + suffix;
    const auto target = engine::Json{
        {"operation", "symphony.knowledge." + operation}, {"resource", resource_value},
        {"audience", "qxctl"}, {"scope", "tops:" + tops_id},
    };
    const auto subject = engine::Json{{"id", "owner.primary"}, {"kind", "symphony.identity.owner"},
                                      {"authority", "unix_peer_credentials"}};
    engine::Json capability{
        {"protocol", "symphony.ssiag.capability.v1"}, {"capability_id", "pending"},
        {"subject", subject}, {"tops_id", tops_id}, {"target", target},
        {"authority_basis", "host_owner"}, {"grant_id", "named-version-" + suffix},
        {"request_id", request_id}, {"correlation_id", correlation_id},
        {"issued_at", "2020-01-01T00:00:00Z"}, {"expires_at", "2099-01-01T00:00:00Z"},
        {"policy_digest", "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"},
        {"config_digest", "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"},
        {"binding_digest", "pending"}, {"transferable", false}, {"canonical_apply", false},
    };
    const std::array<std::string, 19> values{
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
    capability["capability_id"] = "ssiag-capability:" +
        capability.at("binding_digest").get<std::string>().substr(7U);
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

engine::Json command(const fs::path& state_root, const std::string& operation, const engine::Json& operation_id,
                     const engine::Json& expected, const engine::Json& version = nullptr,
                     const engine::Json& validation_result = nullptr, const engine::Json& sav = nullptr,
                     const engine::Json& prepared_operation_id = nullptr,
                     const engine::Json& proposal_digest = nullptr, const engine::Json& alias = nullptr,
                     const engine::Json& selector = nullptr, const std::string& suffix = "1") {
    engine::Json result{
        {"protocol", "symphony.knowledge.named-version-command.v1"}, {"operation", operation},
        {"state_root", fs::canonical(state_root).string()},
        {"tops_id", "018f0c3a-7b2d-7e11-8c12-0242ac120002"},
        {"operation_id", operation_id}, {"expected_registry_digest", expected},
        {"named_version", version}, {"validation_result", validation_result}, {"sav_engine", sav},
        {"prepared_operation_id", prepared_operation_id}, {"proposal_digest", proposal_digest},
        {"alias", alias}, {"selector", selector}, {"authorization_decision", nullptr}, {"client", client()},
    };
    result["authorization_decision"] = authorization(operation, resource(result), suffix);
    return result;
}

engine::Json invoke(const std::string& operation, engine::Json payload) {
    return session::handle_request(engine::Request{
        "request-1", "correlation-1", operation, session::engine_id,
        engine::unix_time_ms() + 60000, std::move(payload),
    });
}

} // namespace

int main() {
    try {
        TemporaryDirectory state_root;
        const auto version = artifact();

        const auto absent = invoke("named_version_status", command(state_root.path(),
            "named_version_status", nullptr, nullptr, nullptr, nullptr, nullptr, nullptr, nullptr,
            nullptr, nullptr, "status-absent"));
        require(!absent.at("registry_present").get<bool>() && absent.at("read_only").get<bool>(),
                "absent status is inconsistent");

        const auto prepare_payload = command(state_root.path(), "named_version_prepare", "prepare-1", "absent",
            version, validation(version), engine_evidence(), nullptr, nullptr, nullptr, nullptr, "prepare-1");
        const auto prepared = invoke("named_version_prepare", prepare_payload);
        require(prepared.at("changed").get<bool>() && prepared.at("proposal_digest").is_string(),
                "prepare did not persist a proposal");
        const auto prepared_replay = invoke("named_version_prepare", prepare_payload);
        require(!prepared_replay.at("changed").get<bool>() &&
                prepared_replay.at("proposal_digest") == prepared.at("proposal_digest"),
                "prepare replay was not idempotent");

        const auto sealed = invoke("named_version_seal", command(state_root.path(), "named_version_seal",
            "seal-1", "absent", nullptr, nullptr, nullptr, "prepare-1", prepared.at("proposal_digest"),
            nullptr, nullptr, "seal-1"));
        require(sealed.at("changed").get<bool>() && sealed.at("registry_present").get<bool>() &&
                sealed.at("artifact").at("named_version_digest") == version.at("named_version_digest"),
                "seal did not commit the immutable artifact");
        const auto sealed_digest = sealed.at("registry_digest");

        const auto lookup = invoke("named_version_lookup", command(state_root.path(), "named_version_lookup",
            nullptr, nullptr, nullptr, nullptr, nullptr, nullptr, nullptr, nullptr,
            engine::Json{{"kind", "id"}, {"value", version.at("named_version_id")}}, "lookup-id"));
        require(lookup.at("read_only").get<bool>() && lookup.at("artifact") == version,
                "ID lookup did not return exact stored bytes");

        const auto alias_payload = command(state_root.path(), "named_version_alias", "alias-1", sealed_digest,
            nullptr, nullptr, nullptr, nullptr, nullptr, "stable",
            engine::Json{{"kind", "digest"}, {"value", version.at("named_version_digest")}}, "alias-1");
        const auto selected = invoke("named_version_alias", alias_payload);
        require(selected.at("changed").get<bool>() && selected.at("selected_alias") == "stable",
                "alias selection did not commit");
        const auto alias_replay = invoke("named_version_alias", alias_payload);
        require(!alias_replay.at("changed").get<bool>(), "alias replay was not idempotent");

        const auto alias_lookup = invoke("named_version_lookup", command(state_root.path(), "named_version_lookup",
            nullptr, nullptr, nullptr, nullptr, nullptr, nullptr, nullptr, nullptr,
            engine::Json{{"kind", "alias"}, {"value", "stable"}}, "lookup-alias"));
        require(alias_lookup.at("artifact") == version && alias_lookup.at("selected_alias") == "stable",
                "alias lookup did not resolve exact bytes");

        require_error([&] {
            static_cast<void>(invoke("named_version_alias", command(state_root.path(), "named_version_alias",
                "alias-stale", sealed_digest, nullptr, nullptr, nullptr, nullptr, nullptr, "other",
                engine::Json{{"kind", "digest"}, {"value", version.at("named_version_digest")}}, "alias-stale")));
        }, "named_version.stale_expected_state");

        const auto store = state_root.path() / "symphony" / "knowledge-session-coordinator" / "accordare" /
            "v1" / "tops" / engine::sha256_hex("tops:018f0c3a-7b2d-7e11-8c12-0242ac120002") /
            "named-versions";
        fs::remove(store / "head.json");
        const auto recovered = invoke("named_version_recover", command(state_root.path(), "named_version_recover",
            "recover-1", "discover", nullptr, nullptr, nullptr, nullptr, nullptr, nullptr, nullptr, "recover-1"));
        require(recovered.at("changed").get<bool>() && recovered.at("recovered").get<bool>() &&
                recovered.at("registry_present").get<bool>(), "head-loss recovery did not commit forward");

        const auto object_path = store / ("object." +
            version.at("named_version_digest").get<std::string>().substr(7U) + ".json");
        fs::remove(object_path);
        fs::create_symlink("/etc/passwd", object_path);
        require_error([&] {
            static_cast<void>(invoke("named_version_lookup", command(state_root.path(), "named_version_lookup",
                nullptr, nullptr, nullptr, nullptr, nullptr, nullptr, nullptr, nullptr,
                engine::Json{{"kind", "id"}, {"value", version.at("named_version_id")}}, "lookup-symlink")));
        }, "named_version.state_read_failed");

        std::cout << "knowledge Named Version persistence tests passed\n";
        return 0;
    } catch (const std::exception& error) {
        std::cerr << "knowledge Named Version persistence tests failed: " << error.what() << '\n';
        return 1;
    }
}
