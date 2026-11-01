#include <chrono>
#include <symphony/scabv/client_portal.hpp>
#include <symphony/source/json.hpp>
#include <unistd.h>
namespace symphony::scabv::client_portal {
namespace {
const Binding empty_session_binding;
}
namespace detail {
struct Session {
  using Clock = std::chrono::steady_clock;
  SessionConfig config;
  GatewayTransport *transport = nullptr;
  Binding binding;
  SessionState state = SessionState::ready;
  pid_t pid = ::getpid();
  Clock::time_point deadline, next_read;
  bool local() const noexcept { return pid == ::getpid(); }
  Status admit(std::stop_token stop) noexcept {
    if (!local() || state != SessionState::ready)
      return Status::stale;
    if (stop.stop_requested())
      return Status::cancelled;
    if (Clock::now() >= deadline) {
      state = SessionState::expired;
      return Status::stale;
    }
    return Status::ok;
  }
  Status exchange(const source::HttpRequest &r, std::stop_token stop,
                  source::HttpResponse &response) {
    auto result = admit(stop);
    if (result != Status::ok)
      return result;
    auto limits = config.http;
    auto remaining = std::chrono::duration_cast<std::chrono::milliseconds>(
                         deadline - Clock::now())
                         .count();
    if (remaining <= 0) {
      state = SessionState::expired;
      return Status::stale;
    }
    limits.timeout_ms = static_cast<std::uint32_t>(
        std::min<std::int64_t>(remaining, limits.timeout_ms));
    limits.connect_timeout_ms =
        std::min(limits.connect_timeout_ms, limits.timeout_ms);
    result = transport->perform(r, limits, stop, response);
    if (response.complete &&
        (result == Status::ok || result == Status::http_error) &&
        response.http_status == 401) {
      state = SessionState::expired;
      return Status::not_authorized;
    }
    if (response.complete &&
        (result == Status::ok || result == Status::http_error) &&
        response.http_status == 403) {
      state = SessionState::failed;
      return Status::not_authorized;
    }
    if (result != Status::ok) {
      state = SessionState::failed;
      return result;
    }
    result = admit(stop);
    if (result != Status::ok)
      return result;
    if (!response.complete) {
      state = SessionState::failed;
      return Status::transport_error;
    }
    if (response.http_status != 200) {
      state = SessionState::failed;
      return Status::http_error;
    }
    return Status::ok;
  }
  Status refresh(std::stop_token stop) {
    source::HttpResponse response;
    // IBKR's current reference specifies POST for this read-only status route.
    source::HttpRequest status{config.gateway + "/v1/api/iserver/auth/status",
                               "", true, source::HttpRequest::Credential::none};
    auto result = exchange(status, stop, response);
    if (result != Status::ok)
      return result;
    nlohmann::json j;
    result = source::parse_json(response.body, config.json, j);
    if (result != Status::ok) {
      state = SessionState::failed;
      return result;
    }
    if (!j.is_object()) {
      state = SessionState::failed;
      return Status::malformed;
    }
    for (const char *key : {"authenticated", "connected", "competing"})
      if (!j.contains(key) || !j.at(key).is_boolean()) {
        state = SessionState::failed;
        return Status::malformed;
      }
    if (j.contains("established") && !j.at("established").is_boolean()) {
      state = SessionState::failed;
      return Status::malformed;
    }
    if (!j.at("authenticated").get<bool>() || !j.at("connected").get<bool>() ||
        j.at("competing").get<bool>() ||
        (j.contains("established") && !j.at("established").get<bool>())) {
      state = SessionState::expired;
      return Status::not_authorized;
    }
    source::HttpRequest accounts{config.gateway + "/v1/api/portfolio/accounts",
                                 "", false,
                                 source::HttpRequest::Credential::none};
    result = exchange(accounts, stop, response);
    if (result != Status::ok)
      return result;
    Binding updated;
    result =
        Binding::admit(config.gateway, config.account, config.private_scope,
                       response, config.json, updated);
    if (result != Status::ok) {
      state = SessionState::failed;
      return result;
    }
    binding = std::move(updated);
    return Status::ok;
  }
  Status collect(const Plan &plan, std::stop_token stop, Page &out) {
    source::HttpResponse response;
    auto result = exchange(plan.request(), stop, response);
    if (result == Status::ok)
      result = Page::admit(plan, response, config.json, out);
    if (result != Status::ok && state == SessionState::ready)
      state = SessionState::failed;
    return result;
  }
};
} // namespace detail
Status HttpsGatewayTransport::perform(const source::HttpRequest &r,
                                      const source::HttpLimits &h,
                                      std::stop_token stop,
                                      source::HttpResponse &response) {
  return source::fetch(r, nullptr, h, stop, response);
}
Session::Session() noexcept = default;
Session::~Session() noexcept = default;
Status Session::open(const SessionConfig &c, GatewayTransport &transport,
                     std::stop_token stop, Session &out) noexcept {
  if (out.state_ || c.maximum_age_ms == 0 || c.maximum_age_ms > 300000 ||
      !source::valid_limits(c.json) || c.http.timeout_ms == 0 ||
      c.http.timeout_ms > 300000 || c.http.connect_timeout_ms == 0 ||
      c.http.connect_timeout_ms > c.http.timeout_ms ||
      c.http.max_header_bytes == 0 || c.http.max_header_bytes > 1048576 ||
      c.http.max_body_bytes == 0 || c.http.max_body_bytes > c.json.bytes)
    return Status::invalid_argument;
  if (stop.stop_requested())
    return Status::cancelled;
  try {
    // Reuse pure admission for endpoint/account/scope validation before I/O.
    source::HttpResponse empty{200, {'[', ']'}, true};
    Binding checked;
    auto result = Binding::admit(c.gateway, c.account, c.private_scope, empty,
                                 c.json, checked);
    if (result != Status::not_authorized)
      return result;
    auto s = std::make_unique<detail::Session>();
    s->config = c;
    s->transport = &transport;
    s->deadline = detail::Session::Clock::now() +
                  std::chrono::milliseconds(c.maximum_age_ms);
    out.state_ = std::move(s);
    return out.state_->refresh(stop);
  } catch (const std::bad_alloc &) {
    if (out.state_)
      out.state_->state = SessionState::failed;
    return Status::no_memory;
  } catch (...) {
    if (out.state_)
      out.state_->state = SessionState::failed;
    return Status::transport_error;
  }
}
SessionState Session::state() const noexcept {
  if (!state_ || !state_->local())
    return SessionState::closed;
  if (state_->state == SessionState::ready &&
      detail::Session::Clock::now() >= state_->deadline)
    return SessionState::expired;
  return state_->state;
}
const Binding &Session::binding() const noexcept {
  return state() == SessionState::ready ? state_->binding
                                        : empty_session_binding;
}
Status Session::positions(std::uint32_t page, std::stop_token stop,
                          Page &out) noexcept {
  if (!state_)
    return Status::stale;
  auto &s = *state_;
  auto result = s.admit(stop);
  if (result != Status::ok)
    return result;
  if (page >= 10000)
    return Status::invalid_argument;
  if (detail::Session::Clock::now() < s.next_read)
    return Status::limit;
  s.next_read = detail::Session::Clock::now() + std::chrono::seconds(1);
  try {
    result = s.refresh(stop);
    if (result != Status::ok)
      return result;
    Plan plan;
    result = Plan::positions(s.binding, page, plan);
    return result == Status::ok ? s.collect(plan, stop, out) : result;
  } catch (const std::bad_alloc &) {
    s.state = SessionState::failed;
    return Status::no_memory;
  } catch (...) {
    s.state = SessionState::failed;
    return Status::transport_error;
  }
}
Status Session::historical_bars(const History &h, std::uint32_t limit,
                                std::stop_token stop, Page &out) noexcept {
  if (!state_)
    return Status::stale;
  auto &s = *state_;
  auto result = s.admit(stop);
  if (result != Status::ok)
    return result;
  try {
    Plan plan;
    result = Plan::historical_bars(s.binding, h, limit, plan);
    if (result != Status::ok)
      return result;
    if (detail::Session::Clock::now() < s.next_read)
      return Status::limit;
    s.next_read = detail::Session::Clock::now() + std::chrono::seconds(1);
    result = s.refresh(stop);
    if (result != Status::ok)
      return result;
    result = Plan::historical_bars(s.binding, h, limit, plan);
    return result == Status::ok ? s.collect(plan, stop, out) : result;
  } catch (const std::bad_alloc &) {
    s.state = SessionState::failed;
    return Status::no_memory;
  } catch (...) {
    s.state = SessionState::failed;
    return Status::transport_error;
  }
}
void Session::close() noexcept {
  if (state_ && state_->local()) {
    state_->state = SessionState::closed;
    state_->binding = {};
  }
}
} // namespace symphony::scabv::client_portal
