#include "../../sqav-databento-dbn-cpp/tests/public_fixture.hpp"
#include <atomic>
#include <chrono>
#include <cstring>
#include <filesystem>
#include <fstream>
#include <iostream>
#include <source_location>
#include <symphony/knowledge/engine/digest.hpp>
#include <symphony/knowledge/engine/error.hpp>
#include <symphony/knowledge/engine/protocol.hpp>
#include <symphony/sbv/sbv.hpp>
#include <symphony/sqav/databento/dbn.hpp>
#include <sys/socket.h>
#include <sys/stat.h>
#include <sys/un.h>
#include <thread>
#include <unistd.h>
namespace e = symphony::knowledge::engine;
namespace s = symphony::sbv;
using J = s::Json;
unsigned checks = 0;
void check(bool ok,
           std::source_location loc = std::source_location::current()) {
  ++checks;
  if (!ok)
    throw std::runtime_error("test failure line " + std::to_string(loc.line()));
}
template <class F> void rejects(F f) {
  bool rejected = false;
  try {
    f();
  } catch (...) {
    rejected = true;
  }
  check(rejected);
}
J call(const std::string &op, const J &p) {
  return s::dispatch(op, p, e::unix_time_ms() + 15000);
}
J read(const std::string &path) {
  std::ifstream f(path);
  J j;
  f >> j;
  return j;
}
void save(const std::string &path, const std::string &bytes) {
  std::ofstream f(path, std::ios::binary);
  f << bytes;
}
J rat(const char *n, const char *d) {
  return {{"numerator", n}, {"denominator", d}};
}
int main(int argc, char **argv) try {
  check(argc == 2);
  char temp[] = "/private/tmp/sbv-contract-XXXXXX";
  check(::mkdtemp(temp) != nullptr);
  const std::string root = temp;
  struct Cleanup {
    std::string root;
    ~Cleanup() { std::filesystem::remove_all(root); }
  } cleanup{root};
  auto caps = call("capabilities",
                   {{"protocol", "symphony.sbv.capabilities-input.v1"}});
  check(caps.at("cpu").at("available") == true);
  check(caps.at("cuda").at("available") == false);
  check(caps.at("live").at("available") == false);
  auto original = public_fixture::fixture();
  std::vector<unsigned char> bytes(original.begin(), original.begin() + 360);
  auto put = [&](std::size_t offset, std::uint64_t n, unsigned length) {
    for (unsigned i = 0; i < length; ++i)
      bytes[offset + i] = static_cast<unsigned char>(n >> (8 * i));
  };
  for (unsigned i = 0; i < 4; ++i) {
    const auto off = bytes.size();
    bytes.insert(bytes.end(), original.begin() + 360, original.begin() + 416);
    put(off + 8, 1609160400000000000ULL + i * 100, 8);
    put(off + 40, 1609160400000000000ULL + i * 100, 8);
    put(off + 24, (100 + i) * 1000000000ULL, 8);
    bytes[off + 38] = 'T';
  }
  const std::string source(reinterpret_cast<const char *>(bytes.data()),
                           bytes.size());
  save(root + "/sample.dbn", source);
  J p{{"protocol", "symphony.sbv.run-input.v1"},
      {"source_path", root + "/sample.dbn"},
      {"source_sha256", e::sha256_hex(source)},
      {"dataset", "GLBX.MDP3"},
      {"output_path", root + "/serial.json"},
      {"criteria",
       {{"rule", "spaced_trades"},
        {"spacing_ns", "100"},
        {"min_trade_size", "1"},
        {"direction", "any"},
        {"max_signals", "4"}}},
      {"replay",
       {{"before_ns", "100"}, {"after_ns", "100"}, {"retain_events", true}}},
      {"execution",
       {{"model", "touch_observation"},
        {"horizon_ns", "100"},
        {"price_offsets_nanos", J::array({"1000000000", "-1000000000"})},
        {"probability_numerator", "0"},
        {"probability_denominator", "1"}}},
      {"studies", J::array({"signal_summary", "forward_markout"})},
      {"workers", "1"},
      {"extensions",
       {{"user/key~name",
         J::array({"9007199254740993", true, "\x1b[31m", J::object()})}}}};
  const auto baseline = call("run", p);
  const auto scientific = read(root + "/serial.json");
  J load{{"protocol", "symphony.sbv.dataset-load-input.v1"},
         {"directory", root},
         {"instance_id", "00000000000000000000000000000001"},
         {"source_path", p["source_path"]},
         {"source_sha256", p["source_sha256"]},
         {"dataset", p["dataset"]},
         {"memory_budget_bytes", "1048576"},
         {"residency", "pageable"},
         {"max_concurrent_jobs", "4"},
         {"worker_budget", "4"},
         {"idle_timeout_ms", "60000"}};
  J ref{{"protocol", "symphony.sbv.dataset-inspect-input.v1"},
        {"directory", root},
        {"instance_id", load["instance_id"]}};
  const auto loaded = call("dataset_load", load);
  check(loaded["source_reads"] == "1" && loaded["decode_passes"] == "1" &&
        loaded["events"] == "4");
  std::vector<std::jthread> callers;
  std::atomic_bool finished{false};
  std::string failure;
  // Stop before joining callers even if a test assertion throws.
  struct Stop {
    J ref;
    ~Stop() {
      ref["protocol"] = "symphony.sbv.dataset-stop-input.v1";
      try { call("dataset_stop", ref); } catch (...) {}
    }
  } stop{ref};
  auto request = [&](const std::string &op) {
    auto v = ref;
    auto slug = op;
    for (auto &c : slug) if (c == '_') c = '-';
    v["protocol"] = "symphony.sbv." + slug + "-input.v1";
    return v;
  };
  auto jobs = request("dataset_jobs");
  jobs.update({{"job_id", nullptr}, {"after_sequence", "0"}, {"limit", "1"}});
  auto settings = request("dataset_configure");
  settings.update({{"expected_generation", "0"}, {"max_concurrent_jobs", nullptr},
                   {"worker_budget", "18446744073709551615"}, {"idle_timeout_ms", "0"},
                   {"on_active_over_limit", "reject"}});
  auto configured = call("dataset_configure", settings);
  check(configured["configuration_generation"] == "1" && configured["worker_budget"] == "18446744073709551615");
  check(configured["source_reads"] == "1" && configured["decode_passes"] == "1");
  rejects([&] { call("dataset_configure", settings); });
  check(call("dataset_inspect", ref)["configuration_generation"] == "1");
  // A partially framed connection cannot block other controls; no frame timeout.
  const auto socket_path = root + "/" + load["instance_id"].get<std::string>() + ".sock";
  int stalled = ::socket(AF_UNIX, SOCK_STREAM, 0);
  check(stalled >= 0);
  sockaddr_un address{};
  address.sun_family = AF_UNIX;
  std::strcpy(address.sun_path, socket_path.c_str());
  check(::connect(stalled, reinterpret_cast<sockaddr *>(&address), sizeof(address)) == 0);
  char partial = 0;
  check(::send(stalled, &partial, 1, 0) == 1);
  check(call("dataset_inspect", ref)["state"] == "ready");
  check(call("dataset_jobs", jobs)["jobs"].empty());
  auto child = p;
  child.erase("output_path");
  auto job = request("dataset_execute");
  job.update({{"job_id", "finished"}, {"operation", "run"}, {"request", child},
              {"output_path", root + "/done.json"}});
  std::filesystem::remove(root + "/sample.dbn");
  auto invalid_output = job; invalid_output["output_path"] = "relative.json";
  rejects([&] { call("dataset_execute", invalid_output); });
  call("dataset_execute", job);
  rejects([&] { call("dataset_execute", job); });
  auto first = call("dataset_jobs", jobs);
  check(first["jobs"].size() == 1 && first["jobs"][0]["state"] == "completed" && first["next_after_sequence"] == "1");
  const auto library = std::filesystem::canonical(argv[1]).string();
  std::ifstream lib(library, std::ios::binary);
  const std::string libbytes((std::istreambuf_iterator<char>(lib)), {});
  J provider{{"protocol", "symphony.sbv.native-provider-selection.v1"},
     {"library", {{"path", library}, {"expected_sha256", e::sha256_hex(libbytes)}}},
     {"id", "sbv-test-provider"}, {"version", "1"}, {"role", "model"},
     {"input_profile", "symphony.sbv.provider-databento-mbo-event.v1"},
     {"concurrency", "per_worker_instances"}, {"parameters", {{"mode", "cancel_poll"}}},
     {"dependencies", {{"capture", "uncaptured"}, {"description", "test fixture"}, {"artifacts", J::array()}}},
     {"extensions", J::object()}};
  auto artifact = read(root + "/done.json");
  J evaluation{{"protocol", "symphony.sbv.evaluate-input.v1"},
      {"source_path", p["source_path"]}, {"source_sha256", p["source_sha256"]}, {"dataset", p["dataset"]},
      {"census", {{"path", root + "/done.json"}, {"expected_sha256", artifact["content_sha256"]}, {"pointer", ""}}},
      {"model", {{"protocol", "symphony.sbv.model-selection.v1"}, {"id", "native_provider"}, {"version", "1"}, {"horizon_ns", "100"},
       {"parameters", {{"provider", provider}, {"measure", "probability"}, {"conditioning", "execution"}, {"calibration_reference", ""}, {"on_unavailable", "reject"}}}}},
      {"replay", p["replay"]}, {"studies", J::array({"signal_summary", "model_summary"})},
      {"workers", "4"}, {"extensions", J::object()}};
  job.update({{"job_id", "cooperative"}, {"operation", "evaluate"}, {"request", evaluation}, {"output_path", root + "/cancelled.json"}});
  auto start = [&](const J &input) {
    finished = false;
    failure.clear();
    callers.emplace_back([&, input] {
      try { call("dataset_execute", input); }
      catch (const e::Error &e) { failure = e.code(); }
      catch (...) { failure = "other"; }
      finished.store(true);
    });
  };
  auto wait_active = [&] {
    for (unsigned i = 0; i < 500; ++i) {
      if (call("dataset_inspect", ref)["active_jobs"] == "1") return;
      if (finished) throw std::runtime_error("job ended before observation: " + failure);
      std::this_thread::sleep_for(std::chrono::milliseconds(10));
    }
    throw std::runtime_error("test watchdog: job not active");
  };
  start(job);
  wait_active();
  // Give all provider worker callbacks time to enter their cancellation loops.
  std::this_thread::sleep_for(std::chrono::milliseconds(100));
  rejects([&] { call("dataset_release", request("dataset_release")); });
  settings.update({{"expected_generation", "1"}, {"worker_budget", "1"}, {"max_concurrent_jobs", "1"}});
  rejects([&] { call("dataset_configure", settings); });
  settings["on_active_over_limit"] = "drain";
  check(call("dataset_configure", settings)["configuration_generation"] == "2");
  auto another = job;
  another.update({{"job_id", "blocked"}, {"output_path", root + "/blocked.json"}});
  rejects([&] { call("dataset_execute", another); });
  auto page = call("dataset_jobs", jobs);
  check(page["more"] == true && page["highest_sequence"] == "2");
  jobs["after_sequence"] = page["next_after_sequence"];
  page = call("dataset_jobs", jobs);
  check(page["more"] == false && page["jobs"][0]["job_id"] == "cooperative");
  auto cancel = request("dataset_cancel"); cancel["job_id"] = "missing";
  rejects([&] { call("dataset_cancel", cancel); });
  cancel["job_id"] = "cooperative";
  check(call("dataset_cancel", cancel)["job"]["cancellation_requested"] == true);
  for (unsigned i = 0; i < 500 && !finished; ++i)
    std::this_thread::sleep_for(std::chrono::milliseconds(10));
  check(finished && failure == "sbv.cancelled");
  callers.clear();
  check(call("dataset_cancel", cancel)["job"]["state"] == "cancelled");
  check(!std::filesystem::exists(root + "/cancelled.json"));
  jobs.update({{"job_id", "finished"}, {"after_sequence", "0"}, {"limit", "1"}});
  check(call("dataset_jobs", jobs)["jobs"][0]["state"] == "completed");
  // Raising admission is immediate and does not reload the deleted source.
  settings.update({{"expected_generation", "2"}, {"worker_budget", nullptr}, {"max_concurrent_jobs", nullptr}});
  check(call("dataset_configure", settings)["configuration_generation"] == "3");
  job.update({{"job_id", "noncooperative"}, {"output_path", root + "/force-stopped.json"}});
  job["request"]["model"]["parameters"]["provider"]["parameters"]["mode"] = "ignore_cancel";
  start(job);
  wait_active();
  std::this_thread::sleep_for(std::chrono::milliseconds(100));
  cancel["job_id"] = "noncooperative";
  check(call("dataset_cancel", cancel)["job"]["state"] == "running");
  std::this_thread::sleep_for(std::chrono::milliseconds(100));
  check(!finished);
  auto stopped = call("dataset_stop", request("dataset_stop"));
  check(stopped["state"] == "termination_requested" && stopped["results_may_be_incomplete"] == true && stopped["active_jobs"] == "1");
  for (unsigned i = 0; i < 500 && !finished; ++i)
    std::this_thread::sleep_for(std::chrono::milliseconds(10));
  check(finished && failure == "sbv.resident_transport");
  callers.clear();
  ::close(stalled);
  check(!std::filesystem::exists(socket_path));
  rejects([&] { call("dataset_inspect", ref); });
  rejects([&] { call("dataset_load", load); });
  std::cout << checks << " resident control checks passed\n";
} catch (const std::exception &e) {
  std::cerr << e.what() << '\n';
  return 1;
}
