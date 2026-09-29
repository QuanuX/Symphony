#ifndef SYMPHONY_SQPV_LOCAL_STORE_HPP
#define SYMPHONY_SQPV_LOCAL_STORE_HPP

#include <cstdint>
#include <memory>
#include <string>
#include <symphony/sqfv/batch.hpp>
#include <symphony/sqmv/metadata.hpp>

namespace symphony::sqpv {

enum class Status : std::uint8_t {
  ok,
  duplicate,
  invalid_argument,
  binding_mismatch,
  limit,
  busy,
  conflict,
  gap,
  stale,
  missing,
  corrupt,
  unsafe_path,
  io_error,
  outcome_uncertain,
  closed,
  no_memory,
  unsupported,
};
struct Limits {
  std::uint64_t max_frame_bytes = 0;
  // Total logical bytes of all module files, including bounded staging.
  std::uint64_t max_store_bytes = 0;
  std::uint32_t max_batches = 0;
};
struct Options {
  std::string partition;
  sqfv::Generation producer_generation{};
  // Immutable store identity; exclusive lock ownership supplies writer fencing.
  sqfv::Generation store_generation{};
  std::uint64_t first_sequence = 0;
  Limits limits;
};
struct Receipt {
  sqfv::Generation store_generation{};
  sqfv::Generation producer_generation{};
  std::uint64_t batch_sequence = 0;
  std::uint64_t frame_bytes = 0;
  sqfv::ContentId content_id{};
  std::string frame_sha256;
  std::string commit_sha256;
};
struct Snapshot {
  std::uint64_t committed_batches = 0;
  std::uint64_t next_sequence = 0;
  std::uint64_t store_bytes = 0;
  bool sequence_exhausted = false;
  bool staged = false;
};
namespace detail {
struct StoreHandle;
}

// Native C++26. One exact manifest/partition/producer generation per root.
// Operations are serialized within a handle. Destruction/move must not race
// operations. Handles cannot be used after fork. Root must be an existing,
// private, empty directory for create; open never silently creates a store.
class Store final {
public:
  Store() noexcept;
  ~Store() noexcept;
  Store(Store &&) noexcept;
  Store &operator=(Store &&) noexcept;
  Store(const Store &) = delete;
  Store &operator=(const Store &) = delete;
  [[nodiscard]] explicit operator bool() const noexcept;
  [[nodiscard]] static Status create(const std::string &absolute_root,
                                     const sqmv::Manifest &, const Options &,
                                     Store &out) noexcept;
  [[nodiscard]] static Status open(const std::string &absolute_root,
                                   const sqmv::Manifest &, const Options &,
                                   Store &out) noexcept;
  // `duplicate` returns the verified original receipt. Other failures preserve
  // outputs. outcome_uncertain closes the handle to further work until reopen.
  [[nodiscard]] Status append(sqfv::Context &, const sqfv::Batch &,
                              Receipt &out) noexcept;
  [[nodiscard]] Status read(std::uint64_t sequence, sqfv::Context &,
                            sqfv::Batch &out, Receipt &receipt) noexcept;
  [[nodiscard]] Status snapshot(Snapshot &out) noexcept;
  void reset() noexcept;

private:
  std::unique_ptr<detail::StoreHandle> impl_;
};
} // namespace symphony::sqpv
#endif
