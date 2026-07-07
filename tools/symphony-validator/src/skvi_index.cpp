#include "skvi_index.hpp"
#include <fstream>
#include <iostream>
#include <sstream>

static std::string trim_list_prefix(const std::string& line) {
    size_t start = 0;
    while (start < line.size() && (line[start] == ' ' || line[start] == '\t' || line[start] == '-')) {
        start++;
    }
    return line.substr(start);
}

static std::string extract_value(const std::string& line, const std::string& prefix) {
    std::string trimmed = trim_list_prefix(line);
    if (trimmed.find(prefix) == 0) {
        std::string val = trimmed.substr(prefix.size());
        size_t start = val.find_first_not_of(" \t\r\n`\"");
        size_t end = val.find_last_not_of(" \t\r\n`\"");
        if (start != std::string::npos && end != std::string::npos) {
            return val.substr(start, end - start + 1);
        }
    }
    return "";
}

SkviCheckResult check_skvi_index(const std::string& index_path) {
    SkviCheckResult result;
    result.success = true;

    std::ifstream file(index_path);
    if (!file.is_open()) {
        result.success = false;
        result.messages.push_back("evidence absent skvi.index.exists " + index_path + " not found");
        return result;
    }

    result.messages.push_back("evidence pass skvi.index.exists " + index_path + " exists");

    std::string line;
    SkviEntry current_entry;
    bool in_entry = false;
    int entry_count = 0;

    auto validate_entry = [&](const SkviEntry& entry) {
        std::vector<std::string> missing;
        if (!entry.has_title) missing.push_back("title");
        if (!entry.has_surface_type) missing.push_back("surface_type");
        if (!entry.has_truth_role) missing.push_back("truth_role");
        if (!entry.has_owner) missing.push_back("owner");
        if (!entry.has_scope) missing.push_back("scope");
        if (!entry.has_relationships) missing.push_back("relationships");
        if (!entry.has_consumers) missing.push_back("consumers");
        if (!entry.has_deferred_projections) missing.push_back("deferred_projections");
        if (!entry.has_status) missing.push_back("status");
        if (!entry.has_notes) missing.push_back("notes");

        if (missing.empty()) {
            result.messages.push_back("evidence pass skvi.entry.shape path=" + entry.path);
            result.indexed_paths.push_back(entry.path);
        } else {
            result.success = false;
            for (const auto& m : missing) {
                result.messages.push_back("evidence violation skvi.entry.missing_field path=" + entry.path + " field=" + m);
            }
        }
    };

    while (std::getline(file, line)) {
        std::string trimmed = trim_list_prefix(line);
        if (trimmed.find("path:") == 0) {
            if (in_entry) {
                validate_entry(current_entry);
            }
            in_entry = true;
            entry_count++;
            current_entry = SkviEntry();
            current_entry.path = extract_value(line, "path:");
        } else if (in_entry) {
            if (trimmed.find("title:") == 0) current_entry.has_title = true;
            else if (trimmed.find("surface_type:") == 0) current_entry.has_surface_type = true;
            else if (trimmed.find("truth_role:") == 0) current_entry.has_truth_role = true;
            else if (trimmed.find("owner:") == 0) current_entry.has_owner = true;
            else if (trimmed.find("scope:") == 0) current_entry.has_scope = true;
            else if (trimmed.find("status:") == 0) {
                current_entry.has_status = true;
                current_entry.status = extract_value(line, "status:");
            }
            else if (trimmed.find("relationships:") == 0) current_entry.has_relationships = true;
            else if (trimmed.find("consumers:") == 0) current_entry.has_consumers = true;
            else if (trimmed.find("deferred_projections:") == 0) current_entry.has_deferred_projections = true;
            else if (trimmed.find("notes:") == 0) current_entry.has_notes = true;
        }
    }

    if (in_entry) {
        validate_entry(current_entry);
        result.entries.push_back(current_entry);
    }

    if (entry_count == 0) {
        result.success = false;
        result.messages.push_back("evidence violation skvi.entry.count no SKVI entries detected");
    }

    return result;
}
