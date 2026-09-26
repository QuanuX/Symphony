#include "batch_internal.hpp"

#include <algorithm>
#include <array>
#include <cstddef>
#include <cstdint>
#include <cstring>
#include <limits>

namespace sqfv_internal {
namespace {

// A streaming implementation keeps frame and content digests on the stack.
// The shared foundation digest copies its entire input before hashing and is
// unsuitable for this allocation-accounted data path.
constexpr std::array<std::uint32_t, 64> kRoundConstants = {
    0x428a2f98U, 0x71374491U, 0xb5c0fbcfU, 0xe9b5dba5U,
    0x3956c25bU, 0x59f111f1U, 0x923f82a4U, 0xab1c5ed5U,
    0xd807aa98U, 0x12835b01U, 0x243185beU, 0x550c7dc3U,
    0x72be5d74U, 0x80deb1feU, 0x9bdc06a7U, 0xc19bf174U,
    0xe49b69c1U, 0xefbe4786U, 0x0fc19dc6U, 0x240ca1ccU,
    0x2de92c6fU, 0x4a7484aaU, 0x5cb0a9dcU, 0x76f988daU,
    0x983e5152U, 0xa831c66dU, 0xb00327c8U, 0xbf597fc7U,
    0xc6e00bf3U, 0xd5a79147U, 0x06ca6351U, 0x14292967U,
    0x27b70a85U, 0x2e1b2138U, 0x4d2c6dfcU, 0x53380d13U,
    0x650a7354U, 0x766a0abbU, 0x81c2c92eU, 0x92722c85U,
    0xa2bfe8a1U, 0xa81a664bU, 0xc24b8b70U, 0xc76c51a3U,
    0xd192e819U, 0xd6990624U, 0xf40e3585U, 0x106aa070U,
    0x19a4c116U, 0x1e376c08U, 0x2748774cU, 0x34b0bcb5U,
    0x391c0cb3U, 0x4ed8aa4aU, 0x5b9cca4fU, 0x682e6ff3U,
    0x748f82eeU, 0x78a5636fU, 0x84c87814U, 0x8cc70208U,
    0x90befffaU, 0xa4506cebU, 0xbef9a3f7U, 0xc67178f2U,
};

constexpr std::uint32_t rotate_right(std::uint32_t value,
                                     unsigned count) noexcept {
  return (value >> count) | (value << (32U - count));
}

class Sha256 final {
 public:
  bool update(const std::uint8_t* input, std::size_t size) noexcept {
    if (size != 0 && input == nullptr) return false;
    constexpr std::uint64_t kMaxBytes =
        std::numeric_limits<std::uint64_t>::max() / 8U;
    if (size > kMaxBytes - bytes_seen_) return false;
    bytes_seen_ += size;
    while (size != 0) {
      const std::size_t take = std::min(size, block_.size() - used_);
      std::memcpy(block_.data() + used_, input, take);
      used_ += take;
      input += take;
      size -= take;
      if (used_ == block_.size()) {
        transform();
        used_ = 0;
      }
    }
    return true;
  }

  void finish(std::uint8_t output[32]) noexcept {
    const std::uint64_t bit_count = bytes_seen_ * 8U;
    block_[used_++] = 0x80U;
    if (used_ > 56U) {
      std::fill(block_.begin() + used_, block_.end(), 0);
      transform();
      used_ = 0;
    }
    std::fill(block_.begin() + used_, block_.begin() + 56U, 0);
    for (std::size_t index = 0; index < 8U; ++index) {
      block_[56U + index] =
          static_cast<std::uint8_t>(bit_count >> ((7U - index) * 8U));
    }
    transform();
    for (std::size_t word = 0; word < state_.size(); ++word) {
      for (std::size_t byte = 0; byte < 4U; ++byte) {
        output[word * 4U + byte] =
            static_cast<std::uint8_t>(state_[word] >> ((3U - byte) * 8U));
      }
    }
  }

