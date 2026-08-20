#include "invariant_ownership.hpp"

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
#include <string>
#include <string_view>
#include <vector>

namespace {

namespace fs = std::filesystem;
namespace engine = symphony::knowledge::engine;
using Json = engine::Json;

constexpr std::string_view registry_path = "knowledge/INVARIANT-OWNERSHIP.json";
constexpr std::size_t maximum_registry_bytes = 4U * 1024U * 1024U;
constexpr std::size_t maximum_evidence_bytes = 8U * 1024U * 1024U;
constexpr std::size_t maximum_adapters = 64U;
constexpr std::size_t maximum_invariants = 4096U;
constexpr std::size_t maximum_references = 128U;
constexpr std::size_t maximum_operations = 64U;
constexpr std::size_t maximum_cases = 128U;

void finding(
    InvariantOwnershipCheckResult& result,
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
    const auto& text = value.get_ref<const std::string&>();
    return text.size() == 71U && text.starts_with("sha256:") &&
        std::all_of(text.begin() + 7, text.end(), [](const unsigned char character) {
            return (character >= '0' && character <= '9') ||
                (character >= 'a' && character <= 'f');
        });
}

bool identifier_character(const unsigned char character) {
    return std::isalnum(character) || character == '.' || character == '_' ||
        character == ':' || character == '-';
}

bool token(const Json& value, const std::size_t maximum = 256U) {
    if (!value.is_string()) {
        return false;
    }
    const auto& text = value.get_ref<const std::string&>();
    return !text.empty() && text.size() <= maximum &&
        std::isalnum(static_cast<unsigned char>(text.front())) &&
        std::all_of(text.begin(), text.end(), [](const unsigned char character) {
            return identifier_character(character);
        });
}

bool path_token(const Json& value) {
    if (!value.is_string()) {
        return false;
    }
    const auto& path = value.get_ref<const std::string&>();
    return engine::is_safe_relative_path(path) &&
        std::all_of(path.begin(), path.end(), [](const unsigned char character) {
            return std::isalnum(character) || character == '.' || character == '_' ||
                character == '/' || character == '-';
        });
}

bool invariant_or_adapter_identifier(const Json& value, const std::string_view prefix) {
    if (!value.is_string()) {
        return false;
    }
    const auto& text = value.get_ref<const std::string&>();
    if (text.size() > 256U || !text.starts_with(prefix) || text.size() == prefix.size()) {
        return false;
    }
    const auto suffix = std::string_view(text).substr(prefix.size());
    if (suffix.front() < 'a' || suffix.front() > 'z') {
        return false;
    }
    bool separator = false;
    for (std::size_t index = 1U; index < suffix.size(); ++index) {
        const auto character = static_cast<unsigned char>(suffix[index]);
        if (character == '.' || character == '-') {
            if (separator) {
                return false;
            }
            separator = true;
        } else if ((character >= 'a' && character <= 'z') || std::isdigit(character)) {
            separator = false;
        } else {
            return false;
        }
    }
    return !separator;
}

bool operation_identifier(const Json& value) {
    if (!value.is_string()) {
        return false;
    }
    const auto& text = value.get_ref<const std::string&>();
    constexpr std::string_view prefix = "engop:";
    if (text.size() > 256U || !text.starts_with(prefix)) {
        return false;
    }
    const auto namespace_end = text.find(':', prefix.size());
    if (namespace_end == std::string::npos || namespace_end == prefix.size() ||
        namespace_end - prefix.size() > 63U || namespace_end + 1U >= text.size()) {
        return false;
    }
    const auto namespace_value = std::string_view(text).substr(
        prefix.size(), namespace_end - prefix.size());
    if (namespace_value.front() < 'a' || namespace_value.front() > 'z' ||
        !std::all_of(namespace_value.begin() + 1U, namespace_value.end(),
            [](const unsigned char character) {
                return (character >= 'a' && character <= 'z') ||
                    std::isdigit(character) || character == '-';
            })) {
        return false;
    }
    const auto name = std::string_view(text).substr(namespace_end + 1U);
    Json name_value = std::string(name);
    return invariant_or_adapter_identifier(name_value, "");
}

std::optional<std::string> bounded_read(
    const fs::path& root,
    const std::string& path,
    const std::size_t maximum,
    InvariantOwnershipCheckResult& result,
    const std::string& rule = "invariant_ownership.unreadable") {
    try {
        return engine::read_regular_file_no_follow(root, path, maximum);
    } catch (const engine::Error& error) {
        finding(result, EvidenceCategory::Violation, rule,
            "path=" + path + " code=" + error.code());
    } catch (const std::exception&) {
        finding(result, EvidenceCategory::Violation, rule,
            "path=" + path + " code=unexpected_error");
    }
    return std::nullopt;
}

template <typename Predicate>
bool sorted_unique_string_array(
    const Json& value,
    const std::size_t minimum,
    const std::size_t maximum,
    Predicate predicate,
    InvariantOwnershipCheckResult& result,
    const std::string& rule,
    const std::string& owner) {
    if (!value.is_array() || value.size() < minimum || value.size() > maximum) {
        finding(result, EvidenceCategory::Violation, rule,
            "owner=" + owner + " reason=count");
        return false;
    }
    std::string prior;
    bool valid = true;
    for (const auto& item : value) {
        if (!predicate(item)) {
            finding(result, EvidenceCategory::Violation, rule,
                "owner=" + owner + " reason=value");
            valid = false;
            continue;
        }
        const auto text = item.template get<std::string>();
        if (!prior.empty() && prior >= text) {
            finding(result, EvidenceCategory::Violation, rule,
                "owner=" + owner + " value=" + text + " reason=order_or_duplicate");
            valid = false;
        }
        prior = text;
    }
    return valid;
}

bool visible_text(const Json& value, const std::size_t maximum) {
    if (!value.is_string()) {
        return false;
    }
    const auto& text = value.get_ref<const std::string&>();
    return !text.empty() && text.size() <= maximum &&
        std::none_of(text.begin(), text.end(), [](const unsigned char character) {
            return character == '\0' || character == '\r' ||
                (character < 0x20U && character != '\n' && character != '\t');
        });
}

bool has_definition(const std::string& contents, const std::string& path, const std::string& name) {
    std::size_t begin = 0U;
    while (begin <= contents.size()) {
        const auto end = contents.find('\n', begin);
        auto line = std::string_view(contents).substr(
            begin, end == std::string::npos ? std::string::npos : end - begin);
        while (!line.empty() && std::isspace(static_cast<unsigned char>(line.front()))) {
            line.remove_prefix(1U);
        }
        if (!line.starts_with("//")) {
            if (path.ends_with(".go")) {
                if (line.starts_with("func")) {
                    line.remove_prefix(4U);
                    while (!line.empty() && std::isspace(static_cast<unsigned char>(line.front()))) {
                        line.remove_prefix(1U);
                    }
                    if (line.starts_with(name) && line.size() > name.size()) {
                        const auto next = static_cast<unsigned char>(line[name.size()]);
                        if (next == '(' || std::isspace(next)) {
                            return true;
                        }
                    }
                }
            } else if (path.ends_with(".cpp") || path.ends_with(".cc") || path.ends_with(".cxx")) {
                const auto position = line.find(name);
                if (position != std::string_view::npos && position > 0U) {
                    const auto before = static_cast<unsigned char>(line[position - 1U]);
                    auto after = position + name.size();
                    while (after < line.size() && std::isspace(static_cast<unsigned char>(line[after]))) {
                        ++after;
                    }
                    const auto declaration_prefix = line.substr(0U, position);
                    if (!identifier_character(before) && after < line.size() && line[after] == '(' &&
                        (declaration_prefix.find("void") != std::string_view::npos ||
                         declaration_prefix.find("bool") != std::string_view::npos ||
                         declaration_prefix.find("int") != std::string_view::npos)) {
                        return true;
                    }
                }
            } else if (path.ends_with(".swift")) {
                const auto marker = "func " + name;
                const auto position = line.find(marker);
                if (position != std::string_view::npos) {
                    auto after = position + marker.size();
                    while (after < line.size() && std::isspace(static_cast<unsigned char>(line[after]))) {
                        ++after;
                    }
                    if (after < line.size() && line[after] == '(' &&
                        (line.starts_with("@Test") || line.starts_with("func") ||
                         line.starts_with("public func") || line.starts_with("private func"))) {
                        return true;
                    }
                }
            } else if (path.ends_with(".sh")) {
                if (line.starts_with(name)) {
                    auto after = name.size();
                    while (after < line.size() && std::isspace(static_cast<unsigned char>(line[after]))) {
                        ++after;
                    }
                    if (after + 1U < line.size() && line[after] == '(' && line[after + 1U] == ')') {
                        after += 2U;
                        while (after < line.size() && std::isspace(static_cast<unsigned char>(line[after]))) {
                            ++after;
                        }
                        if (after < line.size() && line[after] == '{') {
                            return true;
                        }
                    }
                }
            }
        }
        if (end == std::string::npos) {
            break;
        }
        begin = end + 1U;
    }
    return false;
}

struct EvidenceCache {
    std::map<std::string, std::optional<std::string>> files;
};

bool check_regular_path(
    const fs::path& root,
    const std::string& path,
    InvariantOwnershipCheckResult& result,
    EvidenceCache& cache,
    const bool evidence) {
    auto found = cache.files.find(path);
    if (found == cache.files.end()) {
        found = cache.files.emplace(path,
            bounded_read(root, path, maximum_evidence_bytes, result,
                "invariant_ownership.reference_unreadable")).first;
    }
    if (evidence && found->second) {
        ++result.evidence_references_checked;
    }
    return found->second.has_value();
}

bool check_directory_path(
    const fs::path& root,
    const std::string& path,
    InvariantOwnershipCheckResult& result) {
    if (!engine::is_safe_relative_path(path)) {
        finding(result, EvidenceCategory::Violation, "invariant_ownership.reference_unreadable",
            "path=" + path + " code=path.unsafe");
        return false;
    }
    std::error_code error;
    auto current = root;
    std::size_t begin = 0U;
    while (begin < path.size()) {
        const auto end = path.find('/', begin);
        current /= path.substr(begin, end == std::string::npos ? std::string::npos : end - begin);
        const auto status = fs::symlink_status(current, error);
        if (error || status.type() == fs::file_type::not_found) {
            finding(result, EvidenceCategory::Violation, "invariant_ownership.reference_unreadable",
                "path=" + path + " code=missing");
            return false;
        }
        if (status.type() == fs::file_type::symlink) {
            finding(result, EvidenceCategory::Violation, "invariant_ownership.reference_unreadable",
                "path=" + path + " code=path.component_unsafe");
            return false;
        }
        if (end == std::string::npos) {
            if (status.type() != fs::file_type::directory) {
                finding(result, EvidenceCategory::Violation, "invariant_ownership.reference_unreadable",
                    "path=" + path + " code=path.not_directory");
                return false;
            }
            break;
        }
        if (status.type() != fs::file_type::directory) {
            finding(result, EvidenceCategory::Violation, "invariant_ownership.reference_unreadable",
                "path=" + path + " code=path.component_not_directory");
            return false;
        }
        begin = end + 1U;
    }
    return true;
}

struct TestReference {
    std::string path;
    std::vector<std::string> cases;
};

std::vector<TestReference> check_test_references(
    const Json& references,
    const std::string& invariant_id,
    const std::string& kind,
    const std::size_t minimum,
    const fs::path& root,
    InvariantOwnershipCheckResult& result,
    EvidenceCache& cache) {
    std::vector<TestReference> checked;
    if (!references.is_array() || references.size() < minimum || references.size() > maximum_references) {
        finding(result, EvidenceCategory::Violation, "invariant_ownership.regression_shape",
            "invariant_id=" + invariant_id + " kind=" + kind + " reason=count");
        return checked;
    }
    std::string prior_path;
    for (const auto& reference : references) {
        if (!exact_fields(reference, {"cases", "path"}) || !path_token(reference.at("path"))) {
            finding(result, EvidenceCategory::Violation, "invariant_ownership.regression_shape",
                "invariant_id=" + invariant_id + " kind=" + kind + " reason=field_set_or_path");
            continue;
        }
        const auto path = reference.at("path").get<std::string>();
        if (!prior_path.empty() && prior_path >= path) {
            finding(result, EvidenceCategory::Violation, "invariant_ownership.regression_order",
                "invariant_id=" + invariant_id + " kind=" + kind + " path=" + path);
        }
        prior_path = path;
        if (!sorted_unique_string_array(reference.at("cases"), 1U, maximum_cases,
                [](const Json& item) { return token(item); }, result,
                "invariant_ownership.regression_cases", invariant_id + ":" + kind + ":" + path)) {
            continue;
        }
        TestReference value{path, {}};
        for (const auto& item : reference.at("cases")) {
            value.cases.push_back(item.get<std::string>());
        }
        if (check_regular_path(root, path, result, cache, true)) {
            const auto& contents = *cache.files.at(path);
            for (const auto& case_name : value.cases) {
                if (!has_definition(contents, path, case_name)) {
                    finding(result, EvidenceCategory::Violation,
                        "invariant_ownership.regression_case_missing",
                        "invariant_id=" + invariant_id + " kind=" + kind +
                        " path=" + path + " case=" + case_name);
                }
            }
            if (kind == "real_process") {
                const bool go_process_evidence = path.ends_with(".go") &&
                    contents.find("exec.Command(") != std::string::npos &&
                    contents.find(".Stdin") != std::string::npos &&
                    (contents.find(".Output()") != std::string::npos ||
                     contents.find(".CombinedOutput()") != std::string::npos);
                const bool swift_process_evidence = path.ends_with(".swift") &&
                    contents.find("Process()") != std::string::npos &&
                    contents.find("standardInput") != std::string::npos &&
                    contents.find("standardOutput") != std::string::npos &&
                    contents.find(".run()") != std::string::npos &&
                    contents.find("waitUntilExit()") != std::string::npos;
                const bool swift_fixed_no_input_process_evidence = path.ends_with(".swift") &&
                    contents.find("Process()") != std::string::npos &&
                    contents.find("process.arguments = [\"readiness\"]") != std::string::npos &&
                    contents.find("standardOutput") != std::string::npos &&
                    contents.find(".run()") != std::string::npos &&
                    contents.find("waitUntilExit()") != std::string::npos;
                const bool shell_process_evidence = path.ends_with(".sh") &&
                    contents.find("symphony-ssiag-source") != std::string::npos &&
                    contents.find("symphony-ssiag-provider-macos-keychain") != std::string::npos &&
                    contents.find("ssiag provider verify") != std::string::npos &&
                    contents.find("SERVER_PID=$!") != std::string::npos &&
                    contents.find("wait \"$SERVER_PID\"") != std::string::npos;
                const bool process_evidence = go_process_evidence || swift_process_evidence ||
                    swift_fixed_no_input_process_evidence || shell_process_evidence;
                if (!process_evidence) {
                    finding(result, EvidenceCategory::Violation,
                        "invariant_ownership.real_process_mechanics",
                        "invariant_id=" + invariant_id + " path=" + path);
                }
            }
        }
        checked.push_back(std::move(value));
    }
    return checked;
}

struct AdapterEvidence {
    std::string id;
    std::string module_root;
};

std::map<std::string, AdapterEvidence> check_adapters(
    const Json& adapters,
    const fs::path& root,
    InvariantOwnershipCheckResult& result,
    EvidenceCache& cache) {
    std::map<std::string, AdapterEvidence> inventory;
    if (!adapters.is_array() || adapters.size() > maximum_adapters) {
        finding(result, EvidenceCategory::Violation, "invariant_ownership.adapter_shape",
            "reason=count");
        return inventory;
    }
    std::string prior_id;
    std::set<std::string> entry_points;
    std::set<std::string> global_operations;
    for (const auto& adapter : adapters) {
        if (!exact_fields(adapter, {"adapter_id", "command_protocol", "component", "entry_point_id",
                "format_version", "implementation_path", "operation_ids", "owner_contract",
                "version_policy"}) ||
            !invariant_or_adapter_identifier(adapter.value("adapter_id", Json{}), "adapter:symphony:") ||
            !token(adapter.value("component", Json{})) || !token(adapter.value("entry_point_id", Json{})) ||
            !adapter.at("format_version").is_number_unsigned() ||
            adapter.at("format_version").get<unsigned>() != 1U ||
            !path_token(adapter.value("owner_contract", Json{})) ||
            !path_token(adapter.value("implementation_path", Json{})) ||
            !adapter.at("command_protocol").is_string() ||
            (adapter.at("command_protocol").get<std::string>() != "symphony.foundation.lifecycle-command.v1" &&
             adapter.at("command_protocol").get<std::string>() != "symphony.ssiag.provider.control.v1") ||
            !adapter.at("version_policy").is_string() ||
            adapter.at("version_policy").get<std::string>() !=
                "exact_receipt_v2_entry_point_and_capability_compatible") {
            finding(result, EvidenceCategory::Violation, "invariant_ownership.adapter_shape",
                "reason=field_set_type_or_value");
            continue;
        }
        const auto id = adapter.at("adapter_id").get<std::string>();
        const auto component = adapter.at("component").get<std::string>();
        const auto entry_point = adapter.at("entry_point_id").get<std::string>();
        const auto command_protocol = adapter.at("command_protocol").get<std::string>();
        const bool known_pair =
            (component == "ssiag" && entry_point == "ssiag.foundation-lifecycle" &&
             command_protocol == "symphony.foundation.lifecycle-command.v1") ||
            (component == "ssiag" && entry_point == "ssiag.macos-keychain-provider" &&
             command_protocol == "symphony.ssiag.provider.control.v1") ||
            (component == "stav" && entry_point == "stav.foundation-lifecycle" &&
             command_protocol == "symphony.foundation.lifecycle-command.v1");
        const auto expected_id = "adapter:symphony:" + entry_point + ".v1";
        if (!known_pair || id != expected_id) {
            finding(result, EvidenceCategory::Violation, "invariant_ownership.adapter_identity",
                "adapter_id=" + id + " component=" + component + " entry_point_id=" + entry_point);
        }
        if (!prior_id.empty() && prior_id >= id) {
            finding(result, EvidenceCategory::Violation, "invariant_ownership.adapter_order",
                "adapter_id=" + id);
        }
        prior_id = id;
        if (!inventory.emplace(id, AdapterEvidence{id, {}}).second || !entry_points.insert(entry_point).second) {
            finding(result, EvidenceCategory::Violation, "invariant_ownership.adapter_identity",
                "adapter_id=" + id + " reason=duplicate_id_or_entry_point");
            continue;
        }
        const auto owner_contract = adapter.at("owner_contract").get<std::string>();
        const auto implementation_path = adapter.at("implementation_path").get<std::string>();
        static_cast<void>(check_regular_path(root, owner_contract, result, cache, false));
        static_cast<void>(check_directory_path(root, implementation_path, result));
        const auto internal = implementation_path.find("/internal/");
        const auto sources = implementation_path.find("/Sources/");
        const auto boundary = internal != std::string::npos ? internal : sources;
        inventory.at(id).module_root = boundary == std::string::npos
            ? implementation_path : implementation_path.substr(0U, boundary);
        if (sorted_unique_string_array(adapter.at("operation_ids"), 1U, maximum_operations,
                [](const Json& item) { return operation_identifier(item); }, result,
                "invariant_ownership.adapter_operations", id)) {
            if (id == "adapter:symphony:ssiag.macos-keychain-provider.v1") {
                const Json expected_operations = Json::array({
                    "engop:symphony:ssiag.macos-keychain-provider.readiness.observe",
                    "engop:symphony:ssiag.provider.metadata-capabilities",
                    "engop:symphony:ssiag.provider.metadata-handshake",
                    "engop:symphony:ssiag.provider.metadata-status",
                });
                if (adapter.at("operation_ids") != expected_operations) {
                    finding(result, EvidenceCategory::Violation,
                        "invariant_ownership.adapter_operations",
                        "adapter_id=" + id + " reason=not_adapter_owned_operation_set");
                }
                const auto protocol_source = implementation_path + "/Protocol.swift";
                const auto readiness_source = implementation_path + "/SignedBundleReadiness.swift";
                const bool protocol_present = check_regular_path(root, protocol_source, result, cache, false);
                const bool readiness_present = check_regular_path(root, readiness_source, result, cache, false);
                for (const auto& operation : expected_operations) {
                    const auto operation_id = operation.get<std::string>();
                    const bool readiness_operation = operation_id.find(".readiness.") != std::string::npos;
                    const auto& source = readiness_operation ? readiness_source : protocol_source;
                    const bool source_present = readiness_operation ? readiness_present : protocol_present;
                    if (source_present && cache.files.at(source)->find(operation_id) == std::string::npos) {
                        finding(result, EvidenceCategory::Violation,
                            "invariant_ownership.adapter_operation_owner",
                            "adapter_id=" + id + " operation_id=" + operation_id +
                                " reason=missing_from_implementation");
                    }
                }
            }
            for (const auto& operation : adapter.at("operation_ids")) {
                const auto operation_id = operation.get<std::string>();
                if (!global_operations.insert(operation_id).second) {
                    finding(result, EvidenceCategory::Violation,
                        "invariant_ownership.adapter_operation_owner",
                        "operation_id=" + operation_id + " reason=multiple_adapters");
                }
            }
        }
        ++result.adapters_checked;
    }
    return inventory;
}

void check_invariants(
    const Json& invariants,
    const std::map<std::string, AdapterEvidence>& adapters,
    const fs::path& root,
    InvariantOwnershipCheckResult& result,
    EvidenceCache& cache) {
    if (!invariants.is_array() || invariants.empty() || invariants.size() > maximum_invariants) {
        finding(result, EvidenceCategory::Violation, "invariant_ownership.invariant_shape",
            "reason=count");
        return;
    }
    std::string prior_id;
    std::set<std::string> identities;
    std::set<std::string> referenced_adapters;
    for (const auto& invariant : invariants) {
        if (!exact_fields(invariant, {"allowed_adapter_ids", "consumer_boundary_rejections",
                "invariant_id", "ipc_boundary", "owner_component", "owner_contract",
                "producer_implementations", "producer_regressions", "real_process_regressions",
                "statement", "status", "title"}) ||
            !invariant_or_adapter_identifier(
                invariant.value("invariant_id", Json{}), "invariant:symphony:") ||
            !path_token(invariant.value("owner_contract", Json{})) ||
            !token(invariant.value("owner_component", Json{})) ||
            !visible_text(invariant.value("title", Json{}), 256U) ||
            !visible_text(invariant.value("statement", Json{}), 4096U) ||
            !invariant.at("ipc_boundary").is_boolean() ||
            !invariant.at("status").is_string() || invariant.at("status").get<std::string>() != "active") {
            finding(result, EvidenceCategory::Violation, "invariant_ownership.invariant_shape",
                "reason=field_set_type_or_value");
            continue;
        }
        const auto id = invariant.at("invariant_id").get<std::string>();
        if (!prior_id.empty() && prior_id >= id) {
            finding(result, EvidenceCategory::Violation, "invariant_ownership.invariant_order",
                "invariant_id=" + id);
        }
        prior_id = id;
        if (!identities.insert(id).second) {
            finding(result, EvidenceCategory::Violation, "invariant_ownership.invariant_identity",
                "invariant_id=" + id + " reason=duplicate");
            continue;
        }
        static_cast<void>(check_regular_path(root,
            invariant.at("owner_contract").get<std::string>(), result, cache, false));
        if (sorted_unique_string_array(invariant.at("producer_implementations"), 1U,
                maximum_references, [](const Json& item) { return path_token(item); }, result,
                "invariant_ownership.producer_implementations", id)) {
            for (const auto& implementation : invariant.at("producer_implementations")) {
                static_cast<void>(check_regular_path(root, implementation.get<std::string>(),
                    result, cache, false));
            }
        }
        const auto producer = check_test_references(invariant.at("producer_regressions"), id,
            "producer", 1U, root, result, cache);
        const auto consumer = check_test_references(invariant.at("consumer_boundary_rejections"), id,
            "consumer", 1U, root, result, cache);
        static_cast<void>(producer);
        static_cast<void>(consumer);

        std::vector<std::string> allowed;
        if (sorted_unique_string_array(invariant.at("allowed_adapter_ids"), 0U, maximum_adapters,
                [](const Json& item) {
                    return invariant_or_adapter_identifier(item, "adapter:symphony:");
                }, result,
                "invariant_ownership.adapter_references", id)) {
            for (const auto& reference : invariant.at("allowed_adapter_ids")) {
                const auto adapter_id = reference.get<std::string>();
                allowed.push_back(adapter_id);
                referenced_adapters.insert(adapter_id);
                if (!adapters.contains(adapter_id)) {
                    finding(result, EvidenceCategory::Violation,
                        "invariant_ownership.adapter_reference_unresolved",
                        "invariant_id=" + id + " adapter_id=" + adapter_id);
                }
            }
        }
        const bool ipc = invariant.at("ipc_boundary").get<bool>();
        const auto real_process = check_test_references(invariant.at("real_process_regressions"), id,
            "real_process", ipc ? 1U : 0U, root, result, cache);
        std::set<std::string> evidence_relations;
        for (const auto* collection : {&producer, &consumer, &real_process}) {
            for (const auto& reference : *collection) {
                for (const auto& case_name : reference.cases) {
                    const auto relation = reference.path + "\n" + case_name;
                    if (!evidence_relations.insert(relation).second) {
                        finding(result, EvidenceCategory::Violation,
                            "invariant_ownership.regression_role_collision",
                            "invariant_id=" + id + " path=" + reference.path +
                            " case=" + case_name);
                    }
                }
            }
        }
        if (!ipc && !real_process.empty()) {
            finding(result, EvidenceCategory::Violation,
                "invariant_ownership.real_process_policy",
                "invariant_id=" + id + " reason=non_ipc_has_real_process_regression");
        }
        if (ipc && allowed.empty()) {
            finding(result, EvidenceCategory::Violation,
                "invariant_ownership.real_process_policy",
                "invariant_id=" + id + " reason=ipc_has_no_adapter");
        }
        if (ipc) {
            for (const auto& adapter_id : allowed) {
                const auto adapter = adapters.find(adapter_id);
                if (adapter == adapters.end()) {
                    continue;
                }
                const auto prefix = adapter->second.module_root + "/";
                const auto covered = std::any_of(real_process.begin(), real_process.end(),
                    [&prefix](const TestReference& reference) {
                        return reference.path.starts_with(prefix);
                    });
                if (!covered) {
                    finding(result, EvidenceCategory::Violation,
                        "invariant_ownership.real_process_adapter_coverage",
                        "invariant_id=" + id + " adapter_id=" + adapter_id);
                }
            }
        }
        ++result.invariants_checked;
    }
    for (const auto& [adapter_id, ignored] : adapters) {
        static_cast<void>(ignored);
        if (!referenced_adapters.contains(adapter_id)) {
            finding(result, EvidenceCategory::Violation,
                "invariant_ownership.adapter_unreferenced", "adapter_id=" + adapter_id);
        }
    }
}

}

