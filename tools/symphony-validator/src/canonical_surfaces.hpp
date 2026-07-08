#pragma once
#include <string>
#include <vector>

struct CanonicalSurfaceCheckResult {
    bool success;
    std::vector<std::string> messages;
};

std::vector<std::string> get_required_canonical_surfaces();
CanonicalSurfaceCheckResult check_required_canonical_surfaces(const std::string& repo_root);
