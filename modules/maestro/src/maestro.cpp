#include "maestro.hpp"

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
#include <dirent.h>
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
#include <ctime>
#include <unistd.h>
#include <utility>
#include <vector>

namespace symphony::maestro {
namespace engine = symphony::knowledge::engine;
namespace fs = std::filesystem;

namespace {

constexpr const char* command_protocol = "symphony.maestro.knowledge-engine-docking.v1";
constexpr const char* result_protocol = "symphony.maestro.docking-result.v1";
constexpr const char* descriptor_protocol = "symphony.maestro.receptor-descriptor.v1";
constexpr const char* registry_protocol = "symphony.maestro.docking-presence-registry.v1";
constexpr const char* head_protocol = "symphony.maestro.docking-presence-head.v1";
constexpr const char* presence_protocol = "symphony.maestro.docking-presence.v1";
constexpr const char* inventory_command_protocol = "symphony.maestro.receptor-inventory-command.v1";
constexpr const char* inventory_protocol = "symphony.maestro.receptor-inventory.v1";
constexpr const char* inventory_result_protocol = "symphony.maestro.receptor-inventory-result.v1";
constexpr const char* receptor_kind = "symphony.maestro.knowledge-engine.v1";
constexpr const char* decision_protocol = "symphony.ssiag.authorization-decision.v1";
constexpr const char* capability_protocol = "symphony.ssiag.capability.v1";
constexpr std::uint64_t format_version = 1U;
constexpr std::size_t max_state_bytes = engine::Limits::max_response_bytes;
constexpr std::size_t max_components = 4096U;

const std::vector<std::string> required_capabilities = {
    "atomic-head-v1",
    "dual-slot-presence-v1",
    "exact-receipt-binding-v1",
    "expected-state-cas-v1",
    "idempotent-operation-v1",
    "recovery-forward-v1",
    "ssiag-capability-binding-v1",
};

const std::vector<std::string> optional_capabilities = {
    "discovery-recovery-v1",
    "derived-receptor-inventory-v1",
    "nonblocking-lock-v1",
};

constexpr const char* inventory_capability = "derived-receptor-inventory-v1";

[[noreturn]] void system_error(const std::string& code, const std::string& detail);

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

class PresenceLock final {
public:
    PresenceLock(FileDescriptor directory, FileDescriptor lock)
        : directory_(std::move(directory)), lock_(std::move(lock)) {}
    ~PresenceLock() {
        if (lock_.get() >= 0) static_cast<void>(::flock(lock_.get(), LOCK_UN));
    }
    PresenceLock(const PresenceLock&) = delete;
    PresenceLock& operator=(const PresenceLock&) = delete;
    PresenceLock(PresenceLock&&) = default;
    PresenceLock& operator=(PresenceLock&&) = default;
    [[nodiscard]] int directory_fd() const noexcept { return directory_.get(); }

private:
    FileDescriptor directory_;
    FileDescriptor lock_;
};

class DirectoryStream final {
public:
    explicit DirectoryStream(int directory) {
        const int duplicate = ::dup(directory);
        if (duplicate < 0) system_error("maestro.inventory_scan_failed", "could not duplicate receptors directory");
        value_ = ::fdopendir(duplicate);
        if (value_ == nullptr) {
            const int saved = errno;
            static_cast<void>(::close(duplicate));
            errno = saved;
            system_error("maestro.inventory_scan_failed", "could not enumerate receptors directory");
        }
    }
    ~DirectoryStream() { if (value_ != nullptr) static_cast<void>(::closedir(value_)); }
    DirectoryStream(const DirectoryStream&) = delete;
    DirectoryStream& operator=(const DirectoryStream&) = delete;
    [[nodiscard]] DIR* get() const noexcept { return value_; }

private:
    DIR* value_ = nullptr;
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

bool safe_token(std::string_view value, std::size_t maximum = 256U) {
    if (value.empty() || value.size() > maximum) return false;
    return std::all_of(value.begin(), value.end(), [](const unsigned char character) {
        const bool alpha = (character >= 'a' && character <= 'z') ||
            (character >= 'A' && character <= 'Z');
        return alpha || (character >= '0' && character <= '9') || character == '.' ||
            character == '_' || character == ':' || character == '-';
    });
}

bool safe_version(std::string_view value) {
    if (value.empty() || value.size() > 64U) return false;
    return std::all_of(value.begin(), value.end(), [](const unsigned char character) {
        const bool alpha = (character >= 'a' && character <= 'z') ||
            (character >= 'A' && character <= 'Z');
        return alpha || (character >= '0' && character <= '9') || character == '.' ||
            character == '+' || character == '-';
    });
}

bool tagged_digest(std::string_view value) {
    return value.size() == 71U && value.starts_with("sha256:") &&
        std::all_of(value.begin() + 7, value.end(), [](const unsigned char character) {
            return (character >= '0' && character <= '9') ||
                (character >= 'a' && character <= 'f');
        });
}

bool lowercase_uuid(std::string_view value) {
    if (value.size() != 36U || value[8] != '-' || value[13] != '-' || value[18] != '-' ||
        value[23] != '-' || value[14] < '1' || value[14] > '8' ||
        (value[19] != '8' && value[19] != '9' && value[19] != 'a' && value[19] != 'b')) {
        return false;
    }
    for (std::size_t index = 0; index < value.size(); ++index) {
        if (index == 8U || index == 13U || index == 18U || index == 23U) continue;
        const auto character = value[index];
        if (!((character >= '0' && character <= '9') ||
              (character >= 'a' && character <= 'f'))) return false;
    }
    return true;
}

bool safe_absolute_path(const std::string& value) {
    if (value.empty() || value.size() > engine::Limits::max_path_bytes || value == "/" ||
        value.front() != '/' || value.back() == '/' || value.find('\\') != std::string::npos ||
        value.find('\0') != std::string::npos) return false;
    for (const unsigned char character : value) {
        if (character < 0x20U || character == 0x7fU) return false;
    }
    const fs::path path(value);
    if (!path.is_absolute() || path.lexically_normal().string() != value) return false;
    for (const auto& item : path.relative_path()) {
        const auto component = item.string();
        if (component.empty() || component == "." || component == "..") return false;
    }
    return true;
}

void exact_fields(const engine::Json& object, const std::set<std::string>& fields,
                  const std::string& context) {
    if (!object.is_object() || object.size() != fields.size()) {
        throw engine::Error("maestro.field_set", context + " has an invalid field set", 4);
    }
    for (const auto& [key, value] : object.items()) {
        static_cast<void>(value);
        if (!fields.contains(key)) {
            throw engine::Error("maestro.unknown_field", context + " has an unknown field", 4);
        }
    }
}

std::string text(const engine::Json& object, const char* field, std::size_t maximum = 4096U) {
    if (!object.contains(field) || !object.at(field).is_string()) {
        throw engine::Error("maestro.invalid_field", std::string(field) + " must be a string", 4);
    }
    const auto value = object.at(field).get<std::string>();
    if (value.empty() || value.size() > maximum) {
        throw engine::Error("maestro.invalid_field", std::string(field) + " has invalid length", 4);
    }
    for (const unsigned char character : value) {
        if (character < 0x20U || character == 0x7fU) {
            throw engine::Error("maestro.invalid_field", std::string(field) + " contains unsafe text", 4);
        }
    }
    return value;
}

std::uint64_t number(const engine::Json& object, const char* field) {
    if (!object.contains(field) ||
        (!object.at(field).is_number_unsigned() && !object.at(field).is_number_integer())) {
        throw engine::Error("maestro.invalid_field", std::string(field) + " must be an integer", 4);
    }
    try {
        const auto value = object.at(field).get<std::uint64_t>();
        if (value > 9007199254740991ULL) {
            throw engine::Error("maestro.invalid_field", std::string(field) + " is out of range", 4);
        }
        return value;
    } catch (const nlohmann::json::exception&) {
        throw engine::Error("maestro.invalid_field", std::string(field) + " is out of range", 4);
    }
}

std::string utc_now() {
    const auto now = std::chrono::time_point_cast<std::chrono::seconds>(std::chrono::system_clock::now());
    const std::time_t value = std::chrono::system_clock::to_time_t(now);
    std::tm result {};
    if (::gmtime_r(&value, &result) == nullptr) {
        throw engine::Error("maestro.clock_failed", "could not obtain a UTC timestamp", 5);
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

std::vector<std::string> token_array(const engine::Json& value, const std::string& name,
                                     std::size_t maximum, bool nonempty = false) {
    if (!value.is_array() || value.size() > maximum || (nonempty && value.empty())) {
        throw engine::Error("maestro.invalid_field", name + " must be a bounded array", 4);
    }
    std::vector<std::string> result;
    std::set<std::string> seen;
    for (const auto& item : value) {
        if (!item.is_string() || !safe_token(item.get<std::string>()) ||
            !seen.insert(item.get<std::string>()).second) {
            throw engine::Error("maestro.invalid_field", name + " contains an invalid token", 4);
        }
        result.push_back(item.get<std::string>());
    }
    return result;
}

std::vector<std::uint64_t> version_array(const engine::Json& value, const std::string& name) {
    if (!value.is_array() || value.empty() || value.size() > 16U) {
        throw engine::Error("maestro.invalid_field", name + " must be a bounded nonempty array", 4);
    }
    std::vector<std::uint64_t> result;
    std::set<std::uint64_t> seen;
    for (const auto& item : value) {
        if (!item.is_number_integer() && !item.is_number_unsigned()) {
            throw engine::Error("maestro.invalid_field", name + " contains a non-integer", 4);
        }
        const auto version = item.get<std::uint64_t>();
        if (version == 0U || version > 16U || !seen.insert(version).second) {
            throw engine::Error("maestro.invalid_field", name + " contains an invalid version", 4);
        }
        result.push_back(version);
    }
    return result;
}

template <typename T>
bool contains(const std::vector<T>& values, const T& wanted) {
    return std::find(values.begin(), values.end(), wanted) != values.end();
}

std::string capability_binding(const engine::Json& capability) {
    const auto& subject = capability.at("subject");
    const auto& target = capability.at("target");
    const std::array<std::string, 19> values = {
        text(capability, "protocol"), text(subject, "id"), text(subject, "kind"),
        text(subject, "authority"), text(capability, "tops_id"), text(target, "operation"),
        text(target, "resource"), text(target, "audience"), text(target, "scope"),
        text(capability, "authority_basis"), text(capability, "grant_id"),
        text(capability, "request_id"), text(capability, "correlation_id"),
        text(capability, "issued_at"), text(capability, "expires_at"),
        text(capability, "policy_digest"), text(capability, "config_digest"),
        "transferable=false", "canonical_apply=false",
    };
    std::string joined;
    for (std::size_t index = 0; index < values.size(); ++index) {
        if (index != 0U) joined.push_back('\n');
        joined += values[index];
    }
    return engine::tagged_sha256(joined);
}

engine::Json validate_authorization(const engine::Json& decision,
                                    const std::string& expected_operation,
                                    const std::string& tops_id,
                                    const std::string& expected_resource) {
    exact_fields(decision, {
        "schema", "decision_id", "request_id", "correlation_id", "tops_id", "subject",
        "target", "effect", "reason_code", "authority_basis", "capability", "policy_digest",
        "config_digest", "decided_at", "expires_at", "caller_class_used", "canonical_apply",
    }, "SSIAG authorization decision");
    if (text(decision, "schema") != decision_protocol || text(decision, "effect") != "allow" ||
        !safe_token(text(decision, "decision_id")) || !safe_token(text(decision, "request_id")) ||
        !safe_token(text(decision, "correlation_id")) || !safe_token(text(decision, "reason_code")) ||
        !tagged_digest(text(decision, "policy_digest")) ||
        !tagged_digest(text(decision, "config_digest")) ||
        !engine::is_utc_seconds(text(decision, "decided_at")) || text(decision, "tops_id") != tops_id ||
        !decision.at("caller_class_used").is_boolean() ||
        decision.at("caller_class_used").get<bool>() ||
        !decision.at("canonical_apply").is_boolean() || decision.at("canonical_apply").get<bool>() ||
        !decision.at("capability").is_object() || !decision.at("authority_basis").is_string() ||
        !decision.at("expires_at").is_string()) {
        throw engine::Error("maestro.authorization_denied", "SSIAG decision does not carry an allowed non-apply capability", 4);
    }
    const auto& target = decision.at("target");
    exact_fields(target, {"operation", "resource", "audience", "scope"}, "SSIAG target");
    if (text(target, "operation") != expected_operation || text(target, "resource") != expected_resource ||
        text(target, "audience") != "qxctl" || text(target, "scope") != "tops:" + tops_id) {
        throw engine::Error("maestro.authorization_target_mismatch", "SSIAG decision target does not match the Maestro operation", 4);
    }
    const auto& capability = decision.at("capability");
    exact_fields(capability, {
        "protocol", "capability_id", "subject", "tops_id", "target", "authority_basis",
        "grant_id", "request_id", "correlation_id", "issued_at", "expires_at",
        "policy_digest", "config_digest", "binding_digest", "transferable", "canonical_apply",
    }, "SSIAG capability");
    const auto basis = text(decision, "authority_basis");
    if ((basis != "host_owner" && basis != "granted_permission") ||
        text(capability, "protocol") != capability_protocol || text(capability, "tops_id") != tops_id ||
        !safe_token(text(capability, "capability_id")) || !safe_token(text(capability, "grant_id")) ||
        capability.at("subject") != decision.at("subject") || capability.at("target") != target ||
        text(capability, "authority_basis") != basis ||
        text(capability, "request_id") != text(decision, "request_id") ||
        text(capability, "correlation_id") != text(decision, "correlation_id") ||
        text(capability, "policy_digest") != text(decision, "policy_digest") ||
        text(capability, "config_digest") != text(decision, "config_digest") ||
        text(capability, "issued_at") != text(decision, "decided_at") ||
        text(capability, "expires_at") != text(decision, "expires_at") ||
        !capability.at("transferable").is_boolean() || capability.at("transferable").get<bool>() ||
        !capability.at("canonical_apply").is_boolean() || capability.at("canonical_apply").get<bool>()) {
        throw engine::Error("maestro.capability_mismatch", "SSIAG capability does not bind the decision", 4);
    }
    const auto& subject = capability.at("subject");
    exact_fields(subject, {"id", "kind", "authority"}, "SSIAG subject");
    if (!safe_token(text(subject, "id")) || !safe_token(text(subject, "kind")) ||
        text(subject, "authority") != "unix_peer_credentials") {
        throw engine::Error("maestro.subject_invalid", "SSIAG subject evidence is invalid", 4);
    }
    const auto issued = text(capability, "issued_at");
    const auto expires = text(capability, "expires_at");
    const auto now = utc_now();
    const auto binding = text(capability, "binding_digest");
    if (!engine::is_utc_seconds(issued) || !engine::is_utc_seconds(expires) || issued > now ||
        expires <= now || issued >= expires || !tagged_digest(text(capability, "policy_digest")) ||
        !tagged_digest(text(capability, "config_digest")) || !tagged_digest(binding) ||
        capability_binding(capability) != binding ||
        text(capability, "capability_id") != "ssiag-capability:" + binding.substr(7U)) {
        throw engine::Error("maestro.capability_invalid", "SSIAG capability binding or lifetime is invalid", 4);
    }
    return capability;
}

engine::Json compatibility_result(const engine::Json& client, const engine::Json* registry) {
    exact_fields(client, {
        "client_id", "client_version", "process_protocols", "presence_read_versions",
        "presence_write_versions", "capabilities",
    }, "Maestro client");
    if (text(client, "client_id", 128U) != "qxctl" || !safe_version(text(client, "client_version", 64U))) {
        throw engine::Error("maestro.client_invalid", "Maestro client identity is invalid", 4);
    }
    const auto processes = token_array(client.at("process_protocols"), "process_protocols", 8U, true);
    const auto reads = version_array(client.at("presence_read_versions"), "presence_read_versions");
    const auto writes = version_array(client.at("presence_write_versions"), "presence_write_versions");
    const auto capabilities = token_array(client.at("capabilities"), "capabilities", 64U);
    std::vector<std::string> missing;
    for (const auto& required : required_capabilities) {
        if (!contains(capabilities, required)) missing.push_back(required);
    }
    const bool process_ok = contains(processes, std::string(engine::process_protocol_v1));
    const bool read_ok = contains(reads, format_version);
    const bool write_ok = contains(writes, format_version);
    const bool stored_ok = registry == nullptr || number(*registry, "format_version") == format_version;
    const bool full = process_ok && read_ok && write_ok && stored_ok && missing.empty();
    const bool readable = process_ok && read_ok && stored_ok;
    return engine::Json{
        {"mode", full ? "full" : (readable ? "read_only" : "blocked")},
        {"process_protocol", process_ok ? engine::Json(engine::process_protocol_v1) : engine::Json(nullptr)},
        {"presence_read_version", read_ok && stored_ok ? engine::Json(format_version) : engine::Json(nullptr)},
        {"presence_write_version", write_ok ? engine::Json(format_version) : engine::Json(nullptr)},
        {"missing_capabilities", missing}, {"two_way_procedural_compatibility", true},
        {"reason", full ? "client, Maestro, and stored presence share the full v1 contract" :
            (readable ? "presence is readable but mutation capability is incomplete" :
             "client and Maestro have no safe presence read overlap")},
    };
}

engine::Json inventory_compatibility_result(const engine::Json& client) {
    auto result = compatibility_result(client, nullptr);
    const auto capabilities = token_array(client.at("capabilities"), "capabilities", 64U);
    if (!contains(capabilities, std::string(inventory_capability))) {
        auto missing = result.at("missing_capabilities");
        missing.push_back(inventory_capability);
        std::sort(missing.begin(), missing.end());
        result["mode"] = "blocked";
        result["missing_capabilities"] = std::move(missing);
        result["reason"] = "client lacks the derived receptor inventory capability";
    } else if (result.at("mode") == "full") {
        result["reason"] = "client and Maestro share the read-only receptor inventory v1 contract";
    }
    return result;
}

engine::Json descriptor_for(const std::string& tops_id, const std::string& receptor_id) {
    engine::Json value{
        {"protocol", descriptor_protocol}, {"format_version", format_version},
        {"maestro_id", engine_id}, {"maestro_version", engine_version}, {"tops_id", tops_id},
        {"receptor_id", receptor_id}, {"receptor_kind", receptor_kind},
        {"process_protocols", engine::Json::array({engine::process_protocol_v1})},
        {"docking_protocols", engine::Json::array({command_protocol})},
        {"presence_read_versions", engine::Json::array({format_version})},
        {"presence_write_version", format_version},
        {"supported_component_kinds", engine::Json::array({"vector_engine"})},
        {"required_capabilities", required_capabilities}, {"optional_capabilities", optional_capabilities},
        {"limits", engine::Json{{"components", max_components},
                                {"request_bytes", engine::Limits::max_request_bytes},
                                {"response_bytes", engine::Limits::max_response_bytes}}},
        {"thermal_path", "freezing"}, {"execution_enabled", false},
        {"network_listener", false}, {"canonical", false},
    };
    finalize_digest(value, "descriptor_digest");
    return value;
}

void validate_component_evidence(const engine::Json& component) {
    exact_fields(component, {
        "component_id", "component_kind", "module_id", "vector_id", "engine_id",
        "receipt_digest", "executable_digest", "receptor_kind", "evidence_digest",
    }, "component evidence");
    if (!safe_token(text(component, "component_id")) ||
        text(component, "component_kind") != "vector_engine" ||
        !safe_token(text(component, "module_id")) ||
        !component.at("vector_id").is_string() || !safe_token(component.at("vector_id").get<std::string>()) ||
        !component.at("engine_id").is_string() || !safe_token(component.at("engine_id").get<std::string>()) ||
        !tagged_digest(text(component, "receipt_digest")) ||
        !tagged_digest(text(component, "executable_digest")) ||
        text(component, "receptor_kind") != receptor_kind) {
        throw engine::Error("maestro.component_invalid", "component evidence is invalid or unsupported", 4);
    }
    verify_digest(component, "evidence_digest", "maestro.component_invalid");
}

void validate_presence(const engine::Json& presence) {
    exact_fields(presence, {
        "protocol", "format_version", "tops_id", "receptor_id", "receptor_kind",
        "component_id", "component_kind", "module_id", "vector_id", "engine_id",
        "disposition", "receipt_digest", "executable_digest", "operation_id",
        "previous_presence_digest", "capability_binding_digest", "committed_at",
        "canonical", "presence_digest",
    }, "docking presence");
    if (text(presence, "protocol") != presence_protocol ||
        number(presence, "format_version") != format_version ||
        !lowercase_uuid(text(presence, "tops_id")) || !safe_token(text(presence, "receptor_id")) ||
        text(presence, "receptor_kind") != receptor_kind ||
        !safe_token(text(presence, "component_id")) ||
        text(presence, "component_kind") != "vector_engine" ||
        !safe_token(text(presence, "module_id")) ||
        !presence.at("vector_id").is_string() || !safe_token(presence.at("vector_id").get<std::string>()) ||
        !presence.at("engine_id").is_string() || !safe_token(presence.at("engine_id").get<std::string>()) ||
        (text(presence, "disposition") != "docked" && text(presence, "disposition") != "undocked") ||
        !tagged_digest(text(presence, "receipt_digest")) ||
        !tagged_digest(text(presence, "executable_digest")) ||
        !safe_token(text(presence, "operation_id")) ||
        !tagged_digest(text(presence, "capability_binding_digest")) ||
        !engine::is_utc_seconds(text(presence, "committed_at")) || presence.at("canonical") != false) {
        throw engine::Error("maestro.presence_invalid", "docking presence identity is invalid", 5);
    }
    if (!presence.at("previous_presence_digest").is_null() &&
        (!presence.at("previous_presence_digest").is_string() ||
         !tagged_digest(presence.at("previous_presence_digest").get<std::string>()))) {
        throw engine::Error("maestro.presence_invalid", "presence predecessor is invalid", 5);
    }
    verify_digest(presence, "presence_digest", "maestro.presence_invalid");
}

void validate_registry(const engine::Json& registry) {
    exact_fields(registry, {
        "protocol", "format_version", "tops_id", "receptor_id", "receptor_kind",
        "generation", "previous_registry_digest", "components", "extensions", "recovery",
        "updated_at", "canonical", "registry_digest",
    }, "presence registry");
    if (text(registry, "protocol") != registry_protocol ||
        number(registry, "format_version") != format_version ||
        !lowercase_uuid(text(registry, "tops_id")) || !safe_token(text(registry, "receptor_id")) ||
        text(registry, "receptor_kind") != receptor_kind || number(registry, "generation") == 0U ||
        !registry.at("components").is_array() || registry.at("components").size() > max_components ||
        !registry.at("extensions").is_array() || !registry.at("extensions").empty() ||
        !engine::is_utc_seconds(text(registry, "updated_at")) || registry.at("canonical") != false) {
        throw engine::Error("maestro.registry_invalid", "presence registry identity is invalid", 5);
    }
    if ((number(registry, "generation") == 1U) != registry.at("previous_registry_digest").is_null() ||
        (!registry.at("previous_registry_digest").is_null() &&
         (!registry.at("previous_registry_digest").is_string() ||
          !tagged_digest(registry.at("previous_registry_digest").get<std::string>())))) {
        throw engine::Error("maestro.registry_invalid", "registry generation and predecessor are inconsistent", 5);
    }
    exact_fields(registry.at("recovery"), {
        "state", "disposition", "recovered_from_digest", "detail",
    }, "registry recovery");
    const auto recovery_state = text(registry.at("recovery"), "state");
    if ((recovery_state != "clean" && recovery_state != "recovered") ||
        text(registry.at("recovery"), "detail").size() > 4096U ||
        (recovery_state == "clean") != registry.at("recovery").at("recovered_from_digest").is_null() ||
        (!registry.at("recovery").at("recovered_from_digest").is_null() &&
         (!registry.at("recovery").at("recovered_from_digest").is_string() ||
          !tagged_digest(registry.at("recovery").at("recovered_from_digest").get<std::string>())))) {
        throw engine::Error("maestro.registry_invalid", "registry recovery evidence is invalid", 5);
    }
    std::string prior;
    for (const auto& presence : registry.at("components")) {
        validate_presence(presence);
        const auto id = text(presence, "component_id");
        if (!prior.empty() && id <= prior) {
            throw engine::Error("maestro.registry_invalid", "presence components are not uniquely sorted", 5);
        }
        if (presence.at("tops_id") != registry.at("tops_id") ||
            presence.at("receptor_id") != registry.at("receptor_id") ||
            presence.at("receptor_kind") != registry.at("receptor_kind")) {
            throw engine::Error("maestro.registry_invalid", "presence component scope mismatches registry", 5);
        }
        prior = id;
    }
    verify_digest(registry, "registry_digest", "maestro.registry_invalid");
}

void validate_head(const engine::Json& head) {
    exact_fields(head, {
        "protocol", "format_version", "tops_id", "receptor_id", "active_slot", "generation",
        "registry_digest", "previous_head_digest", "updated_at", "head_digest",
    }, "presence head");
    if (text(head, "protocol") != head_protocol || number(head, "format_version") != format_version ||
        !lowercase_uuid(text(head, "tops_id")) || !safe_token(text(head, "receptor_id")) ||
        number(head, "active_slot") > 1U || number(head, "generation") == 0U ||
        !tagged_digest(text(head, "registry_digest")) ||
        !engine::is_utc_seconds(text(head, "updated_at"))) {
        throw engine::Error("maestro.head_invalid", "presence head identity is invalid", 5);
    }
    if (!head.at("previous_head_digest").is_null() &&
        (!head.at("previous_head_digest").is_string() ||
         !tagged_digest(head.at("previous_head_digest").get<std::string>()))) {
        throw engine::Error("maestro.head_invalid", "presence head predecessor is invalid", 5);
    }
    verify_digest(head, "head_digest", "maestro.head_invalid");
}

std::optional<FileDescriptor> open_absolute_directory(const std::string& path, bool create) {
    if (!safe_absolute_path(path)) {
        throw engine::Error("maestro.state_root_invalid", "state root must be a safe absolute descendant", 4);
    }
    FileDescriptor current(::open("/", O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC));
    if (current.get() < 0) system_error("maestro.state_root_open_failed", "could not open filesystem root");
    for (const auto& item : fs::path(path).relative_path()) {
        const auto component = item.string();
        int next = ::openat(current.get(), component.c_str(), O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
        if (next < 0 && errno == ENOENT && !create) return std::nullopt;
        if (next < 0 && errno == ENOENT && create) {
            if (::mkdirat(current.get(), component.c_str(), 0700) != 0 && errno != EEXIST) {
                system_error("maestro.state_directory_create_failed", "could not create state directory");
            }
            if (::fsync(current.get()) != 0) {
                system_error("maestro.state_sync_failed", "could not synchronize parent directory");
            }
            next = ::openat(current.get(), component.c_str(), O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
        }
        if (next < 0) system_error("maestro.state_directory_open_failed", "could not open state directory");
        current = FileDescriptor(next);
    }
    struct stat status {};
    if (::fstat(current.get(), &status) != 0) {
        system_error("maestro.state_directory_stat_failed", "could not inspect state root");
    }
    if (!S_ISDIR(status.st_mode) || status.st_uid != ::geteuid() || (status.st_mode & 0022) != 0) {
        throw engine::Error("maestro.state_directory_unsafe", "state root must be caller-owned and protected", 5);
    }
    return std::optional<FileDescriptor>(std::move(current));
}

std::optional<FileDescriptor> open_child_directory(FileDescriptor parent, const std::string& component,
                                                   bool create) {
    int next = ::openat(parent.get(), component.c_str(), O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
    if (next < 0 && errno == ENOENT && !create) return std::nullopt;
    if (next < 0 && errno == ENOENT && create) {
        if (::mkdirat(parent.get(), component.c_str(), 0700) != 0 && errno != EEXIST) {
            system_error("maestro.state_directory_create_failed", "could not create presence directory");
        }
        if (::fsync(parent.get()) != 0) {
            system_error("maestro.state_sync_failed", "could not synchronize presence parent");
        }
        next = ::openat(parent.get(), component.c_str(), O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
    }
    if (next < 0) system_error("maestro.state_directory_open_failed", "could not open presence directory");
    FileDescriptor result(next);
    struct stat status {};
    if (::fstat(result.get(), &status) != 0) {
        system_error("maestro.state_directory_stat_failed", "could not inspect presence directory");
    }
    if (!S_ISDIR(status.st_mode) || status.st_uid != ::geteuid() || (status.st_mode & 0022) != 0) {
        throw engine::Error("maestro.state_directory_unsafe", "presence directory must be caller-owned and protected", 5);
    }
    if (create && ::fchmod(result.get(), 0700) != 0) {
        system_error("maestro.state_directory_mode_failed", "could not restrict presence directory");
    }
    return std::optional<FileDescriptor>(std::move(result));
}

std::optional<PresenceLock> open_stream(const std::string& root, const std::string& tops_id,
                                        const std::string& receptor_id, bool exclusive, bool create) {
    auto opened_root = open_absolute_directory(root, create);
    if (!opened_root.has_value()) return std::nullopt;
    FileDescriptor current = std::move(*opened_root);
    const std::array<std::string, 8> components = {
        "symphony", "maestro", "docking", "v1", "tops", engine::sha256_hex("tops:" + tops_id),
        "receptors", engine::sha256_hex("receptor:" + receptor_id),
    };
    for (const auto& component : components) {
        auto child = open_child_directory(std::move(current), component, create);
        if (!child.has_value()) return std::nullopt;
        current = std::move(*child);
    }
    const int flags = (create ? O_RDWR | O_CREAT : O_RDONLY) | O_NONBLOCK | O_NOFOLLOW | O_CLOEXEC;
    const int raw = ::openat(current.get(), ".lock", flags, 0600);
    if (raw < 0 && errno == ENOENT && !create) return std::nullopt;
    if (raw < 0) system_error("maestro.lock_open_failed", "could not open presence lock");
    FileDescriptor lock(raw);
    struct stat status {};
    if (::fstat(lock.get(), &status) != 0 || !S_ISREG(status.st_mode) ||
        (status.st_mode & 0777) != 0600 || status.st_uid != ::geteuid() || status.st_nlink != 1) {
        throw engine::Error("maestro.lock_unsafe", "presence lock metadata is unsafe", 5);
    }
    const int operation = exclusive ? LOCK_EX | LOCK_NB : LOCK_SH | LOCK_NB;
    if (::flock(lock.get(), operation) != 0) {
        if (errno == EWOULDBLOCK) {
            throw engine::Error("maestro.lock_busy", "presence registry is busy", 4);
        }
        system_error("maestro.lock_failed", "could not lock presence registry");
    }
    return PresenceLock(std::move(current), std::move(lock));
}

std::optional<FileDescriptor> open_receptors_directory(const std::string& root,
                                                       const std::string& tops_id) {
    auto opened_root = open_absolute_directory(root, false);
    if (!opened_root.has_value()) return std::nullopt;
    FileDescriptor current = std::move(*opened_root);
    const std::array<std::string, 7> components = {
        "symphony", "maestro", "docking", "v1", "tops",
        engine::sha256_hex("tops:" + tops_id), "receptors",
    };
    for (const auto& component : components) {
        auto child = open_child_directory(std::move(current), component, false);
        if (!child.has_value()) return std::nullopt;
        current = std::move(*child);
    }
    return std::optional<FileDescriptor>(std::move(current));
}

bool lowercase_hex_key(std::string_view value) {
    return value.size() == 64U &&
        std::all_of(value.begin(), value.end(), [](const unsigned char character) {
            return (character >= '0' && character <= '9') ||
                (character >= 'a' && character <= 'f');
        });
}

std::vector<std::string> receptor_directory_keys(int directory) {
    DirectoryStream stream(directory);
    std::vector<std::string> keys;
    errno = 0;
    while (const auto* entry = ::readdir(stream.get())) {
        const std::string name(entry->d_name);
        if (name == "." || name == "..") continue;
        if (!lowercase_hex_key(name)) {
            throw engine::Error("maestro.inventory_unknown_state",
                                "receptors directory contains an unrecognized entry", 5);
        }
        struct stat status {};
        if (::fstatat(directory, name.c_str(), &status, AT_SYMLINK_NOFOLLOW) != 0) {
            system_error("maestro.inventory_scan_failed", "could not inspect receptor entry");
        }
        if (!S_ISDIR(status.st_mode) || status.st_uid != ::geteuid() ||
            (status.st_mode & 0777) != 0700) {
            throw engine::Error("maestro.inventory_unknown_state",
                                "receptor entry is not a protected caller-owned directory", 5);
        }
        keys.push_back(name);
        errno = 0;
    }
    if (errno != 0) {
        system_error("maestro.inventory_scan_failed", "could not finish receptor enumeration");
    }
    std::sort(keys.begin(), keys.end());
    if (std::adjacent_find(keys.begin(), keys.end()) != keys.end() || keys.size() > max_components) {
        throw engine::Error("maestro.inventory_bound", "receptor inventory exceeds its bound", 5);
    }
    return keys;
}

PresenceLock open_inventory_stream(int receptors_directory, const std::string& key) {
    const int parent = ::dup(receptors_directory);
    if (parent < 0) {
        system_error("maestro.inventory_scan_failed", "could not duplicate receptors directory");
    }
    auto child = open_child_directory(FileDescriptor(parent), key, false);
    if (!child.has_value()) {
        throw engine::Error("maestro.inventory_changed", "receptor entry disappeared during inventory", 5);
    }
    FileDescriptor directory = std::move(*child);
    const int raw = ::openat(directory.get(), ".lock", O_RDONLY | O_NONBLOCK | O_NOFOLLOW | O_CLOEXEC);
    if (raw < 0) system_error("maestro.lock_open_failed", "could not open receptor inventory lock");
    FileDescriptor lock(raw);
    struct stat status {};
    if (::fstat(lock.get(), &status) != 0 || !S_ISREG(status.st_mode) ||
        (status.st_mode & 0777) != 0600 || status.st_uid != ::geteuid() || status.st_nlink != 1) {
        throw engine::Error("maestro.lock_unsafe", "receptor inventory lock metadata is unsafe", 5);
    }
    if (::flock(lock.get(), LOCK_SH | LOCK_NB) != 0) {
        if (errno == EWOULDBLOCK) {
            throw engine::Error("maestro.lock_busy", "receptor inventory is changing; retry", 4);
        }
        system_error("maestro.lock_failed", "could not lock receptor inventory stream");
    }
    return PresenceLock(std::move(directory), std::move(lock));
}

std::optional<std::string> read_file(int directory, const std::string& name) {
    const int raw = ::openat(directory, name.c_str(), O_RDONLY | O_NONBLOCK | O_NOFOLLOW | O_CLOEXEC);
    if (raw < 0) {
        if (errno == ENOENT) return std::nullopt;
        system_error("maestro.state_file_open_failed", "could not open presence state file");
    }
    FileDescriptor file(raw);
    struct stat status {};
    if (::fstat(file.get(), &status) != 0) {
        system_error("maestro.state_file_stat_failed", "could not inspect presence state file");
    }
    if (!S_ISREG(status.st_mode) || (status.st_mode & 0777) != 0600 ||
        status.st_uid != ::geteuid() || status.st_nlink != 1 || status.st_size < 0 ||
        static_cast<std::uint64_t>(status.st_size) > max_state_bytes) {
        throw engine::Error("maestro.state_file_unsafe", "presence state file metadata is unsafe", 5);
    }
    std::string data;
    data.reserve(static_cast<std::size_t>(status.st_size));
    std::array<char, 16384> buffer {};
    for (;;) {
        const auto count = ::read(file.get(), buffer.data(), buffer.size());
        if (count < 0) {
            if (errno == EINTR) continue;
            system_error("maestro.state_file_read_failed", "could not read presence state file");
        }
        if (count == 0) break;
        if (data.size() + static_cast<std::size_t>(count) > max_state_bytes) {
            throw engine::Error("maestro.state_file_too_large", "presence state exceeds its bound", 5);
        }
        data.append(buffer.data(), static_cast<std::size_t>(count));
    }
    return data;
}

engine::Json parse_file(const std::string& data, const std::string& context) {
    try {
        return engine::parse_bounded_json(data, max_state_bytes);
    } catch (const engine::Error&) {
        throw engine::Error("maestro.state_json_invalid", context + " is invalid bounded JSON", 5);
    }
}

Candidate read_candidate(int directory, int slot) {
    Candidate candidate;
    candidate.slot = slot;
    const auto data = read_file(directory, "registry." + std::to_string(slot) + ".json");
    if (!data.has_value()) return candidate;
    candidate.exists = true;
    try {
        candidate.registry = parse_file(*data, "presence registry slot");
        if (!candidate.registry.is_object() || !candidate.registry.contains("protocol") ||
            !candidate.registry.contains("format_version") ||
            candidate.registry.at("protocol") != registry_protocol ||
            candidate.registry.at("format_version") != format_version) {
            candidate.incompatible = true;
            return candidate;
        }
        validate_registry(candidate.registry);
        candidate.valid = true;
    } catch (const engine::Error& error) {
        candidate.incompatible = error.code() == "maestro.compatibility_required";
    }
    return candidate;
}

std::optional<engine::Json> read_head(int directory, bool tolerate_invalid = false) {
    const auto data = read_file(directory, "head.json");
    if (!data.has_value()) return std::nullopt;
    try {
        auto head = parse_file(*data, "presence head");
        if (head.is_object() && head.contains("protocol") && head.contains("format_version") &&
            (head.at("protocol") != head_protocol || head.at("format_version") != format_version)) {
            throw engine::Error("maestro.compatibility_required",
                                "presence head uses an unsupported protocol version", 4);
        }
        validate_head(head);
        return head;
    } catch (const engine::Error& error) {
        if (tolerate_invalid && error.code() != "maestro.compatibility_required") return std::nullopt;
        throw;
    }
}

State load_state(int directory) {
    const auto head = read_head(directory);
    if (!head.has_value()) {
        const auto zero = read_candidate(directory, 0);
        const auto one = read_candidate(directory, 1);
        if (zero.exists || one.exists) {
            throw engine::Error("maestro.head_missing", "presence slots exist without a valid head; run recover", 5);
        }
        return {};
    }
    const int slot = static_cast<int>(number(*head, "active_slot"));
    const auto active = read_candidate(directory, slot);
    if (active.incompatible) {
        throw engine::Error("maestro.compatibility_required", "active slot contains unsupported state", 4);
    }
    if (!active.valid || active.registry.at("registry_digest") != head->at("registry_digest") ||
        active.registry.at("generation") != head->at("generation") ||
        active.registry.at("tops_id") != head->at("tops_id") ||
        active.registry.at("receptor_id") != head->at("receptor_id")) {
        throw engine::Error("maestro.head_slot_mismatch", "head does not select a matching presence registry; run recover", 5);
    }
    const auto inactive = read_candidate(directory, 1 - slot);
    if (inactive.incompatible) {
        throw engine::Error("maestro.compatibility_required", "inactive slot contains unsupported state", 4);
    }
    const auto active_generation = number(active.registry, "generation");
    if (!inactive.exists && active_generation > 1U) {
        throw engine::Error("maestro.recovery_required",
                            "the inactive predecessor slot is missing; run recover", 5);
    }
    if (inactive.exists && !inactive.valid) {
        throw engine::Error("maestro.recovery_required",
                            "the inactive predecessor slot is invalid; run recover", 5);
    }
    if (inactive.valid) {
        const auto inactive_generation = number(inactive.registry, "generation");
        if (inactive_generation == active_generation &&
            inactive.registry.at("registry_digest") != active.registry.at("registry_digest")) {
            throw engine::Error("maestro.recovery_ambiguous", "presence slots diverge at one generation", 5);
        }
        if (inactive_generation == active_generation) {
            throw engine::Error("maestro.recovery_required",
                                "presence slots duplicate one generation; run recover", 5);
        }
        if (inactive_generation > active_generation) {
            const bool linked = inactive_generation == active_generation + 1U &&
                inactive.registry.at("previous_registry_digest") == active.registry.at("registry_digest");
            throw engine::Error(linked ? "maestro.recovery_required" : "maestro.recovery_ambiguous",
                                linked ? "a linked successor exists beyond the selected head; run recover" :
                                         "a non-linked newer presence slot is ambiguous", 5);
        }
        if (active_generation != inactive_generation + 1U ||
            active.registry.at("previous_registry_digest") != inactive.registry.at("registry_digest")) {
            throw engine::Error("maestro.recovery_ambiguous",
                                "the selected registry and predecessor slot are not one linked chain", 5);
        }
    }
    return State{*head, active.registry, true};
}

engine::Json derive_receptor_inventory(const std::string& state_root,
                                       const std::string& tops_id) {
    engine::Json entries = engine::Json::array();
    std::uint64_t component_count = 0U;
    auto receptors = open_receptors_directory(state_root, tops_id);
    if (receptors.has_value()) {
        for (const auto& key : receptor_directory_keys(receptors->get())) {
            auto stream = open_inventory_stream(receptors->get(), key);
            const auto state = load_state(stream.directory_fd());
            if (!state.present) {
                throw engine::Error("maestro.inventory_unknown_state",
                                    "registered receptor directory has no selected state", 5);
            }
            const auto receptor_id = text(state.registry, "receptor_id");
            if (engine::sha256_hex("receptor:" + receptor_id) != key ||
                state.registry.at("tops_id") != tops_id) {
                throw engine::Error("maestro.inventory_scope_mismatch",
                                    "receptor directory identity does not match its registry", 5);
            }
            engine::Json docked = engine::Json::array();
            for (const auto& component : state.registry.at("components")) {
                if (component.at("disposition") == "docked") {
                    docked.push_back(engine::Json{
                        {"component_id", component.at("component_id")},
                        {"module_id", component.at("module_id")},
                        {"vector_id", component.at("vector_id")},
                        {"engine_id", component.at("engine_id")},
                        {"receipt_digest", component.at("receipt_digest")},
                        {"executable_digest", component.at("executable_digest")},
                        {"presence_digest", component.at("presence_digest")},
                    });
                }
            }
            std::sort(docked.begin(), docked.end(), [](const engine::Json& left, const engine::Json& right) {
                return left.at("component_id").get<std::string>() <
                    right.at("component_id").get<std::string>();
            });
            component_count += docked.size();
            if (component_count > max_components) {
                throw engine::Error("maestro.inventory_bound",
                                    "docked component inventory exceeds its bound", 5);
            }
            entries.push_back(engine::Json{
                {"receptor_id", receptor_id},
                {"receptor_kind", state.registry.at("receptor_kind")},
                {"registry_digest", state.registry.at("registry_digest")},
                {"generation", state.registry.at("generation")},
                {"registry_updated_at", state.registry.at("updated_at")},
                {"docked_components", std::move(docked)},
            });
        }
    }
    std::sort(entries.begin(), entries.end(), [](const engine::Json& left, const engine::Json& right) {
        return left.at("receptor_id").get<std::string>() <
            right.at("receptor_id").get<std::string>();
    });
    engine::Json inventory{
        {"protocol", inventory_protocol},
        {"format_version", format_version},
        {"tops_id", tops_id},
        {"receptors", std::move(entries)},
        {"receptor_count", 0U},
        {"docked_component_count", component_count},
        {"derived", true},
        {"canonical", false},
    };
    inventory["receptor_count"] = inventory.at("receptors").size();
    finalize_digest(inventory, "inventory_digest");
    return inventory;
}

void write_all(int file, const std::string& data) {
    std::size_t offset = 0U;
    while (offset < data.size()) {
        const auto count = ::write(file, data.data() + offset, data.size() - offset);
        if (count < 0) {
            if (errno == EINTR) continue;
            system_error("maestro.state_file_write_failed", "could not write presence state");
        }
        offset += static_cast<std::size_t>(count);
    }
}

void write_slot(int directory, int slot, const engine::Json& registry) {
    const auto name = "registry." + std::to_string(slot) + ".json";
    const int raw = ::openat(directory, name.c_str(),
        O_WRONLY | O_CREAT | O_NONBLOCK | O_NOFOLLOW | O_CLOEXEC, 0600);
    if (raw < 0) system_error("maestro.state_file_open_failed", "could not open inactive registry slot");
    FileDescriptor file(raw);
    struct stat status {};
    if (::fstat(file.get(), &status) != 0 || !S_ISREG(status.st_mode) ||
        status.st_uid != ::geteuid() || (status.st_mode & 0777) != 0600 || status.st_nlink != 1) {
        throw engine::Error("maestro.state_file_unsafe", "registry slot must be private and caller-owned", 5);
    }
    if (::ftruncate(file.get(), 0) != 0) {
        system_error("maestro.state_file_write_failed", "could not prepare registry slot");
    }
    write_all(file.get(), registry.dump() + "\n");
    if (::fsync(file.get()) != 0) {
        system_error("maestro.state_sync_failed", "could not synchronize registry slot");
    }
}

void write_head(int directory, engine::Json head) {
    finalize_digest(head, "head_digest");
    static std::atomic<std::uint64_t> sequence {0U};
    const auto temporary = ".head.tmp." + std::to_string(::getpid()) + "." +
        std::to_string(sequence.fetch_add(1U, std::memory_order_relaxed));
    const int raw = ::openat(directory, temporary.c_str(),
        O_WRONLY | O_CREAT | O_EXCL | O_NOFOLLOW | O_CLOEXEC, 0600);
    if (raw < 0) system_error("maestro.head_write_failed", "could not create temporary presence head");
    {
        FileDescriptor file(raw);
        write_all(file.get(), head.dump() + "\n");
        if (::fsync(file.get()) != 0) {
            system_error("maestro.state_sync_failed", "could not synchronize temporary presence head");
        }
    }
    if (::renameat(directory, temporary.c_str(), directory, "head.json") != 0) {
        static_cast<void>(::unlinkat(directory, temporary.c_str(), 0));
        system_error("maestro.head_commit_failed", "could not atomically replace presence head");
    }
    if (::fsync(directory) != 0) {
        system_error("maestro.state_sync_failed", "could not synchronize presence directory");
    }
}

State commit_to_slot(int directory, engine::Json registry, int slot,
                     const std::optional<engine::Json>& prior_head) {
    finalize_digest(registry, "registry_digest");
    validate_registry(registry);
    write_slot(directory, slot, registry);
    if (::fsync(directory) != 0) {
        system_error("maestro.state_sync_failed", "could not synchronize presence directory");
    }
    engine::Json head{
        {"protocol", head_protocol}, {"format_version", format_version},
        {"tops_id", registry.at("tops_id")}, {"receptor_id", registry.at("receptor_id")},
        {"active_slot", slot}, {"generation", registry.at("generation")},
        {"registry_digest", registry.at("registry_digest")},
        {"previous_head_digest", prior_head.has_value() ? prior_head->at("head_digest") : engine::Json(nullptr)},
        {"updated_at", registry.at("updated_at")},
    };
    write_head(directory, head);
    return State{*read_head(directory), std::move(registry), true};
}

State commit(int directory, engine::Json registry, const std::optional<engine::Json>& prior_head) {
    const int slot = prior_head.has_value() ?
        1 - static_cast<int>(number(*prior_head, "active_slot")) : 0;
    return commit_to_slot(directory, std::move(registry), slot, prior_head);
}

std::string docking_resource(const std::string& tops_id, const std::string& receptor_id,
                             const std::string& operation, const std::string& component_id,
                             const std::string& receipt_digest, const std::string& expected) {
    return "symphony.maestro.docking:" + engine::sha256_hex(
        tops_id + "\n" + receptor_id + "\n" + operation + "\n" + component_id + "\n" +
        receipt_digest + "\n" + expected);
}

std::string inventory_resource(const std::string& tops_id) {
    return "symphony.maestro.receptor-inventory:" +
        engine::sha256_hex(tops_id + "\nall\ninventory-v1");
}

engine::Json select_presence(const engine::Json& registry, const std::string& component_id) {
    for (const auto& presence : registry.at("components")) {
        if (presence.at("component_id") == component_id) return presence;
    }
    return nullptr;
}

bool same_component_identity(const engine::Json& presence, const engine::Json& component) {
    static const std::array<const char*, 7> fields = {
        "component_id", "component_kind", "module_id", "vector_id", "engine_id",
        "receipt_digest", "executable_digest",
    };
    return std::all_of(fields.begin(), fields.end(), [&](const char* field) {
        return presence.at(field) == component.at(field);
    });
}

engine::Json make_result(const std::string& operation, const std::string& tops_id,
                         const std::string& receptor_id, const engine::Json& compatibility,
                         const State& state, const std::optional<std::string>& component_id,
                         const std::string& outcome, bool changed, bool recovered,
                         engine::Json repair_actions, bool read_only) {
    engine::Json presence = nullptr;
    bool presence_present = false;
    if (state.present && component_id.has_value()) {
        presence = select_presence(state.registry, *component_id);
        presence_present = !presence.is_null() && presence.at("disposition") == "docked";
    }
    return engine::Json{
        {"protocol", result_protocol}, {"format_version", format_version}, {"operation", operation},
        {"tops_id", tops_id}, {"receptor_id", receptor_id},
        {"compatibility", compatibility}, {"descriptor", descriptor_for(tops_id, receptor_id)},
        {"registry_present", state.present},
        {"registry", state.present ? state.registry : engine::Json(nullptr)},
        {"registry_digest", state.present ? state.registry.at("registry_digest") : engine::Json(nullptr)},
        {"presence_present", presence_present}, {"presence", presence}, {"outcome", outcome},
        {"changed", changed}, {"recovered", recovered}, {"repair_actions", std::move(repair_actions)},
        {"read_only", read_only}, {"execution_enabled", false}, {"canonical", false},
    };
}

engine::Json make_inventory_result(const std::string& tops_id,
                                   const engine::Json& compatibility,
                                   engine::Json inventory) {
    engine::Json result{
        {"protocol", inventory_result_protocol},
        {"format_version", format_version},
        {"operation", "inventory"},
        {"tops_id", tops_id},
        {"compatibility", compatibility},
        {"inventory", std::move(inventory)},
        {"observed_at", utc_now()},
        {"read_only", true},
        {"derived", true},
        {"canonical", false},
    };
    finalize_digest(result, "observation_digest");
    return result;
}

engine::Json clean_recovery() {
    return engine::Json{
        {"state", "clean"}, {"disposition", "not_applicable"},
        {"recovered_from_digest", nullptr}, {"detail", "no recovery was required"},
    };
}

engine::Json mutate_registry(const State& state, const engine::Json& component,
                             const std::string& tops_id, const std::string& receptor_id,
                             const std::string& disposition, const std::string& operation_id,
                             const std::string& capability_binding_digest) {
    engine::Json components = state.present ? state.registry.at("components") : engine::Json::array();
    engine::Json previous_presence = nullptr;
    std::size_t existing_index = components.size();
    for (std::size_t index = 0; index < components.size(); ++index) {
        if (components[index].at("component_id") == component.at("component_id")) {
            existing_index = index;
            previous_presence = components[index];
            break;
        }
    }
    engine::Json presence{
        {"protocol", presence_protocol}, {"format_version", format_version}, {"tops_id", tops_id},
        {"receptor_id", receptor_id}, {"receptor_kind", receptor_kind},
        {"component_id", component.at("component_id")},
        {"component_kind", component.at("component_kind")}, {"module_id", component.at("module_id")},
        {"vector_id", component.at("vector_id")}, {"engine_id", component.at("engine_id")},
        {"disposition", disposition}, {"receipt_digest", component.at("receipt_digest")},
        {"executable_digest", component.at("executable_digest")}, {"operation_id", operation_id},
        {"previous_presence_digest", previous_presence.is_null() ? engine::Json(nullptr) :
            previous_presence.at("presence_digest")},
        {"capability_binding_digest", capability_binding_digest}, {"committed_at", utc_now()},
        {"canonical", false},
    };
    finalize_digest(presence, "presence_digest");
    if (existing_index == components.size()) components.push_back(presence);
    else components[existing_index] = presence;
    std::sort(components.begin(), components.end(), [](const auto& left, const auto& right) {
        return left.at("component_id").template get<std::string>() <
            right.at("component_id").template get<std::string>();
    });
    engine::Json registry{
        {"protocol", registry_protocol}, {"format_version", format_version}, {"tops_id", tops_id},
        {"receptor_id", receptor_id}, {"receptor_kind", receptor_kind},
        {"generation", state.present ? number(state.registry, "generation") + 1U : 1U},
        {"previous_registry_digest", state.present ? state.registry.at("registry_digest") : engine::Json(nullptr)},
        {"components", std::move(components)}, {"extensions", engine::Json::array()},
        {"recovery", clean_recovery()}, {"updated_at", utc_now()}, {"canonical", false},
    };
    finalize_digest(registry, "registry_digest");
    return registry;
}

Candidate choose_recovery_candidate(int directory) {
    const auto zero = read_candidate(directory, 0);
    const auto one = read_candidate(directory, 1);
    if (zero.incompatible || one.incompatible) {
        throw engine::Error("maestro.compatibility_required", "presence recovery found unsupported state", 4);
    }
    std::vector<Candidate> valid;
    if (zero.valid) valid.push_back(zero);
    if (one.valid) valid.push_back(one);
    if (valid.empty()) {
        throw engine::Error("maestro.recovery_unavailable", "no valid presence registry can be recovered", 5);
    }
    if (valid.size() == 1U) return valid.front();
    const auto zero_generation = number(valid[0].registry, "generation");
    const auto one_generation = number(valid[1].registry, "generation");
    if (zero_generation == one_generation) {
        if (valid[0].registry.at("registry_digest") == valid[1].registry.at("registry_digest")) {
            return valid[0];
        }
        throw engine::Error("maestro.recovery_ambiguous", "presence slots diverge at one generation", 5);
    }
    const Candidate& newer = zero_generation > one_generation ? valid[0] : valid[1];
    const Candidate& older = zero_generation > one_generation ? valid[1] : valid[0];
    if (number(newer.registry, "generation") != number(older.registry, "generation") + 1U ||
        newer.registry.at("previous_registry_digest") != older.registry.at("registry_digest")) {
        throw engine::Error("maestro.recovery_ambiguous", "presence slots do not form one linked forward chain", 5);
    }
    return newer;
}

engine::Json validate_inventory_command(const engine::Request& request) {
    exact_fields(request.payload, {
        "protocol", "format_version", "operation", "state_root", "tops_id",
        "authorization_decision", "client",
    }, "Maestro receptor inventory command");
    if (request.operation != "inventory" ||
        text(request.payload, "protocol") != inventory_command_protocol ||
        number(request.payload, "format_version") != format_version ||
        text(request.payload, "operation") != request.operation ||
        !request.payload.at("state_root").is_string() ||
        !safe_absolute_path(request.payload.at("state_root").get<std::string>()) ||
        !lowercase_uuid(text(request.payload, "tops_id")) ||
        !request.payload.at("authorization_decision").is_object()) {
        throw engine::Error("maestro.inventory_command_invalid",
                            "receptor inventory command is invalid", 4);
    }
    auto compatibility = inventory_compatibility_result(request.payload.at("client"));
    if (compatibility.at("mode") != "full") {
        throw engine::Error("maestro.compatibility_required",
                            "receptor inventory requires explicit read compatibility", 4);
    }
    return compatibility;
}

engine::Json validate_command(const engine::Request& request) {
    exact_fields(request.payload, {
        "protocol", "format_version", "operation", "state_root", "tops_id", "receptor_id",
        "operation_id", "expected_registry_digest", "component", "authorization_decision", "client",
    }, "Maestro docking command");
    if (text(request.payload, "protocol") != command_protocol ||
        number(request.payload, "format_version") != format_version ||
        text(request.payload, "operation") != request.operation ||
        !lowercase_uuid(text(request.payload, "tops_id")) ||
        !safe_token(text(request.payload, "receptor_id"))) {
        throw engine::Error("maestro.command_invalid", "docking command identity is invalid", 4);
    }
    static const std::set<std::string> operations = {"inspect", "status", "dock", "undock", "recover"};
    if (!operations.contains(request.operation)) {
        throw engine::Error("operation.unsupported", "unsupported Maestro operation", 2);
    }
    return compatibility_result(request.payload.at("client"), nullptr);
}

} // namespace

engine::Json descriptor(const std::string& receptor_id) {
    if (!safe_token(receptor_id)) {
        throw engine::Error("maestro.receptor_invalid", "receptor identity is invalid", 4);
    }
    return descriptor_for("00000000-0000-1000-8000-000000000000", receptor_id);
}

engine::Json handle_request(const engine::Request& request) {
    if (request.operation == "inventory") {
        const auto compatibility = validate_inventory_command(request);
        const auto tops_id = text(request.payload, "tops_id");
        static_cast<void>(validate_authorization(
            request.payload.at("authorization_decision"),
            "symphony.maestro.receptor-inventory.read", tops_id,
            inventory_resource(tops_id)));
        auto inventory = derive_receptor_inventory(
            request.payload.at("state_root").get<std::string>(), tops_id);
        return make_inventory_result(tops_id, compatibility, std::move(inventory));
    }
    auto compatibility = validate_command(request);
    const auto tops_id = text(request.payload, "tops_id");
    const auto receptor_id = text(request.payload, "receptor_id");

    if (request.operation == "inspect") {
        if (!request.payload.at("state_root").is_null() || !request.payload.at("operation_id").is_null() ||
            !request.payload.at("expected_registry_digest").is_null() ||
            !request.payload.at("component").is_null() ||
            !request.payload.at("authorization_decision").is_null()) {
            throw engine::Error("maestro.command_invalid", "inspect carries mutation or state fields", 4);
        }
        return make_result(request.operation, tops_id, receptor_id, compatibility, {}, std::nullopt,
                           "inspected", false, false, engine::Json::array(), true);
    }

    if (!request.payload.at("state_root").is_string() ||
        !safe_absolute_path(request.payload.at("state_root").get<std::string>())) {
        throw engine::Error("maestro.state_root_invalid", "operation requires a safe absolute state root", 4);
    }
    const auto state_root = request.payload.at("state_root").get<std::string>();

    if (request.operation == "status") {
        if (!request.payload.at("operation_id").is_null() ||
            !request.payload.at("expected_registry_digest").is_null() ||
            (!request.payload.at("component").is_null() &&
             (!request.payload.at("component").is_object() ||
              request.payload.at("component").size() != 1U ||
              !request.payload.at("component").contains("component_id") ||
              !request.payload.at("component").at("component_id").is_string() ||
              !safe_token(request.payload.at("component").at("component_id").get<std::string>()))) ||
            !request.payload.at("authorization_decision").is_object()) {
            throw engine::Error("maestro.command_invalid", "status fields are invalid", 4);
        }
        const std::optional<std::string> component_id = request.payload.at("component").is_null() ?
            std::nullopt : std::optional<std::string>(
                request.payload.at("component").at("component_id").get<std::string>());
        const auto filter = component_id.has_value() ? *component_id : "all";
        const auto resource = docking_resource(tops_id, receptor_id, "status", filter, "none", "status");
        static_cast<void>(validate_authorization(
            request.payload.at("authorization_decision"), "symphony.maestro.docking.status",
            tops_id, resource));
        auto stream = open_stream(state_root, tops_id, receptor_id, false, false);
        State state;
        if (stream.has_value()) state = load_state(stream->directory_fd());
        compatibility = compatibility_result(request.payload.at("client"), state.present ? &state.registry : nullptr);
        return make_result(request.operation, tops_id, receptor_id, compatibility, state, component_id,
                           "status", false, false, engine::Json::array(), true);
    }

    if (request.operation == "dock" || request.operation == "undock") {
        if (!request.payload.at("operation_id").is_string() ||
            !safe_token(request.payload.at("operation_id").get<std::string>()) ||
            !request.payload.at("expected_registry_digest").is_string() ||
            (request.payload.at("expected_registry_digest") != "absent" &&
             !tagged_digest(request.payload.at("expected_registry_digest").get<std::string>())) ||
            !request.payload.at("component").is_object() ||
            !request.payload.at("authorization_decision").is_object()) {
            throw engine::Error("maestro.command_invalid", "docking mutation fields are invalid", 4);
        }
        validate_component_evidence(request.payload.at("component"));
        const auto& component = request.payload.at("component");
        const auto component_id = text(component, "component_id");
        const auto receipt_digest = text(component, "receipt_digest");
        const auto expected = text(request.payload, "expected_registry_digest");
        const auto resource = docking_resource(
            tops_id, receptor_id, request.operation, component_id, receipt_digest, expected);
        const auto capability = validate_authorization(
            request.payload.at("authorization_decision"),
            "symphony.maestro.docking." + request.operation, tops_id, resource);
        auto stream = open_stream(state_root, tops_id, receptor_id, true, true);
        auto state = load_state(stream->directory_fd());
        compatibility = compatibility_result(request.payload.at("client"), state.present ? &state.registry : nullptr);
        if (compatibility.at("mode") != "full") {
            throw engine::Error("maestro.compatibility_required", "presence mutation requires full protocol overlap", 4);
        }
        const auto existing = state.present ? select_presence(state.registry, component_id) : engine::Json(nullptr);
        const auto target_disposition = request.operation == "dock" ? "docked" : "undocked";
        const bool identity_matches = !existing.is_null() && same_component_identity(existing, component);
        if (request.operation == "dock" && !existing.is_null() &&
            existing.at("disposition") == "docked" && !identity_matches) {
            throw engine::Error("maestro.transition_required",
                                "a different live component identity must be undocked first", 4);
        }
        if (request.operation == "undock" && !existing.is_null() &&
            existing.at("disposition") == "docked" && !identity_matches) {
            throw engine::Error("maestro.component_mismatch",
                                "undock evidence does not match the live component identity", 4);
        }
        const bool already_applied = request.operation == "dock" ?
            (!existing.is_null() && existing.at("disposition") == target_disposition && identity_matches) :
            (existing.is_null() || existing.at("disposition") == target_disposition);
        if (already_applied) {
            return make_result(request.operation, tops_id, receptor_id, compatibility, state, component_id,
                               "already_applied", false, false, engine::Json::array(), false);
        }
        if ((!state.present && expected != "absent") ||
            (state.present && expected != state.registry.at("registry_digest").get<std::string>())) {
            throw engine::Error("maestro.stale_expected_state", "expected presence registry does not match current state", 4);
        }
        auto next = mutate_registry(
            state, component, tops_id, receptor_id, target_disposition,
            text(request.payload, "operation_id"), text(capability, "binding_digest"));
        auto committed = commit(stream->directory_fd(), std::move(next),
                                state.present ? std::optional<engine::Json>(state.head) : std::nullopt);
        return make_result(request.operation, tops_id, receptor_id, compatibility, committed, component_id,
                           "committed", true, false, engine::Json::array(), false);
    }

    if (!request.payload.at("operation_id").is_string() ||
        !safe_token(request.payload.at("operation_id").get<std::string>()) ||
        !request.payload.at("expected_registry_digest").is_string() ||
        (request.payload.at("expected_registry_digest") != "discover" &&
         !tagged_digest(request.payload.at("expected_registry_digest").get<std::string>())) ||
        !request.payload.at("component").is_null() ||
        !request.payload.at("authorization_decision").is_object()) {
        throw engine::Error("maestro.command_invalid", "recovery fields are invalid", 4);
    }
    const auto expected = text(request.payload, "expected_registry_digest");
    const auto resource = docking_resource(tops_id, receptor_id, "recover", "all", "none", expected);
    const auto capability = validate_authorization(
        request.payload.at("authorization_decision"), "symphony.maestro.docking.recover",
        tops_id, resource);
    auto stream = open_stream(state_root, tops_id, receptor_id, true, false);
    if (!stream.has_value()) {
        return make_result(request.operation, tops_id, receptor_id, compatibility, {}, std::nullopt,
                           "absent", false, false, engine::Json::array(), false);
    }
    try {
        auto healthy = load_state(stream->directory_fd());
        if (expected != "discover" &&
            expected != healthy.registry.at("registry_digest").get<std::string>()) {
            throw engine::Error("maestro.stale_expected_state", "expected recovery state does not match current registry", 4);
        }
        compatibility = compatibility_result(request.payload.at("client"), &healthy.registry);
        return make_result(request.operation, tops_id, receptor_id, compatibility, healthy, std::nullopt,
                           "not_required", false, false, engine::Json::array(), false);
    } catch (const engine::Error& error) {
        static const std::set<std::string> recoverable_load_failures = {
            "maestro.head_invalid",
            "maestro.head_missing",
            "maestro.head_slot_mismatch",
            "maestro.recovery_required",
        };
        if (!recoverable_load_failures.contains(error.code())) throw;
    }
    const auto selected = choose_recovery_candidate(stream->directory_fd());
    if (selected.registry.at("tops_id") != tops_id || selected.registry.at("receptor_id") != receptor_id) {
        throw engine::Error("maestro.recovery_ambiguous", "recoverable registry belongs to another scope", 5);
    }
    if (expected != "discover" &&
        expected != selected.registry.at("registry_digest").get<std::string>()) {
        throw engine::Error("maestro.stale_expected_state", "expected recovery state does not match selected evidence", 4);
    }
    auto recovered = selected.registry;
    const auto selected_digest = recovered.at("registry_digest");
    recovered["generation"] = number(recovered, "generation") + 1U;
    recovered["previous_registry_digest"] = selected_digest;
    recovered["updated_at"] = utc_now();
    recovered["recovery"] = engine::Json{
        {"state", "recovered"}, {"disposition", "selected_unique_forward_state"},
        {"recovered_from_digest", selected_digest},
        {"detail", "programmatically selected one unique digest-linked registry and republished the head"},
    };
    finalize_digest(recovered, "registry_digest");
    const auto old_head = read_head(stream->directory_fd(), true);
    auto committed = commit_to_slot(stream->directory_fd(), std::move(recovered), 1 - selected.slot, old_head);
    compatibility = compatibility_result(request.payload.at("client"), &committed.registry);
    engine::Json actions = engine::Json::array({
        "selected one unique digest-linked presence registry",
        "committed a forward recovery generation and atomically republished the head",
    });
    static_cast<void>(capability);
    return make_result(request.operation, tops_id, receptor_id, compatibility, committed, std::nullopt,
                       "recovered", true, true, std::move(actions), false);
}

} // namespace symphony::maestro
