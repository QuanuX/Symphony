#include "native_test.hpp"
#include <dlfcn.h>
using namespace native_test;
namespace {
Json graph() {
  auto a = seal(
      {{"protocol", "caller.example.v1"},
       {"future_field", {{"class", "networking-card"}, {"retired", nullptr}}}});
  Json nodes = Json::array(), edges = Json::array();
  for (int i = 0; i < 3; ++i) {
    nodes.push_back(
        {{"id", std::to_string(i)},
         {"labels", {"component"}},
         {"properties",
          {{"unknown_metric", {{"unit", "caller"}, {"value", i}}}}}});
    edges.push_back({{"id", std::to_string(i)},
                     {"from", "0"},
                     {"to", "1"},
                     {"label", "caller-link"},
                     {"properties", {{"future", {i}}}}});
  }
  return seal({{"protocol", "symphony.graph.exchange.v1"},
               {"owner",
                {{"engine_id", "caller-owner"},
                 {"engine_version", "1"},
                 {"artifact_protocol", a["protocol"]},
                 {"artifact_digest", a["digest"]}}},
               {"owner_artifact", a},
               {"nodes", nodes},
               {"edges", edges}});
}
Json installation(const std::string &version) {
  std::string module = "shv-graph-duckdb-connector", prefix = "/fixture";
  return {{"Role", module},
          {"ModuleID", module},
          {"EngineID", "symphony-" + module},
          {"Version", version},
          {"Prefix", prefix},
          {"ReceiptPath", prefix + "/share/symphony/receipts/" + module + "/" +
                              version + "/install-receipt.json"},
          {"ReceiptProtocol", "symphony.knowledge.install-receipt.v2"},
          {"ReceiptDigest", "sha256:" + std::string(64, '1')},
          {"ExecutablePath", prefix + "/libexec/symphony/" + module + "/" +
                                 version + "/symphony-" + module},
          {"ExecutableDigest", "sha256:" + std::string(64, '2')}};
}
// Corruption oracle calls the selected DuckDB C ABI directly, independently of
// the connector's authorization and graph reconstruction routines.
class SQL {
  void *library = nullptr;
  void *db = nullptr;
  void *connection = nullptr;
  template <class T> T symbol(const char *name) {
    auto value = dlsym(library, name);
    require(value != nullptr, std::string("Missing DuckDB C ABI: ") + name);
    return reinterpret_cast<T>(value);
  }

public:
  SQL(const std::string &selected, const fs::path &path) {
    library = dlopen(selected.c_str(), RTLD_NOW | RTLD_LOCAL);
    require(library != nullptr, "Cannot load selected DuckDB");
    NT_REQUIRE(symbol<int (*)(const char *, void **)>("duckdb_open")(
                   path.c_str(), &db) == 0);
    NT_REQUIRE(symbol<int (*)(void *, void **)>("duckdb_connect")(
                   db, &connection) == 0);
  }
  void query(const std::string &sql) {
    require(symbol<int (*)(void *, const char *, void *)>("duckdb_query")(
                connection, sql.c_str(), nullptr) == 0,
            "DuckDB corruption oracle failed: " + sql);
  }
  ~SQL() {
    if (connection)
      symbol<void (*)(void **)>("duckdb_disconnect")(&connection);
    if (db)
      symbol<void (*)(void **)>("duckdb_close")(&db);
    if (library)
      dlclose(library);
  }
};
struct StoreTests {
  TempDir temp;
  fs::path root;
  Engine &engine;
  std::string library, version;
  Json scope, key, p;
  StoreTests(Engine &e, std::string selected, std::string ver)
      : root(temp.path), engine(e), library(std::move(selected)),
        version(std::move(ver)) {
    NT_REQUIRE(::chmod(root.c_str(), 0700) == 0);
    scope = {{"tops_id", "01993d63-b40d-7000-8000-000000000013"},
             {"namespace", "caller"}};
    key = updated(scope, {{"operation_id", "one"}});
    p = updated(key,
                {{"graph", graph()}, {"connector", installation(version)}});
  }
  Json call(const std::string &op, Json payload, bool ok = true) {
    return engine.call(op, payload, ok, root);
  }
  std::pair<Json, Json> commit() {
    auto prepared = call("prepare", p);
    auto c = updated(
        key, {{"expected_intent_digest", prepared["intent"]["digest"]}});
    return {call("commit", c), c};
  }
  Json inventory(Json extra = Json::object()) {
    return call("inventory",
                updated(updated(scope, {{"expected_revision", nullptr},
                                        {"cursor", nullptr},
                                        {"limit", 1}}),
                        extra));
  }
  void corrupt(const std::string &sql) {
    SQL db(library, root / "index.duckdb");
    db.query(sql);
  }
  void test_descriptor_and_transfer_plan() {
    NT_REQUIRE(call("inspect", Json::object())["operations"].size() == 8);
    auto [stored, unused] = commit();
    auto inv = inventory();
    auto payload =
        updated(scope, {{"expected_revision", inv["manifest"]["digest"]},
                        {"operation_ids", {"one"}},
                        {"source_connector", installation(version)},
                        {"target_connector", installation(version)},
                        {"target_root", "/caller-target"},
                        {"capacity", {{"intents", 1}, {"snapshots", 1}}}});
    auto plan = call("transfer_plan", payload);
    NT_REQUIRE(plan["disposition"] == "ready" &&
               plan["selected"][0]["source"] == stored);
    auto blocked = call(
        "transfer_plan",
        updated(payload, {{"capacity", {{"intents", 0}, {"snapshots", 0}}}}));
    NT_REQUIRE(blocked["blockers"] ==
               Json::array({"intent_capacity", "snapshot_capacity"}));
    for (auto change :
         Json::array({{{"expected_revision", "sha256:" + std::string(64, '0')}},
                      {{"operation_ids", {"absent"}}},
                      {{"operation_ids", {"one", "one"}}},
                      {{"target_root", "/bad/../root"}}}))
      call("transfer_plan", updated(payload, change), false);
    NT_REQUIRE(inventory()["manifest"] == inv["manifest"]);
  }
  void test_reopen_retry_export_and_pagination() {
    auto prepared = call("prepare", p);
    NT_REQUIRE(prepared == call("prepare", p));
    call("export",
         updated(scope, {{"snapshot_digest", prepared["snapshot_digest"]}}),
         false);
    auto [r, c] = commit();
    NT_REQUIRE(r == call("commit", c) && r == call("status", key));
    auto exported = call(
        "export", updated(scope, {{"snapshot_digest", r["snapshot_digest"]}}));
    NT_REQUIRE(exported["snapshot"]["graph"] == p["graph"]);
    auto q = updated(scope, {{"snapshot_digest", r["snapshot_digest"]},
                             {"kind", "edges"},
                             {"filters", Json::object()},
                             {"cursor", nullptr},
                             {"limit", 1}});
    auto one = call("query", q);
    NT_REQUIRE(one["matched_count"] == 3);
    q["cursor"] = one["next_cursor"];
    q["limit"] = 2;
    auto two = call("query", q);
    NT_REQUIRE(two["rows"].size() == 2 && two["next_cursor"].is_null());
    q["filters"] = {{"label", "different"}};
    call("query", q, false);
  }
  void test_operation_conflict_and_expected_digest() {
    call("prepare", p);
    auto altered = p;
    altered["graph"]["nodes"][0]["properties"]["new"] = true;
    altered["graph"] = seal(altered["graph"]);
    call("prepare", altered, false);
    call("commit",
         updated(key, {{"expected_intent_digest",
                        "sha256:" + std::string(64, '0')}}),
         false);
    NT_REQUIRE(call("status", key)["state"] == "prepared");
  }
  void test_scope_isolation() {
    auto [r, c] = commit();
    call("status", updated(key, {{"namespace", "another"}}), false);
    call("export",
         updated(scope, {{"namespace", "another"},
                         {"snapshot_digest", r["snapshot_digest"]}}),
         false);
  }
  void test_complete_inventory_corruption() {
    auto [r, c] = commit();
    corrupt("UPDATE nodes SET value='{}' WHERE row_key='2'");
    call("query",
         updated(scope, {{"snapshot_digest", r["snapshot_digest"]},
                         {"kind", "nodes"},
                         {"filters", {{"id", "0"}}},
                         {"cursor", nullptr},
                         {"limit", 1}}),
         false);
    call("status", key, false);
  }
  void test_extra_row_and_schema() {
    auto [r, c] = commit();
    corrupt("INSERT INTO nodes SELECT "
            "tops_id,namespace,snapshot_digest,'extra',value,'extra' FROM "
            "nodes LIMIT 1");
    call("export", updated(scope, {{"snapshot_digest", r["snapshot_digest"]}}),
         false);
  }
  void test_unknown_schema() {
    commit();
    corrupt("CREATE TABLE unexpected(value VARCHAR)");
    call("status", key, false);
  }
  void test_graph_tamper_and_dangling_endpoint() {
    p["graph"]["owner_artifact"]["future_field"]["new"] = 1;
    call("prepare", p, false);
    p["graph"] = graph();
    p["graph"]["edges"][0]["to"] = "absent";
    p["graph"] = seal(p["graph"]);
    call("prepare", p, false);
  }
  void test_read_does_not_create_and_private_mode() {
    call("status", key, false);
    NT_REQUIRE(!fs::exists(root / "index.duckdb"));
    NT_REQUIRE(::chmod(root.c_str(), 0755) == 0);
    call("prepare", p, false);
    NT_REQUIRE(!fs::exists(root / "index.duckdb"));
    NT_REQUIRE(::chmod(root.c_str(), 0700) == 0);
  }
  void test_symlink_rejection() {
    auto outside = root / "elsewhere";
    write(outside, "caller");
    fs::create_symlink(outside, root / "index.duckdb");
    call("prepare", p, false);
    NT_REQUIRE(read(outside) == "caller");
  }
  void test_empty_graph() {
    p["graph"]["nodes"] = Json::array();
    p["graph"]["edges"] = Json::array();
    p["graph"] = seal(p["graph"]);
    auto [r, c] = commit();
    auto q =
        call("query", updated(scope, {{"snapshot_digest", r["snapshot_digest"]},
                                      {"kind", "nodes"},
                                      {"filters", Json::object()},
                                      {"cursor", nullptr},
                                      {"limit", 1}}));
    NT_REQUIRE(q["rows"].empty() && q["matched_count"] == 0);
  }
  void test_inventory_references_and_revision() {
    commit();
    auto second = call("prepare", updated(p, {{"operation_id", "two"}}));
    auto first = inventory();
    NT_REQUIRE(first["manifest"]["entries"].size() == 2);
    NT_REQUIRE(first["manifest"]["snapshots"][0]["committed_operations"] == 1);
    NT_REQUIRE(first["manifest"]["global_counts"] ==
               Json({{"intents", 2}, {"snapshots", 1}}));
    auto page = inventory({{"cursor", first["next_cursor"]},
                           {"expected_revision", first["manifest"]["digest"]}});
    NT_REQUIRE(page["records"][0]["state"] == "prepared" &&
               page["next_cursor"].is_null());
    call("commit", updated(scope, {{"operation_id", "two"},
                                   {"expected_intent_digest",
                                    second["intent"]["digest"]}}));
    call("inventory",
         updated(first["input"], {{"cursor", first["next_cursor"]}}), false);
    NT_REQUIRE(
        inventory()["manifest"]["snapshots"][0]["committed_operations"] == 2);
  }
  void test_inventory_other_scope_invalidates_without_leaking() {
    commit();
    auto first = inventory();
    call("prepare", updated(p, {{"namespace", "private-other"}}));
    call("inventory",
         updated(first["input"],
                 {{"expected_revision", first["manifest"]["digest"]}}),
         false);
    auto current = inventory();
    NT_REQUIRE(current["manifest"]["entries"].size() == 1 &&
               current["manifest"]["global_counts"]["intents"] == 2);
    auto empty = inventory({{"namespace", "empty"}});
    NT_REQUIRE(empty["records"].empty() &&
               empty["manifest"]["snapshots"].empty());
  }
  void test_inventory_orphan_rows_in_other_scope() {
    commit();
    corrupt(
        "INSERT INTO nodes SELECT "
        "tops_id,'orphan',snapshot_digest,row_key,value,id FROM nodes LIMIT 1");
    call("inventory",
         updated(scope, {{"expected_revision", nullptr},
                         {"cursor", nullptr},
                         {"limit", 1}}),
         false);
  }
  void test_inventory_missing_database_and_bad_cursor() {
    auto q = updated(
        scope,
        {{"expected_revision", nullptr}, {"cursor", nullptr}, {"limit", 1}});
    call("inventory", q, false);
    NT_REQUIRE(!fs::exists(root / "index.duckdb"));
    commit();
    auto first = inventory();
    call("inventory",
         updated(q, {{"cursor",
                      {{"revision", first["manifest"]["digest"]},
                       {"after_operation_id", "absent"}}}}),
         false);
    for (Json value : Json::array({0, 17, true, 1.5}))
      call("inventory", updated(q, {{"limit", value}}), false);
  }
};
} // namespace
int main(int argc, char **argv) {
  return test_main([&] {
    Arguments args(argc, argv);
    Engine engine{fs::absolute(args.require("engine")).string(),
                  "symphony-shv-graph-duckdb-connector"};
    engine.seal_results = false;
    using Case = void (StoreTests::*)();
    std::vector<Case> cases = {
        &StoreTests::test_descriptor_and_transfer_plan,
        &StoreTests::test_reopen_retry_export_and_pagination,
        &StoreTests::test_operation_conflict_and_expected_digest,
        &StoreTests::test_scope_isolation,
        &StoreTests::test_complete_inventory_corruption,
        &StoreTests::test_extra_row_and_schema,
        &StoreTests::test_unknown_schema,
        &StoreTests::test_graph_tamper_and_dangling_endpoint,
        &StoreTests::test_read_does_not_create_and_private_mode,
        &StoreTests::test_symlink_rejection,
        &StoreTests::test_empty_graph,
        &StoreTests::test_inventory_references_and_revision,
        &StoreTests::test_inventory_other_scope_invalidates_without_leaking,
        &StoreTests::test_inventory_orphan_rows_in_other_scope,
        &StoreTests::test_inventory_missing_database_and_bad_cursor};
    for (auto test : cases) {
      StoreTests fixture(engine, args.require("library"),
                         args.get("version", "0.3.0-dev"));
      (fixture.*test)();
    }
    engine.summary();
  });
}
