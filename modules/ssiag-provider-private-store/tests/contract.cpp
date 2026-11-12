#include "native_test.hpp"
#include <symphony/ssiag/private_channel.hpp>
#include <sys/wait.h>
using namespace native_test;
namespace n = symphony::ssiag::native;
int seed(const fs::path &binary, const fs::path &root, const Json &input) {
  int secret[2], control[2];
  NT_REQUIRE(::pipe(secret) == 0 && ::pipe(control) == 0);
  auto pid = ::fork();
  NT_REQUIRE(pid >= 0);
  if (pid == 0) {
    ::dup2(control[0], 0);
    ::dup2(secret[0], 3);
    n::close_others(3);
    auto nil = ::open("/dev/null", O_RDWR);
    ::dup2(nil, 1);
    ::dup2(nil, 2);
    if (nil > 3)
      ::close(nil);
    ::execl(binary.c_str(), binary.c_str(), "--seed", root.c_str(), nullptr);
    _exit(127);
  }
  ::close(secret[0]);
  ::close(control[0]);
  n::write_all(secret[1], "SECRET-SENTINEL");
  ::close(secret[1]);
  n::write_all(control[1], input.dump());
  ::close(control[1]);
  int status{};
  ::waitpid(pid, &status, 0);
  return WIFEXITED(status) ? WEXITSTATUS(status) : -1;
}
int main(int argc, char **argv) {
  try {
    NT_REQUIRE(argc == 2);
    auto binary = fs::canonical(argv[1]);
    TempDir temp("native-store-");
    auto root = temp.path / "vault";
    Json config{{"protocol", "symphony.ssiag.private-store-config.v1"},
                {"tops_id", "11111111-1111-4111-8111-111111111111"},
                {"ssiag_socket", "/private/ssiag.sock"},
                {"ssiag_uid", std::to_string(::geteuid())},
                {"ssiag_digest", "sha256:" + std::string(64, 'a')},
                {"consumer_digest", "sha256:" + std::string(64, 'b')},
                {"toolchain_digest", "sha256:" + std::string(64, 'c')}};
    auto configure = [&](Json expected) {
      return run({binary.string(), "--configure", root.string()},
                 Json{{"expected_digest", expected}, {"configuration", config}}
                     .dump());
    };
    auto first = configure(nullptr);
    NT_REQUIRE(first.returncode == 0);
    auto d = Json::parse(first.stdout_text).at("config_digest");
    NT_REQUIRE(d == n::seal(config, "symphony.ssiag.private-store-config.v1") &&
               configure(nullptr).returncode == 5 &&
               configure(d).returncode == 0);
    Json metadata{{"reference", "ssiag-native:api"},
                  {"purpose", "cloudflare.api-token"},
                  {"account_id", std::string(32, '0')},
                  {"zone_id", nullptr},
                  {"resource_id", nullptr},
                  {"generation", "1"},
                  {"expires_unix_ms", std::to_string(n::now() + 3600000)}};
    NT_REQUIRE(seed(binary, root, metadata) == 0 &&
               seed(binary, root, metadata) == 5);
    NT_REQUIRE(n::read(root / "credentials/api/value", 16384, true) ==
               "SECRET-SENTINEL");
    auto command = [&](const std::string &op, const std::string &generation) {
      return run(
          {binary.string(), op, root.string()},
          Json{{"reference", "ssiag-native:api"}, {"generation", generation}}
              .dump());
    };
    auto inspect = command("--inspect", "1");
    NT_REQUIRE(inspect.returncode == 0 &&
               inspect.stdout_text.find("SENTINEL") == std::string::npos &&
               Json::parse(inspect.stdout_text) == metadata);
    auto revoke = command("--revoke", "1");
    NT_REQUIRE(revoke.returncode == 0 &&
               Json::parse(revoke.stdout_text).at("generation") == "2" &&
               command("--inspect", "1").returncode == 5);
    NT_REQUIRE(Json::parse(command("--inspect", "2").stdout_text)
                   .at("expires_unix_ms") == "0");
    fs::create_directory_symlink(root, temp.path / "alias");
    NT_REQUIRE(
        run({binary.string(), "--inspect", (temp.path / "alias").string()},
            Json{{"reference", "ssiag-native:api"}, {"generation", "2"}}.dump())
            .returncode == 5);
    metadata["reference"] = "ssiag-native:bad";
    metadata["account_id"] = "SECRET-SENTINEL";
    NT_REQUIRE(seed(binary, root, metadata) == 5 &&
               !fs::exists(root / "credentials/bad"));
    std::cout << "Native store configuration CAS, private FD import, duplicate "
                 "refusal, metadata-only inspect, generation revocation and "
                 "path/scope rejection passed\n";
    return 0;
  } catch (const std::exception &e) {
    std::cerr << e.what() << '\n';
    return 1;
  }
}
