#pragma once
#include "native_test.hpp"
#include <dlfcn.h>
#include <iomanip>
#include <optional>
#include <sys/file.h>

namespace scv_test {
namespace nt = native_test;
namespace fs = std::filesystem;
using J = nt::Json;
inline J object() { return J::object(); }
inline J array() { return J::array(); }
inline std::string digest(const J &value) {
  return nt::digest(nt::canonical(value));
}
inline J pick(const J &value, std::initializer_list<const char *> keys) {
  J result = object();
  for (auto key : keys)
    result[key] = value.at(key);
  return result;
}
inline void private_dir(const fs::path &path) {
  fs::create_directories(path);
  NT_REQUIRE(::chmod(path.c_str(), 0700) == 0);
}
inline J installation(const std::string &module, const std::string &role,
                      const std::string &version,
                      const std::string &prefix = "/fixture/installed") {
  auto engine = "symphony-" + (module.ends_with("-engine") ? role : module);
  return {{"Role", role},
          {"ModuleID", module},
          {"EngineID", engine},
          {"Version", version},
          {"Prefix", prefix},
          {"ReceiptPath", prefix + "/share/symphony/receipts/" + module + "/" +
                              version + "/install-receipt.json"},
          {"ReceiptDigest", "sha256:" + std::string(64, '1')},
          {"ReceiptProtocol", "symphony.knowledge.install-receipt.v2"},
          {"ExecutablePath", prefix + "/libexec/symphony/" + module + "/" +
                                 version + "/" + engine},
          {"ExecutableDigest", "sha256:" + std::string(64, '2')}};
}
// Test-only selected DuckDB C ABI. This deliberately bypasses the connector to
// inject corruption and interrupt uncommitted transactions for rejection tests.
class SQL {
  void *library_ = nullptr;
  void *db_ = nullptr;
  void *connection_ = nullptr;
  using Open = int (*)(const char *, void **);
  using Connect = int (*)(void *, void **);
  using Query = int (*)(void *, const char *, void *);
  using Release = void (*)(void **);
  Query query_ = nullptr;
  Release disconnect_ = nullptr, close_ = nullptr;
  template <class T> T symbol(const char *name) {
    auto p = ::dlsym(library_, name);
    nt::require(p != nullptr, std::string("Missing DuckDB symbol ") + name);
    return reinterpret_cast<T>(p);
  }

public:
  SQL(const fs::path &path, const std::string &library) {
    library_ = ::dlopen(library.c_str(), RTLD_NOW | RTLD_LOCAL);
    nt::require(library_ != nullptr, "Cannot load selected DuckDB library");
    auto open = symbol<Open>("duckdb_open");
    auto connect = symbol<Connect>("duckdb_connect");
    query_ = symbol<Query>("duckdb_query");
    disconnect_ = symbol<Release>("duckdb_disconnect");
    close_ = symbol<Release>("duckdb_close");
    NT_REQUIRE(open(path.c_str(), &db_) == 0);
    NT_REQUIRE(connect(db_, &connection_) == 0);
  }
  void query(const std::string &sql) {
    nt::require(query_(connection_, sql.c_str(), nullptr) == 0,
                "DuckDB query failed: " + sql);
  }
  void close() {
    if (connection_)
      disconnect_(&connection_);
    if (db_)
      close_(&db_);
    if (library_) {
      ::dlclose(library_);
      library_ = nullptr;
    }
  }
  ~SQL() { close(); }
  SQL(const SQL &) = delete;
  SQL &operator=(const SQL &) = delete;
};
// Native asynchronous child needed by the actual SIGKILL tests. Signal exits
// retain Python subprocess's negative-signal convention in the evidence format.
class AsyncProcess {
  nt::TempDir files_{"scv-native-child-"};
  pid_t child_ = -1;
  std::optional<int> result_;

public:
  AsyncProcess(const std::vector<std::string> &command,
               const std::string &input = "",
               const std::map<std::string, std::string> &env = {},
               const fs::path &cwd = {}) {
    NT_REQUIRE(!command.empty());
    nt::write(files_.path / "stdin", input);
    child_ = ::fork();
    NT_REQUIRE(child_ >= 0);
    if (child_ == 0) {
      ::setpgid(0, 0);
      int in = ::open((files_.path / "stdin").c_str(), O_RDONLY);
      int out = ::open((files_.path / "stdout").c_str(),
                       O_WRONLY | O_CREAT | O_TRUNC, 0600);
      int err = ::open((files_.path / "stderr").c_str(),
                       O_WRONLY | O_CREAT | O_TRUNC, 0600);
      if (in < 0 || out < 0 || err < 0)
        _exit(126);
      ::dup2(in, 0);
      ::dup2(out, 1);
      ::dup2(err, 2);
      ::close(in);
      ::close(out);
      ::close(err);
      if (!cwd.empty() && ::chdir(cwd.c_str()) != 0)
        _exit(126);
      for (const auto &[key, value] : env)
        ::setenv(key.c_str(), value.c_str(), 1);
      std::vector<char *> argv;
      for (const auto &value : command)
        argv.push_back(const_cast<char *>(value.c_str()));
      argv.push_back(nullptr);
      ::execvp(argv[0], argv.data());
      _exit(127);
    }
  }
  std::optional<int> poll() {
    if (result_)
      return result_;
    int status = 0;
    auto got = ::waitpid(child_, &status, WNOHANG);
    if (got == 0)
      return {};
    if (got < 0 && errno == EINTR)
      return {};
    NT_REQUIRE(got == child_);
    result_ = WIFEXITED(status) ? WEXITSTATUS(status) : -WTERMSIG(status);
    return result_;
  }
  pid_t pid() const { return child_; }
  void wait_stopped(int seconds, int expected_signal = SIGSTOP) {
    const auto deadline =
        std::chrono::steady_clock::now() + std::chrono::seconds(seconds);
    for (;;) {
      int state = 0;
      auto observed = ::waitpid(child_, &state, WUNTRACED | WNOHANG);
      if (observed == child_) {
        if (WIFSTOPPED(state)) {
          NT_REQUIRE(WSTOPSIG(state) == expected_signal);
          return;
        }
        result_ = WIFEXITED(state) ? WEXITSTATUS(state) : -WTERMSIG(state);
        throw std::runtime_error(
            "fault process exited before barrier: " + out() + err());
      }
      NT_REQUIRE(observed == 0 || (observed < 0 && errno == EINTR));
      nt::require(std::chrono::steady_clock::now() < deadline,
                  "fault process did not stop");
      std::this_thread::sleep_for(std::chrono::milliseconds(1));
    }
  }
  void kill(int signal = SIGKILL) {
    if (!poll()) {
      ::kill(-child_, signal);
      ::kill(child_, signal);
    }
  }
  int wait(int seconds = 5) {
    auto until =
        std::chrono::steady_clock::now() + std::chrono::seconds(seconds);
    while (!poll()) {
      nt::require(std::chrono::steady_clock::now() < until,
                  "child wait timeout");
      std::this_thread::sleep_for(std::chrono::milliseconds(2));
    }
    return *result_;
  }
  std::string out() const { return nt::read(files_.path / "stdout"); }
  std::string err() const { return nt::read(files_.path / "stderr"); }
  ~AsyncProcess() {
    if (!result_) {
      ::kill(-child_, SIGKILL);
      ::kill(child_, SIGKILL);
      int ignored;
      while (::waitpid(child_, &ignored, 0) < 0 && errno == EINTR) {
      }
    }
  }
};
struct Config {
  std::string engine, library, version = "0.1.0-dev", self;
  fs::path evidence;
  std::size_t calls = 0;
};
inline std::pair<nt::ProcessResult, J> raw(Config &c, const fs::path &root,
                                           const std::string &operation,
                                           const J &payload, int timeout = 5) {
  auto request = nt::request("symphony-scv-graph-duckdb-connector", operation,
                             payload, "native-" + std::to_string(++c.calls));
  request["deadline_unix_ms"] = nt::now_ms() + timeout * 1000;
  auto process =
      nt::run({c.engine}, nt::canonical(request), timeout + 3, {}, root);
  auto result = J::parse(process.stdout_text);
  if (!c.evidence.empty()) {
    std::ostringstream key;
    key << std::setw(3) << std::setfill('0') << c.calls << "-" << operation;
    nt::write(c.evidence / (key.str() + "-request.json"),
              nt::canonical(request) + "\n");
    nt::write(c.evidence / (key.str() + "-response.json"), process.stdout_text);
  }
  return {process, result};
}
struct ConnectorTests {
  Config &config;
  nt::TempDir temporary;
  fs::path root;
  J graph, base;
  explicit ConnectorTests(Config &c) : config(c), root(temporary.path) {
    private_dir(root);
    graph =
        nt::read_json(fs::path(__FILE__).parent_path() / "fixtures/graph.json");
    base = {{"tops_id", "01993d63-b40d-7000-8000-000000000013"},
            {"namespace", "native-test"},
            {"operation_id", "prepare-1"},
            {"graph", graph},
            {"owner", installation("scv-engine", "scv", "0.10.0-dev")},
            {"connector",
             installation("scv-graph-duckdb-connector",
                          "scv-graph-duckdb-connector", config.version)},
            {"query_time", "2026-09-13T04:30:18Z"}};
  }
  J call(const std::string &operation, const J &payload, bool okay = true,
         int timeout = 5) {
    auto [p, r] = raw(config, root, operation, payload, timeout);
    nt::require((r.at("outcome") == "ok") == okay, r.dump());
    nt::require((p.returncode == 0) == okay, r.dump());
    if (okay) {
      auto v = r.at("result");
      NT_REQUIRE(v == nt::seal(v, operation == "inspect" ? "descriptor_digest"
                                                         : "digest"));
      return v;
    }
    NT_REQUIRE(r.at("result").is_null());
    NT_REQUIRE(!r.at("error").at("code").get<std::string>().empty());
    return r;
  }
  J status_input(const J &other = nullptr) const {
    return pick(other.is_null() ? base : other,
                {"tops_id", "namespace", "operation_id"});
  }
  J prepare() { return call("prepare", base); }
  J commit(const J &prepared = nullptr) {
    auto p = prepared.is_null() ? prepare() : prepared;
    return call("commit",
                nt::updated(status_input(), {{"expected_intent_digest",
                                              p["intent"]["digest"]}}));
  }
  J query_input(const J &done, const std::string &kind = "claims",
                const J &filters = object(), int limit = 128,
                const J &cursor = nullptr) const {
    return nt::updated(pick(base, {"tops_id", "namespace"}),
                       {{"snapshot_digest", done.at("snapshot_digest")},
                        {"kind", kind},
                        {"filters", filters},
                        {"limit", limit},
                        {"cursor", cursor}});
  }
  void test_descriptor_does_not_open_database() {
    auto d = call("inspect", object());
    NT_REQUIRE(d["operations"].size() ==
               (config.version == "0.1.0-dev" ? 6U : 8U));
    NT_REQUIRE(fs::is_empty(root));
    auto op = std::find_if(
        d["operations"].begin(), d["operations"].end(),
        [](const J &v) { return v["operation_name"] == "commit"; });
    NT_REQUIRE(op != d["operations"].end());
    NT_REQUIRE((*op)["expected_state_required"] == true);
    NT_REQUIRE((*op)["administrative_interactions"] ==
               J::array({"invoke", "recover"}));
  }
  void test_prepare_reopen_commit_idempotence() {
    auto p = prepare();
    NT_REQUIRE(p["state"] == "prepared");
    NT_REQUIRE(p["index_verified"] == false);
    NT_REQUIRE(call("status", status_input()) == p);
    NT_REQUIRE(prepare() == p);
    call("export",
         pick(query_input(p), {"tops_id", "namespace", "snapshot_digest"}),
         false);
    auto done = commit(p);
    NT_REQUIRE(done["state"] == "committed");
    NT_REQUIRE(done["index_verified"] == true);
    NT_REQUIRE(commit(p) == done);
    NT_REQUIRE(prepare() == done);
    NT_REQUIRE(call("status", status_input()) == done);
    for (const auto &path : fs::directory_iterator(root)) {
      struct stat info{};
      NT_REQUIRE(::stat(path.path().c_str(), &info) == 0);
      NT_REQUIRE((info.st_mode & 0777) == 0600);
    }
  }
  void test_operation_collision_and_expected_digest_reject() {
    auto p = prepare();
    call("prepare", nt::updated(base, {{"query_time", "2026-09-13T04:30:19Z"}}),
         false);
    call("commit",
         nt::updated(status_input(), {{"expected_intent_digest",
                                       "sha256:" + std::string(64, '0')}}),
         false);
    NT_REQUIRE(call("status", status_input()) == p);
  }
  void test_namespaces_and_shared_snapshot_operations() {
    auto done = commit();
    auto other = nt::updated(base, {{"operation_id", "other"}});
    auto p = call("prepare", other);
    NT_REQUIRE(p["snapshot_digest"] == done["snapshot_digest"]);
    p = call("prepare", nt::updated(base, {{"namespace", "other"}}));
    NT_REQUIRE(p["snapshot_digest"] != done["snapshot_digest"]);
    call("query", nt::updated(query_input(done), {{"namespace", "other"}}),
         false);
  }
  void test_exact_filters_pagination_and_cursor_binding() {
    auto done = commit();
    auto rows = call("query", query_input(done, "nodes"));
    NT_REQUIRE(rows["matched_count"] == graph["native_nodes"].size());
    J got = array(), cursor = nullptr;
    for (;;) {
      auto page =
          call("query", query_input(done, "nodes", object(), 13, cursor));
      for (auto row : page["rows"])
        got.push_back(row);
      cursor = page["next_cursor"];
      if (cursor.is_null())
        break;
    }
    NT_REQUIRE(got == rows["rows"]);
    auto first = call("query", query_input(done, "nodes", object(), 1));
    auto bad = first["next_cursor"];
    bad["query_digest"] = "sha256:" + std::string(64, 'f');
    call("query", query_input(done, "nodes", object(), 1, bad), false);
    bad = first["next_cursor"];
    bad["after_key"] = "missing";
    call("query", query_input(done, "nodes", object(), 128, bad), false);
    auto node = graph["native_nodes"][0];
    auto filtered =
        call("query",
             query_input(done, "nodes",
                         pick(node, {"node_id", "kind", "capture_digest"})));
    NT_REQUIRE(filtered["rows"] ==
               J::array({{{"key", node["node_id"]}, {"value", node}}}));
  }
  void test_claim_scope_and_structural_edges() {
    auto done = commit();
    for (const auto &claim : graph["claims"]) {
      auto r = call("query", query_input(done, "claims",
                                         pick(claim, {"claim_id", "subject",
                                                      "predicate", "scope"})));
      NT_REQUIRE(r["rows"] ==
                 J::array({{{"key", claim["claim_id"]}, {"value", claim}}}));
    }
    auto edge = graph["native_edges"][0];
    auto r = call("query", query_input(done, "edges", edge));
    NT_REQUIRE(r["rows"] ==
               J::array({{{"key", digest(edge)}, {"value", edge}}}));
  }
  void test_opaque_control_ids_and_empty_scope_are_preserved() {
    graph["claims"][0]["claim_id"] = "opaque\nclaim";
    graph["claims"][0]["scope"] = {{"empty", ""},
                                   {"nul", std::string(1, '\0')}};
    base["graph"] = nt::seal(graph);
    auto done = commit(), claim = base["graph"]["claims"][0];
    auto result =
        call("query",
             query_input(done, "claims", pick(claim, {"claim_id", "scope"})));
    NT_REQUIRE(result["rows"] ==
               J::array({{{"key", claim["claim_id"]}, {"value", claim}}}));
    NT_REQUIRE(
        call("query",
             query_input(done, "claims",
                         {{"subject",
                           std::string(65536, 'x')}}))["matched_count"] == 0);
  }
  void corrupt(const std::vector<std::string> &sql) {
    SQL db(root / "index.duckdb", config.library);
    for (const auto &s : sql)
      db.query(s);
    db.close();
  }
  void test_reject_same_named_index_with_wrong_expression() {
    commit();
    corrupt(
        {"DROP INDEX nodes_kind", "CREATE INDEX nodes_kind ON nodes(node_id)"});
    call("status", status_input(), false);
  }
  void test_reject_index_column_and_inventory_corruption() {
    commit();
    corrupt({"UPDATE nodes SET kind='tampered' WHERE row_key=(SELECT "
             "min(row_key) FROM nodes)"});
    call("status", status_input(), false);
  }
  void test_reject_embedded_nul_stored_bytes() {
    commit();
    corrupt({"UPDATE nodes SET value=value || chr(0) || 'garbage' WHERE "
             "row_key=(SELECT min(row_key) FROM nodes)"});
    call("status", status_input(), false);
  }
  void test_reject_deleted_projection_row() {
    auto done = commit();
    corrupt(
        {"DELETE FROM edges WHERE row_key=(SELECT min(row_key) FROM edges)"});
    call("export",
         pick(query_input(done), {"tops_id", "namespace", "snapshot_digest"}),
         false);
  }
  void test_reject_sealed_intent_database_substitution() {
    prepare();
    corrupt({"UPDATE intents SET intent_digest='sha256:" +
             std::string(64, '0') + "'"});
    call("status", status_input(), false);
  }
  void test_reject_schema_changes() {
    prepare();
    corrupt({"CREATE TABLE unrelated (x VARCHAR)"});
    call("status", status_input(), false);
  }
  void test_owned_private_paths_and_symlinks() {
    NT_REQUIRE(::chmod(root.c_str(), 0755) == 0);
    call("prepare", base, false);
    NT_REQUIRE(::chmod(root.c_str(), 0700) == 0);
    auto target = root / "target";
    nt::write(target, "unchanged");
    NT_REQUIRE(::chmod(target.c_str(), 0600) == 0);
    fs::create_symlink(target, root / "index.duckdb");
    call("prepare", base, false);
    NT_REQUIRE(nt::read(target) == "unchanged");
  }
  void test_lock_deadline_and_no_missing_read_creation() {
    call("status", status_input(), false);
    NT_REQUIRE(fs::is_empty(root));
    prepare();
    int fd = ::open((root / "connector.lock").c_str(), O_RDWR);
    NT_REQUIRE(fd >= 0);
    NT_REQUIRE(::flock(fd, LOCK_EX) == 0);
    auto start = std::chrono::steady_clock::now();
    try {
      call("status", status_input(), false, 1);
      NT_REQUIRE(std::chrono::duration<double>(
                     std::chrono::steady_clock::now() - start)
                     .count() < 2.5);
    } catch (...) {
      ::close(fd);
      throw;
    }
    ::close(fd);
  }
  void test_shape_identity_and_calendar_boundaries() {
    for (const auto &patch :
         J::array({{{"query_time", "2026-02-30T00:00:00Z"}},
                   {{"operation_id", "spaces forbidden"}},
                   {{"tops_id", "01993d63-b40d-0000-8000-000000000013"}},
                   {{"namespace", "../escape"}}}))
      call("prepare", nt::updated(base, patch), false);
    auto altered = base;
    altered["connector"]["Role"] = "scv";
    call("prepare", altered, false);
    auto g = graph;
    g["native_nodes"].push_back(g["native_nodes"][0]);
    call("prepare", nt::updated(base, {{"graph", nt::seal(g)}}), false);
    altered = base;
    altered["tops_id"] = "01993d63-b40d-1000-8000-000000000013";
    call("prepare", altered);
  }
  void
  test_abrupt_duckdb_transaction_rollback_preserves_committed_projection() {
    auto done = commit();
    auto ready = root / "writer-ready";
    AsyncProcess child({config.self, "--crash-writer",
                        (root / "index.duckdb").string(), "--library",
                        config.library, "--ready", ready.string()});
    for (int i = 0; i < 300 && !fs::exists(ready); ++i) {
      NT_REQUIRE(!child.poll());
      std::this_thread::sleep_for(std::chrono::milliseconds(10));
    }
    NT_REQUIRE(fs::exists(ready));
    child.kill();
    NT_REQUIRE(child.wait() == -SIGKILL);
    NT_REQUIRE(call("status", status_input()) == done);
  }
};
inline const std::vector<std::pair<std::string, void (ConnectorTests::*)()>>
    connector_cases = {
#define SCV_CONNECTOR_CASE(name) {#name, &ConnectorTests::name}
        SCV_CONNECTOR_CASE(test_descriptor_does_not_open_database),
        SCV_CONNECTOR_CASE(test_prepare_reopen_commit_idempotence),
        SCV_CONNECTOR_CASE(test_operation_collision_and_expected_digest_reject),
        SCV_CONNECTOR_CASE(test_namespaces_and_shared_snapshot_operations),
        SCV_CONNECTOR_CASE(test_exact_filters_pagination_and_cursor_binding),
        SCV_CONNECTOR_CASE(test_claim_scope_and_structural_edges),
        SCV_CONNECTOR_CASE(
            test_opaque_control_ids_and_empty_scope_are_preserved),
        SCV_CONNECTOR_CASE(test_reject_same_named_index_with_wrong_expression),
        SCV_CONNECTOR_CASE(test_reject_index_column_and_inventory_corruption),
        SCV_CONNECTOR_CASE(test_reject_embedded_nul_stored_bytes),
        SCV_CONNECTOR_CASE(test_reject_deleted_projection_row),
        SCV_CONNECTOR_CASE(test_reject_sealed_intent_database_substitution),
        SCV_CONNECTOR_CASE(test_reject_schema_changes),
        SCV_CONNECTOR_CASE(test_owned_private_paths_and_symlinks),
        SCV_CONNECTOR_CASE(test_lock_deadline_and_no_missing_read_creation),
        SCV_CONNECTOR_CASE(test_shape_identity_and_calendar_boundaries),
        SCV_CONNECTOR_CASE(
            test_abrupt_duckdb_transaction_rollback_preserves_committed_projection)
#undef SCV_CONNECTOR_CASE
};
} // namespace scv_test
