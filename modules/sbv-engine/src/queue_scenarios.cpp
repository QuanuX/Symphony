#include "consumer_source.hpp"
#include "resource_admission.hpp"
#include "wide_rational.hpp"
#include <atomic>
#include <mutex>
#include <optional>

namespace symphony::sbv::detail {
namespace {
namespace w = wide_rational;
using U = std::uint64_t;
U add(U a, U b) {
  need(b <= UINT64_MAX - a,
       "queue quantity/time exceeds uint64 representation");
  return a + b;
}
w::R probability(const Json &j) {
  keys(j, {"numerator", "denominator"});
  auto p = w::reduce(
      {w::integer(j.at("numerator")), w::integer(j.at("denominator"))});
  need(p.n >= 0 && p.n <= p.d, "queue probability/fraction outside [0,1]");
  return p;
}
U fraction(U q, w::R r) {
  auto v = w::times({q, 1}, r);
  return static_cast<U>(v.n / v.d);
}
std::string identity(const Json &j) {
  auto s = str(j);
  need(!s.empty(), "nonempty queue identity required");
  return s;
}
struct Event {
  U ordinal, time, size, flags, action, side;
  std::int64_t price;
};
struct Order {
  Json choice;
  std::string id, signal;
  U ordinal, time, quantity, horizon;
  std::int64_t price;
  bool buy;
  std::optional<U> ahead;
  w::R activation;
};
struct Atom {
  std::string id;
  U delay;
  w::R weight;
};
struct Model {
  bool fifo, any_side, same_side, cancel_reset, ignore_modify, allow_flagged;
  U service_action;
  w::R cancel, replenish;
};
Json scenario(const Order &o, const Atom &a, const Model &m,
              const std::vector<Event> &events, const std::map<U, Json> &frames,
              const std::set<U> &invalidations, bool reject, std::int64_t end) {
  deadline(end);
  const auto arrival = add(o.time, a.delay), horizon = add(o.time, o.horizon);
  Json row{{"order_id", o.id},           {"signal_id", o.signal},
           {"latency_id", a.id},         {"weight", w::wire(a.weight)},
           {"arrival_ns", dec(arrival)}, {"horizon_end_ns", dec(horizon)},
           {"status", "available"},      {"reason", ""},
           {"layer", "modeled"},         {"data", nullptr}};
  auto unavailable = [&](const std::string &reason) {
    need(!reject, reason.c_str());
    row["status"] = "unavailable";
    row["reason"] = reason;
    return row;
  };
  U ahead = 0, initial = 0, filled = 0, service = 0, served_ahead = 0,
    canceled = 0, replenished = 0, unused = 0;
  Json fills = Json::array(), first = nullptr, last = nullptr,
       frame_id = nullptr;
  std::string disposition = "horizon_expired";
  if (arrival <= horizon) {
    auto signal =
        std::lower_bound(events.begin(), events.end(), o.ordinal,
                         [](const Event &e, U n) { return e.ordinal < n; });
    if (signal == events.end() || signal->ordinal != o.ordinal ||
        signal->time != o.time)
      return unavailable(
          "signal event is not retained with matching coordinates");
    if (events.back().time < horizon)
      return unavailable("retained replay does not cover the selected horizon");
    auto stop =
        std::upper_bound(signal, events.end(), horizon,
                         [](U t, const Event &e) { return t < e.time; });
    // Check through the first later event as well: otherwise a missing tail
    // could masquerade as an interval with no events.
    auto coverage_stop = stop == events.end() ? stop : stop + 1;
    U ordinal = o.ordinal;
    for (auto it = signal; it != coverage_stop; ++it) {
      deadline(end);
      if (it->ordinal != ordinal)
        return unavailable(
            "replay ordinal gap in selected information interval");
      if (it + 1 != coverage_stop)
        ordinal = add(ordinal, 1);
    }
    auto begin = std::upper_bound(
        signal, stop, arrival, [](U t, const Event &e) { return t < e.time; });
    need(begin != signal, "arrival must follow the signal event");
    const auto &at = *(begin - 1);
    const auto found = frames.find(at.ordinal);
    if (found == frames.end())
      return unavailable(
          "arrival requires a retained frame after the last same-time event");
    const auto &f = found->second;
    if (f.at("status") != "available" || f.at("event_boundary") != true)
      return unavailable("arrival frame lacks initialized complete book state");
    if (u64(f.at("available_ns")) != at.time)
      return unavailable("arrival frame and event timestamp differ");
    if (!m.allow_flagged && (f.at("receive_time_flagged") == true))
      return unavailable("arrival receive timestamp flagged");
    const auto &depth = f.at("data");
    const auto &own = depth.at(o.buy ? "bids" : "asks");
    const auto &other = depth.at(o.buy ? "asks" : "bids");
    if (!other.empty() && (o.buy ? o.price >= i64(other[0].at("price_nanos"))
                                 : o.price <= i64(other[0].at("price_nanos"))))
      return unavailable("selected resting limit crosses the arrival book; "
                         "select a sweep model");
    if (o.ahead)
      ahead = *o.ahead;
    else {
      bool present = false;
      for (const auto &l : own)
        if (i64(l.at("price_nanos")) == o.price) {
          ahead = u64(l.at("size"));
          present = true;
        }
      const auto count =
          u64(depth.at(o.buy ? "bid_level_count" : "ask_level_count"));
      if (!present && count > own.size() &&
          (own.empty() ||
           (o.buy ? o.price < i64(own.back().at("price_nanos"))
                  : o.price > i64(own.back().at("price_nanos")))))
        return unavailable(
            "arrival displayed queue lies outside retained depth");
    }
    initial = ahead;
    frame_id = dec(at.ordinal);
    const U side = o.buy ? 'B' : 'A';
    for (auto it = begin; it != stop && filled < o.quantity; ++it) {
      deadline(end);
      const auto &e = *it;
      if (invalidations.contains(e.ordinal))
        return unavailable("source book invalidated after arrival");
      if ((e.flags & (64 | 32 | 16 | 4 | 2)) ||
          (!m.allow_flagged && (e.flags & 8)))
        return unavailable("gap/snapshot/unsupported or excluded timestamp "
                           "flags after arrival");
      if (e.action == 'R') {
        if (!m.cancel_reset)
          return unavailable("book reset after arrival");
        disposition = "reset_canceled_remainder";
        break;
      }
      if (e.action != 'A' && e.action != 'C' && e.action != 'M' &&
          e.action != 'T' && e.action != 'F' && e.action != 'N')
        return unavailable("unsupported replay action after arrival");
      if (e.action == 'M' && !m.ignore_modify)
        return unavailable("modify event requires a selected queue assumption");
      if (e.price != o.price)
        continue;
      if (e.action == 'C' && e.side == side) {
        auto q = std::min(ahead, fraction(e.size, m.cancel));
        ahead -= q;
        canceled = add(canceled, q);
      } else if (e.action == 'A' && e.side == side) {
        auto q = fraction(e.size, m.replenish);
        ahead = add(ahead, q);
        replenished = add(replenished, q);
      } else if (e.action == m.service_action &&
                 (m.any_side ||
                  (m.same_side ? e.side == side
                               : e.side == (o.buy ? 'A' : 'B')))) {
        const auto remaining = o.quantity - filled;
        U fill = 0, before = ahead, advance = 0;
        if (m.fifo) {
          advance = std::min(ahead, e.size);
          fill = std::min(remaining, e.size - advance);
        } else {
          // Widen the denominator before adding; both quantities may span
          // uint64.
          const auto total = static_cast<w::I>(ahead) + remaining;
          auto share = w::times({e.size, 1}, {remaining, total});
          fill = static_cast<U>(share.n / share.d);
          fill = std::min(fill, remaining);
          advance = std::min(ahead, e.size - fill);
        }
        ahead -= advance;
        served_ahead = add(served_ahead, advance);
        service = add(service, e.size);
        unused = add(unused, e.size - advance - fill);
        filled += fill;
        if (fill) {
          if (first.is_null())
            first = dec(e.time);
          last = dec(e.time);
          fills.push_back({{"source_ordinal", dec(e.ordinal)},
                           {"available_ns", dec(e.time)},
                           {"quantity", dec(fill)},
                           {"price_nanos", dec(o.price)},
                           {"queue_before", dec(before)},
                           {"queue_after", dec(ahead)},
                           {"service_quantity", dec(e.size)}});
        }
      }
    }
  } else
    disposition = "arrival_after_horizon";
  // Independent integer identities make discarded/duplicated volume visible.
  need(static_cast<w::I>(initial) + replenished ==
               static_cast<w::I>(ahead) + canceled + served_ahead &&
           static_cast<w::I>(service) ==
               static_cast<w::I>(filled) + served_ahead + unused,
       "queue quantity conservation failure");
  row["data"] = {
      {"requested_quantity", dec(o.quantity)},
      {"filled_quantity", dec(filled)},
      {"unfilled_quantity", dec(o.quantity - filled)},
      {"initial_queue_ahead", dec(initial)},
      {"final_queue_ahead", dec(ahead)},
      {"canceled_ahead", dec(canceled)},
      {"replenished_ahead", dec(replenished)},
      {"served_ahead", dec(served_ahead)},
      {"service_quantity", dec(service)},
      {"unused_service_quantity", dec(unused)},
      {"arrival_frame_ordinal", frame_id},
      {"first_fill_ns", first},
      {"last_fill_ns", last},
      {"fills", fills},
      {"disposition", filled == o.quantity ? "full_fill" : disposition},
      {"coverage", "retained_observed_span_only"}};
  return row;
}
} // namespace

static logical::Value queue_body(const Json &p, PartitionedOutput *context,
                                 std::int64_t end) {
  keys_optional(p,
                {"protocol", "orders", "latency_atoms", "model",
                 "execution_labels", "on_unavailable", "workers", "extensions"},
                {"path", "expected_sha256", "source", "output_path", "output"});
  need(p.at("on_unavailable") == "unavailable" ||
           p.at("on_unavailable") == "reject",
       "queue unavailable policy required");
  need(p.at("extensions").is_object(), "queue extensions object required");
  const auto workers = u64(p.at("workers"));
  need(workers > 0, "positive queue worker count required");
  const auto &config = p.at("model");
  keys(config, {"id", "version", "priority", "service_action", "service_side",
                "cancel_ahead_fraction", "replenish_ahead_fraction",
                "on_modify", "on_reset", "receive_time_policy"});
  need(config.at("id") == "queue_replay" && config.at("version") == "1",
       "unsupported queue model/version");
  need(config.at("priority") == "fifo" || config.at("priority") == "pro_rata",
       "queue priority invalid");
  need(config.at("service_action") == "T" || config.at("service_action") == "F",
       "select exactly one service action");
  need(config.at("service_side") == "same" ||
           config.at("service_side") == "opposite" ||
           config.at("service_side") == "any",
       "service side invalid");
  need(config.at("on_modify") == "ignore" ||
           config.at("on_modify") == "unavailable",
       "modify policy invalid");
  need(config.at("on_reset") == "cancel_remaining" ||
           config.at("on_reset") == "unavailable",
       "reset policy invalid");
  need(config.at("receive_time_policy") == "allow_flagged" ||
           config.at("receive_time_policy") == "unflagged_only",
       "timestamp policy invalid");
  Model model{config.at("priority") == "fifo",
              config.at("service_side") == "any",
              config.at("service_side") == "same",
              config.at("on_reset") == "cancel_remaining",
              config.at("on_modify") == "ignore",
              config.at("receive_time_policy") == "allow_flagged",
              config.at("service_action") == "T" ? U('T') : U('F'),
              probability(config.at("cancel_ahead_fraction")),
              probability(config.at("replenish_ahead_fraction"))};
  const auto source = consumer::source(p, end).value;
  const auto s = source.at("sections");
  need(s.at("provenance").at("data").at("profile").materialize() ==
               "databento_mbo_orders_strict_v1" &&
           s.at("summary").at("data").at("closed_census").materialize() == true,
       "source-bound book result required");
  std::map<std::string, Json> signals;
  auto signal_rows = s.at("signals").at("data").children();
  while (auto v = signal_rows.next()) {
    deadline(end);
    auto j = v->value.materialize();
    need(signals.emplace(identity(j.at("signal_id")), j).second,
         "duplicate census signal");
  }
  std::vector<Event> events;
  auto replay_rows = s.at("replay").at("data").at("events").children();
  while (auto v = replay_rows.next()) {
    deadline(end);
    auto j = v->value.materialize();
    Event e{u64(j.at("source_ordinal")), u64(j.at("ts_recv")),
            u64(j.at("size")),           u64(j.at("flags")),
            u64(j.at("action")),         u64(j.at("side")),
            i64(j.at("price_nanos"))};
    need(e.time != UINT64_MAX && e.flags <= 255 && e.action <= 255 &&
             e.side <= 255 &&
             (events.empty() || (e.ordinal > events.back().ordinal &&
                                 e.time >= events.back().time)),
         "invalid replay event ordering/domain");
    events.push_back(e);
  }
  std::map<U, Json> frames;
  std::set<U> invalidations;
  const auto diagnostics = s.at("diagnostics").at("data");
  if (diagnostics.kind() == "object" && diagnostics.contains("events")) {
    auto records = diagnostics.at("events").children();
    while (auto v = records.next()) {
      deadline(end);
      if (v->value.at("kind").materialize() == "invalidation")
        invalidations.insert(u64(v->value.at("source_ordinal").materialize()));
    }
  }
  auto frame_rows = s.at("book_frames").at("data").children();
  while (auto v = frame_rows.next()) {
    deadline(end);
    auto j = v->value.materialize();
    need(j.at("protocol") == "symphony.sbv.book-frame.v1" &&
             j.at("layer") == "reconstructed",
         "typed reconstructed frame required");
    need((j.at("status") == "available" || j.at("status") == "unavailable") &&
             j.at("event_boundary").is_boolean() &&
             j.at("receive_time_flagged").is_boolean(),
         "invalid frame status/flags");
    if (j.at("status") == "available") {
      need(j.at("event_boundary") == true,
           "available frame requires event boundary");
      for (auto side : {"bids", "asks"}) {
        const auto &levels = j.at("data").at(side);
        need(levels.is_array(), "frame levels array required");
        std::optional<std::int64_t> prior;
        for (const auto &level : levels) {
          auto price = i64(level.at("price_nanos"));
          need(price != INT64_MAX && u64(level.at("size")) > 0 &&
                   u64(level.at("order_count")) > 0 &&
                   (!prior || (std::string(side) == "bids" ? price < *prior
                                                           : price > *prior)),
               "invalid ordered frame level");
          prior = price;
        }
        auto count = u64(j.at("data").at(std::string(side) == "bids"
                                             ? "bid_level_count"
                                             : "ask_level_count"));
        need(count >= levels.size() && (!count || !levels.empty()),
             "frame depth counts inconsistent");
      }
    } else
      need(j.at("data").is_null(), "unavailable frame contains depth");
    need(frames.emplace(u64(j.at("source_ordinal")), j).second,
         "duplicate book frame");
  }
  auto order_input = consumer::input_rows(p.at("orders"), "queue_orders", end);
  std::vector<Order> orders;
  std::map<std::string, std::size_t> ids;
  auto order_rows = order_input.value.children();
  while (auto v = order_rows.next()) {
    deadline(end);
    auto j = v->value.materialize();
    keys(j, {"order_id", "signal_id", "side", "price_nanos", "quantity",
             "horizon_ns", "initial_queue_ahead", "activation_probability"});
    auto id = identity(j.at("order_id")), signal = identity(j.at("signal_id"));
    need(ids.emplace(id, orders.size()).second && signals.contains(signal),
         "duplicate order or unknown signal");
    need(j.at("side") == "buy" || j.at("side") == "sell", "queue side invalid");
    auto q = u64(j.at("quantity"));
    auto price = i64(j.at("price_nanos"));
    need(q > 0 && price != INT64_MAX,
         "positive quantity and defined price required");
    const auto &sig = signals.at(signal);
    orders.push_back({j, id, signal, u64(sig.at("source_ordinal")),
                      u64(sig.at("available_ns")), q, u64(j.at("horizon_ns")),
                      price, j.at("side") == "buy",
                      j.at("initial_queue_ahead").is_null()
                          ? std::optional<U>{}
                          : u64(j.at("initial_queue_ahead")),
                      probability(j.at("activation_probability"))});
  }
  auto latency_input =
      consumer::input_rows(p.at("latency_atoms"), "latency_atoms", end);
  std::vector<Atom> atoms;
  std::set<std::string> atom_ids;
  w::R mass{};
  auto latency_rows = latency_input.value.children();
  while (auto v = latency_rows.next()) {
    deadline(end);
    auto j = v->value.materialize();
    keys(j, {"id", "delay_ns", "weight"});
    auto id = identity(j.at("id"));
    auto weight = probability(j.at("weight"));
    need(atom_ids.insert(id).second, "duplicate latency atom");
    mass = w::plus(mass, weight);
    atoms.push_back({id, u64(j.at("delay_ns")), weight});
  }
  need(!atoms.empty() && w::compare(mass, {1, 1}) == 0,
       "latency mass must equal one without normalization");
  auto applied = resource::effective(workers, orders.size());
  std::vector<std::optional<logical::Value>> batches(applied);
  std::vector<Json> summaries(orders.size());
  std::atomic<bool> failed{false};
  std::exception_ptr error;
  std::mutex mutex;
  resource::launch(applied, failed, [&](std::size_t worker) {
    try {
      consumer::Rows rows(context, "queue-worker-" + dec(worker));
      const auto base = orders.size() / applied,
                 remainder = orders.size() % applied;
      const auto begin = base * worker + std::min(worker, remainder),
                 count = base + (worker < remainder);
      for (auto i = begin; i < begin + count && !failed; ++i) {
        const auto &o = orders[i];
        w::R any{}, full{}, expected{}, unavailable{};
        for (const auto &a : atoms) {
          auto row = scenario(o, a, model, events, frames, invalidations,
                              p.at("on_unavailable") == "reject", end);
          if (row.at("status") == "unavailable")
            unavailable = w::plus(unavailable, a.weight);
          else {
            auto q = u64(row.at("data").at("filled_quantity"));
            if (q)
              any = w::plus(any, a.weight);
            if (q == o.quantity)
              full = w::plus(full, a.weight);
            expected = w::plus(expected, w::times(a.weight, {q, 1}));
          }
          rows.append(row);
        }
        bool complete = unavailable.n == 0;
        auto claim = [&](w::R x) -> Json {
          return complete ? Json{{"status", "scenario_assumption"},
                                 {"value", w::wire(x)}}
                          : missing("nonzero latency mass has unavailable "
                                    "evidence; no renormalization");
        };
        summaries[i] = {
            {"order_id", o.id},
            {"signal_id", o.signal},
            {"requested_quantity", dec(o.quantity)},
            {"horizon_ns", dec(o.horizon)},
            {"unavailable_latency_mass", w::wire(unavailable)},
            {"any_fill_probability", claim(w::times(any, o.activation))},
            {"complete_fill_probability", claim(w::times(full, o.activation))},
            {"no_fill_probability",
             claim(w::plus({1, 1},
                           w::times(w::times(any, o.activation), {-1, 1})))},
            {"expected_filled_quantity",
             claim(w::times(expected, o.activation))},
            {"calibration", "not_established"}};
      }
      if (!failed)
        batches[worker] = rows.close();
    } catch (...) {
      failed = true;
      std::lock_guard lock(mutex);
      if (!error)
        error = std::current_exception();
    }
  });
  if (error)
    std::rethrow_exception(error);
  consumer::Rows rows(context, "scenarios"), summary(context, "order-summary");
  for (auto &batch : batches) {
    need(batch.has_value(), "queue worker incomplete");
    auto cursor = batch->children();
    while (auto v = cursor.next()) {
      deadline(end);
      rows.append(v->value);
    }
  }
  for (const auto &j : summaries)
    summary.append(j);

  Json assessment =
      section(nullptr, "not_selected", "execution labels not selected");
  std::optional<consumer::InputRows> labels;
  if (!p.at("execution_labels").is_null()) {
    const auto &l = p.at("execution_labels");
    keys(l, {"domain", "source_kind", "source_reference", "permission",
             "training_ids", "rows", "on_unavailable"});
    identity(l.at("domain"));
    identity(l.at("source_reference"));
    need(l.at("source_kind") == "synthetic" ||
             l.at("source_kind") == "external_execution",
         "execution label kind invalid");
    need(l.at("permission") == "declared_permitted" ||
             l.at("permission") == "unavailable",
         "label permission declaration invalid");
    need(l.at("on_unavailable") == "reject" ||
             l.at("on_unavailable") == "unavailable",
         "label policy invalid");
    need(l.at("training_ids").is_array(), "training IDs required");
    std::set<std::string> training, seen, observed_orders;
    for (const auto &id : l.at("training_ids"))
      need(training.insert(identity(id)).second, "duplicate training ID");
    labels = consumer::input_rows(l.at("rows"), "execution_labels", end);
    // Collect the full declared training domain before assessing any held-out
    // row. Input ordering cannot hide reuse through another order for a signal.
    auto training_rows = labels->value.children();
    while (auto v = training_rows.next()) {
      deadline(end);
      auto j = v->value.materialize();
      if (j.at("partition") == "training") {
        auto id = identity(j.at("label_id")),
             order_id = identity(j.at("order_id"));
        need(ids.contains(order_id), "unknown training order");
        training.insert(id);
        training.insert(order_id);
        training.insert(orders[ids.at(order_id)].signal);
      }
    }
    auto cursor = labels->value.children();
    w::R score{};
    U count = 0;
    std::string reason;
    if (l.at("permission") != "declared_permitted")
      reason = "permission unavailable";
    while (auto v = cursor.next()) {
      deadline(end);
      auto j = v->value.materialize();
      keys(j, {"label_id", "order_id", "domain", "partition", "filled_quantity",
               "horizon_ns"});
      auto id = identity(j.at("label_id")),
           order_id = identity(j.at("order_id"));
      need(seen.insert(id).second && observed_orders.insert(order_id).second &&
               ids.contains(order_id),
           "duplicate label/order or unknown order");
      const auto index = ids.at(order_id);
      const auto &o = orders[index];
      auto q = u64(j.at("filled_quantity"));
      need(q <= o.quantity, "label fill exceeds order quantity");
      need(j.at("partition") == "training" || j.at("partition") == "held_out",
           "label partition invalid");
      if (j.at("domain") != l.at("domain") ||
          u64(j.at("horizon_ns")) != o.horizon)
        reason = "label domain or horizon mismatch";
      if (j.at("partition") == "training")
        training.insert(id);
      if (j.at("partition") == "held_out") {
        if (training.contains(id) || training.contains(order_id) ||
            training.contains(o.signal))
          reason = "held-out identity appears in declared training IDs";
        const auto &pred = summaries[index].at("any_fill_probability");
        if (pred.at("status") != "scenario_assumption")
          reason = "prediction unavailable for held-out label";
        else {
          auto delta = w::plus(probability(pred.at("value")), {q ? -1 : 0, 1});
          score = w::plus(score, w::times(delta, delta));
          ++count;
        }
      }
    }
    if (!count && reason.empty())
      reason = "no held-out labels selected";
    if (!reason.empty())
      need(l.at("on_unavailable") != "reject", reason.c_str());
    assessment = section(
        {{"domain", l.at("domain")},
         {"source_kind", l.at("source_kind")},
         {"permission", l.at("permission")},
         {"source_reference", l.at("source_reference")},
         {"held_out_count", dec(count)},
         {"brier_any_fill",
          reason.empty()
              ? Json{{"status", "available"},
                     {"value", w::wire(w::divide(score, {count, 1}))}}
              : missing(reason)},
         {"calibration", "not_established"},
         {"authenticity", "external declaration; not authenticated"},
         {"scope", "selected any-fill assessment only; no learned model, "
                   "independence or deployment-domain validation"}},
        reason.empty() ? "available" : "unavailable", reason);
  }
  auto result = base("native fixed-census latency and queue scenarios");
  result["status"] = "partial";
  auto &out = result["sections"];
  out["summary"] = section(
      {{"order_count", dec(orders.size())},
       {"latency_atom_count", dec(atoms.size())},
       {"scenario_count", dec(rows.size())},
       {"closed_census", true},
       {"source_census_sha256",
        s.at("summary").at("data").at("parent_census_sha256").materialize()}});
  out["execution_label_assessment"] = assessment;
  out["resources"] = section(
      {{"backend", "cpu"},
       {"requested_workers", dec(workers)},
       {"actual_workers", dec(applied)},
       {"working_state",
        "retained event/frame and census indexes, orders and latency tape; "
        "per-worker output spools; no policy ceiling"},
       {"ordering",
        "independent orders; stable input order then latency tape order"}});
  auto choices = p;
  choices.erase("output");
  choices.erase("output_path");
  out["choices"] = section(choices);
  out["provenance"] = section(
      {{"engine_version", version},
       {"model_id", "queue_replay"},
       {"model_version", "1"},
       {"source_result_sha256", source.at("content_sha256").materialize()},
       {"source_census_sha256",
        s.at("summary").at("data").at("parent_census_sha256").materialize()},
       {"numeric_profile", "checked uint64 quantities/times and exact signed "
                           "128-bit rationals; no rounding fallback"},
       {"provider_requests", "0"},
       {"additional_spend_usd", "0"}});
  out["diagnostics"] = section(Json::array(
      {"Independent hypothetical resting orders; historical market response, "
       "hidden liquidity, venue priority and shared capital are not inferred.",
       "Arrival follows every same-receive-time event; fills use exact "
       "limit-price service events strictly after arrival through the "
       "inclusive horizon.",
       "Exactly one T/F service action is selected. C advancement and A "
       "replenishment fractions are assumptions; nonzero cancellation "
       "advancement may overlap trade-related cancellations. M ignore "
       "explicitly discards modification information.",
       "Latency atoms are an exact user-authored finite distribution/tape, not "
       "sampled or calibrated. FIFO/pro-rata and activation are scenario "
       "assumptions. Zero-weight unavailable branches do not remove positive "
       "mass.",
       "All raw source events/reconstructed frames and the fixed census remain "
       "unchanged. Modeled fills occupy a separate portable layer.",
       "Label permission, provenance and train/held-out separation are "
       "declarations; scoring never upgrades scenarios to calibrated market "
       "likelihood."}));
  out["user_extensions"] = section(p.at("extensions"));
  auto sections =
      logical::Value(out)
          .with("queue_scenarios", consumer::section(rows.close()))
          .with("execution", consumer::section(summary.close()))
          .with("order_choices", consumer::section(order_input.value))
          .with("latency_atoms", consumer::section(latency_input.value))
          .with("source_context",
                consumer::section(consumer::source_context(s)));
  for (auto name : {"signals", "book_frames", "book_checkpoint", "replay"})
    sections = sections.with(name, s.at(name));
  sections = sections.with("book_diagnostics", s.at("diagnostics"));
  if (order_input.evidence)
    sections =
        sections.with("order_input", consumer::section(*order_input.evidence));
  if (latency_input.evidence)
    sections = sections.with("latency_input",
                             consumer::section(*latency_input.evidence));
  if (labels)
    sections =
        sections.with("execution_labels", consumer::section(labels->value));
  if (labels && labels->evidence)
    sections =
        sections.with("label_input", consumer::section(*labels->evidence));
  return logical::Value(result).with("sections", std::move(sections));
}
Json queue_scenarios(const Json &p, std::int64_t end) {
  need(p.contains("output") != p.contains("output_path"),
       "select exactly one queue output");
  if (p.contains("output"))
    return partitioned_result(p, "queue-scenarios", end,
                              [&](PartitionedOutput &context) {
                                return queue_body(p, &context, end);
                              });
  return persist(queue_body(p, nullptr, end)
                     .materialize({artifact_bytes, artifact_values}),
                 p, "queue-scenarios", end);
}
} // namespace symphony::sbv::detail
