#include "../../sqav-databento-dbn-cpp/tests/public_fixture.hpp"
#include "../src/dataset.hpp"
#include "../src/stream_census.hpp"
#include <filesystem>
#include <fstream>
#include <iostream>
#include <source_location>
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
store::Reference ref(const J &receipt) {
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
  char temp[] = "/private/tmp/sbv-generate-partitioned-XXXXXX";
#else
  char temp[] = "/tmp/sbv-generate-partitioned-XXXXXX";
#endif
  check(::mkdtemp(temp));
  const std::string root = temp;
  const auto library = std::filesystem::absolute(argv[1]).string();
  auto make = [&](const std::string &name, std::uint64_t count) {
    const auto raw = fixture(count), path = root + "/" + name + ".dbn";
    d::create_file(path, raw, e::no_deadline);
    return J{{"protocol", "symphony.sbv.generate-census-input.v1"},
             {"source_path", path},
             {"source_sha256", e::sha256_hex(raw)},
             {"dataset", "GLBX.MDP3"},
             {"provider", provider(library)},
             {"extensions", {{"owned-by-user", "preserved"}}}};
  };
  auto output = [&](J p, const std::string &name) {
    p.erase("output_path");
    p["output"] = {
        {"kind", "partitioned"},
        {"workspace_path", root + "/" + name + "-work"},
        {"bundle_path", root + "/" + name + "-result"},
        {"write_options", {{"page_bytes", "65536"}, {"index_fanout", "8"}}}};
    return p;
  };
  const auto small_request = make("small", 12);
  auto legacy_request = small_request;
  legacy_request["output_path"] = root + "/legacy.json";
  const auto legacy_receipt =
      d::generate_census(legacy_request, e::no_deadline);
  const auto legacy = read(root + "/legacy.json");
  s::validate_result(legacy);
  const auto partitioned =
      d::generate_census(output(small_request, "small"), e::no_deadline);
  check(partitioned.at("status") == "partial" &&
        legacy_receipt.at("status") == "partial");
  auto selected =
      d::select_logical_result(bundle_source(partitioned), e::no_deadline);
  auto value = selected.value.materialize();
  auto expected = legacy;
  value.erase("content_sha256");
  expected.erase("content_sha256");
  value["sections"]["resources"]["data"].erase("derived_resources");
  check(value == expected);
  auto census = d::stream_census_evidence(selected.value, e::no_deadline);
  check(census.at("census_sha256").materialize() ==
        legacy.at("sections").at("census").at("data").at("census_sha256"));
  check(census.at("signals").size() == 12);
  check(census.at("declaration")
            .at("completion")
            .at("extensions")
            .at("events")
            .materialize() == "12");
  auto original = d::load_dataset(small_request, e::no_deadline);
  d::validate_stream_census_source(census, *original, e::no_deadline);
  const auto resident_receipt = d::generate_census(
      output(small_request, "resident"), e::no_deadline, original.get());
  store::ResultReader resident(ref(resident_receipt));
  check(resident.select("/sections/summary/data/census_sha256").read_value() ==
        census.at("census_sha256").materialize());
  const auto feed =
      resident.select("/sections/resources/data/dataset_feed").read_value();
  check(feed.at("mode") == "resident" &&
        feed.at("source_reads_this_job") == "0");

  const auto large_request = make("large", 5000);
  const auto large =
      d::generate_census(output(large_request, "large"), e::no_deadline);
  check(large.at("status") == "partial" &&
        large.at("summary").at("data").at("signal_count") == "5000");
  store::ResultReader reader(ref(large), {std::nullopt, 0});
  auto signals = reader.select("/sections/signals/data");
  check(signals.describe().at("children") == "5000");
  auto cursor = signals.children();
  std::uint64_t row_count = 0;
  while (auto row = cursor.next()) {
    const auto signal = row->read_value();
    if (signal.at("signal_id") !=
            "provider-" + std::to_string(row_count) + "-0" ||
        signal.at("source_ordinal") != std::to_string(row_count) ||
        signal.at("causal_end_ordinal_exclusive") !=
            std::to_string(row_count + 1))
      throw std::runtime_error("large generated census membership changed");
    ++row_count;
  }
  check(row_count == 5000);
  check(
      reader
          .select(
              "/sections/census/data/declaration/completion/extensions/events")
          .read_value() == "5000");
  const auto resources =
      reader.select("/sections/resources/data/derived_resources").read_value();
  check(resources.at("identity_index").at("entries") == "5000" &&
        resources.at("identity_index").at("max_tracked_bytes").is_null() &&
        resources.at("identity_index").at("spill") == "not_supported");
  check(resources.at("signal_rows").at("aggregate_array_materialized") ==
        false);
  std::filesystem::remove_all(root + "/large-work");
  check(reader.verify_closure().at("verification_extent") ==
        "full_logical_closure");
  auto large_selected =
      d::select_logical_result(bundle_source(large), e::no_deadline);
  check(d::stream_census_evidence(large_selected.value, e::no_deadline)
            .at("signals")
            .size() == 5000);

  auto zero_request = small_request;
  zero_request["provider"]["parameters"]["emissions_per_event"] = "0";
  const auto zero =
      d::generate_census(output(zero_request, "zero"), e::no_deadline);
  check(zero.at("summary").at("data").at("signal_count") == "0");
  store::ResultReader zero_reader(ref(zero));
  check(zero_reader.select("/sections/signals/data").read_value() ==
        J::array());
  auto failure_request = large_request;
  failure_request["provider"]["parameters"]["fail_after"] = "100";
  const auto failure =
      d::generate_census(output(failure_request, "failed"), e::no_deadline);
  check(failure.at("status") == "recovery_required" &&
        failure.at("cause").at("code") == "sbv.contract" &&
        failure.at("recovery").at("phase") == "produce");
  check(!std::filesystem::exists(root + "/failed-result") &&
        !std::filesystem::exists(root +
                                 "/failed-work/census-signals/manifest.json"));
  auto finish_request = large_request;
  finish_request["provider"]["parameters"]["finish_failure"] = "true";
  const auto failed_finish = d::generate_census(
      output(finish_request, "failed-finish"), e::no_deadline);
  check(failed_finish.at("status") == "recovery_required" &&
        failed_finish.at("cause").at("code") == "sbv.contract" &&
        failed_finish.at("recovery").at("phase") == "produce");
  check(!std::filesystem::exists(root + "/failed-finish-result") &&
        !std::filesystem::exists(
            root + "/failed-finish-work/census-signals/manifest.json"));
  auto duplicate_request = small_request;
  duplicate_request["provider"]["parameters"]["mode"] = "duplicate_id";
  const auto duplicate = d::generate_census(
      output(duplicate_request, "duplicate"), e::no_deadline);
  check(duplicate.at("status") == "recovery_required" &&
        !std::filesystem::exists(root + "/duplicate-result"));
  std::cout << J{{"status", "passed"},
                 {"checks", d::dec(checks)},
                 {"scratch", root},
                 {"large_receipt", large},
                 {"failure", failure}}
                   .dump()
            << '\n';
  return 0;
} catch (const std::exception &error) {
  std::cerr << error.what() << '\n';
  return 1;
}
