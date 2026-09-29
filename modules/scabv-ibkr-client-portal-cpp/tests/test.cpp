#ifdef SQV_VERIFY_RETENTION
#include "retention.hpp"
#endif
#include <cstdio>
#include <cstdlib>
#include <symphony/scabv/client_portal.hpp>
#include <symphony/source/json.hpp>
using namespace symphony;
namespace cp = scabv::client_portal;
void check(bool x) {
  if (!x)
    std::abort();
}
source::HttpResponse response(std::string_view s) {
  return {200, {s.begin(), s.end()}, true};
}
int main() {
  source::JsonLimits l{65536, 4096, 4096, 16};
  cp::Binding binding;
  auto accounts = response(R"([{"id":"U_TEST"}])");
  check(cp::Binding::admit("https://localhost:5000", "U_TEST",
                           "private:fixture", accounts, l,
                           binding) == source::Status::ok);
  auto ref = std::string(binding.reference());
  check(cp::Binding::admit("https://localhost:5000", "OTHER", "private:fixture",
                           accounts, l,
                           binding) == source::Status::not_authorized &&
        binding.reference() == ref);
  check(cp::Binding::admit("http://localhost:5000", "U_TEST", "private:fixture",
                           accounts, l,
                           binding) == source::Status::invalid_argument);
  check(cp::Binding::admit("https://localhost:5000", "U_TEST", "public",
                           accounts, l,
                           binding) == source::Status::invalid_argument);
  cp::Plan positions;
  check(
      cp::Plan::positions(binding, 0, positions) == source::Status::ok &&
      !positions.request().post &&
      positions.request().endpoint.ends_with("/portfolio/U_TEST/positions/0"));
  std::string raw =
      R"([{"acctId":"U_TEST","conid":123,"position":0.000000000000000001,"mktPrice":12.34567890123456789,"mktValue":1,"avgCost":1}])";
  cp::Page page;
  check(cp::Page::admit(positions, response(raw), l, page) ==
            source::Status::ok &&
        page.records() == 1 && page.next_page() == 1);
  check(std::string_view(reinterpret_cast<const char *>(page.original().data()),
                         page.original().size()) == raw);
  auto mixed = raw;
  auto at = mixed.find("U_TEST");
  mixed.replace(at, 6, "OTHER_");
  check(cp::Page::admit(positions, response(mixed), l, page) ==
            source::Status::binding_mismatch &&
        page.records() == 1);
  cp::Page empty;
  check(cp::Page::admit(positions, response("[]"), l, empty) ==
            source::Status::ok &&
        !empty.next_page());
  cp::Plan history;
  check(cp::Plan::historical_bars(
            binding, {123, "1d", "1min", "20260925-16:00:00", false}, 100,
            history) == source::Status::ok &&
        history.request().parameters.find("outsideRth=false") !=
            std::string::npos);
  check(cp::Page::admit(
            history,
            response(R"({"data":[{"t":1000,"o":1,"h":2,"l":0,"c":1,"v":10}]})"),
            l, empty) == source::Status::ok &&
        empty.records() == 1);
  check(
      cp::Page::admit(
          history,
          response(
              R"({"data":[{"t":1000,"o":1,"h":2,"l":0,"c":1,"v":10},{"t":1000,"o":1,"h":2,"l":0,"c":1,"v":10}]})"),
          l, empty) == source::Status::malformed);
  sqav::Capture capture;
  check(page.capture("attempt", "fixture",
                     {sqav::TimeRole::acquisition, "2026-09-28", "iso-date",
                      "UTC", "day", "fixture"},
                     {131072, 16384, 4096}, capture) == source::Status::ok &&
        capture.description().source.access_scope == "private:fixture" &&
        capture.description().coverage == sqav::Coverage::partial);
  std::stop_source stop;
  stop.request_stop();
  check(cp::collect(positions, {100, 50, 1024, 65536}, l, stop.get_token(),
                    page) == source::Status::cancelled &&
        page.records() == 1);
#ifdef SQV_VERIFY_RETENTION
  source_composition::verify(capture);
#endif
  std::puts("Client Portal: exact private account, separate read routes, "
            "original decimals, pagination, bars and cancellation passed");
}
