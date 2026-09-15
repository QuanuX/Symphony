#include "symphony/knowledge/engine/manifest_discovery.hpp"

#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/limits.hpp"
#include "symphony/knowledge/engine/path.hpp"

#include <algorithm>
#include <array>
#include <map>
#include <set>
#include <string_view>
#include <utility>

namespace symphony::knowledge::engine {
namespace {

constexpr std::string_view root_manifest_path = "knowledge/MANIFEST.md";
constexpr std::string_view canonical_heading = "## Canonical Surfaces";
constexpr std::string_view subordinate_heading = "## Subordinate Manifests";

constexpr std::array<std::string_view, 3> bootstrap_paths = {
    "README.md",
    "go.work",
    root_manifest_path,
};

enum class Section {
    none,
    canonical_surfaces,
    subordinate_manifests,
};

struct ParsedManifest final {
    std::vector<std::string> canonical_surfaces;
    std::vector<std::string> subordinate_manifests;
    bool has_canonical_section = false;
    bool has_subordinate_section = false;
};

void add_issue(
    CanonicalSurfaceCatalog& catalog,
    std::string code,
    std::string path,
    std::string detail) {
    if (catalog.issues.size() >= Limits::max_manifest_issues) {
        throw Error(
            "manifest.issue_limit",
            "canonical-surface manifest issue limit exceeded",
            5);
    }
    catalog.issues.push_back(ManifestDiscoveryIssue{
        std::move(code), std::move(path), std::move(detail)});
}

bool ends_with_manifest_name(std::string_view path) {
    return path == "MANIFEST.md" || path.ends_with("/MANIFEST.md");
}

std::string line_without_cr(std::string_view line) {
    if (!line.empty() && line.back() == '\r') {
        line.remove_suffix(1U);
    }
    return std::string(line);
}

bool exact_path_bullet(std::string_view line, std::string& path) {
    constexpr std::string_view prefix = "- `";
    if (!line.starts_with(prefix) || line.size() <= prefix.size() || line.back() != '`') {
        return false;
    }
    path = std::string(line.substr(prefix.size(), line.size() - prefix.size() - 1U));
    return !path.empty() && path.find('`') == std::string::npos;
}

ParsedManifest parse_manifest(
    const std::string& manifest_path,
    const std::string& contents,
    CanonicalSurfaceCatalog& catalog) {
    ParsedManifest parsed;
    Section active = Section::none;
    std::size_t line_number = 0;
    std::size_t position = 0;

    while (position <= contents.size()) {
        const auto end = contents.find('\n', position);
        const auto length = end == std::string::npos ? contents.size() - position : end - position;
        const auto line = line_without_cr(std::string_view(contents).substr(position, length));
        ++line_number;

        if (line.size() > Limits::max_manifest_line_bytes) {
            add_issue(
                catalog,
                "manifest.line_too_large",
                manifest_path,
                "line=" + std::to_string(line_number));
            active = Section::none;
        } else if (line == canonical_heading) {
            if (parsed.has_canonical_section) {
                add_issue(
                    catalog,
                    "manifest.section_duplicate",
                    manifest_path,
                    "line=" + std::to_string(line_number) + " heading=Canonical Surfaces");
                active = Section::none;
            } else {
                parsed.has_canonical_section = true;
                active = Section::canonical_surfaces;
            }
        } else if (line == subordinate_heading) {
            if (parsed.has_subordinate_section) {
                add_issue(
                    catalog,
                    "manifest.section_duplicate",
                    manifest_path,
                    "line=" + std::to_string(line_number) + " heading=Subordinate Manifests");
                active = Section::none;
            } else {
                parsed.has_subordinate_section = true;
                active = Section::subordinate_manifests;
            }
        } else if (line.starts_with('#')) {
            active = Section::none;
        } else if (active != Section::none && !line.empty()) {
            std::string path;
            auto& target = active == Section::canonical_surfaces
                ? parsed.canonical_surfaces
                : parsed.subordinate_manifests;
            if (exact_path_bullet(line, path)) {
                if (target.size() >= Limits::max_manifest_entries) {
                    throw Error(
                        "manifest.entry_limit",
                        manifest_path + ": manifest declaration entry limit exceeded",
                        5);
                }
                target.push_back(std::move(path));
            } else if (target.empty() || line.starts_with("- ")) {
                add_issue(
                    catalog,
                    "manifest.declaration_syntax",
                    manifest_path,
                    "line=" + std::to_string(line_number));
                active = Section::none;
            } else {
                // The exact declaration block is the first contiguous list after
                // its heading. Later manifest prose is outside the machine grammar.
                active = Section::none;
            }
        }

        if (end == std::string::npos) {
            break;
        }
        position = end + 1U;
    }

    if (!parsed.has_canonical_section) {
        add_issue(
            catalog,
            "manifest.canonical_section_missing",
            manifest_path,
            "required heading=Canonical Surfaces");
    } else if (parsed.canonical_surfaces.empty()) {
        add_issue(
            catalog,
            "manifest.canonical_section_empty",
            manifest_path,
            "at least one exact path bullet is required");
    }
    if (manifest_path == root_manifest_path && !parsed.has_subordinate_section) {
        add_issue(
            catalog,
            "manifest.subordinate_section_missing",
            manifest_path,
            "required heading=Subordinate Manifests");
    }

    const auto validate_paths = [&](const std::vector<std::string>& paths, bool manifests) {
        std::set<std::string> seen;
        for (const auto& path : paths) {
            if (!is_safe_relative_path(path)) {
                add_issue(
                    catalog,
                    manifests ? "manifest.subordinate_path_unsafe" : "manifest.surface_path_unsafe",
                    manifest_path,
                    "declared_path=" + path);
            } else if (manifests && !ends_with_manifest_name(path)) {
                add_issue(
                    catalog,
                    "manifest.subordinate_path_invalid",
                    manifest_path,
                    "declared_path=" + path);
            }
            if (!seen.insert(path).second) {
                add_issue(
                    catalog,
                    manifests ? "manifest.subordinate_duplicate" : "manifest.surface_duplicate",
                    manifest_path,
                    "declared_path=" + path);
            }
        }
    };
    validate_paths(parsed.canonical_surfaces, false);
    validate_paths(parsed.subordinate_manifests, true);

    std::sort(parsed.canonical_surfaces.begin(), parsed.canonical_surfaces.end());
    std::sort(parsed.subordinate_manifests.begin(), parsed.subordinate_manifests.end());
    return parsed;
}

class Discovery final {
public:
    Discovery(
        const std::filesystem::path& root,
        std::int64_t deadline_unix_ms,
        CanonicalSurfaceCatalog& catalog)
        : root_(root), deadline_unix_ms_(deadline_unix_ms), catalog_(catalog) {}

