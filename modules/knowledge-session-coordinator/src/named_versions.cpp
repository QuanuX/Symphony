#include "named_versions.hpp"

#include "authority_session.hpp"

#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/limits.hpp"
#include "symphony/knowledge/engine/temporal.hpp"

#include <algorithm>
#include <array>
#include <atomic>
#include <cerrno>
#include <chrono>
#include <cstdint>
#include <cstring>
#include <fcntl.h>
#include <filesystem>
#include <iomanip>
#include <optional>
#include <set>
#include <sstream>
#include <string>
#include <string_view>
#include <sys/file.h>
#include <sys/stat.h>
#include <unistd.h>
#include <utility>
#include <vector>

namespace symphony::knowledge::session {
namespace engine = symphony::knowledge::engine;
namespace fs = std::filesystem;

namespace {

constexpr const char* command_protocol = "symphony.knowledge.named-version-command.v1";
constexpr const char* proposal_protocol = "symphony.knowledge.named-version-proposal.v1";
constexpr const char* registry_protocol = "symphony.knowledge.named-version-registry.v1";
constexpr const char* head_protocol = "symphony.knowledge.named-version-head.v1";
constexpr const char* result_protocol = "symphony.knowledge.named-version-result.v1";
constexpr std::uint64_t format_version = 1U;
constexpr std::size_t max_state_bytes = engine::Limits::max_response_bytes;
constexpr std::size_t max_versions = 4096U;
constexpr std::size_t max_aliases = 4096U;
constexpr std::size_t max_operations = 4096U;

const std::vector<std::string> required_capabilities = {
    "atomic-head-v1", "content-addressed-named-version-v1", "dual-slot-registry-v1",
    "expected-state-cas-v1", "idempotent-operation-v1", "immutable-object-v1",
    "opaque-extension-preservation-v1", "recovery-forward-v1", "ssiag-capability-binding-v1",
};

class FileDescriptor final {
public:
    explicit FileDescriptor(int value = -1) : value_(value) {}
    ~FileDescriptor() { if (value_ >= 0) static_cast<void>(::close(value_)); }
    FileDescriptor(const FileDescriptor&) = delete;
    FileDescriptor& operator=(const FileDescriptor&) = delete;
    FileDescriptor(FileDescriptor&& other) noexcept : value_(std::exchange(other.value_, -1)) {}
    FileDescriptor& operator=(FileDescriptor&& other) noexcept {
        if (this != &other) {
            if (value_ >= 0) static_cast<void>(::close(value_));
            value_ = std::exchange(other.value_, -1);
        }
        return *this;
    }
    [[nodiscard]] int get() const noexcept { return value_; }
private:
    int value_;
};

class StoreLock final {
public:
    StoreLock(FileDescriptor directory, FileDescriptor lock)
        : directory_(std::move(directory)), lock_(std::move(lock)) {}
    ~StoreLock() { if (lock_.get() >= 0) static_cast<void>(::flock(lock_.get(), LOCK_UN)); }
    StoreLock(const StoreLock&) = delete;
    StoreLock& operator=(const StoreLock&) = delete;
    StoreLock(StoreLock&&) = default;
    StoreLock& operator=(StoreLock&&) = default;
    [[nodiscard]] int directory_fd() const noexcept { return directory_.get(); }
private:
    FileDescriptor directory_;
    FileDescriptor lock_;
};

struct Candidate final {
    int slot = -1;
    engine::Json registry;
    bool exists = false;
    bool valid = false;
    bool incompatible = false;
};

struct State final {
    engine::Json head;
    engine::Json registry;
    bool present = false;
};

[[noreturn]] void system_error(const std::string& code, const std::string& detail) {
    const int saved = errno;
    throw engine::Error(code, detail + ": " + std::strerror(saved), 5);
}

bool tagged_digest(std::string_view value) {
    return value.size() == 71U && value.starts_with("sha256:") &&
        std::all_of(value.begin() + 7, value.end(), [](const unsigned char character) {
            return (character >= '0' && character <= '9') || (character >= 'a' && character <= 'f');
        });
}

bool safe_token(std::string_view value, std::size_t maximum = 256U) {
    if (value.empty() || value.size() > maximum) return false;
    return std::all_of(value.begin(), value.end(), [](const unsigned char character) {
        const bool alpha = (character >= 'a' && character <= 'z') ||
            (character >= 'A' && character <= 'Z');
        return alpha || (character >= '0' && character <= '9') || character == '.' ||
            character == '_' || character == ':' || character == '-' || character == '+' || character == '/';
    });
}

bool safe_alias(std::string_view value) {
    if (value.empty() || value.size() > 128U) return false;
    return std::all_of(value.begin(), value.end(), [](const unsigned char character) {
        return character >= 0x20U && character != 0x7fU;
    });
}

bool valid_uuid(std::string_view value) {
    if (value.size() != 36U || value[8] != '-' || value[13] != '-' || value[18] != '-' ||
        value[23] != '-' || value[14] < '1' || value[14] > '8' ||
        (value[19] != '8' && value[19] != '9' && value[19] != 'a' && value[19] != 'b')) return false;
    for (std::size_t index = 0; index < value.size(); ++index) {
        if (index == 8U || index == 13U || index == 18U || index == 23U) continue;
        const char character = value[index];
        if (!((character >= '0' && character <= '9') || (character >= 'a' && character <= 'f'))) return false;
    }
    return true;
}

bool safe_absolute_path(const std::string& value) {
    if (value.empty() || value.size() > engine::Limits::max_path_bytes || value == "/" ||
        value.front() != '/' || value.back() == '/' || value.find('\\') != std::string::npos ||
        value.find('\0') != std::string::npos) return false;
    const fs::path path(value);
    if (!path.is_absolute() || path.lexically_normal().string() != value) return false;
    for (const auto& item : path.relative_path()) {
        const auto component = item.string();
        if (component.empty() || component == "." || component == "..") return false;
    }
    return true;
}

void exact_fields(const engine::Json& value, const std::set<std::string>& fields,
                  const std::string& context) {
    if (!value.is_object() || value.size() != fields.size()) {
        throw engine::Error("named_version.field_set", context + " has an invalid field set", 4);
    }
    for (const auto& [key, item] : value.items()) {
        static_cast<void>(item);
        if (!fields.contains(key)) {
            throw engine::Error("named_version.unknown_field", context + " has an unknown field", 4);
        }
    }
}

std::string text(const engine::Json& object, const char* field, std::size_t maximum = 4096U) {
    if (!object.contains(field) || !object.at(field).is_string()) {
        throw engine::Error("named_version.invalid_field", std::string(field) + " must be a string", 4);
    }
    const auto value = object.at(field).get<std::string>();
    if (value.empty() || value.size() > maximum) {
        throw engine::Error("named_version.invalid_field", std::string(field) + " has invalid length", 4);
    }
    return value;
}

std::uint64_t number(const engine::Json& object, const char* field) {
    if (!object.contains(field) ||
        (!object.at(field).is_number_unsigned() && !object.at(field).is_number_integer())) {
        throw engine::Error("named_version.invalid_field", std::string(field) + " must be an integer", 4);
    }
    try {
        const auto value = object.at(field).get<std::uint64_t>();
        if (value > 9007199254740991ULL) throw std::out_of_range("integer");
        return value;
    } catch (const std::exception&) {
        throw engine::Error("named_version.invalid_field", std::string(field) + " is out of range", 4);
    }
}

std::string utc_now() {
    const auto now = std::chrono::time_point_cast<std::chrono::seconds>(std::chrono::system_clock::now());
    const std::time_t value = std::chrono::system_clock::to_time_t(now);
    std::tm result {};
    if (::gmtime_r(&value, &result) == nullptr) {
        throw engine::Error("named_version.clock_failed", "could not obtain a UTC timestamp", 5);
    }
    std::ostringstream output;
    output << std::put_time(&result, "%Y-%m-%dT%H:%M:%SZ");
    return output.str();
}

std::string finalize_digest(engine::Json& value, const char* field) {
    value.erase(field);
    const auto digest = engine::tagged_sha256(value.dump());
    value[field] = digest;
    return digest;
}

void verify_digest(const engine::Json& value, const char* field, const std::string& code) {
    if (!value.contains(field) || !value.at(field).is_string() ||
        !tagged_digest(value.at(field).get<std::string>())) {
        throw engine::Error(code, std::string(field) + " is invalid", 5);
    }
    auto copy = value;
    const auto expected = copy.at(field).get<std::string>();
    copy.erase(field);
    if (engine::tagged_sha256(copy.dump()) != expected) {
        throw engine::Error(code, std::string(field) + " digest mismatch", 5);
    }
}

std::vector<std::string> string_array(const engine::Json& value, const std::string& name,
                                      std::size_t maximum, bool nonempty = false) {
    if (!value.is_array() || value.size() > maximum || (nonempty && value.empty())) {
        throw engine::Error("named_version.invalid_field", name + " must be a bounded array", 4);
    }
    std::vector<std::string> result;
    std::set<std::string> seen;
    for (const auto& item : value) {
        if (!item.is_string() || !safe_token(item.get<std::string>()) ||
            !seen.insert(item.get<std::string>()).second) {
            throw engine::Error("named_version.invalid_field", name + " contains an invalid token", 4);
        }
        result.push_back(item.get<std::string>());
    }
    return result;
}

std::vector<std::uint64_t> version_array(const engine::Json& value, const std::string& name) {
    if (!value.is_array() || value.empty() || value.size() > 16U) {
        throw engine::Error("named_version.invalid_field", name + " must be a bounded array", 4);
    }
    std::vector<std::uint64_t> result;
    std::set<std::uint64_t> seen;
    for (const auto& item : value) {
        if ((!item.is_number_integer() && !item.is_number_unsigned()) ||
            item.get<std::uint64_t>() == 0U || item.get<std::uint64_t>() > 16U ||
            !seen.insert(item.get<std::uint64_t>()).second) {
            throw engine::Error("named_version.invalid_field", name + " contains an invalid version", 4);
        }
        result.push_back(item.get<std::uint64_t>());
    }
    return result;
}

engine::Json compatibility_result(const engine::Json& client, const engine::Json* registry) {
    exact_fields(client, {"client_id", "client_version", "process_protocols", "registry_read_versions",
                          "registry_write_versions", "capabilities"}, "Named Version client");
    if (text(client, "client_id", 128U) != "qxctl" || !safe_token(text(client, "client_version", 64U))) {
        throw engine::Error("named_version.client_invalid", "client identity is invalid", 4);
    }
    const auto protocols = string_array(client.at("process_protocols"), "process_protocols", 8U, true);
    const auto reads = version_array(client.at("registry_read_versions"), "registry_read_versions");
    const auto writes = version_array(client.at("registry_write_versions"), "registry_write_versions");
    const auto capabilities = string_array(client.at("capabilities"), "capabilities", 64U, true);
    std::vector<std::string> missing;
    for (const auto& required : required_capabilities) {
        if (std::find(capabilities.begin(), capabilities.end(), required) == capabilities.end()) missing.push_back(required);
    }
    const bool process_ok = std::find(protocols.begin(), protocols.end(), engine::process_protocol_v1) != protocols.end();
    const bool read_ok = std::find(reads.begin(), reads.end(), format_version) != reads.end();
    const bool write_ok = std::find(writes.begin(), writes.end(), format_version) != writes.end();
    const bool stored_ok = registry == nullptr || number(*registry, "format_version") == format_version;
    const bool full = process_ok && read_ok && write_ok && stored_ok && missing.empty();
    return engine::Json{
        {"mode", full ? "full" : (process_ok && read_ok && stored_ok ? "read_only" : "unsupported")},
        {"process_protocol", process_ok ? engine::Json(engine::process_protocol_v1) : engine::Json(nullptr)},
        {"registry_read_version", read_ok && stored_ok ? engine::Json(format_version) : engine::Json(nullptr)},
        {"registry_write_version", write_ok && stored_ok ? engine::Json(format_version) : engine::Json(nullptr)},
        {"missing_capabilities", missing},
    };
}

void require_full(const engine::Json& compatibility) {
    if (compatibility.at("mode") != "full") {
        throw engine::Error("named_version.compatibility_required", "full v1 Named Version capability overlap is required", 4);
    }
}

void validate_named_version(const engine::Json& value) {
    exact_fields(value, {"protocol", "named_version_id", "alias", "predecessor_digest",
        "component_requirements", "contract_requirements", "accord_reference_ids", "required_traits",
        "extension_points", "platform_bounds", "thermal_restriction", "sealed_at",
        "composition_authority_reference", "sodv_publication_reference", "named_version_digest"},
        "Named Version");
    if (text(value, "protocol") != "symphony.sav.named-version.v1" ||
        !text(value, "named_version_id").starts_with("savver:") ||
        !safe_token(text(value, "named_version_id")) || !safe_alias(text(value, "alias", 128U)) ||
        text(value, "thermal_restriction") != "freezing_only" ||
        !engine::is_utc_seconds(text(value, "sealed_at", 20U)) ||
        !safe_token(text(value, "composition_authority_reference")) ||
        !tagged_digest(text(value, "named_version_digest"))) {
        throw engine::Error("named_version.artifact_invalid", "Named Version identity or bounds are invalid", 4);
    }
    if (!value.at("predecessor_digest").is_null() &&
        (!value.at("predecessor_digest").is_string() ||
         !tagged_digest(value.at("predecessor_digest").get<std::string>()))) {
        throw engine::Error("named_version.artifact_invalid", "Named Version predecessor is invalid", 4);
    }
    if (!value.at("sodv_publication_reference").is_null() &&
        (!value.at("sodv_publication_reference").is_string() ||
         !safe_token(value.at("sodv_publication_reference").get<std::string>()))) {
        throw engine::Error("named_version.artifact_invalid", "Named Version publication reference is invalid", 4);
    }
    for (const auto* field : {"component_requirements", "contract_requirements", "accord_reference_ids",
                              "required_traits", "extension_points", "platform_bounds"}) {
        if (!value.at(field).is_array()) {
            throw engine::Error("named_version.artifact_invalid", std::string(field) + " must be an array", 4);
        }
    }
    verify_digest(value, "named_version_digest", "named_version.artifact_invalid");
}

void validate_validation_result(const engine::Json& value, const engine::Json& artifact) {
    exact_fields(value, {"protocol", "named_version_id", "named_version_digest", "state", "read_only",
                         "seal_authorized", "result_digest"}, "SAV validation result");
    if (text(value, "protocol") != "symphony.sav.named-version-validation-result.v1" ||
        value.at("named_version_id") != artifact.at("named_version_id") ||
        value.at("named_version_digest") != artifact.at("named_version_digest") ||
        text(value, "state") != "valid_immutable_envelope" || !value.at("read_only").is_boolean() ||
        !value.at("read_only").get<bool>() || !value.at("seal_authorized").is_boolean() ||
        value.at("seal_authorized").get<bool>()) {
        throw engine::Error("named_version.validation_invalid", "SAV validation result does not bind the artifact", 4);
    }
    verify_digest(value, "result_digest", "named_version.validation_invalid");
}

void validate_engine_evidence(const engine::Json& value) {
    exact_fields(value, {"module_id", "engine_id", "vector_id", "version", "receipt_digest",
                         "executable_digest", "evidence_digest"}, "SAV engine evidence");
    if (text(value, "module_id") != "sav-engine" || text(value, "engine_id") != "symphony-sav" ||
        text(value, "vector_id") != "sav" || !safe_token(text(value, "version", 64U)) ||
        !tagged_digest(text(value, "receipt_digest")) || !tagged_digest(text(value, "executable_digest"))) {
        throw engine::Error("named_version.engine_invalid", "SAV engine evidence is invalid", 4);
    }
    verify_digest(value, "evidence_digest", "named_version.engine_invalid");
}

void validate_registry(const engine::Json& registry) {
    exact_fields(registry, {"protocol", "format_version", "tops_id", "generation",
        "previous_registry_digest", "versions", "aliases", "operations", "updated_at",
        "recovery", "canonical", "registry_digest"}, "Named Version registry");
    const auto generation = number(registry, "generation");
    if (text(registry, "protocol") != registry_protocol || number(registry, "format_version") != format_version ||
        !valid_uuid(text(registry, "tops_id")) || generation == 0U ||
        !engine::is_utc_seconds(text(registry, "updated_at", 20U)) ||
        !registry.at("canonical").is_boolean() || registry.at("canonical").get<bool>() ||
        (!registry.at("previous_registry_digest").is_null() &&
         (!registry.at("previous_registry_digest").is_string() ||
          !tagged_digest(registry.at("previous_registry_digest").get<std::string>()))) ||
        (generation == 1U) != registry.at("previous_registry_digest").is_null() ||
        !registry.at("versions").is_array() || registry.at("versions").size() > max_versions ||
        !registry.at("aliases").is_array() || registry.at("aliases").size() > max_aliases ||
        !registry.at("operations").is_array() || registry.at("operations").size() > max_operations) {
        throw engine::Error("named_version.registry_invalid", "Named Version registry identity is invalid", 5);
    }
    std::set<std::string> ids;
    std::set<std::string> digests;
    for (const auto& item : registry.at("versions")) {
        exact_fields(item, {"named_version_id", "named_version_digest", "display_alias",
                            "predecessor_digest", "sealed_at", "object_name"}, "Named Version entry");
        const auto id = text(item, "named_version_id");
        const auto digest = text(item, "named_version_digest");
        if (!safe_token(id) || !id.starts_with("savver:") || !tagged_digest(digest) ||
            !ids.insert(id).second || !digests.insert(digest).second ||
            !safe_alias(text(item, "display_alias", 128U)) ||
            !engine::is_utc_seconds(text(item, "sealed_at", 20U)) ||
            text(item, "object_name") != "object." + digest.substr(7U) + ".json" ||
            (!item.at("predecessor_digest").is_null() &&
             (!item.at("predecessor_digest").is_string() ||
              !tagged_digest(item.at("predecessor_digest").get<std::string>())))) {
            throw engine::Error("named_version.registry_invalid", "Named Version entry is invalid", 5);
        }
    }
    for (const auto& item : registry.at("versions")) {
        if (item.at("predecessor_digest").is_string() &&
            !digests.contains(item.at("predecessor_digest").get<std::string>())) {
            throw engine::Error("named_version.registry_invalid",
                                "Named Version predecessor is absent from the registry", 5);
        }
    }
    std::set<std::string> aliases;
    for (const auto& item : registry.at("aliases")) {
        exact_fields(item, {"alias", "named_version_id", "named_version_digest", "updated_at"},
                     "Named Version alias");
        const auto alias_id = text(item, "named_version_id");
        const auto alias_digest = text(item, "named_version_digest");
        const auto target = std::find_if(registry.at("versions").begin(), registry.at("versions").end(),
            [&](const auto& version) {
                return version.at("named_version_id") == alias_id &&
                    version.at("named_version_digest") == alias_digest;
            });
        if (!safe_alias(text(item, "alias", 128U)) || !aliases.insert(text(item, "alias", 128U)).second ||
            target == registry.at("versions").end() ||
            !engine::is_utc_seconds(text(item, "updated_at", 20U))) {
            throw engine::Error("named_version.registry_invalid", "Named Version alias is invalid", 5);
        }
    }
    std::set<std::string> operation_ids;
    for (const auto& item : registry.at("operations")) {
        exact_fields(item, {"operation", "operation_id", "operation_fingerprint", "completed_at",
                            "selected_digest"}, "Named Version operation record");
        if (!safe_token(text(item, "operation")) || !safe_token(text(item, "operation_id")) ||
            !operation_ids.insert(text(item, "operation_id")).second ||
            !tagged_digest(text(item, "operation_fingerprint")) ||
            !engine::is_utc_seconds(text(item, "completed_at", 20U)) ||
            (!item.at("selected_digest").is_null() &&
             (!item.at("selected_digest").is_string() ||
              !tagged_digest(item.at("selected_digest").get<std::string>()))) ) {
            throw engine::Error("named_version.registry_invalid", "Named Version operation record is invalid", 5);
        }
    }
    exact_fields(registry.at("recovery"), {"state", "recovered_from_digest", "detail"},
                 "Named Version recovery");
    if (!safe_token(text(registry.at("recovery"), "state")) ||
        text(registry.at("recovery"), "detail", 512U).empty()) {
        throw engine::Error("named_version.registry_invalid", "Named Version recovery evidence is invalid", 5);
    }
    verify_digest(registry, "registry_digest", "named_version.registry_invalid");
}

void validate_head(const engine::Json& head) {
    exact_fields(head, {"protocol", "format_version", "tops_id", "active_slot", "generation",
                        "registry_digest", "previous_head_digest", "head_digest"}, "Named Version head");
    const auto slot = number(head, "active_slot");
    if (text(head, "protocol") != head_protocol || number(head, "format_version") != format_version ||
        !valid_uuid(text(head, "tops_id")) || slot > 1U || number(head, "generation") == 0U ||
        !tagged_digest(text(head, "registry_digest")) ||
        (!head.at("previous_head_digest").is_null() &&
         (!head.at("previous_head_digest").is_string() ||
          !tagged_digest(head.at("previous_head_digest").get<std::string>()))) ) {
        throw engine::Error("named_version.head_invalid", "Named Version head is invalid", 5);
    }
    verify_digest(head, "head_digest", "named_version.head_invalid");
}

std::optional<FileDescriptor> open_absolute_directory(const std::string& root, bool create) {
    if (!safe_absolute_path(root)) {
        throw engine::Error("named_version.state_root_invalid", "state_root must be a safe absolute path", 4);
    }
    FileDescriptor current(::open("/", O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC));
    if (current.get() < 0) system_error("named_version.state_open_failed", "could not open filesystem root");
    for (const auto& item : fs::path(root).relative_path()) {
        const auto component = item.string();
        int next = ::openat(current.get(), component.c_str(), O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
        if (next < 0 && errno == ENOENT && !create) return std::nullopt;
        if (next < 0 && errno == ENOENT && create) {
            if (::mkdirat(current.get(), component.c_str(), 0700) != 0 && errno != EEXIST) {
                system_error("named_version.state_create_failed", "could not create state directory");
            }
            if (::fsync(current.get()) != 0) system_error("named_version.state_sync_failed", "could not sync parent");
            next = ::openat(current.get(), component.c_str(), O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
        }
        if (next < 0) system_error("named_version.state_open_failed", "could not open state directory");
        current = FileDescriptor(next);
    }
    struct stat status {};
    if (::fstat(current.get(), &status) != 0 || !S_ISDIR(status.st_mode) ||
        status.st_uid != ::geteuid() || (status.st_mode & 0022) != 0) {
        throw engine::Error("named_version.state_unsafe", "state root must be caller-owned and protected", 5);
    }
    return std::optional<FileDescriptor>(std::move(current));
}

std::optional<FileDescriptor> open_child(FileDescriptor parent, const std::string& name, bool create) {
    int next = ::openat(parent.get(), name.c_str(), O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
    if (next < 0 && errno == ENOENT && !create) return std::nullopt;
    if (next < 0 && errno == ENOENT && create) {
        if (::mkdirat(parent.get(), name.c_str(), 0700) != 0 && errno != EEXIST) {
            system_error("named_version.state_create_failed", "could not create store directory");
        }
        if (::fsync(parent.get()) != 0) system_error("named_version.state_sync_failed", "could not sync parent");
        next = ::openat(parent.get(), name.c_str(), O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
    }
    if (next < 0) system_error("named_version.state_open_failed", "could not open store directory");
    FileDescriptor result(next);
    struct stat status {};
    if (::fstat(result.get(), &status) != 0 || !S_ISDIR(status.st_mode) || status.st_uid != ::geteuid() ||
        (status.st_mode & 0022) != 0) {
        throw engine::Error("named_version.state_unsafe", "store directory is unsafe", 5);
    }
    if (create && ::fchmod(result.get(), 0700) != 0) {
        system_error("named_version.state_mode_failed", "could not restrict store directory");
    }
    return std::optional<FileDescriptor>(std::move(result));
}

std::optional<StoreLock> open_store(const std::string& root, const std::string& tops_id,
                                    bool exclusive, bool create) {
    auto opened = open_absolute_directory(root, create);
    if (!opened.has_value()) return std::nullopt;
    FileDescriptor current = std::move(*opened);
    const std::array<std::string, 7> components = {
        "symphony", "knowledge-session-coordinator", "accordare", "v1", "tops",
        engine::sha256_hex("tops:" + tops_id), "named-versions",
    };
    for (const auto& component : components) {
        auto child = open_child(std::move(current), component, create);
        if (!child.has_value()) return std::nullopt;
        current = std::move(*child);
    }
    const int raw = ::openat(current.get(), ".lock",
        (create ? O_RDWR | O_CREAT : O_RDONLY) | O_NONBLOCK | O_NOFOLLOW | O_CLOEXEC, 0600);
    if (raw < 0 && errno == ENOENT && !create) return std::nullopt;
    if (raw < 0) system_error("named_version.lock_open_failed", "could not open store lock");
    FileDescriptor lock(raw);
    struct stat status {};
    if (::fstat(lock.get(), &status) != 0 || !S_ISREG(status.st_mode) || status.st_uid != ::geteuid() ||
        (status.st_mode & 0777) != 0600 || status.st_nlink != 1) {
        throw engine::Error("named_version.lock_unsafe", "store lock is unsafe", 5);
    }
    if (::flock(lock.get(), (exclusive ? LOCK_EX : LOCK_SH) | LOCK_NB) != 0) {
        if (errno == EWOULDBLOCK) throw engine::Error("named_version.lock_busy", "Named Version store is busy", 4);
        system_error("named_version.lock_failed", "could not lock Named Version store");
    }
    return StoreLock(std::move(current), std::move(lock));
}

std::optional<std::string> read_file(int directory, const std::string& name) {
    const int raw = ::openat(directory, name.c_str(), O_RDONLY | O_NONBLOCK | O_NOFOLLOW | O_CLOEXEC);
    if (raw < 0) {
        if (errno == ENOENT) return std::nullopt;
        system_error("named_version.state_read_failed", "could not open state file");
    }
    FileDescriptor file(raw);
    struct stat status {};
    if (::fstat(file.get(), &status) != 0 || !S_ISREG(status.st_mode) || status.st_uid != ::geteuid() ||
        (status.st_mode & 0777) != 0600 || status.st_nlink != 1 || status.st_size < 0 ||
        static_cast<std::uint64_t>(status.st_size) > max_state_bytes) {
        throw engine::Error("named_version.state_unsafe", "Named Version state file is unsafe", 5);
    }
    std::string data;
    data.reserve(static_cast<std::size_t>(status.st_size));
    std::array<char, 16384> buffer {};
    for (;;) {
        const auto count = ::read(file.get(), buffer.data(), buffer.size());
        if (count < 0) {
            if (errno == EINTR) continue;
            system_error("named_version.state_read_failed", "could not read state file");
        }
        if (count == 0) break;
        if (data.size() + static_cast<std::size_t>(count) > max_state_bytes) {
            throw engine::Error("named_version.state_too_large", "Named Version state exceeds its bound", 5);
        }
        data.append(buffer.data(), static_cast<std::size_t>(count));
    }
    return data;
}

engine::Json parse_file(const std::string& data) {
    try { return engine::parse_bounded_json(data, max_state_bytes); }
    catch (const engine::Error&) {
        throw engine::Error("named_version.state_json_invalid", "Named Version state is invalid JSON", 5);
    }
}

void write_all(int file, const std::string& data) {
    std::size_t offset = 0U;
    while (offset < data.size()) {
        const auto count = ::write(file, data.data() + offset, data.size() - offset);
        if (count < 0) {
            if (errno == EINTR) continue;
            system_error("named_version.state_write_failed", "could not write Named Version state");
        }
        offset += static_cast<std::size_t>(count);
    }
}

void write_immutable(int directory, const std::string& name, const engine::Json& value) {
    const auto data = value.dump() + "\n";
    if (data.size() > max_state_bytes) {
        throw engine::Error("named_version.state_too_large", "immutable state exceeds its bound", 5);
    }
    const int raw = ::openat(directory, name.c_str(), O_WRONLY | O_CREAT | O_EXCL | O_NOFOLLOW | O_CLOEXEC, 0600);
    if (raw < 0 && errno == EEXIST) {
        const auto existing = read_file(directory, name);
        if (!existing.has_value() || parse_file(*existing) != value) {
            throw engine::Error("named_version.immutable_conflict", "immutable state name contains different bytes", 5);
        }
        return;
    }
    if (raw < 0) system_error("named_version.state_write_failed", "could not create immutable state");
    FileDescriptor file(raw);
    write_all(file.get(), data);
    if (::fsync(file.get()) != 0 || ::fsync(directory) != 0) {
        system_error("named_version.state_sync_failed", "could not sync immutable state");
    }
}

void write_slot(int directory, int slot, const engine::Json& registry) {
    const auto data = registry.dump() + "\n";
    if (data.size() > max_state_bytes) throw engine::Error("named_version.state_too_large", "registry exceeds its bound", 5);
    const auto name = "registry." + std::to_string(slot) + ".json";
    const int raw = ::openat(directory, name.c_str(), O_WRONLY | O_CREAT | O_NONBLOCK | O_NOFOLLOW | O_CLOEXEC, 0600);
    if (raw < 0) system_error("named_version.state_write_failed", "could not open inactive registry slot");
    FileDescriptor file(raw);
    struct stat status {};
    if (::fstat(file.get(), &status) != 0 || !S_ISREG(status.st_mode) || status.st_uid != ::geteuid() ||
        (status.st_mode & 0777) != 0600 || status.st_nlink != 1) {
        throw engine::Error("named_version.state_unsafe", "registry slot is unsafe", 5);
    }
    if (::ftruncate(file.get(), 0) != 0) system_error("named_version.state_write_failed", "could not truncate registry");
    write_all(file.get(), data);
    if (::fsync(file.get()) != 0) system_error("named_version.state_sync_failed", "could not sync registry");
}

void write_head(int directory, engine::Json head) {
    finalize_digest(head, "head_digest");
    static std::atomic<std::uint64_t> sequence {0U};
    const auto temporary = ".head.tmp." + std::to_string(::getpid()) + "." +
        std::to_string(sequence.fetch_add(1U, std::memory_order_relaxed));
    const int raw = ::openat(directory, temporary.c_str(), O_WRONLY | O_CREAT | O_EXCL | O_NOFOLLOW | O_CLOEXEC, 0600);
    if (raw < 0) system_error("named_version.head_write_failed", "could not create temporary head");
    {
        FileDescriptor file(raw);
        write_all(file.get(), head.dump() + "\n");
        if (::fsync(file.get()) != 0) system_error("named_version.state_sync_failed", "could not sync head");
    }
    if (::renameat(directory, temporary.c_str(), directory, "head.json") != 0) {
        static_cast<void>(::unlinkat(directory, temporary.c_str(), 0));
        system_error("named_version.head_commit_failed", "could not replace Named Version head");
    }
    if (::fsync(directory) != 0) system_error("named_version.state_sync_failed", "could not sync store directory");
}

Candidate read_candidate(int directory, int slot) {
    Candidate candidate;
    candidate.slot = slot;
    const auto data = read_file(directory, "registry." + std::to_string(slot) + ".json");
    if (!data.has_value()) return candidate;
    candidate.exists = true;
    try {
        candidate.registry = parse_file(*data);
        if (!candidate.registry.is_object() || !candidate.registry.contains("protocol") ||
            !candidate.registry.contains("format_version") ||
            candidate.registry.at("protocol") != registry_protocol ||
            candidate.registry.at("format_version") != format_version) {
            candidate.incompatible = true;
            return candidate;
        }
        validate_registry(candidate.registry);
        candidate.valid = true;
    } catch (const engine::Error&) {
        candidate.valid = false;
    }
    return candidate;
}

std::optional<engine::Json> read_head(int directory, bool tolerate_invalid = false) {
    const auto data = read_file(directory, "head.json");
    if (!data.has_value()) return std::nullopt;
    try {
        auto head = parse_file(*data);
        if (head.is_object() && head.contains("protocol") && head.contains("format_version") &&
            (head.at("protocol") != head_protocol || head.at("format_version") != format_version)) {
            throw engine::Error("named_version.compatibility_required", "head uses unsupported state", 4);
        }
        validate_head(head);
        return head;
    } catch (const engine::Error& error) {
        if (tolerate_invalid && error.code() != "named_version.compatibility_required") return std::nullopt;
        throw;
    }
}

State load_state(int directory) {
    const auto head = read_head(directory);
    if (!head.has_value()) {
        const auto zero = read_candidate(directory, 0);
        const auto one = read_candidate(directory, 1);
        if (zero.exists || one.exists) throw engine::Error("named_version.head_missing", "registry slots exist without a head", 5);
        return {};
    }
    const int slot = static_cast<int>(number(*head, "active_slot"));
    const auto active = read_candidate(directory, slot);
    if (active.incompatible) throw engine::Error("named_version.compatibility_required", "active registry is incompatible", 4);
    if (!active.valid || active.registry.at("registry_digest") != head->at("registry_digest") ||
        active.registry.at("generation") != head->at("generation") || active.registry.at("tops_id") != head->at("tops_id")) {
        throw engine::Error("named_version.head_slot_mismatch", "head does not select its registry", 5);
    }
    const auto inactive = read_candidate(directory, 1 - slot);
    if (inactive.incompatible) throw engine::Error("named_version.compatibility_required", "inactive registry is incompatible", 4);
    const auto generation = number(active.registry, "generation");
    if (!inactive.exists && generation > 1U) throw engine::Error("named_version.recovery_required", "registry predecessor is missing", 5);
    if (inactive.exists && !inactive.valid) throw engine::Error("named_version.recovery_required", "registry predecessor is invalid", 5);
    if (inactive.valid) {
        const auto other = number(inactive.registry, "generation");
        if (other == generation) {
            throw engine::Error(active.registry.at("registry_digest") == inactive.registry.at("registry_digest") ?
                "named_version.recovery_required" : "named_version.recovery_ambiguous",
                "registry slots share one generation", 5);
        }
        if (other > generation) {
            const bool linked = other == generation + 1U &&
                inactive.registry.at("previous_registry_digest") == active.registry.at("registry_digest");
            throw engine::Error(linked ? "named_version.recovery_required" : "named_version.recovery_ambiguous",
                                "inactive registry is newer than the selected head", 5);
        }
        if (generation != other + 1U ||
            active.registry.at("previous_registry_digest") != inactive.registry.at("registry_digest")) {
            throw engine::Error("named_version.recovery_ambiguous", "registry slots are not one linked chain", 5);
        }
    }
    return State{*head, active.registry, true};
}

State commit_to_slot(int directory, engine::Json registry, int slot,
                     const std::optional<engine::Json>& prior_head) {
    finalize_digest(registry, "registry_digest");
    validate_registry(registry);
    write_slot(directory, slot, registry);
    if (::fsync(directory) != 0) system_error("named_version.state_sync_failed", "could not sync registry directory");
    engine::Json head{
        {"protocol", head_protocol}, {"format_version", format_version}, {"tops_id", registry.at("tops_id")},
        {"active_slot", slot}, {"generation", registry.at("generation")},
        {"registry_digest", registry.at("registry_digest")},
        {"previous_head_digest", prior_head.has_value() ? prior_head->at("head_digest") : engine::Json(nullptr)},
    };
    write_head(directory, head);
    return State{*read_head(directory), std::move(registry), true};
}

State commit(int directory, engine::Json registry, const std::optional<engine::Json>& prior_head) {
    const int slot = prior_head.has_value() ? 1 - static_cast<int>(number(*prior_head, "active_slot")) : 0;
    return commit_to_slot(directory, std::move(registry), slot, prior_head);
}

Candidate choose_recovery_candidate(int directory) {
    const auto zero = read_candidate(directory, 0);
    const auto one = read_candidate(directory, 1);
    if (zero.incompatible || one.incompatible) {
        throw engine::Error("named_version.compatibility_required", "stored registry uses an unsupported format", 4);
    }
    std::vector<Candidate> valid;
    if (zero.valid) valid.push_back(zero);
    if (one.valid) valid.push_back(one);
    if (valid.empty()) throw engine::Error("named_version.recovery_unavailable", "no valid registry exists", 5);
    if (valid.size() == 1U) return valid.front();
    const auto first = number(valid[0].registry, "generation");
    const auto second = number(valid[1].registry, "generation");
    if (first == second) {
        if (valid[0].registry.at("registry_digest") == valid[1].registry.at("registry_digest")) return valid[0];
        throw engine::Error("named_version.recovery_ambiguous", "registries diverge at one generation", 5);
    }
    const Candidate& newer = first > second ? valid[0] : valid[1];
    const Candidate& older = first > second ? valid[1] : valid[0];
    if (number(newer.registry, "generation") != number(older.registry, "generation") + 1U ||
        newer.registry.at("previous_registry_digest") != older.registry.at("registry_digest")) {
        throw engine::Error("named_version.recovery_ambiguous", "registries do not form one linked chain", 5);
    }
    return newer;
}

engine::Json clean_recovery() {
    return engine::Json{{"state", "clean"}, {"recovered_from_digest", nullptr},
                        {"detail", "no recovery was required"}};
}

std::string current_digest(const State& state) {
    return state.present ? state.registry.at("registry_digest").get<std::string>() : "absent";
}

void require_expected(const State& state, const std::string& expected) {
    if (current_digest(state) != expected) {
        throw engine::Error("named_version.stale_expected_state", "expected registry state is stale", 4);
    }
}

std::string operation_fingerprint(const engine::Json& payload) {
    engine::Json normalized{
        {"operation", payload.at("operation")}, {"operation_id", payload.at("operation_id")},
        {"expected_registry_digest", payload.at("expected_registry_digest")},
        {"named_version", payload.at("named_version")}, {"validation_result", payload.at("validation_result")},
        {"sav_engine", payload.at("sav_engine")}, {"prepared_operation_id", payload.at("prepared_operation_id")},
        {"proposal_digest", payload.at("proposal_digest")}, {"alias", payload.at("alias")},
        {"selector", payload.at("selector")},
    };
    return engine::tagged_sha256(normalized.dump());
}

std::optional<engine::Json> find_operation(const engine::Json& registry, const std::string& operation_id) {
    for (const auto& record : registry.at("operations")) {
        if (record.at("operation_id") == operation_id) return record;
    }
    return std::nullopt;
}

void append_operation(engine::Json& registry, const engine::Json& payload, const engine::Json& selected_digest,
                      const std::string& completed_at) {
    if (registry.at("operations").size() >= max_operations) {
        throw engine::Error("named_version.capacity_review_required", "operation record capacity requires review", 4);
    }
    registry["operations"].push_back(engine::Json{
        {"operation", payload.at("operation")}, {"operation_id", payload.at("operation_id")},
        {"operation_fingerprint", operation_fingerprint(payload)}, {"completed_at", completed_at},
        {"selected_digest", selected_digest},
    });
}

std::optional<engine::Json> find_version(const engine::Json& registry, const std::string& kind,
                                         const std::string& value) {
    if (kind == "digest") {
        for (const auto& entry : registry.at("versions")) {
            if (entry.at("named_version_digest") == value) return entry;
        }
    } else if (kind == "id") {
        for (const auto& entry : registry.at("versions")) {
            if (entry.at("named_version_id") == value) return entry;
        }
    } else if (kind == "alias") {
        for (const auto& alias : registry.at("aliases")) {
            if (alias.at("alias") == value) {
                return find_version(registry, "digest", alias.at("named_version_digest").get<std::string>());
            }
        }
    }
    return std::nullopt;
}

engine::Json validate_selector(const engine::Json& selector) {
    exact_fields(selector, {"kind", "value"}, "Named Version selector");
    const auto kind = text(selector, "kind", 16U);
    const auto value = text(selector, "value", 256U);
    if ((kind == "digest" && !tagged_digest(value)) ||
        (kind == "id" && (!safe_token(value) || !value.starts_with("savver:"))) ||
        (kind == "alias" && !safe_alias(value)) ||
        (kind != "digest" && kind != "id" && kind != "alias")) {
        throw engine::Error("named_version.selector_invalid", "Named Version selector is invalid", 4);
    }
    return selector;
}

std::string resource(const engine::Json& payload) {
    const auto normalized = engine::Json{
        {"tops_id", payload.at("tops_id")}, {"operation", payload.at("operation")},
        {"expected_registry_digest", payload.at("expected_registry_digest")},
        {"named_version_digest", payload.at("named_version").is_object() ?
            payload.at("named_version").at("named_version_digest") : engine::Json(nullptr)},
        {"prepared_operation_id", payload.at("prepared_operation_id")},
        {"proposal_digest", payload.at("proposal_digest")}, {"alias", payload.at("alias")},
        {"selector", payload.at("selector")},
    };
    return "symphony.knowledge.named-version:" + engine::sha256_hex(normalized.dump());
}

engine::Json make_result(const std::string& operation, const engine::Json& compatibility,
                         const State& state, const engine::Json& proposal_digest,
                         const engine::Json& artifact, const engine::Json& selected_alias,
                         bool changed, bool recovered, engine::Json repairs, bool read_only) {
    engine::Json result{
        {"protocol", result_protocol}, {"format_version", format_version}, {"operation", operation},
        {"compatibility", compatibility}, {"registry_present", state.present},
        {"registry_digest", state.present ? state.registry.at("registry_digest") : engine::Json(nullptr)},
        {"version_count", state.present ? state.registry.at("versions").size() : 0U},
        {"alias_count", state.present ? state.registry.at("aliases").size() : 0U},
        {"proposal_digest", proposal_digest}, {"artifact", artifact}, {"selected_alias", selected_alias},
        {"changed", changed}, {"recovered", recovered}, {"repair_actions", std::move(repairs)},
        {"read_only", read_only}, {"canonical_apply_enabled", false}, {"canonical", false},
        {"stav_append_enabled", false}, {"result_digest", nullptr},
    };
    finalize_digest(result, "result_digest");
    return result;
}

engine::Json load_proposal(int directory, const std::string& operation_id, const std::string& digest) {
    const auto name = "proposal." + engine::sha256_hex("operation:" + operation_id) + ".json";
    const auto data = read_file(directory, name);
    if (!data.has_value()) throw engine::Error("named_version.proposal_missing", "prepared proposal is absent", 4);
    const auto proposal = parse_file(*data);
    exact_fields(proposal, {"protocol", "tops_id", "operation_id", "operation_fingerprint",
        "observed_registry_digest", "named_version", "validation_result", "sav_engine", "prepared_at",
        "authority", "canonical", "proposal_digest"}, "Named Version proposal");
    if (text(proposal, "protocol") != proposal_protocol || !valid_uuid(text(proposal, "tops_id")) ||
        text(proposal, "operation_id") != operation_id ||
        !tagged_digest(text(proposal, "operation_fingerprint")) ||
        text(proposal, "proposal_digest") != digest || !tagged_digest(digest) ||
        (!proposal.at("observed_registry_digest").is_string() ||
         (proposal.at("observed_registry_digest") != "absent" &&
          !tagged_digest(proposal.at("observed_registry_digest").get<std::string>()))) ||
        !engine::is_utc_seconds(text(proposal, "prepared_at", 20U)) ||
        !proposal.at("canonical").is_boolean() || proposal.at("canonical").get<bool>()) {
        throw engine::Error("named_version.proposal_invalid", "prepared proposal identity is invalid", 5);
    }
    validate_named_version(proposal.at("named_version"));
    validate_validation_result(proposal.at("validation_result"), proposal.at("named_version"));
    validate_engine_evidence(proposal.at("sav_engine"));
    exact_fields(proposal.at("authority"), {"decision_id", "capability_binding_digest"},
                 "proposal authority evidence");
    if (!safe_token(text(proposal.at("authority"), "decision_id")) ||
        !tagged_digest(text(proposal.at("authority"), "capability_binding_digest"))) {
        throw engine::Error("named_version.proposal_invalid", "proposal authority evidence is invalid", 5);
    }
    verify_digest(proposal, "proposal_digest", "named_version.proposal_invalid");
    return proposal;
}

engine::Json read_artifact(int directory, const engine::Json& entry) {
    const auto data = read_file(directory, text(entry, "object_name"));
    if (!data.has_value()) throw engine::Error("named_version.object_missing", "registered immutable object is absent", 5);
    const auto artifact = parse_file(*data);
    validate_named_version(artifact);
    if (artifact.at("named_version_id") != entry.at("named_version_id") ||
        artifact.at("named_version_digest") != entry.at("named_version_digest")) {
        throw engine::Error("named_version.object_mismatch", "immutable object does not match its registry entry", 5);
    }
    return artifact;
}

} // namespace

engine::Json named_version_capabilities() {
    return engine::Json{
        {"protocol", command_protocol}, {"format_version", format_version},
        {"registry_read_versions", engine::Json::array({format_version})},
        {"registry_write_versions", engine::Json::array({format_version})},
        {"required_capabilities", required_capabilities},
        {"operations", engine::Json::array({"named_version_prepare", "named_version_seal",
            "named_version_alias", "named_version_lookup", "named_version_status", "named_version_recover"})},
        {"canonical_apply_enabled", false}, {"stav_append_enabled", false},
    };
}

engine::Json handle_named_version(const engine::Request& request) {
    const auto& payload = request.payload;
    exact_fields(payload, {"protocol", "operation", "state_root", "tops_id", "operation_id",
        "expected_registry_digest", "named_version", "validation_result", "sav_engine",
        "prepared_operation_id", "proposal_digest", "alias", "selector", "authorization_decision", "client"},
        "Named Version command");
    if (text(payload, "protocol") != command_protocol || text(payload, "operation") != request.operation ||
        !safe_absolute_path(text(payload, "state_root", engine::Limits::max_path_bytes)) ||
        !valid_uuid(text(payload, "tops_id", 36U))) {
        throw engine::Error("named_version.command_invalid", "Named Version command identity is invalid", 4);
    }
    const bool status = request.operation == "named_version_status";
    const bool lookup = request.operation == "named_version_lookup";
    const bool prepare = request.operation == "named_version_prepare";
    const bool seal = request.operation == "named_version_seal";
    const bool alias = request.operation == "named_version_alias";
    const bool recover = request.operation == "named_version_recover";
    if (!status && !lookup && !prepare && !seal && !alias && !recover) {
        throw engine::Error("named_version.operation_invalid", "Named Version operation is unsupported", 4);
    }
    if ((status || lookup) ? !payload.at("operation_id").is_null() :
        (!payload.at("operation_id").is_string() || !safe_token(text(payload, "operation_id")))) {
        throw engine::Error("named_version.command_invalid", "operation_id presence is invalid", 4);
    }
    const auto expected = payload.at("expected_registry_digest").is_string() ?
        payload.at("expected_registry_digest").get<std::string>() : std::string();
    if ((status || lookup) ? !payload.at("expected_registry_digest").is_null() :
        (expected != "absent" && expected != "discover" && !tagged_digest(expected))) {
        throw engine::Error("named_version.command_invalid", "expected registry state is invalid", 4);
    }
    if (recover && expected != "discover" && !tagged_digest(expected)) {
        throw engine::Error("named_version.command_invalid", "recovery requires a digest or discover", 4);
    }
    if (!recover && expected == "discover") {
        throw engine::Error("named_version.command_invalid", "discover is reserved for recovery", 4);
    }
    if (prepare) {
        if (!payload.at("named_version").is_object() || !payload.at("validation_result").is_object() ||
            !payload.at("sav_engine").is_object()) {
            throw engine::Error("named_version.command_invalid", "prepare requires artifact and validation evidence", 4);
        }
        validate_named_version(payload.at("named_version"));
        validate_validation_result(payload.at("validation_result"), payload.at("named_version"));
        validate_engine_evidence(payload.at("sav_engine"));
    } else if (!payload.at("named_version").is_null() || !payload.at("validation_result").is_null() ||
               !payload.at("sav_engine").is_null()) {
        throw engine::Error("named_version.command_invalid", "operation carries unexpected SAV evidence", 4);
    }
    if (seal) {
        if (!payload.at("prepared_operation_id").is_string() ||
            !safe_token(text(payload, "prepared_operation_id")) ||
            !payload.at("proposal_digest").is_string() || !tagged_digest(text(payload, "proposal_digest"))) {
            throw engine::Error("named_version.command_invalid", "seal requires one exact prepared proposal", 4);
        }
    } else if (!payload.at("prepared_operation_id").is_null() || !payload.at("proposal_digest").is_null()) {
        throw engine::Error("named_version.command_invalid", "operation carries unexpected proposal evidence", 4);
    }
    if (alias) {
        if (!payload.at("alias").is_string() || !safe_alias(text(payload, "alias", 128U)) ||
            !payload.at("selector").is_object() || validate_selector(payload.at("selector")).at("kind") != "digest") {
            throw engine::Error("named_version.command_invalid", "alias requires a digest selector and safe alias", 4);
        }
    } else if (lookup) {
        if (!payload.at("selector").is_object()) {
            throw engine::Error("named_version.command_invalid", "lookup requires a selector", 4);
        }
        static_cast<void>(validate_selector(payload.at("selector")));
        if (!payload.at("alias").is_null()) {
            throw engine::Error("named_version.command_invalid", "lookup carries an alias mutation", 4);
        }
    } else if (!payload.at("alias").is_null() || !payload.at("selector").is_null()) {
        throw engine::Error("named_version.command_invalid", "operation carries unexpected selector evidence", 4);
    }
    if (!payload.at("authorization_decision").is_object()) {
        throw engine::Error("named_version.command_invalid", "fresh SSIAG authorization is required", 4);
    }
    static_cast<void>(validate_ssiag_authorization(payload.at("authorization_decision"),
        "symphony.knowledge." + request.operation, text(payload, "tops_id"), resource(payload)));

    const bool mutation = prepare || seal || alias || recover;
    auto store = open_store(text(payload, "state_root", engine::Limits::max_path_bytes),
                            text(payload, "tops_id"), mutation, mutation);
    if (!store.has_value()) {
        const auto compatibility = compatibility_result(payload.at("client"), nullptr);
        if (status) return make_result(request.operation, compatibility, {}, nullptr, nullptr, nullptr,
                                       false, false, engine::Json::array(), true);
        throw engine::Error("named_version.store_absent", "Named Version store is absent", 4);
    }

    State state;
    if (recover) {
        try { state = load_state(store->directory_fd()); }
        catch (const engine::Error& error) {
            static const std::set<std::string> recoverable = {
                "named_version.head_invalid", "named_version.head_missing", "named_version.head_slot_mismatch",
                "named_version.recovery_required", "named_version.state_json_invalid",
            };
            if (!recoverable.contains(error.code())) throw;
        }
    } else {
        state = load_state(store->directory_fd());
    }
    const auto compatibility = compatibility_result(payload.at("client"), state.present ? &state.registry : nullptr);
    if (!mutation) {
        if (compatibility.at("mode") == "unsupported") {
            throw engine::Error("named_version.compatibility_required", "Named Version registry is unreadable", 4);
        }
    } else {
        require_full(compatibility);
    }

    if (status) {
        return make_result(request.operation, compatibility, state, nullptr, nullptr, nullptr,
                           false, false, engine::Json::array(), true);
    }
    if (lookup) {
        if (!state.present) throw engine::Error("named_version.not_found", "Named Version registry is absent", 4);
        const auto selector = validate_selector(payload.at("selector"));
        const auto entry = find_version(state.registry, text(selector, "kind", 16U), text(selector, "value", 256U));
        if (!entry.has_value()) throw engine::Error("named_version.not_found", "Named Version selector did not resolve", 4);
        const auto artifact = read_artifact(store->directory_fd(), *entry);
        engine::Json selected_alias = nullptr;
        if (selector.at("kind") == "alias") selected_alias = selector.at("value");
        return make_result(request.operation, compatibility, state, nullptr, artifact, selected_alias,
                           false, false, engine::Json::array(), true);
    }
    if (prepare) {
        require_expected(state, expected);
        const auto fingerprint = operation_fingerprint(payload);
        const auto proposal_name = "proposal." + engine::sha256_hex(
            "operation:" + text(payload, "operation_id")) + ".json";
        if (const auto existing = read_file(store->directory_fd(), proposal_name); existing.has_value()) {
            const auto proposal = parse_file(*existing);
            if (!proposal.is_object() || !proposal.contains("operation_fingerprint") ||
                proposal.at("operation_fingerprint") != fingerprint || !proposal.contains("proposal_digest")) {
                throw engine::Error("named_version.operation_reuse", "prepare operation ID was reused", 4);
            }
            static_cast<void>(load_proposal(store->directory_fd(), text(payload, "operation_id"),
                                            text(proposal, "proposal_digest")));
            return make_result(request.operation, compatibility, state, proposal.at("proposal_digest"),
                               nullptr, nullptr, false, false, engine::Json::array(), false);
        }
        const auto& decision = payload.at("authorization_decision");
        engine::Json proposal{
            {"protocol", proposal_protocol}, {"tops_id", payload.at("tops_id")},
            {"operation_id", payload.at("operation_id")}, {"operation_fingerprint", fingerprint},
            {"observed_registry_digest", expected}, {"named_version", payload.at("named_version")},
            {"validation_result", payload.at("validation_result")}, {"sav_engine", payload.at("sav_engine")},
            {"prepared_at", utc_now()},
            {"authority", engine::Json{{"decision_id", decision.at("decision_id")},
                {"capability_binding_digest", decision.at("capability").at("binding_digest")}}},
            {"canonical", false}, {"proposal_digest", nullptr},
        };
        finalize_digest(proposal, "proposal_digest");
        write_immutable(store->directory_fd(), proposal_name, proposal);
        return make_result(request.operation, compatibility, state, proposal.at("proposal_digest"),
                           nullptr, nullptr, true, false, engine::Json::array(), false);
    }
    if (recover) {
        if (state.present) {
            if (expected != "discover" && expected != current_digest(state)) {
                throw engine::Error("named_version.stale_expected_state", "recovery expected state is stale", 4);
            }
            return make_result(request.operation, compatibility, state, nullptr, nullptr, nullptr,
                               false, false, engine::Json::array(), false);
        }
        const auto selected = choose_recovery_candidate(store->directory_fd());
        if (expected != "discover" && expected != text(selected.registry, "registry_digest")) {
            throw engine::Error("named_version.stale_expected_state", "recovery expected state is stale", 4);
        }
        auto repaired = selected.registry;
        const auto previous = repaired.at("registry_digest");
        repaired["generation"] = number(repaired, "generation") + 1U;
        repaired["previous_registry_digest"] = previous;
        repaired["updated_at"] = utc_now();
        repaired["recovery"] = engine::Json{{"state", "recovered"}, {"recovered_from_digest", previous},
            {"detail", "selected one unique linked registry and republished the atomic head"}};
        append_operation(repaired, payload, nullptr, text(repaired, "updated_at"));
        state = commit_to_slot(store->directory_fd(), std::move(repaired), 1 - selected.slot, std::nullopt);
        return make_result(request.operation, compatibility, state, nullptr, nullptr, nullptr,
                           true, true, engine::Json::array({"selected_unique_forward_registry", "republished_head"}), false);
    }

    const auto fingerprint = operation_fingerprint(payload);
    if (state.present) {
        if (const auto prior = find_operation(state.registry, text(payload, "operation_id")); prior.has_value()) {
            if (prior->at("operation_fingerprint") != fingerprint) {
                throw engine::Error("named_version.operation_reuse", "operation ID was reused with different semantics", 4);
            }
            engine::Json artifact = nullptr;
            if (prior->at("selected_digest").is_string()) {
                const auto entry = find_version(state.registry, "digest", prior->at("selected_digest"));
                if (entry.has_value()) artifact = read_artifact(store->directory_fd(), *entry);
            }
            return make_result(request.operation, compatibility, state, nullptr, artifact,
                               alias ? payload.at("alias") : engine::Json(nullptr), false, false,
                               engine::Json::array(), false);
        }
    }
    require_expected(state, expected);

    const auto now = utc_now();
    if (seal) {
        const auto proposal = load_proposal(store->directory_fd(), text(payload, "prepared_operation_id"),
                                            text(payload, "proposal_digest"));
        if (proposal.at("tops_id") != payload.at("tops_id")) {
            throw engine::Error("named_version.proposal_invalid", "prepared proposal belongs to another TOPS", 4);
        }
        const auto& artifact = proposal.at("named_version");
        if (state.present) {
            if (const auto by_id = find_version(state.registry, "id", artifact.at("named_version_id"));
                by_id.has_value() && by_id->at("named_version_digest") != artifact.at("named_version_digest")) {
                throw engine::Error("named_version.identity_conflict", "Named Version ID already names another digest", 4);
            }
            if (const auto by_digest = find_version(state.registry, "digest", artifact.at("named_version_digest"));
                by_digest.has_value()) {
                return make_result(request.operation, compatibility, state, proposal.at("proposal_digest"),
                                   read_artifact(store->directory_fd(), *by_digest), nullptr,
                                   false, false, engine::Json::array(), false);
            }
            if (artifact.at("predecessor_digest").is_string() &&
                !find_version(state.registry, "digest", artifact.at("predecessor_digest")).has_value()) {
                throw engine::Error("named_version.predecessor_unresolved", "Named Version predecessor is not sealed", 4);
            }
            if (state.registry.at("versions").size() >= max_versions) {
                throw engine::Error("named_version.capacity_review_required", "version registry capacity requires review", 4);
            }
        }
        const auto object_name = "object." + text(artifact, "named_version_digest").substr(7U) + ".json";
        write_immutable(store->directory_fd(), object_name, artifact);
        engine::Json next = state.present ? state.registry : engine::Json{
            {"protocol", registry_protocol}, {"format_version", format_version}, {"tops_id", payload.at("tops_id")},
            {"generation", 0U}, {"previous_registry_digest", nullptr}, {"versions", engine::Json::array()},
            {"aliases", engine::Json::array()}, {"operations", engine::Json::array()}, {"updated_at", now},
            {"recovery", clean_recovery()}, {"canonical", false}, {"registry_digest", nullptr},
        };
        next["generation"] = state.present ? number(state.registry, "generation") + 1U : 1U;
        next["previous_registry_digest"] = state.present ? state.registry.at("registry_digest") : engine::Json(nullptr);
        next["updated_at"] = now;
        next["recovery"] = clean_recovery();
        next["versions"].push_back(engine::Json{
            {"named_version_id", artifact.at("named_version_id")},
            {"named_version_digest", artifact.at("named_version_digest")}, {"display_alias", artifact.at("alias")},
            {"predecessor_digest", artifact.at("predecessor_digest")}, {"sealed_at", artifact.at("sealed_at")},
            {"object_name", object_name},
        });
        append_operation(next, payload, artifact.at("named_version_digest"), now);
        state = commit(store->directory_fd(), std::move(next), state.present ?
            std::optional<engine::Json>(state.head) : std::nullopt);
        return make_result(request.operation, compatibility, state, proposal.at("proposal_digest"),
                           artifact, nullptr, true, false, engine::Json::array(), false);
    }

    if (!state.present) throw engine::Error("named_version.store_absent", "alias selection requires a registry", 4);
    const auto selector = validate_selector(payload.at("selector"));
    const auto entry = find_version(state.registry, "digest", text(selector, "value", 256U));
    if (!entry.has_value()) throw engine::Error("named_version.not_found", "alias target is not sealed", 4);
    if (state.registry.at("aliases").size() >= max_aliases &&
        std::none_of(state.registry.at("aliases").begin(), state.registry.at("aliases").end(),
            [&](const auto& item) { return item.at("alias") == payload.at("alias"); })) {
        throw engine::Error("named_version.capacity_review_required", "alias registry capacity requires review", 4);
    }
    auto next = state.registry;
    bool replaced = false;
    for (auto& item : next["aliases"]) {
        if (item.at("alias") == payload.at("alias")) {
            if (item.at("named_version_digest") == entry->at("named_version_digest")) {
                return make_result(request.operation, compatibility, state, nullptr,
                                   read_artifact(store->directory_fd(), *entry), payload.at("alias"),
                                   false, false, engine::Json::array(), false);
            }
            item = engine::Json{{"alias", payload.at("alias")}, {"named_version_id", entry->at("named_version_id")},
                {"named_version_digest", entry->at("named_version_digest")}, {"updated_at", now}};
            replaced = true;
            break;
        }
    }
    if (!replaced) {
        next["aliases"].push_back(engine::Json{{"alias", payload.at("alias")},
            {"named_version_id", entry->at("named_version_id")},
            {"named_version_digest", entry->at("named_version_digest")}, {"updated_at", now}});
    }
    next["generation"] = number(next, "generation") + 1U;
    next["previous_registry_digest"] = state.registry.at("registry_digest");
    next["updated_at"] = now;
    next["recovery"] = clean_recovery();
    append_operation(next, payload, entry->at("named_version_digest"), now);
    state = commit(store->directory_fd(), std::move(next), state.head);
    return make_result(request.operation, compatibility, state, nullptr,
                       read_artifact(store->directory_fd(), *entry), payload.at("alias"),
                       true, false, engine::Json::array(), false);
}

} // namespace symphony::knowledge::session
