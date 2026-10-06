#include "wide_rational.hpp"
#include <algorithm>
namespace symphony::sbv::detail {
namespace {
namespace w = wide_rational;
int compare_ratio(w::R a, w::R b) {
  a = w::reduce(a);
  b = w::reduce(b);
  if (a.n < 0 && b.n >= 0)
    return -1;
  if (a.n >= 0 && b.n < 0)
    return 1;
  int sign = a.n < 0 ? -1 : 1;
  a.n = w::absolute(a.n);
  b.n = w::absolute(b.n);
  for (;;) {
    auto aq = a.n / a.d, bq = b.n / b.d;
    if (aq != bq)
      return sign * (aq < bq ? -1 : 1);
    auto ar = a.n % a.d, br = b.n % b.d;
    if (!ar || !br)
      return ar == br ? 0 : sign * (!ar ? -1 : 1);
    a = {a.d, ar};
    b = {b.d, br};
    sign = -sign;
  }
}
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
  return compare_ratio(parse(a), parse(b));
}
struct Predicate {
  std::string pointer, type, op, missing;
  Json value;
};
struct Ordering {
  std::string pointer, type, direction, missing;
};
} // namespace
Json result_select(const Json &p, std::int64_t end) {
  keys(p, {"protocol", "path", "expected_sha256", "pointer", "filters",
           "order_by", "columns", "limit", "cursor"});
  auto source = e::parse_bounded_json(read_file(str(p.at("path")), end),
                                      artifact_bytes, artifact_values);
  validate_result(source);
  need(!str(p.at("expected_sha256")).empty() &&
           source.at("content_sha256") == p.at("expected_sha256"),
       "selection source identity mismatch");
  const auto pointer = str(p.at("pointer"));
  const auto *table = at(source, pointer);
  need(table && table->is_array() && table->size() <= 65536,
       "selection needs a bounded array");
  const auto limit = u64(p.at("limit"));
  need(limit >= 1 && limit <= 256, "selection page size bound");
  need(p.at("filters").is_array() && p.at("filters").size() <= 16 &&
           p.at("order_by").is_array() && p.at("order_by").size() <= 8 &&
           p.at("columns").is_array() && p.at("columns").size() <= 64,
       "selection field bounds");
  std::vector<Predicate> predicates;
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
  std::vector<Ordering> order;
  for (const auto &o : p.at("order_by")) {
    keys(o, {"pointer", "type", "direction", "missing"});
    Ordering v{str(o.at("pointer")), str(o.at("type")), str(o.at("direction")),
               str(o.at("missing"))};
    domain(v.type);
    need(v.direction == "asc" || v.direction == "desc",
         "sort direction invalid");
    need(v.missing == "first" || v.missing == "last" || v.missing == "reject",
         "sort missing policy required");
    (void)at(Json::object(), v.pointer);
    order.push_back(std::move(v));
  }
  std::vector<std::pair<std::string, std::string>> columns;
  std::set<std::string> names;
  for (const auto &c : p.at("columns")) {
    keys(c, {"name", "pointer"});
    auto name = str(c.at("name")), ptr = str(c.at("pointer"));
    need(!name.empty() && name.size() <= 256 && names.insert(name).second,
         "column names must be bounded and unique");
    (void)at(Json::object(), ptr);
    columns.emplace_back(name, ptr);
  }
  std::vector<std::size_t> indices;
  for (std::size_t i = 0; i < table->size(); ++i) {
    deadline(end);
    const auto &row = (*table)[i];
    bool selected = true;
    // All predicate values are validated even if an earlier predicate excludes
    // a row.
    for (const auto &f : predicates) {
      const auto *value = at(row, f.pointer);
      bool accepted;
      if (!value) {
        need(f.missing != "reject", "filter field missing");
        accepted = f.missing == "include";
      } else {
        auto c = compare(*value, f.value, f.type);
        accepted = f.op == "eq"   ? c == 0
                   : f.op == "ne" ? c != 0
                   : f.op == "lt" ? c < 0
                   : f.op == "le" ? c <= 0
                   : f.op == "gt" ? c > 0
                                  : c >= 0;
      }
      selected = selected && accepted;
    }
    if (selected) {
      for (const auto &o : order) {
        const auto *v = at(row, o.pointer);
        if (!v)
          need(o.missing != "reject", "sort field missing");
        else
          (void)compare(*v, *v, o.type);
      }
      indices.push_back(i);
    }
  }
  std::uint64_t comparisons = 0;
  std::stable_sort(indices.begin(), indices.end(), [&](auto i, auto j) {
    if (++comparisons % 1024 == 0)
      deadline(end);
    for (const auto &o : order) {
      const auto *a = at((*table)[i], o.pointer),
                 *b = at((*table)[j], o.pointer);
      int c = 0;
      if (!a || !b) {
        if (!a && !b)
          continue;
        c = !a ? (o.missing == "first" ? -1 : 1)
               : (o.missing == "first" ? 1 : -1);
      } else {
        c = compare(*a, *b, o.type);
        if (o.direction == "desc")
          c = -c;
      }
      if (c)
        return c < 0;
    }
    return i < j;
  });
  auto identity = p;
  identity.erase("cursor");
  const auto query = e::sha256_hex(identity.dump());
  std::uint64_t offset = 0;
  const auto cursor = str(p.at("cursor"));
  if (!cursor.empty()) {
    need(cursor.starts_with(query + ":"),
         "selection cursor belongs to another query");
    offset = u64(cursor.substr(query.size() + 1));
  }
  need(offset <= indices.size(), "selection cursor out of range");
  Json rows = Json::array();
  std::size_t bytes = 0, values = 0;
  for (std::size_t pos = offset; pos < indices.size() && rows.size() < limit;
       ++pos) {
    deadline(end);
    auto idx = indices[pos];
    const auto &row = (*table)[idx];
    Json fields = Json::object();
    for (const auto &[name, ptr] : columns) {
      const auto *v = at(row, ptr);
      fields[name] = v ? Json{{"status", "available"}, {"value", *v}}
                       : missing("field absent");
    }
    Json out{{"source_index", dec(idx)},
             {"pointer", pointer + "/" + dec(idx)},
             {"value", columns.empty() ? row : fields}};
    auto count = [](const auto &self, const Json &v) -> std::size_t {
      std::size_t n = 1;
      if (v.is_structured())
        for (const auto &c : v)
          n += self(self, c);
      return n;
    };
    const auto size = out.dump().size(), n = count(count, out);
    if (bytes + size > 512000 || values + n > 16000) {
      need(!rows.empty(),
           "selected row exceeds page bounds; select narrower columns");
      break;
    }
    bytes += size;
    values += n;
    rows.push_back(std::move(out));
  }
  const auto next = offset + rows.size();
  const bool complete = next == indices.size();
  return {{"protocol", "symphony.sbv.result-select.v1"},
          {"content_sha256", source.at("content_sha256")},
          {"query_sha256", query},
          {"pointer", pointer},
          {"source_rows", dec(table->size())},
          {"matched_rows", dec(indices.size())},
          {"offset", dec(offset)},
          {"rows", rows},
          {"complete", complete},
          {"next_cursor", complete ? "" : query + ":" + dec(next)},
          {"ordering", "selected keys; stable original index for ties; missing "
                       "placement independent of direction"},
          {"columns", p.at("columns")}};
}
} // namespace symphony::sbv::detail
