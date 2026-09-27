#include <symphony/sqdv/delivery.hpp>
#include <symphony/knowledge/engine/digest.hpp>

#include <algorithm>
#include <mutex>
#include <new>
#include <stdexcept>
#include <unistd.h>
#include <utility>
#include <vector>

namespace symphony::sqdv {
namespace {
constexpr std::size_t maximum_string = 4096;
constexpr std::size_t maximum_identity = 65536;
constexpr std::uint64_t maximum_credit = 64ULL << 20;
constexpr std::uint32_t maximum_entries = 65536;

bool valid_string(std::string_view value) noexcept {
  return !value.empty() && value.size() <= maximum_string;
}
bool nonzero(const sqfv::Generation& value) noexcept {
  return std::any_of(value.begin(), value.end(), [](auto byte) { return byte != 0; });
}
bool same_binding(const sqfv::Binding& left, const sqfv::Binding& right) noexcept {
  return left.metadata_ref == right.metadata_ref && left.dataset_revision == right.dataset_revision &&
      left.schema_version == right.schema_version && left.layout_version == right.layout_version &&
      left.access_scope == right.access_scope;
}
Status flow_status(sqfv::Status value) noexcept {
  switch (value) {
    case sqfv::Status::ok: return Status::ok;
    case sqfv::Status::invalid_argument: return Status::invalid_argument;
    case sqfv::Status::limit: return Status::limit;
    case sqfv::Status::no_memory: return Status::no_memory;
    case sqfv::Status::blocked: return Status::blocked;
    case sqfv::Status::duplicate: return Status::duplicate;
    case sqfv::Status::conflict: return Status::conflict;
    case sqfv::Status::gap: return Status::gap;
    case sqfv::Status::stale: return Status::stale;
    case sqfv::Status::scope_mismatch:
    case sqfv::Status::binding_mismatch: return Status::binding_mismatch;
    case sqfv::Status::closed: return Status::closed;
    case sqfv::Status::empty: return Status::empty;
    case sqfv::Status::corrupt_frame: return Status::corrupt;
    case sqfv::Status::unsupported_frame: return Status::unsupported;
    case sqfv::Status::internal_error: return Status::internal_error;
  }
  return Status::internal_error;
}
Status metadata_status(sqmv::Status value) noexcept {
  switch (value) {
    case sqmv::Status::ok: return Status::ok;
    case sqmv::Status::invalid_argument: return Status::invalid_argument;
    case sqmv::Status::limit: return Status::limit;
    case sqmv::Status::no_memory: return Status::no_memory;
    case sqmv::Status::binding_mismatch:
    case sqmv::Status::reference_mismatch: return Status::binding_mismatch;
    case sqmv::Status::corrupt_manifest: return Status::corrupt;
    case sqmv::Status::unsupported_manifest: return Status::unsupported;
    case sqmv::Status::internal_error: return Status::internal_error;
  }
  return Status::internal_error;
}
Status store_status(sqpv::Status value) noexcept {
  switch (value) {
    case sqpv::Status::ok: return Status::ok;
    case sqpv::Status::duplicate: return Status::duplicate;
    case sqpv::Status::invalid_argument: return Status::invalid_argument;
    case sqpv::Status::binding_mismatch: return Status::binding_mismatch;
    case sqpv::Status::limit: return Status::limit;
    case sqpv::Status::busy: return Status::busy;
    case sqpv::Status::conflict: return Status::conflict;
    case sqpv::Status::gap: return Status::gap;
    case sqpv::Status::stale: return Status::stale;
    case sqpv::Status::missing: return Status::missing;
    case sqpv::Status::corrupt: return Status::corrupt;
    case sqpv::Status::unsafe_path: return Status::unsafe_path;
    case sqpv::Status::io_error: return Status::io_error;
    case sqpv::Status::outcome_uncertain: return Status::outcome_uncertain;
    case sqpv::Status::closed: return Status::closed;
    case sqpv::Status::no_memory: return Status::no_memory;
    case sqpv::Status::unsupported: return Status::unsupported;
  }
  return Status::internal_error;
}
template <typename Integer>
void integer(std::string& out, Integer value) {
  for (std::size_t index = 0; index < sizeof(Integer); ++index)
    out.push_back(static_cast<char>(value >> ((sizeof(Integer) - 1 - index) * 8)));
}
void field(std::string& out, std::string_view value) {
  integer(out, static_cast<std::uint32_t>(value.size()));
  out.append(value);
}
void generation(std::string& out, const sqfv::Generation& value) {
  out.append(reinterpret_cast<const char*>(value.data()), value.size());
}
void advance(std::uint64_t& next, bool& exhausted) noexcept {
  if (next == UINT64_MAX) exhausted = true;
  else ++next;
}
}

namespace detail {
struct SourceIdentity {
  pid_t pid = ::getpid();
  std::string absolute_root;
  sqpv::Options options;
  sqfv::Binding binding;
  std::string canonical;
};
struct SourceState {
  std::shared_ptr<const SourceIdentity> identity;
  std::mutex mutex;
  sqpv::Store store;
  bool uncertain = false;
};
struct RetainedBatchState {
  std::shared_ptr<const SourceIdentity> identity;
  sqfv::Batch batch;
  sqpv::Receipt receipt;
};
struct SessionIdentity {
  pid_t pid = ::getpid();
};
struct DeliveryHandle {
  std::shared_ptr<const SessionIdentity> identity;
  sqfv::Lease lease;
  sqfv::ContentId content_id{};
  std::uint64_t sequence = 0;
  Origin origin = Origin::live;
  bool has_receipt = false;
  sqpv::Receipt receipt;
};
struct Entry {
  std::uint64_t sequence = 0;
  sqfv::ContentId content_id{};
  bool delivered = false;
  std::unique_ptr<DeliveryHandle> pending;
};
struct SessionState {
  std::shared_ptr<const SessionIdentity> identity;
  std::shared_ptr<SourceState> source;
  Config config;
  Limits limits;
  sqfv::Binding binding;
  std::string reference;
  mutable std::mutex mutex;
  sqfv::Port port;
  std::vector<Entry> ledger;
  std::uint32_t head = 0;
  std::uint32_t used = 0;
  std::uint32_t delivered = 0;
  std::uint64_t next_offer = 0;
  std::uint64_t next_processed = 0;
  bool offer_exhausted = false;
  bool processed_exhausted = false;
  bool has_last_offer = false;
  std::uint64_t last_offer = 0;
  sqfv::ContentId last_offer_id{};
  bool has_last_ack = false;
  std::uint64_t last_ack = 0;
  sqfv::ContentId last_ack_id{};

