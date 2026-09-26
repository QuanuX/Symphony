#include "symphony/sqfv/batch.h"

#include <algorithm>
#include <array>
#include <atomic>
#include <chrono>
#include <cstdint>
#include <cstring>
#include <iostream>
#include <limits>
#include <stdexcept>
#include <string>
#include <thread>
#include <vector>

namespace {

sqfv_span span(const std::string &value) {
  return {reinterpret_cast<const uint8_t *>(value.data()), value.size()};
}

sqfv_span span(const std::vector<uint8_t> &value) {
  return {value.data(), value.size()};
}

void require(bool condition, const char *message) {
  if (!condition) throw std::runtime_error(message);
}

void expect(sqfv_status actual, sqfv_status wanted, const char *operation) {
  if (actual != wanted) {
    throw std::runtime_error(std::string(operation) + ": got " +
                             std::to_string(static_cast<int>(actual)) + ", want " +
                             std::to_string(static_cast<int>(wanted)));
  }
}

struct FixtureDescriptor {
  std::string metadata_ref = "fixture-metadata-revision-1";
  std::string dataset_revision = "fixture-dataset-revision-1";
  std::string schema_version = "fixture-schema-1";
  std::string layout_version = "fixture-layout-1";
  std::string access_scope = "fixture-research-scope";
  std::string partition = "fixture-partition-0";
  std::string source_binding = "fixture-source-binding";
  std::string source_position = "source-native-position";
  std::array<uint8_t, SQFV_BATCH_GENERATION_BYTES> generation = {1};

  sqfv_binding binding() const {
    return {sizeof(sqfv_binding), SQFV_BATCH_ABI_VERSION,
            span(metadata_ref), span(dataset_revision), span(schema_version),
            span(layout_version), span(access_scope)};
  }

  sqfv_descriptor descriptor(uint64_t sequence) const {
    sqfv_descriptor result{};
    result.struct_size = sizeof(result);
    result.abi_version = SQFV_BATCH_ABI_VERSION;
    result.binding = binding();
    result.partition = span(partition);
    result.source_binding = span(source_binding);
    result.source_position = span(source_position);
    std::copy(generation.begin(), generation.end(), result.producer_generation);
    result.batch_sequence = sequence;
    result.record_count = 128;
    return result;
  }

