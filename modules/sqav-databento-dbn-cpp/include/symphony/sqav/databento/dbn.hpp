#ifndef SYMPHONY_SQAV_DATABENTO_DBN_HPP
#define SYMPHONY_SQAV_DATABENTO_DBN_HPP
#include <cstdint>
#include <optional>
#include <string_view>
#include <symphony/sqav/capture.hpp>
namespace symphony::sqav::databento {
inline constexpr std::string_view adapter_id = "sqav-databento-dbn-cpp";
inline constexpr std::string_view adapter_version = "0.2.0-dev";
inline constexpr std::string_view native_schema = "databento:mbo";
inline constexpr std::string_view native_encoding =
    "databento:dbn-v3-uncompressed";
inline constexpr std::string_view native_encoding_v1 =
    "databento:dbn-v1-uncompressed";
[[nodiscard]] constexpr std::string_view encoding_for_version(std::uint8_t version) noexcept {
  return version == 1 ? native_encoding_v1 : version == 3 ? native_encoding : std::string_view{};
}
enum class Status : std::uint8_t {
  ok,
  invalid_argument,
  unsupported,
  malformed,
  limit,
  binding_mismatch,
  no_memory,
  internal_error
};
struct Limits {
  std::uint64_t max_file_bytes = 0; // Positive; ceiling 64 MiB.
  std::uint32_t max_metadata_bytes =
      0;                         // Includes 8-byte prefix; ceiling 1 MiB.
  std::uint64_t max_records = 0; // Positive; ceiling 1,048,576.
};
struct Mbo {
  std::uint16_t publisher_id = 0;
  std::uint32_t instrument_id = 0;
  std::uint64_t ts_event = 0, order_id = 0;
  std::int64_t price =
      0; // Provider fixed-point units of 1e-9; sentinel retained.
  std::uint32_t size = 0;
  std::uint8_t flags = 0, channel_id = 0, action = 0, side = 0;
  std::uint64_t ts_recv = 0;
  std::int32_t ts_in_delta = 0;
  std::uint32_t sequence = 0;
  std::optional<std::uint64_t> ts_out;
  bool operator==(const Mbo &) const = default;
};
struct Metadata {
  std::string_view dataset;
  std::uint64_t start = 0, end = 0, limit = 0, record_count = 0;
  std::uint8_t stype_in = 0, stype_out = 0;
  bool ts_out = false;
  std::uint16_t symbol_cstr_len = 0;
  std::uint32_t symbols = 0, partial = 0, not_found = 0, mappings = 0;
  std::uint8_t version = 0; // Original DBN wire version; never upgraded.
};
// Borrowed immutable bytes: caller keeps storage alive and unchanged throughout
// use. Parsing and record access allocate nothing; failure preserves out.
class FileView final {
public:
  [[nodiscard]] static Status inspect(ByteView, const Limits &,
                                      FileView &out) noexcept;
  [[nodiscard]] explicit operator bool() const noexcept {
    return !original_.empty();
  }
  [[nodiscard]] const Metadata &metadata() const noexcept { return metadata_; }
  [[nodiscard]] ByteView original() const noexcept { return original_; }
  [[nodiscard]] ByteView encoded_metadata() const noexcept {
    return original_.first(records_offset_);
  }
  [[nodiscard]] Status record(std::uint64_t index, Mbo &out) const noexcept;

private:
  ByteView original_{};
  Metadata metadata_{};
  std::size_t records_offset_ = 0;
};
// Validates one exact DBNv1/v3 MBO record (56 bytes, or 64 with ts_out).
[[nodiscard]] Status decode_mbo(ByteView, bool ts_out, Mbo &out) noexcept;
// Verifies provider/dataset/schema/encoding against the inspected file, derives
// its actual count and fixes adapter identity. Original file bytes stay intact.
[[nodiscard]] Status capture_file(const Description &, ByteView, const Limits &,
                                  const sqav::Limits &, Capture &out) noexcept;
[[nodiscard]] Status inspect_capture(const Capture &, const Limits &,
                                     FileView &out) noexcept;
} // namespace symphony::sqav::databento
#endif
