#include "root_summary.hpp"

#include "evidence.hpp"
#include "feature_administration.hpp"

#include <symphony/knowledge/engine/digest.hpp>
#include <symphony/knowledge/engine/error.hpp>
#include <symphony/knowledge/engine/json.hpp>
#include <symphony/knowledge/engine/path.hpp>

#include <algorithm>
#include <filesystem>
#include <fstream>
#include <map>
#include <set>
#include <sstream>
#include <string>
#include <string_view>
#include <vector>

namespace {

namespace fs = std::filesystem;
namespace engine = symphony::knowledge::engine;
using Json = engine::Json;

constexpr std::size_t maximum_file_bytes = 4U * 1024U * 1024U;
constexpr std::string_view begin_marker = "<!-- symphony:root-summary:v1:begin -->";
constexpr std::string_view end_marker = "<!-- symphony:root-summary:v1:end -->";

void finding(
    RootSummaryResult& result,
    const EvidenceCategory category,
    const std::string& rule,
    const std::string& detail) {
    result.messages.push_back(format_evidence(category, rule, detail));
    if (category == EvidenceCategory::Violation) {
        result.success = false;
    }
}

std::string trim(std::string_view value) {
    const auto first = value.find_first_not_of(" \t\r\n");
    if (first == std::string_view::npos) {
        return {};
    }
    const auto last = value.find_last_not_of(" \t\r\n");
    return std::string(value.substr(first, last - first + 1U));
}

std::string unquote_code(std::string value) {
    value = trim(value);
    if (value.size() >= 2U && value.front() == '`' && value.back() == '`') {
        return value.substr(1U, value.size() - 2U);
    }
    return value;
}

std::vector<std::string> table_cells(const std::string& line) {
    std::vector<std::string> cells;
    std::size_t cursor = 0U;
    while (cursor < line.size()) {
        const auto separator = line.find('|', cursor);
        if (separator == std::string::npos) {
            break;
        }
        const auto next = line.find('|', separator + 1U);
        if (next == std::string::npos) {
            break;
        }
        cells.push_back(trim(std::string_view(line).substr(separator + 1U, next - separator - 1U)));
        cursor = next;
    }
    return cells;
}

std::string read(
    const fs::path& root,
    const std::string& path,
    RootSummaryResult& result) {
    try {
        return engine::read_regular_file_no_follow(root, path, maximum_file_bytes);
    } catch (const engine::Error& error) {
        finding(result, EvidenceCategory::Violation, "root_summary.unreadable",
            "path=" + path + " code=" + error.code());
    } catch (const std::exception&) {
        finding(result, EvidenceCategory::Violation, "root_summary.unreadable",
            "path=" + path + " code=unexpected_error");
    }
    return {};
}

Json parse_json(
    const std::string& contents,
    const std::string& path,
    RootSummaryResult& result) {
    try {
        return Json::parse(contents);
    } catch (const std::exception&) {
        finding(result, EvidenceCategory::Violation, "root_summary.invalid_json",
            "path=" + path);
    }
    return Json{};
}

struct RegistryCounts {
    std::size_t total = 0U;
    std::map<std::string, std::string> feature_scopes;
};

RegistryCounts registry_counts(const std::string& contents, RootSummaryResult& result) {
    constexpr std::string_view feature_prefix = "- feature_id: `";
    constexpr std::string_view scope_prefix = "- source_scope: `";
    constexpr std::string_view parent_prefix = "- parent_feature_id: `";
    RegistryCounts counts;
    std::string current_feature;
    std::string current_scope;
    std::istringstream input(contents);
    std::string line;
    while (std::getline(input, line)) {
        if (line.starts_with(feature_prefix)) {
            if (!current_feature.empty()) {
                finding(result, EvidenceCategory::Violation, "root_summary.registry",
                    "reason=incomplete_record feature_id=" + current_feature);
            }
            if (line.size() <= feature_prefix.size() || line.back() != '`') {
                finding(result, EvidenceCategory::Violation, "root_summary.registry",
                    "reason=invalid_feature_id_line");
                continue;
            }
            ++counts.total;
            current_feature = line.substr(
                feature_prefix.size(), line.size() - feature_prefix.size() - 1U);
            current_scope.clear();
            continue;
        }
        if (!current_feature.empty() && line.starts_with(scope_prefix) && line.back() == '`') {
            current_scope = line.substr(scope_prefix.size(), line.size() - scope_prefix.size() - 1U);
            continue;
        }
        if (current_feature.empty() || !line.starts_with(parent_prefix) || line.back() != '`') {
            continue;
        }
        const auto parent = line.substr(parent_prefix.size(), line.size() - parent_prefix.size() - 1U);
        if (current_scope.empty() || (!parent.starts_with("ssfv:") && parent != "none") ||
            !counts.feature_scopes.emplace(current_feature, current_scope).second) {
            finding(result, EvidenceCategory::Violation, "root_summary.registry",
                "reason=invalid_or_duplicate_record feature_id=" + current_feature);
        }
        current_feature.clear();
        current_scope.clear();
    }
    if (!current_feature.empty()) {
        finding(result, EvidenceCategory::Violation, "root_summary.registry",
            "reason=incomplete_record feature_id=" + current_feature);
    }
    if (counts.total == 0U || counts.feature_scopes.size() != counts.total) {
        finding(result, EvidenceCategory::Violation, "root_summary.registry",
            "reason=empty_or_incomplete_feature_set");
    }
    return counts;
}

std::size_t ratified_nested_count(
    const std::string& coverage,
    RootSummaryResult& result) {
    std::size_t count = 0U;
    bool in_progress_table = false;
    std::istringstream input(coverage);
    std::string line;
    while (std::getline(input, line)) {
        if (line == "## Ratified Nested Review Progress") {
            in_progress_table = true;
            continue;
        }
        if (in_progress_table && line.starts_with("## ")) {
            break;
        }
        if (!in_progress_table || !line.starts_with("| `")) {
            continue;
        }
        const auto cells = table_cells(line);
        if (cells.size() < 2U) {
            continue;
        }
        try {
            std::size_t consumed = 0U;
            const auto value = std::stoull(cells[1], &consumed, 10);
            if (consumed != cells[1].size()) {
                continue;
            }
            count += value;
        } catch (const std::exception&) {
            continue;
        }
    }
    if (count == 0U) {
        finding(result, EvidenceCategory::Violation, "root_summary.coverage",
            "reason=no_ratified_nested_progress");
    }
    return count;
}

std::vector<std::string> registered_owner_features(
    const std::string& coverage,
    const RegistryCounts& registry,
    RootSummaryResult& result) {
    std::vector<std::string> features;
    std::set<std::string> unique;
    std::set<std::string> unique_scopes;
    std::istringstream input(coverage);
    std::string line;
    while (std::getline(input, line)) {
        if (!line.starts_with("| `")) {
            continue;
        }
        const auto cells = table_cells(line);
        if (cells.size() < 3U || cells[1] != "registered") {
            continue;
        }
        const auto scope = unquote_code(cells[0]);
        const auto feature = unquote_code(cells[2]);
        if (scope == ".") {
            continue;
        }
        const auto registered = registry.feature_scopes.find(feature);
        if (!feature.starts_with("ssfv:") || !unique.insert(feature).second ||
            !unique_scopes.insert(scope).second || registered == registry.feature_scopes.end() ||
            registered->second != scope) {
            finding(result, EvidenceCategory::Violation, "root_summary.coverage",
                "reason=invalid_duplicate_or_mismatched_registered_feature feature_id=" + feature +
                " source_scope=" + scope);
            continue;
        }
        features.push_back(feature);
    }
    std::ranges::sort(features);
    if (features.empty()) {
        finding(result, EvidenceCategory::Violation, "root_summary.coverage",
            "reason=no_registered_owner_features");
    }
    return features;
}

struct AdministrationCounts {
    std::size_t expectations = 0U;
    std::size_t required = 0U;
    std::size_t prohibited = 0U;
    std::size_t exemptions = 0U;
    std::size_t unreviewed = 0U;
};

AdministrationCounts administration_counts(const Json& profile, RootSummaryResult& result) {
    AdministrationCounts counts;
    if (!profile.is_object() || !profile.contains("features") || !profile.at("features").is_array()) {
        finding(result, EvidenceCategory::Violation, "root_summary.profile",
            "reason=missing_features");
        return counts;
    }
    for (const auto& feature : profile.at("features")) {
        if (!feature.is_object() || !feature.contains("expectations") ||
            !feature.at("expectations").is_array()) {
            finding(result, EvidenceCategory::Violation, "root_summary.profile",
                "reason=invalid_expectations");
            continue;
        }
        if (feature.at("expectations").empty()) {
            ++counts.unreviewed;
        }
        for (const auto& expectation : feature.at("expectations")) {
            if (!expectation.is_object()) {
                finding(result, EvidenceCategory::Violation, "root_summary.profile",
                    "reason=invalid_expectation");
                continue;
            }
            ++counts.expectations;
            const auto requirement = expectation.value("requirement", "");
            const auto delivery = expectation.value("delivery", "");
            if (requirement == "required") {
                ++counts.required;
            } else if (requirement == "prohibited") {
                ++counts.prohibited;
            }
            if (delivery == "unreviewed") {
                ++counts.unreviewed;
            }
            if (requirement == "not_applicable" || delivery == "runtime_only" ||
                delivery == "system_orchestrated") {
                ++counts.exemptions;
            }
        }
    }
    return counts;
}

std::string render_markdown(const Json& projection) {
    const auto& ssfv = projection.at("ssfv");
    const auto& administration = projection.at("feature_administration");
    std::ostringstream output;
    output << begin_marker << "\n"
           << "## Machine-Checked Repository Snapshot\n\n"
           << "This bounded summary is derived from canonical SSFV coverage and routing, the "
              "feature-administration profile, the qxctl command registry, and completed SODV "
              "publication records. Edit its source contracts, then regenerate; do not hand-edit "
              "the values below.\n\n"
           << "- SSFV catalog state: `" << ssfv.at("catalog_state").get<std::string>()
           << "`; registered features: **" << ssfv.at("registered_features").get<std::size_t>()
           << "**; registered owner scopes: **" << ssfv.at("registered_owner_scopes").get<std::size_t>()
           << "**; ratified nested features: **" << ssfv.at("nested_features").get<std::size_t>() << "**.\n"
           << "- Feature-administration expectations: **"
           << administration.at("expectations").get<std::size_t>()
           << "** reviewed surfaces; **" << administration.at("required").get<std::size_t>()
           << "** required, **" << administration.at("exemptions").get<std::size_t>()
           << "** evidence-backed exemptions, **" << administration.at("prohibited").get<std::size_t>()
           << "** prohibitions, **" << administration.at("unreviewed").get<std::size_t>()
           << "** unreviewed.\n"
           << "- qxctl stable command identities: **"
           << projection.at("qxctl").at("registered_commands").get<std::size_t>() << "**.\n"
           << "- Registered owner capabilities:\n";
    for (const auto& feature : ssfv.at("registered_owner_features")) {
        output << "  - `" << feature.get<std::string>() << "`\n";
    }
    output << "- Completed SODV source publications:\n";
    for (const auto& unit : projection.at("published_source_versions")) {
        output << "  - `" << unit.at("coordinate").get<std::string>() << "` `"
               << unit.at("version").get<std::string>() << "` (tag `"
               << unit.at("tag").get<std::string>() << "`, source `"
               << unit.at("revision").get<std::string>() << "`)\n";
    }
    output << "- Snapshot digest: `" << projection.at("summary_digest").get<std::string>() << "`\n"
           << end_marker << "\n";
    return output.str();
}

RootSummaryResult derive(
    const fs::path& root,
    const SodvReleaseCheckResult& releases) {
    RootSummaryResult result{true, {}, {}, {}};
    if (!releases.success) {
        finding(result, EvidenceCategory::Violation, "root_summary.release_source",
            "reason=sodv_validation_failed");
        return result;
    }
    const auto administration_check = check_feature_administration(root.string());
    if (!administration_check.success) {
        finding(result, EvidenceCategory::Violation, "root_summary.source_validation",
            "reason=feature_administration_failed");
        return result;
    }
    const auto registry = read(root, "knowledge/ssfv/REGISTRY.md", result);
    const auto coverage = read(root, "knowledge/ssfv/COVERAGE.md", result);
    const auto profile_contents = read(root, "knowledge/FEATURE-ADMINISTRATION-PROFILE.json", result);
    const auto command_contents = read(root, "tools/qxctl/COMMANDS.json", result);
    if (!result.success) {
        return result;
    }
    const auto profile = parse_json(profile_contents, "knowledge/FEATURE-ADMINISTRATION-PROFILE.json", result);
    const auto commands = parse_json(command_contents, "tools/qxctl/COMMANDS.json", result);
    if (!result.success || !commands.is_object() || !commands.contains("commands") ||
        !commands.at("commands").is_array()) {
        if (result.success) {
            finding(result, EvidenceCategory::Violation, "root_summary.commands",
                "reason=missing_commands");
        }
        return result;
    }
    const auto registry_evidence = registry_counts(registry, result);
    const auto owner_features = registered_owner_features(coverage, registry_evidence, result);
    const auto nested_features = ratified_nested_count(coverage, result);
    const auto administration = administration_counts(profile, result);
    if (!result.success) {
        return result;
    }
    if (registry_evidence.total != administration_check.features_checked ||
        commands.at("commands").size() != administration_check.commands_checked ||
        nested_features >= registry_evidence.total) {
        finding(result, EvidenceCategory::Violation, "root_summary.catalog_consistency",
            "registered=" + std::to_string(registry_evidence.total) +
            " owners=" + std::to_string(owner_features.size()) +
            " nested=" + std::to_string(nested_features));
        return result;
    }
    Json published = Json::array();
    std::set<std::string> published_keys;
    for (const auto& unit : releases.published_units) {
        const auto key = unit.coordinate + "\n" + unit.version;
        if (!published_keys.insert(key).second) {
            finding(result, EvidenceCategory::Violation, "root_summary.release_source",
                "reason=duplicate_completed_version coordinate=" + unit.coordinate +
                " version=" + unit.version);
            continue;
        }
        published.push_back(Json{{"coordinate", unit.coordinate}, {"revision", unit.revision},
            {"tag", unit.tag}, {"version", unit.version}});
    }
    if (!result.success || published.empty()) {
        if (result.success) {
            finding(result, EvidenceCategory::Violation, "root_summary.release_source",
                "reason=no_completed_publications");
        }
        return result;
    }
    Json owners = Json::array();
    for (const auto& feature : owner_features) {
        owners.push_back(feature);
    }
    Json projection{
        {"feature_administration", {
            {"exemptions", administration.exemptions},
            {"expectations", administration.expectations},
            {"prohibited", administration.prohibited},
            {"required", administration.required},
            {"unreviewed", administration.unreviewed},
        }},
        {"format_version", 1},
        {"protocol", "symphony.repository.root-summary.v1"},
        {"published_source_versions", published},
        {"qxctl", {{"registered_commands", commands.at("commands").size()}}},
        {"ssfv", {
            {"catalog_state", profile.value("catalog_complete", false) ? "complete" : "partial"},
            {"nested_features", nested_features},
            {"registered_features", registry_evidence.total},
            {"registered_owner_features", owners},
            {"registered_owner_scopes", owner_features.size()},
        }},
    };
    projection["summary_digest"] = engine::tagged_sha256(projection.dump());
    result.projection_json = projection.dump(2) + "\n";
    result.projection_markdown = render_markdown(projection);
    finding(result, EvidenceCategory::Pass, "root_summary.projected",
        "features=" + std::to_string(registry_evidence.total) +
        " commands=" + std::to_string(commands.at("commands").size()) +
        " publications=" + std::to_string(published.size()) +
        " digest=" + projection.at("summary_digest").get<std::string>());
    return result;
}

}

