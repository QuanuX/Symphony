#include "detail.hpp"
#include <deque>
#include <symphony/sbv/interop.hpp>
namespace symphony::sbv::detail {
Json backend_plan(const Json &p, std::int64_t end) try {
  deadline(end);
  keys(p,
       {"protocol", "backend", "fallback", "limit_policy", "scheduling",
        "workers", "memory_bytes", "declared_limits", "buffer", "extensions"});
  keys(p.at("declared_limits"), {"workers", "memory_bytes"});
  need(p.at("extensions").is_object(), "extensions object required");
  namespace i = interop;
  i::BackendRequest r{str(p.at("backend")),
                      p.at("fallback").is_null() ? "" : str(p.at("fallback")),
                      str(p.at("limit_policy")),
                      str(p.at("scheduling")),
                      u64(p.at("workers")),
                      u64(p.at("memory_bytes")),
                      u64(p.at("declared_limits").at("workers")),
                      u64(p.at("declared_limits").at("memory_bytes"))};
  auto plan = i::plan_backend(r);
  Json buffer = missing("no buffer selected");
  if (!p.at("buffer").is_null()) {
    const auto &b = p.at("buffer");
    keys(b, {"dtype", "device", "owner_id", "stream_id", "storage_bytes",
             "byte_offset", "shape", "byte_strides", "completion"});
    need(b.at("shape").is_array() && b.at("byte_strides").is_array() &&
             b.at("shape").size() <= 16 && b.at("byte_strides").size() <= 16,
         "buffer dimension bound");
    i::BufferDescriptor d{str(b.at("dtype")),
                          str(b.at("device")),
                          str(b.at("owner_id")),
                          str(b.at("stream_id")),
                          u64(b.at("storage_bytes")),
                          u64(b.at("byte_offset")),
                          {},
                          {}};
    for (const auto &v : b.at("shape"))
      d.shape.push_back(u64(v));
    for (const auto &v : b.at("byte_strides"))
      d.byte_strides.push_back(u64(v));
    auto admitted = i::admit_buffer(d);
    const auto state = str(b.at("completion"));
    need(state == "pending" || state == "ready" || state == "failed" ||
             state == "cancelled",
         "completion state invalid");
    const bool usable = state == "ready" && d.device == plan.backend &&
                        plan.status == "planned" &&
                        d.storage_bytes <= plan.memory_bytes;
    buffer = {{"status", usable ? "descriptor_ready" : "unavailable"},
              {"reason",
               usable ? "descriptor consistent; foreign ownership "
                        "and completion remain caller claims"
                      : "buffer device/completion/capacity is not usable by "
                        "this plan; no implicit transfer or wait"},
              {"element_bytes", dec(admitted.element_bytes)},
              {"required_bytes", dec(admitted.required_bytes)},
              {"empty", admitted.empty}};
    if (!usable && plan.status == "planned") {
      plan.status = "unavailable";
      plan.reason = "selected buffer needs a compatible ready runtime; no "
                    "implicit device transfer";
      plan.workers = 0;
      plan.memory_bytes = 0;
    }
  }
  return {{"protocol", "symphony.sbv.backend-plan.v1"},
          {"engine_version", version},
          {"status", plan.status},
          {"reason", plan.reason},
          {"requested", p},
          {"resolved",
           {{"backend", plan.backend},
            {"workers", dec(plan.workers)},
            {"memory_bytes", dec(plan.memory_bytes)},
            {"fallback_used", plan.fallback_used},
            {"numeric_profile", "preserve selected operation"}}},
          {"buffer", buffer},
          {"applied", nullptr},
          {"reservation", false},
          {"contract", i::contract_version},
          {"capabilities",
           {{"cpu", true},
            {"cuda_runtime", false},
            {"tensor_runtime", false},
            {"adapter_completion", "ready/pending/failed/cancelled; explicit "
                                   "wait and cancellation callback"}}}};
} catch (const std::invalid_argument &x) {
  throw e::Error("sbv.contract", x.what(), 2);
}
Json live_plan(const Json &p, std::int64_t end) {
  keys(p, {"protocol", "pre_ns", "post_ns", "event_cap", "byte_cap",
           "pending_cap", "overflow", "pending_overflow", "observed_until_ns",
           "events", "extensions"});
  need(p.at("extensions").is_object(), "extensions object required");
  const auto pre = u64(p.at("pre_ns")), post = u64(p.at("post_ns")),
             event_cap = u64(p.at("event_cap")),
             byte_cap = u64(p.at("byte_cap")),
             pending_cap = u64(p.at("pending_cap")),
             until = u64(p.at("observed_until_ns"));
  need(event_cap > 0 && event_cap <= 65536 && byte_cap > 0 && pending_cap > 0 &&
           pending_cap <= 128 && until != UINT64_MAX,
       "offline plan bounds");
  auto overflow = str(p.at("overflow")),
       pending_policy = str(p.at("pending_overflow"));
  need(overflow == "reject" || overflow == "drop_oldest" ||
           overflow == "drop_newest",
       "overflow policy invalid");
  need(pending_policy == "reject" || pending_policy == "skip_signal",
       "pending policy invalid");
  const auto &events = p.at("events");
  need(events.is_array() && events.size() <= 512,
       "offline event fixture bound");
  struct Event {
    std::uint64_t time, bytes;
    bool signal;
  };
  std::vector<Event> input;
  std::uint64_t signals = 0;
  for (const auto &row : events) {
    keys(row, {"available_ns", "bytes", "signal"});
    auto time = u64(row.at("available_ns")), bytes = u64(row.at("bytes"));
    need(row.at("signal").is_boolean() && bytes > 0 && bytes <= 1000000000 &&
             time != UINT64_MAX && time <= until &&
             (input.empty() || time >= input.back().time),
         "invalid offline event");
    input.push_back({time, bytes, row.at("signal").get<bool>()});
    if (input.back().signal)
      ++signals;
  }
  need(signals <= 128, "offline signal fixture bound");
  struct Window {
    std::size_t signal;
    std::uint64_t start, finish;
  };
  std::vector<Window> pending;
  std::deque<std::size_t> ring;
  Json windows = Json::array(), dropped = Json::array(),
       skipped = Json::array();
  std::uint64_t bytes = 0, peak_bytes = 0, peak_events = 0, peak_pending = 0;
  auto close = [&](std::uint64_t time, bool inclusive) {
    for (auto it = pending.begin(); it != pending.end();) {
      if (it->finish > time || (!inclusive && it->finish == time)) {
        ++it;
        continue;
      }
      Json retained = Json::array();
      std::uint64_t expected = 0;
      for (const auto &e : input)
        if (e.time >= it->start && e.time <= it->finish)
          ++expected;
      for (auto idx : ring)
        if (input[idx].time >= it->start && input[idx].time <= it->finish)
          retained.push_back(dec(idx));
      windows.push_back(
          {{"signal_index", dec(it->signal)},
           {"start_ns", dec(it->start)},
           {"end_ns", dec(it->finish)},
           {"status",
            retained.size() == expected ? "fixture_complete" : "fixture_loss"},
           {"fixture_expected_events", dec(expected)},
           {"retained_event_count", dec(retained.size())},
           {"market_completeness", "not_established"}});
      it = pending.erase(it);
    }
  };
  for (std::size_t idx = 0; idx < input.size(); ++idx) {
    deadline(end);
    const auto &e = input[idx];
    close(e.time, false);
    auto earliest = e.time >= pre ? e.time - pre : 0;
    for (const auto &w : pending)
      earliest = std::min(earliest, w.start);
    while (!ring.empty() && input[ring.front()].time < earliest) {
      bytes -= input[ring.front()].bytes;
      ring.pop_front();
    }
    bool retain = true;
    auto fits = [&] {
      return ring.size() < event_cap && e.bytes <= byte_cap - bytes;
    };
    if (!fits()) {
      need(overflow != "reject", "offline capture capacity exceeded");
      if (overflow == "drop_oldest")
        while (!ring.empty() && !fits()) {
          dropped.push_back(
              {{"event_index", dec(ring.front())}, {"reason", "drop_oldest"}});
          bytes -= input[ring.front()].bytes;
          ring.pop_front();
        }
      if (!fits()) {
        retain = false;
        dropped.push_back(
            {{"event_index", dec(idx)},
             {"reason", e.bytes > byte_cap ? "event_larger_than_capacity"
                                           : "drop_newest"}});
      }
    }
    if (retain) {
      ring.push_back(idx);
      bytes += e.bytes;
    }
    peak_bytes = std::max(peak_bytes, bytes);
    peak_events =
        std::max(peak_events, static_cast<std::uint64_t>(ring.size()));
    if (e.signal) {
      if (pending.size() == pending_cap) {
        need(pending_policy != "reject",
             "offline pending-window capacity exceeded");
        skipped.push_back(dec(idx));
      } else {
        need(post <= UINT64_MAX - e.time, "offline horizon overflow");
        pending.push_back(
            {idx, e.time >= pre ? e.time - pre : 0, e.time + post});
        peak_pending =
            std::max(peak_pending, static_cast<std::uint64_t>(pending.size()));
      }
    }
  }
  close(until, true);
  for (const auto &w : pending)
    windows.push_back({{"signal_index", dec(w.signal)},
                       {"start_ns", dec(w.start)},
                       {"end_ns", dec(w.finish)},
                       {"status", "pending"},
                       {"fixture_expected_events", nullptr},
                       {"retained_event_count", nullptr},
                       {"market_completeness", "not_established"}});
  std::stable_sort(
      windows.begin(), windows.end(), [](const Json &a, const Json &b) {
        return u64(a.at("signal_index")) < u64(b.at("signal_index"));
      });
  return {{"protocol", "symphony.sbv.live-plan.v1"},
          {"engine_version", version},
          {"can_activate", false},
          {"mode", "disconnected_offline_simulation"},
          {"requested", p},
          {"windows", windows},
          {"dropped_events", dropped},
          {"skipped_signal_indices", skipped},
          {"peak",
           {{"events", dec(peak_events)},
            {"bytes", dec(peak_bytes)},
            {"pending_windows", dec(peak_pending)}}},
          {"provider_requests", "0"},
          {"limitations",
           Json::array({"Caller fixture and closed-through watermark are "
                        "assumptions; no provider completeness is inferred.",
                        "No subscription, socket, entitlement probe, spill "
                        "implementation or live adapter exists.",
                        "Expiry frees only events outside pre-trigger and "
                        "pending windows; overload can cause explicit loss.",
                        "Event indices distinguish equal-time events; window "
                        "time boundaries are inclusive."})}};
}
} // namespace symphony::sbv::detail
