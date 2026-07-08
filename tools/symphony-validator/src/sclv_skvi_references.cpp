#include "sclv_skvi_references.hpp"
#include "evidence.hpp"
#include <algorithm>

SclvSkviReferencesCheckResult check_sclv_skvi_references(const SkviCheckResult& skvi_result, const SclvCheckResult& sclv_result) {
    SclvSkviReferencesCheckResult result;
    result.success = true;

    auto is_indexed = [&](const std::string& path) {
        return std::find(skvi_result.indexed_paths.begin(), skvi_result.indexed_paths.end(), path) != skvi_result.indexed_paths.end();
    };

    for (const auto& rec : sclv_result.records) {
        for (const auto& ref : rec.skvi_references) {
            if (is_indexed(ref)) {
                result.messages.push_back(format_evidence(EvidenceCategory::Pass, "sclv_skvi_reference.indexed", "record_id=" + rec.record_id + " path=" + ref));
            } else {
                result.success = false;
                result.messages.push_back(format_evidence(EvidenceCategory::Violation, "sclv_skvi_reference.unindexed", "record_id=" + rec.record_id + " path=" + ref));
            }
        }
    }

    return result;
}
