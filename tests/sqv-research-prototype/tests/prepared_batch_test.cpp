#include "symphony/sqv/prepared_batch.hpp"

#include <array>
#include <atomic>
#include <cstdint>
#include <functional>
#include <stdexcept>
#include <string>
#include <thread>
#include <vector>

using symphony::sqv::prototype::Descriptor;
using symphony::sqv::prototype::PreparedBatch;
using symphony::sqv::prototype::decode;
using symphony::sqv::prototype::encode;

namespace {

void check(bool condition) {
  if (!condition) throw std::runtime_error("test assertion failed");
}

void rejects(const std::function<void()>& action) {
  bool rejected = false;
  try { action(); } catch (const std::invalid_argument&) { rejected = true; }
  check(rejected);
}

Descriptor fixture() {
  return {.dataset_revision = "fred-GDP-vintage-2020-01-01",
          .schema_name = "fixture.observation",
          .schema_version = "1.0",
          .representation = "raw.v1",
          .access_scope = "research.local",
          .provenance = "capture-a",
          .source_position = "fred:series:GDP:observation:1",
          .generation = 7,
          .first = 42,
          .last = 43,
          .partition = 2,
          .record_count = 2};
}

}  // namespace

int main() {
  auto source = std::vector<std::uint8_t>{0, 1, 2, 0xff};
  auto batch = PreparedBatch::prepare(fixture(), source);
  source[0] = 99;  // The producer's mutable input cannot change published bytes.
  const auto lifetime = batch.lifetime();
  std::atomic<bool> readers_agree{true};
  auto read_many = [&] {
    const auto lease = batch.acquire("research.local");
    for (int iteration = 0; iteration < 1000; ++iteration) {
      const auto bytes = lease.bytes();
      if (bytes.size() != 4 || bytes[0] != 0 || bytes[3] != 0xff)
        readers_agree.store(false, std::memory_order_relaxed);
    }
  };
  std::thread left(read_many);
  std::thread right(read_many);
  left.join();
  right.join();
  check(readers_agree.load(std::memory_order_relaxed));
  {
    const auto reader_a = batch.acquire("research.local");
    const auto reader_b = batch.acquire("research.local");
    check(reader_a.bytes()[0] == 0);
    check(reader_a.bytes().data() == reader_b.bytes().data());
    rejects([&] { (void)batch.acquire("private.account"); });
    const auto frame = encode(batch);
    const auto decoded = decode(frame);
    const auto decoded_reader = decoded.acquire("research.local");
    check(decoded.descriptor() == batch.descriptor());
    check(std::vector<std::uint8_t>(decoded_reader.bytes().begin(), decoded_reader.bytes().end()) ==
          std::vector<std::uint8_t>(reader_a.bytes().begin(), reader_a.bytes().end()));
    for (std::size_t length = 0; length < frame.size(); ++length)
      rejects([&] { (void)decode(std::span(frame.data(), length)); });

    auto invalid = frame;
    invalid[4] = 2;
    rejects([&] { (void)decode(invalid); });
    invalid = frame;
    invalid[8] = 0xff;
    rejects([&] { (void)decode(invalid); });
    invalid = frame;
    invalid[12] = 0xff;  // Declared payload size exceeds the maximum.
    rejects([&] { (void)decode(invalid); });
    invalid = frame;
    invalid[48] = 0;
    invalid[49] = 0;  // Required dataset revision cannot be empty.
    rejects([&] { (void)decode(invalid); });
    invalid = frame;
    invalid.pop_back();
    rejects([&] { (void)decode(invalid); });
    invalid = frame;
    invalid.push_back(0);
    rejects([&] { (void)decode(invalid); });
    invalid = frame;
    invalid[48] = 0xff;  // First string's length exceeds its bound.
    rejects([&] { (void)decode(invalid); });

    auto newer = fixture();
    newer.dataset_revision = "fred-GDP-vintage-2020-02-01";
    newer.schema_version = "2.0";
    const auto newer_batch = PreparedBatch::prepare(newer, source);
    const auto newer_decoded = decode(encode(newer_batch));
    check(newer_decoded.descriptor().dataset_revision != decoded.descriptor().dataset_revision);
    check(newer_decoded.descriptor().schema_version != decoded.descriptor().schema_version);
    batch = {};
    check(!lifetime.expired());  // Readers still retain the frozen payload.
  }
  check(lifetime.expired());

  auto bad = fixture();
  bad.record_count = 3;
  rejects([&] { (void)PreparedBatch::prepare(bad, source); });
  bad = fixture();
  bad.access_scope = "private account";
  rejects([&] { (void)PreparedBatch::prepare(bad, source); });
  const auto too_large = std::vector<std::uint8_t>(
      symphony::sqv::prototype::max_payload_bytes + 1, 0);
  rejects([&] { (void)PreparedBatch::prepare(fixture(), too_large); });
}
