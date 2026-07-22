#include "sodv_releases.hpp"

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
    require(result.records_checked == 3U && result.transactions_checked == 1U,
        "canonical SODV counts mismatch");
    require(contains(result, "sodv.releases.scan_complete records=3 transactions=1 violations=0"),
        "canonical completion evidence missing");
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

}

int main(int argc, char** argv) {
    try {
        if (argc != 2) { throw std::runtime_error("repository root required"); }
        const auto repository = fs::canonical(argv[1]);
        test_actual(repository);
        test_duplicate(repository);
        test_relationship_and_no_follow(repository);
        std::cout << "sodv release validator tests passed\n";
        return 0;
    } catch (const std::exception& error) {
        std::cerr << "sodv release validator tests failed: " << error.what() << '\n';
        return 1;
    }
}
