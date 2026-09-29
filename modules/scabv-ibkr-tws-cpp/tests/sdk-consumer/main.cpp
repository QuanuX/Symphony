#include <cstdlib>
#include <string>
#include <symphony/scabv/tws.hpp>
using namespace symphony;
void check(bool b) {
  if (!b)
    std::abort();
}
void installed_boundary() {
  struct Driver : scabv::tws::Driver {
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
}
int main() { installed_boundary(); }
