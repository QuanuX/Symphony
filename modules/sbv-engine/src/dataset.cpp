#include "dataset.hpp"
#include <algorithm>
#include <limits>
#include <optional>
#include <symphony/knowledge/engine/path.hpp>
#include <sys/mman.h>
namespace symphony::sbv::detail {
namespace db = sqav::databento;
namespace {
std::optional<std::uint64_t> selected(const Json &v) {
  if (v.is_null())
    return std::nullopt;
  const auto n = u64(v);
  need(n > 0, "selected dataset limits must be positive or null");
  return n;
}
Json limits(const Json &p) {
  Json value = p.value("dataset_limits", Json{{"max_source_bytes", nullptr},
                                              {"max_source_events", nullptr},
                                              {"max_metadata_bytes", nullptr}});
  keys(value, {"max_source_bytes", "max_source_events", "max_metadata_bytes"});
  for (const auto &v : value)
    (void)selected(v);
  return value;
}
void within(std::uint64_t actual, const Json &limit, const char *why) {
  const auto n = selected(limit);
  need(!n || actual <= *n, why);
}
std::uint64_t buffer_bytes(std::size_t raw_capacity, std::size_t events) {
  need(raw_capacity < UINT64_MAX &&
           events <= (UINT64_MAX - raw_capacity - 1) / sizeof(db::Mbo),
       "dataset buffer accounting exceeds host integer representation");
  return raw_capacity + 1 + events * sizeof(db::Mbo);
}
} // namespace
Dataset::~Dataset() {
  if (locked)
    ::munlock(events.data(), events.size() * sizeof(db::Mbo));
}
Json Dataset::limits_evidence() const { return selected_limits; }
void Dataset::bind(const Json &p) const {
  need(p.at("source_path") == path && p.at("source_sha256") == sha256 &&
           p.at("dataset") == dataset_name,
       "resident source identity mismatch");
  const auto l = limits(p);
  within(source_bytes, l.at("max_source_bytes"),
         "user source byte limit exceeded");
  within(metadata.record_count, l.at("max_source_events"),
         "user source event limit exceeded");
  within(metadata_bytes, l.at("max_metadata_bytes"),
         "user metadata byte limit exceeded");
  within(load_buffer_bytes, p.value("memory_budget_bytes", Json(nullptr)),
         "user dataset load buffer budget exceeded");
}
Json Dataset::evidence(bool resident) const {
  return {{"mode", resident ? "resident" : "file"},
          {"resident_identity", resident ? resident_identity : Json(nullptr)},
          {"source_reads_this_job", resident ? "0" : "1"},
          {"decodes_this_job", resident ? "0" : "1"},
          {"decoded_bytes", dec(events.size() * sizeof(db::Mbo))},
          {"source_bytes", dec(source_bytes)},
          {"dataset_limits", selected_limits},
          {"memory_budget_bytes", memory_budget},
          {"reader_contract", db::dataset_limits_contract},
          {"residency", locked ? "locked" : "pageable"}};
}
std::unique_ptr<Dataset> load_dataset(const Json &p, std::int64_t end,
                                      bool lock) {
  deadline(end);
  auto d = std::make_unique<Dataset>();
  d->selected_limits = limits(p);
  d->memory_budget = p.value("memory_budget_bytes", Json(nullptr));
  const auto budget = selected(d->memory_budget);
  db::DatasetLimits decoder_limits{
      selected(d->selected_limits.at("max_source_bytes")),
      selected(d->selected_limits.at("max_metadata_bytes")),
      selected(d->selected_limits.at("max_source_events"))};
  auto read_limit = std::numeric_limits<std::size_t>::max();
  if (decoder_limits.max_file_bytes)
    read_limit =
        std::min<std::uint64_t>(read_limit, *decoder_limits.max_file_bytes);
  if (budget)
    read_limit = std::min<std::uint64_t>(read_limit, *budget - 1);
  d->path = str(p.at("source_path"));
  d->sha256 = str(p.at("source_sha256"));
  d->dataset_name = str(p.at("dataset"));
  need(d->path.starts_with('/') && e::is_safe_relative_path(d->path.substr(1)),
       "absolute no-follow source path required");
  const auto bytes =
      e::read_regular_file_no_follow("/", d->path.substr(1), read_limit, end);
  need(e::sha256_hex(bytes) == d->sha256, "source byte digest mismatch");
  deadline(end);
  db::FileView view;
  need(db::FileView::inspect_dataset(
           std::span<const unsigned char>(
               reinterpret_cast<const unsigned char *>(bytes.data()),
               bytes.size()),
           decoder_limits, view) == db::Status::ok,
       "Databento DBN input rejected by format or user dataset limits");
  d->metadata = view.metadata();
  need(d->metadata.dataset == d->dataset_name && d->metadata.record_count > 0,
       "DBN dataset/count mismatch");
  d->metadata.dataset = d->dataset_name;
  d->source_bytes = bytes.size();
  d->metadata_bytes = view.encoded_metadata().size();
  need(d->metadata.record_count <= d->events.max_size(),
       "dataset event storage exceeds host container representation");
  d->load_buffer_bytes =
      buffer_bytes(bytes.capacity(), d->metadata.record_count);
  within(d->load_buffer_bytes, d->memory_budget,
         "user dataset load buffer budget exceeded");
  d->events.resize(static_cast<std::size_t>(d->metadata.record_count));
  d->load_buffer_bytes = buffer_bytes(bytes.capacity(), d->events.capacity());
  d->bind(p);
  for (std::size_t i = 0; i < d->events.size(); ++i) {
    if (i % 1024 == 0)
      deadline(end);
    auto &x = d->events[i];
    need(view.record(i, x) == db::Status::ok && x.ts_recv != UINT64_MAX &&
             (i == 0 || (x.ts_recv >= d->events[i - 1].ts_recv &&
                         x.instrument_id == d->events[0].instrument_id)),
         "one instrument with ordered known receive timestamps required");
    d->book_compatible &= x.publisher_id == d->events[0].publisher_id &&
                          x.channel_id == d->events[0].channel_id;
  }
  if (lock) {
    need(::mlock(d->events.data(), d->events.size() * sizeof(db::Mbo)) == 0,
         "requested locked RAM unavailable; no pageable fallback");
    d->locked = true;
  }
  deadline(end);
  return d;
}
} // namespace symphony::sbv::detail