bool root_summary_is_selected(const std::string& repo_root) {
    const fs::path root(repo_root);
    std::error_code error;
    for (const auto* relative : {
            "knowledge/ssfv",
            "knowledge/FEATURE-ADMINISTRATION-PROFILE.json",
            "tools/qxctl/COMMANDS.json"}) {
        if (fs::exists(root / relative, error)) {
            return true;
        }
        error.clear();
    }

    std::ifstream readme(root / "README.md", std::ios::binary);
    if (!readme.good()) {
        return false;
    }
    const std::string contents(
        (std::istreambuf_iterator<char>(readme)), std::istreambuf_iterator<char>());
    return contents.find(begin_marker) != std::string::npos ||
        contents.find(end_marker) != std::string::npos;
}

RootSummaryResult project_root_summary(
    const std::string& repo_root,
    const SodvReleaseCheckResult& releases) {
    std::error_code error;
    const auto root = fs::canonical(fs::path(repo_root), error);
    if (error) {
        RootSummaryResult result{false, {}, {}, {}};
        finding(result, EvidenceCategory::Violation, "root_summary.unreadable",
            "path=. code=root_unreadable");
        return result;
    }
    try {
        return derive(root, releases);
    } catch (const std::exception&) {
        RootSummaryResult result{false, {}, {}, {}};
        finding(result, EvidenceCategory::Violation, "root_summary.internal",
            "reason=unexpected_projection_error");
        return result;
    }
}

