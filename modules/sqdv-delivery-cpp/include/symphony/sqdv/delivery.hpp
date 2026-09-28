#ifndef SYMPHONY_SQDV_DELIVERY_HPP
#define SYMPHONY_SQDV_DELIVERY_HPP

#include <cstdint>
#include <memory>
#include <string>
#include <string_view>
#include <symphony/sqfv/batch.hpp>
#include <symphony/sqmv/metadata.hpp>
#include <symphony/sqpv/local_store.hpp>
#include <symphony/sqpv/async_store.hpp>

namespace symphony::sqdv {

enum class Status : std::uint8_t {
  ok, invalid_argument, limit, no_memory, blocked, duplicate, conflict, gap,
  stale, missing, corrupt, unsafe_path, io_error, outcome_uncertain, closed,
  empty, unsupported, binding_mismatch, busy, internal_error,
};
enum class Profile : std::uint8_t { disposable = 1, retained_before_delivery = 2, asynchronous_retention = 3 };
enum class Origin : std::uint8_t { live = 1, retained = 2 };

struct Config {
  std::string view_id;
  std::string recipient_id;
  std::string recipient_interface;
  std::string partition;
  sqfv::Generation producer_generation{};
  std::uint64_t first_sequence = 0;
  Profile profile = Profile::disposable;
};
struct Limits {
  std::uint64_t outstanding_byte_credit = 0;
  std::uint32_t max_unacknowledged_batches = 0;
};
struct Checkpoint {
  std::string view_reference;
  std::uint64_t next_sequence = 0;
  bool sequence_exhausted = false;
};
struct SessionStats {
  std::uint64_t next_offer_sequence = 0;
  std::uint64_t next_processed_sequence = 0;
  std::uint64_t outstanding_bytes = 0;
  std::uint32_t unacknowledged_batches = 0;
  std::uint32_t pending_deliveries = 0;
  bool offer_exhausted = false;
  bool processed_exhausted = false;
};

namespace detail {
struct SourceState;
struct RetainedBatchState;
struct DeliveryHandle;
struct SessionState;
}

class RetainedBatch;
class QueuedBatch;
class Session;

// Owns an actual SQPV Store. Retained source handles and sessions share its
// exclusive lock; proof handles do not keep that store open. No Store or caller
// Receipt can be injected. All identities are selected before persistent work.
class RetainedSource final {
 public:
  RetainedSource() noexcept;
  ~RetainedSource() noexcept;
  RetainedSource(RetainedSource&&) noexcept;
  RetainedSource& operator=(RetainedSource&&) noexcept;
  RetainedSource(const RetainedSource&) = delete;
  RetainedSource& operator=(const RetainedSource&) = delete;
  [[nodiscard]] explicit operator bool() const noexcept;
  [[nodiscard]] static Status create(const std::string& absolute_root,
                                     const sqmv::Manifest&, const sqpv::Options&,
                                     RetainedSource& out) noexcept;
  [[nodiscard]] static Status open(const std::string& absolute_root,
                                   const sqmv::Manifest&, const sqpv::Options&,
                                   RetainedSource& out) noexcept;
  [[nodiscard]] static Status create_async(const std::string&, const sqmv::Manifest&,
      const sqpv::Options&, sqfv::Context&, const sqpv::AsyncLimits&, RetainedSource& out) noexcept;
  [[nodiscard]] static Status open_async(const std::string&, const sqmv::Manifest&,
      const sqpv::Options&, sqfv::Context&, const sqpv::AsyncLimits&, RetainedSource& out) noexcept;
  // Produces a queue-admission proof only; preview carries no retention receipt.
  [[nodiscard]] Status enqueue(const sqfv::Batch&, QueuedBatch& out) noexcept;
  [[nodiscard]] Status retention_status(sqpv::AsyncSnapshot& out) const noexcept;
  [[nodiscard]] Status finish_retention() noexcept;
  [[nodiscard]] Status retain(RetainedSource& out) const noexcept;
  // ok and duplicate publish an actual verified retention proof. An uncertain
  // persistent outcome closes this source and its sessions to new retained work;
  // existing processing acknowledgements and checkpoints remain available.
  [[nodiscard]] Status commit(sqfv::Context&, const sqfv::Batch&,
                              RetainedBatch& out) noexcept;
  void reset() noexcept;
 private:
  std::shared_ptr<detail::SourceState> state_;
  friend class Session;
};

class QueuedBatch final {
 public:
  QueuedBatch() noexcept;
  ~QueuedBatch() noexcept;
  QueuedBatch(QueuedBatch&&) noexcept;
  QueuedBatch& operator=(QueuedBatch&&) noexcept;
  QueuedBatch(const QueuedBatch&) = delete;
  QueuedBatch& operator=(const QueuedBatch&) = delete;
  [[nodiscard]] explicit operator bool() const noexcept;
  void reset() noexcept;
 private:
  std::shared_ptr<const detail::RetainedBatchState> state_;
  friend class RetainedSource;
  friend class Session;
};

class RetainedBatch final {
 public:
  RetainedBatch() noexcept;
  ~RetainedBatch() noexcept;
  RetainedBatch(RetainedBatch&&) noexcept;
  RetainedBatch& operator=(RetainedBatch&&) noexcept;
  RetainedBatch(const RetainedBatch&) = delete;
  RetainedBatch& operator=(const RetainedBatch&) = delete;
  [[nodiscard]] explicit operator bool() const noexcept;
  [[nodiscard]] Status retain(RetainedBatch& out) const noexcept;
  // Both accessors require a nonempty handle. Views live with the handle.
  [[nodiscard]] const sqfv::Batch& batch() const noexcept;
  [[nodiscard]] const sqpv::Receipt& receipt() const noexcept;
  void reset() noexcept;
 private:
  std::shared_ptr<const detail::RetainedBatchState> state_;
  friend class RetainedSource;
  friend class Session;
};

// A delivery owns an SQFV lease and an opaque processing ticket. Releasing its
// payload returns byte credit but preserves the ticket; acknowledging processing
// does not release the payload. Reset releases payload and discards the local
// ticket; the session's unacknowledged processing obligation remains until the
// session is reset and resumed from its processed checkpoint.
class Delivery final {
 public:
  Delivery() noexcept;
  ~Delivery() noexcept;
  Delivery(Delivery&&) noexcept;
  Delivery& operator=(Delivery&&) noexcept;
  Delivery(const Delivery&) = delete;
  Delivery& operator=(const Delivery&) = delete;
  [[nodiscard]] explicit operator bool() const noexcept;
  [[nodiscard]] bool has_payload() const noexcept;
  [[nodiscard]] sqfv::ByteView payload() const noexcept;
  // Requires has_payload(). Other direct-value accessors require a nonempty
  // ticket, and remain valid after release_payload().
  [[nodiscard]] const sqfv::Descriptor& descriptor() const noexcept;
  [[nodiscard]] const sqfv::ContentId& content_id() const noexcept;
  [[nodiscard]] std::uint64_t sequence() const noexcept;
  [[nodiscard]] Origin origin() const noexcept;
  // Null for a disposable delivery or empty handle.
  [[nodiscard]] const sqpv::Receipt* retention_receipt() const noexcept;
  void release_payload() noexcept;
  void reset() noexcept;
 private:
  std::unique_ptr<detail::DeliveryHandle> impl_;
  friend class Session;
};

// Trusted same-process full-batch delivery. Calls serialize on each handle and
// reject inherited use after fork before locking. Destruction/move/reset must
// not race calls on the same handle. All fallible calls preserve outputs on
// failure, except commit's explicitly documented duplicate success evidence.
class Session final {
 public:
  Session() noexcept;
  ~Session() noexcept;
  Session(Session&&) noexcept;
  Session& operator=(Session&&) noexcept;
  Session(const Session&) = delete;
  Session& operator=(const Session&) = delete;
  [[nodiscard]] explicit operator bool() const noexcept;
  [[nodiscard]] static Status create(sqfv::Context&, const sqmv::Manifest&,
                                     const Config&, const Limits&,
                                     const RetainedSource* source,
                                     const Checkpoint* resume,
                                     Session& out) noexcept;
  [[nodiscard]] Status offer_preview(const QueuedBatch&) noexcept;
  [[nodiscard]] Status offer_live(const sqfv::Batch&) noexcept;
  [[nodiscard]] Status offer_next(sqfv::Context&,
                                  const RetainedBatch* live_candidate = nullptr) noexcept;
  [[nodiscard]] Status take(Delivery& out) noexcept;
  [[nodiscard]] Status acknowledge_processed(const Delivery&) noexcept;
  [[nodiscard]] Status checkpoint(Checkpoint& out) const noexcept;
  // Requires a retained profile and an actual confirmed prefix covering the
  // processed cursor. Does not persist the returned checkpoint itself.
  [[nodiscard]] Status checkpoint_for_replay(Checkpoint& out) const noexcept;
  [[nodiscard]] Status stats(SessionStats& out) const noexcept;
  [[nodiscard]] std::string_view view_reference() const noexcept;
  void reset() noexcept;
 private:
  std::unique_ptr<detail::SessionState> impl_;
};

} // namespace symphony::sqdv
#endif