 private:
  void transform() noexcept {
    std::array<std::uint32_t, 64> schedule{};
    for (std::size_t index = 0; index < 16U; ++index) {
      const std::size_t base = index * 4U;
      schedule[index] =
          (static_cast<std::uint32_t>(block_[base]) << 24U) |
          (static_cast<std::uint32_t>(block_[base + 1U]) << 16U) |
          (static_cast<std::uint32_t>(block_[base + 2U]) << 8U) |
          static_cast<std::uint32_t>(block_[base + 3U]);
    }
    for (std::size_t index = 16U; index < schedule.size(); ++index) {
      const auto s0 = rotate_right(schedule[index - 15U], 7U) ^
                      rotate_right(schedule[index - 15U], 18U) ^
                      (schedule[index - 15U] >> 3U);
      const auto s1 = rotate_right(schedule[index - 2U], 17U) ^
                      rotate_right(schedule[index - 2U], 19U) ^
                      (schedule[index - 2U] >> 10U);
      schedule[index] = schedule[index - 16U] + s0 +
                        schedule[index - 7U] + s1;
    }

    auto a = state_[0];
    auto b = state_[1];
    auto c = state_[2];
    auto d = state_[3];
    auto e = state_[4];
    auto f = state_[5];
    auto g = state_[6];
    auto h = state_[7];
    for (std::size_t index = 0; index < schedule.size(); ++index) {
      const auto s1 = rotate_right(e, 6U) ^ rotate_right(e, 11U) ^
                      rotate_right(e, 25U);
      const auto choose = (e & f) ^ ((~e) & g);
      const auto first = h + s1 + choose + kRoundConstants[index] +
                         schedule[index];
      const auto s0 = rotate_right(a, 2U) ^ rotate_right(a, 13U) ^
                      rotate_right(a, 22U);
      const auto majority = (a & b) ^ (a & c) ^ (b & c);
      const auto second = s0 + majority;
      h = g;
      g = f;
      f = e;
      e = d + first;
      d = c;
      c = b;
      b = a;
      a = first + second;
    }
    state_[0] += a;
    state_[1] += b;
    state_[2] += c;
    state_[3] += d;
    state_[4] += e;
    state_[5] += f;
    state_[6] += g;
    state_[7] += h;
  }

