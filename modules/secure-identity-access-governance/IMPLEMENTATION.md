# SSIAG Procedural Implementation Guide

## Operating Rule

Complete phases in order. A phase may start only when its entry decisions are ratified and all prior exit gates pass. “Implemented” below describes the current bounded foundation, not overall production readiness. Do not enable a later capability merely because its interface appears in a contract.

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
5. State that caller type does not determine authority, local policy mutation requires target-host permission and audit-before-commit, and no caller edits a ledger through a supported interface.
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
2. Keep the foundation cgo-free and dependency-bounded to the Go standard library, the ratified `golang.org/x/sys` peer-credential boundary, and first-party STAV modules.
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

1. Keep qxctl dependencies bounded to the ratified Cobra/Viper command and configuration stack, cgo-free platform support, and first-party pure-Go protocol clients; never import a provider dependency.
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
4. Provide a bounded one-request/one-response standard-I/O scaffold.
5. Accept exactly `handshake`, `status`, and `capabilities` request operations.
6. Reject unknown fields, oversized/malformed input, and credential operations.
7. Report canonical status `disabled` plus `operational_access_enabled: false`, `provider_operations_enabled: false`, and `secret_channel_enabled: false`.
8. Do not import Apple Security yet; metadata behavior does not need Keychain access.

Verification:

```bash
swift test
swift build -c release
./tests/provider_trust_integration_darwin.sh
```

Exit gate: the adapter is independently installable and removable, while the Go foundation remains Swift-free, Apple-framework-free, and cgo-free.

## Phase 6 — Implement Ratified Caller Authentication and Supervision (implemented)

Ratified architecture:

1. Local v1 uses kernel-attested Unix-socket peer credentials mapped to canonical SSIAG subjects.
2. SSIAG and STAV occupy a foundational bootstrap stratum anchored by a native OS supervisor or explicit owner-provided equivalent.
3. Supervision owns liveness only and never expands security authority.
4. qxctl separates proposal from permission-backed apply; caller type is not an authorization input. Protected local policy status/proposal/apply/recovery is implemented in Phase 8A and remains distinct from canonical knowledge apply.

Implemented: build-tagged Darwin `LOCAL_PEERCRED`/`LOCAL_PEERPID` and Linux/WSL `SO_PEERCRED`; exact UID/GID subject and service mapping; scope-exact trusted configuration; client endpoint authentication before HTTP; per-TOPS launchd/systemd installation; owner-provisioned system identities; distinct service-owned runtime/state children; bounded restart and shutdown; supervisor-independent SSIAG/STAV startup; explicit direct-run policy; and exclusive socket lifecycle locks with conservative stale recovery. Older metadata-only v1 configuration remains structurally readable but cannot start a trusted service/client until re-enrolled.

Implementation procedure:

1. Maintain the implemented accepted-connection credential extraction and exact subject resolver.
2. Maintain the implemented stable service mapping and configure explicit canonical caller subjects that may later receive authority.
3. Preserve exact qxctl/self-client server endpoint authentication; socket groups remain reachability-only.
4. Preserve the implemented native/owner-provided liveness-only supervision and socket lock ordering.
5. Bind every future mutation request to TOPS ID, subject, request ID, operation, and expiry.
6. Add replay detection and strict deadlines with the mutation schema.
7. Authenticate the configured adapter executable by exact path, ownership, digest/signature policy, and protocol identity in Phase 9.
8. Add mutation/provider negative tests for replay, stale sessions, changed binaries, and cross-TOPS requests when those surfaces exist.

Exit gate: no unauthenticated local process can reach a mutation or adapter operation, and supervision does not silently expand authority.

## Phase 7 — Implement the Ratified STAV Append Authority Architecture (implemented)

Ratified architecture: one dedicated Go process per TOPS serialization domain, authenticated local producer IPC, no direct ledger writes by qxctl, producers, or other callers, and fail-closed ordinary security/configuration apply when required audit is unavailable. A future explicit target-host-administrator recovery path must journal audit-deferred evidence durably and reconcile forward without writing the ledger directly.

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

Exit gate: qxctl, SSIAG, node-troll, producers, and all other callers cannot write the ledger file directly through a supported interface; v1 claims tamper evidence but not non-repudiation.

## Phase 8 — Implement Deny-by-Default Policy (implemented for bounded authorization evidence)

Implemented scope: strict canonical authorization request/decision/capability schemas; exact subject and operation/resource/audience/scope grants; `host_owner` and `granted_permission` authority bases; empty-policy compatibility for older v1 enrollments; short-lived capability binding; caller-class-neutral property tests; fail-closed STAV submission; and atomic live policy-snapshot replacement. The capability is non-transferable safe evidence with canonical apply disabled. It authorizes no provider dispatch, safeguard change, credential release, or canonical write.

1. Canonicalize subject, proof summary, requested operation, provider, target, audience, scope, interaction, and time inputs.
2. Reject absent or unknown fields.
3. evaluate exact relationships from the SSIAG contract.
4. Return allow/deny plus a safe reason code and bounded capability.
5. Bind capabilities to subject, TOPS, provider, reference, operation, audience, issue/expiry, and request/correlation IDs.
6. Maintain positive and negative tests for exact grants, wildcard rejection, expiry/freshness, caller-class neutrality, unknown subjects, cross-TOPS/target drift, STAV unavailability, and capability binding.
7. Submit safe policy outcomes to the STAV append authority.

Exit gate passed for noncanonical knowledge-session authorization: no decision is released without exact evaluation and an auditable safe result. Provider dispatch remains disabled and retains its separate Phase 9/10 gates.

## Phase 8A — Protected Local Policy Administration (implemented)

