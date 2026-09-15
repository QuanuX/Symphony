#include "shv_interface.hpp"
#include "test_support.hpp"
using namespace symphony::authoring;
struct InterfaceTests {
  Arguments args;
  std::string module;
  Json original;
  explicit InterfaceTests(const Arguments &selected)
      : args(selected), module("modules/" + args.owner),
        original(read_json(args.root / module / "OWNER-INTERFACE.json")) {}
  void test_frozen_native_parity() {
    const auto hist = read_json(args.root / module /
                                "tests/fixtures/interface-history.v1.json")
                          .at("descriptors");
    test::descriptor(args.engine, shv::latest(hist),
                     original.at("current_version"));
  }
  void test_deterministic_projection_and_drift() {
    shv::validate_v1(original, args.root);
    const auto output = shv::v1_outputs(original);
    test::check(output == shv::v1_outputs(Json(original)),
                "deterministic projections");
    emit(args.root, output, true);
    {
      TempDir temp;
      emit(temp.path, output, false);
      write_bytes(temp.path / output.front().first, "changed");
      test::rejects([&] { emit(temp.path, output, true); }, "generated drift");
    }
  }
  void test_declaration_rejections() {
    const auto old = original.at("releases").begin().key();
    const std::vector<std::pair<std::string, std::function<void(Json &)>>>
        mutations{
            {"extra", [](Json &d) { d["extra"] = true; }},
            {"missing", [](Json &d) { d.erase("schemas"); }},
            {"owner", [](Json &d) { d["module_id"] = "unknown"; }},
            {"namespace", [](Json &d) { d["namespace"] = "foreign"; }},
            {"release",
             [](Json &d) { d["releases"]["latest"] = Json::array(); }},
            {"old admission",
             [&](Json &d) { d["releases"][old].push_back("unknown"); }},
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
             [](Json &d) {
               d["embedded_dependencies"].push_back(
                   {{"engine_id", "foreign"}, {"version", "latest"}});
             }},
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
      test::rejects([&] { shv::validate_v1(bad, args.root); }, name);
    }
  }
  void test_selected_owner_alias_rejected() {
    TempDir temp;
    const std::string alias = "alias-owner";
    write_json(temp.path / "modules" / alias / "OWNER-INTERFACE.json",
               original);
    test::rejects([&] { shv::load_v1(temp.path, alias); },
                  "frozen owner alias rejected",
                  "selected owner differs from declaration");
    test::equal(shv::load_v1(args.root, args.owner), original,
                "exact selected frozen owner remains accepted");
  }
  void test_duplicate_keys_and_schema_refs() {
    {
      TempDir temp;
      write_bytes(temp.path / "duplicate.json", "{\"a\":1,\"a\":2}");
      test::rejects([&] { read_json(temp.path / "duplicate.json"); },
                    "duplicate keys");
    }
    for (const auto *ref : {"../../outside.json", "#/$defs/Absent"}) {
      TempDir temp;
      test::copy(args.root, temp.path,
                 module + "/tests/fixtures/interface-history.v1.json");
      for (const auto *key : {"schemas", "companions"})
        for (const auto &path : original.at(key))
          test::copy(args.root, temp.path, string(path));
      const auto path = temp.path / string(original.at("schemas").at(0));
      auto doc = read_json(path);
      doc["$ref"] = ref;
      write_json(path, doc);
      test::rejects([&] { shv::validate_v1(original, temp.path); },
                    "schema reference");
    }
  }
};
int main(int argc, char **argv) {
  return main_guard([&] {
    const Arguments args(argc, argv);
    {
      InterfaceTests suite(args);
      std::cout << "RUN test_frozen_native_parity\n";
      suite.test_frozen_native_parity();
      std::cout << "PASS test_frozen_native_parity\n";
    }
    {
      InterfaceTests suite(args);
      std::cout << "RUN test_deterministic_projection_and_drift\n";
      suite.test_deterministic_projection_and_drift();
      std::cout << "PASS test_deterministic_projection_and_drift\n";
    }
    {
      InterfaceTests suite(args);
      std::cout << "RUN test_declaration_rejections\n";
      suite.test_declaration_rejections();
      std::cout << "PASS test_declaration_rejections\n";
    }
    {
      InterfaceTests suite(args);
      std::cout << "RUN test_selected_owner_alias_rejected\n";
      suite.test_selected_owner_alias_rejected();
      std::cout << "PASS test_selected_owner_alias_rejected\n";
    }
    {
      InterfaceTests suite(args);
      std::cout << "RUN test_duplicate_keys_and_schema_refs\n";
      suite.test_duplicate_keys_and_schema_refs();
      std::cout << "PASS test_duplicate_keys_and_schema_refs\n";
    }
    test::done();
  });
}
