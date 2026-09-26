#include "symphony/sqfv/batch.hpp"

#include <algorithm>
#include <array>
#include <atomic>
#include <chrono>
#include <cstdint>
#include <iostream>
#include <limits>
#include <stdexcept>
#include <string>
#include <thread>
#include <vector>

namespace {

using namespace symphony::sqfv;

void require(bool condition, const char *message) {
  if (!condition) throw std::runtime_error(message);
}

void expect(Status actual, Status wanted, const char *operation) {
  if (actual != wanted) {
    throw std::runtime_error(std::string(operation) + ": got " +
                             std::to_string(static_cast<int>(actual)) +
                             ", want " +
                             std::to_string(static_cast<int>(wanted)));
  }
}

ByteView bytes(const std::vector<std::uint8_t> &payload) {
  return {payload.data(), payload.size()};
}

struct FixtureDescriptor {
  Binding binding{"fixture-metadata-revision-1", "fixture-dataset-revision-1",
                  "fixture-schema-1", "fixture-layout-1",
                  "fixture-research-scope"};
  std::string partition = "fixture-partition-0";
  std::string source_binding = "fixture-source-binding";
  std::string source_position = "source-native-position";
  Generation generation{1};

  Descriptor descriptor(std::uint64_t sequence) const {
    Descriptor result;
    result.binding = binding;
    result.partition = partition;
    result.source_binding = source_binding;
    result.source_position = source_position;
    result.producer_generation = generation;
    result.batch_sequence = sequence;
    result.record_count = 128;
    return result;
  }

