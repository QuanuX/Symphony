#include "census.hpp"
#include "wide_rational.hpp"
#include <algorithm>
#include <limits>
#include <map>
#include <symphony/sbv/models.hpp>

namespace symphony::sbv::detail {
namespace {
namespace w = wide_rational;
using R = w::R;
void label(const Json &v) { need(!str(v).empty(), "nonempty label required"); }
R minus(R a, R b) { return w::plus(a, {-b.n, b.d}); }
struct Less {
  bool operator()(R a, R b) const { return w::compare(a, b) < 0; }
};
using Distribution = std::map<R, R, Less>;
struct Source {
  Json result, reference, census;
};
Json economic_census(const Json &);
Source load(const Json &ref, std::size_t &bytes_read, std::int64_t end) {
  keys(ref, {"path", "expected_sha256", "pointer"});
  auto bytes = read_file(str(ref.at("path")), end);
  need(bytes.size() <= std::numeric_limits<std::size_t>::max() - bytes_read,
       "source byte accounting overflow");
  bytes_read += bytes.size();
  auto outer = e::parse_bounded_json(bytes, artifact_bytes, artifact_values);
  validate_result(outer);
  need(outer.at("content_sha256") == ref.at("expected_sha256"),
       "composition source digest mismatch");
  auto result = outer.at(Json::json_pointer(str(ref.at("pointer"))));
  validate_result(result);
  return {result,
          {{"path", ref.at("path")},
           {"content_sha256", outer.at("content_sha256")},
           {"pointer", ref.at("pointer")},
           {"selected_sha256", result.at("content_sha256")},
           {"file_sha256", e::sha256_hex(bytes)},
           {"bytes", dec(bytes.size())}},
          economic_census(result)};
}
const Json &matching(const Json &rows, const std::string &id) {
  need(rows.is_array(), "source row array required");
  const Json *found = nullptr;
  for (const auto &row : rows)
    if (str(row.at("signal_id")) == id) {
      need(!found, "duplicate selected source signal");
      found = &row;
    }
  need(found, "selected source signal missing");
  return *found;
}
bool digest(const Json &v) {
  auto value = str(v);
  return value.size() == 64 &&
         std::all_of(value.begin(), value.end(), [](char c) {
           return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f');
         });
}
Json economic_census(const Json &result) {
  const auto &s = result.at("sections"),
             &context_section = s.at("source_context"),
             &context = context_section.at("data"),
             &choices = s.at("choices").at("data"),
             &provenance = s.at("provenance").at("data"),
             &summary = s.at("summary").at("data"),
             &parent_choices = context.at("choices").at("data"),
             &parent_provenance = context.at("provenance").at("data");
  need(context_section.at("status") == "available" && s.contains("replay") &&
           context.at("choices").at("status") == "available" &&
           context.at("provenance").at("status") == "available" &&
           choices.at("protocol") == "symphony.sbv.economics-input.v1" &&
           parent_choices.at("protocol") == "symphony.sbv.evaluate-input.v1",
       "economic census, replay and evaluation context required");
  Json c;
  if (s.contains("census")) {
    need(s.at("census").at("status") == "available" &&
             s.at("census").at("data") == context.at("census"),
         "economic census/context mismatch");
    c = s.at("census").at("data");
  } else {
    // Economics v1 originally retained the complete external declaration.
    // Normalize that identity without reopening or authenticating ancestors.
    const auto &original = context.at("census");
    need(original.at("protocol") == "symphony.sbv.external-census.v1",
         "legacy economics requires its retained external census");
    c = {{"protocol", "symphony.sbv.census-evidence.v1"},
         {"kind", "external"},
         {"identity_domain", "external_declaration"},
         {"census_sha256", e::sha256_hex(original.dump())},
         {"source_sha256", parent_provenance.at("source_sha256")},
         {"dataset", parent_provenance.at("dataset")},
         {"instrument_id", parent_provenance.at("instrument_id")},
         {"mode", original.at("mode")},
         {"producer", original.at("producer")},
         {"signals", original.at("signals")},
         {"declaration", original}};
  }
  validate_census_evidence(c);
  const bool native_provider_model =
      parent_choices.at("model").at("id") == "native_provider";
  need(!context.contains("provider") || native_provider_model,
       "unexpected economic model provider evidence");
  if (native_provider_model)
    validate_native_provider_model_context(parent_choices.at("model"),
                                           context.at("provider"),
                                           s.at("execution").at("data"));
  // Re-admit the retained original identity from saved evidence only. Older
  // economic contexts predate dataset_feed and keep their legacy file profile.
  const auto parent_identity = dataset_result_identity(
      parent_choices, parent_provenance,
      context.contains("dataset_feed")
          ? Json{{"dataset_feed", context.at("dataset_feed")}}
          : Json(nullptr));
  need(c.at("census_sha256") == summary.at("source_census_sha256") &&
           c.at("source_sha256") == parent_identity.at("source_sha256") &&
           c.at("source_sha256") == parent_provenance.at("source_sha256") &&
           c.at("dataset") == parent_identity.at("dataset") &&
           c.at("dataset") == parent_provenance.at("dataset") &&
           c.at("instrument_id") == parent_provenance.at("instrument_id") &&
           c.at("census_sha256") == parent_provenance.at("census_sha256") &&
           c.at("producer") == parent_provenance.at("census_producer"),
       "economic retained census lineage mismatch");
  const auto &parent_selection = parent_choices.at("census");
  if (parent_selection.contains("protocol"))
    need(c.at("kind") == "external" && parent_selection == c.at("declaration"),
         "economic original census selection mismatch");
  else {
    keys(parent_selection, {"path", "expected_sha256", "pointer"});
    need(digest(parent_selection.at("expected_sha256")),
         "economic original census reference digest invalid");
    (void)Json::json_pointer(str(parent_selection.at("pointer")));
  }
  need(digest(provenance.at("source_content_sha256")) &&
           digest(provenance.at("source_file_sha256")),
       "economic parent result identity invalid");
  if (choices.contains("source")) {
    const auto &ref = choices.at("source"), &retained = context.at("reference");
    keys(ref, {"path", "expected_sha256", "pointer"});
    for (const auto *name : {"path", "expected_sha256", "pointer"})
      need(ref.at(name) == retained.at(name),
           "economic parent reference mismatch");
    need(ref.at("path") == provenance.at("source_path") &&
             ref.at("expected_sha256") ==
                 provenance.at("source_outer_content_sha256") &&
             ref.at("pointer") == provenance.at("source_pointer"),
         "economic parent reference/provenance mismatch");
  } else
    need(choices.at("path") == provenance.at("source_path") &&
             choices.at("expected_sha256") ==
                 provenance.at("source_content_sha256"),
         "economic legacy parent reference/provenance mismatch");
  if (context.contains("reference")) {
    const auto &ref = context.at("reference");
    keys(ref, {"path", "expected_sha256", "pointer", "file_sha256",
               "selected_content_sha256", "authorship"});
    need(ref.at("path") == provenance.at("source_path") &&
             ref.at("expected_sha256") ==
                 provenance.at("source_outer_content_sha256") &&
             ref.at("pointer") == provenance.at("source_pointer") &&
             ref.at("file_sha256") == provenance.at("source_file_sha256") &&
             ref.at("selected_content_sha256") ==
                 provenance.at("source_content_sha256") &&
             ref.at("authorship") == "not_verified" &&
             digest(ref.at("expected_sha256")),
         "economic retained parent reference incoherent");
    (void)Json::json_pointer(str(ref.at("pointer")));
    need(!str(ref.at("pointer")).empty() ||
             ref.at("expected_sha256") == ref.at("selected_content_sha256"),
         "economic root reference/selected identity mismatch");
  }
  need(s.at("signals").at("data").is_array() &&
           s.at("execution").at("data").is_array() &&
           s.at("economics").at("data").is_array() &&
           u64(summary.at("source_signal_count")) == c.at("signals").size() &&
           u64(summary.at("selected_signal_count")) ==
               s.at("economics").at("data").size() &&
           s.at("signals").at("data").size() ==
               s.at("economics").at("data").size() &&
           s.at("execution").at("data").size() ==
               s.at("economics").at("data").size(),
       "economic census and selection counts mismatch");
  return c;
}
struct Atom {
  R value, weight, pnl;
};
struct Component {
  Json retained;
  std::vector<Atom> atoms;
  std::string failure, measure;
  R conversion{1, 1};
};
// Re-admit the selected typed economic row and its local correspondences.
// This checks arithmetic/content coherence, not publisher authenticity or
// the empirical truth of the model, prices, conditioning or user economics.
Component component(const Json &choice, const Source &source,
                    const Json &economic, std::int64_t end) {
  keys(choice, {"id", "source", "outcome_pointer", "signal_id", "conditioning",
                "conversion", "extensions"});
  label(choice.at("id"));
  label(choice.at("signal_id"));
  label(choice.at("conditioning"));
  need(choice.at("extensions").is_object(), "component extensions required");
  const auto id = str(choice.at("signal_id"));
  const auto &s = source.result.at("sections");
  const auto &input = s.at("choices").at("data");
  need(input.at("protocol") == "symphony.sbv.economics-input.v1",
       "typed economics source required");
  const auto &row =
      source.result.at(Json::json_pointer(str(choice.at("outcome_pointer"))));
  keys(row, {"protocol", "signal_id", "status", "reason", "measure",
             "conditioning", "mode", "transform", "transformed_support",
             "nonexecution_pnl", "atoms", "calibration"});
  need(row.at("protocol") == "symphony.sbv.economic-outcome.v1" &&
           row.at("signal_id") == choice.at("signal_id") &&
           row.at("conditioning") == choice.at("conditioning") &&
           row == matching(s.at("economics").at("data"), id),
       "selected economic identity or conditioning mismatch");
  const auto &selected = matching(input.at("selections"), id);
  need(selected.at("transform") == row.at("transform") &&
           selected.at("mode") == row.at("mode") &&
           selected.at("nonexecution_pnl") == row.at("nonexecution_pnl"),
       "economic row/selection mismatch");
  const auto &signal = matching(s.at("signals").at("data"), id);
  const auto &model = matching(s.at("execution").at("data"), id);
  const auto &parent_model =
      s.at("source_context").at("data").at("choices").at("data").at("model");
  need(signal == matching(source.census.at("signals"), id),
       "selected economic signal differs from retained census");
  need(model.at("protocol") == "symphony.sbv.model-outcome.v1" &&
           model.at("measure") == row.at("measure") &&
           model.at("model_id") == parent_model.at("id") &&
           model.at("model_version") == parent_model.at("version") &&
           (model.at("conditioning") == "execution" ||
            model.at("conditioning") == "scenario"),
       "economic row/model mismatch");
  const auto &t = row.at("transform");
  keys(t, {"id", "version", "reference_price_nanos", "price_unit_nanos",
           "position_units", "value_per_price_unit", "cost_per_outcome",
           "return_basis", "pnl_unit"});
  need(t.at("id") == "linear_price_pnl" && t.at("version") == "1",
       "economic transform not supported by composer");
  label(t.at("pnl_unit"));
  auto basis = w::artifact(t.at("return_basis"));
  need(basis.n != 0, "economic return basis must be nonzero");
  Component c;
  c.measure = str(row.at("measure"));
  need(c.measure == "probability" || c.measure == "scenario_weight" ||
           c.measure == "signed_coefficient",
       "unknown source measure");
  const bool mixture = row.at("mode") == "execution_mixture";
  need(mixture || row.at("mode") == "support_only",
       "unknown source economic interpretation");
  need(row.at("conditioning") == (mixture ? Json("execution_and_nonexecution")
                                          : model.at("conditioning")),
       "economic source conditioning mismatch");
  if (economic.at("kind") == "multiplicative_return") {
    need(choice.at("conversion").is_null(),
         "multiplicative source returns do not consume P&L conversion");
  } else {
    const auto &conversion = choice.at("conversion");
    keys(conversion, {"factor", "output_unit", "description"});
    c.conversion = w::artifact(conversion.at("factor"));
    label(conversion.at("description"));
    need(conversion.at("output_unit") == economic.at("state_unit"),
         "P&L conversion output unit mismatch");
  }
  const auto status = str(row.at("status"));
  need(status == "available" || status == "unavailable",
       "unknown economic availability");
  need(row.at("atoms").is_array(), "economic atoms required");
  if (status == "unavailable") {
    label(row.at("reason"));
    need(row.at("atoms").empty(), "unavailable economics contains atoms");
    c.failure = "source economic outcome unavailable: " + str(row.at("reason"));
  } else {
    need(str(row.at("reason")).empty() && model.at("status") == "available" &&
             model.at("price_unit") == "price_nanos" &&
             model.at("price_scale") == "1e-9",
         "available economic/model contract mismatch");
    const auto &support = model.at("support");
    const auto &transformed = row.at("transformed_support");
    need(support.is_array() && !support.empty() && transformed.is_array() &&
             transformed.size() == support.size() &&
             row.at("atoms").size() == support.size() + (mixture ? 1U : 0U),
         "economic atom/support correspondence mismatch");
    auto reference = i64(t.at("reference_price_nanos"));
    auto unit = i64(t.at("price_unit_nanos"));
    need(reference != INT64_MAX && unit > 0, "economic price/unit invalid");
    auto position = w::artifact(t.at("position_units"));
    auto value = w::artifact(t.at("value_per_price_unit"));
    auto cost = w::artifact(t.at("cost_per_outcome"));
    R fill{1, 1}, no_fill;
    if (mixture) {
      need(c.measure == "probability" &&
               model.at("conditioning") == "execution" &&
               model.at("fill_probability").at("status") ==
                   "external_assumption" &&
               model.at("no_fill_probability").at("status") ==
                   "external_assumption",
           "execution mixture likelihood contract mismatch");
      fill = w::artifact(model.at("fill_probability").at("value"));
      no_fill = w::artifact(model.at("no_fill_probability").at("value"));
      need(fill.n >= 0 && no_fill.n >= 0 &&
               w::equal(w::plus(fill, no_fill), {1, 1}),
           "execution likelihood complement mismatch");
    } else
      need(row.at("nonexecution_pnl").is_null(),
           "support-only has unexpected nonexecution P&L");
    R total;
    std::set<std::int64_t> prices;
    for (std::size_t i = 0; i < support.size(); ++i) {
      deadline(end);
      const auto &a = support[i], &v = transformed[i],
                 &atom = row.at("atoms")[i];
      auto price = i64(a.at("price_nanos"));
      need(price != INT64_MAX && prices.insert(price).second,
           "economic source has undefined/duplicate support price");
      auto weight = w::artifact(a.at("weight"));
      need(c.measure == "signed_coefficient" || weight.n >= 0,
           "source measure has negative weight");
      if (c.measure == "probability")
        total = w::plus(total, weight);
      auto gross = w::times(
          w::times(
              w::reduce({w::add(price, -static_cast<w::I>(reference)), unit}),
              position),
          value);
      auto net = minus(gross, cost), ret = w::divide(net, basis);
      need(v.at("price_nanos") == a.at("price_nanos") &&
               w::equal(w::artifact(v.at("weight")), weight) &&
               w::equal(w::artifact(v.at("gross_pnl")), gross) &&
               w::equal(w::artifact(v.at("cost")), cost) &&
               w::equal(w::artifact(v.at("net_pnl")), net) &&
               w::equal(w::artifact(v.at("return")), ret),
           "transformed source economics incoherent");
      keys(atom, {"kind", "support_index", "return", "weight"});
      auto resolved_weight = w::times(weight, fill);
      need(atom.at("kind") == "price_outcome" &&
               u64(atom.at("support_index")) == i &&
               w::equal(w::artifact(atom.at("return")), ret) &&
               w::equal(w::artifact(atom.at("weight")), resolved_weight),
           "economic source atom incoherent");
      c.atoms.push_back({ret, resolved_weight, net});
    }
    need(c.measure != "probability" || w::equal(total, {1, 1}),
         "source probabilities must sum exactly to one");
    if (mixture) {
      const auto &atom = row.at("atoms").back();
      keys(atom, {"kind", "support_index", "return", "weight"});
      auto pnl = w::artifact(row.at("nonexecution_pnl"));
      auto ret = w::divide(pnl, basis);
      need(atom.at("kind") == "nonexecution" &&
               atom.at("support_index").is_null() &&
               w::equal(w::artifact(atom.at("return")), ret) &&
               w::equal(w::artifact(atom.at("weight")), no_fill),
           "economic nonexecution atom incoherent");
      c.atoms.push_back({ret, no_fill, pnl});
    }
  }
  auto reference = [&](const char *pointer) {
    auto ref = choice.at("source");
    ref["pointer"] = str(ref.at("pointer")) + pointer;
    return ref;
  };
  c.retained = {
      {"id", choice.at("id")},
      {"source", source.reference},
      {"outcome_pointer", choice.at("outcome_pointer")},
      {"outcome_sha256", e::sha256_hex(row.dump())},
      {"signal", signal},
      {"model", model},
      {"economic_outcome", row},
      {"conversion", choice.at("conversion")},
      {"source_summary", s.at("summary")},
      {"source_provenance", s.at("provenance")},
      {"census", source.census},
      {"census_context_reference", reference("/sections/source_context")},
      {"replay_reference", reference("/sections/replay")},
      {"extensions", choice.at("extensions")}};
  if (parent_model.at("id") == "native_provider") {
    c.retained["model_selection"] = parent_model;
    c.retained["provider"] = s.at("source_context").at("data").at("provider");
  }
  return c;
}
Json wire(const Distribution &distribution) {
  Json out = Json::array();
  for (const auto &[value, mass] : distribution)
    out.push_back({{"state", w::wire(value)}, {"probability", w::wire(mass)}});
  return out;
}
Json validate_studies(const Json &studies) {
  need(studies.is_array(), "composition studies array required");
  std::set<std::string> ids;
  for (const auto &study : studies) {
    keys(study, {"id", "version", "parameters"});
    auto id = str(study.at("id"));
    need(ids.insert(id).second && study.at("version") == "1",
         "unique installed composition study required");
    const auto &parameters = study.at("parameters");
    if (id == "terminal_moments")
      keys(parameters, {});
    else {
      need(id == "terminal_quantiles", "unknown composition study");
      keys(parameters, {"levels"});
      need(parameters.at("levels").is_array(), "quantile levels required");
      for (const auto &q : parameters.at("levels")) {
        auto level = w::artifact(q);
        need(level.n >= 0 && level.n <= level.d,
             "quantile level outside [0,1]");
      }
    }
  }
  return studies;
}
Json studies(const Json &selections, const Distribution &terminal,
             const std::string &unit, const std::string &failure,
             std::int64_t end) {
  Json out = Json::array();
  for (const auto &choice : selections) {
    deadline(end);
    Json row{{"id", choice.at("id")},
             {"version", "1"},
             {"parameters", choice.at("parameters")},
             {"status", failure.empty() ? "available" : "unavailable"},
             {"reason", failure},
             {"state_unit", unit},
             {"input_pointer", "/sections/distributions/data/terminal_atoms"},
             {"uncertainty",
              missing("no parameter-estimation uncertainty model selected")},
             {"data", nullptr}};
    if (failure.empty()) {
      if (choice.at("id") == "terminal_moments") {
        R mean, variance;
        for (const auto &[value, mass] : terminal) {
          deadline(end);
          mean = w::plus(mean, w::times(value, mass));
        }
        for (const auto &[value, mass] : terminal) {
          deadline(end);
          if (mass.n == 0)
            continue;
          auto delta = minus(value, mean);
          variance = w::plus(variance, w::times(mass, w::times(delta, delta)));
        }
        row["data"] = {{"mean", w::wire(mean)},
                       {"population_variance", w::wire(variance)},
                       {"variance_unit", unit + " squared"},
                       {"probability_mass", w::wire({1, 1})}};
      } else {
        Json quantiles = Json::array();
        for (const auto &q : choice.at("parameters").at("levels")) {
          auto level = w::artifact(q);
          R cumulative;
          bool found = false;
          for (const auto &[value, mass] : terminal) {
            deadline(end);
            if (mass.n == 0)
              continue;
            cumulative = w::plus(cumulative, mass);
            if (w::compare(cumulative, level) >= 0) {
              quantiles.push_back(
                  {{"level", w::wire(level)}, {"state", w::wire(value)}});
              found = true;
              break;
            }
          }
          need(found, "composition quantile mass invariant");
        }
        row["data"] = {{"quantiles", quantiles},
                       {"method", "inverse_cdf_positive_probability"},
                       {"interval_kind",
                        "modeled outcome quantiles; not parameter confidence"},
                       {"boundary_rule",
                        "q=0 selects minimum positive-mass state; "
                        "otherwise first cumulative probability >= q"}};
      }
    }
    out.push_back(std::move(row));
  }
  return out;
}
void limit(const Json &v, std::size_t count, const char *message) {
  if (!v.is_null())
    need(count <= u64(v), message);
}
} // namespace

Json compose_economics(const Json &p, std::int64_t end) {
  keys(p, {"protocol", "output_path", "components", "dependence", "economics",
           "conditioning_description", "on_unavailable", "retain_paths",
           "retain_prefix_distributions", "studies", "limits", "extensions"});
  deadline(end);
  need(p.at("extensions").is_object() && p.at("components").is_array() &&
           p.at("retain_paths").is_boolean() &&
           p.at("retain_prefix_distributions").is_boolean(),
       "composition input shape mismatch");
  label(p.at("conditioning_description"));
  need(p.at("on_unavailable") == "reject" ||
           p.at("on_unavailable") == "unavailable",
       "composition unavailable policy required");
  const auto selected_studies = validate_studies(p.at("studies"));
  const auto &economic = p.at("economics"), &dependence = p.at("dependence"),
             &limits = p.at("limits");
  keys(economic,
       {"kind", "initial_state", "state_unit", "state_domain", "description"});
  label(economic.at("state_unit"));
  label(economic.at("description"));
  const bool multiplicative = economic.at("kind") == "multiplicative_return";
  need(multiplicative || economic.at("kind") == "additive_pnl",
       "unknown composition economics");
  const bool nonnegative = economic.at("state_domain") == "nonnegative";
  need(nonnegative || economic.at("state_domain") == "signed",
       "unknown state domain");
  const auto initial = w::artifact(economic.at("initial_state"));
  need(!nonnegative || initial.n >= 0,
       "initial state violates selected nonnegative domain");
  keys(limits, {"max_paths", "max_terminal_atoms"});
  for (const auto *name : {"max_paths", "max_terminal_atoms"})
    if (!limits.at(name).is_null())
      (void)u64(limits.at(name));
  keys(dependence, {"kind", "marginal_policy", "paths", "description"});
  label(dependence.at("description"));
  need(dependence.at("paths").is_array(), "joint paths array required");
  const bool independent = dependence.at("kind") == "independent";
  need(independent || dependence.at("kind") == "joint_indices",
       "explicit dependence required");
  if (independent)
    need(dependence.at("paths").empty() &&
             dependence.at("marginal_policy").is_null(),
         "independent composition has unexpected joint choices");
  else
    need(!dependence.at("paths").empty() &&
             (dependence.at("marginal_policy") == "require_source_match" ||
              dependence.at("marginal_policy") == "support_only"),
         "joint paths and marginal policy required");
  std::set<std::string> path_ids;
  R joint_mass;
  for (const auto &path : dependence.at("paths")) {
    deadline(end);
    keys(path, {"id", "atom_indices", "probability"});
    label(path.at("id"));
    need(path_ids.insert(str(path.at("id"))).second &&
             path.at("atom_indices").is_array() &&
             path.at("atom_indices").size() == p.at("components").size(),
         "joint path identity or index count mismatch");
    for (const auto &index : path.at("atom_indices"))
      (void)u64(index);
    auto mass = w::artifact(path.at("probability"));
    need(mass.n >= 0, "joint probability cannot be negative");
    joint_mass = w::plus(joint_mass, mass);
  }
  need(independent || w::equal(joint_mass, {1, 1}),
       "joint probabilities must sum exactly to one; no normalization");

  std::map<std::string, Source> cache;
  std::vector<Component> components;
  std::size_t bytes_read = 0;
  std::set<std::string> component_ids;
  Json retained = Json::array(), failures = Json::array();
  for (const auto &choice : p.at("components")) {
    deadline(end);
    need(component_ids.insert(str(choice.at("id"))).second,
         "duplicate composition component id");
    auto key = choice.at("source").dump();
    auto it = cache.find(key);
    if (it == cache.end())
      it = cache.emplace(key, load(choice.at("source"), bytes_read, end)).first;
    auto c = component(choice, it->second, economic, end);
    if (c.failure.empty() && c.measure != "probability" &&
        (independent ||
         dependence.at("marginal_policy") == "require_source_match"))
      c.failure = "selected dependence requires a source probability measure; "
                  "source measure is " +
                  c.measure;
    if (!c.failure.empty())
      failures.push_back(
          {{"component_id", choice.at("id")}, {"reason", c.failure}});
    retained.push_back(c.retained);
    components.push_back(std::move(c));
  }
  need(p.at("on_unavailable") != "reject" || failures.empty(),
       "selected economic composition unavailable");
  const std::string failure =
      failures.empty() ? ""
                       : "selected components or measure interpretation "
                         "unavailable; inspect component findings";
  Json paths = Json::array(), prefixes = Json::array(),
       marginals = Json::array();
  Distribution terminal;
  std::size_t count = 0;
  if (failure.empty()) {
    count = independent ? 1 : dependence.at("paths").size();
    if (independent)
      for (const auto &c : components) {
        need(!c.atoms.empty() &&
                 count <=
                     std::numeric_limits<std::size_t>::max() / c.atoms.size(),
             "exact path count exceeds host index representation");
        count *= c.atoms.size();
      }
    limit(limits.at("max_paths"), count, "caller path limit exceeded");
    std::vector<std::vector<R>> induced;
    for (const auto &c : components)
      induced.emplace_back(c.atoms.size());
    std::vector<Distribution> prefix_values;
    if (p.at("retain_prefix_distributions").get<bool>())
      prefix_values.resize(components.size() + 1);
    for (std::size_t i = 0; i < count; ++i) {
      deadline(end);
      std::vector<std::size_t> indices(components.size());
      Json index_wire = Json::array();
      R probability{1, 1};
      if (independent) {
        auto code = i;
        for (std::size_t j = components.size(); j-- > 0;) {
          indices[j] = code % components[j].atoms.size();
          code /= components[j].atoms.size();
        }
        for (std::size_t j = 0; j < components.size(); ++j)
          probability =
              w::times(probability, components[j].atoms[indices[j]].weight);
      } else {
        const auto &path = dependence.at("paths")[i];
        probability = w::artifact(path.at("probability"));
        for (std::size_t j = 0; j < components.size(); ++j) {
          auto index = u64(path.at("atom_indices")[j]);
          need(index < components[j].atoms.size(),
               "joint atom index outside source support");
          indices[j] = static_cast<std::size_t>(index);
        }
      }
      R state = initial;
      Json trajectory = Json::array();
      if (p.at("retain_paths").get<bool>())
        trajectory.push_back(w::wire(state));
      if (!prefix_values.empty())
        prefix_values[0][state] = w::plus(prefix_values[0][state], probability);
      for (std::size_t j = 0; j < components.size(); ++j) {
        deadline(end);
        const auto &c = components[j];
        const auto &a = c.atoms[indices[j]];
        state = multiplicative ? w::times(state, w::plus({1, 1}, a.value))
                               : w::plus(state, w::times(a.pnl, c.conversion));
        need(!nonnegative || state.n >= 0,
             "path state violates selected nonnegative domain");
        induced[j][indices[j]] = w::plus(induced[j][indices[j]], probability);
        if (p.at("retain_paths").get<bool>()) {
          index_wire.push_back(dec(indices[j]));
          trajectory.push_back(w::wire(state));
        }
        if (!prefix_values.empty())
          prefix_values[j + 1][state] =
              w::plus(prefix_values[j + 1][state], probability);
      }
      auto [it, fresh] = terminal.emplace(state, R{});
      if (fresh)
        limit(limits.at("max_terminal_atoms"), terminal.size(),
              "caller terminal-atom limit exceeded");
      it->second = w::plus(it->second, probability);
      if (p.at("retain_paths").get<bool>())
        paths.push_back(
            {{"id",
              independent ? Json(dec(i)) : dependence.at("paths")[i].at("id")},
             {"atom_indices", index_wire},
             {"probability", w::wire(probability)},
             {"state", trajectory}});
    }
    for (std::size_t j = 0; j < components.size(); ++j) {
      Json weights = Json::array();
      bool equal = true;
      for (std::size_t k = 0; k < induced[j].size(); ++k) {
        deadline(end);
        equal &= w::equal(induced[j][k], components[j].atoms[k].weight);
        weights.push_back(
            {{"atom_index", dec(k)},
             {"source_weight", w::wire(components[j].atoms[k].weight)},
             {"induced_probability", w::wire(induced[j][k])}});
      }
      need((!independent &&
            dependence.at("marginal_policy") != "require_source_match") ||
               equal,
           "composed marginal differs from required source measure");
      marginals.push_back({{"component_id", p.at("components")[j].at("id")},
                           {"source_measure", components[j].measure},
                           {"weights_match", equal},
                           {"atoms", weights}});
    }
    R total;
    for (const auto &[value, mass] : terminal) {
      (void)value;
      total = w::plus(total, mass);
    }
    need(w::equal(total, {1, 1}), "terminal probability invariant");
    for (std::size_t j = 0; j < prefix_values.size(); ++j)
      prefixes.push_back({{"completed_components", dec(j)},
                          {"atoms", wire(prefix_values[j])}});
  }
  auto result = base(
      "native exact composition of immutable per-signal economic outcomes");
  result["status"] = failure.empty() ? "completed" : "partial";
  auto &s = result["sections"];
  s["summary"] = section(
      {{"component_count", dec(components.size())},
       {"unavailable_components", dec(failures.size())},
       {"path_count", failure.empty() ? Json(dec(count)) : Json(nullptr)},
       {"retained_path_count", dec(paths.size())},
       {"terminal_atom_count",
        failure.empty() ? Json(dec(terminal.size())) : Json(nullptr)},
       {"probability_mass", failure.empty() ? w::wire({1, 1}) : Json(nullptr)},
       {"composition_status", failure.empty() ? "available" : "unavailable"}});
  s["composition_sources"] = section(retained);
  s["composition_findings"] = section(failures);
  s["distributions"] = section(
      {{"protocol", "symphony.sbv.economic-composition.v1"},
       {"dependence", dependence},
       {"economics", economic},
       {"conditioning_description", p.at("conditioning_description")},
       {"measure", "probability"},
       {"normalization", "none"},
       {"paths", paths},
       {"paths_retained", p.at("retain_paths")},
       {"terminal_atoms", failure.empty() ? wire(terminal) : Json(nullptr)},
       {"prefix_distributions", prefixes},
       {"prefix_distributions_retained", p.at("retain_prefix_distributions")},
       {"marginals", marginals},
       {"calibration", "not_verified"}},
      failure.empty() ? "available" : "unavailable", failure);
  s["studies"] =
      section(studies(selected_studies, terminal,
                      str(economic.at("state_unit")), failure, end),
              selected_studies.empty() ? "not_selected"
              : failure.empty()        ? "available"
                                       : "unavailable",
              selected_studies.empty() ? "zero studies selected" : failure);
  auto choices = p;
  choices.erase("output_path");
  s["choices"] = section(choices);
  s["resources"] =
      section({{"backend", "cpu"},
               {"actual_workers", failure.empty() ? "1" : "0"},
               {"source_bytes_read", dec(bytes_read)},
               {"distinct_source_references", dec(cache.size())},
               {"method", "exact enumeration; no sampling or pruning"},
               {"limits", limits}});
  s["provenance"] = section(
      {{"engine_version", version},
       {"numeric_profile", "checked signed int128 reduced rationals; overflow "
                           "rejects before persistence"},
       {"source_authorship", "not_verified"},
       {"ordering", "declared component order; independent enumeration varies "
                    "last component fastest; joint input order retained"}});
  s["diagnostics"] = section(Json::array(
      {"Source conditions and user-selected dependence remain assumptions; "
       "conditional price support is not an unconditional execution forecast.",
       "Multiplicative returns use each retained source basis. Arithmetic "
       "compounding does not establish reinvestment, capital, shared-liquidity "
       "or feasible account paths.",
       "Explicit joint probabilities are used once. Source marginal weights "
       "are either checked exactly or retained as separate support-only "
       "provenance.",
       "Zero-probability paths and terminal atoms remain present. No hidden "
       "normalization, rounding, pruning or resource-triggered method change.",
       "Replay and census references point to immutable selected source "
       "results; observed events remain distinct from economic overlays.",
       "No additional fixed path/step policy limit is imposed. Existing result "
       "artifact, host index/address-space and checked numeric representation "
       "bounds still apply."}));
  s["user_extensions"] = section(p.at("extensions"));
  return persist(std::move(result), p, "compose-economics", end);
}
} // namespace symphony::sbv::detail
