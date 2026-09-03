#include "skvi_coverage.hpp"
#include "canonical_surfaces.hpp"
#include "evidence.hpp"
#include <unordered_map>
#include <unordered_set>

SkviCoverageCheckResult check_skvi_coverage(const SkviCheckResult& index_res, const std::string& repo_root) {
    SkviCoverageCheckResult result;
    result.success = true;
    
    std::unordered_set<std::string> seen_paths;
    
    const std::vector<std::string> required_surfaces =
        get_required_canonical_surfaces(repo_root);
    std::unordered_map<std::string, std::size_t> indexed_path_counts;
    for (const auto& path : index_res.indexed_paths) {
        ++indexed_path_counts[path];
    }
    
    for (const auto& req_path : required_surfaces) {
        const auto count = indexed_path_counts[req_path];
        if (count == 1U) {
            result.messages.push_back(format_evidence(EvidenceCategory::Pass, "skvi_coverage.declared_surface_indexed_once", "path=" + req_path));
        } else {
            result.success = false;
            result.messages.push_back(format_evidence(
                EvidenceCategory::Violation,
                count == 0U
                    ? "skvi_coverage.declared_surface_unindexed"
                    : "skvi_coverage.declared_surface_indexed_multiple",
                "path=" + req_path + " count=" + std::to_string(count)));
        }
    }

    // Check uniqueness of indexed paths (preserving index order)
    for (const auto& path : index_res.indexed_paths) {
        if (seen_paths.find(path) == seen_paths.end()) {
            seen_paths.insert(path);
            result.messages.push_back(format_evidence(EvidenceCategory::Pass, "skvi_coverage.index_path_unique", "path=" + path));
        } else {
            result.success = false;
            result.messages.push_back(format_evidence(EvidenceCategory::Violation, "skvi_coverage.index_path_duplicate", "path=" + path));
        }
    }

    return result;
}