  std::array<std::uint32_t, 8> state_{
      0x6a09e667U, 0xbb67ae85U, 0x3c6ef372U, 0xa54ff53aU,
      0x510e527fU, 0x9b05688cU, 0x1f83d9abU, 0x5be0cd19U};
  std::array<std::uint8_t, 64> block_{};
  std::size_t used_ = 0;
  std::uint64_t bytes_seen_ = 0;
};

constexpr std::array<std::uint8_t, 4> kMagic{'S', 'Q', 'F', '1'};
constexpr std::uint16_t kMajor = 1;
constexpr std::uint16_t kMinor = 0;
constexpr std::uint32_t kFlags = 0;
constexpr std::uint64_t kPrefixBytes = 24;
constexpr std::uint64_t kDigestBytes = 32;
constexpr std::uint64_t kFixedDescriptorBytes = 8U * 2U + 16U + 8U + 8U;
constexpr char kFrameDomain[] = "symphony.sqfv.batch-frame.v1";
constexpr char kContentDomain[] = "symphony.sqfv.batch-content.v1";

std::array<sqfv_span, 8> fields(const sqfv_descriptor &descriptor) noexcept {
  return {descriptor.binding.metadata_ref,
          descriptor.binding.dataset_revision,
          descriptor.binding.schema_version,
          descriptor.binding.layout_version,
          descriptor.binding.access_scope,
          descriptor.partition,
          descriptor.source_binding,
          descriptor.source_position};
}

bool descriptor_size(const sqfv_descriptor &descriptor,
                     std::uint64_t &size) noexcept {
  size = kFixedDescriptorBytes;
  for (const sqfv_span field : fields(descriptor)) {
    if (field.size > std::numeric_limits<std::uint16_t>::max() ||
        (field.size != 0 && field.data == nullptr)) {
      return false;
    }
    size += field.size;
  }
  return true;
}

template <typename Integer>
void write_be(std::uint8_t *out, Integer value) noexcept {
  for (std::size_t index = 0; index < sizeof(Integer); ++index) {
    out[index] = static_cast<std::uint8_t>(
        value >> ((sizeof(Integer) - 1U - index) * 8U));
  }
}

template <typename Integer>
Integer read_be(const std::uint8_t *input) noexcept {
  Integer value = 0;
  for (std::size_t index = 0; index < sizeof(Integer); ++index) {
    value = static_cast<Integer>((value << 8U) | input[index]);
  }
  return value;
}

bool hash_descriptor(Sha256 &hash,
                     const sqfv_descriptor &descriptor) noexcept {
  std::uint8_t integer[8]{};
  for (const sqfv_span field : fields(descriptor)) {
    if (field.size > std::numeric_limits<std::uint16_t>::max() ||
        (field.size != 0 && field.data == nullptr)) {
      return false;
    }
    write_be<std::uint16_t>(integer,
                            static_cast<std::uint16_t>(field.size));
    if (!hash.update(integer, 2U) ||
        !hash.update(field.data, static_cast<std::size_t>(field.size))) {
      return false;
    }
  }
  if (!hash.update(descriptor.producer_generation, 16U)) return false;
  write_be<std::uint64_t>(integer, descriptor.batch_sequence);
  if (!hash.update(integer, 8U)) return false;
  write_be<std::uint64_t>(integer, descriptor.record_count);
  return hash.update(integer, 8U);
}

void write_descriptor(std::uint8_t *out,
                      const sqfv_descriptor &descriptor) noexcept {
  std::size_t offset = 0;
  for (const sqfv_span field : fields(descriptor)) {
    write_be<std::uint16_t>(out + offset,
                            static_cast<std::uint16_t>(field.size));
    offset += 2U;
    if (field.size != 0) {
      std::memcpy(out + offset, field.data,
                  static_cast<std::size_t>(field.size));
      offset += static_cast<std::size_t>(field.size);
    }
  }
  std::memcpy(out + offset, descriptor.producer_generation, 16U);
  offset += 16U;
  write_be<std::uint64_t>(out + offset, descriptor.batch_sequence);
  offset += 8U;
  write_be<std::uint64_t>(out + offset, descriptor.record_count);
}

class Reader final {
 public:
  Reader(const std::uint8_t *data, std::size_t size) noexcept
      : data_(data), size_(size) {}

  bool span(std::size_t length, sqfv_span &out) noexcept {
    if (length > size_ - offset_) return false;
    out = {data_ + offset_, static_cast<std::uint64_t>(length)};
    offset_ += length;
    return true;
  }

  template <typename Integer>
  bool integer(Integer &out) noexcept {
    if (sizeof(Integer) > size_ - offset_) return false;
    out = read_be<Integer>(data_ + offset_);
    offset_ += sizeof(Integer);
    return true;
  }

  bool raw(std::uint8_t *out, std::size_t length) noexcept {
    if (length > size_ - offset_) return false;
    std::memcpy(out, data_ + offset_, length);
    offset_ += length;
    return true;
  }

  bool finished() const noexcept { return offset_ == size_; }

