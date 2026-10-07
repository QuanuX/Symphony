#include "result_store.hpp"
#include "streaming_sha256.hpp"
#include <algorithm>
#include <array>
#include <bit>
#include <cerrno>
#include <charconv>
#include <fcntl.h>
#include <filesystem>
#include <limits>
#include <list>
#include <map>
#include <set>
#include <symphony/knowledge/engine/path.hpp>
#include <sys/stat.h>
#include <unistd.h>
#include <utility>
#include <vector>

namespace symphony::sbv::result_store {
namespace {
constexpr auto logical_protocol = "symphony.sbv.result.v1";
constexpr auto canonicalization =
    "nlohmann_compact_utf8_sorted_keys_no_numeric_literals.v1";
constexpr std::uint64_t manifest_budget = 1U << 20;
std::string dec(std::uint64_t n) { return std::to_string(n); }
[[noreturn]] void fail(const std::string &code, const std::string &message) {
  throw StoreError(code, message);
}
void need(bool ok, const std::string &message) {
  if (!ok)
    fail("bundle.contract", message);
}
void check(const Checkpoint &c) {
  if (c)
    c();
}
std::uint64_t add(std::uint64_t a, std::uint64_t b) {
  need(b <= UINT64_MAX - a, "uint64 count/length representation exceeded");
  return a + b;
}
std::size_t host(std::uint64_t n) {
  need(n <= std::numeric_limits<std::size_t>::max(),
       "host size representation exceeded");
  return static_cast<std::size_t>(n);
}
std::string str(const Json &v) {
  need(v.is_string(), "string required");
  return v.get<std::string>();
}
std::uint64_t num(const Json &v) {
  const auto s = str(v);
  std::uint64_t n = 0;
  const auto [p, ec] = std::from_chars(s.data(), s.data() + s.size(), n);
  need(ec == std::errc{} && p == s.data() + s.size() && dec(n) == s,
       "canonical uint64 string required");
  return n;
}
void shape(const Json &v, std::initializer_list<const char *> fields) {
  need(v.is_object() && v.size() == fields.size(),
       "exact object shape required");
  for (const auto *f : fields)
    need(v.contains(f), "required field absent");
}
void digest(const std::string &s) {
  need(s.size() == 64 && std::all_of(s.begin(), s.end(),
                                     [](char c) {
                                       return (c >= '0' && c <= '9') ||
                                              (c >= 'a' && c <= 'f');
                                     }),
       "lowercase SHA-256 required");
}
class Hash {
  hash_detail::Sha256 state_;
  bool finished_ = false;

public:
  void update(std::string_view s) {
    need(!finished_ &&
             state_.update(reinterpret_cast<const std::uint8_t *>(s.data()),
                           s.size()),
         "SHA-256 byte-length representation exceeded");
  }
  std::string finish() {
    need(!finished_, "digest already finalized");
    finished_ = true;
    std::uint8_t bytes[32];
    state_.finish(bytes);
    constexpr char hex[] = "0123456789abcdef";
    std::string out;
    out.reserve(64);
    for (auto b : bytes) {
      out += hex[b >> 4];
      out += hex[b & 15];
    }
    return out;
  }
};
std::string hash(std::string_view s) {
  Hash h;
  h.update(s);
  return h.finish();
}
struct FD {
  int n = -1;
  explicit FD(int value = -1) : n(value) {}
  ~FD() {
    if (n >= 0)
      ::close(n);
  }
  FD(const FD &) = delete;
  FD &operator=(const FD &) = delete;
  FD(FD &&o) noexcept : n(std::exchange(o.n, -1)) {}
  FD &operator=(FD &&o) noexcept {
    if (n >= 0)
      ::close(n);
    n = std::exchange(o.n, -1);
    return *this;
  }
};
std::vector<std::string> path_parts(const std::string &path) {
  need(path.size() > 1 && path.size() <= 4096 && path[0] == '/' &&
           path.back() != '/' &&
           knowledge::engine::is_safe_relative_path(path.substr(1)),
       "absolute safe no-follow path within4096 bytes required");
  std::vector<std::string> out;
  std::size_t p = 1;
  while (p < path.size()) {
    auto q = path.find('/', p);
    if (q == std::string::npos)
      q = path.size();
    auto s = path.substr(p, q - p);
    need(!s.empty() && s != "." && s != "..", "safe no-follow path required");
    out.push_back(std::move(s));
    p = q + 1;
  }
  return out;
}
std::pair<FD, std::string> parent(const std::string &path) {
  auto parts = path_parts(path);
  FD dir(::open("/", O_RDONLY | O_DIRECTORY | O_CLOEXEC | O_NOFOLLOW));
  need(dir.n >= 0, "root directory unavailable");
  for (std::size_t i = 0; i + 1 < parts.size(); ++i) {
    FD next(::openat(dir.n, parts[i].c_str(),
                     O_RDONLY | O_DIRECTORY | O_CLOEXEC | O_NOFOLLOW));
    need(next.n >= 0, "parent directory must exist without symlinks");
    dir = std::move(next);
  }
  return {std::move(dir), parts.back()};
}
FD open_directory(const std::string &path) {
  auto [dir, name] = parent(path);
  FD result(::openat(dir.n, name.c_str(),
                     O_RDONLY | O_DIRECTORY | O_CLOEXEC | O_NOFOLLOW));
  need(result.n >= 0, "bundle directory unavailable");
  return result;
}
FD open_file(int dir, const std::string &name) {
  FD file(::openat(dir, name.c_str(),
                   O_RDONLY | O_NONBLOCK | O_CLOEXEC | O_NOFOLLOW));
  if (file.n < 0)
    fail("bundle.missing", "required regular file unavailable: " + name);
  struct stat st{};
  need(::fstat(file.n, &st) == 0 && S_ISREG(st.st_mode) && st.st_size >= 0,
       "regular no-follow file required");
  return file;
}
std::string read_file(int dir, const std::string &name, std::uint64_t maximum,
                      const Checkpoint &checkpoint, IOStats *stats = nullptr) {
  check(checkpoint);
  auto file = open_file(dir, name);
  struct stat st{};
  need(::fstat(file.n, &st) == 0 && st.st_size >= 0, "file stat failed");
  const auto size = static_cast<std::uint64_t>(st.st_size);
  need(size <= maximum, "selected per-file budget exceeded");
  std::string out;
  out.reserve(host(size));
  std::array<char, 65536> buffer{};
  for (;;) {
    check(checkpoint);
    auto n = ::read(file.n, buffer.data(), buffer.size());
    if (n < 0 && errno == EINTR)
      continue;
    if (n < 0)
      fail("bundle.read_failed", "file read failed");
    if (n == 0)
      break;
    need(static_cast<std::uint64_t>(n) <= maximum - out.size(),
         "file grew beyond selected budget");
    out.append(buffer.data(), static_cast<std::size_t>(n));
  }
  need(out.size() == size, "file size changed while reading");
  if (stats) {
    stats->files_read = add(stats->files_read, 1);
    stats->bytes_read = add(stats->bytes_read, size);
    stats->maximum_file_bytes = std::max(stats->maximum_file_bytes, size);
  }
  return out;
}
void write_all(int fd, std::string_view s, const Checkpoint &checkpoint) {
  while (!s.empty()) {
    check(checkpoint);
    const auto n =
        ::write(fd, s.data(), std::min<std::size_t>(65536, s.size()));
    if (n < 0 && errno == EINTR)
      continue;
    if (n <= 0)
      fail("bundle.write_failed", "file write failed");
    s.remove_prefix(static_cast<std::size_t>(n));
  }
}
Json parse(const std::string &bytes) {
  std::vector<std::set<std::string>> keys;
  auto callback = [&](int depth, Json::parse_event_t e, Json &v) {
    need(depth >= 0 && depth <= 72, "physical page nesting bound exceeded");
    if (e == Json::parse_event_t::object_start)
      keys.emplace_back();
    else if (e == Json::parse_event_t::object_end) {
      need(!keys.empty(), "invalid object stack");
      keys.pop_back();
    } else if (e == Json::parse_event_t::key) {
      need(!keys.empty() && keys.back().insert(str(v)).second,
           "duplicate physical JSON key");
    } else if (e == Json::parse_event_t::value)
      need(!v.is_number(), "numeric JSON literals prohibited");
    return true;
  };
  try {
    return Json::parse(bytes, callback);
  } catch (const StoreError &) {
    throw;
  } catch (const std::exception &) {
    fail("bundle.invalid_json", "invalid UTF-8/JSON page");
  }
}
struct Counts {
  std::uint64_t values = 1, nodes = 1, bytes = 0;
};
Counts counts(const Json &v, std::uint64_t depth = 0) {
  need(depth <= 64 && !v.is_number() && !v.is_binary() && !v.is_discarded(),
       "portable logical type/depth required");
  Counts out;
  if (v.is_string())
    need(v.get_ref<const std::string &>().size() <= 65536,
         "portable string bound exceeded");
  if (v.is_structured()) {
    out.bytes = 2;
    std::uint64_t n = 0;
    for (auto i = v.begin(); i != v.end(); ++i) {
      const auto c = counts(i.value(), depth + 1);
      out.values = add(out.values, c.values);
      out.nodes = add(out.nodes, c.nodes);
      out.bytes = add(out.bytes, c.bytes);
      if (n++)
        out.bytes = add(out.bytes, 1);
      if (v.is_object()) {
        need(i.key().size() <= 65536, "portable key bound exceeded");
        out.nodes = add(out.nodes, 1);
        out.bytes = add(out.bytes, add(Json(i.key()).dump().size(), 1));
      }
    }
  } else
    out.bytes = v.dump().size();
  return out;
}
Counts node_counts(const Json &n, std::uint64_t depth = 0) {
  need(n.is_object() && n.contains("kind"), "logical node descriptor required");
  if (n.at("kind") == "inline") {
    shape(n, {"kind", "value"});
    return counts(n.at("value"), depth);
  }
  shape(n, {"kind", "children", "values", "nodes", "bytes", "index"});
  need(n.at("kind") == "object" || n.at("kind") == "array",
       "unknown node kind");
  need(depth <= 64, "logical depth exceeded");
  const auto children = num(n.at("children"));
  Counts c{num(n.at("values")), num(n.at("nodes")), num(n.at("bytes"))};
  need(c.values >= add(children, 1) && c.nodes >= c.values && c.bytes >= 2,
       "invalid node counts");
  need((children == 0) == n.at("index").is_null(), "empty node index mismatch");
  if (children == 0)
    need(c.values == 1 && c.nodes == 1 && c.bytes == 2,
         "empty container metrics mismatch");
  return c;
}
std::string kind(const Json &n) {
  if (n.at("kind") != "inline")
    return str(n.at("kind"));
  const auto &v = n.at("value");
  if (v.is_object())
    return "object";
  if (v.is_array())
    return "array";
  if (v.is_string())
    return "string";
  if (v.is_boolean())
    return "boolean";
  return "null";
}
std::uint64_t child_count(const Json &n) {
  if (n.at("kind") != "inline")
    return num(n.at("children"));
  const auto &v = n.at("value");
  return v.is_structured() ? v.size() : 0;
}
bool container(const Json &n) {
  const auto k = kind(n);
  return k == "array" || k == "object";
}
Json inline_node(const Json &v) { return {{"kind", "inline"}, {"value", v}}; }
Json ref_json(const Reference &r) {
  return {{"manifest_path", r.manifest_path},
          {"manifest_sha256", r.manifest_sha256},
          {"content_sha256", r.content_sha256}};
}
void validate_ref(const Json &r, bool object, std::uint64_t total,
                  std::uint64_t page_bytes) {
  shape(r, {"sha256", "bytes", "level", "start", "count", "values", "first_key",
            "last_key"});
  digest(str(r.at("sha256")));
  need(num(r.at("bytes")) > 0 && num(r.at("bytes")) <= page_bytes,
       "page byte range invalid");
  const auto start = num(r.at("start")), n = num(r.at("count")),
             level = num(r.at("level"));
  need(n > 0 && start <= total && n <= total - start &&
           num(r.at("values")) >= n,
       "page child/value range invalid");
  need(total > 0 &&
           level <= static_cast<std::uint64_t>(std::bit_width(total - 1)),
       "index height exceeds count-derived bound");
  if (object) {
    const auto first = str(r.at("first_key")), last = str(r.at("last_key"));
    need(first.size() <= 65536 && last.size() <= 65536 && first <= last,
         "object key bounds invalid");
  } else
    need(r.at("first_key").is_null() && r.at("last_key").is_null(),
         "array has object key bounds");
}
struct Located {
  Json node;
  std::uint64_t id = 0;
  Json parent = nullptr, edge = nullptr;
  std::uint64_t depth = 0;
  bool root = false;
};
struct Engine {
  FD dir;
  Json manifest;
  ReadOptions options;
  Checkpoint checkpoint;
  IOStats stats;
  struct Cached {
    std::shared_ptr<const Json> value;
    std::uint64_t bytes;
    std::list<std::string>::iterator pos;
  };
  std::map<std::string, Cached> cache;
  std::list<std::string> order;
  std::uint64_t cached_bytes = 0;
  Engine(FD d, Json m, ReadOptions o, Checkpoint c)
      : dir(std::move(d)), manifest(std::move(m)), options(o),
        checkpoint(std::move(c)) {
    admit();
  }
  void admit();
  std::shared_ptr<const Json> page(const Json &, bool, std::uint64_t);
  struct Child {
    Json node, key;
    std::uint64_t index = 0, prefix_values = 0;
  };
  Child by_index(const Json &, std::uint64_t);
  Child by_value(const Json &, std::uint64_t);
  Child by_key(const Json &, const std::string &);
  void children(const Json &, const std::function<void(const Child &)> &);
  Located locate_id(std::uint64_t);
  Located locate_pointer(std::string_view);
  Json node_row(const Located &);
  Json query(const Located &, std::uint64_t, std::uint64_t, std::uint64_t,
             const Reference &);
  void semantics();
  Json traverse(const Sink &, const Sink &, const Reference &);
};
} // namespace

StoreError::StoreError(std::string c, std::string m, Json r)
    : std::runtime_error(std::move(m)), code(std::move(c)),
      recovery(std::move(r)) {}
Json IOStats::json() const {
  return {{"files_read", dec(files_read)},
          {"bytes_read", dec(bytes_read)},
          {"cache_hits", dec(cache_hits)},
          {"maximum_file_bytes", dec(maximum_file_bytes)}};
}
Json Receipt::json() const {
  return {{"reference", ref_json(reference)},
          {"manifest_bytes", dec(manifest_bytes)},
          {"logical_body_bytes", dec(logical_body_bytes)},
          {"logical_body_nodes", dec(logical_body_nodes)},
          {"logical_body_values", dec(logical_body_values)},
          {"exported_value_nodes", dec(add(logical_body_values, 1))},
          {"page_files_created", dec(page_files_created)},
          {"page_bytes_created", dec(page_bytes_created)},
          {"verification_extent", "full_logical_closure"},
          {"io_stats", io_stats.json()}};
}

struct ResultWriter::Impl {
  std::string path;
  WriteOptions options;
  Checkpoint checkpoint;
  FD dir;
  Hash body_hash;
  Json root = nullptr;
  bool begun = false, ended = false, finished = false, published = false,
       durable = false, poisoned = false;
  std::uint64_t serial = 0, files = 0, physical_bytes = 0;
  std::string phase = "create_directory";
  Json sealed_reference = nullptr;
  struct Frame {
    bool object;
    std::optional<std::string> pending, last;
    Counts metrics{1, 1, 2};
    std::uint64_t count = 0, flushed = 0, entry_bytes = 0;
    Json entries = Json::array();
    std::vector<std::vector<Json>> levels;
  };
  std::vector<Frame> stack;
  Impl(std::string p, WriteOptions o, Checkpoint c)
      : path(std::move(p)), options(o), checkpoint(std::move(c)) {
    need(options.page_bytes >= 512 && options.index_fanout >= 2,
         "page budget>=512 and fanout>=2 required");
    (void)host(options.page_bytes);
    (void)host(options.index_fanout);
    (void)path_parts(path + "/manifest.json");
    check(checkpoint);
    auto [parent_dir, name] = parent(path);
    if (::mkdirat(parent_dir.n, name.c_str(), 0700) != 0)
      fail("bundle.exists", "bundle directory must be new");
    try {
      dir = FD(::openat(parent_dir.n, name.c_str(),
                        O_RDONLY | O_DIRECTORY | O_CLOEXEC | O_NOFOLLOW));
      need(dir.n >= 0, "new bundle open failed");
      if (::fsync(parent_dir.n) != 0)
        fail("bundle.outcome_uncertain",
             "new bundle directory parent synchronization failed");
    } catch (StoreError &e) {
      e.recovery = recovery();
      throw;
    }
  }
  Json recovery() const {
    return {{"bundle_path", path},
            {"manifest_path", path + "/manifest.json"},
            {"phase", phase},
            {"manifest_published", published},
            {"durable", durable},
            {"page_files_created", dec(files)},
            {"page_bytes_created", dec(physical_bytes)},
            {"reference", sealed_reference}};
  }
  template <class F> auto guarded(F &&f) {
    need(!poisoned && !finished, "writer is failed or already finished");
    try {
      return f();
    } catch (StoreError &e) {
      poisoned = true;
      if (e.recovery.is_null())
        e.recovery = recovery();
      throw;
    } catch (const std::exception &e) {
      poisoned = true;
      throw StoreError("bundle.interrupted", e.what(), recovery());
    } catch (...) {
      poisoned = true;
      throw StoreError("bundle.interrupted", "writer callback failed",
                       recovery());
    }
  }
  void write_immutable(const std::string &name, const std::string &bytes,
                       bool manifest_file) {
    check(checkpoint);
    const auto temp = ".stage-" + dec(++serial);
    FD f(::openat(dir.n, temp.c_str(),
                  O_WRONLY | O_CREAT | O_EXCL | O_NOFOLLOW | O_CLOEXEC, 0600));
    need(f.n >= 0, "exclusive page staging failed");
    try {
      write_all(f.n, bytes, checkpoint);
      need(::fsync(f.n) == 0, "page synchronization failed");
      check(checkpoint);
      if (::linkat(dir.n, temp.c_str(), dir.n, name.c_str(), 0) != 0) {
        if (!manifest_file && errno == EEXIST) {
          const auto prior =
              read_file(dir.n, name, options.page_bytes, checkpoint);
          need(prior == bytes,
               "content-addressed existing page contradicts bytes");
          ::unlinkat(dir.n, temp.c_str(), 0);
          return;
        }
        fail("bundle.exists",
             "immutable destination already exists or publication failed");
      }
      if (manifest_file) {
        published = true;
        phase = "sync_manifest";
      } else {
        files = add(files, 1);
        physical_bytes = add(physical_bytes, bytes.size());
      }
      ::unlinkat(dir.n, temp.c_str(), 0);
      if (::fsync(dir.n) != 0)
        fail("bundle.outcome_uncertain",
             "published file directory synchronization failed");
      if (manifest_file)
        durable = true;
    } catch (...) {
      ::unlinkat(dir.n, temp.c_str(), 0);
      throw;
    }
  }
  Json store_page(bool object, std::uint64_t level, const Json &entries) {
    need(!entries.empty(), "empty index page prohibited");
    std::uint64_t start = 0, count = 0, values = 0;
    Json first = nullptr, last = nullptr;
    if (level == 0) {
      // Leaf ordinals are supplied separately in a temporary builder member.
      fail("bundle.internal", "leaf ordinal missing");
    }
    start = num(entries.front().at("start"));
    for (const auto &r : entries) {
      need(num(r.at("start")) == add(start, count) &&
               num(r.at("level")) + 1 == level,
           "builder index continuity");
      count = add(count, num(r.at("count")));
      values = add(values, num(r.at("values")));
    }
    if (object) {
      first = entries.front().at("first_key");
      last = entries.back().at("last_key");
    }
    return emit_page(object, level, start, count, values, first, last, entries);
  }
  Json emit_page(bool object, std::uint64_t level, std::uint64_t start,
                 std::uint64_t count, std::uint64_t values, Json first,
                 Json last, const Json &entries) {
    phase = "write_pages";
    Json p{
        {"protocol", page_protocol}, {"container", object ? "object" : "array"},
        {"level", dec(level)},       {"start", dec(start)},
        {"count", dec(count)},       {"values", dec(values)},
        {"entries", entries}};
    const auto bytes = p.dump();
    need(bytes.size() <= options.page_bytes,
         "selected page budget cannot hold index/entry");
    const auto sha = hash(bytes);
    write_immutable(sha + ".json", bytes, false);
    return {{"sha256", sha},
            {"bytes", dec(bytes.size())},
            {"level", dec(level)},
            {"start", dec(start)},
            {"count", dec(count)},
            {"values", dec(values)},
            {"first_key", std::move(first)},
            {"last_key", std::move(last)}};
  }
  void push(Frame &f, Json ref, std::size_t level) {
    if (f.levels.size() <= level)
      f.levels.resize(level + 1);
    auto &bucket = f.levels[level];
    // Byte-aware grouping as well as fanout. A unary non-root right fringe is
    // permitted; a pair unable to fit is a selected page-budget refusal.
    if (!bucket.empty()) {
      std::uint64_t estimate = 512;
      for (const auto &r : bucket)
        estimate = add(estimate, add(r.dump().size(), 1));
      estimate = add(estimate, ref.dump().size());
      if (estimate > options.page_bytes) {
        need(bucket.size() >= 2,
             "selected page budget cannot hold two index references");
        Json entries = bucket;
        bucket.clear();
        auto up = store_page(f.object, level + 1, entries);
        push(f, std::move(up), level + 1);
      }
    }
    // push may resize levels; reacquire its bucket reference.
    f.levels[level].push_back(std::move(ref));
    if (f.levels[level].size() >= options.index_fanout) {
      Json entries = f.levels[level];
      f.levels[level].clear();
      auto up = store_page(f.object, level + 1, entries);
      push(f, std::move(up), level + 1);
    }
  }
  void flush(Frame &f) {
    if (f.entries.empty())
      return;
    std::uint64_t values = 0;
    for (const auto &e : f.entries)
      values = add(values, node_counts(e.at("node")).values);
    Json first = nullptr, last = nullptr;
    if (f.object) {
      first = f.entries.front().at("key");
      last = f.entries.back().at("key");
    }
    const auto n = static_cast<std::uint64_t>(f.entries.size());
    auto r =
        emit_page(f.object, 0, f.flushed, n, values, first, last, f.entries);
    f.flushed = add(f.flushed, n);
    f.entries = Json::array();
    f.entry_bytes = 0;
    push(f, std::move(r), 0);
  }
  Json finish_index(Frame &f) {
    flush(f);
    if (f.count == 0)
      return nullptr;
    for (std::size_t level = 0; level < f.levels.size(); ++level) {
      if (f.levels[level].empty())
        continue;
      bool higher = false;
      for (std::size_t k = level + 1; k < f.levels.size(); ++k)
        higher |= !f.levels[k].empty();
      if (!higher && f.levels[level].size() == 1)
        return f.levels[level].front();
      Json entries = f.levels[level];
      f.levels[level].clear();
      auto up = store_page(f.object, level + 1, entries);
      push(f, std::move(up), level + 1);
    }
    fail("bundle.internal", "index builder lost root");
  }
  void prefix() {
    check(checkpoint);
    if (stack.empty()) {
      need(!begun, "one logical root required");
      begun = true;
      return;
    }
    auto &f = stack.back();
    if (f.count) {
      body_hash.update(",");
      f.metrics.bytes = add(f.metrics.bytes, 1);
    }
    if (f.object) {
      need(f.pending.has_value(), "object key required before value");
      const auto k = Json(*f.pending).dump();
      body_hash.update(k);
      body_hash.update(":");
      f.metrics.bytes = add(f.metrics.bytes, add(k.size(), 1));
    }
  }
  void attach(Json n) {
    const auto c = node_counts(n, stack.size());
    if (stack.empty()) {
      root = std::move(n);
      ended = true;
      return;
    }
    auto &f = stack.back();
    Json k = nullptr;
    if (f.object) {
      k = *f.pending;
      f.pending.reset();
    }
    Json entry{{"key", k}, {"node", std::move(n)}};
    const auto bytes = entry.dump().size();
    need(add(bytes, 512) <= options.page_bytes,
         "one entry exceeds selected page budget; stream subtree or select "
         "larger pages");
    if (add(add(f.entry_bytes, bytes), 512) > options.page_bytes)
      flush(f);
    f.entry_bytes = add(f.entry_bytes, add(bytes, 1));
    f.entries.push_back(std::move(entry));
    f.count = add(f.count, 1);
    f.metrics.values = add(f.metrics.values, c.values);
    f.metrics.nodes = add(f.metrics.nodes, add(c.nodes, f.object ? 1 : 0));
    f.metrics.bytes = add(f.metrics.bytes, c.bytes);
  }
  void begin(bool object) {
    prefix();
    need(stack.size() <= 64, "logical depth exceeded");
    body_hash.update(object ? "{" : "[");
    stack.push_back(
        Frame{object, {}, {}, {1, 1, 2}, 0, 0, 0, Json::array(), {}});
  }
};

ResultWriter::ResultWriter(std::string p, WriteOptions o, Checkpoint c)
    : p_(std::make_unique<Impl>(std::move(p), o, std::move(c))) {}
ResultWriter::~ResultWriter() = default;
void ResultWriter::begin_object() {
  p_->guarded([&] { p_->begin(true); });
}
void ResultWriter::begin_array() {
  p_->guarded([&] { p_->begin(false); });
}
void ResultWriter::key(std::string k) {
  p_->guarded([&] {
    check(p_->checkpoint);
    need(!p_->stack.empty(), "key outside object");
    auto &f = p_->stack.back();
    need(f.object && !f.pending, "key not expected");
    need(k.size() <= 65536, "portable key bound exceeded");
    (void)Json(k).dump();
    need(!f.last || *f.last < k,
         "canonical sorted unique object keys required");
    f.pending = k;
    f.last = std::move(k);
  });
}
void ResultWriter::value(const Json &v) {
  p_->guarded([&] {
    const auto c = counts(v, p_->stack.size());
    (void)c;
    p_->prefix();
    const auto bytes = v.dump();
    p_->body_hash.update(bytes);
    p_->attach(inline_node(v));
  });
}
void ResultWriter::end() {
  p_->guarded([&] {
    check(p_->checkpoint);
    need(!p_->stack.empty(), "unmatched container end");
    auto &top = p_->stack.back();
    need(!top.pending, "object key lacks value");
    p_->body_hash.update(top.object ? "}" : "]");
    bool compact = p_->stack.size() > 1 && top.flushed == 0;
    for (const auto &e : top.entries)
      compact &= e.at("node").at("kind") == "inline";
    Json n;
    if (compact) {
      Json value = top.object ? Json::object() : Json::array();
      for (const auto &e : top.entries) {
        if (top.object)
          value[str(e.at("key"))] = e.at("node").at("value");
        else
          value.push_back(e.at("node").at("value"));
      }
      n = inline_node(value);
      const auto &parent_frame = p_->stack[p_->stack.size() - 2];
      const Json parent_key = parent_frame.object
                                  ? Json(*parent_frame.pending)
                                  : Json(nullptr);
      const Json parent_entry{{"key", parent_key}, {"node", n}};
      compact = add(parent_entry.dump().size(), 512) <= p_->options.page_bytes;
    }
    if (!compact) {
      auto index = p_->finish_index(top);
      n = {{"kind", top.object ? "object" : "array"},
           {"children", dec(top.count)},
           {"values", dec(top.metrics.values)},
           {"nodes", dec(top.metrics.nodes)},
           {"bytes", dec(top.metrics.bytes)},
           {"index", std::move(index)}};
    }
    p_->stack.pop_back();
    p_->attach(std::move(n));
  });
}
Receipt ResultWriter::finish(std::optional<std::string> expected) {
  return p_->guarded([&] {
    need(p_->ended && p_->stack.empty() && !p_->root.is_null(),
         "logical body incomplete");
    p_->phase = "write_pages";
    const auto content = p_->body_hash.finish();
    if (expected) {
      digest(*expected);
      need(*expected == content, "input logical content digest mismatch");
    }
    Json m{{"protocol", manifest_protocol},
           {"logical_protocol", logical_protocol},
           {"canonicalization", canonicalization},
           {"content_sha256", content},
           {"page_bytes", dec(p_->options.page_bytes)},
           {"index_fanout", dec(p_->options.index_fanout)},
           {"root", p_->root}};
    const auto bytes = m.dump();
    need(bytes.size() <= manifest_budget, "manifest format budget exceeded");
    Reference reference{p_->path + "/manifest.json", hash(bytes), content};
    p_->sealed_reference = ref_json(reference);
    Engine verification(open_directory(p_->path), m, {}, p_->checkpoint);
    verification.traverse({}, {}, reference);
    p_->phase = "publish_manifest";
    p_->write_immutable("manifest.json", bytes, true);
    p_->finished = true;
    const auto c = node_counts(p_->root);
    return Receipt{
        reference, bytes.size(),       c.bytes,           c.nodes, c.values,
        p_->files, p_->physical_bytes, verification.stats};
  });
}
Json ResultWriter::recovery() const { return p_->recovery(); }

namespace {
void Engine::admit() {
  shape(manifest, {"protocol", "logical_protocol", "canonicalization",
                   "content_sha256", "page_bytes", "index_fanout", "root"});
  need(manifest.at("protocol") == manifest_protocol &&
           manifest.at("logical_protocol") == logical_protocol &&
           manifest.at("canonicalization") == canonicalization,
       "unsupported storage/logical contract");
  digest(str(manifest.at("content_sha256")));
  const auto budget = num(manifest.at("page_bytes"));
  need(budget >= 512 && num(manifest.at("index_fanout")) >= 2,
       "invalid page layout selection");
  (void)host(budget);
  if (options.max_page_bytes)
    need(budget <= *options.max_page_bytes,
         "selected reader page budget exceeded");
  const auto &root = manifest.at("root");
  const auto c = node_counts(root);
  need(kind(root) == "object" && root.at("kind") != "inline" &&
           child_count(root) == 4,
       "result body root must be four-member streamed object");
  (void)add(c.values, 1);
  (void)add(c.nodes, 2);
  validate_ref(root.at("index"), true, 4, budget);
  need(num(root.at("index").at("start")) == 0 &&
           num(root.at("index").at("count")) == 4 &&
           num(root.at("index").at("values")) == c.values - 1,
       "root index metrics mismatch");
}
std::shared_ptr<const Json> Engine::page(const Json &r, bool object,
                                         std::uint64_t total) {
  check(checkpoint);
  const auto budget = num(manifest.at("page_bytes"));
  validate_ref(r, object, total, budget);
  const auto sha = str(r.at("sha256"));
  std::shared_ptr<const Json> result;
  auto found = cache.find(sha);
  if (found != cache.end()) {
    stats.cache_hits = add(stats.cache_hits, 1);
    order.splice(order.begin(), order, found->second.pos);
    result = found->second.value;
  } else {
    const auto bytes =
        read_file(dir.n, sha + ".json", num(r.at("bytes")), checkpoint, &stats);
    if (bytes.size() != num(r.at("bytes")) || hash(bytes) != sha)
      fail("bundle.corrupt", "page byte digest/length mismatch");
    auto p = parse(bytes);
    need(p.dump() == bytes, "physical page is not canonical compact JSON");
    shape(p, {"protocol", "container", "level", "start", "count", "values",
              "entries"});
    need(p.at("protocol") == page_protocol && p.at("entries").is_array() &&
             !p.at("entries").empty(),
         "invalid page envelope");
    const auto level = num(p.at("level")), start = num(p.at("start"));
    std::uint64_t count = 0, values = 0;
    std::optional<std::string> last;
    for (const auto &e : p.at("entries")) {
      check(checkpoint);
      if (level == 0) {
        shape(e, {"key", "node"});
        const auto c = node_counts(e.at("node"));
        values = add(values, c.values);
        count = add(count, 1);
        if (object) {
          const auto key = str(e.at("key"));
          need(key.size() <= 65536 && (!last || *last < key),
               "leaf keys must be ordered unique");
          last = key;
        } else
          need(e.at("key").is_null(), "array leaf key must be null");
      } else {
        validate_ref(e, object, total, budget);
        need(num(e.at("level")) == level - 1 &&
                 num(e.at("start")) == add(start, count),
             "index child level/range discontinuity");
        count = add(count, num(e.at("count")));
        values = add(values, num(e.at("values")));
        if (object) {
          const auto first = str(e.at("first_key")),
                     final = str(e.at("last_key"));
          need(!last || *last < first, "branch key ranges overlap");
          last = final;
        }
      }
    }
    need(count == num(p.at("count")) && values == num(p.at("values")),
         "page cumulative metrics mismatch");
    if (level > 0)
      need(p.at("entries").size() <= num(manifest.at("index_fanout")),
           "index fanout mismatch");
    result = std::make_shared<const Json>(std::move(p));
    const auto n = static_cast<std::uint64_t>(bytes.size());
    if (n <= options.cache_bytes) {
      while (cached_bytes > options.cache_bytes - n) {
        auto oldest = std::prev(order.end());
        auto at = cache.find(*oldest);
        cached_bytes -= at->second.bytes;
        cache.erase(at);
        order.erase(oldest);
      }
      order.push_front(sha);
      cache.emplace(sha, Cached{result, n, order.begin()});
      cached_bytes = add(cached_bytes, n);
    }
  }
  const auto &p = *result;
  need(p.at("container") == (object ? "object" : "array") &&
           p.at("level") == r.at("level") && p.at("start") == r.at("start") &&
           p.at("count") == r.at("count") && p.at("values") == r.at("values"),
       "page contradicts authenticated reference");
  if (object) {
    const auto level = num(p.at("level"));
    const auto &entries = p.at("entries");
    const auto first =
        level ? entries.front().at("first_key") : entries.front().at("key");
    const auto last =
        level ? entries.back().at("last_key") : entries.back().at("key");
    need(first == r.at("first_key") && last == r.at("last_key"),
         "page key boundary mismatch");
  }
  return result;
}
Engine::Child seek(Engine &e, const Json &node, int mode, std::uint64_t target,
                   const std::string &key = {}) {
  const auto total = child_count(node);
  need(total > 0, "selected child absent");
  const bool object = kind(node) == "object";
  if (node.at("kind") == "inline") {
    std::uint64_t index = 0, prefix = 0;
    const auto &v = node.at("value");
    for (auto i = v.begin(); i != v.end(); ++i) {
      const auto c = counts(i.value());
      if ((mode == 0 && target == index) ||
          (mode == 1 && target >= prefix && target - prefix < c.values) ||
          (mode == 2 && object && i.key() == key))
        return {inline_node(i.value()), object ? Json(i.key()) : Json(nullptr),
                index, prefix};
      prefix = add(prefix, c.values);
      ++index;
    }
    fail("bundle.not_found", "selected child not present");
  }
  const auto &index = node.at("index");
  validate_ref(index, object, total, num(e.manifest.at("page_bytes")));
  const auto metrics = node_counts(node);
  need(num(index.at("start")) == 0 && num(index.at("count")) == total &&
           num(index.at("values")) == metrics.values - 1,
       "node index metrics mismatch");
  Json current = index;
  std::uint64_t prefix = 0;
  for (;;) {
    const auto owned = e.page(current, object, total);
    const auto &p = *owned;
    const auto level = num(p.at("level"));
    if (current == index && level > 0)
      need(p.at("entries").size() >= 2, "unnecessary unary index root");
    bool matched = false;
    std::uint64_t ordinal = num(p.at("start"));
    for (const auto &entry : p.at("entries")) {
      const auto values = level ? num(entry.at("values"))
                                : node_counts(entry.at("node")).values;
      const auto count = level ? num(entry.at("count")) : 1;
      const bool hit =
          mode == 0 ? (target >= ordinal && target - ordinal < count)
          : mode == 1
              ? (target >= prefix && target - prefix < values)
              : (object && (level ? (str(entry.at("first_key")) <= key &&
                                     key <= str(entry.at("last_key")))
                                  : str(entry.at("key")) == key));
      if (hit) {
        if (level == 0)
          return {entry.at("node"), entry.at("key"), ordinal, prefix};
        current = entry;
        matched = true;
        break;
      }
      prefix = add(prefix, values);
      ordinal = add(ordinal, count);
    }
    if (!matched)
      fail("bundle.not_found", "selected child not present");
  }
}
Engine::Child Engine::by_index(const Json &n, std::uint64_t i) {
  need(i < child_count(n), "child ordinal outside container");
  return seek(*this, n, 0, i);
}
Engine::Child Engine::by_value(const Json &n, std::uint64_t i) {
  need(i < node_counts(n).values - 1, "value ordinal outside descendants");
  return seek(*this, n, 1, i);
}
Engine::Child Engine::by_key(const Json &n, const std::string &key) {
  need(kind(n) == "object", "object required for key");
  return seek(*this, n, 2, 0, key);
}
void Engine::children(const Json &node,
                      const std::function<void(const Child &)> &f) {
  const auto total = child_count(node);
  if (!total)
    return;
  const bool object = kind(node) == "object";
  if (node.at("kind") == "inline") {
    const auto &v = node.at("value");
    std::uint64_t index = 0, prefix = 0;
    for (auto i = v.begin(); i != v.end(); ++i) {
      check(checkpoint);
      f(Child{inline_node(i.value()), object ? Json(i.key()) : Json(nullptr),
              index, prefix});
      prefix = add(prefix, counts(i.value()).values);
      ++index;
    }
    return;
  }
  const auto c = node_counts(node);
  const auto &index = node.at("index");
  need(num(index.at("start")) == 0 && num(index.at("count")) == total &&
           num(index.at("values")) == c.values - 1,
       "container index mismatch");
  std::uint64_t ordinal = 0, prefix = 0;
  std::function<void(const Json &, bool)> walk = [&](const Json &r, bool root) {
    const auto owned = page(r, object, total);
    const auto &p = *owned;
    if (root && num(p.at("level")) > 0)
      need(p.at("entries").size() >= 2, "unnecessary unary index root");
    need(num(p.at("start")) == ordinal, "traversal range discontinuity");
    if (num(p.at("level")) == 0) {
      for (const auto &entry : p.at("entries")) {
        f(Child{entry.at("node"), entry.at("key"), ordinal, prefix});
        prefix = add(prefix, node_counts(entry.at("node")).values);
        ordinal = add(ordinal, 1);
      }
    } else
      for (const auto &entry : p.at("entries"))
        walk(entry, false);
  };
  walk(index, true);
  need(ordinal == total && prefix == c.values - 1,
       "traversal aggregate range mismatch");
}
Located Engine::locate_id(std::uint64_t target) {
  const auto &root = manifest.at("root");
  need(target < add(node_counts(root).values, 1), "node ID out of range");
  if (target == 0)
    return {root, 0, nullptr, nullptr, 0, true};
  if (target == 1)
    return {inline_node(manifest.at("content_sha256")),
            1,
            "0",
            "content_sha256",
            1,
            false};
  Located cur{root, 0, nullptr, nullptr, 0, true};
  while (cur.id != target) {
    const auto start = add(cur.id, cur.root ? 2 : 1);
    need(target >= start, "node ordinal contradiction");
    const auto child = by_value(cur.node, target - start);
    const auto id = add(start, child.prefix_values);
    const auto edge =
        kind(cur.node) == "object" ? child.key : Json(dec(child.index));
    cur = {child.node, id, dec(cur.id), edge, add(cur.depth, 1), false};
    (void)node_counts(cur.node, cur.depth);
  }
  return cur;
}
std::vector<std::string> pointer_tokens(std::string_view pointer) {
  if (pointer.empty())
    return {};
  need(pointer.front() == '/', "JSON Pointer must start with slash");
  std::vector<std::string> out;
  std::size_t pos = 1;
  for (;;) {
    auto end = pointer.find('/', pos);
    if (end == std::string_view::npos)
      end = pointer.size();
    std::string token;
    for (auto i = pos; i < end; ++i) {
      if (pointer[i] != '~')
        token += pointer[i];
      else {
        need(++i < end && (pointer[i] == '0' || pointer[i] == '1'),
             "invalid JSON Pointer escape");
        token += pointer[i] == '0' ? '~' : '/';
      }
    }
    out.push_back(std::move(token));
    if (end == pointer.size())
      break;
    pos = end + 1;
  }
  return out;
}
Located Engine::locate_pointer(std::string_view pointer) {
  Located cur{manifest.at("root"), 0, nullptr, nullptr, 0, true};
  for (const auto &token : pointer_tokens(pointer)) {
    check(checkpoint);
    if (cur.root && token == "content_sha256") {
      cur = {inline_node(manifest.at("content_sha256")),
             1,
             "0",
             "content_sha256",
             1,
             false};
      continue;
    }
    need(container(cur.node), "pointer crosses scalar");
    const bool object = kind(cur.node) == "object";
    const auto c =
        object ? by_key(cur.node, token) : by_index(cur.node, num(Json(token)));
    cur = {c.node,
           add(add(cur.id, cur.root ? 2 : 1), c.prefix_values),
           dec(cur.id),
           object ? c.key : Json(dec(c.index)),
           add(cur.depth, 1),
           false};
    (void)node_counts(cur.node, cur.depth);
  }
  return cur;
}
Json Engine::node_row(const Located &n) {
  const auto k = kind(n.node);
  Json value;
  if (k == "object")
    value = Json::object();
  else if (k == "array")
    value = Json::array();
  else
    value = n.node.at("value");
  return {{"node_id", dec(n.id)},
          {"parent_id", n.parent},
          {"edge", n.edge},
          {"kind", container(n.node) ? k : "scalar"},
          {"value", std::move(value)},
          {"children", dec(add(child_count(n.node), n.root ? 1 : 0))}};
}
Json Engine::query(const Located &selected, std::uint64_t offset,
                   std::uint64_t limit, std::uint64_t byte_limit,
                   const Reference &reference) {
  need(limit > 0 && byte_limit > 0, "positive page row/byte limits required");
  const auto total = container(selected.node) ? add(child_count(selected.node),
                                                    selected.root ? 1 : 0)
                                              : 1;
  need(offset <= total, "query offset beyond selected node");
  Json out{{"reference", ref_json(reference)},
           {"selected_node_id", dec(selected.id)},
           {"selected_kind",
            container(selected.node) ? kind(selected.node) : "scalar"},
           {"total", dec(total)},
           {"offset", dec(offset)},
           {"next_offset", dec(offset)},
           {"complete", offset == total},
           {"nodes", Json::array()},
           {"verification_extent", "accessed_pages"},
           {"io_stats", stats.json()}};
  for (auto at = offset; at < total && out["nodes"].size() < limit; ++at) {
    check(checkpoint);
    Located n = selected;
    if (container(selected.node)) {
      if (selected.root && at == 0)
        n = {inline_node(manifest.at("content_sha256")),
             1,
             "0",
             "content_sha256",
             1,
             false};
      else {
        const auto c = by_index(selected.node, at - (selected.root ? 1 : 0));
        n = {c.node,
             add(add(selected.id, selected.root ? 2 : 1), c.prefix_values),
             dec(selected.id),
             kind(selected.node) == "object" ? c.key : Json(dec(c.index)),
             add(selected.depth, 1),
             false};
        (void)node_counts(n.node, n.depth);
      }
    }
    out["nodes"].push_back(node_row(n));
    out["next_offset"] = dec(at + 1);
    out["complete"] = at + 1 == total;
    out["io_stats"] = stats.json();
    if (add(out.dump().size(), 256) > byte_limit) {
      out["nodes"].erase(out["nodes"].end() - 1);
      need(!out["nodes"].empty(), "one query row exceeds selected byte budget");
      out["next_offset"] = dec(at);
      out["complete"] = false;
      break;
    }
  }
  out["io_stats"] = stats.json();
  need(out.dump().size() <= byte_limit,
       "query metadata exceeds selected byte budget");
  return out;
}
void Engine::semantics() {
  const auto &root = manifest.at("root");
  const auto origin = by_key(root, "origin").node,
             protocol = by_key(root, "protocol").node,
             status = by_key(root, "status").node,
             sections = by_key(root, "sections").node;
  need(kind(origin) == "string" && kind(protocol) == "string" &&
           protocol.at("value") == logical_protocol &&
           kind(status) == "string" &&
           (status.at("value") == "completed" ||
            status.at("value") == "partial") &&
           kind(sections) == "object",
       "invalid result-v1 root semantics");
  std::set<std::string> required{"summary",       "signals", "execution",
                                 "distributions", "studies", "replay",
                                 "comparisons",   "search",  "resources",
                                 "diagnostics",   "choices", "provenance"};
  children(sections, [&](const Child &s) {
    need(kind(s.node) == "object" && child_count(s.node) == 3,
         "section needs exact data/reason/status object");
    const auto data = by_key(s.node, "data");
    (void)data;
    const auto reason = by_key(s.node, "reason").node,
               state = by_key(s.node, "status").node;
    need(kind(reason) == "string" && kind(state) == "string",
         "section reason/status must be strings");
    const auto tag = str(state.at("value"));
    need(tag == "available" || tag == "partial" || tag == "not_selected" ||
             tag == "unavailable",
         "unknown section status");
    need(tag == "available" || !str(reason.at("value")).empty(),
         "nonavailable section reason required");
    required.erase(str(s.key));
  });
  need(required.empty(), "required result section absent");
}
Json aggregate(const Json &m, const Reference &r, const IOStats &s,
               std::string extent) {
  const auto c = node_counts(m.at("root"));
  return {{"reference", ref_json(r)},
          {"logical_body_bytes", dec(c.bytes)},
          {"logical_body_nodes", dec(c.nodes)},
          {"logical_body_values", dec(c.values)},
          {"exported_value_nodes", dec(add(c.values, 1))},
          {"verification_extent", std::move(extent)},
          {"io_stats", s.json()}};
}
Json Engine::traverse(const Sink &json_sink, const Sink &node_sink,
                      const Reference &reference) {
  semantics();
  Hash content;
  std::uint64_t observed_values = 0;
  auto emit = [&](std::string_view text, bool root_close = false) {
    content.update(text);
    if (json_sink && !root_close)
      json_sink(text);
  };
  std::function<Counts(const Located &)> visit =
      [&](const Located &n) -> Counts {
    check(checkpoint);
    need(n.depth <= 64, "logical depth exceeded");
    const auto declared = node_counts(n.node, n.depth);
    if (node_sink) {
      auto row = node_row(n);
      row["event"] = "node";
      node_sink(row.dump() + "\n");
    }
    observed_values = add(observed_values, 1);
    const auto k = kind(n.node);
    if (!container(n.node)) {
      const auto text = n.node.at("value").dump();
      emit(text);
      return counts(n.node.at("value"), n.depth);
    }
    const bool object = k == "object";
    emit(object ? "{" : "[");
    if (n.root) {
      if (json_sink)
        json_sink("\"content_sha256\":" +
                  Json(reference.content_sha256).dump() + ",");
      if (node_sink) {
        auto row = node_row(Located{inline_node(reference.content_sha256), 1,
                                    "0", "content_sha256", 1, false});
        row["event"] = "node";
        node_sink(row.dump() + "\n");
      }
    }
    Counts actual{1, 1, 2};
    std::uint64_t count = 0;
    children(n.node, [&](const Child &c) {
      if (count++) {
        emit(",");
        actual.bytes = add(actual.bytes, 1);
      }
      if (object) {
        const auto name = Json(str(c.key)).dump();
        emit(name);
        emit(":");
        actual.bytes = add(actual.bytes, add(name.size(), 1));
        actual.nodes = add(actual.nodes, 1);
      }
      Located child{
          c.node,          add(add(n.id, n.root ? 2 : 1), c.prefix_values),
          dec(n.id),       object ? c.key : Json(dec(c.index)),
          add(n.depth, 1), false};
      const auto got = visit(child);
      actual.bytes = add(actual.bytes, got.bytes);
      actual.values = add(actual.values, got.values);
      actual.nodes = add(actual.nodes, got.nodes);
    });
    emit(object ? "}" : "]", n.root);
    need(count == child_count(n.node) && actual.values == declared.values &&
             actual.nodes == declared.nodes && actual.bytes == declared.bytes,
         "logical subtree metrics mismatch");
    return actual;
  };
  const auto c =
      visit(Located{manifest.at("root"), 0, nullptr, nullptr, 0, true});
  need(c.values == observed_values, "logical traversal value count mismatch");
  if (content.finish() != reference.content_sha256)
    fail("bundle.logical_digest", "full logical content digest mismatch");
  check(checkpoint);
  if (json_sink)
    json_sink("}");
  return aggregate(manifest, reference, stats, "full_logical_closure");
}
} // namespace

struct ResultReader::Impl {
  Reference reference;
  std::unique_ptr<Engine> engine;
  Impl(Reference r, ReadOptions o, Checkpoint c) : reference(std::move(r)) {
    digest(reference.manifest_sha256);
    digest(reference.content_sha256);
    auto [dir, name] = parent(reference.manifest_path);
    need(name == "manifest.json", "manifest filename must be manifest.json");
    IOStats stats;
    auto bytes = read_file(dir.n, name, manifest_budget, c, &stats);
    if (hash(bytes) != reference.manifest_sha256)
      fail("bundle.corrupt", "root manifest digest mismatch");
    auto m = parse(bytes);
    need(m.dump() == bytes, "manifest is not canonical compact JSON");
    need(m.at("content_sha256") == reference.content_sha256,
         "manifest logical digest selection mismatch");
    engine =
        std::make_unique<Engine>(std::move(dir), std::move(m), o, std::move(c));
    engine->stats = stats;
  }
};
ResultReader::ResultReader(Reference r, ReadOptions o, Checkpoint c)
    : p_(std::make_unique<Impl>(std::move(r), o, std::move(c))) {}
ResultReader::~ResultReader() = default;
Json ResultReader::inspect() const {
  const auto &e = *p_->engine;
  auto out = aggregate(e.manifest, p_->reference, e.stats, "manifest_only");
  out["storage_protocol"] = manifest_protocol;
  out["logical_protocol"] = logical_protocol;
  out["canonicalization"] = canonicalization;
  out["root_node_id"] = "0";
  out["root_children"] = "5";
  out["write_options"] = {{"page_bytes", e.manifest.at("page_bytes")},
                          {"index_fanout", e.manifest.at("index_fanout")}};
  return out;
}
Json ResultReader::query(std::string_view pointer, std::uint64_t offset,
                         std::uint64_t rows, std::uint64_t bytes) {
  auto &e = *p_->engine;
  return e.query(e.locate_pointer(pointer), offset, rows, bytes, p_->reference);
}
Json ResultReader::query_node(std::uint64_t id, std::uint64_t offset,
                              std::uint64_t rows, std::uint64_t bytes) {
  auto &e = *p_->engine;
  return e.query(e.locate_id(id), offset, rows, bytes, p_->reference);
}
Json ResultReader::export_json(const Sink &sink) {
  need(static_cast<bool>(sink), "export sink required");
  return p_->engine->traverse(sink, {}, p_->reference);
}
Json ResultReader::export_ndjson(const Sink &sink) {
  need(static_cast<bool>(sink), "export sink required");
  Json begin{
      {"event", "begin"},
      {"protocol", "symphony.sbv.node-stream.v1"},
      {"reference", ref_json(p_->reference)},
      {"logical_protocol", logical_protocol},
      {"order", "canonical_preorder_values.v1"},
      {"source_authorship", "not_verified"},
      {"expected_value_nodes",
       dec(add(node_counts(p_->engine->manifest.at("root")).values, 1))}};
  sink(begin.dump() + "\n");
  auto result = p_->engine->traverse({}, sink, p_->reference);
  Json end{{"event", "end"},
           {"status", "complete"},
           {"nodes", result.at("exported_value_nodes")},
           {"content_sha256", p_->reference.content_sha256},
           {"manifest_sha256", p_->reference.manifest_sha256}};
  sink(end.dump() + "\n");
  return result;
}
Json ResultReader::verify_closure() {
  return p_->engine->traverse({}, {}, p_->reference);
}
IOStats ResultReader::io_stats() const { return p_->engine->stats; }
const Reference &ResultReader::reference() const { return p_->reference; }

namespace {
class Input {
  FD file_;
  Checkpoint checkpoint_;
  std::array<char, 65536> buffer_{};
  std::size_t begin_ = 0, end_ = 0;
  bool eof_ = false;

public:
  Hash hash_state;
  std::uint64_t bytes = 0;
  Input(const std::string &path, Checkpoint c) : checkpoint_(std::move(c)) {
    auto [dir, name] = parent(path);
    file_ = open_file(dir.n, name);
  }
  int peek() {
    if (begin_ == end_ && !eof_) {
      check(checkpoint_);
      for (;;) {
        auto n = ::read(file_.n, buffer_.data(), buffer_.size());
        if (n < 0 && errno == EINTR)
          continue;
        if (n < 0)
          fail("bundle.read_failed", "import file read failed");
        begin_ = 0;
        end_ = static_cast<std::size_t>(n);
        if (n == 0) {
          eof_ = true;
          break;
        }
        bytes = add(bytes, static_cast<std::uint64_t>(n));
        hash_state.update(std::string_view(buffer_.data(), end_));
        break;
      }
    }
    return begin_ == end_ ? -1 : static_cast<unsigned char>(buffer_[begin_]);
  }
  int get() {
    const auto c = peek();
    if (c >= 0)
      ++begin_;
    return c;
  }
  void whitespace() {
    for (;;) {
      const auto c = peek();
      if (c != ' ' && c != '\n' && c != '\r' && c != '\t')
        return;
      get();
    }
  }
  void expected(char c) {
    whitespace();
    need(get() == c, "unexpected import JSON token");
  }
  std::string string() {
    whitespace();
    need(get() == '"', "import string expected");
    std::string raw = "\"";
    bool escape = false;
    for (;;) {
      const auto c = get();
      need(c >= 0, "unterminated import string");
      raw += static_cast<char>(c);
      need(raw.size() <= 6U * 65536U + 2,
           "import token exceeds portable string encoding bound");
      if (!escape && c == '"')
        break;
      if (!escape && c == '\\')
        escape = true;
      else
        escape = false;
    }
    Json parsed;
    try {
      parsed = Json::parse(raw);
    } catch (const std::exception &) {
      fail("bundle.invalid_json", "invalid import string/UTF-8");
    }
    const auto text = str(parsed);
    need(text.size() <= 65536, "portable import string bound exceeded");
    return text;
  }
  void literal(std::string_view token) {
    for (const auto c : token)
      need(get() == c, "invalid JSON literal");
  }
};
void import_value(Input &i, ResultWriter &w, std::uint64_t depth) {
  need(depth <= 64, "portable import logical depth exceeded");
  i.whitespace();
  const auto c = i.peek();
  if (c == '{') {
    i.get();
    w.begin_object();
    i.whitespace();
    if (i.peek() == '}') {
      i.get();
      w.end();
      return;
    }
    std::optional<std::string> prior;
    for (;;) {
      auto key = i.string();
      need(!prior || *prior < key, "import keys must be sorted and unique");
      prior = key;
      w.key(std::move(key));
      i.expected(':');
      import_value(i, w, depth + 1);
      i.whitespace();
      const auto next = i.get();
      if (next == '}')
        break;
      need(next == ',', "object separator expected");
    }
    w.end();
  } else if (c == '[') {
    i.get();
    w.begin_array();
    i.whitespace();
    if (i.peek() == ']') {
      i.get();
      w.end();
      return;
    }
    for (;;) {
      import_value(i, w, depth + 1);
      i.whitespace();
      const auto next = i.get();
      if (next == ']')
        break;
      need(next == ',', "array separator expected");
    }
    w.end();
  } else if (c == '"')
    w.value(i.string());
  else if (c == 't') {
    i.literal("true");
    w.value(true);
  } else if (c == 'f') {
    i.literal("false");
    w.value(false);
  } else if (c == 'n') {
    i.literal("null");
    w.value(nullptr);
  } else
    fail("bundle.invalid_json",
         "only portable strings/booleans/null/containers are admitted");
}
Json initial_import_recovery(const std::string &path) {
  return {{"bundle_path", path},
          {"manifest_path", path + "/manifest.json"},
          {"phase", "create_directory"},
          {"manifest_published", false},
          {"durable", false},
          {"page_files_created", "0"},
          {"page_bytes_created", "0"},
          {"reference", nullptr}};
}
} // namespace
Json import_json(const std::string &input_path,
                 const std::string &expected_file_sha256,
                 const std::string &bundle_path, WriteOptions options,
                 Checkpoint checkpoint) {
  digest(expected_file_sha256);
  (void)path_parts(input_path);
  (void)path_parts(bundle_path + "/manifest.json");
  std::unique_ptr<ResultWriter> writer;
  try {
    Input input(input_path, checkpoint);
    writer = std::make_unique<ResultWriter>(bundle_path, options, checkpoint);
    input.expected('{');
    writer->begin_object();
    need(input.string() == "content_sha256",
         "canonical result import requires digest first");
    input.expected(':');
    const auto expected_content = input.string();
    digest(expected_content);
    input.expected(',');
    std::optional<std::string> prior = "content_sha256";
    for (;;) {
      auto key = input.string();
      need(*prior < key, "import root keys must be sorted and unique");
      prior = key;
      writer->key(std::move(key));
      input.expected(':');
      import_value(input, *writer, 1);
      input.whitespace();
      const auto next = input.get();
      if (next == '}')
        break;
      need(next == ',', "root separator expected");
    }
    writer->end();
    input.whitespace();
    need(input.peek() == -1, "trailing non-whitespace import bytes");
    const auto input_hash = input.hash_state.finish();
    need(input_hash == expected_file_sha256,
         "import input file digest mismatch");
    auto receipt = writer->finish(expected_content);
    auto result = receipt.json();
    result["input_path"] = input_path;
    result["input_file_sha256"] = input_hash;
    result["input_bytes"] = dec(input.bytes);
    result["bundle_path"] = bundle_path;
    result["write_options"] = {{"page_bytes", dec(options.page_bytes)},
                               {"index_fanout", dec(options.index_fanout)}};
    return result;
  } catch (StoreError &e) {
    if (e.recovery.is_null())
      e.recovery =
          writer ? writer->recovery() : initial_import_recovery(bundle_path);
    throw;
  } catch (const std::exception &e) {
    throw StoreError("bundle.interrupted", e.what(),
                     writer ? writer->recovery()
                            : initial_import_recovery(bundle_path));
  }
}
Json export_file(ResultReader &reader, const std::string &output_path,
                 std::string_view format, Checkpoint checkpoint) {
  Json recovery{{"output_path", output_path}, {"stage_path", nullptr},
                {"phase", "create_stage"},    {"published", false},
                {"durable", false},           {"expected_binary", nullptr}};
  FD dir, file;
  std::string stage;
  try {
    need(format == "json" || format == "ndjson",
         "export format must be json or ndjson");
    check(checkpoint);
    auto opened = parent(output_path);
    dir = std::move(opened.first);
    const auto leaf = std::move(opened.second);
    struct stat st{};
    need(::fstatat(dir.n, leaf.c_str(), &st, AT_SYMLINK_NOFOLLOW) != 0 &&
             errno == ENOENT,
         "export destination must not exist");
    // Stable identity plus exclusive process-local suffix; collisions refuse.
    stage = ".sbv-bundle-export-" +
            dec(static_cast<std::uint64_t>(::getpid())) + "-" +
            hash(output_path).substr(0, 24) + ".tmp";
    const auto stage_path =
        std::filesystem::path(output_path).parent_path().string() + "/" + stage;
    (void)path_parts(stage_path);
    file = FD(::openat(dir.n, stage.c_str(),
                       O_WRONLY | O_CREAT | O_EXCL | O_CLOEXEC | O_NOFOLLOW,
                       0600));
    need(file.n >= 0, "exclusive export stage creation failed");
    recovery["stage_path"] = stage_path;
    recovery["phase"] = "write_export";
    Hash digest_state;
    std::uint64_t bytes = 0;
    std::string last;
    std::array<char, 65536> output_buffer{};
    std::size_t buffered = 0;
    auto flush = [&] {
      if (buffered) {
        write_all(file.n, std::string_view(output_buffer.data(), buffered), checkpoint);
        buffered = 0;
      }
    };
    auto sink = [&](std::string_view text) {
      check(checkpoint);
      digest_state.update(text);
      bytes = add(bytes, text.size());
      if (format == "ndjson")
        last = text;
      while (!text.empty()) {
        const auto take = std::min(text.size(), output_buffer.size() - buffered);
        std::memcpy(output_buffer.data() + buffered, text.data(), take);
        buffered += take;
        text.remove_prefix(take);
        if (buffered == output_buffer.size())
          flush();
      }
    };
    auto result = format == "json" ? reader.export_json(sink)
                                   : reader.export_ndjson(sink);
    std::string suffix;
    if (format == "json") {
      sink("\n");
      suffix = "}\n";
    } else
      suffix = last;
    flush();
    const auto file_sha = digest_state.finish();
    recovery["expected_binary"] = {{"reference", ref_json(reader.reference())},
                                   {"format", format},
                                   {"bytes", dec(bytes)},
                                   {"file_sha256", file_sha},
                                   {"completion_suffix", suffix}};
    need(::fsync(file.n) == 0, "export stage synchronization failed");
    check(checkpoint);
    recovery["phase"] = "publish_export";
    if (::linkat(dir.n, stage.c_str(), dir.n, leaf.c_str(), 0) != 0)
      fail("bundle.exists",
           "export publication refused; destination may exist");
    recovery["published"] = true;
    recovery["phase"] = "sync_export";
    if (::unlinkat(dir.n, stage.c_str(), 0) == 0)
      recovery["stage_path"] = nullptr;
    if (::fsync(dir.n) != 0)
      fail("bundle.outcome_uncertain",
           "export published but directory durability uncertain");
    recovery["durable"] = true;
    result["output_path"] = output_path;
    result["format"] = format;
    result["bytes"] = dec(bytes);
    result["file_sha256"] = file_sha;
    result["completion_suffix"] = suffix;
    return result;
  } catch (StoreError &e) {
    e.recovery = recovery;
    throw;
  } catch (const std::exception &e) {
    throw StoreError("bundle.interrupted", e.what(), recovery);
  }
}
} // namespace symphony::sbv::result_store
