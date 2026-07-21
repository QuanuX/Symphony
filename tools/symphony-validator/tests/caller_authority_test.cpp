#include "caller_authority.hpp"

#include <algorithm>
#include <atomic>
#include <chrono>
#include <filesystem>
#include <fstream>
#include <iostream>
#include <stdexcept>
#include <string>
#include <vector>

namespace fs = std::filesystem;

namespace {

constexpr std::size_t MAX_PHYSICAL_LINE = 64 * 1024;
constexpr std::size_t MAX_NORMALIZED_PARAGRAPH = 256 * 1024;
constexpr std::size_t MAX_FILE_SIZE = 4 * 1024 * 1024;

[[noreturn]] void fail(const std::string& message) {
    throw std::runtime_error(message);
}

#define REQUIRE(condition) \
    do { \
        if (!(condition)) { \
            fail(std::string("REQUIRE failed: ") + #condition + " at " + __FILE__ + ":" + std::to_string(__LINE__)); \
        } \
    } while (false)

class TempDirectory {
public:
    explicit TempDirectory(const std::string& prefix = "symphony_caller_authority") {
        static std::atomic<unsigned long> counter{0};
        const auto timestamp = std::chrono::steady_clock::now().time_since_epoch().count();
        path_ = fs::temp_directory_path() /
            (prefix + "_" + std::to_string(timestamp) + "_" + std::to_string(counter++));
        std::error_code ec;
        fs::create_directories(path_, ec);
        if (ec) fail("unable to create temporary directory: " + ec.message());
    }

    TempDirectory(const TempDirectory&) = delete;
    TempDirectory& operator=(const TempDirectory&) = delete;

    ~TempDirectory() {
        std::error_code ec;
        fs::permissions(path_, fs::perms::owner_all, fs::perm_options::add, ec);
        ec.clear();
        fs::remove_all(path_, ec);
    }

    const fs::path& path() const { return path_; }

private:
    fs::path path_;
};

void write_file(const fs::path& repo_root, const fs::path& relative_path, const std::string& content) {
    const fs::path path = repo_root / relative_path;
    std::error_code ec;
    fs::create_directories(path.parent_path(), ec);
    if (ec) fail("unable to create fixture directory: " + ec.message());

    std::ofstream output(path, std::ios::binary | std::ios::trunc);
    if (!output) fail("unable to create fixture file: " + path.string());
    output.write(content.data(), static_cast<std::streamsize>(content.size()));
    if (!output) fail("unable to write fixture file: " + path.string());
}

void remove_path(const fs::path& path) {
    std::error_code ec;
    fs::remove_all(path, ec);
    if (ec) fail("unable to remove fixture path: " + ec.message());
}

bool contains_exact(const CallerAuthorityCheckResult& result, const std::string& evidence) {
    return std::find(result.messages.begin(), result.messages.end(), evidence) != result.messages.end();
}

bool contains_text(const CallerAuthorityCheckResult& result, const std::string& text) {
    return std::any_of(result.messages.begin(), result.messages.end(), [&](const std::string& message) {
        return message.find(text) != std::string::npos;
    });
}

std::size_t count_exact(const CallerAuthorityCheckResult& result, const std::string& evidence) {
    return static_cast<std::size_t>(std::count(result.messages.begin(), result.messages.end(), evidence));
}

std::vector<std::string> violations(const CallerAuthorityCheckResult& result) {
    std::vector<std::string> found;
    for (const auto& message : result.messages) {
        if (message.starts_with("evidence violation ")) found.push_back(message);
    }
    return found;
}

std::string make_paragraph(std::size_t serialized_size) {
    std::string content;
    content.reserve(serialized_size);

    std::size_t full_lines = serialized_size / 4096;
    std::size_t remainder = serialized_size % 4096;
    if (remainder == 1 && full_lines > 0) {
        --full_lines;
        remainder += 4096;
    }

    for (std::size_t index = 0; index < full_lines; ++index) {
        content.append(4095, 'a');
        content.push_back('\n');
    }
    if (remainder > 0) {
        if (remainder <= 4096) {
            content.append(remainder - 1, 'a');
            content.push_back('\n');
        } else {
            content.append(4094, 'a');
            content.push_back('\n');
            content.append(remainder - 4096, 'a');
            content.push_back('\n');
        }
    }

    REQUIRE(content.size() == serialized_size);
    return content;
}

std::string make_safe_file(std::size_t serialized_size) {
    std::string content;
    content.reserve(serialized_size);
    std::size_t lines_in_paragraph = 0;
    while (content.size() < serialized_size) {
        const std::size_t remaining = serialized_size - content.size();
        if (lines_in_paragraph == 100 && remaining > 0) {
            content.push_back('\n');
            lines_in_paragraph = 0;
            continue;
        }
        if (remaining == 1) {
            content.push_back('\n');
            break;
        }
        const std::size_t payload = std::min<std::size_t>(1000, remaining - 1);
        content.append(payload, 'a');
        content.push_back('\n');
        ++lines_in_paragraph;
    }
    REQUIRE(content.size() == serialized_size);
    return content;
}

void run_tests() {
    TempDirectory repository;
    const fs::path repo_root = repository.path();

    fs::create_directories(repo_root / "knowledge/sclv");
    fs::create_directories(repo_root / "knowledge/sodv");
    fs::create_directories(repo_root / "tools/qxctl");
    fs::create_directories(repo_root / "tools/symphony-validator/tests");

    auto check_case = [&](const std::string& content, bool expected_success, const std::string& expected_evidence = "") {
        write_file(repo_root, "README.md", content);
        const auto result = check_caller_authority(repo_root.string());
        if (result.success != expected_success ||
            (!expected_evidence.empty() && !contains_exact(result, expected_evidence))) {
            std::cerr << "Case failed: " << content << "\n";
            std::cerr << "Expected success: " << expected_success << ", actual: " << result.success << "\n";
            std::cerr << "Expected evidence: " << expected_evidence << "\n";
            for (const auto& message : result.messages) std::cerr << "  " << message << "\n";
            fail("caller-authority case failed");
        }
        remove_path(repo_root / "README.md");
    };

    check_case("Authorization MUST NOT assign different authority because a caller is human, AI, agentic, a service, or a workload.", true);
    check_case("Caller classifications are descriptive facts, never authorization inputs.", true);
    check_case("Caller type is not evaluated.", true);
    check_case("No caller may directly mutate the ledger file through a supported interface.", true);
    check_case("Consumers include humans, reviewers, and agentic tools.", true);
    check_case("Install a per-user launchd agent.", true);
    check_case("A future provider may use an SSH agent.", true);
    check_case("The service account must be provisioned by the owner.", true);
    check_case("A troll is not a caller identity or authorization class.", true);
    check_case("The report is human-readable.", true);
    check_case("Authorization MUST NOT assign different authority\nbecause a caller is human or AI.", true);

    check_case("Humans may. Other systems apply.", true);
    check_case("Agents only document behavior. Operators apply updates.", true);
    check_case("Apply is documented. This guidance is for agents only.", true);
    check_case("Caller type is descriptive. Authorization differs based on market conditions.", true);

    check_case("AI agents may never apply.", false,
        "evidence violation caller_authority.class_subject_modal path=README.md line=1 class=ai_agent");
    check_case("AI agents\nmust not ratify.", false,
        "evidence violation caller_authority.class_subject_modal path=README.md line=1-2 class=ai_agent");
    check_case("AI agents are limited to querying and proposing.", false,
        "evidence violation caller_authority.class_subject_status path=README.md line=1 class=ai_agent");
    check_case("Apply is unavailable to AI agents.", false,
        "evidence violation caller_authority.class_targeted_availability path=README.md line=1 class=ai_agent");
    check_case("Only humans may ratify this record.", false,
        "evidence violation caller_authority.class_exclusive_operation path=README.md line=1 class=human");
    check_case("Humans ratify; agents draft.", false,
        "evidence violation caller_authority.human_exclusive_governance path=README.md line=1 class=human");
    check_case("This is a human-ratified record.", false,
        "evidence violation caller_authority.human_exclusive_governance path=README.md line=1 class=human");
    check_case("Repair requires human review.", false,
        "evidence violation caller_authority.human_exclusive_governance path=README.md line=1 class=human");
    check_case("Authorization differs based on whether the caller is an AI agent.", false,
        "evidence violation caller_authority.caller_type_decision path=README.md line=1");
    check_case("Caller type determines apply authority.", false,
        "evidence violation caller_authority.caller_type_decision path=README.md line=1");

    const std::vector<std::pair<std::string, std::string>> class_cases = {
        {"AI may apply.", "ai"},
        {"AI agents may apply.", "ai_agent"},
        {"Humans may apply.", "human"},
        {"Human callers may apply.", "human_caller"},
        {"Agents may apply.", "agent"},
        {"Agentic callers may apply.", "agentic_caller"},
        {"Service callers may apply.", "service_caller"},
        {"Workload callers may apply.", "workload_caller"},
        {"Organization callers may apply.", "organization_caller"},
    };
    for (const auto& [content, class_id] : class_cases) {
        check_case(content, false,
            "evidence violation caller_authority.class_subject_modal path=README.md line=1 class=" + class_id);
    }

    check_case("Caller type is not metadata. Authorization differs based on caller class.", false,
        "evidence violation caller_authority.caller_type_decision path=README.md line=1");
    check_case("Authorization does not vary by caller type.", true);
    check_case("Authorization never determines access by caller type.", true);
    check_case("Authorization is not metadata but depends on caller type.", false,
        "evidence violation caller_authority.caller_type_decision path=README.md line=1");
    check_case("Repair is not subject to human review.", true);
    check_case("Does not require human approval.", true);
    check_case("Requires no human review.", true);
    check_case("Apply is for agents only.", false,
        "evidence violation caller_authority.class_exclusive_operation path=README.md line=1 class=agent");
    check_case("## Governance\nHumans should use SKVI.", false,
        "evidence violation caller_authority.class_subject_modal path=README.md line=2 class=human");

    write_file(repo_root, "README.md", "Humans may approve.\nAgents may approve.");
    const auto deduplicated = check_caller_authority(repo_root.string());
    REQUIRE(!deduplicated.success);
    REQUIRE(count_exact(
        deduplicated,
        "evidence violation caller_authority.class_subject_modal path=README.md line=1 class=human") == 1);
    REQUIRE(violations(deduplicated).size() == 1);
    remove_path(repo_root / "README.md");

    write_file(repo_root, "tools/qxctl/INTENT.md", "AI agents may never apply.");
    const auto qxctl = check_caller_authority(repo_root.string());
    REQUIRE(contains_exact(
        qxctl,
        "evidence violation caller_authority.class_subject_modal path=tools/qxctl/INTENT.md line=1 class=ai_agent"));
    remove_path(repo_root / "tools/qxctl/INTENT.md");

    write_file(repo_root, "knowledge/sclv/CHANGELOG.md", "AI agents may never apply.\n- record_id: 123");
    REQUIRE(!check_caller_authority(repo_root.string()).success);
    write_file(repo_root, "knowledge/sclv/CHANGELOG.md", "Preamble\n- record_id: 123\nAI agents may never apply.");
    REQUIRE(check_caller_authority(repo_root.string()).success);
    write_file(repo_root, "knowledge/sodv/RELEASES.md", "Preamble\n- release_record_id: 123\nHuman review is required.");
    REQUIRE(check_caller_authority(repo_root.string()).success);

    write_file(repo_root, "tools/symphony-validator/tests/bad.md", "AI agents may never apply.");
    REQUIRE(check_caller_authority(repo_root.string()).success);

    write_file(repo_root, "knowledge/doc1.md", "Humans may approve.\nAgents may ratify.\n");
    write_file(repo_root, "knowledge/doc2.md", "Apply is for agents only.\n");
    const auto deterministic_first = check_caller_authority(repo_root.string());
    const auto deterministic_second = check_caller_authority(repo_root.string());
    REQUIRE(deterministic_first.messages == deterministic_second.messages);
    remove_path(repo_root / "knowledge/doc1.md");
    remove_path(repo_root / "knowledge/doc2.md");

    write_file(repo_root, "modules/x/build/some.md", "AI agents may never apply.");
    REQUIRE(check_caller_authority(repo_root.string()).success);
    remove_path(repo_root / "modules/x");

    write_file(repo_root, "knowledge/stav/fixtures/some.md", "AI agents may never apply.");
    REQUIRE(!check_caller_authority(repo_root.string()).success);
    remove_path(repo_root / "knowledge/stav");

    write_file(repo_root, "README.md", "Safe metadata.\n");
    const auto tellg_failure = check_caller_authority_with_test_fault(
        repo_root.string(), CallerAuthorityTestFault::tellg_failure, "README.md");
    REQUIRE(contains_exact(tellg_failure, "evidence violation caller_authority.unreadable path=README.md"));
    remove_path(repo_root / "README.md");

    write_file(repo_root, "knowledge/skvi/metadata.md", "Safe metadata.\n");
    const auto metadata_failure = check_caller_authority_with_test_fault(
        repo_root.string(), CallerAuthorityTestFault::metadata_failure, "knowledge/skvi/metadata.md");
    REQUIRE(contains_exact(
        metadata_failure,
        "evidence violation caller_authority.discovery_failed path=knowledge/skvi/metadata.md"));
    remove_path(repo_root / "knowledge/skvi/metadata.md");

    const auto construction_failure = check_caller_authority_with_test_fault(
        repo_root.string(), CallerAuthorityTestFault::iterator_construction_failure, "knowledge");
    REQUIRE(contains_exact(
        construction_failure,
        "evidence violation caller_authority.discovery_failed path=knowledge"));

    write_file(repo_root, "modules/increment.md", "Safe metadata.\n");
    const auto increment_failure = check_caller_authority_with_test_fault(
        repo_root.string(), CallerAuthorityTestFault::iterator_increment_failure, "modules/increment.md");
    REQUIRE(contains_exact(
        increment_failure,
        "evidence violation caller_authority.discovery_failed path=modules/increment.md"));
    remove_path(repo_root / "modules/increment.md");

    TempDirectory outside("symphony_caller_authority_outside");
    write_file(outside.path(), "target.md", "AI agents may never apply.\n");
    fs::create_directories(repo_root / "knowledge/skvi");
    std::error_code symlink_error;
    fs::create_symlink(outside.path() / "target.md", repo_root / "knowledge/skvi/external.md", symlink_error);
    if (symlink_error) fail("unable to create external symlink fixture: " + symlink_error.message());

    const auto external_symlink = check_caller_authority(repo_root.string());
    REQUIRE(contains_exact(
        external_symlink,
        "evidence violation caller_authority.symlink_unsupported path=knowledge/skvi/external.md"));
    REQUIRE(!contains_text(external_symlink, "class_subject_modal path=knowledge/skvi/external.md"));
    REQUIRE(!contains_text(external_symlink, outside.path().generic_string()));
    remove_path(repo_root / "knowledge/skvi/external.md");

    symlink_error.clear();
    fs::create_symlink(outside.path() / "missing.md", repo_root / "knowledge/skvi/broken.md", symlink_error);
    if (symlink_error) fail("unable to create broken symlink fixture: " + symlink_error.message());
    const auto broken_symlink = check_caller_authority(repo_root.string());
    REQUIRE(contains_exact(
        broken_symlink,
        "evidence violation caller_authority.symlink_unsupported path=knowledge/skvi/broken.md"));
    remove_path(repo_root / "knowledge/skvi/broken.md");

    write_file(repo_root, "README.md", "AI agents may never apply.\n");
    symlink_error.clear();
    fs::create_symlink(outside.path() / "target.md", repo_root / "knowledge/skvi/z.md", symlink_error);
    if (symlink_error) fail("unable to create deterministic symlink fixture: " + symlink_error.message());
    symlink_error.clear();
    fs::create_symlink(outside.path() / "missing.md", repo_root / "knowledge/skvi/a.md", symlink_error);
    if (symlink_error) fail("unable to create deterministic broken-link fixture: " + symlink_error.message());

    const auto failure_first = check_caller_authority(repo_root.string());
    const auto failure_second = check_caller_authority(repo_root.string());
    REQUIRE(failure_first.messages == failure_second.messages);
    const std::vector<std::string> expected_violations = {
        "evidence violation caller_authority.symlink_unsupported path=knowledge/skvi/a.md",
        "evidence violation caller_authority.symlink_unsupported path=knowledge/skvi/z.md",
        "evidence violation caller_authority.class_subject_modal path=README.md line=1 class=ai_agent",
    };
    REQUIRE(violations(failure_first) == expected_violations);
    remove_path(repo_root / "README.md");
    remove_path(repo_root / "knowledge/skvi/a.md");
    remove_path(repo_root / "knowledge/skvi/z.md");

    fs::path cleanup_probe;
    try {
        TempDirectory cleanup_test("symphony_caller_authority_cleanup");
        cleanup_probe = cleanup_test.path();
        write_file(cleanup_probe, "nested/fixture.md", "fixture\n");
        throw std::runtime_error("intentional cleanup probe");
    } catch (const std::runtime_error&) {
    }
    REQUIRE(!fs::exists(cleanup_probe));

    fs::create_directories(repo_root / "knowledge/skvi");
    for (const std::size_t length : {MAX_PHYSICAL_LINE - 1, MAX_PHYSICAL_LINE}) {
        write_file(repo_root, "knowledge/skvi/line.md", std::string(length, 'a'));
        REQUIRE(check_caller_authority(repo_root.string()).success);
    }
    write_file(repo_root, "knowledge/skvi/line.md", std::string(MAX_PHYSICAL_LINE + 1, 'a'));
    const auto long_line = check_caller_authority(repo_root.string());
    REQUIRE(contains_exact(
        long_line,
        "evidence violation caller_authority.line_length_exceeded path=knowledge/skvi/line.md line=1"));
    remove_path(repo_root / "knowledge/skvi/line.md");

    for (const std::size_t size : {MAX_NORMALIZED_PARAGRAPH - 1, MAX_NORMALIZED_PARAGRAPH}) {
        write_file(repo_root, "knowledge/skvi/paragraph.md", make_paragraph(size));
        REQUIRE(check_caller_authority(repo_root.string()).success);
    }
    write_file(repo_root, "knowledge/skvi/paragraph.md", make_paragraph(MAX_NORMALIZED_PARAGRAPH + 1));
    const auto long_paragraph = check_caller_authority(repo_root.string());
    REQUIRE(contains_exact(
        long_paragraph,
        "evidence violation caller_authority.paragraph_size_exceeded path=knowledge/skvi/paragraph.md"));
    remove_path(repo_root / "knowledge/skvi/paragraph.md");

    for (const std::size_t size : {MAX_FILE_SIZE - 1, MAX_FILE_SIZE}) {
        write_file(repo_root, "knowledge/skvi/file.md", make_safe_file(size));
        REQUIRE(check_caller_authority(repo_root.string()).success);
    }
    write_file(repo_root, "knowledge/skvi/file.md", make_safe_file(MAX_FILE_SIZE + 1));
    const auto large_file = check_caller_authority(repo_root.string());
    REQUIRE(contains_exact(
        large_file,
        "evidence violation caller_authority.file_size_exceeded path=knowledge/skvi/file.md"));
}

}

int main() {
    try {
        run_tests();
        std::cout << "All caller-authority tests passed.\n";
        return 0;
    } catch (const std::exception& error) {
        std::cerr << "Caller-authority test failure: " << error.what() << "\n";
        return 1;
    }
}