 private:
  const std::uint8_t *data_;
  std::size_t size_;
  std::size_t offset_ = 0;
};

bool read_descriptor(const std::uint8_t *data, std::size_t size,
                     sqfv_descriptor &descriptor) noexcept {
  Reader reader(data, size);
  sqfv_span *const output[] = {
      &descriptor.binding.metadata_ref,
      &descriptor.binding.dataset_revision,
      &descriptor.binding.schema_version,
      &descriptor.binding.layout_version,
      &descriptor.binding.access_scope,
      &descriptor.partition,
      &descriptor.source_binding,
      &descriptor.source_position};
  for (sqfv_span *field : output) {
    std::uint16_t length = 0;
    if (!reader.integer(length) || !reader.span(length, *field)) return false;
  }
  if (!reader.raw(descriptor.producer_generation, 16U) ||
      !reader.integer(descriptor.batch_sequence) ||
      !reader.integer(descriptor.record_count)) {
    return false;
  }
  return reader.finished();
}

bool measured_size(const sqfv_limits &limits,
                   const sqfv_descriptor &descriptor,
                   sqfv_span payload,
                   std::uint64_t &descriptor_bytes,
                   std::uint64_t &header_bytes,
                   std::uint64_t &frame_bytes) noexcept {
  if (payload.size == 0 || payload.data == nullptr ||
      payload.size > limits.max_payload_bytes ||
      payload.size > std::numeric_limits<std::uint32_t>::max() ||
      !descriptor_size(descriptor, descriptor_bytes) ||
      descriptor_bytes > limits.max_descriptor_bytes ||
      descriptor_bytes > std::numeric_limits<std::uint32_t>::max()) {
    return false;
  }
  header_bytes = kPrefixBytes + descriptor_bytes + kDigestBytes;
  if (header_bytes > std::numeric_limits<std::uint32_t>::max()) return false;
  frame_bytes = header_bytes + payload.size;
  return frame_bytes <= limits.max_frame_bytes;
}

bool frame_digest(const std::uint8_t *prefix_and_descriptor,
                  std::size_t prefix_and_descriptor_size,
                  sqfv_span payload, std::uint8_t out[32]) noexcept {
  Sha256 hash;
  if (!hash.update(reinterpret_cast<const std::uint8_t *>(kFrameDomain),
                   sizeof(kFrameDomain)) ||
      !hash.update(prefix_and_descriptor, prefix_and_descriptor_size) ||
      !hash.update(payload.data, static_cast<std::size_t>(payload.size))) {
    return false;
  }
  hash.finish(out);
  return true;
}

bool digest_equal(const std::uint8_t *left,
                  const std::uint8_t *right) noexcept {
  std::uint8_t difference = 0;
  for (std::size_t index = 0; index < kDigestBytes; ++index) {
    difference |= static_cast<std::uint8_t>(left[index] ^ right[index]);
  }
  return difference == 0;
}

}  // namespace

bool compute_content_id(const sqfv_descriptor &descriptor, sqfv_span payload,
                        std::uint8_t out[32]) noexcept {
  if (out == nullptr || (payload.size != 0 && payload.data == nullptr) ||
      payload.size > std::numeric_limits<std::size_t>::max()) {
    return false;
  }
  std::uint8_t payload_digest[32]{};
  Sha256 payload_hash;
  if (!payload_hash.update(payload.data,
                           static_cast<std::size_t>(payload.size))) {
    return false;
  }
  payload_hash.finish(payload_digest);

  Sha256 content_hash;
  if (!content_hash.update(
          reinterpret_cast<const std::uint8_t *>(kContentDomain),
          sizeof(kContentDomain)) ||
      !hash_descriptor(content_hash, descriptor) ||
      !content_hash.update(payload_digest, sizeof(payload_digest))) {
    return false;
  }
  content_hash.finish(out);
  return true;
}

}  // namespace sqfv_internal

extern "C" sqfv_status sqfv_frame_measure(const sqfv_context *context,
                                            const sqfv_batch *batch,
                                            std::uint64_t *out_size) {
  if (context == nullptr || batch == nullptr || out_size == nullptr ||
      !sqfv_internal::batch_belongs_to_context(batch, context)) {
    return SQFV_INVALID_ARGUMENT;
  }
  *out_size = 0;
  sqfv_descriptor descriptor{};
  descriptor.struct_size = sizeof(descriptor);
  descriptor.abi_version = SQFV_BATCH_ABI_VERSION;
  descriptor.binding.struct_size = sizeof(descriptor.binding);
  descriptor.binding.abi_version = SQFV_BATCH_ABI_VERSION;
  const sqfv_status status =
      sqfv_batch_descriptor_view(batch, &descriptor);
  if (status != SQFV_OK) return status;
  const sqfv_span payload = sqfv_internal::batch_payload_view(batch);
  std::uint64_t descriptor_bytes = 0;
  std::uint64_t header_bytes = 0;
  std::uint64_t frame_bytes = 0;
  if (!sqfv_internal::measured_size(sqfv_internal::context_limits(context),
                                    descriptor, payload, descriptor_bytes,
                                    header_bytes, frame_bytes)) {
    return SQFV_LIMIT;
  }
  *out_size = frame_bytes;
  return SQFV_OK;
}

