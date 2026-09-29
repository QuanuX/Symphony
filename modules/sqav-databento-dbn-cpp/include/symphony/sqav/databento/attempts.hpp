#ifndef SYMPHONY_SQAV_DATABENTO_ATTEMPTS_HPP
#define SYMPHONY_SQAV_DATABENTO_ATTEMPTS_HPP
#include <symphony/sqav/databento/historical.hpp>
#include <symphony/sqpv/local_store.hpp>
namespace symphony::sqav::databento {
enum class AttemptStatus : std::uint8_t {
  ok,
  duplicate,
  invalid_argument,
  limit,
  conflict,
  stale,
  missing,
  busy,
  corrupt,
  unsafe_path,
  io_error,
  outcome_uncertain,
  closed,
  no_memory,
  internal_error
};
enum class AttemptOutcome : std::uint8_t {
  reserved,
  completed,
  rejected,
  cancelled,
  indeterminate
};
struct AttemptBudget {
  sqfv::Generation generation{};
  std::uint64_t ceiling_nano_usd = 0, prior_charge_nano_usd = 0;
  std::uint32_t max_attempts = 0; // 1..4096; at most two records per attempt.
  std::uint64_t max_store_bytes = 0;
};
struct AttemptQuote {
  std::string evidence_ref; // Caller-verified nonsecret quote/reference.
  std::uint64_t ceiling_nano_usd = 0;
  std::uint64_t quoted_unix_ms = 0, expires_unix_ms = 0;
};
struct AttemptSnapshot {
  std::uint64_t charged_ceiling_nano_usd = 0, remaining_nano_usd = 0;
  std::uint32_t attempts = 0, unresolved = 0;
};
namespace detail {
struct AttemptLedgerState;
struct AttemptTicketState;
} // namespace detail
class AttemptLedger;
// A successful reservation is durable before this move-only ticket is returned.
// claim() succeeds once in its original live ledger/process; restart never
// recreates a ticket. No secret, provider authentication or actual cost proof.
class AttemptTicket final {
public:
  AttemptTicket() noexcept;
  ~AttemptTicket() noexcept;
  AttemptTicket(AttemptTicket &&) noexcept;
  AttemptTicket &operator=(AttemptTicket &&) noexcept;
  AttemptTicket(const AttemptTicket &) = delete;
  AttemptTicket &operator=(const AttemptTicket &) = delete;
  [[nodiscard]] AttemptStatus claim(const HistoricalPlan &,
                                    std::uint64_t now_unix_ms) noexcept;
  [[nodiscard]] std::uint8_t ordinal() const noexcept;

private:
  std::unique_ptr<detail::AttemptTicketState> state_;
  friend class AttemptLedger;
};
// Private, exclusively owned SQPV root. Monotone conservative charge
// accounting: no automatic refunds for failures, cancellations or uncertain
// outcomes.
class AttemptLedger final {
public:
  AttemptLedger() noexcept;
  ~AttemptLedger() noexcept;
  AttemptLedger(AttemptLedger &&) noexcept;
  AttemptLedger &operator=(AttemptLedger &&) noexcept;
  AttemptLedger(const AttemptLedger &) = delete;
  AttemptLedger &operator=(const AttemptLedger &) = delete;
  [[nodiscard]] static AttemptStatus create(const std::string &,
                                            const AttemptBudget &,
                                            AttemptLedger &out) noexcept;
  [[nodiscard]] static AttemptStatus
  open(const std::string &, const AttemptBudget &, AttemptLedger &out) noexcept;
  [[nodiscard]] AttemptStatus reserve(const HistoricalPlan &,
                                      std::string_view attempt_id,
                                      const AttemptQuote &,
                                      std::uint64_t now_unix_ms,
                                      AttemptTicket &out) noexcept;
  [[nodiscard]] AttemptStatus finish(const AttemptTicket &,
                                     AttemptOutcome) noexcept;
  [[nodiscard]] AttemptStatus snapshot(AttemptSnapshot &out) const noexcept;
  [[nodiscard]] AttemptStatus lookup(std::string_view attempt_id,
                                     AttemptOutcome &out) const noexcept;
  void reset() noexcept;

private:
  std::shared_ptr<detail::AttemptLedgerState> state_;
};
} // namespace symphony::sqav::databento
#endif
