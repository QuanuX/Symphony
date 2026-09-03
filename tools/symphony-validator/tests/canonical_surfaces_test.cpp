#include "canonical_surfaces.hpp"
#include "skvi_coverage.hpp"

#include <algorithm>
#include <filesystem>
#include <fstream>
#include <iostream>
#include <stdexcept>
#include <string>
#include <unistd.h>

namespace fs = std::filesystem;

namespace {

class TemporaryDirectory final {
public:
    TemporaryDirectory() {
        std::string pattern =
            (fs::canonical(fs::temp_directory_path()) / "symphony-canonical-surfaces-XXXXXX").string();
        pattern.push_back('\0');
        char* result = ::mkdtemp(pattern.data());
        if (result == nullptr) {
            throw std::runtime_error("mkdtemp failed");
        }
        path_ = result;
    }
    ~TemporaryDirectory() {
        std::error_code ignored;
        fs::remove_all(path_, ignored);
    }
    TemporaryDirectory(const TemporaryDirectory&) = delete;
    TemporaryDirectory& operator=(const TemporaryDirectory&) = delete;
    [[nodiscard]] const fs::path& path() const { return path_; }

private:
    fs::path path_;
};

void require(bool condition, const std::string& message) {
    if (!condition) {
        throw std::runtime_error(message);
    }
}

void write_file(const fs::path& root, const std::string& relative, const std::string& contents) {
    const auto path = root / relative;
    fs::create_directories(path.parent_path());
    std::ofstream output(path, std::ios::binary);
    require(output.good(), "could not write fixture file");
    output << contents;
}

bool contains(const std::vector<std::string>& messages, const std::string& text) {
    for (const auto& message : messages) {
        if (message.find(text) != std::string::npos) {
            return true;
        }
    }
    return false;
}

void create_fixture(const fs::path& root) {
    write_file(root, "README.md", "fixture\n");
    write_file(root, "INTENT.md", "fixture\n");
    write_file(root, "go.work", "go 1.26.5\n");
    write_file(root, "knowledge/SPEC.md", "fixture\n");
    write_file(
        root,
        "knowledge/MANIFEST.md",
        "# Root\n\n## Canonical Surfaces\n\n"
        "- `knowledge/MANIFEST.md`\n"
        "- `knowledge/SPEC.md`\n\n"
        "## Subordinate Manifests\n\n"
        "- `knowledge/alpha/MANIFEST.md`\n");
    write_file(root, "knowledge/alpha/SKILL.md", "fixture\n");
    write_file(
        root,
        "knowledge/alpha/MANIFEST.md",
        "# Alpha\n\n## Canonical Surfaces\n\n"
        "- `knowledge/alpha/MANIFEST.md`\n"
        "- `knowledge/alpha/SKILL.md`\n");
}

void test_manifest_driven_presence_and_coverage() {
    TemporaryDirectory temporary;
    create_fixture(temporary.path());

    const auto surfaces = get_required_canonical_surfaces(temporary.path().string());
    require(surfaces.size() == 7U, "manifest-driven canonical-surface count mismatch");
    require(surfaces.front() == "INTENT.md", "canonical-surface order is not deterministic");
    require(surfaces.back() == "knowledge/alpha/SKILL.md", "canonical-surface order drift");

    const auto relative_root = temporary.path().lexically_relative(fs::current_path());
    const auto relative_surfaces = get_required_canonical_surfaces(relative_root.string());
    require(relative_surfaces == surfaces, "relative repository path changed manifest discovery");

    const auto presence = check_required_canonical_surfaces(temporary.path().string());
    require(presence.success, "valid manifest-driven surfaces failed presence checks");
    require(
        contains(presence.messages, "canonical_surface.owner_manifest path=knowledge/alpha/MANIFEST.md"),
        "owner-manifest traversal evidence is absent");

    SkviCheckResult index;
    index.success = true;
    index.indexed_paths = surfaces;
    index.indexed_paths.push_back("src/indexed-implementation.cpp");
    const auto complete = check_skvi_coverage(index, temporary.path().string());
    require(complete.success, "exact-once declared-surface coverage was rejected");

    index.indexed_paths.push_back("knowledge/alpha/SKILL.md");
    const auto duplicate = check_skvi_coverage(index, temporary.path().string());
    require(!duplicate.success, "multiply indexed declared surface was accepted");
    require(
        contains(duplicate.messages, "skvi_coverage.declared_surface_indexed_multiple"),
        "multiply indexed surface produced no focused evidence");

    index.indexed_paths.erase(
        std::remove(index.indexed_paths.begin(), index.indexed_paths.end(), "knowledge/SPEC.md"),
        index.indexed_paths.end());
    const auto missing = check_skvi_coverage(index, temporary.path().string());
    require(!missing.success, "unindexed declared surface was accepted");
    require(
        contains(missing.messages, "skvi_coverage.declared_surface_unindexed path=knowledge/SPEC.md"),
        "unindexed declared surface produced no focused evidence");
}

void test_manifest_failure_reaches_validator_evidence() {
    TemporaryDirectory temporary;
    create_fixture(temporary.path());
    fs::remove(temporary.path() / "knowledge/alpha/SKILL.md");
    const auto result = check_required_canonical_surfaces(temporary.path().string());
    require(!result.success, "missing declared canonical surface was accepted");
    require(
        contains(result.messages, "canonical_surface.manifest.surface_unreadable"),
        "missing declared canonical surface produced no parser evidence");
}

}

int main() {
    try {
        test_manifest_driven_presence_and_coverage();
        test_manifest_failure_reaches_validator_evidence();
        std::cout << "canonical surface validator tests passed\n";
        return 0;
    } catch (const std::exception& error) {
        std::cerr << "test failure: " << error.what() << '\n';
        return 1;
    }
}
