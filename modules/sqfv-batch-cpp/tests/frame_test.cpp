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

void reject_preserving_output(Context &context, ByteView frame,
                              Status wanted, Batch &output,
                              const char *operation) {
  require(static_cast<bool>(output), "rejection sentinel is empty");
  const Descriptor *original_descriptor = &output.descriptor();
  const ContentId original_id = output.content_id();
  ContextStats before;
  expect(context.stats(before), Status::ok, "stats before rejected decode");
  expect(frame_decode(context, frame, output), wanted, operation);
  ContextStats after;
  expect(context.stats(after), Status::ok, "stats after rejected decode");
  require(output && &output.descriptor() == original_descriptor &&
              output.content_id() == original_id,
          "rejected decode replaced the existing output batch");
  require(after.allocation_bytes == before.allocation_bytes &&
              after.live_ports == before.live_ports,
          "rejected decode leaked context reservations");
}

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
  auto decoded = make_batch(context, Fixture{});
  for (std::size_t count = 0; count < frame.size(); ++count) {
    reject_preserving_output(context, ByteView(frame.data(), count),
                             Status::corrupt_frame, decoded,
                             "every truncation boundary");
  }
  auto bad = frame;
  bad.push_back(0);
  reject_preserving_output(context, ByteView(bad), Status::corrupt_frame,
                           decoded, "trailing byte");

  auto reject = [&](std::size_t offset, std::uint8_t mask, Status wanted,
                    const char *name) {
    auto mutated = frame;
    mutated[offset] ^= mask;
    reject_preserving_output(context, ByteView(mutated), wanted, decoded, name);
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
  reject_preserving_output(context, ByteView(bad), Status::limit, decoded,
                           "forged oversized declared lengths");
  bad = frame;
  bad[24] = 0xff;
  bad[25] = 0xff;
  reject_preserving_output(context, ByteView(bad), Status::corrupt_frame,
                           decoded, "truncated descriptor field");
}

void digest_valid_semantic_rejections() {
  // Each digest was independently computed with Python hashlib over the
  // changed prefix/descriptor and unchanged payload. Keeping a valid digest
  // ensures these cases reach descriptor validation after integrity checking.
  struct EmptyField {
    std::size_t value_offset;
    const char *digest;
    const char *name;
  };
  constexpr EmptyField empty_fields[] = {
      {26, "a702aee1c2b8c2ca90b4ae8005c50aa4b76161c130903c42e7aefa046f50d3f7", "empty metadata reference"},
      {29, "608ef4452078b94835e208d2edee75c0898dc22743229cb8112956306801f088", "empty dataset revision"},
      {32, "084fa412b8ea3a903f73c58eacdaa49f2b49d294889aca3273ade69fffd1aebf", "empty schema version"},
      {35, "7e551b2dba425bd8d378a0db77e0492a3e6c1eb48b9a4e2c85d0ba393a77f928", "empty layout version"},
      {38, "2486ead60f2b4caf3bba81103e6d3b65a4e67cac7ac5ae119bbdf0c3b1857507", "empty access scope"},
      {41, "763c256656e317c011abcd2968b69ca270e3e9d1f56aa9d5ef200a1a7426ecc0", "empty partition"},
      {44, "f24e2e3867a8d314e7b9da7099d04a72dc01d5f6250f72260ec5fcfb8a16af9e", "source position without binding"},
  };
  auto context = make_context(limits());
  auto output = make_batch(context, Fixture{});
  for (const auto &test : empty_fields) {
    auto frame = hex_bytes(kGoldenFrame);
    frame[test.value_offset - 1] = 0; // u16 field length: one byte to zero.
    frame.erase(frame.begin() + test.value_offset);
    --frame[15]; // Header and descriptor lengths both shrink by one.
    --frame[23];
    const auto digest = hex_bytes(test.digest);
    std::copy(digest.begin(), digest.end(), frame.begin() + 81);
    reject_preserving_output(context, ByteView(frame), Status::corrupt_frame,
                             output, test.name);
  }

  struct ZeroField {
    std::size_t offset;
    std::size_t width;
    const char *digest;
    const char *name;
  };
  constexpr ZeroField zero_fields[] = {
      {50, 16, "46012b1ee475beec39cb45dd8d501509e69a89e0a2c068518e6ffef637d3b166", "zero producer generation"},
      {74, 8, "0f78f0e81973bf6ff36425718996d07067fecb66bc22fda1301298d103c8facf", "zero record count"},
  };
  for (const auto &test : zero_fields) {
    auto frame = hex_bytes(kGoldenFrame);
    std::fill_n(frame.begin() + test.offset, test.width, 0);
    const auto digest = hex_bytes(test.digest);
    std::copy(digest.begin(), digest.end(), frame.begin() + 82);
    reject_preserving_output(context, ByteView(frame), Status::corrupt_frame,
                             output, test.name);
  }
}