InvariantOwnershipCheckResult check_invariant_ownership(const std::string& repo_root) {
    InvariantOwnershipCheckResult result{true, {}, 0U, 0U, 0U, 0U};
    std::error_code error;
    const auto root = fs::canonical(fs::path(repo_root), error);
    if (error) {
        finding(result, EvidenceCategory::Violation, "invariant_ownership.unreadable",
            "path=. code=root_unreadable");
        return result;
    }
    const auto status = fs::symlink_status(root / registry_path, error);
    if (error || status.type() == fs::file_type::not_found) {
        finding(result, EvidenceCategory::Absent, "invariant_ownership.absent",
            "registry=false");
        return result;
    }
    const auto contents = bounded_read(root, std::string(registry_path), maximum_registry_bytes, result);
    if (!contents) {
        return result;
    }
    std::optional<Json> registry;
    try {
        registry = engine::parse_bounded_json(*contents, maximum_registry_bytes);
    } catch (const engine::Error& parse_error) {
        finding(result, EvidenceCategory::Violation, "invariant_ownership.invalid_json",
            "path=" + std::string(registry_path) + " code=" + parse_error.code());
        return result;
    } catch (const std::exception&) {
        finding(result, EvidenceCategory::Violation, "invariant_ownership.invalid_json",
            "path=" + std::string(registry_path) + " code=unexpected_error");
        return result;
    }
    if (!exact_fields(*registry, {"adapters", "catalog_complete", "catalog_scope", "format_version",
            "forward_gate", "invariants", "protocol", "registry_digest", "scope", "test_policy"}) ||
        !registry->at("protocol").is_string() ||
        registry->at("protocol").get<std::string>() !=
            "symphony.knowledge.invariant-ownership-registry.v1" ||
        !registry->at("format_version").is_number_unsigned() ||
        registry->at("format_version").get<unsigned>() != 1U ||
        !registry->at("scope").is_string() ||
        registry->at("scope").get<std::string>() != "common_lowest_authoritative_layer" ||
        !registry->at("catalog_scope").is_string() ||
        registry->at("catalog_scope").get<std::string>() != "registered_incremental" ||
        !registry->at("catalog_complete").is_boolean() || registry->at("catalog_complete").get<bool>() ||
        !registry->at("forward_gate").is_string() ||
        registry->at("forward_gate").get<std::string>() != "enforce_new_or_modified" ||
        !exact_fields(registry->at("test_policy"), {"consumer_boundary_rejection_required",
            "owner_producer_regression_required", "real_process_required_for_ipc"}) ||
        !registry->at("test_policy").at("consumer_boundary_rejection_required").is_boolean() ||
        !registry->at("test_policy").at("consumer_boundary_rejection_required").get<bool>() ||
        !registry->at("test_policy").at("owner_producer_regression_required").is_boolean() ||
        !registry->at("test_policy").at("owner_producer_regression_required").get<bool>() ||
        !registry->at("test_policy").at("real_process_required_for_ipc").is_boolean() ||
        !registry->at("test_policy").at("real_process_required_for_ipc").get<bool>()) {
        finding(result, EvidenceCategory::Violation, "invariant_ownership.registry_shape",
            "reason=field_set_type_or_value");
        return result;
    }
    if (!tagged_digest(registry->at("registry_digest"))) {
        finding(result, EvidenceCategory::Violation, "invariant_ownership.registry_digest",
            "reason=missing_or_invalid");
    } else {
        const auto expected = registry->at("registry_digest").get<std::string>();
        auto preimage = *registry;
        preimage.erase("registry_digest");
        const auto observed = engine::tagged_sha256(preimage.dump());
        if (expected != observed) {
            finding(result, EvidenceCategory::Violation, "invariant_ownership.registry_digest",
                "expected=" + expected + " observed=" + observed);
        } else {
            finding(result, EvidenceCategory::Pass, "invariant_ownership.registry_digest",
                "digest=" + expected);
        }
    }

    EvidenceCache cache;
    const auto adapters = check_adapters(registry->at("adapters"), root, result, cache);
    check_invariants(registry->at("invariants"), adapters, root, result, cache);
    const auto final_violations = result.violations;
    finding(result, result.success ? EvidenceCategory::Pass : EvidenceCategory::Violation,
        "invariant_ownership.scan_complete",
        "invariants=" + std::to_string(result.invariants_checked) +
        " adapters=" + std::to_string(result.adapters_checked) +
        " evidence_references=" + std::to_string(result.evidence_references_checked) +
        " violations=" + std::to_string(final_violations));
    return result;
}
