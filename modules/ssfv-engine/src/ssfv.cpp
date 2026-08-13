#include "ssfv.hpp"

#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/limits.hpp"
#include "symphony/knowledge/engine/operation.hpp"
#include "symphony/knowledge/engine/path.hpp"
#include "symphony/knowledge/engine/temporal.hpp"

#include <algorithm>
#include <array>
#include <cctype>
#include <cerrno>
#include <fcntl.h>
#include <filesystem>
#include <functional>
#include <map>
#include <optional>
#include <set>
#include <sstream>
#include <stdexcept>
#include <string>
#include <string_view>
#include <sys/stat.h>
#include <unistd.h>
#include <utility>
#include <vector>

namespace symphony::knowledge::ssfv {
namespace engine = symphony::knowledge::engine;
namespace fs = std::filesystem;

namespace {

constexpr std::size_t max_namespaces = 4096U;
constexpr std::size_t max_feature_files = 1024U;
constexpr std::size_t max_feature_records = 8192U;
constexpr std::size_t max_graph_edges = 32768U;
constexpr std::size_t max_findings = 4096U;
constexpr std::size_t max_total_evidence_bytes = 64U * 1024U * 1024U;
constexpr std::size_t max_description_bytes = 65536U;
constexpr const char* namespaces_path = "knowledge/ssfv/NAMESPACES.md";
constexpr const char* registry_path = "knowledge/ssfv/REGISTRY.md";
constexpr const char* skvi_index_path = "knowledge/skvi/INDEX.md";
constexpr const char* feature_file_protocol = "symphony.ssfv.feature-file.v1";
constexpr const char* semantic_snapshot_protocol = "symphony.ssfv.semantic-snapshot.v1";
constexpr const char* check_protocol = "symphony.ssfv.check-result.v2";
constexpr const char* diff_protocol = "symphony.ssfv.diff-result.v2";
constexpr const char* graph_protocol = "symphony.ssfv.graph-projection.v1";
constexpr const char* administration_coverage_protocol =
    "symphony.knowledge.administration-coverage-result.v1";
constexpr const char* administration_coverage_input_protocol =
    "symphony.knowledge.administration-coverage-input.v1";
constexpr const char* proposal_protocol = "symphony.knowledge.proposal.v1";
constexpr std::string_view begin_marker = "<!-- symphony:ssfv:feature-file:v1:begin -->";
constexpr std::string_view end_marker = "<!-- symphony:ssfv:feature-file:v1:end -->";

const std::vector<std::string> contract_paths = {
    "knowledge/SPEC.md",
    "knowledge/ssfv/INTENT.md",
    "knowledge/ssfv/MANIFEST.md",
    "knowledge/ssfv/SKILL.md",
    "knowledge/ssfv/SPEC.md",
    "knowledge/ssfv/COVERAGE.md",
    namespaces_path,
    registry_path,
    "knowledge/ssfv/FEATURE-FILE-FORMAT.md",
    "knowledge/ssfv/schemas/v1/MANIFEST.md",
    "knowledge/ssfv/schemas/v1/check-result.schema.json",
    "knowledge/ssfv/schemas/v1/diff-input.schema.json",
    "knowledge/ssfv/schemas/v1/diff-result.schema.json",
    "knowledge/ssfv/schemas/v1/feature-file.schema.json",
    "knowledge/ssfv/schemas/v1/feature-record.schema.json",
    "knowledge/ssfv/schemas/v1/graph-input.schema.json",
    "knowledge/ssfv/schemas/v1/graph-projection.schema.json",
    "knowledge/ssfv/schemas/v1/namespace-entry.schema.json",
    "knowledge/ssfv/schemas/v1/proposal-input.schema.json",
    "knowledge/ssfv/schemas/v1/registry-entry.schema.json",
    "knowledge/ssfv/schemas/v1/semantic-snapshot.schema.json",
    "knowledge/ssfv/schemas/v2/MANIFEST.md",
    "knowledge/ssfv/schemas/v2/check-input.schema.json",
    "knowledge/ssfv/schemas/v2/check-result.schema.json",
    "knowledge/ssfv/schemas/v2/diff-input.schema.json",
    "knowledge/ssfv/schemas/v2/diff-result.schema.json",
    "knowledge/ssfv/schemas/v2/feature-record.schema.json",
    "knowledge/ssfv/schemas/v2/proposal-input.schema.json",
    "knowledge/ssfv/schemas/v2/registry-entry.schema.json",
    skvi_index_path,
};

constexpr std::array<const char*, 7> namespace_fields = {
    "namespace", "id_prefix", "owner_contract", "scope", "status", "evidence", "notes",
};

constexpr std::array<const char*, 8> registry_fields = {
    "feature_id", "feature_file", "owner_contract", "source_scope",
    "status", "parent_feature_id", "record_digest", "notes",
};

const std::set<std::string> feature_record_fields = {
    "feature_id", "record_version", "title", "kind", "status", "parent_feature_id",
    "owner_contract", "source_scope", "implementation_paths", "implementation_languages",
    "who", "what", "how", "when", "where", "why", "relationships", "distinctions",
    "cross_vector_references", "evidence", "non_claims",
};

struct DomainError final : std::runtime_error {
    DomainError(std::string code_value, std::string path_value, std::string feature_value,
                std::string detail_value)
        : std::runtime_error(std::move(detail_value)), code(std::move(code_value)),
          path(std::move(path_value)), feature_id(std::move(feature_value)) {}
    std::string code;
    std::string path;
    std::string feature_id;
};

struct Finding final {
    std::string severity;
    std::string code;
    std::optional<std::string> path;
    std::optional<std::string> feature_id;
    std::string detail;
    std::string basis = "deterministic";
};

struct NamespaceEntry final {
    std::map<std::string, std::string> fields;
};

struct RegistryEntry final {
    std::map<std::string, std::string> fields;
};

struct FeatureRecord final {
    engine::Json normalized;
    std::string feature_id;
    std::string title;
    std::string kind;
    std::string status;
    std::optional<std::string> parent_feature_id;
    std::string owner_contract;
    std::string source_scope;
    std::vector<std::string> implementation_paths;
    std::vector<std::string> cross_vector_paths;
    std::string record_digest;
};

struct FeatureFile final {
    std::string path;
    engine::FileDigest file;
    std::string owner_contract;
    std::string source_scope;
    std::string prefix;
    std::string suffix;
    std::map<std::string, FeatureRecord> records;
};

struct State final {
    engine::Snapshot contract_snapshot;
    engine::FileDigest namespace_file;
    engine::FileDigest registry_file;
    engine::FileDigest skvi_file;
    std::vector<NamespaceEntry> namespaces;
    std::vector<RegistryEntry> registry_entries;
    std::map<std::string, RegistryEntry> registry_by_id;
    std::map<std::string, FeatureFile> feature_files;
    std::map<std::string, FeatureRecord> records;
    std::map<std::string, engine::FileDigest> evidence_files;
    std::set<std::string> counted_evidence_paths;
    std::vector<Finding> findings;
    std::size_t passes = 0U;
    std::size_t total_evidence_bytes = 0U;
    bool registry_empty = false;
    bool structural_valid = true;
    std::string coverage_state = "empty";
    engine::Json semantic_snapshot;
};

class FileDescriptor final {
public:
    explicit FileDescriptor(int value = -1) : value_(value) {}
    ~FileDescriptor() {
        if (value_ >= 0) {
            static_cast<void>(::close(value_));
        }
    }
    FileDescriptor(const FileDescriptor&) = delete;
    FileDescriptor& operator=(const FileDescriptor&) = delete;
    FileDescriptor(FileDescriptor&& other) noexcept : value_(std::exchange(other.value_, -1)) {}
    FileDescriptor& operator=(FileDescriptor&& other) noexcept {
        if (this != &other) {
            if (value_ >= 0) {
                static_cast<void>(::close(value_));
            }
            value_ = std::exchange(other.value_, -1);
        }
        return *this;
    }
    [[nodiscard]] int get() const noexcept { return value_; }
private:
    int value_;
};

std::string trim(std::string_view value) {
    std::size_t begin = 0U;
    while (begin < value.size() && std::isspace(static_cast<unsigned char>(value[begin])) != 0) {
        ++begin;
    }
    std::size_t end = value.size();
    while (end > begin && std::isspace(static_cast<unsigned char>(value[end - 1U])) != 0) {
        --end;
    }
    return std::string(value.substr(begin, end - begin));
}

std::string clean_value(std::string value) {
    value = trim(value);
    if (value.size() >= 2U && value.front() == '`' && value.back() == '`') {
        return value.substr(1U, value.size() - 2U);
    }
    return value;
}

bool printable_bounded(std::string_view value, std::size_t max_bytes, bool allow_empty = false) {
    if ((!allow_empty && value.empty()) || value.size() > max_bytes) {
        return false;
    }
    if (!std::all_of(value.begin(), value.end(), [](const unsigned char character) {
        return character == '\n' || character == '\t' ||
               (character >= 0x20U && character != 0x7fU);
    })) {
        return false;
    }
    try {
        static_cast<void>(engine::Json(std::string(value)).dump(
            -1, ' ', false, nlohmann::json::error_handler_t::strict));
        return true;
    } catch (const nlohmann::json::exception&) {
        return false;
    }
}

bool tagged_digest(std::string_view value) {
    return value.size() == 71U && value.starts_with("sha256:") &&
           std::all_of(value.begin() + 7, value.end(), [](const unsigned char character) {
               return (character >= '0' && character <= '9') ||
                      (character >= 'a' && character <= 'f');
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

bool safe_language(std::string_view value) {
    if (value.empty() || value.size() > 64U) {
        return false;
    }
    return std::all_of(value.begin(), value.end(), [](const unsigned char character) {
        const bool alphanumeric =
            (character >= 'a' && character <= 'z') ||
            (character >= 'A' && character <= 'Z') ||
            (character >= '0' && character <= '9');
        return alphanumeric || character == '+' || character == '.' ||
               character == '#' || character == '_' || character == '-';
    });
}

bool strict_utc(std::string_view value) {
    return engine::is_utc_seconds(value);
}

bool safe_feature_id(std::string_view value) {
    if (!value.starts_with("ssfv:") || value.size() > 256U) {
        return false;
    }
    const auto namespace_end = value.find(':', 5U);
    if (namespace_end == std::string_view::npos || namespace_end == 5U ||
        namespace_end + 1U >= value.size()) {
        return false;
    }
    const auto valid_namespace = [](std::string_view token) {
        if (token.empty() || token.size() > 63U || token.front() < 'a' || token.front() > 'z') {
            return false;
        }
        return std::all_of(token.begin() + 1, token.end(), [](const unsigned char character) {
            return (character >= 'a' && character <= 'z') ||
                   (character >= '0' && character <= '9') || character == '-';
        });
    };
    if (!valid_namespace(value.substr(5U, namespace_end - 5U))) {
        return false;
    }
    const auto key = value.substr(namespace_end + 1U);
    if (key.empty() || key.front() < 'a' || key.front() > 'z') {
        return false;
    }
    bool separator = false;
    for (const unsigned char character : key) {
        const bool alphanumeric =
            (character >= 'a' && character <= 'z') ||
            (character >= '0' && character <= '9');
        if (!alphanumeric && character != '.' && character != '-') {
            return false;
        }
        if (character == '.' || character == '-') {
            if (separator) {
                return false;
            }
            separator = true;
        } else {
            separator = false;
        }
    }
    return !separator;
}

bool safe_prefixed_id(std::string_view value, std::string_view prefix) {
    if (!value.starts_with(prefix) || value.size() > 256U) {
        return false;
    }
    const auto namespace_end = value.find(':', prefix.size());
    if (namespace_end == std::string_view::npos || namespace_end == prefix.size() ||
        namespace_end + 1U >= value.size()) {
        return false;
    }
    const auto name_space = value.substr(prefix.size(), namespace_end - prefix.size());
    if (name_space.size() > 63U || name_space.front() < 'a' || name_space.front() > 'z' ||
        !std::all_of(name_space.begin() + 1, name_space.end(), [](const unsigned char character) {
            return (character >= 'a' && character <= 'z') ||
                   (character >= '0' && character <= '9') || character == '-';
        })) {
        return false;
    }
    const auto key = value.substr(namespace_end + 1U);
    if (key.front() < 'a' || key.front() > 'z') {
        return false;
    }
    bool separator = false;
    for (const unsigned char character : key) {
        const bool alphanumeric = (character >= 'a' && character <= 'z') ||
                                  (character >= '0' && character <= '9');
        if (!alphanumeric && character != '.' && character != '-') {
            return false;
        }
        if (character == '.' || character == '-') {
            if (separator) {
                return false;
            }
            separator = true;
        } else {
            separator = false;
        }
    }
    return !separator;
}

std::string feature_namespace(const std::string& feature_id) {
    const auto end = feature_id.find(':', 5U);
    return feature_id.substr(5U, end - 5U);
}

bool safe_repository_path(const std::string& value) {
    return engine::is_safe_relative_path(value) &&
           value.find_first_of("*?[]{}") == std::string::npos;
}

bool safe_scope(const std::string& value) {
    return value == "." || safe_repository_path(value);
}

std::string expected_feature_file(const std::string& scope) {
    return scope == "." ? "FEATURES.md" : scope + "/FEATURES.md";
}

bool contains_exact_line(const std::string& contents, const std::string& line) {
    std::size_t position = 0U;
    while ((position = contents.find(line, position)) != std::string::npos) {
        const bool line_start = position == 0U || contents[position - 1U] == '\n';
        const auto after = position + line.size();
        const bool line_end = after == contents.size() || contents[after] == '\n';
        if (line_start && line_end) {
            return true;
        }
        position = after;
    }
    return false;
}

void require_exact_fields(const engine::Json& object, const std::set<std::string>& fields,
                          const std::string& code = "payload.field_set") {
    if (!object.is_object() || object.size() != fields.size()) {
        throw engine::Error(code, "object is incomplete or contains unknown fields", 4);
    }
    for (const auto& [key, value] : object.items()) {
        static_cast<void>(value);
        if (!fields.contains(key)) {
            throw engine::Error(code, "object contains an unknown field", 4);
        }
    }
}

template <std::size_t Size>
void require_exact_fields(const engine::Json& object, const std::array<const char*, Size>& fields,
                          const std::string& code) {
    std::set<std::string> expected;
    for (const auto* field : fields) {
        expected.emplace(field);
    }
    require_exact_fields(object, expected, code);
}

std::string require_string(const engine::Json& object, const char* field,
                           std::size_t max_bytes, bool token = false) {
    if (!object.contains(field) || !object.at(field).is_string()) {
        throw engine::Error("payload.invalid_field", std::string(field) + " must be a string", 4);
    }
    const auto value = object.at(field).get<std::string>();
    if ((token && !safe_token(value, max_bytes)) ||
        (!token && !printable_bounded(value, max_bytes))) {
        throw engine::Error("payload.invalid_field", std::string(field) + " has invalid syntax", 4);
    }
    return value;
}

void add_finding(State& state, Finding finding) {
    if (state.findings.size() >= max_findings) {
        throw engine::Error("ssfv.findings.limit", "SSFV finding limit exceeded", 5);
    }
    if (finding.severity == "pass") {
        ++state.passes;
    } else if (finding.severity == "violation") {
        state.structural_valid = false;
    }
    state.findings.push_back(std::move(finding));
}

engine::Json file_json(const engine::FileDigest& file) {
    return engine::Json{{"path", file.path}, {"size", file.size}, {"digest", file.digest}};
}

engine::Json snapshot_json(const engine::Snapshot& snapshot) {
    engine::Json files = engine::Json::array();
    for (const auto& file : snapshot.files) {
        files.push_back(file_json(file));
    }
    return engine::Json{{"digest", snapshot.digest}, {"files", std::move(files)}};
}

std::string section(const std::string& contents, const std::string& heading) {
    std::size_t heading_position = std::string::npos;
    if (contents.starts_with(heading + "\n")) {
        heading_position = 0U;
    } else {
        const auto positioned = contents.find("\n" + heading + "\n");
        if (positioned != std::string::npos) {
            heading_position = positioned + 1U;
        }
    }
    if (heading_position == std::string::npos) {
        throw DomainError("ssfv.markdown.section_missing", "", "", "required Markdown section is missing");
    }
    const auto begin = contents.find('\n', heading_position + heading.size());
    if (begin == std::string::npos) {
        return "";
    }
    const auto end = contents.find("\n## ", begin + 1U);
    return contents.substr(begin + 1U, end == std::string::npos ? std::string::npos : end - begin - 1U);
}

template <std::size_t Size>
std::vector<std::map<std::string, std::string>> parse_ordered_blocks(
    const std::string& contents, const std::array<const char*, Size>& fields,
    const std::string& first_field, std::size_t maximum) {
    std::vector<std::map<std::string, std::string>> result;
    std::istringstream input(contents);
    std::string line;
    std::optional<std::map<std::string, std::string>> current;
    std::size_t next = 0U;
    while (std::getline(input, line)) {
        const auto normalized = trim(line);
        if (normalized.empty()) {
            continue;
        }
        if (!normalized.starts_with("- ")) {
            throw DomainError("ssfv.markdown.unexpected_content", "", "",
                              "canonical entry section contains unexpected content");
        }
        const auto colon = normalized.find(':', 2U);
        if (colon == std::string::npos) {
            throw DomainError("ssfv.markdown.field_syntax", "", "",
                              "canonical entry field has invalid syntax");
        }
        const auto key = normalized.substr(2U, colon - 2U);
        const auto value = clean_value(normalized.substr(colon + 1U));
        if (key == first_field) {
            if (current) {
                if (next != Size) {
                    throw DomainError("ssfv.markdown.block_incomplete", "", "", "ordered entry block is incomplete");
                }
                result.push_back(std::move(*current));
            }
            current = std::map<std::string, std::string>{};
            next = 0U;
        }
        if (!current) {
            continue;
        }
        if (next >= Size || key != fields[next] ||
            !printable_bounded(value, max_description_bytes)) {
            throw DomainError("ssfv.markdown.field_order", "", "", "ordered entry field is missing, empty, or reordered");
        }
        current->emplace(key, value);
        ++next;
    }
    if (current) {
        if (next != Size) {
            throw DomainError("ssfv.markdown.block_incomplete", "", "", "ordered entry block is incomplete");
        }
        result.push_back(std::move(*current));
    }
    if (result.size() > maximum) {
        throw DomainError("ssfv.markdown.count_limit", "", "", "entry count limit exceeded");
    }
    return result;
}

engine::FileDigest read_state_file(State& state, const std::string& path,
                                   std::int64_t deadline_unix_ms) {
    if (const auto found = state.evidence_files.find(path); found != state.evidence_files.end()) {
        return found->second;
    }
    const auto contents = engine::read_regular_file_no_follow(
        fs::current_path(), path, engine::Limits::max_snapshot_file_bytes, deadline_unix_ms);
    if (state.counted_evidence_paths.insert(path).second) {
        if (state.total_evidence_bytes + contents.size() > max_total_evidence_bytes) {
            throw DomainError("ssfv.resource.total_bytes", path, "",
                              "total SSFV evidence byte limit exceeded");
        }
        state.total_evidence_bytes += contents.size();
    }
    engine::FileDigest file{path, static_cast<std::uint64_t>(contents.size()),
                            engine::tagged_sha256(contents)};
    state.evidence_files.emplace(path, file);
    return file;
}

FileDescriptor open_directory_root(const fs::path& root) {
    const int slash = ::open("/", O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
    if (slash < 0) {
        throw DomainError("ssfv.scope.root_unreadable", "", "", "repository root is unreadable");
    }
    FileDescriptor current(slash);
    for (const auto& value : root.relative_path()) {
        const auto component = value.string();
        if (component.empty() || component == ".") {
            continue;
        }
        if (component == "..") {
            throw DomainError("ssfv.scope.root_unsafe", "", "", "repository root contains traversal");
        }
        const int next = ::openat(current.get(), component.c_str(),
                                  O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
        if (next < 0) {
            throw DomainError("ssfv.scope.root_unsafe", "", "", "repository root contains an unsafe component");
        }
        current = FileDescriptor(next);
    }
    return current;
}

void validate_scope_directory(const std::string& scope) {
    if (!safe_scope(scope)) {
        throw DomainError("ssfv.scope.unsafe", scope, "", "source scope is unsafe");
    }
    auto current = open_directory_root(fs::current_path());
    if (scope == ".") {
        return;
    }
    std::size_t begin = 0U;
    while (begin < scope.size()) {
        const auto end = scope.find('/', begin);
        const auto component = scope.substr(
            begin, end == std::string::npos ? std::string::npos : end - begin);
        const int next = ::openat(current.get(), component.c_str(),
                                  O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
        if (next < 0) {
            throw DomainError("ssfv.scope.unreadable", scope, "", "source scope is absent, unsafe, or not a directory");
        }
        current = FileDescriptor(next);
        if (end == std::string::npos) {
            break;
        }
        begin = end + 1U;
    }
}

void validate_string_array(const engine::Json& value, std::size_t minimum, std::size_t maximum,
                           std::size_t max_string, const std::string& field) {
    if (!value.is_array() || value.size() < minimum || value.size() > maximum) {
        throw DomainError("ssfv.record.array_invalid", "", "", field + " has invalid cardinality");
    }
    for (const auto& item : value) {
        if (!item.is_string() ||
            !printable_bounded(item.get_ref<const std::string&>(), max_string)) {
            throw DomainError("ssfv.record.array_invalid", "", "", field + " contains an invalid string");
        }
    }
}

void sort_unique_json(engine::Json& value, const std::string& field,
                      const std::function<std::string(const engine::Json&)>& key) {
    std::sort(value.begin(), value.end(), [&](const engine::Json& left, const engine::Json& right) {
        return key(left) < key(right);
    });
    std::set<std::string> seen;
    for (const auto& item : value) {
        if (!seen.insert(key(item)).second) {
            throw DomainError("ssfv.record.duplicate_value", "", "", field + " contains a duplicate");
        }
    }
}

FeatureRecord normalize_record(const engine::Json& input) {
    require_exact_fields(input, feature_record_fields, "ssfv.record.field_set");
    if (!input.at("record_version").is_number_integer() ||
        input.at("record_version").get<int>() != 2) {
        throw DomainError("ssfv.record.version", "", "", "feature record version must be 2");
    }
    FeatureRecord record;
    record.normalized = input;
    record.feature_id = require_string(input, "feature_id", 256U);
    if (!safe_feature_id(record.feature_id)) {
        throw DomainError("ssfv.record.feature_id", "", record.feature_id, "feature ID is invalid");
    }
    record.title = require_string(input, "title", 256U);
    record.kind = require_string(input, "kind", 32U, true);
    record.status = require_string(input, "status", 32U, true);
    if (!std::set<std::string>{"capability", "feature", "subfeature", "microfeature"}.contains(record.kind) ||
        !std::set<std::string>{"experimental", "implemented", "deprecated", "retired"}.contains(record.status)) {
        throw DomainError("ssfv.record.enum", "", record.feature_id, "feature kind or status is invalid");
    }
    if (input.at("parent_feature_id").is_null()) {
        if (record.kind != "capability") {
            throw DomainError("ssfv.record.parent_required", "", record.feature_id,
                              "non-capability record requires a parent");
        }
    } else if (input.at("parent_feature_id").is_string()) {
        record.parent_feature_id = input.at("parent_feature_id").get<std::string>();
        if (record.kind == "capability" || !safe_feature_id(*record.parent_feature_id) ||
            *record.parent_feature_id == record.feature_id) {
            throw DomainError("ssfv.record.parent_invalid", "", record.feature_id,
                              "feature parent is invalid");
        }
    } else {
        throw DomainError("ssfv.record.parent_invalid", "", record.feature_id,
                          "feature parent has invalid type");
    }
    record.owner_contract = require_string(input, "owner_contract", engine::Limits::max_path_bytes);
    record.source_scope = require_string(input, "source_scope", engine::Limits::max_path_bytes);
    if (!safe_repository_path(record.owner_contract) || !safe_scope(record.source_scope)) {
        throw DomainError("ssfv.record.owner_path", record.owner_contract, record.feature_id,
                          "owner contract or source scope is unsafe");
    }
    for (const auto* field : {"who", "what", "how", "when", "where", "why"}) {
        static_cast<void>(require_string(input, field, max_description_bytes));
    }

    validate_string_array(input.at("implementation_paths"), 1U, 256U,
                          engine::Limits::max_path_bytes, "implementation_paths");
    for (const auto& path_value : input.at("implementation_paths")) {
        const auto path = path_value.get<std::string>();
        if (!safe_repository_path(path)) {
            throw DomainError("ssfv.record.implementation_path", path, record.feature_id,
                              "implementation path is unsafe");
        }
        record.implementation_paths.push_back(path);
    }
    std::sort(record.implementation_paths.begin(), record.implementation_paths.end());
    if (std::adjacent_find(record.implementation_paths.begin(), record.implementation_paths.end()) !=
        record.implementation_paths.end()) {
        throw DomainError("ssfv.record.duplicate_value", "", record.feature_id,
                          "implementation_paths contains a duplicate");
    }
    record.normalized["implementation_paths"] = record.implementation_paths;

    auto& languages = record.normalized["implementation_languages"];
    if (!languages.is_array() || languages.empty() || languages.size() > 32U) {
        throw DomainError("ssfv.record.languages", "", record.feature_id,
                          "implementation_languages has invalid cardinality");
    }
    for (const auto& language : languages) {
        require_exact_fields(language, std::set<std::string>{"language", "role"},
                             "ssfv.record.language_fields");
        static_cast<void>(require_string(language, "language", 64U));
        static_cast<void>(require_string(language, "role", max_description_bytes));
    }
    sort_unique_json(languages, "implementation_languages", [](const engine::Json& item) {
        return item.at("language").get<std::string>() + "\n" + item.at("role").get<std::string>();
    });

    auto& relationships = record.normalized["relationships"];
    if (!relationships.is_array() || relationships.size() > 256U) {
        throw DomainError("ssfv.record.relationships", "", record.feature_id,
                          "relationships has invalid cardinality");
    }
    const std::set<std::string> relationship_types = {
        "depends_on", "enables", "composes_with", "extends",
        "alternative_to", "supersedes", "distinguished_from",
    };
    for (const auto& relationship : relationships) {
        require_exact_fields(relationship,
            std::set<std::string>{"type", "target_feature_id", "rationale"},
            "ssfv.record.relationship_fields");
        const auto type = require_string(relationship, "type", 64U, true);
        const auto target = require_string(relationship, "target_feature_id", 256U);
        static_cast<void>(require_string(relationship, "rationale", max_description_bytes));
        if (!relationship_types.contains(type) || !safe_feature_id(target) || target == record.feature_id) {
            throw DomainError("ssfv.record.relationship_invalid", "", record.feature_id,
                              "relationship type or target is invalid");
        }
    }
    sort_unique_json(relationships, "relationships", [](const engine::Json& item) {
        return item.at("type").get<std::string>() + "\n" +
               item.at("target_feature_id").get<std::string>();
    });

    auto& distinctions = record.normalized["distinctions"];
    if (!distinctions.is_array() || distinctions.size() > 128U) {
        throw DomainError("ssfv.record.distinctions", "", record.feature_id,
                          "distinctions has invalid cardinality");
    }
    for (const auto& distinction : distinctions) {
        require_exact_fields(distinction,
            std::set<std::string>{"target_feature_id", "distinction"},
            "ssfv.record.distinction_fields");
        const auto target = require_string(distinction, "target_feature_id", 256U);
        static_cast<void>(require_string(distinction, "distinction", max_description_bytes));
        if (!safe_feature_id(target) || target == record.feature_id) {
            throw DomainError("ssfv.record.distinction_invalid", "", record.feature_id,
                              "distinction target is invalid");
        }
    }
    sort_unique_json(distinctions, "distinctions", [](const engine::Json& item) {
        return item.at("target_feature_id").get<std::string>();
    });

    auto& cross = record.normalized["cross_vector_references"];
    if (!cross.is_array() || cross.size() > 128U) {
        throw DomainError("ssfv.record.cross_vector", "", record.feature_id,
                          "cross_vector_references has invalid cardinality");
    }
    const std::set<std::string> vectors = {
        "skvi", "sclv", "sacv", "sodv", "stav", "ssiag", "maestro",
    };
    for (const auto& reference : cross) {
        require_exact_fields(reference,
            std::set<std::string>{"vector", "applicability", "reference", "reason"},
            "ssfv.record.cross_vector_fields");
        const auto vector = require_string(reference, "vector", 32U, true);
        const auto applicability = require_string(reference, "applicability", 32U, true);
        static_cast<void>(require_string(reference, "reason", max_description_bytes));
        if (!vectors.contains(vector) ||
            (applicability != "applicable" && applicability != "not_applicable")) {
            throw DomainError("ssfv.record.cross_vector_invalid", "", record.feature_id,
                              "cross-vector reference has an invalid vector or applicability");
        }
        if (applicability == "applicable") {
            if (!reference.at("reference").is_string()) {
                throw DomainError("ssfv.record.cross_vector_invalid", "", record.feature_id,
                                  "applicable cross-vector reference requires a path");
            }
            const auto path = reference.at("reference").get<std::string>();
            if (!safe_repository_path(path)) {
                throw DomainError("ssfv.record.cross_vector_invalid", path, record.feature_id,
                                  "cross-vector reference path is unsafe");
            }
            record.cross_vector_paths.push_back(path);
        } else if (!reference.at("reference").is_null()) {
            throw DomainError("ssfv.record.cross_vector_invalid", "", record.feature_id,
                              "not-applicable cross-vector reference must be null");
        }
    }
    sort_unique_json(cross, "cross_vector_references", [](const engine::Json& item) {
        return item.at("vector").get<std::string>();
    });
    std::sort(record.cross_vector_paths.begin(), record.cross_vector_paths.end());

    validate_string_array(input.at("evidence"), 1U, 256U,
                          engine::Limits::max_path_bytes, "evidence");
    validate_string_array(input.at("non_claims"), 1U, 128U,
                          engine::Limits::max_path_bytes, "non_claims");
    record.record_digest = engine::tagged_sha256(record.normalized.dump());
    return record;
}

FeatureFile parse_feature_file(const std::string& path, const std::string& contents,
                               const engine::FileDigest& file) {
    if (!printable_bounded(contents, engine::Limits::max_snapshot_file_bytes) ||
        contents.starts_with("\xef\xbb\xbf")) {
        throw DomainError("ssfv.feature_file.encoding", path, "",
                          "feature file must be BOM-free UTF-8 text");
    }
    const auto begin = contents.find(begin_marker);
    if (begin == std::string::npos || contents.find(begin_marker, begin + begin_marker.size()) !=
        std::string::npos) {
        throw DomainError("ssfv.feature_file.marker", path, "",
                          "feature file must contain exactly one opening marker");
    }
    const auto end = contents.find(end_marker, begin + begin_marker.size());
    if (end == std::string::npos || contents.find(end_marker, end + end_marker.size()) !=
        std::string::npos) {
        throw DomainError("ssfv.feature_file.marker", path, "",
                          "feature file must contain exactly one closing marker");
    }
    const std::string opening = std::string(begin_marker) + "\n```json\n";
    if (contents.compare(begin, opening.size(), opening) != 0) {
        throw DomainError("ssfv.feature_file.fence", path, "",
                          "feature file opening marker or JSON fence is not exact");
    }
    const std::string closing = "\n```\n" + std::string(end_marker);
    if (end < 5U || contents.compare(end - 5U, closing.size(), closing) != 0) {
        throw DomainError("ssfv.feature_file.fence", path, "",
                          "feature file closing fence or marker is not exact");
    }
    const auto json_begin = begin + opening.size();
    const auto json_end = end - 5U;
    const auto json_text = contents.substr(json_begin, json_end - json_begin);
    engine::Json envelope;
    try {
        envelope = engine::parse_bounded_json(json_text, engine::Limits::max_snapshot_file_bytes);
    } catch (const engine::Error& error) {
        throw DomainError("ssfv.feature_file.json", path, "", error.what());
    }
    require_exact_fields(envelope,
        std::set<std::string>{"protocol", "owner_contract", "source_scope", "records"},
        "ssfv.feature_file.field_set");
    if (require_string(envelope, "protocol", 128U, true) != feature_file_protocol) {
        throw DomainError("ssfv.feature_file.protocol", path, "", "feature-file protocol mismatch");
    }
    FeatureFile result;
    result.path = path;
    result.file = file;
    result.owner_contract = require_string(envelope, "owner_contract", engine::Limits::max_path_bytes);
    result.source_scope = require_string(envelope, "source_scope", engine::Limits::max_path_bytes);
    if (!safe_repository_path(result.owner_contract) || !safe_scope(result.source_scope) ||
        expected_feature_file(result.source_scope) != path) {
        throw DomainError("ssfv.feature_file.routing", path, "",
                          "feature-file owner or source routing is invalid");
    }
    if (!envelope.at("records").is_array() || envelope.at("records").empty() ||
        envelope.at("records").size() > max_feature_records) {
        throw DomainError("ssfv.feature_file.records", path, "",
                          "feature-file record cardinality is invalid");
    }
    engine::Json normalized_records = engine::Json::array();
    std::string previous_id;
    for (const auto& value : envelope.at("records")) {
        auto record = normalize_record(value);
        if (record.owner_contract != result.owner_contract || record.source_scope != result.source_scope) {
            throw DomainError("ssfv.feature_file.owner_mismatch", path, record.feature_id,
                              "record owner routing differs from its feature file");
        }
        if (!previous_id.empty() && record.feature_id <= previous_id) {
            throw DomainError("ssfv.feature_file.order", path, record.feature_id,
                              "records must be unique and ordered by feature ID");
        }
        previous_id = record.feature_id;
        normalized_records.push_back(record.normalized);
        result.records.emplace(record.feature_id, std::move(record));
    }
    const engine::Json normalized_envelope{
        {"owner_contract", result.owner_contract},
        {"protocol", feature_file_protocol},
        {"records", std::move(normalized_records)},
        {"source_scope", result.source_scope},
    };
    if (json_text != normalized_envelope.dump(2)) {
        throw DomainError("ssfv.feature_file.canonical_format", path, "",
                          "feature-file JSON is not in deterministic canonical form");
    }
    result.prefix = contents.substr(0U, json_begin);
    result.suffix = contents.substr(json_end);
    return result;
}

std::string render_feature_file(const std::string& owner_contract, const std::string& source_scope,
                                const std::map<std::string, FeatureRecord>& records,
                                const std::optional<FeatureFile>& existing = std::nullopt) {
    if (records.empty()) {
        throw engine::Error("proposal.empty_feature_file", "canonical feature files cannot be empty", 4);
    }
    engine::Json values = engine::Json::array();
    for (const auto& [feature_id, record] : records) {
        static_cast<void>(feature_id);
        values.push_back(record.normalized);
    }
    engine::Json envelope{{"protocol", feature_file_protocol}, {"owner_contract", owner_contract},
                          {"source_scope", source_scope}, {"records", std::move(values)}};
    if (existing) {
        return existing->prefix + envelope.dump(2) + existing->suffix;
    }
    return "# Symphony Semantic Features\n\n" + std::string(begin_marker) +
           "\n```json\n" + envelope.dump(2) + "\n```\n" + std::string(end_marker) + "\n";
}

std::vector<NamespaceEntry> parse_namespaces(const std::string& contents) {
    if (!printable_bounded(contents, engine::Limits::max_snapshot_file_bytes)) {
        throw DomainError("ssfv.namespace.encoding", namespaces_path, "",
                          "namespace registry must be UTF-8 text");
    }
    const auto blocks = parse_ordered_blocks(
        section(contents, "## Canonical Namespace Entries"),
        namespace_fields, "namespace", max_namespaces);
    std::vector<NamespaceEntry> result;
    std::set<std::string> names;
    std::set<std::string> prefixes;
    for (const auto& fields : blocks) {
        const auto& name = fields.at("namespace");
        const auto& prefix = fields.at("id_prefix");
        if (!safe_feature_id(prefix + "a") || prefix != "ssfv:" + name + ":" ||
            !names.insert(name).second || !prefixes.insert(prefix).second ||
            !safe_repository_path(fields.at("owner_contract")) ||
            !safe_repository_path(fields.at("evidence")) ||
            !std::set<std::string>{"canonical", "deprecated", "retired"}.contains(fields.at("status"))) {
            throw DomainError("ssfv.namespace.invalid", namespaces_path, "",
                              "namespace entry is invalid or duplicated");
        }
        result.push_back(NamespaceEntry{fields});
    }
    if (result.empty()) {
        throw DomainError("ssfv.namespace.empty", namespaces_path, "",
                          "namespace registry must contain at least one entry");
    }
    return result;
}

std::vector<RegistryEntry> parse_registry(const std::string& contents, bool& empty) {
    if (!printable_bounded(contents, engine::Limits::max_snapshot_file_bytes)) {
        throw DomainError("ssfv.registry.encoding", registry_path, "",
                          "feature registry must be UTF-8 text");
    }
    const auto canonical = section(contents, "## Canonical Entries");
    const bool has_none = canonical.find("None.") != std::string::npos;
    const bool has_entries = canonical.find("- feature_id:") != std::string::npos;
    if (has_none) {
        if (has_entries || trim(canonical) != "None.") {
            throw DomainError("ssfv.registry.empty_grammar", registry_path, "",
                              "empty registry representation is mixed or malformed");
        }
        empty = true;
        return {};
    }
    empty = false;
    const auto blocks = parse_ordered_blocks(
        canonical, registry_fields, "feature_id", max_feature_records);
    if (blocks.empty()) {
        throw DomainError("ssfv.registry.empty_grammar", registry_path, "",
                          "registry has neither canonical entries nor exact None.");
    }
    std::vector<RegistryEntry> result;
    std::set<std::string> feature_ids;
    std::map<std::string, std::pair<std::string, std::string>> routes;
    for (const auto& fields : blocks) {
        const auto& id = fields.at("feature_id");
        const auto& file = fields.at("feature_file");
        const auto& owner = fields.at("owner_contract");
        const auto& scope = fields.at("source_scope");
        const auto& parent = fields.at("parent_feature_id");
        if (!safe_feature_id(id) || !feature_ids.insert(id).second ||
            !safe_repository_path(file) || !file.ends_with("FEATURES.md") ||
            !safe_repository_path(owner) || !safe_scope(scope) ||
            file != expected_feature_file(scope) || !tagged_digest(fields.at("record_digest")) ||
            !std::set<std::string>{"experimental", "implemented", "deprecated", "retired"}.contains(fields.at("status")) ||
            (parent != "none" && !safe_feature_id(parent))) {
            throw DomainError("ssfv.registry.entry_invalid", registry_path, id,
                              "registry entry is invalid or duplicated");
        }
        const auto route = std::make_pair(file, owner);
        if (const auto found = routes.find(scope); found != routes.end() && found->second != route) {
            throw DomainError("ssfv.registry.route_conflict", registry_path, id,
                              "one source scope maps to inconsistent owner routing");
        }
        routes[scope] = route;
        result.push_back(RegistryEntry{fields});
    }
    return result;
}

engine::Json build_semantic_snapshot(const State& state) {
    engine::Json feature_files = engine::Json::array();
    for (const auto& [path, feature_file] : state.feature_files) {
        static_cast<void>(path);
        feature_files.push_back(file_json(feature_file.file));
    }
    std::ostringstream source_canonical;
    for (const auto& [path, file] : state.evidence_files) {
        source_canonical << path.size() << ':' << path << ':' << file.size << ':' << file.digest << '\n';
    }
    engine::Json records = engine::Json::array();
    for (const auto& [feature_id, record] : state.records) {
        const auto registry = state.registry_by_id.find(feature_id);
        if (registry == state.registry_by_id.end()) {
            continue;
        }
        bool evidence_complete = true;
        for (const auto& path : record.implementation_paths) {
            evidence_complete = evidence_complete && state.evidence_files.contains(path);
        }
        for (const auto& path : record.cross_vector_paths) {
            evidence_complete = evidence_complete && state.evidence_files.contains(path);
        }
        if (!evidence_complete) {
            continue;
        }
        engine::Json implementation = engine::Json::array();
        for (const auto& path : record.implementation_paths) {
            implementation.push_back(file_json(state.evidence_files.at(path)));
        }
        engine::Json cross = engine::Json::array();
        for (const auto& path : record.cross_vector_paths) {
            cross.push_back(file_json(state.evidence_files.at(path)));
        }
        const auto& registry_entry = registry->second.fields;
        records.push_back(engine::Json{
            {"feature_id", feature_id},
            {"feature_file", registry_entry.at("feature_file")},
            {"owner_contract", record.owner_contract},
            {"source_scope", record.source_scope},
            {"kind", record.kind},
            {"status", record.status},
            {"parent_feature_id", record.parent_feature_id
                ? engine::Json(*record.parent_feature_id) : engine::Json(nullptr)},
            {"record_digest", record.record_digest},
            {"implementation_files", std::move(implementation)},
            {"cross_vector_files", std::move(cross)},
        });
    }
    engine::Json result{
        {"protocol", semantic_snapshot_protocol},
        {"module_id", module_id},
        {"engine_id", engine_id},
        {"engine_version", engine_version},
        {"vector_id", vector_id},
        {"contract_digest", state.contract_snapshot.digest},
        {"namespace_digest", state.namespace_file.digest},
        {"registry_digest", state.registry_file.digest},
        {"source_digest", engine::tagged_sha256(source_canonical.str())},
        {"feature_files", std::move(feature_files)},
        {"records", std::move(records)},
    };
    result["snapshot_digest"] = engine::tagged_sha256(result.dump());
    return result;
}

void validate_hierarchy(State& state) {
    const std::map<std::string, std::string> expected_parent_kind = {
        {"feature", "capability"}, {"subfeature", "feature"}, {"microfeature", "subfeature"},
    };
    for (const auto& [feature_id, record] : state.records) {
        if (record.kind == "capability") {
            continue;
        }
        const auto parent = state.records.find(*record.parent_feature_id);
        if (parent == state.records.end()) {
            add_finding(state, Finding{"violation", "ssfv.hierarchy.parent_missing", std::nullopt,
                feature_id, "primary parent is not registered"});
        } else if (parent->second.kind != expected_parent_kind.at(record.kind)) {
            add_finding(state, Finding{"violation", "ssfv.hierarchy.parent_kind", std::nullopt,
                feature_id, "primary parent kind violates strict hierarchy progression"});
        }
    }
    for (const auto& [feature_id, record] : state.records) {
        std::set<std::string> visited;
        auto current = &record;
        while (current->parent_feature_id) {
            if (!visited.insert(current->feature_id).second) {
                add_finding(state, Finding{"violation", "ssfv.hierarchy.cycle", std::nullopt,
                    feature_id, "primary-parent cycle detected"});
                break;
            }
            const auto parent = state.records.find(*current->parent_feature_id);
            if (parent == state.records.end()) {
                break;
            }
            current = &parent->second;
        }
        for (const auto& relationship : record.normalized.at("relationships")) {
            const auto target = relationship.at("target_feature_id").get<std::string>();
            if (!state.records.contains(target)) {
                add_finding(state, Finding{"violation", "ssfv.relationship.target_missing", std::nullopt,
                    feature_id, "relationship target is not registered"});
            }
        }
        for (const auto& distinction : record.normalized.at("distinctions")) {
            const auto target = distinction.at("target_feature_id").get<std::string>();
            if (!state.records.contains(target)) {
                add_finding(state, Finding{"violation", "ssfv.distinction.target_missing", std::nullopt,
                    feature_id, "distinction target is not registered"});
            }
        }
    }
}

State load_state(std::int64_t deadline_unix_ms) {
    State state;
    state.contract_snapshot = engine::snapshot_files(fs::current_path(), contract_paths, deadline_unix_ms);
    for (const auto& file : state.contract_snapshot.files) {
        if (state.total_evidence_bytes + file.size > max_total_evidence_bytes) {
            throw engine::Error("ssfv.resource.total_bytes",
                                "total SSFV evidence byte limit exceeded", 5);
        }
        state.total_evidence_bytes += file.size;
        state.counted_evidence_paths.insert(file.path);
    }
    const auto namespace_contents = engine::read_regular_file_no_follow(
        fs::current_path(), namespaces_path, engine::Limits::max_snapshot_file_bytes, deadline_unix_ms);
    if (state.total_evidence_bytes + namespace_contents.size() > max_total_evidence_bytes) {
        throw engine::Error("ssfv.resource.total_bytes",
                            "total SSFV evidence byte limit exceeded", 5);
    }
    state.total_evidence_bytes += namespace_contents.size();
    state.counted_evidence_paths.insert(namespaces_path);
    state.namespace_file = engine::FileDigest{namespaces_path,
        static_cast<std::uint64_t>(namespace_contents.size()), engine::tagged_sha256(namespace_contents)};
    const auto registry_contents = engine::read_regular_file_no_follow(
        fs::current_path(), registry_path, engine::Limits::max_snapshot_file_bytes, deadline_unix_ms);
    if (state.total_evidence_bytes + registry_contents.size() > max_total_evidence_bytes) {
        throw engine::Error("ssfv.resource.total_bytes",
                            "total SSFV evidence byte limit exceeded", 5);
    }
    state.total_evidence_bytes += registry_contents.size();
    state.counted_evidence_paths.insert(registry_path);
    state.registry_file = engine::FileDigest{registry_path,
        static_cast<std::uint64_t>(registry_contents.size()), engine::tagged_sha256(registry_contents)};
    const auto skvi_contents = engine::read_regular_file_no_follow(
        fs::current_path(), skvi_index_path, engine::Limits::max_snapshot_file_bytes, deadline_unix_ms);
    state.skvi_file = engine::FileDigest{skvi_index_path,
        static_cast<std::uint64_t>(skvi_contents.size()), engine::tagged_sha256(skvi_contents)};

    try {
        state.namespaces = parse_namespaces(namespace_contents);
        add_finding(state, Finding{"pass", "ssfv.namespace.valid", namespaces_path, std::nullopt,
            "namespace registry is structurally valid"});
    } catch (const DomainError& error) {
        add_finding(state, Finding{"violation", error.code,
            error.path.empty() ? std::optional<std::string>(namespaces_path)
                               : std::optional<std::string>(error.path),
            std::nullopt, error.what()});
    }
    try {
        state.registry_entries = parse_registry(registry_contents, state.registry_empty);
        add_finding(state, Finding{"pass", "ssfv.registry.valid", registry_path, std::nullopt,
            state.registry_empty ? "explicit empty registry is valid" : "feature registry is structurally valid"});
    } catch (const DomainError& error) {
        add_finding(state, Finding{"violation", error.code,
            error.path.empty() ? std::optional<std::string>(registry_path)
                               : std::optional<std::string>(error.path),
            error.feature_id.empty() ? std::nullopt : std::optional<std::string>(error.feature_id),
            error.what()});
    }

    std::set<std::string> active_namespaces;
    for (const auto& entry : state.namespaces) {
        active_namespaces.insert(entry.fields.at("namespace"));
        try {
            static_cast<void>(read_state_file(state, entry.fields.at("owner_contract"), deadline_unix_ms));
            static_cast<void>(read_state_file(state, entry.fields.at("evidence"), deadline_unix_ms));
        } catch (const DomainError& error) {
            add_finding(state, Finding{"violation", error.code, error.path, std::nullopt, error.what()});
        } catch (const engine::Error& error) {
            add_finding(state, Finding{"violation", "ssfv.namespace.reference_unreadable",
                namespaces_path, std::nullopt, error.what()});
        }
    }

    std::map<std::string, std::vector<RegistryEntry>> by_file;
    for (const auto& entry : state.registry_entries) {
        state.registry_by_id.emplace(entry.fields.at("feature_id"), entry);
        by_file[entry.fields.at("feature_file")].push_back(entry);
        if (!active_namespaces.contains(feature_namespace(entry.fields.at("feature_id")))) {
            add_finding(state, Finding{"violation", "ssfv.registry.namespace_unallocated",
                registry_path, entry.fields.at("feature_id"),
                "feature ID uses an unallocated namespace"});
        }
    }
    if (by_file.size() > max_feature_files) {
        throw engine::Error("ssfv.resource.feature_files", "feature-file count limit exceeded", 5);
    }

    for (const auto& [path, entries] : by_file) {
        try {
            const auto contents = engine::read_regular_file_no_follow(
                fs::current_path(), path, engine::Limits::max_snapshot_file_bytes, deadline_unix_ms);
            if (state.counted_evidence_paths.insert(path).second) {
                if (state.total_evidence_bytes + contents.size() > max_total_evidence_bytes) {
                    throw DomainError("ssfv.resource.total_bytes", path, "",
                                      "total SSFV evidence byte limit exceeded");
                }
                state.total_evidence_bytes += contents.size();
            }
            const engine::FileDigest file{path, static_cast<std::uint64_t>(contents.size()),
                                          engine::tagged_sha256(contents)};
            auto feature_file = parse_feature_file(path, contents, file);
            for (const auto& entry : entries) {
                if (entry.fields.at("owner_contract") != feature_file.owner_contract ||
                    entry.fields.at("source_scope") != feature_file.source_scope) {
                    throw DomainError("ssfv.registry.route_mismatch", path,
                                      entry.fields.at("feature_id"),
                                      "registry route differs from feature-file envelope");
                }
            }
            state.feature_files.emplace(path, std::move(feature_file));
        } catch (const DomainError& error) {
            add_finding(state, Finding{"violation", error.code, path,
                error.feature_id.empty() ? std::nullopt : std::optional<std::string>(error.feature_id),
                error.what()});
        } catch (const engine::Error& error) {
            add_finding(state, Finding{"violation", "ssfv.feature_file.unreadable", path,
                std::nullopt, error.what()});
        }
    }

    for (const auto& [path, file] : state.feature_files) {
        for (const auto& [feature_id, record] : file.records) {
            if (!state.records.emplace(feature_id, record).second) {
                add_finding(state, Finding{"violation", "ssfv.record.duplicate_id", path,
                    feature_id, "feature identity appears in more than one owner record"});
            }
        }
        if (!contains_exact_line(skvi_contents, "- path: `" + path + "`")) {
            add_finding(state, Finding{"violation", "ssfv.feature_file.skvi_unindexed", path,
                std::nullopt, "registered feature file is not indexed by SKVI"});
        }
    }

    for (const auto& [feature_id, entry] : state.registry_by_id) {
        const auto record = state.records.find(feature_id);
        if (record == state.records.end()) {
            add_finding(state, Finding{"violation", "ssfv.registry.record_missing",
                entry.fields.at("feature_file"), feature_id,
                "registry entry has no feature record"});
            continue;
        }
        const auto expected_parent = record->second.parent_feature_id
            ? *record->second.parent_feature_id : "none";
        if (entry.fields.at("record_digest") != record->second.record_digest ||
            entry.fields.at("status") != record->second.status ||
            entry.fields.at("parent_feature_id") != expected_parent ||
            entry.fields.at("owner_contract") != record->second.owner_contract ||
            entry.fields.at("source_scope") != record->second.source_scope) {
            add_finding(state, Finding{"violation", "ssfv.registry.record_mismatch",
                entry.fields.at("feature_file"), feature_id,
                "registry state differs from the normalized feature record"});
        }
    }
    for (const auto& [feature_id, record] : state.records) {
        if (!state.registry_by_id.contains(feature_id)) {
            add_finding(state, Finding{"violation", "ssfv.record.registry_missing",
                std::nullopt, feature_id, "feature record is not registered"});
            continue;
        }
        try {
            validate_scope_directory(record.source_scope);
            static_cast<void>(read_state_file(state, record.owner_contract, deadline_unix_ms));
            for (const auto& path : record.implementation_paths) {
                static_cast<void>(read_state_file(state, path, deadline_unix_ms));
            }
            for (const auto& path : record.cross_vector_paths) {
                static_cast<void>(read_state_file(state, path, deadline_unix_ms));
            }
        } catch (const DomainError& error) {
            add_finding(state, Finding{"violation", error.code,
                error.path.empty() ? std::nullopt : std::optional<std::string>(error.path),
                feature_id, error.what()});
        } catch (const engine::Error& error) {
            add_finding(state, Finding{"violation", "ssfv.record.evidence_unreadable",
                std::nullopt, feature_id, error.what()});
        }
    }
    validate_hierarchy(state);

    if (state.registry_empty && state.records.empty()) {
        state.coverage_state = "empty";
    } else {
        state.coverage_state = "partial";
    }
    if (state.structural_valid) {
        add_finding(state, Finding{"pass", "ssfv.structure.valid", std::nullopt, std::nullopt,
            "SSFV structural relationships are valid"});
    }
    state.semantic_snapshot = build_semantic_snapshot(state);
    return state;
}

struct SnapshotRecord final {
    std::string feature_id;
    std::string record_digest;
    std::map<std::string, std::string> implementation_digests;
    std::map<std::string, std::string> cross_vector_digests;
    engine::Json raw;
};

struct ParsedSnapshot final {
    std::string digest;
    std::string contract_digest;
    std::string namespace_digest;
    std::string registry_digest;
    std::string source_digest;
    std::map<std::string, SnapshotRecord> records;
};

ParsedSnapshot parse_snapshot(const engine::Json& value) {
    require_exact_fields(value, std::set<std::string>{
        "protocol", "module_id", "engine_id", "engine_version", "vector_id",
        "contract_digest", "namespace_digest", "registry_digest", "source_digest",
        "feature_files", "records", "snapshot_digest",
    }, "ssfv.snapshot.field_set");
    if (value.at("protocol") != semantic_snapshot_protocol ||
        value.at("module_id") != module_id || value.at("engine_id") != engine_id ||
        value.at("vector_id") != vector_id || !value.at("engine_version").is_string() ||
        !safe_token(value.at("engine_version").get_ref<const std::string&>(), 64U)) {
        throw engine::Error("ssfv.snapshot.identity", "semantic snapshot identity mismatch", 4);
    }
    for (const auto* field : {"contract_digest", "namespace_digest", "registry_digest",
                              "source_digest", "snapshot_digest"}) {
        if (!value.at(field).is_string() ||
            !tagged_digest(value.at(field).get_ref<const std::string&>())) {
            throw engine::Error("ssfv.snapshot.digest", "semantic snapshot contains an invalid digest", 4);
        }
    }
    auto canonical = value;
    const auto supplied_digest = canonical.at("snapshot_digest").get<std::string>();
    canonical.erase("snapshot_digest");
    if (engine::tagged_sha256(canonical.dump()) != supplied_digest) {
        throw engine::Error("ssfv.snapshot.digest_mismatch", "semantic snapshot digest mismatch", 4);
    }
    if (!value.at("feature_files").is_array() || value.at("feature_files").size() > max_feature_files ||
        !value.at("records").is_array() || value.at("records").size() > max_feature_records) {
        throw engine::Error("ssfv.snapshot.count", "semantic snapshot count limit exceeded", 4);
    }
    const auto parse_files = [&](const engine::Json& files, std::size_t minimum,
                                 std::size_t maximum,
                                 std::map<std::string, std::string>* digests = nullptr) {
        if (!files.is_array() || files.size() < minimum || files.size() > maximum) {
            throw engine::Error("ssfv.snapshot.file", "semantic snapshot file list is invalid", 4);
        }
        std::string previous_path;
        for (const auto& file : files) {
            require_exact_fields(file, std::set<std::string>{"path", "size", "digest"},
                                 "ssfv.snapshot.file_fields");
            const auto path = require_string(file, "path", engine::Limits::max_path_bytes);
            const auto digest = require_string(file, "digest", 71U);
            if (!safe_repository_path(path) || !tagged_digest(digest) ||
                !file.at("size").is_number_unsigned() ||
                file.at("size").get<std::uint64_t>() >
                    engine::Limits::max_snapshot_file_bytes ||
                (!previous_path.empty() && path <= previous_path)) {
                throw engine::Error("ssfv.snapshot.file", "semantic snapshot file is invalid", 4);
            }
            previous_path = path;
            if (digests && !digests->emplace(path, digest).second) {
                throw engine::Error("ssfv.snapshot.file",
                                    "semantic snapshot contains duplicate source paths", 4);
            }
        }
    };
    std::map<std::string, std::string> feature_file_digests;
    parse_files(value.at("feature_files"), 0U, max_feature_files,
                &feature_file_digests);
    ParsedSnapshot result{supplied_digest,
        value.at("contract_digest").get<std::string>(),
        value.at("namespace_digest").get<std::string>(),
        value.at("registry_digest").get<std::string>(),
        value.at("source_digest").get<std::string>(), {}};
    std::string previous;
    std::set<std::string> referenced_feature_files;
    for (const auto& record : value.at("records")) {
        require_exact_fields(record, std::set<std::string>{
            "feature_id", "feature_file", "owner_contract", "source_scope", "kind", "status",
            "parent_feature_id", "record_digest", "implementation_files", "cross_vector_files",
        }, "ssfv.snapshot.record_fields");
        const auto feature_id = require_string(record, "feature_id", 256U);
        const auto record_digest = require_string(record, "record_digest", 71U);
        const auto feature_file_path = require_string(
            record, "feature_file", engine::Limits::max_path_bytes);
        const auto owner_contract = require_string(
            record, "owner_contract", engine::Limits::max_path_bytes);
        const auto source_scope = require_string(
            record, "source_scope", engine::Limits::max_path_bytes);
        const auto kind = require_string(record, "kind", 32U, true);
        const auto status = require_string(record, "status", 32U, true);
        if (!safe_feature_id(feature_id) || !tagged_digest(record_digest) ||
            (!previous.empty() && feature_id <= previous) ||
            !safe_repository_path(feature_file_path) ||
            !safe_repository_path(owner_contract) || !safe_scope(source_scope) ||
            feature_file_path != expected_feature_file(source_scope) ||
            !feature_file_digests.contains(feature_file_path) ||
            !std::set<std::string>{"capability", "feature", "subfeature", "microfeature"}.contains(kind) ||
            !std::set<std::string>{"experimental", "implemented", "deprecated", "retired"}.contains(status)) {
            throw engine::Error("ssfv.snapshot.record", "semantic snapshot record is invalid", 4);
        }
        referenced_feature_files.insert(feature_file_path);
        if (record.at("parent_feature_id").is_null()) {
            if (kind != "capability") {
                throw engine::Error("ssfv.snapshot.record",
                                    "snapshot non-capability record has no parent", 4);
            }
        } else if (!record.at("parent_feature_id").is_string() ||
                   kind == "capability" ||
                   !safe_feature_id(record.at("parent_feature_id").get_ref<const std::string&>()) ||
                   record.at("parent_feature_id") == feature_id) {
            throw engine::Error("ssfv.snapshot.record",
                                "snapshot feature parent is invalid", 4);
        }
        previous = feature_id;
        SnapshotRecord parsed{feature_id, record_digest, {}, {}, record};
        parse_files(record.at("implementation_files"), 1U, 256U,
                    &parsed.implementation_digests);
        parse_files(record.at("cross_vector_files"), 0U, 128U,
                    &parsed.cross_vector_digests);
        result.records.emplace(feature_id, std::move(parsed));
    }
    if (referenced_feature_files.size() != feature_file_digests.size()) {
        throw engine::Error("ssfv.snapshot.feature_file",
                            "semantic snapshot contains an unreferenced feature file", 4);
    }
    const std::map<std::string, std::string> expected_parent_kind = {
        {"feature", "capability"}, {"subfeature", "feature"},
        {"microfeature", "subfeature"},
    };
    for (const auto& [feature_id, record] : result.records) {
        const auto kind = record.raw.at("kind").get<std::string>();
        if (kind == "capability") {
            continue;
        }
        const auto parent_id =
            record.raw.at("parent_feature_id").get<std::string>();
        const auto parent = result.records.find(parent_id);
        if (parent == result.records.end() ||
            parent->second.raw.at("kind").get<std::string>() !=
                expected_parent_kind.at(kind)) {
            throw engine::Error("ssfv.snapshot.hierarchy",
                                "semantic snapshot hierarchy is incomplete or invalid", 4);
        }
    }
    for (const auto& [feature_id, record] : result.records) {
        static_cast<void>(feature_id);
        std::set<std::string> visited;
        auto current = &record;
        while (!current->raw.at("parent_feature_id").is_null()) {
            if (!visited.insert(current->feature_id).second) {
                throw engine::Error("ssfv.snapshot.hierarchy",
                                    "semantic snapshot contains a parent cycle", 4);
            }
            const auto parent = result.records.find(
                current->raw.at("parent_feature_id").get<std::string>());
            if (parent == result.records.end()) {
                throw engine::Error("ssfv.snapshot.hierarchy",
                                    "semantic snapshot hierarchy is incomplete", 4);
            }
            current = &parent->second;
        }
    }
    return result;
}

struct Comparison final {
    std::vector<std::string> added;
    std::vector<std::string> changed;
    std::vector<std::string> removed;
    std::set<std::string> stale_paths;
    std::set<std::string> uncovered_paths;
    bool contract_changed = false;
    bool namespace_changed = false;
    bool registry_changed = false;
    bool source_changed = false;
    engine::Json candidates = engine::Json::array();
};

Comparison compare_snapshots(const ParsedSnapshot& baseline, const ParsedSnapshot& current,
                             const std::set<std::string>& scope, bool candidates) {
    Comparison result;
    result.contract_changed = baseline.contract_digest != current.contract_digest;
    result.namespace_changed = baseline.namespace_digest != current.namespace_digest;
    result.registry_changed = baseline.registry_digest != current.registry_digest;
    result.source_changed = scope.empty() && baseline.source_digest != current.source_digest;
    const auto included = [&](const std::string& feature_id) {
        return scope.empty() || scope.contains(feature_id);
    };
    for (const auto& [feature_id, current_record] : current.records) {
        if (!included(feature_id)) {
            continue;
        }
        const auto old = baseline.records.find(feature_id);
        if (old == baseline.records.end()) {
            result.added.push_back(feature_id);
            continue;
        }
        if (old->second.record_digest != current_record.record_digest) {
            result.changed.push_back(feature_id);
        }
        std::set<std::string> paths;
        for (const auto& [path, digest] : old->second.implementation_digests) {
            static_cast<void>(digest);
            paths.insert(path);
        }
        for (const auto& [path, digest] : current_record.implementation_digests) {
            static_cast<void>(digest);
            paths.insert(path);
        }
        for (const auto& path : paths) {
            const auto old_digest = old->second.implementation_digests.find(path);
            const auto new_digest = current_record.implementation_digests.find(path);
            if (old_digest != old->second.implementation_digests.end() &&
                new_digest == current_record.implementation_digests.end()) {
                result.uncovered_paths.insert(path);
            } else if (old_digest == old->second.implementation_digests.end() ||
                       old_digest->second != new_digest->second) {
                result.stale_paths.insert(path);
                if (candidates && result.candidates.size() < max_findings) {
                    result.candidates.push_back(engine::Json{
                        {"path", path}, {"feature_id", feature_id},
                        {"candidate_kind", "stale_semantics"},
                        {"reason", "cited implementation evidence changed; semantic review may be required"},
                        {"ratification_required", true},
                    });
                }
            }
        }
        paths.clear();
        for (const auto& [path, digest] : old->second.cross_vector_digests) {
            static_cast<void>(digest);
            paths.insert(path);
        }
        for (const auto& [path, digest] : current_record.cross_vector_digests) {
            static_cast<void>(digest);
            paths.insert(path);
        }
        for (const auto& path : paths) {
            const auto old_digest = old->second.cross_vector_digests.find(path);
            const auto new_digest = current_record.cross_vector_digests.find(path);
            if (old_digest == old->second.cross_vector_digests.end() ||
                new_digest == current_record.cross_vector_digests.end() ||
                old_digest->second != new_digest->second) {
                result.stale_paths.insert(path);
                if (candidates && result.candidates.size() < max_findings) {
                    result.candidates.push_back(engine::Json{
                        {"path", path}, {"feature_id", feature_id},
                        {"candidate_kind", "stale_semantics"},
                        {"reason", "applicable cross-vector evidence changed; semantic review may be required"},
                        {"ratification_required", true},
                    });
                }
            }
        }
    }
    for (const auto& [feature_id, record] : baseline.records) {
        static_cast<void>(record);
        if (included(feature_id) && !current.records.contains(feature_id)) {
            result.removed.push_back(feature_id);
        }
    }
    if (candidates) {
        const auto append_global_candidate = [&](bool changed, const char* path,
                                                 const char* kind, const char* reason) {
            if (changed && result.candidates.size() < max_findings) {
                result.candidates.push_back(engine::Json{
                    {"path", path}, {"feature_id", nullptr},
                    {"candidate_kind", kind}, {"reason", reason},
                    {"ratification_required", true},
                });
            }
        };
        append_global_candidate(
            result.contract_changed, "knowledge/ssfv/SPEC.md", "distinction_review",
            "SSFV contract snapshot changed between semantic snapshots");
        append_global_candidate(
            result.namespace_changed, namespaces_path, "stale_semantics",
            "SSFV namespace truth changed between semantic snapshots");
        append_global_candidate(
            result.registry_changed, registry_path, "stale_semantics",
            "SSFV feature routing changed between semantic snapshots");
        if (result.source_changed && result.stale_paths.empty() &&
            result.uncovered_paths.empty()) {
            append_global_candidate(
                true, ".", "stale_semantics",
                "cited owner or supporting source evidence changed between semantic snapshots");
        }
    }
    return result;
}

engine::Json inspect(const engine::Json& payload) {
    require_exact_fields(payload, std::set<std::string>{});
    return engine::Json{
        {"readiness", "read_check_diff_propose_graph"},
        {"descriptor", descriptor()},
        {"canonical_paths", engine::Json{
            {"namespaces", namespaces_path}, {"registry", registry_path},
            {"feature_file_format", "knowledge/ssfv/FEATURE-FILE-FORMAT.md"},
            {"skvi_index", skvi_index_path}}},
        {"empty_registry_valid", true},
        {"engine_decides_feature_worthiness", false},
        {"engine_decides_semantic_truth", false},
        {"canonical_apply_enabled", false},
    };
}

std::optional<std::string> optional_digest(const engine::Json& payload, const char* field) {
    if (payload.at(field).is_null()) {
        return std::nullopt;
    }
    if (!payload.at(field).is_string() ||
        !tagged_digest(payload.at(field).get_ref<const std::string&>())) {
        throw engine::Error("payload.invalid_digest", std::string(field) + " is not a tagged SHA-256", 4);
    }
    return payload.at(field).get<std::string>();
}

engine::Json check(const engine::Json& payload, std::int64_t deadline_unix_ms) {
    require_exact_fields(payload, std::set<std::string>{
        "expected_namespace_digest", "expected_registry_digest", "freshness", "baseline",
    });
    const auto expected_namespace = optional_digest(payload, "expected_namespace_digest");
    const auto expected_registry = optional_digest(payload, "expected_registry_digest");
    const auto freshness = require_string(payload, "freshness", 16U, true);
    if (!std::set<std::string>{"disabled", "report", "require"}.contains(freshness)) {
        throw engine::Error("payload.invalid_freshness", "freshness mode is unsupported", 4);
    }
    if (freshness == "disabled" && !payload.at("baseline").is_null()) {
        throw engine::Error("payload.baseline_prohibited", "disabled freshness requires null baseline", 4);
    }
    if (freshness != "disabled" && payload.at("baseline").is_null()) {
        throw engine::Error("payload.baseline_required", "report/require freshness needs a baseline", 4);
    }
    auto state = load_state(deadline_unix_ms);
    const auto structural_valid = state.structural_valid;
    std::optional<bool> namespace_matches;
    std::optional<bool> registry_matches;
    if (expected_namespace) {
        namespace_matches = *expected_namespace == state.namespace_file.digest;
        if (!*namespace_matches) {
            add_finding(state, Finding{"violation", "ssfv.namespace.expected_digest_mismatch",
                namespaces_path, std::nullopt, "namespace registry differs from expected digest"});
        }
    }
    if (expected_registry) {
        registry_matches = *expected_registry == state.registry_file.digest;
        if (!*registry_matches) {
            add_finding(state, Finding{"violation", "ssfv.registry.expected_digest_mismatch",
                registry_path, std::nullopt, "feature registry differs from expected digest"});
        }
    }
    std::string freshness_state = "not_evaluated";
    if (freshness != "disabled") {
        const auto baseline = parse_snapshot(payload.at("baseline"));
        const auto current = parse_snapshot(state.semantic_snapshot);
        const auto comparison = compare_snapshots(baseline, current, {}, true);
        const bool stale = !comparison.added.empty() || !comparison.changed.empty() ||
                           !comparison.removed.empty() || !comparison.stale_paths.empty() ||
                           !comparison.uncovered_paths.empty() ||
                           comparison.contract_changed || comparison.namespace_changed ||
                           comparison.registry_changed || comparison.source_changed;
        freshness_state = stale ? "stale" : "current";
        if (stale) {
            for (const auto& candidate : comparison.candidates) {
                add_finding(state, Finding{
                    freshness == "require" ? "violation" : "warning",
                    "ssfv.semantic_freshness.review_required",
                    candidate.at("path").get<std::string>(),
                    candidate.at("feature_id").is_string()
                        ? std::optional<std::string>(candidate.at("feature_id").get<std::string>())
                        : std::nullopt,
                    candidate.at("reason").get<std::string>(),
                    "semantic_candidate",
                });
            }
            for (const auto& id : comparison.added) {
                add_finding(state, Finding{
                    freshness == "require" ? "violation" : "warning",
                    "ssfv.semantic_freshness.feature_added", std::nullopt, id,
                    "feature identity was added after the baseline", "semantic_candidate"});
            }
            for (const auto& id : comparison.removed) {
                add_finding(state, Finding{
                    freshness == "require" ? "violation" : "warning",
                    "ssfv.semantic_freshness.feature_removed", std::nullopt, id,
                    "feature identity was removed after the baseline", "semantic_candidate"});
            }
            for (const auto& id : comparison.changed) {
                add_finding(state, Finding{
                    freshness == "require" ? "violation" : "warning",
                    "ssfv.semantic_freshness.record_changed", std::nullopt, id,
                    "feature semantics changed after the baseline", "semantic_candidate"});
            }
            for (const auto& path : comparison.uncovered_paths) {
                add_finding(state, Finding{
                    freshness == "require" ? "violation" : "warning",
                    "ssfv.semantic_freshness.path_uncovered", path, std::nullopt,
                    "previously cited implementation evidence is no longer covered",
                    "semantic_candidate"});
            }
        }
    }
    std::size_t warnings = 0U;
    std::size_t violations = 0U;
    engine::Json evidence = engine::Json::array();
    for (const auto& finding : state.findings) {
        warnings += finding.severity == "warning" ? 1U : 0U;
        violations += finding.severity == "violation" ? 1U : 0U;
        evidence.push_back(engine::Json{
            {"severity", finding.severity}, {"code", finding.code},
            {"path", finding.path ? engine::Json(*finding.path) : engine::Json(nullptr)},
            {"feature_id", finding.feature_id ? engine::Json(*finding.feature_id) : engine::Json(nullptr)},
            {"detail", finding.detail}, {"basis", finding.basis},
        });
    }
    return engine::Json{
        {"protocol", check_protocol},
        {"namespace_registry", file_json(state.namespace_file)},
        {"feature_registry", file_json(state.registry_file)},
        {"contract_snapshot", snapshot_json(state.contract_snapshot)},
        {"semantic_snapshot", state.semantic_snapshot},
        {"expected_namespace_matches", namespace_matches
            ? engine::Json(*namespace_matches) : engine::Json(nullptr)},
        {"expected_registry_matches", registry_matches
            ? engine::Json(*registry_matches) : engine::Json(nullptr)},
        {"freshness_mode", freshness},
        {"coverage_state", state.coverage_state},
        {"namespace_count", state.namespaces.size()},
        {"feature_count", state.records.size()},
        {"feature_file_count", state.feature_files.size()},
        {"structural_state", structural_valid ? "valid" : "invalid"},
        {"semantic_freshness_state", freshness_state},
        {"evidence", std::move(evidence)},
        {"summary", engine::Json{{"pass", state.passes}, {"warning", warnings},
            {"violation", violations}, {"state", violations == 0U ? "valid" : "invalid"}}},
        {"read_only", true},
        {"canonical_apply_enabled", false},
    };
}

engine::Json diff(const engine::Json& payload, std::int64_t deadline_unix_ms) {
    require_exact_fields(payload, std::set<std::string>{
        "baseline", "expected_current_snapshot_digest", "scope_feature_ids",
        "include_semantic_candidates",
    });
    const auto expected_current = optional_digest(payload, "expected_current_snapshot_digest");
    if (!payload.at("scope_feature_ids").is_array() ||
        payload.at("scope_feature_ids").size() > max_feature_records ||
        !payload.at("include_semantic_candidates").is_boolean()) {
        throw engine::Error("payload.invalid_diff", "diff scope or semantic-candidate flag is invalid", 4);
    }
    std::set<std::string> scope;
    for (const auto& value : payload.at("scope_feature_ids")) {
        if (!value.is_string() || !safe_feature_id(value.get_ref<const std::string&>()) ||
            !scope.insert(value.get<std::string>()).second) {
            throw engine::Error("payload.invalid_diff_scope", "diff scope contains an invalid or duplicate feature ID", 4);
        }
    }
    const auto baseline = parse_snapshot(payload.at("baseline"));
    const auto state = load_state(deadline_unix_ms);
    if (!state.structural_valid) {
        throw engine::Error("ssfv.diff.current_invalid",
                            "live canonical SSFV state is structurally invalid", 4);
    }
    const auto current = parse_snapshot(state.semantic_snapshot);
    if (expected_current && *expected_current != current.digest) {
        throw engine::Error("ssfv.diff.current_digest_mismatch",
                            "live semantic snapshot differs from expected digest", 4);
    }
    const bool include_candidates = payload.at("include_semantic_candidates").get<bool>();
    auto comparison = compare_snapshots(baseline, current, scope, include_candidates);
    std::string result_state = "identical";
    if (!comparison.removed.empty()) {
        result_state = "removal";
    } else if (!comparison.changed.empty()) {
        result_state = "semantic_change";
    } else if (!comparison.added.empty()) {
        result_state = "additive";
    }
    if (!comparison.stale_paths.empty() || !comparison.uncovered_paths.empty() ||
        comparison.contract_changed || comparison.namespace_changed ||
        comparison.registry_changed || comparison.source_changed) {
        result_state = "review_required";
    }
    engine::Json stale = engine::Json::array();
    for (const auto& path : comparison.stale_paths) {
        stale.push_back(path);
    }
    engine::Json uncovered = engine::Json::array();
    for (const auto& path : comparison.uncovered_paths) {
        uncovered.push_back(path);
    }
    engine::Json result{
        {"protocol", diff_protocol},
        {"baseline_digest", baseline.digest},
        {"current_digest", current.digest},
        {"state", result_state},
        {"added_feature_ids", comparison.added},
        {"changed_feature_ids", comparison.changed},
        {"removed_feature_ids", comparison.removed},
        {"uncovered_paths", std::move(uncovered)},
        {"stale_references", std::move(stale)},
        {"semantic_candidates", include_candidates ? comparison.candidates : engine::Json::array()},
        {"summary", engine::Json{
            {"added", comparison.added.size()}, {"changed", comparison.changed.size()},
            {"removed", comparison.removed.size()}, {"uncovered", comparison.uncovered_paths.size()},
            {"stale", comparison.stale_paths.size()},
            {"review_required", comparison.candidates.size()}}},
        {"read_only", true},
        {"noncanonical", true},
    };
    result["result_digest"] = engine::tagged_sha256(result.dump());
    return result;
}

std::string absence_digest(const std::string& path) {
    return engine::tagged_sha256("symphony.knowledge.path-state.v1\nabsent\n" + path);
}

engine::Json normalize_repository(const engine::Json& repository) {
    require_exact_fields(repository,
        std::set<std::string>{"repository_id", "revision", "worktree_id", "tree_digest"});
    const auto repository_id = require_string(repository, "repository_id", 256U);
    const auto worktree_id = require_string(repository, "worktree_id", 128U, true);
    const auto tree_digest = require_string(repository, "tree_digest", 71U);
    require_exact_fields(repository.at("revision"), std::set<std::string>{"scheme", "value"});
    const auto scheme = require_string(repository.at("revision"), "scheme", 128U, true);
    const auto value = require_string(repository.at("revision"), "value", 256U);
    if (!tagged_digest(tree_digest)) {
        throw engine::Error("payload.repository_digest", "repository tree digest is invalid", 4);
    }
    return engine::Json{{"repository_id", repository_id},
        {"revision", engine::Json{{"scheme", scheme}, {"value", value}}},
        {"worktree_id", worktree_id}, {"tree_digest", tree_digest}};
}

void validate_semantic_declaration(const engine::Json& declaration) {
    require_exact_fields(declaration, std::set<std::string>{
        "feature_worthiness_ratified", "owner_ratified", "rationale", "evidence",
    });
    if (!declaration.at("feature_worthiness_ratified").is_boolean() ||
        !declaration.at("owner_ratified").is_boolean()) {
        throw engine::Error("payload.semantic_declaration", "semantic declaration booleans are invalid", 4);
    }
    static_cast<void>(require_string(declaration, "rationale", max_description_bytes));
    try {
        validate_string_array(declaration.at("evidence"), 1U, 256U,
                              engine::Limits::max_path_bytes, "semantic_declaration.evidence");
    } catch (const DomainError&) {
        throw engine::Error("payload.semantic_declaration",
                            "semantic declaration evidence is invalid", 4);
    }
}

std::set<std::string> target_path_set(const engine::Json& value) {
    if (!value.is_array() || value.empty() || value.size() > 4U) {
        throw engine::Error("payload.target_paths", "target_paths has invalid cardinality", 4);
    }
    std::set<std::string> result;
    for (const auto& item : value) {
        if (!item.is_string() || !safe_repository_path(item.get_ref<const std::string&>()) ||
            !result.insert(item.get<std::string>()).second) {
            throw engine::Error("payload.target_paths", "target_paths contains an unsafe or duplicate path", 4);
        }
    }
    return result;
}

engine::Json namespace_entry_json(const engine::Json& value) {
    require_exact_fields(value, namespace_fields, "payload.namespace_entry_fields");
    const auto name = require_string(value, "namespace", 63U, true);
    const auto prefix = require_string(value, "id_prefix", 70U);
    const auto owner = require_string(value, "owner_contract", engine::Limits::max_path_bytes);
    const auto scope = require_string(value, "scope", engine::Limits::max_path_bytes);
    const auto status = require_string(value, "status", 32U, true);
    const auto notes = require_string(value, "notes", max_description_bytes);
    if (prefix != "ssfv:" + name + ":" || !safe_feature_id(prefix + "a") ||
        !safe_repository_path(owner) ||
        !std::set<std::string>{"canonical", "deprecated", "retired"}.contains(status) ||
        !value.at("evidence").is_array() || value.at("evidence").empty() ||
        value.at("evidence").size() > 64U) {
        throw engine::Error("payload.namespace_entry", "namespace entry is invalid", 4);
    }
    std::vector<std::string> evidence;
    for (const auto& item : value.at("evidence")) {
        if (!item.is_string() ||
            !safe_repository_path(item.get_ref<const std::string&>())) {
            throw engine::Error("payload.namespace_entry", "namespace evidence path is invalid", 4);
        }
        evidence.push_back(item.get<std::string>());
    }
    std::sort(evidence.begin(), evidence.end());
    if (std::adjacent_find(evidence.begin(), evidence.end()) != evidence.end()) {
        throw engine::Error("payload.namespace_entry", "namespace evidence contains a duplicate", 4);
    }
    return engine::Json{{"namespace", name}, {"id_prefix", prefix}, {"owner_contract", owner},
        {"scope", scope}, {"status", status}, {"evidence", evidence}, {"notes", notes}};
}

void validate_proposed_record(State& state, const FeatureRecord& record,
                              std::int64_t deadline_unix_ms) {
    const auto allocated = std::any_of(
        state.namespaces.begin(), state.namespaces.end(), [&](const NamespaceEntry& entry) {
            return entry.fields.at("namespace") == feature_namespace(record.feature_id);
        });
    if (!allocated) {
        throw engine::Error("proposal.namespace_unallocated",
                            "proposed feature ID uses an unallocated namespace", 4);
    }
    try {
        validate_scope_directory(record.source_scope);
        static_cast<void>(read_state_file(state, record.owner_contract, deadline_unix_ms));
        for (const auto& path : record.implementation_paths) {
            static_cast<void>(read_state_file(state, path, deadline_unix_ms));
        }
        for (const auto& path : record.cross_vector_paths) {
            static_cast<void>(read_state_file(state, path, deadline_unix_ms));
        }
    } catch (const DomainError& error) {
        throw engine::Error("proposal.record_reference", error.what(), 4);
    } catch (const engine::Error& error) {
        throw engine::Error("proposal.record_reference", error.what(), 4);
    }

    State prospective;
    prospective.records = state.records;
    prospective.records[record.feature_id] = record;
    validate_hierarchy(prospective);
    if (!prospective.structural_valid) {
        throw engine::Error("proposal.record_graph",
                            "proposed feature record violates hierarchy or graph constraints", 4);
    }
}

engine::Json proposal(const engine::Json& payload, std::int64_t deadline_unix_ms) {
    require_exact_fields(payload, std::set<std::string>{
        "repository", "session_ref", "context_ref", "created_at", "expires_at", "operation",
    });
    const auto repository = normalize_repository(payload.at("repository"));
    const auto created_at = require_string(payload, "created_at", 20U);
    const auto expires_at = require_string(payload, "expires_at", 20U);
    if (!strict_utc(created_at) || !strict_utc(expires_at) || expires_at <= created_at) {
        throw engine::Error("payload.proposal_time", "proposal time interval is invalid", 4);
    }
    for (const auto* field : {"session_ref", "context_ref"}) {
        if (!payload.at(field).is_null() &&
            (!payload.at(field).is_string() ||
             !safe_token(payload.at(field).get_ref<const std::string&>()))) {
            throw engine::Error("payload.proposal_reference", "session/context reference is invalid", 4);
        }
    }
    const auto& operation = payload.at("operation");
    require_exact_fields(operation, std::set<std::string>{
        "type", "expected_contract_digest", "expected_namespace_digest",
        "expected_registry_digest", "expected_feature_digest", "affected_feature_ids",
        "target_paths", "namespace_entry", "feature_file", "feature_record",
        "registry_notes", "semantic_declaration", "authorization_ref",
    });
    const auto type = require_string(operation, "type", 64U, true);
    const auto expected_contract = require_string(operation, "expected_contract_digest", 71U);
    const auto expected_namespace = require_string(operation, "expected_namespace_digest", 71U);
    const auto expected_registry = require_string(operation, "expected_registry_digest", 71U);
    if (!tagged_digest(expected_contract) || !tagged_digest(expected_namespace) ||
        !tagged_digest(expected_registry)) {
        throw engine::Error("payload.expected_digest", "proposal expected digest is invalid", 4);
    }
    validate_semantic_declaration(operation.at("semantic_declaration"));
    const auto authorization_ref = require_string(
        operation, "authorization_ref", engine::Limits::max_path_bytes);
    const auto supplied_targets = target_path_set(operation.at("target_paths"));
    auto state = load_state(deadline_unix_ms);
    if (!state.structural_valid) {
        throw engine::Error("proposal.current_invalid", "canonical SSFV state is structurally invalid", 4);
    }
    if (expected_contract != state.contract_snapshot.digest ||
        expected_namespace != state.namespace_file.digest ||
        expected_registry != state.registry_file.digest) {
        throw engine::Error("proposal.expected_state_mismatch", "proposal expected state is stale", 4);
    }

    engine::Json writes = engine::Json::array();
    engine::Json operations = engine::Json::array();
    std::set<std::string> computed_targets;
    const auto add_write = [&](const std::string& path, const std::string& prior,
                               const std::string& desired, const std::string& operation_type,
                               engine::Json data) {
        computed_targets.insert(path);
        writes.push_back(engine::Json{{"target_path", path},
            {"expected_prior_digest", prior}, {"desired_change_digest", desired}});
        operations.push_back(engine::Json{
            {"operation_id", "ssfv-operation:" + std::to_string(operations.size() + 1U)},
            {"type", operation_type}, {"target_path", path},
            {"expected_state_digest", prior}, {"desired_change_digest", desired},
            {"data", std::move(data)},
        });
    };

    if (type == "allocate_namespace") {
        if (!operation.at("affected_feature_ids").is_array() ||
            !operation.at("affected_feature_ids").empty() ||
            !operation.at("expected_feature_digest").is_null() ||
            operation.at("namespace_entry").is_null() ||
            !operation.at("feature_file").is_null() ||
            !operation.at("feature_record").is_null() ||
            !operation.at("registry_notes").is_null()) {
            throw engine::Error("proposal.namespace_shape", "namespace proposal shape is invalid", 4);
        }
        const auto entry = namespace_entry_json(operation.at("namespace_entry"));
        const auto name = entry.at("namespace").get<std::string>();
        if (std::any_of(state.namespaces.begin(), state.namespaces.end(), [&](const NamespaceEntry& existing) {
                return existing.fields.at("namespace") == name;
            })) {
            throw engine::Error("proposal.namespace_conflict", "namespace is already allocated", 4);
        }
        try {
            static_cast<void>(read_state_file(
                state, entry.at("owner_contract").get<std::string>(), deadline_unix_ms));
            for (const auto& evidence : entry.at("evidence")) {
                static_cast<void>(read_state_file(
                    state, evidence.get<std::string>(), deadline_unix_ms));
            }
        } catch (const DomainError& error) {
            throw engine::Error("proposal.namespace_reference", error.what(), 4);
        } catch (const engine::Error& error) {
            throw engine::Error("proposal.namespace_reference", error.what(), 4);
        }
        const engine::Json data{{"namespace_entry", entry},
            {"authorization_ref", authorization_ref}, {"target_must_be_absent", false}};
        add_write(namespaces_path, state.namespace_file.digest,
                  engine::tagged_sha256(data.dump()), "allocate_namespace", data);
    } else {
        const std::set<std::string> feature_types = {
            "add_feature", "update_feature", "move_feature",
            "deprecate_feature", "retire_feature",
        };
        if (!feature_types.contains(type) || !operation.at("affected_feature_ids").is_array() ||
            operation.at("affected_feature_ids").size() != 1U ||
            !operation.at("affected_feature_ids").at(0).is_string() ||
            !operation.at("namespace_entry").is_null() ||
            !operation.at("feature_file").is_string() ||
            operation.at("feature_record").is_null() ||
            !operation.at("registry_notes").is_string()) {
            throw engine::Error("proposal.feature_shape", "feature proposal shape is invalid", 4);
        }
        auto record = [&]() {
            try {
                return normalize_record(operation.at("feature_record"));
            } catch (const DomainError& error) {
                throw engine::Error(error.code, error.what(), 4);
            }
        }();
        validate_proposed_record(state, record, deadline_unix_ms);
        const auto affected_id = operation.at("affected_feature_ids").at(0).get<std::string>();
        const auto feature_file_path = operation.at("feature_file").get<std::string>();
        const auto notes = operation.at("registry_notes").get<std::string>();
        if (affected_id != record.feature_id || feature_file_path != expected_feature_file(record.source_scope) ||
            !printable_bounded(notes, max_description_bytes)) {
            throw engine::Error("proposal.feature_identity", "feature proposal identity or routing mismatch", 4);
        }
        const auto existing_entry = state.registry_by_id.find(record.feature_id);
        const bool exists = existing_entry != state.registry_by_id.end();
        if (type == "add_feature") {
            if (exists || !operation.at("expected_feature_digest").is_null()) {
                throw engine::Error("proposal.feature_conflict", "add_feature requires a new identity and null prior digest", 4);
            }
        } else {
            if (!exists || !operation.at("expected_feature_digest").is_string() ||
                operation.at("expected_feature_digest").get<std::string>() !=
                    state.records.at(record.feature_id).record_digest) {
                throw engine::Error("proposal.expected_feature_mismatch", "existing feature digest is absent or stale", 4);
            }
        }
        if (type == "deprecate_feature" && record.status != "deprecated") {
            throw engine::Error("proposal.lifecycle", "deprecate_feature requires deprecated status", 4);
        }
        if (type == "retire_feature" && record.status != "retired") {
            throw engine::Error("proposal.lifecycle", "retire_feature requires retired status", 4);
        }
        if (exists && type != "move_feature" &&
            feature_file_path != existing_entry->second.fields.at("feature_file")) {
            throw engine::Error("proposal.move_required", "owner-file change requires move_feature", 4);
        }
        if (type == "move_feature" && exists &&
            feature_file_path == existing_entry->second.fields.at("feature_file")) {
            throw engine::Error("proposal.move_noop", "move_feature requires a changed owner file", 4);
        }

        std::map<std::string, FeatureRecord> target_records;
        std::optional<FeatureFile> target_existing;
        if (const auto found = state.feature_files.find(feature_file_path);
            found != state.feature_files.end()) {
            target_existing = found->second;
            if (found->second.owner_contract != record.owner_contract ||
                found->second.source_scope != record.source_scope) {
                throw engine::Error("proposal.route_conflict", "target feature file belongs to another route", 4);
            }
            target_records = found->second.records;
        } else {
            std::error_code status_error;
            const auto status = fs::symlink_status(
                fs::current_path() / feature_file_path, status_error);
            if (status_error && status_error != std::errc::no_such_file_or_directory) {
                throw engine::Error("proposal.target_state",
                                    "prospective feature-file state is unreadable", 4);
            }
            if (!status_error && status.type() != fs::file_type::not_found) {
                throw engine::Error("proposal.target_exists_unregistered",
                                    "prospective feature file must be absent", 4);
            }
        }
        target_records[record.feature_id] = record;
        const auto rendered = render_feature_file(
            record.owner_contract, record.source_scope, target_records, target_existing);
        const auto target_prior = target_existing
            ? target_existing->file.digest : absence_digest(feature_file_path);
        const auto target_data = engine::Json{
            {"feature_id", record.feature_id}, {"feature_record", record.normalized},
            {"record_digest", record.record_digest}, {"rendered_content_digest", engine::tagged_sha256(rendered)},
            {"target_must_be_absent", !target_existing.has_value()},
            {"authorization_ref", authorization_ref},
        };
        add_write(feature_file_path, target_prior, engine::tagged_sha256(rendered),
                  target_existing ? "replace_feature_file_region" : "create_feature_file",
                  target_data);

        if (type == "move_feature") {
            const auto old_path = existing_entry->second.fields.at("feature_file");
            auto old_records = state.feature_files.at(old_path).records;
            old_records.erase(record.feature_id);
            if (old_records.empty()) {
                const auto desired = absence_digest(old_path);
                add_write(old_path, state.feature_files.at(old_path).file.digest, desired,
                          "delete_empty_feature_file",
                          engine::Json{{"feature_id", record.feature_id},
                              {"target_must_be_absent", false},
                              {"desired_path_state", "absent"}});
            } else {
                const auto rendered_old = render_feature_file(
                    state.feature_files.at(old_path).owner_contract,
                    state.feature_files.at(old_path).source_scope, old_records,
                    state.feature_files.at(old_path));
                add_write(old_path, state.feature_files.at(old_path).file.digest,
                          engine::tagged_sha256(rendered_old), "replace_feature_file_region",
                          engine::Json{{"feature_id", record.feature_id},
                              {"rendered_content_digest", engine::tagged_sha256(rendered_old)},
                              {"operation", "remove_moved_record"}});
            }
        }

        const engine::Json registry_data{
            {"feature_id", record.feature_id}, {"feature_file", feature_file_path},
            {"owner_contract", record.owner_contract}, {"source_scope", record.source_scope},
            {"status", record.status},
            {"parent_feature_id", record.parent_feature_id
                ? engine::Json(*record.parent_feature_id) : engine::Json("none")},
            {"record_digest", record.record_digest}, {"notes", notes},
            {"operation", type},
        };
        add_write(registry_path, state.registry_file.digest,
                  engine::tagged_sha256(registry_data.dump()), "update_feature_registry",
                  registry_data);
        const bool index_change = !target_existing.has_value() || type == "move_feature";
        if (index_change) {
            const engine::Json index_data{
                {"operation", type == "move_feature" ? "move_feature_owner" : "add_feature_owner"},
                {"feature_file", feature_file_path},
                {"old_feature_file", exists
                    ? engine::Json(existing_entry->second.fields.at("feature_file"))
                    : engine::Json(nullptr)},
            };
            add_write(skvi_index_path, state.skvi_file.digest,
                      engine::tagged_sha256(index_data.dump()), "update_skvi_index", index_data);
        }
    }

    if (computed_targets != supplied_targets) {
        throw engine::Error("proposal.target_set_mismatch",
                            "caller target_paths differ from deterministic write targets", 4);
    }
    std::vector<engine::FileDigest> read_files = state.contract_snapshot.files;
    for (const auto& [path, file] : state.feature_files) {
        static_cast<void>(path);
        read_files.push_back(file.file);
    }
    for (const auto& [path, file] : state.evidence_files) {
        static_cast<void>(path);
        read_files.push_back(file);
    }
    std::sort(read_files.begin(), read_files.end(), [](const auto& left, const auto& right) {
        return left.path < right.path;
    });
    read_files.erase(std::unique(read_files.begin(), read_files.end(), [](const auto& left, const auto& right) {
        return left.path == right.path;
    }), read_files.end());
    engine::Json read_set = engine::Json::array();
    for (const auto& file : read_files) {
        read_set.push_back(file_json(file));
    }
    engine::Json result{
        {"protocol", proposal_protocol},
        {"module_id", module_id},
        {"engine_id", engine_id},
        {"engine_version", engine_version},
        {"vector_id", vector_id},
        {"contract_versions", engine::Json::array({
            "knowledge/SPEC.md@v1", "knowledge/ssfv/SPEC.md@engine-v1",
            "symphony.ssfv.feature-record.v2", "symphony.ssfv.proposal-input.v2"})},
        {"repository", repository},
        {"session_ref", payload.at("session_ref")},
        {"context_ref", payload.at("context_ref")},
        {"read_set", std::move(read_set)},
        {"write_set", std::move(writes)},
        {"operations", std::move(operations)},
        {"validation", engine::Json::array({
            engine::Json{{"code", "ssfv.expected_state.bound"}, {"outcome", "pass"},
                         {"detail", "contract, namespace, registry, and feature state are content-addressed"}},
            engine::Json{{"code", "ssfv.semantic_authority.external"}, {"outcome", "pass"},
                         {"detail", "engine output remains caller-declared and unratified"}},
        })},
        {"authority", engine::Json{{"caller_declared_operation", true},
            {"engine_decided_domain_truth", false}, {"ratified", false}}},
        {"created_at", created_at},
        {"expires_at", expires_at},
        {"canonical_apply_enabled", false},
    };
    result["proposal_id"] = "ssfv-proposal:" + engine::sha256_hex(result.dump()).substr(0U, 48U);
    result["proposal_digest"] = engine::tagged_sha256(result.dump());
    return result;
}

engine::Json graph(const engine::Json& payload, std::int64_t deadline_unix_ms) {
    require_exact_fields(payload, std::set<std::string>{"format"});
    if (require_string(payload, "format", 16U, true) != "json") {
        throw engine::Error("payload.unsupported_format", "graph format must be json", 4);
    }
    const auto state = load_state(deadline_unix_ms);
    if (!state.structural_valid) {
        throw engine::Error("ssfv.graph.current_invalid",
                            "live canonical SSFV state is structurally invalid", 4);
    }
    engine::Json nodes = engine::Json::array();
    engine::Json edges = engine::Json::array();
    std::set<std::string> edge_keys;
    const auto append_edge = [&](engine::Json edge) {
        if (edges.size() >= max_graph_edges) {
            throw engine::Error("ssfv.graph.edge_limit", "graph edge limit exceeded", 5);
        }
        edges.push_back(std::move(edge));
    };
    for (const auto& [feature_id, record] : state.records) {
        const auto& route = state.registry_by_id.at(feature_id).fields;
        nodes.push_back(engine::Json{
            {"feature_id", feature_id}, {"title", record.title}, {"kind", record.kind},
            {"status", record.status}, {"feature_file", route.at("feature_file")},
            {"source_scope", record.source_scope}, {"record_digest", record.record_digest},
        });
        if (record.parent_feature_id) {
            const auto key = feature_id + "\n" + *record.parent_feature_id + "\nprimary_parent";
            edge_keys.insert(key);
            append_edge(engine::Json{
                {"source_feature_id", feature_id},
                {"target_feature_id", *record.parent_feature_id},
                {"type", "primary_parent"},
                {"rationale", "primary semantic containment"},
            });
        }
        for (const auto& relationship : record.normalized.at("relationships")) {
            const auto target = relationship.at("target_feature_id").get<std::string>();
            const auto type = relationship.at("type").get<std::string>();
            const auto key = feature_id + "\n" + target + "\n" + type;
            if (edge_keys.insert(key).second) {
                append_edge(engine::Json{
                    {"source_feature_id", feature_id}, {"target_feature_id", target},
                    {"type", type}, {"rationale", relationship.at("rationale")},
                });
            }
        }
        for (const auto& distinction : record.normalized.at("distinctions")) {
            const auto target = distinction.at("target_feature_id").get<std::string>();
            const auto key = feature_id + "\n" + target + "\ndistinguished_from";
            if (edge_keys.insert(key).second) {
                append_edge(engine::Json{
                    {"source_feature_id", feature_id}, {"target_feature_id", target},
                    {"type", "distinguished_from"}, {"rationale", distinction.at("distinction")},
                });
            }
        }
    }
    engine::Json result{
        {"protocol", graph_protocol},
        {"projection_kind", "semantic-feature-graph"},
        {"format", "json"},
        {"module_id", module_id},
        {"engine_id", engine_id},
        {"engine_version", engine_version},
        {"vector_id", vector_id},
        {"contract_snapshot", snapshot_json(state.contract_snapshot)},
        {"namespace_digest", state.namespace_file.digest},
        {"registry_digest", state.registry_file.digest},
        {"node_count", nodes.size()},
        {"edge_count", edges.size()},
        {"nodes", std::move(nodes)},
        {"edges", std::move(edges)},
        {"noncanonical", true},
        {"rebuildable", true},
    };
    result["projection_digest"] = engine::tagged_sha256(result.dump());
    return result;
}

void require_deadline(std::int64_t deadline_unix_ms) {
    if (engine::unix_time_ms() >= deadline_unix_ms) {
        throw engine::Error("request.deadline_expired", "request deadline has expired", 3);
    }
}

void verify_self_digest(const engine::Json& document, const char* field,
                        const std::string& code) {
    if (!document.contains(field) || !document.at(field).is_string() ||
        !tagged_digest(document.at(field).get_ref<const std::string&>())) {
        throw engine::Error(code, std::string(field) + " must be a tagged SHA-256", 4);
    }
    auto canonical = document;
    const auto supplied = canonical.at(field).get<std::string>();
    canonical.erase(field);
    if (engine::tagged_sha256(canonical.dump()) != supplied) {
        throw engine::Error(code, std::string(field) + " does not bind the document", 4);
    }
}

std::vector<std::string> strict_string_array(
    const engine::Json& value, std::size_t maximum, const std::string& code,
    const std::function<bool(std::string_view)>& validator, bool sorted = true) {
    if (!value.is_array() || value.size() > maximum) {
        throw engine::Error(code, "array cardinality is invalid", 4);
    }
    std::vector<std::string> result;
    for (const auto& item : value) {
        if (!item.is_string() || !validator(item.get_ref<const std::string&>())) {
            throw engine::Error(code, "array contains an invalid value", 4);
        }
        result.push_back(item.get<std::string>());
    }
    if (sorted && (!std::is_sorted(result.begin(), result.end()) ||
                   std::adjacent_find(result.begin(), result.end()) != result.end())) {
        throw engine::Error(code, "array must be sorted and unique", 4);
    }
    return result;
}

struct ParsedCommandRegistry final {
    std::string kind;
    std::string digest;
    std::map<std::string, engine::Json> commands;
};

ParsedCommandRegistry parse_command_registry(const engine::Json& value,
                                              const std::string& required_kind) {
    const auto version = [](std::string_view item) {
        return !item.empty() && item.size() <= 64U &&
            std::all_of(item.begin(), item.end(), [](const unsigned char character) {
                return (character >= '0' && character <= '9') ||
                       (character >= 'A' && character <= 'Z') ||
                       (character >= 'a' && character <= 'z') ||
                       character == '.' || character == '+' || character == '-';
            });
    };
    require_exact_fields(value, std::set<std::string>{
        "protocol", "format_version", "registry_kind", "client_id", "client_version",
        "client_trust", "executable_digest", "receipt_digest", "commands", "registry_digest",
    }, "administration.command_registry.field_set");
    if (value.at("protocol") != "symphony.qxctl.command-registry.v1" ||
        value.at("format_version") != 1 || value.at("registry_kind") != required_kind ||
        value.at("client_id") != "qxctl" || !value.at("commands").is_array() ||
        value.at("commands").size() > 1024U) {
        throw engine::Error("administration.command_registry.identity",
                            "command registry identity or cardinality is invalid", 4);
    }
    if (!value.at("client_trust").is_string() ||
        !std::set<std::string>{"receipted", "unreceipted"}.contains(
            value.at("client_trust").get<std::string>())) {
        throw engine::Error("administration.command_registry.identity",
                            "command registry client trust is invalid", 4);
    }
    if (required_kind == "expected") {
        if (!value.at("client_version").is_null() ||
            value.at("client_trust") != "unreceipted" ||
            !value.at("executable_digest").is_null() ||
            !value.at("receipt_digest").is_null()) {
            throw engine::Error("administration.command_registry.identity",
                                "expected command registry contains observed identity", 4);
        }
    } else if (!value.at("client_version").is_string() ||
               !version(value.at("client_version").get_ref<const std::string&>()) ||
               !value.at("executable_digest").is_string() ||
               !tagged_digest(value.at("executable_digest").get_ref<const std::string&>())) {
        throw engine::Error("administration.command_registry.identity",
                            "observed qxctl identity is invalid", 4);
    }
    if (!value.at("receipt_digest").is_null() &&
        (!value.at("receipt_digest").is_string() ||
         !tagged_digest(value.at("receipt_digest").get_ref<const std::string&>()))) {
        throw engine::Error("administration.command_registry.identity",
                            "command registry receipt digest is invalid", 4);
    }
    if ((value.at("client_trust") == "receipted") !=
        value.at("receipt_digest").is_string()) {
        throw engine::Error("administration.command_registry.identity",
                            "command registry trust and receipt evidence differ", 4);
    }
    verify_self_digest(value, "registry_digest", "administration.command_registry.digest");

    ParsedCommandRegistry result{required_kind,
        value.at("registry_digest").get<std::string>(), {}};
    std::string previous;
    const auto stable = [](std::string_view item) { return safe_token(item, 256U); };
    const auto printable = [](std::string_view item) {
        return printable_bounded(item, 1024U);
    };
    for (const auto& command : value.at("commands")) {
        require_exact_fields(command, std::set<std::string>{
            "command_id", "status", "introduced_in", "deprecated_in", "replacement_ids",
            "grammar", "aliases", "visibility", "feature_bindings", "infrastructure_purpose",
            "backend_operation_ids", "mutability", "authority_mode", "target_scope",
            "input_protocols", "output_protocols", "result_validation_protocols",
            "recovery_command_id", "noninteractive", "json_output",
        }, "administration.command.field_set");
        const auto command_id = require_string(command, "command_id", 256U);
        if (!safe_prefixed_id(command_id, "qxcmd:") ||
            (!previous.empty() && command_id <= previous)) {
            throw engine::Error("administration.command.identity",
                                "command IDs must be valid, sorted, and unique", 4);
        }
        previous = command_id;
        for (const auto* field : {"status", "visibility",
                                  "mutability", "authority_mode", "target_scope"}) {
            static_cast<void>(require_string(command, field, 4096U));
        }
        if (!command.at("introduced_in").is_string() ||
            !version(command.at("introduced_in").get_ref<const std::string&>())) {
            throw engine::Error("administration.command.lifecycle",
                                "introduced version is invalid", 4);
        }
        if (!command.at("grammar").is_null() &&
            (!command.at("grammar").is_string() ||
             !printable_bounded(command.at("grammar").get_ref<const std::string&>(), 1024U))) {
            throw engine::Error("administration.command.grammar", "command grammar is invalid", 4);
        }
        const auto status = command.at("status").get<std::string>();
        if (!std::set<std::string>{"experimental", "stable", "deprecated", "retired"}
                 .contains(status) ||
            !std::set<std::string>{"public", "hidden"}.contains(
                command.at("visibility").get<std::string>()) ||
            !std::set<std::string>{"read_only", "evidence_only", "proposal_only",
                "permission_backed_mutation", "prohibited"}.contains(
                command.at("mutability").get<std::string>()) ||
            !std::set<std::string>{"none", "target_host_permission", "ssiag"}.contains(
                command.at("authority_mode").get<std::string>()) ||
            !std::set<std::string>{"local", "target_host"}.contains(
                command.at("target_scope").get<std::string>()) ||
            (status == "retired") != command.at("grammar").is_null()) {
            throw engine::Error("administration.command.classification",
                                "command classification or grammar lifecycle is invalid", 4);
        }
        if (!command.at("deprecated_in").is_null() &&
            (!command.at("deprecated_in").is_string() ||
             !version(command.at("deprecated_in").get_ref<const std::string&>()))) {
            throw engine::Error("administration.command.lifecycle",
                                "deprecated version is invalid", 4);
        }
        const bool deprecated = status == "deprecated" || status == "retired";
        if (deprecated != command.at("deprecated_in").is_string()) {
            throw engine::Error("administration.command.lifecycle",
                                "command deprecation evidence differs from status", 4);
        }
        if (!command.at("infrastructure_purpose").is_null() &&
            (!command.at("infrastructure_purpose").is_string() ||
             !printable_bounded(command.at("infrastructure_purpose")
                                   .get_ref<const std::string&>(), 4096U))) {
            throw engine::Error("administration.command.purpose",
                                "infrastructure purpose is invalid", 4);
        }
        if (!command.at("recovery_command_id").is_null() &&
            (!command.at("recovery_command_id").is_string() ||
             !safe_prefixed_id(command.at("recovery_command_id").get_ref<const std::string&>(),
                               "qxcmd:"))) {
            throw engine::Error("administration.command.recovery",
                                "recovery command identity is invalid", 4);
        }
        for (const auto* field : {"replacement_ids", "backend_operation_ids"}) {
            static_cast<void>(strict_string_array(command.at(field), 256U,
                "administration.command.array", stable));
        }
        for (const auto* field : {"input_protocols", "output_protocols",
                                  "result_validation_protocols"}) {
            static_cast<void>(strict_string_array(command.at(field), 64U,
                "administration.command.protocol", stable));
        }
        static_cast<void>(strict_string_array(command.at("aliases"), 32U,
            "administration.command.alias", printable));
        const auto replacements = strict_string_array(command.at("replacement_ids"), 32U,
            "administration.command.replacements", [](std::string_view item) {
                return safe_prefixed_id(item, "qxcmd:");
            });
        if (status == "deprecated" && replacements.empty()) {
            throw engine::Error("administration.command.lifecycle",
                                "deprecated command has no replacement", 4);
        }
        static_cast<void>(strict_string_array(command.at("backend_operation_ids"), 256U,
            "administration.command.operations", [](std::string_view item) {
                return safe_prefixed_id(item, "engop:");
            }));
        if (!command.at("feature_bindings").is_array() ||
            command.at("feature_bindings").size() > 256U ||
            !command.at("noninteractive").is_boolean() ||
            !command.at("json_output").is_boolean()) {
            throw engine::Error("administration.command.binding",
                                "command binding or machine behavior is invalid", 4);
        }
        std::string previous_binding;
        for (const auto& binding : command.at("feature_bindings")) {
            require_exact_fields(binding, std::set<std::string>{"feature_id", "interaction"},
                                 "administration.command.binding_fields");
            const auto feature_id = require_string(binding, "feature_id", 256U);
            const auto interaction = require_string(binding, "interaction", 32U, true);
            const auto key = feature_id + "\n" + interaction;
            if (!safe_feature_id(feature_id) ||
                !std::set<std::string>{"discover", "inspect", "query", "validate", "configure",
                    "propose", "invoke", "apply", "lifecycle", "recover"}.contains(interaction) ||
                (!previous_binding.empty() && key <= previous_binding)) {
                throw engine::Error("administration.command.binding",
                                    "command feature binding is invalid or unsorted", 4);
            }
            previous_binding = key;
        }
        const bool has_bindings = !command.at("feature_bindings").empty();
        const bool has_purpose = command.at("infrastructure_purpose").is_string();
        if (has_bindings == has_purpose) {
            throw engine::Error("administration.command.purpose",
                                "command must bind features or one infrastructure purpose", 4);
        }
        result.commands.emplace(command_id, command);
    }
    for (const auto& [command_id, command] : result.commands) {
        static_cast<void>(command_id);
        for (const auto& replacement : command.at("replacement_ids")) {
            if (!result.commands.contains(replacement.get<std::string>())) {
                throw engine::Error("administration.command.lifecycle",
                                    "command replacement identity is absent", 4);
            }
        }
        if (command.at("recovery_command_id").is_string() &&
            !result.commands.contains(command.at("recovery_command_id").get<std::string>())) {
            throw engine::Error("administration.command.recovery",
                                "recovery command identity is absent", 4);
        }
    }
    return result;
}

struct AdministrationExpectation final {
    std::string feature_id;
    std::string interaction;
    std::string requirement;
    std::string delivery;
    std::vector<std::string> command_ids;
    std::vector<std::string> engine_operation_ids;
    std::optional<std::string> inherited_from;
};

struct ParsedProfile final {
    std::string digest;
    std::string registry_digest;
    std::string forward_gate;
    std::map<std::string, std::vector<AdministrationExpectation>> features;
};

const AdministrationExpectation& resolve_expectation(
    const ParsedProfile& profile, const AdministrationExpectation& expectation) {
    const AdministrationExpectation* resolved = &expectation;
    while (resolved->inherited_from) {
        const auto parent = profile.features.find(*resolved->inherited_from);
        if (parent == profile.features.end()) {
            throw engine::Error("administration.profile.inheritance",
                                "administration inheritance target is absent", 4);
        }
        const auto inherited = std::find_if(
            parent->second.begin(), parent->second.end(),
            [&](const AdministrationExpectation& candidate) {
                return candidate.interaction == expectation.interaction &&
                       candidate.requirement == expectation.requirement;
            });
        if (inherited == parent->second.end()) {
            throw engine::Error("administration.profile.inheritance",
                                "inherited expectation classification differs", 4);
        }
        resolved = &*inherited;
    }
    return *resolved;
}

ParsedProfile parse_administration_profile(const engine::Json& value,
                                           const ParsedSnapshot& snapshot) {
    require_exact_fields(value, std::set<std::string>{
        "protocol", "format_version", "profile_id", "ssfv_registry_digest",
        "catalog_scope", "catalog_complete", "registered_feature_count", "forward_gate",
        "features", "profile_digest",
    }, "administration.profile.field_set");
    if (value.at("protocol") != "symphony.knowledge.feature-administration-profile.v1" ||
        value.at("format_version") != 1 ||
        value.at("catalog_scope") != "registered_partial_catalog" ||
        value.at("catalog_complete") != false ||
        !value.at("registered_feature_count").is_number_unsigned() ||
        value.at("registered_feature_count").get<std::size_t>() != snapshot.records.size() ||
        !value.at("features").is_array() || value.at("features").size() > max_feature_records) {
        throw engine::Error("administration.profile.identity",
                            "administration profile identity or catalog binding is invalid", 4);
    }
    static_cast<void>(require_string(value, "profile_id", 256U, true));
    const auto registry_digest = require_string(value, "ssfv_registry_digest", 71U);
    const auto forward_gate = require_string(value, "forward_gate", 32U, true);
    if (!tagged_digest(registry_digest) ||
        !std::set<std::string>{"report_only", "enforce_new_records", "enforce_all_records"}
             .contains(forward_gate)) {
        throw engine::Error("administration.profile.identity",
                            "administration profile digest or forward gate is invalid", 4);
    }
    verify_self_digest(value, "profile_digest", "administration.profile.digest");
    ParsedProfile result{value.at("profile_digest").get<std::string>(),
                         registry_digest, forward_gate, {}};
    std::string previous_feature;
    static const std::set<std::string> interactions = {
        "discover", "inspect", "query", "validate", "configure",
        "propose", "invoke", "apply", "lifecycle", "recover",
    };
    static const std::set<std::string> requirements = {
        "required", "optional", "prohibited", "not_applicable",
    };
    static const std::set<std::string> deliveries = {
        "direct", "composed", "delegated", "lifecycle_only", "observation_only",
        "runtime_only", "system_orchestrated", "none", "unreviewed",
    };
    for (const auto& feature : value.at("features")) {
        require_exact_fields(feature, std::set<std::string>{"feature_id", "expectations"},
                             "administration.profile.feature_fields");
        const auto feature_id = require_string(feature, "feature_id", 256U);
        if (!safe_feature_id(feature_id) || !snapshot.records.contains(feature_id) ||
            (!previous_feature.empty() && feature_id <= previous_feature) ||
            !feature.at("expectations").is_array() ||
            feature.at("expectations").size() > 10U) {
            throw engine::Error("administration.profile.feature",
                                "profile feature is invalid, absent, or unsorted", 4);
        }
        previous_feature = feature_id;
        std::string previous_interaction;
        auto& expectations = result.features[feature_id];
        for (const auto& expectation : feature.at("expectations")) {
            require_exact_fields(expectation, std::set<std::string>{
                "interaction", "requirement", "delivery", "command_ids",
                "engine_operation_ids", "inherited_from_feature_id", "rationale", "evidence",
            }, "administration.profile.expectation_fields");
            AdministrationExpectation parsed;
            parsed.feature_id = feature_id;
            parsed.interaction = require_string(expectation, "interaction", 32U, true);
            parsed.requirement = require_string(expectation, "requirement", 32U, true);
            parsed.delivery = require_string(expectation, "delivery", 32U, true);
            if (!interactions.contains(parsed.interaction) ||
                !requirements.contains(parsed.requirement) ||
                !deliveries.contains(parsed.delivery) ||
                (!previous_interaction.empty() && parsed.interaction <= previous_interaction)) {
                throw engine::Error("administration.profile.expectation",
                                    "profile expectation classification is invalid or unsorted", 4);
            }
            previous_interaction = parsed.interaction;
            parsed.command_ids = strict_string_array(expectation.at("command_ids"), 256U,
                "administration.profile.command_ids", [](std::string_view item) {
                    return safe_prefixed_id(item, "qxcmd:");
                });
            parsed.engine_operation_ids = strict_string_array(
                expectation.at("engine_operation_ids"), 256U,
                "administration.profile.engine_operation_ids", [](std::string_view item) {
                    return safe_prefixed_id(item, "engop:");
                });
            if (!expectation.at("inherited_from_feature_id").is_null()) {
                if (!expectation.at("inherited_from_feature_id").is_string() ||
                    !safe_feature_id(expectation.at("inherited_from_feature_id")
                                         .get_ref<const std::string&>())) {
                    throw engine::Error("administration.profile.inheritance",
                                        "inherited feature identity is invalid", 4);
                }
                parsed.inherited_from =
                    expectation.at("inherited_from_feature_id").get<std::string>();
            }
            const auto rationale = require_string(expectation, "rationale", 4096U);
            static_cast<void>(rationale);
            static_cast<void>(strict_string_array(expectation.at("evidence"), 256U,
                "administration.profile.evidence", [](std::string_view item) {
                    return printable_bounded(item, 4096U);
                }));
            const bool terminal = parsed.requirement == "prohibited" ||
                                  parsed.requirement == "not_applicable";
            if (terminal && (parsed.delivery != "none" || !parsed.command_ids.empty() ||
                             !parsed.engine_operation_ids.empty() || parsed.inherited_from)) {
                throw engine::Error("administration.profile.disposition",
                                    "prohibited or not-applicable expectation has delivery", 4);
            }
            if (parsed.requirement == "required" && parsed.delivery == "none") {
                throw engine::Error("administration.profile.disposition",
                                    "required expectation is unreviewed or has no delivery", 4);
            }
            if (parsed.delivery == "unreviewed" &&
                (!parsed.command_ids.empty() || !parsed.engine_operation_ids.empty() ||
                 parsed.inherited_from)) {
                throw engine::Error("administration.profile.disposition",
                                    "unreviewed expectation contains resolved delivery", 4);
            }
            expectations.push_back(std::move(parsed));
        }
    }
    for (const auto& [feature_id, expectations] : result.features) {
        if (expectations.empty() && forward_gate != "report_only") {
            throw engine::Error("administration.profile.forward_gate",
                                "enforced profile contains an unreviewed feature", 4);
        }
        for (const auto& expectation : expectations) {
            if (expectation.delivery == "unreviewed" && forward_gate != "report_only") {
                throw engine::Error("administration.profile.forward_gate",
                                    "enforced profile contains an unreviewed delivery", 4);
            }
            if (!expectation.inherited_from) {
                continue;
            }
            std::set<std::string> visited{feature_id};
            auto parent_id = *expectation.inherited_from;
            while (true) {
                if (!visited.insert(parent_id).second) {
                    throw engine::Error("administration.profile.inheritance",
                                        "administration inheritance contains a cycle", 4);
                }
                const auto parent = result.features.find(parent_id);
                if (parent == result.features.end()) {
                    throw engine::Error("administration.profile.inheritance",
                                        "administration inheritance target is absent", 4);
                }
                const auto inherited = std::find_if(
                    parent->second.begin(), parent->second.end(),
                    [&](const AdministrationExpectation& candidate) {
                        return candidate.interaction == expectation.interaction &&
                               candidate.requirement == expectation.requirement;
                    });
                if (inherited == parent->second.end()) {
                    throw engine::Error("administration.profile.inheritance",
                                        "inherited expectation classification differs", 4);
                }
                if (!inherited->inherited_from) {
                    break;
                }
                parent_id = *inherited->inherited_from;
            }
        }
    }
    if (result.features.size() != snapshot.records.size()) {
        throw engine::Error("administration.profile.catalog_set",
                            "profile feature set differs from the semantic snapshot", 4);
    }
    return result;
}

struct ParsedEngineDescriptor final {
    std::string module_id;
    std::string engine_id;
    std::string engine_version;
    std::string digest;
    std::vector<engine::Json> operations;
};

ParsedEngineDescriptor parse_engine_descriptor(const engine::Json& value) {
    require_exact_fields(value, std::set<std::string>{
        "protocol", "format_version", "module_id", "engine_id", "vector_id",
        "engine_version", "process_protocols", "contract_versions", "operations", "limits",
        "supported_scopes", "language", "thermal_path", "canonical_apply_enabled",
        "session_mutation_enabled", "network_listener", "descriptor_digest",
    }, "administration.engine_descriptor.field_set");
    if (value.at("protocol") != engine::descriptor_protocol_v2 ||
        value.at("format_version") != 2 || !value.at("operations").is_array() ||
        value.at("operations").empty() || value.at("operations").size() > 1024U ||
        !value.at("thermal_path").is_string() ||
        !std::set<std::string>{"freezing", "warm", "hot"}.contains(
            value.at("thermal_path").get<std::string>())) {
        throw engine::Error("administration.engine_descriptor.identity",
                            "engine descriptor identity is invalid", 4);
    }
    ParsedEngineDescriptor result{
        require_string(value, "module_id", 256U, true),
        require_string(value, "engine_id", 256U, true),
        require_string(value, "engine_version", 64U, true), "", {},
    };
    if ((!value.at("vector_id").is_null() &&
         (!value.at("vector_id").is_string() ||
          !safe_token(value.at("vector_id").get_ref<const std::string&>(), 256U))) ||
        !value.at("language").is_string() ||
        !safe_language(value.at("language").get_ref<const std::string&>()) ||
        !value.at("canonical_apply_enabled").is_boolean() ||
        !value.at("session_mutation_enabled").is_boolean() ||
        !value.at("network_listener").is_boolean()) {
        throw engine::Error("administration.engine_descriptor.field",
                            "engine descriptor field is invalid", 4);
    }
    const auto validate_array = [](const engine::Json& array, std::size_t minimum,
                                   std::size_t maximum, std::size_t string_limit,
                                   bool tokens, const std::string& code) {
        if (!array.is_array() || array.size() < minimum || array.size() > maximum) {
            throw engine::Error(code, "descriptor array cardinality is invalid", 4);
        }
        std::set<std::string> values;
        for (const auto& item : array) {
            if (!item.is_string() ||
                (tokens && !safe_token(item.get_ref<const std::string&>(), string_limit)) ||
                (!tokens && !printable_bounded(item.get_ref<const std::string&>(), string_limit)) ||
                !values.insert(item.get<std::string>()).second) {
                throw engine::Error(code, "descriptor array value is invalid or duplicated", 4);
            }
        }
    };
    validate_array(value.at("process_protocols"), 1U, 16U, 256U, true,
                   "administration.engine_descriptor.process_protocols");
    validate_array(value.at("contract_versions"), 1U, 64U, 256U, false,
                   "administration.engine_descriptor.contract_versions");
    validate_array(value.at("supported_scopes"), 1U, 3U, 16U, true,
                   "administration.engine_descriptor.supported_scopes");
    for (const auto& scope : value.at("supported_scopes")) {
        if (!std::set<std::string>{"user", "system", "tops"}.contains(
                scope.get<std::string>())) {
            throw engine::Error("administration.engine_descriptor.supported_scopes",
                                "descriptor scope is invalid", 4);
        }
    }
    require_exact_fields(value.at("limits"), std::set<std::string>{
        "request_bytes", "response_bytes", "json_depth", "json_values", "path_bytes",
        "snapshot_files", "snapshot_file_bytes", "deadline_ahead_ms",
    }, "administration.engine_descriptor.limit_fields");
    for (const auto& [name, limit] : value.at("limits").items()) {
        static_cast<void>(name);
        const bool positive = limit.is_number_unsigned()
            ? limit.get<std::uint64_t>() > 0U
            : limit.is_number_integer() && limit.get<std::int64_t>() > 0;
        if (!positive) {
            throw engine::Error("administration.engine_descriptor.limit",
                                "descriptor limit is not a positive integer", 4);
        }
    }
    verify_self_digest(value, "descriptor_digest", "administration.engine_descriptor.digest");
    result.digest = value.at("descriptor_digest").get<std::string>();
    std::set<std::string> operation_ids;
    std::set<std::string> operation_names;
    for (const auto& operation : value.at("operations")) {
        require_exact_fields(operation, std::set<std::string>{
            "engine_operation_id", "operation_name", "availability", "feature_ids",
            "administrative_interactions", "administration_disposition", "input_protocol",
            "output_protocol", "mutability", "idempotency", "expected_state_required",
            "authorization_requirement", "recovery_operation_id", "direct_invocation",
            "thermal_path",
        }, "administration.engine_operation.field_set");
        const auto operation_id = require_string(operation, "engine_operation_id", 256U);
        if (!safe_prefixed_id(operation_id, "engop:") ||
            !operation_ids.insert(operation_id).second ||
            !operation.at("feature_ids").is_array() ||
            operation.at("feature_ids").size() > 256U ||
            !operation.at("administrative_interactions").is_array() ||
            operation.at("administrative_interactions").empty() ||
            operation.at("administrative_interactions").size() > 10U) {
            throw engine::Error("administration.engine_operation.identity",
                                "engine operation identity or cardinality is invalid", 4);
        }
        std::set<std::string> feature_ids;
        for (const auto& feature_id : operation.at("feature_ids")) {
            if (!feature_id.is_string() ||
                !safe_feature_id(feature_id.get_ref<const std::string&>()) ||
                !feature_ids.insert(feature_id.get<std::string>()).second) {
                throw engine::Error("administration.engine_operation.feature",
                                    "engine operation feature identity is invalid or duplicated", 4);
            }
        }
        for (const auto* field : {"operation_name", "availability", "administration_disposition",
                                  "mutability", "idempotency",
                                  "authorization_requirement", "direct_invocation", "thermal_path"}) {
            static_cast<void>(require_string(operation, field, 256U, true));
        }
        if (!std::set<std::string>{"implemented", "reserved", "disabled"}.contains(
                operation.at("availability").get<std::string>()) ||
            !std::set<std::string>{"unreviewed", "qxctl_required", "lifecycle_only",
                "runtime_only", "system_orchestrated", "prohibited", "not_applicable"}.contains(
                operation.at("administration_disposition").get<std::string>()) ||
            !std::set<std::string>{"read_only", "evidence_only", "proposal_only",
                "permission_backed_mutation", "prohibited"}.contains(
                operation.at("mutability").get<std::string>()) ||
            !std::set<std::string>{"not_applicable", "idempotent",
                "idempotent_with_invocation_id", "non_idempotent"}.contains(
                operation.at("idempotency").get<std::string>()) ||
            !std::set<std::string>{"none", "target_host_permission", "ssiag"}.contains(
                operation.at("authorization_requirement").get<std::string>()) ||
            !std::set<std::string>{"supported", "diagnostic_only", "prohibited"}.contains(
                operation.at("direct_invocation").get<std::string>()) ||
            !std::set<std::string>{"freezing", "warm", "hot"}.contains(
                operation.at("thermal_path").get<std::string>())) {
            throw engine::Error("administration.engine_operation.field",
                                "engine operation classification is invalid", 4);
        }
        const auto operation_name = operation.at("operation_name").get<std::string>();
        if (!operation_names.insert(operation_name).second ||
            (operation.at("availability") == "implemented" &&
             (operation.at("administration_disposition") == "prohibited" ||
              operation.at("mutability") == "prohibited"))) {
            throw engine::Error("administration.engine_operation.field",
                                "engine operation identity or prohibition is invalid", 4);
        }
        for (const auto* field : {"input_protocol", "output_protocol"}) {
            if (!operation.at(field).is_null() &&
                (!operation.at(field).is_string() ||
                 !safe_token(operation.at(field).get_ref<const std::string&>(), 256U))) {
                throw engine::Error("administration.engine_operation.field",
                                    "engine operation protocol identity is invalid", 4);
            }
        }
        if (!operation.at("expected_state_required").is_boolean() ||
            (!operation.at("recovery_operation_id").is_null() &&
             (!operation.at("recovery_operation_id").is_string() ||
              !safe_prefixed_id(operation.at("recovery_operation_id")
                                    .get_ref<const std::string&>(), "engop:")))) {
            throw engine::Error("administration.engine_operation.field",
                                "engine operation field is invalid", 4);
        }
        std::set<std::string> interactions;
        for (const auto& interaction : operation.at("administrative_interactions")) {
            if (!interaction.is_string() ||
                !std::set<std::string>{"discover", "inspect", "query", "validate", "configure",
                    "propose", "invoke", "apply", "lifecycle", "recover"}
                     .contains(interaction.get<std::string>()) ||
                !interactions.insert(interaction.get<std::string>()).second) {
                throw engine::Error("administration.engine_operation.field",
                                    "administrative interaction is invalid", 4);
            }
        }
        result.operations.push_back(operation);
    }
    for (const auto& operation : value.at("operations")) {
        if (operation.at("recovery_operation_id").is_string() &&
            !operation_ids.contains(operation.at("recovery_operation_id").get<std::string>())) {
            throw engine::Error("administration.engine_operation.recovery",
                                "engine recovery operation identity is absent", 4);
        }
    }
    return result;
}

bool command_binds(const engine::Json& command, const std::string& feature_id,
                   const std::string& interaction) {
    return std::any_of(command.at("feature_bindings").begin(),
                       command.at("feature_bindings").end(),
                       [&](const engine::Json& binding) {
        return binding.at("feature_id") == feature_id &&
               binding.at("interaction") == interaction;
    });
}

bool command_targets_operation(const engine::Json& command, const std::string& operation_id) {
    return std::find(command.at("backend_operation_ids").begin(),
                     command.at("backend_operation_ids").end(),
                     engine::Json(operation_id)) != command.at("backend_operation_ids").end();
}

bool observed_command_compatible(const engine::Json& expected,
                                 const engine::Json& observed,
                                 const std::string& feature_id,
                                 const std::string& interaction) {
    if (observed.at("status") == "retired" ||
        observed.at("mutability") != expected.at("mutability") ||
        observed.at("authority_mode") != expected.at("authority_mode") ||
        observed.at("target_scope") != expected.at("target_scope") ||
        (expected.at("noninteractive") == true && observed.at("noninteractive") != true) ||
        (expected.at("json_output") == true && observed.at("json_output") != true) ||
        !command_binds(observed, feature_id, interaction)) {
        return false;
    }
    for (const auto* field : {"backend_operation_ids", "input_protocols", "output_protocols",
                              "result_validation_protocols"}) {
        for (const auto& required : expected.at(field)) {
            if (std::find(observed.at(field).begin(), observed.at(field).end(), required) ==
                observed.at(field).end()) {
                return false;
            }
        }
    }
    return true;
}

engine::Json administration_check(const engine::Json& payload,
                                  std::int64_t deadline_unix_ms) {
    require_exact_fields(payload, std::set<std::string>{
        "protocol", "format_version", "semantic_snapshot", "profile",
        "expected_command_registry", "observed_qxctl_state", "observed_command_registry",
        "engine_descriptors", "requested_feature_id",
    });
    if (payload.at("protocol") != "symphony.knowledge.administration-coverage-input.v1" ||
        payload.at("format_version") != 1 || !payload.at("engine_descriptors").is_array() ||
        payload.at("engine_descriptors").size() > 1024U) {
        throw engine::Error("administration.input.identity",
                            "administration coverage input identity or cardinality is invalid", 4);
    }
    require_deadline(deadline_unix_ms);
    const auto snapshot = parse_snapshot(payload.at("semantic_snapshot"));
    const auto profile = parse_administration_profile(payload.at("profile"), snapshot);
    if (payload.at("expected_command_registry").is_null()) {
        throw engine::Error("administration.input.expected_registry",
                            "expected command registry is required", 4);
    }
    const auto expected =
        parse_command_registry(payload.at("expected_command_registry"), "expected");
    const auto observed_state = require_string(payload, "observed_qxctl_state", 32U, true);
    if (!std::set<std::string>{"not_evaluated", "absent", "present"}.contains(observed_state)) {
        throw engine::Error("administration.input.observed_state",
                            "observed qxctl state is invalid", 4);
    }
    std::optional<ParsedCommandRegistry> observed;
    if (observed_state == "present") {
        if (payload.at("observed_command_registry").is_null()) {
            throw engine::Error("administration.input.observed_registry",
                                "present qxctl requires an observed registry", 4);
        }
        observed = parse_command_registry(payload.at("observed_command_registry"), "observed");
    } else if (!payload.at("observed_command_registry").is_null()) {
        throw engine::Error("administration.input.observed_registry",
                            "absent or unevaluated qxctl prohibits an observed registry", 4);
    }
    std::optional<std::string> requested_feature;
    if (!payload.at("requested_feature_id").is_null()) {
        if (!payload.at("requested_feature_id").is_string() ||
            !safe_feature_id(payload.at("requested_feature_id").get_ref<const std::string&>()) ||
            !snapshot.records.contains(payload.at("requested_feature_id").get<std::string>())) {
            throw engine::Error("administration.input.requested_feature",
                                "requested feature identity is invalid or absent", 4);
        }
        requested_feature = payload.at("requested_feature_id").get<std::string>();
    }

    engine::Json surfaces = engine::Json::array();
    engine::Json feature_findings = engine::Json::array();
    engine::Json remediation = engine::Json::array();
    engine::Json module_integrations = engine::Json::array();
    engine::Json descriptor_digests = engine::Json::array();
    std::map<std::string, std::size_t> counts = {
        {"satisfied", 0U}, {"uncovered", 0U}, {"exempt", 0U},
        {"prohibited", 0U}, {"stale", 0U}, {"unresolved", 0U},
    };
    std::size_t features_checked = 0U;
    const bool profile_stale = profile.registry_digest != snapshot.registry_digest;
    const auto finding = [&](const std::string& severity, const engine::Json& feature_id,
                             const engine::Json& interaction,
                             const engine::Json& operation_id,
                             const engine::Json& command_id, const std::string& reason,
                             engine::Json missing) {
        engine::Json finding{{"severity", severity}, {"feature_id", feature_id},
            {"interaction", interaction}, {"engine_operation_id", operation_id},
            {"command_id", command_id}, {"reason", reason}, {"missing", std::move(missing)},
            {"proposal_only", true}, {"ratification_required", true}};
        finding["finding_id"] = "administration-finding:" +
            engine::sha256_hex(finding.dump()).substr(0U, 48U);
        return finding;
    };
    const auto add_remediation = [&](const engine::Json& source_finding,
                                     const std::string& feature_id,
                                     const std::string& interaction,
                                     const std::vector<std::string>& operation_ids,
                                     const std::string& mutability,
                                     const std::string& authority,
                                     engine::Json required_evidence) {
        engine::Json recipe{{"finding_ids", engine::Json::array({
                source_finding.at("finding_id")})},
            {"feature_id", feature_id}, {"interaction", interaction},
            {"backend_operation_ids", operation_ids},
            {"required_mutability", mutability}, {"required_authority_mode", authority},
            {"required_target_scope", "local"},
            {"required_evidence", std::move(required_evidence)},
            {"proposed_command_id", nullptr}, {"proposed_grammar", nullptr},
            {"proposal_only", true}, {"ratification_required", true}};
        recipe["remediation_id"] = "administration-remediation:" +
            engine::sha256_hex(recipe.dump()).substr(0U, 48U);
        remediation.push_back(std::move(recipe));
    };

    for (const auto& [feature_id, record] : snapshot.records) {
        require_deadline(deadline_unix_ms);
        static_cast<void>(record);
        if (requested_feature && *requested_feature != feature_id) {
            continue;
        }
        ++features_checked;
        const auto feature = profile.features.find(feature_id);
        if (feature == profile.features.end() || feature->second.empty()) {
            ++counts.at("unresolved");
            feature_findings.push_back(finding("warning", feature_id, nullptr, nullptr,
                nullptr, "registered feature has no explicit administration expectation",
                engine::Json::array({"administration_expectation"})));
            continue;
        }
        for (const auto& expectation : feature->second) {
            const auto& delivery_expectation = resolve_expectation(profile, expectation);
            engine::Json surface_findings = engine::Json::array();
            std::string design_state = profile_stale ? "stale" : "satisfied";
            const bool exceptional =
                std::set<std::string>{"lifecycle_only", "observation_only", "runtime_only",
                                      "system_orchestrated"}
                    .contains(expectation.delivery);
            if (!profile_stale && (expectation.delivery == "unreviewed" ||
                                   delivery_expectation.delivery == "unreviewed")) {
                design_state = "unresolved";
                surface_findings.push_back(finding("warning", feature_id,
                    expectation.interaction, nullptr, nullptr,
                    "known interaction has no reviewed administration delivery",
                    engine::Json::array({"administration_expectation"})));
            } else if (!profile_stale && expectation.requirement == "prohibited") {
                design_state = "prohibited";
            } else if (!profile_stale &&
                       (expectation.requirement == "not_applicable" || exceptional)) {
                design_state = "exempt";
            } else if (!profile_stale && expectation.requirement == "optional" &&
                       delivery_expectation.command_ids.empty()) {
                design_state = "exempt";
            } else if (!profile_stale && expectation.requirement == "required" &&
                       !exceptional && !expectation.inherited_from &&
                       delivery_expectation.command_ids.empty()) {
                design_state = "uncovered";
                auto gap = finding("violation", feature_id, expectation.interaction,
                    delivery_expectation.engine_operation_ids.empty()
                        ? engine::Json(nullptr)
                        : engine::Json(delivery_expectation.engine_operation_ids.front()),
                    nullptr, "required interaction has no expected qxctl command",
                    engine::Json::array({"qxctl_command"}));
                surface_findings.push_back(gap);
                add_remediation(gap, feature_id, expectation.interaction,
                    delivery_expectation.engine_operation_ids, "read_only", "none",
                    engine::Json::array({"backend_binding", "cobra_leaf", "command_spec",
                        "feature_binding", "implementation_test", "json_output",
                        "noninteractive_support", "result_validator"}));
            }
            for (const auto& command_id : delivery_expectation.command_ids) {
                const auto command = expected.commands.find(command_id);
                if (command == expected.commands.end() ||
                    command->second.at("status") == "retired" ||
                    command->second.at("mutability") == "prohibited" ||
                    !command_binds(command->second, feature_id, expectation.interaction) ||
                    std::any_of(delivery_expectation.engine_operation_ids.begin(),
                                delivery_expectation.engine_operation_ids.end(),
                                [&](const std::string& operation_id) {
                        return !command_targets_operation(command->second, operation_id);
                    })) {
                    design_state = profile_stale ? "stale" : "uncovered";
                    auto gap = finding("violation", feature_id, expectation.interaction,
                        nullptr, command_id,
                        "expected command is absent or lacks the exact feature binding",
                        engine::Json::array({"feature_registration", "qxctl_command"}));
                    surface_findings.push_back(gap);
                    add_remediation(gap, feature_id, expectation.interaction,
                        delivery_expectation.engine_operation_ids, "read_only", "none",
                        engine::Json::array({"backend_binding", "cobra_leaf", "command_spec",
                            "feature_binding", "implementation_test", "result_validator"}));
                }
            }
            std::string live_state = "not_evaluated";
            if (observed_state == "absent") {
                live_state = "qxctl_absent";
            } else if (observed_state == "present") {
                live_state = "ready";
                for (const auto& command_id : delivery_expectation.command_ids) {
                    const auto expected_command = expected.commands.find(command_id);
                    const auto observed_command = observed->commands.find(command_id);
                    if (expected_command == expected.commands.end() ||
                        observed_command == observed->commands.end() ||
                        !observed_command_compatible(expected_command->second,
                                                     observed_command->second,
                                                     feature_id, expectation.interaction)) {
                        live_state = "incompatible";
                        surface_findings.push_back(finding("warning", feature_id,
                            expectation.interaction, nullptr, command_id,
                            "observed qxctl command is semantically incompatible with expected coverage",
                            engine::Json::array({"implementation_evidence"})));
                        break;
                    }
                }
            }
            ++counts.at(design_state);
            surfaces.push_back(engine::Json{
                {"feature_id", feature_id}, {"interaction", expectation.interaction},
                {"design_state", design_state}, {"live_state", live_state},
                {"authorization_state", "not_evaluated"},
                {"command_ids", delivery_expectation.command_ids},
                {"engine_operation_ids", delivery_expectation.engine_operation_ids},
                {"findings", std::move(surface_findings)},
            });
        }
    }

    std::vector<engine::Json> ordered_descriptors;
    std::set<std::string> supplied_descriptor_digests;
    std::set<std::string> supplied_engine_identities;
    for (const auto& raw_descriptor : payload.at("engine_descriptors")) {
        if (raw_descriptor.is_object() && raw_descriptor.contains("descriptor_digest") &&
            raw_descriptor.at("descriptor_digest").is_string() &&
            !supplied_descriptor_digests.insert(
                raw_descriptor.at("descriptor_digest").get<std::string>()).second) {
            throw engine::Error("administration.engine_descriptor.duplicate_digest",
                                "engine descriptor digest is duplicated", 4);
        }
        if (raw_descriptor.is_object() && raw_descriptor.contains("module_id") &&
            raw_descriptor.at("module_id").is_string() && raw_descriptor.contains("engine_id") &&
            raw_descriptor.at("engine_id").is_string()) {
            const auto identity = raw_descriptor.at("module_id").get<std::string>() + "\n" +
                                  raw_descriptor.at("engine_id").get<std::string>();
            if (!supplied_engine_identities.insert(identity).second) {
                throw engine::Error("administration.engine_descriptor.duplicate_identity",
                                    "module and engine descriptor identity is duplicated", 4);
            }
        }
        ordered_descriptors.push_back(raw_descriptor);
    }
    std::sort(ordered_descriptors.begin(), ordered_descriptors.end(),
              [](const engine::Json& left, const engine::Json& right) {
        return left.dump() < right.dump();
    });
    for (const auto& raw_descriptor : ordered_descriptors) {
        require_deadline(deadline_unix_ms);
        try {
            const auto descriptor = parse_engine_descriptor(raw_descriptor);
            descriptor_digests.push_back(descriptor.digest);
            std::string state = "integration_ready";
            engine::Json module_findings = engine::Json::array();
            if (profile_stale) {
                state = "blocked_incompatible";
                module_findings.push_back(finding("violation", nullptr, nullptr, nullptr,
                    nullptr, "engine descriptor is assessed against a stale administration profile",
                    engine::Json::array({"administration_expectation"})));
            }
            for (const auto& operation : descriptor.operations) {
                if (operation.at("availability") != "implemented") {
                    continue;
                }
                const auto operation_id = operation.at("engine_operation_id").get<std::string>();
                const bool semantic_identity_missing = operation.at("feature_ids").empty() ||
                    std::any_of(operation.at("feature_ids").begin(),
                                operation.at("feature_ids").end(),
                                [&](const engine::Json& feature_id) {
                        return !snapshot.records.contains(feature_id.get<std::string>());
                    });
                if (semantic_identity_missing) {
                    state = "semantic_registration_required";
                    const auto feature_id = operation.at("feature_ids").empty()
                        ? engine::Json(nullptr) : operation.at("feature_ids").front();
                    module_findings.push_back(finding("violation", feature_id, nullptr,
                        operation_id, nullptr,
                        "implemented engine operation has no registered SSFV feature identity",
                        engine::Json::array({"feature_registration"})));
                    continue;
                }
                const auto disposition =
                    operation.at("administration_disposition").get<std::string>();
                if (disposition == "unreviewed") {
                    if (state == "integration_ready") {
                        state = "administration_unintegrated";
                    }
                    module_findings.push_back(finding("violation",
                        operation.at("feature_ids").front(),
                        operation.at("administrative_interactions").empty()
                            ? engine::Json(nullptr)
                            : operation.at("administrative_interactions").front(),
                        operation_id, nullptr,
                        "implemented engine operation has no reviewed administration disposition",
                        engine::Json::array({"administration_expectation"})));
                    continue;
                }
                for (const auto& feature_id_value : operation.at("feature_ids")) {
                    const auto feature_id = feature_id_value.get<std::string>();
                    for (const auto& interaction_value :
                         operation.at("administrative_interactions")) {
                        const auto interaction = interaction_value.get<std::string>();
                        const auto& feature_expectations = profile.features.at(feature_id);
                        const auto expectation = std::find_if(
                            feature_expectations.begin(), feature_expectations.end(),
                            [&](const AdministrationExpectation& candidate) {
                                return candidate.interaction == interaction;
                            });
                        if (expectation == feature_expectations.end() ||
                            expectation->delivery == "unreviewed") {
                            if (state == "integration_ready") {
                                state = "administration_unintegrated";
                            }
                            module_findings.push_back(finding("violation", feature_id,
                                interaction, operation_id, nullptr,
                                "implemented engine operation has no reviewed feature administration expectation",
                                engine::Json::array({"administration_expectation"})));
                            continue;
                        }
                        const auto& resolved = resolve_expectation(profile, *expectation);
                        if (resolved.delivery == "unreviewed") {
                            if (state == "integration_ready") {
                                state = "administration_unintegrated";
                            }
                            module_findings.push_back(finding("violation", feature_id,
                                interaction, operation_id, nullptr,
                                "inherited engine administration expectation is unreviewed",
                                engine::Json::array({"administration_expectation"})));
                            continue;
                        }
                        if (disposition != "qxctl_required") {
                            const std::map<std::string, std::string> disposition_delivery = {
                                {"lifecycle_only", "lifecycle_only"},
                                {"runtime_only", "runtime_only"},
                                {"system_orchestrated", "system_orchestrated"},
                                {"prohibited", "none"},
                                {"not_applicable", "none"},
                            };
                            const auto compatible = disposition_delivery.find(disposition);
                            const bool terminal_compatible = disposition == "prohibited"
                                ? expectation->requirement == "prohibited"
                                : disposition != "not_applicable" ||
                                  expectation->requirement == "not_applicable";
                            if (compatible == disposition_delivery.end() ||
                                expectation->delivery != compatible->second ||
                                !terminal_compatible) {
                                if (state != "semantic_registration_required") {
                                    state = "blocked_incompatible";
                                }
                                module_findings.push_back(finding("violation", feature_id,
                                    interaction, operation_id, nullptr,
                                    "engine administration disposition conflicts with the feature profile",
                                    engine::Json::array({"administration_expectation"})));
                            }
                            continue;
                        }
                        bool mapped = false;
                        const bool operation_declared = std::find(
                            resolved.engine_operation_ids.begin(),
                            resolved.engine_operation_ids.end(), operation_id) !=
                            resolved.engine_operation_ids.end();
                        for (const auto& command_id : resolved.command_ids) {
                            const auto command = expected.commands.find(command_id);
                            if (command == expected.commands.end()) {
                                continue;
                            }
                            if (operation_declared && command->second.at("status") != "retired" &&
                                command->second.at("mutability") != "prohibited" &&
                                command_targets_operation(command->second, operation_id) &&
                                command_binds(command->second, feature_id, interaction)) {
                                mapped = true;
                                break;
                            }
                        }
                        if (mapped) {
                            continue;
                        }
                        if (state == "integration_ready") {
                            state = "administration_unintegrated";
                        }
                        auto gap = finding("violation", feature_id, interaction,
                            operation_id, nullptr,
                            "qxctl-required engine operation pair has no expected command binding",
                            engine::Json::array({"qxctl_command"}));
                        module_findings.push_back(gap);
                        add_remediation(gap,
                            feature_id, interaction,
                            {operation_id}, operation.at("mutability").get<std::string>(),
                            operation.at("authorization_requirement").get<std::string>(),
                            engine::Json::array({"backend_binding", "cobra_leaf", "command_spec",
                                "feature_binding", "implementation_test", "result_validator"}));
                    }
                }
            }
            module_integrations.push_back(engine::Json{
                {"module_id", descriptor.module_id}, {"engine_id", descriptor.engine_id},
                {"descriptor_digest", descriptor.digest}, {"integration_state", state},
                {"docking_ready", state == "integration_ready"},
                {"findings", std::move(module_findings)},
            });
        } catch (const engine::Error& error) {
            module_integrations.push_back(engine::Json{
                {"module_id", "unavailable"}, {"engine_id", nullptr},
                {"descriptor_digest", nullptr}, {"integration_state", "descriptor_invalid"},
                {"docking_ready", false}, {"findings", engine::Json::array({
                    finding("violation", nullptr, nullptr, nullptr, nullptr, error.what(),
                            engine::Json::array({"implementation_evidence"}))})},
            });
        } catch (const std::exception&) {
            module_integrations.push_back(engine::Json{
                {"module_id", "unavailable"}, {"engine_id", nullptr},
                {"descriptor_digest", nullptr}, {"integration_state", "descriptor_invalid"},
                {"docking_ready", false}, {"findings", engine::Json::array({
                    finding("violation", nullptr, nullptr, nullptr, nullptr,
                            "engine descriptor failed bounded validation",
                            engine::Json::array({"implementation_evidence"}))})},
            });
        }
    }
    std::sort(descriptor_digests.begin(), descriptor_digests.end());
    std::sort(module_integrations.begin(), module_integrations.end(),
              [](const engine::Json& left, const engine::Json& right) {
        const auto left_key = left.at("module_id").get<std::string>() + "\n" +
            (left.at("engine_id").is_string()
                ? left.at("engine_id").get<std::string>() : std::string{});
        const auto right_key = right.at("module_id").get<std::string>() + "\n" +
            (right.at("engine_id").is_string()
                ? right.at("engine_id").get<std::string>() : std::string{});
        return left_key < right_key;
    });
    engine::Json result{
        {"protocol", administration_coverage_protocol}, {"format_version", 1},
        {"semantic_snapshot_digest", snapshot.digest}, {"profile_digest", profile.digest},
        {"expected_command_registry_digest", expected.digest},
        {"observed_command_registry_digest",
            observed ? engine::Json(observed->digest) : engine::Json(nullptr)},
        {"engine_descriptor_digests", std::move(descriptor_digests)},
        {"feature_findings", std::move(feature_findings)},
        {"surfaces", std::move(surfaces)},
        {"module_integrations", std::move(module_integrations)},
        {"remediation_constraints", std::move(remediation)},
        {"summary", engine::Json{
            {"features_checked", features_checked},
            {"surfaces_checked", counts.at("satisfied") + counts.at("uncovered") +
                counts.at("exempt") + counts.at("prohibited") + counts.at("stale")},
            {"satisfied", counts.at("satisfied")}, {"uncovered", counts.at("uncovered")},
            {"exempt", counts.at("exempt")}, {"prohibited", counts.at("prohibited")},
            {"stale", counts.at("stale")}, {"unresolved", counts.at("unresolved")}}},
        {"read_only", true}, {"canonical", false},
    };
    result["result_digest"] = engine::tagged_sha256(result.dump());
    return result;
}

using OperationHandler = engine::Json (*)(const engine::Request&);

struct RegisteredOperation final {
    engine::OperationSpec spec;
    OperationHandler handler;
};

const std::vector<RegisteredOperation>& registered_operations();

const std::vector<engine::OperationSpec>& operation_specs() {
    static const std::vector<engine::OperationSpec> specs = [] {
        std::vector<engine::OperationSpec> result;
        for (const auto& operation : registered_operations()) {
            result.push_back(operation.spec);
        }
        engine::validate_operation_specs(result);
        return result;
    }();
    return specs;
}

const std::vector<RegisteredOperation>& registered_operations() {
    static const std::vector<RegisteredOperation> operations = {
        RegisteredOperation{engine::OperationSpec{
            "engop:symphony:ssfv.inspect", "inspect", "implemented", false, true,
            {"ssfv:symphony:ssfv-engine"}, {"inspect"}, "qxctl_required",
            "symphony.ssfv.inspect-input.v1", "symphony.ssfv.inspect-result.v1",
            "read_only", "idempotent", false, "none", "", "supported", "freezing",
        }, [](const engine::Request& request) { return inspect(request.payload); }},
        RegisteredOperation{engine::OperationSpec{
            "engop:symphony:ssfv.check", "check", "implemented", false, true,
            {"ssfv:symphony:ssfv-engine.catalog-integrity-snapshot"},
            {"validate"}, "qxctl_required",
            "symphony.ssfv.check-input.v2", "symphony.ssfv.check-result.v2",
            "evidence_only", "idempotent", false, "none", "", "supported", "freezing",
        }, [](const engine::Request& request) {
            return check(request.payload, request.deadline_unix_ms);
        }},
        RegisteredOperation{engine::OperationSpec{
            "engop:symphony:ssfv.diff", "diff", "implemented", false, true,
            {"ssfv:symphony:ssfv-engine.semantic-freshness-comparison"},
            {"validate"}, "qxctl_required",
            "symphony.ssfv.diff-input.v2", "symphony.ssfv.diff-result.v2",
            "evidence_only", "idempotent", false, "none", "", "supported", "freezing",
        }, [](const engine::Request& request) {
            return diff(request.payload, request.deadline_unix_ms);
        }},
        RegisteredOperation{engine::OperationSpec{
            "engop:symphony:ssfv.propose", "propose", "implemented", false, true,
            {"ssfv:symphony:ssfv-engine.catalog-change-proposal"}, {"propose"},
            "qxctl_required", "symphony.ssfv.proposal-input.v2",
            "symphony.knowledge.proposal.v1", "proposal_only", "idempotent", true,
            "target_host_permission", "", "supported", "freezing",
        }, [](const engine::Request& request) {
            return proposal(request.payload, request.deadline_unix_ms);
        }},
        RegisteredOperation{engine::OperationSpec{
            "engop:symphony:ssfv.graph", "graph", "implemented", false, true,
            {"ssfv:symphony:ssfv-engine.semantic-graph-projection"}, {"query"},
            "qxctl_required", "symphony.ssfv.graph-input.v1",
            "symphony.ssfv.graph-projection.v1", "read_only", "idempotent", false,
            "none", "", "supported", "freezing",
        }, [](const engine::Request& request) {
            return graph(request.payload, request.deadline_unix_ms);
        }},
        RegisteredOperation{engine::OperationSpec{
            "engop:symphony:ssfv.administration-check", "administration-check",
            "implemented", false, false,
            {"ssfv:symphony:ssfv-engine.administration-assurance"},
            {"validate"}, "qxctl_required", administration_coverage_input_protocol,
            administration_coverage_protocol, "evidence_only", "idempotent", false,
            "none", "", "supported", "freezing",
        }, [](const engine::Request& request) {
            return administration_check(request.payload, request.deadline_unix_ms);
        }},
        RegisteredOperation{engine::OperationSpec{
            "engop:symphony:ssfv.apply", "apply", "disabled", true, true,
            {"ssfv:symphony:ssfv-engine.catalog-change-proposal"}, {"apply"}, "prohibited",
            "symphony.ssfv.apply-input.v1", "symphony.ssfv.apply-result.v1",
            "prohibited", "not_applicable", true, "ssiag", "", "prohibited", "freezing",
        }, nullptr},
    };
    static const bool validated = [] {
        std::vector<engine::OperationSpec> specs;
        for (const auto& operation : operations) {
            specs.push_back(operation.spec);
        }
        engine::validate_operation_specs(specs);
        for (const auto& operation : operations) {
            if ((operation.spec.availability == "implemented") !=
                (operation.handler != nullptr)) {
                throw engine::Error("operation.handler_parity",
                                    "operation handler and availability differ", 5);
            }
        }
        return true;
    }();
    static_cast<void>(validated);
    return operations;
}

}

engine::Json descriptor() {
    return engine::Json{
        {"protocol", engine::descriptor_protocol_v1},
        {"module_id", module_id},
        {"engine_id", engine_id},
        {"vector_id", vector_id},
        {"engine_version", engine_version},
        {"process_protocols", engine::Json::array({engine::process_protocol_v1})},
        {"contract_versions", engine::Json::array({
            "knowledge/SPEC.md@v1", "knowledge/ssfv/SPEC.md@engine-v1",
            "symphony.ssfv.feature-file.v1", "symphony.ssfv.feature-record.v2",
            "symphony.ssfv.semantic-snapshot.v1", "symphony.ssfv.check-result.v2",
            "symphony.ssfv.diff-result.v2", "symphony.ssfv.graph-projection.v1"})},
        {"operations", engine::legacy_operation_descriptors(operation_specs())},
        {"limits", engine::Json{
            {"request_bytes", engine::Limits::max_request_bytes},
            {"response_bytes", engine::Limits::max_response_bytes},
            {"json_depth", engine::Limits::max_json_depth},
            {"json_values", engine::Limits::max_json_values},
            {"path_bytes", engine::Limits::max_path_bytes},
            {"file_bytes", engine::Limits::max_snapshot_file_bytes},
            {"total_evidence_bytes", max_total_evidence_bytes},
            {"namespace_entries", max_namespaces},
            {"feature_files", max_feature_files},
            {"feature_records", max_feature_records},
            {"graph_edges", max_graph_edges},
            {"deadline_ahead_ms", engine::Limits::max_deadline_ahead_ms}}},
        {"supported_scopes", engine::Json::array({"user"})},
        {"language", "C++26"},
        {"thermal_path", "freezing"},
        {"install_state", "installed_undocked"},
        {"default_receptor", nullptr},
        {"empty_registry_valid", true},
        {"engine_decides_feature_worthiness", false},
        {"engine_decides_semantic_truth", false},
        {"canonical_apply_enabled", false},
        {"session_mutation_enabled", false},
        {"network_listener", false},
    };
}

engine::Json descriptor_v2() {
    engine::Json result{
        {"protocol", engine::descriptor_protocol_v2},
        {"format_version", 2},
        {"module_id", module_id},
        {"engine_id", engine_id},
        {"vector_id", vector_id},
        {"engine_version", engine_version},
        {"process_protocols", engine::Json::array({engine::process_protocol_v1})},
        {"contract_versions", engine::Json::array({
            "knowledge/SPEC.md@v1", "knowledge/ssfv/SPEC.md@engine-v1",
            "symphony.ssfv.feature-file.v1", "symphony.ssfv.feature-record.v2",
            "symphony.ssfv.semantic-snapshot.v1", "symphony.ssfv.check-result.v2",
            "symphony.ssfv.diff-result.v2", "symphony.ssfv.graph-projection.v1",
            administration_coverage_protocol})},
        {"operations", engine::administration_operation_descriptors(operation_specs())},
        {"limits", engine::Json{
            {"request_bytes", engine::Limits::max_request_bytes},
            {"response_bytes", engine::Limits::max_response_bytes},
            {"json_depth", engine::Limits::max_json_depth},
            {"json_values", engine::Limits::max_json_values},
            {"path_bytes", engine::Limits::max_path_bytes},
            {"snapshot_files", engine::Limits::max_snapshot_files},
            {"snapshot_file_bytes", engine::Limits::max_snapshot_file_bytes},
            {"deadline_ahead_ms", engine::Limits::max_deadline_ahead_ms}}},
        {"supported_scopes", engine::Json::array({"user"})},
        {"language", "C++26"},
        {"thermal_path", "freezing"},
        {"canonical_apply_enabled", false},
        {"session_mutation_enabled", false},
        {"network_listener", false},
    };
    result["descriptor_digest"] = engine::tagged_sha256(result.dump());
    return result;
}

engine::Json handle_request(const engine::Request& request) {
    const auto& operations = registered_operations();
    const auto found = std::find_if(operations.begin(), operations.end(),
                                    [&](const RegisteredOperation& operation) {
        return operation.spec.operation_name == request.operation;
    });
    if (found == operations.end() || found->handler == nullptr) {
        throw engine::Error("operation.unsupported", "operation is reserved or unsupported", 4);
    }
    return found->handler(request);
}

}
