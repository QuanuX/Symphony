// Inert managed connector fixture. No network and no credential output.
#include <cstdlib>
#include <string>
#include <unistd.h>
int main(int argc, char **argv) {
  if (argc < 2 || std::string(argv[1]) != "tunnel")
    return 2;
  auto token = std::getenv("TUNNEL_TOKEN");
  if (!token || std::string(token) != "TUNNEL-SENTINEL" || ::getuid() == 0)
    return 3;
  for (;;)
    ::pause();
}
