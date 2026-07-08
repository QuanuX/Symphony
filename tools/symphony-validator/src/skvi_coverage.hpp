#pragma once
#include <string>
#include <vector>
#include "skvi_index.hpp"

struct SkviCoverageCheckResult {
    bool success;
    std::vector<std::string> messages;
};

SkviCoverageCheckResult check_skvi_coverage(const SkviCheckResult& index_res);
