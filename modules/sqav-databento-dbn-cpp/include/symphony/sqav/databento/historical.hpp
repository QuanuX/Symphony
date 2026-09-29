#ifndef SYMPHONY_SQAV_DATABENTO_HISTORICAL_HPP
#define SYMPHONY_SQAV_DATABENTO_HISTORICAL_HPP
#include <memory>
#include <symphony/sqav/databento/dbn.hpp>

namespace symphony::sqav::databento {
inline constexpr std::string_view historical_endpoint =
    "https://hist.databento.com/v0/timeseries.get_range";
// This local profile selects explicit raw symbols, MBO, instrument_id output,
// DBN with no compression. No credentials, transport or cost authority are
// held.
struct HistoricalSelection {
  std::string dataset;
  std::vector<std::string> symbols;
  std::uint64_t start = 0, end = 0, record_limit = 0; // ns, [start,end)
};
struct HistoricalLimits {
  Limits dbn;
  std::uint64_t max_window_ns = 0; // Positive; <= 24 hours.
  std::uint16_t max_symbols = 0;   // Positive; <= 128.
  std::uint8_t max_attempts =
      0; // Positive; <= 8, caller tracks actual attempts.
  std::uint32_t max_retry_after_seconds = 0; // Positive; <= 86,400.
};
namespace detail {
struct HistoricalPlanState;
struct HistoricalResponseState;
} // namespace detail
class HistoricalPlan final {
public:
  [[nodiscard]] static Status create(const HistoricalSelection &,
                                     const HistoricalLimits &,
                                     HistoricalPlan &out) noexcept;
  [[nodiscard]] explicit operator bool() const noexcept { return bool(state_); }
  [[nodiscard]] std::string_view reference() const noexcept;
  [[nodiscard]] std::string_view parameters() const noexcept;
  [[nodiscard]] const HistoricalSelection *selection() const noexcept;
  [[nodiscard]] const HistoricalLimits *limits() const noexcept;
  // Reacquire both halves; this does not consume/commit a partial response.
  // A one-nanosecond window cannot split. Failure preserves both outputs.
  [[nodiscard]] Status split(HistoricalPlan &left,
                             HistoricalPlan &right) const noexcept;

private:
  std::shared_ptr<const detail::HistoricalPlanState> state_;
};
enum class TransportEnd : std::uint8_t { complete, interrupted, cancelled };
enum class HistoricalOutcome : std::uint8_t {
  complete_window,
  partial_window,
  interrupted,
  cancelled,
  http_error,
  byte_limit,
  invalid_dbn,
  binding_mismatch
};
enum class Recovery : std::uint8_t {
  none,
  repeat_window,
  split_window,
  review
};
struct HistoricalReport {
  HistoricalPlan
      request; // Retains exact selection/policy, including on failure.
  HistoricalOutcome outcome = HistoricalOutcome::interrupted;
  Recovery recovery = Recovery::review;
  Coverage coverage = Coverage::gap;
  Status dbn_status = Status::invalid_argument;
  std::uint16_t http_status = 0;
  std::uint8_t attempt = 0;
  TransportEnd transport = TransportEnd::interrupted;
  std::uint64_t accepted_bytes = 0, records = 0;
  std::optional<std::uint64_t> first_ts_recv, last_ts_recv;
  std::optional<std::uint32_t> retry_after_seconds;
  bool record_limit_reached = false, unresolved_symbols = false;
};
struct HistoricalAttribution {
  std::string attempt_id, observer_ref, dataset_revision, access_scope;
  TimeEvidence acquisition;
};
// Move-only, single-thread-owned response accumulator. Append never retains
// caller storage. Non-200 bodies are counted and discarded. No raw headers.
class HistoricalResponse final {
public:
  HistoricalResponse() noexcept;
  ~HistoricalResponse();
  HistoricalResponse(HistoricalResponse &&) noexcept;
  HistoricalResponse &operator=(HistoricalResponse &&) noexcept;
  HistoricalResponse(const HistoricalResponse &) = delete;
  HistoricalResponse &operator=(const HistoricalResponse &) = delete;
  [[nodiscard]] static Status
  begin(const HistoricalPlan &, std::uint8_t attempt, std::uint16_t http_status,
        std::optional<std::uint32_t> retry_after_seconds,
        HistoricalResponse &out) noexcept;
  [[nodiscard]] Status append(ByteView) noexcept;
  // Completion is the trusted transport caller's observation, not DBN EOF.
  // Classifies failures in report; ok means classification succeeded.
  [[nodiscard]] Status finish(TransportEnd, HistoricalReport &out) noexcept;
  [[nodiscard]] ByteView body() const noexcept;
  // Only a finished, bound 200 DBN response can become a capture. Its coverage
  // and status evidence are derived internally; caller supplies attribution.
  [[nodiscard]] Status capture(const HistoricalAttribution &,
                               const sqav::Limits &,
                               Capture &out) const noexcept;

private:
  std::unique_ptr<detail::HistoricalResponseState> state_;
};
} // namespace symphony::sqav::databento
#endif
