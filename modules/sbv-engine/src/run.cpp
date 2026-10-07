#include "census.hpp"
#include "detail.hpp"
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
Json run(const Json &p, std::int64_t end, const Dataset *resident) {
  keys_optional(p,
                {"protocol", "source_path", "source_sha256", "dataset",
                 "output_path", "criteria", "replay", "execution", "studies",
                 "workers", "extensions"},
                {"memory_budget_bytes", "dataset_limits"});
  const auto &c = p.at("criteria");
  keys(c, {"rule", "spacing_ns", "min_trade_size", "direction", "max_signals"});
  auto rule = str(c.at("rule")), direction = str(c.at("direction"));
  need(rule == "spaced_trades" || rule == "trade_direction",
       "unknown signal rule");
  need(direction == "any" || direction == "up" || direction == "down" ||
           direction == "unchanged",
       "unknown signal direction");
  need(rule != "spaced_trades" || direction == "any",
       "spaced_trades requires direction any");
  const auto spacing = u64(c.at("spacing_ns")),
             minimum = u64(c.at("min_trade_size")),
             cap = u64(c.at("max_signals"));
  need(minimum > 0 && minimum <= UINT32_MAX && cap >= 1 && cap <= 4096,
       "signal criteria bounds");
  const auto &replay = p.at("replay");
  keys(replay, {"before_ns", "after_ns", "retain_events"});
  const auto before = u64(replay.at("before_ns")),
             after = u64(replay.at("after_ns"));
  need(replay.at("retain_events").is_boolean(),
       "replay retention boolean required");
  const auto &ex = p.at("execution");
  keys(ex, {"model", "horizon_ns", "price_offsets_nanos",
            "probability_numerator", "probability_denominator"});
  const auto model = str(ex.at("model"));
  need(model == "none" || model == "touch_observation" ||
           model == "user_probability",
       "unsupported execution model");
  const auto horizon = u64(ex.at("horizon_ns")),
             pn = u64(ex.at("probability_numerator")),
             pd = u64(ex.at("probability_denominator"));
  need(pd > 0 && pn <= pd, "probability must be within zero and one");
  need(model == "user_probability" || (pn == 0 && pd == 1),
       "unused probability must be 0/1");
  const auto &offsets = ex.at("price_offsets_nanos");
  need(offsets.is_array() && offsets.size() <= 64, "price profile bound");
  need(model == "none" ? offsets.empty() : !offsets.empty(),
       "explicit execution profile required, or empty for none");
  std::vector<std::int64_t> levels;
  std::set<std::int64_t> distinct;
  for (const auto &o : offsets) {
    auto n = i64(o);
    need(distinct.insert(n).second, "duplicate profile offset");
    levels.push_back(n);
  }
  need(p.at("studies").is_array() && p.at("studies").size() <= 2,
       "study selection bound");
  std::set<std::string> studies;
  for (const auto &s : p.at("studies")) {
    auto name = str(s);
    need((name == "signal_summary" || name == "forward_markout") &&
             studies.insert(name).second,
         "unsupported or duplicate study");
  }
  const auto workers = u64(p.at("workers"));
  need(workers >= 1 && workers <= 64, "workers 1..64");
  need(p.at("extensions").is_object(), "extensions object required");
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
      if (census.size() < cap) {
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
  struct Row {
    Json clip, execution, markout;
    std::size_t begin{}, finish{};
  };
  std::vector<Row> rows(census.size());
  std::atomic<std::size_t> next{0};
  std::atomic<bool> failed{false};
  std::exception_ptr failure;
  std::mutex failure_mutex;
  auto lower = [&](U time) {
    return std::lower_bound(events.begin(), events.end(), time,
                            [](const auto &x, U t) { return x.ts_recv < t; }) -
           events.begin();
  };
  auto upper = [&](U time) {
    return std::upper_bound(events.begin(), events.end(), time,
                            [](U t, const auto &x) { return t < x.ts_recv; }) -
           events.begin();
  };
  auto worker = [&] {
    try {
      while (!failed) {
        const auto j = next.fetch_add(1);
        if (j >= census.size())
          break;
        deadline(end);
        const auto ordinal = census[j];
        const auto &x = events[ordinal];
        const auto start = x.ts_recv < before ? 0 : x.ts_recv - before;
        const auto stop = end_at(x.ts_recv, after),
                   horizon_end = end_at(x.ts_recv, horizon);
        const auto begin = static_cast<std::size_t>(lower(start)),
                   finish = static_cast<std::size_t>(upper(stop)),
                   future_end = static_cast<std::size_t>(upper(horizon_end));
        auto &row = rows[j];
        row.begin = begin;
        row.finish = finish;
        row.clip = {
            {"signal_id", "signal-" + dec(j)},
            {"requested_start_ns", dec(start)},
            {"requested_end_ns_inclusive", dec(stop)},
            {"start_clamped_at_epoch", x.ts_recv < before},
            {"first_ordinal", dec(begin)},
            {"end_ordinal_exclusive", dec(finish)},
            {"causal_end_ordinal_exclusive", dec(ordinal + 1)},
            {"start_within_observed_span", start >= events.front().ts_recv},
            {"end_within_observed_span", stop <= events.back().ts_recv},
            {"completeness", "unproven"},
            {"book_state", "uninitialized"},
            {"payload_layer", "observed"},
            {"event_retention", replay.at("retain_events")},
            {"hydration", "not performed; pre-signal display duration is not "
                          "book hydration"}};
        Json outcomes = Json::array();
        for (auto offset : levels) {
          std::int64_t target{};
          need(!__builtin_add_overflow(x.price, offset, &target),
               "profile price overflow");
          std::optional<std::size_t> touch;
          for (auto k = ordinal + 1; k < future_end; ++k) {
            if (k % 1024 == 0)
              deadline(end);
            if (trade(events[k]) && (offset < 0 ? events[k].price <= target
                                                : events[k].price >= target)) {
              touch = k;
              break;
            }
          }
          outcomes.push_back(
              {{"offset_nanos", dec(offset)},
               {"target_price_nanos", dec(target)},
               {"observed_trade_touch", touch.has_value()},
               {"touch_source_ordinal",
                touch
                    ? Json(dec(*touch))
                    : missing(
                          "no qualifying trade observed in retained horizon")},
               {"touch_latency_ns",
                touch ? Json(dec(events[*touch].ts_recv - x.ts_recv))
                      : missing("no qualifying trade observed")},
               {"fill_probability",
                model == "user_probability"
                    ? Json{{"numerator", dec(pn)},
                           {"denominator", dec(pd)},
                           {"status", "user_assumption"},
                           {"calibrated", false}}
                    : missing(
                          "trade touch does not establish fill probability")},
               {"no_fill_probability",
                model == "user_probability"
                    ? Json{{"numerator", dec(pd - pn)},
                           {"denominator", dec(pd)},
                           {"status", "user_assumption"}}
                    : missing("no execution probability model selected")},
               {"fill_quantity",
                missing("queue, order size and execution policy not modeled")},
               {"fill_time", missing("touch time is not a fill time")},
               {"conditional_fill_price",
                missing("conditional fill distribution not modeled")}});
        }
        row.execution = {{"signal_id", "signal-" + dec(j)},
                         {"model", model},
                         {"horizon_ns", dec(horizon)},
                         {"horizon_within_observed_span",
                          horizon_end <= events.back().ts_recv},
                         {"outcomes", outcomes},
                         {"actual_execution",
                          missing("historical research; no broker execution")}};
        if (studies.contains("forward_markout")) {
          std::optional<std::size_t> last;
          for (auto k = ordinal + 1; k < future_end; ++k) {
            if (k % 1024 == 0)
              deadline(end);
            if (trade(events[k]))
              last = k;
          }
          Json value = missing("no post-signal trade, horizon outside observed "
                               "span, or arithmetic overflow");
          std::int64_t change{};
          if (last && horizon_end <= events.back().ts_recv &&
              !__builtin_sub_overflow(events[*last].price, x.price, &change))
            value = {{"status", "available"},
                     {"value", dec(change)},
                     {"unit", "price_nanos"},
                     {"mark_source_ordinal", dec(*last)},
                     {"meaning", "last observed trade at or before horizon "
                                 "minus anchor; not P&L or fill"}};
          row.markout = {{"signal_id", "signal-" + dec(j)},
                         {"horizon_ns", dec(horizon)},
                         {"price_change", value}};
        }
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
               {"selection_cap", dec(cap)},
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
      {{"source_path", p.at("source_path")},
       {"source_sha256", p.at("source_sha256")},
       {"dataset", p.at("dataset")},
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
