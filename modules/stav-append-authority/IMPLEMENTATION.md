# STAV Append Authority Implementation Guide

## Phase 0 — Namespace Scaffold (Implemented)

1. Record the owner-ratified module, executable, socket, schema reservations, and qxctl grammar in `knowledge/stav/`.
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

## Phase 2 — Integrity and Durability (Blocked Pending Ratification)

1. Ratify digest algorithm identifiers and genesis representation.
2. Ratify framing, file segmentation, fsync/commit semantics, crash recovery, retention, rotation, and evidence-preserving repair.
3. Implement a single-writer storage engine with partial-write and concurrent-submission tests.

## Phase 3 — Authenticated Local IPC (Blocked Pending Ratification)

1. Ratify producer subjects, enrollment, permissions, and provider-specific service identities.
2. Reuse the Go-only kernel peer-credential posture established by SSIAG.
3. Bind one listener to one TOPS serialization domain and reject scope mismatch before decoding a candidate.
4. Keep ingestion outside HTTP, OpenAPI, NATS, and qxctl.

## Phase 4 — Read-Only Administration (Partially Specified; Runtime Blocked)

1. Use the ratified query, query-page, and verification content and bounded qxctl query grammar.
2. Ratify status and local request/response envelope content after storage/recovery and reader-auth semantics are known.
3. Activate `qxctl stav status|verify|query|doctor` only against the authenticated local interface.
3. Apply TOPS authorization and redaction before output; never expose raw secret-bearing material.

## Phase 5 — Producer Integration (Blocked Pending Ratification)

1. Integrate SSIAG first with explicit event-class allowlists.
2. Add node-troll only through a separately reviewed producer authorization.
3. Fail security and configuration mutations closed when the required audit event cannot be committed.
4. Never add a producer-side spool or secondary writer.

## Go 1.27 Migration (Confirmed Release Only)

1. Keep `go.mod` and the root workspace pinned to Go 1.26.5 while 1.27 is unreleased.
2. After general availability, run the kernel's differential fixture/digest gate and this module's lifecycle/path/race/cross-build tests under both toolchains.
3. Permit JSON/UUID implementation substitutions only inside the shared kernel and only when protocol bytes and rejections remain identical.
4. Update all workspace module pins atomically. A toolchain change does not unlock any blocked phase above.
