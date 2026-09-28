#ifndef SYMPHONY_SQDV_CHECKPOINT_HPP
#define SYMPHONY_SQDV_CHECKPOINT_HPP
#include <symphony/sqdv/delivery.hpp>
namespace symphony::sqdv {
struct CheckpointOptions {
  sqfv::Generation generation{};
  std::uint32_t max_checkpoints = 0; // Includes baseline; 1..65,536.
  std::uint64_t max_store_bytes = 0; // SQPV logical disk budget.
};
namespace detail {
struct CheckpointJournalState;
}
// Exact-view, monotone, bounded checkpoint journal on the existing SQPV store.
// Persists processing acknowledgments; it does not prove destination commits.
class CheckpointJournal final {
public:
  CheckpointJournal() noexcept;
  ~CheckpointJournal() noexcept;
  CheckpointJournal(CheckpointJournal &&) noexcept;
  CheckpointJournal &operator=(CheckpointJournal &&) noexcept;
  CheckpointJournal(const CheckpointJournal &) = delete;
  CheckpointJournal &operator=(const CheckpointJournal &) = delete;
  [[nodiscard]] static Status create(const std::string &root,
                                     const Session &baseline,
                                     const CheckpointOptions &,
                                     CheckpointJournal &out) noexcept;
  // Pass the original baseline, obtainable from a fresh exact-view session.
  // A store initialized before a failed baseline append is recovered
  // explicitly.
  [[nodiscard]] static Status open(const std::string &root,
                                   const Checkpoint &baseline,
                                   const CheckpointOptions &,
                                   CheckpointJournal &out) noexcept;
  [[nodiscard]] Status load(Checkpoint &out) const noexcept;
  // Reads actual session progress. Duplicate returns the durable checkpoint;
  // other failures preserve out. Uncertain writes require reset/open.
  [[nodiscard]] Status save(const Session &, Checkpoint &out) noexcept;
  // Also requires the actual retained source to cover processed progress.
  [[nodiscard]] Status save_retained(const Session &, Checkpoint &out) noexcept;
  void reset() noexcept;

private:
  [[nodiscard]] Status save_impl(const Session &, bool retained,
                                 Checkpoint &out) noexcept;
  std::unique_ptr<detail::CheckpointJournalState> state_;
};
} // namespace symphony::sqdv
#endif
