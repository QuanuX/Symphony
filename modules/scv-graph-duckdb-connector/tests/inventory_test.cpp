#include "connector_test_support.hpp"
using namespace scv_test;
struct InventoryTests : ConnectorTests {
  using ConnectorTests::ConnectorTests;
  J inventory(const J &revision = nullptr, const J &cursor = nullptr,
              const J &limit = 1, const J &name = nullptr, bool okay = true) {
    return call("inventory",
                {{"tops_id", base["tops_id"]},
                 {"namespace", name.is_null() ? base["namespace"] : name},
                 {"expected_revision", revision},
                 {"cursor", cursor},
                 {"limit", limit}},
                okay);
  }
  J plan_input(const J &inv, const J &ids) {
    return {{"tops_id", base["tops_id"]},
            {"namespace", base["namespace"]},
            {"expected_revision", inv["manifest"]["digest"]},
            {"operation_ids", ids},
            {"source_connector", base["connector"]},
            {"target_connector",
             installation("scv-graph-duckdb-connector",
                          "scv-graph-duckdb-connector", "0.2.0-dev",
                          (root / "target-package").string())},
            {"target_root", (root / "not-created-target").string()},
            {"capacity", {{"intents", 128}, {"snapshots", 128}}}};
  }
  void test_inventory_revision_alias_and_pages() {
    commit();
    auto a = inventory(),
         alias = nt::updated(base, {{"operation_id", "alias"}});
    call("prepare", alias);
    auto b = inventory();
    NT_REQUIRE(a["manifest"]["digest"] != b["manifest"]["digest"]);
    NT_REQUIRE(b["manifest"]["snapshots"].size() == 1);
    NT_REQUIRE(b["manifest"]["snapshots"][0]["operation_ids"] ==
               J::array({"alias", "prepare-1"}));
    NT_REQUIRE(b["manifest"]["snapshots"][0]["committed_operations"] == 1);
    NT_REQUIRE(b["records"][0]["intent"]["operation_id"] == "alias");
    auto page = inventory(nullptr, b["next_cursor"]);
    NT_REQUIRE(page["manifest"] == b["manifest"]);
    NT_REQUIRE(page["records"][0]["intent"]["operation_id"] == "prepare-1");
    NT_REQUIRE(page["next_cursor"].is_null());
    call("prepare", alias);
    NT_REQUIRE(inventory()["manifest"] == b["manifest"]);
    call("commit", nt::updated(status_input(alias),
                               {{"expected_intent_digest",
                                 b["records"][0]["intent"]["digest"]}}));
    auto c = inventory();
    NT_REQUIRE(c["manifest"]["digest"] != b["manifest"]["digest"]);
    inventory(nullptr, b["next_cursor"], 1, nullptr, false);
    inventory(b["manifest"]["digest"], nullptr, 1, nullptr, false);
  }
  void test_inventory_other_namespace_revision_without_disclosure() {
    commit();
    auto a = inventory();
    call("prepare", nt::updated(base, {{"namespace", "other-secret"},
                                       {"operation_id", "other-op"}}));
    auto b = inventory();
    NT_REQUIRE(a["manifest"]["digest"] != b["manifest"]["digest"]);
    NT_REQUIRE(a["manifest"]["entries"] == b["manifest"]["entries"]);
    NT_REQUIRE((b["manifest"]["global_counts"] ==
                J{{"intents", 2}, {"snapshots", 1}}));
    NT_REQUIRE(b.dump().find("other-secret") == std::string::npos);
    NT_REQUIRE(b.dump().find("other-op") == std::string::npos);
    auto empty = inventory(nullptr, nullptr, 1, "empty");
    NT_REQUIRE(empty["manifest"]["entries"] == array());
    NT_REQUIRE(empty["records"] == array());
  }
  void test_inventory_rejects_orphan_rows_and_snapshots() {
    commit();
    corrupt({"INSERT INTO nodes SELECT "
             "tops_id,'orphan',snapshot_digest,row_key,value,node_id,kind,"
             "capture_digest FROM nodes LIMIT 1"});
    inventory(nullptr, nullptr, 1, nullptr, false);
    corrupt(
        {"DELETE FROM nodes WHERE namespace='orphan'", "DELETE FROM intents"});
    inventory(nullptr, nullptr, 1, nullptr, false);
  }
  void test_transfer_plan_preserves_selection_and_lineage() {
    commit();
    call("prepare", nt::updated(base, {{"operation_id", "alias"}}));
    auto inv = inventory(nullptr, nullptr, 16), before = inv["manifest"],
         input = plan_input(inv, J::array({"prepare-1", "alias"})),
         plan = call("transfer_plan", input);
    J selected = array();
    for (const auto &row : plan["selected"])
      selected.push_back(row["source"]["intent"]["operation_id"]);
    NT_REQUIRE(selected == input["operation_ids"]);
    NT_REQUIRE((plan["requirements"] == J{{"intents", 2}, {"snapshots", 1}}));
    NT_REQUIRE(plan["disposition"] == "ready");
    NT_REQUIRE(plan["excluded_operation_ids"] == array());
    for (const auto &row : plan["selected"]) {
      auto intent = row["source"]["intent"], snapshot = intent["snapshot"];
      snapshot["connector"] = input["target_connector"];
      intent["snapshot"] = nt::seal(snapshot);
      NT_REQUIRE(row["target_snapshot_digest"] == intent["snapshot"]["digest"]);
      NT_REQUIRE(row["target_intent_digest"] == nt::seal(intent)["digest"]);
    }
    NT_REQUIRE(!fs::exists(root / "not-created-target"));
    NT_REQUIRE(inventory()["manifest"] == before);
    input["operation_ids"] = J::array({"alias"});
    auto partial = call("transfer_plan", input);
    NT_REQUIRE(
        (partial["requirements"] == J{{"intents", 1}, {"snapshots", 0}}));
    NT_REQUIRE(partial["excluded_operation_ids"] == J::array({"prepare-1"}));
  }
  void test_transfer_plan_capacity_and_stale_rejection() {
    commit();
    auto inv = inventory(), p = plan_input(inv, J::array({"prepare-1"}));
    p["capacity"] = {{"intents", 0}, {"snapshots", 0}};
    auto blocked = call("transfer_plan", p);
    NT_REQUIRE(blocked["blockers"] ==
               J::array({"intent_capacity", "snapshot_capacity"}));
    NT_REQUIRE(blocked["disposition"] == "blocked");
    for (const auto &change :
         J::array({{{"operation_ids", J::array({"missing"})}},
                   {{"operation_ids", J::array({"prepare-1", "prepare-1"})}},
                   {{"expected_revision", "sha256:" + std::string(64, '0')}},
                   {{"capacity", {{"intents", 129}, {"snapshots", 128}}}}}))
      call("transfer_plan", nt::updated(p, change), false);
    NT_REQUIRE(inventory()["manifest"] == inv["manifest"]);
  }
  void test_inventory_shapes_and_missing_store() {
    inventory(nullptr, nullptr, 1, nullptr, false);
    NT_REQUIRE(fs::is_empty(root));
    commit();
    for (const J &limit :
         J::array({0, 17, 1.5, 4294967297LL, 9007199254740991LL}))
      inventory(nullptr, nullptr, limit, nullptr, false);
    inventory(nullptr,
              {{"revision", "sha256:" + std::string(64, '0')},
               {"after_operation_id", "prepare-1"}},
              1, nullptr, false);
  }
  void test_transfer_plan_rejects_unsupported_target() {
    commit();
    auto p = plan_input(inventory(), J::array({"prepare-1"}));
    p["target_connector"]["Version"] = "9.0.0";
    call("transfer_plan", p, false);
  }
};
int main(int argc, char **argv) {
  return nt::test_main([&] {
    nt::Arguments args(argc, argv);
    Config config;
    config.engine = fs::canonical(args.require("engine")).string();
    config.library = fs::canonical(args.require("library")).string();
    config.version = "0.2.0-dev";
    nt::TempDir temporary;
    config.evidence = args.has("evidence")
                          ? fs::absolute(args.require("evidence"))
                          : temporary.path / "evidence";
    NT_REQUIRE(!fs::exists(config.evidence));
    fs::create_directories(config.evidence);
    const std::pair<const char *, void (InventoryTests::*)()> cases[] = {
#define SCV_INV_CASE(name) {#name, &InventoryTests::name}
        SCV_INV_CASE(test_inventory_revision_alias_and_pages),
        SCV_INV_CASE(
            test_inventory_other_namespace_revision_without_disclosure),
        SCV_INV_CASE(test_inventory_rejects_orphan_rows_and_snapshots),
        SCV_INV_CASE(test_transfer_plan_preserves_selection_and_lineage),
        SCV_INV_CASE(test_transfer_plan_capacity_and_stale_rejection),
        SCV_INV_CASE(test_inventory_shapes_and_missing_store),
        SCV_INV_CASE(test_transfer_plan_rejects_unsupported_target)
#undef SCV_INV_CASE
    };
    for (const auto &[name, method] : cases) {
      InventoryTests suite(config);
      (suite.*method)();
      std::cout << name << " passed\n";
    }
    std::cout << J{{"status", "passed"},
                   {"tests", std::size(cases)},
                   {"calls", config.calls}}
                     .dump()
              << "\n";
  });
}
