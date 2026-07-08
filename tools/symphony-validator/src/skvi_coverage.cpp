#include "skvi_coverage.hpp"
#include "canonical_surfaces.hpp"
#include "evidence.hpp"
#include <unordered_set>

SkviCoverageCheckResult check_skvi_coverage(const SkviCheckResult& index_res) {
    SkviCoverageCheckResult result;
    result.success = true;
    
    std::unordered_set<std::string> seen_paths;
    
    // Check required surfaces coverage in required canonical surface order
    std::vector<std::string> required_surfaces = get_required_canonical_surfaces();
    std::unordered_set<std::string> indexed_paths_set(index_res.indexed_paths.begin(), index_res.indexed_paths.end());
    
    for (const auto& req_path : required_surfaces) {
        if (indexed_paths_set.find(req_path) != indexed_paths_set.end()) {
            result.messages.push_back(format_evidence(EvidenceCategory::Pass, "skvi_coverage.required_surface_indexed", "path=" + req_path));
        } else {
            result.success = false;
            result.messages.push_back(format_evidence(EvidenceCategory::Violation, "skvi_coverage.required_surface_unindexed", "path=" + req_path));
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
