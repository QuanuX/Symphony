#ifndef SYMPHONY_SQAV_CAPTURE_HPP
#define SYMPHONY_SQAV_CAPTURE_HPP

#include <cstdint>
#include <memory>
#include <optional>
#include <string>
#include <string_view>
#include <symphony/sqmv/metadata.hpp>
#include <vector>

namespace symphony::sqav {
inline constexpr std::string_view capture_schema = "sqav-capture-v1";
inline constexpr std::string_view capture_layout = "sqav-capture-v1";
using ByteView = sqfv::ByteView;
enum class Status : std::uint8_t {
  ok,
  invalid_argument,
  limit,
  no_memory,
  corrupt_capture,
  unsupported_capture,
  reference_mismatch,
  binding_mismatch,
  stale,
  closed,
  internal_error
};
struct Limits {
  std::uint64_t max_capture_bytes = 0; // Entire encoded object; ceiling 64 MiB.
  std::uint32_t max_metadata_bytes = 0; // Ceiling 65,536.
  std::uint16_t max_field_bytes = 0;    // Ceiling 4,096; all limits mandatory.
};
// Exact caller assertions, not discovered provider capability or authority.
struct Source {
  std::string provider_ref;
  std::string interface_ref;
  std::string interface_version;
  std::string operation;
  std::string adapter_ref;
  std::string adapter_version;
  std::string dataset_id;
  std::string dataset_revision;
  std::string selection_ref;
  std::string native_schema_ref;
  std::string native_encoding_ref;
  std::string access_scope;
};
enum class TimeRole : std::uint8_t {
  acquisition = 1,
  event = 2,
  publication = 3,
  revision = 4,
  receipt = 5
};
struct TimeEvidence {
  TimeRole role = TimeRole::acquisition;
  std::string value;
  std::string format_ref;
  std::string clock_ref;
  std::string precision_ref;
  std::string evidence_ref;
};
enum class Coverage : std::uint8_t {
  unknown = 0,
  complete = 1,
  partial = 2,
  gap = 3
};
struct Description {
  Source source;
  std::string attempt_id;
  std::string attribution_ref;
  std::string source_position; // Empty means absent; never the transfer cursor.
  Coverage coverage = Coverage::unknown;
  std::string coverage_scope;
  std::string coverage_evidence_ref; // Required unless coverage is unknown.
  std::optional<std::uint64_t> source_record_count;
  std::vector<TimeEvidence>
      times; // Exactly one acquisition; unique roles, max 5.
};
struct Position {
  std::string partition;
  sqfv::Generation producer_generation{};
  std::uint64_t batch_sequence = 0;
};
namespace detail {
struct CaptureState;
}

// Optional immutable capture representation. No I/O, provider parser, clock
// sampling, credential resolution or instruction execution occurs here.
// Fallible operations preserve outputs on failure. Borrowed views require a
// live retaining handle; destruction/move must not race calls on that handle.
class Capture final {
public:
  Capture() noexcept;
  ~Capture() noexcept;
  Capture(Capture &&) noexcept;
  Capture &operator=(Capture &&) noexcept;
  Capture(const Capture &) = delete;
  Capture &operator=(const Capture &) = delete;
  [[nodiscard]] explicit operator bool() const noexcept;
  [[nodiscard]] static Status create(const Description &, ByteView original,
                                     const Limits &, Capture &out) noexcept;
  [[nodiscard]] static Status resolve(ByteView encoded,
                                      std::string_view expected_reference,
                                      const Limits &, Capture &out) noexcept;
  [[nodiscard]] Status retain(Capture &out) const noexcept;
  // Requires nonempty handle. Other accessors return empty views for empty
  // handles.
  [[nodiscard]] const Description &description() const noexcept;
  [[nodiscard]] ByteView encoded() const noexcept;
  [[nodiscard]] ByteView original() const noexcept;
  [[nodiscard]] std::string_view reference() const noexcept;
  [[nodiscard]] std::string_view source_reference() const noexcept;
  // One SQFV record denotes one capture envelope, not the provider record
  // count.
  [[nodiscard]] Status prepare(sqfv::Context &, const sqmv::Manifest &,
                               const Position &,
                               sqfv::Batch &out) const noexcept;
  [[nodiscard]] static Status from_delivery(ByteView, const sqfv::Descriptor &,
                                            const sqmv::Manifest &,
                                            const Limits &,
                                            Capture &out) noexcept;

private:
  std::shared_ptr<const detail::CaptureState> state_;
};
} // namespace symphony::sqav
#endif
