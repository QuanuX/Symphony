#include "profile.hpp"
#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/limits.hpp"
#include "symphony/knowledge/engine/operation.hpp"
#include <algorithm>
namespace symphony::knowledge::shv_profile {
namespace {
Json seal(Json j, const std::string &key) {
  j[key] = engine::tagged_sha256(j.dump());
  return j;
}
} // namespace
Json descriptor() {
  std::vector<engine::OperationSpec> specs;
  for (const auto &operation : interface_operations) {
    const std::string name = operation.name, output = operation.output;
    auto id = name;
    std::replace(id.begin(), id.end(), '_', '.');
    specs.push_back(engine::OperationSpec{"engop:symphony:shv-profile." + id,
                                          name,
                                          "implemented",
                                          false,
                                          true,
                                          {"ssfv:symphony:shv-profile-engine"},
                                          {operation.interaction},
                                          "qxctl_required",
                                          operation.input,
                                          output,
                                          "read_only",
                                          "idempotent",
                                          false,
                                          "none",
                                          "",
                                          "supported",
                                          "freezing"});
  }
  engine::validate_operation_specs(specs);
  return seal(
      Json{{"protocol", engine::descriptor_protocol_v2},
           {"format_version", 2},
           {"module_id", "shv-profile-engine"},
           {"engine_id", engine_id},
           {"vector_id", "shv"},
           {"engine_version", version},
           {"process_protocols", Json::array({engine::process_protocol_v1})},
           {"contract_versions", Json::array({"knowledge/shv/PROFILES.md@v1"})},
           {"operations", engine::administration_operation_descriptors(specs)},
           {"limits",
            Json{{"request_bytes", engine::Limits::max_request_bytes},
                 {"response_bytes", engine::Limits::max_response_bytes},
                 {"json_depth", engine::Limits::max_json_depth},
                 {"json_values", engine::Limits::max_json_values},
                 {"path_bytes", engine::Limits::max_path_bytes},
                 {"snapshot_files", engine::Limits::max_snapshot_files},
                 {"snapshot_file_bytes",
                  engine::Limits::max_snapshot_file_bytes},
                 {"deadline_ahead_ms", engine::Limits::max_deadline_ahead_ms}}},
           {"supported_scopes", Json::array({"user"})},
           {"language", "C++26"},
           {"thermal_path", "freezing"},
           {"canonical_apply_enabled", false},
           {"session_mutation_enabled", false},
           {"network_listener", false}},
      "descriptor_digest");
}
} // namespace symphony::knowledge::shv_profile
