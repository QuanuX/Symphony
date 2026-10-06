#include "detail.hpp"
#include "wide_rational.hpp"
#include <algorithm>
#include <limits>
#include <map>
namespace symphony::sbv::detail {
namespace {
namespace w = wide_rational;
using R = w::R;
R minus(R a, R b) { return w::plus(a, {-b.n, b.d}); }
void text_field(const Json &v, std::size_t bound = 4096) {
  need(!str(v).empty() && str(v).size() <= bound,
       "nonempty bounded field required");
}
const Json *point(const Json &v, const std::string &p) {
  need(p.size() <= 4096, "model pointer bound");
  Json::json_pointer ptr(p);
  return v.contains(ptr) ? &v.at(ptr) : nullptr;
}
R number(const Json &v, const Json &type) {
  return type == "integer" ? R{w::integer(v), 1} : w::artifact(v);
}
void column(const Json &v, bool feature) {
  if (feature)
    keys(v, {"id", "pointer", "value_type", "unit"});
  else
    keys(v, {"pointer", "value_type", "unit"});
  if (feature)
    text_field(v.at("id"), 256);
  text_field(v.at("unit"), 128);
  (void)point(Json::object(), str(v.at("pointer")));
  need(v.at("value_type") == "integer" || v.at("value_type") == "rational",
       "unknown model numeric input type");
}
void features(const Json &v) {
  need(v.is_array(), "feature array required");
  std::set<std::string> ids;
  for (const auto &f : v) {
    column(f, true);
    need(ids.insert(str(f.at("id"))).second, "duplicate feature id");
  }
}
void model_metadata(const Json &m) {
  const auto &weights = m.at("weights");
  keys(weights, {"kind", "pointer", "domain"});
  need(weights.at("kind") == "uniform" || weights.at("kind") == "supplied",
       "unknown model weight kind");
  need(weights.at("domain") == "nonnegative" ||
           weights.at("domain") == "signed",
       "unknown model weight domain");
  if (weights.at("kind") == "uniform")
    need(weights.at("pointer").is_null(),
         "uniform model has no weight pointer");
  else
    (void)point(Json::object(), str(weights.at("pointer")));
  const auto &reg = m.at("regularization");
  keys(reg, {"feature_penalties", "intercept_penalty"});
  need(reg.at("feature_penalties").is_array() &&
           reg.at("feature_penalties").size() == m.at("features").size(),
       "model penalty shape mismatch");
  for (const auto &v : reg.at("feature_penalties"))
    need(w::artifact(v).n >= 0, "negative model penalty");
  const auto cp = w::artifact(reg.at("intercept_penalty"));
  need(cp.n >= 0 && (m.at("intercept").get<bool>() || cp.n == 0),
       "invalid model intercept penalty");
}
Json load(const Json &p, std::int64_t end) {
  keys(p, {"path", "expected_sha256", "pointer"});
  auto r = e::parse_bounded_json(read_file(str(p.at("path")), end),
                                 artifact_bytes, artifact_values);
  validate_result(r);
  need(r.at("content_sha256") == p.at("expected_sha256"),
       "model source digest mismatch");
  return r;
}
struct Sample {
  std::size_t index;
  std::string id;
  std::vector<R> x;
  R y, weight;
};
struct Data {
  Json source, excluded = Json::array();
  std::vector<Sample> samples;
  std::size_t input_rows = 0;
};
Data data(const Json &p, const Json &fs, const Json &target, std::int64_t end) {
  text_field(p.at("identity_namespace"));
  text_field(p.at("input_description"));
  need(p.at("missing") == "reject" || p.at("missing") == "exclude",
       "unknown missing policy");
  const auto &weight = p.at("weights");
  keys(weight, {"kind", "pointer", "domain"});
  const bool uniform = weight.at("kind") == "uniform";
  need(uniform || weight.at("kind") == "supplied", "unknown weighting kind");
  need(weight.at("domain") == "nonnegative" || weight.at("domain") == "signed",
       "unknown weight domain");
  need(uniform ? weight.at("pointer").is_null()
               : weight.at("pointer").is_string(),
       "weight pointer mismatch");
  auto ip = str(p.at("id_pointer"));
  (void)point(Json::object(), ip);
  const auto wp = uniform ? std::string{} : str(weight.at("pointer"));
  if (!uniform)
    (void)point(Json::object(), wp);
  Data out;
  out.source = load(p.at("source"), end);
  const auto *rows = point(out.source, str(p.at("source").at("pointer")));
  need(rows && rows->is_array(), "model source array required");
  out.input_rows = rows->size();
  std::set<std::string> ids;
  for (std::size_t i = 0; i < rows->size(); ++i) {
    deadline(end);
    const auto &row = (*rows)[i];
    const auto *id = point(row, ip);
    need(id != nullptr, "observation id missing");
    text_field(*id, 256);
    need(ids.insert(str(*id)).second, "duplicate observation id");
    Sample sample{i, str(*id), {}, {}, {1, 1}};
    bool missing = false;
    for (const auto &f : fs) {
      const auto *v = point(row, str(f.at("pointer")));
      missing |= !v;
      sample.x.push_back(v ? number(*v, f.at("value_type")) : R{});
    }
    if (!target.is_null()) {
      const auto *v = point(row, str(target.at("pointer")));
      missing |= !v;
      if (v)
        sample.y = number(*v, target.at("value_type"));
    }
    if (!uniform) {
      const auto *v = point(row, wp);
      missing |= !v;
      if (v)
        sample.weight = w::artifact(*v);
    }
    need(weight.at("domain") == "signed" || sample.weight.n >= 0,
         "negative weight requires signed domain");
    if (missing) {
      need(p.at("missing") != "reject",
           "feature, target or weight field absent");
      out.excluded.push_back(
          {{"source_index", dec(i)},
           {"id", sample.id},
           {"reason",
            "selected field absent; no imputation or weight normalization"}});
    } else
      out.samples.push_back(std::move(sample));
  }
  return out;
}
R predict_one(const Sample &s, const std::vector<R> &beta, bool intercept) {
  R value{};
  std::size_t j = 0;
  if (intercept)
    value = beta[j++];
  for (const auto &x : s.x)
    value = w::plus(value, w::times(beta[j++], x));
  return value;
}
Json loss(const std::vector<Sample> &samples, const std::vector<R> &beta,
          bool intercept, std::int64_t end) {
  R sum{}, abs{}, squares{}, mass{};
  for (const auto &s : samples) {
    deadline(end);
    const auto residual = minus(predict_one(s, beta, intercept), s.y);
    mass = w::plus(mass, s.weight);
    sum = w::plus(sum, w::times(s.weight, residual));
    abs =
        w::plus(abs, w::times(s.weight, {w::absolute(residual.n), residual.d}));
    squares =
        w::plus(squares, w::times(s.weight, w::times(residual, residual)));
  }
  auto mean = [&](R r) {
    return mass.n == 0 ? missing("zero algebraic weight total")
                       : Json{{"status", "available"},
                              {"value", w::wire(w::divide(r, mass))}};
  };
  return {{"count", dec(samples.size())},
          {"weight_total", w::wire(mass)},
          {"weighted_residual_sum", w::wire(sum)},
          {"weighted_absolute_error_sum", w::wire(abs)},
          {"weighted_squared_error_sum", w::wire(squares)},
          {"mean_error", mean(sum)},
          {"mean_absolute_error", mean(abs)},
          {"mean_squared_error", mean(squares)}};
}
Json predictions(const std::vector<Sample> &samples, const std::vector<R> &beta,
                 bool intercept, bool has_target, std::int64_t end) {
  Json out = Json::array();
  for (const auto &s : samples) {
    deadline(end);
    const auto y = predict_one(s, beta, intercept);
    out.push_back(
        {{"source_index", dec(s.index)},
         {"id", s.id},
         {"prediction", w::wire(y)},
         {"target", has_target ? w::wire(s.y) : Json(nullptr)},
         {"residual", has_target ? w::wire(minus(y, s.y)) : Json(nullptr)},
         {"weight", w::wire(s.weight)}});
  }
  return out;
}
Json choices(const Json &p) {
  auto c = p;
  c.erase("output_path");
  return c;
}
void common(Json &r, const Json &p, const Data &d) {
  auto &s = r["sections"];
  s["choices"] = section(choices(p));
  s["exclusions"] = section(d.excluded);
  s["user_extensions"] = section(p.at("extensions"));
  s["provenance"] =
      section({{"engine_version", version},
               {"numeric_profile", "checked_exact_rational_128"},
               {"source_result_sha256", d.source.at("content_sha256")},
               {"source_pointer", p.at("source").at("pointer")},
               {"identity_namespace", p.at("identity_namespace")},
               {"provider_requests", "0"},
               {"additional_spend_usd", "0"}});
  s["resources"] = section({{"backend", "cpu"}, {"actual_workers", "1"}});
  s["diagnostics"] = section(Json::array(
      {"Features, targets, units, observation identity and weights are caller "
       "declarations. Availability and causal lineage are not inferred.",
       "Signed weights describe algebraic stationary equations/losses, not "
       "probability or guaranteed minima. Weights are never normalized.",
       "Exact checked rational intermediates reject overflow without changing "
       "precision, imputing values or dropping columns.",
       "Artifact digests verify supplied content integrity, not model "
       "authorship or external causal claims."}));
}
} // namespace
Json fit(const Json &p, std::int64_t end) {
  keys(p, {"protocol", "source", "output_path", "features", "target",
           "id_pointer", "identity_namespace", "input_description", "weights",
           "intercept", "regularization", "missing", "on_unsolved",
           "retain_training_predictions", "extensions"});
  features(p.at("features"));
  column(p.at("target"), false);
  need(p.at("intercept").is_boolean() &&
           p.at("retain_training_predictions").is_boolean() &&
           p.at("extensions").is_object(),
       "fit selections required");
  need(p.at("on_unsolved") == "reject" || p.at("on_unsolved") == "unavailable",
       "unknown unsolved policy");
  const bool intercept = p.at("intercept").get<bool>();
  const auto dim = p.at("features").size() + std::size_t(intercept);
  need(dim < std::numeric_limits<std::size_t>::max() &&
           (dim == 0 ||
            dim + 1 <= std::numeric_limits<std::size_t>::max() / dim),
       "normal system exceeds host address space");
  const auto &reg = p.at("regularization");
  keys(reg, {"feature_penalties", "intercept_penalty"});
  need(reg.at("feature_penalties").is_array() &&
           reg.at("feature_penalties").size() == p.at("features").size(),
       "penalty/feature count mismatch");
  const auto constant_penalty = w::artifact(reg.at("intercept_penalty"));
  need(constant_penalty.n >= 0 && (intercept || constant_penalty.n == 0),
       "invalid intercept penalty");
  std::vector<R> penalties;
  if (intercept)
    penalties.push_back(constant_penalty);
  for (const auto &x : reg.at("feature_penalties")) {
    auto v = w::artifact(x);
    need(v.n >= 0, "negative quadratic penalty");
    penalties.push_back(v);
  }
  auto d = data(p, p.at("features"), p.at("target"), end);
  std::vector<std::vector<R>> system(dim, std::vector<R>(dim + 1));
  for (const auto &s : d.samples) {
    deadline(end);
    auto x = s.x;
    if (intercept)
      x.insert(x.begin(), {1, 1});
    for (std::size_t i = 0; i < dim; ++i) {
      deadline(end);
      auto wx = w::times(s.weight, x[i]);
      for (std::size_t j = 0; j < dim; ++j)
        system[i][j] = w::plus(system[i][j], w::times(wx, x[j]));
      system[i][dim] = w::plus(system[i][dim], w::times(wx, s.y));
    }
  }
  for (std::size_t i = 0; i < dim; ++i)
    system[i][i] = w::plus(system[i][i], penalties[i]);
  std::size_t rank = 0;
  std::vector<std::size_t> pivots;
  for (std::size_t col = 0; col < dim; ++col) {
    deadline(end);
    std::size_t pivot = rank;
    while (pivot < dim && system[pivot][col].n == 0)
      ++pivot;
    if (pivot == dim)
      continue;
    std::swap(system[rank], system[pivot]);
    const auto scale = system[rank][col];
    for (std::size_t j = col; j <= dim; ++j)
      system[rank][j] = w::divide(system[rank][j], scale);
    for (std::size_t i = 0; i < dim; ++i) {
      deadline(end);
      if (i == rank)
        continue;
      auto factor = system[i][col];
      if (factor.n == 0)
        continue;
      for (std::size_t j = col; j <= dim; ++j)
        system[i][j] = minus(system[i][j], w::times(factor, system[rank][j]));
    }
    pivots.push_back(col);
    ++rank;
  }
  bool consistent = true;
  for (std::size_t i = rank; i < dim; ++i)
    consistent &= system[i][dim].n == 0;
  const std::string reason =
      d.samples.empty() ? "no retained training observations"
      : !consistent     ? "normal equations have no stationary solution"
      : rank < dim      ? "normal equations have no unique solution"
                        : "";
  need(reason.empty() || p.at("on_unsolved") == "unavailable", reason.c_str());
  auto r = base("native exact linear-model fit");
  common(r, p, d);
  auto &s = r["sections"];
  s["summary"] =
      section({{"input_rows", dec(d.input_rows)},
               {"training_rows", dec(d.samples.size())},
               {"excluded_rows", dec(d.excluded.size())},
               {"parameters", dec(dim)},
               {"system_rank", dec(rank)},
               {"system_consistent", consistent},
               {"model_status", reason.empty() ? "available" : "unavailable"}});
  if (!reason.empty()) {
    s["model"] = section(nullptr, "unavailable", reason);
    s["training_predictions"] = section(nullptr, "unavailable", reason);
    return persist(std::move(r), p, "fit", end);
  }
  std::vector<R> beta(dim);
  for (std::size_t i = 0; i < dim; ++i)
    beta[pivots[i]] = system[i][dim];
  Json coefficients = Json::array(), ids = Json::array();
  for (auto v : beta)
    coefficients.push_back(w::wire(v));
  for (const auto &row : d.samples)
    ids.push_back(row.id);
  s["model"] =
      section({{"protocol", "symphony.sbv.linear-model.v1"},
               {"method", "penalized_linear_normal_equations_v1"},
               {"features", p.at("features")},
               {"target", p.at("target")},
               {"intercept", intercept},
               {"coefficients", coefficients},
               {"regularization", reg},
               {"weights", p.at("weights")},
               {"training",
                {{"source_result_sha256", d.source.at("content_sha256")},
                 {"source_pointer", p.at("source").at("pointer")},
                 {"identity_namespace", p.at("identity_namespace")},
                 {"observation_ids", ids}}}});
  auto errors = loss(d.samples, beta, intercept, end);
  R penalty{};
  for (std::size_t i = 0; i < dim; ++i)
    penalty =
        w::plus(penalty, w::times(penalties[i], w::times(beta[i], beta[i])));
  errors["quadratic_penalty"] = w::wire(penalty);
  errors["objective_value"] = w::wire(
      w::plus(w::artifact(errors.at("weighted_squared_error_sum")), penalty));
  s["fit_diagnostics"] = section(errors);
  s["training_predictions"] =
      p.at("retain_training_predictions").get<bool>()
          ? section(predictions(d.samples, beta, intercept, true, end))
          : section(nullptr, "not_selected",
                    "training predictions not selected");
  return persist(std::move(r), p, "fit", end);
}
Json predict(const Json &p, std::int64_t end) {
  keys(p, {"protocol", "model", "source", "output_path", "features", "target",
           "id_pointer", "identity_namespace", "input_description", "weights",
           "missing", "purpose", "studies", "extensions"});
  features(p.at("features"));
  if (!p.at("target").is_null())
    column(p.at("target"), false);
  text_field(p.at("purpose"));
  need(p.at("extensions").is_object(), "extensions object required");
  need(p.at("studies").is_array() && p.at("studies").size() <= 1,
       "prediction study selection required");
  for (const auto &x : p.at("studies"))
    need(x == "regression_errors", "unknown prediction study");
  auto model_source = load(p.at("model"), end);
  const auto *m = point(model_source, str(p.at("model").at("pointer")));
  need(m != nullptr, "model pointer missing");
  keys(*m, {"protocol", "method", "features", "target", "intercept",
            "coefficients", "regularization", "weights", "training"});
  need(m->at("protocol") == "symphony.sbv.linear-model.v1" &&
           m->at("method") == "penalized_linear_normal_equations_v1" &&
           m->at("intercept").is_boolean(),
       "unsupported linear model");
  features(m->at("features"));
  column(m->at("target"), false);
  model_metadata(*m);
  const auto &train = m->at("training");
  keys(train, {"source_result_sha256", "source_pointer", "identity_namespace",
               "observation_ids"});
  text_field(train.at("identity_namespace"));
  const auto digest = str(train.at("source_result_sha256"));
  need(digest.size() == 64 && std::all_of(digest.begin(), digest.end(),
                                          [](char c) {
                                            return (c >= '0' && c <= '9') ||
                                                   (c >= 'a' && c <= 'f');
                                          }),
       "invalid model training digest");
  (void)point(Json::object(), str(train.at("source_pointer")));
  need(train.at("observation_ids").is_array(), "training identities required");
  std::set<std::string> trained;
  for (const auto &id : train.at("observation_ids")) {
    text_field(id, 256);
    need(trained.insert(str(id)).second, "duplicate training identity");
  }
  const bool intercept = m->at("intercept").get<bool>();
  need(m->at("coefficients").is_array() &&
           m->at("coefficients").size() ==
               m->at("features").size() + std::size_t(intercept),
       "coefficient shape mismatch");
  std::vector<R> beta;
  for (const auto &v : m->at("coefficients"))
    beta.push_back(w::artifact(v));
  need(p.at("features").size() == m->at("features").size(),
       "prediction feature count mismatch");
  std::map<std::string, Json> mapping;
  for (const auto &f : p.at("features"))
    mapping.emplace(str(f.at("id")), f);
  Json fs = Json::array();
  for (const auto &f : m->at("features")) {
    auto it = mapping.find(str(f.at("id")));
    need(it != mapping.end() && it->second.at("unit") == f.at("unit"),
         "prediction feature id/unit mismatch");
    fs.push_back(it->second);
  }
  const bool target = !p.at("target").is_null();
  if (target)
    need(p.at("target").at("unit") == m->at("target").at("unit"),
         "prediction target unit mismatch");
  auto d = data(p, fs, p.at("target"), end);
  auto r = base("native linear-model prediction");
  common(r, p, d);
  auto &s = r["sections"];
  s["predictions"] =
      section(predictions(d.samples, beta, intercept, target, end));
  const bool comparable =
      train.at("identity_namespace") == p.at("identity_namespace");
  Json reused = Json::array();
  if (comparable)
    for (const auto &sample : d.samples)
      if (trained.contains(sample.id))
        reused.push_back(
            {{"source_index", dec(sample.index)}, {"id", sample.id}});
  s["training_overlap"] = section(
      {{"identity_comparable", comparable},
       {"reused_observations", comparable ? reused : Json(nullptr)},
       {"reused_count", comparable ? Json(dec(reused.size())) : Json(nullptr)},
       {"scope", "caller-defined identity namespace; not complete "
                 "feature/label overlap or cross-run access history"}});
  s["summary"] = section({{"input_rows", dec(d.input_rows)},
                          {"predicted_rows", dec(d.samples.size())},
                          {"excluded_rows", dec(d.excluded.size())},
                          {"target_supplied", target},
                          {"purpose", p.at("purpose")},
                          {"prediction_unit", m->at("target").at("unit")}});
  s["model_reference"] =
      section({{"result_sha256", model_source.at("content_sha256")},
               {"pointer", p.at("model").at("pointer")},
               {"training", train},
               {"resolved_feature_mapping", fs},
               {"authorship", "supplied artifact; fitting provenance not "
                              "authenticated or recomputed"}});
  if (!p.at("studies").empty())
    s["studies"] = section(
        Json::array({{{"study_id", "regression_errors"},
                      {"version", "1"},
                      {"status", target ? "available" : "unavailable"},
                      {"reason", target ? "" : "targets not selected"},
                      {"data", target ? loss(d.samples, beta, intercept, end)
                                      : Json(nullptr)}}}));
  s["diagnostics"]["data"].push_back(
      "Purpose is a user label, not a claim of untouched holdout data. Reused "
      "IDs are disclosed without refusing exploratory reuse; different "
      "namespaces make overlap unknown.");
  return persist(std::move(r), p, "predict", end);
}
} // namespace symphony::sbv::detail
