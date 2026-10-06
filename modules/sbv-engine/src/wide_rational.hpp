#pragma once
#include "rational.hpp"
#include <algorithm>
namespace symphony::sbv::detail::wide_rational {
using I = __int128_t;
using U = __uint128_t;
inline constexpr I max = static_cast<I>((~U{0}) >> 1);
struct R {
  I n = 0, d = 1;
};
inline I absolute(I x) {
  need(x != -max - 1, "wide minimum excluded");
  return x < 0 ? -x : x;
}
inline I gcd(I a, I b) {
  a = absolute(a);
  b = absolute(b);
  while (b) {
    I t = a % b;
    a = b;
    b = t;
  }
  return a;
}
inline I add(I a, I b) {
  I n;
  need(!__builtin_add_overflow(a, b, &n) && n != -max - 1,
       "wide addition overflow");
  return n;
}
inline I mul(I a, I b) {
  I n;
  need(!__builtin_mul_overflow(a, b, &n) && n != -max - 1,
       "wide multiplication overflow");
  return n;
}
inline R reduce(R r) {
  need(r.d > 0, "positive wide denominator required");
  auto g = gcd(r.n, r.d);
  return {r.n / g, r.d / g};
}
inline R plus(R a, R b) {
  auto g = gcd(a.d, b.d);
  return reduce({add(mul(a.n, b.d / g), mul(b.n, a.d / g)), mul(a.d, b.d / g)});
}
inline R times(R a, R b) {
  auto g = gcd(a.n, b.d), h = gcd(b.n, a.d);
  return reduce({mul(a.n / g, b.n / h), mul(a.d / h, b.d / g)});
}
inline R divide(R a, R b) {
  need(b.n != 0, "zero divisor");
  return times(a, {b.n < 0 ? -b.d : b.d, absolute(b.n)});
}
inline std::string decimal(I n) {
  bool neg = n < 0;
  n = absolute(n);
  std::string s;
  do {
    s.push_back(static_cast<char>('0' + n % 10));
    n /= 10;
  } while (n);
  if (neg)
    s.push_back('-');
  std::reverse(s.begin(), s.end());
  return s;
}
inline I integer(const Json &j) {
  auto s = str(j);
  need(!s.empty() && s.size() <= 40, "wide decimal bound");
  bool neg = s[0] == '-';
  I n = 0;
  for (std::size_t i = neg ? 1 : 0; i < s.size(); ++i) {
    need(s[i] >= '0' && s[i] <= '9', "wide decimal required");
    n = add(mul(n, 10), s[i] - '0');
  }
  n = neg ? -n : n;
  need(decimal(n) == s, "canonical wide decimal required");
  return n;
}
inline R parameter(const Json &j) {
  auto r = rational::read_ratio(j);
  return {r.n, r.d};
}
inline Json wire(R r) {
  r = reduce(r);
  return {{"numerator", decimal(r.n)}, {"denominator", decimal(r.d)}};
}
inline bool equal(R a, R b) {
  a = reduce(a);
  b = reduce(b);
  return a.n == b.n && a.d == b.d;
}
} // namespace symphony::sbv::detail::wide_rational
