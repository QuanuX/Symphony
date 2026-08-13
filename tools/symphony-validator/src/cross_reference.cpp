#include "cross_reference.hpp"
#include "evidence.hpp"
#include <filesystem>
#include <set>

namespace fs = std::filesystem;

CrossReferenceResult check_cross_references(const std::string& repo_path, const SkviCheckResult& skvi_result, const SclvCheckResult& sclv_result) {
    CrossReferenceResult result;
    result.success = true;

    std::size_t records = 0U;
    std::size_t occurrences = 0U;
    std::set<std::string> unique_paths;
    for (const auto& rec : sclv_result.records) {
        if (!rec.affected_surfaces.empty()) {
            ++records;
        }
        for (const auto& surface : rec.affected_surfaces) {
            ++occurrences;
            unique_paths.insert(surface);
        }
    }

    const std::set<std::string> indexed_paths(skvi_result.indexed_paths.begin(), skvi_result.indexed_paths.end());
    std::size_t present_paths = 0U;
    std::size_t absent_paths = 0U;
    std::size_t unknown_paths = 0U;
    std::size_t currently_indexed_paths = 0U;
    for (const auto& surface : unique_paths) {
        std::error_code error;
        const auto status = fs::symlink_status(fs::path(repo_path) / surface, error);
        if (error == std::errc::no_such_file_or_directory || error == std::errc::not_a_directory) {
            ++absent_paths;
        } else if (error) {
            ++unknown_paths;
        } else if (fs::exists(status)) {
            ++present_paths;
        } else {
            ++absent_paths;
        }
        if (indexed_paths.contains(surface)) {
            ++currently_indexed_paths;
        }
    }

    result.messages.push_back(format_evidence(
        EvidenceCategory::Pass,
        "sclv.affected_surface.provenance_summary",
        "records=" + std::to_string(records) +
            " occurrences=" + std::to_string(occurrences) +
            " unique_paths=" + std::to_string(unique_paths.size()) +
            " present_paths=" + std::to_string(present_paths) +
            " absent_paths=" + std::to_string(absent_paths) +
            " unknown_paths=" + std::to_string(unknown_paths) +
            " indexed_paths=" + std::to_string(currently_indexed_paths) +
            " unindexed_paths=" + std::to_string(unique_paths.size() - currently_indexed_paths)));

    return result;
}
