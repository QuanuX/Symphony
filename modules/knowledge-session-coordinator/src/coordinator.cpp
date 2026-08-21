#include "coordinator.hpp"
#include "authority_session.hpp"
#include "lifecycle.hpp"
#include "lifecycle_journal.hpp"
#include "named_versions.hpp"
#include "reconciliation.hpp"
#include "ssfv_maintenance.hpp"

#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/limits.hpp"
#include "symphony/knowledge/engine/path.hpp"

#include <filesystem>
#include <algorithm>
#include <set>
#include <string>
#include <vector>

namespace symphony::knowledge::session {
namespace engine = symphony::knowledge::engine;

namespace {

void require_exact_fields(const engine::Json& object, const std::set<std::string>& fields) {
    if (!object.is_object() || object.size() != fields.size()) {
        throw engine::Error(
            "payload.field_set", "operation payload is incomplete or contains unknown fields", 4);
    }
    for (const auto& [key, value] : object.items()) {
        static_cast<void>(value);
        if (!fields.contains(key)) {
            throw engine::Error("payload.unknown_field", "operation payload contains an unknown field", 4);
        }
    }
}

engine::Json inspect(const engine::Json& payload) {
    require_exact_fields(payload, {});
    return engine::Json{
        {"descriptor", descriptor()},
        {"readiness", "authenticated_session_foundation"},
        {"reconciliation", reconciliation_capabilities()},
        {"authenticated_session", authority_session_capabilities()},
        {"lifecycle", lifecycle_capabilities()},
        {"lifecycle_journal", lifecycle_journal_capabilities()},
        {"ssfv_maintenance", ssfv_maintenance_capabilities()},
        {"named_versions", named_version_capabilities()},
        {"canonical_apply_enabled", false},
        {"session_mutation_enabled", true},
        {"maestro_docking_enabled", true},
    };
}

engine::Json check(const engine::Json& payload, std::int64_t deadline_unix_ms) {
    require_exact_fields(payload, {"expected_snapshot_digest", "paths"});
    const auto& paths_value = payload.at("paths");
    if (!paths_value.is_array() || paths_value.empty() ||
        paths_value.size() > engine::Limits::max_snapshot_files) {
        throw engine::Error("payload.invalid_paths", "paths must be a non-empty bounded array", 4);
    }
    std::vector<std::string> paths;
    paths.reserve(paths_value.size());
    for (const auto& value : paths_value) {
        if (!value.is_string()) {
            throw engine::Error("payload.invalid_paths", "every path must be a string", 4);
        }
        paths.push_back(value.get<std::string>());
    }

    const auto& expected = payload.at("expected_snapshot_digest");
    if (!expected.is_null() && !expected.is_string()) {
        throw engine::Error(
            "payload.invalid_expected_digest", "expected_snapshot_digest must be a string or null", 4);
    }
    engine::Json expected_matches = nullptr;
    std::string expected_text;
    if (expected.is_string()) {
        expected_text = expected.get<std::string>();
        const bool hex_suffix = expected_text.size() == 71U &&
            std::all_of(expected_text.begin() + 7, expected_text.end(), [](const unsigned char character) {
                return (character >= '0' && character <= '9') ||
                       (character >= 'a' && character <= 'f');
            });
        if (!expected_text.starts_with("sha256:") || !hex_suffix) {
            throw engine::Error(
                "payload.invalid_expected_digest", "expected snapshot digest has invalid syntax", 4);
        }
    }

    const auto snapshot = engine::snapshot_files(
        std::filesystem::current_path(), paths, deadline_unix_ms);
    engine::Json files = engine::Json::array();
    for (const auto& file : snapshot.files) {
        files.push_back(engine::Json{
            {"path", file.path},
            {"size", file.size},
            {"digest", file.digest},
        });
    }
    if (!expected_text.empty()) {
        expected_matches = expected_text == snapshot.digest;
    }

    return engine::Json{
        {"snapshot", engine::Json{{"digest", snapshot.digest}, {"files", std::move(files)}}},
        {"expected_snapshot_matches", expected_matches},
        {"read_only", true},
        {"canonical_apply_enabled", false},
        {"session_mutation_enabled", true},
    };
}

}

engine::Json descriptor() {
    return engine::Json{
        {"protocol", engine::descriptor_protocol_v1},
        {"module_id", module_id},
        {"engine_id", engine_id},
        {"vector_id", nullptr},
        {"engine_version", engine_version},
        {"process_protocols", engine::Json::array({engine::process_protocol_v1})},
        {"contract_versions", engine::Json::array({"knowledge/SPEC.md@v1"})},
        {"operations", engine::Json::array({
            engine::Json{{"name", "inspect"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "check"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "compatibility"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "begin"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "status"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "checkpoint"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "close"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "recover"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "session_begin"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "session_status"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "session_checkpoint"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "session_close"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "session_recover"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "ssfv_maintenance_begin"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "ssfv_maintenance_status"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "ssfv_maintenance_checkpoint"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "ssfv_maintenance_close"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "ssfv_maintenance_recover"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "named_version_prepare"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "named_version_seal"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "named_version_alias"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "named_version_lookup"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "named_version_status"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "named_version_recover"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "lifecycle_plan"}, {"availability", "implemented_report_only"}, {"mutates_canonical", false}},
            engine::Json{{"name", "lifecycle_boot"}, {"availability", "implemented_report_only_persistence"}, {"mutates_canonical", false}},
            engine::Json{{"name", "lifecycle_boot_status"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "lifecycle_boot_recover"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "lifecycle_apply_prepare"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "lifecycle_apply_finalize"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "lifecycle_apply_close"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "lifecycle_apply_status"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "lifecycle_apply_recover"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "apply"}, {"availability", "disabled"}, {"mutates_canonical", true}},
        })},
        {"limits", engine::Json{
            {"request_bytes", engine::Limits::max_request_bytes},
            {"response_bytes", engine::Limits::max_response_bytes},
            {"json_depth", engine::Limits::max_json_depth},
            {"json_values", engine::Limits::max_json_values},
            {"path_bytes", engine::Limits::max_path_bytes},
            {"snapshot_files", engine::Limits::max_snapshot_files},
            {"snapshot_file_bytes", engine::Limits::max_snapshot_file_bytes},
            {"deadline_ahead_ms", engine::Limits::max_deadline_ahead_ms},
        }},
        {"supported_scopes", engine::Json::array({"user"})},
        {"language", "C++26"},
        {"thermal_path", "freezing"},
        {"install_state", "installed_undocked"},
        {"default_receptor", nullptr},
        {"canonical_apply_enabled", false},
        {"session_mutation_enabled", true},
        {"network_listener", false},
    };
}

engine::Json handle_request(const engine::Request& request) {
    if (request.operation == "inspect") {
        return inspect(request.payload);
    }
    if (request.operation == "check") {
        return check(request.payload, request.deadline_unix_ms);
    }
    if (request.operation == "compatibility" || request.operation == "begin" ||
        request.operation == "status" || request.operation == "checkpoint" ||
        request.operation == "close" || request.operation == "recover") {
        return handle_reconciliation(request);
    }
    if (request.operation == "session_begin" || request.operation == "session_status" ||
        request.operation == "session_checkpoint" || request.operation == "session_close" ||
        request.operation == "session_recover") {
        return handle_authority_session(request);
    }
    if (request.operation == "ssfv_maintenance_begin" ||
        request.operation == "ssfv_maintenance_status" ||
        request.operation == "ssfv_maintenance_checkpoint" ||
        request.operation == "ssfv_maintenance_close" ||
        request.operation == "ssfv_maintenance_recover") {
        return handle_ssfv_maintenance(request);
    }
    if (request.operation == "named_version_prepare" || request.operation == "named_version_seal" ||
        request.operation == "named_version_alias" || request.operation == "named_version_lookup" ||
        request.operation == "named_version_status" || request.operation == "named_version_recover") {
        return handle_named_version(request);
    }
    if (request.operation == "lifecycle_plan") {
        return handle_lifecycle_plan(request);
    }
    if (request.operation == "lifecycle_boot" || request.operation == "lifecycle_boot_status" ||
        request.operation == "lifecycle_boot_recover") {
        return handle_lifecycle_journal(request);
    }
    if (request.operation == "lifecycle_apply_prepare" || request.operation == "lifecycle_apply_finalize" ||
        request.operation == "lifecycle_apply_status" || request.operation == "lifecycle_apply_recover" ||
        request.operation == "lifecycle_apply_close") {
        return handle_lifecycle_apply_request(request);
    }
    throw engine::Error("operation.unsupported", "operation is reserved or unsupported", 4);
}

}
