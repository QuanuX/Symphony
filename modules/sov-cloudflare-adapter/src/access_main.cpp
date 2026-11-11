#include "access.hpp"
#include "access_schema.hpp"
#include <iostream>
#include <sys/resource.h>
int main(int argc, char **argv) {
  rlimit limit{0, 0};
  if (::setrlimit(RLIMIT_CORE, &limit) != 0)
    return 2;
  try {
    if (argc == 2 && std::string(argv[1]) == "--describe") {
      std::cout << access_schema;
      return 0;
    }
    auto r = cloudflare::access::runtime(
        symphony::ssiag::native::executable(::getpid()));
    if (argc == 2 && std::string(argv[1]) == "--session") {
      cloudflare::access::parent(r, false);
      auto raw = symphony::knowledge::engine::read_bounded(std::cin, 65536);
      auto result = cloudflare::access::session(
          r, symphony::ssiag::native::parse(raw, 65536));
      std::cout << result.dump() << '\n';
      return result.at("completed") == true ? 0 : 5;
    }
    if (argc != 7 || std::string(argv[1]) != "--application" ||
        std::string(argv[3]) != "--session-directory" ||
        std::string(argv[5]) != "--session-ref")
      return 2;
    cloudflare::access::parent(r, true);
    return cloudflare::access::connect(r, argv[2], argv[4], argv[6]);
  } catch (...) {
    return 5;
  } // Never forward provider diagnostics or session bytes.
}
