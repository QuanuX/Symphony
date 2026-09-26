#include "symphony/sqfv/batch.h"

#include <algorithm>
#include <array>
#include <cstdint>
#include <cstring>
#include <iostream>
#include <stdexcept>
#include <string>
#include <vector>

namespace {

void require(bool condition, const char *message) {
  if (!condition) throw std::runtime_error(message);
}

void expect(sqfv_status actual, sqfv_status wanted, const char *operation) {
  if (actual != wanted) {
    throw std::runtime_error(std::string(operation) + ": got " +
                             std::to_string(static_cast<int>(actual)) +
                             ", want " +
                             std::to_string(static_cast<int>(wanted)));
  }
}

sqfv_span span(const std::string &value) {
  return {reinterpret_cast<const std::uint8_t *>(value.data()), value.size()};
}

template <std::size_t N>
sqfv_span span(const std::array<std::uint8_t, N> &value) {
  return {value.data(), value.size()};
}

sqfv_span span(const std::vector<std::uint8_t> &value) {
  return {value.data(), value.size()};
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
  std::string metadata_ref = "m";
  std::string dataset_revision = "d";
  std::string schema_version = "s";
  std::string layout_version = "l";
  std::string access_scope = "a";
  std::string partition = "p";
  std::string source_binding = "b";
  std::array<std::uint8_t, 3> source_position{0x00, 0xff, 0x41};
  std::vector<std::uint8_t> payload{0x10, 0x20, 0x30};

  sqfv_descriptor descriptor() const {
    sqfv_descriptor result{};
    result.struct_size = sizeof(result);
    result.abi_version = SQFV_BATCH_ABI_VERSION;
    result.binding = {sizeof(sqfv_binding), SQFV_BATCH_ABI_VERSION,
                      span(metadata_ref), span(dataset_revision),
                      span(schema_version), span(layout_version),
                      span(access_scope)};
    result.partition = span(partition);
    result.source_binding = span(source_binding);
    result.source_position = span(source_position);
    result.producer_generation[0] = 1;
    result.batch_sequence = 7;
    result.record_count = 2;
    return result;
  }
};

sqfv_limits limits(std::uint64_t payload = 3,
                   std::uint64_t frame = 117,
                   std::uint64_t descriptor = 58) {
  return {sizeof(sqfv_limits), SQFV_BATCH_ABI_VERSION, payload, frame,
          descriptor, 4096, 1, 0};
}

sqfv_context *make_context(sqfv_limits selected) {
  sqfv_context *context = nullptr;
  expect(sqfv_context_create(&selected, &context), SQFV_OK, "create context");
  require(context != nullptr, "context handle missing");
  return context;
}

sqfv_batch *make_batch(sqfv_context *context, const Fixture &fixture) {
  auto descriptor = fixture.descriptor();
  sqfv_batch *batch = nullptr;
  expect(sqfv_batch_prepare_copy(context, &descriptor, span(fixture.payload),
                                 &batch),
         SQFV_OK, "prepare fixture batch");
  require(batch != nullptr, "batch handle missing");
  return batch;
}

// Independently computed with Python hashlib over the specified domain,
// canonical descriptor, prefix and payload; this is not an encode/decode loop.
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
  auto *context = make_context(limits());
  auto *batch = make_batch(context, fixture);

  std::uint64_t measured = 0;
  expect(sqfv_frame_measure(context, batch, &measured), SQFV_OK,
         "measure frame");
  require(measured == 117, "golden frame size changed");
  std::vector<std::uint8_t> frame(measured, 0xa5);
  std::uint64_t written = 0;
  expect(sqfv_frame_encode(context, batch, frame.data(), measured - 1,
                           &written),
         SQFV_LIMIT, "one byte short output");
  require(written == measured &&
              std::all_of(frame.begin(), frame.end(),
                          [](std::uint8_t value) { return value == 0xa5; }),
          "short output wrote bytes or lost required size");
  expect(sqfv_frame_encode(context, batch, frame.data(), frame.size(),
                           &written),
         SQFV_OK, "encode exact frame limit");
  const auto expected_frame = hex_bytes(kGoldenFrame);
  require(written == measured, "encoded frame length differs from measure");
  if (frame != expected_frame) {
    const std::size_t first = [&] {
      for (std::size_t index = 0;
           index < std::min(frame.size(), expected_frame.size()); ++index) {
        if (frame[index] != expected_frame[index]) return index;
      }
      return std::min(frame.size(), expected_frame.size());
    }();
    std::string actual_digest;
    constexpr char hex[] = "0123456789abcdef";
    for (std::size_t index = 82; index < 114; ++index) {
      actual_digest += hex[frame[index] >> 4U];
      actual_digest += hex[frame[index] & 0x0fU];
    }
    throw std::runtime_error("frame differs from independent golden vector at " +
                             std::to_string(first) + " (actual size " +
                             std::to_string(frame.size()) + ", expected size " +
                             std::to_string(expected_frame.size()) +
                             ", actual digest " + actual_digest + ")");
  }

