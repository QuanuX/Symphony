#include "symphony/sqfv/batch.hpp"

#include <algorithm>
#include <array>
#include <cstdint>
#include <iostream>
#include <stdexcept>
#include <string>
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

std::vector<std::uint8_t> hex_bytes(const std::string &hex) {
  require(hex.size() % 2 == 0, "odd hex fixture");
  std::vector<std::uint8_t> result;
  result.reserve(hex.size() / 2);
  for (std::size_t index = 0; index < hex.size(); index += 2) {
    result.push_back(static_cast<std::uint8_t>(
        std::stoul(hex.substr(index, 2), nullptr, 16)));
  }
  return result;
}

struct Fixture {
  std::vector<std::uint8_t> payload{0x10, 0x20, 0x30};

  Descriptor descriptor() const {
    Descriptor result;
    result.binding = {"m", "d", "s", "l", "a"};
    result.partition = "p";
    result.source_binding = "b";
    result.source_position = std::string("\0\xff" "A", 3);
    result.producer_generation[0] = 1;
    result.batch_sequence = 7;
    result.record_count = 2;
    return result;
  }
};

Limits limits(std::uint64_t payload = 3,
              std::uint64_t frame = 117,
              std::uint64_t descriptor = 58) {
  return {payload, frame, descriptor, 4096, 1};
}

Context make_context(const Limits &selected) {
  Context context;
  expect(Context::create(selected, context), Status::ok, "create context");
  require(static_cast<bool>(context), "context handle missing");
  return context;
}

Batch make_batch(const Context &context, const Fixture &fixture) {
  Batch batch;
  expect(context.prepare_copy(fixture.descriptor(), ByteView(fixture.payload),
                              batch),
         Status::ok, "prepare fixture batch");
  require(static_cast<bool>(batch), "batch handle missing");
  return batch;
}

// Independent Python hashlib vectors over the precise domain, canonical
// descriptor, prefix and payload. They do not depend on encoder output.
constexpr const char *kGoldenFrame =
    "53514631000100000000000000000072000000030000003a"
    "00016d00016400017300016c000161000170000162000300ff41"
    "01000000000000000000000000000000"
    "00000000000000070000000000000002"
    "51ac8c13ca16ad794e35775d097cebaf48e795fb4851b709a8fdeb7b81197533"
    "102030";
constexpr const char *kGoldenContentId =
    "7589379f3046a7768b9181634adb3305bf921e7b5fcce6e92f5281981eb4a189";

void golden_round_trip_and_exact_limits() {
  Fixture fixture;
  auto context = make_context(limits());
  auto batch = make_batch(context, fixture);

  std::size_t measured = 0;
  expect(frame_measure(context, batch, measured), Status::ok, "measure frame");
  require(measured == 117, "golden frame size changed");
  std::vector<std::uint8_t> frame(measured, 0xa5);
  std::size_t written = 999;
  expect(frame_encode(context, batch, MutableBytes(frame.data(), measured - 1),
                      written),
         Status::limit, "one byte short output");
  require(written == 999 &&
              std::all_of(frame.begin(), frame.end(),
                          [](std::uint8_t value) { return value == 0xa5; }),
          "short output changed caller output");
  expect(frame_encode(context, batch, MutableBytes(frame), written), Status::ok,
         "encode exact frame limit");
  require(written == measured && frame == hex_bytes(kGoldenFrame),
          "frame differs from independent golden vector");

  const auto expected_id = hex_bytes(kGoldenContentId);
  require(std::equal(expected_id.begin(), expected_id.end(),
                     batch.content_id().begin()),
          "content digest differs from independent golden vector");

  Batch decoded;
  expect(frame_decode(context, ByteView(frame), decoded), Status::ok,
         "decode golden frame");
  require(static_cast<bool>(decoded), "decode returned no batch");
  const auto &result = decoded.descriptor();
  require(result.batch_sequence == 7 && result.record_count == 2 &&
              result.source_position == fixture.descriptor().source_position,
          "decoded cursor or opaque source position changed");
  Lease lease;
  expect(decoded.acquire("a", lease), Status::ok, "decoded read lease");
  frame.back() ^= 0xff;
  const ByteView frozen = lease.payload();
  require(frozen.size() == fixture.payload.size() && frozen[2] == 0x30,
          "decoded batch borrowed mutable caller frame");
}

