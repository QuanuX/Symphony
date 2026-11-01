#ifdef SQV_VERIFY_RETENTION
#include "retention.hpp"
#endif
#include <cstdio>
#include <cstdlib>
#include <source_location>
#include <symphony/source/json.hpp>
#include <symphony/sqav/fred.hpp>
using namespace symphony;
namespace f = sqav::fred;
void check(bool x, std::source_location loc = std::source_location::current()) {
  if (!x) {
    std::fprintf(stderr, "FRED assertion line %u\n", loc.line());
    std::abort();
  }
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
void collection_boundaries() {
  const source::JsonLimits json{65536, 4096, 4096, 16};
  const auto first = plan("2021-01-01");
  f::Page a, b;
  check(f::Page::admit(first, fixture("2021-01-01", 4), json, a) ==
        source::Status::ok);
  auto last = fixture("2021-01-01", 4, 2);
  auto value = nlohmann::json::parse(last.body);
  value["observations"][0]["date"] = "2020-03-01";
  value["observations"][1]["date"] = "2020-04-01";
  auto encoded = value.dump();
  last.body.assign(encoded.begin(), encoded.end());
  check(f::Page::admit(plan("2021-01-01", 2), last, json, b) ==
        source::Status::ok);
  f::Collection sequence;
  const f::CollectionLimits limits{2, 4, 1, 65536};
  check(f::Collection::begin(first, limits, sequence) == source::Status::ok);
  check(sequence.accept(b) == source::Status::binding_mismatch &&
        sequence.progress().pages == 0);
  check(sequence.accept(a) == source::Status::ok &&
        sequence.next_plan()->selection().offset == 2 &&
        sequence.progress().pages == 1 && !sequence.progress().complete);
  const auto pending = std::string(sequence.next_plan()->reference());
  check(sequence.accept(a) == source::Status::binding_mismatch);
  f::Page bad;
  check(f::Page::admit(plan("2021-01-01", 2),
                       fixture("2021-01-01", 6, 2), json, bad) == source::Status::ok);
  check(sequence.accept(bad) == source::Status::binding_mismatch);
  check(f::Page::admit(plan("2021-01-01", 2),
                       fixture("2021-01-01", 4, 2), json, bad) == source::Status::ok);
  check(sequence.accept(bad) == source::Status::binding_mismatch &&
        sequence.next_plan()->reference() == pending &&
        sequence.pages().size() == 1);
  // Same-date boundary duplicates are refused even though each page is valid.
  value["observations"][0]["date"] = "2020-02-01";
  encoded = value.dump();
  auto duplicate = last;
  duplicate.body.assign(encoded.begin(), encoded.end());
  check(f::Page::admit(plan("2021-01-01", 2), duplicate, json, bad) == source::Status::ok);
  check(sequence.accept(bad) == source::Status::binding_mismatch);
  check(sequence.accept(b) == source::Status::ok &&
        sequence.progress().complete && !sequence.next_plan() &&
        sequence.progress().records == 4 && sequence.progress().declared_count == 4 &&
        sequence.progress().original_bytes == a.original().size() + b.original().size());
  check(sequence.accept(b) == source::Status::invalid_argument);
  f::Collection replay;
  check(f::Collection::begin(first, limits, replay) == source::Status::ok);
  for (const auto &retained : sequence.pages())
    check(replay.accept(retained) == source::Status::ok);
  check(replay.progress().complete && replay.progress().requests == 0 &&
        replay.pages()[1].reference() == b.reference());
  check(f::Collection::begin(plan("2021-01-01", 2), limits, replay) ==
        source::Status::invalid_argument && replay.progress().complete);
  for (const auto bounded : {f::CollectionLimits{1, 4, 1, 65536},
                            f::CollectionLimits{2, 3, 1, 65536},
                            f::CollectionLimits{2, 4, 1, 1}}) {
    f::Collection limited;
    check(f::Collection::begin(first, bounded, limited) == source::Status::ok &&
          limited.accept(a) == source::Status::limit &&
          limited.progress().pages == 0);
  }
  struct Deny : source::CredentialUse {
    unsigned calls = 0;
    std::string request;
    source::Status with_secret(std::string_view ref, std::uint32_t,
                               std::stop_token, source::SecretSink &) override {
      ++calls;
      check(ref == request);
      return source::Status::not_authorized;
    }
  } deny;
  deny.request = first.reference();
  f::Collection attempted;
  check(f::Collection::begin(first, limits, attempted) == source::Status::ok);
  std::stop_source stopped;
  stopped.request_stop();
  check(attempted.collect_next(deny, {100, 50, 1024, 65536}, json,
                               stopped.get_token()) == source::Status::cancelled &&
        attempted.progress().requests == 0);
  check(attempted.collect_next(deny, {100, 50, 1024, 65536}, json, {}) ==
        source::Status::not_authorized && attempted.progress().requests == 1 &&
        attempted.next_plan()->reference() == first.reference());
  check(attempted.collect_next(deny, {100, 50, 1024, 65536}, json, {}) ==
        source::Status::limit && deny.calls == 1 && attempted.pages().empty());
}
int main() {
  collection_boundaries();
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
