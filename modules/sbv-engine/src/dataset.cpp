#include "dataset.hpp"
#include <algorithm>
#include <symphony/knowledge/engine/path.hpp>
#include <sys/mman.h>
namespace symphony::sbv::detail {
namespace db = sqav::databento;
Dataset::~Dataset() {
  if (locked)
    ::munlock(events.data(), events.size() * sizeof(db::Mbo));
}
void Dataset::bind(const Json &p) const {
  need(p.at("source_path") == path && p.at("source_sha256") == sha256 &&
           p.at("dataset") == dataset_name,
       "resident source identity mismatch");
}
Json Dataset::evidence(bool resident) const {
  return {{"mode", resident ? "resident" : "file"},
          {"resident_identity", resident ? resident_identity : Json(nullptr)},
          {"source_reads_this_job", resident ? "0" : "1"},
          {"decodes_this_job", resident ? "0" : "1"},
          {"decoded_bytes", dec(events.size() * sizeof(db::Mbo))},
          {"source_bytes", dec(source_bytes)},
          {"residency", locked ? "locked" : "pageable"}};
}
std::unique_ptr<Dataset> load_dataset(const Json &p, std::int64_t end,
                                      std::uint64_t budget, bool lock) {
  deadline(end);
  auto d = std::make_unique<Dataset>();
  d->path = str(p.at("source_path"));
  d->sha256 = str(p.at("source_sha256"));
  d->dataset_name = str(p.at("dataset"));
  need(d->path.starts_with('/') && e::is_safe_relative_path(d->path.substr(1)),
       "absolute no-follow source path required");
  const auto bytes = e::read_regular_file_no_follow(
      "/", d->path.substr(1), std::min<std::uint64_t>(64U << 20, budget), end);
  need(e::sha256_hex(bytes) == d->sha256, "source byte digest mismatch");
  db::FileView view;
  need(db::FileView::inspect(
           std::span<const unsigned char>(
               reinterpret_cast<const unsigned char *>(bytes.data()),
               bytes.size()),
           {64U << 20, 1U << 20, 200000}, view) == db::Status::ok,
       "Databento DBN input rejected");
  d->metadata = view.metadata();
  need(d->metadata.dataset == d->dataset_name && d->metadata.record_count > 0,
       "DBN dataset/count mismatch");
  d->metadata.dataset = d->dataset_name;
  d->source_bytes = bytes.size();
  d->load_buffer_bytes =
      bytes.capacity() + 1 + d->metadata.record_count * sizeof(db::Mbo);
  need(d->load_buffer_bytes <= budget, "dataset load buffer budget exceeded");
  d->events.resize(d->metadata.record_count);
  d->load_buffer_bytes =
      bytes.capacity() + 1 + d->events.capacity() * sizeof(db::Mbo);
  need(d->load_buffer_bytes <= budget, "dataset allocation budget exceeded");
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
