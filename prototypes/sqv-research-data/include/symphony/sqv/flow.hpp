#pragma once

#include "symphony/sqv/prepared_batch.hpp"

#include <cstddef>
#include <map>
#include <memory>
#include <optional>
#include <string>

namespace symphony::sqv::prototype {

struct FlowPortState;

enum class OfferResult { accepted, blocked, oversize, scope_mismatch, unknown_port };

// One consumed delivery owns one byte-credit obligation. Destroying it
// returns credit, even if the producer or Fanout object has already gone away.
class Delivery {
 public:
  Delivery(const Delivery&) = delete;
  Delivery& operator=(const Delivery&) = delete;
  Delivery(Delivery&&) noexcept = default;
  Delivery& operator=(Delivery&&) = delete;
  ~Delivery();

  const Descriptor& descriptor() const noexcept { return descriptor_; }
  std::span<const std::uint8_t> bytes() const noexcept { return lease_.bytes(); }

 private:
  friend class Fanout;
  Delivery(Descriptor descriptor, ReadLease lease,
           std::shared_ptr<FlowPortState> state, std::size_t charged_bytes);
  Descriptor descriptor_;
  ReadLease lease_;
  std::shared_ptr<FlowPortState> state_;
  std::size_t charged_bytes_;
};

// Offline in-process proof of independent edge budgets. A blocked offer is
// reported to the caller; it is never silently dropped or called delivered.
// This is not a durable, network, or cross-process delivery contract.
class Fanout {
 public:
  void add_port(std::string id, std::string access_scope,
                std::size_t byte_budget, std::size_t max_pending);
  OfferResult offer(const std::string& id, const PreparedBatch& batch);
  std::optional<Delivery> take(const std::string& id);
  std::size_t outstanding_bytes(const std::string& id) const;

 private:
  std::map<std::string, std::shared_ptr<FlowPortState>> ports_;
};

}  // namespace symphony::sqv::prototype
