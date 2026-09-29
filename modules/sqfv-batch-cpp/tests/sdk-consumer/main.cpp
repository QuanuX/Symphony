#include <symphony/sqfv/batch.hpp>

#include <array>
#include <cstdint>
#include <iostream>
#include <string>
#include <utility>
#include <vector>

namespace sqfv = symphony::sqfv;

namespace {

bool expect(sqfv::Status actual, sqfv::Status wanted, const char* step) {
  if (actual == wanted) return true;
  std::cerr << step << ": wanted status " << static_cast<int>(wanted)
            << ", got " << static_cast<int>(actual) << '\n';
  return false;
}

} // namespace

int consumer_retains_frozen_payload() {
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

int consumer_rejects_incompatible_binding() {
  const sqfv::Limits limits{1024, 4096, 512, 16384, 2};
  sqfv::Context context;
  if (!expect(sqfv::Context::create(limits, context), sqfv::Status::ok,
              "rejection context")) return 1;
  const sqfv::Binding binding{"metadata", "revision", "schema", "layout", "scope"};
  const sqfv::Descriptor descriptor{binding, "partition", {}, {}, {1}, 7, 1};
  const std::array<std::uint8_t, 3> payload{1, 2, 3};
  sqfv::Batch batch;
  if (!expect(context.prepare_copy(descriptor, payload, batch), sqfv::Status::ok,
              "rejection batch")) return 1;
  sqfv::Lease lease;
  if (!expect(batch.acquire(binding.access_scope, lease), sqfv::Status::ok,
              "initial lease")) return 1;
  const auto* retained_payload = lease.payload().data();
  if (!expect(batch.acquire("other-scope", lease), sqfv::Status::scope_mismatch,
              "scope rejection") || lease.payload().data() != retained_payload)
    return 1;

  sqfv::PortConfig config{binding, "partition", {1}, 7, 1024, 2};
  config.binding.schema_version = "other-schema";
  sqfv::Port incompatible;
  if (!expect(context.add_port(config, incompatible), sqfv::Status::ok,
              "incompatible port") ||
      !expect(incompatible.offer(batch), sqfv::Status::binding_mismatch,
              "schema rejection")) return 1;
  sqfv::PortStats before;
  if (!expect(incompatible.stats(before), sqfv::Status::ok, "rejected stats") ||
      before.next_sequence != 7 || before.pending_entries != 0 ||
      before.outstanding_bytes != 0) return 1;
  config.binding = binding;
  sqfv::Port compatible;
  if (!expect(context.add_port(config, compatible), sqfv::Status::ok,
              "compatible port") ||
      !expect(compatible.offer(batch), sqfv::Status::ok, "compatible offer") ||
      !expect(compatible.offer(batch), sqfv::Status::duplicate,
              "duplicate rejection")) return 1;
  sqfv::PortStats after;
  if (!expect(compatible.stats(after), sqfv::Status::ok, "duplicate stats") ||
      after.next_sequence != 8 || after.pending_entries != 1 ||
      after.outstanding_bytes != payload.size()) return 1;
  return 0;
}

int consumer_rejects_corrupt_frame() {
  const sqfv::Limits limits{1024, 4096, 512, 16384, 1};
  sqfv::Context context;
  if (!expect(sqfv::Context::create(limits, context), sqfv::Status::ok,
              "frame context")) return 1;
  const sqfv::Descriptor descriptor{
      {"metadata", "revision", "schema", "layout", "scope"},
      "partition", {}, {}, {1}, 7, 1};
  const std::array<std::uint8_t, 3> payload{1, 2, 3};
  sqfv::Batch batch;
  if (!expect(context.prepare_copy(descriptor, payload, batch), sqfv::Status::ok,
              "frame batch")) return 1;
  const auto identity = batch.content_id();
  std::size_t size = 0;
  if (!expect(sqfv::frame_measure(context, batch, size), sqfv::Status::ok,
              "frame measure")) return 1;
  std::vector<std::uint8_t> frame(size);
  if (!expect(sqfv::frame_encode(context, batch, frame, size), sqfv::Status::ok,
              "frame encode")) return 1;
  sqfv::ContextStats before;
  if (!expect(context.stats(before), sqfv::Status::ok, "frame baseline")) return 1;
  frame.back() ^= 1;
  if (!expect(sqfv::frame_decode(context, frame, batch), sqfv::Status::corrupt_frame,
              "corrupt frame rejection") || batch.content_id() != identity)
    return 1;
  sqfv::ContextStats after;
  if (!expect(context.stats(after), sqfv::Status::ok, "frame failure stats") ||
      after.allocation_bytes != before.allocation_bytes) return 1;
  frame.back() ^= 1;
  if (!expect(sqfv::frame_decode(context, frame, batch), sqfv::Status::ok,
              "verified frame decode") || batch.content_id() != identity)
    return 1;
  return 0;
}

int consumer_rejects_exhausted_budget() {
  const sqfv::Limits limits{1024, 4096, 512, 8192, 1};
  sqfv::Context context;
  if (!expect(sqfv::Context::create(limits, context), sqfv::Status::ok,
              "bounded context")) return 1;
  const sqfv::Descriptor descriptor{
      {"metadata", "revision", "schema", "layout", "scope"},
      "partition", {}, {}, {1}, 7, 1};
  std::array<std::uint8_t, 1024> payload{};
  payload.front() = 42;
  std::vector<sqfv::Lease> retained;
  bool rejected = false;
  for (int attempt = 0; attempt != 16; ++attempt) {
    sqfv::Batch batch;
    const auto prepared = context.prepare_copy(descriptor, payload, batch);
    if (prepared == sqfv::Status::limit) {
      if (batch) return 1;
      rejected = true;
      break;
    }
    if (!expect(prepared, sqfv::Status::ok, "bounded preparation")) return 1;
    sqfv::Lease lease;
    const auto acquired = batch.acquire("scope", lease);
    if (acquired == sqfv::Status::limit) {
      if (lease) return 1;
      rejected = true;
      break;
    }
    if (!expect(acquired, sqfv::Status::ok, "bounded lease")) return 1;
    retained.push_back(std::move(lease));
  }
  sqfv::ContextStats stats;
  if (!rejected || retained.empty() ||
      !expect(context.stats(stats), sqfv::Status::ok, "bounded stats") ||
      stats.allocation_bytes > limits.global_allocation_bytes) return 1;
  context = sqfv::Context{};
  for (const auto& lease : retained) {
    if (lease.payload().size() != payload.size() ||
        lease.payload().front() != 42) return 1;
  }
  return 0;
}

int main() {
  if (consumer_retains_frozen_payload() ||
      consumer_rejects_incompatible_binding() || consumer_rejects_corrupt_frame() ||
      consumer_rejects_exhausted_budget())
    return 1;
  std::cout << "installed consumer: incompatible bindings, duplicates and corrupt "
               "frames rejected without changing retained outputs\n";
  return 0;
}
