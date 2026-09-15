#include "installed_support.hpp"
using namespace native_test;
void test_installed_profile_source_binding(const Arguments &args) {
  Evidence evidence(args);
  auto prefix = fs::canonical(args.require("prefix"));
  auto receipt = prefix / "share/symphony/receipts/shv-profile-engine" /
                 args.require("version") / "install-receipt.json";
  auto installation = evidence.installation(
      "shv-profile-engine", "symphony-shv-profile", {"profile"}, receipt);
  auto call = [&](const std::string &op, const Json &payload,
                  const std::string &error = "") {
    return evidence.call(installation, op, payload, error);
  };
  TempDir temporary("shv-profile-installed-");
  auto root = temporary.path, source = root / "source.html";
  std::string content =
      "<div id=\"overview\"><h1>Synthetic GPU</h1></div><article "
      "id=\"spec\"><dl><dt>Value</dt><dd>16</dd><dt>End</dt><dd>done</dd></"
      "dl></article>";
  write(source, content);
  Json mapping = {{"id", "gpu"},
                  {"manufacturer", "Synthetic fixture"},
                  {"model", "Synthetic GPU"},
                  {"hardware_class", "gpu"},
                  {"source_id", "source"},
                  {"heading_section", "div#overview"},
                  {"field_section", "article#spec"},
                  {"fields", Json::array({{{"predicate", "memory"},
                                           {"value_type", "integer"},
                                           {"qualifier", "unit=caller"},
                                           {"label", "Value"},
                                           {"next_label", "End"}}})}};
  auto profile = call(
      "profile_compile",
      {{"id", "caller"},
       {"revision", "v1"},
       {"hardware_class", "gpu"},
       {"metrics", Json::array({{{"predicate", "memory"},
                                 {"value_type", "integer"},
                                 {"qualifier", "unit=caller"},
                                 {"required", true},
                                 {"description", "Synthetic caller metric"},
                                 {"extensions", Json::object()}}})},
       {"extensions", Json::object()}});
  Json definition = {
      {"id", "selected"},
      {"revision", "v1"},
      {"kernel_version", "0.3.0-dev"},
      {"coverage", seal({{"protocol", "symphony.shv.coverage-profile.v1"},
                         {"as_of", "2026-09-15"},
                         {"selector", {{"op", "all"}}}})},
      {"profiles", {profile}},
      {"sources", Json::array({{{"id", "source"},
                                {"path", source.filename().string()},
                                {"bytes", content.size()},
                                {"digest", digest(content)},
                                {"format", "html"}}})},
      {"mapping", {mapping}},
      {"locators", Json::array()},
      {"extensions", Json::object()}};
  auto universe = call("universe_build", definition);
  Json payload = {
      {"universe", universe},
      {"bindings",
       {{"source_root", root.string()}, {"decoders", Json::object()}}}};
  auto bound = call("universe_bind", payload);
  NT_REQUIRE(bound["catalogue"]["subjects"][0]["assertions"][0]["value"] == 16);
  NT_REQUIRE(bound["coverage"]["counts"] ==
             Json({{"included", 1}, {"excluded", 0}, {"unresolved", 0}}));
  NT_REQUIRE(bound["reader"]["version"] == "0.3.0-dev" &&
             bound["canonical_apply_enabled"] == false);
  write(source, replace(content, ">16<", ">32<"));
  call("universe_bind", payload, "shv.invalid");
  fs::remove(source);
  auto target = root / "other.html";
  write(target, content);
  fs::create_symlink(target, source);
  call("universe_bind", payload, "path.file_unreadable");
  call("universe_build", updated(definition, {{"kernel_version", "latest"}}),
       "shv.invalid");
  evidence.finish("test_installed_profile_source_binding");
}
int main(int argc, char **argv) {
  return test_main(
      [&] { test_installed_profile_source_binding(Arguments(argc, argv)); });
}
