#include "profile_support.hpp"
using namespace native_test;
void run(profile_test::Process &process) {
  auto &root = process.root;
  auto call = [&](const std::string &op, Json p, bool good = true) {
    return process.call(op, p, good);
  };
  auto descriptor = call("inspect", Json::object());
  NT_REQUIRE(descriptor["operations"].size() ==
             (descriptor["engine_version"] == "0.1.0-dev" ? 5 : 7));
  Json sources = Json::array(), mapping = Json::array(),
       profiles = Json::array();
  struct Metric {
    std::string cls, pred, unit;
    int value;
  };
  std::vector<Metric> metrics = {
      {"gpu", "device_memory_bytes", "bytes", 16},
      {"memory", "module_capacity_bytes", "bytes", 32},
      {"storage", "namespace_capacity_bytes", "bytes", 64},
      {"networking-card", "port_line_rate_bps", "bits_per_second", 128},
      {"custom-accelerator", "caller_metric", "caller_unit", 256}};
  auto source_root = root / "sources";
  fs::create_directory(source_root);
  for (const auto &[cls, pred, unit, value] : metrics) {
    auto model = "Synthetic " + cls;
    std::string html =
        "<div id=\"overview\"><h1>" + model +
        "</h1></div><article id=\"spec\"><dl><dt>Value</dt><dd>" +
        std::to_string(value) +
        "</dd><dt>Launch</dt><dd>03/15/2021</dd><dt>End</dt><dd>done</dd></"
        "dl></article>";
    write(source_root / (cls + ".html"), html);
    sources.push_back({{"id", cls},
                       {"path", cls + ".html"},
                       {"bytes", html.size()},
                       {"digest", digest(html)},
                       {"format", "html"}});
    mapping.push_back(
        {{"id", cls},
         {"manufacturer", "Synthetic fixture"},
         {"model", model},
         {"hardware_class", cls},
         {"source_id", cls},
         {"heading_section", "div#overview"},
         {"field_section", "article#spec"},
         {"fields", Json::array({{{"predicate", pred},
                                  {"value_type", "integer"},
                                  {"qualifier", "unit=" + unit},
                                  {"label", "Value"},
                                  {"next_label", "Launch"}},
                                 {{"predicate", "model_introduction"},
                                  {"value_type", "date"},
                                  {"qualifier", "documented"},
                                  {"label", "Launch"},
                                  {"next_label", "End"}}})}});
    Json d = {
        {"id", cls + "-comparison"},
        {"revision", "v1"},
        {"hardware_class", cls},
        {"metrics",
         Json::array(
             {{{"predicate", pred},
               {"value_type", "integer"},
               {"qualifier", "unit=" + unit},
               {"required", true},
               {"description",
                "Synthetic conformance metric; caller defines its meaning."},
               {"extensions", Json::object()}},
              {{"predicate", "optional_fixture"},
               {"value_type", "string"},
               {"qualifier", "caller"},
               {"required", false},
               {"description", "Missing optional declaration is permitted."},
               {"extensions", Json::object()}}})},
        {"extensions", {{"fixture_only", true}}}};
    auto pr = call("profile_compile", d);
    profiles.push_back(pr);
    auto result =
        call("mapping_diagnose", {{"profile", pr}, {"mapping", mapping}});
    NT_REQUIRE(result["counts"] ==
               Json({{"conformant", 1},
                     {"incomplete", 0},
                     {"not_applicable", mapping.size() - 1}}));
    NT_REQUIRE(result["subjects"].back()["extension_predicates"] ==
               Json::array({"model_introduction"}));
    Json statuses = Json::array();
    for (auto f : result["subjects"].back()["findings"])
      statuses.push_back(f["status"]);
    NT_REQUIRE(statuses == Json::array({"matched", "unmapped_optional"}));
  }
  auto diag = call("mapping_diagnose",
                   {{"profile", profiles[0]}, {"mapping", mapping}});
  NT_REQUIRE(diag["counts"]["not_applicable"] == 4);
  auto d = profiles[0]["definition"];
  d["revision"] = "v2";
  d["metrics"][0]["value_type"] = "string";
  d["metrics"][0]["qualifier"] = "unit=other";
  auto changed = call("profile_compile", d);
  auto mismatch =
      call("mapping_diagnose", {{"profile", changed}, {"mapping", mapping}});
  NT_REQUIRE(mismatch["subjects"][0]["status"] == "incomplete" &&
             mismatch["subjects"][0]["findings"][0]["differences"] ==
                 Json::array({"value_type", "qualifier"}));
  d["revision"] = "v3";
  d["metrics"][0]["predicate"] = "future_metric";
  auto future = call("profile_compile", d);
  NT_REQUIRE(
      call("mapping_diagnose",
           {{"profile", future},
            {"mapping", mapping}})["subjects"][0]["findings"][0]["status"] ==
      "unmapped_required");
  d["revision"] = "v4";
  d["metrics"] = Json::array();
  auto retired = call("profile_compile", d);
  NT_REQUIRE(call("mapping_diagnose",
                  {{"profile", retired},
                   {"mapping", mapping}})["subjects"][0]["status"] ==
             "conformant");
  NT_REQUIRE(call("mapping_diagnose",
                  {{"profile", profiles[0]}, {"mapping", mapping}}) == diag);
  for (std::string type :
       {"string", "integer", "date", "tokens", "quarter_20yy", "table_rows"}) {
    d = profiles[0]["definition"];
    d["metrics"][0]["value_type"] = type;
    call("profile_compile", d);
  }
  std::vector<std::function<void(Json &)>> invalid_profiles = {
      [](Json &x) { x["extra"] = true; },
      [](Json &x) { x["metrics"].push_back(x["metrics"][0]); },
      [](Json &x) { x["metrics"][0]["value_type"] = "float"; },
      [](Json &x) { x["metrics"][0]["required"] = "yes"; },
      [](Json &x) { x["hardware_class"] = ""; },
      [](Json &x) { x["extensions"] = Json::array(); },
      [](Json &x) { x["metrics"][0]["description"] = "bad\ncontrol"; }};
  for (auto mutate : invalid_profiles) {
    d = profiles[0]["definition"];
    mutate(d);
    call("profile_compile", d, false);
  }
  auto p = profiles[0];
  p["definition"]["revision"] = "tampered";
  call("mapping_diagnose", {{"profile", p}, {"mapping", mapping}}, false);
  std::vector<std::function<void(Json &)>> invalid_mapping = {
      [](Json &m) { m[0]["extra"] = true; }, [](Json &m) { m.push_back(m[0]); },
      [](Json &m) { m[0]["fields"].push_back(m[0]["fields"][0]); },
      [](Json &m) { m[0]["fields"][0]["value_type"] = "float"; }};
  for (auto mutate : invalid_mapping) {
    auto m = mapping;
    mutate(m);
    call("mapping_diagnose", {{"profile", profiles[0]}, {"mapping", m}}, false);
  }
  auto coverage = seal({{"protocol", "symphony.shv.coverage-profile.v1"},
                        {"as_of", "2026-09-14"},
                        {"selector",
                         {{"op", "date"},
                          {"basis", "model_introduction"},
                          {"from", "2018-01-01"},
                          {"through", "2026-09-14"}}}});
  auto selected_profiles = profiles;
  selected_profiles.erase(4);
  Json locators = Json::array();
  for (auto s : sources)
    locators.push_back(
        {{"source_id", s["id"]},
         {"uri", "local-reference:" + s["path"].get<std::string>()},
         {"upstream_revision", nullptr}});
  Json definition = {{"id", "caller-universe"},
                     {"revision", "v1"},
                     {"kernel_version", "0.3.0-dev"},
                     {"coverage", coverage},
                     {"profiles", selected_profiles},
                     {"sources", sources},
                     {"mapping", mapping},
                     {"locators", locators},
                     {"extensions", {{"fixture_only", true}}}};
  auto u = call("universe_build", definition);
  Json bindings = {{"source_root", source_root.string()},
                   {"decoders", Json::object()}};
  auto bound = call("universe_bind", {{"universe", u}, {"bindings", bindings}});
  NT_REQUIRE(bound["coverage"]["counts"] ==
             Json({{"included", 5}, {"excluded", 0}, {"unresolved", 0}}));
  NT_REQUIRE(bound["unprofiled_classes"] ==
             Json::array({"custom-accelerator"}));
  NT_REQUIRE(bound["canonical_apply_enabled"] == false);
  Json gpu_values = Json::array();
  for (auto s : bound["catalogue"]["subjects"])
    if (s["id"] == "gpu")
      gpu_values.push_back(s["assertions"][0]["value"]);
  NT_REQUIRE(gpu_values == Json::array({16}));
  auto alternate = root / "alternate";
  fs::create_directory(alternate);
  for (auto s : sources)
    fs::copy_file(source_root / s["path"].get<std::string>(),
                  alternate / s["path"].get<std::string>());
  auto relocated = call(
      "universe_bind",
      {{"universe", u},
       {"bindings", updated(bindings, {{"source_root", alternate.string()}})}});
  NT_REQUIRE(relocated["catalogue"] == bound["catalogue"]);
  NT_REQUIRE(relocated["input"]["universe"] == u &&
             relocated["digest"] != bound["digest"]);
  auto moved = definition;
  moved["revision"] = "v2";
  moved["locators"][0]["uri"] = "custom-api:new-source";
  auto u2 = call("universe_build", moved);
  NT_REQUIRE(u2["digest"] != u["digest"]);
  NT_REQUIRE(call("universe_bind",
                  {{"universe", u2}, {"bindings", bindings}})["catalogue"] ==
             bound["catalogue"]);
  for (auto pair : std::vector<std::pair<Json, int>>{
           {{{"op", "all"}}, 5},
           {{{"op", "class"}, {"values", {"gpu"}}}, 1},
           {{{"op", "date"},
             {"basis", "model_introduction"},
             {"from", "2022-01-01"},
             {"through", "2026-09-14"}},
            0}}) {
    d = definition;
    d["coverage"] = seal(updated(coverage, {{"selector", pair.first}}));
    auto cu = call("universe_build", d),
         cb = call("universe_bind", {{"universe", cu}, {"bindings", bindings}});
    NT_REQUIRE(cb["coverage"]["counts"]["included"] == pair.second &&
               cb["catalogue"]["subjects"].size() == 5);
  }
  d = definition;
  d["mapping"][0]["field_section"] = "article#absent";
  auto du = call("universe_build", d);
  call("universe_bind", {{"universe", du}, {"bindings", bindings}}, false);
  std::vector<std::function<void(Json &)>> invalid_universes = {
      [&](Json &x) { x["source_root"] = source_root.string(); },
      [](Json &x) { x["kernel_version"] = "latest"; },
      [](Json &x) { x["sources"][0]["path"] = "../escape"; },
      [](Json &x) { x["mapping"][0]["source_id"] = "absent"; },
      [](Json &x) { x["profiles"].push_back(x["profiles"][0]); },
      [](Json &x) { x["locators"][0]["source_id"] = "unknown"; }};
  for (auto mutate : invalid_universes) {
    d = definition;
    mutate(d);
    call("universe_build", d, false);
  }
  for (auto b :
       {updated(bindings, {{"source_root", "relative"}}),
        updated(bindings, {{"source_root", source_root.string() + "/"}}),
        updated(bindings, {{"decoders", {{"unused", root.string()}}}})})
    call("universe_bind", {{"universe", u}, {"bindings", b}}, false);
  auto alternate_file = alternate / sources[0]["path"].get<std::string>();
  auto original_file = source_root / sources[0]["path"].get<std::string>();
  auto relocated_bindings =
      updated(bindings, {{"source_root", alternate.string()}});
  write(alternate_file, "changed");
  call("universe_bind", {{"universe", u}, {"bindings", relocated_bindings}},
       false);
  fs::remove(alternate_file);
  fs::create_symlink(original_file, alternate_file);
  call("universe_bind", {{"universe", u}, {"bindings", relocated_bindings}},
       false);
  fs::remove(alternate_file);
  fs::copy_file(original_file, alternate_file);
  call("universe_bind", {{"universe", u}, {"bindings", bindings}});
  Json classes = Json::array();
  for (auto metric : metrics)
    classes.push_back(metric.cls);
  process.summary({{"classes", classes}, {"fixture_only", true}});
}
int main(int argc, char **argv) {
  return test_main([&] {
    Arguments args(argc, argv);
    TempDir temp("shv-profile-");
    auto root = args.has("cases")
                    ? fs::weakly_canonical(fs::absolute(args.get("cases")))
                    : temp.path;
    fs::create_directories(root);
    profile_test::Process process(args, root);
    run(process);
  });
}
