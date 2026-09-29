#include "batch_internal.hpp"

#include <algorithm>
#include <array>
#include <cstring>
#include <exception>
#include <limits>
#include <memory>
#include <mutex>
#include <new>
#include <string>
#include <string_view>
#include <utility>
#include <vector>

namespace symphony::sqfv {
namespace {

constexpr std::uint64_t kMaxPayload = 64ull * 1024ull * 1024ull;
constexpr std::uint64_t kMaxFrame = 128ull * 1024ull * 1024ull;
constexpr std::uint64_t kMaxDescriptor = 64ull * 1024ull;
constexpr std::uint32_t kMaxPorts = 1024;
constexpr std::uint32_t kMaxPending = 65536;
constexpr std::uint64_t kAllocationAllowance = 64;

bool add_u64(std::uint64_t left, std::uint64_t right,
             std::uint64_t& out) noexcept {
  if (right > std::numeric_limits<std::uint64_t>::max() - left) return false;
  out = left + right;
  return true;
}

bool required(std::string_view field) noexcept {
  return !field.empty() && field.size() <= UINT16_MAX;
}

bool optional(std::string_view field) noexcept {
  return field.size() <= UINT16_MAX;
}

bool nonzero_generation(const Generation& generation) noexcept {
  return std::any_of(generation.begin(), generation.end(),
                     [](std::uint8_t byte) { return byte != 0; });
}

bool same(std::string_view borrowed, const std::string& owned) noexcept {
  return borrowed == std::string_view(owned);
}

Status validate_binding(const detail::BindingView& binding) noexcept {
  if (!required(binding.metadata_ref) ||
      !required(binding.dataset_revision) ||
      !required(binding.schema_version) ||
      !required(binding.layout_version) ||
      !required(binding.access_scope))
    return Status::invalid_argument;
  return Status::ok;
}

Status validate_descriptor(const detail::DescriptorView& descriptor,
                           std::uint64_t max_descriptor) noexcept {
  const auto binding = validate_binding(descriptor.binding);
  if (binding != Status::ok) return binding;
  if (!required(descriptor.partition) ||
      !optional(descriptor.source_binding) ||
      !optional(descriptor.source_position) ||
      (!descriptor.source_position.empty() &&
       descriptor.source_binding.empty()) ||
      !nonzero_generation(descriptor.producer_generation) ||
      descriptor.record_count == 0)
    return Status::invalid_argument;

  // Eight u16 lengths, one generation, sequence, and record count.
  std::uint64_t wire_size = 48;
  const std::array fields{
      descriptor.binding.metadata_ref,
      descriptor.binding.dataset_revision,
      descriptor.binding.schema_version,
      descriptor.binding.layout_version,
      descriptor.binding.access_scope,
      descriptor.partition,
      descriptor.source_binding,
      descriptor.source_position};
  for (const auto field : fields)
    if (!add_u64(wire_size, field.size(), wire_size)) return Status::limit;
  return wire_size > max_descriptor ? Status::limit : Status::ok;
}

detail::BindingView view_of_binding(const Binding& binding) noexcept {
  return {binding.metadata_ref, binding.dataset_revision,
          binding.schema_version, binding.layout_version,
          binding.access_scope};
}

bool same_binding(const detail::BindingView& borrowed,
                  const Binding& owned) noexcept {
  return same(borrowed.metadata_ref, owned.metadata_ref) &&
         same(borrowed.dataset_revision, owned.dataset_revision) &&
         same(borrowed.schema_version, owned.schema_version) &&
         same(borrowed.layout_version, owned.layout_version) &&
         same(borrowed.access_scope, owned.access_scope);
}

Descriptor copy_descriptor(const detail::DescriptorView& in) {
  return {{std::string(in.binding.metadata_ref),
           std::string(in.binding.dataset_revision),
           std::string(in.binding.schema_version),
           std::string(in.binding.layout_version),
           std::string(in.binding.access_scope)},
          std::string(in.partition),
          std::string(in.source_binding),
          std::string(in.source_position),
          in.producer_generation,
          in.batch_sequence,
          in.record_count};
}

std::uint64_t descriptor_string_bytes(const detail::DescriptorView& in) noexcept {
  return in.binding.metadata_ref.size() +
         in.binding.dataset_revision.size() +
         in.binding.schema_version.size() +
         in.binding.layout_version.size() +
         in.binding.access_scope.size() +
         in.partition.size() + in.source_binding.size() +
         in.source_position.size();
}

} // namespace

namespace detail {

struct ContextState {
  // The initial total includes this shared state's permanent reservation and
  // the first ContextHandle. Only the handle's portion is released when that
  // handle dies: batches, ports, and leases can still retain this state.
  explicit ContextState(Limits configured, std::uint64_t initial)
      : limits(configured), used(initial), peak(initial) {}

