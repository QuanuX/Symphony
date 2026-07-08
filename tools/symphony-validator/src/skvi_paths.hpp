#pragma once
#include <string>
#include <vector>
#include "skvi_index.hpp"

struct SkviPathsCheckResult {
    bool success;
    std::vector<std::string> messages;
};

SkviPathsCheckResult check_skvi_paths(const std::string& repo_root, const SkviCheckResult& skvi_result);
