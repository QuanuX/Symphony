#include "feature_administration.hpp"

#include "evidence.hpp"

#include <symphony/knowledge/engine/digest.hpp>
#include <symphony/knowledge/engine/error.hpp>
#include <symphony/knowledge/engine/json.hpp>
#include <symphony/knowledge/engine/path.hpp>
#include <symphony/knowledge/engine/protocol.hpp>

#include <algorithm>
#include <array>
#include <cctype>
#include <filesystem>
#include <initializer_list>
#include <map>
#include <optional>
#include <set>
#include <sstream>
#include <string>
#include <string_view>
#include <vector>

namespace {

namespace fs = std::filesystem;
namespace engine = symphony::knowledge::engine;

constexpr std::string_view profile_path = "knowledge/FEATURE-ADMINISTRATION-PROFILE.json";
constexpr std::string_view feature_registry_path = "knowledge/ssfv/REGISTRY.md";
constexpr std::string_view command_registry_path = "tools/qxctl/COMMANDS.json";
constexpr std::size_t maximum_file_bytes = 4U * 1024U * 1024U;
constexpr std::size_t maximum_features = 8192U;
constexpr std::size_t maximum_commands = 1024U;
constexpr std::size_t maximum_expectations = 10U;
constexpr std::size_t maximum_references = 256U;
constexpr std::size_t maximum_modules = 1024U;

using Json = engine::Json;

void finding(
    FeatureAdministrationCheckResult& result,
    const EvidenceCategory category,
    const std::string& rule,
    const std::string& detail) {
    result.messages.push_back(format_evidence(category, rule, detail));
    if (category == EvidenceCategory::Violation) {
        result.success = false;
        ++result.violations;
    }
}

bool exact_fields(const Json& object, std::initializer_list<std::string_view> fields) {
    if (!object.is_object() || object.size() != fields.size()) {
        return false;
    }
    return std::all_of(fields.begin(), fields.end(), [&object](const std::string_view field) {
        return object.contains(std::string(field));
    });
}

bool tagged_digest(const Json& value) {
    if (!value.is_string()) {
        return false;
    }
    const auto text = value.get<std::string>();
    return text.size() == 71U && text.starts_with("sha256:") &&
        std::all_of(text.begin() + 7, text.end(), [](const unsigned char character) {
            return (character >= '0' && character <= '9') || (character >= 'a' && character <= 'f');
        });
}

bool string_array(const Json& value, const std::size_t maximum) {
    if (!value.is_array() || value.size() > maximum) {
        return false;
    }
    return std::all_of(value.begin(), value.end(), [](const Json& item) {
        return item.is_string() && !item.get_ref<const std::string&>().empty() &&
            item.get_ref<const std::string&>().size() <= 4096U;
    });
}

bool member_of(const std::string& value, std::initializer_list<std::string_view> choices) {
    return std::any_of(choices.begin(), choices.end(), [&value](const std::string_view choice) {
        return value == choice;
    });
}

std::optional<std::string> bounded_read(
    const fs::path& root,
    const std::string& path,
    FeatureAdministrationCheckResult& result) {
    try {
        return engine::read_regular_file_no_follow(root, path, maximum_file_bytes);
    } catch (const engine::Error& error) {
        finding(result, EvidenceCategory::Violation, "feature_administration.unreadable",
            "path=" + path + " code=" + error.code());
    } catch (const std::exception&) {
        finding(result, EvidenceCategory::Violation, "feature_administration.unreadable",
            "path=" + path + " code=unexpected_error");
    }
    return std::nullopt;
}

std::optional<Json> bounded_json(
    const std::string& contents,
    const std::string& path,
    FeatureAdministrationCheckResult& result) {
    try {
        return engine::parse_bounded_json(contents, maximum_file_bytes);
    } catch (const engine::Error& error) {
        finding(result, EvidenceCategory::Violation, "feature_administration.invalid_json",
            "path=" + path + " code=" + error.code());
    } catch (const std::exception&) {
        finding(result, EvidenceCategory::Violation, "feature_administration.invalid_json",
            "path=" + path + " code=unexpected_error");
    }
    return std::nullopt;
}

std::vector<std::string> registry_feature_ids(
    const std::string& contents,
    FeatureAdministrationCheckResult& result) {
    constexpr std::string_view prefix = "- feature_id: `";
    std::vector<std::string> ids;
    std::set<std::string> unique;
    std::istringstream stream(contents);
    std::string line;
    std::size_t line_number = 0U;
    while (std::getline(stream, line)) {
        ++line_number;
        if (line.size() > 65536U) {
            finding(result, EvidenceCategory::Violation, "feature_administration.registry_line_limit",
                "line=" + std::to_string(line_number));
            continue;
        }
        if (!line.starts_with(prefix)) {
            continue;
        }
        if (line.size() <= prefix.size() || line.back() != '`' || ids.size() >= maximum_features) {
            finding(result, EvidenceCategory::Violation, "feature_administration.registry_shape",
                "line=" + std::to_string(line_number));
            continue;
        }
        const auto id = line.substr(prefix.size(), line.size() - prefix.size() - 1U);
        if (!id.starts_with("ssfv:") || id.size() > 256U || !unique.insert(id).second) {
            finding(result, EvidenceCategory::Violation, "feature_administration.registry_feature_id",
                "line=" + std::to_string(line_number));
            continue;
        }
        ids.push_back(id);
    }
    if (ids.empty()) {
        finding(result, EvidenceCategory::Violation, "feature_administration.registry_count",
            "reason=empty");
    } else {
        finding(result, EvidenceCategory::Pass, "feature_administration.registry_count",
            "features=" + std::to_string(ids.size()));
    }
    return ids;
}

bool self_digest(
    const Json& document,
    const char* field,
    const std::string& rule,
    FeatureAdministrationCheckResult& result) {
    if (!document.contains(field) || !tagged_digest(document.at(field))) {
        finding(result, EvidenceCategory::Violation, rule, "reason=missing_or_invalid");
        return false;
    }
    const auto expected = document.at(field).get<std::string>();
    auto preimage = document;
    preimage.erase(field);
    const auto observed = engine::tagged_sha256(preimage.dump());
    if (expected != observed) {
        finding(result, EvidenceCategory::Violation, rule,
            "expected=" + expected + " observed=" + observed);
        return false;
    }
    finding(result, EvidenceCategory::Pass, rule, "digest=" + expected);
    return true;
}

bool inheritance_cycle(
    const std::string& feature,
    const std::map<std::string, std::set<std::string>>& edges,
    std::set<std::string>& visiting,
    std::set<std::string>& visited) {
    if (visited.contains(feature)) {
        return false;
    }
    if (!visiting.insert(feature).second) {
        return true;
    }
    const auto found = edges.find(feature);
    if (found != edges.end()) {
        for (const auto& parent : found->second) {
            if (inheritance_cycle(parent, edges, visiting, visited)) {
                return true;
            }
        }
    }
    visiting.erase(feature);
    visited.insert(feature);
    return false;
}

struct ProfileEvidence {
    std::vector<std::string> feature_ids;
    std::set<std::string> command_references;
    std::size_t unreviewed = 0U;
};

struct ModuleCensus {
    std::vector<std::string> implemented_scopes;
    std::size_t documentation_only = 0U;
};

bool safe_module_name(const std::string& name) {
    return !name.empty() && name.size() <= 256U &&
        std::isalnum(static_cast<unsigned char>(name.front())) &&
        std::all_of(name.begin(), name.end(), [](const unsigned char character) {
            return std::isalnum(character) || character == '.' || character == '_' || character == '-';
        });
}

bool implementation_marker(
    const fs::path& path,
    const bool directory,
    const std::string& relative,
    FeatureAdministrationCheckResult& result) {
    std::error_code error;
    const auto status = fs::symlink_status(path, error);
    if (error || status.type() == fs::file_type::not_found) {
        return false;
    }
    if (status.type() == fs::file_type::symlink ||
        (directory && status.type() != fs::file_type::directory) ||
        (!directory && status.type() != fs::file_type::regular)) {
        finding(result, EvidenceCategory::Violation, "feature_administration.module_admission",
            "scope=" + relative.substr(0U, relative.find_last_of('/')) +
            " marker=" + relative + " reason=unsafe_implementation_marker");
        return true;
    }
    return true;
}

ModuleCensus module_census(
    const fs::path& root,
    FeatureAdministrationCheckResult& result) {
    ModuleCensus census;
    std::error_code error;
    const auto modules_status = fs::symlink_status(root / "modules", error);
    if (error || modules_status.type() == fs::file_type::not_found) {
        return census;
    }
    if (modules_status.type() != fs::file_type::directory) {
        finding(result, EvidenceCategory::Violation, "feature_administration.module_admission",
            "scope=modules reason=unsafe_module_root");
        return census;
    }
    std::vector<fs::path> entries;
    fs::directory_iterator iterator(root / "modules", error);
    const fs::directory_iterator end;
    while (!error && iterator != end) {
        if (entries.size() >= maximum_modules) {
            finding(result, EvidenceCategory::Violation, "feature_administration.module_admission",
                "scope=modules reason=module_count_limit");
            return census;
        }
        entries.push_back(iterator->path());
        iterator.increment(error);
    }
    if (error) {
        finding(result, EvidenceCategory::Violation, "feature_administration.module_admission",
            "scope=modules reason=discovery_failed");
        return census;
    }
    std::sort(entries.begin(), entries.end());
    for (const auto& path : entries) {
        const auto name = path.filename().string();
        const auto scope = "modules/" + name;
        const auto status = fs::symlink_status(path, error);
        if (error || status.type() == fs::file_type::symlink) {
            finding(result, EvidenceCategory::Violation, "feature_administration.module_admission",
                "scope=" + scope + " reason=unsafe_module_entry");
            error.clear();
            continue;
        }
        if (status.type() != fs::file_type::directory) {
            continue;
        }
        if (!safe_module_name(name)) {
            finding(result, EvidenceCategory::Violation, "feature_administration.module_admission",
                "scope=" + scope + " reason=unsafe_module_name");
            continue;
        }
        bool implemented = false;
        for (const auto* marker : {"CMakeLists.txt", "go.mod", "Package.swift"}) {
            implemented = implementation_marker(path / marker, false, scope + "/" + marker, result) ||
                implemented;
        }
        for (const auto* marker : {"src", "cmd", "Sources"}) {
            implemented = implementation_marker(path / marker, true, scope + "/" + marker, result) ||
                implemented;
        }
        if (implemented) {
            census.implemented_scopes.push_back(scope);
        } else {
            ++census.documentation_only;
        }
    }
    return census;
}

std::optional<std::string> backtick_value(
    const std::string& line,
    const std::string_view prefix) {
    if (!line.starts_with(prefix) || line.size() <= prefix.size() || line.back() != '`') {
        return std::nullopt;
    }
    return line.substr(prefix.size(), line.size() - prefix.size() - 1U);
}

struct RegistryRoute {
    std::string feature_id;
    std::string feature_file;
};

std::map<std::string, std::vector<RegistryRoute>> registry_module_routes(
    const std::string& contents) {
    constexpr std::string_view feature_prefix = "- feature_id: `";
    constexpr std::string_view file_prefix = "- feature_file: `";
    constexpr std::string_view scope_prefix = "- source_scope: `";
    std::map<std::string, std::vector<RegistryRoute>> routes;
    std::istringstream input(contents);
    std::string line;
    std::string feature_id;
    std::string feature_file;
    while (std::getline(input, line)) {
        if (const auto value = backtick_value(line, feature_prefix)) {
            feature_id = *value;
            feature_file.clear();
        } else if (const auto value = backtick_value(line, file_prefix)) {
            feature_file = *value;
        } else if (const auto value = backtick_value(line, scope_prefix)) {
            if (!feature_id.empty() && !feature_file.empty() && value->starts_with("modules/")) {
                routes[*value].push_back(RegistryRoute{feature_id, feature_file});
            }
        }
    }
    return routes;
}

void check_module_admission(
    const fs::path& root,
    const ModuleCensus& census,
    const std::string& registry_contents,
    const ProfileEvidence& profile,
    FeatureAdministrationCheckResult& result) {
    const auto routes = registry_module_routes(registry_contents);
    const std::set<std::string> profile_ids(profile.feature_ids.begin(), profile.feature_ids.end());
    for (const auto& scope : census.implemented_scopes) {
        const auto feature_file = scope + "/FEATURES.md";
        if (!bounded_read(root, feature_file, result)) {
            finding(result, EvidenceCategory::Violation, "feature_administration.module_admission",
                "scope=" + scope + " reason=features_missing");
        }
        const auto found = routes.find(scope);
        bool exact_route = false;
        bool profile_entry = false;
        if (found != routes.end()) {
            for (const auto& route : found->second) {
                if (route.feature_file == feature_file) {
                    exact_route = true;
                    profile_entry = profile_entry || profile_ids.contains(route.feature_id);
                }
            }
        }
        if (!exact_route) {
            finding(result, EvidenceCategory::Violation, "feature_administration.module_admission",
                "scope=" + scope + " reason=registry_route_missing");
        }
        if (!profile_entry) {
            finding(result, EvidenceCategory::Violation, "feature_administration.module_admission",
                "scope=" + scope + " reason=profile_mapping_missing");
        }
    }
    finding(result, EvidenceCategory::Pass, "feature_administration.module_admission",
        "implemented=" + std::to_string(census.implemented_scopes.size()) +
        " documentation_only=" + std::to_string(census.documentation_only));
}

ProfileEvidence check_profile(
    const Json& profile,
    const std::vector<std::string>& registry_ids,
    const std::string& registry_contents,
    FeatureAdministrationCheckResult& result) {
    ProfileEvidence evidence;
    if (!exact_fields(profile, {"catalog_complete", "catalog_scope", "features", "format_version",
            "forward_gate", "profile_digest", "profile_id", "protocol", "registered_feature_count",
            "ssfv_registry_digest"})) {
        finding(result, EvidenceCategory::Violation, "feature_administration.profile_shape",
            "reason=top_level_field_set");
        return evidence;
    }
    if (!profile.at("protocol").is_string() ||
        profile.at("protocol").get<std::string>() != "symphony.knowledge.feature-administration-profile.v1" ||
        !profile.at("format_version").is_number_integer() || profile.at("format_version").get<int>() != 1 ||
        !profile.at("profile_id").is_string() || !profile.at("catalog_scope").is_string() ||
        profile.at("catalog_scope").get<std::string>() != "registered_partial_catalog" ||
        !profile.at("catalog_complete").is_boolean() || profile.at("catalog_complete").get<bool>() ||
        !profile.at("registered_feature_count").is_number_unsigned() ||
        !profile.at("forward_gate").is_string() || !profile.at("features").is_array() ||
        profile.at("features").size() > maximum_features || !tagged_digest(profile.at("ssfv_registry_digest"))) {
        finding(result, EvidenceCategory::Violation, "feature_administration.profile_shape",
            "reason=field_type_or_value");
        return evidence;
    }
    self_digest(profile, "profile_digest", "feature_administration.profile_digest", result);
    const auto expected_registry_digest = engine::tagged_sha256(registry_contents);
    const auto profile_registry_digest = profile.at("ssfv_registry_digest").get<std::string>();
    if (profile_registry_digest != expected_registry_digest) {
        finding(result, EvidenceCategory::Violation, "feature_administration.profile_registry_digest",
            "expected=" + expected_registry_digest + " observed=" + profile_registry_digest);
    } else {
        finding(result, EvidenceCategory::Pass, "feature_administration.profile_registry_digest",
            "digest=" + expected_registry_digest);
    }

    const auto gate = profile.at("forward_gate").get<std::string>();
    if (!member_of(gate, {"report_only", "enforce_new_records", "enforce_all_records"})) {
        finding(result, EvidenceCategory::Violation, "feature_administration.forward_gate",
            "reason=unsupported_gate");
    }
    std::set<std::string> unique_features;
    std::map<std::string, std::set<std::string>> inheritance;
    std::string prior_feature;
    for (const auto& feature : profile.at("features")) {
        if (!exact_fields(feature, {"expectations", "feature_id"}) ||
            !feature.at("feature_id").is_string() || !feature.at("expectations").is_array() ||
            feature.at("expectations").size() > maximum_expectations) {
            finding(result, EvidenceCategory::Violation, "feature_administration.profile_feature_shape",
                "reason=field_set_or_type");
            continue;
        }
        const auto feature_id = feature.at("feature_id").get<std::string>();
        if (feature_id.size() > 256U || !feature_id.starts_with("ssfv:") ||
            !unique_features.insert(feature_id).second) {
            finding(result, EvidenceCategory::Violation, "feature_administration.profile_feature_id",
                "feature_id=" + feature_id);
            continue;
        }
        if (!prior_feature.empty() && prior_feature >= feature_id) {
            finding(result, EvidenceCategory::Violation, "feature_administration.profile_feature_order",
                "feature_id=" + feature_id);
        }
        prior_feature = feature_id;
        evidence.feature_ids.push_back(feature_id);
        if (feature.at("expectations").empty()) {
            ++evidence.unreviewed;
            finding(result, gate == "report_only" ? EvidenceCategory::Warning : EvidenceCategory::Violation,
                "feature_administration.profile_unreviewed",
                "feature_id=" + feature_id + " reason=empty_expectations gate=" + gate);
            continue;
        }
        std::set<std::string> interactions;
        for (const auto& expectation : feature.at("expectations")) {
            if (!exact_fields(expectation, {"command_ids", "delivery", "engine_operation_ids", "evidence",
                    "inherited_from_feature_id", "interaction", "rationale", "requirement"}) ||
                !expectation.at("interaction").is_string() || !expectation.at("requirement").is_string() ||
                !expectation.at("delivery").is_string() || !expectation.at("rationale").is_string() ||
                expectation.at("rationale").get_ref<const std::string&>().empty() ||
                !string_array(expectation.at("command_ids"), maximum_references) ||
                !string_array(expectation.at("engine_operation_ids"), maximum_references) ||
                !string_array(expectation.at("evidence"), maximum_references) ||
                !(expectation.at("inherited_from_feature_id").is_null() ||
                  expectation.at("inherited_from_feature_id").is_string())) {
                finding(result, EvidenceCategory::Violation, "feature_administration.profile_expectation_shape",
                    "feature_id=" + feature_id);
                continue;
            }
            const auto interaction = expectation.at("interaction").get<std::string>();
            const auto requirement = expectation.at("requirement").get<std::string>();
            const auto delivery = expectation.at("delivery").get<std::string>();
            if (!member_of(interaction, {"discover", "inspect", "query", "validate", "configure",
                    "propose", "invoke", "apply", "lifecycle", "recover"}) ||
                !member_of(requirement, {"required", "optional", "prohibited", "not_applicable"}) ||
                !member_of(delivery, {"direct", "composed", "delegated", "lifecycle_only",
                    "observation_only", "runtime_only", "system_orchestrated", "none", "unreviewed"}) ||
                !interactions.insert(interaction).second) {
                finding(result, EvidenceCategory::Violation, "feature_administration.profile_expectation_value",
                    "feature_id=" + feature_id + " interaction=" + interaction);
            }
            if (delivery == "unreviewed") {
                ++evidence.unreviewed;
                finding(result, gate == "report_only" ? EvidenceCategory::Warning : EvidenceCategory::Violation,
                    "feature_administration.profile_unreviewed",
                    "feature_id=" + feature_id + " reason=unreviewed_delivery gate=" + gate);
            }
            std::set<std::string> command_ids;
            for (const auto& value : expectation.at("command_ids")) {
                const auto command_id = value.get<std::string>();
                if (!command_ids.insert(command_id).second) {
                    finding(result, EvidenceCategory::Violation,
                        "feature_administration.profile_command_reference",
                        "feature_id=" + feature_id + " command_id=" + command_id + " reason=duplicate");
                }
                evidence.command_references.insert(command_id);
            }
            if (expectation.at("inherited_from_feature_id").is_string()) {
                inheritance[feature_id].insert(
                    expectation.at("inherited_from_feature_id").get<std::string>());
            }
        }
    }
    const std::set<std::string> registry_features(registry_ids.begin(), registry_ids.end());
    const std::set<std::string> profile_features(
        evidence.feature_ids.begin(), evidence.feature_ids.end());
    if (profile.at("registered_feature_count").get<std::size_t>() != registry_ids.size() ||
        evidence.feature_ids.size() != registry_ids.size() || profile_features != registry_features) {
        finding(result, EvidenceCategory::Violation, "feature_administration.profile_feature_set",
            "declared=" +
            std::to_string(profile.at("registered_feature_count").get<std::size_t>()) +
            " " +
            "registry=" + std::to_string(registry_ids.size()) +
            " profile=" + std::to_string(evidence.feature_ids.size()));
    } else {
        finding(result, EvidenceCategory::Pass, "feature_administration.profile_feature_set",
            "features=" + std::to_string(registry_ids.size()));
    }
    for (const auto& [feature, parents] : inheritance) {
        for (const auto& parent : parents) {
            if (!unique_features.contains(parent)) {
                finding(result, EvidenceCategory::Violation, "feature_administration.profile_inheritance",
                    "feature_id=" + feature + " parent_feature_id=" + parent + " reason=unresolved");
            }
        }
    }
    std::set<std::string> visiting;
    std::set<std::string> visited;
    for (const auto& feature : unique_features) {
        if (inheritance_cycle(feature, inheritance, visiting, visited)) {
            finding(result, EvidenceCategory::Violation, "feature_administration.profile_inheritance",
                "feature_id=" + feature + " reason=cycle");
            break;
        }
    }
    return evidence;
}

std::set<std::string> check_commands(
    const Json& registry,
    const std::set<std::string>& registered_features,
    FeatureAdministrationCheckResult& result) {
    std::set<std::string> command_ids;
    if (!exact_fields(registry, {"client_id", "client_trust", "client_version", "commands",
            "executable_digest", "format_version", "protocol", "receipt_digest", "registry_digest",
            "registry_kind"})) {
        finding(result, EvidenceCategory::Violation, "feature_administration.commands_shape",
            "reason=top_level_field_set");
        return command_ids;
    }
    if (!registry.at("protocol").is_string() ||
        registry.at("protocol").get<std::string>() != "symphony.qxctl.command-registry.v1" ||
        !registry.at("format_version").is_number_integer() || registry.at("format_version").get<int>() != 1 ||
        !registry.at("registry_kind").is_string() ||
        registry.at("registry_kind").get<std::string>() != "expected" ||
        !registry.at("client_id").is_string() || registry.at("client_id").get<std::string>() != "qxctl" ||
        !registry.at("client_version").is_null() || !registry.at("executable_digest").is_null() ||
        !registry.at("receipt_digest").is_null() || !registry.at("client_trust").is_string() ||
        registry.at("client_trust").get<std::string>() != "unreceipted" ||
        !registry.at("commands").is_array() || registry.at("commands").size() > maximum_commands) {
        finding(result, EvidenceCategory::Violation, "feature_administration.commands_shape",
            "reason=field_type_or_value");
        return command_ids;
    }
    self_digest(registry, "registry_digest", "feature_administration.commands_digest", result);
    std::string prior_id;
    for (const auto& command : registry.at("commands")) {
        if (!exact_fields(command, {"aliases", "authority_mode", "backend_operation_ids", "command_id",
                "deprecated_in", "feature_bindings", "grammar", "infrastructure_purpose", "input_protocols",
                "introduced_in", "json_output", "mutability", "noninteractive", "output_protocols",
                "recovery_command_id", "replacement_ids", "result_validation_protocols", "status",
                "target_scope", "visibility"}) || !command.at("command_id").is_string() ||
            !command.at("feature_bindings").is_array() || command.at("feature_bindings").size() > maximum_references) {
            finding(result, EvidenceCategory::Violation, "feature_administration.command_shape",
                "reason=field_set_or_type");
            continue;
        }
        const auto command_id = command.at("command_id").get<std::string>();
        if (!command_id.starts_with("qxcmd:") || command_id.size() > 256U ||
            !command_ids.insert(command_id).second) {
            finding(result, EvidenceCategory::Violation, "feature_administration.command_id",
                "command_id=" + command_id + " reason=invalid_or_duplicate");
            continue;
        }
        if (!prior_id.empty() && prior_id >= command_id) {
            finding(result, EvidenceCategory::Violation, "feature_administration.command_order",
                "command_id=" + command_id);
        }
        prior_id = command_id;
        std::set<std::string> bindings;
        for (const auto& binding : command.at("feature_bindings")) {
            if (!exact_fields(binding, {"feature_id", "interaction"}) ||
                !binding.at("feature_id").is_string() || !binding.at("interaction").is_string()) {
                finding(result, EvidenceCategory::Violation, "feature_administration.command_feature_binding",
                    "command_id=" + command_id + " reason=shape");
                continue;
            }
            const auto feature_id = binding.at("feature_id").get<std::string>();
            const auto interaction = binding.at("interaction").get<std::string>();
            if (!registered_features.contains(feature_id) ||
                !member_of(interaction, {"discover", "inspect", "query", "validate", "configure",
                    "propose", "invoke", "apply", "lifecycle", "recover"}) ||
                !bindings.insert(feature_id + "\n" + interaction).second) {
                finding(result, EvidenceCategory::Violation, "feature_administration.command_feature_binding",
                    "command_id=" + command_id + " feature_id=" + feature_id);
            }
        }
        if (command.at("feature_bindings").empty()) {
            if (!command.at("infrastructure_purpose").is_string() ||
                command.at("infrastructure_purpose").get_ref<const std::string&>().empty()) {
                finding(result, EvidenceCategory::Violation, "feature_administration.command_feature_binding",
                    "command_id=" + command_id + " reason=unowned");
            }
        } else if (!command.at("infrastructure_purpose").is_null()) {
            finding(result, EvidenceCategory::Violation, "feature_administration.command_feature_binding",
                "command_id=" + command_id + " reason=ambiguous_ownership");
        }
    }
    result.commands_checked = command_ids.size();
    finding(result, EvidenceCategory::Pass, "feature_administration.command_inventory",
        "commands=" + std::to_string(command_ids.size()));
    return command_ids;
}

}

