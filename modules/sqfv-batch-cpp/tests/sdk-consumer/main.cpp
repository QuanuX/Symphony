#include <symphony/sqfv/batch.hpp>

#include <array>
#include <cstdint>
#include <iostream>
#include <string>

namespace sqfv = symphony::sqfv;

namespace {

bool expect(sqfv::Status actual, sqfv::Status wanted, const char* step) {
  if (actual == wanted) return true;
  std::cerr << step << ": wanted status " << static_cast<int>(wanted)
            << ", got " << static_cast<int>(actual) << '\n';
  return false;
}

} // namespace

int main() {
  const sqfv::Limits limits{
      .max_payload_bytes = 1024,
      .max_frame_bytes = 4096,
      .max_descriptor_bytes = 512,
      .global_allocation_bytes = 8192,
      .max_ports = 2,
  };
  sqfv::Context context;
  if (!expect(sqfv::Context::create(limits, context), sqfv::Status::ok,
              "context create")) return 1;
  sqfv::ContextStats baseline;
  if (!expect(context.stats(baseline), sqfv::Status::ok,
              "initial context stats")) return 1;

  const sqfv::Binding binding{
      .metadata_ref = "fixture:metadata:1",
      .dataset_revision = "revision-1",
      .schema_version = "schema-1",
      .layout_version = "row-v1",
      .access_scope = "fixture-scope",
  };
  sqfv::Descriptor descriptor{
      .binding = binding,
      .partition = "partition-a",
      .source_binding = "fixture-source",
      .source_position = "opaque-7",
      .producer_generation = {1},
      .batch_sequence = 7,
      .record_count = 3,
  };
  std::array<std::uint8_t, 16> caller_payload{
      0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15};
  sqfv::Batch batch;
  if (!expect(context.prepare_copy(descriptor, caller_payload, batch),
              sqfv::Status::ok, "prepare copy")) return 1;
  caller_payload[0] = 99;

  sqfv::Lease direct;
  if (!expect(batch.acquire(binding.access_scope, direct), sqfv::Status::ok,
              "read lease acquire")) return 1;
  const auto view = direct.payload();
  if (view.size() != 16 || view[0] != 0 || view[15] != 15 ||
      direct.descriptor().binding.dataset_revision != "revision-1" ||
      direct.content_id() != batch.content_id()) {
    std::cerr << "frozen payload or descriptor mismatch\n";
    return 1;
  }

  const sqfv::PortConfig port_config{
      .binding = binding,
      .partition = "partition-a",
      .producer_generation = {1},
      .next_sequence = 7,
      .outstanding_byte_credit = 1024,
      .max_pending_entries = 2,
  };
  sqfv::Port port;
  if (!expect(context.add_port(port_config, port), sqfv::Status::ok,
              "port add") ||
      !expect(port.offer(batch), sqfv::Status::ok, "port offer")) return 1;
  sqfv::Lease taken;
  if (!expect(port.take(taken), sqfv::Status::ok, "port take")) return 1;
  port.reset();
  batch = sqfv::Batch{};
  if (taken.payload().size() != 16 || taken.payload()[0] != 0 ||
      taken.descriptor().batch_sequence != 7) {
    std::cerr << "taken lease lost its retained batch\n";
    return 1;
  }
  taken.reset();
  direct.reset();

  sqfv::ContextStats stats;
  if (!expect(context.stats(stats), sqfv::Status::ok, "context stats") ||
      stats.allocation_bytes != baseline.allocation_bytes ||
      stats.peak_allocation_bytes <= baseline.allocation_bytes) {
    std::cerr << "unexpected allocation accounting\n";
    return 1;
  }
  std::cout << "installed C++26 consumer: frozen copy, exact binding, "
               "offer/take, lease lifetime, reservation accounting\n";
  return 0;
}
