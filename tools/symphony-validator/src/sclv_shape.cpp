#include "sclv_shape.hpp"

#include "evidence.hpp"

#include <algorithm>
#include <cctype>
#include <set>
#include <string>
#include <string_view>

namespace {

bool lower_hex(const std::string& value, std::size_t length) {
    return value.size() == length && std::all_of(value.begin(), value.end(), [](const unsigned char character) {
        return (character >= '0' && character <= '9') || (character >= 'a' && character <= 'f');
    });
}

bool tagged_digest(const std::string& value) {
    return value.starts_with("sha256:") && lower_hex(value.substr(7U), 64U);
}

bool bounded_text(const std::string& value, std::size_t maximum = 4096U) {
    if (value.empty() || value.size() > maximum) return false;
    return std::all_of(value.begin(), value.end(), [](const unsigned char character) {
        return character >= 0x20U && character != 0x7fU;
    });
}

bool token(const std::string& value) {
    if (value.empty() || value.size() > 128U) return false;
    return std::all_of(value.begin(), value.end(), [](const unsigned char character) {
        return std::isalnum(character) || character == '.' || character == '_' ||
               character == ':' || character == '-';
    });
}

bool long_text(const std::string& value) {
    if (value.empty() || value == "|" || value.size() > 65536U) return false;
    return std::all_of(value.begin(), value.end(), [](const unsigned char character) {
        return character == '\n' || (character >= 0x20U && character != 0x7fU);
    });
}

bool safe_relative_path(const std::string& value) {
    if (value.empty() || value.size() > 4096U || value.front() == '/' ||
        value.find('\\') != std::string::npos || value.find("//") != std::string::npos) return false;
    std::size_t position = 0;
    while (position <= value.size()) {
        const auto end = value.find('/', position);
        const auto component = std::string_view(value).substr(
            position, end == std::string::npos ? std::string::npos : end - position);
        if (component.empty() || component == "." || component == ".." ||
            !std::all_of(component.begin(), component.end(), [](const unsigned char character) {
                return character >= 0x20U && character != 0x7fU;
            })) return false;
        if (end == std::string::npos) break;
        position = end + 1U;
    }
    return true;
}

bool bounded_unique_list(const std::vector<std::string>& values, bool paths) {
    if (values.empty() || values.size() > 1024U) return false;
    std::set<std::string> unique;
    for (const auto& value : values) {
        if ((paths && !safe_relative_path(value)) || (!paths && !bounded_text(value)) ||
            !unique.insert(value).second) return false;
    }
    return true;
}

bool v3_record_id(const std::string& value) {
    constexpr const char* prefix = "SCLV-CHG-";
    if (!value.starts_with(prefix) || value.size() < 17U || value.size() > 105U) return false;
    return token(value.substr(9U));
}

void pass(SclvShapeCheckResult& result, const std::string& code, const SclvRecord& record) {
    result.messages.push_back(format_evidence(EvidenceCategory::Pass, code, "record_id=" + record.record_id));
}

void violation(SclvShapeCheckResult& result, const std::string& code, const SclvRecord& record, const std::string& detail) {
    result.success = false;
    result.messages.push_back(format_evidence(EvidenceCategory::Violation, code, "record_id=" + record.record_id + " " + detail));
}

void check_v3(SclvShapeCheckResult& result, const SclvRecord& record) {
    if (!v3_record_id(record.record_id)) violation(result, "sclv.v3.record_id_invalid", record, "value=" + record.record_id);
    else pass(result, "sclv.v3.record_id_valid", record);

    if (record.status != "canonical") violation(result, "sclv.v3.status_invalid", record, "status=" + record.status);
    if (!bounded_text(record.title) || !token(record.change_type)) violation(result, "sclv.v3.text_invalid", record, "title_or_change_type");

    const bool request_present = record.change_request_state == "present";
    const bool request_absent = record.change_request_state == "not_applicable";
    if (!request_present && !request_absent) {
        violation(result, "sclv.v3.change_request_state_invalid", record, "state=" + record.change_request_state);
    } else if (request_present) {
        if (!token(record.change_request_provider) || !bounded_text(record.change_request_id) ||
            !bounded_text(record.change_request_reference) || record.change_request_absence_reason != "not_applicable") {
            violation(result, "sclv.v3.change_request_invalid", record, "present_state_fields_invalid");
        } else {
            pass(result, "sclv.v3.change_request_valid", record);
        }
    } else if (record.change_request_provider != "not_applicable" ||
               record.change_request_id != "not_applicable" ||
               record.change_request_reference != "not_applicable" ||
               !bounded_text(record.change_request_absence_reason) ||
               record.change_request_absence_reason == "not_applicable") {
        violation(result, "sclv.v3.change_request_invalid", record, "not_applicable_state_fields_invalid");
    } else {
        pass(result, "sclv.v3.change_request_valid", record);
    }

    bool revision_valid = token(record.revision_scheme) && bounded_text(record.revision_value);
    if (record.revision_scheme == "git-sha1") revision_valid = lower_hex(record.revision_value, 40U);
    if (record.revision_scheme == "git-sha256") revision_valid = lower_hex(record.revision_value, 64U);
    if (!revision_valid || !tagged_digest(record.tree_digest)) {
        violation(result, "sclv.v3.revision_invalid", record, "revision_or_tree_digest");
    } else {
        pass(result, "sclv.v3.revision_valid", record);
    }

    if (!bounded_text(record.ratification_subject) || !bounded_text(record.ratification_permission) ||
        !bounded_text(record.ratification_method) || !bounded_text(record.ratification_evidence_reference) ||
        record.ratification_subject == "not_applicable" ||
        record.ratification_permission == "not_applicable" ||
        record.ratification_method == "not_applicable" ||
        record.ratification_evidence_reference == "not_applicable" ||
        !tagged_digest(record.ratification_evidence_digest)) {
        violation(result, "sclv.v3.ratification_invalid", record, "ratification_evidence");
    } else {
        pass(result, "sclv.v3.ratification_bound", record);
    }

    if (record.recording_disposition == "post_merge" && record.recovery_reason != "not_applicable") {
        violation(result, "sclv.v3.recovery_reason_invalid", record, "post_merge_requires_not_applicable");
    } else if (record.recording_disposition == "late_recovery" &&
               (!bounded_text(record.recovery_reason) || record.recovery_reason == "not_applicable")) {
        violation(result, "sclv.v3.recovery_reason_invalid", record, "late_recovery_requires_reason");
    }

    if (record.has_related_pr || record.has_merge_commit) {
        violation(result, "sclv.v3.legacy_provider_field", record, "related_pr_or_merge_commit");
    }

    const bool paths_valid = bounded_unique_list(record.affected_surfaces, true) &&
        bounded_unique_list(record.skvi_references, true);
    const bool evidence_valid = bounded_unique_list(record.evidence, false) &&
        bounded_unique_list(record.non_authorizations, false);
    const bool narrative_valid = long_text(record.change_summary) &&
        long_text(record.relationship_changes) && long_text(record.doctrine_changes) &&
        long_text(record.compatibility_consequences) && long_text(record.publication_consequences) &&
        long_text(record.projection_consequences) && long_text(record.notes);
    if (!paths_valid || !evidence_valid || !narrative_valid) {
        violation(result, "sclv.v3.content_invalid", record, "path_list_evidence_or_narrative");
    } else {
        pass(result, "sclv.v3.content_valid", record);
    }
}

}

