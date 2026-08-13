#include "root_summary.hpp"

#include <symphony/knowledge/engine/digest.hpp>
#include <symphony/knowledge/engine/json.hpp>

#include <filesystem>
#include <fstream>
#include <iostream>
#include <iterator>
#include <stdexcept>
#include <string>
#include <unistd.h>

namespace {

namespace fs = std::filesystem;
namespace engine = symphony::knowledge::engine;

class TemporaryDirectory final {
public:
    TemporaryDirectory() {
        std::string pattern =
            (fs::canonical(fs::temp_directory_path()) / "symphony-root-summary-XXXXXX").string();
        pattern.push_back('\0');
        char* value = ::mkdtemp(pattern.data());
        if (value == nullptr) {
            throw std::runtime_error("mkdtemp failed");
        }
        path_ = value;
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

void require(const bool condition, const std::string& message) {
    if (!condition) {
        throw std::runtime_error(message);
    }
}

std::string read_file(const fs::path& path) {
    std::ifstream input(path, std::ios::binary);
    if (!input.good()) {
        throw std::runtime_error("read failed: " + path.string());
    }
    return std::string((std::istreambuf_iterator<char>(input)), std::istreambuf_iterator<char>());
}

void write_file(const fs::path& path, const std::string& contents) {
    fs::create_directories(path.parent_path());
    std::ofstream output(path, std::ios::binary);
    if (!output.good()) {
        throw std::runtime_error("write failed: " + path.string());
    }
    output << contents;
}

void write_json(const fs::path& path, engine::Json value, const char* digest_field) {
    value.erase(digest_field);
    value[digest_field] = engine::tagged_sha256(value.dump());
    write_file(path, value.dump(2) + "\n");
}

void copy_inputs(const fs::path& repository, const fs::path& destination) {
    for (const auto* relative : {
            "README.md",
            "knowledge/FEATURE-ADMINISTRATION-PROFILE.json",
            "knowledge/sodv/RELEASES.md",
            "knowledge/ssfv/COVERAGE.md",
            "knowledge/ssfv/REGISTRY.md",
            "tools/qxctl/COMMANDS.json"}) {
        const auto target = destination / relative;
        fs::create_directories(target.parent_path());
        fs::copy_file(repository / relative, target, fs::copy_options::overwrite_existing);
    }
}

bool contains(const RootSummaryResult& result, const std::string& text) {
    for (const auto& message : result.messages) {
        if (message.find(text) != std::string::npos) {
            return true;
        }
    }
    return false;
}

void test_canonical(const fs::path& repository) {
    require(root_summary_is_selected(repository.string()),
        "canonical root-summary contract was not selected");
    const auto releases = check_sodv_releases(repository.string());
    require(releases.success, "canonical SODV source is invalid");
    const auto projection = project_root_summary(repository.string(), releases);
    require(projection.success, "canonical projection failed");
    require(projection.projection_json.find("symphony.repository.root-summary.v1") != std::string::npos,
        "projection protocol missing");
    require(projection.projection_markdown.find("Machine-Checked Repository Snapshot") != std::string::npos,
        "Markdown projection missing");
    require(projection.projection_markdown.find("github.com/QuanuX/Symphony/libraries/stav-protocol-go") !=
        std::string::npos, "completed SODV unit missing");
    const auto checked = check_root_summary(repository.string(), releases);
    require(checked.success, "canonical README summary is stale");
}

void test_selection_boundary() {
    TemporaryDirectory temporary;
    write_file(temporary.path() / "README.md", "# Pre-SSFV Fixture\n");
    require(!root_summary_is_selected(temporary.path().string()),
        "pre-contract repository selected root-summary assurance");

    write_file(temporary.path() / "README.md",
        "# Fixture\n<!-- symphony:root-summary:v1:begin -->\n");
    require(root_summary_is_selected(temporary.path().string()),
        "managed marker did not select root-summary assurance");
}

void test_stale_and_missing_regions(const fs::path& repository) {
    TemporaryDirectory temporary;
    copy_inputs(repository, temporary.path());
    const auto releases = check_sodv_releases(temporary.path().string());
    require(releases.success, "fixture SODV source is invalid");

    auto readme = read_file(temporary.path() / "README.md");
    const auto marker = readme.find("registered features: **");
    require(marker != std::string::npos, "canonical fixture count not found");
    const auto count = marker + std::string("registered features: **").size();
    readme.insert(count, "0");
    write_file(temporary.path() / "README.md", readme);
    const auto stale_before = read_file(temporary.path() / "README.md");
    const auto stale = check_root_summary(temporary.path().string(), releases);
    require(!stale.success && contains(stale, "root_summary.stale"),
        "stale generated count passed");
    require(read_file(temporary.path() / "README.md") == stale_before,
        "stale root-summary validation changed repository bytes");

    write_file(temporary.path() / "README.md", "# Fixture\n");
    const auto missing_before = read_file(temporary.path() / "README.md");
    const auto missing = check_root_summary(temporary.path().string(), releases);
    require(!missing.success && contains(missing, "root_summary.readme_region"),
        "missing generated region passed");
    require(read_file(temporary.path() / "README.md") == missing_before,
        "malformed root-summary validation changed repository bytes");

    auto inline_marker = read_file(repository / "README.md");
    const auto inline_begin = inline_marker.find("<!-- symphony:root-summary:v1:begin -->");
    require(inline_begin != std::string::npos, "canonical begin marker missing");
    inline_marker.insert(inline_begin, "prefix");
    write_file(temporary.path() / "README.md", inline_marker);
    const auto inline_result = check_root_summary(temporary.path().string(), releases);
    require(!inline_result.success && contains(inline_result, "root_summary.readme_region"),
        "same-line marker contamination passed");
}

void test_source_change_requires_regeneration(const fs::path& repository) {
    TemporaryDirectory temporary;
    copy_inputs(repository, temporary.path());
    auto commands = engine::Json::parse(
        read_file(temporary.path() / "tools/qxctl/COMMANDS.json"));
    auto fixture = commands.at("commands").back();
    fixture["command_id"] = "qxcmd:symphony:zz-root-summary-fixture";
    fixture["grammar"] = "qxctl zz-root-summary-fixture";
    commands["commands"].push_back(std::move(fixture));
    write_json(temporary.path() / "tools/qxctl/COMMANDS.json", commands, "registry_digest");
    const auto releases = check_sodv_releases(temporary.path().string());
    const auto checked = check_root_summary(temporary.path().string(), releases);
    require(!checked.success && contains(checked, "root_summary.stale"),
        "valid changed source did not stale the root summary");
}

}

int main(int argc, char** argv) {
    try {
        if (argc != 2) {
            throw std::runtime_error("repository argument required");
        }
        const fs::path repository = fs::canonical(argv[1]);
        test_selection_boundary();
        test_canonical(repository);
        test_stale_and_missing_regions(repository);
        test_source_change_requires_regeneration(repository);
        std::cout << "root summary tests passed\n";
        return 0;
    } catch (const std::exception& error) {
        std::cerr << "root summary tests failed: " << error.what() << "\n";
        return 1;
    }
}
