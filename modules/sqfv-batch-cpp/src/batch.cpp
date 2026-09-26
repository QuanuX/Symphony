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
#include <utility>
#include <vector>

namespace {

constexpr uint64_t kMaxPayload = 64ull * 1024ull * 1024ull;
constexpr uint64_t kMaxFrame = 128ull * 1024ull * 1024ull;
constexpr uint64_t kMaxDescriptor = 64ull * 1024ull;
constexpr uint32_t kMaxPorts = 1024u;
constexpr uint32_t kMaxPending = 65536u;
// Conservative per-allocation allowance beyond the requested object/bytes.
constexpr uint64_t kAllocationAllowance = 64u;

bool add_u64(uint64_t left, uint64_t right, uint64_t &out) noexcept {
  if (right > std::numeric_limits<uint64_t>::max() - left)
    return false;
  out = left + right;
  return true;
}

bool valid_span(sqfv_span span) noexcept {
  return span.size == 0 || span.data != nullptr;
}

bool required_span(sqfv_span span) noexcept {
  return valid_span(span) && span.size != 0 && span.size <= UINT16_MAX;
}

bool optional_span(sqfv_span span) noexcept {
  return valid_span(span) && span.size <= UINT16_MAX;
}

bool same(sqfv_span span, const std::string &owned) noexcept {
  return span.size == owned.size() &&
         (span.size == 0 ||
          std::memcmp(span.data, owned.data(), owned.size()) == 0);
}

sqfv_span view(const std::string &owned) noexcept {
  return {reinterpret_cast<const uint8_t *>(owned.data()),
          static_cast<uint64_t>(owned.size())};
}

std::string copy(sqfv_span span) {
  if (span.size == 0)
    return {};
  return {reinterpret_cast<const char *>(span.data),
          static_cast<size_t>(span.size)};
}

bool nonzero_generation(const uint8_t generation[16]) noexcept {
  for (size_t i = 0; i != 16; ++i)
    if (generation[i] != 0)
      return true;
  return false;
}

sqfv_status checked_binding(const sqfv_binding &binding) noexcept {
  if (binding.struct_size != sizeof(sqfv_binding) ||
      binding.abi_version != SQFV_BATCH_ABI_VERSION)
    return SQFV_UNSUPPORTED_ABI;
  if (!required_span(binding.metadata_ref) ||
      !required_span(binding.dataset_revision) ||
      !required_span(binding.schema_version) ||
      !required_span(binding.layout_version) ||
      !required_span(binding.access_scope))
    return SQFV_INVALID_ARGUMENT;
  return SQFV_OK;
}

struct BindingValue {
  std::string metadata_ref;
  std::string dataset_revision;
  std::string schema_version;
  std::string layout_version;
  std::string access_scope;

  explicit BindingValue(const sqfv_binding &in)
      : metadata_ref(copy(in.metadata_ref)),
        dataset_revision(copy(in.dataset_revision)),
        schema_version(copy(in.schema_version)),
        layout_version(copy(in.layout_version)),
        access_scope(copy(in.access_scope)) {}

  sqfv_binding as_view() const noexcept {
    return {sizeof(sqfv_binding), SQFV_BATCH_ABI_VERSION,
            view(metadata_ref),   view(dataset_revision),
            view(schema_version), view(layout_version),
            view(access_scope)};
  }

  uint64_t bytes() const noexcept {
    return metadata_ref.size() + dataset_revision.size() +
           schema_version.size() + layout_version.size() + access_scope.size();
  }

  bool exactly(const sqfv_binding &other) const noexcept {
    return same(other.metadata_ref, metadata_ref) &&
           same(other.dataset_revision, dataset_revision) &&
           same(other.schema_version, schema_version) &&
           same(other.layout_version, layout_version) &&
           same(other.access_scope, access_scope);
  }
};

struct DescriptorValue {
  BindingValue binding;
  std::string partition;
  std::string source_binding;
  std::string source_position;
  std::array<uint8_t, 16> generation{};
  uint64_t sequence;
  uint64_t record_count;

  explicit DescriptorValue(const sqfv_descriptor &in)
      : binding(in.binding), partition(copy(in.partition)),
        source_binding(copy(in.source_binding)),
        source_position(copy(in.source_position)), sequence(in.batch_sequence),
        record_count(in.record_count) {
    std::copy_n(in.producer_generation, 16, generation.data());
  }

