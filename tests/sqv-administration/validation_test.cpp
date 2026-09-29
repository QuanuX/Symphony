#include "request.hpp"
#include <fstream>
#include <iostream>
namespace owner = SQV_NAMESPACE;
using namespace symphony::sqv_admin;
void check(bool yes) {
  if (!yes)
    throw std::runtime_error("administration test assertion");
}
Json invoke(std::string op, Json p) {
  Json envelope{{"protocol", engine::process_protocol_v1},
                {"request_id", "test"},
                {"correlation_id", "test"},
                {"target_engine", owner::engine_id},
                {"operation", op},
                {"deadline_unix_ms", engine::unix_time_ms() + 4000},
                {"payload", p}};
  return owner::handle_request(engine::parse_request(envelope.dump(),
                                                     owner::engine_id,
                                                     engine::unix_time_ms()))
      .at("data");
}
void rejects(std::string op, Json p) {
  bool refused = false;
  try {
    (void)invoke(op, p);
  } catch (const engine::Error &) {
    refused = true;
  }
  check(refused);
}
int main(int argc, char **argv) try {
  check(argc == 3);
  std::ifstream file(argv[2]);
  Json p;
  file >> p;
  std::string op = argv[1];
  auto result = invoke(op, p);
  auto wrong = p;
  wrong["extra"] = true;
  rejects(op, wrong);
  wrong = p;
  wrong["protocol"] = "wrong";
  rejects(op, wrong);
  rejects("unknown_operation", p);
  if (op == "metadata_validate") {
    wrong = p;
    wrong["description"]["dataset_id"] = "00fffe01";
    auto binary = invoke(op, wrong);
    check(binary["description"]["dataset_id"] == "00fffe01");
    Json inspect{{"protocol", "symphony.sqmv.metadata-inspect-input.v1"},
                 {"encoded_hex_chunks", binary["encoded_hex_chunks"]},
                 {"expected_reference", binary["metadata_reference"]},
                 {"projection", "all"},
                 {"limits", p["limits"]}};
    check(invoke("metadata_inspect", inspect)["description"] ==
          binary["description"]);
    inspect["expected_reference"] = "sqmv1-sha256-" + std::string(64, '0');
    rejects("metadata_inspect", inspect);
    wrong = p;
    std::reverse(wrong["description"]["evidence"].begin(),
                 wrong["description"]["evidence"].end());
    check(invoke(op, wrong)["metadata_reference"] ==
          result["metadata_reference"]);
    wrong = p;
    wrong["description"]["evidence"].push_back(
        wrong["description"]["evidence"][0]);
    rejects(op, wrong);
    wrong = p;
    wrong["limits"]["max_field_bytes"] = "1";
    rejects(op, wrong);
    // Exercise a manifest beyond one hex JSON string's transport capacity.
    wrong = p;
    for (int i = 0; i < 8; ++i)
      wrong["description"]["evidence"].push_back(
          {{"role", "lineage"},
           {"producer_ref", "74"},
           {"evidence_ref",
            hex(std::string(3900, static_cast<char>('a' + i)))}});
    auto large = invoke(op, wrong);
    check(large["encoded_hex_chunks"].size() >= 2);
    inspect["encoded_hex_chunks"] = large["encoded_hex_chunks"];
    inspect["expected_reference"] = large["metadata_reference"];
    inspect["projection"] = "lineage";
    auto projection = invoke("metadata_inspect", inspect);
    check(projection["description"]["evidence"].size() == 8);
  } else if (op == "flow_validate") {
    check(result["payload_bytes"] == "8");
    for (const auto &[section, key, value] :
         std::vector<std::tuple<std::string, std::string, std::string>>{
             {"limits", "global_allocation_bytes", "1"},
             {"limits", "max_frame_bytes", "64"},
             {"port", "max_pending_entries", "0"},
             {"port", "outstanding_byte_credit", "1"},
             {"port", "producer_generation", std::string(32, '0')},
             {"port", "next_sequence", "0"},
             {"frame", "payload_bytes", "18446744073709551615"}}) {
      wrong = p;
      wrong[section][key] = value;
      rejects(op, wrong);
    }
  } else if (op == "conversion_validate") {
    check(result["input_bytes"] == "32" && result["output_bytes"] == "64");
    std::vector<std::string> layouts;
    for (auto bits : {8, 16, 32, 64})
      for (const auto *sign : {"i", "u"})
        for (const auto *order : {"le", "be"})
          layouts.push_back("sqtv-int-v1-" + std::string(sign) +
                            std::to_string(bits) + "-" + order);
    for (const auto &from : layouts)
      for (const auto &to : layouts) {
        wrong = p;
        wrong["input_layout"] = from;
        wrong["output_layout"] = to;
        (void)invoke(op, wrong);
      }
    for (const auto *n :
         {"0", "01", "-1", "18446744073709551615", "18446744073709551616"}) {
      wrong = p;
      wrong["element_count"] = n;
      rejects(op, wrong);
    }
    wrong = p;
    wrong["limits"]["max_elements"] = "8388609";
    rejects(op, wrong);
    wrong = p;
    wrong["limits"]["max_output_bytes"] = "1";
    rejects(op, wrong);
    wrong = p;
    wrong["output_layout"] = "float32";
    rejects(op, wrong);
  }
  std::cout << op << " native boundaries passed\n";
  return 0;
} catch (const std::exception &e) {
  std::cerr << e.what() << '\n';
  return 1;
}
