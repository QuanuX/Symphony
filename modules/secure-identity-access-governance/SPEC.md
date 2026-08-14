# Symphony Secure Identity and Access Governance Module Specification

## Status

Safe metadata, audited authorization, protected local policy administration, receipt-v2 packaging, a module-owned foundational lifecycle adapter, exact metadata-only provider trust, and protected provider-binding lifecycle implement the Architect-ratified architecture in `knowledge/ssiag/SPEC.md`. Exact provider inventory, binding plan/apply/apply-status/recovery, and the receipt-bound metadata handshake are implemented. All provider-operation, Keychain-access, and secret-channel flags remain false.

## Invariants

1. Monorepo visibility is not runtime authority.
2. The foundation is Go-only and builds with `CGO_ENABLED=0`.
3. qxctl is the administrative/query voice, not schema or provider authority.
4. Immutable TOPS UUIDs and mutable display names remain separate.
5. Every per-TOPS path, socket, policy scope, and future STAV relationship uses the ID only.
6. Authorization defaults to deny.
7. Provider capability claims are truthful and fail closed.
8. Secrets and security proofs never enter administrative or knowledge surfaces.
9. SSIAG remains outside trading hot paths.

## Host Lifecycle

`package install --prefix ABSOLUTE --version VERSION` installs the binary at `libexec/symphony/secure-identity-access-governance/<version>/symphony-ssiag` and commits a strict `symphony.knowledge.install-receipt.v2` last. Requested, path, receipt, and compiled versions must agree. The receipt binds the exact bytes, platform, `ssiag.foundation-lifecycle` adapter entry point, command protocol, and lifecycle capability. `package uninstall` validates owned bytes and refuses retained supervisor, live-endpoint, or unresolved lifecycle-attempt references.

The historical `install --scope user|system` and `symphony.ssiag.install.v1` fixed-path layout remain a verified dual-read migration path.

Both uninstall paths remove only owned package evidence. They refuse active TOPS references and cannot purge per-TOPS configuration or state.

## Foundational Lifecycle Adapter

`foundation-lifecycle describe --json` emits one digest-bearing adapter descriptor. The machine form accepts exactly one bounded strict `symphony.foundation.lifecycle-command.v1` JSON value on stdin and emits exactly one bounded result on stdout. It supports enrollment and supervisor `observe`, `plan`, `apply`, `apply_status`, and `recover`, with exact state/attempt compare-and-swap, deadline and future-skew validation, immutable installation evidence, result digests, STSC timestamps, replay, and crash recovery. Attempts are no-follow, owner-controlled, fsynced, digest-linked, and stored in a host lifecycle root outside every per-TOPS purge subtree.

Observation is offline and does not invoke the native manager. Apply refuses an unavailable manager before attempt creation. Ordinary mutation fails before persistent attempt or external mutation until a real committed STAV receipt path is wired. Explicit `audit_deferred` mutation journals the state and returns `reconciliation_required=true`; the typed receipt-binding hook cannot itself validate or manufacture a receipt and is not a completed reconciliation endpoint. Purge is absent from qxctl/foundation lifecycle v1.

## TOPS Enrollment

`enroll --tops-id UUID --tops-name NAME` requires an existing host installation and creates `symphony.ssiag.config.v1` plus `symphony.ssiag.enrollment.v1` under one TOPS namespace. The ID is a canonical lowercase UUID. The non-empty display name is mutable safe metadata.

`unenroll` removes the enrollment marker and preserves data. Native-only `unenroll --purge` removes only the selected TOPS SSIAG configuration and state after proving the supervisor descriptor absent, refusing live/foreign endpoints, and acquiring the same adjacent socket lifecycle lock. It never deletes protected lifecycle attempts.

## Configuration