  sqfv_descriptor as_view() const noexcept {
    sqfv_descriptor out{};
    out.struct_size = sizeof(out);
    out.abi_version = SQFV_BATCH_ABI_VERSION;
    out.binding = binding.as_view();
    out.partition = view(partition);
    out.source_binding = view(source_binding);
    out.source_position = view(source_position);
    std::copy(generation.begin(), generation.end(), out.producer_generation);
    out.batch_sequence = sequence;
    out.record_count = record_count;
    return out;
  }
};

sqfv_status validate_descriptor(const sqfv_descriptor &in,
                                uint64_t max_descriptor) noexcept {
  if (in.struct_size != sizeof(sqfv_descriptor) ||
      in.abi_version != SQFV_BATCH_ABI_VERSION)
    return SQFV_UNSUPPORTED_ABI;
  const auto binding_status = checked_binding(in.binding);
  if (binding_status != SQFV_OK)
    return binding_status;
  if (!required_span(in.partition) || !optional_span(in.source_binding) ||
      !optional_span(in.source_position) ||
      (in.source_position.size != 0 && in.source_binding.size == 0) ||
      !nonzero_generation(in.producer_generation) || in.record_count == 0)
    return SQFV_INVALID_ARGUMENT;
  // Eight u16 lengths, generation, sequence and record count are mandatory.
  uint64_t wire_size = 48;
  const sqfv_span fields[] = {
      in.binding.metadata_ref,   in.binding.dataset_revision,
      in.binding.schema_version, in.binding.layout_version,
      in.binding.access_scope,   in.partition,
      in.source_binding,         in.source_position};
  for (const auto &field : fields)
    if (!add_u64(wire_size, field.size, wire_size))
      return SQFV_LIMIT;
  if (wire_size > max_descriptor)
    return SQFV_LIMIT;
  return SQFV_OK;
}

bool validate_output(uint32_t size, uint32_t version,
                     uint32_t expected) noexcept {
  return size == expected && version == SQFV_BATCH_ABI_VERSION;
}

} // namespace

struct ContextState {
  explicit ContextState(sqfv_limits configured, uint64_t initial_bytes)
      : limits(configured), used(initial_bytes), peak(initial_bytes) {}

  bool reserve(uint64_t bytes) {
    std::lock_guard lock(mutex);
    if (bytes > limits.global_allocation_bytes - used)
      return false;
    used += bytes;
    peak = std::max(peak, used);
    return true;
  }

  void release(uint64_t bytes) noexcept {
    std::lock_guard lock(mutex);
    used -= bytes;
  }

  bool claim_port() {
    std::lock_guard lock(mutex);
    if (ports == limits.max_ports)
      return false;
    ++ports;
    return true;
  }

  void release_port() noexcept {
    std::lock_guard lock(mutex);
    --ports;
  }

  const sqfv_limits limits;
  std::mutex mutex;
  uint64_t used = 0;
  uint64_t peak = 0;
  uint32_t ports = 0;
};

struct Reservation {
  Reservation() = default;
  Reservation(std::shared_ptr<ContextState> state, uint64_t amount)
      : context(std::move(state)), bytes(amount) {}
  Reservation(const Reservation &) = delete;
  Reservation &operator=(const Reservation &) = delete;
  Reservation(Reservation &&other) noexcept
      : context(std::move(other.context)),
        bytes(std::exchange(other.bytes, 0)) {}
  Reservation &operator=(Reservation &&other) noexcept {
    if (this != &other) {
      reset();
      context = std::move(other.context);
      bytes = std::exchange(other.bytes, 0);
    }
    return *this;
  }
  ~Reservation() { reset(); }

  void reset() noexcept {
    if (context && bytes != 0)
      context->release(bytes);
    context.reset();
    bytes = 0;
  }

  std::shared_ptr<ContextState> context;
  uint64_t bytes = 0;
};

struct BatchData {
  BatchData(Reservation charge, const sqfv_descriptor &descriptor,
            sqfv_span input)
      : reservation(std::move(charge)), descriptor(descriptor),
        payload(input.data, input.data + input.size) {}

  Reservation reservation;
  DescriptorValue descriptor;
  std::vector<uint8_t> payload;
  std::array<uint8_t, SQFV_BATCH_CONTENT_ID_BYTES> content_id{};
};

struct PortState;

