#include "symphony/sqv/flow.hpp"

#include <deque>
#include <mutex>
#include <stdexcept>
#include <utility>

namespace symphony::sqv::prototype {

struct Pending {
  Descriptor descriptor;
  ReadLease lease;
  std::size_t charged_bytes;
};

struct FlowPortState {
  std::string access_scope;
  std::size_t byte_budget;
  std::size_t max_pending;
  std::size_t outstanding_bytes = 0;
  std::deque<Pending> pending;
  mutable std::mutex mutex;
};

Delivery::Delivery(Descriptor descriptor, ReadLease lease,
                   std::shared_ptr<FlowPortState> state,
                   std::size_t charged_bytes)
    : descriptor_(std::move(descriptor)), lease_(std::move(lease)),
      state_(std::move(state)), charged_bytes_(charged_bytes) {}

Delivery::~Delivery() {
  if (!state_) return;
  std::lock_guard lock(state_->mutex);
  state_->outstanding_bytes -= charged_bytes_;
}

void Fanout::add_port(std::string id, std::string access_scope,
                      std::size_t byte_budget, std::size_t max_pending) {
  if (id.empty() || access_scope.empty() || byte_budget == 0 || max_pending == 0)
    throw std::invalid_argument("invalid port configuration");
  auto state = std::make_shared<FlowPortState>();
  state->access_scope = std::move(access_scope);
  state->byte_budget = byte_budget;
  state->max_pending = max_pending;
  if (!ports_.emplace(std::move(id), std::move(state)).second)
    throw std::invalid_argument("duplicate port");
}

OfferResult Fanout::offer(const std::string& id, const PreparedBatch& batch) {
  const auto it = ports_.find(id);
  if (it == ports_.end()) return OfferResult::unknown_port;
  const auto& state = it->second;
  if (batch.descriptor().access_scope != state->access_scope)
    return OfferResult::scope_mismatch;
  const auto lease = batch.acquire(state->access_scope);
  const auto bytes = lease.bytes().size();
  std::lock_guard lock(state->mutex);
  if (bytes > state->byte_budget) return OfferResult::oversize;
  if (state->pending.size() >= state->max_pending ||
      bytes > state->byte_budget - state->outstanding_bytes)
    return OfferResult::blocked;
  state->pending.push_back({batch.descriptor(), lease, bytes});
  state->outstanding_bytes += bytes;
  return OfferResult::accepted;
}

std::optional<Delivery> Fanout::take(const std::string& id) {
  const auto it = ports_.find(id);
  if (it == ports_.end()) throw std::invalid_argument("unknown port");
  const auto& state = it->second;
  std::lock_guard lock(state->mutex);
  if (state->pending.empty()) return std::nullopt;
  auto pending = std::move(state->pending.front());
  state->pending.pop_front();
  return Delivery(std::move(pending.descriptor), std::move(pending.lease),
                  state, pending.charged_bytes);
}

std::size_t Fanout::outstanding_bytes(const std::string& id) const {
  const auto it = ports_.find(id);
  if (it == ports_.end()) throw std::invalid_argument("unknown port");
  std::lock_guard lock(it->second->mutex);
  return it->second->outstanding_bytes;
}

}  // namespace symphony::sqv::prototype
