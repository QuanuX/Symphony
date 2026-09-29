#ifdef SQV_VERIFY_RETENTION
#include "retention.hpp"
#endif
#include <bit>
#include <cstdio>
#include <cstdlib>
#include <symphony/scabv/tws.hpp>
#include <symphony/source/json.hpp>
#include <thread>
using namespace symphony;
namespace tws = scabv::tws;
void check(bool x) {
  if (!x)
    std::abort();
}
struct Contract {
  int conId = 0;
  std::string exchange;
};
struct Tags {};
struct FakeClient {
  int starts = 0, cancels = 0;
  bool history = false;
  void reqPositionsMulti(int id, const std::string &a, const std::string &m) {
    check(id > 0 && a == "U_TEST" && m.empty());
    ++starts;
  }
  void reqHistoricalData(int id, const Contract &c, const std::string &,
                         const std::string &, const std::string &,
                         const std::string &what, int rth, int format,
                         bool live, Tags) {
    check(id > 0 && c.conId == 123 && c.exchange == "SMART" &&
          what == "TRADES" && rth == 1 && format == 2 && !live);
    ++starts;
    history = true;
  }
  void cancelPositionsMulti(int) { ++cancels; }
  void cancelHistoricalData(int) { ++cancels; }
  // Deliberately no order methods: compilation proves the adapter does not need
  // them.
};
tws::Request request() {
  return {tws::Operation::positions,
          1,
          "session-1",
          "U_TEST",
          "",
          "private:test",
          0,
          "",
          "",
          "",
          "",
          100,
          65536,
          1000};
}
tws::Position position() {
  return {"U_TEST", "",    123,     "TEST",
          "STK",    "USD", "SMART", "0.000000000000000001",
          -0.0};
}
int main() {
  FakeClient client;
  tws::Sdk1045Driver<FakeClient, Contract, Tags> driver(client);
  tws::Collector c;
  auto r = request();
  check(tws::Collector::start(r, driver, c) == source::Status::ok &&
        client.starts == 1);
  check(c.position("stale", 1, position()) == source::Status::binding_mismatch);
  check(c.position("session-1", 2, position()) ==
        source::Status::binding_mismatch);
  check(c.position("session-1", 1, position()) == source::Status::ok);
  tws::Snapshot snap;
  check(c.snapshot(snap) == source::Status::stale);
  check(c.end("session-1", 1) == source::Status::ok && client.cancels == 1);
  check(c.position("session-1", 1, position()) == source::Status::stale);
  check(c.snapshot(snap) == source::Status::ok &&
        snap.state() == tws::State::complete && snap.records() == 1);
  auto raw = std::string(reinterpret_cast<const char *>(snap.payload().data()),
                         snap.payload().size());
  check(raw.find("0.000000000000000001") != std::string::npos &&
        raw.find("9223372036854775808") != std::string::npos);
  sqav::Capture capture;
  check(snap.capture("attempt", "fixture",
                     {sqav::TimeRole::acquisition, "2026-09-28", "iso-date",
                      "UTC", "day", "fixture"},
                     {131072, 16384, 4096}, capture) == source::Status::ok &&
        capture.description().coverage == sqav::Coverage::complete);
  c.reset();
  r.request_id = 2;
  check(tws::Collector::start(r, driver, c) == source::Status::ok);
  auto mixed = position();
  mixed.account = "OTHER";
  check(c.position("session-1", 2, mixed) == source::Status::binding_mismatch &&
        c.end("session-1", 2) == source::Status::stale);
  check(c.snapshot(snap) == source::Status::ok &&
        snap.state() == tws::State::failed);
  c.reset();
  r.request_id = 3;
  check(tws::Collector::start(r, driver, c) == source::Status::ok);
  check(c.disconnected("session-1") == source::Status::ok &&
        c.snapshot(snap) == source::Status::ok &&
        snap.state() == tws::State::disconnected);
  check(snap.capture("attempt", "fixture",
                     {sqav::TimeRole::acquisition, "2026-09-28", "iso-date",
                      "UTC", "day", "fixture"},
                     {131072, 16384, 4096}, capture) == source::Status::ok &&
        capture.description().coverage == sqav::Coverage::gap);
  c.reset();
  r.request_id = 4;
  r.max_records = 1;
  check(tws::Collector::start(r, driver, c) == source::Status::ok);
  check(c.position("session-1", 4, position()) == source::Status::ok &&
        c.position("session-1", 4, position()) == source::Status::limit);
  check(c.end("session-1", 4) == source::Status::stale);
  c.reset();
  r = request();
  r.operation = tws::Operation::historical_bars;
  r.conid = 123;
  r.exchange = "SMART";
  r.end_utc = "20260925 16:00:00 UTC";
  r.duration = "1 D";
  r.bar = "1 min";
  check(tws::Collector::start(r, driver, c) == source::Status::ok &&
        client.history);
  check(c.historical_bar("session-1", 1,
                         {"1790341200", 1, 2, 0, 1, "100.0001",
                          "1.123456789012345", 5}) == source::Status::ok);
  check(c.end("session-1", 1, "start", "end") == source::Status::ok &&
        c.snapshot(snap) == source::Status::ok && snap.records() == 1);
  c.reset();
  r = request();
  check(tws::Collector::start(r, driver, c) == source::Status::ok);
  std::stop_source stop;
  stop.request_stop();
  check(c.poll(stop.get_token()) == source::Status::cancelled);
  check(c.snapshot(snap) == source::Status::ok &&
        snap.state() == tws::State::cancelled);
  c.reset();
  r = request();
  r.timeout_ms = 1;
  check(tws::Collector::start(r, driver, c) == source::Status::ok);
  std::this_thread::sleep_for(std::chrono::milliseconds(3));
  check(c.poll() == source::Status::timeout);
  c.reset();
  r = request();
  check(tws::Collector::start(r, driver, c) == source::Status::ok);
  std::vector<std::jthread> workers;
  for (unsigned i = 0; i < 16; ++i)
    workers.emplace_back([&, i] {
      auto p = position();
      p.conid = 100 + i;
      check(c.position("session-1", 1, p) == source::Status::ok);
    });
  workers.clear();
  check(c.end("session-1", 1) == source::Status::ok &&
        c.snapshot(snap) == source::Status::ok && snap.records() == 16);
#ifdef SQV_VERIFY_RETENTION
  source_composition::verify(capture);
#endif
  std::puts("TWS: read-only SDK shape, exact account/generation, initial "
            "end/cancel, gaps, bounds, price bits and deadline passed; no "
            "vendor SDK conformance claimed");
}
