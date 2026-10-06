#include "../../sqav-databento-dbn-cpp/tests/public_fixture.hpp"
#include <cstdlib>
#include <filesystem>
#include <fstream>
#include <iostream>
#include <source_location>
#include <symphony/knowledge/engine/digest.hpp>
#include <symphony/knowledge/engine/protocol.hpp>
#include <symphony/sbv/models.hpp>
#include <unistd.h>
namespace e = symphony::knowledge::engine;
namespace s = symphony::sbv;
using J = s::Json;
unsigned checks = 0;
void check(bool ok,
           std::source_location where = std::source_location::current()) {
  ++checks;
  if (!ok)
    throw std::runtime_error("model test failure line " +
                             std::to_string(where.line()));
}
J read(const std::string &path) {
  std::ifstream f(path);
  J j;
  f >> j;
  return j;
}
J rat(const char *n, const char *d) {
  return {{"numerator", n}, {"denominator", d}};
}
J call(const char *op, const J &p) {
  return s::dispatch(op, p, e::unix_time_ms() + 30000);
}
int main() try {
  char tmp[] = "/private/tmp/sbv-models-XXXXXX";
  check(::mkdtemp(tmp) != nullptr);
  const std::string root = tmp;
  struct Cleanup {
    std::string root;
    ~Cleanup() { std::filesystem::remove_all(root); }
  } cleanup{root};
  auto fixture = public_fixture::fixture();
  std::vector<unsigned char> bytes(fixture.begin(), fixture.begin() + 360);
  auto put = [&](std::size_t offset, std::uint64_t v, unsigned n) {
    for (unsigned i = 0; i < n; ++i)
      bytes[offset + i] = static_cast<unsigned char>(v >> (i * 8));
  };
  const std::vector<std::int64_t> prices{100, 95, 105, 96, 104, 97,
                                         103, 98, 102, 99, 101, 100};
  const std::uint64_t t = 1609160400000000000ULL;
  for (std::size_t i = 0; i < prices.size(); ++i) {
    const auto offset = bytes.size();
    bytes.insert(bytes.end(), fixture.begin() + 360, fixture.begin() + 416);
    put(offset + 8, t + i * 100, 8);
    put(offset + 40, t + i * 100, 8);
    put(offset + 24, prices[i] * 1000000000ULL, 8);
    bytes[offset + 38] = 'T';
  }
  const std::string source(reinterpret_cast<const char *>(bytes.data()),
                           bytes.size());
  {
    std::ofstream f(root + "/fixture.dbn", std::ios::binary);
    f << source;
  }
  const auto sha = e::sha256_hex(source);
  J producer{{"id", "external-test"},
             {"version", "1"},
             {"artifact_sha256", ""},
             {"reproducibility", "nondeterministic"}};
  auto signal = [&](std::size_t i, const char *id) -> J {
    return {{"signal_id", id},
            {"source_ordinal", std::to_string(i)},
            {"available_ns", std::to_string(t + i * 100)},
            {"anchor_price_nanos", std::to_string(prices[i] * 1000000000LL)},
            {"causal_end_ordinal_exclusive", std::to_string(i + 1)},
            {"context_reference", "user://private-context"}};
  };
  J p{{"protocol", "symphony.sbv.evaluate-input.v1"},
      {"source_path", root + "/fixture.dbn"},
      {"source_sha256", sha},
      {"dataset", "GLBX.MDP3"},
      {"output_path", root + "/first.json"},
      {"census",
       {{"protocol", "symphony.sbv.external-census.v1"},
        {"source_sha256", sha},
        {"mode", "causal_declared"},
        {"producer", producer},
        {"signals", J::array({signal(0, "signal/a~B"), signal(1, "other")})}}},
      {"model",
       {{"protocol", "symphony.sbv.model-selection.v1"},
        {"id", "observed_trade_levels"},
        {"version", "1"},
        {"horizon_ns", "1100"},
        {"parameters",
         {{"levels_per_side", "5"},
          {"include_anchor", false},
          {"thin_support", "unavailable"}}}}},
      {"replay",
       {{"before_ns", "0"}, {"after_ns", "1100"}, {"retain_events", true}}},
      {"studies",
       J::array({"signal_summary", "model_summary", "path_excursion"})},
      {"workers", "1"},
      {"extensions", J::object()}};
  if (const char *destination = std::getenv("SYMPHONY_SBV_EMIT_FIXTURE")) {
    const std::filesystem::path directory = destination;
    check(directory.is_absolute() && std::filesystem::is_directory(directory));
    auto emitted = p;
    emitted["source_path"] = (directory / "fixture.dbn").string();
    emitted["output_path"] = (directory / "new-evaluation.json").string();
    std::ofstream raw(directory / "fixture.dbn", std::ios::binary);
    raw << source;
    std::ofstream request(directory / "evaluate-request.json");
    request << emitted.dump(2) << '\n';
    check(raw.good() && request.good());
  }
  auto receipt = call("evaluate", p);
  const auto original = read(root + "/first.json");
  const auto &sections = original.at("sections");
  const auto &outcomes = sections.at("execution").at("data");
  check(outcomes.size() == 2 && outcomes[0].at("support").size() == 10);
  check(outcomes[0].at("support")[0].at("price_nanos") == "95000000000");
  check(outcomes[0].at("support")[9].at("price_nanos") == "105000000000");
  for (const auto &a : outcomes[0].at("support")) {
    check(a.at("weight") == rat("1", "10"));
    check(a.at("price_nanos") != "100000000000");
  }
  check(outcomes[0].at("fill_probability").at("status") == "unavailable");
  check(outcomes[1].at("status") == "unavailable" &&
        outcomes[1].at("support").empty());
  check(outcomes[1].at("observed_candidates").size() == 5);
  check(sections.at("studies").at("data")[1].at("available") == "1");
  const auto &excursion = sections.at("studies").at("data")[2].at("results")[0];
  check(excursion.at("minimum").at("value") == "-5000000000" &&
        excursion.at("minimum").at("source_ordinal") == "1");
  check(excursion.at("maximum").at("value") == "5000000000" &&
        excursion.at("maximum").at("source_ordinal") == "2");
  check(sections.at("signals").at("data") == p.at("census").at("signals"));
  check(sections.at("summary").at("data").at("external_causality_verified") ==
        false);
  check(sections.at("provenance")
            .at("data")
            .at("census_producer")
            .at("reproducibility") == "nondeterministic");
  auto run = [&](J request) {
    static unsigned i = 0;
    const auto file = root + "/result-" + std::to_string(++i) + ".json";
    request["output_path"] = file;
    call("evaluate", request);
    return read(file);
  };
  auto reject = [&](J request) {
    static unsigned i = 0;
    const auto file = root + "/invalid-" + std::to_string(++i) + ".json";
    request["output_path"] = file;
    bool failed = false;
    try {
      call("evaluate", request);
    } catch (...) {
      failed = true;
    }
    check(failed && !std::filesystem::exists(file));
  };
  auto q = p;
  q["workers"] = "4";
  const auto threaded = run(q);
  for (const char *name :
       {"signals", "execution", "replay", "studies", "summary"})
    check(threaded.at("sections").at(name) == sections.at(name));
  q = p;
  q["model"]["parameters"]["include_anchor"] = true;
  auto included = run(q).at("sections").at("execution").at("data")[0];
  check(included.at("support").size() == 11 &&
        included.at("support")[5].at("price_nanos") == "100000000000");
  check(included.at("support")[5].at("source_ordinal") == "0" &&
        included.at("support")[5].at("weight") == rat("1", "11"));
  q = p;
  q["model"]["parameters"]["thin_support"] = "reject";
  reject(q);
  q = p;
  q["census"]["signals"][0]["causal_end_ordinal_exclusive"] = "12";
  reject(q);
  q["census"]["mode"] = "retrospective";
  check(run(q).at("sections").at("summary").at("data").at("causality") ==
        "retrospective");
  q = p;
  q["census"]["signals"][1]["signal_id"] = "signal/a~B";
  reject(q);
  q = p;
  q["census"]["signals"][0]["available_ns"] = "1";
  reject(q);
  q = p;
  q["census"]["source_sha256"] = std::string(64, '0');
  reject(q);
  q = p;
  std::swap(q["census"]["signals"][0], q["census"]["signals"][1]);
  reject(q);
  q = p;
  q["census"]["signals"][0]["anchor_price_nanos"] = "-9223372036854775808";
  check(run(q)
            .at("sections")
            .at("studies")
            .at("data")[2]
            .at("results")[0]
            .at("minimum")
            .at("status") == "unavailable");
  q = p;
  q["census"]["signals"] = J::array();
  q["studies"] = J::array();
  const auto empty = run(q);
  check(empty.at("sections").at("studies").at("status") == "not_selected" &&
        empty.at("sections").at("resources").at("data").at("actual_workers") ==
            "0");
  J supplied = J::array();
  for (const auto &sig : p.at("census").at("signals"))
    supplied.push_back(
        {{"signal_id", sig.at("signal_id")},
         {"support",
          J::array(
              {{{"price_nanos", "99000000000"}, {"weight", rat("1", "2")}},
               {{"price_nanos", "101000000000"}, {"weight", rat("1", "2")}}})},
         {"execution_probability",
          {{"status", "supplied"}, {"value", rat("3", "4")}}},
         {"evidence_reference", "external://fit-v1"}});
  auto external = p;
  external["model"]["id"] = "external_outcomes";
  external["model"]["parameters"] = {
      {"producer", producer},
      {"measure", "probability"},
      {"conditioning", "execution"},
      {"calibration_reference", "external://unverified-calibration"},
      {"outcomes", supplied}};
  const auto modeled = run(external);
  check(modeled.at("sections").at("summary").at("data").at("census_sha256") ==
        sections.at("summary").at("data").at("census_sha256"));
  const auto &modeled_row =
      modeled.at("sections").at("execution").at("data")[0];
  check(modeled_row.at("no_fill_probability").at("value") == rat("1", "4") &&
        modeled_row.at("calibration") == "not_verified");
  q = external;
  q["model"]["parameters"]["outcomes"][0]["support"][0]["weight"] =
      rat("1", "3");
  reject(q);
  q = external;
  q["model"]["parameters"]["outcomes"][0]["execution_probability"]["value"] =
      rat("5", "4");
  reject(q);
  q = external;
  q["model"]["parameters"]["outcomes"][0]["signal_id"] = "unknown";
  reject(q);
  q = external;
  q["model"]["parameters"]["outcomes"][1]["signal_id"] = "signal/a~B";
  reject(q);
  q = external;
  q["model"]["version"] = "2";
  reject(q);
  auto signed_model = external;
  signed_model["model"]["parameters"]["measure"] = "signed_coefficient";
  signed_model["model"]["parameters"]["conditioning"] = "scenario";
  for (auto &o : signed_model["model"]["parameters"]["outcomes"]) {
    o["execution_probability"] = {{"status", "unavailable"},
                                  {"reason", "not a probability experiment"}};
    o["support"][0]["weight"] = rat("-1", "2");
  }
  const auto signed_result =
      run(signed_model).at("sections").at("execution").at("data")[0];
  check(signed_result.at("support")[0].at("weight") == rat("-1", "2") &&
        signed_result.at("measure") == "signed_coefficient");
  check(signed_result.at("fill_probability").at("reason") ==
        "not a probability experiment");
  q = signed_model;
  q["model"]["parameters"]["measure"] = "probability";
  reject(q);
  q = signed_model;
  q["model"]["parameters"]["measure"] = "scenario_weight";
  reject(q);
  q = signed_model;
  q["model"]["parameters"]["conditioning"] = "execution";
  for (auto &o : q["model"]["parameters"]["outcomes"]) {
    o["execution_probability"] = {{"status", "supplied"},
                                  {"value", rat("3", "4")}};
    o["support"] = J::array(
        {{{"price_nanos", "99000000000"}, {"weight", rat("-1", "999999937")}},
         {{"price_nanos", "100000000000"}, {"weight", rat("1", "999999929")}},
         {{"price_nanos", "101000000000"}, {"weight", rat("1", "999999893")}}});
  }
  const auto separate = run(q).at("sections").at("execution").at("data")[0];
  check(separate.at("measure") == "signed_coefficient" &&
        separate.at("support")[0].at("weight") == rat("-1", "999999937"));
  check(separate.at("fill_probability").at("value") == rat("3", "4"));
  auto catalogue =
      call("catalogue", {{"protocol", "symphony.sbv.catalogue-input.v1"}});
  check(catalogue.at("models").size() == 6 &&
        catalogue.at("studies").size() == 11);
  J joint{
      {"protocol", "symphony.sbv.compose-joint-input.v1"},
      {"output_path", root + "/joint.json"},
      {"extensions", J::object()},
      {"paths",
       J::array({{{"path_id", "mixed-a"},
                  {"returns", J::array({rat("-1", "100"), rat("1", "100")})},
                  {"probability", rat("1", "2")}},
                 {{"path_id", "mixed-b"},
                  {"returns", J::array({rat("1", "100"), rat("-1", "100")})},
                  {"probability", rat("1", "2")}}})}};
  call("compose_joint", joint);
  const auto atoms = read(root + "/joint.json")
                         .at("sections")
                         .at("distributions")
                         .at("data")
                         .at("terminal_atoms");
  check(atoms.size() == 1 && atoms[0].at("wealth") == rat("9999", "10000") &&
        atoms[0].at("mass") == rat("1", "1"));
  joint["paths"][0]["probability"] = rat("1", "3");
  joint["output_path"] = root + "/bad-joint.json";
  bool refused = false;
  try {
    call("compose_joint", joint);
  } catch (...) {
    refused = true;
  }
  check(refused && !std::filesystem::exists(root + "/bad-joint.json"));
  J query{{"protocol", "symphony.sbv.result-query-input.v1"},
          {"path", root + "/first.json"},
          {"expected_sha256", receipt.at("content_sha256")},
          {"pointer", "/sections/execution/data/0/support/0"},
          {"limit", "10"},
          {"cursor", ""}};
  check(call("result_query", query).at("total") == "4");
  std::cout << checks << " SBV model/census/joint contract assertions passed\n";
  return 0;
} catch (const std::exception &x) {
  std::cerr << x.what() << '\n';
  return 1;
}
