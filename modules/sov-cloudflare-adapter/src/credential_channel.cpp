#include "execution.hpp"
#include <signal.h>
#include <sys/wait.h>
namespace cloudflare {
namespace n = symphony::ssiag::native;
Json execution_binding(const Json &e) {
  n::fields(e, {"provider_prefix", "provider_version", "provider_receipt",
                "vault", "config_digest", "credential_generation", "result_ref",
                "max_pages"});
  n::need(e.at("provider_version") == "0.1.0-dev" &&
          n::tagged(e.at("provider_receipt")) &&
          n::tagged(e.at("config_digest")));
  auto binary = n::installation(e.at("provider_prefix").get<std::string>(),
                                "ssiag-provider-private-store", "0.1.0-dev",
                                e.at("provider_receipt"));
  auto root = fs::path(e.at("vault").get<std::string>());
  n::private_dir(root);
  auto config = n::parse(n::read(root / "config.json", 32768, true));
  n::fields(config, {"protocol", "tops_id", "ssiag_socket", "ssiag_uid",
                     "ssiag_digest", "consumer_digest", "toolchain_digest"});
  n::need(e.at("config_digest") ==
          n::seal(config, "symphony.ssiag.private-store-config.v1"));
  return Json{
      {"executable", binary.string()},
      {"executable_digest", n::hash(n::read(binary, 512 * 1024 * 1024))},
      {"config", config}};
}
struct CredentialSession::Impl {
  n::FD channel;
  pid_t pid = -1;
  std::string request_hash;
  bool finished = false;
  bool finish_attempted = false;
  Impl(int fd, pid_t child, std::string hash)
      : channel(fd), pid(child), request_hash(std::move(hash)) {}
  ~Impl() {
    if (pid > 0) {
      int status{};
      if (!finished)
        ::kill(pid, SIGKILL);
      while (::waitpid(pid, &status, 0) < 0 && errno == EINTR) {
      }
    }
  }
};
CredentialSession::CredentialSession(const Json &e, const Json &intent) {
  auto binding = execution_binding(e);
  n::need(binding.at("config").at("consumer_digest") ==
              intent.at("adapter_digest") &&
          binding.at("config").at("toolchain_digest") ==
              intent.at("toolchain_digest"));
  int pair[2];
  n::need(::socketpair(AF_UNIX, SOCK_STREAM, 0, pair) == 0);
  n::FD parent(pair[0]), child(pair[1]);
  n::timeout(parent.n, 60);
  std::vector<std::string> args{binding.at("executable").get<std::string>(),
                                "--session", e.at("vault").get<std::string>(),
                                e.at("config_digest").get<std::string>()};
  std::vector<char *> av;
  for (auto &a : args)
    av.push_back(a.data());
  av.push_back(nullptr);
  auto pid = ::fork();
  n::need(pid >= 0);
  if (pid == 0) {
    if (::dup2(child.n, 3) < 0)
      _exit(126);
    n::close_others(3);
    int nil = ::open("/dev/null", O_RDWR);
    ::dup2(nil, 0);
    ::dup2(nil, 1);
    ::dup2(nil, 2);
    if (nil > 3)
      ::close(nil);
    char path[] = "PATH=/usr/bin:/bin";
    char *env[]{path, nullptr};
    ::execve(av[0], av.data(), env);
    _exit(127);
  }
  auto hash = n::seal(intent, "symphony.ssiag.cloudflare-use.v1");
  impl = std::make_unique<Impl>(parent.n, pid, hash);
  parent.n = -1;
  ::close(child.n);
  child.n = -1;
  n::frame(impl->channel.n,
           Json{{"intent", intent},
                {"credential_generation", e.at("credential_generation")}}
               .dump());
  grant = n::parse(n::frame(impl->channel.n));
  n::need(grant.at("request_digest") == hash &&
          grant.at("uid") == std::to_string(::geteuid()) &&
          grant.at("receipt").at("disposition") == "committed" &&
          std::stoll(grant.at("deadline_unix_ms").get<std::string>()) >
              n::now());
  secret.bytes = n::frame(impl->channel.n, 16384);
  n::need(std::all_of(secret.bytes.begin(), secret.bytes.end(),
                      [](unsigned char c) { return c >= 33 && c < 127; }));
}
CredentialSession::~CredentialSession() = default;
Json CredentialSession::finish(const std::string &outcome,
                               const std::string *sensitive) {
  n::need(impl && !impl->finish_attempted);
  impl->finish_attempted = true;
  n::wipe(secret.bytes);
  n::frame(impl->channel.n, Json{{"request_digest", impl->request_hash},
                                 {"outcome", outcome},
                                 {"sensitive_result", sensitive != nullptr}}
                                .dump());
  if (sensitive)
    n::frame(impl->channel.n, *sensitive);
  auto response = n::parse(n::frame(impl->channel.n));
  n::need(
      response.at("completion").at("request_digest") == impl->request_hash &&
      response.at("completion").at("outcome") == outcome &&
      response.at("completion").at("receipt").at("disposition") == "committed");
  impl->finished = true;
  return response;
}
} // namespace cloudflare
