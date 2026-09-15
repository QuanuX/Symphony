#include "shv_interface.hpp"
#include "test_support.hpp"
using namespace symphony::authoring;
struct InterfaceTests {
  Arguments args;
  Json r, d;
  explicit InterfaceTests(const Arguments &selected) : args(selected) {
    require(!args.registration.empty(), "--registration is required");
    const auto loaded = shv::load(args.root, args.registration);
    r = loaded.first;
    d = loaded.second;
  }
  void test_frozen_descriptor() {
    const auto hist =
        read_json(args.root / string(r.at("history"))).at("descriptors");
    test::descriptor(args.engine, shv::latest(hist), d.at("current_version"));
  }
  void test_generated_parity() {
    emit(args.root, shv::owner_outputs(r, d), true, true);
    test::check(shv::owner_outputs(r, d) == shv::owner_outputs(r, Json(d)),
                "deterministic projections");
  }
  void test_future_owner_and_collision() {
    {
      TempDir temp;
      auto future = d;
      future["module_id"] = "user-sensor-engine";
      future["engine_id"] = "user-sensor";
      future["namespace"] = "user::sensor";
      future["current_version"] = "1.0.0";
      future["releases"] = {{"1.0.0", shv::names(d.at("operations"))}};
      for (const auto *key : {"schemas", "companions", "embedded_dependencies"})
        future[key] = Json::array();
      write_json(temp.path / "declaration.json", future);
      write_bytes(temp.path / "history.json", "{\"descriptors\":{}}");
      auto registration = r;
      registration["declaration"] = "declaration.json";
      registration["history"] = "history.json";
      registration["history_digest"] =
          digest_bytes(read_bytes(temp.path / "history.json"));
      registration["go_title"] = "UserSensor";
      registration["go_stem"] = "user_sensor";
      registration["cpp_output"] = "out/sensor.hpp";
      registration["go_output"] = "out/sensor.go";
      registration["cmake_output"] = "out/sensor.cmake";
      registration["embedded_compatibility_version"] = nullptr;
      registration["embedded_macro"] = "USER_SENSOR_PREVIOUS";
      write_json(temp.path / "registration.json", registration);
      const auto [reg, decl] =
          shv::load(temp.path, temp.path / "registration.json");
      const auto outputs = shv::owner_outputs(reg, decl);
      emit(temp.path, outputs, false, true);
      emit(temp.path, outputs, true, true);
      auto other = decl;
      other["module_id"] = "other-sensor-engine";
      other["engine_id"] = "other-sensor";
      test::rejects(
          [&] { emit(temp.path, shv::owner_outputs(reg, other), false, true); },
          "other owner collision");
      for (const auto &[path, body] : outputs)
        test::check(read_bytes(temp.path / path) == body,
                    "preflight leaves prior output unchanged");
      write_bytes(temp.path / "out/sensor.go", "caller-owned");
      test::rejects([&] { emit(temp.path, outputs, false, true); },
                    "caller output collision");
      test::check(read_bytes(temp.path / "out/sensor.go") == "caller-owned",
                  "caller bytes unchanged");
    }
  }
  void test_registration_rejections() {
    const std::vector<std::pair<std::string, std::function<void(Json &)>>>
        mutations{
            {"path", [](Json &x) { x["cpp_output"] = "../escape"; }},
            {"symbol", [](Json &x) { x["go_title"] = "bad;code"; }},
            {"collision", [](Json &x) { x["cpp_output"] = x["declaration"]; }},
            {"history",
             [](Json &x) {
               x["history_digest"] = "sha256:" + std::string(64, '0');
             }},
            {"embedded",
             [](Json &x) { x["embedded_compatibility_version"] = "latest"; }}};
    for (const auto &[name, mutate] : mutations) {
      TempDir temp;
      auto bad = r;
      mutate(bad);
      write_json(temp.path / "reg.json", bad);
      test::rejects([&] { shv::load(args.root, temp.path / "reg.json"); },
                    name);
    }
  }
};
int main(int argc, char **argv) {
  return main_guard([&] {
    const Arguments args(argc, argv);
    {
      InterfaceTests suite(args);
      std::cout << "RUN test_frozen_descriptor\n";
      suite.test_frozen_descriptor();
      std::cout << "PASS test_frozen_descriptor\n";
    }
    {
      InterfaceTests suite(args);
      std::cout << "RUN test_generated_parity\n";
      suite.test_generated_parity();
      std::cout << "PASS test_generated_parity\n";
    }
    {
      InterfaceTests suite(args);
      std::cout << "RUN test_future_owner_and_collision\n";
      suite.test_future_owner_and_collision();
      std::cout << "PASS test_future_owner_and_collision\n";
    }
    {
      InterfaceTests suite(args);
      std::cout << "RUN test_registration_rejections\n";
      suite.test_registration_rejections();
      std::cout << "PASS test_registration_rejections\n";
    }
    test::done();
  });
}
