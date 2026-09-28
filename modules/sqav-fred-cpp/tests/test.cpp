#ifdef SQV_VERIFY_RETENTION
#include "retention.hpp"
#endif
#include <cstdio>
#include <cstdlib>
#include <symphony/source/json.hpp>
#include <symphony/sqav/fred.hpp>
using namespace symphony;
namespace f = sqav::fred;
void check(bool x) {
  if (!x)
    std::abort();
}
source::HttpResponse fixture(std::string_view vintage, std::uint32_t count = 2,
                             std::uint32_t offset = 0) {
  nlohmann::json j = {{"realtime_start", vintage},
                      {"realtime_end", vintage},
                      {"observation_start", "2020-01-01"},
                      {"observation_end", "2020-12-31"},
                      {"units", "lin"},
                      {"output_type", 1},
                      {"file_type", "json"},
                      {"order_by", "observation_date"},
                      {"sort_order", "asc"},
                      {"count", count},
                      {"offset", offset},
                      {"limit", 2}};
  j["observations"] = nlohmann::json::array({{{"realtime_start", vintage},
                                              {"realtime_end", vintage},
                                              {"date", "2020-01-01"},
                                              {"value", "1.00000000000000001"}},
                                             {{"realtime_start", vintage},
                                              {"realtime_end", vintage},
                                              {"date", "2020-02-01"},
                                              {"value", "."}}});
  auto s = j.dump();
  return {200, {s.begin(), s.end()}, true};
}
f::Plan plan(std::string_view date, std::uint32_t offset = 0) {
  f::Plan p;
  check(f::Plan::create({f::Operation::observations, "SERIES", "2020-01-01",
                         "2020-12-31", std::string(date), std::string(date), 2,
                         offset},
                        p) == source::Status::ok);
  return p;
}
int main() {
  source::JsonLimits limits{65536, 4096, 4096, 16};
  auto first = plan("2021-01-01"), later = plan("2022-01-01");
  f::Page a, b;
  auto one = fixture("2021-01-01"), two = fixture("2022-01-01");
  check(f::Page::admit(first, one, limits, a) == source::Status::ok);
  check(f::Page::admit(later, two, limits, b) == source::Status::ok);
  check(a.reference() != b.reference() &&
        first.reference() != later.reference());
  check(a.observations()[0].value == "1.00000000000000001" &&
        a.observations()[1].missing && !a.observations()[0].missing &&
        !a.next_offset());
  auto saved = std::string(a.reference());
  check(f::Page::admit(first, two, limits, a) ==
            source::Status::binding_mismatch &&
        a.reference() == saved);
  auto paged = fixture("2021-01-01", 4);
  check(f::Page::admit(first, paged, limits, b) == source::Status::ok &&
        b.next_offset() == 2);
  auto last = fixture("2021-01-01", 4, 2);
  check(f::Page::admit(plan("2021-01-01", 2), last, limits, b) ==
            source::Status::ok &&
        !b.next_offset());
  for (std::size_t i = 0; i < one.body.size(); ++i) {
    auto bad = one;
    bad.body.resize(i);
    check(f::Page::admit(first, bad, limits, a) != source::Status::ok &&
          a.reference() == saved);
  }
  auto bad = one;
  bad.complete = false;
  check(f::Page::admit(first, bad, limits, a) ==
        source::Status::transport_error);
  sqav::Capture c;
  check(a.capture("attempt", "fixture", "private:test",
                  {sqav::TimeRole::acquisition, "2026-09-28", "iso-date",
                   "UTC-calendar", "day", "fixture"},
                  {131072, 16384, 4096}, c) == source::Status::ok);
  check(c.description().coverage == sqav::Coverage::complete &&
        std::ranges::equal(c.original(), one.body));
  f::Plan v;
  check(f::Plan::create({f::Operation::vintage_dates, "SERIES", "", "",
                         "2020-01-01", "2022-12-31", 2, 0},
                        v) == source::Status::ok);
  std::string raw =
      R"({"realtime_start":"2020-01-01","realtime_end":"2022-12-31","order_by":"vintage_date","sort_order":"asc","count":2,"offset":0,"limit":2,"vintage_dates":["2021-01-01","2022-01-01"]})";
  source::HttpResponse vr{200, {raw.begin(), raw.end()}, true};
  check(f::Page::admit(v, vr, limits, b) == source::Status::ok &&
        b.vintage_dates().size() == 2);
  struct Deny : source::CredentialUse {
    source::Status with_secret(std::string_view, std::uint32_t, std::stop_token,
                               source::SecretSink &) override {
      return source::Status::not_authorized;
    }
  } deny;
  check(f::collect(first, deny, {100, 50, 1024, 65536}, limits, {}, a) ==
            source::Status::not_authorized &&
        a.reference() == saved);
#ifdef SQV_VERIFY_RETENTION
  source_composition::verify(c);
#endif
  std::puts("FRED: exact vintages, decimal/missing values, pagination, "
            "truncations, capture and credential refusal passed");
}
