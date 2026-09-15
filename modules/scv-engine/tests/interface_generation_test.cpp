#include "scv_interface.hpp"
#include "test_support.hpp"
using namespace symphony::authoring;
struct InterfaceGeneration {
  Arguments args;
  Json manifest;
  TempDir temp;
  fs::path root;
  Json catalog;
  std::string catalog_path;
  const std::string fixture_path =
      "knowledge/scv/schemas/v1/fixture.schema.json";
  explicit InterfaceGeneration(const Arguments &selected)
      : args(selected), manifest(read_json(args.root / scv::manifest_path)),
        root(temp.path), catalog_path(string(manifest.at("schema_catalog"))) {
    write_json(root / scv::manifest_path, manifest);
    test::copy(args.root, root, scv::history_path);
    catalog = {{"protocol", "symphony.scv.schema-catalog.v1"},
               {"engine_version", manifest.at("current_release")},
               {"entries", Json::array()}};
    Json definitions = Json::object();
    for (const auto &op : manifest.at("operations"))
      for (const auto &[field, kind] :
           std::vector<std::pair<std::string, std::string>>{
               {"input_protocol", "input"}, {"output_protocol", "output"}}) {
        const auto protocol = string(op.at(field));
        definitions[protocol] = {{"type", "object"}};
        catalog["entries"].push_back(
            {{"protocol", protocol},
             {"kind", kind},
             {"operations", Json::array({op.at("name")})},
             {"file", "fixture.schema.json"},
             {"fragment", "#/$defs/" + protocol}});
      }
    write_json(root / catalog_path, catalog);
    write_json(root / fixture_path, {{"$defs", definitions}});
  }
  void test_historical_projection_is_frozen() {
    scv::check_history(manifest, args.root);
    const std::vector<int> counts{13, 17, 20, 21, 22, 26, 26, 28, 30, 30};
    for (int n = 1; n <= 10; ++n)
      test::check(scv::projection(manifest, "0." + std::to_string(n) + ".0-dev")
                          .at("operations")
                          .size() == static_cast<std::size_t>(counts.at(n - 1)),
                  "historical operation count");
    auto changed = manifest;
    changed["operations"][1]["output_protocol"] = "symphony.scv.changed.v1";
    test::rejects([&] { scv::check_history(changed, args.root); },
                  "frozen projection", "frozen interface changed");
  }
  void test_malformed_history_release_inventory_rejected() {
    const auto history = read_json(root / scv::history_path);
    for (const auto &invalid :
         Json::array({nullptr,
                      Json{{"arbitrary-key", history.at("releases").at(0)}}})) {
      auto changed = history;
      changed["releases"] = invalid;
      write_json(root / scv::history_path, changed);
      test::rejects([&] { scv::check_history(manifest, root); },
                    "malformed frozen release inventory",
                    "frozen history releases must be an array");
    }
  }
  void test_future_domain_does_not_change_old_admission_or_definition() {
    auto future = manifest;
    auto release = future.at("releases").back();
    release["version"] = "0.11.0-dev";
    future["releases"].push_back(release);
    future["current_release"] = "0.11.0-dev";
    future["domains"].push_back(
        {{"name", "scev-new-provider"}, {"introduced_in", "0.11.0-dev"}});
    scv::validate(future);
    for (const auto &r : manifest.at("releases")) {
      const auto version = r.at("version");
      test::equal(scv::projection(manifest, version),
                  scv::projection(future, version),
                  "future domain preserves previous projection");
      test::check(digest(scv::release_manifest(manifest, version), false) ==
                      digest(scv::release_manifest(future, version), false),
                  "future domain preserves definition digest");
    }
    test::check(contains(scv::projection(future, "0.11.0-dev").at("domains"),
                         "scev-new-provider"),
                "future domain admitted only in new release");
  }
  void frozen_release(int n) {
    const auto version = "0." + std::to_string(n) + ".0-dev";
    const auto old = read_json(
        args.root / ("modules/scv-engine/tests/fixtures/owner-interface-0." +
                     std::to_string(n) + ".v1.json"));
    test::equal(old.at("current_release"), version,
                "frozen declaration version");
    test::equal(scv::release_manifest(manifest, version), old,
                "complete frozen release declaration");
    test::equal(scv::projection(manifest, version),
                scv::projection(old, version),
                "complete frozen release projection");
  }
  void test_sixth_release_declaration_remains_exact() {
    frozen_release(6);
    test::check(!contains(scv::projection(manifest, "0.6.0-dev").at("surfaces"),
                          "composition_workflow"),
                "sixth release excludes later workflow");
    test::check(contains(scv::projection(manifest, "0.7.0-dev").at("surfaces"),
                         "composition_workflow"),
                "seventh release admits workflow");
  }
  void test_seventh_release_declaration_remains_exact() { frozen_release(7); }
  void test_eighth_release_declaration_remains_exact() { frozen_release(8); }
  void test_ninth_release_declaration_remains_exact() {
    frozen_release(9);
    test::check(!contains(scv::projection(manifest, "0.9.0-dev").at("surfaces"),
                          "composition_bundle_workflow"),
                "ninth release excludes later bundle workflow");
    test::check(contains(scv::projection(manifest, "0.10.0-dev").at("surfaces"),
                         "composition_bundle_workflow"),
                "tenth release admits bundle workflow");
  }
  void test_complete_generation_is_deterministic() {
    const auto output = scv::render(manifest, root);
    test::check(output == scv::render(manifest, root),
                "deterministic complete generation");
    test::check(output.size() == 3 && output[0].first == scv::cpp_path &&
                    output[1].first == scv::go_path &&
                    output[2].first == scv::cmake_path,
                "complete output inventory");
    test::check(
        output[2].second.find("OWNER-INTERFACE.json") != std::string::npos &&
            output[2].second.find("fixture.schema.json") != std::string::npos,
        "CMake inventory names owner and schema");
  }
  void test_check_detects_output_drift() {
    const std::vector<std::string> command{SCV_GENERATOR_EXECUTABLE, "--root",
                                           root.string()};
    const auto generated = run(command);
    test::check(generated.status == 0,
                "generator CLI succeeds: " + generated.error);
    auto check = command;
    check.push_back("--check");
    test::check(run(check).status == 0, "generator CLI exact check succeeds");
    write_bytes(root / scv::go_path,
                read_bytes(root / scv::go_path) + "// drift\n");
    const auto result = run(check);
    test::check(result.status != 0 &&
                    result.error.find("generated interface drift") !=
                        std::string::npos,
                "generator CLI drift rejection");
  }
  void test_malformed_manifest_rejected() {
    const std::vector<std::function<void(Json &)>> mutations{
        [](Json &m) { m["unknown"] = true; },
        [](Json &m) { m["current_release"] = "0.11.0-dev"; },
        [](Json &m) {
          const auto d = m["domains"][0];
          m["domains"].push_back(d);
        },
        [](Json &m) { m["domains"][0]["introduced_in"] = "unknown"; },
        [](Json &m) {
          const auto op = m["operations"][0];
          m["operations"].push_back(op);
        },
        [](Json &m) { m["operations"][0]["handler"] = "run_shell"; },
        [](Json &m) { m["operations"][0]["mutability"] = "unrestricted"; },
        [](Json &m) { m["operations"][0]["expected_state"] = 1; },
        [](Json &m) {
          m["operations"][0]["interactions"] = Json::array({Json::object()});
        },
        [](Json &m) {
          m["operations"].back()["artifact"]["introduced_in"] = "0.1.0-dev";
        },
        [](Json &m) {
          m["releases"][0]["companions"] = Json::array({Json::object()});
        }};
    for (std::size_t i = 0; i < mutations.size(); ++i) {
      auto bad = manifest;
      mutations[i](bad);
      test::rejects([&] { scv::validate(bad); },
                    "malformed declaration " + std::to_string(i));
    }
  }
  void test_malformed_catalog_and_references_rejected() {
    const std::vector<std::function<void(Json &)>> mutations{
        [](Json &c) { c["engine_version"] = "0.5.0-dev"; },
        [](Json &c) { c["entries"].erase(0); },
        [](Json &c) {
          const auto entry = c["entries"][0];
          c["entries"].push_back(entry);
        },
        [](Json &c) { c["entries"].push_back(4); },
        [](Json &c) {
          c["entries"][0]["kind"] =
              c["entries"][0]["kind"] == "output" ? "input" : "output";
        },
        [](Json &c) { c["entries"][0]["file"] = "../escape.schema.json"; },
        [](Json &c) { c["entries"][0]["fragment"] = "#/$defs/missing"; }};
    for (std::size_t i = 0; i < mutations.size(); ++i) {
      auto bad = catalog;
      mutations[i](bad);
      write_json(root / catalog_path, bad);
      test::rejects([&] { scv::schema_inventory(manifest, root); },
                    "malformed catalog " + std::to_string(i));
    }
    write_json(root / catalog_path, catalog);
    const auto doc = read_json(root / fixture_path);
    for (const auto *ref :
         {"https://example.invalid/schema", "../escape.schema.json",
          "#/$defs/missing", "#/$defs/~2invalid"}) {
      auto bad = doc;
      bad["$ref"] = ref;
      write_json(root / fixture_path, bad);
      test::rejects([&] { scv::schema_inventory(manifest, root); },
                    "schema reference rejection");
    }
  }
  void test_duplicate_json_keys_rejected() {
    write_bytes(root / "duplicate.json", "{\"key\":1,\"key\":2}");
    test::rejects([&] { read_json(root / "duplicate.json"); },
                  "duplicate JSON keys");
  }
  void test_checked_in_metadata_has_no_drift() {
    emit(args.root, scv::render(manifest, args.root, true), true);
  }
  void test_checked_in_complete_inventory_has_no_drift() {
    emit(args.root, scv::render(manifest, args.root), true);
  }
  void test_artifact_schema_admissions_match_owner_interface() {
    const auto defs =
        read_json(args.root /
                  "knowledge/scv/schemas/v1/scv-artifact.schema.json")
            .at("$defs");
    Json artifacts = Json::array(), operation_names = Json::array(),
         kinds = Json::array();
    for (const auto &op : manifest.at("operations"))
      if (!op.at("artifact").is_null()) {
        artifacts.push_back(op);
        operation_names.push_back(op.at("name"));
        kinds.push_back(op.at("artifact").at("kind"));
      }
    test::equal(
        defs.at("ImportInput").at("properties").at("operation").at("enum"),
        operation_names, "artifact import operations");
    test::equal(defs.at("Record").at("properties").at("operation").at("enum"),
                operation_names, "artifact record operations");
    test::equal(defs.at("Record").at("properties").at("kind").at("enum"), kinds,
                "artifact record kinds");
    const auto real_catalog = read_json(args.root / catalog_path);
    std::map<std::string, Json> entries;
    for (const auto &entry : real_catalog.at("entries"))
      entries[string(entry.at("protocol"))] = entry;
    const auto reference = [&](const Json &protocol) {
      const auto &entry = entries.at(string(protocol));
      return string(entry.at("file")) + string(entry.at("fragment"));
    };
    test::check(defs.at("Record").at("allOf").size() == artifacts.size() &&
                    defs.at("ImportInput").at("allOf").size() ==
                        artifacts.size(),
                "complete artifact admission branches");
    for (std::size_t i = 0; i < artifacts.size(); ++i) {
      const auto &op = artifacts.at(i);
      const auto &imported = defs.at("ImportInput").at("allOf").at(i);
      const auto &recorded = defs.at("Record").at("allOf").at(i);
      for (const auto &branch : {imported, recorded}) {
        test::equal(
            branch.at("if").at("properties").at("operation").at("const"),
            op.at("name"), "artifact branch identity");
        test::equal(branch.at("then").at("properties").at("input"),
                    {{"$ref", reference(op.at("input_protocol"))}},
                    "artifact input schema");
      }
      test::equal(
          imported.at("then").at("properties").at("result"),
          {{"anyOf",
            Json::array({{{"type", "null"}},
                         {{"$ref", reference(op.at("output_protocol"))}}})}},
          "artifact imported result schema");
      const auto &properties = recorded.at("then").at("properties");
      test::equal(properties.at("artifact"),
                  {{"$ref", reference(op.at("output_protocol"))}},
                  "artifact output schema");
      test::equal(properties.at("kind").at("const"),
                  op.at("artifact").at("kind"), "artifact kind identity");
      Json admitted = Json::array();
      for (const auto &version : scv::versions(manifest)) {
        const auto projected = scv::projection(manifest, version);
        for (const auto &p : projected.at("operations"))
          if (p.at("name") == op.at("name") && !p.at("artifact_kind").is_null())
            admitted.push_back(version);
      }
      test::equal(properties.at("installation")
                      .at("properties")
                      .at("Version")
                      .at("enum"),
                  admitted, "exact artifact release admissions");
    }
  }
};
int main(int argc, char **argv) {
  return main_guard([&] {
    const Arguments args(argc, argv);
    const std::vector<std::pair<std::string, void (InterfaceGeneration::*)()>>
        cases{
            {"test_historical_projection_is_frozen",
             &InterfaceGeneration::test_historical_projection_is_frozen},
            {"test_malformed_history_release_inventory_rejected",
             &InterfaceGeneration::
                 test_malformed_history_release_inventory_rejected},
            {"test_future_domain_does_not_change_old_admission_or_definition",
             &InterfaceGeneration::
                 test_future_domain_does_not_change_old_admission_or_definition},
            {"test_sixth_release_declaration_remains_exact",
             &InterfaceGeneration::
                 test_sixth_release_declaration_remains_exact},
            {"test_seventh_release_declaration_remains_exact",
             &InterfaceGeneration::
                 test_seventh_release_declaration_remains_exact},
            {"test_eighth_release_declaration_remains_exact",
             &InterfaceGeneration::
                 test_eighth_release_declaration_remains_exact},
            {"test_ninth_release_declaration_remains_exact",
             &InterfaceGeneration::
                 test_ninth_release_declaration_remains_exact},
            {"test_complete_generation_is_deterministic",
             &InterfaceGeneration::test_complete_generation_is_deterministic},
            {"test_check_detects_output_drift",
             &InterfaceGeneration::test_check_detects_output_drift},
            {"test_malformed_manifest_rejected",
             &InterfaceGeneration::test_malformed_manifest_rejected},
            {"test_malformed_catalog_and_references_rejected",
             &InterfaceGeneration::
                 test_malformed_catalog_and_references_rejected},
            {"test_duplicate_json_keys_rejected",
             &InterfaceGeneration::test_duplicate_json_keys_rejected},
            {"test_checked_in_metadata_has_no_drift",
             &InterfaceGeneration::test_checked_in_metadata_has_no_drift},
            {"test_checked_in_complete_inventory_has_no_drift",
             &InterfaceGeneration::
                 test_checked_in_complete_inventory_has_no_drift},
            {"test_artifact_schema_admissions_match_owner_interface",
             &InterfaceGeneration::
                 test_artifact_schema_admissions_match_owner_interface}};
    for (const auto &[name, function] : cases) {
      InterfaceGeneration suite(args);
      std::cout << "RUN " << name << '\n';
      (suite.*function)();
      std::cout << "PASS " << name << '\n';
    }
    test::done();
  });
}
