#ifndef SYMPHONY_SQTV_INTEGER_CONVERSION_HPP
#define SYMPHONY_SQTV_INTEGER_CONVERSION_HPP
#include <cstdint>
#include <memory>
#include <string>
#include <string_view>
#include <symphony/sqmv/metadata.hpp>
namespace symphony::sqtv {
inline constexpr std::string_view integer_schema = "sqtv-integer-values-v1";
inline constexpr std::string_view converter_identity =
    "sqtv-integer-conversion-cpp/0.2.0-dev";
enum class Signedness : std::uint8_t {
  signed_integer = 1,
  unsigned_integer = 2
};
enum class ByteOrder : std::uint8_t { little = 1, big = 2 };
struct Format {
  std::uint8_t bits = 0;
  Signedness signedness = Signedness::signed_integer;
  ByteOrder order = ByteOrder::little;
  bool operator==(const Format &) const = default;
};
enum class Status : std::uint8_t {
  ok,
  invalid_argument,
  unsupported_representation,
  binding_mismatch,
  malformed_input,
  overflow,
  limit,
  no_memory,
  stale,
  closed,
  internal_error
};
struct Limits {
  std::uint64_t max_elements = 0;     // Positive, at most 8,388,608.
  std::uint64_t max_input_bytes = 0;  // Positive, at most 64 MiB.
  std::uint64_t max_output_bytes = 0; // Positive, at most 64 MiB.
};
struct Position {
  std::string partition;
  sqfv::Generation producer_generation{};
  std::uint64_t batch_sequence = 0;
};
// Exact layout IDs, not provider format detection. Failure preserves out.
[[nodiscard]] Status layout_id(Format, std::string &out) noexcept;
[[nodiscard]] Status parse_layout(std::string_view, Format &out) noexcept;
[[nodiscard]] Status operation_reference(Format input, Format output,
                                         std::string &out) noexcept;
namespace detail {
struct ResultState;
}
// Immutable converted batch plus its derived manifest. Borrowed references
// require a nonempty live Result. Use the existing Batch/Manifest retain APIs
// for independently owned handles. Move/destruction must not race borrowed use.
class Result final {
public:
  Result() noexcept;
  ~Result() noexcept;
  Result(Result &&) noexcept;
  Result &operator=(Result &&) noexcept;
  Result(const Result &) = delete;
  Result &operator=(const Result &) = delete;
  [[nodiscard]] explicit operator bool() const noexcept;
  [[nodiscard]] const sqfv::Batch &batch() const noexcept;
  [[nodiscard]] const sqmv::Manifest &metadata() const noexcept;
  [[nodiscard]] std::string_view operation_reference() const noexcept;
  [[nodiscard]] std::string_view input_reference() const noexcept;
  // All values must fit exactly; no clipping, rounding, sentinel or null
  // policy. Input is an actual immutable SQFV Batch; output uses the selected
  // Context. Output is replaced only after manifest and batch preparation both
  // succeed.
  [[nodiscard]] static Status convert(sqfv::Context &output_context,
                                      const sqfv::Batch &input,
                                      const sqmv::Manifest &input_metadata,
                                      Format output_format,
                                      const Position &output_position,
                                      const Limits &, Result &out) noexcept;

private:
  std::unique_ptr<detail::ResultState> state_;
};
} // namespace symphony::sqtv
#endif
