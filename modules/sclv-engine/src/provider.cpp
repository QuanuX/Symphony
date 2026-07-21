#include "provider.hpp"

#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/limits.hpp"
#include "symphony/knowledge/engine/protocol.hpp"

#include <algorithm>
#include <cctype>
#include <chrono>
#include <set>
#include <string>

namespace symphony::knowledge::sclv::provider {

namespace {

bool safe_version(std::string_view value) {
    if (value.empty() || value.size() > 64U) return false;
    return std::all_of(value.begin(), value.end(), [](const unsigned char character) {
        const bool alphanumeric =
            (character >= 'a' && character <= 'z') ||
            (character >= 'A' && character <= 'Z') ||
            (character >= '0' && character <= '9');
        return alphanumeric || character == '.' || character == '+' || character == '-';
    });
}

std::string required_string(
    const engine::Json& object,
    const char* field,
    std::size_t maximum = 4096U,
    bool token_value = false) {
    const auto& value = object.at(field);
    if (!value.is_string()) {
        throw engine::Error("provider.invalid_field", std::string(field) + " must be a string", 4);
    }
    const auto text = value.get<std::string>();
    if ((token_value && !safe_token(text, maximum)) ||
        (!token_value && !bounded_text(text, maximum))) {
        throw engine::Error("provider.invalid_field", std::string(field) + " has invalid syntax", 4);
    }
    return text;
}

void validate_repository(const engine::Json& value) {
    if (!value.is_object()) {
        throw engine::Error("provider.repository_required", "repository evidence must be an object", 4);
    }
    require_exact_fields(value, {"revision_scheme", "revision_value", "tree_digest"});
    const auto scheme = required_string(value, "revision_scheme", 128U, true);
    const auto revision = required_string(value, "revision_value", 256U);
    if ((scheme == "git-sha1" && (revision.size() != 40U ||
         !std::all_of(revision.begin(), revision.end(), [](unsigned char c) { return std::isxdigit(c) && !std::isupper(c); }))) ||
        (scheme == "git-sha256" && (revision.size() != 64U ||
         !std::all_of(revision.begin(), revision.end(), [](unsigned char c) { return std::isxdigit(c) && !std::isupper(c); })))) {
        throw engine::Error("provider.revision_invalid", "Git revision does not match its declared scheme", 4);
    }
    if (!tagged_digest(required_string(value, "tree_digest", 71U))) {
        throw engine::Error("provider.tree_digest_invalid", "tree_digest must be tagged SHA-256", 4);
    }
}

}

bool safe_token(std::string_view value, std::size_t maximum) {
    if (value.empty() || value.size() > maximum) return false;
    return std::all_of(value.begin(), value.end(), [](const unsigned char character) {
        const bool alphanumeric =
            (character >= 'a' && character <= 'z') ||
            (character >= 'A' && character <= 'Z') ||
            (character >= '0' && character <= '9');
        return alphanumeric || character == '.' || character == '_' ||
               character == ':' || character == '-';
    });
}

bool bounded_text(std::string_view value, std::size_t maximum, bool allow_newlines) {
    if (value.empty() || value.size() > maximum) return false;
    return std::all_of(value.begin(), value.end(), [&](const unsigned char character) {
        return (allow_newlines && character == '\n') ||
               (character >= 0x20U && character != 0x7fU);
    });
}

bool strict_utc(std::string_view value) {
    if (value.size() != 20U || value[4] != '-' || value[7] != '-' || value[10] != 'T' ||
        value[13] != ':' || value[16] != ':' || value[19] != 'Z') return false;
    for (const std::size_t index : {0U, 1U, 2U, 3U, 5U, 6U, 8U, 9U, 11U, 12U, 14U, 15U, 17U, 18U}) {
        if (value[index] < '0' || value[index] > '9') return false;
    }
    const auto number = [&](std::size_t begin, std::size_t count) {
        int result = 0;
        for (std::size_t index = begin; index < begin + count; ++index) result = result * 10 + (value[index] - '0');
        return result;
    };
    const auto date = std::chrono::year{number(0, 4)} /
                      std::chrono::month{static_cast<unsigned int>(number(5, 2))} /
                      std::chrono::day{static_cast<unsigned int>(number(8, 2))};
    return date.ok() && number(11, 2) <= 23 && number(14, 2) <= 59 && number(17, 2) <= 59;
}

bool tagged_digest(std::string_view value) {
    return value.size() == 71U && value.starts_with("sha256:") &&
           std::all_of(value.begin() + 7, value.end(), [](const unsigned char character) {
               return (character >= '0' && character <= '9') ||
                      (character >= 'a' && character <= 'f');
           });
}

void require_exact_fields(const engine::Json& value, std::initializer_list<const char*> fields) {
    if (!value.is_object() || value.size() != fields.size()) {
        throw engine::Error("provider.field_set", "object is incomplete or contains unknown fields", 4);
    }
    for (const auto* field : fields) {
        if (!value.contains(field)) {
            throw engine::Error("provider.field_set", std::string("required field is absent: ") + field, 4);
        }
    }
}

void validate_change_request(const engine::Json& value) {
    require_exact_fields(value, {"state", "provider", "id", "reference", "absence_reason"});
    const auto state = required_string(value, "state", 32U, true);
    const auto provider_name = required_string(value, "provider");
    const auto id = required_string(value, "id");
    const auto reference = required_string(value, "reference");
    const auto reason = required_string(value, "absence_reason");
    if (state == "present") {
        if (!safe_token(provider_name) || id == "not_applicable" || reference == "not_applicable" ||
            reason != "not_applicable") {
            throw engine::Error("provider.change_request_invalid", "present change-request fields are inconsistent", 4);
        }
    } else if (state == "not_applicable") {
        if (provider_name != "not_applicable" || id != "not_applicable" ||
            reference != "not_applicable" || reason == "not_applicable") {
            throw engine::Error("provider.change_request_invalid", "absent change-request fields are inconsistent", 4);
        }
    } else {
        throw engine::Error("provider.change_request_invalid", "unknown change-request state", 4);
    }
}

void validate_ratification(const engine::Json& value, bool require_asserted) {
    require_exact_fields(value, {
        "state", "subject", "effective_permission", "method",
        "evidence_reference", "evidence_digest", "absence_reason",
    });
    const auto state = required_string(value, "state", 32U, true);
    const auto subject = required_string(value, "subject");
    const auto permission = required_string(value, "effective_permission");
    const auto method = required_string(value, "method");
    const auto reference = required_string(value, "evidence_reference");
    const auto digest = required_string(value, "evidence_digest");
    const auto reason = required_string(value, "absence_reason");
    if (state == "asserted") {
        if (subject == "not_applicable" || permission == "not_applicable" || method == "not_applicable" ||
            reference == "not_applicable" || !tagged_digest(digest) || reason != "not_applicable") {
            throw engine::Error("provider.ratification_invalid", "asserted ratification fields are inconsistent", 4);
        }
    } else if (state == "not_asserted" && !require_asserted) {
        if (subject != "not_applicable" || permission != "not_applicable" || method != "not_applicable" ||
            reference != "not_applicable" || digest != "not_applicable" || reason == "not_applicable") {
            throw engine::Error("provider.ratification_invalid", "absent ratification fields are inconsistent", 4);
        }
    } else {
        throw engine::Error("provider.ratification_invalid", "ratification must be asserted", 4);
    }
}

void validate_evidence(const engine::Json& value) {
    require_exact_fields(value, {
        "protocol", "adapter_id", "adapter_version", "provider_namespace",
        "evidence_kind", "observed_at", "source_reference", "repository",
        "change_request", "ratification", "evidence_digest",
    });
    if (required_string(value, "protocol") != evidence_protocol ||
        !safe_token(required_string(value, "adapter_id")) ||
        !safe_version(required_string(value, "adapter_version", 64U)) ||
        !safe_token(required_string(value, "provider_namespace"))) {
        throw engine::Error("provider.identity_invalid", "provider evidence identity is invalid", 4);
    }
    const auto kind = required_string(value, "evidence_kind", 32U, true);
    if (kind != "revision" && kind != "ratification" && kind != "combined") {
        throw engine::Error("provider.kind_invalid", "provider evidence kind is invalid", 4);
    }
    if (!strict_utc(required_string(value, "observed_at", 20U))) {
        throw engine::Error("provider.timestamp_invalid", "provider evidence timestamp is invalid", 4);
    }
    static_cast<void>(required_string(value, "source_reference"));
    if (!value.at("repository").is_null()) validate_repository(value.at("repository"));
    validate_change_request(value.at("change_request"));
    validate_ratification(value.at("ratification"), false);
    const auto digest = required_string(value, "evidence_digest", 71U);
    if (!tagged_digest(digest)) throw engine::Error("provider.digest_invalid", "provider evidence digest is invalid", 4);
    auto canonical = value;
    canonical.erase("evidence_digest");
    if (digest != engine::tagged_sha256(canonical.dump())) {
        throw engine::Error("provider.digest_mismatch", "provider evidence digest does not match content", 4);
    }
}

engine::Json descriptor(const std::string& adapter_id) {
    return engine::Json{
        {"protocol", engine::descriptor_protocol_v1},
        {"module_id", "sclv-engine"},
        {"engine_id", adapter_id},
        {"vector_id", "sclv"},
        {"engine_version", adapter_version},
        {"process_protocols", engine::Json::array({engine::process_protocol_v1})},
        {"contract_versions", engine::Json::array({"knowledge/SPEC.md@v1", "knowledge/sclv/SPEC.md@v3"})},
        {"operations", engine::Json::array({
            engine::Json{{"name", "normalize"}, {"availability", "implemented"}, {"mutates_canonical", false}},
        })},
        {"limits", engine::Json{
            {"request_bytes", engine::Limits::max_request_bytes},
            {"response_bytes", engine::Limits::max_response_bytes},
            {"json_depth", engine::Limits::max_json_depth},
            {"json_values", engine::Limits::max_json_values},
            {"path_bytes", engine::Limits::max_path_bytes},
            {"snapshot_files", engine::Limits::max_snapshot_files},
            {"snapshot_file_bytes", engine::Limits::max_snapshot_file_bytes},
            {"deadline_ahead_ms", engine::Limits::max_deadline_ahead_ms},
        }},
        {"supported_scopes", engine::Json::array({"user"})},
        {"language", "C++26"},
        {"thermal_path", "freezing"},
        {"install_state", "installed_undocked"},
        {"default_receptor", nullptr},
        {"canonical_apply_enabled", false},
        {"session_mutation_enabled", false},
        {"network_listener", false},
    };
}

engine::Json normalize_airgap(const engine::Json& payload) {
    require_exact_fields(payload, {"observed_at", "source_reference", "repository", "change_request", "ratification"});
    if (!strict_utc(required_string(payload, "observed_at", 20U))) {
        throw engine::Error("provider.timestamp_invalid", "observed_at is invalid", 4);
    }
    static_cast<void>(required_string(payload, "source_reference"));
    validate_repository(payload.at("repository"));
    validate_change_request(payload.at("change_request"));
    validate_ratification(payload.at("ratification"), true);
    engine::Json result{
        {"protocol", evidence_protocol},
        {"adapter_id", "symphony-sclv-evidence-airgap"},
        {"adapter_version", adapter_version},
        {"provider_namespace", "airgap"},
        {"evidence_kind", "combined"},
        {"observed_at", payload.at("observed_at")},
        {"source_reference", payload.at("source_reference")},
        {"repository", payload.at("repository")},
        {"change_request", payload.at("change_request")},
        {"ratification", payload.at("ratification")},
    };
    result["evidence_digest"] = engine::tagged_sha256(result.dump());
    validate_evidence(result);
    return result;
}

}
