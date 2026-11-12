#include "access.hpp"
#include <iostream>
#include <thread>
using namespace cloudflare::access;
int main(int argc, char **argv) {
  try {
    n::user();
    n::need(argc >= 3 && std::string(argv[1]) == "access");
    fs::path home = ::getenv("HOME");
    auto root = home.parent_path();
    auto cfg = n::parse(n::read(root / "fixture.json", 65536, true));
    n::need(!::getenv("CF_ACCESS_TOKEN") && !::getenv("HTTPS_PROXY"));
    if (std::string(argv[2]) == "login") {
      n::need(argc == 5 && std::string(argv[3]) == "--quiet" &&
              std::string(argv[4]) == "https://access.example.invalid");
      std::cout << "SECRET-PROVIDER-STDOUT";
      std::cerr << "SECRET-PROVIDER-STDERR";
      if (cfg.value("delay", false))
        std::this_thread::sleep_for(std::chrono::seconds(10));
      if (cfg.value("fail", false))
        return 7;
      n::put(home / ".cloudflared" /
                 ("access.example.invalid-" +
                  cfg.at("audience").get<std::string>() + "-token"),
             cfg.at("token"));
      n::put(home / ".cloudflared/team-org-token", "SECRET-ORG");
    } else {
      n::need(argc == 7 && std::string(argv[2]) == "ssh" &&
              std::string(argv[3]) == "--hostname" &&
              std::string(argv[4]) == "access.example.invalid" &&
              std::string(argv[5]) == "--loglevel" &&
              std::string(argv[6]) == "error");
      auto cache = home / ".cloudflared";
      n::need(!fs::exists(cache / "team-org-token"));
      auto token = cache / ("access.example.invalid-" +
                            cfg.at("audience").get<std::string>() + "-token");
      n::need(n::read(token, 32768, true) == cfg.at("token").get<std::string>());
      n::FD f(::open((token.string() + ".lock").c_str(),
                     O_WRONLY | O_CREAT | O_EXCL, 0600));
      n::need(f.n < 0 && errno == EACCES);
      n::need(::unlink(token.c_str()) < 0 && errno == EACCES);
      if (cfg.value("deny", false))
        return 9;
    }
    return 0;
  } catch (...) {
    return 99;
  }
}
