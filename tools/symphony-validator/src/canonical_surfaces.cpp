#include "canonical_surfaces.hpp"

#include "evidence.hpp"

#include <symphony/knowledge/engine/manifest_discovery.hpp>

#include <filesystem>
#include <set>

namespace engine = symphony::knowledge::engine;
namespace fs = std::filesystem;

namespace {

fs::path internal_repository_root(const std::string& repo_root) {
    return fs::absolute(fs::path(repo_root)).lexically_normal();
}

}

std::vector<std::string> get_required_canonical_surfaces(const std::string& repo_root) {
    const auto catalog = engine::discover_canonical_surfaces(internal_repository_root(repo_root));
    std::vector<std::string> surfaces;
    surfaces.reserve(catalog.surfaces.size());
    for (const auto& surface : catalog.surfaces) {
        surfaces.push_back(surface.path);
    }
    return surfaces;
}

CanonicalSurfaceCheckResult check_required_canonical_surfaces(const std::string& repo_root) {
    CanonicalSurfaceCheckResult result;
    result.success = true;
    const auto catalog = engine::discover_canonical_surfaces(internal_repository_root(repo_root));

    std::set<std::string> unreadable_paths;
    for (const auto& issue : catalog.issues) {
        result.success = false;
        if (issue.code == "manifest.bootstrap_unreadable" ||
            issue.code == "manifest.surface_unreadable" ||
            issue.code == "manifest.unreadable") {
            unreadable_paths.insert(issue.path);
        }
        result.messages.push_back(format_evidence(
            EvidenceCategory::Violation,
            "canonical_surface." + issue.code,
            "path=" + issue.path + " " + issue.detail));
    }

    for (const auto& manifest : catalog.manifests) {
        if (!unreadable_paths.contains(manifest.path)) {
            result.messages.push_back(format_evidence(
                EvidenceCategory::Pass,
                "canonical_surface.owner_manifest",
                "path=" + manifest.path));
        }
    }
    for (const auto& surface : catalog.surfaces) {
        if (!unreadable_paths.contains(surface.path)) {
            result.messages.push_back(format_evidence(
                EvidenceCategory::Pass,
                "canonical_surface.exists",
                "path=" + surface.path + " owner=" + surface.owner_manifest));
        }
    }
    return result;
}
