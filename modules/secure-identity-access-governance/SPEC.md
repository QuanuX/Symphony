# Symphony Secure Identity and Access Governance Module Specification

## Status

Metadata-only API plus safe STAV producer foundation implementing the Architect-ratified architecture in `knowledge/ssiag/SPEC.md`. Kernel caller authentication and mutually authenticated, typed STAV submission are implemented. No mutation, credential delivery, provider operation, or supervision installer is enabled. Canonical relationship and provider semantics remain owned by that Knowledge Vector.

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

Configuration is strict JSON, bounded to 1 MiB, rejects unknown fields and trailing values, and contains `schema`, `mode`, `tops`, local `listen`, an authentication mapping, and provider descriptors. The authentication mechanism is `unix_peer_credentials`. Its `service` member separately binds ID `symphony.ssiag.service` and kind `symphony.identity.service` to an explicitly present effective UID/GID pair; `subjects` maps caller identities. Missing authentication/service members from earlier metadata-only v1 enrollments remain structurally readable but cannot start a trusted service or client until safely re-enrolled. Configuration MUST NOT contain credential values, assertions, tokens, recovery material, private keys, or provider payloads.

## Mutual Endpoint Authentication

Before the server (`symphony-ssiag`) changes runtime state or starts listening, it verifies that its process effective UID and GID match `authentication.service.uid` and `authentication.service.gid`. If there is a mismatch, the server fails closed.

On the client side, both the self-client and `qxctl` load the configuration corresponding to the requested TOPS ID and scope, enforcing that:
1. The configuration file is a regular file (no symlinks).
2. User-scope trust is owned by the current effective user and owner-only; system-scope trust is administrator-owned and not writable by group or other.
3. The configuration contains the exact canonical `authentication.service` ID, kind, UID, and GID.
4. The configured socket belongs to the requested TOPS layout; an explicit absolute socket override changes location only.

Before dialing, the client requires a Unix socket owned by the configured service UID. Upon dialing, it retrieves kernel-attested peer credentials (using Darwin `LOCAL_PEERCRED`/`LOCAL_PEERPID` or Linux `SO_PEERCRED`) and verifies that the connected peer exact UID and GID match the configured service identity. If verification fails, the connection is closed before any HTTP bytes are exchanged. Socket group/mode remains reachability policy and never substitutes for the post-dial check.

## Metadata API

The scaffold listens on one Unix socket for one TOPS and exposes only:

- `GET /v1/status`: version, readiness, mode, TOPS ID/name, transport, provider count;
- `GET /v1/providers`: safe declared descriptors.

TCP binding and mutation routes are prohibited. Socket paths are absolute, restrictive, and collision-safe. The runtime rejects non-socket objects rather than replacing them. Every request must carry connection context produced from Darwin `LOCAL_PEERCRED`/`LOCAL_PEERPID` or Linux/WSL `SO_PEERCRED`; missing or invalid kernel credentials return a safe authentication failure. Unmapped peers remain limited to these read-only routes and cannot resolve a canonical subject.

## qxctl Contract

`qxctl ssiag status|providers|doctor --tops-id UUID [--scope user|system]` resolves the same TOPS-isolated socket, rejects unsupported schemas, bounds responses, and binds every operation to a ready status response with the requested TOPS identity and scope before output. It accepts and prints no secret values.

## Provider Contract

Foundation provider entries are descriptive only. Operational adapters require mutual executable trust, kernel-authenticated local caller identity, time/size bounds, safe errors, cancellation, capability truth, and provider-specific review. Native code remains out-of-process. General control messages carry no secret bytes. Explicitly exportable bytes use a request-bound one-shot protected local channel; non-exportable operations remain in the provider. No implicit fallback is permitted.

## STAV Contract

SSIAG submits only the closed safe outcome vocabulary defined by `knowledge/ssiag/SPEC.md` to the dedicated per-TOPS Go append authority. The producer authenticates the authority endpoint, constructs no trusted ledger fields, requires a committed receipt, and never edits or spools ledger data.

## Implemented and Disabled Gates

Local peer authentication, exact UID/GID subject resolution, STAV endpoint identity verification, and typed SSIAG STAV submission are implemented. Proposal/apply mutation, administrative authorization, lease issuance, credential delivery, operational provider calls, and service supervision remain disabled. Remote access and agent apply authority are unauthorized.
