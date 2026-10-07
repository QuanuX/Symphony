#pragma once
#include "detail.hpp"
#include <memory>
#include <span>

namespace symphony::sbv::detail {
// Optional exact SQV local owner profile. No provider acquisition occurs.
Json source_retain(const Json &, std::int64_t);
Json source_export(const Json &, std::int64_t);
Json source_owner_profile();
// Pure retained-result correspondence checks: no ancestor/store/file access,
// source authentication, or claim to reverify the original bytes.
Json retained_delivery_identity(const Json &retained_source,
                                const Json &delivery_evidence);

// The original span borrows retained owner storage. Complete verification and
// decode into Dataset-owned storage before calling after_admission().
// Destruction never acknowledges processing. Calls/move/destruction must not
// race.
class RetainedOriginal final {
public:
  explicit RetainedOriginal(const Json &retained_source, std::int64_t);
  ~RetainedOriginal();
  RetainedOriginal(RetainedOriginal &&) noexcept;
  RetainedOriginal &operator=(RetainedOriginal &&) noexcept;
  RetainedOriginal(const RetainedOriginal &) = delete;
  RetainedOriginal &operator=(const RetainedOriginal &) = delete;
  [[nodiscard]] std::span<const std::uint8_t> original_bytes() const;
  [[nodiscard]] const Json &identity() const;
  // Single use. Returns actual delivery/ack evidence, then releases every owner
  // handle. The prior original_bytes() span is invalid after this call.
  [[nodiscard]] Json after_admission();

private:
  struct Impl;
  std::unique_ptr<Impl> impl_;
  friend Json source_export(const Json &, std::int64_t);
};
} // namespace symphony::sbv::detail
