#include "cross_reference.hpp"
#include "sclv_references.hpp"
#include "sclv_skvi_references.hpp"

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
        std::string pattern = (fs::canonical(fs::temp_directory_path()) / "symphony-validator-cross-reference-XXXXXX").string();
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

void write_file(const fs::path& root, const fs::path& relative) {
    const auto path = root / relative;
    fs::create_directories(path.parent_path());
    std::ofstream output(path, std::ios::binary);
    if (!output.good()) {
        throw std::runtime_error("could not write fixture file");
    }
    output << "fixture\n";
}

void test_affected_surfaces_are_summarized_historical_provenance() {
    TemporaryDirectory repository;
    write_file(repository.path(), "src/module.cpp");
    write_file(repository.path(), "tests/module_test.cpp");
    write_file(repository.path(), "cmake/module.cmake");
    write_file(repository.path(), "src/shared.cpp");

    SclvRecord first;
    first.record_id = "SCLV-CHG-FIRST";
    first.affected_surfaces = {
        "src/module.cpp",
        "tests/module_test.cpp",
        "cmake/module.cmake",
        "removed/legacy.cpp",
        "src/shared.cpp",
    };
    SclvRecord second;
    second.record_id = "SCLV-CHG-SECOND";
    second.affected_surfaces = {"src/shared.cpp"};

    SkviCheckResult skvi;
    skvi.success = true;
    skvi.indexed_paths = {"src/shared.cpp"};
    SclvCheckResult sclv;
    sclv.success = true;
    sclv.records = {first, second};

    const auto result = check_cross_references(repository.path().string(), skvi, sclv);
    require(result.success, "historical affected surfaces unexpectedly failed current validation");
    require(result.messages.size() == 1U, "affected-surface provenance was not summarized into one finding");
    require(result.messages.front() ==
        "evidence pass sclv.affected_surface.provenance_summary records=2 occurrences=6 unique_paths=5 present_paths=4 absent_paths=1 unknown_paths=0 indexed_paths=1 unindexed_paths=4",
        "affected-surface provenance summary was not exact and deterministic");
    require(!result.messages.front().contains("src/module.cpp") &&
            !result.messages.front().contains("tests/module_test.cpp") &&
            !result.messages.front().contains("cmake/module.cmake") &&
            !result.messages.front().contains("removed/legacy.cpp"),
        "summary leaked per-occurrence historical path noise");
}

void test_explicit_skvi_references_remain_hard_obligations() {
    TemporaryDirectory repository;
    fs::create_directories(repository.path() / "knowledge/prior-file.md");
    SclvRecord record;
    record.record_id = "SCLV-CHG-EXPLICIT-REFERENCE";
    record.affected_surfaces = {
        "removed/legacy-contract.md",
        "knowledge/prior-file.md",
    };
    record.skvi_references = {"knowledge/required-contract.md"};

    SkviCheckResult skvi;
    skvi.success = true;
    SclvCheckResult sclv;
    sclv.success = true;
    sclv.records = {record};

    const auto absent_provenance = check_sclv_references(
        repository.path().string(), sclv);
    require(!absent_provenance.success,
        "missing explicit SKVI file reference was accepted");
    require(absent_provenance.messages.size() == 1U &&
            absent_provenance.messages.front().contains("field=skvi_references"),
        "historical affected surface leaked into current reference evidence");

    const auto missing = check_sclv_skvi_references(skvi, sclv);
    require(!missing.success, "unindexed explicit SKVI reference was accepted");
    require(missing.messages.size() == 1U && missing.messages.front() ==
        "evidence violation sclv_skvi_reference.unindexed record_id=SCLV-CHG-EXPLICIT-REFERENCE path=knowledge/required-contract.md",
        "unindexed explicit SKVI reference did not retain exact violation evidence");

    write_file(repository.path(), "knowledge/required-contract.md");
    const auto present_reference = check_sclv_references(
        repository.path().string(), sclv);
    require(present_reference.success && present_reference.messages.size() == 1U &&
            present_reference.messages.front() ==
                "evidence pass sclv_reference.skvi_reference_exists record_id=SCLV-CHG-EXPLICIT-REFERENCE path=knowledge/required-contract.md",
        "present explicit SKVI reference did not retain exact pass evidence");

    skvi.indexed_paths.push_back("knowledge/required-contract.md");
    const auto indexed = check_sclv_skvi_references(skvi, sclv);
    require(indexed.success, "indexed explicit SKVI reference failed");
    require(indexed.messages.size() == 1U && indexed.messages.front() ==
        "evidence pass sclv_skvi_reference.indexed record_id=SCLV-CHG-EXPLICIT-REFERENCE path=knowledge/required-contract.md",
        "indexed explicit SKVI reference did not produce exact pass evidence");
}

void test_affected_surface_path_safety_remains_enforced() {
    SclvRecord record;
    record.record_id = "SCLV-CHG-UNSAFE-PROVENANCE";
    record.affected_surfaces = {"../outside.md"};
    SclvCheckResult sclv;
    sclv.success = true;
    sclv.records = {record};
    const auto result = check_sclv_references(".", sclv);
    require(!result.success && result.messages.size() == 1U &&
            result.messages.front() ==
                "evidence violation sclv_reference.invalid_relative_path record_id=SCLV-CHG-UNSAFE-PROVENANCE field=affected_surfaces path=../outside.md",
        "unsafe historical affected-surface path was accepted");
}

} // namespace

int main() {
    try {
        test_affected_surfaces_are_summarized_historical_provenance();
        test_explicit_skvi_references_remain_hard_obligations();
        test_affected_surface_path_safety_remains_enforced();
        std::cout << "sclv cross-reference tests passed\n";
        return 0;
    } catch (const std::exception& error) {
        std::cerr << "sclv cross-reference tests failed: " << error.what() << '\n';
        return 1;
    }
}