  sqfv_port_config port_config(uint64_t next_sequence, uint64_t byte_credit,
                               uint32_t max_pending) const {
    sqfv_port_config result{};
    result.struct_size = sizeof(result);
    result.abi_version = SQFV_BATCH_ABI_VERSION;
    result.binding = binding();
    result.partition = span(partition);
    std::copy(generation.begin(), generation.end(), result.producer_generation);
    result.next_sequence = next_sequence;
    result.outstanding_byte_credit = byte_credit;
    result.max_pending_entries = max_pending;
    return result;
  }
};

sqfv_limits limits(uint64_t allocation_bytes = 8U * 1024U * 1024U,
                   uint32_t max_ports = 3) {
  return {sizeof(sqfv_limits), SQFV_BATCH_ABI_VERSION,
          65536, 73728, 4096, allocation_bytes, max_ports, 0};
}

sqfv_context *make_context(sqfv_limits selected = limits()) {
  sqfv_context *context = nullptr;
  expect(sqfv_context_create(&selected, &context), SQFV_OK, "context create");
  require(context != nullptr, "context create returned no handle");
  return context;
}

sqfv_batch *make_batch(sqfv_context *context, const FixtureDescriptor &fixture,
                       uint64_t sequence, const std::vector<uint8_t> &payload) {
  const auto descriptor = fixture.descriptor(sequence);
  sqfv_batch *batch = nullptr;
  expect(sqfv_batch_prepare_copy(context, &descriptor, span(payload), &batch),
         SQFV_OK, "batch prepare");
  require(batch != nullptr, "batch prepare returned no handle");
  return batch;
}

sqfv_port *make_port(sqfv_context *context, const FixtureDescriptor &fixture,
                     uint64_t next_sequence, uint64_t credit,
                     uint32_t pending) {
  const auto config = fixture.port_config(next_sequence, credit, pending);
  sqfv_port *port = nullptr;
  expect(sqfv_port_add(context, &config, &port), SQFV_OK, "port add");
  require(port != nullptr, "port add returned no handle");
  return port;
}

sqfv_lease *take(sqfv_port *port) {
  sqfv_lease *lease = nullptr;
  expect(sqfv_port_take(port, &lease), SQFV_OK, "port take");
  require(lease != nullptr, "port take returned no lease");
  return lease;
}

void immutability_and_lifetime() {
  auto invalid = limits();
  invalid.abi_version++;
  sqfv_context *rejected = nullptr;
  expect(sqfv_context_create(&invalid, &rejected), SQFV_UNSUPPORTED_ABI,
         "unsupported ABI");
  require(rejected == nullptr, "unsupported ABI returned a context");

  FixtureDescriptor fixture;
  auto *context = make_context();
  std::vector<uint8_t> payload(65536, 0x36);
  auto *batch = make_batch(context, fixture, 1, payload);
  std::fill(payload.begin(), payload.end(), 0x90);

  sqfv_descriptor batch_descriptor{};
  batch_descriptor.struct_size = 8;
  batch_descriptor.abi_version = SQFV_BATCH_ABI_VERSION;
  expect(sqfv_batch_descriptor_view(batch, &batch_descriptor),
         SQFV_UNSUPPORTED_ABI, "undersized batch descriptor output");
  batch_descriptor.struct_size = sizeof(batch_descriptor);
  batch_descriptor.abi_version = SQFV_BATCH_ABI_VERSION + 1;
  expect(sqfv_batch_descriptor_view(batch, &batch_descriptor),
         SQFV_UNSUPPORTED_ABI, "wrong-version batch descriptor output");
  batch_descriptor.abi_version = SQFV_BATCH_ABI_VERSION;
  expect(sqfv_batch_descriptor_view(batch, &batch_descriptor), SQFV_OK,
         "batch descriptor view");
  require(batch_descriptor.batch_sequence == 1,
          "batch descriptor view lost sequence");

  sqfv_lease *wrong_scope = nullptr;
  expect(sqfv_lease_acquire(batch, span(std::string("other-scope")), &wrong_scope),
         SQFV_SCOPE_MISMATCH, "scope mismatch");
  require(wrong_scope == nullptr, "scope mismatch created a lease");

  sqfv_lease *lease = nullptr;
  expect(sqfv_lease_acquire(batch, span(fixture.access_scope), &lease), SQFV_OK,
         "lease acquire");
  sqfv_span view{};
  expect(sqfv_lease_view(lease, &view), SQFV_OK, "lease view");
  require(view.size == 65536 && view.data[0] == 0x36 &&
              view.data[65535] == 0x36,
          "reader observed caller mutation or a short payload");

  sqfv_descriptor retained{};
  retained.struct_size = 8;
  retained.abi_version = SQFV_BATCH_ABI_VERSION;
  expect(sqfv_lease_descriptor_view(lease, &retained), SQFV_UNSUPPORTED_ABI,
         "undersized lease descriptor output");
  retained.struct_size = sizeof(retained);
  retained.abi_version = SQFV_BATCH_ABI_VERSION + 1;
  expect(sqfv_lease_descriptor_view(lease, &retained), SQFV_UNSUPPORTED_ABI,
         "wrong-version lease descriptor output");
  retained.abi_version = SQFV_BATCH_ABI_VERSION;
  expect(sqfv_lease_descriptor_view(lease, &retained), SQFV_OK,
         "lease descriptor view");
  require(retained.batch_sequence == 1 && retained.record_count == 128,
          "lease lost descriptor/cursor");
  require(retained.binding.dataset_revision.size ==
              fixture.dataset_revision.size() &&
              std::memcmp(retained.binding.dataset_revision.data,
                          fixture.dataset_revision.data(),
                          fixture.dataset_revision.size()) == 0,
          "lease lost exact metadata binding");

  uint8_t batch_id[SQFV_BATCH_CONTENT_ID_BYTES]{};
  uint8_t lease_id[SQFV_BATCH_CONTENT_ID_BYTES]{};
  expect(sqfv_batch_content_id(batch, batch_id), SQFV_OK, "batch content ID");
  expect(sqfv_lease_content_id(lease, lease_id), SQFV_OK,
         "lease content ID");
  require(std::memcmp(batch_id, lease_id, sizeof(batch_id)) == 0,
          "lease content identity differs from producer batch");

  std::array<sqfv_lease *, 4> concurrent_leases{};
  for (auto &reader : concurrent_leases)
    expect(sqfv_lease_acquire(batch, span(fixture.access_scope), &reader),
           SQFV_OK, "concurrent reader acquire");
  std::atomic<bool> readers_saw_frozen_bytes{true};
  std::vector<std::thread> readers;
  readers.reserve(concurrent_leases.size());
  for (const auto *reader : concurrent_leases) {
    readers.emplace_back([reader, &readers_saw_frozen_bytes, view] {
      for (unsigned repeat = 0; repeat < 512; ++repeat) {
        sqfv_span observed{};
        if (sqfv_lease_view(reader, &observed) != SQFV_OK ||
            observed.size != view.size || observed.data != view.data ||
            observed.data[repeat] != 0x36 ||
            observed.data[observed.size - 1] != 0x36)
          readers_saw_frozen_bytes.store(false, std::memory_order_relaxed);
      }
    });
  }
  for (auto &reader : readers) reader.join();
  require(readers_saw_frozen_bytes.load(std::memory_order_relaxed),
          "concurrent leases did not see one frozen allocation");
  for (auto *reader : concurrent_leases) sqfv_lease_release(reader);

  sqfv_batch_release(batch);
  sqfv_context_destroy(context);
  expect(sqfv_lease_view(lease, &view), SQFV_OK,
         "view after producer/context release");
  require(view.size == 65536 && view.data[0] == 0x36,
          "lease did not retain the frozen allocation");
  sqfv_lease_release(lease);
}

void global_budget() {
  FixtureDescriptor fixture;
  auto *context = make_context(limits(80U * 1024U));
  sqfv_context_stats baseline{};
  baseline.struct_size = sizeof(baseline);
  baseline.abi_version = SQFV_BATCH_ABI_VERSION;
  expect(sqfv_context_get_stats(context, &baseline), SQFV_OK,
         "baseline allocation stats");
  std::vector<uint8_t> payload(65536, 0x55);
  auto *first = make_batch(context, fixture, 1, payload);
  sqfv_context_stats held{};
  held.struct_size = sizeof(held);
  held.abi_version = SQFV_BATCH_ABI_VERSION;
  expect(sqfv_context_get_stats(context, &held), SQFV_OK,
         "held allocation stats");
  require(held.allocation_bytes > baseline.allocation_bytes &&
              held.allocation_bytes <= held.allocation_limit_bytes,
          "prepared allocation was not charged within its limit");

  auto descriptor = fixture.descriptor(2);
  sqfv_batch *rejected = nullptr;
  expect(sqfv_batch_prepare_copy(context, &descriptor, span(payload), &rejected),
         SQFV_LIMIT, "global allocation bound");
  require(rejected == nullptr, "budget rejection returned a batch");
  sqfv_batch_release(first);
  sqfv_context_stats released{};
  released.struct_size = sizeof(released);
  released.abi_version = SQFV_BATCH_ABI_VERSION;
  expect(sqfv_context_get_stats(context, &released), SQFV_OK,
         "released allocation stats");
  require(released.allocation_bytes == baseline.allocation_bytes,
          "last producer/reader release did not return allocation reservation");
  sqfv_context_destroy(context);
}

void cursor_and_credit_isolation() {
  FixtureDescriptor fixture;
  auto *context = make_context();
  constexpr uint64_t bytes = 65536;
  auto *slow = make_port(context, fixture, 1, bytes * 2, 2);
  auto *fast = make_port(context, fixture, 1, bytes * 2, 2);
  std::vector<uint8_t> payload(bytes, 0x2a);
  auto *first = make_batch(context, fixture, 1, payload);
  expect(sqfv_port_offer(slow, first), SQFV_OK, "slow first offer");
  expect(sqfv_port_offer(fast, first), SQFV_OK, "fast first offer");
  expect(sqfv_port_offer(slow, first), SQFV_DUPLICATE,
         "same-position duplicate");

  payload[0] ^= 0xff;
  auto *changed = make_batch(context, fixture, 1, payload);
  expect(sqfv_port_offer(slow, changed), SQFV_CONFLICT,
         "changed bytes at same position");
  sqfv_batch_release(changed);
  payload[0] ^= 0xff;

  auto *third = make_batch(context, fixture, 3, payload);
  expect(sqfv_port_offer(slow, third), SQFV_GAP, "unannounced gap");
  auto *old = make_batch(context, fixture, 0, payload);
  expect(sqfv_port_offer(slow, old), SQFV_STALE, "older position");
  sqfv_batch_release(old);

  auto *second = make_batch(context, fixture, 2, payload);
  expect(sqfv_port_offer(slow, second), SQFV_OK, "slow second offer");
  expect(sqfv_port_offer(slow, third), SQFV_BLOCKED,
         "full slow pending queue");
  sqfv_port_stats slow_state{};
  slow_state.struct_size = sizeof(slow_state);
  slow_state.abi_version = SQFV_BATCH_ABI_VERSION;
  expect(sqfv_port_get_stats(slow, &slow_state), SQFV_OK, "slow stats");
  require(slow_state.next_sequence == 3 && slow_state.pending_entries == 2 &&
              slow_state.outstanding_bytes == bytes * 2,
          "blocked offer advanced cursor or grew slow credit");

  expect(sqfv_port_offer(fast, second), SQFV_OK, "fast second offer");
  auto *fast_first = take(fast);
  sqfv_lease_release(fast_first);
  auto *fast_second = take(fast);
  sqfv_lease_release(fast_second);
  expect(sqfv_port_offer(fast, third), SQFV_OK,
         "fast branch blocked by slow credit");
  auto *fast_third = take(fast);
  sqfv_lease_release(fast_third);

  auto *slow_first = take(slow);
  expect(sqfv_port_cancel(slow, 2), SQFV_OK,
         "cancel in full-data-pressure state");
  expect(sqfv_port_offer(slow, third), SQFV_OK,
         "blocked offer did not remain retryable");
  sqfv_port_destroy(slow);
  sqfv_port_destroy(fast);
  sqfv_batch_release(first);
  sqfv_batch_release(second);
  sqfv_batch_release(third);
  sqfv_context_destroy(context);
  sqfv_span retained{};
  expect(sqfv_lease_view(slow_first, &retained), SQFV_OK,
         "taken lease after port/context destruction");
  require(retained.size == bytes && retained.data[0] == 0x2a,
          "taken lease lost payload after owner destruction");
  sqfv_lease_release(slow_first);
}

void sequence_exhaustion() {
  FixtureDescriptor fixture;
  auto *context = make_context();
  const auto maximum = std::numeric_limits<uint64_t>::max();
  auto *port = make_port(context, fixture, maximum, 1024, 1);
  std::vector<uint8_t> payload(16, 0x31);
  auto *last = make_batch(context, fixture, maximum, payload);
  expect(sqfv_port_offer(port, last), SQFV_OK, "maximum sequence offer");
  expect(sqfv_port_offer(port, last), SQFV_DUPLICATE,
         "maximum sequence duplicate");

  payload[0] = 0x32;
  auto *changed = make_batch(context, fixture, maximum, payload);
  expect(sqfv_port_offer(port, changed), SQFV_CONFLICT,
         "maximum sequence conflict");
  auto *wrapped = make_batch(context, fixture, 0, payload);
  expect(sqfv_port_offer(port, wrapped), SQFV_STALE,
         "sequence must not wrap to zero");
  sqfv_port_stats stats{};
  stats.struct_size = sizeof(stats);
  stats.abi_version = SQFV_BATCH_ABI_VERSION;
  expect(sqfv_port_get_stats(port, &stats), SQFV_OK,
         "exhausted sequence stats");
  require(stats.sequence_exhausted == 1 && stats.next_sequence == maximum,
          "maximum sequence did not report exhausted cursor");

  sqfv_lease_release(take(port));
  sqfv_batch_release(wrapped);
  sqfv_batch_release(changed);
  sqfv_batch_release(last);
  sqfv_port_destroy(port);
  sqfv_context_destroy(context);
}

void profile_10000() {
  FixtureDescriptor fixture;
  auto *context = make_context(limits(8U * 1024U * 1024U, 2));
  constexpr uint64_t bytes = 65536;
  auto *fast = make_port(context, fixture, 1, 4U * 1024U * 1024U, 64);
  auto *slow = make_port(context, fixture, 1, 4U * 1024U * 1024U, 64);
  std::vector<uint8_t> payload(bytes);
  for (uint64_t i = 0; i < bytes; ++i) payload[i] = static_cast<uint8_t>(i);
  std::vector<uint64_t> batch_latency_us;
  batch_latency_us.reserve(10000);

  const auto start = std::chrono::steady_clock::now();
  for (uint64_t sequence = 1; sequence <= 10000; ++sequence) {
    const auto batch_start = std::chrono::steady_clock::now();
    auto *batch = make_batch(context, fixture, sequence, payload);
    expect(sqfv_port_offer(fast, batch), SQFV_OK, "profile fast offer");
    expect(sqfv_port_offer(slow, batch), SQFV_OK, "profile slow offer");
    auto *fast_lease = take(fast);
    auto *slow_lease = take(slow);
    sqfv_span fast_view{}, slow_view{};
    expect(sqfv_lease_view(fast_lease, &fast_view), SQFV_OK,
           "profile fast view");
    expect(sqfv_lease_view(slow_lease, &slow_view), SQFV_OK,
           "profile slow view");
    require(fast_view.size == bytes && slow_view.size == bytes &&
                fast_view.data == slow_view.data &&
                fast_view.data[sequence % bytes] ==
                    static_cast<uint8_t>(sequence % bytes),
            "profile delivery lost bytes or duplicated the physical payload");
    sqfv_batch_release(batch);
    sqfv_lease_release(fast_lease);
    sqfv_lease_release(slow_lease);
    batch_latency_us.push_back(static_cast<uint64_t>(
        std::chrono::duration_cast<std::chrono::microseconds>(
            std::chrono::steady_clock::now() - batch_start)
            .count()));
  }
  const auto stop = std::chrono::steady_clock::now();
  std::sort(batch_latency_us.begin(), batch_latency_us.end());

  std::vector<sqfv_lease *> stalled;
  stalled.reserve(64);
  for (uint64_t sequence = 10001; sequence <= 10064; ++sequence) {
    auto *batch = make_batch(context, fixture, sequence, payload);
    expect(sqfv_port_offer(fast, batch), SQFV_OK, "stall fast offer");
    expect(sqfv_port_offer(slow, batch), SQFV_OK, "stall slow offer");
    auto *fast_lease = take(fast);
    sqfv_lease_release(fast_lease);
    stalled.push_back(take(slow));
    sqfv_batch_release(batch);
  }
  auto *next = make_batch(context, fixture, 10065, payload);
  expect(sqfv_port_offer(slow, next), SQFV_BLOCKED,
         "stalled slow branch exceeded its byte credit");
  expect(sqfv_port_offer(fast, next), SQFV_OK,
         "stalled slow branch blocked the independent fast branch");
  auto *fast_lease = take(fast);
  sqfv_lease_release(fast_lease);
  sqfv_port_stats fast_state{};
  fast_state.struct_size = sizeof(fast_state);
  fast_state.abi_version = SQFV_BATCH_ABI_VERSION;
  sqfv_port_stats slow_state{};
  slow_state.struct_size = sizeof(slow_state);
  slow_state.abi_version = SQFV_BATCH_ABI_VERSION;
  expect(sqfv_port_get_stats(fast, &fast_state), SQFV_OK,
         "profile fast stats");
  expect(sqfv_port_get_stats(slow, &slow_state), SQFV_OK,
         "profile slow stats");
  require(fast_state.outstanding_bytes == 0 &&
              slow_state.outstanding_bytes == 4U * 1024U * 1024U,
          "profile ports did not retain independent byte credit");
  sqfv_lease_release(stalled.front());
  stalled.erase(stalled.begin());
  expect(sqfv_port_offer(slow, next), SQFV_OK,
         "slow retry remained blocked after credit release");
  sqfv_batch_release(next);

  sqfv_context_stats final{};
  final.struct_size = sizeof(final);
  final.abi_version = SQFV_BATCH_ABI_VERSION;
  expect(sqfv_context_get_stats(context, &final), SQFV_OK,
         "profile allocation stats");
  require(final.peak_allocation_bytes <= final.allocation_limit_bytes &&
              final.allocation_bytes <= final.allocation_limit_bytes,
          "profile exceeded declared global allocation budget");
  const auto elapsed = std::chrono::duration_cast<std::chrono::milliseconds>(
                           stop - start)
                           .count();
  std::cout << "profile_batches=10000 payload_bytes=65536 elapsed_ms="
            << elapsed << " peak_module_bytes=" << final.peak_allocation_bytes
            << " current_module_bytes=" << final.allocation_bytes
            << " stalled_slow_outstanding_bytes="
            << slow_state.outstanding_bytes
            << " stalled_fast_outstanding_bytes="
            << fast_state.outstanding_bytes
            << " batch_latency_us_p50=" << batch_latency_us[4999]
            << " batch_latency_us_p95=" << batch_latency_us[9499]
            << " batch_latency_us_p99=" << batch_latency_us[9899] << '\n';
  sqfv_port_destroy(fast);
  sqfv_port_destroy(slow);
  sqfv_context_destroy(context);
  for (auto *lease : stalled) sqfv_lease_release(lease);
}

} // namespace

int main(int argc, char **argv) {
  try {
    if (argc == 2 && std::string(argv[1]) == "--profile") {
      profile_10000();
    } else if (argc == 1) {
      immutability_and_lifetime();
      global_budget();
      cursor_and_credit_isolation();
      sequence_exhaustion();
    } else {
      throw std::runtime_error("expected no arguments or --profile");
    }
    return 0;
  } catch (const std::exception &error) {
    std::cerr << "sqfv batch test: " << error.what() << '\n';
    return 1;
  }
}
