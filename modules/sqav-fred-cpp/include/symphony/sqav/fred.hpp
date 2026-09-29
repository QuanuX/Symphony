#ifndef SYMPHONY_SQAV_FRED_HPP
#define SYMPHONY_SQAV_FRED_HPP
#include <memory>
#include <optional>
#include <symphony/source/support.hpp>
#include <symphony/sqav/capture.hpp>
namespace symphony::sqav::fred {
using Status = source::Status;
enum class Operation : std::uint8_t { observations, vintage_dates };
struct Selection {
  Operation operation = Operation::observations;
  std::string series, observation_start, observation_end, realtime_start,
      realtime_end;
  std::uint32_t limit = 0, offset = 0;
};
namespace detail {
struct PlanState;
struct PageState;
} // namespace detail
class Plan {
  std::shared_ptr<const detail::PlanState> state_;

public:
  [[nodiscard]] static Status create(const Selection &, Plan &) noexcept;
  [[nodiscard]] explicit operator bool() const noexcept { return bool(state_); }
  [[nodiscard]] const Selection &selection() const noexcept;
  [[nodiscard]] const source::HttpRequest &request() const noexcept;
  [[nodiscard]] std::string_view reference() const noexcept;
};
struct Observation {
  std::string date, realtime_start, realtime_end, value;
  bool missing = false;
};
class Page {
  std::shared_ptr<const detail::PageState> state_;

public:
  [[nodiscard]] static Status admit(const Plan &, const source::HttpResponse &,
                                    const source::JsonLimits &,
                                    Page &) noexcept;
  [[nodiscard]] explicit operator bool() const noexcept { return bool(state_); }
  [[nodiscard]] std::span<const Observation> observations() const noexcept;
  [[nodiscard]] std::span<const std::string> vintage_dates() const noexcept;
  [[nodiscard]] source::Bytes original() const noexcept;
  [[nodiscard]] std::string_view reference() const noexcept;
  [[nodiscard]] std::uint32_t total_count() const noexcept;
  [[nodiscard]] std::optional<std::uint32_t> next_offset() const noexcept;
  [[nodiscard]] Status capture(std::string_view attempt,
                               std::string_view attribution,
                               std::string_view access_scope,
                               const TimeEvidence &acquisition,
                               const sqav::Limits &, Capture &) const noexcept;
};
[[nodiscard]] Status collect(const Plan &, source::CredentialUse &,
                             const source::HttpLimits &,
                             const source::JsonLimits &, std::stop_token,
                             Page &) noexcept;
} // namespace symphony::sqav::fred
#endif
