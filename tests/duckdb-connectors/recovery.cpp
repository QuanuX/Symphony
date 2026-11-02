#include "connector_test_support.hpp"
#include <poll.h>
using namespace scv_test;
J shv_graph() {
  auto artifact = nt::seal({{"protocol", "caller.example.v1"}, {"value", "retained"}});
  return nt::seal({{"protocol", "symphony.graph.exchange.v1"},
    {"owner", {{"engine_id", "caller-owner"}, {"engine_version", "1"},
      {"artifact_protocol", artifact["protocol"]}, {"artifact_digest", artifact["digest"]}}},
    {"owner_artifact", artifact},
    {"nodes", J::array({{{"id", "one"}, {"labels", {"component"}}, {"properties", J::object()}}})},
    {"edges", J::array()}});
}
int main(int argc, char **argv) {
  return nt::test_main([&] {
    nt::Arguments args(argc, argv);
    const auto domain = args.require("domain"), version = args.require("version");
    NT_REQUIRE(domain == "scv" || domain == "shv");
    const auto engine = fs::canonical(args.require("engine")), fault = fs::canonical(args.require("fault-engine"));
    std::size_t cases = 0;
    for (const std::string stage : {"prepare.before_commit", "prepare.after_commit", "commit.after_snapshot",
                                  "commit.after_row", "commit.before_commit", "commit.after_commit"}) {
      nt::TempDir temp("duckdb-recovery-");
      Config config; config.version = version;
      ConnectorTests scv(config);
      J base = scv.base;
      if (domain == "shv") {
        base.erase("owner"); base.erase("query_time");
        base["graph"] = shv_graph();
        base["connector"] = installation("shv-graph-duckdb-connector", "shv-graph-duckdb-connector", version);
      }
      base["operation_id"] = "sentinel";
      const auto engine_id = "symphony-" + domain + "-graph-duckdb-connector";
      auto call = [&](const std::string &op, const J &payload, bool okay = true) {
        auto process = nt::run({engine.string()}, nt::canonical(nt::request(engine_id, op, payload)), 15, {}, temp.path);
        auto response = J::parse(process.stdout_text);
        nt::require((process.returncode == 0) == okay && (response["outcome"] == "ok") == okay, response.dump());
        return response["result"];
      };
      auto key = pick(base, {"tops_id", "namespace", "operation_id"});
      auto prepared = call("prepare", base);
      auto sentinel = call("commit", nt::updated(key, {{"expected_intent_digest", prepared["intent"]["digest"]}}));
      base["namespace"] = "target"; base["operation_id"] = "target";
      const auto target_key = pick(base, {"tops_id", "namespace", "operation_id"});
      J payload = base;
      std::string op = "prepare";
      if (stage.starts_with("commit.")) {
        prepared = call("prepare", base);
        payload = nt::updated(target_key, {{"expected_intent_digest", prepared["intent"]["digest"]}});
        op = "commit";
      }
      int descriptors[2]; NT_REQUIRE(::pipe(descriptors) == 0);
      const auto environment = "SYMPHONY_" + std::string(domain == "scv" ? "SCV" : "SHV") + "_TEST_BARRIER";
      try {
        AsyncProcess child({fault.string()}, nt::canonical(nt::request(engine_id, op, payload)),
          {{environment, stage}, {environment + "_FD", std::to_string(descriptors[1])}}, temp.path);
        ::close(descriptors[1]); descriptors[1] = -1;
        struct pollfd fd{descriptors[0], POLLIN, 0};
        int ready; do { ready = ::poll(&fd, 1, 20000); } while (ready < 0 && errno == EINTR);
        NT_REQUIRE(ready > 0);
        char marker[128]; const auto size = ::read(descriptors[0], marker, sizeof(marker));
        NT_REQUIRE(size > 0 && std::string(marker, static_cast<std::size_t>(size)) == stage + "\n");
        child.wait_stopped(10); child.kill(); NT_REQUIRE(child.wait() == -SIGKILL && child.out().empty());
      } catch (...) {
        ::close(descriptors[0]); if (descriptors[1] >= 0) ::close(descriptors[1]); throw;
      }
      ::close(descriptors[0]);
      NT_REQUIRE(call("status", key) == sentinel);
      if (stage == "prepare.before_commit") call("status", target_key, false);
      else NT_REQUIRE(call("status", target_key)["state"] == (stage == "commit.after_commit" ? "committed" : "prepared"));
      prepared = call("prepare", base);
      const auto commit_payload = nt::updated(target_key, {{"expected_intent_digest", prepared["intent"]["digest"]}});
      auto result = call("commit", commit_payload);
      NT_REQUIRE(result["state"] == "committed" && call("commit", commit_payload) == result && call("status", target_key) == result);
      NT_REQUIRE(call("status", key) == sentinel);
      if (stage == "commit.after_commit" && args.has("retain-root")) {
        const auto retained = fs::path(args.require("retain-root"));
        NT_REQUIRE(!fs::exists(retained));
        fs::create_directories(retained.parent_path());
        fs::copy(temp.path, retained, fs::copy_options::recursive);
        nt::write_json(retained / "expected-status.json", result);
        nt::write_json(retained / "status-request.json", nt::request(engine_id, "status", target_key));
      }
      std::cout << domain << " " << stage << " passed\n"; ++cases;
    }
    std::cout << J{{"status", "passed"}, {"cases", cases}, {"power_loss_tested", false},
      {"scope", "fault-instrumented native prepare/commit; installed production reopen and retry"}}.dump() << '\n';
  });
}
