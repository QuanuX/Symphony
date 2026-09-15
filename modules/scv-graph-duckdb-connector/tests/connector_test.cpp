#include "connector_test_support.hpp"
using namespace scv_test;
int main(int argc, char **argv) {
  return nt::test_main([&] {
    nt::Arguments args(argc, argv);
    Config config;
    config.library = fs::canonical(args.require("library")).string();
    config.self = fs::canonical(argv[0]).string();
    config.version = args.get("version", "0.1.0-dev");
    NT_REQUIRE(config.version == "0.1.0-dev" || config.version == "0.2.0-dev");
    if (args.has("crash-writer")) {
      SQL db(args.require("crash-writer"), config.library);
      db.query("BEGIN TRANSACTION");
      db.query("DELETE FROM nodes");
      nt::write(args.require("ready"), "uncommitted delete completed\n");
      std::this_thread::sleep_for(std::chrono::seconds(60));
      return;
    }
    config.engine = fs::canonical(args.require("engine")).string();
    if (args.has("evidence")) {
      config.evidence = fs::absolute(args.require("evidence"));
      NT_REQUIRE(!fs::exists(config.evidence));
      fs::create_directories(config.evidence);
    }
    for (const auto &[name, method] : connector_cases) {
      ConnectorTests suite(config);
      (suite.*method)();
      std::cout << name << " passed\n";
    }
    std::cout << J{{"status", "passed"},
                   {"tests", connector_cases.size()},
                   {"calls", config.calls}}
                     .dump()
              << "\n";
  });
}
