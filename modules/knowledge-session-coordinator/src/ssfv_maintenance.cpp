#include "ssfv_maintenance.hpp"

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
#include <ctime>
#include <unistd.h>
#include <utility>
#include <vector>

namespace symphony::knowledge::session {
namespace engine = symphony::knowledge::engine;
namespace fs = std::filesystem;

namespace {

constexpr const char* command_protocol = "symphony.knowledge.ssfv-maintenance-command.v1";
constexpr const char* journal_protocol = "symphony.knowledge.ssfv-maintenance-journal.v1";
constexpr const char* head_protocol = "symphony.knowledge.ssfv-maintenance-head.v1";
constexpr const char* result_protocol = "symphony.knowledge.ssfv-maintenance-result.v1";
constexpr std::uint64_t format_version = 1U;
constexpr std::size_t max_state_bytes = engine::Limits::max_response_bytes;
constexpr std::size_t max_checkpoints = 256U;

const std::vector<std::string> required_capabilities = {
    "atomic-head-v1", "content-addressed-ssfv-baseline-v1", "dual-slot-journal-v1",
    "expected-state-cas-v1", "idempotent-operation-v1",
    "maestro-inventory-lineage-v1", "opaque-extension-preservation-v1",
    "recovery-forward-v1", "ssiag-capability-binding-v1",
};

const std::vector<std::string> optional_capabilities = {
    "discovery-recovery-v1", "nonblocking-lock-v1",
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

struct Candidate final {
    int slot = -1;
    engine::Json journal;
    bool exists = false;
    bool valid = false;
    bool incompatible = false;
};

struct State final {
    engine::Json head;
    engine::Json journal;
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

bool valid_uuid(std::string_view value) {
    if (value.size() != 36U || value[8] != '-' || value[13] != '-' || value[18] != '-' ||
        value[23] != '-' || value[14] < '1' || value[14] > '8' ||
        (value[19] != '8' && value[19] != '9' && value[19] != 'a' && value[19] != 'b')) {
        return false;
    }
    for (std::size_t index = 0; index < value.size(); ++index) {
        if (index == 8U || index == 13U || index == 18U || index == 23U) continue;
        const char character = value[index];
        if (!((character >= '0' && character <= '9') ||
              (character >= 'a' && character <= 'f'))) return false;
    }
    return true;
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
        throw engine::Error("ssfv_maintenance.field_set", context + " has an invalid field set", 4);
    }
    for (const auto& [key, value] : object.items()) {
        static_cast<void>(value);
        if (!fields.contains(key)) {
            throw engine::Error("ssfv_maintenance.unknown_field", context + " has an unknown field", 4);
        }
    }
}

std::string text(const engine::Json& object, const char* field, std::size_t maximum = 4096U) {
    if (!object.contains(field) || !object.at(field).is_string()) {
        throw engine::Error("ssfv_maintenance.invalid_field", std::string(field) + " must be a string", 4);
    }
    const auto value = object.at(field).get<std::string>();
    if (value.empty() || value.size() > maximum) {
        throw engine::Error("ssfv_maintenance.invalid_field", std::string(field) + " has invalid length", 4);
    }
    for (const unsigned char character : value) {
        if (character < 0x20U || character == 0x7fU) {
            throw engine::Error("ssfv_maintenance.invalid_field", std::string(field) + " contains unsafe text", 4);
        }
    }
    return value;
}

std::uint64_t number(const engine::Json& object, const char* field) {
    if (!object.contains(field) ||
        (!object.at(field).is_number_unsigned() && !object.at(field).is_number_integer())) {
        throw engine::Error("ssfv_maintenance.invalid_field", std::string(field) + " must be an integer", 4);
    }
    try {
        const auto value = object.at(field).get<std::uint64_t>();
        if (value > 9007199254740991ULL) throw std::out_of_range("integer");
        return value;
    } catch (const std::exception&) {
        throw engine::Error("ssfv_maintenance.invalid_field", std::string(field) + " is out of range", 4);
    }
}

std::string utc_now() {
    const auto now = std::chrono::time_point_cast<std::chrono::seconds>(std::chrono::system_clock::now());
    const std::time_t value = std::chrono::system_clock::to_time_t(now);
    std::tm result {};
    if (::gmtime_r(&value, &result) == nullptr) {
        throw engine::Error("ssfv_maintenance.clock_failed", "could not obtain a UTC timestamp", 5);
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
        throw engine::Error("ssfv_maintenance.invalid_field", name + " must be a bounded array", 4);
    }
    std::vector<std::string> result;
    std::set<std::string> seen;
    for (const auto& item : value) {
        if (!item.is_string() || !safe_token(item.get<std::string>()) ||
            !seen.insert(item.get<std::string>()).second) {
            throw engine::Error("ssfv_maintenance.invalid_field", name + " contains an invalid token", 4);
        }
        result.push_back(item.get<std::string>());
    }
    return result;
}

std::vector<std::uint64_t> version_array(const engine::Json& value, const std::string& name) {
    if (!value.is_array() || value.empty() || value.size() > 16U) {
        throw engine::Error("ssfv_maintenance.invalid_field", name + " must be a bounded array", 4);
    }
    std::vector<std::uint64_t> result;
    std::set<std::uint64_t> seen;
    for (const auto& item : value) {
        if (!item.is_number_integer() && !item.is_number_unsigned()) {
            throw engine::Error("ssfv_maintenance.invalid_field", name + " contains a non-integer", 4);
        }
        const auto version = item.get<std::uint64_t>();
        if (version == 0U || version > 16U || !seen.insert(version).second) {
            throw engine::Error("ssfv_maintenance.invalid_field", name + " contains an invalid version", 4);
        }
        result.push_back(version);
    }
    return result;
}

engine::Json compatibility_result(const engine::Json& client, const engine::Json* journal) {
    exact_fields(client, {"client_id", "client_version", "process_protocols", "journal_read_versions",
                          "journal_write_versions", "capabilities"}, "SSFV maintenance client");
    if (text(client, "client_id", 128U) != "qxctl" || !safe_version(text(client, "client_version", 64U))) {
        throw engine::Error("ssfv_maintenance.client_invalid", "client identity is invalid", 4);
    }
    const auto protocols = string_array(client.at("process_protocols"), "process_protocols", 8U, true);
    const auto reads = version_array(client.at("journal_read_versions"), "journal_read_versions");
    const auto writes = version_array(client.at("journal_write_versions"), "journal_write_versions");
    const auto capabilities = string_array(client.at("capabilities"), "capabilities", 64U, true);
    std::vector<std::string> missing;
    for (const auto& required : required_capabilities) {
        if (std::find(capabilities.begin(), capabilities.end(), required) == capabilities.end()) {
            missing.push_back(required);
        }
    }
    const bool process_ok = std::find(protocols.begin(), protocols.end(), engine::process_protocol_v1) != protocols.end();
    const bool read_ok = std::find(reads.begin(), reads.end(), format_version) != reads.end();
    const bool write_ok = std::find(writes.begin(), writes.end(), format_version) != writes.end();
    const bool stored_ok = journal == nullptr || number(*journal, "format_version") == format_version;
    const bool full = process_ok && read_ok && write_ok && stored_ok && missing.empty();
    const bool readable = process_ok && read_ok && stored_ok;
    return engine::Json{
        {"mode", full ? "full" : (readable ? "read_only" : "blocked")},
        {"process_protocol", process_ok ? engine::Json(engine::process_protocol_v1) : engine::Json(nullptr)},
        {"journal_read_version", read_ok && stored_ok ? engine::Json(format_version) : engine::Json(nullptr)},
        {"journal_write_version", write_ok ? engine::Json(format_version) : engine::Json(nullptr)},
        {"missing_capabilities", missing}, {"two_way_procedural_compatibility", true},
        {"reason", full ? "client, coordinator, and stored SSFV maintenance journal share v1" :
            (readable ? "journal is readable but mutation capability is incomplete" :
                        "no safe SSFV maintenance read overlap")},
    };
}

engine::Json journal_compatibility() {
    return engine::Json{
        {"process_protocols", engine::Json::array({engine::process_protocol_v1})},
        {"journal_read_versions", engine::Json::array({format_version})},
        {"journal_write_version", format_version},
        {"minimum_reader_version", format_version},
        {"required_capabilities", required_capabilities},
        {"optional_capabilities", optional_capabilities},
        {"opaque_extensions_preserved", true},
    };
}

void validate_engine_evidence(const engine::Json& value) {
    exact_fields(value, {"module_id", "engine_id", "vector_id", "version", "receipt_digest",
                         "executable_digest", "evidence_digest"}, "SSFV engine evidence");
    if (text(value, "module_id") != "ssfv-engine" || text(value, "engine_id") != "symphony-ssfv" ||
        text(value, "vector_id") != "ssfv" || !safe_version(text(value, "version", 64U)) ||
        !tagged_digest(text(value, "receipt_digest", 71U)) ||
        !tagged_digest(text(value, "executable_digest", 71U))) {
        throw engine::Error("ssfv_maintenance.engine_invalid", "SSFV engine evidence is invalid", 4);
    }
    verify_digest(value, "evidence_digest", "ssfv_maintenance.engine_invalid");
}

void validate_snapshot(const engine::Json& value, const engine::Json& engine_evidence) {
    exact_fields(value, {"protocol", "module_id", "engine_id", "engine_version", "vector_id",
                         "contract_digest", "namespace_digest", "registry_digest", "source_digest",
                         "feature_files", "records", "snapshot_digest"}, "SSFV semantic snapshot");
    if (text(value, "protocol") != "symphony.ssfv.semantic-snapshot.v1" ||
        text(value, "module_id") != "ssfv-engine" || text(value, "engine_id") != "symphony-ssfv" ||
        text(value, "vector_id") != "ssfv" || text(value, "engine_version", 64U) != text(engine_evidence, "version", 64U) ||
        !tagged_digest(text(value, "contract_digest", 71U)) ||
        !tagged_digest(text(value, "namespace_digest", 71U)) ||
        !tagged_digest(text(value, "registry_digest", 71U)) ||
        !tagged_digest(text(value, "source_digest", 71U)) ||
        !value.at("feature_files").is_array() || value.at("feature_files").size() > 1024U ||
        !value.at("records").is_array() || value.at("records").size() > 8192U) {
        throw engine::Error("ssfv_maintenance.snapshot_invalid", "SSFV semantic snapshot is invalid", 4);
    }
    verify_digest(value, "snapshot_digest", "ssfv_maintenance.snapshot_invalid");
}

void validate_diff(const engine::Json& value, const std::string& baseline,
                   const std::string& current) {
    exact_fields(value, {"protocol", "baseline_digest", "current_digest", "state",
                         "added_feature_ids", "changed_feature_ids", "removed_feature_ids",
                         "uncovered_paths", "stale_references", "semantic_candidates", "summary",
                         "read_only", "noncanonical", "result_digest"}, "SSFV diff result");
    static const std::set<std::string> states = {
        "identical", "additive", "semantic_change", "removal", "review_required",
    };
    if (text(value, "protocol") != "symphony.ssfv.diff-result.v2" ||
        text(value, "baseline_digest", 71U) != baseline || text(value, "current_digest", 71U) != current ||
        !states.contains(text(value, "state", 32U)) || value.at("read_only") != true ||
        value.at("noncanonical") != true || !value.at("summary").is_object()) {
        throw engine::Error("ssfv_maintenance.diff_invalid", "SSFV diff result is invalid", 4);
    }
    for (const auto* field : {"added_feature_ids", "changed_feature_ids", "removed_feature_ids",
                              "uncovered_paths", "stale_references", "semantic_candidates"}) {
        if (!value.at(field).is_array() || value.at(field).size() > 8192U) {
            throw engine::Error("ssfv_maintenance.diff_invalid", "SSFV diff collection is invalid", 4);
        }
    }
    verify_digest(value, "result_digest", "ssfv_maintenance.diff_invalid");
}

void validate_maestro_evidence(const engine::Json& value, const std::string& tops_id) {
    exact_fields(value, {"availability", "reason", "observation", "evidence_digest"},
                 "Maestro inventory evidence");
    const auto availability = text(value, "availability", 32U);
    static_cast<void>(text(value, "reason", 4096U));
    if (availability == "not_configured") {
        if (!value.at("observation").is_null()) {
            throw engine::Error("ssfv_maintenance.inventory_invalid", "not-configured inventory carries an observation", 4);
        }
    } else if (availability == "observed") {
        if (!value.at("observation").is_object()) {
            throw engine::Error("ssfv_maintenance.inventory_invalid", "observed inventory is absent", 4);
        }
        const auto& observation = value.at("observation");
        exact_fields(observation, {"protocol", "format_version", "operation", "tops_id", "compatibility",
                                   "inventory", "observed_at", "read_only", "derived", "canonical",
                                   "observation_digest"}, "Maestro inventory observation");
        if (text(observation, "protocol") != "symphony.maestro.receptor-inventory-result.v1" ||
            number(observation, "format_version") != 1U || text(observation, "operation") != "inventory" ||
            text(observation, "tops_id") != tops_id || observation.at("read_only") != true ||
            observation.at("derived") != true || observation.at("canonical") != false ||
            !engine::is_utc_seconds(text(observation, "observed_at", 20U)) ||
            !observation.at("inventory").is_object()) {
            throw engine::Error("ssfv_maintenance.inventory_invalid", "Maestro inventory observation is invalid", 4);
        }
        verify_digest(observation, "observation_digest", "ssfv_maintenance.inventory_invalid");
        const auto& inventory = observation.at("inventory");
        if (!inventory.contains("inventory_digest") || !inventory.contains("tops_id") ||
            inventory.at("tops_id") != tops_id) {
            throw engine::Error("ssfv_maintenance.inventory_invalid", "Maestro stable inventory identity is invalid", 4);
        }
        verify_digest(inventory, "inventory_digest", "ssfv_maintenance.inventory_invalid");
    } else {
        throw engine::Error("ssfv_maintenance.inventory_invalid", "unknown Maestro inventory availability", 4);
    }
    verify_digest(value, "evidence_digest", "ssfv_maintenance.inventory_invalid");
}

std::string review_state(const engine::Json& diff) {
    return diff.at("state") == "identical" ? "current" : "review_required";
}

engine::Json diff_summary(const engine::Json& diff) {
    return diff.at("summary");
}

void validate_extensions(const engine::Json& extensions) {
    if (!extensions.is_array() || extensions.size() > 64U) {
        throw engine::Error("ssfv_maintenance.journal_invalid", "journal extensions are invalid", 5);
    }
    std::set<std::string> seen;
    for (const auto& extension : extensions) {
        exact_fields(extension, {"extension_id", "extension_version", "critical", "payload", "payload_digest"},
                     "SSFV maintenance extension");
        const auto identity = text(extension, "extension_id");
        if (!safe_token(identity) || !safe_version(text(extension, "extension_version", 64U)) ||
            !extension.at("critical").is_boolean() ||
            !tagged_digest(text(extension, "payload_digest", 71U)) ||
            engine::tagged_sha256(extension.at("payload").dump()) !=
                extension.at("payload_digest").get<std::string>() ||
            !seen.insert(identity).second) {
            throw engine::Error("ssfv_maintenance.journal_invalid", "journal extension is invalid", 5);
        }
        if (extension.at("critical") == true) {
            throw engine::Error("ssfv_maintenance.critical_extension_unknown",
                                "unknown critical maintenance state requires a compatible reader", 4);
        }
    }
}

void validate_checkpoint(const engine::Json& checkpoint) {
    exact_fields(checkpoint, {"sequence", "kind", "operation_id", "operation_fingerprint",
                              "session_journal_digest", "binding_registry_digest", "engine_evidence_digest",
                              "current_snapshot_digest", "diff_result_digest", "maestro_inventory_digest",
                              "review_state", "diff_summary", "observed_at", "previous_checkpoint_digest",
                              "checkpoint_digest"}, "SSFV maintenance checkpoint");
    static const std::set<std::string> kinds = {"begin", "checkpoint", "close", "recover"};
    static const std::set<std::string> reviews = {"current", "review_required"};
    if (number(checkpoint, "sequence") == 0U || number(checkpoint, "sequence") > max_checkpoints ||
        !kinds.contains(text(checkpoint, "kind", 32U)) || !safe_token(text(checkpoint, "operation_id")) ||
        !tagged_digest(text(checkpoint, "operation_fingerprint", 71U)) ||
        !tagged_digest(text(checkpoint, "session_journal_digest", 71U)) ||
        !tagged_digest(text(checkpoint, "binding_registry_digest", 71U)) ||
        !tagged_digest(text(checkpoint, "engine_evidence_digest", 71U)) ||
        !tagged_digest(text(checkpoint, "current_snapshot_digest", 71U)) ||
        !tagged_digest(text(checkpoint, "maestro_inventory_digest", 71U)) ||
        !reviews.contains(text(checkpoint, "review_state", 32U)) ||
        !checkpoint.at("diff_summary").is_object() ||
        !engine::is_utc_seconds(text(checkpoint, "observed_at", 20U))) {
        throw engine::Error("ssfv_maintenance.journal_invalid", "checkpoint identity is invalid", 5);
    }
    if (!checkpoint.at("diff_result_digest").is_null() &&
        (!checkpoint.at("diff_result_digest").is_string() ||
         !tagged_digest(checkpoint.at("diff_result_digest").get<std::string>()))) {
        throw engine::Error("ssfv_maintenance.journal_invalid", "checkpoint diff digest is invalid", 5);
    }
    const auto& previous = checkpoint.at("previous_checkpoint_digest");
    if (!previous.is_null() && (!previous.is_string() || !tagged_digest(previous.get<std::string>()))) {
        throw engine::Error("ssfv_maintenance.journal_invalid", "checkpoint predecessor is invalid", 5);
    }
    verify_digest(checkpoint, "checkpoint_digest", "ssfv_maintenance.journal_invalid");
}

void validate_journal(const engine::Json& journal) {
    exact_fields(journal, {"protocol", "format_version", "journal_id", "context_key", "generation",
                           "previous_journal_digest", "state", "tops_id", "subject_id", "repository_root",
                           "session_journal_digest", "binding_registry_digest", "baseline_snapshot",
                           "baseline_engine", "current_snapshot_digest", "current_engine", "maestro_inventory_digest",
                           "review_state", "checkpoints", "compatibility", "extensions", "recovery",
                           "started_at", "updated_at", "closed_at", "canonical", "journal_digest"},
                 "SSFV maintenance journal");
    if (text(journal, "protocol") != journal_protocol || number(journal, "format_version") != format_version ||
        !safe_token(text(journal, "journal_id")) || !tagged_digest(text(journal, "context_key", 71U)) ||
        number(journal, "generation") == 0U ||
        (text(journal, "state") != "open" && text(journal, "state") != "closed") ||
        !valid_uuid(text(journal, "tops_id")) || !safe_token(text(journal, "subject_id")) ||
        !safe_absolute_path(text(journal, "repository_root")) ||
        !tagged_digest(text(journal, "session_journal_digest", 71U)) ||
        !tagged_digest(text(journal, "binding_registry_digest", 71U)) ||
        !tagged_digest(text(journal, "current_snapshot_digest", 71U)) ||
        !tagged_digest(text(journal, "maestro_inventory_digest", 71U)) ||
        (text(journal, "review_state") != "current" && text(journal, "review_state") != "review_required") ||
        journal.at("canonical") != false) {
        throw engine::Error("ssfv_maintenance.journal_invalid", "journal identity is invalid", 5);
    }
    if ((number(journal, "generation") == 1U) != journal.at("previous_journal_digest").is_null() ||
        (!journal.at("previous_journal_digest").is_null() &&
         (!journal.at("previous_journal_digest").is_string() ||
          !tagged_digest(journal.at("previous_journal_digest").get<std::string>())))) {
        throw engine::Error("ssfv_maintenance.journal_invalid", "journal predecessor is invalid", 5);
    }
    // The baseline must remain readable after a later checkpoint is produced by
    // a newer compatible SSFV engine.  Keep both identities so upgrade order
    // cannot silently reinterpret historical evidence.
    validate_engine_evidence(journal.at("baseline_engine"));
    validate_engine_evidence(journal.at("current_engine"));
    validate_snapshot(journal.at("baseline_snapshot"), journal.at("baseline_engine"));
    if (!journal.at("checkpoints").is_array() || journal.at("checkpoints").empty() ||
        journal.at("checkpoints").size() > max_checkpoints ||
        journal.at("compatibility") != journal_compatibility()) {
        throw engine::Error("ssfv_maintenance.journal_invalid", "journal collections or compatibility are invalid", 5);
    }
    validate_extensions(journal.at("extensions"));
    std::string previous;
    std::uint64_t sequence = 1U;
    for (const auto& checkpoint : journal.at("checkpoints")) {
        validate_checkpoint(checkpoint);
        if (number(checkpoint, "sequence") != sequence ||
            (sequence == 1U) != checkpoint.at("previous_checkpoint_digest").is_null() ||
            (sequence > 1U && checkpoint.at("previous_checkpoint_digest") != previous)) {
            throw engine::Error("ssfv_maintenance.journal_invalid", "checkpoint chain is discontinuous", 5);
        }
        previous = checkpoint.at("checkpoint_digest");
        ++sequence;
    }
    exact_fields(journal.at("recovery"), {"state", "disposition", "recovered_from_digest", "detail"},
                 "SSFV maintenance recovery");
    if (!engine::is_utc_seconds(text(journal, "started_at", 20U)) ||
        !engine::is_utc_seconds(text(journal, "updated_at", 20U)) ||
        text(journal, "started_at", 20U) > text(journal, "updated_at", 20U)) {
        throw engine::Error("ssfv_maintenance.journal_invalid", "journal time evidence is invalid", 5);
    }
    if (journal.at("state") == "closed") {
        if (!journal.at("closed_at").is_string() ||
            !engine::is_utc_seconds(journal.at("closed_at").get<std::string>())) {
            throw engine::Error("ssfv_maintenance.journal_invalid", "closed journal lacks a close time", 5);
        }
    } else if (!journal.at("closed_at").is_null()) {
        throw engine::Error("ssfv_maintenance.journal_invalid", "open journal carries a close time", 5);
    }
    verify_digest(journal, "journal_digest", "ssfv_maintenance.journal_invalid");
}

void validate_head(const engine::Json& head) {
    exact_fields(head, {"protocol", "format_version", "context_key", "active_slot", "generation",
                        "journal_digest", "previous_head_digest", "updated_at", "head_digest"},
                 "SSFV maintenance head");
    if (text(head, "protocol") != head_protocol || number(head, "format_version") != format_version ||
        !tagged_digest(text(head, "context_key", 71U)) || number(head, "active_slot") > 1U ||
        number(head, "generation") == 0U || !tagged_digest(text(head, "journal_digest", 71U)) ||
        !engine::is_utc_seconds(text(head, "updated_at", 20U))) {
        throw engine::Error("ssfv_maintenance.head_invalid", "maintenance head is invalid", 5);
    }
    if (!head.at("previous_head_digest").is_null() &&
        (!head.at("previous_head_digest").is_string() ||
         !tagged_digest(head.at("previous_head_digest").get<std::string>()))) {
        throw engine::Error("ssfv_maintenance.head_invalid", "head predecessor is invalid", 5);
    }
    verify_digest(head, "head_digest", "ssfv_maintenance.head_invalid");
}

std::optional<FileDescriptor> open_absolute_directory(const std::string& path, bool create) {
    if (!safe_absolute_path(path)) {
        throw engine::Error("ssfv_maintenance.state_root_invalid", "state root is invalid", 4);
    }
    FileDescriptor current(::open("/", O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC));
    if (current.get() < 0) system_error("ssfv_maintenance.state_open_failed", "could not open filesystem root");
    for (const auto& item : fs::path(path).relative_path()) {
        const auto component = item.string();
        int next = ::openat(current.get(), component.c_str(), O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
        if (next < 0 && errno == ENOENT && !create) return std::nullopt;
        if (next < 0 && errno == ENOENT && create) {
            if (::mkdirat(current.get(), component.c_str(), 0700) != 0 && errno != EEXIST) {
                system_error("ssfv_maintenance.state_create_failed", "could not create state directory");
            }
            if (::fsync(current.get()) != 0) system_error("ssfv_maintenance.state_sync_failed", "could not sync parent");
            next = ::openat(current.get(), component.c_str(), O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
        }
        if (next < 0) system_error("ssfv_maintenance.state_open_failed", "could not open state directory");
        current = FileDescriptor(next);
    }
    struct stat status {};
    if (::fstat(current.get(), &status) != 0 || !S_ISDIR(status.st_mode) ||
        status.st_uid != ::geteuid() || (status.st_mode & 0022) != 0) {
        throw engine::Error("ssfv_maintenance.state_unsafe", "state root must be caller-owned and protected", 5);
    }
    return std::optional<FileDescriptor>(std::move(current));
}

std::optional<FileDescriptor> open_child_directory(FileDescriptor parent, const std::string& name, bool create) {
    int next = ::openat(parent.get(), name.c_str(), O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
    if (next < 0 && errno == ENOENT && !create) return std::nullopt;
    if (next < 0 && errno == ENOENT && create) {
        if (::mkdirat(parent.get(), name.c_str(), 0700) != 0 && errno != EEXIST) {
            system_error("ssfv_maintenance.state_create_failed", "could not create maintenance directory");
        }
        if (::fsync(parent.get()) != 0) system_error("ssfv_maintenance.state_sync_failed", "could not sync parent");
        next = ::openat(parent.get(), name.c_str(), O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
    }
    if (next < 0) system_error("ssfv_maintenance.state_open_failed", "could not open maintenance directory");
    FileDescriptor result(next);
    struct stat status {};
    if (::fstat(result.get(), &status) != 0 || !S_ISDIR(status.st_mode) || status.st_uid != ::geteuid() ||
        (status.st_mode & 0022) != 0) {
        throw engine::Error("ssfv_maintenance.state_unsafe", "maintenance directory is unsafe", 5);
    }
    if (create && ::fchmod(result.get(), 0700) != 0) {
        system_error("ssfv_maintenance.state_mode_failed", "could not restrict maintenance directory");
    }
    return std::optional<FileDescriptor>(std::move(result));
}

std::optional<JournalLock> open_stream(const std::string& root, const std::string& context_key,
                                       bool exclusive, bool create) {
    auto opened = open_absolute_directory(root, create);
    if (!opened.has_value()) return std::nullopt;
    FileDescriptor current = std::move(*opened);
    const std::array<std::string, 6> components = {
        "symphony", "knowledge-session-coordinator", "ssfv-maintenance", "v1", "contexts",
        context_key.substr(7U),
    };
    for (const auto& component : components) {
        auto child = open_child_directory(std::move(current), component, create);
        if (!child.has_value()) return std::nullopt;
        current = std::move(*child);
    }
    const int raw = ::openat(current.get(), ".lock",
        (create ? O_RDWR | O_CREAT : O_RDONLY) | O_NONBLOCK | O_NOFOLLOW | O_CLOEXEC, 0600);
    if (raw < 0 && errno == ENOENT && !create) return std::nullopt;
    if (raw < 0) system_error("ssfv_maintenance.lock_open_failed", "could not open maintenance lock");
    FileDescriptor lock(raw);
    struct stat status {};
    if (::fstat(lock.get(), &status) != 0 || !S_ISREG(status.st_mode) ||
        (status.st_mode & 0777) != 0600 || status.st_uid != ::geteuid() || status.st_nlink != 1) {
        throw engine::Error("ssfv_maintenance.lock_unsafe", "maintenance lock is unsafe", 5);
    }
    if (::flock(lock.get(), (exclusive ? LOCK_EX : LOCK_SH) | LOCK_NB) != 0) {
        if (errno == EWOULDBLOCK) throw engine::Error("ssfv_maintenance.lock_busy", "maintenance state is busy", 4);
        system_error("ssfv_maintenance.lock_failed", "could not lock maintenance state");
    }
    return JournalLock(std::move(current), std::move(lock));
}

std::optional<std::string> read_file(int directory, const std::string& name) {
    const int raw = ::openat(directory, name.c_str(), O_RDONLY | O_NONBLOCK | O_NOFOLLOW | O_CLOEXEC);
    if (raw < 0) {
        if (errno == ENOENT) return std::nullopt;
        system_error("ssfv_maintenance.state_read_failed", "could not open maintenance file");
    }
    FileDescriptor file(raw);
    struct stat status {};
    if (::fstat(file.get(), &status) != 0 || !S_ISREG(status.st_mode) ||
        (status.st_mode & 0777) != 0600 || status.st_uid != ::geteuid() || status.st_nlink != 1 ||
        status.st_size < 0 || static_cast<std::uint64_t>(status.st_size) > max_state_bytes) {
        throw engine::Error("ssfv_maintenance.state_unsafe", "maintenance file is unsafe", 5);
    }
    std::string data;
    data.reserve(static_cast<std::size_t>(status.st_size));
    std::array<char, 16384> buffer {};
    for (;;) {
        const auto count = ::read(file.get(), buffer.data(), buffer.size());
        if (count < 0) {
            if (errno == EINTR) continue;
            system_error("ssfv_maintenance.state_read_failed", "could not read maintenance file");
        }
        if (count == 0) break;
        if (data.size() + static_cast<std::size_t>(count) > max_state_bytes) {
            throw engine::Error("ssfv_maintenance.state_too_large", "maintenance file exceeds its bound", 5);
        }
        data.append(buffer.data(), static_cast<std::size_t>(count));
    }
    return data;
}

engine::Json parse_file(const std::string& data) {
    try { return engine::parse_bounded_json(data, max_state_bytes); }
    catch (const engine::Error&) {
        throw engine::Error("ssfv_maintenance.state_json_invalid", "maintenance state is invalid JSON", 5);
    }
}

Candidate read_candidate(int directory, int slot) {
    Candidate candidate;
    candidate.slot = slot;
    const auto data = read_file(directory, "journal." + std::to_string(slot) + ".json");
    if (!data.has_value()) return candidate;
    candidate.exists = true;
    try {
        candidate.journal = parse_file(*data);
        if (!candidate.journal.is_object() || !candidate.journal.contains("protocol") ||
            !candidate.journal.contains("format_version") ||
            candidate.journal.at("protocol") != journal_protocol ||
            candidate.journal.at("format_version") != format_version) {
            candidate.incompatible = true;
            return candidate;
        }
        validate_journal(candidate.journal);
        candidate.valid = true;
    } catch (const engine::Error& error) {
        candidate.incompatible = error.code() == "ssfv_maintenance.critical_extension_unknown";
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
            throw engine::Error("ssfv_maintenance.compatibility_required", "head uses unsupported state", 4);
        }
        validate_head(head);
        return head;
    } catch (const engine::Error& error) {
        if (tolerate_invalid && error.code() != "ssfv_maintenance.compatibility_required") return std::nullopt;
        throw;
    }
}

State load_state(int directory) {
    const auto head = read_head(directory);
    if (!head.has_value()) {
        const auto zero = read_candidate(directory, 0);
        const auto one = read_candidate(directory, 1);
        if (zero.exists || one.exists) throw engine::Error("ssfv_maintenance.head_missing", "journal slots exist without a head", 5);
        return {};
    }
    const int slot = static_cast<int>(number(*head, "active_slot"));
    const auto active = read_candidate(directory, slot);
    if (active.incompatible) throw engine::Error("ssfv_maintenance.compatibility_required", "active journal is incompatible", 4);
    if (!active.valid || active.journal.at("journal_digest") != head->at("journal_digest") ||
        active.journal.at("generation") != head->at("generation") ||
        active.journal.at("context_key") != head->at("context_key")) {
        throw engine::Error("ssfv_maintenance.head_slot_mismatch", "head does not select its journal", 5);
    }
    const auto inactive = read_candidate(directory, 1 - slot);
    if (inactive.incompatible) throw engine::Error("ssfv_maintenance.compatibility_required", "inactive journal is incompatible", 4);
    const auto generation = number(active.journal, "generation");
    if (!inactive.exists && generation > 1U) throw engine::Error("ssfv_maintenance.recovery_required", "journal predecessor is missing", 5);
    if (inactive.exists && !inactive.valid) throw engine::Error("ssfv_maintenance.recovery_required", "journal predecessor is invalid", 5);
    if (inactive.valid) {
        const auto other = number(inactive.journal, "generation");
        if (other == generation) {
            throw engine::Error(active.journal.at("journal_digest") == inactive.journal.at("journal_digest") ?
                "ssfv_maintenance.recovery_required" : "ssfv_maintenance.recovery_ambiguous",
                "journal slots share one generation", 5);
        }
        if (other > generation) {
            const bool linked = other == generation + 1U &&
                inactive.journal.at("previous_journal_digest") == active.journal.at("journal_digest");
            throw engine::Error(linked ? "ssfv_maintenance.recovery_required" : "ssfv_maintenance.recovery_ambiguous",
                                "inactive journal is newer than the selected head", 5);
        }
        if (generation != other + 1U ||
            active.journal.at("previous_journal_digest") != inactive.journal.at("journal_digest")) {
            throw engine::Error("ssfv_maintenance.recovery_ambiguous", "journal slots are not one linked chain", 5);
        }
    }
    return State{*head, active.journal, true};
}

void write_all(int file, const std::string& data) {
    std::size_t offset = 0U;
    while (offset < data.size()) {
        const auto count = ::write(file, data.data() + offset, data.size() - offset);
        if (count < 0) {
            if (errno == EINTR) continue;
            system_error("ssfv_maintenance.state_write_failed", "could not write maintenance state");
        }
        offset += static_cast<std::size_t>(count);
    }
}

void write_slot(int directory, int slot, const engine::Json& journal) {
    const auto data = journal.dump() + "\n";
    if (data.size() > max_state_bytes) throw engine::Error("ssfv_maintenance.state_too_large", "journal exceeds its bound", 5);
    const auto name = "journal." + std::to_string(slot) + ".json";
    const int raw = ::openat(directory, name.c_str(), O_WRONLY | O_CREAT | O_NONBLOCK | O_NOFOLLOW | O_CLOEXEC, 0600);
    if (raw < 0) system_error("ssfv_maintenance.state_write_failed", "could not open inactive journal slot");
    FileDescriptor file(raw);
    struct stat status {};
    if (::fstat(file.get(), &status) != 0 || !S_ISREG(status.st_mode) || status.st_uid != ::geteuid() ||
        (status.st_mode & 0777) != 0600 || status.st_nlink != 1) {
        throw engine::Error("ssfv_maintenance.state_unsafe", "journal slot is unsafe", 5);
    }
    if (::ftruncate(file.get(), 0) != 0) system_error("ssfv_maintenance.state_write_failed", "could not truncate journal");
    write_all(file.get(), data);
    if (::fsync(file.get()) != 0) system_error("ssfv_maintenance.state_sync_failed", "could not sync journal");
}

void write_head(int directory, engine::Json head) {
    finalize_digest(head, "head_digest");
    static std::atomic<std::uint64_t> sequence {0U};
    const auto temporary = ".head.tmp." + std::to_string(::getpid()) + "." +
        std::to_string(sequence.fetch_add(1U, std::memory_order_relaxed));
    const int raw = ::openat(directory, temporary.c_str(), O_WRONLY | O_CREAT | O_EXCL | O_NOFOLLOW | O_CLOEXEC, 0600);
    if (raw < 0) system_error("ssfv_maintenance.head_write_failed", "could not create temporary head");
    {
        FileDescriptor file(raw);
        write_all(file.get(), head.dump() + "\n");
        if (::fsync(file.get()) != 0) system_error("ssfv_maintenance.state_sync_failed", "could not sync head");
    }
    if (::renameat(directory, temporary.c_str(), directory, "head.json") != 0) {
        static_cast<void>(::unlinkat(directory, temporary.c_str(), 0));
        system_error("ssfv_maintenance.head_commit_failed", "could not replace maintenance head");
    }
    if (::fsync(directory) != 0) system_error("ssfv_maintenance.state_sync_failed", "could not sync maintenance directory");
}

State commit_to_slot(int directory, engine::Json journal, int slot,
                     const std::optional<engine::Json>& prior_head) {
    finalize_digest(journal, "journal_digest");
    validate_journal(journal);
    write_slot(directory, slot, journal);
    if (::fsync(directory) != 0) system_error("ssfv_maintenance.state_sync_failed", "could not sync journal directory");
    engine::Json head{
        {"protocol", head_protocol}, {"format_version", format_version},
        {"context_key", journal.at("context_key")}, {"active_slot", slot},
        {"generation", journal.at("generation")}, {"journal_digest", journal.at("journal_digest")},
        {"previous_head_digest", prior_head.has_value() ? prior_head->at("head_digest") : engine::Json(nullptr)},
        {"updated_at", journal.at("updated_at")},
    };
    write_head(directory, head);
    return State{*read_head(directory), std::move(journal), true};
}

State commit(int directory, engine::Json journal, const std::optional<engine::Json>& prior_head) {
    const int slot = prior_head.has_value() ? 1 - static_cast<int>(number(*prior_head, "active_slot")) : 0;
    return commit_to_slot(directory, std::move(journal), slot, prior_head);
}

Candidate choose_recovery_candidate(int directory) {
    const auto zero = read_candidate(directory, 0);
    const auto one = read_candidate(directory, 1);
    if (zero.incompatible || one.incompatible) throw engine::Error("ssfv_maintenance.compatibility_required", "stored state is incompatible", 4);
    std::vector<Candidate> valid;
    if (zero.valid) valid.push_back(zero);
    if (one.valid) valid.push_back(one);
    if (valid.empty()) throw engine::Error("ssfv_maintenance.recovery_unavailable", "no valid maintenance journal exists", 5);
    if (valid.size() == 1U) return valid.front();
    const auto first = number(valid[0].journal, "generation");
    const auto second = number(valid[1].journal, "generation");
    if (first == second) {
        if (valid[0].journal.at("journal_digest") == valid[1].journal.at("journal_digest")) return valid[0];
        throw engine::Error("ssfv_maintenance.recovery_ambiguous", "journals diverge at one generation", 5);
    }
    const Candidate& newer = first > second ? valid[0] : valid[1];
    const Candidate& older = first > second ? valid[1] : valid[0];
    if (number(newer.journal, "generation") != number(older.journal, "generation") + 1U ||
        newer.journal.at("previous_journal_digest") != older.journal.at("journal_digest")) {
        throw engine::Error("ssfv_maintenance.recovery_ambiguous", "journals do not form one linked chain", 5);
    }
    return newer;
}

std::string context_key(const std::string& tops_id, const std::string& subject_id,
                        const std::string& repository_root) {
    return engine::tagged_sha256(tops_id + "\n" + subject_id + "\n" + repository_root);
}

std::string resource(const std::string& tops_id, const std::string& repository_root,
                     const std::string& operation, const std::string& expected,
                     const std::string& session_digest, const std::string& snapshot_digest,
                     const std::string& inventory_digest) {
    return "symphony.knowledge.ssfv-maintenance:" + engine::sha256_hex(
        tops_id + "\n" + repository_root + "\n" + operation + "\n" + expected + "\n" +
        session_digest + "\n" + snapshot_digest + "\n" + inventory_digest);
}

std::string operation_fingerprint(const engine::Json& payload) {
    engine::Json value{
        {"operation", payload.at("operation")}, {"operation_id", payload.at("operation_id")},
        {"expected_journal_digest", payload.at("expected_journal_digest")},
        {"repository_root", payload.at("repository_root")},
        {"session_journal_digest", payload.at("session_journal_digest")},
        {"binding_registry_digest", payload.at("binding_registry_digest")},
        {"engine", payload.at("engine")}, {"semantic_snapshot", payload.at("semantic_snapshot")},
        {"diff_result", payload.at("diff_result")}, {"maestro_inventory", payload.at("maestro_inventory")},
    };
    return engine::tagged_sha256(value.dump());
}

std::optional<bool> replayed(const engine::Json& journal, const std::string& operation_id,
                             const std::string& fingerprint) {
    for (const auto& checkpoint : journal.at("checkpoints")) {
        if (checkpoint.at("operation_id") == operation_id) {
            if (checkpoint.at("operation_fingerprint") != fingerprint) {
                throw engine::Error("ssfv_maintenance.operation_reused", "operation id was reused for different evidence", 4);
            }
            return true;
        }
    }
    return std::nullopt;
}

engine::Json clean_recovery() {
    return engine::Json{{"state", "clean"}, {"disposition", "not_applicable"},
                        {"recovered_from_digest", nullptr}, {"detail", "no recovery was required"}};
}

engine::Json checkpoint(const std::string& kind, const std::string& operation_id,
                        const std::string& fingerprint, const engine::Json& payload,
                        const engine::Json& engine_evidence, const engine::Json& snapshot,
                        const engine::Json* diff, std::uint64_t sequence,
                        const engine::Json& previous) {
    const auto inventory_digest = text(payload.at("maestro_inventory"), "evidence_digest", 71U);
    engine::Json value{
        {"sequence", sequence}, {"kind", kind}, {"operation_id", operation_id},
        {"operation_fingerprint", fingerprint},
        {"session_journal_digest", payload.at("session_journal_digest")},
        {"binding_registry_digest", payload.at("binding_registry_digest")},
        {"engine_evidence_digest", engine_evidence.at("evidence_digest")},
        {"current_snapshot_digest", snapshot.at("snapshot_digest")},
        {"diff_result_digest", diff == nullptr ? engine::Json(nullptr) : diff->at("result_digest")},
        {"maestro_inventory_digest", inventory_digest},
        {"review_state", diff == nullptr ? "current" : review_state(*diff)},
        {"diff_summary", diff == nullptr ? engine::Json{{"added", 0}, {"changed", 0}, {"removed", 0},
            {"uncovered", 0}, {"stale", 0}, {"review_required", 0}} : diff_summary(*diff)},
        {"observed_at", utc_now()}, {"previous_checkpoint_digest", previous},
    };
    finalize_digest(value, "checkpoint_digest");
    return value;
}

engine::Json make_result(const std::string& operation, const engine::Json& compatibility,
                         const State& state, const std::string& effective_state,
                         bool changed, bool recovered, engine::Json actions, bool read_only) {
    return engine::Json{
        {"protocol", result_protocol}, {"format_version", format_version}, {"operation", operation},
        {"compatibility", compatibility}, {"journal_present", state.present},
        {"journal", state.present ? state.journal : engine::Json(nullptr)},
        {"journal_digest", state.present ? state.journal.at("journal_digest") : engine::Json(nullptr)},
        {"effective_state", effective_state}, {"review_state", state.present ? state.journal.at("review_state") : engine::Json("not_evaluated")},
        {"changed", changed}, {"recovered", recovered}, {"repair_actions", std::move(actions)},
        {"read_only", read_only}, {"canonical_apply_enabled", false}, {"canonical", false},
    };
}

engine::Json validate_command(const engine::Request& request) {
    exact_fields(request.payload, {"protocol", "operation", "state_root", "tops_id", "operation_id",
                                   "expected_journal_digest", "repository_root", "session_journal_digest",
                                   "binding_registry_digest", "engine", "semantic_snapshot", "diff_result",
                                   "maestro_inventory", "authorization_decision", "client"},
                 "SSFV maintenance command");
    static const std::set<std::string> operations = {
        "ssfv_maintenance_begin", "ssfv_maintenance_status", "ssfv_maintenance_checkpoint",
        "ssfv_maintenance_close", "ssfv_maintenance_recover",
    };
    if (text(request.payload, "protocol") != command_protocol ||
        text(request.payload, "operation") != request.operation || !operations.contains(request.operation) ||
        !safe_absolute_path(text(request.payload, "state_root")) ||
        !valid_uuid(text(request.payload, "tops_id")) ||
        !safe_absolute_path(text(request.payload, "repository_root")) ||
        !request.payload.at("authorization_decision").is_object()) {
        throw engine::Error("ssfv_maintenance.command_invalid", "SSFV maintenance command is invalid", 4);
    }
    return compatibility_result(request.payload.at("client"), nullptr);
}

} // namespace

engine::Json ssfv_maintenance_capabilities() {
    return engine::Json{
        {"protocol", command_protocol}, {"journal_protocol", journal_protocol},
        {"format_version", format_version}, {"required_capabilities", required_capabilities},
        {"optional_capabilities", optional_capabilities}, {"canonical_apply_enabled", false},
    };
}

engine::Json handle_ssfv_maintenance(const engine::Request& request) {
    auto compatibility = validate_command(request);
    const auto& payload = request.payload;
    const auto tops_id = text(payload, "tops_id");
    const auto repository_root = text(payload, "repository_root");
    const bool status = request.operation == "ssfv_maintenance_status";
    const bool recover = request.operation == "ssfv_maintenance_recover";
    const bool begin = request.operation == "ssfv_maintenance_begin";
    const bool close = request.operation == "ssfv_maintenance_close";

    if (status) {
        if (!payload.at("operation_id").is_null() || !payload.at("expected_journal_digest").is_null() ||
            !payload.at("session_journal_digest").is_null() || !payload.at("binding_registry_digest").is_null() ||
            !payload.at("engine").is_null() || !payload.at("semantic_snapshot").is_null() ||
            !payload.at("diff_result").is_null() || !payload.at("maestro_inventory").is_null()) {
            throw engine::Error("ssfv_maintenance.command_invalid", "status carries mutation evidence", 4);
        }
    } else if (!payload.at("operation_id").is_string() ||
               !safe_token(payload.at("operation_id").get<std::string>()) ||
               !payload.at("expected_journal_digest").is_string() ||
               !payload.at("session_journal_digest").is_string() ||
               !tagged_digest(payload.at("session_journal_digest").get<std::string>())) {
        throw engine::Error("ssfv_maintenance.command_invalid", "mutation identity is invalid", 4);
    }

    const auto& decision = payload.at("authorization_decision");
    if (!decision.contains("subject") || !decision.at("subject").is_object() ||
        !decision.at("subject").contains("id") || !decision.at("subject").at("id").is_string()) {
        throw engine::Error("ssfv_maintenance.authorization_invalid", "authorization subject is absent", 4);
    }
    const auto subject_id = decision.at("subject").at("id").get<std::string>();
    if (!safe_token(subject_id)) throw engine::Error("ssfv_maintenance.authorization_invalid", "authorization subject is invalid", 4);
    const auto key = context_key(tops_id, subject_id, repository_root);

    std::string expected = status ? "status" : text(payload, "expected_journal_digest", 71U);
    if (!status && expected != "absent" && expected != "discover" && !tagged_digest(expected)) {
        throw engine::Error("ssfv_maintenance.command_invalid", "expected journal state is invalid", 4);
    }
    if ((begin && expected == "discover") ||
        (!recover && !begin && !status && !tagged_digest(expected)) ||
        (recover && expected != "discover" && !tagged_digest(expected))) {
        throw engine::Error("ssfv_maintenance.command_invalid", "operation expected state is invalid", 4);
    }

    std::string session_digest = status ? "none" : text(payload, "session_journal_digest", 71U);
    std::string snapshot_digest = "none";
    std::string inventory_digest = "none";
    if (!status && !recover) {
        if (!payload.at("binding_registry_digest").is_string() ||
            !tagged_digest(payload.at("binding_registry_digest").get<std::string>()) ||
            !payload.at("engine").is_object() || !payload.at("semantic_snapshot").is_object() ||
            !payload.at("maestro_inventory").is_object()) {
            throw engine::Error("ssfv_maintenance.command_invalid", "semantic maintenance evidence is incomplete", 4);
        }
        validate_engine_evidence(payload.at("engine"));
        validate_snapshot(payload.at("semantic_snapshot"), payload.at("engine"));
        validate_maestro_evidence(payload.at("maestro_inventory"), tops_id);
        snapshot_digest = text(payload.at("semantic_snapshot"), "snapshot_digest", 71U);
        inventory_digest = text(payload.at("maestro_inventory"), "evidence_digest", 71U);
        if (begin) {
            if (!payload.at("diff_result").is_null()) {
                throw engine::Error("ssfv_maintenance.command_invalid", "begin carries a diff result", 4);
            }
        } else {
            if (!payload.at("diff_result").is_object()) {
                throw engine::Error("ssfv_maintenance.command_invalid", "checkpoint or close lacks a diff result", 4);
            }
        }
    } else if (!status && (!payload.at("binding_registry_digest").is_null() || !payload.at("engine").is_null() ||
                           !payload.at("semantic_snapshot").is_null() || !payload.at("diff_result").is_null() ||
                           !payload.at("maestro_inventory").is_null())) {
        throw engine::Error("ssfv_maintenance.command_invalid", "recovery carries semantic evidence", 4);
    }

    const auto authorization_resource = resource(
        tops_id, repository_root, request.operation, expected, session_digest,
        snapshot_digest, inventory_digest);
    static_cast<void>(validate_ssiag_authorization(
        decision, "symphony.knowledge." + request.operation, tops_id, authorization_resource));

    const auto state_root = text(payload, "state_root");
    auto stream = open_stream(state_root, key, !status, !status && !recover);
    if (!stream.has_value()) {
        if (recover) {
            return make_result(request.operation, compatibility, {}, "absent", false, false,
                               engine::Json::array(), false);
        }
        return make_result(request.operation, compatibility, {}, "absent", false, false,
                           engine::Json::array(), true);
    }

    if (recover) {
        try {
            auto healthy = load_state(stream->directory_fd());
            if (expected != "discover" &&
                expected != healthy.journal.at("journal_digest").get<std::string>()) {
                throw engine::Error("ssfv_maintenance.stale_expected_state", "recovery expected state is stale", 4);
            }
            compatibility = compatibility_result(payload.at("client"), &healthy.journal);
            return make_result(request.operation, compatibility, healthy,
                               healthy.journal.at("state").get<std::string>(), false, false,
                               engine::Json::array(), false);
        } catch (const engine::Error& error) {
            static const std::set<std::string> recoverable = {
                "ssfv_maintenance.head_invalid", "ssfv_maintenance.head_missing",
                "ssfv_maintenance.head_slot_mismatch", "ssfv_maintenance.recovery_required",
                "ssfv_maintenance.state_json_invalid",
            };
            if (!recoverable.contains(error.code())) throw;
        }
        const auto selected = choose_recovery_candidate(stream->directory_fd());
        if (selected.journal.at("context_key").get<std::string>() != key) {
            throw engine::Error("ssfv_maintenance.recovery_ambiguous", "recoverable journal belongs to another context", 5);
        }
        if (expected != "discover" &&
            expected != selected.journal.at("journal_digest").get<std::string>()) {
            throw engine::Error("ssfv_maintenance.stale_expected_state", "recovery expected state is stale", 4);
        }
        auto repaired = selected.journal;
        const auto previous = repaired.at("journal_digest");
        if (repaired.at("checkpoints").size() >= max_checkpoints) {
            throw engine::Error("ssfv_maintenance.checkpoint_limit", "maintenance checkpoint limit reached", 4);
        }
        const auto recovery_id = text(payload, "operation_id");
        engine::Json recovery_payload = payload;
        const auto fingerprint = operation_fingerprint(recovery_payload);
        engine::Json recovery_checkpoint{
            {"sequence", repaired.at("checkpoints").size() + 1U}, {"kind", "recover"},
            {"operation_id", recovery_id}, {"operation_fingerprint", fingerprint},
            {"session_journal_digest", repaired.at("session_journal_digest")},
            {"binding_registry_digest", repaired.at("binding_registry_digest")},
            {"engine_evidence_digest", repaired.at("current_engine").at("evidence_digest")},
            {"current_snapshot_digest", repaired.at("current_snapshot_digest")},
            {"diff_result_digest", nullptr}, {"maestro_inventory_digest", repaired.at("maestro_inventory_digest")},
            {"review_state", repaired.at("review_state")},
            {"diff_summary", engine::Json{{"added", 0}, {"changed", 0}, {"removed", 0},
                {"uncovered", 0}, {"stale", 0}, {"review_required", 0}}},
            {"observed_at", utc_now()},
            {"previous_checkpoint_digest", repaired.at("checkpoints").back().at("checkpoint_digest")},
        };
        finalize_digest(recovery_checkpoint, "checkpoint_digest");
        repaired["generation"] = number(repaired, "generation") + 1U;
        repaired["previous_journal_digest"] = previous;
        repaired["checkpoints"].push_back(std::move(recovery_checkpoint));
        repaired["updated_at"] = utc_now();
        repaired["recovery"] = engine::Json{{"state", "recovered"}, {"disposition", "selected_unique_forward_state"},
            {"recovered_from_digest", previous}, {"detail", "selected one unique linked journal and republished the head"}};
        const auto old_head = read_head(stream->directory_fd(), true);
        auto committed = commit_to_slot(stream->directory_fd(), std::move(repaired), 1 - selected.slot, old_head);
        return make_result(request.operation, compatibility_result(payload.at("client"), &committed.journal),
                           committed, committed.journal.at("state").get<std::string>(), true, true,
                           engine::Json::array({"selected one unique linked SSFV maintenance journal",
                                                "committed a forward recovery generation and republished the head"}), false);
    }

    auto state = load_state(stream->directory_fd());
    compatibility = compatibility_result(payload.at("client"), state.present ? &state.journal : nullptr);
    if (status) {
        const auto effective = !state.present ? "absent" : state.journal.at("state").get<std::string>();
        return make_result(request.operation, compatibility, state, effective, false, false,
                           engine::Json::array(), true);
    }
    if (compatibility.at("mode") != "full") {
        throw engine::Error("ssfv_maintenance.compatibility_required", "maintenance mutation requires full compatibility", 4);
    }
    const auto operation_id = text(payload, "operation_id");
    const auto fingerprint = operation_fingerprint(payload);
    if (state.present && replayed(state.journal, operation_id, fingerprint).has_value()) {
        return make_result(request.operation, compatibility, state,
                           state.journal.at("state").get<std::string>(),
                           false, false, engine::Json::array(), false);
    }
    if ((!state.present && expected != "absent") ||
        (state.present && expected != state.journal.at("journal_digest").get<std::string>())) {
        throw engine::Error("ssfv_maintenance.stale_expected_state", "expected journal state is stale", 4);
    }

    const auto& engine_evidence = payload.at("engine");
    const auto& snapshot = payload.at("semantic_snapshot");
    if (!begin) {
        if (!state.present || state.journal.at("state") != "open") {
            throw engine::Error("ssfv_maintenance.state_invalid", "checkpoint or close requires an open journal", 4);
        }
        validate_diff(payload.at("diff_result"),
                      state.journal.at("baseline_snapshot").at("snapshot_digest"),
                      snapshot.at("snapshot_digest"));
    } else if (state.present && state.journal.at("state") == "open") {
        throw engine::Error("ssfv_maintenance.state_invalid", "begin requires absent or closed maintenance state", 4);
    }

    engine::Json next;
    if (begin) {
        const auto now = utc_now();
        const auto sequence = 1U;
        auto first = checkpoint("begin", operation_id, fingerprint, payload, engine_evidence,
                                snapshot, nullptr, sequence, nullptr);
        next = engine::Json{
            {"protocol", journal_protocol}, {"format_version", format_version},
            {"journal_id", "ssfv-maintenance:" + engine::sha256_hex(key + "\n" + operation_id)},
            {"context_key", key}, {"generation", state.present ? number(state.journal, "generation") + 1U : 1U},
            {"previous_journal_digest", state.present ? state.journal.at("journal_digest") : engine::Json(nullptr)},
            {"state", "open"}, {"tops_id", tops_id}, {"subject_id", subject_id},
            {"repository_root", repository_root}, {"session_journal_digest", payload.at("session_journal_digest")},
            {"binding_registry_digest", payload.at("binding_registry_digest")},
            {"baseline_snapshot", snapshot}, {"baseline_engine", engine_evidence},
            {"current_snapshot_digest", snapshot.at("snapshot_digest")},
            {"current_engine", engine_evidence},
            {"maestro_inventory_digest", payload.at("maestro_inventory").at("evidence_digest")},
            {"review_state", "current"}, {"checkpoints", engine::Json::array({std::move(first)})},
            {"compatibility", journal_compatibility()}, {"extensions", engine::Json::array()},
            {"recovery", clean_recovery()}, {"started_at", now}, {"updated_at", now},
            {"closed_at", nullptr}, {"canonical", false},
        };
    } else {
        next = state.journal;
        if (next.at("checkpoints").size() >= max_checkpoints) {
            throw engine::Error("ssfv_maintenance.checkpoint_limit", "maintenance checkpoint limit reached", 4);
        }
        auto next_checkpoint = checkpoint(close ? "close" : "checkpoint", operation_id, fingerprint,
            payload, engine_evidence, snapshot, &payload.at("diff_result"),
            next.at("checkpoints").size() + 1U, next.at("checkpoints").back().at("checkpoint_digest"));
        next["generation"] = number(next, "generation") + 1U;
        next["previous_journal_digest"] = state.journal.at("journal_digest");
        next["session_journal_digest"] = payload.at("session_journal_digest");
        next["binding_registry_digest"] = payload.at("binding_registry_digest");
        next["current_snapshot_digest"] = snapshot.at("snapshot_digest");
        next["current_engine"] = engine_evidence;
        next["maestro_inventory_digest"] = payload.at("maestro_inventory").at("evidence_digest");
        next["review_state"] = review_state(payload.at("diff_result"));
        next["checkpoints"].push_back(std::move(next_checkpoint));
        next["updated_at"] = utc_now();
        next["recovery"] = clean_recovery();
        if (close) {
            next["state"] = "closed";
            next["closed_at"] = next.at("updated_at");
        }
    }
    auto committed = commit(stream->directory_fd(), std::move(next),
                            state.present ? std::optional<engine::Json>(state.head) : std::nullopt);
    return make_result(request.operation, compatibility_result(payload.at("client"), &committed.journal),
                       committed, committed.journal.at("state").get<std::string>(), true, false,
                       engine::Json::array(), false);
}

} // namespace symphony::knowledge::session
