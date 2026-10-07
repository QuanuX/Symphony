#include "exact_id_index.hpp"
#include <limits>
#include <stdexcept>
#include <utility>

namespace symphony::sbv::detail {
namespace {
std::uint64_t checked_add(std::uint64_t a, std::uint64_t b) {
  if (b > std::numeric_limits<std::uint64_t>::max() - a)
    throw std::length_error("exact ID index accounting exceeds uint64 range");
  return a + b;
}
} // namespace

ExactIdIndex::ExactIdIndex(std::optional<std::uint64_t> max_tracked_bytes,
                           std::function<void()> checkpoint)
    : max_tracked_bytes_(max_tracked_bytes), checkpoint_(std::move(checkpoint)) {}

ExactIdInsert ExactIdIndex::insert(std::string_view key) {
  if (checkpoint_)
    checkpoint_();
  if (const auto found = entries_.find(key); found != entries_.end())
    return {false, found->second};
  if constexpr (sizeof(std::size_t) > sizeof(std::uint64_t)) {
    if (key.size() > std::numeric_limits<std::uint64_t>::max())
      throw std::length_error("exact ID key size exceeds uint64 range");
  }
  const auto bytes = static_cast<std::uint64_t>(key.size());
  const auto next_count = checked_add(count_, 1);
  const auto next_keys = checked_add(key_bytes_, bytes);
  // Eight logical bytes describe the exact uint64 ordinal. This is an
  // accounting convention, not sizeof(std::map::node_type) or heap usage.
  const auto next_tracked = checked_add(tracked_bytes_, checked_add(bytes, 8));
  if (max_tracked_bytes_ && next_tracked > *max_tracked_bytes_)
    throw std::length_error("selected exact ID tracked-byte budget exceeded");
  const auto ordinal = count_;
  const auto inserted = entries_.emplace(std::string(key), ordinal).second;
  if (!inserted)
    throw std::logic_error("single-caller exact ID insertion contradiction");
  // Map insertion has a strong exception guarantee. Counters change only
  // after allocation/key ownership succeeds, and the checked updates cannot
  // subsequently throw.
  count_ = next_count;
  key_bytes_ = next_keys;
  tracked_bytes_ = next_tracked;
  return {true, ordinal};
}

std::optional<std::uint64_t> ExactIdIndex::find(std::string_view key) const {
  if (checkpoint_)
    checkpoint_();
  const auto found = entries_.find(key);
  if (found == entries_.end())
    return {};
  return found->second;
}
std::uint64_t ExactIdIndex::size() const noexcept { return count_; }
std::uint64_t ExactIdIndex::retained_key_bytes() const noexcept { return key_bytes_; }
std::uint64_t ExactIdIndex::tracked_bytes() const noexcept { return tracked_bytes_; }
knowledge::engine::Json ExactIdIndex::evidence() const {
  using Json = knowledge::engine::Json;
  return {{"backend", "memory_exact_ordered_map.v1"},
          {"entries", std::to_string(count_)},
          {"retained_key_bytes", std::to_string(key_bytes_)},
          {"tracked_bytes", std::to_string(tracked_bytes_)},
          {"max_tracked_bytes", max_tracked_bytes_
                                    ? Json(std::to_string(*max_tracked_bytes_))
                                    : Json(nullptr)},
          {"tracked_bytes_formula", "sum(exact_key_byte_lengths) + 8 * entries"},
          {"ordinal_semantics", "first_successful_insertion"},
          {"accounting_scope", "logical retained key and uint64 ordinal bytes; "
                               "excludes container nodes, pointers, allocator "
                               "overhead, capacities, temporary copies and RSS"},
          {"aggregate_state", "O(entries + retained_key_bytes)"},
          {"spill", "not_supported"},
          {"automatic_limit", false}};
}
} // namespace symphony::sbv::detail
