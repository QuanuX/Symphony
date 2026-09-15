#include "schemas.hpp"
using namespace symphony::authoring;
int main(int argc, char **argv) {
  return main_guard([&] {
    Arguments args(argc, argv);
    if (args.help) {
      std::cout << "Native schema authoring: [--root ROOT] [--check]\n";
      return;
    }
    schema::generate(args, schema::activation);
  });
}
