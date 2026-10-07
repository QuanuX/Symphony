#pragma once
#include "logical_value.hpp"
namespace symphony::sbv::detail {
// Validate a retained economics result without reopening any ancestor. The
// caller first admits the enclosing complete sealed logical result. Equality
// of aggregate retained arrays is checked by canonical digest/ordered rows.
logical::Value economic_stream_census(const logical::Value &, std::int64_t);
} // namespace symphony::sbv::detail
