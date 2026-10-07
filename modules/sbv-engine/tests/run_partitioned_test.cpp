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
    throw std::runtime_error("partitioned run check line " +
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
store::Reference ref(const J &receipt) {
  const auto &r = receipt.at("storage").at("reference");
  return {r.at("manifest_path"), r.at("manifest_sha256"),
          r.at("content_sha256")};
}
J whole(const J &receipt) {
  store::ResultReader reader(ref(receipt));
  return reader.select_node(0).read_value();
}
J science(J value) {
  value.erase("content_sha256");
  value["sections"]["resources"]["data"].erase("derived_resources");
  return value;
}
int main() try {
#if defined(__APPLE__)
  char temp[] = "/private/tmp/sbv-run-partitioned-XXXXXX";
#else
  char temp[] = "/tmp/sbv-run-partitioned-XXXXXX";
#endif
  check(::mkdtemp(temp));
  const std::string root = temp;
  auto make = [&](const std::string &name, std::uint64_t count) {
    const auto raw = fixture(count), path = root + "/" + name + ".dbn";
    d::create_file(path, raw, e::no_deadline);
    return J{
        {"protocol", "symphony.sbv.run-input.v1"},
        {"source_path", path},
        {"source_sha256", e::sha256_hex(raw)},
        {"dataset", "GLBX.MDP3"},
        {"criteria",
         {{"rule", "spaced_trades"},
          {"direction", "any"},
          {"spacing_ns", "0"},
          {"min_trade_size", "1"},
          {"max_signals", nullptr}}},
        {"execution",
         {{"model", "user_probability"},
          {"horizon_ns", "250"},
          {"price_offsets_nanos", J::array({"-1000000000", "0", "1000000000"})},
          {"probability_numerator", "1"},
          {"probability_denominator", "3"}}},
        {"replay",
         {{"before_ns", "150"}, {"after_ns", "175"}, {"retain_events", true}}},
        {"studies", J::array({"signal_summary", "forward_markout"})},
        {"workers", "3"},
        {"extensions", {{"user-selected", "preserved"}}}};
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
  auto parity = [&](J p, const std::string &name) {
    auto legacy = p;
    legacy["output_path"] = root + "/" + name + "-legacy.json";
    check(d::run(legacy, e::no_deadline).at("status") == "partial");
    const auto expected = read(legacy.at("output_path"));
    s::validate_result(expected);
    d::validate_census_evidence(
        expected.at("sections").at("census").at("data"));
    const auto receipt = d::run(output(p, name), e::no_deadline);
    check(receipt.at("status") == "partial");
    auto actual = whole(receipt);
    check(science(actual) == science(expected));
    check(
        actual.at("sections").at("census").at("data").at("census_sha256") ==
        e::sha256_hex(expected.at("sections").at("signals").at("data").dump()));
    return actual;
  };
  const auto p = make("small", 12);
  const auto baseline = parity(p, "uncapped");
  const auto &sections = baseline.at("sections");
  check(sections.at("summary").at("data").at("signal_count") == "12" &&
        sections.at("summary").at("data").at("selection_cap").is_null() &&
        sections.at("summary").at("data").at("additional_eligible_after_cap") ==
            "0");
  // Independent arithmetic oracle for the first 100->101->102 price sequence.
  const auto &first = sections.at("execution").at("data").at(0);
  check(first.at("outcomes").at(0).at("target_price_nanos") == "99000000000" &&
        first.at("outcomes").at(0).at("observed_trade_touch") == false);
  for (std::size_t i : {1, 2})
    check(first.at("outcomes").at(i).at("touch_source_ordinal") == "1" &&
          first.at("outcomes").at(i).at("touch_latency_ns") == "100");
  check(first.at("outcomes").at(1).at("fill_probability").at("numerator") ==
            "1" &&
        first.at("outcomes").at(1).at("no_fill_probability").at("numerator") ==
            "2");
  const auto &markout =
      sections.at("studies").at("data").at(1).at("results").at(0).at(
          "price_change");
  check(markout.at("value") == "2000000000" &&
        markout.at("mark_source_ordinal") == "2");
  check(sections.at("replay").at("data").at("retained_event_count") == "12");
  auto capped = p;
  capped["criteria"]["max_signals"] = "2";
  capped["criteria"]["spacing_ns"] = "300";
  const auto capped_value = parity(capped, "capped");
  check(capped_value.at("sections")
            .at("signals")
            .at("data")
            .at(1)
            .at("source_ordinal") == "3");
  check(capped_value.at("sections")
            .at("summary")
            .at("data")
            .at("additional_eligible_after_cap") == "6");
  auto directional = p;
  directional["criteria"]["rule"] = "trade_direction";
  directional["criteria"]["direction"] = "up";
  directional["criteria"]["min_trade_size"] = "3";
  directional["replay"]["retain_events"] = false;
  auto directional_value = parity(directional, "directional");
  J ordinals = J::array();
  for (const auto &signal :
       directional_value.at("sections").at("signals").at("data"))
    ordinals.push_back(signal.at("source_ordinal"));
  check(ordinals == J::array({"2", "4", "7", "8"}));
  check(directional_value.at("sections")
            .at("replay")
            .at("data")
            .at("events")
            .empty());
  auto none = p;
  none["execution"] = {{"model", "none"},
                       {"horizon_ns", "0"},
                       {"price_offsets_nanos", J::array()},
                       {"probability_numerator", "0"},
                       {"probability_denominator", "1"}};
  none["studies"] = J::array();
  const auto none_value = parity(none, "none");
  check(none_value.at("sections").at("execution").at("status") ==
            "not_selected" &&
        none_value.at("sections").at("execution").at("data").size() == 12);
  auto zero = none;
  zero["criteria"]["min_trade_size"] = "4294967295";
  const auto zero_value = parity(zero, "zero");
  check(zero_value.at("sections").at("summary").at("data").at("signal_count") ==
            "0" &&
        zero_value.at("sections")
                .at("resources")
                .at("data")
                .at("actual_workers") == "1");
  auto serial = p;
  serial["workers"] = "1";
  auto serial_value = whole(d::run(output(serial, "serial"), e::no_deadline));
  for (const auto *name :
       {"summary", "signals", "execution", "replay", "studies", "census"})
    check(serial_value.at("sections").at(name) ==
          baseline.at("sections").at(name));
  auto resident = d::load_dataset(p, e::no_deadline);
  auto resident_value =
      whole(d::run(output(p, "resident"), e::no_deadline, resident.get()));
  check(resident_value.at("sections")
            .at("resources")
            .at("data")
            .at("dataset_feed")
            .at("source_reads_this_job") == "0");
  for (const auto *name :
       {"summary", "signals", "execution", "replay", "studies", "census"})
    check(resident_value.at("sections").at(name) ==
          baseline.at("sections").at(name));
  auto large_request = make("large", 5000);
  large_request["execution"] = none.at("execution");
  large_request["replay"]["before_ns"] = "0";
  large_request["replay"]["after_ns"] = "0";
  large_request["studies"] = J::array({"signal_summary"});
  const auto large_receipt =
      d::run(output(large_request, "large"), e::no_deadline);
  check(large_receipt.at("status") == "partial" &&
        large_receipt.at("summary").at("data").at("signal_count") == "5000");
  store::ResultReader large(ref(large_receipt), {std::nullopt, 0});
  auto rows = large.select("/sections/signals/data").children();
  std::uint64_t number = 0;
  while (auto row = rows.next()) {
    const auto signal = row->read_value();
    if (signal.at("signal_id") != "signal-" + std::to_string(number) ||
        signal.at("source_ordinal") != std::to_string(number))
      throw std::runtime_error("large census order mismatch");
    ++number;
  }
  check(number == 5000 &&
        large.select("/sections/execution/data").describe().at("children") ==
            "5000");
  check(
      large.select("/sections/replay/data/events").describe().at("children") ==
      "5000");
  auto census = d::stream_census_evidence(
      s::logical::Value(large.select_node(0)), e::no_deadline);
  check(census.at("signals").size() == 5000);
  // A selected cap above the former ceiling is valid in both representations.
  auto positive = large_request;
  positive["criteria"]["max_signals"] = "4500";
  positive["output_path"] = root + "/large-legacy.json";
  check(d::run(positive, e::no_deadline).at("summary").at("data").at("signal_count") ==
        "4500");
  const auto legacy_large = read(positive.at("output_path"));
  d::validate_census_evidence(
      legacy_large.at("sections").at("census").at("data"));
  check(legacy_large.at("sections")
            .at("summary")
            .at("data")
            .at("additional_eligible_after_cap") == "500");
  std::filesystem::remove_all(root + "/large-work");
  check(large.verify_closure().at("verification_extent") ==
        "full_logical_closure");
  auto bad = p;
  bad["execution"]["price_offsets_nanos"] = J::array({"9223372036854775807"});
  const auto failed = d::run(output(bad, "overflow"), e::no_deadline);
  check(failed.at("status") == "recovery_required" &&
        failed.at("cause").at("code") == "sbv.contract");
  check(!std::filesystem::exists(root + "/overflow-result") &&
        std::filesystem::exists(root +
                                "/overflow-work/census-signals/manifest.json"));
  auto tampered = baseline.at("sections").at("census").at("data");
  tampered["declaration"]["additional_eligible_after_cap"] = "1";
  bool rejected = false;
  try {
    d::validate_census_evidence(tampered);
  } catch (...) {
    rejected = true;
  }
  check(rejected);
  std::cout << "partitioned run checks: " << checks
            << "\nretained fixtures: " << root << '\n';
  return 0;
} catch (const std::exception &error) {
  std::cerr << error.what() << '\n';
  return 1;
}
