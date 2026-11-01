#ifndef SYMPHONY_SCABV_CLIENT_PORTAL_HPP
#define SYMPHONY_SCABV_CLIENT_PORTAL_HPP
#include <memory>
#include <optional>
#include <symphony/source/support.hpp>
#include <symphony/sqav/capture.hpp>
namespace symphony::scabv::client_portal {
using Status = source::Status;
namespace detail {
struct BindingState;
struct PlanState;
struct PageState;
} // namespace detail
class Binding {
  std::shared_ptr<const detail::BindingState> state_;

public:
  // Admission verifies the selected account appears in the bounded account-list
  // response. It does not authenticate a caller-supplied response or Gateway.
  [[nodiscard]] static Status
  admit(std::string_view gateway, std::string_view account,
        std::string_view private_scope, const source::HttpResponse &accounts,
        const source::JsonLimits &, Binding &) noexcept;
  [[nodiscard]] explicit operator bool() const noexcept { return bool(state_); }
  [[nodiscard]] std::string_view account() const noexcept;
  [[nodiscard]] std::string_view gateway() const noexcept;
  [[nodiscard]] std::string_view access_scope() const noexcept;
  [[nodiscard]] std::string_view reference() const noexcept;
};
enum class Operation : std::uint8_t { positions, historical_bars };
struct History {
  std::uint32_t conid = 0;
  std::string period, bar, start_time;
  bool outside_regular_hours = false;
};
class Plan {
  std::shared_ptr<const detail::PlanState> state_;

public:
  [[nodiscard]] static Status positions(const Binding &, std::uint32_t page,
                                        Plan &) noexcept;
  [[nodiscard]] static Status historical_bars(const Binding &, const History &,
                                              std::uint32_t max_records,
                                              Plan &) noexcept;
  [[nodiscard]] explicit operator bool() const noexcept { return bool(state_); }
  [[nodiscard]] const source::HttpRequest &request() const noexcept;
  [[nodiscard]] const Binding &binding() const noexcept;
  [[nodiscard]] Operation operation() const noexcept;
  [[nodiscard]] std::uint32_t page() const noexcept;
  [[nodiscard]] std::uint32_t record_limit() const noexcept;
  [[nodiscard]] std::string_view reference() const noexcept;
};
class Page {
  std::shared_ptr<const detail::PageState> state_;

public:
  [[nodiscard]] static Status admit(const Plan &, const source::HttpResponse &,
                                    const source::JsonLimits &,
                                    Page &) noexcept;
  [[nodiscard]] explicit operator bool() const noexcept { return bool(state_); }
  [[nodiscard]] source::Bytes original() const noexcept;
  [[nodiscard]] std::uint32_t records() const noexcept;
  [[nodiscard]] std::optional<std::uint32_t> next_page() const noexcept;
  [[nodiscard]] Status capture(std::string_view attempt,
                               std::string_view attribution,
                               const sqav::TimeEvidence &acquisition,
                               const sqav::Limits &,
                               sqav::Capture &) const noexcept;
};
// Host-owned read session. Its injected transport is a trusted owner boundary,
// not proof of authenticated provider access. The default transport uses the
// existing verified HTTPS fetch with no cookies, secrets, redirects or retries.
class GatewayTransport {
public:
  virtual ~GatewayTransport() = default;
  [[nodiscard]] virtual Status perform(const source::HttpRequest &,
                                       const source::HttpLimits &,
                                       std::stop_token,
                                       source::HttpResponse &) = 0;
};
class HttpsGatewayTransport final : public GatewayTransport {
public:
  [[nodiscard]] Status perform(const source::HttpRequest &,
                               const source::HttpLimits &, std::stop_token,
                               source::HttpResponse &) override;
};
struct SessionConfig {
  std::string gateway, account, private_scope;
  source::HttpLimits http;
  source::JsonLimits json;
  // Local lifetime cap, not a prediction of the provider's expiration time.
  std::uint32_t maximum_age_ms = 0;
};
enum class SessionState : std::uint8_t { ready, expired, failed, closed };
namespace detail { struct Session; }
class Session {
  std::unique_ptr<detail::Session> state_;
public:
  Session() noexcept;
  ~Session() noexcept;
  Session(const Session &) = delete;
  Session &operator=(const Session &) = delete;
  // Explicit status POST then account discovery GET, never login or keepalive.
  // No automatic reconnect. Transport outlives Session; methods are serialized
  // by the host, and the transport must not reenter Session.
  [[nodiscard]] static Status open(const SessionConfig &, GatewayTransport &,
                                    std::stop_token, Session &) noexcept;
  [[nodiscard]] SessionState state() const noexcept;
  [[nodiscard]] const Binding &binding() const noexcept;
  // Each read rechecks status and account membership before the data GET.
  // Failed reads preserve Page output. No automatic pagination or retries.
  [[nodiscard]] Status positions(std::uint32_t page, std::stop_token,
                                 Page &) noexcept;
  [[nodiscard]] Status historical_bars(const History &, std::uint32_t limit,
                                       std::stop_token, Page &) noexcept;
  void close() noexcept;
};
// Gateway must already have an authenticated, selected session and trusted TLS.
// This read surface has no session mutation, order entry, or TLS bypass.
[[nodiscard]] Status connect(std::string_view gateway, std::string_view account,
                             std::string_view private_scope,
                             const source::HttpLimits &,
                             const source::JsonLimits &, std::stop_token,
                             Binding &) noexcept;
[[nodiscard]] Status collect(const Plan &, const source::HttpLimits &,
                             const source::JsonLimits &, std::stop_token,
                             Page &) noexcept;
} // namespace symphony::scabv::client_portal
#endif
