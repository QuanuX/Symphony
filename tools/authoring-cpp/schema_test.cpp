#include "schemas.hpp"
#include "test_support.hpp"
using namespace symphony::authoring;
int main(int argc, char **argv) {
  return main_guard([&] {
    Arguments args(argc, argv);
    const std::vector<
        std::pair<std::string, std::function<Outputs(const fs::path &)>>>
        builders{{"kernel", schema::kernel},
                 {"profile", schema::profile},
                 {"source", schema::source},
                 {"activation", schema::activation},
                 {"refresh", schema::refresh}};
    for (const auto &[name, builder] : builders) {
      const auto output = builder(args.root);
      test::check(output == builder(args.root), name + " deterministic output");
      for (const auto &[path, body] : output) {
        const auto previous = read_bytes(args.root / path);
        test::equal(Json::parse(body), Json::parse(previous),
                    name +
                        " complete schema/template semantic parity: " + path);
        test::check(body == previous, name + " exact byte parity: " + path);
      }
      TempDir temp;
      emit(temp.path, output, false);
      emit(temp.path, output, true);
      write_bytes(temp.path / output.front().first, "drift");
      test::rejects([&] { emit(temp.path, output, true); },
                    name + " drift detected");
    }
    // A mutation to the explicitly imported generic contract must be projected,
    // proving these builders consume original-owner definitions instead of
    // copies.
    {
      TempDir temp;
      const std::vector<std::string> files{
          "modules/shv-graph-adapter/schemas/v1/graph-adapter.schema.json",
          "modules/shv-pdf-adapter/schemas/v1/pdf.schema.json",
          "modules/shv-engine/schemas/v1/shv.schema.json"};
      for (const auto &file : files)
        test::copy(args.root, temp.path, file);
      auto graph = read_json(temp.path / files[0]);
      graph["$defs"]["Node"]["$comment"] = "isolated owner mutation";
      write_json(temp.path / files[0], graph);
      const auto projected =
          Json::parse(schema::kernel(temp.path).front().second);
      test::equal(projected.at("$defs").at("Node"),
                  graph.at("$defs").at("Node"),
                  "kernel reads current generic owner definition");
      const auto source = Json::parse(schema::source(temp.path).front().second);
      test::equal(source.at("$defs").at("Node"), graph.at("$defs").at("Node"),
                  "source reads current generic owner definition");
    }
    {
      TempDir temp;
      const auto imported = temp.path / "imported.schema.json";
      for (const auto &invalid :
           Json::array({nullptr, 12, true, Json::array(), Json::object()})) {
        write_json(
            imported,
            {{"$defs",
              {{"Nested", {{"allOf", Json::array({{{"$ref", invalid}}})}}}}}});
        test::rejects([&] { schema::scoped(imported, "Imported"); },
                      "imported non-string schema reference rejected",
                      "schema reference must be a string");
      }
    }
    test::done();
  });
}
