#include "vocabulary.hpp"
#include "evidence.hpp"
#include <unordered_set>
#include <iostream>

VocabularyCheckResult check_vocabulary(const SkviCheckResult& skvi_result, const SclvCheckResult& sclv_result) {
    VocabularyCheckResult result;
    result.success = true;

    // SKVI Status
    for (const auto& entry : skvi_result.entries) {
        if (entry.has_status) {
            if (entry.status == "canonical") {
                result.messages.push_back(format_evidence(EvidenceCategory::Pass, "skvi.status.valid", "path=" + entry.path + " status=" + entry.status));
            } else {
                result.success = false;
                result.messages.push_back(format_evidence(EvidenceCategory::Violation, "skvi.status.invalid", "path=" + entry.path + " status=" + entry.status));
            }
        }
    }

    std::unordered_set<std::string> valid_change_types = {
        "canonical_addition",
        "canonical_update",
        "canonical_removal",
        "doctrine_change",
        "namespace_change",
        "projection_change",
        "publication_boundary_change",
        "compatibility_boundary_change",
        "implementation_change",
        "tooling_change",
        "documentation_change",
        "backfill_record",
        "audit_record"
    };

    // SCLV Records
    for (const auto& rec : sclv_result.records) {
        if (rec.has_status) {
            if (rec.status == "canonical") {
                result.messages.push_back(format_evidence(EvidenceCategory::Pass, "sclv.status.valid", "record_id=" + rec.record_id + " status=" + rec.status));
            } else {
                result.success = false;
                result.messages.push_back(format_evidence(EvidenceCategory::Violation, "sclv.status.invalid", "record_id=" + rec.record_id + " status=" + rec.status));
            }
        }
        
        if (rec.has_change_type) {
            if (valid_change_types.count(rec.change_type)) {
                result.messages.push_back(format_evidence(EvidenceCategory::Pass, "sclv.change_type.valid", "record_id=" + rec.record_id + " change_type=" + rec.change_type));
            } else {
                result.success = false;
                result.messages.push_back(format_evidence(EvidenceCategory::Violation, "sclv.change_type.invalid", "record_id=" + rec.record_id + " change_type=" + rec.change_type));
            }
        }
    }

    return result;
}
