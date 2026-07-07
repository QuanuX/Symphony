#pragma once
#include <string>
#include <vector>

struct SkviEntry {
    std::string path;
    bool has_title = false;
    bool has_surface_type = false;
    bool has_truth_role = false;
    bool has_owner = false;
    bool has_scope = false;
    bool has_status = false;
    
    // Optional fields just for tracking if needed
    bool has_relationships = false;
    bool has_consumers = false;
    bool has_deferred_projections = false;
    bool has_notes = false;
};

struct SkviCheckResult {
    bool success;
    std::vector<std::string> messages;
};

SkviCheckResult check_skvi_index(const std::string& index_path);
