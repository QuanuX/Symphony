#include "sacv.hpp"

#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/limits.hpp"
#include "symphony/knowledge/engine/path.hpp"

#include <algorithm>
#include <array>
#include <chrono>
#include <cctype>
#include <cmath>
#include <filesystem>
#include <map>
#include <optional>
#include <set>
#include <string>
#include <string_view>
#include <utility>
#include <vector>

namespace symphony::knowledge::sacv {
namespace engine = symphony::knowledge::engine;
namespace fs = std::filesystem;

namespace {

constexpr std::size_t max_entries = 256;
constexpr std::size_t max_evidence = 1024;
constexpr std::size_t max_changes = 2048;
constexpr std::size_t max_field_bytes = 64U * 1024U;
constexpr const char* registry_path = "knowledge/sacv/REGISTRY.md";
constexpr const char* skvi_index_path = "knowledge/skvi/INDEX.md";
constexpr const char* check_protocol = "symphony.sacv.check-result.v1";
constexpr const char* diff_protocol = "symphony.sacv.diff-result.v1";
constexpr const char* projection_protocol = "symphony.sacv.projection.v1";
constexpr const char* proposal_protocol = "symphony.knowledge.proposal.v1";

constexpr std::array<const char*, 13> entry_fields = {
    "api_id", "title", "owner", "path", "openapi", "api_version", "audience",
    "transport_profile", "security_profile", "publication_state", "sdk_state",
    "status", "notes",
};

const std::vector<std::string> contract_paths = {
    "knowledge/SPEC.md",
    "knowledge/sacv/INTENT.md",
    "knowledge/sacv/MANIFEST.md",
    "knowledge/sacv/SKILL.md",
    "knowledge/sacv/SPEC.md",
    "knowledge/sacv/profiles/openapi-3.2.md",
    "knowledge/sacv/profiles/mintlify-publication.md",
    "knowledge/sacv/schemas/v1/MANIFEST.md",
    "knowledge/sacv/schemas/v1/registry-entry.schema.json",
    "knowledge/sacv/schemas/v1/check-result.schema.json",
    "knowledge/sacv/schemas/v1/diff-input.schema.json",
    "knowledge/sacv/schemas/v1/diff-result.schema.json",
    "knowledge/sacv/schemas/v1/proposal-input.schema.json",
    "knowledge/sacv/schemas/v1/projection.schema.json",
    skvi_index_path,
};

struct Entry final {
    std::map<std::string, std::string> fields;
    std::size_t line = 0;
};

struct Finding final {
    std::string severity;
    std::string code;
    std::string path;
    std::string detail;
};

struct Operation final {
    std::string key;
    std::string operation_id;
    bool request_body_required = false;
    bool protected_operation = false;
    std::set<std::string> response_codes;
    std::string digest;
};

struct Document final {
    engine::FileDigest file;
    std::string title;
    std::string version;
    std::map<std::string, Operation> operations;
};

struct RegistryState final {
    std::string contents;
    engine::FileDigest registry_file;
    engine::Snapshot contract_snapshot;
    std::vector<Entry> entries;
    std::map<std::string, Document> documents;
    std::vector<Finding> findings;
    std::size_t passes = 0;
    std::size_t operations_checked = 0;
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

std::string normalize_line(std::string_view line) {
    std::size_t begin = 0;
    while (begin < line.size() && (line[begin] == ' ' || line[begin] == '\t' || line[begin] == '-')) {
        ++begin;
    }
    return trim(line.substr(begin));
}

std::string clean_value(std::string value) {
    value = trim(value);
    if (value.size() >= 2U && value.front() == '`' && value.back() == '`') {
        value = value.substr(1U, value.size() - 2U);
    }
    return value;
}

std::string lower(std::string value) {
    std::transform(value.begin(), value.end(), value.begin(), [](const unsigned char character) {
        return static_cast<char>(std::tolower(character));
    });
    return value;
}

bool printable_bounded(std::string_view value, std::size_t max_bytes, bool allow_empty = false) {
    if ((!allow_empty && value.empty()) || value.size() > max_bytes) {
        return false;
    }
    return std::all_of(value.begin(), value.end(), [](const unsigned char character) {
        return character == '\n' || character == '\t' ||
               (character >= 0x20U && character != 0x7fU);
    });
}

bool safe_token(std::string_view value, std::size_t max_bytes = engine::Limits::max_token_bytes) {
    if (value.empty() || value.size() > max_bytes) {
        return false;
    }
    return std::all_of(value.begin(), value.end(), [](const unsigned char character) {
        const bool alphanumeric =
            (character >= 'a' && character <= 'z') ||
            (character >= 'A' && character <= 'Z') ||
            (character >= '0' && character <= '9');
        return alphanumeric || character == '.' || character == '_' ||
               character == ':' || character == '-';
    });
}

bool tagged_digest(std::string_view value) {
    if (value.size() != 71U || !value.starts_with("sha256:")) {
        return false;
    }
    return std::all_of(value.begin() + 7, value.end(), [](const unsigned char character) {
        return (character >= '0' && character <= '9') ||
               (character >= 'a' && character <= 'f');
    });
}

bool strict_utc(std::string_view value) {
    if (value.size() != 20U || value[4] != '-' || value[7] != '-' || value[10] != 'T' ||
        value[13] != ':' || value[16] != ':' || value[19] != 'Z') {
        return false;
    }
    for (const std::size_t index : {0U, 1U, 2U, 3U, 5U, 6U, 8U, 9U, 11U, 12U, 14U, 15U, 17U, 18U}) {
        if (value[index] < '0' || value[index] > '9') {
            return false;
        }
    }
    const auto number = [&](std::size_t begin, std::size_t count) {
        int result = 0;
        for (std::size_t index = begin; index < begin + count; ++index) {
            result = result * 10 + (value[index] - '0');
        }
        return result;
    };
    const auto date = std::chrono::year{number(0, 4)} /
                      std::chrono::month{static_cast<unsigned int>(number(5, 2))} /
                      std::chrono::day{static_cast<unsigned int>(number(8, 2))};
    return date.ok() && number(11, 2) <= 23 && number(14, 2) <= 59 && number(17, 2) <= 59;
}

void require_exact_fields(const engine::Json& object, const std::set<std::string>& fields) {
    if (!object.is_object() || object.size() != fields.size()) {
        throw engine::Error("payload.field_set", "operation payload is incomplete or contains unknown fields", 4);
    }
    for (const auto& [key, value] : object.items()) {
        static_cast<void>(value);
        if (!fields.contains(key)) {
            throw engine::Error("payload.unknown_field", "operation payload contains an unknown field", 4);
        }
    }
}

std::string require_string(const engine::Json& object, const char* field,
                           std::size_t max_bytes, bool token = false) {
    const auto& value = object.at(field);
    if (!value.is_string()) {
        throw engine::Error("payload.invalid_field", std::string(field) + " must be a string", 4);
    }
    const auto text = value.get<std::string>();
    if ((token && !safe_token(text, max_bytes)) || (!token && !printable_bounded(text, max_bytes))) {
        throw engine::Error("payload.invalid_field", std::string(field) + " has invalid syntax", 4);
    }
    return text;
}

void add_finding(RegistryState& state, Finding finding) {
    if (finding.severity == "pass") {
        ++state.passes;
        return;
    }
    if (state.findings.size() >= max_evidence) {
        throw engine::Error("sacv.evidence_limit", "SACV evidence item limit exceeded", 5);
    }
    state.findings.push_back(std::move(finding));
}

std::size_t finding_count(const RegistryState& state, const std::string& severity) {
    if (severity == "pass") {
        return state.passes;
    }
    return static_cast<std::size_t>(std::count_if(
        state.findings.begin(), state.findings.end(), [&](const Finding& finding) {
            return finding.severity == severity;
        }));
}

engine::Json file_json(const engine::FileDigest& file) {
    return engine::Json{{"path", file.path}, {"size", file.size}, {"digest", file.digest}};
}

engine::Json snapshot_json(const engine::Snapshot& snapshot) {
    auto files = engine::Json::array();
    for (const auto& file : snapshot.files) {
        files.push_back(file_json(file));
    }
    return engine::Json{{"digest", snapshot.digest}, {"files", std::move(files)}};
}

class BoundedOpenApiSax final : public nlohmann::json_sax<engine::Json> {
public:
    bool null() override { return value(); }
    bool boolean(bool) override { return value(); }
    bool number_integer(number_integer_t) override { return value(); }
    bool number_unsigned(number_unsigned_t) override { return value(); }
    bool number_float(number_float_t number, const string_t&) override {
        if (!std::isfinite(number)) {
            failure_ = "non-finite number";
            return false;
        }
        return value();
    }
    bool string(string_t& text) override {
        if (text.size() > engine::Limits::max_string_bytes) {
            failure_ = "string bound exceeded";
            return false;
        }
        return value();
    }
    bool binary(binary_t&) override {
        failure_ = "binary value is not JSON";
        return false;
    }
    bool start_object(std::size_t) override {
        if (!container()) {
            return false;
        }
        object_keys_.emplace_back(std::set<std::string>{});
        return true;
    }
    bool key(string_t& text) override {
        if (text.size() > engine::Limits::max_string_bytes || object_keys_.empty() ||
            !object_keys_.back().has_value() || !object_keys_.back()->insert(text).second) {
            failure_ = "duplicate or invalid object key";
            return false;
        }
        return value();
    }
    bool end_object() override {
        if (object_keys_.empty() || !object_keys_.back().has_value()) {
            failure_ = "object nesting mismatch";
            return false;
        }
        object_keys_.pop_back();
        return true;
    }
    bool start_array(std::size_t) override {
        if (!container()) {
            return false;
        }
        object_keys_.emplace_back(std::nullopt);
        return true;
    }
    bool end_array() override {
        if (object_keys_.empty() || object_keys_.back().has_value()) {
            failure_ = "array nesting mismatch";
            return false;
        }
        object_keys_.pop_back();
        return true;
    }
    bool parse_error(std::size_t, const std::string&, const nlohmann::detail::exception&) override {
        failure_ = "invalid JSON syntax";
        return false;
    }
    [[nodiscard]] const std::string& failure() const { return failure_; }

private:
    bool value() {
        ++values_;
        if (values_ > engine::Limits::max_json_values) {
            failure_ = "JSON value bound exceeded";
            return false;
        }
        return true;
    }
    bool container() {
        if (!value()) {
            return false;
        }
        if (object_keys_.size() >= engine::Limits::max_json_depth) {
            failure_ = "JSON depth bound exceeded";
            return false;
        }
        return true;
    }