struct sqfv_context {
  Reservation reservation;
  std::shared_ptr<ContextState> state;
};

struct sqfv_batch {
  Reservation reservation;
  std::shared_ptr<BatchData> data;
};

struct sqfv_lease {
  Reservation reservation;
  std::shared_ptr<PortState> port;
  std::shared_ptr<BatchData> data;
  uint64_t credit_bytes = 0;
};

struct PortState {
  PortState(Reservation charge, const sqfv_port_config &config)
      : reservation(std::move(charge)), binding(config.binding),
        partition(copy(config.partition)), next_sequence(config.next_sequence),
        credit(config.outstanding_byte_credit),
        max_pending(config.max_pending_entries),
        queue(config.max_pending_entries, nullptr) {
    std::copy_n(config.producer_generation, 16, generation.data());
  }

  Reservation reservation;
  std::mutex mutex;
  BindingValue binding;
  std::string partition;
  std::array<uint8_t, 16> generation{};
  uint64_t next_sequence;
  uint64_t last_sequence = 0;
  uint64_t outstanding = 0;
  const uint64_t credit;
  const uint32_t max_pending;
  std::vector<sqfv_lease *> queue;
  uint32_t head = 0;
  uint32_t pending = 0;
  bool has_last = false;
  bool sequence_exhausted = false;
  bool closed = false;
  std::array<uint8_t, SQFV_BATCH_CONTENT_ID_BYTES> last_id{};
};

struct sqfv_port {
  Reservation reservation;
  std::shared_ptr<PortState> state;
};

namespace {

uint64_t batch_reservation_bytes(const sqfv_descriptor &in,
                                 uint64_t payload_bytes) noexcept {
  // Shared-control block, eight potential string allocations, vector storage,
  // and allocator slack are covered before any copy. A single payload is
  // charged once even if several handles or leases retain it.
  uint64_t result = sizeof(BatchData) + kAllocationAllowance * 10;
  const sqfv_span fields[] = {
      in.binding.metadata_ref,   in.binding.dataset_revision,
      in.binding.schema_version, in.binding.layout_version,
      in.binding.access_scope,   in.partition,
      in.source_binding,         in.source_position};
  for (const auto &field : fields)
    result += field.size;
  result += payload_bytes;
  return result;
}

uint64_t port_reservation_bytes(const sqfv_port_config &in) noexcept {
  uint64_t result = sizeof(PortState) + kAllocationAllowance * 9 +
                    uint64_t(in.max_pending_entries) * sizeof(sqfv_lease *);
  const sqfv_span fields[] = {
      in.binding.metadata_ref,   in.binding.dataset_revision,
      in.binding.schema_version, in.binding.layout_version,
      in.binding.access_scope,   in.partition};
  for (const auto &field : fields)
    result += field.size;
  return result;
}

template <class Handle> uint64_t handle_reservation_bytes() noexcept {
  return sizeof(Handle) + kAllocationAllowance;
}

sqfv_status create_batch_handle(const std::shared_ptr<BatchData> &data,
                                sqfv_batch **out) {
  const auto state = data->reservation.context;
  const auto amount = handle_reservation_bytes<sqfv_batch>();
  if (!state->reserve(amount))
    return SQFV_LIMIT;
  Reservation charge(state, amount);
  auto handle = std::make_unique<sqfv_batch>();
  handle->reservation = std::move(charge);
  handle->data = data;
  *out = handle.release();
  return SQFV_OK;
}

sqfv_status create_lease(const std::shared_ptr<BatchData> &data,
                         const std::shared_ptr<PortState> &port,
                         sqfv_lease **out) {
  const auto state = data->reservation.context;
  const auto amount = handle_reservation_bytes<sqfv_lease>();
  if (!state->reserve(amount))
    return SQFV_LIMIT;
  Reservation charge(state, amount);
  auto lease = std::make_unique<sqfv_lease>();
  lease->reservation = std::move(charge);
  lease->port = port;
  lease->data = data;
  lease->credit_bytes = port ? data->payload.size() : 0;
  *out = lease.release();
  return SQFV_OK;
}

void release_lease(sqfv_lease *lease) noexcept {
  if (!lease)
    return;
  try {
    if (lease->port) {
      std::lock_guard lock(lease->port->mutex);
      lease->port->outstanding -= lease->credit_bytes;
    }
    delete lease;
  } catch (...) {
    // A void release cannot report a failed synchronization operation. Never
    // cross the C ABI or claim that a retained allocation was reclaimed.
    std::terminate();
  }
}

sqfv_status port_cursor_result(const PortState &port,
                               const BatchData &batch) noexcept {
  const uint64_t sequence = batch.descriptor.sequence;
  if (port.has_last && sequence == port.last_sequence) {
    return batch.content_id == port.last_id ? SQFV_DUPLICATE : SQFV_CONFLICT;
  }
  if (port.sequence_exhausted || sequence < port.next_sequence)
    return SQFV_STALE;
  if (sequence > port.next_sequence)
    return SQFV_GAP;
  return SQFV_OK;
}

} // namespace

