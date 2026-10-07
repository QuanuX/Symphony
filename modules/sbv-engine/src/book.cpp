#include "census.hpp"
#include "detail.hpp"
#include <algorithm>
#include <map>
#include <symphony/sqav/databento/dbn.hpp>
namespace symphony::sbv::detail {
namespace db = sqav::databento;
Json event_json(const db::Mbo &, std::size_t);
std::uint64_t end_at(std::uint64_t, std::uint64_t);
namespace {
struct Order {
  std::int64_t price;
  std::uint32_t size;
  char side;
};
struct Level {
  std::uint64_t size{}, orders{};
};
// Ordered maps make exported levels and full checkpoints deterministic. The
// display depth never limits the state used by subsequent updates.
struct Book {
  std::map<std::uint64_t, Order> orders;
  std::map<std::int64_t, Level> bids, asks;
  bool initialized{}, loading{}, boundary{};
  std::string reason = "initial state absent", origin = "none";
  void clear() {
    orders.clear();
    bids.clear();
    asks.clear();
  }
  void add(std::uint64_t id, Order o) {
    auto &l = (o.side == 'B' ? bids : asks)[o.price];
    need(l.size <= UINT64_MAX - o.size, "aggregate quantity overflow");
    l.size += o.size;
    ++l.orders;
    orders.emplace(id, o);
  }
  void remove(std::uint64_t id) {
    const auto o = orders.at(id);
    auto &levels = o.side == 'B' ? bids : asks;
    auto &l = levels.at(o.price);
    l.size -= o.size;
    if (--l.orders == 0)
      levels.erase(o.price);
    orders.erase(id);
  }
  bool ready() const { return initialized && !loading && boundary; }
  Json depth(std::size_t maximum) const {
    Json b = Json::array(), a = Json::array();
    auto row = [](const auto &x) -> Json {
      return {{"price_nanos", dec(x.first)},
              {"size", dec(x.second.size)},
              {"order_count", dec(x.second.orders)}};
    };
    for (auto i = bids.rbegin(); i != bids.rend() && b.size() < maximum; ++i)
      b.push_back(row(*i));
    for (auto i = asks.begin(); i != asks.end() && a.size() < maximum; ++i)
      a.push_back(row(*i));
    std::string market = "one_or_both_sides_empty";
    if (!bids.empty() && !asks.empty())
      market = bids.rbegin()->first < asks.begin()->first    ? "uncrossed"
               : bids.rbegin()->first == asks.begin()->first ? "locked"
                                                             : "crossed";
    return {{"bids", b},
            {"asks", a},
            {"order_count", dec(orders.size())},
            {"bid_level_count", dec(bids.size())},
            {"ask_level_count", dec(asks.size())},
            {"market_state", market}};
  }
};
Json load_result(const Json &ref, std::int64_t end) {
  keys(ref, {"path", "expected_sha256"});
  auto result = e::parse_bounded_json(read_file(str(ref.at("path")), end),
                                      artifact_bytes, artifact_values);
  validate_result(result);
  need(!str(ref.at("expected_sha256")).empty() &&
           result.at("content_sha256") == ref.at("expected_sha256"),
       "result identity mismatch");
  return result;
}
} // namespace
Json book(const Json &p, std::int64_t end, const Dataset *resident) {
  keys_optional(p,
                {"protocol", "census_result", "signal_ids", "output_path",
                 "initial_state", "on_anomaly", "replay", "frames",
                 "emit_checkpoint", "extensions"},
                {"memory_budget_bytes", "dataset_limits", "source_path",
                 "source_sha256", "dataset", "retained_source"});
  need(p.at("extensions").is_object() && p.at("emit_checkpoint").is_boolean(),
       "invalid book choices");
  const auto policy = str(p.at("on_anomaly"));
  need(policy == "reject" || policy == "invalidate_until_reset",
       "unknown book anomaly policy");
  const auto &replay = p.at("replay"), &frames = p.at("frames");
  keys(replay, {"before_ns", "after_ns", "retain_events"});
  keys(frames, {"cadence", "levels_per_side", "maximum"});
  const auto before = u64(replay.at("before_ns")),
             after = u64(replay.at("after_ns"));
  need(replay.at("retain_events").is_boolean(), "retention boolean required");
  const auto depth = u64(frames.at("levels_per_side")),
             maximum = u64(frames.at("maximum"));
  const auto cadence = str(frames.at("cadence"));
  need(depth >= 1 && depth <= 64 && maximum >= 1 && maximum <= 4096 &&
           (cadence == "signals" || cadence == "signals_and_event_ends"),
       "book frame bounds/cadence");
  const auto owned = resident ? nullptr : load_dataset(p, end);
  const auto &source = resident ? *resident : *owned;
  source.bind(p);
  const auto &events = source.events;

  need(source.book_compatible,
       "book requires one publisher/instrument/channel");
  const auto parent = load_result(p.at("census_result"), end);
  const auto admitted = census_evidence(parent);
  validate_census_source(admitted, source, end);
  const auto &ps = parent.at("sections"),
             &prov = ps.at("provenance").at("data"),
             &signals = admitted.at("signals");
  need(prov.at("source_sha256") == source.sha256 &&
           prov.at("dataset") == source.dataset_name &&
           prov.at("instrument_id") == dec(events[0].instrument_id) &&
           ps.at("summary").at("data").at("closed_census") == true &&
           signals.is_array(),
       "source-bound closed census required");
  std::map<std::string, Json> census;
  for (const auto &s : signals) {
    const auto id = str(s.at("signal_id"));
    const auto ordinal = u64(s.at("source_ordinal"));
    need(census.emplace(id, s).second && ordinal < events.size() &&
             u64(s.at("available_ns")) == events[ordinal].ts_recv &&
             u64(s.at("causal_end_ordinal_exclusive")) <= events.size(),
         "census coordinates invalid");
  }
  need(p.at("signal_ids").is_array() && p.at("signal_ids").size() <= 4096,
       "signal selection bound");
  Json selected = Json::array(), windows = Json::array();
  std::vector<bool> retained(events.size(), false),
      wanted(events.size(), false);
  std::map<std::size_t, Json> at_signal;
  std::set<std::string> ids;
  std::size_t first_needed = events.size(), finish = 0;
  for (const auto &idv : p.at("signal_ids")) {
    const auto id = str(idv);
    need(ids.insert(id).second && census.contains(id),
         "unknown or duplicate signal");
    const auto &s = census.at(id);
    selected.push_back(s);
    const auto ordinal = u64(s.at("source_ordinal")),
               t = events[ordinal].ts_recv;
    if (!at_signal.contains(ordinal))
      at_signal[ordinal] = Json::array();
    at_signal[ordinal].push_back(id);
    wanted[ordinal] = true;
    const auto start = t < before ? 0 : t - before, stop = end_at(t, after);
    const auto a = static_cast<std::size_t>(
        std::lower_bound(events.begin(), events.end(), start,
                         [](const auto &x, auto n) { return x.ts_recv < n; }) -
        events.begin());
    const auto b = static_cast<std::size_t>(
        std::upper_bound(events.begin(), events.end(), stop,
                         [](auto n, const auto &x) { return n < x.ts_recv; }) -
        events.begin());
    first_needed = std::min(first_needed, a);
    finish = std::max(finish, b);
    std::fill(retained.begin() + a, retained.begin() + b, true);
    windows.push_back(
        {{"signal_id", id},
         {"first_ordinal", dec(a)},
         {"end_ordinal_exclusive", dec(b)},
         {"requested_start_ns", dec(start)},
         {"requested_end_ns_inclusive", dec(stop)},
         {"start_clamped_at_epoch", t < before},
         {"start_within_observed_span", start >= events.front().ts_recv},
         {"end_within_observed_span", stop <= events.back().ts_recv},
         {"completeness", "unproven"}});
  }
  if (cadence == "signals_and_event_ends")
    for (std::size_t i = 0; i < events.size(); ++i)
      if (retained[i] && (events[i].flags & 128))
        wanted[i] = true;
  need(static_cast<std::uint64_t>(
           std::count(wanted.begin(), wanted.end(), true)) <= maximum,
       "frame limit exceeded; choose a smaller window or signals cadence");
  const Json binding{{"source_sha256", source.sha256},
                     {"dataset", source.dataset_name},
                     {"publisher_id", dec(events[0].publisher_id)},
                     {"instrument_id", dec(events[0].instrument_id)},
                     {"channel_id", dec(events[0].channel_id)}};
  Book state;
  std::size_t begin = 0;
  const auto &initial = p.at("initial_state");
  need(initial.is_object() && initial.contains("mode"),
       "initial state mode required");
  if (initial.at("mode") == "uninitialized")
    keys(initial, {"mode"});
  else {
    keys(initial, {"mode", "result"});
    need(initial.at("mode") == "checkpoint", "unknown initial state mode");
    const auto prior = load_result(initial.at("result"), end);
    const auto &section = prior.at("sections").at("book_checkpoint");
    need(section.at("status") == "available", "checkpoint unavailable");
    const auto &c = section.at("data");
    keys(c, {"protocol", "profile", "binding", "next_ordinal",
             "preceding_available_ns", "event_boundary", "initialization",
             "orders"});
    need(c.at("protocol") == "symphony.sbv.book-checkpoint.v1" &&
             c.at("profile") == "databento_mbo_orders_strict_v1" &&
             c.at("binding") == binding && c.at("event_boundary") == true,
         "checkpoint scope/profile mismatch");
    begin = u64(c.at("next_ordinal"));
    need(begin > 0 && begin <= events.size() && begin <= first_needed &&
             (selected.empty() || begin <= finish) &&
             (events[begin - 1].flags & 128) &&
             u64(c.at("preceding_available_ns")) == events[begin - 1].ts_recv,
         "checkpoint cursor or requested window mismatch");
    need(c.at("orders").is_array() && c.at("orders").size() <= 200000,
         "checkpoint order bound");
    std::uint64_t previous = 0;
    bool first = true;
    for (const auto &o : c.at("orders")) {
      keys(o, {"order_id", "price_nanos", "size", "side"});
      const auto id = u64(o.at("order_id")), size = u64(o.at("size"));
      const auto price = i64(o.at("price_nanos"));
      const auto side = str(o.at("side"));
      need((first || id > previous) && size > 0 && size <= UINT32_MAX &&
               price != INT64_MAX && (side == "B" || side == "A"),
           "checkpoint order invalid");
      state.add(id, {price, static_cast<std::uint32_t>(size), side[0]});
      previous = id;
      first = false;
    }
    state.initialized = true;
    state.boundary = true;
    state.reason = "";
    state.origin = "supplied checkpoint; integrity checked, authorship and "
                   "state truth unverified";
    (void)str(c.at("initialization"));
  }
  if (selected.empty())
    finish = begin;
  Json diagnostics = Json::array(), output = Json::array(), raw = Json::array();
  std::uint64_t inaccurate = 0, unavailable = 0;
  auto anomaly = [&](std::size_t i, const char *reason) {
    if (policy == "reject")
      need(false, reason);
    diagnostics.push_back({{"source_ordinal", dec(i)},
                           {"kind", "invalidation"},
                           {"reason", reason}});
    state.clear();
    state.initialized = false;
    state.loading = false;
    state.reason = reason;
    state.origin = "none";
  };
  for (std::size_t i = begin; i < finish; ++i) {
    if (i % 1024 == 0)
      deadline(end);
    const auto &x = events[i];
    state.boundary = (x.flags & 128) != 0;
    if (x.flags & 8)
      ++inaccurate;
    // Bit 1 is explicitly reserved by DBN. Publisher-specific semantics are
    // not guessed. Sequence numbers are retained, never treated as contiguous
    // in a filtered single-instrument stream.
    if (x.flags & (64 | 16 | 4 | 2))
      anomaly(i, "unsupported TOB/MBP/publisher-specific semantics or reported "
                 "book gap");
    else if (x.action == 'R') {
      state.clear();
      state.initialized = true;
      state.loading = (x.flags & 32) != 0;
      state.reason = "";
      state.origin = "source reset at ordinal " + dec(i);
      diagnostics.push_back({{"source_ordinal", dec(i)},
                             {"kind", "reset"},
                             {"reason", state.origin}});
    } else if ((x.flags & 32) && state.initialized && !state.loading)
      anomaly(i, "snapshot records without an opening reset");
    else if (x.action != 'A' && x.action != 'M' && x.action != 'C' &&
             x.action != 'T' && x.action != 'F' && x.action != 'N')
      anomaly(i, "unsupported order action");
    else if (state.initialized &&
             (x.action == 'A' || x.action == 'M' || x.action == 'C')) {
      if ((x.side != 'A' && x.side != 'B') || x.price == INT64_MAX ||
          x.size == 0)
        anomaly(i, "invalid order side/price/size");
      else {
        const auto found = state.orders.find(x.order_id);
        if (x.action == 'A') {
          if (found != state.orders.end())
            anomaly(i, "duplicate add order");
          else
            state.add(x.order_id, {x.price, x.size, static_cast<char>(x.side)});
        } else if (found == state.orders.end())
          anomaly(i, "modify/cancel references absent order");
        else if (x.action == 'M') {
          state.remove(x.order_id);
          state.add(x.order_id, {x.price, x.size, static_cast<char>(x.side)});
        } else {
          auto o = found->second;
          if (o.side != x.side || o.price != x.price || x.size > o.size)
            anomaly(i, "cancel side/price/quantity mismatch");
          else {
            state.remove(x.order_id);
            o.size -= x.size;
            if (o.size)
              state.add(x.order_id, o);
          }
        }
      }
    }
    // Trades and fills are informational. Their accompanying C/M records own
    // quantity changes; applying both would consume liquidity twice.
    if (state.initialized && state.boundary)
      state.loading = false;
    if (wanted[i]) {
      const auto ready = state.ready();
      if (!ready)
        ++unavailable;
      const auto reason = ready                ? ""
                          : !state.initialized ? state.reason
                                               : "event or snapshot incomplete";
      output.push_back({{"protocol", "symphony.sbv.book-frame.v1"},
                        {"source_ordinal", dec(i)},
                        {"available_ns", dec(x.ts_recv)},
                        {"signal_ids", at_signal.contains(i) ? at_signal.at(i)
                                                             : Json::array()},
                        {"status", ready ? "available" : "unavailable"},
                        {"reason", reason},
                        {"event_boundary", state.boundary},
                        {"receive_time_flagged", (x.flags & 8) != 0},
                        {"initialization", state.origin},
                        {"layer", "reconstructed"},
                        {"data", ready ? state.depth(depth) : Json(nullptr)}});
    }
    if (retained[i] && replay.at("retain_events").get<bool>())
      raw.push_back(event_json(x, i));
  }
  auto result = base("native source-bound order-book reconstruction");
  result["status"] = "partial";
  auto &s = result["sections"];
  s["signals"] = section(selected);
  s["summary"] =
      section({{"signal_count", dec(selected.size())},
               {"frame_count", dec(output.size())},
               {"available_frames", dec(output.size() - unavailable)},
               {"unavailable_frames", dec(unavailable)},
               {"parent_census_sha256",
                ps.at("summary").at("data").at("census_sha256")},
               {"closed_census", true}});
  s["book_frames"] =
      section(output, unavailable ? "partial" : "available",
              unavailable ? "some frames lack usable initialized state" : "");
  s["book_checkpoint"] =
      section(nullptr, "not_selected", "checkpoint not selected");
  if (p.at("emit_checkpoint").get<bool>()) {
    if (finish > begin && state.ready()) {
      Json orders = Json::array();
      for (const auto &[id, o] : state.orders)
        orders.push_back({{"order_id", dec(id)},
                          {"price_nanos", dec(o.price)},
                          {"size", dec(o.size)},
                          {"side", std::string(1, o.side)}});
      s["book_checkpoint"] =
          section({{"protocol", "symphony.sbv.book-checkpoint.v1"},
                   {"profile", "databento_mbo_orders_strict_v1"},
                   {"binding", binding},
                   {"next_ordinal", dec(finish)},
                   {"preceding_available_ns", dec(events[finish - 1].ts_recv)},
                   {"event_boundary", true},
                   {"initialization", state.origin},
                   {"orders", orders}});
    } else
      s["book_checkpoint"] =
          section(nullptr, "unavailable",
                  "no complete initialized state at selected end");
  }
  s["replay"] = section(
      {{"windows", windows},
       {"events", raw},
       {"retained_event_count", dec(raw.size())},
       {"retention_policy", replay},
       {"clock", "ts_recv with source ordinal tie-break"},
       {"price_scale", "1e-9"},
       {"observed_payload_layer", "observed"},
       {"frame_pointer", "/sections/book_frames/data"},
       {"hydration_first_ordinal", dec(begin)},
       {"hydration_end_ordinal_exclusive", dec(finish)},
       {"hydration_payload", "full source/checkpoint reference; raw retention "
                             "covers selected windows only"}},
      "partial", "source completeness and queue/fill feasibility unverified");
  s["diagnostics"] = section(
      {{"events", diagnostics},
       {"inaccurate_receive_timestamp_count", dec(inaccurate)},
       {"sequence_policy",
        "retained; no contiguous sequence inference on filtered instrument"},
       {"initial_state", initial},
       {"calibration", "not verified"},
       {"queue_priority", "not modeled"}});
  s["resources"] =
      section({{"backend", "cpu"},
               {"state_workers", "1"},
               {"reason", "ordered book state is sequential; independent runs "
                          "can be scheduled concurrently"},
               {"maximum_frames", dec(maximum)},
               {"display_levels_per_side", dec(depth)},
               {"full_state_order_count", dec(state.orders.size())}});
  auto choices = p;
  choices.erase("output_path");
  s["choices"] = section(choices);
  s["provenance"] =
      section({{"engine_version", version},
               {"binding", binding},
               {"source_path", source.path},
               {"adapter", db::adapter_id},
               {"adapter_version", db::adapter_version},
               {"dbn_version", dec(source.metadata.version)},
               {"parent_result_sha256", parent.at("content_sha256")},
               {"parent_census", ps.at("summary")},
               {"parent_provenance", ps.at("provenance")},
               {"profile", "databento_mbo_orders_strict_v1"},
               {"provider_requests", "0"},
               {"additional_spend_usd", "0"}});
  s["user_extensions"] = section(p.at("extensions"));
  result["sections"]["resources"]["data"]["dataset_feed"] =
      source.evidence(resident != nullptr);
  return persist(std::move(result), p, "book", end);
}
} // namespace symphony::sbv::detail
