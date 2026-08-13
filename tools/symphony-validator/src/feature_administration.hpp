#pragma once

#include <cstddef>
#include <string>
#include <vector>

struct FeatureAdministrationCheckResult {
    bool success;
    std::vector<std::string> messages;
    std::size_t features_checked;
    std::size_t commands_checked;
    std::size_t unreviewed_features;
    std::size_t violations;
};

FeatureAdministrationCheckResult check_feature_administration(const std::string& repo_root);
