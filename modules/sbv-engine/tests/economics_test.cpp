#include "../src/detail.hpp"
#include <cstdlib>
#include <filesystem>
#include <fstream>
#include <iostream>
#include <source_location>
#include <symphony/sbv/models.hpp>
#include <unistd.h>
namespace e = symphony::knowledge::engine;
namespace s = symphony::sbv;
namespace d = s::detail;
using J = s::Json;
unsigned checks = 0;
void check(bool ok, std::source_location at = std::source_location::current()) {
  ++checks;
  if (!ok)
    throw std::runtime_error("economics assertion line " +
                             std::to_string(at.line()));
}
J rat(const char *n, const char *den = "1") {
  return {{"numerator", n}, {"denominator", den}};
}
J read(const std::string &p) {
  J j;
  std::ifstream f(p);
  f >> j;
  return j;
}
void write(const std::string &p, const J &j) {
  std::ofstream f(p);
  f << j.dump();
}
J call(const char *op, const J &j) {
  return s::dispatch(op, j, e::unix_time_ms() + 30000);
}
int main() try {
  char temp[] = "/private/tmp/sbv-economics-XXXXXX";
  check(::mkdtemp(temp) != nullptr);
  const std::string root = temp;
  struct Cleanup {
    std::string root;
    ~Cleanup() { std::filesystem::remove_all(root); }
  } cleanup{root};
  J signal{{"signal_id", "signal/0~"},
           {"source_ordinal", "0"},
           {"available_ns", "1000"},
           {"anchor_price_nanos", "100000000000"},
           {"causal_end_ordinal_exclusive", "1"},
           {"context_reference", "user://context"}};
  J producer{{"id", "fixture"},
             {"version", "1"},
             {"artifact_sha256", ""},
             {"reproducibility", "uncaptured"}};
  J census{{"protocol", "symphony.sbv.external-census.v1"},
           {"source_sha256", std::string(64, 'a')},
           {"mode", "causal_declared"},
           {"producer", producer},
           {"signals", J::array({signal})}};
  J support_points = J::array();
  support_points.push_back(
      {{"price_nanos", "99000000000"}, {"weight", rat("1", "2")}});
  support_points.push_back(
      {{"price_nanos", "101000000000"}, {"weight", rat("1", "2")}});
  J supplied{{"signal_id", "signal/0~"},
             {"evidence_reference", "synthetic arithmetic fixture"},
             {"execution_probability",
              {{"status", "supplied"}, {"value", rat("3", "5")}}},
             {"support", support_points}};
  J parameters{{"producer", producer},
               {"measure", "probability"},
               {"conditioning", "execution"},
               {"calibration_reference", ""},
               {"outcomes", J::array({supplied})}};
  J selection{{"protocol", "symphony.sbv.model-selection.v1"},
              {"id", "external_outcomes"},
              {"version", "1"},
              {"horizon_ns", "10"},
              {"parameters", parameters}};
  s::AdmittedModel model(selection, {"signal/0~"});
  s::ModelFrame frame{"signal/0~", 0, 1000, 1010, 100000000000, true, true, {}};
  auto outcome = model.evaluate(frame, e::unix_time_ms() + 30000);
  auto source =
      d::base("synthetic model arithmetic fixture; no market evidence");
  source["sections"]["signals"] = d::section(census.at("signals"));
  source["sections"]["execution"] = d::section(J::array({outcome}));
  source["sections"]["choices"] = d::section({{"census", census}});
  source["sections"]["summary"] =
      d::section({{"census_sha256", e::sha256_hex(census.dump())}});
  source["sections"]["replay"] =
      d::section({{"synthetic", true},
                  {"events", J::array({{{"ts_recv", "9007199254740993"}}})}});
  auto bind = [&](J src) {
    src = s::seal_result(src);
    write(root + "/source.json", src);
    return src.at("content_sha256");
  };
  J transform{{"id", "linear_price_pnl"},
              {"version", "1"},
              {"reference_price_nanos", "100000000000"},
              {"price_unit_nanos", "1000000000"},
              {"position_units", rat("2")},
              {"value_per_price_unit", rat("5")},
              {"cost_per_outcome", rat("1")},
              {"return_basis", rat("100")},
              {"pnl_unit", "test-unit"}};
  J studies = J::array();
  studies.push_back({{"id", "weighted_return_sum"},
                     {"version", "1"},
                     {"parameters", J::object()}});
  studies.push_back({{"id", "return_moments"},
                     {"version", "1"},
                     {"parameters", J::object()}});
  J levels = J::array({rat("0"), rat("3", "10"), rat("1", "2"), rat("1")});
  studies.push_back({{"id", "return_quantiles"},
                     {"version", "1"},
                     {"parameters", {{"levels", levels}}}});
  J p{{"protocol", "symphony.sbv.economics-input.v1"},
      {"path", root + "/source.json"},
      {"expected_sha256", bind(source)},
      {"output_path", root + "/initial.json"},
      {"selections", J::array({{{"signal_id", "signal/0~"},
                                {"transform", transform},
                                {"mode", "execution_mixture"},
                                {"nonexecution_pnl", rat("-2")}}})},
      {"studies", studies},
      {"on_incompatible", "unavailable"},
      {"extensions", {{"private-study", "\x1b[31m user-defined"}}}};
  unsigned count = 0;
  auto run = [&](J request) {
    request["output_path"] = root + "/r" + std::to_string(++count) + ".json";
    auto receipt = call("economics", request);
    auto result = read(request.at("output_path").get<std::string>());
    check(result.at("content_sha256") == receipt.at("content_sha256"));
    return result;
  };
  auto rejects = [&](J request) {
    request["output_path"] = root + "/bad" + std::to_string(++count) + ".json";
    bool failed = false;
    try {
      call("economics", request);
    } catch (const std::exception &) {
      failed = true;
    }
    check(failed && !std::filesystem::exists(
                        request.at("output_path").get<std::string>()));
  };
  auto result = run(p);
  const auto &row = result.at("sections").at("economics").at("data")[0];
  check(row.at("transformed_support")[0].at("gross_pnl") == rat("-10"));
  check(row.at("transformed_support")[0].at("net_pnl") == rat("-11"));
  check(row.at("transformed_support")[1].at("net_pnl") == rat("9"));
  check(row.at("atoms")[0].at("weight") == rat("3", "10"));
  check(row.at("atoms")[1].at("weight") == rat("3", "10"));
  check(row.at("atoms")[2].at("weight") == rat("2", "5"));
  check(row.at("atoms")[2].at("return") == rat("-1", "50"));
  const auto &measures = result.at("sections").at("studies").at("data");
  check(measures[0].at("data").at("weighted_sum").at("value") ==
        rat("-7", "500"));
  check(measures[1].at("data").at("mean").at("value") == rat("-7", "500"));
  check(measures[1].at("data").at("variance").at("value") ==
        rat("753", "125000"));
  check(measures[2].at("data").at("quantiles")[0].at("return") ==
        rat("-11", "100"));
  check(measures[2].at("data").at("quantiles")[1].at("return") ==
        rat("-11", "100"));
  check(measures[2].at("data").at("quantiles")[2].at("return") ==
        rat("-1", "50"));
  check(measures[2].at("data").at("quantiles")[3].at("return") ==
        rat("9", "100"));
  check(result.at("sections").at("replay") ==
        source.at("sections").at("replay"));
  check(result.at("sections").at("source_context").at("data").at("census") ==
        census);
  auto support = p;
  support["selections"][0]["mode"] = "support_only";
  support["selections"][0]["nonexecution_pnl"] = nullptr;
  result = run(support);
  check(result["sections"]["economics"]["data"][0]["atoms"].size() == 2);
  check(result["sections"]["studies"]["data"][1]["data"]["mean"]["value"] ==
        rat("-1", "100"));
  auto q = support;
  q["selections"][0]["transform"]["position_units"] = rat("-2");
  result = run(q);
  check(result["sections"]["economics"]["data"][0]["atoms"][0]["return"] ==
        rat("9", "100"));
  q["selections"][0]["transform"]["return_basis"] = rat("-1");
  result = run(q);
  check(result["sections"]["economics"]["data"][0]["atoms"][0]["return"] ==
        rat("-9"));
  q = support;
  q["selections"][0]["transform"]["cost_per_outcome"] = rat("-1");
  result = run(q);
  check(result["sections"]["economics"]["data"][0]["transformed_support"][1]
              ["net_pnl"] == rat("11"));
  q = support;
  q["studies"] = J::array();
  result = run(q);
  check(result["sections"]["studies"]["data"].empty());
  q["selections"] = J::array();
  result = run(q);
  check(result["sections"]["summary"]["data"]["selected_signal_count"] == "0");
  for (const auto *field :
       {"position_units", "value_per_price_unit", "cost_per_outcome"}) {
    q = support;
    q["selections"][0]["transform"][field] = rat("0");
    result = run(q);
    check(result["status"] == "completed");
  }
  q = p;
  q["expected_sha256"] = std::string(64, 'b');
  rejects(q);
  q = p;
  q["selections"][0]["signal_id"] = "unknown";
  rejects(q);
  q = p;
  q["selections"].push_back(q["selections"][0]);
  rejects(q);
  q = p;
  q["selections"][0]["transform"]["return_basis"] = rat("0");
  rejects(q);
  q = p;
  q["selections"][0]["transform"]["price_unit_nanos"] = "0";
  rejects(q);
  q = p;
  q["selections"][0]["transform"]["reference_price_nanos"] =
      "9223372036854775807";
  rejects(q);
  q = p;
  q["selections"][0]["transform"]["reference_price_nanos"] =
      "-9223372036854775808";
  rejects(q);
  q = p;
  q["selections"][0]["transform"]["position_units"] = rat("1000000000");
  q["selections"][0]["transform"]["value_per_price_unit"] = rat("1000000000");
  q["selections"][0]["transform"]["price_unit_nanos"] = "1";
  rejects(q);
  q = p;
  q["studies"][2]["parameters"]["levels"] = J::array({rat("2")});
  rejects(q);
  q = p;
  q["studies"].push_back(q["studies"][0]);
  rejects(q);
  q = p;
  q["extra"] = true;
  rejects(q);
  auto mutate = [&](auto edit) {
    auto src = source;
    edit(src["sections"]["execution"]["data"][0]);
    q = p;
    q["expected_sha256"] = bind(src);
  };
  mutate([&](J &m) { m["no_fill_probability"]["value"] = rat("1", "2"); });
  rejects(q);
  mutate([&](J &m) { m["support"][0]["weight"] = rat("-1", "2"); });
  rejects(q);
  mutate([&](J &m) { m["support"][1]["price_nanos"] = "99000000000"; });
  rejects(q);
  mutate([&](J &m) { m["price_scale"] = "1e-2"; });
  rejects(q);
  mutate([&](J &m) { m["fill_probability"] = d::missing("not estimated"); });
  result = run(q);
  check(result["status"] == "partial");
  check(result["sections"]["economics"]["data"][0]["transformed_support"]
            .size() == 2);
  check(result["sections"]["economics"]["data"][0]["atoms"].empty());
  q["on_incompatible"] = "reject";
  rejects(q);
  mutate([&](J &m) {
    m["status"] = "unavailable";
    m["reason"] = "thin support";
    m["support"] = J::array();
  });
  result = run(q);
  check(
      result["sections"]["summary"]["data"]["unavailable_economic_outcomes"] ==
      "1");
  mutate([&](J &m) {
    m["measure"] = "signed_coefficient";
    m["support"][0]["weight"] = rat("-1", "2");
  });
  q["selections"] = support["selections"];
  result = run(q);
  check(result["sections"]["economics"]["data"][0]["status"] == "available");
  check(result["sections"]["studies"]["data"][0]["data"]["weighted_sum"]
              ["value"] == rat("1", "10"));
  check(result["sections"]["studies"]["data"][1]["status"] == "unavailable");
  check(result["sections"]["studies"]["data"][2]["status"] == "unavailable");
  q["on_incompatible"] = "reject";
  rejects(q);
  mutate([&](J &m) {
    m["measure"] = "scenario_weight";
    m["support"][0]["weight"] = rat("2");
    m["support"][1]["weight"] = rat("3");
  });
  q["selections"] = support["selections"];
  q["studies"] = J::array({studies[0]});
  result = run(q);
  check(result["sections"]["studies"]["data"][0]["data"]["weighted_sum"]
              ["value"] == rat("1", "20"));
  mutate([&](J &m) {
    m["fill_probability"]["value"] = rat("0");
    m["no_fill_probability"]["value"] = rat("1");
  });
  result = run(q);
  check(result["sections"]["studies"]["data"][2]["data"]["quantiles"][0]
              ["return"] == rat("-1", "50"));
  check(result["sections"]["studies"]["data"][1]["data"]["variance"]["value"] ==
        rat("0"));
  mutate([&](J &m) {
    m["fill_probability"]["value"] = rat("1");
    m["no_fill_probability"]["value"] = rat("0");
  });
  result = run(q);
  check(result["sections"]["studies"]["data"][2]["data"]["quantiles"][3]
              ["return"] == rat("9", "100"));
  // Ordering must distinguish adjacent integers above 2^53, in reversed input
  // order.
  mutate([&](J &m) {
    m["support"][0]["price_nanos"] = "9007199254740993";
    m["support"][1]["price_nanos"] = "9007199254740992";
  });
  q["selections"] = support["selections"];
  q["selections"][0]["transform"]["reference_price_nanos"] = "0";
  q["selections"][0]["transform"]["price_unit_nanos"] = "1";
  q["selections"][0]["transform"]["position_units"] = rat("1");
  q["selections"][0]["transform"]["value_per_price_unit"] = rat("1");
  q["selections"][0]["transform"]["cost_per_outcome"] = rat("0");
  q["selections"][0]["transform"]["return_basis"] = rat("1");
  result = run(q);
  check(result["sections"]["studies"]["data"][2]["data"]["quantiles"][2]
              ["return"] == rat("9007199254740992"));
  check(result["sections"]["studies"]["data"][1]["data"]["variance"]["value"] ==
        rat("1", "4"));
  // Zero-mass terms do not cause an irrelevant square to overflow.
  auto large = q;
  auto no_execution = source;
  auto &no_model = no_execution["sections"]["execution"]["data"][0];
  no_model["support"][0]["price_nanos"] = "9007199254740993";
  no_model["fill_probability"]["value"] = rat("0");
  no_model["no_fill_probability"]["value"] = rat("1");
  large["expected_sha256"] = bind(no_execution);
  large["selections"][0]["mode"] = "execution_mixture";
  large["selections"][0]["nonexecution_pnl"] = rat("0");
  result = run(large);
  check(result["sections"]["studies"]["data"][1]["data"]["variance"]["value"] ==
        rat("0"));
  // No implicit summation of nonprobability weights when no reducer is
  // selected.
  mutate([&](J &m) {
    m["measure"] = "signed_coefficient";
    m["support"][0]["weight"] = rat("1000000000", "999999937");
    m["support"][1]["weight"] = rat("1000000000", "999999929");
    m["support"].push_back({{"price_nanos", "102000000000"},
                            {"weight", rat("1000000000", "999999893")}});
  });
  q["selections"] = support["selections"];
  q["studies"] = J::array();
  result = run(q);
  check(result["status"] == "completed");
  check(result["sections"]["economics"]["data"][0]["atoms"].size() == 3);
  auto changed_census = source;
  changed_census["sections"]["signals"]["data"][0]["source_ordinal"] = "1";
  q = p;
  q["expected_sha256"] = bind(changed_census);
  rejects(q);
  auto duplicate_source = source;
  duplicate_source["sections"]["execution"]["data"].push_back(outcome);
  q = p;
  q["expected_sha256"] = bind(duplicate_source);
  rejects(q);
  p["expected_sha256"] = bind(source);
  // Exact retained output is readable without the source artifact or DBN.
  auto receipt = call("economics", p);
  std::filesystem::remove(root + "/source.json");
  auto query =
      call("result_query",
           {{"protocol", "symphony.sbv.result-query-input.v1"},
            {"path", p["output_path"]},
            {"expected_sha256", receipt["content_sha256"]},
            {"pointer", "/sections/economics/data/0/atoms/2/weight/numerator"},
            {"limit", "1"},
            {"cursor", ""}});
  check(query["nodes"][0]["value"] == "2");
  p["expected_sha256"] = bind(source);
  // A published destination is never replaced.
  bool refused = false;
  try {
    call("economics", p);
  } catch (const std::exception &) {
    refused = true;
  }
  check(refused);
  if (const auto *out = std::getenv("SYMPHONY_SBV_EMIT_ECONOMICS_FIXTURE")) {
    std::filesystem::create_directories(out);
    p["path"] = std::string(out) + "/economics-source.json";
    p["output_path"] = std::string(out) + "/economics-result.json";
    write(p["path"].get<std::string>(), s::seal_result(source));
    write(std::string(out) + "/economics-request.json", p);
  }
  const auto cat =
      call("catalogue", {{"protocol", "symphony.sbv.catalogue-input.v1"}});
  check(cat["transforms"][0]["id"] == "linear_price_pnl");
  check(cat["studies"].size() == 11);
  std::cout << checks << " native economic assertions passed\n";
  return 0;
} catch (const std::exception &ex) {
  std::cerr << ex.what() << '\n';
  return 1;
}
