#include "request.hpp"
namespace SQV_NAMESPACE {
using namespace symphony::sqv_admin;
Json descriptor() {
  const auto ops = interface_operations();
  engine::validate_operation_specs(ops);
  Json contracts =
      Json::array({std::string("modules/") + module_id + "/SPEC.md@v1"});
  for (const auto &op : ops) {
    contracts.push_back(*op.input_protocol);
    contracts.push_back(*op.output_protocol);
  }
  return seal(
      Json{{"protocol", engine::descriptor_protocol_v2},
           {"format_version", 2},
           {"module_id", module_id},
           {"engine_id", engine_id},
           {"vector_id", vector_id},
           {"engine_version", version},
           {"process_protocols", Json::array({engine::process_protocol_v1})},
           {"contract_versions", contracts},
           {"operations", engine::administration_operation_descriptors(ops)},
           {"limits",
            {{"request_bytes", process_bytes},
             {"response_bytes", process_bytes},
             {"json_depth", engine::Limits::max_json_depth},
             {"json_values", engine::Limits::max_json_values},
             {"path_bytes", engine::Limits::max_path_bytes},
             {"snapshot_files", 131080},
             {"snapshot_file_bytes", 67108864},
             {"deadline_ahead_ms", 5000}}},
           {"supported_scopes", Json::array({"user"})},
           {"language", "C++26"},
           {"thermal_path", "freezing"},
           {"canonical_apply_enabled", false},
           {"session_mutation_enabled", false},
           {"network_listener", false}},
      "descriptor_digest");
}
Json handle_request(const engine::Request &r) {
  const auto ops = interface_operations();
  const auto op = std::find_if(ops.begin(), ops.end(), [&](const auto &v) {
    return v.operation_name == r.operation;
  });
  if (op == ops.end())
    refuse("operation.unsupported", "Unsupported SQV administration operation",
           3);
  require(r.payload.is_object() && r.payload.dump().size() <= payload_bytes &&
          r.payload.contains("protocol") &&
          r.payload.at("protocol") == *op->input_protocol);
  if (r.deadline_unix_ms > engine::unix_time_ms() + 5000)
    refuse("request.invalid_deadline",
           "SQV administration deadline exceeds five seconds");
  deadline(r.deadline_unix_ms);
  auto data = administer(r.operation, r.payload, r.deadline_unix_ms);
  deadline(r.deadline_unix_ms);
  auto out =
      seal(Json{{"protocol", *op->output_protocol},
                {"operation", r.operation},
                {"owner", vector_id},
                {"request_digest", engine::tagged_sha256(r.payload.dump())},
                {"data", std::move(data)},
                {"provider_observation", "not_performed"},
                {"persistent_mutation", false}},
           "result_digest");
  require(out.dump().size() <= 450000);
  return out;
}
} // namespace SQV_NAMESPACE
