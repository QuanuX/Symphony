#include "economics_kernel.hpp"
#include "exact_id_index.hpp"
#include "partitioned_output.hpp"
#include "stream_census.hpp"
#include "stream_model.hpp"
#include <limits>
#include <optional>
#include <symphony/sbv/models.hpp>

namespace symphony::sbv::detail {
namespace {
using Value = logical::Value;
Json owned(const Value &value, std::int64_t end) {
  return value.materialize({}, [end] { deadline(end); });
}
std::string id(const Json &value) {
  const auto result = str(value);
  need(!result.empty() && result.size() <= 256,
       "nonempty bounded signal identity required");
  return result;
}
Value section_value(Value value, const std::string &status = "available",
                    const std::string &reason = "") {
  return Value(section(nullptr, status, reason)).with("data", std::move(value));
}
std::uint64_t plus(std::uint64_t a, std::uint64_t b) {
  need(b <= UINT64_MAX - a, "economic counter representation exceeded");
  return a + b;
}
void retained_model_attribution(const Json &model, const Json &selected,
                                const Json &captured) {
  need(model.at("model_id") == selected.at("id") &&
           model.at("model_version") == selected.at("version"),
       "retained model selection identity mismatch");
  if (selected.at("id") == "native_provider") {
    validate_native_provider_model_context(selected, captured,
                                           Json::array({model}));
  } else if (selected.at("id") == "external_outcomes") {
    const auto &parameters = selected.at("parameters");
    need(model.at("evidence_origin") == "externally_supplied" &&
             model.at("producer") == parameters.at("producer") &&
             model.at("measure") == parameters.at("measure") &&
             model.at("conditioning") == parameters.at("conditioning") &&
             model.at("calibration_reference") ==
                 parameters.at("calibration_reference"),
         "retained external model attribution mismatch");
  } else {
    need(selected.at("id") == "observed_trade_levels" &&
             model.at("evidence_origin") == "observed_post_signal_trades" &&
             model.at("measure") == "probability" &&
             model.at("conditioning") == "execution",
         "retained observed model attribution mismatch");
  }
}
// Extract only the bounded installed model metadata. The original supplied
// rows stay in their owning Value and are checked by the retained StreamModel.
Json model_metadata(const Value &selected, std::int64_t end) {
  need(selected.kind() == "object" && selected.size() == 5,
       "retained model selection shape");
  Json result = Json::object();
  for (const auto *key : {"protocol", "id", "version", "horizon_ns"})
    result[key] = owned(selected.at(key), end);
  const auto parameters = selected.at("parameters");
  if (result.at("id") == "external_outcomes") {
    Json p = Json::object();
    for (const auto *key :
         {"producer", "measure", "conditioning", "calibration_reference"})
      p[key] = owned(parameters.at(key), end);
    // No metadata or aggregate input limit is added here. Full exact keys and
    // input-array correspondence are checked by the retained model adapter.
    result["parameters"] = std::move(p);
  } else
    result["parameters"] = owned(parameters, end);
  return result;
}
} // namespace

Json economics_partitioned(const Json &p, std::int64_t end) {
  keys_optional(p,
                {"protocol", "output", "selections", "studies",
                 "on_incompatible", "extensions"},
                {"path", "expected_sha256", "source"});
  need(p.at("protocol") == "symphony.sbv.economics-input.v1",
       "economic request protocol mismatch");
  const bool referenced = p.contains("source");
  need(referenced ? !p.contains("path") && !p.contains("expected_sha256")
                  : p.contains("path") && p.contains("expected_sha256"),
       "select exactly one economic source reference form");
  const Json selected_source =
      referenced ? p.at("source")
                 : Json{{"path", p.at("path")},
                        {"expected_sha256", p.at("expected_sha256")},
                        {"pointer", ""}};
  need(p.at("extensions").is_object(), "extensions object required");
  const auto policy = str(p.at("on_incompatible"));
  need(policy == "unavailable" || policy == "reject",
       "explicit economic domain policy required");
  const bool reject = policy == "reject";
  const auto studies = economic_study_selections(p.at("studies"));

  return partitioned_result(
      p, "economics", end, [&](PartitionedOutput &output) {
        auto admitted = select_logical_result(selected_source, end);
        const auto source = admitted.value;
        const auto sections = source.at("sections");
        const auto parent_choices = sections.at("choices").at("data");
        need(owned(parent_choices.at("protocol"), end) ==
                 "symphony.sbv.evaluate-input.v1",
             "economic source must be a complete evaluation result");
        const auto census = stream_census_evidence(source, end);
        const auto signals = census.at("signals");
        const auto models = sections.at("execution").at("data");
        need(signals.kind() == "array" && models.kind() == "array" &&
                 signals.size() == models.size(),
             "source signal/model correspondence required");
        ExactIdIndex signal_index({}, output.checkpoint());
        auto signal_rows = signals.children();
        while (auto signal = signal_rows.next()) {
          deadline(end);
          need(
              signal_index.insert(id(owned(signal->value.at("signal_id"), end)))
                  .inserted,
              "duplicate source signal");
        }

        std::optional<Json> parent_model;
        Json provider_context = nullptr;
        std::unique_ptr<StreamModel> input_model;
        const bool has_model = parent_choices.contains("model");
        bool native_provider = false;
        if (has_model) {
          const auto chosen = parent_choices.at("model");
          parent_model = model_metadata(chosen, end);
          native_provider = parent_model->at("id") == "native_provider";
          if (native_provider) {
            need(sections.contains("provider") &&
                     owned(sections.at("provider").at("status"), end) ==
                         "available",
                 "retained model provider evidence required");
            provider_context = owned(sections.at("provider").at("data"), end);
            validate_native_provider_model_context(
                *parent_model, provider_context, Json::array());
          } else if (parent_model->at("id") == "external_outcomes") {
            const auto parameters = chosen.at("parameters");
            if (sections.contains("model_inputs")) {
              const auto inputs_section = sections.at("model_inputs");
              need(owned(inputs_section.at("status"), end) == "available",
                   "retained model inputs must be available");
              const auto inputs = inputs_section.at("data");
              need(inputs.kind() == "object" && inputs.size() == 2 &&
                       inputs.contains("outcomes") &&
                       inputs.contains("reference"),
                   "retained model input shape");
              input_model = std::make_unique<StreamModel>(
                  chosen, signals, inputs.at("outcomes"),
                  owned(inputs.at("reference"), end), end);
            } else {
              need(parameters.contains("outcomes"),
                   "referenced model outcomes require retained input evidence");
              input_model = std::make_unique<StreamModel>(
                  chosen, signals, parameters.at("outcomes"), Json(nullptr),
                  end);
            }
          } else
            input_model =
                std::make_unique<StreamModel>(*parent_model, signals, end);
        }
        need(!sections.contains("provider") || native_provider,
             "unexpected model provider evidence");
        need(
            !sections.contains("model_inputs") ||
                (parent_model && parent_model->at("id") == "external_outcomes"),
            "unexpected retained model input evidence");

        ExactIdIndex model_index({}, output.checkpoint());
        auto model_rows = models.children();
        while (auto row = model_rows.next()) {
          deadline(end);
          const auto model = owned(row->value, end);
          const auto signal_id = id(model.at("signal_id"));
          need(model.at("protocol") == "symphony.sbv.model-outcome.v1" &&
                   signal_index.find(signal_id).has_value() &&
                   model_index.insert(signal_id).inserted,
               "source model identity mismatch");
          validate_economic_source_model(model);
          if (parent_model)
            retained_model_attribution(model, *parent_model, provider_context);
          if (input_model && parent_model->at("id") == "external_outcomes")
            validate_retained_external_outcome(model, *input_model, false, end);
        }
        need(model_index.size() == signal_index.size(),
             "source model signal set mismatch");

        auto economic_spool = output.spool("economic_rows");
        auto study_spool = output.spool("economic_studies");
        auto signal_spool = output.spool("selected_signals");
        auto model_spool = output.spool("selected_models");
        auto selection_spool = output.spool("expanded_selections");
        ExactIdIndex selected_ids({}, output.checkpoint());
        std::uint64_t unavailable = 0, unavailable_studies = 0;
        auto apply = [&](const Json &choice) {
          deadline(end);
          keys(choice, {"signal_id", "transform", "mode", "nonexecution_pnl"});
          const auto signal_id = id(choice.at("signal_id"));
          const auto signal_at = signal_index.find(signal_id);
          const auto model_at = model_index.find(signal_id);
          need(signal_at && model_at && selected_ids.insert(signal_id).inserted,
               "unique known selected signal required");
          const auto signal = signals.at(*signal_at);
          const auto model_value = models.at(*model_at);
          const auto model = owned(model_value, end);
          need(owned(signal.at("signal_id"), end) == signal_id &&
                   model.at("signal_id") == signal_id,
               "source row identity changed after admission");
          auto row = economic_row(choice, model, studies, reject, end);
          if (row.outcome.at("status") == "unavailable")
            unavailable = plus(unavailable, 1);
          unavailable_studies =
              plus(unavailable_studies, row.unavailable_studies);
          economic_spool->append(row.outcome);
          for (const auto &study : row.studies)
            study_spool->append(study);
          signal_spool->append(owned(signal, end));
          model_spool->append(model);
          selection_spool->append(choice);
        };
        std::optional<Value> selection_input;
        const auto &selections = p.at("selections");
        if (selections.is_array()) {
          for (const auto &choice : selections)
            apply(choice);
        } else {
          need(selections.is_object() && selections.contains("kind"),
               "economic selection declaration required");
          if (selections.at("kind") == "referenced_rows") {
            keys(selections, {"kind", "source"});
            auto rows = select_bundle_value(selections.at("source"), end);
            need(rows.value.kind() == "array",
                 "selected economic rows array required");
            selection_input =
                Value::object({{"role", Value(Json("economic_selection_rows"))},
                               {"reference", Value(rows.reference)},
                               {"value", rows.value}});
            auto cursor = rows.value.children();
            while (auto choice = cursor.next())
              apply(owned(choice->value, end));
          } else {
            keys(selections, {"kind", "signals", "economics"});
            need(selections.at("kind") == "apply",
                 "unknown economic selection declaration");
            const auto &economics = selections.at("economics");
            keys(economics, {"transform", "mode", "nonexecution_pnl"});
            auto apply_id = [&](const std::string &signal_id) {
              auto choice = economics;
              choice["signal_id"] = signal_id;
              apply(choice);
            };
            const auto &chosen = selections.at("signals");
            need(chosen.is_object() && chosen.contains("kind"),
                 "signal selection required");
            if (chosen.at("kind") == "all" || chosen.at("kind") == "range") {
              std::uint64_t first = 0, count = signals.size();
              if (chosen.at("kind") == "all")
                keys(chosen, {"kind"});
              else {
                keys(chosen, {"kind", "first", "count"});
                first = u64(chosen.at("first"));
                need(first <= signals.size(),
                     "economic range starts outside census");
                count = chosen.at("count").is_null() ? signals.size() - first
                                                     : u64(chosen.at("count"));
                need(count <= signals.size() - first,
                     "economic range extends outside census");
              }
              auto cursor = signals.children(first, count);
              while (auto row = cursor.next())
                apply_id(id(owned(row->value.at("signal_id"), end)));
            } else {
              std::optional<Value> wanted;
              if (chosen.at("kind") == "ids") {
                keys(chosen, {"kind", "ids", "order"});
                wanted.emplace(chosen.at("ids"));
              } else {
                keys(chosen, {"kind", "source", "order"});
                need(chosen.at("kind") == "referenced_ids",
                     "unknown signal selection kind");
                auto input = select_bundle_value(chosen.at("source"), end);
                wanted = input.value;
                selection_input =
                    Value::object({{"role", Value(Json("signal_ids"))},
                                   {"reference", Value(input.reference)},
                                   {"value", input.value}});
              }
              need(wanted->kind() == "array",
                   "selected signal IDs array required");
              const auto order = str(chosen.at("order"));
              need(order == "supplied" || order == "census",
                   "explicit selected signal order required");
              ExactIdIndex wanted_ids({}, output.checkpoint());
              auto cursor = wanted->children();
              while (auto value = cursor.next()) {
                const auto signal_id = id(owned(value->value, end));
                need(signal_index.find(signal_id).has_value() &&
                         wanted_ids.insert(signal_id).inserted,
                     "requested signal IDs must be unique and known");
                if (order == "supplied")
                  apply_id(signal_id);
              }
              if (order == "census") {
                auto ordered = signals.children();
                while (auto row = ordered.next()) {
                  const auto signal_id =
                      id(owned(row->value.at("signal_id"), end));
                  if (wanted_ids.find(signal_id))
                    apply_id(signal_id);
                }
              }
            }
          }
        }
        const auto outcomes = economic_spool->close();
        const auto study_rows = study_spool->close();
        const auto kept_signals = signal_spool->close();
        const auto kept_models = model_spool->close();
        const auto expanded = selection_spool->close();
        const auto selected_count = outcomes.row_count;
        const auto census_hash = owned(census.at("census_sha256"), end);
        auto result = Value(base("native exact user-selected linear economics "
                                 "of admitted model support"));
        result =
            result.with("status", Value(Json(unavailable || unavailable_studies
                                                 ? "partial"
                                                 : "completed")));
        auto out = result.at("sections");
        out = out.with(
            "summary",
            Value(section({{"source_census_sha256", census_hash},
                           {"source_signal_count", dec(signals.size())},
                           {"selected_signal_count", dec(selected_count)},
                           {"available_economic_outcomes",
                            dec(selected_count - unavailable)},
                           {"unavailable_economic_outcomes", dec(unavailable)},
                           {"unavailable_studies", dec(unavailable_studies)},
                           {"selection_sha256", expanded.array_verification.at(
                                                    "canonical_sha256")}})));
        out = out.with("economics",
                       section_value(Value(outcomes.rows),
                                     unavailable ? "partial" : "available",
                                     unavailable
                                         ? "selected outcomes unavailable; "
                                           "inspect per-signal reasons"
                                         : ""));
        out = out.with(
            "studies",
            section_value(
                Value(study_rows.rows),
                unavailable_studies ? "partial" : "available",
                unavailable_studies
                    ? "selected studies unavailable; inspect per-signal reasons"
                    : ""));
        out = out.with("signals", section_value(Value(kept_signals.rows)));
        out = out.with("execution", section_value(Value(kept_models.rows)));
        out = out.with("replay", sections.at("replay"));
        out = out.with("census", section_value(census));
        if (!selections.is_array())
          out = out.with("selections", section_value(Value(expanded.rows)));
        if (selection_input)
          out = out.with("selection_input", section_value(*selection_input));
        auto context =
            Value::object({{"census", census},
                           {"choices", sections.at("choices")},
                           {"provenance", sections.at("provenance")},
                           {"reference", Value(admitted.reference)}});
        const auto resources = sections.at("resources").at("data");
        if (resources.kind() == "object" && resources.contains("dataset_feed"))
          context = context.with("dataset_feed", resources.at("dataset_feed"));
        if (sections.contains("census_reference")) {
          const auto reference_section = sections.at("census_reference");
          if (owned(reference_section.at("status"), end) == "available")
            context =
                context.with("census_reference", reference_section.at("data"));
          else
            need(owned(reference_section.at("status"), end) == "not_selected" &&
                     owned(reference_section.at("data"), end).is_null() &&
                     parent_choices.at("census").contains("protocol"),
                 "retained census reference must be available when selected");
        }
        if (native_provider)
          context =
              context.with("provider", sections.at("provider").at("data"));
        if (sections.contains("model_inputs"))
          context = context.with("model_inputs",
                                 sections.at("model_inputs").at("data"));
        out = out.with("source_context", section_value(context));
        auto choices = p;
        choices.erase("output");
        out = out.with("choices", Value(section(choices)));
        Json provenance{
            {"engine_version", version},
            {"source_content_sha256",
             admitted.reference.at("selected_content_sha256")},
            {"source_authorship", "not_verified"},
            {"replay_scope", "full source replay copied verbatim"},
            {"numeric_contract",
             "exact reduced int64 rationals; overflow rejects; no rounding"}};
        if (selected_source.contains("kind")) {
          provenance["source_reference"] = admitted.reference;
          provenance["source_outer_content_sha256"] =
              selected_source.at("reference").at("content_sha256");
        } else {
          provenance["source_path"] = selected_source.at("path");
          provenance["source_pointer"] = selected_source.at("pointer");
          provenance["source_outer_content_sha256"] =
              selected_source.at("expected_sha256");
          provenance["source_file_sha256"] =
              admitted.reference.at("file_sha256");
        }
        out = out.with("provenance", Value(section(provenance)));
        out = out.with(
            "resources",
            Value(section(
                {{"backend", "cpu"},
                 {"workers", "1"},
                 {"derived_resources",
                  {{"source_signal_id_index", signal_index.evidence()},
                   {"source_model_id_index", model_index.evidence()},
                   {"selected_id_index", selected_ids.evidence()},
                   {"model_input_admission",
                    input_model ? input_model->evidence() : Json(nullptr)},
                   {"spooled_arrays", "5"},
                   {"aggregate_state", "exact in-memory ID indices; output "
                                       "rows retained in private paged spools"},
                   {"storage", "self-contained partitioned result"}}}})));
        out = out.with(
            "diagnostics",
            Value(section(Json::array(
                {"Per-signal scenario economics only; no cross-signal "
                 "independence, trade pairing, portfolio or account simulation "
                 "is inferred.",
                 "Position, value, costs and return basis are explicit user "
                 "assumptions; calibration is not verified."}))));
        out = out.with("user_extensions", Value(section(p.at("extensions"))));
        return result.with("sections", out);
      });
}
} // namespace symphony::sbv::detail