Configuration is strict JSON, bounded to 1 MiB, rejects unknown fields and trailing values, and contains `schema`, `mode`, `tops`, local `listen`, an authentication mapping, an authorization policy, and provider descriptors. The authentication mechanism is `unix_peer_credentials`. Its `service` member separately binds ID `symphony.ssiag.service` and kind `symphony.identity.service` to an explicitly present effective UID/GID pair; `subjects` maps caller identities. Authorization is explicitly deny-by-default, bounds capability lifetime to 1–86,400 seconds, and contains an explicit grant array. Each grant binds one configured subject and `host_owner` or `granted_permission` basis to one exact operation/resource/audience/scope tuple. Wildcards and implicit grants are unsupported. Missing authorization from an earlier v1 enrollment is readable only as an empty deny policy. Missing authentication/service members remain structurally readable but cannot start a trusted service or client until safely re-enrolled. Configuration MUST NOT contain credential values, assertions, tokens, recovery material, private keys, or provider payloads.

## Mutual Endpoint Authentication

Before the server (`symphony-ssiag`) changes runtime state or starts listening, it verifies that its process effective UID and GID match `authentication.service.uid` and `authentication.service.gid`. If there is a mismatch, the server fails closed.

On the client side, both the self-client and `qxctl` load the configuration corresponding to the requested TOPS ID and scope, enforcing that:
1. The configuration file is a regular file (no symlinks).
2. User-scope trust is owned by the current effective user and owner-only; system-scope trust is administrator-owned and not writable by group or other.
3. The configuration contains the exact canonical `authentication.service` ID, kind, UID, and GID.
4. The configured socket belongs to the requested TOPS layout; an explicit absolute socket override changes location only.

Before dialing, the client requires a Unix socket owned by the configured service UID. Upon dialing, it retrieves kernel-attested peer credentials (using Darwin `LOCAL_PEERCRED`/`LOCAL_PEERPID` or Linux `SO_PEERCRED`) and verifies that the connected peer exact UID and GID match the configured service identity. If verification fails, the connection is closed before any HTTP bytes are exchanged. Socket group/mode remains reachability policy and never substitutes for the post-dial check.

## Local API

The foundation listens on one Unix socket for one TOPS and exposes:

- `GET /v1/status`: version, readiness, mode, TOPS ID/name, transport, provider count;
- `GET /v1/providers`: safe declared descriptors.
- `POST /v1/authorization/decisions`: a bounded strict request containing request/correlation IDs, exact operation/resource/audience/scope, and fresh UTC issue/expiry intent. It contains no subject or caller-class field.
- `GET /v1/policy/status`: safe effective-policy digest, source, generation, state digest, and recovery metadata; never policy content.
- `POST /v1/policy/proposals`: subject-free desired policy or reset intent bound by SSIAG to kernel-derived authority and exact current/config/desired digests.
- `POST /v1/policy/apply`: exact proposal replay, compare-and-swap, durable prepare, committed STAV safe evidence, atomic generation commit, and live evaluator replacement.
- `POST /v1/policy/recover`: exact-operation recovery by attempt digest or explicit discovery, with idempotent audit replay and roll-forward only.
- `GET /v1/provider-installations/<provider>`: bounded exact-pair inventory using opaque IDs and no selection side effect.
- `GET /v1/provider-bindings/<provider>` and `POST .../plans|apply|recover`, plus `GET .../attempts/<operation-id>`: protected observation, proposal, audited mutation, durable completion lookup, and recovery.

SSIAG derives the authorization subject from kernel connection context, evaluates exact non-overlapping grants, and returns allow or deny only after the safe STAV policy event commits. Duplicate grants for the same subject and target tuple invalidate policy rather than creating an order-dependent result. An allow may include an expiring capability bound to the complete target, subject, TOPS, authority basis, grant, request/correlation pair, and policy/configuration digests. The capability is explicitly non-transferable and has no canonical apply authority. TCP binding, provider operation, credential, lease, general safeguard, and canonical apply routes are prohibited. Socket paths are absolute, restrictive, and collision-safe. Every request carries Darwin `LOCAL_PEERCRED`/`LOCAL_PEERPID` or Linux/WSL `SO_PEERCRED` context. Unmapped peers cannot request a normal decision; only a peer whose UID independently proves target-host ownership may receive the deterministic host-owner subject for local policy administration.

