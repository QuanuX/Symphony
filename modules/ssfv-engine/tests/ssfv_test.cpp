#include "ssfv.hpp"

#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/limits.hpp"
#include "symphony/knowledge/engine/protocol.hpp"

#include <algorithm>
#include <chrono>
#include <filesystem>
#include <fstream>
#include <functional>
#include <iomanip>
#include <iostream>
#include <sstream>
#include <stdexcept>
#include <string>
#include <unistd.h>
#include <utility>
#include <vector>

namespace fs = std::filesystem;
namespace ssfv = symphony::knowledge::ssfv;
namespace engine = symphony::knowledge::engine;

namespace {

class TemporaryDirectory final {
public:
    TemporaryDirectory() {
        std::string pattern =
            (fs::canonical(fs::temp_directory_path()) / "symphony-ssfv-test-XXXXXX").string();
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
    explicit CurrentDirectory(const fs::path& path)
        : previous_(fs::current_path()) {
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

void write_file(const fs::path& path, const std::string& contents) {
    fs::create_directories(path.parent_path());
    std::ofstream output(path, std::ios::binary);
    if (!output.good()) {
        throw std::runtime_error("could not create fixture: " + path.string());
    }
    output << contents;
}

engine::Request request(std::string operation, engine::Json payload) {
    return engine::Request{"request-1", "correlation-1", std::move(operation), ssfv::engine_id,
                           engine::unix_time_ms() + 60000, std::move(payload)};
}

const std::vector<std::string> contract_paths = {
    "knowledge/SPEC.md",
    "knowledge/ssfv/INTENT.md",
    "knowledge/ssfv/MANIFEST.md",
    "knowledge/ssfv/SKILL.md",
    "knowledge/ssfv/SPEC.md",
    "knowledge/ssfv/COVERAGE.md",
    "knowledge/ssfv/NAMESPACES.md",
    "knowledge/ssfv/REGISTRY.md",
    "knowledge/ssfv/FEATURE-FILE-FORMAT.md",
    "knowledge/ssfv/schemas/v1/MANIFEST.md",
    "knowledge/ssfv/schemas/v1/check-result.schema.json",
    "knowledge/ssfv/schemas/v1/diff-input.schema.json",
    "knowledge/ssfv/schemas/v1/diff-result.schema.json",
    "knowledge/ssfv/schemas/v1/feature-file.schema.json",
    "knowledge/ssfv/schemas/v1/feature-record.schema.json",
    "knowledge/ssfv/schemas/v1/graph-input.schema.json",
    "knowledge/ssfv/schemas/v1/graph-projection.schema.json",
    "knowledge/ssfv/schemas/v1/namespace-entry.schema.json",
    "knowledge/ssfv/schemas/v1/proposal-input.schema.json",
    "knowledge/ssfv/schemas/v1/registry-entry.schema.json",
    "knowledge/ssfv/schemas/v1/semantic-snapshot.schema.json",
    "knowledge/ssfv/schemas/v2/MANIFEST.md",
    "knowledge/ssfv/schemas/v2/check-input.schema.json",
    "knowledge/ssfv/schemas/v2/check-result.schema.json",
    "knowledge/ssfv/schemas/v2/diff-input.schema.json",
    "knowledge/ssfv/schemas/v2/diff-result.schema.json",
    "knowledge/ssfv/schemas/v2/feature-record.schema.json",
    "knowledge/ssfv/schemas/v2/proposal-input.schema.json",
    "knowledge/ssfv/schemas/v2/registry-entry.schema.json",
    "knowledge/skvi/INDEX.md",
};

engine::Json feature_record(const std::string& feature_id, const std::string& title,
                            const std::string& kind,
                            const engine::Json& parent_feature_id) {
    return engine::Json{
        {"feature_id", feature_id},
        {"record_version", 2},
        {"title", title},
        {"kind", kind},
        {"status", "implemented"},
        {"parent_feature_id", parent_feature_id},
        {"owner_contract", "modules/example/SPEC.md"},
        {"source_scope", "modules/example"},
        {"implementation_paths", engine::Json::array({"modules/example/src/example.cpp"})},
        {"implementation_languages", engine::Json::array({
            engine::Json{{"language", "C++26"}, {"role", "Deterministic fixture execution."}},
        })},
        {"who", "A caller holding effective repository permissions."},
        {"what", "A deterministic semantic fixture capability."},
        {"how", "It binds explicit records to implementation evidence."},
        {"when", "During bounded freezing-path knowledge operations."},
        {"where", "Inside the fixture repository source scope."},
        {"why", "To verify semantic feature contracts without domain inference."},
        {"relationships", engine::Json::array()},
        {"distinctions", engine::Json::array()},
        {"cross_vector_references", engine::Json::array({
            engine::Json{{"vector", "skvi"}, {"applicability", "not_applicable"},
                         {"reference", nullptr}, {"reason", "No cross-vector fixture reference."}},
        })},
        {"evidence", engine::Json::array({"fixture-owner-ratification"})},
        {"non_claims", engine::Json::array({"The engine does not decide feature worthiness."})},
    };
}

std::string feature_file(const std::vector<engine::Json>& records,
                         const std::string& owner_contract = "modules/example/SPEC.md",
                         const std::string& source_scope = "modules/example") {
    engine::Json envelope{
        {"owner_contract", owner_contract},
        {"protocol", "symphony.ssfv.feature-file.v1"},
        {"records", records},
        {"source_scope", source_scope},
    };
    return "# Symphony Semantic Features\n\n"
           "<!-- symphony:ssfv:feature-file:v1:begin -->\n"
           "```json\n" + envelope.dump(2) + "\n```\n"
           "<!-- symphony:ssfv:feature-file:v1:end -->\n";
}

std::string registry_entry(const engine::Json& record,
                           const std::string& feature_file_path = "modules/example/FEATURES.md") {
    const auto parent = record.at("parent_feature_id").is_null()
        ? "none" : record.at("parent_feature_id").get<std::string>();
    return "- feature_id: `" + record.at("feature_id").get<std::string>() + "`\n"
           "- feature_file: `" + feature_file_path + "`\n"
           "- owner_contract: `" + record.at("owner_contract").get<std::string>() + "`\n"
           "- source_scope: `" + record.at("source_scope").get<std::string>() + "`\n"
           "- status: `" + record.at("status").get<std::string>() + "`\n"
           "- parent_feature_id: `" + parent + "`\n"
           "- record_digest: `" + engine::tagged_sha256(record.dump()) + "`\n"
           "- notes: deterministic fixture\n";
}

std::vector<engine::Json> fixture_records() {
    return {
        feature_record("ssfv:symphony:fixture-capability", "Fixture Capability", "capability", nullptr),
        feature_record("ssfv:symphony:fixture-feature", "Fixture Feature", "feature",
                       "ssfv:symphony:fixture-capability"),
        feature_record("ssfv:symphony:fixture-microfeature", "Fixture Microfeature", "microfeature",
                       "ssfv:symphony:fixture-subfeature"),
        feature_record("ssfv:symphony:fixture-subfeature", "Fixture Subfeature", "subfeature",
                       "ssfv:symphony:fixture-feature"),
    };
}

void write_registry(const fs::path& root, const std::vector<engine::Json>& records,
                    const std::string& feature_file_path = "modules/example/FEATURES.md") {
    std::string entries;
    for (const auto& record : records) {
        entries += registry_entry(record, feature_file_path) + "\n";
    }
    write_file(root / "knowledge/ssfv/REGISTRY.md",
        "# Registry\n\n"
        "Inline `## Canonical Entries` references do not select a section.\n\n"
        "## Canonical Entries\n\n" + entries + "## Prohibited Entries\n\nNone.\n");
}

void create_fixture(const fs::path& root, bool with_features = true) {
    for (const auto& path : contract_paths) {
        write_file(root / path, path.ends_with(".json") ? "{}\n" : "fixture\n");
    }
    write_file(root / "knowledge/ssfv/NAMESPACES.md",
        "# Namespace Registry\n\n"
        "Inline reference to `## Canonical Namespace Entries` is not a heading.\n\n"
        "## Canonical Namespace Entries\n\n"
        "- namespace: `symphony`\n"
        "- id_prefix: `ssfv:symphony:`\n"
        "- owner_contract: `knowledge/ssfv/SPEC.md`\n"
        "- scope: deterministic fixture identities\n"
        "- status: `canonical`\n"
        "- evidence: `knowledge/ssfv/INTENT.md`\n"
        "- notes: deterministic fixture\n");
    write_file(root / "modules/example/SPEC.md", "fixture owner contract\n");
    write_file(root / "modules/example/src/example.cpp", "int fixture() { return 1; }\n");

    if (!with_features) {
        write_file(root / "knowledge/ssfv/REGISTRY.md",
            "# Registry\n\n"
            "The literal `None.` beneath `## Canonical Entries` is the empty form.\n\n"
            "## Canonical Entries\n\nNone.\n\n## Prohibited Entries\n\nNone.\n");
        write_file(root / "knowledge/skvi/INDEX.md", "# Index\n");
        return;
    }

    const auto records = fixture_records();
    write_file(root / "modules/example/FEATURES.md", feature_file(records));
    write_registry(root, records);
    write_file(root / "knowledge/skvi/INDEX.md",
               "# Index\n\n- path: `modules/example/FEATURES.md`\n");
}

engine::Json disabled_check() {
    return engine::Json{{"expected_namespace_digest", nullptr},
                        {"expected_registry_digest", nullptr},
                        {"freshness", "disabled"},
                        {"baseline", nullptr}};
}

bool has_finding(const engine::Json& check, const std::string& code) {
    return std::any_of(check.at("evidence").begin(), check.at("evidence").end(),
                       [&](const engine::Json& finding) {
                           return finding.at("code") == code;
                       });
}

void refresh_snapshot_digest(engine::Json& snapshot) {
    snapshot.erase("snapshot_digest");
    snapshot["snapshot_digest"] = engine::tagged_sha256(snapshot.dump());
}

void refresh_self_digest(engine::Json& value, const char* field) {
    value.erase(field);
    value[field] = engine::tagged_sha256(value.dump());
}

engine::Json administration_command(const std::string& command_id,
                                    const std::string& feature_id,
                                    const std::string& interaction,
                                    const std::string& operation_id) {
    return engine::Json{
        {"command_id", command_id}, {"status", "experimental"},
        {"introduced_in", "0.1.0-dev"}, {"deprecated_in", nullptr},
        {"replacement_ids", engine::Json::array()},
        {"grammar", "qxctl fixture inspect"}, {"aliases", engine::Json::array()},
        {"visibility", "public"},
        {"feature_bindings", engine::Json::array({engine::Json{
            {"feature_id", feature_id}, {"interaction", interaction}}})},
        {"infrastructure_purpose", nullptr},
        {"backend_operation_ids", engine::Json::array({operation_id})},
        {"mutability", "read_only"}, {"authority_mode", "none"},
        {"target_scope", "local"}, {"input_protocols", engine::Json::array()},
        {"output_protocols", engine::Json::array()},
        {"result_validation_protocols", engine::Json::array()},
        {"recovery_command_id", nullptr}, {"noninteractive", true},
        {"json_output", true},
    };
}

engine::Json command_registry(const std::string& kind, engine::Json commands) {
    engine::Json result{
        {"protocol", "symphony.qxctl.command-registry.v1"}, {"format_version", 1},
        {"registry_kind", kind},
        {"client_id", "qxctl"},
        {"client_version", kind == "observed" ? engine::Json("0.1.0-dev") : engine::Json(nullptr)},
        {"client_trust", "unreceipted"},
        {"executable_digest", kind == "observed"
            ? engine::Json("sha256:1111111111111111111111111111111111111111111111111111111111111111")
            : engine::Json(nullptr)},
        {"receipt_digest", nullptr}, {"commands", std::move(commands)},
    };
    refresh_self_digest(result, "registry_digest");
    return result;
}

engine::Json administration_profile(const engine::Json& snapshot,
                                    const std::string& feature_id,
                                    const std::string& command_id,
                                    const std::string& operation_id) {
    const engine::Json expectation{
        {"interaction", "inspect"}, {"requirement", "required"},
        {"delivery", "direct"},
        {"command_ids", engine::Json::array({command_id})},
        {"engine_operation_ids", engine::Json::array({operation_id})},
        {"inherited_from_feature_id", nullptr},
        {"rationale", "Fixture administration is explicitly required."},
        {"evidence", engine::Json::array({"fixture administration evidence"})},
    };
    engine::Json features = engine::Json::array();
    for (const auto& record : snapshot.at("records")) {
        const auto candidate = record.at("feature_id").get<std::string>();
        features.push_back(engine::Json{
            {"feature_id", candidate},
            {"expectations", candidate == feature_id
                ? engine::Json::array({expectation}) : engine::Json::array()},
        });
    }
    engine::Json result{
        {"protocol", "symphony.knowledge.feature-administration-profile.v1"},
        {"format_version", 1}, {"profile_id", "fixture-administration"},
        {"ssfv_registry_digest", snapshot.at("registry_digest")},
        {"catalog_scope", "registered_partial_catalog"}, {"catalog_complete", false},
        {"registered_feature_count", snapshot.at("records").size()},
        {"forward_gate", "report_only"},
        {"features", std::move(features)},
    };
    refresh_self_digest(result, "profile_digest");
    return result;
}

engine::Json fixture_engine_descriptor(const engine::Json& feature_ids,
                                       const std::string& disposition,
                                       const std::string& availability = "implemented") {
    auto result = ssfv::descriptor_v2();
    result["module_id"] = "fixture-engine";
    result["engine_id"] = "fixture-engine";
    const engine::Json operation{
        {"engine_operation_id", "engop:fixture:engine.inspect"},
        {"operation_name", "inspect"}, {"availability", availability},
        {"feature_ids", feature_ids},
        {"administrative_interactions", engine::Json::array({"inspect"})},
        {"administration_disposition", disposition},
        {"input_protocol", "fixture.engine.inspect-input.v1"},
        {"output_protocol", "fixture.engine.inspect-result.v1"},
        {"mutability", "read_only"}, {"idempotency", "idempotent"},
        {"expected_state_required", false}, {"authorization_requirement", "none"},
        {"recovery_operation_id", nullptr}, {"direct_invocation", "supported"},
        {"thermal_path", "freezing"},
    };
    result["operations"] = engine::Json::array({operation});
    refresh_self_digest(result, "descriptor_digest");
    return result;
}

engine::Json administration_payload(const engine::Json& snapshot,
                                    engine::Json profile,
                                    engine::Json expected_registry,
                                    const std::string& observed_state,
                                    engine::Json observed_registry,
                                    engine::Json descriptors,
                                    const engine::Json& requested_feature_id) {
    return engine::Json{
        {"protocol", "symphony.knowledge.administration-coverage-input.v1"},
        {"format_version", 1}, {"semantic_snapshot", snapshot},
        {"profile", std::move(profile)},
        {"expected_command_registry", std::move(expected_registry)},
        {"observed_qxctl_state", observed_state},
        {"observed_command_registry", std::move(observed_registry)},
        {"engine_descriptors", std::move(descriptors)},
        {"requested_feature_id", requested_feature_id},
    };
}

engine::Json proposal_payload(const engine::Json& state, const std::string& type,
                              const engine::Json& record,
                              const engine::Json& expected_feature_digest,
                              const std::vector<std::string>& target_paths,
                              const std::string& feature_file_path) {
    return engine::Json{
        {"repository", engine::Json{
            {"repository_id", "fixture"},
            {"revision", engine::Json{{"scheme", "git"}, {"value", "fixture-revision"}}},
            {"worktree_id", "fixture-worktree"},
            {"tree_digest", "sha256:1111111111111111111111111111111111111111111111111111111111111111"},
        }},
        {"session_ref", nullptr},
        {"context_ref", "fixture-context"},
        {"created_at", "2026-07-29T12:00:00Z"},
        {"expires_at", "2026-07-29T13:00:00Z"},
        {"operation", engine::Json{
            {"type", type},
            {"expected_contract_digest", state.at("contract_snapshot").at("digest")},
            {"expected_namespace_digest", state.at("namespace_registry").at("digest")},
            {"expected_registry_digest", state.at("feature_registry").at("digest")},
            {"expected_feature_digest", expected_feature_digest},
            {"affected_feature_ids", engine::Json::array({record.at("feature_id")})},
            {"target_paths", target_paths},
            {"namespace_entry", nullptr},
            {"feature_file", feature_file_path},
            {"feature_record", record},
            {"registry_notes", "Architect-ratified feature proposal."},
            {"semantic_declaration", engine::Json{
                {"feature_worthiness_ratified", true},
                {"owner_ratified", true},
                {"rationale", "Fixture declaration supplied by the authorized caller."},
                {"evidence", engine::Json::array({"fixture-ratification"})},
            }},
            {"authorization_ref", "fixture-authorization"},
        }},
    };
}

engine::Json check_fixture_records(const std::vector<engine::Json>& records) {
    TemporaryDirectory temporary;
    create_fixture(temporary.path());
    write_file(temporary.path() / "modules/example/FEATURES.md", feature_file(records));
    write_registry(temporary.path(), records);
    CurrentDirectory current(temporary.path());
    return ssfv::handle_request(request("check", disabled_check()));
}

void test_descriptor_and_actual_repository(const fs::path& repository_root) {
    const auto descriptor = ssfv::descriptor();
    require(engine::sha256_hex(descriptor.dump()) ==
                "e3ae6f7cc29402c9df275c0bb7a34553c1358ea5667f68c15bf35d493f8910ab",
            "engine-descriptor.v1 compatibility changed");
    require(descriptor.at("protocol") == engine::descriptor_protocol_v1 &&
                descriptor.at("operations").size() == 6U &&
                !descriptor.at("operations").at(0).contains("engine_operation_id"),
            "legacy descriptor negotiation surface changed");
    require(descriptor.at("language") == "C++26", "language contract mismatch");
    require(descriptor.at("thermal_path") == "freezing", "thermal-path contract mismatch");
    require(descriptor.at("install_state") == "installed_undocked", "install state mismatch");
    require(descriptor.at("canonical_apply_enabled") == false, "apply enabled");
    require(descriptor.at("session_mutation_enabled") == false, "session mutation enabled");
    require(descriptor.at("network_listener") == false, "network listener enabled");
    const auto descriptor_v2 = ssfv::descriptor_v2();
    require(descriptor_v2.at("protocol") == engine::descriptor_protocol_v2 &&
                descriptor_v2.at("format_version") == 2 &&
                !descriptor_v2.contains("install_state") &&
                descriptor_v2.at("operations").size() == 7U,
            "engine-descriptor.v2 negotiation surface mismatch");
    const auto descriptor_digest = descriptor_v2.at("descriptor_digest");
    auto descriptor_v2_preimage = descriptor_v2;
    descriptor_v2_preimage.erase("descriptor_digest");
    require(descriptor_digest == engine::tagged_sha256(descriptor_v2_preimage.dump()),
            "engine-descriptor.v2 digest mismatch");
    require(std::any_of(descriptor_v2.at("operations").begin(),
                        descriptor_v2.at("operations").end(),
                        [](const engine::Json& operation) {
        return operation.at("engine_operation_id") ==
                   "engop:symphony:ssfv.administration-check" &&
               operation.at("operation_name") == "administration-check";
    }), "administration-check stable operation identity is absent");

    CurrentDirectory current(repository_root);
    const auto inspect = ssfv::handle_request(request("inspect", engine::Json::object()));
    require(inspect.at("engine_decides_feature_worthiness") == false,
            "engine claimed feature-worthiness authority");
    const auto check = ssfv::handle_request(request("check", disabled_check()));
    require(check.at("summary").at("state") == "valid", "canonical partial registry invalid");
    require(check.at("coverage_state") == "partial",
            "canonical bootstrap must not imply repository-wide completeness");
    require(check.at("feature_count") == 70U && check.at("feature_file_count") == 15U,
            "canonical partial-catalog record counts mismatch");
    const auto graph = ssfv::handle_request(
        request("graph", engine::Json{{"format", "json"}}));
    require(graph.at("node_count") == 70U && graph.at("edge_count") == 283U,
            "canonical partial-catalog graph count mismatch");
    require(graph.at("noncanonical") == true && graph.at("rebuildable") == true,
            "graph authority escalated");

    const auto read_json = [&](const fs::path& relative_path) {
        std::ifstream input(repository_root / relative_path, std::ios::binary);
        require(input.good(), "canonical administration input is unreadable");
        return engine::parse_bounded_json(
            engine::read_bounded(input, engine::Limits::max_response_bytes),
            engine::Limits::max_response_bytes);
    };
    const auto canonical_administration = ssfv::handle_request(request(
        "administration-check", administration_payload(
            check.at("semantic_snapshot"),
            read_json("knowledge/FEATURE-ADMINISTRATION-PROFILE.json"),
            read_json("tools/qxctl/COMMANDS.json"), "not_evaluated", nullptr,
            engine::Json::array({descriptor_v2}),
            "ssfv:symphony:ssfv-engine.administration-assurance")));
    require(canonical_administration.at("feature_findings").empty() &&
                canonical_administration.at("summary").at("satisfied") == 1U &&
                canonical_administration.at("summary").at("unresolved") == 0U &&
                canonical_administration.at("module_integrations").at(0)
                    .at("integration_state") == "integration_ready" &&
                canonical_administration.at("module_integrations").at(0)
                    .at("docking_ready") == true,
            "ratified SSFV administration mapping is not integration ready");

    const auto full_administration = ssfv::handle_request(request(
        "administration-check", administration_payload(
            check.at("semantic_snapshot"),
            read_json("knowledge/FEATURE-ADMINISTRATION-PROFILE.json"),
            read_json("tools/qxctl/COMMANDS.json"), "not_evaluated", nullptr,
            engine::Json::array({descriptor_v2}), nullptr)));
    require(full_administration.at("summary").at("features_checked") == 70U &&
                full_administration.at("summary").at("surfaces_checked") == 131U &&
                full_administration.at("summary").at("satisfied") == 109U &&
                full_administration.at("summary").at("uncovered") == 0U &&
                full_administration.at("summary").at("exempt") == 13U &&
                full_administration.at("summary").at("prohibited") == 9U &&
                full_administration.at("summary").at("stale") == 0U &&
                full_administration.at("summary").at("unresolved") == 0U,
            "canonical administration baseline counts mismatch");
    require(std::none_of(full_administration.at("surfaces").begin(),
                         full_administration.at("surfaces").end(),
                         [](const engine::Json& surface) {
                             return surface.at("design_state") == "uncovered";
                         }),
            "canonical administration profile retained an uncovered surface");
}

void test_administration_coverage_and_module_admission() {
    TemporaryDirectory temporary;
    create_fixture(temporary.path());
    CurrentDirectory current(temporary.path());
    const auto snapshot = ssfv::handle_request(
        request("check", disabled_check())).at("semantic_snapshot");
    const std::string feature_id = "ssfv:symphony:fixture-feature";
    const std::string command_id = "qxcmd:fixture:engine.inspect";
    const std::string operation_id = "engop:fixture:engine.inspect";
    const auto profile = administration_profile(
        snapshot, feature_id, command_id, operation_id);
    const auto expected = command_registry("expected", engine::Json::array({
        administration_command(command_id, feature_id, "inspect", operation_id),
    }));
    const auto descriptor = fixture_engine_descriptor(
        engine::Json::array({feature_id}), "qxctl_required");

    const auto absent_payload = administration_payload(
        snapshot, profile, expected, "absent", nullptr,
        engine::Json::array({descriptor}), feature_id);
    const auto first = ssfv::handle_request(
        request("administration-check", absent_payload));
    const auto second = ssfv::handle_request(
        request("administration-check", absent_payload));
    require(first == second, "administration coverage result is nondeterministic");
    require(first.at("surfaces").at(0).at("design_state") == "satisfied" &&
                first.at("surfaces").at(0).at("live_state") == "qxctl_absent" &&
                first.at("surfaces").at(0).at("authorization_state") == "not_evaluated" &&
                first.at("module_integrations").at(0).at("integration_state") ==
                    "integration_ready",
            "qxctl absence was confused with a design or integration gap");
    require(first.at("module_integrations").at(0).at("docking_ready") == true &&
                first.at("read_only") == true &&
                first.at("canonical") == false,
            "integrated module readiness or read-only boundary mismatch");
    auto without_digest = first;
    const auto result_digest = without_digest.at("result_digest");
    without_digest.erase("result_digest");
    require(result_digest == engine::tagged_sha256(without_digest.dump()),
            "administration coverage result digest mismatch");

    {
        TemporaryDirectory empty;
        CurrentDirectory engine_only(empty.path());
        const auto engine_only_first = ssfv::handle_request(
            request("administration-check", absent_payload));
        const auto engine_only_second = ssfv::handle_request(
            request("administration-check", absent_payload));
        require(engine_only_first == first && engine_only_second == first,
                "administration-check depends on repository or qxctl files");
    }

    const auto observed = command_registry("observed", engine::Json::array({
        administration_command(command_id, feature_id, "inspect", operation_id),
    }));
    const auto ready = ssfv::handle_request(request("administration-check",
        administration_payload(snapshot, profile, expected, "present", observed,
            engine::Json::array({descriptor}), feature_id)));
    require(ready.at("surfaces").at(0).at("live_state") == "ready",
            "matching observed qxctl registry was not ready");

    const auto incompatible = ssfv::handle_request(request("administration-check",
        administration_payload(snapshot, profile, expected, "present",
            command_registry("observed", engine::Json::array()),
            engine::Json::array({descriptor}), feature_id)));
    require(incompatible.at("surfaces").at(0).at("design_state") == "satisfied" &&
                incompatible.at("surfaces").at(0).at("live_state") == "incompatible",
            "observed client incompatibility changed design coverage");
    auto semantically_incompatible_observed = observed;
    semantically_incompatible_observed["commands"].at(0)["backend_operation_ids"] =
        engine::Json::array();
    refresh_self_digest(semantically_incompatible_observed, "registry_digest");
    const auto semantic_incompatibility = ssfv::handle_request(request("administration-check",
        administration_payload(snapshot, profile, expected, "present",
            semantically_incompatible_observed, engine::Json::array({descriptor}), feature_id)));
    require(semantic_incompatibility.at("surfaces").at(0).at("live_state") ==
                "incompatible" &&
                !semantic_incompatibility.at("surfaces").at(0).at("findings").empty(),
            "observed stable ID masked incompatible semantic command evidence");

    const auto expected_without_route = command_registry(
        "expected", engine::Json::array());
    const auto uncovered = ssfv::handle_request(request("administration-check",
        administration_payload(snapshot, profile, expected_without_route, "absent", nullptr,
            engine::Json::array({descriptor}), feature_id)));
    require(uncovered.at("surfaces").at(0).at("design_state") == "uncovered" &&
                uncovered.at("module_integrations").at(0).at("integration_state") ==
                    "administration_unintegrated" &&
                uncovered.at("module_integrations").at(0).at("docking_ready") == false,
            "missing required qxctl route did not fail integration/docking readiness");
    require(!uncovered.at("remediation_constraints").empty() &&
                uncovered.at("remediation_constraints").at(0).at("proposed_command_id").is_null() &&
                uncovered.at("remediation_constraints").at(0).at("proposed_grammar").is_null(),
            "remediation constraints invented command identity or grammar");

    auto expected_without_backend = expected;
    expected_without_backend["commands"].at(0)["backend_operation_ids"] =
        engine::Json::array();
    refresh_self_digest(expected_without_backend, "registry_digest");
    const auto uncovered_backend = ssfv::handle_request(request("administration-check",
        administration_payload(snapshot, profile, expected_without_backend, "absent", nullptr,
            engine::Json::array({descriptor}), feature_id)));
    require(uncovered_backend.at("surfaces").at(0).at("design_state") == "uncovered" &&
                uncovered_backend.at("module_integrations").at(0).at("docking_ready") == false,
            "command identity without an exact backend binding satisfied coverage");

    const std::string second_command_id = "qxcmd:fixture:engine.query";
    const std::string second_operation_id = "engop:fixture:engine.query";
    auto composed_profile = profile;
    auto& composed_expectation = composed_profile["features"].at(1)
        ["expectations"].at(0);
    composed_expectation["delivery"] = "composed";
    composed_expectation["command_ids"] =
        engine::Json::array({command_id, second_command_id});
    composed_expectation["engine_operation_ids"] =
        engine::Json::array({operation_id, second_operation_id});
    refresh_self_digest(composed_profile, "profile_digest");
    const auto composed_expected = command_registry("expected", engine::Json::array({
        administration_command(command_id, feature_id, "inspect", operation_id),
        administration_command(second_command_id, feature_id, "inspect", second_operation_id),
    }));
    const auto composed = ssfv::handle_request(request("administration-check",
        administration_payload(snapshot, composed_profile, composed_expected,
            "not_evaluated", nullptr, engine::Json::array(), feature_id)));
    require(composed.at("surfaces").at(0).at("design_state") == "satisfied" &&
                composed.at("surfaces").at(0).at("findings").empty(),
            "composed commands did not satisfy their distinct backend-operation union");

    auto retired_expected = expected;
    retired_expected["commands"].at(0)["status"] = "retired";
    retired_expected["commands"].at(0)["deprecated_in"] = "0.2.0";
    retired_expected["commands"].at(0)["grammar"] = nullptr;
    refresh_self_digest(retired_expected, "registry_digest");
    const auto retired_route = ssfv::handle_request(request("administration-check",
        administration_payload(snapshot, profile, retired_expected, "absent", nullptr,
            engine::Json::array({descriptor}), feature_id)));
    require(retired_route.at("surfaces").at(0).at("design_state") == "uncovered" &&
                retired_route.at("module_integrations").at(0).at("docking_ready") == false,
            "retired expected command satisfied design or module coverage");

    auto prohibited_expected = expected;
    prohibited_expected["commands"].at(0)["mutability"] = "prohibited";
    refresh_self_digest(prohibited_expected, "registry_digest");
    const auto prohibited_route = ssfv::handle_request(request("administration-check",
        administration_payload(snapshot, profile, prohibited_expected, "absent", nullptr,
            engine::Json::array({descriptor}), feature_id)));
    require(prohibited_route.at("surfaces").at(0).at("design_state") == "uncovered" &&
                prohibited_route.at("module_integrations").at(0).at("docking_ready") == false,
            "prohibited expected command satisfied design or module coverage");

    require_error([&] {
        static_cast<void>(ssfv::handle_request(request("administration-check",
            administration_payload(snapshot, profile, nullptr, "absent", nullptr,
                engine::Json::array(), feature_id))));
    }, "administration.input.expected_registry");

    const auto missing_semantics = ssfv::handle_request(request("administration-check",
        administration_payload(snapshot, profile, expected, "not_evaluated", nullptr,
            engine::Json::array({fixture_engine_descriptor(
                engine::Json::array(), "unreviewed")}), feature_id)));
    require(missing_semantics.at("module_integrations").at(0).at("integration_state") ==
                "semantic_registration_required" &&
                missing_semantics.at("module_integrations").at(0).at("docking_ready") == false,
            "undeclared engine semantics did not block integration readiness");

    const auto unreviewed = ssfv::handle_request(request("administration-check",
        administration_payload(snapshot, profile, expected, "not_evaluated", nullptr,
            engine::Json::array({fixture_engine_descriptor(
                engine::Json::array({feature_id}), "unreviewed")}), feature_id)));
    require(unreviewed.at("module_integrations").at(0).at("integration_state") ==
                "administration_unintegrated",
            "missing administration declaration was normalized to not-applicable");

    auto unreviewed_profile = profile;
    auto& unreviewed_expectation = unreviewed_profile["features"].at(1)
        ["expectations"].at(0);
    unreviewed_expectation["delivery"] = "unreviewed";
    unreviewed_expectation["command_ids"] = engine::Json::array();
    unreviewed_expectation["engine_operation_ids"] = engine::Json::array();
    refresh_self_digest(unreviewed_profile, "profile_digest");
    const auto known_unreviewed = ssfv::handle_request(request("administration-check",
        administration_payload(snapshot, unreviewed_profile, expected, "not_evaluated", nullptr,
            engine::Json::array(), feature_id)));
    require(known_unreviewed.at("surfaces").at(0).at("design_state") == "unresolved" &&
                known_unreviewed.at("summary").at("unresolved") == 1U,
            "known unreviewed interaction did not remain explicit unresolved debt");

    auto invalid_unreviewed_profile = unreviewed_profile;
    invalid_unreviewed_profile["features"].at(1)["expectations"].at(0)
        ["command_ids"] = engine::Json::array({command_id});
    refresh_self_digest(invalid_unreviewed_profile, "profile_digest");
    require_error([&] {
        static_cast<void>(ssfv::handle_request(request("administration-check",
            administration_payload(snapshot, invalid_unreviewed_profile, expected,
                "not_evaluated", nullptr, engine::Json::array(), feature_id))));
    }, "administration.profile.disposition");

    const std::string child_feature_id = "ssfv:symphony:fixture-subfeature";
    auto inherited_profile = profile;
    auto inherited_expectation = inherited_profile["features"].at(1)
        ["expectations"].at(0);
    inherited_expectation["delivery"] = "delegated";
    inherited_expectation["command_ids"] = engine::Json::array();
    inherited_expectation["engine_operation_ids"] = engine::Json::array();
    inherited_expectation["inherited_from_feature_id"] = feature_id;
    inherited_profile["features"].at(3)["expectations"] =
        engine::Json::array({inherited_expectation});
    refresh_self_digest(inherited_profile, "profile_digest");
    auto inherited_command = administration_command(
        command_id, feature_id, "inspect", operation_id);
    inherited_command["feature_bindings"].push_back(engine::Json{
        {"feature_id", child_feature_id}, {"interaction", "inspect"}});
    const auto inherited_expected = command_registry(
        "expected", engine::Json::array({inherited_command}));
    const auto inherited_descriptor = fixture_engine_descriptor(
        engine::Json::array({feature_id, child_feature_id}), "qxctl_required");
    const auto inherited_result = ssfv::handle_request(request("administration-check",
        administration_payload(snapshot, inherited_profile, inherited_expected,
            "not_evaluated", nullptr, engine::Json::array({inherited_descriptor}),
            child_feature_id)));
    require(inherited_result.at("surfaces").at(0).at("design_state") == "satisfied" &&
                inherited_result.at("surfaces").at(0).at("command_ids").at(0) == command_id,
            "finite compatible administration inheritance did not resolve delivery");

    auto nullable_protocol_descriptor = descriptor;
    nullable_protocol_descriptor["language"] = "Rust-1.90";
    nullable_protocol_descriptor["thermal_path"] = "warm";
    nullable_protocol_descriptor["operations"].at(0)["input_protocol"] = nullptr;
    nullable_protocol_descriptor["operations"].at(0)["output_protocol"] = nullptr;
    refresh_self_digest(nullable_protocol_descriptor, "descriptor_digest");
    const auto nullable_protocol = ssfv::handle_request(request("administration-check",
        administration_payload(snapshot, profile, expected, "not_evaluated", nullptr,
            engine::Json::array({nullable_protocol_descriptor}), feature_id)));
    require(nullable_protocol.at("module_integrations").at(0).at("integration_state") ==
                "integration_ready",
            "schema-valid nullable protocols, language, or warm path were rejected");

    auto pair_descriptor = descriptor;
    pair_descriptor["operations"].at(0)["administrative_interactions"] =
        engine::Json::array({"inspect", "query"});
    refresh_self_digest(pair_descriptor, "descriptor_digest");
    const auto missing_pair = ssfv::handle_request(request("administration-check",
        administration_payload(snapshot, profile, expected, "not_evaluated", nullptr,
            engine::Json::array({pair_descriptor}), feature_id)));
    require(missing_pair.at("module_integrations").at(0).at("integration_state") ==
                "administration_unintegrated",
            "one mapped interaction masked another uncovered engine-operation surface");

    auto second_descriptor = descriptor;
    second_descriptor["module_id"] = "fixture-second-engine";
    second_descriptor["engine_id"] = "fixture-second-engine";
    second_descriptor["operations"].at(0)["engine_operation_id"] =
        "engop:fixture:second.inspect";
    second_descriptor["operations"].at(0)["administration_disposition"] = "not_applicable";
    refresh_self_digest(second_descriptor, "descriptor_digest");
    const auto ordered = ssfv::handle_request(request("administration-check",
        administration_payload(snapshot, profile, expected, "not_evaluated", nullptr,
            engine::Json::array({descriptor, second_descriptor}), feature_id)));
    const auto reversed = ssfv::handle_request(request("administration-check",
        administration_payload(snapshot, profile, expected, "not_evaluated", nullptr,
            engine::Json::array({second_descriptor, descriptor}), feature_id)));
    require(ordered == reversed, "engine descriptor input order changed coverage evidence");
    require_error([&] {
        static_cast<void>(ssfv::handle_request(request("administration-check",
            administration_payload(snapshot, profile, expected, "not_evaluated", nullptr,
                engine::Json::array({descriptor, descriptor}), feature_id))));
    }, "administration.engine_descriptor.duplicate_digest");

    auto duplicate_operation_descriptor = descriptor;
    auto duplicate_operation = duplicate_operation_descriptor["operations"].at(0);
    duplicate_operation["operation_name"] = "inspect-again";
    duplicate_operation_descriptor["operations"].push_back(duplicate_operation);
    refresh_self_digest(duplicate_operation_descriptor, "descriptor_digest");
    const auto duplicate_operation_result = ssfv::handle_request(request("administration-check",
        administration_payload(snapshot, profile, expected, "not_evaluated", nullptr,
            engine::Json::array({duplicate_operation_descriptor}), feature_id)));
    require(duplicate_operation_result.at("module_integrations").at(0)
                .at("integration_state") == "descriptor_invalid",
            "duplicated engine operation identity was admitted");

    auto missing_recovery_descriptor = descriptor;
    missing_recovery_descriptor["operations"].at(0)["recovery_operation_id"] =
        "engop:fixture:engine.recover";
    refresh_self_digest(missing_recovery_descriptor, "descriptor_digest");
    const auto missing_recovery_result = ssfv::handle_request(request("administration-check",
        administration_payload(snapshot, profile, expected, "not_evaluated", nullptr,
            engine::Json::array({missing_recovery_descriptor}), feature_id)));
    require(missing_recovery_result.at("module_integrations").at(0)
                .at("integration_state") == "descriptor_invalid",
            "missing engine recovery identity was admitted");

    auto invalid_language_descriptor = descriptor;
    invalid_language_descriptor["language"] = "Rust:1.90";
    refresh_self_digest(invalid_language_descriptor, "descriptor_digest");
    const auto invalid_language_result = ssfv::handle_request(request("administration-check",
        administration_payload(snapshot, profile, expected, "not_evaluated", nullptr,
            engine::Json::array({invalid_language_descriptor}), feature_id)));
    require(invalid_language_result.at("module_integrations").at(0)
                .at("integration_state") == "descriptor_invalid",
            "schema-invalid descriptor language was admitted");

    auto omitted_profile = profile;
    omitted_profile["features"].erase(omitted_profile["features"].begin());
    refresh_self_digest(omitted_profile, "profile_digest");
    require_error([&] {
        static_cast<void>(ssfv::handle_request(request("administration-check",
            administration_payload(snapshot, omitted_profile, expected, "absent", nullptr,
                engine::Json::array(), feature_id))));
    }, "administration.profile.catalog_set");
    auto enforced_profile = profile;
    enforced_profile["forward_gate"] = "enforce_all_records";
    refresh_self_digest(enforced_profile, "profile_digest");
    require_error([&] {
        static_cast<void>(ssfv::handle_request(request("administration-check",
            administration_payload(snapshot, enforced_profile, expected, "absent", nullptr,
                engine::Json::array(), feature_id))));
    }, "administration.profile.forward_gate");

    auto invalid_expected = expected;
    invalid_expected["client_id"] = "not-qxctl";
    refresh_self_digest(invalid_expected, "registry_digest");
    require_error([&] {
        static_cast<void>(ssfv::handle_request(request("administration-check",
            administration_payload(snapshot, profile, invalid_expected, "absent", nullptr,
                engine::Json::array(), feature_id))));
    }, "administration.command_registry.identity");
    invalid_expected = expected;
    invalid_expected["commands"].at(0)["mutability"] = "invented";
    refresh_self_digest(invalid_expected, "registry_digest");
    require_error([&] {
        static_cast<void>(ssfv::handle_request(request("administration-check",
            administration_payload(snapshot, profile, invalid_expected, "absent", nullptr,
                engine::Json::array(), feature_id))));
    }, "administration.command.classification");

    auto stale_profile = profile;
    stale_profile["ssfv_registry_digest"] =
        "sha256:2222222222222222222222222222222222222222222222222222222222222222";
    refresh_self_digest(stale_profile, "profile_digest");
    const auto stale = ssfv::handle_request(request("administration-check",
        administration_payload(snapshot, stale_profile, expected, "absent", nullptr,
            engine::Json::array(), feature_id)));
    require(stale.at("surfaces").at(0).at("design_state") == "stale",
            "profile-to-SSFV digest drift was not reported as stale");

    auto invalid_descriptor = descriptor;
    invalid_descriptor["descriptor_digest"] =
        "sha256:3333333333333333333333333333333333333333333333333333333333333333";
    const auto invalid = ssfv::handle_request(request("administration-check",
        administration_payload(snapshot, profile, expected, "absent", nullptr,
            engine::Json::array({invalid_descriptor}), feature_id)));
    require(invalid.at("module_integrations").at(0).at("integration_state") ==
                "descriptor_invalid" &&
                invalid.at("module_integrations").at(0).at("docking_ready") == false,
            "invalid descriptor was admitted");

    auto expired = request("administration-check", absent_payload);
    expired.deadline_unix_ms = engine::unix_time_ms() - 1;
    require_error([&] { static_cast<void>(ssfv::handle_request(expired)); },
                  "request.deadline_expired");
}

void test_valid_hierarchy_and_deterministic_graph() {
    TemporaryDirectory temporary;
    create_fixture(temporary.path());
    CurrentDirectory current(temporary.path());
    const auto check = ssfv::handle_request(request("check", disabled_check()));
    require(check.at("summary").at("state") == "valid", "valid hierarchy rejected");
    require(check.at("coverage_state") == "partial",
            "populated catalog must not imply repository-wide completeness");
    require(check.at("feature_count") == 4U && check.at("feature_file_count") == 1U,
            "feature counts mismatch");
    const auto first = ssfv::handle_request(
        request("graph", engine::Json{{"format", "json"}}));
    const auto second = ssfv::handle_request(
        request("graph", engine::Json{{"format", "json"}}));
    require(first == second, "graph projection is nondeterministic");
    require(first.at("node_count") == 4U && first.at("edge_count") == 3U,
            "graph topology mismatch");
}

void test_root_scope_and_crosslinks() {
    TemporaryDirectory temporary;
    create_fixture(temporary.path(), false);
    write_file(temporary.path() / "SPEC.md", "root feature owner\n");
    write_file(temporary.path() / "root-feature.cpp", "int root_feature() { return 1; }\n");

    auto capability = feature_record(
        "ssfv:symphony:root-capability", "Root Capability", "capability", nullptr);
    auto feature = feature_record(
        "ssfv:symphony:root-feature", "Root Feature", "feature",
        "ssfv:symphony:root-capability");
    for (auto* record : {&capability, &feature}) {
        (*record)["owner_contract"] = "SPEC.md";
        (*record)["source_scope"] = ".";
        (*record)["implementation_paths"] = engine::Json::array({"root-feature.cpp"});
    }
    capability["relationships"] = engine::Json::array({
        engine::Json{{"type", "enables"},
                     {"target_feature_id", "ssfv:symphony:root-feature"},
                     {"rationale", "The capability enables its explicit feature."}},
    });
    feature["distinctions"] = engine::Json::array({
        engine::Json{{"target_feature_id", "ssfv:symphony:root-capability"},
                     {"distinction", "The feature is an implementation facet, not the root capability."}},
    });
    const std::vector<engine::Json> records = {capability, feature};
    write_file(temporary.path() / "FEATURES.md", feature_file(records, "SPEC.md", "."));
    write_registry(temporary.path(), records, "FEATURES.md");
    write_file(temporary.path() / "knowledge/skvi/INDEX.md",
               "# Index\n\n- path: `FEATURES.md`\n");

    CurrentDirectory current(temporary.path());
    const auto check = ssfv::handle_request(request("check", disabled_check()));
    require(check.at("summary").at("state") == "valid", "valid root scope rejected");
    require(check.at("feature_count") == 2U && check.at("feature_file_count") == 1U,
            "root-scope counts mismatch");
    const auto graph = ssfv::handle_request(
        request("graph", engine::Json{{"format", "json"}}));
    require(graph.at("edge_count") == 3U,
            "primary parent, crosslink, and distinction were not projected");
}

void test_freshness_and_diff() {
    TemporaryDirectory temporary;
    create_fixture(temporary.path());
    CurrentDirectory current(temporary.path());
    const auto baseline_check = ssfv::handle_request(request("check", disabled_check()));
    const auto baseline = baseline_check.at("semantic_snapshot");
    write_file(temporary.path() / "modules/example/src/example.cpp",
               "int fixture() { return 2; }\n");

    const auto report = ssfv::handle_request(request("check", engine::Json{
        {"expected_namespace_digest", nullptr}, {"expected_registry_digest", nullptr},
        {"freshness", "report"}, {"baseline", baseline},
    }));
    require(report.at("summary").at("state") == "valid", "report mode became a hard failure");
    require(report.at("semantic_freshness_state") == "stale", "stale evidence not detected");
    require(report.at("summary").at("warning").get<std::size_t>() > 0U,
            "report mode omitted warning");

    const auto required = ssfv::handle_request(request("check", engine::Json{
        {"expected_namespace_digest", nullptr}, {"expected_registry_digest", nullptr},
        {"freshness", "require"}, {"baseline", baseline},
    }));
    require(required.at("summary").at("state") == "invalid",
            "require mode did not fail unresolved semantic freshness");

    const auto diff = ssfv::handle_request(request("diff", engine::Json{
        {"baseline", baseline},
        {"expected_current_snapshot_digest", nullptr},
        {"scope_feature_ids", engine::Json::array()},
        {"include_semantic_candidates", true},
    }));
    require(diff.at("state") == "review_required", "stale source was not review-required");
    require(diff.at("stale_references").size() == 1U, "stale path count mismatch");
    require(!diff.at("semantic_candidates").empty(), "semantic candidate omitted");
}

void test_snapshot_diff_variants_and_bounds() {
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        CurrentDirectory current(temporary.path());
        const auto first = ssfv::handle_request(request("check", disabled_check()));
        const auto second = ssfv::handle_request(request("check", disabled_check()));
        require(first.at("semantic_snapshot") == second.at("semantic_snapshot"),
                "unchanged inputs produced different semantic snapshots");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        CurrentDirectory current(temporary.path());
        const auto baseline = ssfv::handle_request(
            request("check", disabled_check())).at("semantic_snapshot");
        auto records = fixture_records();
        records.at(1)["title"] = "Updated Fixture Feature";
        write_file(temporary.path() / "modules/example/FEATURES.md", feature_file(records));
        write_registry(temporary.path(), records);
        const auto diff = ssfv::handle_request(request("diff", engine::Json{
            {"baseline", baseline},
            {"expected_current_snapshot_digest", nullptr},
            {"scope_feature_ids", engine::Json::array()},
            {"include_semantic_candidates", true},
        }));
        require(diff.at("changed_feature_ids") ==
                    engine::Json::array({"ssfv:symphony:fixture-feature"}),
                "record-only change was not isolated");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        auto records = fixture_records();
        records.at(0)["cross_vector_references"] = engine::Json::array({
            engine::Json{{"vector", "skvi"}, {"applicability", "applicable"},
                         {"reference", "knowledge/skvi/INDEX.md"},
                         {"reason", "The feature owner file is indexed here."}},
        });
        write_file(temporary.path() / "modules/example/FEATURES.md", feature_file(records));
        write_registry(temporary.path(), records);
        CurrentDirectory current(temporary.path());
        const auto baseline = ssfv::handle_request(
            request("check", disabled_check())).at("semantic_snapshot");
        write_file(temporary.path() / "knowledge/skvi/INDEX.md",
                   "# Index\n\n- path: `modules/example/FEATURES.md`\n\nChanged fixture note.\n");
        const auto diff = ssfv::handle_request(request("diff", engine::Json{
            {"baseline", baseline},
            {"expected_current_snapshot_digest", nullptr},
            {"scope_feature_ids", engine::Json::array()},
            {"include_semantic_candidates", true},
        }));
        require(std::find(diff.at("stale_references").begin(),
                          diff.at("stale_references").end(),
                          engine::Json("knowledge/skvi/INDEX.md")) !=
                    diff.at("stale_references").end(),
                "cross-vector evidence change was not stale");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        CurrentDirectory current(temporary.path());
        const auto baseline = ssfv::handle_request(
            request("check", disabled_check())).at("semantic_snapshot");
        write_file(temporary.path() / "modules/example/SPEC.md",
                   "fixture owner contract changed\n");
        const auto diff = ssfv::handle_request(request("diff", engine::Json{
            {"baseline", baseline},
            {"expected_current_snapshot_digest", nullptr},
            {"scope_feature_ids", engine::Json::array()},
            {"include_semantic_candidates", true},
        }));
        require(diff.at("state") == "review_required" &&
                    !diff.at("semantic_candidates").empty(),
                "owner-contract-only change was not review-required");
        const auto required = ssfv::handle_request(request("check", engine::Json{
            {"expected_namespace_digest", nullptr},
            {"expected_registry_digest", nullptr},
            {"freshness", "require"},
            {"baseline", baseline},
        }));
        require(required.at("summary").at("state") == "invalid" &&
                    has_finding(required, "ssfv.semantic_freshness.review_required"),
                "required freshness accepted changed owner evidence");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        auto initial = fixture_records();
        initial.at(0)["implementation_paths"] =
            engine::Json::array({"modules/example/src/example.cpp",
                                 "modules/example/src/second.cpp"});
        write_file(temporary.path() / "modules/example/src/second.cpp",
                   "int second_fixture() { return 2; }\n");
        write_file(temporary.path() / "modules/example/FEATURES.md", feature_file(initial));
        write_registry(temporary.path(), initial);
        CurrentDirectory current(temporary.path());
        const auto baseline = ssfv::handle_request(
            request("check", disabled_check())).at("semantic_snapshot");
        auto current_records = initial;
        current_records.at(0)["implementation_paths"] =
            engine::Json::array({"modules/example/src/example.cpp"});
        write_file(temporary.path() / "modules/example/FEATURES.md",
                   feature_file(current_records));
        write_registry(temporary.path(), current_records);
        const auto diff = ssfv::handle_request(request("diff", engine::Json{
            {"baseline", baseline},
            {"expected_current_snapshot_digest", nullptr},
            {"scope_feature_ids", engine::Json::array()},
            {"include_semantic_candidates", false},
        }));
        require(diff.at("uncovered_paths") ==
                    engine::Json::array({"modules/example/src/second.cpp"}),
                "removed implementation coverage was not reported");
        require(diff.at("state") == "review_required",
                "removed implementation coverage did not require review");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        CurrentDirectory current(temporary.path());
        const auto state = ssfv::handle_request(request("check", disabled_check()));
        auto records = fixture_records();
        records.at(0)["title"] = "Scoped Capability Change";
        records.at(1)["title"] = "Out-of-Scope Feature Change";
        write_file(temporary.path() / "modules/example/FEATURES.md", feature_file(records));
        write_registry(temporary.path(), records);
        const auto diff = ssfv::handle_request(request("diff", engine::Json{
            {"baseline", state.at("semantic_snapshot")},
            {"expected_current_snapshot_digest", nullptr},
            {"scope_feature_ids", engine::Json::array({
                "ssfv:symphony:fixture-capability",
            })},
            {"include_semantic_candidates", true},
        }));
        require(diff.at("changed_feature_ids") ==
                    engine::Json::array({"ssfv:symphony:fixture-capability"}),
                "bounded feature scope leaked another change");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        CurrentDirectory current(temporary.path());
        const auto state = ssfv::handle_request(request("check", disabled_check()));
        require_error([&] {
            static_cast<void>(ssfv::handle_request(request("diff", engine::Json{
                {"baseline", state.at("semantic_snapshot")},
                {"expected_current_snapshot_digest",
                 "sha256:0000000000000000000000000000000000000000000000000000000000000000"},
                {"scope_feature_ids", engine::Json::array()},
                {"include_semantic_candidates", false},
            })));
        }, "ssfv.diff.current_digest_mismatch");
        const auto stale_check = ssfv::handle_request(request("check", engine::Json{
            {"expected_namespace_digest", nullptr},
            {"expected_registry_digest",
             "sha256:0000000000000000000000000000000000000000000000000000000000000000"},
            {"freshness", "disabled"},
            {"baseline", nullptr},
        }));
        require(stale_check.at("summary").at("state") == "invalid" &&
                    has_finding(stale_check, "ssfv.registry.expected_digest_mismatch"),
                "stale expected registry digest accepted");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        CurrentDirectory current(temporary.path());
        auto baseline = ssfv::handle_request(
            request("check", disabled_check())).at("semantic_snapshot");
        baseline["contract_digest"] =
            "sha256:0000000000000000000000000000000000000000000000000000000000000000";
        refresh_snapshot_digest(baseline);
        const auto diff = ssfv::handle_request(request("diff", engine::Json{
            {"baseline", baseline},
            {"expected_current_snapshot_digest", nullptr},
            {"scope_feature_ids", engine::Json::array()},
            {"include_semantic_candidates", true},
        }));
        require(diff.at("state") == "review_required" &&
                    !diff.at("semantic_candidates").empty(),
                "baseline contract mismatch omitted review requirement");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        CurrentDirectory current(temporary.path());
        auto baseline = ssfv::handle_request(
            request("check", disabled_check())).at("semantic_snapshot");
        baseline["records"] = engine::Json::array();
        for (std::size_t index = 0U; index < 8193U; ++index) {
            baseline["records"].push_back(engine::Json::object());
        }
        refresh_snapshot_digest(baseline);
        require_error([&] {
            static_cast<void>(ssfv::handle_request(request("diff", engine::Json{
                {"baseline", baseline},
                {"expected_current_snapshot_digest", nullptr},
                {"scope_feature_ids", engine::Json::array()},
                {"include_semantic_candidates", false},
            })));
        }, "ssfv.snapshot.count");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        CurrentDirectory current(temporary.path());
        auto baseline = ssfv::handle_request(
            request("check", disabled_check())).at("semantic_snapshot");
        baseline["records"].at(1)["parent_feature_id"] = nullptr;
        refresh_snapshot_digest(baseline);
        require_error([&] {
            static_cast<void>(ssfv::handle_request(request("diff", engine::Json{
                {"baseline", baseline},
                {"expected_current_snapshot_digest", nullptr},
                {"scope_feature_ids", engine::Json::array()},
                {"include_semantic_candidates", false},
            })));
        }, "ssfv.snapshot.record");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        CurrentDirectory current(temporary.path());
        auto baseline = ssfv::handle_request(
            request("check", disabled_check())).at("semantic_snapshot");
        baseline["records"].at(1)["parent_feature_id"] =
            "ssfv:symphony:missing-capability";
        refresh_snapshot_digest(baseline);
        require_error([&] {
            static_cast<void>(ssfv::handle_request(request("diff", engine::Json{
                {"baseline", baseline},
                {"expected_current_snapshot_digest", nullptr},
                {"scope_feature_ids", engine::Json::array()},
                {"include_semantic_candidates", false},
            })));
        }, "ssfv.snapshot.hierarchy");

        baseline = ssfv::handle_request(
            request("check", disabled_check())).at("semantic_snapshot");
        baseline["feature_files"] = engine::Json::array();
        refresh_snapshot_digest(baseline);
        require_error([&] {
            static_cast<void>(ssfv::handle_request(request("diff", engine::Json{
                {"baseline", baseline},
                {"expected_current_snapshot_digest", nullptr},
                {"scope_feature_ids", engine::Json::array()},
                {"include_semantic_candidates", false},
            })));
        }, "ssfv.snapshot.record");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        CurrentDirectory current(temporary.path());
        auto expired = request("check", disabled_check());
        expired.deadline_unix_ms = engine::unix_time_ms() - 1;
        require_error([&] {
            static_cast<void>(ssfv::handle_request(expired));
        }, "request.deadline_expired");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        write_file(temporary.path() / contract_paths.front(),
                   std::string((4U << 20) + 1U, 'x'));
        CurrentDirectory current(temporary.path());
        require_error([&] {
            static_cast<void>(ssfv::handle_request(request("check", disabled_check())));
        }, "path.file_too_large");
    }
}

void test_structural_failures_and_no_follow() {
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        auto contents = feature_file({
            feature_record("ssfv:symphony:fixture-capability", "Fixture Capability",
                           "capability", nullptr),
        });
        contents += "<!-- symphony:ssfv:feature-file:v1:begin -->\n";
        write_file(temporary.path() / "modules/example/FEATURES.md", contents);
        CurrentDirectory current(temporary.path());
        const auto check = ssfv::handle_request(request("check", disabled_check()));
        require(check.at("summary").at("state") == "invalid",
                "duplicate managed marker accepted");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        fs::remove(temporary.path() / "modules/example/FEATURES.md");
        write_file(temporary.path() / "outside.md", "unsafe target\n");
        fs::create_symlink(temporary.path() / "outside.md",
                           temporary.path() / "modules/example/FEATURES.md");
        CurrentDirectory current(temporary.path());
        const auto check = ssfv::handle_request(request("check", disabled_check()));
        require(check.at("summary").at("state") == "invalid",
                "symlinked feature owner accepted");
    }
}

void test_record_graph_and_evidence_failures() {
    {
        auto records = fixture_records();
        records.at(3)["parent_feature_id"] = "ssfv:symphony:fixture-capability";
        const auto check = check_fixture_records(records);
        require(check.at("summary").at("state") == "invalid" &&
                    has_finding(check, "ssfv.hierarchy.parent_kind"),
                "invalid primary-parent kind accepted");
    }
    {
        auto records = fixture_records();
        records.at(1)["parent_feature_id"] = "ssfv:symphony:missing-capability";
        const auto check = check_fixture_records(records);
        require(check.at("summary").at("state") == "invalid" &&
                    has_finding(check, "ssfv.hierarchy.parent_missing"),
                "missing primary parent accepted");
    }
    {
        auto records = fixture_records();
        records.at(1)["parent_feature_id"] = "ssfv:symphony:fixture-microfeature";
        const auto check = check_fixture_records(records);
        require(check.at("summary").at("state") == "invalid" &&
                    has_finding(check, "ssfv.hierarchy.cycle"),
                "primary-parent cycle accepted");
    }
    {
        auto records = fixture_records();
        records.at(0)["relationships"] = engine::Json::array({
            engine::Json{{"type", "depends_on"},
                         {"target_feature_id", "ssfv:symphony:missing-target"},
                         {"rationale", "Fixture missing-target relation."}},
        });
        const auto check = check_fixture_records(records);
        require(check.at("summary").at("state") == "invalid" &&
                    has_finding(check, "ssfv.relationship.target_missing"),
                "missing relationship target accepted");
    }
    {
        auto records = fixture_records();
        const engine::Json relation{
            {"type", "depends_on"},
            {"target_feature_id", "ssfv:symphony:fixture-feature"},
            {"rationale", "Fixture repeated relation."},
        };
        records.at(0)["relationships"] = engine::Json::array({relation, relation});
        const auto check = check_fixture_records(records);
        require(check.at("summary").at("state") == "invalid" &&
                    has_finding(check, "ssfv.record.duplicate_value"),
                "repeated relationship accepted");
    }
    {
        auto records = fixture_records();
        records.at(0)["distinctions"] = engine::Json::array({
            engine::Json{{"target_feature_id", "ssfv:symphony:fixture-feature"},
                         {"distinction", ""}},
        });
        const auto check = check_fixture_records(records);
        require(check.at("summary").at("state") == "invalid",
                "distinction without factual text accepted");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        auto registry = registry_entry(fixture_records().at(0));
        const auto digest_at = registry.find("sha256:");
        require(digest_at != std::string::npos, "fixture registry digest absent");
        registry.replace(digest_at, 71U,
                         "sha256:0000000000000000000000000000000000000000000000000000000000000000");
        for (std::size_t index = 1U; index < fixture_records().size(); ++index) {
            registry += "\n" + registry_entry(fixture_records().at(index));
        }
        write_file(temporary.path() / "knowledge/ssfv/REGISTRY.md",
            "# Registry\n\n## Canonical Entries\n\n" + registry +
            "\n## Prohibited Entries\n\nNone.\n");
        CurrentDirectory current(temporary.path());
        const auto check = ssfv::handle_request(request("check", disabled_check()));
        require(check.at("summary").at("state") == "invalid" &&
                    has_finding(check, "ssfv.registry.record_mismatch"),
                "registry-to-record digest mismatch accepted");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        fs::remove(temporary.path() / "modules/example/src/example.cpp");
        CurrentDirectory current(temporary.path());
        const auto check = ssfv::handle_request(request("check", disabled_check()));
        require(check.at("summary").at("state") == "invalid" &&
                    has_finding(check, "ssfv.record.evidence_unreadable"),
                "missing implementation evidence accepted");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        fs::remove(temporary.path() / "modules/example/src/example.cpp");
        write_file(temporary.path() / "outside.cpp", "int outside() { return 0; }\n");
        fs::create_symlink(temporary.path() / "outside.cpp",
                           temporary.path() / "modules/example/src/example.cpp");
        CurrentDirectory current(temporary.path());
        const auto check = ssfv::handle_request(request("check", disabled_check()));
        require(check.at("summary").at("state") == "invalid",
                "symlinked implementation evidence accepted");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        fs::remove(temporary.path() / "modules/example/SPEC.md");
        write_file(temporary.path() / "outside-spec.md", "outside owner\n");
        fs::create_symlink(temporary.path() / "outside-spec.md",
                           temporary.path() / "modules/example/SPEC.md");
        CurrentDirectory current(temporary.path());
        const auto check = ssfv::handle_request(request("check", disabled_check()));
        require(check.at("summary").at("state") == "invalid",
                "symlinked owner contract accepted");
    }
    {
        auto records = fixture_records();
        for (auto& record : records) {
            record["implementation_paths"] = engine::Json::array({"modules/example/src"});
        }
        const auto check = check_fixture_records(records);
        require(check.at("summary").at("state") == "invalid",
                "non-regular implementation evidence accepted");
    }
    {
        auto records = fixture_records();
        records.at(0)["cross_vector_references"] = engine::Json::array({
            engine::Json{{"vector", "skvi"}, {"applicability", "applicable"},
                         {"reference", "knowledge/skvi/MISSING.md"},
                         {"reason", "Fixture applicable reference."}},
        });
        const auto check = check_fixture_records(records);
        require(check.at("summary").at("state") == "invalid",
                "missing applicable cross-vector reference accepted");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        write_file(temporary.path() / "modules/example/FEATURES.md",
            "# Features\n\n<!-- symphony:ssfv:feature-file:v1:begin -->\n"
            "```json\n{\"protocol\":\n```\n"
            "<!-- symphony:ssfv:feature-file:v1:end -->\n");
        CurrentDirectory current(temporary.path());
        const auto check = ssfv::handle_request(request("check", disabled_check()));
        require(check.at("summary").at("state") == "invalid" &&
                    has_finding(check, "ssfv.feature_file.json"),
                "malformed embedded JSON accepted");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        auto contents = feature_file(fixture_records());
        const auto object = contents.find("{\n");
        require(object != std::string::npos, "fixture JSON object absent");
        contents.insert(object + 2U,
                        "  \"protocol\": \"symphony.ssfv.feature-file.v1\",\n");
        write_file(temporary.path() / "modules/example/FEATURES.md", contents);
        CurrentDirectory current(temporary.path());
        const auto check = ssfv::handle_request(request("check", disabled_check()));
        require(check.at("summary").at("state") == "invalid" &&
                    has_finding(check, "ssfv.feature_file.json"),
                "duplicate embedded JSON field accepted");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        auto contents = feature_file(fixture_records());
        const auto title = contents.find("Fixture Capability");
        require(title != std::string::npos, "fixture title absent");
        contents.insert(title, 1U, static_cast<char>(0xff));
        write_file(temporary.path() / "modules/example/FEATURES.md", contents);
        CurrentDirectory current(temporary.path());
        const auto check = ssfv::handle_request(request("check", disabled_check()));
        require(check.at("summary").at("state") == "invalid" &&
                    has_finding(check, "ssfv.feature_file.encoding"),
                "invalid UTF-8 in embedded JSON accepted");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        auto contents = feature_file(fixture_records());
        const std::string opening =
            "<!-- symphony:ssfv:feature-file:v1:begin -->\n```json\n";
        const std::string closing =
            "\n```\n<!-- symphony:ssfv:feature-file:v1:end -->";
        const auto begin = contents.find(opening);
        const auto end = contents.find(closing);
        require(begin != std::string::npos && end != std::string::npos,
                "fixture managed region absent");
        const auto json_begin = begin + opening.size();
        const auto parsed = engine::Json::parse(
            contents.substr(json_begin, end - json_begin));
        contents.replace(json_begin, end - json_begin, parsed.dump());
        write_file(temporary.path() / "modules/example/FEATURES.md", contents);
        CurrentDirectory current(temporary.path());
        const auto check = ssfv::handle_request(request("check", disabled_check()));
        require(check.at("summary").at("state") == "invalid" &&
                    has_finding(check, "ssfv.feature_file.canonical_format"),
                "noncanonical feature-file rendering accepted");
    }
    {
        auto records = fixture_records();
        records.at(0)["implementation_paths"] = engine::Json::array({"../outside.cpp"});
        const auto check = check_fixture_records(records);
        require(check.at("summary").at("state") == "invalid",
                "traversal implementation path accepted");
    }
    {
        auto records = fixture_records();
        records.at(0)["implementation_paths"] =
            engine::Json::array({"modules/example/src/*.cpp"});
        const auto check = check_fixture_records(records);
        require(check.at("summary").at("state") == "invalid",
                "glob implementation path accepted");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        auto registry = registry_entry(fixture_records().at(0));
        const std::string feature_line = "- feature_file: `modules/example/FEATURES.md`\n";
        const std::string owner_line = "- owner_contract: `modules/example/SPEC.md`\n";
        const auto feature_at = registry.find(feature_line);
        const auto owner_at = registry.find(owner_line);
        require(feature_at != std::string::npos && owner_at != std::string::npos,
                "fixture registry fields absent");
        registry.replace(feature_at, feature_line.size() + owner_line.size(),
                         owner_line + feature_line);
        write_file(temporary.path() / "knowledge/ssfv/REGISTRY.md",
                   "# Registry\n\n## Canonical Entries\n\n" + registry +
                   "\n## Prohibited Entries\n\nNone.\n");
        CurrentDirectory current(temporary.path());
        const auto check = ssfv::handle_request(request("check", disabled_check()));
        require(check.at("summary").at("state") == "invalid",
                "reordered or incomplete registry block accepted");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        std::string registry =
            "# Registry\n\n## Canonical Entries\n\n" +
            registry_entry(fixture_records().at(0)) +
            "\n## Prohibited Entries\n\nNone.\n";
        const auto notes = registry.find("deterministic fixture");
        require(notes != std::string::npos, "fixture registry notes absent");
        registry.insert(notes, 1U, static_cast<char>(0xff));
        write_file(temporary.path() / "knowledge/ssfv/REGISTRY.md", registry);
        CurrentDirectory current(temporary.path());
        const auto check = ssfv::handle_request(request("check", disabled_check()));
        require(check.at("summary").at("state") == "invalid",
                "invalid UTF-8 in registry accepted");
    }
}

void test_add_proposal_and_reserved_apply() {
    TemporaryDirectory temporary;
    create_fixture(temporary.path(), false);
    CurrentDirectory current(temporary.path());
    const auto state = ssfv::handle_request(request("check", disabled_check()));
    const auto record = feature_record(
        "ssfv:symphony:new-capability", "New Capability", "capability", nullptr);
    const auto payload = proposal_payload(
        state, "add_feature", record, nullptr,
        {"knowledge/ssfv/REGISTRY.md", "knowledge/skvi/INDEX.md",
         "modules/example/FEATURES.md"},
        "modules/example/FEATURES.md");
    const auto first = ssfv::handle_request(request("propose", payload));
    const auto second = ssfv::handle_request(request("propose", payload));
    require(first == second, "proposal output is nondeterministic");
    require(first.at("canonical_apply_enabled") == false, "proposal enabled apply");
    require(first.at("authority").at("engine_decided_domain_truth") == false,
            "proposal escalated engine authority");
    require(first.at("write_set").size() == 3U, "proposal write set mismatch");
    require(fs::exists(temporary.path() / "modules/example/FEATURES.md") == false,
            "proposal mutated canonical state");

    require_error([&] {
        static_cast<void>(ssfv::handle_request(request("apply", engine::Json::object())));
    }, "operation.unsupported");
}

void test_proposal_operation_matrix() {
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path(), false);
        write_file(temporary.path() / "knowledge/ssfv/new-namespace.md",
                   "namespace evidence\n");
        CurrentDirectory current(temporary.path());
        const auto state = ssfv::handle_request(request("check", disabled_check()));
        auto payload = proposal_payload(
            state, "add_feature",
            feature_record("ssfv:symphony:unused", "Unused", "capability", nullptr),
            nullptr, {"knowledge/ssfv/NAMESPACES.md"}, "modules/example/FEATURES.md");
        auto& operation = payload["operation"];
        operation["type"] = "allocate_namespace";
        operation["affected_feature_ids"] = engine::Json::array();
        operation["target_paths"] = engine::Json::array({"knowledge/ssfv/NAMESPACES.md"});
        operation["expected_feature_digest"] = nullptr;
        operation["namespace_entry"] = engine::Json{
            {"namespace", "enterprise"},
            {"id_prefix", "ssfv:enterprise:"},
            {"owner_contract", "knowledge/ssfv/SPEC.md"},
            {"scope", "Enterprise extension feature identities."},
            {"status", "canonical"},
            {"evidence", engine::Json::array({"knowledge/ssfv/new-namespace.md"})},
            {"notes", "Architect-ratified fixture allocation."},
        };
        operation["feature_file"] = nullptr;
        operation["feature_record"] = nullptr;
        operation["registry_notes"] = nullptr;
        const auto result = ssfv::handle_request(request("propose", payload));
        require(result.at("write_set").size() == 1U &&
                    result.at("operations").at(0).at("type") == "allocate_namespace",
                "namespace allocation proposal shape mismatch");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        CurrentDirectory current(temporary.path());
        const auto state = ssfv::handle_request(request("check", disabled_check()));
        const auto record = feature_record(
            "ssfv:symphony:fixture-second-feature", "Second Fixture Feature", "feature",
            "ssfv:symphony:fixture-capability");
        const auto payload = proposal_payload(
            state, "add_feature", record, nullptr,
            {"knowledge/ssfv/REGISTRY.md", "modules/example/FEATURES.md"},
            "modules/example/FEATURES.md");
        const auto result = ssfv::handle_request(request("propose", payload));
        require(result.at("write_set").size() == 2U,
                "add to existing owner file unexpectedly changed SKVI");
    }
    for (const auto& [type, status] : std::vector<std::pair<std::string, std::string>>{
             {"update_feature", "implemented"},
             {"deprecate_feature", "deprecated"},
             {"retire_feature", "retired"},
         }) {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        CurrentDirectory current(temporary.path());
        const auto state = ssfv::handle_request(request("check", disabled_check()));
        auto record = fixture_records().at(1);
        record["title"] = "Proposed " + type;
        record["status"] = status;
        const auto prior_digest = engine::tagged_sha256(fixture_records().at(1).dump());
        const auto payload = proposal_payload(
            state, type, record, prior_digest,
            {"knowledge/ssfv/REGISTRY.md", "modules/example/FEATURES.md"},
            "modules/example/FEATURES.md");
        const auto result = ssfv::handle_request(request("propose", payload));
        require(result.at("write_set").size() == 2U,
                type + " proposal write-set mismatch");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        write_file(temporary.path() / "modules/other/SPEC.md", "other owner\n");
        write_file(temporary.path() / "modules/other/src/feature.cpp",
                   "int moved_feature() { return 3; }\n");
        CurrentDirectory current(temporary.path());
        const auto state = ssfv::handle_request(request("check", disabled_check()));
        auto record = fixture_records().at(1);
        record["owner_contract"] = "modules/other/SPEC.md";
        record["source_scope"] = "modules/other";
        record["implementation_paths"] =
            engine::Json::array({"modules/other/src/feature.cpp"});
        const auto payload = proposal_payload(
            state, "move_feature", record,
            engine::tagged_sha256(fixture_records().at(1).dump()),
            {"knowledge/ssfv/REGISTRY.md", "knowledge/skvi/INDEX.md",
             "modules/example/FEATURES.md", "modules/other/FEATURES.md"},
            "modules/other/FEATURES.md");
        const auto result = ssfv::handle_request(request("propose", payload));
        require(result.at("write_set").size() == 4U,
                "move proposal did not atomically bind old/new owners, registry, and SKVI");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path(), false);
        write_file(temporary.path() / "modules/example/FEATURES.md", "");
        CurrentDirectory current(temporary.path());
        const auto state = ssfv::handle_request(request("check", disabled_check()));
        const auto record = feature_record(
            "ssfv:symphony:new-capability", "New Capability", "capability", nullptr);
        const auto payload = proposal_payload(
            state, "add_feature", record, nullptr,
            {"knowledge/ssfv/REGISTRY.md", "knowledge/skvi/INDEX.md",
             "modules/example/FEATURES.md"},
            "modules/example/FEATURES.md");
        require_error([&] {
            static_cast<void>(ssfv::handle_request(request("propose", payload)));
        }, "proposal.target_exists_unregistered");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        CurrentDirectory current(temporary.path());
        const auto state = ssfv::handle_request(request("check", disabled_check()));
        auto record = fixture_records().at(1);
        record["title"] = "Stale Proposal";
        auto payload = proposal_payload(
            state, "update_feature", record,
            "sha256:0000000000000000000000000000000000000000000000000000000000000000",
            {"knowledge/ssfv/REGISTRY.md", "modules/example/FEATURES.md"},
            "modules/example/FEATURES.md");
        require_error([&] {
            static_cast<void>(ssfv::handle_request(request("propose", payload)));
        }, "proposal.expected_feature_mismatch");
        payload = proposal_payload(
            state, "update_feature", record,
            engine::tagged_sha256(fixture_records().at(1).dump()),
            {"README.md"}, "modules/example/FEATURES.md");
        require_error([&] {
            static_cast<void>(ssfv::handle_request(request("propose", payload)));
        }, "proposal.target_set_mismatch");
        payload = proposal_payload(
            state, "update_feature", record,
            engine::tagged_sha256(fixture_records().at(1).dump()),
            {"knowledge/ssfv/REGISTRY.md", "modules/example/FEATURES.md"},
            "modules/example/FEATURES.md");
        payload["operation"]["semantic_declaration"]["evidence"] = engine::Json::array();
        require_error([&] {
            static_cast<void>(ssfv::handle_request(request("propose", payload)));
        }, "payload.semantic_declaration");
        payload = proposal_payload(
            state, "update_feature", record,
            engine::tagged_sha256(fixture_records().at(1).dump()),
            {"knowledge/ssfv/REGISTRY.md", "modules/example/FEATURES.md"},
            "modules/example/FEATURES.md");
        payload["operation"]["feature_record"]["implementation_paths"] =
            engine::Json::array({"../unsafe.cpp"});
        require_error([&] {
            static_cast<void>(ssfv::handle_request(request("propose", payload)));
        }, "ssfv.record.implementation_path");
    }
}