  bool reserve(std::uint64_t bytes) {
    std::lock_guard lock(mutex);
    if (bytes > limits.global_allocation_bytes - used) return false;
    used += bytes;
    peak = std::max(peak, used);
    return true;
  }

  void release(std::uint64_t bytes) noexcept {
    try {
      std::lock_guard lock(mutex);
      if (bytes > used) std::terminate();
      used -= bytes;
    } catch (...) {
      std::terminate();
    }
  }

  bool claim_port() {
    std::lock_guard lock(mutex);
    if (ports == limits.max_ports) return false;
    ++ports;
    return true;
  }

  void release_port() noexcept {
    try {
      std::lock_guard lock(mutex);
      if (ports == 0) std::terminate();
      --ports;
    } catch (...) {
      std::terminate();
    }
  }

  const Limits limits;
  std::mutex mutex;
  std::uint64_t used = 0;
  std::uint64_t peak = 0;
  std::uint32_t ports = 0;
};

struct Reservation {
  Reservation() = default;
  Reservation(std::shared_ptr<ContextState> owner, std::uint64_t amount)
      : context(std::move(owner)), bytes(amount) {}
  Reservation(const Reservation&) = delete;
  Reservation& operator=(const Reservation&) = delete;
  Reservation(Reservation&& other) noexcept
      : context(std::move(other.context)),
        bytes(std::exchange(other.bytes, 0)) {}
  Reservation& operator=(Reservation&& other) noexcept {
    if (this != &other) {
      reset();
      context = std::move(other.context);
      bytes = std::exchange(other.bytes, 0);
    }
    return *this;
  }
  ~Reservation() { reset(); }

  void reset() noexcept {
    if (context && bytes != 0) context->release(bytes);
    context.reset();
    bytes = 0;
  }

  std::shared_ptr<ContextState> context;
  std::uint64_t bytes = 0;
};

struct BatchData {
  BatchData(Reservation charge, const DescriptorView& input_descriptor,
            ByteView input_payload)
      : reservation(std::move(charge)),
        descriptor(copy_descriptor(input_descriptor)),
        payload(input_payload.begin(), input_payload.end()) {}

  Reservation reservation;
  Descriptor descriptor;
  std::vector<std::uint8_t> payload;
  ContentId content_id{};
};

struct PortState;

struct ContextHandle {
  std::shared_ptr<ContextState> state;
  Reservation reservation;
};

struct BatchHandle {
  Reservation reservation;
  std::shared_ptr<BatchData> data;
};

struct LeaseHandle {
  Reservation reservation;
  std::shared_ptr<PortState> port;
  std::shared_ptr<BatchData> data;
  std::uint64_t credit_bytes = 0;
  ~LeaseHandle() noexcept;
};

struct PortState {
  PortState(Reservation charge, const PortConfig& config)
      : reservation(std::move(charge)),
        binding(config.binding),
        partition(config.partition),
        generation(config.producer_generation),
        next_sequence(config.next_sequence),
        credit(config.outstanding_byte_credit),
        max_pending(config.max_pending_entries),
        queue(config.max_pending_entries, nullptr) {}

