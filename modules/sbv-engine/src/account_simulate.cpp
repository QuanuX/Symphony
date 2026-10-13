#include "consumer_source.hpp"
#include "depth_sweep.hpp"
#include "provider.hpp"
#include "wide_rational.hpp"
#include <map>
#include <optional>
#include <queue>

namespace symphony::sbv::detail {
namespace {
namespace w = wide_rational;
using Balances = std::map<std::string, w::R>;
struct Instrument {
  std::string unit;
  w::R multiplier;
  bool variation = false;
};
std::string identity(const Json &v) {
  auto s = str(v);
  need(!s.empty() && s.size() <= 256, "account identity must be 1..256 bytes");
  return s;
}
Json balances(const Balances &values) {
  Json out = Json::object();
  for (const auto &[id, value] : values)
    out[id] = w::wire(value);
  return out;
}
struct State {
  Balances cash, positions, marks, basis;
  bool feedback = false;
  Json memory = Json::object(), previous = nullptr;
  std::map<std::string, depth_sweep::Snapshot> pools;
  std::optional<std::uint64_t> time;
};
Json snapshot(const State &s, const std::string &model) {
  Json out = {{"protocol", "symphony.sbv.account-state.v1"},
              {"model_sha256", model},
              {"cash", balances(s.cash)},
              {"positions", balances(s.positions)},
              {"marks", balances(s.marks)},
              {"time_ns", s.time ? Json(dec(*s.time)) : Json(nullptr)}};
  if (!s.pools.empty()) {
    out["protocol"] = "symphony.sbv.account-state.v2";
    Json pools = Json::object();
    for (const auto &[id, pool] : s.pools) {
      Json bids = Json::array(), asks = Json::array();
      for (const auto &v : pool.bids)
        bids.push_back(dec(v.remaining));
      for (const auto &v : pool.asks)
        asks.push_back(dec(v.remaining));
      pools[id] = {{"bids", bids}, {"asks", asks}};
    }
    out["liquidity_remaining"] = pools;
  }
  if (!s.basis.empty() || s.feedback) {
    out["protocol"] = "symphony.sbv.account-state.v3";
    out["basis"] = balances(s.basis);
    if (s.feedback) {
      out["feedback_memory"] = s.memory;
      out["feedback_previous"] = s.previous;
    }
  }
  return out;
}
struct Bounds {
  std::optional<w::R> minimum, maximum;
};
Bounds bounds(const Json &v) {
  keys(v, {"minimum", "maximum"});
  Bounds b;
  if (!v.at("minimum").is_null())
    b.minimum = w::artifact(v.at("minimum"));
  if (!v.at("maximum").is_null())
    b.maximum = w::artifact(v.at("maximum"));
  need(!b.minimum || !b.maximum || w::compare(*b.minimum, *b.maximum) <= 0,
       "account bounds reversed");
  return b;
}
bool inside(w::R v, const Bounds &b) {
  return (!b.minimum || w::compare(v, *b.minimum) >= 0) &&
         (!b.maximum || w::compare(v, *b.maximum) <= 0);
}
struct Stream {
  std::string id;
  consumer::InputRows input;
  std::uint64_t ordinal = 0, time = 0;
  Json row;
  std::set<std::string> ids;
};
void load(Stream &s, std::optional<std::uint64_t> previous, std::int64_t end) {
  deadline(end);
  s.row = s.input.value.at(s.ordinal).materialize();
  need(s.row.is_object() && s.row.contains("time_ns") &&
           s.row.contains("event_id"),
       "account event identity/time required");
  s.time = u64(s.row.at("time_ns"));
  need(!previous || s.time >= *previous, "account stream time decreases");
  need(s.ids.insert(identity(s.row.at("event_id"))).second,
       "duplicate event identity within account stream");
}
bool guards(const Json &values, const State &s) {
  need(values.is_array(), "account guards array required");
  bool admitted = true;
  for (const auto &g : values) {
    keys(g, {"balance", "key", "comparison", "value"});
    auto kind = str(g.at("balance")), key = identity(g.at("key"));
    need(kind == "cash" || kind == "position", "unknown guard balance");
    const auto &map = kind == "cash" ? s.cash : s.positions;
    need(map.contains(key), "unknown guard key");
    int c = w::compare(map.at(key), w::artifact(g.at("value")));
    auto op = str(g.at("comparison"));
    need(op == "lt" || op == "le" || op == "eq" || op == "ne" || op == "ge" ||
             op == "gt",
         "unknown guard comparison");
    admitted &= op == "lt"   ? c < 0
                : op == "le" ? c <= 0
                : op == "eq" ? c == 0
                : op == "ne" ? c != 0
                : op == "ge" ? c >= 0
                             : c > 0;
  }
  return admitted;
}
logical::Value body(const Json &p, PartitionedOutput *context,
                    std::int64_t end) {
  keys_optional(p,
                {"protocol", "mode", "cash_units", "instruments",
                 "initial_state", "constraints", "on_constraint", "streams",
                 "ordering", "retain_snapshots", "extensions"},
                {"output", "output_path", "liquidity_pools", "feedback"});
  need(p.at("extensions").is_object() && p.at("retain_snapshots").is_boolean(),
       "account extensions/retention invalid");
  need(p.at("mode") == "fixed_intents" || p.at("mode") == "stateful_feedback",
       "explicit account mode required");
  need(p.at("ordering") == "time_stream_order", "account ordering unsupported");
  need(p.at("on_constraint") == "reject_event" ||
           p.at("on_constraint") == "abort",
       "account constraint disposition required");
  State state;
  std::unique_ptr<NativeProvider> provider;
  std::unique_ptr<NativeProviderInstance> instance;
  if (p.contains("feedback")) {
    keys(p.at("feedback"), {"provider", "on_unavailable"});
    need(p.at("mode") == "stateful_feedback" &&
             p.at("feedback").at("provider").at("role") == "feedback",
         "feedback provider requires stateful feedback mode");
    need(p.at("feedback").at("on_unavailable") == "abort" ||
             p.at("feedback").at("on_unavailable") == "skip_trigger",
         "feedback unavailable policy required");
    provider =
        std::make_unique<NativeProvider>(p.at("feedback").at("provider"), end);
    instance = provider->create(end);
    state.feedback = true;
  }
  need(p.at("cash_units").is_array() && p.at("instruments").is_array(),
       "cash units/instruments arrays required");
  for (const auto &unit : p.at("cash_units"))
    need(state.cash.emplace(identity(unit), w::R{}).second,
         "duplicate cash unit");
  std::map<std::string, Instrument> instruments;
  for (const auto &v : p.at("instruments")) {
    keys(v, {"instrument_id", "cash_unit", "settlement", "price_multiplier"});
    auto id = identity(v.at("instrument_id")),
         unit = identity(v.at("cash_unit"));
    auto multiplier = w::artifact(v.at("price_multiplier"));
    need(state.cash.contains(unit) && multiplier.n > 0 &&
             (v.at("settlement") == "cash_notional" ||
              v.at("settlement") == "linear_variation"),
         "unsupported account instrument economics");
    need(instruments
             .emplace(id, Instrument{unit, multiplier,
                                     v.at("settlement") == "linear_variation"})
             .second,
         "duplicate account instrument");
    state.positions.emplace(id, w::R{});
    if (v.at("settlement") == "linear_variation")
      state.basis.emplace(id, w::R{});
  }
  std::map<std::string, std::string> pool_instruments;
  Json pool_evidence = Json::array();
  std::set<std::string> pool_sources;
  if (p.contains("liquidity_pools")) {
    need(p.at("liquidity_pools").is_array(), "liquidity pools array required");
    for (const auto &v : p.at("liquidity_pools")) {
      deadline(end);
      keys(v, {"pool_id", "instrument_id", "source"});
      auto id = identity(v.at("pool_id")),
           instrument = identity(v.at("instrument_id"));
      need(instruments.contains(instrument), "unknown pool instrument");
      auto admitted = consumer::value_source(v.at("source"), end);
      auto selected = admitted.value.materialize();
      auto source_key =
          str(admitted.outer_digest) + ":" + e::sha256_hex(selected.dump());
      need(pool_sources.insert(source_key).second,
           "duplicate selected pool source");
      need(state.pools.emplace(id, depth_sweep::frame(selected)).second,
           "duplicate liquidity pool");
      pool_instruments.emplace(id, instrument);
      pool_evidence.push_back({{"pool_id", id},
                               {"instrument_id", instrument},
                               {"reference", admitted.reference},
                               {"frame", selected}});
    }
  }
  Json model = {{"cash_units", p.at("cash_units")},
                {"instruments", p.at("instruments")}};
  if (!state.pools.empty())
    model["liquidity_pools"] = p.at("liquidity_pools");
  if (provider)
    model["feedback"] = p.at("feedback");
  const auto model_hash = e::sha256_hex(model.dump());
  Json initial = p.at("initial_state");
  std::optional<consumer::ValueSource> parent;
  if (initial.contains("kind")) {
    keys(initial, {"kind", "source"});
    need(initial.at("kind") == "reference", "unknown initial state kind");
    parent = consumer::value_source(initial.at("source"), end);
    initial = parent->value.materialize();
    keys_optional(
        initial,
        {"protocol", "model_sha256", "cash", "positions", "marks", "time_ns"},
        {"liquidity_remaining", "basis", "feedback_memory",
         "feedback_previous"});
    need(initial.at("protocol") == ((!state.basis.empty() || state.feedback)
                                        ? "symphony.sbv.account-state.v3"
                                    : state.pools.empty()
                                        ? "symphony.sbv.account-state.v1"
                                        : "symphony.sbv.account-state.v2") &&
             initial.at("model_sha256") == model_hash,
         "account snapshot model mismatch");
    need(initial.contains("basis") == (!state.basis.empty() || state.feedback),
         "snapshot basis coverage mismatch");
    need(initial.contains("liquidity_remaining") == !state.pools.empty(),
         "snapshot liquidity coverage mismatch");
    if (!state.pools.empty()) {
      const auto &remaining = initial.at("liquidity_remaining");
      need(remaining.is_object() && remaining.size() == state.pools.size(),
           "snapshot pool keys mismatch");
      for (auto &[id, pool] : state.pools) {
        need(remaining.contains(id), "snapshot pool absent");
        const auto &v = remaining.at(id);
        keys(v, {"bids", "asks"});
        auto restore = [&](const Json &rows, auto &levels) {
          need(rows.is_array() && rows.size() == levels.size(),
               "snapshot depth mismatch");
          for (std::size_t i = 0; i < levels.size(); ++i) {
            auto q = u64(rows.at(i));
            need(q <= levels[i].original,
                 "snapshot remaining exceeds original capacity");
            levels[i].remaining = q;
          }
        };
        restore(v.at("bids"), pool.bids);
        restore(v.at("asks"), pool.asks);
      }
    }
    if (!initial.at("time_ns").is_null())
      state.time = u64(initial.at("time_ns"));
  } else
    keys_optional(initial, {"cash", "positions", "marks", "time_ns"},
                  {"basis", "feedback_memory", "feedback_previous"});
  if (!initial.at("time_ns").is_null())
    state.time = u64(initial.at("time_ns"));
  auto read = [&](const Json &v, Balances &out, bool marks) {
    need(v.is_object() && (marks || v.size() == out.size()),
         "initial account keys mismatch");
    for (const auto &[id, value] : v.items()) {
      need(marks ? instruments.contains(id) : out.contains(id),
           "unknown initial account key");
      out[id] = w::artifact(value);
    }
  };
  read(initial.at("cash"), state.cash, false);
  read(initial.at("positions"), state.positions, false);
  read(initial.at("marks"), state.marks, true);
  need(state.basis.empty() || initial.contains("basis"),
       "explicit basis required for variation instruments");
  if (initial.contains("basis"))
    read(initial.at("basis"), state.basis, false);
  need(state.feedback || (!initial.contains("feedback_memory") &&
                          !initial.contains("feedback_previous")),
       "feedback state requires provider selection");
  if (initial.contains("feedback_memory"))
    state.memory = initial.at("feedback_memory");
  if (initial.contains("feedback_previous"))
    state.previous = initial.at("feedback_previous");
  need(state.memory.is_object() &&
           (state.previous.is_null() || state.previous.is_object()),
       "feedback state objects required");
  if (parent && state.feedback)
    need(initial.contains("feedback_memory") &&
             initial.contains("feedback_previous"),
         "feedback snapshot state incomplete");
  const auto &constraints = p.at("constraints");
  keys(constraints, {"cash", "positions"});
  std::map<std::string, Bounds> cash_bounds, position_bounds;
  auto limits = [&](const Json &v, const Balances &values, auto &out) {
    need(v.is_object() && v.size() == values.size(),
         "explicit bounds for every balance required");
    for (const auto &[id, b] : v.items()) {
      need(values.contains(id), "unknown constraint key");
      out.emplace(id, bounds(b));
    }
  };
  limits(constraints.at("cash"), state.cash, cash_bounds);
  limits(constraints.at("positions"), state.positions, position_bounds);
  auto violations = [&](const State &s) {
    Json v = Json::array();
    for (const auto &[id, b] : cash_bounds)
      if (!inside(s.cash.at(id), b))
        v.push_back({{"balance", "cash"}, {"key", id}});
    for (const auto &[id, b] : position_bounds)
      if (!inside(s.positions.at(id), b))
        v.push_back({{"balance", "position"}, {"key", id}});
    return v;
  };
  need(violations(state).empty(),
       "initial state violates selected account bounds");
  need(p.at("streams").is_array(), "account streams array required");
  std::vector<Stream> streams;
  std::set<std::string> stream_ids;
  for (const auto &v : p.at("streams")) {
    deadline(end);
    keys(v, {"stream_id", "events"});
    auto id = identity(v.at("stream_id"));
    need(stream_ids.insert(id).second, "duplicate account stream identity");
    auto input = consumer::input_rows(v.at("events"), "account_events", end);
    need(input.value.kind() == "array", "account event rows required");
    streams.push_back({id, std::move(input), 0, 0, {}, {}});
  }
  auto later = [&](std::size_t a, std::size_t b) {
    return streams[a].time != streams[b].time
               ? streams[a].time > streams[b].time
               : a > b;
  };
  std::priority_queue<std::size_t, std::vector<std::size_t>, decltype(later)>
      queue(later);
  for (std::size_t i = 0; i < streams.size(); ++i)
    if (streams[i].input.value.size()) {
      load(streams[i], state.time, end);
      queue.push(i);
    }
  consumer::Rows ledger(context, "account-ledger"),
      signals(context, "account-signals"),
      snapshots(context, "account-snapshots");
  const auto initial_snapshot = snapshot(state, model_hash);
  if (p.at("retain_snapshots").get<bool>())
    snapshots.append(initial_snapshot);
  std::uint64_t applied = 0, rejected = 0, suppressed = 0, unavailable = 0;
  while (!queue.empty()) {
    deadline(end);
    auto index = queue.top();
    queue.pop();
    auto &stream = streams[index];
    Json event = stream.row, feedback_record = nullptr;
    bool no_action = false, provider_missing = false;
    if (event.at("kind") == "feedback_trigger") {
      keys(event, {"event_id", "time_ns", "kind", "context"});
      need(instance && event.at("context").is_object(),
           "feedback trigger requires selected provider and context object");
      Json call = {
          {"protocol", "symphony.sbv.account-feedback-call.v1"},
          {"occurrence",
           {{"stream_id", stream.id},
            {"event_id", event.at("event_id")},
            {"source_ordinal", dec(stream.ordinal)}}},
          {"time_ns", event.at("time_ns")},
          {"context", event.at("context")},
          {"state",
           {{"cash", balances(state.cash)},
            {"positions", balances(state.positions)},
            {"marks", balances(state.marks)},
            {"basis", balances(state.basis)},
            {"time_ns", state.time ? Json(dec(*state.time)) : Json(nullptr)}}},
          {"memory", state.memory},
          {"previous_event", state.previous}};
      auto reply = instance->decide(call, end);
      if (reply.status == SBV_PROVIDER_CANCELLED)
        throw e::Error("sbv.cancelled", "feedback provider cancelled", 2);
      need(reply.status == SBV_PROVIDER_OK ||
               (reply.status == SBV_PROVIDER_UNAVAILABLE &&
                p.at("feedback").at("on_unavailable") == "skip_trigger"),
           "feedback provider failed or unavailable");
      feedback_record = {
          {"call", call},
          {"status", dec(static_cast<std::uint64_t>(reply.status))},
          {"decision", reply.value},
          {"error", reply.error}};
      provider_missing = reply.status == SBV_PROVIDER_UNAVAILABLE;
      if (!provider_missing) {
        keys(reply.value, {"protocol", "action", "memory", "extensions"});
        need(reply.value.at("protocol") ==
                     "symphony.sbv.feedback-decision.v1" &&
                 reply.value.at("memory").is_object() &&
                 reply.value.at("extensions").is_object(),
             "invalid feedback decision");
        state.memory = reply.value.at("memory");
        no_action = reply.value.at("action").is_null();
        if (!no_action) {
          event = reply.value.at("action");
          need(event.is_object() &&
                   event.at("event_id") == stream.row.at("event_id") &&
                   event.at("time_ns") == stream.row.at("time_ns") &&
                   event.at("kind") != "feedback_trigger",
               "feedback action identity/time or recursive trigger invalid");
        }
      }
    }
    auto kind = str(event.at("kind"));
    State next = state;
    Json cash_delta = Json::object(), position_delta = Json::object();
    auto cash = [&](const std::string &unit, w::R delta) {
      need(next.cash.contains(unit), "unknown event cash unit");
      next.cash.at(unit) = w::plus(next.cash.at(unit), delta);
      cash_delta[unit] = w::wire(w::plus(
          cash_delta.contains(unit) ? w::artifact(cash_delta.at(unit)) : w::R{},
          delta));
    };
    auto trade = [&](const std::string &id, w::R quantity, w::R price,
                     w::R fee) {
      const auto &instrument = instruments.at(id);
      auto old = next.positions.at(id), current = w::plus(old, quantity);
      if (!instrument.variation) {
        auto cost = w::plus(
            w::times(w::times(quantity, price), instrument.multiplier), fee);
        cash(instrument.unit, {-cost.n, cost.d});
      } else {
        auto &basis = next.basis.at(id);
        w::R realized;
        if (!old.n || ((old.n > 0) == (quantity.n > 0))) {
          basis = w::divide(
              w::plus(w::times(old, basis), w::times(quantity, price)),
              current);
        } else {
          auto magnitude_old = w::R{w::absolute(old.n), old.d},
               magnitude_q = w::R{w::absolute(quantity.n), quantity.d};
          auto closed = w::compare(magnitude_old, magnitude_q) < 0
                            ? magnitude_old
                            : magnitude_q;
          if (old.n < 0)
            closed.n = -closed.n;
          realized =
              w::times(w::times(closed, w::plus(price, {-basis.n, basis.d})),
                       instrument.multiplier);
          if (!current.n)
            basis = {};
          else if ((current.n > 0) != (old.n > 0))
            basis = price;
        }
        cash(instrument.unit, w::plus(realized, {-fee.n, fee.d}));
      }
      next.positions.at(id) = current;
      const auto previous = state.positions.at(id);
      position_delta[id] = w::wire(w::plus(current, {-previous.n, previous.d}));
    };
    bool signal = false, admitted = true, execution_missing = provider_missing;
    Json execution_outcome = nullptr, resolved_quantity = nullptr;
    std::string suppression_reason;
    if (no_action || provider_missing) {
      admitted = false;
      suppression_reason =
          provider_missing ? "provider_unavailable" : "provider_no_action";
    } else if (kind == "settle") {
      keys(event, {"event_id", "time_ns", "kind", "instrument_id", "price"});
      auto id = identity(event.at("instrument_id"));
      need(instruments.contains(id) && instruments.at(id).variation,
           "settle requires linear variation instrument");
      auto price = w::artifact(event.at("price"));
      auto &basis = next.basis.at(id);
      cash(instruments.at(id).unit,
           w::times(w::times(next.positions.at(id),
                             w::plus(price, {-basis.n, basis.d})),
                    instruments.at(id).multiplier));
      basis = next.positions.at(id).n ? price : w::R{};
      next.marks[id] = price;
    } else if (kind == "liquidity_order") {
      keys(event,
           {"event_id", "time_ns", "kind", "pool_id", "quantity_rule",
            "limit_price_nanos", "time_in_force", "max_level_participation",
            "fee_fixed", "fee_per_unit", "guards", "depth_policy",
            "market_state_policy", "receive_time_policy", "on_unavailable"});
      const auto pool_id = identity(event.at("pool_id"));
      need(state.pools.contains(pool_id), "unknown liquidity pool");
      auto &pool = next.pools.at(pool_id);
      const auto &id = pool_instruments.at(pool_id);
      const auto &instrument = instruments.at(id);
      need(stream.time == pool.time,
           "snapshot order time must equal frame availability");
      need(event.at("time_in_force") == "IOC" ||
               event.at("time_in_force") == "FOK",
           "unsupported account time in force");
      need(event.at("depth_policy") == "require_sufficient" ||
               event.at("depth_policy") == "visible_only",
           "unsupported depth policy");
      need(event.at("market_state_policy") == "uncrossed_only" ||
               event.at("market_state_policy") == "allow_locked_crossed",
           "unsupported market policy");
      need(event.at("receive_time_policy") == "unflagged_only" ||
               event.at("receive_time_policy") == "allow_flagged",
           "unsupported receive time policy");
      need(event.at("on_unavailable") == "abort" ||
               event.at("on_unavailable") == "skip_event",
           "unsupported unavailable policy");
      auto participation =
          rational::read_ratio(event.at("max_level_participation"));
      need(participation.n >= 0 && participation.n <= participation.d,
           "participation outside [0,1]");
      std::optional<std::int64_t> limit;
      if (!event.at("limit_price_nanos").is_null()) {
        limit = i64(event.at("limit_price_nanos"));
        need(*limit != INT64_MAX, "undefined limit price");
      }
      auto fixed = w::artifact(event.at("fee_fixed")),
           rate = w::artifact(event.at("fee_per_unit"));
      const auto &rule = event.at("quantity_rule");
      keys(rule, {"kind", "value"});
      auto quantity = w::artifact(rule.at("value"));
      need(rule.at("kind") == "fixed" || rule.at("kind") == "target_position",
           "unsupported quantity rule");
      need(p.at("mode") != "fixed_intents" ||
               (event.at("guards").empty() && rule.at("kind") == "fixed"),
           "fixed intents cannot use state feedback");
      if (rule.at("kind") == "fixed")
        need(quantity.d == 1 && quantity.n != 0 &&
                 w::absolute(quantity.n) <= UINT64_MAX,
             "fixed snapshot quantity requires nonzero exact signed uint64 "
             "magnitude");
      admitted = guards(event.at("guards"), state);
      if (!admitted)
        suppression_reason = "guard_false";
      if (rule.at("kind") == "target_position" && admitted) {
        auto prior = state.positions.at(id);
        quantity = w::plus(quantity, {-prior.n, prior.d});
      }
      if (admitted) {
        need(quantity.d == 1 && w::absolute(quantity.n) <= UINT64_MAX,
             "snapshot quantity must be an exact signed uint64 magnitude");
        need(quantity.n != 0 || rule.at("kind") == "target_position",
             "fixed order quantity must be nonzero");
        if (!quantity.n) {
          admitted = false;
          suppression_reason = "target_reached";
        }
        resolved_quantity = w::wire(quantity);
      }
      signal = admitted;
      if (admitted) {
        auto count = static_cast<std::uint64_t>(w::absolute(quantity.n));
        const Json occurrence = {{"stream_id", stream.id},
                                 {"event_id", event.at("event_id")},
                                 {"source_ordinal", dec(stream.ordinal)}};
        depth_sweep::Order order{event,
                                 str(event.at("event_id")),
                                 e::sha256_hex(occurrence.dump()),
                                 pool.ordinal,
                                 count,
                                 pool.ordinal,
                                 stream.time,
                                 0,
                                 quantity.n > 0,
                                 event.at("time_in_force") == "FOK",
                                 limit,
                                 participation,
                                 rational::R{1, 1}};
        Json policy = {{"liquidity_mode", "shared_snapshot"},
                       {"depth_policy", event.at("depth_policy")},
                       {"market_state_policy", event.at("market_state_policy")},
                       {"receive_time_policy", event.at("receive_time_policy")},
                       {"on_unavailable", event.at("on_unavailable") == "abort"
                                              ? "reject"
                                              : "unavailable"}};
        execution_outcome =
            depth_sweep::simulate(order, &pool, policy, "", end);
        execution_outcome["conditioning"] =
            "prior_account_state_and_committed_pool_remaining";
        execution_missing = execution_outcome.at("status") == "unavailable";
        if (!execution_missing) {
          const auto &data = execution_outcome.at("data");
          auto filled = u64(data.at("filled_quantity"));
          if (filled) {
            for (const auto &fill : data.at("fills")) {
              auto q = static_cast<w::I>(u64(fill.at("quantity")));
              trade(id, {quantity.n > 0 ? q : -q, 1},
                    {i64(fill.at("price_nanos")), 1000000000}, {});
            }
            auto fee =
                w::plus(fixed, w::times({static_cast<w::I>(filled), 1}, rate));
            cash(instrument.unit, {-fee.n, fee.d});
          }
        }
      }
    } else if (kind == "position_split") {
      keys(event, {"event_id", "time_ns", "kind", "instrument_id", "factor"});
      auto id = identity(event.at("instrument_id"));
      need(instruments.contains(id), "unknown split instrument");
      auto factor = w::artifact(event.at("factor"));
      need(factor.n > 0, "positive split factor required");
      for (const auto &[pool_id, instrument_id] : pool_instruments) {
        (void)pool_id;
        need(instrument_id != id, "split requires a separate account branch "
                                  "with explicitly adjusted liquidity pools");
      }
      auto previous = next.positions.at(id);
      next.positions.at(id) = w::times(previous, factor);
      position_delta[id] =
          w::wire(w::plus(next.positions.at(id), {-previous.n, previous.d}));
      if (next.basis.contains(id))
        next.basis.at(id) = w::divide(next.basis.at(id), factor);
      if (next.marks.contains(id))
        next.marks.at(id) = w::divide(next.marks.at(id), factor);
    } else if (kind == "cashflow") {
      keys(event, {"event_id", "time_ns", "kind", "unit", "amount"});
      cash(identity(event.at("unit")), w::artifact(event.at("amount")));
    } else if (kind == "fx") {
      keys(event, {"event_id", "time_ns", "kind", "from_unit", "to_unit",
                   "from_amount", "to_amount"});
      auto from = identity(event.at("from_unit")),
           to = identity(event.at("to_unit"));
      auto debit = w::artifact(event.at("from_amount")),
           credit = w::artifact(event.at("to_amount"));
      need(from != to && debit.n > 0 && credit.n > 0,
           "explicit positive FX legs in distinct units required");
      cash(from, {-debit.n, debit.d});
      cash(to, credit);
    } else if (kind == "mark" || kind == "fill_intent") {
      if (kind == "mark")
        keys(event, {"event_id", "time_ns", "kind", "instrument_id", "price"});
      else
        keys(event, {"event_id", "time_ns", "kind", "instrument_id", "price",
                     "quantity", "fee", "guards"});
      auto id = identity(event.at("instrument_id"));
      need(instruments.contains(id), "unknown event instrument");
      auto price = w::artifact(event.at("price"));
      if (kind == "mark")
        next.marks[id] = price;
      else {
        auto quantity = w::artifact(event.at("quantity")),
             fee = w::artifact(event.at("fee"));
        need(quantity.n != 0, "fill intent requires nonzero signed quantity");
        need(p.at("mode") != "fixed_intents" || event.at("guards").empty(),
             "fixed intents cannot contain feedback guards");
        admitted = guards(event.at("guards"), state);
        signal = admitted;
        if (admitted) {
          trade(id, quantity, price, fee);
        }
      }
    } else
      need(false, "unsupported account event kind");
    auto breached =
        admitted && !execution_missing ? violations(next) : Json::array();
    if (!breached.empty() && p.at("on_constraint") == "abort")
      need(false, "account event violates caller-selected bounds");
    std::string disposition = execution_missing   ? "execution_unavailable"
                              : !admitted         ? "guard_suppressed"
                              : !breached.empty() ? "constraint_rejected"
                                                  : "applied";
    const Json occurrence = {{"stream_id", stream.id},
                             {"event_id", event.at("event_id")},
                             {"source_ordinal", dec(stream.ordinal)}};
    if (signal)
      signals.append(Json{{"protocol", "symphony.sbv.account-intent.v1"},
                          {"signal_id", e::sha256_hex(occurrence.dump())},
                          {"occurrence", occurrence},
                          {"time_ns", dec(stream.time)},
                          {"mode", p.at("mode")},
                          {"intent", event}});
    Json before = {{"cash", balances(state.cash)},
                   {"positions", balances(state.positions)},
                   {"marks", balances(state.marks)},
                   {"basis", balances(state.basis)}};
    if (disposition == "applied") {
      state = std::move(next);
      ++applied;
    } else {
      cash_delta = Json::object();
      position_delta = Json::object();
      if (execution_missing)
        ++unavailable;
      else if (admitted)
        ++rejected;
      else
        ++suppressed;
    }
    if (!execution_outcome.is_null()) {
      execution_outcome["account_disposition"] = disposition;
      execution_outcome["committed"] = disposition == "applied";
    }
    state.time = stream.time;
    Json row{{"ordinal", dec(ledger.size())},
             {"occurrence", occurrence},
             {"event", stream.row},
             {"resolved_event", event},
             {"feedback", feedback_record},
             {"disposition", disposition},
             {"constraint_violations", breached},
             {"suppression_reason", suppression_reason},
             {"resolved_quantity", resolved_quantity},
             {"execution_outcome", execution_outcome},
             {"cash_delta", cash_delta},
             {"position_delta", position_delta},
             {"before", before},
             {"after",
              {{"cash", balances(state.cash)},
               {"positions", balances(state.positions)},
               {"marks", balances(state.marks)},
               {"basis", balances(state.basis)}}}};
    if (state.feedback)
      state.previous = {{"event", event},
                        {"disposition", disposition},
                        {"cash_delta", cash_delta},
                        {"position_delta", position_delta},
                        {"execution_outcome", execution_outcome}};
    ledger.append(row);
    if (p.at("retain_snapshots").get<bool>())
      snapshots.append(snapshot(state, model_hash));
    auto previous = stream.time;
    if (++stream.ordinal < stream.input.value.size()) {
      load(stream, previous, end);
      queue.push(index);
    }
  }
  Json equity = Json::object();
  for (const auto &[unit, cash] : state.cash) {
    auto value = cash;
    Json absent = Json::array();
    for (const auto &[id, instrument] : instruments) {
      if (instrument.unit != unit || state.positions.at(id).n == 0)
        continue;
      if (!state.marks.contains(id))
        absent.push_back(id);
      else
        value = w::plus(value,
                        w::times(w::times(state.positions.at(id),
                                          (instrument.variation
                                               ? w::plus(state.marks.at(id),
                                                         {-state.basis.at(id).n,
                                                          state.basis.at(id).d})
                                               : state.marks.at(id))),
                                 instrument.multiplier));
    }
    equity[unit] = {{"status", absent.empty() ? "available" : "unavailable"},
                    {"missing_marks", absent},
                    {"value", absent.empty() ? w::wire(value) : Json(nullptr)}};
  }
  auto result = base("native exact supplied-event account simulation; no "
                     "execution calibration");
  if (unavailable)
    result["status"] = "partial";
  auto choices = p;
  choices.erase("output");
  choices.erase("output_path");
  auto &s = result["sections"];
  s["summary"] = section({{"events", dec(ledger.size())},
                          {"signals", dec(signals.size())},
                          {"applied", dec(applied)},
                          {"constraint_rejected", dec(rejected)},
                          {"guard_suppressed", dec(suppressed)},
                          {"execution_unavailable", dec(unavailable)},
                          {"terminal_equity_by_unit", equity}});
  s["choices"] = section(choices);
  s["liquidity_inputs"] = section(pool_evidence);
  if (provider)
    s["feedback_provider"] = section(provider->evidence());
  s["initial_state"] = section(initial_snapshot);
  s["terminal_state"] = section(snapshot(state, model_hash));
  s["provenance"] =
      section({{"engine_version", version},
               {"mode", p.at("mode")},
               {"model_sha256", model_hash},
               {"execution_evidence",
                "supplied fills and/or snapshot-conditioned depth sweeps; "
                "no calibrated likelihood or market response"},
               {"census_lineage",
                "new account simulation; no mutation of referenced inputs"}});
  s["resources"] = section(
      {{"backend", "cpu"},
       {"workers", "1"},
       {"ordering", "ascending time, caller stream order, source ordinal"},
       {"resident_state", "account balances, settlement basis, provider memory "
                          "and selected depth pools, one "
                          "event per stream, per-stream identity sets; no "
                          "policy event limit or automatic spill"}});
  s["user_extensions"] = section(p.at("extensions"));
  auto out =
      logical::Value(s)
          .with("execution", consumer::section(ledger.close()))
          .with("signals", consumer::section(signals.close()))
          .with("snapshots",
                consumer::section(snapshots.close(),
                                  p.at("retain_snapshots").get<bool>()
                                      ? "available"
                                      : "not_selected",
                                  p.at("retain_snapshots").get<bool>()
                                      ? ""
                                      : "snapshot retention not selected"));
  std::vector<logical::Value> inputs;
  for (const auto &stream : streams)
    inputs.push_back(logical::Value::object(
        {{"stream_id", logical::Value(stream.id)},
         {"events", stream.input.value},
         {"evidence", stream.input.evidence ? *stream.input.evidence
                                            : logical::Value(Json(nullptr))}}));
  out = out.with("event_inputs",
                 consumer::section(logical::Value::array(std::move(inputs))));
  if (parent)
    out = out.with("branch_parent",
                   consumer::section(logical::Value(parent->reference)));
  return logical::Value(result).with("sections", std::move(out));
}
} // namespace
Json account_simulate(const Json &p, std::int64_t end) {
  need(p.contains("output") != p.contains("output_path"),
       "select exactly one account output");
  if (p.contains("output"))
    return partitioned_result(
        p, "account-simulate", end,
        [&](PartitionedOutput &context) { return body(p, &context, end); });
  return persist(
      body(p, nullptr, end).materialize({artifact_bytes, artifact_values}), p,
      "account-simulate", end);
}
} // namespace symphony::sbv::detail
