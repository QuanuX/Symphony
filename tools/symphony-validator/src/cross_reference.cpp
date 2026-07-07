#include "cross_reference.hpp"
#include <fstream>
#include <algorithm>

CrossReferenceResult check_cross_references(const std::string& repo_path, const SkviCheckResult& skvi_result, const SclvCheckResult& sclv_result) {
    CrossReferenceResult result;
    result.success = true;

    auto is_indexed = [&](const std::string& path) {
        if (path == "knowledge/skvi/INDEX.md") return true; // Implicitly indexed
        return std::find(skvi_result.indexed_paths.begin(), skvi_result.indexed_paths.end(), path) != skvi_result.indexed_paths.end();
    };

    auto file_exists = [&](const std::string& path) {
        std::ifstream file(repo_path + "/" + path);
        return file.good();
    };

    for (const auto& rec : sclv_result.records) {
        // Rule 1: skvi_references must be indexed
        for (const auto& ref : rec.skvi_references) {
            if (is_indexed(ref)) {
                result.messages.push_back("evidence pass sclv.skvi_reference.indexed record_id=" + rec.record_id + " path=" + ref);
            } else {
                result.success = false;
                result.messages.push_back("evidence violation sclv.skvi_reference.unindexed record_id=" + rec.record_id + " path=" + ref);
            }
        }

        // Rule 2 & 3: affected_surfaces must exist
        for (const auto& surface : rec.affected_surfaces) {
            if (file_exists(surface)) {
                if (is_indexed(surface)) {
                    result.messages.push_back("evidence pass sclv.affected_surface.exists record_id=" + rec.record_id + " path=" + surface);
                } else {
                    result.messages.push_back("evidence warning sclv.affected_surface.unindexed record_id=" + rec.record_id + " path=" + surface);
                }
            } else {
                result.success = false;
                result.messages.push_back("evidence violation sclv.affected_surface.absent record_id=" + rec.record_id + " path=" + surface);
            }
        }
    }

    return result;
}
