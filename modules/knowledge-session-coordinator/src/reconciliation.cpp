#include "reconciliation.hpp"

#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/limits.hpp"
#include "symphony/knowledge/engine/path.hpp"

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
#include <map>
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

constexpr const char* journal_protocol = "symphony.knowledge.reconciliation-journal.v1";
constexpr const char* head_protocol = "symphony.knowledge.reconciliation-head.v1";
constexpr const char* command_protocol = "symphony.knowledge.reconciliation-command.v1";
constexpr const char* result_protocol = "symphony.knowledge.reconciliation-result.v1";
constexpr std::uint64_t format_version = 1U;
constexpr std::size_t max_state_bytes = engine::Limits::max_response_bytes;
constexpr std::size_t max_checkpoints = 256U;

const std::vector<std::string> required_capabilities = {
    "atomic-head-v1",
    "content-snapshot-v1",
    "dual-slot-journal-v1",
    "expected-state-cas-v1",
    "idempotent-operation-v1",
    "opaque-extension-preservation-v1",
    "recovery-forward-v1",
};

const std::vector<std::string> optional_capabilities = {
    "discovery-recovery-v1",
    "nonblocking-lock-v1",
};

class FileDescriptor final {
public:
    explicit FileDescriptor(int value = -1) : value_(value) {}
    ~FileDescriptor() {
        if (value_ >= 0) {
            ::close(value_);
        }
    }
    FileDescriptor(const FileDescriptor&) = delete;
    FileDescriptor& operator=(const FileDescriptor&) = delete;
    FileDescriptor(FileDescriptor&& other) noexcept : value_(std::exchange(other.value_, -1)) {}
    FileDescriptor& operator=(FileDescriptor&& other) noexcept {
        if (this != &other) {
            if (value_ >= 0) {
                ::close(value_);
            }
            value_ = std::exchange(other.value_, -1);
        }
        return *this;
    }
    [[nodiscard]] int get() const noexcept { return value_; }

private:
    int value_;
};

class ContextLock final {
public:
    ContextLock(FileDescriptor directory, FileDescriptor lock)
        : directory_(std::move(directory)), lock_(std::move(lock)) {}
    ~ContextLock() {
        if (lock_.get() >= 0) {
            static_cast<void>(::flock(lock_.get(), LOCK_UN));
        }
    }
    ContextLock(const ContextLock&) = delete;
    ContextLock& operator=(const ContextLock&) = delete;
    ContextLock(ContextLock&&) = default;
    ContextLock& operator=(ContextLock&&) = default;
    [[nodiscard]] int directory_fd() const noexcept { return directory_.get(); }

private:
    FileDescriptor directory_;
    FileDescriptor lock_;
};

struct LoadedState final {
    engine::Json head;
    engine::Json journal;
    bool present = false;
};

struct Candidate final {
    int slot = -1;
    engine::Json journal;
    bool valid = false;
    bool exists = false;
    bool incompatible = false;
    std::string invalid_reason;
};

[[noreturn]] void system_error(const std::string& code, const std::string& detail) {
    const int saved = errno;
    throw engine::Error(code, detail + ": " + std::strerror(saved), 5);
}

