#pragma once

#include <cstdint>
#include <filesystem>
#include <limits>
#include <string>
#include <vector>

namespace symphony::knowledge::engine {

struct ManifestDiscoveryIssue final {
    std::string code;
    std::string path;
    std::string detail;
};

struct CanonicalSurfaceDeclaration final {
    std::string path;
    std::string owner_manifest;
};

struct OwnerManifestDeclaration final {
    std::string path;
    std::vector<std::string> canonical_surfaces;
    std::vector<std::string> subordinate_manifests;
};

struct CanonicalSurfaceCatalog final {
    std::vector<std::string> bootstrap_paths;
    std::vector<OwnerManifestDeclaration> manifests;
    std::vector<CanonicalSurfaceDeclaration> surfaces;
    std::vector<ManifestDiscoveryIssue> issues;

    [[nodiscard]] bool valid() const noexcept { return issues.empty(); }
};

// Reads only the fixed repository bootstrap and owner-declared manifest graph.
// It reports declaration and repository-shape problems as bounded issues; it
// never infers an owner from a directory or from an implementation file.
[[nodiscard]] CanonicalSurfaceCatalog discover_canonical_surfaces(
    const std::filesystem::path& root,
    std::int64_t deadline_unix_ms = std::numeric_limits<std::int64_t>::max());

}
