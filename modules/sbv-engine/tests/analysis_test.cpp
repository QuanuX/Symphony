#include "../src/wide_rational.hpp"
#include <bit>
#include <cmath>
#include <cstdlib>
#include <filesystem>
#include <fstream>
#include <iostream>
#include <source_location>
#include <unistd.h>
namespace s = symphony::sbv;
namespace d = s::detail;
namespace e = symphony::knowledge::engine;
using J = s::Json;
unsigned checks = 0;
void check(bool b, std::source_location l = std::source_location::current()) {
  ++checks;
  if (!b)
    throw std::runtime_error("analysis assertion " + std::to_string(l.line()));
}
J rat(const char *n, const char *d = "1") {
  return {{"numerator", n}, {"denominator", d}};
}
int main() try {
  char tmp[] = "/private/tmp/sbv-analysis-XXXXXX";
  const auto *dir = std::getenv("SYMPHONY_SBV_ANALYSIS_FIXTURES");
  std::filesystem::path root = dir ? dir : mkdtemp(tmp);
  std::filesystem::create_directories(root);
  unsigned seq = 0;
  auto save = [&](const J &v) {
    auto path = (root / ("source-" + std::to_string(++seq) + ".json")).string();
    d::create_file(path, v.dump(), e::unix_time_ms() + 30000);
    return path;
  };
  auto source = [&](J rows) {
    auto v = d::base("mechanical study fixture");
    v["sections"]["values"] = d::section(rows);
    v["sections"]["replay"] = d::section(J{{"fixture", "retained"}});
    return s::seal_result(v);
  };
  auto v = source(J::array({{{"value", rat("-1", "10")}},
                            {{"value", rat("0")}},
                            {{"value", rat("1", "10")}}}));
  J p{{"protocol", "symphony.sbv.analyze-input.v1"},
      {"path", save(v)},
      {"expected_sha256", v.at("content_sha256")},
      {"output_path", ""},
      {"pointer", "/sections/values/data"},
      {"value_pointer", "/value"},
      {"value_type", "rational"},
      {"weighting", "equal_probability"},
      {"weight_pointer", nullptr},
      {"series_role", "returns"},
      {"unit", "dimensionless"},
      {"input_description", "three mechanical equal-mass return observations"},
      {"missing", "reject"},
      {"studies", J::array({"series_summary", "series_moments",
                            "series_quantiles", "return_ratios"})},
      {"quantiles", J::array({rat("0"), rat("1", "2"), rat("1")})},
      {"variance_estimator", "sample_equal_weight"},
      {"relative_drawdown", "positive_peak_only"},
      {"benchmark", rat("-1", "100")},
      {"periods_per_year", rat("100")},
      {"numeric_profile", "binary64_roundtrip"},
      {"on_incompatible", "unavailable"},
      {"extensions", J::object()}};
  auto call = [&](const char *op, J q) {
    q["output_path"] =
        (root / ("result-" + std::to_string(++seq) + ".json")).string();
    auto receipt = s::dispatch(op, q, e::unix_time_ms() + 30000);
    check(receipt.at("path") == q.at("output_path"));
    auto result = e::parse_bounded_json(
        d::read_file(q.at("output_path"), e::unix_time_ms() + 30000),
        d::artifact_bytes, d::artifact_values);
    s::validate_result(result);
    check(result.at("content_sha256") == receipt.at("content_sha256"));
    return result;
  };
  auto reject = [&](const char *op, J q) {
    q["output_path"] =
        (root / ("refused-" + std::to_string(++seq) + ".json")).string();
    bool failed = false;
    try {
      s::dispatch(op, q, e::unix_time_ms() + 30000);
    } catch (const std::exception &) {
      failed = true;
    }
    check(failed);
    check(!std::filesystem::exists(q.at("output_path").get<std::string>()));
  };
  auto studies = [](const J &r) -> const J & {
    return r.at("sections").at("studies").at("data");
  };
  auto r = call("analyze", p);
  check(studies(r)[0]["data"]["weighted_sum"] == rat("0"));
  check(studies(r)[1]["data"]["mean"] == rat("0"));
  check(studies(r)[1]["data"]["variance"]["value"] == rat("1", "100"));
  check(studies(r)[2]["data"]["values"][1]["value"] == rat("0"));
  check(studies(r)[2]["data"]["values"][0]["value"] == rat("-1", "10"));
  check(studies(r)[2]["data"]["values"][2]["value"] == rat("1", "10"));
  const auto &ratio = studies(r)[3]["data"];
  const auto sharpe =
      std::stod(ratio["sharpe"]["value"]["decimal"].get<std::string>());
  check(std::abs(sharpe - 1.0) < 1e-14);
  check(std::bit_cast<std::uint64_t>(sharpe) ==
        std::stoull(ratio["sharpe"]["value"]["bits_hex"].get<std::string>(),
                    nullptr, 16));
  check(ratio["annualized_volatility"]["value"]["decimal"] == "1");
  check(ratio["downside_second_moment"] == rat("27", "10000"));
  check(std::abs(
            std::stod(ratio["sortino"]["value"]["decimal"].get<std::string>()) -
            1.9245008972987525) < 1e-14);
  check(r["sections"]["replay"] == v["sections"]["replay"]);
  auto q = p;
  q["variance_estimator"] = "population";
  r = call("analyze", q);
  check(studies(r)[1]["data"]["variance"]["value"] == rat("1", "150"));
  q = p;
  q["benchmark"] = rat("1", "100");
  check(std::abs(std::stod(studies(call(
                     "analyze", q))[3]["data"]["sharpe"]["value"]["decimal"]
                               .get<std::string>()) +
                 1.0) < 1e-14);
  q = p;
  q["numeric_profile"] = "exact_rational";
  check(studies(call("analyze", q))[3]["status"] == "unavailable");
  q["on_incompatible"] = "reject";
  reject("analyze", q);
  auto use = [&](J rows, J &req) {
    auto src = source(rows);
    req["path"] = save(src);
    req["expected_sha256"] = src.at("content_sha256");
  };
  q = p;
  use(J::array({{{"value", rat("1", "10")}}}), q);
  r = call("analyze", q);
  check(studies(r)[1]["data"]["variance"]["status"] == "unavailable");
  q["variance_estimator"] = "population";
  r = call("analyze", q);
  check(studies(r)[3]["data"]["sharpe"]["status"] == "unavailable");
  check(studies(r)[3]["data"]["sortino"]["status"] == "unavailable");
  q = p;
  use(J::array(), q);
  r = call("analyze", q);
  check(studies(r)[0]["data"]["count"] == "0");
  check(studies(r)[1]["status"] == "unavailable");
  check(studies(r)[2]["status"] == "unavailable");
  q = p;
  q["studies"] = J::array({"equity_drawdown"});
  q["quantiles"] = J::array();
  q["series_role"] = "equity";
  J equity = J::array();
  for (auto value : {"100", "120", "90", "110", "125"})
    equity.push_back({{"value", rat(value)}});
  use(equity, q);
  r = call("analyze", q);
  const auto &dd = studies(r)[0]["data"];
  check(dd["maximum_absolute_drawdown"] == rat("30"));
  check(dd["maximum_relative_drawdown"]["value"] == rat("1", "4"));
  check(dd["maximum_absolute_peak_index"] == "1");
  check(dd["maximum_absolute_trough_index"] == "2");
  check(dd["maximum_absolute_recovery_index"]["value"] == "4");
  check(dd["curve"][2]["underwater"] == true);
  q["series_role"] = "observations";
  check(studies(call("analyze", q))[0]["status"] == "unavailable");
  q["series_role"] = "equity";
  use(J::array({{{"value", rat("-10")}}, {{"value", rat("-20")}}}), q);
  check(studies(call("analyze",
                     q))[0]["data"]["maximum_relative_drawdown"]["status"] ==
        "unavailable");
  q["relative_drawdown"] = "absolute_peak";
  check(studies(call("analyze",
                     q))[0]["data"]["maximum_relative_drawdown"]["value"] ==
        rat("1"));
  use(J::array({{{"value", rat("0")}}, {{"value", rat("-20")}}}), q);
  check(studies(call("analyze",
                     q))[0]["data"]["maximum_relative_drawdown"]["status"] ==
        "unavailable");
  q = p;
  q["weighting"] = "supplied_probability";
  q["weight_pointer"] = "/weight";
  q["variance_estimator"] = "population";
  use(J::array({{{"value", rat("-10")}, {"weight", rat("0")}},
                {{"value", rat("2")}, {"weight", rat("1", "4")}},
                {{"value", rat("4")}, {"weight", rat("3", "4")}}}),
      q);
  r = call("analyze", q);
  check(studies(r)[1]["data"]["mean"] == rat("7", "2"));
  check(studies(r)[1]["data"]["variance"]["value"] == rat("3", "4"));
  check(studies(r)[2]["data"]["values"][0]["value"] == rat("2"));
  check(studies(r)[0]["data"]["minimum"]["value"] == rat("-10"));
  q["variance_estimator"] = "sample_equal_weight";
  check(studies(call("analyze", q))[1]["data"]["variance"]["status"] ==
        "unavailable");
  use(J::array({{{"value", rat("2")}, {"weight", rat("1", "4")}},
                {{"weight", rat("3", "4")}}}),
      q);
  reject("analyze", q);
  q["missing"] = "exclude";
  r = call("analyze", q);
  check(r["sections"]["exclusions"]["data"].size() == 1);
  check(studies(r)[1]["status"] == "unavailable");
  check(studies(r)[0]["data"]["weight_total"] == rat("1", "4"));
  q["weighting"] = "scenario_weight";
  check(studies(call("analyze", q))[1]["status"] == "unavailable");
  q["weighting"] = "signed_coefficient";
  use(J::array({{{"value", rat("2")}, {"weight", rat("-3")}}}), q);
  check(studies(call("analyze", q))[0]["data"]["weighted_sum"] == rat("-6"));
  q["weighting"] = "supplied_probability";
  reject("analyze", q);
  q = p;
  q["studies"] = J::array();
  q["quantiles"] = J::array();
  check(call("analyze", q)["sections"]["studies"]["status"] == "not_selected");
  for (unsigned i = 0; i < 10; ++i) {
    q = p;
    if (i == 0)
      q["expected_sha256"] = "wrong";
    if (i == 1)
      q["quantiles"] = J::array({rat("2")});
    if (i == 2)
      q["studies"].push_back("series_summary");
    if (i == 3)
      q["periods_per_year"] = rat("0");
    if (i == 4)
      q["weight_pointer"] = "/weight";
    if (i == 5)
      q["value_type"] = "float";
    if (i == 6)
      q["value_pointer"] = "bad";
    if (i == 7)
      q["pointer"] = "/sections";
    if (i == 8)
      use(J::array({{{"value", nullptr}}}), q);
    if (i == 9)
      use(J::array(
              {{{"value", rat("170141183460469231731687303715884105727")}},
               {{"value", rat("-170141183460469231731687303715884105727")}}}),
          q);
    reject("analyze", q);
  }
  q = p;
  q["studies"] = J::array({"series_quantiles"});
  q["quantiles"] = J::array({rat("1")});
  use(J::array({{{"value", rat("170141183460469231731687303715884105726",
                               "170141183460469231731687303715884105727")}},
                {{"value", rat("170141183460469231731687303715884105725",
                               "170141183460469231731687303715884105726")}}}),
      q);
  check(studies(
            call("analyze", q))[0]["data"]["values"][0]["value"]["numerator"] ==
        "170141183460469231731687303715884105726");
  auto candidate = [&](const char *id, const char *gain, const char *risk) {
    auto src = d::base("mechanical comparison");
    src["sections"]["metrics"] =
        d::section({{"gain", rat(gain)}, {"risk", rat(risk)}});
    src = s::seal_result(src);
    return J{{"id", id},
             {"state", "completed"},
             {"reason", ""},
             {"path", save(src)},
             {"expected_sha256", src.at("content_sha256")},
             {"parameters", {{"fixture", id}}},
             {"lineage", J::object()}};
  };
  J comp{{"protocol", "symphony.sbv.compare-input.v1"},
         {"output_path", ""},
         {"candidates", J::array({candidate("a", "10", "3"),
                                  candidate("b", "8", "2"),
                                  candidate("c", "7", "4"),
                                  candidate("tie", "10", "3"),
                                  {{"id", "failed"},
                                   {"state", "failed"},
                                   {"reason", "caller timeout"},
                                   {"path", nullptr},
                                   {"expected_sha256", nullptr},
                                   {"parameters", J::object()},
                                   {"lineage", J::object()}}})},
         {"objectives", J::array({{{"id", "gain"},
                                   {"pointer", "/sections/metrics/data/gain"},
                                   {"type", "rational"},
                                   {"direction", "maximize"},
                                   {"weight", rat("1")},
                                   {"scale", rat("1")},
                                   {"unit", "pnl"}},
                                  {{"id", "risk"},
                                   {"pointer", "/sections/metrics/data/risk"},
                                   {"type", "rational"},
                                   {"direction", "minimize"},
                                   {"weight", rat("1")},
                                   {"scale", rat("1")},
                                   {"unit", "risk"}}})},
         {"methods", J::array({"weighted_sum", "pareto"})},
         {"missing", "exclude_candidate"},
         {"comparison_description",
          "Mechanical tradeoff fixture, not strategy evidence"},
         {"extensions", J::object()}};
  r = call("compare", comp);
  auto cr = r["sections"]["comparisons"]["data"];
  check(cr["weighted_order"] == J::array({"a", "tie", "b", "c"}));
  check(cr["candidates"][1]["weighted_rank"] == "3");
  check(cr["candidates"][3]["weighted_rank"] == "1");
  check(cr["candidates"][4]["utility"]["status"] == "unavailable");
  check(cr["pareto_fronts"][0]["candidate_ids"] == J::array({"a", "b", "tie"}));
  check(cr["pareto_fronts"][1]["candidate_ids"] == J::array({"c"}));
  check(cr["candidates"][4]["state"] == "failed");
  check(r["sections"]["search"]["data"]["executed_trials"] == "0");
  q = comp;
  q["objectives"][0]["weight"] = rat("-1");
  r = call("compare", q);
  check(r["sections"]["comparisons"]["data"]["weighted_order"][0] == "b");
  check(r["sections"]["comparisons"]["data"]["pareto_fronts"] ==
        cr["pareto_fronts"]);
  q["objectives"][0]["weight"] = rat("0");
  check(call("compare",
             q)["sections"]["comparisons"]["data"]["weighted_order"][0] == "b");
  q = comp;
  q["objectives"][0]["pointer"] = "/missing";
  r = call("compare", q);
  check(r["sections"]["summary"]["data"]["eligible"] == "0");
  check(r["sections"]["comparisons"]["data"]["weighted_order"].empty());
  q["missing"] = "reject";
  reject("compare", q);
  q = comp;
  q["methods"] = J::array();
  check(call("compare", q)["sections"]["comparisons"]["data"]["weighted_order"]
            .empty());
  q["candidates"] = J::array();
  check(call("compare", q)["sections"]["summary"]["data"]["candidates"] == "0");
  for (unsigned i = 0; i < 9; ++i) {
    q = comp;
    if (i == 0)
      q["candidates"][1]["id"] = "a";
    if (i == 1)
      q["objectives"][0]["scale"] = rat("0");
    if (i == 2)
      q["candidates"][0]["expected_sha256"] = "bad";
    if (i == 3)
      q["candidates"][4]["path"] = "/tmp/unread";
    if (i == 4)
      q["objectives"][0]["pointer"] = "/sections/metrics/data";
    if (i == 5)
      q["methods"].push_back("pareto");
    if (i == 6)
      q["objectives"][1]["id"] = "gain";
    if (i == 7)
      q["candidates"][4]["reason"] = "";
    if (i == 8)
      q["candidates"][0]["state"] = "magic";
    reject("compare", q);
  }
  // Inputs are immutable; fixture emission allows installed process
  // verification.
  check(s::seal_result(v).at("content_sha256") == p.at("expected_sha256"));
  if (dir) {
    p["output_path"] = (root / "analysis-terminal.json").string();
    comp["output_path"] = (root / "comparison-terminal.json").string();
    d::create_file((root / "analyze-request.json").string(), p.dump(2),
                   e::unix_time_ms() + 30000);
    d::create_file((root / "compare-request.json").string(), comp.dump(2),
                   e::unix_time_ms() + 30000);
  } else
    std::filesystem::remove_all(root);
  std::cout << checks << " analysis/comparison assertions passed\n";
  return 0;
} catch (const std::exception &e) {
  std::cerr << e.what() << '\n';
  return 1;
}
