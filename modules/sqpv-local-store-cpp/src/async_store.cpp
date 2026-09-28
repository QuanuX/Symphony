#include <symphony/sqpv/async_store.hpp>
#include <condition_variable>
#include <mutex>
#include <new>
#include <thread>
#include <unistd.h>
#include <vector>

namespace symphony::sqpv {
namespace {
Status flow(sqfv::Status s) noexcept {
  switch (s) {
    case sqfv::Status::ok: return Status::ok;
    case sqfv::Status::no_memory: return Status::no_memory;
    case sqfv::Status::limit: return Status::limit;
    default: return Status::invalid_argument;
  }
}
bool same(const sqfv::Binding& a, const sqfv::Binding& b) noexcept {
  return a.metadata_ref == b.metadata_ref && a.dataset_revision == b.dataset_revision &&
    a.schema_version == b.schema_version && a.layout_version == b.layout_version && a.access_scope == b.access_scope;
}
}
namespace detail {
struct AsyncEntry { sqfv::Batch batch; std::uint64_t bytes = 0; };
struct AsyncStoreState {
  const pid_t pid = ::getpid();
  mutable std::mutex mutex;
  std::condition_variable changed;
  sqfv::Context context;
  sqfv::Binding binding;
  Options options;
  AsyncLimits limits;
  Store store;
  std::vector<AsyncEntry> entries;
  std::uint32_t head = 0;
  AsyncSnapshot status;
  bool started = false;
  bool stop = false;
  bool active = false;
  std::thread worker;
  ~AsyncStoreState() {
    // A fork child must exec/_exit; touching inherited thread/mutex state is unsafe.
    if (pid != ::getpid()) std::terminate();
    { std::lock_guard lock(mutex); stop = true; status.accepting = false; }
    changed.notify_all();
    if (worker.joinable()) worker.join();
  }
  void run() noexcept {
    std::unique_lock lock(mutex);
    for (;;) {
      changed.wait(lock, [&] { return stop || (started && status.pending_batches && status.failure == Status::ok); });
      if (stop) return;
      auto& entry = entries[head];
      active = true;
      lock.unlock();
      Receipt receipt;
      const auto result = store.append(context, entry.batch, receipt);
      Snapshot confirmed;
      const auto observed = result == Status::ok || result == Status::duplicate
          ? store.snapshot(confirmed) : result;
      lock.lock();
      active = false;
      if (observed != Status::ok) {
        status.failure = observed;
        status.accepting = false;
        changed.notify_all();
        continue;
      }
      status.confirmed = confirmed;
      status.pending_frame_bytes -= entry.bytes;
      entry.batch = {}; entry.bytes = 0;
      head = (head + 1) % limits.max_pending_batches;
      --status.pending_batches;
      changed.notify_all();
    }
  }
};
}
namespace {
Status make(bool create, const std::string& root, const sqmv::Manifest& manifest,
            const Options& options, sqfv::Context& context, const AsyncLimits& limits,
            std::unique_ptr<detail::AsyncStoreState>& out) {
  if (!manifest || !context || !limits.max_pending_batches || limits.max_pending_batches > 65536 ||
      !limits.max_pending_frame_bytes || limits.max_pending_frame_bytes > (64ULL << 20)) return Status::invalid_argument;
  auto state = std::make_unique<detail::AsyncStoreState>();
  state->options = options; state->limits = limits;
  state->entries.resize(limits.max_pending_batches);
  auto bound = manifest.binding(state->binding);
  if (bound != sqmv::Status::ok) return bound == sqmv::Status::no_memory ? Status::no_memory : Status::invalid_argument;
  if (auto s = flow(context.retain(state->context)); s != Status::ok) return s;
  // Allocate/start the dormant thread before any persistent create/open effects.
  state->worker = std::thread([p = state.get()] { p->run(); });
  auto result = create ? Store::create(root, manifest, options, state->store)
                       : Store::open(root, manifest, options, state->store);
  if (result != Status::ok) return result;
  if (auto s = state->store.snapshot(state->status.confirmed); s != Status::ok) return s;
  state->status.next_admission_sequence = state->status.confirmed.next_sequence;
  state->status.admission_exhausted = state->status.confirmed.sequence_exhausted;
  { std::lock_guard lock(state->mutex); state->started = true; state->status.accepting = true; }
  out = std::move(state);
  return Status::ok;
}
}
AsyncStore::AsyncStore() noexcept = default;
AsyncStore::~AsyncStore() noexcept = default;
AsyncStore::AsyncStore(AsyncStore&&) noexcept = default;
AsyncStore& AsyncStore::operator=(AsyncStore&&) noexcept = default;
AsyncStore::operator bool() const noexcept { return bool(state_); }
void AsyncStore::reset() noexcept { state_.reset(); }
Status AsyncStore::create(const std::string& r, const sqmv::Manifest& m, const Options& o,
                          sqfv::Context& c, const AsyncLimits& l, AsyncStore& out) noexcept {
  try { AsyncStore ready; auto s = make(true, r, m, o, c, l, ready.state_);
    if (s == Status::ok) out = std::move(ready); return s;
  } catch (const std::bad_alloc&) { return Status::no_memory; }
    catch (...) { return Status::io_error; }
}
Status AsyncStore::open(const std::string& r, const sqmv::Manifest& m, const Options& o,
                        sqfv::Context& c, const AsyncLimits& l, AsyncStore& out) noexcept {
  try { AsyncStore ready; auto s = make(false, r, m, o, c, l, ready.state_);
    if (s == Status::ok) out = std::move(ready); return s;
  } catch (const std::bad_alloc&) { return Status::no_memory; }
    catch (...) { return Status::io_error; }
}
Status AsyncStore::submit(const sqfv::Batch& batch) noexcept {
  if (!state_) return Status::closed;
  auto& s = *state_;
  if (s.pid != ::getpid()) return Status::stale;
  if (!batch) return Status::invalid_argument;
  try {
    const auto& d = batch.descriptor();
    if (!same(d.binding, s.binding) || d.partition != s.options.partition ||
        d.producer_generation != s.options.producer_generation) return Status::binding_mismatch;
    std::size_t bytes = 0;
    if (auto result = flow(sqfv::frame_measure(s.context, batch, bytes)); result != Status::ok) return result;
    if (bytes > s.options.limits.max_frame_bytes || bytes > s.limits.max_pending_frame_bytes) return Status::limit;
    std::lock_guard lock(s.mutex);
    if (s.status.failure != Status::ok) return s.status.failure;
    if (!s.status.accepting) return Status::closed;
    // All queued positions remain bounded and permit exact retry without enqueue.
    for (std::uint32_t i = 0; i < s.status.pending_batches; ++i) {
      const auto& prior = s.entries[(s.head + i) % s.limits.max_pending_batches].batch;
      if (prior.descriptor().batch_sequence == d.batch_sequence)
        return prior.content_id() == batch.content_id() ? Status::duplicate : Status::conflict;
    }
    if (s.status.admission_exhausted || d.batch_sequence < s.status.next_admission_sequence) return Status::stale;
    if (d.batch_sequence > s.status.next_admission_sequence) return Status::gap;
    if (s.status.pending_batches == s.limits.max_pending_batches ||
        bytes > s.limits.max_pending_frame_bytes - s.status.pending_frame_bytes) return Status::busy;
    sqfv::Batch retained;
    if (auto result = flow(batch.retain(retained)); result != Status::ok) return result;
    auto& entry = s.entries[(s.head + s.status.pending_batches) % s.limits.max_pending_batches];
    entry.batch = std::move(retained); entry.bytes = bytes;
    ++s.status.pending_batches; s.status.pending_frame_bytes += bytes;
    if (d.batch_sequence == UINT64_MAX) s.status.admission_exhausted = true;
    else ++s.status.next_admission_sequence;
    s.changed.notify_all();
    return Status::ok;
  } catch (const std::bad_alloc&) { return Status::no_memory; }
    catch (...) { return Status::io_error; }
}
Status AsyncStore::snapshot(AsyncSnapshot& out) const noexcept {
  if (!state_) return Status::closed;
  if (state_->pid != ::getpid()) return Status::stale;
  try { std::lock_guard lock(state_->mutex); out = state_->status; return Status::ok; }
  catch (...) { return Status::io_error; }
}
Status AsyncStore::finish() noexcept {
  if (!state_) return Status::closed;
  auto& s = *state_;
  if (s.pid != ::getpid()) return Status::stale;
  try {
    std::unique_lock lock(s.mutex); s.status.accepting = false;
    s.changed.wait(lock, [&] { return !s.status.pending_batches || s.status.failure != Status::ok; });
    return s.status.failure;
  } catch (...) { return Status::io_error; }
}
Status AsyncStore::read(std::uint64_t seq, sqfv::Context& context, sqfv::Batch& batch, Receipt& receipt) noexcept {
  if (!state_) return Status::closed;
  if (state_->pid != ::getpid()) return Status::stale;
  return state_->store.read(seq, context, batch, receipt);
}
} // namespace symphony::sqpv
