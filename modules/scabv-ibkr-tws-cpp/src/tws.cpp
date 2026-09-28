#include <bit>
#include <chrono>
#include <concepts>
#include <mutex>
#include <symphony/scabv/tws.hpp>
#include <symphony/source/json.hpp>
#include <unistd.h>
namespace symphony::scabv::tws {
using Clock = std::chrono::steady_clock;
namespace detail {
struct SnapshotState {
  Request request;
  State state;
  std::vector<std::uint8_t> bytes;
  std::string reference;
  std::uint32_t records;
};
struct CollectorState {
  Request request;
  Driver *driver = nullptr;
  std::mutex mutex;
  pid_t pid = ::getpid();
  State state = State::idle;
  Clock::time_point deadline;
  std::string frames;
  std::uint32_t records = 0;
  bool cancelled = false;
  std::string provider_start, provider_end;
  std::int32_t provider_code = 0;
  ~CollectorState() {
    if (pid == ::getpid() && driver && !cancelled && state != State::idle) {
      try {
        (void)driver->cancel(request);
      } catch (...) {
      }
    }
  }
  Status cancel() {
    if (cancelled)
      return Status::ok;
    auto s = driver->cancel(request);
    if (s == Status::ok)
      cancelled = true;
    return s;
  }
};
} // namespace detail
namespace {
bool text(std::string_view s, std::size_t max) {
  return !s.empty() && s.size() <= max && s.find('\0') == s.npos;
}
bool valid(const Request &r) {
  if (static_cast<unsigned>(r.operation) > 1 || r.request_id <= 0 ||
      !source::token(r.session_generation) || !source::token(r.account, 64) ||
      (!r.model.empty() && !source::token(r.model)) ||
      !r.private_scope.starts_with("private:") ||
      !source::token(r.private_scope, 256) || r.max_records == 0 ||
      r.max_records > 100000 || r.max_payload_bytes < 1024 ||
      r.max_payload_bytes > (64U << 20) || r.timeout_ms == 0 ||
      r.timeout_ms > 300000)
    return false;
  if (r.operation == Operation::positions)
    return r.conid == 0 && r.exchange.empty() && r.end_utc.empty() &&
           r.duration.empty() && r.bar.empty();
  // Explicit finite history, RTH only, formatDate=2, keepUpToDate=false.
  // Provider validates exact duration/bar compatibility under its own pacing
  // rules.
  if (r.conid == 0 || r.conid > 2147483647 || !source::token(r.exchange, 32) ||
      !text(r.end_utc, 64) || !r.end_utc.ends_with(" UTC") ||
      !text(r.duration, 32) || !text(r.bar, 32))
    return false;
  return r.end_utc.find_first_of("\r\n") == r.end_utc.npos &&
         r.duration.find_first_of("\r\n") == r.duration.npos &&
         r.bar.find_first_of("\r\n") == r.bar.npos;
}
Status ready(detail::CollectorState &s, std::string_view generation,
             std::int32_t id, Operation op) {
  if (generation != s.request.session_generation ||
      id != s.request.request_id || op != s.request.operation)
    return Status::binding_mismatch;
  if (s.state != State::receiving)
    return Status::stale;
  if (Clock::now() >= s.deadline) {
    s.state = State::cancelled;
    (void)s.cancel();
    return Status::timeout;
  }
  return Status::ok;
}
// Flat callback projections avoid JSON container destruction, which can
// allocate in the pinned JSON dependency. Scalar string destruction does not.
struct Field {
  std::string_view key;
  std::string value;
  Field(std::string_view k, std::string_view v)
      : key(k), value(nlohmann::json(v).dump()) {}
  Field(std::string_view k, const std::string &v)
      : Field(k, std::string_view(v)) {}
  Field(std::string_view k, const char *v) : Field(k, std::string_view(v)) {}
  template <std::integral T>
  Field(std::string_view k, T v) : key(k), value(std::to_string(v)) {}
};
std::string encode(std::initializer_list<Field> fields) {
  std::string out = "{";
  bool first = true;
  for (const auto &f : fields) {
    if (!first)
      out += ',';
    first = false;
    out += '"';
    out += f.key;
    out += "\":";
    out += f.value;
  }
  out += '}';
  return out;
}
Status append(detail::CollectorState &s, std::initializer_list<Field> j) {
  auto frame = encode(j);
  frame += '\n';
  if (s.records >= s.request.max_records ||
      frame.size() > s.request.max_payload_bytes - s.frames.size()) {
    s.state = State::limit;
    (void)s.cancel();
    return Status::limit;
  }
  s.frames += frame;
  ++s.records;
  return Status::ok;
}
} // namespace
Collector::Collector() noexcept = default;
Collector::~Collector() noexcept = default;
Collector::Collector(Collector &&) noexcept = default;
Collector &Collector::operator=(Collector &&) noexcept = default;
void Collector::reset() noexcept { state_.reset(); }
Status Collector::start(const Request &r, Driver &driver,
                        Collector &out) noexcept {
  if (!valid(r) || driver.release() != sdk_release || out.state_)
    return Status::invalid_argument;
  try {
    auto s = std::make_unique<detail::CollectorState>();
    s->request = r;
    s->driver = &driver;
    s->deadline = Clock::now() + std::chrono::milliseconds(r.timeout_ms);
    s->state = State::receiving;
    out.state_ = std::move(s);
    std::lock_guard lock(out.state_->mutex);
    Status result = Status::internal_error;
    try {
      result = driver.start(r);
    } catch (const std::bad_alloc &) {
      result = Status::no_memory;
    } catch (...) {
      result = Status::internal_error;
    }
    if (result != Status::ok) {
      out.state_->state = State::failed;
      (void)out.state_->cancel();
    }
    return result;
  } catch (const std::bad_alloc &) {
    if (out.state_) {
      try {
        std::lock_guard lock(out.state_->mutex);
        out.state_->state = State::failed;
        (void)out.state_->cancel();
      } catch (...) {
      }
    }
    return Status::no_memory;
  } catch (...) {
    if (out.state_) {
      try {
        std::lock_guard lock(out.state_->mutex);
        out.state_->state = State::failed;
        (void)out.state_->cancel();
      } catch (...) {
      }
    }
    return Status::internal_error;
  }
}
Status Collector::position(std::string_view generation, std::int32_t id,
                           const Position &p) noexcept {
  if (!state_)
    return Status::invalid_argument;
  if (state_->pid != ::getpid())
    return Status::stale;
  try {
    std::lock_guard lock(state_->mutex);
    auto &s = *state_;
    auto status = ready(s, generation, id, Operation::positions);
    if (status != Status::ok)
      return status;
    if (p.account != s.request.account || p.model != s.request.model) {
      s.state = State::failed;
      (void)s.cancel();
      return Status::binding_mismatch;
    }
    if (p.conid == 0 || p.conid > 2147483647 || !text(p.symbol, 128) ||
        !source::token(p.security_type, 32) || !source::token(p.currency, 32) ||
        (!p.exchange.empty() && !source::token(p.exchange, 32)) ||
        !source::decimal(p.quantity)) {
      s.state = State::failed;
      (void)s.cancel();
      return Status::malformed;
    }
    try {
      return append(s, {{"type", "positionMulti"},
                        {"account", p.account},
                        {"model", p.model},
                        {"conid", p.conid},
                        {"symbol", p.symbol},
                        {"security_type", p.security_type},
                        {"currency", p.currency},
                        {"exchange", p.exchange},
                        {"quantity_decimal", p.quantity},
                        {"average_cost_ieee754_bits",
                         std::bit_cast<std::uint64_t>(p.average_cost)}});
    } catch (const std::bad_alloc &) {
      s.state = State::failed;
      try {
        (void)s.cancel();
      } catch (...) {
      }
      return Status::no_memory;
    } catch (...) {
      s.state = State::failed;
      try {
        (void)s.cancel();
      } catch (...) {
      }
      return Status::internal_error;
    }
  } catch (...) {
    return Status::internal_error;
  }
}
Status Collector::historical_bar(std::string_view generation, std::int32_t id,
                                 const Bar &b) noexcept {
  if (!state_)
    return Status::invalid_argument;
  if (state_->pid != ::getpid())
    return Status::stale;
  try {
    std::lock_guard lock(state_->mutex);
    auto &s = *state_;
    auto status = ready(s, generation, id, Operation::historical_bars);
    if (status != Status::ok)
      return status;
    if (!text(b.time, 64) || !source::decimal(b.volume) ||
        !source::decimal(b.weighted_average) || b.count < 0) {
      s.state = State::failed;
      (void)s.cancel();
      return Status::malformed;
    }
    try {
      return append(
          s, {{"type", "historicalData"},
              {"time", b.time},
              {"open_ieee754_bits", std::bit_cast<std::uint64_t>(b.open)},
              {"high_ieee754_bits", std::bit_cast<std::uint64_t>(b.high)},
              {"low_ieee754_bits", std::bit_cast<std::uint64_t>(b.low)},
              {"close_ieee754_bits", std::bit_cast<std::uint64_t>(b.close)},
              {"volume_decimal", b.volume},
              {"wap_decimal", b.weighted_average},
              {"count", b.count}});
    } catch (const std::bad_alloc &) {
      s.state = State::failed;
      try {
        (void)s.cancel();
      } catch (...) {
      }
      return Status::no_memory;
    } catch (...) {
      s.state = State::failed;
      try {
        (void)s.cancel();
      } catch (...) {
      }
      return Status::internal_error;
    }
  } catch (...) {
    return Status::internal_error;
  }
}
Status Collector::end(std::string_view generation, std::int32_t id,
                      std::string_view first, std::string_view last) noexcept {
  if (!state_)
    return Status::invalid_argument;
  if (state_->pid != ::getpid())
    return Status::stale;
  try {
    std::lock_guard lock(state_->mutex);
    auto &s = *state_;
    auto status = ready(s, generation, id, s.request.operation);
    if (status != Status::ok)
      return status;
    // A matching end callback is terminal even if validation, cancellation or
    // allocation fails. A concurrent callback cannot promote failed completion.
    s.state = State::failed;
    try {
      if (s.request.operation == Operation::positions &&
          (!first.empty() || !last.empty())) {
        (void)s.cancel();
        return Status::invalid_argument;
      }
      if (s.request.operation == Operation::historical_bars &&
          (!text(first, 64) || !text(last, 64))) {
        (void)s.cancel();
        return Status::invalid_argument;
      }
      std::string a(first), b(last);
      auto cancelled = s.cancel();
      if (cancelled != Status::ok)
        return cancelled;
      s.provider_start = std::move(a);
      s.provider_end = std::move(b);
      s.state = State::complete;
      return Status::ok;
    } catch (const std::bad_alloc &) {
      try {
        (void)s.cancel();
      } catch (...) {
      }
      return Status::no_memory;
    } catch (...) {
      try {
        (void)s.cancel();
      } catch (...) {
      }
      return Status::internal_error;
    }
  } catch (const std::bad_alloc &) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}
Status Collector::fail(std::string_view generation, std::int32_t id,
                       std::int32_t code) noexcept {
  if (!state_)
    return Status::invalid_argument;
  if (state_->pid != ::getpid())
    return Status::stale;
  try {
    std::lock_guard lock(state_->mutex);
    auto &s = *state_;
    if (generation != s.request.session_generation ||
        id != s.request.request_id)
      return Status::binding_mismatch;
    if (s.state != State::receiving)
      return Status::stale;
    s.provider_code = code;
    s.state = State::failed;
    return s.cancel();
  } catch (...) {
    return Status::internal_error;
  }
}
Status Collector::disconnected(std::string_view generation) noexcept {
  if (!state_)
    return Status::invalid_argument;
  if (state_->pid != ::getpid())
    return Status::stale;
  try {
    std::lock_guard lock(state_->mutex);
    auto &s = *state_;
    if (generation != s.request.session_generation)
      return Status::binding_mismatch;
    if (s.state != State::receiving)
      return Status::stale;
    s.state = State::disconnected;
    return s.cancel();
  } catch (...) {
    return Status::internal_error;
  }
}
Status Collector::poll(std::stop_token stop) noexcept {
  if (!state_)
    return Status::invalid_argument;
  if (state_->pid != ::getpid())
    return Status::stale;
  try {
    std::lock_guard lock(state_->mutex);
    auto &s = *state_;
    if (s.state != State::receiving)
      return Status::stale;
    if (stop.stop_requested() || Clock::now() >= s.deadline) {
      s.state = State::cancelled;
      auto status = s.cancel();
      return status == Status::ok
                 ? (stop.stop_requested() ? Status::cancelled : Status::timeout)
                 : status;
    }
    return Status::ok;
  } catch (...) {
    return Status::internal_error;
  }
}
Status Collector::snapshot(Snapshot &out) const noexcept {
  if (!state_)
    return Status::invalid_argument;
  if (state_->pid != ::getpid())
    return Status::stale;
  try {
    std::lock_guard lock(state_->mutex);
    auto &s = *state_;
    if (s.state == State::receiving)
      return Status::stale;
    auto snapshot = std::make_shared<detail::SnapshotState>();
    snapshot->request = s.request;
    snapshot->state = s.state;
    snapshot->records = s.records;
    auto header = encode({{"schema", "ibkr-tws-callback-projection-v1"},
                          {"sdk_release", sdk_release},
                          {"session_generation", s.request.session_generation},
                          {"request_id", s.request.request_id},
                          {"operation", unsigned(s.request.operation)},
                          {"account", s.request.account},
                          {"model", s.request.model},
                          {"conid", s.request.conid},
                          {"exchange", s.request.exchange},
                          {"end_utc", s.request.end_utc},
                          {"duration", s.request.duration},
                          {"bar", s.request.bar},
                          {"state", unsigned(s.state)},
                          {"provider_start", s.provider_start},
                          {"provider_end", s.provider_end},
                          {"provider_code", s.provider_code},
                          {"records", s.records}});
    auto payload = header + "\n" + s.frames;
    if (payload.size() > std::uint64_t(s.request.max_payload_bytes) + 4096)
      return Status::limit;
    snapshot->bytes.assign(payload.begin(), payload.end());
    snapshot->reference = "ibkr-tws-snapshot-v1-" + source::digest(payload);
    out.state_ = std::move(snapshot);
    return Status::ok;
  } catch (const std::bad_alloc &) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}
State Snapshot::state() const noexcept {
  return state_ ? state_->state : State::idle;
}
source::Bytes Snapshot::payload() const noexcept {
  return state_ ? source::Bytes(state_->bytes) : source::Bytes{};
}
std::uint32_t Snapshot::records() const noexcept {
  return state_ ? state_->records : 0;
}
Status Snapshot::capture(std::string_view attempt, std::string_view attribution,
                         const sqav::TimeEvidence &acq,
                         const sqav::Limits &limits,
                         sqav::Capture &out) const noexcept {
  if (!state_ || acq.role != sqav::TimeRole::acquisition)
    return Status::invalid_argument;
  try {
    const auto &r = state_->request;
    const auto operation = r.operation == Operation::positions
                               ? "reqPositionsMulti"
                               : "reqHistoricalData";
    sqav::Description d{
        {"ibkr", "tws-cpp", std::string(sdk_release), operation,
         "scabv-ibkr-tws-cpp", "0.1.0-dev", r.account, r.session_generation,
         state_->reference, "ibkr-tws-callback-projection-v1",
         "utf8-ndjson;ieee754-bits;decimal-text", r.private_scope},
        std::string(attempt),
        std::string(attribution),
        "request=" + std::to_string(r.request_id),
        state_->state == State::complete ? sqav::Coverage::complete
                                         : sqav::Coverage::gap,
        "selected-callback-fields;initial-response;not-atomic-account",
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
} // namespace symphony::scabv::tws
