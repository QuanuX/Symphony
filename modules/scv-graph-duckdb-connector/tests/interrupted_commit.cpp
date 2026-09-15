#include "installed_campaign.hpp"
#include <poll.h>
using namespace scv_test;
struct InterruptedCommitCampaign : Campaign {
  J faults = array(), cases = array();
  fs::path fault_engine;
  explicit InterruptedCommitCampaign(const nt::Arguments &args)
      : Campaign(args),
        fault_engine(fs::canonical(args.require("fault-engine"))) {
    summary.update(
        J{{"recovery_scope",
           "SIGKILL at deterministic barriers in the connector's own "
           "prepare/commit implementation; production qxctl recovery."},
          {"fault_executable", fault_engine.string()},
          {"fault_executable_sha256",
           nt::digest(nt::read(fault_engine)).substr(7)},
          {"fault_executable_installed", false},
          {"power_loss_tested", false}});
    private_dir(out / "faults");
    save();
  }
  void interrupt(const std::string &name, const std::string &stage,
                 const std::string &operation, const J &payload) {
    auto request = nt::request("symphony-scv-graph-duckdb-connector", operation,
                               payload, uuid());
    request["correlation_id"] = uuid();
    auto prefix = out / "faults" / name;
    nt::write(prefix.string() + ".request.json", nt::canonical(request));
    int descriptors[2];
    NT_REQUIRE(::pipe(descriptors) == 0);
    try {
      AsyncProcess process(
          {fault_engine.string()}, nt::canonical(request),
          {{"SYMPHONY_SCV_TEST_BARRIER", stage},
           {"SYMPHONY_SCV_TEST_BARRIER_FD", std::to_string(descriptors[1])}},
          root);
      ::close(descriptors[1]);
      descriptors[1] = -1;
      auto deadline =
          std::chrono::steady_clock::now() + std::chrono::seconds(20);
      std::string marker;
      while (!marker.ends_with('\n')) {
        auto remaining = std::chrono::duration_cast<std::chrono::milliseconds>(
                             deadline - std::chrono::steady_clock::now())
                             .count();
        NT_REQUIRE(remaining > 0);
        struct pollfd fd{descriptors[0], POLLIN, 0};
        int ready;
        do {
          ready = ::poll(&fd, 1, static_cast<int>(remaining));
        } while (ready < 0 && errno == EINTR);
        nt::require(ready > 0, name + ": barrier timeout");
        char block[256];
        ssize_t count;
        do {
          count = ::read(descriptors[0], block, sizeof(block));
        } while (count < 0 && errno == EINTR);
        nt::require(count > 0, name + ": writer exited before barrier");
        marker.append(block, static_cast<std::size_t>(count));
        NT_REQUIRE(marker.size() <= 128);
      }
      NT_REQUIRE(marker == stage + "\n");
      process.wait_stopped(20, SIGSTOP);
      process.kill();
      auto exit_code = process.wait(5);
      auto stdout_text = process.out(), stderr_text = process.err();
      nt::write(prefix.string() + ".stdout", stdout_text);
      nt::write(prefix.string() + ".stderr", stderr_text);
      check(
          exit_code == -SIGKILL && stdout_text.empty(),
          name +
              ": stopped writer killed without a response or graceful cleanup");
      faults.push_back(
          {{"name", name},
           {"stage", stage},
           {"operation", operation},
           {"pid", process.pid()},
           {"stop_signal", "SIGSTOP"},
           {"exit_code", exit_code},
           {"marker", stage},
           {"response_bytes", stdout_text.size()},
           {"request",
            fs::relative(prefix.string() + ".request.json", out).string()},
           {"writer_reaped", true}});
      nt::write(out / "FAULTS.json", nt::canonical(faults) + "\n");
    } catch (...) {
      ::close(descriptors[0]);
      if (descriptors[1] >= 0)
        ::close(descriptors[1]);
      throw;
    }
    ::close(descriptors[0]);
    if (descriptors[1] >= 0)
      ::close(descriptors[1]);
  }
  void one_case(const std::string &stage, bool shared = false) {
    auto name = nt::replace(stage, ".", "-") + (shared ? "-shared" : "");
    root = out / name;
    private_dir(root);
    namespace_name = name + "-sentinel";
    auto sentinel = qx(name + "-sentinel", "import",
                       {{"operation_id", "sentinel"},
                        {"graph", graph},
                        {"query_time", time}})["connector_result"],
         original = sentinel["intent"]["snapshot"];
    if (!shared)
      namespace_name = name + "-target";
    J payload = {{"tops_id", tops},
                 {"namespace", namespace_name},
                 {"operation_id", "target"},
                 {"graph", graph},
                 {"query_time", time},
                 {"owner", original["owner"]},
                 {"connector", original["connector"]}};
    auto snapshot =
        nt::seal(nt::updated(original, {{"namespace", namespace_name}}));
    auto intent = nt::seal({{"protocol", "symphony.scv.graph-index-intent.v1"},
                            {"operation_id", "target"},
                            {"snapshot", snapshot},
                            {"validation_query_time", time}});
    J fault_payload;
    std::string operation;
    if (stage.starts_with("commit.")) {
      auto prepared =
          native(name + "-prepare", "prepare", payload, original["connector"]);
      check(prepared["state"] == "prepared" && prepared["intent"] == intent,
            name + ": exact intent durably prepared before faulted commit");
      fault_payload =
          nt::updated(pick(payload, {"tops_id", "namespace", "operation_id"}),
                      {{"expected_intent_digest", intent["digest"]}});
      operation = "commit";
    } else {
      fault_payload = payload;
      operation = "prepare";
    }
    interrupt(name, stage, operation, fault_payload);
    bool absent = stage == "prepare.before_commit",
         committed = stage == "commit.after_commit";
    auto observed =
        qx(name + "-status__target", "status", nullptr, {{"ok", !absent}});
    if (!absent) {
      auto status = observed["connector_result"];
      check(status["state"] == (committed ? "committed" : "prepared") &&
                status["intent"] == intent &&
                status["index_verified"] == committed,
            name + ": recovery observation matches the durable transaction "
                   "boundary");
    }
    bool visible = shared || committed;
    J export_input = {{"snapshot_digest", snapshot["digest"]},
                      {"query_time", time}};
    auto observed_export =
        qx(name + "-export-before", "export", export_input, {{"ok", visible}});
    if (visible)
      check(observed_export["connector_result"]["snapshot"] == snapshot,
            name + ": only the complete expected snapshot is visible before "
                   "recovery");
    auto sentinel_export =
        qx(name + "-sentinel-after", "export",
           {{"snapshot_digest", original["digest"]}, {"query_time", time}},
           {{"namespace", original["namespace"]}});
    check(sentinel_export["connector_result"]["snapshot"] == original,
          name + ": earlier committed snapshot preserved in full");
    J recovered;
    if (absent) {
      qx(name + "-missing-recover__target", "recover", nullptr,
         {{"ok", false}});
      recovered = qx(
          name + "-reimport", "import",
          {{"operation_id", "target"}, {"graph", graph}, {"query_time", time}});
    } else
      recovered = qx(name + "-recover__target", "recover");
    auto result = recovered["connector_result"];
    check(result["state"] == "committed" && result["index_verified"] == true &&
              result["intent"] == intent &&
              recovered["owner_evaluation"]["graph_digest"] == graph["digest"],
          name + ": installed qxctl recovers exact intent with original "
                 "semantic owner");
    auto repeated = qx(name + "-repeat__target", "recover");
    check(!absent
              ? repeated == nt::updated(recovered, {{"operation", "recover"}})
              : (repeated["connector_result"] == result &&
                 repeated["owner_evaluation"] == recovered["owner_evaluation"]),
          name + ": recovery retry preserves the complete logical result");
    auto exported = qx(name + "-export-after", "export", export_input);
    check(exported["connector_result"]["snapshot"] == snapshot &&
              exported["connector_result"]["counts"] == sentinel["counts"] &&
              exported["connector_result"]["projection_digest"] ==
                  sentinel["projection_digest"],
          name + ": complete row inventory and graph identity preserved after "
                 "recovery");
    native(name + "-stale-commit", "commit",
           {{"tops_id", tops},
            {"namespace", namespace_name},
            {"operation_id", "target"},
            {"expected_intent_digest", "sha256:" + std::string(64, '0')}},
           original["connector"], false);
    cases.push_back(
        {{"name", name},
         {"stage", stage},
         {"shared_snapshot", shared},
         {"observed_state",
          absent ? "absent" : (committed ? "committed" : "prepared")},
         {"recovered_snapshot_digest", snapshot["digest"]},
         {"status", "passed"}});
    nt::write(out / "CASES.json", nt::canonical(cases) + "\n");
  }
  void run() {
    for (const auto stage : {"prepare.before_commit", "prepare.after_commit",
                             "commit.after_snapshot", "commit.after_row",
                             "commit.before_commit", "commit.after_commit"})
      one_case(stage);
    for (const auto stage : {"commit.before_commit", "commit.after_commit"})
      one_case(stage, true);
    summary.update(J{{"status", "passed"},
                     {"interruption_cases", cases.size()},
                     {"faulted_writers_reaped", faults.size()}});
    save();
  }
};
int main(int argc, char **argv) {
  return nt::test_main([&] {
    nt::Arguments args(argc, argv);
    InterruptedCommitCampaign campaign(args);
    campaign.checked_run([&] { campaign.run(); });
    std::cout << J{{"status", "passed"},
                   {"cases", campaign.cases.size()},
                   {"calls", campaign.calls.size()},
                   {"assertions", campaign.assertions.size()}}
                     .dump()
              << "\n";
  });
}
