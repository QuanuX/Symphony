#include "native_test.hpp"
using namespace native_test;
namespace {
Json desired() {
  return {{"source_id", "cpu-family"},
          {"publisher", "Original component designer"},
          {"authority_role", "component-specification"},
          {"subject_ids", {"cpu-a", "cpu-b"}},
          {"locators", Json::array({{{"id", "spec"},
                                     {"uri", "https://example.com/spec"},
                                     {"format", "html"}}})}};
}
Json proposed(Json current = nullptr, Json d = nullptr,
              const std::string &op = "onboard") {
  return {{"operation_id", op},
          {"current", current},
          {"desired", d.is_null() ? desired() : d},
          {"reason", "Caller-selected source"}};
}
void test_source_conformance(Engine &e) {
  TempDir temp;
  auto root = temp.path;
  std::string body = "<h1>Component evidence fixture</h1>";
  write(root / "source.html", body);
  auto call = [&](auto op, Json p, bool fail = false) {
    return e.call(op, p, !fail);
  };
  auto desc = call("inspect", Json::object());
  NT_REQUIRE(desc["operations"].size() == 8);
  for (auto o : desc["operations"]) {
    NT_REQUIRE(o["mutability"] == "read_only" &&
               o["authorization_requirement"] == "none");
    NT_REQUIRE(o["expected_state_required"] ==
               (o["operation_name"] == "source_plan" ||
                o["operation_name"] == "source_reduce"));
  }
  auto p1 = call("source_plan", proposed()), s1 = p1["source"];
  NT_REQUIRE(p1["change_kind"] == "onboard" && s1["generation"] == 1 &&
             s1["previous_digest"].is_null());
  NT_REQUIRE(call("source_reduce",
                  {{"current", nullptr}, {"plan", p1}})["source"] == s1);
  auto d = desired();
  d["locators"][0]["uri"] = "https://new.example.net/spec";
  auto p2 = call("source_plan", proposed(s1, d, "move")), s2 = p2["source"];
  NT_REQUIRE(p2["change_kind"] == "relocation" && s2["generation"] == 2 &&
             s2["previous_digest"] == s1["digest"]);
  call("source_reduce", {{"current", s1}, {"plan", p2}});
  auto d2 = d;
  d2["publisher"] = "Variant manufacturer";
  auto p3 = call("source_plan", proposed(s2, d2, "authority"));
  NT_REQUIRE(p3["change_kind"] == "authority_change");
  NT_REQUIRE(call("source_status",
                  {{"history", {s1, s2, p3["source"]}}})["history_digests"] ==
             Json::array({s1["digest"], s2["digest"], p3["source"]["digest"]}));
  call("source_plan", proposed(s1, desired()), true);
  auto bad = desired();
  bad["source_id"] = "different";
  call("source_plan", proposed(s1, bad), true);
  for (auto change :
       Json::array({{{"generation", 3}},
                    {{"previous_digest", "sha256:" + std::string(64, '0')}}})) {
    auto forged = p2;
    forged["source"] = seal(updated(forged["source"], change));
    call("source_reduce", {{"current", s1}, {"plan", seal(forged)}}, true);
  }
  call("source_reduce",
       {{"current", s1},
        {"plan", seal(updated(p2, {{"change_kind", "onboard"}}))}},
       true);
  call("source_reduce", {{"current", s2}, {"plan", p2}}, true);
  for (auto h : Json::array({Json::array(), Json::array({s2}),
                             Json::array({s1, p3["source"]}),
                             Json::array({s1, s1}), Json::array({s2, s1})}))
    call("source_status", {{"history", h}}, true);
  for (std::string field : {"publisher", "authority_role", "subject_ids"}) {
    d = desired();
    d[field] = field == "subject_ids" ? Json::array({"cpu-c"})
                                      : Json("Different scope");
    NT_REQUIRE(call("source_plan", proposed(s1, d))["change_kind"] ==
               "authority_change");
  }
  for (std::string uri :
       {"file:///source", "https://user:pass@example.com/spec",
        "https://example.com/#secret", "https://:80/path",
        "https://example.com:080/path", "https://example.com:65536/path",
        "https://%65xample.com/", "https://[::1]/", "https://example.com/\\x",
        "https://example.com/\"x", "HTTPS://example.com/",
        "https://example.com/é"}) {
    d = desired();
    d["locators"][0]["uri"] = uri;
    call("source_plan", proposed(nullptr, d), true);
  }
  for (std::string field : {"subject_ids", "locators"}) {
    d = desired();
    d[field] = Json::array();
    call("source_plan", proposed(nullptr, d), true);
    d = desired();
    d[field].push_back(d[field][0]);
    call("source_plan", proposed(nullptr, d), true);
  }
  call("source_plan",
       proposed(seal(updated(s2, {{"generation", 32}})), desired()), true);
  Json p = {{"source_root", root.string()},
            {"source", s1},
            {"locator_id", "spec"},
            {"resolved_uri", "https://example.com/spec"},
            {"redirect_chain", Json::array()},
            {"observed_at", "2026-09-13T19:00:00Z"},
            {"upstream_revision", nullptr},
            {"manifest",
             {{"id", "capture-a"},
              {"path", "source.html"},
              {"bytes", body.size()},
              {"digest", digest(body)},
              {"format", "html"}}},
            {"completeness", "complete"},
            {"issues", Json::array()}};
  auto c1 = call("capture_import", p);
  NT_REQUIRE(c1["source"] == s1 && !c1.contains("source_root"));
  auto now = updated(p, {{"observed_at", "2026-09-13T19:00:01Z"}}),
       c2 = call("capture_import", now);
  auto compare = [&](Json a, Json b, bool fail = false) {
    return call(
        "capture_compare",
        {{"source_root", root.string()}, {"previous", a}, {"current", b}},
        fail);
  };
  auto cmp = compare(c1, c2);
  NT_REQUIRE(cmp["changes"] == Json::array({"observation_time"}) &&
             cmp["same_logical_source"] == true);
  auto c3 =
      call("capture_import",
           updated(p, {{"source", s2},
                       {"resolved_uri", "https://new.example.net/spec"}}));
  NT_REQUIRE(compare(c1, c3)["changes"] ==
             Json::array({"source_revision", "acquisition_route"}));
  for (auto change :
       Json::array({{{"locator_id", "missing"}},
                    {{"resolved_uri", "https://elsewhere.example/spec"}},
                    {{"redirect_chain", {"https://example.com/spec"}}},
                    {{"observed_at", "2026-02-30T19:00:00Z"}},
                    {{"observed_at", "2026-09-13T19:00:60Z"}},
                    {{"completeness", "partial"}}}))
    call("capture_import", updated(p, change), true);
  NT_REQUIRE(
      call("capture_import",
           updated(p, {{"completeness", "partial"},
                       {"issues",
                        {"Only a subsection retained"}}}))["completeness"] ==
      "partial");
  NT_REQUIRE(call("capture_import",
                  updated(p, {{"resolved_uri", "https://new.example.org/spec"},
                              {"redirect_chain",
                               {"https://new.example.org/spec"}}}))["source"] ==
             s1);
  for (auto change :
       Json::array({{{"bytes", body.size() + 1}},
                    {{"digest", "sha256:" + std::string(64, '0')}},
                    {{"path", "../source.html"}},
                    {{"format", "opaque"}}}))
    call("capture_import",
         updated(p, {{"manifest", updated(p["manifest"], change)}}), true);
  fs::create_symlink(root / "source.html", root / "link.html");
  call("capture_import",
       updated(p,
               {{"manifest", updated(p["manifest"], {{"path", "link.html"}})}}),
       true);
  auto newer = now;
  newer["manifest"]["id"] = "capture-b";
  auto c4 = call("capture_import", newer);
  auto g = call("graph_project",
                {{"source_root", root.string()}, {"captures", {c4, c1}}});
  NT_REQUIRE(g["nodes"].size() == 3 && g["edges"].size() == 2);
  std::vector<std::string> ids;
  for (auto n : g["nodes"])
    ids.push_back(n["id"]);
  NT_REQUIRE(std::is_sorted(ids.begin(), ids.end()));
  auto validate = [&](Json graph, bool fail = false) {
    return call("graph_validate",
                {{"source_root", root.string()}, {"graph", graph}}, fail);
  };
  validate(g);
  auto forged = g;
  forged["edges"][0]["properties"]["source_id"] = "wrong";
  validate(seal(forged), true);
  call("graph_project",
       {{"source_root", root.string()}, {"captures", {c1, c1}}}, true);
  Json nine = Json::array();
  for (int i = 0; i < 9; ++i)
    nine.push_back(c1);
  call("graph_project", {{"source_root", root.string()}, {"captures", nine}},
       true);
  call("graph_project",
       {{"source_root", root.string() + "/"}, {"captures", Json::array()}},
       true);
  call("capture_import", updated(p, {{"source_root", root.string() + "/"}}),
       true);
  call("graph_project",
       {{"source_root", root.string() + "\1"}, {"captures", Json::array()}},
       true);
  auto empty = call("graph_project", {{"source_root", root.string()},
                                      {"captures", Json::array()}});
  NT_REQUIRE(empty["nodes"].empty());
  validate(empty);
  write(root / "source.html", "changed");
  compare(c1, c2, true);
  validate(g, true);
  call("source_plan", updated(proposed(), {{"approved", true}}), true);
}
} // namespace
void test_installed_process(const std::string &prefix) {
  Engine e{installed_engine(prefix, "shv-source-engine", "symphony-shv-source",
                            "0.1.0-dev", "vector_engine"),
           "symphony-shv-source"};
  test_source_conformance(e);
  e.summary();
}
int main(int argc, char **argv) {
  return test_main([&] {
    Arguments a(argc, argv);
    require(a.has("engine") != a.has("prefix"), "Select --engine or --prefix");
    if (a.has("prefix")) {
      test_installed_process(a.require("prefix"));
    } else {
      Engine e{a.require("engine"), "symphony-shv-source"};
      test_source_conformance(e);
      e.summary();
    }
  });
}
