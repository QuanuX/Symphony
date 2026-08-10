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
#include <optional>
#include <set>
#include <string>
#include <string_view>
#include <sys/file.h>
#include <sys/stat.h>
#include <ctime>
#include <unistd.h>
#include <utility>
#include <vector>

namespace symphony::knowledge::session {
namespace engine = symphony::knowledge::engine;
namespace fs = std::filesystem;

namespace {

constexpr const char* journal_protocol = "symphony.knowledge.session-journal.v1";
constexpr const char* head_protocol = "symphony.knowledge.session-head.v1";
constexpr const char* command_protocol = "symphony.knowledge.session-command.v1";
constexpr const char* result_protocol = "symphony.knowledge.session-result.v1";
constexpr const char* decision_protocol = "symphony.ssiag.authorization-decision.v1";
constexpr const char* capability_protocol = "symphony.ssiag.capability.v1";
constexpr std::uint64_t format_version = 1U;
constexpr std::size_t max_state_bytes = engine::Limits::max_response_bytes;
constexpr std::size_t max_checkpoints = 256U;

const std::vector<std::string> required_capabilities = {
    "atomic-head-v1",
    "authority-epoch-v1",
    "dual-slot-journal-v1",
    "expected-state-cas-v1",
    "idempotent-operation-v1",
    "opaque-extension-preservation-v1",
    "recovery-forward-v1",
    "ssiag-capability-binding-v1",
};

const std::vector<std::string> optional_capabilities = {
    "discovery-recovery-v1",
    "nonblocking-lock-v1",
};

class FileDescriptor final {
public:
    explicit FileDescriptor(int value = -1) : value_(value) {}
    ~FileDescriptor() { if (value_ >= 0) ::close(value_); }
    FileDescriptor(const FileDescriptor&) = delete;
    FileDescriptor& operator=(const FileDescriptor&) = delete;
    FileDescriptor(FileDescriptor&& other) noexcept : value_(std::exchange(other.value_, -1)) {}
    FileDescriptor& operator=(FileDescriptor&& other) noexcept {
        if (this != &other) {
            if (value_ >= 0) ::close(value_);
            value_ = std::exchange(other.value_, -1);
        }
        return *this;
    }
    [[nodiscard]] int get() const noexcept { return value_; }
private:
    int value_;
};

class SessionLock final {
public:
    SessionLock(FileDescriptor directory, FileDescriptor lock)
        : directory_(std::move(directory)), lock_(std::move(lock)) {}
    ~SessionLock() { if (lock_.get() >= 0) static_cast<void>(::flock(lock_.get(), LOCK_UN)); }
    SessionLock(const SessionLock&) = delete;
    SessionLock& operator=(const SessionLock&) = delete;
    SessionLock(SessionLock&&) = default;
    SessionLock& operator=(SessionLock&&) = default;
    [[nodiscard]] int directory_fd() const noexcept { return directory_.get(); }
private:
    FileDescriptor directory_;
    FileDescriptor lock_;
};

struct State final {
    engine::Json head;
    engine::Json journal;
    bool present = false;
};

struct Candidate final {
    int slot = -1;
    engine::Json journal;
    bool exists = false;
    bool valid = false;
    bool incompatible = false;
};

[[noreturn]] void system_error(const std::string& code, const std::string& detail) {
    const int saved = errno;
    throw engine::Error(code, detail + ": " + std::strerror(saved), 5);
}

bool safe_token(std::string_view value, std::size_t maximum = 256U) {
    if (value.empty() || value.size() > maximum) return false;
    return std::all_of(value.begin(), value.end(), [](const unsigned char c) {
        const bool alpha = (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z');
        return alpha || (c >= '0' && c <= '9') || c == '.' || c == '_' || c == ':' || c == '-';
    });
}

bool tagged_digest(std::string_view value) {
    return value.size() == 71U && value.starts_with("sha256:") &&
        std::all_of(value.begin() + 7, value.end(), [](const unsigned char c) {
            return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f');
        });
}

bool lowercase_uuid(std::string_view value) {
    if (value.size() != 36U || value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' ||
        value[14] < '1' || value[14] > '8' || (value[19] != '8' && value[19] != '9' && value[19] != 'a' && value[19] != 'b')) {
        return false;
    }
    for (std::size_t index = 0; index < value.size(); ++index) {
        if (index == 8U || index == 13U || index == 18U || index == 23U) continue;
        const auto character = value[index];
        if (!((character >= '0' && character <= '9') || (character >= 'a' && character <= 'f'))) return false;
    }
    return true;
}

std::string utc_now() {
    const auto now = std::chrono::time_point_cast<std::chrono::seconds>(std::chrono::system_clock::now());
    const std::time_t value = std::chrono::system_clock::to_time_t(now);
    std::tm result {};
    gmtime_r(&value, &result);
    char buffer[21] {};
    if (std::strftime(buffer, sizeof(buffer), "%Y-%m-%dT%H:%M:%SZ", &result) != 20U) {
        throw engine::Error("session.clock_failed", "could not format the current UTC time", 5);
    }
    return buffer;
}

bool safe_absolute_path(const std::string& value) {
    if (value.empty() || value.size() > engine::Limits::max_path_bytes || value.front() != '/' ||
        (value.size() > 1U && value.back() == '/') || value.find('\\') != std::string::npos ||
        value.find('\0') != std::string::npos) return false;
    const fs::path path(value);
    if (!path.is_absolute() || path.lexically_normal().string() != value) return false;
    for (const auto& part_value : path.relative_path()) {
        const auto part = part_value.string();
        if (part.empty() || part == "." || part == "..") return false;
    }
    return std::all_of(value.begin(), value.end(), [](const unsigned char c) { return c >= 0x20U && c != 0x7fU; });
}

void require_fields(const engine::Json& object, const std::set<std::string>& fields) {
    if (!object.is_object() || object.size() != fields.size()) {
        throw engine::Error("session.field_set", "session object is incomplete or contains unknown fields", 4);
    }
    for (const auto& [key, value] : object.items()) {
        static_cast<void>(value);
        if (!fields.contains(key)) throw engine::Error("session.unknown_field", "session object contains an unknown field", 4);
    }
}

std::string text(const engine::Json& object, const char* field, std::size_t maximum = 256U) {
    if (!object.contains(field) || !object.at(field).is_string()) {
        throw engine::Error("session.invalid_field", std::string(field) + " must be a string", 4);
    }
    const auto value = object.at(field).get<std::string>();
    if (value.empty() || value.size() > maximum) {
        throw engine::Error("session.invalid_field", std::string(field) + " has invalid length", 4);
    }
    return value;
}

std::uint64_t number(const engine::Json& object, const char* field) {
    if (!object.contains(field) || (!object.at(field).is_number_unsigned() && !object.at(field).is_number_integer())) {
        throw engine::Error("session.invalid_field", std::string(field) + " must be an integer", 4);
    }
    try {
        const auto value = object.at(field).get<std::uint64_t>();
        if (value > 9007199254740991ULL) throw engine::Error("session.invalid_field", "integer is out of range", 4);
        return value;
    } catch (const nlohmann::json::exception&) {
        throw engine::Error("session.invalid_field", "integer is out of range", 4);
    }
}

std::vector<std::string> string_array(const engine::Json& value, const char* field, std::size_t maximum) {
    if (!value.is_array() || value.size() > maximum) {
        throw engine::Error("session.invalid_field", std::string(field) + " must be a bounded array", 4);
    }
    std::vector<std::string> result;
    std::set<std::string> seen;
    for (const auto& item : value) {
        if (!item.is_string() || !safe_token(item.get<std::string>()) || !seen.insert(item.get<std::string>()).second) {
            throw engine::Error("session.invalid_field", std::string(field) + " contains an invalid or duplicate token", 4);
        }
        result.push_back(item.get<std::string>());
    }
    return result;
}

std::vector<std::uint64_t> version_array(const engine::Json& value, const char* field) {
    if (!value.is_array() || value.empty() || value.size() > 8U) {
        throw engine::Error("session.invalid_field", std::string(field) + " must be a bounded nonempty array", 4);
    }
    std::vector<std::uint64_t> result;
    std::set<std::uint64_t> seen;
    for (const auto& item : value) {
        if ((!item.is_number_unsigned() && !item.is_number_integer())) {
            throw engine::Error("session.invalid_field", std::string(field) + " contains a non-integer", 4);
        }
        try {
            const auto version = item.get<std::uint64_t>();
            if (version == 0U || version > 1024U || !seen.insert(version).second) {
                throw engine::Error("session.invalid_field", std::string(field) + " contains an invalid or duplicate version", 4);
            }
            result.push_back(version);
        } catch (const nlohmann::json::exception&) {
            throw engine::Error("session.invalid_field", std::string(field) + " contains an out-of-range version", 4);
        }
    }
    return result;
}

std::string operation_fingerprint(const engine::Request& request) {
    std::vector<std::string> contexts;
    for (const auto& value : request.payload.at("context_refs")) contexts.push_back(value.get<std::string>());
    std::sort(contexts.begin(), contexts.end());
    const engine::Json semantic{
        {"operation", request.operation},
        {"operation_id", request.payload.at("operation_id")},
        {"expected_journal_digest", request.payload.at("expected_journal_digest")},
        {"repository_root", request.payload.at("repository_root")},
        {"context_refs", contexts},
    };
    return engine::tagged_sha256(semantic.dump());
}

std::string finalize_digest(engine::Json& value, const char* field) {
    value.erase(field);
    const auto digest = engine::tagged_sha256(value.dump());
    value[field] = digest;
    return digest;
}

void verify_digest(const engine::Json& value, const char* field, const std::string& code) {
    if (!value.contains(field) || !value.at(field).is_string() || !tagged_digest(value.at(field).get<std::string>())) {
        throw engine::Error(code, std::string(field) + " is invalid", 5);
    }
    auto copy = value;
    const auto expected = copy.at(field).get<std::string>();
    copy.erase(field);
    if (engine::tagged_sha256(copy.dump()) != expected) throw engine::Error(code, std::string(field) + " digest mismatch", 5);
}

std::string capability_binding(const engine::Json& capability) {
    const auto& subject = capability.at("subject");
    const auto& target = capability.at("target");
    const std::array<std::string, 19> values = {
        text(capability, "protocol"), text(subject, "id"), text(subject, "kind"), text(subject, "authority"),
        text(capability, "tops_id"), text(target, "operation"), text(target, "resource"),
        text(target, "audience"), text(target, "scope"), text(capability, "authority_basis"),
        text(capability, "grant_id"), text(capability, "request_id"), text(capability, "correlation_id"),
        text(capability, "issued_at"), text(capability, "expires_at"), text(capability, "policy_digest"),
        text(capability, "config_digest"), "transferable=false", "canonical_apply=false",
    };
    std::string joined;
    for (std::size_t index = 0; index < values.size(); ++index) {
        if (index != 0U) joined.push_back('\n');
        joined += values[index];
    }
    return engine::tagged_sha256(joined);
}

std::string repository_resource(const std::string& repository_root) {
    return "symphony.knowledge.repository:" + engine::sha256_hex("repository-root:" + repository_root);
}

engine::Json validate_authorization(const engine::Json& decision, const std::string& expected_operation,
                                    const std::string& tops_id, const std::string& expected_resource) {
    require_fields(decision, {
        "schema", "decision_id", "request_id", "correlation_id", "tops_id", "subject", "target",
        "effect", "reason_code", "authority_basis", "capability", "policy_digest", "config_digest",
        "decided_at", "expires_at", "caller_class_used", "canonical_apply",
    });
    if (text(decision, "schema") != decision_protocol || text(decision, "effect") != "allow" ||
        !safe_token(text(decision, "decision_id")) || !safe_token(text(decision, "request_id")) ||
        !safe_token(text(decision, "correlation_id")) || !safe_token(text(decision, "reason_code")) ||
        !tagged_digest(text(decision, "policy_digest")) || !tagged_digest(text(decision, "config_digest")) ||
        !engine::is_utc_seconds(text(decision, "decided_at")) ||
        text(decision, "tops_id") != tops_id || !decision.at("caller_class_used").is_boolean() ||
        decision.at("caller_class_used").get<bool>() || !decision.at("canonical_apply").is_boolean() ||
        decision.at("canonical_apply").get<bool>() || !decision.at("capability").is_object() ||
        !decision.at("authority_basis").is_string() || !decision.at("expires_at").is_string()) {
        throw engine::Error("session.authorization_denied", "SSIAG decision does not carry an allowed non-apply capability", 4);
    }
    const auto& target = decision.at("target");
    require_fields(target, {"operation", "resource", "audience", "scope"});
    if (text(target, "operation") != expected_operation || text(target, "resource") != expected_resource ||
        text(target, "audience") != "qxctl" || text(target, "scope") != "tops:" + tops_id) {
        throw engine::Error("session.authorization_target_mismatch", "SSIAG decision target does not match the session command", 4);
    }
    const auto& capability = decision.at("capability");
    require_fields(capability, {
        "protocol", "capability_id", "subject", "tops_id", "target", "authority_basis", "grant_id",
        "request_id", "correlation_id", "issued_at", "expires_at", "policy_digest", "config_digest",
        "binding_digest", "transferable", "canonical_apply",
    });
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
        throw engine::Error("session.capability_mismatch", "SSIAG capability does not bind the decision", 4);
    }
    const auto& subject = capability.at("subject");
    require_fields(subject, {"id", "kind", "authority"});
    if (!safe_token(text(subject, "id")) || !safe_token(text(subject, "kind")) ||
        text(subject, "authority") != "unix_peer_credentials") {
        throw engine::Error("session.subject_invalid", "SSIAG subject evidence is invalid", 4);
    }
    const auto issued = text(capability, "issued_at");
    const auto expires = text(capability, "expires_at");
    const auto now = utc_now();
    const auto binding = text(capability, "binding_digest");
    if (!engine::is_utc_seconds(issued) || !engine::is_utc_seconds(expires) || issued > now || expires <= now || issued >= expires ||
        !tagged_digest(text(capability, "policy_digest")) || !tagged_digest(text(capability, "config_digest")) ||
        !tagged_digest(binding) || capability_binding(capability) != binding ||
        text(capability, "capability_id") != "ssiag-capability:" + binding.substr(7U)) {
        throw engine::Error("session.capability_invalid", "SSIAG capability binding or lifetime is invalid", 4);
    }
    return capability;
}

engine::Json compatibility_result(const engine::Json& client, const engine::Json* journal) {
    require_fields(client, {"client_id", "client_version", "process_protocols", "journal_read_versions", "journal_write_versions", "capabilities"});
    const auto protocols = string_array(client.at("process_protocols"), "process_protocols", 8U);
    const auto capabilities = string_array(client.at("capabilities"), "capabilities", 64U);
    if (protocols.empty() || capabilities.empty() || text(client, "client_id") != "qxctl" ||
        !safe_token(text(client, "client_version"))) {
        throw engine::Error("session.invalid_field", "session client identity or capabilities are invalid", 4);
    }
    const auto read_versions = version_array(client.at("journal_read_versions"), "journal_read_versions");
    const auto write_versions = version_array(client.at("journal_write_versions"), "journal_write_versions");
    std::vector<std::string> missing;
    std::vector<std::string> shared;
    for (const auto& required : required_capabilities) {
        if (std::find(capabilities.begin(), capabilities.end(), required) == capabilities.end()) missing.push_back(required);
        else shared.push_back(required);
    }
    for (const auto& optional : optional_capabilities) {
        if (std::find(capabilities.begin(), capabilities.end(), optional) != capabilities.end()) shared.push_back(optional);
    }
    const bool process = std::find(protocols.begin(), protocols.end(), engine::process_protocol_v1) != protocols.end();
    const bool read = std::find(read_versions.begin(), read_versions.end(), format_version) != read_versions.end();
    const bool write = std::find(write_versions.begin(), write_versions.end(), format_version) != write_versions.end();
    std::string mode = process && read && write && missing.empty() ? "full" : (process && read ? "read_only" : "unsupported");
    std::vector<std::string> reasons {mode == "full" ? "full v1 session capability overlap" : "session capability overlap is insufficient for mutation"};
    if (journal != nullptr && journal->at("format_version") != format_version) mode = "migration_required";
    return engine::Json{
        {"mode", mode}, {"process_protocol", process ? engine::Json(engine::process_protocol_v1) : engine::Json(nullptr)},
        {"journal_read_version", read ? engine::Json(format_version) : engine::Json(nullptr)},
        {"journal_write_version", write ? engine::Json(format_version) : engine::Json(nullptr)},
        {"shared_capabilities", shared}, {"missing_capabilities", missing}, {"reasons", reasons},
    };
}

engine::Json command_compatibility();

void validate_stored_capability(const engine::Json& capability, const std::string& tops_id,
                                const engine::Json& subject, const std::string& repository_root) {
    require_fields(capability, {
        "protocol", "capability_id", "subject", "tops_id", "target", "authority_basis", "grant_id",
        "request_id", "correlation_id", "issued_at", "expires_at", "policy_digest", "config_digest",
        "binding_digest", "transferable", "canonical_apply",
    });
    require_fields(subject, {"id", "kind", "authority"});
    const auto& target = capability.at("target");
    require_fields(target, {"operation", "resource", "audience", "scope"});
    const auto operation = text(target, "operation");
    const std::set<std::string> operations{
        "symphony.knowledge.session.begin", "symphony.knowledge.session.status",
        "symphony.knowledge.session.checkpoint", "symphony.knowledge.session.close",
        "symphony.knowledge.session.recover",
    };
    const auto basis = text(capability, "authority_basis");
    const auto binding = text(capability, "binding_digest");
    if (text(capability, "protocol") != capability_protocol || !safe_token(text(capability, "capability_id")) ||
        capability.at("subject") != subject || text(capability, "tops_id") != tops_id ||
        !operations.contains(operation) || text(target, "resource") != repository_resource(repository_root) ||
        text(target, "audience") != "qxctl" || text(target, "scope") != "tops:" + tops_id ||
        (basis != "host_owner" && basis != "granted_permission") || !safe_token(text(capability, "grant_id")) ||
        !safe_token(text(capability, "request_id")) || !safe_token(text(capability, "correlation_id")) ||
        !engine::is_utc_seconds(text(capability, "issued_at")) || !engine::is_utc_seconds(text(capability, "expires_at")) ||
        text(capability, "issued_at") >= text(capability, "expires_at") ||
        !tagged_digest(text(capability, "policy_digest")) || !tagged_digest(text(capability, "config_digest")) ||
        !tagged_digest(binding) ||
        !capability.at("transferable").is_boolean() || capability.at("transferable").get<bool>() ||
        !capability.at("canonical_apply").is_boolean() || capability.at("canonical_apply").get<bool>() ||
        capability_binding(capability) != binding ||
        text(capability, "capability_id") != "ssiag-capability:" + binding.substr(7U)) {
        throw engine::Error("session.capability_invalid", "stored SSIAG capability is invalid", 5);
    }
}

void validate_checkpoint(const engine::Json& checkpoint, std::uint64_t sequence,
                         const engine::Json& previous_digest) {
    require_fields(checkpoint, {"sequence", "kind", "operation_id", "operation_fingerprint", "observed_at", "decision_id", "capability_binding_digest", "previous_checkpoint_digest", "checkpoint_digest"});
    if (number(checkpoint, "sequence") != sequence || !safe_token(text(checkpoint, "kind")) ||
        !safe_token(text(checkpoint, "operation_id")) || !tagged_digest(text(checkpoint, "operation_fingerprint")) ||
        !engine::is_utc_seconds(text(checkpoint, "observed_at")) || checkpoint.at("previous_checkpoint_digest") != previous_digest ||
        !safe_token(text(checkpoint, "decision_id")) || !tagged_digest(text(checkpoint, "capability_binding_digest"))) {
        throw engine::Error("session.checkpoint_invalid", "session checkpoint identity is invalid", 5);
    }
    verify_digest(checkpoint, "checkpoint_digest", "session.checkpoint_invalid");
}

void validate_journal(const engine::Json& journal) {
    require_fields(journal, {
        "protocol", "format_version", "session_id", "session_key", "generation", "previous_journal_digest",
        "state", "close_reason", "tops_id", "subject", "repository", "authority_epoch", "contexts",
        "checkpoints", "compatibility", "extensions", "recovery", "started_at", "updated_at", "closed_at",
        "canonical", "journal_digest",
    });
    const auto generation = number(journal, "generation");
    if (text(journal, "protocol") != journal_protocol || number(journal, "format_version") != format_version ||
        !safe_token(text(journal, "session_id")) || !tagged_digest(text(journal, "session_key")) ||
        generation == 0U || !engine::is_utc_seconds(text(journal, "started_at")) ||
        !engine::is_utc_seconds(text(journal, "updated_at")) || text(journal, "started_at") > text(journal, "updated_at") ||
        !lowercase_uuid(text(journal, "tops_id")) || !journal.at("canonical").is_boolean() || journal.at("canonical").get<bool>() ||
        (!journal.at("previous_journal_digest").is_null() &&
         (!journal.at("previous_journal_digest").is_string() || !tagged_digest(journal.at("previous_journal_digest").get<std::string>())))) {
        throw engine::Error("session.journal_invalid", "session journal identity is invalid", 5);
    }
    if ((generation == 1U) != journal.at("previous_journal_digest").is_null()) {
        throw engine::Error("session.journal_invalid", "session predecessor linkage is invalid", 5);
    }
    const auto state = text(journal, "state");
    if ((state != "open" && state != "closed") || !journal.at("checkpoints").is_array() ||
        journal.at("checkpoints").empty() || journal.at("checkpoints").size() > max_checkpoints) {
        throw engine::Error("session.journal_invalid", "session journal state or checkpoints are invalid", 5);
    }
    if (state == "open" && (!journal.at("closed_at").is_null() || !journal.at("close_reason").is_null())) {
        throw engine::Error("session.journal_invalid", "open session contains terminal fields", 5);
    }
    const std::set<std::string> close_reasons{
        "logout", "expired", "revoked", "policy_changed", "config_changed",
        "administrator_closed", "recovery_closed",
    };
    if (state == "closed" && (!journal.at("closed_at").is_string() ||
        !engine::is_utc_seconds(text(journal, "closed_at")) || text(journal, "closed_at") < text(journal, "started_at") ||
        !journal.at("close_reason").is_string() || !close_reasons.contains(text(journal, "close_reason")))) {
        throw engine::Error("session.journal_invalid", "closed session lacks terminal fields", 5);
    }

    const auto& subject = journal.at("subject");
    require_fields(subject, {"id", "kind", "authority"});
    if (!safe_token(text(subject, "id")) || !safe_token(text(subject, "kind")) ||
        text(subject, "authority") != "unix_peer_credentials") {
        throw engine::Error("session.journal_invalid", "session subject is invalid", 5);
    }
    const auto& repository = journal.at("repository");
    require_fields(repository, {"repository_root", "repository_key"});
    const auto repository_root = text(repository, "repository_root", engine::Limits::max_path_bytes);
    if (!safe_absolute_path(repository_root) || !tagged_digest(text(repository, "repository_key")) ||
        text(repository, "repository_key") != engine::tagged_sha256("repository-root:" + repository_root)) {
        throw engine::Error("session.journal_invalid", "session repository binding is invalid", 5);
    }
    const auto expected_session_key = engine::tagged_sha256(
        "session-key:" + text(journal, "tops_id") + "|" + text(subject, "id") + "|" + repository_root);
    if (text(journal, "session_key") != expected_session_key) {
        throw engine::Error("session.journal_invalid", "session key does not bind its TOPS, subject, and repository", 5);
    }
    const auto& authority = journal.at("authority_epoch");
    require_fields(authority, {"decision_id", "capability", "began_at", "expires_at"});
    if (!safe_token(text(authority, "decision_id")) || !engine::is_utc_seconds(text(authority, "began_at")) ||
        !engine::is_utc_seconds(text(authority, "expires_at")) || text(authority, "began_at") >= text(authority, "expires_at") ||
        text(authority, "began_at") != text(journal, "started_at")) {
        throw engine::Error("session.journal_invalid", "session authority epoch is invalid", 5);
    }
    validate_stored_capability(authority.at("capability"), text(journal, "tops_id"), subject, repository_root);
    if (text(authority, "expires_at") != text(authority.at("capability"), "expires_at")) {
        throw engine::Error("session.journal_invalid", "session authority expiry is inconsistent", 5);
    }
    if (!journal.at("contexts").is_array() || journal.at("contexts").size() > 64U) {
        throw engine::Error("session.journal_invalid", "session contexts are invalid", 5);
    }
    std::set<std::string> context_refs;
    std::vector<std::string> context_decisions;
    for (const auto& context : journal.at("contexts")) {
        require_fields(context, {"context_ref", "attached_at", "decision_id"});
        const auto reference = text(context, "context_ref");
        if (!safe_token(reference) || !context_refs.insert(reference).second ||
            !engine::is_utc_seconds(text(context, "attached_at")) || text(context, "attached_at") < text(journal, "started_at") ||
            text(context, "attached_at") > text(journal, "updated_at") || !safe_token(text(context, "decision_id"))) {
            throw engine::Error("session.journal_invalid", "session context reference is invalid", 5);
        }
        context_decisions.push_back(text(context, "decision_id"));
    }
    if (journal.at("compatibility") != command_compatibility()) {
        throw engine::Error("session.journal_incompatible", "session compatibility contract is unsupported", 4);
    }
    if (!journal.at("extensions").is_array() || journal.at("extensions").size() > 64U) {
        throw engine::Error("session.journal_invalid", "session extensions are invalid", 5);
    }
    for (const auto& extension : journal.at("extensions")) {
        require_fields(extension, {"extension_id", "extension_version", "critical", "payload", "payload_digest"});
        if (!safe_token(text(extension, "extension_id")) || !safe_token(text(extension, "extension_version")) ||
            !extension.at("critical").is_boolean() || !tagged_digest(text(extension, "payload_digest")) ||
            text(extension, "payload_digest") != engine::tagged_sha256(extension.at("payload").dump())) {
            throw engine::Error("session.extension_invalid", "session extension is invalid", 5);
        }
        if (extension.at("critical").get<bool>()) {
            throw engine::Error("session.critical_extension_unknown", "unknown critical session extension prevents mutation", 4);
        }
    }
    const auto& recovery = journal.at("recovery");
    require_fields(recovery, {"state", "last_recovery_at", "disposition", "recovered_from_digest", "detail"});
    const auto recovery_state = text(recovery, "state");
    const auto disposition = text(recovery, "disposition");
    const std::set<std::string> dispositions{
        "not_applicable", "adopted_linked_successor", "repaired_head", "rolled_forward_from_valid_slot",
    };
    if ((recovery_state != "clean" && recovery_state != "recovered") || !dispositions.contains(disposition) ||
        !recovery.at("detail").is_string() || recovery.at("detail").get<std::string>().empty() ||
        recovery.at("detail").get<std::string>().size() > 4096U ||
        (recovery_state == "clean" && (!recovery.at("last_recovery_at").is_null() ||
          disposition != "not_applicable" || !recovery.at("recovered_from_digest").is_null())) ||
        (recovery_state == "recovered" &&
         (!recovery.at("last_recovery_at").is_string() || !engine::is_utc_seconds(text(recovery, "last_recovery_at")) ||
          disposition == "not_applicable" || !recovery.at("recovered_from_digest").is_string() ||
          !tagged_digest(text(recovery, "recovered_from_digest"))))) {
        throw engine::Error("session.journal_invalid", "session recovery evidence is invalid", 5);
    }
    std::uint64_t sequence = 1U;
    engine::Json previous = nullptr;
    std::string previous_observed;
    std::string last_kind;
    std::string last_observed;
    std::set<std::string> checkpoint_decisions;
    std::set<std::string> recovery_times;
    bool saw_close = false;
    for (const auto& checkpoint_value : journal.at("checkpoints")) {
        validate_checkpoint(checkpoint_value, sequence++, previous);
        const auto kind = text(checkpoint_value, "kind");
        const auto observed = text(checkpoint_value, "observed_at");
        if ((sequence == 2U && kind != "begin") ||
            (sequence > 2U && (kind == "begin" || saw_close)) ||
            (kind != "begin" && kind != "checkpoint" && kind != "close" && kind != "recover") ||
            observed < text(journal, "started_at") || observed > text(journal, "updated_at") ||
            (!previous_observed.empty() && observed < previous_observed)) {
            throw engine::Error("session.checkpoint_invalid", "session checkpoint sequence semantics are invalid", 5);
        }
        if (kind == "close") saw_close = true;
        if (kind == "recover") recovery_times.insert(observed);
        checkpoint_decisions.insert(text(checkpoint_value, "decision_id"));
        previous_observed = observed;
        last_observed = observed;
        last_kind = kind;
        previous = checkpoint_value.at("checkpoint_digest");
    }
    const auto& first = journal.at("checkpoints").front();
    if (text(first, "decision_id") != text(authority, "decision_id") ||
        text(first, "capability_binding_digest") != text(authority.at("capability"), "binding_digest") ||
        last_observed != text(journal, "updated_at") ||
        (state == "open" && saw_close) ||
        (state == "closed" && last_kind != "close" && last_kind != "recover") ||
        (state == "closed" && text(journal, "closed_at") != last_observed)) {
        throw engine::Error("session.journal_invalid", "session checkpoint chain does not match journal state", 5);
    }
    for (const auto& decision_id : context_decisions) {
        if (!checkpoint_decisions.contains(decision_id)) {
            throw engine::Error("session.journal_invalid", "session context refers to an unknown decision", 5);
        }
    }
    if (recovery_state == "recovered" && !recovery_times.contains(text(recovery, "last_recovery_at"))) {
        throw engine::Error("session.journal_invalid", "session recovery time has no matching checkpoint", 5);
    }
    verify_digest(journal, "journal_digest", "session.journal_invalid");
}

void validate_head(const engine::Json& head) {
    require_fields(head, {"protocol", "format_version", "session_key", "active_slot", "generation", "journal_digest", "previous_head_digest", "updated_at", "head_digest"});
    if (text(head, "protocol") != head_protocol || number(head, "format_version") != format_version ||
        !tagged_digest(text(head, "session_key")) || number(head, "active_slot") > 1U ||
        number(head, "generation") == 0U || !tagged_digest(text(head, "journal_digest")) ||
        !engine::is_utc_seconds(text(head, "updated_at")) ||
        (!head.at("previous_head_digest").is_null() &&
         (!head.at("previous_head_digest").is_string() ||
          !tagged_digest(head.at("previous_head_digest").get<std::string>())))) {
        throw engine::Error("session.head_invalid", "session head identity is invalid", 5);
    }
    verify_digest(head, "head_digest", "session.head_invalid");
}

void validate_owned_directory(int fd, bool managed) {
    struct stat status {};
    if (::fstat(fd, &status) != 0) system_error("session.state_stat_failed", "could not inspect state directory");
    if (!S_ISDIR(status.st_mode) || status.st_uid != ::geteuid() || (status.st_mode & 0022) != 0) {
        throw engine::Error("session.state_directory_unsafe", "state directory must be caller-owned and protected", 5);
    }
    if (managed && ::fchmod(fd, 0700) != 0) system_error("session.state_mode_failed", "could not restrict managed state directory");
}

void validate_owned_regular(int fd) {
    struct stat status {};
    if (::fstat(fd, &status) != 0) system_error("session.state_stat_failed", "could not inspect state file");
    if (!S_ISREG(status.st_mode) || status.st_uid != ::geteuid() || (status.st_mode & 0077) != 0 || status.st_nlink != 1) {
        throw engine::Error("session.state_file_unsafe", "state file must be a private caller-owned single-link regular file", 5);
    }
}

std::optional<FileDescriptor> open_root(const std::string& path, bool create) {
    if (!safe_absolute_path(path) || path == "/") throw engine::Error("session.state_root_invalid", "state root must be an absolute descendant path", 4);
    FileDescriptor current(::open("/", O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC));
    if (current.get() < 0) system_error("session.state_open_failed", "could not open filesystem root");
    for (const auto& part_value : fs::path(path).relative_path()) {
        const auto part = part_value.string();
        int next = ::openat(current.get(), part.c_str(), O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
        if (next < 0 && errno == ENOENT && !create) return std::nullopt;
        if (next < 0 && errno == ENOENT) {
            if (::mkdirat(current.get(), part.c_str(), 0700) != 0 && errno != EEXIST) system_error("session.state_create_failed", "could not create state root");
            if (::fsync(current.get()) != 0) system_error("session.state_sync_failed", "could not synchronize state root");
            next = ::openat(current.get(), part.c_str(), O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
        }
        if (next < 0) system_error("session.state_open_failed", "could not safely open state root");
        current = FileDescriptor(next);
    }
    validate_owned_directory(current.get(), false);
    return std::optional<FileDescriptor>(std::move(current));
}

std::optional<FileDescriptor> open_child(int parent, const std::string& name, bool create) {
    int next = ::openat(parent, name.c_str(), O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
    if (next < 0 && errno == ENOENT && !create) return std::nullopt;
    if (next < 0 && errno == ENOENT) {
        if (::mkdirat(parent, name.c_str(), 0700) != 0 && errno != EEXIST) system_error("session.state_create_failed", "could not create managed state directory");
        if (::fsync(parent) != 0) system_error("session.state_sync_failed", "could not synchronize managed state directory");
        next = ::openat(parent, name.c_str(), O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
    }
    if (next < 0) system_error("session.state_open_failed", "could not safely open managed state directory");
    FileDescriptor result(next);
    validate_owned_directory(result.get(), create);
    return std::optional<FileDescriptor>(std::move(result));
}

std::optional<SessionLock> open_session(const std::string& root, const std::string& key_hex,
                                        bool exclusive, bool create) {
    auto root_directory = open_root(root, create);
    if (!root_directory.has_value()) return std::nullopt;
    auto current = std::move(*root_directory);
    for (const auto& component : std::array<std::string, 6>{"symphony", "knowledge-session-coordinator", "sessions", "v1", "epochs", key_hex}) {
        auto child = open_child(current.get(), component, create);
        if (!child.has_value()) return std::nullopt;
        current = std::move(*child);
    }
    const int flags = (create ? O_RDWR | O_CREAT : O_RDONLY) | O_NOFOLLOW | O_CLOEXEC;
    int raw = ::openat(current.get(), "journal.lock", flags, 0600);
    if (raw < 0 && errno == ENOENT && !create) return std::nullopt;
    if (raw < 0) system_error("session.lock_open_failed", "could not open session lock");
    FileDescriptor lock(raw);
    validate_owned_regular(lock.get());
    const int operation = (exclusive ? LOCK_EX : LOCK_SH) | LOCK_NB;
    if (::flock(lock.get(), operation) != 0) {
        if (errno == EWOULDBLOCK || errno == EAGAIN) throw engine::Error("session.busy", "authenticated session is busy", 4);
        system_error("session.lock_failed", "could not lock authenticated session");
    }
    return std::optional<SessionLock>(std::in_place, std::move(current), std::move(lock));
}

std::optional<std::string> read_file(int directory, const std::string& name) {
    int raw = ::openat(directory, name.c_str(), O_RDONLY | O_NONBLOCK | O_NOFOLLOW | O_CLOEXEC);
    if (raw < 0 && errno == ENOENT) return std::nullopt;
    if (raw < 0) system_error("session.state_file_open_failed", "could not safely open " + name);
    FileDescriptor file(raw);
    validate_owned_regular(file.get());
    std::string data;
    std::array<char, 16384> buffer {};
    for (;;) {
        const auto count = ::read(file.get(), buffer.data(), buffer.size());
        if (count < 0 && errno == EINTR) continue;
        if (count < 0) system_error("session.state_file_read_failed", "could not read " + name);
        if (count == 0) break;
        if (data.size() + static_cast<std::size_t>(count) > max_state_bytes) throw engine::Error("session.state_file_oversized", name + " exceeds state limit", 5);
        data.append(buffer.data(), static_cast<std::size_t>(count));
    }
    return data;
}

engine::Json parse_file(const std::string& data, const std::string& name) {
    try { return engine::parse_bounded_json(data, max_state_bytes); }
    catch (const engine::Error&) { throw engine::Error("session.state_json_invalid", name + " is not valid bounded JSON", 5); }
}

Candidate read_candidate(int directory, int slot) {
    Candidate candidate;
    candidate.slot = slot;
    const auto data = read_file(directory, "journal." + std::to_string(slot) + ".json");
    if (!data.has_value()) return candidate;
    candidate.exists = true;
    try {
        candidate.journal = parse_file(*data, "session journal slot");
        if (!candidate.journal.is_object() || !candidate.journal.contains("protocol") ||
            candidate.journal.at("protocol") != journal_protocol || candidate.journal.at("format_version") != format_version) {
            throw engine::Error("session.journal_incompatible", "session journal uses an unsupported protocol", 4);
        }
        validate_journal(candidate.journal);
        candidate.valid = true;
    } catch (const engine::Error& error) {
        candidate.incompatible = error.code() == "session.journal_incompatible" || error.code() == "session.critical_extension_unknown";
    }
    return candidate;
}

std::optional<engine::Json> read_head(int directory, bool tolerate_invalid) {
    const auto data = read_file(directory, "head.json");
    if (!data.has_value()) return std::nullopt;
    try {
        auto head = parse_file(*data, "session head");
        validate_head(head);
        return head;
    } catch (const engine::Error&) {
        if (tolerate_invalid) return std::nullopt;
        throw;
    }
}

State load_state(int directory) {
    const auto head = read_head(directory, false);
    if (!head.has_value()) {
        const auto zero = read_candidate(directory, 0);
        const auto one = read_candidate(directory, 1);
        if (zero.exists || one.exists) throw engine::Error("session.head_missing", "session slots exist without a valid head; run recover", 5);
        return {};
    }
    const int slot = static_cast<int>(number(*head, "active_slot"));
    const auto candidate = read_candidate(directory, slot);
    if (!candidate.valid || candidate.journal.at("journal_digest") != head->at("journal_digest") ||
        candidate.journal.at("generation") != head->at("generation") || candidate.journal.at("session_key") != head->at("session_key")) {
        throw engine::Error("session.head_slot_mismatch", "session head does not select a valid matching journal; run recover", 5);
    }
    const auto inactive = read_candidate(directory, 1 - slot);
    if (inactive.incompatible) {
        throw engine::Error("session.compatibility_required", "inactive slot contains unknown critical or newer session state", 4);
    }
    if (inactive.valid) {
        const auto active_generation = number(candidate.journal, "generation");
        const auto inactive_generation = number(inactive.journal, "generation");
        if (inactive_generation == active_generation &&
            inactive.journal.at("journal_digest") != candidate.journal.at("journal_digest")) {
            throw engine::Error("session.recovery_ambiguous", "session slots diverge at the active generation", 5);
        }
        if (inactive_generation > active_generation) {
            if (inactive_generation == active_generation + 1U &&
                inactive.journal.at("previous_journal_digest") == candidate.journal.at("journal_digest")) {
                throw engine::Error("session.recovery_required", "a linked durable session successor requires explicit recovery", 4);
            }
            throw engine::Error("session.recovery_ambiguous", "inactive session state is unexpectedly ahead or unlinked", 5);
        }
    }
    return State{*head, candidate.journal, true};
}

void write_all(int fd, const std::string& data) {
    std::size_t offset = 0;
    while (offset < data.size()) {
        const auto count = ::write(fd, data.data() + offset, data.size() - offset);
        if (count < 0 && errno == EINTR) continue;
        if (count <= 0) system_error("session.state_file_write_failed", "could not write durable session state");
        offset += static_cast<std::size_t>(count);
    }
}

void write_slot(int directory, int slot, const engine::Json& journal) {
    const auto name = "journal." + std::to_string(slot) + ".json";
    int raw = ::openat(directory, name.c_str(), O_CREAT | O_RDWR | O_NOFOLLOW | O_CLOEXEC, 0600);
    if (raw < 0) system_error("session.state_file_open_failed", "could not open inactive session slot");
    FileDescriptor file(raw);
    validate_owned_regular(file.get());
    if (::ftruncate(file.get(), 0) != 0) system_error("session.state_file_write_failed", "could not prepare inactive session slot");
    write_all(file.get(), journal.dump() + '\n');
    if (::fsync(file.get()) != 0) system_error("session.state_sync_failed", "could not synchronize session slot");
}

void write_head(int directory, const engine::Json& head) {
    static std::atomic<std::uint64_t> counter {0U};
    const auto temporary = ".head.tmp-" + std::to_string(::getpid()) + "-" + std::to_string(counter.fetch_add(1U));
    int raw = ::openat(directory, temporary.c_str(), O_CREAT | O_EXCL | O_WRONLY | O_NOFOLLOW | O_CLOEXEC, 0600);
    if (raw < 0) system_error("session.state_file_open_failed", "could not create temporary session head");
    FileDescriptor file(raw);
    try {
        write_all(file.get(), head.dump() + '\n');
        if (::fsync(file.get()) != 0) system_error("session.state_sync_failed", "could not synchronize session head");
        if (::renameat(directory, temporary.c_str(), directory, "head.json") != 0) system_error("session.state_commit_failed", "could not atomically replace session head");
        if (::fsync(directory) != 0) system_error("session.state_sync_failed", "could not synchronize session directory");
    } catch (...) {
        static_cast<void>(::unlinkat(directory, temporary.c_str(), 0));
        throw;
    }
}

State commit(int directory, engine::Json journal, const std::optional<engine::Json>& prior_head) {
    finalize_digest(journal, "journal_digest");
    validate_journal(journal);
    const int slot = prior_head.has_value() ? 1 - static_cast<int>(number(*prior_head, "active_slot")) : 0;
    write_slot(directory, slot, journal);
    engine::Json head{
        {"protocol", head_protocol}, {"format_version", format_version}, {"session_key", journal.at("session_key")},
        {"active_slot", slot}, {"generation", journal.at("generation")}, {"journal_digest", journal.at("journal_digest")},
        {"previous_head_digest", prior_head.has_value() ? prior_head->at("head_digest") : engine::Json(nullptr)},
        {"updated_at", journal.at("updated_at")},
    };
    finalize_digest(head, "head_digest");
    validate_head(head);
    write_head(directory, head);
    return State{head, journal, true};
}

engine::Json checkpoint(std::uint64_t sequence, const std::string& kind, const std::string& operation_id,
                        const std::string& fingerprint, const std::string& observed,
                        const engine::Json& decision, const engine::Json& previous) {
    engine::Json value{
        {"sequence", sequence}, {"kind", kind}, {"operation_id", operation_id}, {"observed_at", observed},
        {"operation_fingerprint", fingerprint},
        {"decision_id", decision.at("decision_id")},
        {"capability_binding_digest", decision.at("capability").at("binding_digest")},
        {"previous_checkpoint_digest", previous},
    };
    finalize_digest(value, "checkpoint_digest");
    return value;
}

bool replayed(const engine::Json& journal, const engine::Request& request, const std::string& kind) {
    const auto operation_id = request.payload.at("operation_id").get<std::string>();
    const auto fingerprint = operation_fingerprint(request);
    for (const auto& item : journal.at("checkpoints")) {
        if (item.at("operation_id") == operation_id) {
            if (item.at("kind") != kind || item.at("operation_fingerprint") != fingerprint) {
                throw engine::Error("session.operation_reused", "operation identifier was already used for different mutation semantics", 4);
            }
            return true;
        }
    }
    return false;
}

void require_expected(const State& state, const std::string& expected, bool begin) {
    if (expected == "absent") {
        if (!begin || state.present) throw engine::Error("session.expected_state_mismatch", "session journal was not absent", 4);
        return;
    }
    if (!tagged_digest(expected) || !state.present || state.journal.at("journal_digest") != expected) {
        throw engine::Error("session.expected_state_mismatch", state.present ? "session journal digest is stale" : "session journal is absent", 4);
    }
}

engine::Json command_compatibility() {
    return engine::Json{
        {"process_protocols", engine::Json::array({engine::process_protocol_v1})},
        {"journal_read_versions", engine::Json::array({format_version})}, {"journal_write_version", format_version},
        {"minimum_reader_version", format_version}, {"required_capabilities", required_capabilities},
        {"optional_capabilities", optional_capabilities}, {"opaque_extensions_preserved", true},
    };
}

engine::Json make_result(const std::string& operation, engine::Json compatibility, const State& state,
                         bool changed, bool recovered, const std::vector<std::string>& actions, bool read_only) {
    engine::Json journal = nullptr;
    engine::Json digest = nullptr;
    std::string effective = "absent";
    if (state.present) {
        journal = state.journal;
        digest = state.journal.at("journal_digest");
        effective = state.journal.at("state").get<std::string>();
        if (effective == "open" && state.journal.at("authority_epoch").at("expires_at").get<std::string>() <= utc_now()) effective = "expired";
    }
    return engine::Json{
        {"protocol", result_protocol}, {"operation", operation}, {"compatibility", std::move(compatibility)},
        {"journal_present", state.present}, {"journal", std::move(journal)}, {"journal_digest", std::move(digest)},
        {"effective_state", effective}, {"changed", changed}, {"recovered", recovered},
        {"repair_actions", actions}, {"read_only", read_only}, {"canonical_apply_enabled", false}, {"canonical", false},
    };
}

engine::Json validate_command(const engine::Request& request) {
    const auto& payload = request.payload;
    require_fields(payload, {"protocol", "operation", "state_root", "operation_id", "expected_journal_digest", "repository_root", "context_refs", "authorization_decision", "client"});
    static const std::set<std::string> operations = {"session_begin", "session_status", "session_checkpoint", "session_close", "session_recover"};
    if (text(payload, "protocol") != command_protocol || text(payload, "operation") != request.operation || !operations.contains(request.operation) ||
        !safe_absolute_path(text(payload, "state_root", engine::Limits::max_path_bytes)) ||
        !safe_absolute_path(text(payload, "repository_root", engine::Limits::max_path_bytes))) {
        throw engine::Error("session.command_invalid", "session command identity or path is invalid", 4);
    }
    static_cast<void>(string_array(payload.at("context_refs"), "context_refs", 64U));
    const auto& decision = payload.at("authorization_decision");
    const auto tops_id = text(decision, "tops_id");
    static_cast<void>(validate_authorization(
        decision, "symphony.knowledge.session." + request.operation.substr(8U), tops_id,
        repository_resource(text(payload, "repository_root", engine::Limits::max_path_bytes))));
    const bool read = request.operation == "session_status";
    if (read) {
        if (!payload.at("operation_id").is_null() || !payload.at("expected_journal_digest").is_null() || !payload.at("context_refs").empty()) {
            throw engine::Error("session.command_invalid", "session status fields are inconsistent", 4);
        }
    } else {
        if (!payload.at("operation_id").is_string() || !safe_token(payload.at("operation_id").get<std::string>()) || !payload.at("expected_journal_digest").is_string()) {
            throw engine::Error("session.command_invalid", "session mutation identity is invalid", 4);
        }
        const auto expected = payload.at("expected_journal_digest").get<std::string>();
        if (request.operation == "session_begin") {
            if (expected != "absent" && !tagged_digest(expected)) throw engine::Error("session.command_invalid", "session begin expected state is invalid", 4);
        } else if (request.operation == "session_recover") {
            if (expected != "discover" && !tagged_digest(expected)) throw engine::Error("session.command_invalid", "session recovery expected state is invalid", 4);
        } else if (!tagged_digest(expected)) {
            throw engine::Error("session.command_invalid", "session expected state is invalid", 4);
        }
        if ((request.operation == "session_close" || request.operation == "session_recover") && !payload.at("context_refs").empty()) {
            throw engine::Error("session.command_invalid", "session close/recover cannot attach contexts", 4);
        }
    }
    return compatibility_result(payload.at("client"), nullptr);
}

std::string canonical_repository(const std::string& supplied) {
    std::error_code error;
    const auto actual = fs::canonical(fs::current_path(), error);
    if (error || actual.string() != supplied || !fs::is_directory(actual)) {
        throw engine::Error("session.repository_mismatch", "session repository does not match the canonical process working directory", 4);
    }
    return supplied;
}

engine::Json begin_journal(const engine::Request& request, const engine::Json& capability,
                           const std::string& session_key, const std::string& repository,
                           const State& prior) {
    const auto observed = utc_now();
    const auto operation_id = request.payload.at("operation_id").get<std::string>();
    const auto session_id = "session:" + engine::sha256_hex(session_key + "|" + operation_id + "|" + observed);
    engine::Json contexts = engine::Json::array();
    for (const auto& ref : request.payload.at("context_refs")) contexts.push_back(engine::Json{{"context_ref", ref}, {"attached_at", observed}, {"decision_id", request.payload.at("authorization_decision").at("decision_id")}});
    const auto first = checkpoint(1U, "begin", operation_id, operation_fingerprint(request), observed,
                                  request.payload.at("authorization_decision"), nullptr);
    return engine::Json{
        {"protocol", journal_protocol}, {"format_version", format_version}, {"session_id", session_id}, {"session_key", session_key},
        {"generation", prior.present ? number(prior.journal, "generation") + 1U : 1U},
        {"previous_journal_digest", prior.present ? prior.journal.at("journal_digest") : engine::Json(nullptr)},
        {"state", "open"}, {"close_reason", nullptr}, {"tops_id", capability.at("tops_id")}, {"subject", capability.at("subject")},
        {"repository", engine::Json{{"repository_root", repository}, {"repository_key", engine::tagged_sha256("repository-root:" + repository)}}},
        {"authority_epoch", engine::Json{{"decision_id", request.payload.at("authorization_decision").at("decision_id")}, {"capability", capability}, {"began_at", observed}, {"expires_at", capability.at("expires_at")}}},
        {"contexts", contexts}, {"checkpoints", engine::Json::array({first})}, {"compatibility", command_compatibility()},
        {"extensions", engine::Json::array()}, {"recovery", engine::Json{{"state", "clean"}, {"last_recovery_at", nullptr}, {"disposition", "not_applicable"}, {"recovered_from_digest", nullptr}, {"detail", "no recovery has been required"}}},
        {"started_at", observed}, {"updated_at", observed}, {"closed_at", nullptr}, {"canonical", false},
    };
}

engine::Json mutate(const engine::Request& request, const State& current, const std::string& kind,
                    const std::string& recovery_disposition = "not_applicable") {
    auto next = current.journal;
    const auto& decision = request.payload.at("authorization_decision");
    const auto& capability = decision.at("capability");
    if (capability.at("subject") != next.at("subject") || capability.at("tops_id") != next.at("tops_id")) {
        throw engine::Error("session.identity_mismatch", "session mutation subject or TOPS does not match the authority epoch", 4);
    }
    const auto observed = utc_now();
    const bool same_policy = capability.at("policy_digest") == next.at("authority_epoch").at("capability").at("policy_digest");
    const bool same_config = capability.at("config_digest") == next.at("authority_epoch").at("capability").at("config_digest");
    const bool expired = next.at("authority_epoch").at("expires_at").get<std::string>() <= observed;
    if (kind == "checkpoint" && (!same_policy || !same_config || expired || next.at("state") != "open")) {
        throw engine::Error("session.reauthentication_required", "authority epoch is closed, expired, or changed", 4);
    }
    std::set<std::string> known;
    for (const auto& context : next.at("contexts")) known.insert(context.at("context_ref").get<std::string>());
    for (const auto& ref : request.payload.at("context_refs")) {
        if (known.insert(ref.get<std::string>()).second) {
            if (known.size() > 64U) {
                throw engine::Error("session.context_limit", "session context-reference history is full", 4);
            }
            next["contexts"].push_back(engine::Json{{"context_ref", ref}, {"attached_at", observed}, {"decision_id", decision.at("decision_id")}});
        }
    }
    const auto sequence = next.at("checkpoints").size() + 1U;
    if (sequence > max_checkpoints) throw engine::Error("session.checkpoint_limit", "session checkpoint history is full", 4);
    next["checkpoints"].push_back(checkpoint(
        sequence, kind, request.payload.at("operation_id").get<std::string>(),
        operation_fingerprint(request), observed, decision,
        next.at("checkpoints").back().at("checkpoint_digest")));
    next["generation"] = number(next, "generation") + 1U;
    next["previous_journal_digest"] = current.journal.at("journal_digest");
    next["updated_at"] = observed;
    if (kind == "close" || (kind == "recover" && expired && next.at("state") == "open")) {
        next["state"] = "closed";
        next["closed_at"] = observed;
        if (expired) next["close_reason"] = "expired";
        else if (!same_policy) next["close_reason"] = "policy_changed";
        else if (!same_config) next["close_reason"] = "config_changed";
        else next["close_reason"] = kind == "recover" ? "recovery_closed" : "logout";
    }
    if (kind == "recover") {
        next["recovery"] = engine::Json{{"state", "recovered"}, {"last_recovery_at", observed}, {"disposition", recovery_disposition}, {"recovered_from_digest", current.journal.at("journal_digest")}, {"detail", "recovered from uniquely selected valid local session evidence"}};
    }
    return next;
}

State recover(const engine::Request& request, int directory, const std::string& session_key,
              std::vector<std::string>* actions, bool* changed) {
    const auto head = read_head(directory, true);
    auto zero = read_candidate(directory, 0);
    auto one = read_candidate(directory, 1);
    if (zero.incompatible || one.incompatible) throw engine::Error("session.compatibility_required", "recovery found unknown critical or newer session state", 4);
    std::vector<Candidate*> valid;
    if (zero.valid) valid.push_back(&zero);
    if (one.valid) valid.push_back(&one);
    if (valid.empty()) {
        if (!zero.exists && !one.exists && !head.has_value()) {
            if (request.payload.at("expected_journal_digest") != "discover") {
                throw engine::Error("session.expected_state_mismatch", "recovery expected a session journal but the stream is absent", 4);
            }
            actions->push_back("no authenticated session state was discovered");
            return {};
        }
        throw engine::Error("session.unrecoverable", "no valid session journal slot remains", 5);
    }
    for (const auto* candidate : valid) if (candidate->journal.at("session_key") != session_key) throw engine::Error("session.unrecoverable", "session slot belongs to another identity/repository key", 5);
    Candidate* selected = valid.front();
    if (valid.size() == 2U) {
        auto* left = valid[0];
        auto* right = valid[1];
        const auto left_generation = number(left->journal, "generation");
        const auto right_generation = number(right->journal, "generation");
        if (left_generation == right_generation) {
            if (left->journal.at("journal_digest") != right->journal.at("journal_digest")) {
                throw engine::Error("session.recovery_ambiguous", "multiple divergent session journals have the same generation", 5);
            }
            selected = left;
        } else {
            auto* lower = left_generation < right_generation ? left : right;
            auto* higher = left_generation < right_generation ? right : left;
            if (number(higher->journal, "generation") != number(lower->journal, "generation") + 1U ||
                higher->journal.at("previous_journal_digest") != lower->journal.at("journal_digest")) {
                throw engine::Error("session.recovery_ambiguous", "session slots do not form one uniquely linked forward chain", 5);
            }
            selected = higher;
        }
    }
    const auto expected = request.payload.at("expected_journal_digest").get<std::string>();
    if (expected != "discover" && selected->journal.at("journal_digest") != expected) throw engine::Error("session.expected_state_mismatch", "recovery expected digest is stale", 4);
    const bool healthy = head.has_value() && head->at("journal_digest") == selected->journal.at("journal_digest") &&
        head->at("generation") == selected->journal.at("generation") &&
        head->at("session_key") == selected->journal.at("session_key");
    State baseline{head.value_or(engine::Json{}), selected->journal, true};
    if (healthy && baseline.journal.at("state") == "closed") {
        actions->push_back("session head and closed journal are already healthy");
        return baseline;
    }
    const bool linked_successor = head.has_value() && !healthy &&
        number(selected->journal, "generation") == number(*head, "generation") + 1U &&
        selected->journal.at("previous_journal_digest") == head->at("journal_digest");
    auto next = mutate(request, baseline, "recover",
        healthy ? "rolled_forward_from_valid_slot" :
        (linked_successor ? "adopted_linked_successor" : "repaired_head"));
    auto committed = commit(directory, std::move(next), head);
    actions->push_back("committed a forward recovery checkpoint and repaired the session head");
    *changed = true;
    return committed;
}

} // namespace

engine::Json validate_ssiag_authorization(
    const engine::Json& decision,
    const std::string& expected_operation,
    const std::string& tops_id,
    const std::string& expected_resource) {
    return validate_authorization(decision, expected_operation, tops_id, expected_resource);
}

engine::Json authority_session_capabilities() {
    return engine::Json{
        {"process_protocols", engine::Json::array({engine::process_protocol_v1})},
        {"journal_read_versions", engine::Json::array({format_version})},
        {"journal_write_versions", engine::Json::array({format_version})},
        {"required_capabilities", required_capabilities}, {"optional_capabilities", optional_capabilities},
        {"two_way_procedural_compatibility", true}, {"silent_downgrade", false}, {"lossy_migration", false},
    };
}

engine::Json handle_authority_session(const engine::Request& request) {
    auto compatibility = validate_command(request);
    const auto& decision = request.payload.at("authorization_decision");
    const auto repository = canonical_repository(request.payload.at("repository_root").get<std::string>());
    const auto capability = validate_authorization(
        decision, "symphony.knowledge.session." + request.operation.substr(8U),
        decision.at("tops_id").get<std::string>(), repository_resource(repository));
    const auto session_key = engine::tagged_sha256(
        "session-key:" + capability.at("tops_id").get<std::string>() + "|" +
        capability.at("subject").at("id").get<std::string>() + "|" + repository);
    const bool exclusive = request.operation != "session_status";
    auto session = open_session(
        request.payload.at("state_root").get<std::string>(), session_key.substr(7), exclusive,
        request.operation != "session_status");
    if (!session.has_value()) {
        return make_result(request.operation, std::move(compatibility), {}, false, false, {}, true);
    }

    if (request.operation == "session_recover") {
        if (compatibility.at("mode") != "full") {
            throw engine::Error("session.compatibility_required", "session recovery requires full v1 capability overlap", 4);
        }
        std::vector<std::string> actions;
        bool changed = false;
        auto recovered = recover(request, session->directory_fd(), session_key, &actions, &changed);
        compatibility = compatibility_result(request.payload.at("client"), recovered.present ? &recovered.journal : nullptr);
        return make_result(request.operation, std::move(compatibility), recovered, changed, changed, actions, false);
    }

    auto current = load_state(session->directory_fd());
    compatibility = compatibility_result(request.payload.at("client"), current.present ? &current.journal : nullptr);
    if (request.operation == "session_status") return make_result(request.operation, std::move(compatibility), current, false, false, {}, true);
    if (compatibility.at("mode") != "full") throw engine::Error("session.compatibility_required", "session mutation requires full v1 capability overlap", 4);

    const auto expected = request.payload.at("expected_journal_digest").get<std::string>();
    if (request.operation == "session_begin") {
        if (current.present && replayed(current.journal, request, "begin")) return make_result(request.operation, std::move(compatibility), current, false, false, {"replayed session begin already committed"}, false);
        require_expected(current, expected, true);
        if (current.present && current.journal.at("state") != "closed") throw engine::Error("session.already_open", "an authenticated session is already open", 4);
        auto next = begin_journal(request, capability, session_key, repository, current);
        auto committed = commit(session->directory_fd(), std::move(next), current.present ? std::optional<engine::Json>(current.head) : std::nullopt);
        return make_result(request.operation, std::move(compatibility), committed, true, false, {"committed authenticated authority epoch"}, false);
    }
    if (!current.present) throw engine::Error("session.absent", "authenticated session journal is absent", 4);
    const std::string kind = request.operation.substr(8U);
    if (replayed(current.journal, request, kind)) return make_result(request.operation, std::move(compatibility), current, false, false, {"replayed session mutation already committed"}, false);
    require_expected(current, expected, false);
    if (request.operation == "session_close" && current.journal.at("state") == "closed") throw engine::Error("session.already_closed", "authenticated session is already closed", 4);
    auto next = mutate(request, current, kind);
    auto committed = commit(session->directory_fd(), std::move(next), current.head);
    return make_result(request.operation, std::move(compatibility), committed, true, false, {"committed authenticated session " + kind}, false);
}

} // namespace symphony::knowledge::session
