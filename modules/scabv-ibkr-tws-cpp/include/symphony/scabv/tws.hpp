#ifndef SYMPHONY_SCABV_TWS_HPP
#define SYMPHONY_SCABV_TWS_HPP
#include <memory>
#include <symphony/source/support.hpp>
#include <symphony/sqav/capture.hpp>
namespace symphony::scabv::tws {
using Status = source::Status;
inline constexpr std::string_view sdk_release = "10.45";
enum class Operation : std::uint8_t { positions, historical_bars };
struct Request {
  Operation operation = Operation::positions;
  std::int32_t request_id = 0;
  std::string session_generation, account, model, private_scope;
  std::uint32_t conid = 0;
  std::string exchange, end_utc, duration, bar;
  std::uint32_t max_records = 0, max_payload_bytes = 0, timeout_ms = 0;
};
// Only these provider methods are reachable through a research Driver. The
// owning SDK session provides account admission, unique request IDs and pacing.
class Driver {
public:
  virtual ~Driver() = default;
  [[nodiscard]] virtual std::string_view release() const noexcept = 0;
  [[nodiscard]] virtual Status start(const Request &) = 0;
  [[nodiscard]] virtual Status cancel(const Request &) = 0;
};
struct Position {
  std::string account, model;
  std::uint32_t conid = 0;
  std::string symbol, security_type, currency, exchange, quantity;
  double average_cost = 0;
};
struct Bar {
  std::string time;
  double open = 0, high = 0, low = 0, close = 0;
  std::string volume, weighted_average;
  std::int32_t count = 0;
};
enum class State : std::uint8_t {
  idle,
  receiving,
  complete,
  cancelled,
  failed,
  disconnected,
  limit
};
namespace detail {
struct CollectorState;
struct SnapshotState;
} // namespace detail
class Snapshot {
  std::shared_ptr<const detail::SnapshotState> state_;
  friend class Collector;

public:
  [[nodiscard]] explicit operator bool() const noexcept { return bool(state_); }
  [[nodiscard]] State state() const noexcept;
  [[nodiscard]] source::Bytes payload() const noexcept;
  [[nodiscard]] std::uint32_t records() const noexcept;
  [[nodiscard]] Status capture(std::string_view attempt,
                               std::string_view attribution,
                               const sqav::TimeEvidence &acquisition,
                               const sqav::Limits &,
                               sqav::Capture &) const noexcept;
};
// Caller drives the selected SDK event loop and periodically calls poll, which
// enforces the elapsed deadline/stop and cancels the provider request. No live
// updates are admitted after the initial positions end marker; cancellation is
// sent to the provider without claiming a provider cancellation acknowledgment.
// The Driver must outlive Collector; methods must not reenter Collector
// synchronously.
class Collector {
  std::unique_ptr<detail::CollectorState> state_;

public:
  Collector() noexcept;
  ~Collector() noexcept;
  Collector(Collector &&) noexcept;
  Collector &operator=(Collector &&) noexcept;
  Collector(const Collector &) = delete;
  Collector &operator=(const Collector &) = delete;
  // After provider dispatch begins, output remains available even on a start
  // failure, exposing terminal gap state. Preflight failures preserve output.
  [[nodiscard]] static Status start(const Request &, Driver &,
                                    Collector &) noexcept;
  [[nodiscard]] Status position(std::string_view generation,
                                std::int32_t request_id,
                                const Position &) noexcept;
  [[nodiscard]] Status historical_bar(std::string_view generation,
                                      std::int32_t request_id,
                                      const Bar &) noexcept;
  [[nodiscard]] Status end(std::string_view generation, std::int32_t request_id,
                           std::string_view provider_start = {},
                           std::string_view provider_end = {}) noexcept;
  [[nodiscard]] Status fail(std::string_view generation,
                            std::int32_t request_id,
                            std::int32_t provider_code) noexcept;
  [[nodiscard]] Status disconnected(std::string_view generation) noexcept;
  [[nodiscard]] Status poll(std::stop_token = {}) noexcept;
  [[nodiscard]] Status snapshot(Snapshot &) const noexcept;
  void reset() noexcept;
};
// External SDK adaptation is an explicit template instantiation using the
// authorized SDK's EClient, Contract and TagValueListSPtr. No SDK source is
// bundled. Shape fixtures do not establish SDK 10.45 or deployment conformance.
template <class Client, class Contract, class TagOptions>
class Sdk1045Driver final : public Driver {
  Client &client_;

public:
  explicit Sdk1045Driver(Client &client) : client_(client) {}
  std::string_view release() const noexcept override { return sdk_release; }
  Status start(const Request &r) override {
    try {
      if (r.operation == Operation::positions)
        client_.reqPositionsMulti(r.request_id, r.account, r.model);
      else {
        Contract contract{};
        contract.conId = static_cast<decltype(contract.conId)>(r.conid);
        contract.exchange = r.exchange;
        client_.reqHistoricalData(r.request_id, contract, r.end_utc, r.duration,
                                  r.bar, "TRADES", 1, 2, false, TagOptions{});
      }
      return Status::ok;
    } catch (...) {
      return Status::transport_error;
    }
  }
  Status cancel(const Request &r) override {
    try {
      if (r.operation == Operation::positions)
        client_.cancelPositionsMulti(r.request_id);
      else
        client_.cancelHistoricalData(r.request_id);
      return Status::ok;
    } catch (...) {
      return Status::transport_error;
    }
  }
};
} // namespace symphony::scabv::tws
#endif
