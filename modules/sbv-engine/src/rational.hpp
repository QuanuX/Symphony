#pragma once
#include "detail.hpp"
#include <numeric>
namespace symphony::sbv::detail::rational {
struct R {
  std::int64_t n = 0, d = 1;
};
inline R reduce(R r) {
  need(r.d > 0 && r.n != INT64_MIN, "rational range exceeded");
  auto g = std::gcd(r.n, r.d);
  return {r.n / g, r.d / g};
}
inline std::int64_t mul(std::int64_t a, std::int64_t b) {
  std::int64_t x;
  need(!__builtin_mul_overflow(a, b, &x),
       "exact rational multiplication exceeds int64 range");
  return x;
}
inline std::int64_t add(std::int64_t a, std::int64_t b) {
  std::int64_t x;
  need(!__builtin_add_overflow(a, b, &x),
       "exact rational addition exceeds int64 range");
  return x;
}
inline R times(R a, R b) {
  auto g = std::gcd(a.n, b.d), h = std::gcd(b.n, a.d);
  return reduce({mul(a.n / g, b.n / h), mul(a.d / h, b.d / g)});
}
inline R plus(R a, R b) {
  auto g = std::gcd(a.d, b.d);
  return reduce({add(mul(a.n, b.d / g), mul(b.n, a.d / g)), mul(a.d, b.d / g)});
}
inline R read_ratio(const Json &j) {
  keys(j, {"numerator", "denominator"});
  R r{i64(j.at("numerator")), i64(j.at("denominator"))};
  need(r.n >= -1000000000 && r.n <= 1000000000 && r.d >= 1 && r.d <= 1000000000,
       "input rational range");
  return reduce(r);
}
inline Json wire(R r) {
  return {{"numerator", dec(r.n)}, {"denominator", dec(r.d)}};
}
} // namespace symphony::sbv::detail::rational
