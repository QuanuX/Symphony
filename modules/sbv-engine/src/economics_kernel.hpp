#pragma once
#include "detail.hpp"

namespace symphony::sbv::detail {
// Shared existing finite-support int64 rational economic profile. No storage,
// source join, normalization, aggregation, or implicit selection is performed.
struct EconomicRow {
  Json outcome;
  Json studies;
  std::uint64_t unavailable_studies;
};
class StreamModel;
// Deterministic external-input projection only: this does not replay market
// data, infer execution, or recompute native/observed provider models.
void validate_retained_external_outcome(const Json &, const StreamModel &,
                                        bool historical, std::int64_t);
Json economics_partitioned(const Json &, std::int64_t);
void validate_economic_source_model(const Json &model);
Json economic_study_selections(const Json &);
EconomicRow economic_row(const Json &choice, const Json &model,
                         const Json &studies, bool reject,
                         std::int64_t deadline_ms);
} // namespace symphony::sbv::detail
