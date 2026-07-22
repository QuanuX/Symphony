#include "sodv.hpp"

#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/limits.hpp"
#include "symphony/knowledge/engine/path.hpp"

#include <algorithm>
#include <array>
#include <chrono>
#include <cctype>
#include <filesystem>
#include <exception>
#include <map>
#include <optional>
#include <set>
#include <sstream>
#include <string>
#include <string_view>
#include <utility>
#include <vector>

namespace symphony::knowledge::sodv {
namespace engine = symphony::knowledge::engine;
namespace fs = std::filesystem;

namespace {

constexpr std::size_t max_records = 4096;
constexpr std::size_t max_units = 128;
constexpr std::size_t max_evidence = 2048;
constexpr std::size_t max_text = 64U * 1024U;
constexpr const char* ledger_path = "knowledge/sodv/RELEASES.md";
constexpr const char* check_protocol = "symphony.sodv.check-result.v1";
constexpr const char* verify_protocol = "symphony.sodv.verify-result.v1";
constexpr const char* recovery_protocol = "symphony.sodv.recovery-result.v1";
constexpr const char* projection_protocol = "symphony.sodv.projection.v1";
constexpr const char* proposal_protocol = "symphony.knowledge.proposal.v1";

const std::vector<std::string> contract_paths = {
    "knowledge/SPEC.md",
    "knowledge/sodv/INTENT.md",
    "knowledge/sodv/MANIFEST.md",
    "knowledge/sodv/SKILL.md",
    "knowledge/sodv/SPEC.md",
    "knowledge/sodv/RELEASES.md",
    "knowledge/sodv/schemas/v1/MANIFEST.md",
    "knowledge/sodv/schemas/v1/release-record-v2.schema.json",
    "knowledge/sodv/schemas/v1/observed-state.schema.json",
    "knowledge/sodv/schemas/v1/check-result.schema.json",
    "knowledge/sodv/schemas/v1/verify-result.schema.json",
    "knowledge/sodv/schemas/v1/proposal-input.schema.json",
    "knowledge/sodv/schemas/v1/recovery-input.schema.json",
    "knowledge/sodv/schemas/v1/recovery-result.schema.json",
    "knowledge/sodv/schemas/v1/projection.schema.json",
    "knowledge/skvi/INDEX.md",
};

struct Unit final {
    std::string unit_id;
    std::string artifact_kind = "go_module";
    std::string coordinate;
    std::string version;
    std::string tag;
    std::string revision_scheme = "git-sha1";
    std::string revision_value;
    std::string source_reference;
    std::optional<std::string> tag_object;
    std::optional<std::string> content_digest;
    std::optional<std::string> metadata_digest;
};

struct Record final {
    std::string id;
    int version = 0;
    std::string type;
    std::string status;
    std::string disposition;
    std::string recorded_at;
    std::string recorded_by;
    std::vector<std::string> subjects;
    std::vector<Unit> units;
    std::string purpose;
    std::vector<std::string> evidence;
    std::vector<std::string> non_authorizations;
    std::string notes;
    std::size_t line = 0;
    std::string digest;
};

struct Finding final {
    std::string severity;
    std::string code;
    std::size_t line;
    std::string detail;
};

struct LedgerState final {
    std::string contents;
    engine::FileDigest ledger_file;
    engine::Snapshot contract_snapshot;
    std::vector<Record> records;
    std::vector<Finding> findings;
    std::size_t passes = 0;
    std::size_t transactions = 0;
};

struct ObservedUnit final {
    std::string coordinate;
    std::string version;
    std::string tag;
    std::optional<std::string> tag_object;
    std::optional<std::string> tag_target_revision;
    std::string public_state;
    std::optional<std::string> content_digest;
    std::optional<std::string> metadata_digest;
    std::string evidence_digest;
};

struct ObservedState final {
    std::string authorization_id;
    std::string observed_at;
    std::string source_reference;
    std::vector<ObservedUnit> units;
    engine::Json json;
};

std::string trim(std::string_view value) {
    std::size_t begin = 0;
    while (begin < value.size() && std::isspace(static_cast<unsigned char>(value[begin])) != 0) {
        ++begin;
    }
    std::size_t end = value.size();
    while (end > begin && std::isspace(static_cast<unsigned char>(value[end - 1U])) != 0) {
        --end;
    }
    return std::string(value.substr(begin, end - begin));
}

std::string clean_value(std::string value) {
    value = trim(value);
    if (value.size() >= 2U && value.front() == '`' && value.back() == '`') {
        value = value.substr(1U, value.size() - 2U);
    } else if (value.size() >= 2U && value.front() == '"' && value.back() == '"') {
        try {
            const auto decoded = engine::Json::parse(value);
            if (decoded.is_string()) { value = decoded.get<std::string>(); }
        } catch (const std::exception&) {
            // Preserve malformed quoted text so the record validator can fail it visibly.
        }
    }
    return value;
}

bool printable_bounded(std::string_view value, std::size_t maximum, bool allow_empty = false) {
    if ((!allow_empty && value.empty()) || value.size() > maximum) {
        return false;
    }
    return std::all_of(value.begin(), value.end(), [](const unsigned char character) {
        return character == '\n' || character == '\t' ||
               (character >= 0x20U && character != 0x7fU);
    });
}

bool safe_token(std::string_view value, std::size_t maximum = engine::Limits::max_token_bytes) {
    if (value.empty() || value.size() > maximum) {
        return false;
    }
    return std::all_of(value.begin(), value.end(), [](const unsigned char character) {
        const bool alphanumeric = (character >= 'a' && character <= 'z') ||
            (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9');
        return alphanumeric || character == '.' || character == '_' || character == ':' || character == '-';
    });
}

bool tagged_digest(std::string_view value) {
    return value.size() == 71U && value.starts_with("sha256:") &&
        std::all_of(value.begin() + 7, value.end(), [](const unsigned char character) {
            return (character >= '0' && character <= '9') || (character >= 'a' && character <= 'f');
        });
}

bool revision(std::string_view value) {
    return (value.size() == 40U || value.size() == 64U) &&
        std::all_of(value.begin(), value.end(), [](const unsigned char character) {
            return (character >= '0' && character <= '9') || (character >= 'a' && character <= 'f');
        });
}

bool strict_utc(std::string_view value) {
    if (value.size() != 20U || value[4] != '-' || value[7] != '-' || value[10] != 'T' ||
        value[13] != ':' || value[16] != ':' || value[19] != 'Z') {
        return false;
    }
    for (const auto index : {0U, 1U, 2U, 3U, 5U, 6U, 8U, 9U, 11U, 12U, 14U, 15U, 17U, 18U}) {
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

bool semantic_version(std::string_view value) {
    if (value.size() < 6U || value.front() != 'v') {
        return false;
    }
    std::size_t dots = 0;
    bool digit_in_component = false;
    bool suffix = false;
    for (std::size_t index = 1; index < value.size(); ++index) {
        const auto character = static_cast<unsigned char>(value[index]);
        if (!suffix && character == '.') {
            if (!digit_in_component || dots >= 2U) { return false; }
            ++dots;
            digit_in_component = false;
        } else if (!suffix && (character == '-' || character == '+')) {
            if (dots != 2U || !digit_in_component || index + 1U == value.size()) { return false; }
            suffix = true;
        } else if (character >= '0' && character <= '9') {
            digit_in_component = true;
        } else if (!suffix || !((character >= 'A' && character <= 'Z') ||
                   (character >= 'a' && character <= 'z') || character == '.' || character == '-')) {
            return false;
        }
    }
    return dots == 2U && digit_in_component;
}

bool safe_tag(std::string_view value) {
    if (!printable_bounded(value, engine::Limits::max_path_bytes) || value.front() == '/' ||
        value.back() == '/' || value.find("//") != std::string_view::npos ||
        value.find('\\') != std::string_view::npos) {
        return false;
    }
    std::size_t begin = 0;
    while (begin < value.size()) {
        const auto end = value.find('/', begin);
        const auto component = value.substr(begin, end == std::string_view::npos ? value.size() - begin : end - begin);
        if (component.empty() || component == "." || component == "..") { return false; }
        if (end == std::string_view::npos) { break; }
        begin = end + 1U;
    }
    return true;
}

bool record_id(std::string_view value) {
    return value.starts_with("SODV-REL-") && value.size() >= 12U && value.size() <= 105U &&
        std::all_of(value.begin() + 9, value.end(), [](const unsigned char character) {
            return (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9') ||
                   character == '.' || character == '_' || character == ':' || character == '-';
        });
}

void require_exact_fields(const engine::Json& object, const std::set<std::string>& fields) {
    if (!object.is_object() || object.size() != fields.size()) {
        throw engine::Error("payload.field_set", "operation payload is incomplete or contains unknown fields", 4);
    }
    for (const auto& [key, value] : object.items()) {
        static_cast<void>(value);
        if (!fields.contains(key)) {
            throw engine::Error("payload.unknown_field", "operation payload contains an unknown field", 4);
        }
    }
}

std::string require_string(const engine::Json& object, const char* field, std::size_t maximum,
                           bool token = false) {
    const auto& value = object.at(field);
    if (!value.is_string()) {
        throw engine::Error("payload.invalid_field", std::string(field) + " must be a string", 4);
    }
    const auto text = value.get<std::string>();
    if ((token && !safe_token(text, maximum)) || (!token && !printable_bounded(text, maximum))) {
        throw engine::Error("payload.invalid_field", std::string(field) + " has invalid syntax", 4);
    }
    return text;
}

std::optional<std::string> optional_string(const engine::Json& object, const char* field,
                                           std::size_t maximum) {
    if (object.at(field).is_null()) { return std::nullopt; }
    return require_string(object, field, maximum);
}

engine::Json file_json(const engine::FileDigest& file) {
    return engine::Json{{"path", file.path}, {"size", file.size}, {"digest", file.digest}};
}

engine::Json snapshot_json(const engine::Snapshot& snapshot) {
    auto files = engine::Json::array();
    for (const auto& file : snapshot.files) { files.push_back(file_json(file)); }
    return engine::Json{{"digest", snapshot.digest}, {"files", std::move(files)}};
}

void add_finding(LedgerState& state, Finding finding) {
    if (finding.severity == "pass") { ++state.passes; return; }
    if (state.findings.size() >= max_evidence) {
        throw engine::Error("sodv.evidence_limit", "SODV evidence limit exceeded", 5);
    }
    state.findings.push_back(std::move(finding));
}

std::size_t finding_count(const LedgerState& state, const std::string& severity) {
    if (severity == "pass") { return state.passes; }
    return static_cast<std::size_t>(std::count_if(state.findings.begin(), state.findings.end(),
        [&](const Finding& finding) { return finding.severity == severity; }));
}

std::vector<std::string> extract_record_ids(std::string_view value) {
    std::vector<std::string> result;
    std::size_t position = 0;
    while ((position = value.find("SODV-REL-", position)) != std::string_view::npos) {
        std::size_t end = position + 9U;
        while (end < value.size()) {
            const auto character = static_cast<unsigned char>(value[end]);
            if (!((character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9') ||
                  character == '.' || character == '_' || character == ':' || character == '-')) { break; }
            ++end;
        }
        const auto id = std::string(value.substr(position, end - position));
        if (record_id(id) && std::find(result.begin(), result.end(), id) == result.end()) { result.push_back(id); }
        position = end;
    }
    return result;
}

engine::Json unit_json(const Unit& unit) {
    return engine::Json{
        {"unit_id", unit.unit_id}, {"artifact_kind", unit.artifact_kind},
        {"coordinate", unit.coordinate}, {"version", unit.version}, {"tag", unit.tag},
        {"revision_scheme", unit.revision_scheme}, {"revision_value", unit.revision_value},
        {"source_reference", unit.source_reference},
        {"tag_object", unit.tag_object ? engine::Json(*unit.tag_object) : engine::Json(nullptr)},
        {"public_digests", engine::Json{
            {"content", unit.content_digest ? engine::Json(*unit.content_digest) : engine::Json(nullptr)},
            {"metadata", unit.metadata_digest ? engine::Json(*unit.metadata_digest) : engine::Json(nullptr)}}},
    };
}

engine::Json record_json_without_digest(const Record& record) {
    auto subjects = engine::Json::array();
    for (const auto& subject : record.subjects) { subjects.push_back(subject); }
    auto units = engine::Json::array();
    for (const auto& unit : record.units) { units.push_back(unit_json(unit)); }
    auto evidence = engine::Json::array();
    for (const auto& item : record.evidence) { evidence.push_back(item); }
    auto non_authorizations = engine::Json::array();
    for (const auto& item : record.non_authorizations) { non_authorizations.push_back(item); }
    return engine::Json{
        {"release_record_id", record.id}, {"record_version", record.version},
        {"record_type", record.type}, {"status", record.status}, {"disposition", record.disposition},
        {"recorded_at", record.recorded_at}, {"recorded_by", record.recorded_by},
        {"subject_record_ids", std::move(subjects)}, {"publication_units", std::move(units)},
        {"purpose", record.purpose}, {"evidence", std::move(evidence)},
        {"non_authorizations", std::move(non_authorizations)}, {"notes", record.notes},
    };
}

engine::Json record_json(const Record& record) {
    auto result = record_json_without_digest(record);
    result["record_digest"] = record.digest;
    result["source_line"] = record.line;
    return result;
}

std::string unit_key(const Unit& unit) { return unit.coordinate + "\n" + unit.version + "\n" + unit.tag; }

std::string field_value(const std::map<std::string, std::string>& fields, const std::string& key) {
    const auto found = fields.find(key);
    return found == fields.end() ? std::string{} : found->second;
}

void append_unique(std::vector<std::string>& values, const std::string& value) {
    if (!value.empty() && std::find(values.begin(), values.end(), value) == values.end()) {
        values.push_back(value);
    }
}

std::vector<Record> parse_ledger(std::string_view contents) {
    struct RawRecord final {
        std::map<std::string, std::string> fields;
        std::map<std::string, std::vector<std::string>> lists;
        std::vector<std::map<std::string, std::string>> units;
        std::size_t line = 0;
    };
    std::vector<RawRecord> raw_records;
    RawRecord* current = nullptr;
    std::map<std::string, std::string>* current_unit = nullptr;
    std::string section;
    std::istringstream input{std::string(contents)};
    std::string line;
    std::size_t line_number = 0;
    while (std::getline(input, line)) {
        ++line_number;
        if (line.size() > max_text) {
            throw engine::Error("sodv.ledger.line_limit", "release ledger line exceeds bound", 5);
        }
        if (line.starts_with("- release_record_id:")) {
            if (raw_records.size() >= max_records) {
                throw engine::Error("sodv.ledger.record_limit", "release record limit exceeded", 5);
            }
            raw_records.push_back(RawRecord{});
            current = &raw_records.back();
            current->line = line_number;
            current->fields["release_record_id"] = clean_value(line.substr(20U));
            current_unit = nullptr;
            section.clear();
            continue;
        }
        if (current == nullptr) { continue; }
        if (line.starts_with("- ")) {
            const auto colon = line.find(':', 2U);
            if (colon == std::string::npos) { continue; }
            const auto key = trim(std::string_view(line).substr(2U, colon - 2U));
            const auto value = clean_value(line.substr(colon + 1U));
            current->fields[key] = value;
            section = value.empty() ? key : std::string{};
            current_unit = nullptr;
            continue;
        }
        if (current_unit != nullptr && line.starts_with("    ") && !line.starts_with("  - ")) {
            const auto continuation = std::string_view(line).substr(4U);
            const auto continuation_colon = continuation.find(':');
            if (continuation_colon != std::string_view::npos) {
                (*current_unit)[trim(continuation.substr(0, continuation_colon))] =
                    clean_value(std::string(continuation.substr(continuation_colon + 1U)));
            }
            continue;
        }
        if (!line.starts_with("  - ")) { continue; }
        const auto item = std::string_view(line).substr(4U);
        const auto colon = item.find(':');
        if ((section == "publication_units" || section == "corrected_publication_units") &&
            colon != std::string_view::npos) {
            if (current->units.size() >= max_units) {
                throw engine::Error("sodv.ledger.unit_limit", "publication unit limit exceeded", 5);
            }
            current->units.emplace_back();
            current_unit = &current->units.back();
            (*current_unit)[trim(item.substr(0, colon))] = clean_value(std::string(item.substr(colon + 1U)));
            continue;
        }
        append_unique(current->lists[section], clean_value(std::string(item)));
    }

    std::vector<Record> records;
    for (const auto& raw : raw_records) {
        Record record;
        record.id = field_value(raw.fields, "release_record_id");
        record.line = raw.line;
        const auto version_text = field_value(raw.fields, "record_version");
        record.version = version_text == "1" ? 1 : (version_text == "2" ? 2 : 0);
        record.type = field_value(raw.fields, "record_type");
        record.status = field_value(raw.fields, "status");
        record.disposition = field_value(raw.fields, "disposition");
        record.recorded_at = field_value(raw.fields, "recorded_at");
        if (record.recorded_at.empty()) {
            for (const auto* field : {"authorized_at", "discovered_at", "completed_at"}) {
                if (!field_value(raw.fields, field).empty()) { record.recorded_at = field_value(raw.fields, field); break; }
            }
        }
        record.recorded_by = field_value(raw.fields, "recorded_by");
        if (record.recorded_by.empty()) {
            for (const auto* field : {"authorized_by", "ratified_by", "completed_by"}) {
                if (!field_value(raw.fields, field).empty()) { record.recorded_by = field_value(raw.fields, field); break; }
            }
        }
        if (record.version == 2) {
            record.subjects = raw.lists.contains("subject_record_ids") ? raw.lists.at("subject_record_ids") : std::vector<std::string>{};
        } else if (record.type == "authorization") {
            record.subjects = {"not_applicable"};
        } else {
            const auto relation = record.type == "authorization_correction" ?
                field_value(raw.fields, "corrects") : field_value(raw.fields, "completes");
            record.subjects = extract_record_ids(relation);
        }
        record.purpose = field_value(raw.fields, "purpose");
        if (record.purpose.empty()) { record.purpose = "Legacy SODV v1 record; consult canonical Markdown for narrative."; }
        record.evidence = raw.lists.contains("evidence") ? raw.lists.at("evidence") : std::vector<std::string>{"legacy_record"};
        record.non_authorizations = raw.lists.contains("non_authorizations") ? raw.lists.at("non_authorizations") : std::vector<std::string>{"not_specified"};
        record.notes = field_value(raw.fields, "notes");
        if (record.notes.empty()) { record.notes = "Legacy SODV v1 normalized by the read-only engine."; }
        for (const auto& fields : raw.units) {
            Unit unit;
            unit.coordinate = field_value(fields, "coordinate");
            if (unit.coordinate.empty()) { unit.coordinate = field_value(fields, "module_path"); }
            unit.version = field_value(fields, "version");
            unit.tag = field_value(fields, "tag");
            unit.revision_value = field_value(fields, "revision_value");
            if (unit.revision_value.empty()) { unit.revision_value = field_value(fields, "source_commit"); }
            unit.revision_scheme = field_value(fields, "revision_scheme");
            if (unit.revision_scheme.empty()) { unit.revision_scheme = unit.revision_value.size() == 64U ? "git-sha256" : "git-sha1"; }
            unit.artifact_kind = field_value(fields, "artifact_kind");
            if (unit.artifact_kind.empty()) { unit.artifact_kind = "go_module"; }
            unit.source_reference = field_value(fields, "source_reference");
            if (unit.source_reference.empty()) { unit.source_reference = field_value(fields, "source_pr"); }
            if (unit.source_reference.empty()) { unit.source_reference = ledger_path; }
            const auto tag_object = field_value(fields, "tag_object");
            if (!tag_object.empty() && tag_object != "null") { unit.tag_object = tag_object; }
            if (record.version == 2) {
                unit.unit_id = field_value(fields, "unit_id");
            }
            if (unit.unit_id.empty()) {
                unit.unit_id = "unit:" + engine::sha256_hex(unit.coordinate + "\n" + unit.version).substr(0U, 24U);
            }
            if (record.type == "authorization_correction") {
                const auto content = field_value(fields, "canonical_go_sum");
                const auto metadata = field_value(fields, "go_mod_sum");
                if (!content.empty()) { unit.content_digest = content; }
                if (!metadata.empty()) { unit.metadata_digest = metadata; }
            } else if (record.type == "completion") {
                const auto content = field_value(fields, "public_go_sum");
                const auto metadata = field_value(fields, "public_go_mod_sum");
                if (!content.empty()) { unit.content_digest = content; }
                if (!metadata.empty()) { unit.metadata_digest = metadata; }
            } else if (record.version == 2) {
                const auto content = field_value(fields, "content");
                const auto metadata = field_value(fields, "metadata");
                if (!content.empty() && content != "null") { unit.content_digest = content; }
                if (!metadata.empty() && metadata != "null") { unit.metadata_digest = metadata; }
            }
            record.units.push_back(std::move(unit));
        }
        record.digest = engine::tagged_sha256(record_json_without_digest(record).dump());
        records.push_back(std::move(record));
    }
    return records;
}

const Record* find_record(const LedgerState& state, const std::string& id) {
    const auto found = std::find_if(state.records.begin(), state.records.end(),
        [&](const Record& record) { return record.id == id; });
    return found == state.records.end() ? nullptr : &*found;
}

const Record* authorization_for(const LedgerState& state, const Record& record) {
    if (record.type == "authorization") { return &record; }
    for (const auto& subject : record.subjects) {
        const auto* candidate = find_record(state, subject);
        if (candidate != nullptr && candidate->type == "authorization") { return candidate; }
    }
    return nullptr;
}

void validate_unit_shape(LedgerState& state, const Record& record, const Unit& unit) {
    const auto failure = [&](const std::string& code, const std::string& detail) {
        add_finding(state, Finding{"violation", code, record.line, detail});
    };
    if (!safe_token(unit.unit_id) || unit.artifact_kind != "go_module" ||
        !printable_bounded(unit.coordinate, 4096U) || !semantic_version(unit.version) ||
        (!unit.tag.empty() && !safe_tag(unit.tag)) || !revision(unit.revision_value) ||
        !printable_bounded(unit.source_reference, 4096U)) {
        failure("sodv.unit.invalid", "publication unit fields violate the bounded v1/v2 contract");
    }
    if ((unit.revision_value.size() == 40U && unit.revision_scheme != "git-sha1") ||
        (unit.revision_value.size() == 64U && unit.revision_scheme != "git-sha256")) {
        failure("sodv.unit.revision_scheme", "revision scheme does not match revision width");
    }
    if (unit.tag_object && !revision(*unit.tag_object)) {
        failure("sodv.unit.tag_object", "tag object is not a lowercase Git object identifier");
    }
    for (const auto& digest : {unit.content_digest, unit.metadata_digest}) {
        if (digest && !printable_bounded(*digest, 512U)) {
            failure("sodv.unit.public_digest", "public digest is invalid or exceeds its bound");
        }
    }
}

void validate_ledger(LedgerState& state) {
    std::set<std::string> ids;
    std::string previous_time;
    for (auto& record : state.records) {
        if (!record_id(record.id) || !ids.insert(record.id).second) {
            add_finding(state, Finding{"violation", "sodv.record.id", record.line, "record ID is invalid or duplicated"});
        }
        if ((record.version != 1 && record.version != 2) ||
            !std::set<std::string>{"authorization", "authorization_correction", "completion", "failure"}.contains(record.type) ||
            !strict_utc(record.recorded_at) || !printable_bounded(record.recorded_by, 4096U) ||
            !safe_token(record.disposition) || record.units.empty() || record.units.size() > max_units) {
            add_finding(state, Finding{"violation", "sodv.record.shape", record.line, "release record header or cardinality is invalid"});
        }
        if (record.status == "pending") {
            add_finding(state, Finding{"violation", "sodv.record.pending_forbidden", record.line, "canonical pending release state is forbidden"});
        }
        const std::map<std::string, std::string> expected_status{{"authorization", "authorized"},
            {"authorization_correction", "canonical"}, {"completion", "completed"}, {"failure", "failed"}};
        if (expected_status.at(record.type) != record.status) {
            add_finding(state, Finding{"violation", "sodv.record.status", record.line, "record type and status are inconsistent"});
        }
        if (!previous_time.empty() && record.recorded_at < previous_time) {
            add_finding(state, Finding{"violation", "sodv.record.time_order", record.line, "record timestamp precedes the prior record"});
        }
        previous_time = record.recorded_at;
        if (record.type == "authorization") {
            if (record.subjects != std::vector<std::string>{"not_applicable"}) {
                add_finding(state, Finding{"violation", "sodv.record.authorization_subject", record.line, "authorization must use the explicit not_applicable subject"});
            }
        } else if (record.subjects.empty()) {
            add_finding(state, Finding{"violation", "sodv.record.subject_missing", record.line, "non-authorization record must identify an earlier subject"});
        }
        std::set<std::string> unit_keys;
        for (auto& unit : record.units) {
            if (unit.tag.empty() && record.type == "authorization_correction") {
                for (const auto& earlier : state.records) {
                    if (earlier.line >= record.line || earlier.type != "authorization") { continue; }
                    const auto match = std::find_if(earlier.units.begin(), earlier.units.end(), [&](const Unit& candidate) {
                        return candidate.coordinate == unit.coordinate && candidate.version == unit.version;
                    });
                    if (match != earlier.units.end()) { unit.tag = match->tag; break; }
                }
            }
            validate_unit_shape(state, record, unit);
            if (!unit_keys.insert(unit_key(unit)).second) {
                add_finding(state, Finding{"violation", "sodv.unit.duplicate", record.line, "publication unit is duplicated within a record"});
            }
        }
        record.digest = engine::tagged_sha256(record_json_without_digest(record).dump());
        add_finding(state, Finding{"pass", "sodv.record.parsed", record.line, "release record parsed within bounds"});
    }

    std::set<std::string> completed_authorizations;
    for (const auto& record : state.records) {
        if (record.type == "authorization") { ++state.transactions; continue; }
        for (const auto& subject : record.subjects) {
            if (subject == "not_applicable") { continue; }
            const auto* earlier = find_record(state, subject);
            if (earlier == nullptr || earlier->line >= record.line) {
                add_finding(state, Finding{"violation", "sodv.record.subject_order", record.line, "subject must identify an earlier canonical record"});
            }
        }
        const auto* authorization = authorization_for(state, record);
        if (authorization == nullptr) {
            add_finding(state, Finding{"violation", "sodv.record.authorization_missing", record.line, "release lineage has no earlier authorization"});
            continue;
        }
        if (record.type == "completion" && !completed_authorizations.insert(authorization->id).second) {
            add_finding(state, Finding{"violation", "sodv.record.completion_duplicate", record.line, "authorization has more than one completion"});
        }
        for (const auto& unit : record.units) {
            const auto match = std::find_if(authorization->units.begin(), authorization->units.end(), [&](const Unit& candidate) {
                return candidate.coordinate == unit.coordinate && candidate.version == unit.version && candidate.tag == unit.tag;
            });
            if (match == authorization->units.end() || match->revision_value != unit.revision_value) {
                add_finding(state, Finding{"violation", "sodv.unit.authorization_mismatch", record.line,
                    "correction/completion unit does not preserve authorization coordinate, version, tag, and revision"});
            }
        }
        if (record.units.size() != authorization->units.size()) {
            add_finding(state, Finding{"violation", "sodv.unit.set_mismatch", record.line, "release lineage changed publication unit cardinality"});
        }
        if (record.type == "completion") {
            const Record* latest_correction = nullptr;
            for (const auto& candidate : state.records) {
                if (candidate.line >= record.line || candidate.type != "authorization_correction") { continue; }
                if (authorization_for(state, candidate) == authorization) { latest_correction = &candidate; }
            }
            if (latest_correction != nullptr) {
                for (const auto& unit : record.units) {
                    const auto corrected = std::find_if(latest_correction->units.begin(), latest_correction->units.end(),
                        [&](const Unit& candidate) { return unit_key(candidate) == unit_key(unit); });
                    if (corrected == latest_correction->units.end() || corrected->tag_object != unit.tag_object ||
                        corrected->content_digest != unit.content_digest || corrected->metadata_digest != unit.metadata_digest) {
                        add_finding(state, Finding{"violation", "sodv.completion.correction_mismatch", record.line,
                            "completion does not preserve the latest correction evidence"});
                    }
                }
            }
        }
    }
    if (state.records.empty()) {
        add_finding(state, Finding{"pass", "sodv.ledger.empty", 0, "release ledger contains no records"});
    }
}

LedgerState analyze_ledger(const fs::path& root, std::int64_t deadline_unix_ms) {
    LedgerState state;
    state.contents = engine::read_regular_file_no_follow(root, ledger_path,
        engine::Limits::max_snapshot_file_bytes, deadline_unix_ms);
    state.ledger_file = engine::FileDigest{ledger_path, static_cast<std::uint64_t>(state.contents.size()),
        engine::tagged_sha256(state.contents)};
    state.contract_snapshot = engine::snapshot_files(root, contract_paths, deadline_unix_ms);
    state.records = parse_ledger(state.contents);
    validate_ledger(state);
    return state;
}

void require_clean(const LedgerState& state) {
    if (finding_count(state, "violation") != 0U) {
        throw engine::Error("sodv.ledger.invalid", "canonical SODV release ledger is invalid", 4);
    }
}

engine::Json evidence_json(const LedgerState& state) {
    auto result = engine::Json::array();
    for (const auto& finding : state.findings) {
        result.push_back(engine::Json{{"severity", finding.severity}, {"code", finding.code},
            {"path", ledger_path}, {"line", finding.line}, {"detail", finding.detail}});
    }
    return result;
}

ObservedState observed_from_json(const engine::Json& value) {
    require_exact_fields(value, {"authorization_record_id", "observed_at", "source_reference", "units"});
    ObservedState observed;
    observed.authorization_id = require_string(value, "authorization_record_id", 105U);
    observed.observed_at = require_string(value, "observed_at", 20U);
    observed.source_reference = require_string(value, "source_reference", 4096U);
    if (!record_id(observed.authorization_id) || !strict_utc(observed.observed_at) ||
        !value.at("units").is_array() || value.at("units").empty() || value.at("units").size() > max_units) {
        throw engine::Error("payload.invalid_observed_state", "observed state header or cardinality is invalid", 4);
    }
    std::set<std::string> keys;
    for (const auto& item : value.at("units")) {
        require_exact_fields(item, {"coordinate", "version", "tag", "tag_object", "tag_target_revision",
            "public_state", "public_content_digest", "public_metadata_digest", "evidence_digest"});
        ObservedUnit unit;
        unit.coordinate = require_string(item, "coordinate", 4096U);
        unit.version = require_string(item, "version", 128U);
        unit.tag = require_string(item, "tag", 4096U);
        unit.tag_object = optional_string(item, "tag_object", 64U);
        unit.tag_target_revision = optional_string(item, "tag_target_revision", 64U);
        unit.public_state = require_string(item, "public_state", 32U, true);
        unit.content_digest = optional_string(item, "public_content_digest", 512U);
        unit.metadata_digest = optional_string(item, "public_metadata_digest", 512U);
        unit.evidence_digest = require_string(item, "evidence_digest", 71U);
        if (!semantic_version(unit.version) || !safe_tag(unit.tag) ||
            (unit.tag_object && !revision(*unit.tag_object)) ||
            (unit.tag_target_revision && !revision(*unit.tag_target_revision)) ||
            !std::set<std::string>{"not_observed", "resolved", "failed"}.contains(unit.public_state) ||
            !tagged_digest(unit.evidence_digest)) {
            throw engine::Error("payload.invalid_observed_unit", "observed publication unit is invalid", 4);
        }
        const auto key = unit.coordinate + "\n" + unit.version + "\n" + unit.tag;
        if (!keys.insert(key).second) {
            throw engine::Error("payload.duplicate_observed_unit", "observed publication unit is duplicated", 4);
        }
        observed.units.push_back(std::move(unit));
    }
    observed.json = value;
    return observed;
}

const Record* latest_correction(const LedgerState& state, const Record& authorization) {
    const Record* result = nullptr;
    for (const auto& record : state.records) {
        if (record.type == "authorization_correction" && authorization_for(state, record) == &authorization) {
            result = &record;
        }
    }
    return result;
}

const Record* canonical_completion(const LedgerState& state, const Record& authorization) {
    for (const auto& record : state.records) {
        if (record.type == "completion" && authorization_for(state, record) == &authorization) { return &record; }
    }
    return nullptr;
}

engine::Json verify_observed(const LedgerState& state, const ObservedState& observed) {
    const auto* authorization = find_record(state, observed.authorization_id);
    if (authorization == nullptr || authorization->type != "authorization") {
        throw engine::Error("sodv.verify.authorization_missing", "observed state does not name a canonical authorization", 4);
    }
    const auto* correction = latest_correction(state, *authorization);
    const auto* completion = canonical_completion(state, *authorization);
    auto unit_results = engine::Json::array();
    bool mismatch = observed.units.size() != authorization->units.size();
    bool any_absent = false;
    bool any_waiting = false;
    for (const auto& authorized : authorization->units) {
        const auto found = std::find_if(observed.units.begin(), observed.units.end(), [&](const ObservedUnit& unit) {
            return unit.coordinate == authorized.coordinate && unit.version == authorized.version && unit.tag == authorized.tag;
        });
        std::string unit_state = "mismatch";
        std::string detail = "authorized publication unit is absent from observed evidence";
        if (found == observed.units.end()) {
            mismatch = true;
        } else if (!found->tag_object && !found->tag_target_revision && found->public_state == "not_observed" &&
                   !found->content_digest && !found->metadata_digest) {
            unit_state = "tag_absent";
            detail = "tag and public artifact were not observed";
            any_absent = true;
        } else if (!found->tag_object || !found->tag_target_revision ||
                   *found->tag_target_revision != authorized.revision_value) {
            mismatch = true;
            detail = "tag object or target revision does not match the authorization";
        } else if (found->public_state != "resolved" || !found->content_digest || !found->metadata_digest) {
            unit_state = "tag_matches_waiting";
            detail = "tag target matches but complete public evidence is not yet resolved";
            any_waiting = true;
        } else {
            const Unit* expected = &authorized;
            if (correction != nullptr) {
                const auto corrected = std::find_if(correction->units.begin(), correction->units.end(),
                    [&](const Unit& unit) { return unit_key(unit) == unit_key(authorized); });
                if (corrected != correction->units.end()) { expected = &*corrected; }
            }
            if (expected->tag_object && *found->tag_object != *expected->tag_object) {
                mismatch = true;
                detail = "observed tag object conflicts with canonical correction/completion evidence";
            } else if (expected->content_digest && *found->content_digest != *expected->content_digest) {
                mismatch = true;
                detail = "observed public content digest conflicts with canonical evidence";
            } else if (expected->metadata_digest && *found->metadata_digest != *expected->metadata_digest) {
                mismatch = true;
                detail = "observed public metadata digest conflicts with canonical evidence";
            } else if (completion != nullptr) {
                const auto completed = std::find_if(completion->units.begin(), completion->units.end(),
                    [&](const Unit& unit) { return unit_key(unit) == unit_key(authorized); });
                if (completed == completion->units.end() || completed->tag_object != found->tag_object ||
                    completed->content_digest != found->content_digest || completed->metadata_digest != found->metadata_digest) {
                    mismatch = true;
                    detail = "observed evidence differs from the canonical completion";
                } else {
                    unit_state = "completed_matches";
                    detail = "observed evidence matches the canonical completion";
                }
            } else {
                unit_state = "completion_ready";
                detail = "caller-supplied evidence satisfies the authorization and is ready for ratified completion review";
            }
        }
        unit_results.push_back(engine::Json{{"coordinate", authorized.coordinate}, {"version", authorized.version},
            {"tag", authorized.tag}, {"state", unit_state}, {"detail", detail}});
    }
    std::string verification_state;
    if (mismatch) { verification_state = "blocked_mismatch"; }
    else if (completion != nullptr && !any_absent && !any_waiting) { verification_state = "verified_completed"; }
    else if (any_waiting) { verification_state = "published_waiting_evidence"; }
    else if (any_absent) { verification_state = "authorized_unpublished"; }
    else { verification_state = "completion_candidate"; }
    engine::Json result{{"protocol", verify_protocol}, {"authorization_record_id", authorization->id},
        {"canonical_completion_record", completion ? engine::Json(completion->id) : engine::Json(nullptr)},
        {"observed_state_digest", engine::tagged_sha256(observed.json.dump())},
        {"verification_state", verification_state}, {"units", std::move(unit_results)},
        {"read_only", true}, {"noncanonical", true}, {"engine_declares_completion", false}};
    result["result_digest"] = engine::tagged_sha256(result.dump());
    return result;
}

void validate_repository_envelope(const engine::Json& payload) {
    const auto& repository = payload.at("repository");
    require_exact_fields(repository, {"repository_id", "revision", "worktree_id", "tree_digest"});
    static_cast<void>(require_string(repository, "repository_id", 256U));
    static_cast<void>(require_string(repository, "worktree_id", 128U, true));
    const auto digest = require_string(repository, "tree_digest", 71U);
    if (!tagged_digest(digest)) { throw engine::Error("payload.invalid_tree_digest", "tree_digest must be tagged SHA-256", 4); }
    const auto& revision_value = repository.at("revision");
    require_exact_fields(revision_value, {"scheme", "value"});
    static_cast<void>(require_string(revision_value, "scheme", 64U, true));
    static_cast<void>(require_string(revision_value, "value", 256U));
    for (const auto* field : {"session_ref", "context_ref"}) {
        const auto& value = payload.at(field);
        if (!value.is_null() && (!value.is_string() || !safe_token(value.get<std::string>()))) {
            throw engine::Error("payload.invalid_reference", std::string(field) + " must be a token or null", 4);
        }
    }
}

std::vector<std::string> string_list(const engine::Json& value, const std::string& name) {
    if (!value.is_array() || value.empty() || value.size() > 1024U) {
        throw engine::Error("payload.invalid_list", name + " must be a nonempty bounded array", 4);
    }
    std::vector<std::string> result;
    std::set<std::string> unique;
    for (const auto& item : value) {
        if (!item.is_string() || !printable_bounded(item.get_ref<const std::string&>(), 4096U) ||
            !unique.insert(item.get<std::string>()).second) {
            throw engine::Error("payload.invalid_list", name + " contains an invalid or duplicate value", 4);
        }
        result.push_back(item.get<std::string>());
    }
    return result;
}

Unit unit_from_json(const engine::Json& value) {
    require_exact_fields(value, {"unit_id", "artifact_kind", "coordinate", "version", "tag",
        "revision_scheme", "revision_value", "source_reference", "tag_object", "public_digests"});
    Unit unit;
    unit.unit_id = require_string(value, "unit_id", 128U, true);
    unit.artifact_kind = require_string(value, "artifact_kind", 32U, true);
    unit.coordinate = require_string(value, "coordinate", 4096U);
    unit.version = require_string(value, "version", 128U);
    unit.tag = require_string(value, "tag", 4096U);
    unit.revision_scheme = require_string(value, "revision_scheme", 32U, true);
    unit.revision_value = require_string(value, "revision_value", 64U);
    unit.source_reference = require_string(value, "source_reference", 4096U);
    unit.tag_object = optional_string(value, "tag_object", 64U);
    const auto& digests = value.at("public_digests");
    require_exact_fields(digests, {"content", "metadata"});
    unit.content_digest = optional_string(digests, "content", 512U);
    unit.metadata_digest = optional_string(digests, "metadata", 512U);
    return unit;
}

Record record_from_json(const engine::Json& value) {
    require_exact_fields(value, {"release_record_id", "record_version", "record_type", "status",
        "disposition", "recorded_at", "recorded_by", "subject_record_ids", "publication_units",
        "purpose", "evidence", "non_authorizations", "notes"});
    Record record;
    record.id = require_string(value, "release_record_id", 105U);
    if (!value.at("record_version").is_number_integer() || value.at("record_version") != 2) {
        throw engine::Error("payload.invalid_record_version", "proposal record_version must be 2", 4);
    }
    record.version = 2;
    record.type = require_string(value, "record_type", 64U, true);
    record.status = require_string(value, "status", 32U, true);
    record.disposition = require_string(value, "disposition", 128U, true);
    record.recorded_at = require_string(value, "recorded_at", 20U);
    record.recorded_by = require_string(value, "recorded_by", 4096U);
    record.subjects = string_list(value.at("subject_record_ids"), "subject_record_ids");
    if (!value.at("publication_units").is_array() || value.at("publication_units").empty() ||
        value.at("publication_units").size() > max_units) {
        throw engine::Error("payload.invalid_units", "publication_units must be a nonempty bounded array", 4);
    }
    for (const auto& item : value.at("publication_units")) { record.units.push_back(unit_from_json(item)); }
    record.purpose = require_string(value, "purpose", max_text);
    record.evidence = string_list(value.at("evidence"), "evidence");
    record.non_authorizations = string_list(value.at("non_authorizations"), "non_authorizations");
    record.notes = require_string(value, "notes", max_text);
    record.digest = engine::tagged_sha256(record_json_without_digest(record).dump());
    return record;
}

std::string render_record(const Record& record) {
    const auto encoded = [](const std::string& value) { return engine::Json(value).dump(); };
    std::ostringstream out;
    out << "\n- release_record_id: " << encoded(record.id) << "\n"
        << "- record_version: `2`\n- record_type: `" << record.type << "`\n"
        << "- status: `" << record.status << "`\n- disposition: `" << record.disposition << "`\n"
        << "- recorded_at: `" << record.recorded_at << "`\n- recorded_by: " << encoded(record.recorded_by) << "\n"
        << "- subject_record_ids:\n";
    for (const auto& subject : record.subjects) { out << "  - " << encoded(subject) << "\n"; }
    out << "- publication_units:\n";
    for (const auto& unit : record.units) {
        out << "  - unit_id: " << encoded(unit.unit_id) << "\n"
            << "    artifact_kind: `" << unit.artifact_kind << "`\n"
            << "    coordinate: " << encoded(unit.coordinate) << "\n"
            << "    version: `" << unit.version << "`\n"
            << "    tag: " << encoded(unit.tag) << "\n"
            << "    revision_scheme: `" << unit.revision_scheme << "`\n"
            << "    revision_value: `" << unit.revision_value << "`\n"
            << "    source_reference: " << encoded(unit.source_reference) << "\n"
            << "    tag_object: " << (unit.tag_object ? "`" + *unit.tag_object + "`" : "null") << "\n"
            << "    content: " << (unit.content_digest ? encoded(*unit.content_digest) : "null") << "\n"
            << "    metadata: " << (unit.metadata_digest ? encoded(*unit.metadata_digest) : "null") << "\n";
    }
    out << "- purpose: " << encoded(record.purpose) << "\n- evidence:\n";
    for (const auto& item : record.evidence) { out << "  - " << encoded(item) << "\n"; }
    out << "- non_authorizations:\n";
    for (const auto& item : record.non_authorizations) { out << "  - " << encoded(item) << "\n"; }
    out << "- notes: " << encoded(record.notes) << "\n";
    return out.str();
}

engine::Json inspect(const engine::Json& payload) {
    require_exact_fields(payload, {});
    return engine::Json{{"descriptor", descriptor()}, {"canonical_ledger", ledger_path},
        {"canonical_contracts", contract_paths}, {"engine_decides_release_truth", false},
        {"caller_supplies_external_observations", true}, {"network_access", false},
        {"canonical_apply_enabled", false}};
}

engine::Json check(const engine::Json& payload, std::int64_t deadline_unix_ms) {
    require_exact_fields(payload, {"expected_ledger_digest"});
    std::optional<std::string> expected;
    if (!payload.at("expected_ledger_digest").is_null()) {
        expected = require_string(payload, "expected_ledger_digest", 71U);
        if (!tagged_digest(*expected)) { throw engine::Error("payload.invalid_expected_digest", "expected ledger digest must be tagged SHA-256", 4); }
    }
    const auto state = analyze_ledger(fs::current_path(), deadline_unix_ms);
    const bool expected_matches = !expected || *expected == state.ledger_file.digest;
    auto evidence = evidence_json(state);
    if (!expected_matches) {
        evidence.push_back(engine::Json{{"severity", "violation"}, {"code", "sodv.ledger.expected_state_mismatch"},
            {"path", ledger_path}, {"line", 0}, {"detail", "caller expected-state digest is stale"}});
    }
    const auto violations = finding_count(state, "violation") + (expected_matches ? 0U : 1U);
    return engine::Json{{"protocol", check_protocol}, {"ledger", file_json(state.ledger_file)},
        {"contract_snapshot", snapshot_json(state.contract_snapshot)}, {"expected_ledger_matches", expected_matches},
        {"records_checked", state.records.size()}, {"transactions_checked", state.transactions},
        {"evidence", std::move(evidence)}, {"summary", engine::Json{{"state", violations == 0U ? "valid" : "invalid"},
            {"passes", state.passes}, {"warnings", finding_count(state, "warning")}, {"violations", violations}}},
        {"read_only", true}, {"canonical_apply_enabled", false}};
}

engine::Json verify(const engine::Json& payload, std::int64_t deadline_unix_ms) {
    const auto state = analyze_ledger(fs::current_path(), deadline_unix_ms);
    require_clean(state);
    return verify_observed(state, observed_from_json(payload));
}

engine::Json propose(const engine::Json& payload, std::int64_t deadline_unix_ms) {
    require_exact_fields(payload, {"repository", "session_ref", "context_ref", "proposal_expires_at",
        "expected_ledger_digest", "record", "observed_state"});
    validate_repository_envelope(payload);
    const auto expires_at = require_string(payload, "proposal_expires_at", 20U);
    const auto expected_digest = require_string(payload, "expected_ledger_digest", 71U);
    if (!strict_utc(expires_at) || !tagged_digest(expected_digest)) {
        throw engine::Error("payload.invalid_proposal_envelope", "proposal expiry or expected digest is invalid", 4);
    }
    auto state = analyze_ledger(fs::current_path(), deadline_unix_ms);
    require_clean(state);
    if (state.ledger_file.digest != expected_digest) {
        throw engine::Error("proposal.expected_state_mismatch", "release ledger expected-state digest is stale", 4);
    }
    auto record = record_from_json(payload.at("record"));
    record.line = state.records.empty() ? 1U : state.records.back().line + 1U;
    if (find_record(state, record.id) != nullptr || (!state.records.empty() && record.recorded_at < state.records.back().recorded_at) ||
        record.recorded_at >= expires_at) {
        throw engine::Error("proposal.record_conflict", "proposed record ID or timestamp conflicts with canonical state", 4);
    }
    auto candidate = state;
    candidate.records.push_back(record);
    candidate.findings.clear();
    candidate.passes = 0;
    candidate.transactions = 0;
    validate_ledger(candidate);
    require_clean(candidate);
    std::optional<ObservedState> observation;
    std::optional<engine::Json> verification;
    if (!payload.at("observed_state").is_null()) {
        observation = observed_from_json(payload.at("observed_state"));
        verification = verify_observed(state, *observation);
    }
    if (record.type == "authorization") {
        if (verification || record.subjects != std::vector<std::string>{"not_applicable"} ||
            std::any_of(record.units.begin(), record.units.end(), [](const Unit& unit) {
                return unit.tag_object || unit.content_digest || unit.metadata_digest;
            })) {
            throw engine::Error("proposal.authorization_invalid", "authorization requires null publication evidence and no observed state", 4);
        }
    } else if (!verification) {
        throw engine::Error("proposal.observed_state_required", "non-authorization proposal requires caller-supplied observed state", 4);
    }
    if (record.type != "authorization") {
        const Record* subject_authorization = nullptr;
        for (const auto& subject : record.subjects) {
            const auto* candidate_record = find_record(state, subject);
            if (candidate_record == nullptr) { continue; }
            const auto* candidate_authorization = authorization_for(state, *candidate_record);
            if (candidate_authorization != nullptr) { subject_authorization = candidate_authorization; break; }
        }
        if (subject_authorization == nullptr || observation->authorization_id != subject_authorization->id) {
            throw engine::Error("proposal.observation_lineage_mismatch",
                "observed state must name the authorization lineage changed by the proposal", 4);
        }
    }
    if (record.type == "completion" && verification->at("verification_state") != "completion_candidate") {
        throw engine::Error("proposal.completion_not_ready", "completion proposal requires exact completion-candidate evidence", 4);
    }
    if (record.type == "completion" || record.type == "authorization_correction") {
        for (const auto& unit : record.units) {
            const auto observed = std::find_if(observation->units.begin(), observation->units.end(),
                [&](const ObservedUnit& candidate_unit) {
                    return candidate_unit.coordinate == unit.coordinate && candidate_unit.version == unit.version &&
                           candidate_unit.tag == unit.tag;
                });
            if (observed == observation->units.end() || !unit.tag_object || !unit.content_digest ||
                !unit.metadata_digest || unit.tag_object != observed->tag_object ||
                unit.content_digest != observed->content_digest || unit.metadata_digest != observed->metadata_digest) {
                throw engine::Error("proposal.publication_evidence_mismatch",
                    "correction/completion record must preserve exact caller-observed publication evidence", 4);
            }
        }
    }
    const auto markdown = render_record(record);
    const auto desired_digest = engine::tagged_sha256(markdown);
    auto read_set = engine::Json::array();
    read_set.push_back(file_json(state.ledger_file));
    for (const auto& file : state.contract_snapshot.files) { read_set.push_back(file_json(file)); }
    auto data = record_json_without_digest(record);
    data["append_markdown"] = markdown;
    if (verification) { data["verification"] = *verification; }
    engine::Json proposal{{"protocol", proposal_protocol}, {"module_id", module_id}, {"engine_id", engine_id},
        {"engine_version", engine_version}, {"vector_id", vector_id},
        {"contract_versions", engine::Json::array({"knowledge/SPEC.md@v1", "knowledge/sodv/SPEC.md@v1",
            "knowledge/sodv/schemas/v1/release-record-v2.schema.json"})}, {"repository", payload.at("repository")},
        {"session_ref", payload.at("session_ref")}, {"context_ref", payload.at("context_ref")},
        {"read_set", std::move(read_set)}, {"write_set", engine::Json::array({engine::Json{{"target_path", ledger_path},
            {"expected_prior_digest", state.ledger_file.digest}, {"desired_change_digest", desired_digest}}})},
        {"operations", engine::Json::array({engine::Json{{"operation_id", "sodv-op:" + engine::sha256_hex(data.dump()).substr(0U, 48U)},
            {"type", "append_release_record"}, {"target_path", ledger_path}, {"expected_state_digest", state.ledger_file.digest},
            {"desired_change_digest", desired_digest}, {"data", std::move(data)}}})},
        {"validation", engine::Json::array({
            engine::Json{{"code", "sodv.ledger.valid"}, {"outcome", "pass"}, {"detail", "current release ledger passed deterministic checks"}},
            engine::Json{{"code", "sodv.record.valid"}, {"outcome", "pass"}, {"detail", "caller-supplied record passed v2 relationship checks"}},
            engine::Json{{"code", "sodv.observation.caller_supplied"}, {"outcome", "pass"}, {"detail", "external state was supplied by the caller; the engine performed no network access"}}})},
        {"authority", engine::Json{{"caller_declared_operation", true}, {"engine_decided_domain_truth", false}, {"ratified", false}}},
        {"created_at", record.recorded_at}, {"expires_at", expires_at}, {"canonical_apply_enabled", false}};
    proposal["proposal_id"] = "sodv-proposal:" + engine::sha256_hex(proposal.dump()).substr(0U, 48U);
    proposal["proposal_digest"] = engine::tagged_sha256(proposal.dump());
    return proposal;
}

engine::Json recover(const engine::Json& payload, std::int64_t deadline_unix_ms) {
    require_exact_fields(payload, {"journal", "journal_digest", "observed_state", "proposal_input", "recovery_reason"});
    static_cast<void>(require_string(payload, "recovery_reason", max_text));
    const auto journal_digest = require_string(payload, "journal_digest", 71U);
    if (!tagged_digest(journal_digest) || journal_digest != engine::tagged_sha256(payload.at("journal").dump())) {
        throw engine::Error("sodv.recover.journal_digest", "journal digest is invalid or stale", 4);
    }
    const auto& journal = payload.at("journal");
    require_exact_fields(journal, {"format_version", "transaction_id", "authorization_record_id", "started_at", "intended_tags", "local_state"});
    if (!journal.at("format_version").is_number_integer() || journal.at("format_version") != 1) {
        throw engine::Error("sodv.recover.journal_version", "unsupported recovery journal version", 4);
    }
    static_cast<void>(require_string(journal, "transaction_id", 128U, true));
    const auto authorization_id = require_string(journal, "authorization_record_id", 105U);
    const auto started_at = require_string(journal, "started_at", 20U);
    const auto local_state = require_string(journal, "local_state", 32U, true);
    if (!record_id(authorization_id) || !strict_utc(started_at) ||
        !std::set<std::string>{"prepared", "tags_published", "awaiting_public", "completion_ready"}.contains(local_state)) {
        throw engine::Error("sodv.recover.journal_shape", "recovery journal is invalid", 4);
    }
    const auto intended_tags = string_list(journal.at("intended_tags"), "intended_tags");
    const auto observed = observed_from_json(payload.at("observed_state"));
    if (observed.authorization_id != authorization_id) {
        throw engine::Error("sodv.recover.authorization_mismatch", "journal and observation name different authorizations", 4);
    }
    auto state = analyze_ledger(fs::current_path(), deadline_unix_ms);
    require_clean(state);
    const auto* authorization = find_record(state, authorization_id);
    if (authorization == nullptr || authorization->type != "authorization") {
        throw engine::Error("sodv.recover.authorization_missing", "journal authorization is not canonical", 4);
    }
    std::set<std::string> expected_tags;
    for (const auto& unit : authorization->units) { expected_tags.insert(unit.tag); }
    if (std::set<std::string>(intended_tags.begin(), intended_tags.end()) != expected_tags) {
        throw engine::Error("sodv.recover.tag_set_mismatch", "journal intended tags differ from the canonical authorization", 4);
    }
    const auto verification = verify_observed(state, observed);
    const auto verification_state = verification.at("verification_state").get<std::string>();
    std::string action;
    bool delete_recommended = false;
    engine::Json proposal = nullptr;
    if (verification_state == "authorized_unpublished") { action = "resume_authorized_publication"; }
    else if (verification_state == "published_waiting_evidence") { action = "await_public_evidence"; }
    else if (verification_state == "completion_candidate") {
        if (payload.at("proposal_input").is_null()) { action = "completion_proposal_required"; }
        else {
            proposal = propose(payload.at("proposal_input"), deadline_unix_ms);
            const auto& data = proposal.at("operations").at(0).at("data");
            bool subject_matches = false;
            if (data.at("subject_record_ids").is_array()) {
                subject_matches = std::any_of(data.at("subject_record_ids").begin(),
                    data.at("subject_record_ids").end(), [&](const engine::Json& subject) {
                        return subject.is_string() && subject == authorization_id;
                    });
            }
            if (data.at("record_type") != "completion" || !subject_matches ||
                data.at("verification").at("authorization_record_id") != authorization_id) {
                throw engine::Error("sodv.recover.proposal_lineage_mismatch",
                    "recovery proposal must complete the journal authorization lineage", 4);
            }
            action = "propose_forward_completion";
        }
    } else if (verification_state == "verified_completed") {
        action = "no_op_completed";
        delete_recommended = true;
    } else { action = "fail_closed_review"; }
    engine::Json result{{"protocol", recovery_protocol}, {"action", action}, {"verification", verification},
        {"proposal", std::move(proposal)}, {"journal_mutated", false}, {"delete_recommended", delete_recommended},
        {"canonical_apply_enabled", false}};
    result["result_digest"] = engine::tagged_sha256(result.dump());
    return result;
}

engine::Json project(const engine::Json& payload, std::int64_t deadline_unix_ms) {
    require_exact_fields(payload, {"format"});
    if (require_string(payload, "format", 16U, true) != "json") {
        throw engine::Error("payload.unsupported_format", "only json projection is implemented", 4);
    }
    const auto state = analyze_ledger(fs::current_path(), deadline_unix_ms);
    require_clean(state);
    auto records = engine::Json::array();
    auto transactions = engine::Json::array();
    for (const auto& record : state.records) {
        records.push_back(record_json(record));
        if (record.type != "authorization") { continue; }
        const auto* correction = latest_correction(state, record);
        const auto* completion = canonical_completion(state, record);
        transactions.push_back(engine::Json{{"authorization_record_id", record.id},
            {"latest_correction_record", correction ? engine::Json(correction->id) : engine::Json(nullptr)},
            {"completion_record", completion ? engine::Json(completion->id) : engine::Json(nullptr)},
            {"state", completion ? "completed" : "authorized"}, {"unit_count", record.units.size()}});
    }
    engine::Json result{{"protocol", projection_protocol}, {"projection_kind", "release-transaction-inventory"},
        {"format", "json"}, {"module_id", module_id}, {"engine_id", engine_id}, {"engine_version", engine_version},
        {"vector_id", vector_id}, {"canonical_ledger", file_json(state.ledger_file)},
        {"contract_snapshot", snapshot_json(state.contract_snapshot)}, {"record_count", state.records.size()},
        {"transaction_count", state.transactions}, {"records", std::move(records)},
        {"transactions", std::move(transactions)}, {"noncanonical", true}, {"rebuildable", true}};
    result["projection_digest"] = engine::tagged_sha256(result.dump());
    return result;
}

}

engine::Json descriptor() {
    return engine::Json{{"protocol", engine::descriptor_protocol_v1}, {"module_id", module_id},
        {"engine_id", engine_id}, {"vector_id", vector_id}, {"engine_version", engine_version},
        {"process_protocols", engine::Json::array({engine::process_protocol_v1})},
        {"contract_versions", engine::Json::array({"knowledge/SPEC.md@v1", "knowledge/sodv/SPEC.md@v1"})},
        {"operations", engine::Json::array({
            engine::Json{{"name", "inspect"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "check"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "verify"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "propose"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "recover"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "project"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "apply"}, {"availability", "disabled"}, {"mutates_canonical", true}},
            engine::Json{{"name", "publish"}, {"availability", "disabled"}, {"mutates_canonical", true}},
            engine::Json{{"name", "tag"}, {"availability", "disabled"}, {"mutates_canonical", true}}})},
        {"limits", engine::Json{{"request_bytes", engine::Limits::max_request_bytes},
            {"response_bytes", engine::Limits::max_response_bytes}, {"json_depth", engine::Limits::max_json_depth},
            {"json_values", engine::Limits::max_json_values}, {"path_bytes", engine::Limits::max_path_bytes},
            {"snapshot_files", engine::Limits::max_snapshot_files},
            {"snapshot_file_bytes", engine::Limits::max_snapshot_file_bytes},
            {"deadline_ahead_ms", engine::Limits::max_deadline_ahead_ms}}},
        {"supported_scopes", engine::Json::array({"user"})}, {"language", "C++26"},
        {"thermal_path", "freezing"}, {"artifact_kinds", engine::Json::array({"go_module"})},
        {"provider_input", "caller_supplied"}, {"network_access", false},
        {"install_state", "installed_undocked"}, {"default_receptor", nullptr},
        {"canonical_apply_enabled", false}, {"session_mutation_enabled", false}, {"network_listener", false}};
}

engine::Json handle_request(const engine::Request& request) {
    if (request.operation == "inspect") { return inspect(request.payload); }
    if (request.operation == "check") { return check(request.payload, request.deadline_unix_ms); }
    if (request.operation == "verify") { return verify(request.payload, request.deadline_unix_ms); }
    if (request.operation == "propose") { return propose(request.payload, request.deadline_unix_ms); }
    if (request.operation == "recover") { return recover(request.payload, request.deadline_unix_ms); }
    if (request.operation == "project") { return project(request.payload, request.deadline_unix_ms); }
    throw engine::Error("operation.unsupported", "operation is reserved or unsupported", 4);
}

}
