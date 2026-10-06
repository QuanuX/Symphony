#include "../src/detail.hpp"
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
    throw std::runtime_error("dependency assertion " +
                             std::to_string(l.line()));
}
J data(const J &r, const std::string &name) {
  return r.at("sections").at(name).at("data");
}
int main(int argc, char **argv) try {
  check(argc == 2);
  const auto templates =
      e::parse_bounded_json(d::read_file(argv[1], e::no_deadline),
                            d::artifact_bytes, d::artifact_values)
          .at("templates");
  char tmp[] = "/private/tmp/sbv-dependencies-XXXXXX";
  const auto *env = std::getenv("SYMPHONY_SBV_DEPENDENCIES_FIXTURES");
  const auto *created = env ? env : mkdtemp(tmp);
  check(created);
  std::filesystem::path root(created);
  std::filesystem::create_directories(root);
  auto path = [&](const std::string &n) { return (root / n).string(); };
  auto write = [&](const std::string &p, const J &j) {
    d::create_file(p, j.dump(), e::no_deadline);
  };
  auto read = [&](const std::string &p) {
    auto r = e::parse_bounded_json(d::read_file(p, e::no_deadline),
                                   d::artifact_bytes, d::artifact_values);
    s::validate_result(r);
    return r;
  };
  auto source = d::base("independent exact line with explicit point intervals");
  J observations = J::array();
  for (int i = 0; i < 12; ++i)
    observations.push_back({{"id", d::dec(i)},
                            {"x", d::dec(i)},
                            {"y", d::dec(1 + 2 * i)},
                            {"start_ns", d::dec(10 * i)},
                            {"end_ns", d::dec(10 * i)},
                            {"available_ns", d::dec(10 * i)}});
  source["sections"]["user_extensions"] =
      d::section({{"observations", observations}});
  source = s::seal_result(source);
  write(path("source.json"), source);
  auto split = templates.at("split");
  split["path"] = path("source.json");
  split["expected_sha256"] = source.at("content_sha256");
  split["retain_rows"] = true;
  split["plan"]["first_train_end_ns"] = "60";
  split["plan"]["first_fit_cutoff_ns"] = "60";
  split["plan"]["fold_count"] = "2";
  auto fit = templates.at("fit"), predict = templates.at("predict");
  fit["source"] = nullptr;
  predict["source"] = nullptr;
  predict["model"] = nullptr;
  auto trial = [&](std::string id, std::string op, J req, J deps = J::array(),
                   J bindings = J::array()) {
    req.erase("output_path");
    return J{{"id", id},
             {"state", "ready"},
             {"operation", op},
             {"request", req},
             {"reason", ""},
             {"parameters", J::object()},
             {"lineage", J::object()},
             {"depends_on", deps},
             {"bindings", bindings}};
  };
  auto bind = [&](std::string target, std::string id, std::string pointer) {
    return J{{"target_pointer", target},
             {"trial_id", id},
             {"result_pointer", pointer}};
  };
  auto exp = templates.at("experiment");
  exp["experiment_id"] = "graph";
  exp["workers"] = "2";
  exp["trials"] = J::array();
  // Forward references, fork and join: order in the ledger is the user's order.
  for (int i = 0; i < 2; ++i) {
    auto suffix = d::dec(i), base = "/sections/folds/data/" + suffix;
    exp["trials"].push_back(trial(
        "predict" + suffix, "predict", predict,
        J::array({"split", "fit" + suffix}),
        J::array({bind("/source", "split", base + "/test_rows"),
                  bind("/model", "fit" + suffix, "/sections/model/data")})));
    exp["trials"].push_back(
        trial("fit" + suffix, "fit", fit, J::array({"split"}),
              J::array({bind("/source", "split", base + "/train_rows")})));
  }
  exp["trials"].push_back(trial("split", "split", split));
  unsigned seq = 0;
  auto fresh = [&](J p) {
    p["directory"] = path("journal-" + d::dec(++seq));
    std::filesystem::create_directory(p.at("directory").get<std::string>());
    return p;
  };
  auto call = [&](J p) {
    p["output_path"] = path("summary-" + d::dec(++seq) + ".json");
    auto receipt = s::dispatch("experiment", p, e::no_deadline);
    auto r = read(p.at("output_path"));
    check(receipt.at("content_sha256") == r.at("content_sha256"));
    return r;
  };
  auto reject = [&](J p, bool preflight = true) {
    p = fresh(p);
    p["output_path"] = path("rejected-" + d::dec(++seq) + ".json");
    bool failed = false;
    try {
      s::dispatch("experiment", p, e::no_deadline);
    } catch (const std::exception &) {
      failed = true;
    }
    check(failed);
    check(!std::filesystem::exists(p.at("output_path").get<std::string>()));
    if (preflight)
      check(!std::filesystem::exists(p.at("directory").get<std::string>() +
                                     "/experiment-plan.json"));
  };
  auto plan = fresh(exp);
  const auto dir = plan.at("directory").get<std::string>();
  auto result = call(plan);
  check(result.at("status") == "completed");
  auto ledger = data(result, "search").at("trials");
  check(data(result, "search").at("wave_count") == "3");
  check(data(result, "summary").at("completed") == "5");
  check(data(result, "resources").at("actual_outer_workers") == "2");
  for (std::size_t i = 0; i < ledger.size(); ++i) {
    check(ledger[i].at("id") == plan["trials"][i]["id"]);
    check(ledger[i].at("state") == "completed");
    auto claim = data(
        read(dir + "/" + ledger[i].at("id").get<std::string>() + ".claim.json"),
        "search");
    check(e::sha256_hex(claim.at("resolved_request").dump()) ==
          ledger[i].at("request_sha256").get<std::string>());
    check(claim.at("dependencies") == ledger[i].at("dependencies"));
    check(claim.at("bindings") == ledger[i].at("bindings"));
  }
  for (int i = 0; i < 2; ++i) {
    auto model = read(dir + "/fit" + d::dec(i) + ".result.json");
    check(data(model, "model").at("coefficients") ==
          J::array({J{{"numerator", "1"}, {"denominator", "1"}},
                    J{{"numerator", "2"}, {"denominator", "1"}}}));
    auto pred = read(dir + "/predict" + d::dec(i) + ".result.json");
    check(data(pred, "training_overlap").at("reused_count") == "0");
    for (const auto &row : data(pred, "predictions"))
      check(row.at("prediction") == row.at("target"));
    // Direct native invocation from the durable resolved request is
    // byte-equivalent.
    auto req =
        data(read(dir + "/predict" + d::dec(i) + ".claim.json"), "search")
            .at("resolved_request");
    req["output_path"] = path("direct-" + d::dec(i) + ".json");
    s::dispatch("predict", req, e::no_deadline);
    check(read(req.at("output_path")) == pred);
  }
  auto resumed = call(plan);
  check(data(resumed, "search").at("executed_this_invocation") == "0");
  check(data(resumed, "search").at("reused_receipts") == "5");
  check(data(resumed, "search").at("trials") == ledger);
  auto serial = exp;
  serial["workers"] = "1";
  serial = fresh(serial);
  auto serial_result = call(serial);
  check(data(serial_result, "summary") == data(result, "summary"));
  for (int i = 0; i < 2; ++i)
    check(data(read(serial.at("directory").get<std::string>() + "/predict" +
                    d::dec(i) + ".result.json"),
               "predictions") ==
          data(read(dir + "/predict" + d::dec(i) + ".result.json"),
               "predictions"));
  // Complete graph admission before journal creation.
  auto q = exp;
  q["trials"][4]["depends_on"] = J::array({"predict0"});
  reject(q);
  q = exp;
  q["trials"][1]["depends_on"] = J::array({"fit0"});
  reject(q);
  q = exp;
  q["trials"][1]["depends_on"] = J::array({"unknown"});
  reject(q);
  q = exp;
  q["trials"][1]["depends_on"] = J::array({"split", "split"});
  reject(q);
  q = exp;
  q["trials"][1]["depends_on"] = nullptr;
  reject(q);
  q = exp;
  q["trials"][1]["bindings"] = nullptr;
  reject(q);
  q = exp;
  q["trials"][1]["depends_on"] = J::array();
  reject(q);
  for (auto target : {"", "/no_slot", "/output_path", "/protocol",
                      "/source/child", "/bad~2pointer"}) {
    q = exp;
    q["trials"][1]["bindings"][0]["target_pointer"] = target;
    reject(q);
  }
  q = exp;
  q["trials"][1]["bindings"][0]["result_pointer"] = "/bad~2pointer";
  reject(q);
  q = exp;
  q["trials"][1]["bindings"].push_back(q["trials"][1]["bindings"][0]);
  reject(q);
  // Missing pointer blocks before claim, with available siblings still
  // executing.
  q = exp;
  q["trials"][1]["bindings"][0]["result_pointer"] = "/absent";
  q = fresh(q);
  auto blocked = call(q);
  check(data(blocked, "summary").at("blocked") == "2");
  check(data(blocked, "summary").at("completed") == "3");
  check(!std::filesystem::exists(q.at("directory").get<std::string>() +
                                 "/fit0.claim.json"));
  check(data(call(q), "search").at("executed_this_invocation") == "0");
  // Failure and caller pruning propagate without fabricating failed child runs.
  q = exp;
  q["trials"][4]["request"]["path"] = path("missing.json");
  q = fresh(q);
  auto failed = call(q);
  check(data(failed, "summary").at("failed") == "1");
  check(data(failed, "summary").at("blocked") == "4");
  q = exp;
  q["trials"][4].update({{"state", "pruned"},
                         {"reason", "user choice"},
                         {"operation", nullptr},
                         {"request", nullptr}});
  q = fresh(q);
  auto pruned = call(q);
  check(data(pruned, "summary").at("pruned") == "1");
  check(data(pruned, "summary").at("blocked") == "4");
  // Unavailable model is a completed producer; consuming its null model fails
  // visibly.
  q = exp;
  q["trials"][4]["request"]["plan"]["first_train_end_ns"] = "10";
  q["trials"][4]["request"]["plan"]["first_fit_cutoff_ns"] = "10";
  q = fresh(q);
  auto unavailable = call(q);
  check(data(unavailable, "search").at("trials")[1].at("state") == "completed");
  check(data(unavailable, "search").at("trials")[0].at("state") == "failed");
  // Stable topological stop order, not unresolvable caller array order.
  q = exp;
  q["workers"] = "1";
  q["on_failure"] = "stop";
  q["trials"][1]["bindings"][0]["result_pointer"] = "/absent";
  q = fresh(q);
  auto stopped = call(q);
  check(data(stopped, "summary").at("completed") == "1");
  check(data(stopped, "summary").at("blocked") == "1");
  check(data(stopped, "summary").at("not_started") == "3");
  // A simulated interrupted middle node: keep its exact durable claim, remove
  // completion.
  q = fresh(exp);
  call(q);
  auto uncertain = q.at("directory").get<std::string>();
  for (auto suffix : {".receipt.json", ".result.json"})
    std::filesystem::remove(uncertain + "/fit0" + suffix);
  for (auto suffix : {".claim.json", ".receipt.json", ".result.json"})
    std::filesystem::remove(uncertain + "/predict0" + suffix);
  auto ambiguity = call(q);
  check(data(ambiguity, "summary").at("ambiguous") == "1");
  check(data(ambiguity, "summary").at("blocked") == "1");
  check(data(ambiguity, "search").at("executed_this_invocation") == "0");
  check(!std::filesystem::exists(uncertain + "/predict0.claim.json"));
  // Explicit ordering-only edges and escaped nested destinations are supported.
  q = exp;
  q["trials"][1]["request"]["extensions"]["a/b~c"] = nullptr;
  q["trials"][1]["bindings"].push_back(
      bind("/extensions/a~1b~0c", "split", ""));
  q["trials"][3]["depends_on"].push_back("fit0");
  q = fresh(q);
  auto nested = call(q);
  check(data(nested, "summary").at("completed") == "5");
  check(data(nested, "search").at("wave_count") == "4");
  auto nested_fit =
      read(q.at("directory").get<std::string>() + "/fit0.result.json");
  check(data(nested_fit, "user_extensions").at("a/b~c").at("pointer") == "");
  // Invalid array-index syntax is a blocked selection, not an abort of
  // independent work.
  q = exp;
  q["trials"][1]["bindings"][0]["result_pointer"] =
      "/sections/folds/data/00/train_rows";
  q = fresh(q);
  check(data(call(q), "summary").at("blocked") == "2");
  q = exp;
  q["trials"] = J::array();
  q = fresh(q);
  auto empty = call(q);
  check(empty.at("status") == "completed");
  check(data(empty, "search").at("wave_count") == "0");
  check(data(empty, "resources").at("actual_outer_workers") == "0");
  // Reconciliation must not silently use changed dependencies or resolved
  // claims.
  auto altered = read(dir + "/split.result.json");
  {
    std::ofstream f(dir + "/split.result.json", std::ios::app);
    f << ' ';
  }
  bool refused = false;
  try {
    call(plan);
  } catch (const std::exception &) {
    refused = true;
  }
  check(refused);
  {
    std::ofstream f(dir + "/split.result.json");
    f << altered.dump();
  }
  auto claim = read(dir + "/fit0.claim.json");
  auto corrupt = claim;
  corrupt["sections"]["search"]["data"]["resolved_request"]["source"]
         ["pointer"] = "/different";
  corrupt = s::seal_result(corrupt);
  {
    std::ofstream f(dir + "/fit0.claim.json");
    f << corrupt.dump();
  }
  refused = false;
  try {
    call(plan);
  } catch (const std::exception &) {
    refused = true;
  }
  check(refused);
  {
    std::ofstream f(dir + "/fit0.claim.json");
    f << claim.dump();
  }
  if (env) {
    auto installed = exp;
    installed["output_path"] = path("installed-summary.json");
    installed["directory"] = path("installed-journal");
    write(path("experiment-request.json"), installed);
  }
  std::cout << "SBV dependency assertions: " << checks << "\n";
  return 0;
} catch (const std::exception &error) {
  std::cerr << error.what() << '\n';
  return 1;
}
