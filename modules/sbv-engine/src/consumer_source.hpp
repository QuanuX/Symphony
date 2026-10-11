#pragma once
#include "partitioned_output.hpp"
#include "stream_census.hpp"
#include <charconv>
namespace symphony::sbv::detail::consumer {
inline logical::Value pointer(logical::Value value, const std::string &path) {
  (void)Json::json_pointer(path);
  std::size_t at = 0;
  while (at < path.size()) {
    const auto next = path.find('/', at + 1);
    const auto edge =
        path.substr(at + 1, next == std::string::npos ? next : next - at - 1);
    std::string decoded;
    for (std::size_t i = 0; i < edge.size(); ++i) {
      if (edge[i] == '~')
        decoded += edge[++i] == '0' ? '~' : '/';
      else
        decoded += edge[i];
    }
    if (value.kind() == "array") {
      const auto index = u64(decoded);
      need(index < value.size(), "consumer array selection outside source");
      value = value.at(index);
    } else {
      need(value.kind() == "object" && value.contains(decoded),
           "consumer source selection absent");
      value = value.at(decoded);
    }
    if (next == std::string::npos)
      break;
    at = next;
  }
  return value;
}
inline LogicalSelection source(const Json &p, std::int64_t end) {
  if (p.contains("source")) {
    need(!p.contains("path") && !p.contains("expected_sha256"),
         "consumer selects exactly one source form");
    return select_logical_result(p.at("source"), end);
  }
  return select_logical_result({{"path", p.at("path")},
                                {"expected_sha256", p.at("expected_sha256")},
                                {"pointer", ""}},
                               end);
}
inline logical::Value section(logical::Value data,
                              const std::string &status = "available",
                              const std::string &reason = "") {
  return logical::Value::object({{"data", std::move(data)},
                                 {"status", logical::Value(status)},
                                 {"reason", logical::Value(reason)}});
}
} // namespace symphony::sbv::detail::consumer

namespace symphony::sbv::detail::consumer {
// A single-caller append-only result array. Partitioned producers retain rows
// in private immutable pages; the explicitly selected legacy profile uses DOM.
class Rows {
  std::unique_ptr<result_store::RowSpool> spool_;
  Json legacy_ = Json::array();

public:
  Rows(PartitionedOutput *context, const std::string &name) {
    if (context)
      spool_ = context->spool(name);
  }
  void append(const Json &row) {
    if (spool_)
      spool_->append(row);
    else
      legacy_.push_back(row);
  }
  void append(const logical::Value &row) {
    if (spool_)
      spool_->append_logical(row);
    else
      legacy_.push_back(row.materialize());
  }
  std::uint64_t size() const {
    return spool_ ? spool_->size() : legacy_.size();
  }
  bool empty() const { return size() == 0; }
  logical::Value close() {
    return spool_ ? logical::Value(spool_->close().rows)
                  : logical::Value(std::move(legacy_));
  }
};
inline logical::Value source_context(const logical::Value &parent) {
  auto context =
      logical::Value::object({{"choices", parent.at("choices")},
                              {"provenance", parent.at("provenance")}});
  const auto resources = parent.at("resources").at("data");
  if (resources.kind() == "object" && resources.contains("dataset_feed"))
    context = context.with("dataset_feed", resources.at("dataset_feed"));
  else if (parent.contains("source_context")) {
    const auto original = parent.at("source_context").at("data");
    if (original.kind() == "object" && original.contains("dataset_feed"))
      context = context.with("dataset_feed", original.at("dataset_feed"));
  }
  return context;
}
} // namespace symphony::sbv::detail::consumer

