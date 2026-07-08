#pragma once

#include <string>
#include <vector>
#include "skvi_index.hpp"
#include "sclv_changelog.hpp"

struct SclvSkviReferencesCheckResult {
    bool success;
    std::vector<std::string> messages;
};

SclvSkviReferencesCheckResult check_sclv_skvi_references(const SkviCheckResult& skvi_result, const SclvCheckResult& sclv_result);
