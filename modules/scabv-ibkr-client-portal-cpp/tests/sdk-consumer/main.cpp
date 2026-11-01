#include <cstdlib>
#include <string>
#include <symphony/scabv/client_portal.hpp>
using namespace symphony;
void check(bool b) {
  if (!b)
    std::abort();
}
void installed_boundary() {
  std::string s = R"([{"id":"U_TEST"}])";
  source::HttpResponse response{200, {s.begin(), s.end()}, true};
  scabv::client_portal::Binding b;
  check(scabv::client_portal::Binding::admit(
            "https://localhost:5000", "U_TEST", "private:fixture", response,
            {4096, 128, 128, 8}, b) == source::Status::ok);
  auto ref = std::string(b.reference());
  check(scabv::client_portal::Binding::admit(
            "https://localhost:5000", "OTHER", "private:fixture", response,
            {4096, 128, 128, 8}, b) == source::Status::not_authorized &&
        b.reference() == ref);
  scabv::client_portal::Plan p;
  check(scabv::client_portal::Plan::positions(b, 10000, p) ==
        source::Status::invalid_argument);
  check(scabv::client_portal::Plan::positions(b, 0, p) == source::Status::ok);
  struct Expired : scabv::client_portal::GatewayTransport {
    source::Status perform(const source::HttpRequest &r,
                           const source::HttpLimits &, std::stop_token,
                           source::HttpResponse &out) override {
      check(r.post && r.endpoint.ends_with("/iserver/auth/status") &&
            r.parameters.empty());
      out = {401, {}, true};
      return source::Status::http_error;
    }
  } transport;
  scabv::client_portal::Session session;
  check(scabv::client_portal::Session::open({"https://localhost:5000",
                                             "U_TEST",
                                             "private:fixture",
                                             {1000, 100, 1024, 4096},
                                             {4096, 128, 128, 8},
                                             1000},
                                            transport, {}, session) ==
        source::Status::not_authorized);
  check(session.state() == scabv::client_portal::SessionState::expired &&
        !session.binding());
}
int main() { installed_boundary(); }