    std::size_t values_ = 0;
    std::vector<std::optional<std::set<std::string>>> object_keys_;
    std::string failure_;
};

engine::Json parse_openapi_json(const std::string& input) {
    if (input.empty() || input.size() > engine::Limits::max_snapshot_file_bytes) {
        throw engine::Error("sacv.document.json_invalid", "OpenAPI JSON byte bound violated", 4);
    }
    BoundedOpenApiSax sax;
    if (!engine::Json::sax_parse(input, &sax)) {
        throw engine::Error("sacv.document.json_invalid", "OpenAPI JSON is invalid: " + sax.failure(), 4);
    }
    try {
        auto result = engine::Json::parse(input);
        if (!result.is_object()) {
            throw engine::Error("sacv.document.json_invalid", "OpenAPI JSON root must be an object", 4);
        }
        return result;
    } catch (const engine::Error&) {
        throw;
    } catch (const std::exception&) {
        throw engine::Error("sacv.document.json_invalid", "OpenAPI JSON could not be materialized", 4);
    }
}

std::string detected_field(const std::string& line) {
    for (const auto* field : entry_fields) {
        const auto prefix = std::string(field) + ':';
        if (line.starts_with(prefix)) {
            return field;
        }
    }
    return {};
}

std::vector<Entry> parse_entries(const std::string& contents, RegistryState& state) {
    std::vector<Entry> result;
    Entry current;
    bool active = false;
    std::string active_field;
    std::size_t next_field = 0;
    std::size_t line_number = 0;
    std::size_t position = 0;
    auto finish = [&]() {
        if (active) {
            result.push_back(std::move(current));
            current = Entry{};
            active = false;
            active_field.clear();
            next_field = 0;
        }
    };
    while (position <= contents.size()) {
        const auto end = contents.find('\n', position);
        const auto raw = contents.substr(position, end == std::string::npos ? std::string::npos : end - position);
        ++line_number;
        const auto normalized = normalize_line(raw);
        const auto field = detected_field(normalized);
        if (field == "api_id") {
            finish();
            active = true;
            current.line = line_number;
        }
        if (active && !field.empty()) {
            const auto position_in_contract = static_cast<std::size_t>(
                std::distance(entry_fields.begin(), std::find(entry_fields.begin(), entry_fields.end(), field)));
            if (position_in_contract != next_field) {
                add_finding(state, Finding{"violation", "sacv.registry.field_order_invalid", registry_path,
                    "line=" + std::to_string(line_number) + " field=" + field});
            }
            next_field = position_in_contract + 1U;
            if (current.fields.contains(field)) {
                add_finding(state, Finding{"violation", "sacv.registry.duplicate_field", registry_path,
                    "line=" + std::to_string(line_number) + " field=" + field});
            } else {
                current.fields[field] = clean_value(normalized.substr(normalized.find(':') + 1U));
            }
            active_field = field;
        } else if (active && !normalized.empty() && normalized.contains(':') &&
                   trim(raw).starts_with('-')) {
            add_finding(state, Finding{"violation", "sacv.registry.unknown_field", registry_path,
                "line=" + std::to_string(line_number)});
            active_field.clear();
        } else if (active && !active_field.empty() && !normalized.empty() && !normalized.starts_with('#')) {
            auto value = clean_value(normalized);
            if (!value.empty()) {
                auto& target = current.fields[active_field];
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
    if (result.size() > max_entries) {
        throw engine::Error("sacv.registry.entry_limit", "SACV registry entry-count limit exceeded", 5);
    }
    if (result.empty()) {
        if (contents.find("## Canonical Entries") == std::string::npos ||
            contents.find("None.") == std::string::npos) {
            add_finding(state, Finding{"violation", "sacv.registry.empty_marker_missing", registry_path,
                "empty registry must explicitly declare None."});
        } else {
            add_finding(state, Finding{"pass", "sacv.registry.empty_valid", registry_path,
                "empty registry is explicit and valid"});
        }
    }
    return result;
}

engine::Json entry_json_without_digest(const Entry& entry) {
    auto result = engine::Json::object();
    for (const auto* field : entry_fields) {
        result[field] = entry.fields.contains(field) ? entry.fields.at(field) : "";
    }
    return result;
}

engine::Json entry_json(const Entry& entry) {
    auto result = entry_json_without_digest(entry);
    result["entry_digest"] = engine::tagged_sha256(result.dump());
    return result;
}

bool exact_enum(const std::string& value, std::initializer_list<const char*> options) {
    return std::any_of(options.begin(), options.end(), [&](const char* option) { return value == option; });
}

bool owner_path(const std::string& value) {
    if (!engine::is_safe_relative_path(value) || value.ends_with('/')) {
        return false;
    }
    const auto slash = value.find('/');
    return slash != std::string::npos && value.find('/', slash + 1U) == std::string::npos &&
           (value.starts_with("knowledge/") || value.starts_with("modules/"));
}

bool registered_document_path(const std::string& path) {
    return engine::is_safe_relative_path(path) &&
           (path.ends_with(".openapi.json") || path.ends_with(".openapi.yaml"));
}

bool canonical_profile_path(const std::string& path) {
    if (!engine::is_safe_relative_path(path) ||
        (!path.starts_with("knowledge/") && !path.starts_with("modules/"))) {
        return false;
    }
    const auto first = path.find('/');
    return first != std::string::npos && path.find('/', first + 1U) != std::string::npos;
}

void require_valid_entry_contract(const Entry& entry) {
    const auto& fields = entry.fields;
    const auto& api_id = fields.at("api_id");
    const auto& owner = fields.at("owner");
    const auto& path = fields.at("path");
    if (!safe_token(api_id) || api_id != lower(api_id)) {
        throw engine::Error("payload.invalid_entry_identity", "api_id must be a stable lowercase token", 4);
    }
    if (!owner_path(owner) || !registered_document_path(path) || !path.starts_with(owner + "/")) {
        throw engine::Error("payload.invalid_entry_ownership", "entry owner/path placement is invalid", 4);
    }
    if (fields.at("openapi") != "3.2.0" ||
        !exact_enum(fields.at("audience"), {"local_internal", "remote_administrative", "partner", "public"}) ||
        !exact_enum(fields.at("publication_state"), {"internal_only", "candidate", "sodv_approved"}) ||
        !exact_enum(fields.at("sdk_state"), {"not_eligible", "candidate", "approved"}) ||
        !exact_enum(fields.at("status"), {"draft", "ratified", "deprecated", "retired"}) ||
        !safe_token(fields.at("api_version")) || !safe_token(fields.at("transport_profile")) ||
        !canonical_profile_path(fields.at("security_profile"))) {
        throw engine::Error("payload.invalid_entry_classification", "entry version, profile, or lifecycle field is invalid", 4);
    }
}

bool indexed_by_skvi(const std::string& index, const std::string& path) {
    return index.find("- path: `" + path + "`") != std::string::npos ||
           index.find("- path: " + path + "\n") != std::string::npos;
}

bool nonempty_security(const engine::Json& value) {
    return value.is_array() && !value.empty();
}

void scan_refs_and_examples(const engine::Json& value, const std::string& current_key,
                            const Entry& entry, RegistryState& state) {
    const auto path = entry.fields.at("path");
    if (current_key == "$ref" && value.is_string() && !value.get_ref<const std::string&>().starts_with('#')) {
        add_finding(state, Finding{"violation", "sacv.reference.external_unavailable", path,
            "v1 implements repository-local fragment references only"});
    }
    if ((current_key == "example" || current_key == "examples") && !value.is_null()) {
        const auto content = lower(value.dump());
        for (const auto* marker : {"bearer ", "private key", "api_key", "api-key", "password", "credential", "secret"}) {
            if (content.find(marker) != std::string::npos) {
                add_finding(state, Finding{"violation", "sacv.example.secret_shaped", path,
                    std::string("example contains prohibited marker: ") + marker});
                break;
            }
        }
    }
    if (entry.fields.at("audience") != "public" && current_key == "servers" &&
        value.is_array() && !value.empty()) {
        add_finding(state, Finding{"violation", "sacv.server.internal_nonempty", path,
            "non-public contracts must not declare a live server target"});
    }
    if (value.is_object()) {
        for (const auto& [key, child] : value.items()) {
            scan_refs_and_examples(child, key, entry, state);
        }
    } else if (value.is_array()) {
        for (const auto& child : value) {
            scan_refs_and_examples(child, current_key, entry, state);
        }
    }
}

bool response_class(const std::string& code, char first) {
    return code.size() == 3U && code[0] == first &&
           std::isdigit(static_cast<unsigned char>(code[1])) != 0 &&
           std::isdigit(static_cast<unsigned char>(code[2])) != 0;
}

std::map<std::string, Operation> extract_operations(const engine::Json& document,
                                                     const std::string& path,
                                                     RegistryState* state,
                                                     const Entry* entry) {
    if (!document.contains("openapi") || document.at("openapi") != "3.2.0" ||
        !document.contains("info") || !document.at("info").is_object() ||
        !document.at("info").contains("title") || !document.at("info").at("title").is_string() ||
        document.at("info").at("title").get_ref<const std::string&>().empty() ||
        !document.at("info").contains("version") || !document.at("info").at("version").is_string() ||
        document.at("info").at("version").get_ref<const std::string&>().empty() ||
        !document.contains("paths") || !document.at("paths").is_object()) {
        throw engine::Error("sacv.document.profile_invalid",
            "OpenAPI document lacks required 3.2.0, info, or paths fields", 4);
    }
    if (document.contains("security") && !document.at("security").is_array() && state != nullptr) {
        add_finding(*state, Finding{"violation", "sacv.document.security_invalid", path,
            "top-level security must be an array"});
    }
    const bool root_security = document.contains("security") && nonempty_security(document.at("security"));
    const std::set<std::string> methods = {
        "get", "put", "post", "delete", "options", "head", "patch", "trace", "query",
    };
    std::map<std::string, Operation> operations;
    std::set<std::string> operation_ids;
    for (const auto& [route, path_item] : document.at("paths").items()) {
        if (!route.starts_with('/') || !path_item.is_object()) {
            if (state != nullptr) {
                add_finding(*state, Finding{"violation", "sacv.path_item.invalid", path, "route=" + route});
            }
            continue;
        }
        for (const auto& [method, operation_value] : path_item.items()) {
            const auto normalized_method = lower(method);
            if (!methods.contains(normalized_method)) {
                continue;
            }
            const auto key = normalized_method + " " + route;
            Operation operation;
            operation.key = key;
            if (!operation_value.is_object() || !operation_value.contains("operationId") ||
                !operation_value.at("operationId").is_string() ||
                !safe_token(operation_value.at("operationId").get_ref<const std::string&>())) {
                if (state != nullptr) {
                    add_finding(*state, Finding{"violation", "sacv.operation.operation_id_invalid", path, key});
                }
            } else {
                operation.operation_id = operation_value.at("operationId").get<std::string>();
                if (!operation_ids.insert(operation.operation_id).second && state != nullptr) {
                    add_finding(*state, Finding{"violation", "sacv.operation.operation_id_duplicate", path,
                        operation.operation_id});
                }
            }

            operation.protected_operation = root_security;
            if (operation_value.is_object() && operation_value.contains("security")) {
                if (!operation_value.at("security").is_array() && state != nullptr) {
                    add_finding(*state, Finding{"violation", "sacv.operation.security_invalid", path, key});
                }
                operation.protected_operation = nonempty_security(operation_value.at("security"));
            }
            if (entry != nullptr && entry->fields.at("audience") != "public" &&
                !operation.protected_operation && state != nullptr) {
                add_finding(*state, Finding{"violation", "sacv.operation.security_missing", path, key});
            }

            if (operation_value.is_object() && operation_value.contains("requestBody")) {
                const auto& request_body = operation_value.at("requestBody");
                if (!request_body.is_object()) {
                    if (state != nullptr) {
                        add_finding(*state, Finding{"violation", "sacv.operation.request_body_invalid", path, key});
                    }
                } else {
                    if (request_body.contains("required") && !request_body.at("required").is_boolean()) {
                        if (state != nullptr) {
                            add_finding(*state, Finding{"violation", "sacv.operation.request_body_required_invalid",
                                path, key});
                        }
                    } else {
                        operation.request_body_required =
                            request_body.contains("required") && request_body.at("required").get<bool>();
                    }
                    if (!request_body.contains("content") || !request_body.at("content").is_object() ||
                        request_body.at("content").empty()) {
                        if (state != nullptr) {
                            add_finding(*state, Finding{"violation", "sacv.operation.request_content_missing", path, key});
                        }
                    }
                }
            } else if ((normalized_method == "post" || normalized_method == "put" ||
                        normalized_method == "patch" || normalized_method == "query") && state != nullptr) {
                add_finding(*state, Finding{"violation", "sacv.operation.request_body_missing", path, key});
            }

            bool success = false;
            bool error = false;
            if (!operation_value.is_object() || !operation_value.contains("responses") ||
                !operation_value.at("responses").is_object() || operation_value.at("responses").empty()) {
                if (state != nullptr) {
                    add_finding(*state, Finding{"violation", "sacv.operation.responses_missing", path, key});
                }
            } else {
                for (const auto& [code, response] : operation_value.at("responses").items()) {
                    operation.response_codes.insert(code);
                    success = success || response_class(code, '2');
                    error = error || response_class(code, '4') || response_class(code, '5') || code == "default";
                    if (!response.is_object() || !response.contains("description") ||
                        !response.at("description").is_string() || response.at("description").get_ref<const std::string&>().empty()) {
                        if (state != nullptr) {
                            add_finding(*state, Finding{"violation", "sacv.response.description_missing", path,
                                key + " response=" + code});
                        }
                    }
                    if (response.is_object() && response.contains("content") &&
                        (!response.at("content").is_object() || response.at("content").empty()) && state != nullptr) {
                        add_finding(*state, Finding{"violation", "sacv.response.content_invalid", path,
                            key + " response=" + code});
                    }
                }
            }
            if ((!success || !error) && state != nullptr) {
                add_finding(*state, Finding{"violation", "sacv.operation.response_classes_incomplete", path, key});
            }
            operation.digest = engine::tagged_sha256(operation_value.dump());
            operations.emplace(key, std::move(operation));
            if (state != nullptr) {
                ++state->operations_checked;
                add_finding(*state, Finding{"pass", "sacv.operation.checked", path, key});
            }
        }
    }
    return operations;
}

Document validate_document(const fs::path& root, const Entry& entry, RegistryState& state,
                           std::int64_t deadline_unix_ms) {
    const auto path = entry.fields.at("path");
    const auto contents = engine::read_regular_file_no_follow(
        root, path, engine::Limits::max_snapshot_file_bytes, deadline_unix_ms);
    Document result;
    result.file = engine::FileDigest{path, static_cast<std::uint64_t>(contents.size()),
                                     engine::tagged_sha256(contents)};
    if (path.ends_with(".openapi.yaml")) {
        add_finding(state, Finding{"violation", "sacv.document.parser_unavailable", path,
            "YAML remains canonical-capable but the v1 parser compatibility gate is not implemented"});
        return result;
    }
    try {
        const auto document = parse_openapi_json(contents);
        result.title = document.at("info").value("title", "");
        result.version = document.at("info").value("version", "");
        result.operations = extract_operations(document, path, &state, &entry);
        scan_refs_and_examples(document, "", entry, state);
        if (document.at("openapi") != entry.fields.at("openapi")) {
            add_finding(state, Finding{"violation", "sacv.document.openapi_mismatch", path,
                "registry=" + entry.fields.at("openapi")});
        }
        if (result.title != entry.fields.at("title")) {
            add_finding(state, Finding{"violation", "sacv.document.title_mismatch", path,
                "registry title does not equal info.title"});
        }
        if (result.version != entry.fields.at("api_version")) {
            add_finding(state, Finding{"violation", "sacv.document.version_mismatch", path,
                "registry api_version does not equal info.version"});
        }
        if (!document.contains("x-symphony-security-profile") ||
            !document.at("x-symphony-security-profile").is_string() ||
            document.at("x-symphony-security-profile") != entry.fields.at("security_profile")) {
            add_finding(state, Finding{"violation", "sacv.document.security_profile_mismatch", path,
                "top-level x-symphony-security-profile must equal the registry reference"});
        }
        add_finding(state, Finding{"pass", "sacv.document.parsed", path, "bounded OpenAPI JSON parsed"});
    } catch (const engine::Error& error) {
        add_finding(state, Finding{"violation", error.code(), path, error.what()});
    }
    return result;
}

RegistryState analyze_registry(const fs::path& root, std::int64_t deadline_unix_ms) {
    RegistryState state;
    state.contents = engine::read_regular_file_no_follow(
        root, registry_path, engine::Limits::max_snapshot_file_bytes, deadline_unix_ms);
    state.registry_file = engine::FileDigest{registry_path, static_cast<std::uint64_t>(state.contents.size()),
                                             engine::tagged_sha256(state.contents)};
    state.contract_snapshot = engine::snapshot_files(root, contract_paths, deadline_unix_ms);
    const auto skvi = engine::read_regular_file_no_follow(
        root, skvi_index_path, engine::Limits::max_snapshot_file_bytes, deadline_unix_ms);
    state.entries = parse_entries(state.contents, state);
    std::set<std::string> ids;
    std::set<std::string> paths;
    for (const auto& entry : state.entries) {
        std::string identity = "unavailable";
        if (const auto found = entry.fields.find("api_id"); found != entry.fields.end()) {
            identity = found->second;
        }
        bool shape = true;
        for (const auto* field : entry_fields) {
            const auto found = entry.fields.find(field);
            if (found == entry.fields.end() || !printable_bounded(found->second, max_field_bytes)) {
                shape = false;
                add_finding(state, Finding{"violation", "sacv.registry.field_invalid", identity,
                    "line=" + std::to_string(entry.line) + " field=" + field});
            }
        }
        if (!shape) {
            continue;
        }
        const auto& fields = entry.fields;
        const auto& path = fields.at("path");
        if (!safe_token(fields.at("api_id")) || fields.at("api_id") != lower(fields.at("api_id"))) {
            add_finding(state, Finding{"violation", "sacv.registry.api_id_invalid", identity, "stable lowercase token required"});
        }
        if (!ids.insert(fields.at("api_id")).second) {
            add_finding(state, Finding{"violation", "sacv.registry.api_id_duplicate", identity, "api_id is not unique"});
        }
        if (!paths.insert(path).second) {
            add_finding(state, Finding{"violation", "sacv.registry.path_duplicate", path, "path is not unique"});
        }
        if (!owner_path(fields.at("owner")) || !registered_document_path(path) ||
            !path.starts_with(fields.at("owner") + "/")) {
            add_finding(state, Finding{"violation", "sacv.registry.ownership_invalid", path,
                "owner/path placement is unsafe or inconsistent"});
        }
        if (fields.at("openapi") != "3.2.0" ||
            !exact_enum(fields.at("audience"), {"local_internal", "remote_administrative", "partner", "public"}) ||
            !exact_enum(fields.at("publication_state"), {"internal_only", "candidate", "sodv_approved"}) ||
            !exact_enum(fields.at("sdk_state"), {"not_eligible", "candidate", "approved"}) ||
            !exact_enum(fields.at("status"), {"draft", "ratified", "deprecated", "retired"}) ||
            !safe_token(fields.at("api_version")) || !safe_token(fields.at("transport_profile")) ||
            !canonical_profile_path(fields.at("security_profile"))) {
            add_finding(state, Finding{"violation", "sacv.registry.classification_invalid", identity,
                "version, audience, profile, lifecycle, or publication field is invalid"});
        }
        if (!indexed_by_skvi(skvi, path)) {
            add_finding(state, Finding{"violation", "sacv.registry.skvi_unindexed", path,
                "canonical API entry document is not indexed by SKVI"});
        }
        if (!indexed_by_skvi(skvi, fields.at("security_profile"))) {
            add_finding(state, Finding{"violation", "sacv.registry.security_profile_unindexed",
                fields.at("security_profile"), "ratified security profile reference is not indexed by SKVI"});
        } else {
            try {
                static_cast<void>(engine::read_regular_file_no_follow(root, fields.at("security_profile"),
                    engine::Limits::max_snapshot_file_bytes, deadline_unix_ms));
            } catch (const engine::Error& error) {
                add_finding(state, Finding{"violation", "sacv.registry.security_profile_unreadable",
                    fields.at("security_profile"), error.code()});
            }
        }
        try {
            state.documents.emplace(fields.at("api_id"),
                validate_document(root, entry, state, deadline_unix_ms));
        } catch (const engine::Error& error) {
            add_finding(state, Finding{"violation", "sacv.document.unreadable", path, error.code()});
        }
        add_finding(state, Finding{"pass", "sacv.registry.entry_checked", identity, "entry shape and identity checked"});
    }
    return state;
}

engine::Json evidence_json(const RegistryState& state) {
    auto result = engine::Json::array();
    for (const auto& finding : state.findings) {
        result.push_back(engine::Json{{"severity", finding.severity}, {"code", finding.code},
                                      {"path", finding.path}, {"detail", finding.detail}});
    }
    return result;
}

void require_clean(const RegistryState& state) {
    if (finding_count(state, "violation") != 0U) {
        throw engine::Error("sacv.registry_invalid", "canonical SACV registry or registered contract failed checks", 4);
    }
}

engine::Json inspect(const engine::Json& payload) {
    require_exact_fields(payload, {});
    return engine::Json{
        {"descriptor", descriptor()},
        {"canonical_registry", registry_path},
        {"readiness", "read_check_diff_propose_project"},
        {"parser_formats", engine::Json{{"json", "implemented"}, {"yaml", "fail_closed_unavailable"}}},
        {"empty_registry_valid", true},
        {"engine_decides_ownership", false},
        {"canonical_apply_enabled", false},
        {"session_mutation_enabled", false},
        {"maestro_docking_enabled", false},
    };
}

engine::Json check(const engine::Json& payload, std::int64_t deadline_unix_ms) {
    require_exact_fields(payload, {"expected_registry_digest"});
    const auto& expected = payload.at("expected_registry_digest");
    if (!expected.is_null() && (!expected.is_string() || !tagged_digest(expected.get<std::string>()))) {
        throw engine::Error("payload.invalid_expected_digest",
            "expected_registry_digest must be a tagged SHA-256 digest or null", 4);
    }
    auto state = analyze_registry(fs::current_path(), deadline_unix_ms);
    engine::Json matches = nullptr;
    if (expected.is_string()) {
        matches = expected.get<std::string>() == state.registry_file.digest;
        if (!matches.get<bool>()) {
            add_finding(state, Finding{"violation", "sacv.registry.expected_digest_mismatch", registry_path,
                "expected=" + expected.get<std::string>() + " observed=" + state.registry_file.digest});
        }
    }
    const auto violation = finding_count(state, "violation");
    return engine::Json{
        {"protocol", check_protocol},
        {"registry", file_json(state.registry_file)},
        {"contract_snapshot", snapshot_json(state.contract_snapshot)},
        {"expected_registry_matches", matches},
        {"entries_checked", state.entries.size()},
        {"documents_checked", state.documents.size()},
        {"operations_checked", state.operations_checked},
        {"evidence", evidence_json(state)},
        {"summary", engine::Json{{"pass", finding_count(state, "pass")},
                                  {"warning", finding_count(state, "warning")},
                                  {"violation", violation},
                                  {"state", violation == 0U ? "valid" : "invalid"}}},
        {"read_only", true},
        {"canonical_apply_enabled", false},
    };
}

struct DiffDocument final {
    engine::FileDigest file;
    std::map<std::string, Operation> operations;
};

DiffDocument read_diff_document(const std::string& path, const std::string& expected,
                                std::int64_t deadline_unix_ms) {
    if (!engine::is_safe_relative_path(path) || !path.ends_with(".openapi.json")) {
        throw engine::Error("sacv.diff.path_invalid", "diff paths must be safe .openapi.json files", 4);
    }
    const auto contents = engine::read_regular_file_no_follow(
        fs::current_path(), path, engine::Limits::max_snapshot_file_bytes, deadline_unix_ms);
    DiffDocument result;
    result.file = engine::FileDigest{path, static_cast<std::uint64_t>(contents.size()),
                                     engine::tagged_sha256(contents)};
    if (result.file.digest != expected) {
        throw engine::Error("sacv.diff.digest_mismatch", "diff input digest is stale", 4);
    }
    const auto document = parse_openapi_json(contents);
    RegistryState validation;
    if (!document.contains("x-symphony-security-profile") ||
        !document.at("x-symphony-security-profile").is_string() ||
        !canonical_profile_path(document.at("x-symphony-security-profile").get<std::string>())) {
        throw engine::Error("sacv.diff.document_invalid",
            "diff input must declare a canonical x-symphony-security-profile", 4);
    }
    result.operations = extract_operations(document, path, &validation, nullptr);
    Entry diff_entry;
    diff_entry.fields["path"] = path;
    diff_entry.fields["audience"] = "public";
    scan_refs_and_examples(document, "", diff_entry, validation);
    if (finding_count(validation, "violation") != 0U) {
        throw engine::Error("sacv.diff.document_invalid",
            "diff input must satisfy the bounded structural OpenAPI profile", 4);
    }
    return result;
}

engine::Json diff(const engine::Json& payload, std::int64_t deadline_unix_ms) {
    require_exact_fields(payload, {"baseline_path", "baseline_digest", "candidate_path", "candidate_digest"});
    const auto baseline_path = require_string(payload, "baseline_path", engine::Limits::max_path_bytes);
    const auto candidate_path = require_string(payload, "candidate_path", engine::Limits::max_path_bytes);
    const auto baseline_digest = require_string(payload, "baseline_digest", 71U);
    const auto candidate_digest = require_string(payload, "candidate_digest", 71U);
    if (!tagged_digest(baseline_digest) || !tagged_digest(candidate_digest)) {
        throw engine::Error("sacv.diff.digest_invalid", "diff digests must be tagged SHA-256 values", 4);
    }
    const auto baseline = read_diff_document(baseline_path, baseline_digest, deadline_unix_ms);
    const auto candidate = read_diff_document(candidate_path, candidate_digest, deadline_unix_ms);
    auto changes = engine::Json::array();
    std::size_t additive = 0;
    std::size_t breaking = 0;
    std::size_t review = 0;
    auto add_change = [&](const std::string& classification, const std::string& code,
                          const std::string& operation, const std::string& detail) {
        if (changes.size() >= max_changes) {
            throw engine::Error("sacv.diff.change_limit", "SACV diff change limit exceeded", 5);
        }
        changes.push_back(engine::Json{{"classification", classification}, {"code", code},
                                       {"operation", operation}, {"detail", detail}});
        if (classification == "additive") { ++additive; }
        else if (classification == "breaking") { ++breaking; }
        else { ++review; }
    };
    for (const auto& [key, before] : baseline.operations) {
        const auto found = candidate.operations.find(key);
        if (found == candidate.operations.end()) {
            add_change("breaking", "sacv.diff.operation_removed", key, "operation removed");
            continue;
        }
        const auto& after = found->second;
        bool recognized = false;
        if (before.operation_id != after.operation_id) {
            add_change("breaking", "sacv.diff.operation_id_changed", key, "stable operationId changed");
            recognized = true;
        }
        if (!before.request_body_required && after.request_body_required) {
            add_change("breaking", "sacv.diff.request_became_required", key, "optional request body became required");
            recognized = true;
        }
        for (const auto& code : before.response_codes) {
            if (!after.response_codes.contains(code)) {
                add_change("breaking", "sacv.diff.response_removed", key, "response removed: " + code);
                recognized = true;
            }
        }
        if (before.protected_operation && !after.protected_operation) {
            add_change("breaking", "sacv.diff.security_weakened", key, "effective security was removed");
            recognized = true;
        }
        if (before.digest != after.digest && !recognized) {
            add_change("review_required", "sacv.diff.semantic_review", key,
                "operation changed outside deterministic v1 compatibility rules");
        }
    }
    for (const auto& [key, after] : candidate.operations) {
        static_cast<void>(after);
        if (!baseline.operations.contains(key)) {
            add_change("additive", "sacv.diff.operation_added", key, "operation added");
        }
    }
    std::string state = "identical";
    if (breaking != 0U) { state = "breaking"; }
    else if (review != 0U) { state = "review_required"; }
    else if (additive != 0U) { state = "compatible_additive"; }
    engine::Json result{
        {"protocol", diff_protocol},
        {"baseline", file_json(baseline.file)},
        {"candidate", file_json(candidate.file)},
        {"state", state},
        {"changes", std::move(changes)},
        {"summary", engine::Json{{"additive", additive}, {"breaking", breaking}, {"review_required", review}}},
        {"read_only", true},
        {"noncanonical", true},
    };
    result["result_digest"] = engine::tagged_sha256(result.dump());
    return result;
}

Entry entry_from_payload(const engine::Json& value) {
    std::set<std::string> fields;
    for (const auto* field : entry_fields) {
        fields.insert(field);
    }
    require_exact_fields(value, fields);
    Entry entry;
    for (const auto& field : fields) {
        if (!value.at(field).is_string() ||
            !printable_bounded(value.at(field).get_ref<const std::string&>(), max_field_bytes)) {
            throw engine::Error("payload.invalid_entry_field", "SACV registry entry field is invalid: " + field, 4);
        }
        entry.fields[field] = value.at(field).get<std::string>();
    }
    return entry;
}

void validate_repository_envelope(const engine::Json& payload) {
    const auto& repository = payload.at("repository");
    require_exact_fields(repository, {"repository_id", "revision", "worktree_id", "tree_digest"});
    static_cast<void>(require_string(repository, "repository_id", 256U));
    static_cast<void>(require_string(repository, "worktree_id", 128U, true));
    const auto tree_digest = require_string(repository, "tree_digest", 71U);
    if (!tagged_digest(tree_digest)) {
        throw engine::Error("payload.invalid_tree_digest", "tree_digest must be tagged SHA-256", 4);
    }
    const auto& revision = repository.at("revision");
    require_exact_fields(revision, {"scheme", "value"});
    static_cast<void>(require_string(revision, "scheme", 64U, true));
    static_cast<void>(require_string(revision, "value", 256U));
    for (const auto* field : {"session_ref", "context_ref"}) {
        const auto& value = payload.at(field);
        if (!value.is_null() && (!value.is_string() || !safe_token(value.get<std::string>()))) {
            throw engine::Error("payload.invalid_reference", std::string(field) + " must be a token or null", 4);
        }
    }
}

engine::Json propose(const engine::Json& payload, std::int64_t deadline_unix_ms) {
    require_exact_fields(payload, {"repository", "session_ref", "context_ref", "created_at", "expires_at", "operation"});
    validate_repository_envelope(payload);
    const auto created_at = require_string(payload, "created_at", 20U);
    const auto expires_at = require_string(payload, "expires_at", 20U);
    if (!strict_utc(created_at) || !strict_utc(expires_at) || created_at >= expires_at) {
        throw engine::Error("payload.invalid_time", "created_at and expires_at must be ordered strict UTC timestamps", 4);
    }
    const auto& operation = payload.at("operation");
    require_exact_fields(operation, {"type", "expected_registry_digest", "expected_entry_digest", "entry"});
    const auto type = require_string(operation, "type", 32U, true);
    if (type != "register_contract" && type != "replace_contract") {
        throw engine::Error("payload.invalid_operation", "unsupported caller-declared SACV operation", 4);
    }
    const auto expected_registry = require_string(operation, "expected_registry_digest", 71U);
    if (!tagged_digest(expected_registry)) {
        throw engine::Error("payload.invalid_expected_digest", "expected_registry_digest must be tagged SHA-256", 4);
    }
    const auto entry = entry_from_payload(operation.at("entry"));
    require_valid_entry_contract(entry);
    auto state = analyze_registry(fs::current_path(), deadline_unix_ms);
    require_clean(state);
    if (expected_registry != state.registry_file.digest) {
        throw engine::Error("proposal.expected_state_mismatch", "registry expected-state digest is stale", 4);
    }
    const auto current = std::find_if(state.entries.begin(), state.entries.end(), [&](const Entry& item) {
        return item.fields.at("api_id") == entry.fields.at("api_id");
    });
    engine::Json expected_entry = nullptr;
    if (type == "register_contract") {
        if (current != state.entries.end() || !operation.at("expected_entry_digest").is_null()) {
            throw engine::Error("proposal.register_conflict", "register requires a new api_id and null expected entry digest", 4);
        }
    } else {
        if (current == state.entries.end() || !operation.at("expected_entry_digest").is_string() ||
            !tagged_digest(operation.at("expected_entry_digest").get<std::string>())) {
            throw engine::Error("proposal.expected_state_mismatch", "replace requires an existing entry digest", 4);
        }
        const auto observed = entry_json(*current).at("entry_digest").get<std::string>();
        if (operation.at("expected_entry_digest") != observed) {
            throw engine::Error("proposal.expected_state_mismatch", "entry expected-state digest is stale", 4);
        }
        expected_entry = observed;
    }
    RegistryState candidate_state;
    const auto candidate = validate_document(fs::current_path(), entry, candidate_state, deadline_unix_ms);
    const auto skvi = engine::read_regular_file_no_follow(fs::current_path(), skvi_index_path,
        engine::Limits::max_snapshot_file_bytes, deadline_unix_ms);
    if (!indexed_by_skvi(skvi, entry.fields.at("path"))) {
        add_finding(candidate_state, Finding{"violation", "sacv.registry.skvi_unindexed", entry.fields.at("path"),
            "candidate owner contract must already be indexed by SKVI"});
    }
    if (!indexed_by_skvi(skvi, entry.fields.at("security_profile"))) {
        add_finding(candidate_state, Finding{"violation", "sacv.registry.security_profile_unindexed",
            entry.fields.at("security_profile"), "candidate security profile must already be indexed by SKVI"});
    }
    std::string security_profile_contents;
    try {
        security_profile_contents = engine::read_regular_file_no_follow(fs::current_path(),
            entry.fields.at("security_profile"), engine::Limits::max_snapshot_file_bytes, deadline_unix_ms);
    } catch (const engine::Error& error) {
        add_finding(candidate_state, Finding{"violation", "sacv.registry.security_profile_unreadable",
            entry.fields.at("security_profile"), error.code()});
    }
    if (finding_count(candidate_state, "violation") != 0U) {
        throw engine::Error("proposal.candidate_invalid", "candidate owner contract failed SACV checks", 4);
    }
    const auto normalized_entry = entry_json_without_digest(entry);
    const auto desired_change_digest = engine::tagged_sha256(normalized_entry.dump());
    auto read_set = engine::Json::array();
    read_set.push_back(file_json(state.registry_file));
    for (const auto& file : state.contract_snapshot.files) {
        read_set.push_back(file_json(file));
    }
    read_set.push_back(file_json(candidate.file));
    read_set.push_back(file_json(engine::FileDigest{entry.fields.at("security_profile"),
        static_cast<std::uint64_t>(security_profile_contents.size()),
        engine::tagged_sha256(security_profile_contents)}));
    engine::Json proposal{
        {"protocol", proposal_protocol}, {"module_id", module_id}, {"engine_id", engine_id},
        {"engine_version", engine_version}, {"vector_id", vector_id},
        {"contract_versions", engine::Json::array({"knowledge/SPEC.md@v1", "knowledge/sacv/SPEC.md@v1"})},
        {"repository", payload.at("repository")}, {"session_ref", payload.at("session_ref")},
        {"context_ref", payload.at("context_ref")}, {"read_set", std::move(read_set)},
        {"write_set", engine::Json::array({engine::Json{{"target_path", registry_path},
            {"expected_prior_digest", state.registry_file.digest}, {"desired_change_digest", desired_change_digest}}})},
        {"operations", engine::Json::array({engine::Json{
            {"operation_id", "sacv-op:" + engine::sha256_hex(operation.dump())}, {"type", type},
            {"target_path", registry_path}, {"expected_state_digest", expected_entry},
            {"desired_change_digest", desired_change_digest}, {"data", normalized_entry}}})},
        {"validation", engine::Json::array({
            engine::Json{{"code", "sacv.registry.valid"}, {"outcome", "pass"}, {"detail", "current registry passed checks"}},
            engine::Json{{"code", "sacv.contract.valid"}, {"outcome", "pass"}, {"detail", "candidate owner contract passed JSON profile checks"}},
            engine::Json{{"code", "sacv.operation.caller_declared"}, {"outcome", "pass"}, {"detail", "operation and semantic owner were supplied by the caller"}}})},
        {"authority", engine::Json{{"caller_declared_operation", true},
            {"engine_decided_domain_truth", false}, {"ratified", false}}},
        {"created_at", created_at}, {"expires_at", expires_at}, {"canonical_apply_enabled", false},
    };
    proposal["proposal_id"] = "sacv-proposal:" + engine::sha256_hex(proposal.dump());
    proposal["proposal_digest"] = engine::tagged_sha256(proposal.dump());
    return proposal;
}

engine::Json project(const engine::Json& payload, std::int64_t deadline_unix_ms) {
    require_exact_fields(payload, {"format"});
    if (require_string(payload, "format", 16U, true) != "json") {
        throw engine::Error("payload.unsupported_format", "only json projection is implemented", 4);
    }
    const auto state = analyze_registry(fs::current_path(), deadline_unix_ms);
    require_clean(state);
    auto entries = engine::Json::array();
    for (const auto& entry : state.entries) {
        auto projected = entry_json(entry);
        const auto found = state.documents.find(entry.fields.at("api_id"));
        projected["contract_digest"] = found == state.documents.end() ? nullptr : engine::Json(found->second.file.digest);
        projected["operation_count"] = found == state.documents.end() ? 0U : found->second.operations.size();
        entries.push_back(std::move(projected));
    }
    engine::Json result{
        {"protocol", projection_protocol}, {"projection_kind", "registry-conformance-inventory"},
        {"format", "json"}, {"module_id", module_id}, {"engine_id", engine_id},
        {"engine_version", engine_version}, {"vector_id", vector_id},
        {"canonical_registry", file_json(state.registry_file)},
        {"contract_snapshot", snapshot_json(state.contract_snapshot)},
        {"entry_count", state.entries.size()}, {"entries", std::move(entries)},
        {"noncanonical", true}, {"rebuildable", true},
    };
    result["projection_digest"] = engine::tagged_sha256(result.dump());
    return result;
}

}

engine::Json descriptor() {
    return engine::Json{
        {"protocol", engine::descriptor_protocol_v1}, {"module_id", module_id},
        {"engine_id", engine_id}, {"vector_id", vector_id}, {"engine_version", engine_version},
        {"process_protocols", engine::Json::array({engine::process_protocol_v1})},
        {"contract_versions", engine::Json::array({"knowledge/SPEC.md@v1", "knowledge/sacv/SPEC.md@v1"})},
        {"operations", engine::Json::array({
            engine::Json{{"name", "inspect"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "check"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "diff"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "propose"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "project"}, {"availability", "implemented"}, {"mutates_canonical", false}},
            engine::Json{{"name", "apply"}, {"availability", "disabled"}, {"mutates_canonical", true}}})},
        {"limits", engine::Json{{"request_bytes", engine::Limits::max_request_bytes},
            {"response_bytes", engine::Limits::max_response_bytes}, {"json_depth", engine::Limits::max_json_depth},
            {"json_values", engine::Limits::max_json_values}, {"path_bytes", engine::Limits::max_path_bytes},
            {"snapshot_files", engine::Limits::max_snapshot_files},
            {"snapshot_file_bytes", engine::Limits::max_snapshot_file_bytes},
            {"deadline_ahead_ms", engine::Limits::max_deadline_ahead_ms}}},
        {"supported_scopes", engine::Json::array({"user"})}, {"language", "C++26"},
        {"thermal_path", "freezing"}, {"openapi_target", "3.2.0"},
        {"parser_formats", engine::Json{{"json", "implemented"}, {"yaml", "fail_closed_unavailable"}}},
        {"install_state", "installed_undocked"}, {"default_receptor", nullptr},
        {"canonical_apply_enabled", false}, {"session_mutation_enabled", false},
        {"network_listener", false},
    };
}

engine::Json handle_request(const engine::Request& request) {
    if (request.operation == "inspect") { return inspect(request.payload); }
    if (request.operation == "check") { return check(request.payload, request.deadline_unix_ms); }
    if (request.operation == "diff") { return diff(request.payload, request.deadline_unix_ms); }
    if (request.operation == "propose") { return propose(request.payload, request.deadline_unix_ms); }
    if (request.operation == "project") { return project(request.payload, request.deadline_unix_ms); }
    throw engine::Error("operation.unsupported", "operation is reserved or unsupported", 4);
}

}
