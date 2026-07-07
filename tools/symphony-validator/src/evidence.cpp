#include "evidence.hpp"

std::string evidence_category_to_string(EvidenceCategory category) {
    switch (category) {
        case EvidenceCategory::Pass: return "pass";
        case EvidenceCategory::Warning: return "warning";
        case EvidenceCategory::Violation: return "violation";
        case EvidenceCategory::Deferred: return "deferred";
        case EvidenceCategory::Absent: return "absent";
        case EvidenceCategory::Stale: return "stale";
        case EvidenceCategory::Unknown: return "unknown";
        case EvidenceCategory::Blocked: return "blocked";
    }
    return "unknown";
}
