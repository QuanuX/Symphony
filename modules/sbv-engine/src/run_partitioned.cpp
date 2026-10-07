#include "partitioned_output.hpp"
#include "run_math.hpp"
#include "stream_census.hpp"
#include <atomic>
#include <mutex>
#include <thread>
namespace symphony::sbv::detail {
namespace {
using Value = logical::Value;
Value streamed_section(Value data, std::string status = "available",
                       std::string reason = "") {
  return Value::object({{"data", std::move(data)},
                        {"reason", Value(Json(std::move(reason)))},
                        {"status", Value(Json(std::move(status)))}});
}
} // namespace
Json run_partitioned(const Json &p, std::int64_t end, const Dataset *resident) {
  keys_optional(p,
                {"protocol", "output", "criteria", "replay", "execution",
                 "studies", "workers", "extensions"},
                {"memory_budget_bytes", "dataset_limits", "source_path",
                 "source_sha256", "dataset", "retained_source"});
  const auto settings = run_settings(p);
  const auto &replay = p.at("replay");
  return partitioned_result(p, "run", end, [&](PartitionedOutput &output) {
    const auto owned = resident ? nullptr : load_dataset(p, end);
    const auto &source = resident ? *resident : *owned;
    source.bind(p);
    const auto &events = source.events;
    auto signal_spool = output.spool("census-signals");
    std::optional<std::int64_t> previous;
    std::optional<U> first_available, last_available;
    U prior_signal = 0, eligible = 0;
    // The previous observed trade updates on every trade; the spacing anchor
    // updates only when a signal is admitted, including after a selected cap.
    for (std::size_t i = 0; i < events.size(); ++i) {
      if (i % 1024 == 0)
        deadline(end);
      const auto &x = events[i];
      if (!trade(x))
        continue;
      const auto &direction = settings.direction;
      const bool match =
          direction == "any" ||
          (previous && (direction == "up"     ? x.price > *previous
                        : direction == "down" ? x.price < *previous
                                              : x.price == *previous));
      if (x.size >= settings.minimum && match &&
          (signal_spool->size() == 0 ||
           x.ts_recv - prior_signal >= settings.spacing)) {
        need(eligible < UINT64_MAX, "eligible signal counter overflow");
        ++eligible;
        if (!settings.cap || signal_spool->size() < *settings.cap) {
          const auto index = signal_spool->size();
          signal_spool->append(
              Json{{"signal_id", "signal-" + dec(index)},
                   {"source_ordinal", dec(i)},
                   {"available_ns", dec(x.ts_recv)},
                   {"anchor_price_nanos", dec(x.price)},
                   {"causal_end_ordinal_exclusive", dec(i + 1)},
                   {"previous_trade_price_nanos",
                    previous ? Json(dec(*previous))
                             : missing("no preceding trade")}});
          prior_signal = x.ts_recv;
          if (!first_available)
            first_available = x.ts_recv;
          last_available = x.ts_recv;
        }
      }
      previous = x.price;
    }
    auto closed_signals = signal_spool->close();
    const auto count = closed_signals.row_count;
    const auto census_digest =
        str(closed_signals.array_verification.at("canonical_sha256"));
    const Value signals(closed_signals.rows);
    // Each worker owns its reader cache and one append stream. Contiguous
    // ranges make stable concatenation possible without an aggregate row map.
    struct Partition {
      U first, count;
      Value signals;
      std::unique_ptr<result_store::RowSpool> spool;
      std::optional<result_store::ClosedRows> closed;
    };
    const auto actual_workers =
        std::min<U>(settings.workers, std::max<U>(1, count));
    std::vector<Partition> partitions;
    U first = 0;
    for (U worker = 0; worker < actual_workers; ++worker) {
      const auto length =
          count / actual_workers + (worker < count % actual_workers ? 1 : 0);
      partitions.push_back({first,
                            length,
                            signals.independent_reader(output.read_options(),
                                                       output.checkpoint()),
                            output.spool("worker-" + dec(worker)),
                            {}});
      first += length;
    }
    std::atomic<bool> failed{false};
    std::exception_ptr failure;
    std::mutex failure_mutex;
    const auto worker = [&](std::size_t index) {
      try {
        auto &partition = partitions.at(index);
        auto cursor =
            partition.signals.children(partition.first, partition.count);
        U signal_index = partition.first;
        while (!failed) {
          auto selected = cursor.next();
          if (!selected)
            break;
          deadline(end);
          const auto signal = selected->value.materialize();
          const auto ordinal = u64(signal.at("source_ordinal"));
          need(ordinal < events.size() &&
                   signal.at("signal_id") == "signal-" + dec(signal_index),
               "closed run census row contradiction");
          auto row = run_followup(events, static_cast<std::size_t>(ordinal),
                                  signal_index, settings, replay, end);
          partition.spool->append(Json{{"clip", std::move(row.clip)},
                                       {"execution", std::move(row.execution)},
                                       {"markout", std::move(row.markout)}});
          ++signal_index;
        }
        if (!failed) {
          need(signal_index == partition.first + partition.count,
               "run worker census count contradiction");
          partition.closed.emplace(partition.spool->close());
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
      for (std::size_t i = 0; i < partitions.size(); ++i)
        threads.emplace_back(worker, i);
    }
    if (failure)
      std::rethrow_exception(failure);
    deadline(end);
    auto clips = output.spool("replay-windows"),
         executions = output.spool("execution"),
         payload = output.spool("replay-events");
    auto markouts = settings.studies.contains("forward_markout")
                        ? output.spool("markouts")
                        : nullptr;
    U merged = 0, emitted_end = 0, previous_begin = 0, previous_finish = 0;
    for (const auto &partition : partitions) {
      need(partition.closed && partition.closed->row_count == partition.count,
           "run worker completion missing");
      auto rows = partition.closed->rows.children();
      while (auto selected = rows.next()) {
        deadline(end);
        const auto row = selected->read_value();
        const auto &clip = row.at("clip");
        need(clip.at("signal_id") == "signal-" + dec(merged) &&
                 row.at("execution").at("signal_id") == clip.at("signal_id"),
             "run worker merge order contradiction");
        clips->append(clip);
        executions->append(row.at("execution"));
        if (markouts)
          markouts->append(row.at("markout"));
        const auto begin = u64(clip.at("first_ordinal")),
                   finish = u64(clip.at("end_ordinal_exclusive"));
        need(begin <= finish && finish <= events.size() &&
                 (merged == 0 ||
                  (begin >= previous_begin && finish >= previous_finish)),
             "run replay windows must be monotone");
        // Fixed durations over source-ordered anchors give monotone windows.
        // Append only the not-yet-emitted suffix, skipping disjoint gaps.
        if (replay.at("retain_events").get<bool>()) {
          for (U ordinal = std::max(begin, emitted_end); ordinal < finish;
               ++ordinal) {
            if (ordinal % 1024 == 0)
              deadline(end);
            payload->append(
                event_json(events.at(static_cast<std::size_t>(ordinal)),
                           static_cast<std::size_t>(ordinal)));
          }
          emitted_end = std::max(emitted_end, finish);
        }
        previous_begin = begin;
        previous_finish = finish;
        ++merged;
      }
    }
    need(merged == count, "run merge membership contradiction");
    auto closed_clips = clips->close(), closed_execution = executions->close(),
         closed_payload = payload->close();
    std::optional<result_store::ClosedRows> closed_markouts;
    if (markouts)
      closed_markouts.emplace(markouts->close());
    auto result =
        base("native historical Databento MBO closed-census research");
    result["status"] = "partial";
    auto &s = result["sections"];
    s["summary"] =
        section({{"signal_count", dec(count)},
                 {"record_count", dec(events.size())},
                 {"closed_census", true},
                 {"census_sha256", census_digest},
                 {"selection_cap", p.at("criteria").at("max_signals")},
                 {"additional_eligible_after_cap", dec(eligible - count)},
                 {"performance_return",
                  missing("no portfolio/accounting model executed")}});
    s["resources"] = section(
        {{"requested_workers", dec(settings.workers)},
         {"actual_workers", dec(actual_workers)},
         {"backend", "cpu"},
         {"cuda", "unavailable"},
         {"tensor", "unavailable"},
         {"dataset_feed", source.evidence(resident != nullptr)},
         {"derived_resources",
          {{"representation", "private_partitioned_row_spools"},
           {"aggregate_array_materialized", false},
           {"worker_partition", "stable_contiguous_census_ranges"},
           {"worker_readers", "independent_cache_per_worker"},
           {"replay_retention", "monotone_interval_union_without_event_bitmap"},
           {"workspace_retention", "caller_owned"},
           {"retained_census_validation",
            "memory_exact_ordered_map.v1; O(signal_count) identity state"},
           {"accounting_scope",
            "Row/page buffers and retained census identity indexing are "
            "separate from dataset raw/decoded admission; no total process RSS "
            "claim."}}}});
    s["diagnostics"] = section(
        Json::array({"No order book hydration or broker fills. Trade touches "
                     "are observations only.",
                     "Source completeness remains unproven; window span flags "
                     "are not completeness assertions.",
                     "Fixed census closes before all execution/replay/study "
                     "follow-up. Signal caps and settings are explicit."}));
    auto choices = p;
    choices.erase("output");
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
    std::vector<Value> study_results;
    if (settings.studies.contains("signal_summary"))
      study_results.emplace_back(Json{
          {"study_id", "signal_summary"},
          {"version", "1"},
          {"count", dec(count)},
          {"first_available_ns", first_available ? Json(dec(*first_available))
                                                 : missing("empty census")},
          {"last_available_ns", last_available ? Json(dec(*last_available))
                                               : missing("empty census")}});
    if (closed_markouts)
      study_results.push_back(
          Value(Json{{"study_id", "forward_markout"},
                     {"version", "1"},
                     {"results", nullptr}})
              .with("results", Value(closed_markouts->rows)));
    Value logical_result(std::move(result));
    auto sections = logical_result.at("sections");
    sections = sections.with("signals", streamed_section(signals));
    sections = sections.with(
        "execution",
        streamed_section(
            Value(closed_execution.rows),
            settings.model == "none" ? "not_selected" : "partial",
            settings.model == "none"
                ? "user selected no execution model"
                : "touch observations and explicit user assumptions only; no "
                  "calibrated fill inference"));
    auto replay_data =
        Value(Json{{"windows", nullptr},
                   {"events", nullptr},
                   {"event_key", "source_ordinal"},
                   {"retained_event_count", dec(closed_payload.row_count)},
                   {"retention_policy", replay},
                   {"price_scale", "1e-9"},
                   {"clock", "ts_recv, source_ordinal tie-break"}})
            .with("windows", Value(closed_clips.rows))
            .with("events", Value(closed_payload.rows));
    sections = sections.with(
        "replay", streamed_section(std::move(replay_data), "partial",
                                   "capped or uninitialized input cannot "
                                   "establish complete exchange history"));
    sections = sections.with(
        "studies",
        streamed_section(
            Value::array(std::move(study_results)),
            settings.studies.empty() ? "not_selected" : "available",
            settings.studies.empty() ? "user selected zero studies" : ""));
    logical_result = logical_result.with("sections", std::move(sections));
    const auto census = stream_census_evidence(logical_result, end);
    return logical_result.with(
        "sections",
        logical_result.at("sections").with("census", streamed_section(census)));
  });
}
} // namespace symphony::sbv::detail
