#include "request.hpp"
#include <symphony/sqfv/batch.hpp>
namespace symphony::sqfv::administration {
using namespace symphony::sqv_admin;
namespace {
Binding binding(const Json &j) {
  fields(j, {"metadata_ref", "dataset_revision", "schema_version",
             "layout_version", "access_scope"});
  return {bytes(j, "metadata_ref", 4096), bytes(j, "dataset_revision", 4096),
          bytes(j, "schema_version", 4096), bytes(j, "layout_version", 4096),
          bytes(j, "access_scope", 4096)};
}
} // namespace
Json administer(std::string_view, const Json &p, std::int64_t end) {
  fields(p, {"protocol", "limits", "port", "frame"});
  const auto &l = p.at("limits");
  fields(l, {"max_payload_bytes", "max_frame_bytes", "max_descriptor_bytes",
             "global_allocation_bytes", "max_ports"});
  Limits limits{u64(l, "max_payload_bytes"), u64(l, "max_frame_bytes"),
                u64(l, "max_descriptor_bytes"),
                u64(l, "global_allocation_bytes"),
                narrow<std::uint32_t>(u64(l, "max_ports"))};
  Context context;
  accepted(Context::create(limits, context));
  const auto &pc = p.at("port");
  fields(pc, {"binding", "partition", "producer_generation", "next_sequence",
              "outstanding_byte_credit", "max_pending_entries"});
  PortConfig config{binding(pc.at("binding")),
                    bytes(pc, "partition", 4096),
                    fixed<16>(pc, "producer_generation"),
                    u64(pc, "next_sequence"),
                    u64(pc, "outstanding_byte_credit"),
                    narrow<std::uint32_t>(u64(pc, "max_pending_entries"))};
  Port port;
  accepted(context.add_port(config, port));
  const auto &f = p.at("frame");
  fields(f, {"binding", "partition", "source_binding", "source_position",
             "producer_generation", "batch_sequence", "record_count",
             "payload_bytes"});
  Descriptor d{binding(f.at("binding")),
               bytes(f, "partition", 4096),
               bytes(f, "source_binding", 4096),
               bytes(f, "source_position", 4096),
               fixed<16>(f, "producer_generation"),
               u64(f, "batch_sequence"),
               u64(f, "record_count")};
  auto size = u64(f, "payload_bytes");
  require(size <= limits.max_payload_bytes && size <= (64ULL << 20) &&
          size <= limits.global_allocation_bytes);
  std::vector<std::uint8_t> trial(static_cast<std::size_t>(size));
  deadline(end);
  Batch batch;
  accepted(context.prepare_copy(d, trial, batch));
  std::size_t frame_bytes = 0;
  accepted(frame_measure(context, batch, frame_bytes));
  accepted(port.offer(batch));
  ContextStats stats;
  accepted(context.stats(stats));
  return {{"validation_scope", "isolated_configuration_trial"},
          {"frame_bytes", std::to_string(frame_bytes)},
          {"trial_peak_allocation_bytes",
           std::to_string(stats.peak_allocation_bytes)},
          {"payload_bytes", std::to_string(size)},
          {"binding_resolution", "not_performed"},
          {"runtime_observation", "not_performed"}};
}
} // namespace symphony::sqfv::administration
