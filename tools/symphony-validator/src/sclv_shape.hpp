#pragma once
#include <string>
#include <vector>
#include "sclv_changelog.hpp"

struct SclvShapeCheckResult {
    bool success;
    std::vector<std::string> messages;
};

SclvShapeCheckResult check_sclv_shapes(const SclvCheckResult& sclv_result);
