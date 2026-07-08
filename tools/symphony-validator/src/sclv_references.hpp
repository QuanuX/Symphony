#pragma once
#include <string>
#include <vector>
#include "sclv_changelog.hpp"

struct SclvReferencesCheckResult {
    bool success;
    std::vector<std::string> messages;
};

SclvReferencesCheckResult check_sclv_references(const std::string& repo_root, const SclvCheckResult& sclv_result);
