#include "shv_interface.hpp"
#include "test_support.hpp"
using namespace symphony::authoring;
struct InterfaceTests {
  Arguments args;
  std::string module;
  Json original;
  explicit InterfaceTests(const Arguments &selected)
      : args(selected), module("modules/shv-profile-engine"),
        original(read_json(args.root / module / "OWNER-INTERFACE.json")) {}
  void test_frozen_native_descriptor() {
    const auto hist = read_json(args.root / module /
                                "tests/fixtures/interface-history.v1.json")
                          .at("descriptors");
    test::descriptor(args.engine, hist.at("0.2.0-dev"), "0.3.0-dev");
  }
  void test_deterministic_projection() {
    shv::validate_profile(original, args.root);
    const auto output = shv::profile_outputs(original);
    test::check(output == shv::profile_outputs(Json(original)),
                "deterministic projections");
    emit(args.root, output, true);
  }
  void test_rejects_declaration_mutations() {
    const std::vector<std::pair<std::string, std::function<void(Json &)>>>
        mutations{
            {"extra", [](Json &d) { d["extra"] = true; }},
            {"missing", [](Json &d) { d.erase("schemas"); }},
            {"release",
             [](Json &d) { d["releases"]["latest"] = Json::array(); }},
            {"old admission",
             [](Json &d) {
               d["releases"]["0.1.0-dev"].push_back("references_analyze");
             }},
            {"duplicate",
             [](Json &d) {
               const auto op = d["operations"][0];
               d["operations"].push_back(op);
             }},
            {"mutation",
             [](Json &d) { d["operations"][0]["mutability"] = "write"; }},
            {"protocol",
             [](Json &d) {
               d["operations"][0]["output_protocol"] = "invented";
             }},
            {"interaction",
             [](Json &d) {
               d["operations"][0]["administrative_interactions"] =
                   Json::array({"mutate"});
             }},
            {"reader",
             [](Json &d) { d["embedded_kernel_version"] = "latest"; }},
            {"escape", [](Json &d) { d["companions"].push_back("../escape"); }},
            {"duplicate schema",
             [](Json &d) {
               const auto schema = d["schemas"][0];
               d["schemas"].push_back(schema);
             }},
            {"missing schema",
             [](Json &d) { d["schemas"].erase(d["schemas"].size() - 1); }}};
    for (const auto &[name, mutate] : mutations) {
      auto bad = original;
      mutate(bad);
      test::rejects([&] { shv::validate_profile(bad, args.root); }, name);
    }
  }
  void test_rejects_duplicate_json_keys() {
    {
      TempDir temp;
      write_bytes(temp.path / "duplicate.json",
                  "{\"module_id\":\"first\",\"module_id\":\"second\"}");
      test::rejects([&] { read_json(temp.path / "duplicate.json"); },
                    "duplicate JSON keys");
    }
  }
  void test_rejects_schema_escape_and_missing_target() {
    for (const auto *ref : {"../../outside.json", "#/$defs/Absent"}) {
      TempDir temp;
      test::copy(args.root, temp.path,
                 module + "/tests/fixtures/interface-history.v1.json");
      for (const auto *key : {"schemas", "companions"})
        for (const auto &path : original.at(key))
          test::copy(args.root, temp.path, string(path));
      const auto path = temp.path / module / "schemas/v1/profile.schema.json";
      auto doc = read_json(path);
      doc["$ref"] = ref;
      write_json(path, doc);
      test::rejects([&] { shv::validate_profile(original, temp.path); },
                    "schema reference");
    }
  }
};
int main(int argc, char **argv) {
  return main_guard([&] {
    const Arguments args(argc, argv);
    {
      InterfaceTests suite(args);
      std::cout << "RUN test_frozen_native_descriptor\n";
      suite.test_frozen_native_descriptor();
      std::cout << "PASS test_frozen_native_descriptor\n";
    }
    {
      InterfaceTests suite(args);
      std::cout << "RUN test_deterministic_projection\n";
      suite.test_deterministic_projection();
      std::cout << "PASS test_deterministic_projection\n";
    }
    {
      InterfaceTests suite(args);
      std::cout << "RUN test_rejects_declaration_mutations\n";
      suite.test_rejects_declaration_mutations();
      std::cout << "PASS test_rejects_declaration_mutations\n";
    }
    {
      InterfaceTests suite(args);
      std::cout << "RUN test_rejects_duplicate_json_keys\n";
      suite.test_rejects_duplicate_json_keys();
      std::cout << "PASS test_rejects_duplicate_json_keys\n";
    }
    {
      InterfaceTests suite(args);
      std::cout << "RUN test_rejects_schema_escape_and_missing_target\n";
      suite.test_rejects_schema_escape_and_missing_target();
      std::cout << "PASS test_rejects_schema_escape_and_missing_target\n";
    }
    test::done();
  });
}
