# Native SSIAG private credential provider specification

## Identity, storage and scope

This opt-in provider is a private-file store, not an OS keychain, encryption service or defense against hostile root/same-UID processes. It runs only with actual/effective non-root UID/GID agreement. No SUDO_USER inference. A selected canonical directory is owned by the user, mode 0700, beneath nonsymlink non-writable ancestors (root-owned sticky temporary ancestors excepted). Secret/configuration files are regular single-link private files. Core dumps are disabled. File publication and containing directories are synced; filesystem durability remains platform dependent.

`--configure ROOT` consumes closed nonsecret stdin `{expected_digest,configuration}`. Configuration selects the protocol `symphony.ssiag.private-store-config.v1`, TOPS ID, SSIAG Unix socket/UID/executable digest, exact consuming native executable digest and exact toolchain digest. It uses compare-and-swap and an exclusive store lock. No path, executable or trust selection comes from provider output.

`--seed ROOT` consumes metadata on stdin: reference, purpose, account_id, explicit nullable zone_id/resource_id, generation `1`, and local expires_unix_ms. Raw printable token bytes arrive only through inherited FD 3 (EOF bounded to 16 KiB). References are `ssiag-native:NAME`; purpose is `cloudflare.api-token` or `cloudflare.tunnel-token`. No existing credential is overwritten. `--inspect ROOT` consumes `{reference,generation}` and returns metadata only. `--revoke ROOT` requires the current generation, increments it and sets local expiry to zero. Revocation does not revoke the token at Cloudflare or terminate an already running connector.

Bootstrap/configuration/revocation require local ownership and never call Cloudflare. They are explicit native administration steps; qxctl has no raw-secret import parameter. API-token and Tunnel-token purposes remain distinct. Local expiry is an operator bound, not an assertion about provider expiry.

## One-shot private use

`--session ROOT CONFIG_DIGEST` accepts only an inherited connected Unix socket at FD 3. The kernel peer must be the actual parent with the same non-root UID/GID, and its process executable bytes must match the selected consumer digest. A bounded frame selects `{intent,credential_generation}`. The intent binds request UUID, TOPS, plan, adapter/toolchain digests, credential/result references, operation, account/zone/resource and an at-most-60-second deadline.

The owner verifies reference generation, purpose, target restrictions and expiry, takes a shared store lock, and creates an exclusive private use journal. It connects to the selected SSIAG socket and verifies the service UID, actual process and executable digest. Go receives only the nonsecret intent. SSIAG requires a mapped principal and exact operation/resource/audience/scope permission; there is no transferable local-v1 capability. A committed STAV intent is required before export. The upgraded connection holds the policy snapshot pinned until completion or deadline. Concurrent configuration, seed and revoke fail while use holds the store lock; subsequent use rechecks current state.

The native owner sends a grant frame followed by a separate raw-secret frame. Only the native adapter sees the secret. It verifies admission binding and constructs the exact authorized child's environment. Lost replies do not authorize re-export or mutation replay. A use UUID cannot reopen its journal. The service records intent/outcome under distinct STAV request IDs with common correlation. Invalid/missing receipts fail closed; lost completion is audited as unavailable, not success.

## Sensitive result import and cleanup

The adapter sends a completion control frame and, only when explicitly declared, a separate raw sensitive-result frame. Tunnel token retrieval imports a Tunnel-token reference; raw Tunnel create/get results are retained under `cloudflare.sensitive-result`. A returned opaque reference never contains credential bytes. Existing results are acknowledged only for the same originating plan; other overwrites fail before export. Import is bounded to 4 MiB. Result import and final committed audit must finish before a successful control response.

Owned string buffers are overwritten where practical; arbitrary JSON/SDK/runtime heap copies cannot be guaranteed erased. Private files intentionally retain selected credentials. Child environment inspection by the same UID or root remains outside this trust boundary. Raw stdout/stderr, error bodies and sensitive response fields never enter the Go/control/audit path. Process crashes or missing durable records can leave unresolved private state requiring operator inspection; they do not silently retry.

## Qualification

Native local fixtures plus actual Go SSIAG policy/Unix-peer tests qualify the implemented boundary, not a live Cloudflare account. Exact pinned CLI tests use synthetic responses. macOS and Linux IPC implementations are supplied; each deployment/platform must satisfy I6 acceptance. No SSH v1 behavior, root profile publication, Cloudflare Access login or automatic service activation is implemented by this component.
