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
#include <utility>
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

void retained_context_budget() {
  FixtureDescriptor fixture;
  std::vector<std::uint8_t> payload(16, 0x55);
  Context probe = make_context();
  Batch measured = make_batch(probe, fixture, 1, payload);
  ContextStats exact;
  expect(probe.stats(exact), Status::ok, "measure exact batch budget");
  measured = Batch{};
  probe = Context{};

  Context context = make_context(limits(exact.allocation_bytes));
  Batch batch = make_batch(context, fixture, 1, payload);
  Lease rejected;
  expect(batch.acquire(fixture.binding.access_scope, rejected), Status::limit,
         "exact budget rejects another lease");
  context = Context{};
  // Returning only the Context handle's reservation cannot fund a read lease
  // in this package. Returning the retained shared state as well could fund
  // two, even though that state is still required by this batch.
  expect(batch.acquire(fixture.binding.access_scope, rejected), Status::limit,
         "retained shared context must remain charged");
  require(!rejected && batch.descriptor().batch_sequence == 1,
          "retained context budget failure changed an output");
}

void empty_move_and_replace() {
  FixtureDescriptor fixture;
  std::vector<std::uint8_t> payload(16, 0x45);
  Context context = make_context();
  Batch batch = make_batch(context, fixture, 1, payload);
  Lease lease;
  expect(batch.acquire(fixture.binding.access_scope, lease), Status::ok,
         "replacement lease setup");
  const auto id = batch.content_id();
  const auto* original_payload = lease.payload().data();

  Context empty_context;
  Batch empty_batch;
  Port empty_port;
  Lease empty_lease;
  ContextStats context_sentinel{11, 12, 13, 14};
  PortStats port_sentinel{11, 12, 13, 14, 15, true};
  expect(empty_context.stats(context_sentinel), Status::invalid_argument,
         "empty context stats");
  expect(empty_context.prepare_copy(fixture.descriptor(2), payload, batch),
         Status::invalid_argument, "empty context prepare");
  Port live = make_port(context, fixture, 1, 32, 2);
  expect(empty_context.add_port(fixture.port_config(1, 32, 2), live),
         Status::invalid_argument, "empty context port");
  expect(empty_batch.retain(batch), Status::invalid_argument,
         "empty batch retain");
  expect(empty_batch.acquire(fixture.binding.access_scope, lease),
         Status::invalid_argument, "empty batch acquire");
  expect(live.offer(empty_batch), Status::invalid_argument,
         "offer empty batch");
  expect(empty_port.offer(batch), Status::invalid_argument,
         "empty port offer");
  expect(empty_port.take(lease), Status::invalid_argument, "empty port take");
  expect(empty_port.cancel(1), Status::invalid_argument, "empty port cancel");
  expect(empty_port.stats(port_sentinel), Status::invalid_argument,
         "empty port stats");
  require(context_sentinel.allocation_bytes == 11 &&
              context_sentinel.live_ports == 14 &&
              port_sentinel.next_sequence == 11 &&
              port_sentinel.sequence_exhausted && batch.content_id() == id &&
              lease.payload().data() == original_payload && live &&
              empty_lease.payload().empty(),
          "empty input failure changed output or empty payload");

  Batch moved = std::move(batch);
  require(!batch && moved.content_id() == id,
          "batch move lost ownership");
  expect(batch.retain(empty_batch), Status::invalid_argument,
         "moved-from batch retain");
  expect(moved.retain(moved), Status::ok, "self retain");
  expect(context.prepare_copy(moved.descriptor(), lease.payload(), moved),
         Status::ok, "replace with borrowed descriptor and payload");
  require(moved.content_id() == id, "alias replacement changed content");

  expect(live.offer(moved), Status::ok, "replace lease first offer");
  expect(live.take(lease), Status::ok, "replace existing direct lease");
  Batch second = make_batch(context, fixture, 2, payload);
  expect(live.offer(second), Status::ok, "replace lease second offer");
  expect(live.take(lease), Status::ok, "replace existing port lease");
  PortStats stats;
  expect(live.stats(stats), Status::ok, "replacement port stats");
  require(stats.outstanding_bytes == payload.size() &&
              stats.pending_entries == 0 &&
              lease.descriptor().batch_sequence == 2,
          "lease replacement did not release previous credit");
  Lease moved_lease = std::move(lease);
  require(!lease && lease.payload().empty() &&
              moved_lease.descriptor().batch_sequence == 2,
          "lease move lost ownership");
  moved_lease.reset();
  Port moved_port = std::move(live);
  expect(live.take(lease), Status::invalid_argument, "moved-from port take");
  Context moved_context = std::move(context);
  expect(context.stats(context_sentinel), Status::invalid_argument,
         "moved-from context stats");
  expect(moved_context.stats(context_sentinel), Status::ok,
         "moved context stats");
  moved_port.reset();
  moved_port.reset();
}

