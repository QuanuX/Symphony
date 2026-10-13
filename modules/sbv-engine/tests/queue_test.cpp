#include <private_temp.hpp>
#include "../src/detail.hpp"
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
void check(bool v, std::source_location loc = std::source_location::current()) {
  ++checks;
  if (!v)
    throw std::runtime_error("queue assertion line " +
                             std::to_string(loc.line()));
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
J rat(const char *n, const char *den = "1") {
  return {{"numerator", n}, {"denominator", den}};
}
int main(int argc, char **argv) try {
  check(argc == 2);
  auto temp = symphony::testing::temporary_pattern("sbv-queue-XXXXXX");
  check(mkdtemp(temp.data()) != nullptr);
  std::string root = temp;
  struct Cleanup {
    std::string path;
    ~Cleanup() { std::filesystem::remove_all(path); }
  } cleanup{root};
  if (const char *retain = getenv("SBV_QUEUE_FIXTURE_ROOT")) {
    root = retain;
    std::filesystem::create_directories(root);
    root = std::filesystem::canonical(root).string();
  }
  auto source =
      d::base("mechanical queue fixture, not market or calibrated evidence");
  auto &ss = source["sections"];
  ss["signals"] =
      d::section(J::array({{{"signal_id", "s0"},
                            {"source_ordinal", "0"},
                            {"available_ns", "100"},
                            {"anchor_price_nanos", "100"},
                            {"causal_end_ordinal_exclusive", "1"}}}));
  ss["summary"] = d::section({{"closed_census", true},
                              {"parent_census_sha256", std::string(64, 'a')}});
  ss["provenance"] = d::section({{"profile", "databento_mbo_orders_strict_v1"},
                                 {"scope", "mechanical fixture"}});
  auto level = [](const char *price, const char *q) {
    return J{{"price_nanos", price}, {"size", q}, {"order_count", "1"}};
  };
  auto frame = [&](unsigned n, const char *time, const char *queue) {
    return J{{"protocol", "symphony.sbv.book-frame.v1"},
             {"layer", "reconstructed"},
             {"source_ordinal", std::to_string(n)},
             {"available_ns", time},
             {"status", "available"},
             {"reason", ""},
             {"event_boundary", true},
             {"receive_time_flagged", false},
             {"data",
              {{"bids", J::array({level("100", queue)})},
               {"asks", J::array({level("110", "10")})},
               {"bid_level_count", "1"},
               {"ask_level_count", "1"},
               {"market_state", "uncrossed"}}}};
  };
  ss["book_frames"] = d::section(J::array(
      {frame(0, "100", "5"), frame(1, "110", "5"), frame(2, "120", "3")}));
  ss["book_checkpoint"] = d::section(nullptr, "not_selected", "fixture");
  J events = J::array();
  auto event = [&](unsigned n, unsigned t, char action, unsigned size,
                   char side = 'B') {
    events.push_back({{"source_ordinal", std::to_string(n)},
                      {"ts_recv", std::to_string(t)},
                      {"action", std::to_string(action)},
                      {"size", std::to_string(size)},
                      {"side", std::to_string(side)},
                      {"flags", "128"},
                      {"price_nanos", "100"}});
  };
  event(0, 100, 'N', 0);
  event(1, 110, 'T', 7);
  event(2, 120, 'C', 2);
  event(3, 130, 'A', 3);
  event(4, 140, 'T', 8);
  event(5, 150, 'N', 0);
  ss["replay"] = d::section({{"events", events}, {"windows", J::array()}});
  unsigned serial = 0;
  auto save = [&](J v) {
    auto sealed = s::seal_result(v);
    auto path = root + "/source-" + std::to_string(serial++) + ".json";
    write(path, sealed);
    return J{{"path", path}, {"expected_sha256", sealed["content_sha256"]}};
  };
  auto ref = save(source);
  auto p = read(argv[1])["templates"]["queue_scenarios"];
  p["path"] = ref["path"];
  p["expected_sha256"] = ref["expected_sha256"];
  p["orders"] = J::array({{{"order_id", "o0"},
                           {"signal_id", "s0"},
                           {"side", "buy"},
                           {"price_nanos", "100"},
                           {"quantity", "6"},
                           {"horizon_ns", "50"},
                           {"initial_queue_ahead", nullptr},
                           {"activation_probability", rat("3", "5")}}});
  p["latency_atoms"] = J::array(
      {{{"id", "fast"}, {"delay_ns", "0"}, {"weight", rat("1", "2")}},
       {{"id", "slow"}, {"delay_ns", "20"}, {"weight", rat("1", "2")}}});
  p["model"]["cancel_ahead_fraction"] = rat("1");
  p["model"]["replenish_ahead_fraction"] = rat("1");
  auto call = [&](J q) {
    q["output_path"] = root + "/result-" + std::to_string(serial++) + ".json";
    auto receipt = s::dispatch("queue_scenarios", q, e::no_deadline);
    check(receipt.at("protocol") == "symphony.sbv.queue-scenarios.v1");
    return read(q["output_path"]);
  };
  auto refuse = [&](J q) {
    bool bad = false;
    try {
      call(q);
    } catch (const std::exception &) {
      bad = true;
    }
    check(bad);
  };
  auto with_source = [&](J v) {
    auto q = p;
    auto r = save(v);
    q["path"] = r["path"];
    q["expected_sha256"] = r["expected_sha256"];
    return q;
  };
  auto result = call(p);
  auto rows = result["sections"]["queue_scenarios"]["data"];
  check(rows.size() == 2);
  check(rows[0]["data"]["filled_quantity"] == "6");
  check(rows[0]["data"]["served_ahead"] == "8");
  check(rows[0]["data"]["service_quantity"] == "15");
  check(rows[0]["data"]["unused_service_quantity"] == "1");
  check(rows[0]["data"]["first_fill_ns"] == "110");
  check(rows[0]["data"]["last_fill_ns"] == "140");
  check(rows[1]["data"]["filled_quantity"] == "2");
  check(rows[1]["data"]["served_ahead"] == "6");
  auto summary = result["sections"]["execution"]["data"][0];
  check(summary["any_fill_probability"]["value"] == rat("3", "5"));
  check(summary["complete_fill_probability"]["value"] == rat("3", "10"));
  check(summary["no_fill_probability"]["value"] == rat("2", "5"));
  check(summary["expected_filled_quantity"]["value"] == rat("12", "5"));
  for (auto k : {"signals", "replay", "book_frames"})
    check(result["sections"][k] == source["sections"][k]);
  auto q = p;
  q["workers"] = "1";
  auto one = call(q);
  check(one["sections"]["queue_scenarios"] ==
        result["sections"]["queue_scenarios"]);
  q = p;
  q["orders"].push_back(q["orders"][0]);
  q["orders"][1]["order_id"] = "o1";
  auto multi = call(q);
  check(multi["sections"]["resources"]["data"]["actual_workers"] == "2");
  q["workers"] = "1";
  auto serial_result = call(q);
  check(serial_result["sections"]["execution"] ==
        multi["sections"]["execution"]);
  check(serial_result["sections"]["queue_scenarios"] ==
        multi["sections"]["queue_scenarios"]);
  q = p;
  q["model"]["priority"] = "pro_rata";
  auto pro = call(q)["sections"]["queue_scenarios"]["data"];
  check(pro[0]["data"]["fills"][0]["quantity"] == "3");
  check(pro[0]["data"]["filled_quantity"] == "6");
  check(pro[1]["data"]["filled_quantity"] == "4");
  q = p;
  q["latency_atoms"] =
      J::array({{{"id", "late"}, {"delay_ns", "51"}, {"weight", rat("1")}}});
  check(
      call(
          q)["sections"]["queue_scenarios"]["data"][0]["data"]["disposition"] ==
      "arrival_after_horizon");
  q = p;
  q["orders"][0]["quantity"] = "18446744073709551615";
  q["orders"][0]["initial_queue_ahead"] = "0";
  q["model"]["replenish_ahead_fraction"] = rat("0");
  check(call(q)["sections"]["queue_scenarios"]["data"][0]["data"]
               ["filled_quantity"] == "15");
  q = p;
  q["orders"][0]["activation_probability"] = rat("0");
  check(call(q)["sections"]["execution"]["data"][0]["any_fill_probability"]
               ["value"] == rat("0"));
  auto missing = source;
  missing["sections"]["book_frames"]["data"][0]["status"] = "unavailable";
  missing["sections"]["book_frames"]["data"][0]["data"] = nullptr;
  q = with_source(missing);
  auto absent = call(q);
  check(absent["sections"]["execution"]["data"][0]["any_fill_probability"]
              ["status"] == "unavailable");
  q["on_unavailable"] = "reject";
  refuse(q);
  auto gap = source;
  gap["sections"]["replay"]["data"]["events"][3]["flags"] = "132";
  check(call(with_source(
            gap))["sections"]["queue_scenarios"]["data"][0]["status"] ==
        "unavailable");
  gap = source;
  gap["sections"]["replay"]["data"]["events"].erase(3);
  check(call(with_source(
            gap))["sections"]["queue_scenarios"]["data"][0]["status"] ==
        "unavailable");
  auto reset = source;
  reset["sections"]["replay"]["data"]["events"][3]["action"] = "82";
  q = with_source(reset);
  check(call(q)["sections"]["queue_scenarios"]["data"][0]["status"] ==
        "unavailable");
  q["model"]["on_reset"] = "cancel_remaining";
  auto canceled = call(q)["sections"]["queue_scenarios"]["data"][0]["data"];
  check(canceled["filled_quantity"] == "2");
  check(canceled["disposition"] == "reset_canceled_remainder");
  q = p;
  q["orders"][0]["price_nanos"] = "110";
  check(call(q)["sections"]["queue_scenarios"]["data"][0]["status"] ==
        "unavailable");
  q = p;
  q["orders"][0]["horizon_ns"] = "51";
  check(call(q)["sections"]["queue_scenarios"]["data"][0]["status"] ==
        "unavailable");
  q = p;
  q["latency_atoms"][0]["weight"] = rat("3", "4");
  refuse(q);
  q = p;
  q["model"]["priority"] = "magic";
  refuse(q);
  q = p;
  q["orders"][0]["quantity"] = "0";
  refuse(q);
  q = p;
  q["latency_atoms"][0]["delay_ns"] = "18446744073709551615";
  refuse(q);
  // Opposite side and action selections are independent from venue-specific
  // semantics.
  auto sell = source;
  for (auto &f : sell["sections"]["book_frames"]["data"]) {
    f["data"]["bids"][0]["price_nanos"] = "90";
    f["data"]["asks"][0]["price_nanos"] = "100";
    f["data"]["asks"][0]["size"] = f["source_ordinal"] == "2" ? "3" : "5";
  }
  for (auto &v : sell["sections"]["replay"]["data"]["events"])
    v["side"] = "65";
  q = with_source(sell);
  q["orders"][0]["side"] = "sell";
  check(call(q)["sections"]["execution"]["data"][0]["expected_filled_quantity"]
               ["value"] == rat("12", "5"));
  q = p;
  q["model"]["service_side"] = "opposite";
  check(call(q)["sections"]["execution"]["data"][0]["any_fill_probability"]
               ["value"] == rat("0"));
  q = p;
  q["model"]["service_action"] = "F";
  check(call(q)["sections"]["execution"]["data"][0]["any_fill_probability"]
               ["value"] == rat("0"));
  q = p;
  q["execution_labels"] = {{"domain", "fixture"},
                           {"source_kind", "synthetic"},
                           {"source_reference", "mechanical hand fixture"},
                           {"permission", "declared_permitted"},
                           {"training_ids", J::array()},
                           {"rows", J::array({{{"label_id", "l0"},
                                               {"order_id", "o0"},
                                               {"domain", "fixture"},
                                               {"partition", "held_out"},
                                               {"filled_quantity", "2"},
                                               {"horizon_ns", "50"}}})},
                           {"on_unavailable", "unavailable"}};
  auto assessed = call(q)["sections"]["execution_label_assessment"];
  check(assessed["data"]["brier_any_fill"]["value"] == rat("4", "25"));
  check(assessed["data"]["calibration"] == "not_established");
  auto labeled = q;
  q["execution_labels"]["training_ids"].push_back("s0");
  check(call(q)["sections"]["execution_label_assessment"]["status"] ==
        "unavailable");
  q = labeled;
  q["execution_labels"]["permission"] = "unavailable";
  check(call(q)["sections"]["execution_label_assessment"]["status"] ==
        "unavailable");
  q["execution_labels"]["on_unavailable"] = "reject";
  refuse(q);
  q = labeled;
  q["execution_labels"]["rows"][0]["domain"] = "other";
  check(call(q)["sections"]["execution_label_assessment"]["status"] ==
        "unavailable");
  // Held-out rows before training rows cannot hide reuse of a signal.
  q = labeled;
  q["orders"].push_back(q["orders"][0]);
  q["orders"][1]["order_id"] = "training-order";
  auto training = q["execution_labels"]["rows"][0];
  training["label_id"] = "training-label";
  training["order_id"] = "training-order";
  training["partition"] = "training";
  q["execution_labels"]["rows"].push_back(training);
  check(call(q)["sections"]["execution_label_assessment"]["status"] ==
        "unavailable");
  // Arrival follows every event at the same timestamp, including service.
  auto tied = source;
  tied["sections"]["replay"]["data"]["events"][1]["ts_recv"] = "100";
  tied["sections"]["book_frames"]["data"][1]["available_ns"] = "100";
  q = with_source(tied);
  auto tied_row = call(q)["sections"]["queue_scenarios"]["data"][0]["data"];
  check(tied_row["arrival_frame_ordinal"] == "1");
  check(tied_row["service_quantity"] == "8");
  check(tied_row["first_fill_ns"] == "140");
  // The inclusive horizon includes a service event exactly at its boundary.
  q = p;
  q["orders"][0]["horizon_ns"] = "40";
  check(call(q)["sections"]["queue_scenarios"]["data"][0]["data"]
               ["filled_quantity"] == "6");
  q["orders"][0]["horizon_ns"] = "39";
  check(call(q)["sections"]["queue_scenarios"]["data"][0]["data"]
               ["filled_quantity"] == "2");
  // Pro-rata reduction supports a full uint64 request without intermediate
  // overflow.
  q = p;
  q["model"]["priority"] = "pro_rata";
  q["model"]["replenish_ahead_fraction"] = rat("0");
  q["orders"][0]["quantity"] = "18446744073709551615";
  q["orders"][0]["initial_queue_ahead"] = "0";
  check(call(q)["sections"]["queue_scenarios"]["data"][0]["data"]
               ["filled_quantity"] == "15");
  // Explicit modification policy and zero-mass unavailable branch behavior.
  auto modified = source;
  modified["sections"]["replay"]["data"]["events"][3]["action"] = "77";
  q = with_source(modified);
  check(call(q)["sections"]["queue_scenarios"]["data"][0]["status"] ==
        "unavailable");
  q["model"]["on_modify"] = "ignore";
  check(call(q)["sections"]["queue_scenarios"]["data"][0]["data"]
               ["filled_quantity"] == "6");
  q = with_source(missing);
  q["latency_atoms"][0]["weight"] = rat("0");
  q["latency_atoms"][1]["weight"] = rat("1");
  check(call(q)["sections"]["execution"]["data"][0]["any_fill_probability"]
               ["value"] == rat("3", "5"));
  q = p;
  q["orders"] = J::array();
  check(call(q)["sections"]["summary"]["data"]["scenario_count"] == "0");
  // Retain portable fixtures for independent installed CLI/SDK consumers.
  auto invalidated = source;
  invalidated["sections"]["diagnostics"] = d::section(
      {{"events",
        J::array({{{"kind", "invalidation"},
                   {"source_ordinal", "3"},
                   {"reason", "mechanical absent-order cancellation"}}})}});
  q = with_source(invalidated);
  check(call(q)["sections"]["queue_scenarios"]["data"][0]["reason"] ==
        "source book invalidated after arrival");
  p["output_path"] = root + "/installed-queue.json";
  write(root + "/request.json", p);
  write(root + "/expected.json", result);
  write(root + "/labeled-request.json", labeled);
  std::cout << checks << " queue checks passed\n";
  return 0;
} catch (const std::exception &ex) {
  std::cerr << ex.what() << '\n';
  return 1;
}
