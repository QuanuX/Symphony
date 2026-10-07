#include "dataset.hpp"
#include "source_owners.hpp"
#include <algorithm>
#include <limits>
#include <optional>
#include <symphony/knowledge/engine/path.hpp>
#include <sys/mman.h>
namespace symphony::sbv::detail {
namespace db = sqav::databento;
namespace {
constexpr const char *retained_buffer_scope =
    "original byte span plus decoded event allocation; excludes owner "
    "capture/batch/store/delivery allocations";
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
std::uint64_t buffer_bytes(std::uint64_t original_bytes, std::size_t events) {
  need(events <= (UINT64_MAX - original_bytes) / sizeof(db::Mbo),
       "dataset buffer accounting exceeds host integer representation");
  return original_bytes + events * sizeof(db::Mbo);
}
} // namespace
Json dataset_source_selection(const Json &p) {
  const bool retained = p.contains("retained_source");
  const unsigned fields = unsigned(p.contains("source_path")) +
                          unsigned(p.contains("source_sha256")) +
                          unsigned(p.contains("dataset"));
  need((retained && fields == 0) || (!retained && fields == 3),
       "select exactly one file source tuple or retained_source");
  if (retained) {
    keys(p.at("retained_source"), {"reference", "delivery"});
    need(p.at("retained_source").at("reference").is_object() &&
             p.at("retained_source").at("delivery").is_object(),
         "retained source reference and delivery objects required");
    return {{"retained_source", p.at("retained_source")}};
  }
  return {{"source_path", str(p.at("source_path"))},
          {"source_sha256", str(p.at("source_sha256"))},
          {"dataset", str(p.at("dataset"))}};
}
Json dataset_result_identity(const Json &choices, const Json &provenance,
                             const Json &resources) {
  const auto feed = resources.is_object() && resources.contains("dataset_feed")
                        ? resources.at("dataset_feed")
                        : Json(nullptr);
  // Historical run/evaluate fixtures predate dataset_feed and may only retain
  // digest/dataset choices. Preserve that admitted profile; current Dataset
  // results carry their full source selection and provenance.
  const bool retained = choices.contains("retained_source");
  if (!retained && feed.is_null()) {
    need(choices.at("source_sha256") == provenance.at("source_sha256") &&
             choices.at("dataset") == provenance.at("dataset"),
         "legacy dataset choices/provenance disagree");
    return {{"source_path",
             provenance.value("source_path",
                              choices.value("source_path", Json(nullptr)))},
            {"source_sha256", provenance.at("source_sha256")},
            {"dataset", provenance.at("dataset")}};
  }
  need(feed.is_object(), "current dataset feed evidence object required");
  const auto mode = str(feed.at("mode"));
  need(mode == "file" || mode == "resident" || mode == "retained_source",
       "unknown dataset feed mode");
  const bool delivered = feed.contains("source_delivery");
  need(mode != "retained_source" || delivered,
       "retained dataset feed delivery evidence missing");
  if (delivered) {
    need(mode != "file" &&
             feed.at("load_buffer_scope") == retained_buffer_scope,
         "retained dataset feed mode or buffer scope mismatch");
    const auto &original =
        feed.at("source_delivery").at("source").at("original");
    need(feed.at("source_bytes") == original.at("bytes") &&
             provenance.at("dbn_version") == original.at("dbn_version"),
         "dataset feed byte count or DBN version disagrees with delivery");
  } else
    need(!feed.contains("load_buffer_scope"),
         "file dataset feed cannot claim retained buffer scope");
  need(feed.at("source_reads_this_job") == (mode == "resident" ? "0" : "1") &&
           feed.at("decodes_this_job") == (mode == "resident" ? "0" : "1"),
       "dataset feed per-job work counters disagree with mode");
  const auto selection = dataset_source_selection(choices);
  const Json identity{{"source_path", provenance.at("source_path")},
                      {"source_sha256", provenance.at("source_sha256")},
                      {"dataset", provenance.at("dataset")}};
  if (retained) {
    need(feed.is_object() && feed.contains("source_delivery") &&
             (feed.at("mode") == "retained_source" ||
              feed.at("mode") == "resident"),
         "retained dataset result delivery evidence missing");
    need(retained_delivery_identity(selection.at("retained_source"),
                                    feed.at("source_delivery")) == identity,
         "retained dataset choices/delivery/provenance disagree");
  } else {
    need(selection == identity, "file dataset choices/provenance disagree");
    if (feed.is_object() && feed.contains("source_delivery")) {
      need(feed.at("mode") == "resident",
           "direct file source cannot claim retained delivery");
      const auto &delivery = feed.at("source_delivery");
      const Json selected{{"reference", delivery.at("reference")},
                          {"delivery", delivery.at("choices")}};
      need(retained_delivery_identity(selected, delivery) == identity,
           "resident delivery/file identity disagree");
    }
  }
  return identity;
}

Dataset::~Dataset() {
  if (locked)
    ::munlock(events.data(), events.size() * sizeof(db::Mbo));
}
Json Dataset::limits_evidence() const { return selected_limits; }
void Dataset::bind(const Json &p) const {
  const auto selected_source = dataset_source_selection(p);
  if (selected_source.contains("retained_source"))
    need(selected_source == source_selection,
         "resident retained source selection mismatch");
  else
    need(selected_source.at("source_path") == path &&
             selected_source.at("source_sha256") == sha256 &&
             selected_source.at("dataset") == dataset_name,
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
  Json result{
      {"mode", resident                    ? "resident"
               : source_delivery.is_null() ? "file"
                                           : "retained_source"},
      {"resident_identity", resident ? resident_identity : Json(nullptr)},
      {"source_reads_this_job", resident ? "0" : "1"},
      {"decodes_this_job", resident ? "0" : "1"},
      {"decoded_bytes", dec(events.size() * sizeof(db::Mbo))},
      {"source_bytes", dec(source_bytes)},
      {"dataset_limits", selected_limits},
      {"memory_budget_bytes", memory_budget},
      {"reader_contract", db::dataset_limits_contract},
      {"residency", locked ? "locked" : "pageable"}};
  if (!source_delivery.is_null()) {
    result["source_delivery"] = source_delivery;
    result["load_buffer_scope"] = load_buffer_scope;
  }
  return result;
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
  d->source_selection = dataset_source_selection(p);
  std::unique_ptr<RetainedOriginal> retained;
  std::string file_bytes;
  std::span<const std::uint8_t> original;
  std::uint64_t raw_buffer_bytes = 0;
  if (d->source_selection.contains("retained_source")) {
    retained = std::make_unique<RetainedOriginal>(p.at("retained_source"), end);
    const auto &identity = retained->identity();
    d->path = str(identity.at("source_path"));
    d->sha256 = str(identity.at("source_sha256"));
    d->dataset_name = str(identity.at("dataset"));
    original = retained->original_bytes();
    raw_buffer_bytes = original.size();
    d->load_buffer_scope = retained_buffer_scope;
  } else {
    auto read_limit = std::numeric_limits<std::size_t>::max();
    if (decoder_limits.max_file_bytes)
      read_limit =
          std::min<std::uint64_t>(read_limit, *decoder_limits.max_file_bytes);
    if (budget)
      read_limit = std::min<std::uint64_t>(read_limit, *budget - 1);
    d->path = str(p.at("source_path"));
    d->sha256 = str(p.at("source_sha256"));
    d->dataset_name = str(p.at("dataset"));
    need(d->path.starts_with('/') &&
             e::is_safe_relative_path(d->path.substr(1)),
         "absolute no-follow source path required");
    file_bytes =
        e::read_regular_file_no_follow("/", d->path.substr(1), read_limit, end);
    original = {reinterpret_cast<const std::uint8_t *>(file_bytes.data()),
                file_bytes.size()};
    need(file_bytes.capacity() < UINT64_MAX,
         "dataset original buffer size exceeds representation");
    raw_buffer_bytes = file_bytes.capacity() + 1;
  }
  need(e::sha256_hex(original) == d->sha256, "source byte digest mismatch");
  deadline(end);
  db::FileView view;
  need(db::FileView::inspect_dataset(original, decoder_limits, view) ==
           db::Status::ok,
       "Databento DBN input rejected by format or user dataset limits");
  d->metadata = view.metadata();
  need(d->metadata.dataset == d->dataset_name && d->metadata.record_count > 0,
       "DBN dataset/count mismatch");
  d->metadata.dataset = d->dataset_name;
  d->source_bytes = original.size();
  d->metadata_bytes = view.encoded_metadata().size();
  need(d->metadata.record_count <= d->events.max_size(),
       "dataset event storage exceeds host container representation");
  d->load_buffer_bytes =
      buffer_bytes(raw_buffer_bytes, d->metadata.record_count);
  within(d->load_buffer_bytes, d->memory_budget,
         "user dataset load buffer budget exceeded");
  d->events.resize(static_cast<std::size_t>(d->metadata.record_count));
  d->load_buffer_bytes = buffer_bytes(raw_buffer_bytes, d->events.capacity());
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
  // Acknowledge only after digest, full decode, source/user constraints and
  // optional locked residency are admitted. Failure before here destroys the
  // owner handles without acknowledging processing.
  if (retained)
    d->source_delivery = retained->after_admission();
  return d;
}
} // namespace symphony::sbv::detail
