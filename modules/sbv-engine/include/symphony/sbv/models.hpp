#pragma once
#include <map>
#include <span>
#include <symphony/sbv/sbv.hpp>

namespace symphony::sbv {
struct ObservedTrade {
  std::uint64_t source_ordinal{}, available_ns{};
  std::int64_t price_nanos{};
  std::uint32_t size{};
};
// The caller owns the immutable backing storage throughout evaluate().
// Future observations are retrospective inputs, never strategy-visible inputs.
struct ModelFrame {
  std::string signal_id;
  std::uint64_t source_ordinal{}, available_ns{}, horizon_end_ns{};
  std::int64_t anchor_price_nanos{};
  bool anchor_is_observed_trade{}, horizon_within_observed_span{};
  std::span<const ObservedTrade> future_trades;
};
// Admission checks a complete, exact signal-id set for supplied outcomes.
// Instances are immutable after construction and safe for concurrent
// evaluation. Model claims remain attributable input, not proof of calibration
// or causality.
class AdmittedModel {
public:
  AdmittedModel(Json selection, const std::vector<std::string> &signal_ids);
  Json evaluate(const ModelFrame &, std::int64_t deadline_ms) const;
  std::uint64_t horizon_ns() const;

private:
  Json selection_;
  std::map<std::string, Json> supplied_;
};
Json model_catalogue();
} // namespace symphony::sbv
