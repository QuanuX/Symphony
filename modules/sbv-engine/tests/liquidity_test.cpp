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
    throw std::runtime_error("liquidity assertion line " +
                             std::to_string(where.line()));
}
J read(const std::string &p) {
  std::ifstream f(p);
  J j;
  f >> j;
  return j;
}
void write(const std::string &p, const J &j) {
  std::ofstream f(p);
  f << j.dump();
  check(f.good());
}
J rat(const char *n, const char *d = "1") {
  return {{"numerator", n}, {"denominator", d}};
}
J level(const char *p, const char *q) {
  return {{"price_nanos", p}, {"size", q}, {"order_count", "1"}};
}
int main() try {
  char tmp[] = "/private/tmp/sbv-liquidity-XXXXXX";
  check(::mkdtemp(tmp) != nullptr);
  const std::string root = tmp;
  struct Cleanup {
    std::string root;
    ~Cleanup() { std::filesystem::remove_all(root); }
  } cleanup{root};
  unsigned serial = 0;
  auto source =
      d::base("mechanical reconstructed-book fixture; no market evidence");
  source["sections"]["signals"] =
      d::section(J::array({{{"signal_id", "s0"},
                            {"source_ordinal", "0"},
                            {"available_ns", "100"},
                            {"anchor_price_nanos", "100"},
                            {"causal_end_ordinal_exclusive", "1"}},
                           {{"signal_id", "s1"},
                            {"source_ordinal", "1"},
                            {"available_ns", "100"},
                            {"anchor_price_nanos", "100"},
                            {"causal_end_ordinal_exclusive", "2"}}}));
  source["sections"]["summary"] =
      d::section({{"closed_census", true},
                  {"parent_census_sha256", std::string(64, 'a')}});
  source["sections"]["provenance"] =
      d::section({{"profile", "databento_mbo_orders_strict_v1"},
                  {"scope", "mechanical unit fixture"}});
  J frame{{"protocol", "symphony.sbv.book-frame.v1"},
          {"source_ordinal", "1"},
          {"available_ns", "100"},
          {"signal_ids", J::array({"s0", "s1"})},
          {"status", "available"},
          {"reason", ""},
          {"event_boundary", true},
          {"receive_time_flagged", false},
          {"initialization", "mechanical fixture"},
          {"layer", "reconstructed"},
          {"data",
           {{"bids", J::array({level("99", "5"), level("98", "7")})},
            {"asks", J::array({level("101", "4"), level("102", "6")})},
            {"bid_level_count", "2"},
            {"ask_level_count", "2"},
            {"order_count", "4"},
            {"market_state", "uncrossed"}}}};
  source["sections"]["book_frames"] = d::section(J::array({frame}));
  source["sections"]["book_checkpoint"] =
      d::section(nullptr, "not_selected", "fixture has no checkpoint");
  source["sections"]["replay"] =
      d::section({{"events", J::array()},
                  {"windows", J::array()},
                  {"test", "retained replay unchanged"}});
  auto save = [&](J j) {
    j = s::seal_result(j);
    const auto path = root + "/source-" + std::to_string(serial++) + ".json";
    write(path, j);
    return J{{"path", path}, {"expected_sha256", j["content_sha256"]}};
  };
  auto ref = save(source);
  J order{{"order_id", "o0"},
          {"signal_id", "s0"},
          {"frame_source_ordinal", "1"},
          {"side", "buy"},
          {"quantity", "8"},
          {"limit_price_nanos", nullptr},
          {"time_in_force", "IOC"},
          {"max_level_participation", rat("1")},
          {"activation_probability",
           {{"status", "supplied"}, {"value", rat("3", "5")}}}};
  J p{{"protocol", "symphony.sbv.liquidity-input.v1"},
      {"path", ref["path"]},
      {"expected_sha256", ref["expected_sha256"]},
      {"output_path", root + "/first.json"},
      {"orders", J::array({order})},
      {"liquidity_mode", "independent"},
      {"depth_policy", "require_sufficient"},
      {"market_state_policy", "uncrossed_only"},
      {"receive_time_policy", "unflagged_only"},
      {"on_unavailable", "unavailable"},
      {"studies", J::array({"liquidity_summary", "fill_quality"})},
      {"workers", "4"},
      {"extensions", J::object()}};
  auto call = [&](J q) {
    q["output_path"] = root + "/output-" + std::to_string(serial++) + ".json";
    s::dispatch("liquidity", q, e::unix_time_ms() + 30000);
    return read(q["output_path"].get<std::string>());
  };
  auto refused = [&](J q) {
    q["output_path"] = root + "/bad-" + std::to_string(serial++) + ".json";
    bool failed = false;
    try {
      s::dispatch("liquidity", q, e::unix_time_ms() + 30000);
    } catch (const std::exception &) {
      failed = true;
    }
    check(failed);
    check(!std::filesystem::exists(q["output_path"].get<std::string>()));
  };
  auto bind = [&](J q, J j) {
    auto r = save(j);
    q["path"] = r["path"];
    q["expected_sha256"] = r["expected_sha256"];
    return q;
  };
  auto row = [](const J &j) -> J {
    return j.at("sections").at("execution").at("data").at(0);
  };
  auto a = call(p), a0 = row(a);
  check(a0["data"]["filled_quantity"] == "8");
  check(a0["data"]["fills"].size() == 2);
  check(a0["data"]["notional_price_nanos_times_quantity"] == "812");
  check(a0["data"]["vwap_price_nanos"]["value"] == rat("203", "2"));
  check(a0["any_fill_probability"]["value"] == rat("3", "5"));
  check(a0["complete_fill_probability"]["value"] == rat("3", "5"));
  check(a0["no_fill_probability"]["value"] == rat("2", "5"));
  check(a0["quantity_distribution"]["value"][1]["filled_quantity"] == "0");
  check(a["sections"]["studies"]["data"][1]["results"][0]["data"]
         ["adverse_slippage_price_nanos"]["value"] == rat("3", "2"));
  check(a["sections"]["replay"] == source["sections"]["replay"]);
  check(a["sections"]["book_frames"] == source["sections"]["book_frames"]);
  auto q = p;
  q["orders"][0]["side"] = "sell";
  auto sell = row(call(q));
  check(sell["data"]["notional_price_nanos_times_quantity"] == "789");
  check(sell["data"]["vwap_price_nanos"]["value"] == rat("789", "8"));
  check(call(q)["sections"]["studies"]["data"][1]["results"][0]["data"]
               ["adverse_slippage_price_nanos"]["value"] == rat("11", "8"));
  q = p;
  q["orders"][0]["quantity"] = "12";
  auto partial = row(call(q));
  check(partial["data"]["filled_quantity"] == "10");
  check(partial["data"]["disposition"] == "partial_fill");
  check(partial["complete_fill_probability"]["value"] == rat("0"));
  q["orders"][0]["time_in_force"] = "FOK";
  auto fok = row(call(q));
  check(fok["data"]["filled_quantity"] == "0");
  check(fok["data"]["fills"].empty());
  check(fok["any_fill_probability"]["value"] == rat("0"));
  check(fok["no_fill_probability"]["value"] == rat("1"));
  check(fok["quantity_distribution"]["value"].size() == 1);
  q = p;
  q["orders"][0]["limit_price_nanos"] = "101";
  check(row(call(q))["data"]["filled_quantity"] == "4");
  q["orders"][0]["limit_price_nanos"] = "100";
  check(row(call(q))["data"]["filled_quantity"] == "0");
  q = p;
  q["orders"][0]["side"] = "sell";
  q["orders"][0]["limit_price_nanos"] = "99";
  check(row(call(q))["data"]["filled_quantity"] == "5");
  q = p;
  q["orders"][0]["max_level_participation"] = rat("1", "2");
  auto haircut = row(call(q));
  check(haircut["data"]["filled_quantity"] == "5");
  check(haircut["data"]["vwap_price_nanos"]["value"] == rat("508", "5"));
  q = p;
  q["orders"][0]["activation_probability"] = {
      {"status", "unavailable"}, {"reason", "no probability chosen"}};
  check(row(call(q))["any_fill_probability"]["status"] == "unavailable");
  check(row(call(q))["data"]["filled_quantity"] == "8");
  for (const char *prob : {"0", "1"}) {
    q = p;
    q["orders"][0]["activation_probability"]["value"] = rat(prob);
    auto value = row(call(q));
    check(value["any_fill_probability"]["value"] == rat(prob));
    check(value["data"]["filled_quantity"] == "8");
  }
  auto thin = source;
  thin["sections"]["book_frames"]["data"][0]["data"]["ask_level_count"] = "3";
  q = bind(p, thin);
  q["orders"][0]["quantity"] = "11";
  check(row(call(q))["status"] == "unavailable");
  q["depth_policy"] = "visible_only";
  check(row(call(q))["data"]["filled_quantity"] == "10");
  check(row(call(q))["data"]["coverage"] == "visible_depth_only");
  q["depth_policy"] = "require_sufficient";
  q["orders"][0]["limit_price_nanos"] = "102";
  check(row(call(q))["data"]["coverage"] == "sufficient_for_selected_order");
  q["orders"][0]["limit_price_nanos"] = nullptr;
  q["orders"][0]["max_level_participation"] = rat("0");
  check(row(call(q))["data"]["filled_quantity"] == "0");
  auto unavailable = source;
  unavailable["sections"]["book_frames"]["data"][0]["status"] = "unavailable";
  unavailable["sections"]["book_frames"]["data"][0]["reason"] =
      "initial state absent";
  unavailable["sections"]["book_frames"]["data"][0]["data"] = nullptr;
  q = bind(p, unavailable);
  check(row(call(q))["status"] == "unavailable");
  check(row(call(q))["any_fill_probability"]["status"] == "unavailable");
  q["on_unavailable"] = "reject";
  refused(q);
  auto locked = source;
  locked["sections"]["book_frames"]["data"][0]["data"]["bids"][0]
        ["price_nanos"] = "101";
  locked["sections"]["book_frames"]["data"][0]["data"]["market_state"] =
      "locked";
  q = bind(p, locked);
  check(row(call(q))["status"] == "unavailable");
  q["market_state_policy"] = "allow_locked_crossed";
  check(row(call(q))["status"] == "available");
  auto flagged = source;
  flagged["sections"]["book_frames"]["data"][0]["receive_time_flagged"] = true;
  q = bind(p, flagged);
  check(row(call(q))["status"] == "unavailable");
  q["receive_time_policy"] = "allow_flagged";
  check(row(call(q))["data"]["receive_time_flagged"] == true);
  q = p;
  q["orders"][0]["frame_source_ordinal"] = "2";
  check(row(call(q))["status"] == "unavailable");
  q["orders"][0]["signal_id"] = "s1";
  q["orders"][0]["frame_source_ordinal"] = "0";
  refused(q);
  // Shared snapshot consumes physical depth in caller order; FOK failure rolls
  // back.
  auto shared = p;
  shared["liquidity_mode"] = "shared_snapshot";
  shared["orders"][0]["activation_probability"]["value"] = rat("1");
  auto other = shared["orders"][0];
  other["order_id"] = "o1";
  shared["orders"].push_back(other);
  auto sh = call(shared);
  check(sh["sections"]["execution"]["data"][0]["data"]["filled_quantity"] ==
        "8");
  check(sh["sections"]["execution"]["data"][1]["data"]["filled_quantity"] ==
        "2");
  check(sh["sections"]["resources"]["data"]["actual_workers"] == "1");
  shared["orders"][0]["quantity"] = "11";
  shared["orders"][0]["time_in_force"] = "FOK";
  sh = call(shared);
  check(sh["sections"]["execution"]["data"][0]["data"]["disposition"] ==
        "fok_canceled");
  check(sh["sections"]["execution"]["data"][1]["data"]["filled_quantity"] ==
        "8");
  auto blocked = bind(shared, thin);
  blocked["orders"][0]["time_in_force"] = "IOC";
  auto br = call(blocked);
  check(br["sections"]["execution"]["data"][1]["reason"] ==
        "earlier shared order unresolved; remaining liquidity unknown");
  shared["orders"][0]["activation_probability"]["value"] = rat("1", "2");
  refused(shared);
  shared["orders"][0]["activation_probability"]["value"] = rat("1");
  shared["orders"][1]["frame_source_ordinal"] = "2";
  refused(shared);
  auto participation_batch = p;
  participation_batch["liquidity_mode"] = "shared_snapshot";
  participation_batch["orders"][0]["activation_probability"] = {
      {"status", "unavailable"}, {"reason", "conditional batch only"}};
  participation_batch["orders"][0]["max_level_participation"] = rat("1", "2");
  auto cap_order = participation_batch["orders"][0];
  cap_order["order_id"] = "cap-second";
  participation_batch["orders"].push_back(cap_order);
  auto cap_result = call(participation_batch);
  check(cap_result["sections"]["execution"]["data"][0]["data"]
                  ["filled_quantity"] == "5");
  check(cap_result["sections"]["execution"]["data"][1]["data"]
                  ["filled_quantity"] == "5");
  check(cap_result["sections"]["execution"]["data"][1]["any_fill_probability"]
                  ["status"] == "unavailable");
  auto odd = source;
  odd["sections"]["book_frames"]["data"][0]["data"]["asks"][0]["size"] = "5";
  auto rounded = bind(p, odd);
  rounded["orders"][0]["max_level_participation"] = rat("1", "2");
  check(row(call(rounded))["data"]["fills"][0]["participation_cap"] == "2");
  // Serial/threaded independent outcomes and studies must be identical.
  q = p;
  q["orders"] = J::array();
  for (unsigned i = 0; i < 32; ++i) {
    auto o = order;
    o["order_id"] = "parallel-" + std::to_string(i);
    o["quantity"] = std::to_string(i + 1);
    q["orders"].push_back(o);
  }
  q["workers"] = "1";
  auto one = call(q);
  q["workers"] = "8";
  auto many = call(q);
  check(one["sections"]["execution"] == many["sections"]["execution"]);
  check(one["sections"]["studies"] == many["sections"]["studies"]);
  check(many["sections"]["resources"]["data"]["actual_workers"] == "8");
  // Wide exact values above int64 notional and 2^53 price; negative prices
  // valid.
  auto large = source;
  auto &ld = large["sections"]["book_frames"]["data"][0]["data"];
  ld["bids"] = J::array();
  ld["bid_level_count"] = "0";
  ld["asks"] = J::array({level("9007199254740993", "18446744073709551615")});
  ld["ask_level_count"] = "1";
  ld["market_state"] = "one_or_both_sides_empty";
  q = bind(p, large);
  q["orders"][0]["quantity"] = "1000000000";
  auto lr = row(call(q));
  check(lr["data"]["notional_price_nanos_times_quantity"] ==
        "9007199254740993000000000");
  check(lr["data"]["vwap_price_nanos"]["value"] == rat("9007199254740993"));
  q["orders"][0]["max_level_participation"] = rat("999999999", "1000000000");
  check(row(call(q))["data"]["filled_quantity"] == "1000000000");
  check(row(call(q))["data"]["fills"][0]["participation_cap"] ==
        "18446744055262807541");
  ld["asks"][0]["price_nanos"] = "-100";
  q = bind(p, large);
  check(row(call(q))["data"]["notional_price_nanos_times_quantity"] == "-800");
  for (unsigned mode = 0; mode < 8; ++mode) {
    auto corrupt = source;
    auto &cf = corrupt["sections"]["book_frames"]["data"][0];
    if (mode == 0)
      cf["event_boundary"] = false;
    if (mode == 1)
      cf["data"]["asks"][1]["price_nanos"] = "100";
    if (mode == 2)
      cf["data"]["asks"][0]["size"] = "0";
    if (mode == 3)
      cf["data"]["ask_level_count"] = "1";
    if (mode == 4)
      cf["available_ns"] = "99";
    if (mode == 5)
      cf["data"]["market_state"] = "crossed";
    if (mode == 6)
      cf["data"]["asks"][0]["price_nanos"] = "9223372036854775807";
    if (mode == 7)
      corrupt["sections"]["book_frames"]["data"].push_back(cf);
    refused(bind(p, corrupt));
  }
  for (unsigned mode = 0; mode < 8; ++mode) {
    auto bad = p;
    if (mode == 0)
      bad["orders"][0]["quantity"] = "0";
    if (mode == 1)
      bad["orders"][0]["max_level_participation"] = rat("2");
    if (mode == 2)
      bad["orders"][0]["activation_probability"]["value"] = rat("-1");
    if (mode == 3)
      bad["orders"].push_back(bad["orders"][0]);
    if (mode == 4)
      bad["workers"] = "0";
    if (mode == 5)
      bad["expected_sha256"] = std::string(64, 'b');
    if (mode == 6)
      bad["orders"][0]["limit_price_nanos"] = "9223372036854775807";
    if (mode == 7)
      bad["studies"].push_back("liquidity_summary");
    refused(bad);
  }
  q = p;
  q["orders"] = J::array();
  q["studies"] = J::array();
  auto empty = call(q);
  check(empty["sections"]["execution"]["data"].empty());
  check(empty["sections"]["studies"]["status"] == "not_selected");
  check(empty["sections"]["resources"]["data"]["actual_workers"] == "0");
  if (const char *emit = std::getenv("SYMPHONY_SBV_EMIT_LIQUIDITY_FIXTURE")) {
    const std::filesystem::path dir = emit;
    check(dir.is_absolute() && std::filesystem::is_directory(dir));
    auto fixture = s::seal_result(source);
    write((dir / "liquidity-source.json").string(), fixture);
    auto request = p;
    request["path"] = (dir / "liquidity-source.json").string();
    request["expected_sha256"] = fixture["content_sha256"];
    request["output_path"] = (dir / "liquidity.json").string();
    write((dir / "liquidity-request.json").string(), request);
  }
  auto output = p;
  output["output_path"] = root + "/retained.json";
  auto receipt = s::dispatch("liquidity", output, e::unix_time_ms() + 30000);
  std::filesystem::remove(ref["path"].get<std::string>());
  check(s::dispatch("result_inspect",
                    {{"protocol", "symphony.sbv.result-inspect-input.v1"},
                     {"path", output["output_path"]},
                     {"expected_sha256", receipt["content_sha256"]}},
                    e::unix_time_ms() + 30000)["content_sha256"] ==
        receipt["content_sha256"]);
  std::cout << checks << " liquidity assertions passed\n";
  return 0;
} catch (const std::exception &x) {
  std::cerr << x.what() << '\n';
  return 1;
}