void decode_reservation_failure_preserves_output() {
  auto calibration = make_context(limits());
  ContextStats base;
  expect(calibration.stats(base), Status::ok, "measure context reservation");
  auto batch = make_batch(calibration, Fixture{});
  ContextStats prepared;
  expect(calibration.stats(prepared), Status::ok, "measure batch reservation");
  const auto batch_charge = prepared.allocation_bytes - base.allocation_bytes;
  require(batch_charge > 0, "batch has no reservation charge");

  auto selected = limits();
  selected.global_allocation_bytes = prepared.allocation_bytes + batch_charge - 1;
  auto context = make_context(selected);
  auto output = make_batch(context, Fixture{});
  const auto frame = hex_bytes(kGoldenFrame);
  reject_preserving_output(context, ByteView(frame), Status::limit, output,
                           "decode one reservation unit short");
}

void sha256_padding_boundaries() {
  struct Golden {
    std::size_t payload_size;
    const char *frame_digest;
    const char *content_id;
  };
  // Independent hashlib vectors: payload bytes are 0, 1, ..., n-1.
  // Frame hashing adds 111 bytes before the payload, so n=8/9 crosses its
  // 55/56-byte padding boundary. n=55/56 crosses the payload hash boundary.
  constexpr Golden goldens[] = {
      {8, "234fc70cf4483db219193e1507de35dab3c74d40da10dac8fa801df02c581d43",
       "8f92dba4f3c53fee58a9c86f9be6e88959b42f9be411861af7f3217b22e1e53c"},
      {9, "5cb1f3773b31f437ef975599e2673f6f0e26223b01e9468e1dc6f9f69389e6f0",
       "cd7332068d1f3b158f63261ca6b932eca20930fa6357481f49dfa1e6e3675253"},
      {55, "412d6bf8c4ee367b077212a6bf247fd37c9200a59f7ea3342738c1de8006980a",
       "d21e1c52e74c03e450a41ba98441367190d0baf29326e4c637527bf546a7fa69"},
      {56, "d6ab8b48a9a56fa962cc058ee405f11477716f0e0f526312603ead1e94aebd1b",
       "741250d9efd36d1fcc7a395aa6472abaeb08cfd926988297ce1d6a75fb3b291e"},
  };
  auto context = make_context(limits(56, 170, 58));
  for (const auto &golden : goldens) {
    Fixture fixture;
    fixture.payload.resize(golden.payload_size);
    for (std::size_t index = 0; index < fixture.payload.size(); ++index)
      fixture.payload[index] = static_cast<std::uint8_t>(index);
    auto batch = make_batch(context, fixture);
    std::vector<std::uint8_t> frame(114 + golden.payload_size);
    std::size_t written = 0;
    expect(frame_encode(context, batch, MutableBytes(frame), written),
           Status::ok, "encode SHA padding boundary");
    require(written == frame.size(), "padding-boundary frame size changed");
    const auto expected_digest = hex_bytes(golden.frame_digest);
    const auto expected_id = hex_bytes(golden.content_id);
    require(std::equal(expected_digest.begin(), expected_digest.end(),
                       frame.begin() + 82),
            "frame digest differs at SHA padding boundary");
    require(std::equal(expected_id.begin(), expected_id.end(),
                       batch.content_id().begin()),
            "content ID differs at SHA padding boundary");
    Batch decoded;
    expect(frame_decode(context, ByteView(frame), decoded), Status::ok,
           "decode SHA padding boundary");
    require(decoded.content_id() == batch.content_id(),
            "decode changed content ID at SHA padding boundary");
  }
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
    digest_valid_semantic_rejections();
    decode_reservation_failure_preserves_output();
    sha256_padding_boundaries();
    configured_max_plus_one();
    fixture_profile_boundary_and_streaming_digest();
    return 0;
  } catch (const std::exception &error) {
    std::cerr << "sqfv frame test: " << error.what() << '\n';
    return 1;
  }
}
