#pragma once
#include "dataset.hpp"
namespace symphony::sbv::detail {
// Caller validates the result envelope on input. New producers may call this
// before sealing. No digest or producer label authenticates source authorship.
Json census_evidence(const Json &result);
// Validate a retained normalized declaration without reopening its ancestors.
void validate_census_evidence(const Json &);
void validate_census_source(const Json &, const Dataset &, std::int64_t);
Json admit_census(const Json &selection, const Dataset &, Json &reference,
                  std::int64_t deadline_ms);
} // namespace symphony::sbv::detail
