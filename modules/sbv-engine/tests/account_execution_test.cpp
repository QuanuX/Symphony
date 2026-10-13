#include "../src/consumer_source.hpp"
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
void check(bool ok, std::source_location at = std::source_location::current()) {
  ++checks;
  if (!ok)
    throw std::runtime_error("account execution assertion line " +
                             std::to_string(at.line()));
}
J read(const std::string &path) {
  std::ifstream f(path);
  J j;
  f >> j;
  return j;
}
void write(const std::string &path, const J &j) {
  std::ofstream f(path);
  f << j.dump();
  check(f.good());
}
J q(const char *n, const char *den = "1") {
  return {{"numerator", n}, {"denominator", den}};
}
J level(const char *p, const char *n) {
  return {{"price_nanos", p}, {"size", n}, {"order_count", "1"}};
}
int main(int argc, char **argv) try {
  check(argc == 2);
  char temp[] = "/private/tmp/sbv-account-execution-XXXXXX";
  check(mkdtemp(temp) != nullptr);
  std::string root = temp;
  struct Cleanup {
    std::string path;
    ~Cleanup() { std::filesystem::remove_all(path); }
  } cleanup{root};
  if (const char *retain = getenv("SBV_ACCOUNT_EXECUTION_FIXTURE_ROOT")) {
    root = retain;
    std::filesystem::create_directories(root);
    root = std::filesystem::canonical(root).string();
  }
  auto templates = read(argv[1]);
  auto input = templates["variants"]["account_simulate"]["shared_liquidity"];
  auto frame =
      J{{"protocol", "symphony.sbv.book-frame.v1"},
        {"source_ordinal", "1"},
        {"available_ns", "100"},
        {"signal_ids", J::array({"s0"})},
        {"status", "available"},
        {"reason", ""},
        {"event_boundary", true},
        {"receive_time_flagged", false},
        {"initialization", "mechanical initialized depth"},
        {"layer", "reconstructed"},
        {"data",
         {{"bids", J::array({level("9000000000", "5")})},
          {"asks",
           J::array({level("10000000000", "4"), level("11000000000", "6")})},
          {"bid_level_count", "1"},
          {"ask_level_count", "2"},
          {"order_count", "3"},
          {"market_state", "uncrossed"}}}};
  auto source = d::base("mechanical reconstructed depth; not market evidence");
  source["sections"]["book_frames"] = d::section(J::array({frame}));
  source["sections"]["signals"] =
      d::section(J::array({{{"signal_id", "s0"},
                            {"source_ordinal", "0"},
                            {"available_ns", "100"},
                            {"anchor_price_nanos", "10000000000"}}}));
  source["sections"]["summary"] =
      d::section({{"closed_census", true},
                  {"parent_census_sha256", std::string(64, 'a')}});
  source["sections"]["provenance"] =
      d::section({{"profile", "databento_mbo_orders_strict_v1"},
                  {"scope", "mechanical fixture"}});
  source["sections"]["book_checkpoint"] =
      d::section(nullptr, "not_selected", "no checkpoint");
  source["sections"]["replay"] =
      d::section({{"events", J::array()}, {"windows", J::array()}});
  unsigned serial = 0;
  auto save = [&](J j) {
    j = s::seal_result(j);
    auto path = root + "/source-" + std::to_string(serial++) + ".json";
    write(path, j);
    return J{{"path", path},
             {"expected_sha256", j["content_sha256"]},
             {"pointer", "/sections/book_frames/data/0"}};
  };
  input["liquidity_pools"][0]["source"] = save(source);
  input["initial_state"]["cash"]["USD"] = q("1000");
  auto event = input["streams"][0]["events"][0];
  auto order = [&](const char *id, const char *quantity) {
    auto v = event;
    v["event_id"] = id;
    v["quantity_rule"]["value"] = q(quantity);
    return v;
  };
  input["streams"][0]["events"] = J::array(
      {order("first", "8"), order("second", "8"), order("third", "1")});
  J last_request;
  auto run = [&](J p) {
    p["output_path"] = root + "/account-" + std::to_string(serial++) + ".json";
    last_request = p;
    s::dispatch("account_simulate", p, e::no_deadline);
    return read(p["output_path"]);
  };
  auto refused = [&](J p) {
    auto path = root + "/account-" + std::to_string(serial) + ".json";
    bool bad = false;
    try {
      (void)run(p);
    } catch (const std::exception &) {
      bad = true;
    }
    check(bad);
    check(!std::filesystem::exists(path));
  };
  auto terminal = [](const J &j) -> J {
    return j["sections"]["terminal_state"]["data"];
  };
  auto ledger = [](const J &j) -> J {
    return j["sections"]["execution"]["data"];
  };
  auto result = run(input);
  auto rows = ledger(result);
  check(terminal(result)["cash"]["USD"] == q("894"));
  check(terminal(result)["positions"]["A"] == q("10"));
  check(terminal(result)["liquidity_remaining"]["A-depth"]["asks"] ==
        J::array({"0", "0"}));
  check(rows[0]["execution_outcome"]["data"]["filled_quantity"] == "8");
  check(rows[1]["execution_outcome"]["data"]["filled_quantity"] == "2");
  check(rows[2]["execution_outcome"]["data"]["disposition"] == "no_fill");
  write(root + "/request.json", input);
  write(root + "/expected.json", result);
  // The existing native liquidity operation and account-coupled sweep agree.
  J lp = templates["templates"]["liquidity"];
  lp["path"] = input["liquidity_pools"][0]["source"]["path"];
  lp["expected_sha256"] =
      input["liquidity_pools"][0]["source"]["expected_sha256"];
  lp["output_path"] = root + "/liquidity.json";
  lp["liquidity_mode"] = "shared_snapshot";
  lp["orders"] = J::array();
  for (auto quantity : {"8", "8", "1"})
    lp["orders"].push_back(
        {{"order_id", "o" + std::to_string(lp["orders"].size())},
         {"signal_id", "s0"},
         {"frame_source_ordinal", "1"},
         {"side", "buy"},
         {"quantity", quantity},
         {"limit_price_nanos", nullptr},
         {"time_in_force", "IOC"},
         {"max_level_participation", q("1")},
         {"activation_probability",
          {{"status", "supplied"}, {"value", q("1")}}}});
  s::dispatch("liquidity", lp, e::no_deadline);
  auto liq = read(lp["output_path"]);
  for (std::size_t i = 0; i < 3; ++i)
    check(rows[i]["execution_outcome"]["data"] ==
          liq["sections"]["execution"]["data"][i]["data"]);
  // Capital rejection rolls liquidity back; a later smaller order can use it.
  auto p = input;
  p["initial_state"]["cash"]["USD"] = q("60");
  p["streams"][0]["events"] =
      J::array({order("too-big", "8"), order("fits", "5")});
  auto rejected = run(p);
  check(terminal(rejected)["cash"]["USD"] == q("9"));
  check(terminal(rejected)["positions"]["A"] == q("5"));
  check(ledger(rejected)[0]["execution_outcome"]["committed"] == false);
  check(ledger(rejected)[0]["before"] == ledger(rejected)[0]["after"]);
  check(ledger(rejected)[1]["execution_outcome"]["data"]["fills"][0]
                        ["remaining_before"] == "4");
  p["on_constraint"] = "abort";
  refused(p);
  // A full-or-kill attempt does not consume capacity when it cannot fill.
  p = input;
  p["streams"][0]["events"] =
      J::array({order("fok", "11"), order("after-fok", "10")});
  p["streams"][0]["events"][0]["time_in_force"] = "FOK";
  auto fok = run(p);
  check(ledger(fok)[0]["execution_outcome"]["data"]["disposition"] ==
        "fok_canceled");
  check(terminal(fok)["positions"]["A"] == q("10"));
  // Position targets resolve against prior state, including earlier modeled
  // fills.
  p = input;
  p["mode"] = "stateful_feedback";
  p["streams"][0]["events"] = J::array(
      {order("target", "6"), order("reached", "6"), order("reduce", "3")});
  for (auto &v : p["streams"][0]["events"]) {
    v["quantity_rule"]["kind"] = "target_position";
    v["fee_fixed"] = q("1");
    v["fee_per_unit"] = q("1", "2");
  }
  auto target = run(p);
  check(terminal(target)["positions"]["A"] == q("3"));
  check(terminal(target)["cash"]["USD"] == q("1917", "2"));
  check(ledger(target)[1]["suppression_reason"] == "target_reached");
  check(ledger(target)[2]["resolved_quantity"] == q("-3"));
  check(target["sections"]["summary"]["data"]["signals"] == "2");
  write(root + "/feedback-request.json", p);
  write(root + "/feedback-expected.json", target);
  p["mode"] = "fixed_intents";
  refused(p);
  p = input;
  p["mode"] = "stateful_feedback";
  p["streams"][0]["events"][0]["guards"] = J::array({{{"balance", "cash"},
                                                      {"key", "USD"},
                                                      {"comparison", "lt"},
                                                      {"value", q("1")}}});
  auto guarded = run(p);
  check(ledger(guarded)[0]["execution_outcome"].is_null());
  check(ledger(guarded)[1]["execution_outcome"]["data"]["filled_quantity"] ==
        "8");
  // Explicit limits and fractional participation use the same sweep contract.
  p = input;
  p["streams"][0]["events"] = J::array({order("limited", "8")});
  p["streams"][0]["events"][0]["limit_price_nanos"] = "10000000000";
  p["streams"][0]["events"][0]["max_level_participation"] = q("1", "2");
  auto limited = run(p);
  check(terminal(limited)["positions"]["A"] == q("2"));
  check(terminal(limited)["cash"]["USD"] == q("980"));
  // Explicit branch preserves used capacity; it cannot replenish its parent
  // pool.
  p = input;
  p["streams"][0]["events"] = J::array({order("first", "8")});
  auto first = run(p);
  auto first_path = last_request["output_path"];
  p["initial_state"] = {{"kind", "reference"},
                        {"source",
                         {{"path", first_path},
                          {"expected_sha256", first["content_sha256"]},
                          {"pointer", "/sections/terminal_state/data"}}}};
  p["streams"][0]["events"] = J::array({order("branch", "8")});
  auto branch = run(p);
  check(terminal(branch) == terminal(result));
  check(ledger(branch)[0]["execution_outcome"]["data"]["filled_quantity"] ==
        "2");
  check(read(first_path) == first);
  p["liquidity_pools"][0]["pool_id"] = "different";
  refused(p);
  // Tampered or structurally wrong selected evidence is rejected.
  p = input;
  p["liquidity_pools"][0]["source"]["expected_sha256"] = std::string(64, '0');
  refused(p);
  p = input;
  p["liquidity_pools"].push_back(p["liquidity_pools"][0]);
  p["liquidity_pools"][1]["pool_id"] = "alias";
  refused(p);
  for (auto time : {"99", "101"}) {
    p = input;
    p["streams"][0]["events"][0]["time_ns"] = time;
    refused(p);
  }
  for (auto quantity : {"0", "18446744073709551616"}) {
    p = input;
    p["streams"][0]["events"][0]["quantity_rule"]["value"] = q(quantity);
    refused(p);
  }
  p = input;
  p["streams"][0]["events"][0]["quantity_rule"]["value"] = q("1", "2");
  refused(p);
  // Unknown depth is an explicit partial/skip, or abort. It is not a zero fill.
  auto missing = source;
  missing["sections"]["book_frames"]["data"][0]["status"] = "unavailable";
  missing["sections"]["book_frames"]["data"][0]["reason"] = "fixture gap";
  missing["sections"]["book_frames"]["data"][0]["data"] = nullptr;
  p = input;
  p["liquidity_pools"][0]["source"] = save(missing);
  refused(p);
  for (auto &v : p["streams"][0]["events"])
    v["on_unavailable"] = "skip_event";
  auto skipped = run(p);
  check(skipped["status"] == "partial");
  check(skipped["sections"]["summary"]["data"]["execution_unavailable"] == "3");
  check(terminal(skipped)["positions"]["A"] == q("0"));
  check(ledger(skipped)[0]["execution_outcome"]["data"].is_null());
  // Splits preserve exact marked equity and use caller bounds atomically.
  p = templates["variants"]["account_simulate"]["position_split"];
  auto split = run(p);
  check(terminal(split)["positions"]["A"] == q("6"));
  check(terminal(split)["marks"]["A"] == q("5"));
  check(split["sections"]["summary"]["data"]["terminal_equity_by_unit"]["USD"]
             ["value"] == q("130"));
  p["streams"][0]["events"][0]["factor"] = q("1", "2");
  auto reverse = run(p);
  check(terminal(reverse)["positions"]["A"] == q("3", "2"));
  check(terminal(reverse)["marks"]["A"] == q("20"));
  p["constraints"]["positions"]["A"]["minimum"] = q("2");
  auto split_reject = run(p);
  check(terminal(split_reject)["positions"]["A"] == q("3"));
  check(terminal(split_reject)["marks"]["A"] == q("10"));
  p["streams"][0]["events"][0]["factor"] = q("0");
  refused(p);
  p = input;
  p["streams"][0]["events"].push_back({{"event_id", "split-with-old-pool"},
                                       {"time_ns", "101"},
                                       {"kind", "position_split"},
                                       {"instrument_id", "A"},
                                       {"factor", q("2")}});
  refused(p);
  // Caller stream ordering controls competing orders at a shared timestamp.
  p = input;
  p["streams"] = J::array(
      {{{"stream_id", "one"}, {"events", J::array({order("large", "8")})}},
       {{"stream_id", "two"}, {"events", J::array({order("small", "5")})}}});
  auto tied = run(p);
  check(ledger(tied)[1]["execution_outcome"]["data"]["filled_quantity"] == "2");
  std::swap(p["streams"][0], p["streams"][1]);
  auto tied_reverse = run(p);
  check(
      ledger(tied_reverse)[1]["execution_outcome"]["data"]["filled_quantity"] ==
      "5");
  check(terminal(tied) == terminal(tied_reverse));
  // Large requested quantity is a caller choice; physical capacity still bounds
  // fills.
  p = input;
  p["streams"][0]["events"] =
      J::array({order("large-request", "18446744073709551615")});
  auto large = run(p);
  check(terminal(large)["positions"]["A"] == q("10"));
  // Negative prices and fee rebates stay exact through the same native sweep.
  auto negative_source = source;
  negative_source["sections"]["book_frames"]["data"][0]["data"]["bids"] =
      J::array({level("-12000000000", "5")});
  negative_source["sections"]["book_frames"]["data"][0]["data"]["asks"] =
      J::array({level("-11000000000", "4"), level("-10000000000", "6")});
  p = input;
  p["liquidity_pools"][0]["source"] = save(negative_source);
  p["streams"][0]["events"] = J::array({order("negative", "1")});
  p["streams"][0]["events"][0]["fee_fixed"] = q("-1");
  auto negative = run(p);
  check(terminal(negative)["cash"]["USD"] == q("1012"));
  // Full payload equality across native partitioned and legacy storage.
  p = input;
  p.erase("output_path");
  p["output"] =
      templates["variants"]["account_simulate"]["partitioned"]["output"];
  p["output"]["bundle_path"] = root + "/bundle";
  p["output"]["workspace_path"] = root + "/workspace";
  auto receipt = s::dispatch("account_simulate", p, e::no_deadline);
  J selected = {
      {"kind", "bundle"},
      {"reference", receipt["storage"]["reference"]},
      {"selector", {{"kind", "pointer"}, {"pointer", ""}}},
      {"read_options", {{"max_page_bytes", nullptr}, {"cache_bytes", "0"}}}};
  auto logical =
      d::consumer::value_source(selected, e::no_deadline).value.materialize();
  check(logical["sections"] == result["sections"]);
  auto branch_source = selected;
  branch_source["selector"]["pointer"] = "/sections/terminal_state/data";
  p = input;
  p["initial_state"] = {{"kind", "reference"}, {"source", branch_source}};
  p["streams"][0]["events"] = J::array({order("after-exhaustion", "1")});
  auto exhausted = run(p);
  check(terminal(exhausted) == terminal(result));
  check(ledger(exhausted)[0]["execution_outcome"]["data"]["disposition"] ==
        "no_fill");
  write(root + "/partitioned-receipt.json", receipt);
  std::cout << checks << " account execution checks\n";
  return 0;
} catch (const std::exception &ex) {
  std::cerr << ex.what() << '\n';
  return 1;
}
