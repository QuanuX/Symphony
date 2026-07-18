#include "sclv_changelog.hpp"
#include "sclv_ledger.hpp"
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