  std::uint8_t content_id[SQFV_BATCH_CONTENT_ID_BYTES]{};
  expect(sqfv_batch_content_id(batch, content_id), SQFV_OK,
         "batch content ID");
  const auto expected_id = hex_bytes(kGoldenContentId);
  require(std::equal(expected_id.begin(), expected_id.end(), content_id),
          "content digest differs from independent golden vector");

  sqfv_batch *decoded = nullptr;
  expect(sqfv_frame_decode(context, span(frame), &decoded), SQFV_OK,
         "decode golden frame");
  require(decoded != nullptr, "decode returned no batch");
  sqfv_descriptor result{};
  result.struct_size = sizeof(result);
  result.abi_version = SQFV_BATCH_ABI_VERSION;
  expect(sqfv_batch_descriptor_view(decoded, &result), SQFV_OK,
         "decoded descriptor");
  require(result.batch_sequence == 7 && result.record_count == 2 &&
              result.source_position.size == fixture.source_position.size() &&
              std::memcmp(result.source_position.data,
                          fixture.source_position.data(),
                          fixture.source_position.size()) == 0,
          "decoded cursor or opaque source position changed");
  sqfv_lease *lease = nullptr;
  expect(sqfv_lease_acquire(decoded, span(fixture.access_scope), &lease), SQFV_OK,
         "decoded read lease");
  frame.back() ^= 0xff;
  sqfv_span frozen{};
  expect(sqfv_lease_view(lease, &frozen), SQFV_OK, "decoded frozen view");
  require(frozen.size == fixture.payload.size() && frozen.data[2] == 0x30,
          "decoded batch borrowed mutable caller frame");
  sqfv_lease_release(lease);
  sqfv_batch_release(decoded);
  sqfv_batch_release(batch);
  sqfv_context_destroy(context);
}

void rejection_matrix() {
  Fixture fixture;
  auto *context = make_context(limits(10, 256, 128));
  auto *batch = make_batch(context, fixture);
  std::vector<std::uint8_t> frame = hex_bytes(kGoldenFrame);
  for (std::size_t count = 0; count < frame.size(); ++count) {
    sqfv_batch *decoded = nullptr;
    expect(sqfv_frame_decode(context, {frame.data(), count}, &decoded),
           SQFV_CORRUPT_FRAME, "every truncation boundary");
    require(decoded == nullptr, "truncated frame published a batch");
  }
  auto bad = frame;
  bad.push_back(0);
  sqfv_batch *decoded = nullptr;
  expect(sqfv_frame_decode(context, span(bad), &decoded),
         SQFV_CORRUPT_FRAME, "trailing byte");

  auto reject = [&](std::size_t offset, std::uint8_t mask,
                    sqfv_status wanted, const char *name) {
    auto mutated = frame;
    mutated[offset] ^= mask;
    sqfv_batch *result = nullptr;
    expect(sqfv_frame_decode(context, span(mutated), &result), wanted, name);
    require(result == nullptr, "invalid frame published a batch");
  };
  reject(0, 1, SQFV_CORRUPT_FRAME, "bad magic");
  reject(5, 1, SQFV_UNSUPPORTED_FRAME, "unsupported major");
  reject(7, 1, SQFV_UNSUPPORTED_FRAME, "unsupported minor");
  reject(11, 1, SQFV_UNSUPPORTED_FRAME, "unknown required flag");
  reject(15, 1, SQFV_CORRUPT_FRAME, "bad header length");
  reject(19, 1, SQFV_CORRUPT_FRAME, "bad payload length");
  reject(23, 1, SQFV_CORRUPT_FRAME, "bad descriptor length");
  reject(26, 1, SQFV_CORRUPT_FRAME, "corrupted descriptor value");
  reject(58, 1, SQFV_CORRUPT_FRAME, "corrupted generation");
  reject(82, 1, SQFV_CORRUPT_FRAME, "corrupted integrity field");
  reject(116, 1, SQFV_CORRUPT_FRAME, "corrupted payload");

  bad = frame;
  std::fill(bad.begin() + 12, bad.begin() + 24, 0xff);
  expect(sqfv_frame_decode(context, span(bad), &decoded), SQFV_LIMIT,
         "forged oversized declared lengths");
  bad = frame;
  bad[24] = 0xff;
  bad[25] = 0xff;
  expect(sqfv_frame_decode(context, span(bad), &decoded),
         SQFV_CORRUPT_FRAME, "truncated descriptor field");

  sqfv_batch_release(batch);
  sqfv_context_destroy(context);
}

