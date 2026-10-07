#include "stream_model.hpp"
#include "detail.hpp"
#include "exact_id_index.hpp"
#include "stream_census.hpp"
#include <algorithm>
#include <mutex>
#include <optional>
#include <symphony/knowledge/engine/path.hpp>

namespace symphony::sbv::detail {
namespace {
using Value = logical::Value;
void logical_keys(const Value &value,
                  std::initializer_list<const char *> names) {
  need(value.kind() == "object" && value.size() == names.size(),
       "model row object fields mismatch");
  for (const auto *name : names)
    need(value.contains(name), "model row field missing");
}
Json scalar(const Value &value, std::int64_t end) {
  need(value.kind() == "scalar", "model row scalar required");
  return value.materialize({}, [end] { deadline(end); });
}
std::string identity(const Value &value, std::int64_t end) {
  const auto name = str(scalar(value, end));
  need(!name.empty() && name.size() <= 256,
       "bounded nonempty model signal identity required");
  return name;
}
Json ratio(const Value &value, std::int64_t end) {
  logical_keys(value, {"numerator", "denominator"});
  return {{"numerator", scalar(value.at("numerator"), end)},
          {"denominator", scalar(value.at("denominator"), end)}};
}
Json scalar_fields(const Value &value,
                   std::initializer_list<const char *> names,
                   std::int64_t end) {
  logical_keys(value, names);
  Json result = Json::object();
  for (const auto *name : names)
    result[name] = scalar(value.at(name), end);
  return result;
}
Json source_selection(const Value &value, std::int64_t end) {
  logical_keys(value, {"kind", "reference", "selector", "read_options"});
  const auto selector = value.at("selector");
  need(selector.kind() == "object" && selector.contains("kind"),
       "model source selector required");
  const auto kind = scalar(selector.at("kind"), end);
  const auto selected = kind == "node_id"
                            ? scalar_fields(selector, {"kind", "node_id"}, end)
                            : scalar_fields(selector, {"kind", "pointer"}, end);
  return {
      {"kind", scalar(value.at("kind"), end)},
      {"reference",
       scalar_fields(value.at("reference"),
                     {"manifest_path", "manifest_sha256", "content_sha256"},
                     end)},
      {"selector", selected},
      {"read_options", scalar_fields(value.at("read_options"),
                                     {"max_page_bytes", "cache_bytes"}, end)}};
}
struct Selection {
  Json metadata;
  std::optional<Value> inline_outcomes;
};
Selection retained_selection(const Value &value, std::int64_t end) {
  deadline(end);
  logical_keys(value,
               {"protocol", "id", "version", "horizon_ns", "parameters"});
  Json metadata = Json::object();
  for (const auto *name : {"protocol", "id", "version", "horizon_ns"})
    metadata[name] = scalar(value.at(name), end);
  const auto parameters = value.at("parameters");
  if (metadata.at("id") == "observed_trade_levels") {
    metadata["parameters"] = scalar_fields(
        parameters, {"levels_per_side", "include_anchor", "thin_support"}, end);
    return {std::move(metadata), {}};
  }
  need(metadata.at("id") == "external_outcomes",
       "retained stream model supports external or built-in observed models");
  need(parameters.kind() == "object" && parameters.size() == 5 &&
           parameters.contains("producer") && parameters.contains("measure") &&
           parameters.contains("conditioning") &&
           parameters.contains("calibration_reference"),
       "retained model parameters mismatch");
  const bool inline_rows = parameters.contains("outcomes");
  need(inline_rows != parameters.contains("outcomes_source"),
       "retained model requires exactly one outcome source");
  Json compact{{"producer", scalar_fields(parameters.at("producer"),
                                          {"id", "version", "artifact_sha256",
                                           "reproducibility"},
                                          end)},
               {"measure", scalar(parameters.at("measure"), end)},
               {"conditioning", scalar(parameters.at("conditioning"), end)},
               {"calibration_reference",
                scalar(parameters.at("calibration_reference"), end)}};
  std::optional<Value> original;
  if (inline_rows) {
    original = parameters.at("outcomes");
    compact["outcomes"] = Json::array();
  } else
    compact["outcomes_source"] =
        source_selection(parameters.at("outcomes_source"), end);
  metadata["parameters"] = std::move(compact);
  return {std::move(metadata), std::move(original)};
}
// Read only the exact bounded row shape before materializing. Reject a giant
// unexpected object/array without constructing its aggregate JSON value.
// Numerical/measure admission is deliberately delegated to AdmittedModel.
Json row(const Value &value, std::int64_t end) {
  deadline(end);
  logical_keys(value, {"signal_id", "support", "execution_probability",
                       "evidence_reference"});
  const auto signal = identity(value.at("signal_id"), end);
  const auto support = value.at("support");
  need(support.kind() == "array" && support.size() >= 1 && support.size() <= 64,
       "support bound 1..64");
  Json atoms = Json::array();
  auto children = support.children();
  while (auto atom = children.next()) {
    deadline(end);
    logical_keys(atom->value, {"price_nanos", "weight"});
    atoms.push_back(
        {{"price_nanos", scalar(atom->value.at("price_nanos"), end)},
         {"weight", ratio(atom->value.at("weight"), end)}});
  }
  const auto execution = value.at("execution_probability");
  need(execution.kind() == "object" && execution.contains("status"),
       "execution probability status required");
  const auto status = scalar(execution.at("status"), end);
  Json fill;
  if (status == "supplied") {
    logical_keys(execution, {"status", "value"});
    fill = {{"status", status}, {"value", ratio(execution.at("value"), end)}};
  } else {
    logical_keys(execution, {"status", "reason"});
    fill = {{"status", status},
            {"reason", scalar(execution.at("reason"), end)}};
  }
  return {{"signal_id", signal},
          {"support", std::move(atoms)},
          {"execution_probability", std::move(fill)},
          {"evidence_reference", scalar(value.at("evidence_reference"), end)}};
}
bool digest(const Json &value) {
  if (!value.is_string())
    return false;
  const auto &s = value.get_ref<const std::string &>();
  return s.size() == 64 && std::all_of(s.begin(), s.end(), [](char c) {
           return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f');
         });
}
void safe_retained_path(const Json &value) {
  const auto name = str(value);
  need(name.size() > 1 && name.size() <= 4096 && name.front() == '/' &&
           e::is_safe_relative_path(name.substr(1)),
       "absolute clean retained source path required");
}
// No filesystem access. These declarations prove internal correspondence only;
// the already-admitted enclosing result supplies their retained provenance.
void retained_reference(const Json &selection, const Json &reference,
                        const Value &outcomes, std::int64_t end) {
  keys(selection, {"kind", "reference", "selector", "read_options"});
  keys(reference,
       {"kind", "reference", "selector", "read_options", "selected_node_id",
        "selected_value_sha256", "selected_content_sha256",
        "verification_extent", "authorship"});
  need(selection.at("kind") == "bundle", "bundle outcome source required");
  for (const auto *field : {"kind", "reference", "selector", "read_options"})
    need(reference.at(field) == selection.at(field),
         "retained model source selection mismatch");
  const auto &source = selection.at("reference");
  keys(source, {"manifest_path", "manifest_sha256", "content_sha256"});
  safe_retained_path(source.at("manifest_path"));
  need(digest(source.at("manifest_sha256")) &&
           digest(source.at("content_sha256")),
       "retained model source digest required");
  const auto &options = selection.at("read_options");
  keys(options, {"max_page_bytes", "cache_bytes"});
  (void)u64(options.at("cache_bytes"));
  if (!options.at("max_page_bytes").is_null())
    need(u64(options.at("max_page_bytes")) > 0,
         "positive retained source page budget required");
  const auto selected = u64(reference.at("selected_node_id"));
  need(selected >= 2,
       "an outcome array cannot be the outer result or virtual digest node");
  const auto &selector = selection.at("selector");
  need(selector.is_object() && selector.contains("kind"),
       "retained source selector required");
  if (selector.at("kind") == "node_id") {
    keys(selector, {"kind", "node_id"});
    need(u64(selector.at("node_id")) == selected,
         "retained selected node mismatch");
  } else {
    keys(selector, {"kind", "pointer"});
    need(selector.at("kind") == "pointer", "retained selector kind mismatch");
    const auto pointer = str(selector.at("pointer"));
    (void)Json::json_pointer(pointer);
    need(!pointer.empty() && pointer != "/content_sha256",
         "retained outcome pointer must select an array role");
  }
  need(reference.at("selected_content_sha256").is_null() &&
           reference.at("verification_extent") ==
               "full_outer_logical_closure" &&
           reference.at("authorship") == "not_verified" &&
           digest(reference.at("selected_value_sha256")) &&
           reference.at("selected_value_sha256") ==
               outcomes.sha256({}, [end] { deadline(end); }),
       "retained outcome source correspondence mismatch");
}
} // namespace

struct StreamModel::Impl {
  Json selection, reference = nullptr;
  std::optional<Value> supplied;
  std::unique_ptr<AdmittedModel> observed;
  ExactIdIndex index;
  mutable std::mutex reader_mutex;
  std::string access = "not_applicable", admission = "built_in_parameters";
  Json transient_index = nullptr;

