#pragma once

#include "sodv_releases.hpp"

#include <string>
#include <vector>

struct RootSummaryResult {
    bool success;
    std::vector<std::string> messages;
    std::string projection_json;
    std::string projection_markdown;
};

bool root_summary_is_selected(const std::string& repo_root);

RootSummaryResult project_root_summary(
    const std::string& repo_root,
    const SodvReleaseCheckResult& releases);

RootSummaryResult check_root_summary(
    const std::string& repo_root,
    const SodvReleaseCheckResult& releases);
