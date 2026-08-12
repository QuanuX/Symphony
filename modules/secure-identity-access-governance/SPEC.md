# Symphony Secure Identity and Access Governance Module Specification

## Status

Safe metadata, audited authorization, protected local policy administration, and the STAV producer foundation implementing the Architect-ratified architecture in `knowledge/ssiag/SPEC.md`. Kernel caller authentication, native supervision, exact deny-by-default evaluation, non-transferable capability evidence, local policy proposal/apply/recovery, and mutually authenticated typed STAV submission are implemented. Local policy apply is operational and noncanonical; safeguard administration, canonical knowledge apply, credential delivery, and provider operation remain disabled. Canonical relationship and provider semantics remain owned by that Knowledge Vector.

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

`install --scope user|system` atomically copies one binary and writes `symphony.ssiag.install.v1` containing scope, version, exact binary path, and SHA-256 digest. Identical installation is idempotent. Replacing or removing a changed binary requires `--force`.

`uninstall` validates that record and removes only the host binary and install manifest. It cannot purge per-TOPS configuration or state.

## TOPS Enrollment

`enroll --tops-id UUID --tops-name NAME` requires an existing host installation and creates `symphony.ssiag.config.v1` plus `symphony.ssiag.enrollment.v1` under one TOPS namespace. The ID is a canonical lowercase UUID. The non-empty display name is mutable safe metadata.

`unenroll` removes the enrollment marker and preserves data. `unenroll --purge` removes only the selected TOPS SSIAG configuration, state, and socket after path and object-type validation.

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

SSIAG derives the authorization subject from kernel connection context, evaluates exact non-overlapping grants, and returns allow or deny only after the safe STAV policy event commits. Duplicate grants for the same subject and target tuple invalidate policy rather than creating an order-dependent result. An allow may include an expiring capability bound to the complete target, subject, TOPS, authority basis, grant, request/correlation pair, and policy/configuration digests. The capability is explicitly non-transferable and has no canonical apply authority. TCP binding, provider operation, credential, lease, general safeguard, and canonical apply routes are prohibited. Socket paths are absolute, restrictive, and collision-safe. Every request carries Darwin `LOCAL_PEERCRED`/`LOCAL_PEERPID` or Linux/WSL `SO_PEERCRED` context. Unmapped peers cannot request a normal decision; only a peer whose UID independently proves target-host ownership may receive the deterministic host-owner subject for local policy administration.

The enrolled config is not rewritten. Policy state lives under the selected per-TOPS state root in a private `policy/` directory. State and attempt files are bounded, no-follow, owner-controlled, digest-verified, fsynced, and atomically replaced under `policy.lock`. A prepared attempt blocks competing proposals until apply or recovery closes it. Older binaries ignore this separate state and safely evaluate enrolled config only. Reset creates a new `source=config` generation, preserving evidence instead of erasing history.

## Supervision and Socket Lifecycle

`supervisor install|uninstall --tops-id UUID --scope user|system` manages one per-TOPS launchd job or systemd unit without touching configuration or state. User services run as the invoking user. System service accounts are provisioned by the owner/package manager; the descriptor consumes the exact numeric UID/GID already recorded during enrollment. SSIAG and STAV units are deliberately independent.

The Go server verifies its service identity, locks the persistent adjacent `ssiag.sock.lock`, proves any existing socket stale, binds the socket, and releases the lock only after graceful SIGTERM drain and socket removal. Native restart rate and shutdown time are bounded. System `serve` requires `--supervised`; direct user `serve` remains an explicit development diagnostic.

## qxctl Contract

`qxctl ssiag status|providers|doctor|policy ... --tops-id UUID [--scope user|system]` resolves the same TOPS-isolated socket, rejects unsupported schemas, bounds files/responses, and binds every operation to a ready status response with the requested TOPS identity and scope. `policy propose` consumes a bounded deny-by-default policy or reset intent; `apply` consumes the exact returned proposal; `recover` requires explicit evidence. It accepts and prints no secret values.

## Provider Contract

Foundation provider entries are descriptive only. Operational adapters require mutual executable trust, kernel-authenticated local caller identity, time/size bounds, safe errors, cancellation, capability truth, and provider-specific review. Native code remains out-of-process. General control messages carry no secret bytes. Explicitly exportable bytes use a request-bound one-shot protected local channel; non-exportable operations remain in the provider. No implicit fallback is permitted.

## STAV Contract

SSIAG submits only the closed safe outcome vocabulary defined by `knowledge/ssiag/SPEC.md` to the dedicated per-TOPS Go append authority. The producer authenticates the authority endpoint, constructs no trusted ledger fields, requires a committed receipt, and never edits or spools ledger data.

## Implemented and Disabled Gates

Local peer authentication, exact UID/GID subject resolution, target-host-owner derivation, endpoint verification, native supervision/runtime ownership, serialized socket recovery, exact-grant authorization decisions, non-transferable capability evidence, protected policy overlay administration, and typed SSIAG STAV submission are implemented. Policy recovery still requires STAV and is not the separately deferred-audit recovery concept. General safeguard administration, audit-deferred recovery, lease issuance, credential delivery, operational providers, and canonical knowledge apply remain disabled. Remote access and any non-permission-backed apply path are unauthorized.
