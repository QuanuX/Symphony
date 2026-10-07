#pragma once
#include "census.hpp"
#include "logical_value.hpp"
namespace symphony::sbv::detail {
// The selected value owns all backing bytes/reader state. Admission verifies
// the complete outer immutable source, then its selected logical role. It does
// not authenticate an author or reopen a captured provider library.
struct LogicalSelection {
  logical::Value value;
  Json reference;
};
LogicalSelection select_bundle_value(const Json &, std::int64_t);
LogicalSelection select_logical_result(const Json &, std::int64_t);
void validate_logical_reference_lineage(const Json &, const Json &);
void validate_logical_result(const logical::Value &, std::int64_t);
void validate_stream_census(const logical::Value &, std::int64_t);
logical::Value stream_census_evidence(const logical::Value &, std::int64_t);
void validate_stream_census_source(const logical::Value &, const Dataset &,
                                   std::int64_t);
LogicalSelection admit_stream_census(const Json &, const Dataset &,
                                     std::int64_t);
} // namespace symphony::sbv::detail