  Impl(Json chosen, Value signals, std::int64_t end,
       std::optional<Value> retained = {}, Json retained_source = nullptr,
       std::optional<Value> inline_selection = {})
      : selection(std::move(chosen)) {
    deadline(end);
    keys(selection, {"protocol", "id", "version", "horizon_ns", "parameters"});
    need(signals.kind() == "array", "admitted census signals array required");
    if (selection.at("id") == "observed_trade_levels") {
      need(!retained && retained_source.is_null(),
           "built-in model has no supplied outcome source");
      observed = std::make_unique<AdmittedModel>(selection,
                                                 std::vector<std::string>{});
      return;
    }
    need(selection.at("id") == "external_outcomes",
         "stream model supports built-in observed or external outcomes");
    auto &parameters = selection.at("parameters");
    keys_optional(
        parameters,
        {"producer", "measure", "conditioning", "calibration_reference"},
        {"outcomes", "outcomes_source"});
    const bool inline_rows = parameters.contains("outcomes");
    need(inline_rows != parameters.contains("outcomes_source"),
         "select exactly one external outcome source");
    if (inline_rows) {
      Value original = inline_selection
                           ? std::move(*inline_selection)
                           : Value(std::move(parameters.at("outcomes")));
      parameters["outcomes"] = Json::array();
      if (retained) {
        need(retained_source.is_null() &&
                 original.sha256({}, [end] { deadline(end); }) ==
                     retained->sha256({}, [end] { deadline(end); }),
             "retained inline outcomes mismatch");
        supplied = std::move(retained);
        admission = "retained_inline_correspondence";
      } else {
        supplied = std::move(original);
        admission = "inline_rows";
      }
      access = "inline";
    } else {
      if (retained) {
        need(retained->kind() == "array",
             "retained outcome rows array required");
        retained_reference(parameters.at("outcomes_source"), retained_source,
                           *retained, end);
        supplied = std::move(retained);
        reference = std::move(retained_source);
        admission = "retained_source_correspondence";
      } else {
        auto selected =
            select_bundle_value(parameters.at("outcomes_source"), end);
        supplied = std::move(selected.value);
        reference = std::move(selected.reference);
        admission = "full_outer_logical_closure";
      }
      parameters.erase("outcomes_source");
      parameters["outcomes"] = Json::array();
      access = "bundle";
    }
    // Validate the complete selection metadata even for an empty census.
    (void)AdmittedModel(selection, {});
    need(supplied->kind() == "array" && supplied->size() == signals.size(),
         "one outcome per census signal required");
    auto rows = supplied->children();
    while (auto child = rows.next()) {
      deadline(end);
      auto value = row(child->value, end);
      const auto id = str(value.at("signal_id"));
      const auto inserted = index.insert(id);
      need(inserted.inserted, "duplicate supplied model signal identity");
      auto one = selection;
      one["parameters"]["outcomes"] = Json::array({std::move(value)});
      (void)AdmittedModel(std::move(one), {id});
    }
    // A transient second exact index rejects a malformed duplicate census;
    // only the outcome ID->row ordinal map survives construction.
    ExactIdIndex census_ids;
    auto census = signals.children();
    while (auto signal = census.next()) {
      deadline(end);
      need(signal->value.kind() == "object" &&
               signal->value.contains("signal_id"),
           "census signal identity required");
      const auto id = identity(signal->value.at("signal_id"), end);
      need(census_ids.insert(id).inserted && index.find(id).has_value(),
           "unknown or duplicate census model signal identity");
    }
    need(census_ids.size() == index.size(),
         "supplied model signal set mismatch");
    transient_index = census_ids.evidence();
    deadline(end);
  }
};

StreamModel::StreamModel(Json selection, logical::Value signals,
                         std::int64_t end)
    : p_(std::make_unique<Impl>(std::move(selection), std::move(signals),
                                end)) {}
StreamModel::StreamModel(Json selection, logical::Value signals,
                         logical::Value outcomes, Json source_reference,
                         std::int64_t end)
    : p_(std::make_unique<Impl>(std::move(selection), std::move(signals), end,
                                std::move(outcomes),
                                std::move(source_reference))) {}
StreamModel::StreamModel(logical::Value selection, logical::Value signals,
                         logical::Value outcomes, Json source_reference,
                         std::int64_t end) {
  auto chosen = retained_selection(selection, end);
  p_ = std::make_unique<Impl>(
      std::move(chosen.metadata), std::move(signals), end, std::move(outcomes),
      std::move(source_reference), std::move(chosen.inline_outcomes));
}
StreamModel::~StreamModel() = default;
Json StreamModel::evaluate(const ModelFrame &frame, std::int64_t end) const {
  deadline(end);
  if (p_->observed)
    return p_->observed->evaluate(frame, end);
  const auto ordinal = p_->index.find(frame.signal_id);
  need(ordinal.has_value(), "model signal not admitted");
  Json owned;
  {
    // NodeHandles share a single-caller mutable cache. No view escapes this
    // lock into worker math; owned JSON contains exactly one bounded row.
    std::lock_guard lock(p_->reader_mutex);
    deadline(end);
    owned = row(p_->supplied->at(*ordinal), end);
  }
  need(owned.at("signal_id") == frame.signal_id,
       "model row identity changed after admission");
  auto one = p_->selection;
  one["parameters"]["outcomes"] = Json::array({std::move(owned)});
  return AdmittedModel(std::move(one), {frame.signal_id}).evaluate(frame, end);
}
std::uint64_t StreamModel::horizon_ns() const {
  return u64(p_->selection.at("horizon_ns"));
}
Json StreamModel::evidence() const {
  return {{"model_id", p_->selection.at("id")},
          {"outcome_source", p_->access},
          {"source_admission", p_->admission},
          {"outcome_rows", dec(p_->index.size())},
          {"retained_outcome_id_index",
           p_->observed ? Json(nullptr) : p_->index.evidence()},
          {"temporary_census_id_index", p_->transient_index},
          {"temporary_census_id_index_released", true},
          {"row_access", p_->observed
                             ? "immutable_native_model"
                             : "mutex_owned_row_then_independent_math"},
          {"aggregate_state",
           p_->observed ? "no_aggregate_signal_id_state"
                        : "logical_outcome_backing_plus_one_retained_exact_"
                          "index_and_one_temporary_admission_index"},
          {"authorship", "not_verified"}};
}
const Json &StreamModel::source_reference() const { return p_->reference; }
const logical::Value &StreamModel::outcomes() const {
  need(p_->supplied.has_value(), "built-in model has no supplied outcomes");
  return *p_->supplied;
}
} // namespace symphony::sbv::detail