  PortConfig port_config(std::uint64_t next_sequence,
                         std::uint64_t byte_credit,
                         std::uint32_t max_pending) const {
    PortConfig result;
    result.binding = binding;
    result.partition = partition;
    result.producer_generation = generation;
    result.next_sequence = next_sequence;
    result.outstanding_byte_credit = byte_credit;
    result.max_pending_entries = max_pending;
    return result;
  }
};

Limits limits(std::uint64_t allocation_bytes = 8U * 1024U * 1024U,
              std::uint32_t max_ports = 3) {
  return {65536, 73728, 4096, allocation_bytes, max_ports};
}

Context make_context(Limits selected = limits()) {
  Context context;
  expect(Context::create(selected, context), Status::ok, "context create");
  require(static_cast<bool>(context), "context create returned no handle");
  return context;
}

Batch make_batch(const Context &context, const FixtureDescriptor &fixture,
                 std::uint64_t sequence,
                 const std::vector<std::uint8_t> &payload) {
  Batch batch;
  const Descriptor descriptor = fixture.descriptor(sequence);
  expect(context.prepare_copy(descriptor, bytes(payload), batch), Status::ok,
         "batch prepare");
  require(static_cast<bool>(batch), "batch prepare returned no handle");
  return batch;
}

Port make_port(const Context &context, const FixtureDescriptor &fixture,
               std::uint64_t next_sequence, std::uint64_t credit,
               std::uint32_t pending) {
  Port port;
  expect(context.add_port(fixture.port_config(next_sequence, credit, pending),
                          port),
         Status::ok, "port add");
  require(static_cast<bool>(port), "port add returned no handle");
  return port;
}

Lease take(Port &port) {
  Lease lease;
  expect(port.take(lease), Status::ok, "port take");
  require(static_cast<bool>(lease), "port take returned no lease");
  return lease;
}

void immutability_and_lifetime() {
  Limits invalid = limits();
  invalid.max_payload_bytes = 0;
  Context rejected;
  expect(Context::create(invalid, rejected), Status::invalid_argument,
         "zero payload ceiling");
  require(!rejected, "invalid limits returned a context");

  FixtureDescriptor fixture;
  Context context = make_context();
  std::vector<std::uint8_t> payload(65536, 0x36);
  Batch batch = make_batch(context, fixture, 1, payload);
  std::fill(payload.begin(), payload.end(), 0x90);

  Lease wrong_scope;
  expect(batch.acquire("other-scope", wrong_scope), Status::scope_mismatch,
         "scope mismatch");
  require(!wrong_scope, "scope mismatch created a lease");

  Lease lease;
  expect(batch.acquire(fixture.binding.access_scope, lease), Status::ok,
         "lease acquire");
  const ByteView view = lease.payload();
  require(view.size() == 65536 && view.front() == 0x36 &&
              view.back() == 0x36,
          "reader observed caller mutation or a short payload");
  require(lease.descriptor().batch_sequence == 1 &&
              lease.descriptor().record_count == 128 &&
              lease.descriptor().binding.dataset_revision ==
                  fixture.binding.dataset_revision,
          "lease lost descriptor or exact binding");
  require(batch.content_id() == lease.content_id(),
          "lease content identity differs from batch");

  Batch retained;
  expect(batch.retain(retained), Status::ok, "batch retain");
  require(retained.descriptor().batch_sequence == 1 &&
              retained.content_id() == batch.content_id(),
          "retained producer handle lost identity");

  std::array<Lease, 4> concurrent_leases;
  for (auto &reader : concurrent_leases)
    expect(batch.acquire(fixture.binding.access_scope, reader), Status::ok,
           "concurrent reader acquire");
  std::atomic<bool> readers_saw_frozen_bytes{true};
  std::vector<std::thread> readers;
  readers.reserve(concurrent_leases.size());
  for (auto &reader : concurrent_leases) {
    Lease *const active = &reader;
    readers.emplace_back([active, &readers_saw_frozen_bytes, view] {
      for (unsigned repeat = 0; repeat < 512; ++repeat) {
        const ByteView observed = active->payload();
        if (observed.size() != view.size() ||
            observed.data() != view.data() ||
            observed[repeat] != 0x36 || observed.back() != 0x36)
          readers_saw_frozen_bytes.store(false, std::memory_order_relaxed);
      }
    });
  }
  for (auto &reader : readers) reader.join();
  require(readers_saw_frozen_bytes.load(std::memory_order_relaxed),
          "concurrent leases did not see one frozen allocation");
  for (auto &reader : concurrent_leases) reader.reset();

  batch = Batch{};
  retained = Batch{};
  context = Context{};
  require(lease.payload().size() == 65536 &&
              lease.payload().front() == 0x36,
          "lease did not survive producer and context destruction");
  lease.reset();
}

void global_budget() {
  FixtureDescriptor fixture;
  Context context = make_context(limits(80U * 1024U));
  ContextStats baseline;
  expect(context.stats(baseline), Status::ok, "baseline allocation stats");
  std::vector<std::uint8_t> payload(65536, 0x55);
  Batch first = make_batch(context, fixture, 1, payload);
  ContextStats held;
  expect(context.stats(held), Status::ok, "held allocation stats");
  require(held.allocation_bytes > baseline.allocation_bytes &&
              held.allocation_bytes <= held.allocation_limit_bytes,
          "prepared allocation was not charged within its limit");

  Batch rejected;
  expect(context.prepare_copy(fixture.descriptor(2), bytes(payload), rejected),
         Status::limit, "global allocation bound");
  require(!rejected, "budget rejection returned a batch");
  first = Batch{};
  ContextStats released;
  expect(context.stats(released), Status::ok, "released allocation stats");
  require(released.allocation_bytes == baseline.allocation_bytes,
          "last handle release did not return reservation");
}

void cursor_and_credit_isolation() {
  FixtureDescriptor fixture;
  Context context = make_context();
  constexpr std::uint64_t payload_bytes = 65536;
  Port slow = make_port(context, fixture, 1, payload_bytes * 2, 2);
  Port fast = make_port(context, fixture, 1, payload_bytes * 2, 2);
  std::vector<std::uint8_t> payload(payload_bytes, 0x2a);
  Batch first = make_batch(context, fixture, 1, payload);
  expect(slow.offer(first), Status::ok, "slow first offer");
  expect(fast.offer(first), Status::ok, "fast first offer");
  expect(slow.offer(first), Status::duplicate, "same-position duplicate");

  payload[0] ^= 0xff;
  Batch changed = make_batch(context, fixture, 1, payload);
  expect(slow.offer(changed), Status::conflict,
         "changed bytes at same position");
  changed = Batch{};
  payload[0] ^= 0xff;

  Batch third = make_batch(context, fixture, 3, payload);
  expect(slow.offer(third), Status::gap, "unannounced gap");
  Batch old = make_batch(context, fixture, 0, payload);
  expect(slow.offer(old), Status::stale, "older position");
  old = Batch{};

  Batch second = make_batch(context, fixture, 2, payload);
  expect(slow.offer(second), Status::ok, "slow second offer");
  expect(slow.offer(third), Status::blocked, "full slow pending queue");
  PortStats slow_state;
  expect(slow.stats(slow_state), Status::ok, "slow stats");
  require(slow_state.next_sequence == 3 && slow_state.pending_entries == 2 &&
              slow_state.outstanding_bytes == payload_bytes * 2,
          "blocked offer advanced cursor or grew slow credit");

  expect(fast.offer(second), Status::ok, "fast second offer");
  Lease fast_first = take(fast);
  fast_first.reset();
  Lease fast_second = take(fast);
  fast_second.reset();
  expect(fast.offer(third), Status::ok,
         "fast branch blocked by slow credit");
  Lease fast_third = take(fast);
  fast_third.reset();

  Lease slow_first = take(slow);
  expect(slow.cancel(2), Status::ok, "cancel under full data pressure");
  expect(slow.offer(third), Status::ok,
         "blocked offer did not remain retryable");
  slow.reset();
  fast.reset();
  first = Batch{};
  second = Batch{};
  third = Batch{};
  context = Context{};
  require(slow_first.payload().size() == payload_bytes &&
              slow_first.payload().front() == 0x2a,
          "taken lease lost payload after owner destruction");
  slow_first.reset();
}

void sequence_exhaustion() {
  FixtureDescriptor fixture;
  Context context = make_context();
  const auto maximum = std::numeric_limits<std::uint64_t>::max();
  Port port = make_port(context, fixture, maximum, 1024, 1);
  std::vector<std::uint8_t> payload(16, 0x31);
  Batch last = make_batch(context, fixture, maximum, payload);
  expect(port.offer(last), Status::ok, "maximum sequence offer");
  expect(port.offer(last), Status::duplicate,
         "maximum sequence duplicate");
  payload[0] = 0x32;
  Batch changed = make_batch(context, fixture, maximum, payload);
  expect(port.offer(changed), Status::conflict,
         "maximum sequence conflict");
  Batch wrapped = make_batch(context, fixture, 0, payload);
  expect(port.offer(wrapped), Status::stale,
         "sequence must not wrap to zero");
  PortStats stats;
  expect(port.stats(stats), Status::ok, "exhausted sequence stats");
  require(stats.sequence_exhausted && stats.next_sequence == maximum,
          "maximum sequence did not report exhausted cursor");
  Lease received = take(port);
  received.reset();
}

void profile_10000() {
  FixtureDescriptor fixture;
  Context context = make_context(limits(8U * 1024U * 1024U, 2));
  constexpr std::uint64_t payload_bytes = 65536;
  Port fast = make_port(context, fixture, 1, 4U * 1024U * 1024U, 64);
  Port slow = make_port(context, fixture, 1, 4U * 1024U * 1024U, 64);
  std::vector<std::uint8_t> payload(payload_bytes);
  for (std::uint64_t i = 0; i < payload_bytes; ++i)
    payload[i] = static_cast<std::uint8_t>(i);
  std::vector<std::uint64_t> batch_latency_us;
  batch_latency_us.reserve(10000);

  const auto start = std::chrono::steady_clock::now();
  for (std::uint64_t sequence = 1; sequence <= 10000; ++sequence) {
    const auto batch_start = std::chrono::steady_clock::now();
    Batch batch = make_batch(context, fixture, sequence, payload);
    expect(fast.offer(batch), Status::ok, "profile fast offer");
    expect(slow.offer(batch), Status::ok, "profile slow offer");
    Lease fast_lease = take(fast);
    Lease slow_lease = take(slow);
    const ByteView fast_view = fast_lease.payload();
    const ByteView slow_view = slow_lease.payload();
    require(fast_view.size() == payload_bytes &&
                slow_view.size() == payload_bytes &&
                fast_view.data() == slow_view.data() &&
                fast_view[sequence % payload_bytes] ==
                    static_cast<std::uint8_t>(sequence % payload_bytes) &&
                fast_lease.descriptor().batch_sequence == sequence &&
                slow_lease.descriptor().batch_sequence == sequence &&
                fast_lease.content_id() == slow_lease.content_id(),
            "profile delivery lost bytes, identity, or shared payload");
    fast_lease.reset();
    slow_lease.reset();
    batch_latency_us.push_back(static_cast<std::uint64_t>(
        std::chrono::duration_cast<std::chrono::microseconds>(
            std::chrono::steady_clock::now() - batch_start)
            .count()));
  }
  const auto stop = std::chrono::steady_clock::now();
  std::sort(batch_latency_us.begin(), batch_latency_us.end());

  std::vector<Lease> stalled;
  stalled.reserve(64);
  for (std::uint64_t sequence = 10001; sequence <= 10064; ++sequence) {
    Batch batch = make_batch(context, fixture, sequence, payload);
    expect(fast.offer(batch), Status::ok, "stall fast offer");
    expect(slow.offer(batch), Status::ok, "stall slow offer");
    Lease fast_lease = take(fast);
    fast_lease.reset();
    stalled.push_back(take(slow));
  }
  Batch next = make_batch(context, fixture, 10065, payload);
  expect(slow.offer(next), Status::blocked,
         "stalled slow branch exceeded its byte credit");
  expect(fast.offer(next), Status::ok,
         "stalled slow branch blocked independent fast branch");
  Lease fast_lease = take(fast);
  fast_lease.reset();
  PortStats fast_state, slow_state;
  expect(fast.stats(fast_state), Status::ok, "profile fast stats");
  expect(slow.stats(slow_state), Status::ok, "profile slow stats");
  require(fast_state.outstanding_bytes == 0 &&
              slow_state.outstanding_bytes == 4U * 1024U * 1024U,
          "profile ports did not retain independent byte credit");
  stalled.front().reset();
  stalled.erase(stalled.begin());
  expect(slow.offer(next), Status::ok,
         "slow retry remained blocked after credit release");

  ContextStats final;
  expect(context.stats(final), Status::ok, "profile allocation stats");
  require(final.peak_allocation_bytes <= final.allocation_limit_bytes &&
              final.allocation_bytes <= final.allocation_limit_bytes,
          "profile exceeded declared global reservation budget");
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
  fast.reset();
  slow.reset();
  context = Context{};
  for (auto &lease : stalled) lease.reset();
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
