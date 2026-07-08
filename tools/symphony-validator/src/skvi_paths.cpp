#include "skvi_paths.hpp"
#include "evidence.hpp"
#include <sys/stat.h>

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

        std::string full_path = repo_root + "/" + path;
        struct stat path_stat;
        if (stat(full_path.c_str(), &path_stat) != 0) {
            result.success = false;
            result.messages.push_back(format_evidence(EvidenceCategory::Violation, "skvi_path.indexed_path_missing", "path=" + path));
            continue;
        }

        if (!S_ISREG(path_stat.st_mode)) {
            result.success = false;
            result.messages.push_back(format_evidence(EvidenceCategory::Violation, "skvi_path.indexed_path_not_file", "path=" + path));
            continue;
        }

        result.messages.push_back(format_evidence(EvidenceCategory::Pass, "skvi_path.indexed_path_exists", "path=" + path));
    }

    return result;
}
