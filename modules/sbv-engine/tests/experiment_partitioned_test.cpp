#include <private_temp.hpp>
#include "../../sqav-databento-dbn-cpp/tests/public_fixture.hpp"
#include "../src/dataset.hpp"
#include "../src/experiment_paths.hpp"
#include "../src/experiment_result.hpp"
#include "../src/stream_census.hpp"
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
void check(bool yes,
           std::source_location at = std::source_location::current()) {
  ++checks;
  if (!yes)
    throw std::runtime_error("partitioned run check line " +
                             std::to_string(at.line()));
}
J read(const std::string &path) {
  std::ifstream file(path);
  J result;
  file >> result;
  return result;
}
std::string fixture(std::uint64_t count) {
  const auto source = public_fixture::fixture();
  std::vector<unsigned char> bytes(source.begin(), source.begin() + 360);
  auto put = [&](std::size_t offset, std::uint64_t value, unsigned length) {
    for (unsigned i = 0; i < length; ++i)
      bytes[offset + i] = static_cast<unsigned char>(value >> (i * 8));
  };
  // Independently encoded mechanical MBO sequence; not market-performance data.
  // Header selected record limit is zero; every emitted record is preserved.
  put(42, 0, 8);
  for (std::uint64_t i = 0; i < count; ++i) {
    const auto at = bytes.size();
    bytes.insert(bytes.end(), source.begin() + 360, source.begin() + 416);
    put(at + 8, 1609160400000000000ULL + i * 100, 8);
    put(at + 16, i + 100, 8);
    put(at + 24, (100 + i % 3) * 1000000000ULL, 8);
    put(at + 32, 1 + i % 5, 4);
    bytes[at + 36] = 128;
    bytes[at + 37] = 3;
    bytes[at + 38] = 'T';
    bytes[at + 39] = i % 2 ? 'A' : 'B';
    put(at + 40, 1609160400000000000ULL + i * 100 + 17, 8);
    put(at + 48, 17, 4);
    put(at + 52, i + 1, 4);
  }
  return std::string(reinterpret_cast<const char *>(bytes.data()),
                     bytes.size());
}
store::Reference ref(const J &receipt) {
  const auto &r = receipt.at("storage").at("reference");
  return {r.at("manifest_path"), r.at("manifest_sha256"),
          r.at("content_sha256")};
}
J whole(const J &receipt) {
  store::ResultReader reader(ref(receipt));
  return reader.select_node(0).read_value();
}
J science(J value) {
  value.erase("content_sha256");
  value["sections"]["resources"]["data"].erase("derived_resources");
  return value;
}
int main(int argc, char **argv) try {
  check(argc == 2);
  auto templates = read(argv[1]);
  auto temporary = symphony::testing::temporary_pattern("sbv-trials-partitioned-XXXXXX");
  check(::mkdtemp(temporary.data()));
  const std::string root = temporary;
  struct Cleanup {
    std::string root;
    ~Cleanup() { std::filesystem::remove_all(root); }
  } cleanup{root};
  const auto raw = fixture(8);
  d::create_file(root + "/source.dbn", raw, e::no_deadline);
  J options{{"max_page_bytes", nullptr}, {"cache_bytes", "1048576"}};
  auto output = [&](const std::string &name) {
    return J{
        {"kind", "partitioned"},
        {"bundle_path", root + "/" + name + "-bundle"},
        {"workspace_path", root + "/" + name + "-work"},
        {"write_options", {{"page_bytes", "65536"}, {"index_fanout", "8"}}}};
  };
  auto run = templates["variants"]["run"]["partitioned"];
  run["source_path"] = root + "/source.dbn";
  run["source_sha256"] = e::sha256_hex(raw);
  run["criteria"]["spacing_ns"] = "0";
  run["criteria"]["max_signals"] = nullptr;
  run["criteria"]["min_trade_size"] = "1";
  run["execution"]["model"] = "none";
  run["execution"]["price_offsets_nanos"] = J::array();
  run["execution"]["probability_numerator"] = "0";
  run["execution"]["probability_denominator"] = "1";
  run["output"] = output("run");
  auto evaluation = templates["variants"]["evaluate"]["partitioned"];
  evaluation["source_path"] = run["source_path"];
  evaluation["source_sha256"] = run["source_sha256"];
  evaluation["census"] = nullptr;
  evaluation["model"]["parameters"] = {{"levels_per_side", "1"},
                                       {"include_anchor", true},
                                       {"thin_support", "unavailable"}};
  evaluation["output"] = output("evaluate");
  auto economics = templates["variants"]["economics"]["partitioned"];
  economics["source"] = nullptr;
  economics["selections"] = templates["templates"]["economics"]["selections"];
  economics["selections"][0]["signal_id"] = "signal-1";
  economics["output"] = output("economics");
  auto legacy = run;
  legacy.erase("output");
  auto load = templates["templates"]["dataset_load"];
  load["directory"] = root;
  load["instance_id"] = "00112233445566778899aabbccddeeff";
  for (const auto *key : {"source_path", "source_sha256", "dataset"})
    load[key] = run[key];
  load["idle_timeout_ms"] = "0";
  load["memory_budget_bytes"] = nullptr;
  load["max_concurrent_jobs"] = nullptr;
  load["worker_budget"] = nullptr;
  s::dispatch("dataset_load", load, e::no_deadline);
  J release{{"protocol", "symphony.sbv.dataset-release-input.v1"},
            {"directory", root},
            {"instance_id", load["instance_id"]}};
  struct Release {
    J request;
    ~Release() {
      try {
        s::dispatch("dataset_release", request, e::no_deadline);
      } catch (...) {
      }
    }
  } release_guard{release};
  J resident{{"protocol", "symphony.sbv.dataset-execute-input.v1"},
             {"directory", root},
             {"instance_id", load["instance_id"]},
             {"operation", "run"},
             {"request", legacy},
             {"output", output("resident")}};
  auto trial = [&](const char *name, const char *op, J request,
                   const char *parent, const char *target) {
    J row{{"id", name},
          {"state", "ready"},
          {"operation", op},
          {"request", request},
          {"reason", ""},
          {"parameters", J::object()},
          {"lineage", J::object()},
          {"depends_on", J::array()},
          {"bindings", J::array()}};
    if (request.contains("output"))
      row["result_read_options"] = options;
    if (*parent) {
      row["depends_on"].push_back(parent);
      row["bindings"].push_back({{"target_pointer", target},
                                 {"trial_id", parent},
                                 {"result_pointer", ""}});
    }
    return row;
  };
  std::filesystem::create_directory(root + "/journal");
  // Caller order deliberately precedes dependencies.
  J plan{{"protocol", "symphony.sbv.experiment-input.v1"},
         {"output_path", root + "/summary.json"},
         {"directory", root + "/journal"},
         {"experiment_id", "mixed"},
         {"trials",
          J::array({trial("economics", "economics", economics, "evaluate",
                          "/source"),
                    trial("evaluate", "evaluate", evaluation, "run", "/census"),
                    trial("run", "run", run, "", ""),
                    trial("legacy", "run", legacy, "", ""),
                    trial("resident", "dataset_execute", resident, "", "")})},
         {"workers", "2"},
         {"on_failure", "continue"},
         {"extensions", J::object()}};
  auto result = s::dispatch("experiment", plan, e::no_deadline);
  check(result["summary"]["data"]["completed"] == "5");
  auto summary = read(root + "/summary.json");
  auto ledger = summary["sections"]["search"]["data"]["trials"];
  check(ledger[0]["state"] == "completed");
  check(ledger[0]["bindings"][0]["reference"]["kind"] == "bundle");
  auto produced = whole(ledger[2]["result"]);
  auto classical = read(root + "/journal/legacy.result.json");
  check(produced["sections"]["signals"] == classical["sections"]["signals"]);
  check(produced["sections"]["replay"] == classical["sections"]["replay"]);
  auto actual = whole(ledger[0]["result"]);
  check(actual["sections"]["economics"]["data"].size() == 1);
  auto ram = whole(ledger[4]["result"]);
  check(ram["sections"]["signals"] == produced["sections"]["signals"]);
  check(ram["sections"]["resources"]["data"]["dataset_feed"]
           ["source_reads_this_job"] == "0");
  s::dispatch("dataset_release", release, e::no_deadline);
  // Source removal must not prevent exact completed-child reconciliation.
  std::filesystem::remove(root + "/source.dbn");
  plan["output_path"] = root + "/resumed.json";
  s::dispatch("experiment", plan, e::no_deadline);
  auto resumed = read(root + "/resumed.json");
  check(resumed["sections"]["search"]["data"]["executed_this_invocation"] ==
        "0");
  check(resumed["sections"]["search"]["data"]["reused_receipts"] == "5");
  auto reject = [&](J bad) {
    bool refused = false;
    try {
      s::dispatch("experiment", bad, e::no_deadline);
    } catch (...) {
      refused = true;
    }
    check(refused);
  };
  // Changed manifest bytes cannot be adopted through a retained logical hash.
  const auto manifest = root + "/run-bundle/manifest.json";
  auto bytes = d::read_file(manifest, e::no_deadline);
  {
    std::ofstream f(manifest, std::ios::app);
    f << '\n';
  }
  plan["output_path"] = root + "/tampered.json";
  reject(plan);
  {
    std::ofstream f(manifest);
    f << bytes;
  }
  auto wrong = ledger[2]["result"];
  wrong["protocol"] = "symphony.sbv.economics.v1";
  bool refused = false;
  try {
    d::trial::verify(wrong, run, "run", options, e::no_deadline);
  } catch (...) {
    refused = true;
  }
  check(refused);
  auto logical = d::trial::verify(ledger[2]["result"], run, "run", options,
                                  e::no_deadline);
  check(!d::trial::selects(logical, "/sections/absent"));
  check(d::trial::selects(logical, "/sections/signals/data/0"));
  check(!d::trial::selects(logical, "/sections/signals/data/00"));
  auto wrong_output = run;
  wrong_output["output"] = output("wrong");
  refused = false;
  try {
    d::trial::verify(ledger[2]["result"], wrong_output, "run", options,
                     e::no_deadline);
  } catch (...) {
    refused = true;
  }
  check(refused);
  // Failure after workspace creation is retained as recovery, never completion.
  std::filesystem::create_directory(root + "/recovery-journal");
  auto recovery = plan;
  recovery["directory"] = root + "/recovery-journal";
  recovery["output_path"] = root + "/recovery-summary.json";
  recovery["trials"] =
      J::array({trial("run", "run", run, "", ""),
                trial("evaluate", "evaluate", evaluation, "run", "/census")});
  recovery["trials"][0]["request"]["output"] = output("failed");
  recovery["trials"][1]["request"]["output"] = output("blocked");
  auto failed = s::dispatch("experiment", recovery, e::no_deadline);
  check(failed["summary"]["data"]["recovery_required"] == "1");
  check(failed["summary"]["data"]["blocked"] == "1");
  check(
      !std::filesystem::exists(root + "/recovery-journal/evaluate.claim.json"));
  recovery["output_path"] = root + "/recovery-resume.json";
  s::dispatch("experiment", recovery, e::no_deadline);
  check(read(root + "/recovery-resume.json")["sections"]["search"]["data"]
                                            ["executed_this_invocation"] ==
        "0");
  // Destination aliases and nested workspaces are rejected before any plan.
  std::filesystem::create_directory(root + "/alias-journal");
  auto alias = recovery;
  alias["directory"] = root + "/alias-journal";
  alias["output_path"] = root + "/alias-summary.json";
  alias["trials"][0]["request"]["output"] = output("same");
  alias["trials"][1]["request"]["output"] = output("same");
  reject(alias);
  check(!std::filesystem::exists(root + "/alias-journal/experiment-plan.json"));
  alias["trials"][1]["request"]["output"] = output("another");
  alias["trials"][0]["request"]["output"]["bundle_path"] =
      root + "/alias-journal/nested";
  reject(alias);
  check(!std::filesystem::exists(root + "/alias-journal/experiment-plan.json"));
  // Ask this filesystem whether case and Unicode spellings alias; assert the
  // destination checker makes that same decision for not-yet-created paths.
  unsigned aliases = 0;
  for (const auto &pair : std::vector<std::pair<std::string, std::string>>{
           {"Case", "case"}, {"caf\xc3\xa9", "cafe\xcc\x81"}}) {
    const auto first = root + "/" + pair.first,
               second = root + "/" + pair.second;
    std::filesystem::create_directory(first);
    bool same = std::filesystem::exists(second) &&
                std::filesystem::equivalent(first, second);
    std::filesystem::remove(first);
    refused = false;
    try {
      d::trial::destinations(root + "/alias-journal",
                             {{first, true}, {second, true}}, e::no_deadline);
    } catch (...) {
      refused = true;
    }
    check(refused == same);
    aliases += same;
  }
  std::cout << "filesystem alias pairs " << aliases << '\n';
  std::cout << "partitioned experiment checks " << checks << '\n';
} catch (const std::exception &error) {
  std::cerr << error.what() << '\n';
  return 1;
}