bool safe_token(std::string_view value, std::size_t maximum = 256U) {
    if (value.empty() || value.size() > maximum) {
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
    return value.size() == 71U && value.starts_with("sha256:") &&
           std::all_of(value.begin() + 7, value.end(), [](const unsigned char character) {
               return (character >= '0' && character <= '9') ||
                      (character >= 'a' && character <= 'f');
           });
}

bool safe_version(std::string_view value) {
    if (value.empty() || value.size() > 64U) {
        return false;
    }
    return std::all_of(value.begin(), value.end(), [](const unsigned char character) {
        const bool alphanumeric =
            (character >= 'a' && character <= 'z') ||
            (character >= 'A' && character <= 'Z') ||
            (character >= '0' && character <= '9');
        return alphanumeric || character == '.' || character == '+' || character == '-';
    });
}

bool safe_absolute_path(const std::string& value) {
    if (value.empty() || value.size() > engine::Limits::max_path_bytes ||
        value.front() != '/' || (value.size() > 1U && value.back() == '/') ||
        value.find('\\') != std::string::npos || value.find('\0') != std::string::npos) {
        return false;
    }
    for (const unsigned char character : value) {
        if (character < 0x20U || character == 0x7fU) {
            return false;
        }
    }
    const fs::path path(value);
    if (!path.is_absolute() || path.lexically_normal().string() != value) {
        return false;
    }
    for (const auto& component_value : path.relative_path()) {
        const auto component = component_value.string();
        if (component.empty() || component == "." || component == "..") {
            return false;
        }
    }
    return true;
}

void require_exact_fields(const engine::Json& object, const std::set<std::string>& fields) {
    if (!object.is_object() || object.size() != fields.size()) {
        throw engine::Error(
            "reconcile.field_set",
            "reconciliation object is incomplete or contains unknown fields",
            4);
    }
    for (const auto& [key, value] : object.items()) {
        static_cast<void>(value);
        if (!fields.contains(key)) {
            throw engine::Error(
                "reconcile.unknown_field",
                "reconciliation object contains an unknown field",
                4);
        }
    }
}

std::string required_string(
    const engine::Json& object,
    const char* field,
    std::size_t maximum = engine::Limits::max_string_bytes) {
    const auto& value = object.at(field);
    if (!value.is_string()) {
        throw engine::Error("reconcile.invalid_field", std::string(field) + " must be a string", 4);
    }
    const auto text = value.get<std::string>();
    if (text.empty() || text.size() > maximum) {
        throw engine::Error("reconcile.invalid_field", std::string(field) + " has invalid length", 4);
    }
    for (const unsigned char character : text) {
        if (character < 0x20U || character == 0x7fU) {
            throw engine::Error(
                "reconcile.invalid_field",
                std::string(field) + " contains unsafe text",
                4);
        }
    }
    return text;
}

std::uint64_t required_unsigned(const engine::Json& object, const char* field) {
    const auto& value = object.at(field);
    if (!value.is_number_unsigned() && !value.is_number_integer()) {
        throw engine::Error("reconcile.invalid_field", std::string(field) + " must be an integer", 4);
    }
    try {
        const auto result = value.get<std::uint64_t>();
        if (result > 9007199254740991ULL) {
            throw engine::Error("reconcile.invalid_field", std::string(field) + " is out of range", 4);
        }
        return result;
    } catch (const nlohmann::json::exception&) {
        throw engine::Error("reconcile.invalid_field", std::string(field) + " is out of range", 4);
    }
}

std::string utc_now() {
    const auto now = std::chrono::system_clock::now();
    const auto seconds = std::chrono::time_point_cast<std::chrono::seconds>(now);
    const std::time_t value = std::chrono::system_clock::to_time_t(seconds);
    std::tm result {};
    if (::gmtime_r(&value, &result) == nullptr) {
        throw engine::Error("reconcile.clock_failed", "could not obtain a UTC timestamp", 5);
    }
    std::ostringstream output;
    output << std::put_time(&result, "%Y-%m-%dT%H:%M:%SZ");
    return output.str();
}

bool strict_utc(std::string_view value) {
    if (value.size() != 20U || value[4] != '-' || value[7] != '-' ||
        value[10] != 'T' || value[13] != ':' || value[16] != ':' ||
        value[19] != 'Z') {
        return false;
    }
    for (const std::size_t index : {
             0U, 1U, 2U, 3U, 5U, 6U, 8U, 9U,
             11U, 12U, 14U, 15U, 17U, 18U}) {
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
    return date.ok() && number(11, 2) <= 23 &&
           number(14, 2) <= 59 && number(17, 2) <= 59;
}

std::vector<std::string> string_array(
    const engine::Json& value,
    const std::string& name,
    std::size_t maximum,
    bool tokens) {
    if (!value.is_array() || value.size() > maximum) {
        throw engine::Error("reconcile.invalid_field", name + " must be a bounded array", 4);
    }
    std::vector<std::string> result;
    std::set<std::string> seen;
    for (const auto& item : value) {
        if (!item.is_string()) {
            throw engine::Error("reconcile.invalid_field", name + " contains a non-string", 4);
        }
        const auto text = item.get<std::string>();
        if ((tokens && !safe_token(text)) ||
            (!tokens && !engine::is_safe_relative_path(text)) ||
            !seen.insert(text).second) {
            throw engine::Error("reconcile.invalid_field", name + " contains an unsafe or duplicate value", 4);
        }
        result.push_back(text);
    }
    return result;
}

std::vector<std::uint64_t> version_array(
    const engine::Json& value,
    const std::string& name) {
    if (!value.is_array() || value.empty() || value.size() > 8U) {
        throw engine::Error("reconcile.invalid_field", name + " must be a bounded nonempty array", 4);
    }
    std::vector<std::uint64_t> result;
    std::set<std::uint64_t> seen;
    for (const auto& item : value) {
        if ((!item.is_number_integer() && !item.is_number_unsigned())) {
            throw engine::Error("reconcile.invalid_field", name + " contains a non-integer", 4);
        }
        const auto number = item.get<std::uint64_t>();
        if (number == 0U || number > 1024U || !seen.insert(number).second) {
            throw engine::Error("reconcile.invalid_field", name + " contains an invalid version", 4);
        }
        result.push_back(number);
    }
    return result;
}

bool contains(const std::vector<std::string>& values, const std::string& wanted) {
    return std::find(values.begin(), values.end(), wanted) != values.end();
}

bool contains(const std::vector<std::uint64_t>& values, std::uint64_t wanted) {
    return std::find(values.begin(), values.end(), wanted) != values.end();
}

engine::Json compatibility_result(const engine::Json& client, const engine::Json* journal) {
    require_exact_fields(client, {
        "client_id", "client_version", "process_protocols", "journal_read_versions",
        "journal_write_versions", "capabilities",
    });
    if (required_string(client, "client_id", 128U) != "qxctl" ||
        !safe_version(required_string(client, "client_version", 64U))) {
        throw engine::Error("reconcile.client_invalid", "reconciliation client identity is invalid", 4);
    }
    const auto process = string_array(client.at("process_protocols"), "process_protocols", 8U, true);
    const auto reads = version_array(client.at("journal_read_versions"), "journal_read_versions");
    const auto writes = version_array(client.at("journal_write_versions"), "journal_write_versions");
    const auto capabilities = string_array(client.at("capabilities"), "capabilities", 64U, true);

    const bool process_ok = contains(process, engine::process_protocol_v1);
    const bool read_ok = contains(reads, format_version);
    const bool write_ok = contains(writes, format_version);
    std::vector<std::string> shared;
    std::vector<std::string> missing;
    for (const auto& capability : required_capabilities) {
        if (contains(capabilities, capability)) {
            shared.push_back(capability);
        } else {
            missing.push_back(capability);
        }
    }
    for (const auto& capability : optional_capabilities) {
        if (contains(capabilities, capability)) {
            shared.push_back(capability);
        }
    }
    std::sort(shared.begin(), shared.end());
    std::sort(missing.begin(), missing.end());

    auto reasons = engine::Json::array();
    std::string mode = "full";
    if (!process_ok || !read_ok) {
        mode = "unsupported";
        reasons.push_back("client and coordinator have no safe process/journal read overlap");
    } else if (!write_ok || !missing.empty()) {
        mode = "read_only";
        reasons.push_back("read overlap exists but write version or required capabilities are missing");
    }
    if (journal != nullptr) {
        const auto journal_version = required_unsigned(*journal, "format_version");
        if (journal_version != format_version) {
            mode = read_ok ? "migration_required" : "unsupported";
            reasons.push_back("stored journal format is outside the coordinator write window");
        }
    }
    if (reasons.empty()) {
        reasons.push_back("client, coordinator, and stored journal share the full v1 capability set");
    }
    return engine::Json{
        {"mode", mode},
        {"process_protocol", process_ok ? engine::Json(engine::process_protocol_v1) : engine::Json(nullptr)},
        {"journal_read_version", read_ok ? engine::Json(format_version) : engine::Json(nullptr)},
        {"journal_write_version", write_ok ? engine::Json(format_version) : engine::Json(nullptr)},
        {"shared_capabilities", shared},
        {"missing_capabilities", missing},
        {"reasons", reasons},
    };
}

engine::Json command_compatibility() {
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

engine::Json snapshot_json(const engine::Snapshot& snapshot) {
    auto files = engine::Json::array();
    for (const auto& file : snapshot.files) {
        files.push_back(engine::Json{
            {"path", file.path},
            {"size", file.size},
            {"digest", file.digest},
        });
    }
    return engine::Json{{"digest", snapshot.digest}, {"files", std::move(files)}};
}

std::size_t changed_file_count(const engine::Json& before, const engine::Json& after) {
    std::map<std::string, std::string> previous;
    for (const auto& file : before.at("files")) {
        previous[file.at("path").get<std::string>()] = file.at("digest").get<std::string>();
    }
    std::size_t changed = 0;
    for (const auto& file : after.at("files")) {
        const auto path = file.at("path").get<std::string>();
        const auto digest = file.at("digest").get<std::string>();
        const auto found = previous.find(path);
        if (found == previous.end() || found->second != digest) {
            ++changed;
        }
        if (found != previous.end()) {
            previous.erase(found);
        }
    }
    return changed + previous.size();
}

std::string finalize_digest(engine::Json& value, const char* field) {
    value.erase(field);
    const auto digest = engine::tagged_sha256(value.dump());
    value[field] = digest;
    return digest;
}

engine::Json make_checkpoint(
    std::uint64_t sequence,
    const std::string& kind,
    const std::string& operation_id,
    const std::string& observed_at,
    const std::string& snapshot_digest,
    std::size_t changed,
    const std::string& engine_inventory_digest,
    const engine::Json& previous_digest) {
    engine::Json checkpoint{
        {"sequence", sequence},
        {"kind", kind},
        {"operation_id", operation_id},
        {"observed_at", observed_at},
        {"snapshot_digest", snapshot_digest},
        {"changed_file_count", changed},
        {"engine_inventory_digest", engine_inventory_digest},
        {"previous_checkpoint_digest", previous_digest},
    };
    checkpoint["checkpoint_digest"] = engine::tagged_sha256(checkpoint.dump());
    return checkpoint;
}

void validate_checkpoint(const engine::Json& checkpoint, std::uint64_t expected_sequence) {
    require_exact_fields(checkpoint, {
        "sequence", "kind", "operation_id", "observed_at", "snapshot_digest",
        "changed_file_count", "engine_inventory_digest",
        "previous_checkpoint_digest", "checkpoint_digest",
    });
    if (required_unsigned(checkpoint, "sequence") != expected_sequence ||
        !safe_token(required_string(checkpoint, "operation_id", 256U)) ||
        !strict_utc(required_string(checkpoint, "observed_at", 20U)) ||
        !tagged_digest(required_string(checkpoint, "snapshot_digest", 71U)) ||
        !tagged_digest(required_string(checkpoint, "engine_inventory_digest", 71U)) ||
        !tagged_digest(required_string(checkpoint, "checkpoint_digest", 71U))) {
        throw engine::Error("reconcile.journal_invalid", "checkpoint identity is invalid", 5);
    }
    if (required_unsigned(checkpoint, "changed_file_count") >
        engine::Limits::max_snapshot_files) {
        throw engine::Error("reconcile.journal_invalid", "checkpoint change count is invalid", 5);
    }
    const auto& previous = checkpoint.at("previous_checkpoint_digest");
    if (!previous.is_null() &&
        (!previous.is_string() ||
         !tagged_digest(previous.get<std::string>()))) {
        throw engine::Error("reconcile.journal_invalid", "checkpoint predecessor is invalid", 5);
    }
    const auto kind = required_string(checkpoint, "kind", 32U);
    if (kind != "begin" && kind != "checkpoint" && kind != "close" && kind != "recover") {
        throw engine::Error("reconcile.journal_invalid", "checkpoint kind is invalid", 5);
    }
    auto canonical = checkpoint;
    const auto digest = canonical.at("checkpoint_digest").get<std::string>();
    canonical.erase("checkpoint_digest");
    if (digest != engine::tagged_sha256(canonical.dump())) {
        throw engine::Error("reconcile.journal_invalid", "checkpoint digest mismatch", 5);
    }
}

void validate_snapshot(const engine::Json& snapshot) {
    require_exact_fields(snapshot, {"digest", "files"});
    if (!tagged_digest(required_string(snapshot, "digest", 71U)) ||
        !snapshot.at("files").is_array() || snapshot.at("files").empty() ||
        snapshot.at("files").size() > engine::Limits::max_snapshot_files) {
        throw engine::Error("reconcile.journal_invalid", "snapshot is invalid", 5);
    }
    std::string previous;
    std::ostringstream canonical;
    for (const auto& file : snapshot.at("files")) {
        require_exact_fields(file, {"path", "size", "digest"});
        const auto path = required_string(file, "path", engine::Limits::max_path_bytes);
        if (!engine::is_safe_relative_path(path) || (!previous.empty() && path <= previous) ||
            !tagged_digest(required_string(file, "digest", 71U))) {
            throw engine::Error("reconcile.journal_invalid", "snapshot file evidence is invalid", 5);
        }
        const auto size = required_unsigned(file, "size");
        const auto digest = file.at("digest").get<std::string>();
        canonical << path.size() << ':' << path << ':' << size << ':' << digest << '\n';
        previous = path;
    }
    if (snapshot.at("digest") != engine::tagged_sha256(canonical.str())) {
        throw engine::Error("reconcile.journal_invalid", "snapshot aggregate digest is invalid", 5);
    }
}

engine::Json engine_inventory_from_command(const engine::Json& payload) {
    if (!tagged_digest(required_string(payload, "binding_registry_digest", 71U)) ||
        !payload.at("engine_inventory").is_array() ||
        payload.at("engine_inventory").empty() ||
        payload.at("engine_inventory").size() > 6U) {
        throw engine::Error("reconcile.inventory_invalid", "engine inventory is absent or invalid", 4);
    }
    auto entries = engine::Json::array();
    std::string previous_role;
    bool coordinator_present = false;
    static const std::map<std::string, std::pair<std::string, std::string>> identities = {
        {"coordinator", {"knowledge-session-coordinator", "symphony-knowledge-session"}},
        {"sacv", {"sacv-engine", "symphony-sacv"}},
        {"sclv", {"sclv-engine", "symphony-sclv"}},
        {"skvi", {"skvi-engine", "symphony-skvi"}},
        {"sodv", {"sodv-engine", "symphony-sodv"}},
        {"ssfv", {"ssfv-engine", "symphony-ssfv"}},
    };
    for (const auto& entry : payload.at("engine_inventory")) {
        require_exact_fields(entry, {
            "role", "module_id", "engine_id", "version", "receipt_digest",
            "executable_digest",
        });
        const auto role = required_string(entry, "role", 32U);
        const auto identity = identities.find(role);
        if (identity == identities.end() ||
            (!previous_role.empty() && role <= previous_role) ||
            required_string(entry, "module_id", 128U) != identity->second.first ||
            required_string(entry, "engine_id", 128U) != identity->second.second ||
            !safe_version(required_string(entry, "version", 64U)) ||
            !tagged_digest(required_string(entry, "receipt_digest", 71U)) ||
            !tagged_digest(required_string(entry, "executable_digest", 71U))) {
            throw engine::Error(
                "reconcile.inventory_invalid",
                "engine inventory entries must be safe, unique, and role-sorted",
                4);
        }
        coordinator_present = coordinator_present || role == "coordinator";
        previous_role = role;
        entries.push_back(entry);
    }
    if (!coordinator_present) {
        throw engine::Error(
            "reconcile.inventory_invalid",
            "engine inventory must contain the exact bound coordinator",
            4);
    }
    engine::Json inventory{
        {"binding_registry_digest", payload.at("binding_registry_digest")},
        {"entries", entries},
    };
    inventory["digest"] = engine::tagged_sha256(inventory.dump());
    return inventory;
}

void validate_engine_inventory(const engine::Json& inventory) {
    require_exact_fields(inventory, {"binding_registry_digest", "entries", "digest"});
    engine::Json payload{
        {"binding_registry_digest", inventory.at("binding_registry_digest")},
        {"engine_inventory", inventory.at("entries")},
    };
    const auto normalized = engine_inventory_from_command(payload);
    if (normalized != inventory) {
        throw engine::Error("reconcile.journal_invalid", "engine inventory digest is invalid", 5);
    }
}

void validate_journal(const engine::Json& journal) {
    require_exact_fields(journal, {
        "protocol", "format_version", "journal_id", "context_id", "generation",
        "previous_journal_digest", "state", "repository", "inventory",
        "engine_inventory", "current_snapshot", "checkpoints", "compatibility", "extensions",
        "recovery", "started_at", "updated_at", "closed_at", "canonical",
        "journal_digest",
    });
    if (required_string(journal, "protocol") != journal_protocol ||
        required_unsigned(journal, "format_version") != format_version ||
        !safe_token(required_string(journal, "journal_id", 256U)) ||
        !safe_token(required_string(journal, "context_id", 256U)) ||
        required_unsigned(journal, "generation") == 0U ||
        journal.at("canonical") != false ||
        !tagged_digest(required_string(journal, "journal_digest", 71U))) {
        throw engine::Error("reconcile.journal_invalid", "journal identity is invalid", 5);
    }
    const auto state = required_string(journal, "state", 32U);
    if (state != "open" && state != "closed") {
        throw engine::Error("reconcile.journal_invalid", "journal state is invalid", 5);
    }
    if (state == "open" && !journal.at("closed_at").is_null()) {
        throw engine::Error("reconcile.journal_invalid", "open journal has a close timestamp", 5);
    }
    if (state == "closed" && !journal.at("closed_at").is_string()) {
        throw engine::Error("reconcile.journal_invalid", "closed journal lacks a close timestamp", 5);
    }
    if (!strict_utc(required_string(journal, "started_at", 20U)) ||
        !strict_utc(required_string(journal, "updated_at", 20U)) ||
        (journal.at("closed_at").is_string() &&
         !strict_utc(required_string(journal, "closed_at", 20U)))) {
        throw engine::Error("reconcile.journal_invalid", "journal timestamp is invalid", 5);
    }
    const auto started_at = journal.at("started_at").get<std::string>();
    const auto updated_at = journal.at("updated_at").get<std::string>();
    if (started_at > updated_at ||
        (journal.at("closed_at").is_string() &&
         (journal.at("closed_at").get<std::string>() < started_at ||
          journal.at("closed_at").get<std::string>() > updated_at))) {
        throw engine::Error("reconcile.journal_invalid", "journal timestamp order is invalid", 5);
    }
    const auto& previous_journal = journal.at("previous_journal_digest");
    if (!previous_journal.is_null() &&
        (!previous_journal.is_string() ||
         !tagged_digest(previous_journal.get<std::string>()))) {
        throw engine::Error("reconcile.journal_invalid", "journal predecessor is invalid", 5);
    }
    const auto journal_generation = required_unsigned(journal, "generation");
    if ((journal_generation == 1U) != previous_journal.is_null()) {
        throw engine::Error(
            "reconcile.journal_invalid",
            "journal generation and predecessor are inconsistent",
            5);
    }
    const auto& repository = journal.at("repository");
    require_exact_fields(repository, {"repository_root", "repository_key", "worktree_key"});
    const auto repository_root =
        required_string(repository, "repository_root", engine::Limits::max_path_bytes);
    if (!safe_absolute_path(repository_root) ||
        repository.at("repository_key") !=
            engine::tagged_sha256("repository-root:" + repository_root) ||
        repository.at("worktree_key") !=
            engine::tagged_sha256("worktree-root:" + repository_root)) {
        throw engine::Error("reconcile.journal_invalid", "repository identity is invalid", 5);
    }
    const auto& inventory = journal.at("inventory");
    require_exact_fields(inventory, {"source", "paths", "digest"});
    if (required_string(inventory, "source") != "caller_declared" ||
        !tagged_digest(required_string(inventory, "digest", 71U))) {
        throw engine::Error("reconcile.journal_invalid", "inventory identity is invalid", 5);
    }
    const auto paths = string_array(
        inventory.at("paths"), "inventory.paths", engine::Limits::max_snapshot_files, false);
    if (paths.empty()) {
        throw engine::Error("reconcile.journal_invalid", "inventory is empty", 5);
    }
    auto sorted_paths = paths;
    std::sort(sorted_paths.begin(), sorted_paths.end());
    if (paths != sorted_paths ||
        inventory.at("digest") != engine::tagged_sha256(engine::Json(paths).dump())) {
        throw engine::Error("reconcile.journal_invalid", "inventory digest or ordering is invalid", 5);
    }
    validate_snapshot(journal.at("current_snapshot"));
    validate_engine_inventory(journal.at("engine_inventory"));

    const auto& checkpoints = journal.at("checkpoints");
    if (!checkpoints.is_array() || checkpoints.empty() || checkpoints.size() > max_checkpoints) {
        throw engine::Error("reconcile.journal_invalid", "checkpoint history is invalid", 5);
    }
    engine::Json previous_checkpoint = nullptr;
    std::string previous_observation;
    std::set<std::string> operation_ids;
    bool close_seen = false;
    for (std::size_t index = 0; index < checkpoints.size(); ++index) {
        validate_checkpoint(checkpoints[index], index + 1U);
        if (checkpoints[index].at("previous_checkpoint_digest") != previous_checkpoint) {
            throw engine::Error("reconcile.journal_invalid", "checkpoint chain is discontinuous", 5);
        }
        const auto checkpoint_kind =
            checkpoints[index].at("kind").get<std::string>();
        const auto checkpoint_operation =
            checkpoints[index].at("operation_id").get<std::string>();
        if ((index == 0U && checkpoint_kind != "begin") ||
            (index > 0U && checkpoint_kind == "begin") ||
            !operation_ids.insert(checkpoint_operation).second ||
            (close_seen && checkpoint_kind != "recover")) {
            throw engine::Error(
                "reconcile.journal_invalid",
                "checkpoint lifecycle or operation identity is invalid",
                5);
        }
        if (checkpoint_kind == "close") {
            if (close_seen) {
                throw engine::Error(
                    "reconcile.journal_invalid",
                    "checkpoint history contains more than one close",
                    5);
            }
            close_seen = true;
        }
        const auto observation =
            checkpoints[index].at("observed_at").get<std::string>();
        if ((!previous_observation.empty() && observation < previous_observation) ||
            observation < started_at || observation > updated_at) {
            throw engine::Error(
                "reconcile.journal_invalid",
                "checkpoint timestamp order is invalid",
                5);
        }
        previous_observation = observation;
        previous_checkpoint = checkpoints[index].at("checkpoint_digest");
    }
    if ((state == "closed") != close_seen) {
        throw engine::Error(
            "reconcile.journal_invalid",
            "journal state and checkpoint lifecycle are inconsistent",
            5);
    }
    if (checkpoints.back().at("snapshot_digest") !=
            journal.at("current_snapshot").at("digest") ||
        checkpoints.back().at("engine_inventory_digest") !=
            journal.at("engine_inventory").at("digest")) {
        throw engine::Error(
            "reconcile.journal_invalid",
            "current snapshot or engine inventory is not checkpointed",
            5);
    }

    const auto& compatibility = journal.at("compatibility");
    require_exact_fields(compatibility, {
        "process_protocols", "journal_read_versions", "journal_write_version",
        "minimum_reader_version", "required_capabilities", "optional_capabilities",
        "opaque_extensions_preserved",
    });
    if (compatibility.at("journal_write_version") != format_version ||
        compatibility.at("minimum_reader_version") != format_version ||
        compatibility.at("opaque_extensions_preserved") != true) {
        throw engine::Error("reconcile.journal_incompatible", "journal compatibility window is unsupported", 4);
    }
    const auto journal_required = string_array(
        compatibility.at("required_capabilities"), "required_capabilities", 32U, true);
    const auto journal_optional = string_array(
        compatibility.at("optional_capabilities"), "optional_capabilities", 32U, true);
    const auto journal_process = string_array(
        compatibility.at("process_protocols"), "process_protocols", 8U, true);
    const auto journal_reads = version_array(
        compatibility.at("journal_read_versions"), "journal_read_versions");
    if (!contains(journal_process, engine::process_protocol_v1) ||
        !contains(journal_reads, format_version) ||
        journal_required != required_capabilities ||
        journal_optional != optional_capabilities) {
        throw engine::Error(
            "reconcile.journal_incompatible",
            "journal compatibility declaration is not the supported v1 contract",
            4);
    }
    for (const auto& capability : journal_required) {
        if (!contains(required_capabilities, capability)) {
            throw engine::Error(
                "reconcile.critical_capability_unknown",
                "journal requires an unsupported critical capability",
                4);
        }
    }

    if (!journal.at("extensions").is_array() || journal.at("extensions").size() > 64U) {
        throw engine::Error("reconcile.journal_invalid", "extension collection is invalid", 5);
    }
    std::set<std::string> extension_ids;
    for (const auto& extension : journal.at("extensions")) {
        require_exact_fields(extension, {
            "extension_id", "extension_version", "critical", "payload", "payload_digest",
        });
        const auto extension_id = required_string(extension, "extension_id", 256U);
        if (!safe_token(extension_id) || !extension_ids.insert(extension_id).second ||
            !safe_version(required_string(extension, "extension_version", 64U)) ||
            !extension.at("critical").is_boolean() ||
            !tagged_digest(required_string(extension, "payload_digest", 71U)) ||
            extension.at("payload_digest") != engine::tagged_sha256(extension.at("payload").dump())) {
            throw engine::Error("reconcile.journal_invalid", "extension envelope is invalid", 5);
        }
        if (extension.at("critical").get<bool>()) {
            throw engine::Error(
                "reconcile.critical_extension_unknown",
                "journal contains an unknown critical extension",
                4);
        }
    }
    const auto& recovery = journal.at("recovery");
    require_exact_fields(recovery, {
        "state", "last_recovery_at", "disposition", "recovered_from_digest", "detail",
    });
    const auto recovery_state = required_string(recovery, "state", 32U);
    const auto disposition = required_string(recovery, "disposition", 64U);
    const std::set<std::string> dispositions = {
        "not_applicable", "adopted_linked_successor", "repaired_head",
        "rolled_forward_from_valid_slot",
    };
    if ((recovery_state != "clean" && recovery_state != "recovered") ||
        !dispositions.contains(disposition) ||
        required_string(recovery, "detail", 1024U).empty()) {
        throw engine::Error("reconcile.journal_invalid", "recovery state is invalid", 5);
    }
    const auto& recovery_at = recovery.at("last_recovery_at");
    const auto& recovered_from = recovery.at("recovered_from_digest");
    if ((recovery_at.is_string() &&
         !strict_utc(recovery_at.get<std::string>())) ||
        (!recovery_at.is_null() && !recovery_at.is_string()) ||
        (recovered_from.is_string() &&
         !tagged_digest(recovered_from.get<std::string>())) ||
        (!recovered_from.is_null() && !recovered_from.is_string()) ||
        (recovery_state == "clean" &&
         (!recovery_at.is_null() || disposition != "not_applicable" ||
          !recovered_from.is_null())) ||
        (recovery_state == "recovered" &&
         (!recovery_at.is_string() || disposition == "not_applicable" ||
          !recovered_from.is_string()))) {
        throw engine::Error("reconcile.journal_invalid", "recovery evidence is inconsistent", 5);
    }

    auto canonical = journal;
    const auto digest = canonical.at("journal_digest").get<std::string>();
    canonical.erase("journal_digest");
    if (digest != engine::tagged_sha256(canonical.dump())) {
        throw engine::Error("reconcile.journal_invalid", "journal digest mismatch", 5);
    }
}

void validate_head(const engine::Json& head) {
    require_exact_fields(head, {
        "protocol", "format_version", "context_id", "worktree_key", "active_slot",
        "generation", "journal_digest", "previous_head_digest", "updated_at", "head_digest",
    });
    if (required_string(head, "protocol") != head_protocol ||
        required_unsigned(head, "format_version") != format_version ||
        !safe_token(required_string(head, "context_id", 256U)) ||
        !tagged_digest(required_string(head, "worktree_key", 71U)) ||
        !tagged_digest(required_string(head, "journal_digest", 71U)) ||
        !tagged_digest(required_string(head, "head_digest", 71U)) ||
        required_unsigned(head, "generation") == 0U ||
        !strict_utc(required_string(head, "updated_at", 20U))) {
        throw engine::Error("reconcile.head_invalid", "reconciliation head identity is invalid", 5);
    }
    const auto slot = required_unsigned(head, "active_slot");
    if (slot > 1U) {
        throw engine::Error("reconcile.head_invalid", "reconciliation head slot is invalid", 5);
    }
    const auto& previous_head = head.at("previous_head_digest");
    if (!previous_head.is_null() &&
        (!previous_head.is_string() ||
         !tagged_digest(previous_head.get<std::string>()))) {
        throw engine::Error("reconcile.head_invalid", "reconciliation head predecessor is invalid", 5);
    }
    auto canonical = head;
    const auto digest = canonical.at("head_digest").get<std::string>();
    canonical.erase("head_digest");
    if (digest != engine::tagged_sha256(canonical.dump())) {
        throw engine::Error("reconcile.head_invalid", "reconciliation head digest mismatch", 5);
    }
}

void validate_owned_directory(int fd, bool managed) {
    struct stat status {};
    if (::fstat(fd, &status) != 0) {
        system_error("reconcile.state_stat_failed", "could not inspect state directory");
    }
    if (!S_ISDIR(status.st_mode) || status.st_uid != ::geteuid() || (status.st_mode & 0022) != 0) {
        throw engine::Error(
            "reconcile.state_directory_unsafe",
            "state directory must be caller-owned and not writable by group or other",
            5);
    }
    if (managed && ::fchmod(fd, 0700) != 0) {
        system_error("reconcile.state_mode_failed", "could not restrict managed state directory");
    }
}

void validate_owned_regular(int fd) {
    struct stat status {};
    if (::fstat(fd, &status) != 0) {
        system_error("reconcile.state_stat_failed", "could not inspect state file");
    }
    if (!S_ISREG(status.st_mode) || status.st_uid != ::geteuid() ||
        (status.st_mode & 0077) != 0 || status.st_nlink != 1) {
        throw engine::Error(
            "reconcile.state_file_unsafe",
            "state file must be a private caller-owned single-link regular file",
            5);
    }
}

FileDescriptor open_absolute_directory(const std::string& path, bool create) {
    if (!safe_absolute_path(path) || path == "/") {
        throw engine::Error("reconcile.state_root_invalid", "state root must be an absolute descendant path", 4);
    }
    FileDescriptor current(::open("/", O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC));
    if (current.get() < 0) {
        system_error("reconcile.state_open_failed", "could not open filesystem root");
    }
    for (const auto& component_value : fs::path(path).relative_path()) {
        const auto component = component_value.string();
        int next = ::openat(
            current.get(), component.c_str(), O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
        if (next < 0 && errno == ENOENT && create) {
            if (::mkdirat(current.get(), component.c_str(), 0700) != 0 && errno != EEXIST) {
                system_error("reconcile.state_create_failed", "could not create state root component");
            }
            if (::fsync(current.get()) != 0) {
                system_error("reconcile.state_sync_failed", "could not synchronize state root component");
            }
            next = ::openat(
                current.get(), component.c_str(), O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
        }
        if (next < 0) {
            system_error("reconcile.state_open_failed", "could not safely open state root component");
        }
        current = FileDescriptor(next);
    }
    validate_owned_directory(current.get(), false);
    return current;
}

FileDescriptor open_managed_directory(int parent, const std::string& name) {
    int next = ::openat(parent, name.c_str(), O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
    if (next < 0 && errno == ENOENT) {
        if (::mkdirat(parent, name.c_str(), 0700) != 0 && errno != EEXIST) {
            system_error("reconcile.state_create_failed", "could not create managed state directory");
        }
        if (::fsync(parent) != 0) {
            system_error("reconcile.state_sync_failed", "could not synchronize managed state directory");
        }
        next = ::openat(parent, name.c_str(), O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
    }
    if (next < 0) {
        system_error("reconcile.state_open_failed", "could not safely open managed state directory");
    }
    FileDescriptor result(next);
    validate_owned_directory(result.get(), true);
    return result;
}

ContextLock open_context(
    const std::string& state_root,
    const std::string& worktree_hex,
    bool exclusive) {
    FileDescriptor current = open_absolute_directory(state_root, true);
    const std::array<std::string, 6> components = {
        "symphony", "knowledge-session-coordinator", "reconciliation", "v1", "contexts",
        worktree_hex,
    };
    for (const auto& component : components) {
        current = open_managed_directory(current.get(), component);
    }
    int lock_fd = ::openat(
        current.get(), "journal.lock",
        O_CREAT | O_RDWR | O_NOFOLLOW | O_CLOEXEC,
        0600);
    if (lock_fd < 0) {
        system_error("reconcile.lock_open_failed", "could not open reconciliation lock");
    }
    FileDescriptor lock(lock_fd);
    validate_owned_regular(lock.get());
    if (::fchmod(lock.get(), 0600) != 0) {
        system_error("reconcile.state_mode_failed", "could not restrict reconciliation lock");
    }
    const int operation = (exclusive ? LOCK_EX : LOCK_SH) | LOCK_NB;
    if (::flock(lock.get(), operation) != 0) {
        if (errno == EWOULDBLOCK || errno == EAGAIN) {
            throw engine::Error("reconcile.busy", "worktree reconciliation context is busy", 4);
        }
        system_error("reconcile.lock_failed", "could not lock reconciliation context");
    }
    return ContextLock(std::move(current), std::move(lock));
}

std::optional<std::string> read_state_file(int directory, const std::string& name) {
    const int raw = ::openat(
        directory, name.c_str(), O_RDONLY | O_NONBLOCK | O_NOFOLLOW | O_CLOEXEC);
    if (raw < 0 && errno == ENOENT) {
        return std::nullopt;
    }
    if (raw < 0) {
        system_error("reconcile.state_file_open_failed", "could not safely open " + name);
    }
    FileDescriptor file(raw);
    validate_owned_regular(file.get());
    std::string data;
    data.reserve(65536U);
    std::array<char, 16384> buffer {};
    for (;;) {
        const auto count = ::read(file.get(), buffer.data(), buffer.size());
        if (count < 0 && errno == EINTR) {
            continue;
        }
        if (count < 0) {
            system_error("reconcile.state_file_read_failed", "could not read " + name);
        }
        if (count == 0) {
            break;
        }
        if (data.size() + static_cast<std::size_t>(count) > max_state_bytes) {
            throw engine::Error("reconcile.state_file_oversized", name + " exceeds the state limit", 5);
        }
        data.append(buffer.data(), static_cast<std::size_t>(count));
    }
    return data;
}

engine::Json parse_state_json(const std::string& data, const std::string& name) {
    try {
        return engine::parse_bounded_json(data, max_state_bytes);
    } catch (const engine::Error&) {
        throw engine::Error("reconcile.state_json_invalid", name + " is not valid bounded JSON", 5);
    }
}

Candidate read_candidate(int directory, int slot) {
    Candidate result;
    result.slot = slot;
    const auto data = read_state_file(directory, "journal." + std::to_string(slot) + ".json");
    if (!data.has_value()) {
        return result;
    }
    result.exists = true;
    try {
        result.journal = parse_state_json(*data, "journal slot");
        if (!result.journal.is_object() ||
            !result.journal.contains("protocol") ||
            !result.journal.at("protocol").is_string() ||
            result.journal.at("protocol") != journal_protocol ||
            !result.journal.contains("format_version") ||
            (!result.journal.at("format_version").is_number_integer() &&
             !result.journal.at("format_version").is_number_unsigned()) ||
            result.journal.at("format_version") != format_version) {
            throw engine::Error(
                "reconcile.journal_incompatible",
                "journal slot uses an unsupported protocol or format version",
                4);
        }
        validate_journal(result.journal);
        result.valid = true;
    } catch (const engine::Error& error) {
        result.invalid_reason = error.code();
        result.incompatible =
            error.code() == "reconcile.journal_incompatible" ||
            error.code() == "reconcile.critical_capability_unknown" ||
            error.code() == "reconcile.critical_extension_unknown";
    }
    return result;
}

std::size_t cleanup_stale_head_temps(int directory) {
    const int duplicate = ::dup(directory);
    if (duplicate < 0) {
        system_error("reconcile.state_open_failed", "could not inspect context directory");
    }
    DIR* raw = ::fdopendir(duplicate);
    if (raw == nullptr) {
        const int saved = errno;
        ::close(duplicate);
        errno = saved;
        system_error("reconcile.state_open_failed", "could not inspect context directory");
    }
    std::size_t removed = 0;
    errno = 0;
    while (const auto* entry = ::readdir(raw)) {
        const std::string name(entry->d_name);
        if (!name.starts_with(".head.tmp-")) {
            continue;
        }
        const int fd = ::openat(
            directory, name.c_str(), O_RDONLY | O_NOFOLLOW | O_CLOEXEC);
        if (fd < 0) {
            const int saved = errno;
            ::closedir(raw);
            errno = saved;
            system_error(
                "reconcile.state_file_open_failed",
                "stale head temporary file is unsafe");
        }
        try {
            FileDescriptor file(fd);
            validate_owned_regular(file.get());
        } catch (...) {
            ::closedir(raw);
            throw;
        }
        if (::unlinkat(directory, name.c_str(), 0) != 0) {
            const int saved = errno;
            ::closedir(raw);
            errno = saved;
            system_error(
                "reconcile.state_cleanup_failed",
                "could not remove stale head temporary file");
        }
        ++removed;
    }
    const int read_error = errno;
    ::closedir(raw);
    if (read_error != 0) {
        errno = read_error;
        system_error("reconcile.state_read_failed", "could not enumerate context directory");
    }
    if (removed > 0U && ::fsync(directory) != 0) {
        system_error(
            "reconcile.state_sync_failed",
            "could not synchronize stale temporary cleanup");
    }
    return removed;
}

std::optional<engine::Json> read_valid_head(int directory, std::string* invalid_reason = nullptr) {
    const auto data = read_state_file(directory, "head.json");
    if (!data.has_value()) {
        return std::nullopt;
    }
    try {
        auto head = parse_state_json(*data, "reconciliation head");
        if (!head.is_object() || !head.contains("protocol") ||
            !head.at("protocol").is_string() || head.at("protocol") != head_protocol ||
            !head.contains("format_version") ||
            (!head.at("format_version").is_number_integer() &&
             !head.at("format_version").is_number_unsigned()) ||
            head.at("format_version") != format_version) {
            throw engine::Error(
                "reconcile.head_incompatible",
                "reconciliation head uses an unsupported protocol or format version",
                4);
        }
        validate_head(head);
        return head;
    } catch (const engine::Error& error) {
        if (invalid_reason != nullptr) {
            *invalid_reason = error.code();
            return std::nullopt;
        }
        throw;
    }
}

LoadedState load_normal(int directory) {
    const auto head = read_valid_head(directory);
    if (!head.has_value()) {
        const auto slot0 = read_state_file(directory, "journal.0.json");
        const auto slot1 = read_state_file(directory, "journal.1.json");
        if (slot0.has_value() || slot1.has_value()) {
            throw engine::Error(
                "reconcile.recovery_required",
                "journal slots exist without a valid head; run explicit discovery recovery",
                4);
        }
        return {};
    }
    const int slot = static_cast<int>(required_unsigned(*head, "active_slot"));
    const auto candidate = read_candidate(directory, slot);
    if (candidate.incompatible) {
        throw engine::Error(
            candidate.invalid_reason,
            "active journal requires a newer or unknown critical contract; state was preserved",
            4);
    }
    if (!candidate.valid ||
        candidate.journal.at("journal_digest") != head->at("journal_digest") ||
        candidate.journal.at("generation") != head->at("generation") ||
        candidate.journal.at("context_id") != head->at("context_id") ||
        candidate.journal.at("repository").at("worktree_key") != head->at("worktree_key")) {
        throw engine::Error(
            "reconcile.recovery_required",
            "head and active journal slot do not agree; run explicit recovery",
            4);
    }
    const int other_slot = 1 - slot;
    const auto other = read_candidate(directory, other_slot);
    if (other.incompatible) {
        throw engine::Error(
            "reconcile.compatibility_required",
            "inactive journal slot contains newer or unknown critical state; state was preserved",
            4);
    }
    if (other.valid &&
        other.journal.at("journal_digest") != candidate.journal.at("journal_digest")) {
        const auto current_generation = required_unsigned(candidate.journal, "generation");
        const auto other_generation = required_unsigned(other.journal, "generation");
        if (other_generation == current_generation + 1U &&
            other.journal.at("previous_journal_digest") ==
                candidate.journal.at("journal_digest")) {
            throw engine::Error(
                "reconcile.recovery_required",
                "a uniquely linked durable successor exists beyond the atomic head; run recovery",
                4);
        }
        if (other_generation >= current_generation) {
            throw engine::Error(
                "reconcile.recovery_ambiguous",
                "inactive journal evidence is equally ranked or ahead without valid continuity",
                4);
        }
    }
    return LoadedState{*head, candidate.journal, true};
}

void write_all(int fd, const std::string& data, const std::string& name) {
    std::size_t written = 0;
    while (written < data.size()) {
        const auto count = ::write(fd, data.data() + written, data.size() - written);
        if (count < 0 && errno == EINTR) {
            continue;
        }
        if (count <= 0) {
            system_error("reconcile.state_file_write_failed", "could not write " + name);
        }
        written += static_cast<std::size_t>(count);
    }
}

void write_slot(int directory, int slot, const engine::Json& journal) {
    const auto name = "journal." + std::to_string(slot) + ".json";
    const int raw = ::openat(
        directory, name.c_str(), O_CREAT | O_RDWR | O_NOFOLLOW | O_CLOEXEC, 0600);
    if (raw < 0) {
        system_error("reconcile.state_file_open_failed", "could not open inactive journal slot");
    }
    FileDescriptor file(raw);
    validate_owned_regular(file.get());
    if (::fchmod(file.get(), 0600) != 0 || ::ftruncate(file.get(), 0) != 0) {
        system_error("reconcile.state_file_write_failed", "could not prepare inactive journal slot");
    }
    const auto data = journal.dump() + '\n';
    write_all(file.get(), data, name);
    if (::fsync(file.get()) != 0) {
        system_error("reconcile.state_sync_failed", "could not synchronize inactive journal slot");
    }
}

std::string temporary_head_name() {
    static std::atomic<std::uint64_t> counter {0U};
    return ".head.tmp-" + std::to_string(::getpid()) + "-" +
           std::to_string(engine::unix_time_ms()) + "-" +
           std::to_string(counter.fetch_add(1U));
}

void ensure_head_target_safe(int directory) {
    const int raw = ::openat(directory, "head.json", O_RDONLY | O_NOFOLLOW | O_CLOEXEC);
    if (raw < 0 && errno == ENOENT) {
        return;
    }
    if (raw < 0) {
        system_error("reconcile.state_file_open_failed", "existing head is unsafe");
    }
    FileDescriptor file(raw);
    validate_owned_regular(file.get());
}

void write_head_atomic(int directory, const engine::Json& head) {
    ensure_head_target_safe(directory);
    const auto temporary = temporary_head_name();
    const int raw = ::openat(
        directory,
        temporary.c_str(),
        O_CREAT | O_EXCL | O_WRONLY | O_NOFOLLOW | O_CLOEXEC,
        0600);
    if (raw < 0) {
        system_error("reconcile.state_file_open_failed", "could not create temporary head");
    }
    FileDescriptor file(raw);
    validate_owned_regular(file.get());
    const auto data = head.dump() + '\n';
    try {
        write_all(file.get(), data, "temporary head");
        if (::fsync(file.get()) != 0) {
            system_error("reconcile.state_sync_failed", "could not synchronize temporary head");
        }
        if (::renameat(directory, temporary.c_str(), directory, "head.json") != 0) {
            system_error("reconcile.state_commit_failed", "could not atomically replace head");
        }
        if (::fsync(directory) != 0) {
            system_error("reconcile.state_sync_failed", "could not synchronize context directory");
        }
    } catch (...) {
        static_cast<void>(::unlinkat(directory, temporary.c_str(), 0));
        throw;
    }
}

LoadedState commit_journal(
    int directory,
    engine::Json journal,
    const std::optional<engine::Json>& prior_head) {
    finalize_digest(journal, "journal_digest");
    validate_journal(journal);
    const int slot = prior_head.has_value()
        ? 1 - static_cast<int>(required_unsigned(*prior_head, "active_slot"))
        : 0;
    write_slot(directory, slot, journal);
    engine::Json previous_head_digest = nullptr;
    if (prior_head.has_value()) {
        previous_head_digest = prior_head->at("head_digest");
    }
    engine::Json head{
        {"protocol", head_protocol},
        {"format_version", format_version},
        {"context_id", journal.at("context_id")},
        {"worktree_key", journal.at("repository").at("worktree_key")},
        {"active_slot", slot},
        {"generation", journal.at("generation")},
        {"journal_digest", journal.at("journal_digest")},
        {"previous_head_digest", previous_head_digest},
        {"updated_at", journal.at("updated_at")},
    };
    finalize_digest(head, "head_digest");
    validate_head(head);
    write_head_atomic(directory, head);
    return LoadedState{head, journal, true};
}

bool replayed_operation(const engine::Json& journal, const std::string& operation_id, const std::string& kind) {
    for (const auto& checkpoint : journal.at("checkpoints")) {
        if (checkpoint.at("operation_id") == operation_id) {
            if (checkpoint.at("kind") != kind) {
                throw engine::Error(
                    "reconcile.operation_reused",
                    "operation identifier was already used for another mutation",
                    4);
            }
            return true;
        }
    }
    return false;
}

void require_expected(
    const LoadedState& state,
    const std::string& expected,
    bool allow_absent) {
    if (expected == "absent") {
        if (!allow_absent || state.present) {
            throw engine::Error(
                "reconcile.expected_state_mismatch",
                state.present
                    ? "journal was expected absent but current digest is " +
                          state.journal.at("journal_digest").get<std::string>()
                    : "absent is not valid for this operation",
                4);
        }
        return;
    }
    if (!tagged_digest(expected) || !state.present ||
        state.journal.at("journal_digest") != expected) {
        throw engine::Error(
            "reconcile.expected_state_mismatch",
            state.present
                ? "journal expected-state digest is stale"
                : "journal is absent",
            4);
    }
}

engine::Json make_result(
    const std::string& operation,
    engine::Json compatibility,
    const LoadedState& state,
    bool changed,
    bool recovered,
    const std::vector<std::string>& repair_actions,
    bool read_only) {
    engine::Json journal = nullptr;
    engine::Json digest = nullptr;
    if (state.present) {
        journal = state.journal;
        digest = state.journal.at("journal_digest");
    }
    return engine::Json{
        {"protocol", result_protocol},
        {"operation", operation},
        {"compatibility", std::move(compatibility)},
        {"journal_present", state.present},
        {"journal", std::move(journal)},
        {"journal_digest", std::move(digest)},
        {"changed", changed},
        {"recovered", recovered},
        {"repair_actions", repair_actions},
        {"read_only", read_only},
        {"canonical_apply_enabled", false},
        {"canonical", false},
    };
}

engine::Json validate_command(const engine::Request& request) {
    const auto& payload = request.payload;
    require_exact_fields(payload, {
        "protocol", "operation", "state_root", "operation_id",
        "expected_journal_digest", "paths", "binding_registry_digest",
        "engine_inventory", "client",
    });
    static const std::set<std::string> operations = {
        "compatibility", "begin", "status", "checkpoint", "close", "recover",
    };
    if (required_string(payload, "protocol") != command_protocol ||
        required_string(payload, "operation", 64U) != request.operation ||
        !operations.contains(request.operation)) {
        throw engine::Error("reconcile.command_mismatch", "reconciliation command identity is invalid", 4);
    }
    if (!safe_absolute_path(required_string(
            payload, "state_root", engine::Limits::max_path_bytes))) {
        throw engine::Error("reconcile.state_root_invalid", "state_root is not a safe absolute path", 4);
    }
    static_cast<void>(engine_inventory_from_command(payload));
    const auto operation = request.operation;
    const bool read = operation == "compatibility" || operation == "status";
    if (read) {
        if (!payload.at("operation_id").is_null() ||
            !payload.at("expected_journal_digest").is_null() ||
            !payload.at("paths").is_array() || !payload.at("paths").empty()) {
            throw engine::Error("reconcile.command_invalid", "read operation fields are inconsistent", 4);
        }
    } else {
        if (!payload.at("operation_id").is_string() ||
            !safe_token(payload.at("operation_id").get<std::string>()) ||
            !payload.at("expected_journal_digest").is_string()) {
            throw engine::Error("reconcile.command_invalid", "mutation identity or expected state is invalid", 4);
        }
        const auto expected = payload.at("expected_journal_digest").get<std::string>();
        if (operation == "begin") {
            const auto paths = string_array(
                payload.at("paths"), "paths", engine::Limits::max_snapshot_files, false);
            if (paths.empty() || (expected != "absent" && !tagged_digest(expected))) {
                throw engine::Error("reconcile.command_invalid", "begin fields are invalid", 4);
            }
        } else {
            if (!payload.at("paths").is_array() || !payload.at("paths").empty()) {
                throw engine::Error("reconcile.command_invalid", "mutation paths must be empty", 4);
            }
            if (operation == "recover") {
                if (expected != "discover" && !tagged_digest(expected)) {
                    throw engine::Error("reconcile.command_invalid", "recover expected state is invalid", 4);
                }
            } else if (!tagged_digest(expected)) {
                throw engine::Error("reconcile.command_invalid", "expected journal digest is invalid", 4);
            }
        }
    }
    return compatibility_result(payload.at("client"), nullptr);
}

std::string canonical_repository_root() {
    std::error_code error;
    const auto root = fs::canonical(fs::current_path(), error);
    if (error || !fs::is_directory(root) || !safe_absolute_path(root.string())) {
        throw engine::Error("reconcile.repository_invalid", "process working directory is not a canonical repository root", 5);
    }
    return root.string();
}

std::vector<std::string> inventory_paths(const engine::Json& journal) {
    return string_array(
        journal.at("inventory").at("paths"),
        "inventory.paths",
        engine::Limits::max_snapshot_files,
        false);
}

engine::Json begin_journal(
    const engine::Request& request,
    const std::string& repository_root,
    const std::string& repository_key,
    const std::string& worktree_key,
    const LoadedState& prior) {
    auto paths = string_array(
        request.payload.at("paths"), "paths", engine::Limits::max_snapshot_files, false);
    std::sort(paths.begin(), paths.end());
    const auto snapshot = snapshot_json(engine::snapshot_files(
        repository_root, paths, request.deadline_unix_ms));
    const auto observed = utc_now();
    const auto operation_id = request.payload.at("operation_id").get<std::string>();
    const auto identity_input = worktree_key + "|" + operation_id + "|" + observed;
    const auto context_id = "reconcile:" + engine::sha256_hex(identity_input);
    const auto journal_id = "journal:" + engine::sha256_hex(context_id + "|" + snapshot.at("digest").get<std::string>());
    engine::Json previous = nullptr;
    std::uint64_t generation = 1U;
    if (prior.present) {
        previous = prior.journal.at("journal_digest");
        generation = required_unsigned(prior.journal, "generation") + 1U;
    }
    const auto engine_inventory = engine_inventory_from_command(request.payload);
    engine::Json checkpoint = make_checkpoint(
        1U, "begin", operation_id, observed,
        snapshot.at("digest").get<std::string>(),
        snapshot.at("files").size(),
        engine_inventory.at("digest").get<std::string>(),
        nullptr);
    return engine::Json{
        {"protocol", journal_protocol},
        {"format_version", format_version},
        {"journal_id", journal_id},
        {"context_id", context_id},
        {"generation", generation},
        {"previous_journal_digest", previous},
        {"state", "open"},
        {"repository", engine::Json{
            {"repository_root", repository_root},
            {"repository_key", repository_key},
            {"worktree_key", worktree_key},
        }},
        {"inventory", engine::Json{
            {"source", "caller_declared"},
            {"paths", paths},
            {"digest", engine::tagged_sha256(engine::Json(paths).dump())},
        }},
        {"engine_inventory", engine_inventory},
        {"current_snapshot", snapshot},
        {"checkpoints", engine::Json::array({checkpoint})},
        {"compatibility", command_compatibility()},
        {"extensions", engine::Json::array()},
        {"recovery", engine::Json{
            {"state", "clean"},
            {"last_recovery_at", nullptr},
            {"disposition", "not_applicable"},
            {"recovered_from_digest", nullptr},
            {"detail", "no recovery has been required"},
        }},
        {"started_at", observed},
        {"updated_at", observed},
        {"closed_at", nullptr},
        {"canonical", false},
    };
}

engine::Json mutate_checkpoint(
    const engine::Request& request,
    const LoadedState& current,
    const std::string& kind,
    const std::string& recovery_disposition = "not_applicable",
    const engine::Json& recovered_from = nullptr,
    const std::string& recovery_detail = "no recovery has been required") {
    auto next = current.journal;
    const auto paths = inventory_paths(next);
    const bool closed_recovery = kind == "recover" && next.at("state") == "closed";
    const auto snapshot = closed_recovery
        ? next.at("current_snapshot")
        : snapshot_json(engine::snapshot_files(
              next.at("repository").at("repository_root").get<std::string>(),
              paths,
              request.deadline_unix_ms));
    const auto current_engine_inventory = engine_inventory_from_command(request.payload);
    const auto changed = changed_file_count(next.at("current_snapshot"), snapshot);
    const auto observed = utc_now();
    const auto sequence = next.at("checkpoints").size() + 1U;
    if (sequence > max_checkpoints) {
        throw engine::Error(
            "reconcile.checkpoint_limit",
            "checkpoint history is full; close this context and begin a new one",
            4);
    }
    const auto previous_checkpoint = next.at("checkpoints").back().at("checkpoint_digest");
    next["checkpoints"].push_back(make_checkpoint(
        sequence,
        kind,
        request.payload.at("operation_id").get<std::string>(),
        observed,
        snapshot.at("digest").get<std::string>(),
        changed,
        current_engine_inventory.at("digest").get<std::string>(),
        previous_checkpoint));
    next["generation"] = required_unsigned(next, "generation") + 1U;
    next["previous_journal_digest"] = current.journal.at("journal_digest");
    next["current_snapshot"] = snapshot;
    next["engine_inventory"] = current_engine_inventory;
    next["updated_at"] = observed;
    if (kind == "close") {
        next["state"] = "closed";
        next["closed_at"] = observed;
    }
    if (kind == "recover") {
        next["recovery"] = engine::Json{
            {"state", "recovered"},
            {"last_recovery_at", observed},
            {"disposition", recovery_disposition},
            {"recovered_from_digest", recovered_from},
            {"detail", recovery_detail},
        };
    }
    return next;
}

LoadedState recover_state(
    const engine::Request& request,
    int directory,
    const std::string& worktree_key,
    bool discover,
    std::vector<std::string>* actions,
    bool* changed) {
    const auto removed_temporaries = cleanup_stale_head_temps(directory);
    if (removed_temporaries > 0U) {
        actions->push_back(
            "removed " + std::to_string(removed_temporaries) +
            " stale atomic-head temporary file(s)");
        *changed = true;
    }
    std::string head_invalid;
    auto head = read_valid_head(directory, &head_invalid);
    auto slot0 = read_candidate(directory, 0);
    auto slot1 = read_candidate(directory, 1);
    if (head_invalid == "reconcile.head_incompatible" ||
        slot0.incompatible || slot1.incompatible) {
        throw engine::Error(
            "reconcile.compatibility_required",
            "recovery found newer or unknown critical state; no downgrade or overwrite was attempted",
            4);
    }
    std::vector<Candidate*> valid;
    if (slot0.valid) valid.push_back(&slot0);
    if (slot1.valid) valid.push_back(&slot1);
    if (valid.empty()) {
        if (!slot0.exists && !slot1.exists && !head.has_value() && head_invalid.empty()) {
            return {};
        }
        throw engine::Error(
            "reconcile.unrecoverable",
            "no valid journal slot remains; preserve state and restore from external backup",
            5);
    }
    for (const auto* candidate : valid) {
        if (candidate->journal.at("repository").at("worktree_key") != worktree_key) {
            throw engine::Error("reconcile.unrecoverable", "journal slot belongs to another worktree", 5);
        }
    }

    Candidate* selected = nullptr;
    bool adopted_linked_successor = false;
    bool selected_alternate = false;
    if (head.has_value()) {
        const int active = static_cast<int>(required_unsigned(*head, "active_slot"));
        Candidate* active_candidate = active == 0 ? &slot0 : &slot1;
        Candidate* other_candidate = active == 0 ? &slot1 : &slot0;
        if (active_candidate->valid &&
            active_candidate->journal.at("journal_digest") == head->at("journal_digest")) {
            selected = active_candidate;
            if (other_candidate->valid &&
                required_unsigned(other_candidate->journal, "generation") ==
                    required_unsigned(selected->journal, "generation") + 1U &&
                other_candidate->journal.at("previous_journal_digest") ==
                    selected->journal.at("journal_digest")) {
                selected = other_candidate;
                adopted_linked_successor = true;
                actions->push_back("adopted one uniquely linked durable successor slot");
            }
        } else if (other_candidate->valid) {
            selected = other_candidate;
            selected_alternate = true;
            actions->push_back("selected intact alternate slot after active-slot validation failed");
        }
    }
    if (selected == nullptr) {
        std::sort(valid.begin(), valid.end(), [](const Candidate* left, const Candidate* right) {
            return required_unsigned(left->journal, "generation") >
                   required_unsigned(right->journal, "generation");
        });
        if (valid.size() == 2U &&
            required_unsigned(valid[0]->journal, "generation") ==
                required_unsigned(valid[1]->journal, "generation") &&
            valid[0]->journal.at("journal_digest") != valid[1]->journal.at("journal_digest")) {
            throw engine::Error(
                "reconcile.recovery_ambiguous",
                "journal slots diverge at the same generation; administrator review is required",
                4);
        }
        selected = valid.front();
        actions->push_back("selected the highest unique valid journal slot");
    }

    LoadedState baseline;
    baseline.present = true;
    baseline.journal = selected->journal;
    if (head.has_value()) {
        baseline.head = *head;
    }
    if (replayed_operation(
            baseline.journal,
            request.payload.at("operation_id").get<std::string>(),
            "recover")) {
        actions->push_back("replayed recovery operation already committed");
        return baseline;
    }
    const auto expected_recovery_state = head.has_value()
        ? head->at("journal_digest")
        : selected->journal.at("journal_digest");
    if (!discover &&
        request.payload.at("expected_journal_digest") != expected_recovery_state) {
        throw engine::Error("reconcile.expected_state_mismatch", "recovery expected-state digest is stale", 4);
    }
    if (baseline.journal.at("state") != "open") {
        actions->push_back("closed context required no snapshot repair");
    }

    const bool head_matches_selected =
        head.has_value() &&
        selected->journal.at("journal_digest") == head->at("journal_digest") &&
        selected->journal.at("generation") == head->at("generation");
    const auto current_inventory = engine_inventory_from_command(request.payload);
    bool snapshot_matches = true;
    if (baseline.journal.at("state") == "open") {
        const auto observed_snapshot = snapshot_json(engine::snapshot_files(
            baseline.journal.at("repository").at("repository_root").get<std::string>(),
            inventory_paths(baseline.journal),
            request.deadline_unix_ms));
        snapshot_matches =
            observed_snapshot.at("digest") ==
            baseline.journal.at("current_snapshot").at("digest");
    }
    const bool inventory_matches =
        current_inventory.at("digest") ==
        baseline.journal.at("engine_inventory").at("digest");
    if (head_matches_selected && !adopted_linked_successor && !selected_alternate &&
        head_invalid.empty() && snapshot_matches && inventory_matches) {
        actions->push_back("journal, head, content snapshot, and engine inventory are already healthy");
        return baseline;
    }

    engine::Json recovered_from = baseline.journal.at("journal_digest");
    std::string disposition = "rolled_forward_from_valid_slot";
    if (adopted_linked_successor) {
        disposition = "adopted_linked_successor";
        recovered_from = head->at("journal_digest");
    } else if (!head_matches_selected || !head_invalid.empty() || selected_alternate) {
        disposition = "repaired_head";
        if (head.has_value()) {
            recovered_from = head->at("journal_digest");
        }
    } else if (head.has_value()) {
        recovered_from = head->at("journal_digest");
    }
    auto next = mutate_checkpoint(
        request,
        baseline,
        "recover",
        disposition,
        recovered_from,
        "recovered from uniquely validated local journal/head evidence");
    const std::optional<engine::Json> prior_head = head;
    auto committed = commit_journal(directory, std::move(next), prior_head);
    actions->push_back("committed a forward recovery checkpoint and repaired atomic head");
    *changed = true;
    return committed;
}

}

engine::Json reconciliation_capabilities() {
    return engine::Json{
        {"process_protocols", engine::Json::array({engine::process_protocol_v1})},
        {"journal_read_versions", engine::Json::array({format_version})},
        {"journal_write_versions", engine::Json::array({format_version})},
        {"required_capabilities", required_capabilities},
        {"optional_capabilities", optional_capabilities},
        {"two_way_procedural_compatibility", true},
        {"silent_downgrade", false},
        {"lossy_migration", false},
    };
}

engine::Json handle_reconciliation(const engine::Request& request) {
    auto compatibility = validate_command(request);
    if (request.operation == "recover" &&
        compatibility.at("mode") != "full") {
        throw engine::Error(
            "reconcile.compatibility_required",
            "recovery requires full process, journal, and capability overlap",
            4);
    }
    const auto repository_root = canonical_repository_root();
    const auto repository_key = engine::tagged_sha256("repository-root:" + repository_root);
    const auto worktree_key = engine::tagged_sha256("worktree-root:" + repository_root);
    const auto worktree_hex = worktree_key.substr(7);
    const bool exclusive =
        request.operation != "compatibility" && request.operation != "status";
    auto context = open_context(
        request.payload.at("state_root").get<std::string>(),
        worktree_hex,
        exclusive);

    if (request.operation == "recover") {
        const bool discover =
            request.payload.at("expected_journal_digest").get<std::string>() == "discover";
        std::vector<std::string> actions;
        bool changed = false;
        auto recovered = recover_state(
            request,
            context.directory_fd(),
            worktree_key,
            discover,
            &actions,
            &changed);
        compatibility = compatibility_result(
            request.payload.at("client"),
            recovered.present ? &recovered.journal : nullptr);
        return make_result(
            request.operation,
            std::move(compatibility),
            recovered,
            changed,
            changed,
            actions,
            false);
    }

    auto current = load_normal(context.directory_fd());
    if (current.present &&
        current.journal.at("repository").at("repository_root") != repository_root) {
        throw engine::Error(
            "reconcile.repository_mismatch",
            "stored reconciliation state belongs to another repository root",
            5);
    }
    compatibility = compatibility_result(
        request.payload.at("client"),
        current.present ? &current.journal : nullptr);
    const auto mode = compatibility.at("mode").get<std::string>();

    if (request.operation == "compatibility") {
        std::vector<std::string> actions;
        if (mode != "full") {
            actions.push_back("bind a coordinator/client pair with overlapping v1 capabilities");
        }
        return make_result(
            request.operation,
            std::move(compatibility),
            current,
            false,
            false,
            actions,
            true);
    }
    if (request.operation == "status") {
        return make_result(
            request.operation,
            std::move(compatibility),
            current,
            false,
            false,
            {},
            true);
    }
    if (mode != "full") {
        throw engine::Error(
            "reconcile.compatibility_required",
            "mutation requires full process, journal, and capability overlap",
            4);
    }

    const auto operation_id = request.payload.at("operation_id").get<std::string>();
    if (request.operation == "begin") {
        if (current.present &&
            replayed_operation(current.journal, operation_id, "begin")) {
            return make_result(
                request.operation,
                std::move(compatibility),
                current,
                false,
                false,
                {"replayed begin operation already committed"},
                false);
        }
        require_expected(
            current,
            request.payload.at("expected_journal_digest").get<std::string>(),
            true);
        if (current.present && current.journal.at("state") != "closed") {
            throw engine::Error(
                "reconcile.context_open",
                "an open context already exists for this worktree",
                4);
        }
        auto journal = begin_journal(
            request,
            repository_root,
            repository_key,
            worktree_key,
            current);
        const std::optional<engine::Json> prior =
            current.present ? std::optional<engine::Json>(current.head) : std::nullopt;
        auto committed = commit_journal(context.directory_fd(), std::move(journal), prior);
        return make_result(
            request.operation,
            std::move(compatibility),
            committed,
            true,
            false,
            {},
            false);
    }

    if (!current.present) {
        throw engine::Error("reconcile.context_absent", "no reconciliation context exists", 4);
    }
    const auto kind = request.operation;
    if (replayed_operation(current.journal, operation_id, kind)) {
        return make_result(
            request.operation,
            std::move(compatibility),
            current,
            false,
            false,
            {"replayed mutation operation already committed"},
            false);
    }
    require_expected(
        current,
        request.payload.at("expected_journal_digest").get<std::string>(),
        false);
    if (current.journal.at("state") != "open") {
        throw engine::Error("reconcile.context_closed", "reconciliation context is already closed", 4);
    }
    auto next = mutate_checkpoint(request, current, kind);
    auto committed = commit_journal(
        context.directory_fd(), std::move(next), std::optional<engine::Json>(current.head));
    return make_result(
        request.operation,
        std::move(compatibility),
        committed,
        true,
        false,
        {},
        false);
}

}
