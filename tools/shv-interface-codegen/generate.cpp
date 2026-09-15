#include "shv_interface.hpp"
using namespace symphony::authoring;
int main(int argc, char **argv) {
  return main_guard([&] {
    Arguments args(argc, argv);
    if (args.help) {
      std::cout << "SHV interface authoring: (--owner MODULE | --registration "
                   "PATH) [--root ROOT] [--check]\n";
      return;
    }
    if (!args.registration.empty())
      shv::owner_main(args);
    else {
      require(!args.owner.empty(), "--owner is required");
      const auto d = shv::load_v1(args.root, args.owner);
      emit(args.root, shv::v1_outputs(d), args.check);
      std::cout << args.owner << (args.check ? " checked\n" : " generated\n");
    }
  });
}
