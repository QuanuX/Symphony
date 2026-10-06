#include "../src/wide_rational.hpp"
#include <cstdlib>
#include <filesystem>
#include <fstream>
#include <iostream>
#include <source_location>
#include <unistd.h>
namespace s = symphony::sbv;
namespace d = s::detail;
namespace e = symphony::knowledge::engine;
namespace w = d::wide_rational;
using J = s::Json;
unsigned checks = 0;
void check(bool b, std::source_location l = std::source_location::current()) {
  ++checks;
  if (!b)
    throw std::runtime_error("allocation assertion line " +
                             std::to_string(l.line()));
}
J rat(const char *n, const char *d = "1") {
  return {{"numerator", n}, {"denominator", d}};
}
void write(const std::string &p, const J &j) {
  std::ofstream f(p);
  f << j.dump();
  check(f.good());
}
J read(const std::string &p) {
  std::ifstream f(p);
  J j;
  f >> j;
  return j;
}
int main() try {
  char tmp[] = "/private/tmp/sbv-allocation-XXXXXX";
  check(::mkdtemp(tmp) != nullptr);
  std::string root = tmp;
  struct Cleanup {
    std::string p;
    ~Cleanup() { std::filesystem::remove_all(p); }
  } cleanup{root};
  unsigned serial = 0;
  J order{{"order_id", "o0"},
          {"signal_id", "s0"},
          {"frame_source_ordinal", "1"},
          {"side", "buy"},
          {"quantity", "10"},
          {"time_in_force", "IOC"},
          {"activation_probability",
           {{"status", "supplied"}, {"value", rat("3", "5")}}}};
  J row{{"protocol", "symphony.sbv.liquidity-outcome.v1"},
        {"model_id", "displayed_depth_sweep"},
        {"model_version", "1"},
        {"order_id", "o0"},
        {"signal_id", "s0"},
        {"frame_source_ordinal", "1"},
        {"order", order},
        {"conditioning", "selected_order_active"},
        {"status", "available"},
        {"reason", ""},
        {"data",
         {{"filled_quantity", "8"},
          {"requested_quantity", "10"},
          {"unfilled_quantity", "2"},
          {"frame_available_ns", "100"},
          {"disposition", "partial_fill"},
          {"fills", J::array({{{"price_nanos", "101"}, {"quantity", "4"}},
                              {{"price_nanos", "102"}, {"quantity", "4"}}})},
          {"notional_price_nanos_times_quantity", "812"},
          {"vwap_price_nanos",
           {{"status", "available"}, {"value", rat("203", "2")}}}}}};
  auto source = d::base("mechanical allocation fixture; not market evidence");
  auto &sec = source["sections"];
  sec["execution"] = d::section(J::array({row}));
  sec["choices"] = d::section({{"protocol", "symphony.sbv.liquidity-input.v1"},
                               {"liquidity_mode", "independent"},
                               {"orders", J::array({order})}});
  sec["provenance"] =
      d::section({{"model_id", "displayed_depth_sweep"},
                  {"model_version", "1"},
                  {"source_census_sha256", std::string(64, 'a')}});
  sec["source_context"] = d::section({{"mechanical", true}});
  sec["book_frames"] = d::section(J::array());
  sec["book_checkpoint"] = d::section(nullptr, "not_selected", "fixture");
  sec["replay"] = d::section({{"test", "portable unchanged"}});
  J valuation{{"id", "filled_quantity_markout"},
              {"version", "1"},
              {"mark_price_nanos", "110"},
              {"mark_available_ns", "200"},
              {"mark_evidence", "caller mechanical scenario"},
              {"price_unit_nanos", "1"},
              {"value_per_price_unit", rat("2")},
              {"activation_cost", rat("1")},
              {"filled_order_cost", rat("2")},
              {"per_filled_unit_cost", rat("1", "2")},
              {"return_basis", rat("100")},
              {"pnl_unit", "test units"}};
  J p{{"protocol", "symphony.sbv.allocation-economics-input.v1"},
      {"path", ""},
      {"expected_sha256", ""},
      {"output_path", ""},
      {"selections", J::array({{{"order_id", "o0"},
                                {"valuation", valuation},
                                {"mode", "activation_mixture"},
                                {"inactive_pnl", rat("-3")}}})},
      {"studies", J::array({"allocation_costs", "activation_moments"})},
      {"on_unavailable", "unavailable"},
      {"extensions", J::object()}};
  auto bind = [&](J q, J j) {
    j = s::seal_result(j);
    auto path = root + "/source-" + std::to_string(serial++) + ".json";
    write(path, j);
    q["path"] = path;
    q["expected_sha256"] = j["content_sha256"];
    return q;
  };
  p = bind(p, source);
  auto call = [&](J q) {
    q["output_path"] = root + "/out-" + std::to_string(serial++) + ".json";
    auto receipt =
        s::dispatch("allocation_economics", q, e::unix_time_ms() + 30000);
    check(receipt["protocol"] == "symphony.sbv.allocation-economics.v1");
    return read(q["output_path"].get<std::string>());
  };
  auto refused = [&](J q) {
    q["output_path"] = root + "/bad-" + std::to_string(serial++) + ".json";
    bool fail = false;
    try {
      s::dispatch("allocation_economics", q, e::unix_time_ms() + 30000);
    } catch (const std::exception &) {
      fail = true;
    }
    check(fail);
    check(!std::filesystem::exists(q["output_path"].get<std::string>()));
  };
  auto outcome = [](const J &j) {
    return j.at("sections").at("economics").at("data").at(0);
  };
  auto value = [&](const J &j) {
    return outcome(j).at("conditional").at("value");
  };
  auto revise = [&](J q, J j) {
    j["sections"]["choices"]["data"]["orders"][0] =
        j["sections"]["execution"]["data"][0]["order"];
    return bind(q, j);
  };
  auto a = call(p);
  auto v = value(a);
  check(v["gross_markout"] == rat("136"));
  check(v["costs"]["total"] == rat("7"));
  check(v["net_markout"] == rat("129"));
  check(v["return"] == rat("129", "100"));
  check(v["filled_quantity"] == "8");
  check(v["unfilled_quantity"] == "2");
  check(outcome(a)["mixture"]["value"][1]["weight"] == rat("2", "5"));
  auto m = a["sections"]["studies"]["data"][1]["results"][0]["result"]["value"];
  check(m["expected_net_markout"] == rat("381", "5"));
  check(m["net_markout_variance"] == rat("104544", "25"));
  check(m["return_variance"] == rat("6534", "15625"));
  for (const auto *key :
       {"execution", "replay", "book_frames", "book_checkpoint", "signals"})
    check(a["sections"][key] == source["sections"][key]);
  auto q = p;
  q["selections"][0]["valuation"]["per_filled_unit_cost"] = rat("-1");
  check(value(call(q))["net_markout"] == rat("141"));
  q["selections"][0]["valuation"]["return_basis"] = rat("-100");
  check(value(call(q))["return"] == rat("-141", "100"));
  auto sell = source;
  sell["sections"]["execution"]["data"][0]["order"]["side"] = "sell";
  check(value(call(revise(p, sell)))["net_markout"] == rat("-143"));
  q = p;
  q["selections"][0]["mode"] = "conditional_only";
  q["selections"][0]["inactive_pnl"] = nullptr;
  check(outcome(call(q))["mixture"]["status"] == "not_selected");
  auto unknown = source;
  unknown["sections"]["execution"]["data"][0]["order"]
         ["activation_probability"] = {{"status", "unavailable"},
                                       {"reason", "not chosen"}};
  auto u = call(revise(p, unknown));
  check(outcome(u)["status"] == "partial");
  check(value(u)["net_markout"] == rat("129"));
  check(outcome(u)["mixture"]["status"] == "unavailable");
  check(outcome(call(revise(q, unknown)))["status"] == "available");
  auto r = revise(p, unknown);
  r["on_unavailable"] = "reject";
  refused(r);
  auto zero = source;
  auto &z = zero["sections"]["execution"]["data"][0];
  z["order"]["time_in_force"] = "FOK";
  z["data"]["filled_quantity"] = "0";
  z["data"]["unfilled_quantity"] = "10";
  z["data"]["fills"] = J::array();
  z["data"]["notional_price_nanos_times_quantity"] = "0";
  z["data"]["vwap_price_nanos"] = d::missing("no fill");
  z["data"]["disposition"] = "fok_canceled";
  auto zr = call(revise(p, zero));
  check(value(zr)["net_markout"] == rat("-1"));
  check(value(zr)["costs"]["filled_order"] == rat("0"));
  check(outcome(zr)["mixture"]["value"].size() == 2);
  check(outcome(zr)["mixture"]["value"][1]["net_markout"] == rat("-3"));
  for (const auto *prob : {"0", "1"}) {
    auto endpoint = source;
    endpoint["sections"]["execution"]["data"][0]["order"]
            ["activation_probability"]["value"] = rat(prob);
    auto result = call(revise(p, endpoint));
    check(result["sections"]["studies"]["data"][1]["results"][0]["result"]
                ["value"]["net_markout_variance"] == rat("0"));
  }
  auto shared = source;
  shared["sections"]["choices"]["data"]["liquidity_mode"] = "shared_snapshot";
  shared["sections"]["execution"]["data"][0]["conditioning"] =
      "all_orders_active_in_input_order";
  shared["sections"]["execution"]["data"][0]["order"]["activation_probability"]
        ["value"] = rat("1");
  check(outcome(call(revise(p, shared)))["conditioning"] ==
        "all_orders_active_in_input_order");
  shared["sections"]["execution"]["data"][0]["order"]["activation_probability"]
        ["value"] = rat("1", "2");
  refused(revise(p, shared));
  auto two = source;
  auto second = row;
  second["order_id"] = "o1";
  second["order"]["order_id"] = "o1";
  two["sections"]["execution"]["data"].push_back(second);
  two["sections"]["choices"]["data"]["orders"].push_back(second["order"]);
  auto subset = call(bind(p, two));
  check(subset["sections"]["execution"]["data"].size() == 2);
  check(subset["sections"]["economics"]["data"].size() == 1);
  two["sections"]["choices"]["data"]["liquidity_mode"] = "shared_snapshot";
  for (unsigned i = 0; i < 2; ++i) {
    auto &r = two["sections"]["execution"]["data"][i];
    r["conditioning"] = "all_orders_active_in_input_order";
    r["frame_source_ordinal"] = std::to_string(i + 1);
    r["order"]["frame_source_ordinal"] = r["frame_source_ordinal"];
    r["order"]["activation_probability"]["value"] = rat("1");
    two["sections"]["choices"]["data"]["orders"][i] = r["order"];
  }
  refused(bind(p, two));
  auto absent = source;
  absent["sections"]["execution"]["data"][0]["status"] = "unavailable";
  absent["sections"]["execution"]["data"][0]["reason"] = "no initialized book";
  absent["sections"]["execution"]["data"][0]["data"] = nullptr;
  check(outcome(call(bind(p, absent)))["status"] == "unavailable");
  auto missing_result = call(bind(p, absent));
  check(missing_result["sections"]["studies"]["data"][1]["results"][0]["result"]
                      ["reason"] == "no initialized book");
  q = bind(p, absent);
  q["on_unavailable"] = "reject";
  refused(q);
  auto large = source;
  auto &lr = large["sections"]["execution"]["data"][0];
  lr["order"]["quantity"] = "1000000000";
  lr["data"]["requested_quantity"] = "1000000000";
  lr["data"]["filled_quantity"] = "1000000000";
  lr["data"]["unfilled_quantity"] = "0";
  lr["data"]["disposition"] = "full_fill";
  lr["data"]["fills"] = J::array(
      {{{"price_nanos", "9007199254740993"}, {"quantity", "1000000000"}}});
  lr["data"]["notional_price_nanos_times_quantity"] =
      "9007199254740993000000000";
  lr["data"]["vwap_price_nanos"]["value"] = rat("9007199254740993");
  q = revise(p, large);
  q["selections"][0]["valuation"]["mark_price_nanos"] = "9007199254740994";
  check(value(call(q))["gross_markout"] == rat("2000000000"));
  // Optional variance can overflow while the selected conditional
  // transformation remains representable.
  q["selections"][0]["valuation"]["mark_price_nanos"] = "-9223372036854775808";
  refused(q);
  q["studies"] = J::array({"allocation_costs"});
  check(outcome(call(q))["status"] == "available");
  for (unsigned i = 0; i < 12; ++i) {
    auto bad = p;
    if (i == 0)
      bad["expected_sha256"] = std::string(64, 'b');
    if (i == 1)
      bad["selections"][0]["valuation"]["mark_available_ns"] = "99";
    if (i == 2)
      bad["selections"][0]["valuation"]["return_basis"] = rat("0");
    if (i == 3)
      bad["selections"][0]["valuation"]["price_unit_nanos"] = "0";
    if (i == 4)
      bad["selections"].push_back(bad["selections"][0]);
    if (i == 5)
      bad["selections"][0]["order_id"] = "unknown";
    if (i == 6)
      bad["studies"].push_back("allocation_costs");
    if (i == 7)
      bad["selections"][0]["valuation"]["mark_price_nanos"] =
          "9223372036854775807";
    if (i == 8)
      bad["selections"][0]["valuation"]["value_per_price_unit"] = rat("01");
    if (i == 9)
      bad["selections"][0]["mode"] = "conditional_only";
    if (i == 10)
      bad["selections"][0]["valuation"]["mark_evidence"] = "";
    if (i == 11)
      bad["extensions"] = nullptr;
    refused(bad);
  }
  for (unsigned i = 0; i < 9; ++i) {
    auto bad = source;
    auto &b = bad["sections"]["execution"]["data"][0];
    if (i == 0)
      b["data"]["notional_price_nanos_times_quantity"] = "813";
    if (i == 1)
      b["data"]["filled_quantity"] = "11";
    if (i == 2)
      b["data"]["fills"][0]["quantity"] = "0";
    if (i == 3)
      b["data"]["vwap_price_nanos"]["value"] = rat("102");
    if (i == 4)
      b["data"]["disposition"] = "full_fill";
    if (i == 5)
      b["order"]["time_in_force"] = "FOK";
    if (i == 6)
      b["order_id"] = "forged";
    if (i == 7)
      b["data"]["vwap_price_nanos"]["value"]["denominator"] = "0";
    if (i == 8)
      bad["sections"]["execution"]["data"].push_back(b);
    refused(revise(p, bad));
  }
  check(w::wire(w::times({w::max, 2}, {2, w::max})) == rat("1"));
  check(w::decimal(w::integer("170141183460469231731687303715884105727")) ==
        "170141183460469231731687303715884105727");
  q = p;
  q["selections"] = J::array();
  q["studies"] = J::array();
  auto empty = call(q);
  check(empty["sections"]["economics"]["data"].empty());
  check(empty["sections"]["studies"]["status"] == "not_selected");
  if (const char *emit = std::getenv("SYMPHONY_SBV_EMIT_ALLOCATION_FIXTURE")) {
    std::filesystem::path dir = emit;
    check(dir.is_absolute() && std::filesystem::is_directory(dir));
    auto fixture = s::seal_result(source);
    write((dir / "allocation-source.json").string(), fixture);
    q = p;
    q["path"] = (dir / "allocation-source.json").string();
    q["expected_sha256"] = fixture["content_sha256"];
    q["output_path"] = (dir / "allocation.json").string();
    write((dir / "allocation-request.json").string(), q);
  }
  std::cout << checks << " allocation economics assertions passed\n";
  return 0;
} catch (const std::exception &x) {
  std::cerr << x.what() << '\n';
  return 1;
}
