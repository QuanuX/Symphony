#include <private_temp.hpp>
#include "../src/consumer_source.hpp"
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
    throw std::runtime_error("binary assertion " + std::to_string(at.line()));
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
J data(const J &r, const char *id) {
  for (const auto &v : r.at("sections").at("studies").at("data"))
    if (v.at("study_id") == id)
      return v;
  throw std::runtime_error("missing study");
}
int main(int argc, char **argv) try {
  check(argc == 2);
  auto temp = symphony::testing::temporary_pattern("sbv-binary-XXXXXX");
  const char *owned = std::getenv("SYMPHONY_SBV_BINARY_FIXTURES");
  const std::string root = owned ? owned : mkdtemp(temp.data());
  std::filesystem::create_directories(root);
  auto catalogue = s::dispatch(
      "catalogue", {{"protocol", "symphony.sbv.catalogue-input.v1"}},
      e::no_deadline);
  for (const auto *id :
       {"binary_confusion", "binary_probability", "binary_ranking"}) {
    bool found = false;
    for (const auto &card : catalogue.at("studies"))
      found |= card.at("id") == id;
    check(found);
  }
  auto p = read(argv[1])["variants"]["analyze"]["binary"];
  p["pointer"] = "/sections/values/data";
  p["weighting"] = "scenario_weight";
  p["weight_pointer"] = "/weight";
  p["binary"]["thresholds"] =
      J::array({J{{"value", rat("1", "2")}, {"positive_when", "ge"}},
                J{{"value", rat("1", "2")}, {"positive_when", "gt"}}});
  p["binary"]["bin_edges"] =
      J::array({rat("0"), rat("1", "4"), rat("1", "2"), rat("1")});
  J rows = J::array(
      {J{{"score", rat("0")}, {"label", "0"}, {"weight", rat("1")}},
       J{{"score", rat("1", "2")}, {"label", "1"}, {"weight", rat("2")}},
       J{{"score", rat("1", "2")}, {"label", "0"}, {"weight", rat("1")}},
       J{{"score", rat("1")}, {"label", "1"}, {"weight", rat("2")}}});
  unsigned seq = 0;
  auto run = [&](J q, J values, bool partitioned = false) {
    const auto name = root + "/result-" + std::to_string(++seq);
    auto source = d::base("independent binary oracle");
    source["sections"]["values"] = d::section(values);
    source["sections"]["summary"] =
        d::section(J{{"rows", d::dec(values.size())}});
    const auto body = source;
    source = s::seal_result(source);
    q["path"] = name + "-source.json";
    q["expected_sha256"] = source["content_sha256"];
    d::create_file(q["path"], source.dump(), e::no_deadline);
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
    if (partitioned) {
      auto sourceOutput = q["output"];
      sourceOutput["bundle_path"] = name + "-source-bundle";
      sourceOutput["workspace_path"] = name + "-source-work";
      auto sourceReceipt = d::partitioned_result(
          {{"output", sourceOutput}}, "run", e::no_deadline,
          [&](d::PartitionedOutput &) { return s::logical::Value(body); });
      check(sourceReceipt["content_sha256"] == source["content_sha256"]);
      q.erase("path");
      q.erase("expected_sha256");
      q["source"] = {
          {"kind", "bundle"},
          {"reference", sourceReceipt["storage"]["reference"]},
          {"selector", {{"kind", "pointer"}, {"pointer", ""}}},
          {"read_options",
           {{"max_page_bytes", nullptr}, {"cache_bytes", "1048576"}}}};
    }
    d::create_file(name + "-request.json", q.dump(2), e::no_deadline);
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
  auto rejects = [&](J q, J values) {
    bool rejected = false;
    try {
      (void)run(q, values);
    } catch (const std::exception &) {
      rejected = true;
    }
    check(rejected);
  };
  auto one = [&](const char *id) {
    auto q = p;
    q["studies"] = J::array({id});
    if (std::string(id) != "binary_confusion")
      q["binary"]["thresholds"] = J::array();
    if (std::string(id) != "binary_probability")
      q["binary"]["bin_edges"] = J::array();
    if (std::string(id) != "binary_ranking")
      q["binary"]["tie_credit"] = nullptr;
    return q;
  };
  const auto result = run(p, rows), bundle = run(p, rows, true);
  for (const auto *id :
       {"binary_confusion", "binary_probability", "binary_ranking"})
    check(data(result, id) == data(bundle, id));
  const auto confusion = data(result, "binary_confusion")["data"]["thresholds"];
  check(confusion[0]["true_positive"] == rat("4") &&
        confusion[0]["false_positive"] == rat("1") &&
        confusion[0]["true_negative"] == rat("1") &&
        confusion[0]["false_negative"] == rat("0"));
  check(confusion[0]["precision"]["value"] == rat("4", "5") &&
        confusion[0]["recall"]["value"] == rat("1") &&
        confusion[0]["specificity"]["value"] == rat("1", "2"));
  check(confusion[0]["accuracy"]["value"] == rat("5", "6") &&
        confusion[0]["f1"]["value"] == rat("8", "9") &&
        confusion[0]["balanced_accuracy"]["value"] == rat("3", "4"));
  check(confusion[1]["true_positive"] == rat("2") &&
        confusion[1]["false_positive"] == rat("0") &&
        confusion[1]["true_negative"] == rat("2") &&
        confusion[1]["false_negative"] == rat("2"));
  check(confusion[1]["precision"]["value"] == rat("1") &&
        confusion[1]["recall"]["value"] == rat("1", "2") &&
        confusion[1]["accuracy"]["value"] == rat("2", "3") &&
        confusion[1]["f1"]["value"] == rat("2", "3"));
  const auto probability = data(result, "binary_probability")["data"];
  check(probability["brier_score"] == rat("1", "8") &&
        probability["weighted_squared_error_sum"] == rat("3", "4"));
  check(probability["expected_calibration_error"] == rat("1", "12") &&
        probability["maximum_calibration_error"] == rat("1", "10"));
  check(probability["bins"][1]["mean_prediction"]["status"] == "unavailable" &&
        probability["bins"][1]["rows"] == "0");
  check(probability["bins"][2]["mean_prediction"]["value"] == rat("7", "10") &&
        probability["bins"][2]["positive_rate"]["value"] == rat("4", "5") &&
        probability["bins"][2]["rows"] == "3");
  const auto ranking = data(result, "binary_ranking")["data"];
  check(ranking["roc_auc"]["value"] == rat("7", "8") &&
        ranking["average_precision"]["value"] == rat("9", "10"));
  check(ranking["pairwise_concordance"]["value"] == rat("7", "8") &&
        ranking["tied_pair_weight"] == rat("2") &&
        ranking["positive_negative_pair_weight"] == rat("8"));
  check(ranking["curve"].size() == 4 &&
        ranking["curve"][0]["precision"]["status"] == "unavailable");
  auto q = one("binary_ranking");
  q["binary"]["tie_credit"] = rat("0");
  check(data(run(q, rows),
             "binary_ranking")["data"]["pairwise_concordance"]["value"] ==
        rat("3", "4"));
  q["binary"]["tie_credit"] = rat("1");
  check(data(run(q, rows),
             "binary_ranking")["data"]["pairwise_concordance"]["value"] ==
        rat("1"));
  auto reversed = rows;
  std::reverse(reversed.begin(), reversed.end());
  check(data(run(p, reversed), "binary_ranking") ==
        data(result, "binary_ranking"));
  q = p;
  q["binary"]["weight_normalization"] = "require_unit_mass";
  check(data(run(q, rows), "binary_probability")["status"] == "unavailable");
  auto unit = rows;
  unit[0]["weight"] = rat("1", "6");
  unit[1]["weight"] = rat("1", "3");
  unit[2]["weight"] = rat("1", "6");
  unit[3]["weight"] = rat("1", "3");
  check(data(run(q, unit), "binary_probability")["data"]["brier_score"] ==
        rat("1", "8"));
  q = p;
  q["binary"]["positive_label"] = rat("0");
  q["binary"]["negative_label"] = rat("1");
  check(data(run(q, rows), "binary_ranking")["data"]["roc_auc"]["value"] ==
        rat("1", "8"));
  q = p;
  q["weighting"] = "equal_probability";
  q["weight_pointer"] = nullptr;
  check(data(run(q, rows), "binary_ranking")["data"]["roc_auc"]["value"] ==
        rat("7", "8"));
  auto excluded = rows;
  excluded[0].erase("score");
  excluded[1].erase("label");
  excluded[2]["label"] = "2";
  q = p;
  q["missing"] = "exclude";
  q["binary"]["unknown_label"] = "exclude";
  q["studies"].push_back("series_summary");
  auto filtered = run(q, excluded);
  check(filtered["sections"]["summary"]["data"]["retained_rows"] == "3");
  auto bi = filtered["sections"]["binary_input"]["data"];
  check(bi["retained_rows"] == "1" && bi["excluded_rows"] == "2" &&
        bi["rows"][0]["source_index"] == "3" &&
        bi["exclusions"][0]["source_index"] == "1");
  rejects(p, excluded);
  q["binary"]["unknown_label"] = "reject";
  rejects(q, excluded);
  q = p;
  q["binary"]["retain_rows"] = false;
  check(run(q, rows)["sections"]["binary_input"]["data"]["rows"].empty());
  for (const auto *id :
       {"binary_confusion", "binary_probability", "binary_ranking"})
    check(run(one(id), rows)["sections"]["studies"]["data"].size() == 1);
  q = p;
  q.erase("binary");
  q["studies"] = J::array();
  auto noLabels = rows;
  for (auto &row : noLabels)
    row.erase("label");
  auto none = run(q, noLabels);
  check(none["sections"]["studies"]["status"] == "not_selected" &&
        none["sections"]["studies"]["data"].empty() &&
        !none["sections"].contains("binary_input"));
  auto zero = rows;
  for (auto &r : zero)
    r["weight"] = rat("0");
  check(data(run(p, zero), "binary_probability")["status"] == "unavailable");
  check(data(run(p, J::array()), "binary_ranking")["status"] == "unavailable");
  auto single = rows;
  for (auto &r : single)
    r["label"] = "0";
  auto singleResult = run(p, single);
  check(data(singleResult, "binary_ranking")["data"]["roc_auc"]["status"] ==
            "unavailable" &&
        data(singleResult,
             "binary_ranking")["data"]["average_precision"]["status"] ==
            "unavailable");
  q = p;
  q["weighting"] = "signed_coefficient";
  auto negative = rows;
  negative[0]["weight"] = rat("-1");
  check(data(run(q, negative), "binary_ranking")["status"] == "unavailable");
  q["on_incompatible"] = "reject";
  rejects(q, negative);
  rejects(p, negative);
  auto outside = rows;
  outside[0]["score"] = rat("-1");
  auto unbounded = run(p, outside);
  check(data(unbounded, "binary_probability")["status"] == "unavailable" &&
        data(unbounded, "binary_ranking")["status"] == "available");
  q = p;
  q["on_incompatible"] = "reject";
  rejects(q, outside);
  // Comparison must retain distinct integers beyond binary64 precision.
  auto large = J::array({J{{"score", rat("9007199254740992")},
                           {"label", "0"},
                           {"weight", rat("1")}},
                         J{{"score", rat("9007199254740993")},
                           {"label", "1"},
                           {"weight", rat("1")}}});
  check(data(run(one("binary_ranking"), large),
             "binary_ranking")["data"]["roc_auc"]["value"] == rat("1"));
  auto rationalLabels = rows;
  for (auto &r : rationalLabels)
    r["label"] = rat(r["label"] == "1" ? "1" : "0");
  q = p;
  q["binary"]["label_type"] = "rational";
  check(data(run(q, rationalLabels), "binary_probability")["data"] ==
        probability);
  for (int bad = 0; bad < 8; ++bad) {
    q = p;
    switch (bad) {
    case 0:
      q["binary"]["bin_edges"][1] = rat("0");
      break;
    case 1:
      q["binary"]["bin_edges"][0] = rat("1", "8");
      break;
    case 2:
      q["binary"]["tie_credit"] = rat("2");
      break;
    case 3:
      q["binary"]["positive_label"] = rat("0");
      break;
    case 4:
      q["binary"]["thresholds"][0]["positive_when"] = "implicit";
      break;
    case 5:
      q.erase("binary");
      break;
    case 6:
      q["studies"] = J::array();
      break;
    case 7:
      q = one("binary_ranking");
      q["binary"]["bin_edges"] = J::array({rat("0"), rat("1")});
      break;
    }
    rejects(q, rows);
  }
  if (owned) {
    auto request = read(root + "/result-1-request.json");
    request["output_path"] = root + "/installed-analysis.json";
    d::create_file(root + "/request.json", request.dump(2), e::no_deadline);
  } else
    std::filesystem::remove_all(root);
  std::cout << "binary study checks " << checks << '\n';
  return 0;
} catch (const std::exception &error) {
  std::cerr << error.what() << '\n';
  return 1;
}
