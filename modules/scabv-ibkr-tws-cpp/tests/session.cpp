#include <cstdio>
#include <cstdlib>
#include <limits>
#include <source_location>
#include <symphony/scabv/tws.hpp>
#include <thread>
using namespace symphony;
namespace tws = scabv::tws;
void check(bool x, std::source_location at = std::source_location::current()) {
  if (!x) {
    std::fprintf(stderr, "failure at %u\n", at.line());
    std::abort();
  }
}
struct Runtime final : tws::RuntimeDriver {
  int connects = 0, starts = 0, cancels = 0, closes = 0, pumps = 0;
  source::Status connect_result = source::Status::ok,
                 start_result = source::Status::ok;
  tws::Request last;
  tws::Session *session = nullptr;
  std::string generation;
  bool deliver = false;
  std::string_view release() const noexcept override { return "10.45"; }
  source::Status connect(const tws::Connection &c,
                         std::string_view g) override {
    check(c.host == "127.0.0.1" && c.port == 4002 && c.client_id == 71);
    generation = g;
    ++connects;
    return connect_result;
  }
  source::Status start(const tws::Request &r) override {
    last = r;
    ++starts;
    return start_result;
  }
  source::Status cancel(const tws::Request &r) override {
    check(r.request_id == last.request_id);
    ++cancels;
    return source::Status::ok;
  }
  void disconnect() noexcept override { ++closes; }
  source::Status pump() override {
    ++pumps;
    if (deliver) {
      deliver = false;
      check(session->position(generation, last.request_id,
                              {"U_TEST", "", 123, "TEST", "STK", "USD", "SMART",
                               "0.000000000000000001", -0.0}) ==
            source::Status::ok);
      check(session->end(generation, last.request_id) == source::Status::ok);
    }
    return source::Status::ok;
  }
};
tws::Connection connection() {
  return {"127.0.0.1", "U_TEST", "", "private:fixture", "test-session", 4002,
          71,          1000,     1};
}
tws::Request request() {
  tws::Request r;
  r.max_records = 10;
  r.max_payload_bytes = 4096;
  r.timeout_ms = 1000;
  return r;
}
void ready(tws::Session &s) {
  std::string accounts[]{"U_TEST"};
  check(s.next_valid_id(s.generation(), 9000) == source::Status::ok);
  check(s.managed_accounts(s.generation(), accounts) == source::Status::ok);
  check(s.state() == tws::SessionState::ready);
}
int main() {
  Runtime runtime;
  tws::Session s;
  runtime.session = &s;
  auto c = connection();
  auto bad = c;
  bad.host = "remote.example";
  check(tws::Session::open(bad, runtime, s) ==
            source::Status::invalid_argument &&
        runtime.connects == 0);
  check(tws::Session::open(c, runtime, s) == source::Status::ok);
  auto first = std::string(s.generation());
  int id = 77;
  check(s.start(request(), id) == source::Status::stale &&
        runtime.starts == 0 && id == 77);
  check(s.next_valid_id("stale", 1) == source::Status::binding_mismatch);
  ready(s);
  auto spoof = request();
  spoof.account = "OTHER";
  check(s.start(spoof, id) == source::Status::invalid_argument &&
        runtime.starts == 0);
  check(s.start(request(), id) == source::Status::ok && id == 1 &&
        runtime.last.account == "U_TEST");
  check(s.release_request() == source::Status::stale);
  check(s.position(first, id + 1, {}) == source::Status::binding_mismatch);
  check(s.fail(first, -1, 2104) == source::Status::binding_mismatch);
  runtime.deliver = true;
  check(s.poll() ==
        source::Status::stale); // End callback closed the active request.
  tws::Snapshot snapshot;
  check(s.snapshot(snapshot) == source::Status::ok &&
        snapshot.state() == tws::State::complete && snapshot.records() == 1);
  auto raw = std::string_view(
      reinterpret_cast<const char *>(snapshot.payload().data()),
      snapshot.payload().size());
  check(raw.find("0.000000000000000001") != raw.npos &&
        raw.find("9223372036854775808") != raw.npos);
  check(s.release_request() == source::Status::ok);
  check(s.start(request(), id) == source::Status::limit);
  s.close();
  check(s.reconnect() == source::Status::ok && s.generation() != first);
  check(s.next_valid_id(first, 1) == source::Status::binding_mismatch);
  ready(s);
  std::this_thread::sleep_for(std::chrono::milliseconds(110));
  check(s.start(request(), id) == source::Status::ok && id == 2);
  check(s.position(first, id, {}) == source::Status::binding_mismatch);
  check(s.disconnected(s.generation()) == source::Status::ok);
  check(s.snapshot(snapshot) == source::Status::ok &&
        snapshot.state() == tws::State::disconnected);
  check(s.reconnect() == source::Status::stale);
  check(s.release_request() == source::Status::ok);
  Runtime history_runtime;
  tws::Session hs;
  check(tws::Session::open(c, history_runtime, hs) == source::Status::ok);
  ready(hs);
  auto h = request();
  h.operation = tws::Operation::historical_bars;
  h.conid = 123;
  h.exchange = "SMART";
  h.end_utc = "20261101 12:00:00 UTC";
  h.duration = "1 D";
  h.bar = "1 min";
  check(hs.start(h, id) == source::Status::ok);
  check(hs.historical_bar(hs.generation(), id,
                          {"1793534400", 1, 2, 0, 1, "100.0000000000001",
                           "1.0000000000001", 1}) == source::Status::ok);
  check(hs.end(hs.generation(), id, "start", "end") == source::Status::ok);
  check(hs.release_request() == source::Status::ok);
  hs.close();
  check(hs.reconnect() == source::Status::ok);
  ready(hs);
  std::this_thread::sleep_for(std::chrono::milliseconds(110));
  check(hs.start(h, id) == source::Status::limit &&
        history_runtime.starts == 1);
  Runtime denied;
  tws::Session ds;
  check(tws::Session::open(c, denied, ds) == source::Status::ok);
  std::string accounts[]{"OTHER"};
  check(ds.managed_accounts(ds.generation(), accounts) ==
            source::Status::not_authorized &&
        denied.closes == 1);
  Runtime duplicate;
  tws::Session duplicate_session;
  check(tws::Session::open(c, duplicate, duplicate_session) ==
        source::Status::ok);
  std::string duplicates[]{"U_TEST", "U_TEST"};
  check(duplicate_session.managed_accounts(duplicate_session.generation(),
                                           duplicates) ==
        source::Status::malformed);
  Runtime oversized;
  tws::Session oversized_session;
  check(tws::Session::open(c, oversized, oversized_session) ==
        source::Status::ok);
  std::vector<std::string> oversized_accounts(1025, "U_TEST");
  check(oversized_session.managed_accounts(oversized_session.generation(),
                                           oversized_accounts) ==
            source::Status::malformed &&
        oversized.closes == 1);
  Runtime deadline;
  tws::Session timeout;
  c.handshake_timeout_ms = 1;
  check(tws::Session::open(c, deadline, timeout) == source::Status::ok);
  std::this_thread::sleep_for(std::chrono::milliseconds(3));
  check(timeout.next_valid_id(timeout.generation(), 1) ==
            source::Status::timeout &&
        deadline.pumps == 0);
  Runtime exhausted;
  tws::Session exhausted_session;
  c = connection();
  c.first_request_id = std::numeric_limits<int>::max();
  check(tws::Session::open(c, exhausted, exhausted_session) ==
        source::Status::ok);
  ready(exhausted_session);
  check(exhausted_session.start(request(), id) == source::Status::ok &&
        id == std::numeric_limits<int>::max());
  std::stop_source stop;
  stop.request_stop();
  check(exhausted_session.poll(stop.get_token()) == source::Status::cancelled);
  check(exhausted_session.snapshot(snapshot) == source::Status::ok &&
        snapshot.state() == tws::State::cancelled);
  check(exhausted_session.release_request() == source::Status::ok);
  check(exhausted_session.reconnect() == source::Status::ok);
  ready(exhausted_session);
  check(exhausted_session.start(request(), id) == source::Status::limit);
  Runtime uncertain;
  tws::Session failure;
  uncertain.start_result = source::Status::transport_error;
  c = connection();
  check(tws::Session::open(c, uncertain, failure) == source::Status::ok);
  ready(failure);
  check(failure.start(request(), id) == source::Status::transport_error);
  check(failure.snapshot(snapshot) == source::Status::ok &&
        snapshot.state() == tws::State::failed);
  std::puts("TWS host session: handshake/account, monotonic IDs, routing, "
            "Decimal strings, pacing, cancel, gap, reconnect and exhaustion "
            "passed; fixture only");
}
