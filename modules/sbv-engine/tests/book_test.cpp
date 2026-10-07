#include "../../sqav-databento-dbn-cpp/tests/public_fixture.hpp"
#include "../src/detail.hpp"
#include <cstdlib>
#include <filesystem>
#include <fstream>
#include <iostream>
#include <source_location>
#include <unistd.h>
namespace s = symphony::sbv;
namespace d = s::detail;
namespace e = symphony::knowledge::engine;
using J = s::Json;
unsigned checks = 0;
void check(bool ok,
           std::source_location where = std::source_location::current()) {
  ++checks;
  if (!ok)
    throw std::runtime_error("book check line " + std::to_string(where.line()));
}
J read(const std::string &p) {
  std::ifstream f(p);
  J j;
  f >> j;
  return j;
}
void write(const std::string &p, const std::string &v) {
  std::ofstream f(p, std::ios::binary);
  f << v;
  check(f.good());
}
struct E {
  char action;
  std::uint64_t id;
  std::int64_t price;
  std::uint32_t size;
  char side;
  unsigned flags = 128;
};
int main() try {
  char tmp[] = "/private/tmp/sbv-book-XXXXXX";
  check(::mkdtemp(tmp) != nullptr);
  const std::string root = tmp;
  struct Cleanup {
    std::string p;
    ~Cleanup() { std::filesystem::remove_all(p); }
  } cleanup{root};
  const auto fixture = public_fixture::fixture();
  const std::uint64_t t = 1609160400000000000ULL;
  unsigned serial = 0;
  auto request = [&](const std::vector<E> &events) -> J {
    const auto suffix = std::to_string(serial++);
    std::vector<unsigned char> bytes(fixture.begin(), fixture.begin() + 360);
    J signals = J::array(), ids = J::array();
    auto put = [&](std::size_t o, std::uint64_t n, unsigned size) {
      for (unsigned i = 0; i < size; ++i)
        bytes[o + i] = static_cast<unsigned char>(n >> (i * 8));
    };
    for (std::size_t i = 0; i < events.size(); ++i) {
      auto o = bytes.size();
      bytes.insert(bytes.end(), fixture.begin() + 360, fixture.begin() + 416);
      const auto &x = events[i];
      put(o + 8, t + i * 100, 8);
      put(o + 40, t + i * 100, 8);
      put(o + 16, x.id, 8);
      put(o + 24, x.price, 8);
      put(o + 32, x.size, 4);
      bytes[o + 36] = x.flags;
      bytes[o + 37] = 0;
      bytes[o + 38] = x.action;
      bytes[o + 39] = x.side;
      put(o + 52, i * 7, 4);
      const auto id = "s" + std::to_string(i);
      ids.push_back(id);
      signals.push_back(
          {{"signal_id", id},
           {"source_ordinal", std::to_string(i)},
           {"available_ns", std::to_string(t + i * 100)},
           {"anchor_price_nanos", "100"},
           {"causal_end_ordinal_exclusive", std::to_string(i + 1)},
           {"context_reference", "synthetic book fixture"}});
    }
    const std::string source(reinterpret_cast<const char *>(bytes.data()),
                             bytes.size());
    write(root + "/" + suffix + ".dbn", source);
    auto sha = e::sha256_hex(source);
    auto parent =
        d::base("synthetic mechanical test census, not market evidence");
    parent["sections"]["signals"] = d::section(signals);
    J producer{{"id", "book-fixture"},
               {"version", "1"},
               {"artifact_sha256", ""},
               {"reproducibility", "uncaptured"}};
    J declared{{"protocol", "symphony.sbv.external-census.v1"},
               {"source_sha256", sha},
               {"mode", "causal_declared"},
               {"producer", producer},
               {"signals", signals}};
    parent["sections"]["choices"] =
        d::section({{"protocol", "symphony.sbv.evaluate-input.v1"},
                    {"source_sha256", sha},
                    {"dataset", "GLBX.MDP3"},
                    {"census", declared}});
    parent["sections"]["summary"] =
        d::section({{"closed_census", true},
                    {"signal_count", std::to_string(signals.size())},
                    {"causality", "causal_declared"},
                    {"census_sha256", e::sha256_hex(declared.dump())}});
    parent["sections"]["provenance"] =
        d::section({{"source_sha256", sha},
                    {"dataset", "GLBX.MDP3"},
                    {"instrument_id", "5482"},
                    {"census_producer", producer}});
    parent = s::seal_result(parent);
    write(root + "/" + suffix + "-census.json", parent.dump());
    return {{"protocol", "symphony.sbv.book-input.v1"},
            {"source_path", root + "/" + suffix + ".dbn"},
            {"source_sha256", sha},
            {"dataset", "GLBX.MDP3"},
            {"census_result",
             {{"path", root + "/" + suffix + "-census.json"},
              {"expected_sha256", parent.at("content_sha256")}}},
            {"signal_ids", ids},
            {"output_path", root + "/" + suffix + "-book.json"},
            {"initial_state", {{"mode", "uninitialized"}}},
            {"on_anomaly", "invalidate_until_reset"},
            {"replay",
             {{"before_ns", "0"}, {"after_ns", "0"}, {"retain_events", true}}},
            {"frames",
             {{"cadence", "signals"},
              {"levels_per_side", "1"},
              {"maximum", "4096"}}},
            {"emit_checkpoint", true},
            {"extensions", J::object()}};
  };
  auto call = [&](J p) {
    s::dispatch("book", p, e::unix_time_ms() + 30000);
    return read(p.at("output_path").get<std::string>());
  };
  auto fresh = [&](J p) {
    p["output_path"] = root + "/output-" + std::to_string(serial++) + ".json";
    return p;
  };
  auto refused = [&](J p) {
    p = fresh(p);
    bool failed = false;
    try {
      call(p);
    } catch (const std::exception &) {
      failed = true;
    }
    check(failed);
    check(!std::filesystem::exists(p.at("output_path").get<std::string>()));
  };
  const std::vector<E> sequence{
      {'R', 0, 0, 0, 'N', 40},    {'A', 1, 99, 10, 'B', 40},
      {'A', 2, 101, 7, 'A', 168}, {'A', 3, 98, 4, 'B'},
      {'F', 1, 99, 3, 'B'},       {'T', 1, 99, 3, 'B'},
      {'C', 1, 99, 3, 'B'},       {'M', 2, 102, 8, 'A'},
      {'C', 1, 99, 7, 'B'},       {'N', 0, 0, 0, 'N'}};
  auto p = request(sequence);
  auto result = call(p);
  const auto &f = result["sections"]["book_frames"]["data"];
  check(f.size() == 10);
  check(f[0]["status"] == "unavailable" && f[1]["status"] == "unavailable");
  check(f[2]["status"] == "available");
  check(f[2]["receive_time_flagged"] == true);
  check(f[3]["data"]["bids"].size() == 1 &&
        f[3]["data"]["bid_level_count"] == "2");
  check(f[4]["data"] == f[3]["data"] && f[5]["data"] == f[3]["data"]);
  check(f[6]["data"]["bids"][0]["size"] == "7");
  check(f[7]["data"]["asks"][0]["price_nanos"] == "102");
  check(f[8]["data"]["bids"][0]["price_nanos"] == "98");
  check(result["sections"]["book_checkpoint"]["data"]["orders"].size() == 2);
  check(result["sections"]["diagnostics"]["data"]
              ["inaccurate_receive_timestamp_count"] == "3");
  check(result["sections"]["replay"]["data"]["events"].size() == 10);
  auto prefix = fresh(p);
  prefix["signal_ids"] = J::array({"s3"});
  auto initial = call(prefix);
  auto resumed = fresh(p);
  resumed["signal_ids"] = J::array({"s4", "s5", "s6", "s7", "s8", "s9"});
  resumed["initial_state"] = {
      {"mode", "checkpoint"},
      {"result",
       {{"path", prefix["output_path"]},
        {"expected_sha256", initial["content_sha256"]}}}};
  auto continuation = call(resumed);
  const auto &rf = continuation["sections"]["book_frames"]["data"];
  for (std::size_t i = 0; i < rf.size(); ++i)
    check(rf[i]["data"] == f[i + 4]["data"]);
  check(continuation["sections"]["book_checkpoint"]["data"]["orders"] ==
        result["sections"]["book_checkpoint"]["data"]["orders"]);
  auto bad = resumed;
  bad["signal_ids"] = J::array({"s2"});
  refused(bad);
  bad = resumed;
  bad["replay"]["before_ns"] = "100";
  refused(bad);
  bad = resumed;
  bad["initial_state"]["result"]["expected_sha256"] = std::string(64, 'a');
  refused(bad);
  bad = p;
  bad["source_sha256"] = std::string(64, 'b');
  refused(bad);
  bad = p;
  bad["signal_ids"] = J::array({"absent"});
  refused(bad);
  bad = p;
  bad["signal_ids"] = J::array({"s2", "s2"});
  refused(bad);
  bad = p;
  bad["frames"]["maximum"] = "1";
  refused(bad);
  bad = p;
  bad["frames"]["levels_per_side"] = "0";
  refused(bad);
  bad = p;
  bad["on_anomaly"] = "ignore";
  refused(bad);
  auto none = fresh(p);
  none["signal_ids"] = J::array();
  auto empty = call(none);
  check(empty["sections"]["book_frames"]["data"].empty());
  check(empty["sections"]["book_checkpoint"]["status"] == "unavailable");
  auto no_raw = fresh(p);
  no_raw["replay"]["retain_events"] = false;
  no_raw["emit_checkpoint"] = false;
  auto nr = call(no_raw);
  check(nr["sections"]["replay"]["data"]["events"].empty());
  check(nr["sections"]["book_checkpoint"]["status"] == "not_selected");
  auto window = fresh(p);
  window["signal_ids"] = J::array({"s2"});
  window["frames"]["cadence"] = "signals_and_event_ends";
  window["replay"]["before_ns"] = "100";
  window["replay"]["after_ns"] = "200";
  auto wr = call(window);
  check(wr["sections"]["book_frames"]["data"].size() == 3);
  check(wr["sections"]["replay"]["data"]["events"].size() == 4);
  check(wr["sections"]["replay"]["data"]["hydration_first_ordinal"] == "0");
  auto missing = request(
      {{'A', 1, 99, 10, 'B'}, {'T', 1, 99, 1, 'B'}, {'C', 1, 99, 10, 'B'}});
  auto mr = call(missing);
  for (const auto &row : mr["sections"]["book_frames"]["data"])
    check(row["status"] == "unavailable" && row["data"].is_null());
  for (const auto anomalous : std::vector<E>{{'N', 0, 0, 0, 'N', 132},
                                             {'A', 1, 99, 1, 'B'},
                                             {'C', 99, 99, 1, 'B'},
                                             {'C', 1, 99, 11, 'B'},
                                             {'C', 1, 100, 1, 'B'},
                                             {'A', 2, 100, 0, 'B'},
                                             {'A', 2, INT64_MAX, 1, 'B'},
                                             {'A', 2, 100, 1, 'N'},
                                             {'Z', 0, 0, 0, 'N'},
                                             {'A', 2, 100, 1, 'B', 160},
                                             {'A', 2, 100, 1, 'B', 192},
                                             {'A', 2, 100, 1, 'B', 144},
                                             {'A', 2, 100, 1, 'B', 130}}) {
    auto a = request({{'R', 0, 0, 0, 'N'},
                      {'A', 1, 99, 10, 'B'},
                      anomalous,
                      {'A', 8, 95, 2, 'B'},
                      {'R', 0, 0, 0, 'N'},
                      {'A', 9, 96, 3, 'B'}});
    auto ar = call(a);
    auto &af = ar["sections"]["book_frames"]["data"];
    check(af[2]["status"] == "unavailable" && af[3]["status"] == "unavailable");
    check(af[4]["status"] == "available" &&
          af[4]["data"]["order_count"] == "0");
    check(af[5]["data"]["order_count"] == "1");
    a["on_anomaly"] = "reject";
    refused(a);
  }
  auto snapshot = request({{'R', 0, 0, 0, 'N', 40},
                           {'A', 1, 99, 10, 'B', 40},
                           {'N', 0, 0, 0, 'N', 0},
                           {'N', 0, 0, 0, 'N'}});
  auto sr = call(snapshot);
  check(sr["sections"]["book_frames"]["data"][2]["status"] == "unavailable");
  check(sr["sections"]["book_frames"]["data"][3]["status"] == "available");
  auto incomplete =
      request({{'R', 0, 0, 0, 'N', 40}, {'A', 1, 99, 10, 'B', 40}});
  auto ir = call(incomplete);
  check(ir["sections"]["book_checkpoint"]["status"] == "unavailable");
  auto levels = request({{'R', 0, 0, 0, 'N'},
                         {'A', 1, 100, UINT32_MAX, 'B'},
                         {'A', 2, 100, UINT32_MAX, 'B'},
                         {'A', 3, 100, 1, 'A'},
                         {'M', 3, 99, 1, 'A'}});
  auto lr = call(levels);
  check(lr["sections"]["book_frames"]["data"][2]["data"]["bids"][0]["size"] ==
        "8589934590");
  check(lr["sections"]["book_frames"]["data"][3]["data"]["market_state"] ==
        "locked");
  check(lr["sections"]["book_frames"]["data"][4]["data"]["market_state"] ==
        "crossed");
  // Checkpoint structural tampering with a valid envelope hash still fails.
  for (int mode = 0; mode < 4; ++mode) {
    auto cp = initial;
    auto &c = cp["sections"]["book_checkpoint"]["data"];
    if (mode == 0)
      c["binding"]["channel_id"] = "99";
    if (mode == 1)
      c["orders"][0]["size"] = "0";
    if (mode == 2)
      c["orders"].push_back(c["orders"][0]);
    if (mode == 3)
      c["next_ordinal"] = "2";
    cp = s::seal_result(cp);
    auto path = root + "/bad-cp-" + std::to_string(mode) + ".json";
    write(path, cp.dump());
    auto r = resumed;
    r["initial_state"]["result"] = {{"path", path},
                                    {"expected_sha256", cp["content_sha256"]}};
    refused(r);
  }
  auto empty_resume = fresh(resumed);
  empty_resume["signal_ids"] = J::array();
  check(call(empty_resume)["sections"]["book_checkpoint"]["status"] ==
        "unavailable");
  // Corrupt source scope/time while re-binding its digest and census. Refusal
  // must come from actual decode admission, not a stale checksum.
  for (unsigned mode = 0; mode < 4; ++mode) {
    auto q = request(sequence);
    auto path = q["source_path"].get<std::string>();
    std::ifstream input(path, std::ios::binary);
    std::string bytes((std::istreambuf_iterator<char>(input)), {});
    const auto offset = 360 + 56;
    if (mode == 0)
      bytes[offset + 2] ^= 1;
    if (mode == 1)
      bytes[offset + 4] ^= 1;
    if (mode == 2)
      bytes[offset + 37] = 1;
    if (mode == 3)
      for (unsigned n = 0; n < 8; ++n)
        bytes[offset + 40 + n] = 0;
    write(path, bytes);
    q["source_sha256"] = e::sha256_hex(bytes);
    auto cp = read(q["census_result"]["path"].get<std::string>());
    cp["sections"]["provenance"]["data"]["source_sha256"] = q["source_sha256"];
    cp = s::seal_result(cp);
    write(q["census_result"]["path"].get<std::string>(), cp.dump());
    q["census_result"]["expected_sha256"] = cp["content_sha256"];
    refused(q);
  }
  auto tied = request(sequence);
  auto tied_path = tied["source_path"].get<std::string>();
  std::ifstream tied_input(tied_path, std::ios::binary);
  std::string tied_bytes((std::istreambuf_iterator<char>(tied_input)), {});
  for (std::size_t i = 0; i < sequence.size(); ++i)
    for (unsigned n = 0; n < 8; ++n)
      tied_bytes[360 + i * 56 + 40 + n] = static_cast<char>(t >> (n * 8));
  write(tied_path, tied_bytes);
  tied["source_sha256"] = e::sha256_hex(tied_bytes);
  auto tied_census = read(tied["census_result"]["path"].get<std::string>());
  tied_census["sections"]["provenance"]["data"]["source_sha256"] =
      tied["source_sha256"];
  for (auto &row : tied_census["sections"]["signals"]["data"])
    row["available_ns"] = std::to_string(t);
  tied_census["sections"]["choices"]["data"]["source_sha256"] =
      tied["source_sha256"];
  auto &tied_declaration = tied_census["sections"]["choices"]["data"]["census"];
  tied_declaration["source_sha256"] = tied["source_sha256"];
  tied_declaration["signals"] = tied_census["sections"]["signals"]["data"];
  tied_census["sections"]["summary"]["data"]["census_sha256"] =
      e::sha256_hex(tied_declaration.dump());
  tied_census = s::seal_result(tied_census);
  write(tied["census_result"]["path"].get<std::string>(), tied_census.dump());
  tied["census_result"]["expected_sha256"] = tied_census["content_sha256"];
  auto tied_result = call(tied);
  for (std::size_t i = 0; i < f.size(); ++i)
    check(tied_result["sections"]["book_frames"]["data"][i]["data"] ==
          f[i]["data"]);
  // Artifact remains readable without its source dependencies.
  if (const char *emit = std::getenv("SYMPHONY_SBV_EMIT_BOOK_FIXTURE")) {
    std::filesystem::path dir = emit;
    check(dir.is_absolute() && std::filesystem::is_directory(dir));
    std::filesystem::copy_file(p["source_path"].get<std::string>(),
                               dir / "fixture.dbn");
    std::filesystem::copy_file(p["census_result"]["path"].get<std::string>(),
                               dir / "census.json");
    auto q = p;
    q["source_path"] = (dir / "fixture.dbn").string();
    q["census_result"]["path"] = (dir / "census.json").string();
    q["output_path"] = (dir / "book.json").string();
    write((dir / "book-request.json").string(), q.dump(2));
  }
  std::filesystem::remove(p["source_path"].get<std::string>());
  check(s::dispatch("result_inspect",
                    {{"protocol", "symphony.sbv.result-inspect-input.v1"},
                     {"path", p["output_path"]},
                     {"expected_sha256", result["content_sha256"]}},
                    e::unix_time_ms() + 30000)["content_sha256"] ==
        result["content_sha256"]);
  std::cout << checks << " book assertions passed\n";
  return 0;
} catch (const std::exception &x) {
  std::cerr << x.what() << '\n';
  return 1;
}
