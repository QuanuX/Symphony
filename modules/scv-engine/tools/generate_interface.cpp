#include "scv_interface.hpp"
using namespace symphony::authoring;
int main(int argc, char **argv) {
  return main_guard([&] {
    Arguments args(argc, argv);
    if (args.help) {
      std::cout << "SCV interface authoring: [--root ROOT] [--check] "
                   "[--metadata-only]\n";
      return;
    }
    const auto output =
        scv::render(read_json(args.root / scv::manifest_path, 1048576),
                    args.root, args.metadata_only);
    emit(args.root, output, args.check);
    Json files = Json::array();
    for (const auto &[name, body] : output) {
      static_cast<void>(body);
      files.push_back(name);
    }
    std::cout << Json{{"complete", !args.metadata_only},
                      {"files", files},
                      {"result", args.check ? "checked" : "generated"}}
                     .dump()
              << '\n';
  });
}
