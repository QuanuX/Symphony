#include "provider.hpp"
#include "rational.hpp"
#include <algorithm>
#include <symphony/sbv/models.hpp>
namespace symphony::sbv {
namespace d = detail;
namespace r = detail::rational;
namespace {
void label(const Json &v) {
  const auto s = d::str(v);
  d::need(!s.empty() && s.size() <= 256, "bounded nonempty identity required");
}
void producer(const Json &p) {
  d::keys(p, {"id", "version", "artifact_sha256", "reproducibility"});
  label(p.at("id"));
  label(p.at("version"));
  const auto hash = d::str(p.at("artifact_sha256"));
  d::need(hash.empty() || (hash.size() == 64 &&
                           std::all_of(hash.begin(), hash.end(),
                                       [](char c) {
                                         return (c >= '0' && c <= '9') ||
                                                (c >= 'a' && c <= 'f');
                                       })),
          "producer digest must be empty or lowercase SHA256");
  const auto mode = d::str(p.at("reproducibility"));
  d::need(mode == "deterministic_declared" || mode == "nondeterministic" ||
              mode == "uncaptured",
          "unsupported reproducibility declaration");
}
Json outcome(const ModelFrame &f, const std::string &id) {
  return {
      {"protocol", "symphony.sbv.model-outcome.v1"},
      {"signal_id", f.signal_id},
      {"model_id", id},
      {"model_version", "1"},
      {"status", "available"},
      {"reason", ""},
      {"price_unit", "price_nanos"},
      {"price_scale", "1e-9"},
      {"horizon_end_ns", d::dec(f.horizon_end_ns)},
      {"horizon_within_observed_span", f.horizon_within_observed_span},
      {"calibration", "not_verified"},
      {"support", Json::array()},
      {"fill_probability",
       d::missing("not supplied; conditional prices do not imply execution")},
      {"no_fill_probability", d::missing("execution probability unavailable")},
      {"fill_quantity", d::missing("not modeled by this contract")},
      {"fill_time", d::missing("not modeled by this contract")},
      {"actual_execution", d::missing("no actual execution supplied")}};
}
} // namespace
AdmittedModel::AdmittedModel(Json selection,
                             const std::vector<std::string> &ids)
    : selection_(std::move(selection)) {
  d::keys(selection_,
          {"protocol", "id", "version", "horizon_ns", "parameters"});
  d::need(selection_.at("protocol") == "symphony.sbv.model-selection.v1" &&
              selection_.at("version") == "1",
          "model contract/version mismatch");
  (void)d::u64(selection_.at("horizon_ns"));
  const auto id = d::str(selection_.at("id"));
  const auto &p = selection_.at("parameters");
  std::set<std::string> expected(ids.begin(), ids.end());
  d::need(ids.size() <= 4096 && expected.size() == ids.size(),
          "unique bounded model signal set required");
  for (const auto &name : ids)
    label(name);
  if (id == "observed_trade_levels") {
    d::keys(p, {"levels_per_side", "include_anchor", "thin_support"});
    const auto n = d::u64(p.at("levels_per_side"));
    d::need(n >= 1 && n <= 32 && p.at("include_anchor").is_boolean(),
            "observed support parameters");
    d::need(p.at("thin_support") == "unavailable" ||
                p.at("thin_support") == "reject",
            "explicit thin-support policy required");
  } else if (id == "external_outcomes") {
    d::keys(p, {"producer", "measure", "conditioning", "calibration_reference",
                "outcomes"});
    producer(p.at("producer"));
    (void)d::str(p.at("calibration_reference"));
    const auto measure = d::str(p.at("measure")),
               conditioning = d::str(p.at("conditioning"));
    d::need(measure == "probability" || measure == "scenario_weight" ||
                measure == "signed_coefficient",
            "unsupported measure domain");
    d::need(conditioning == "execution" || conditioning == "scenario",
            "conditioning required");
    const auto &outcomes = p.at("outcomes");
    d::need(outcomes.is_array() && outcomes.size() == ids.size(),
            "one outcome per census signal required");
    for (const auto &o : outcomes) {
      d::keys(o, {"signal_id", "support", "execution_probability",
                  "evidence_reference"});
      const auto signal = d::str(o.at("signal_id"));
      d::need(expected.erase(signal) == 1,
              "unknown or duplicate model signal identity");
      (void)d::str(o.at("evidence_reference"));
      const auto &support = o.at("support");
      d::need(support.is_array() && !support.empty() && support.size() <= 64,
              "support bound 1..64");
      r::R total;
      std::set<std::int64_t> prices;
      for (const auto &a : support) {
        d::keys(a, {"price_nanos", "weight"});
        const auto price = d::i64(a.at("price_nanos"));
        d::need(price != INT64_MAX && prices.insert(price).second,
                "unique defined prices required");
        auto w = r::read_ratio(a.at("weight"));
        d::need(measure == "signed_coefficient" || w.n >= 0,
                "negative mass requires signed coefficient domain");
        if (measure == "probability")
          total = r::plus(total, w);
      }
      d::need(measure != "probability" || total.n == total.d,
              "probabilities must sum exactly to one; no renormalization");
      const auto &fill = o.at("execution_probability");
      d::need(fill.is_object() && fill.contains("status"),
              "execution probability status required");
      if (fill.at("status") == "supplied") {
        d::keys(fill, {"status", "value"});
        const auto value = r::read_ratio(fill.at("value"));
        d::need(value.n >= 0 && value.n <= value.d,
                "separately supplied execution likelihood must be in [0,1]");
      } else {
        d::keys(fill, {"status", "reason"});
        d::need(fill.at("status") == "unavailable" &&
                    !d::str(fill.at("reason")).empty(),
                "unavailable execution probability needs reason");
      }
      supplied_.emplace(signal, o);
    }
  } else
    d::need(false, "model not installed");
}
std::uint64_t AdmittedModel::horizon_ns() const {
  return d::u64(selection_.at("horizon_ns"));
}
namespace {
Json provider_candidate_selection(const Json &selection, const Json &author,
                                  const Json &candidates) {
  d::keys(selection, {"protocol", "id", "version", "horizon_ns", "parameters"});
  d::need(selection.at("protocol") == "symphony.sbv.model-selection.v1" &&
              selection.at("id") == "native_provider" &&
              selection.at("version") == "1",
          "native provider model selection mismatch");
  const auto &p = selection.at("parameters");
  d::keys(p, {"provider", "measure", "conditioning", "calibration_reference",
              "on_unavailable"});
  d::need(p.at("provider").at("role") == "model" &&
              (p.at("on_unavailable") == "reject" ||
               p.at("on_unavailable") == "unavailable"),
          "native model role/unavailable policy required");
  return {{"protocol", "symphony.sbv.model-selection.v1"},
          {"id", "external_outcomes"},
          {"version", "1"},
          {"horizon_ns", selection.at("horizon_ns")},
          {"parameters",
           {{"producer", author},
            {"measure", p.at("measure")},
            {"conditioning", p.at("conditioning")},
            {"calibration_reference", p.at("calibration_reference")},
            {"outcomes", candidates}}}};
}
} // namespace
void validate_native_provider_model(const Json &selection, const Json &author) {
  (void)AdmittedModel(
      provider_candidate_selection(selection, author, Json::array()), {});
}
void validate_native_provider_model_context(const Json &selection,
                                            const Json &captured,
                                            const Json &outcomes) {
  d::keys(captured, {"evidence", "role"});
  d::need(captured.at("role") == "model",
          "retained model provider role required");
  const auto &evidence = captured.at("evidence");
  const auto &parameters = selection.at("parameters");
  d::validate_native_provider_evidence(parameters.at("provider"), evidence);
  const Json author{
      {"id", evidence.at("id")},
      {"version", evidence.at("version")},
      {"artifact_sha256", evidence.at("library").at("expected_sha256")},
      {"reproducibility", evidence.at("reproducibility")}};
  validate_native_provider_model(selection, author);
  d::need(outcomes.is_array(), "retained model outcome array required");
  for (const auto &outcome : outcomes)
    d::need(outcome.at("protocol") == "symphony.sbv.model-outcome.v1" &&
                outcome.at("model_id") == "native_provider" &&
                outcome.at("model_version") == selection.at("version") &&
                outcome.at("evidence_origin") == "native_provider" &&
                outcome.at("producer") == author &&
                outcome.at("measure") == parameters.at("measure") &&
                outcome.at("conditioning") == parameters.at("conditioning") &&
                outcome.at("calibration_reference") ==
                    parameters.at("calibration_reference"),
            "retained model provider attribution mismatch");
}
Json admit_native_provider_outcome(const Json &selection, const Json &author,
                                   const Json &candidate, const Json &error,
                                   const ModelFrame &frame, std::int64_t end) {
  d::deadline(end);
  Json result;
  if (error.is_null()) {
    const AdmittedModel admitted(
        provider_candidate_selection(selection, author,
                                     Json::array({candidate})),
        {frame.signal_id});
    result = admitted.evaluate(frame, end);
    result["model_id"] = "native_provider";
  } else {
    validate_native_provider_model(selection, author);
    d::need(selection.at("parameters").at("on_unavailable") == "unavailable",
            ("native model unavailable: " + error.dump()).c_str());
    result = outcome(frame, "native_provider");
    result["status"] = "unavailable";
    // Canonical JSON text preserves every attributed error field without
    // coercing arbitrary provider details into the exact-number result tree.
    result["reason"] = error.dump();
    result["measure"] = selection.at("parameters").at("measure");
    result["conditioning"] = selection.at("parameters").at("conditioning");
    result["producer"] = author;
    result["calibration_reference"] =
        selection.at("parameters").at("calibration_reference");
    result["evidence_reference"] = "native provider returned unavailable";
  }
  result["evidence_origin"] = "native_provider";
  return result;
}
Json AdmittedModel::evaluate(const ModelFrame &f, std::int64_t end) const {
  d::deadline(end);
  d::need(f.horizon_end_ns >= f.available_ns &&
              f.horizon_end_ns - f.available_ns == horizon_ns(),
          "model horizon mismatch");
  auto result = outcome(f, d::str(selection_.at("id")));
  const auto &p = selection_.at("parameters");
  if (selection_.at("id") == "external_outcomes") {
    const auto it = supplied_.find(f.signal_id);
    d::need(it != supplied_.end(), "model signal not admitted");
    const auto &o = it->second;
    result["evidence_origin"] = "externally_supplied";
    result["measure"] = p.at("measure");
    result["conditioning"] = p.at("conditioning");
    result["producer"] = p.at("producer");
    result["calibration_reference"] = p.at("calibration_reference");
    result["evidence_reference"] = o.at("evidence_reference");
    result["support"] = o.at("support");
    if (o.at("execution_probability").at("status") == "supplied") {
      const auto fill =
          r::read_ratio(o.at("execution_probability").at("value"));
      result["fill_probability"] = {{"status", "external_assumption"},
                                    {"value", r::wire(fill)}};
      result["no_fill_probability"] = {
          {"status", "external_assumption"},
          {"value", r::wire(r::plus({1, 1}, {-fill.n, fill.d}))}};
    } else {
      result["fill_probability"] =
          d::missing(d::str(o.at("execution_probability").at("reason")));
    }
    return result;
  }
  std::map<std::int64_t, ObservedTrade> observed;
  std::uint64_t ordinal = f.source_ordinal, time = f.available_ns;
  for (const auto &t : f.future_trades) {
    d::deadline(end);
    d::need(t.source_ordinal > ordinal && t.available_ns >= time &&
                t.available_ns <= f.horizon_end_ns && t.size > 0 &&
                t.price_nanos != INT64_MAX,
            "model frame must contain ordered defined future trades within "
            "horizon");
    observed.try_emplace(t.price_nanos, t);
    ordinal = t.source_ordinal;
    time = t.available_ns;
  }
  const auto n = d::u64(p.at("levels_per_side"));
  auto lo = observed.lower_bound(f.anchor_price_nanos),
       hi = observed.upper_bound(f.anchor_price_nanos);
  std::vector<ObservedTrade> selected;
  std::size_t below = 0, above = 0;
  while (lo != observed.begin() && below < n) {
    --lo;
    selected.push_back(lo->second);
    ++below;
  }
  while (hi != observed.end() && above < n) {
    selected.push_back(hi->second);
    ++hi;
    ++above;
  }
  result["evidence_origin"] = "observed_post_signal_trades";
  result["measure"] = "probability";
  result["conditioning"] = "execution";
  result["assumption"] =
      "equal conditional price mass over selected distinct observed trade "
      "levels; not book liquidity or fill likelihood";
  result["observed_below_count"] = d::dec(below);
  result["observed_above_count"] = d::dec(above);
  result["observed_candidates"] = Json::array();
  for (const auto &t : selected)
    result["observed_candidates"].push_back(
        {{"price_nanos", d::dec(t.price_nanos)},
         {"source_ordinal", d::dec(t.source_ordinal)},
         {"available_ns", d::dec(t.available_ns)}});
  if (!f.anchor_is_observed_trade || below != n || above != n ||
      !f.horizon_within_observed_span) {
    if (p.at("thin_support") == "reject")
      d::need(false, "selected observed-level model lacks anchor, full support "
                     "or observed horizon");
    result["status"] = "unavailable";
    result["reason"] = "requires an observed trade anchor, requested levels on "
                       "both sides and horizon inside observed span";
    return result;
  }
  if (p.at("include_anchor").get<bool>())
    selected.push_back(
        {f.source_ordinal, f.available_ns, f.anchor_price_nanos, 0});
  std::sort(selected.begin(), selected.end(), [](const auto &a, const auto &b) {
    return a.price_nanos < b.price_nanos;
  });
  for (const auto &t : selected)
    result["support"].push_back(
        {{"price_nanos", d::dec(t.price_nanos)},
         {"weight", r::wire({1, static_cast<std::int64_t>(selected.size())})},
         {"source_ordinal", d::dec(t.source_ordinal)},
         {"available_ns", d::dec(t.available_ns)}});
  return result;
}
Json model_catalogue() {
  Json result = {
      {"protocol", "symphony.sbv.catalogue.v1"},
      {"engine_version", version},
      {"models",
       Json::array(
           {{{"id", "observed_trade_levels"},
             {"version", "1"},
             {"input_protocol", "symphony.sbv.model-selection.v1"},
             {"output_protocol", "symphony.sbv.model-outcome.v1"},
             {"measure", "conditional_probability"},
             {"evidence", "defined positive-size post-signal trades; not "
                          "initialized book or executable liquidity"},
             {"parameters", Json::array({"levels_per_side", "include_anchor",
                                         "thin_support"})}},
            {{"id", "external_outcomes"},
             {"version", "1"},
             {"input_protocol", "symphony.sbv.model-selection.v1"},
             {"output_protocol", "symphony.sbv.model-outcome.v1"},
             {"measure",
              "explicit probability, scenario_weight or signed_coefficient"},
             {"evidence", "external claims and exact declared signal identity; "
                          "calibration not verified"},
             {"parameters",
              Json::array({"producer", "measure", "conditioning",
                           "calibration_reference", "outcomes"})}}})},
      {"studies",
       Json::array(
           {{{"id", "signal_summary"},
             {"version", "1"},
             {"unit", "count"},
             {"definition", "exact admitted census size"}},
            {{"id", "model_summary"},
             {"version", "1"},
             {"unit", "count"},
             {"definition", "available/unavailable selected-model outcomes; no "
                            "probability averaging"}},
            {{"id", "path_excursion"},
             {"version", "1"},
             {"unit", "price_nanos"},
             {"definition",
              "minimum/maximum observed post-signal trade minus anchor with "
              "first extremum cursor; not fill-based MAE/MFE"}}})}};
  result["models"].push_back(
      {{"id", "native_provider"},
       {"version", "1"},
       {"input_protocol", "symphony.sbv.model-selection.v1"},
       {"output_protocol", "symphony.sbv.model-outcome.v1"},
       {"measure",
        "explicit probability, scenario_weight or signed_coefficient"},
       {"evidence",
        "Explicitly selected trusted native provider with full MBO "
        "causal/followup views; host admits exact candidate values. Execution "
        "and calibration remain unverified assumptions."},
       {"parameters", Json::array({"provider", "measure", "conditioning",
                                   "calibration_reference", "on_unavailable"})},
       {"concurrency",
        Json::array({"serialized_instance", "per_worker_instances",
                     "shared_reentrant_instance"})}});
  for (auto &card : result["models"])
    card["operations"] = Json::array({"evaluate"});
  for (auto &card : result["studies"])
    card["operations"] = card.at("id") == "signal_summary"
                             ? Json::array({"run", "evaluate"})
                             : Json::array({"evaluate"});
  result["studies"].push_back(
      {{"id", "forward_markout"},
       {"version", "1"},
       {"unit", "price_nanos"},
       {"definition", "last observed post-signal trade at/before the complete "
                      "observed horizon minus anchor; not P&L"},
       {"operations", Json::array({"run"})}});
  for (const auto *id : {"none", "touch_observation", "user_probability"})
    result["models"].push_back(
        {{"id", id},
         {"version", "1"},
         {"input_protocol", "symphony.sbv.run-input.v1"},
         {"output_protocol", "symphony.sbv.result.v1"},
         {"measure",
          "observed touches; probability only when explicitly supplied"},
         {"evidence",
          "legacy run profile; no calibration or actual fill claim"},
         {"parameters",
          Json::array({"horizon_ns", "price_offsets_nanos",
                       "probability_numerator", "probability_denominator"})},
         {"operations", Json::array({"run"})}});
  for (const auto &card : d::economic_studies())
    result["studies"].push_back(card);
  result["models"].push_back(Json::parse(
      R"LIQ({"id":"displayed_depth_sweep","version":"1","input_protocol":"symphony.sbv.liquidity-input.v1","output_protocol":"symphony.sbv.liquidity-outcome.v1","measure":"conditional deterministic snapshot scenario; activation probability only when supplied","evidence":"initialized retained depth under explicit participation, depth and market-state policies; no calibrated execution or market response","parameters":["orders","liquidity_mode","depth_policy","market_state_policy","receive_time_policy","on_unavailable","workers"],"operations":["liquidity"]})LIQ"));
  for (
      const auto &card : Json::parse(
          R"LIQ([{"id":"liquidity_summary","version":"1","unit":"count","definition":"conditional available/full/partial/no-fill counts; no frequency-estimated probability","operations":["liquidity"]},{"id":"fill_quality","version":"1","unit":"quantity ratio and price_nanos","definition":"exact conditional filled/requested ratio and side-adjusted VWAP minus signal anchor; no account P&L","operations":["liquidity"]}])LIQ"))
    result["studies"].push_back(card);
  result["transforms"] = d::economic_transforms();
  result["transforms"].push_back(Json::parse(
      R"ALLOC({"id":"filled_quantity_markout","version":"1","operations":["allocation_economics"],"definition":"gross=side*(mark*filled_quantity-sum(fill_price*fill_quantity))/price_unit*value; net=gross-activation_cost-filled_order_cost-per_unit_cost*filled_quantity; return=net/basis","domain":"caller mark and units; signed multiplier and costs; nonzero signed return basis; filled-order cost only for nonzero fill","input_schema":"allocation_economics/$defs/allocation_valuation","output_schema":"allocation_economics/$defs/allocation_economic_outcome"})ALLOC"));
  for (
      const auto &card : Json::parse(
          R"ALLOC([{"id":"allocation_costs","version":"1","unit":"caller P&L unit","definition":"separate activation, filled-order and filled-unit costs; negative costs represent caller rebates","operations":["allocation_economics"]},{"id":"activation_moments","version":"1","unit":"caller P&L unit, its square and dimensionless returns","definition":"per-order two-branch activation mean and variance; no cross-order joint assumption","operations":["allocation_economics"]}])ALLOC"))
    result["studies"].push_back(card);
  for (
      const auto &card : Json::parse(
          R"SERIES([{"id":"series_summary","version":"1","unit":"caller unit","definition":"Count, exact sum and extrema without normalizing supplied weights","operations":["analyze"]},{"id":"series_moments","version":"1","unit":"caller unit and square","definition":"Exact probability mean with selected population or equal-weight sample variance","operations":["analyze"]},{"id":"series_quantiles","version":"1","unit":"caller unit","definition":"Exact inverse CDF on positive probability mass","operations":["analyze"]},{"id":"equity_drawdown","version":"1","unit":"caller equity unit and ratio","definition":"Ordered supplied valuations, absolute/relative peak drawdown and recovery index","operations":["analyze"]},{"id":"return_ratios","version":"1","unit":"return ratio","definition":"Caller benchmark and annualization; explicit binary64 square roots, round-trip decimal and bit identity","operations":["analyze"]}])SERIES"))
    result["studies"].push_back(card);
  for (
      const auto &card : Json::parse(
          R"BOOT([{"id":"bootstrap_mean_distribution","version":"1","unit":"caller unit and square","definition":"Exact population moments of deterministic replicate means","operations":["resample"]},{"id":"bootstrap_mean_quantiles","version":"1","unit":"caller unit and square","definition":"Inverse empirical CDF of uniform-row or block bootstrap means; no confidence coverage claim","operations":["resample"]}])BOOT"))
    result["studies"].push_back(card);
  result["studies"].push_back(
      {{"id", "regression_errors"},
       {"version", "1"},
       {"unit", "caller target unit and square"},
       {"operations", Json::array({"predict"})},
       {"definition",
        "Exact weighted residual, absolute and squared error sums and "
        "algebraic weight-normalized means. Signed weights do not define "
        "probability; zero total weight leaves means unavailable."}});
  result["book_profiles"] = Json::parse(
      R"BOOK([{"id":"databento_mbo_orders_strict_v1","version":"1","operations":["book"],"initialization":"source reset or exact source-bound supplied checkpoint","anomalies":"reject or invalidate_until_reset","scope":"one publisher/instrument/channel; A/M/C/R updates; T/F/N informational; no queue priority or fill prediction","input_schema":"symphony.sbv.book-input.v1","output_schemas":["symphony.sbv.book-frame.v1","symphony.sbv.book-checkpoint.v1"]}])BOOK");
  return result;
}
} // namespace symphony::sbv
