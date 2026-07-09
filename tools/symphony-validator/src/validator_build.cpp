#include "validator_build.hpp"
#include "evidence.hpp"
#include <filesystem>
#include <fstream>
#include <sstream>
#include <set>
#include <algorithm>

namespace fs = std::filesystem;

ValidatorBuildCheckResult check_validator_build(const std::string& repo_root) {
    ValidatorBuildCheckResult result;
    result.success = true;

    fs::path repo_path(repo_root);
    fs::path cmake_path = repo_path / "tools" / "symphony-validator" / "CMakeLists.txt";
    fs::path src_dir = repo_path / "tools" / "symphony-validator" / "src";

    if (!fs::exists(cmake_path)) {
        return result; // Or handle if needed, but missing validator is handled elsewhere
    }

    std::vector<std::string> listed_sources;
    std::ifstream file(cmake_path);
    std::string line;
    bool in_add_executable = false;

    while (std::getline(file, line)) {
        auto comment_pos = line.find('#');
        if (comment_pos != std::string::npos) {
            line = line.substr(0, comment_pos);
        }

        std::istringstream iss(line);
        std::string token;
        while (iss >> token) {
            if (!in_add_executable) {
                if (token.find("add_executable(symphony-validator") == 0) {
                    in_add_executable = true;
                    std::string remainder = token.substr(std::string("add_executable(symphony-validator").length());
                    if (!remainder.empty()) {
                        if (remainder.find(")") != std::string::npos) {
                            std::string src = remainder.substr(0, remainder.find(")"));
                            if (!src.empty()) listed_sources.push_back(src);
                            in_add_executable = false;
                        } else {
                            listed_sources.push_back(remainder);
                        }
                    }
                }
            } else {
                if (token == ")") {
                    in_add_executable = false;
                } else if (token.find(")") != std::string::npos) {
                    in_add_executable = false;
                    std::string src = token.substr(0, token.find(")"));
                    if (!src.empty()) listed_sources.push_back(src);
                } else {
                    listed_sources.push_back(token);
                }
            }
        }
    }

    std::set<std::string> seen_sources;

    for (const auto& raw_path : listed_sources) {
        fs::path p(raw_path);
        bool has_error = false;

        if (seen_sources.count(raw_path)) {
            result.messages.push_back(format_evidence(EvidenceCategory::Violation, "validator_build.source_list_duplicate", "path=" + raw_path));
            result.success = false;
            has_error = true;
        }
        seen_sources.insert(raw_path);

        if (p.is_absolute()) {
            result.messages.push_back(format_evidence(EvidenceCategory::Violation, "validator_build.invalid_source_path", "path=" + raw_path));
            result.success = false;
            has_error = true;
        }

        bool has_traversal = false;
        for (const auto& part : p) {
            if (part == "..") {
                has_traversal = true;
                break;
            }
        }
        if (has_traversal && !p.is_absolute()) {
            result.messages.push_back(format_evidence(EvidenceCategory::Violation, "validator_build.invalid_source_path", "path=" + raw_path));
            result.success = false;
            has_error = true;
        }

        std::string expected_prefix = "src/";
        if (raw_path.substr(0, expected_prefix.length()) != expected_prefix) {
            result.messages.push_back(format_evidence(EvidenceCategory::Violation, "validator_build.source_outside_src", "path=" + raw_path));
            result.success = false;
            has_error = true;
        }

        if (p.extension() != ".cpp") {
            result.messages.push_back(format_evidence(EvidenceCategory::Violation, "validator_build.invalid_source_extension", "path=" + raw_path));
            result.success = false;
            has_error = true;
        }

        fs::path full_path = repo_path / "tools" / "symphony-validator" / p;
        if (!fs::exists(full_path)) {
            result.messages.push_back(format_evidence(EvidenceCategory::Violation, "validator_build.source_missing", "path=" + raw_path));
            result.success = false;
            has_error = true;
        } else if (!fs::is_regular_file(full_path)) {
            result.messages.push_back(format_evidence(EvidenceCategory::Violation, "validator_build.source_not_file", "path=" + raw_path));
            result.success = false;
            has_error = true;
        }

        if (!has_error) {
            result.messages.push_back(format_evidence(EvidenceCategory::Pass, "validator_build.source_list_entry_valid", "path=" + raw_path));
        }
    }

    std::vector<std::string> physical_sources;
    if (fs::exists(src_dir) && fs::is_directory(src_dir)) {
        for (const auto& entry : fs::directory_iterator(src_dir)) {
            if (entry.is_regular_file() && entry.path().extension() == ".cpp") {
                std::string rel_path = "src/" + entry.path().filename().string();
                physical_sources.push_back(rel_path);
            }
        }
    }
    std::sort(physical_sources.begin(), physical_sources.end());

    for (const auto& phys_path : physical_sources) {
        int count = std::count(listed_sources.begin(), listed_sources.end(), phys_path);
        if (count == 1) {
            result.messages.push_back(format_evidence(EvidenceCategory::Pass, "validator_build.source_file_listed", "path=" + phys_path));
        } else if (count == 0) {
            result.messages.push_back(format_evidence(EvidenceCategory::Violation, "validator_build.source_file_unlisted", "path=" + phys_path));
            result.success = false;
        }
    }

    return result;
}
