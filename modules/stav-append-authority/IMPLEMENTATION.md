# STAV Append Authority Implementation Guide

## Phase 0 — Namespace Scaffold (Implemented)

1. Record the Architect-ratified module, executable, socket, schema reservations, and qxctl grammar in `knowledge/stav/`.
2. Create the independent Go module and Contract Quad.
3. Implement canonical TOPS UUID validation and pure user/system path resolution.
4. Implement atomic, digest-guarded executable install and uninstall without a manifest or per-TOPS state.
5. Reserve the read-only qxctl grammar behind an explicit protocol-content gate.
6. Test the module with cgo disabled and validate repository contracts.

Exit gate: no socket, configuration, state directory, schema type, listener, event, receipt, ledger, or producer capability exists.

## Phase 1 — Canonical Protocol Kernel (Implemented, Runtime Disabled)

1. Ratify candidate, canonical event, rejected/committed receipt representation, query, query-page, and verification schemas in `knowledge/stav/`.
2. Ratify strict I-JSON/JCS, safe integers, field presence, size limits, closed outcomes/redaction values, SHA-256 domains, and four-byte local frame mechanics.
3. Add canonical schemas, registries, and valid/invalid fixtures under `knowledge/stav/`.
4. Implement and test the authority-free pure-Go kernel at `libraries/stav-protocol-go/`.
5. Reuse kernel TOPS-ID validation in this module without activating candidate ingestion.

Exit gate: the schemas and codec exist, but this executable still cannot listen, ingest, assign trusted fields, emit a committed receipt, or write state.

## Phase 2 — Integrity and Durability (Implemented)

1. Implement the ratified SHA-256 digest domains and per-TOPS genesis representation.
2. Record and implement the exact ledger framing, fsync/commit semantics, deterministic crash recovery, retention posture, rotation posture, and evidence-preserving repair boundary.
3. Implement a single-writer storage engine with partial-write and concurrent-submission tests.

## Phase 3 — Authenticated Local IPC (Implemented)

1. Record and implement exact producer subjects, enrollment, permissions, and provider-specific service identities.
2. Reuse the Go-only kernel peer-credential posture established by SSIAG.
3. Bind one listener to one TOPS serialization domain and reject scope mismatch before decoding a candidate.
4. Keep ingestion outside HTTP, OpenAPI, NATS, and qxctl.

## Phase 4 — Read-Only Administration (Implemented)

1. Use the ratified query, query-page, and verification content and bounded qxctl query grammar.
2. Record and implement status and local request/response envelope content atomically with storage/recovery and reader-auth semantics.
3. Activate `qxctl stav status|verify|query|doctor` only against the authenticated local interface.
4. Apply TOPS authorization and redaction before output; never expose raw secret-bearing material.

## Phase 5 — Producer Integration (SSIAG Implemented)

1. Integrate SSIAG first with explicit event-class allowlists.
2. Add node-troll only through a separately reviewed producer authorization.
3. Fail security and configuration mutations closed when the required audit event cannot be committed.
4. Never add a producer-side spool or secondary writer.

Implemented evidence includes strict operational schemas/fixtures, exclusive-lock and restart tests, identical/conflicting request tests, incomplete-tail and corruption tests, authenticated endpoint/caller integration tests, qxctl activation, and the closed SSIAG producer mapping. node-troll remains a separate future producer review.

## Phase 6 — Foundation Supervision (Implemented)

1. Install deterministic per-TOPS launchd jobs or systemd units without coupling STAV startup to SSIAG.
2. Consume the owner-provisioned numeric authority identity from configuration; never create accounts or infer root.
3. Give only the selected state/recovery/runtime children to that authority identity while preserving administrator-owned trust configuration and shared parents.
4. Require native/owner-controlled supervision for system scope and retain warned foreground user scope for diagnostics.
5. Acquire a persistent no-follow exclusive socket lifecycle lock after identity verification and release it only after graceful drain and socket cleanup.
6. Bound native restart cadence and SIGTERM shutdown; preserve descriptor-only no-start/no-stop integration for other owner-provided supervisors.

## Go 1.27 Migration (Confirmed Release Only)

1. Keep `go.mod` and the root workspace pinned to Go 1.26.5 while 1.27 is unreleased.
2. After general availability, run the kernel's differential fixture/digest gate and this module's lifecycle/path/race/cross-build tests under both toolchains.
3. Permit JSON/UUID implementation substitutions only inside the shared kernel and only when protocol bytes and rejections remain identical.
4. Update all workspace module pins atomically. A toolchain change does not alter any operational contract above.
