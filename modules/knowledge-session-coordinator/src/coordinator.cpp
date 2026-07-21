#include "coordinator.hpp"

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
        {"readiness", "read_only_foundation"},
        {"canonical_apply_enabled", false},
        {"session_mutation_enabled", false},
        {"maestro_docking_enabled", false},
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
        {"session_mutation_enabled", false},
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
            engine::Json{{"name", "begin"}, {"availability", "reserved"}, {"mutates_canonical", false}},
            engine::Json{{"name", "status"}, {"availability", "reserved"}, {"mutates_canonical", false}},
            engine::Json{{"name", "checkpoint"}, {"availability", "reserved"}, {"mutates_canonical", false}},
            engine::Json{{"name", "close"}, {"availability", "reserved"}, {"mutates_canonical", false}},
            engine::Json{{"name", "recover"}, {"availability", "reserved"}, {"mutates_canonical", false}},
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
        {"session_mutation_enabled", false},
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
    throw engine::Error("operation.unsupported", "operation is reserved or unsupported", 4);
}

}
