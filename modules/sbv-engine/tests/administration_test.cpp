#include "../../ssfv-engine/src/ssfv.hpp"
#include <filesystem>
#include <fstream>
#include <iostream>
#include <symphony/knowledge/engine/digest.hpp>
#include <symphony/knowledge/engine/error.hpp>
#include <symphony/sbv/sbv.hpp>
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
  const auto capabilities = symphony::sbv::dispatch(
      "capabilities", {{"protocol", "symphony.sbv.capabilities-input.v1"}},
      e::no_deadline);
  const auto catalogue = symphony::sbv::dispatch(
      "catalogue", {{"protocol", "symphony.sbv.catalogue-input.v1"}},
      e::no_deadline);
  for (const auto &[cap, cards] : {std::pair{"studies", "studies"},
                                   {"execution_models", "models"},
                                   {"economic_transforms", "transforms"}}) {
    auto expected = J::array();
    for (const auto &card : catalogue.at(cards))
      expected.push_back(card.at("id"));
    if (capabilities.at(cap) != expected)
      throw std::runtime_error("capability/catalogue inventory mismatch");
  }
  if (capabilities.at("resident_data").at("operations") !=
          J::array({"run", "evaluate", "book", "generate_census"}) ||
      !capabilities.at("limits").at("max_generated_census_signals").is_null())
    throw std::runtime_error("native generation capability mismatch");
  auto descriptor = symphony::sbv::descriptor();
  if (e::sha256_hex(f::descriptor().dump()) !=
      "e3ae6f7cc29402c9df275c0bb7a34553c1358ea5667f68c15bf35d493f8910ab")
    throw std::runtime_error("frozen SSFV descriptor-v1 bytes changed");
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
  auto assess_descriptor = [&](J candidate) {
    candidate.erase("descriptor_digest");
    candidate["descriptor_digest"] = e::tagged_sha256(candidate.dump());
    return call(
        "administration-check",
        {{"protocol", "symphony.knowledge.administration-coverage-input.v1"},
         {"format_version", 1},
         {"semantic_snapshot", check.at("semantic_snapshot")},
         {"profile", read("knowledge/FEATURE-ADMINISTRATION-PROFILE.json")},
         {"expected_command_registry", read("tools/qxctl/COMMANDS.json")},
         {"observed_qxctl_state", "not_evaluated"},
         {"observed_command_registry", nullptr},
         {"engine_descriptors", J::array({candidate})},
         {"requested_feature_id", "ssfv:symphony:sbv-engine"}});
  };
  // A module may describe protocols beyond its operation I/O pairs. This
  // independent larger inventory must not acquire a replacement policy cap.
  auto larger = descriptor;
  for (unsigned i = 0; i < 2048; ++i)
    larger["contract_versions"].push_back("fixture:retained-contract:" +
                                          std::to_string(i));
  // Existing printable text admits newline/tab; widening cardinality must not
  // silently narrow this established scalar-string behavior.
  larger["contract_versions"].push_back("fixture:retained\ncontract\ttext");
  if (assess_descriptor(larger).at("module_integrations")[0].at(
          "integration_state") != "integration_ready")
    throw std::runtime_error("larger descriptor contract inventory rejected");
  const auto reject_contracts = [&](const char *label, const J &contracts) {
    auto invalid = descriptor;
    invalid["contract_versions"] = contracts;
    if (assess_descriptor(invalid).at("module_integrations")[0].at(
            "integration_state") != "descriptor_invalid")
      throw std::runtime_error(
          std::string("invalid descriptor contract inventory admitted: ") +
          label);
  };
  reject_contracts("empty array", J::array());
  reject_contracts("duplicate", J::array({"duplicate", "duplicate"}));
  reject_contracts("empty string", J::array({""}));
  reject_contracts("overlong string", J::array({std::string(257, 'x')}));
  reject_contracts("control byte",
                   J::array({std::string("not") + char(1) + "printable"}));
  reject_contracts("non-string", J::array({17}));
  auto oversized_contracts = J::array();
  for (std::size_t i = 0; i <= f::max_process_json_values; ++i)
    oversized_contracts.push_back(std::to_string(i));
  reject_contracts("parser-budget cardinality", oversized_contracts);

  // Exercise the same parser entry point and unchanged budgets as SSFV main;
  // budget rejection occurs before administration semantics can run.
  const auto reject_process = [&](const J &payload, const std::string &code) {
    const auto now = e::unix_time_ms();
    const J wire{{"protocol", e::process_protocol_v1},
                 {"request_id", "budget-test"},
                 {"correlation_id", "budget-test"},
                 {"operation", "administration-check"},
                 {"target_engine", f::engine_id},
                 {"deadline_unix_ms", now + 60000},
                 {"payload", payload}};
    try {
      (void)e::parse_request(wire.dump(), f::engine_id, now,
                             f::max_process_json_values);
    } catch (const e::Error &error) {
      if (error.code() == code)
        return;
      throw;
    }
    throw std::runtime_error("SSFV process budget was widened");
  };
  reject_process({{"contract_versions", oversized_contracts}},
                 "json.value_count_exceeded");
  reject_process({{"text", std::string(e::Limits::max_request_bytes, 'x')}},
                 "input.too_large");
  for (bool legacy : {true, false}) {
    auto invalid = descriptor;
    if (legacy)
      invalid["process_protocols"] = J::array({e::process_protocol_v1});
    else
      invalid["limits"]["response_bytes"] = nullptr;
    invalid.erase("descriptor_digest");
    invalid["descriptor_digest"] = e::tagged_sha256(invalid.dump());
    auto result = call(
        "administration-check",
        {{"protocol", "symphony.knowledge.administration-coverage-input.v1"},
         {"format_version", 1},
         {"semantic_snapshot", check.at("semantic_snapshot")},
         {"profile", read("knowledge/FEATURE-ADMINISTRATION-PROFILE.json")},
         {"expected_command_registry", read("tools/qxctl/COMMANDS.json")},
         {"observed_qxctl_state", "not_evaluated"},
         {"observed_command_registry", nullptr},
         {"engine_descriptors", J::array({invalid})},
         {"requested_feature_id", "ssfv:symphony:sbv-engine"}});
    if (result.at("module_integrations")[0].at("integration_state") !=
        "descriptor_invalid")
      throw std::runtime_error("deadline-profile exception escaped its scope");
  }
  std::cout << evidence.dump(2) << '\n';
  return 0;
} catch (const std::exception &x) {
  std::cerr << x.what() << '\n';
  return 1;
}