extern "C" sqfv_status sqfv_frame_encode(const sqfv_context *context,
                                           const sqfv_batch *batch,
                                           std::uint8_t *out,
                                           std::uint64_t capacity,
                                           std::uint64_t *out_size) {
  if (context == nullptr || batch == nullptr || out_size == nullptr ||
      !sqfv_internal::batch_belongs_to_context(batch, context)) {
    return SQFV_INVALID_ARGUMENT;
  }
  *out_size = 0;
  sqfv_descriptor descriptor{};
  descriptor.struct_size = sizeof(descriptor);
  descriptor.abi_version = SQFV_BATCH_ABI_VERSION;
  descriptor.binding.struct_size = sizeof(descriptor.binding);
  descriptor.binding.abi_version = SQFV_BATCH_ABI_VERSION;
  const sqfv_status view_status =
      sqfv_batch_descriptor_view(batch, &descriptor);
  if (view_status != SQFV_OK) return view_status;
  const sqfv_span payload = sqfv_internal::batch_payload_view(batch);
  std::uint64_t descriptor_bytes = 0;
  std::uint64_t header_bytes = 0;
  std::uint64_t frame_bytes = 0;
  if (!sqfv_internal::measured_size(sqfv_internal::context_limits(context),
                                    descriptor, payload, descriptor_bytes,
                                    header_bytes, frame_bytes)) {
    return SQFV_LIMIT;
  }
  *out_size = frame_bytes;
  if (capacity < frame_bytes) return SQFV_LIMIT;
  if (out == nullptr) return SQFV_INVALID_ARGUMENT;

  // The 24-byte prefix and canonical descriptor precede the stored digest.
  // The digest covers the prefix and descriptor, then the raw payload.
  std::memcpy(out, sqfv_internal::kMagic.data(), sqfv_internal::kMagic.size());
  sqfv_internal::write_be<std::uint16_t>(out + 4U, sqfv_internal::kMajor);
  sqfv_internal::write_be<std::uint16_t>(out + 6U, sqfv_internal::kMinor);
  sqfv_internal::write_be<std::uint32_t>(out + 8U, sqfv_internal::kFlags);
  sqfv_internal::write_be<std::uint32_t>(
      out + 12U, static_cast<std::uint32_t>(header_bytes));
  sqfv_internal::write_be<std::uint32_t>(
      out + 16U, static_cast<std::uint32_t>(payload.size));
  sqfv_internal::write_be<std::uint32_t>(
      out + 20U, static_cast<std::uint32_t>(descriptor_bytes));
  sqfv_internal::write_descriptor(out + sqfv_internal::kPrefixBytes,
                                   descriptor);
  const std::size_t digest_offset = static_cast<std::size_t>(
      sqfv_internal::kPrefixBytes + descriptor_bytes);
  std::uint8_t digest[32]{};
  if (!sqfv_internal::frame_digest(out, digest_offset, payload, digest)) {
    return SQFV_INTERNAL_ERROR;
  }
  std::memcpy(out + digest_offset, digest, sizeof(digest));
  std::memcpy(out + static_cast<std::size_t>(header_bytes), payload.data,
              static_cast<std::size_t>(payload.size));
  return SQFV_OK;
}