void measure_empty_and_populated_operations(const fs::path& repository_root) {
    const auto measure = [](const std::function<void()>& operation) {
        const auto started = std::chrono::steady_clock::now();
        operation();
        return std::chrono::duration_cast<std::chrono::microseconds>(
            std::chrono::steady_clock::now() - started).count();
    };
    std::int64_t empty_check_us = 0;
    std::int64_t empty_graph_us = 0;
    {
        CurrentDirectory current(repository_root);
        empty_check_us = measure([] {
            static_cast<void>(ssfv::handle_request(request("check", disabled_check())));
        });
        empty_graph_us = measure([] {
            static_cast<void>(ssfv::handle_request(
                request("graph", engine::Json{{"format", "json"}})));
        });
    }

    TemporaryDirectory temporary;
    create_fixture(temporary.path(), false);
    std::vector<engine::Json> records;
    records.push_back(feature_record(
        "ssfv:symphony:performance-capability", "Performance Capability",
        "capability", nullptr));
    for (std::size_t index = 0U; index < 256U; ++index) {
        std::ostringstream identifier;
        identifier << "ssfv:symphony:performance-feature-"
                   << std::setw(3) << std::setfill('0') << index;
        records.push_back(feature_record(
            identifier.str(), "Bounded Performance Feature", "feature",
            "ssfv:symphony:performance-capability"));
    }
    write_file(temporary.path() / "modules/example/FEATURES.md", feature_file(records));
    write_registry(temporary.path(), records);
    write_file(temporary.path() / "knowledge/skvi/INDEX.md",
               "# Index\n\n- path: `modules/example/FEATURES.md`\n");
    std::int64_t populated_check_us = 0;
    std::int64_t populated_graph_us = 0;
    {
        CurrentDirectory current(temporary.path());
        populated_check_us = measure([&] {
            const auto result = ssfv::handle_request(request("check", disabled_check()));
            require(result.at("feature_count") == 257U,
                    "bounded performance fixture count mismatch");
        });
        populated_graph_us = measure([&] {
            const auto result = ssfv::handle_request(
                request("graph", engine::Json{{"format", "json"}}));
            require(result.at("node_count") == 257U && result.at("edge_count") == 256U,
                    "bounded performance graph count mismatch");
            static_cast<void>(engine::serialize_response(engine::success_response(
                request("graph", engine::Json{{"format", "json"}}),
                ssfv::engine_id, ssfv::engine_version, result)));
        });
    }
    std::cout << "SSFV performance microseconds"
              << " empty_check=" << empty_check_us
              << " empty_graph=" << empty_graph_us
              << " populated_257_check=" << populated_check_us
              << " populated_257_graph_and_serialize=" << populated_graph_us << '\n';
}

}

int main(int argc, char** argv) {
    try {
        if (argc != 2) {
            throw std::runtime_error("repository root argument is required");
        }
        test_descriptor_and_actual_repository(fs::canonical(argv[1]));
        test_administration_coverage_and_module_admission();
        test_valid_hierarchy_and_deterministic_graph();
        test_root_scope_and_crosslinks();
        test_freshness_and_diff();
        test_snapshot_diff_variants_and_bounds();
        test_structural_failures_and_no_follow();
        test_record_graph_and_evidence_failures();
        test_add_proposal_and_reserved_apply();
        test_proposal_operation_matrix();
        measure_empty_and_populated_operations(fs::canonical(argv[1]));
        std::cout << "SSFV engine tests passed\n";
        return 0;
    } catch (const std::exception& error) {
        std::cerr << "SSFV engine tests failed: " << error.what() << '\n';
        return 1;
    }
}
