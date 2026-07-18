# SSIAG Procedural Implementation Guide

## Operating Rule

Complete phases in order. A phase may start only when its entry decisions are ratified and all prior exit gates pass. “Implemented” below describes scaffold behavior, not production readiness. Do not enable a later capability merely because its interface appears in a contract.

## Phase 0 — Canonicalize Names and Authority (implemented and ratified)

1. Use the long name “Symphony Secure Identity and Access Governance” and acronym SSIAG.
2. Use the foundation namespace `modules/secure-identity-access-governance/`, binary `symphony-ssiag`, qxctl group `ssiag`, schemas `symphony.ssiag.*`, and environment prefix `SYMPHONY_SSIAG_*`.
3. Reserve `knowledge/ssiag/` for canonical SSIAG truth and `knowledge/stav/` for canonical STAV truth.
4. Remove the trading-ambiguous predecessor term from active SSIAG command, schema, path, and contract surfaces.
5. Keep the earlier handoff as coordination history; do not treat it as repository source truth.

Exit gate: one Architect-ratified namespace map, no mixed executable/API grammar, and canonical merge evidence recorded.

## Phase 1 — Establish Knowledge Contracts (implemented and ratified)

1. Define SSIAG's full decision chain and graph-like relationships in `knowledge/ssiag/`.
2. Define extension rules without authorizing a graph database.
3. Define STAV authority, storage, ten-group envelope, append behavior, and exclusions in `knowledge/stav/`.
4. State that qxctl implements but does not own either schema.
5. State that agents query/propose only and never edit a ledger.
6. Add every new canonical Markdown surface to SKVI.
7. Do not create SCLV merge evidence before a real PR and merge commit exist.

Exit gate: canonical relationships and authority splits can be reviewed without reading implementation code.

## Phase 2 — Separate Host Installation from TOPS Enrollment (implemented)

1. Resolve a host `InstallLayout` containing only the shared binary and install manifest.
2. Resolve an `InstanceLayout` from scope plus canonical lowercase TOPS UUID.
3. Store `tops_id` and `tops_name` separately.
4. Use the ID only under configuration, state, runtime, and socket roots.
5. Write `symphony.ssiag.install.v1` with the exact binary and SHA-256 digest.
6. Write `symphony.ssiag.enrollment.v1` with one TOPS's exact paths and display metadata.
7. Make installation and enrollment idempotent.
8. Permit a display-name update without moving paths.
9. Make host uninstall preserve all TOPS data unconditionally.
10. Make unenroll preserve data unless one-TOPS `--purge` is explicit.
11. Reject symlinks, non-regular manifests/binaries/configuration, non-socket collisions, unsafe scopes, and invalid IDs.

Verification:

```bash
go test ./internal/paths ./internal/lifecycle ./internal/config
```

Exit gate: two TOPS UUIDs produce distinct configuration, state, and socket paths, and host uninstall leaves both configurations intact.

## Phase 3 — Build the Go-Only Metadata Foundation (implemented)

1. Keep the Go module independently buildable.
2. Use only the Go standard library in the scaffold.
3. compile and test with `CGO_ENABLED=0`.
4. Keep domain types in separate identity, policy, credential, and provider packages.
5. Reject unknown configuration fields and multiple JSON values.
6. Bind each process to exactly one enrolled TOPS configuration.
7. Expose only `GET /v1/status` and `GET /v1/providers` on a Unix socket.
8. Include TOPS ID and display name in safe status.
9. Bound headers, response bodies, and server timeouts.
10. Restrict socket permissions and reject ordinary files at the endpoint path.
11. Leave every provider descriptive and non-operational.

Verification:

```bash
go test ./...
go vet ./...
CGO_ENABLED=0 go build -trimpath -o symphony-ssiag ./cmd/symphony-ssiag
```

Exit gate: the binary contains no mutation route, secret storage, cgo linkage, network listener, or operational provider.

## Phase 4 — Integrate qxctl Read-Only Administration (implemented)

1. Keep qxctl free of third-party dependencies; ratified first-party pure-Go protocol libraries remain allowed and authority-free.
2. Add `qxctl ssiag status`, `providers`, and `doctor`.
3. Require `--tops-id` or `SYMPHONY_SSIAG_TOPS_ID`.
4. Resolve the same per-TOPS socket layout as the foundation.
5. Bound HTTP responses and reject unknown major schemas.
6. Compare the returned status TOPS ID with the requested ID.
7. Print only safe metadata in text and versioned JSON.
8. Return nonzero for connection, schema, identity, or readiness failure.

Exit gate: qxctl never imports a provider dependency, accepts a secret flag, or reaches an operational provider directly.

## Phase 5 — Scaffold the macOS Adapter Boundary (implemented, metadata only)

