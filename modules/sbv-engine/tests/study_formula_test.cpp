#include "../src/consumer_source.hpp"
#include "../src/study_formula.hpp"
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
void check(bool ok, std::source_location at = std::source_location::current()) {
  ++checks;
  if (!ok)
    throw std::runtime_error("formula assertion " + std::to_string(at.line()));
}
J read(const std::string &path) {
  std::ifstream f(path);
  J j;
  f >> j;
  return j;
}
J rat(const char *n, const char *den = "1") {
  return {{"numerator", n}, {"denominator", den}};
}
int main(int argc, char **argv) try {
  check(argc == 2);
  char temp[] = "/private/tmp/sbv-formula-XXXXXX";
  const char *owned = std::getenv("SYMPHONY_SBV_FORMULA_FIXTURES");
  const auto root = std::string(owned ? owned : mkdtemp(temp));
  std::filesystem::create_directories(root);
  auto p = read(argv[1])["variants"]["analyze"]["formula"];
  auto source = d::base("independent formula oracle");
  source["sections"]["values"] =
      d::section(J::array({J{{"value", "2"}, {"weight", rat("2")}},
                           J{{"value", "-1"}, {"weight", rat("-1")}},
                           J{{"value", "5"}, {"weight", rat("0")}}}));
  source = s::seal_result(source);
  p["path"] = root + "/source.json";
  p["expected_sha256"] = source["content_sha256"];
  d::create_file(p["path"], source.dump(), e::no_deadline);
  p["pointer"] = "/sections/values/data";
  p["value_type"] = "integer";
  unsigned seq = 0;
  auto run = [&](J q, bool partitioned = false) {
    const auto name = root + "/result-" + std::to_string(++seq);
    if (partitioned) {
      q.erase("output_path");
      q["output"] = {
          {"kind", "partitioned"},
          {"bundle_path", name},
          {"workspace_path", name + "-work"},
          {"write_options", {{"page_bytes", "16384"}, {"index_fanout", "16"}}}};
    } else {
      q.erase("output");
      q["output_path"] = name + ".json";
    }
    auto receipt = s::dispatch("analyze", q, e::no_deadline);
    check(receipt["status"] == "completed");
    if (!partitioned)
      return read(q["output_path"]);
    J tag{{"kind", "bundle"},
          {"reference", receipt["storage"]["reference"]},
          {"selector", {{"kind", "pointer"}, {"pointer", ""}}},
          {"read_options",
           {{"max_page_bytes", nullptr}, {"cache_bytes", "1048576"}}}};
    return d::consumer::source(J{{"source", tag}}, e::no_deadline)
        .value.materialize();
  };
  auto study = [](const J &result) {
    return result["sections"]["studies"]["data"][0];
  };
  auto legacy = study(run(p)), partitioned = study(run(p, true));
  check(legacy == partitioned && legacy["data"]["value"] == rat("2"));
  check(legacy["descriptor"] == p["study_definitions"][0] &&
        legacy["data"]["unit"] == "caller_unit");
  auto q = p;
  q["weighting"] = "signed_coefficient";
  q["weight_pointer"] = "/weight";
  q["study_definitions"][0]["expression"] = {
      {"op", "sum"},
      {"value",
       {{"op", "multiply"},
        {"left", {{"op", "row_value"}}},
        {"right", {{"op", "row_weight"}}}}}};
  check(study(run(q))["data"]["value"] == rat("5")); // 2*2 + (-1)*(-1) + 5*0
  q = p;
  const J zero{{"op", "constant"}, {"value", rat("0")}};
  auto positive =
      J{{"op", "sum"},
        {"value",
         {{"op", "max"}, {"left", {{"op", "row_value"}}}, {"right", zero}}}};
  auto negative = J{
      {"op", "absolute"},
      {"value",
       {{"op", "sum"},
        {"value",
         {{"op", "min"}, {"left", {{"op", "row_value"}}}, {"right", zero}}}}}};
  q["study_definitions"][0]["expression"] = {
      {"op", "divide"}, {"left", positive}, {"right", negative}};
  check(study(run(q))["data"]["value"] ==
        rat("7")); // explicit profit factor 7/1
  q["study_definitions"][0]["expression"]["right"] = zero;
  check(study(run(q))["status"] == "unavailable" &&
        study(run(q))["data"].is_null());
  q["on_incompatible"] = "reject";
  bool refused = false;
  try {
    (void)run(q);
  } catch (...) {
    refused = true;
  }
  check(refused);
  q["studies"] = J::array();
  auto none = run(q);
  check(none["sections"]["studies"]["status"] == "not_selected");
  check(none["sections"]["studies"]["data"].empty());
  for (J expression :
       {J{{"op", "row_value"}}, J{{"op", "execute"}},
        J{{"op", "sum"}, {"value", {{"op", "count"}}}},
        J{{"op", "constant"}, {"value", rat("1")}, {"extra", true}}}) {
    auto bad = p;
    bad["study_definitions"][0]["expression"] = expression;
    refused = false;
    try {
      (void)run(bad);
    } catch (...) {
      refused = true;
    }
    check(refused);
  }
  q = p;
  q["study_definitions"][0]["id"] = "series_summary";
  refused = false;
  try {
    (void)run(q);
  } catch (...) {
    refused = true;
  }
  check(refused);
  q = p;
  q["study_definitions"][0]["expression"] = {
      {"op", "multiply"},
      {"left",
       {{"op", "constant"},
        {"value", rat("170141183460469231731687303715884105727")}}},
      {"right", {{"op", "constant"}, {"value", rat("2")}}}};
  refused = false;
  try {
    (void)run(q);
  } catch (...) {
    refused = true;
  }
  check(refused);
  std::vector<d::formula::Sample> empty;
  const J minimum{{"op", "minimum"}, {"value", {{"op", "row_value"}}}};
  check(!d::formula::evaluate(minimum, empty, nullptr, e::no_deadline).value);
  check(
      d::formula::evaluate(positive, empty, nullptr, e::no_deadline).value->n ==
      0);
  if (owned) {
    p["output_path"] = root + "/installed-analysis.json";
    d::create_file(root + "/request.json", p.dump(2), e::no_deadline);
  } else
    std::filesystem::remove_all(root);
  std::cout << "formula checks " << checks << '\n';
  return 0;
} catch (const std::exception &error) {
  std::cerr << error.what() << '\n';
  return 1;
}
