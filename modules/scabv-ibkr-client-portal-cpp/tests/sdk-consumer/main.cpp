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
}
int main() { installed_boundary(); }
