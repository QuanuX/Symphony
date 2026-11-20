#pragma once

#include <gio/gio.h>

#include <cstddef>
#include <functional>
#include <span>
#include <string>

namespace symphony::ssiag::linux_secret_service {

// Native provider API for an already authorized caller. No principal or lease is
// inferred from a reference. The caller owns admission, cancellation and audit.
struct Scope {
    std::string collection_path;
    std::string tops_id;       // canonical lowercase UUID
    std::string reference;     // nonsecret canonical lowercase UUID
};

struct Item {
    Scope scope;
    std::string generation;    // unique UUID, including on uncertain creation
};

enum class Status { ok, invalid, unavailable, locked, not_found, ambiguous, cancelled, indeterminate };
struct ProvisionResult { Status status; Item item; };
using Bytes = std::span<const std::byte>;
using Consumer = std::function<void(Bytes)>;
inline constexpr std::size_t maximum_size = 4096;

// Creates a new immutable generation. No existing item is overwritten.
ProvisionResult provision(const Scope&, Bytes, GCancellable*);
// The callback borrows bytes only until it returns and is called at most once
// for this invocation. Durable one-shot issuance belongs to the SSIAG caller.
Status use(const Item&, std::size_t byte_ceiling, GCancellable*, const Consumer&);
// Select an exact local Unix socket path independently of process environment.
// This is a connection selector. The caller still owns service and peer trust.
Status use_on_bus(const std::string& socket_path, const Item&, std::size_t byte_ceiling,
                  GCancellable*, const Consumer&);
Status retire(const Item&, GCancellable*);

} // namespace symphony::ssiag::linux_secret_service
