#pragma once

#include <cstddef>
#include <cstdint>
#include <memory>
#include <span>
#include <string>
#include <utility>
#include <vector>

namespace symphony::sqv::prototype {

// Development-only in-process contract. This is not an installed ABI or an
// admitted SQV wire protocol. Source ordering is deliberately independent of
// the internal cursor described by generation/partition/first/last.
struct Descriptor {
  std::string dataset_revision;
  std::string schema_name;
  std::string schema_version;
  std::string representation;
  std::string access_scope;
  std::string provenance;
  std::string source_position;
  std::uint64_t generation = 0;
  std::uint64_t first = 0;
  std::uint64_t last = 0;
  std::uint32_t partition = 0;
  std::uint32_t record_count = 0;

  bool operator==(const Descriptor&) const = default;
};

constexpr std::size_t max_payload_bytes = 1024 * 1024;
constexpr std::size_t max_header_bytes = 4096;

class ReadLease {
 public:
  std::span<const std::uint8_t> bytes() const noexcept { return *payload_; }

 private:
  friend class PreparedBatch;
  explicit ReadLease(std::shared_ptr<const std::vector<std::uint8_t>> payload)
      : payload_(std::move(payload)) {}
  std::shared_ptr<const std::vector<std::uint8_t>> payload_;
};

class PreparedBatch {
 public:
  PreparedBatch() = default;

  // Copies mutable preparation bytes before publication. The returned payload
  // has no mutation API and stays alive until the last reader releases it.
  static PreparedBatch prepare(Descriptor descriptor,
                               std::span<const std::uint8_t> payload);

  const Descriptor& descriptor() const noexcept { return descriptor_; }
  std::weak_ptr<const std::vector<std::uint8_t>> lifetime() const noexcept {
    return payload_;
  }

  // A lease retains actual read lifetime until it is destroyed. A span
  // returned by a lease must not outlive that lease. Exact scope equality is
  // a guard for trusted in-process callers only.
  // Cross-process/private access needs a separate admitted authority boundary.
  ReadLease acquire(const std::string& requested_scope) const;

 private:
  PreparedBatch(Descriptor descriptor,
                std::shared_ptr<const std::vector<std::uint8_t>> payload);
  Descriptor descriptor_;
  std::shared_ptr<const std::vector<std::uint8_t>> payload_;
};

// Development frame v1: big-endian integers, bounded ASCII metadata, raw
// bytes, no compression or integrity/authentication claim. No implicit schema
// resolution or provider-version upgrade is performed.
std::vector<std::uint8_t> encode(const PreparedBatch& batch);
PreparedBatch decode(std::span<const std::uint8_t> frame);

}  // namespace symphony::sqv::prototype
