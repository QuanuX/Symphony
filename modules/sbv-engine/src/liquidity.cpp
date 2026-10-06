#include "rational.hpp"
#include <algorithm>
#include <atomic>
#include <map>
#include <mutex>
#include <optional>
#include <thread>
namespace symphony::sbv::detail {
namespace {
namespace r = rational;
using Wide = __int128_t;
std::string wide(Wide n) {
  const bool negative = n < 0;
  if (negative)
    n = -n;
  std::string s;
  do {
    s.push_back(static_cast<char>('0' + n % 10));
    n /= 10;
  } while (n);
  if (negative)
    s.push_back('-');
  std::reverse(s.begin(), s.end());
  return s;
}
Json ratio(Wide n, std::uint64_t d) {
  need(d > 0, "ratio denominator zero");
  const auto remainder = static_cast<std::uint64_t>((n < 0 ? -n : n) % d);
  const auto g = std::gcd(remainder, d);
  return {{"numerator", wide(n / g)}, {"denominator", dec(d / g)}};
}
struct Level {
  std::int64_t price;
  std::uint64_t original, remaining;
};
struct Snapshot {
  std::uint64_t ordinal{}, time{};
  bool available{}, flagged{};
  std::string reason, market;
  std::vector<Level> bids, asks;
  std::uint64_t bid_count{}, ask_count{};
};
Snapshot frame(const Json &f) {
  need(f.at("protocol") == "symphony.sbv.book-frame.v1" &&
           f.at("layer") == "reconstructed",
       "typed reconstructed frame required");
  Snapshot x;
  x.ordinal = u64(f.at("source_ordinal"));
  x.time = u64(f.at("available_ns"));
  need(x.time != UINT64_MAX, "known frame receive timestamp required");
  need(f.at("event_boundary").is_boolean() &&
           f.at("receive_time_flagged").is_boolean(),
       "frame flags required");
  x.flagged = f.at("receive_time_flagged").get<bool>();
  need(f.at("status") == "available" || f.at("status") == "unavailable",
       "frame status invalid");
  x.available = f.at("status") == "available";
  if (!x.available) {
    need(f.at("data").is_null(), "unavailable frame contains invented depth");
    x.reason = str(f.at("reason"));
    need(!x.reason.empty(), "unavailable frame needs reason");
    return x;
  }
  need(f.at("event_boundary") == true,
       "complete frame event boundary required");
  const auto &data = f.at("data");
  auto levels = [&](const char *key, bool bids) {
    const auto &rows = data.at(key);
    need(rows.is_array() && rows.size() <= 64, "frame depth bound");
    std::vector<Level> out;
    for (const auto &row : rows) {
      const auto p = i64(row.at("price_nanos"));
      const auto q = u64(row.at("size"));
      need(p != INT64_MAX && q > 0 && u64(row.at("order_count")) > 0 &&
               (out.empty() ||
                (bids ? p < out.back().price : p > out.back().price)),
           "invalid or unordered frame level");
      out.push_back({p, q, q});
    }
    return out;
  };
  x.bids = levels("bids", true);
  x.asks = levels("asks", false);
  x.bid_count = u64(data.at("bid_level_count"));
  x.ask_count = u64(data.at("ask_level_count"));
  need(x.bid_count >= x.bids.size() && x.ask_count >= x.asks.size() &&
           (x.bid_count == 0 || !x.bids.empty()) &&
           (x.ask_count == 0 || !x.asks.empty()),
       "truncated depth counts inconsistent");
  x.market = x.bids.empty() || x.asks.empty() ? "one_or_both_sides_empty"
             : x.bids.front().price < x.asks.front().price  ? "uncrossed"
             : x.bids.front().price == x.asks.front().price ? "locked"
                                                            : "crossed";
  need(data.at("market_state") == x.market,
       "market state classification mismatch");
  return x;
}
struct Order {
  Json choice;
  std::string id, signal;
  std::uint64_t frame_ordinal, quantity, signal_ordinal, signal_time;
  std::int64_t anchor;
  bool buy, fok;
  std::optional<std::int64_t> limit;
  r::R participation;
  std::optional<r::R> activation;
};
Order order(const Json &o, const std::map<std::string, Json> &signals) {
  keys(o, {"order_id", "signal_id", "frame_source_ordinal", "side", "quantity",
           "limit_price_nanos", "time_in_force", "max_level_participation",
           "activation_probability"});
  const auto id = str(o.at("order_id")), signal = str(o.at("signal_id"));
  need(!id.empty() && id.size() <= 256 && signals.contains(signal),
       "bounded order identity and known signal required");
  const auto &s = signals.at(signal);
  const auto q = u64(o.at("quantity")),
             ordinal = u64(o.at("frame_source_ordinal"));
  need(q >= 1 && q <= 1000000000 && ordinal >= u64(s.at("source_ordinal")),
       "quantity bound or pre-signal frame");
  need(o.at("side") == "buy" || o.at("side") == "sell", "order side invalid");
  need(o.at("time_in_force") == "IOC" || o.at("time_in_force") == "FOK",
       "time in force unavailable");
  auto participation = r::read_ratio(o.at("max_level_participation"));
  need(participation.n >= 0 && participation.n <= participation.d,
       "participation outside [0,1]");
  std::optional<std::int64_t> limit;
  if (!o.at("limit_price_nanos").is_null()) {
    limit = i64(o.at("limit_price_nanos"));
    need(*limit != INT64_MAX, "undefined limit price");
  }
  std::optional<r::R> activation;
  const auto &p = o.at("activation_probability");
  need(p.is_object() && p.contains("status"),
       "activation declaration required");
  if (p.at("status") == "supplied") {
    keys(p, {"status", "value"});
    activation = r::read_ratio(p.at("value"));
    need(activation->n >= 0 && activation->n <= activation->d,
         "activation outside [0,1]");
  } else {
    keys(p, {"status", "reason"});
    need(p.at("status") == "unavailable" && !str(p.at("reason")).empty(),
         "activation status/reason required");
  }
  const auto anchor = i64(s.at("anchor_price_nanos"));
  need(anchor != INT64_MAX, "undefined anchor");
  return {o,
          id,
          signal,
          ordinal,
          q,
          u64(s.at("source_ordinal")),
          u64(s.at("available_ns")),
          anchor,
          o.at("side") == "buy",
          o.at("time_in_force") == "FOK",
          limit,
          participation,
          activation};
}
Json simulate(const Order &o, Snapshot *snapshot, const Json &p,
              const std::string &dependency, std::int64_t end) {
  deadline(end);
  Json result{
      {"protocol", "symphony.sbv.liquidity-outcome.v1"},
      {"model_id", "displayed_depth_sweep"},
      {"model_version", "1"},
      {"order_id", o.id},
      {"signal_id", o.signal},
      {"frame_source_ordinal", dec(o.frame_ordinal)},
      {"status", "available"},
      {"reason", ""},
      {"order", o.choice},
      {"calibration", "not_verified"},
      {"conditioning", p.at("liquidity_mode") == "shared_snapshot"
                           ? "all_orders_active_in_input_order"
                           : "selected_order_active"},
      {"data", nullptr},
      {"any_fill_probability", missing("activation probability unavailable")},
      {"complete_fill_probability",
       missing("activation probability unavailable")},
      {"no_fill_probability", missing("activation probability unavailable")},
      {"quantity_distribution", missing("activation probability unavailable")},
      {"actual_execution",
       missing("historical snapshot scenario; no actual order execution")}};
  auto reason = dependency;
  if (reason.empty()) {
    if (!snapshot)
      reason = "selected frame is not retained";
    else if (!snapshot->available)
      reason = "selected book frame unavailable: " + snapshot->reason;
    else if ((snapshot->market == "locked" || snapshot->market == "crossed") &&
             p.at("market_state_policy") == "uncrossed_only")
      reason = "locked/crossed frame excluded by selected policy";
    else if (snapshot->flagged &&
             p.at("receive_time_policy") == "unflagged_only")
      reason = "selected frame receive timestamp flagged";
  }
  if (snapshot)
    need(snapshot->time >= o.signal_time,
         "selected frame precedes signal availability");
  auto unavailable = [&] {
    need(p.at("on_unavailable") != "reject", reason.c_str());
    result["status"] = "unavailable";
    result["reason"] = reason;
    for (const auto *key : {"any_fill_probability", "complete_fill_probability",
                            "no_fill_probability", "quantity_distribution"})
      result[key] = missing(reason);
    return result;
  };
  if (!reason.empty())
    return unavailable();
  auto &levels = o.buy ? snapshot->asks : snapshot->bids;
  const auto total_levels = o.buy ? snapshot->ask_count : snapshot->bid_count;
  std::uint64_t remaining = o.quantity;
  Json fills = Json::array();
  bool limit_boundary = false;
  std::vector<std::pair<std::size_t, std::uint64_t>> consumption;
  Wide notional = 0;
  for (std::size_t i = 0; i < levels.size() && remaining; ++i) {
    const auto &l = levels[i];
    if (o.limit && (o.buy ? l.price > *o.limit : l.price < *o.limit)) {
      limit_boundary = true;
      break;
    }
    const auto cap = static_cast<std::uint64_t>(
        static_cast<__uint128_t>(l.original) *
        static_cast<std::uint64_t>(o.participation.n) /
        static_cast<std::uint64_t>(o.participation.d));
    const auto take = std::min({remaining, l.remaining, cap});
    if (!take)
      continue;
    fills.push_back({{"price_nanos", dec(l.price)},
                     {"quantity", dec(take)},
                     {"displayed_quantity", dec(l.original)},
                     {"remaining_before", dec(l.remaining)},
                     {"participation_cap", dec(cap)},
                     {"remaining_after", dec(l.remaining - take)}});
    consumption.emplace_back(i, take);
    notional += static_cast<Wide>(l.price) * take;
    remaining -= take;
  }
  // At equality, all omitted levels are strictly worse and cannot match a
  // limit. A zero participation cap also makes omitted levels irrelevant.
  if (o.limit && !levels.empty() && o.limit.value() == levels.back().price)
    limit_boundary = true;
  const bool unknown_tail = remaining > 0 && total_levels > levels.size() &&
                            !limit_boundary && o.participation.n != 0;
  if (unknown_tail && p.at("depth_policy") == "require_sufficient") {
    reason = "retained depth cannot resolve the remaining order quantity";
    return unavailable();
  }
  const bool canceled = o.fok && remaining != 0;
  if (canceled) {
    remaining = o.quantity;
    fills = Json::array();
    consumption.clear();
    notional = 0;
  }
  for (const auto &[i, q] : consumption)
    levels[i].remaining -= q;
  const auto filled = o.quantity - remaining;
  result["data"] = {
      {"requested_quantity", dec(o.quantity)},
      {"filled_quantity", dec(filled)},
      {"unfilled_quantity", dec(remaining)},
      {"disposition", canceled         ? "fok_canceled"
                      : filled == 0    ? "no_fill"
                      : remaining == 0 ? "full_fill"
                                       : "partial_fill"},
      {"fills", fills},
      {"notional_price_nanos_times_quantity", wide(notional)},
      {"vwap_price_nanos", filled ? Json{{"status", "available"},
                                         {"value", ratio(notional, filled)}}
                                  : missing("no scenario fill")},
      {"coverage",
       unknown_tail ? "visible_depth_only" : "sufficient_for_selected_order"},
      {"frame_available_ns", dec(snapshot->time)},
      {"frame_delay_from_signal_ns", dec(snapshot->time - o.signal_time)},
      {"receive_time_flagged", snapshot->flagged},
      {"market_state", snapshot->market}};
  if (o.activation) {
    const auto probability = filled ? *o.activation : r::R{};
    const auto complete = filled == o.quantity ? *o.activation : r::R{};
    auto claim = [](r::R v) -> Json {
      return {{"status", "scenario_assumption"}, {"value", r::wire(v)}};
    };
    result["any_fill_probability"] = claim(probability);
    result["complete_fill_probability"] = claim(complete);
    result["no_fill_probability"] =
        claim(r::plus({1, 1}, {-probability.n, probability.d}));
    Json atoms = Json::array();
    if (filled) {
      atoms.push_back({{"filled_quantity", dec(filled)},
                       {"weight", r::wire(*o.activation)}});
      atoms.push_back(
          {{"filled_quantity", "0"},
           {"weight",
            r::wire(r::plus({1, 1}, {-o.activation->n, o.activation->d}))}});
    } else
      atoms.push_back({{"filled_quantity", "0"}, {"weight", r::wire({1, 1})}});
    result["quantity_distribution"] = {{"status", "scenario_assumption"},
                                       {"value", atoms}};
  }
  return result;
}
} // namespace
Json liquidity(const Json &p, std::int64_t end) {
  keys(p, {"protocol", "path", "expected_sha256", "output_path", "orders",
           "liquidity_mode", "depth_policy", "market_state_policy",
           "receive_time_policy", "on_unavailable", "studies", "workers",
           "extensions"});
  const auto workers = u64(p.at("workers"));
  need(workers >= 1 && workers <= 64 && p.at("extensions").is_object(),
       "worker/extension bound");
  const auto mode = str(p.at("liquidity_mode"));
  need(mode == "independent" || mode == "shared_snapshot",
       "liquidity mode invalid");
  need(p.at("depth_policy") == "require_sufficient" ||
           p.at("depth_policy") == "visible_only",
       "depth policy invalid");
  need(p.at("market_state_policy") == "uncrossed_only" ||
           p.at("market_state_policy") == "allow_locked_crossed",
       "market policy invalid");
  need(p.at("receive_time_policy") == "unflagged_only" ||
           p.at("receive_time_policy") == "allow_flagged",
       "receive-time policy invalid");
  need(p.at("on_unavailable") == "unavailable" ||
           p.at("on_unavailable") == "reject",
       "unavailable policy invalid");
  need(p.at("studies").is_array() && p.at("studies").size() <= 2,
       "study bound");
  std::set<std::string> studies;
  for (const auto &v : p.at("studies")) {
    const auto id = str(v);
    need((id == "liquidity_summary" || id == "fill_quality") &&
             studies.insert(id).second,
         "unknown/duplicate liquidity study");
  }
  auto source = e::parse_bounded_json(read_file(str(p.at("path")), end),
                                      artifact_bytes, artifact_values);
  validate_result(source);
  need(!str(p.at("expected_sha256")).empty() &&
           source.at("content_sha256") == p.at("expected_sha256"),
       "source result identity mismatch");
  const auto &s = source.at("sections");
  need(s.at("provenance").at("data").at("profile") ==
               "databento_mbo_orders_strict_v1" &&
           s.at("summary").at("data").at("closed_census") == true,
       "admitted source-bound book result required");
  std::map<std::string, Json> signals;
  const auto &signal_rows = s.at("signals").at("data");
  need(signal_rows.is_array() && signal_rows.size() <= 4096,
       "source census bound");
  for (const auto &row : signal_rows)
    need(signals.emplace(str(row.at("signal_id")), row).second,
         "duplicate source signal");
  std::map<std::uint64_t, Snapshot> snapshots;
  const auto &frames = s.at("book_frames").at("data");
  need(frames.is_array() && frames.size() <= 4096, "source frame bound");
  for (const auto &f : frames) {
    auto x = frame(f);
    need(snapshots.emplace(x.ordinal, std::move(x)).second,
         "duplicate source frame");
  }
  need(p.at("orders").is_array() && p.at("orders").size() <= 4096,
       "order bound");
  std::vector<Order> orders;
  std::set<std::string> ids;
  for (const auto &o : p.at("orders")) {
    auto x = order(o, signals);
    need(ids.insert(x.id).second, "duplicate order ID");
    if (mode == "shared_snapshot")
      need(
          (orders.empty() || x.frame_ordinal == orders.front().frame_ordinal) &&
              (!x.activation || x.activation->n == x.activation->d),
          "shared snapshot requires one frame and all-active conditioning; "
          "stochastic branch enumeration not implemented");
    orders.push_back(std::move(x));
  }
  std::vector<Json> outcomes(orders.size());
  std::size_t applied = 0;
  if (mode == "shared_snapshot") {
    applied = orders.empty() ? 0 : 1;
    std::string dependency;
    for (std::size_t i = 0; i < orders.size(); ++i) {
      auto it = snapshots.find(orders[i].frame_ordinal);
      outcomes[i] =
          simulate(orders[i], it == snapshots.end() ? nullptr : &it->second, p,
                   dependency, end);
      if (outcomes[i].at("status") == "unavailable")
        dependency =
            "earlier shared order unresolved; remaining liquidity unknown";
    }
  } else {
    applied = std::min<std::size_t>(workers, orders.size());
    std::atomic<std::size_t> next{0};
    std::atomic<bool> failed{false};
    std::exception_ptr error;
    std::mutex mutex;
    const auto worker = [&] {
      try {
        while (!failed) {
          const auto i = next.fetch_add(1);
          if (i >= orders.size())
            break;
          auto it = snapshots.find(orders[i].frame_ordinal);
          std::optional<Snapshot> copy;
          if (it != snapshots.end())
            copy = it->second;
          outcomes[i] =
              simulate(orders[i], copy ? &*copy : nullptr, p, "", end);
        }
      } catch (...) {
        failed = true;
        std::lock_guard lock(mutex);
        if (!error)
          error = std::current_exception();
      }
    };
    {
      std::vector<std::jthread> threads;
      for (std::size_t i = 0; i < applied; ++i)
        threads.emplace_back(worker);
    }
    if (error)
      std::rethrow_exception(error);
  }
  Json execution = Json::array(), selected = Json::array(),
       quality = Json::array();
  std::set<std::string> selected_ids;
  std::uint64_t available = 0, full = 0, partial = 0, none = 0;
  for (std::size_t i = 0; i < orders.size(); ++i) {
    deadline(end);
    const auto &o = orders[i];
    auto &row = outcomes[i];
    if (selected_ids.insert(o.signal).second)
      selected.push_back(signals.at(o.signal));
    if (row.at("status") == "available") {
      ++available;
      const auto &data = row.at("data");
      const auto q = u64(data.at("filled_quantity"));
      if (q == o.quantity)
        ++full;
      else if (q)
        ++partial;
      else
        ++none;
    }
    if (studies.contains("fill_quality")) {
      Json metrics = nullptr;
      std::string reason = str(row.at("reason"));
      if (row.at("status") == "available") {
        const auto q = u64(row.at("data").at("filled_quantity"));
        Wide total = 0;
        for (const auto &f : row.at("data").at("fills"))
          total += static_cast<Wide>(i64(f.at("price_nanos"))) *
                   u64(f.at("quantity"));
        metrics = {
            {"fill_ratio", ratio(q, o.quantity)},
            {"adverse_slippage_price_nanos",
             q ? Json{{"status", "available"},
                      {"value",
                       ratio((o.buy ? 1 : -1) *
                                 (total - static_cast<Wide>(o.anchor) * q),
                             q)}}
               : missing("no scenario fill")}};
      }
      quality.push_back({{"order_id", o.id},
                         {"status", row.at("status")},
                         {"reason", reason},
                         {"data", metrics}});
    }
    execution.push_back(std::move(row));
  }
  auto result = base("native displayed-depth execution scenarios");
  result["status"] = "partial";
  auto &sections = result["sections"];
  sections["signals"] = section(selected);
  sections["execution"] =
      section(execution, available == orders.size() ? "available" : "partial",
              "scenario evidence; calibration not verified");
  sections["summary"] =
      section({{"order_count", dec(orders.size())},
               {"available", dec(available)},
               {"unavailable", dec(orders.size() - available)},
               {"source_census_sha256",
                s.at("summary").at("data").at("parent_census_sha256")}});
  Json computed = Json::array();
  if (studies.contains("liquidity_summary"))
    computed.push_back(
        {{"study_id", "liquidity_summary"},
         {"version", "1"},
         {"available", dec(available)},
         {"unavailable", dec(orders.size() - available)},
         {"full_fill", dec(full)},
         {"partial_fill", dec(partial)},
         {"no_fill", dec(none)},
         {"interpretation", "conditional scenario counts, not "
                            "frequency-estimated probabilities"}});
  if (studies.contains("fill_quality"))
    computed.push_back(
        {{"study_id", "fill_quality"},
         {"version", "1"},
         {"results", quality},
         {"interpretation",
          "conditional filled/requested ratio and side-adjusted VWAP minus "
          "signal anchor; no fees, markout or account P&L"}});
  sections["studies"] =
      section(computed, studies.empty() ? "not_selected" : "available",
              studies.empty() ? "zero studies selected" : "");
  for (const auto *name : {"replay", "book_frames", "book_checkpoint"})
    sections[name] = s.at(name);
  sections["source_context"] = section({{"signals", s.at("signals")},
                                        {"choices", s.at("choices")},
                                        {"provenance", s.at("provenance")}});
  sections["resources"] = section(
      {{"backend", "cpu"},
       {"requested_workers", dec(workers)},
       {"actual_workers", dec(applied)},
       {"ordering",
        mode == "shared_snapshot"
            ? "sequential input order on one snapshot"
            : "independent immutable snapshots; stable input order output"}});
  auto choices = p;
  choices.erase("output_path");
  sections["choices"] = section(choices);
  sections["diagnostics"] = section(Json::array(
      {"Snapshot consumption is hypothetical; market response, hidden "
       "liquidity, queue priority and venue routing are not modeled.",
       "Frame selection is explicit post-signal evidence; frame delay is not a "
       "calibrated latency model or an inferred arrival book.",
       "Activation probability is a user assumption; any/full/no-fill "
       "probabilities and quantity mass follow only the selected scenario.",
       "Shared snapshot mode is conditioned on all supplied orders being "
       "active; missing earlier outcomes leave dependent later outcomes "
       "unavailable.",
       "Per-level participation is floor(original displayed size * selected "
       "fraction), capped by physical remainder. It is not a replenishment "
       "model."}));
  sections["provenance"] =
      section({{"engine_version", version},
               {"source_result_sha256", source.at("content_sha256")},
               {"source_census_sha256",
                s.at("summary").at("data").at("parent_census_sha256")},
               {"model_id", "displayed_depth_sweep"},
               {"model_version", "1"},
               {"numeric_profile", "exact signed 128-bit notional and reduced "
                                   "rational VWAP; integer source quantities"},
               {"provider_requests", "0"},
               {"additional_spend_usd", "0"}});
  sections["user_extensions"] = section(p.at("extensions"));
  return persist(std::move(result), p, "liquidity", end);
}
} // namespace symphony::sbv::detail
