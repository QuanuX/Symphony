#pragma once
#include "adapter.hpp"
namespace cloudflare {
Json connector_admin(const Json &, const fs::path &, Json &, bool &,
                     bool fixture_binary = false);
int connector_supervise(bool fixture_binary = false);
void connector_barrier(const fs::path &);
} // namespace cloudflare
