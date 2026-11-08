#pragma once
#include "adapter.hpp"
#include <functional>
#include <symphony/ssiag/private_channel.hpp>
namespace cloudflare {
struct HTTPResult {
  bool attempted = false;
  int status = 0;
  Json body = nullptr;
  bool clean_exit = false;
};
// Injection is a native test/composition seam only; installed control frames do
// not select a runner, API endpoint, environment, hook or alternate command.
using HTTPRunner =
    std::function<HTTPResult(const Json &, const std::string &,
                             const fs::path &, const fs::path &, std::int64_t)>;
struct RuntimeHooks {
  HTTPRunner http;
  bool fixture_toolchain = false;
};
void resource_barrier(const fs::path &);
Json execution_binding(const Json &);
Json execute_resource(const Json &, const Json &, const fs::path &, Json &,
                      bool &, RuntimeHooks * = nullptr);
HTTPResult run_http(const Json &, const std::string &, const fs::path &,
                    const fs::path &, std::int64_t);
class CredentialSession {
  struct Impl;
  std::unique_ptr<Impl> impl;

public:
  symphony::ssiag::native::Secret secret;
  Json grant;
  CredentialSession(const Json &, const Json &);
  ~CredentialSession();
  Json finish(const std::string &, const std::string *sensitive = nullptr);
  CredentialSession(const CredentialSession &) = delete;
};
} // namespace cloudflare
