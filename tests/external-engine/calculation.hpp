#pragma once
#include <charconv>
#include <cstdint>
#include <map>
#include <numeric>
#include <stdexcept>
#include <string>
#include <symphony/knowledge/engine/digest.hpp>
#include <symphony/knowledge/engine/json.hpp>
#include <vector>
namespace beta_example {
using J = symphony::knowledge::engine::Json;
inline constexpr const char *id = "beta-empirical-price-example";
inline std::int64_t integer(const J &value) {
  if (!value.is_string())
    throw std::runtime_error("canonical decimal string required");
  const auto s = value.get<std::string>();
  std::int64_t n{};
  const auto [end, error] = std::from_chars(s.data(), s.data() + s.size(), n);
  if (error != std::errc{} || end != s.data() + s.size() ||
      std::to_string(n) != s)
    throw std::runtime_error("noncanonical integer");
  return n;
}
inline J rational(std::int64_t n, std::int64_t d) {
  if (d <= 0)
    throw std::runtime_error("positive denominator required");
  const auto gcd = std::gcd(n, d);
  return J{{"numerator", std::to_string(n / gcd)},
           {"denominator", std::to_string(d / gcd)}};
}
inline J calculate(const std::vector<std::int64_t> &prices) {
  if (prices.empty() || prices.size() > 64)
    throw std::runtime_error("one to 64 trade prices required");
  std::map<std::int64_t, std::int64_t> counts;
  std::int64_t sum = 0;
  for (const auto price : prices) {
    if (price < -1000000000000000LL || price > 1000000000000000LL)
      throw std::runtime_error("price bound");
    ++counts[price];
    sum += price;
  }
  J support = J::array();
  const auto count = static_cast<std::int64_t>(prices.size());
  for (const auto &[price, n] : counts)
    support.push_back(J{{"price_nanos", std::to_string(price)},
                        {"weight", rational(n, count)}});
  return J{{"protocol", "quanux.beta.empirical-price-result.v1"},
           {"engine_id", id},
           {"engine_version", BETA_EXAMPLE_VERSION},
           {"sample_count", std::to_string(count)},
           {"support", support},
           {"mean_price_nanos", rational(sum, count)}};
}
inline std::string evidence(const J &result) {
  return "sha256:" + symphony::knowledge::engine::sha256_hex(result.dump());
}
} // namespace beta_example
