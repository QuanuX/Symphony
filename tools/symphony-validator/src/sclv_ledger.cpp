#include "sclv_ledger.hpp"
#include "evidence.hpp"
#include <unordered_set>
#include <cctype>
#include <string>
#include <utility>

// Helper to extract PR number from a GitHub pull request URL.
// E.g., "https://github.com/QuanuX/Symphony/pull/33" -> 33
int extract_pr_number(const std::string& url) {
    auto pos = url.rfind("/pull/");
    if (pos == std::string::npos) {
        return -1; // Not found
    }
    std::string num_str = url.substr(pos + 6);
    try {
        return std::stoi(num_str);
    } catch (...) {
        return -1;
    }
}

// Helper to extract PR number from record_id.
// E.g., "SCLV-PR-033" -> 33
int extract_record_number(const std::string& record_id) {
    if (record_id.size() >= 11 && record_id.substr(0, 8) == "SCLV-PR-") {
        try {
            return std::stoi(record_id.substr(8));
        } catch (...) {
            return -1;
        }
    }
    return -1;
}

static bool is_leap_year(int year) {
    return year % 4 == 0 && (year % 100 != 0 || year % 400 == 0);
}

static bool is_strict_utc_timestamp(const std::string& value) {
    if (value.size() != 20 || value[4] != '-' || value[7] != '-' ||
        value[10] != 'T' || value[13] != ':' || value[16] != ':' || value[19] != 'Z') {
        return false;
    }

    const int digit_positions[] = {0, 1, 2, 3, 5, 6, 8, 9, 11, 12, 14, 15, 17, 18};
    for (int position : digit_positions) {
        if (!std::isdigit(static_cast<unsigned char>(value[position]))) {
            return false;
        }
    }

    const int year = std::stoi(value.substr(0, 4));
    const int month = std::stoi(value.substr(5, 2));
    const int day = std::stoi(value.substr(8, 2));
    const int hour = std::stoi(value.substr(11, 2));
    const int minute = std::stoi(value.substr(14, 2));
    const int second = std::stoi(value.substr(17, 2));
    if (year == 0 || month < 1 || month > 12 || hour > 23 || minute > 59 || second > 59) {
        return false;
    }

    const int days_by_month[] = {31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31};
    int maximum_day = days_by_month[month - 1];
    if (month == 2 && is_leap_year(year)) {
        maximum_day = 29;
    }
    return day >= 1 && day <= maximum_day;
}

