#include "native_test.hpp"
using namespace native_test;
void test_adapter_conformance(Engine &e) {
  e.seal_results = false;
  auto call = [&](auto op, Json p, bool good = true) {
    return e.call(op, p, good);
  };
  auto artifact = seal({{"protocol", "example.independent-owner.v1"},
                        {"literal",
                         {{"x", "λ\\u2028\u2028"},
                          {"large", 9007199254740991LL},
                          {"decimal", "1.25"}}}});
  auto graph = seal(
      {{"protocol", "symphony.graph.exchange.v1"},
       {"owner",
        {{"engine_id", "example-owner"},
         {"engine_version", "exact.7"},
         {"artifact_protocol", artifact["protocol"]},
         {"artifact_digest", artifact["digest"]}}},
       {"owner_artifact", artifact},
       {"nodes",
        Json::array({{{"id", "a"},
                      {"labels", {"arbitrary"}},
                      {"properties",
                       {{"nested", {nullptr, false, {{"label", "😀"}}}}}}},
                     {{"id", "b"},
                      {"labels", Json::array()},
                      {"properties", {{"keep", -7}}}}})},
       {"edges",
        Json::array({{{"id", "edge"},
                      {"from", "a"},
                      {"to", "b"},
                      {"label", "custom"},
                      {"properties",
                       {{"opaque", {{"never_execute", "rm -rf /"}}}}}}})}});
  NT_REQUIRE(call("roundtrip", {{"graph", graph}}) ==
             seal({{"protocol", "symphony.graph.adapter-result.v1"},
                   {"backend", "portable-reference"},
                   {"operation", "roundtrip"},
                   {"graph", graph}}));
  NT_REQUIRE(call("query", {{"graph", graph},
                            {"kind", "nodes"},
                            {"ids", {"missing", "b", "a"}}}) ==
             seal({{"protocol", "symphony.graph.adapter-result.v1"},
                   {"backend", "portable-reference"},
                   {"operation", "query"},
                   {"graph_digest", graph["digest"]},
                   {"kind", "nodes"},
                   {"ids", {"missing", "b", "a"}},
                   {"rows", graph["nodes"]},
                   {"missing_ids", {"missing"}}}));
  NT_REQUIRE(call("query", {{"graph", graph},
                            {"kind", "edges"},
                            {"ids", Json::array()}})["rows"] == graph["edges"]);
  NT_REQUIRE(call("roundtrip", {{"graph", graph}})["graph"] == graph);
  std::vector<std::function<void(Json &)>> mutations = {
      [](Json &g) { g["edges"][0]["to"] = "absent"; },
      [](Json &g) { g["nodes"].push_back(g["nodes"][1]); },
      [](Json &g) { std::reverse(g["nodes"].begin(), g["nodes"].end()); },
      [](Json &g) { g["edges"].push_back(g["edges"][0]); },
      [](Json &g) { g["nodes"][0]["labels"] = {"x", "x"}; },
      [](Json &g) { g["nodes"][0]["labels"] = {"z", "a"}; },
      [](Json &g) { g["owner"]["artifact_protocol"] = "wrong"; },
      [](Json &g) { g["owner_artifact"]["literal"] = "tampered"; },
      [](Json &g) { g["extra"] = 1; },
      [](Json &g) { g["nodes"][0]["extra"] = 1; },
      [](Json &g) { g["nodes"][0]["properties"] = Json::array(); },
      [](Json &g) { g["nodes"][0]["properties"] = {{"x", 0.1}}; },
      [](Json &g) {
        g["nodes"][0]["properties"] = {{"x", 9007199254740992LL}};
      },
      [](Json &g) {
        g["nodes"][0]["properties"] = {{"x", std::string(65537, 'x')}};
      }};
  for (auto change : mutations) {
    auto changed = graph;
    change(changed);
    call("roundtrip", {{"graph", seal(changed)}}, false);
  }
  call("roundtrip",
       {{"graph",
         updated(graph, {{"digest", "sha256:" + std::string(64, '0')}})}},
       false);
  call("query", {{"graph", graph}, {"kind", "nodes"}, {"ids", {"a", "a"}}},
       false);
  call("query", {{"graph", graph}, {"kind", "sql"}, {"ids", Json::array()}},
       false);
  call("roundtrip", {{"graph", graph}, {"extra", true}}, false);
  call("unsupported", Json::object(), false);
  call("inspect", {{"extra", true}}, false);
  auto descriptor = call("inspect", Json::object());
  NT_REQUIRE(descriptor == seal(descriptor, "descriptor_digest"));
  std::set<std::string> operations;
  for (auto op : descriptor["operations"])
    operations.insert(op["operation_name"]);
  NT_REQUIRE(operations ==
             std::set<std::string>({"inspect", "roundtrip", "query"}));
  NT_REQUIRE(descriptor["canonical_apply_enabled"] == false &&
             descriptor["network_listener"] == false);
}
void test_installed_process(const std::string &prefix) {
  Engine e{installed_engine(prefix, "shv-graph-adapter",
                            "symphony-shv-graph-adapter", "0.1.0-dev",
                            "adapter"),
           "symphony-shv-graph-adapter"};
  test_adapter_conformance(e);
  e.summary();
}
int main(int argc, char **argv) {
  return test_main([&] {
    Arguments a(argc, argv);
    require(a.has("engine") != a.has("prefix"), "Select --engine or --prefix");
    if (a.has("prefix")) {
      test_installed_process(a.require("prefix"));
    } else {
      Engine e{a.require("engine"), "symphony-shv-graph-adapter"};
      test_adapter_conformance(e);
      e.summary();
    }
  });
}
