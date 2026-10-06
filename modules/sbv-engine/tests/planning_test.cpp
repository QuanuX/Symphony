#include "../src/detail.hpp"
#include <cstdlib>
#include <filesystem>
#include <fstream>
#include <iostream>
#include <source_location>
#include <symphony/sbv/interop.hpp>
#include <unistd.h>
namespace s = symphony::sbv;
namespace d = s::detail;
namespace e = symphony::knowledge::engine;
namespace i = s::interop;
using J = s::Json;
unsigned checks = 0;
void check(bool b, std::source_location l = std::source_location::current()) {
  ++checks;
  if (!b)
    throw std::runtime_error("planning assertion " + std::to_string(l.line()));
}
J rat(const char *n, const char *d = "1") {
  return {{"numerator", n}, {"denominator", d}};
}
int main() try {
  auto call = [](const char *op, const J &p) {
    return s::dispatch(op, p, e::unix_time_ms() + 30000);
  };
  auto refused = [&](const char *op, J p) {
    bool fail = false;
    try {
      call(op, p);
    } catch (const std::exception &) {
      fail = true;
    }
    check(fail);
  };
  J backend{{"protocol", "symphony.sbv.backend-plan-input.v1"},
            {"backend", "cuda"},
            {"fallback", nullptr},
            {"limit_policy", "reject"},
            {"scheduling", "independent"},
            {"workers", "4"},
            {"memory_bytes", "1024"},
            {"declared_limits", {{"workers", "8"}, {"memory_bytes", "2048"}}},
            {"buffer", nullptr},
            {"extensions", J::object()}};
  auto r = call("backend_plan", backend);
  check(r["status"] == "unavailable");
  check(r["resolved"]["backend"] == "cuda");
  check(r["applied"].is_null() && r["reservation"] == false);
  auto p = backend;
  p["fallback"] = "cpu";
  r = call("backend_plan", p);
  check(r["status"] == "planned" && r["resolved"]["fallback_used"] == true);
  check(r["resolved"]["workers"] == "4");
  p["scheduling"] = "serial_state";
  check(call("backend_plan", p)["status"] == "unavailable");
  p["limit_policy"] = "reduce";
  check(call("backend_plan", p)["resolved"]["workers"] == "1");
  p["scheduling"] = "independent";
  p["workers"] = "100";
  p["declared_limits"]["workers"] = "80";
  p["memory_bytes"] = "4096";
  r = call("backend_plan", p);
  check(r["resolved"]["workers"] == "64");
  check(r["resolved"]["memory_bytes"] == "2048");
  J buffer{{"dtype", "f32"},
           {"device", "cpu"},
           {"owner_id", "caller-lease"},
           {"stream_id", "host"},
           {"storage_bytes", "32"},
           {"byte_offset", "0"},
           {"shape", J::array({"2", "3"})},
           {"byte_strides", J::array({"16", "4"})},
           {"completion", "ready"}};
  p["buffer"] = buffer;
  r = call("backend_plan", p);
  check(r["buffer"]["required_bytes"] == "28");
  check(r["buffer"]["status"] == "descriptor_ready");
  auto tight = p;
  tight["declared_limits"]["memory_bytes"] = "16";
  check(call("backend_plan", tight)["status"] == "unavailable");
  for (const auto *status : {"pending", "failed", "cancelled"}) {
    auto q = p;
    q["buffer"]["completion"] = status;
    check(call("backend_plan", q)["status"] == "unavailable");
  }
  auto q = p;
  q["buffer"]["device"] = "cuda";
  check(call("backend_plan", q)["status"] == "unavailable");
  q = p;
  q["buffer"]["shape"] = J::array({"0", "18446744073709551615"});
  q["buffer"]["byte_strides"] = J::array({"0", "18446744073709551612"});
  check(call("backend_plan", q)["buffer"]["empty"] == true);
  for (unsigned n = 0; n < 8; ++n) {
    q = p;
    if (n == 0)
      q["workers"] = "0";
    if (n == 1)
      q["buffer"]["storage_bytes"] = "27";
    if (n == 2)
      q["buffer"]["byte_strides"][0] = "15";
    if (n == 3)
      q["buffer"]["shape"][0] = "18446744073709551615";
    if (n == 4)
      q["buffer"]["owner_id"] = "";
    if (n == 5)
      q["buffer"]["dtype"] = "magic";
    if (n == 6)
      q["buffer"]["byte_offset"] = "1";
    if (n == 7)
      q["buffer"]["byte_strides"].push_back("4");
    refused("backend_plan", q);
  }
  // Compileable completion/ownership contract without vendor runtime headers.
  struct Done : i::Completion {
    i::CompletionState current = i::CompletionState::pending;
    i::CompletionState state() const noexcept override { return current; }
    i::CompletionState wait_until(std::uint64_t) override { return current; }
    bool request_cancel() noexcept override {
      current = i::CompletionState::cancelled;
      return true;
    }
  };
  auto owner = std::make_shared<int>(7);
  auto done = std::make_shared<Done>();
  i::TensorLease lease{
      {"i32", "cpu", "lease", "host", 4, 0, {}, {}}, owner, owner.get(), done};
  lease.validate();
  bool denied = false;
  try {
    lease.ready_host_base();
  } catch (const std::exception &) {
    denied = true;
  }
  check(denied);
  done->current = i::CompletionState::ready;
  check(lease.ready_host_base() == owner.get());
  check(done->request_cancel());
  check(done->state() == i::CompletionState::cancelled);
  // A disconnected capture plan preserves inclusive equal-time windows and
  // loss.
  auto event = [](const char *t, bool signal = false, const char *bytes = "8") {
    return J{{"available_ns", t}, {"bytes", bytes}, {"signal", signal}};
  };
  J live{{"protocol", "symphony.sbv.live-plan-input.v1"},
         {"pre_ns", "10"},
         {"post_ns", "10"},
         {"event_cap", "8"},
         {"byte_cap", "64"},
         {"pending_cap", "4"},
         {"overflow", "reject"},
         {"pending_overflow", "reject"},
         {"observed_until_ns", "30"},
         {"events", J::array({event("0"), event("10", true), event("10"),
                              event("20"), event("21")})},
         {"extensions", J::object()}};
  r = call("live_plan", live);
  check(r["can_activate"] == false && r["provider_requests"] == "0");
  check(r["windows"][0]["retained_event_count"] == "4");
  check(r["windows"][0]["status"] == "fixture_complete");
  check(r["peak"]["bytes"] == "32");
  q = live;
  q["event_cap"] = "2";
  refused("live_plan", q);
  q["overflow"] = "drop_oldest";
  r = call("live_plan", q);
  check(r["windows"][0]["status"] == "fixture_loss");
  check(r["windows"][0]["retained_event_count"] == "2");
  check(r["dropped_events"].size() == 2);
  q["overflow"] = "drop_newest";
  r = call("live_plan", q);
  check(r["windows"][0]["retained_event_count"] == "2");
  q = live;
  q["observed_until_ns"] = "21";
  q["post_ns"] = "20";
  check(call("live_plan", q)["windows"][0]["status"] == "pending");
  q = live;
  q["events"][2]["signal"] = true;
  q["pending_cap"] = "1";
  refused("live_plan", q);
  q["pending_overflow"] = "skip_signal";
  check(call("live_plan", q)["skipped_signal_indices"] == J::array({"2"}));
  q = live;
  q["pre_ns"] = "0";
  q["post_ns"] = "0";
  r = call("live_plan", q);
  check(r["windows"][0]["retained_event_count"] == "2");
  q = live;
  q["events"][0]["bytes"] = "100";
  q["overflow"] = "drop_oldest";
  check(call("live_plan", q)["dropped_events"][0]["reason"] ==
        "event_larger_than_capacity");
  for (unsigned n = 0; n < 5; ++n) {
    q = live;
    if (n == 0)
      q["events"][2]["available_ns"] = "9";
    if (n == 1)
      q["events"][0]["bytes"] = "0";
    if (n == 2)
      q["post_ns"] = "18446744073709551615";
    if (n == 3)
      q["observed_until_ns"] = "20";
    if (n == 4)
      q["event_cap"] = "0";
    refused("live_plan", q);
  }
  q = live;
  q["events"] = J::array();
  check(call("live_plan", q)["windows"].empty());
  // Typed stable selection over exact source values, including private studies.
  char tmp[] = "/private/tmp/sbv-selection-XXXXXX";
  check(::mkdtemp(tmp) != nullptr);
  std::string root = tmp;
  struct Cleanup {
    std::string p;
    ~Cleanup() { std::filesystem::remove_all(p); }
  } cleanup{root};
  unsigned serial = 0;
  auto source = d::base("private extension selection fixture");
  source["sections"]["studies"] = d::section(J::array(
      {{{"label", "large"}, {"n", "9007199254740993"}, {"r", rat("3", "5")}},
       {{"label", "small"}, {"n", "9007199254740992"}, {"r", rat("1", "2")}},
       {{"label", "tie"}, {"n", "9007199254740993"}, {"r", rat("6", "10")}},
       {{"label", "missing"}}}));
  auto bind = [&](J x, J src) {
    src = s::seal_result(src);
    auto file = root + "/source-" + std::to_string(serial++) + ".json";
    std::ofstream f(file);
    f << src.dump();
    f.close();
    x["path"] = file;
    x["expected_sha256"] = src["content_sha256"];
    return x;
  };
  J select{{"protocol", "symphony.sbv.result-select-input.v1"},
           {"path", ""},
           {"expected_sha256", ""},
           {"pointer", "/sections/studies/data"},
           {"filters", J::array()},
           {"order_by", J::array({{{"pointer", "/n"},
                                   {"type", "integer"},
                                   {"direction", "desc"},
                                   {"missing", "last"}}})},
           {"columns", J::array({{{"name", "name"}, {"pointer", "/label"}},
                                 {{"name", "n"}, {"pointer", "/n"}}})},
           {"limit", "2"},
           {"cursor", ""}};
  select = bind(select, source);
  r = call("result_select", select);
  check(r["rows"][0]["source_index"] == "0");
  check(r["rows"][1]["source_index"] == "2");
  check(r["complete"] == false);
  q = select;
  q["cursor"] = r["next_cursor"];
  auto page = call("result_select", q);
  check(page["rows"][0]["source_index"] == "1");
  check(page["rows"][1]["value"]["n"]["status"] == "unavailable");
  check(page["complete"] == true);
  q["limit"] = "1";
  refused("result_select", q);
  q = select;
  q["filters"] = J::array({{{"pointer", "/r"},
                            {"type", "rational"},
                            {"op", "gt"},
                            {"value", rat("1", "2")},
                            {"missing", "exclude"}}});
  r = call("result_select", q);
  check(r["matched_rows"] == "2");
  check(r["complete"] == true);
  q["filters"][0]["missing"] = "include";
  check(call("result_select", q)["matched_rows"] == "3");
  q["filters"][0]["missing"] = "reject";
  refused("result_select", q);
  q = select;
  q["order_by"][0]["missing"] = "first";
  check(call("result_select", q)["rows"][0]["source_index"] == "3");
  q = select;
  q["order_by"][0]["missing"] = "reject";
  refused("result_select", q);
  auto wide = source;
  wide["sections"]["studies"]["data"] =
      J::array({{{"r", rat("170141183460469231731687303715884105726",
                           "170141183460469231731687303715884105727")}},
                {{"r", rat("170141183460469231731687303715884105725",
                           "170141183460469231731687303715884105726")}},
                {{"r", rat("-1", "2")}}});
  q = select;
  q["order_by"][0] = {{"pointer", "/r"},
                      {"type", "rational"},
                      {"direction", "asc"},
                      {"missing", "reject"}};
  q["columns"] = J::array();
  q["limit"] = "3";
  r = call("result_select", bind(q, wide));
  check(r["rows"][0]["source_index"] == "2");
  check(r["rows"][1]["source_index"] == "1");
  check(r["rows"][2]["source_index"] == "0");
  for (unsigned n = 0; n < 7; ++n) {
    q = select;
    if (n == 0)
      q["expected_sha256"] = std::string(64, 'b');
    if (n == 1)
      q["columns"].push_back(q["columns"][0]);
    if (n == 2)
      q["limit"] = "0";
    if (n == 3)
      q["order_by"][0]["type"] = "float";
    if (n == 4)
      q["pointer"] = "/sections/studies";
    if (n == 5)
      q["order_by"][0]["pointer"] = "bad";
    if (n == 6)
      q["order_by"][0]["type"] = "boolean";
    refused("result_select", q);
  }
  if (const char *emit = std::getenv("SYMPHONY_SBV_EMIT_PLANNING_FIXTURE")) {
    std::filesystem::path dir = emit;
    check(dir.is_absolute() && std::filesystem::is_directory(dir));
    auto write = [&](const char *n, const J &v) {
      std::ofstream f(dir / n);
      f << v.dump();
      check(f.good());
    };
    auto fixture = s::seal_result(source);
    write("selection-source.json", fixture);
    q = select;
    q["path"] = (dir / "selection-source.json").string();
    q["expected_sha256"] = fixture["content_sha256"];
    write("selection-request.json", q);
    write("backend-request.json", backend);
    write("live-request.json", live);
  }
  std::cout << checks << " planning/selection/interop assertions passed\n";
  return 0;
} catch (const std::exception &x) {
  std::cerr << x.what() << '\n';
  return 1;
}