namespace sqfv_internal {

sqfv_limits context_limits(const sqfv_context *context) noexcept {
  return context ? context->state->limits : sqfv_limits{};
}

sqfv_span batch_payload_view(const sqfv_batch *batch) noexcept {
  if (!batch)
    return {};
  return {batch->data->payload.data(),
          static_cast<uint64_t>(batch->data->payload.size())};
}

bool batch_belongs_to_context(const sqfv_batch *batch,
                              const sqfv_context *context) noexcept {
  return batch && context &&
         batch->data->reservation.context.get() == context->state.get();
}

} // namespace sqfv_internal

extern "C" sqfv_status sqfv_context_create(const sqfv_limits *limits,
                                           sqfv_context **out_context) {
  if (out_context)
    *out_context = nullptr;
  if (!limits || !out_context)
    return SQFV_INVALID_ARGUMENT;
  if (limits->struct_size != sizeof(sqfv_limits) ||
      limits->abi_version != SQFV_BATCH_ABI_VERSION)
    return SQFV_UNSUPPORTED_ABI;
  if (limits->reserved != 0 || limits->max_payload_bytes == 0 ||
      limits->max_payload_bytes > kMaxPayload || limits->max_frame_bytes == 0 ||
      limits->max_frame_bytes > kMaxFrame ||
      limits->max_descriptor_bytes == 0 ||
      limits->max_descriptor_bytes > kMaxDescriptor ||
      limits->global_allocation_bytes == 0 || limits->max_ports == 0 ||
      limits->max_ports > kMaxPorts)
    return SQFV_INVALID_ARGUMENT;
  const uint64_t initial =
      sizeof(ContextState) + sizeof(sqfv_context) + kAllocationAllowance * 2;
  if (limits->global_allocation_bytes < initial)
    return SQFV_LIMIT;
  try {
    auto state = std::make_shared<ContextState>(*limits, initial);
    auto context = std::make_unique<sqfv_context>();
    context->state = std::move(state);
    *out_context = context.release();
    return SQFV_OK;
  } catch (const std::bad_alloc &) {
    return SQFV_NO_MEMORY;
  } catch (...) {
    return SQFV_INTERNAL_ERROR;
  }
}

extern "C" void sqfv_context_destroy(sqfv_context *context) {
  try {
    delete context;
  } catch (...) {
    std::terminate();
  }
}

extern "C" sqfv_status sqfv_context_get_stats(const sqfv_context *context,
                                              sqfv_context_stats *out_stats) {
  try {
    if (!context || !out_stats)
      return SQFV_INVALID_ARGUMENT;
    if (!validate_output(out_stats->struct_size, out_stats->abi_version,
                         sizeof(*out_stats)))
      return SQFV_UNSUPPORTED_ABI;
    std::lock_guard lock(context->state->mutex);
    out_stats->allocation_bytes = context->state->used;
    out_stats->peak_allocation_bytes = context->state->peak;
    out_stats->allocation_limit_bytes =
        context->state->limits.global_allocation_bytes;
    out_stats->live_ports = context->state->ports;
    out_stats->reserved = 0;
    return SQFV_OK;
  } catch (const std::bad_alloc &) {
    return SQFV_NO_MEMORY;
  } catch (...) {
    return SQFV_INTERNAL_ERROR;
  }
}

