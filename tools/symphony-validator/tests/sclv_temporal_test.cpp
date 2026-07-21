#include "sclv_changelog.hpp"
#include "sclv_ledger.hpp"
#include "sclv_shape.hpp"
#include <filesystem>
#include <fstream>
#include <iostream>
#include <string>

namespace fs = std::filesystem;

static bool contains(const std::vector<std::string>& messages, const std::string& needle) {
    for (const auto& message : messages) {
        if (message.find(needle) != std::string::npos) {
            return true;
        }
    }
    return false;
}

static SclvRecord record(const std::string& id, const std::string& pr, const std::string& commit) {
    SclvRecord value;
    value.record_id = id;
    value.has_related_pr = true;
    value.related_pr = pr;
    value.has_merge_commit = true;
    value.merge_commit = commit;
    return value;
}

static SclvRecord version_two_record(const std::string& id, const std::string& pr, const std::string& commit) {
    SclvRecord value = record(id, pr, commit);
    value.has_record_version = true;
    value.record_version = 2;
    value.has_change_started_at = true;
    value.change_started_at = "2026-07-18T01:00:00Z";
    value.has_change_completed_at = true;
    value.change_completed_at = "2026-07-18T02:00:00Z";
    value.has_recorded_at = true;
    value.recorded_at = "2026-07-18T03:00:00Z";
    value.has_recording_disposition = true;
    value.recording_disposition = "post_merge";
    return value;
}

static SclvRecord version_three_record(const std::string& id, const std::string& revision) {
    SclvRecord value;
    value.record_id = id;
    value.has_record_version = true;
    value.record_version = 3;
    value.has_change_started_at = true;
    value.change_started_at = "2026-07-21T01:00:00Z";
    value.has_change_completed_at = true;
    value.change_completed_at = "2026-07-21T02:00:00Z";
    value.has_recorded_at = true;
    value.recorded_at = "2026-07-21T03:00:00Z";
    value.has_date = true;
    value.date = "2026-07-21";
    value.has_recording_disposition = true;
    value.recording_disposition = "post_merge";
    value.has_recovery_reason = true;
    value.recovery_reason = "not_applicable";
    value.has_change_request_state = true;
    value.change_request_state = "not_applicable";
    value.has_change_request_provider = true;
    value.change_request_provider = "not_applicable";
    value.has_change_request_id = true;
    value.change_request_id = "not_applicable";
    value.has_change_request_reference = true;
    value.change_request_reference = "not_applicable";
    value.has_change_request_absence_reason = true;
    value.change_request_absence_reason = "local air-gapped change";
    value.has_revision_scheme = true;
    value.revision_scheme = "git-sha1";
    value.has_revision_value = true;
    value.revision_value = revision;
    value.has_tree_digest = true;
    value.tree_digest = "sha256:1111111111111111111111111111111111111111111111111111111111111111";
    value.has_title = true;
    value.title = "provider neutral change";
    value.has_status = true;
    value.status = "canonical";
    value.has_change_type = true;
    value.change_type = "tooling_change";
    value.has_ratification_subject = true;
    value.ratification_subject = "repository-owner";
    value.has_ratification_permission = true;
    value.ratification_permission = "repository-transition-owner";
    value.has_ratification_method = true;
    value.ratification_method = "airgap-declaration";
    value.has_ratification_evidence_reference = true;
    value.ratification_evidence_reference = "local:ratification-record";
    value.has_ratification_evidence_digest = true;
    value.ratification_evidence_digest = "sha256:2222222222222222222222222222222222222222222222222222222222222222";
    value.has_affected_surfaces = true;
    value.affected_surfaces = {"README.md"};
    value.has_skvi_references = true;
    value.skvi_references = {"README.md"};
    value.has_change_summary = true;
    value.change_summary = "summary";
    value.has_relationship_changes = true;
    value.relationship_changes = "none";
    value.has_doctrine_changes = true;
    value.doctrine_changes = "none";
    value.has_compatibility_consequences = true;
    value.compatibility_consequences = "none";
    value.has_publication_consequences = true;
    value.publication_consequences = "none";
    value.has_projection_consequences = true;
    value.projection_consequences = "none";
    value.has_evidence = true;
    value.evidence = {"fixture evidence"};
    value.has_non_authorizations = true;
    value.non_authorizations = {"canonical mutation"};
    value.has_notes = true;
    value.notes = "notes";
    return value;
}

