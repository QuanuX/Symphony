#include <chrono>
#include <limits>
#include <set>
#include <symphony/scabv/tws.hpp>
#include <unistd.h>
namespace symphony::scabv::tws {
namespace detail {
struct Session {
  using Clock = std::chrono::steady_clock;
  Connection connection;
  RuntimeDriver *driver = nullptr;
  pid_t pid = ::getpid();
  SessionState state = SessionState::connecting;
  std::uint64_t epoch = 0, next_id = 1;
  std::string generation;
  bool handshake = false, account = false, active = false, connected = false;
  Clock::time_point deadline{}, next_start{}, next_history{};
  Collector collector;
  ~Session() { close(); }
  bool local() const noexcept { return pid == ::getpid(); }
  void close() noexcept {
    if (!local())
      return;
    if (active)
      (void)collector.disconnected(generation);
    if (connected)
      driver->disconnect();
    connected = false;
    state = SessionState::disconnected;
  }
  Status connect() {
    if (epoch == std::numeric_limits<std::uint64_t>::max())
      return Status::limit;
    auto next = connection.session_tag + "." + std::to_string(epoch + 1);
    generation = std::move(next);
    ++epoch;
    handshake = account = false;
    state = SessionState::connecting;
    deadline = Clock::now() +
               std::chrono::milliseconds(connection.handshake_timeout_ms);
    connected = true; // Includes uncertain partial connection failures.
    auto result = driver->connect(connection, generation);
    if (result != Status::ok) {
      close();
      state = SessionState::failed;
    }
    return result;
  }
  Status admission(std::string_view g) noexcept {
    if (!local())
      return Status::stale;
    if (g != generation)
      return Status::binding_mismatch;
    if (state != SessionState::connecting && state != SessionState::ready)
      return Status::stale;
    if (state == SessionState::connecting && Clock::now() >= deadline) {
      close();
      return Status::timeout;
    }
    return Status::ok;
  }
  void ready() noexcept {
    if (handshake && account)
      state = SessionState::ready;
  }
};
} // namespace detail
Session::Session() noexcept = default;
Session::~Session() noexcept = default;
Status Session::open(const Connection &c, RuntimeDriver &driver,
                     Session &out) noexcept {
  if (out.state_ || (c.host != "localhost" && c.host != "127.0.0.1") ||
      c.port == 0 || c.client_id < 0 || c.first_request_id <= 0 ||
      c.handshake_timeout_ms == 0 || c.handshake_timeout_ms > 300000 ||
      !source::token(c.account, 64) ||
      (!c.model.empty() && !source::token(c.model)) ||
      !source::token(c.session_tag, 64) ||
      !c.private_scope.starts_with("private:") ||
      !source::token(c.private_scope, 256) || driver.release() != sdk_release)
    return Status::invalid_argument;
  try {
    auto s = std::make_unique<detail::Session>();
    s->connection = c;
    s->driver = &driver;
    s->next_id = static_cast<std::uint64_t>(c.first_request_id);
    out.state_ = std::move(s);
    return out.state_->connect();
  } catch (const std::bad_alloc &) {
    if (out.state_) {
      out.state_->close();
      out.state_->state = SessionState::failed;
    }
    return Status::no_memory;
  } catch (...) {
    if (out.state_) {
      out.state_->close();
      out.state_->state = SessionState::failed;
    }
    return Status::transport_error;
  }
}
std::string_view Session::generation() const noexcept {
  return state_ && state_->local() ? state_->generation : std::string_view{};
}
SessionState Session::state() const noexcept {
  return state_ && state_->local() ? state_->state : SessionState::disconnected;
}
Status Session::next_valid_id(std::string_view g, std::int32_t id) noexcept {
  if (!state_)
    return Status::invalid_argument;
  auto status = state_->admission(g);
  if (status != Status::ok)
    return status;
  if (id < 0)
    return Status::invalid_argument;
  // nextValidId is an order ID and readiness signal, never a read ID allocator.
  state_->handshake = true;
  state_->ready();
  return Status::ok;
}
Status
Session::managed_accounts(std::string_view g,
                          std::span<const std::string> accounts) noexcept {
  if (!state_)
    return Status::invalid_argument;
  auto status = state_->admission(g);
  if (status != Status::ok)
    return status;
  try {
    std::set<std::string_view> seen;
    bool found = false;
    if (accounts.empty() || accounts.size() > 1024) {
      state_->close();
      state_->state = SessionState::failed;
      return Status::malformed;
    }
    for (const auto &a : accounts) {
      if (!source::token(a, 64) || !seen.insert(a).second) {
        state_->close();
        state_->state = SessionState::failed;
        return Status::malformed;
      }
      if (a == state_->connection.account)
        found = true;
    }
    if (status == Status::ok && !found)
      status = Status::not_authorized;
    if (status != Status::ok) {
      state_->close();
      state_->state = SessionState::failed;
      return status;
    }
    state_->account = true;
    state_->ready();
    return Status::ok;
  } catch (const std::bad_alloc &) {
    state_->close();
    return Status::no_memory;
  } catch (...) {
    state_->close();
    return Status::internal_error;
  }
}
Status Session::start(const Request &input, std::int32_t &id) noexcept {
  if (!state_)
    return Status::invalid_argument;
  auto &s = *state_;
  if (!s.local() || s.state != SessionState::ready || s.active)
    return Status::stale;
  if (input.request_id != 0 || !input.session_generation.empty() ||
      !input.account.empty() || !input.model.empty() ||
      !input.private_scope.empty())
    return Status::invalid_argument;
  auto now = detail::Session::Clock::now();
  if (now < s.next_start ||
      (input.operation == Operation::historical_bars && now < s.next_history) ||
      s.next_id >
          static_cast<std::uint64_t>(std::numeric_limits<std::int32_t>::max()))
    return Status::limit;
  try {
    Request r = input;
    r.request_id = static_cast<std::int32_t>(s.next_id);
    r.session_generation = s.generation;
    r.account = s.connection.account;
    r.model = s.connection.model;
    r.private_scope = s.connection.private_scope;
    // Reserve before an uncertain dispatch. IDs and pacing survive reconnect.
    ++s.next_id;
    s.next_start = now + std::chrono::milliseconds(100);
    if (r.operation == Operation::historical_bars)
      s.next_history = now + std::chrono::seconds(16);
    auto status = Collector::start(r, *s.driver, s.collector);
    s.active = bool(s.collector);
    if (s.active)
      id = r.request_id;
    return status;
  } catch (const std::bad_alloc &) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}
Status Session::position(std::string_view g, std::int32_t id,
                         const Position &p) noexcept {
  return state_ && state_->local() && state_->active
             ? state_->collector.position(g, id, p)
             : Status::stale;
}
Status Session::historical_bar(std::string_view g, std::int32_t id,
                               const Bar &b) noexcept {
  return state_ && state_->local() && state_->active
             ? state_->collector.historical_bar(g, id, b)
             : Status::stale;
}
Status Session::end(std::string_view g, std::int32_t id, std::string_view a,
                    std::string_view b) noexcept {
  return state_ && state_->local() && state_->active
             ? state_->collector.end(g, id, a, b)
             : Status::stale;
}
Status Session::fail(std::string_view g, std::int32_t id,
                     std::int32_t code) noexcept {
  return state_ && state_->local() && state_->active
             ? state_->collector.fail(g, id, code)
             : Status::stale;
}
Status Session::disconnected(std::string_view g) noexcept {
  if (!state_)
    return Status::invalid_argument;
  auto status = state_->admission(g);
  if (status != Status::ok)
    return status;
  state_->close();
  return Status::ok;
}
Status Session::poll(std::stop_token stop) noexcept {
  if (!state_ || !state_->local())
    return Status::stale;
  auto &s = *state_;
  if (stop.stop_requested()) {
    if (s.active)
      (void)s.collector.poll(stop);
    s.close();
    return Status::cancelled;
  }
  if (s.state != SessionState::connecting && s.state != SessionState::ready)
    return Status::stale;
  if (s.state == SessionState::connecting &&
      detail::Session::Clock::now() >= s.deadline) {
    s.close();
    return Status::timeout;
  }
  try {
    auto status = s.driver->pump();
    if (status != Status::ok) {
      s.close();
      return status;
    }
    if (s.state != SessionState::connecting && s.state != SessionState::ready)
      return Status::stale;
    if (s.state == SessionState::connecting &&
        detail::Session::Clock::now() >= s.deadline) {
      s.close();
      return Status::timeout;
    }
    return s.active ? s.collector.poll(stop) : Status::ok;
  } catch (...) {
    s.close();
    return Status::transport_error;
  }
}
Status Session::snapshot(Snapshot &out) const noexcept {
  return state_ && state_->local() && state_->active
             ? state_->collector.snapshot(out)
             : Status::stale;
}
Status Session::release_request() noexcept {
  if (!state_ || !state_->local() || !state_->active)
    return Status::stale;
  Snapshot out;
  auto status = state_->collector.snapshot(out);
  if (status != Status::ok)
    return status;
  state_->collector.reset();
  state_->active = false;
  return Status::ok;
}
Status Session::reconnect() noexcept {
  if (!state_ || !state_->local() || state_->connected || state_->active)
    return Status::stale;
  try {
    return state_->connect();
  } catch (const std::bad_alloc &) {
    state_->close();
    return Status::no_memory;
  } catch (...) {
    state_->close();
    return Status::transport_error;
  }
}
void Session::close() noexcept {
  if (state_)
    state_->close();
}
} // namespace symphony::scabv::tws
