#include "profile_support.hpp"
using namespace native_test;
void run(profile_test::Process &process) {
  auto &root = process.root;
  process.request_id = "diagnose";
  process.include_exit_code = false;
  process.request_file = "input.json";
  process.save_cases = true;
  auto call = [&](const std::string &op, Json payload, bool good = true) {
    if (op == "extraction_diagnose" &&
        payload.value("source_root", "") == (root / "sources").string()) {
      std::ostringstream name;
      name << "case-" << std::setfill('0') << std::setw(3)
           << process.calls.size() << "-sources";
      auto target = root / name.str();
      fs::copy(root / "sources", target,
               fs::copy_options::recursive | fs::copy_options::copy_symlinks);
      payload["source_root"] = target.string();
    }
    return process.call(op, payload, good);
  };
  NT_REQUIRE(call("inspect", Json::object())["operations"].size() == 7);
  auto sources = root / "sources";
  fs::create_directory(sources);
  std::string raw =
      "<div id=\"head\"><h1>Fixture</h1></div><article "
      "id=\"spec\"><dl><dt>Count</dt><dd>16</dd><dt>Modes</dt><dd>1P / "
      "2P</dd><dt>End</dt><dd>done</dd></dl></article>";
  write(sources / "one.html", raw);
  Json source = {{"id", "one"},
                 {"path", "one.html"},
                 {"bytes", raw.size()},
                 {"digest", digest(raw)},
                 {"format", "html"}},
       fields = Json::array({{{"predicate", "count"},
                              {"label", "Count"},
                              {"next_label", "Modes"},
                              {"value_type", "integer"},
                              {"qualifier", "documented"}},
                             {{"predicate", "modes"},
                              {"label", "Modes"},
                              {"next_label", "End"},
                              {"value_type", "tokens"},
                              {"qualifier", "documented"}}});
  Json mapping = {{"id", "fixture"},
                  {"source_id", "one"},
                  {"manufacturer", "Fixture"},
                  {"model", "Fixture"},
                  {"hardware_class", "custom"},
                  {"heading_section", "div#head"},
                  {"field_section", "article#spec"},
                  {"fields", fields}},
       p = {{"source_root", sources.string()},
            {"sources", {source}},
            {"subjects", {mapping}}};
  auto r = call("extraction_diagnose", p);
  NT_REQUIRE(r["counts"] ==
             Json({{"extracted", 2}, {"failed", 0}, {"unavailable", 0}}));
  for (auto change : Json::array({{{"label", "absent"}},
                                  {{"next_label", "absent"}},
                                  {{"value_type", "date"}}})) {
    auto bad = p;
    bad["subjects"][0]["fields"][0].update(change);
    r = call("extraction_diagnose", bad);
    NT_REQUIRE(r["counts"] ==
               Json({{"extracted", 1}, {"failed", 1}, {"unavailable", 0}}));
    NT_REQUIRE(!r["subjects"][0]["fields"][0]["error"]["message"]
                    .get<std::string>()
                    .empty());
  }
  for (auto change : Json::array({{{"field_section", "article#missing"}},
                                  {{"heading_section", "div#missing"}},
                                  {{"field_section", "invalid-selector"}}})) {
    auto bad = p;
    bad["subjects"][0].update(change);
    NT_REQUIRE(call("extraction_diagnose", bad)["counts"]["failed"] == 2);
  }
  auto bad = p;
  bad["sources"][0]["digest"] = "sha256:" + std::string(64, '0');
  r = call("extraction_diagnose", bad);
  NT_REQUIRE(r["sources"][0]["status"] == "unverified" &&
             r["counts"]["unavailable"] == 2);
  bad = p;
  bad["sources"].push_back(
      updated(source, {{"id", "two"}, {"path", "missing.html"}}));
  bad["subjects"].push_back(
      updated(mapping, {{"id", "missing"}, {"source_id", "two"}}));
  NT_REQUIRE(call("extraction_diagnose", bad)["counts"] ==
             Json({{"extracted", 2}, {"failed", 0}, {"unavailable", 2}}));
  fs::create_symlink(sources / "one.html", sources / "link.html");
  bad = p;
  bad["sources"][0]["path"] = "link.html";
  NT_REQUIRE(call("extraction_diagnose", bad)["counts"]["unavailable"] == 2);
  bad = p;
  bad["subjects"][0]["fields"] = Json::array();
  NT_REQUIRE(call("extraction_diagnose", bad)["subjects"][0]["fields"].empty());
  std::vector<std::function<void(Json &)>> invalid_mapping = {
      [](Json &x) { x["extra"] = true; },
      [](Json &x) { x["source_root"] = "relative"; },
      [](Json &x) { x["sources"].push_back(x["sources"][0]); },
      [](Json &x) { x["subjects"][0]["source_id"] = "unknown"; },
      [](Json &x) {
        x["subjects"][0]["fields"].push_back(x["subjects"][0]["fields"][0]);
      }};
  for (auto mutate : invalid_mapping) {
    bad = p;
    mutate(bad);
    call("extraction_diagnose", bad, false);
  }
  std::string table =
      "<div id=\"head\"><h1>Table</h1><section "
      "id=\"spec\"><table><tr><th>Count</th><td>16</td></tr><tr><th>Launch</"
      "th><td>Q1'23</td></tr></table></section></div>";
  write(sources / "table.html", table);
  auto s = updated(source, {{"path", "table.html"},
                            {"bytes", table.size()},
                            {"digest", digest(table)}}),
       m = updated(mapping, {{"model", "Table"},
                             {"interpretation_profile", "scoped_tables.v1"}});
  m.erase("field_section");
  m["fields"] = Json::array({updated(fields[0], {{"section", "section#spec"},
                                                 {"next_label", "Launch"}}),
                             {{"predicate", "model_introduction"},
                              {"section", "section#spec"},
                              {"label", "Launch"},
                              {"next_label", nullptr},
                              {"value_type", "quarter_20yy"},
                              {"qualifier", "documented"}}});
  auto tp = updated(p, {{"sources", {s}}, {"subjects", {m}}});
  NT_REQUIRE(call("extraction_diagnose", tp)["counts"]["extracted"] == 2);
  bad = tp;
  bad["subjects"][0]["fields"][0]["section"] = "section#absent";
  NT_REQUIRE(call("extraction_diagnose", bad)["counts"] ==
             Json({{"extracted", 1}, {"failed", 1}, {"unavailable", 0}}));
  std::string source_digest = digest(raw);
  auto snapshot = seal({{"protocol", "fixture.snapshot"},
                        {"source_digest", source_digest}}),
       history = seal({{"protocol", "fixture.history"},
                       {"entries", {snapshot["digest"], snapshot["digest"]}}}),
       operation = seal(
           {{"protocol", "fixture.operation"}, {"history", history["digest"]}}),
       isolated = seal({{"protocol", "fixture.isolated"}});
  auto obj = [](std::string id, std::string kind, Json doc) {
    return Json{{"id", id},
                {"kind", kind},
                {"digest", doc["digest"]},
                {"document", doc}};
  };
  Json objects = Json::array({obj("snapshot", "graph_snapshot", snapshot),
                              obj("history", "catalogue_history", history),
                              obj("operation", "retained_operation", operation),
                              {{"id", "source"},
                               {"kind", "capture"},
                               {"digest", source_digest},
                               {"document", nullptr}},
                              obj("isolated", "custom", isolated)});
  Json edges = Json::array(
      {{{"from", "history"}, {"to", "snapshot"}, {"pointer", "/entries/0"}},
       {{"from", "history"}, {"to", "snapshot"}, {"pointer", "/entries/1"}},
       {{"from", "snapshot"}, {"to", "source"}, {"pointer", "/source_digest"}},
       {{"from", "operation"}, {"to", "history"}, {"pointer", "/history"}}});
  Json refs = {{"objects", objects},
               {"edges", edges},
               {"root_ids", {"operation"}},
               {"candidate_ids", {"source", "snapshot", "isolated"}}};
  r = call("references_analyze", refs);
  NT_REQUIRE(r["candidates"][0]["path"] ==
             Json::array({"operation", "history", "snapshot", "source"}));
  NT_REQUIRE(r["candidates"][1]["incoming_edges"].size() == 2);
  NT_REQUIRE(r["candidates"][2]["status"] == "not_reachable_in_supplied_graph");
  NT_REQUIRE(r["deletion_authorized"] == false &&
             r["uninspected_object_ids"] == Json::array({"source"}));
  auto multi = refs;
  multi["root_ids"].push_back("snapshot");
  NT_REQUIRE(call("references_analyze", multi)["candidates"][0]["path"] ==
             Json::array({"snapshot", "source"}));
  auto empty = refs;
  empty["root_ids"] = Json::array();
  NT_REQUIRE(call("references_analyze", empty)["reachable_ids"].empty());
  auto cycle = refs;
  cycle["edges"].push_back(
      {{"from", "history"}, {"to", "history"}, {"pointer", "/digest"}});
  NT_REQUIRE(call("references_analyze", cycle)["reachable_ids"] ==
             r["reachable_ids"]);
  auto custom = refs;
  Json doc = {{"a/b~c", source_digest}};
  custom["objects"].push_back({{"id", "custom-root"},
                               {"kind", "user_module"},
                               {"digest", digest(canonical(doc))},
                               {"document", doc}});
  custom["edges"].push_back(
      {{"from", "custom-root"}, {"to", "source"}, {"pointer", "/a~1b~0c"}});
  custom["root_ids"] = {"custom-root"};
  NT_REQUIRE(call("references_analyze", custom)["candidates"][0]["path"] ==
             Json::array({"custom-root", "source"}));
  std::vector<std::function<void(Json &)>> invalid_refs = {
      [](Json &x) { x["edges"][0]["pointer"] = "/entries/2"; },
      [](Json &x) { x["edges"][0]["pointer"] = "/entries/00"; },
      [](Json &x) { x["edges"][0]["pointer"] = "/~2bad"; },
      [](Json &x) { x["edges"][0]["to"] = "isolated"; },
      [](Json &x) { x["edges"].push_back(x["edges"][0]); },
      [](Json &x) {
        x["objects"][0]["digest"] = "sha256:" + std::string(64, '0');
      },
      [](Json &x) { x["objects"][0]["document"]["extra"] = true; },
      [](Json &x) { x["edges"][0]["from"] = "source"; },
      [](Json &x) { x["root_ids"].push_back("unknown"); },
      [](Json &x) { x["deletion_authorized"] = true; }};
  for (auto mutate : invalid_refs) {
    bad = refs;
    mutate(bad);
    call("references_analyze", bad, false);
  }
  process.summary();
}
int main(int argc, char **argv) {
  return test_main([&] {
    Arguments args(argc, argv);
    TempDir temp("shv-diagnostics-");
    auto root = args.has("cases")
                    ? fs::weakly_canonical(fs::absolute(args.get("cases")))
                    : temp.path;
    fs::create_directories(root);
    profile_test::Process process(args, root);
    run(process);
  });
}
