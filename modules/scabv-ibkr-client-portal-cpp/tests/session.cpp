#include <cstdio>
#include <cstdlib>
#include <deque>
#include <source_location>
#include <symphony/scabv/client_portal.hpp>
#include <thread>
using namespace symphony;
namespace cp = scabv::client_portal;
void check(bool x, std::source_location at = std::source_location::current()) {
  if (!x) {
    std::fprintf(stderr, "failure at %u\n", at.line());
    std::abort();
  }
}
source::HttpResponse response(std::string_view body, unsigned code = 200) {
  return {static_cast<std::uint16_t>(code), {body.begin(), body.end()}, true};
}
constexpr auto authenticated =
    R"({"authenticated":true,"connected":true,"competing":false,"established":true})";
constexpr auto accounts = R"([{"id":"U_TEST"}])";
constexpr auto position =
    R"([{"acctId":"U_TEST","conid":123,"position":0.000000000000000001,"mktPrice":12.34567890123456789,"mktValue":1,"avgCost":1}])";
struct Transport final : cp::GatewayTransport {
  struct Reply {
    std::string suffix;
    bool post;
    source::HttpResponse response;
  };
  std::deque<Reply> replies;
  unsigned calls = 0;
  void refresh(std::string_view status = authenticated,
               std::string_view a = accounts) {
    replies.push_back({"/iserver/auth/status", true, response(status)});
    replies.push_back({"/portfolio/accounts", false, response(a)});
  }
  source::Status perform(const source::HttpRequest &r,
                         const source::HttpLimits &h, std::stop_token stop,
                         source::HttpResponse &out) override {
    check(!replies.empty() && !stop.stop_requested());
    const auto &next = replies.front();
    check(r.endpoint.starts_with("https://localhost:5000/v1/api/") &&
          r.endpoint.ends_with(next.suffix));
    check(r.post == next.post &&
          r.credential == source::HttpRequest::Credential::none);
    check(h.timeout_ms > 0 && h.timeout_ms <= 1000 &&
          h.connect_timeout_ms <= h.timeout_ms);
    if (r.post)
      check(r.parameters.empty());
    out = next.response;
    replies.pop_front();
    ++calls;
    return out.http_status == 200 ? source::Status::ok
                                  : source::Status::http_error;
  }
};
cp::SessionConfig config() {
  return {"https://localhost:5000", "U_TEST",
          "private:fixture",        {1000, 100, 1024, 65536},
          {65536, 4096, 4096, 16},  300000};
}
int main() {
  auto c = config();
  Transport transport;
  transport.refresh();
  cp::Session s;
  cp::Page page;
  auto bad = c;
  bad.gateway = "http://localhost:5000";
  check(cp::Session::open(bad, transport, {}, s) ==
            source::Status::invalid_argument &&
        transport.calls == 0);
  check(cp::Session::open(c, transport, {}, s) == source::Status::ok &&
        transport.calls == 2);
  check(s.binding().account() == "U_TEST" &&
        s.state() == cp::SessionState::ready);
  check(s.positions(10000, {}, page) == source::Status::invalid_argument &&
        transport.calls == 2);
  transport.refresh();
  transport.replies.push_back(
      {"/portfolio/U_TEST/positions/0", false, response(position)});
  check(s.positions(0, {}, page) == source::Status::ok &&
        page.next_page() == 1 && page.records() == 1 && transport.calls == 5);
  check(std::string_view(reinterpret_cast<const char *>(page.original().data()),
                         page.original().size()) == position);
  check(s.positions(1, {}, page) == source::Status::limit &&
        transport.calls == 5 && page.records() == 1);
  std::this_thread::sleep_for(std::chrono::milliseconds(1010));
  transport.refresh();
  transport.replies.push_back(
      {"/portfolio/U_TEST/positions/1", false, response("[]")});
  check(s.positions(1, {}, page) == source::Status::ok && !page.next_page() &&
        page.records() == 0);
  s.close();
  check(s.state() == cp::SessionState::closed && !s.binding());
  check(s.positions(0, {}, page) == source::Status::stale &&
        transport.calls == 8);
  Transport history;
  history.refresh();
  cp::Session hs;
  check(cp::Session::open(c, history, {}, hs) == source::Status::ok);
  check(hs.historical_bars({123, "1d", "1min", "bad", false}, 10, {}, page) ==
            source::Status::invalid_argument &&
        history.calls == 2);
  history.refresh();
  history.replies.push_back(
      {"/iserver/marketdata/history", false,
       response(
           R"({"data":[{"t":1000,"o":1,"h":2,"l":0,"c":1.1234567890123456789,"v":10}]})")});
  check(hs.historical_bars({123, "1d", "1min", "20261101-12:00:00", false}, 10,
                           {}, page) == source::Status::ok &&
        page.records() == 1);
  auto preserved =
      std::string(reinterpret_cast<const char *>(page.original().data()),
                  page.original().size());
  // HTTP401, HTTP403 and status expiry never become successful empty pages.
  for (unsigned code : {401U, 403U}) {
    Transport t;
    t.refresh();
    cp::Session session;
    check(cp::Session::open(c, t, {}, session) == source::Status::ok);
    t.replies.push_back({"/iserver/auth/status", true, response("", code)});
    check(session.positions(0, {}, page) == source::Status::not_authorized &&
          t.calls == 3);
    check(session.state() ==
          (code == 401 ? cp::SessionState::expired : cp::SessionState::failed));
    check(session.positions(0, {}, page) == source::Status::stale &&
          t.calls == 3);
    check(
        std::string_view(reinterpret_cast<const char *>(page.original().data()),
                         page.original().size()) == preserved);
  }
  for (
      auto status :
      {R"({"authenticated":false,"connected":true,"competing":false})",
       R"({"authenticated":true,"connected":true,"competing":true})",
       R"({"authenticated":true,"connected":true,"competing":false,"established":false})",
       R"({"authenticated":"true","connected":true,"competing":false})"}) {
    Transport t;
    t.replies.push_back({"/iserver/auth/status", true, response(status)});
    cp::Session session;
    auto result = cp::Session::open(c, t, {}, session);
    check((result == source::Status::not_authorized ||
           result == source::Status::malformed) &&
          t.calls == 1 && !session.binding());
  }
  Transport changed;
  changed.refresh();
  cp::Session changed_session;
  check(cp::Session::open(c, changed, {}, changed_session) ==
        source::Status::ok);
  changed.refresh(authenticated, R"([{"id":"OTHER"}])");
  check(changed_session.positions(0, {}, page) ==
            source::Status::not_authorized &&
        changed.calls == 4 && !changed_session.binding());
  Transport denied;
  denied.refresh();
  cp::Session denied_session;
  check(cp::Session::open(c, denied, {}, denied_session) == source::Status::ok);
  denied.refresh();
  denied.replies.push_back(
      {"/iserver/marketdata/history", false, response("", 403)});
  check(denied_session.historical_bars(
            {123, "1d", "1min", "20261101-12:00:00", false}, 10, {}, page) ==
            source::Status::not_authorized &&
        denied.calls == 5);
  Transport short_lived;
  short_lived.refresh();
  cp::Session expiry;
  c.maximum_age_ms = 20;
  check(cp::Session::open(c, short_lived, {}, expiry) == source::Status::ok);
  std::this_thread::sleep_for(std::chrono::milliseconds(25));
  check(expiry.positions(0, {}, page) == source::Status::stale &&
        expiry.state() == cp::SessionState::expired && short_lived.calls == 2);
  std::stop_source stop;
  stop.request_stop();
  Transport stopped;
  cp::Session stopped_session;
  c = config();
  check(cp::Session::open(c, stopped, stop.get_token(), stopped_session) ==
            source::Status::cancelled &&
        stopped.calls == 0);
  cp::HttpsGatewayTransport https;
  source::HttpResponse unused;
  check(https.perform({"https://localhost:5000/v1/api/portfolio/accounts", "",
                       false, source::HttpRequest::Credential::none},
                      c.http, stop.get_token(),
                      unused) == source::Status::cancelled);
  std::puts(
      "Client Portal host session: exact status POST/data GET, account "
      "refresh, original decimals, pagination, history, expiry, entitlement "
      "refusal, limits and cancellation passed; fixture only");
}
