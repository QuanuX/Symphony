#ifndef SYMPHONY_SQAV_DATABENTO_REFERENCE_HPP
#define SYMPHONY_SQAV_DATABENTO_REFERENCE_HPP
#include <memory>
#include <symphony/source/support.hpp>
#include <symphony/sqav/databento/attempts.hpp>
namespace symphony::sqav::databento::reference {
using Status = source::Status;
enum class Operation : std::uint8_t {
  corporate_actions,
  adjustment_factors,
  security_master_range,
  security_master_last
};
struct Selection {
  Operation operation = Operation::security_master_range;
  std::vector<std::string> symbols;
  std::string start, end;
};
struct Limits {
  source::JsonLimits record_json;
  std::uint32_t compressed_bytes = 0, decompressed_bytes = 0, records = 0;
  std::uint8_t window_log = 0;
};
namespace detail {
struct PlanState;
struct PageState;
struct QuoteState;
} // namespace detail
class Plan {
  std::shared_ptr<const detail::PlanState> state_;

public:
  [[nodiscard]] static Status create(const Selection &, Plan &) noexcept;
  [[nodiscard]] explicit operator bool() const noexcept { return bool(state_); }
  [[nodiscard]] const source::HttpRequest &request() const noexcept;
  [[nodiscard]] const Selection &selection() const noexcept;
  [[nodiscard]] std::string_view reference() const noexcept;
};
class Page {
  std::shared_ptr<const detail::PageState> state_;

public:
  [[nodiscard]] static Status admit(const Plan &, const source::HttpResponse &,
                                    const Limits &, Page &) noexcept;
  [[nodiscard]] explicit operator bool() const noexcept { return bool(state_); }
  [[nodiscard]] source::Bytes compressed_original() const noexcept;
  [[nodiscard]] source::Bytes jsonl() const noexcept;
  [[nodiscard]] std::uint32_t records() const noexcept;
  [[nodiscard]] std::string_view reference() const noexcept;
  [[nodiscard]] Status capture(std::string_view attempt,
                               std::string_view attribution,
                               std::string_view access_scope,
                               const TimeEvidence &, const sqav::Limits &,
                               Capture &) const noexcept;
};
[[nodiscard]] Status collect(const Plan &, source::CredentialUse &,
                             const source::HttpLimits &, const Limits &,
                             std::stop_token, Page &) noexcept;
// Exact provider cost response for an existing historical plan. Decimal USD is
// rounded upward to integral nano-USD without binary floating-point arithmetic.
// This is a provider estimate, not a guaranteed invoice or account-wide budget.
class CostQuote {
  std::shared_ptr<const detail::QuoteState> state_;

public:
  [[nodiscard]] static Status request(const HistoricalPlan &,
                                      source::HttpRequest &) noexcept;
  [[nodiscard]] static Status admit(const HistoricalPlan &,
                                    const source::HttpResponse &,
                                    std::uint64_t quoted_unix_ms,
                                    std::uint32_t validity_ms,
                                    CostQuote &) noexcept;
  [[nodiscard]] Status for_attempt(const HistoricalPlan &,
                                   std::uint64_t now_unix_ms,
                                   AttemptQuote &) const noexcept;
  [[nodiscard]] std::uint64_t nano_usd() const noexcept;
};
[[nodiscard]] Status quote(const HistoricalPlan &, source::CredentialUse &,
                           const source::HttpLimits &,
                           std::uint64_t quoted_unix_ms,
                           std::uint32_t validity_ms, std::stop_token,
                           CostQuote &) noexcept;
} // namespace symphony::sqav::databento::reference
#endif
