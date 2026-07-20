#pragma once
#include <string>
#include <vector>

struct CallerAuthorityCheckResult {
    bool success;
    std::vector<std::string> messages;
};

CallerAuthorityCheckResult check_caller_authority(const std::string& repo_root);
