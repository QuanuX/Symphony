#include "detail.hpp"
#include "partitioned_output.hpp"
#include "provider.hpp"
#include "stream_census.hpp"
#include "stream_model.hpp"
#include <algorithm>
#include <atomic>
#include <mutex>
#include <symphony/sbv/models.hpp>
#include <symphony/sqav/databento/dbn.hpp>
#include <thread>
namespace symphony::sbv::detail {
namespace db = sqav::databento;
Json event_json(const db::Mbo &, std::size_t);
std::uint64_t end_at(std::uint64_t, std::uint64_t);
Json evaluate_partitioned(const Json &p, std::int64_t end,
                          const Dataset *resident) {
  keys_optional(p,
                {"protocol", "output", "census", "model", "replay", "studies",
                 "workers", "extensions"},
                {"memory_budget_bytes", "dataset_limits", "source_path",
                 "source_sha256", "dataset", "retained_source"});
  const auto workers = u64(p.at("workers"));
  need(workers >= 1 && workers <= 64, "workers 1..64");
  need(p.at("extensions").is_object(), "extensions object required");
  const auto &replay = p.at("replay");
  keys(replay, {"before_ns", "after_ns", "retain_events"});
  const auto before = u64(replay.at("before_ns")),
             after = u64(replay.at("after_ns"));
  need(replay.at("retain_events").is_boolean(), "retention boolean required");
  need(p.at("studies").is_array() && p.at("studies").size() <= 3,
       "study selection bound");
  std::set<std::string> studies;
  for (const auto &v : p.at("studies")) {
    const auto id = str(v);
    need((id == "signal_summary" || id == "model_summary" ||
          id == "path_excursion") &&
             studies.insert(id).second,
         "unknown or duplicate evaluation study");
  }
  return partitioned_result(
      p, "evaluate", end, [&](PartitionedOutput &output) -> logical::Value {
        const auto owned = resident ? nullptr : load_dataset(p, end);
        const auto &source = resident ? *resident : *owned;
        source.bind(p);
        const auto &events = source.events;

        std::vector<ObservedTrade> trades;
        for (std::size_t i = 0; i < events.size(); ++i) {
          if (i % 1024 == 0)
            deadline(end);
          const auto &x = events[i];
          if (x.action == 'T' && x.size > 0 && x.price != INT64_MAX)
            trades.push_back({i, x.ts_recv, x.price, x.size});
        }
        const auto admitted = admit_stream_census(p.at("census"), source, end);
        const auto &c = admitted.value;
        const auto signals = c.at("signals");
        const auto producer = c.at("producer").materialize();
        const auto mode = str(c.at("mode").materialize());
        const auto census_digest = str(c.at("census_sha256").materialize());
        const auto &reference = admitted.reference;
        const auto source_sha = c.at("source_sha256").materialize(),
                   census_dataset = c.at("dataset").materialize(),
                   census_instrument = c.at("instrument_id").materialize();
        const bool native_model = p.at("model").at("id") == "native_provider";
        std::unique_ptr<StreamModel> model;
        std::unique_ptr<NativeProvider> provider;
        std::vector<std::unique_ptr<NativeProviderInstance>> instances;
        Json provider_author = nullptr;
        std::string concurrency = "immutable_native_model";
        auto applied = signals.size() == 0
                           ? std::size_t{0}
                           : std::min<std::size_t>(workers, signals.size());
        if (native_model) {
          const auto &selection = p.at("model").at("parameters").at("provider");
          need(selection.at("role") == "model", "model provider role required");
          provider = std::make_unique<NativeProvider>(selection, end);
          const auto &descriptor = provider->descriptor();
          provider_author = {
              {"id", descriptor.at("id")},
              {"version", descriptor.at("version")},
              {"artifact_sha256",
               selection.at("library").at("expected_sha256")},
              {"reproducibility", descriptor.at("reproducibility")}};
          validate_native_provider_model(p.at("model"), provider_author);
          concurrency = str(selection.at("concurrency"));
          if (concurrency == "serialized_instance" && applied != 0)
            applied = 1;
          const auto count = applied == 0 ? std::size_t{0}
                             : concurrency == "per_worker_instances"
                                 ? applied
                                 : std::size_t{1};
          for (std::size_t i = 0; i < count; ++i)
            instances.push_back(provider->create(end));
        } else
          model = std::make_unique<StreamModel>(p.at("model"), signals, end);
        const auto horizon_ns = u64(p.at("model").at("horizon_ns"));
        struct Row {
          Json outcome, window, excursion;
          std::size_t begin{}, finish{};
        };
        struct Partition {
          std::uint64_t first, count;
          logical::Value signals;
          std::unique_ptr<result_store::RowSpool> spool;
          std::optional<result_store::ClosedRows> closed;
        };
        std::vector<Partition> partitions;
        std::uint64_t first_signal = 0;
        for (std::size_t i = 0; i < applied; ++i) {
          const auto count =
              signals.size() / applied + (i < signals.size() % applied ? 1 : 0);
          partitions.push_back({first_signal,
                                count,
                                signals.independent_reader(
                                    output.read_options(), output.checkpoint()),
                                output.spool("worker-" + dec(i)),
                                {}});
          first_signal += count;
        }
        std::atomic<bool> failed{false};
        std::exception_ptr failure;
        std::mutex mutex;
        const auto worker = [&](std::size_t worker_index) {
          try {
            auto &partition = partitions.at(worker_index);
            auto cursor =
                partition.signals.children(partition.first, partition.count);
            while (!failed) {
              auto selected = cursor.next();
              if (!selected)
                break;
              deadline(end);
              const auto s = selected->value.materialize();
              const auto signal_id = str(s.at("signal_id"));
              const auto ordinal = u64(s.at("source_ordinal")),
                         available = u64(s.at("available_ns"));
              const auto anchor = i64(s.at("anchor_price_nanos"));
              const auto horizon_end = end_at(available, horizon_ns);
              auto begin_trade = std::upper_bound(
                  trades.begin(), trades.end(), ordinal,
                  [](auto v, const auto &t) { return v < t.source_ordinal; });
              auto end_trade = std::upper_bound(
                  begin_trade, trades.end(), horizon_end,
                  [](auto v, const auto &t) { return v < t.available_ns; });
              const auto &event = events[ordinal];
              ModelFrame frame{
                  signal_id,
                  ordinal,
                  available,
                  horizon_end,
                  anchor,
                  event.action == 'T' && event.size > 0 &&
                      event.price == anchor,
                  horizon_end <= events.back().ts_recv,
                  std::span<const ObservedTrade>(begin_trade, end_trade)};
              Row row;
              if (provider) {
                const auto followup_end = static_cast<std::size_t>(
                    std::upper_bound(
                        events.begin() + ordinal + 1, events.end(), horizon_end,
                        [](auto t, const auto &x) { return t < x.ts_recv; }) -
                    events.begin());
                const Json context{{"signal", s},
                                   {"census_sha256", census_digest},
                                   {"source_sha256", source_sha},
                                   {"dataset", census_dataset},
                                   {"instrument_id", census_instrument}};
                const Json coverage{
                    {"clock", "ts_recv with source ordinal tie-break"},
                    {"causal_first_ordinal", "0"},
                    {"causal_end_ordinal_exclusive", dec(ordinal + 1)},
                    {"followup_first_ordinal", dec(ordinal + 1)},
                    {"followup_end_ordinal_exclusive", dec(followup_end)},
                    {"horizon_end_ns_inclusive", dec(horizon_end)},
                    {"horizon_within_observed_span",
                     frame.horizon_within_observed_span},
                    {"observed_first_ns", dec(events.front().ts_recv)},
                    {"observed_last_ns", dec(events.back().ts_recv)},
                    {"completeness", "unproven"},
                    {"book_state", "uninitialized"},
                    {"payload_layer", "observed"}};
                auto &instance = instances[concurrency == "per_worker_instances"
                                               ? worker_index
                                               : 0];
                auto reply =
                    instance->evaluate(events, ordinal, followup_end, context,
                                       coverage, horizon_end, end, &failed);
                need(reply.status == SBV_PROVIDER_OK ||
                         reply.status == SBV_PROVIDER_UNAVAILABLE,
                     ("native model provider failed: " + reply.error.dump())
                         .c_str());
                row.outcome = admit_native_provider_outcome(
                    p.at("model"), provider_author, reply.value,
                    reply.status == SBV_PROVIDER_UNAVAILABLE ? reply.error
                                                             : Json(nullptr),
                    frame, end);
              } else
                row.outcome = model->evaluate(frame, end);
              const auto start = available < before ? 0 : available - before,
                         stop = end_at(available, after);
              const auto first = std::lower_bound(
                  events.begin(), events.end(), start,
                  [](const auto &x, auto t) { return x.ts_recv < t; });
              const auto last = std::upper_bound(
                  first, events.end(), stop,
                  [](auto t, const auto &x) { return t < x.ts_recv; });
              row.begin = static_cast<std::size_t>(first - events.begin());
              row.finish = static_cast<std::size_t>(last - events.begin());
              row.window = {
                  {"signal_id", signal_id},
                  {"requested_start_ns", dec(start)},
                  {"requested_end_ns_inclusive", dec(stop)},
                  {"start_clamped_at_epoch", available < before},
                  {"first_ordinal", dec(row.begin)},
                  {"end_ordinal_exclusive", dec(row.finish)},
                  {"causal_end_ordinal_exclusive",
                   s.at("causal_end_ordinal_exclusive")},
                  {"start_within_observed_span",
                   start >= events.front().ts_recv},
                  {"end_within_observed_span", stop <= events.back().ts_recv},
                  {"completeness", "unproven"},
                  {"book_state", "uninitialized"},
                  {"payload_layer", "observed"}};
              if (studies.contains("path_excursion")) {
                row.excursion = {
                    {"signal_id", signal_id},
                    {"horizon_end_ns", dec(horizon_end)},
                    {"unit", "price_nanos"},
                    {"coverage", frame.horizon_within_observed_span
                                     ? "observed_span_only"
                                     : "truncated_observed_span"},
                    {"minimum", missing("no observed future trade")},
                    {"maximum", missing("no observed future trade")}};
                if (begin_trade != end_trade) {
                  const auto minimum = std::min_element(
                      begin_trade, end_trade, [](const auto &a, const auto &b) {
                        return a.price_nanos < b.price_nanos;
                      });
                  const auto maximum = std::max_element(
                      begin_trade, end_trade, [](const auto &a, const auto &b) {
                        return a.price_nanos < b.price_nanos;
                      });
                  auto extremum = [&](const ObservedTrade &t) -> Json {
                    std::int64_t change{};
                    if (__builtin_sub_overflow(t.price_nanos, anchor, &change))
                      return missing("exact price difference overflows int64");
                    return {{"status", "available"},
                            {"value", dec(change)},
                            {"source_ordinal", dec(t.source_ordinal)},
                            {"available_ns", dec(t.available_ns)}};
                  };
                  row.excursion["minimum"] = extremum(*minimum);
                  row.excursion["maximum"] = extremum(*maximum);
                }
              }
              partition.spool->append({{"outcome", row.outcome},
                                       {"window", row.window},
                                       {"excursion", row.excursion}});
            }
            if (!failed)
              partition.closed.emplace(partition.spool->close());
          } catch (...) {
            failed = true;
            std::lock_guard lock(mutex);
            if (!failure)
              failure = std::current_exception();
          }
        };
        {
          std::vector<std::jthread> threads;
          for (std::size_t i = 0; i < applied; ++i)
            threads.emplace_back(worker, i);
        }
        if (failure)
          std::rethrow_exception(failure);
        const auto instance_count = instances.size();
        instances.clear();
        auto result = base("native follow-up of an immutable admitted census");
        result["status"] = "partial";
        auto &s = result["sections"];
        s["signals"] = section(Json::array());
        s["census"] = section(nullptr);
        s["census_reference"] = reference.is_null()
                                    ? section(nullptr, "not_selected",
                                              "inline external declaration")
                                    : section(reference);
        s["summary"] = section({{"signal_count", dec(signals.size())},
                                {"record_count", dec(events.size())},
                                {"census_sha256", census_digest},
                                {"closed_census", true},
                                {"causality", mode},
                                {"external_causality_verified", false}});
        auto outcomes_spool = output.spool("outcomes"),
             windows_spool = output.spool("windows"),
             excursions_spool = output.spool("excursions"),
             events_spool = output.spool("events");
        std::vector<std::pair<std::uint64_t, std::uint64_t>> ranges;
        std::uint64_t available_count = 0;
        for (auto &partition : partitions) {
          need(partition.closed.has_value() &&
                   partition.closed->row_count == partition.count,
               "worker partition did not close every selected signal");
          auto rows = partition.closed->rows.children();
          while (auto selected = rows.next()) {
            deadline(end);
            auto row = selected->read_value();
            outcomes_spool->append(row.at("outcome"));
            windows_spool->append(row.at("window"));
            if (row.at("outcome").at("status") == "available")
              ++available_count;
            if (studies.contains("path_excursion"))
              excursions_spool->append(row.at("excursion"));
            if (replay.at("retain_events").get<bool>()) {
              const auto first = u64(row.at("window").at("first_ordinal")),
                         last =
                             u64(row.at("window").at("end_ordinal_exclusive"));
              if (first != last) {
                if (ranges.empty() || first > ranges.back().second)
                  ranges.emplace_back(first, last);
                else
                  ranges.back().second = std::max(ranges.back().second, last);
              }
            }
          }
        }
        for (const auto &[first, last] : ranges)
          for (auto ordinal = first; ordinal < last; ++ordinal) {
            deadline(end);
            events_spool->append(
                event_json(events.at(static_cast<std::size_t>(ordinal)),
                           static_cast<std::size_t>(ordinal)));
          }
        auto outcomes = outcomes_spool->close(),
             windows = windows_spool->close(),
             excursions = excursions_spool->close(),
             retained_events = events_spool->close();
        s["execution"] =
            section(Json::array(), "partial",
                    "conditional model evidence or external claims; "
                    "actual fills and calibration unverified");
        s["replay"] =
            section({{"windows", Json::array()},
                     {"events", Json::array()},
                     {"retained_event_count", dec(retained_events.row_count)},
                     {"event_key", "source_ordinal"},
                     {"retention_policy", replay},
                     {"price_scale", "1e-9"},
                     {"clock", "ts_recv with source ordinal tie-break"}},
                    "partial",
                    "observed fragments; no initialized book or exchange "
                    "completeness proof");
        Json selected = Json::array();
        if (studies.contains("signal_summary"))
          selected.push_back({{"study_id", "signal_summary"},
                              {"version", "1"},
                              {"count", dec(signals.size())}});
        if (studies.contains("model_summary"))
          selected.push_back(
              {{"study_id", "model_summary"},
               {"version", "1"},
               {"available", dec(available_count)},
               {"unavailable", dec(signals.size() - available_count)}});
        if (studies.contains("path_excursion"))
          selected.push_back({{"study_id", "path_excursion"},
                              {"version", "1"},
                              {"results", Json::array()}});
        s["studies"] =
            section(selected, studies.empty() ? "not_selected" : "available",
                    studies.empty() ? "zero studies selected" : "");
        s["resources"] = section({{"requested_workers", dec(workers)},
                                  {"actual_workers", dec(applied)},
                                  {"backend", "cpu"}});
        if (provider) {
          s["provider"] =
              section({{"evidence", provider->evidence()}, {"role", "model"}});
          auto &resources = s["resources"]["data"];
          resources["provider_concurrency"] = concurrency;
          resources["provider_instances"] = dec(instance_count);
          resources["signal_assignment"] =
              concurrency == "serialized_instance"
                  ? "source signal order"
                  : "contiguous census partitions in ascending worker index; "
                    "ordered merge";
        }
        s["diagnostics"] = section(Json::array(
            {"Census identity and source coordinates are checked; producer "
             "authorship and external causal declarations are not "
             "authenticated.",
             "Observed trade levels are retrospective support, not book "
             "liquidity or unconditional fill likelihood.",
             "No normalization or probability interpretation is imposed "
             "on a declared non-probability measure."}));
        if (provider) {
          auto &messages = s["diagnostics"]["data"];
          messages.push_back("Native provider candidates are admitted under "
                             "the existing exact "
                             "model domains. The external_assumption "
                             "likelihood status denotes "
                             "provider-supplied assumptions, not a foreign "
                             "language runtime or "
                             "calibrated execution.");
          messages.push_back(
              "Callbacks receive full raw causal/followup MBO fields. "
              "Trusted native code is not sandboxed, and raw events "
              "do not establish book initialization.");
          messages.push_back(
              "The selected instance topology and actual workers are retained. "
              "Stateful per-worker or shared providers may depend on "
              "concurrent "
              "assignment; deterministic ordering of output rows does not "
              "prove "
              "repeatable provider state.");
        }
        auto choices = p;
        choices.erase("output");
        s["choices"] = section(choices);
        const auto &meta = source.metadata;
        s["provenance"] = section(
            {{"engine_version", version},
             {"source_sha256", source.sha256},
             {"source_path", source.path},
             {"dataset", source.dataset_name},
             {"adapter", db::adapter_id},
             {"adapter_version", db::adapter_version},
             {"dbn_version", dec(meta.version)},
             {"instrument_id", dec(events.front().instrument_id)},
             {"source_record_limit", dec(meta.limit)},
             {"source_cap_reached",
              meta.limit != 0 && events.size() >= meta.limit},
             {"requested_start_ns", dec(meta.start)},
             {"requested_end_ns_exclusive", dec(meta.end)},
             {"observed_first_ns", dec(events.front().ts_recv)},
             {"observed_last_ns", dec(events.back().ts_recv)},
             {"census_producer", producer},
             {"census_sha256", census_digest},
             {"external_dependency_capture",
              "declared only; SBV cannot audit uncaptured producer inputs"},
             {"provider_requests", "0"},
             {"additional_spend_usd", "0"}});
        s["user_extensions"] = section(p.at("extensions"));
        result["sections"]["resources"]["data"]["dataset_feed"] =
            source.evidence(resident != nullptr);
        auto &resource = s["resources"]["data"];
        resource["signal_assignment"] = "contiguous census partitions in "
                                        "ascending worker index; ordered merge";
        resource["worker_row_storage"] =
            "private partitioned row spools; self-contained ordered final copy";
        resource["replay_intervals"] = dec(ranges.size());
        resource["replay_interval_state"] =
            "coalesced exact ordinal pairs; O(disjoint intervals), no policy "
            "quota";
        resource["signal_reader_cache_bytes_per_worker"] = "0";
        if (model)
          resource["model_admission"] = model->evidence();
        logical::Value body(std::move(result));
        auto sections = body.at("sections");
        auto graft = [&](const char *name, logical::Value data) {
          sections = sections.with(
              name, sections.at(name).with("data", std::move(data)));
        };
        graft("signals", signals);
        graft("census", c);
        graft("execution", logical::Value(outcomes.rows));
        graft("replay",
              sections.at("replay")
                  .at("data")
                  .with("windows", logical::Value(windows.rows))
                  .with("events", logical::Value(retained_events.rows)));
        std::vector<logical::Value> study_values;
        auto study_cursor = sections.at("studies").at("data").children();
        while (auto study = study_cursor.next()) {
          auto value = study->value;
          if (value.at("study_id").materialize() == "path_excursion")
            value = value.with("results", logical::Value(excursions.rows));
          study_values.push_back(std::move(value));
        }
        graft("studies", logical::Value::array(std::move(study_values)));
        if (model && p.at("model").at("id") == "external_outcomes") {
          auto model_inputs =
              logical::Value(Json{{"outcomes", nullptr},
                                  {"reference", model->source_reference()}})
                  .with("outcomes", model->outcomes());
          sections = sections.with("model_inputs",
                                   logical::Value(section(nullptr))
                                       .with("data", std::move(model_inputs)));
        }
        return body.with("sections", std::move(sections));
      });
}
} // namespace symphony::sbv::detail
