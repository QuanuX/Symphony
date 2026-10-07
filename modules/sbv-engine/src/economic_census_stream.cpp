#include "economic_census_stream.hpp"
#include "economics_kernel.hpp"
#include "exact_id_index.hpp"
#include "stream_census.hpp"
#include "stream_model.hpp"
#include <algorithm>
#include <optional>
#include <symphony/sbv/models.hpp>
namespace symphony::sbv::detail {
namespace {
using Value = logical::Value;
Json own(const Value &v, std::int64_t end) {
  return v.materialize({}, [end] { deadline(end); });
}
bool digest(const Json &v) {
  const auto s = str(v);
  return s.size() == 64 && std::all_of(s.begin(), s.end(), [](char c) {
           return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f');
         });
}
void fields(const Value &v, std::initializer_list<const char *> names) {
  need(v.kind() == "object" && v.size() == names.size(),
       "retained economic object fields mismatch");
  for (auto name : names)
    need(v.contains(name), "retained economic field missing");
}
bool equal(const Value &a, const Value &b, std::int64_t end) {
  return a.sha256({}, [end] { deadline(end); }) ==
         b.sha256({}, [end] { deadline(end); });
}
Json source_fields(const Value &choices, std::int64_t end) {
  Json out = Json::object();
  for (const auto *field :
       {"source_path", "source_sha256", "dataset", "retained_source"})
    if (choices.contains(field))
      out[field] = own(choices.at(field), end);
  return out;
}
Value external(const Value &original, const Json &provenance,
               std::int64_t end) {
  need(own(original.at("protocol"), end) == "symphony.sbv.external-census.v1",
       "legacy economics requires retained external declaration");
  return Value(Json{{"protocol", "symphony.sbv.census-evidence.v1"},
                    {"kind", "external"},
                    {"identity_domain", "external_declaration"},
                    {"census_sha256",
                     original.sha256({}, [end] { deadline(end); })},
                    {"source_sha256", provenance.at("source_sha256")},
                    {"dataset", provenance.at("dataset")},
                    {"instrument_id", provenance.at("instrument_id")},
                    {"mode", own(original.at("mode"), end)},
                    {"producer", own(original.at("producer"), end)}})
      .with("signals", original.at("signals"))
      .with("declaration", original);
}
std::string row_id(const Value &row, std::int64_t end) {
  const auto id = str(own(row.at("signal_id"), end));
  need(!id.empty() && id.size() <= 256, "bounded economic signal ID required");
  return id;
}
// A retained source reference is internal correspondence evidence, not an
// author authentication or permission to reopen that ancestor.
void retained_array(const Value &section, const Json &selection,
                    const std::string &role, std::int64_t end) {
  need(own(section.at("status"), end) == "available",
       "selected input evidence unavailable");
  const auto input = section.at("data");
  fields(input, {"role", "reference", "value"});
  const auto ref = own(input.at("reference"), end);
  need(selection.contains("kind") && selection.at("kind") == "bundle",
       "bundle selection input required");
  validate_logical_reference_lineage(selection, ref);
  need(own(input.at("role"), end) == role &&
           input.at("value").kind() == "array" &&
           ref.at("selected_content_sha256").is_null() &&
           ref.at("selected_value_sha256") ==
               input.at("value").sha256({}, [end] { deadline(end); }),
       "retained economic selection input identity mismatch");
}
void validate_expansion(const Value &sections, const Value &choices,
                        const Value &census, const Json &summary,
                        std::int64_t end) {
  const auto selected = choices.at("selections");
  const bool compact = selected.kind() == "object";
  auto expanded = [&]() -> Value {
    if (!compact) {
      need(selected.kind() == "array",
           "economic row selections array required");
      if (sections.contains("selections")) {
        need(own(sections.at("selections").at("status"), end) == "available" &&
                 equal(selected, sections.at("selections").at("data"), end),
             "duplicate expanded economic selections mismatch");
      }
      return selected;
    }
    need(own(sections.at("selections").at("status"), end) == "available",
         "expanded economic selections unavailable");
    return sections.at("selections").at("data");
  }();
  need(expanded.kind() == "array" &&
           expanded.size() == u64(summary.at("selected_signal_count")),
       "expanded economic selection count mismatch");
  need(!compact || summary.contains("selection_sha256"),
       "compact economic selections require expanded digest");
  if (summary.contains("selection_sha256"))
    need(expanded.sha256({}, [end] { deadline(end); }) ==
             str(summary.at("selection_sha256")),
         "expanded economic selection digest mismatch");
  const auto census_signals = census.at("signals");
  ExactIdIndex source_ids({}, [end] { deadline(end); });
  auto source = census_signals.children();
  while (auto row = source.next())
    need(source_ids.insert(row_id(row->value, end)).inserted,
         "duplicate census signal");
  ExactIdIndex selected_ids({}, [end] { deadline(end); });
  auto rows = expanded.children();
  while (auto row = rows.next()) {
    fields(row->value, {"signal_id", "transform", "mode", "nonexecution_pnl"});
    const auto id = row_id(row->value, end);
    need(source_ids.find(id) && selected_ids.insert(id).inserted,
         "expanded economic selections must be unique known signals");
  }
  if (!compact) {
    need(!sections.contains("selection_input"),
         "inline economic rows cannot claim a referenced selection input");
    return;
  }
  const auto kind = own(selected.at("kind"), end);
  if (kind == "referenced_rows") {
    fields(selected, {"kind", "source"});
    retained_array(sections.at("selection_input"),
                   own(selected.at("source"), end), "economic_selection_rows",
                   end);
    need(equal(expanded, sections.at("selection_input").at("data").at("value"),
               end),
         "expanded economic rows differ from retained selected rows");
    return;
  }
  fields(selected, {"kind", "signals", "economics"});
  need(kind == "apply", "unknown economic expansion kind");
  const auto economics = selected.at("economics");
  fields(economics, {"transform", "mode", "nonexecution_pnl"});
  const auto signal_selection = selected.at("signals");
  const auto signal_kind = own(signal_selection.at("kind"), end);
  auto actual = expanded.children();
  auto expected = [&](const std::string &id) {
    auto row = actual.next();
    need(row && row_id(row->value, end) == id &&
             equal(row->value.without("signal_id"), economics, end),
         "economic expansion differs from declared selected order or template");
  };
  if (signal_kind == "all" || signal_kind == "range") {
    need(!sections.contains("selection_input"),
         "unreferenced economic selection cannot claim input evidence");
    std::uint64_t first = 0, count = census_signals.size();
    if (signal_kind == "all")
      fields(signal_selection, {"kind"});
    else {
      fields(signal_selection, {"kind", "first", "count"});
      first = u64(own(signal_selection.at("first"), end));
      need(first <= census_signals.size(),
           "economic range starts outside census");
      const auto selected_count = own(signal_selection.at("count"), end);
      count = selected_count.is_null() ? census_signals.size() - first
                                       : u64(selected_count);
      need(count <= census_signals.size() - first,
           "economic range extends outside census");
    }
    auto cursor = census_signals.children(first, count);
    while (auto row = cursor.next())
      expected(row_id(row->value, end));
  } else {
    const auto wanted = [&]() -> Value {
      if (signal_kind == "ids") {
        fields(signal_selection, {"kind", "ids", "order"});
        need(!sections.contains("selection_input"),
             "inline economic IDs cannot claim referenced evidence");
        return signal_selection.at("ids");
      }
      fields(signal_selection, {"kind", "source", "order"});
      need(signal_kind == "referenced_ids", "unknown selected signal kind");
      retained_array(sections.at("selection_input"),
                     own(signal_selection.at("source"), end), "signal_ids",
                     end);
      return sections.at("selection_input").at("data").at("value");
    }();
    need(wanted.kind() == "array", "selected economic IDs array required");
    const auto order = own(signal_selection.at("order"), end);
    need(order == "census" || order == "supplied",
         "explicit economic selected order required");
    ExactIdIndex wanted_ids({}, [end] { deadline(end); });
    auto wanted_cursor = wanted.children();
    while (auto value = wanted_cursor.next()) {
      const auto id = str(own(value->value, end));
      need(source_ids.find(id) && wanted_ids.insert(id).inserted,
           "selected economic IDs must be known and unique");
      if (order == "supplied")
        expected(id);
    }
    if (order == "census") {
      auto cursor = census_signals.children();
      while (auto row = cursor.next()) {
        const auto id = row_id(row->value, end);
        if (wanted_ids.find(id))
          expected(id);
      }
    }
  }
  need(!actual.next(), "economic expansion contains undeclared rows");
}
} // namespace

logical::Value economic_stream_census(const Value &result, std::int64_t end) {
  deadline(end);
  const auto sections = result.at("sections"),
             context_section = sections.at("source_context"),
             context = context_section.at("data"),
             choices = sections.at("choices").at("data"),
             parent_choices = context.at("choices").at("data");
  const auto provenance = own(sections.at("provenance").at("data"), end),
             summary = own(sections.at("summary").at("data"), end),
             parent_provenance = own(context.at("provenance").at("data"), end);
  need(own(context_section.at("status"), end) == "available" &&
           sections.contains("replay") &&
           own(context.at("choices").at("status"), end) == "available" &&
           own(context.at("provenance").at("status"), end) == "available" &&
           own(choices.at("protocol"), end) ==
               "symphony.sbv.economics-input.v1" &&
           own(parent_choices.at("protocol"), end) ==
               "symphony.sbv.evaluate-input.v1",
       "economic census, replay and original evaluation context required");
  const auto census = [&]() -> Value {
    if (!sections.contains("census"))
      return external(context.at("census"), parent_provenance, end);
    need(own(sections.at("census").at("status"), end) == "available" &&
             equal(sections.at("census").at("data"), context.at("census"), end),
         "economic census/context mismatch");
    return sections.at("census").at("data");
  }();
  validate_stream_census(census, end);
  const auto original_identity = dataset_result_identity(
      source_fields(parent_choices, end), parent_provenance,
      context.contains("dataset_feed")
          ? Json{{"dataset_feed", own(context.at("dataset_feed"), end)}}
          : Json(nullptr));
  need(own(census.at("census_sha256"), end) ==
               summary.at("source_census_sha256") &&
           own(census.at("source_sha256"), end) ==
               original_identity.at("source_sha256") &&
           own(census.at("source_sha256"), end) ==
               parent_provenance.at("source_sha256") &&
           own(census.at("dataset"), end) == original_identity.at("dataset") &&
           own(census.at("dataset"), end) == parent_provenance.at("dataset") &&
           own(census.at("instrument_id"), end) ==
               parent_provenance.at("instrument_id") &&
           own(census.at("census_sha256"), end) ==
               parent_provenance.at("census_sha256") &&
           own(census.at("producer"), end) ==
               parent_provenance.at("census_producer"),
       "economic retained census lineage mismatch");
  const auto parent_census = parent_choices.at("census");
  if (parent_census.contains("protocol")) {
    need(!context.contains("census_reference") &&
             own(census.at("kind"), end) == "external" &&
             equal(parent_census, census.at("declaration"), end),
         "economic original inline census selection mismatch");
  } else {
    const auto selection = own(parent_census, end);
    if (context.contains("census_reference")) {
      const auto ref = own(context.at("census_reference"), end);
      validate_logical_reference_lineage(selection, ref);
      if (selection.contains("kind") &&
          ref.at("selected_content_sha256").is_null())
        need(own(census.at("kind"), end) == "external" &&
                 ref.at("selected_value_sha256") ==
                     own(census.at("census_sha256"), end),
             "economic external declaration reference identity mismatch");
    } else {
      // Historical economics retained only the legacy census reference. Keep
      // that declared profile; never invent a modern full-closure reference.
      keys(selection, {"path", "expected_sha256", "pointer"});
      need(digest(selection.at("expected_sha256")),
           "economic original census reference digest invalid");
      (void)Json::json_pointer(str(selection.at("pointer")));
    }
  }
  const bool historical_context =
      !context.contains("reference") && !choices.contains("source") &&
      !provenance.contains("source_outer_content_sha256");
  const Json source_selection =
      choices.contains("source")
          ? own(choices.at("source"), end)
          : Json{{"path", own(choices.at("path"), end)},
                 {"expected_sha256", own(choices.at("expected_sha256"), end)},
                 {"pointer", ""}};
  need(digest(provenance.at("source_content_sha256")),
       "economic parent logical digest invalid");
  if (context.contains("reference")) {
    const auto ref = own(context.at("reference"), end);
    validate_logical_reference_lineage(source_selection, ref);
    need(ref.at("selected_content_sha256") ==
             provenance.at("source_content_sha256"),
         "economic parent selected identity mismatch");
    if (source_selection.contains("kind")) {
      need(ref == provenance.at("source_reference") &&
               ref.at("reference").at("content_sha256") ==
                   provenance.at("source_outer_content_sha256"),
           "economic bundle parent provenance mismatch");
    } else {
      need(digest(provenance.at("source_file_sha256")) &&
               ref.at("path") == provenance.at("source_path") &&
               ref.at("pointer") == provenance.at("source_pointer") &&
               ref.at("expected_sha256") ==
                   provenance.at("source_outer_content_sha256") &&
               ref.at("file_sha256") == provenance.at("source_file_sha256") &&
               (!str(ref.at("pointer")).empty() ||
                ref.at("expected_sha256") == ref.at("selected_content_sha256")),
           "economic file parent provenance mismatch");
      (void)Json::json_pointer(str(ref.at("pointer")));
    }
  } else {
    need(historical_context && digest(provenance.at("source_file_sha256")) &&
             own(choices.at("path"), end) == provenance.at("source_path") &&
             own(choices.at("expected_sha256"), end) ==
                 provenance.at("source_content_sha256"),
         "economic legacy parent reference mismatch");
  }
  const auto signals = sections.at("signals").at("data"),
             models = sections.at("execution").at("data"),
             outcomes = sections.at("economics").at("data");
  need(signals.kind() == "array" && models.kind() == "array" &&
           outcomes.kind() == "array" &&
           u64(summary.at("source_signal_count")) ==
               census.at("signals").size() &&
           u64(summary.at("selected_signal_count")) == outcomes.size() &&
           signals.size() == outcomes.size() &&
           models.size() == outcomes.size(),
       "economic census/selection counts mismatch");
  validate_expansion(sections, choices, census, summary, end);
  const auto selected_model = parent_choices.at("model");
  const auto selected_id = own(selected_model.at("id"), end),
             selected_version = own(selected_model.at("version"), end);
  const bool native = selected_id == "native_provider",
             external_model = selected_id == "external_outcomes";
  need(!context.contains("provider") || native,
       "unexpected retained economic model provider evidence");
  need(!context.contains("model_inputs") || external_model,
       "unexpected retained economic model input evidence");
  Json native_selection, native_context;
  std::unique_ptr<StreamModel> external_input;
  if (native) {
    native_selection = own(selected_model, end);
    native_context = own(context.at("provider"), end);
    validate_native_provider_model_context(native_selection, native_context,
                                           Json::array());
  } else if (external_model) {
    if (context.contains("model_inputs")) {
      const auto inputs = context.at("model_inputs");
      fields(inputs, {"outcomes", "reference"});
      external_input = std::make_unique<StreamModel>(
          selected_model, census.at("signals"), inputs.at("outcomes"),
          own(inputs.at("reference"), end), end);
    } else {
      const auto parameters = selected_model.at("parameters");
      need(parameters.contains("outcomes"),
           "referenced model requires retained original input rows");
      external_input = std::make_unique<StreamModel>(
          selected_model, census.at("signals"), parameters.at("outcomes"),
          Json(nullptr), end);
    }
  } else {
    StreamModel admission(own(selected_model, end), census.at("signals"), end);
  }
  ExactIdIndex census_ids({}, [end] { deadline(end); }),
      economic_ids({}, [end] { deadline(end); }),
      model_ids({}, [end] { deadline(end); }),
      selected_signal_ids({}, [end] { deadline(end); });
  auto census_rows = census.at("signals").children();
  while (auto row = census_rows.next())
    need(census_ids.insert(row_id(row->value, end)).inserted,
         "duplicate original economic census signal");
  auto signal_rows = signals.children();
  while (auto row = signal_rows.next()) {
    const auto id = row_id(row->value, end);
    const auto ordinal = census_ids.find(id);
    need(ordinal && selected_signal_ids.insert(id).inserted &&
             equal(row->value, census.at("signals").at(*ordinal), end),
         "economic retained signal differs from original census");
  }
  const auto expanded = choices.at("selections").kind() == "array"
                            ? choices.at("selections")
                            : sections.at("selections").at("data");
  auto selections = expanded.children();
  while (auto row = selections.next())
    need(selected_signal_ids.find(row_id(row->value, end)).has_value(),
         "economic selected signals differ from expanded choices");
  auto model_rows = models.children();
  while (auto row = model_rows.next()) {
    const auto model = own(row->value, end);
    const auto id = str(model.at("signal_id"));
    need(selected_signal_ids.find(id) && model_ids.insert(id).inserted &&
             model.at("protocol") == "symphony.sbv.model-outcome.v1" &&
             model.at("model_id") == selected_id &&
             model.at("model_version") == selected_version,
         "economic retained model identity mismatch");
    validate_economic_source_model(model);
    if (external_input)
      validate_retained_external_outcome(model, *external_input,
                                         historical_context, end);
    if (native)
      validate_native_provider_model_context(native_selection, native_context,
                                             Json::array({model}));
    else if (external_model) {
      const auto parameters = selected_model.at("parameters");
      for (const auto *name :
           {"evidence_origin", "producer", "calibration_reference"}) {
        need(historical_context || model.contains(name),
             "economic model attribution field missing");
        if (model.contains(name))
          need(model.at(name) == (std::string(name) == "evidence_origin"
                                      ? Json("externally_supplied")
                                      : own(parameters.at(name), end)),
               "economic external model attribution mismatch");
      }
      need(model.at("measure") == own(parameters.at("measure"), end) &&
               model.at("conditioning") ==
                   own(parameters.at("conditioning"), end),
           "economic external model domain mismatch");
    } else {
      need(historical_context || model.contains("evidence_origin"),
           "economic model attribution field missing");
      need((!model.contains("evidence_origin") ||
            model.at("evidence_origin") == "observed_post_signal_trades") &&
               model.at("measure") == "probability" &&
               model.at("conditioning") == "execution",
           "economic observed model attribution mismatch");
    }
  }
  auto rows = outcomes.children();
  while (auto row = rows.next()) {
    const auto id = row_id(row->value, end);
    need(selected_signal_ids.find(id) && economic_ids.insert(id).inserted,
         "economic retained outcome signal mismatch");
  }
  need(economic_ids.size() == selected_signal_ids.size() &&
           model_ids.size() == selected_signal_ids.size(),
       "economic retained signal sets mismatch");
  return census;
}
} // namespace symphony::sbv::detail
