#include "../src/wide_rational.hpp"
#include <cstdlib>
#include <fcntl.h>
#include <filesystem>
#include <fstream>
#include <iostream>
#include <signal.h>
#include <source_location>
#include <sys/file.h>
#include <sys/wait.h>
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
    throw std::runtime_error("research assertion " + std::to_string(l.line()));
}
J rat(const char *n, const char *d = "1") {
  return {{"numerator", n}, {"denominator", d}};
}
int main() try {
  char tmp[] = "/private/tmp/sbv-research-XXXXXX";
  const auto *env = std::getenv("SYMPHONY_SBV_RESEARCH_FIXTURES");
  std::filesystem::path root = env ? env : mkdtemp(tmp);
  std::filesystem::create_directories(root);
  unsigned seq = 0;
  auto path = [&](const std::string &name) { return (root / name).string(); };
  auto write = [&](const std::string &file, const J &j) {
    d::create_file(file, j.dump(), e::unix_time_ms() + 30000);
  };
  auto read = [&](const std::string &file) {
    auto r =
        e::parse_bounded_json(d::read_file(file, e::unix_time_ms() + 30000),
                              d::artifact_bytes, d::artifact_values);
    s::validate_result(r);
    return r;
  };
  auto source = d::base("independent mechanical bootstrap values");
  source["sections"]["values"] = d::section(J::array({"1", "2", "3", "4"}));
  source = s::seal_result(source);
  write(path("source.json"), source);
  J p{{"protocol", "symphony.sbv.resample-input.v1"},
      {"path", path("source.json")},
      {"expected_sha256", source.at("content_sha256")},
      {"output_path", ""},
      {"pointer", "/sections/values/data"},
      {"value_pointer", ""},
      {"value_type", "integer"},
      {"sampling_measure", "uniform_rows"},
      {"scheme", "iid"},
      {"sample_size", "8"},
      {"block_size", "1"},
      {"replicates", "12"},
      {"seed", "18446744073709551615"},
      {"workers", "1"},
      {"retain_indices", true},
      {"studies",
       J::array({"bootstrap_mean_distribution", "bootstrap_mean_quantiles"})},
      {"quantiles", J::array({rat("0"), rat("1", "2"), rat("1")})},
      {"unit", "mechanical units"},
      {"input_description",
       "uniform bootstrap fixture; not predictive market evidence"},
      {"extensions", J::object()}};
  auto call = [&](const char *op, J q) {
    q["output_path"] = path("result-" + std::to_string(++seq) + ".json");
    auto receipt = s::dispatch(op, q, e::unix_time_ms() + 30000);
    auto r = read(q.at("output_path"));
    check(r.at("content_sha256") == receipt.at("content_sha256"));
    return r;
  };
  auto reject = [&](const char *op, J q) {
    q["output_path"] = path("refused-" + std::to_string(++seq) + ".json");
    bool failed = false;
    try {
      s::dispatch(op, q, e::unix_time_ms() + 30000);
    } catch (const std::exception &) {
      failed = true;
    }
    check(failed);
    check(!std::filesystem::exists(q.at("output_path").get<std::string>()));
  };
  auto r = call("resample", p), q = p;
  q["workers"] = "4";
  auto parallel = call("resample", q);
  check(r["sections"]["resamples"] == parallel["sections"]["resamples"]);
  check(r["sections"]["studies"] == parallel["sections"]["studies"]);
  check(parallel["sections"]["resources"]["data"]["actual_workers"] == "4");
  for (const auto &draw : r["sections"]["resamples"]["data"]) {
    unsigned sum = 0;
    for (const auto &idx : draw["source_indices"]) {
      auto n = d::u64(idx);
      check(n < 4);
      sum += n + 1;
    }
    check(w::equal(w::artifact(draw["mean"]), {sum, 8}));
  }
  q = p;
  q["seed"] = "0";
  check(call("resample", q)["sections"]["resamples"] !=
        r["sections"]["resamples"]);
  for (const auto *scheme : {"moving_block", "circular_block"}) {
    q = p;
    q["scheme"] = scheme;
    q["block_size"] = "3";
    auto blocks = call("resample", q);
    for (const auto &draw : blocks["sections"]["resamples"]["data"]) {
      auto indices = draw["source_indices"];
      for (std::size_t start = 0; start < indices.size(); start += 3) {
        auto first = d::u64(indices[start]);
        if (q["scheme"] == "moving_block")
          check(first <= 1);
        for (std::size_t off = 1; off < 3 && start + off < indices.size();
             ++off)
          check(d::u64(indices[start + off]) == (first + off) % 4);
      }
    }
  }
  q = p;
  q["retain_indices"] = false;
  q["studies"] = J::array();
  q["quantiles"] = J::array();
  auto none = call("resample", q);
  check(none["sections"]["studies"]["status"] == "not_selected");
  check(none["sections"]["resamples"]["data"][0]["source_indices"].is_null());
  for (unsigned i = 0; i < 11; ++i) {
    q = p;
    if (i == 0)
      q["replicates"] = "0";
    if (i == 1)
      q["sample_size"] = "0";
    if (i == 2)
      q["block_size"] = "2";
    if (i == 3) {
      q["scheme"] = "moving_block";
      q["block_size"] = "5";
    }
    if (i == 4)
      q["workers"] = "65";
    if (i == 5)
      q["sampling_measure"] = "inferred";
    if (i == 6)
      q["seed"] = "18446744073709551616";
    if (i == 7)
      q["expected_sha256"] = "changed";
    if (i == 8)
      q["value_pointer"] = "/absent";
    if (i == 9) {
      q["replicates"] = "1024";
      q["sample_size"] = "1024";
    }
    if (i == 10)
      q["quantiles"] = J::array({rat("-1")});
    reject("resample", q);
  }
  auto trial = [&](const char *id, J request) {
    request.erase("output_path");
    return J{{"id", id},
             {"state", "ready"},
             {"operation", "resample"},
             {"request", request},
             {"reason", ""},
             {"parameters", {{"seed", request.at("seed")}}},
             {"lineage", J::object()}};
  };
  auto expdir = path("experiment");
  std::filesystem::create_directory(expdir);
  J exp{{"protocol", "symphony.sbv.experiment-input.v1"},
        {"output_path", ""},
        {"directory", expdir},
        {"experiment_id", "mechanical-sweep"},
        {"trials", J::array({trial("a", p),
                             trial("b", p),
                             {{"id", "pruned"},
                              {"state", "pruned"},
                              {"operation", nullptr},
                              {"request", nullptr},
                              {"reason", "caller excluded candidate"},
                              {"parameters", J::object()},
                              {"lineage", J::object()}}})},
        {"workers", "2"},
        {"on_failure", "continue"},
        {"extensions", J::object()}};
  // Identical child results exercise concurrent no-replace staging to distinct
  // destinations.
  r = call("experiment", exp);
  check(r["sections"]["summary"]["data"]["completed"] == "2");
  check(r["sections"]["summary"]["data"]["pruned"] == "1");
  check(r["sections"]["search"]["data"]["executed_this_invocation"] == "2");
  auto child = read(expdir + "/a.result.json");
  check(read(expdir + "/b.result.json") == child);
  auto again = call("experiment", exp);
  check(again["sections"]["search"]["data"]["executed_this_invocation"] == "0");
  check(again["sections"]["search"]["data"]["reused_receipts"] == "2");
  check(read(expdir + "/a.result.json") == child);
  q = exp;
  q["trials"][0]["request"]["seed"] = "123";
  reject("experiment", q);
  q = exp;
  q["workers"] = "16";
  q["trials"][0]["request"]["workers"] = "64";
  reject("experiment", q);
  q = exp;
  q["trials"][0]["operation"] = "experiment";
  reject("experiment", q);
  q = exp;
  q["trials"][0]["id"] = "../escape";
  reject("experiment", q);
  // Same directory lock is held by an independent descriptor.
  int lock = ::open((expdir + "/experiment.lock").c_str(), O_RDWR);
  check(lock >= 0 && ::flock(lock, LOCK_EX | LOCK_NB) == 0);
  reject("experiment", exp);
  ::close(lock);
  // Completed result drift is detected before it can be reused.
  {
    std::ofstream f(expdir + "/a.result.json", std::ios::app);
    f << ' ';
  }
  reject("experiment", exp);
  {
    std::ofstream f(expdir + "/a.result.json", std::ios::trunc);
    f << child.dump();
  }
  auto stopdir = path("stop");
  std::filesystem::create_directory(stopdir);
  q = exp;
  q["directory"] = stopdir;
  q["workers"] = "1";
  q["on_failure"] = "stop";
  q["trials"] = J::array({trial("bad", p), trial("later", p)});
  q["trials"][0]["request"]["value_pointer"] = "/missing";
  r = call("experiment", q);
  check(r["status"] == "partial");
  check(r["sections"]["summary"]["data"]["failed"] == "1");
  check(r["sections"]["summary"]["data"]["not_started"] == "1");
  check(!std::filesystem::exists(stopdir + "/later.claim.json"));
  again = call("experiment", q);
  check(again["sections"]["search"]["data"]["executed_this_invocation"] == "0");
  // Kill a real child after its durable claim and before completion.
  auto killdir = path("killed");
  std::filesystem::create_directory(killdir);
  q = exp;
  q["directory"] = killdir;
  q["workers"] = "1";
  q["trials"] = J::array({trial("interrupted", p)});
  auto &heavy = q["trials"][0]["request"];
  heavy["retain_indices"] = false;
  heavy["replicates"] = "1024";
  heavy["sample_size"] = "1024";
  q["output_path"] = path("killed-summary.json");
  pid_t pid = ::fork();
  check(pid >= 0);
  if (pid == 0) {
    try {
      s::dispatch("experiment", q, e::unix_time_ms() + 30000);
      ::_exit(0);
    } catch (...) {
      ::_exit(2);
    }
  }
  bool claimed = false;
  for (unsigned wait = 0; wait < 10000; ++wait) {
    if (std::filesystem::exists(killdir + "/interrupted.claim.json")) {
      claimed = true;
      break;
    }
    ::usleep(1000);
  }
  check(claimed);
  check(::kill(pid, SIGKILL) == 0);
  int status = 0;
  check(::waitpid(pid, &status, 0) == pid && WIFSIGNALED(status));
  check(!std::filesystem::exists(killdir + "/interrupted.receipt.json"));
  r = call("experiment", q);
  check(r["sections"]["summary"]["data"]["ambiguous"] == "1");
  check(r["sections"]["search"]["data"]["executed_this_invocation"] == "0");
  check(!std::filesystem::exists(killdir + "/interrupted.result.json"));
  // Symlink and unowned-artifact refusal, before starting any child.
  auto unowned = path("unowned");
  std::filesystem::create_directory(unowned);
  std::filesystem::create_symlink(path("source.json"),
                                  unowned + "/a.result.json");
  q = exp;
  q["directory"] = unowned;
  reject("experiment", q);
  check(!std::filesystem::exists(unowned + "/experiment-plan.json"));
  auto collision = path("collision");
  std::filesystem::create_directory(collision);
  q = exp;
  q["directory"] = collision;
  q["output_path"] = collision + "/a.result.json";
  bool collision_refused = false;
  try {
    s::dispatch("experiment", q, e::unix_time_ms() + 30000);
  } catch (const std::exception &) {
    collision_refused = true;
  }
  check(collision_refused);
  check(!std::filesystem::exists(collision + "/experiment-plan.json"));
  if (env) {
    p["output_path"] = path("resample-terminal.json");
    write(path("resample-request.json"), p);
    auto f = exp;
    f["directory"] = path("installed-experiment");
    std::filesystem::create_directory(f["directory"].get<std::string>());
    f["output_path"] = path("experiment-terminal.json");
    write(path("experiment-request.json"), f);
    write(path("bootstrap-native-result.json"), parallel);
  } else
    std::filesystem::remove_all(root);
  std::cout << checks
            << " resampling/experiment assertions passed; real SIGKILL "
               "reconciled without duplicate trial\n";
  return 0;
} catch (const std::exception &e) {
  std::cerr << e.what() << '\n';
  return 1;
}
