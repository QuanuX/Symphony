#include "census.hpp"
#include "rational.hpp"
#include <algorithm>
#include <map>
#include <symphony/knowledge/engine/limits.hpp>
#include <symphony/sbv/models.hpp>

namespace symphony::sbv::detail {
namespace {
namespace r = rational;
using r::R;
R negative(R a) { return {-a.n, a.d}; }
R inverse(R a) {
  need(a.n != 0, "return basis must be nonzero");
  return a.n < 0 ? R{-a.d, -a.n} : R{a.d, a.n};
}
R artifact_ratio(const Json &j) {
  keys(j, {"numerator", "denominator"});
  return r::reduce({i64(j.at("numerator")), i64(j.at("denominator"))});
}
bool less(R a, R b) {
  // Each product of two signed int64 magnitudes fits this exact comparison.
  return static_cast<__int128_t>(a.n) * b.d <
         static_cast<__int128_t>(b.n) * a.d;
}
bool same(R a, R b) { return a.n == b.n && a.d == b.d; }
void label(const Json &v) {
  const auto s = str(v);
  need(!s.empty() && s.size() <= 256, "nonempty bounded identity required");
}
struct Transform {
  std::int64_t reference, unit;
  R position, value, cost, basis;
  explicit Transform(const Json &j) {
    keys(j, {"id", "version", "reference_price_nanos", "price_unit_nanos",
             "position_units", "value_per_price_unit", "cost_per_outcome",
             "return_basis", "pnl_unit"});
    need(j.at("id") == "linear_price_pnl" && j.at("version") == "1",
         "economic transform not installed");
    reference = i64(j.at("reference_price_nanos"));
    unit = i64(j.at("price_unit_nanos"));
    need(reference != INT64_MAX && unit > 0,
         "defined reference and positive price unit required");
    position = r::read_ratio(j.at("position_units"));
    value = r::read_ratio(j.at("value_per_price_unit"));
    cost = r::read_ratio(j.at("cost_per_outcome"));
    basis = r::read_ratio(j.at("return_basis"));
    (void)inverse(basis);
    label(j.at("pnl_unit"));
  }
  R gross(std::int64_t price) const {
    std::int64_t difference;
    need(!__builtin_sub_overflow(price, reference, &difference),
         "price difference exceeds selected int64 numeric profile");
    return r::times(r::times(r::reduce({difference, unit}), position), value);
  }
};
struct Atom {
  R value, weight;
};
Json study_selections(const Json &studies) {
  need(studies.is_array() && studies.size() <= 3, "study selection bound 0..3");
  std::set<std::string> ids;
  for (const auto &study : studies) {
    keys(study, {"id", "version", "parameters"});
    const auto id = str(study.at("id"));
    need(ids.insert(id).second && study.at("version") == "1",
         "unique installed study version required");
    const auto &p = study.at("parameters");
    if (id == "return_quantiles") {
      keys(p, {"levels"});
      const auto &levels = p.at("levels");
      need(levels.is_array() && !levels.empty() && levels.size() <= 32,
           "quantile levels bound 1..32");
      for (const auto &q : levels) {
        const auto level = r::read_ratio(q);
        need(level.n >= 0 && level.n <= level.d,
             "quantile level outside [0,1]");
      }
    } else {
      need(id == "weighted_return_sum" || id == "return_moments",
           "economic study not installed");
      keys(p, {});
    }
  }
  return studies;
}
Json metric(R value, const char *unit) {
  return {{"value", r::wire(value)}, {"unit", unit}};
}
Json study_result(const Json &choice, const std::string &signal,
                  const std::string &measure, const std::string &conditioning,
                  const std::vector<Atom> &atoms, const std::string &failure,
                  bool reject) {
  const auto id = str(choice.at("id"));
  auto reason = failure;
  if (reason.empty() && id != "weighted_return_sum" && measure != "probability")
    reason = "selected probability study cannot consume " + measure;
  need(!reject || reason.empty(), "selected study domain unavailable");
  Json result{{"protocol", "symphony.sbv.economic-study.v1"},
              {"id", id},
              {"version", "1"},
              {"signal_id", signal},
              {"status", reason.empty() ? "available" : "unavailable"},
              {"reason", reason},
              {"measure", measure},
              {"conditioning", conditioning},
              {"parameters", choice.at("parameters")},
              {"input_pointer", "/sections/economics/data"},
              {"atom_count", dec(atoms.size())},
              {"method", id == "return_quantiles" ? "inverse_cdf_positive_mass"
                         : id == "return_moments"
                             ? "exact_population_moments"
                             : "exact_unnormalized_weighted_sum"},
              {"uncertainty", missing("no estimation-error model selected")},
              {"data", nullptr}};
  if (!reason.empty())
    return result;
  // No reducer runs unless selected. A signed measure is never summed merely
  // to admit it, and no implicit denominator or renormalization is
  // introduced.
  if (id == "weighted_return_sum" || id == "return_moments") {
    R sum;
    for (const auto &a : atoms)
      sum = r::plus(sum, r::times(a.value, a.weight));
    if (id == "weighted_return_sum") {
      result["data"] = {{"weighted_sum", metric(sum, "return")},
                        {"normalization", "none"}};
    } else {
      R variance;
      for (const auto &a : atoms) {
        if (a.weight.n == 0)
          continue;
        const auto delta = r::plus(a.value, negative(sum));
        variance =
            r::plus(variance, r::times(a.weight, r::times(delta, delta)));
      }
      result["data"] = {{"mean", metric(sum, "return")},
                        {"variance", metric(variance, "return_squared")},
                        {"probability_mass", r::wire({1, 1})}};
    }
  } else {
    auto sorted = atoms;
    std::erase_if(sorted, [](const auto &a) { return a.weight.n == 0; });
    std::stable_sort(
        sorted.begin(), sorted.end(),
        [](const auto &a, const auto &b) { return less(a.value, b.value); });
    need(!sorted.empty(), "positive probability support required");
    Json values = Json::array();
    for (const auto &q : choice.at("parameters").at("levels")) {
      const auto level = r::read_ratio(q);
      R cumulative;
      R value = sorted.back().value;
      for (const auto &a : sorted) {
        cumulative = r::plus(cumulative, a.weight);
        if (!less(cumulative, level)) {
          value = a.value;
          break;
        }
      }
      values.push_back({{"level", r::wire(level)}, {"return", r::wire(value)}});
    }
    result["data"] = {{"quantiles", values},
                      {"boundary_rule",
                       "q=0 selects minimum positive-mass support; "
                       "otherwise first cumulative mass >= q"}};
  }
  return result;
}
} // namespace

Json economics(const Json &p, std::int64_t end) {
  keys_optional(p,
                {"protocol", "output_path", "selections", "studies",
                 "on_incompatible", "extensions"},
                {"path", "expected_sha256", "source"});
  const bool referenced = p.contains("source");
  need(referenced ? !p.contains("path") && !p.contains("expected_sha256")
                  : p.contains("path") && p.contains("expected_sha256"),
       "select exactly one economic source reference form");
  const Json selection =
      referenced ? p.at("source")
                 : Json{{"path", p.at("path")},
                        {"expected_sha256", p.at("expected_sha256")},
                        {"pointer", ""}};
  keys(selection, {"path", "expected_sha256", "pointer"});
  need(p.at("extensions").is_object(), "extensions object required");
  const auto policy = str(p.at("on_incompatible"));
  need(policy == "unavailable" || policy == "reject",
       "explicit domain policy required");
  const bool reject = policy == "reject";
  const auto studies = study_selections(p.at("studies"));
  const auto &selections = p.at("selections");
  need(selections.is_array() && selections.size() <= 4096,
       "selection bound 0..4096");
  const auto bytes = read_file(str(selection.at("path")), end);
  const auto outer =
      e::parse_bounded_json(bytes, artifact_bytes, artifact_values);
  validate_result(outer);
  need(outer.at("content_sha256") == selection.at("expected_sha256"),
       "source snapshot mismatch");
  const Json::json_pointer pointer(str(selection.at("pointer")));
  need(outer.contains(pointer), "economic source result pointer missing");
  const auto &source = outer.at(pointer);
  validate_result(source);
  const auto &sections = source.at("sections");
  need(sections.at("choices").at("data").at("protocol") ==
           "symphony.sbv.evaluate-input.v1",
       "economic source must be a complete evaluation result");
  const Json reference{{"path", selection.at("path")},
                       {"expected_sha256", selection.at("expected_sha256")},
                       {"pointer", selection.at("pointer")},
                       {"file_sha256", e::sha256_hex(bytes)},
                       {"selected_content_sha256", source.at("content_sha256")},
                       {"authorship", "not_verified"}};
  const auto census = census_evidence(source);
  const auto &signals = census.at("signals");
  const auto &models = sections.at("execution").at("data");
  need(signals.is_array() && signals.size() <= 4096 && models.is_array() &&
           models.size() == signals.size(),
       "source signal/model correspondence required");
  const auto &parent_choices = sections.at("choices").at("data");
  const bool native_provider_model =
      parent_choices.contains("model") &&
      parent_choices.at("model").at("id") == "native_provider";
  need(!sections.contains("provider") || native_provider_model,
       "unexpected model provider evidence");
  if (native_provider_model) {
    need(sections.at("provider").at("status") == "available",
         "model provider evidence must be available");
    validate_native_provider_model_context(
        parent_choices.at("model"), sections.at("provider").at("data"), models);
  }
  std::map<std::string, const Json *> signal_map, model_map;
  for (const auto &s : signals) {
    label(s.at("signal_id"));
    need(signal_map.emplace(str(s.at("signal_id")), &s).second,
         "duplicate source signal");
  }
  for (const auto &m : models) {
    need(m.at("protocol") == "symphony.sbv.model-outcome.v1",
         "unsupported model outcome protocol");
    const auto id = str(m.at("signal_id"));
    need(signal_map.contains(id) && model_map.emplace(id, &m).second,
         "source model identity mismatch");
  }
  Json rows = Json::array(), study_rows = Json::array(),
       kept_models = Json::array(), kept_signals = Json::array();
  std::set<std::string> selected;
  std::size_t unavailable = 0, unavailable_studies = 0;
  for (const auto &choice : selections) {
    deadline(end);
    keys(choice, {"signal_id", "transform", "mode", "nonexecution_pnl"});
    const auto id = str(choice.at("signal_id"));
    need(selected.insert(id).second && model_map.contains(id),
         "unique known selected signal required");
    Transform transform(choice.at("transform"));
    const auto mode = str(choice.at("mode"));
    need(mode == "support_only" || mode == "execution_mixture",
         "unsupported outcome interpretation");
    R no_fill_pnl;
    if (mode == "support_only")
      need(choice.at("nonexecution_pnl").is_null(),
           "support-only does not consume nonexecution P&L");
    else
      no_fill_pnl = r::read_ratio(choice.at("nonexecution_pnl"));
    const auto &m = *model_map.at(id);
    const auto measure = str(m.at("measure")),
               conditioning = str(m.at("conditioning"));
    need(measure == "probability" || measure == "scenario_weight" ||
             measure == "signed_coefficient",
         "unsupported measure domain");
    need(conditioning == "execution" || conditioning == "scenario",
         "unsupported source conditioning");
    need(m.at("price_unit") == "price_nanos" && m.at("price_scale") == "1e-9",
         "model price unit mismatch");
    const auto status = str(m.at("status"));
    need(status == "available" || status == "unavailable",
         "unsupported model status");
    std::string failure;
    if (status == "unavailable") {
      need(!str(m.at("reason")).empty(), "unavailable model reason required");
      failure = "source model unavailable: " + str(m.at("reason"));
    }
    const auto &support = m.at("support");
    need(support.is_array() && support.size() <= 65 &&
             (status == "available" ? !support.empty() : support.empty()),
         "model support shape mismatch");
    Json transformed = Json::array(), wire_atoms = Json::array();
    std::vector<Atom> atoms;
    R total;
    std::set<std::int64_t> prices;
    for (const auto &a : support) {
      const auto price = i64(a.at("price_nanos"));
      need(price != INT64_MAX && prices.insert(price).second,
           "unique defined source prices required");
      const auto weight = artifact_ratio(a.at("weight"));
      need(measure == "signed_coefficient" || weight.n >= 0,
           "measure cannot contain negative weight");
      if (measure == "probability")
        total = r::plus(total, weight);
      const auto gross = transform.gross(price);
      const auto net = r::plus(gross, negative(transform.cost));
      const auto value = r::times(net, inverse(transform.basis));
      transformed.push_back({{"price_nanos", dec(price)},
                             {"weight", r::wire(weight)},
                             {"gross_pnl", r::wire(gross)},
                             {"cost", r::wire(transform.cost)},
                             {"net_pnl", r::wire(net)},
                             {"return", r::wire(value)}});
      atoms.push_back({value, weight});
    }
    need(status != "available" || measure != "probability" ||
             same(total, {1, 1}),
         "source probabilities must sum to one");
    if (failure.empty() && mode == "execution_mixture") {
      if (measure != "probability" || conditioning != "execution")
        failure = "execution mixture requires price probabilities conditional "
                  "on execution";
      else if (m.at("fill_probability").at("status") == "unavailable")
        failure = "execution mixture requires separately supplied execution "
                  "likelihood";
      else {
        need(m.at("fill_probability").at("status") == "external_assumption" &&
                 m.at("no_fill_probability").at("status") ==
                     "external_assumption",
             "unsupported likelihood evidence contract");
        const auto fill = artifact_ratio(m.at("fill_probability").at("value"));
        const auto no_fill =
            artifact_ratio(m.at("no_fill_probability").at("value"));
        need(fill.n >= 0 && no_fill.n >= 0 &&
                 same(r::plus(fill, no_fill), {1, 1}),
             "likelihood complement mismatch");
        for (auto &a : atoms)
          a.weight = r::times(a.weight, fill);
        atoms.push_back(
            {r::times(no_fill_pnl, inverse(transform.basis)), no_fill});
      }
    }
    need(!reject || failure.empty(),
         "selected economic interpretation unavailable");
    if (failure.empty()) {
      for (std::size_t i = 0; i < atoms.size(); ++i)
        wire_atoms.push_back(
            {{"kind", i < support.size() ? "price_outcome" : "nonexecution"},
             {"support_index",
              i < support.size() ? Json(dec(i)) : Json(nullptr)},
             {"return", r::wire(atoms[i].value)},
             {"weight", r::wire(atoms[i].weight)}});
    } else {
      ++unavailable;
      atoms.clear();
    }
    const auto resolved_conditioning = mode == "execution_mixture"
                                           ? "execution_and_nonexecution"
                                           : conditioning;
    rows.push_back({{"protocol", "symphony.sbv.economic-outcome.v1"},
                    {"signal_id", id},
                    {"status", failure.empty() ? "available" : "unavailable"},
                    {"reason", failure},
                    {"measure", measure},
                    {"conditioning", resolved_conditioning},
                    {"mode", mode},
                    {"transform", choice.at("transform")},
                    {"transformed_support", transformed},
                    {"nonexecution_pnl", choice.at("nonexecution_pnl")},
                    {"atoms", wire_atoms},
                    {"calibration", "not_verified"}});
    for (const auto &study : studies) {
      auto row = study_result(study, id, measure, resolved_conditioning, atoms,
                              failure, reject);
      if (row.at("status") == "unavailable")
        ++unavailable_studies;
      study_rows.push_back(std::move(row));
    }
    kept_models.push_back(m);
    kept_signals.push_back(*signal_map.at(id));
  }
  auto result = base(
      "native exact user-selected linear economics of admitted model support");
  auto &s = result["sections"];
  const bool partial = unavailable || unavailable_studies;
  result["status"] = partial ? "partial" : "completed";
  s["summary"] =
      section({{"source_census_sha256", census.at("census_sha256")},
               {"source_signal_count", dec(signals.size())},
               {"selected_signal_count", dec(rows.size())},
               {"available_economic_outcomes", dec(rows.size() - unavailable)},
               {"unavailable_economic_outcomes", dec(unavailable)},
               {"unavailable_studies", dec(unavailable_studies)},
               {"selection_sha256", e::sha256_hex(selections.dump())}});
  s["economics"] = section(
      rows, unavailable ? "partial" : "available",
      unavailable ? "selected outcomes unavailable; inspect per-signal reasons"
                  : "");
  s["studies"] =
      section(study_rows, unavailable_studies ? "partial" : "available",
              unavailable_studies
                  ? "selected studies unavailable; inspect per-signal reasons"
                  : "");
  s["signals"] = section(kept_signals);
  s["execution"] = section(kept_models);
  // Preserve the full source replay verbatim. It may include unselected
  // signals; it remains observed evidence, separate from the economic overlay.
  s["replay"] = sections.at("replay");
  s["census"] = section(census);
  s["source_context"] = section({{"census", census},
                                 {"choices", sections.at("choices")},
                                 {"provenance", sections.at("provenance")},
                                 {"reference", reference}});
  if (sections.at("resources").at("data").contains("dataset_feed"))
    s["source_context"]["data"]["dataset_feed"] =
        sections.at("resources").at("data").at("dataset_feed");
  if (native_provider_model)
    s["source_context"]["data"]["provider"] =
        sections.at("provider").at("data");
  auto choices = p;
  choices.erase("output_path");
  s["choices"] = section(choices);
  s["provenance"] = section(
      {{"engine_version", version},
       {"source_path", selection.at("path")},
       {"source_pointer", selection.at("pointer")},
       {"source_outer_content_sha256", outer.at("content_sha256")},
       {"source_content_sha256", source.at("content_sha256")},
       {"source_file_sha256", e::sha256_hex(bytes)},
       {"source_authorship", "not_verified"},
       {"replay_scope", "full source replay copied verbatim"},
       {"numeric_contract",
        "exact reduced int64 rationals; overflow rejects; no rounding"}});
  s["resources"] = section({{"backend", "cpu"}, {"workers", "1"}});
  s["diagnostics"] = section(Json::array(
      {"Per-signal scenario economics only; no cross-signal independence, "
       "trade pairing, portfolio or account simulation is inferred.",
       "Position, value, costs and return basis are explicit user assumptions; "
       "calibration is not verified."}));
  s["user_extensions"] = section(p.at("extensions"));
  return persist(std::move(result), p, "economics", end);
}

Json economic_transforms() {
  return Json::array(
      {{{"id", "linear_price_pnl"},
        {"version", "1"},
        {"operations", Json::array({"economics"})},
        {"definition",
         "gross=(price-reference)/price_unit * position_units * "
         "value_per_price_unit; net=gross-cost; return=net/return_basis"},
        {"domain", "signed position, value, cost and nonzero signed return "
                   "basis; no wealth-floor restriction"},
        {"input_schema", "economics/$defs/economic_transform"},
        {"output_schema", "economics/$defs/economic_outcome"}}});
}
Json economic_studies() {
  Json cards = Json::array();
  for (const auto *id :
       {"weighted_return_sum", "return_moments", "return_quantiles"})
    cards.push_back(
        {{"id", id},
         {"version", "1"},
         {"operations", Json::array({"economics"})},
         {"unit", std::string(id) == "return_moments"
                      ? "return and return_squared"
                      : "return"},
         {"definition",
          std::string(id) == "weighted_return_sum"
              ? "exact sum(weight * return), without normalization"
          : std::string(id) == "return_moments"
              ? "exact mean and population variance"
              : "inverse CDF on positive probability mass; q=0 minimum; first "
                "cumulative >= q"},
         {"descriptor",
          {{"protocol", "symphony.sbv.study-descriptor.v1"},
           {"required_inputs",
            Json::array({"symphony.sbv.economic-outcome.v1"})},
           {"weighting_domains",
            std::string(id) == "weighted_return_sum"
                ? Json::array(
                      {"probability", "scenario_weight", "signed_coefficient"})
                : Json::array({"probability"})},
           {"parameters",
            std::string(id) == "return_quantiles"
                ? Json::array({"levels: 1..32 exact ratios in [0,1]"})
                : Json::array()},
           {"grouping",
            "each explicitly selected signal; no cross-signal aggregation"},
           {"missingness", "user selects unavailable or reject; no exclusion "
                           "or renormalization"},
           {"method", "exact finite support; int64 rational overflow rejects"},
           {"backend", "cpu"},
           {"maximum_atoms", "66"},
           {"output_schema", "economics/$defs/economic_study"}}}});
  for (const auto *id : {"terminal_moments", "terminal_quantiles"}) {
    const bool moments = std::string(id) == "terminal_moments";
    cards.push_back(
        {{"id", id},
         {"version", "1"},
         {"operations", Json::array({"compose_economics"})},
         {"unit",
          moments ? "caller state unit and its square" : "caller state unit"},
         {"definition",
          moments ? "exact probability mean and population variance of the "
                    "selected terminal state distribution"
                  : "inverse CDF on positive terminal probability; q=0 "
                    "minimum; otherwise first cumulative probability >= q; "
                    "modeled outcomes, not parameter confidence"},
         {"descriptor",
          {{"protocol", "symphony.sbv.study-descriptor.v1"},
           {"required_inputs",
            Json::array({"compose_economics terminal_atoms: exact state and "
                         "probability"})},
           {"weighting_domains", Json::array({"probability"})},
           {"parameters",
            moments ? Json::array()
                    : Json::array({"levels: exact int128 ratios in [0,1]; "
                                   "empty selection is valid"})},
           {"grouping", "one selected composed terminal state distribution"},
           {"missingness", "unavailable composition yields unavailable study; "
                           "no exclusion or renormalization"},
           {"method", moments ? "exact probability population moments"
                              : "inverse_cdf_positive_probability"},
           {"numeric_contract", "exact reduced int128 rationals; overflow "
                                "rejects; no rounding or normalization"},
           {"uncertainty", "modeled outcome distribution; no "
                           "parameter-estimation confidence claim"},
           {"backend", "cpu"},
           {"maximum_atoms", nullptr},
           {"output_schema",
            "compose_economics/$defs/composed_economic_study"}}}});
  }
  return cards;
}
} // namespace symphony::sbv::detail