FeatureAdministrationCheckResult check_feature_administration(const std::string& repo_root) {
    FeatureAdministrationCheckResult result{true, {}, 0U, 0U, 0U, 0U};
    std::error_code error;
    const auto root = fs::canonical(fs::path(repo_root), error);
    if (error) {
        finding(result, EvidenceCategory::Violation, "feature_administration.unreadable",
            "path=. code=root_unreadable");
        return result;
    }
    const auto census = module_census(root, result);
    const std::array<std::string, 3> paths = {
        std::string(profile_path), std::string(feature_registry_path), std::string(command_registry_path),
    };
    const auto present = std::count_if(paths.begin(), paths.end(), [&root](const std::string& path) {
        std::error_code status_error;
        const auto status = fs::symlink_status(root / path, status_error);
        return !status_error && status.type() != fs::file_type::not_found;
    });
    if (present == 0) {
        for (const auto& scope : census.implemented_scopes) {
            finding(result, EvidenceCategory::Violation, "feature_administration.module_admission",
                "scope=" + scope + " reason=administration_surfaces_absent");
        }
        if (!census.implemented_scopes.empty()) {
            return result;
        }
        finding(result, EvidenceCategory::Absent, "feature_administration.absent",
            "profile=false commands=false registry=false");
        return result;
    }
    if (present != paths.size()) {
        for (const auto& path : paths) {
            std::error_code status_error;
            if (fs::symlink_status(root / path, status_error).type() == fs::file_type::not_found) {
                finding(result, EvidenceCategory::Violation, "feature_administration.unreadable",
                    "path=" + path + " code=missing");
            }
        }
        return result;
    }

    const auto registry_contents = bounded_read(root, std::string(feature_registry_path), result);
    const auto profile_contents = bounded_read(root, std::string(profile_path), result);
    const auto commands_contents = bounded_read(root, std::string(command_registry_path), result);
    if (!registry_contents || !profile_contents || !commands_contents) {
        return result;
    }
    const auto profile = bounded_json(*profile_contents, std::string(profile_path), result);
    const auto commands = bounded_json(*commands_contents, std::string(command_registry_path), result);
    if (!profile || !commands) {
        return result;
    }
    const auto registry_ids = registry_feature_ids(*registry_contents, result);
    const auto profile_evidence = check_profile(*profile, registry_ids, *registry_contents, result);
    check_module_admission(root, census, *registry_contents, profile_evidence, result);
    result.features_checked = profile_evidence.feature_ids.size();
    result.unreviewed_features = profile_evidence.unreviewed;
    const std::set<std::string> registered_features(registry_ids.begin(), registry_ids.end());
    const auto command_ids = check_commands(*commands, registered_features, result);
    for (const auto& command_id : profile_evidence.command_references) {
        if (!command_ids.contains(command_id)) {
            finding(result, EvidenceCategory::Violation, "feature_administration.profile_command_reference",
                "command_id=" + command_id + " reason=unresolved");
        }
    }
    const auto completion_violations = result.violations + (result.success ? 0U : 1U);
    finding(result, result.success ? EvidenceCategory::Pass : EvidenceCategory::Violation,
        "feature_administration.scan_complete",
        "features=" + std::to_string(result.features_checked) +
        " commands=" + std::to_string(result.commands_checked) +
        " unreviewed=" + std::to_string(result.unreviewed_features) +
        " violations=" + std::to_string(completion_violations));
    return result;
}
