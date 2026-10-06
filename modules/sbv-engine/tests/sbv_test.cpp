#include "../../sqav-databento-dbn-cpp/tests/public_fixture.hpp"
#include <filesystem>
#include <fstream>
#include <iostream>
#include <source_location>
#include <symphony/knowledge/engine/digest.hpp>
#include <symphony/knowledge/engine/error.hpp>
#include <symphony/knowledge/engine/protocol.hpp>
#include <symphony/sbv/sbv.hpp>
#include <unistd.h>
namespace e = symphony::knowledge::engine;
namespace s = symphony::sbv;
using J = s::Json;
unsigned checks = 0;
void check(bool ok,
           std::source_location loc = std::source_location::current()) {
  ++checks;
  if (!ok)
    throw std::runtime_error("test failure line " + std::to_string(loc.line()));
}
template <class F> void rejects(F f) {
  bool rejected = false;
  try {
    f();
  } catch (...) {
    rejected = true;
  }
  check(rejected);
}
J call(const std::string &op, const J &p) {
  return s::dispatch(op, p, e::unix_time_ms() + 30000);
}
J read(const std::string &path) {
  std::ifstream f(path);
  J j;
  f >> j;
  return j;
}
void save(const std::string &path, const std::string &bytes) {
  std::ofstream f(path, std::ios::binary);
  f << bytes;
}
J rat(const char *n, const char *d) {
  return {{"numerator", n}, {"denominator", d}};
}
int main() try {
  char temp[] = "/private/tmp/sbv-contract-XXXXXX";
  check(::mkdtemp(temp) != nullptr);
  const std::string root = temp;
  struct Cleanup {
    std::string root;
    ~Cleanup() { std::filesystem::remove_all(root); }
  } cleanup{root};
  auto caps = call("capabilities",
                   {{"protocol", "symphony.sbv.capabilities-input.v1"}});
  check(caps.at("cpu").at("available") == true);
  check(caps.at("cuda").at("available") == false);
  check(caps.at("live").at("available") == false);
  auto original = public_fixture::fixture();
  std::vector<unsigned char> bytes(original.begin(), original.begin() + 360);
  auto put = [&](std::size_t offset, std::uint64_t n, unsigned length) {
    for (unsigned i = 0; i < length; ++i)
      bytes[offset + i] = static_cast<unsigned char>(n >> (8 * i));
  };
  for (unsigned i = 0; i < 4; ++i) {
    const auto off = bytes.size();
    bytes.insert(bytes.end(), original.begin() + 360, original.begin() + 416);
    put(off + 8, 1609160400000000000ULL + i * 100, 8);
    put(off + 40, 1609160400000000000ULL + i * 100, 8);
    put(off + 24, (100 + i) * 1000000000ULL, 8);
    bytes[off + 38] = 'T';
  }
  const std::string source(reinterpret_cast<const char *>(bytes.data()),
                           bytes.size());
  save(root + "/sample.dbn", source);
  J p{{"protocol", "symphony.sbv.run-input.v1"},
      {"source_path", root + "/sample.dbn"},
      {"source_sha256", e::sha256_hex(source)},
      {"dataset", "GLBX.MDP3"},
      {"output_path", root + "/serial.json"},
      {"criteria",
       {{"rule", "spaced_trades"},
        {"spacing_ns", "100"},
        {"min_trade_size", "1"},
        {"direction", "any"},
        {"max_signals", "4"}}},
      {"replay",
       {{"before_ns", "100"}, {"after_ns", "100"}, {"retain_events", true}}},
      {"execution",
       {{"model", "touch_observation"},
        {"horizon_ns", "100"},
        {"price_offsets_nanos", J::array({"1000000000", "-1000000000"})},
        {"probability_numerator", "0"},
        {"probability_denominator", "1"}}},
      {"studies", J::array({"signal_summary", "forward_markout"})},
      {"workers", "1"},
      {"extensions",
       {{"user/key~name",
         J::array({"9007199254740993", true, "\x1b[31m", J::object()})}}}};
  auto receipt = call("run", p), serial = read(root + "/serial.json");
  s::validate_result(serial);
  check(serial.at("sections").at("signals").at("data").size() == 4);
  check(serial.at("sections").at("replay").at("data").at("events").size() == 4);
  check(serial.at("sections")
            .at("execution")
            .at("data")[0]
            .at("outcomes")[0]
            .at("observed_trade_touch") == true);
  check(serial.at("sections")
            .at("execution")
            .at("data")[0]
            .at("outcomes")[1]
            .at("observed_trade_touch") == false);
  check(serial.at("sections")
            .at("execution")
            .at("data")[0]
            .at("outcomes")[0]
            .at("fill_probability")
            .at("status") == "unavailable");
  p["workers"] = "4";
  p["output_path"] = root + "/parallel.json";
  call("run", p);
  auto parallel = read(root + "/parallel.json");
  for (auto section : {"signals", "execution", "replay", "studies", "summary"})
    check(serial.at("sections").at(section) ==
          parallel.at("sections").at(section));
  auto model = p;
  model["execution"]["model"] = "user_probability";
  model["execution"]["probability_numerator"] = "3";
  model["execution"]["probability_denominator"] = "4";
  model["output_path"] = root + "/model.json";
  call("run", model);
  auto modeled = read(root + "/model.json");
  check(modeled.at("sections").at("signals") ==
        serial.at("sections").at("signals"));
  check(modeled.at("sections")
            .at("execution")
            .at("data")[0]
            .at("outcomes")[0]
            .at("no_fill_probability")
            .at("numerator") == "1");
  // Mutate only the future of the first signal: its causal observation stays
  // identical while the follow-up touch changes.
  put(360 + 56 + 24, 99000000000ULL, 8);
  std::string future_bytes(reinterpret_cast<const char *>(bytes.data()),
                           bytes.size());
  save(root + "/future.dbn", future_bytes);
  auto future = p;
  future["source_path"] = root + "/future.dbn";
  future["source_sha256"] = e::sha256_hex(future_bytes);
  future["output_path"] = root + "/future.json";
  call("run", future);
  auto changed = read(root + "/future.json");
  check(changed.at("sections").at("signals").at("data")[0] ==
        serial.at("sections").at("signals").at("data")[0]);
  check(changed.at("sections")
            .at("execution")
            .at("data")[0]
            .at("outcomes")[0]
            .at("observed_trade_touch") == false);
  // Equal timestamps still retain ordinal causality: a same-time later trade
  // can be a follow-up observation, never part of the earlier signal prefix.
  put(360 + 56 + 24, 101000000000ULL, 8);
  put(360 + 56 + 40, 1609160400000000000ULL, 8);
  std::string tied_bytes(reinterpret_cast<const char *>(bytes.data()),
                         bytes.size());
  save(root + "/tied.dbn", tied_bytes);
  auto tied = p;
  tied["source_path"] = root + "/tied.dbn";
  tied["source_sha256"] = e::sha256_hex(tied_bytes);
  tied["output_path"] = root + "/tied.json";
  tied["replay"]["before_ns"] = "0";
  tied["replay"]["after_ns"] = "0";
  tied["execution"]["horizon_ns"] = "0";
  call("run", tied);
  auto same_time = read(root + "/tied.json");
  check(same_time.at("sections")
            .at("signals")
            .at("data")[0]
            .at("causal_end_ordinal_exclusive") == "1");
  check(same_time.at("sections")
            .at("replay")
            .at("data")
            .at("windows")[0]
            .at("end_ordinal_exclusive") == "2");
  check(same_time.at("sections")
            .at("execution")
            .at("data")[0]
            .at("outcomes")[0]
            .at("touch_source_ordinal") == "1");
  auto sparse = p;
  sparse["studies"] = J::array();
  sparse["replay"]["retain_events"] = false;
  sparse["output_path"] = root + "/sparse.json";
  call("run", sparse);
  auto no = read(root + "/sparse.json");
  check(no.at("sections").at("studies").at("status") == "not_selected");
  check(no.at("sections").at("replay").at("data").at("events").empty());
  auto zero = p;
  zero["replay"]["before_ns"] = "0";
  zero["replay"]["after_ns"] = "0";
  zero["output_path"] = root + "/zero.json";
  call("run", zero);
  auto z = read(root + "/zero.json");
  check(z.at("sections")
            .at("replay")
            .at("data")
            .at("windows")[0]
            .at("end_ordinal_exclusive") == "1");
  for (const auto field : {"source_sha256", "dataset"}) {
    auto bad = p;
    bad[field] = "wrong";
    bad["output_path"] = root + "/bad.json";
    rejects([&] { call("run", bad); });
    check(!std::filesystem::exists(root + "/bad.json"));
  }
  auto bad = p;
  bad["workers"] = "0";
  rejects([&] { call("run", bad); });
  bad = p;
  bad["studies"] = J::array({"not-installed"});
  rejects([&] { call("run", bad); });
  bad = p;
  bad["unknown"] = true;
  rejects([&] { call("run", bad); });
  bad = p;
  bad["replay"]["after_ns"] = "18446744073709551615";
  rejects([&] { call("run", bad); });
  bad = p;
  bad["execution"]["price_offsets_nanos"] = J::array({"9223372036854775807"});
  rejects([&] { call("run", bad); });
  rejects([&] { call("run", p); });
  check(read(root + "/parallel.json") == parallel);
  rejects([&] { s::dispatch("run", p, e::unix_time_ms() - 1); });
  ::symlink((root + "/sample.dbn").c_str(), (root + "/link.dbn").c_str());
  bad = p;
  bad["source_path"] = root + "/link.dbn";
  rejects([&] { call("run", bad); });
  ::symlink(root.c_str(), (root + "/parent-link").c_str());
  bad = p;
  bad["output_path"] = root + "/parent-link/unsafe.json";
  rejects([&] { call("run", bad); });
  check(!std::filesystem::exists(root + "/unsafe.json"));
  J query{{"protocol", "symphony.sbv.result-query-input.v1"},
          {"path", root + "/serial.json"},
          {"expected_sha256", receipt.at("content_sha256")},
          {"pointer", "/sections/signals/data"},
          {"limit", "2"},
          {"cursor", ""}};
  auto page = call("result_query", query);
  check(page.at("nodes").size() == 2 && page.at("complete") == false);
  check(page.at("nodes")[0].at("value") == J::object());
  query["cursor"] = page.at("next_cursor");
  auto tail = call("result_query", query);
  check(tail.at("complete") == true && tail.at("offset") == "2");
  auto stale = query;
  stale["limit"] = "1";
  rejects([&] { call("result_query", stale); });
  stale = query;
  stale["expected_sha256"] = std::string(64, '0');
  rejects([&] { call("result_query", stale); });
  query["cursor"] = "";
  query["pointer"] = "/sections/user_extensions/data/user~1key~0name/0";
  auto exact = call("result_query", query);
  check(exact.at("nodes")[0].at("value") == "9007199254740993");
  query["pointer"] = "/absent";
  rejects([&] { call("result_query", query); });
  auto tamper = serial;
  tamper["sections"]["summary"]["data"]["signal_count"] = "99";
  rejects([&] { s::validate_result(tamper); });
  tamper = serial;
  tamper["sections"]["user_extensions"]["data"]["float"] = 0.25;
  tamper = s::seal_result(tamper);
  rejects([&] { s::validate_result(tamper); });
  auto large = serial;
  large["sections"]["user_extensions"]["data"] = J::array();
  for (unsigned i = 0; i < 10; ++i)
    large["sections"]["user_extensions"]["data"].push_back(
        std::string(65536, static_cast<char>('a' + i)));
  large = s::seal_result(large);
  save(root + "/large.json", large.dump());
  query["path"] = root + "/large.json";
  query["expected_sha256"] = large.at("content_sha256");
  query["pointer"] = "/sections/user_extensions/data";
  query["limit"] = "10";
  query["cursor"] = "";
  auto bounded = call("result_query", query);
  check(bounded.at("nodes").size() == 7 && bounded.at("complete") == false);
  query["cursor"] = bounded.at("next_cursor");
  auto remainder = call("result_query", query);
  check(remainder.at("nodes").size() == 3 && remainder.at("offset") == "7" &&
        remainder.at("complete") == true);
  for (unsigned i = 0; i < 7; ++i)
    check(bounded.at("nodes")[i].at("value") ==
          large.at("sections").at("user_extensions").at("data")[i]);
  for (unsigned i = 0; i < 3; ++i)
    check(remainder.at("nodes")[i].at("value") ==
          large.at("sections").at("user_extensions").at("data")[i + 7]);
  J compose{{"protocol", "symphony.sbv.compose-input.v1"},
            {"returns", J::array({rat("-1", "100"), rat("1", "100")})},
            {"weights", J::array({rat("1", "2"), rat("1", "2")})},
            {"steps", "2"},
            {"dependence", "independent"},
            {"output_path", root + "/compose.json"},
            {"extensions", J::object()}};
  call("compose", compose);
  auto distribution = read(root + "/compose.json")
                          .at("sections")
                          .at("distributions")
                          .at("data");
  check(distribution.at("paths").size() == 4);
  check(distribution.at("terminal_atoms").size() == 3);
  bool low = false, mixed = false, high = false;
  for (const auto &a : distribution.at("terminal_atoms")) {
    if (a.at("wealth") == rat("9801", "10000")) {
      low = true;
      check(a.at("mass") == rat("1", "4"));
    }
    if (a.at("wealth") == rat("9999", "10000")) {
      mixed = true;
      check(a.at("mass") == rat("1", "2"));
    }
    if (a.at("wealth") == rat("10201", "10000")) {
      high = true;
      check(a.at("mass") == rat("1", "4"));
    }
  }
  check(low && mixed && high);
  compose["dependence"] = "same_draw";
  compose["output_path"] = root + "/dependent.json";
  call("compose", compose);
  auto dep = read(root + "/dependent.json");
  check(dep.at("sections").at("distributions").at("data").at("paths").size() ==
        2);
  compose["weights"][0] = rat("1", "3");
  rejects([&] { call("compose", compose); });
  compose["weights"][0] = rat("1", "2");
  compose["steps"] = "16";
  compose["output_path"] = root + "/overflow.json";
  rejects([&] { call("compose", compose); });
  check(!std::filesystem::exists(root + "/overflow.json"));
  std::cout << checks << " SBV focused contract assertions passed\n";
  return 0;
} catch (const std::exception &x) {
  std::cerr << x.what() << '\n';
  return 1;
}
