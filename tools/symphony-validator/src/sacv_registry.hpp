#pragma once

#include "skvi_index.hpp"

#include <string>
#include <vector>

struct SacvRegistryCheckResult {
    bool success;
    std::vector<std::string> messages;
    std::size_t entries_checked;
};

SacvRegistryCheckResult check_sacv_registry(
    const std::string& repo_root,
    const SkviCheckResult& skvi_result);
