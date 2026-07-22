#include "sodv_releases.hpp"

#include "evidence.hpp"

#include <algorithm>
#include <cctype>
#include <filesystem>
#include <fstream>
#include <map>
#include <optional>
#include <set>
#include <sstream>
#include <string>
#include <string_view>
#include <utility>
#include <vector>

namespace fs = std::filesystem;

namespace {

constexpr std::size_t max_file_bytes = 4U * 1024U * 1024U;
constexpr std::size_t max_records = 4096;
constexpr std::size_t max_units = 128;

struct Unit {
    std::string coordinate;
    std::string version;
    std::string tag;
    std::string revision;
    std::optional<std::string> tag_object;
    std::optional<std::string> content;
    std::optional<std::string> metadata;
};

struct Record {
    std::string id;
    std::string version;
    std::string type;
    std::string status;
    std::string recorded_at;
    std::vector<std::string> subjects;
    std::vector<Unit> units;
    std::size_t line = 0;
};

std::string trim(std::string_view value) {
    std::size_t begin = 0;
    while (begin < value.size() && std::isspace(static_cast<unsigned char>(value[begin])) != 0) { ++begin; }
    std::size_t end = value.size();
    while (end > begin && std::isspace(static_cast<unsigned char>(value[end - 1U])) != 0) { --end; }
    return std::string(value.substr(begin, end - begin));
}

std::string clean(std::string value) {
    value = trim(value);
    if (value.size() >= 2U && value.front() == '`' && value.back() == '`') {
        value = value.substr(1U, value.size() - 2U);
    } else if (value.size() >= 2U && value.front() == '"' && value.back() == '"') {
        std::string decoded;
        decoded.reserve(value.size() - 2U);
        for (std::size_t index = 1U; index + 1U < value.size(); ++index) {
            if (value[index] != '\\' || index + 2U >= value.size()) {
                decoded.push_back(value[index]);
                continue;
            }
            const auto escaped = value[++index];
            switch (escaped) {
                case '"': decoded.push_back('"'); break;
                case '\\': decoded.push_back('\\'); break;
                case '/': decoded.push_back('/'); break;
                case 'b': decoded.push_back('\b'); break;
                case 'f': decoded.push_back('\f'); break;
                case 'n': decoded.push_back('\n'); break;
                case 'r': decoded.push_back('\r'); break;
                case 't': decoded.push_back('\t'); break;
                default: decoded.push_back('\\'); decoded.push_back(escaped); break;
            }
        }
        value = std::move(decoded);
    }
    return value;
}

std::string field(const std::map<std::string, std::string>& values, const std::string& name) {
    const auto found = values.find(name);
    return found == values.end() ? std::string{} : found->second;
}

bool record_id(std::string_view value) {
    return value.starts_with("SODV-REL-") && value.size() >= 12U && value.size() <= 105U &&
        std::all_of(value.begin() + 9, value.end(), [](const unsigned char character) {
            return (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9') ||
                   character == '.' || character == '_' || character == ':' || character == '-';
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
        value[13] != ':' || value[16] != ':' || value[19] != 'Z') { return false; }
    for (const auto index : {0U, 1U, 2U, 3U, 5U, 6U, 8U, 9U, 11U, 12U, 14U, 15U, 17U, 18U}) {
        if (value[index] < '0' || value[index] > '9') { return false; }
    }
    return value.substr(5U, 2U) >= "01" && value.substr(5U, 2U) <= "12" &&
           value.substr(8U, 2U) >= "01" && value.substr(8U, 2U) <= "31" &&
           value.substr(11U, 2U) <= "23" && value.substr(14U, 2U) <= "59" && value.substr(17U, 2U) <= "59";
}

bool semantic_version(std::string_view value) {
    if (value.size() < 6U || value.front() != 'v') { return false; }
    std::size_t dots = 0;
    bool digit = false;
    bool suffix = false;
    for (std::size_t index = 1; index < value.size(); ++index) {
        const auto character = static_cast<unsigned char>(value[index]);
        if (!suffix && character == '.') {
            if (!digit || dots >= 2U) { return false; }
            ++dots;
            digit = false;
        } else if (!suffix && (character == '-' || character == '+')) {
            if (dots != 2U || !digit || index + 1U == value.size()) { return false; }
            suffix = true;
        } else if (character >= '0' && character <= '9') {
            digit = true;
        } else if (!suffix || !std::isalnum(character)) {
            if (!suffix || (character != '.' && character != '-')) { return false; }
        }
    }
    return dots == 2U && digit;
}

std::vector<std::string> extract_ids(std::string_view value) {
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

std::string key(const Unit& unit) { return unit.coordinate + "\n" + unit.version; }

std::vector<Record> parse(std::string_view content, SodvReleaseCheckResult& result) {
    struct Raw {
        std::map<std::string, std::string> fields;
        std::map<std::string, std::vector<std::string>> lists;
        std::vector<std::map<std::string, std::string>> units;
        std::size_t line = 0;
    };
    std::vector<Raw> raw;
    Raw* current = nullptr;
    std::map<std::string, std::string>* current_unit = nullptr;
    std::string section;
    std::istringstream input{std::string(content)};
    std::string line;
    std::size_t line_number = 0;
    while (std::getline(input, line)) {
        ++line_number;
        if (line.size() > 65536U) {
            result.success = false;
            result.messages.push_back(format_evidence(EvidenceCategory::Violation,
                "sodv.releases.line_limit", "line=" + std::to_string(line_number)));
            continue;
        }
        if (line.starts_with("- release_record_id:")) {
            if (raw.size() >= max_records) {
                result.success = false;
                result.messages.push_back(format_evidence(EvidenceCategory::Violation,
                    "sodv.releases.record_limit", "max=4096"));
                break;
            }
            raw.push_back(Raw{});
            current = &raw.back();
            current->line = line_number;
            current->fields["release_record_id"] = clean(line.substr(20U));
            current_unit = nullptr;
            section.clear();
            continue;
        }
        if (current == nullptr) { continue; }
        if (line.starts_with("- ")) {
            const auto colon = line.find(':', 2U);
            if (colon == std::string::npos) { continue; }
            const auto name = trim(std::string_view(line).substr(2U, colon - 2U));
            const auto value = clean(line.substr(colon + 1U));
            current->fields[name] = value;
            section = value.empty() ? name : std::string{};
            current_unit = nullptr;
            continue;
        }
        if (current_unit != nullptr && line.starts_with("    ") && !line.starts_with("  - ")) {
            const auto continuation = std::string_view(line).substr(4U);
            const auto colon = continuation.find(':');
            if (colon != std::string_view::npos) {
                (*current_unit)[trim(continuation.substr(0, colon))] = clean(std::string(continuation.substr(colon + 1U)));
            }
            continue;
        }
        if (!line.starts_with("  - ")) { continue; }
        const auto item = std::string_view(line).substr(4U);
        const auto colon = item.find(':');
        if ((section == "publication_units" || section == "corrected_publication_units") &&
            colon != std::string_view::npos) {
            if (current->units.size() >= max_units) {
                result.success = false;
                result.messages.push_back(format_evidence(EvidenceCategory::Violation,
                    "sodv.releases.unit_limit", "line=" + std::to_string(line_number)));
                continue;
            }
            current->units.emplace_back();
            current_unit = &current->units.back();
            (*current_unit)[trim(item.substr(0, colon))] = clean(std::string(item.substr(colon + 1U)));
        } else {
            const auto value = clean(std::string(item));
            if (!value.empty() && std::find(current->lists[section].begin(), current->lists[section].end(), value) ==
                current->lists[section].end()) { current->lists[section].push_back(value); }
        }
    }

    std::vector<Record> records;
    for (const auto& item : raw) {
        Record record;
        record.id = field(item.fields, "release_record_id");
        record.version = field(item.fields, "record_version");
        record.type = field(item.fields, "record_type");
        record.status = field(item.fields, "status");
        record.recorded_at = field(item.fields, "recorded_at");
        if (record.recorded_at.empty()) {
            for (const auto* name : {"authorized_at", "discovered_at", "completed_at"}) {
                if (!field(item.fields, name).empty()) { record.recorded_at = field(item.fields, name); break; }
            }
        }
        if (record.version == "2" && item.lists.contains("subject_record_ids")) {
            record.subjects = item.lists.at("subject_record_ids");
        } else if (record.type == "authorization") {
            record.subjects = {"not_applicable"};
        } else {
            record.subjects = extract_ids(field(item.fields, record.type == "authorization_correction" ? "corrects" : "completes"));
        }
        record.line = item.line;
        for (const auto& values : item.units) {
            Unit unit;
            unit.coordinate = field(values, "coordinate");
            if (unit.coordinate.empty()) { unit.coordinate = field(values, "module_path"); }
            unit.version = field(values, "version");
            unit.tag = field(values, "tag");
            unit.revision = field(values, "revision_value");
            if (unit.revision.empty()) { unit.revision = field(values, "source_commit"); }
            const auto tag_object = field(values, "tag_object");
            if (!tag_object.empty() && tag_object != "null") { unit.tag_object = tag_object; }
            for (const auto* name : {"content", "canonical_go_sum", "public_go_sum"}) {
                const auto value = field(values, name);
                if (!value.empty() && value != "null") { unit.content = value; break; }
            }
            for (const auto* name : {"metadata", "go_mod_sum", "public_go_mod_sum"}) {
                const auto value = field(values, name);
                if (!value.empty() && value != "null") { unit.metadata = value; break; }
            }
            record.units.push_back(std::move(unit));
        }
        records.push_back(std::move(record));
    }
    return records;
}

const Record* find_record(const std::vector<Record>& records, const std::string& id) {
    const auto found = std::find_if(records.begin(), records.end(), [&](const Record& record) { return record.id == id; });
    return found == records.end() ? nullptr : &*found;
}

const Record* authorization_for(const std::vector<Record>& records, const Record& record) {
    if (record.type == "authorization") { return &record; }
    for (const auto& subject : record.subjects) {
        const auto* candidate = find_record(records, subject);
        if (candidate != nullptr && candidate->type == "authorization") { return candidate; }
    }
    return nullptr;
}

const Record* latest_correction_for(const std::vector<Record>& records, const Record& authorization,
                                    const Record& completion) {
    const Record* latest = nullptr;
    for (const auto& candidate : records) {
        if (candidate.type != "authorization_correction" || candidate.line >= completion.line) { continue; }
        const auto* candidate_authorization = authorization_for(records, candidate);
        if (candidate_authorization != nullptr && candidate_authorization->id == authorization.id &&
            (latest == nullptr || candidate.line > latest->line)) {
            latest = &candidate;
        }
    }
    return latest;
}

void violation(SodvReleaseCheckResult& result, const std::string& code,
               const Record& record, const std::string& detail) {
    result.success = false;
    result.messages.push_back(format_evidence(EvidenceCategory::Violation, code,
        "record_id=" + (record.id.empty() ? std::string("unavailable") : record.id) +
        " line=" + std::to_string(record.line) + " detail=" + detail));
}

}

SodvReleaseCheckResult check_sodv_releases(const std::string& repo_root) {
    SodvReleaseCheckResult result{true, {}, 0, 0};
    const fs::path root(repo_root);
    const auto ledger = root / "knowledge/sodv/RELEASES.md";
    std::error_code error;
    const auto status = fs::symlink_status(ledger, error);
    if (error || !fs::is_regular_file(status) || fs::is_symlink(status)) {
        if (!fs::exists(root / "go.work")) {
            result.messages.push_back(format_evidence(EvidenceCategory::Pass,
                "sodv.releases.fixture_not_applicable", "legacy minimal validator fixture"));
            return result;
        }
        result.success = false;
        result.messages.push_back(format_evidence(EvidenceCategory::Violation,
            "sodv.releases.unreadable", "path=knowledge/sodv/RELEASES.md"));
        return result;
    }
    const auto size = fs::file_size(ledger, error);
    if (error || size > max_file_bytes) {
        result.success = false;
        result.messages.push_back(format_evidence(EvidenceCategory::Violation,
            "sodv.releases.size", "path=knowledge/sodv/RELEASES.md"));
        return result;
    }
    std::ifstream input(ledger, std::ios::binary);
    if (!input.good()) {
        result.success = false;
        result.messages.push_back(format_evidence(EvidenceCategory::Violation,
            "sodv.releases.unreadable", "path=knowledge/sodv/RELEASES.md"));
        return result;
    }
    const std::string content((std::istreambuf_iterator<char>(input)), std::istreambuf_iterator<char>());
    auto records = parse(content, result);
    result.records_checked = records.size();
    std::set<std::string> ids;
    std::set<std::string> completed;
    std::string previous_time;
    for (auto& record : records) {
        if (!record_id(record.id) || !ids.insert(record.id).second) {
            violation(result, "sodv.releases.record_id", record, "invalid_or_duplicate");
        }
        if ((record.version != "1" && record.version != "2") ||
            !std::set<std::string>{"authorization", "authorization_correction", "completion", "failure"}.contains(record.type) ||
            !strict_utc(record.recorded_at) || record.units.empty() || record.units.size() > max_units) {
            violation(result, "sodv.releases.record_shape", record, "header_or_cardinality_invalid");
        }
        const std::map<std::string, std::string> expected{{"authorization", "authorized"},
            {"authorization_correction", "canonical"}, {"completion", "completed"}, {"failure", "failed"}};
        if (expected.contains(record.type) && expected.at(record.type) != record.status) {
            violation(result, "sodv.releases.status", record, "type_status_mismatch");
        }
        if (record.status == "pending") {
            violation(result, "sodv.releases.pending_forbidden", record, "canonical_pending_state");
        }
        if (!previous_time.empty() && record.recorded_at < previous_time) {
            violation(result, "sodv.releases.time_order", record, "timestamp_precedes_prior_record");
        }
        previous_time = record.recorded_at;
        if (record.type == "authorization") {
            ++result.transactions_checked;
            if (record.subjects != std::vector<std::string>{"not_applicable"}) {
                violation(result, "sodv.releases.authorization_subject", record, "not_applicable_required");
            }
        } else if (record.subjects.empty()) {
            violation(result, "sodv.releases.subject_missing", record, "earlier_subject_required");
        }
        std::set<std::string> unit_keys;
        for (auto& unit : record.units) {
            if (unit.tag.empty() && record.type == "authorization_correction") {
                for (const auto& earlier : records) {
                    if (earlier.line >= record.line || earlier.type != "authorization") { continue; }
                    const auto found = std::find_if(earlier.units.begin(), earlier.units.end(),
                        [&](const Unit& candidate) { return key(candidate) == key(unit); });
                    if (found != earlier.units.end()) { unit.tag = found->tag; break; }
                }
            }
            if (unit.coordinate.empty() || !semantic_version(unit.version) || unit.tag.empty() || !revision(unit.revision) ||
                (unit.tag_object && !revision(*unit.tag_object))) {
                violation(result, "sodv.releases.unit_shape", record, "publication_unit_invalid");
            }
            if (!unit_keys.insert(key(unit) + "\n" + unit.tag).second) {
                violation(result, "sodv.releases.unit_duplicate", record, "duplicate_publication_unit");
            }
        }
        result.messages.push_back(format_evidence(EvidenceCategory::Pass,
            "sodv.releases.record_checked", "record_id=" + record.id + " line=" + std::to_string(record.line)));
    }
    for (const auto& record : records) {
        if (record.type == "authorization") { continue; }
        for (const auto& subject : record.subjects) {
            const auto* earlier = find_record(records, subject);
            if (earlier == nullptr || earlier->line >= record.line) {
                violation(result, "sodv.releases.subject_order", record, "subject_not_earlier");
            }
        }
        const auto* authorization = authorization_for(records, record);
        if (authorization == nullptr) {
            violation(result, "sodv.releases.authorization_missing", record, "lineage_has_no_authorization");
            continue;
        }
        if (record.type == "completion" && !completed.insert(authorization->id).second) {
            violation(result, "sodv.releases.completion_duplicate", record, "authorization_already_completed");
        }
        const Record* expected_record = authorization;
        if (record.type == "completion") {
            if (const auto* correction = latest_correction_for(records, *authorization, record); correction != nullptr) {
                expected_record = correction;
            }
        }
        if (record.units.size() != expected_record->units.size()) {
            violation(result, "sodv.releases.unit_set", record, "unit_cardinality_changed");
        }
        for (const auto& unit : record.units) {
            const auto found = std::find_if(expected_record->units.begin(), expected_record->units.end(),
                [&](const Unit& candidate) {
                    return candidate.coordinate == unit.coordinate && candidate.version == unit.version &&
                           candidate.tag == unit.tag && candidate.revision == unit.revision &&
                           (record.type != "completion" || (candidate.tag_object == unit.tag_object &&
                            candidate.content == unit.content && candidate.metadata == unit.metadata));
                });
            if (found == expected_record->units.end()) {
                violation(result, "sodv.releases.authorization_mismatch", record,
                    record.type == "completion" ? "authorized_or_corrected_publication_unit_changed" :
                    "coordinate_version_tag_or_revision_changed");
            }
        }
    }
    result.messages.push_back(format_evidence(EvidenceCategory::Pass,
        "sodv.releases.scan_complete", "records=" + std::to_string(result.records_checked) +
        " transactions=" + std::to_string(result.transactions_checked) +
        " violations=" + std::to_string(result.success ? 0 : 1)));
    return result;
}
