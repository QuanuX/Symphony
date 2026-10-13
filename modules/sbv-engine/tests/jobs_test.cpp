#include "../src/experiment_result.hpp"
#include <csignal>
#include <fstream>
#include <iostream>
#include <source_location>
#include <sys/stat.h>
#include <sys/wait.h>
#include <thread>
#include <unistd.h>
namespace s = symphony::sbv;
namespace d = s::detail;
namespace e = symphony::knowledge::engine;
using J = s::Json;
unsigned checks = 0;
void check(bool ok, std::source_location at = std::source_location::current()) {
  ++checks;
  if (!ok)
    throw std::runtime_error("job assertion line " + std::to_string(at.line()));
}
J read(const std::string &path) {
  std::ifstream f(path);
  J j;
  f >> j;
  return j;
}
void write(const std::string &path, const J &j) {
  std::ofstream f(path);
  f << j.dump();
  check(f.good());
}
J invoke(const std::string &op, J p) {
  p["protocol"] =
      d::trial::protocol(op).substr(0, d::trial::protocol(op).size() - 3) +
      "-input.v1";
  return s::dispatch(op, p, e::no_deadline);
}
void refused(const std::string &op, J p) {
  bool caught = false;
  try {
    (void)invoke(op, p);
  } catch (const std::exception &) {
    caught = true;
  }
  check(caught);
}
struct Child {
  pid_t pid;
  explicit Child(const J &p) : pid(::fork()) {
    check(pid >= 0);
    if (pid == 0) {
      try {
        (void)invoke("job_execute", p);
        _exit(0);
      } catch (const std::exception &x) {
        std::cerr << x.what() << '\n';
        _exit(2);
      }
    }
  }
  ~Child() {
    if (pid > 0) {
      ::kill(pid, SIGKILL);
      ::waitpid(pid, nullptr, 0);
    }
  }
  void wait(bool killed = false) {
    int status = 0;
    check(::waitpid(pid, &status, 0) == pid);
    pid = 0;
    check(killed ? WIFSIGNALED(status)
                 : WIFEXITED(status) && WEXITSTATUS(status) == 0);
  }
};
int main(int argc, char **argv) try {
  check(argc == 3);
  char temp[] = "/private/tmp/sbv-jobs-XXXXXX";
  check(mkdtemp(temp));
  std::string root = temp;
  struct Cleanup {
    std::string path;
    ~Cleanup() { std::filesystem::remove_all(path); }
  } cleanup{root};
  if (auto retained = getenv("SBV_JOB_FIXTURE_ROOT")) {
    root = retained;
    std::filesystem::create_directories(root);
    root = std::filesystem::canonical(root).string();
  }
  auto templates = read(argv[1]);
  unsigned serial = 0;
  auto create = [&](J child, J deadline = nullptr) {
    auto id = "job-" + std::to_string(serial++);
    auto directory = root + "/" + id;
    std::filesystem::create_directory(directory);
    check(::chmod(directory.c_str(), 0700) == 0);
    if (child.contains("output")) {
      child["output"]["bundle_path"] = root + "/" + id + "-bundle";
      child["output"]["workspace_path"] = root + "/" + id + "-workspace";
    } else
      child["output_path"] = root + "/" + id + "-result.json";
    J p{{"directory", directory},
        {"job_id", id},
        {"operation", "account_simulate"},
        {"request", child},
        {"deadline_unix_ms", deadline},
        {"read_options", {{"max_page_bytes", nullptr}, {"cache_bytes", "0"}}},
        {"extensions", J::object()}};
    auto receipt = invoke("job_create", p);
    check(receipt["phase"] == "queued" && receipt["generation"] == "0" &&
          receipt["admitted"] == false);
    check(invoke("job_create", p) == receipt);
    auto changed = p;
    changed["extensions"]["different"] = true;
    refused("job_create", changed);
    J address{{"directory", directory},
              {"job_id", id},
              {"expected_plan_sha256", receipt["plan_sha256"]}};
    return std::pair{p, address};
  };
  auto [plan, address] = create(templates["templates"]["account_simulate"]);
  auto wrong = address;
  wrong["expected_plan_sha256"] = std::string(64, '0');
  refused("job_execute", wrong);
  wrong = address;
  wrong["job_id"] = "other";
  refused("job_inspect", wrong);
  auto queued = read(address["directory"].get<std::string>() + "/state.json");
  auto done = invoke("job_execute", address);
  check(done["phase"] == "completed" && done["admitted"] == true &&
        done["automatic_retry"] == false);
  check(invoke("job_execute", address) == done);
  check(invoke("job_create", plan)["result_receipt"] == done["result_receipt"]);
  auto result = d::trial::verify(done["result_receipt"], plan["request"],
                                 "account_simulate", plan["read_options"],
                                 e::no_deadline);
  check(result.at("sections")
            .at("summary")
            .at("data")
            .at("events")
            .materialize() == "5");
  auto recon = address;
  recon["expected_generation"] = "0";
  refused("job_reconcile", recon);
  // Deterministic crash-image: outcome durable, state head still at queued.
  write(address["directory"].get<std::string>() + "/state.json", queued);
  auto recovered = invoke("job_reconcile", recon);
  check(recovered["phase"] == "completed" &&
        recovered["result_receipt"] == done["result_receipt"]);
  recon["expected_generation"] = recovered["generation"];
  check(invoke("job_reconcile", recon) == recovered);
  // Stable saved state reference remains readable after head replacement.
  auto old = s::dispatch(
      "result_inspect",
      {{"protocol", "symphony.sbv.result-inspect-input.v1"},
       {"path", done["state_reference"]["path"]},
       {"expected_sha256", done["state_reference"]["expected_sha256"]}},
      e::no_deadline);
  check(old["content_sha256"] == done["state_reference"]["expected_sha256"]);
  auto altered = read(plan["request"]["output_path"]);
  altered["origin"] = "altered";
  write(plan["request"]["output_path"], altered);
  refused("job_reconcile", recon);

  auto [p2, a2] =
      create(templates["variants"]["account_simulate"]["partitioned"]);
  auto partitioned = invoke("job_execute", a2);
  check(partitioned["phase"] == "completed" &&
        partitioned["result_receipt"].contains("storage"));
  auto logical =
      d::trial::verify(partitioned["result_receipt"], p2["request"],
                       "account_simulate", p2["read_options"], e::no_deadline);
  check(logical.at("sections")
            .at("summary")
            .at("data")
            .at("events")
            .materialize() == "5");
  auto [p3, a3] = create(templates["templates"]["account_simulate"]);
  auto cancel = a3;
  cancel.update({{"expected_generation", "1"},
                 {"request_id", "cancel-1"},
                 {"reason", "test"}});
  refused("job_cancel", cancel);
  cancel["expected_generation"] = "0";
  auto cancelled = invoke("job_cancel", cancel);
  check(cancelled["phase"] == "cancelled" && cancelled["admitted"] == false &&
        cancelled["cancellation_observed"] == true);
  check(invoke("job_cancel", cancel) == cancelled);
  check(invoke("job_execute", a3)["phase"] == "cancelled");
  check(!std::filesystem::exists(
      p3["request"]["output_path"].get<std::string>()));
  cancel["reason"] = "changed";
  refused("job_cancel", cancel);
  auto [p4, a4] = create(templates["templates"]["account_simulate"], "1");
  auto expired = invoke("job_execute", a4);
  check(expired["phase"] == "failed" &&
        expired["error_code"] == "deadline.exceeded");
  check(invoke("job_execute", a4) == expired);
  auto malformed = templates["templates"]["account_simulate"];
  malformed["mode"] = "invalid";
  auto [p5, a5] = create(malformed);
  check(invoke("job_execute", a5)["phase"] == "failed");

  auto barrier = templates["variants"]["account_simulate"]["native_feedback"];
  std::ifstream lib(argv[2], std::ios::binary);
  const std::string binary{std::istreambuf_iterator<char>(lib), {}};
  barrier["feedback"]["provider"]["library"] = {
      {"path", std::filesystem::canonical(argv[2]).string()},
      {"expected_sha256", e::sha256_hex(binary)}};
  auto run_barrier = [&](bool kill_worker, bool cancel_worker) {
    const auto marker = root + "/entered-" + std::to_string(serial);
    const auto release = root + "/release-" + std::to_string(serial);
    barrier["streams"][0]["events"][0]["context"] = {
        {"mode", "job_barrier"}, {"entered", marker}, {"release", release}};
    auto [p, a] = create(barrier);
    Child child(a);
    const auto until =
        std::chrono::steady_clock::now() + std::chrono::seconds(15);
    while (!std::filesystem::exists(marker) &&
           std::chrono::steady_clock::now() < until)
      std::this_thread::sleep_for(std::chrono::milliseconds(2));
    check(std::filesystem::exists(marker));
    auto active = invoke("job_inspect", a);
    check(active["phase"] == "running" &&
          active["execution_lock_held"] == true && active["admitted"] == true);
    // A second process cannot dispatch an admitted job, even while the first
    // waits.
    Child duplicate(a);
    duplicate.wait();
    auto r = a;
    r["expected_generation"] = active["generation"];
    auto observed_active = invoke("job_reconcile", r);
    observed_active["protocol"] = active["protocol"];
    check(observed_active == active);
    if (cancel_worker) {
      r.update({{"request_id", "cancel-running"}, {"reason", "stop callback"}});
      auto requested = invoke("job_cancel", r);
      check(requested["phase"] == "running" &&
            requested["cancellation_requested"] == true &&
            requested["cancellation_observed"] == false);
    }
    if (kill_worker) {
      check(::kill(child.pid, SIGKILL) == 0);
      child.wait(true);
    } else {
      if (!cancel_worker)
        write(release, J::object());
      child.wait();
    }
    auto observed = invoke("job_inspect", a);
    check(observed["execution_lock_held"] == false);
    if (kill_worker) {
      check(observed["phase"] == "running");
      r = a;
      r["expected_generation"] = observed["generation"];
      observed = invoke("job_reconcile", r);
      check(observed["phase"] == "unknown" &&
            observed["outcome_reference"].is_null());
    } else if (cancel_worker) {
      check(observed["phase"] == "cancelled" &&
            observed["cancellation_observed"] == true);
    } else
      check(observed["phase"] == "completed");
    auto repeated = invoke("job_execute", a);
    repeated["protocol"] = observed["protocol"];
    check(repeated == observed);
    if (kill_worker || cancel_worker)
      check(!std::filesystem::exists(
          p["request"]["output_path"].get<std::string>()));
  };
  run_barrier(true, false);
  run_barrier(false, true);
  run_barrier(false, false);
  barrier["feedback"]["on_unavailable"] = "skip_trigger";
  barrier["streams"][0]["events"][0]["context"] = {{"mode", "unavailable"}};
  auto [partial_plan, partial_address] = create(barrier);
  auto partial = invoke("job_execute", partial_address);
  check(partial["phase"] == "completed_partial" &&
        partial["result_receipt"]["status"] == "partial");
  auto partial_reconcile = partial_address;
  partial_reconcile["expected_generation"] = partial["generation"];
  check(invoke("job_reconcile", partial_reconcile)["phase"] ==
        "completed_partial");
  // No-follow and owner-private admission also cover lock files.
  auto [p6, a6] = create(templates["templates"]["account_simulate"]);
  auto dir = a6["directory"].get<std::string>();
  auto existing = dir + "/execution.lock";
  std::filesystem::remove(existing);
  std::filesystem::create_symlink(root + "/target", existing);
  refused("job_execute", a6);
  check(!std::filesystem::exists(root + "/target"));
  auto [p7, a7] = create(templates["templates"]["account_simulate"]);
  check(::chmod(a7["directory"].get<std::string>().c_str(), 0755) == 0);
  refused("job_inspect", a7);
  write(root + "/REPORT.json",
        {{"checks", std::to_string(checks)},
         {"scope", "local durable jobs; synthetic account inputs"},
         {"partitioned", partitioned}});
  std::cout << checks << " durable job assertions passed\n";
  return 0;
} catch (const std::exception &x) {
  std::cerr << x.what() << '\n';
  return 1;
}