namespace symphony::sbv::detail::consumer {
inline LogicalSelection result_reference(Json reference, std::int64_t end) {
  if (!reference.contains("kind")) {
    keys_optional(reference, {"path", "expected_sha256"}, {"pointer"});
    if (!reference.contains("pointer"))
      reference["pointer"] = "";
  }
  return select_logical_result(reference, end);
}
inline void fields(const logical::Value &value,
                   std::initializer_list<const char *> names) {
  need(value.kind() == "object" && value.size() == names.size(),
       "unexpected logical object fields");
  for (auto name : names)
    need(value.contains(name), "required logical field absent");
}
inline std::optional<logical::Value> select_signals(
    const Json &selection, const logical::Value &signals,
    const std::map<std::string, logical::Value> &index,
    const std::function<void(const std::string &, const logical::Value &)>
        &apply,
    std::int64_t end) {
  std::set<std::string> seen;
  auto selected = [&](const std::string &id) {
    deadline(end);
    need(index.contains(id) && seen.insert(id).second,
         "unknown or duplicate selected signal");
    apply(id, index.at(id));
  };
  if (selection.is_array()) {
    for (const auto &id : selection)
      selected(str(id));
    return {};
  }
  need(selection.is_object() && selection.contains("kind"),
       "signal selection required");
  if (selection.at("kind") == "all" || selection.at("kind") == "range") {
    std::uint64_t first = 0, count = signals.size();
    if (selection.at("kind") == "all")
      keys(selection, {"kind"});
    else {
      keys(selection, {"kind", "first", "count"});
      first = u64(selection.at("first"));
      need(first <= signals.size(), "selected range outside census");
      count = selection.at("count").is_null() ? signals.size() - first
                                              : u64(selection.at("count"));
      need(count <= signals.size() - first,
           "selected range extends beyond census");
    }
    auto cursor = signals.children(first, count);
    while (auto row = cursor.next())
      selected(str(row->value.at("signal_id").materialize()));
    return {};
  }
  std::optional<logical::Value> ids, evidence;
  if (selection.at("kind") == "ids") {
    keys(selection, {"kind", "ids", "order"});
    ids.emplace(selection.at("ids"));
  } else {
    keys(selection, {"kind", "source", "order"});
    need(selection.at("kind") == "referenced_ids",
         "unknown signal selection kind");
    auto admitted = select_bundle_value(selection.at("source"), end);
    ids = admitted.value;
    evidence = logical::Value::object(
        {{"role", logical::Value(Json("signal_ids"))},
         {"reference", logical::Value(admitted.reference)},
         {"value", admitted.value}});
  }
  need(ids->kind() == "array", "selected IDs array required");
  const auto order = str(selection.at("order"));
  need(order == "supplied" || order == "census", "selected ID order required");
  std::set<std::string> wanted;
  auto cursor = ids->children();
  while (auto child = cursor.next()) {
    deadline(end);
    auto id = str(child->value.materialize());
    need(index.contains(id) && wanted.insert(id).second,
         "unknown or duplicate selected ID");
    if (order == "supplied")
      selected(id);
  }
  if (order == "census") {
    auto rows = signals.children();
    while (auto row = rows.next()) {
      auto id = str(row->value.at("signal_id").materialize());
      if (wanted.contains(id))
        selected(id);
    }
  }
  return evidence;
}
} // namespace symphony::sbv::detail::consumer

namespace symphony::sbv::detail::consumer {
struct InputRows {
  logical::Value value;
  std::optional<logical::Value> evidence;
};
inline InputRows input_rows(const Json &selection, const std::string &role,
                            std::int64_t end) {
  if (selection.is_array())
    return {logical::Value(selection), {}};
  keys(selection, {"kind", "source"});
  need(selection.at("kind") == "referenced_rows",
       "inline or explicitly referenced rows required");
  auto admitted = select_bundle_value(selection.at("source"), end);
  need(admitted.value.kind() == "array",
       "selected rows source must be an array");
  return {
      admitted.value,
      logical::Value::object({{"role", logical::Value(role)},
                              {"reference", logical::Value(admitted.reference)},
                              {"value", admitted.value}})};
}
} // namespace symphony::sbv::detail::consumer
