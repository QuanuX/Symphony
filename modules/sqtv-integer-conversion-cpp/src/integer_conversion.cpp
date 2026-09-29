#include "limits.hpp"
#include <algorithm>
#include <array>
#include <limits>
#include <new>
#include <symphony/knowledge/engine/digest.hpp>
#include <symphony/sqtv/integer_conversion.hpp>
#include <utility>
#include <vector>
namespace symphony::sqtv {
namespace detail {
struct ResultState {
  sqfv::Batch batch;
  sqmv::Manifest metadata;
  std::string operation;
  std::string input;
};
} // namespace detail
namespace {
constexpr std::array<std::string_view, 16> layouts{
    "sqtv-int-v1-i8-le",  "sqtv-int-v1-i8-be",  "sqtv-int-v1-u8-le",
    "sqtv-int-v1-u8-be",  "sqtv-int-v1-i16-le", "sqtv-int-v1-i16-be",
    "sqtv-int-v1-u16-le", "sqtv-int-v1-u16-be", "sqtv-int-v1-i32-le",
    "sqtv-int-v1-i32-be", "sqtv-int-v1-u32-le", "sqtv-int-v1-u32-be",
    "sqtv-int-v1-i64-le", "sqtv-int-v1-i64-be", "sqtv-int-v1-u64-le",
    "sqtv-int-v1-u64-be"};
constexpr std::array<std::uint8_t, 4> widths{8, 16, 32, 64};
int index(Format f) noexcept {
  const auto width = std::find(widths.begin(), widths.end(), f.bits);
  if (width == widths.end() ||
      (f.signedness != Signedness::signed_integer &&
       f.signedness != Signedness::unsigned_integer) ||
      (f.order != ByteOrder::little && f.order != ByteOrder::big))
    return -1;
  return static_cast<int>(width - widths.begin()) * 4 +
         (static_cast<int>(f.signedness) - 1) * 2 +
         (static_cast<int>(f.order) - 1);
}
std::uint64_t mask(unsigned bits) noexcept {
  return bits == 64 ? UINT64_MAX : (std::uint64_t{1} << bits) - 1;
}
std::uint64_t read(const std::uint8_t *p, Format f) noexcept {
  std::uint64_t n = 0;
  const unsigned bytes = f.bits / 8;
  for (unsigned i = 0; i < bytes; ++i)
    n = (n << 8) | p[f.order == ByteOrder::big ? i : bytes - 1 - i];
  return n;
}
// Signed magnitude is derived entirely with unsigned arithmetic, including
// INT64_MIN; no signed overflow or implementation-defined narrowing occurs.
bool recode(std::uint64_t bits, Format from, Format to,
            std::uint64_t &out) noexcept {
  const bool negative = from.signedness == Signedness::signed_integer &&
                        (bits & (std::uint64_t{1} << (from.bits - 1)));
  const auto magnitude = negative ? ((~bits + 1) & mask(from.bits)) : bits;
  if (negative) {
    if (to.signedness == Signedness::unsigned_integer ||
        magnitude > (std::uint64_t{1} << (to.bits - 1)))
      return false;
    out = (std::uint64_t{0} - magnitude) & mask(to.bits);
  } else {
    const auto maximum = to.signedness == Signedness::signed_integer
                             ? mask(to.bits) >> 1
                             : mask(to.bits);
    if (magnitude > maximum)
      return false;
    out = magnitude;
  }
  return true;
}
void write(std::uint8_t *p, Format f, std::uint64_t n) noexcept {
  const unsigned bytes = f.bits / 8;
  for (unsigned i = 0; i < bytes; ++i) {
    p[f.order == ByteOrder::little ? i : bytes - 1 - i] =
        static_cast<std::uint8_t>(n);
    n >>= 8;
  }
}
Status flow_status(sqfv::Status s) noexcept {
  switch (s) {
  case sqfv::Status::ok:
    return Status::ok;
  case sqfv::Status::invalid_argument:
    return Status::invalid_argument;
  case sqfv::Status::limit:
    return Status::limit;
  case sqfv::Status::no_memory:
    return Status::no_memory;
  case sqfv::Status::stale:
    return Status::stale;
  case sqfv::Status::closed:
    return Status::closed;
  case sqfv::Status::binding_mismatch:
  case sqfv::Status::scope_mismatch:
    return Status::binding_mismatch;
  default:
    return Status::internal_error;
  }
}
Status metadata_status(sqmv::Status s) noexcept {
  switch (s) {
  case sqmv::Status::ok:
    return Status::ok;
  case sqmv::Status::limit:
    return Status::limit;
  case sqmv::Status::no_memory:
    return Status::no_memory;
  case sqmv::Status::invalid_argument:
    return Status::invalid_argument;
  case sqmv::Status::binding_mismatch:
    return Status::binding_mismatch;
  default:
    return Status::internal_error;
  }
}
std::string input_reference(const sqfv::ContentId &id) {
  constexpr char hex[] = "0123456789abcdef";
  std::string s = "sqtv-input-sha256-";
  s.reserve(s.size() + 64);
  for (auto byte : id) {
    s.push_back(hex[byte >> 4]);
    s.push_back(hex[byte & 15]);
  }
  return s;
}
void lineage(sqmv::Description &d, std::string_view ref) {
  for (const auto &e : d.evidence)
    if (e.role == sqmv::EvidenceRole::lineage &&
        e.producer_ref == converter_identity && e.evidence_ref == ref)
      return;
  d.evidence.push_back({sqmv::EvidenceRole::lineage,
                        std::string(converter_identity), std::string(ref)});
}
} // namespace
Status layout_id(Format f, std::string &out) noexcept {
  const auto i = index(f);
  if (i < 0)
    return Status::unsupported_representation;
  try {
    std::string ready(layouts[static_cast<std::size_t>(i)]);
    out = std::move(ready);
    return Status::ok;
  } catch (const std::bad_alloc &) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}
Status parse_layout(std::string_view s, Format &out) noexcept {
  const auto found = std::find(layouts.begin(), layouts.end(), s);
  if (found == layouts.end())
    return Status::unsupported_representation;
  const auto i = static_cast<unsigned>(found - layouts.begin());
  out = {widths[i / 4], static_cast<Signedness>((i % 4) / 2 + 1),
         static_cast<ByteOrder>(i % 2 + 1)};
  return Status::ok;
}
Status operation_reference(Format from, Format to, std::string &out) noexcept {
  if (index(from) < 0 || index(to) < 0)
    return Status::unsupported_representation;
  try {
    constexpr char domain[] = "symphony.sqtv.integer-conversion.v1";
    std::string bytes(domain, sizeof(domain));
    for (auto f : {from, to}) {
      bytes.push_back(static_cast<char>(f.bits));
      bytes.push_back(static_cast<char>(f.signedness));
      bytes.push_back(static_cast<char>(f.order));
    }
    auto h = knowledge::engine::sha256_hex(bytes);
    if (h.size() != 64)
      return Status::internal_error;
    auto ready = "sqtv-int1-sha256-" + h;
    out = std::move(ready);
    return Status::ok;
  } catch (const std::bad_alloc &) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}
Result::Result() noexcept = default;
Result::~Result() noexcept = default;
Result::Result(Result &&) noexcept = default;
Result &Result::operator=(Result &&) noexcept = default;
Result::operator bool() const noexcept { return static_cast<bool>(state_); }
const sqfv::Batch &Result::batch() const noexcept { return state_->batch; }
const sqmv::Manifest &Result::metadata() const noexcept {
  return state_->metadata;
}
std::string_view Result::operation_reference() const noexcept {
  return state_ ? state_->operation : std::string_view{};
}
std::string_view Result::input_reference() const noexcept {
  return state_ ? state_->input : std::string_view{};
}
Status Result::convert(sqfv::Context &context, const sqfv::Batch &input,
                       const sqmv::Manifest &manifest, Format output,
                       const Position &position, const Limits &limits,
                       Result &out) noexcept {
  if (!context || !input || !manifest || !detail::valid_limits(limits))
    return Status::invalid_argument;
  if (index(output) < 0)
    return Status::unsupported_representation;
  if (manifest.verify_binding(input.descriptor().binding) != sqmv::Status::ok)
    return Status::binding_mismatch;
  if (manifest.description().schema_version != integer_schema)
    return Status::unsupported_representation;
  Format from;
  if (auto s = parse_layout(manifest.description().layout_version, from);
      s != Status::ok)
    return s;
  const auto count = input.descriptor().record_count;
  const std::uint64_t from_bytes = from.bits / 8, to_bytes = output.bits / 8;
  if (count == 0)
    return Status::malformed_input;
  if (count > limits.max_elements ||
      count > limits.max_input_bytes / from_bytes ||
      count > limits.max_output_bytes / to_bytes)
    return Status::limit;
  try {
    sqfv::Lease lease;
    const auto acquired =
        input.acquire(manifest.description().access_scope, lease);
    if (acquired != sqfv::Status::ok)
      return flow_status(acquired);
    const auto payload = lease.payload();
    if (payload.size() != count * from_bytes)
      return Status::malformed_input;
    std::uint64_t value = 0;
    for (std::size_t i = 0; i < count; ++i)
      if (!recode(read(payload.data() + i * from_bytes, from), from, output,
                  value))
        return Status::overflow;
    auto state = std::make_unique<detail::ResultState>();
    if (auto s =
            symphony::sqtv::operation_reference(from, output, state->operation);
        s != Status::ok)
      return s;
    state->input = symphony::sqtv::input_reference(input.content_id());
    auto derived = manifest.description();
    if (auto s = layout_id(output, derived.layout_version); s != Status::ok)
      return s;
    derived.producer_ref = converter_identity;
    std::erase_if(derived.evidence, [](const auto &e) {
      return e.role == sqmv::EvidenceRole::layout;
    });
    derived.evidence.push_back({sqmv::EvidenceRole::layout,
                                std::string(converter_identity),
                                derived.layout_version});
    lineage(derived, manifest.reference());
    lineage(derived, state->input);
    lineage(derived, state->operation);
    if (auto s = metadata_status(sqmv::Manifest::create(
            derived, {65536, 4096, 128}, state->metadata));
        s != Status::ok)
      return s;
    std::vector<std::uint8_t> converted(
        static_cast<std::size_t>(count * to_bytes));
    for (std::size_t i = 0; i < count; ++i) {
      if (!recode(read(payload.data() + i * from_bytes, from), from, output,
                  value))
        return Status::internal_error;
      write(converted.data() + i * to_bytes, output, value);
    }
    sqfv::Descriptor descriptor;
    if (auto s = metadata_status(state->metadata.binding(descriptor.binding));
        s != Status::ok)
      return s;
    descriptor.partition = position.partition;
    descriptor.producer_generation = position.producer_generation;
    descriptor.batch_sequence = position.batch_sequence;
    descriptor.record_count = count;
    descriptor.source_binding = state->operation;
    descriptor.source_position = state->input;
    if (auto s = flow_status(
            context.prepare_copy(descriptor, converted, state->batch));
        s != Status::ok)
      return s;
    Result ready;
    ready.state_ = std::move(state);
    out = std::move(ready);
    return Status::ok;
  } catch (const std::bad_alloc &) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}
} // namespace symphony::sqtv
