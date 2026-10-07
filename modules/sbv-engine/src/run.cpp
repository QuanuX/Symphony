#include "census.hpp"
#include "detail.hpp"
#include "run_math.hpp"
#include <algorithm>
#include <atomic>
#include <limits>
#include <mutex>
#include <symphony/sqav/databento/dbn.hpp>
#include <thread>
namespace symphony::sbv::detail {
namespace db = sqav::databento;
using U = std::uint64_t;
Json event_json(const db::Mbo &x, std::size_t ordinal) {
  return {{"source_ordinal", dec(ordinal)},
          {"publisher_id", dec(x.publisher_id)},
          {"instrument_id", dec(x.instrument_id)},
          {"ts_event", dec(x.ts_event)},
          {"ts_recv", dec(x.ts_recv)},
          {"order_id", dec(x.order_id)},
          {"price_nanos", dec(x.price)},
          {"size", dec(x.size)},
          {"flags", dec(x.flags)},
          {"channel_id", dec(x.channel_id)},
          {"action", dec(x.action)},
          {"side", dec(x.side)},
          {"ts_in_delta", dec(x.ts_in_delta)},
          {"sequence", dec(x.sequence)},
          {"ts_out", x.ts_out ? Json(dec(*x.ts_out))
                              : missing("DBN record has no ts_out")}};
}
bool trade(const db::Mbo &x) {
  return x.action == 'T' && x.size > 0 && x.price != INT64_MAX;
}
U end_at(U t, U delta) {
  need(delta <= UINT64_MAX - t, "time window overflow");
  return t + delta;
}
Json run_partitioned(const Json &, std::int64_t, const Dataset *);
Json run(const Json &p, std::int64_t end, const Dataset *resident) {
  if (p.contains("output"))
    return run_partitioned(p, end, resident);
  keys_optional(p,
                {"protocol", "output_path", "criteria", "replay", "execution",
                 "studies", "workers", "extensions"},
                {"memory_budget_bytes", "dataset_limits", "source_path",
                 "source_sha256", "dataset", "retained_source"});
  const auto settings = run_settings(p);
  const auto &direction = settings.direction, &model = settings.model;
  const auto spacing = settings.spacing, minimum = settings.minimum,
             workers = settings.workers;
  const auto &cap = settings.cap;
  const auto &studies = settings.studies;
  const auto &replay = p.at("replay");
  const auto owned = resident ? nullptr : load_dataset(p, end);
  const auto &source = resident ? *resident : *owned;
  source.bind(p);
  const auto &events = source.events;

  Json signals = Json::array();
  std::vector<std::size_t> census;
  std::optional<std::int64_t> previous;
  U prior_signal = 0, eligible = 0;
  // Causal pass: the predicate receives only the current record and previous
  // observed trade. Future-window machinery starts after this census is sealed.
  for (std::size_t i = 0; i < events.size(); ++i) {
    if (i % 1024 == 0)
      deadline(end);
    const auto &x = events[i];
    if (!trade(x))
      continue;
    const bool match =
        direction == "any" ||
        (previous && (direction == "up"     ? x.price > *previous
                      : direction == "down" ? x.price < *previous
                                            : x.price == *previous));
    if (x.size >= minimum && match &&
        (census.empty() || x.ts_recv - prior_signal >= spacing)) {
      ++eligible;
      if (!cap || census.size() < *cap) {
        signals.push_back({{"signal_id", "signal-" + dec(census.size())},
                           {"source_ordinal", dec(i)},
                           {"available_ns", dec(x.ts_recv)},
                           {"anchor_price_nanos", dec(x.price)},
                           {"causal_end_ordinal_exclusive", dec(i + 1)},
                           {"previous_trade_price_nanos",
                            previous ? Json(dec(*previous))
                                     : missing("no preceding trade")}});
        census.push_back(i);
        prior_signal = x.ts_recv;
      }
    }
    previous = x.price;
  }
  const auto census_digest = e::sha256_hex(signals.dump());
  std::vector<RunFollowup> rows(census.size());
  std::atomic<std::size_t> next{0};
  std::atomic<bool> failed{false};
  std::exception_ptr failure;
  std::mutex failure_mutex;
  auto worker = [&] {
    try {
      while (!failed) {
        const auto j = next.fetch_add(1);
        if (j >= census.size())
          break;
        deadline(end);
        const auto ordinal = census[j];
        rows[j] = run_followup(events, ordinal, j, settings, replay, end);
      }
    } catch (...) {
      failed = true;
      std::lock_guard lock(failure_mutex);
      if (!failure)
        failure = std::current_exception();
    }
  };
  {
    std::vector<std::jthread> threads;
    for (std::size_t i = 0;
         i < std::min<std::size_t>(workers,
                                   std::max<std::size_t>(1, census.size()));
         ++i)
      threads.emplace_back(worker);
  }
  if (failure)
    std::rethrow_exception(failure);
  deadline(end);
  auto result = base("native historical Databento MBO closed-census research");
  auto &s = result["sections"];
  result["status"] = "partial";
  s["summary"] =
      section({{"signal_count", dec(census.size())},
               {"record_count", dec(events.size())},
               {"closed_census", true},
               {"census_sha256", census_digest},
               {"selection_cap", p.at("criteria").at("max_signals")},
               {"additional_eligible_after_cap", dec(eligible - census.size())},
               {"performance_return",
                missing("no portfolio/accounting model executed")}});
  s["signals"] = section(signals);
  Json clips = Json::array(), execution = Json::array(),
       markouts = Json::array();
  std::vector<bool> retained(events.size(), false);
  for (const auto &row : rows) {
    clips.push_back(row.clip);
    execution.push_back(row.execution);
    if (studies.contains("forward_markout"))
      markouts.push_back(row.markout);
    if (replay.at("retain_events").get<bool>())
      std::fill(retained.begin() + row.begin, retained.begin() + row.finish,
                true);
  }
  Json payload = Json::array();
  for (std::size_t i = 0; i < events.size(); ++i) {
    if (i % 1024 == 0)
      deadline(end);
    if (retained[i])
      payload.push_back(event_json(events[i], i));
  }
  s["execution"] = section(
      execution, model == "none" ? "not_selected" : "partial",
      model == "none" ? "user selected no execution model"
                      : "touch observations and explicit user assumptions "
                        "only; no calibrated fill inference");
  s["replay"] = section({{"windows", clips},
                         {"events", payload},
                         {"event_key", "source_ordinal"},
                         {"retained_event_count", dec(payload.size())},
                         {"retention_policy", replay},
                         {"price_scale", "1e-9"},
                         {"clock", "ts_recv, source_ordinal tie-break"}},
                        "partial",
                        "capped or uninitialized input cannot establish "
                        "complete exchange history");
  Json study_results = Json::array();
  if (studies.contains("signal_summary"))
    study_results.push_back(
        {{"study_id", "signal_summary"},
         {"version", "1"},
         {"count", dec(census.size())},
         {"first_available_ns",
          census.empty() ? missing("empty census")
                         : Json(dec(events[census.front()].ts_recv))},
         {"last_available_ns",
          census.empty() ? missing("empty census")
                         : Json(dec(events[census.back()].ts_recv))}});
  if (studies.contains("forward_markout"))
    study_results.push_back({{"study_id", "forward_markout"},
                             {"version", "1"},
                             {"results", markouts}});
  s["studies"] =
      section(study_results, studies.empty() ? "not_selected" : "available",
              studies.empty() ? "user selected zero studies" : "");
  s["resources"] =
      section({{"requested_workers", dec(workers)},
               {"actual_workers",
                dec(std::min<std::size_t>(
                    workers, std::max<std::size_t>(1, census.size())))},
               {"backend", "cpu"},
               {"cuda", "unavailable"},
               {"tensor", "unavailable"}});
  s["diagnostics"] = section(
      Json::array({"No order book hydration or broker fills. Trade touches are "
                   "observations only.",
                   "Source completeness remains unproven; window span flags "
                   "are not completeness assertions.",
                   "Fixed census closes before all execution/replay/study "
                   "follow-up. Signal caps and settings are explicit."}));
  auto choices = p;
  choices.erase("output_path");
  s["choices"] = section(choices);
  const auto &meta = source.metadata;
  s["provenance"] = section(
      {{"source_path", source.path},
       {"source_sha256", source.sha256},
       {"dataset", source.dataset_name},
       {"adapter", db::adapter_id},
       {"adapter_version", db::adapter_version},
       {"dbn_version", dec(meta.version)},
       {"instrument_id", dec(events.front().instrument_id)},
       {"requested_start_ns", dec(meta.start)},
       {"requested_end_ns_exclusive", dec(meta.end)},
       {"observed_first_ns", dec(events.front().ts_recv)},
       {"observed_last_ns", dec(events.back().ts_recv)},
       {"source_record_limit", dec(meta.limit)},
       {"source_cap_reached", meta.limit != 0 && events.size() >= meta.limit},
       {"engine_version", version},
       {"provider_requests", "0"},
       {"additional_spend_usd", "0"},
       {"lineage",
        "SQAV decoded immutable source; SBV semantic derived artifact; "
        "SQPV/SQDV are optional separate storage/delivery contracts"}});
  s["user_extensions"] = section(p.at("extensions"));
  result["sections"]["resources"]["data"]["dataset_feed"] =
      source.evidence(resident != nullptr);
  s["census"] = section(census_evidence(result));
  return persist(std::move(result), p, "run", end);
}
} // namespace symphony::sbv::detail
