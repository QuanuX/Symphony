#include <symphony/ssiag/linux_secret_service.hpp>

#include <signal.h>
#include <unistd.h>

namespace symphony::ssiag::linux_secret_service {
// Linked only into the receiver regression binary. Simulate a lost process
// immediately after the durable reservation, before the outcome can be saved.
ProvisionResult provision(const Scope& scope, Bytes bytes, GCancellable*) {
    constexpr auto generation = "cfbaec86-99e4-4bfd-9c8f-d84968e85c1c";
    switch (std::to_integer<unsigned>(bytes.front())) {
    case 0x02: return {Status::locked, {scope, {}}};
    case 0x03: return {Status::indeterminate, {scope, generation}};
    case 0x04: kill(getpid(), SIGKILL); _exit(99);
    default: return {Status::ok, {scope, generation}};
    }
}
}
