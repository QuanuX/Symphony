#include "authoring.hpp"
#include "test_support.hpp"
using namespace symphony::authoring;
int main() {
  return main_guard([&] {
    const Json closed{{"type", "object"},
                      {"additionalProperties", false},
                      {"properties", {{"kind", {{"const", "selected"}}}}},
                      {"required", Json::array({"kind"})}};
    Json resource{{"$id", "urn:symphony:embedded:result:v1"},
                  {"$defs", {{"attempt", closed}}},
                  {"oneOf", Json::array({Json{{"$ref", "#/$defs/attempt"}}})}};
    Json document{{"$defs", {{"Outer", {{"type", "string"}}}}},
                  {"cli_results", {{"result", resource}}}};
    local_refs(document);
    test::check(true, "embedded ID scopes closed tagged result references");
    auto missing = document;
    missing["cli_results"]["result"]["$defs"].erase("attempt");
    missing["$defs"]["attempt"] = closed;
    test::rejects([&] { local_refs(missing); },
                  "embedded missing reference cannot fall back to outer scope");
    for (const auto *invalid :
         {"../../outside.json", "urn:external:attempt", "#/$defs/Absent"}) {
      auto bad = document;
      bad["cli_results"]["result"]["oneOf"][0]["$ref"] = invalid;
      test::rejects([&] { local_refs(bad); },
                    "unsupported or missing embedded reference");
    }
    auto unscoped = document;
    unscoped["cli_results"]["result"].erase("$id");
    unscoped["cli_results"]["result"]["oneOf"][0]["$ref"] = "#/$defs/Outer";
    local_refs(unscoped);
    test::check(true,
                "nested defs alone do not change JSON Schema resource base");
    auto schema_property = document;
    schema_property["properties"]["$id"] = {{"type", "string"}};
    schema_property["$defs"]["$ref"] = {{"type", "string"}};
    local_refs(schema_property);
    test::check(
        true,
        "schema property and definition names are not mistaken for keywords");
    for (const auto &invalid : Json::array({nullptr, false, 42, ""})) {
      auto bad = document;
      bad["cli_results"]["result"]["$id"] = invalid;
      test::rejects([&] { local_refs(bad); }, "malformed resource ID rejected");
    }
    local_refs(read_json(fs::path(SYMPHONY_AUTHORING_ROOT) /
                         "modules/snv-engine/schemas/v1/admin.schema.json"));
    test::check(true, "actual SNV closed result and CLI resource references resolve");
    test::done();
  });
}
