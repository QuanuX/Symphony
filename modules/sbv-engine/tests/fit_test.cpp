#include "../src/detail.hpp"
#include "../src/wide_rational.hpp"
#include <cstdlib>
#include <filesystem>
#include <iostream>
#include <random>
#include <source_location>
#include <unistd.h>
namespace s = symphony::sbv;
namespace d = s::detail;
namespace e = symphony::knowledge::engine;
namespace w = d::wide_rational;
using J = s::Json;
unsigned checks = 0;
void check(bool b, std::source_location l = std::source_location::current()) {
  ++checks;
  if (!b)
    throw std::runtime_error("fit assertion " + std::to_string(l.line()));
}
J ratio(long long n, long long d = 1) { return w::wire({n, d}); }
int main() try {
  char tmp[] = "/private/tmp/sbv-fit-XXXXXX";
  const auto *env = std::getenv("SYMPHONY_SBV_FIT_FIXTURES");
  const auto *created = env ? env : mkdtemp(tmp);
  check(created);
  std::filesystem::path root(created);
  std::filesystem::create_directories(root);
  unsigned seq = 0;
  auto path = [&](std::string n) { return (root / n).string(); };
  auto read = [&](std::string p) {
    auto r = e::parse_bounded_json(d::read_file(p, e::no_deadline),
                                   d::artifact_bytes, d::artifact_values);
    s::validate_result(r);
    return r;
  };
  auto source = [&](J rows) {
    auto r = d::base("mechanical regression observations");
    r["sections"]["observations"] = d::section(rows);
    r = s::seal_result(r);
    auto p = path("source-" + d::dec(++seq) + ".json");
    d::create_file(p, r.dump(), e::no_deadline);
    return J{{"path", p},
             {"expected_sha256", r["content_sha256"]},
             {"pointer", "/sections/observations/data"}};
  };
  auto feature = [](std::string id, std::string pointer) {
    return J{{"id", id},
             {"pointer", pointer},
             {"value_type", "integer"},
             {"unit", "x_unit"}};
  };
  J rows = J::array({{{"id", "a"}, {"x", "0"}, {"y", "1"}, {"w", ratio(1)}},
                     {{"id", "b"}, {"x", "1"}, {"y", "3"}, {"w", ratio(3)}},
                     {{"id", "c"}, {"x", "2"}, {"y", "5"}, {"w", ratio(2)}}});
  J p{{"protocol", "symphony.sbv.fit-input.v1"},
      {"source", source(rows)},
      {"output_path", ""},
      {"features", J::array({feature("x", "/x")})},
      {"target",
       {{"pointer", "/y"}, {"value_type", "integer"}, {"unit", "y_unit"}}},
      {"id_pointer", "/id"},
      {"identity_namespace", "fixture"},
      {"input_description", "known linear observations"},
      {"weights",
       {{"kind", "supplied"}, {"pointer", "/w"}, {"domain", "nonnegative"}}},
      {"intercept", true},
      {"regularization",
       {{"feature_penalties", J::array({ratio(0)})},
        {"intercept_penalty", ratio(0)}}},
      {"missing", "reject"},
      {"on_unsolved", "unavailable"},
      {"retain_training_predictions", true},
      {"extensions", {{"unknown", "\u001b"}}}};
  J last_request, last_receipt;
  auto call = [&](std::string op, J q) {
    q["output_path"] = path(op + "-" + d::dec(++seq) + ".json");
    last_request = q;
    last_receipt = s::dispatch(op, q, e::no_deadline);
    auto r = read(q["output_path"]);
    check(last_receipt["content_sha256"] == r["content_sha256"]);
    return r;
  };
  auto reject = [&](std::string op, J q, std::int64_t end = e::no_deadline) {
    q["output_path"] = path("refused-" + d::dec(++seq) + ".json");
    bool failed = false;
    try {
      s::dispatch(op, q, end);
    } catch (const std::exception &) {
      failed = true;
    }
    check(failed);
    check(!std::filesystem::exists(q["output_path"].get<std::string>()));
  };
  auto model = call("fit", p);
  auto model_ref = J{{"path", last_request["output_path"]},
                     {"expected_sha256", model["content_sha256"]},
                     {"pointer", "/sections/model/data"}};
  check(model["sections"]["model"]["data"]["coefficients"] ==
        J::array({ratio(1), ratio(2)}));
  check(model["sections"]["fit_diagnostics"]["data"]["objective_value"] ==
        ratio(0));
  check(model["sections"]["user_extensions"]["data"] == p["extensions"]);
  J pred{{"protocol", "symphony.sbv.predict-input.v1"},
         {"model", model_ref},
         {"source", p["source"]},
         {"output_path", ""},
         {"features", p["features"]},
         {"target", p["target"]},
         {"id_pointer", "/id"},
         {"identity_namespace", "fixture"},
         {"input_description", "explicit reuse for diagnostics"},
         {"weights", p["weights"]},
         {"missing", "reject"},
         {"purpose", "user declares test; not enforced untouched"},
         {"studies", J::array({"regression_errors"})},
         {"extensions", J::object()}};
  auto prediction = call("predict", pred);
  check(prediction["sections"]["training_overlap"]["data"]["reused_count"] ==
        "3");
  for (unsigned i = 0; i < 3; ++i)
    check(prediction["sections"]["predictions"]["data"][i]["prediction"] ==
          ratio(1 + 2 * i));
  auto q = pred;
  q["target"] = nullptr;
  auto infer = call("predict", q);
  check(infer["sections"]["studies"]["data"][0]["status"] == "unavailable");
  check(infer["sections"]["predictions"]["data"][0]["residual"].is_null());
  q["studies"] = J::array();
  q["identity_namespace"] = "other";
  check(
      call("predict", q)["sections"]["training_overlap"]["data"]["reused_count"]
          .is_null());
  // A closed-form 2x2 oracle is independent of pivoted elimination.
  std::mt19937 rng(4751);
  for (unsigned trial = 0; trial < 80; ++trial) {
    J samples = J::array();
    long long mass = 0, sx = 0, sy = 0, sxx = 0, sxy = 0;
    long long pi = rng() % 3, pf = rng() % 4;
    for (unsigned i = 0; i < 9; ++i) {
      long long x = static_cast<int>(rng() % 19) - 9,
                y = static_cast<int>(rng() % 31) - 15, weight = rng() % 5;
      samples.push_back({{"id", d::dec(i)},
                         {"x", d::dec(x)},
                         {"y", d::dec(y)},
                         {"w", ratio(weight)}});
      mass += weight;
      sx += weight * x;
      sy += weight * y;
      sxx += weight * x * x;
      sxy += weight * x * y;
    }
    const auto det = (mass + pi) * (sxx + pf) - sx * sx;
    check(det > 0);
    q = p;
    q["source"] = source(samples);
    q["regularization"] = {{"feature_penalties", J::array({ratio(pf)})},
                           {"intercept_penalty", ratio(pi)}};
    auto got = call("fit", q);
    auto coefficients = got["sections"]["model"]["data"]["coefficients"];
    check(coefficients == J::array({ratio(sy * (sxx + pf) - sx * sxy, det),
                                    ratio((mass + pi) * sxy - sx * sy, det)}));
    auto beta0 = w::artifact(coefficients[0]),
         beta1 = w::artifact(coefficients[1]);
    w::R sse{};
    for (const auto &row : samples) {
      auto residual =
          w::plus(w::plus(beta0, w::times(beta1, {w::integer(row["x"]), 1})),
                  {-w::integer(row["y"]), 1});
      sse = w::plus(
          sse, w::times(w::artifact(row["w"]), w::times(residual, residual)));
    }
    check(got["sections"]["fit_diagnostics"]["data"]
             ["weighted_squared_error_sum"] == w::wire(sse));
  }
  // Multiple columns and reordered prediction mappings preserve trained order.
  rows = J::array();
  for (int i = -3; i <= 3; ++i)
    for (int j = -2; j <= 2; ++j)
      rows.push_back({{"id", d::dec(i) + ":" + d::dec(j)},
                      {"x", d::dec(i)},
                      {"z", d::dec(j)},
                      {"y", d::dec(7 + 2 * i - 3 * j)}});
  q = p;
  q["weights"] = {
      {"kind", "uniform"}, {"pointer", nullptr}, {"domain", "nonnegative"}};
  q["features"].push_back(feature("z", "/z"));
  q["regularization"]["feature_penalties"].push_back(ratio(0));
  q["source"] = source(rows);
  auto mult = call("fit", q);
  check(mult["sections"]["model"]["data"]["coefficients"] ==
        J::array({ratio(7), ratio(2), ratio(-3)}));
  auto mp = pred;
  mp["source"] = q["source"];
  mp["weights"] = q["weights"];
  mp["features"] = J::array({feature("z", "/z"), feature("x", "/x")});
  mp["model"] = {{"path", last_request["output_path"]},
                 {"expected_sha256", mult["content_sha256"]},
                 {"pointer", "/sections/model/data"}};
  auto multi_predictions = call("predict", mp);
  for (unsigned i = 0; i < rows.size(); ++i)
    check(
        multi_predictions["sections"]["predictions"]["data"][i]["prediction"] ==
        ratio(d::i64(rows[i]["y"])));
  // Signed weights can produce a unique stationary solution with zero total
  // weight; this must not be normalized or represented as probability.
  q = p;
  rows = J::array({{{"id", "a"}, {"x", "0"}, {"y", "1"}, {"w", ratio(1)}},
                   {{"id", "b"}, {"x", "1"}, {"y", "3"}, {"w", ratio(-1)}}});
  q["source"] = source(rows);
  reject("fit", q);
  q["weights"]["domain"] = "signed";
  auto signed_fit = call("fit", q);
  check(signed_fit["sections"]["model"]["data"]["coefficients"] ==
        J::array({ratio(1), ratio(2)}));
  check(signed_fit["sections"]["fit_diagnostics"]["data"]["mean_squared_error"]
                  ["status"] == "unavailable");
  rows[0]["x"] = "1";
  q["source"] = source(rows);
  q["intercept"] = false;
  auto inconsistent = call("fit", q);
  check(inconsistent["sections"]["summary"]["data"]["system_consistent"] ==
        false);
  check(inconsistent["sections"]["model"]["status"] == "unavailable");
  q["on_unsolved"] = "reject";
  reject("fit", q);
  // Singular columns are not silently dropped or regularized.
  q = p;
  rows = J::array({{{"id", "a"}, {"x", "1"}, {"y", "1"}, {"w", ratio(1)}},
                   {{"id", "b"}, {"x", "1"}, {"y", "3"}, {"w", ratio(1)}}});
  q["source"] = source(rows);
  check(call("fit", q)["sections"]["model"]["status"] == "unavailable");
  q["regularization"]["feature_penalties"][0] = ratio(1);
  check(call("fit", q)["sections"]["model"]["data"]["coefficients"] ==
        J::array({ratio(2), ratio(0)}));
  q = p;
  q["source"] = source(J::array());
  check(call("fit", q)["sections"]["model"]["status"] == "unavailable");
  q["source"] = p["source"];
  q["features"] = J::array();
  q["regularization"]["feature_penalties"] = J::array();
  check(call("fit", q)["sections"]["model"]["data"]["coefficients"] ==
        J::array({ratio(10, 3)}));
  q["intercept"] = false;
  check(call("fit", q)["sections"]["model"]["data"]["coefficients"].empty());
  // Missing fields are selectable; present malformed data always rejects, even
  // in a row missing a different selected field.
  rows = J::array({{{"id", "a"}, {"x", "1"}, {"w", ratio(1)}}});
  q = p;
  q["source"] = source(rows);
  reject("fit", q);
  q["missing"] = "exclude";
  check(call("fit", q)["sections"]["exclusions"]["data"].size() == 1);
  rows[0]["x"] = "malformed";
  q["source"] = source(rows);
  reject("fit", q);
  q = p;
  q["regularization"]["feature_penalties"][0] = ratio(-1);
  reject("fit", q);
  q = p;
  q["features"].push_back(q["features"][0]);
  reject("fit", q);
  q = p;
  q["regularization"]["feature_penalties"] = J::array();
  reject("fit", q);
  q = pred;
  q["features"][0]["unit"] = "wrong";
  reject("predict", q);
  q = pred;
  q["features"][0]["id"] = "wrong";
  reject("predict", q);
  q = pred;
  q["target"]["unit"] = "wrong";
  reject("predict", q);
  q = pred;
  q["model"]["expected_sha256"] = std::string(64, '0');
  reject("predict", q);
  // Supplied models must satisfy the model contract, even when resealed.
  for (unsigned change = 0; change < 3; ++change) {
    auto bad = model;
    if (change == 0)
      bad["sections"]["model"]["data"]["coefficients"].push_back(ratio(0));
    if (change == 1)
      bad["sections"]["model"]["data"]["regularization"]["feature_penalties"]
         [0] = ratio(-1);
    if (change == 2)
      bad["sections"]["model"]["data"]["training"]["source_result_sha256"] =
          "bad";
    bad = s::seal_result(bad);
    auto file = path("bad-model-" + d::dec(change) + ".json");
    d::create_file(file, bad.dump(), e::no_deadline);
    q = pred;
    q["model"]["path"] = file;
    q["model"]["expected_sha256"] = bad["content_sha256"];
    reject("predict", q);
  }
  rows = J::array({{{"id", "a"},
                    {"x", "170141183460469231731687303715884105727"},
                    {"y", "1"},
                    {"w", ratio(1)}}});
  q = p;
  q["source"] = source(rows);
  reject("fit", q);
  reject("fit", p, e::unix_time_ms() - 1);
  reject("predict", pred, e::unix_time_ms() - 1);
  // Immutable output and stable installed fixtures.
  p["output_path"] = path("installed-fit.json");
  pred["output_path"] = path("installed-predict.json");
  d::create_file(path("fit-request.json"), p.dump(2), e::no_deadline);
  d::create_file(path("predict-request.json"), pred.dump(2), e::no_deadline);
  std::cout << checks << " fit/predict assertions passed\n";
  if (!env)
    std::filesystem::remove_all(root);
  return 0;
} catch (const std::exception &x) {
  std::cerr << x.what() << '\n';
  return 1;
}
