#include "installed_campaign.hpp"
using namespace scv_test;
struct InventoryCampaign : Campaign {
  J target;
  using Campaign::Campaign;
  J inv(const std::string &name, const J &revision = nullptr,
        const J &cursor = nullptr, int limit = 1, const J &options = object()) {
    return qx(
        name, "inventory",
        {{"expected_revision", revision}, {"cursor", cursor}, {"limit", limit}},
        options);
  }
  J plan(const std::string &name, const J &revision, const J &ids,
         const J &selected = nullptr, const J &capacity = nullptr,
         const J &options = object()) {
    return qx(name, "transfer-plan",
              {{"expected_revision", revision},
               {"operation_ids", ids},
               {"target", selected.is_null() ? target : selected},
               {"capacity", capacity.is_null()
                                ? J{{"intents", 128}, {"snapshots", 128}}
                                : capacity}},
              options);
  }
  void test_inventory_and_transfer_planning() {
    summary["recovery_scope"] = "Inventory and planning only; no transfer, "
                                "retirement or migration execution.";
    auto destination = out / "empty-target";
    private_dir(destination);
    target = {{"prefix", connector_prefix.string()},
              {"version", "0.2.0-dev"},
              {"root", destination.string()}};
    auto first = qx("initial-import", "import",
                    {{"operation_id", "original"},
                     {"graph", graph},
                     {"query_time", time}})["connector_result"],
         snapshot = first["intent"]["snapshot"];
    auto a = inv("inventory-one")["connector_result"],
         revision = a["manifest"]["digest"];
    check(a["manifest"]["entries"].size() == 1 && a["records"][0] == first,
          "Inventory full record matches exact installed import");
    J alias = {{"tops_id", tops},
               {"namespace", namespace_name},
               {"operation_id", "alias"},
               {"graph", graph},
               {"query_time", time},
               {"owner", snapshot["owner"]},
               {"connector", snapshot["connector"]}};
    native("prepare-alias", "prepare", alias, snapshot["connector"]);
    auto b = inv("inventory-prepared")["connector_result"];
    check(
        b["manifest"]["digest"] != revision &&
            b["manifest"]["snapshots"][0]["committed_operations"] == 1,
        "Prepared alias changes revision without a second published snapshot");
    inv("stale-observation", revision, nullptr, 1, {{"ok", false}});
    auto page = inv("inventory-page-two", nullptr,
                    b["next_cursor"])["connector_result"];
    check(page["manifest"] == b["manifest"] &&
              page["records"][0]["intent"]["operation_id"] == "original",
          "Inventory pages retain one exact revision and byte-sorted record "
          "order");
    qx("recover__alias", "recover");
    auto c = inv("inventory-committed")["connector_result"];
    revision = c["manifest"]["digest"];
    check(c["manifest"]["snapshots"][0]["committed_operations"] == 2 &&
              c["manifest"]["snapshots"].size() == 1,
          "Committed aliases preserve a shared snapshot reference group");
    inv("stale-cursor", nullptr, b["next_cursor"], 1, {{"ok", false}});
    auto planned =
        plan("same-install-plan", revision, J::array({"alias", "original"}));
    check(
        planned["disposition"] == "ready" &&
            planned["owner_evaluations"].size() == 2 &&
            std::all_of(planned["owner_evaluations"].begin(),
                        planned["owner_evaluations"].end(),
                        [](const J &v) { return v["outcome"] == "validated"; }),
        "Ready plan retains native owner evaluation for every selected "
        "operation");
    auto body = planned["connector_result"];
    check(body["requirements"] == J({{"intents", 2}, {"snapshots", 1}}) &&
              std::all_of(body["selected"].begin(), body["selected"].end(),
                          [&](const J &v) {
                            return v["target_snapshot_digest"] ==
                                   snapshot["digest"];
                          }),
          "Same installation preserves snapshot identity and deduplicates "
          "required published snapshots");
    auto legacy_prefix = fs::canonical(args.require("legacy-prefix"));
    auto legacy_target = nt::updated(
        target, {{"prefix", legacy_prefix.string()}, {"version", "0.1.0-dev"}});
    auto mapped = plan("legacy-target-plan", revision, J::array({"original"}),
                       legacy_target)["connector_result"];
    check(mapped["selected"][0]["target_snapshot_digest"] !=
                  snapshot["digest"] &&
              mapped["excluded_operation_ids"] == J::array({"alias"}),
          "Different exact target installation creates explicit new identity "
          "and excludes unselected aliases");
    auto blocked =
        plan("capacity-blockers", revision, J::array({"alias", "original"}),
             nullptr, {{"intents", 0}, {"snapshots", 0}});
    J blockers = array();
    for (const auto &v : blocked["blockers"])
      blockers.push_back(v["code"]);
    check(blocked["disposition"] == "blocked" &&
              blockers == J::array({"intent_capacity", "snapshot_capacity"}),
          "Caller capacity limits are explicit plan blockers");
    nt::write(destination / "caller-marker", "retained target bytes");
    auto occupied = plan("nonempty-target", revision, J::array({"original"}));
    check(occupied["disposition"] == "blocked" &&
              std::any_of(
                  occupied["blockers"].begin(), occupied["blockers"].end(),
                  [](const J &v) { return v["code"] == "target_not_empty"; }),
          "Nonempty target is reported without modifying it");
    check(nt::read(destination / "caller-marker") == "retained target bytes" &&
              std::distance(fs::directory_iterator(destination),
                            fs::directory_iterator{}) == 1,
          "Planning creates no target database, lock or receipt");
    fs::remove(destination / "caller-marker");
    plan("missing-selection", revision, J::array({"missing"}), nullptr, nullptr,
         {{"ok", false}});
    plan("unsupported-target", revision, J::array({"original"}),
         nt::updated(target, {{"version", "9.0.0"}}), nullptr, {{"ok", false}});
    check(inv("source-after-plans")["connector_result"]["manifest"] ==
              c["manifest"],
          "Successful and blocked plans preserve complete logical source "
          "inventory");
    qx("other-namespace", "import",
       {{"operation_id", "private-other"},
        {"graph", graph},
        {"query_time", time}},
       {{"namespace", "private-other-namespace"}});
    auto d = inv("cross-namespace-inventory")["connector_result"];
    check(d["manifest"]["entries"] == c["manifest"]["entries"] &&
              d["manifest"]["digest"] != revision,
          "Other namespace activity invalidates revision without disclosing "
          "its records");
    plan("cross-namespace-stale-plan", revision, J::array({"original"}),
         nullptr, nullptr, {{"ok", false}});
    check(d.dump().find("private-other") == std::string::npos,
          "Scoped inventory omits other namespace identities");
    auto owner = clone_owner();
    auto imported = qx("copy-owner-import", "import",
                       {{"operation_id", "copy-owner"},
                        {"graph", graph},
                        {"query_time", time}},
                       {{"owner", owner.string()}})["connector_result"];
    fs::path executable =
        imported["intent"]["snapshot"]["owner"]["ExecutablePath"]
            .get<std::string>();
    auto hidden = executable;
    hidden.replace_extension(".hidden");
    {
      RestorePath restore(executable, hidden);
      auto observed = inv("inventory-without-owner")["connector_result"],
           missing = plan("plan-without-owner", observed["manifest"]["digest"],
                          J::array({"copy-owner"}));
      check(missing["disposition"] == "blocked" &&
                missing["owner_evaluations"][0]["outcome"] == "unavailable",
            "Missing exact owner permits inventory and explicitly blocks "
            "semantic transfer planning");
    }
    auto selected_prefix = connector_prefix;
    auto selected_version = connector_version;
    auto selected_root = root;
    auto selected_namespace = namespace_name;
    root = out / "legacy-index";
    private_dir(root);
    namespace_name = "legacy";
    connector_prefix = legacy_prefix;
    connector_version = "0.1.0-dev";
    auto old = qx("legacy-import", "import",
                  {{"operation_id", "legacy"},
                   {"graph", graph},
                   {"query_time", time}})["connector_result"];
    inv("legacy-inventory-rejected", nullptr, nullptr, 1, {{"ok", false}});
    connector_prefix = selected_prefix;
    connector_version = selected_version;
    auto old_inventory = inv("new-reader-legacy-inventory")["connector_result"];
    check(old_inventory["records"][0] == old,
          "New reader preserves exact old connector and intent identities");
    qx("new-reader-legacy-export-rejected", "export",
       {{"snapshot_digest", old["snapshot_digest"]}, {"query_time", time}},
       {{"ok", false}});
    connector_prefix = legacy_prefix;
    connector_version = "0.1.0-dev";
    auto exported =
        qx("exact-legacy-export", "export",
           {{"snapshot_digest", old["snapshot_digest"]}, {"query_time", time}});
    check(exported["connector_result"]["snapshot"] == old["intent"]["snapshot"],
          "Exact legacy data operations remain usable after new inventory "
          "observation");
    connector_prefix = selected_prefix;
    connector_version = selected_version;
    root = selected_root;
    namespace_name = selected_namespace;
    check(fs::is_empty(destination),
          "Destination remains empty after all planning");
    summary.update(J{{"status", "passed"},
                     {"transfers_executed", 0},
                     {"source_records_deleted", 0},
                     {"target_databases_created", 0}});
    save();
  }
};
int main(int argc, char **argv) {
  return nt::test_main([&] {
    nt::Arguments args(argc, argv);
    args.values["connector-version"] = "0.2.0-dev";
    InventoryCampaign campaign(args);
    campaign.checked_run(
        [&] { campaign.test_inventory_and_transfer_planning(); });
    campaign.print();
  });
}
