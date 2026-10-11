#include "consumer_source.hpp"
#include "resource_admission.hpp"
#include "selection_query.hpp"
#include <symphony/knowledge/engine/limits.hpp>
namespace symphony::sbv::detail {
namespace {
namespace store = result_store;
Json page(const Json &p, std::int64_t end) {
  keys(p, {"protocol", "view", "read_options", "limit", "cursor"});
  const auto &reference = p.at("view"), &options = p.at("read_options");
  keys(reference, {"manifest_path", "manifest_sha256", "content_sha256"});
  keys(options, {"max_page_bytes", "cache_bytes"});
  store::ReadOptions read{resource::limit(options.at("max_page_bytes")),
                          u64(options.at("cache_bytes"))};
  store::ResultReader reader({str(reference.at("manifest_path")),
                              str(reference.at("manifest_sha256")),
                              str(reference.at("content_sha256"))},
                             read, [end] { deadline(end); });
  need(reader.select("/origin").read_value() ==
           "immutable typed selection view v1",
       "selected artifact is not a typed selection view");
  const auto summary =
      reader.select("/sections/summary/data").read_value({512000, 16000});
  const auto rows = reader.select("/sections/selection/data");
  const auto total = u64(summary.at("matched_rows")),
             source_rows = u64(summary.at("source_rows"));
  need(rows.describe().at("kind") == "array" &&
           u64(rows.describe().at("children")) == total && total <= source_rows,
       "selection view row count mismatch");
  const auto limit = u64(p.at("limit"));
  need(limit > 0, "positive selection page size required");
  auto identity = p;
  identity.erase("cursor");
  const auto hash = e::sha256_hex(identity.dump());
  const auto cursor = str(p.at("cursor"));
  std::uint64_t offset = 0;
  if (!cursor.empty()) {
    need(cursor.starts_with(hash + ":"),
         "view cursor belongs to another query");
    offset = u64(cursor.substr(hash.size() + 1));
  }
  need(offset <= total, "view cursor outside retained rows");
  Json out = Json::array();
  std::uint64_t bytes = 0, nodes = 0;
  auto children = rows.children(offset, std::min(limit, total - offset));
  while (auto row = children.next()) {
    deadline(end);
    // This is a control-page budget only. The complete retained row remains
    // accessible through bundle query/export when it exceeds this transport.
    auto value = row->read_value({512000, 16000});
    const auto size = value.dump().size();
    auto count = [](const auto &self, const Json &v) -> std::uint64_t {
      std::uint64_t n = 1;
      if (v.is_structured())
        for (const auto &child : v)
          n += self(self, child);
      return n;
    };
    const auto n = count(count, value);
    if (bytes + size > 512000 || nodes + n > 16000)
      break;
    bytes += size;
    nodes += n;
    out.push_back(std::move(value));
  }
  need(!out.empty() || offset == total,
       "view row exceeds typed control page; use bundle query/export");
  const auto next = offset + out.size();
  return {
      {"protocol", "symphony.sbv.result-select.v1"},
      {"content_sha256", reference.at("content_sha256")},
      {"query_sha256", hash},
      {"pointer", summary.at("pointer")},
      {"source_rows", summary.at("source_rows")},
      {"matched_rows", dec(total)},
      {"offset", dec(offset)},
      {"rows", out},
      {"complete", next == total},
      {"next_cursor", next == total ? "" : hash + ":" + dec(next)},
      {"columns", summary.at("columns")},
      {"ordering", "immutable prepared view; stable original index for ties"},
      {"verification_extent", "pinned_manifest_and_selected_page_closure"},
      {"io_stats", reader.io_stats().json()}};
}
} // namespace
Json result_select_view(const Json &p, std::int64_t end) {
  if (p.contains("view"))
    return page(p, end);
  keys(p, {"protocol", "source", "pointer", "filters", "order_by", "columns",
           "ordering_storage", "max_index_rows", "output"});
  need(p.at("ordering_storage") == "memory_index",
       "only explicitly selected memory_index ordering implemented");
  const auto maximum = resource::limit(p.at("max_index_rows"));
  selection::Query query(p);
  const auto pointer = str(p.at("pointer"));
  need(pointer.size() <= 4096,
       "selection pointer exceeds control representation");
  (void)Json::json_pointer(pointer);
  return partitioned_result(
      p, "result-select", end, [&](PartitionedOutput &context) {
        auto admitted = consumer::source(p, end);
        auto table = consumer::pointer(admitted.value, pointer);
        need(table.kind() == "array", "selection requires an array");
        auto spool = context.spool("selected-rows");
        struct Index {
          std::uint64_t index;
          std::vector<std::optional<Json>> keys;
          logical::Value row;
        };
        std::vector<Index> index;
        std::uint64_t ordinal = 0;
        auto children = table.children();
        while (auto child = children.next()) {
          deadline(end);
          const auto row = child->value.materialize();
          if (query.matches(row)) {
            if (query.order.empty())
              spool->append(query.project(row, pointer, ordinal));
            else {
              need(!maximum || index.size() < *maximum,
                   "selected sort-index row budget exceeded");
              index.push_back({ordinal, query.sort_keys(row), child->value});
            }
          }
          ++ordinal;
        }
        std::uint64_t comparisons = 0;
        std::sort(index.begin(), index.end(),
                  [&](const Index &a, const Index &b) {
                    if (++comparisons % 1024 == 0)
                      deadline(end);
                    return query.less(a, b);
                  });
        for (const auto &selected : index) {
          deadline(end);
          spool->append(query.project(selected.row.materialize(), pointer,
                                      selected.index));
        }
        auto retained = spool->close();
        auto definition = p;
        definition.erase("output");
        auto body = base("immutable typed selection view v1");
        auto &sections = body["sections"];
        sections["summary"] = section(
            {{"source_rows", dec(table.size())},
             {"matched_rows", dec(retained.row_count)},
             {"source_content_sha256",
              admitted.value.at("content_sha256").materialize()},
             {"pointer", pointer},
             {"selection_query_sha256", e::sha256_hex(definition.dump())},
             {"columns", p.at("columns")}});
        sections["choices"] = section(definition);
        sections["provenance"] =
            section({{"source_reference", admitted.reference},
                     {"engine_version", version}});
        sections["resources"] =
            section({{"backend", "cpu"},
                     {"actual_workers", "1"},
                     {"source_rows_scanned", dec(ordinal)},
                     {"sort_index_rows", dec(index.size())},
                     {"max_index_rows", p.at("max_index_rows")},
                     {"ordering_storage", "memory_index"},
                     {"sort_key_memory",
                      "O(matched rows times selected sort keys); typed row "
                      "materialization; host allocation applies"},
                     {"scratch_read_options",
                      context.output().value("scratch_read_options",
                                             Json{{"max_page_bytes", nullptr},
                                                  {"cache_bytes", "0"}})}});
        return logical::Value(body).with(
            "sections",
            logical::Value(sections).with(
                "selection", consumer::section(logical::Value(retained.rows))));
      });
}
} // namespace symphony::sbv::detail
