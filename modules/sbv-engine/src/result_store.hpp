#pragma once

#include <cstdint>
#include <functional>
#include <memory>
#include <optional>
#include <stdexcept>
#include <string>
#include <string_view>
#include <symphony/knowledge/engine/json.hpp>

namespace symphony::sbv::result_store {
using Json = knowledge::engine::Json;
using Checkpoint = std::function<void()>;
using Sink = std::function<void(std::string_view)>;

// Private incremental result storage API; the installed process/SDK wire is
// registered separately. This header is not an installed C++ ABI.
inline constexpr auto manifest_protocol = "symphony.sbv.partitioned-result.v1";
inline constexpr auto page_protocol = "symphony.sbv.result-page.v1";

struct Reference {
  std::string manifest_path;
  std::string manifest_sha256;
  std::string content_sha256;
};
struct WriteOptions {
  std::uint64_t page_bytes = 1U << 20;
  std::uint64_t index_fanout = 64;
};
struct ReadOptions {
  // Null selects no additional caller page-memory constraint. The manifest's
  // exact per-page format budget remains checked, not an aggregate limit.
  std::optional<std::uint64_t> max_page_bytes;
  std::uint64_t cache_bytes = 8U << 20;
};
struct IOStats {
  std::uint64_t files_read = 0;
  std::uint64_t bytes_read = 0;
  std::uint64_t cache_hits = 0;
  std::uint64_t maximum_file_bytes = 0;
  Json json() const;
};
struct Receipt {
  Reference reference;
  std::uint64_t manifest_bytes = 0;
  std::uint64_t logical_body_bytes = 0;
  std::uint64_t logical_body_nodes = 0; // includes keys, excludes digest member
  std::uint64_t logical_body_values = 0;
  std::uint64_t page_files_created = 0;
  std::uint64_t page_bytes_created = 0;
  IOStats io_stats;
  Json json() const;
};
class StoreError : public std::runtime_error {
public:
  StoreError(std::string code, std::string message, Json recovery = nullptr);
  std::string code;
  Json recovery;
};

// R3.2 EXTERNAL PROTOTYPE additions. Physical wire is unchanged. These
// source-owned views/cursors extend reader lifetime; they share its checkpoint
// and cache and remain single-caller. Use independent readers or owned rows for
// concurrent workers. Callback views expire on callback return.
struct ValueLimits {
  std::optional<std::uint64_t> max_canonical_bytes;
  std::optional<std::uint64_t> max_counted_nodes; // object keys plus values
};
struct ValueVisitor {
  std::function<void()> begin_object, begin_array, end;
  std::function<void(std::string_view)> key;
  std::function<void(const Json &)> scalar;
};
class ChildCursor;
class NodeHandle {
public:
  NodeHandle(const NodeHandle &) = default;
  NodeHandle(NodeHandle &&) noexcept = default;
  NodeHandle &operator=(const NodeHandle &) = default;
  NodeHandle &operator=(NodeHandle &&) noexcept = default;
  Json describe() const;
  NodeHandle child(std::string_view key) const;
  NodeHandle child(std::uint64_t index) const;
  ChildCursor children(std::uint64_t first = 0,
                       std::optional<std::uint64_t> count = {}) const;
  Json read_value(ValueLimits = {}) const;
  Json visit(const ValueVisitor &, ValueLimits = {}) const;
  Json export_json(const Sink &, ValueLimits = {}) const;
  Json hash(ValueLimits = {}) const;
private:
  struct Impl;
  std::shared_ptr<Impl> p_;
  explicit NodeHandle(std::shared_ptr<Impl>);
  friend class ResultReader;
  friend class ResultWriter;
  friend class ChildCursor;
};
class ChildCursor {
public:
  ~ChildCursor();
  ChildCursor(ChildCursor &&) noexcept;
  ChildCursor &operator=(ChildCursor &&) noexcept;
  ChildCursor(const ChildCursor &) = delete;
  ChildCursor &operator=(const ChildCursor &) = delete;
  std::optional<NodeHandle> next();
  Json progress() const;
private:
  struct Impl;
  std::unique_ptr<Impl> p_;
  explicit ChildCursor(std::unique_ptr<Impl>);
  friend class NodeHandle;
};

// One writer owns one new bundle directory. It never overwrites/resumes an
// existing directory. A failed/abandoned bundle has no published manifest.
// All input object keys must be in native Json::dump() order, strictly unique.
// value() accepts one bounded subtree; begin/end allow arbitrary aggregate
// containers to be streamed. Root is the result-v1 body without its digest.
class ResultWriter {
public:
  ResultWriter(std::string bundle_path, WriteOptions = {}, Checkpoint = {});
  ~ResultWriter();
  ResultWriter(const ResultWriter &) = delete;
  ResultWriter &operator=(const ResultWriter &) = delete;
  void begin_object();
  void begin_array();
  void key(std::string);
  void value(const Json &);
  void end();
  Receipt finish(std::optional<std::string> expected_content_sha256 = {});
  Json recovery() const;
  // Verified streaming re-encoding: destination owns all resulting pages.
  // Root NodeHandle copies the full five-member logical result, not its body.
  Json append_subtree(const NodeHandle &, ValueLimits = {});

private:
  struct Impl;
  std::unique_ptr<Impl> p_;
};

class ResultReader {
public:
  ResultReader(Reference, ReadOptions = {}, Checkpoint = {});
  ~ResultReader();
  ResultReader(const ResultReader &) = delete;
  ResultReader &operator=(const ResultReader &) = delete;
  // Root-only metadata; inventories are pointers/counts, never flat lists.
  Json inspect() const;
  NodeHandle select(std::string_view pointer);
  NodeHandle select_node(std::uint64_t node_id);
  // Immediate children, or the selected scalar itself. Containers are reported
  // as empty object/array plus child count. Offset is a logical child ordinal;
  // the external control layer binds any continuation to the full reference,
  // pointer and query/schema/access semantics. All returned Json is owned.
  Json query(std::string_view pointer, std::uint64_t offset,
             std::uint64_t row_limit, std::uint64_t byte_limit);
  // Stable preorder VALUE ordinal (object keys do not take ordinals), bound to
  // both reference digests. Root=0; virtual content_sha256=1. This reaches
  // every admitted node without transporting an expanded/deep JSON Pointer.
  Json query_node(std::uint64_t node_id, std::uint64_t offset,
                  std::uint64_t row_limit, std::uint64_t byte_limit);
  // Stream exact compact result-v1 JSON, including virtual content_sha256.
  // On any failure throws; final closing brace is emitted only after full body
  // digest/count validation. Sink views expire when that call returns.
  Json export_json(const Sink &);
  Json export_ndjson(const Sink &);
  // Full logical traversal and hash/count verification without materialization.
  Json verify_closure();
  IOStats io_stats() const;
  const Reference &reference() const;

private:
  struct Impl;
  std::unique_ptr<Impl> p_;
};

// Streaming selected canonical-key-order result-v1 JSON import. Input file
// bytes are pinned independently; root content_sha256 is required and checked
// before manifest publication. No input aggregate size/default deadline cap.
Json import_json(const std::string &input_path,
                 const std::string &expected_file_sha256,
                 const std::string &bundle_path, WriteOptions = {},
                 Checkpoint = {});
// Native bounded-memory export; destination must not exist. Receipt identifies
// exact exported bytes and logical source separately, and includes the exact
// withheld completion_suffix a caller may release only after checking bytes.
Json export_file(ResultReader &, const std::string &output_path,
                 std::string_view format, Checkpoint = {});
} // namespace symphony::sbv::result_store
