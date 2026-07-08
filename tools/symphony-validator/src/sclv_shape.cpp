#include "sclv_shape.hpp"
#include "evidence.hpp"
#include <cctype>

bool is_valid_related_pr(const std::string& pr) {
    const std::string prefix = "https://github.com/QuanuX/Symphony/pull/";
    if (pr.size() <= prefix.size()) {
        return false;
    }
    
    if (pr.substr(0, prefix.size()) != prefix) {
        return false;
    }
    
    // Check remaining part is at least one digit and only digits
    for (size_t i = prefix.size(); i < pr.size(); ++i) {
        if (!std::isdigit(static_cast<unsigned char>(pr[i]))) {
            return false;
        }
    }
    
    return true;
}

bool is_valid_merge_commit(const std::string& commit) {
    if (commit.size() != 40) {
        return false;
    }
    
    for (char c : commit) {
        if (!std::isxdigit(static_cast<unsigned char>(c))) {
            return false;
        }
    }
    
    return true;
}

SclvShapeCheckResult check_sclv_shapes(const SclvCheckResult& sclv_result) {
    SclvShapeCheckResult result;
    result.success = true;

    for (const auto& rec : sclv_result.records) {
        if (rec.has_related_pr) {
            if (is_valid_related_pr(rec.related_pr)) {
                result.messages.push_back(format_evidence(EvidenceCategory::Pass, "sclv.related_pr.shape", "record_id=" + rec.record_id + " related_pr=" + rec.related_pr));
            } else {
                result.success = false;
                result.messages.push_back(format_evidence(EvidenceCategory::Violation, "sclv.related_pr.shape_invalid", "record_id=" + rec.record_id + " related_pr=" + rec.related_pr));
            }
        }
        
        if (rec.has_merge_commit) {
            if (is_valid_merge_commit(rec.merge_commit)) {
                result.messages.push_back(format_evidence(EvidenceCategory::Pass, "sclv.merge_commit.shape", "record_id=" + rec.record_id + " merge_commit=" + rec.merge_commit));
            } else {
                result.success = false;
                result.messages.push_back(format_evidence(EvidenceCategory::Violation, "sclv.merge_commit.shape_invalid", "record_id=" + rec.record_id + " merge_commit=" + rec.merge_commit));
            }
        }
    }

    return result;
}
