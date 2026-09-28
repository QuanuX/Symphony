#include <cstdio>
#include <cstdlib>
#include <new>
#include <symphony/scabv/tws.hpp>
thread_local long fail_at = -1;
void *operator new(std::size_t n) {
  if (fail_at >= 0 && fail_at-- == 0) {
    fail_at = -1;
    throw std::bad_alloc();
  }
  if (auto p = std::malloc(n ? n : 1))
    return p;
  throw std::bad_alloc();
}
void operator delete(void *p) noexcept { std::free(p); }
void *operator new[](std::size_t n) { return ::operator new(n); }
void operator delete[](void *p) noexcept { ::operator delete(p); }
using namespace symphony;
namespace tws = scabv::tws;
void check(bool b) {
  if (!b)
    std::abort();
}
struct Driver : tws::Driver {
  std::string_view release() const noexcept override {
    return tws::sdk_release;
  }
  source::Status start(const tws::Request &) override {
    return source::Status::ok;
  }
  source::Status cancel(const tws::Request &) override {
    return source::Status::ok;
  }
};
int main() {
  Driver d;
  unsigned failures = 0;
  for (bool ending : {false, true}) {
    bool completed = false;
    for (long n = 0; n < 200; ++n) {
      tws::Request r{ending ? tws::Operation::historical_bars
                            : tws::Operation::positions,
                     1,
                     "generation",
                     "U_TEST",
                     "",
                     "private:fixture",
                     ending ? 123U : 0U,
                     ending ? "SMART" : "",
                     ending ? "20260925 16:00:00 UTC" : "",
                     ending ? "1 D" : "",
                     ending ? "1 min" : "",
                     100,
                     65536,
                     1000};
      tws::Collector c;
      check(tws::Collector::start(r, d, c) == source::Status::ok);
      tws::Position pos{"U_TEST", "",    123,     "TEST",
                        "STK",    "USD", "SMART", "0.123456789012345678",
                        1};
      std::string end(50, 'x');
      fail_at = n;
      auto s = ending ? c.end("generation", 1, end, end)
                      : c.position("generation", 1, pos);
      fail_at = -1;
      if (s == source::Status::ok) {
        completed = true;
        break;
      }
      check(s == source::Status::no_memory);
      ++failures;
      check(c.end("generation", 1, end, end) == source::Status::stale);
      tws::Snapshot snapshot;
      check(c.snapshot(snapshot) == source::Status::ok &&
            snapshot.state() == tws::State::failed && snapshot.records() == 0);
    }
    check(completed);
  }
  check(failures > 2);
  std::printf(
      "TWS allocation: %u callback/end failure points remain terminal gaps\n",
      failures);
}
