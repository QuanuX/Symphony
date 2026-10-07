#include "census.hpp"
#include <algorithm>
#include <optional>
namespace symphony::sbv::detail {
namespace {
bool digest(const Json &v) {
  const auto s = str(v);
  return s.size() == 64 && std::all_of(s.begin(), s.end(), [](char c) {
           return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f');
         });
}
void producer(const Json &p) {
  keys(p, {"id", "version", "artifact_sha256", "reproducibility"});
  for (auto key : {"id", "version"})
    need(!str(p.at(key)).empty() && str(p.at(key)).size() <= 256,
         "bounded census producer required");
  need(str(p.at("artifact_sha256")).empty() || digest(p.at("artifact_sha256")),
       "census producer digest format");
  need(p.at("reproducibility") == "deterministic_declared" ||
           p.at("reproducibility") == "nondeterministic" ||
           p.at("reproducibility") == "uncaptured",
       "census reproducibility declaration required");
}
void validate(const Json &c) {
  keys(c, {"protocol", "kind", "identity_domain", "census_sha256",
           "source_sha256", "dataset", "instrument_id", "mode", "producer",
           "signals", "declaration"});
  need(c.at("protocol") == "symphony.sbv.census-evidence.v1" &&
           digest(c.at("source_sha256")) && digest(c.at("census_sha256")),
       "census evidence protocol/source identity required");
  need(!str(c.at("dataset")).empty(), "census dataset required");
  (void)u64(c.at("instrument_id"));
  const bool native = c.at("kind") == "native";
  need(native || c.at("kind") == "external", "unknown census kind");
  const auto &signals = c.at("signals"), &d = c.at("declaration");
  need(signals.is_array() && signals.size() <= 4096,
       "census signal profile bound");
  if (native) {
    need(c.at("identity_domain") == "native_signals" &&
             c.at("mode") == "native_causal_prefix" &&
             c.at("producer").is_null(),
         "native census identity domain required");
    keys(d, {"criteria", "engine_version", "selection_cap",
             "additional_eligible_after_cap"});
    const auto &criteria = d.at("criteria");
    keys(criteria,
         {"rule", "spacing_ns", "min_trade_size", "direction", "max_signals"});
    need(criteria.at("rule") == "spaced_trades" ||
             criteria.at("rule") == "trade_direction",
         "unknown retained native rule");
    const auto direction = str(criteria.at("direction"));
    need(direction == "any" || direction == "up" || direction == "down" ||
             direction == "unchanged",
         "unknown retained native direction");
    need(criteria.at("rule") != "spaced_trades" || direction == "any",
         "invalid retained native direction");
    (void)u64(criteria.at("spacing_ns"));
    const auto minimum = u64(criteria.at("min_trade_size")),
               cap = u64(criteria.at("max_signals"));
    need(minimum > 0 && minimum <= UINT32_MAX && cap >= 1 && cap <= 4096 &&
             d.at("selection_cap") == criteria.at("max_signals") &&
             signals.size() <= cap && !str(d.at("engine_version")).empty(),
         "retained native selection mismatch");
    (void)u64(d.at("additional_eligible_after_cap"));
    need(e::sha256_hex(signals.dump()) == str(c.at("census_sha256")),
         "native census digest mismatch");
  } else {
    need(c.at("identity_domain") == "external_declaration" &&
             (c.at("mode") == "causal_declared" ||
              c.at("mode") == "retrospective"),
         "external census identity domain required");
    keys(d, {"protocol", "source_sha256", "mode", "producer", "signals"});
    producer(c.at("producer"));
    need(d.at("protocol") == "symphony.sbv.external-census.v1" &&
             d.at("source_sha256") == c.at("source_sha256") &&
             d.at("mode") == c.at("mode") &&
             d.at("producer") == c.at("producer") &&
             d.at("signals") == signals &&
             e::sha256_hex(d.dump()) == str(c.at("census_sha256")),
         "external census declaration mismatch");
  }
  std::set<std::string> ids;
  std::optional<std::uint64_t> prior;
  for (const auto &s : signals) {
    if (native)
      keys(s,
           {"signal_id", "source_ordinal", "available_ns", "anchor_price_nanos",
            "causal_end_ordinal_exclusive", "previous_trade_price_nanos"});
    else
      keys(s,
           {"signal_id", "source_ordinal", "available_ns", "anchor_price_nanos",
            "causal_end_ordinal_exclusive", "context_reference"});
    const auto id = str(s.at("signal_id"));
    const auto ordinal = u64(s.at("source_ordinal")),
               causal = u64(s.at("causal_end_ordinal_exclusive"));
    (void)u64(s.at("available_ns"));
    need(!id.empty() && id.size() <= 256 && ids.insert(id).second &&
             ordinal < UINT64_MAX &&
             (!prior || (native ? ordinal > *prior : ordinal >= *prior)),
         "census identity/order mismatch");
    need(i64(s.at("anchor_price_nanos")) != INT64_MAX,
         "undefined census anchor");
    need(native ? causal == ordinal + 1
                : (c.at("mode") != "causal_declared" || causal <= ordinal + 1),
         "census causal boundary mismatch");
    if (native) {
      const auto &previous = s.at("previous_trade_price_nanos");
      if (previous.is_string())
        need(i64(previous) != INT64_MAX, "undefined previous trade");
      else {
        keys(previous, {"status", "reason", "value"});
        need(previous.at("status") == "unavailable" &&
                 previous.at("value").is_null() &&
                 !str(previous.at("reason")).empty(),
             "previous trade unavailable shape");
      }
    } else
      (void)str(s.at("context_reference"));
    prior = ordinal;
  }
}
Json external(const Json &d, const Json &prov) {
  Json c{{"protocol", "symphony.sbv.census-evidence.v1"},
         {"kind", "external"},
         {"identity_domain", "external_declaration"},
         {"census_sha256", e::sha256_hex(d.dump())},
         {"source_sha256", prov.at("source_sha256")},
         {"dataset", prov.at("dataset")},
         {"instrument_id", prov.at("instrument_id")},
         {"mode", d.at("mode")},
         {"producer", d.at("producer")},
         {"signals", d.at("signals")},
         {"declaration", d}};
  validate(c);
  return c;
}
} // namespace
void validate_census_evidence(const Json &c) { validate(c); }
Json census_evidence(const Json &result) {
  const auto &s = result.at("sections"), &p = s.at("provenance").at("data"),
             &choices = s.at("choices").at("data"),
             &summary = s.at("summary").at("data");
  const auto operation = str(choices.at("protocol"));
  const bool native_run = operation == "symphony.sbv.run-input.v1";
  need(native_run || operation == "symphony.sbv.evaluate-input.v1",
       "census requires a run/evaluate result");
  Json c;
  if (s.contains("census")) {
    need(s.at("census").at("status") == "available",
         "census evidence must be available");
    c = s.at("census").at("data");
  } else if (native_run) {
    c = {{"protocol", "symphony.sbv.census-evidence.v1"},
         {"kind", "native"},
         {"identity_domain", "native_signals"},
         {"census_sha256", summary.at("census_sha256")},
         {"source_sha256", p.at("source_sha256")},
         {"dataset", p.at("dataset")},
         {"instrument_id", p.at("instrument_id")},
         {"mode", "native_causal_prefix"},
         {"producer", nullptr},
         {"signals", s.at("signals").at("data")},
         {"declaration",
          {{"criteria", choices.at("criteria")},
           {"engine_version", p.at("engine_version")},
           {"selection_cap", summary.at("selection_cap")},
           {"additional_eligible_after_cap",
            summary.at("additional_eligible_after_cap")}}}};
  } else
    c = external(choices.at("census"), p);
  validate(c);
  need(c.at("signals") == s.at("signals").at("data") &&
           summary.at("closed_census") == true &&
           u64(summary.at("signal_count")) == c.at("signals").size() &&
           summary.at("census_sha256") == c.at("census_sha256") &&
           c.at("source_sha256") == p.at("source_sha256") &&
           c.at("dataset") == p.at("dataset") &&
           c.at("instrument_id") == p.at("instrument_id") &&
           c.at("source_sha256") == choices.at("source_sha256") &&
           c.at("dataset") == choices.at("dataset"),
       "result and census evidence disagree");
  if (p.contains("census_sha256"))
    need(p.at("census_sha256") == c.at("census_sha256"),
         "provenance census mismatch");
  if (native_run) {
    const auto &d = c.at("declaration");
    need(c.at("kind") == "native" &&
             d.at("criteria") == choices.at("criteria") &&
             d.at("engine_version") == p.at("engine_version") &&
             d.at("selection_cap") == summary.at("selection_cap") &&
             d.at("additional_eligible_after_cap") ==
                 summary.at("additional_eligible_after_cap"),
         "native census selection context mismatch");
  } else {
    need(summary.at("causality") == c.at("mode") &&
             p.at("census_producer") == c.at("producer"),
         "evaluation census declaration mismatch");
    const auto &selection = choices.at("census");
    if (selection.contains("protocol"))
      need(c.at("kind") == "external" && selection == c.at("declaration"),
           "inline census differs from retained declaration");
    else {
      keys(selection, {"path", "expected_sha256", "pointer"});
      const auto &ref = s.at("census_reference").at("data");
      keys(ref, {"path", "expected_sha256", "pointer", "file_sha256",
                 "selected_content_sha256", "authorship"});
      for (auto key : {"path", "expected_sha256", "pointer"})
        need(ref.at(key) == selection.at(key),
             "census reference choices mismatch");
      need(digest(ref.at("expected_sha256")) && digest(ref.at("file_sha256")) &&
               digest(ref.at("selected_content_sha256")) &&
               ref.at("authorship") == "not_verified",
           "invalid census reference lineage");
    }
  }
  return c;
}
void validate_census_source(const Json &c, const Dataset &source,
                            std::int64_t end) {
  need(c.at("source_sha256") == source.sha256 &&
           c.at("dataset") == source.dataset_name &&
           c.at("instrument_id") == dec(source.events.front().instrument_id),
       "retained census source mismatch");
  const bool native = c.at("kind") == "native";
  std::size_t scan = 0;
  std::optional<std::int64_t> previous;
  for (const auto &s : c.at("signals")) {
    deadline(end);
    const auto ordinal = u64(s.at("source_ordinal"));
    need(ordinal < source.events.size() &&
             u64(s.at("causal_end_ordinal_exclusive")) <= source.events.size(),
         "retained census coordinate out of source");
    const auto &event = source.events[ordinal];
    need(u64(s.at("available_ns")) == event.ts_recv,
         "census availability mismatch");
    if (native) {
      while (scan < ordinal) {
        if (scan % 1024 == 0)
          deadline(end);
        const auto &prior = source.events[scan++];
        if (prior.action == 'T' && prior.size > 0 && prior.price != INT64_MAX)
          previous = prior.price;
      }
      need(event.action == 'T' && event.size > 0 && event.price != INT64_MAX &&
               i64(s.at("anchor_price_nanos")) == event.price,
           "native census trade anchor mismatch");
      const auto &p = s.at("previous_trade_price_nanos");
      need(previous ? (p.is_string() && i64(p) == *previous) : p.is_object(),
           "native preceding trade mismatch");
    }
  }
}
Json admit_census(const Json &selection, const Dataset &source, Json &reference,
                  std::int64_t end) {
  Json c;
  reference = nullptr;
  if (selection.contains("protocol"))
    c = external(selection,
                 {{"source_sha256", source.sha256},
                  {"dataset", source.dataset_name},
                  {"instrument_id", dec(source.events.front().instrument_id)}});
  else {
    keys(selection, {"path", "expected_sha256", "pointer"});
    const auto bytes = read_file(str(selection.at("path")), end);
    const auto outer =
        e::parse_bounded_json(bytes, artifact_bytes, artifact_values);
    validate_result(outer);
    need(digest(selection.at("expected_sha256")) &&
             outer.at("content_sha256") == selection.at("expected_sha256"),
         "retained census result digest mismatch");
    const Json::json_pointer ptr(str(selection.at("pointer")));
    need(outer.contains(ptr), "retained census result pointer missing");
    const auto &selected = outer.at(ptr);
    validate_result(selected);
    c = census_evidence(selected);
    reference = {{"path", selection.at("path")},
                 {"expected_sha256", selection.at("expected_sha256")},
                 {"pointer", selection.at("pointer")},
                 {"file_sha256", e::sha256_hex(bytes)},
                 {"selected_content_sha256", selected.at("content_sha256")},
                 {"authorship", "not_verified"}};
  }
  validate_census_source(c, source, end);
  return c;
}
} // namespace symphony::sbv::detail
