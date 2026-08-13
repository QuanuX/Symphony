#include "symphony/knowledge/engine/operation.hpp"

#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/limits.hpp"

#include <algorithm>
#include <set>
#include <string_view>

namespace symphony::knowledge::engine {
namespace {

bool safe_token(std::string_view value, std::size_t maximum) {
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

bool stable_id(std::string_view value, std::string_view prefix) {
    if (!value.starts_with(prefix) || value.size() > 256U) {
        return false;
    }
    const auto namespace_end = value.find(':', prefix.size());
    if (namespace_end == std::string_view::npos || namespace_end == prefix.size() ||
        namespace_end + 1U >= value.size()) {
        return false;
    }
    const auto name_space = value.substr(prefix.size(), namespace_end - prefix.size());
    if (name_space.empty() || name_space.size() > 63U ||
        name_space.front() < 'a' || name_space.front() > 'z' ||
        !std::all_of(name_space.begin() + 1, name_space.end(), [](const unsigned char character) {
            return (character >= 'a' && character <= 'z') ||
                   (character >= '0' && character <= '9') || character == '-';
        })) {
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

}

void validate_operation_specs(const std::vector<OperationSpec>& operations) {
    if (operations.empty() || operations.size() > 1024U) {
        throw Error("operation.registry_cardinality",
                    "operation registry cardinality is outside bounds", 5);
    }
    static const std::set<std::string> availabilities = {
        "implemented", "reserved", "disabled",
    };
    static const std::set<std::string> interactions = {
        "discover", "inspect", "query", "validate", "configure",
        "propose", "invoke", "apply", "lifecycle", "recover",
    };
    static const std::set<std::string> dispositions = {
        "unreviewed", "qxctl_required", "lifecycle_only", "runtime_only",
        "system_orchestrated", "prohibited", "not_applicable",
    };
    static const std::set<std::string> mutabilities = {
        "read_only", "evidence_only", "proposal_only",
        "permission_backed_mutation", "prohibited",
    };
    static const std::set<std::string> idempotencies = {
        "not_applicable", "idempotent", "idempotent_with_invocation_id", "non_idempotent",
    };
    static const std::set<std::string> authorization_requirements = {
        "none", "target_host_permission", "ssiag",
    };
    static const std::set<std::string> direct_invocations = {
        "supported", "diagnostic_only", "prohibited",
    };
    static const std::set<std::string> thermal_paths = {"freezing", "warm", "hot"};
    std::set<std::string> ids;
    std::set<std::string> names;
    for (const auto& operation : operations) {
        if (!stable_id(operation.engine_operation_id, "engop:") ||
            !ids.insert(operation.engine_operation_id).second) {
            throw Error("operation.identity", "operation ID is invalid or duplicated", 5);
        }
        if (!safe_token(operation.operation_name, Limits::max_operation_bytes) ||
            !names.insert(operation.operation_name).second) {
            throw Error("operation.name", "operation name is invalid or duplicated", 5);
        }
        if (!availabilities.contains(operation.availability) ||
            !dispositions.contains(operation.administration_disposition) ||
            !mutabilities.contains(operation.mutability) ||
            !idempotencies.contains(operation.idempotency) ||
            !authorization_requirements.contains(operation.authorization_requirement) ||
            !direct_invocations.contains(operation.direct_invocation) ||
            !thermal_paths.contains(operation.thermal_path)) {
            throw Error("operation.classification", "operation classification is invalid", 5);
        }
        if (operation.administrative_interactions.empty() ||
            operation.administrative_interactions.size() > 10U) {
            throw Error("operation.classification", "operation interactions are absent or excessive", 5);
        }
        std::set<std::string> distinct_interactions;
        for (const auto& interaction : operation.administrative_interactions) {
            if (!interactions.contains(interaction) ||
                !distinct_interactions.insert(interaction).second) {
                throw Error("operation.classification", "operation interaction is invalid or duplicated", 5);
            }
        }
        if (operation.feature_ids.size() > 256U) {
            throw Error("operation.feature_identity", "operation feature identities are excessive", 5);
        }
        std::set<std::string> features;
        for (const auto& feature_id : operation.feature_ids) {
            if (!stable_id(feature_id, "ssfv:") || !features.insert(feature_id).second) {
                throw Error("operation.feature_identity",
                            "operation feature identity is invalid or duplicated", 5);
            }
        }
        if ((operation.input_protocol && !safe_token(*operation.input_protocol, 256U)) ||
            (operation.output_protocol && !safe_token(*operation.output_protocol, 256U))) {
            throw Error("operation.protocol_identity", "operation protocol identity is invalid", 5);
        }
        if (!operation.recovery_operation_id.empty() &&
            !stable_id(operation.recovery_operation_id, "engop:")) {
            throw Error("operation.recovery_identity", "recovery operation ID is invalid", 5);
        }
        if ((operation.administration_disposition == "prohibited" ||
             operation.mutability == "prohibited") &&
            operation.availability == "implemented") {
            throw Error("operation.prohibited_available",
                        "a prohibited operation cannot be implemented", 5);
        }
    }
    for (const auto& operation : operations) {
        if (!operation.recovery_operation_id.empty() &&
            !ids.contains(operation.recovery_operation_id)) {
            throw Error("operation.recovery_missing",
                        "recovery operation ID is not present in the registry", 5);
        }
    }
}

const OperationSpec* find_operation(
    const std::vector<OperationSpec>& operations,
    std::string_view name) noexcept {
    const auto found = std::find_if(operations.begin(), operations.end(),
                                    [&](const OperationSpec& operation) {
        return operation.operation_name == name;
    });
    return found == operations.end() ? nullptr : &*found;
}

Json legacy_operation_descriptors(const std::vector<OperationSpec>& operations) {
    validate_operation_specs(operations);
    Json result = Json::array();
    for (const auto& operation : operations) {
        if (!operation.exposed_in_descriptor_v1) {
            continue;
        }
        result.push_back(Json{{"name", operation.operation_name},
                              {"availability", operation.availability},
                              {"mutates_canonical", operation.mutates_canonical}});
    }
    return result;
}

Json administration_operation_descriptors(const std::vector<OperationSpec>& operations) {
    validate_operation_specs(operations);
    Json result = Json::array();
    for (const auto& operation : operations) {
        result.push_back(Json{
            {"engine_operation_id", operation.engine_operation_id},
            {"operation_name", operation.operation_name},
            {"availability", operation.availability},
            {"feature_ids", operation.feature_ids},
            {"administrative_interactions", operation.administrative_interactions},
            {"administration_disposition", operation.administration_disposition},
            {"input_protocol", operation.input_protocol
                ? Json(*operation.input_protocol) : Json(nullptr)},
            {"output_protocol", operation.output_protocol
                ? Json(*operation.output_protocol) : Json(nullptr)},
            {"mutability", operation.mutability},
            {"idempotency", operation.idempotency},
            {"expected_state_required", operation.expected_state_required},
            {"authorization_requirement", operation.authorization_requirement},
            {"recovery_operation_id", operation.recovery_operation_id.empty()
                ? Json(nullptr) : Json(operation.recovery_operation_id)},
            {"direct_invocation", operation.direct_invocation},
            {"thermal_path", operation.thermal_path},
        });
    }
    return result;
}

std::string operation_registry_digest(const std::vector<OperationSpec>& operations) {
    return tagged_sha256(administration_operation_descriptors(operations).dump());
}

}
