#include "paths.hpp"
#include <filesystem>
#include <iostream>

PathCheckResult check_repository_path(const std::string& path_str) {
    std::filesystem::path p(path_str);
    
    if (!std::filesystem::exists(p)) {
        return {false, EvidenceCategory::Absent, "path absent"};
    }
    
    if (!std::filesystem::is_directory(p)) {
        return {false, EvidenceCategory::Violation, "path not directory"};
    }
    
    return {true, EvidenceCategory::Pass, "valid directory"};
}
