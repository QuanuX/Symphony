#include "connector.hpp"
#include "execution_fixture.hpp"
int main(int argc, char **argv) {
  try {
    if (argc == 2 && std::string(argv[1]) == "--connector-supervise")
      return cloudflare::connector_supervise(true);
    NT_REQUIRE(argc == 3);
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
    Authority authority(h.temp.path / "a.sock", 1);
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
    auto start = invoke("instance", "plan",
                        {{"action", "start"},
                         {"expected_state", current()},
                         {"configuration", nullptr}});
    auto running = apply("instance", start);
    NT_REQUIRE(running.at("current").at("runtime").at("state") == "running" &&
               running.at("current").at("provider_readiness") == "not_proven");
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
    authority.check();
    std::cout << "Connector package/configuration/start/stop/update/retire, "
                 "native token purpose, exact supervisor identity, replay "
                 "refusal and observation-only recovery passed with inert "
                 "child; zero provider calls\n";
    return 0;
  } catch (const std::exception &e) {
    std::cerr << e.what() << '\n';
    return 1;
  }
}
