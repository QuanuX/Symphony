#include "stream_census.hpp"
#include "exact_id_index.hpp"
#include "provider.hpp"
#include <algorithm>
#include <optional>
#include <symphony/knowledge/engine/path.hpp>
namespace symphony::sbv::detail {
namespace {
using Value = logical::Value;
Json metadata_value(const Value &v) {
  // Selected typed metadata is materialized with no implicit resource quota.
  // Signal/declaration arrays are kept as owned streamed values separately.
  return v.materialize();
}
Json metadata(const Value &v, std::initializer_list<const char *> omitted) {
  need(v.kind() == "object", "typed metadata object required");
  Json out = Json::object();
  auto children = v.children();
  while (auto child = children.next()) {
    bool omit = false;
    for (auto key : omitted)
      omit |= child->edge == key;
    out[child->edge] = omit ? Json(nullptr) : metadata_value(child->value);
  }
  return out;
}
void value_keys(const Value &v, std::initializer_list<const char *> expected) {
  need(v.kind() == "object" && v.size() == expected.size(),
       "unexpected logical object fields");
  for (auto key : expected)
    need(v.contains(key), "required logical field missing");
}
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
void validate(const Value &value, std::int64_t end) {
  const auto c = metadata(value, {"signals", "declaration"});
  const auto signals = value.at("signals"),
             declaration = value.at("declaration");
  const auto d = metadata(declaration, {"signals"});
  const auto signal_digest = signals.sha256({}, [&] { deadline(end); });
  const auto declaration_digest =
      declaration.sha256({}, [&] { deadline(end); });
  keys(c, {"protocol", "kind", "identity_domain", "census_sha256",
           "source_sha256", "dataset", "instrument_id", "mode", "producer",
           "signals", "declaration"});
  need(c.at("protocol") == "symphony.sbv.census-evidence.v1" &&
           digest(c.at("source_sha256")) && digest(c.at("census_sha256")),
       "census evidence protocol/source identity required");
  need(!str(c.at("dataset")).empty(), "census dataset required");
  (void)u64(c.at("instrument_id"));
  const bool native = c.at("kind") == "native",
             provider_kind = c.at("kind") == "native_provider";
  need(native || provider_kind || c.at("kind") == "external",
       "unknown census kind");
  need(signals.kind() == "array", "census signals array required");
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
    const auto minimum = u64(criteria.at("min_trade_size"));
    const auto &cap = criteria.at("max_signals");
    need(minimum > 0 && minimum <= UINT32_MAX &&
             (cap.is_null() || (u64(cap) >= 1 && signals.size() <= u64(cap))) &&
             d.at("selection_cap") == cap &&
             !str(d.at("engine_version")).empty(),
         "retained native selection mismatch");
    const auto after_cap = u64(d.at("additional_eligible_after_cap"));
    need(!cap.is_null() || after_cap == 0,
         "uncapped census cannot have post-cap eligibility");
    need(signal_digest == str(c.at("census_sha256")),
         "native census digest mismatch");
  } else if (provider_kind) {
    need(c.at("identity_domain") == "native_provider_declaration" &&
             c.at("mode") == "native_provider_prefix",
         "provider census identity domain required");
    keys(d, {"protocol", "source_sha256", "dataset", "instrument_id",
             "provider", "provider_evidence", "signals", "completion"});
    producer(c.at("producer"));
    const auto &selection = d.at("provider"), &author = c.at("producer");
    keys(selection,
         {"protocol", "library", "id", "version", "role", "input_profile",
          "concurrency", "parameters", "dependencies", "extensions"});
    keys(selection.at("library"), {"path", "expected_sha256"});
    need(d.at("protocol") == "symphony.sbv.native-provider-census.v1" &&
             d.at("source_sha256") == c.at("source_sha256") &&
             d.at("dataset") == c.at("dataset") &&
             d.at("instrument_id") == c.at("instrument_id") &&
             declaration.at("signals").sha256({}, [&] { deadline(end); }) ==
                 signal_digest &&
             selection.at("protocol") ==
                 "symphony.sbv.native-provider-selection.v1" &&
             selection.at("role") == "strategy" &&
             selection.at("concurrency") == "serialized_instance" &&
             selection.at("input_profile") ==
                 "symphony.sbv.provider-databento-mbo-event.v1" &&
             selection.at("id") == author.at("id") &&
             selection.at("version") == author.at("version") &&
             selection.at("library").at("expected_sha256") ==
                 author.at("artifact_sha256") &&
             selection.at("parameters").is_object() &&
             selection.at("extensions").is_object() &&
             d.at("provider_evidence").is_object() &&
             declaration_digest == str(c.at("census_sha256")),
         "provider census declaration mismatch");
    const auto &completion = d.at("completion");
    keys(completion, {"diagnostics", "extensions"});
    need(completion.at("diagnostics").is_array() &&
             completion.at("extensions").is_object(),
         "provider census completion shape");
    for (const auto &message : completion.at("diagnostics"))
      (void)str(message);
    const auto &evidence = d.at("provider_evidence");
    validate_native_provider_evidence(selection, evidence);
    need(evidence.at("protocol") ==
                 "symphony.sbv.native-provider-evidence.v1" &&
             evidence.at("selection_sha256") ==
                 e::sha256_hex(selection.dump()) &&
             evidence.at("id") == author.at("id") &&
             evidence.at("version") == author.at("version") &&
             evidence.at("role") == "strategy" &&
             evidence.at("concurrency") == "serialized_instance" &&
             evidence.at("reproducibility") == author.at("reproducibility") &&
             evidence.at("library").at("path") ==
                 selection.at("library").at("path") &&
             evidence.at("library").at("expected_sha256") ==
                 author.at("artifact_sha256") &&
             evidence.at("descriptor_sha256") ==
                 e::sha256_hex(str(evidence.at("descriptor_json"))),
         "provider census captured evidence mismatch");
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
             declaration.at("signals").sha256({}, [&] { deadline(end); }) ==
                 signal_digest &&
             declaration_digest == str(c.at("census_sha256")),
         "external census declaration mismatch");
  }
  ExactIdIndex ids({}, [&] { deadline(end); });
  std::optional<std::uint64_t> prior;
  auto rows = signals.children();
  while (auto selected = rows.next()) {
    deadline(end);
    const auto s = metadata_value(selected->value);
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
    need(!id.empty() && id.size() <= 256 && ids.insert(id).inserted &&
             ordinal < UINT64_MAX &&
             (!prior || (native ? ordinal > *prior : ordinal >= *prior)),
         "census identity/order mismatch");
    need(i64(s.at("anchor_price_nanos")) != INT64_MAX,
         "undefined census anchor");
    need(native || provider_kind
             ? causal == ordinal + 1
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
Value external_value(const Value &d, const Json &prov, std::int64_t end) {
  auto signals = d.at("signals");
  Value c(Json{{"protocol", "symphony.sbv.census-evidence.v1"},
               {"kind", "external"},
               {"identity_domain", "external_declaration"},
               {"census_sha256", d.sha256({}, [&] { deadline(end); })},
               {"source_sha256", prov.at("source_sha256")},
               {"dataset", prov.at("dataset")},
               {"instrument_id", prov.at("instrument_id")},
               {"mode", metadata_value(d.at("mode"))},
               {"producer", metadata_value(d.at("producer"))},
               {"signals", nullptr},
               {"declaration", nullptr}});
  c = c.with("signals", signals).with("declaration", d);
  validate(c, end);
  return c;
}
std::string path(const Json &j) {
  auto value = str(j);
  need(value.size() > 1 && value.size() <= 4096 && value.front() == '/' &&
           e::is_safe_relative_path(value.substr(1)),
       "absolute clean source path required");
  return value;
}
void reference_lineage(const Json &selection, const Json &ref) {
  if (selection.contains("kind")) {
    keys(selection, {"kind", "reference", "selector", "read_options"});
    keys(ref, {"kind", "reference", "selector", "read_options",
               "selected_node_id", "selected_value_sha256",
               "selected_content_sha256", "verification_extent", "authorship"});
    for (auto key : {"kind", "reference", "selector", "read_options"})
      need(selection.at(key) == ref.at(key),
           "bundle census reference choices mismatch");
    need(ref.at("kind") == "bundle" &&
             digest(ref.at("selected_value_sha256")) &&
             (ref.at("selected_content_sha256").is_null() ||
              digest(ref.at("selected_content_sha256"))) &&
             ref.at("verification_extent") == "full_outer_logical_closure" &&
             ref.at("authorship") == "not_verified",
         "invalid bundle census lineage");
    (void)u64(ref.at("selected_node_id"));
    const auto &r = ref.at("reference");
    keys(r, {"manifest_path", "manifest_sha256", "content_sha256"});
    (void)path(r.at("manifest_path"));
    need(digest(r.at("manifest_sha256")) && digest(r.at("content_sha256")),
         "invalid bundle source digests");
    const auto &o = ref.at("read_options");
    keys(o, {"max_page_bytes", "cache_bytes"});
    need(o.at("max_page_bytes").is_null() || u64(o.at("max_page_bytes")) > 0,
         "positive selected page budget required");
    (void)u64(o.at("cache_bytes"));
    const auto &sel = ref.at("selector");
    if (sel.at("kind") == "node_id") {
      keys(sel, {"kind", "node_id"});
      need(sel.at("node_id") == ref.at("selected_node_id"),
           "selected node differs from declared selector");
    } else {
      keys(sel, {"kind", "pointer"});
      need(sel.at("kind") == "pointer", "unknown bundle selector");
      (void)Json::json_pointer(str(sel.at("pointer")));
    }
    const bool root_selected =
        (sel.at("kind") == "node_id" && sel.at("node_id") == "0") ||
        (sel.at("kind") == "pointer" && sel.at("pointer") == "");
    need(!root_selected ||
             (ref.at("selected_node_id") == "0" &&
              ref.at("selected_content_sha256") == r.at("content_sha256")),
         "root bundle reference identity mismatch");
    need(ref.at("selected_node_id") != "0" || root_selected,
         "nonroot selector cannot claim root");

  } else {
    keys(selection, {"path", "expected_sha256", "pointer"});
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
} // namespace
void validate_logical_reference_lineage(const Json &selection, const Json &ref) {
  reference_lineage(selection, ref);
}
void validate_stream_census(const Value &c, std::int64_t end) {
  validate(c, end);
}
void validate_logical_result(const Value &r, std::int64_t end) {
  value_keys(r, {"protocol", "origin", "status", "sections", "content_sha256"});
  need(metadata_value(r.at("protocol")) == result_protocol &&
           metadata_value(r.at("origin")).is_string(),
       "unsupported logical result");
  const auto status = metadata_value(r.at("status"));
  need(status == "completed" || status == "partial",
       "invalid logical result status");
  const auto sections = r.at("sections");
  need(sections.kind() == "object", "logical sections object required");
  for (auto key : {"summary", "signals", "execution", "distributions",
                   "studies", "replay", "comparisons", "search", "resources",
                   "diagnostics", "choices", "provenance"})
    need(sections.contains(key), "logical required section absent");
  auto all = sections.children();
  while (auto s = all.next()) {
    deadline(end);
    value_keys(s->value, {"status", "reason", "data"});
    const auto status = str(metadata_value(s->value.at("status"))),
               reason = str(metadata_value(s->value.at("reason")));
    need((status == "available" || status == "partial" ||
          status == "not_selected" || status == "unavailable") &&
             (status == "available" || !reason.empty()),
         "invalid logical section semantics");
  }
  const auto expected = metadata_value(r.at("content_sha256"));
  need(digest(expected) &&
           str(expected) ==
               r.without("content_sha256").sha256({}, [&] { deadline(end); }),
       "logical selected-result digest mismatch");
}
LogicalSelection select_bundle_value(const Json &selection, std::int64_t end) {
  keys(selection, {"kind", "reference", "selector", "read_options"});
  need(selection.at("kind") == "bundle", "bundle value source required");
  const auto &r = selection.at("reference");
  keys(r, {"manifest_path", "manifest_sha256", "content_sha256"});
  need(digest(r.at("manifest_sha256")) && digest(r.at("content_sha256")),
       "bundle source digest required");
  const auto &o = selection.at("read_options");
  keys(o, {"max_page_bytes", "cache_bytes"});
  result_store::ReadOptions options;
  options.cache_bytes = u64(o.at("cache_bytes"));
  if (!o.at("max_page_bytes").is_null()) {
    options.max_page_bytes = u64(o.at("max_page_bytes"));
    need(*options.max_page_bytes > 0, "positive selected page budget required");
  }
  const auto &s = selection.at("selector");
  need(s.is_object() && s.contains("kind"), "bundle selector required");
  if (s.at("kind") == "node_id") {
    keys(s, {"kind", "node_id"});
    (void)u64(s.at("node_id"));
  } else {
    keys(s, {"kind", "pointer"});
    need(s.at("kind") == "pointer", "unknown bundle selector");
    (void)Json::json_pointer(str(s.at("pointer")));
  }
  result_store::ResultReader reader({path(r.at("manifest_path")),
                                     str(r.at("manifest_sha256")),
                                     str(r.at("content_sha256"))},
                                    options, [end] { deadline(end); });
  reader.verify_closure();
  auto node = s.at("kind") == "node_id"
                  ? reader.select_node(u64(s.at("node_id")))
                  : reader.select(str(s.at("pointer")));
  const auto id = node.describe().at("node_id");
  Value value(std::move(node));
  Json ref = selection;
  ref["selected_node_id"] = id;
  ref["selected_value_sha256"] = value.sha256({}, [&] { deadline(end); });
  ref["selected_content_sha256"] = nullptr;
  if (value.kind() == "object" && value.contains("protocol") &&
      metadata_value(value.at("protocol")) == result_protocol) {
    validate_logical_result(value, end);
    ref["selected_content_sha256"] = metadata_value(value.at("content_sha256"));
  }
  ref["verification_extent"] = "full_outer_logical_closure";
  ref["authorship"] = "not_verified";
  return {std::move(value), std::move(ref)};
}
LogicalSelection select_logical_result(const Json &selection,
                                       std::int64_t end) {
  if (selection.contains("kind")) {
    auto selected = select_bundle_value(selection, end);
    need(!selected.reference.at("selected_content_sha256").is_null(),
         "complete result source required");
    return selected;
  }
  keys(selection, {"path", "expected_sha256", "pointer"});
  auto bytes = read_file(str(selection.at("path")), end);
  auto outer = e::parse_bounded_json(bytes, artifact_bytes, artifact_values);
  validate_result(outer);
  need(digest(selection.at("expected_sha256")) &&
           outer.at("content_sha256") == selection.at("expected_sha256"),
       "legacy outer logical source mismatch");
  Json::json_pointer pointer(str(selection.at("pointer")));
  need(outer.contains(pointer), "legacy selected result absent");
  auto selected = outer.at(pointer);
  validate_result(selected);
  Json reference{{"path", selection.at("path")},
                 {"expected_sha256", selection.at("expected_sha256")},
                 {"pointer", selection.at("pointer")},
                 {"file_sha256", e::sha256_hex(bytes)},
                 {"selected_content_sha256", selected.at("content_sha256")},
                 {"authorship", "not_verified"}};
  return {Value(std::move(selected)), std::move(reference)};
}
Value stream_census_evidence(const Value &result, std::int64_t end) {
  const auto sections = result.at("sections");
  const auto p = metadata_value(sections.at("provenance").at("data")),
             summary = metadata_value(sections.at("summary").at("data"));
  const auto choices_value = sections.at("choices").at("data");
  Json choices = Json::object();
  for (auto key : {"protocol", "source_path", "source_sha256", "dataset",
                   "retained_source", "criteria", "provider"})
    if (choices_value.contains(key))
      choices[key] = metadata_value(choices_value.at(key));
  const auto operation = str(choices.at("protocol"));
  const bool native_run = operation == "symphony.sbv.run-input.v1",
             generated = operation == "symphony.sbv.generate-census-input.v1";
  need(native_run || generated || operation == "symphony.sbv.evaluate-input.v1",
       "census requires run/generate/evaluate result");
  need(!generated || sections.contains("census"),
       "generated provider census evidence missing");
  auto c = [&]() -> Value {
    if (sections.contains("census")) {
      auto retained = sections.at("census");
      need(metadata_value(retained.at("status")) == "available",
           "census evidence unavailable");
      return retained.at("data");
    }
    if (native_run) {
      Value normalized(Json{{"protocol", "symphony.sbv.census-evidence.v1"},
                            {"kind", "native"},
                            {"identity_domain", "native_signals"},
                            {"census_sha256", summary.at("census_sha256")},
                            {"source_sha256", p.at("source_sha256")},
                            {"dataset", p.at("dataset")},
                            {"instrument_id", p.at("instrument_id")},
                            {"mode", "native_causal_prefix"},
                            {"producer", nullptr},
                            {"signals", nullptr},
                            {"declaration",
                             {{"criteria", choices.at("criteria")},
                              {"engine_version", p.at("engine_version")},
                              {"selection_cap", summary.at("selection_cap")},
                              {"additional_eligible_after_cap",
                               summary.at("additional_eligible_after_cap")}}}});
      return normalized.with("signals", sections.at("signals").at("data"));
    }
    return external_value(choices_value.at("census"), p, end);
  }();
  validate(c, end);
  const auto meta = metadata(c, {"signals", "declaration"}),
             d = metadata(c.at("declaration"), {"signals"});
  const auto selected_source = dataset_result_identity(
      choices, p, metadata_value(sections.at("resources").at("data")));
  need(c.at("signals").sha256({}, [&] { deadline(end); }) ==
               sections.at("signals").at("data").sha256(
                   {}, [&] { deadline(end); }) &&
           summary.at("closed_census") == true &&
           u64(summary.at("signal_count")) == c.at("signals").size() &&
           summary.at("census_sha256") == meta.at("census_sha256") &&
           meta.at("source_sha256") == p.at("source_sha256") &&
           meta.at("dataset") == p.at("dataset") &&
           meta.at("instrument_id") == p.at("instrument_id") &&
           meta.at("source_sha256") == selected_source.at("source_sha256") &&
           meta.at("dataset") == selected_source.at("dataset"),
       "result and census evidence disagree");
  if (p.contains("census_sha256"))
    need(p.at("census_sha256") == meta.at("census_sha256"),
         "provenance census mismatch");
  if (native_run) {
    need(meta.at("kind") == "native" &&
             d.at("criteria") == choices.at("criteria") &&
             d.at("engine_version") == p.at("engine_version") &&
             d.at("selection_cap") == summary.at("selection_cap") &&
             d.at("additional_eligible_after_cap") ==
                 summary.at("additional_eligible_after_cap"),
         "native census selection context mismatch");
  } else if (generated) {
    const auto provider = metadata_value(sections.at("provider"));
    need(meta.at("kind") == "native_provider" &&
             summary.at("causality") == meta.at("mode") &&
             p.at("census_producer") == meta.at("producer") &&
             d.at("provider") == choices.at("provider") &&
             provider.at("status") == "available" &&
             provider.at("data").at("evidence") == d.at("provider_evidence") &&
             provider.at("data").at("completion") == d.at("completion"),
         "generated provider census context mismatch");
  } else {
    need(summary.at("causality") == meta.at("mode") &&
             p.at("census_producer") == meta.at("producer"),
         "evaluation census declaration mismatch");
    const auto selection_value = choices_value.at("census");
    if (selection_value.contains("protocol"))
      need(meta.at("kind") == "external" &&
               selection_value.sha256({}, [&] { deadline(end); }) ==
                   c.at("declaration").sha256({}, [&] { deadline(end); }),
           "inline census differs from retained declaration");
    else {
      const auto selection = metadata_value(selection_value);
      const auto ref =
          metadata_value(sections.at("census_reference").at("data"));
      reference_lineage(selection, ref);
      if (selection.contains("kind") &&
          ref.at("selected_content_sha256").is_null())
        need(meta.at("kind") == "external" &&
                 ref.at("selected_value_sha256") == meta.at("census_sha256"),
             "referenced external declaration identity mismatch");
    }
  }
  return c;
}
void validate_stream_census_source(const Value &value, const Dataset &source,
                                   std::int64_t end) {
  const auto c = metadata(value, {"signals", "declaration"});
  need(c.at("source_sha256") == source.sha256 &&
           c.at("dataset") == source.dataset_name &&
           c.at("instrument_id") == dec(source.events.front().instrument_id),
       "retained census source mismatch");
  const bool native = c.at("kind") == "native";
  std::size_t scan = 0;
  std::optional<std::int64_t> previous;
  auto rows = value.at("signals").children();
  while (auto row = rows.next()) {
    const auto s = metadata_value(row->value);
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
LogicalSelection admit_stream_census(const Json &selection,
                                     const Dataset &source, std::int64_t end) {
  Json ref = nullptr;
  auto c = [&]() -> Value {
    if (selection.contains("protocol"))
      return external_value(
          Value(selection),
          {{"source_sha256", source.sha256},
           {"dataset", source.dataset_name},
           {"instrument_id", dec(source.events.front().instrument_id)}},
          end);
    if (selection.contains("kind")) {
      auto selected = select_bundle_value(selection, end);
      ref = selected.reference;
      need(selected.value.kind() == "object" &&
               selected.value.contains("protocol"),
           "selected census declaration/result required");
      auto protocol = metadata_value(selected.value.at("protocol"));
      if (protocol == "symphony.sbv.external-census.v1")
        return external_value(
            selected.value,
            {{"source_sha256", source.sha256},
             {"dataset", source.dataset_name},
             {"instrument_id", dec(source.events.front().instrument_id)}},
            end);
      need(protocol == result_protocol &&
               !ref.at("selected_content_sha256").is_null(),
           "selected full census result required");
      return stream_census_evidence(selected.value, end);
    }
    auto selected = select_logical_result(selection, end);
    ref = selected.reference;
    return stream_census_evidence(selected.value, end);
  }();
  validate_stream_census_source(c, source, end);
  return {std::move(c), std::move(ref)};
}
} // namespace symphony::sbv::detail
