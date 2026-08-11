#pragma once

#include <string>
#include <vector>

std::string project_validation_result(
    const std::string& repository_path,
    const std::vector<std::string>& messages,
    int exit_code);
