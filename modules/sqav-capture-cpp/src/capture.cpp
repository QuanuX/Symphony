#include <algorithm>
#include <array>
#include <cstring>
#include <new>
#include <stdexcept>
#include <symphony/knowledge/engine/digest.hpp>
#include <symphony/sqav/capture.hpp>
#include <utility>

namespace symphony::sqav {
namespace detail {
struct CaptureState {
  Description description;
  std::vector<std::uint8_t> bytes;
  std::size_t payload_offset = 0;
  std::string reference;
  std::string source_reference;
};
} // namespace detail
namespace {
constexpr std::size_t header_size = 24, digest_size = 32;
constexpr std::size_t source_field_count = 12;
constexpr std::string_view prefix = "sqac1-sha256-";
constexpr char hex[] = "0123456789abcdef";
bool valid(const Limits &l) noexcept {
  return l.max_capture_bytes > 0 && l.max_capture_bytes <= (64ULL << 20) &&
         l.max_metadata_bytes > 0 && l.max_metadata_bytes <= 65536 &&
         l.max_field_bytes > 0 && l.max_field_bytes <= 4096;
}
template <class S> auto fields(S &s) {
  return std::array{
      &s.provider_ref,      &s.interface_ref,       &s.interface_version,
      &s.operation,         &s.adapter_ref,         &s.adapter_version,
      &s.dataset_id,        &s.dataset_revision,    &s.selection_ref,
      &s.native_schema_ref, &s.native_encoding_ref, &s.access_scope};
}
template <class T> auto time_fields(T &t) {
  return std::array{&t.value, &t.format_ref, &t.clock_ref, &t.precision_ref,
                    &t.evidence_ref};
}
Status measure(const Description &d, std::size_t payload, const Limits &l,
               std::size_t &metadata) noexcept {
  if (!valid(l))
    return Status::invalid_argument;
  if (d.times.size() > 5)
    return Status::limit;
  if (d.times.empty() || static_cast<unsigned>(d.coverage) > 3)
    return Status::invalid_argument;
  std::size_t n = 11; // Coverage, optional-count flag/value, time count.
  auto field = [&](const std::string &s, bool required = true) {
    if (required && s.empty())
      return Status::invalid_argument;
    if (s.size() > l.max_field_bytes)
      return Status::limit;
    n += 2 + s.size();
    return Status::ok;
  };
  for (auto p : fields(d.source))
    if (auto s = field(*p); s != Status::ok)
      return s;
  for (auto p : {&d.attempt_id, &d.attribution_ref, &d.coverage_scope})
    if (auto s = field(*p); s != Status::ok)
      return s;
  if (auto s = field(d.source_position, false); s != Status::ok)
    return s;
  if (auto s = field(d.coverage_evidence_ref, d.coverage != Coverage::unknown);
      s != Status::ok)
    return s;
  unsigned roles = 0;
  for (const auto &t : d.times) {
    const auto r = static_cast<unsigned>(t.role);
    if (r < 1 || r > 5 || (roles & (1U << r)))
      return Status::invalid_argument;
    roles |= 1U << r;
    ++n;
    for (auto p : time_fields(t))
      if (auto s = field(*p); s != Status::ok)
        return s;
  }
  if (!(roles & 2U))
    return Status::invalid_argument;
  if (n > l.max_metadata_bytes || payload > l.max_capture_bytes ||
      header_size + digest_size + n > l.max_capture_bytes - payload)
    return Status::limit;
  metadata = n;
  return Status::ok;
}
template <class I> void put(std::uint8_t *&p, I value) noexcept {
  for (std::size_t i = 0; i < sizeof(I); ++i)
    *p++ = static_cast<std::uint8_t>(value >> ((sizeof(I) - 1 - i) * 8));
}
void put_string(std::uint8_t *&p, std::string_view s) noexcept {
  put(p, static_cast<std::uint16_t>(s.size()));
  if (!s.empty())
    std::memcpy(p, s.data(), s.size());
  p += s.size();
}
template <class I> I get(const std::uint8_t *&p) noexcept {
  I n = 0;
  for (std::size_t i = 0; i < sizeof(I); ++i)
    n = static_cast<I>((n << 8) | *p++);
  return n;
}
std::string digest(ByteView bytes) {
  auto s = knowledge::engine::sha256_hex(bytes);
  if (s.size() != 64)
    throw std::runtime_error("incomplete SHA-256");
  return s;
}
std::string source_reference(ByteView encoded_source) {
  std::string input("SQAS1\0", 6);
  input.append(reinterpret_cast<const char *>(encoded_source.data()),
               encoded_source.size());
  auto d = knowledge::engine::sha256_hex(input);
  if (d.size() != 64)
    throw std::runtime_error("incomplete SHA-256");
  return "sqas1-sha256-" + d;
}
unsigned nibble(char c) noexcept {
  return c <= '9' ? static_cast<unsigned>(c - '0')
                  : static_cast<unsigned>(c - 'a' + 10);
}
bool reference_valid(std::string_view r) noexcept {
  return r.size() == prefix.size() + 64 && r.starts_with(prefix) &&
         std::all_of(r.begin() + static_cast<std::ptrdiff_t>(prefix.size()),
                     r.end(), [](char c) {
                       return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f');
                     });
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
Status manifest_matches(const Capture &c, const sqmv::Manifest &m) noexcept {
  if (!c || !m)
    return Status::invalid_argument;
  const auto &d = c.description();
  const auto &md = m.description();
  if (md.dataset_id != d.source.dataset_id ||
      md.dataset_revision != d.source.dataset_revision ||
      md.access_scope != d.source.access_scope ||
      md.schema_version != capture_schema ||
      md.layout_version != capture_layout)
    return Status::binding_mismatch;
  for (const auto &e : md.evidence)
    if (e.role == sqmv::EvidenceRole::source &&
        e.producer_ref == d.attribution_ref &&
        e.evidence_ref == c.source_reference())
      return Status::ok;
  return Status::binding_mismatch;
}
} // namespace
Capture::Capture() noexcept = default;
Capture::~Capture() noexcept = default;
Capture::Capture(Capture &&) noexcept = default;
Capture &Capture::operator=(Capture &&) noexcept = default;
Capture::operator bool() const noexcept { return static_cast<bool>(state_); }
const Description &Capture::description() const noexcept {
  return state_->description;
}
ByteView Capture::encoded() const noexcept {
  return state_ ? ByteView(state_->bytes) : ByteView{};
}
ByteView Capture::original() const noexcept {
  return state_ ? encoded().subspan(state_->payload_offset,
                                    state_->bytes.size() -
                                        state_->payload_offset - digest_size)
                : ByteView{};
}
std::string_view Capture::reference() const noexcept {
  return state_ ? state_->reference : std::string_view{};
}
std::string_view Capture::source_reference() const noexcept {
  return state_ ? state_->source_reference : std::string_view{};
}
Status Capture::retain(Capture &out) const noexcept {
  if (!state_)
    return Status::invalid_argument;
  Capture ready;
  ready.state_ = state_;
  out = std::move(ready);
  return Status::ok;
}
Status Capture::create(const Description &d, ByteView raw, const Limits &l,
                       Capture &out) noexcept {
  std::size_t metadata = 0;
  if (auto s = measure(d, raw.size(), l, metadata); s != Status::ok)
    return s;
  try {
    auto state = std::make_shared<detail::CaptureState>();
    state->description = d;
    auto &canonical = state->description;
    std::sort(canonical.times.begin(), canonical.times.end(),
              [](const auto &a, const auto &b) { return a.role < b.role; });
    state->payload_offset = header_size + metadata;
    state->bytes.resize(header_size + metadata + raw.size() + digest_size);
    auto *p = state->bytes.data();
    std::memcpy(p, "SQA1", 4);
    p += 4;
    put(p, std::uint16_t{1});
    put(p, std::uint16_t{0});
    put(p, std::uint32_t{0});
    put(p, static_cast<std::uint32_t>(metadata));
    put(p, static_cast<std::uint64_t>(raw.size()));
    for (auto f : fields(canonical.source))
      put_string(p, *f);
    state->source_reference = symphony::sqav::source_reference(
        {state->bytes.data() + header_size,
         static_cast<std::size_t>(p - state->bytes.data() - header_size)});
    put_string(p, canonical.attempt_id);
    put_string(p, canonical.attribution_ref);
    put_string(p, canonical.source_position);
    *p++ = static_cast<std::uint8_t>(canonical.coverage);
    put_string(p, canonical.coverage_scope);
    put_string(p, canonical.coverage_evidence_ref);
    *p++ = canonical.source_record_count.has_value() ? 1 : 0;
    put(p, canonical.source_record_count.value_or(0));
    *p++ = static_cast<std::uint8_t>(canonical.times.size());
    for (const auto &t : canonical.times) {
      *p++ = static_cast<std::uint8_t>(t.role);
      for (auto f : time_fields(t))
        put_string(p, *f);
    }
    if (static_cast<std::size_t>(p - state->bytes.data()) !=
        state->payload_offset)
      return Status::internal_error;
    if (!raw.empty())
      std::memcpy(p, raw.data(), raw.size());
    p += raw.size();
    const auto hash =
        digest(ByteView(state->bytes).first(state->bytes.size() - digest_size));
    for (std::size_t i = 0; i < digest_size; ++i)
      *p++ = static_cast<std::uint8_t>((nibble(hash[i * 2]) << 4) |
                                       nibble(hash[i * 2 + 1]));
    state->reference = std::string(prefix) + hash;
    Capture ready;
    ready.state_ = std::move(state);
    out = std::move(ready);
    return Status::ok;
  } catch (const std::bad_alloc &) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}

Status Capture::resolve(ByteView bytes, std::string_view expected,
                        const Limits &l, Capture &out) noexcept {
  if (!valid(l) || !reference_valid(expected))
    return Status::invalid_argument;
  if (bytes.size() > l.max_capture_bytes)
    return Status::limit;
  if (bytes.size() < header_size + digest_size ||
      std::memcmp(bytes.data(), "SQA1", 4) != 0)
    return Status::corrupt_capture;
  const auto *p = bytes.data() + 4;
  const auto major = get<std::uint16_t>(p), minor = get<std::uint16_t>(p);
  const auto flags = get<std::uint32_t>(p), metadata = get<std::uint32_t>(p);
  const auto raw_size = get<std::uint64_t>(p);
  if (major != 1 || minor != 0 || flags != 0)
    return Status::unsupported_capture;
  if (metadata > l.max_metadata_bytes)
    return Status::limit;
  if (metadata > bytes.size() - header_size - digest_size ||
      raw_size != bytes.size() - header_size - digest_size - metadata)
    return Status::corrupt_capture;
  const auto *end = p + metadata;
  // Validate bounded string views, counts and canonical order before
  // allocating.
  std::array<std::string_view, source_field_count + 5> values{};
  std::array<std::array<std::string_view, 5>, 5> times{};
  std::array<TimeRole, 5> roles{};
  auto read_string = [&](std::string_view &value, bool required = true) {
    if (end - p < 2)
      return Status::corrupt_capture;
    const auto n = get<std::uint16_t>(p);
    if (n > l.max_field_bytes)
      return Status::limit;
    if (end - p < n || (required && n == 0))
      return Status::corrupt_capture;
    value = {reinterpret_cast<const char *>(p), n};
    p += n;
    return Status::ok;
  };
  for (std::size_t i = 0; i < source_field_count; ++i)
    if (auto s = read_string(values[i]); s != Status::ok)
      return s;
  const auto source_size =
      static_cast<std::size_t>(p - bytes.data() - header_size);
  for (std::size_t i = source_field_count; i < source_field_count + 3; ++i)
    if (auto s = read_string(values[i], i != source_field_count + 2);
        s != Status::ok)
      return s;
  if (p == end)
    return Status::corrupt_capture;
  const auto coverage = *p++;
  if (coverage > 3)
    return Status::unsupported_capture;
  if (auto s = read_string(values[source_field_count + 3]); s != Status::ok)
    return s;
  if (auto s = read_string(values[source_field_count + 4], coverage != 0);
      s != Status::ok)
    return s;
  if (end - p < 10)
    return Status::corrupt_capture;
  const auto has_count = *p++;
  const auto count = get<std::uint64_t>(p);
  const auto time_count = *p++;
  if (has_count > 1 || (!has_count && count != 0) || time_count == 0)
    return Status::corrupt_capture;
  if (time_count > 5)
    return Status::limit;
  unsigned previous = 0;
  for (unsigned i = 0; i < time_count; ++i) {
    if (p == end)
      return Status::corrupt_capture;
    const auto role = *p++;
    if (role < 1 || role > 5)
      return Status::unsupported_capture;
    if (role <= previous || (i == 0 && role != 1))
      return Status::corrupt_capture;
    previous = role;
    roles[i] = static_cast<TimeRole>(role);
    for (auto &f : times[i])
      if (auto s = read_string(f); s != Status::ok)
        return s;
  }
  if (p != end)
    return Status::corrupt_capture;
  try {
    const auto hash = digest(bytes.first(bytes.size() - digest_size));
    const auto trailer = bytes.last(digest_size);
    for (std::size_t i = 0; i < digest_size; ++i)
      if (hex[trailer[i] >> 4] != hash[i * 2] ||
          hex[trailer[i] & 15] != hash[i * 2 + 1])
        return Status::corrupt_capture;
    if (expected.substr(prefix.size()) != hash)
      return Status::reference_mismatch;
    auto state = std::make_shared<detail::CaptureState>();
    auto &d = state->description;
    std::size_t i = 0;
    for (auto f : fields(d.source))
      *f = values[i++];
    d.attempt_id = values[source_field_count];
    d.attribution_ref = values[source_field_count + 1];
    d.source_position = values[source_field_count + 2];
    d.coverage = static_cast<Coverage>(coverage);
    d.coverage_scope = values[source_field_count + 3];
    d.coverage_evidence_ref = values[source_field_count + 4];
    if (has_count)
      d.source_record_count = count;
    d.times.resize(time_count);
    for (unsigned j = 0; j < time_count; ++j) {
      auto &t = d.times[j];
      t.role = roles[j];
      i = 0;
      for (auto f : time_fields(t))
        *f = times[j][i++];
    }
    state->bytes.assign(bytes.begin(), bytes.end());
    state->payload_offset = header_size + metadata;
    state->reference = std::string(prefix) + hash;
    state->source_reference = symphony::sqav::source_reference(
        bytes.subspan(header_size, source_size));
    Capture ready;
    ready.state_ = std::move(state);
    out = std::move(ready);
    return Status::ok;
  } catch (const std::bad_alloc &) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}
Status Capture::metadata(bool raw, const sqmv::EvidenceReference& access,
                         const sqmv::Limits& limits, sqmv::Manifest& out) const noexcept {
  if (!state_ || access.role != sqmv::EvidenceRole::access) return Status::invalid_argument;
  const auto& d = description();
  if (raw && (!d.source_record_count || *d.source_record_count == 0)) return Status::invalid_argument;
  try {
    sqmv::Description md{d.source.dataset_id, d.source.dataset_revision,
      raw ? d.source.native_schema_ref : std::string(capture_schema),
      raw ? d.source.native_encoding_ref : std::string(capture_layout),
      d.source.access_scope, d.attribution_ref, {}};
    md.evidence = {{sqmv::EvidenceRole::schema, d.attribution_ref, md.schema_version},
      {sqmv::EvidenceRole::layout, d.attribution_ref, md.layout_version}, access,
      {sqmv::EvidenceRole::source, d.attribution_ref, std::string(source_reference())},
      {sqmv::EvidenceRole::lineage, d.attribution_ref, std::string(reference())}};
    for (const auto& time : d.times) {
      const auto duplicate = std::any_of(md.evidence.begin(), md.evidence.end(), [&](const auto& e) {
        return e.role == sqmv::EvidenceRole::time && e.evidence_ref == time.evidence_ref;
      });
      if (!duplicate) md.evidence.push_back({sqmv::EvidenceRole::time, d.attribution_ref, time.evidence_ref});
    }
    if (d.coverage != Coverage::unknown)
      md.evidence.push_back({sqmv::EvidenceRole::coverage, d.attribution_ref, d.coverage_evidence_ref});
    const auto status = sqmv::Manifest::create(md, limits, out);
    switch (status) {
      case sqmv::Status::ok: return Status::ok;
      case sqmv::Status::limit: return Status::limit;
      case sqmv::Status::no_memory: return Status::no_memory;
      case sqmv::Status::invalid_argument: return Status::invalid_argument;
      default: return Status::internal_error;
    }
  } catch (const std::bad_alloc&) { return Status::no_memory; }
    catch (...) { return Status::internal_error; }
}
Status Capture::prepare_original(sqfv::Context& context, const sqmv::Manifest& manifest,
                                 const Position& position, sqfv::Batch& out) const noexcept {
  if (!state_ || !manifest || !description().source_record_count ||
      *description().source_record_count == 0) return Status::invalid_argument;
  const auto& source = description().source;
  const auto& md = manifest.description();
  if (md.dataset_id != source.dataset_id || md.dataset_revision != source.dataset_revision ||
      md.access_scope != source.access_scope || md.schema_version != source.native_schema_ref ||
      md.layout_version != source.native_encoding_ref) return Status::binding_mismatch;
  bool has_source = false, has_capture = false;
  for (const auto& e : md.evidence) {
    if (e.producer_ref != description().attribution_ref) continue;
    has_source |= e.role == sqmv::EvidenceRole::source && e.evidence_ref == source_reference();
    has_capture |= e.role == sqmv::EvidenceRole::lineage && e.evidence_ref == reference();
  }
  if (!has_source || !has_capture) return Status::binding_mismatch;
  try {
    sqfv::Descriptor d;
    const auto bound = manifest.binding(d.binding);
    if (bound == sqmv::Status::no_memory) return Status::no_memory;
    if (bound != sqmv::Status::ok) return Status::internal_error;
    d.partition = position.partition; d.producer_generation = position.producer_generation;
    d.batch_sequence = position.batch_sequence; d.record_count = *description().source_record_count;
    d.source_binding = source_reference(); d.source_position = reference();
    return flow_status(context.prepare_copy(d, original(), out));
  } catch (const std::bad_alloc&) { return Status::no_memory; }
    catch (...) { return Status::internal_error; }
}

Status Capture::prepare(sqfv::Context &context, const sqmv::Manifest &manifest,
                        const Position &pos, sqfv::Batch &out) const noexcept {
  if (auto s = manifest_matches(*this, manifest); s != Status::ok)
    return s;
  try {
    sqfv::Descriptor d;
    const auto bound = manifest.binding(d.binding);
    if (bound == sqmv::Status::no_memory)
      return Status::no_memory;
    if (bound != sqmv::Status::ok)
      return Status::internal_error;
    d.partition = pos.partition;
    d.producer_generation = pos.producer_generation;
    d.batch_sequence = pos.batch_sequence;
    d.source_binding = source_reference();
    d.source_position = reference();
    d.record_count = 1;
    return flow_status(context.prepare_copy(d, encoded(), out));
  } catch (const std::bad_alloc &) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}
Status Capture::from_delivery(ByteView bytes, const sqfv::Descriptor &d,
                              const sqmv::Manifest &manifest, const Limits &l,
                              Capture &out) noexcept {
  if (!manifest)
    return Status::invalid_argument;
  if (manifest.verify_binding(d.binding) != sqmv::Status::ok ||
      d.record_count != 1)
    return Status::binding_mismatch;
  Capture ready;
  if (auto s = resolve(bytes, d.source_position, l, ready); s != Status::ok)
    return s;
  if (auto s = manifest_matches(ready, manifest); s != Status::ok)
    return s;
  if (d.source_binding != ready.source_reference())
    return Status::binding_mismatch;
  out = std::move(ready);
  return Status::ok;
}
} // namespace symphony::sqav
