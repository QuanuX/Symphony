#pragma once
#include <string>
#include <vector>

struct ArtifactCheckResult {
    bool success;
    std::vector<std::string> messages;
};

ArtifactCheckResult check_unauthorized_artifacts(const std::string& repo_root);
