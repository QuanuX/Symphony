#ifndef SYMPHONY_SQAV_DATABENTO_HTTP_HPP
#define SYMPHONY_SQAV_DATABENTO_HTTP_HPP
#include <stop_token>
#include <symphony/sqav/databento/attempts.hpp>
namespace symphony::sqav::databento {
struct HttpLimits {
  std::uint32_t timeout_ms = 0,
                connect_timeout_ms = 0; // Positive; total <=300,000.
  std::uint32_t max_header_bytes =
      0; // Positive; <=65,536 across all responses.
};
enum class HttpStatus : std::uint8_t {
  ok,
  invalid_argument,
  not_authorized,
  cancelled,
  timeout,
  transport_error,
  response_limit,
  malformed_headers,
  attempt_refused,
  persistence_failure,
  no_memory,
  internal_error,
  stale
};
// Implemented by the actual SSIAG native bridge. It must authenticate and hold
// the request-bound lease/recipient/deadline through use, then release/clean
// up. No installed operational implementation is supplied by this HTTP library.
// Arbitrary callbacks or test providers are not evidence of SSIAG
// authorization.
class HistoricalKeySink {
public:
  virtual ~HistoricalKeySink() = default;
  [[nodiscard]] virtual HttpStatus use(ByteView borrowed_key) noexcept = 0;
};
class SsiagHistoricalUse {
public:
  virtual ~SsiagHistoricalUse() = default;
  [[nodiscard]] virtual HttpStatus with_key(std::string_view request_reference,
                                            std::uint32_t maximum_duration_ms,
                                            std::stop_token,
                                            HistoricalKeySink &) noexcept = 0;
};
struct HttpResult {
  HttpStatus status = HttpStatus::internal_error;
  AttemptStatus persistence = AttemptStatus::closed;
  HistoricalReport response;
  bool retry_after_uninterpreted = false; // HTTP-date or invalid/overflowing delta.
};
// Fixed historical HTTPS endpoint, verified TLS, no redirects, proxies, netrc,
// diagnostics or ambient credentials. Ticket claim precedes credential use.
// One provider callback and one HTTP attempt. No automatic retry or sleep.
// Report is published for operational failures; response output changes only
// when an actual response accumulator was created. Not callable after fork.
[[nodiscard]] HttpResult
execute_historical(const HistoricalPlan &, AttemptLedger &, AttemptTicket &,
                   SsiagHistoricalUse &, const HttpLimits &,
                   std::uint64_t now_unix_ms, std::stop_token,
                   HistoricalResponse &out) noexcept;
} // namespace symphony::sqav::databento
#endif
