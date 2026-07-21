#include "sclv_changelog.hpp"

#include "evidence.hpp"

#include <fstream>
#include <map>
#include <set>
#include <string>
#include <vector>

namespace {

constexpr std::size_t maximum_changelog_bytes = 4U * 1024U * 1024U;

std::string clean_value(std::string value) {
    const auto begin = value.find_first_not_of(" \t\r\n`\"");
    const auto end = value.find_last_not_of(" \t\r\n`\"");
    if (begin == std::string::npos || end == std::string::npos) {
        return {};
    }
    return value.substr(begin, end - begin + 1U);
}

bool top_level_field(const std::string& line, std::string& field, std::string& value) {
    if (!line.starts_with("- ")) {
        return false;
    }
    const auto colon = line.find(':', 2U);
    if (colon == std::string::npos) {
        return false;
    }
    field = line.substr(2U, colon - 2U);
    if (field.empty() || field.front() == '`') {
        return false;
    }
    value = clean_value(line.substr(colon + 1U));
    return true;
}

bool list_item(const std::string& line, std::string& value) {
    if (!line.starts_with("  - ")) {
        return false;
    }
    value = clean_value(line.substr(4U));
    return !value.empty();
}

const std::set<std::string> v3_fields = {
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

void assign_field(SclvRecord& record, const std::string& field, const std::string& value) {
    if (!record.fields.insert(field).second) {
        record.parse_violations.push_back("duplicate_field=" + field);
        return;
    }
    if (field == "record_id") record.record_id = value;
    else if (field == "record_version") {
        record.has_record_version = true;
        try { record.record_version = std::stoi(value); } catch (...) { record.record_version = 0; }
    }
    else if (field == "title") { record.has_title = true; record.title = value; }
    else if (field == "status") { record.has_status = true; record.status = value; }
    else if (field == "date") { record.has_date = true; record.date = value; }
    else if (field == "change_started_at") { record.has_change_started_at = true; record.change_started_at = value; }
    else if (field == "change_completed_at") { record.has_change_completed_at = true; record.change_completed_at = value; }
    else if (field == "recorded_at") { record.has_recorded_at = true; record.recorded_at = value; }
    else if (field == "recording_disposition") { record.has_recording_disposition = true; record.recording_disposition = value; }
    else if (field == "recovery_reason") { record.has_recovery_reason = true; record.recovery_reason = value; }
    else if (field == "change_type") { record.has_change_type = true; record.change_type = value; }
    else if (field == "related_pr") { record.has_related_pr = true; record.related_pr = value; }
    else if (field == "merge_commit") { record.has_merge_commit = true; record.merge_commit = value; }
    else if (field == "change_request_state") { record.has_change_request_state = true; record.change_request_state = value; }
    else if (field == "change_request_provider") { record.has_change_request_provider = true; record.change_request_provider = value; }
    else if (field == "change_request_id") { record.has_change_request_id = true; record.change_request_id = value; }
    else if (field == "change_request_reference") { record.has_change_request_reference = true; record.change_request_reference = value; }
    else if (field == "change_request_absence_reason") { record.has_change_request_absence_reason = true; record.change_request_absence_reason = value; }
    else if (field == "revision_scheme") { record.has_revision_scheme = true; record.revision_scheme = value; }
    else if (field == "revision_value") { record.has_revision_value = true; record.revision_value = value; }
    else if (field == "tree_digest") { record.has_tree_digest = true; record.tree_digest = value; }
    else if (field == "ratification_subject") { record.has_ratification_subject = true; record.ratification_subject = value; }
    else if (field == "ratification_permission") { record.has_ratification_permission = true; record.ratification_permission = value; }
    else if (field == "ratification_method") { record.has_ratification_method = true; record.ratification_method = value; }
    else if (field == "ratification_evidence_reference") { record.has_ratification_evidence_reference = true; record.ratification_evidence_reference = value; }
    else if (field == "ratification_evidence_digest") { record.has_ratification_evidence_digest = true; record.ratification_evidence_digest = value; }
    else if (field == "affected_surfaces") record.has_affected_surfaces = true;
    else if (field == "skvi_references") record.has_skvi_references = true;
    else if (field == "change_summary") { record.has_change_summary = true; record.change_summary = value; }
    else if (field == "relationship_changes") { record.has_relationship_changes = true; record.relationship_changes = value; }
    else if (field == "doctrine_changes") { record.has_doctrine_changes = true; record.doctrine_changes = value; }
    else if (field == "compatibility_consequences") { record.has_compatibility_consequences = true; record.compatibility_consequences = value; }
    else if (field == "publication_consequences") { record.has_publication_consequences = true; record.publication_consequences = value; }
    else if (field == "projection_consequences") { record.has_projection_consequences = true; record.projection_consequences = value; }
    else if (field == "evidence") record.has_evidence = true;
    else if (field == "non_authorizations") record.has_non_authorizations = true;
    else if (field == "notes") { record.has_notes = true; record.notes = value; }
    else record.parse_violations.push_back("unknown_field=" + field);
}

void validate_record(SclvCheckResult& result, const SclvRecord& record) {
    std::vector<std::string> missing;
    const auto require = [&](bool present, const char* field) {
        if (!present) missing.emplace_back(field);
    };
    require(record.has_title, "title");
    require(record.has_status, "status");
    require(record.has_date, "date");
    require(record.has_change_type, "change_type");
    require(record.has_affected_surfaces, "affected_surfaces");
    require(record.has_skvi_references, "skvi_references");
    require(record.has_change_summary, "change_summary");
    require(record.has_relationship_changes, "relationship_changes");
    require(record.has_doctrine_changes, "doctrine_changes");
    require(record.has_compatibility_consequences, "compatibility_consequences");
    require(record.has_publication_consequences, "publication_consequences");
    require(record.has_projection_consequences, "projection_consequences");
    require(record.has_evidence, "evidence");
    require(record.has_non_authorizations, "non_authorizations");
    require(record.has_notes, "notes");

    if (!record.has_record_version || record.record_version <= 2) {
        require(record.has_related_pr, "related_pr");
        require(record.has_merge_commit, "merge_commit");
    }
    if (record.has_record_version && (record.record_version == 2 || record.record_version == 3)) {
        require(record.has_change_started_at, "change_started_at");
        require(record.has_change_completed_at, "change_completed_at");
        require(record.has_recorded_at, "recorded_at");
        require(record.has_recording_disposition, "recording_disposition");
        require(
            record.has_recovery_reason ||
                (record.record_version == 2 && record.recording_disposition != "late_recovery"),
            "recovery_reason");
    }
    if (record.has_record_version && record.record_version == 3) {
        for (const auto& field : v3_fields) {
            if (!record.fields.contains(field)) missing.push_back(field);
        }
        if (record.fields.size() != v3_fields.size()) {
            result.success = false;
            result.messages.push_back(format_evidence(EvidenceCategory::Violation, "sclv.record.v3_field_set", "record_id=" + record.record_id));
        }
    }
    if (!record.parse_violations.empty() && record.record_version == 3) {
        result.success = false;
        for (const auto& detail : record.parse_violations) {
            result.messages.push_back(format_evidence(EvidenceCategory::Violation, "sclv.record.v3_parse", "record_id=" + record.record_id + " " + detail));
        }
    }
    if (missing.empty()) {
        result.messages.push_back(format_evidence(EvidenceCategory::Pass, "sclv.record.shape", "record_id=" + record.record_id));
    } else {
        result.success = false;
        for (const auto& field : missing) {
            result.messages.push_back(format_evidence(EvidenceCategory::Violation, "sclv.record.missing_field", "record_id=" + record.record_id + " field=" + field));
        }
    }
}

}

SclvCheckResult check_sclv_changelog(const std::string& changelog_path) {
    SclvCheckResult result{true, {}, {}};
    std::ifstream input(changelog_path, std::ios::binary);
    if (!input.is_open()) {
        result.success = false;
        result.messages.push_back(format_evidence(EvidenceCategory::Violation, "sclv.changelog.absent", "path=" + changelog_path));
        return result;
    }
    input.seekg(0, std::ios::end);
    const auto size = input.tellg();
    if (size < 0 || static_cast<std::size_t>(size) > maximum_changelog_bytes) {
        result.success = false;
        result.messages.push_back(format_evidence(EvidenceCategory::Violation, "sclv.changelog.resource_limit", "path=" + changelog_path));
        return result;
    }
    input.seekg(0, std::ios::beg);
    result.messages.push_back(format_evidence(EvidenceCategory::Pass, "sclv.changelog.exists", changelog_path + " exists"));

    SclvRecord current;
    bool in_record = false;
    std::string active_list;
    std::string active_block;
    bool found_pr10 = false;
    bool found_pr11 = false;
    std::string line;
    while (std::getline(input, line)) {
        std::string field;
        std::string value;
        if (top_level_field(line, field, value)) {
            if (field == "record_id") {
                if (in_record) {
                    validate_record(result, current);
                    if (current.record_id == "SCLV-PR-010") found_pr10 = true;
                    if (current.record_id == "SCLV-PR-011") found_pr11 = true;
                    result.records.push_back(current);
                }
                current = SclvRecord{};
                in_record = true;
            }
            if (in_record) {
                assign_field(current, field, value);
                active_list = field == "affected_surfaces" || field == "skvi_references" ||
                    field == "evidence" || field == "non_authorizations" ? field : "";
                active_block = field == "change_summary" || field == "relationship_changes" ||
                    field == "doctrine_changes" || field == "compatibility_consequences" ||
                    field == "publication_consequences" || field == "projection_consequences" ||
                    field == "notes" ? field : "";
                if (!active_block.empty() && value != "|") active_block.clear();
            }
            continue;
        }
        std::string item;
        if (in_record && list_item(line, item)) {
            if (active_list == "affected_surfaces") current.affected_surfaces.push_back(item);
            if (active_list == "skvi_references") current.skvi_references.push_back(item);
            if (active_list == "evidence") current.evidence.push_back(item);
            if (active_list == "non_authorizations") current.non_authorizations.push_back(item);
            continue;
        }
        if (in_record && !active_block.empty() && line.starts_with("    ")) {
            std::string* target = nullptr;
            if (active_block == "change_summary") target = &current.change_summary;
            else if (active_block == "relationship_changes") target = &current.relationship_changes;
            else if (active_block == "doctrine_changes") target = &current.doctrine_changes;
            else if (active_block == "compatibility_consequences") target = &current.compatibility_consequences;
            else if (active_block == "publication_consequences") target = &current.publication_consequences;
            else if (active_block == "projection_consequences") target = &current.projection_consequences;
            else if (active_block == "notes") target = &current.notes;
            if (target != nullptr) {
                if (*target == "|") target->clear();
                if (!target->empty()) target->push_back('\n');
                target->append(line.substr(4U));
            }
        }
    }
    if (input.bad()) {
        result.success = false;
        result.messages.push_back(format_evidence(EvidenceCategory::Violation, "sclv.changelog.unreadable", "path=" + changelog_path));
        return result;
    }
    if (in_record) {
        validate_record(result, current);
        if (current.record_id == "SCLV-PR-010") found_pr10 = true;
        if (current.record_id == "SCLV-PR-011") found_pr11 = true;
        result.records.push_back(current);
    }
    if (result.records.empty() || !found_pr10 || !found_pr11) {
        result.success = false;
        result.messages.push_back(format_evidence(EvidenceCategory::Violation, "sclv.record.none", "no SCLV records detected or missing required records"));
    }
    return result;
}
