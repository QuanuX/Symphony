#include <algorithm>
#include <bit>
#include <limits>
#include <new>
#include <symphony/sqav/databento/dbn.hpp>
namespace symphony::sqav::databento {
namespace {
template <class T> T le(const std::uint8_t *p) noexcept {
  T n = 0;
  for (std::size_t i = 0; i < sizeof(T); ++i)
    n |= static_cast<T>(p[i]) << (8 * i);
  return n;
}
bool cstr(ByteView b, std::string_view *out = nullptr) noexcept {
  auto end = std::find(b.begin(), b.end(), 0);
  if (end == b.end())
    return false;
  if (out)
    *out = std::string_view(reinterpret_cast<const char *>(b.data()),
                            static_cast<std::size_t>(end - b.begin()));
  return true;
}
struct Cursor {
  ByteView bytes;
  std::size_t at = 108;
  bool take(std::size_t n, ByteView &b) noexcept {
    if (n > bytes.size() - at)
      return false;
    b = bytes.subspan(at, n);
    at += n;
    return true;
  }
  bool count(std::uint32_t &n) noexcept {
    ByteView b;
    if (!take(4, b))
      return false;
    n = le<std::uint32_t>(b.data());
    return true;
  }
  bool strings(std::uint16_t width, std::uint32_t &count_out) noexcept {
    if (!count(count_out) || count_out > (bytes.size() - at) / width)
      return false;
    for (std::uint32_t i = 0; i < count_out; ++i) {
      ByteView b;
      if (!take(width, b) || !cstr(b))
        return false;
    }
    return true;
  }
};
bool binding(const Description &d, const FileView &f) noexcept {
  return d.source.provider_ref == "databento" &&
         d.source.dataset_id == f.metadata().dataset &&
         d.source.native_schema_ref == native_schema &&
         d.source.native_encoding_ref == encoding_for_version(f.metadata().version) &&
         (!d.source_record_count ||
          *d.source_record_count == f.metadata().record_count);
}
Status mapped(sqav::Status s) noexcept {
  switch (s) {
  case sqav::Status::ok:
    return Status::ok;
  case sqav::Status::invalid_argument:
    return Status::invalid_argument;
  case sqav::Status::limit:
    return Status::limit;
  case sqav::Status::no_memory:
    return Status::no_memory;
  default:
    return Status::internal_error;
  }
}
} // namespace
Status decode_mbo(ByteView b, bool ts_out, Mbo &out) noexcept {
  const std::size_t size = ts_out ? 64 : 56;
  if (b.size() != size || b[0] * 4U != size)
    return Status::malformed;
  if (b[1] != 160)
    return Status::unsupported;
  Mbo m;
  m.publisher_id = le<std::uint16_t>(b.data() + 2);
  m.instrument_id = le<std::uint32_t>(b.data() + 4);
  m.ts_event = le<std::uint64_t>(b.data() + 8);
  m.order_id = le<std::uint64_t>(b.data() + 16);
  m.price = std::bit_cast<std::int64_t>(le<std::uint64_t>(b.data() + 24));
  m.size = le<std::uint32_t>(b.data() + 32);
  m.flags = b[36];
  m.channel_id = b[37];
  m.action = b[38];
  m.side = b[39];
  m.ts_recv = le<std::uint64_t>(b.data() + 40);
  m.ts_in_delta = std::bit_cast<std::int32_t>(le<std::uint32_t>(b.data() + 48));
  m.sequence = le<std::uint32_t>(b.data() + 52);
  if (ts_out)
    m.ts_out = le<std::uint64_t>(b.data() + 56);
  out = m;
  return Status::ok;
}
Status FileView::inspect(ByteView b, const Limits &l, FileView &out) noexcept {
  if (l.max_file_bytes == 0 || l.max_file_bytes > (64ULL << 20) ||
      l.max_metadata_bytes == 0 || l.max_metadata_bytes > (1U << 20) ||
      l.max_records == 0 || l.max_records > (1ULL << 20))
    return Status::invalid_argument;
  if (b.size() > l.max_file_bytes)
    return Status::limit;
  if (b.size() < 8)
    return Status::malformed;
  if (b[0] != 'D' || b[1] != 'B' || b[2] != 'N' || (b[3] != 1 && b[3] != 3))
    return Status::unsupported;
  const std::uint64_t meta_size = 8ULL + le<std::uint32_t>(b.data() + 4);
  if (meta_size > l.max_metadata_bytes)
    return Status::limit;
  if (meta_size < 128 || meta_size > b.size() || (b[3] == 3 && meta_size % 8 != 0))
    return Status::malformed;
  if (le<std::uint16_t>(b.data() + 24) != 0)
    return Status::unsupported; // single MBO schema
  FileView ready;
  ready.original_ = b;
  ready.records_offset_ = static_cast<std::size_t>(meta_size);
  auto &m = ready.metadata_;
  if (!cstr(b.subspan(8, 16), &m.dataset) || m.dataset.empty())
    return Status::malformed;
  m.version = b[3];
  m.start = le<std::uint64_t>(b.data() + 26);
  m.end = le<std::uint64_t>(b.data() + 34);
  m.limit = le<std::uint64_t>(b.data() + 42);
  const std::size_t stype_offset = m.version == 1 ? 58 : 50;
  m.stype_in = b[stype_offset];
  m.stype_out = b[stype_offset + 1];
  if (b[stype_offset + 2] > 1)
    return Status::malformed;
  m.ts_out = b[stype_offset + 2] != 0;
  m.symbol_cstr_len = m.version == 1 ? 22 : le<std::uint16_t>(b.data() + 53);
  if (m.symbol_cstr_len == 0 || m.symbol_cstr_len > 4096)
    return Status::unsupported;
  Cursor cursor{b.first(ready.records_offset_)};
  std::uint32_t schema_size = 0;
  if (!cursor.count(schema_size))
    return Status::malformed;
  if (schema_size != 0)
    return Status::unsupported;
  if (!cursor.strings(m.symbol_cstr_len, m.symbols) ||
      !cursor.strings(m.symbol_cstr_len, m.partial) ||
      !cursor.strings(m.symbol_cstr_len, m.not_found) ||
      !cursor.count(m.mappings))
    return Status::malformed;
  if (m.mappings > (cursor.bytes.size() - cursor.at) / (m.symbol_cstr_len + 4U))
    return Status::malformed;
  for (std::uint32_t i = 0; i < m.mappings; ++i) {
    ByteView symbol;
    std::uint32_t intervals = 0;
    if (!cursor.take(m.symbol_cstr_len, symbol) || !cstr(symbol) ||
        !cursor.count(intervals) ||
        intervals >
            (cursor.bytes.size() - cursor.at) / (8U + m.symbol_cstr_len))
      return Status::malformed;
    for (std::uint32_t j = 0; j < intervals; ++j) {
      ByteView interval;
      if (!cursor.take(8U + m.symbol_cstr_len, interval) ||
          !cstr(interval.subspan(8)))
        return Status::malformed;
    }
  }
  if ((m.version == 1 && cursor.bytes.size() != cursor.at) ||
      (m.version == 3 && cursor.bytes.size() - cursor.at > 7))
    return Status::malformed;
  const std::size_t stride = m.ts_out ? 64 : 56;
  const auto body_size = b.size() - ready.records_offset_;
  if (body_size % stride != 0)
    return Status::malformed;
  m.record_count = body_size / stride;
  if (m.record_count > l.max_records)
    return Status::limit;
  Mbo record;
  for (std::uint64_t i = 0; i < m.record_count; ++i) {
    auto status = decode_mbo(
        b.subspan(ready.records_offset_ + static_cast<std::size_t>(i) * stride,
                  stride),
        m.ts_out, record);
    if (status != Status::ok)
      return status;
  }
  out = ready;
  return Status::ok;
}
Status FileView::record(std::uint64_t index, Mbo &out) const noexcept {
  if (!*this || index >= metadata_.record_count)
    return Status::invalid_argument;
  const std::size_t stride = metadata_.ts_out ? 64 : 56;
  return decode_mbo(
      original_.subspan(
          records_offset_ + static_cast<std::size_t>(index) * stride, stride),
      metadata_.ts_out, out);
}
Status capture_file(const Description &d, ByteView bytes, const Limits &limits,
                    const sqav::Limits &capture_limits, Capture &out) noexcept {
  FileView file;
  if (auto s = FileView::inspect(bytes, limits, file); s != Status::ok)
    return s;
  if (!binding(d, file))
    return Status::binding_mismatch;
  if (d.coverage == Coverage::complete &&
      (file.metadata().partial || file.metadata().not_found))
    return Status::binding_mismatch;
  // Bound caller-owned evidence before copying its strings or time vector.
  if (capture_limits.max_capture_bytes == 0 ||
      capture_limits.max_capture_bytes > (64ULL << 20) ||
      capture_limits.max_metadata_bytes == 0 ||
      capture_limits.max_metadata_bytes > 65536 ||
      capture_limits.max_field_bytes == 0 ||
      capture_limits.max_field_bytes > 4096)
    return Status::invalid_argument;
  if (d.times.size() > 5)
    return Status::limit;
  for (const auto *field :
       {&d.source.provider_ref, &d.source.interface_ref,
        &d.source.interface_version, &d.source.operation, &d.source.adapter_ref,
        &d.source.adapter_version, &d.source.dataset_id,
        &d.source.dataset_revision, &d.source.selection_ref,
        &d.source.native_schema_ref, &d.source.native_encoding_ref,
        &d.source.access_scope, &d.attempt_id, &d.attribution_ref,
        &d.source_position, &d.coverage_scope, &d.coverage_evidence_ref})
    if (field->size() > capture_limits.max_field_bytes)
      return Status::limit;
  for (const auto &t : d.times)
    for (const auto *field : {&t.value, &t.format_ref, &t.clock_ref,
                              &t.precision_ref, &t.evidence_ref})
      if (field->size() > capture_limits.max_field_bytes)
        return Status::limit;
  try {
    auto description = d;
    description.source.adapter_ref = adapter_id;
    description.source.adapter_version = adapter_version;
    description.source_record_count = file.metadata().record_count;
    return mapped(Capture::create(description, bytes, capture_limits, out));
  } catch (const std::bad_alloc &) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}
Status inspect_capture(const Capture &capture, const Limits &limits,
                       FileView &out) noexcept {
  if (!capture)
    return Status::invalid_argument;
  FileView ready;
  if (auto s = FileView::inspect(capture.original(), limits, ready);
      s != Status::ok)
    return s;
  const auto &d = capture.description();
  if (!binding(d, ready) || !d.source_record_count ||
      d.source.adapter_ref != adapter_id ||
      d.source.adapter_version != adapter_version ||
      (d.coverage == Coverage::complete &&
       (ready.metadata().partial || ready.metadata().not_found)))
    return Status::binding_mismatch;
  out = ready;
  return Status::ok;
}
} // namespace symphony::sqav::databento
