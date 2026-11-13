// Inert managed connector fixture. No network and no credential output.
#include <cstdlib>
#include <fcntl.h>
#include <string>
#include <unistd.h>
int main(int argc, char **argv) {
  if (argc < 2 || std::string(argv[1]) != "tunnel")
    return 2;
  auto token = std::getenv("TUNNEL_TOKEN");
  if (!token || std::string(token) != "TUNNEL-SENTINEL" || ::getuid() == 0)
    return 3;
  // A retained nonsecret effect proves whether a child was actually launched.
  int marker = ::open("child-launched", O_WRONLY | O_CREAT | O_EXCL, 0600);
  if (marker < 0)
    return 4;
  ::close(marker);
  if (::access("test-early-exit", F_OK) == 0) {
    int exited = ::open("child-exited", O_WRONLY | O_CREAT | O_EXCL, 0600);
    if (exited < 0)
      return 5;
    ::close(exited);
    return 7;
  }
  for (;;)
    ::pause();
}
