#include "cross_reference.hpp"
#include "evidence.hpp"
#include <fstream>
#include <algorithm>

CrossReferenceResult check_cross_references(const std::string& repo_path, const SkviCheckResult& skvi_result, const SclvCheckResult& sclv_result) {
    CrossReferenceResult result;
    result.success = true;

    auto is_indexed = [&](const std::string& path) {
        return std::find(skvi_result.indexed_paths.begin(), skvi_result.indexed_paths.end(), path) != skvi_result.indexed_paths.end();
    };

    auto file_exists = [&](const std::string& path) {
        std::ifstream file(repo_path + "/" + path);
        return file.good();
    };

    for (const auto& rec : sclv_result.records) {
        // Rule 2 & 3: affected_surfaces must exist
        for (const auto& surface : rec.affected_surfaces) {
            if (file_exists(surface)) {
                if (is_indexed(surface)) {
                    result.messages.push_back(format_evidence(EvidenceCategory::Pass, "sclv.affected_surface.exists", "record_id=" + rec.record_id + " path=" + surface));
                } else {
                    result.messages.push_back(format_evidence(EvidenceCategory::Warning, "sclv.affected_surface.unindexed", "record_id=" + rec.record_id + " path=" + surface));
                }
            } else {
                result.success = false;
                result.messages.push_back(format_evidence(EvidenceCategory::Violation, "sclv.affected_surface.absent", "record_id=" + rec.record_id + " path=" + surface));
            }
        }
    }

    return result;
}
