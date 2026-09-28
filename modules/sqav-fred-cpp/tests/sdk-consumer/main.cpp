#include <cstdlib>
#include <string>
#include <symphony/sqav/fred.hpp>
using namespace symphony;
void check(bool b) {
  if (!b)
    std::abort();
}
void installed_boundary() {
  sqav::fred::Plan p;
  check(sqav::fred::Plan::create({sqav::fred::Operation::vintage_dates, "GDP",
                                  "", "", "2020-01-01", "2022-01-01", 2, 0},
                                 p) == source::Status::ok);
  std::string s =
      R"({"realtime_start":"2020-01-01","realtime_end":"2022-01-01","order_by":"vintage_date","sort_order":"asc","count":1,"offset":0,"limit":2,"vintage_dates":["2021-01-01"]})";
  source::HttpResponse response{200, {s.begin(), s.end()}, true};
  sqav::fred::Page page;
  check(sqav::fred::Page::admit(p, response, {4096, 128, 128, 8}, page) ==
            source::Status::ok &&
        page.vintage_dates().size() == 1);
  auto before = std::string(page.reference());
  response.complete = false;
  check(sqav::fred::Page::admit(p, response, {4096, 128, 128, 8}, page) ==
            source::Status::transport_error &&
        page.reference() == before);
}
int main() { installed_boundary(); }
