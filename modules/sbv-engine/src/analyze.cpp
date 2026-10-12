#include "binary_studies.hpp"
#include "consumer_source.hpp"
#include "study_formula.hpp"
#include "wide_rational.hpp"
#include <algorithm>
#include <bit>
#include <cfenv>
#include <cmath>
#include <limits>
#include <optional>
namespace symphony::sbv::detail {
namespace {
namespace w = wide_rational;
using Sample = formula::Sample;
const Json *point(const Json &v, const std::string &p) {
  need(p.size() <= 4096 && (p.empty() || p.front() == '/'),
       "bounded series pointer required");
  Json::json_pointer ptr(p);
  return v.contains(ptr) ? &v.at(ptr) : nullptr;
}
Json available(w::R r) {
  return {{"status", "available"}, {"value", w::wire(r)}};
}
w::R negative(w::R r) { return {-r.n, r.d}; }
Json approximate(double v) {
  if (!std::isfinite(v))
    return missing("nonfinite binary64 estimate");
  char decimal[64], bits[17];
  auto [tail, ec] = std::to_chars(decimal, decimal + sizeof(decimal), v,
                                  std::chars_format::general,
                                  std::numeric_limits<double>::max_digits10);
  need(ec == std::errc{}, "binary64 conversion failed");
  auto word = std::bit_cast<std::uint64_t>(v);
  constexpr char hex[] = "0123456789abcdef";
  for (unsigned i = 0; i < 16; ++i)
    bits[15 - i] = hex[(word >> (i * 4)) & 15];
  bits[16] = 0;
  return {{"status", "available"},
          {"value",
           {{"numeric_profile", "ieee754_binary64_roundtrip"},
            {"decimal", std::string(decimal, tail)},
            {"bits_hex", std::string(bits)}}}};
}
double real(w::R r) {
  return static_cast<double>(r.n) / static_cast<double>(r.d);
}
} // namespace
static logical::Value analyze_body(const Json &p, PartitionedOutput *context,
                                   std::int64_t end) {
  keys_optional(p,
                {"protocol", "pointer", "value_pointer", "value_type",
                 "weighting", "weight_pointer", "series_role", "unit",
                 "input_description", "missing", "studies", "quantiles",
                 "variance_estimator", "relative_drawdown", "benchmark",
                 "periods_per_year", "numeric_profile", "on_incompatible",
                 "extensions"},
                {"path", "expected_sha256", "source", "output_path", "output",
                 "study_definitions", "binary"});
  need(p.at("extensions").is_object(), "extensions object required");
  const auto role = str(p.at("series_role")), measure = str(p.at("weighting")),
             numeric = str(p.at("numeric_profile"));
  need(role == "observations" || role == "returns" || role == "equity",
       "unknown series role");
  need(measure == "equal_probability" || measure == "supplied_probability" ||
           measure == "scenario_weight" || measure == "signed_coefficient",
       "unknown series measure");
  need(p.at("value_type") == "integer" || p.at("value_type") == "rational",
       "unsupported series value type");
  need(numeric == "exact_rational" || numeric == "binary64_roundtrip",
       "unknown numeric profile");
  need(p.at("missing") == "exclude" || p.at("missing") == "reject",
       "missing policy required");
  need(p.at("on_incompatible") == "unavailable" ||
           p.at("on_incompatible") == "reject",
       "incompatibility policy required");
  need(p.at("variance_estimator") == "population" ||
           p.at("variance_estimator") == "sample_equal_weight",
       "variance estimator required");
  need(p.at("relative_drawdown") == "absolute_peak" ||
           p.at("relative_drawdown") == "positive_peak_only",
       "drawdown convention required");
  need(!str(p.at("unit")).empty() && str(p.at("unit")).size() <= 128 &&
           !str(p.at("input_description")).empty() &&
           str(p.at("input_description")).size() <= 4096,
       "unit and input description required");
  const auto benchmark = w::parameter(p.at("benchmark")),
             periods = w::parameter(p.at("periods_per_year"));
  need(periods.n > 0, "positive periods-per-year convention required");
  const bool equal = measure == "equal_probability";
  if (equal)
    need(p.at("weight_pointer").is_null(),
         "equal weighting has no supplied weight pointer");
  else
    need(p.at("weight_pointer").is_string(),
         "supplied weight pointer required");
  const std::set<std::string> builtins{"series_summary",     "series_moments",
                                       "series_quantiles",   "equity_drawdown",
                                       "return_ratios",      "binary_confusion",
                                       "binary_probability", "binary_ranking"};
  std::map<std::string, Json> definitions;
  if (p.contains("study_definitions")) {
    need(p.at("study_definitions").is_array(),
         "study definitions array required");
    for (const auto &definition : p.at("study_definitions")) {
      keys(definition,
           {"protocol", "id", "version", "unit", "description", "expression"});
      need(definition.at("protocol") == "symphony.sbv.exact-series-formula.v1",
           "unsupported study definition protocol");
      for (auto key : {"id", "version", "unit", "description"})
        need(!str(definition.at(key)).empty(),
             "nonempty study descriptor text required");
      const auto id = str(definition.at("id"));
      need(!builtins.contains(id) && definitions.emplace(id, definition).second,
           "duplicate or builtin study identity");
      formula::validate(definition.at("expression"), false, end);
    }
  }
  std::set<std::string> selected;
  need(p.at("studies").is_array(), "series study selection array required");
  for (const auto &v : p.at("studies")) {
    auto id = str(v);
    need((builtins.contains(id) || definitions.contains(id)) &&
             selected.insert(id).second,
         "unknown/duplicate series study");
  }
  const bool binary_selected =
      std::any_of(selected.begin(), selected.end(), binary::study);
  need(p.contains("binary") == binary_selected,
       "binary choices required only when a binary study is selected");
  need(p.at("quantiles").is_array() &&
           (selected.contains("series_quantiles") ? !p.at("quantiles").empty()
                                                  : p.at("quantiles").empty()),
       "quantile selection required only for selected study");
  std::vector<w::R> quantiles;
  for (const auto &q : p.at("quantiles")) {
    auto r = w::parameter(q);
    need(r.n >= 0 && r.n <= r.d, "quantile outside [0,1]");
    quantiles.push_back(r);
  }
  auto admitted = consumer::source(p, end);
  const auto source = admitted.value;
  const auto data = consumer::pointer(source, str(p.at("pointer")));
  need(data.kind() == "array", "source series array required");
  const auto value_pointer = str(p.at("value_pointer")),
             weight_pointer = equal ? "" : str(p.at("weight_pointer"));
  (void)point(Json::object(), value_pointer);
  if (!equal)
    (void)point(Json::object(), weight_pointer);
  std::vector<Sample> samples;
  consumer::Rows excluded(context, "exclusions");
  std::size_t idx = 0;
  auto children = data.children();
  while (auto child = children.next()) {
    deadline(end);
    const auto row = child->value.materialize();
    const auto source_index = idx++;
    const auto *v = point(row, value_pointer);
    const auto *weight = equal ? nullptr : point(row, weight_pointer);
    if (!v || (!equal && !weight)) {
      need(p.at("missing") != "reject", "series value/weight field absent");
      excluded.append(
          {{"source_index", dec(source_index)},
           {"reason", "value or weight field absent; no imputation or "
                      "supplied-weight normalization"}});
      continue;
    }
    auto value = p.at("value_type") == "integer" ? w::R{w::integer(*v), 1}
                                                 : w::artifact(*v);
    auto mass = equal ? w::R{} : w::artifact(*weight);
    need(measure == "signed_coefficient" || mass.n >= 0,
         "negative probability/scenario weight");
    samples.push_back({source_index, value, mass});
  }
  if (equal && !samples.empty())
    for (auto &sample : samples)
      sample.weight = {1, static_cast<w::I>(samples.size())};
  const bool probability = equal || measure == "supplied_probability";
  const bool need_mass = selected.contains("series_summary") ||
                         selected.contains("series_moments") ||
                         selected.contains("series_quantiles") ||
                         selected.contains("return_ratios");
  w::R mass{}, sum{};
  if (need_mass)
    for (const auto &s : samples) {
      deadline(end);
      mass = w::plus(mass, s.weight);
      if (selected.contains("series_summary") ||
          selected.contains("series_moments") ||
          selected.contains("return_ratios"))
        sum = w::plus(sum, w::times(s.weight, s.value));
    }
  const bool normalized =
      probability && !samples.empty() && w::equal(mass, {1, 1});
  Json studies = Json::array();
  std::optional<logical::Value> drawdown_curve;
  std::size_t drawdown_study_index = 0;
  auto add = [&](const std::string &id, Json data,
                 const std::string &reason = "") {
    need(reason.empty() || p.at("on_incompatible") != "reject", reason.c_str());
    studies.push_back({{"study_id", id},
                       {"version", "1"},
                       {"status", reason.empty() ? "available" : "unavailable"},
                       {"reason", reason},
                       {"data", reason.empty() ? data : Json(nullptr)}});
  };
  if (selected.contains("series_summary")) {
    Json min = missing("no retained observations"), max = min;
    if (!samples.empty()) {
      auto lo = samples[0].value, hi = lo;
      for (const auto &s : samples) {
        if (w::compare(s.value, lo) < 0)
          lo = s.value;
        if (w::compare(s.value, hi) > 0)
          hi = s.value;
      }
      min = available(lo);
      max = available(hi);
    }
    add("series_summary",
        {{"count", dec(samples.size())},
         {"weight_total", w::wire(mass)},
         {"weighted_sum", w::wire(sum)},
         {"minimum", min},
         {"maximum", max},
         {"definition",
          "extrema over retained observations, including zero-weight "
          "observations; weighted sum never normalizes supplied weights"}});
  }
  auto probability_reason = [&] {
    return samples.empty() ? "no retained observations"
           : !probability ? "study requires a probability measure"
                          : "retained probability mass does not sum to one; no "
                            "normalization selected";
  };
  std::optional<w::R> variance;
  std::string variance_reason;
  if (selected.contains("series_moments") ||
      selected.contains("return_ratios")) {
    if (normalized) {
      w::R v{};
      for (const auto &s : samples) {
        deadline(end);
        auto delta = w::plus(s.value, negative(sum));
        v = w::plus(v, w::times(s.weight, w::times(delta, delta)));
      }
      if (p.at("variance_estimator") == "sample_equal_weight") {
        if (!equal)
          variance_reason =
              "sample estimator requires equal observational weights";
        else if (samples.size() < 2)
          variance_reason = "sample variance needs at least two observations";
        else
          v = w::times(v, {static_cast<w::I>(samples.size()),
                           static_cast<w::I>(samples.size() - 1)});
      }
      if (variance_reason.empty())
        variance = v;
    } else
      variance_reason = probability_reason();
  }
  if (selected.contains("series_moments")) {
    if (!normalized)
      add("series_moments", nullptr, probability_reason());
    else {
      need(variance || p.at("on_incompatible") != "reject",
           variance_reason.c_str());
      add("series_moments", {{"mean", w::wire(sum)},
                             {"variance", variance ? available(*variance)
                                                   : missing(variance_reason)},
                             {"estimator", p.at("variance_estimator")},
                             {"measure", measure}});
    }
  }
  if (selected.contains("series_quantiles")) {
    if (!normalized)
      add("series_quantiles", nullptr, probability_reason());
    else {
      auto sorted = samples;
      std::erase_if(sorted, [](const Sample &s) { return s.weight.n == 0; });
      std::uint64_t comparisons = 0;
      std::stable_sort(sorted.begin(), sorted.end(),
                       [&](const Sample &a, const Sample &b) {
                         if (++comparisons % 1024 == 0)
                           deadline(end);
                         return w::compare(a.value, b.value) < 0;
                       });
      Json values = Json::array();
      for (auto q : quantiles) {
        w::R cumulative{};
        auto value = sorted.front().value;
        for (const auto &s : sorted) {
          deadline(end);
          cumulative = w::plus(cumulative, s.weight);
          value = s.value;
          if (q.n == 0 || w::compare(cumulative, q) >= 0)
            break;
        }
        values.push_back({{"q", w::wire(q)}, {"value", w::wire(value)}});
      }
      add("series_quantiles",
          {{"rule",
            "inverse CDF on positive mass; q=0 minimum; first cumulative>=q"},
           {"values", values}});
    }
  }
  if (selected.contains("equity_drawdown")) {
    if (role != "equity" || samples.empty())
      add("equity_drawdown", nullptr,
          samples.empty() ? "no retained equity observations"
                          : "study requires caller-declared ordered equity");
    else {
      auto peak = samples.front().value, max_abs = w::R{};
      std::size_t peak_index = samples.front().index, max_peak = peak_index,
                  max_trough = peak_index;
      w::R peak_at_max = peak;
      std::optional<w::R> max_ratio;
      consumer::Rows curve(context, "drawdown-curve");
      for (const auto &s : samples) {
        deadline(end);
        if (w::compare(s.value, peak) > 0) {
          peak = s.value;
          peak_index = s.index;
        }
        auto dd = w::plus(peak, negative(s.value));
        if (w::compare(dd, max_abs) > 0) {
          max_abs = dd;
          max_peak = peak_index;
          max_trough = s.index;
          peak_at_max = peak;
        }
        Json ratio = missing(
            "peak is zero or outside selected relative-drawdown domain");
        if (peak.n != 0 &&
            (p.at("relative_drawdown") == "absolute_peak" || peak.n > 0)) {
          auto r = w::divide(dd, {w::absolute(peak.n), peak.d});
          ratio = available(r);
          if (!max_ratio || w::compare(r, *max_ratio) > 0)
            max_ratio = r;
        }
        curve.append({{"source_index", dec(s.index)},
                      {"equity", w::wire(s.value)},
                      {"peak", w::wire(peak)},
                      {"peak_source_index", dec(peak_index)},
                      {"absolute_drawdown", w::wire(dd)},
                      {"relative_drawdown", ratio},
                      {"underwater", dd.n > 0}});
      }
      Json recovery = missing("not recovered within retained series");
      for (const auto &s : samples)
        if (s.index >= max_trough && w::compare(s.value, peak_at_max) >= 0) {
          recovery = {{"status", "available"}, {"value", dec(s.index)}};
          break;
        }
      drawdown_study_index = studies.size();
      drawdown_curve = curve.close();
      add("equity_drawdown",
          {{"maximum_absolute_drawdown", w::wire(max_abs)},
           {"maximum_relative_drawdown",
            max_ratio ? available(*max_ratio)
                      : missing("no peak in selected relative domain")},
           {"maximum_absolute_peak_index", dec(max_peak)},
           {"maximum_absolute_trough_index", dec(max_trough)},
           {"maximum_absolute_recovery_index", recovery},
           {"curve", nullptr},
           {"definition",
            "ordered supplied valuations; strictly higher values update peak; "
            "first equal maximum drawdown retained; no weighting, cashflow "
            "adjustment or elapsed-time inference"}});
    }
  }
  if (selected.contains("return_ratios")) {
    std::string reason =
        role != "returns"
            ? "ratio study requires caller-declared per-period returns"
        : !normalized ? probability_reason()
        : numeric != "binary64_roundtrip"
            ? "square-root ratios require explicit binary64_roundtrip profile"
        : !variance ? variance_reason
                    : "";
    if (!reason.empty())
      add("return_ratios", nullptr, reason);
    else {
      static_assert(sizeof(double) == 8 &&
                    std::numeric_limits<double>::is_iec559);
      need(std::fegetround() == FE_TONEAREST,
           "binary64 profile requires nearest rounding");
      auto excess = w::plus(sum, negative(benchmark));
      w::R downside{};
      for (const auto &s : samples) {
        auto diff = w::plus(s.value, negative(benchmark));
        if (diff.n < 0)
          downside =
              w::plus(downside, w::times(s.weight, w::times(diff, diff)));
      }
      const auto scale = std::sqrt(real(periods)),
                 sigma = std::sqrt(real(*variance)),
                 down = std::sqrt(real(downside));
      add("return_ratios",
          {{"benchmark_per_period", w::wire(benchmark)},
           {"periods_per_year", w::wire(periods)},
           {"mean_differential_return", w::wire(excess)},
           {"variance_estimator", p.at("variance_estimator")},
           {"downside_second_moment", w::wire(downside)},
           {"annualized_volatility", approximate(sigma * scale)},
           {"sharpe", variance->n
                          ? approximate(real(excess) / sigma * scale)
                          : missing("zero differential-return variance")},
           {"sortino", downside.n ? approximate(real(excess) / down * scale)
                                  : missing("zero downside second moment")},
           {"definition",
            "constant selected benchmark; downside uses all probability mass; "
            "square-root time scaling is a caller convention, not a "
            "serial-independence or predictive claim"}});
    }
  }
  for (const auto &id : p.at("studies")) {
    const auto found = definitions.find(str(id));
    if (found == definitions.end())
      continue;
    const auto &definition = found->second;
    const auto calculated =
        formula::evaluate(definition.at("expression"), samples, nullptr, end);
    need(calculated.value || p.at("on_incompatible") != "reject",
         calculated.reason.c_str());
    studies.push_back(
        {{"study_id", id},
         {"version", definition.at("version")},
         {"status", calculated.value ? "available" : "unavailable"},
         {"reason", calculated.reason},
         {"descriptor", definition},
         {"data", calculated.value
                      ? Json{{"value", w::wire(*calculated.value)},
                             {"unit", definition.at("unit")},
                             {"numeric_profile", "checked_exact_rational_128"},
                             {"input_rows", dec(samples.size())},
                             {"weighting", measure},
                             {"authorship",
                              "caller-defined formula executed by native SBV"}}
                      : Json(nullptr)}});
  }
  consumer::Rows series(context, "series");
  for (const auto &s : samples)
    series.append({{"source_index", dec(s.index)},
                   {"value", w::wire(s.value)},
                   {"weight", w::wire(s.weight)}});
  auto result = base("selected source-series analysis");
  auto &sections = result["sections"];
  sections["summary"] = section({{"source_rows", dec(data.size())},
                                 {"retained_rows", dec(samples.size())},
                                 {"excluded_rows", dec(excluded.size())},
                                 {"unit", p.at("unit")},
                                 {"series_role", role},
                                 {"weighting", measure}});

  sections["studies"] =
      section(studies, selected.empty() ? "not_selected" : "available",
              selected.empty() ? "zero studies selected" : "");
  auto choices = p;
  choices.erase("output_path");
  choices.erase("output");
  sections["choices"] = section(choices);
  sections["resources"] = section(
      {{"backend", "cpu"},
       {"actual_workers", samples.empty() ? "0" : "1"},
       {"ordering", "stable source order; exact user-selected weighting"}});
  sections["provenance"] = section(
      {{"engine_version", version},
       {"source_result_sha256", source.at("content_sha256").materialize()},
       {"input_description", p.at("input_description")},
       {"input_claims",
        "caller attribution; no independent market/portfolio verification"},
       {"numeric_profile", numeric},
       {"provider_requests", "0"},
       {"additional_spend_usd", "0"}});
  sections["diagnostics"] = section(Json::array(
      {"Source series role, unit and ordering are caller selections, not "
       "inferred portfolio feasibility.",
       "Missing observations follow the selected policy; supplied input "
       "weights "
       "are retained. Normalization requires an explicit study convention.",
       "Exact intermediates use checked signed 128-bit rationals; "
       "unrepresentable arithmetic rejects before persistence.",
       "Binary64 estimates retain round-trip decimal and bit identity and "
       "require explicit selection; exact inputs/moments remain retained.",
       "Observed statistics, probability scenarios and predictive calibration "
       "remain distinct."}));
  sections["user_extensions"] = section(p.at("extensions"));
  auto out = logical::Value(sections)
                 .with("series", consumer::section(series.close()))
                 .with("exclusions", consumer::section(excluded.close()));
  if (drawdown_curve || binary_selected) {
    std::vector<logical::Value> entries;
    for (std::size_t i = 0; i < studies.size(); ++i) {
      auto entry = logical::Value(studies[i]);
      if (drawdown_curve && i == drawdown_study_index)
        entry =
            entry.with("data", entry.at("data").with("curve", *drawdown_curve));
      entries.push_back(std::move(entry));
    }
    if (binary_selected) {
      auto report = binary::calculate(
          p.at("binary"), selected, samples, data, measure,
          str(p.at("missing")), str(p.at("on_incompatible")), context, end);
      for (auto &entry : report.studies)
        entries.push_back(std::move(entry));
      out = out.with("binary_input", consumer::section(report.input));
    }
    out =
        out.with("studies",
                 consumer::section(logical::Value::array(std::move(entries))));
  }
  const auto parent = source.at("sections");
  for (const auto *key :
       {"signals", "execution", "replay", "book_frames", "book_checkpoint"})
    if (parent.contains(key))
      out = out.with(key, parent.at(key));
  out = out.with("source_context",
                 consumer::section(consumer::source_context(parent).with(
                     "pointer", logical::Value(p.at("pointer")))));
  if (context)
    out = out.with(
        "resources",
        out.at("resources")
            .with("data",
                  out.at("resources")
                      .at("data")
                      .with("memory_scope",
                            logical::Value(
                                "typed sample vector and optional sorted copy "
                                "are O(retained rows); output curves/rows "
                                "streamed; host allocation applies"))));
  return logical::Value(result).with("sections", std::move(out));
}
Json analyze(const Json &p, std::int64_t end) {
  need(p.contains("output") != p.contains("output_path"),
       "select exactly one analysis output");
  if (p.contains("output"))
    return partitioned_result(p, "analyze", end,
                              [&](PartitionedOutput &context) {
                                return analyze_body(p, &context, end);
                              });
  return persist(analyze_body(p, nullptr, end)
                     .materialize({artifact_bytes, artifact_values}),
                 p, "analyze", end);
}
} // namespace symphony::sbv::detail
