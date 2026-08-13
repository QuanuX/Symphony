#include "sclv_references.hpp"
#include "evidence.hpp"
#include <filesystem>

namespace {

bool invalid_relative_path(const std::string& path) {
    if (path.empty() || path.front() == '/') {
        return true;
    }
    for (const auto& component : std::filesystem::path(path)) {
        if (component == "..") {
            return true;
        }
    }
    return false;
}

}

SclvReferencesCheckResult check_sclv_references(const std::string& repo_root, const SclvCheckResult& sclv_result) {
    SclvReferencesCheckResult result;
    result.success = true;

    for (const auto& rec : sclv_result.records) {
        // affected_surfaces are immutable historical provenance. Their bounded path
        // shape is checked here (and by the v3 shape gate), while their current state
        // is summarized by cross_reference; they are not filesystem obligations.
        for (const auto& path : rec.affected_surfaces) {
            if (invalid_relative_path(path)) {
                result.success = false;
                result.messages.push_back(format_evidence(
                    EvidenceCategory::Violation,
                    "sclv_reference.invalid_relative_path",
                    "record_id=" + rec.record_id + " field=affected_surfaces path=" + path));
            }
        }

        for (const auto& path : rec.skvi_references) {
            std::string field = "skvi_references";

            if (invalid_relative_path(path)) {
                result.success = false;
                result.messages.push_back(format_evidence(EvidenceCategory::Violation, "sclv_reference.invalid_relative_path", "record_id=" + rec.record_id + " field=" + field + " path=" + path));
                continue;
            }

            std::filesystem::path full_path = std::filesystem::path(repo_root) / path;
            
            if (!std::filesystem::exists(full_path)) {
                result.success = false;
                result.messages.push_back(format_evidence(EvidenceCategory::Violation, "sclv_reference.path_missing", "record_id=" + rec.record_id + " field=" + field + " path=" + path));
            } else if (!std::filesystem::is_regular_file(full_path)) {
                result.success = false;
                result.messages.push_back(format_evidence(EvidenceCategory::Violation, "sclv_reference.path_not_file", "record_id=" + rec.record_id + " field=" + field + " path=" + path));
            } else {
                result.messages.push_back(format_evidence(EvidenceCategory::Pass, "sclv_reference.skvi_reference_exists", "record_id=" + rec.record_id + " path=" + path));
            }
        }
    }

    return result;
}
