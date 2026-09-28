#ifndef SYMPHONY_SQPV_ASYNC_STORE_HPP
#define SYMPHONY_SQPV_ASYNC_STORE_HPP
#include <symphony/sqpv/local_store.hpp>

namespace symphony::sqpv {
struct AsyncLimits {
  std::uint32_t max_pending_batches = 0; // Includes the active write; max 65,536.
  std::uint64_t max_pending_frame_bytes = 0; // Includes active write; max 64 MiB.
};
struct AsyncSnapshot {
  Snapshot confirmed; // Last successful append/open; never queued progress.
  std::uint64_t next_admission_sequence = 0;
  std::uint64_t pending_frame_bytes = 0;
  std::uint32_t pending_batches = 0;
  Status failure = Status::ok; // First write failure stops this writer.
  bool admission_exhausted = false;
  bool accepting = false;
};
namespace detail { struct AsyncStoreState; }
// One owned background writer, exact store identity and retained SQFV context.
// submit success is RAM queue admission only. Destruction discards unwritten
// work and joins any active write. Call finish explicitly to drain and inspect
// failure. No cancellation of an in-flight filesystem call or bounded join time
// is claimed. Calls/moves/destruction after fork are unsupported except that
// status-returning calls reject inherited handles before touching a mutex.
class AsyncStore final {
 public:
  AsyncStore() noexcept;
  ~AsyncStore() noexcept;
  AsyncStore(AsyncStore&&) noexcept;
  AsyncStore& operator=(AsyncStore&&) noexcept;
  AsyncStore(const AsyncStore&) = delete;
  AsyncStore& operator=(const AsyncStore&) = delete;
  [[nodiscard]] explicit operator bool() const noexcept;
  [[nodiscard]] static Status create(const std::string&, const sqmv::Manifest&,
      const Options&, sqfv::Context&, const AsyncLimits&, AsyncStore& out) noexcept;
  [[nodiscard]] static Status open(const std::string&, const sqmv::Manifest&,
      const Options&, sqfv::Context&, const AsyncLimits&, AsyncStore& out) noexcept;
  [[nodiscard]] Status submit(const sqfv::Batch&) noexcept;
  [[nodiscard]] Status snapshot(AsyncSnapshot& out) const noexcept;
  // Stops admission, drains all accepted work or returns the first failure.
  [[nodiscard]] Status finish() noexcept;
  // Reads only confirmed storage; serializes with the actual SQPV Store writer.
  [[nodiscard]] Status read(std::uint64_t, sqfv::Context&, sqfv::Batch&, Receipt&) noexcept;
  void reset() noexcept;
 private:
  std::unique_ptr<detail::AsyncStoreState> state_;
};
} // namespace symphony::sqpv
#endif