extern "C" sqfv_status sqfv_frame_decode(sqfv_context *context,
                                           sqfv_span frame,
                                           sqfv_batch **out_batch) {
  if (context == nullptr || out_batch == nullptr ||
      (frame.size != 0 && frame.data == nullptr)) {
    return SQFV_INVALID_ARGUMENT;
  }
  *out_batch = nullptr;
  const sqfv_limits limits = sqfv_internal::context_limits(context);
  if (frame.size > limits.max_frame_bytes ||
      frame.size > std::numeric_limits<std::size_t>::max()) {
    return SQFV_LIMIT;
  }
  if (frame.size < sqfv_internal::kPrefixBytes +
                       sqfv_internal::kFixedDescriptorBytes +
                       sqfv_internal::kDigestBytes) {
    return SQFV_CORRUPT_FRAME;
  }
  const std::uint8_t *const bytes = frame.data;
  if (std::memcmp(bytes, sqfv_internal::kMagic.data(),
                  sqfv_internal::kMagic.size()) != 0) {
    return SQFV_CORRUPT_FRAME;
  }
  const auto major = sqfv_internal::read_be<std::uint16_t>(bytes + 4U);
  const auto minor = sqfv_internal::read_be<std::uint16_t>(bytes + 6U);
  const auto flags = sqfv_internal::read_be<std::uint32_t>(bytes + 8U);
  if (major != sqfv_internal::kMajor || minor != sqfv_internal::kMinor ||
      flags != sqfv_internal::kFlags) {
    return SQFV_UNSUPPORTED_FRAME;
  }
  const auto header_bytes = sqfv_internal::read_be<std::uint32_t>(bytes + 12U);
  const auto payload_bytes = sqfv_internal::read_be<std::uint32_t>(bytes + 16U);
  const auto descriptor_bytes =
      sqfv_internal::read_be<std::uint32_t>(bytes + 20U);
  if (descriptor_bytes < sqfv_internal::kFixedDescriptorBytes ||
      payload_bytes == 0) {
    return SQFV_CORRUPT_FRAME;
  }
  if (descriptor_bytes > limits.max_descriptor_bytes ||
      payload_bytes > limits.max_payload_bytes) {
    return SQFV_LIMIT;
  }
  const std::uint64_t required_header =
      sqfv_internal::kPrefixBytes + descriptor_bytes +
      sqfv_internal::kDigestBytes;
  if (header_bytes != required_header ||
      static_cast<std::uint64_t>(header_bytes) + payload_bytes != frame.size) {
    return SQFV_CORRUPT_FRAME;
  }
  sqfv_descriptor descriptor{};
  descriptor.struct_size = sizeof(descriptor);
  descriptor.abi_version = SQFV_BATCH_ABI_VERSION;
  descriptor.binding.struct_size = sizeof(descriptor.binding);
  descriptor.binding.abi_version = SQFV_BATCH_ABI_VERSION;
  if (!sqfv_internal::read_descriptor(
          bytes + sqfv_internal::kPrefixBytes,
          static_cast<std::size_t>(descriptor_bytes), descriptor)) {
    return SQFV_CORRUPT_FRAME;
  }

  const std::size_t digest_offset = static_cast<std::size_t>(
      sqfv_internal::kPrefixBytes + descriptor_bytes);
  const sqfv_span payload{
      bytes + header_bytes, static_cast<std::uint64_t>(payload_bytes)};
  std::uint8_t actual_digest[32]{};
  if (!sqfv_internal::frame_digest(bytes, digest_offset, payload,
                                   actual_digest)) {
    return SQFV_CORRUPT_FRAME;
  }
  if (!sqfv_internal::digest_equal(bytes + digest_offset, actual_digest)) {
    return SQFV_CORRUPT_FRAME;
  }
  const sqfv_status prepare_status =
      sqfv_batch_prepare_copy(context, &descriptor, payload, out_batch);
  if (prepare_status == SQFV_INVALID_ARGUMENT ||
      prepare_status == SQFV_UNSUPPORTED_ABI) {
    return SQFV_CORRUPT_FRAME;
  }
  return prepare_status;
}
