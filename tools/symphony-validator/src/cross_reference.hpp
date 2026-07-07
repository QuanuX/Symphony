#pragma once
#include <string>
#include <vector>
#include "skvi_index.hpp"
#include "sclv_changelog.hpp"

struct CrossReferenceResult {
    bool success;
    std::vector<std::string> messages;
};

CrossReferenceResult check_cross_references(const std::string& repo_path, const SkviCheckResult& skvi_result, const SclvCheckResult& sclv_result);
