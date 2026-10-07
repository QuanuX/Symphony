#pragma once
#include "dataset.hpp"
#include <algorithm>
#include <optional>
namespace symphony::sbv::detail {
namespace db = sqav::databento;
using U = std::uint64_t;
Json event_json(const db::Mbo &, std::size_t);
bool trade(const db::Mbo &);
U end_at(U, U);
struct RunSettings {
  std::string direction, model;
  U spacing, minimum, before, after, horizon, pn, pd, workers;
  std::optional<U> cap;
  std::vector<std::int64_t> levels;
  std::set<std::string> studies;
};
inline RunSettings run_settings(const Json &p) {
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
             minimum = u64(c.at("min_trade_size"));
  std::optional<U> cap;
  if (!c.at("max_signals").is_null())
    cap = u64(c.at("max_signals"));
  need(minimum > 0 && minimum <= UINT32_MAX && (!cap || *cap >= 1),
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

  return {direction,         model, spacing, minimum, before, after,
          horizon,           pn,    pd,      workers, cap,    std::move(levels),
          std::move(studies)};
}
struct RunFollowup {
  Json clip, execution, markout;
  std::size_t begin{}, finish{};
};
// Both representations call this exact arithmetic. A row observes only the
// already-closed census anchor and its explicitly selected follow-up windows.
inline RunFollowup run_followup(const std::vector<db::Mbo> &events,
                                std::size_t ordinal, U j,
                                const RunSettings &settings, const Json &replay,
                                std::int64_t end) {
  const auto before = settings.before, after = settings.after,
             horizon = settings.horizon, pn = settings.pn, pd = settings.pd;
  const auto &levels = settings.levels;
  const auto &studies = settings.studies;
  const auto &model = settings.model;
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

  const auto &x = events[ordinal];
  const auto start = x.ts_recv < before ? 0 : x.ts_recv - before;
  const auto stop = end_at(x.ts_recv, after),
             horizon_end = end_at(x.ts_recv, horizon);
  const auto begin = static_cast<std::size_t>(lower(start)),
             finish = static_cast<std::size_t>(upper(stop)),
             future_end = static_cast<std::size_t>(upper(horizon_end));
  RunFollowup row;
  row.begin = begin;
  row.finish = finish;
  row.clip = {{"signal_id", "signal-" + dec(j)},
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
          touch ? Json(dec(*touch))
                : missing("no qualifying trade observed in retained horizon")},
         {"touch_latency_ns",
          touch ? Json(dec(events[*touch].ts_recv - x.ts_recv))
                : missing("no qualifying trade observed")},
         {"fill_probability",
          model == "user_probability"
              ? Json{{"numerator", dec(pn)},
                     {"denominator", dec(pd)},
                     {"status", "user_assumption"},
                     {"calibrated", false}}
              : missing("trade touch does not establish fill probability")},
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
  row.execution = {
      {"signal_id", "signal-" + dec(j)},
      {"model", model},
      {"horizon_ns", dec(horizon)},
      {"horizon_within_observed_span", horizon_end <= events.back().ts_recv},
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
  return row;
}
} // namespace symphony::sbv::detail