static int fail(const std::string& message) {
    std::cerr << "failure: " << message << '\n';
    return 1;
}

int main() {
    SclvCheckResult sparse{true, {}, {
        record("SCLV-PR-010", "https://github.com/QuanuX/Symphony/pull/10", "1111111111111111111111111111111111111111"),
        record("SCLV-PR-033", "https://github.com/QuanuX/Symphony/pull/33", "2222222222222222222222222222222222222222"),
    }};
    auto sparse_result = check_sclv_ledger_continuity(sparse);
    if (!sparse_result.success || !contains(sparse_result.messages, "sclv_ledger.sparse_pr_namespace") ||
        contains(sparse_result.messages, "sclv_ledger.record_gap")) {
        return fail("sparse PR identifiers must be valid without gap evidence");
    }

    SclvCheckResult valid{true, {}, {
        version_two_record("SCLV-PR-064", "https://github.com/QuanuX/Symphony/pull/64", "3333333333333333333333333333333333333333"),
    }};
    auto valid_result = check_sclv_ledger_continuity(valid);
    if (!valid_result.success || !contains(valid_result.messages, "sclv_ledger.temporal_order_valid")) {
        return fail("valid v2 temporal record was rejected");
    }

    auto reversed_record = version_two_record("SCLV-PR-065", "https://github.com/QuanuX/Symphony/pull/65", "4444444444444444444444444444444444444444");
    reversed_record.change_completed_at = "2026-07-18T00:30:00Z";
    SclvCheckResult reversed{true, {}, {reversed_record}};
    auto reversed_result = check_sclv_ledger_continuity(reversed);
    if (reversed_result.success || !contains(reversed_result.messages, "sclv_ledger.temporal_order_invalid")) {
        return fail("reversed temporal record was accepted");
    }

    auto recovery_record = version_two_record("SCLV-PR-066", "https://github.com/QuanuX/Symphony/pull/66", "5555555555555555555555555555555555555555");
    recovery_record.recording_disposition = "late_recovery";
    SclvCheckResult recovery{true, {}, {recovery_record}};
    auto recovery_result = check_sclv_ledger_continuity(recovery);
    if (recovery_result.success || !contains(recovery_result.messages, "sclv_ledger.recovery_reason_absent")) {
        return fail("late recovery without a reason was accepted");
    }

    auto first = version_two_record("SCLV-PR-067", "https://github.com/QuanuX/Symphony/pull/67", "6666666666666666666666666666666666666666");
    auto second = version_two_record("SCLV-PR-068", "https://github.com/QuanuX/Symphony/pull/68", "7777777777777777777777777777777777777777");
    first.recorded_at = "2026-07-18T05:00:00Z";
    second.change_started_at = "2026-07-18T03:30:00Z";
    second.change_completed_at = "2026-07-18T04:00:00Z";
    second.recorded_at = "2026-07-18T04:30:00Z";
    SclvCheckResult nonmonotonic{true, {}, {first, second}};
    auto nonmonotonic_result = check_sclv_ledger_continuity(nonmonotonic);
    if (nonmonotonic_result.success || !contains(nonmonotonic_result.messages, "sclv_ledger.recording_order_invalid")) {
        return fail("nonmonotonic file recording order was accepted");
    }

    auto v3 = version_three_record(
        "SCLV-CHG-20260721-example", "8888888888888888888888888888888888888888");
    SclvCheckResult valid_v3{true, {}, {v3}};
    auto valid_v3_result = check_sclv_ledger_continuity(valid_v3);
    if (!valid_v3_result.success || !contains(valid_v3_result.messages, "sclv_ledger.revision_unique")) {
        return fail("provider-neutral v3 record was rejected");
    }
    auto valid_v3_shape = check_sclv_shapes(valid_v3);
    if (!valid_v3_shape.success || !contains(valid_v3_shape.messages, "sclv.v3.ratification_bound")) {
        return fail("provider-neutral v3 shape was rejected");
    }

    auto invalid_v3 = v3;
    invalid_v3.change_request_provider = "github";
    SclvCheckResult invalid_v3_shape_input{true, {}, {invalid_v3}};
    auto invalid_v3_shape = check_sclv_shapes(invalid_v3_shape_input);
    if (invalid_v3_shape.success || !contains(invalid_v3_shape.messages, "sclv.v3.change_request_invalid")) {
        return fail("inconsistent v3 change-request absence was accepted");
    }

    auto invalid_v3_content = v3;
    invalid_v3_content.evidence.clear();
    SclvCheckResult invalid_v3_content_input{true, {}, {invalid_v3_content}};
    auto invalid_v3_content_result = check_sclv_shapes(invalid_v3_content_input);
    if (invalid_v3_content_result.success ||
        !contains(invalid_v3_content_result.messages, "sclv.v3.content_invalid")) {
        return fail("v3 record with empty evidence was accepted");
    }

    auto minimum_id_v3 = v3;
    minimum_id_v3.record_id = "SCLV-CHG-12345678";
    SclvCheckResult minimum_id_input{true, {}, {minimum_id_v3}};
    auto minimum_id_result = check_sclv_shapes(minimum_id_input);
    if (!minimum_id_result.success) {
        return fail("schema-minimum v3 record identifier was rejected");
    }

    auto duplicate_v3 = version_three_record(
        "SCLV-CHG-20260721-duplicate", "8888888888888888888888888888888888888888");
    SclvCheckResult duplicate_revision{true, {}, {v3, duplicate_v3}};
    auto duplicate_revision_result = check_sclv_ledger_continuity(duplicate_revision);
    if (duplicate_revision_result.success || !contains(duplicate_revision_result.messages, "sclv_ledger.revision_duplicate")) {
        return fail("duplicate v3 revision was accepted");
    }

    const fs::path fixture = fs::temp_directory_path() / "symphony-sclv-v2-missing-field.md";
    {
        std::ofstream out(fixture);
        out << "- record_id: `SCLV-PR-010`\n"
               "- record_version: `2`\n"
               "- title: `test`\n"
               "- status: `canonical`\n"
               "- date: `2026-07-18`\n"
               "- change_started_at: `2026-07-18T01:00:00Z`\n"
               "- change_completed_at: `2026-07-18T02:00:00Z`\n"
               "- recording_disposition: `post_merge`\n"
               "- change_type: `canonical_update`\n"
               "- related_pr: `https://github.com/QuanuX/Symphony/pull/10`\n"
               "- merge_commit: `1111111111111111111111111111111111111111`\n"
               "- affected_surfaces: `none`\n"
               "- skvi_references: `none`\n"
               "- change_summary: `none`\n"
               "- relationship_changes: `none`\n"
               "- doctrine_changes: `none`\n"
               "- compatibility_consequences: `none`\n"
               "- publication_consequences: `none`\n"
               "- projection_consequences: `none`\n"
               "- evidence: `none`\n"
               "- non_authorizations: `none`\n"
               "- notes: `none`\n"
               "- record_id: `SCLV-PR-011`\n"
               "- title: `legacy`\n"
               "- status: `canonical`\n"
               "- date: `2026-07-18`\n"
               "- change_type: `canonical_update`\n"
               "- related_pr: `https://github.com/QuanuX/Symphony/pull/11`\n"
               "- merge_commit: `2222222222222222222222222222222222222222`\n"
               "- affected_surfaces: `none`\n"
               "- skvi_references: `none`\n"
               "- change_summary: `none`\n"
               "- relationship_changes: `none`\n"
               "- doctrine_changes: `none`\n"
               "- compatibility_consequences: `none`\n"
               "- publication_consequences: `none`\n"
               "- projection_consequences: `none`\n"
               "- evidence: `none`\n"
               "- non_authorizations: `none`\n"
               "- notes: `none`\n";
    }
    auto parsed = check_sclv_changelog(fixture.string());
    fs::remove(fixture);
    if (parsed.success || !contains(parsed.messages, "field=recorded_at")) {
        return fail("v2 parser accepted a record without recorded_at");
    }

    return 0;
}
