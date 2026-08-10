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
                                       const std::string& profile_id, bool exclusive, bool create) {
    auto opened_root = open_absolute_directory(root, create);
    if (!opened_root.has_value()) return std::nullopt;
    FileDescriptor current = std::move(*opened_root);
    const std::array<std::string, 8> components = {
        "symphony", "knowledge-session-coordinator", "lifecycle", "v1", "tops",
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
    candidate.journal = parse_file(*data, "lifecycle journal slot");
    if (!candidate.journal.is_object() || !candidate.journal.contains("protocol") ||
        !candidate.journal.contains("format_version") ||
        candidate.journal.at("protocol") != journal_protocol ||
        !candidate.journal.at("format_version").is_number_integer() ||
        candidate.journal.at("format_version") != format_version) {
        candidate.incompatible = true;
        return candidate;
    }
    try {
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
        {"journal_read_versions", engine::Json::array({format_version})},
        {"journal_write_versions", engine::Json::array({format_version})},
        {"required_capabilities", required_capabilities}, {"optional_capabilities", optional_capabilities},
        {"two_way_procedural_compatibility", true}, {"dynamic_replanning", true},
        {"persistence_enabled", true}, {"action_execution_enabled", false},
        {"applied_state_persistence_enabled", false}, {"apply_authorized", false},
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

} // namespace symphony::knowledge::session
