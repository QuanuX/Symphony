#include "sodv_releases.hpp"

#include <filesystem>
#include <fstream>
#include <iostream>
#include <stdexcept>
#include <string>
#include <utility>
#include <unistd.h>

namespace fs = std::filesystem;

namespace {

class TemporaryDirectory final {
public:
    TemporaryDirectory() {
        std::string pattern = (fs::canonical(fs::temp_directory_path()) / "symphony-validator-sodv-XXXXXX").string();
        pattern.push_back('\0');
        char* result = ::mkdtemp(pattern.data());
        if (result == nullptr) { throw std::runtime_error("mkdtemp failed"); }
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
    if (!condition) { throw std::runtime_error(message); }
}

std::string read_file(const fs::path& path) {
    std::ifstream input(path, std::ios::binary);
    if (!input.good()) { throw std::runtime_error("could not read fixture source"); }
    return std::string((std::istreambuf_iterator<char>(input)), std::istreambuf_iterator<char>());
}

void write_file(const fs::path& path, const std::string& contents) {
    fs::create_directories(path.parent_path());
    std::ofstream output(path, std::ios::binary);
    if (!output.good()) { throw std::runtime_error("could not write fixture"); }
    output << contents;
}

bool contains(const SodvReleaseCheckResult& result, const std::string& text) {
    for (const auto& message : result.messages) {
        if (message.find(text) != std::string::npos) { return true; }
    }
    return false;
}

void test_actual(const fs::path& repository) {
    const auto result = check_sodv_releases(repository.string());
    require(result.success, "canonical SODV ledger failed independent validation");
    require(result.records_checked >= 3U && result.transactions_checked >= 1U,
        "canonical SODV historical records missing");
    require(contains(result, "sodv.releases.scan_complete records=" + std::to_string(result.records_checked) +
        " transactions=" + std::to_string(result.transactions_checked) + " violations=0"),
        "canonical completion evidence missing");
}

void replace_once(std::string& contents, const std::string& before, const std::string& after) {
    const auto position = contents.find(before);
    require(position != std::string::npos, "release fixture value absent: " + before);
    contents.replace(position, before.size(), after);
}

std::string v2_record(const std::string& id, const std::string& type, const std::string& status,
                      const std::string& time, const std::string& subject) {
    return "\n- release_record_id: \"" + id + "\"\n"
        "- record_version: `2`\n- record_type: `" + type + "`\n- status: `" + status + "`\n"
        "- disposition: `fixture`\n- recorded_at: `" + time + "`\n- recorded_by: \"fixture\"\n"
        "- subject_record_ids:\n  - \"" + subject + "\"\n"
        "- publication_units:\n  - unit_id: \"fixture-module\"\n"
        "    artifact_kind: `go_module`\n    coordinate: \"example.test/module\"\n"
        "    version: `v0.1.0`\n    tag: \"modules/example/v0.1.0\"\n"
        "    revision_scheme: `git-sha1`\n"
        "    revision_value: `1111111111111111111111111111111111111111`\n"
        "    source_reference: \"fixture-source\"\n"
        "    tag_object: null\n    content: null\n    metadata: null\n"
        "- purpose: \"fixture\"\n- evidence:\n  - \"fixture-evidence\"\n"
        "- non_authorizations:\n  - \"fixture-only\"\n- notes: \"fixture\"\n";
}

SodvReleaseCheckResult check_fixture(const std::string& contents) {
    TemporaryDirectory temporary;
    write_file(temporary.path() / "go.work", "go 1.26.5\n");
    write_file(temporary.path() / "knowledge/sodv/RELEASES.md", contents);
    return check_sodv_releases(temporary.path().string());
}

void test_v2_publication_evidence() {
    const auto authorization = v2_record("SODV-REL-V2-AUTH", "authorization", "authorized",
        "2026-11-01T00:00:00Z", "not_applicable");
    const auto completion = v2_record("SODV-REL-V2-DONE", "completion", "completed",
        "2026-11-01T00:02:00Z", "SODV-REL-V2-AUTH");
    const std::pair<std::string, std::string> publication_values[] = {
        {"tag_object", "`2222222222222222222222222222222222222222`"},
        {"content", "\"h1:published-content\""},
        {"metadata", "\"h1:published-metadata\""},
    };
    auto published = completion;
    for (const auto& [field, value] : publication_values) {
        replace_once(published, "    " + field + ": null", "    " + field + ": " + value);
    }
    const auto result = check_fixture(authorization + published);
    require(result.success && result.records_checked == 2U && result.transactions_checked == 1U &&
        result.published_units.size() == 1U,
        "v2 completion could not learn publication evidence from a null authorization");

    for (const auto& [field, value] : publication_values) {
        auto declared = authorization;
        replace_once(declared, "    " + field + ": null", "    " + field + ": " + value);
        require(check_fixture(declared + published).success,
            "completion rejected matching declared " + field);
        for (const auto& changed : {std::string("null"),
                 std::string(field == "tag_object" ? "`3333333333333333333333333333333333333333`" : "\"h1:changed\"")}) {
            auto mismatch = published;
            replace_once(mismatch, "    " + field + ": " + value, "    " + field + ": " + changed);
            const auto rejected = check_fixture(declared + mismatch);
            require(!rejected.success && contains(rejected, "sodv.releases.authorization_mismatch"),
                "completion changed or dropped declared " + field);
        }
    }

    auto correction = v2_record("SODV-REL-V2-CORRECTION", "authorization_correction", "canonical",
        "2026-11-01T00:01:00Z", "SODV-REL-V2-AUTH");
    for (const auto& [field, value] : publication_values) {
        replace_once(correction, "    " + field + ": null", "    " + field + ": " + value);
    }
    require(check_fixture(authorization + correction + published).success,
        "completion rejected matching correction evidence");
    for (const auto& [field, value] : publication_values) {
        auto mismatch = published;
        replace_once(mismatch, "    " + field + ": " + value, "    " + field + ": null");
        const auto rejected = check_fixture(authorization + correction + mismatch);
        require(!rejected.success && contains(rejected, "sodv.releases.authorization_mismatch"),
            "completion dropped corrected " + field);
        auto incomplete_correction = correction;
        replace_once(incomplete_correction, "    " + field + ": " + value, "    " + field + ": null");
        require(!check_fixture(authorization + incomplete_correction + published).success,
            "completion changed null correction evidence for " + field);
    }

    const std::pair<std::string, std::string> immutable_values[] = {
        {"example.test/module", "example.test/other"},
        {"version: `v0.1.0`", "version: `v0.2.0`"},
        {"modules/example/v0.1.0", "modules/other/v0.1.0"},
        {"1111111111111111111111111111111111111111", "4444444444444444444444444444444444444444"},
    };
    for (const auto& [before, after] : immutable_values) {
        auto mismatch = published;
        replace_once(mismatch, before, after);
        const auto rejected = check_fixture(authorization + mismatch);
        require(!rejected.success && contains(rejected, "sodv.releases.authorization_mismatch"),
            "completion changed an immutable publication binding: " + before);
    }
}

void test_duplicate(const fs::path& repository) {
    TemporaryDirectory temporary;
    auto contents = read_file(repository / "knowledge/sodv/RELEASES.md");
    const auto boundary = contents.find("- release_record_id:");
    require(boundary != std::string::npos, "record boundary absent");
    contents.append("\n").append(contents.substr(boundary));
    write_file(temporary.path() / "go.work", "go 1.26.5\n");
    write_file(temporary.path() / "knowledge/sodv/RELEASES.md", contents);
    const auto result = check_sodv_releases(temporary.path().string());
    require(!result.success && contains(result, "sodv.releases.record_id"),
        "duplicate release records were accepted");
}

void test_relationship_and_no_follow(const fs::path& repository) {
    {
        TemporaryDirectory temporary;
        auto contents = read_file(repository / "knowledge/sodv/RELEASES.md");
        const auto value = contents.find("55f8faf26f4f85213ac23cc1de7ba897b2129a4c", contents.find("SODV-REL-003"));
        require(value != std::string::npos, "completion revision fixture absent");
        contents.replace(value, 40U, "9999999999999999999999999999999999999999");
        write_file(temporary.path() / "go.work", "go 1.26.5\n");
        write_file(temporary.path() / "knowledge/sodv/RELEASES.md", contents);
        const auto result = check_sodv_releases(temporary.path().string());
        require(!result.success && contains(result, "sodv.releases.authorization_mismatch"),
            "completion revision drift was accepted");
    }
    {
        TemporaryDirectory temporary;
        auto contents = read_file(repository / "knowledge/sodv/RELEASES.md");
        const auto completion = contents.find("SODV-REL-003");
        const auto value = contents.find("h1:DGVd771sqzeRpEkTUuuF+9TOK1JVQtyMh2GYR840g70=", completion);
        require(value != std::string::npos, "completion checksum fixture absent");
        contents.replace(value, 47U, "h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=");
        write_file(temporary.path() / "go.work", "go 1.26.5\n");
        write_file(temporary.path() / "knowledge/sodv/RELEASES.md", contents);
        const auto result = check_sodv_releases(temporary.path().string());
        require(!result.success && contains(result, "sodv.releases.authorization_mismatch"),
            "completion checksum drift from the latest correction was accepted");
    }
    {
        TemporaryDirectory temporary;
        write_file(temporary.path() / "go.work", "go 1.26.5\n");
        write_file(temporary.path() / "outside.md", read_file(repository / "knowledge/sodv/RELEASES.md"));
        fs::create_directories(temporary.path() / "knowledge/sodv");
        fs::create_symlink(temporary.path() / "outside.md", temporary.path() / "knowledge/sodv/RELEASES.md");
        const auto result = check_sodv_releases(temporary.path().string());
        require(!result.success && contains(result, "sodv.releases.unreadable"),
            "symlinked SODV ledger was accepted");
    }
}

void test_impossible_timestamp(const fs::path& repository) {
    TemporaryDirectory temporary;
    auto contents = read_file(repository / "knowledge/sodv/RELEASES.md");
    const auto timestamp = contents.find("2026-07-18T07:07:20Z");
    require(timestamp != std::string::npos, "authorization timestamp fixture absent");
    contents.replace(timestamp, 20U, "2026-02-31T07:07:20Z");
    write_file(temporary.path() / "go.work", "go 1.26.5\n");
    write_file(temporary.path() / "knowledge/sodv/RELEASES.md", contents);
    const auto result = check_sodv_releases(temporary.path().string());
    require(!result.success && contains(result, "sodv.releases.record_shape"),
        "impossible SODV calendar date was accepted");
}

}

void test_sodv_artifacts(const std::filesystem::path& repository);

int main(int argc, char** argv) {
    try {
        if (argc != 2) { throw std::runtime_error("repository root required"); }
        const auto repository = fs::canonical(argv[1]);
        test_sodv_artifacts(repository);
        test_v2_publication_evidence();
        test_actual(repository);
        test_duplicate(repository);
        test_relationship_and_no_follow(repository);
        test_impossible_timestamp(repository);
        std::cout << "sodv release validator tests passed\n";
        return 0;
    } catch (const std::exception& error) {
        std::cerr << "sodv release validator tests failed: " << error.what() << '\n';
        return 1;
    }
}
