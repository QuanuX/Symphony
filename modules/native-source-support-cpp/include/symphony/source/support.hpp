#ifndef SYMPHONY_SOURCE_SUPPORT_HPP
#define SYMPHONY_SOURCE_SUPPORT_HPP
#include <cstdint>
#include <span>
#include <stop_token>
#include <string>
#include <string_view>
#include <vector>
namespace symphony::source {
using Bytes = std::span<const std::uint8_t>;
enum class Status : std::uint8_t {
  ok,
  invalid_argument,
  limit,
  malformed,
  binding_mismatch,
  unsupported,
  no_memory,
  internal_error,
  cancelled,
  timeout,
  transport_error,
  http_error,
  not_authorized,
  stale
};
struct JsonLimits {
  std::uint32_t bytes = 0, values = 0, string_bytes = 0;
  std::uint16_t depth = 0;
};
[[nodiscard]] bool valid_limits(const JsonLimits &) noexcept;
[[nodiscard]] bool date(std::string_view) noexcept;
[[nodiscard]] bool token(std::string_view, std::size_t maximum = 128) noexcept;
// Exact finite decimal syntax, without floating-point conversion.
[[nodiscard]] bool decimal(std::string_view) noexcept;
[[nodiscard]] std::string encode(std::string_view);
[[nodiscard]] std::string digest(std::string_view);
struct HttpRequest {
  std::string
      endpoint; // Explicit HTTPS origin/path; no query, userinfo or fragment.
  std::string parameters; // Nonsecret encoded parameters only.
  bool post = false;
  enum class Credential : std::uint8_t {
    none,
    basic_username,
    fred_query_key
  } credential = Credential::none;
  [[nodiscard]] std::string reference() const;
};
struct HttpLimits {
  std::uint32_t timeout_ms = 0, connect_timeout_ms = 0, max_header_bytes = 0,
                max_body_bytes = 0;
};
struct HttpResponse {
  std::uint16_t http_status = 0;
  std::vector<std::uint8_t> body;
  bool complete = false;
};
class SecretSink {
public:
  virtual ~SecretSink() = default;
  [[nodiscard]] virtual Status use(Bytes) = 0;
};
// Actual provider must enforce its own authenticated authority/recipient/lease
// and bounded lifetime. This interface itself does not implement SSIAG.
class CredentialUse {
public:
  virtual ~CredentialUse() = default;
  [[nodiscard]] virtual Status with_secret(std::string_view request_reference,
                                           std::uint32_t maximum_duration_ms,
                                           std::stop_token, SecretSink &) = 0;
};
// Explicit one-shot HTTP operation; no retries, diagnostics, redirects,
// proxies, ambient credentials, cookies or trust-verification bypass. Error
// bodies are discarded. Output is published for an attempted operation,
// including failure.
[[nodiscard]] Status fetch(const HttpRequest &, CredentialUse *,
                           const HttpLimits &, std::stop_token,
                           HttpResponse &) noexcept;
} // namespace symphony::source
#endif