bool is_valid_related_pr(const std::string& pr) {
    const std::string prefix = "https://github.com/QuanuX/Symphony/pull/";
    if (pr.size() <= prefix.size() || !pr.starts_with(prefix)) return false;
    return std::all_of(pr.begin() + static_cast<std::ptrdiff_t>(prefix.size()), pr.end(), [](const unsigned char character) {
        return std::isdigit(character);
    });
}

bool is_valid_merge_commit(const std::string& commit) {
    return commit.size() == 40U && std::all_of(commit.begin(), commit.end(), [](const unsigned char character) {
        return std::isxdigit(character);
    });
}

SclvShapeCheckResult check_sclv_shapes(const SclvCheckResult& sclv_result) {
    SclvShapeCheckResult result{true, {}};
    for (const auto& record : sclv_result.records) {
        if (record.record_version == 3) {
            check_v3(result, record);
            continue;
        }
        if (record.has_related_pr) {
            if (is_valid_related_pr(record.related_pr)) {
                result.messages.push_back(format_evidence(EvidenceCategory::Pass, "sclv.related_pr.shape", "record_id=" + record.record_id + " related_pr=" + record.related_pr));
            } else {
                result.success = false;
                result.messages.push_back(format_evidence(EvidenceCategory::Violation, "sclv.related_pr.shape_invalid", "record_id=" + record.record_id + " related_pr=" + record.related_pr));
            }
        }
        if (record.has_merge_commit) {
            if (is_valid_merge_commit(record.merge_commit)) {
                result.messages.push_back(format_evidence(EvidenceCategory::Pass, "sclv.merge_commit.shape", "record_id=" + record.record_id + " merge_commit=" + record.merge_commit));
            } else {
                result.success = false;
                result.messages.push_back(format_evidence(EvidenceCategory::Violation, "sclv.merge_commit.shape_invalid", "record_id=" + record.record_id + " merge_commit=" + record.merge_commit));
            }
        }
    }
    return result;
}
