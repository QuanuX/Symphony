#include "sclv_ledger.hpp"
#include "evidence.hpp"
#include <set>
#include <unordered_set>
#include <map>
#include <algorithm>
#include <string>

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

std::string format_missing_gap(int start, int end) {
    std::string result = "";
    for (int i = start; i <= end; ++i) {
        if (!result.empty()) result += ",";
        char buf[16];
        snprintf(buf, sizeof(buf), "SCLV-PR-%03d", i);
        result += buf;
    }
    return result;
}

SclvLedgerContinuityResult check_sclv_ledger_continuity(const SclvCheckResult& sclv_result) {
    SclvLedgerContinuityResult result;
    result.success = true;

    std::unordered_set<std::string> seen_record_ids;
    std::unordered_set<std::string> seen_related_prs;
    std::unordered_set<std::string> seen_merge_commits;
    std::set<int> valid_record_numbers;

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
            valid_record_numbers.insert(rec_num);
            if (record.has_related_pr) {
                int pr_num = extract_pr_number(record.related_pr);
                if (pr_num != -1) {
                    if (rec_num == pr_num) {
                        result.messages.push_back(format_evidence(EvidenceCategory::Pass, "sclv_ledger.record_pr_aligned", "record_id=" + record.record_id + " related_pr=" + record.related_pr));
                    } else {
                        result.success = false;
                        result.messages.push_back(format_evidence(EvidenceCategory::Violation, "sclv_ledger.record_pr_mismatch", "record_id=" + record.record_id + " related_pr=" + record.related_pr));
                    }
                }
            }
        }
    }

    // Ledger Gap Check
    if (!valid_record_numbers.empty()) {
        bool has_gap = false;
        auto it = valid_record_numbers.begin();
        int prev = *it;
        int first_record = prev;
        
        ++it;
        for (; it != valid_record_numbers.end(); ++it) {
            int current = *it;
            if (current > prev + 1) {
                has_gap = true;
                std::string missing_list = format_missing_gap(prev + 1, current - 1);
                char prev_str[16];
                char curr_str[16];
                snprintf(prev_str, sizeof(prev_str), "SCLV-PR-%03d", prev);
                snprintf(curr_str, sizeof(curr_str), "SCLV-PR-%03d", current);
                result.messages.push_back(format_evidence(EvidenceCategory::Warning, "sclv_ledger.record_gap", "from=" + std::string(prev_str) + " to=" + std::string(curr_str) + " missing=" + missing_list));
            }
            prev = current;
        }
        
        if (!has_gap) {
            char first_str[16];
            char last_str[16];
            snprintf(first_str, sizeof(first_str), "SCLV-PR-%03d", first_record);
            snprintf(last_str, sizeof(last_str), "SCLV-PR-%03d", prev);
            result.messages.push_back(format_evidence(EvidenceCategory::Pass, "sclv_ledger.no_record_gaps", "range=" + std::string(first_str) + ".." + std::string(last_str)));
        }
    }

    return result;
}
