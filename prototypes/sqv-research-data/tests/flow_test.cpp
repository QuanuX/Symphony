#include "symphony/sqv/flow.hpp"

#include <cstdint>
#include <optional>
#include <stdexcept>
#include <utility>
#include <vector>

using namespace symphony::sqv::prototype;

namespace {
void check(bool condition) {
  if (!condition) throw std::runtime_error("flow test assertion failed");
}

Descriptor descriptor(std::uint64_t first, const char* scope = "research.local") {
  return {.dataset_revision = "fixture-revision-1",
          .schema_name = "fixture.record",
          .schema_version = "1.0",
          .representation = "raw.v1",
          .access_scope = scope,
          .provenance = "fixture-a",
          .source_position = "source-opaque-1",
          .generation = 1,
          .first = first,
          .last = first,
          .partition = 0,
          .record_count = 1};
}
}  // namespace

int main() {
  const std::vector<std::uint8_t> payload{1, 2, 3, 4};
  const auto first = PreparedBatch::prepare(descriptor(1), payload);
  const auto second = PreparedBatch::prepare(descriptor(2), payload);
  Fanout flow;
  flow.add_port("fast", "research.local", 8, 2);
  flow.add_port("slow", "research.local", 4, 1);
  flow.add_port("private", "private.account", 4, 1);

  check(flow.offer("fast", first) == OfferResult::accepted);
  check(flow.offer("slow", first) == OfferResult::accepted);
  check(flow.offer("slow", second) == OfferResult::blocked);
  check(flow.offer("fast", second) == OfferResult::accepted);
  check(flow.offer("private", first) == OfferResult::scope_mismatch);
  check(flow.offer("missing", first) == OfferResult::unknown_port);
  check(flow.outstanding_bytes("fast") == 8);
  check(flow.outstanding_bytes("slow") == 4);

  {
    auto fast_first = flow.take("fast");
    check(fast_first.has_value() && fast_first->descriptor().first == 1);
    check(fast_first->bytes()[0] == 1);
    check(flow.outstanding_bytes("fast") == 8);  // Processing retains credit.
  }
  check(flow.outstanding_bytes("fast") == 4);
  check(flow.offer("fast", first) == OfferResult::accepted);
  check(flow.offer("fast", first) == OfferResult::blocked);

  {
    auto slow_first = flow.take("slow");
    check(slow_first.has_value());
    check(flow.offer("slow", second) == OfferResult::blocked);
  }
  check(flow.outstanding_bytes("slow") == 0);
  check(flow.offer("slow", second) == OfferResult::accepted);
  auto slow_second = flow.take("slow");
  check(slow_second.has_value() && slow_second->descriptor().first == 2);

  Fanout oversize;
  oversize.add_port("tiny", "research.local", 3, 1);
  check(oversize.offer("tiny", first) == OfferResult::oversize);
  check(oversize.outstanding_bytes("tiny") == 0);
  Fanout queue_limited;
  queue_limited.add_port("one", "research.local", 8, 1);
  check(queue_limited.offer("one", first) == OfferResult::accepted);
  check(queue_limited.offer("one", second) == OfferResult::blocked);
  check(queue_limited.outstanding_bytes("one") == 4);

  // A taken delivery keeps its accounting state and payload valid after the
  // producer's Fanout object is destroyed.
  std::optional<Delivery> detached;
  {
    Fanout temporary;
    temporary.add_port("reader", "research.local", 4, 1);
    check(temporary.offer("reader", first) == OfferResult::accepted);
    auto taken = temporary.take("reader");
    check(taken.has_value());
    detached.emplace(std::move(*taken));
  }
  check(detached.has_value() && detached->bytes()[3] == 4);
}
