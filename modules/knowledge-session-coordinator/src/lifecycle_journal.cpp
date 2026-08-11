#include "lifecycle_journal.hpp"

#include "authority_session.hpp"
#include "lifecycle.hpp"

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
#include <ctime>
#include <unistd.h>
#include <utility>
#include <vector>

namespace symphony::knowledge::session {
namespace engine = symphony::knowledge::engine;
namespace fs = std::filesystem;

namespace {

constexpr const char* command_protocol = "symphony.knowledge.lifecycle-boot-command.v1";
constexpr const char* result_protocol = "symphony.knowledge.lifecycle-boot-result.v1";
constexpr const char* journal_protocol = "symphony.knowledge.lifecycle-boot-journal.v1";
constexpr const char* head_protocol = "symphony.knowledge.lifecycle-boot-head.v1";
constexpr std::uint64_t format_version = 1U;
constexpr const char* apply_command_protocol = "symphony.knowledge.lifecycle-apply-command.v1";
constexpr const char* apply_result_protocol = "symphony.knowledge.lifecycle-apply-result.v1";
constexpr const char* apply_journal_protocol = "symphony.knowledge.lifecycle-boot-journal.v2";
constexpr const char* apply_head_protocol = "symphony.knowledge.lifecycle-boot-head.v2";
constexpr std::uint64_t apply_format_version = 2U;
constexpr std::size_t max_state_bytes = engine::Limits::max_response_bytes;
constexpr std::size_t max_plan_revisions = 256U;
constexpr std::size_t max_checkpoints = 32768U;

const std::vector<std::string> required_capabilities = {
    "atomic-head-v1",
    "dual-slot-journal-v1",
    "dynamic-replanning-v1",
    "expected-state-cas-v1",
    "idempotent-operation-v1",
    "opaque-extension-preservation-v1",
    "recovery-forward-v1",
    "report-only-v1",
};

const std::vector<std::string> optional_capabilities = {
    "discovery-recovery-v1",
    "nonblocking-lock-v1",
};

const std::vector<std::string> apply_required_capabilities = {
    "action-attempt-journal-v2",
    "applied-state-v1",
    "dynamic-replanning-v1",
    "expected-state-cas-v1",
    "external-action-adapter-v1",
    "forward-inverse-v1",
    "opaque-extension-preservation-v1",
    "per-action-authorization-v1",
    "recovery-forward-v1",
    "verified-observation-commit-v1",
};

const std::vector<std::string> apply_optional_capabilities = {
    "discovery-recovery-v2",
    "staged-package-v2",
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

class JournalLock final {
public:
    JournalLock(FileDescriptor directory, FileDescriptor lock)
        : directory_(std::move(directory)), lock_(std::move(lock)) {}
    ~JournalLock() { if (lock_.get() >= 0) static_cast<void>(::flock(lock_.get(), LOCK_UN)); }
    JournalLock(const JournalLock&) = delete;
    JournalLock& operator=(const JournalLock&) = delete;
    JournalLock(JournalLock&&) = default;
    JournalLock& operator=(JournalLock&&) = default;
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
    return std::all_of(value.begin(), value.end(), [](const unsigned char character) {
        const bool alpha = (character >= 'a' && character <= 'z') ||
            (character >= 'A' && character <= 'Z');
        return alpha || (character >= '0' && character <= '9') || character == '.' ||
            character == '_' || character == ':' || character == '-';
    });
}

bool tagged_digest(std::string_view value) {
    return value.size() == 71U && value.starts_with("sha256:") &&
        std::all_of(value.begin() + 7, value.end(), [](const unsigned char character) {
            return (character >= '0' && character <= '9') ||
                (character >= 'a' && character <= 'f');
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
        throw engine::Error("lifecycle_journal.field_set", context + " has an invalid field set", 4);
    }
    for (const auto& [key, value] : object.items()) {
        static_cast<void>(value);
        if (!fields.contains(key)) {
            throw engine::Error("lifecycle_journal.unknown_field", context + " has an unknown field", 4);
        }
    }
}

std::string text(const engine::Json& object, const char* field, std::size_t maximum = 4096U) {
    if (!object.contains(field) || !object.at(field).is_string()) {
        throw engine::Error("lifecycle_journal.invalid_field", std::string(field) + " must be a string", 4);
    }
    const auto value = object.at(field).get<std::string>();
    if (value.empty() || value.size() > maximum) {
        throw engine::Error("lifecycle_journal.invalid_field", std::string(field) + " has invalid length", 4);
    }
    for (const unsigned char character : value) {
        if (character < 0x20U || character == 0x7fU) {
            throw engine::Error("lifecycle_journal.invalid_field", std::string(field) + " contains unsafe text", 4);
        }
    }
    return value;
}

std::uint64_t number(const engine::Json& object, const char* field) {
    if (!object.contains(field) ||
        (!object.at(field).is_number_unsigned() && !object.at(field).is_number_integer())) {
        throw engine::Error("lifecycle_journal.invalid_field", std::string(field) + " must be an integer", 4);
    }
    try {
        const auto value = object.at(field).get<std::uint64_t>();
        if (value > 9007199254740991ULL) {
            throw engine::Error("lifecycle_journal.invalid_field", std::string(field) + " is out of range", 4);
        }
        return value;
    } catch (const nlohmann::json::exception&) {
        throw engine::Error("lifecycle_journal.invalid_field", std::string(field) + " is out of range", 4);
    }
}

std::string utc_now() {
    const auto now = std::chrono::system_clock::now();
    const auto seconds = std::chrono::time_point_cast<std::chrono::seconds>(now);
    const std::time_t value = std::chrono::system_clock::to_time_t(seconds);
    std::tm result {};
    if (::gmtime_r(&value, &result) == nullptr) {
        throw engine::Error("lifecycle_journal.clock_failed", "could not obtain a UTC timestamp", 5);
    }
    std::ostringstream output;
    output << std::put_time(&result, "%Y-%m-%dT%H:%M:%SZ");
    return output.str();
}

std::vector<std::string> token_array(const engine::Json& value, const std::string& name,
                                     std::size_t maximum, bool nonempty = false) {
    if (!value.is_array() || value.size() > maximum || (nonempty && value.empty())) {
        throw engine::Error("lifecycle_journal.invalid_field", name + " must be a bounded array", 4);
    }
    std::vector<std::string> result;
    std::set<std::string> seen;
    for (const auto& item : value) {
        if (!item.is_string() || !safe_token(item.get<std::string>()) ||
            !seen.insert(item.get<std::string>()).second) {
            throw engine::Error("lifecycle_journal.invalid_field", name + " contains an invalid token", 4);
        }
        result.push_back(item.get<std::string>());
    }
    return result;
}

std::vector<std::uint64_t> version_array(const engine::Json& value, const std::string& name) {
    if (!value.is_array() || value.empty() || value.size() > 16U) {
        throw engine::Error("lifecycle_journal.invalid_field", name + " must be a bounded nonempty array", 4);
    }
    std::vector<std::uint64_t> result;
    std::set<std::uint64_t> seen;
    for (const auto& item : value) {
        if (!item.is_number_integer() && !item.is_number_unsigned()) {
            throw engine::Error("lifecycle_journal.invalid_field", name + " contains a non-integer", 4);
        }
        const auto version = item.get<std::uint64_t>();
        if (version == 0U || version > 16U || !seen.insert(version).second) {
            throw engine::Error("lifecycle_journal.invalid_field", name + " contains an invalid version", 4);
        }
        result.push_back(version);
    }
    return result;
}

bool contains(const std::vector<std::string>& values, const std::string& wanted) {
    return std::find(values.begin(), values.end(), wanted) != values.end();
}

bool contains(const std::vector<std::uint64_t>& values, std::uint64_t wanted) {
    return std::find(values.begin(), values.end(), wanted) != values.end();
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

engine::Json compatibility_result(const engine::Json& client, const engine::Json* journal) {
    exact_fields(client, {
        "client_id", "client_version", "process_protocols", "journal_read_versions",
        "journal_write_versions", "capabilities"
    }, "lifecycle journal client");
    if (text(client, "client_id", 128U) != "qxctl" ||
        !safe_version(text(client, "client_version", 64U))) {
        throw engine::Error("lifecycle_journal.client_invalid", "lifecycle journal client identity is invalid", 4);
    }
    const auto processes = token_array(client.at("process_protocols"), "process_protocols", 8U, true);
    const auto reads = version_array(client.at("journal_read_versions"), "journal_read_versions");
    const auto writes = version_array(client.at("journal_write_versions"), "journal_write_versions");
    const auto capabilities = token_array(client.at("capabilities"), "capabilities", 64U);
    std::vector<std::string> missing;
    for (const auto& capability : required_capabilities) {
        if (!contains(capabilities, capability)) missing.push_back(capability);
    }
    const bool process_ok = contains(processes, engine::process_protocol_v1);
    const bool read_ok = contains(reads, format_version);
    const bool write_ok = contains(writes, format_version);
    bool stored_ok = true;
    if (journal != nullptr) stored_ok = number(*journal, "format_version") == format_version;
    const bool full = process_ok && read_ok && write_ok && stored_ok && missing.empty();
    const bool readable = process_ok && read_ok && stored_ok;
    return engine::Json{
        {"mode", full ? "full" : (readable ? "read_only" : "blocked")},
        {"process_protocol", process_ok ? engine::Json(engine::process_protocol_v1) : engine::Json(nullptr)},
        {"journal_read_version", read_ok && stored_ok ? engine::Json(format_version) : engine::Json(nullptr)},
        {"journal_write_version", write_ok ? engine::Json(format_version) : engine::Json(nullptr)},
        {"missing_capabilities", missing},
        {"two_way_procedural_compatibility", true},
        {"reason", full ? "client, coordinator, and stored journal share the full v1 contract" :
            (readable ? "journal is readable but mutation capability is incomplete" :
             "client and coordinator have no safe lifecycle journal read overlap")},
    };
}

engine::Json journal_compatibility() {
    return engine::Json{
        {"journal_read_versions", engine::Json::array({1})}, {"journal_write_version", 1},
        {"desired_read_versions", engine::Json::array({1})}, {"desired_write_version", 1},
        {"observation_read_versions", engine::Json::array({1})}, {"observation_write_version", 1},
        {"plan_read_versions", engine::Json::array({1})}, {"plan_write_version", 1},
        {"applied_read_versions", engine::Json::array({1})}, {"applied_write_version", 1},
        {"receipt_read_versions", engine::Json::array({1, 2})},
        {"required_capabilities", required_capabilities},
        {"optional_capabilities", optional_capabilities},
        {"opaque_extensions_preserved", true},
    };
}

void validate_checkpoint(const engine::Json& checkpoint) {
    exact_fields(checkpoint, {
        "sequence", "kind", "plan_revision", "operation_id", "observed_at",
        "observation_digest", "ready_set_digest", "previous_checkpoint_digest",
        "checkpoint_digest"
    }, "lifecycle checkpoint");
    static const std::set<std::string> kinds = {
        "begin", "observe", "replan", "attempt", "verify", "recover", "close"
    };
    if (number(checkpoint, "sequence") == 0U || number(checkpoint, "sequence") > max_checkpoints ||
        number(checkpoint, "plan_revision") == 0U ||
        number(checkpoint, "plan_revision") > max_plan_revisions ||
        !kinds.contains(text(checkpoint, "kind", 32U)) ||
        !safe_token(text(checkpoint, "operation_id", 256U)) ||
        !engine::is_utc_seconds(text(checkpoint, "observed_at", 20U)) ||
        !tagged_digest(text(checkpoint, "observation_digest", 71U)) ||
        !tagged_digest(text(checkpoint, "ready_set_digest", 71U))) {
        throw engine::Error("lifecycle_journal.journal_invalid", "checkpoint fields are invalid", 5);
    }
    const auto& previous = checkpoint.at("previous_checkpoint_digest");
    if (!previous.is_null() && (!previous.is_string() || !tagged_digest(previous.get<std::string>()))) {
        throw engine::Error("lifecycle_journal.journal_invalid", "checkpoint predecessor is invalid", 5);
    }
    verify_digest(checkpoint, "checkpoint_digest", "lifecycle_journal.journal_invalid");
}

void validate_blocker(const engine::Json& blocker) {
    exact_fields(blocker, {"class", "component_id", "action_id", "retryable", "resolved", "detail"},
                 "lifecycle blocker");
    static const std::set<std::string> classes = {
        "dependency_wait", "observation_retryable", "compatibility_blocked",
        "authorization_denied", "integrity_fatal", "critical_state_unknown", "cycle_detected"
    };
    if (!classes.contains(text(blocker, "class", 64U)) ||
        !safe_token(text(blocker, "component_id", 256U)) ||
        !blocker.at("retryable").is_boolean() || !blocker.at("resolved").is_boolean() ||
        text(blocker, "detail", 4096U).empty()) {
        throw engine::Error("lifecycle_journal.journal_invalid", "blocker fields are invalid", 5);
    }
    if (!blocker.at("action_id").is_null() &&
        (!blocker.at("action_id").is_string() || !safe_token(blocker.at("action_id").get<std::string>()))) {
        throw engine::Error("lifecycle_journal.journal_invalid", "blocker action identity is invalid", 5);
    }
}

void validate_extensions(const engine::Json& extensions) {
    if (!extensions.is_array() || extensions.size() > 64U) {
        throw engine::Error("lifecycle_journal.journal_invalid", "journal extensions are invalid", 5);
    }
    std::set<std::string> identities;
    for (const auto& extension : extensions) {
        exact_fields(extension, {
            "extension_id", "extension_version", "critical", "payload", "payload_digest"
        }, "lifecycle journal extension");
        const auto identity = text(extension, "extension_id", 256U);
        if (!safe_token(identity) || !safe_version(text(extension, "extension_version", 64U)) ||
            !extension.at("critical").is_boolean() ||
            !tagged_digest(text(extension, "payload_digest", 71U)) ||
            engine::tagged_sha256(extension.at("payload").dump()) !=
                extension.at("payload_digest").get<std::string>() ||
            !identities.insert(identity).second) {
            throw engine::Error("lifecycle_journal.journal_invalid", "journal extension evidence is invalid", 5);
        }
        if (extension.at("critical") == true) {
            throw engine::Error(
                "lifecycle_journal.critical_extension_unknown",
                "an unknown critical lifecycle journal extension requires a compatible reader",
                4);
        }
    }
}

void validate_journal(const engine::Json& journal) {
    exact_fields(journal, {
        "protocol", "format_version", "journal_id", "transaction_id", "operation_id",
        "generation", "previous_journal_digest", "profile_id", "profile_digest", "tops_id", "mode", "state",
        "desired_state_digest", "observation_key", "current_observation_digest",
        "current_stable_inventory_digest",
        "prior_applied_state_digest", "current_plan_digest", "current_plan_revision",
        "replan_count", "action_attempts", "blockers", "checkpoints", "compatibility",
        "extensions", "recovery", "started_at", "updated_at", "closed_at", "canonical",
        "apply_authorized", "journal_digest"
    }, "lifecycle journal");
    static const std::set<std::string> modes = {"report", "apply-compatible"};
    static const std::set<std::string> states = {"open", "blocked", "verified", "closed"};
    if (text(journal, "protocol") != journal_protocol || number(journal, "format_version") != format_version ||
        !safe_token(text(journal, "journal_id", 256U)) ||
        !safe_token(text(journal, "transaction_id", 256U)) ||
        !safe_token(text(journal, "operation_id", 256U)) ||
        !safe_token(text(journal, "profile_id", 256U)) ||
        !tagged_digest(text(journal, "profile_digest", 71U)) ||
        !safe_token(text(journal, "tops_id", 256U)) ||
        !modes.contains(text(journal, "mode", 32U)) || !states.contains(text(journal, "state", 32U)) ||
        number(journal, "generation") == 0U ||
        !tagged_digest(text(journal, "desired_state_digest", 71U)) ||
        !tagged_digest(text(journal, "observation_key", 71U)) ||
        !tagged_digest(text(journal, "current_observation_digest", 71U)) ||
        !tagged_digest(text(journal, "current_stable_inventory_digest", 71U)) ||
        !tagged_digest(text(journal, "current_plan_digest", 71U)) ||
        number(journal, "current_plan_revision") == 0U ||
        number(journal, "current_plan_revision") > max_plan_revisions ||
        number(journal, "replan_count") > max_plan_revisions ||
        journal.at("canonical") != false || journal.at("apply_authorized") != false) {
        throw engine::Error("lifecycle_journal.journal_invalid", "lifecycle journal identity is invalid", 5);
    }
    for (const auto* field : {"previous_journal_digest", "prior_applied_state_digest"}) {
        const auto& value = journal.at(field);
        if (!value.is_null() && (!value.is_string() || !tagged_digest(value.get<std::string>()))) {
            throw engine::Error("lifecycle_journal.journal_invalid", std::string(field) + " is invalid", 5);
        }
    }
    if ((number(journal, "generation") == 1U) != journal.at("previous_journal_digest").is_null()) {
        throw engine::Error("lifecycle_journal.journal_invalid", "journal generation and predecessor are inconsistent", 5);
    }
    if (!engine::is_utc_seconds(text(journal, "started_at", 20U)) ||
        !engine::is_utc_seconds(text(journal, "updated_at", 20U)) ||
        text(journal, "started_at", 20U) > text(journal, "updated_at", 20U)) {
        throw engine::Error("lifecycle_journal.journal_invalid", "journal timestamp is invalid", 5);
    }
    if (journal.at("state") == "closed") {
        if (!journal.at("closed_at").is_string() ||
            !engine::is_utc_seconds(journal.at("closed_at").get<std::string>())) {
            throw engine::Error("lifecycle_journal.journal_invalid", "closed journal lacks a timestamp", 5);
        }
    } else if (!journal.at("closed_at").is_null()) {
        throw engine::Error("lifecycle_journal.journal_invalid", "open journal carries a close timestamp", 5);
    }
    if (!journal.at("action_attempts").is_array() || !journal.at("action_attempts").empty() ||
        !journal.at("blockers").is_array() || journal.at("blockers").size() > 4096U ||
        !journal.at("checkpoints").is_array() || journal.at("checkpoints").empty() ||
        journal.at("checkpoints").size() > max_checkpoints) {
        throw engine::Error("lifecycle_journal.journal_invalid", "journal collections are invalid", 5);
    }
    validate_extensions(journal.at("extensions"));
    for (const auto& blocker : journal.at("blockers")) validate_blocker(blocker);
    std::string previous_checkpoint;
    std::uint64_t expected_sequence = 1U;
    for (const auto& checkpoint : journal.at("checkpoints")) {
        validate_checkpoint(checkpoint);
        if (number(checkpoint, "sequence") != expected_sequence ||
            (expected_sequence == 1U) != checkpoint.at("previous_checkpoint_digest").is_null() ||
            (expected_sequence > 1U && checkpoint.at("previous_checkpoint_digest") != previous_checkpoint)) {
            throw engine::Error("lifecycle_journal.journal_invalid", "checkpoint chain is discontinuous", 5);
        }
        previous_checkpoint = checkpoint.at("checkpoint_digest").get<std::string>();
        ++expected_sequence;
    }
    exact_fields(journal.at("compatibility"), {
        "journal_read_versions", "journal_write_version", "desired_read_versions",
        "desired_write_version", "observation_read_versions", "observation_write_version",
        "plan_read_versions", "plan_write_version", "applied_read_versions",
        "applied_write_version", "receipt_read_versions", "required_capabilities",
        "optional_capabilities", "opaque_extensions_preserved"
    }, "journal compatibility");
    if (journal.at("compatibility") != journal_compatibility()) {
        throw engine::Error("lifecycle_journal.journal_incompatible", "journal compatibility is unsupported", 4);
    }
    exact_fields(journal.at("recovery"), {"state", "disposition", "recovered_from_digest", "detail"},
                 "journal recovery");
    verify_digest(journal, "journal_digest", "lifecycle_journal.journal_invalid");
}

void validate_head(const engine::Json& head) {
    exact_fields(head, {
        "protocol", "format_version", "profile_id", "tops_id", "transaction_id",
        "active_slot", "generation", "journal_digest", "previous_head_digest",
        "updated_at", "head_digest"
    }, "lifecycle journal head");
    if (text(head, "protocol") != head_protocol || number(head, "format_version") != format_version ||
        !safe_token(text(head, "profile_id", 256U)) || !safe_token(text(head, "tops_id", 256U)) ||
        !safe_token(text(head, "transaction_id", 256U)) || number(head, "active_slot") > 1U ||
        number(head, "generation") == 0U || !tagged_digest(text(head, "journal_digest", 71U)) ||
        !engine::is_utc_seconds(text(head, "updated_at", 20U))) {
        throw engine::Error("lifecycle_journal.head_invalid", "lifecycle head identity is invalid", 5);
    }
    const auto& previous = head.at("previous_head_digest");
    if (!previous.is_null() && (!previous.is_string() || !tagged_digest(previous.get<std::string>()))) {
        throw engine::Error("lifecycle_journal.head_invalid", "head predecessor is invalid", 5);
    }
    verify_digest(head, "head_digest", "lifecycle_journal.head_invalid");
}

std::optional<FileDescriptor> open_absolute_directory(const std::string& path, bool create) {
    if (!safe_absolute_path(path)) {
        throw engine::Error("lifecycle_journal.state_root_invalid", "state root must be a safe absolute descendant", 4);
    }
    FileDescriptor current(::open("/", O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC));
    if (current.get() < 0) system_error("lifecycle_journal.state_root_open_failed", "could not open filesystem root");
    for (const auto& item : fs::path(path).relative_path()) {
        const auto component = item.string();
        int next = ::openat(current.get(), component.c_str(), O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
        if (next < 0 && errno == ENOENT && !create) return std::nullopt;
        if (next < 0 && errno == ENOENT && create) {
            if (::mkdirat(current.get(), component.c_str(), 0700) != 0 && errno != EEXIST) {
                system_error("lifecycle_journal.state_directory_create_failed", "could not create state directory");
            }
            if (::fsync(current.get()) != 0) {
                system_error("lifecycle_journal.state_sync_failed", "could not synchronize state directory");
            }
            next = ::openat(current.get(), component.c_str(), O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
        }
        if (next < 0) system_error("lifecycle_journal.state_directory_open_failed", "could not open state directory");
        current = FileDescriptor(next);
    }
    struct stat status {};
    if (::fstat(current.get(), &status) != 0) {
        system_error("lifecycle_journal.state_directory_stat_failed", "could not inspect state root");
    }
    if (!S_ISDIR(status.st_mode) || status.st_uid != ::geteuid() || (status.st_mode & 0022) != 0) {
        throw engine::Error(
            "lifecycle_journal.state_directory_unsafe",
            "state root must be caller-owned and not writable by group or other", 5);
    }
    return std::optional<FileDescriptor>(std::move(current));
}

std::optional<FileDescriptor> open_child_directory(
    FileDescriptor parent, const std::string& component, bool create) {
    int next = ::openat(parent.get(), component.c_str(), O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
    if (next < 0 && errno == ENOENT && !create) return std::nullopt;
    if (next < 0 && errno == ENOENT && create) {
        if (::mkdirat(parent.get(), component.c_str(), 0700) != 0 && errno != EEXIST) {
            system_error("lifecycle_journal.state_directory_create_failed", "could not create lifecycle state directory");
        }
        if (::fsync(parent.get()) != 0) {
            system_error("lifecycle_journal.state_sync_failed", "could not synchronize lifecycle state directory");
        }
        next = ::openat(parent.get(), component.c_str(), O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
    }
    if (next < 0) system_error("lifecycle_journal.state_directory_open_failed", "could not open lifecycle state directory");
    FileDescriptor result(next);
    struct stat status {};
    if (::fstat(result.get(), &status) != 0) {
        system_error("lifecycle_journal.state_directory_stat_failed", "could not inspect lifecycle state directory");
    }
    if (!S_ISDIR(status.st_mode) || status.st_uid != ::geteuid() || (status.st_mode & 0022) != 0) {
        throw engine::Error(
            "lifecycle_journal.state_directory_unsafe",
            "lifecycle state directory must be caller-owned and protected", 5);
    }
    if (create && ::fchmod(result.get(), 0700) != 0) {
        system_error("lifecycle_journal.state_directory_mode_failed", "could not restrict lifecycle state directory");
    }
    return std::optional<FileDescriptor>(std::move(result));
}

std::optional<JournalLock> open_stream(const std::string& root, const std::string& tops_id,
                                       const std::string& profile_id, bool exclusive, bool create,
                                       const std::string& storage_version = "v1") {
    auto opened_root = open_absolute_directory(root, create);
    if (!opened_root.has_value()) return std::nullopt;
    FileDescriptor current = std::move(*opened_root);
    const std::array<std::string, 8> components = {
        "symphony", "knowledge-session-coordinator", "lifecycle", storage_version, "tops",
        engine::sha256_hex("tops:" + tops_id), "profiles", engine::sha256_hex("profile:" + profile_id)
    };
    for (const auto& component : components) {
        auto child = open_child_directory(std::move(current), component, create);
        if (!child.has_value()) return std::nullopt;
        current = std::move(*child);
    }
    const int flags = (create ? O_RDWR | O_CREAT : O_RDONLY) |
        O_NONBLOCK | O_NOFOLLOW | O_CLOEXEC;
    const int raw = ::openat(current.get(), ".lock", flags, 0600);
    if (raw < 0 && errno == ENOENT && !create) return std::nullopt;
    if (raw < 0) system_error("lifecycle_journal.lock_open_failed", "could not open lifecycle lock");
    FileDescriptor lock(raw);
    struct stat status {};
    if (::fstat(lock.get(), &status) != 0 || !S_ISREG(status.st_mode) ||
        (status.st_mode & 0777) != 0600 || status.st_uid != ::geteuid() || status.st_nlink != 1) {
        throw engine::Error("lifecycle_journal.lock_unsafe", "lifecycle lock metadata is unsafe", 5);
    }
    const int operation = exclusive ? LOCK_EX | LOCK_NB : LOCK_SH | LOCK_NB;
    if (::flock(lock.get(), operation) != 0) {
        if (errno == EWOULDBLOCK) throw engine::Error("lifecycle_journal.lock_busy", "lifecycle journal is busy", 4);
        system_error("lifecycle_journal.lock_failed", "could not lock lifecycle journal");
    }
    return JournalLock(std::move(current), std::move(lock));
}

std::optional<std::string> read_file(int directory, const std::string& name) {
    const int raw = ::openat(
        directory, name.c_str(), O_RDONLY | O_NONBLOCK | O_NOFOLLOW | O_CLOEXEC);
    if (raw < 0) {
        if (errno == ENOENT) return std::nullopt;
        system_error("lifecycle_journal.state_file_open_failed", "could not open lifecycle state file");
    }
    FileDescriptor file(raw);
    struct stat status {};
    if (::fstat(file.get(), &status) != 0) system_error("lifecycle_journal.state_file_stat_failed", "could not inspect lifecycle state file");
    if (!S_ISREG(status.st_mode) || (status.st_mode & 0777) != 0600 || status.st_uid != ::geteuid() ||
        status.st_nlink != 1 ||
        status.st_size < 0 || static_cast<std::uint64_t>(status.st_size) > max_state_bytes) {
        throw engine::Error("lifecycle_journal.state_file_unsafe", "lifecycle state file metadata is unsafe", 5);
    }
    std::string data;
    data.reserve(static_cast<std::size_t>(status.st_size));
    std::array<char, 16384> buffer {};
    for (;;) {
        const auto count = ::read(file.get(), buffer.data(), buffer.size());
        if (count < 0) {
            if (errno == EINTR) continue;
            system_error("lifecycle_journal.state_file_read_failed", "could not read lifecycle state file");
        }
        if (count == 0) break;
        if (data.size() + static_cast<std::size_t>(count) > max_state_bytes) {
            throw engine::Error("lifecycle_journal.state_file_too_large", "lifecycle state exceeds its bound", 5);
        }
        data.append(buffer.data(), static_cast<std::size_t>(count));
    }
    return data;
}

engine::Json parse_file(const std::string& data, const std::string& context) {
    try {
        return engine::parse_bounded_json(data, max_state_bytes);
    } catch (const engine::Error&) {
        throw engine::Error("lifecycle_journal.state_json_invalid", context + " is invalid bounded JSON", 5);
    }
}

Candidate read_candidate(int directory, int slot) {
    Candidate candidate;
    candidate.slot = slot;
    const auto data = read_file(directory, "journal." + std::to_string(slot) + ".json");
    if (!data.has_value()) return candidate;
    candidate.exists = true;
    try {
        candidate.journal = parse_file(*data, "lifecycle journal slot");
        if (!candidate.journal.is_object() || !candidate.journal.contains("protocol") ||
            !candidate.journal.contains("format_version") ||
            candidate.journal.at("protocol") != journal_protocol ||
            !candidate.journal.at("format_version").is_number_integer() ||
            candidate.journal.at("format_version") != format_version) {
            candidate.incompatible = true;
            return candidate;
        }
        validate_journal(candidate.journal);
        candidate.valid = true;
    } catch (const engine::Error& error) {
        candidate.incompatible = error.code() == "lifecycle_journal.critical_extension_unknown" ||
            error.code() == "lifecycle_journal.journal_incompatible";
        candidate.valid = false;
    }
    return candidate;
}

std::optional<engine::Json> read_head(int directory, bool tolerate_invalid = false) {
    const auto data = read_file(directory, "head.json");
    if (!data.has_value()) return std::nullopt;
    try {
        auto head = parse_file(*data, "lifecycle journal head");
        validate_head(head);
        return head;
    } catch (const engine::Error&) {
        if (tolerate_invalid) return std::nullopt;
        throw;
    }
}

State load_state(int directory) {
    const auto head = read_head(directory);
    if (!head.has_value()) {
        const auto zero = read_candidate(directory, 0);
        const auto one = read_candidate(directory, 1);
        if (zero.exists || one.exists) {
            throw engine::Error("lifecycle_journal.head_missing", "journal slots exist without a valid head; run recover", 5);
        }
        return {};
    }
    const int slot = static_cast<int>(number(*head, "active_slot"));
    const auto active = read_candidate(directory, slot);
    if (!active.valid || active.journal.at("journal_digest") != head->at("journal_digest") ||
        active.journal.at("generation") != head->at("generation") ||
        active.journal.at("profile_id") != head->at("profile_id") ||
        active.journal.at("tops_id") != head->at("tops_id") ||
        active.journal.at("transaction_id") != head->at("transaction_id")) {
        throw engine::Error("lifecycle_journal.head_slot_mismatch", "head does not select a valid matching journal; run recover", 5);
    }
    const auto inactive = read_candidate(directory, 1 - slot);
    if (inactive.incompatible) {
        throw engine::Error("lifecycle_journal.compatibility_required", "inactive slot contains newer or unknown critical state", 4);
    }
    if (inactive.valid && number(inactive.journal, "generation") == number(active.journal, "generation") &&
        inactive.journal.at("journal_digest") != active.journal.at("journal_digest")) {
        throw engine::Error("lifecycle_journal.recovery_ambiguous", "journal slots diverge at the active generation", 5);
    }
    return State{*head, active.journal, true};
}

void write_all(int file, const std::string& data) {
    std::size_t offset = 0U;
    while (offset < data.size()) {
        const auto count = ::write(file, data.data() + offset, data.size() - offset);
        if (count < 0) {
            if (errno == EINTR) continue;
            system_error("lifecycle_journal.state_file_write_failed", "could not write lifecycle state");
        }
        offset += static_cast<std::size_t>(count);
    }
}

void write_slot(int directory, int slot, const engine::Json& journal) {
    const auto name = "journal." + std::to_string(slot) + ".json";
    const int raw = ::openat(
        directory, name.c_str(), O_WRONLY | O_CREAT | O_NONBLOCK | O_NOFOLLOW | O_CLOEXEC, 0600);
    if (raw < 0) system_error("lifecycle_journal.state_file_open_failed", "could not open inactive journal slot");
    FileDescriptor file(raw);
    struct stat status {};
    if (::fstat(file.get(), &status) != 0 || !S_ISREG(status.st_mode) ||
        status.st_uid != ::geteuid() || (status.st_mode & 0777) != 0600 || status.st_nlink != 1) {
        throw engine::Error(
            "lifecycle_journal.state_file_unsafe",
            "inactive journal slot must be a private caller-owned single-link regular file", 5);
    }
    if (::ftruncate(file.get(), 0) != 0) {
        system_error("lifecycle_journal.state_file_write_failed", "could not prepare inactive journal slot");
    }
    write_all(file.get(), journal.dump() + "\n");
    if (::fsync(file.get()) != 0) system_error("lifecycle_journal.state_sync_failed", "could not synchronize journal slot");
}

void write_head(int directory, engine::Json head) {
    finalize_digest(head, "head_digest");
    static std::atomic<std::uint64_t> sequence {0U};
    const auto temporary = ".head.tmp." + std::to_string(::getpid()) + "." +
        std::to_string(sequence.fetch_add(1U, std::memory_order_relaxed));
    const int raw = ::openat(directory, temporary.c_str(), O_WRONLY | O_CREAT | O_EXCL | O_NOFOLLOW | O_CLOEXEC, 0600);
    if (raw < 0) system_error("lifecycle_journal.head_write_failed", "could not create temporary head");
    {
        FileDescriptor file(raw);
        write_all(file.get(), head.dump() + "\n");
        if (::fsync(file.get()) != 0) system_error("lifecycle_journal.state_sync_failed", "could not synchronize temporary head");
    }
    if (::renameat(directory, temporary.c_str(), directory, "head.json") != 0) {
        static_cast<void>(::unlinkat(directory, temporary.c_str(), 0));
        system_error("lifecycle_journal.head_commit_failed", "could not atomically replace lifecycle head");
    }
    if (::fsync(directory) != 0) system_error("lifecycle_journal.state_sync_failed", "could not synchronize lifecycle directory");
}

State commit(int directory, engine::Json journal, const std::optional<engine::Json>& prior_head) {
    finalize_digest(journal, "journal_digest");
    validate_journal(journal);
    const int slot = prior_head.has_value() ?
        1 - static_cast<int>(number(*prior_head, "active_slot")) : 0;
    write_slot(directory, slot, journal);
    if (::fsync(directory) != 0) system_error("lifecycle_journal.state_sync_failed", "could not synchronize lifecycle directory");
    engine::Json head{
        {"protocol", head_protocol}, {"format_version", format_version},
        {"profile_id", journal.at("profile_id")}, {"tops_id", journal.at("tops_id")},
        {"transaction_id", journal.at("transaction_id")}, {"active_slot", slot},
        {"generation", journal.at("generation")}, {"journal_digest", journal.at("journal_digest")},
        {"previous_head_digest", prior_head.has_value() ? prior_head->at("head_digest") : engine::Json(nullptr)},
        {"updated_at", journal.at("updated_at")},
    };
    write_head(directory, head);
    auto stored_head = read_head(directory);
    return State{*stored_head, std::move(journal), true};
}

std::string lifecycle_resource(const std::string& tops_id, const std::string& profile_id,
                               const std::string& evidence) {
    return "symphony.knowledge.lifecycle:" +
        engine::sha256_hex(tops_id + "\n" + profile_id + "\n" + evidence);
}

engine::Json ready_set(const engine::Json& plan) {
    return engine::Json{
        {"ready_action_ids", plan.at("ready_action_ids")},
        {"deferred_action_ids", plan.at("deferred_action_ids")},
        {"blocked_action_ids", plan.at("blocked_action_ids")},
        {"fatal_blockers", plan.at("fatal_blockers")},
    };
}

engine::Json checkpoint(std::uint64_t sequence, const std::string& kind,
                        std::uint64_t plan_revision, const std::string& operation_id,
                        const std::string& observed_at, const std::string& observation_digest,
                        const std::string& ready_digest, const engine::Json& previous) {
    engine::Json value{
        {"sequence", sequence}, {"kind", kind}, {"plan_revision", plan_revision},
        {"operation_id", operation_id}, {"observed_at", observed_at},
        {"observation_digest", observation_digest}, {"ready_set_digest", ready_digest},
        {"previous_checkpoint_digest", previous},
    };
    finalize_digest(value, "checkpoint_digest");
    return value;
}

engine::Json journal_blockers(const engine::Json& plan) {
    engine::Json result = engine::Json::array();
    std::set<std::string> seen;
    const auto add = [&](const engine::Json& blocker, engine::Json& destination, std::set<std::string>& identities) {
        auto value = blocker;
        value["resolved"] = false;
        const auto identity = value.dump();
        if (identities.insert(identity).second) destination.push_back(std::move(value));
    };
    for (const auto& blocker : plan.at("fatal_blockers")) add(blocker, result, seen);
    for (const auto& action : plan.at("actions")) {
        for (const auto& blocker : action.at("blockers")) add(blocker, result, seen);
    }
    std::sort(result.begin(), result.end(), [](const engine::Json& left, const engine::Json& right) {
        return left.dump() < right.dump();
    });
    return result;
}

std::string plan_state(const engine::Json& plan, const engine::Json& blockers) {
    if (!plan.at("fatal_blockers").empty() || !blockers.empty()) return "blocked";
    if (plan.at("actions").empty()) return "verified";
    return "open";
}

void finalize_plan(engine::Json& plan, const State& current) {
    if (!current.present || current.journal.at("state") == "closed") return;
    const auto revision = number(current.journal, "current_plan_revision") + 1U;
    if (revision > max_plan_revisions) {
        throw engine::Error("lifecycle_journal.replan_limit", "lifecycle transaction exhausted its plan revision bound", 4);
    }
    plan["transaction_id"] = current.journal.at("transaction_id");
    plan["revision"] = revision;
    plan["previous_plan_digest"] = current.journal.at("current_plan_digest");
    finalize_digest(plan, "plan_digest");
}

engine::Json begin_or_replan(const engine::Request& request, const State& current,
                             engine::Json& plan) {
    const auto observed = utc_now();
    const auto operation_id = text(request.payload, "operation_id", 256U);
    finalize_plan(plan, current);
    const auto blockers = journal_blockers(plan);
    const auto ready_digest = engine::tagged_sha256(ready_set(plan).dump());
    if (current.present && current.journal.at("operation_id") == operation_id) {
        if (current.journal.at("profile_digest") == request.payload.at("profile_digest") &&
            current.journal.at("mode") == request.payload.at("mode") &&
            current.journal.at("desired_state_digest") == plan.at("desired_state_digest") &&
            current.journal.at("current_observation_digest") == plan.at("observation_digest") &&
            current.journal.at("current_stable_inventory_digest") == request.payload.at("stable_inventory_digest") &&
            current.journal.at("prior_applied_state_digest") == plan.at("prior_applied_state_digest") &&
            current.journal.at("current_plan_digest") == plan.at("plan_digest")) {
            return current.journal;
        }
        throw engine::Error("lifecycle_journal.operation_conflict", "operation ID was reused with different lifecycle evidence", 4);
    }
    const bool continuing = current.present && current.journal.at("state") != "closed";
    const auto generation = current.present ? number(current.journal, "generation") + 1U : 1U;
    engine::Json checkpoints = continuing ? current.journal.at("checkpoints") : engine::Json::array();
    if (checkpoints.size() >= max_checkpoints) {
        throw engine::Error("lifecycle_journal.checkpoint_limit", "lifecycle transaction exhausted its checkpoint bound", 4);
    }
    const engine::Json previous_checkpoint = checkpoints.empty() ? engine::Json(nullptr) :
        checkpoints.back().at("checkpoint_digest");
    checkpoints.push_back(checkpoint(
        checkpoints.size() + 1U, continuing ? "replan" : "begin", number(plan, "revision"), operation_id,
        observed, plan.at("observation_digest").get<std::string>(), ready_digest, previous_checkpoint));
    const auto journal_id = continuing ? current.journal.at("journal_id").get<std::string>() :
        "lifecycle-journal:" + engine::sha256_hex(
            text(request.payload, "tops_id", 256U) + "|" + text(request.payload, "profile_id", 256U) +
            "|" + plan.at("transaction_id").get<std::string>());
    const auto started_at = continuing ? current.journal.at("started_at").get<std::string>() : observed;
    const auto replan_count = continuing ? number(current.journal, "replan_count") + 1U : 0U;
    if (replan_count > max_plan_revisions) {
        throw engine::Error("lifecycle_journal.replan_limit", "lifecycle transaction exhausted its replan bound", 4);
    }
    return engine::Json{
        {"protocol", journal_protocol}, {"format_version", format_version}, {"journal_id", journal_id},
        {"transaction_id", plan.at("transaction_id")}, {"operation_id", operation_id},
        {"generation", generation},
        {"previous_journal_digest", current.present ? current.journal.at("journal_digest") : engine::Json(nullptr)},
        {"profile_id", request.payload.at("profile_id")},
        {"profile_digest", request.payload.at("profile_digest")},
        {"tops_id", request.payload.at("tops_id")},
        {"mode", request.payload.at("mode")}, {"state", plan_state(plan, blockers)},
        {"desired_state_digest", plan.at("desired_state_digest")},
        {"observation_key", continuing ? current.journal.at("observation_key") : plan.at("observation_key")},
        {"current_observation_digest", plan.at("observation_digest")},
        {"current_stable_inventory_digest", request.payload.at("stable_inventory_digest")},
        {"prior_applied_state_digest", plan.at("prior_applied_state_digest")},
        {"current_plan_digest", plan.at("plan_digest")}, {"current_plan_revision", plan.at("revision")},
        {"replan_count", replan_count}, {"action_attempts", engine::Json::array()},
        {"blockers", blockers}, {"checkpoints", std::move(checkpoints)},
        {"compatibility", journal_compatibility()},
        {"extensions", continuing ? current.journal.at("extensions") : engine::Json::array()},
        {"recovery", engine::Json{{"state", "clean"}, {"disposition", "not_applicable"},
            {"recovered_from_digest", nullptr}, {"detail", "no recovery has been required"}}},
        {"started_at", started_at}, {"updated_at", observed}, {"closed_at", nullptr},
        {"canonical", false}, {"apply_authorized", false},
    };
}

engine::Json make_result(const std::string& operation, engine::Json compatibility, const State& state,
                         engine::Json plan, bool changed, bool recovered,
                         const std::vector<std::string>& actions, bool read_only) {
    return engine::Json{
        {"protocol", result_protocol}, {"operation", operation},
        {"compatibility", std::move(compatibility)}, {"journal_present", state.present},
        {"journal", state.present ? state.journal : engine::Json(nullptr)},
        {"journal_digest", state.present ? state.journal.at("journal_digest") : engine::Json(nullptr)},
        {"plan", std::move(plan)}, {"changed", changed}, {"recovered", recovered},
        {"repair_actions", actions}, {"read_only", read_only},
        {"apply_authorized", false}, {"canonical", false},
    };
}

void require_expected(const State& state, const std::string& expected, bool allow_absent) {
    if (expected == "absent") {
        if (!allow_absent || state.present) {
            throw engine::Error("lifecycle_journal.expected_state_mismatch", "lifecycle journal was not absent", 4);
        }
        return;
    }
    if (!tagged_digest(expected) || !state.present || state.journal.at("journal_digest") != expected) {
        throw engine::Error("lifecycle_journal.expected_state_mismatch",
                            state.present ? "lifecycle journal digest is stale" : "lifecycle journal is absent", 4);
    }
}

State recover_state(const engine::Request& request, int directory, std::vector<std::string>* actions,
                    bool* changed) {
    const auto head = read_head(directory, true);
    auto zero = read_candidate(directory, 0);
    auto one = read_candidate(directory, 1);
    if (zero.incompatible || one.incompatible) {
        throw engine::Error("lifecycle_journal.compatibility_required", "recovery found newer or unknown critical state", 4);
    }
    std::vector<Candidate*> valid;
    if (zero.valid) valid.push_back(&zero);
    if (one.valid) valid.push_back(&one);
    if (valid.empty()) {
        if (!zero.exists && !one.exists && !head.has_value() && request.payload.at("expected_journal_digest") == "discover") {
            actions->push_back("no lifecycle boot journal was discovered");
            return {};
        }
        throw engine::Error("lifecycle_journal.unrecoverable", "no valid lifecycle journal slot remains", 5);
    }
    const auto profile_id = request.payload.at("profile_id");
    const auto tops_id = request.payload.at("tops_id");
    for (const auto* candidate : valid) {
        if (candidate->journal.at("profile_id") != profile_id || candidate->journal.at("tops_id") != tops_id) {
            throw engine::Error("lifecycle_journal.unrecoverable", "journal slot belongs to another lifecycle stream", 5);
        }
    }
    Candidate* selected = valid.front();
    if (valid.size() == 2U) {
        auto* lower = valid[0];
        auto* higher = valid[1];
        const auto left_generation = number(lower->journal, "generation");
        const auto right_generation = number(higher->journal, "generation");
        if (left_generation == right_generation) {
            if (lower->journal.at("journal_digest") != higher->journal.at("journal_digest")) {
                throw engine::Error("lifecycle_journal.recovery_ambiguous", "journals diverge at the same generation", 5);
            }
            selected = lower;
        } else {
            if (left_generation > right_generation) std::swap(lower, higher);
            if (number(higher->journal, "generation") != number(lower->journal, "generation") + 1U ||
                higher->journal.at("previous_journal_digest") != lower->journal.at("journal_digest")) {
                throw engine::Error("lifecycle_journal.recovery_ambiguous", "journal slots do not form one linked forward chain", 5);
            }
            selected = higher;
        }
    }
    const auto expected = request.payload.at("expected_journal_digest").get<std::string>();
    if (expected != "discover" && selected->journal.at("journal_digest") != expected) {
        throw engine::Error("lifecycle_journal.expected_state_mismatch", "recovery expected digest is stale", 4);
    }
    const bool healthy = head.has_value() && head->at("journal_digest") == selected->journal.at("journal_digest") &&
        head->at("generation") == selected->journal.at("generation") &&
        head->at("profile_id") == profile_id && head->at("tops_id") == tops_id;
    State baseline{head.value_or(engine::Json{}), selected->journal, true};
    if (healthy) {
        actions->push_back("lifecycle head and active journal are already healthy");
        return baseline;
    }
    auto next = baseline.journal;
    const auto observed = utc_now();
    next["generation"] = number(next, "generation") + 1U;
    next["previous_journal_digest"] = baseline.journal.at("journal_digest");
    next["operation_id"] = request.payload.at("operation_id");
    next["updated_at"] = observed;
    const auto disposition = head.has_value() &&
        number(selected->journal, "generation") == number(*head, "generation") + 1U &&
        selected->journal.at("previous_journal_digest") == head->at("journal_digest") ?
        "adopted_linked_successor" : "repaired_head";
    const auto previous = next.at("checkpoints").back().at("checkpoint_digest");
    next["checkpoints"].push_back(checkpoint(
        next.at("checkpoints").size() + 1U, "recover", number(next, "current_plan_revision"),
        text(request.payload, "operation_id", 256U), observed,
        text(next, "current_observation_digest", 71U),
        next.at("checkpoints").back().at("ready_set_digest").get<std::string>(), previous));
    next["recovery"] = engine::Json{
        {"state", "recovered"}, {"disposition", disposition},
        {"recovered_from_digest", baseline.journal.at("journal_digest")},
        {"detail", "recovered from one uniquely selected digest-linked local lifecycle state"},
    };
    auto committed = commit(directory, std::move(next), head);
    actions->push_back("committed a forward recovery checkpoint and repaired the lifecycle head");
    *changed = true;
    return committed;
}

engine::Json apply_journal_compatibility() {
    return engine::Json{
        {"journal_read_versions", engine::Json::array({1, 2})}, {"journal_write_version", 2},
        {"source_journal_read_versions", engine::Json::array({1})},
        {"applied_read_versions", engine::Json::array({1})}, {"applied_write_version", 1},
        {"required_capabilities", apply_required_capabilities},
        {"optional_capabilities", apply_optional_capabilities},
        {"opaque_extensions_preserved", true},
    };
}

engine::Json apply_compatibility_result(const engine::Json& client, const engine::Json* journal) {
    exact_fields(client, {
        "client_id", "client_version", "process_protocols", "journal_read_versions",
        "journal_write_versions", "capabilities"
    }, "lifecycle apply client");
    if (text(client, "client_id", 128U) != "qxctl" || !safe_version(text(client, "client_version", 64U))) {
        throw engine::Error("lifecycle_apply.client_invalid", "lifecycle apply client identity is invalid", 4);
    }
    const auto processes = token_array(client.at("process_protocols"), "process_protocols", 8U, true);
    const auto reads = version_array(client.at("journal_read_versions"), "journal_read_versions");
    const auto writes = version_array(client.at("journal_write_versions"), "journal_write_versions");
    const auto capabilities = token_array(client.at("capabilities"), "capabilities", 128U);
    std::vector<std::string> missing;
    for (const auto& capability : apply_required_capabilities) {
        if (!contains(capabilities, capability)) missing.push_back(capability);
    }
    const bool process_ok = contains(processes, engine::process_protocol_v1);
    const bool read_ok = contains(reads, apply_format_version);
    const bool write_ok = contains(writes, apply_format_version);
    const bool stored_ok = journal == nullptr || number(*journal, "format_version") == apply_format_version;
    const bool full = process_ok && read_ok && write_ok && stored_ok && missing.empty();
    const bool readable = process_ok && read_ok && stored_ok;
    return engine::Json{
        {"mode", full ? "full" : (readable ? "read_only" : "blocked")},
        {"process_protocol", process_ok ? engine::Json(engine::process_protocol_v1) : engine::Json(nullptr)},
        {"journal_read_version", read_ok && stored_ok ? engine::Json(apply_format_version) : engine::Json(nullptr)},
        {"journal_write_version", write_ok ? engine::Json(apply_format_version) : engine::Json(nullptr)},
        {"missing_capabilities", missing}, {"two_way_procedural_compatibility", true},
        {"reason", full ? "client and coordinator share the complete apply-capable v2 contract" :
            (readable ? "apply state is readable but mutation capability is incomplete" :
                        "client and coordinator have no safe apply journal read overlap")},
    };
}

void validate_apply_attempt(const engine::Json& attempt) {
    exact_fields(attempt, {
        "sequence", "plan_revision", "action_id", "component_id", "kind", "direction",
        "attempt", "state", "blocker_class", "expected_before_digest", "observed_after_digest",
        "evidence_digest", "started_at", "completed_at"
    }, "lifecycle apply attempt");
    static const std::set<std::string> kinds = {
        "install", "uninstall", "select", "deselect", "activate", "deactivate", "dock", "undock"
    };
    static const std::set<std::string> states = {
        "started", "committed", "already_applied", "blocked", "failed"
    };
    if (number(attempt, "sequence") == 0U || number(attempt, "sequence") > max_checkpoints ||
        number(attempt, "plan_revision") == 0U || number(attempt, "plan_revision") > max_plan_revisions ||
        !safe_token(text(attempt, "action_id", 256U)) || !safe_token(text(attempt, "component_id", 256U)) ||
        !kinds.contains(text(attempt, "kind", 32U)) ||
        (attempt.at("direction") != "forward" && attempt.at("direction") != "inverse" &&
         attempt.at("direction") != "neutral") ||
        number(attempt, "attempt") == 0U || number(attempt, "attempt") > 8U ||
        !states.contains(text(attempt, "state", 32U)) ||
        !engine::is_utc_seconds(text(attempt, "started_at", 20U))) {
        throw engine::Error("lifecycle_apply.journal_invalid", "lifecycle action attempt is invalid", 5);
    }
    for (const auto* field : {"expected_before_digest", "observed_after_digest", "evidence_digest"}) {
        const auto& value = attempt.at(field);
        if (!value.is_null() && (!value.is_string() || !tagged_digest(value.get<std::string>()))) {
            throw engine::Error("lifecycle_apply.journal_invalid", std::string(field) + " is invalid", 5);
        }
    }
    const bool started = attempt.at("state") == "started";
    if (started != attempt.at("completed_at").is_null() ||
        (!started && (!attempt.at("completed_at").is_string() ||
                      !engine::is_utc_seconds(attempt.at("completed_at").get<std::string>()))) ||
        (started && (!attempt.at("observed_after_digest").is_null() || !attempt.at("evidence_digest").is_null()))) {
        throw engine::Error("lifecycle_apply.journal_invalid", "lifecycle action attempt completion is invalid", 5);
    }
    static const std::set<std::string> blocker_classes = {
        "dependency_wait", "observation_retryable", "compatibility_blocked", "authorization_denied",
        "integrity_fatal", "critical_state_unknown", "cycle_detected"
    };
    if (!attempt.at("blocker_class").is_null() &&
        (!attempt.at("blocker_class").is_string() ||
         !blocker_classes.contains(attempt.at("blocker_class").get<std::string>()))) {
        throw engine::Error("lifecycle_apply.journal_invalid", "lifecycle action blocker class is invalid", 5);
    }
    const bool unsuccessful = attempt.at("state") == "blocked" || attempt.at("state") == "failed";
    if ((!started && attempt.at("completed_at").get<std::string>() < attempt.at("started_at").get<std::string>()) ||
        (unsuccessful && attempt.at("blocker_class").is_null()) ||
        (!started && !unsuccessful && !attempt.at("blocker_class").is_null())) {
        throw engine::Error("lifecycle_apply.journal_invalid", "lifecycle action outcome evidence is inconsistent", 5);
    }
}

void validate_apply_action(const engine::Json& action) {
    exact_fields(action, {
        "action_id", "component_id", "kind", "direction", "prerequisite_action_ids",
        "inverse_action_id", "expected_before_digest", "target_state_digest", "target_receptor_id",
        "expected_artifact_digests", "expected_evidence", "disposition", "blockers"
    }, "lifecycle apply action");
    static const std::set<std::string> kinds = {
        "install", "uninstall", "select", "deselect", "activate", "deactivate", "dock", "undock"
    };
    static const std::set<std::string> directions = {"forward", "inverse", "neutral"};
    static const std::set<std::string> dispositions = {"ready", "blocked"};
    if (!safe_token(text(action, "action_id", 256U)) || !safe_token(text(action, "component_id", 256U)) ||
        !kinds.contains(text(action, "kind", 32U)) ||
        !directions.contains(text(action, "direction", 32U)) ||
        !dispositions.contains(text(action, "disposition", 32U)) ||
        !tagged_digest(text(action, "target_state_digest", 71U)) ||
        !action.at("prerequisite_action_ids").is_array() || !action.at("prerequisite_action_ids").empty() ||
        !action.at("expected_artifact_digests").is_array() ||
        action.at("expected_artifact_digests").size() > 4096U ||
        !action.at("expected_evidence").is_array() || action.at("expected_evidence").size() > 128U ||
        !action.at("blockers").is_array() || action.at("blockers").size() > 1U) {
        throw engine::Error("lifecycle_apply.journal_invalid", "lifecycle apply action is invalid", 5);
    }
    for (const auto* field : {"inverse_action_id", "target_receptor_id"}) {
        const auto& value = action.at(field);
        if (!value.is_null() && (!value.is_string() || !safe_token(value.get<std::string>()))) {
            throw engine::Error("lifecycle_apply.journal_invalid", std::string(field) + " is invalid", 5);
        }
    }
    const auto kind = text(action, "kind", 32U);
    const auto& target_receptor = action.at("target_receptor_id");
    if ((kind == "dock") != target_receptor.is_string() ||
        (kind != "dock" && kind != "undock" && !target_receptor.is_null())) {
        throw engine::Error(
            "lifecycle_apply.journal_invalid",
            "lifecycle action carries an invalid Maestro receptor target",
            5);
    }
    if (!action.at("expected_before_digest").is_null() &&
        (!action.at("expected_before_digest").is_string() ||
         !tagged_digest(action.at("expected_before_digest").get<std::string>()))) {
        throw engine::Error("lifecycle_apply.journal_invalid", "action pre-state digest is invalid", 5);
    }
    std::set<std::string> artifacts;
    for (const auto& value : action.at("expected_artifact_digests")) {
        if (!value.is_string() || !tagged_digest(value.get<std::string>()) ||
            !artifacts.insert(value.get<std::string>()).second) {
            throw engine::Error("lifecycle_apply.journal_invalid", "action artifact evidence is invalid", 5);
        }
    }
    static_cast<void>(token_array(action.at("expected_evidence"), "expected_evidence", 128U));
    for (const auto& blocker : action.at("blockers")) {
        exact_fields(blocker, {"class", "component_id", "action_id", "retryable", "detail"},
                     "lifecycle apply action blocker");
        static const std::set<std::string> classes = {
            "dependency_wait", "observation_retryable", "compatibility_blocked", "authorization_denied",
            "integrity_fatal", "critical_state_unknown", "cycle_detected"
        };
        const auto blocker_class = text(blocker, "class", 64U);
        if (!classes.contains(blocker_class) || !safe_token(text(blocker, "component_id", 256U)) ||
            !blocker.at("retryable").is_boolean() || text(blocker, "detail", 4096U).empty() ||
            (!blocker.at("action_id").is_string() || !safe_token(blocker.at("action_id").get<std::string>())) ||
            ((blocker_class == "dependency_wait" || blocker_class == "observation_retryable") &&
             blocker.at("retryable") != true) ||
            ((blocker_class == "authorization_denied" || blocker_class == "integrity_fatal" ||
              blocker_class == "critical_state_unknown" || blocker_class == "cycle_detected") &&
             blocker.at("retryable") != false)) {
            throw engine::Error("lifecycle_apply.journal_invalid", "action blocker evidence is invalid", 5);
        }
    }
    if ((action.at("disposition") == "ready") != action.at("blockers").empty()) {
        throw engine::Error("lifecycle_apply.journal_invalid", "action disposition and blocker evidence differ", 5);
    }
}

void validate_apply_journal(const engine::Json& journal) {
    exact_fields(journal, {
        "protocol", "format_version", "journal_id", "transaction_id", "operation_id", "generation",
        "previous_journal_digest", "source_report_journal_digest", "profile_id", "profile_digest",
        "tops_id", "mode", "state", "desired_state_digest", "observation_key",
        "current_observation_digest", "current_stable_inventory_digest", "prior_applied_state_digest",
        "current_plan_digest", "current_plan_revision", "replan_count", "active_action",
        "action_attempts", "blockers", "checkpoints", "compatibility", "extensions", "recovery",
        "applied_state_digest", "started_at", "updated_at", "closed_at", "canonical",
        "apply_authorized", "journal_digest"
    }, "apply-capable lifecycle journal");
    static const std::set<std::string> states = {"open", "acting", "blocked", "verified", "closed"};
    if (journal.at("protocol") != apply_journal_protocol || number(journal, "format_version") != apply_format_version ||
        !safe_token(text(journal, "journal_id", 256U)) || !safe_token(text(journal, "transaction_id", 256U)) ||
        !safe_token(text(journal, "operation_id", 256U)) || !safe_token(text(journal, "profile_id", 256U)) ||
        !safe_token(text(journal, "tops_id", 256U)) || journal.at("mode") != "apply-compatible" ||
        !states.contains(text(journal, "state", 32U)) || number(journal, "generation") == 0U ||
        !tagged_digest(text(journal, "source_report_journal_digest", 71U)) ||
        !tagged_digest(text(journal, "profile_digest", 71U)) ||
        !tagged_digest(text(journal, "desired_state_digest", 71U)) ||
        !tagged_digest(text(journal, "observation_key", 71U)) ||
        !tagged_digest(text(journal, "current_observation_digest", 71U)) ||
        !tagged_digest(text(journal, "current_stable_inventory_digest", 71U)) ||
        !tagged_digest(text(journal, "current_plan_digest", 71U)) ||
        number(journal, "current_plan_revision") == 0U ||
        number(journal, "current_plan_revision") > max_plan_revisions ||
        number(journal, "replan_count") > max_plan_revisions || journal.at("canonical") != false ||
        journal.at("apply_authorized") != true || !engine::is_utc_seconds(text(journal, "started_at", 20U)) ||
        !engine::is_utc_seconds(text(journal, "updated_at", 20U)) ||
        text(journal, "started_at", 20U) > text(journal, "updated_at", 20U)) {
        throw engine::Error("lifecycle_apply.journal_invalid", "apply journal identity is invalid", 5);
    }
    for (const auto* field : {"previous_journal_digest", "prior_applied_state_digest", "applied_state_digest"}) {
        const auto& value = journal.at(field);
        if (!value.is_null() && (!value.is_string() || !tagged_digest(value.get<std::string>()))) {
            throw engine::Error("lifecycle_apply.journal_invalid", std::string(field) + " is invalid", 5);
        }
    }
    if ((number(journal, "generation") == 1U) != journal.at("previous_journal_digest").is_null()) {
        throw engine::Error("lifecycle_apply.journal_invalid", "apply journal predecessor is invalid", 5);
    }
    if ((journal.at("state") == "acting") != journal.at("active_action").is_object()) {
        throw engine::Error("lifecycle_apply.journal_invalid", "active action and journal state differ", 5);
    }
    if (journal.at("active_action").is_object()) validate_apply_action(journal.at("active_action"));
    if (!journal.at("action_attempts").is_array() || journal.at("action_attempts").size() > max_checkpoints ||
        !journal.at("blockers").is_array() || journal.at("blockers").size() > 4096U ||
        !journal.at("checkpoints").is_array() || journal.at("checkpoints").empty() ||
        journal.at("checkpoints").size() > max_checkpoints) {
        throw engine::Error("lifecycle_apply.journal_invalid", "apply journal collections are invalid", 5);
    }
    std::uint64_t expected_attempt = 1U;
    std::size_t started_attempts = 0U;
    for (const auto& attempt : journal.at("action_attempts")) {
        validate_apply_attempt(attempt);
        if (number(attempt, "sequence") != expected_attempt++) {
            throw engine::Error("lifecycle_apply.journal_invalid", "apply attempt sequence is discontinuous", 5);
        }
        if (attempt.at("started_at") < journal.at("started_at") ||
            (!attempt.at("completed_at").is_null() && attempt.at("completed_at") > journal.at("updated_at"))) {
            throw engine::Error("lifecycle_apply.journal_invalid", "apply attempt timestamp is outside its journal", 5);
        }
        if (attempt.at("state") == "started") ++started_attempts;
    }
    if (journal.at("state") == "acting") {
        if (started_attempts != 1U || journal.at("action_attempts").empty() ||
            journal.at("action_attempts").back().at("state") != "started") {
            throw engine::Error("lifecycle_apply.journal_invalid", "acting journal lacks one active attempt", 5);
        }
        const auto& active = journal.at("active_action");
        const auto& attempt = journal.at("action_attempts").back();
        if (attempt.at("action_id") != active.at("action_id") ||
            attempt.at("component_id") != active.at("component_id") ||
            attempt.at("kind") != active.at("kind") || attempt.at("direction") != active.at("direction") ||
            attempt.at("expected_before_digest") != active.at("expected_before_digest")) {
            throw engine::Error("lifecycle_apply.journal_invalid", "active action and attempt evidence differ", 5);
        }
    } else if (std::any_of(journal.at("action_attempts").begin(), journal.at("action_attempts").end(),
                           [](const engine::Json& attempt) { return attempt.at("state") == "started"; })) {
        throw engine::Error("lifecycle_apply.journal_invalid", "non-acting journal contains an unfinished attempt", 5);
    }
    validate_extensions(journal.at("extensions"));
    for (const auto& blocker : journal.at("blockers")) validate_blocker(blocker);
    std::string previous_checkpoint;
    std::uint64_t expected_checkpoint = 1U;
    for (const auto& item : journal.at("checkpoints")) {
        validate_checkpoint(item);
        if (number(item, "sequence") != expected_checkpoint ||
            (expected_checkpoint == 1U) != item.at("previous_checkpoint_digest").is_null() ||
            (expected_checkpoint > 1U && item.at("previous_checkpoint_digest") != previous_checkpoint)) {
            throw engine::Error("lifecycle_apply.journal_invalid", "apply checkpoint chain is discontinuous", 5);
        }
        if (item.at("observed_at") < journal.at("started_at") || item.at("observed_at") > journal.at("updated_at")) {
            throw engine::Error("lifecycle_apply.journal_invalid", "apply checkpoint timestamp is outside its journal", 5);
        }
        previous_checkpoint = item.at("checkpoint_digest").get<std::string>();
        ++expected_checkpoint;
    }
    if (journal.at("compatibility") != apply_journal_compatibility()) {
        throw engine::Error("lifecycle_apply.journal_incompatible", "apply journal compatibility is unsupported", 4);
    }
    exact_fields(journal.at("recovery"), {"state", "disposition", "recovered_from_digest", "detail"},
                 "apply journal recovery");
    static const std::set<std::string> recovery_states = {"clean", "recovered"};
    static const std::set<std::string> recovery_dispositions = {
        "not_applicable", "adopted_linked_successor", "repaired_head", "resumed_compatible_actions",
        "replanned_after_evidence_change", "closed_verified_transaction"
    };
    const auto& recovery = journal.at("recovery");
    if (!recovery_states.contains(text(recovery, "state", 32U)) ||
        !recovery_dispositions.contains(text(recovery, "disposition", 64U)) ||
        text(recovery, "detail", 4096U).empty() ||
        (!recovery.at("recovered_from_digest").is_null() &&
         (!recovery.at("recovered_from_digest").is_string() ||
          !tagged_digest(recovery.at("recovered_from_digest").get<std::string>()))) ||
        (recovery.at("state") == "clean") != recovery.at("recovered_from_digest").is_null() ||
        (recovery.at("state") == "clean") != (recovery.at("disposition") == "not_applicable")) {
        throw engine::Error("lifecycle_apply.journal_invalid", "apply recovery evidence is invalid", 5);
    }
    if (journal.at("state") == "closed") {
        if (!journal.at("closed_at").is_string() || !engine::is_utc_seconds(journal.at("closed_at").get<std::string>()) ||
            journal.at("closed_at").get<std::string>() < journal.at("updated_at").get<std::string>() ||
            !journal.at("applied_state_digest").is_string()) {
            throw engine::Error("lifecycle_apply.journal_invalid", "closed apply journal lacks applied evidence", 5);
        }
    } else if (!journal.at("closed_at").is_null()) {
        throw engine::Error("lifecycle_apply.journal_invalid", "open apply journal carries a close time", 5);
    }
    verify_digest(journal, "journal_digest", "lifecycle_apply.journal_invalid");
}

void validate_apply_head(const engine::Json& head) {
    exact_fields(head, {
        "protocol", "format_version", "profile_id", "tops_id", "transaction_id", "active_slot",
        "generation", "journal_digest", "previous_head_digest", "updated_at", "head_digest"
    }, "apply-capable lifecycle head");
    if (head.at("protocol") != apply_head_protocol || number(head, "format_version") != apply_format_version ||
        !safe_token(text(head, "profile_id", 256U)) || !safe_token(text(head, "tops_id", 256U)) ||
        !safe_token(text(head, "transaction_id", 256U)) || number(head, "active_slot") > 1U ||
        number(head, "generation") == 0U || !tagged_digest(text(head, "journal_digest", 71U)) ||
        !engine::is_utc_seconds(text(head, "updated_at", 20U))) {
        throw engine::Error("lifecycle_apply.head_invalid", "apply head identity is invalid", 5);
    }
    if (!head.at("previous_head_digest").is_null() &&
        (!head.at("previous_head_digest").is_string() ||
         !tagged_digest(head.at("previous_head_digest").get<std::string>()))) {
        throw engine::Error("lifecycle_apply.head_invalid", "apply head predecessor is invalid", 5);
    }
    verify_digest(head, "head_digest", "lifecycle_apply.head_invalid");
}

Candidate read_apply_candidate(int directory, int slot) {
    Candidate candidate;
    candidate.slot = slot;
    const auto data = read_file(directory, "journal." + std::to_string(slot) + ".json");
    if (!data.has_value()) return candidate;
    candidate.exists = true;
    try {
        candidate.journal = parse_file(*data, "apply lifecycle journal slot");
        if (!candidate.journal.is_object() || !candidate.journal.contains("protocol") ||
            !candidate.journal.contains("format_version") ||
            candidate.journal.at("protocol") != apply_journal_protocol ||
            candidate.journal.at("format_version") != apply_format_version) {
            candidate.incompatible = true;
            return candidate;
        }
        validate_apply_journal(candidate.journal);
        candidate.valid = true;
    } catch (const engine::Error& error) {
        candidate.incompatible = error.code() == "lifecycle_journal.critical_extension_unknown" ||
            error.code() == "lifecycle_apply.journal_incompatible";
    }
    return candidate;
}

std::optional<engine::Json> read_apply_head(int directory, bool tolerate_invalid = false) {
    const auto data = read_file(directory, "head.json");
    if (!data.has_value()) return std::nullopt;
    try {
        auto head = parse_file(*data, "apply lifecycle head");
        validate_apply_head(head);
        return head;
    } catch (const engine::Error&) {
        if (tolerate_invalid) return std::nullopt;
        throw;
    }
}

State load_apply_state(int directory) {
    const auto head = read_apply_head(directory);
    if (!head.has_value()) {
        const auto zero = read_apply_candidate(directory, 0);
        const auto one = read_apply_candidate(directory, 1);
        if (zero.exists || one.exists) {
            throw engine::Error("lifecycle_apply.head_missing", "apply slots exist without a valid head; run recover", 5);
        }
        return {};
    }
    const int slot = static_cast<int>(number(*head, "active_slot"));
    const auto active = read_apply_candidate(directory, slot);
    if (!active.valid || active.journal.at("journal_digest") != head->at("journal_digest") ||
        active.journal.at("generation") != head->at("generation") ||
        active.journal.at("profile_id") != head->at("profile_id") ||
        active.journal.at("tops_id") != head->at("tops_id") ||
        active.journal.at("transaction_id") != head->at("transaction_id")) {
        throw engine::Error("lifecycle_apply.head_slot_mismatch", "apply head does not select a valid journal", 5);
    }
    const auto inactive = read_apply_candidate(directory, 1 - slot);
    if (inactive.incompatible) {
        throw engine::Error("lifecycle_apply.compatibility_required", "inactive apply slot is incompatible", 4);
    }
    if (inactive.valid && number(inactive.journal, "generation") == number(active.journal, "generation") &&
        inactive.journal.at("journal_digest") != active.journal.at("journal_digest")) {
        throw engine::Error("lifecycle_apply.recovery_ambiguous", "apply slots diverge at one generation", 5);
    }
    return State{*head, active.journal, true};
}

void write_apply_head(int directory, engine::Json head) {
    finalize_digest(head, "head_digest");
    static std::atomic<std::uint64_t> apply_sequence {0U};
    const auto temporary = ".head.tmp.apply." + std::to_string(::getpid()) + "." +
        std::to_string(apply_sequence.fetch_add(1U, std::memory_order_relaxed));
    const int raw = ::openat(directory, temporary.c_str(),
        O_WRONLY | O_CREAT | O_EXCL | O_NOFOLLOW | O_CLOEXEC, 0600);
    if (raw < 0) system_error("lifecycle_apply.head_write_failed", "could not create temporary apply head");
    {
        FileDescriptor file(raw);
        write_all(file.get(), head.dump() + "\n");
        if (::fsync(file.get()) != 0) system_error("lifecycle_apply.state_sync_failed", "could not synchronize apply head");
    }
    if (::renameat(directory, temporary.c_str(), directory, "head.json") != 0) {
        static_cast<void>(::unlinkat(directory, temporary.c_str(), 0));
        system_error("lifecycle_apply.head_commit_failed", "could not atomically replace apply head");
    }
    if (::fsync(directory) != 0) system_error("lifecycle_apply.state_sync_failed", "could not synchronize apply directory");
}

State commit_apply(int directory, engine::Json journal, const std::optional<engine::Json>& prior_head) {
    finalize_digest(journal, "journal_digest");
    validate_apply_journal(journal);
    const int slot = prior_head.has_value() ? 1 - static_cast<int>(number(*prior_head, "active_slot")) : 0;
    write_slot(directory, slot, journal);
    if (::fsync(directory) != 0) system_error("lifecycle_apply.state_sync_failed", "could not synchronize apply directory");
    engine::Json head{
        {"protocol", apply_head_protocol}, {"format_version", apply_format_version},
        {"profile_id", journal.at("profile_id")}, {"tops_id", journal.at("tops_id")},
        {"transaction_id", journal.at("transaction_id")}, {"active_slot", slot},
        {"generation", journal.at("generation")}, {"journal_digest", journal.at("journal_digest")},
        {"previous_head_digest", prior_head.has_value() ? prior_head->at("head_digest") : engine::Json(nullptr)},
        {"updated_at", journal.at("updated_at")},
    };
    write_apply_head(directory, head);
    return State{*read_apply_head(directory), std::move(journal), true};
}

std::string applied_state_name(const std::string& digest) {
    if (!tagged_digest(digest)) {
        throw engine::Error("lifecycle_apply.applied_state_invalid", "applied state digest is invalid", 5);
    }
    return "applied." + digest.substr(7U) + ".json";
}

void validate_applied_state(const engine::Json& applied, const State* selector = nullptr) {
    exact_fields(applied, {
        "protocol", "format_version", "profile_id", "tops_id", "generation",
        "previous_applied_state_digest", "stabilized_observation_key", "desired_state_digest",
        "verified_observation_digest", "plan_digest", "transaction_id", "components",
        "execution_trace", "unresolved_blockers", "extensions", "canonical", "applied_state_digest"
    }, "lifecycle applied state");
    if (applied.at("protocol") != "symphony.knowledge.lifecycle-applied-state.v1" ||
        number(applied, "format_version") != 1U || number(applied, "generation") == 0U ||
        !safe_token(text(applied, "profile_id", 256U)) || !safe_token(text(applied, "tops_id", 256U)) ||
        !safe_token(text(applied, "transaction_id", 256U)) ||
        !tagged_digest(text(applied, "stabilized_observation_key", 71U)) ||
        !tagged_digest(text(applied, "desired_state_digest", 71U)) ||
        !tagged_digest(text(applied, "verified_observation_digest", 71U)) ||
        !tagged_digest(text(applied, "plan_digest", 71U)) || applied.at("canonical") != false ||
        !tagged_digest(text(applied, "applied_state_digest", 71U))) {
        throw engine::Error("lifecycle_apply.applied_state_invalid", "applied state identity is invalid", 5);
    }
    const auto& previous = applied.at("previous_applied_state_digest");
    if ((number(applied, "generation") == 1U) != previous.is_null() ||
        (!previous.is_null() && (!previous.is_string() || !tagged_digest(previous.get<std::string>())))) {
        throw engine::Error("lifecycle_apply.applied_state_invalid", "applied state predecessor is invalid", 5);
    }
    if (!applied.at("components").is_array() || applied.at("components").size() > 4096U ||
        !applied.at("execution_trace").is_array() || applied.at("execution_trace").size() > max_checkpoints ||
        !applied.at("unresolved_blockers").is_array() || !applied.at("unresolved_blockers").empty()) {
        throw engine::Error("lifecycle_apply.applied_state_invalid", "applied state collections are invalid", 5);
    }
    std::set<std::string> component_ids;
    std::string previous_component;
    for (const auto& component : applied.at("components")) {
        exact_fields(component, {
            "component_id", "selected_receipt_digest", "presence", "activation", "docking",
            "receptor_id", "component_state_digest"
        }, "lifecycle applied component");
        static const std::set<std::string> presences = {"present", "absent", "unmanaged"};
        static const std::set<std::string> activations = {"inactive", "active", "unmanaged", "unknown"};
        static const std::set<std::string> dockings = {"undocked", "docked", "unmanaged", "unavailable", "unknown"};
        const auto id = text(component, "component_id", 256U);
        const auto selected = component.at("selected_receipt_digest");
        const auto receptor = component.at("receptor_id");
        if (!safe_token(id) || !component_ids.insert(id).second ||
            (!previous_component.empty() && id <= previous_component) ||
            !presences.contains(text(component, "presence", 32U)) ||
            !activations.contains(text(component, "activation", 32U)) ||
            !dockings.contains(text(component, "docking", 32U)) ||
            (!selected.is_null() && (!selected.is_string() || !tagged_digest(selected.get<std::string>()))) ||
            (!receptor.is_null() && (!receptor.is_string() || !safe_token(receptor.get<std::string>()))) ||
            ((component.at("docking") == "docked") != receptor.is_string())) {
            throw engine::Error("lifecycle_apply.applied_state_invalid", "applied component evidence is invalid", 5);
        }
        verify_digest(component, "component_state_digest", "lifecycle_apply.applied_state_invalid");
        previous_component = id;
    }
    std::uint64_t expected_sequence = 1U;
    for (const auto& execution : applied.at("execution_trace")) {
        exact_fields(execution, {
            "sequence", "plan_revision", "action_id", "component_id", "direction", "attempt", "outcome",
            "before_digest", "after_digest", "completed_at", "evidence_digest"
        }, "lifecycle applied execution");
        if (number(execution, "sequence") != expected_sequence++ ||
            number(execution, "plan_revision") == 0U || number(execution, "plan_revision") > max_plan_revisions ||
            number(execution, "attempt") == 0U || number(execution, "attempt") > 8U ||
            !safe_token(text(execution, "action_id", 256U)) ||
            !safe_token(text(execution, "component_id", 256U)) ||
            (execution.at("direction") != "forward" && execution.at("direction") != "inverse" &&
             execution.at("direction") != "neutral") ||
            (execution.at("outcome") != "committed" && execution.at("outcome") != "already_applied") ||
            !tagged_digest(text(execution, "after_digest", 71U)) ||
            !tagged_digest(text(execution, "evidence_digest", 71U)) ||
            !engine::is_utc_seconds(text(execution, "completed_at", 20U))) {
            throw engine::Error("lifecycle_apply.applied_state_invalid", "applied execution evidence is invalid", 5);
        }
        const auto& before = execution.at("before_digest");
        if (!before.is_null() && (!before.is_string() || !tagged_digest(before.get<std::string>()))) {
            throw engine::Error("lifecycle_apply.applied_state_invalid", "applied execution pre-state is invalid", 5);
        }
    }
    validate_extensions(applied.at("extensions"));
    verify_digest(applied, "applied_state_digest", "lifecycle_apply.applied_state_invalid");
    if (selector != nullptr && selector->present) {
        const bool identity_mismatch =
            applied.at("profile_id") != selector->journal.at("profile_id") ||
            applied.at("tops_id") != selector->journal.at("tops_id") ||
            applied.at("applied_state_digest") != selector->journal.at("applied_state_digest");
        const bool closed_mismatch = selector->journal.at("state") == "closed" &&
            (applied.at("desired_state_digest") != selector->journal.at("desired_state_digest") ||
             applied.at("verified_observation_digest") != selector->journal.at("current_observation_digest") ||
             applied.at("plan_digest") != selector->journal.at("current_plan_digest") ||
             applied.at("transaction_id") != selector->journal.at("transaction_id"));
        const bool open_mismatch = selector->journal.at("state") != "closed" &&
            applied.at("applied_state_digest") != selector->journal.at("prior_applied_state_digest");
        if (identity_mismatch || closed_mismatch || open_mismatch) {
            throw engine::Error("lifecycle_apply.applied_state_invalid", "applied state does not match its journal selector", 5);
        }
    }
}

std::optional<engine::Json> read_applied_state(int directory, const State& state) {
    if (!state.present || state.journal.at("applied_state_digest").is_null()) return std::nullopt;
    const auto expected = state.journal.at("applied_state_digest").get<std::string>();
    const auto data = read_file(directory, applied_state_name(expected));
    if (!data.has_value()) {
        throw engine::Error("lifecycle_apply.applied_state_missing", "selected applied evidence is absent", 5);
    }
    auto applied = parse_file(*data, "lifecycle applied state");
    if (!applied.is_object()) {
        throw engine::Error("lifecycle_apply.applied_state_invalid", "applied state is not an object", 5);
    }
    validate_applied_state(applied, &state);
    if (applied.at("applied_state_digest") != expected) {
        throw engine::Error("lifecycle_apply.applied_state_invalid", "journal selects different applied evidence", 5);
    }
    return applied;
}

void write_applied_state(int directory, const engine::Json& applied) {
    const auto digest = text(applied, "applied_state_digest", 71U);
    validate_applied_state(applied);
    const auto name = applied_state_name(digest);
    if (const auto existing = read_file(directory, name); existing.has_value()) {
        const auto decoded = parse_file(*existing, "existing lifecycle applied state");
        if (decoded != applied) {
            throw engine::Error("lifecycle_apply.applied_conflict", "content-addressed applied evidence conflicts", 5);
        }
        return;
    }
    static std::atomic<std::uint64_t> applied_sequence {0U};
    const auto temporary = ".applied.tmp." + std::to_string(::getpid()) + "." +
        std::to_string(applied_sequence.fetch_add(1U, std::memory_order_relaxed));
    const int raw = ::openat(directory, temporary.c_str(),
        O_WRONLY | O_CREAT | O_EXCL | O_NOFOLLOW | O_CLOEXEC, 0600);
    if (raw < 0) system_error("lifecycle_apply.applied_write_failed", "could not create temporary applied state");
    {
        FileDescriptor file(raw);
        write_all(file.get(), applied.dump() + "\n");
        if (::fsync(file.get()) != 0) system_error("lifecycle_apply.state_sync_failed", "could not synchronize applied state");
    }
    if (::linkat(directory, temporary.c_str(), directory, name.c_str(), 0) != 0) {
        static_cast<void>(::unlinkat(directory, temporary.c_str(), 0));
        system_error("lifecycle_apply.applied_commit_failed", "could not publish content-addressed applied state");
    }
    static_cast<void>(::unlinkat(directory, temporary.c_str(), 0));
    if (::fsync(directory) != 0) system_error("lifecycle_apply.state_sync_failed", "could not synchronize applied state directory");
}

const engine::Json* find_plan_action(const engine::Json& plan, const std::string& action_id) {
    for (const auto& action : plan.at("actions")) {
        if (action.at("action_id") == action_id) return &action;
    }
    return nullptr;
}

bool digest_array_contains(const engine::Json& values, const std::string& wanted) {
    return std::any_of(values.begin(), values.end(), [&](const engine::Json& item) {
        return item.is_string() && item == wanted;
    });
}

bool apply_action_ready(const engine::Json& action, const engine::Json& available_artifacts) {
    if (action.at("disposition") == "ready") return true;
    if (action.at("kind") != "install" || action.at("disposition") != "blocked" ||
        action.at("expected_artifact_digests").empty() ||
        !action.at("prerequisite_action_ids").empty() || action.at("blockers").size() != 1U) return false;
    for (const auto& digest : action.at("expected_artifact_digests")) {
        if (!digest_array_contains(available_artifacts, digest.get<std::string>())) return false;
    }
    const auto& blocker = action.at("blockers").front();
    return blocker.at("class") == "dependency_wait" &&
        blocker.at("component_id") == action.at("component_id") &&
        blocker.at("action_id") == action.at("action_id");
}

engine::Json apply_attempt(const engine::Json& action, const engine::Json& attempts,
                           std::uint64_t revision, const std::string& observed) {
    std::uint64_t count = 1U;
    for (const auto& prior : attempts) {
        if (prior.at("action_id") == action.at("action_id")) ++count;
    }
    if (count > 8U) throw engine::Error("lifecycle_apply.attempt_limit", "action exhausted its attempt bound", 4);
    return engine::Json{
        {"sequence", attempts.size() + 1U}, {"plan_revision", revision},
        {"action_id", action.at("action_id")}, {"component_id", action.at("component_id")},
        {"kind", action.at("kind")}, {"direction", action.at("direction")}, {"attempt", count},
        {"state", "started"}, {"blocker_class", nullptr},
        {"expected_before_digest", action.at("expected_before_digest")}, {"observed_after_digest", nullptr},
        {"evidence_digest", nullptr}, {"started_at", observed}, {"completed_at", nullptr},
    };
}

engine::Json build_applied_state(const engine::Json& desired, const engine::Json& observation,
                                 const engine::Json& plan, const engine::Json& journal,
                                 const std::optional<engine::Json>& prior) {
    engine::Json components = engine::Json::array();
    for (const auto& wanted : desired.at("components")) {
        const auto id = wanted.at("component_id");
        const engine::Json* observed = nullptr;
        for (const auto& item : observation.at("components")) {
            if (item.at("component_id") == id) { observed = &item; break; }
        }
        engine::Json component{
            {"component_id", id}, {"selected_receipt_digest", nullptr},
            {"presence", observed == nullptr ? "absent" : "present"},
            {"activation", observed == nullptr ? "inactive" : observed->at("activation")},
            {"docking", observed == nullptr ? "undocked" : observed->at("docking")},
            {"receptor_id", observed == nullptr ? engine::Json(nullptr) : observed->at("receptor_id")},
        };
        if (observed != nullptr) component["selected_receipt_digest"] = observed->at("selected_package_digest");
        finalize_digest(component, "component_state_digest");
        components.push_back(std::move(component));
    }
    std::sort(components.begin(), components.end(), [](const engine::Json& left, const engine::Json& right) {
        return left.at("component_id") < right.at("component_id");
    });
    engine::Json trace = engine::Json::array();
    for (const auto& attempt : journal.at("action_attempts")) {
        if (attempt.at("state") != "committed" && attempt.at("state") != "already_applied") continue;
        trace.push_back(engine::Json{
            {"sequence", trace.size() + 1U}, {"plan_revision", attempt.at("plan_revision")},
            {"action_id", attempt.at("action_id")}, {"component_id", attempt.at("component_id")},
            {"direction", attempt.at("direction")}, {"attempt", attempt.at("attempt")},
            {"outcome", attempt.at("state")}, {"before_digest", attempt.at("expected_before_digest")},
            {"after_digest", attempt.at("observed_after_digest")}, {"completed_at", attempt.at("completed_at")},
            {"evidence_digest", attempt.at("evidence_digest")},
        });
    }
    engine::Json applied{
        {"protocol", "symphony.knowledge.lifecycle-applied-state.v1"}, {"format_version", 1},
        {"profile_id", journal.at("profile_id")}, {"tops_id", journal.at("tops_id")},
        {"generation", prior.has_value() ? number(*prior, "generation") + 1U : 1U},
        {"previous_applied_state_digest", prior.has_value() ? prior->at("applied_state_digest") : engine::Json(nullptr)},
        {"stabilized_observation_key", plan.at("observation_key")},
        {"desired_state_digest", desired.at("desired_state_digest")},
        {"verified_observation_digest", observation.at("observation_digest")},
        {"plan_digest", plan.at("plan_digest")}, {"transaction_id", journal.at("transaction_id")},
        {"components", std::move(components)}, {"execution_trace", std::move(trace)},
        {"unresolved_blockers", engine::Json::array()}, {"extensions", engine::Json::array()},
        {"canonical", false},
    };
    finalize_digest(applied, "applied_state_digest");
    return applied;
}

bool plan_is_converged(const engine::Json& plan) {
    if (!plan.at("fatal_blockers").empty()) return false;
    for (const auto& action : plan.at("actions")) {
        if (action.at("kind") == "preserve" || action.at("kind") == "report") continue;
        return false;
    }
    return true;
}

bool apply_action_observed(const engine::Json& action, const engine::Json& desired,
                           const engine::Json& observation) {
    const auto& component_id = action.at("component_id");
    const engine::Json* wanted = nullptr;
    const engine::Json* observed = nullptr;
    for (const auto& component : desired.at("components")) {
        if (component.at("component_id") == component_id) { wanted = &component; break; }
    }
    for (const auto& component : observation.at("components")) {
        if (component.at("component_id") == component_id) { observed = &component; break; }
    }
    const auto kind = action.at("kind").get<std::string>();
    if (kind == "uninstall") return observed == nullptr;
    if (wanted == nullptr) return false;
    if (kind == "install") {
        if (observed == nullptr || wanted->at("selected_package").is_null()) return false;
        const auto& receipt = wanted->at("selected_package").at("receipt_digest");
        return std::any_of(observed->at("packages").begin(), observed->at("packages").end(),
                           [&](const engine::Json& package) {
                               return package.at("receipt_digest") == receipt && package.at("integrity") == "valid";
                           });
    }
    if (kind == "select") {
        return observed != nullptr && !wanted->at("selected_package").is_null() &&
            observed->at("selected_package_digest") == wanted->at("selected_package").at("receipt_digest");
    }
    if (kind == "deselect") {
        return observed == nullptr || observed->at("selected_package_digest").is_null();
    }
    if (kind == "activate") return observed != nullptr && observed->at("activation") == "active";
    if (kind == "deactivate") return observed == nullptr || observed->at("activation") == "inactive";
    if (kind == "dock") {
        return observed != nullptr && observed->at("docking") == "docked" &&
            action.at("target_receptor_id").is_string() &&
            observed->at("receptor_id") == action.at("target_receptor_id");
    }
    if (kind == "undock") {
        return observed == nullptr ||
            (observed->at("docking") == "undocked" && observed->at("receptor_id").is_null());
    }
    return false;
}

engine::Json make_apply_result(const std::string& operation, engine::Json compatibility,
                               const State& state, engine::Json plan, engine::Json action,
                               engine::Json applied, bool changed, bool recovered,
                               const std::vector<std::string>& repair_actions, bool read_only) {
    return engine::Json{
        {"protocol", apply_result_protocol}, {"operation", operation},
        {"compatibility", std::move(compatibility)}, {"journal_present", state.present},
        {"journal", state.present ? state.journal : engine::Json(nullptr)},
        {"journal_digest", state.present ? state.journal.at("journal_digest") : engine::Json(nullptr)},
        {"plan", std::move(plan)}, {"action", std::move(action)}, {"applied_state", std::move(applied)},
        {"changed", changed}, {"recovered", recovered}, {"repair_actions", repair_actions},
        {"read_only", read_only}, {"apply_authorized", !read_only}, {"canonical", false},
    };
}

void require_apply_expected(const State& state, const std::string& expected) {
    if (expected == "absent") {
        if (state.present) throw engine::Error("lifecycle_apply.expected_state_mismatch", "apply journal was not absent", 4);
        return;
    }
    if (!tagged_digest(expected) || !state.present || state.journal.at("journal_digest") != expected) {
        throw engine::Error("lifecycle_apply.expected_state_mismatch", state.present ? "apply journal digest is stale" : "apply journal is absent", 4);
    }
}

engine::Json build_apply_plan(const engine::Request& request, const State& current) {
    engine::Json payload{
        {"protocol", "symphony.knowledge.lifecycle-plan-command.v1"}, {"operation", "lifecycle_plan"},
        {"desired_state", request.payload.at("desired_state")}, {"observation", request.payload.at("observation")},
        {"prior_applied_state_digest", request.payload.at("prior_applied_state_digest")},
        {"client", request.payload.at("planner_client")},
    };
    auto plan = build_lifecycle_plan(payload, request.deadline_unix_ms);
    if (current.present && current.journal.at("state") != "closed") finalize_plan(plan, current);
    return plan;
}

engine::Json validate_apply_command(const engine::Request& request) {
    static const std::set<std::string> operations = {
        "lifecycle_apply_prepare", "lifecycle_apply_finalize", "lifecycle_apply_status",
        "lifecycle_apply_recover", "lifecycle_apply_close"
    };
    if (!operations.contains(request.operation)) {
        throw engine::Error("lifecycle_apply.command_invalid", "unsupported lifecycle apply operation", 4);
    }
    const bool state_only = request.operation == "lifecycle_apply_status" || request.operation == "lifecycle_apply_recover";
    if (state_only) {
        exact_fields(request.payload, {
            "protocol", "operation", "state_root", "operation_id", "expected_journal_digest",
            "profile_id", "tops_id", "authorization_decision", "journal_client"
        }, "lifecycle apply state command");
    } else {
        exact_fields(request.payload, {
            "protocol", "operation", "state_root", "operation_id", "expected_journal_digest",
            "expected_applied_state_digest", "source_report_journal_digest", "profile_id", "tops_id",
            "profile_digest", "stable_inventory_digest", "desired_state", "observation",
            "prior_applied_state_digest", "action_id", "available_artifact_digests", "outcome",
            "blocker_class", "execution_evidence_digest", "authorization_decision", "planner_client",
            "journal_client"
        }, "lifecycle apply mutation command");
    }
    if (request.payload.at("protocol") != apply_command_protocol || request.payload.at("operation") != request.operation ||
        !safe_absolute_path(text(request.payload, "state_root", engine::Limits::max_path_bytes)) ||
        !safe_token(text(request.payload, "profile_id", 256U)) || !safe_token(text(request.payload, "tops_id", 256U))) {
        throw engine::Error("lifecycle_apply.command_invalid", "lifecycle apply command identity is invalid", 4);
    }
    const bool status = request.operation == "lifecycle_apply_status";
    if (status) {
        if (!request.payload.at("operation_id").is_null() || !request.payload.at("expected_journal_digest").is_null()) {
            throw engine::Error("lifecycle_apply.command_invalid", "apply status mutation fields must be null", 4);
        }
    } else {
        if (!request.payload.at("operation_id").is_string() ||
            !safe_token(request.payload.at("operation_id").get<std::string>()) ||
            !request.payload.at("expected_journal_digest").is_string()) {
            throw engine::Error("lifecycle_apply.command_invalid", "apply mutation identity is invalid", 4);
        }
    }
    std::string evidence = status ? "status" : request.payload.at("expected_journal_digest").get<std::string>();
    std::string permission = status ? "apply.status" : "apply.recover";
    if (!state_only) {
        for (const auto* field : {"profile_digest", "stable_inventory_digest", "source_report_journal_digest"}) {
            if (!tagged_digest(text(request.payload, field, 71U))) {
                throw engine::Error("lifecycle_apply.command_invalid", std::string(field) + " is invalid", 4);
            }
        }
        if (lifecycle_stable_inventory_digest(request.payload.at("observation"), request.deadline_unix_ms) !=
            request.payload.at("stable_inventory_digest").get<std::string>()) {
            throw engine::Error("lifecycle_apply.inventory_digest_mismatch", "apply inventory evidence is inconsistent", 4);
        }
        const bool close = request.operation == "lifecycle_apply_close";
        if (!request.payload.at("expected_applied_state_digest").is_string() ||
            (request.payload.at("expected_applied_state_digest") != "absent" &&
             !tagged_digest(request.payload.at("expected_applied_state_digest").get<std::string>())) ||
            (close ? !request.payload.at("action_id").is_null() :
                     !safe_token(text(request.payload, "action_id", 256U))) ||
            !request.payload.at("available_artifact_digests").is_array()) {
            throw engine::Error("lifecycle_apply.command_invalid", "apply action evidence is invalid", 4);
        }
        const auto expected_applied = request.payload.at("expected_applied_state_digest").get<std::string>();
        const auto& prior_applied = request.payload.at("prior_applied_state_digest");
        if ((expected_applied == "absent" && !prior_applied.is_null()) ||
            (expected_applied != "absent" &&
             (!prior_applied.is_string() || prior_applied.get<std::string>() != expected_applied))) {
            throw engine::Error(
                "lifecycle_apply.command_invalid",
                "prior applied-state evidence must equal the expected applied-state compare-and-swap value", 4);
        }
        const auto phase = request.operation == "lifecycle_apply_prepare" ? "prepare" :
            (close ? "close" : "finalize");
        permission = "apply." + std::string(phase);
        evidence = request.payload.at("profile_digest").get<std::string>() + "\n" +
            request.payload.at("desired_state").at("desired_state_digest").get<std::string>() + "\n" +
            request.payload.at("stable_inventory_digest").get<std::string>() + "\n" +
            request.payload.at("source_report_journal_digest").get<std::string>() + "\n" +
            request.payload.at("expected_journal_digest").get<std::string>() + "\n" +
            (close ? "converged" : request.payload.at("action_id").get<std::string>());
        if (request.operation == "lifecycle_apply_prepare" || close) {
            if (!request.payload.at("outcome").is_null() || !request.payload.at("blocker_class").is_null() ||
                !request.payload.at("execution_evidence_digest").is_null()) {
                throw engine::Error("lifecycle_apply.command_invalid", "prepare or close carries completion evidence", 4);
            }
        } else {
            if (!request.payload.at("outcome").is_string() ||
                (request.payload.at("outcome") != "committed" && request.payload.at("outcome") != "already_applied" &&
                 request.payload.at("outcome") != "blocked" && request.payload.at("outcome") != "failed") ||
                !request.payload.at("execution_evidence_digest").is_string() ||
                !tagged_digest(request.payload.at("execution_evidence_digest").get<std::string>())) {
                throw engine::Error("lifecycle_apply.command_invalid", "finalize completion evidence is invalid", 4);
            }
            static const std::set<std::string> blocker_classes = {
                "dependency_wait", "observation_retryable", "compatibility_blocked", "authorization_denied",
                "integrity_fatal", "critical_state_unknown", "cycle_detected"
            };
            const bool unsuccessful = request.payload.at("outcome") == "blocked" || request.payload.at("outcome") == "failed";
            const auto& blocker = request.payload.at("blocker_class");
            if ((unsuccessful && (!blocker.is_string() || !blocker_classes.contains(blocker.get<std::string>()))) ||
                (!unsuccessful && !blocker.is_null())) {
                throw engine::Error("lifecycle_apply.command_invalid", "finalize blocker evidence is inconsistent", 4);
            }
        }
    }
    if (request.operation == "lifecycle_apply_recover") {
        const auto expected = request.payload.at("expected_journal_digest").get<std::string>();
        if (expected != "discover" && !tagged_digest(expected)) {
            throw engine::Error("lifecycle_apply.command_invalid", "apply recovery expected state is invalid", 4);
        }
    }
    const auto tops = request.payload.at("tops_id").get<std::string>();
    const auto profile = request.payload.at("profile_id").get<std::string>();
    static_cast<void>(validate_ssiag_authorization(
        request.payload.at("authorization_decision"), "symphony.knowledge.lifecycle." + permission,
        tops, lifecycle_resource(tops, profile, evidence)));
    return apply_compatibility_result(request.payload.at("journal_client"), nullptr);
}

State recover_apply_state(const engine::Request& request, int directory,
                          std::vector<std::string>* actions, bool* changed) {
    const auto head = read_apply_head(directory, true);
    auto zero = read_apply_candidate(directory, 0);
    auto one = read_apply_candidate(directory, 1);
    if (zero.incompatible || one.incompatible) {
        throw engine::Error("lifecycle_apply.compatibility_required", "recovery found incompatible apply state", 4);
    }
    std::vector<Candidate*> valid;
    if (zero.valid) valid.push_back(&zero);
    if (one.valid) valid.push_back(&one);
    if (valid.empty()) {
        if (!zero.exists && !one.exists && !head.has_value() && request.payload.at("expected_journal_digest") == "discover") {
            actions->push_back("no apply journal was discovered");
            return {};
        }
        throw engine::Error("lifecycle_apply.unrecoverable", "no valid apply journal slot remains", 5);
    }
    const auto profile_id = request.payload.at("profile_id");
    const auto tops_id = request.payload.at("tops_id");
    for (const auto* candidate : valid) {
        if (candidate->journal.at("profile_id") != profile_id || candidate->journal.at("tops_id") != tops_id) {
            throw engine::Error("lifecycle_apply.unrecoverable", "apply slot belongs to another lifecycle stream", 5);
        }
    }
    Candidate* selected = valid.front();
    if (valid.size() == 2U) {
        auto* lower = valid[0];
        auto* higher = valid[1];
        if (number(lower->journal, "generation") == number(higher->journal, "generation")) {
            if (lower->journal.at("journal_digest") != higher->journal.at("journal_digest")) {
                throw engine::Error("lifecycle_apply.recovery_ambiguous", "apply journals diverge at one generation", 5);
            }
        } else {
            if (number(lower->journal, "generation") > number(higher->journal, "generation")) std::swap(lower, higher);
            if (number(higher->journal, "generation") != number(lower->journal, "generation") + 1U ||
                higher->journal.at("previous_journal_digest") != lower->journal.at("journal_digest")) {
                throw engine::Error("lifecycle_apply.recovery_ambiguous", "apply slots do not form one linked chain", 5);
            }
            selected = higher;
        }
    }
    const auto expected = request.payload.at("expected_journal_digest").get<std::string>();
    if (expected != "discover" && selected->journal.at("journal_digest") != expected) {
        throw engine::Error("lifecycle_apply.expected_state_mismatch", "apply recovery expected digest is stale", 4);
    }
    State baseline{head.value_or(engine::Json{}), selected->journal, true};
    const bool healthy = head.has_value() && head->at("journal_digest") == selected->journal.at("journal_digest") &&
        head->at("generation") == selected->journal.at("generation") &&
        head->at("profile_id") == profile_id && head->at("tops_id") == tops_id;
    if (healthy) {
        actions->push_back("apply head and active journal are already healthy");
        return baseline;
    }
    auto next = baseline.journal;
    const auto observed = utc_now();
    next["generation"] = number(next, "generation") + 1U;
    next["previous_journal_digest"] = baseline.journal.at("journal_digest");
    next["operation_id"] = request.payload.at("operation_id");
    next["updated_at"] = observed;
    const auto previous = next.at("checkpoints").back().at("checkpoint_digest");
    next["checkpoints"].push_back(checkpoint(
        next.at("checkpoints").size() + 1U, "recover", number(next, "current_plan_revision"),
        text(request.payload, "operation_id", 256U), observed,
        text(next, "current_observation_digest", 71U),
        next.at("checkpoints").back().at("ready_set_digest").get<std::string>(), previous));
    next["recovery"] = engine::Json{
        {"state", "recovered"}, {"disposition", "repaired_head"},
        {"recovered_from_digest", baseline.journal.at("journal_digest")},
        {"detail", "recovered one unique digest-linked apply state"},
    };
    *changed = true;
    actions->push_back("committed a forward apply recovery checkpoint");
    return commit_apply(directory, std::move(next), head);
}

engine::Json handle_lifecycle_apply(const engine::Request& request) {
    auto compatibility = validate_apply_command(request);
    const bool status = request.operation == "lifecycle_apply_status";
    const bool recover = request.operation == "lifecycle_apply_recover";
    auto stream = open_stream(
        request.payload.at("state_root").get<std::string>(), request.payload.at("tops_id").get<std::string>(),
        request.payload.at("profile_id").get<std::string>(), !status, !status, "v2");
    if (!stream.has_value()) {
        return make_apply_result(request.operation, std::move(compatibility), {}, nullptr, nullptr, nullptr,
                                 false, false, {}, true);
    }
    if (recover) {
        if (compatibility.at("mode") != "full") {
            throw engine::Error("lifecycle_apply.compatibility_required", "apply recovery requires full v2 overlap", 4);
        }
        std::vector<std::string> actions;
        bool changed = false;
        auto recovered_state = recover_apply_state(request, stream->directory_fd(), &actions, &changed);
        auto applied = read_applied_state(stream->directory_fd(), recovered_state);
        compatibility = apply_compatibility_result(
            request.payload.at("journal_client"), recovered_state.present ? &recovered_state.journal : nullptr);
        return make_apply_result(request.operation, std::move(compatibility), recovered_state, nullptr,
                                 recovered_state.present ? recovered_state.journal.at("active_action") : engine::Json(nullptr),
                                 applied.value_or(engine::Json(nullptr)), changed, changed, actions, false);
    }
    auto current = load_apply_state(stream->directory_fd());
    compatibility = apply_compatibility_result(request.payload.at("journal_client"), current.present ? &current.journal : nullptr);
    auto applied = read_applied_state(stream->directory_fd(), current);
    if (status) {
        return make_apply_result(request.operation, std::move(compatibility), current, nullptr,
                                 current.present ? current.journal.at("active_action") : engine::Json(nullptr),
                                 applied.value_or(engine::Json(nullptr)), false, false, {}, true);
    }
    if (compatibility.at("mode") != "full") {
        throw engine::Error("lifecycle_apply.compatibility_required", "apply mutation requires full v2 overlap", 4);
    }
    require_apply_expected(current, request.payload.at("expected_journal_digest").get<std::string>());
    const auto expected_applied = request.payload.at("expected_applied_state_digest").get<std::string>();
    if ((expected_applied == "absent") != !applied.has_value() ||
        (applied.has_value() && expected_applied != applied->at("applied_state_digest").get<std::string>())) {
        throw engine::Error("lifecycle_apply.applied_state_mismatch", "applied-state compare-and-swap failed", 4);
    }
    if (request.operation == "lifecycle_apply_prepare" || request.operation == "lifecycle_apply_close") {
        auto source_stream = open_stream(
            request.payload.at("state_root").get<std::string>(), request.payload.at("tops_id").get<std::string>(),
            request.payload.at("profile_id").get<std::string>(), false, false, "v1");
        if (!source_stream.has_value()) {
            throw engine::Error("lifecycle_apply.source_journal_missing", "report-only source journal is absent", 4);
        }
        const auto source = load_state(source_stream->directory_fd());
        if (!source.present || source.journal.at("journal_digest") != request.payload.at("source_report_journal_digest") ||
            source.journal.at("mode") != "apply-compatible" ||
            source.journal.at("profile_digest") != request.payload.at("profile_digest") ||
            source.journal.at("desired_state_digest") != request.payload.at("desired_state").at("desired_state_digest")) {
            throw engine::Error("lifecycle_apply.source_journal_mismatch", "report-only source journal does not authorize this evidence", 4);
        }
    }
    if (request.operation == "lifecycle_apply_finalize") {
        if (!current.present || current.journal.at("state") != "acting") {
            throw engine::Error("lifecycle_apply.action_mismatch", "no prepared lifecycle action exists", 4);
        }
        if (current.journal.at("profile_digest") != request.payload.at("profile_digest") ||
            current.journal.at("desired_state_digest") != request.payload.at("desired_state").at("desired_state_digest") ||
            current.journal.at("source_report_journal_digest") != request.payload.at("source_report_journal_digest") ||
            current.journal.at("prior_applied_state_digest") != request.payload.at("prior_applied_state_digest")) {
            throw engine::Error(
                "lifecycle_apply.prepared_evidence_mismatch",
                "finalize evidence does not belong to the prepared lifecycle action", 4);
        }
    }
    if (request.operation == "lifecycle_apply_close") {
        if (current.present && current.journal.at("state") == "acting") {
            throw engine::Error("lifecycle_apply.action_in_progress", "an exact lifecycle action must be finalized before close", 4);
        }
        auto plan = build_apply_plan(request, current);
        if (!plan_is_converged(plan)) {
            throw engine::Error("lifecycle_apply.not_converged", "verified observation still requires lifecycle work", 4);
        }
        if (current.present && current.journal.at("state") == "closed" && applied.has_value() &&
            current.journal.at("profile_digest") == request.payload.at("profile_digest") &&
            current.journal.at("desired_state_digest") == plan.at("desired_state_digest") &&
            current.journal.at("current_stable_inventory_digest") == request.payload.at("stable_inventory_digest") &&
            current.journal.at("source_report_journal_digest") == request.payload.at("source_report_journal_digest")) {
            return make_apply_result(request.operation, std::move(compatibility), current, std::move(plan), nullptr,
                                     *applied, false, false, {"converged applied evidence is already committed"}, false);
        }
        const auto observed = utc_now();
        const bool continuing = current.present && current.journal.at("state") != "closed";
        engine::Json checkpoints = continuing ? current.journal.at("checkpoints") : engine::Json::array();
        const auto previous_checkpoint = checkpoints.empty() ? engine::Json(nullptr) :
            checkpoints.back().at("checkpoint_digest");
        checkpoints.push_back(checkpoint(
            checkpoints.size() + 1U, "close", number(plan, "revision"),
            text(request.payload, "operation_id", 256U), observed,
            text(request.payload.at("observation"), "observation_digest", 71U),
            engine::tagged_sha256(ready_set(plan).dump()), previous_checkpoint));
        const auto transaction = continuing ? current.journal.at("transaction_id") : plan.at("transaction_id");
        engine::Json next{
            {"protocol", apply_journal_protocol}, {"format_version", apply_format_version},
            {"journal_id", continuing ? current.journal.at("journal_id") : engine::Json(
                "lifecycle-apply:" + engine::sha256_hex(text(request.payload, "tops_id", 256U) + "|" +
                    text(request.payload, "profile_id", 256U) + "|" + transaction.get<std::string>()))},
            {"transaction_id", transaction}, {"operation_id", request.payload.at("operation_id")},
            {"generation", current.present ? number(current.journal, "generation") + 1U : 1U},
            {"previous_journal_digest", current.present ? current.journal.at("journal_digest") : engine::Json(nullptr)},
            {"source_report_journal_digest", request.payload.at("source_report_journal_digest")},
            {"profile_id", request.payload.at("profile_id")}, {"profile_digest", request.payload.at("profile_digest")},
            {"tops_id", request.payload.at("tops_id")}, {"mode", "apply-compatible"}, {"state", "closed"},
            {"desired_state_digest", plan.at("desired_state_digest")}, {"observation_key", plan.at("observation_key")},
            {"current_observation_digest", plan.at("observation_digest")},
            {"current_stable_inventory_digest", request.payload.at("stable_inventory_digest")},
            {"prior_applied_state_digest", plan.at("prior_applied_state_digest")},
            {"current_plan_digest", plan.at("plan_digest")}, {"current_plan_revision", plan.at("revision")},
            {"replan_count", continuing ? number(current.journal, "replan_count") + 1U : 0U},
            {"active_action", nullptr},
            {"action_attempts", continuing ? current.journal.at("action_attempts") : engine::Json::array()},
            {"blockers", engine::Json::array()}, {"checkpoints", std::move(checkpoints)},
            {"compatibility", apply_journal_compatibility()},
            {"extensions", current.present ? current.journal.at("extensions") : engine::Json::array()},
            {"recovery", engine::Json{{"state", "clean"}, {"disposition", "not_applicable"},
                {"recovered_from_digest", nullptr}, {"detail", "no apply recovery has been required"}}},
            {"applied_state_digest", nullptr}, {"started_at", continuing ? current.journal.at("started_at") : engine::Json(observed)},
            {"updated_at", observed}, {"closed_at", observed}, {"canonical", false}, {"apply_authorized", true},
        };
        auto applied_value = build_applied_state(
            request.payload.at("desired_state"), request.payload.at("observation"), plan, next, applied);
        write_applied_state(stream->directory_fd(), applied_value);
        next["applied_state_digest"] = applied_value.at("applied_state_digest");
        auto committed = commit_apply(
            stream->directory_fd(), std::move(next),
            current.present ? std::optional<engine::Json>(current.head) : std::nullopt);
        return make_apply_result(request.operation, std::move(compatibility), committed, std::move(plan), nullptr,
                                 std::move(applied_value), true, false,
                                 {"verified convergence and committed content-addressed applied evidence"}, false);
    }
    if (request.operation == "lifecycle_apply_prepare") {
        if (current.present && current.journal.at("state") == "acting") {
            if (current.journal.at("active_action").at("action_id") == request.payload.at("action_id") &&
                current.journal.at("operation_id") == request.payload.at("operation_id")) {
                return make_apply_result(request.operation, std::move(compatibility), current, nullptr,
                                         current.journal.at("active_action"), applied.value_or(engine::Json(nullptr)),
                                         false, false, {"replayed prepared action attempt"}, false);
            }
            throw engine::Error("lifecycle_apply.action_in_progress", "another lifecycle action attempt is active", 4);
        }
        auto plan = build_apply_plan(request, current);
        const auto wanted_id = request.payload.at("action_id").get<std::string>();
        const auto* selected = find_plan_action(plan, wanted_id);
        if (selected == nullptr || !apply_action_ready(*selected, request.payload.at("available_artifact_digests"))) {
            throw engine::Error("lifecycle_apply.action_not_ready", "requested action is not ready under verified evidence", 4);
        }
        if (selected->at("kind") == "preserve" || selected->at("kind") == "report" || selected->at("kind") == "verify") {
            throw engine::Error("lifecycle_apply.non_mutating_action", "non-mutating report evidence is not an executable action", 4);
        }
        const auto observed = utc_now();
        const bool continuing = current.present && current.journal.at("state") != "closed";
        engine::Json attempts = continuing ? current.journal.at("action_attempts") : engine::Json::array();
        attempts.push_back(apply_attempt(*selected, attempts, number(plan, "revision"), observed));
        engine::Json checkpoints = continuing ? current.journal.at("checkpoints") : engine::Json::array();
        const auto previous = checkpoints.empty() ? engine::Json(nullptr) : checkpoints.back().at("checkpoint_digest");
        const auto ready_digest = engine::tagged_sha256(ready_set(plan).dump());
        checkpoints.push_back(checkpoint(
            checkpoints.size() + 1U, "attempt", number(plan, "revision"),
            text(request.payload, "operation_id", 256U), observed,
            text(request.payload.at("observation"), "observation_digest", 71U), ready_digest, previous));
        const auto generation = current.present ? number(current.journal, "generation") + 1U : 1U;
        engine::Json next{
            {"protocol", apply_journal_protocol}, {"format_version", apply_format_version},
            {"journal_id", continuing ? current.journal.at("journal_id") : engine::Json(
                "lifecycle-apply:" + engine::sha256_hex(text(request.payload, "tops_id", 256U) + "|" +
                    text(request.payload, "profile_id", 256U) + "|" + plan.at("transaction_id").get<std::string>()))},
            {"transaction_id", continuing ? current.journal.at("transaction_id") : plan.at("transaction_id")},
            {"operation_id", request.payload.at("operation_id")}, {"generation", generation},
            {"previous_journal_digest", current.present ? current.journal.at("journal_digest") : engine::Json(nullptr)},
            {"source_report_journal_digest", request.payload.at("source_report_journal_digest")},
            {"profile_id", request.payload.at("profile_id")}, {"profile_digest", request.payload.at("profile_digest")},
            {"tops_id", request.payload.at("tops_id")}, {"mode", "apply-compatible"}, {"state", "acting"},
            {"desired_state_digest", plan.at("desired_state_digest")},
            {"observation_key", continuing ? current.journal.at("observation_key") : plan.at("observation_key")},
            {"current_observation_digest", plan.at("observation_digest")},
            {"current_stable_inventory_digest", request.payload.at("stable_inventory_digest")},
            {"prior_applied_state_digest", plan.at("prior_applied_state_digest")},
            {"current_plan_digest", plan.at("plan_digest")}, {"current_plan_revision", plan.at("revision")},
            {"replan_count", continuing ? number(current.journal, "replan_count") + 1U : 0U},
            {"active_action", *selected}, {"action_attempts", std::move(attempts)},
            {"blockers", journal_blockers(plan)}, {"checkpoints", std::move(checkpoints)},
            {"compatibility", apply_journal_compatibility()},
            {"extensions", current.present ? current.journal.at("extensions") : engine::Json::array()},
            {"recovery", engine::Json{{"state", "clean"}, {"disposition", "not_applicable"},
                {"recovered_from_digest", nullptr}, {"detail", "no apply recovery has been required"}}},
            {"applied_state_digest", applied.has_value() ? applied->at("applied_state_digest") : engine::Json(nullptr)},
            {"started_at", continuing ? current.journal.at("started_at") : engine::Json(observed)},
            {"updated_at", observed}, {"closed_at", nullptr}, {"canonical", false}, {"apply_authorized", true},
        };
        auto committed = commit_apply(stream->directory_fd(), std::move(next),
                                      current.present ? std::optional<engine::Json>(current.head) : std::nullopt);
        return make_apply_result(request.operation, std::move(compatibility), committed, std::move(plan),
                                 committed.journal.at("active_action"), applied.value_or(engine::Json(nullptr)),
                                 true, false, {"durably prepared one exact lifecycle action"}, false);
    }
    if (!current.present || current.journal.at("state") != "acting" ||
        current.journal.at("active_action").at("action_id") != request.payload.at("action_id")) {
        throw engine::Error("lifecycle_apply.action_mismatch", "no matching prepared lifecycle action exists", 4);
    }
    auto plan = build_apply_plan(request, current);
    const auto& active = current.journal.at("active_action");
    const auto outcome = request.payload.at("outcome").get<std::string>();
    const bool success = outcome == "committed" || outcome == "already_applied";
    if (success) {
        if (!apply_action_observed(
                active, request.payload.at("desired_state"), request.payload.at("observation"))) {
            throw engine::Error(
                "lifecycle_apply.verification_failed",
                "post-action observation does not prove the prepared lifecycle transition", 4);
        }
        for (const auto& action : plan.at("actions")) {
            if (action.at("component_id") == active.at("component_id") && action.at("kind") == active.at("kind") &&
                action.at("disposition") != "completed" && action.at("disposition") != "skipped") {
                throw engine::Error("lifecycle_apply.verification_failed", "post-action observation does not prove the requested transition", 4);
            }
        }
    }
    auto next = current.journal;
    const auto observed = utc_now();
    next["generation"] = number(next, "generation") + 1U;
    next["previous_journal_digest"] = current.journal.at("journal_digest");
    next["operation_id"] = request.payload.at("operation_id");
    next["source_report_journal_digest"] = request.payload.at("source_report_journal_digest");
    next["current_observation_digest"] = plan.at("observation_digest");
    next["current_stable_inventory_digest"] = request.payload.at("stable_inventory_digest");
    next["current_plan_digest"] = plan.at("plan_digest");
    next["current_plan_revision"] = plan.at("revision");
    next["replan_count"] = number(next, "replan_count") + 1U;
    next["active_action"] = nullptr;
    auto& attempt = next.at("action_attempts").back();
    attempt["state"] = outcome;
    attempt["blocker_class"] = request.payload.at("blocker_class");
    attempt["observed_after_digest"] = request.payload.at("observation").at("observation_digest");
    attempt["evidence_digest"] = request.payload.at("execution_evidence_digest");
    attempt["completed_at"] = observed;
    next["blockers"] = journal_blockers(plan);
    const auto previous = next.at("checkpoints").back().at("checkpoint_digest");
    next["checkpoints"].push_back(checkpoint(
        next.at("checkpoints").size() + 1U, "verify", number(plan, "revision"),
        text(request.payload, "operation_id", 256U), observed,
        text(request.payload.at("observation"), "observation_digest", 71U),
        engine::tagged_sha256(ready_set(plan).dump()), previous));
    next["updated_at"] = observed;
    engine::Json new_applied = nullptr;
    if (success && plan_is_converged(plan)) {
        auto applied_value = build_applied_state(
            request.payload.at("desired_state"), request.payload.at("observation"), plan, next, applied);
        write_applied_state(stream->directory_fd(), applied_value);
        next["state"] = "closed";
        next["closed_at"] = observed;
        next["applied_state_digest"] = applied_value.at("applied_state_digest");
        new_applied = std::move(applied_value);
    } else {
        next["state"] = success ? (next.at("blockers").empty() ? "open" : "blocked") : "blocked";
        next["closed_at"] = nullptr;
        next["applied_state_digest"] = applied.has_value() ? applied->at("applied_state_digest") : engine::Json(nullptr);
    }
    auto committed = commit_apply(stream->directory_fd(), std::move(next), current.head);
    return make_apply_result(request.operation, std::move(compatibility), committed, std::move(plan), nullptr,
                             new_applied.is_null() ? applied.value_or(engine::Json(nullptr)) : std::move(new_applied),
                             true, false, {success ? "verified and committed lifecycle action evidence" :
                                                   "recorded blocked lifecycle action evidence"}, false);
}

engine::Json validate_command(const engine::Request& request) {
    static const std::set<std::string> operations = {
        "lifecycle_boot", "lifecycle_boot_status", "lifecycle_boot_recover"
    };
    if (!operations.contains(request.operation)) {
        throw engine::Error("lifecycle_journal.command_invalid", "unsupported lifecycle journal operation", 4);
    }
    if (request.operation == "lifecycle_boot") {
        exact_fields(request.payload, {
            "protocol", "operation", "state_root", "operation_id", "expected_journal_digest",
            "profile_id", "tops_id", "profile_digest", "stable_inventory_digest", "mode",
            "desired_state", "observation", "prior_applied_state_digest", "authorization_decision",
            "planner_client", "journal_client"
        }, "lifecycle boot command");
    } else {
        exact_fields(request.payload, {
            "protocol", "operation", "state_root", "operation_id", "expected_journal_digest",
            "profile_id", "tops_id", "authorization_decision", "journal_client"
        }, "lifecycle boot state command");
    }
    if (text(request.payload, "protocol") != command_protocol ||
        text(request.payload, "operation") != request.operation ||
        !safe_absolute_path(text(request.payload, "state_root", engine::Limits::max_path_bytes)) ||
        !safe_token(text(request.payload, "profile_id", 256U)) ||
        !safe_token(text(request.payload, "tops_id", 256U))) {
        throw engine::Error("lifecycle_journal.command_invalid", "lifecycle command identity is invalid", 4);
    }
    const bool status = request.operation == "lifecycle_boot_status";
    if (status) {
        if (!request.payload.at("operation_id").is_null() ||
            !request.payload.at("expected_journal_digest").is_null()) {
            throw engine::Error("lifecycle_journal.command_invalid", "status mutation fields must be null", 4);
        }
    } else {
        if (!request.payload.at("operation_id").is_string() ||
            !safe_token(request.payload.at("operation_id").get<std::string>()) ||
            !request.payload.at("expected_journal_digest").is_string()) {
            throw engine::Error("lifecycle_journal.command_invalid", "lifecycle mutation identity is invalid", 4);
        }
        const auto expected = request.payload.at("expected_journal_digest").get<std::string>();
        if (request.operation == "lifecycle_boot_recover") {
            if (expected != "discover" && !tagged_digest(expected)) {
                throw engine::Error("lifecycle_journal.command_invalid", "recovery expected state is invalid", 4);
            }
        } else if (expected != "absent" && !tagged_digest(expected)) {
            throw engine::Error("lifecycle_journal.command_invalid", "boot expected state is invalid", 4);
        }
    }
    std::string evidence = status ? "status" :
        request.payload.at("expected_journal_digest").get<std::string>();
    if (request.operation == "lifecycle_boot") {
        if (!tagged_digest(text(request.payload, "profile_digest", 71U)) ||
            !tagged_digest(text(request.payload, "stable_inventory_digest", 71U)) ||
            (request.payload.at("mode") != "report" && request.payload.at("mode") != "apply-compatible")) {
            throw engine::Error("lifecycle_journal.command_invalid", "boot profile evidence is invalid", 4);
        }
        if (lifecycle_stable_inventory_digest(
                request.payload.at("observation"), request.deadline_unix_ms) !=
            request.payload.at("stable_inventory_digest").get<std::string>()) {
            throw engine::Error(
                "lifecycle_journal.inventory_digest_mismatch",
                "boot authorization evidence does not match the stable lifecycle inventory",
                4);
        }
        evidence = request.payload.at("profile_digest").get<std::string>() + "\n" +
            request.payload.at("desired_state").at("desired_state_digest").get<std::string>() + "\n" +
            request.payload.at("mode").get<std::string>() + "\n" +
            request.payload.at("stable_inventory_digest").get<std::string>();
    }
    const auto tops_id = request.payload.at("tops_id").get<std::string>();
    const auto profile_id = request.payload.at("profile_id").get<std::string>();
    static_cast<void>(validate_ssiag_authorization(
        request.payload.at("authorization_decision"),
        "symphony.knowledge.lifecycle." +
            std::string(request.operation == "lifecycle_boot" ? "boot" :
                        (status ? "boot.status" : "boot.recover")),
        tops_id, lifecycle_resource(tops_id, profile_id, evidence)));
    return compatibility_result(request.payload.at("journal_client"), nullptr);
}

} // namespace

engine::Json lifecycle_journal_capabilities() {
    return engine::Json{
        {"command_protocol", command_protocol}, {"result_protocol", result_protocol},
        {"journal_protocol", journal_protocol}, {"head_protocol", head_protocol},
        {"journal_read_versions", engine::Json::array({format_version, apply_format_version})},
        {"journal_write_versions", engine::Json::array({format_version, apply_format_version})},
        {"required_capabilities", required_capabilities}, {"optional_capabilities", optional_capabilities},
        {"two_way_procedural_compatibility", true}, {"dynamic_replanning", true},
        {"apply_command_protocol", apply_command_protocol}, {"apply_result_protocol", apply_result_protocol},
        {"apply_journal_protocol", apply_journal_protocol}, {"apply_head_protocol", apply_head_protocol},
        {"apply_required_capabilities", apply_required_capabilities},
        {"persistence_enabled", true}, {"action_execution_enabled", false},
        {"external_action_coordination_enabled", true}, {"host_action_execution_enabled", false},
        {"applied_state_persistence_enabled", true}, {"apply_authorized", false},
    };
}

engine::Json handle_lifecycle_journal(const engine::Request& request) {
    auto compatibility = validate_command(request);
    const bool status = request.operation == "lifecycle_boot_status";
    const bool recover = request.operation == "lifecycle_boot_recover";
    auto stream = open_stream(
        request.payload.at("state_root").get<std::string>(),
        request.payload.at("tops_id").get<std::string>(),
        request.payload.at("profile_id").get<std::string>(),
        !status, !status);
    if (!stream.has_value()) {
        return make_result(request.operation, std::move(compatibility), {}, nullptr, false, false, {}, true);
    }
    if (recover) {
        if (compatibility.at("mode") != "full") {
            throw engine::Error("lifecycle_journal.compatibility_required", "recovery requires full v1 capability overlap", 4);
        }
        std::vector<std::string> actions;
        bool changed = false;
        auto recovered = recover_state(request, stream->directory_fd(), &actions, &changed);
        compatibility = compatibility_result(request.payload.at("journal_client"),
                                             recovered.present ? &recovered.journal : nullptr);
        return make_result(request.operation, std::move(compatibility), recovered, nullptr,
                           changed, changed, actions, false);
    }
    auto current = load_state(stream->directory_fd());
    compatibility = compatibility_result(request.payload.at("journal_client"),
                                         current.present ? &current.journal : nullptr);
    if (status) {
        return make_result(request.operation, std::move(compatibility), current, nullptr,
                           false, false, {}, true);
    }
    if (compatibility.at("mode") != "full") {
        throw engine::Error("lifecycle_journal.compatibility_required", "boot mutation requires full v1 capability overlap", 4);
    }
    const auto expected = request.payload.at("expected_journal_digest").get<std::string>();
    if (current.present && current.journal.at("operation_id") == request.payload.at("operation_id")) {
        if (current.journal.at("profile_digest") == request.payload.at("profile_digest") &&
            current.journal.at("mode") == request.payload.at("mode") &&
            current.journal.at("desired_state_digest") == request.payload.at("desired_state").at("desired_state_digest") &&
            current.journal.at("current_observation_digest") == request.payload.at("observation").at("observation_digest") &&
            current.journal.at("current_stable_inventory_digest") == request.payload.at("stable_inventory_digest") &&
            current.journal.at("prior_applied_state_digest") == request.payload.at("prior_applied_state_digest")) {
            return make_result(request.operation, std::move(compatibility), current, nullptr,
                               false, false, {"replayed lifecycle boot operation already committed"}, false);
        }
        throw engine::Error(
            "lifecycle_journal.operation_conflict",
            "operation ID was reused with different lifecycle evidence", 4);
    }
    require_expected(current, expected, !current.present);
    if (current.present && current.journal.at("state") != "closed" &&
        current.journal.at("profile_digest") == request.payload.at("profile_digest") &&
        current.journal.at("mode") == request.payload.at("mode") &&
        current.journal.at("desired_state_digest") == request.payload.at("desired_state").at("desired_state_digest") &&
        current.journal.at("current_stable_inventory_digest") == request.payload.at("stable_inventory_digest") &&
        current.journal.at("prior_applied_state_digest") == request.payload.at("prior_applied_state_digest")) {
        return make_result(request.operation, std::move(compatibility), current, nullptr,
                           false, false,
                           {"stable lifecycle inventory and desired state are already durably recorded"}, false);
    }
    engine::Json plan_payload{
        {"protocol", "symphony.knowledge.lifecycle-plan-command.v1"}, {"operation", "lifecycle_plan"},
        {"desired_state", request.payload.at("desired_state")},
        {"observation", request.payload.at("observation")},
        {"prior_applied_state_digest", request.payload.at("prior_applied_state_digest")},
        {"client", request.payload.at("planner_client")},
    };
    auto plan = build_lifecycle_plan(plan_payload, request.deadline_unix_ms);
    auto next = begin_or_replan(request, current, plan);
    auto committed = commit(
        stream->directory_fd(), std::move(next),
        current.present ? std::optional<engine::Json>(current.head) : std::nullopt);
    return make_result(request.operation, std::move(compatibility), committed, std::move(plan),
                       true, false, {current.present ? "committed linked lifecycle plan revision" :
                                                    "committed lifecycle boot transaction"}, false);
}

engine::Json handle_lifecycle_apply_request(const engine::Request& request) {
    return handle_lifecycle_apply(request);
}

} // namespace symphony::knowledge::session
