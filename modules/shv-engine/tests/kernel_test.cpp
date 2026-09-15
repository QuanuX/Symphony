#include "native_test.hpp"
using namespace native_test;
void test_kernel_conformance(Engine &e) {
  e.request_id = "kernel-check";
  std::string head = "<div id=\"product-overview\"><h1>Example CPU™</h1></div>",
              body = "<article "
                     "id=\"product-specifications\"><dl><dt>Cores</dt><dd>16</"
                     "dd><dt>Sockets</dt><dd>1P / "
                     "2P</dd><dt>Launch</dt><dd>03/15/2021</dd><dt>End</"
                     "dt><dd>done</dd></dl></article>",
              html = head + body;
  Json spec = {{"id", "cpu"},
               {"manufacturer", "Example"},
               {"model", "Example CPU™"},
               {"hardware_class", "cpu"},
               {"source_id", "source"},
               {"heading_section", "div#product-overview"},
               {"field_section", "article#product-specifications"},
               {"fields", Json::array({{{"predicate", "cores"},
                                        {"label", "Cores"},
                                        {"next_label", "Sockets"},
                                        {"value_type", "integer"},
                                        {"qualifier", "documented"}},
                                       {{"predicate", "socket_modes"},
                                        {"label", "Sockets"},
                                        {"next_label", "Launch"},
                                        {"value_type", "tokens"},
                                        {"qualifier", "finite_supported_set"}},
                                       {{"predicate", "model_introduction"},
                                        {"label", "Launch"},
                                        {"next_label", "End"},
                                        {"value_type", "date"},
                                        {"qualifier", "documented"}}})}};
  TempDir tmp("shv-kernel-");
  auto root = tmp.path;
  auto build = [&](const std::string &raw,
                   std::function<void(Json &)> change = {}, bool good = true) {
    write(root / "source.html", raw);
    Json payload = {{"source_root", root.string()},
                    {"sources", Json::array({{{"id", "source"},
                                              {"path", "source.html"},
                                              {"bytes", raw.size()},
                                              {"digest", digest(raw)},
                                              {"format", "html"}}})},
                    {"subjects", {spec}}};
    if (change)
      change(payload);
    return e.call("catalogue_build", payload, good);
  };
  build(html + char(0xff), {}, false);
  auto cat = build(html), sub = cat["subjects"][0];
  NT_REQUIRE(build(head.substr(0, head.size() - 6) + body +
                   "</div>")["subjects"] == cat["subjects"]);
  NT_REQUIRE(build(head.substr(0, head.size() - 6) +
                   "<div id=\"product-specifications\">" + body +
                   "</div></div>")["subjects"] == cat["subjects"]);
  NT_REQUIRE(sub["introduced"] ==
             Json({{"from", "2021-03-15"}, {"through", "2021-03-15"}}));
  NT_REQUIRE(sub["assertions"] ==
             Json::array({{{"predicate", "cores"},
                           {"value", 16},
                           {"qualifier", "documented"},
                           {"source_id", "source"}},
                          {{"predicate", "model_introduction"},
                           {"value", "2021-03-15"},
                           {"qualifier", "documented"},
                           {"source_id", "source"}},
                          {{"predicate", "socket_modes"},
                           {"value", {"1P", "2P"}},
                           {"qualifier", "finite_supported_set"},
                           {"source_id", "source"}}}));
  for (auto raw :
       {replace(head, "CPU™", "CPU&trade;") + body,
        std::string("<nav><h1>Wrong</h1><dt>Cores</dt><dd>99</dd></nav>") +
            html,
        std::string(
            "<script>\"</scriptx><dt>Cores</dt><dd>99</dd>\"</script>") +
            html,
        head +
            replace(
                body, "<dl>",
                "<dl><script>\"</scriptx><dt>Cores</dt><dd>99</dd>\"</script>"),
        head + replace(body, "<dl>",
                       "<dl><dt-fake>Cores</dt-fake><dd-fake>99</dd-fake>")})
    NT_REQUIRE(build(raw)["subjects"] == cat["subjects"]);
  for (auto raw :
       {replace(replace(html, "<h1>", "<h1-fake>"), "</h1>", "</h1-fake>"),
        replace(html, "<dt>Cores</dt>", "<dt-fake>Cores</dt-fake>"),
        replace(html, "Example CPU™", "Different CPU"), head + head + body,
        head + body + body,
        head + replace(body, "product-specifications", "unrelated"),
        head + replace(body, "</article>", ""),
        replace(html, "03/15/2021", "02/29/2021"),
        replace(html, "<dd>16</dd>", "<dd>016</dd>"),
        replace(html, "1P / 2P", "1P / 1P"),
        replace(html, "<dt>End</dt>", "<dt>Cores</dt>"),
        replace(html, "CPU™", "CPU&unknown;"), html + char(0)})
    build(raw, {}, false);
  std::vector<std::function<void(Json &)>> changes = {
      [](Json &p) {
        p["sources"][0]["digest"] = "sha256:" + std::string(64, '0');
      },
      [](Json &p) { p["sources"][0]["bytes"] = 1; },
      [](Json &p) { p["sources"][0]["path"] = "../source.html"; },
      [](Json &p) { p["sources"][0]["format"] = "opaque"; },
      [](Json &p) {
        p["subjects"][0]["fields"].push_back(p["subjects"][0]["fields"][0]);
      },
      [](Json &p) { p["subjects"][0]["fields"][0]["next_label"] = "End"; },
      [](Json &p) {
        p["subjects"][0]["field_section"] = "div#product-overview";
      },
      [](Json &p) { p["source_root"] = "relative"; }};
  for (auto change : changes)
    build(html, change, false);
  fs::create_symlink(root / "source.html", root / "link.html");
  build(html, [](Json &p) { p["sources"][0]["path"] = "link.html"; }, false);
  cat = build(html);
  Json query = {{"source_root", root.string()},
                {"catalogue", cat},
                {"subject_ids", {"missing", "cpu"}}};
  auto qr = e.call("catalogue_query", query);
  NT_REQUIRE(qr["missing_subject_ids"] == Json::array({"missing"}) &&
             qr["subjects"] == cat["subjects"]);
  Json reqs = Json::array({{{"id", "enough"},
                            {"predicate", "cores"},
                            {"operator", "gte"},
                            {"value", 8},
                            {"qualifier", "documented"}},
                           {{"id", "too_many"},
                            {"predicate", "cores"},
                            {"operator", "gte"},
                            {"value", 32},
                            {"qualifier", "documented"}},
                           {{"id", "dual"},
                            {"predicate", "socket_modes"},
                            {"operator", "contains"},
                            {"value", "2P"},
                            {"qualifier", "finite_supported_set"}},
                           {{"id", "quad"},
                            {"predicate", "socket_modes"},
                            {"operator", "contains"},
                            {"value", "4P"},
                            {"qualifier", "finite_supported_set"}},
                           {{"id", "board"},
                            {"predicate", "whole_board_compatible"},
                            {"operator", "eq"},
                            {"value", "yes"},
                            {"qualifier", "documented"}}});
  auto evaluation =
      e.call("evaluate", updated(query, {{"requirements", reqs}}));
  Json statuses = Json::array();
  for (auto f : evaluation["findings"])
    statuses.push_back(f["status"]);
  NT_REQUIRE(statuses == Json::array({"supported", "contradicted", "supported",
                                      "contradicted", "unresolved"}));
  NT_REQUIRE(evaluation["missing_subject_ids"] == Json::array({"missing"}));
  auto wrong = reqs;
  wrong[0]["value"] = "eight";
  e.call("evaluate", updated(query, {{"requirements", wrong}}), false);
  e.call("catalogue_query", updated(query, {{"subject_ids", {"cpu", "cpu"}}}),
         false);
  auto graph = e.call("graph_project",
                      {{"source_root", root.string()}, {"catalogue", cat}});
  Json ids = Json::array();
  for (auto n : graph["nodes"])
    ids.push_back(n["id"]);
  NT_REQUIRE(ids == Json::array({"source:source", "subject:cpu"}) &&
             graph["edges"].size() == 3);
  NT_REQUIRE(e.call("graph_validate", {{"source_root", root.string()},
                                       {"graph", graph}})["valid"] == true);
  auto forged = cat;
  forged["subjects"][0]["assertions"][0]["value"] = 99;
  forged = seal(forged);
  e.call("catalogue_query", updated(query, {{"catalogue", forged}}), false);
  e.call("evaluate",
         updated(query, {{"catalogue", forged}, {"requirements", reqs}}),
         false);
  e.call("graph_project",
         {{"source_root", root.string()}, {"catalogue", forged}}, false);
  auto badgraph = graph;
  badgraph["edges"][0]["properties"]["value"] = 99;
  e.call("graph_validate",
         {{"source_root", root.string()}, {"graph", seal(badgraph)}}, false);
  write(root / "source.html", replace(html, "<dd>16</dd>", "<dd>99</dd>"));
  e.call("catalogue_query", query, false);
  e.call("graph_validate", {{"source_root", root.string()}, {"graph", graph}},
         false);
  auto profile = e.call("coverage_default", {{"as_of", "2026-09-13"}});
  NT_REQUIRE(profile["selector"]["from"] == "2018-01-01");
  auto summary = [](std::string id, Json date) {
    return Json{{"id", id},
                {"manufacturer", "Example"},
                {"model", id},
                {"hardware_class", "cpu"},
                {"introduced", date}};
  };
  Json subjects = {
      summary("old", {{"from", "2017-01-01"}, {"through", "2017-01-01"}}),
      summary("new", {{"from", "2021-03-15"}, {"through", "2021-03-15"}}),
      summary("unknown", nullptr),
      summary("straddle", {{"from", "2017-01-01"}, {"through", "2019-01-01"}})};
  NT_REQUIRE(e.call("coverage_plan",
                    {{"profile", profile}, {"subjects", subjects}})["counts"] ==
             Json({{"included", 1}, {"excluded", 1}, {"unresolved", 2}}));
  auto plan = [&](Json selector) {
    return e.call(
        "coverage_plan",
        {{"profile", seal(updated(profile, {{"selector", selector}}))},
         {"subjects", subjects}});
  };
  NT_REQUIRE(plan({{"op", "all"}})["counts"] ==
             Json({{"included", 4}, {"excluded", 0}, {"unresolved", 0}}));
  NT_REQUIRE(plan({{"op", "date"},
                   {"basis", "model_introduction"},
                   {"from", "2010-01-01"},
                   {"through", "2026-09-13"}})["counts"] ==
             Json({{"included", 3}, {"excluded", 0}, {"unresolved", 1}}));
  NT_REQUIRE(plan({{"op", "not"}, {"arg", profile["selector"]}})["counts"] ==
             Json({{"included", 1}, {"excluded", 1}, {"unresolved", 2}}));
  NT_REQUIRE(plan({{"op", "and"},
                   {"args",
                    {profile["selector"],
                     {{"op", "ids"},
                      {"values", Json::array()}}}}})["counts"]["excluded"] ==
             4);
  NT_REQUIRE(plan({{"op", "or"},
                   {"args",
                    {profile["selector"],
                     {{"op", "all"}}}}})["counts"]["included"] == 4);
  e.call("coverage_default", {{"as_of", "2026-02-29"}}, false);
  e.call("coverage_plan",
         {{"profile",
           seal(updated(profile,
                        {{"selector", updated(profile["selector"],
                                              {{"basis", "manufacture"}})}}))},
          {"subjects", subjects}},
         false);
  e.call("coverage_plan",
         {{"profile", updated(profile, {{"as_of", "2026-09-12"}})},
          {"subjects", subjects}},
         false);
  e.call("inspect", {{"extra", true}}, false);
  auto d = e.call("inspect", Json::object());
  NT_REQUIRE(d["operations"].size() == 8 && d["network_listener"] == false);
}
void test_installed_process(const std::string &prefix) {
  Engine e{installed_engine(prefix, "shv-engine", "symphony-shv", "0.3.0-dev",
                            "vector_engine"),
           "symphony-shv"};
  test_kernel_conformance(e);
  e.summary();
}
int main(int argc, char **argv) {
  return test_main([&] {
    Arguments a(argc, argv);
    require(a.has("engine") != a.has("prefix"), "Select --engine or --prefix");
    if (a.has("prefix")) {
      test_installed_process(a.require("prefix"));
    } else {
      Engine e{a.require("engine"), "symphony-shv"};
      test_kernel_conformance(e);
      e.summary();
    }
  });
}
