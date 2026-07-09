#pragma once
#include <string>
#include <vector>

struct ValidatorBuildCheckResult {
    bool success;
    std::vector<std::string> messages;
};

ValidatorBuildCheckResult check_validator_build(const std::string& repo_root);
