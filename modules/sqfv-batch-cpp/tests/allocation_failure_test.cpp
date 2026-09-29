#include "symphony/sqfv/batch.hpp"

#include <atomic>
#include <cstdio>
#include <cstdlib>
#include <new>
#include <string>
#include <vector>

// Fault injection is confined to this test executable. The selected allocation
// fails once; assertions and cleanup run after injection has been disabled.
namespace {
std::atomic<int> fail_at{-1};
std::atomic<int> allocation_calls{0};
}

void* operator new(std::size_t size) {
  const auto selected = fail_at.load();
  if (selected >= 0 && allocation_calls.fetch_add(1) == selected)
    throw std::bad_alloc();
  if (void* result = std::malloc(size == 0 ? 1 : size)) return result;
  throw std::bad_alloc();
}

void* operator new[](std::size_t size) { return ::operator new(size); }
void operator delete(void* pointer) noexcept { std::free(pointer); }
void operator delete[](void* pointer) noexcept { std::free(pointer); }

namespace {

using namespace symphony::sqfv;

void require(bool condition, const char* message) {
  if (condition) return;
  std::fprintf(stderr, "sqfv allocation failure test: %s\n", message);
  std::abort();
}

Limits limits() { return {65536, 73728, 4096, 8U * 1024U * 1024U, 3}; }

Descriptor descriptor(std::uint64_t sequence) {
  Descriptor result;
  // Force separately allocated strings on the selected standard library so
  // failures inside partial descriptor/port construction are exercised.
  result.binding = {std::string(80, 'm'), std::string(80, 'd'),
                    std::string(80, 's'), std::string(80, 'l'),
                    std::string(80, 'a')};
  result.partition = std::string(80, 'p');
  result.source_binding = std::string(80, 'b');
  result.source_position = std::string(80, 'x');
  result.producer_generation[0] = 1;
  result.batch_sequence = sequence;
  result.record_count = 1;
  return result;
}

PortConfig port_config(const Descriptor& selected, std::uint64_t next) {
  return {selected.binding, selected.partition, selected.producer_generation,
          next, 65536, 8};
}

ContextStats stats(const Context& context) {
  ContextStats result;
  require(context.stats(result) == Status::ok, "context stats failed");
  return result;
}

enum class Operation { create, prepare, port, retain, acquire, offer, decode };

void exercise(Operation operation, unsigned& total_rejections) {
  unsigned rejected = 0;
  bool reached_success = false;
  // Each iteration allows one more allocation to succeed. Stop only when the
  // operation succeeds, proving that every preceding allocation was injected.
  for (int fail_index = 0; fail_index != 64; ++fail_index) {
    const auto input = descriptor(0);
    const auto output_descriptor = descriptor(42);
    const auto initial_port = port_config(input, 0);
    const auto replacement_port = port_config(input, 10);
    const std::vector<std::uint8_t> payload(1024, 0x42);
    Context context;
    Context context_output;
    require(Context::create(limits(), context) == Status::ok, "context setup");
    require(Context::create(limits(), context_output) == Status::ok,
            "output context setup");
    Batch batch;
    Batch batch_output;
    Batch original_context_batch;
    require(context.prepare_copy(input, payload, batch) == Status::ok,
            "input batch setup");
    require(context.prepare_copy(output_descriptor, payload, batch_output) ==
                Status::ok,
            "output batch setup");
    require(context_output.prepare_copy(input, payload, original_context_batch) ==
                Status::ok,
            "output context identity setup");
    Port port;
    require(context.add_port(initial_port, port) == Status::ok, "port setup");
    Lease lease_output;
    require(batch_output.acquire(input.binding.access_scope, lease_output) ==
                Status::ok,
            "output lease setup");
    std::size_t frame_size = 0;
    require(frame_measure(context, batch, frame_size) == Status::ok,
            "frame size setup");
    std::vector<std::uint8_t> frame(frame_size);
    require(frame_encode(context, batch, frame, frame_size) == Status::ok,
            "frame setup");
    const auto before = stats(context);
    const auto output_context_before = stats(context_output);
    const auto* previous_id = &batch_output.content_id();
    const auto* previous_payload = lease_output.payload().data();

    allocation_calls = 0;
    fail_at = fail_index;
    Status result = Status::internal_error;
    switch (operation) {
      case Operation::create:
        result = Context::create(limits(), context_output);
        break;
      case Operation::prepare:
        result = context.prepare_copy(input, payload, batch_output);
        break;
      case Operation::port:
        result = context.add_port(replacement_port, port);
        break;
      case Operation::retain:
        result = batch.retain(batch_output);
        break;
      case Operation::acquire:
        result = batch.acquire(input.binding.access_scope, lease_output);
        break;
      case Operation::offer:
        result = port.offer(batch);
        break;
      case Operation::decode:
        result = frame_decode(context, frame, batch_output);
        break;
    }
    fail_at = -1;

    if (result == Status::ok) {
      reached_success = true;
      break;
    }
    require(result == Status::no_memory, "bad_alloc was not contained");
    ++rejected;
    const auto after = stats(context);
    const auto output_context_after = stats(context_output);
    require(before.allocation_bytes == after.allocation_bytes &&
                before.live_ports == after.live_ports,
            "failure leaked reservation or port claim");
    require(output_context_before.allocation_bytes ==
                output_context_after.allocation_bytes,
            "failure changed output context accounting");
    require(&batch_output.content_id() == previous_id &&
                batch_output.descriptor().batch_sequence == 42 &&
                lease_output.payload().data() == previous_payload &&
                lease_output.descriptor().batch_sequence == 42,
            "failure replaced an owning output");
    require(frame_measure(context_output, original_context_batch, frame_size) ==
                Status::ok,
            "failure replaced output context identity");
    PortStats port_state;
    require(port.stats(port_state) == Status::ok &&
                port_state.pending_entries == 0 &&
                port_state.next_sequence == 0 &&
                port_state.outstanding_bytes == 0,
            "failure changed port output, cursor, queue, or credit");
  }
  require(reached_success && rejected != 0,
          "allocation sweep did not exercise failure and success");
  total_rejections += rejected;
}

void release_without_allocation() {
  Context context;
  require(Context::create(limits(), context) == Status::ok,
          "control context setup");
  const auto baseline = stats(context);
  const auto first = descriptor(0);
  Port port;
  require(context.add_port(port_config(first, 0), port) == Status::ok,
          "control port setup");
  const std::vector<std::uint8_t> payload(1024, 0x42);
  for (std::uint64_t sequence = 0; sequence != 3; ++sequence) {
    Batch batch;
    require(context.prepare_copy(descriptor(sequence), payload, batch) ==
                Status::ok &&
                port.offer(batch) == Status::ok,
            "control queue setup");
  }
  Lease lease;
  allocation_calls = 0;
  fail_at = 0;
  const auto taken = port.take(lease);
  const auto cancelled = port.cancel(1);
  port.reset();
  const bool survived = lease && lease.payload().front() == 0x42;
  lease.reset();
  const auto calls = allocation_calls.load();
  fail_at = -1;
  require(taken == Status::ok && cancelled == Status::ok && survived &&
              calls == 0,
          "release/control path required a new allocation");
  const auto final = stats(context);
  require(final.allocation_bytes == baseline.allocation_bytes &&
              final.live_ports == 0,
          "control release did not return exact ownership charges");
}

}  // namespace

int main() {
  unsigned rejected = 0;
  for (const auto operation : {Operation::create, Operation::prepare,
                               Operation::port, Operation::retain,
                               Operation::acquire, Operation::offer,
                               Operation::decode})
    exercise(operation, rejected);
  release_without_allocation();
  std::printf("allocation failures contained with unchanged outputs and current "
              "accounting: %u; successful terminal controls: 7\n",
              rejected);
}
