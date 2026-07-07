#pragma once
#include <string>
#include <vector>
#include "skvi_index.hpp"
#include "sclv_changelog.hpp"

struct VocabularyCheckResult {
    bool success;
    std::vector<std::string> messages;
};

VocabularyCheckResult check_vocabulary(const SkviCheckResult& skvi_result, const SclvCheckResult& sclv_result);
