#pragma once

#include "symphony/knowledge/engine/json.hpp"

#include <string>
#include <string_view>
#include <optional>
#include <vector>

namespace symphony::knowledge::engine {

struct OperationSpec final {
    std::string engine_operation_id;
    std::string operation_name;
    std::string availability;
    bool mutates_canonical;
    bool exposed_in_descriptor_v1;
    std::vector<std::string> feature_ids;
    std::vector<std::string> administrative_interactions;
    std::string administration_disposition;
    std::optional<std::string> input_protocol;
    std::optional<std::string> output_protocol;
    std::string mutability;
    std::string idempotency;
    bool expected_state_required;
    std::string authorization_requirement;
    std::string recovery_operation_id;
    std::string direct_invocation;
    std::string thermal_path;
};

void validate_operation_specs(const std::vector<OperationSpec>& operations);
[[nodiscard]] const OperationSpec* find_operation(
    const std::vector<OperationSpec>& operations,
    std::string_view name) noexcept;
[[nodiscard]] Json legacy_operation_descriptors(
    const std::vector<OperationSpec>& operations);
[[nodiscard]] Json administration_operation_descriptors(
    const std::vector<OperationSpec>& operations);
[[nodiscard]] std::string operation_registry_digest(
    const std::vector<OperationSpec>& operations);

}
