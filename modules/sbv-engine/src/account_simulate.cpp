#include "consumer_source.hpp"
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
  Balances cash, positions, marks;
  std::optional<std::uint64_t> time;
};
Json snapshot(const State &s, const std::string &model) {
  return {{"protocol", "symphony.sbv.account-state.v1"},
          {"model_sha256", model},
          {"cash", balances(s.cash)},
          {"positions", balances(s.positions)},
          {"marks", balances(s.marks)},
          {"time_ns", s.time ? Json(dec(*s.time)) : Json(nullptr)}};
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
                {"output", "output_path"});
  need(p.at("extensions").is_object() && p.at("retain_snapshots").is_boolean(),
       "account extensions/retention invalid");
  need(p.at("mode") == "fixed_intents" || p.at("mode") == "stateful_feedback",
       "explicit account mode required");
  need(p.at("ordering") == "time_stream_order", "account ordering unsupported");
  need(p.at("on_constraint") == "reject_event" ||
           p.at("on_constraint") == "abort",
       "account constraint disposition required");
  State state;
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
             v.at("settlement") == "cash_notional",
         "unsupported account instrument economics");
    need(instruments.emplace(id, Instrument{unit, multiplier}).second,
         "duplicate account instrument");
    state.positions.emplace(id, w::R{});
  }
  const Json model = {{"cash_units", p.at("cash_units")},
                      {"instruments", p.at("instruments")}};
  const auto model_hash = e::sha256_hex(model.dump());
  Json initial = p.at("initial_state");
  std::optional<consumer::ValueSource> parent;
  if (initial.contains("kind")) {
    keys(initial, {"kind", "source"});
    need(initial.at("kind") == "reference", "unknown initial state kind");
    parent = consumer::value_source(initial.at("source"), end);
    initial = parent->value.materialize();
    keys(initial,
         {"protocol", "model_sha256", "cash", "positions", "marks", "time_ns"});
    need(initial.at("protocol") == "symphony.sbv.account-state.v1" &&
             initial.at("model_sha256") == model_hash,
         "account snapshot model mismatch");
    if (!initial.at("time_ns").is_null())
      state.time = u64(initial.at("time_ns"));
  } else
    keys(initial, {"cash", "positions", "marks", "time_ns"});
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
  std::uint64_t applied = 0, rejected = 0, suppressed = 0;
  while (!queue.empty()) {
    deadline(end);
    auto index = queue.top();
    queue.pop();
    auto &stream = streams[index];
    const auto &event = stream.row;
    auto kind = str(event.at("kind"));
    State next = state;
    Json cash_delta = Json::object(), position_delta = Json::object();
    auto cash = [&](const std::string &unit, w::R delta) {
      need(next.cash.contains(unit), "unknown event cash unit");
      next.cash.at(unit) = w::plus(next.cash.at(unit), delta);
      cash_delta[unit] = w::wire(delta);
    };
    bool signal = false, admitted = true;
    if (kind == "cashflow") {
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
        const auto &instrument = instruments.at(id);
        auto quantity = w::artifact(event.at("quantity")),
             fee = w::artifact(event.at("fee"));
        need(quantity.n != 0, "fill intent requires nonzero signed quantity");
        need(p.at("mode") != "fixed_intents" || event.at("guards").empty(),
             "fixed intents cannot contain feedback guards");
        admitted = guards(event.at("guards"), state);
        signal = admitted;
        if (admitted) {
          auto cost = w::plus(
              w::times(w::times(quantity, price), instrument.multiplier), fee);
          cash(instrument.unit, {-cost.n, cost.d});
          next.positions.at(id) = w::plus(next.positions.at(id), quantity);
          position_delta[id] = w::wire(quantity);
        }
      }
    } else
      need(false, "unsupported account event kind");
    auto breached = admitted ? violations(next) : Json::array();
    if (!breached.empty() && p.at("on_constraint") == "abort")
      need(false, "account event violates caller-selected bounds");
    std::string disposition = !admitted           ? "guard_suppressed"
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
                   {"positions", balances(state.positions)}};
    if (disposition == "applied") {
      state = std::move(next);
      ++applied;
    } else {
      cash_delta = Json::object();
      position_delta = Json::object();
      if (admitted)
        ++rejected;
      else
        ++suppressed;
    }
    state.time = stream.time;
    ledger.append(Json{{"ordinal", dec(ledger.size())},
                       {"occurrence", occurrence},
                       {"event", event},
                       {"disposition", disposition},
                       {"constraint_violations", breached},
                       {"cash_delta", cash_delta},
                       {"position_delta", position_delta},
                       {"before", before},
                       {"after",
                        {{"cash", balances(state.cash)},
                         {"positions", balances(state.positions)}}}});
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
        value = w::plus(value, w::times(w::times(state.positions.at(id),
                                                 state.marks.at(id)),
                                        instrument.multiplier));
    }
    equity[unit] = {{"status", absent.empty() ? "available" : "unavailable"},
                    {"missing_marks", absent},
                    {"value", absent.empty() ? w::wire(value) : Json(nullptr)}};
  }
  auto result = base("native exact supplied-event account simulation; no "
                     "execution calibration");
  auto choices = p;
  choices.erase("output");
  choices.erase("output_path");
  auto &s = result["sections"];
  s["summary"] = section({{"events", dec(ledger.size())},
                          {"signals", dec(signals.size())},
                          {"applied", dec(applied)},
                          {"constraint_rejected", dec(rejected)},
                          {"guard_suppressed", dec(suppressed)},
                          {"terminal_equity_by_unit", equity}});
  s["choices"] = section(choices);
  s["initial_state"] = section(initial_snapshot);
  s["terminal_state"] = section(snapshot(state, model_hash));
  s["provenance"] = section(
      {{"engine_version", version},
       {"mode", p.at("mode")},
       {"model_sha256", model_hash},
       {"execution_evidence", "caller-supplied conditional fills; matching and "
                              "market feasibility not verified"},
       {"census_lineage",
        "new account simulation; no mutation of referenced inputs"}});
  s["resources"] = section(
      {{"backend", "cpu"},
       {"workers", "1"},
       {"ordering", "ascending time, caller stream order, source ordinal"},
       {"resident_state",
        "account balances, one event per stream, per-stream identity sets; no "
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
