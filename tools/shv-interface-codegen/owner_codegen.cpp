#include "shv_interface.hpp"
using namespace symphony::authoring;
int main(int argc, char **argv) {
  return main_guard([&] {
    Arguments args(argc, argv);
    if (args.help) {
      std::cout << "SHV owner authoring: --registration PATH [--root ROOT] "
                   "[--check]\n";
      return;
    }
    shv::owner_main(args);
  });
}
