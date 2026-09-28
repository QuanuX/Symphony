#include <cstdlib>
#include <string>
#include <symphony/sqav/databento/reference.hpp>
using namespace symphony;
void check(bool b) {
  if (!b)
    std::abort();
}
void installed_boundary() {
  namespace ref = sqav::databento::reference;
  ref::Plan p;
  check(ref::Plan::create({ref::Operation::security_master_range,
                           {"AAPL"},
                           "2026-09-24",
                           "2026-09-25"},
                          p) == source::Status::ok);
  auto before = std::string(p.reference());
  check(ref::Plan::create({ref::Operation::security_master_range,
                           {"AAPL"},
                           "2026-09-25",
                           "2026-09-24"},
                          p) == source::Status::invalid_argument &&
        p.reference() == before);
  ref::Page page;
  source::HttpResponse response{200, {1, 2, 3}, true};
  check(ref::Page::admit(p, response,
                         {{4096, 128, 1024, 8}, 4096, 4096, 10, 20},
                         page) != source::Status::ok &&
        !page);
}
int main() { installed_boundary(); }
