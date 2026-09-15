#include "shv_interface.hpp"
using namespace symphony::authoring;
int main(int argc, char **argv) {
  return main_guard([&] {
    Arguments args(argc, argv);
    if (args.help) {
      std::cout << "SHV profile interface authoring: [--root ROOT] [--check]\n";
      return;
    }
    const auto d = read_json(args.root /
                             "modules/shv-profile-engine/OWNER-INTERFACE.json");
    shv::validate_profile(d, args.root);
    emit(args.root, shv::profile_outputs(d), args.check);
    std::cout << (args.check ? "profile interface: checked\n"
                             : "profile interface: generated\n");
  });
}
