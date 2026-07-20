#include "caller_authority.hpp"
#include <iostream>
#include <filesystem>
#include <fstream>
#include <cstdlib>
#include <vector>
#include <chrono>

namespace fs = std::filesystem;

#define REQUIRE(cond) \
    do { \
        if (!(cond)) { \
            std::cerr << "REQUIRE failed: " << #cond << " at " << __FILE__ << ":" << __LINE__ << "\n"; \
            std::exit(1); \
        } \
    } while(0)

std::string create_temp_dir() {
    auto now = std::chrono::system_clock::now().time_since_epoch().count();
    fs::path temp_dir = fs::temp_directory_path() / ("symphony_test_" + std::to_string(now));
    fs::create_directories(temp_dir);
    return temp_dir.string();
}

void write_file(const std::string& repo_root, const std::string& rel_path, const std::string& content) {
    fs::path p = fs::path(repo_root) / rel_path;
    fs::create_directories(p.parent_path());
    std::ofstream out(p);
    out << content;
}

int main() {
    std::string repo_root = create_temp_dir();
    
    fs::create_directories(fs::path(repo_root) / "knowledge" / "sclv");
    fs::create_directories(fs::path(repo_root) / "knowledge" / "sodv");
    fs::create_directories(fs::path(repo_root) / "tools" / "qxctl");
    fs::create_directories(fs::path(repo_root) / "tools" / "symphony-validator" / "tests");
    
    auto check_case_count = [&](const std::string& content, bool expect_pass, const std::string& expected_evidence, int expected_count) {
        std::string rel_path = "README.md";
        write_file(repo_root, rel_path, content);
        auto result = check_caller_authority(repo_root);
        
        int count = 0;
        for (const auto& msg : result.messages) {
            if (!expected_evidence.empty() && msg.find(expected_evidence) != std::string::npos) {
                count++;
            }
        }
        
        if (result.success != expect_pass || count != expected_count) {
            std::cerr << "Test failed for content: " << content << "\n";
            std::cerr << "Expected pass: " << expect_pass << ", Got pass: " << result.success << "\n";
            std::cerr << "Expected count: " << expected_count << ", Got count: " << count << "\n";
            std::cerr << "Expected evidence: " << expected_evidence << "\nMessages:\n";
            for (const auto& msg : result.messages) std::cerr << "  " << msg << "\n";
            std::exit(1);
        }
        fs::remove(fs::path(repo_root) / rel_path);
    };

    auto check_case = [&](const std::string& content, bool expect_pass, const std::string& expected_evidence = "") {
        std::string rel_path = "README.md";
        write_file(repo_root, rel_path, content);
        auto result = check_caller_authority(repo_root);
        
        bool found_evidence = expected_evidence.empty();
        for (const auto& msg : result.messages) {
            if (!expected_evidence.empty() && msg.find(expected_evidence) != std::string::npos) {
                found_evidence = true;
            }
        }
        
        if (result.success != expect_pass || !found_evidence) {
            std::cerr << "Test failed for content: " << content << "\n";
            std::cerr << "Expected pass: " << expect_pass << ", Got: " << result.success << "\n";
            std::cerr << "Expected evidence: " << expected_evidence << "\nMessages:\n";
            for (const auto& msg : result.messages) std::cerr << "  " << msg << "\n";
            std::exit(1);
        }
        fs::remove(fs::path(repo_root) / rel_path);
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

    check_case("AI agents may never apply.", false, "caller_authority.class_subject_modal");
    check_case("AI agents\nmust not ratify.", false, "caller_authority.class_subject_modal");
    check_case("AI agents are limited to querying and proposing.", false, "caller_authority.class_subject_status");
    check_case("Apply is unavailable to AI agents.", false, "caller_authority.class_targeted_availability");
    check_case("Only humans may ratify this record.", false, "caller_authority.class_exclusive_operation");
    check_case("Humans ratify; agents draft.", false, "caller_authority.human_exclusive_governance");
    check_case("This is a human-ratified record.", false, "caller_authority.human_exclusive_governance");
    check_case("Repair requires human review.", false, "caller_authority.human_exclusive_governance");
    check_case("Service callers may not administer safeguards.", false, "caller_authority.class_subject_modal");
    check_case("Authorization differs based on whether the caller is an AI agent.", false, "caller_authority.caller_type_decision");
    check_case("Caller type determines apply authority.", false, "caller_authority.caller_type_decision");

    // Defect A: Negation scope permits evasion
    check_case("Caller type is not metadata. Authorization differs based on caller class.", false, "caller_authority.caller_type_decision");
    check_case("Authorization does not vary by caller type.", true);
    check_case("Never determine authorization based on caller type.", true);
    // Defect B: Negated human-review doctrine
    check_case("Repair is not subject to human review.", true);
    check_case("Does not require human approval.", true);
    check_case("Requires no human review.", true);
    // Defect C: Reverse class-only construction
    check_case("Apply is for agents only.", false, "caller_authority.class_exclusive_operation");
    // Defect D: Heading concatenation bypass
    check_case("## Governance\nHumans should use SKVI.", false, "caller_authority.class_subject_modal");
    // Defect E: Deduplication
    check_case_count("Humans may approve.\nAgents may approve.", false, "caller_authority.class_subject_modal", 1);

    write_file(repo_root, "tools/qxctl/INTENT.md", "AI agents may never apply");
    auto res1 = check_caller_authority(repo_root);
    REQUIRE(res1.success == false);
    fs::remove(fs::path(repo_root) / "tools/qxctl/INTENT.md");

    write_file(repo_root, "knowledge/sclv/CHANGELOG.md", "AI agents may never apply\n- record_id: 123");
    auto res2 = check_caller_authority(repo_root);
    REQUIRE(res2.success == false);

    write_file(repo_root, "knowledge/sclv/CHANGELOG.md", "Preamble\n- record_id: 123\nAI agents may never apply");
    auto res3 = check_caller_authority(repo_root);
    REQUIRE(res3.success == true);

    write_file(repo_root, "knowledge/sodv/RELEASES.md", "Preamble\n- release_record_id: 123\nhuman review");
    auto res4 = check_caller_authority(repo_root);
    REQUIRE(res4.success == true);

    write_file(repo_root, "tools/symphony-validator/tests/bad.md", "AI agents may never apply");
    auto res5 = check_caller_authority(repo_root);
    REQUIRE(res5.success == true);

    // Byte-for-byte determinism test
    write_file(repo_root, "knowledge/doc1.md", "Humans may approve.\nAgents may ratify.\n");
    write_file(repo_root, "knowledge/doc2.md", "Apply is for agents only.\n");
    auto run1 = check_caller_authority(repo_root);
    auto run2 = check_caller_authority(repo_root);
    REQUIRE(run1.messages == run2.messages);
    fs::remove(fs::path(repo_root) / "knowledge/doc1.md");
    fs::remove(fs::path(repo_root) / "knowledge/doc2.md");

    // Nested build exclusion
    write_file(repo_root, "modules/x/build/some.md", "AI agents may never apply");
    auto res_build = check_caller_authority(repo_root);
    REQUIRE(res_build.success == true);
    fs::remove_all(fs::path(repo_root) / "modules/x");

    // STAV fixture inclusion
    write_file(repo_root, "knowledge/stav/fixtures/some.md", "AI agents may never apply");
    auto res_stav = check_caller_authority(repo_root);
    REQUIRE(res_stav.success == false);
    fs::remove_all(fs::path(repo_root) / "knowledge/stav/fixtures");

    // Tellg failure test / Discovery error
    // We simulate by making a directory unreadable
    {
        struct RaiiDir {
            std::string p;
            RaiiDir(std::string path) : p(path) { fs::create_directories(p); fs::permissions(p, fs::perms::none); }
            ~RaiiDir() { fs::permissions(p, fs::perms::all); fs::remove_all(p); }
        } bad_dir(fs::path(repo_root) / "knowledge" / "skvi" / "bad_dir");

        auto res_err = check_caller_authority(repo_root);
        REQUIRE(res_err.success == false);
        bool found_discovery = false;
        for (const auto& msg : res_err.messages) {
            if (msg.find("discovery_failed") != std::string::npos) found_discovery = true;
        }
        REQUIRE(found_discovery);
    }

    // Symlink bypass verification
    std::error_code ec;
    fs::path target = fs::path(repo_root) / "real.md";
    write_file(repo_root, "real.md", "some text");
    {
        struct RaiiSymlink {
            std::string p;
            RaiiSymlink(fs::path target, std::string link) : p(link) { std::error_code e; fs::create_symlink(target, link, e); }
            ~RaiiSymlink() { std::error_code e; fs::remove(p, e); }
        } bad_symlink(target, fs::path(repo_root) / "knowledge" / "skvi" / "sym.md");

        auto res6 = check_caller_authority(repo_root);
        REQUIRE(res6.success == false);
        bool found_sym = false;
        for (const auto& msg : res6.messages) {
            if (msg.find("symlink_unsupported") != std::string::npos) found_sym = true;
        }
        REQUIRE(found_sym);
    }
    fs::remove(target);

    // Resource limits
    std::string long_line = std::string(65537, 'a') + "\n";
    fs::create_directories(fs::path(repo_root) / "knowledge" / "skvi");
    write_file(repo_root, "knowledge/skvi/long_line.md", long_line);
    auto res_ll = check_caller_authority(repo_root);
    REQUIRE(res_ll.success == false);
    bool found_ll = false;
    for (const auto& msg : res_ll.messages) {
        if (msg.find("line_length_exceeded") != std::string::npos) found_ll = true;
    }
    REQUIRE(found_ll);
    fs::remove(fs::path(repo_root) / "knowledge" / "skvi" / "long_line.md");

    std::string long_para = "";
    for (int i=0; i<5000; i++) long_para += std::string(60, 'a') + "\n";
    write_file(repo_root, "knowledge/skvi/long_para.md", long_para);
    auto res_lp = check_caller_authority(repo_root);
    REQUIRE(res_lp.success == false);
    bool found_lp = false;
    for (const auto& msg : res_lp.messages) {
        if (msg.find("paragraph_size_exceeded") != std::string::npos) found_lp = true;
    }
    REQUIRE(found_lp);
    fs::remove(fs::path(repo_root) / "knowledge" / "skvi" / "long_para.md");

    std::string big_file = std::string(4 * 1024 * 1024 + 1, 'a');
    write_file(repo_root, "knowledge/skvi/big_file.md", big_file);
    auto res_bf = check_caller_authority(repo_root);
    REQUIRE(res_bf.success == false);
    bool found_bf = false;
    for (const auto& msg : res_bf.messages) {
        if (msg.find("file_size_exceeded") != std::string::npos) found_bf = true;
    }
    REQUIRE(found_bf);
    fs::remove(fs::path(repo_root) / "knowledge" / "skvi" / "big_file.md");

    fs::remove_all(repo_root);
    std::cout << "All tests passed.\n";
    return 0;
}
