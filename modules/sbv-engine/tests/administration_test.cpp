#include "../../ssfv-engine/src/ssfv.hpp"
#include <filesystem>
#include <fstream>
#include <iostream>
#include <symphony/sbv/sbv.hpp>
#include <symphony/knowledge/engine/digest.hpp>
namespace e = symphony::knowledge::engine;
namespace f = symphony::knowledge::ssfv;
using J = e::Json;
J read(const char *path) {
  std::ifstream file(path);
  J j;
  file >> j;
  return j;
}
J call(const std::string &op, const J &p) {
  return f::handle_request(
      {"sbv-test", "sbv-test", op, f::engine_id, e::unix_time_ms() + 60000, p});
}
int main(int argc, char **argv) try {
  if (argc != 2)
    return 2;
  std::filesystem::current_path(argv[1]);
  auto check = call("check", {{"expected_namespace_digest", nullptr},
                              {"expected_registry_digest", nullptr},
                              {"freshness", "disabled"},
                              {"baseline", nullptr}});
  if (check.at("summary").at("state") != "valid") {
    std::cerr << check.dump(2);
    return 1;
  }
  auto descriptor = symphony::sbv::descriptor();
  auto declaration = read("modules/sbv-engine/OWNER-INTERFACE.json");
  if (descriptor.at("operations") != declaration.at("operations") ||
      descriptor.at("contract_versions") !=
          declaration.at("contract_versions")) {
    std::cerr << "descriptor/declaration mismatch\n";
    return 1;
  }
  J evidence = J::array();
  for (const auto *feature :
       {"ssfv:symphony:sbv-engine", "ssfv:symphony:qxctl-sbv-administration"}) {
    auto result = call(
        "administration-check",
        {{"protocol", "symphony.knowledge.administration-coverage-input.v1"},
         {"format_version", 1},
         {"semantic_snapshot", check.at("semantic_snapshot")},
         {"profile", read("knowledge/FEATURE-ADMINISTRATION-PROFILE.json")},
         {"expected_command_registry", read("tools/qxctl/COMMANDS.json")},
         {"observed_qxctl_state", "not_evaluated"},
         {"observed_command_registry", nullptr},
         {"engine_descriptors", J::array({descriptor})},
         {"requested_feature_id", feature}});
    if (!result.at("feature_findings").empty() ||
        result.at("summary").at("satisfied") != 4 ||
        result.at("summary").at("unresolved") != 0 ||
        result.at("module_integrations")[0].at("integration_state") !=
            "integration_ready") {
      std::cerr << result.dump(2);
      return 1;
    }
    evidence.push_back(result);
  }
  for (bool legacy : {true, false}) {
    auto invalid = descriptor;
    if (legacy) invalid["process_protocols"] = J::array({e::process_protocol_v1});
    else invalid["limits"]["response_bytes"] = nullptr;
    invalid.erase("descriptor_digest");
    invalid["descriptor_digest"] = e::tagged_sha256(invalid.dump());
    auto result = call("administration-check",
        {{"protocol", "symphony.knowledge.administration-coverage-input.v1"},
         {"format_version", 1}, {"semantic_snapshot", check.at("semantic_snapshot")},
         {"profile", read("knowledge/FEATURE-ADMINISTRATION-PROFILE.json")},
         {"expected_command_registry", read("tools/qxctl/COMMANDS.json")},
         {"observed_qxctl_state", "not_evaluated"}, {"observed_command_registry", nullptr},
         {"engine_descriptors", J::array({invalid})},
         {"requested_feature_id", "ssfv:symphony:sbv-engine"}});
    if (result.at("module_integrations")[0].at("integration_state") != "descriptor_invalid")
      throw std::runtime_error("deadline-profile exception escaped its scope");
  }
  std::cout << evidence.dump(2) << '\n';
  return 0;
} catch (const std::exception &x) {
  std::cerr << x.what() << '\n';
  return 1;
}