1. Place the independent Swift Package in `modules/ssiag-provider-macos-keychain/`.
2. Build `symphony-ssiag-provider-macos-keychain` separately from SSIAG.
3. implement independent digest-safe `install` and `uninstall` commands.
4. Provide a bounded JSON-lines standard-I/O scaffold.
5. Accept exactly `hello`, `status`, and `capabilities` request operations.
6. Reject unknown fields, oversized/malformed input, and credential operations.
7. Report `declared_not_operational` and `operational_access_enabled: false`.
8. Do not import Apple Security yet; metadata behavior does not need Keychain access.

Verification:

```bash
swift test
swift build -c release
.build/release/symphony-ssiag-provider-macos-keychain status
```

Exit gate: the adapter is independently installable and removable, while the Go foundation remains Swift-free, Apple-framework-free, and cgo-free.

## Phase 6 — Implement Ratified Caller Authentication and Supervision

Ratified architecture:

1. Local v1 uses kernel-attested Unix-socket peer credentials mapped to canonical SSIAG subjects.
2. SSIAG and STAV occupy a foundational bootstrap stratum anchored by a native OS supervisor or explicit owner-provided equivalent.
3. Supervision owns liveness only and never expands security authority.
4. qxctl separates proposal from apply; agents may query and propose only.

Implemented increment: build-tagged Darwin `LOCAL_PEERCRED`/`LOCAL_PEERPID` and Linux/WSL `SO_PEERCRED` wrappers; request-context enforcement; exact UID/GID-to-subject mapping; duplicate/ambiguous mapping rejection; unmapped-subject failure; server process self-verification; scope-exact trusted configuration loading; pre-dial socket type/owner checks; and exact client-side kernel endpoint verification before HTTP bytes. New enrollments declare `unix_peer_credentials`, a stable canonical service mapping, and an explicit caller-subject array. Older metadata-only v1 configuration remains structurally readable but cannot start a trusted service/client until re-enrolled.

Remaining entry details: launchd/service-manager labels, runtime directory and service-account provisioning, restart bounds, direct-run production warnings, and negative integration tests using distinct operating-system accounts. Adapter executable trust and future mutation replay/binding remain later gates.

Implementation procedure:

1. Maintain the implemented accepted-connection credential extraction and exact subject resolver.
2. Maintain the implemented stable service mapping and configure explicit canonical caller subjects that may later receive authority.
3. Preserve exact qxctl/self-client server endpoint authentication; socket groups remain reachability-only.
4. Bind every future mutation request to TOPS ID, subject, request ID, operation, and expiry.
5. Add replay detection and strict deadlines.
6. Authenticate the configured adapter executable by exact path, ownership, digest/signature policy, and protocol identity.
7. Add negative tests for fake sockets, wrong users, stale sessions, changed binaries, replay, and cross-TOPS requests.

Exit gate: no unauthenticated local process can reach a mutation or adapter operation, and supervision does not silently expand authority.

## Phase 7 — Implement the Ratified STAV Append Authority Architecture

Ratified architecture: one dedicated Go process per TOPS serialization domain, authenticated local producer IPC, no qxctl/producer/agent file writes, and fail-closed security/configuration apply when required audit is unavailable.

Implemented namespace: `modules/stav-append-authority/`, `symphony-stav-append-authority`, nested per-TOPS `stav/append.sock`, canonical `symphony.stav.*` contracts, and read-only `qxctl stav` grammar.

Implemented entry details: exact canonical serialization, genesis digest, append-only record framing, fsync-before-receipt, startup verification, incomplete-tail evidence recovery, preserve-all retention, disabled rotation, mutual peer authentication, exact grants, and bounded read projection.

Implementation procedure:

1. Implement the schema from `knowledge/stav/SPEC.md`, not from qxctl types.
2. Build one append authority per TOPS sequence.
3. Accept candidate events only from authorized producer identities.
4. validate all ten groups and explicit `not_applicable` configuration digest reason.
5. Assign timestamp, monotonic sequence, and preceding digest inside the authority.
6. Canonically serialize, hash, durably append, and return a safe receipt.
7. Recover a final partial write without inventing a valid event.
8. Implement a read-only verifier and redacted query projection.
9. Expose qxctl queries/proposals without file access.
10. Test concurrent producers, crashes at each write boundary, deletion/insertion/reordering/modification, cross-TOPS injection, redaction, and projection rebuild.

Exit gate: agents, qxctl, SSIAG, and node-troll cannot write the ledger file directly; v1 claims tamper evidence but not non-repudiation.

## Phase 8 — Implement Deny-by-Default Policy