extern "C" sqfv_status
sqfv_batch_prepare_copy(sqfv_context *context,
                        const sqfv_descriptor *descriptor, sqfv_span payload,
                        sqfv_batch **out_batch) {
  if (out_batch)
    *out_batch = nullptr;
  if (!context || !descriptor || !out_batch || !valid_span(payload))
    return SQFV_INVALID_ARGUMENT;
  const auto state = context->state;
  if (payload.size == 0 || payload.size > state->limits.max_payload_bytes ||
      payload.size > SIZE_MAX)
    return SQFV_LIMIT;
  const auto descriptor_status =
      validate_descriptor(*descriptor, state->limits.max_descriptor_bytes);
  if (descriptor_status != SQFV_OK)
    return descriptor_status;
  const uint64_t cost = batch_reservation_bytes(*descriptor, payload.size);
  try {
    if (!state->reserve(cost))
      return SQFV_LIMIT;
    Reservation charge(state, cost);
    auto data =
        std::make_shared<BatchData>(std::move(charge), *descriptor, payload);
    const auto desc = data->descriptor.as_view();
    const auto bytes = sqfv_span{data->payload.data(), data->payload.size()};
    if (!sqfv_internal::compute_content_id(desc, bytes,
                                           data->content_id.data()))
      return SQFV_INTERNAL_ERROR;
    return create_batch_handle(data, out_batch);
  } catch (const std::bad_alloc &) {
    return SQFV_NO_MEMORY;
  } catch (...) {
    return SQFV_INTERNAL_ERROR;
  }
}

extern "C" sqfv_status sqfv_batch_retain(const sqfv_batch *batch,
                                         sqfv_batch **out_batch) {
  if (out_batch)
    *out_batch = nullptr;
  if (!batch || !out_batch)
    return SQFV_INVALID_ARGUMENT;
  try {
    return create_batch_handle(batch->data, out_batch);
  } catch (const std::bad_alloc &) {
    return SQFV_NO_MEMORY;
  } catch (...) {
    return SQFV_INTERNAL_ERROR;
  }
}

extern "C" void sqfv_batch_release(sqfv_batch *batch) {
  try {
    delete batch;
  } catch (...) {
    std::terminate();
  }
}

extern "C" sqfv_status
sqfv_batch_descriptor_view(const sqfv_batch *batch,
                           sqfv_descriptor *out_descriptor) {
  if (!batch || !out_descriptor)
    return SQFV_INVALID_ARGUMENT;
  if (!validate_output(out_descriptor->struct_size,
                       out_descriptor->abi_version,
                       sizeof(*out_descriptor)))
    return SQFV_UNSUPPORTED_ABI;
  *out_descriptor = batch->data->descriptor.as_view();
  return SQFV_OK;
}

extern "C" sqfv_status
sqfv_batch_content_id(const sqfv_batch *batch,
                      uint8_t out_id[SQFV_BATCH_CONTENT_ID_BYTES]) {
  if (!batch || !out_id)
    return SQFV_INVALID_ARGUMENT;
  std::copy(batch->data->content_id.begin(), batch->data->content_id.end(),
            out_id);
  return SQFV_OK;
}

extern "C" sqfv_status sqfv_lease_acquire(const sqfv_batch *batch,
                                          sqfv_span access_scope,
                                          sqfv_lease **out_lease) {
  if (out_lease)
    *out_lease = nullptr;
  if (!batch || !out_lease || !valid_span(access_scope))
    return SQFV_INVALID_ARGUMENT;
  if (!same(access_scope, batch->data->descriptor.binding.access_scope))
    return SQFV_SCOPE_MISMATCH;
  try {
    return create_lease(batch->data, {}, out_lease);
  } catch (const std::bad_alloc &) {
    return SQFV_NO_MEMORY;
  } catch (...) {
    return SQFV_INTERNAL_ERROR;
  }
}

extern "C" sqfv_status sqfv_lease_view(const sqfv_lease *lease,
                                       sqfv_span *out_payload) {
  if (!lease || !out_payload)
    return SQFV_INVALID_ARGUMENT;
  *out_payload = {lease->data->payload.data(),
                  static_cast<uint64_t>(lease->data->payload.size())};
  return SQFV_OK;
}

extern "C" sqfv_status
sqfv_lease_descriptor_view(const sqfv_lease *lease,
                           sqfv_descriptor *out_descriptor) {
  if (!lease || !out_descriptor)
    return SQFV_INVALID_ARGUMENT;
  if (!validate_output(out_descriptor->struct_size,
                       out_descriptor->abi_version,
                       sizeof(*out_descriptor)))
    return SQFV_UNSUPPORTED_ABI;
  *out_descriptor = lease->data->descriptor.as_view();
  return SQFV_OK;
}

