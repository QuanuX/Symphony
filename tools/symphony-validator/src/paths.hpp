#pragma once
#include <string>
#include "evidence.hpp"

struct PathCheckResult {
    bool is_valid_directory;
    EvidenceCategory evidence;
    std::string message;
};

PathCheckResult check_repository_path(const std::string& path_str);
