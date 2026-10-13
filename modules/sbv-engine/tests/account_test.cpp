#include "../src/consumer_source.hpp"
#include <filesystem>
#include <fstream>
#include <iostream>
#include <source_location>
#include <symphony/sbv/sdk.hpp>
#include <unistd.h>
namespace s = symphony::sbv;
namespace d = s::detail;
namespace e = symphony::knowledge::engine;
using J = s::Json;
unsigned checks = 0;
void check(bool ok, std::source_location at = std::source_location::current()) {
  ++checks;
  if (!ok)
    throw std::runtime_error("account assertion line " +
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
int main(int argc, char **argv) try {
  check(argc == 2);
  char temp[] = "/private/tmp/sbv-account-XXXXXX";
  check(mkdtemp(temp) != nullptr);
  std::string root = temp;
  struct Cleanup {
    std::string path;
    ~Cleanup() { std::filesystem::remove_all(path); }
  } cleanup{root};
  if (const char *retain = getenv("SBV_ACCOUNT_FIXTURE_ROOT")) {
    root = retain;
    std::filesystem::create_directories(root);
    root = std::filesystem::canonical(root).string();
  }
  const auto templates = read(argv[1]);
  const auto input = templates.at("templates").at("account_simulate");
  unsigned serial = 0;
  J last_request, last_receipt;
  auto run = [&](J p) {
    p["output_path"] = root + "/account-" + std::to_string(serial++) + ".json";
    last_request = p;
    last_receipt = s::dispatch("account_simulate", p, e::no_deadline);
    return read(p.at("output_path"));
  };
  auto refused = [&](J p) {
    bool bad = false;
    auto dest = root + "/account-" + std::to_string(serial) + ".json";
    try {
      (void)run(p);
    } catch (const std::exception &) {
      bad = true;
    }
    check(bad);
    check(!std::filesystem::exists(dest));
  };
  const auto result = run(input);
  const auto &sections = result.at("sections");
  const auto &summary = sections.at("summary").at("data");
  check(summary.at("events") == "5" && summary.at("signals") == "4");
  check(summary.at("applied") == "4" &&
        summary.at("constraint_rejected") == "1");
  check(summary.at("terminal_equity_by_unit").at("USD").at("value") ==
        q("115"));
  const auto &terminal = sections.at("terminal_state").at("data");
  check(terminal.at("cash").at("USD") == q("60"));
  check(terminal.at("positions").at("A") == q("0") &&
        terminal.at("positions").at("B") == q("1"));
  const auto &ledger = sections.at("execution").at("data");
  check(ledger.at(1).at("event").at("event_id") == "compete-B");
  check(ledger.at(1).at("before") == ledger.at(1).at("after"));
  check(ledger.at(1).at("cash_delta").empty());
  check(ledger.at(2).at("event").at("event_id") == "sell-A");
  check(sections.at("snapshots").at("data").size() == 6);
  write(root + "/request.json", input);
  write(root + "/expected.json", result);
  // Hand-computed reversed tie: B spends 50 first, A's 60 buy fails,
  // A's later unbounded short sale credits 70; second B buy leaves 70 cash.
  auto p = input;
  std::swap(p["streams"][0], p["streams"][1]);
  auto reversed = run(p);
  check(reversed["sections"]["terminal_state"]["data"]["cash"]["USD"] ==
        q("70"));
  check(reversed["sections"]["terminal_state"]["data"]["positions"]["A"] ==
        q("-1"));
  auto feedback = templates["variants"]["account_simulate"]["feedback"];
  auto f = run(feedback);
  check(f["sections"]["summary"]["data"]["signals"] == "3");
  check(f["sections"]["summary"]["data"]["guard_suppressed"] == "1");
  check(f["sections"]["terminal_state"]["data"] == terminal);
  feedback["mode"] = "fixed_intents";
  refused(feedback);
  p = input;
  p["constraints"]["cash"]["USD"]["minimum"] = nullptr;
  auto unrestricted = run(p);
  check(unrestricted["sections"]["terminal_state"]["data"]["cash"]["USD"] ==
        q("10"));
  check(unrestricted["sections"]["terminal_state"]["data"]["positions"]["B"] ==
        q("2"));
  check(unrestricted["sections"]["execution"]["data"][1]["after"]["cash"]
                    ["USD"] == q("-10"));
  p = input;
  p["on_constraint"] = "abort";
  refused(p);
  p = input;
  p["streams"][0]["events"][1]["time_ns"] = "0";
  refused(p);
  p = input;
  p["streams"][0]["events"][1]["event_id"] = "buy-A";
  refused(p);
  p = input;
  p["instruments"][0]["settlement"] = "futures_margin";
  refused(p);
  p = input;
  p["streams"][0]["events"][0]["quantity"] = q("0");
  refused(p);
  p = input;
  p["initial_state"]["positions"]["unknown"] = q("0");
  refused(p);
  p = input;
  p["streams"][0]["events"][0]["price"] = q("1", "0");
  refused(p);
  p = input;
  p["constraints"]["positions"]["A"]["maximum"] = q("0");
  check(run(p)["sections"]["execution"]["data"][0]["disposition"] ==
        "constraint_rejected");
  p = input;
  p["retain_snapshots"] = false;
  check(run(p)["sections"]["snapshots"]["status"] == "not_selected");
  p = input;
  p["streams"][1]["events"].erase(2);
  check(run(p)["sections"]["summary"]["data"]["terminal_equity_by_unit"]["USD"]
              ["status"] == "unavailable");
  // Exact fractional quantity, signed price and fee rebate.
  p = input;
  p["streams"] = J::array({p["streams"][0]});
  p["streams"][0]["events"].erase(1);
  p["streams"][0]["events"][0]["quantity"] = q("1", "3");
  p["streams"][0]["events"][0]["price"] = q("-6");
  p["streams"][0]["events"][0]["fee"] = q("-1", "2");
  auto fractional = run(p);
  check(fractional["sections"]["terminal_state"]["data"]["cash"]["USD"] ==
        q("205", "2"));
  check(fractional["sections"]["terminal_state"]["data"]["positions"]["A"] ==
        q("1", "3"));
  // FX uses two declared legs; no inferred conversion or unit aggregation.
  p = input;
  p["cash_units"].push_back("EUR");
  p["initial_state"]["cash"]["EUR"] = q("0");
  p["constraints"]["cash"]["EUR"] = {{"minimum", nullptr},
                                     {"maximum", nullptr}};
  p["streams"] = J::array({{{"stream_id", "fx"},
                            {"events", J::array({{{"event_id", "convert"},
                                                  {"time_ns", "1"},
                                                  {"kind", "fx"},
                                                  {"from_unit", "USD"},
                                                  {"to_unit", "EUR"},
                                                  {"from_amount", q("20")},
                                                  {"to_amount", q("18")}},
                                                 {{"event_id", "fee"},
                                                  {"time_ns", "2"},
                                                  {"kind", "cashflow"},
                                                  {"unit", "EUR"},
                                                  {"amount", q("-1")}}})}}});
  auto fx = run(p);
  check(fx["sections"]["terminal_state"]["data"]["cash"]["USD"] == q("80"));
  check(fx["sections"]["terminal_state"]["data"]["cash"]["EUR"] == q("17"));
  // Branch the immutable exact terminal snapshot, never rewrite its parent.
  p = input;
  p["initial_state"] = {{"kind", "reference"},
                        {"source",
                         {{"path", root + "/expected.json"},
                          {"expected_sha256", result.at("content_sha256")},
                          {"pointer", "/sections/terminal_state/data"}}}};
  p["streams"] = J::array({{{"stream_id", "branch"},
                            {"events", J::array({{{"event_id", "dividend"},
                                                  {"time_ns", "4"},
                                                  {"kind", "cashflow"},
                                                  {"unit", "USD"},
                                                  {"amount", q("5")}}})}}});
  auto branch = run(p);
  check(branch["sections"]["terminal_state"]["data"]["cash"]["USD"] == q("65"));
  check(read(root + "/expected.json") == result);
  auto mismatch = p;
  mismatch["instruments"][0]["price_multiplier"] = q("2");
  refused(mismatch);
  mismatch = p;
  mismatch["streams"][0]["events"][0]["time_ns"] = "2";
  refused(mismatch);
  // Partitioned producer and shared reader preserve the same complete science.
  auto part = input;
  part.erase("output_path");
  part["output"] = {
      {"kind", "partitioned"},
      {"bundle_path", root + "/bundle"},
      {"workspace_path", root + "/scratch"},
      {"write_options", {{"page_bytes", "4096"}, {"index_fanout", "8"}}}};
  auto receipt = s::dispatch("account_simulate", part, e::no_deadline);
  check(receipt.at("status") == "completed");
  J source = {
      {"kind", "bundle"},
      {"reference", receipt["storage"]["reference"]},
      {"selector", {{"kind", "pointer"}, {"pointer", ""}}},
      {"read_options", {{"max_page_bytes", nullptr}, {"cache_bytes", "0"}}}};
  auto copy =
      d::consumer::value_source(source, e::no_deadline).value.materialize();
  for (auto name : {"summary", "execution", "signals", "snapshots",
                    "initial_state", "terminal_state"})
    check(copy["sections"][name] == result["sections"][name]);
  auto stream_source = source;
  stream_source["selector"]["pointer"] = "/sections/event_inputs/data/0/events";
  auto referenced = input;
  referenced["streams"][0]["events"] = {{"kind", "referenced_rows"},
                                        {"source", stream_source}};
  check(run(referenced)["sections"]["execution"] ==
        result["sections"]["execution"]);
  p["initial_state"]["source"] = source;
  p["initial_state"]["source"]["selector"]["pointer"] =
      "/sections/terminal_state/data";
  check(run(p)["sections"]["terminal_state"] ==
        branch["sections"]["terminal_state"]);
  // C ABI/C++ SDK executes the same request/result contract.
  p = input;
  p["output_path"] = root + "/sdk.json";
  J envelope = {{"protocol", e::process_protocol_v2},
                {"request_id", "account"},
                {"correlation_id", "account"},
                {"operation", "account_simulate"},
                {"target_engine", "symphony-sbv"},
                {"deadline_unix_ms", nullptr},
                {"payload", p}};
  auto sdk = s::sdk::process(envelope.dump());
  check(sdk.status == 0);
  auto native_sdk = read(p["output_path"]);
  check(native_sdk["sections"]["execution"] == result["sections"]["execution"]);
  // Cancellation and expired deadlines never publish a result.
  p = input;
  p["output_path"] = root + "/cancelled.json";
  std::atomic_bool cancel{true};
  bool stopped = false;
  {
    d::CancellationScope scope(&cancel);
    try {
      (void)s::dispatch("account_simulate", p, e::no_deadline);
    } catch (const e::Error &error) {
      stopped = error.code() == "sbv.cancelled";
    }
  }
  check(stopped &&
        !std::filesystem::exists(p["output_path"].get<std::string>()));
  write(root + "/partitioned-receipt.json", receipt);
  std::cout << checks << " account assertions passed\n";
  return 0;
} catch (const std::exception &error) {
  std::cerr << error.what() << '\n';
  return 1;
}
