#include "sacv_registry.hpp"

#include "evidence.hpp"

#include <algorithm>
#include <array>
#include <cctype>
#include <filesystem>
#include <fstream>
#include <map>
#include <set>
#include <string>
#include <string_view>
#include <vector>

namespace fs = std::filesystem;

namespace {

constexpr std::size_t max_entries = 256;
constexpr std::size_t max_file_bytes = 4U * 1024U * 1024U;
constexpr std::array<const char*, 13> fields = {
    "api_id", "title", "owner", "path", "openapi", "api_version", "audience",
    "transport_profile", "security_profile", "publication_state", "sdk_state", "status", "notes",
};

struct Entry {
    std::map<std::string, std::string> values;
    std::size_t line = 0;
};

std::string trim(std::string_view value) {
    std::size_t begin = 0;
    while (begin < value.size() && std::isspace(static_cast<unsigned char>(value[begin])) != 0) {
        ++begin;
    }
    std::size_t end = value.size();
    while (end > begin && std::isspace(static_cast<unsigned char>(value[end - 1U])) != 0) {
        --end;
    }
    return std::string(value.substr(begin, end - begin));
}

std::string normalize(std::string_view value) {
    std::size_t begin = 0;
    while (begin < value.size() && (value[begin] == ' ' || value[begin] == '\t' || value[begin] == '-')) {
        ++begin;
    }
    return trim(value.substr(begin));
}

std::string clean(std::string value) {
    value = trim(value);
    if (value.size() >= 2U && value.front() == '`' && value.back() == '`') {
        value = value.substr(1U, value.size() - 2U);
    }
    return value;
}

std::string detected_field(const std::string& value) {
    for (const auto* field : fields) {
        if (value.starts_with(std::string(field) + ':')) {
            return field;
        }
    }
    return {};
}

bool safe_token(const std::string& value) {
    if (value.empty() || value.size() > 128U) {
        return false;
    }
    return std::all_of(value.begin(), value.end(), [](const unsigned char character) {
        return (character >= 'a' && character <= 'z') ||
               (character >= '0' && character <= '9') ||
               character == '.' || character == '_' || character == '-';
    });
}

bool generic_token(const std::string& value) {
    if (value.empty() || value.size() > 128U) {
        return false;
    }
    return std::all_of(value.begin(), value.end(), [](const unsigned char character) {
        return (character >= 'a' && character <= 'z') ||
               (character >= 'A' && character <= 'Z') ||
               (character >= '0' && character <= '9') ||
               character == '.' || character == '_' || character == ':' || character == '-';
    });
}

bool printable_bounded(const std::string& value) {
    return !value.empty() && value.size() <= 65536U &&
           std::all_of(value.begin(), value.end(), [](const unsigned char character) {
               return character == '\n' || character == '\t' ||
                      (character >= 0x20U && character != 0x7fU);
           });
}

bool safe_relative_path(const std::string& value) {
    if (value.empty() || value.size() > 4096U || value.front() == '/' ||
        value.find('\\') != std::string::npos || value.find("//") != std::string::npos) {
        return false;
    }
    std::size_t position = 0;
    while (position <= value.size()) {
        const auto end = value.find('/', position);
        const auto component = value.substr(position, end == std::string::npos ? std::string::npos : end - position);
        if (component.empty() || component == "." || component == ".." ||
            std::any_of(component.begin(), component.end(), [](const unsigned char character) {
                return character < 0x20U || character == 0x7fU;
            })) {
            return false;
        }
        if (end == std::string::npos) {
            break;
        }
        position = end + 1U;
    }
    return true;
}

bool owner_path(const std::string& value) {
    if (!safe_relative_path(value)) {
        return false;
    }
    const auto slash = value.find('/');
    return slash != std::string::npos && value.find('/', slash + 1U) == std::string::npos &&
           (value.starts_with("knowledge/") || value.starts_with("modules/"));
}

bool canonical_profile_path(const std::string& value) {
    if (!safe_relative_path(value) ||
        (!value.starts_with("knowledge/") && !value.starts_with("modules/"))) {
        return false;
    }
    const auto first = value.find('/');
    return first != std::string::npos && value.find('/', first + 1U) != std::string::npos;
}

bool bounded_regular_file_no_follow(const fs::path& path) {
    std::error_code error;
    const auto status = fs::symlink_status(path, error);
    if (error || !fs::is_regular_file(status) || fs::is_symlink(status)) {
        return false;
    }
    const auto size = fs::file_size(path, error);
    return !error && size <= max_file_bytes;
}

bool one_of(const std::string& value, std::initializer_list<const char*> options) {
    return std::any_of(options.begin(), options.end(), [&](const char* option) { return value == option; });
}

std::vector<Entry> parse_entries(const std::string& content, SacvRegistryCheckResult& result) {
    std::vector<Entry> entries;
    Entry current;
    bool active = false;
    std::string active_field;
    std::size_t next_field = 0;
    std::size_t line_number = 0;
    std::size_t position = 0;
    auto finish = [&]() {
        if (active) {
            entries.push_back(std::move(current));
            current = Entry{};
            active = false;
            active_field.clear();
            next_field = 0;
        }
    };
    while (position <= content.size()) {
        const auto end = content.find('\n', position);
        const auto line = content.substr(position, end == std::string::npos ? std::string::npos : end - position);
        ++line_number;
        const auto normalized = normalize(line);
        const auto field = detected_field(normalized);
        if (field == "api_id") {
            finish();
            active = true;
            current.line = line_number;
        }
        if (active && !field.empty()) {
            const auto position_in_contract = static_cast<std::size_t>(
                std::distance(fields.begin(), std::find(fields.begin(), fields.end(), field)));
            if (position_in_contract != next_field) {
                result.success = false;
                result.messages.push_back(format_evidence(EvidenceCategory::Violation,
                    "sacv.registry.field_order_invalid", "line=" + std::to_string(line_number) + " field=" + field));
            }
            next_field = position_in_contract + 1U;
            if (current.values.contains(field)) {
                result.success = false;
                result.messages.push_back(format_evidence(EvidenceCategory::Violation,
                    "sacv.registry.duplicate_field", "line=" + std::to_string(line_number) + " field=" + field));
            } else {
                current.values[field] = clean(normalized.substr(normalized.find(':') + 1U));
            }
            active_field = field;
        } else if (active && !normalized.empty() && normalized.find(':') != std::string::npos &&
                   trim(line).starts_with('-')) {
            result.success = false;
            result.messages.push_back(format_evidence(EvidenceCategory::Violation,
                "sacv.registry.unknown_field", "line=" + std::to_string(line_number)));
            active_field.clear();
        } else if (active && !active_field.empty() && !normalized.empty() && !normalized.starts_with('#')) {
            auto value = clean(normalized);
            if (!value.empty()) {
                auto& target = current.values[active_field];
                if (!target.empty()) {
                    target.push_back('\n');
                }
                target += value;
            }
        } else if (normalized.starts_with('#')) {
            active_field.clear();
        }
        if (end == std::string::npos) {
            break;
        }
        position = end + 1U;
    }
    finish();
    if (entries.size() > max_entries) {
        result.success = false;
        result.messages.push_back(format_evidence(EvidenceCategory::Violation,
            "sacv.registry.entry_limit", "entries=" + std::to_string(entries.size()) + " max=256"));
    }
    return entries;
}

}

