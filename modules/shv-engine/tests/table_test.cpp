#include "native_test.hpp"
using namespace native_test;
void test_table_interpretation(Engine &e, const std::string &version) {
  e.request_id = "tables";
  e.seal_results = false;
  std::string
      head = "<div id=\"heading\"><h1>Example CPU®</h1></div>",
      scalar = "<section "
               "id=\"spec\"><table><tr><th>Cores</th><td>24</td></"
               "tr><tr><th>Launch</th><td>Q1'23</td></tr><tr><th>End</"
               "th><td>done</td></tr></table></section>",
      matrix =
          "<section "
          "id=\"modes\"><table><tr><th>Config</th><th>Cores</th><th>Frequency</"
          "th><th>TDP</th><th>Description</th></tr><tr><td>CPU(0)</td><td>24</"
          "td><td>2.0</td><td>185</td><td></td></tr><tr><td>CPU(1)</td><td>16</"
          "td><td>2.3</td><td>165</td><td></td></tr><tr><td>CPU(2)</td><td>12</"
          "td><td>2.7</td><td>165</td><td></td></tr></table></section>",
      html = head + scalar + matrix;
  Json spec = {
      {"id", "cpu"},
      {"manufacturer", "Example"},
      {"model", "Example CPU®"},
      {"hardware_class", "cpu"},
      {"source_id", "s"},
      {"heading_section", "div#heading"},
      {"interpretation_profile", "scoped_tables.v1"},
      {"fields",
       Json::array({{{"predicate", "cores"},
                     {"section", "section#spec"},
                     {"label", "Cores"},
                     {"next_label", "Launch"},
                     {"value_type", "integer"},
                     {"qualifier", "documented"}},
                    {{"predicate", "model_introduction"},
                     {"section", "section#spec"},
                     {"label", "Launch"},
                     {"next_label", "End"},
                     {"value_type", "quarter_20yy"},
                     {"qualifier", "documented"}},
                    {{"predicate", "profiles"},
                     {"section", "section#modes"},
                     {"columns",
                      {"Config", "Cores", "Frequency", "TDP", "Description"}},
                     {"value_type", "table_rows"},
                     {"qualifier", "source_rows_units_unspecified"}}})}};
  TempDir temp("shv-tables-");
  auto root = temp.path;
  auto build = [&](const std::string &raw,
                   std::function<void(Json &)> mutate = {}, bool good = true) {
    write(root / "s.html", raw);
    Json payload = {{"source_root", root.string()},
                    {"sources", Json::array({{{"id", "s"},
                                              {"path", "s.html"},
                                              {"bytes", raw.size()},
                                              {"digest", digest(raw)},
                                              {"format", "html"}}})},
                    {"subjects", {spec}}};
    if (mutate)
      mutate(payload);
    return e.call("catalogue_build", payload, good);
  };
  auto cat = build(html);
  Json values = Json::object();
  for (auto a : cat["subjects"][0]["assertions"])
    values[a["predicate"].get<std::string>()] = a["value"];
  NT_REQUIRE(values["cores"] == 24);
  NT_REQUIRE(values["model_introduction"] == Json({{"precision", "quarter"},
                                                   {"source_text", "Q1'23"},
                                                   {"from", "2023-01-01"},
                                                   {"through", "2023-03-31"}}));
  NT_REQUIRE(cat["subjects"][0]["introduced"] ==
             Json({{"from", "2023-01-01"}, {"through", "2023-03-31"}}));
  Json expected_rows = {{"CPU(0)", "24", "2.0", "185", ""},
                        {"CPU(1)", "16", "2.3", "165", ""},
                        {"CPU(2)", "12", "2.7", "165", ""}};
  NT_REQUIRE(
      values["profiles"] ==
      Json({{"columns", {"Config", "Cores", "Frequency", "TDP", "Description"}},
            {"rows", expected_rows}}));
  auto last_field = [](Json &p) {
    p["subjects"][0]["fields"][0].update(
        {{"label", "End"}, {"next_label", nullptr}, {"value_type", "string"}});
  };
  auto last = build(html, last_field);
  for (auto a : last["subjects"][0]["assertions"])
    if (a["predicate"] == "cores")
      NT_REQUIRE(a["value"] == "done");
  build(replace(html, "</table></section>",
                "<tr><th>Appended</th><td>new</td></tr></table></section>", 1),
        last_field, false);
  build(
      html,
      [](Json &p) { p["subjects"][0]["fields"][0]["next_label"] = nullptr; },
      false);
  std::string marker = "<tr><td>CPU(2)</td><td>12</td><td>2.7</td><td>165</"
                       "td><td></td></tr>",
              extra;
  for (int i = 3; i < 32; ++i)
    extra += replace(marker, "CPU(2)", "CPU(" + std::to_string(i) + ")");
  NT_REQUIRE(
      build(replace(html, marker,
                    marker +
                        extra))["subjects"][0]["assertions"][2]["value"]["rows"]
          .size() == 32);
  build(replace(html, marker,
                marker + extra + replace(marker, "CPU(2)", "CPU(32)")),
        {}, false);
  for (auto [quarter, end] : std::vector<std::pair<std::string, std::string>>{
           {"1", "03-31"}, {"2", "06-30"}, {"3", "09-30"}, {"4", "12-31"}})
    NT_REQUIRE(
        build(replace(html, "Q1'23",
                      "Q" + quarter +
                          "'00"))["subjects"][0]["introduced"]["through"] ==
        Json("2000-" + end));
  for (auto raw :
       {replace(html, "CPU®", "CPU&#174;"),
        replace(replace(html, "<table>", "<table><tbody>"), "</table>",
                "</tbody></table>"),
        std::string(
            "<nav><table><tr><th>Cores</th><td>99</td></tr></table></nav>") +
            html,
        replace(html, "<td>24</td>",
                "<td><span>24</span><!-- ignored "
                "--><script>\"</scriptx>99\"</script></td>",
                1)})
    NT_REQUIRE(build(raw)["subjects"] == cat["subjects"]);
  std::vector<std::pair<std::string, std::string>> invalid = {
      {"<h1>", "<h1-fake>"},
      {"Example CPU®", "Wrong CPU"},
      {"id=\"spec\"", "id=\"missing\""},
      {"Q1'23", "Q0'23"},
      {"Q1'23", "Q5'23"},
      {"Q1'23", "Q1’23"},
      {"Q1'23", "Q1'2023"},
      {"Q1'23", "Q1'2x"},
      {"<td>24</td>", "<td>024</td>"},
      {"<th>Launch</th>", "<th>Cores</th>"},
      {"<th>End</th>", "<th>Different</th>"},
      {"<th>Config</th>", "<th>Renamed</th>"},
      {"<td>CPU(1)</td>", "<td>CPU(0)</td>"},
      {"<td>CPU(1)</td>", "<td></td>"},
      {"<td>CPU(1)</td>", "<th>CPU(1)</th>"},
      {"<td>2.3</td>", ""},
      {"<td>2.3</td>", "<td colspan=\"1\">2.3</td>"},
      {"<td>2.3</td>", "<td rowspan=\"2\">2.3</td>"},
      {"<td>2.3</td>", "<td>2.3<td>3.0</td></td>"},
      {"<td>2.3</td>", "<td><table><tr><td>2.3</td></tr></table></td>"},
      {"<td>2.3</td>", "<td>" + std::string(4097, 'x') + "</td>"},
      {"</table></section>", "</section>"},
      {"<td>24</td>", "<td>24</td> only valid in reduced mode"},
      {"<td>24</td>", "<td>24</td><div> only valid in reduced mode </div>"},
      {"<td>24</td>", "<td>24</td colspan=\"2\">"},
      {"<table>", "<table><tbody>"},
      {"<td>24</td>", "<td><tbody>24</tbody></td>"}};
  std::string budget = "<td>24";
  for (int i = 0; i < 33000; ++i)
    budget += "<b></b>";
  budget += "</td>";
  invalid.emplace_back("<td>24</td>", budget);
  for (auto [old, value] : invalid)
    build(replace(html, old, value, 1), {}, false);
  for (auto raw :
       {head + head + scalar + matrix, head + scalar + scalar + matrix,
        head + scalar +
            replace(matrix, "</section>",
                    "<table><tr><td>x</td></tr></table></section>"),
        html + char(0), html + char(0xff)})
    build(raw, {}, false);
  std::vector<std::function<void(Json &)>> mutations = {
      [](Json &p) {
        p["subjects"][0]["interpretation_profile"] = "unknown.v1";
      },
      [](Json &p) { p["subjects"][0]["field_section"] = "section#spec"; },
      [](Json &p) { p["subjects"][0]["fields"][0]["section"] = "div#heading"; },
      [](Json &p) {
        p["subjects"][0]["fields"][2]["columns"] = {"Config", "Config"};
      },
      [](Json &p) { p["subjects"][0]["fields"][2]["columns"] = Json::array(); },
      [](Json &p) {
        p["subjects"][0]["fields"][0]["value_type"] = "table_rows";
      },
      [](Json &p) { p["subjects"][0]["fields"][1]["value_type"] = "string"; },
      [](Json &p) {
        p["subjects"][0]["fields"].push_back(p["subjects"][0]["fields"][0]);
      }};
  for (auto mutate : mutations)
    build(html, mutate, false);
  cat = build(html);
  Json query = {{"source_root", root.string()},
                {"catalogue", cat},
                {"subject_ids", {"cpu"}}};
  NT_REQUIRE(e.call("catalogue_query", query)["subjects"] == cat["subjects"]);
  Json req = {{"id", "cores"},
              {"predicate", "cores"},
              {"operator", "gte"},
              {"value", 20},
              {"qualifier", "documented"}};
  NT_REQUIRE(
      e.call("evaluate", updated(query, {{"requirements",
                                          {req}}}))["findings"][0]["status"] ==
      "supported");
  e.call(
      "evaluate",
      updated(
          query,
          {{"requirements",
            {updated(req, {{"predicate", "profiles"},
                           {"operator", "eq"},
                           {"value", "24"},
                           {"qualifier", "source_rows_units_unspecified"}})}}}),
      false);
  auto graph = e.call("graph_project",
                      {{"source_root", root.string()}, {"catalogue", cat}});
  NT_REQUIRE(graph["owner"]["engine_version"] == version);
  NT_REQUIRE(e.call("graph_validate", {{"source_root", root.string()},
                                       {"graph", graph}})["valid"] == true);
  auto wrong_version = graph;
  wrong_version["owner"]["engine_version"] = "0.1.0-dev";
  e.call("graph_validate",
         {{"source_root", root.string()}, {"graph", seal(wrong_version)}},
         false);
  for (std::string mutation : {"date", "row"}) {
    auto forged = cat;
    if (mutation == "date")
      forged["subjects"][0]["introduced"]["through"] = "2023-01-01";
    else
      for (auto &a : forged["subjects"][0]["assertions"])
        if (a["predicate"] == "profiles")
          a["value"]["rows"][0][2] = "2.7";
    forged = seal(forged);
    e.call("catalogue_query", updated(query, {{"catalogue", forged}}), false);
    e.call("graph_project",
           {{"source_root", root.string()}, {"catalogue", forged}}, false);
  }
  auto sub = cat["subjects"][0];
  sub.erase("assertions");
  auto profile = seal({{"protocol", "symphony.shv.coverage-profile.v1"},
                       {"as_of", "2026-09-13"},
                       {"selector",
                        {{"op", "date"},
                         {"basis", "model_introduction"},
                         {"from", "2023-02-01"},
                         {"through", "2026-09-13"}}}});
  NT_REQUIRE(e.call("coverage_plan",
                    {{"profile", profile},
                     {"subjects", {sub}}})["counts"]["unresolved"] == 1);
  write(root / "s.html", replace(html, "<td>2.0</td>", "<td>2.7</td>"));
  e.call("catalogue_query", query, false);
}
void test_installed_table_process(const std::string &prefix,
                                  const std::string &version) {
  Engine e{installed_engine(prefix, "shv-engine", "symphony-shv", "0.3.0-dev"),
           "symphony-shv"};
  test_table_interpretation(e, version);
  e.summary();
}
int main(int argc, char **argv) {
  return test_main([&] {
    Arguments a(argc, argv);
    require(a.has("engine") != a.has("prefix"), "Select --engine or --prefix");
    auto version = a.get("version", "0.3.0-dev");
    if (a.has("prefix")) {
      test_installed_table_process(a.require("prefix"), version);
    } else {
      Engine e{a.require("engine"), "symphony-shv"};
      test_table_interpretation(e, version);
      e.summary();
    }
  });
}
