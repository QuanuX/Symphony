#include <cstdlib>
#include <string>
#include <symphony/scabv/tws.hpp>
using namespace symphony;
void check(bool b) {
  if (!b)
    std::abort();
}
void installed_boundary() {
  struct Driver : scabv::tws::RuntimeDriver {
    source::Status connect(const scabv::tws::Connection &,
                           std::string_view) override {
      return source::Status::ok;
    }
    source::Status pump() override { return source::Status::ok; }
    void disconnect() noexcept override {}
    std::string_view release() const noexcept override { return "10.45"; }
    source::Status start(const scabv::tws::Request &) override {
      return source::Status::ok;
    }
    source::Status cancel(const scabv::tws::Request &) override {
      return source::Status::ok;
    }
  } driver;
  scabv::tws::Request r{scabv::tws::Operation::positions,
                        1,
                        "generation",
                        "U_TEST",
                        "",
                        "private:fixture",
                        0,
                        "",
                        "",
                        "",
                        "",
                        4,
                        4096,
                        1000};
  scabv::tws::Collector c;
  check(scabv::tws::Collector::start(r, driver, c) == source::Status::ok);
  check(c.end("wrong", 1) == source::Status::binding_mismatch);
  check(c.disconnected("generation") == source::Status::ok);
  scabv::tws::Snapshot s;
  check(c.snapshot(s) == source::Status::ok &&
        s.state() == scabv::tws::State::disconnected);
  check(c.end("generation", 1) == source::Status::stale);
  scabv::tws::Session host;
  check(scabv::tws::Session::open({"127.0.0.1", "U_TEST", "", "private:fixture",
                                   "installed", 4002, 9, 1000, 1},
                                  driver, host) == source::Status::ok);
  std::string accounts[]{"U_TEST"};
  check(host.next_valid_id(host.generation(), 100) == source::Status::ok);
  check(host.managed_accounts(host.generation(), accounts) ==
        source::Status::ok);
  r.request_id = 0;
  r.session_generation.clear();
  r.account.clear();
  r.model.clear();
  r.private_scope.clear();
  int request_id = 0;
  check(host.start(r, request_id) == source::Status::ok && request_id == 1);
  check(host.end(host.generation(), request_id) == source::Status::ok);
  check(host.snapshot(s) == source::Status::ok &&
        s.state() == scabv::tws::State::complete);
  host.close();
}
int main() { installed_boundary(); }
