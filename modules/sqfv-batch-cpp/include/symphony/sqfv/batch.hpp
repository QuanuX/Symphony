#ifndef SYMPHONY_SQFV_BATCH_HPP
#define SYMPHONY_SQFV_BATCH_HPP

#include <array>
#include <cstddef>
#include <cstdint>
#include <memory>
#include <span>
#include <string>
#include <string_view>

namespace symphony::sqfv {

inline constexpr std::size_t content_id_bytes = 32;
inline constexpr std::size_t generation_bytes = 16;
using ContentId = std::array<std::uint8_t, content_id_bytes>;
using Generation = std::array<std::uint8_t, generation_bytes>;
using ByteView = std::span<const std::uint8_t>;
using MutableBytes = std::span<std::uint8_t>;

// Fallible operations return a status. Outputs change only on success.
// A borrowed span or reference remains valid only while its retaining handle
// remains live. Do not destroy or move one handle concurrently with its use.
enum class Status : std::uint8_t {
  ok,
  invalid_argument,
  limit,
  no_memory,
  blocked,
  duplicate,
  conflict,
  gap,
  stale,
  scope_mismatch,
  binding_mismatch,
  closed,
  empty,
  corrupt_frame,
  unsupported_frame,
  internal_error,
};

// Each limit is mandatory, finite, and positive. Technical ceilings are in
// the installed SPEC; they are local to this library release.
struct Limits {
  std::uint64_t max_payload_bytes = 0;
  std::uint64_t max_frame_bytes = 0;
  std::uint64_t max_descriptor_bytes = 0;
  std::uint64_t global_allocation_bytes = 0;
  std::uint32_t max_ports = 0;
};

// Strings carry opaque bytes, including embedded NULs. SQFV compares their
// bytes exactly; SQMV owns the meaning of metadata references.
struct Binding {
  std::string metadata_ref;
  std::string dataset_revision;
  std::string schema_version;
  std::string layout_version;
  std::string access_scope;
};

struct Descriptor {
  Binding binding;
  std::string partition;
  std::string source_binding;
  std::string source_position;
  Generation producer_generation{};
  std::uint64_t batch_sequence = 0;
  std::uint64_t record_count = 0;
};

struct PortConfig {
  Binding binding;
  std::string partition;
  Generation producer_generation{};
  std::uint64_t next_sequence = 0;
  std::uint64_t outstanding_byte_credit = 0;
  std::uint32_t max_pending_entries = 0;
};

struct ContextStats {
  std::uint64_t allocation_bytes = 0;
  std::uint64_t peak_allocation_bytes = 0;
  std::uint64_t allocation_limit_bytes = 0;
  std::uint32_t live_ports = 0;
};

struct PortStats {
  std::uint64_t next_sequence = 0;
  std::uint64_t outstanding_bytes = 0;
  std::uint64_t outstanding_byte_credit = 0;
  std::uint32_t pending_entries = 0;
  std::uint32_t max_pending_entries = 0;
  bool sequence_exhausted = false;
};

class Batch;
class Lease;
class Port;
namespace detail {
struct FrameAccess;
struct ContextHandle;
struct BatchHandle;
struct LeaseHandle;
struct PortHandle;
}

class Context final {
 public:
  Context() noexcept;
  ~Context() noexcept;
  Context(Context&&) noexcept;
  Context& operator=(Context&&) noexcept;
  Context(const Context&) = delete;
  Context& operator=(const Context&) = delete;

  [[nodiscard]] explicit operator bool() const noexcept;
  [[nodiscard]] static Status create(const Limits&, Context& out) noexcept;
  [[nodiscard]] Status prepare_copy(const Descriptor&, ByteView payload,
                                    Batch& out) const noexcept;
  [[nodiscard]] Status add_port(const PortConfig&, Port& out) const noexcept;
  [[nodiscard]] Status stats(ContextStats& out) const noexcept;

 private:
  std::unique_ptr<detail::ContextHandle> impl_;
  friend struct detail::FrameAccess;
};

// A Batch is an owning, move-only producer handle. Retain explicitly when a
// second handle is needed. A lease may outlive its producer Batch and Context.
class Batch final {
 public:
  Batch() noexcept;
  ~Batch() noexcept;
  Batch(Batch&&) noexcept;
  Batch& operator=(Batch&&) noexcept;
  Batch(const Batch&) = delete;
  Batch& operator=(const Batch&) = delete;

  [[nodiscard]] explicit operator bool() const noexcept;
  [[nodiscard]] Status retain(Batch& out) const noexcept;
  [[nodiscard]] Status acquire(std::string_view access_scope,
                               Lease& out) const noexcept;
  // Requires a nonempty handle (explicit operator bool() is true).
  [[nodiscard]] const Descriptor& descriptor() const noexcept;
  // Requires a nonempty handle (explicit operator bool() is true).
  [[nodiscard]] const ContentId& content_id() const noexcept;

 private:
  std::unique_ptr<detail::BatchHandle> impl_;
  friend class Context;
  friend class Port;
  friend struct detail::FrameAccess;
};

// A Lease holds one immutable payload. A port delivery also holds its byte
// credit until this object is destroyed, moved over, or explicitly reset.
class Lease final {
 public:
  Lease() noexcept;
  ~Lease() noexcept;
  Lease(Lease&&) noexcept;
  Lease& operator=(Lease&&) noexcept;
  Lease(const Lease&) = delete;
  Lease& operator=(const Lease&) = delete;

  [[nodiscard]] explicit operator bool() const noexcept;
  // An empty handle returns an empty span.
  [[nodiscard]] ByteView payload() const noexcept;
  // Requires a nonempty handle (explicit operator bool() is true).
  [[nodiscard]] const Descriptor& descriptor() const noexcept;
  // Requires a nonempty handle (explicit operator bool() is true).
  [[nodiscard]] const ContentId& content_id() const noexcept;
  void reset() noexcept;

 private:
  std::unique_ptr<detail::LeaseHandle> impl_;
  friend class Batch;
  friend class Port;
};

// Acceptance advances a local cursor. Taking or cancelling a pending delivery
// does not roll that cursor back. Destruction cancels pending deliveries;
// already taken leases remain valid until their own destruction.
class Port final {
 public:
  Port() noexcept;
  ~Port() noexcept;
  Port(Port&&) noexcept;
  Port& operator=(Port&&) noexcept;
  Port(const Port&) = delete;
  Port& operator=(const Port&) = delete;

  [[nodiscard]] explicit operator bool() const noexcept;
  [[nodiscard]] Status offer(const Batch&) noexcept;
  [[nodiscard]] Status take(Lease& out) noexcept;
  [[nodiscard]] Status cancel(std::uint64_t batch_sequence) noexcept;
  [[nodiscard]] Status stats(PortStats& out) const noexcept;
  void reset() noexcept;

 private:
  std::unique_ptr<detail::PortHandle> impl_;
  friend class Context;
};

// The SQF1 frame is a bounded, integrity-checked local serialization. The
// caller owns the output buffer; decode publishes no Batch on failure.
[[nodiscard]] Status frame_measure(const Context&, const Batch&,
                                   std::size_t& out_size) noexcept;
[[nodiscard]] Status frame_encode(const Context&, const Batch&,
                                  MutableBytes out, std::size_t& out_size) noexcept;
[[nodiscard]] Status frame_decode(Context&, ByteView frame, Batch& out) noexcept;

} // namespace symphony::sqfv

#endif
