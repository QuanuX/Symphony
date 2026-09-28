#include <algorithm>
#include <array>
#include <mutex>
#include <new>
#include <symphony/sqdv/checkpoint.hpp>
#include <unistd.h>
namespace symphony::sqdv {
namespace {
constexpr std::uint64_t frame_limit = 16384;
bool valid(const Checkpoint &c) noexcept {
  return !c.view_reference.empty() && c.view_reference.size() <= 128 &&
         (!c.sequence_exhausted || c.next_sequence == UINT64_MAX);
}
bool equal(const Checkpoint &a, const Checkpoint &b) noexcept {
  return a.view_reference == b.view_reference &&
         a.next_sequence == b.next_sequence &&
         a.sequence_exhausted == b.sequence_exhausted;
}
bool advances(const Checkpoint &a, const Checkpoint &b) noexcept {
  return a.view_reference == b.view_reference && !a.sequence_exhausted &&
         (b.next_sequence > a.next_sequence ||
          (a.next_sequence == UINT64_MAX && b.next_sequence == UINT64_MAX &&
           b.sequence_exhausted));
}
Status mapped(sqpv::Status s) noexcept {
  switch (s) {
#define MAP(x)                                                                 \
  case sqpv::Status::x:                                                        \
    return Status::x
    MAP(ok);
    MAP(duplicate);
    MAP(invalid_argument);
    MAP(binding_mismatch);
    MAP(limit);
    MAP(busy);
    MAP(conflict);
    MAP(gap);
    MAP(stale);
    MAP(missing);
    MAP(corrupt);
    MAP(unsafe_path);
    MAP(io_error);
    MAP(outcome_uncertain);
    MAP(closed);
    MAP(no_memory);
    MAP(unsupported);
#undef MAP
  }
  return Status::internal_error;
}
Status flow(sqfv::Status s) noexcept {
  if (s == sqfv::Status::ok)
    return Status::ok;
  if (s == sqfv::Status::no_memory)
    return Status::no_memory;
  if (s == sqfv::Status::limit)
    return Status::limit;
  return Status::internal_error;
}
Status metadata(sqmv::Status s) noexcept {
  if (s == sqmv::Status::ok)
    return Status::ok;
  if (s == sqmv::Status::no_memory)
    return Status::no_memory;
  if (s == sqmv::Status::limit)
    return Status::limit;
  return Status::invalid_argument;
}
std::array<std::uint8_t, 13> encoded(const Checkpoint &c) noexcept {
  std::array<std::uint8_t, 13> b{'S', 'Q', 'C', 1};
  for (unsigned i = 0; i < 8; ++i)
    b[4 + i] = static_cast<std::uint8_t>(c.next_sequence >> (56 - 8 * i));
  b[12] = c.sequence_exhausted;
  return b;
}
Status decode(sqfv::ByteView b, Checkpoint &c) noexcept {
  if (b.size() != 13 || b[0] != 'S' || b[1] != 'Q' || b[2] != 'C' ||
      b[3] != 1 || b[12] > 1)
    return Status::corrupt;
  c.next_sequence = 0;
  for (unsigned i = 0; i < 8; ++i)
    c.next_sequence = (c.next_sequence << 8) | b[4 + i];
  c.sequence_exhausted = b[12] != 0;
  return valid(c) ? Status::ok : Status::corrupt;
}
} // namespace
namespace detail {
struct CheckpointJournalState {
  pid_t pid = ::getpid();
  mutable std::mutex mutex;
  sqpv::Store store;
  sqfv::Context context;
  sqmv::Manifest manifest;
  sqfv::Descriptor descriptor;
  Checkpoint initial, latest;
  std::uint64_t entries = 0;
  std::uint32_t maximum = 0;
  bool uncertain = false;
  Status prepare(const Checkpoint &c, std::uint64_t sequence,
                 sqfv::Batch &batch) {
    descriptor.batch_sequence = sequence;
    const auto bytes = encoded(c);
    return flow(context.prepare_copy(descriptor, bytes, batch));
  }
};
} // namespace detail
namespace {
Status initialize(bool create, const std::string &root,
                  const Checkpoint &initial, const CheckpointOptions &options,
                  std::unique_ptr<detail::CheckpointJournalState> &out) {
  if (!valid(initial) || options.max_checkpoints == 0 ||
      options.max_checkpoints > 65536 ||
      !std::any_of(options.generation.begin(), options.generation.end(),
                   [](auto b) { return b != 0; }) ||
      options.max_store_bytes == 0)
    return Status::invalid_argument;
  auto s = std::make_unique<detail::CheckpointJournalState>();
  s->initial = initial;
  s->latest = initial;
  s->maximum = options.max_checkpoints;
  auto status = flow(
      sqfv::Context::create({8192, frame_limit, 8192, 131072, 1}, s->context));
  if (status != Status::ok)
    return status;
  sqmv::Description d{
      initial.view_reference,
      "baseline:" + std::to_string(initial.next_sequence) +
          (initial.sequence_exhausted ? ":exhausted" : ":open"),
      "sqdv-checkpoint-v1",
      "sqdv-checkpoint-v1",
      "private-checkpoint:" + initial.view_reference,
      "sqdv-delivery-cpp",
      {{sqmv::EvidenceRole::schema, "sqdv", "sqdv-checkpoint-v1"},
       {sqmv::EvidenceRole::layout, "sqdv", "sqdv-checkpoint-v1"},
       {sqmv::EvidenceRole::access, "sqdv", initial.view_reference},
       {sqmv::EvidenceRole::source, "sqdv", initial.view_reference}}};
  status = metadata(sqmv::Manifest::create(d, {65536, 4096, 8}, s->manifest));
  if (status != Status::ok)
    return status;
  status = metadata(s->manifest.binding(s->descriptor.binding));
  if (status != Status::ok)
    return status;
  s->descriptor.partition = "checkpoints";
  s->descriptor.producer_generation = options.generation;
  s->descriptor.source_binding = initial.view_reference;
  s->descriptor.record_count = 1;
  sqpv::Options stored{
      "checkpoints",
      options.generation,
      options.generation,
      1,
      {frame_limit, options.max_store_bytes, options.max_checkpoints}};
  // Prepare baseline and all persistent state before create/open may mutate.
  sqfv::Batch baseline;
  status = s->prepare(initial, 1, baseline);
  if (status != Status::ok)
    return status;
  status =
      mapped(create ? sqpv::Store::create(root, s->manifest, stored, s->store)
                    : sqpv::Store::open(root, s->manifest, stored, s->store));
  if (status != Status::ok)
    return status;
  sqpv::Snapshot snapshot;
  status = mapped(s->store.snapshot(snapshot));
  if (status != Status::ok)
    return status;
  if (snapshot.committed_batches == 0) {
    sqpv::Receipt receipt;
    status = mapped(s->store.append(s->context, baseline, receipt));
    if (status != Status::ok)
      return status;
    s->entries = 1;
  } else {
    baseline = {};
    auto previous = initial;
    for (std::uint64_t i = 1; i <= snapshot.committed_batches; ++i) {
      sqfv::Batch batch;
      sqpv::Receipt receipt;
      status = mapped(s->store.read(i, s->context, batch, receipt));
      if (status != Status::ok)
        return status;
      const auto &bd = batch.descriptor();
      if (bd.record_count != 1 || bd.source_binding != initial.view_reference ||
          !bd.source_position.empty())
        return Status::corrupt;
      sqfv::Lease lease;
      status = flow(batch.acquire(s->descriptor.binding.access_scope, lease));
      if (status != Status::ok)
        return status;
      Checkpoint c = initial;
      status = decode(lease.payload(), c);
      if (status != Status::ok)
        return status;
      if ((i == 1 && !equal(c, initial)) || (i > 1 && !advances(previous, c)))
        return Status::corrupt;
      previous = std::move(c);
    }
    s->latest = std::move(previous);
    s->entries = snapshot.committed_batches;
  }
  out = std::move(s);
  return Status::ok;
}
} // namespace
CheckpointJournal::CheckpointJournal() noexcept = default;
CheckpointJournal::~CheckpointJournal() noexcept = default;
CheckpointJournal::CheckpointJournal(CheckpointJournal &&) noexcept = default;
CheckpointJournal &
CheckpointJournal::operator=(CheckpointJournal &&) noexcept = default;
void CheckpointJournal::reset() noexcept { state_.reset(); }
Status CheckpointJournal::create(const std::string &root,
                                 const Session &baseline,
                                 const CheckpointOptions &options,
                                 CheckpointJournal &out) noexcept {
  Checkpoint initial;
  auto status = baseline.checkpoint(initial);
  if (status != Status::ok)
    return status;
  try {
    CheckpointJournal ready;
    status = initialize(true, root, initial, options, ready.state_);
    if (status == Status::ok)
      out = std::move(ready);
    return status;
  } catch (const std::bad_alloc &) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}
Status CheckpointJournal::open(const std::string &root,
                               const Checkpoint &baseline,
                               const CheckpointOptions &options,
                               CheckpointJournal &out) noexcept {
  try {
    CheckpointJournal ready;
    auto status = initialize(false, root, baseline, options, ready.state_);
    if (status == Status::ok)
      out = std::move(ready);
    return status;
  } catch (const std::bad_alloc &) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}
Status CheckpointJournal::load(Checkpoint &out) const noexcept {
  if (!state_)
    return Status::closed;
  if (state_->pid != ::getpid())
    return Status::stale;
  try {
    std::lock_guard lock(state_->mutex);
    if (state_->uncertain)
      return Status::closed;
    auto ready = state_->latest;
    out = std::move(ready);
    return Status::ok;
  } catch (const std::bad_alloc &) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}
Status CheckpointJournal::save(const Session &session,
                               Checkpoint &out) noexcept {
  return save_impl(session, false, out);
}
Status CheckpointJournal::save_retained(const Session &session,
                                        Checkpoint &out) noexcept {
  return save_impl(session, true, out);
}
Status CheckpointJournal::save_impl(const Session &session, bool retained,
                                    Checkpoint &out) noexcept {
  if (!state_)
    return Status::closed;
  if (state_->pid != ::getpid())
    return Status::stale;
  try {
    std::lock_guard lock(state_->mutex);
    auto &s = *state_;
    if (s.uncertain)
      return Status::closed;
    Checkpoint candidate;
    auto status = retained ? session.checkpoint_for_replay(candidate)
                           : session.checkpoint(candidate);
    if (status != Status::ok)
      return status;
    if (candidate.view_reference != s.initial.view_reference)
      return Status::binding_mismatch;
    if (equal(candidate, s.latest)) {
      out = std::move(candidate);
      return Status::duplicate;
    }
    if (!advances(s.latest, candidate))
      return Status::stale;
    if (s.entries >= s.maximum)
      return Status::limit;
    auto output = candidate;
    sqfv::Batch batch;
    status = s.prepare(candidate, s.entries + 1, batch);
    if (status != Status::ok)
      return status;
    sqpv::Receipt receipt;
    status = mapped(s.store.append(s.context, batch, receipt));
    if (status == Status::outcome_uncertain)
      s.uncertain = true;
    if (status != Status::ok)
      return status;
    // No allocation remains after the owner confirms the durable append.
    s.latest = std::move(candidate);
    ++s.entries;
    out = std::move(output);
    return Status::ok;
  } catch (const std::bad_alloc &) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}
} // namespace symphony::sqdv
