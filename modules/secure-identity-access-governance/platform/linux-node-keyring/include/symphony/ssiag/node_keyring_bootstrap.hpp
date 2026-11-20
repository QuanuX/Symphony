#pragma once

#include <atomic>
#include <chrono>
#include <cstddef>
#include <functional>
#include <span>
#include <stop_token>
#include <string>
#include <sys/types.h>

namespace symphony::ssiag::node_keyring {

using Clock = std::chrono::steady_clock;
enum class Handshake { established, refused, indeterminate };
enum class BootstrapStatus {
    established, refused, indeterminate, already_attempted, binding_refused,
    cancelled, expired, invalid, unavailable, locked, not_found, ambiguous
};

// Borrowed bytes expire on return. The callback owns its provider connection,
// observes the cutoff/cancellation, and reports the actual handshake outcome.
using Authenticate = std::function<Handshake(std::span<const std::byte>, Clock::time_point, std::stop_token)>;

// Call on a control thread during session startup. Bindings identify one
// service UID and item generation. A reconnect needs a fresh instance and
// rereads the host owner's binding. No secret or session is cached here.
class SessionBootstrap final {
public:
    explicit SessionBootstrap(std::string binding_path);
    SessionBootstrap(const SessionBootstrap&) = delete;
    SessionBootstrap& operator=(const SessionBootstrap&) = delete;
    [[nodiscard]] BootstrapStatus authenticate(Clock::time_point deadline, std::stop_token stop,
                                               const Authenticate& connect);

private:
    const std::string binding_path_;
    const pid_t pid_;
    std::atomic_flag used_ = ATOMIC_FLAG_INIT;
};

} // namespace symphony::ssiag::node_keyring