The enrolled config is not rewritten. Policy state lives under the selected per-TOPS state root in a private `policy/` directory. State and attempt files are bounded, no-follow, owner-controlled, digest-verified, fsynced, and atomically replaced under `policy.lock`. A prepared attempt blocks competing proposals until apply or recovery closes it. Older binaries ignore this separate state and safely evaluate enrolled config only. Reset creates a new `source=config` generation, preserving evidence instead of erasing history.

## Supervision and Socket Lifecycle

`supervisor install|uninstall --tops-id UUID --scope user|system` manages one per-TOPS launchd job or systemd unit through the same lifecycle transaction engine without touching configuration or state. Receipt-v2 descriptors pin the exact immutable invoked libexec path, not the legacy fixed path. User services run as the invoking user. System service accounts are provisioned by the owner/package manager; the descriptor consumes the exact numeric UID/GID already recorded during enrollment. SSIAG and STAV units are deliberately independent.

The Go server verifies its service identity, locks the persistent adjacent `ssiag.sock.lock`, proves any existing socket stale, binds the socket, and releases the lock only after graceful SIGTERM drain and socket removal. Native restart rate and shutdown time are bounded. System `serve` requires `--supervised`; direct user `serve` remains an explicit development diagnostic.

## qxctl Contract

`qxctl ssiag status|providers|doctor|policy ...|provider show|verify|installations|binding ... --tops-id UUID [--scope user|system]` resolves the same TOPS-isolated socket and validates source-owned responses. Provider binding accepts only an opaque installation ID, exact state or plan digest, and bounded reason; qxctl never receives paths, selects a version, opens receipts, launches adapters, or writes STAV. SSIAG owns observation, ordering, candidate verification, audit, state, and recovery.

## Provider Contract

The Phase 9 `provider-executable-trust.v1` file remains immutable bootstrap/legacy evidence. Lifecycle-capable foundations store managed state separately under the service-owned per-TOPS state root. Inventory inspects only admitted roots, assigns opaque content-addressed exact-pair IDs, never chooses newest, and never treats receipt presence as authority. A changed binding durably preserves its initiating safe audit identity, advances `prepared -> candidate_verified -> audited -> state -> committed -> result -> cleanup`, retains one predecessor without automatic fallback, and persists the last fully completed result for apply-status recovery. Any still-present attempt remains recovery-required. The native offline recovery command is receipt-v2-bound, requires target-host ownership, entry into the enrolled service UID/GID, exclusive ownership of the persistent socket-lifecycle lease, and an absent socket while that lease is held; it reconstructs the original STAV candidate and still requires normal commitment.

Provider v1 native code remains out of process. One child receives one strict control request and returns one strict control response, each at most 65,536 bytes, under a five-second default and thirty-second maximum deadline. All required members must be explicitly present, and duplicate, unknown, or omitted members fail closed. The foundation admits no more than four simultaneous metadata verifications globally and one per provider; saturation returns safe `busy` evidence without queuing another child. The complete request/correlation/TOPS/provider/adapter/operation/deadline tuple and foundation evidence bind the response. The adapter independently observes its parent path and installed receipt/signature; a caller assertion is never mutual trust. Go context cancellation terminates and reaps the one-shot child, so v1 defines no persistent cancellation frame. General control messages and embedded errors carry no secret bytes. The one-shot inherited descriptor is synthetic and non-operational in this phase; non-exportable operations and all operational Keychain access remain disabled. No implicit fallback is permitted.

## STAV Contract

SSIAG submits only the closed safe outcome vocabulary defined by `knowledge/ssiag/SPEC.md` to the dedicated per-TOPS Go append authority. The producer authenticates the authority endpoint, constructs no trusted ledger fields, requires a committed receipt, and never edits or spools ledger data.

## Implemented and Disabled Gates

Local peer authentication, exact UID/GID subject resolution, target-host-owner derivation, endpoint verification, native supervision, receipt-v2 identity, transactional recovery, exact-grant authorization, protected policy administration, typed SSIAG STAV submission, metadata-only provider mutual trust, and exact provider-binding lifecycle are implemented. General safeguards, lease issuance, credential delivery, operational provider operations, secret delivery, and canonical knowledge apply remain disabled.