SacvRegistryCheckResult check_sacv_registry(
    const std::string& repo_root,
    const SkviCheckResult& skvi_result) {
    SacvRegistryCheckResult result{true, {}, 0};
    const fs::path root(repo_root);
    const auto registry = root / "knowledge/sacv/REGISTRY.md";
    std::error_code error;
    const auto status = fs::symlink_status(registry, error);
    if (error || !fs::is_regular_file(status) || fs::is_symlink(status)) {
        if (!fs::exists(root / "go.work")) {
            result.messages.push_back(format_evidence(EvidenceCategory::Pass,
                "sacv.registry.fixture_not_applicable", "legacy minimal validator fixture"));
            return result;
        }
        result.success = false;
        result.messages.push_back(format_evidence(EvidenceCategory::Violation,
            "sacv.registry.unreadable", "path=knowledge/sacv/REGISTRY.md"));
        return result;
    }
    const auto size = fs::file_size(registry, error);
    if (error || size > max_file_bytes) {
        result.success = false;
        result.messages.push_back(format_evidence(EvidenceCategory::Violation,
            "sacv.registry.size", "path=knowledge/sacv/REGISTRY.md"));
        return result;
    }
    std::ifstream input(registry, std::ios::binary);
    const std::string content((std::istreambuf_iterator<char>(input)), std::istreambuf_iterator<char>());
    auto entries = parse_entries(content, result);
    result.entries_checked = entries.size();
    if (entries.empty()) {
        if (content.find("## Canonical Entries") == std::string::npos || content.find("None.") == std::string::npos) {
            result.success = false;
            result.messages.push_back(format_evidence(EvidenceCategory::Violation,
                "sacv.registry.empty_marker_missing", "path=knowledge/sacv/REGISTRY.md"));
        } else {
            result.messages.push_back(format_evidence(EvidenceCategory::Pass,
                "sacv.registry.empty_valid", "entries=0"));
        }
        return result;
    }

    const std::set<std::string> indexed(skvi_result.indexed_paths.begin(), skvi_result.indexed_paths.end());
    std::set<std::string> ids;
    std::set<std::string> paths;
    for (const auto& entry : entries) {
        std::string identity = "unavailable";
        if (const auto found = entry.values.find("api_id"); found != entry.values.end()) {
            identity = found->second;
        }
        bool shape = true;
        for (const auto* field : fields) {
            const auto found = entry.values.find(field);
            if (found == entry.values.end() || !printable_bounded(found->second)) {
                shape = false;
                result.success = false;
                result.messages.push_back(format_evidence(EvidenceCategory::Violation,
                    "sacv.registry.field_invalid", "api_id=" + identity + " line=" +
                    std::to_string(entry.line) + " field=" + field));
            }
        }
        if (!shape) {
            continue;
        }
        const auto& values = entry.values;
        const auto& path = values.at("path");
        if (!safe_token(values.at("api_id")) || !ids.insert(values.at("api_id")).second) {
            result.success = false;
            result.messages.push_back(format_evidence(EvidenceCategory::Violation,
                "sacv.registry.api_id_invalid", "api_id=" + identity));
        }
        if (!paths.insert(path).second) {
            result.success = false;
            result.messages.push_back(format_evidence(EvidenceCategory::Violation,
                "sacv.registry.path_duplicate", "path=" + path));
        }
        if (!owner_path(values.at("owner")) || !safe_relative_path(path) ||
            (!path.ends_with(".openapi.json") && !path.ends_with(".openapi.yaml")) ||
            !path.starts_with(values.at("owner") + "/")) {
            result.success = false;
            result.messages.push_back(format_evidence(EvidenceCategory::Violation,
                "sacv.registry.ownership_invalid", "api_id=" + identity + " path=" + path));
        }
        if (values.at("openapi") != "3.2.0" ||
            !one_of(values.at("audience"), {"local_internal", "remote_administrative", "partner", "public"}) ||
            !one_of(values.at("publication_state"), {"internal_only", "candidate", "sodv_approved"}) ||
            !one_of(values.at("sdk_state"), {"not_eligible", "candidate", "approved"}) ||
            !one_of(values.at("status"), {"draft", "ratified", "deprecated", "retired"}) ||
            !generic_token(values.at("api_version")) || !generic_token(values.at("transport_profile"))) {
            result.success = false;
            result.messages.push_back(format_evidence(EvidenceCategory::Violation,
                "sacv.registry.classification_invalid", "api_id=" + identity));
        }
        const auto& security_profile = values.at("security_profile");
        if (!canonical_profile_path(security_profile) || !indexed.contains(security_profile)) {
            result.success = false;
            result.messages.push_back(format_evidence(EvidenceCategory::Violation,
                "sacv.registry.security_profile_invalid", "api_id=" + identity +
                " security_profile=" + security_profile));
        } else {
            if (!bounded_regular_file_no_follow(root / security_profile)) {
                result.success = false;
                result.messages.push_back(format_evidence(EvidenceCategory::Violation,
                    "sacv.registry.security_profile_unreadable", "api_id=" + identity +
                    " security_profile=" + security_profile));
            }
        }
        if (!indexed.contains(path)) {
            result.success = false;
            result.messages.push_back(format_evidence(EvidenceCategory::Violation,
                "sacv.registry.skvi_unindexed", "api_id=" + identity + " path=" + path));
        }
        if (!bounded_regular_file_no_follow(root / path)) {
            result.success = false;
            result.messages.push_back(format_evidence(EvidenceCategory::Violation,
                "sacv.registry.document_unreadable", "api_id=" + identity + " path=" + path));
        }
        result.messages.push_back(format_evidence(EvidenceCategory::Pass,
            "sacv.registry.entry_checked", "api_id=" + identity + " path=" + path));
    }
    result.messages.push_back(format_evidence(EvidenceCategory::Pass,
        "sacv.registry.scan_complete", "entries=" + std::to_string(entries.size())));
    return result;
}