extern "C" sqfv_status
sqfv_lease_content_id(const sqfv_lease *lease,
                      uint8_t out_id[SQFV_BATCH_CONTENT_ID_BYTES]) {
  if (!lease || !out_id)
    return SQFV_INVALID_ARGUMENT;
  std::copy(lease->data->content_id.begin(), lease->data->content_id.end(),
            out_id);
  return SQFV_OK;
}

extern "C" void sqfv_lease_release(sqfv_lease *lease) { release_lease(lease); }

extern "C" sqfv_status sqfv_port_add(sqfv_context *context,
                                     const sqfv_port_config *config,
                                     sqfv_port **out_port) {
  if (out_port)
    *out_port = nullptr;
  if (!context || !config || !out_port)
    return SQFV_INVALID_ARGUMENT;
  if (config->struct_size != sizeof(sqfv_port_config) ||
      config->abi_version != SQFV_BATCH_ABI_VERSION)
    return SQFV_UNSUPPORTED_ABI;
  const auto binding_status = checked_binding(config->binding);
  if (binding_status != SQFV_OK)
    return binding_status;
  if (!required_span(config->partition) ||
      !nonzero_generation(config->producer_generation) ||
      config->outstanding_byte_credit == 0 ||
      config->max_pending_entries == 0 ||
      config->max_pending_entries > kMaxPending || config->reserved != 0)
    return SQFV_INVALID_ARGUMENT;
  const auto state = context->state;
  const uint64_t cost = port_reservation_bytes(*config);
  bool claimed = false;
  try {
    if (!state->reserve(cost))
      return SQFV_LIMIT;
    Reservation charge(state, cost);
    if (!state->claim_port())
      return SQFV_LIMIT;
    claimed = true;
    auto port_state = std::make_shared<PortState>(std::move(charge), *config);
    const auto handle_cost = handle_reservation_bytes<sqfv_port>();
    if (!state->reserve(handle_cost)) {
      state->release_port();
      return SQFV_LIMIT;
    }
    Reservation handle_charge(state, handle_cost);
    auto handle = std::make_unique<sqfv_port>();
    handle->reservation = std::move(handle_charge);
    handle->state = std::move(port_state);
    *out_port = handle.release();
    return SQFV_OK;
  } catch (const std::bad_alloc &) {
    if (claimed)
      state->release_port();
    return SQFV_NO_MEMORY;
  } catch (...) {
    if (claimed)
      state->release_port();
    return SQFV_INTERNAL_ERROR;
  }
}

extern "C" void sqfv_port_destroy(sqfv_port *port) {
  if (!port)
    return;
  try {
    auto state = port->state;
    {
      std::lock_guard lock(state->mutex);
      state->closed = true;
    }
    for (;;) {
      sqfv_lease *pending = nullptr;
      {
        std::lock_guard lock(state->mutex);
        if (state->pending == 0)
          break;
        pending = state->queue[state->head];
        state->queue[state->head] = nullptr;
        state->head = (state->head + 1) % state->max_pending;
        --state->pending;
      }
      release_lease(pending);
    }
    state->reservation.context->release_port();
    delete port;
  } catch (...) {
    // Destruction cannot signal failure through this ABI. Keep the release
    // obligation visible by terminating rather than silently leaking credit.
    std::terminate();
  }
}

