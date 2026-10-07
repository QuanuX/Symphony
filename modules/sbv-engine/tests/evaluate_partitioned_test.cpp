#include "../../sqav-databento-dbn-cpp/tests/public_fixture.hpp"
#include "../src/dataset.hpp"
#include "../src/stream_census.hpp"
#include <filesystem>
#include <fstream>
#include <iostream>
#include <source_location>
#include <sys/stat.h>
#include <unistd.h>

namespace s = symphony::sbv;
namespace d = s::detail;
namespace e = symphony::knowledge::engine;
namespace store = s::result_store;
using J = s::Json;
unsigned checks = 0;
void check(bool yes,
           std::source_location at = std::source_location::current()) {
  ++checks;
  if (!yes)
    throw std::runtime_error("partitioned generation check line " +
                             std::to_string(at.line()));
}
J read(const std::string &path) {
  std::ifstream file(path);
  J result;
  file >> result;
  return result;
}
std::string fixture(std::uint64_t count) {
  const auto source = public_fixture::fixture();
  std::vector<unsigned char> bytes(source.begin(), source.begin() + 360);
  auto put = [&](std::size_t offset, std::uint64_t value, unsigned length) {
    for (unsigned i = 0; i < length; ++i)
      bytes[offset + i] = static_cast<unsigned char>(value >> (i * 8));
  };
  // Independently encoded mechanical MBO sequence; not market-performance data.
  // Header selected record limit is zero; every emitted record is preserved.
  put(42, 0, 8);
  for (std::uint64_t i = 0; i < count; ++i) {
    const auto at = bytes.size();
    bytes.insert(bytes.end(), source.begin() + 360, source.begin() + 416);
    put(at + 8, 1609160400000000000ULL + i * 100, 8);
    put(at + 16, i + 100, 8);
    put(at + 24, (100 + i % 3) * 1000000000ULL, 8);
    put(at + 32, 1 + i % 5, 4);
    bytes[at + 36] = 128;
    bytes[at + 37] = 3;
    bytes[at + 38] = 'T';
    bytes[at + 39] = i % 2 ? 'A' : 'B';
    put(at + 40, 1609160400000000000ULL + i * 100 + 17, 8);
    put(at + 48, 17, 4);
    put(at + 52, i + 1, 4);
  }
  return std::string(reinterpret_cast<const char *>(bytes.data()),
                     bytes.size());
}
J provider(const std::string &library) {
  return {
      {"protocol", "symphony.sbv.native-provider-selection.v1"},
      {"library",
       {{"path", library},
        {"expected_sha256",
         e::sha256_hex(d::read_file(library, e::no_deadline))}}},
      {"id", "sbv-test-provider"},
      {"version", "1"},
      {"role", "strategy"},
      {"input_profile", "symphony.sbv.provider-databento-mbo-event.v1"},
      {"concurrency", "serialized_instance"},
      {"parameters",
       {{"action", "T"}, {"emit_every", "1"}, {"emissions_per_event", "1"}}},
      {"dependencies",
       {{"capture", "uncaptured"},
        {"description", "independent fixture"},
        {"artifacts", J::array()}}},
      {"extensions", J::object()}};
}
store::Reference receipt_reference(const J &receipt) {
  const auto &r = receipt.at("storage").at("reference");
  return {r.at("manifest_path"), r.at("manifest_sha256"),
          r.at("content_sha256")};
}
J bundle_source(const J &receipt) {
  return {
      {"kind", "bundle"},
      {"reference", receipt.at("storage").at("reference")},
      {"selector", {{"kind", "node_id"}, {"node_id", "0"}}},
      {"read_options", {{"max_page_bytes", nullptr}, {"cache_bytes", "0"}}}};
}
int main(int argc, char **argv) try {
  check(argc == 2);
#if defined(__APPLE__)
  char temp[] = "/private/tmp/sbv-evaluate-partitioned-XXXXXX";
#else
  char temp[] = "/tmp/sbv-evaluate-partitioned-XXXXXX";
#endif
  check(::mkdtemp(temp));
  const std::string root = temp,
                    library = std::filesystem::absolute(argv[1]).string();
  const auto raw = fixture(50), file = root + "/source.dbn";
  d::create_file(file, raw, e::no_deadline);
  J generate{{"protocol", "symphony.sbv.generate-census-input.v1"},
             {"source_path", file},
             {"source_sha256", e::sha256_hex(raw)},
             {"dataset", "GLBX.MDP3"},
             {"provider", provider(library)},
             {"output_path", root + "/census.json"},
             {"extensions", J::object()}};
  auto old_census = d::generate_census(generate, e::no_deadline);
  auto output = [&](J p, const std::string &name) {
    p.erase("output_path");
    p["output"] = {
        {"kind", "partitioned"},
        {"bundle_path", root + "/" + name},
        {"workspace_path", root + "/" + name + "-work"},
        {"write_options", {{"page_bytes", "8192"}, {"index_fanout", "8"}}}};
    return p;
  };
  auto generated =
      d::generate_census(output(generate, "census"), e::no_deadline);
  check(generated.at("status") == "partial");
  J evaluate{
      {"protocol", "symphony.sbv.evaluate-input.v1"},
      {"source_path", file},
      {"source_sha256", e::sha256_hex(raw)},
      {"dataset", "GLBX.MDP3"},
      {"census",
       {{"path", old_census.at("path")},
        {"expected_sha256", old_census.at("content_sha256")},
        {"pointer", ""}}},
      {"model",
       {{"protocol", "symphony.sbv.model-selection.v1"},
        {"id", "observed_trade_levels"},
        {"version", "1"},
        {"horizon_ns", "700"},
        {"parameters",
         {{"levels_per_side", "1"},
          {"include_anchor", true},
          {"thin_support", "unavailable"}}}}},
      {"replay",
       {{"before_ns", "250"}, {"after_ns", "950"}, {"retain_events", true}}},
      {"studies",
       J::array({"signal_summary", "model_summary", "path_excursion"})},
      {"workers", "3"},
      {"output_path", root + "/legacy.json"},
      {"extensions", {{"custom", "preserved"}}}};
  auto legacy_receipt = d::evaluate(evaluate, e::no_deadline);
  auto legacy = read(legacy_receipt.at("path"));
  auto export_result = [&](const J &receipt) {
    store::ResultReader reader(receipt_reference(receipt));
    reader.verify_closure();
    return reader.select("").read_value();
  };
  auto partitioned =
      d::evaluate(output(evaluate, "evaluation"), e::no_deadline);
  check(partitioned.at("status") == "partial");
  auto actual = export_result(partitioned);
  for (auto it = legacy.at("sections").begin();
       it != legacy.at("sections").end(); ++it)
    if (it.key() != "resources")
      check(actual.at("sections").at(it.key()) == it.value());
  check(actual.at("sections").at("resources").at("data").at("actual_workers") ==
        "3");
  check(
      actual.at("sections").at("resources").at("data").at("replay_intervals") ==
      "1");
  // Equivalent science from the partitioned census and resident decoded feed.
  auto retained_request = evaluate;
  retained_request["census"] = bundle_source(generated);
  auto resident = d::load_dataset(retained_request, e::no_deadline);
  auto resident_receipt = d::evaluate(output(retained_request, "resident"),
                                      e::no_deadline, resident.get());
  check(resident_receipt.at("status") == "partial");
  auto resident_result = export_result(resident_receipt);
  for (auto key :
       {"signals", "census", "execution", "replay", "studies", "summary"})
    check(resident_result.at("sections").at(key) ==
          actual.at("sections").at(key));
  check(resident_result.at("sections")
            .at("resources")
            .at("data")
            .at("dataset_feed")
            .at("source_reads_this_job") == "0");
  // A second independently loaded native model receives the same closed census.
  auto model = provider(library);
  model["role"] = "model";
  model["concurrency"] = "per_worker_instances";
  auto native = retained_request;
  native["model"] = {{"protocol", "symphony.sbv.model-selection.v1"},
                     {"id", "native_provider"},
                     {"version", "1"},
                     {"horizon_ns", "700"},
                     {"parameters",
                      {{"provider", model},
                       {"measure", "probability"},
                       {"conditioning", "execution"},
                       {"calibration_reference", "fixture supplied assumption"},
                       {"on_unavailable", "unavailable"}}}};
  auto native_receipt =
      d::evaluate(output(native, "native-model"), e::no_deadline);
  check(native_receipt.at("status") == "partial");
  auto native_result = export_result(native_receipt);
  check(native_result.at("sections").at("census") ==
        actual.at("sections").at("census"));
  check(native_result.at("sections").at("signals") ==
        actual.at("sections").at("signals"));
  check(native_result.at("sections")
            .at("resources")
            .at("data")
            .at("provider_instances") == "3");
  auto empty_replay = retained_request;
  empty_replay["replay"] = {
      {"before_ns", "0"}, {"after_ns", "0"}, {"retain_events", false}};
  empty_replay["studies"] = J::array();
  auto empty_result = export_result(
      d::evaluate(output(empty_replay, "no-events"), e::no_deadline));
  check(
      empty_result.at("sections").at("replay").at("data").at("events").empty());
  check(empty_result.at("sections")
            .at("replay")
            .at("data")
            .at("windows")
            .size() == 50);
  auto ratio = [](const char *n, const char *den = "1") {
    return J{{"numerator", n}, {"denominator", den}};
  };
  J transform{{"id", "linear_price_pnl"},
              {"version", "1"},
              {"reference_price_nanos", "100000000000"},
              {"price_unit_nanos", "1000000000"},
              {"position_units", ratio("1")},
              {"value_per_price_unit", ratio("1")},
              {"cost_per_outcome", ratio("0")},
              {"return_basis", ratio("100")},
              {"pnl_unit", "fixture-unit"}};
  J econ{{"protocol", "symphony.sbv.economics-input.v1"},
         {"source", bundle_source(native_receipt)},
         {"selections",
          {{"kind", "apply"},
           {"signals", {{"kind", "range"}, {"first", "0"}, {"count", "2"}}},
           {"economics",
            {{"transform", transform},
             {"mode", "execution_mixture"},
             {"nonexecution_pnl", ratio("0")}}}}},
         {"studies", J::array()},
         {"on_incompatible", "reject"},
         {"extensions", J::object()}};
  auto econ_receipt = d::economics(output(econ, "economics"), e::no_deadline);
  check(econ_receipt.at("status") == "completed");
  auto econ_result = export_result(econ_receipt);
  check(econ_result.at("sections").at("economics").at("data").size() == 2);
  auto components = J::array();
  for (unsigned i = 0; i < 2; ++i)
    components.push_back(
        {{"id", std::to_string(i)},
         {"source", bundle_source(econ_receipt)},
         {"outcome_pointer", "/sections/economics/data/" + std::to_string(i)},
         {"signal_id", native_result.at("sections")
                           .at("signals")
                           .at("data")[i]
                           .at("signal_id")},
         {"conditioning", "execution_and_nonexecution"},
         {"conversion", nullptr},
         {"extensions", J::object()}});
  J compose{
      {"protocol", "symphony.sbv.compose-economics-input.v1"},
      {"components", components},
      {"dependence",
       {{"kind", "independent"},
        {"marginal_policy", nullptr},
        {"paths", J::array()},
        {"description", "fixture assumption"}}},
      {"economics",
       {{"kind", "multiplicative_return"},
        {"initial_state", ratio("1")},
        {"state_unit", "factor"},
        {"state_domain", "signed"},
        {"description", "fixture arithmetic"}}},
      {"conditioning_description", "fixture independent mixture"},
      {"on_unavailable", "reject"},
      {"retain_paths", true},
      {"retain_prefix_distributions", true},
      {"studies", J::array()},
      {"limits", {{"max_paths", nullptr}, {"max_terminal_atoms", nullptr}}},
      {"extensions", J::object()}};
  auto composed_receipt =
      d::compose_economics(output(compose, "composition"), e::no_deadline);
  check(composed_receipt.at("status") == "completed");
  const auto composed = export_result(composed_receipt);
  check(composed.at("sections").at("summary").at("data").at("path_count") ==
        "9");
  check(composed.at("sections")
            .at("distributions")
            .at("data")
            .at("prefix_distributions")
            .size() == 3);
  check(composed.at("sections")
            .at("composition_sources")
            .at("data")[0]
            .at("replay") == native_result.at("sections").at("replay"));
  // Exporting the same logical economics to a file or embedding it must not
  // change its financial interpretation or force ancestor reopening.
  auto economic_file = root + "/economic-export.json";
  d::create_file(economic_file, econ_result.dump(), e::no_deadline);
  auto outer = d::base("embedded economic result fixture");
  outer["sections"]["retained"] = d::section(econ_result);
  outer = s::seal_result(outer);
  auto embedded_file = root + "/embedded-economic.json";
  d::create_file(embedded_file, outer.dump(), e::no_deadline);
  for (unsigned representation = 0; representation < 2; ++representation) {
    auto alternate = compose;
    for (auto &component : alternate["components"])
      component["source"] = {
          {"path", representation ? embedded_file : economic_file},
          {"expected_sha256", representation
                                  ? outer.at("content_sha256")
                                  : econ_result.at("content_sha256")},
          {"pointer", representation ? "/sections/retained/data" : ""}};
    const auto receipt = d::compose_economics(
        output(alternate,
               "composition-export-" + std::to_string(representation)),
        e::no_deadline);
    check(receipt.at("status") == "completed");
    const auto result = export_result(receipt);
    for (auto key : {"summary", "distributions", "studies"})
      check(result.at("sections").at(key) == composed.at("sections").at(key));
  }
  auto wrong_row = compose;
  wrong_row["components"][0]["outcome_pointer"] = "";
  auto wrong_receipt = d::compose_economics(
      output(wrong_row, "whole-result-as-row"), e::no_deadline);
  check(wrong_receipt.at("status") == "recovery_required");
  // The actual companion process reuses frozen RAM, including after source
  // removal.
  const auto host_dir = root + "/host";
  check(::mkdir(host_dir.c_str(), 0700) == 0);
  J load{{"protocol", "symphony.sbv.dataset-load-input.v1"},
         {"directory", host_dir},
         {"instance_id", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa01"},
         {"source_path", file},
         {"source_sha256", e::sha256_hex(raw)},
         {"dataset", "GLBX.MDP3"},
         {"memory_budget_bytes", nullptr},
         {"residency", "pageable"},
         {"max_concurrent_jobs", "2"},
         {"worker_budget", "3"},
         {"idle_timeout_ms", "0"}};
  const auto loaded = s::dispatch("dataset_load", load, e::no_deadline);
  check(loaded.at("events") == "50");
  struct ResidentRelease {
    J request;
    bool active = true;
    ~ResidentRelease() {
      if (active) {
        try {
          (void)s::dispatch("dataset_release", request, e::no_deadline);
        } catch (...) {
        }
      }
    }
  } release_guard{J{{"protocol", "symphony.sbv.dataset-release-input.v1"},
                    {"directory", host_dir},
                    {"instance_id", load.at("instance_id")}}};

  auto job_request = retained_request;
  job_request.erase("output_path");
  auto host_output = output(J::object(), "host-evaluate").at("output");
  J job{{"protocol", "symphony.sbv.dataset-execute-input.v1"},
        {"directory", host_dir},
        {"instance_id", load.at("instance_id")},
        {"operation", "evaluate"},
        {"request", job_request},
        {"output", host_output}};
  std::filesystem::remove(file);
  const auto host_receipt = s::dispatch("dataset_execute", job, e::no_deadline);
  check(host_receipt.at("protocol") == "symphony.sbv.dataset-execute.v1" &&
        host_receipt.at("status") == "partial");
  const auto host_result = export_result(host_receipt);
  for (auto key :
       {"signals", "census", "execution", "replay", "studies", "summary"})
    check(host_result.at("sections").at(key) == actual.at("sections").at(key));
  auto bad_job = job;
  bad_job["output"] = output(J::object(), "host-failed").at("output");
  bad_job["request"]["model"]["parameters"]["unexpected"] = "invalid";
  const auto failed = s::dispatch("dataset_execute", bad_job, e::no_deadline);
  check(failed.at("status") == "recovery_required" &&
        failed.at("request_sha256") == e::sha256_hex(bad_job.dump()));
  auto expanded = bad_job.at("request");
  expanded["output"] = bad_job.at("output");
  check(failed.at("child_recovery").at("request_sha256") ==
        e::sha256_hex(expanded.dump()));
  J host_ref{{"protocol", "symphony.sbv.dataset-inspect-input.v1"},
             {"directory", host_dir},
             {"instance_id", load.at("instance_id")}};
  auto status = s::dispatch("dataset_inspect", host_ref, e::no_deadline);
  check(status.at("completed_jobs") == "1" && status.at("failed_jobs") == "1");
  host_ref["protocol"] = "symphony.sbv.dataset-release-input.v1";
  check(s::dispatch("dataset_release", host_ref, e::no_deadline).at("state") ==
        "released");
  release_guard.active = false;
  // Results contain original census and replay after original data/ancestors
  // go.
  std::filesystem::remove(file);
  std::filesystem::remove(generate.at("output_path").get<std::string>());
  std::filesystem::remove_all(root + "/census");
  std::filesystem::remove_all(root + "/resident-work");
  store::ResultReader portable(receipt_reference(resident_receipt));
  portable.verify_closure();
  auto c = d::stream_census_evidence(s::logical::Value(portable.select("")),
                                     e::no_deadline);
  check(c.at("signals").size() == 50);
  for (const auto *name : {"economics", "economics-work", "composition-work",
                           "native-model", "native-model-work"})
    std::filesystem::remove_all(root + "/" + name);
  store::ResultReader portable_composition(receipt_reference(composed_receipt));
  portable_composition.verify_closure();
  check(portable_composition.select("").read_value() == composed);
  std::cout << J{{"checks", std::to_string(checks)},
                 {"status", "passed"},
                 {"directory", root},
                 {"claim_scope", "actual small provider census, alternate "
                                 "model evaluation, RAM and replay parity"}}
                   .dump()
            << "\n";
} catch (const std::exception &ex) {
  std::cerr << ex.what() << "\n";
  return 1;
}