SclvLedgerContinuityResult check_sclv_ledger_continuity(const SclvCheckResult& sclv_result) {
    SclvLedgerContinuityResult result;
    result.success = true;

    std::unordered_set<std::string> seen_record_ids;
    std::unordered_set<std::string> seen_related_prs;
    std::unordered_set<std::string> seen_merge_commits;
    std::string previous_recorded_at;
    int aligned_record_count = 0;

    for (const auto& record : sclv_result.records) {
        // Uniqueness checks
        if (!record.record_id.empty()) {
            if (seen_record_ids.count(record.record_id) > 0) {
                result.success = false;
                result.messages.push_back(format_evidence(EvidenceCategory::Violation, "sclv_ledger.record_id_duplicate", "record_id=" + record.record_id));
            } else {
                seen_record_ids.insert(record.record_id);
                result.messages.push_back(format_evidence(EvidenceCategory::Pass, "sclv_ledger.record_id_unique", "record_id=" + record.record_id));
            }
        }

        if (record.has_related_pr && !record.related_pr.empty()) {
            if (seen_related_prs.count(record.related_pr) > 0) {
                result.success = false;
                result.messages.push_back(format_evidence(EvidenceCategory::Violation, "sclv_ledger.related_pr_duplicate", "related_pr=" + record.related_pr));
            } else {
                seen_related_prs.insert(record.related_pr);
                result.messages.push_back(format_evidence(EvidenceCategory::Pass, "sclv_ledger.related_pr_unique", "related_pr=" + record.related_pr));
            }
        }

        if (record.has_merge_commit && !record.merge_commit.empty()) {
            if (seen_merge_commits.count(record.merge_commit) > 0) {
                result.success = false;
                result.messages.push_back(format_evidence(EvidenceCategory::Violation, "sclv_ledger.merge_commit_duplicate", "merge_commit=" + record.merge_commit));
            } else {
                seen_merge_commits.insert(record.merge_commit);
                result.messages.push_back(format_evidence(EvidenceCategory::Pass, "sclv_ledger.merge_commit_unique", "merge_commit=" + record.merge_commit));
            }
        }

        // Numeric Alignment check
        int rec_num = extract_record_number(record.record_id);
        if (rec_num != -1) {
            if (record.has_related_pr) {
                int pr_num = extract_pr_number(record.related_pr);
                if (pr_num != -1) {
                    if (rec_num == pr_num) {
                        aligned_record_count++;
                        result.messages.push_back(format_evidence(EvidenceCategory::Pass, "sclv_ledger.record_pr_aligned", "record_id=" + record.record_id + " related_pr=" + record.related_pr));
                    } else {
                        result.success = false;
                        result.messages.push_back(format_evidence(EvidenceCategory::Violation, "sclv_ledger.record_pr_mismatch", "record_id=" + record.record_id + " related_pr=" + record.related_pr));
                    }
                }
            }
        }

        const bool has_temporal_field = record.has_change_started_at || record.has_change_completed_at ||
            record.has_recorded_at || record.has_recording_disposition || record.has_recovery_reason;
        if (!record.has_record_version && has_temporal_field) {
            result.success = false;
            result.messages.push_back(format_evidence(EvidenceCategory::Violation, "sclv_ledger.temporal_version_absent", "record_id=" + record.record_id));
        }

        if (record.has_record_version && record.record_version != 1 && record.record_version != 2) {
            result.success = false;
            result.messages.push_back(format_evidence(EvidenceCategory::Violation, "sclv_ledger.record_version_invalid", "record_id=" + record.record_id + " record_version=" + std::to_string(record.record_version)));
        }

        if (record.has_record_version && record.record_version == 2) {
            bool timestamps_valid = true;
            const std::pair<std::string, std::string> timestamps[] = {
                {"change_started_at", record.change_started_at},
                {"change_completed_at", record.change_completed_at},
                {"recorded_at", record.recorded_at},
            };
            for (const auto& [field, value] : timestamps) {
                if (!is_strict_utc_timestamp(value)) {
                    timestamps_valid = false;
                    result.success = false;
                    result.messages.push_back(format_evidence(EvidenceCategory::Violation, "sclv_ledger.timestamp_invalid", "record_id=" + record.record_id + " field=" + field + " value=" + value));
                }
            }

            if (timestamps_valid) {
                if (record.change_started_at > record.change_completed_at || record.change_completed_at > record.recorded_at) {
                    result.success = false;
                    result.messages.push_back(format_evidence(EvidenceCategory::Violation, "sclv_ledger.temporal_order_invalid", "record_id=" + record.record_id));
                } else {
                    result.messages.push_back(format_evidence(EvidenceCategory::Pass, "sclv_ledger.temporal_order_valid", "record_id=" + record.record_id));
                }

                if (!previous_recorded_at.empty() && record.recorded_at < previous_recorded_at) {
                    result.success = false;
                    result.messages.push_back(format_evidence(EvidenceCategory::Violation, "sclv_ledger.recording_order_invalid", "record_id=" + record.record_id + " previous_recorded_at=" + previous_recorded_at));
                }
                previous_recorded_at = record.recorded_at;
            }

            if (record.recording_disposition != "post_merge" && record.recording_disposition != "late_recovery") {
                result.success = false;
                result.messages.push_back(format_evidence(EvidenceCategory::Violation, "sclv_ledger.recording_disposition_invalid", "record_id=" + record.record_id + " disposition=" + record.recording_disposition));
            } else if (record.recording_disposition == "late_recovery" && record.recovery_reason.empty()) {
                result.success = false;
                result.messages.push_back(format_evidence(EvidenceCategory::Violation, "sclv_ledger.recovery_reason_absent", "record_id=" + record.record_id));
            } else {
                result.messages.push_back(format_evidence(EvidenceCategory::Pass, "sclv_ledger.recording_disposition_valid", "record_id=" + record.record_id + " disposition=" + record.recording_disposition));
            }
        }
    }

    result.messages.push_back(format_evidence(EvidenceCategory::Pass, "sclv_ledger.sparse_pr_namespace", "aligned_records=" + std::to_string(aligned_record_count) + " github_pr_numbers_need_not_be_contiguous=true"));

    return result;
}