RootSummaryResult check_root_summary(
    const std::string& repo_root,
    const SodvReleaseCheckResult& releases) {
    auto result = project_root_summary(repo_root, releases);
    if (!result.success) {
        return result;
    }
    std::error_code error;
    const auto root = fs::canonical(fs::path(repo_root), error);
    if (error) {
        finding(result, EvidenceCategory::Violation, "root_summary.unreadable",
            "path=. code=root_unreadable");
        return result;
    }
    const auto readme = read(root, "README.md", result);
    if (!result.success) {
        return result;
    }
    const auto begin_token = std::string(begin_marker) + "\n";
    const auto end_token = std::string(end_marker) + "\n";
    const auto begin = readme.find(begin_token);
    const auto end = readme.find(end_token);
    if (begin == std::string::npos || end == std::string::npos || end < begin ||
        (begin != 0U && readme[begin - 1U] != '\n') ||
        readme.find(begin_token, begin + begin_token.size()) != std::string::npos ||
        readme.find(end_token, end + end_token.size()) != std::string::npos) {
        finding(result, EvidenceCategory::Violation, "root_summary.readme_region",
            "reason=missing_duplicate_misordered_or_non_line_marker expected_command=root-summary_--repo_PATH");
        return result;
    }
    const auto observed = readme.substr(begin, end + end_token.size() - begin);
    if (observed != result.projection_markdown) {
        finding(result, EvidenceCategory::Violation, "root_summary.stale",
            "path=README.md expected_projection=root-summary_--repo_PATH");
        return result;
    }
    finding(result, EvidenceCategory::Pass, "root_summary.current",
        "path=README.md");
    return result;
}
