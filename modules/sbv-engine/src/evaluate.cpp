#include "detail.hpp"
#include <algorithm>
#include <atomic>
#include <mutex>
#include <symphony/sbv/models.hpp>
#include <symphony/sqav/databento/dbn.hpp>
#include <thread>
namespace symphony::sbv::detail {
namespace db = sqav::databento;
Json event_json(const db::Mbo &, std::size_t);
std::uint64_t end_at(std::uint64_t, std::uint64_t);
Json evaluate(const Json &p, std::int64_t end) {
  keys(p, {"protocol", "source_path", "source_sha256", "dataset", "output_path",
           "census", "model", "replay", "studies", "workers", "extensions"});
  const auto workers = u64(p.at("workers"));
  need(workers >= 1 && workers <= 64, "workers 1..64");
  need(p.at("extensions").is_object(), "extensions object required");
  const auto &replay = p.at("replay");
  keys(replay, {"before_ns", "after_ns", "retain_events"});
  const auto before = u64(replay.at("before_ns")),
             after = u64(replay.at("after_ns"));
  need(replay.at("retain_events").is_boolean(), "retention boolean required");
  need(p.at("studies").is_array() && p.at("studies").size() <= 3,
       "study selection bound");
  std::set<std::string> studies;
  for (const auto &v : p.at("studies")) {
    const auto id = str(v);
    need((id == "signal_summary" || id == "model_summary" ||
          id == "path_excursion") &&
             studies.insert(id).second,
         "unknown or duplicate evaluation study");
  }
  const auto bytes = read_file(str(p.at("source_path")), end);
  need(bytes.size() <= (64U << 20) &&
           e::sha256_hex(bytes) == str(p.at("source_sha256")),
       "source bytes/digest mismatch");
  db::FileView view;
  const auto span = std::span<const unsigned char>(
      reinterpret_cast<const unsigned char *>(bytes.data()), bytes.size());
  need(db::FileView::inspect(span, {64U << 20, 1U << 20, 200000}, view) ==
               db::Status::ok &&
           view.metadata().dataset == str(p.at("dataset")) &&
           view.metadata().record_count > 0,
       "DBN source contract mismatch");
  std::vector<db::Mbo> events(view.metadata().record_count);
  std::vector<ObservedTrade> trades;
  for (std::size_t i = 0; i < events.size(); ++i) {
    if (i % 1024 == 0)
      deadline(end);
    auto &x = events[i];
    need(view.record(i, x) == db::Status::ok && x.ts_recv != UINT64_MAX &&
             (i == 0 || (x.ts_recv >= events[i - 1].ts_recv &&
                         x.instrument_id == events[0].instrument_id)),
         "one instrument with ordered known receive timestamps required");
    if (x.action == 'T' && x.size > 0 && x.price != INT64_MAX)
      trades.push_back({i, x.ts_recv, x.price, x.size});
  }
  const auto &c = p.at("census");
  keys(c, {"protocol", "source_sha256", "mode", "producer", "signals"});
  need(c.at("protocol") == "symphony.sbv.external-census.v1" &&
           c.at("source_sha256") == p.at("source_sha256"),
       "census source/protocol mismatch");
  const auto mode = str(c.at("mode"));
  need(mode == "causal_declared" || mode == "retrospective",
       "external causality declaration required");
  const auto &producer = c.at("producer");
  keys(producer, {"id", "version", "artifact_sha256", "reproducibility"});
  for (const auto *key : {"id", "version"})
    need(!str(producer.at(key)).empty() && str(producer.at(key)).size() <= 256,
         "bounded producer identity required");
  const auto ph = str(producer.at("artifact_sha256"));
  need(ph.empty() ||
           (ph.size() == 64 && std::all_of(ph.begin(), ph.end(),
                                           [](char x) {
                                             return (x >= '0' && x <= '9') ||
                                                    (x >= 'a' && x <= 'f');
                                           })),
       "producer digest format");
  const auto reproducibility = str(producer.at("reproducibility"));
  need(reproducibility == "deterministic_declared" ||
           reproducibility == "nondeterministic" ||
           reproducibility == "uncaptured",
       "reproducibility declaration required");
  const auto &signals = c.at("signals");
  need(signals.is_array() && signals.size() <= 4096, "external census bound");
  std::vector<std::string> ids;
  std::set<std::string> unique;
  std::uint64_t prior_ordinal = 0;
  for (const auto &s : signals) {
    keys(s,
         {"signal_id", "source_ordinal", "available_ns", "anchor_price_nanos",
          "causal_end_ordinal_exclusive", "context_reference"});
    const auto id = str(s.at("signal_id"));
    const auto ordinal = u64(s.at("source_ordinal")),
               available = u64(s.at("available_ns")),
               causal_end = u64(s.at("causal_end_ordinal_exclusive"));
    need(!id.empty() && id.size() <= 256 && unique.insert(id).second &&
             ordinal < events.size() && ordinal >= prior_ordinal &&
             available == events[ordinal].ts_recv,
         "census identity/order/availability mismatch");
    need(causal_end <= events.size() &&
             (mode != "causal_declared" || causal_end <= ordinal + 1),
         "census causal prefix exceeds declared boundary");
    need(i64(s.at("anchor_price_nanos")) != INT64_MAX,
         "undefined price cannot be an anchor");
    (void)str(s.at("context_reference"));
    ids.push_back(id);
    prior_ordinal = ordinal;
  }
  // Seal identity before model admission/follow-up; native code can validate
  // references, but cannot prove how an external producer obtained a signal.
  const auto census_digest = e::sha256_hex(c.dump());
  const AdmittedModel model(p.at("model"), ids);
  struct Row {
    Json outcome, window, excursion;
    std::size_t begin{}, finish{};
  };
  std::vector<Row> rows(signals.size());
  std::atomic<std::size_t> next{0};
  std::atomic<bool> failed{false};
  std::exception_ptr failure;
  std::mutex mutex;
  const auto worker = [&] {
    try {
      while (!failed) {
        const auto j = next.fetch_add(1);
        if (j >= signals.size())
          break;
        deadline(end);
        const auto &s = signals[j];
        const auto ordinal = u64(s.at("source_ordinal")),
                   available = u64(s.at("available_ns"));
        const auto anchor = i64(s.at("anchor_price_nanos"));
        const auto horizon_end = end_at(available, model.horizon_ns());
        auto begin_trade = std::upper_bound(
            trades.begin(), trades.end(), ordinal,
            [](auto v, const auto &t) { return v < t.source_ordinal; });
        auto end_trade = std::upper_bound(
            begin_trade, trades.end(), horizon_end,
            [](auto v, const auto &t) { return v < t.available_ns; });
        const auto &event = events[ordinal];
        ModelFrame frame{
            ids[j],
            ordinal,
            available,
            horizon_end,
            anchor,
            event.action == 'T' && event.size > 0 && event.price == anchor,
            horizon_end <= events.back().ts_recv,
            std::span<const ObservedTrade>(begin_trade, end_trade)};
        auto &row = rows[j];
        row.outcome = model.evaluate(frame, end);
        const auto start = available < before ? 0 : available - before,
                   stop = end_at(available, after);
        const auto first = std::lower_bound(
            events.begin(), events.end(), start,
            [](const auto &x, auto t) { return x.ts_recv < t; });
        const auto last = std::upper_bound(
            first, events.end(), stop,
            [](auto t, const auto &x) { return t < x.ts_recv; });
        row.begin = static_cast<std::size_t>(first - events.begin());
        row.finish = static_cast<std::size_t>(last - events.begin());
        row.window = {
            {"signal_id", ids[j]},
            {"requested_start_ns", dec(start)},
            {"requested_end_ns_inclusive", dec(stop)},
            {"start_clamped_at_epoch", available < before},
            {"first_ordinal", dec(row.begin)},
            {"end_ordinal_exclusive", dec(row.finish)},
            {"causal_end_ordinal_exclusive",
             s.at("causal_end_ordinal_exclusive")},
            {"start_within_observed_span", start >= events.front().ts_recv},
            {"end_within_observed_span", stop <= events.back().ts_recv},
            {"completeness", "unproven"},
            {"book_state", "uninitialized"},
            {"payload_layer", "observed"}};
        if (studies.contains("path_excursion")) {
          row.excursion = {{"signal_id", ids[j]},
                           {"horizon_end_ns", dec(horizon_end)},
                           {"unit", "price_nanos"},
                           {"coverage", frame.horizon_within_observed_span
                                            ? "observed_span_only"
                                            : "truncated_observed_span"},
                           {"minimum", missing("no observed future trade")},
                           {"maximum", missing("no observed future trade")}};
          if (begin_trade != end_trade) {
            const auto minimum = std::min_element(
                begin_trade, end_trade, [](const auto &a, const auto &b) {
                  return a.price_nanos < b.price_nanos;
                });
            const auto maximum = std::max_element(
                begin_trade, end_trade, [](const auto &a, const auto &b) {
                  return a.price_nanos < b.price_nanos;
                });
            auto extremum = [&](const ObservedTrade &t) -> Json {
              std::int64_t change{};
              if (__builtin_sub_overflow(t.price_nanos, anchor, &change))
                return missing("exact price difference overflows int64");
              return {{"status", "available"},
                      {"value", dec(change)},
                      {"source_ordinal", dec(t.source_ordinal)},
                      {"available_ns", dec(t.available_ns)}};
            };
            row.excursion["minimum"] = extremum(*minimum);
            row.excursion["maximum"] = extremum(*maximum);
          }
        }
      }
    } catch (...) {
      failed = true;
      std::lock_guard lock(mutex);
      if (!failure)
        failure = std::current_exception();
    }
  };
  const auto applied =
      signals.empty() ? 0 : std::min<std::size_t>(workers, signals.size());
  {
    std::vector<std::jthread> threads;
    for (std::size_t i = 0; i < applied; ++i)
      threads.emplace_back(worker);
  }
  if (failure)
    std::rethrow_exception(failure);
  auto result =
      base("native follow-up of an externally supplied closed census");
  result["status"] = "partial";
  auto &s = result["sections"];
  s["signals"] = section(signals);
  s["summary"] = section({{"signal_count", dec(signals.size())},
                          {"record_count", dec(events.size())},
                          {"census_sha256", census_digest},
                          {"closed_census", true},
                          {"causality", mode},
                          {"external_causality_verified", false}});
  Json outcomes = Json::array(), windows = Json::array(),
       excursions = Json::array(), retained_events = Json::array();
  std::vector<bool> retained(events.size(), false);
  std::size_t available_count = 0;
  for (const auto &row : rows) {
    outcomes.push_back(row.outcome);
    windows.push_back(row.window);
    if (row.outcome.at("status") == "available")
      ++available_count;
    if (studies.contains("path_excursion"))
      excursions.push_back(row.excursion);
    if (replay.at("retain_events").get<bool>())
      std::fill(retained.begin() + row.begin, retained.begin() + row.finish,
                true);
  }
  for (std::size_t i = 0; i < events.size(); ++i) {
    if (i % 1024 == 0)
      deadline(end);
    if (retained[i])
      retained_events.push_back(event_json(events[i], i));
  }
  s["execution"] = section(outcomes, "partial",
                           "conditional model evidence or external claims; "
                           "actual fills and calibration unverified");
  s["replay"] = section(
      {{"windows", windows},
       {"events", retained_events},
       {"retained_event_count", dec(retained_events.size())},
       {"event_key", "source_ordinal"},
       {"retention_policy", replay},
       {"price_scale", "1e-9"},
       {"clock", "ts_recv with source ordinal tie-break"}},
      "partial",
      "observed fragments; no initialized book or exchange completeness proof");
  Json selected = Json::array();
  if (studies.contains("signal_summary"))
    selected.push_back({{"study_id", "signal_summary"},
                        {"version", "1"},
                        {"count", dec(signals.size())}});
  if (studies.contains("model_summary"))
    selected.push_back(
        {{"study_id", "model_summary"},
         {"version", "1"},
         {"available", dec(available_count)},
         {"unavailable", dec(signals.size() - available_count)}});
  if (studies.contains("path_excursion"))
    selected.push_back({{"study_id", "path_excursion"},
                        {"version", "1"},
                        {"results", excursions}});
  s["studies"] =
      section(selected, studies.empty() ? "not_selected" : "available",
              studies.empty() ? "zero studies selected" : "");
  s["resources"] = section({{"requested_workers", dec(workers)},
                            {"actual_workers", dec(applied)},
                            {"backend", "cpu"}});
  s["diagnostics"] = section(
      Json::array({"External causal/reproducibility declarations are "
                   "preserved, not independently proven.",
                   "Observed trade levels are retrospective support, not book "
                   "liquidity or unconditional fill likelihood.",
                   "No normalization or probability interpretation is imposed "
                   "on a declared non-probability measure."}));
  auto choices = p;
  choices.erase("output_path");
  s["choices"] = section(choices);
  const auto &meta = view.metadata();
  s["provenance"] = section(
      {{"engine_version", version},
       {"source_sha256", p.at("source_sha256")},
       {"source_path", p.at("source_path")},
       {"dataset", p.at("dataset")},
       {"adapter", db::adapter_id},
       {"adapter_version", db::adapter_version},
       {"dbn_version", dec(meta.version)},
       {"instrument_id", dec(events.front().instrument_id)},
       {"source_record_limit", dec(meta.limit)},
       {"source_cap_reached", meta.limit != 0 && events.size() >= meta.limit},
       {"requested_start_ns", dec(meta.start)},
       {"requested_end_ns_exclusive", dec(meta.end)},
       {"observed_first_ns", dec(events.front().ts_recv)},
       {"observed_last_ns", dec(events.back().ts_recv)},
       {"census_producer", producer},
       {"census_sha256", census_digest},
       {"external_dependency_capture",
        "declared only; SBV cannot audit uncaptured producer inputs"},
       {"provider_requests", "0"},
       {"additional_spend_usd", "0"}});
  s["user_extensions"] = section(p.at("extensions"));
  return persist(std::move(result), p, "evaluate", end);
}
} // namespace symphony::sbv::detail
