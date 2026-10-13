#pragma once
#include "rational.hpp"
#include <optional>
namespace symphony::sbv::detail::depth_sweep {
namespace r = rational;
struct Level {
  std::int64_t price;
  std::uint64_t original, remaining;
};
struct Snapshot {
  std::uint64_t ordinal{}, time{};
  bool available{}, flagged{};
  std::string reason, market;
  std::vector<Level> bids, asks;
  std::uint64_t bid_count{}, ask_count{};
};
struct Order {
  Json choice;
  std::string id, signal;
  std::uint64_t frame_ordinal, quantity, signal_ordinal, signal_time;
  std::int64_t anchor;
  bool buy, fok;
  std::optional<std::int64_t> limit;
  r::R participation;
  std::optional<r::R> activation;
};
Snapshot frame(const Json &);
Json simulate(const Order &, Snapshot *, const Json &, const std::string &,
              std::int64_t);
} // namespace symphony::sbv::detail::depth_sweep
