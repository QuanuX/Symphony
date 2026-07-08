#include "canonical_surfaces.hpp"
#include "evidence.hpp"
#include <filesystem>
#include <array>

namespace fs = std::filesystem;

CanonicalSurfaceCheckResult check_required_canonical_surfaces(const std::string& repo_root) {
    CanonicalSurfaceCheckResult result;
    result.success = true;
    fs::path root(repo_root);

    const std::array<std::string, 35> required_surfaces = {
        "README.md",
        "INTENT.md",
        "modules/node-troll/INTENT.md",
        "modules/node-troll/MANIFEST.md",
        "modules/node-troll/INSTALL.md",
        "modules/node-troll/SKILL.md",
        "modules/bus-troll/INTENT.md",
        "modules/bus-troll/MANIFEST.md",
        "modules/bus-troll/INSTALL.md",
        "modules/bus-troll/SKILL.md",
        "modules/hotpath-runtime/INTENT.md",
        "modules/hotpath-runtime/MANIFEST.md",
        "modules/hotpath-runtime/INSTALL.md",
        "modules/hotpath-runtime/SKILL.md",
        "knowledge/INTENT.md",
        "knowledge/skvi/INTENT.md",
        "knowledge/skvi/MANIFEST.md",
        "knowledge/skvi/SKILL.md",
        "knowledge/skvi/SPEC.md",
        "knowledge/skvi/INDEX.md",
        "knowledge/sclv/INTENT.md",
        "knowledge/sclv/MANIFEST.md",
        "knowledge/sclv/SKILL.md",
        "knowledge/sclv/SPEC.md",
        "knowledge/sclv/CHANGELOG.md",
        "knowledge/sodv/INTENT.md",
        "knowledge/sodv/MANIFEST.md",
        "knowledge/sodv/SKILL.md",
        "knowledge/sodv/SPEC.md",
        "tools/symphony-validator/INTENT.md",
        "tools/symphony-validator/MANIFEST.md",
        "tools/symphony-validator/INSTALL.md",
        "tools/symphony-validator/SKILL.md",
        "tools/symphony-validator/SPEC.md",
        "tools/symphony-validator/CMakeLists.txt"
    };

    for (const auto& surface : required_surfaces) {
        fs::path p = root / surface;
        if (fs::exists(p)) {
            result.messages.push_back(format_evidence(EvidenceCategory::Pass, "canonical_surface.exists", "path=" + surface));
        } else {
            result.success = false;
            result.messages.push_back(format_evidence(EvidenceCategory::Violation, "canonical_surface.missing", "path=" + surface));
        }
    }

    return result;
}
