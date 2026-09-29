#include "symphony/sqv/prepared_batch.hpp"

#include <algorithm>
#include <array>
#include <limits>
#include <stdexcept>
#include <type_traits>
#include <utility>

namespace symphony::sqv::prototype {
namespace {

constexpr std::array<std::uint8_t, 4> magic{'S', 'Q', 'B', '1'};
constexpr std::size_t fixed_header_bytes = 48;
constexpr std::size_t field_limit = 128;

void require(bool condition, const char* message) {
  if (!condition) throw std::invalid_argument(message);
}

void validate_field(const std::string& value, bool required) {
  require(value.size() <= field_limit, "metadata field too long");
  require(!required || !value.empty(), "required metadata missing");
  require(std::all_of(value.begin(), value.end(), [](unsigned char c) {
    return c >= 0x21 && c <= 0x7e;
  }), "metadata must be printable ASCII without whitespace");
}

void validate(const Descriptor& d, std::size_t payload_size) {
  validate_field(d.dataset_revision, true);
  validate_field(d.schema_name, true);
  validate_field(d.schema_version, true);
  validate_field(d.representation, true);
  validate_field(d.access_scope, true);
  validate_field(d.provenance, true);
  validate_field(d.source_position, false);
  require(d.generation != 0, "generation must be nonzero");
  require(d.first <= d.last, "cursor range is reversed");
  require(d.record_count != 0, "record count must be nonzero");
  require(d.last - d.first == static_cast<std::uint64_t>(d.record_count) - 1,
          "cursor range and record count disagree");
  require(payload_size != 0 && payload_size <= max_payload_bytes,
          "payload is empty or exceeds bound");
}

template <typename Integer>
void put(std::vector<std::uint8_t>& out, Integer value) {
  static_assert(std::is_unsigned_v<Integer>);
  for (std::size_t i = sizeof(Integer); i > 0; --i)
    out.push_back(static_cast<std::uint8_t>(value >> ((i - 1) * 8)));
}

class Reader {
 public:
  explicit Reader(std::span<const std::uint8_t> bytes) : bytes_(bytes) {}

  template <typename Integer>
  Integer get() {
    static_assert(std::is_unsigned_v<Integer>);
    require(remaining() >= sizeof(Integer), "truncated frame");
    Integer value = 0;
    for (std::size_t i = 0; i < sizeof(Integer); ++i)
      value = static_cast<Integer>((value << 8) | bytes_[offset_++]);
    return value;
  }

  std::string get_string() {
    const auto size = get<std::uint16_t>();
    require(size <= field_limit, "metadata field too long");
    require(remaining() >= size, "truncated metadata");
    std::string result(reinterpret_cast<const char*>(bytes_.data() + offset_), size);
    offset_ += size;
    return result;
  }

  std::size_t offset() const noexcept { return offset_; }
  std::size_t remaining() const noexcept { return bytes_.size() - offset_; }

 private:
  std::span<const std::uint8_t> bytes_;
  std::size_t offset_ = 0;
};

void put_string(std::vector<std::uint8_t>& out, const std::string& value) {
  put<std::uint16_t>(out, static_cast<std::uint16_t>(value.size()));
  out.insert(out.end(), value.begin(), value.end());
}

}  // namespace

PreparedBatch::PreparedBatch(
    Descriptor descriptor,
    std::shared_ptr<const std::vector<std::uint8_t>> payload)
    : descriptor_(std::move(descriptor)), payload_(std::move(payload)) {}

PreparedBatch PreparedBatch::prepare(Descriptor descriptor,
                                     std::span<const std::uint8_t> payload) {
  validate(descriptor, payload.size());
  auto frozen = std::make_shared<const std::vector<std::uint8_t>>(
      payload.begin(), payload.end());
  return PreparedBatch(std::move(descriptor), std::move(frozen));
}

ReadLease PreparedBatch::acquire(const std::string& requested_scope) const {
  require(payload_ != nullptr, "batch is empty");
  require(requested_scope == descriptor_.access_scope, "scope mismatch");
  return ReadLease(payload_);
}

std::vector<std::uint8_t> encode(const PreparedBatch& batch) {
  const auto lease = batch.acquire(batch.descriptor().access_scope);
  const auto payload = lease.bytes();
  validate(batch.descriptor(), payload.size());
  const auto& d = batch.descriptor();
  const std::size_t header_size = fixed_header_bytes + 14 +
      d.dataset_revision.size() + d.schema_name.size() +
      d.schema_version.size() + d.representation.size() +
      d.access_scope.size() + d.provenance.size() + d.source_position.size();
  require(header_size <= max_header_bytes, "header exceeds bound");
  std::vector<std::uint8_t> out;
  out.reserve(header_size + payload.size());
  out.insert(out.end(), magic.begin(), magic.end());
  put<std::uint16_t>(out, 1);  // major
  put<std::uint16_t>(out, 0);  // minor
  put<std::uint32_t>(out, static_cast<std::uint32_t>(header_size));
  put<std::uint32_t>(out, static_cast<std::uint32_t>(payload.size()));
  put<std::uint64_t>(out, d.generation);
  put<std::uint64_t>(out, d.first);
  put<std::uint64_t>(out, d.last);
  put<std::uint32_t>(out, d.partition);
  put<std::uint32_t>(out, d.record_count);
  put_string(out, d.dataset_revision);
  put_string(out, d.schema_name);
  put_string(out, d.schema_version);
  put_string(out, d.representation);
  put_string(out, d.access_scope);
  put_string(out, d.provenance);
  put_string(out, d.source_position);
  require(out.size() == header_size, "internal header size mismatch");
  out.insert(out.end(), payload.begin(), payload.end());
  return out;
}

PreparedBatch decode(std::span<const std::uint8_t> frame) {
  require(frame.size() >= fixed_header_bytes &&
              frame.size() <= max_header_bytes + max_payload_bytes,
          "frame size outside bounds");
  Reader reader(frame);
  for (auto byte : magic) require(reader.get<std::uint8_t>() == byte, "invalid magic");
  require(reader.get<std::uint16_t>() == 1, "unsupported major version");
  require(reader.get<std::uint16_t>() == 0, "unsupported minor version");
  const auto header_size = reader.get<std::uint32_t>();
  const auto payload_size = reader.get<std::uint32_t>();
  require(header_size >= fixed_header_bytes + 14 && header_size <= max_header_bytes,
          "invalid header length");
  require(payload_size != 0 && payload_size <= max_payload_bytes,
          "invalid payload length");
  require(frame.size() == static_cast<std::size_t>(header_size) + payload_size,
          "frame length mismatch");
  Descriptor d;
  d.generation = reader.get<std::uint64_t>();
  d.first = reader.get<std::uint64_t>();
  d.last = reader.get<std::uint64_t>();
  d.partition = reader.get<std::uint32_t>();
  d.record_count = reader.get<std::uint32_t>();
  d.dataset_revision = reader.get_string();
  d.schema_name = reader.get_string();
  d.schema_version = reader.get_string();
  d.representation = reader.get_string();
  d.access_scope = reader.get_string();
  d.provenance = reader.get_string();
  d.source_position = reader.get_string();
  require(reader.offset() == header_size, "header contains unknown or missing fields");
  validate(d, payload_size);
  return PreparedBatch::prepare(std::move(d), frame.subspan(header_size, payload_size));
}

}  // namespace symphony::sqv::prototype
