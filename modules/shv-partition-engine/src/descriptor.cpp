#include "partition.hpp"
#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/limits.hpp"
#include "symphony/knowledge/engine/operation.hpp"
#include <algorithm>
namespace symphony::knowledge::shv_partition {
namespace {
Json seal(Json j, const std::string &key) {
  j[key] = engine::tagged_sha256(j.dump());
  return j;
}
} // namespace
Json descriptor() {
  std::vector<engine::OperationSpec> specs;
  const std::vector<std::pair<std::string, std::string>> operations = {
      {"inspect", engine::descriptor_protocol_v2},
      {"partition_build", "symphony.shv.partition.v1"},
      {"manifest_build", "symphony.shv.partition-manifest.v1"},
      {"manifest_query", "symphony.shv.partition-query.v1"}};
  for (const auto &[name, output] : operations) {
    auto id = name, input = name;
    std::replace(id.begin(), id.end(), '_', '.');
    std::replace(input.begin(), input.end(), '_', '-');
    if (name == "partition_build")
      input = "build";
    specs.push_back(engine::OperationSpec{
        "engop:symphony:shv-partition." + id,
        name,
        "implemented",
        false,
        true,
        {"ssfv:symphony:shv-partition-engine"},
        {name == "inspect" ? "inspect"
                           : (name == "manifest_query" ? "query" : "invoke")},
        "qxctl_required",
        std::string("symphony.shv.partition-") + input + "-input.v1",
        output,
        "read_only",
        "idempotent",
        false,
        "none",
        "",
        "supported",
        "freezing"});
  }
  return seal(
      Json{{"protocol", engine::descriptor_protocol_v2},
           {"format_version", 2},
           {"module_id", "shv-partition-engine"},
           {"engine_id", engine_id},
           {"vector_id", "shv"},
           {"engine_version", version},
           {"process_protocols", Json::array({engine::process_protocol_v1})},
           {"contract_versions",
            Json::array({"knowledge/shv/PARTITIONS.md@v1",
                         "symphony.shv.partition.v1",
                         "symphony.shv.partition-manifest.v1",
                         "symphony.shv.partition-query.v1"})},
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
} // namespace symphony::knowledge::shv_partition
