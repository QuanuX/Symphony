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
