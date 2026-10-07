#include "partitioned_output.hpp"
#include "detail.hpp"
#include <cerrno>
#include <fcntl.h>
#include <limits>
#include <new>
#include <symphony/knowledge/engine/limits.hpp>
#include <symphony/knowledge/engine/path.hpp>
#include <sys/stat.h>
#include <unistd.h>
#include <utility>
#include <vector>

namespace symphony::sbv::detail {
namespace {
namespace store = result_store;
constexpr std::size_t resident_bytes = 1U << 20;
constexpr std::size_t envelope_bytes = 2048, envelope_values = 64;
constexpr std::size_t private_name_bytes = 64;
constexpr std::size_t maximum_page_leaf_bytes = 69; // hex SHA256 + ".json"
struct OutputFd {
  int value = -1;
  explicit OutputFd(int fd = -1) : value(fd) {}
  ~OutputFd() {
    if (value >= 0)
      ::close(value);
  }
  OutputFd(const OutputFd &) = delete;
  OutputFd &operator=(const OutputFd &) = delete;
  OutputFd(OutputFd &&other) noexcept : value(std::exchange(other.value, -1)) {}
  OutputFd &operator=(OutputFd &&other) noexcept {
    if (value >= 0)
      ::close(value);
    value = std::exchange(other.value, -1);
    return *this;
  }
};
struct Parent {
  OutputFd directory;
  std::string leaf;
  struct stat identity{};
};
bool same(const struct stat &a, const struct stat &b) {
  return a.st_dev == b.st_dev && a.st_ino == b.st_ino;
}
struct stat identity(int fd) {
  struct stat st{};
  need(::fstat(fd, &st) == 0 && S_ISDIR(st.st_mode),
       "partitioned directory identity unavailable");
  return st;
}
std::string safe_path(const Json &value) {
  const auto path = str(value);
  need(path.size() > 1 && path.size() <= 4096 && path.front() == '/' &&
           path.back() != '/' && e::is_safe_relative_path(path.substr(1)),
       "partitioned path must be absolute, safe and within4096 bytes");
  return path;
}
Parent parent(const std::string &path,
              const std::optional<struct stat> &outside = {}) {
  (void)safe_path(path);
  OutputFd directory(
      ::open("/", O_RDONLY | O_DIRECTORY | O_CLOEXEC | O_NOFOLLOW));
  need(directory.value >= 0, "partitioned filesystem root unavailable");
  auto check = [&] {
    const auto actual = identity(directory.value);
    if (outside)
      need(!same(actual, *outside),
           "final bundle must be outside private workspace");
  };
  check();
  std::size_t at = 1;
  for (;;) {
    const auto slash = path.find('/', at);
    if (slash == std::string::npos) {
      const auto actual = identity(directory.value);
      return {std::move(directory), path.substr(at), actual};
    }
    const auto component = path.substr(at, slash - at);
    OutputFd next(::openat(directory.value, component.c_str(),
                           O_RDONLY | O_DIRECTORY | O_CLOEXEC | O_NOFOLLOW));
    need(next.value >= 0, "partitioned parent must exist without symlinks");
    directory = std::move(next);
    check();
    at = slash + 1;
  }
}
void absent(const Parent &p) {
  struct stat st{};
  errno = 0;
  need(::fstatat(p.directory.value, p.leaf.c_str(), &st, AT_SYMLINK_NOFOLLOW) !=
               0 &&
           errno == ENOENT,
       "partitioned destination must not already exist");
}
bool ancestor(const std::string &a, const std::string &b) {
  return b.size() > a.size() && b.compare(0, a.size(), a) == 0 &&
         b[a.size()] == '/';
}
void portable_control(const Json &value, std::size_t &nodes,
                      std::size_t depth) {
  need(depth <= e::Limits::max_json_depth,
       "partitioned receipt depth exceeds control framing");
  need(!value.is_number(), "partitioned receipt uses exact numeric strings");
  need(nodes < e::Limits::max_json_values,
       "partitioned receipt values exceed control framing");
  ++nodes;
  if (value.is_string())
    need(value.get_ref<const std::string &>().size() <=
             e::Limits::max_string_bytes,
         "partitioned receipt string exceeds control framing");
  if (value.is_object()) {
    need(value.size() <= e::Limits::max_json_values - nodes,
         "partitioned receipt keys exceed control framing");
    nodes += value.size();
    for (auto it = value.begin(); it != value.end(); ++it) {
      need(it.key().size() <= e::Limits::max_string_bytes,
           "partitioned receipt key exceeds control framing");
      portable_control(it.value(), nodes, depth + 1);
    }
  } else if (value.is_array())
    for (const auto &child : value)
      portable_control(child, nodes, depth + 1);
}
void fits_control(const Json &receipt) {
  std::size_t nodes = 0;
  // Four outer levels reserve ordinary process and resident parent framing.
  portable_control(receipt, nodes, 4);
  need(nodes <= e::Limits::max_json_values - envelope_values,
       "partitioned receipt values leave no control envelope reserve");
  need(receipt.dump().size() <= resident_bytes - envelope_bytes,
       "partitioned receipt bytes leave no resident envelope reserve");
}
Json reference_json(const store::Reference &r) {
  return {{"manifest_path", r.manifest_path},
          {"manifest_sha256", r.manifest_sha256},
          {"content_sha256", r.content_sha256}};
}
bool symbolic_code(const std::string &code) {
  return !code.empty() && code.size() <= 128 && code.front() >= 'a' &&
         code.front() <= 'z' &&
         code.find_first_not_of("abcdefghijklmnopqrstuvwxyz0123456789_.-") ==
             std::string::npos;
}
Json native_cause(const std::string &code, bool storage_error) {
  if (!symbolic_code(code))
    return {{"category", "unexpected"},
            {"code", "sbv.invalid_native_error_code"}};
  std::string category;
  if (code == "deadline.exceeded" || code == "request.deadline_expired")
    category = "deadline";
  else if (code == "sbv.allocation_failed")
    category = "allocation";
  else if (code == "bundle.interrupted" || code == "sbv.unexpected_exception" ||
           code == "sbv.unknown_exception" ||
           code == "sbv.invalid_native_error_code")
    category = "unexpected";
  else if (code.starts_with("bundle."))
    category = "storage";
  else if (code == "sbv.contract" || code == "sbv.length_error" ||
           code.starts_with("logical.") || !storage_error)
    category = "contract";
  else
    category = "unexpected";
  return {{"category", category}, {"code", code}};
}
Json current_cause() {
  // Called only from an active exception handler. Preserve symbolic native
  // codes, never arbitrary what() text or filesystem/provider error contents.
  try {
    throw;
  } catch (const e::Error &error) {
    return native_cause(error.code(), false);
  } catch (const store::StoreError &error) {
    return native_cause(error.code, true);
  } catch (const std::bad_alloc &) {
    return {{"category", "allocation"}, {"code", "sbv.allocation_failed"}};
  } catch (const std::length_error &) {
    return {{"category", "contract"}, {"code", "sbv.length_error"}};
  } catch (const std::exception &) {
    return {{"category", "unexpected"}, {"code", "sbv.unexpected_exception"}};
  } catch (...) {
    return {{"category", "unexpected"}, {"code", "sbv.unknown_exception"}};
  }
}
Json successful_receipt(const std::string &protocol, const std::string &status,
                        const Json &output, const Json &summary,
                        const store::Receipt *receipt) {
  const auto maximum =
      std::to_string(std::numeric_limits<std::uint64_t>::max());
  const auto digest =
      receipt ? receipt->reference.content_sha256 : std::string(64, 'f');
  const Json reference =
      receipt ? reference_json(receipt->reference)
              : Json{{"manifest_path",
                      str(output.at("bundle_path")) + "/manifest.json"},
                     {"manifest_sha256", std::string(64, 'f')},
                     {"content_sha256", digest}};
  Json storage{{"kind", "partitioned"},
               {"reference", reference},
               {"workspace_path", output.at("workspace_path")},
               {"workspace_retained", true},
               {"write_options", output.at("write_options")}};
  const Json actual = receipt ? receipt->json() : Json::object();
  for (const auto *field :
       {"manifest_bytes", "logical_body_bytes", "logical_body_nodes",
        "logical_body_values", "exported_value_nodes", "page_files_created",
        "page_bytes_created"})
    storage[field] = receipt ? actual.at(field) : Json(maximum);
  return {{"protocol", protocol},
          {"status", status},
          {"content_sha256", digest},
          {"storage", std::move(storage)},
          {"summary", summary}};
}
} // namespace

struct PartitionedOutput::Impl {
  Json selection;
  std::string workspace, bundle;
  store::WriteOptions write;
  store::Checkpoint check;
  Parent workspace_parent, bundle_parent;
  OutputFd workspace_directory;
  std::optional<struct stat> workspace_identity;
  bool creation_attempted = false, created = false, durable = false;
  std::string phase = "create_workspace";
  Json final_storage = nullptr;
  std::unique_ptr<store::ResultWriter> writer;