1. Canonicalize subject, proof summary, requested operation, provider, target, audience, scope, interaction, and time inputs.
2. Reject absent or unknown fields.
3. evaluate exact relationships from the SSIAG contract.
4. Return allow/deny plus a safe reason code and bounded capability.
5. Bind capabilities to subject, TOPS, provider, reference, operation, audience, issue/expiry, and request/correlation IDs.
6. Add property and fuzz tests for wildcards, extension types, expiry, interaction, assurance, and cross-TOPS confusion.
7. Submit safe policy outcomes to the STAV append authority.

Exit gate: no provider dispatch occurs without an exact allow decision and auditable safe result.

## Phase 9 — Implement Ratified Provider Trust and Channel Separation

1. Freeze handshake, request, response, cancellation, and error schemas in `knowledge/ssiag/`.
2. Define maximum message size, concurrency, timeout, interaction, and restart rules.
3. Keep safe metadata/control on the bounded protocol and put explicitly exportable secret bytes on a request-bound one-shot protected descriptor/channel.
4. implement a Go adapter launcher with sanitized environment, exact executable trust, bounded pipes, cancellation, and child cleanup.
5. Verify descriptor identity, protocol major, platform, capabilities, exportability, and interaction requirements.
6. Reject duplicate identities, downgrade, unadvertised operations, extra output, malformed responses, timeouts, and early exit.
7. Keep metadata discovery available independently of credential operations.

Exit gate: an incompatible or compromised-looking adapter fails closed without fallback or secret-bearing diagnostics.

## Phase 10 — Enable the Ratified Per-User macOS Keychain Profile

Ratified architecture: per-user and session-aware operation; no system/headless login-Keychain access; TOPS-scoped non-synchronizing items by default; most restrictive usable accessibility/user presence; preference for non-exportable key operations before general export.

Remaining entry details: exact Keychain item namespace, classes and operations; access groups; access-control matrix; locked-session behavior; signing requirements; entitlements; notarization; distribution; update trust; provisioning; deletion/rotation semantics.

1. Import Apple Security only in the Swift adapter.
2. Map each Keychain operation to a canonical provider capability.
3. Keep non-exportable operations non-exportable.
4. Require policy authorization before every operation.
5. Use bounded buffers and minimize material lifetime when export is explicitly allowed.
6. Sanitize OSStatus and native errors into safe reason codes.
7. Test locked/unlocked Keychain, denied prompt, missing item, duplicate item, user cancel, timeout, changed access control, adapter replacement, and concurrent requests.
8. Verify stdout/stderr, process arguments, environment snapshots, qxctl, STAV, and crash logs contain no secret test markers.
9. Mark readiness operational only after platform security review.

Exit gate: no plaintext fallback exists and every operation's actual Apple semantics match its declared SSIAG capability.

## Phase 11 — Add Remaining Providers in Order

For each provider, repeat contract, threat-model, adapter trust, negative conformance, fail-closed, leakage, install/upgrade/uninstall, and STAV mapping gates.

1. Linux Secret Service, including locked session and absent desktop bus.
2. Explicit headless Linux/NVIDIA ARM hardware or workload provider; never pretend Secret Service exists.
3. WSL interoperability with explicit Windows session/provider detection.
4. Only later: OIDC/OAuth exchange, workload identity, FIDO2/passkeys, YubiKey/PIV, SSH agent, remote secret providers.

No provider becomes a universal default. Deployment configuration selects an exact reviewed provider and its absence is an error.

## Phase 12 — Release, Rollback, and Evidence

Before merge:

1. run Go unit, race, vet, fuzz, and cgo-disabled builds;
2. run Swift tests and release build on supported macOS architectures;
3. run lifecycle tests in temporary user/system roots;
4. run qxctl metadata and failure-path tests;
5. run STAV verifier/adversarial tests when present;
6. run symphony-validator with zero new violations;
7. inspect dependencies, produced artifacts, permissions, and secret markers;
8. review all deferred decisions and threat-model deltas.

After real review and merge evidence exists, add an SCLV entry with the actual PR URL and 40-character merge commit. State compatibility, rollback, projection, and publication consequences. SODV alone decides public publication.

Rollback order:

1. disable the affected provider without deleting provider-held material;
2. revoke leases/tokens where supported;
3. stop new operations while preserving safe STAV evidence;
4. restore the last compatible adapter and foundation binaries;
5. restore compatible configuration without changing TOPS identity;
6. verify qxctl status and provider readiness;
7. record canonical rollback facts only through the normal review/merge/SCLV process.

## Production-Ready Definition

Production readiness requires ratified authentication, authorization, supervision, provider IPC, STAV durability/recovery/retention, at least one operational provider, platform-specific security review, secret-leakage tests, install/upgrade/rollback/uninstall tests, signed release provenance, and zero weakening of qxctl/SKV/agent boundaries.
