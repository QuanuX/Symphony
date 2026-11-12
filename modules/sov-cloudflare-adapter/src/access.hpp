#pragma once
#include <functional>
#include <symphony/ssiag/private_channel.hpp>
namespace cloudflare::access {
namespace n = symphony::ssiag::native;
using n::Json;
namespace fs = std::filesystem;
struct Runtime {
  fs::path helper, cloudflared;
  std::string helper_digest, cloudflared_digest, carrier_digest, ssh_digest;
};
// Production obtains this only from the protected, receipt-owned package.
Runtime runtime(const fs::path &executable);
void parent(const Runtime &, bool connection);
Json session(const Runtime &, const Json &request);
int connect(const Runtime &, const std::string &application,
            const fs::path &directory, const std::string &reference);
// Pure local inspection; unsigned JWT metadata never establishes provider
// admission.
Json claims(const std::string &token, const Json &identity,
            std::int64_t now_ms);
} // namespace cloudflare::access