extern "C" sqfv_status sqfv_port_offer(sqfv_port *port,
                                       const sqfv_batch *batch) {
  try {
    if (!port || !batch)
      return SQFV_INVALID_ARGUMENT;
    const auto &state = port->state;
    const auto &data = batch->data;
    if (state->reservation.context.get() != data->reservation.context.get())
      return SQFV_INVALID_ARGUMENT;
    std::lock_guard lock(state->mutex);
    if (state->closed)
      return SQFV_CLOSED;
    const auto descriptor = data->descriptor.as_view();
    if (!same(descriptor.binding.access_scope, state->binding.access_scope))
      return SQFV_SCOPE_MISMATCH;
    if (!state->binding.exactly(descriptor.binding) ||
        !same(descriptor.partition, state->partition) ||
        !std::equal(state->generation.begin(), state->generation.end(),
                    descriptor.producer_generation))
      return SQFV_BINDING_MISMATCH;
    const auto cursor_status = port_cursor_result(*state, *data);
    if (cursor_status != SQFV_OK)
      return cursor_status;
    const uint64_t size = data->payload.size();
    if (size > state->credit)
      return SQFV_LIMIT;
    if (state->pending == state->max_pending ||
        size > state->credit - state->outstanding)
      return SQFV_BLOCKED;
    sqfv_lease *lease = nullptr;
    try {
      const auto result = create_lease(data, state, &lease);
      if (result != SQFV_OK)
        return result;
    } catch (const std::bad_alloc &) {
      return SQFV_NO_MEMORY;
    } catch (...) {
      return SQFV_INTERNAL_ERROR;
    }
    const auto tail = (state->head + state->pending) % state->max_pending;
    state->queue[tail] = lease;
    ++state->pending;
    state->outstanding += size;
    state->last_sequence = descriptor.batch_sequence;
    state->last_id = data->content_id;
    state->has_last = true;
    if (state->next_sequence == UINT64_MAX)
      state->sequence_exhausted = true;
    else
      ++state->next_sequence;
    return SQFV_OK;
  } catch (const std::bad_alloc &) {
    return SQFV_NO_MEMORY;
  } catch (...) {
    return SQFV_INTERNAL_ERROR;
  }
}

extern "C" sqfv_status sqfv_port_take(sqfv_port *port, sqfv_lease **out_lease) {
  if (out_lease)
    *out_lease = nullptr;
  try {
    if (!port || !out_lease)
      return SQFV_INVALID_ARGUMENT;
    const auto &state = port->state;
    std::lock_guard lock(state->mutex);
    if (state->closed)
      return SQFV_CLOSED;
    if (state->pending == 0)
      return SQFV_EMPTY;
    *out_lease = state->queue[state->head];
    state->queue[state->head] = nullptr;
    state->head = (state->head + 1) % state->max_pending;
    --state->pending;
    return SQFV_OK;
  } catch (const std::bad_alloc &) {
    return SQFV_NO_MEMORY;
  } catch (...) {
    return SQFV_INTERNAL_ERROR;
  }
}

extern "C" sqfv_status sqfv_port_cancel(sqfv_port *port,
                                        uint64_t batch_sequence) {
  try {
    if (!port)
      return SQFV_INVALID_ARGUMENT;
    const auto &state = port->state;
    sqfv_lease *cancelled = nullptr;
    {
      std::lock_guard lock(state->mutex);
      if (state->closed)
        return SQFV_CLOSED;
      for (uint32_t offset = 0; offset < state->pending; ++offset) {
        const auto index = (state->head + offset) % state->max_pending;
        if (state->queue[index]->data->descriptor.sequence != batch_sequence)
          continue;
        cancelled = state->queue[index];
        for (uint32_t shift = offset; shift + 1 < state->pending; ++shift) {
          const auto here = (state->head + shift) % state->max_pending;
          const auto after = (state->head + shift + 1) % state->max_pending;
          state->queue[here] = state->queue[after];
        }
        const auto tail =
            (state->head + state->pending - 1) % state->max_pending;
        state->queue[tail] = nullptr;
        --state->pending;
        break;
      }
    }
    if (!cancelled)
      return SQFV_EMPTY;
    release_lease(cancelled);
    return SQFV_OK;
  } catch (const std::bad_alloc &) {
    return SQFV_NO_MEMORY;
  } catch (...) {
    return SQFV_INTERNAL_ERROR;
  }
}

extern "C" sqfv_status sqfv_port_get_stats(const sqfv_port *port,
                                           sqfv_port_stats *out_stats) {
  try {
    if (!port || !out_stats)
      return SQFV_INVALID_ARGUMENT;
    if (!validate_output(out_stats->struct_size, out_stats->abi_version,
                         sizeof(*out_stats)))
      return SQFV_UNSUPPORTED_ABI;
    const auto &state = port->state;
    std::lock_guard lock(state->mutex);
    out_stats->next_sequence = state->next_sequence;
    out_stats->outstanding_bytes = state->outstanding;
    out_stats->outstanding_byte_credit = state->credit;
    out_stats->pending_entries = state->pending;
    out_stats->max_pending_entries = state->max_pending;
    out_stats->sequence_exhausted = state->sequence_exhausted ? 1u : 0u;
    out_stats->reserved = 0;
    return SQFV_OK;
  } catch (const std::bad_alloc &) {
    return SQFV_NO_MEMORY;
  } catch (...) {
    return SQFV_INTERNAL_ERROR;
  }
}
