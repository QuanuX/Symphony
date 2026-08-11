#include "skvi_coverage.hpp"
#include "canonical_surfaces.hpp"
#include "evidence.hpp"
#include <filesystem>
#include <unordered_set>

namespace fs = std::filesystem;

SkviCoverageCheckResult check_skvi_coverage(const SkviCheckResult& index_res, const std::string& repo_root) {
    SkviCoverageCheckResult result;
    result.success = true;
    
    std::unordered_set<std::string> seen_paths;
    
    // Check required surfaces coverage in required canonical surface order
    std::vector<std::string> required_surfaces = get_required_canonical_surfaces();
    if (fs::exists(fs::path(repo_root) / "knowledge/ssfv")) {
        const std::vector<std::string> ssfv_required = {
            "knowledge/ssfv/INTENT.md",
            "knowledge/ssfv/MANIFEST.md",
            "knowledge/ssfv/SKILL.md",
            "knowledge/ssfv/SPEC.md",
            "knowledge/ssfv/NAMESPACES.md",
            "knowledge/ssfv/REGISTRY.md",
            "knowledge/ssfv/FEATURE-FILE-FORMAT.md"
        };
        required_surfaces.insert(required_surfaces.end(), ssfv_required.begin(), ssfv_required.end());
    }
    if (fs::exists(fs::path(repo_root) / "modules/ssfv-engine")) {
        const std::vector<std::string> ssfv_engine_required = {
            "modules/ssfv-engine/INTENT.md",
            "modules/ssfv-engine/MANIFEST.md",
            "modules/ssfv-engine/INSTALL.md",
            "modules/ssfv-engine/SKILL.md",
            "modules/ssfv-engine/SPEC.md",
            "modules/ssfv-engine/CMakeLists.txt"
        };
        required_surfaces.insert(required_surfaces.end(),
            ssfv_engine_required.begin(), ssfv_engine_required.end());
    }
    if (fs::exists(fs::path(repo_root) / "modules/maestro")) {
        const std::vector<std::string> maestro_required = {
            "modules/maestro/INTENT.md", "modules/maestro/MANIFEST.md",
            "modules/maestro/INSTALL.md", "modules/maestro/SKILL.md",
            "modules/maestro/SPEC.md", "modules/maestro/CMakeLists.txt"
        };
        required_surfaces.insert(required_surfaces.end(),
            maestro_required.begin(), maestro_required.end());
    }
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
