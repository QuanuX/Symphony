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
void check(bool value,
           std::source_location at = std::source_location::current()) {
  ++checks;
  if (!value)
    throw std::runtime_error("nested split assertion " +
                             std::to_string(at.line()));
}
J data(const J &r, const std::string &name) {
  return r.at("sections").at(name).at("data");
}
J read(const std::string &path) {
  std::ifstream f(path);
  J j;
  f >> j;
  return j;
}
int main(int argc, char **argv) try {
  check(argc == 2);
  auto templates = read(argv[1]).at("templates");
  char tmp[] = "/private/tmp/sbv-nested-XXXXXX";
  const auto *env = std::getenv("SYMPHONY_SBV_NESTED_FIXTURES");
  const auto *created = env ? env : mkdtemp(tmp);
  check(created);
  const std::filesystem::path root(created);
  std::filesystem::create_directories(root);
  auto path = [&](std::string name) { return (root / name).string(); };
  auto write = [&](std::string name, const J &j) {
    d::create_file(path(name), j.dump(), e::no_deadline);
  };
  J rows = J::array();
  for (unsigned i = 0; i < 12; ++i)
    rows.push_back(
        {{"id", d::dec(i)},
         {"x", d::dec(i)},
         {"y", d::dec(1 + 2 * i)},
         {"start_ns", d::dec(10 * i)},
         {"end_ns", d::dec(i == 3 ? 80 : 10 * i)},
         {"available_ns", d::dec(i == 2 ? 999 : 10 * i)},
         {"display_replay",
          {{"before_ns", "999999999999"}, {"after_ns", "999999999999"}}}});
  auto source = d::base("synthetic exact line; interval and availability "
                        "fixture, not market labels");
  source["sections"]["user_extensions"] = d::section({{"observations", rows}});
  source = s::seal_result(source);
  write("source.json", source);
  auto fold = [](std::string id, unsigned a, unsigned b, unsigned c, unsigned z,
                 unsigned cutoff) {
    return J{{"id", id},
             {"train_start_ns", d::dec(a)},
             {"train_end_ns", d::dec(b)},
             {"test_start_ns", d::dec(c)},
             {"test_end_ns", d::dec(z)},
             {"fit_cutoff_ns", d::dec(cutoff)}};
  };
  auto explicit_plan = [](J f) {
    return J{{"kind", "explicit"}, {"folds", f}};
  };
  auto p = templates.at("split");
  p["path"] = path("source.json");
  p["expected_sha256"] = source.at("content_sha256");
  p["retain_rows"] = true;
  J policies;
  for (auto key : {"chronology", "purge", "availability", "purge_before_ns",
                   "purge_after_ns", "embargo_ns"})
    policies[key] = p.at(key);
  auto unrestricted = policies;
  unrestricted["chronology"] = "unrestricted";
  p["plan"] = {
      {"kind", "nested"},
      {"outer", explicit_plan(J::array({fold("a", 0, 80, 80, 100, 80),
                                        fold("b", 20, 100, 100, 120, 100)}))},
      {"inner",
       J::array({J{{"outer_fold_id", "a"},
                   {"population", "outer_train"},
                   {"policies", policies},
                   {"plan", explicit_plan(J::array(
                                {fold("validation", 0, 40, 40, 60, 40),
                                 fold("crosses", 0, 70, 70, 110, 70)}))}},
                 J{{"outer_fold_id", "b"},
                   {"population", "outer_train"},
                   {"policies", unrestricted},
                   {"plan", explicit_plan(J::array({fold("validation", 0, 120,
                                                         40, 60, 120)}))}}})}};
  unsigned serial = 0;
  auto call = [&](J request) {
    request["output_path"] =
        path("split-" + std::to_string(++serial) + ".json");
    auto receipt = s::dispatch("split", request, e::no_deadline);
    auto result = read(request.at("output_path"));
    s::validate_result(result);
    check(receipt.at("content_sha256") == result.at("content_sha256"));
    return result;
  };
  auto result = call(p);
  const auto folds = data(result, "folds");
  check(folds.size() == 5);
  check(folds[0].at("train_indices") ==
        J::array({"0", "1", "4", "5", "6", "7"}));
  check(folds[0].at("test_indices") == J::array({"8", "9"}));
  check(folds[1].at("train_indices") == J::array({"0", "1"}));
  check(folds[1].at("test_indices") == J::array({"4", "5"}));
  check(folds[2].at("train_indices") == J::array({"0", "1", "4", "5", "6"}));
  check(folds[2].at("test_indices") == J::array({"7"}));
  check(folds[3].at("train_indices") ==
        J::array({"3", "4", "5", "6", "7", "8", "9"}));
  check(folds[4].at("train_indices") == J::array({"6", "7", "8", "9"}));
  check(folds[4].at("test_indices") == J::array({"4", "5"}));
  check(folds[1].at("id") == "outer-0/inner-0" &&
        folds[4].at("id") == "outer-1/inner-0");
  check(data(result, "nesting").at("prior_selection_history") == "unknown");
  for (auto i : {1, 2, 4}) {
    check(folds[i].at("nesting").at("outer_test_in_train") == "0");
    check(folds[i].at("nesting").at("test_outside_outer_train") == "0");
  }
  auto reuse = p;
  reuse["plan"]["inner"][0]["population"] = "source";
  auto reused = data(call(reuse), "folds");
  check(reused[2].at("test_indices") == J::array({"7", "8", "9", "10"}));
  check(reused[2].at("nesting").at("outer_test_in_test") == "2");
  check(reused[2].at("nesting").at("test_outside_outer_train") == "3");
  reuse["plan"]["inner"][0]["policies"]["purge"] = "none";
  reuse["plan"]["inner"][0]["policies"]["availability"] = "ignore";
  reused = data(call(reuse), "folds");
  check(reused[1].at("train_indices") == J::array({"0", "1", "2", "3"}));
  check(reused[1].at("nesting").at("train_outside_outer_train") == "2");
  // No replay/lookback inference: huge display windows leave the exact above
  // populations intact. Explicit policies still diagnose the overlap and late
  // row.
  check(reused[1].at("diagnostics").at("retained_overlaps") == "1");
  check(reused[1].at("diagnostics").at("retained_late") == "1");
  auto empty = p;
  empty["plan"]["outer"]["folds"] = J::array();
  empty["plan"]["inner"] = J::array();
  check(data(call(empty), "folds").empty());
  empty = p;
  empty["plan"]["inner"][0]["plan"]["folds"] = J::array();
  check(data(call(empty), "folds").size() == 3);
  auto no_rows = p;
  no_rows["retain_rows"] = false;
  auto index_only = data(call(no_rows), "folds");
  check(index_only[1].at("train_rows").is_null() &&
        index_only[1].at("train_indices") == folds[1].at("train_indices"));
  // More than the obsolete schema's 128-fold annotation is caller-selectable.
  auto many = p;
  many["plan"]["inner"][0]["plan"]["folds"] = J::array();
  for (unsigned i = 0; i < 129; ++i)
    many["plan"]["inner"][0]["plan"]["folds"].push_back(
        fold(d::dec(i), 0, 40, 40, 60, 40));
  many["retain_rows"] = false;
  check(data(call(many), "folds").size() == 132);
  // An unavailable outer fold cannot leak source rows into isolated inner
  // folds.
  auto unavailable = p;
  unavailable["plan"]["outer"]["folds"][0] = fold("a", 200, 210, 220, 230, 210);
  auto absent = data(call(unavailable), "folds");
  check(absent[0].at("status") == "unavailable");
  check(absent[1].at("status") == "unavailable" &&
        absent[1].at("nesting").at("eligible_rows") == "0");
  // Source order, not chronological or parent-subset ordinal, controls indices.
  auto reversed = source;
  auto &reverse_rows =
      reversed["sections"]["user_extensions"]["data"]["observations"];
  std::reverse(reverse_rows.begin(), reverse_rows.end());
  reversed = s::seal_result(reversed);
  write("reversed.json", reversed);
  auto reverse_request = p;
  reverse_request["path"] = path("reversed.json");
  reverse_request["expected_sha256"] = reversed.at("content_sha256");
  auto reverse_result = data(call(reverse_request), "folds");
  check(reverse_result[1].at("train_indices") == J::array({"10", "11"}));
  check(reverse_result[1].at("train_rows")[0].at("id") == "1");
  check(reverse_result[1].at("test_indices") == J::array({"6", "7"}));
  auto rejects = [&](J request) {
    request["output_path"] =
        path("rejected-" + std::to_string(++serial) + ".json");
    bool rejected = false;
    try {
      (void)s::dispatch("split", request, e::no_deadline);
    } catch (const std::exception &) {
      rejected = true;
    }
    check(rejected && !std::filesystem::exists(
                          request.at("output_path").get<std::string>()));
  };
  auto bad = p;
  bad["plan"]["inner"].erase(1);
  rejects(bad);
  bad = p;
  bad["plan"]["inner"][1]["outer_fold_id"] = "a";
  rejects(bad);
  bad = p;
  bad["plan"]["inner"][1]["outer_fold_id"] = "unknown";
  rejects(bad);
  bad = p;
  bad["plan"]["inner"][0]["population"] = "silently_reuse";
  rejects(bad);
  bad = p;
  bad["plan"]["inner"][0]["policies"]["extra"] = true;
  rejects(bad);
  bad = p;
  bad["plan"]["inner"][0]["policies"]["purge"] = "none";
  bad["plan"]["inner"][0]["policies"]["purge_after_ns"] = "1";
  rejects(bad);
  bad = p;
  bad["available_pointer"] = nullptr;
  bad["availability"] = "ignore";
  rejects(bad);
  bad = p;
  bad["plan"]["inner"][0]["plan"]["folds"][1]["id"] = "validation";
  rejects(bad);
  bad = p;
  bad["plan"]["inner"][0]["plan"]["folds"][0]["test_end_ns"] = "0";
  rejects(bad);
  bad = p;
  bad["plan"]["inner"][0]["policies"]["purge_after_ns"] =
      "18446744073709551615";
  rejects(bad);
  auto partitioned = p;
  partitioned.erase("output_path");
  partitioned["output"] = {
      {"kind", "partitioned"},
      {"bundle_path", path("bundle")},
      {"workspace_path", path("scratch")},
      {"write_options", {{"page_bytes", "2048"}, {"index_fanout", "4"}}}};
  auto receipt = s::dispatch("split", partitioned, e::no_deadline);
  auto ref = receipt.at("storage").at("reference");
  J tag{
      {"kind", "bundle"},
      {"reference", ref},
      {"selector", {{"kind", "pointer"}, {"pointer", ""}}},
      {"read_options", {{"max_page_bytes", nullptr}, {"cache_bytes", "4096"}}}};
  auto decoded = d::consumer::source(J{{"source", tag}}, e::no_deadline).value;
  check(decoded.at("sections").at("folds").at("data").materialize() == folds);
  check(decoded.at("sections").at("nesting").at("data").materialize() ==
        data(result, "nesting"));
  // Existing exact-plan dependency orchestration consumes nested retained rows.
  // There is no implicit winner: this fixture explicitly fits each inner and
  // outer fold, predicts its held-out rows, and checks an independently known
  // line.
  auto exp = templates.at("experiment");
  std::filesystem::create_directory(path("journal"));
  exp["directory"] = path("journal");
  exp["experiment_id"] = "nested";
  exp["workers"] = "2";
  exp["trials"] = J::array();
  auto trial = [&](std::string id, std::string operation, J request, J deps,
                   J bindings) {
    request.erase("output_path");
    return J{{"id", id},
             {"state", "ready"},
             {"operation", operation},
             {"request", request},
             {"reason", ""},
             {"parameters", J::object()},
             {"lineage", {{"scope", "synthetic explicit nested evaluation"}}},
             {"depends_on", deps},
             {"bindings", bindings}};
  };
  auto bind = [](std::string target, std::string id, std::string pointer) {
    return J{{"target_pointer", target},
             {"trial_id", id},
             {"result_pointer", pointer}};
  };
  exp["trials"].push_back(
      trial("partitions", "split", p, J::array(), J::array()));
  for (unsigned i = 0; i < folds.size(); ++i) {
    const auto suffix = d::dec(i), ptr = "/sections/folds/data/" + suffix;
    auto fit = templates.at("fit");
    fit["source"] = nullptr;
    auto predict = templates.at("predict");
    predict["source"] = nullptr;
    predict["model"] = nullptr;
    predict["purpose"] = folds[i].at("nesting").at("role") == "inner"
                             ? "inner validation"
                             : "outer evaluation";
    exp["trials"].push_back(
        trial("fit-" + suffix, "fit", fit, J::array({"partitions"}),
              J::array({bind("/source", "partitions", ptr + "/train_rows")})));
    exp["trials"].push_back(trial(
        "predict-" + suffix, "predict", predict,
        J::array({"partitions", "fit-" + suffix}),
        J::array({bind("/source", "partitions", ptr + "/test_rows"),
                  bind("/model", "fit-" + suffix, "/sections/model/data")})));
  }
  exp["output_path"] = path("experiment.json");
  (void)s::dispatch("experiment", exp, e::no_deadline);
  auto first = read(path("experiment.json"));
  check(data(first, "search").at("executed_this_invocation") == "11");
  for (const auto &t : data(first, "search").at("trials"))
    check(t.at("state") == "completed");
  for (unsigned i = 0; i < folds.size(); ++i) {
    auto prediction =
        read(path("journal/predict-" + d::dec(i) + ".result.json"));
    check(data(prediction, "studies")
              .at(0)
              .at("data")
              .at("weighted_squared_error_sum")
              .at("numerator") == "0");
  }
  exp["output_path"] = path("resumed.json");
  (void)s::dispatch("experiment", exp, e::no_deadline);
  auto resumed = read(path("resumed.json"));
  check(data(resumed, "search").at("executed_this_invocation") == "0");
  check(data(resumed, "search").at("reused_receipts") == "11");
  write("nested-request.json", p);
  write("nested-expected.json", result);
  write("nested-partitioned-receipt.json", receipt);
  write("nested-many-request.json", many);
  write("nested-experiment-request.json", exp);
  std::cout << "nested split checks " << checks << "\n";
  return 0;
} catch (const std::exception &error) {
  std::cerr << error.what() << '\n';
  return 1;
}
