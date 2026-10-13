#include "../src/provider.hpp"
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
    throw std::runtime_error("feedback/settlement assertion line " +
                             std::to_string(at.line()));
}
J read(const std::string &p) {
  std::ifstream f(p);
  J j;
  f >> j;
  return j;
}
std::string bytes(const std::string &p) {
  std::ifstream f(p, std::ios::binary);
  return {std::istreambuf_iterator<char>(f), {}};
}
void write(const std::string &p, const J &j) {
  std::ofstream f(p);
  f << j.dump();
  check(f.good());
}
J q(const char *n, const char *den = "1") {
  return {{"numerator", n}, {"denominator", den}};
}
J event(const char *id, const char *time, const char *quantity,
        const char *price, const char *fee = "0") {
  return {{"event_id", id},          {"time_ns", time},
          {"kind", "fill_intent"},   {"instrument_id", "A"},
          {"quantity", q(quantity)}, {"price", q(price)},
          {"fee", q(fee)},           {"guards", J::array()}};
}
J trigger(const char *id, const char *time, const J &context = J::object()) {
  return {{"event_id", id},
          {"time_ns", time},
          {"kind", "feedback_trigger"},
          {"context", context}};
}
int main(int argc, char **argv) try {
  check(argc == 4);
  char temp[] = "/private/tmp/sbv-feedback-XXXXXX";
  check(mkdtemp(temp));
  std::string root = temp;
  struct Cleanup {
    std::string path;
    ~Cleanup() { std::filesystem::remove_all(path); }
  } cleanup{root};
  if (auto p = getenv("SBV_FEEDBACK_FIXTURE_ROOT")) {
    root = p;
    std::filesystem::create_directories(root);
    root = std::filesystem::canonical(root).string();
  }
  auto templates = read(argv[1]);
  auto input = templates["variants"]["account_simulate"]["native_feedback"];
  auto &selection = input["feedback"]["provider"];
  selection["library"] = {
      {"path", std::filesystem::canonical(argv[2]).string()},
      {"expected_sha256", e::sha256_hex(bytes(argv[2]))}};
  input["initial_state"]["cash"]["USD"] = q("100");
  input["streams"][0]["events"] =
      J::array({trigger("one", "1"), trigger("two", "2")});
  unsigned serial = 0;
  J last;
  auto run = [&](J p) {
    p["output_path"] = root + "/result-" + std::to_string(serial++) + ".json";
    last = p;
    s::dispatch("account_simulate", p, e::no_deadline);
    return read(p["output_path"]);
  };
  auto refused = [&](J p) {
    auto path = root + "/result-" + std::to_string(serial) + ".json";
    bool bad = false;
    try {
      run(p);
    } catch (const std::exception &) {
      bad = true;
    }
    check(bad);
    check(!std::filesystem::exists(path));
  };
  auto terminal = [](const J &j) {
    return j.at("sections").at("terminal_state").at("data");
  };
  auto ledger = [](const J &j) {
    return j.at("sections").at("execution").at("data");
  };
  auto log = root + "/lifetime.log";
  check(setenv("SBV_FEEDBACK_LIFETIME_LOG", log.c_str(), 1) == 0);
  auto result = run(input);
  auto rows = ledger(result);
  auto state = terminal(result);
  check(state["protocol"] == "symphony.sbv.account-state.v3");
  check(state["cash"]["USD"] == q("100"));
  check(state["positions"]["A"] == q("0"));
  check(state["feedback_memory"]["calls"] == "2");
  check(state["feedback_memory"]["prior_position"] == q("1"));
  check(rows[0]["feedback"]["call"]["previous_event"].is_null());
  check(rows[1]["feedback"]["call"]["state"]["positions"]["A"] == q("1"));
  check(rows[1]["feedback"]["call"]["state"]["time_ns"] == "1");
  check(rows[0]["event"]["kind"] == "feedback_trigger");
  check(rows[0]["resolved_event"]["kind"] == "fill_intent");
  check(bytes(log) == "release\ncreate\nrelease\nrelease\ndestroy\n");
  d::validate_native_provider_evidence(
      selection, result["sections"]["feedback_provider"]["data"]);
  write(root + "/request.json", input);
  write(root + "/expected.json", result);
  auto branch = input;
  branch["initial_state"] = {{"kind", "reference"},
                             {"source",
                              {{"path", last["output_path"]},
                               {"expected_sha256", result["content_sha256"]},
                               {"pointer", "/sections/snapshots/data/1"}}}};
  // Snapshot section name follows the public operation contract.
  branch["initial_state"]["source"]["pointer"] = "/sections/snapshots/data/1";
  branch["streams"][0]["events"] = J::array({trigger("two", "2")});
  check(terminal(run(branch)) == state);
  auto bad = branch;
  bad["feedback"]["provider"]["parameters"]["different"] = true;
  refused(bad);
  for (const char *mode : {"failed", "cancelled", "malformed", "wrong_time",
                           "recursive", "bad_memory"}) {
    bad = input;
    bad["streams"][0]["events"] =
        J::array({trigger("one", "1", {{"mode", mode}})});
    refused(bad);
  }
  bad = input;
  bad["mode"] = "fixed_intents";
  refused(bad);
  bad = input;
  bad["feedback"]["provider"]["concurrency"] = "shared_reentrant_instance";
  refused(bad);
  bad = input;
  bad["feedback"]["provider"]["library"]["expected_sha256"] =
      std::string(64, '0');
  refused(bad);
  bad = input;
  bad["feedback"]["provider"]["library"] = {
      {"path", std::filesystem::canonical(argv[3]).string()},
      {"expected_sha256", e::sha256_hex(bytes(argv[3]))}};
  refused(bad);
  bad = input;
  bad["feedback"]["provider"]["parameters"]["fail_create"] = true;
  refused(bad);
  check(bytes(log).ends_with("create\nrelease\ndestroy\n"));
  auto skip = input;
  skip["feedback"]["on_unavailable"] = "skip_trigger";
  skip["streams"][0]["events"] = J::array(
      {trigger("one", "1", {{"mode", "unavailable"}}), trigger("two", "2")});
  auto skipped = run(skip);
  check(terminal(skipped)["feedback_memory"]["calls"] == "1");
  check(ledger(skipped)[0]["disposition"] == "execution_unavailable");
  check(terminal(skipped)["positions"]["A"] == q("1"));
  skip["feedback"]["on_unavailable"] = "abort";
  refused(skip);
  skip = input;
  skip["streams"][0]["events"] = J::array(
      {trigger("one", "1", {{"mode", "no_action"}}), trigger("two", "2")});
  auto suppressed = run(skip);
  check(terminal(suppressed)["feedback_memory"]["calls"] == "2");
  check(terminal(suppressed)["positions"]["A"] == q("1"));
  check(ledger(suppressed)[0]["suppression_reason"] == "provider_no_action");
  auto rejected = input;
  rejected["constraints"]["positions"]["A"]["maximum"] = q("0");
  auto rejection = run(rejected);
  check(terminal(rejection)["positions"]["A"] == q("0"));
  check(terminal(rejection)["feedback_memory"]["calls"] == "2");
  check(terminal(rejection)["feedback_memory"]["previous_disposition"] ==
        "constraint_rejected");
  {
    d::NativeProvider p(selection, e::no_deadline);
    auto instance = p.create(e::no_deadline);
    auto call = rows[0]["feedback"]["call"];
    call["context"]["mode"] = "wait_cancel";
    bool cancelled = false;
    try {
      auto reply = instance->decide(call, e::unix_time_ms() + 20);
      cancelled = reply.status == SBV_PROVIDER_CANCELLED;
    } catch (const e::Error &) {
      cancelled = true;
    }
    check(cancelled);
  }
  unsetenv("SBV_FEEDBACK_LIFETIME_LOG");
  auto variation =
      templates["variants"]["account_simulate"]["linear_variation"];
  variation["initial_state"]["cash"]["USD"] = q("1000");
  variation["initial_state"]["positions"]["A"] = q("0");
  variation["initial_state"]["basis"]["A"] = q("0");
  variation["instruments"][0]["price_multiplier"] = q("5");
  auto settle = J{{"event_id", "settle"},
                  {"time_ns", "5"},
                  {"kind", "settle"},
                  {"instrument_id", "A"},
                  {"price", q("8")}};
  variation["streams"][0]["events"] = J::array(
      {event("open", "1", "2", "10", "1"), event("add", "2", "2", "14", "1"),
       event("close", "3", "-1", "15", "1"),
       event("reverse", "4", "-5", "11", "1"), settle,
       event("cover", "6", "2", "9", "1")});
  auto v = run(variation);
  auto vr = ledger(v);
  auto vt = terminal(v);
  check(vt["cash"]["USD"] == q("1015"));
  check(vt["positions"]["A"] == q("0"));
  check(vt["basis"]["A"] == q("0"));
  check(vr[0]["cash_delta"]["USD"] == q("-1"));
  check(vr[2]["cash_delta"]["USD"] == q("14"));
  check(vr[3]["cash_delta"]["USD"] == q("-16"));
  check(vr[4]["cash_delta"]["USD"] == q("30"));
  check(vr[5]["cash_delta"]["USD"] == q("-11"));
  write(root + "/variation-request.json", variation);
  write(root + "/variation-expected.json", v);
  auto vb = variation;
  vb["initial_state"] = {{"kind", "reference"},
                         {"source",
                          {{"path", last["output_path"]},
                           {"expected_sha256", v["content_sha256"]},
                           {"pointer", "/sections/snapshots/data/4"}}}};
  vb["streams"][0]["events"] =
      J::array({settle, event("cover", "6", "2", "9", "1")});
  check(terminal(run(vb)) == vt);
  auto small = variation;
  small["streams"][0]["events"] = J::array(
      {event("negative", "1", "-2", "-5"), event("cover", "2", "1", "-7")});
  auto negative = run(small);
  check(terminal(negative)["cash"]["USD"] == q("1010"));
  check(terminal(negative)["basis"]["A"] == q("-5"));
  small = variation;
  small["streams"][0]["events"] = J::array(
      {event("fraction", "1", "1", "10"), event("fraction2", "2", "2", "11")});
  check(terminal(run(small))["basis"]["A"] == q("32", "3"));
  small = variation;
  small["streams"][0]["events"] = J::array({event("open", "1", "2", "10"),
                                            J{{"event_id", "split"},
                                              {"time_ns", "2"},
                                              {"kind", "position_split"},
                                              {"instrument_id", "A"},
                                              {"factor", q("2")}},
                                            settle});
  auto split = run(small);
  check(terminal(split)["positions"]["A"] == q("4"));
  check(terminal(split)["cash"]["USD"] == q("1060"));
  small = variation;
  small["constraints"]["cash"]["USD"]["minimum"] = q("1000");
  small["streams"][0]["events"] =
      J::array({event("open", "1", "2", "10", "1")});
  auto rollback = run(small);
  check(terminal(rollback)["basis"]["A"] == q("0"));
  check(terminal(rollback)["positions"]["A"] == q("0"));
  check(terminal(rollback)["cash"]["USD"] == q("1000"));
  bad = variation;
  bad["initial_state"].erase("basis");
  refused(bad);
  bad = variation;
  bad["instruments"][0]["settlement"] = "cash_notional";
  bad["initial_state"].erase("basis");
  refused(bad);
  small = variation;
  small["constraints"]["cash"]["USD"]["minimum"] = nullptr;
  small["initial_state"]["marks"]["A"] = q("10");
  small["streams"][0]["events"] =
      J::array({event("zero-wealth", "1", "1", "10", "1000")});
  check(terminal(run(small))["cash"]["USD"] == q("0"));
  small["streams"][0]["events"][0]["fee"] = q("1001");
  check(terminal(run(small))["cash"]["USD"] == q("-1"));
  small = variation;
  small["initial_state"]["positions"]["A"] = q("2");
  small["initial_state"]["basis"]["A"] = q("10");
  small["initial_state"]["marks"]["A"] = q("10");
  small["constraints"]["cash"]["USD"]["minimum"] = q("1000");
  small["streams"][0]["events"] = J::array({settle});
  auto settlement_rejected = run(small);
  check(terminal(settlement_rejected)["basis"]["A"] == q("10"));
  check(terminal(settlement_rejected)["marks"]["A"] == q("10"));
  check(terminal(settlement_rejected)["cash"]["USD"] == q("1000"));
  auto combined = input;
  combined["instruments"][0]["settlement"] = "linear_variation";
  combined["initial_state"]["basis"] = {{"A", q("0")}};
  check(terminal(run(combined))["cash"]["USD"] == q("100"));
  std::cout << "feedback/settlement assertions=" << checks << "\n";
  return 0;
} catch (const std::exception &ex) {
  std::cerr << ex.what() << '\n';
  return 1;
}
