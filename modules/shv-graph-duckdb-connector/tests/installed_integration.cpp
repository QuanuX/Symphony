#include "installed_support.hpp"
using namespace native_test;
void test_installed_store_provenance(const Arguments &args) {
  Evidence evidence(args);
  auto prefix = fs::canonical(args.require("prefix"));
  auto receipt = prefix / "share/symphony/receipts/shv-graph-duckdb-connector" /
                 args.require("version") / "install-receipt.json";
  auto installation = evidence.installation(
      "shv-graph-duckdb-connector", "symphony-shv-graph-duckdb-connector",
      {"graph", "store"}, receipt);
  TempDir temporary("shv-store-installed-");
  auto root = temporary.path;
  NT_REQUIRE(::chmod(root.c_str(), 0700) == 0);
  auto call = [&](const std::string &op, const Json &payload,
                  const std::string &error = "") {
    return evidence.call(installation, op, payload, error, root);
  };
  auto artifact = seal({{"protocol", "caller.example.v1"},
                        {"future_field", {{"retired", nullptr}}}});
  auto graph =
      seal({{"protocol", "symphony.graph.exchange.v1"},
            {"owner",
             {{"engine_id", "caller-owner"},
              {"engine_version", "1"},
              {"artifact_protocol", artifact["protocol"]},
              {"artifact_digest", artifact["digest"]}}},
            {"owner_artifact", artifact},
            {"nodes", Json::array({{{"id", "a"},
                                    {"labels", {"caller"}},
                                    {"properties", {{"custom", 16}}}},
                                   {{"id", "b"},
                                    {"labels", {"caller"}},
                                    {"properties", {{"retired", nullptr}}}}})},
            {"edges", Json::array({{{"id", "link"},
                                    {"from", "a"},
                                    {"to", "b"},
                                    {"label", "caller-link"},
                                    {"properties", Json::object()}}})}});
  Json scope = {{"tops_id", "00000000-0000-4000-8000-000000000001"},
                {"namespace", "caller"}};
  auto key = updated(scope, {{"operation_id", "one"}}),
       payload = updated(key, {{"graph", graph}, {"connector", installation}});
  auto prepared = call("prepare", payload);
  NT_REQUIRE(prepared["state"] == "prepared");
  call("commit",
       updated(key,
               {{"expected_intent_digest", "sha256:" + std::string(64, '0')}}),
       "connector.invalid");
  auto committed = call(
      "commit",
      updated(key, {{"expected_intent_digest", prepared["intent"]["digest"]}}));
  NT_REQUIRE(committed["state"] == "committed");
  auto exported =
      call("export",
           updated(scope, {{"snapshot_digest", committed["snapshot_digest"]}}));
  NT_REQUIRE(exported["snapshot"]["graph"] == graph &&
             exported["snapshot"]["connector"] == installation);
  call("export",
       updated(scope, {{"namespace", "other"},
                       {"snapshot_digest", committed["snapshot_digest"]}}),
       "connector.invalid");
  auto forged = graph;
  forged["edges"][0]["to"] = "absent";
  call("prepare",
       updated(payload, {{"operation_id", "forged"}, {"graph", seal(forged)}}),
       "graph.invalid_request");
  NT_REQUIRE(call("status", key) == committed);
  evidence.finish("test_installed_store_provenance");
}
int main(int argc, char **argv) {
  return test_main(
      [&] { test_installed_store_provenance(Arguments(argc, argv)); });
}
