#pragma once

#include <cstddef>
#include <string>
#include <vector>

struct SodvReleaseCheckResult {
    bool success;
    std::vector<std::string> messages;
    std::size_t records_checked;
    std::size_t transactions_checked;
};

SodvReleaseCheckResult check_sodv_releases(const std::string& repo_root);
