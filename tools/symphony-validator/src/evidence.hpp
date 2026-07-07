#pragma once
#include <string>

enum class EvidenceCategory {
    Pass,
    Warning,
    Violation,
    Deferred,
    Absent,
    Stale,
    Unknown,
    Blocked
};

std::string evidence_category_to_string(EvidenceCategory category);
