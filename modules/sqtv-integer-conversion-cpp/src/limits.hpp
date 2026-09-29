#pragma once
#include <symphony/sqtv/integer_conversion.hpp>
namespace symphony::sqtv::detail {
inline bool valid_limits(const Limits &l) noexcept {
  return l.max_elements > 0 && l.max_elements <= (1ULL << 23) &&
         l.max_input_bytes > 0 && l.max_input_bytes <= (64ULL << 20) &&
         l.max_output_bytes > 0 && l.max_output_bytes <= (64ULL << 20);
}
} // namespace symphony::sqtv::detail
