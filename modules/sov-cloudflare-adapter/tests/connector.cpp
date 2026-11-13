#include "connector.hpp"
#include "execution_fixture.hpp"
struct FixtureRuntimeCleanup {
  fs::path run;
  std::string supervisor_digest;
  ~FixtureRuntimeCleanup() {
    try {
      auto file = fs::exists(run / "runtime.json") ? run / "runtime.json"
                                                   : run / "runtime.saved";
      auto record = n::parse(n::read(file, 32768, true));
      if (record.at("state") != "running")
        return;
      auto channel = n::connect(
          run.parent_path().parent_path() / (run.filename().string() + ".sock"),
          ::geteuid(), supervisor_digest);
      n::timeout(channel.n, 5);
      n::frame(channel.n, Json{{"operation", "stop"},
                               {"instance_digest", record.at("instance_digest")}}.dump());
      (void)n::frame(channel.n);
    } catch (...) {
    }
  }
};
int main(int argc, char **argv) {
  try {
    if (argc == 2 && std::string(argv[1]) == "--connector-supervise")
      return cloudflare::connector_supervise(true);
    NT_REQUIRE(argc == 3 || argc == 4);
    const std::string fault = argc == 4 ? argv[3] : "";
    NT_REQUIRE(fault.empty() || fault == "early-exit" || fault == "audit-loss");
    auto self = n::executable(::getpid());
    Harness h(fs::canonical(argv[1]), self);
    auto binary = fs::canonical(argv[2]);
    auto bd = n::hash(read(binary));
    auto config = n::parse(n::read(h.vault / "config.json", 32768, true));
    config["toolchain_digest"] = bd;
    n::atomic(h.vault / "config.json", config);
    h.execution["config_digest"] =
        n::seal(config, "symphony.ssiag.private-store-config.v1");
    n::private_dir(h.vault / "credentials/tunnel", true);
    n::put(h.vault / "credentials/tunnel/value", "TUNNEL-SENTINEL");
    auto md = n::parse(
        n::read(h.vault / "credentials/api/metadata.json", 32768, true));
    md["reference"] = "ssiag-native:tunnel";
    md["purpose"] = "cloudflare.tunnel-token";
    md["resource_id"] = "11111111-1111-4111-8111-111111111111";
    n::put(h.vault / "credentials/tunnel/metadata.json", md.dump());
    std::unique_ptr<Authority> authority;
    Json result;
    bool started = false;
    auto invoke = [&](const std::string &family, const std::string &verb,
                      Json in, Json plan = nullptr, bool fixture = true) {
      std::cerr << "fixture " << family << "." << verb << " " << in.value("action",std::string()) << "\n";
      in["store"] = h.root.string();
      Json request{{"operation", "connector." + family + "." + verb},
                   {"request_id", n::uuid()},
                   {"descriptor_ref",
                    cloudflare::digest(cloudflare::owner(),
                                       "symphony.sov.cloudflare-owner.v1")},
                   {"installation_ref",
                    {{"executable_digest", n::hash(read(self))},
                     {"version", "0.1.0-dev"},
                     {"receipt_digest", "sha256:" + std::string(64, 'e')}}},
                   {"plan_ref", plan},
                   {"input", in}};
      result = Json::object();
      started = false;
      auto data =
          cloudflare::connector_admin(request, self, result, started, fixture);
      NT_REQUIRE(data.dump().find("SENTINEL") == std::string::npos);
      return data;
    };
    auto current = [&] {
      return invoke("installation", "list", Json::object()).at("state_digest");
    };
    auto install = Json{{"action", "install"},
                        {"expected_state", current()},
                        {"source", binary.string()},
                        {"package_digest", bd}};
    bool refused = false;
    try {
      invoke("installation", "plan", install, nullptr, false);
    } catch (...) {
      refused = true;
    }
    NT_REQUIRE(refused);
    auto apply = [&](const std::string &family, const Json &p) {
      auto d = p.at("plan_digest");
      return invoke(family, "apply", {{"plan_digest", d}}, d);
    };
    auto install_plan = invoke("installation", "plan", install);
    // Re-sealing a malformed plan must not bypass descriptor validation.
    // This also catches a descriptor loop over an already-destroyed owner JSON.
    auto malformed = install_plan.at("plan");
    malformed["unexpected"] = true;
    auto malformed_digest = cloudflare::digest(
        malformed, "symphony.sov.cloudflare-connector-plan.v1");
    n::put(h.root / "connector/plans" / (malformed_digest.substr(7) + ".json"),
           malformed.dump());
    bool malformed_refused = false;
    try {
      apply("installation", Json{{"plan_digest", malformed_digest}});
    } catch (...) {
      malformed_refused = true;
    }
    NT_REQUIRE(malformed_refused && current() == install.at("expected_state"));
    apply("installation", install_plan);
    apply("installation", invoke("installation", "plan",
                                 {{"action", "select"},
                                  {"expected_state", current()},
                                  {"source", nullptr},
                                  {"package_digest", bd}}));
    Json cfg{{"account_id", std::string(32, '0')},
             {"tunnel_id", md.at("resource_id")},
             {"package_digest", bd},
             {"credential_ref", "ssiag-native:tunnel"},
             {"execution", h.execution}};
    apply("instance", invoke("instance", "plan",
                             {{"action", "configure"},
                              {"expected_state", current()},
                              {"configuration", cfg}}));
    auto start_plan = [&] {
      return invoke("instance", "plan",
                    {{"action", "start"},
                     {"expected_state", current()},
                     {"configuration", nullptr}});
    };
    auto journal_path = [&](const Json &p) {
      return h.root / "connector/attempts" /
             (p.at("plan_digest").get<std::string>().substr(7) + ".json");
    };
    auto run_path = [&](const Json &p) {
      return h.root / "connector/runs" /
             p.at("plan").at("after").at("run_id").get<std::string>();
    };
    auto rejects = [&](auto fn) {
      bool caught = false;
      try {
        fn();
      } catch (...) {
        caught = true;
      }
      NT_REQUIRE(caught);
    };
    if (!fault.empty()) {
      auto p = start_plan();
      auto run = run_path(p);
      FixtureRuntimeCleanup cleanup{run, n::hash(read(self))};
      n::private_dir(run, true);
      if (fault == "early-exit")
        n::put(run / "test-early-exit", "inert fixture\n");
      authority = std::make_unique<Authority>(
          h.temp.path / "a.sock", 1,
          [](const Json &) { return true; },
          [run, fault](const Json &) {
            // Observe the actual child effect before deliberately losing the
            // audit reply. The early-exit case combines both faults without
            // assuming exec completes inside the production liveness interval.
            auto marker = run / (fault == "early-exit" ? "child-exited"
                                                       : "child-launched");
            auto deadline = n::now() + 3000;
            while (!fs::exists(marker) && n::now() < deadline)
              ::poll(nullptr, 0, 20);
            NT_REQUIRE(fs::exists(marker));
            return true;
          });
      rejects([&] { apply("instance", p); });
      NT_REQUIRE(fs::exists(run / "child-launched") &&
                 fs::exists(run / "child-exited") == (fault == "early-exit") &&
                 !fs::exists(run / "startup-refusal.json") &&
                 !fs::exists(run / "runtime.json") &&
                 n::parse(n::read(journal_path(p))).at("effect_state") ==
                     "unresolved");
      auto d = p.at("plan_digest");
      rejects([&] {
        invoke("instance", "reconcile", {{"plan_digest", d}}, d);
      });
      rejects([&] { start_plan(); });
      authority->check();
      std::cout << "Post-launch "
                << (fault == "early-exit" ? "early-exit with audit-reply loss"
                                           : "audit-reply loss")
                << " remains unknown without a pre-launch refusal receipt\n";
      return 0;
    }
    // Real private-provider admission failures, including authority-fixture refusal,
    // must leave a consumed, not-applied start rather than brick the store.
    for (const auto &failure : {"expired", "revoked", "denied"}) {
      auto selected = md;
      if (std::string(failure) == "expired")
        selected["expires_unix_ms"] = "1";
      if (std::string(failure) == "revoked") {
        selected["generation"] = "2";
        selected["expires_unix_ms"] = "0";
      }
      n::atomic(h.vault / "credentials/tunnel/metadata.json", selected);
      auto refused_plan = start_plan();
      auto before = current();
      if (std::string(failure) == "denied")
        authority = std::make_unique<Authority>(
            h.temp.path / "a.sock", 1,
            [](const Json &) { return false; });
      auto refused_start = apply("instance", refused_plan);
      if (authority) {
        authority->check();
        authority.reset();
      }
      NT_REQUIRE(result.at("outcome") == "rejected" &&
                 result.at("effect_state") == "not_applied" &&
                 refused_start.at("attempt").at("stage") == "settled" &&
                 refused_start.at("attempt").at("effect_state") == "not_applied" &&
                 current() == before &&
                 !fs::exists(run_path(refused_plan) / "runtime.json") &&
                 !fs::exists(run_path(refused_plan) / "child-launched"));
      rejects([&] { apply("instance", refused_plan); });
      // Emulate loss after the supervisor's durable refusal and before parent
      // settlement. Reconciliation must use the bound receipt without dispatch.
      auto d = refused_plan.at("plan_digest");
      n::atomic(journal_path(refused_plan),
                Json{{"plan_digest", d}, {"stage", "dispatch_started"},
                     {"effect_state", "unresolved"}});
      if (std::string(failure) == "expired") {
        auto receipt_path = run_path(refused_plan) / "startup-refusal.json";
        auto receipt = n::parse(n::read(receipt_path));
        NT_REQUIRE(::unlink(receipt_path.c_str()) == 0);
        rejects([&] {
          invoke("instance", "reconcile", {{"plan_digest", d}}, d);
        });
        rejects([&] { start_plan(); });
        auto wrong = receipt;
        wrong["run_id"] = n::uuid();
        n::atomic(receipt_path, wrong);
        rejects([&] {
          invoke("instance", "reconcile", {{"plan_digest", d}}, d);
        });
        n::atomic(receipt_path, receipt);
      }
      auto reconciled = invoke("instance", "reconcile", {{"plan_digest", d}}, d);
      NT_REQUIRE(reconciled.at("attempt").at("effect_state") == "not_applied" &&
                 current() == before &&
                 !fs::exists(run_path(refused_plan) / "child-launched"));
      // Repeated inspection/reconciliation preserves the same terminal outcome.
      NT_REQUIRE(invoke("instance", "reconcile", {{"plan_digest", d}}, d)
                     .at("attempt").at("effect_state") == "not_applied");
      n::atomic(h.vault / "credentials/tunnel/metadata.json", md);
      apply("instance", invoke("instance", "plan",
                               {{"action", "update"},
                                {"expected_state", current()},
                                {"configuration", cfg}}));
    }
    auto start = invoke("instance", "plan",
                        {{"action", "start"},
                         {"expected_state", current()},
                         {"configuration", nullptr}});
    // Begin each bounded authority wait only when its request is imminent;
    // offline setup/refusal cases must not consume the fixture's accept window.
    authority = std::make_unique<Authority>(h.temp.path / "a.sock", 1);
    auto run = run_path(start);
    FixtureRuntimeCleanup cleanup{run, n::hash(read(self))};
    auto running = apply("instance", start);
    NT_REQUIRE(running.at("current").at("runtime").at("state") == "running" &&
               running.at("current").at("provider_readiness") == "not_proven");
    // Production start establishes liveness, not completed exec/fixture setup.
    // Wait only in the test for the independently observable child effect.
    auto marker_deadline = n::now() + 3000;
    while (!fs::exists(run / "child-launched") && n::now() < marker_deadline)
      ::poll(nullptr, 0, 20);
    NT_REQUIRE(fs::exists(run / "child-launched") &&
               !fs::exists(run / "startup-refusal.json"));
    auto runtime = n::parse(n::read(run / "runtime.json", 32768, true));
    // Model a lost start reply while the actual child remains alive. No runtime
    // observation and no refusal receipt means unknown, never a safe refusal.
    n::atomic(h.root / "connector/state.json", start.at("plan").at("before"));
    n::atomic(journal_path(start),
              Json{{"plan_digest", start.at("plan_digest")},
                   {"stage", "dispatch_started"},
                   {"effect_state", "unresolved"}});
    NT_REQUIRE(::rename((run / "runtime.json").c_str(),
                        (run / "runtime.saved").c_str()) == 0);
    auto start_digest = start.at("plan_digest");
    rejects([&] {
      invoke("instance", "reconcile", {{"plan_digest", start_digest}}, start_digest);
    });
    rejects([&] { start_plan(); });
    NT_REQUIRE(n::parse(n::read(journal_path(start))).at("effect_state") ==
               "unresolved");
    NT_REQUIRE(::rename((run / "runtime.saved").c_str(),
                        (run / "runtime.json").c_str()) == 0);
    auto observed = invoke("instance", "reconcile",
                           {{"plan_digest", start_digest}}, start_digest);
    NT_REQUIRE(observed.at("attempt").at("effect_state") == "applied_verified" &&
               observed.at("current").at("runtime").at("pid") == runtime.at("pid"));
    bool replay = false;
    try {
      apply("instance", start);
    } catch (...) {
      replay = true;
    }
    NT_REQUIRE(replay);
    auto stop = invoke("instance", "plan",
                       {{"action", "stop"},
                        {"expected_state", current()},
                        {"configuration", nullptr}});
    auto stopped = apply("instance", stop);
    NT_REQUIRE(stopped.at("current").at("runtime").at("state") == "stopped");
    auto d = stop.at("plan_digest");
    NT_REQUIRE(invoke("instance", "reconcile", {{"plan_digest", d}}, d)
                   .at("attempt")
                   .at("effect_state") == "applied_verified");
    apply("instance", invoke("instance", "plan",
                             {{"action", "update"},
                              {"expected_state", current()},
                              {"configuration", cfg}}));
    apply("instance", invoke("instance", "plan",
                             {{"action", "retire"},
                              {"expected_state", current()},
                              {"configuration", nullptr}}));
    authority->check();
    authority.reset();
    for (auto scenario : {"early-exit", "audit-loss"}) {
      auto child_result = native_test::run(
          {self.string(), argv[1], argv[2], scenario});
      if (child_result.returncode != 0)
        std::cerr << child_result.stderr_text;
      NT_REQUIRE(child_result.returncode == 0);
      std::cout << child_result.stdout_text;
    }
    std::cout << "Connector package/configuration/start/stop/update/retire, "
                 "native token purpose, exact supervisor identity, replay "
                 "refusal, expired/revoked/denied pre-launch settlement, "
                 "lost-refusal recovery and live-child unknown recovery passed with inert "
                 "child; zero provider calls\n";
    return 0;
  } catch (const std::exception &e) {
    std::cerr << e.what() << '\n';
    return 1;
  }
}
