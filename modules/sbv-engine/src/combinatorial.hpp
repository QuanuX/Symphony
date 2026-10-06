#pragma once
#include "detail.hpp"
#include <algorithm>
#include <limits>

namespace symphony::sbv::detail::combinatorial {
// Exact nonnegative integers for combination counts/ranks. Dataset and page
// sizes still use host containers; the space being addressed need not fit them.
class Natural {
  static constexpr std::uint64_t radix = 1000000000;
  std::vector<std::uint32_t> limbs{0};
  void trim() {
    while (limbs.size() > 1 && limbs.back() == 0)
      limbs.pop_back();
  }

public:
  explicit Natural(std::uint64_t n = 0) {
    limbs.clear();
    do {
      limbs.push_back(n % radix);
      n /= radix;
    } while (n);
  }
  explicit Natural(const Json &j) {
    const auto s = str(j);
    need(!s.empty() && (s.size() == 1 || s[0] != '0') &&
             std::all_of(s.begin(), s.end(),
                         [](char c) { return c >= '0' && c <= '9'; }),
         "canonical nonnegative combination rank required");
    limbs.clear();
    for (auto end = s.size(); end != 0;) {
      const auto start = end > 9 ? end - 9 : 0;
      std::uint32_t n = 0;
      for (auto i = start; i < end; ++i)
        n = n * 10 + (s[i] - '0');
      limbs.push_back(n);
      end = start;
    }
  }
  std::string text() const {
    auto out = dec(limbs.back());
    for (auto i = limbs.size() - 1; i > 0; --i) {
      const auto s = dec(limbs[i - 1]);
      out += std::string(9 - s.size(), '0') + s;
    }
    return out;
  }
  int compare(const Natural &b) const {
    if (limbs.size() != b.limbs.size())
      return limbs.size() < b.limbs.size() ? -1 : 1;
    for (auto i = limbs.size(); i > 0; --i)
      if (limbs[i - 1] != b.limbs[i - 1])
        return limbs[i - 1] < b.limbs[i - 1] ? -1 : 1;
    return 0;
  }
  void subtract(const Natural &b) {
    need(compare(b) >= 0, "negative combination rank");
    std::int64_t borrow = 0;
    for (std::size_t i = 0; i < limbs.size(); ++i) {
      auto n = static_cast<std::int64_t>(limbs[i]) - borrow -
               (i < b.limbs.size() ? b.limbs[i] : 0);
      borrow = n < 0;
      limbs[i] = static_cast<std::uint32_t>(n + (borrow ? radix : 0));
    }
    trim();
  }
  void increment() {
    for (auto &x : limbs) {
      if (++x != radix)
        return;
      x = 0;
    }
    limbs.push_back(1);
  }
  void multiply(std::uint64_t n) {
    __uint128_t carry = 0;
    for (auto &x : limbs) {
      const auto value = static_cast<__uint128_t>(x) * n + carry;
      x = value % radix;
      carry = value / radix;
    }
    while (carry) {
      limbs.push_back(carry % radix);
      carry /= radix;
    }
    trim();
  }
  void divide_exact(std::uint64_t n) {
    need(n > 0, "zero combination divisor");
    __uint128_t remainder = 0;
    for (auto i = limbs.size(); i > 0; --i) {
      const auto value = remainder * radix + limbs[i - 1];
      limbs[i - 1] = value / n;
      remainder = value % n;
    }
    need(remainder == 0, "inexact combination arithmetic");
    trim();
  }
};
inline Natural choose(std::size_t n, std::size_t k, std::int64_t end) {
  need(k <= n, "combination size exceeds group count");
  k = std::min(k, n - k);
  Natural result(1);
  for (std::size_t i = 1; i <= k; ++i) {
    deadline(end);
    result.multiply(n - k + i);
    result.divide_exact(i);
  }
  return result;
}
// Lexicographic unranking skips whole subtrees, never enumerating prior folds.
inline std::vector<std::size_t> unrank(std::size_t n, std::size_t k,
                                       Natural rank, std::int64_t end) {
  std::vector<std::size_t> out;
  auto block = choose(n - 1, k - 1, end);
  for (std::size_t candidate = 0; k != 0; ++candidate) {
    deadline(end);
    need(candidate < n, "combination rank outside space");
    if (rank.compare(block) < 0) {
      out.push_back(candidate);
      --k;
      if (!k)
        break;
      block.multiply(k);
    } else {
      rank.subtract(block);
      block.multiply(n - candidate - k);
    }
    block.divide_exact(n - candidate - 1);
  }
  return out;
}
inline void next(std::vector<std::size_t> &v, std::size_t n) {
  for (auto i = v.size(); i > 0; --i)
    if (v[i - 1] < n - v.size() + i - 1) {
      ++v[i - 1];
      for (auto j = i; j < v.size(); ++j)
        v[j] = v[j - 1] + 1;
      return;
    }
  need(false, "combination successor beyond final fold");
}
} // namespace symphony::sbv::detail::combinatorial
