#include "skvi.hpp"

#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/limits.hpp"
#include "symphony/knowledge/engine/path.hpp"

#include <algorithm>
#include <array>
#include <chrono>
#include <cctype>
#include <filesystem>
#include <map>
#include <set>
#include <string>
#include <string_view>
#include <utility>
#include <vector>

namespace symphony::knowledge::skvi {
namespace engine = symphony::knowledge::engine;
namespace fs = std::filesystem;

namespace {

constexpr std::size_t max_entries = 512;
constexpr std::size_t max_evidence = 1024;
constexpr std::size_t max_field_bytes = 64U * 1024U;
constexpr const char* index_path = "knowledge/skvi/INDEX.md";
constexpr const char* check_protocol = "symphony.skvi.check-result.v1";
constexpr const char* projection_protocol = "symphony.skvi.projection.v1";
constexpr const char* proposal_protocol = "symphony.knowledge.proposal.v1";

constexpr std::array<const char*, 10> entry_fields = {
    "path", "title", "surface_type", "truth_role", "owner", "scope",
    "relationships", "consumers", "deferred_projections", "notes",
};

const std::vector<std::string> contract_paths = {
    "knowledge/skvi/INTENT.md",
    "knowledge/skvi/MANIFEST.md",
    "knowledge/skvi/SKILL.md",
    "knowledge/skvi/SPEC.md",
};

const std::vector<std::string> required_indexed_paths = {
    "README.md",
    "INTENT.md",
    "go.work",
    "knowledge/INTENT.md",
    "knowledge/MANIFEST.md",
    "knowledge/SKILL.md",
    "knowledge/SPEC.md",
    "knowledge/skvi/INDEX.md",
    "knowledge/skvi/INTENT.md",
    "knowledge/skvi/MANIFEST.md",
    "knowledge/skvi/SKILL.md",
    "knowledge/skvi/SPEC.md",
};

struct Entry final {
    std::map<std::string, std::string> fields;
    std::size_t line = 0;
};

struct Finding final {
    std::string severity;
    std::string code;
    std::string path;
    std::string detail;
};

struct IndexState final {
    std::string contents;
    engine::FileDigest index_file;
    engine::Snapshot contract_snapshot;
    std::vector<Entry> entries;
    std::map<std::string, engine::FileDigest> indexed_files;
    std::vector<Finding> findings;
    std::size_t passes = 0;
    std::size_t relationships_checked = 0;
};

std::string trim(std::string_view value) {
    std::size_t begin = 0;
    while (begin < value.size() &&
           (value[begin] == ' ' || value[begin] == '\t' || value[begin] == '\r' || value[begin] == '\n')) {
        ++begin;
    }
    std::size_t end = value.size();
    while (end > begin &&
           (value[end - 1U] == ' ' || value[end - 1U] == '\t' ||
            value[end - 1U] == '\r' || value[end - 1U] == '\n')) {
        --end;
    }
    return std::string(value.substr(begin, end - begin));
}

std::string trim_list_prefix(std::string_view line) {
    std::size_t begin = 0;
    while (begin < line.size() && (line[begin] == ' ' || line[begin] == '\t' || line[begin] == '-')) {
        ++begin;
    }
    return trim(line.substr(begin));
}

std::string clean_value(std::string value) {
    value = trim(value);
    if (value == "|") {
        return {};
    }
    if (value.size() >= 2U && value.front() == '`' && value.back() == '`') {
        value = value.substr(1U, value.size() - 2U);
    }
    return value;
}

bool printable_bounded(std::string_view value, std::size_t max_bytes, bool allow_empty = false) {
    if ((!allow_empty && value.empty()) || value.size() > max_bytes) {
        return false;
    }
    return std::all_of(value.begin(), value.end(), [](const unsigned char character) {
        return character >= 0x20U && character != 0x7fU;
    });
}

bool field_bounded(std::string_view value) {
    if (value.empty() || value.size() > max_field_bytes) {
        return false;
    }
    return std::all_of(value.begin(), value.end(), [](const unsigned char character) {
        return character == '\n' || (character >= 0x20U && character != 0x7fU);
    });
}

bool safe_token(std::string_view value, std::size_t max_bytes = engine::Limits::max_token_bytes) {
    if (value.empty() || value.size() > max_bytes) {
        return false;
    }
    return std::all_of(value.begin(), value.end(), [](const unsigned char character) {
        const bool alphanumeric =
            (character >= 'a' && character <= 'z') ||
            (character >= 'A' && character <= 'Z') ||
            (character >= '0' && character <= '9');
        return alphanumeric || character == '.' || character == '_' ||
               character == ':' || character == '-';
    });
}

bool tagged_digest(std::string_view value) {
    if (value.size() != 71U || !value.starts_with("sha256:")) {
        return false;
    }
    return std::all_of(value.begin() + 7, value.end(), [](const unsigned char character) {
        return (character >= '0' && character <= '9') ||
               (character >= 'a' && character <= 'f');
    });
}

bool strict_utc(std::string_view value) {
    if (value.size() != 20U || value[4] != '-' || value[7] != '-' || value[10] != 'T' ||
        value[13] != ':' || value[16] != ':' || value[19] != 'Z') {
        return false;
    }
    for (const std::size_t index : {0U, 1U, 2U, 3U, 5U, 6U, 8U, 9U, 11U, 12U, 14U, 15U, 17U, 18U}) {
        if (value[index] < '0' || value[index] > '9') {
            return false;
        }
    }
    const auto number = [&](std::size_t begin, std::size_t count) {
        int result = 0;
        for (std::size_t index = begin; index < begin + count; ++index) {
            result = result * 10 + (value[index] - '0');
        }
        return result;
    };
    const auto date = std::chrono::year{number(0, 4)} /
                      std::chrono::month{static_cast<unsigned int>(number(5, 2))} /
                      std::chrono::day{static_cast<unsigned int>(number(8, 2))};
    return date.ok() && number(11, 2) <= 23 && number(14, 2) <= 59 && number(17, 2) <= 59;
}

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

std::string require_string(
    const engine::Json& object,
    const char* field,
    std::size_t max_bytes,
    bool token = false) {
    const auto& value = object.at(field);
    if (!value.is_string()) {
        throw engine::Error("payload.invalid_field", std::string(field) + " must be a string", 4);
    }
    const auto text = value.get<std::string>();
    if ((token && !safe_token(text, max_bytes)) || (!token && !printable_bounded(text, max_bytes))) {
        throw engine::Error("payload.invalid_field", std::string(field) + " has invalid syntax", 4);
    }
    return text;
}

void add_finding(IndexState& state, Finding finding) {
    if (finding.severity == "pass") {
        ++state.passes;
        return;
    }
    if (state.findings.size() >= max_evidence) {
        throw engine::Error("skvi.evidence_limit", "SKVI evidence item limit exceeded", 5);
    }
    state.findings.push_back(std::move(finding));
}

std::string detected_field(const std::string& normalized) {
    for (const auto* field : entry_fields) {
        const std::string prefix = std::string(field) + ':';
        if (normalized.starts_with(prefix)) {
            return field;
        }
    }
    if (normalized.starts_with("status:")) {
        return "status";
    }
    return {};
}

std::vector<Entry> parse_entries(const std::string& contents, IndexState& state) {
    std::vector<Entry> entries;
    Entry current;
    bool in_entry = false;
    std::string active_field;
    std::size_t line_number = 0;
    std::size_t position = 0;

    auto finish_entry = [&]() {
        if (!in_entry) {
            return;
        }
        entries.push_back(std::move(current));
        current = Entry{};
        active_field.clear();
        in_entry = false;
    };

    while (position <= contents.size()) {
        const auto end = contents.find('\n', position);
        const auto line = contents.substr(
            position, end == std::string::npos ? std::string::npos : end - position);
        ++line_number;
        const auto normalized = trim_list_prefix(line);
        const auto field = detected_field(normalized);

        if (field == "path") {
            finish_entry();
            in_entry = true;
            current.line = line_number;
        }
        if (in_entry && !field.empty()) {
            if (current.fields.contains(field)) {
                add_finding(state, Finding{
                    "violation", "skvi.entry.duplicate_field",
                    current.fields.contains("path") ? current.fields.at("path") : "unavailable",
                    "line=" + std::to_string(line_number) + " field=" + field,
                });
            } else {
                const auto colon = normalized.find(':');
                current.fields[field] = clean_value(normalized.substr(colon + 1U));
            }
            active_field = field;
        } else if (in_entry && !active_field.empty()) {
            if (!normalized.empty() && !normalized.starts_with('#')) {
                auto value = clean_value(normalized);
                if (!value.empty()) {
                    auto& target = current.fields[active_field];
                    if (!target.empty()) {
                        target.push_back('\n');
                    }
                    target += value;
                }
            } else if (normalized.starts_with('#')) {
                active_field.clear();
            }
        }

        if (end == std::string::npos) {
            break;
        }
        position = end + 1U;
    }
    finish_entry();
    if (entries.empty()) {
        add_finding(state, Finding{"violation", "skvi.entry.none", index_path, "no entries detected"});
    }
    if (entries.size() > max_entries) {
        throw engine::Error("skvi.entry_limit", "SKVI entry-count limit exceeded", 5);
    }
    return entries;
}

engine::Json entry_json_without_digest(const Entry& entry) {
    engine::Json result = engine::Json::object();
    for (const auto* field : entry_fields) {
        result[field] = entry.fields.contains(field) ? entry.fields.at(field) : "";
    }
    result["status"] = entry.fields.contains("status") ? entry.fields.at("status") : "";
    return result;
}

engine::Json entry_json(const Entry& entry) {
    auto result = entry_json_without_digest(entry);
    result["entry_digest"] = engine::tagged_sha256(result.dump());
    return result;
}

std::vector<std::string> relationship_targets(const std::string& value) {
    std::vector<std::string> targets;
    std::size_t position = 0;
    while (position < value.size()) {
        const auto begin = value.find('`', position);
        if (begin == std::string::npos) {
            break;
        }
        const auto end = value.find('`', begin + 1U);
        if (end == std::string::npos) {
            break;
        }
        auto candidate = value.substr(begin + 1U, end - begin - 1U);
        const bool looks_like_path = candidate == "go.work" || candidate.contains('/') ||
                                     candidate.ends_with(".md") || candidate.ends_with(".json");
        if (looks_like_path && engine::is_safe_relative_path(candidate)) {
            targets.push_back(std::move(candidate));
        }
        position = end + 1U;
    }
    std::sort(targets.begin(), targets.end());
    targets.erase(std::unique(targets.begin(), targets.end()), targets.end());
    return targets;
}

IndexState analyze_index(const fs::path& root, std::int64_t deadline_unix_ms) {
    IndexState state;
    state.contents = engine::read_regular_file_no_follow(
        root, index_path, engine::Limits::max_snapshot_file_bytes, deadline_unix_ms);
    state.index_file = engine::FileDigest{
        index_path,
        static_cast<std::uint64_t>(state.contents.size()),
        engine::tagged_sha256(state.contents),
    };
    state.contract_snapshot = engine::snapshot_files(root, contract_paths, deadline_unix_ms);
    state.entries = parse_entries(state.contents, state);

    std::set<std::string> indexed_paths;
    for (const auto& entry : state.entries) {
        const auto path_it = entry.fields.find("path");
        const auto path = path_it == entry.fields.end() ? std::string{} : path_it->second;
        bool shape_valid = true;
        for (const auto* field : entry_fields) {
            const auto found = entry.fields.find(field);
            if (found == entry.fields.end() || !field_bounded(found->second)) {
                shape_valid = false;
                add_finding(state, Finding{
                    "violation", "skvi.entry.missing_or_invalid_field",
                    path.empty() ? "unavailable" : path,
                    "line=" + std::to_string(entry.line) + " field=" + field,
                });
            }
        }
        const auto status_it = entry.fields.find("status");
        if (status_it == entry.fields.end() || status_it->second != "canonical") {
            shape_valid = false;
            add_finding(state, Finding{
                "violation", "skvi.entry.invalid_status",
                path.empty() ? "unavailable" : path,
                "line=" + std::to_string(entry.line),
            });
        }
        if (shape_valid) {
            add_finding(state, Finding{"pass", "skvi.entry.shape", path, "required fields present"});
        }

        if (!engine::is_safe_relative_path(path)) {
            add_finding(state, Finding{"violation", "skvi.path.unsafe", path.empty() ? "unavailable" : path, "unsafe relative path"});
            continue;
        }
        add_finding(state, Finding{"pass", "skvi.path.safe", path, "safe relative path"});

        if (!indexed_paths.insert(path).second) {
            add_finding(state, Finding{"violation", "skvi.path.duplicate", path, "path appears more than once"});
            continue;
        }
        add_finding(state, Finding{"pass", "skvi.path.unique", path, "path is unique"});

        try {
            const auto contents = engine::read_regular_file_no_follow(
                root, path, engine::Limits::max_snapshot_file_bytes, deadline_unix_ms);
            const engine::FileDigest file{
                path,
                static_cast<std::uint64_t>(contents.size()),
                engine::tagged_sha256(contents),
            };
            state.indexed_files.emplace(path, file);
            add_finding(state, Finding{"pass", "skvi.path.regular_file", path, "no-follow regular file exists"});
        } catch (const engine::Error& error) {
            add_finding(state, Finding{"violation", "skvi.path.unreadable", path, error.code()});
        }
    }

    for (const auto& required : required_indexed_paths) {
        if (indexed_paths.contains(required)) {
            add_finding(state, Finding{"pass", "skvi.required.indexed", required, "required surface indexed"});
        } else {
            add_finding(state, Finding{"violation", "skvi.required.unindexed", required, "required surface missing from index"});
        }
    }

    for (const auto& entry : state.entries) {
        const auto path_it = entry.fields.find("path");
        const auto relationship_it = entry.fields.find("relationships");
        if (path_it == entry.fields.end() || relationship_it == entry.fields.end()) {
            continue;
        }
        for (const auto& target : relationship_targets(relationship_it->second)) {
            ++state.relationships_checked;
            if (indexed_paths.contains(target)) {
                add_finding(state, Finding{"pass", "skvi.relationship.indexed", path_it->second, "target=" + target});
            } else {
                add_finding(state, Finding{"warning", "skvi.relationship.unindexed", path_it->second, "target=" + target});
            }
        }
    }
    return state;
}

std::size_t finding_count(const IndexState& state, const std::string& severity) {
    if (severity == "pass") {
        return state.passes;
    }
    return static_cast<std::size_t>(std::count_if(
        state.findings.begin(), state.findings.end(), [&](const Finding& finding) {
            return finding.severity == severity;
        }));
}

engine::Json evidence_json(const IndexState& state) {
    auto evidence = engine::Json::array();
    for (const auto& finding : state.findings) {
        evidence.push_back(engine::Json{
            {"severity", finding.severity},
            {"code", finding.code},
            {"path", finding.path},
            {"detail", finding.detail},
        });
    }
    return evidence;
}

engine::Json file_json(const engine::FileDigest& file) {
    return engine::Json{{"path", file.path}, {"size", file.size}, {"digest", file.digest}};
}

engine::Json snapshot_json(const engine::Snapshot& snapshot) {
    auto files = engine::Json::array();
    for (const auto& file : snapshot.files) {
        files.push_back(file_json(file));
    }
    return engine::Json{{"digest", snapshot.digest}, {"files", std::move(files)}};
}

engine::Json check_result(const engine::Json& payload, std::int64_t deadline_unix_ms) {
    require_exact_fields(payload, {"expected_index_digest"});
    const auto& expected = payload.at("expected_index_digest");
    if (!expected.is_null() && (!expected.is_string() || !tagged_digest(expected.get<std::string>()))) {
        throw engine::Error(
            "payload.invalid_expected_digest", "expected_index_digest must be a tagged SHA-256 digest or null", 4);
    }

    auto state = analyze_index(fs::current_path(), deadline_unix_ms);
    engine::Json expected_matches = nullptr;
    if (expected.is_string()) {
        expected_matches = expected.get<std::string>() == state.index_file.digest;
        if (!expected_matches.get<bool>()) {
            add_finding(state, Finding{
                "violation", "skvi.index.expected_digest_mismatch", index_path,
                "expected=" + expected.get<std::string>() + " observed=" + state.index_file.digest,
            });
        }
    }
    const auto pass = finding_count(state, "pass");
    const auto warning = finding_count(state, "warning");
    const auto violation = finding_count(state, "violation");
    return engine::Json{
        {"protocol", check_protocol},
        {"index", file_json(state.index_file)},
        {"contract_snapshot", snapshot_json(state.contract_snapshot)},
        {"expected_index_matches", expected_matches},
        {"entries_checked", state.entries.size()},
        {"relationships_checked", state.relationships_checked},
        {"evidence", evidence_json(state)},
        {"summary", engine::Json{
            {"pass", pass},
            {"warning", warning},
            {"violation", violation},
            {"state", violation == 0U ? "valid" : "invalid"},
        }},
        {"read_only", true},
        {"canonical_apply_enabled", false},
    };
}

void require_clean(const IndexState& state) {
    if (finding_count(state, "violation") != 0U) {
        throw engine::Error("skvi.index_invalid", "canonical SKVI index failed structural checks", 4);
    }
}

engine::Json inspect(const engine::Json& payload) {
    require_exact_fields(payload, {});
    return engine::Json{
        {"descriptor", descriptor()},
        {"canonical_index", index_path},
        {"readiness", "read_check_propose_project"},
        {"engine_decides_membership", false},
        {"canonical_apply_enabled", false},
        {"session_mutation_enabled", false},
        {"maestro_docking_enabled", false},
    };
}

engine::Json project(const engine::Json& payload, std::int64_t deadline_unix_ms) {
    require_exact_fields(payload, {"format"});
    const auto format = require_string(payload, "format", 16, true);
    if (format != "json") {
        throw engine::Error("payload.unsupported_format", "only the json projection format is implemented", 4);
    }
    const auto state = analyze_index(fs::current_path(), deadline_unix_ms);
    require_clean(state);

    auto entries = engine::Json::array();
    for (const auto& entry : state.entries) {
        entries.push_back(entry_json(entry));
    }
    engine::Json result{
        {"protocol", projection_protocol},
        {"projection_kind", "structural-index"},
        {"format", "json"},
        {"module_id", module_id},
        {"engine_id", engine_id},
        {"engine_version", engine_version},
        {"vector_id", vector_id},
        {"canonical_index", file_json(state.index_file)},
        {"contract_snapshot", snapshot_json(state.contract_snapshot)},
        {"entry_count", state.entries.size()},
        {"entries", std::move(entries)},
        {"noncanonical", true},
        {"rebuildable", true},
    };
    result["projection_digest"] = engine::tagged_sha256(result.dump());
    return result;
}

Entry entry_from_payload(const engine::Json& value) {
    std::set<std::string> fields;
    for (const auto* field : entry_fields) {
        fields.insert(field);
    }
    fields.insert("status");
    require_exact_fields(value, fields);
    Entry entry;
    for (const auto& field : fields) {
        const auto& item = value.at(field);
        if (!item.is_string() || !field_bounded(item.get_ref<const std::string&>())) {
            throw engine::Error("payload.invalid_entry_field", "entry field is absent or invalid: " + field, 4);
        }
        entry.fields[field] = item.get<std::string>();
    }
    if (!engine::is_safe_relative_path(entry.fields.at("path"))) {
        throw engine::Error("payload.invalid_entry_path", "entry path is not a safe relative path", 4);
    }
    if (entry.fields.at("status") != "canonical") {
        throw engine::Error("payload.invalid_entry_status", "entry status must be canonical", 4);
    }
    return entry;
}

engine::Json propose(const engine::Json& payload, std::int64_t deadline_unix_ms) {
    require_exact_fields(payload, {
        "repository", "session_ref", "context_ref", "created_at", "expires_at", "operation",
    });
    const auto created_at = require_string(payload, "created_at", 20);
    const auto expires_at = require_string(payload, "expires_at", 20);
    if (!strict_utc(created_at) || !strict_utc(expires_at) || created_at >= expires_at) {
        throw engine::Error("payload.invalid_time", "created_at and expires_at must be ordered strict UTC timestamps", 4);
    }

    const auto& repository = payload.at("repository");
    require_exact_fields(repository, {"repository_id", "revision", "worktree_id", "tree_digest"});
    static_cast<void>(require_string(repository, "repository_id", 256));
    static_cast<void>(require_string(repository, "worktree_id", 128, true));
    const auto tree_digest = require_string(repository, "tree_digest", 71);
    if (!tagged_digest(tree_digest)) {
        throw engine::Error("payload.invalid_tree_digest", "tree_digest must be a tagged SHA-256 digest", 4);
    }
    const auto& revision = repository.at("revision");
    require_exact_fields(revision, {"scheme", "value"});
    static_cast<void>(require_string(revision, "scheme", 64, true));
    static_cast<void>(require_string(revision, "value", 256));

    for (const auto* reference : {"session_ref", "context_ref"}) {
        const auto& value = payload.at(reference);
        if (!value.is_null() && (!value.is_string() || !safe_token(value.get<std::string>()))) {
            throw engine::Error("payload.invalid_reference", std::string(reference) + " must be a token or null", 4);
        }
    }

    const auto& operation = payload.at("operation");
    require_exact_fields(operation, {"type", "target_path", "expected_entry_digest", "entry"});
    const auto type = require_string(operation, "type", 32, true);
    if (type != "add_entry" && type != "replace_entry" && type != "remove_entry") {
        throw engine::Error("payload.invalid_operation", "unsupported caller-declared SKVI operation", 4);
    }
    const auto target_path = require_string(operation, "target_path", engine::Limits::max_path_bytes);
    if (!engine::is_safe_relative_path(target_path)) {
        throw engine::Error("payload.invalid_target_path", "target_path is not a safe relative path", 4);
    }

    const auto state = analyze_index(fs::current_path(), deadline_unix_ms);
    require_clean(state);
    const auto current = std::find_if(state.entries.begin(), state.entries.end(), [&](const Entry& entry) {
        const auto found = entry.fields.find("path");
        return found != entry.fields.end() && found->second == target_path;
    });

    engine::Json operation_data = operation;
    engine::Json expected_state_digest = nullptr;
    if (type == "add_entry") {
        if (current != state.entries.end() || !operation.at("expected_entry_digest").is_null()) {
            throw engine::Error("proposal.add_conflict", "add_entry requires an unindexed path and null expected digest", 4);
        }
        if (operation.at("entry").is_null()) {
            throw engine::Error("proposal.entry_required", "add_entry requires an explicit entry", 4);
        }
        const auto candidate = entry_from_payload(operation.at("entry"));
        if (candidate.fields.at("path") != target_path) {
            throw engine::Error("proposal.path_mismatch", "entry path does not match target_path", 4);
        }
        static_cast<void>(engine::read_regular_file_no_follow(
            fs::current_path(), target_path, engine::Limits::max_snapshot_file_bytes, deadline_unix_ms));
    } else {
        if (current == state.entries.end()) {
            throw engine::Error("proposal.target_unindexed", "replace/remove target is not currently indexed", 4);
        }
        const auto current_digest = entry_json(*current).at("entry_digest").get<std::string>();
        const auto& expected = operation.at("expected_entry_digest");
        if (!expected.is_string() || !tagged_digest(expected.get<std::string>()) ||
            expected.get<std::string>() != current_digest) {
            throw engine::Error("proposal.expected_state_mismatch", "entry expected-state digest is absent, invalid, or stale", 4);
        }
        expected_state_digest = current_digest;
        if (type == "replace_entry") {
            if (operation.at("entry").is_null()) {
                throw engine::Error("proposal.entry_required", "replace_entry requires an explicit entry", 4);
            }
            const auto candidate = entry_from_payload(operation.at("entry"));
            if (candidate.fields.at("path") != target_path) {
                throw engine::Error("proposal.path_mismatch", "entry path does not match target_path", 4);
            }
        } else if (!operation.at("entry").is_null()) {
            throw engine::Error("proposal.entry_prohibited", "remove_entry requires a null entry", 4);
        }
    }

    const auto desired_change_digest = engine::tagged_sha256(operation_data.dump());
    const auto operation_id = "skvi-op:" + engine::sha256_hex(operation_data.dump());

    std::map<std::string, engine::FileDigest> read_files;
    read_files.emplace(state.index_file.path, state.index_file);
    for (const auto& file : state.contract_snapshot.files) {
        read_files.emplace(file.path, file);
    }
    if (const auto found = state.indexed_files.find(target_path); found != state.indexed_files.end()) {
        read_files.emplace(found->first, found->second);
    } else {
        const auto contents = engine::read_regular_file_no_follow(
            fs::current_path(), target_path, engine::Limits::max_snapshot_file_bytes, deadline_unix_ms);
        read_files.emplace(target_path, engine::FileDigest{
            target_path,
            static_cast<std::uint64_t>(contents.size()),
            engine::tagged_sha256(contents),
        });
    }
    auto read_set = engine::Json::array();
    for (const auto& [path, file] : read_files) {
        static_cast<void>(path);
        read_set.push_back(file_json(file));
    }

    engine::Json proposal{
        {"protocol", proposal_protocol},
        {"module_id", module_id},
        {"engine_id", engine_id},
        {"engine_version", engine_version},
        {"vector_id", vector_id},
        {"contract_versions", engine::Json::array({
            "knowledge/SPEC.md@v1", "knowledge/skvi/SPEC.md@v1",
        })},
        {"repository", repository},
        {"session_ref", payload.at("session_ref")},
        {"context_ref", payload.at("context_ref")},
        {"read_set", std::move(read_set)},
        {"write_set", engine::Json::array({engine::Json{
            {"target_path", index_path},
            {"expected_prior_digest", state.index_file.digest},
            {"desired_change_digest", desired_change_digest},
        }})},
        {"operations", engine::Json::array({engine::Json{
            {"operation_id", operation_id},
            {"type", type},
            {"target_path", target_path},
            {"expected_state_digest", expected_state_digest},
            {"desired_change_digest", desired_change_digest},
            {"data", operation_data},
        }})},
        {"validation", engine::Json::array({
            engine::Json{{"code", "skvi.index.valid"}, {"outcome", "pass"}, {"detail", "current index passed structural checks"}},
            engine::Json{{"code", "skvi.operation.caller_declared"}, {"outcome", "pass"}, {"detail", "operation semantics were supplied explicitly by the caller"}},
            engine::Json{{"code", "skvi.expected_state.bound"}, {"outcome", "pass"}, {"detail", "index and entry expected state are content-addressed"}},
        })},
        {"authority", engine::Json{
            {"caller_declared_operation", true},
            {"engine_decided_domain_truth", false},
            {"ratified", false},
        }},
        {"created_at", created_at},
        {"expires_at", expires_at},
        {"canonical_apply_enabled", false},
    };
    const auto identity = engine::sha256_hex(proposal.dump());
    proposal["proposal_id"] = "skvi-proposal:" + identity;
    proposal["proposal_digest"] = engine::tagged_sha256(proposal.dump());
    return proposal;
}

}

engine::Json descriptor() {
    return engine::Json{
        {"protocol", engine::descriptor_protocol_v1},
        {"module_id", module_id},
        {"engine_id", engine_id},
        {"vector_id", vector_id},
        {"engine_version", engine_version},
        {"process_protocols", engine::Json::array({engine::process_protocol_v1})},
        {"contract_versions", engine::Json::array({
            "knowledge/SPEC.md@v1", "knowledge/skvi/SPEC.md@v1",
        })},
        {"operations", engine::Json::array({
            engine::Json{{"name", "inspect"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "check"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "propose"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "project"}, {"availability", "implemented"}, {"mutates_canonical", false}},
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
        return check_result(request.payload, request.deadline_unix_ms);
    }
    if (request.operation == "propose") {
        return propose(request.payload, request.deadline_unix_ms);
    }
    if (request.operation == "project") {
        return project(request.payload, request.deadline_unix_ms);
    }
    throw engine::Error("operation.unsupported", "operation is reserved or unsupported", 4);
}

}
