#pragma once
#include <string>
#include <vector>

struct SclvRecord {
    std::string record_id;
    bool has_title = false;
    bool has_status = false;
    bool has_date = false;
    bool has_change_type = false;
    bool has_related_pr = false;
    bool has_merge_commit = false;
    bool has_affected_surfaces = false;
    bool has_skvi_references = false;
    bool has_change_summary = false;
    bool has_relationship_changes = false;
    bool has_doctrine_changes = false;
    bool has_compatibility_consequences = false;
    bool has_publication_consequences = false;
    bool has_projection_consequences = false;
    bool has_evidence = false;
    bool has_non_authorizations = false;
    bool has_notes = false;
    std::vector<std::string> affected_surfaces;
    std::vector<std::string> skvi_references;
};

struct SclvCheckResult {
    bool success;
    std::vector<std::string> messages;
    std::vector<SclvRecord> records;
};

SclvCheckResult check_sclv_changelog(const std::string& changelog_path);
