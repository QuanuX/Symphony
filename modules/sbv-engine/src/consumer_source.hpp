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
