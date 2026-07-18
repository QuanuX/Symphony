#pragma once
#include <string>
#include <vector>

struct SclvRecord {
    std::string record_id;
    bool has_record_version = false;
    int record_version = 1;
    bool has_title = false;
    bool has_status = false;
    std::string status;
    bool has_date = false;
    bool has_change_started_at = false;
    std::string change_started_at;
    bool has_change_completed_at = false;
    std::string change_completed_at;
    bool has_recorded_at = false;
    std::string recorded_at;
    bool has_recording_disposition = false;
    std::string recording_disposition;
    bool has_recovery_reason = false;
    std::string recovery_reason;
    bool has_change_type = false;
    std::string change_type;
    bool has_related_pr = false;
    std::string related_pr;
    bool has_merge_commit = false;
    std::string merge_commit;
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
