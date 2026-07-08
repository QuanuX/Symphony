#include "skvi_paths.hpp"
#include "evidence.hpp"
#include <filesystem>

SkviPathsCheckResult check_skvi_paths(const std::string& repo_root, const SkviCheckResult& skvi_result) {
    SkviPathsCheckResult result;
    result.success = true;

    for (const auto& entry : skvi_result.entries) {
        const std::string& path = entry.path;
        if (path.empty()) {
            continue;
        }

        if (path[0] == '/') {
            result.success = false;
            result.messages.push_back(format_evidence(EvidenceCategory::Violation, "skvi_path.invalid_relative_path", "path=" + path));
            continue;
        }

        if (path.find("..") != std::string::npos) {
            result.success = false;
            result.messages.push_back(format_evidence(EvidenceCategory::Violation, "skvi_path.invalid_relative_path", "path=" + path));
            continue;
        }

        std::filesystem::path full_path = std::filesystem::path(repo_root) / path;
        
        if (!std::filesystem::exists(full_path)) {
            result.success = false;
            result.messages.push_back(format_evidence(EvidenceCategory::Violation, "skvi_path.indexed_path_missing", "path=" + path));
            continue;
        }

        if (!std::filesystem::is_regular_file(full_path)) {
            result.success = false;
            result.messages.push_back(format_evidence(EvidenceCategory::Violation, "skvi_path.indexed_path_not_file", "path=" + path));
            continue;
        }

        result.messages.push_back(format_evidence(EvidenceCategory::Pass, "skvi_path.indexed_path_exists", "path=" + path));
    }

    return result;
}
