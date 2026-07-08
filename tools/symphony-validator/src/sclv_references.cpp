#include "sclv_references.hpp"
#include "evidence.hpp"
#include <filesystem>

SclvReferencesCheckResult check_sclv_references(const std::string& repo_root, const SclvCheckResult& sclv_result) {
    SclvReferencesCheckResult result;
    result.success = true;

    for (const auto& rec : sclv_result.records) {
        // affected_surfaces
        for (const auto& path : rec.affected_surfaces) {
            std::string field = "affected_surfaces";
            
            if (path.empty() || path[0] == '/' || path.find("..") != std::string::npos) {
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
                result.messages.push_back(format_evidence(EvidenceCategory::Pass, "sclv_reference.affected_surface_exists", "record_id=" + rec.record_id + " path=" + path));
            }
        }

        // skvi_references
        for (const auto& path : rec.skvi_references) {
            std::string field = "skvi_references";
            
            if (path.empty() || path[0] == '/' || path.find("..") != std::string::npos) {
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
