#pragma once

#include <cstddef>
#include <string>
#include <vector>

struct SodvPublishedUnit {
    std::string coordinate;
    std::string version;
    std::string tag;
    std::string revision;
};

struct SodvReleaseCheckResult {
    bool success;
    std::vector<std::string> messages;
    std::size_t records_checked;
    std::size_t transactions_checked;
    std::vector<SodvPublishedUnit> published_units;
};

SodvReleaseCheckResult check_sodv_releases(const std::string& repo_root);