void rejection_matrix() {
  auto context = make_context(limits(10, 256, 128));
  const auto frame = hex_bytes(kGoldenFrame);
  for (std::size_t count = 0; count < frame.size(); ++count) {
    Batch decoded;
    expect(frame_decode(context, ByteView(frame.data(), count), decoded),
           Status::corrupt_frame, "every truncation boundary");
    require(!decoded, "truncated frame published a batch");
  }
  auto bad = frame;
  bad.push_back(0);
  Batch decoded;
  expect(frame_decode(context, ByteView(bad), decoded), Status::corrupt_frame,
         "trailing byte");

  auto reject = [&](std::size_t offset, std::uint8_t mask, Status wanted,
                    const char *name) {
    auto mutated = frame;
    mutated[offset] ^= mask;
    Batch result;
    expect(frame_decode(context, ByteView(mutated), result), wanted, name);
    require(!result, "invalid frame published a batch");
  };
  reject(0, 1, Status::corrupt_frame, "bad magic");
  reject(5, 1, Status::unsupported_frame, "unsupported major");
  reject(7, 1, Status::unsupported_frame, "unsupported minor");
  reject(11, 1, Status::unsupported_frame, "unknown required flag");
  reject(15, 1, Status::corrupt_frame, "bad header length");
  reject(19, 1, Status::corrupt_frame, "bad payload length");
  reject(23, 1, Status::corrupt_frame, "bad descriptor length");
  reject(26, 1, Status::corrupt_frame, "corrupted descriptor value");
  reject(58, 1, Status::corrupt_frame, "corrupted generation");
  reject(82, 1, Status::corrupt_frame, "corrupted integrity field");
  reject(116, 1, Status::corrupt_frame, "corrupted payload");

  bad = frame;
  std::fill(bad.begin() + 12, bad.begin() + 24, 0xff);
  expect(frame_decode(context, ByteView(bad), decoded), Status::limit,
         "forged oversized declared lengths");
  bad = frame;
  bad[24] = 0xff;
  bad[25] = 0xff;
  expect(frame_decode(context, ByteView(bad), decoded), Status::corrupt_frame,
         "truncated descriptor field");
}

void configured_max_plus_one() {
  const auto frame = hex_bytes(kGoldenFrame);
  Batch decoded;
  auto frame_limited = make_context(limits(3, 116, 58));
  expect(frame_decode(frame_limited, ByteView(frame), decoded), Status::limit,
         "frame maximum plus one");
  auto payload_limited = make_context(limits(2, 117, 58));
  expect(frame_decode(payload_limited, ByteView(frame), decoded), Status::limit,
         "payload maximum plus one");
  auto descriptor_limited = make_context(limits(3, 117, 57));
  expect(frame_decode(descriptor_limited, ByteView(frame), decoded),
         Status::limit, "descriptor maximum plus one");

  Fixture fixture;
  fixture.payload.push_back(0x40);
  auto context = make_context(limits());
  Batch oversized;
  expect(context.prepare_copy(fixture.descriptor(), ByteView(fixture.payload),
                              oversized),
         Status::limit, "prepare payload maximum plus one");
  require(!oversized, "oversized payload created a batch");
}

void fixture_profile_boundary_and_streaming_digest() {
  Fixture fixture;
  fixture.payload.resize(65536);
  for (std::size_t index = 0; index < fixture.payload.size(); ++index) {
    fixture.payload[index] = static_cast<std::uint8_t>(index);
  }
  auto context = make_context({65536, 65650, 58, 1024U * 1024U, 1});
  auto batch = make_batch(context, fixture);
  std::size_t measured = 0;
  expect(frame_measure(context, batch, measured), Status::ok,
         "measure 64 KiB fixture");
  require(measured == 65650, "64 KiB fixture frame size changed");
  std::vector<std::uint8_t> frame(measured);
  std::size_t written = 0;
  expect(frame_encode(context, batch, MutableBytes(frame), written), Status::ok,
         "encode 64 KiB fixture at exact frame bound");
  require(written == measured, "64 KiB frame was not fully written");
  const auto expected_digest = hex_bytes(
      "c9da41dd8150ebd9e39a1b97f5b4f1ac3ab84bda0a8efa8ba52f5b053e08b45d");
  require(std::equal(expected_digest.begin(), expected_digest.end(),
                     frame.begin() + 82),
          "streaming digest disagrees with independent 64 KiB vector");
  const auto expected_id = hex_bytes(
      "7721db1900ea185ee06633f0208be72e0a5688e6cda5452378bc941e5ae8ffe0");
  require(std::equal(expected_id.begin(), expected_id.end(),
                     batch.content_id().begin()),
          "streaming content ID disagrees with independent 64 KiB vector");
  Batch decoded;
  expect(frame_decode(context, ByteView(frame), decoded), Status::ok,
         "decode 64 KiB frame at exact bound");
  require(static_cast<bool>(decoded), "64 KiB decode returned no batch");

  fixture.payload.push_back(0x00);
  Batch oversized;
  expect(context.prepare_copy(fixture.descriptor(), ByteView(fixture.payload),
                              oversized),
         Status::limit, "64 KiB payload plus one");
  require(!oversized, "oversized 64 KiB payload was published");
}

}  // namespace

int main() {
  try {
    golden_round_trip_and_exact_limits();
    rejection_matrix();
    configured_max_plus_one();
    fixture_profile_boundary_and_streaming_digest();
    return 0;
  } catch (const std::exception &error) {
    std::cerr << "sqfv frame test: " << error.what() << '\n';
    return 1;
  }
}
