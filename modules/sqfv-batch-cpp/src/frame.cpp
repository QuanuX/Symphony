#include "batch_internal.hpp"

#include <algorithm>
#include <array>
#include <cstddef>
#include <cstdint>
#include <cstring>
#include <limits>
#include <string_view>
#include <utility>

namespace symphony::sqfv {
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
constexpr std::size_t kPrefixBytes = 24;
constexpr std::size_t kDigestBytes = 32;
constexpr std::size_t kFixedDescriptorBytes = 8U * 2U + 16U + 8U + 8U;
constexpr char kFrameDomain[] = "symphony.sqfv.batch-frame.v1";
constexpr char kContentDomain[] = "symphony.sqfv.batch-content.v1";

std::array<std::string_view, 8> fields(
    const detail::DescriptorView &descriptor) noexcept {
  return {descriptor.binding.metadata_ref,
          descriptor.binding.dataset_revision,
          descriptor.binding.schema_version,
          descriptor.binding.layout_version,
          descriptor.binding.access_scope,
          descriptor.partition,
          descriptor.source_binding,
          descriptor.source_position};
}

bool descriptor_size(const detail::DescriptorView &descriptor,
                     std::size_t &size) noexcept {
  size = kFixedDescriptorBytes;
  for (const std::string_view field : fields(descriptor)) {
    if (field.size() > std::numeric_limits<std::uint16_t>::max()) return false;
    size += field.size();
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
                     const detail::DescriptorView &descriptor) noexcept {
  std::uint8_t integer[8]{};
  for (const std::string_view field : fields(descriptor)) {
    if (field.size() > std::numeric_limits<std::uint16_t>::max()) return false;
    write_be<std::uint16_t>(integer,
                            static_cast<std::uint16_t>(field.size()));
    const auto *bytes =
        reinterpret_cast<const std::uint8_t *>(field.data());
    if (!hash.update(integer, 2U) || !hash.update(bytes, field.size())) {
      return false;
    }
  }
  if (!hash.update(descriptor.producer_generation.data(), 16U)) return false;
  write_be<std::uint64_t>(integer, descriptor.batch_sequence);
  if (!hash.update(integer, 8U)) return false;
  write_be<std::uint64_t>(integer, descriptor.record_count);
  return hash.update(integer, 8U);
}

void write_descriptor(std::uint8_t *out,
                      const detail::DescriptorView &descriptor) noexcept {
  std::size_t offset = 0;
  for (const std::string_view field : fields(descriptor)) {
    write_be<std::uint16_t>(out + offset,
                            static_cast<std::uint16_t>(field.size()));
    offset += 2U;
    if (!field.empty()) {
      std::memcpy(out + offset, field.data(), field.size());
      offset += field.size();
    }
  }
  std::memcpy(out + offset, descriptor.producer_generation.data(), 16U);
  offset += 16U;
  write_be<std::uint64_t>(out + offset, descriptor.batch_sequence);
  offset += 8U;
  write_be<std::uint64_t>(out + offset, descriptor.record_count);
}

class Reader final {
 public:
  Reader(const std::uint8_t *data, std::size_t size) noexcept
      : data_(data), size_(size) {}

  bool string(std::size_t length, std::string_view &out) noexcept {
    if (length > size_ - offset_) return false;
    out = {reinterpret_cast<const char *>(data_ + offset_), length};
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
                     detail::DescriptorView &descriptor) noexcept {
  Reader reader(data, size);
  std::string_view *const output[] = {
      &descriptor.binding.metadata_ref,
      &descriptor.binding.dataset_revision,
      &descriptor.binding.schema_version,
      &descriptor.binding.layout_version,
      &descriptor.binding.access_scope,
      &descriptor.partition,
      &descriptor.source_binding,
      &descriptor.source_position};
  for (std::string_view *field : output) {
    std::uint16_t length = 0;
    if (!reader.integer(length) || !reader.string(length, *field)) return false;
  }
  if (!reader.raw(descriptor.producer_generation.data(), 16U) ||
      !reader.integer(descriptor.batch_sequence) ||
      !reader.integer(descriptor.record_count)) {
    return false;
  }
  return reader.finished();
}

bool measured_size(const Limits &limits,
                   const detail::DescriptorView &descriptor,
                   ByteView payload,
                   std::size_t &descriptor_bytes,
                   std::size_t &header_bytes,
                   std::size_t &frame_bytes) noexcept {
  if (payload.empty() || payload.size() > limits.max_payload_bytes ||
      payload.size() > std::numeric_limits<std::uint32_t>::max() ||
      !descriptor_size(descriptor, descriptor_bytes) ||
      descriptor_bytes > limits.max_descriptor_bytes ||
      descriptor_bytes > std::numeric_limits<std::uint32_t>::max()) {
    return false;
  }
  header_bytes = kPrefixBytes + descriptor_bytes + kDigestBytes;
  if (header_bytes > std::numeric_limits<std::uint32_t>::max()) return false;
  frame_bytes = header_bytes + payload.size();
  return frame_bytes <= limits.max_frame_bytes;
}

std::array<std::uint8_t, kPrefixBytes> prefix_bytes(
    std::size_t descriptor_bytes, std::size_t header_bytes,
    std::size_t payload_bytes) noexcept {
  std::array<std::uint8_t, kPrefixBytes> prefix{};
  std::memcpy(prefix.data(), kMagic.data(), kMagic.size());
  write_be<std::uint16_t>(prefix.data() + 4U, kMajor);
  write_be<std::uint16_t>(prefix.data() + 6U, kMinor);
  write_be<std::uint32_t>(prefix.data() + 8U, kFlags);
  write_be<std::uint32_t>(prefix.data() + 12U,
                          static_cast<std::uint32_t>(header_bytes));
  write_be<std::uint32_t>(prefix.data() + 16U,
                          static_cast<std::uint32_t>(payload_bytes));
  write_be<std::uint32_t>(prefix.data() + 20U,
                          static_cast<std::uint32_t>(descriptor_bytes));
  return prefix;
}

bool frame_digest(ByteView prefix_and_descriptor, ByteView payload,
                  ContentId &out) noexcept {
  Sha256 hash;
  if (!hash.update(reinterpret_cast<const std::uint8_t *>(kFrameDomain),
                   sizeof(kFrameDomain)) ||
      !hash.update(prefix_and_descriptor.data(),
                   prefix_and_descriptor.size()) ||
      !hash.update(payload.data(), payload.size())) {
    return false;
  }
  hash.finish(out.data());
  return true;
}

bool frame_digest(const std::array<std::uint8_t, kPrefixBytes> &prefix,
                  const detail::DescriptorView &descriptor,
                  ByteView payload, ContentId &out) noexcept {
  Sha256 hash;
  if (!hash.update(reinterpret_cast<const std::uint8_t *>(kFrameDomain),
                   sizeof(kFrameDomain)) ||
      !hash.update(prefix.data(), prefix.size()) ||
      !hash_descriptor(hash, descriptor) ||
      !hash.update(payload.data(), payload.size())) {
    return false;
  }
  hash.finish(out.data());
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

namespace detail {

bool compute_content_id(const DescriptorView &descriptor, ByteView payload,
                        ContentId &out) noexcept {
  ContentId payload_digest{};
  Sha256 payload_hash;
  if (!payload_hash.update(payload.data(), payload.size())) return false;
  payload_hash.finish(payload_digest.data());

  Sha256 content_hash;
  if (!content_hash.update(
          reinterpret_cast<const std::uint8_t *>(kContentDomain),
          sizeof(kContentDomain)) ||
      !hash_descriptor(content_hash, descriptor) ||
      !content_hash.update(payload_digest.data(), payload_digest.size())) {
    return false;
  }
  content_hash.finish(out.data());
  return true;
}

}  // namespace detail

Status frame_measure(const Context &context, const Batch &batch,
                     std::size_t &out_size) noexcept {
  if (!context || !batch || !detail::FrameAccess::same_context(context, batch))
    return Status::invalid_argument;
  const Limits *limits = detail::FrameAccess::limits(context);
  if (limits == nullptr) return Status::invalid_argument;
  const auto descriptor = detail::view_of(batch.descriptor());
  const ByteView payload = detail::FrameAccess::payload(batch);
  std::size_t descriptor_bytes = 0;
  std::size_t header_bytes = 0;
  std::size_t frame_bytes = 0;
  if (!measured_size(*limits, descriptor, payload, descriptor_bytes,
                     header_bytes, frame_bytes)) {
    return Status::limit;
  }
  out_size = frame_bytes;
  return Status::ok;
}

Status frame_encode(const Context &context, const Batch &batch,
                    MutableBytes out, std::size_t &out_size) noexcept {
  if (!context || !batch || !detail::FrameAccess::same_context(context, batch))
    return Status::invalid_argument;
  const Limits *limits = detail::FrameAccess::limits(context);
  if (limits == nullptr) return Status::invalid_argument;
  const auto descriptor = detail::view_of(batch.descriptor());
  const ByteView payload = detail::FrameAccess::payload(batch);
  std::size_t descriptor_bytes = 0;
  std::size_t header_bytes = 0;
  std::size_t frame_bytes = 0;
  if (!measured_size(*limits, descriptor, payload, descriptor_bytes,
                     header_bytes, frame_bytes)) {
    return Status::limit;
  }
  if (out.size() < frame_bytes) return Status::limit;
  const auto prefix = prefix_bytes(descriptor_bytes, header_bytes, payload.size());
  ContentId digest{};
  if (!frame_digest(prefix, descriptor, payload, digest)) {
    return Status::internal_error;
  }
  std::memcpy(out.data(), prefix.data(), prefix.size());
  write_descriptor(out.data() + kPrefixBytes, descriptor);
  const std::size_t digest_offset = kPrefixBytes + descriptor_bytes;
  std::memcpy(out.data() + digest_offset, digest.data(), digest.size());
  std::memcpy(out.data() + header_bytes, payload.data(), payload.size());
  out_size = frame_bytes;
  return Status::ok;
}

Status frame_decode(Context &context, ByteView frame, Batch &out) noexcept {
  if (!context) return Status::invalid_argument;
  const Limits *limits = detail::FrameAccess::limits(context);
  if (limits == nullptr) return Status::invalid_argument;
  if (frame.size() > limits->max_frame_bytes) return Status::limit;
  if (frame.size() < kPrefixBytes + kFixedDescriptorBytes + kDigestBytes)
    return Status::corrupt_frame;
  const std::uint8_t *bytes = frame.data();
  if (std::memcmp(bytes, kMagic.data(), kMagic.size()) != 0)
    return Status::corrupt_frame;
  const auto major = read_be<std::uint16_t>(bytes + 4U);
  const auto minor = read_be<std::uint16_t>(bytes + 6U);
  const auto flags = read_be<std::uint32_t>(bytes + 8U);
  if (major != kMajor || minor != kMinor || flags != kFlags)
    return Status::unsupported_frame;
  const auto header_bytes = read_be<std::uint32_t>(bytes + 12U);
  const auto payload_bytes = read_be<std::uint32_t>(bytes + 16U);
  const auto descriptor_bytes = read_be<std::uint32_t>(bytes + 20U);
  if (descriptor_bytes < kFixedDescriptorBytes || payload_bytes == 0)
    return Status::corrupt_frame;
  if (descriptor_bytes > limits->max_descriptor_bytes ||
      payload_bytes > limits->max_payload_bytes)
    return Status::limit;
  const std::uint64_t required_header =
      kPrefixBytes + std::uint64_t(descriptor_bytes) + kDigestBytes;
  if (header_bytes != required_header ||
      std::uint64_t(header_bytes) + payload_bytes != frame.size())
    return Status::corrupt_frame;

  detail::DescriptorView descriptor{};
  if (!read_descriptor(bytes + kPrefixBytes, descriptor_bytes, descriptor))
    return Status::corrupt_frame;
  const std::size_t digest_offset = kPrefixBytes + descriptor_bytes;
  const ByteView payload(bytes + header_bytes, payload_bytes);
  ContentId actual_digest{};
  if (!frame_digest(ByteView(bytes, digest_offset), payload, actual_digest) ||
      !digest_equal(bytes + digest_offset, actual_digest.data()))
    return Status::corrupt_frame;

  Batch decoded;
  const Status status =
      detail::FrameAccess::prepare_copy_views(context, descriptor, payload, decoded);
  if (status == Status::invalid_argument) return Status::corrupt_frame;
  if (status != Status::ok) return status;
  out = std::move(decoded);
  return Status::ok;
}

}  // namespace symphony::sqfv
