#include "installed_campaign.hpp"
using namespace scv_test;
struct TransferCampaign : Campaign {
  J faults = array();
  using Campaign::Campaign;
  J plan(const std::string &name, const J &ids, const fs::path &target = {},
         bool legacy = false) {
    auto inv = qx(name + "-inventory", "inventory",
                  {{"expected_revision", nullptr},
                   {"cursor", nullptr},
                   {"limit", 16}})["connector_result"];
    auto destination = target.empty() ? out / name : target;
    NT_REQUIRE(!fs::exists(destination));
    private_dir(destination);
    auto prefix = legacy ? fs::canonical(args.require("legacy-prefix"))
                         : connector_prefix;
    auto p = qx(name + "-plan", "transfer-plan",
                {{"expected_revision", inv["manifest"]["digest"]},
                 {"operation_ids", ids},
                 {"target",
                  {{"prefix", prefix.string()},
                   {"version", legacy ? "0.1.0-dev" : "0.2.0-dev"},
                   {"root", destination.string()}}},
                 {"capacity",
                  {{"intents", 128}, {"snapshots", 128}}}})["connector_result"];
    return {{"plan", p}, {"expected_plan_digest", p["digest"]}};
  }
  void run() {
    summary["recovery_scope"] =
        "Test-only coordinator SIGKILL at five durable/native boundaries; "
        "exact installed production recovery.";
    auto first = qx("source-import", "import",
                    {{"operation_id", "original"},
                     {"graph", graph},
                     {"query_time", time}})["connector_result"],
         snap = first["intent"]["snapshot"];
    J alias = {{"tops_id", tops},
               {"namespace", namespace_name},
               {"operation_id", "prepared-alias"},
               {"graph", graph},
               {"query_time", time},
               {"owner", snap["owner"]},
               {"connector", snap["connector"]}};
    native("source-prepare-alias", "prepare", alias, snap["connector"]);
    auto request = plan("normal", J::array({"original", "prepared-alias"})),
         before = request["plan"]["manifest"];
    auto result = qx("execute", "transfer", request);
    check(result["status"] == "complete",
          "Transfer completes exact caller selection");
    auto manifest = result["target_manifest"];
    check(manifest["entries"].size() == 2 && manifest["snapshots"].size() == 1,
          "Target shared snapshot and both operation identities retained");
    J states = object();
    for (const auto &e : manifest["entries"])
      states[e["operation_id"].get<std::string>()] = e["state"];
    check(states ==
              J({{"original", "committed"}, {"prepared-alias", "prepared"}}),
          "Prepared source remains prepared");
    check(qx("replay", "transfer", request) == result,
          "Completed exact replay revalidates without new progress");
    fs::path destination =
        request["plan"]["input"]["target_root"].get<std::string>();
    auto status = execute(
        "status", {qxctl.string(), "scv", "graph-index", "transfer-status",
                   "--target-root", destination.string(), "--transfer-digest",
                   result["transfer_digest"].get<std::string>(), "--json"});
    check(status["status"] == "recorded_complete" &&
              status["target_manifest"].is_null(),
          "Journal-only inspection distinguishes recorded completion from "
          "current verification");
    qx("wrong-digest", "transfer",
       nt::updated(request, {{"expected_plan_digest",
                              "sha256:" + std::string(64, '0')}}),
       {{"ok", false}});
    auto source = qx("source-after", "inventory",
                     {{"expected_revision", before["digest"]},
                      {"cursor", nullptr},
                      {"limit", 16}})["connector_result"]["manifest"];
    check(source == before, "Source complete manifest unchanged");
    nt::write(destination / "foreign", "caller data");
    auto rejected = qx("unexpected-target", "transfer-recover", request);
    check(rejected["status"] == "incomplete" &&
              rejected["problem"].get<std::string>().find("unexpected") !=
                  std::string::npos,
          "Unexpected target content prevents replay completion");
    fs::remove(destination / "foreign");
    qx("foreign-operation", "import",
       {{"operation_id", "foreign-operation"},
        {"graph", graph},
        {"query_time", time}},
       {{"root", destination.string()}});
    auto conflict = qx("foreign-operation-reject", "transfer-recover", request);
    check(conflict["status"] == "incomplete" &&
              conflict["problem"].get<std::string>().find("unexpected") !=
                  std::string::npos,
          "Unselected native destination operation prevents completion");
    if (args.has("fault-qxctl"))
      for (const auto phase : {"prepare_pending", "after_prepare",
                               "commit_pending", "after_commit", "complete"}) {
        auto req = plan(std::string("fault-") + phase, J::array({"original"}));
        auto path = out / (std::string("fault-") + phase + "-input.json");
        nt::write(path, nt::canonical(req));
        auto fault_qxctl = fs::canonical(args.require("fault-qxctl"));
        std::vector<std::string> command = {fault_qxctl.string(),
                                            "scv",
                                            "graph-index",
                                            "transfer",
                                            "--connector-prefix",
                                            connector_prefix.string(),
                                            "--connector-version",
                                            "0.2.0-dev",
                                            "--index-root",
                                            root.string(),
                                            "--tops-id",
                                            tops,
                                            "--namespace",
                                            namespace_name,
                                            "--input",
                                            path.string(),
                                            "--json"};
        AsyncProcess process(command, "",
                             {{"SYMPHONY_TRANSFER_TEST_STOP", phase}});
        process.wait_stopped(45);
        process.kill();
        auto exit_code = process.wait();
        auto output = process.out();
        check(
            exit_code == -SIGKILL && output.empty(),
            std::string("Test-only coordinator killed after named boundary: ") +
                phase);
        faults.push_back(
            {{"phase", phase},
             {"observed_stop", true},
             {"exit_code", exit_code},
             {"stdout_bytes", output.size()},
             {"test_cli_sha256", nt::digest(nt::read(fault_qxctl)).substr(7)}});
        nt::write_json(out / "FAULTS.json", faults);
        auto recovered =
            qx(std::string("recover-") + phase, "transfer-recover", req);
        check(recovered["status"] == "complete",
              std::string("Production qxctl recovers exact transfer after ") +
                  phase);
        check(recovered["target_manifest"]["entries"].size() == 1,
              std::string("Recovery retains one exact operation after ") +
                  phase);
      }
    auto stale = plan("stale", J::array({"original"}));
    qx("source-addition", "import",
       {{"operation_id", "later"}, {"graph", graph}, {"query_time", time}});
    qx("stale-reject", "transfer", stale, {{"ok", false}});
    check(
        fs::is_empty(stale["plan"]["input"]["target_root"].get<std::string>()),
        "Stale source rejected before destination journal or index mutation");
    auto owner = clone_owner();
    qx("copied-owner-import", "import",
       {{"operation_id", "copied-owner"},
        {"graph", graph},
        {"query_time", time}},
       {{"owner", owner.string()}});
    auto missing = plan("missing-owner", J::array({"copied-owner"}));
    fs::path executable = missing["plan"]["selected"][0]["source"]["intent"]
                                 ["snapshot"]["owner"]["ExecutablePath"]
                                     .get<std::string>();
    auto hidden = executable;
    hidden.replace_extension(".hidden");
    {
      RestorePath restore(executable, hidden);
      qx("missing-owner-reject", "transfer", missing, {{"ok", false}});
      check(fs::is_empty(
                missing["plan"]["input"]["target_root"].get<std::string>()),
            "Missing original semantic owner prevents reservation");
    }
    auto legacy = plan("legacy-target",
                       J::array({"original", "prepared-alias"}), {}, true),
         legacy_result = qx("legacy-target-execute", "transfer", legacy);
    check(legacy_result["status"] == "complete" &&
              std::all_of(legacy_result["target_manifest"]["entries"].begin(),
                          legacy_result["target_manifest"]["entries"].end(),
                          [](const J &e) {
                            return e["connector"]["Version"] == "0.1.0-dev";
                          }),
          "Exact legacy writer target completes with preserved version "
          "identities");
    summary.update(J{{"status", "passed"},
                     {"fault_cases", faults.size()},
                     {"source_records_deleted", 0},
                     {"scope", "Recoverable target copy only; no retirement, "
                               "graph-head switch or database migration"}});
    save();
  }
};
int main(int argc, char **argv) {
  return nt::test_main([&] {
    nt::Arguments args(argc, argv);
    args.values["connector-version"] = "0.2.0-dev";
    TransferCampaign c(args);
    c.checked_run([&] { c.run(); });
    std::cout << J{{"status", "passed"},
                   {"calls", c.calls.size()},
                   {"assertions", c.assertions.size()},
                   {"faults", c.faults.size()}}
                     .dump()
              << "\n";
  });
}