  Reservation reservation;
  std::mutex mutex;
  Binding binding;
  std::string partition;
  Generation generation{};
  std::uint64_t next_sequence = 0;
  std::uint64_t last_sequence = 0;
  std::uint64_t outstanding = 0;
  const std::uint64_t credit;
  const std::uint32_t max_pending;
  std::vector<LeaseHandle*> queue;
  std::uint32_t head = 0;
  std::uint32_t pending = 0;
  bool has_last = false;
  bool sequence_exhausted = false;
  bool closed = false;
  ContentId last_id{};
};

LeaseHandle::~LeaseHandle() noexcept {
  if (!port) return;
  try {
    std::lock_guard lock(port->mutex);
    if (credit_bytes > port->outstanding) std::terminate();
    port->outstanding -= credit_bytes;
  } catch (...) {
    std::terminate();
  }
}

struct PortHandle {
  Reservation reservation;
  std::shared_ptr<PortState> state;
  ~PortHandle() noexcept {
    if (!state) return;
    try {
      {
        std::lock_guard lock(state->mutex);
        state->closed = true;
      }
      for (;;) {
        LeaseHandle* pending = nullptr;
        {
          std::lock_guard lock(state->mutex);
          if (state->pending == 0) break;
          pending = state->queue[state->head];
          state->queue[state->head] = nullptr;
          state->head = (state->head + 1) % state->max_pending;
          --state->pending;
        }
        delete pending;
      }
      state->reservation.context->release_port();
    } catch (...) {
      std::terminate();
    }
  }
};

} // namespace detail

namespace {

std::uint64_t batch_reservation_bytes(const detail::DescriptorView& descriptor,
                                      std::uint64_t payload_bytes) noexcept {
  return sizeof(detail::BatchData) + 10 * kAllocationAllowance +
         descriptor_string_bytes(descriptor) + payload_bytes;
}

std::uint64_t port_reservation_bytes(const PortConfig& config) noexcept {
  const auto& b = config.binding;
  return sizeof(detail::PortState) + 9 * kAllocationAllowance +
         std::uint64_t(config.max_pending_entries) *
             sizeof(detail::LeaseHandle*) +
         b.metadata_ref.size() + b.dataset_revision.size() +
         b.schema_version.size() + b.layout_version.size() +
         b.access_scope.size() + config.partition.size();
}

template <typename Handle>
std::uint64_t handle_reservation_bytes() noexcept {
  return sizeof(Handle) + kAllocationAllowance;
}

Status create_batch_handle(const std::shared_ptr<detail::BatchData>& data,
                           std::unique_ptr<detail::BatchHandle>& out) {
  const auto state = data->reservation.context;
  const auto amount = handle_reservation_bytes<detail::BatchHandle>();
  if (!state->reserve(amount)) return Status::limit;
  detail::Reservation charge(state, amount);
  auto handle = std::make_unique<detail::BatchHandle>();
  handle->reservation = std::move(charge);
  handle->data = data;
  out = std::move(handle);
  return Status::ok;
}

Status create_lease(const std::shared_ptr<detail::BatchData>& data,
                    const std::shared_ptr<detail::PortState>& port,
                    std::unique_ptr<detail::LeaseHandle>& out) {
  const auto state = data->reservation.context;
  const auto amount = handle_reservation_bytes<detail::LeaseHandle>();
  if (!state->reserve(amount)) return Status::limit;
  detail::Reservation charge(state, amount);
  auto lease = std::make_unique<detail::LeaseHandle>();
  lease->reservation = std::move(charge);
  lease->port = port;
  lease->data = data;
  lease->credit_bytes = port ? data->payload.size() : 0;
  out = std::move(lease);
  return Status::ok;
}

Status prepare_copy_state(const std::shared_ptr<detail::ContextState>& state,
                          const detail::DescriptorView& descriptor,
                          ByteView payload,
                          std::unique_ptr<detail::BatchHandle>& out) {
  if (payload.empty() || payload.size() > state->limits.max_payload_bytes)
    return Status::limit;
  const auto checked =
      validate_descriptor(descriptor, state->limits.max_descriptor_bytes);
  if (checked != Status::ok) return checked;
  const auto amount = batch_reservation_bytes(descriptor, payload.size());
  if (!state->reserve(amount)) return Status::limit;
  detail::Reservation charge(state, amount);
  auto data =
      std::make_shared<detail::BatchData>(std::move(charge), descriptor, payload);
  if (!detail::compute_content_id(detail::view_of(data->descriptor),
                                  data->payload, data->content_id))
    return Status::internal_error;
  return create_batch_handle(data, out);
}

Status cursor_result(const detail::PortState& port,
                     const detail::BatchData& batch) noexcept {
  const auto sequence = batch.descriptor.batch_sequence;
  if (port.has_last && sequence == port.last_sequence)
    return batch.content_id == port.last_id
               ? Status::duplicate
               : Status::conflict;
  if (port.sequence_exhausted || sequence < port.next_sequence)
    return Status::stale;
  if (sequence > port.next_sequence) return Status::gap;
  return Status::ok;
}

} // namespace

Context::Context() noexcept = default;
Context::~Context() noexcept = default;
Context::Context(Context&&) noexcept = default;
Context& Context::operator=(Context&&) noexcept = default;

Context::operator bool() const noexcept { return bool(impl_); }

Status Context::create(const Limits& limits, Context& out) noexcept {
  if (limits.max_payload_bytes == 0 ||
      limits.max_payload_bytes > kMaxPayload ||
      limits.max_frame_bytes == 0 ||
      limits.max_frame_bytes > kMaxFrame ||
      limits.max_descriptor_bytes == 0 ||
      limits.max_descriptor_bytes > kMaxDescriptor ||
      limits.global_allocation_bytes == 0 ||
      limits.max_ports == 0 || limits.max_ports > kMaxPorts)
    return Status::invalid_argument;
  const auto state_amount =
      sizeof(detail::ContextState) + kAllocationAllowance;
  const auto handle_amount =
      handle_reservation_bytes<detail::ContextHandle>();
  const auto initial = state_amount + handle_amount;
  if (limits.global_allocation_bytes < initial) return Status::limit;
  try {
    auto state = std::make_shared<detail::ContextState>(limits, initial);
    auto handle = std::make_unique<detail::ContextHandle>();
    handle->state = state;
    handle->reservation = detail::Reservation(state, handle_amount);
    out.impl_ = std::move(handle);
    return Status::ok;
  } catch (const std::bad_alloc&) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}

Status Context::retain(Context& out) const noexcept {
  if (!impl_) return Status::invalid_argument;
  try {
    auto state = impl_->state;
    const auto amount = handle_reservation_bytes<detail::ContextHandle>();
    if (!state->reserve(amount)) return Status::limit;
    detail::Reservation charge(state, amount);
    auto handle = std::make_unique<detail::ContextHandle>();
    handle->state = std::move(state);
    handle->reservation = std::move(charge);
    out.impl_ = std::move(handle);
    return Status::ok;
  } catch (const std::bad_alloc&) { return Status::no_memory; }
    catch (...) { return Status::internal_error; }
}

Status Context::prepare_copy(const Descriptor& descriptor, ByteView payload,
                             Batch& out) const noexcept {
  if (!impl_) return Status::invalid_argument;
  try {
    return prepare_copy_state(impl_->state, detail::view_of(descriptor),
                              payload, out.impl_);
  } catch (const std::bad_alloc&) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}

Status Context::add_port(const PortConfig& config, Port& out) const noexcept {
  if (!impl_) return Status::invalid_argument;
  const auto binding = validate_binding(view_of_binding(config.binding));
  if (binding != Status::ok) return binding;
  if (!required(config.partition) ||
      !nonzero_generation(config.producer_generation) ||
      config.outstanding_byte_credit == 0 ||
      config.max_pending_entries == 0 ||
      config.max_pending_entries > kMaxPending)
    return Status::invalid_argument;
  const auto state = impl_->state;
  const auto amount = port_reservation_bytes(config);
  bool claimed = false;
  try {
    if (!state->reserve(amount)) return Status::limit;
    detail::Reservation charge(state, amount);
    if (!state->claim_port()) return Status::limit;
    claimed = true;
    auto port_state =
        std::make_shared<detail::PortState>(std::move(charge), config);
    const auto handle_amount =
        handle_reservation_bytes<detail::PortHandle>();
    if (!state->reserve(handle_amount)) {
      state->release_port();
      return Status::limit;
    }
    detail::Reservation handle_charge(state, handle_amount);
    auto handle = std::make_unique<detail::PortHandle>();
    handle->reservation = std::move(handle_charge);
    handle->state = std::move(port_state);
    claimed = false;
    out.impl_ = std::move(handle);
    return Status::ok;
  } catch (const std::bad_alloc&) {
    if (claimed) state->release_port();
    return Status::no_memory;
  } catch (...) {
    if (claimed) state->release_port();
    return Status::internal_error;
  }
}

Status Context::stats(ContextStats& out) const noexcept {
  if (!impl_) return Status::invalid_argument;
  try {
    const auto state = impl_->state;
    std::lock_guard lock(state->mutex);
    out = {state->used, state->peak,
           state->limits.global_allocation_bytes, state->ports};
    return Status::ok;
  } catch (...) {
    return Status::internal_error;
  }
}

Batch::Batch() noexcept = default;
Batch::~Batch() noexcept = default;
Batch::Batch(Batch&&) noexcept = default;
Batch& Batch::operator=(Batch&&) noexcept = default;

Batch::operator bool() const noexcept { return bool(impl_); }

Status Batch::retain(Batch& out) const noexcept {
  if (!impl_) return Status::invalid_argument;
  try {
    return create_batch_handle(impl_->data, out.impl_);
  } catch (const std::bad_alloc&) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}

Status Batch::acquire(std::string_view access_scope,
                      Lease& out) const noexcept {
  if (!impl_) return Status::invalid_argument;
  if (!same(access_scope, impl_->data->descriptor.binding.access_scope))
    return Status::scope_mismatch;
  try {
    return create_lease(impl_->data, {}, out.impl_);
  } catch (const std::bad_alloc&) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}

const Descriptor& Batch::descriptor() const noexcept {
  return impl_->data->descriptor;
}

const ContentId& Batch::content_id() const noexcept {
  return impl_->data->content_id;
}

Lease::Lease() noexcept = default;
Lease::~Lease() noexcept = default;
Lease::Lease(Lease&&) noexcept = default;
Lease& Lease::operator=(Lease&&) noexcept = default;

Lease::operator bool() const noexcept { return bool(impl_); }

ByteView Lease::payload() const noexcept {
  if (!impl_) return {};
  return impl_->data->payload;
}

const Descriptor& Lease::descriptor() const noexcept {
  return impl_->data->descriptor;
}

const ContentId& Lease::content_id() const noexcept {
  return impl_->data->content_id;
}

void Lease::reset() noexcept { impl_.reset(); }

Port::Port() noexcept = default;
Port::~Port() noexcept = default;
Port::Port(Port&&) noexcept = default;
Port& Port::operator=(Port&&) noexcept = default;

Port::operator bool() const noexcept { return bool(impl_); }

Status Port::offer(const Batch& batch) noexcept {
  if (!impl_ || !batch.impl_) return Status::invalid_argument;
  try {
    const auto& state = impl_->state;
    const auto& data = batch.impl_->data;
    if (state->reservation.context.get() !=
        data->reservation.context.get())
      return Status::invalid_argument;
    std::lock_guard lock(state->mutex);
    if (state->closed) return Status::closed;
    const auto descriptor = detail::view_of(data->descriptor);
    if (!same(descriptor.binding.access_scope,
              state->binding.access_scope))
      return Status::scope_mismatch;
    if (!same_binding(descriptor.binding, state->binding) ||
        !same(descriptor.partition, state->partition) ||
        descriptor.producer_generation != state->generation)
      return Status::binding_mismatch;
    const auto cursor = cursor_result(*state, *data);
    if (cursor != Status::ok) return cursor;
    const auto size = data->payload.size();
    if (size > state->credit) return Status::limit;
    if (state->pending == state->max_pending ||
        size > state->credit - state->outstanding)
      return Status::blocked;
    std::unique_ptr<detail::LeaseHandle> lease;
    const auto created = create_lease(data, state, lease);
    if (created != Status::ok) return created;
    const auto tail = (state->head + state->pending) % state->max_pending;
    state->queue[tail] = lease.release();
    ++state->pending;
    state->outstanding += size;
    state->last_sequence = descriptor.batch_sequence;
    state->last_id = data->content_id;
    state->has_last = true;
    if (state->next_sequence == UINT64_MAX)
      state->sequence_exhausted = true;
    else
      ++state->next_sequence;
    return Status::ok;
  } catch (const std::bad_alloc&) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}

Status Port::take(Lease& out) noexcept {
  if (!impl_) return Status::invalid_argument;
  try {
    detail::LeaseHandle* taken = nullptr;
    {
      const auto& state = impl_->state;
      std::lock_guard lock(state->mutex);
      if (state->closed) return Status::closed;
      if (state->pending == 0) return Status::empty;
      taken = state->queue[state->head];
      state->queue[state->head] = nullptr;
      state->head = (state->head + 1) % state->max_pending;
      --state->pending;
    }
    out.impl_.reset(taken);
    return Status::ok;
  } catch (const std::bad_alloc&) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}

Status Port::cancel(std::uint64_t batch_sequence) noexcept {
  if (!impl_) return Status::invalid_argument;
  try {
    detail::LeaseHandle* cancelled = nullptr;
    {
      const auto& state = impl_->state;
      std::lock_guard lock(state->mutex);
      if (state->closed) return Status::closed;
      for (std::uint32_t offset = 0; offset < state->pending; ++offset) {
        const auto index = (state->head + offset) % state->max_pending;
        if (state->queue[index]->data->descriptor.batch_sequence !=
            batch_sequence)
          continue;
        cancelled = state->queue[index];
        for (std::uint32_t shift = offset; shift + 1 < state->pending;
             ++shift) {
          const auto here = (state->head + shift) % state->max_pending;
          const auto after =
              (state->head + shift + 1) % state->max_pending;
          state->queue[here] = state->queue[after];
        }
        const auto tail =
            (state->head + state->pending - 1) % state->max_pending;
        state->queue[tail] = nullptr;
        --state->pending;
        break;
      }
    }
    if (!cancelled) return Status::empty;
    delete cancelled;
    return Status::ok;
  } catch (const std::bad_alloc&) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}

Status Port::stats(PortStats& out) const noexcept {
  if (!impl_) return Status::invalid_argument;
  try {
    const auto& state = impl_->state;
    std::lock_guard lock(state->mutex);
    out = {state->next_sequence,
           state->outstanding,
           state->credit,
           state->pending,
           state->max_pending,
           state->sequence_exhausted};
    return Status::ok;
  } catch (...) {
    return Status::internal_error;
  }
}

void Port::reset() noexcept { impl_.reset(); }

const Limits* detail::FrameAccess::limits(const Context& context) noexcept {
  return context.impl_ ? &context.impl_->state->limits : nullptr;
}

ByteView detail::FrameAccess::payload(const Batch& batch) noexcept {
  return batch.impl_ ? ByteView(batch.impl_->data->payload) : ByteView{};
}

bool detail::FrameAccess::same_context(const Context& context,
                                       const Batch& batch) noexcept {
  return context.impl_ && batch.impl_ &&
         context.impl_->state.get() ==
             batch.impl_->data->reservation.context.get();
}

Status detail::FrameAccess::prepare_copy_views(
    Context& context, const DescriptorView& descriptor, ByteView payload,
    Batch& out) noexcept {
  if (!context.impl_) return Status::invalid_argument;
  try {
    return prepare_copy_state(context.impl_->state, descriptor, payload,
                              out.impl_);
  } catch (const std::bad_alloc&) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}

} // namespace symphony::sqfv
