#pragma once
// Independent C++26 header-only backend/tensor capability contract, v1.
// No vendor headers, runtime initialization, data copy or device call.
#include <algorithm>
#include <cstdint>
#include <limits>
#include <memory>
#include <stdexcept>
#include <string>
#include <vector>
namespace symphony::sbv::interop {
inline constexpr auto contract_version = "symphony.sbv.interop.v1";
inline void require(bool b, const char *message) {
  if (!b)
    throw std::invalid_argument(message);
}
inline std::uint64_t add(std::uint64_t a, std::uint64_t b) {
  require(b <= UINT64_MAX - a, "tensor extent overflow");
  return a + b;
}
inline std::uint64_t mul(std::uint64_t a, std::uint64_t b) {
  require(!a || b <= UINT64_MAX / a, "tensor extent overflow");
  return a * b;
}
struct BufferDescriptor {
  std::string dtype, device, owner_id, stream_id;
  std::uint64_t storage_bytes{}, byte_offset{};
  std::vector<std::uint64_t> shape, byte_strides;
};
struct BufferAdmission {
  std::uint64_t element_bytes{}, required_bytes{};
  bool empty{};
};
inline BufferAdmission admit_buffer(const BufferDescriptor &b) {
  std::uint64_t width =
      b.dtype == "u8" || b.dtype == "i8" || b.dtype == "bool" ? 1
      : b.dtype == "f16" || b.dtype == "bf16" || b.dtype == "i16" ||
              b.dtype == "u16"
          ? 2
      : b.dtype == "f32" || b.dtype == "i32" || b.dtype == "u32" ? 4
      : b.dtype == "f64" || b.dtype == "i64" || b.dtype == "u64" ? 8
                                                                 : 0;
  require(width > 0, "unsupported dtype");
  require(b.device == "cpu" || b.device == "cuda" || b.device == "tensor",
          "unknown device domain");
  require(!b.owner_id.empty() && b.owner_id.size() <= 256 &&
              !b.stream_id.empty() && b.stream_id.size() <= 256,
          "owner/stream identity required");
  require(b.shape.size() <= 16 && b.shape.size() == b.byte_strides.size(),
          "shape/stride rank mismatch");
  require(b.byte_offset <= b.storage_bytes && b.byte_offset % width == 0,
          "invalid tensor offset/alignment");
  bool empty = std::find(b.shape.begin(), b.shape.end(), 0) != b.shape.end();
  std::uint64_t extent = b.byte_offset;
  for (std::size_t i = 0; i < b.shape.size(); ++i) {
    require(b.byte_strides[i] % width == 0, "unaligned tensor stride");
    if (!empty)
      extent = add(extent, mul(b.shape[i] - 1, b.byte_strides[i]));
  }
  if (!empty)
    extent = add(extent, width);
  require(extent <= b.storage_bytes, "tensor storage is too short");
  return {width, extent, empty};
}
enum class CompletionState { pending, ready, failed, cancelled };
// Adapter owns the implementation and explicitly reports cancellation support.
// wait_until uses a caller agreed monotonic nanosecond clock; no destructor
// waits.
struct Completion {
  virtual ~Completion() = default;
  virtual CompletionState state() const noexcept = 0;
  virtual CompletionState wait_until(std::uint64_t monotonic_deadline_ns) = 0;
  virtual bool request_cancel() noexcept = 0;
};
struct TensorLease {
  BufferDescriptor descriptor;
  std::shared_ptr<const void> owner;
  const void *base = nullptr;
  std::shared_ptr<Completion> completion;
  // The adapter promises that owner covers base/storage for the lease lifetime.
  // Admission proves descriptor arithmetic, never validity of a foreign
  // pointer.
  void validate() const {
    admit_buffer(descriptor);
    require(bool(owner) && bool(completion), "lease owner/completion required");
    require(base || descriptor.storage_bytes == 0, "lease base required");
  }
  const void *ready_host_base() const {
    validate();
    require(descriptor.device == "cpu",
            "device buffer requires selected adapter");
    require(completion->state() == CompletionState::ready,
            "buffer completion is not ready");
    return base;
  }
};
struct BackendRequest {
  std::string backend, fallback, limit_policy, scheduling;
  std::uint64_t workers{}, memory_bytes{}, declared_worker_limit{},
      declared_memory_limit{};
};
struct BackendPlan {
  std::string status, backend, reason;
  std::uint64_t workers{}, memory_bytes{};
  bool fallback_used{};
};
inline BackendPlan plan_backend(const BackendRequest &r) {
  require(r.backend == "cpu" || r.backend == "cuda" || r.backend == "tensor",
          "unknown backend");
  require(r.fallback.empty() || r.fallback == "cpu",
          "unsupported explicit fallback");
  require(r.limit_policy == "reject" || r.limit_policy == "reduce",
          "invalid limit policy");
  require(r.scheduling == "independent" || r.scheduling == "serial_state",
          "invalid scheduling contract");
  require(r.workers > 0 && r.memory_bytes > 0 && r.declared_worker_limit > 0 &&
              r.declared_memory_limit > 0,
          "positive resource bounds required");
  if (r.backend != "cpu" && r.fallback.empty())
    return {"unavailable",
            r.backend,
            "selected adapter runtime is not installed",
            0,
            0,
            false};
  auto workers = std::min(
      {r.workers, r.declared_worker_limit, std::uint64_t{64},
       r.scheduling == "serial_state" ? std::uint64_t{1} : std::uint64_t{64}});
  auto memory = std::min(r.memory_bytes, r.declared_memory_limit);
  if (r.limit_policy == "reject" &&
      (workers != r.workers || memory != r.memory_bytes))
    return {"unavailable",
            "cpu",
            "requested resources exceed selected limits or dependency-safe "
            "concurrency",
            0,
            0,
            r.backend != "cpu"};
  return {"planned", "cpu",  "planning only; no reservation or execution",
          workers,   memory, r.backend != "cpu"};
}
} // namespace symphony::sbv::interop