  Impl(const Json &request, std::int64_t end)
      : selection(request.at("output")), check([end] { deadline(end); }) {
    need(!request.contains("output_path"),
         "select only one result output form");
    keys(selection, {"kind", "bundle_path", "workspace_path", "write_options"});
    need(selection.at("kind") == "partitioned",
         "partitioned output kind required");
    bundle = safe_path(selection.at("bundle_path"));
    workspace = safe_path(selection.at("workspace_path"));
    need(bundle != workspace && !ancestor(bundle, workspace) &&
             !ancestor(workspace, bundle),
         "workspace and final bundle must be disjoint");
    keys(selection.at("write_options"), {"page_bytes", "index_fanout"});
    write = {u64(selection.at("write_options").at("page_bytes")),
             u64(selection.at("write_options").at("index_fanout"))};
    need(write.page_bytes >= 512 && write.index_fanout >= 2 &&
             write.page_bytes <= std::numeric_limits<std::size_t>::max(),
         "invalid selected partitioned page layout");
    (void)safe_path(bundle + "/manifest.json");
    (void)safe_path(bundle + "/" + std::string(maximum_page_leaf_bytes, 'p'));
    (void)safe_path(workspace + "/" + std::string(private_name_bytes, 's') +
                    "/" + std::string(maximum_page_leaf_bytes, 'p'));
    check();
    workspace_parent = parent(workspace);
    bundle_parent = parent(bundle);
    absent(workspace_parent);
    absent(bundle_parent);
  }
  void create() {
    check();
    creation_attempted = true;
    need(::mkdirat(workspace_parent.directory.value,
                   workspace_parent.leaf.c_str(), 0700) == 0,
         "exclusive private workspace creation failed");
    created = true;
    workspace_directory = OutputFd(::openat(
        workspace_parent.directory.value, workspace_parent.leaf.c_str(),
        O_RDONLY | O_DIRECTORY | O_CLOEXEC | O_NOFOLLOW));
    need(workspace_directory.value >= 0,
         "new private workspace cannot be opened");
    workspace_identity = identity(workspace_directory.value);
    need(::fsync(workspace_directory.value) == 0 &&
             ::fsync(workspace_parent.directory.value) == 0,
         "private workspace creation synchronization failed");
    durable = true;
    // Nonexistent case/Unicode aliases cannot all be resolved read-only.
    // Resolve the now-created directory before any producer callback.
    auto actual_final = parent(bundle, workspace_identity);
    need(same(actual_final.identity, bundle_parent.identity),
         "final parent identity changed during workspace creation");
    absent(actual_final);
    verify_workspace();
    check();
    phase = "produce";
  }
  void verify_workspace() const {
    need(workspace_identity.has_value(), "private workspace identity absent");
    auto current = parent(workspace);
    need(same(current.identity, workspace_parent.identity),
         "private workspace parent changed");
    OutputFd reopened(
        ::openat(current.directory.value, current.leaf.c_str(),
                 O_RDONLY | O_DIRECTORY | O_CLOEXEC | O_NOFOLLOW));
    need(reopened.value >= 0 &&
             same(identity(reopened.value), *workspace_identity),
         "private workspace path no longer names owned directory");
  }
  Json recovery() const {
    return {
        {"output", selection},          {"phase", phase},
        {"workspace_created", created}, {"workspace_creation_durable", durable},
        {"automatic_cleanup", false},   {"final_storage", final_storage}};
  }
};
PartitionedOutput::PartitionedOutput(const Json &p, std::int64_t end)
    : p_(std::make_unique<Impl>(p, end)) {}
PartitionedOutput::~PartitionedOutput() = default;
const Json &PartitionedOutput::output() const { return p_->selection; }
const std::string &PartitionedOutput::workspace_path() const {
  return p_->workspace;
}
const std::string &PartitionedOutput::bundle_path() const { return p_->bundle; }
store::WriteOptions PartitionedOutput::write_options() const {
  return p_->write;
}
store::ReadOptions PartitionedOutput::read_options() const {
  return {std::nullopt, 0};
}
store::Checkpoint PartitionedOutput::checkpoint() const {
  return [check = p_->check] {
    try {
      check();
    } catch (const e::Error &error) {
      // Preserve deadline identity through writer guards, while direct
      // pre-workspace checks retain the normal native exception boundary.
      throw store::StoreError(error.code(),
                              "selected operation deadline exceeded");
    }
  };
}
std::unique_ptr<store::RowSpool>
PartitionedOutput::spool(std::string_view name) {
  need(p_->phase == "produce", "row spools belong to the producing phase");
  need(!name.empty() && name.size() <= private_name_bytes &&
           name.front() >= 'a' && name.front() <= 'z' &&
           name.find_first_not_of("abcdefghijklmnopqrstuvwxyz0123456789_-") ==
               std::string_view::npos,
       "invalid private spool name");
  p_->check();
  p_->verify_workspace();
  const auto path = p_->workspace + "/" + std::string(name);
  (void)safe_path(path + "/manifest.json");
  return std::make_unique<store::RowSpool>(path, p_->write, read_options(),
                                           checkpoint());
}

Json partitioned_result(const Json &request, const std::string &slug,
                        std::int64_t end, const PartitionedBody &produce) {
  need(static_cast<bool>(produce), "partitioned producer callback required");
  need(!slug.empty() && slug.size() <= 64 &&
           slug.find_first_not_of("abcdefghijklmnopqrstuvwxyz-") ==
               std::string::npos,
       "invalid existing operation result slug");
  const auto protocol = "symphony.sbv." + slug + ".v1";
  PartitionedOutput context(request, end); // read-only/static preflight
  try {
    context.p_->create();
    auto body = produce(context);
    context.p_->phase = "finalize_result";
    context.p_->check();
    need(body.kind() == "object" && body.size() == 4 &&
             body.contains("origin") && body.contains("protocol") &&
             body.contains("sections") && body.contains("status"),
         "partitioned producer must return four-field result body");
    const auto status =
        body.at("status").materialize({64, 1}, context.checkpoint());
    need(status == "partial" || status == "completed",
         "invalid logical result status");
    need(body.at("protocol").materialize({128, 1}, context.checkpoint()) ==
             result_protocol,
         "invalid logical result protocol");
    const auto summary =
        body.at("sections")
            .at("summary")
            .materialize({resident_bytes - envelope_bytes,
                          e::Limits::max_json_values - envelope_values},
                         context.checkpoint());
    keys(summary, {"status", "reason", "data"});
    need(summary.at("status") == "available" &&
             summary.at("reason").is_string() && summary.at("data").is_object(),
         "bounded available summary object required");
    // Maximum-width counters/digests reserve the complete final receipt before
    // final-directory creation. Summary never shrinks or truncates silently.
    fits_control(successful_receipt(protocol, str(status), context.output(),
                                    summary, nullptr));
    context.p_->verify_workspace();
    auto actual_final =
        parent(context.bundle_path(), context.p_->workspace_identity);
    need(same(actual_final.identity, context.p_->bundle_parent.identity),
         "final parent identity changed before publication");
    absent(actual_final);
    context.p_->check();
    try {
      context.p_->writer = std::make_unique<store::ResultWriter>(
          context.bundle_path(), context.write_options(), context.checkpoint());
    } catch (const store::StoreError &error) {
      if (!error.recovery.is_null())
        context.p_->final_storage = error.recovery;
      throw;
    }
    try {
      body.write(*context.p_->writer);
      const auto receipt = context.p_->writer->finish();
      context.p_->final_storage = context.p_->writer->recovery();
      auto result = successful_receipt(protocol, str(status), context.output(),
                                       summary, &receipt);
      fits_control(result);
      return result;
    } catch (...) {
      context.p_->final_storage = context.p_->writer->recovery();
      throw;
    }
  } catch (...) {
    if (!context.p_->creation_attempted)
      throw;
    const auto cause = current_cause();
    Json result{{"protocol", protocol},
                {"status", "recovery_required"},
                {"code", "sbv.partitioned_result_incomplete"},
                {"cause", cause},
                {"request_sha256", e::sha256_hex(request.dump())},
                {"automatic_retry", false},
                {"rollback_performed", false},
                {"recovery", context.p_->recovery()}};
    fits_control(result);
    return result;
  }
}
} // namespace symphony::sbv::detail
