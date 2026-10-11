#pragma once
#include "wide_rational.hpp"
#include <algorithm>
#include <optional>
namespace symphony::sbv::detail::selection {
namespace w = wide_rational;
const Json *at(const Json &row, const std::string &pointer) {
  need(pointer.size() <= 4096 && (pointer.empty() || pointer.front() == '/'),
       "bounded JSON Pointer required");
  const Json::json_pointer p(pointer);
  if (!row.contains(p))
    return nullptr;
  return &row.at(p);
}
void domain(const std::string &type) {
  need(type == "text" || type == "integer" || type == "rational" ||
           type == "boolean",
       "unknown selection type");
}
int compare(const Json &a, const Json &b, const std::string &type) {
  if (type == "text") {
    auto x = str(a), y = str(b);
    return x == y ? 0 : x < y ? -1 : 1;
  }
  if (type == "boolean") {
    need(a.is_boolean() && b.is_boolean(), "boolean selection values required");
    return a == b ? 0 : a == false ? -1 : 1;
  }
  if (type == "integer") {
    auto x = w::integer(a), y = w::integer(b);
    return x == y ? 0 : x < y ? -1 : 1;
  }
  auto parse = [](const Json &j) {
    keys(j, {"numerator", "denominator"});
    return w::reduce(
        {w::integer(j.at("numerator")), w::integer(j.at("denominator"))});
  };
  return w::compare(parse(a), parse(b));
}
struct Predicate {
  std::string pointer, type, op, missing;
  Json value;
};
struct Ordering {
  std::string pointer, type, direction, missing;
};

struct Query {
  std::vector<Predicate> predicates;
  std::vector<Ordering> order;
  std::vector<std::pair<std::string, std::string>> columns;
  explicit Query(const Json &p) {
    need(p.at("filters").is_array() && p.at("order_by").is_array() &&
             p.at("columns").is_array(),
         "selection choices must be arrays");

    for (const auto &f : p.at("filters")) {
      keys(f, {"pointer", "type", "op", "value", "missing"});
      Predicate v{str(f.at("pointer")), str(f.at("type")), str(f.at("op")),
                  str(f.at("missing")), f.at("value")};
      domain(v.type);
      need(v.op == "eq" || v.op == "ne" || v.op == "lt" || v.op == "le" ||
               v.op == "gt" || v.op == "ge",
           "filter comparison unsupported");
      need(v.missing == "include" || v.missing == "exclude" ||
               v.missing == "reject",
           "filter missing policy required");
      (void)at(Json::object(), v.pointer);
      (void)compare(v.value, v.value, v.type);
      predicates.push_back(std::move(v));
    }

    for (const auto &o : p.at("order_by")) {
      keys(o, {"pointer", "type", "direction", "missing"});
      Ordering v{str(o.at("pointer")), str(o.at("type")),
                 str(o.at("direction")), str(o.at("missing"))};
      domain(v.type);
      need(v.direction == "asc" || v.direction == "desc",
           "sort direction invalid");
      need(v.missing == "first" || v.missing == "last" || v.missing == "reject",
           "sort missing policy required");
      (void)at(Json::object(), v.pointer);
      order.push_back(std::move(v));
    }

    std::set<std::string> names;
    for (const auto &c : p.at("columns")) {
      keys(c, {"name", "pointer"});
      auto name = str(c.at("name")), ptr = str(c.at("pointer"));
      need(!name.empty() && name.size() <= 256 && names.insert(name).second,
           "column names must be bounded and unique");
      (void)at(Json::object(), ptr);
      columns.emplace_back(name, ptr);
    }
  }
  bool matches(const Json &row) const {
    bool selected = true;
    for (const auto &f : predicates) {
      const auto *value = at(row, f.pointer);
      bool accepted;
      if (!value) {
        need(f.missing != "reject", "filter field missing");
        accepted = f.missing == "include";
      } else {
        const auto c = compare(*value, f.value, f.type);
        accepted = f.op == "eq"   ? c == 0
                   : f.op == "ne" ? c != 0
                   : f.op == "lt" ? c < 0
                   : f.op == "le" ? c <= 0
                   : f.op == "gt" ? c > 0
                                  : c >= 0;
      }
      selected = selected && accepted;
    }
    return selected;
  }
  std::vector<std::optional<Json>> sort_keys(const Json &row) const {
    std::vector<std::optional<Json>> keys;
    for (const auto &o : order) {
      const auto *value = at(row, o.pointer);
      if (!value) {
        need(o.missing != "reject", "sort field missing");
        keys.push_back(std::nullopt);
      } else {
        (void)compare(*value, *value, o.type);
        keys.push_back(*value);
      }
    }
    return keys;
  }
  bool less(const auto &a, const auto &b) const {
    for (std::size_t i = 0; i < order.size(); ++i) {
      const auto &o = order[i];
      const auto &x = a.keys[i], &y = b.keys[i];
      int c = 0;
      if (!x || !y) {
        if (!x && !y)
          continue;
        c = !x ? (o.missing == "first" ? -1 : 1)
               : (o.missing == "first" ? 1 : -1);
      } else {
        c = compare(*x, *y, o.type);
        if (o.direction == "desc")
          c = -c;
      }
      if (c)
        return c < 0;
    }
    return a.index < b.index;
  }
  Json project(const Json &row, const std::string &pointer,
               std::uint64_t index) const {
    Json fields = Json::object();
    for (const auto &[name, ptr] : columns) {
      const auto *v = at(row, ptr);
      fields[name] = v ? Json{{"status", "available"}, {"value", *v}}
                       : missing("field absent");
    }
    return {{"source_index", dec(index)},
            {"pointer", pointer + "/" + dec(index)},
            {"value", columns.empty() ? row : fields}};
  }
};
} // namespace symphony::sbv::detail::selection