    void visit(const std::string& manifest_path) {
        if (!is_safe_relative_path(manifest_path) || !ends_with_manifest_name(manifest_path)) {
            add_issue(
                catalog_,
                "manifest.traversal_path_invalid",
                manifest_path.empty() ? std::string(root_manifest_path) : manifest_path,
                "subordinate manifest path is not a safe MANIFEST.md path");
            return;
        }
        if (active_.contains(manifest_path)) {
            add_issue(
                catalog_,
                "manifest.traversal_cycle",
                manifest_path,
                "manifest is already active in the traversal");
            return;
        }
        if (visited_.contains(manifest_path)) {
            add_issue(
                catalog_,
                "manifest.traversal_duplicate",
                manifest_path,
                "manifest is reachable more than once");
            return;
        }
        if (visited_.size() >= Limits::max_manifest_files) {
            throw Error(
                "manifest.file_limit",
                "canonical-surface owner-manifest limit exceeded",
                5);
        }

        visited_.insert(manifest_path);
        active_.insert(manifest_path);

        std::string contents;
        try {
            contents = read_regular_file_no_follow(
                root_, manifest_path, Limits::max_snapshot_file_bytes, deadline_unix_ms_);
        } catch (const Error& error) {
            add_issue(
                catalog_,
                "manifest.unreadable",
                manifest_path,
                "reason=" + error.code());
            active_.erase(manifest_path);
            return;
        }

        auto parsed = parse_manifest(manifest_path, contents, catalog_);
        if (std::find(
                parsed.canonical_surfaces.begin(),
                parsed.canonical_surfaces.end(),
                manifest_path) == parsed.canonical_surfaces.end()) {
            add_issue(
                catalog_,
                "manifest.self_undeclared",
                manifest_path,
                "owner manifest must declare its own path");
        }

        catalog_.manifests.push_back(OwnerManifestDeclaration{
            manifest_path,
            parsed.canonical_surfaces,
            parsed.subordinate_manifests,
        });
        for (const auto& subordinate : parsed.subordinate_manifests) {
            visit(subordinate);
        }
        active_.erase(manifest_path);
    }

private:
    const std::filesystem::path& root_;
    std::int64_t deadline_unix_ms_;
    CanonicalSurfaceCatalog& catalog_;
    std::set<std::string> visited_;
    std::set<std::string> active_;
};

}

CanonicalSurfaceCatalog discover_canonical_surfaces(
    const std::filesystem::path& root,
    std::int64_t deadline_unix_ms) {
    CanonicalSurfaceCatalog catalog;
    for (const auto path : bootstrap_paths) {
        catalog.bootstrap_paths.emplace_back(path);
        try {
            static_cast<void>(read_regular_file_no_follow(
                root,
                std::string(path),
                Limits::max_snapshot_file_bytes,
                deadline_unix_ms));
        } catch (const Error& error) {
            add_issue(
                catalog,
                "manifest.bootstrap_unreadable",
                std::string(path),
                "reason=" + error.code());
        }
    }

    Discovery discovery(root, deadline_unix_ms, catalog);
    discovery.visit(std::string(root_manifest_path));

    std::map<std::string, std::string> owners;
    for (const auto& manifest : catalog.manifests) {
        for (const auto& surface : manifest.canonical_surfaces) {
            if (!is_safe_relative_path(surface)) {
                continue;
            }
            const auto [iterator, inserted] = owners.emplace(surface, manifest.path);
            if (!inserted && iterator->second != manifest.path) {
                add_issue(
                    catalog,
                    "manifest.surface_owner_duplicate",
                    surface,
                    "owners=" + iterator->second + "," + manifest.path);
            }
        }
    }

    for (const auto& bootstrap : catalog.bootstrap_paths) {
        owners.try_emplace(bootstrap, "bootstrap");
    }
    for (const auto& [surface, owner] : owners) {
        try {
            static_cast<void>(read_regular_file_no_follow(
                root,
                surface,
                Limits::max_snapshot_file_bytes,
                deadline_unix_ms));
        } catch (const Error& error) {
            add_issue(
                catalog,
                "manifest.surface_unreadable",
                surface,
                "owner=" + owner + " reason=" + error.code());
        }
        catalog.surfaces.push_back(CanonicalSurfaceDeclaration{surface, owner});
    }

    std::sort(catalog.bootstrap_paths.begin(), catalog.bootstrap_paths.end());
    std::sort(catalog.manifests.begin(), catalog.manifests.end(), [](const auto& left, const auto& right) {
        return left.path < right.path;
    });
    std::sort(catalog.issues.begin(), catalog.issues.end(), [](const auto& left, const auto& right) {
        if (left.path != right.path) {
            return left.path < right.path;
        }
        if (left.code != right.code) {
            return left.code < right.code;
        }
        return left.detail < right.detail;
    });
    return catalog;
}

}