void binding_validation() {
  FixtureDescriptor fixture;
  std::vector<std::uint8_t> payload(16, 0x32);
  Context context = make_context(limits(8U * 1024U * 1024U, 1));
  Batch retained = make_batch(context, fixture, 1, payload);
  const auto original_id = retained.content_id();
  auto descriptor = fixture.descriptor(1);
  descriptor.producer_generation = {};
  expect(context.prepare_copy(descriptor, payload, retained),
         Status::invalid_argument, "zero generation");
  descriptor = fixture.descriptor(1);
  descriptor.record_count = 0;
  expect(context.prepare_copy(descriptor, payload, retained),
         Status::invalid_argument, "zero records");
  descriptor = fixture.descriptor(1);
  descriptor.source_binding.clear();
  expect(context.prepare_copy(descriptor, payload, retained),
         Status::invalid_argument, "source position without binding");
  descriptor = fixture.descriptor(1);
  descriptor.partition.assign(4096, 'p');
  expect(context.prepare_copy(descriptor, payload, retained), Status::limit,
         "combined descriptor ceiling");
  require(retained.content_id() == original_id,
          "descriptor rejection replaced a batch");

  Port port = make_port(context, fixture, 1, 32, 2);
  Port rejected;
  expect(context.add_port(fixture.port_config(1, 32, 2), rejected),
         Status::limit, "context port count");
  auto changed = fixture;
  changed.binding.access_scope += "different";
  Batch wrong_scope = make_batch(context, changed, 1, payload);
  expect(port.offer(wrong_scope), Status::scope_mismatch, "port scope binding");
  changed = fixture;
  changed.binding.dataset_revision += "different";
  Batch wrong_binding = make_batch(context, changed, 1, payload);
  expect(port.offer(wrong_binding), Status::binding_mismatch,
         "port metadata binding");
  changed = fixture;
  changed.generation[0] = 2;
  Batch wrong_generation = make_batch(context, changed, 1, payload);
  expect(port.offer(wrong_generation), Status::binding_mismatch,
         "port generation binding");
  Context another = make_context();
  Batch foreign = make_batch(another, fixture, 1, payload);
  expect(port.offer(foreign), Status::invalid_argument, "foreign context");
  PortStats stats;
  expect(port.stats(stats), Status::ok, "rejected binding stats");
  require(!rejected && stats.next_sequence == 1 && stats.pending_entries == 0 &&
              stats.outstanding_bytes == 0,
          "binding rejection consumed cursor or credit");
}

void concurrent_flow() {
  FixtureDescriptor fixture;
  Context context = make_context();
  Port port = make_port(context, fixture, 1, 4096, 4);
  constexpr unsigned count = 4096;
  std::atomic<bool> failed{false};
  std::atomic<bool> readers_done{false};
  std::atomic<unsigned> received{0};
  const auto deadline = std::chrono::steady_clock::now() +
                        std::chrono::seconds(30);
  std::thread consumer([&] {
    Lease lease;
    for (unsigned sequence = 1; sequence <= count && !failed.load();) {
      const auto result = port.take(lease);
      if (result == Status::empty) {
        if (std::chrono::steady_clock::now() > deadline) {
          failed = true;
          break;
        }
        std::this_thread::yield();
        continue;
      }
      if (result != Status::ok || lease.descriptor().batch_sequence != sequence ||
          lease.payload().size() != 1024 ||
          lease.payload().front() != static_cast<std::uint8_t>(sequence)) {
        failed = true;
        break;
      }
      // Replacing this live lease on the next take must return its old credit.
      received = sequence++;
    }
  });
  std::thread observer([&] {
    while (!readers_done.load() && !failed.load()) {
      ContextStats global;
      PortStats local;
      if (context.stats(global) != Status::ok ||
          port.stats(local) != Status::ok ||
          global.allocation_bytes > global.allocation_limit_bytes ||
          local.outstanding_bytes > local.outstanding_byte_credit ||
          local.pending_entries > local.max_pending_entries) {
        failed = true;
        break;
      }
      std::this_thread::yield();
    }
  });
  std::vector<std::uint8_t> payload(1024);
  for (unsigned sequence = 1; sequence <= count && !failed.load(); ++sequence) {
    payload.front() = static_cast<std::uint8_t>(sequence);
    Batch batch;
    if (context.prepare_copy(fixture.descriptor(sequence), payload, batch) !=
        Status::ok) {
      failed = true;
      break;
    }
    while (!failed.load()) {
      const auto result = port.offer(batch);
      if (result == Status::ok) break;
      if (result != Status::blocked ||
          std::chrono::steady_clock::now() > deadline) {
        failed = true;
        break;
      }
      std::this_thread::yield();
    }
  }
  consumer.join();
  readers_done = true;
  observer.join();
  require(!failed.load() && received.load() == count,
          "concurrent flow lost a delivery, raced accounting, or failed");
  PortStats final;
  expect(port.stats(final), Status::ok, "concurrent final stats");
  require(final.pending_entries == 0 && final.outstanding_bytes == 0 &&
              final.next_sequence == count + 1,
          "concurrent flow did not return all credits");
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
      retained_context_budget();
      empty_move_and_replace();
      binding_validation();
      concurrent_flow();
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