  Status validate_batch(const sqfv::Batch& batch) const noexcept {
    if (!batch) return Status::invalid_argument;
    const auto& descriptor = batch.descriptor();
    if (!same_binding(binding, descriptor.binding) || descriptor.partition != config.partition ||
        descriptor.producer_generation != config.producer_generation) return Status::binding_mismatch;
    return Status::ok;
  }
  // Caller holds the session mutex; all fallible preparations precede offer.
  Status enqueue(const sqfv::Batch& batch, Origin origin, const sqpv::Receipt* receipt) {
    if (const auto status = validate_batch(batch); status != Status::ok) return status;
    const auto sequence = batch.descriptor().batch_sequence;
    if (has_last_offer && sequence == last_offer)
      return flow_status(port.offer(batch)); // SQFV verifies context and exact retry.
    if (offer_exhausted) return Status::limit;
    if (sequence < next_offer) return Status::stale;
    if (sequence > next_offer) return Status::gap;
    if (used == ledger.size()) return Status::blocked;
    auto ready = std::make_unique<DeliveryHandle>();
    ready->identity = identity;
    ready->sequence = sequence;
    ready->content_id = batch.content_id();
    ready->origin = origin;
    if (receipt) {
      ready->receipt = *receipt;
      ready->has_receipt = true;
    }
    const auto accepted = port.offer(batch);
    if (accepted != sqfv::Status::ok) return flow_status(accepted);
    auto& entry = ledger[(static_cast<std::size_t>(head) + used) % ledger.size()];
    entry.sequence = sequence;
    entry.content_id = batch.content_id();
    entry.delivered = false;
    entry.pending = std::move(ready);
    ++used;
    has_last_offer = true;
    last_offer = sequence;
    last_offer_id = batch.content_id();
    advance(next_offer, offer_exhausted);
    return Status::ok;
  }
};
}

namespace {
Status make_source(bool create, const std::string& root, const sqmv::Manifest& manifest,
                   const sqpv::Options& options, std::shared_ptr<detail::SourceState>& out) {
  if (!manifest || !valid_string(root) || !valid_string(options.partition)) return Status::invalid_argument;
  auto identity = std::make_shared<detail::SourceIdentity>();
  identity->absolute_root = root;
  identity->options = options;
  const auto bound = manifest.binding(identity->binding);
  if (bound != sqmv::Status::ok) return metadata_status(bound);
  constexpr char domain[] = "symphony.sqdv.retained-source.v1";
  identity->canonical.assign(domain, sizeof(domain));
  field(identity->canonical, root);
  field(identity->canonical, manifest.reference());
  field(identity->canonical, options.partition);
  generation(identity->canonical, options.producer_generation);
  generation(identity->canonical, options.store_generation);
  integer(identity->canonical, options.first_sequence);
  integer(identity->canonical, options.limits.max_frame_bytes);
  integer(identity->canonical, options.limits.max_store_bytes);
  integer(identity->canonical, options.limits.max_batches);
  if (identity->canonical.size() > maximum_identity) return Status::limit;
  auto state = std::make_shared<detail::SourceState>();
  state->identity = std::move(identity);
  // All wrapper-owned allocations are complete before create/open can mutate.
  const auto status = create ? sqpv::Store::create(root, manifest, options, state->store)
                             : sqpv::Store::open(root, manifest, options, state->store);
  if (status != sqpv::Status::ok) return store_status(status);
  out = std::move(state);
  return Status::ok;
}
bool valid_config(const Config& config, const Limits& limits) noexcept {
  return valid_string(config.view_id) && valid_string(config.recipient_id) &&
      valid_string(config.recipient_interface) && valid_string(config.partition) &&
      nonzero(config.producer_generation) &&
      (config.profile == Profile::disposable || config.profile == Profile::retained_before_delivery) &&
      limits.outstanding_byte_credit > 0 && limits.outstanding_byte_credit <= maximum_credit &&
      limits.max_unacknowledged_batches > 0 && limits.max_unacknowledged_batches <= maximum_entries;
}
std::string view_reference(const sqfv::Binding& binding, const Config& config,
                           const detail::SourceIdentity* source) {
  constexpr char domain[] = "symphony.sqdv.delivery-view.v1";
  std::string canonical(domain, sizeof(domain));
  canonical.push_back(static_cast<char>(config.profile));
  field(canonical, binding.metadata_ref);
  field(canonical, config.view_id);
  field(canonical, config.recipient_id);
  field(canonical, config.recipient_interface);
  field(canonical, config.partition);
  generation(canonical, config.producer_generation);
  integer(canonical, config.first_sequence);
  field(canonical, source ? std::string_view(source->canonical) : std::string_view{});
  if (canonical.size() > maximum_identity) throw std::length_error("delivery identity exceeds bound");
  auto digest = knowledge::engine::sha256_hex(canonical);
  if (digest.size() != 64) throw std::runtime_error("incomplete SHA-256 digest");
  return "sqdv1-sha256-" + digest;
}
}

RetainedSource::RetainedSource() noexcept = default;
RetainedSource::~RetainedSource() noexcept = default;
RetainedSource::RetainedSource(RetainedSource&&) noexcept = default;
RetainedSource& RetainedSource::operator=(RetainedSource&&) noexcept = default;
RetainedSource::operator bool() const noexcept { return static_cast<bool>(state_); }
void RetainedSource::reset() noexcept { state_.reset(); }
Status RetainedSource::create(const std::string& root, const sqmv::Manifest& manifest,
                              const sqpv::Options& options, RetainedSource& out) noexcept {
  try {
    RetainedSource ready;
    const auto status = make_source(true, root, manifest, options, ready.state_);
    if (status == Status::ok) out = std::move(ready);
    return status;
  } catch (const std::bad_alloc&) { return Status::no_memory; }
    catch (...) { return Status::internal_error; }
}
Status RetainedSource::open(const std::string& root, const sqmv::Manifest& manifest,
                            const sqpv::Options& options, RetainedSource& out) noexcept {
  try {
    RetainedSource ready;
    const auto status = make_source(false, root, manifest, options, ready.state_);
    if (status == Status::ok) out = std::move(ready);
    return status;
  } catch (const std::bad_alloc&) { return Status::no_memory; }
    catch (...) { return Status::internal_error; }
}
Status RetainedSource::retain(RetainedSource& out) const noexcept {
  if (!state_) return Status::invalid_argument;
  if (state_->identity->pid != ::getpid()) return Status::stale;
  try {
    RetainedSource ready;
    {
      std::lock_guard lock(state_->mutex);
      if (state_->uncertain) return Status::closed;
      ready.state_ = state_;
    }
    out = std::move(ready);
    return Status::ok;
  } catch (...) { return Status::internal_error; }
}
Status RetainedSource::commit(sqfv::Context& context, const sqfv::Batch& batch,
                              RetainedBatch& out) noexcept {
  if (!state_) return Status::closed;
  if (state_->identity->pid != ::getpid()) return Status::stale;
  if (!batch) return Status::invalid_argument;
  try {
    RetainedBatch result;
    auto proof = std::make_shared<detail::RetainedBatchState>();
    proof->identity = state_->identity;
    const auto retained = batch.retain(proof->batch);
    if (retained != sqfv::Status::ok) return flow_status(retained);
    Status status;
    {
      std::lock_guard lock(state_->mutex);
      if (state_->uncertain) return Status::closed;
      status = store_status(state_->store.append(context, batch, proof->receipt));
      if (status == Status::outcome_uncertain) state_->uncertain = true;
      if (status != Status::ok && status != Status::duplicate) return status;
      result.state_ = std::move(proof);
    }
    out = std::move(result);
    return status;
  } catch (const std::bad_alloc&) { return Status::no_memory; }
    catch (...) { return Status::internal_error; }
}

RetainedBatch::RetainedBatch() noexcept = default;
RetainedBatch::~RetainedBatch() noexcept = default;
RetainedBatch::RetainedBatch(RetainedBatch&&) noexcept = default;
RetainedBatch& RetainedBatch::operator=(RetainedBatch&&) noexcept = default;
RetainedBatch::operator bool() const noexcept { return static_cast<bool>(state_); }
void RetainedBatch::reset() noexcept { state_.reset(); }
Status RetainedBatch::retain(RetainedBatch& out) const noexcept {
  if (!state_) return Status::invalid_argument;
  if (state_->identity->pid != ::getpid()) return Status::stale;
  RetainedBatch ready;
  ready.state_ = state_;
  out = std::move(ready);
  return Status::ok;
}
const sqfv::Batch& RetainedBatch::batch() const noexcept { return state_->batch; }
const sqpv::Receipt& RetainedBatch::receipt() const noexcept { return state_->receipt; }

Delivery::Delivery() noexcept = default;
Delivery::~Delivery() noexcept = default;
Delivery::Delivery(Delivery&&) noexcept = default;
Delivery& Delivery::operator=(Delivery&&) noexcept = default;
Delivery::operator bool() const noexcept { return static_cast<bool>(impl_); }
bool Delivery::has_payload() const noexcept { return impl_ && static_cast<bool>(impl_->lease); }
sqfv::ByteView Delivery::payload() const noexcept { return impl_ ? impl_->lease.payload() : sqfv::ByteView{}; }
const sqfv::Descriptor& Delivery::descriptor() const noexcept { return impl_->lease.descriptor(); }
const sqfv::ContentId& Delivery::content_id() const noexcept { return impl_->content_id; }
std::uint64_t Delivery::sequence() const noexcept { return impl_->sequence; }
Origin Delivery::origin() const noexcept { return impl_->origin; }
const sqpv::Receipt* Delivery::retention_receipt() const noexcept {
  return impl_ && impl_->has_receipt ? &impl_->receipt : nullptr;
}
void Delivery::release_payload() noexcept { if (impl_) impl_->lease.reset(); }
void Delivery::reset() noexcept { impl_.reset(); }

Session::Session() noexcept = default;
Session::~Session() noexcept = default;
Session::Session(Session&&) noexcept = default;
Session& Session::operator=(Session&&) noexcept = default;
Session::operator bool() const noexcept { return static_cast<bool>(impl_); }
void Session::reset() noexcept { impl_.reset(); }
std::string_view Session::view_reference() const noexcept { return impl_ ? impl_->reference : std::string_view{}; }
Status Session::create(sqfv::Context& context, const sqmv::Manifest& manifest,
                       const Config& config, const Limits& limits,
                       const RetainedSource* source, const Checkpoint* resume,
                       Session& out) noexcept {
  if (!context || !manifest || !valid_config(config, limits)) return Status::invalid_argument;
  if ((config.profile == Profile::retained_before_delivery) != (source != nullptr)) return Status::invalid_argument;
  if (source && !source->state_) return Status::invalid_argument;
  if (source && source->state_->identity->pid != ::getpid()) return Status::stale;
  try {
    auto state = std::make_unique<detail::SessionState>();
    state->identity = std::make_shared<detail::SessionIdentity>();
    state->config = config;
    state->limits = limits;
    const auto bound = manifest.binding(state->binding);
    if (bound != sqmv::Status::ok) return metadata_status(bound);
    if (source) {
      state->source = source->state_;
      const auto& actual = *state->source->identity;
      if (!same_binding(actual.binding, state->binding) || actual.options.partition != config.partition ||
          actual.options.producer_generation != config.producer_generation ||
          config.first_sequence < actual.options.first_sequence) return Status::binding_mismatch;
      std::lock_guard source_lock(state->source->mutex);
      if (state->source->uncertain) return Status::closed;
    }
    state->reference = symphony::sqdv::view_reference(state->binding, config,
        source ? source->state_->identity.get() : nullptr);
    auto next = config.first_sequence;
    bool exhausted = false;
    if (resume) {
      if (resume->view_reference != state->reference) return Status::binding_mismatch;
      if (resume->next_sequence < config.first_sequence ||
          (resume->sequence_exhausted && resume->next_sequence != UINT64_MAX)) return Status::invalid_argument;
      next = resume->next_sequence;
      exhausted = resume->sequence_exhausted;
    }
    state->ledger.resize(limits.max_unacknowledged_batches);
    state->next_offer = state->next_processed = next;
    state->offer_exhausted = state->processed_exhausted = exhausted;
    sqfv::PortConfig port_config{state->binding, config.partition, config.producer_generation,
                                next, limits.outstanding_byte_credit, limits.max_unacknowledged_batches};
    const auto added = context.add_port(port_config, state->port);
    if (added != sqfv::Status::ok) return flow_status(added);
    Session ready;
    ready.impl_ = std::move(state);
    out = std::move(ready);
    return Status::ok;
  } catch (const std::bad_alloc&) { return Status::no_memory; }
    catch (const std::length_error&) { return Status::limit; }
    catch (...) { return Status::internal_error; }
}
Status Session::offer_live(const sqfv::Batch& batch) noexcept {
  if (!impl_) return Status::closed;
  if (impl_->identity->pid != ::getpid()) return Status::stale;
  try {
    std::lock_guard lock(impl_->mutex);
    if (impl_->config.profile != Profile::disposable) return Status::invalid_argument;
    return impl_->enqueue(batch, Origin::live, nullptr);
  } catch (const std::bad_alloc&) { return Status::no_memory; }
    catch (...) { return Status::internal_error; }
}
Status Session::offer_next(sqfv::Context& context, const RetainedBatch* candidate) noexcept {
  if (!impl_) return Status::closed;
  if (impl_->identity->pid != ::getpid()) return Status::stale;
  try {
    std::lock_guard lock(impl_->mutex);
    if (impl_->config.profile != Profile::retained_before_delivery) return Status::invalid_argument;
    std::lock_guard source_lock(impl_->source->mutex);
    if (impl_->source->uncertain) return Status::closed;
    if (candidate) {
      if (!candidate->state_) return Status::invalid_argument;
      if (candidate->state_->identity.get() != impl_->source->identity.get()) return Status::binding_mismatch;
      const auto status = impl_->validate_batch(candidate->state_->batch);
      if (status != Status::ok) return status;
      std::size_t frame_size = 0;
      const auto measured = sqfv::frame_measure(context, candidate->state_->batch, frame_size);
      if (measured != sqfv::Status::ok) return flow_status(measured);
    }
    if (impl_->offer_exhausted) return Status::limit;
    if (impl_->used == impl_->ledger.size()) return Status::blocked;
    if (candidate && candidate->state_->batch.descriptor().batch_sequence == impl_->next_offer)
      return impl_->enqueue(candidate->state_->batch, Origin::live, &candidate->state_->receipt);
    sqfv::Batch batch;
    sqpv::Receipt receipt;
    const auto read = impl_->source->store.read(impl_->next_offer, context, batch, receipt);
    if (read != sqpv::Status::ok) return store_status(read);
    return impl_->enqueue(batch, Origin::retained, &receipt);
  } catch (const std::bad_alloc&) { return Status::no_memory; }
    catch (...) { return Status::internal_error; }
}
Status Session::take(Delivery& out) noexcept {
  if (!impl_) return Status::closed;
  if (impl_->identity->pid != ::getpid()) return Status::stale;
  try {
    Delivery ready;
    {
      std::lock_guard lock(impl_->mutex);
      if (impl_->source) {
        std::lock_guard source_lock(impl_->source->mutex);
        if (impl_->source->uncertain) return Status::closed;
      }
      if (impl_->delivered == impl_->used) return Status::empty;
      auto& entry = impl_->ledger[(static_cast<std::size_t>(impl_->head) + impl_->delivered) % impl_->ledger.size()];
      const auto taken = impl_->port.take(entry.pending->lease);
      if (taken != sqfv::Status::ok) return flow_status(taken);
      ready.impl_ = std::move(entry.pending);
      entry.delivered = true;
      ++impl_->delivered;
    }
    out = std::move(ready);
    return Status::ok;
  } catch (...) { return Status::internal_error; }
}
Status Session::acknowledge_processed(const Delivery& delivery) noexcept {
  if (!impl_) return Status::closed;
  if (impl_->identity->pid != ::getpid()) return Status::stale;
  if (!delivery.impl_) return Status::invalid_argument;
  try {
    std::lock_guard lock(impl_->mutex);
    const auto& ticket = *delivery.impl_;
    if (ticket.identity.get() != impl_->identity.get()) return Status::binding_mismatch;
    if (impl_->has_last_ack && ticket.sequence == impl_->last_ack)
      return ticket.content_id == impl_->last_ack_id ? Status::duplicate : Status::conflict;
    if (impl_->processed_exhausted || ticket.sequence < impl_->next_processed) return Status::stale;
    if (ticket.sequence > impl_->next_processed) return Status::gap;
    if (impl_->used == 0 || !impl_->ledger[impl_->head].delivered) return Status::gap;
    auto& entry = impl_->ledger[impl_->head];
    if (entry.sequence != ticket.sequence || entry.content_id != ticket.content_id) return Status::conflict;
    entry.delivered = false;
    impl_->head = (impl_->head + 1) % static_cast<std::uint32_t>(impl_->ledger.size());
    --impl_->used;
    --impl_->delivered;
    impl_->has_last_ack = true;
    impl_->last_ack = ticket.sequence;
    impl_->last_ack_id = ticket.content_id;
    advance(impl_->next_processed, impl_->processed_exhausted);
    return Status::ok;
  } catch (...) { return Status::internal_error; }
}
Status Session::checkpoint(Checkpoint& out) const noexcept {
  if (!impl_) return Status::closed;
  if (impl_->identity->pid != ::getpid()) return Status::stale;
  try {
    Checkpoint ready;
    {
      std::lock_guard lock(impl_->mutex);
      ready.view_reference = impl_->reference;
      ready.next_sequence = impl_->next_processed;
      ready.sequence_exhausted = impl_->processed_exhausted;
    }
    out = std::move(ready);
    return Status::ok;
  } catch (const std::bad_alloc&) { return Status::no_memory; }
    catch (...) { return Status::internal_error; }
}
Status Session::stats(SessionStats& out) const noexcept {
  if (!impl_) return Status::closed;
  if (impl_->identity->pid != ::getpid()) return Status::stale;
  try {
    SessionStats ready;
    {
      std::lock_guard lock(impl_->mutex);
      sqfv::PortStats port;
      const auto status = impl_->port.stats(port);
      if (status != sqfv::Status::ok) return flow_status(status);
      ready = {impl_->next_offer, impl_->next_processed, port.outstanding_bytes,
               impl_->used, impl_->used - impl_->delivered,
               impl_->offer_exhausted, impl_->processed_exhausted};
    }
    out = ready;
    return Status::ok;
  } catch (...) { return Status::internal_error; }
}

} // namespace symphony::sqdv