void configured_max_plus_one() {
  const auto frame = hex_bytes(kGoldenFrame);
  auto *frame_limited = make_context(limits(3, 116, 58));
  sqfv_batch *decoded = nullptr;
  expect(sqfv_frame_decode(frame_limited, span(frame), &decoded), SQFV_LIMIT,
         "frame maximum plus one");
  sqfv_context_destroy(frame_limited);

  auto *payload_limited = make_context(limits(2, 117, 58));
  expect(sqfv_frame_decode(payload_limited, span(frame), &decoded), SQFV_LIMIT,
         "payload maximum plus one");
  sqfv_context_destroy(payload_limited);

  auto *descriptor_limited = make_context(limits(3, 117, 57));
  expect(sqfv_frame_decode(descriptor_limited, span(frame), &decoded),
         SQFV_LIMIT, "descriptor maximum plus one");
  sqfv_context_destroy(descriptor_limited);

  Fixture fixture;
  fixture.payload.push_back(0x40);
  auto *context = make_context(limits());
  auto descriptor = fixture.descriptor();
  sqfv_batch *oversized = nullptr;
  expect(sqfv_batch_prepare_copy(context, &descriptor, span(fixture.payload),
                                 &oversized),
         SQFV_LIMIT, "prepare payload maximum plus one");
  require(oversized == nullptr, "oversized payload created a batch");
  sqfv_context_destroy(context);
}

void fixture_profile_boundary_and_streaming_digest() {
  Fixture fixture;
  fixture.payload.resize(65536);
  for (std::size_t index = 0; index < fixture.payload.size(); ++index) {
    fixture.payload[index] = static_cast<std::uint8_t>(index);
  }
  auto *context = make_context(
      {sizeof(sqfv_limits), SQFV_BATCH_ABI_VERSION, 65536, 65650, 58,
       1024U * 1024U, 1, 0});
  auto *batch = make_batch(context, fixture);
  std::uint64_t measured = 0;
  expect(sqfv_frame_measure(context, batch, &measured), SQFV_OK,
         "measure 64 KiB fixture");
  require(measured == 65650, "64 KiB fixture frame size changed");
  std::vector<std::uint8_t> frame(measured);
  std::uint64_t written = 0;
  expect(sqfv_frame_encode(context, batch, frame.data(), frame.size(),
                           &written),
         SQFV_OK, "encode 64 KiB fixture at exact frame bound");
  require(written == measured, "64 KiB frame was not fully written");
  const auto expected_digest = hex_bytes(
      "c9da41dd8150ebd9e39a1b97f5b4f1ac3ab84bda0a8efa8ba52f5b053e08b45d");
  require(std::equal(expected_digest.begin(), expected_digest.end(),
                     frame.begin() + 82),
          "streaming digest disagrees with independent 64 KiB vector");
  const auto expected_id = hex_bytes(
      "7721db1900ea185ee06633f0208be72e0a5688e6cda5452378bc941e5ae8ffe0");
  std::uint8_t id[SQFV_BATCH_CONTENT_ID_BYTES]{};
  expect(sqfv_batch_content_id(batch, id), SQFV_OK,
         "64 KiB batch content ID");
  require(std::equal(expected_id.begin(), expected_id.end(), id),
          "streaming content ID disagrees with independent 64 KiB vector");
  sqfv_batch *decoded = nullptr;
  expect(sqfv_frame_decode(context, span(frame), &decoded), SQFV_OK,
         "decode 64 KiB frame at exact bound");
  require(decoded != nullptr, "64 KiB decode returned no batch");
  sqfv_batch_release(decoded);

  fixture.payload.push_back(0x00);
  auto descriptor = fixture.descriptor();
  sqfv_batch *oversized = nullptr;
  expect(sqfv_batch_prepare_copy(context, &descriptor, span(fixture.payload),
                                 &oversized),
         SQFV_LIMIT, "64 KiB payload plus one");
  require(oversized == nullptr, "oversized 64 KiB payload was published");
  sqfv_batch_release(batch);
  sqfv_context_destroy(context);
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
