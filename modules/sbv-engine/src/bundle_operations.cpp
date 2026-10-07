#include "detail.hpp"
#include "result_store.hpp"
#include <algorithm>
#include <limits>
#include <symphony/knowledge/engine/limits.hpp>
#include <symphony/knowledge/engine/path.hpp>

namespace symphony::sbv::detail {
namespace {
namespace store = result_store;
constexpr auto order = "canonical_immediate_children.v1";
constexpr auto access = "all_admitted_fields.v1";
constexpr auto cursor_protocol = "symphony.sbv.bundle-cursor.v1";
// The fixed process-v2 envelope includes caller identifiers (at most 128 bytes
// each) and its digest. This reserve concerns control framing, never result
// size.
constexpr std::size_t envelope_bytes = 2048, envelope_values = 64;
std::size_t values(const Json &v, std::size_t depth = 0) {
  need(depth <= e::Limits::max_json_depth,
       "bundle control nesting exceeds transport");
  need(!v.is_number(), "bundle control numbers must be decimal strings");
  if (v.is_string())
    need(v.get_ref<const std::string &>().size() <= e::Limits::max_string_bytes,
         "bundle control string exceeds transport");
  std::size_t n = 1;
  if (v.is_object()) {
    n += v.size();
    for (const auto &[key, child] : v.items()) {
      need(key.size() <= e::Limits::max_string_bytes,
           "bundle control key exceeds transport");
      n += values(child, depth + 1);
      need(n <= e::Limits::max_json_values,
           "bundle control values exceed transport");
    }
  } else if (v.is_array()) {
    for (const auto &child : v) {
      n += values(child, depth + 1);
      need(n <= e::Limits::max_json_values,
           "bundle control values exceed transport");
    }
  }
  return n;
}
std::string digest(const Json &v) {
  const auto s = str(v);
  need(s.size() == 64 &&
           s.find_first_not_of("0123456789abcdef") == std::string::npos,
       "bundle SHA256 must be lowercase hexadecimal");
  return s;
}
std::string local_path(const Json &v) {
  const auto p = str(v);
  need(p.size() > 1 && p.size() <= 4096 && p[0] == '/' &&
           e::is_safe_relative_path(p.substr(1)),
       "absolute clean bundle path required");
  return p;
}
store::Reference reference(const Json &v) {
  keys(v, {"manifest_path", "manifest_sha256", "content_sha256"});
  return {local_path(v.at("manifest_path")), digest(v.at("manifest_sha256")),
          digest(v.at("content_sha256"))};
}
store::ReadOptions read_options(const Json &v) {
  keys(v, {"max_page_bytes", "cache_bytes"});
  store::ReadOptions options;
  if (!v.at("max_page_bytes").is_null()) {
    options.max_page_bytes = u64(v.at("max_page_bytes"));
    need(*options.max_page_bytes > 0,
         "selected page memory budget must be positive");
  }
  options.cache_bytes = u64(v.at("cache_bytes"));
  return options;
}
store::WriteOptions write_options(const Json &v) {
  keys(v, {"page_bytes", "index_fanout"});
  store::WriteOptions options{u64(v.at("page_bytes")),
                              u64(v.at("index_fanout"))};
  need(options.page_bytes >= 512 && options.index_fanout >= 2,
       "bundle layout requires page_bytes >= 512 and index_fanout >= 2");
  return options;
}
Json selected_fields(const Json &source,
                     std::initializer_list<const char *> names) {
  Json out = Json::object();
  for (auto name : names)
    out[name] = source.at(name);
  return out;
}
void counts(Json &out, const Json &source) {
  for (auto name : {"logical_body_bytes", "logical_body_nodes",
                    "logical_body_values", "exported_value_nodes"})
    out[name] = source.at(name);
}
Json finish(Json out, const std::string &op, const Json &p) {
  auto slug = op;
  std::replace(slug.begin(), slug.end(), '_', '-');
  out["protocol"] = "symphony.sbv." + slug + ".v1";
  out["status"] = "complete";
  out["source_authorship"] = "not_verified";
  out["extensions"] = p.at("extensions");
  return out;
}
Json cursor(const Json &p, const Json &page, std::uint64_t next) {
  return {{"protocol", cursor_protocol},
          {"reference", p.at("reference")},
          {"selector", p.at("selector")},
          {"selected_node_id", page.at("selected_node_id")},
          {"order", order},
          {"access_profile", access},
          {"next_offset", dec(next)}};
}
std::uint64_t cursor_offset(const Json &p) {
  const auto &v = p.at("cursor");
  if (v.is_null())
    return 0;
  keys(v, {"protocol", "reference", "selector", "selected_node_id", "order",
           "access_profile", "next_offset"});
  need(v.at("protocol") == cursor_protocol &&
           v.at("reference") == p.at("reference") &&
           v.at("selector") == p.at("selector") && v.at("order") == order &&
           v.at("access_profile") == access,
       "bundle cursor does not match source and query");
  (void)u64(v.at("selected_node_id"));
  const auto offset = u64(v.at("next_offset"));
  need(offset > 0, "bundle cursor must advance");
  return offset;
}
Json query(store::ResultReader &reader, const Json &p) {
  const auto &selector = p.at("selector");
  need(selector.is_object() && selector.contains("kind"),
       "bundle selector required");
  const auto kind = str(selector.at("kind"));
  if (kind == "node_id")
    keys(selector, {"kind", "node_id"});
  else {
    need(kind == "pointer", "unsupported bundle selector kind");
    keys(selector, {"kind", "pointer"});
    (void)str(selector.at("pointer"));
  }
  const auto rows = u64(p.at("row_limit"));
  const auto budget = u64(p.at("byte_limit"));
  need(rows > 0 && budget > envelope_bytes,
       "bundle query requires positive rows and process-envelope headroom");
  const auto offset = cursor_offset(p);
  // A page never needs more rows than the fixed transport can represent.
  const auto admitted_rows =
      std::min<std::uint64_t>(rows, e::Limits::max_json_values / 13);
  const auto admitted_bytes =
      std::min<std::uint64_t>(budget, e::Limits::max_response_bytes) -
      envelope_bytes;
  auto page = kind == "node_id"
                  ? reader.query_node(u64(selector.at("node_id")), offset,
                                      admitted_rows, admitted_bytes)
                  : reader.query(str(selector.at("pointer")), offset,
                                 admitted_rows, admitted_bytes);
  if (!p.at("cursor").is_null())
    need(p.at("cursor").at("selected_node_id") == page.at("selected_node_id"),
         "bundle cursor selected node changed");
  auto out = selected_fields(page, {"reference", "verification_extent",
                                    "selected_node_id", "selected_kind",
                                    "offset", "total", "nodes", "io_stats"});
  out["selector"] = selector;
  out["order"] = order;
  out["access_profile"] = access;
  out = finish(std::move(out), "bundle_query", p);
  const auto total = u64(out.at("total"));
  auto &nodes = out.at("nodes");
  need(offset <= total && nodes.is_array() && nodes.size() <= total - offset,
       "bundle query range inconsistent");
  // Trim the already-read bounded page, without rescanning any source rows.
  // Count/byte budgets include the full cursor, echoed choices and envelope.
  for (;;) {
    const auto next = offset + nodes.size();
    out["complete"] = next == total;
    out["next_cursor"] = next == total ? Json(nullptr) : cursor(p, out, next);
    bool fits = false;
    try {
      fits = values(out) + envelope_values <= e::Limits::max_json_values &&
             out.dump().size() + envelope_bytes <
                 std::min<std::uint64_t>(budget, e::Limits::max_response_bytes);
    } catch (const e::Error &) {
      fits = false;
    }
    if (fits) {
      need(!nodes.empty() || next == total,
           "bundle query cannot make progress within selected control budget");
      return out;
    }
    need(!nodes.empty(),
         "bundle query metadata exceeds selected control budget");
    nodes.erase(nodes.size() - 1);
  }
}
} // namespace

Json bundle_control(const std::string &op, const Json &p, std::int64_t end) {
  deadline(end);
  need(p.contains("extensions") && p.at("extensions").is_object(),
       "bundle extensions object required");
  // Reserve bounded metadata before creating anything. Without this preflight a
  // nearly-full extension echo could make an otherwise committed result
  // impossible to return over process-v2. This is only a control-frame budget.
  need(values(p.at("extensions")) + 512 <= e::Limits::max_json_values,
       "bundle extensions leave insufficient response metadata capacity");
  const auto checkpoint = [end] { deadline(end); };
  try {
    if (op == "bundle_import") {
      keys(p, {"protocol", "input_path", "expected_file_sha256", "bundle_path",
               "write_options", "extensions"});
      const auto destination = local_path(p.at("bundle_path"));
      (void)local_path(destination + "/manifest.json");
      const auto result = store::import_json(
          local_path(p.at("input_path")), digest(p.at("expected_file_sha256")),
          destination, write_options(p.at("write_options")), checkpoint);
      auto out = selected_fields(
          result, {"reference", "verification_extent", "input_path",
                   "input_file_sha256", "input_bytes", "bundle_path",
                   "manifest_bytes", "page_files_created", "page_bytes_created",
                   "write_options", "io_stats"});
      counts(out, result);
      return finish(std::move(out), op, p);
    }
    if (op == "bundle_query")
      keys(p, {"protocol", "reference", "read_options", "selector", "cursor",
               "row_limit", "byte_limit", "extensions"});
    else if (op == "bundle_export")
      keys(p, {"protocol", "reference", "read_options", "output_path", "format",
               "extensions"});
    else {
      need(op == "bundle_inspect" || op == "bundle_verify",
           "unsupported bundle operation");
      keys(p, {"protocol", "reference", "read_options", "extensions"});
    }
    store::ResultReader reader(reference(p.at("reference")),
                               read_options(p.at("read_options")), checkpoint);
    if (op == "bundle_query")
      return query(reader, p);
    Json result;
    Json out;
    if (op == "bundle_inspect") {
      result = reader.inspect();
      out = selected_fields(
          result, {"reference", "verification_extent", "logical_protocol",
                   "storage_protocol", "canonicalization", "root_node_id",
                   "root_children", "write_options", "io_stats"});
    } else if (op == "bundle_verify") {
      result = reader.verify_closure();
      out = selected_fields(result,
                            {"reference", "verification_extent", "io_stats"});
    } else {
      const auto format = str(p.at("format"));
      need(format == "json" || format == "ndjson",
           "unsupported bundle export format");
      result = store::export_file(reader, local_path(p.at("output_path")),
                                  format, checkpoint);
      out = selected_fields(
          result, {"reference", "verification_extent", "output_path", "format",
                   "bytes", "file_sha256", "completion_suffix", "io_stats"});
    }
    counts(out, result);
    return finish(std::move(out), op, p);
  } catch (const store::StoreError &error) {
    if (error.recovery.is_null())
      throw e::Error(error.code,
                     "SBV bundle operation refused; inspect the selected "
                     "source or destination",
                     2);
    need(op == "bundle_import" || op == "bundle_export",
         "unexpected read-only bundle recovery");
    auto slug = op;
    std::replace(slug.begin(), slug.end(), '_', '-');
    return {{"protocol", "symphony.sbv." + slug + ".v1"},
            {"status", "recovery_required"},
            {"code", op == "bundle_import" ? "sbv.bundle_import_incomplete"
                                           : "sbv.bundle_export_incomplete"},
            {"request_sha256", e::sha256_hex(p.dump())},
            {"automatic_retry", false},
            {"rollback_performed", false},
            {"recovery", error.recovery}};
  }
}
} // namespace symphony::sbv::detail
