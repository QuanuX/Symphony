#include "sclv.hpp"

#include "provider.hpp"
#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/limits.hpp"
#include "symphony/knowledge/engine/path.hpp"
#include "symphony/knowledge/engine/protocol.hpp"

#include <algorithm>
#include <array>
#include <cstddef>
#include <cstdint>
#include <filesystem>
#include <map>
#include <set>
#include <sstream>
#include <string>
#include <string_view>
#include <utility>
#include <vector>

namespace symphony::knowledge::sclv {
namespace fs = std::filesystem;

namespace {

constexpr const char* ledger_path = "knowledge/sclv/CHANGELOG.md";
constexpr const char* check_protocol = "symphony.sclv.check-result.v1";
constexpr const char* projection_protocol = "symphony.sclv.projection.v1";
constexpr const char* proposal_protocol = "symphony.knowledge.proposal.v1";
constexpr const char* recovery_protocol = "symphony.sclv.recovery-result.v1";
constexpr std::size_t max_records = 4096U;
constexpr std::size_t max_references = 1024U;
constexpr std::size_t max_findings = 1024U;
constexpr std::size_t max_provider_evidence = 8U;

const std::vector<std::string> contract_paths = {
    "knowledge/SPEC.md",
    "knowledge/schemas/v1/provider-evidence.schema.json",
    "knowledge/sclv/INTENT.md",
    "knowledge/sclv/MANIFEST.md",
    "knowledge/sclv/RECOVERY.md",
    "knowledge/sclv/SKILL.md",
    "knowledge/sclv/SPEC.md",
    "knowledge/sclv/schemas/v3/record.schema.json",
    "knowledge/sclv/schemas/v3/proposal-input.schema.json",
    "knowledge/sclv/schemas/v3/recovery-input.schema.json",
    "knowledge/sclv/schemas/v3/check-result.schema.json",
    "knowledge/sclv/schemas/v3/projection.schema.json",
    "knowledge/sclv/templates/v3/record.md",
    "knowledge/skvi/INDEX.md",
};

const std::array<const char*, 35> v3_field_order = {
    "record_id", "record_version", "title", "status", "date",
    "change_started_at", "change_completed_at", "recorded_at",
    "recording_disposition", "recovery_reason", "change_type",
    "change_request_state", "change_request_provider", "change_request_id",
    "change_request_reference", "change_request_absence_reason",
    "revision_scheme", "revision_value", "tree_digest",
    "ratification_subject", "ratification_permission", "ratification_method",
    "ratification_evidence_reference", "ratification_evidence_digest",
    "affected_surfaces", "skvi_references", "change_summary",
    "relationship_changes", "doctrine_changes", "compatibility_consequences",
    "publication_consequences", "projection_consequences", "evidence",
    "non_authorizations", "notes",
};

const std::set<std::string> v3_fields(v3_field_order.begin(), v3_field_order.end());

const std::set<std::string> list_fields = {
    "affected_surfaces", "skvi_references", "evidence", "non_authorizations",
};

const std::set<std::string> long_text_fields = {
    "change_summary", "relationship_changes", "doctrine_changes",
    "compatibility_consequences", "publication_consequences",
    "projection_consequences", "notes",
};

struct Record final {
    std::map<std::string, std::string> fields;
    std::map<std::string, std::vector<std::string>> lists;
    std::set<std::string> duplicates;
    std::size_t line = 0;
    int version = 1;
};

struct Finding final {
    std::string severity;
    std::string code;
    std::string record_id;
    std::string detail;
};

struct LedgerState final {
    std::string contents;
    engine::FileDigest ledger;
    engine::Snapshot contracts;
    std::vector<Record> records;
    std::vector<Finding> findings;
    std::size_t passes = 0;
    std::array<std::size_t, 3> version_counts{};
};

std::string trim(std::string_view value) {
    std::size_t begin = 0;
    while (begin < value.size() &&
           (value[begin] == ' ' || value[begin] == '\t' || value[begin] == '\r' || value[begin] == '\n')) ++begin;
    std::size_t end = value.size();
    while (end > begin &&
           (value[end - 1U] == ' ' || value[end - 1U] == '\t' ||
            value[end - 1U] == '\r' || value[end - 1U] == '\n')) --end;
    return std::string(value.substr(begin, end - begin));
}

std::string clean_scalar(std::string value) {
    value = trim(value);
    if (value.size() >= 2U && value.front() == '`' && value.back() == '`') {
        value = value.substr(1U, value.size() - 2U);
    }
    return value;
}

bool top_level_field(const std::string& line, std::string& field, std::string& value) {
    if (!line.starts_with("- ")) return false;
    const auto colon = line.find(':', 2U);
    if (colon == std::string::npos) return false;
    field = line.substr(2U, colon - 2U);
    if (field.empty() || field.front() == '`') return false;
    value = clean_scalar(line.substr(colon + 1U));
    return true;
}

bool list_item(const std::string& line, std::string& value) {
    if (!line.starts_with("  - ")) return false;
    value = clean_scalar(line.substr(4U));
    return !value.empty();
}

engine::Json file_json(const engine::FileDigest& file) {
    return engine::Json{{"path", file.path}, {"size", file.size}, {"digest", file.digest}};
}

engine::Json snapshot_json(const engine::Snapshot& snapshot) {
    auto files = engine::Json::array();
    for (const auto& file : snapshot.files) files.push_back(file_json(file));
    return engine::Json{{"digest", snapshot.digest}, {"files", std::move(files)}};
}

void add_finding(LedgerState& state, Finding finding) {
    if (finding.severity == "pass") {
        ++state.passes;
        return;
    }
    if (state.findings.size() >= max_findings) {
        throw engine::Error("sclv.evidence_limit", "SCLV exception evidence limit exceeded", 5);
    }
    state.findings.push_back(std::move(finding));
}

std::string record_id(const Record& record) {
    const auto found = record.fields.find("record_id");
    return found == record.fields.end() || found->second.empty()
        ? "line:" + std::to_string(record.line)
        : found->second;
}

void require_exact_fields(const engine::Json& object, const std::set<std::string>& fields) {
    if (!object.is_object() || object.size() != fields.size()) {
        throw engine::Error("payload.field_set", "object is incomplete or contains unknown fields", 4);
    }
    for (const auto& [key, value] : object.items()) {
        static_cast<void>(value);
        if (!fields.contains(key)) {
            throw engine::Error("payload.unknown_field", "object contains an unknown field", 4);
        }
    }
}

std::string required_text(
    const engine::Json& object,
    const char* field,
    std::size_t maximum = 4096U,
    bool token = false,
    bool newlines = false) {
    const auto& value = object.at(field);
    if (!value.is_string()) {
        throw engine::Error("payload.invalid_field", std::string(field) + " must be a string", 4);
    }
    const auto result = value.get<std::string>();
    if ((token && !provider::safe_token(result, maximum)) ||
        (!token && !provider::bounded_text(result, maximum, newlines))) {
        throw engine::Error("payload.invalid_field", std::string(field) + " has invalid syntax", 4);
    }
    return result;
}

std::vector<std::string> required_list(
    const engine::Json& object,
    const char* field,
    bool paths) {
    const auto& value = object.at(field);
    if (!value.is_array() || value.empty() || value.size() > max_references) {
        throw engine::Error("payload.invalid_list", std::string(field) + " must be a nonempty bounded array", 4);
    }
    std::vector<std::string> result;
    std::set<std::string> unique;
    for (const auto& item : value) {
        if (!item.is_string()) throw engine::Error("payload.invalid_list", std::string(field) + " contains a non-string", 4);
        const auto text = item.get<std::string>();
        if ((paths && !engine::is_safe_relative_path(text)) ||
            (!paths && !provider::bounded_text(text))) {
            throw engine::Error("payload.invalid_list", std::string(field) + " contains an invalid value", 4);
        }
        if (!unique.insert(text).second) {
            throw engine::Error("payload.duplicate_list_item", std::string(field) + " contains a duplicate", 4);
        }
        result.push_back(text);
    }
    return result;
}

bool lower_hex(std::string_view value, std::size_t length) {
    return value.size() == length && std::all_of(value.begin(), value.end(), [](const unsigned char character) {
        return (character >= '0' && character <= '9') || (character >= 'a' && character <= 'f');
    });
}

void validate_record_json(const engine::Json& record) {
    require_exact_fields(record, v3_fields);
    if (!record.at("record_version").is_number_integer() || record.at("record_version").get<int>() != 3) {
        throw engine::Error("record.version", "record_version must equal 3", 4);
    }
    const auto id = required_text(record, "record_id", 104U);
    if (!id.starts_with("SCLV-CHG-") || id.size() < 17U ||
        !provider::safe_token(id.substr(9U), 95U)) {
        throw engine::Error("record.id", "record_id must be a stable SCLV-CHG identifier", 4);
    }
    static_cast<void>(required_text(record, "title"));
    if (required_text(record, "status", 32U, true) != "canonical") {
        throw engine::Error("record.status", "status must be canonical", 4);
    }
    const auto date = required_text(record, "date", 10U);
    const auto started = required_text(record, "change_started_at", 20U);
    const auto completed = required_text(record, "change_completed_at", 20U);
    const auto recorded = required_text(record, "recorded_at", 20U);
    if (!provider::strict_utc(started) || !provider::strict_utc(completed) ||
        !provider::strict_utc(recorded) || started > completed || completed > recorded ||
        date != recorded.substr(0U, 10U)) {
        throw engine::Error("record.temporal", "record timestamps or date are invalid or unordered", 4);
    }
    const auto disposition = required_text(record, "recording_disposition", 32U, true);
    const auto recovery_reason = required_text(record, "recovery_reason");
    if ((disposition == "post_merge" && recovery_reason != "not_applicable") ||
        (disposition == "late_recovery" && recovery_reason == "not_applicable") ||
        (disposition != "post_merge" && disposition != "late_recovery")) {
        throw engine::Error("record.recovery", "recording disposition and recovery reason are inconsistent", 4);
    }
    static_cast<void>(required_text(record, "change_type", 128U, true));

    const engine::Json change_request{
        {"state", record.at("change_request_state")},
        {"provider", record.at("change_request_provider")},
        {"id", record.at("change_request_id")},
        {"reference", record.at("change_request_reference")},
        {"absence_reason", record.at("change_request_absence_reason")},
    };
    provider::validate_change_request(change_request);

    const auto scheme = required_text(record, "revision_scheme", 128U, true);
    const auto revision = required_text(record, "revision_value", 256U);
    if ((scheme == "git-sha1" && !lower_hex(revision, 40U)) ||
        (scheme == "git-sha256" && !lower_hex(revision, 64U))) {
        throw engine::Error("record.revision", "Git revision does not match its declared scheme", 4);
    }
    if (!provider::tagged_digest(required_text(record, "tree_digest", 71U)) ||
        !provider::tagged_digest(required_text(record, "ratification_evidence_digest", 71U))) {
        throw engine::Error("record.digest", "record digests must be tagged SHA-256", 4);
    }
    for (const auto* field : {
        "ratification_subject", "ratification_permission", "ratification_method",
        "ratification_evidence_reference",
    }) {
        if (required_text(record, field) == "not_applicable") {
            throw engine::Error("record.ratification", "ratification fields must name effective evidence", 4);
        }
    }
    static_cast<void>(required_list(record, "affected_surfaces", true));
    static_cast<void>(required_list(record, "skvi_references", true));
    static_cast<void>(required_list(record, "evidence", false));
    static_cast<void>(required_list(record, "non_authorizations", false));
    for (const auto* field : {
        "change_summary", "relationship_changes", "doctrine_changes",
        "compatibility_consequences", "publication_consequences",
        "projection_consequences", "notes",
    }) {
        static_cast<void>(required_text(record, field, 65536U, false, true));
    }
}

engine::Json parsed_v3_json(const Record& record) {
    engine::Json result = engine::Json::object();
    for (const auto* field : v3_field_order) {
        if (std::string_view(field) == "record_version") {
            result[field] = record.version;
        } else if (list_fields.contains(field)) {
            const auto found = record.lists.find(field);
            result[field] = found == record.lists.end() ? engine::Json::array() : engine::Json(found->second);
        } else {
            const auto found = record.fields.find(field);
            result[field] = found == record.fields.end() ? "" : found->second;
        }
    }
    return result;
}

std::vector<Record> parse_records(const std::string& contents) {
    std::vector<Record> records;
    Record current;
    bool in_record = false;
    std::string active_list;
    std::string active_block;
    std::size_t line_number = 0;
    std::size_t position = 0;
    while (position <= contents.size()) {
        const auto end = contents.find('\n', position);
        const auto line = contents.substr(position, end == std::string::npos ? std::string::npos : end - position);
        ++line_number;
        std::string field;
        std::string value;
        if (top_level_field(line, field, value)) {
            if (field == "record_id") {
                if (in_record) records.push_back(std::move(current));
                if (records.size() >= max_records) {
                    throw engine::Error("sclv.record_limit", "SCLV record count exceeds the bounded limit", 5);
                }
                current = Record{};
                current.line = line_number;
                in_record = true;
            }
            if (in_record) {
                if (!current.fields.emplace(field, value).second) current.duplicates.insert(field);
                active_list = list_fields.contains(field) ? field : "";
                active_block = long_text_fields.contains(field) && value == "|" ? field : "";
            }
        } else if (in_record) {
            std::string item;
            if (!active_list.empty() && list_item(line, item)) {
                current.lists[active_list].push_back(std::move(item));
            } else if (!active_block.empty() && line.starts_with("    ")) {
                auto& block = current.fields[active_block];
                if (block == "|") block.clear();
                if (!block.empty()) block.push_back('\n');
                block.append(line.substr(4U));
            } else if (!trim(line).empty()) {
                active_list.clear();
                active_block.clear();
            }
        }
        if (end == std::string::npos) break;
        position = end + 1U;
    }
    if (in_record) records.push_back(std::move(current));
    return records;
}

std::set<std::string> indexed_paths(const fs::path& root, std::int64_t deadline_unix_ms);

LedgerState analyze_ledger(const fs::path& root, std::int64_t deadline_unix_ms) {
    LedgerState state;
    state.contents = engine::read_regular_file_no_follow(
        root, ledger_path, engine::Limits::max_snapshot_file_bytes, deadline_unix_ms);
    state.ledger = engine::FileDigest{
        ledger_path,
        static_cast<std::uint64_t>(state.contents.size()),
        engine::tagged_sha256(state.contents),
    };
    state.contracts = engine::snapshot_files(root, contract_paths, deadline_unix_ms);
    const auto index = indexed_paths(root, deadline_unix_ms);
    const auto index_contents = engine::read_regular_file_no_follow(
        root, "knowledge/skvi/INDEX.md", engine::Limits::max_snapshot_file_bytes, deadline_unix_ms);
    const auto snapshot_index = std::find_if(
        state.contracts.files.begin(), state.contracts.files.end(), [](const engine::FileDigest& file) {
            return file.path == "knowledge/skvi/INDEX.md";
        });
    if (snapshot_index == state.contracts.files.end() ||
        snapshot_index->digest != engine::tagged_sha256(index_contents)) {
        throw engine::Error("sclv.snapshot_changed", "SKVI index changed during bounded ledger analysis", 3);
    }
    state.records = parse_records(state.contents);
    if (state.records.empty()) {
        add_finding(state, {"violation", "sclv.record.none", "ledger", "no canonical records were detected"});
        return state;
    }

    std::set<std::string> ids;
    std::set<std::string> revisions;
    std::set<std::string> change_requests;
    std::string latest_recorded;
    for (auto& record : state.records) {
        const auto id = record_id(record);
        const auto version_found = record.fields.find("record_version");
        int version = 1;
        if (version_found != record.fields.end()) {
            if (version_found->second == "1") version = 1;
            else if (version_found->second == "2") version = 2;
            else if (version_found->second == "3") version = 3;
            else version = 0;
        }
        record.version = version;
        if (version >= 1 && version <= 3) ++state.version_counts[static_cast<std::size_t>(version - 1)];
        else add_finding(state, {"violation", "sclv.record.version", id, "unsupported record version"});
        if (!ids.insert(id).second) {
            add_finding(state, {"violation", "sclv.record.duplicate_id", id, "record identifier is duplicated"});
        } else {
            add_finding(state, {"pass", "sclv.record.unique_id", id, "record identifier is unique"});
        }
        if (!record.duplicates.empty()) {
            add_finding(state, {"violation", "sclv.record.duplicate_field", id, "record contains duplicate fields"});
        }
        if (version == 3) {
            if (record.fields.size() != v3_fields.size()) {
                add_finding(state, {"violation", "sclv.record.v3_field_set", id, "v3 field set is not exact"});
            }
            try {
                const auto value = parsed_v3_json(record);
                validate_record_json(value);
                add_finding(state, {"pass", "sclv.record.v3_shape", id, "v3 record satisfies the provider-neutral contract"});
                const auto revision_key = value.at("revision_scheme").get<std::string>() + ":" +
                                          value.at("revision_value").get<std::string>();
                if (!revisions.insert(revision_key).second) {
                    add_finding(state, {"violation", "sclv.record.duplicate_revision", id, "revision is already recorded"});
                }
                if (value.at("change_request_state") == "present") {
                    const auto request_key = value.at("change_request_provider").get<std::string>() + ":" +
                                             value.at("change_request_id").get<std::string>();
                    if (!change_requests.insert(request_key).second) {
                        add_finding(state, {"violation", "sclv.record.duplicate_change_request", id, "change request is already recorded"});
                    }
                }
                const auto recorded = value.at("recorded_at").get<std::string>();
                if (!latest_recorded.empty() && recorded < latest_recorded) {
                    add_finding(state, {"violation", "sclv.record.recording_order", id, "recorded_at moves backward"});
                }
                latest_recorded = recorded;
            } catch (const engine::Error& error) {
                add_finding(state, {"violation", error.code(), id, error.what()});
            }
        } else {
            const std::set<std::string> base_required = {
                "record_id", "title", "status", "date", "change_type", "related_pr", "merge_commit",
                "affected_surfaces", "skvi_references", "change_summary", "relationship_changes",
                "doctrine_changes", "compatibility_consequences", "publication_consequences",
                "projection_consequences", "evidence", "non_authorizations", "notes",
            };
            bool valid = true;
            for (const auto& field : base_required) {
                if (!record.fields.contains(field)) valid = false;
            }
            if (version == 2) {
                for (const auto& field : {
                    "record_version", "change_started_at", "change_completed_at",
                    "recorded_at", "recording_disposition",
                }) {
                    if (!record.fields.contains(field)) valid = false;
                }
                const auto started = record.fields.find("change_started_at");
                const auto completed = record.fields.find("change_completed_at");
                const auto recorded = record.fields.find("recorded_at");
                if (started == record.fields.end() || completed == record.fields.end() || recorded == record.fields.end() ||
                    !provider::strict_utc(started->second) || !provider::strict_utc(completed->second) ||
                    !provider::strict_utc(recorded->second) || started->second > completed->second ||
                    completed->second > recorded->second) valid = false;
                if (recorded != record.fields.end()) {
                    if (!latest_recorded.empty() && recorded->second < latest_recorded) valid = false;
                    latest_recorded = recorded->second;
                }
            }
            add_finding(state, {
                valid ? "pass" : "violation",
                valid ? "sclv.record.legacy_shape" : "sclv.record.legacy_invalid",
                id,
                valid ? "legacy record satisfies its immutable compatibility shape" : "legacy record is incomplete or temporally invalid",
            });
        }
        for (const auto& field : {"affected_surfaces", "skvi_references"}) {
            const auto found = record.lists.find(field);
            if (found == record.lists.end() || found->second.empty() || found->second.size() > max_references) {
                add_finding(state, {"violation", "sclv.record.reference_list", id, std::string(field) + " is absent or exceeds its bound"});
                continue;
            }
            const bool safe = std::all_of(found->second.begin(), found->second.end(), engine::is_safe_relative_path);
            add_finding(state, {
                safe ? "pass" : "violation",
                safe ? "sclv.record.paths_safe" : "sclv.record.path_invalid",
                id,
                safe ? std::string(field) + " paths are safe" : std::string(field) + " contains an unsafe path",
            });
            if (std::string_view(field) == "skvi_references" && safe) {
                for (const auto& path : found->second) {
                    if (!index.contains(path)) {
                        add_finding(state, {"violation", "sclv.record.skvi_unindexed", id, "SKVI reference is not indexed: " + path});
                    }
                }
            }
        }
    }
    return state;
}

std::size_t violations(const LedgerState& state) {
    return static_cast<std::size_t>(std::count_if(state.findings.begin(), state.findings.end(), [](const Finding& item) {
        return item.severity == "violation";
    }));
}

std::size_t warnings(const LedgerState& state) {
    return static_cast<std::size_t>(std::count_if(state.findings.begin(), state.findings.end(), [](const Finding& item) {
        return item.severity == "warning";
    }));
}

engine::Json inspect(const engine::Json& payload) {
    require_exact_fields(payload, {});
    return engine::Json{
        {"descriptor", descriptor()},
        {"canonical_ledger", ledger_path},
        {"record_contract", "knowledge/sclv/schemas/v3/record.schema.json"},
        {"record_template", "knowledge/sclv/templates/v3/record.md"},
        {"evidence_adapters", engine::Json::array({
            "symphony-sclv-evidence-local-git", "symphony-sclv-evidence-airgap",
        })},
        {"record_versions", engine::Json::array({1, 2, 3})},
        {"read_only", true},
        {"canonical_apply_enabled", false},
    };
}

engine::Json check(const engine::Json& payload, std::int64_t deadline_unix_ms) {
    require_exact_fields(payload, {"expected_ledger_digest"});
    const auto& expected = payload.at("expected_ledger_digest");
    if (!expected.is_null() && (!expected.is_string() || !provider::tagged_digest(expected.get<std::string>()))) {
        throw engine::Error("payload.expected_digest", "expected_ledger_digest must be null or tagged SHA-256", 4);
    }
    auto state = analyze_ledger(fs::current_path(), deadline_unix_ms);
    engine::Json expected_matches = nullptr;
    if (!expected.is_null()) {
        expected_matches = expected.get<std::string>() == state.ledger.digest;
        add_finding(state, {
            expected_matches.get<bool>() ? "pass" : "violation",
            expected_matches.get<bool>() ? "sclv.ledger.expected_match" : "sclv.ledger.expected_mismatch",
            "ledger",
            expected_matches.get<bool>() ? "ledger digest matches expected state" : "ledger digest differs from expected state",
        });
    }
    auto evidence = engine::Json::array();
    for (const auto& item : state.findings) {
        if (item.severity != "pass") {
            evidence.push_back(engine::Json{
                {"severity", item.severity}, {"code", item.code},
                {"record_id", item.record_id}, {"detail", item.detail},
            });
        }
    }
    const auto violation_count = violations(state);
    const auto warning_count = warnings(state);
    return engine::Json{
        {"protocol", check_protocol},
        {"ledger", file_json(state.ledger)},
        {"contract_snapshot", snapshot_json(state.contracts)},
        {"expected_ledger_matches", expected_matches},
        {"records_checked", state.records.size()},
        {"version_counts", engine::Json{
            {"v1", state.version_counts[0]}, {"v2", state.version_counts[1]}, {"v3", state.version_counts[2]},
        }},
        {"evidence", std::move(evidence)},
        {"summary", engine::Json{
            {"pass", state.passes}, {"warning", warning_count}, {"violation", violation_count},
            {"state", violation_count == 0U ? "valid" : "invalid"},
        }},
        {"read_only", true},
        {"canonical_apply_enabled", false},
    };
}

std::string block_value(const std::string& value) {
    std::ostringstream result;
    std::size_t position = 0;
    for (;;) {
        const auto end = value.find('\n', position);
        result << "    " << value.substr(position, end == std::string::npos ? std::string::npos : end - position) << '\n';
        if (end == std::string::npos) break;
        position = end + 1U;
    }
    return result.str();
}

std::string render_record(const engine::Json& record) {
    std::ostringstream output;
    for (const auto* field : v3_field_order) {
        const std::string name = field;
        if (name == "record_version") {
            output << "- record_version: `3`\n";
        } else if (list_fields.contains(name)) {
            output << "- " << name << ":\n";
            for (const auto& item : record.at(name)) output << "  - `" << item.get<std::string>() << "`\n";
        } else if (long_text_fields.contains(name)) {
            output << "- " << name << ": |\n" << block_value(record.at(name).get<std::string>());
        } else {
            output << "- " << name << ": `" << record.at(name).get<std::string>() << "`\n";
        }
    }
    return output.str();
}

std::set<std::string> indexed_paths(const fs::path& root, std::int64_t deadline_unix_ms) {
    const auto contents = engine::read_regular_file_no_follow(
        root, "knowledge/skvi/INDEX.md", engine::Limits::max_snapshot_file_bytes, deadline_unix_ms);
    std::set<std::string> result;
    std::size_t position = 0;
    while (position <= contents.size()) {
        const auto end = contents.find('\n', position);
        const auto line = contents.substr(position, end == std::string::npos ? std::string::npos : end - position);
        const auto normalized = trim(line);
        if (normalized.starts_with("- path:")) {
            auto value = clean_scalar(normalized.substr(7U));
            if (engine::is_safe_relative_path(value)) result.insert(std::move(value));
        }
        if (end == std::string::npos) break;
        position = end + 1U;
    }
    return result;
}

void validate_repository(const engine::Json& repository) {
    require_exact_fields(repository, {"repository_id", "revision", "worktree_id", "tree_digest"});
    static_cast<void>(required_text(repository, "repository_id", 256U));
    static_cast<void>(required_text(repository, "worktree_id", 128U, true));
    if (!provider::tagged_digest(required_text(repository, "tree_digest", 71U))) {
        throw engine::Error("payload.repository_digest", "repository tree_digest must be tagged SHA-256", 4);
    }
    const auto& revision = repository.at("revision");
    require_exact_fields(revision, {"scheme", "value"});
    static_cast<void>(required_text(revision, "scheme", 128U, true));
    static_cast<void>(required_text(revision, "value", 256U));
}

engine::Json propose(const engine::Json& payload, std::int64_t deadline_unix_ms) {
    require_exact_fields(payload, {
        "repository", "session_ref", "context_ref", "proposal_expires_at", "record", "provider_evidence",
    });
    validate_repository(payload.at("repository"));
    for (const auto* field : {"session_ref", "context_ref"}) {
        const auto& value = payload.at(field);
        if (!value.is_null() && (!value.is_string() || !provider::safe_token(value.get<std::string>()))) {
            throw engine::Error("payload.reference", std::string(field) + " must be null or a safe token", 4);
        }
    }
    const auto expires_at = required_text(payload, "proposal_expires_at", 20U);
    if (!provider::strict_utc(expires_at)) throw engine::Error("payload.expiry", "proposal_expires_at is invalid", 4);
    const auto& record = payload.at("record");
    validate_record_json(record);
    const auto created_at = record.at("recorded_at").get<std::string>();
    if (expires_at <= created_at) throw engine::Error("payload.expiry", "proposal must expire after recorded_at", 4);

    const auto& evidence = payload.at("provider_evidence");
    if (!evidence.is_array() || evidence.empty() || evidence.size() > max_provider_evidence) {
        throw engine::Error("payload.evidence_count", "provider_evidence must contain one to eight envelopes", 4);
    }
    bool repository_match = false;
    bool ratification_match = false;
    bool change_request_match = false;
    for (const auto& item : evidence) {
        provider::validate_evidence(item);
        if (!item.at("repository").is_null()) {
            const auto& repository_evidence = item.at("repository");
            repository_match = repository_match ||
                (repository_evidence.at("revision_scheme") == record.at("revision_scheme") &&
                 repository_evidence.at("revision_value") == record.at("revision_value") &&
                 repository_evidence.at("tree_digest") == record.at("tree_digest"));
        }
        const auto& ratification = item.at("ratification");
        ratification_match = ratification_match ||
            (ratification.at("state") == "asserted" &&
             ratification.at("subject") == record.at("ratification_subject") &&
             ratification.at("effective_permission") == record.at("ratification_permission") &&
             ratification.at("method") == record.at("ratification_method") &&
             ratification.at("evidence_reference") == record.at("ratification_evidence_reference") &&
             ratification.at("evidence_digest") == record.at("ratification_evidence_digest"));
        const auto& request = item.at("change_request");
        change_request_match = change_request_match ||
            (request.at("state") == record.at("change_request_state") &&
             request.at("provider") == record.at("change_request_provider") &&
             request.at("id") == record.at("change_request_id") &&
             request.at("reference") == record.at("change_request_reference") &&
             request.at("absence_reason") == record.at("change_request_absence_reason"));
    }
    if (!repository_match || !ratification_match || !change_request_match) {
        throw engine::Error("proposal.evidence_mismatch", "provider evidence does not bind revision, change request, and ratification claims", 4);
    }
    const auto& repository = payload.at("repository");
    if (repository.at("revision").at("scheme") != record.at("revision_scheme") ||
        repository.at("revision").at("value") != record.at("revision_value") ||
        repository.at("tree_digest") != record.at("tree_digest")) {
        throw engine::Error("proposal.repository_mismatch", "caller repository state does not match the record", 4);
    }

    const auto root = fs::current_path();
    auto state = analyze_ledger(root, deadline_unix_ms);
    if (violations(state) != 0U) throw engine::Error("proposal.ledger_invalid", "canonical SCLV ledger is not clean", 4);
    const auto id = record.at("record_id").get<std::string>();
    for (const auto& existing : state.records) {
        if (record_id(existing) == id) throw engine::Error("proposal.duplicate_id", "record_id already exists", 4);
        if (existing.version == 3) {
            const auto normalized = parsed_v3_json(existing);
            if (normalized.at("revision_scheme") == record.at("revision_scheme") &&
                normalized.at("revision_value") == record.at("revision_value")) {
                throw engine::Error("proposal.duplicate_revision", "revision is already recorded", 4);
            }
            if (record.at("change_request_state") == "present" &&
                normalized.at("change_request_state") == "present" &&
                normalized.at("change_request_provider") == record.at("change_request_provider") &&
                normalized.at("change_request_id") == record.at("change_request_id")) {
                throw engine::Error("proposal.duplicate_change_request", "change request is already recorded", 4);
            }
        }
    }
    if (!state.records.empty()) {
        const auto& last = state.records.back();
        if (last.version >= 2) {
            const auto found = last.fields.find("recorded_at");
            if (found != last.fields.end() && created_at < found->second) {
                throw engine::Error("proposal.recording_order", "recorded_at moves backward", 4);
            }
        }
    }

    const auto index = indexed_paths(root, deadline_unix_ms);
    std::map<std::string, engine::FileDigest> reads;
    reads.emplace(state.ledger.path, state.ledger);
    for (const auto& file : state.contracts.files) reads.emplace(file.path, file);
    for (const auto* list : {"affected_surfaces", "skvi_references"}) {
        for (const auto& item : record.at(list)) {
            const auto path = item.get<std::string>();
            const auto contents = engine::read_regular_file_no_follow(
                root, path, engine::Limits::max_snapshot_file_bytes, deadline_unix_ms);
            reads.emplace(path, engine::FileDigest{
                path, static_cast<std::uint64_t>(contents.size()), engine::tagged_sha256(contents),
            });
            if (std::string_view(list) == "skvi_references" && !index.contains(path)) {
                throw engine::Error("proposal.skvi_reference", "skvi_references contains an unindexed path", 4);
            }
        }
    }
    auto read_set = engine::Json::array();
    for (const auto& [path, file] : reads) {
        static_cast<void>(path);
        read_set.push_back(file_json(file));
    }

    const auto markdown = render_record(record);
    const auto desired_change_digest = engine::tagged_sha256(markdown);
    const auto operation_id = "sclv-op:" + engine::sha256_hex(id + ":" + desired_change_digest);
    engine::Json proposal{
        {"protocol", proposal_protocol},
        {"module_id", module_id}, {"engine_id", engine_id},
        {"engine_version", engine_version}, {"vector_id", vector_id},
        {"contract_versions", engine::Json::array({"knowledge/SPEC.md@v1", "knowledge/sclv/SPEC.md@v3"})},
        {"repository", repository},
        {"session_ref", payload.at("session_ref")}, {"context_ref", payload.at("context_ref")},
        {"read_set", std::move(read_set)},
        {"write_set", engine::Json::array({engine::Json{
            {"target_path", ledger_path}, {"expected_prior_digest", state.ledger.digest},
            {"desired_change_digest", desired_change_digest},
        }})},
        {"operations", engine::Json::array({engine::Json{
            {"operation_id", operation_id}, {"type", "append_record_v3"},
            {"target_path", ledger_path}, {"expected_state_digest", state.ledger.digest},
            {"desired_change_digest", desired_change_digest},
            {"data", engine::Json{{"record", record}, {"markdown", markdown}}},
        }})},
        {"validation", engine::Json::array({
            engine::Json{{"code", "sclv.ledger.valid"}, {"outcome", "pass"}, {"detail", "current ledger passed bounded validation"}},
            engine::Json{{"code", "sclv.record.v3_valid"}, {"outcome", "pass"}, {"detail", "caller-declared v3 record passed exact validation"}},
            engine::Json{{"code", "sclv.provider_evidence.bound"}, {"outcome", "pass"}, {"detail", "revision, change-request, and ratification claims match normalized evidence"}},
        })},
        {"authority", engine::Json{
            {"caller_declared_operation", true}, {"engine_decided_domain_truth", false},
            {"ratified", false},
        }},
        {"created_at", created_at}, {"expires_at", expires_at},
        {"canonical_apply_enabled", false},
    };
    proposal["proposal_id"] = "sclv-proposal:" + engine::sha256_hex(proposal.dump());
    proposal["proposal_digest"] = engine::tagged_sha256(proposal.dump());
    return proposal;
}

engine::Json recovery(const engine::Json& payload, std::int64_t deadline_unix_ms) {
    require_exact_fields(payload, {"journal", "journal_digest", "observed_state", "proposal_input", "recovery_reason"});
    const auto& journal = payload.at("journal");
    require_exact_fields(journal, {
        "format_version", "session_id", "source_operation", "base_revision", "intended_surfaces",
        "started_at", "known_change_request", "known_revision", "local_state",
    });
    if (!journal.at("format_version").is_number_integer() || journal.at("format_version").get<int>() != 1) {
        throw engine::Error("recovery.journal_version", "journal format_version must equal 1", 4);
    }
    static_cast<void>(required_text(journal, "session_id", 128U, true));
    static_cast<void>(required_text(journal, "source_operation", 128U, true));
    if (!provider::strict_utc(required_text(journal, "started_at", 20U))) {
        throw engine::Error("recovery.started_at", "journal started_at is invalid", 4);
    }
    static_cast<void>(required_list(journal, "intended_surfaces", true));
    const auto validate_revision = [](const engine::Json& revision) {
        require_exact_fields(revision, {"scheme", "value"});
        static_cast<void>(required_text(revision, "scheme", 128U, true));
        static_cast<void>(required_text(revision, "value", 256U));
    };
    validate_revision(journal.at("base_revision"));
    validate_revision(journal.at("known_revision"));
    provider::validate_change_request(journal.at("known_change_request"));
    const auto local_state = required_text(journal, "local_state", 32U, true);
    if (local_state != "pending" && local_state != "resumable" && local_state != "abandonable") {
        throw engine::Error("recovery.local_state", "journal local_state is invalid", 4);
    }
    const auto digest = required_text(payload, "journal_digest", 71U);
    if (!provider::tagged_digest(digest) || digest != engine::tagged_sha256(journal.dump())) {
        throw engine::Error("recovery.journal_digest", "journal_digest does not match the exact journal", 4);
    }
    const auto observed = required_text(payload, "observed_state", 64U, true);
    const auto reason = required_text(payload, "recovery_reason");
    const std::set<std::string> states = {
        "still_open", "closed_without_merge", "merged_already_recorded", "merged_unrecorded", "indeterminate",
    };
    if (!states.contains(observed)) throw engine::Error("recovery.observed_state", "observed_state is invalid", 4);
    if (observed == "indeterminate") {
        throw engine::Error("recovery.indeterminate", "recovery evidence is indeterminate and requires permission-backed review", 4);
    }

    engine::Json result{
        {"protocol", recovery_protocol}, {"journal_digest", digest}, {"observed_state", observed},
        {"reason", reason}, {"journal_mutated", false}, {"canonical_apply_enabled", false},
        {"proposal", nullptr}, {"delete_recommended", false},
    };
    if (observed == "still_open") {
        if (!payload.at("proposal_input").is_null()) {
            throw engine::Error("recovery.proposal_prohibited", "still_open does not accept proposal_input", 4);
        }
        result["action"] = "resume";
    } else if (observed == "closed_without_merge") {
        if (!payload.at("proposal_input").is_null()) {
            throw engine::Error("recovery.proposal_prohibited", "closed_without_merge does not accept proposal_input", 4);
        }
        result["action"] = "abandon";
        result["delete_recommended"] = true;
    } else if (observed == "merged_already_recorded") {
        if (!payload.at("proposal_input").is_null()) {
            throw engine::Error("recovery.proposal_prohibited", "merged_already_recorded does not accept proposal_input", 4);
        }
        result["action"] = "no_op";
        result["delete_recommended"] = true;
    } else {
        if (payload.at("proposal_input").is_null()) {
            throw engine::Error("recovery.proposal_required", "merged_unrecorded requires proposal_input", 4);
        }
        const auto& proposal_input = payload.at("proposal_input");
        if (!proposal_input.is_object() || !proposal_input.contains("record") ||
            !proposal_input.at("record").is_object() ||
            !proposal_input.at("record").contains("recording_disposition") ||
            !proposal_input.at("record").contains("recovery_reason") ||
            proposal_input.at("record").at("recording_disposition") != "late_recovery" ||
            proposal_input.at("record").at("recovery_reason") != reason) {
            throw engine::Error("recovery.late_record", "late-recovery record must bind the supplied recovery reason", 4);
        }
        result["action"] = "propose_late_recovery";
        result["proposal"] = propose(proposal_input, deadline_unix_ms);
    }
    result["result_digest"] = engine::tagged_sha256(result.dump());
    return result;
}

engine::Json projection_record(const Record& record) {
    const auto id = record_id(record);
    const auto field = [&](const char* name, const std::string& fallback = "not_available_legacy") {
        const auto found = record.fields.find(name);
        return found == record.fields.end() || found->second.empty() ? fallback : found->second;
    };
    const auto list = [&](const char* name) {
        const auto found = record.lists.find(name);
        return found == record.lists.end() ? std::vector<std::string>{} : found->second;
    };
    engine::Json result;
    if (record.version == 3) {
        const auto value = parsed_v3_json(record);
        result = engine::Json{
            {"record_id", id}, {"record_version", 3}, {"title", value.at("title")},
            {"status", value.at("status")}, {"recorded_at", value.at("recorded_at")},
            {"change_request", engine::Json{
                {"state", value.at("change_request_state")}, {"provider", value.at("change_request_provider")},
                {"id", value.at("change_request_id")}, {"reference", value.at("change_request_reference")},
                {"absence_reason", value.at("change_request_absence_reason")},
            }},
            {"revision", engine::Json{{"scheme", value.at("revision_scheme")}, {"value", value.at("revision_value")}}},
            {"tree_digest", value.at("tree_digest")},
            {"ratification", engine::Json{
                {"subject", value.at("ratification_subject")}, {"permission", value.at("ratification_permission")},
                {"method", value.at("ratification_method")}, {"evidence_reference", value.at("ratification_evidence_reference")},
                {"evidence_digest", value.at("ratification_evidence_digest")},
            }},
            {"affected_surfaces", value.at("affected_surfaces")}, {"legacy_normalization", false},
        };
    } else {
        result = engine::Json{
            {"record_id", id}, {"record_version", record.version}, {"title", field("title")},
            {"status", field("status")}, {"recorded_at", field("recorded_at", field("date"))},
            {"change_request", engine::Json{
                {"state", "present"}, {"provider", "legacy-forge"}, {"id", field("related_pr")},
                {"reference", field("related_pr")}, {"absence_reason", "not_applicable"},
            }},
            {"revision", engine::Json{{"scheme", "git-sha1"}, {"value", field("merge_commit")}}},
            {"tree_digest", "not_available_legacy"},
            {"ratification", engine::Json{{"state", "not_available_legacy"}}},
            {"affected_surfaces", list("affected_surfaces")}, {"legacy_normalization", true},
        };
    }
    result["record_digest"] = engine::tagged_sha256(result.dump());
    return result;
}

engine::Json project(const engine::Json& payload, std::int64_t deadline_unix_ms) {
    require_exact_fields(payload, {"format"});
    if (required_text(payload, "format", 16U, true) != "json") {
        throw engine::Error("payload.format", "only the json projection format is implemented", 4);
    }
    const auto state = analyze_ledger(fs::current_path(), deadline_unix_ms);
    if (violations(state) != 0U) throw engine::Error("projection.ledger_invalid", "canonical SCLV ledger is not clean", 4);
    auto records = engine::Json::array();
    for (const auto& record : state.records) records.push_back(projection_record(record));
    engine::Json result{
        {"protocol", projection_protocol}, {"projection_kind", "provider-neutral-change-ledger"},
        {"format", "json"}, {"module_id", module_id}, {"engine_id", engine_id},
        {"engine_version", engine_version}, {"vector_id", vector_id},
        {"canonical_ledger", file_json(state.ledger)}, {"contract_snapshot", snapshot_json(state.contracts)},
        {"record_count", records.size()}, {"records", std::move(records)},
        {"noncanonical", true}, {"rebuildable", true},
    };
    result["projection_digest"] = engine::tagged_sha256(result.dump());
    return result;
}

}

engine::Json descriptor() {
    return engine::Json{
        {"protocol", engine::descriptor_protocol_v1}, {"module_id", module_id},
        {"engine_id", engine_id}, {"vector_id", vector_id}, {"engine_version", engine_version},
        {"process_protocols", engine::Json::array({engine::process_protocol_v1})},
        {"contract_versions", engine::Json::array({"knowledge/SPEC.md@v1", "knowledge/sclv/SPEC.md@v3"})},
        {"operations", engine::Json::array({
            engine::Json{{"name", "inspect"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "check"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "propose"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "recover"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "project"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "apply"}, {"availability", "disabled"}, {"mutates_canonical", true}},
        })},
        {"limits", engine::Json{
            {"request_bytes", engine::Limits::max_request_bytes}, {"response_bytes", engine::Limits::max_response_bytes},
            {"json_depth", engine::Limits::max_json_depth}, {"json_values", engine::Limits::max_json_values},
            {"path_bytes", engine::Limits::max_path_bytes}, {"snapshot_files", engine::Limits::max_snapshot_files},
            {"snapshot_file_bytes", engine::Limits::max_snapshot_file_bytes},
            {"deadline_ahead_ms", engine::Limits::max_deadline_ahead_ms},
        }},
        {"supported_scopes", engine::Json::array({"user"})}, {"language", "C++26"},
        {"thermal_path", "freezing"}, {"install_state", "installed_undocked"},
        {"default_receptor", nullptr}, {"canonical_apply_enabled", false},
        {"session_mutation_enabled", false}, {"network_listener", false},
    };
}

engine::Json handle_request(const engine::Request& request) {
    if (request.operation == "inspect") return inspect(request.payload);
    if (request.operation == "check") return check(request.payload, request.deadline_unix_ms);
    if (request.operation == "propose") return propose(request.payload, request.deadline_unix_ms);
    if (request.operation == "recover") return recovery(request.payload, request.deadline_unix_ms);
    if (request.operation == "project") return project(request.payload, request.deadline_unix_ms);
    throw engine::Error("operation.unsupported", "operation is reserved or unsupported", 4);
}

}
