#pragma once
#include "wide_rational.hpp"
#include <optional>
#include <span>
namespace symphony::sbv::detail::formula {
namespace w = wide_rational;
struct Sample {
  std::size_t index;
  w::R value, weight;
};
inline void validate(const Json &node, bool row, std::int64_t end,
                     unsigned depth = 0) {
  deadline(end);
  need(depth < 64 && node.is_object() && node.contains("op"),
       "formula node/depth invalid");
  const auto op = str(node.at("op"));
  if (op == "constant") {
    keys(node, {"op", "value"});
    (void)w::artifact(node.at("value"));
  } else if (op == "row_value" || op == "row_weight") {
    keys(node, {"op"});
    need(row, "formula row variable outside reduction");
  } else if (op == "count") {
    keys(node, {"op"});
    need(!row, "formula count belongs to aggregate scope");
  } else if (op == "sum" || op == "minimum" || op == "maximum") {
    keys(node, {"op", "value"});
    need(!row,
         "nested formula reductions unsupported in exact_series_formula_v1");
    validate(node.at("value"), true, end, depth + 1);
  } else if (op == "negate" || op == "absolute") {
    keys(node, {"op", "value"});
    validate(node.at("value"), row, end, depth + 1);
  } else {
    need(op == "add" || op == "subtract" || op == "multiply" ||
             op == "divide" || op == "min" || op == "max",
         "unknown formula operator");
    keys(node, {"op", "left", "right"});
    validate(node.at("left"), row, end, depth + 1);
    validate(node.at("right"), row, end, depth + 1);
  }
}
struct Value {
  std::optional<w::R> value;
  std::string reason;
};
inline Value evaluate(const Json &node, std::span<const Sample> samples,
                      const Sample *row, std::int64_t end) {
  deadline(end);
  const auto op = str(node.at("op"));
  if (op == "constant")
    return {w::artifact(node.at("value")), {}};
  if (op == "row_value")
    return {row->value, {}};
  if (op == "row_weight")
    return {row->weight, {}};
  if (op == "count")
    return {w::R{static_cast<w::I>(samples.size()), 1}, {}};
  if (op == "sum" || op == "minimum" || op == "maximum") {
    if (samples.empty())
      return op == "sum"
                 ? Value{w::R{}, {}}
                 : Value{{}, "extremum of empty retained series is undefined"};
    std::optional<w::R> result;
    for (const auto &s : samples) {
      auto v = evaluate(node.at("value"), samples, &s, end);
      if (!v.value)
        return v;
      if (!result)
        result = *v.value;
      else if (op == "sum")
        result = w::plus(*result, *v.value);
      else if ((op == "minimum" && w::compare(*v.value, *result) < 0) ||
               (op == "maximum" && w::compare(*v.value, *result) > 0))
        result = *v.value;
    }
    return {result, {}};
  }
  if (op == "negate" || op == "absolute") {
    auto v = evaluate(node.at("value"), samples, row, end);
    if (v.value)
      v.value->n = op == "negate" ? -v.value->n : w::absolute(v.value->n);
    return v;
  }
  auto a = evaluate(node.at("left"), samples, row, end);
  if (!a.value)
    return a;
  auto b = evaluate(node.at("right"), samples, row, end);
  if (!b.value)
    return b;
  if (op == "add")
    return {w::plus(*a.value, *b.value), {}};
  if (op == "subtract")
    return {w::plus(*a.value, {-b.value->n, b.value->d}), {}};
  if (op == "multiply")
    return {w::times(*a.value, *b.value), {}};
  if (op == "divide")
    return b.value->n == 0 ? Value{{}, "formula division by zero is undefined"}
                           : Value{w::divide(*a.value, *b.value), {}};
  const auto order = w::compare(*a.value, *b.value);
  return {op == "min" ? (order <= 0 ? *a.value : *b.value)
                      : (order >= 0 ? *a.value : *b.value),
          {}};
}
} // namespace symphony::sbv::detail::formula