1. Keep enrolled `config.json` immutable and select either its policy or one protected per-TOPS overlay.
2. Expose metadata-only status and subject-free proposal requests through SSIAG; administer them through `qxctl ssiag policy`.
3. Derive `host_owner` from the accepted connection's kernel UID, or require an explicit mapped subject and exact current grants for propose/apply/recover.
4. Bind proposals to TOPS, subject, authority basis, operation/request/correlation IDs, exact current/desired/config digests, UTC expiry, and proposal digest; durably retain the one current server-issued proposal so apply cannot bypass issuance.
5. Persist a no-follow, owner-controlled `prepared` attempt under an exclusive lock before STAV interaction.
6. Submit only safe references and prior/new digests through the closed SSIAG policy-decision producer; require its committed idempotent receipt.
7. Persist `audited`, atomically commit the next generation, remove the attempt, and replace the live evaluator snapshot.
8. Recover by exact operation plus attempt digest or explicit discovery; replay prepared audit idempotently and roll audited evidence forward.
9. Reject stale proposals, CAS drift, changed subjects/config, tampering, symlinks, oversized files, ambiguous recovery, and concurrent attempts.
10. Reset by committing `source=config`; do not erase generation evidence. Older binaries remain config-only and fail safely across upgrade-order differences.

Verification:

```bash
go test ./internal/policyadmin ./internal/policy ./internal/server
go test ./... # SSIAG and qxctl modules
CGO_ENABLED=0 go build -trimpath ./cmd/symphony-ssiag
```

Exit gate: local policy apply is caller-neutral, target-host-permission-backed, STAV-before-commit, CAS-safe, crash-recoverable, and explicitly noncanonical. It does not enable general safeguards, providers, credentials, or canonical knowledge apply.

## Phase 9 — Implement Ratified Provider Trust and Channel Separation (implemented, metadata only)

1. Maintain the frozen handshake, control request/response with embedded safe error, executable-trust, trust-query/result, and synthetic one-shot-channel schemas in `knowledge/ssiag/schemas/v1/`.
2. Enforce one request and one response per child process, 65,536-byte control bounds in each direction, five-second default and thirty-second maximum timeout, and 128-entry capability/check bounds.
3. Load one exact externally administered provider binding from the protected per-TOPS SSIAG configuration tree. Treat absence as `unbound`; never select the newest installed version or fall back.
4. Implement a Go adapter launcher with sanitized environment, exact executable trust, bounded pipes, invocation-context cancellation, process termination, and child cleanup. Do not add a cancellation message: a hung one-shot child cannot be made to observe it.
5. Have the adapter independently inspect its parent executable and installed receipt/signature and compare those observations to the request's exact foundation evidence. Caller assertions alone are not trust.
6. Keep safe metadata/control on the bounded protocol. The Phase 9 inherited descriptor remains synthetic and non-operational; operational secret delivery is a later gate.
7. Verify descriptor identity, exact protocol major, platform, capabilities, exportability, interaction requirements, and both halves of mutual executable trust.
8. Reject duplicate identities, downgrade, unadvertised operations, extra output, malformed responses, timeouts, early exit, parent-observation mismatch, and false trust claims.
9. Keep metadata discovery available independently of credential operations.

Exit gate: achieved for the exact macOS metadata adapter. An incompatible or compromised-looking adapter fails closed without fallback or secret-bearing diagnostics; operational Keychain and secret-channel behavior remain later gates.

## Phase 10A — Exact Provider Binding Lifecycle (implemented, metadata only)

1. Inventory receipt-v2 adapters only from admitted scope roots and legacy bootstrap evidence; pair each with the executing receipt-bound foundation and emit only opaque exact-pair IDs.
2. Keep managed state under the service-owned per-TOPS state root, separate from immutable Phase 9 configuration evidence.
3. Expose status, plan, apply, completed apply-status, and recovery through SSIAG; qxctl remains the preferred headless caller and never receives paths or selection authority.
4. Use exact state/plan CAS, one active ID, one predecessor, explicit forward/reverse graph actions, and no newest or fallback semantics.
5. Persist `prepared` with the initiating safe audit identity, independently execute the metadata handshake before `candidate_verified`, require the distinct committed STAV lifecycle receipt before `audited`, statically recheck the exact candidate, replace state while the attempt remains recoverable, persist `committed`, save the completed result, then remove the attempt.
6. Normalize initial `absent` into the deterministic provider-binding absence digest only for STAV's required previous-digest field; persisted state continues to report `absent` truthfully.
7. Provide a narrow native offline recovery path requiring an immutable receipt-v2 foundation, target-host ownership, entry into the enrolled service UID/GID, exclusive ownership of the persistent socket-lifecycle lease, an absent service socket, exact pending evidence, and normal idempotent STAV commitment using the original audit identity.
8. Keep `operational_access_enabled`, `provider_operations_enabled`, and `secret_channel_enabled` false throughout.

Verification:

```bash
go test ./internal/provider ./internal/server ./internal/stavproducer ./cmd/symphony-ssiag
go test ./... # SSIAG and qxctl modules
CGO_ENABLED=0 go build -trimpath ./cmd/symphony-ssiag
```

Exit gate: exact metadata-provider versions can be installed in either order, explicitly docked, undocked, rolled backward, queried after completion, and recovered after interruption without path leakage, recency guesses, fallback, unaudited mutation, or Keychain access.

## Phase 10B–10E — Enable the Ratified Per-User macOS Keychain Profile (future gates)

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

Production readiness requires ratified authentication, caller-class-neutral authorization, supervision, provider IPC, STAV durability/recovery/retention, at least one operational provider, platform-specific security review, secret-leakage tests, install/upgrade/rollback/uninstall tests, signed release provenance, and zero weakening of qxctl, SKV, host-authority, or protocol-integrity boundaries.
