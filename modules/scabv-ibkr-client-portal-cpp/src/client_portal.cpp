#include <algorithm>
#include <charconv>
#include <set>
#include <symphony/scabv/client_portal.hpp>
#include <symphony/source/json.hpp>
namespace symphony::scabv::client_portal {
namespace detail {
struct BindingState {
  std::string gateway, account, scope, reference;
};
struct PlanState {
  Binding binding;
  source::HttpRequest request;
  std::string reference;
  Operation operation;
  std::uint32_t page = 0, limit = 0;
};
struct PageState {
  Plan plan;
  std::vector<std::uint8_t> bytes;
  std::string reference;
  std::uint32_t records = 0;
  std::optional<std::uint32_t> next;
};
} // namespace detail
namespace {
const Binding empty_binding{};
const source::HttpRequest empty_request{};
bool gateway_ok(std::string_view g) {
  constexpr std::string_view a = "https://localhost:", b = "https://127.0.0.1:";
  auto prefix = g.starts_with(a)   ? a
                : g.starts_with(b) ? b
                                   : std::string_view{};
  if (prefix.empty())
    return false;
  g.remove_prefix(prefix.size());
  unsigned port = 0;
  auto [end, ec] = std::from_chars(g.data(), g.data() + g.size(), port);
  return ec == std::errc{} && end == g.data() + g.size() && port > 0 &&
         port <= 65535;
}
bool interval(std::string_view s, bool bar) {
  auto p = s.find_first_not_of("0123456789");
  if (p == 0 || p == s.npos || p > 5)
    return false;
  unsigned n = 0;
  auto [end, ec] = std::from_chars(s.data(), s.data() + p, n);
  if (ec != std::errc{} || end != s.data() + p || n == 0)
    return false;
  auto unit = s.substr(p);
  return unit == "min" || unit == "h" || unit == "d" || unit == "w" ||
         unit == "m" || (bar ? unit == "S" : unit == "y");
}
bool time(std::string_view s) {
  if (s.size() != 17 || s[8] != '-' || s[11] != ':' || s[14] != ':')
    return false;
  std::string d = std::string(s.substr(0, 4)) + "-" +
                  std::string(s.substr(4, 2)) + "-" +
                  std::string(s.substr(6, 2));
  if (!source::date(d))
    return false;
  for (auto p : {9U, 12U, 15U}) {
    if (s[p] < '0' || s[p] > '9' || s[p + 1] < '0' || s[p + 1] > '9')
      return false;
    unsigned n = unsigned(s[p] - '0') * 10 + unsigned(s[p + 1] - '0');
    if (n > (p == 9 ? 23U : 59U))
      return false;
  }
  return true;
}
} // namespace
Status Binding::admit(std::string_view gateway, std::string_view account,
                      std::string_view scope, const source::HttpResponse &r,
                      const source::JsonLimits &limits, Binding &out) noexcept {
  if (!gateway_ok(gateway) || !source::token(account, 64) ||
      !scope.starts_with("private:") || !source::token(scope, 256))
    return Status::invalid_argument;
  if (!r.complete)
    return Status::transport_error;
  if (r.http_status != 200)
    return Status::http_error;
  nlohmann::json j;
  auto s = source::parse_json(r.body, limits, j);
  if (s != Status::ok)
    return s;
  try {
    if (!j.is_array() || j.size() > 1024)
      return Status::malformed;
    bool found = false;
    std::set<std::string> accounts;
    for (const auto &entry : j) {
      auto id = entry.at("id").get<std::string>();
      if (!source::token(id, 64) || !accounts.insert(id).second)
        return Status::malformed;
      if (id == account)
        found = true;
    }
    if (!found)
      return Status::not_authorized;
    auto state = std::make_shared<detail::BindingState>();
    state->gateway = gateway;
    state->account = account;
    state->scope = scope;
    state->reference =
        "ibkr-cp-binding-v1-" +
        source::digest(
            std::string(gateway) + "\n" + std::string(account) + "\n" +
            std::string(scope) + "\n" +
            std::string(reinterpret_cast<const char *>(r.body.data()),
                        r.body.size()));
    out.state_ = std::move(state);
    return Status::ok;
  } catch (const std::bad_alloc &) {
    return Status::no_memory;
  } catch (const nlohmann::json::exception &) {
    return Status::malformed;
  } catch (...) {
    return Status::internal_error;
  }
}
std::string_view Binding::account() const noexcept {
  return state_ ? state_->account : std::string_view{};
}
std::string_view Binding::gateway() const noexcept {
  return state_ ? state_->gateway : std::string_view{};
}
std::string_view Binding::access_scope() const noexcept {
  return state_ ? state_->scope : std::string_view{};
}
std::string_view Binding::reference() const noexcept {
  return state_ ? state_->reference : std::string_view{};
}
Status Plan::positions(const Binding &b, std::uint32_t page,
                       Plan &out) noexcept {
  if (!b || page >= 10000)
    return Status::invalid_argument;
  try {
    auto s = std::make_shared<detail::PlanState>();
    s->binding = b;
    s->operation = Operation::positions;
    s->page = page;
    s->limit = 100;
    s->request.endpoint = std::string(b.gateway()) + "/v1/api/portfolio/" +
                          source::encode(b.account()) + "/positions/" +
                          std::to_string(page);
    s->reference =
        "ibkr-cp-plan-v1-" + source::digest(std::string(b.reference()) + "\n" +
                                            s->request.reference());
    out.state_ = std::move(s);
    return Status::ok;
  } catch (const std::bad_alloc &) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}
Status Plan::historical_bars(const Binding &b, const History &h,
                             std::uint32_t limit, Plan &out) noexcept {
  if (!b || h.conid == 0 || h.conid > 2147483647 ||
      !interval(h.period, false) || !interval(h.bar, true) ||
      !time(h.start_time) || limit == 0 || limit > 1000)
    return Status::invalid_argument;
  try {
    auto s = std::make_shared<detail::PlanState>();
    s->binding = b;
    s->operation = Operation::historical_bars;
    s->limit = limit;
    s->request.endpoint =
        std::string(b.gateway()) + "/v1/api/iserver/marketdata/history";
    s->request.parameters = "conid=" + std::to_string(h.conid) +
                            "&period=" + source::encode(h.period) +
                            "&bar=" + source::encode(h.bar) +
                            "&startTime=" + source::encode(h.start_time) +
                            "&direction=-1&source=Last&outsideRth=" +
                            (h.outside_regular_hours ? "true" : "false");
    s->reference =
        "ibkr-cp-plan-v1-" +
        source::digest(std::string(b.reference()) + "\n" +
                       s->request.reference() + "\n" + std::to_string(limit));
    out.state_ = std::move(s);
    return Status::ok;
  } catch (const std::bad_alloc &) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}
const source::HttpRequest &Plan::request() const noexcept {
  return state_ ? state_->request : empty_request;
}
const Binding &Plan::binding() const noexcept {
  return state_ ? state_->binding : empty_binding;
}
Operation Plan::operation() const noexcept {
  return state_ ? state_->operation : Operation::positions;
}
std::uint32_t Plan::page() const noexcept { return state_ ? state_->page : 0; }
std::uint32_t Plan::record_limit() const noexcept {
  return state_ ? state_->limit : 0;
}
std::string_view Plan::reference() const noexcept {
  return state_ ? state_->reference : std::string_view{};
}
Status Page::admit(const Plan &p, const source::HttpResponse &r,
                   const source::JsonLimits &l, Page &out) noexcept {
  if (!p)
    return Status::invalid_argument;
  if (!r.complete)
    return Status::transport_error;
  if (r.http_status != 200)
    return Status::http_error;
  nlohmann::json j;
  auto status = source::parse_json(r.body, l, j);
  if (status != Status::ok)
    return status;
  try {
    auto s = std::make_shared<detail::PageState>();
    s->plan = p;
    if (p.operation() == Operation::positions) {
      if (!j.is_array() || j.size() > 100)
        return Status::malformed;
      std::set<std::uint32_t> seen;
      for (const auto &row : j) {
        if (row.at("acctId") != p.binding().account())
          return Status::binding_mismatch;
        const auto &c = row.at("conid");
        if (!c.is_number_unsigned() || c.get<std::uint64_t>() == 0 ||
            c.get<std::uint64_t>() > 2147483647 ||
            !seen.insert(c.get<std::uint32_t>()).second)
          return Status::malformed;
        for (auto field : {"position", "mktPrice", "mktValue", "avgCost"})
          if (!row.at(field).is_number())
            return Status::malformed;
      }
      s->records = static_cast<std::uint32_t>(j.size());
      // An empty page is explicit exhaustion. A shorter nonempty page does not
      // prove a stable account snapshot across separate provider requests.
      if (!j.empty()) {
        if (p.page() == 9999)
          return Status::limit;
        s->next = p.page() + 1;
      }
    } else {
      const auto &rows = j.at("data");
      if (!rows.is_array() || rows.size() > p.record_limit())
        return Status::limit;
      std::uint64_t previous = 0;
      bool has = false;
      for (const auto &row : rows) {
        const auto &t = row.at("t");
        if (!t.is_number_unsigned())
          return Status::malformed;
        auto timestamp = t.get<std::uint64_t>();
        if (has && timestamp <= previous)
          return Status::malformed;
        has = true;
        previous = timestamp;
        for (auto field : {"o", "h", "l", "c", "v"})
          if (!row.at(field).is_number())
            return Status::malformed;
      }
      s->records = static_cast<std::uint32_t>(rows.size());
    }
    s->bytes = r.body;
    s->reference =
        "ibkr-cp-page-v1-" +
        source::digest(
            std::string(p.reference()) + "\n" +
            std::string(reinterpret_cast<const char *>(r.body.data()),
                        r.body.size()));
    out.state_ = std::move(s);
    return Status::ok;
  } catch (const std::bad_alloc &) {
    return Status::no_memory;
  } catch (const nlohmann::json::exception &) {
    return Status::malformed;
  } catch (...) {
    return Status::internal_error;
  }
}
source::Bytes Page::original() const noexcept {
  return state_ ? source::Bytes(state_->bytes) : source::Bytes{};
}
std::uint32_t Page::records() const noexcept {
  return state_ ? state_->records : 0;
}
std::optional<std::uint32_t> Page::next_page() const noexcept {
  return state_ ? state_->next : std::nullopt;
}
Status Page::capture(std::string_view attempt, std::string_view attribution,
                     const sqav::TimeEvidence &acq, const sqav::Limits &limits,
                     sqav::Capture &out) const noexcept {
  if (!state_ || acq.role != sqav::TimeRole::acquisition)
    return Status::invalid_argument;
  try {
    const auto &p = state_->plan;
    const auto &b = p.binding();
    auto operation = p.operation() == Operation::positions
                         ? "portfolio.positions"
                         : "iserver.marketdata.history";
    sqav::Description d{{"ibkr", "client-portal-v1", "observed-2026-09-28",
                         operation, "scabv-ibkr-client-portal-cpp", "0.1.0-dev",
                         std::string(b.account()), std::string(b.reference()),
                         std::string(p.reference()),
                         "ibkr-cp-json-" + std::string(operation), "utf8-json",
                         std::string(b.access_scope())},
                        std::string(attempt),
                        std::string(attribution),
                        "page=" + std::to_string(p.page()),
                        sqav::Coverage::partial,
                        "one-request;no-atomic-account-snapshot",
                        state_->reference,
                        state_->records,
                        {acq}};
    auto s = sqav::Capture::create(d, state_->bytes, limits, out);
    return s == sqav::Status::ok          ? Status::ok
           : s == sqav::Status::limit     ? Status::limit
           : s == sqav::Status::no_memory ? Status::no_memory
                                          : Status::invalid_argument;
  } catch (const std::bad_alloc &) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}
Status connect(std::string_view gateway, std::string_view account,
               std::string_view scope, const source::HttpLimits &h,
               const source::JsonLimits &j, std::stop_token stop,
               Binding &out) noexcept {
  if (!gateway_ok(gateway) || !source::token(account, 64) ||
      !scope.starts_with("private:") || !source::token(scope, 256) ||
      !source::valid_limits(j) || h.max_body_bytes > j.bytes)
    return Status::invalid_argument;
  try {
    source::HttpRequest r{std::string(gateway) + "/v1/api/portfolio/accounts",
                          "", false, source::HttpRequest::Credential::none};
    source::HttpResponse response;
    auto s = source::fetch(r, nullptr, h, stop, response);
    return s == Status::ok
               ? Binding::admit(gateway, account, scope, response, j, out)
               : s;
  } catch (const std::bad_alloc &) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}
Status collect(const Plan &p, const source::HttpLimits &h,
               const source::JsonLimits &j, std::stop_token stop,
               Page &out) noexcept {
  if (!p || !source::valid_limits(j) || h.max_body_bytes > j.bytes)
    return Status::invalid_argument;
  source::HttpResponse response;
  auto s = source::fetch(p.request(), nullptr, h, stop, response);
  return s == Status::ok ? Page::admit(p, response, j, out) : s;
}
} // namespace symphony::scabv::client_portal
