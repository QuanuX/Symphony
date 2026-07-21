#pragma once

#include <set>
#include <string>
#include <vector>

struct SclvRecord {
    std::string record_id;
    bool has_record_version = false;
    int record_version = 1;
    std::set<std::string> fields;
    std::vector<std::string> parse_violations;

    bool has_title = false;
    std::string title;
    bool has_status = false;
    std::string status;
    bool has_date = false;
    std::string date;
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

    bool has_change_request_state = false;
    std::string change_request_state;
    bool has_change_request_provider = false;
    std::string change_request_provider;
    bool has_change_request_id = false;
    std::string change_request_id;
    bool has_change_request_reference = false;
    std::string change_request_reference;
    bool has_change_request_absence_reason = false;
    std::string change_request_absence_reason;
    bool has_revision_scheme = false;
    std::string revision_scheme;
    bool has_revision_value = false;
    std::string revision_value;
    bool has_tree_digest = false;
    std::string tree_digest;
    bool has_ratification_subject = false;
    std::string ratification_subject;
    bool has_ratification_permission = false;
    std::string ratification_permission;
    bool has_ratification_method = false;
    std::string ratification_method;
    bool has_ratification_evidence_reference = false;
    std::string ratification_evidence_reference;
    bool has_ratification_evidence_digest = false;
    std::string ratification_evidence_digest;

    bool has_affected_surfaces = false;
    bool has_skvi_references = false;
    bool has_change_summary = false;
    std::string change_summary;
    bool has_relationship_changes = false;
    std::string relationship_changes;
    bool has_doctrine_changes = false;
    std::string doctrine_changes;
    bool has_compatibility_consequences = false;
    std::string compatibility_consequences;
    bool has_publication_consequences = false;
    std::string publication_consequences;
    bool has_projection_consequences = false;
    std::string projection_consequences;
    bool has_evidence = false;
    std::vector<std::string> evidence;
    bool has_non_authorizations = false;
    std::vector<std::string> non_authorizations;
    bool has_notes = false;
    std::string notes;
    std::vector<std::string> affected_surfaces;
    std::vector<std::string> skvi_references;
};

struct SclvCheckResult {
    bool success;
    std::vector<std::string> messages;
    std::vector<SclvRecord> records;
};

SclvCheckResult check_sclv_changelog(const std::string& changelog_path);
