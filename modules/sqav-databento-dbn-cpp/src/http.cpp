#include <array>
#include <charconv>
#include <chrono>
#include <curl/curl.h>
#include <limits>
#include <memory>
#include <mutex>
#include <new>
#include <symphony/sqav/databento/http.hpp>
#include <unistd.h>
namespace symphony::sqav::databento {
#ifdef SQAV_HTTP_TESTING
namespace detail {
thread_local std::string test_endpoint;
}
#endif
namespace {
const pid_t load_pid = ::getpid();
using Clock = std::chrono::steady_clock;
bool valid(const HttpLimits &l) noexcept {
  return l.timeout_ms > 0 && l.timeout_ms <= 300000 &&
         l.connect_timeout_ms > 0 && l.connect_timeout_ms <= l.timeout_ms &&
         l.max_header_bytes > 0 && l.max_header_bytes <= 65536;
}
CURLcode runtime() {
  static std::once_flag once;
  static CURLcode code = CURLE_FAILED_INIT;
  std::call_once(once, [] { code = curl_global_init(CURL_GLOBAL_DEFAULT); });
  return code;
}
bool iequal(std::string_view a, std::string_view b) noexcept {
  if (a.size() != b.size())
    return false;
  for (std::size_t i = 0; i < a.size(); ++i) {
    auto c = a[i];
    if (c >= 'A' && c <= 'Z')
      c = static_cast<char>(c - 'A' + 'a');
    if (c != b[i])
      return false;
  }
  return true;
}
struct Sink final : HistoricalKeySink {
  const HistoricalPlan &plan;
  std::uint8_t attempt;
  HttpLimits limits;
  std::stop_token stop;
  Clock::time_point deadline;
  HistoricalResponse response;
  bool used = false, has_response = false, headers_ready = false;
  std::uint16_t code = 0;
  std::uint32_t header_bytes = 0;
  std::optional<std::uint32_t> retry_after;
  bool retry_seen = false, encoding_seen = false, retry_uninterpreted = false;
  HttpStatus last_result = HttpStatus::not_authorized;
  HttpStatus failure = HttpStatus::ok;
  Status append_status = Status::ok;
  Sink(const HistoricalPlan &p, std::uint8_t a, HttpLimits l, std::stop_token s,
       Clock::time_point d)
      : plan(p), attempt(a), limits(l), stop(s), deadline(d) {}
  static size_t header(char *bytes, size_t size, size_t count,
                       void *ptr) noexcept {
    auto &s = *static_cast<Sink *>(ptr);
    if (size && count > std::numeric_limits<size_t>::max() / size) {
      s.failure = HttpStatus::response_limit;
      return 0;
    }
    const auto n = size * count;
    if (n > s.limits.max_header_bytes - s.header_bytes) {
      s.failure = HttpStatus::response_limit;
      return 0;
    }
    s.header_bytes += static_cast<std::uint32_t>(n);
    std::string_view line(bytes, n);
    if (!line.ends_with("\r\n")) {
      s.failure = HttpStatus::malformed_headers;
      return 0;
    }
    line.remove_suffix(2);
    if (line.starts_with("HTTP/")) {
      if (s.headers_ready) {
        s.failure = HttpStatus::malformed_headers;
        return 0;
      }
      const auto space = line.find(' ');
      unsigned status = 0;
      if (space == line.npos || line.size() < space + 4) {
        s.failure = HttpStatus::malformed_headers;
        return 0;
      }
      auto [end, err] = std::from_chars(line.data() + space + 1,
                                        line.data() + space + 4, status);
      if (err != std::errc{} || end != line.data() + space + 4 ||
          status < 100 || status > 599 ||
          (line.size() > space + 4 && line[space + 4] != ' ')) {
        s.failure = HttpStatus::malformed_headers;
        return 0;
      }
      s.code = static_cast<std::uint16_t>(status);
      s.retry_after.reset();
      s.retry_seen = false;
      s.retry_uninterpreted = false;
      s.encoding_seen = false;
      return n;
    }
    if (line.empty()) {
      if (s.code < 100) {
        s.failure = HttpStatus::malformed_headers;
        return 0;
      }
      if (s.code < 200)
        return n; // interim response; no body accumulator yet.
      if (s.headers_ready)
        return n;
      const auto status = HistoricalResponse::begin(s.plan, s.attempt, s.code,
                                                    s.retry_after, s.response);
      if (status != Status::ok) {
        s.failure = status == Status::no_memory ? HttpStatus::no_memory
                                                : HttpStatus::internal_error;
        return 0;
      }
      s.headers_ready = s.has_response = true;
      return n;
    }
    // Trailers do not rewrite admission facts; reject them in this exact
    // profile.
    if (s.headers_ready) {
      s.failure = HttpStatus::malformed_headers;
      return 0;
    }
    const auto colon = line.find(':');
    if (colon == line.npos || colon == 0) {
      s.failure = HttpStatus::malformed_headers;
      return 0;
    }
    const auto name = line.substr(0, colon);
    auto value = line.substr(colon + 1);
    while (!value.empty() && (value.front() == ' ' || value.front() == '\t'))
      value.remove_prefix(1);
    while (!value.empty() && (value.back() == ' ' || value.back() == '\t'))
      value.remove_suffix(1);
    if (iequal(name, "retry-after")) {
      if (s.retry_seen) {
        s.failure = HttpStatus::malformed_headers;
        return 0;
      }
      s.retry_seen = true;
      std::uint32_t seconds = 0;
      auto [end, error] =
          std::from_chars(value.data(), value.data() + value.size(), seconds);
      // HTTP-date or unrepresentable delay stays unknown and cannot shorten a
      // retry.
      if (error == std::errc{} && end == value.data() + value.size())
        s.retry_after = seconds;
      else {
        s.retry_after.reset();
        s.retry_uninterpreted = true;
      }
    } else if (iequal(name, "content-encoding")) {
      if (s.encoding_seen || !iequal(value, "identity")) {
        s.failure = HttpStatus::malformed_headers;
        return 0;
      }
      s.encoding_seen = true;
    }
    return n;
  }
  static size_t body(char *bytes, size_t size, size_t count,
                     void *ptr) noexcept {
    auto &s = *static_cast<Sink *>(ptr);
    if (size && count > std::numeric_limits<size_t>::max() / size) {
      s.failure = HttpStatus::response_limit;
      return 0;
    }
    const auto n = size * count;
    if (!s.headers_ready) {
      s.failure = HttpStatus::malformed_headers;
      return 0;
    }
    s.append_status =
        s.response.append({reinterpret_cast<const std::uint8_t *>(bytes), n});
    if (s.append_status != Status::ok) {
      s.failure = s.append_status == Status::limit ? HttpStatus::response_limit
                  : s.append_status == Status::no_memory
                      ? HttpStatus::no_memory
                      : HttpStatus::internal_error;
      return 0;
    }
    return n;
  }
  static int progress(void *p, curl_off_t, curl_off_t, curl_off_t,
                      curl_off_t) noexcept {
    auto &s = *static_cast<Sink *>(p);
    return s.stop.stop_requested() || Clock::now() >= s.deadline;
  }
  HttpStatus use(ByteView key) noexcept override {
    if (used) {
      failure = HttpStatus::not_authorized;
      return failure;
    }
    used = true;
    last_result = perform(key);
    return last_result;
  }
  HttpStatus perform(ByteView key) noexcept {
    if (stop.stop_requested())
      return HttpStatus::cancelled;
    if (Clock::now() >= deadline)
      return HttpStatus::timeout;
    if (key.size() != 32 || key[0] != 'd' || key[1] != 'b' || key[2] != '-')
      return HttpStatus::not_authorized;
    for (std::size_t i = 3; i < key.size(); ++i)
      if (!((key[i] >= 'A' && key[i] <= 'Z') ||
            (key[i] >= 'a' && key[i] <= 'z') ||
            (key[i] >= '0' && key[i] <= '9')))
        return HttpStatus::not_authorized;
    try {
      if (runtime() != CURLE_OK)
        return HttpStatus::internal_error;
      auto handle = std::unique_ptr<CURL, decltype(&curl_easy_cleanup)>(
          curl_easy_init(), curl_easy_cleanup);
      if (!handle)
        return HttpStatus::no_memory;
      struct Key {
        std::array<char, 33> b{};
        ~Key() {
          volatile char *p = b.data();
          for (std::size_t i = 0; i < b.size(); ++i)
            p[i] = 0;
        }
      } storage;
      std::copy(key.begin(), key.end(), storage.b.begin());
      const auto remaining =
          std::chrono::duration_cast<std::chrono::milliseconds>(deadline -
                                                                Clock::now())
              .count();
      if (remaining <= 0)
        return HttpStatus::timeout;
      const char *endpoint = historical_endpoint.data();
      const char *protocols = "https";
#ifdef SQAV_HTTP_TESTING
      if (!detail::test_endpoint.empty()) {
        endpoint = detail::test_endpoint.c_str();
        protocols = "http";
      }
#endif
      auto set = [&](auto option, auto value) {
        return curl_easy_setopt(handle.get(), option, value) == CURLE_OK;
      };
      if (!set(CURLOPT_URL, endpoint) ||
          !set(CURLOPT_PROTOCOLS_STR, protocols) ||
          !set(CURLOPT_FOLLOWLOCATION, 0L) || !set(CURLOPT_PROXY, "") ||
          !set(CURLOPT_NETRC, static_cast<long>(CURL_NETRC_IGNORED)) ||
          !set(CURLOPT_VERBOSE, 0L) || !set(CURLOPT_SSL_VERIFYPEER, 1L) ||
          !set(CURLOPT_SSL_VERIFYHOST, 2L) ||
          !set(CURLOPT_SSLVERSION,
               static_cast<long>(CURL_SSLVERSION_TLSv1_2)) ||
          !set(CURLOPT_HTTPAUTH, static_cast<long>(CURLAUTH_BASIC)) ||
          !set(CURLOPT_USERNAME, storage.b.data()) ||
          !set(CURLOPT_PASSWORD, "") || !set(CURLOPT_POST, 1L) ||
          !set(CURLOPT_POSTFIELDSIZE_LARGE,
               static_cast<curl_off_t>(plan.parameters().size())) ||
          !set(CURLOPT_POSTFIELDS, plan.parameters().data()) ||
          !set(CURLOPT_NOSIGNAL, 1L) ||
          !set(CURLOPT_TIMEOUT_MS, static_cast<long>(remaining)) ||
          !set(CURLOPT_CONNECTTIMEOUT_MS,
               std::min(static_cast<long>(limits.connect_timeout_ms),
                        static_cast<long>(remaining))) ||
          !set(CURLOPT_HEADERFUNCTION, &header) ||
          !set(CURLOPT_HEADERDATA, this) ||
          !set(CURLOPT_WRITEFUNCTION, &body) || !set(CURLOPT_WRITEDATA, this) ||
          !set(CURLOPT_NOPROGRESS, 0L) ||
          !set(CURLOPT_XFERINFOFUNCTION, &progress) ||
          !set(CURLOPT_XFERINFODATA, this))
        return HttpStatus::internal_error;
      const auto result = curl_easy_perform(handle.get());
      if (stop.stop_requested())
        return HttpStatus::cancelled;
      if (result == CURLE_OPERATION_TIMEDOUT || Clock::now() >= deadline)
        return HttpStatus::timeout;
      if (failure != HttpStatus::ok)
        return failure;
      return result == CURLE_OK && headers_ready ? HttpStatus::ok
                                                 : HttpStatus::transport_error;
    } catch (const std::bad_alloc &) {
      return HttpStatus::no_memory;
    } catch (...) {
      return HttpStatus::internal_error;
    }
  }
};
} // namespace
HttpResult execute_historical(const HistoricalPlan &plan, AttemptLedger &ledger,
                              AttemptTicket &ticket,
                              SsiagHistoricalUse &provider,
                              const HttpLimits &limits, std::uint64_t now,
                              std::stop_token stop,
                              HistoricalResponse &out) noexcept {
  HttpResult result;
  if (::getpid() != load_pid) {
    result.status = HttpStatus::stale;
    return result;
  }
  if (!plan || !valid(limits)) {
    result.status = HttpStatus::invalid_argument;
    return result;
  }
  if (stop.stop_requested()) {
    result.status = HttpStatus::cancelled;
    return result;
  }
  try {
    result.persistence = ticket.claim(plan, now);
    if (result.persistence != AttemptStatus::ok) {
      result.status = HttpStatus::attempt_refused;
      return result;
    }
    Sink sink(plan, ticket.ordinal(), limits, stop,
              Clock::now() + std::chrono::milliseconds(limits.timeout_ms));
    result.status =
        provider.with_key(plan.reference(), limits.timeout_ms, stop, sink);
    if (result.status == HttpStatus::ok &&
        (!sink.used || sink.last_result != HttpStatus::ok))
      result.status = sink.last_result;
    if (stop.stop_requested())
      result.status = HttpStatus::cancelled;
    else if (Clock::now() >= sink.deadline)
      result.status = HttpStatus::timeout;
    // A provider must propagate the sink result. Its own refusal cannot promote
    // an interrupted response to successful completion.
    if (result.status == HttpStatus::ok && sink.failure != HttpStatus::ok)
      result.status = sink.failure;
    if (sink.has_response) {
      const auto end = result.status == HttpStatus::ok ? TransportEnd::complete
                       : result.status == HttpStatus::cancelled
                           ? TransportEnd::cancelled
                           : TransportEnd::interrupted;
      if (sink.response.finish(end, result.response) != Status::ok)
        result.status = HttpStatus::internal_error;
      result.retry_after_uninterpreted = sink.retry_uninterpreted;
      if (sink.retry_uninterpreted && result.response.recovery == Recovery::repeat_window)
        result.response.recovery = Recovery::review;
      out = std::move(sink.response);
    } else {
      result.response.request = plan;
      result.response.attempt = ticket.ordinal();
      result.response.transport = result.status == HttpStatus::cancelled
                                      ? TransportEnd::cancelled
                                      : TransportEnd::interrupted;
    }
    const auto outcome =
        result.status == HttpStatus::ok          ? AttemptOutcome::completed
        : result.status == HttpStatus::cancelled ? AttemptOutcome::cancelled
        : !sink.used                             ? AttemptOutcome::rejected
                     : AttemptOutcome::indeterminate;
    result.persistence = ledger.finish(ticket, outcome);
    if (result.persistence != AttemptStatus::ok &&
        result.persistence != AttemptStatus::duplicate)
      result.status = HttpStatus::persistence_failure;
    return result;
  } catch (const std::bad_alloc &) {
    result.status = HttpStatus::no_memory;
    return result;
  } catch (...) {
    result.status = HttpStatus::internal_error;
    return result;
  }
}
} // namespace symphony::sqav::databento
