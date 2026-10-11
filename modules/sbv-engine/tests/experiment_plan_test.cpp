#include "../src/consumer_source.hpp"
#include <filesystem>
#include <fstream>
#include <iostream>
#include <source_location>
#include <unistd.h>
namespace s = symphony::sbv;
namespace d = s::detail;
namespace e = symphony::knowledge::engine;
namespace store = s::result_store;
using J = s::Json;
unsigned checks = 0;
void check(bool good,
           std::source_location loc = std::source_location::current()) {
  ++checks;
  if (!good)
    throw std::runtime_error("referenced experiment assertion " +
                             std::to_string(loc.line()));
}
J read(const std::string &path) {
  std::ifstream f(path);
  J j;
  f >> j;
  return j;
}
store::Reference bundle_ref(const J &r) {
  const auto &v = r.at("storage").at("reference");
  return {v.at("manifest_path"), v.at("manifest_sha256"),
          v.at("content_sha256")};
}
int main(int argc, char **argv) try {
  check(argc == 2);
  const auto templates = read(argv[1]).at("templates");
  char tmp[] = "/private/tmp/sbv-experiment-plan-XXXXXX";
  check(::mkdtemp(tmp));
  const std::string root = tmp;
  struct Cleanup {
    std::string path;
    ~Cleanup() { std::filesystem::remove_all(path); }
  } cleanup{root};
  J options{{"max_page_bytes", nullptr}, {"cache_bytes", "16777216"}};
  unsigned serial = 0;
  auto output = [&](const std::string &name) {
    auto path = root + "/" + name + "-" + d::dec(serial++);
    return J{
        {"kind", "partitioned"},
        {"bundle_path", path},
        {"workspace_path", path + "-work"},
        {"write_options", {{"page_bytes", "16384"}, {"index_fanout", "16"}}},
        {"scratch_read_options", options}};
  };
  auto publish = [&](const J &rows) {
    auto body = d::base("caller-authored large experiment plan");
    body["sections"]["summary"] =
        d::section(J{{"trials", d::dec(rows.size())}});
    body["sections"]["trials"] = d::section(rows);
    auto receipt = d::partitioned_result(
        {{"output", output("input")}}, "experiment", e::no_deadline,
        [&](d::PartitionedOutput &) { return s::logical::Value(body); });
    check(receipt.at("status") == "completed");
    return receipt;
  };
  auto tag = [&](const J &receipt) {
    return J{{"kind", "bundle"},
             {"reference", receipt.at("storage").at("reference")},
             {"selector",
              {{"kind", "pointer"}, {"pointer", "/sections/trials/data"}}},
             {"read_options", options}};
  };
  auto remove = [&](const J &receipt) {
    const auto path = bundle_ref(receipt).manifest_path;
    std::filesystem::remove_all(path.substr(0, path.rfind('/')));
  };
  auto data = d::base("three constant independent values");
  data["sections"]["values"] = d::section(
      J::array({{{"value", "3"}}, {{"value", "3"}}, {{"value", "3"}}}));
  data = s::seal_result(data);
  d::create_file(root + "/observations.json", data.dump(), e::no_deadline);
  auto analysis = templates.at("analyze");
  analysis.erase("output_path");
  analysis["path"] = root + "/observations.json";
  analysis["expected_sha256"] = data.at("content_sha256");
  analysis["pointer"] = "/sections/values/data";
  analysis["value_pointer"] = "/value";
  analysis["value_type"] = "integer";
  analysis["weighting"] = "equal_probability";
  analysis["weight_pointer"] = nullptr;
  analysis["studies"] = J::array({"series_moments"});
  analysis["variance_estimator"] = "population";
  J rows = J::array();
  for (unsigned i = 0; i < 300; ++i)
    rows.push_back({{"id", "pruned-" + d::dec(i)},
                    {"state", "pruned"},
                    {"operation", nullptr},
                    {"request", nullptr},
                    {"reason", std::string(4096, 'x')},
                    {"parameters", J::object()},
                    {"lineage", J::object()}});
  rows.push_back({{"id", "measure"},
                  {"state", "ready"},
                  {"operation", "analyze"},
                  {"request", analysis},
                  {"reason", ""},
                  {"parameters", J::object()},
                  {"lineage", J::object()}});
  check(rows.dump().size() > 1048576);
  auto input = publish(rows);
  J p{{"protocol", "symphony.sbv.experiment-input.v1"},
      {"directory", root + "/journal"},
      {"experiment_id", "referenced"},
      {"trials", {{"kind", "referenced_rows"}, {"source", tag(input)}}},
      {"workers", "2"},
      {"worker_budget", nullptr},
      {"on_failure", "continue"},
      {"extensions", J::object()},
      {"plan_output", output("plan")},
      {"plan_read_options", options},
      {"output", output("results")}};
  std::filesystem::create_directory(root + "/journal");
  auto first = s::dispatch("experiment", p, e::no_deadline);
  if (first.at("status") == "recovery_required")
    throw std::runtime_error(first.dump());
  store::ResultReader reader(bundle_ref(first), {std::nullopt, 16777216});
  check(reader.select("/sections/search/data/trials").describe()["children"] ==
        "301");
  check(reader.select("/sections/trial_plan/data").describe()["children"] ==
        "301");
  check(reader.select("/sections/search/data/executed_this_invocation")
            .read_value() == "1");
  check(reader.select("/sections/summary/data/pruned").read_value() == "300");
  check(reader.select("/sections/summary/data/completed").read_value() == "1");
  const auto saved = read(root + "/journal/experiment-plan.json"),
             child = read(root + "/journal/measure.result.json");
  check(child["sections"]["studies"]["data"][0]["data"]["mean"] ==
        J({{"numerator", "3"}, {"denominator", "1"}}));
  const auto claim_bytes =
      d::read_file(root + "/journal/measure.claim.json", e::no_deadline);
  remove(input);
  std::filesystem::remove(root + "/observations.json");
  p["output"] = output("resume");
  auto resumed = s::dispatch("experiment", p, e::no_deadline);
  if (resumed.at("status") == "recovery_required")
    throw std::runtime_error(resumed.dump());
  store::ResultReader resumed_reader(bundle_ref(resumed),
                                     {std::nullopt, 16777216});
  check(resumed_reader.select("/sections/search/data/executed_this_invocation")
            .read_value() == "0");
  check(resumed_reader.select("/sections/search/data/reused_receipts")
            .read_value() == "1");
  check(read(root + "/journal/experiment-plan.json") == saved);
  check(d::read_file(root + "/journal/measure.claim.json", e::no_deadline) ==
        claim_bytes);
  auto old = p;
  old.erase("output");
  old["output_path"] = root + "/legacy-summary.json";
  s::dispatch("experiment", old, e::no_deadline);
  check(read(root + "/legacy-summary.json")["sections"]["search"]["data"]
                                           ["reused_receipts"] == "1");
  auto bad = p;
  bad["output"] = output("wrong-plan");
  bad["trials"]["source"]["reference"]["content_sha256"] = std::string(64, 'a');
  auto wrong = s::dispatch("experiment", bad, e::no_deadline);
  check(wrong.at("status") == "recovery_required");
  check(d::read_file(root + "/journal/measure.claim.json", e::no_deadline) ==
        claim_bytes);
  const auto manifest =
      p["plan_output"]["bundle_path"].get<std::string>() + "/manifest.json";
  const auto original = d::read_file(manifest, e::no_deadline);
  {
    std::ofstream f(manifest, std::ios::binary | std::ios::trunc);
    f << original << ' ';
  }
  bad = p;
  bad["output"] = output("corrupt-plan");
  check(s::dispatch("experiment", bad, e::no_deadline).at("status") ==
        "recovery_required");
  {
    std::ofstream f(manifest, std::ios::binary | std::ios::trunc);
    f << original;
  }
  // A too-small caller-selected scalar page rejects plan publication before any
  // child claim, retaining the exact inner recovery for explicit inspection.
  auto failed_input = publish(rows);
  auto failure = p;
  failure["directory"] = root + "/failed-journal";
  std::filesystem::create_directory(root + "/failed-journal");
  failure["trials"]["source"] = tag(failed_input);
  failure["plan_output"] = output("failed-plan");
  failure["plan_output"]["write_options"]["page_bytes"] = "512";
  failure["output"] = output("failure-observation");
  auto failed = s::dispatch("experiment", failure, e::no_deadline);
  check(failed.at("status") == "recovery_required");
  const auto recovery_path =
      root + "/failed-journal/experiment-plan-recovery.json";
  auto record = read(recovery_path);
  check(record["sections"]["search"]["data"]["status"] == "recovery_required");
  check(record["sections"]["search"]["data"]["automatic_retry"] == false);
  check(!std::filesystem::exists(root + "/failed-journal/measure.claim.json"));
  check(
      !std::filesystem::exists(root + "/failed-journal/experiment-plan.json"));
  remove(failed_input);
  failure["output"] = output("failure-reobserve");
  check(s::dispatch("experiment", failure, e::no_deadline).at("status") ==
        "recovery_required");
  check(read(recovery_path) == record);
  std::filesystem::remove_all(root + "/journal");
  std::filesystem::remove_all(
      p["plan_output"]["bundle_path"].get<std::string>());
  store::ResultReader portable(bundle_ref(resumed), {std::nullopt, 16777216});
  check(portable.verify_closure()["verification_extent"] ==
        "full_logical_closure");
  std::cout << "referenced plan checks " << checks << " trial bytes "
            << rows.dump().size() << std::endl;
} catch (const std::exception &error) {
  std::cerr << error.what() << '\n';
  return 1;
}
