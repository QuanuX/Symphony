#pragma once
#include "consumer_source.hpp"
#include "study_formula.hpp"
#include <algorithm>
namespace symphony::sbv::detail::binary {
namespace w = wide_rational;
inline bool study(const std::string &id) {
  return id == "binary_confusion" || id == "binary_probability" ||
         id == "binary_ranking";
}
inline w::R minus(w::R a, w::R b) { return w::plus(a, {-b.n, b.d}); }
inline Json ratio(w::R a, w::R b) {
  return b.n ? Json{{"status", "available"},
                    {"value", w::wire(w::divide(a, b))}}
             : missing("zero denominator");
}
struct Observation {
  std::size_t index;
  w::R score, weight;
  bool positive;
};
struct Group {
  w::R score, positive, negative, before_positive, before_negative;
};
struct Result {
  std::vector<logical::Value> studies;
  logical::Value input;
};
inline Result calculate(const Json &p, const std::set<std::string> &selected,
                        const std::vector<formula::Sample> &samples,
                        const logical::Value &source,
                        const std::string &measure,
                        const std::string &missing_policy,
                        const std::string &incompatible,
                        PartitionedOutput *context, std::int64_t end) {
  keys(p, {"label_pointer", "label_type", "positive_label", "negative_label",
           "unknown_label", "weight_normalization", "thresholds", "bin_edges",
           "tie_credit", "retain_rows", "label_description"});
  const auto pointer = str(p.at("label_pointer"));
  need(pointer.size() <= 4096 && (pointer.empty() || pointer[0] == '/'),
       "bounded label pointer required");
  Json::json_pointer label_pointer(pointer);
  need(p.at("label_type") == "integer" || p.at("label_type") == "rational",
       "numeric binary label type required");
  const auto positive_label = w::parameter(p.at("positive_label")),
             negative_label = w::parameter(p.at("negative_label"));
  need(!w::equal(positive_label, negative_label),
       "distinct binary label values required");
  need(p.at("unknown_label") == "exclude" || p.at("unknown_label") == "reject",
       "unknown label policy required");
  need(p.at("weight_normalization") == "by_total" ||
           p.at("weight_normalization") == "require_unit_mass",
       "binary weight normalization required");
  need(p.at("retain_rows").is_boolean() &&
           !str(p.at("label_description")).empty() &&
           str(p.at("label_description")).size() <= 4096,
       "binary row retention and label description required");
  need(p.at("thresholds").is_array() &&
           (selected.contains("binary_confusion") ? !p.at("thresholds").empty()
                                                  : p.at("thresholds").empty()),
       "thresholds belong only to selected confusion study");
  std::vector<w::R> thresholds;
  for (const auto &t : p.at("thresholds")) {
    keys(t, {"value", "positive_when"});
    need(t.at("positive_when") == "ge" || t.at("positive_when") == "gt",
         "explicit threshold equality rule required");
    thresholds.push_back(w::parameter(t.at("value")));
  }
  need(p.at("bin_edges").is_array() && (selected.contains("binary_probability")
                                            ? p.at("bin_edges").size() >= 2
                                            : p.at("bin_edges").empty()),
       "bin edges belong only to selected probability study");
  std::vector<w::R> edges;
  for (const auto &v : p.at("bin_edges")) {
    const auto edge = w::parameter(v);
    need(edge.n >= 0 && edge.n <= edge.d &&
             (edges.empty() || w::compare(edges.back(), edge) < 0),
         "strictly increasing probability bin edges required");
    edges.push_back(edge);
  }
  if (!edges.empty())
    need(w::equal(edges.front(), {0, 1}) && w::equal(edges.back(), {1, 1}),
         "probability bins must cover [0,1]");
  w::R tie_credit{};
  if (selected.contains("binary_ranking")) {
    tie_credit = w::parameter(p.at("tie_credit"));
    need(tie_credit.n >= 0 && tie_credit.n <= tie_credit.d,
         "concordance tie credit outside [0,1]");
  } else
    need(p.at("tie_credit").is_null(),
         "tie credit belongs only to selected ranking study");
  consumer::Rows rows(context, "binary-rows"),
      excluded(context, "binary-exclusions");
  std::vector<Observation> observations;
  w::R total{}, positives{}, negatives{};
  bool probability_domain = true;
  for (const auto &sample : samples) {
    deadline(end);
    const auto row = source.at(sample.index).materialize();
    const bool present = row.contains(label_pointer);
    if (!present) {
      need(missing_policy != "reject", "binary label field absent");
      excluded.append({{"source_index", dec(sample.index)},
                       {"reason", "label field absent"}});
      continue;
    }
    const auto label = p.at("label_type") == "integer"
                           ? w::R{w::integer(row.at(label_pointer)), 1}
                           : w::artifact(row.at(label_pointer));
    const bool positive = w::equal(label, positive_label);
    if (!positive && !w::equal(label, negative_label)) {
      need(p.at("unknown_label") != "reject",
           "label outside selected binary mapping");
      excluded.append({{"source_index", dec(sample.index)},
                       {"reason", "label outside selected binary mapping"}});
      continue;
    }
    observations.push_back(
        {sample.index, sample.value, sample.weight, positive});
    total = w::plus(total, sample.weight);
    if (positive)
      positives = w::plus(positives, sample.weight);
    else
      negatives = w::plus(negatives, sample.weight);
    probability_domain &=
        sample.value.n >= 0 && sample.value.n <= sample.value.d;
    if (p.at("retain_rows") == true)
      rows.append({{"source_index", dec(sample.index)},
                   {"score", w::wire(sample.value)},
                   {"label", w::wire(label)},
                   {"positive", positive},
                   {"weight", w::wire(sample.weight)}});
  }
  auto input = logical::Value(Json{{"source_series_rows", dec(samples.size())},
                                   {"retained_rows", dec(observations.size())},
                                   {"excluded_rows", dec(excluded.size())},
                                   {"rows_retained", p.at("retain_rows")},
                                   {"weight_total", w::wire(total)},
                                   {"positive_weight", w::wire(positives)},
                                   {"negative_weight", w::wire(negatives)},
                                   {"choices", p},
                                   {"prior_selection_history", "unknown"}})
                   .with("rows", rows.close())
                   .with("exclusions", excluded.close());
  Result result{{}, input};
  const std::string reason =
      observations.empty() ? "no retained binary observations"
      : measure == "signed_coefficient"
          ? "binary metrics require a nonnegative weighting domain"
      : total.n <= 0 ? "positive total weight required"
      : p.at("weight_normalization") == "require_unit_mass" &&
              !w::equal(total, {1, 1})
          ? "retained binary weight is not unit mass"
          : "";
  auto add = [&](const std::string &id, logical::Value data,
                 const std::string &why = "") {
    need(why.empty() || incompatible != "reject", why.c_str());
    Json descriptor{{"numeric_profile", "checked_exact_rational_128"},
                    {"weight_normalization", p.at("weight_normalization")},
                    {"label_description", p.at("label_description")},
                    {"label_mapping",
                     {{"positive", p.at("positive_label")},
                      {"negative", p.at("negative_label")}}}};
    Json entry{{"study_id", id},
               {"version", "1"},
               {"status", why.empty() ? "available" : "unavailable"},
               {"reason", why},
               {"descriptor", descriptor}};
    result.studies.push_back(logical::Value(entry).with(
        "data", why.empty() ? data : logical::Value(Json(nullptr))));
  };
  if (!reason.empty()) {
    for (const auto *id :
         {"binary_confusion", "binary_probability", "binary_ranking"})
      if (selected.contains(id))
        add(id, logical::Value(Json(nullptr)), reason);
    return result;
  }
  std::vector<Group> groups;
  if (selected.contains("binary_confusion") ||
      selected.contains("binary_ranking")) {
    auto sorted = observations;
    std::stable_sort(sorted.begin(), sorted.end(),
                     [&](const auto &a, const auto &b) {
                       deadline(end);
                       return w::compare(a.score, b.score) < 0;
                     });
    for (const auto &o : sorted) {
      deadline(end);
      if (groups.empty() || !w::equal(groups.back().score, o.score))
        groups.push_back({o.score, {}, {}, {}, {}});
      auto &mass = o.positive ? groups.back().positive : groups.back().negative;
      mass = w::plus(mass, o.weight);
    }
    w::R pos{}, neg{};
    for (auto &g : groups) {
      deadline(end);
      g.before_positive = pos;
      g.before_negative = neg;
      pos = w::plus(pos, g.positive);
      neg = w::plus(neg, g.negative);
    }
  }
  if (selected.contains("binary_confusion")) {
    consumer::Rows table(context, "binary-thresholds");
    for (std::size_t i = 0; i < thresholds.size(); ++i) {
      deadline(end);
      const auto ge = p.at("thresholds")[i].at("positive_when") == "ge";
      const auto it =
          std::lower_bound(groups.begin(), groups.end(), thresholds[i],
                           [&](const auto &g, const auto &v) {
                             return ge ? w::compare(g.score, v) < 0
                                       : w::compare(g.score, v) <= 0;
                           });
      const auto fn = it == groups.end() ? positives : it->before_positive,
                 tn = it == groups.end() ? negatives : it->before_negative;
      const auto tp = minus(positives, fn), fp = minus(negatives, tn);
      table.append(
          {{"threshold", p.at("thresholds")[i]},
           {"true_positive", w::wire(tp)},
           {"false_positive", w::wire(fp)},
           {"true_negative", w::wire(tn)},
           {"false_negative", w::wire(fn)},
           {"precision", ratio(tp, w::plus(tp, fp))},
           {"recall", ratio(tp, positives)},
           {"specificity", ratio(tn, negatives)},
           {"false_positive_rate", ratio(fp, negatives)},
           {"accuracy", ratio(w::plus(tp, tn), total)},
           {"f1", ratio(w::times({2, 1}, tp),
                        w::plus(w::times({2, 1}, tp), w::plus(fp, fn)))},
           {"balanced_accuracy",
            positives.n && negatives.n
                ? ratio(w::plus(w::divide(tp, positives),
                                w::divide(tn, negatives)),
                        {2, 1})
                : missing("both class weights must be positive")}});
    }
    add("binary_confusion",
        logical::Value(
            Json{{"weight_total", w::wire(total)},
                 {"definition",
                  "weighted confusion masses; ratios use explicit empirical "
                  "denominators; larger scores indicate positive class"}})
            .with("thresholds", table.close()));
  }
  if (selected.contains("binary_probability")) {
    if (!probability_domain)
      add("binary_probability", logical::Value(Json(nullptr)),
          "probability study requires every retained score in [0,1]; no "
          "clipping");
    else {
      struct Bin {
        std::size_t count = 0;
        w::R mass, prediction, positive;
      };
      std::vector<Bin> bins(edges.size() - 1);
      w::R error{};
      for (const auto &o : observations) {
        deadline(end);
        auto pos = std::upper_bound(
                       edges.begin(), edges.end(), o.score,
                       [](auto a, auto b) { return w::compare(a, b) < 0; }) -
                   edges.begin() - 1;
        auto &b =
            bins.at(std::min(static_cast<std::size_t>(pos), bins.size() - 1));
        ++b.count;
        b.mass = w::plus(b.mass, o.weight);
        b.prediction = w::plus(b.prediction, w::times(o.weight, o.score));
        if (o.positive)
          b.positive = w::plus(b.positive, o.weight);
        auto delta = minus(o.score, {o.positive ? 1 : 0, 1});
        error = w::plus(error, w::times(o.weight, w::times(delta, delta)));
      }
      consumer::Rows table(context, "binary-reliability");
      w::R gap_sum{}, max_gap{};
      for (std::size_t i = 0; i < bins.size(); ++i) {
        deadline(end);
        const auto &b = bins[i];
        w::R gap{};
        if (b.mass.n) {
          gap = w::divide(minus(b.prediction, b.positive), b.mass);
          gap.n = w::absolute(gap.n);
          gap_sum = w::plus(gap_sum, w::times(b.mass, gap));
          if (w::compare(gap, max_gap) > 0)
            max_gap = gap;
        }
        table.append(
            {{"lower", w::wire(edges[i])},
             {"upper", w::wire(edges[i + 1])},
             {"upper_inclusive", i + 1 == bins.size()},
             {"rows", dec(b.count)},
             {"weight", w::wire(b.mass)},
             {"mean_prediction", ratio(b.prediction, b.mass)},
             {"positive_rate", ratio(b.positive, b.mass)},
             {"absolute_gap",
              b.mass.n ? ratio(gap, {1, 1}) : missing("empty weighted bin")}});
      }
      add("binary_probability",
          logical::Value(
              Json{{"brier_score", w::wire(w::divide(error, total))},
                   {"weighted_squared_error_sum", w::wire(error)},
                   {"weight_total", w::wire(total)},
                   {"expected_calibration_error",
                    w::wire(w::divide(gap_sum, total))},
                   {"maximum_calibration_error", w::wire(max_gap)},
                   {"definition",
                    "binary Brier mean squared error; empirical bin gaps "
                    "weighted by bin mass; left closed/right open bins except "
                    "final upper endpoint; no predictive calibration claim"}})
              .with("bins", table.close()));
    }
  }
  if (selected.contains("binary_ranking")) {
    w::R strict_wins{}, ties{}, ap{}, tp{}, fp{};
    for (const auto &g : groups) {
      deadline(end);
      strict_wins =
          w::plus(strict_wins, w::times(g.positive, g.before_negative));
      ties = w::plus(ties, w::times(g.positive, g.negative));
    }
    consumer::Rows curve(context, "binary-ranking-curve");
    curve.append({{"threshold", nullptr},
                  {"positive_when", "none"},
                  {"true_positive", w::wire(tp)},
                  {"false_positive", w::wire(fp)},
                  {"recall", ratio(tp, positives)},
                  {"false_positive_rate", ratio(fp, negatives)},
                  {"precision", missing("no predicted positive weight")}});
    for (auto it = groups.rbegin(); it != groups.rend(); ++it) {
      deadline(end);
      if (it->positive.n == 0 && it->negative.n == 0)
        continue;
      tp = w::plus(tp, it->positive);
      fp = w::plus(fp, it->negative);
      if (positives.n)
        ap = w::plus(ap, w::times(w::divide(it->positive, positives),
                                  w::divide(tp, w::plus(tp, fp))));
      curve.append({{"threshold", w::wire(it->score)},
                    {"positive_when", "ge"},
                    {"true_positive", w::wire(tp)},
                    {"false_positive", w::wire(fp)},
                    {"recall", ratio(tp, positives)},
                    {"false_positive_rate", ratio(fp, negatives)},
                    {"precision", ratio(tp, w::plus(tp, fp))}});
    }
    const auto pairs = w::times(positives, negatives);
    add("binary_ranking",
        logical::Value(
            Json{{"roc_auc",
                  ratio(w::plus(strict_wins, w::times({1, 2}, ties)), pairs)},
                 {"pairwise_concordance",
                  ratio(w::plus(strict_wins, w::times(tie_credit, ties)),
                        pairs)},
                 {"tie_credit", w::wire(tie_credit)},
                 {"tied_pair_weight", w::wire(ties)},
                 {"positive_negative_pair_weight", w::wire(pairs)},
                 {"average_precision",
                  positives.n ? ratio(ap, {1, 1})
                              : missing("no positive label weight")},
                 {"definition",
                  "descending exact score groups; ROC AUC uses one-half "
                  "credit for ties; concordance uses caller tie credit; "
                  "average precision is sum(recall increment * precision), "
                  "without PR trapezoid interpolation"}})
            .with("curve", curve.close()));
  }
  return result;
}
} // namespace symphony::sbv::detail::binary
