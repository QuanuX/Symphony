#include "sclv_changelog.hpp"
#include <fstream>
#include <iostream>
#include <sstream>

static std::string trim_list_prefix(const std::string& line) {
    size_t start = 0;
    while (start < line.size() && (line[start] == ' ' || line[start] == '\t' || line[start] == '-')) {
        start++;
    }
    return line.substr(start);
}

static std::string extract_value(const std::string& line, const std::string& prefix) {
    std::string trimmed = trim_list_prefix(line);
    if (trimmed.find(prefix) == 0) {
        std::string val = trimmed.substr(prefix.size());
        size_t start = val.find_first_not_of(" \t\r\n`\"");
        size_t end = val.find_last_not_of(" \t\r\n`\"");
        if (start != std::string::npos && end != std::string::npos) {
            return val.substr(start, end - start + 1);
        }
    }
    return "";
}

SclvCheckResult check_sclv_changelog(const std::string& changelog_path) {
    SclvCheckResult result;
    result.success = true;

    std::ifstream file(changelog_path);
    if (!file.is_open()) {
        result.success = false;
        result.messages.push_back("evidence violation sclv.changelog.absent path=" + changelog_path);
        return result;
    }

    result.messages.push_back("evidence pass sclv.changelog.exists " + changelog_path + " exists");

    std::string line;
    SclvRecord current_record;
    bool in_record = false;
    int record_count = 0;
    bool found_pr10 = false;
    bool found_pr11 = false;

    auto validate_record = [&](const SclvRecord& rec) {
        std::vector<std::string> missing;
        if (!rec.has_title) missing.push_back("title");
        if (!rec.has_status) missing.push_back("status");
        if (!rec.has_date) missing.push_back("date");
        if (!rec.has_change_type) missing.push_back("change_type");
        if (!rec.has_related_pr) missing.push_back("related_pr");
        if (!rec.has_merge_commit) missing.push_back("merge_commit");
        if (!rec.has_affected_surfaces) missing.push_back("affected_surfaces");
        if (!rec.has_skvi_references) missing.push_back("skvi_references");
        if (!rec.has_change_summary) missing.push_back("change_summary");
        if (!rec.has_relationship_changes) missing.push_back("relationship_changes");
        if (!rec.has_doctrine_changes) missing.push_back("doctrine_changes");
        if (!rec.has_compatibility_consequences) missing.push_back("compatibility_consequences");
        if (!rec.has_publication_consequences) missing.push_back("publication_consequences");
        if (!rec.has_projection_consequences) missing.push_back("projection_consequences");
        if (!rec.has_evidence) missing.push_back("evidence");
        if (!rec.has_non_authorizations) missing.push_back("non_authorizations");
        if (!rec.has_notes) missing.push_back("notes");

        if (missing.empty()) {
            result.messages.push_back("evidence pass sclv.record.shape record_id=" + rec.record_id);
        } else {
            result.success = false;
            for (const auto& m : missing) {
                result.messages.push_back("evidence violation sclv.record.missing_field record_id=" + rec.record_id + " field=" + m);
            }
        }

        if (rec.record_id == "SCLV-PR-010") found_pr10 = true;
        if (rec.record_id == "SCLV-PR-011") found_pr11 = true;

        result.records.push_back(rec);
    };

    std::string active_list = "";
    while (std::getline(file, line)) {
        std::string trimmed = trim_list_prefix(line);
        if (trimmed.find("record_id:") == 0) {
            if (in_record) {
                validate_record(current_record);
            }
            in_record = true;
            record_count++;
            current_record = SclvRecord();
            current_record.record_id = extract_value(line, "record_id:");
        } else if (in_record) {
            std::string current_key = "";
            if (trimmed.find("title:") == 0) { current_record.has_title = true; active_list = ""; }
            else if (trimmed.find("status:") == 0) { current_record.has_status = true; active_list = ""; }
            else if (trimmed.find("date:") == 0) { current_record.has_date = true; active_list = ""; }
            else if (trimmed.find("change_type:") == 0) { current_record.has_change_type = true; active_list = ""; }
            else if (trimmed.find("related_pr:") == 0) { current_record.has_related_pr = true; active_list = ""; }
            else if (trimmed.find("merge_commit:") == 0) { current_record.has_merge_commit = true; active_list = ""; }
            else if (trimmed.find("affected_surfaces:") == 0) {
                current_record.has_affected_surfaces = true;
                active_list = "affected_surfaces";
                std::string val = extract_value(line, "affected_surfaces:");
                if (!val.empty() && val != "none") current_record.affected_surfaces.push_back(val);
            }
            else if (trimmed.find("skvi_references:") == 0) {
                current_record.has_skvi_references = true;
                active_list = "skvi_references";
                std::string val = extract_value(line, "skvi_references:");
                if (!val.empty() && val != "none") current_record.skvi_references.push_back(val);
            }
            else if (trimmed.find("change_summary:") == 0) { current_record.has_change_summary = true; active_list = ""; }
            else if (trimmed.find("relationship_changes:") == 0) { current_record.has_relationship_changes = true; active_list = ""; }
            else if (trimmed.find("doctrine_changes:") == 0) { current_record.has_doctrine_changes = true; active_list = ""; }
            else if (trimmed.find("compatibility_consequences:") == 0) { current_record.has_compatibility_consequences = true; active_list = ""; }
            else if (trimmed.find("publication_consequences:") == 0) { current_record.has_publication_consequences = true; active_list = ""; }
            else if (trimmed.find("projection_consequences:") == 0) { current_record.has_projection_consequences = true; active_list = ""; }
            else if (trimmed.find("evidence:") == 0) { current_record.has_evidence = true; active_list = ""; }
            else if (trimmed.find("non_authorizations:") == 0) { current_record.has_non_authorizations = true; active_list = ""; }
            else if (trimmed.find("notes:") == 0) { current_record.has_notes = true; active_list = ""; }
            else if (active_list == "affected_surfaces" || active_list == "skvi_references") {
                // Not a new field, could be a list item
                if (line.find("-") != std::string::npos || line.find("`") != std::string::npos) {
                    size_t start = trimmed.find_first_not_of(" \t\r\n`\"");
                    size_t end = trimmed.find_last_not_of(" \t\r\n`\"");
                    if (start != std::string::npos && end != std::string::npos) {
                        std::string val = trimmed.substr(start, end - start + 1);
                        if (!val.empty() && val != "none") {
                            if (active_list == "affected_surfaces") current_record.affected_surfaces.push_back(val);
                            if (active_list == "skvi_references") current_record.skvi_references.push_back(val);
                        }
                    }
                }
            }
        }
    }

    if (in_record) {
        validate_record(current_record);
    }

    if (record_count == 0 || !found_pr10 || !found_pr11) {
        result.success = false;
        result.messages.push_back("evidence violation sclv.record.none no SCLV records detected or missing required records");
    }

    return result;
}
