# Symphony TOPS Audit Vector Specification

## Status

Architect-ratified v1 canonical content, read-message content, framing mechanics, append-authority architecture, operational durability, authenticated local IPC, and producer/reader authorization contracts. Runtime enablement requires a conforming implementation and verification evidence.

## Ratified Implementation Namespace

- module: `modules/stav-append-authority/`;
- Go module: `github.com/QuanuX/Symphony/modules/stav-append-authority`;
- executable: `symphony-stav-append-authority`;
- nested socket: `.../symphony/<tops-id>/stav/append.sock`;
- schema prefix: `symphony.stav.*`;
- qxctl group: read-only `qxctl stav status|verify|query|doctor`.
- protocol kernel: `libraries/stav-protocol-go/`, Go module `github.com/QuanuX/Symphony/libraries/stav-protocol-go`, package `stavprotocol`.

The protocol kernel is a build-time library with no binary, installer, resident, socket, state, or runtime authority. It implements this specification but does not own or amend it.

## Canonical v1 Artifacts

The following identifiers and JSON Schema Draft 2020-12 documents are ratified:

| Identifier | Canonical schema |
|---|---|
| `symphony.stav.candidate.v1` | `schemas/v1/candidate.schema.json` |
| `symphony.stav.event.v1` | `schemas/v1/event.schema.json` |
| `symphony.stav.receipt.v1` | `schemas/v1/receipt.schema.json` |
| `symphony.stav.query.v1` | `schemas/v1/query.schema.json` |
| `symphony.stav.query-page.v1` | `schemas/v1/query-page.schema.json` |
| `symphony.stav.verification.v1` | `schemas/v1/verification.schema.json` |
| `symphony.stav.append-authority.config.v1` | `schemas/v1/append-authority-config.schema.json` |
| `symphony.stav.append-authority.status.v1` | `schemas/v1/append-authority-status.schema.json` |
| `symphony.stav.local.request.v1` | `schemas/v1/local-request.schema.json` |
| `symphony.stav.local.response.v1` | `schemas/v1/local-response.schema.json` |

`schemas/v1/common.schema.json`, `registries/v1/base.md`, and `fixtures/v1/` are also canonical. Canonical artifacts under `knowledge/stav/` outrank Go types, generated output, qxctl, or module code.

## Canonical Serialization Profile

STAV v1 uses UTF-8 I-JSON serialized with RFC 8785 JCS. A strict decoder MUST reject invalid UTF-8, unpaired surrogate escapes, Unicode noncharacters, duplicate names, case-mismatched or unknown members, trailing values, and `null`.

Numbers are limited to canonical, non-negative base-10 JSON integers from `0` through `9007199254740991`. Negative values, negative zero, leading zeros, fractions, exponents, and out-of-range integers fail closed. Event sequences begin at `1`; a query cursor may use `0`. The v1 sequence MUST never wrap or silently cross the safe-integer ceiling.

Registered identifiers are lowercase dotted ASCII values. UUIDs are canonical lowercase RFC 9562 values: event IDs are UUIDv4; request and correlation IDs are UUIDv4 or UUIDv7. Timestamps use UTC `YYYY-MM-DDTHH:MM:SS.NNNNNNNNNZ` with exactly nine fractional digits and no leap-second emission. Digests use `sha256:` followed by exactly 64 lowercase hexadecimal characters.

Typed wire decoders MUST require supplied bytes to equal their JCS re-encoding. Ordinary Go `encoding/json` struct unmarshalling is not a conforming decoder by itself.

## Per-TOPS Ledger Boundary

One ledger sequence belongs to exactly one immutable TOPS ID. Multiple TOPS on one host MUST use distinct state roots, append serialization, sequence counters, digest chains, and query scopes. Display names MUST NOT determine any ledger path or identity.

## Ten-Group Event Envelope

Every event contains all ten groups. A group or subfield MUST NOT disappear silently.

1. `identity`: schema version and unique event ID — required.
2. `topology`: TOPS ID and an explicit TROG value or `not_applicable` reason — required.
3. `actor`: safe actor reference and authentication-method identifier — required.
4. `operation`: command/operation identifier and safe target reference — required.
5. `correlation`: correlation ID and request ID — required.
6. `result`: safe intent identifier, outcome, and reason code — required.
7. `configuration`: previous and new configuration digests — conditionally required. A non-mutational event carries an explicit `not_applicable` reason.
8. `ordering`: high-precision timestamp and monotonically increasing sequence — required.
9. `integrity`: preceding-event digest — required, including a defined genesis value.
10. `redaction`: redaction classification — required.

The exact field maps and tagged-union variants are defined by the canonical v1 schemas. Implementations MUST NOT invent optional omission behavior or unknown extension members.

## Candidate and Receipt Boundary

A candidate contains producer-proposed topology, domain principal/authentication context, operation, correlation, result, configuration presence, and redaction classification. It never contains authoritative event identity, producer identity, timestamp, sequence, predecessor digest, or event digest. Its maximum canonical size is 61,440 bytes; a canonical event and request frame are limited to 65,536 bytes.

A receipt binds the request ID to the candidate digest. A rejected receipt carries only a safe registered reason and an explicit `not_committed` result. A committed receipt may be emitted only after the complete ledger frame has been written and synchronized to durable storage. A repeated request ID with byte-equivalent candidate content returns the original committed receipt; the same request ID with different candidate content fails closed as an idempotency conflict.

## Append Protocol

Producers submit candidate events without a trusted ledger event ID, timestamp, sequence, or chain digest. One dedicated Go append-authority process per TOPS serialization domain authenticates the local producer, validates authorization, schema, presence, redaction, TOPS scope, reason codes, and size; assigns the trusted ledger identity and ordering values; writes atomically and durably; then returns a safe receipt. Concurrent submissions MUST serialize deterministically. Failure MUST NOT leave a valid-looking partial event.

The producer-to-authority transport is mutually authenticated local IPC and is not HTTP or OpenAPI. On Darwin and Linux, the accepted Unix-socket connection's kernel-attested UID and GID MUST map exactly to a configured producer or reader grant. A producer grant binds one canonical subject and producer identity to an explicit allowlist of `(event_class, operation_id)` tuples. A reader grant binds one canonical subject to an explicit allowlist of redaction classifications. The same contract names the append authority's expected canonical subject and UID/GID: the authority verifies its effective identity before listening, and qxctl or a producer verifies the connected server's kernel-attested identity before sending a request. Unknown, ambiguous, duplicate, or mismatched mappings fail closed. Socket permissions are defense in depth. A production producer and authority SHOULD run under distinct operating-system service identities; sharing a user identity intentionally shares that kernel-authenticated authority. The append authority's supervisor owns liveness only and MUST NOT gain producer or ledger authority.

The append authority owns socket creation and never uses supervisor socket activation. After verifying its configured process identity, it acquires an exclusive non-blocking lock on the persistent adjacent `append.sock.lock` regular file before inspecting or removing the socket. It refuses a live or foreign endpoint, removes only a provably stale socket, drains accepted bounded requests on SIGTERM, removes the socket, and releases the lifecycle lock last. The native profiles use launchd label `io.github.quanux.symphony.stav.<tops-id>` or systemd unit `symphony-stav@<tops-id>.service`, have no SSIAG dependency, and consume the configured numeric authority identity rather than creating or inferring one. System state and runtime children are owned by that authority identity; configuration remains administrator-owned and readable trust metadata. Linux uses `/run/symphony/<tops-id>/stav`; macOS uses `/var/run/symphony/<tops-id>/stav`.

Direct file mutation is prohibited through supported interfaces for every caller, including qxctl clients and producers. Enrollment creates no producer or reader grant; grants require explicit host-administrator configuration and do not vary by caller type. Recovery is limited to the evidence-preserving incomplete-tail procedure below. Any other repair remains a separately authorized administrative operation.

The configuration contains endpoint trust and grant metadata but no secrets. Its final path component MUST NOT be a symbolic link and it MUST NOT be writable by group or other. User enrollment writes it as `0600`. System enrollment writes it as administrator-owned `0644` so separately identified producers and readers can obtain the authority identity and public grants without gaining configuration mutation authority.

## Durable Ledger

STAV v1 uses one append-only ledger file per TOPS serialization domain. It has no automatic retention or rotation: the ratified policies are `preserve_all` and `disabled`. A configured finite maximum byte size is mandatory; append fails closed before exceeding it.

Each ledger record is exactly:

```text
uint32_be(event_length) || JCS(event) || SHA-256(JCS(event))
```

`event_length` is non-zero and no greater than 65,536 bytes. The final digest is 32 raw bytes and protects record framing; it is distinct from the domain-separated canonical event digest. The writer holds an exclusive non-blocking operating-system lock for the life of the open ledger, writes one complete frame, synchronizes the file, and only then emits a committed receipt. The empty file is a valid ledger with sequence zero and the per-TOPS genesis digest.

The authority creates the ledger as `0600`, refuses symbolic-link or non-regular ledger targets, and refuses an existing ledger with any group or other permission bits. The file lock serializes conforming authorities; restrictive filesystem permissions remain mandatory because an advisory lock cannot stop a non-conforming process that already has write access.

Startup scans every complete frame before accepting IPC. Each frame MUST pass its length, checksum, strict canonical event decoding, immutable TOPS ID, sequence, predecessor, and digest-chain checks. A complete malformed or inconsistent frame is corruption and prevents startup. Only an incomplete final frame caused by an interrupted append may be recovered automatically: its exact bytes are copied to a uniquely named recovery evidence file, the evidence file and containing directory are synchronized, the ledger is truncated to the last verified frame, and the ledger is synchronized before readiness. Status records that recovery occurred without exposing evidence contents. No middle-frame salvage, resynchronization search, or silent byte discard is permitted.

Startup reconstructs the request-ID idempotency index from canonical events. The candidate portion is re-derived from each event and its candidate digest recalculated; duplicate historical request IDs with different candidate digests are corruption.

## Integrity

SHA-256 digest inputs use an ASCII domain followed by one zero byte and the exact payload:

```text
candidate_digest = SHA-256("SYMPHONY-STAV-CANDIDATE-V1" || 0x00 || JCS(candidate))
genesis_digest   = SHA-256("SYMPHONY-STAV-GENESIS-V1"   || 0x00 || UTF8(canonical_tops_id))
event_digest     = SHA-256("SYMPHONY-STAV-EVENT-V1"     || 0x00 || JCS(event))
```

The first event references its per-TOPS genesis digest. Every later event references the immediately preceding event digest. Verification detects deletion, insertion, modification, and reordering within the available linear chain. This provides tamper evidence only; it does not authenticate the historical writer or prove non-repudiation.

A future Merkle transparency projection or signed checkpoint may be derived from canonical event digests after a separate threat-model ratification. It must not replace the v1 linear chain or alter canonical event bytes.

## Authorized SSIAG Outcome Classes

Safe metadata MAY represent authentication allowed/denied, policy allowed/denied, provider operation requested/completed, credential rotation, enrollment/re-enrollment, provider locked/unavailable, and lease issued/revoked.

Cryptographic proofs or assertions, raw tokens, credential values, provider payloads, secret-bearing native errors, policy contents, and routine heartbeat or performance telemetry MUST NOT be recorded.

## Query and Projection

Queries MUST be scoped to an authorized TOPS and apply redaction before output. The v1 query is forward-only and conjunctive. It supports an exclusive `after_sequence`, an optional inclusive sequence/time ceiling, optional time floor, no more than sixteen event classes, no more than five generic outcomes, optional correlation/request IDs, and a limit from `1` through `1000`. It has no raw expression, SQL, JSONPath, regex, descending order, raw offset, or unbounded mode.

Query pages are ascending redacted projections and identify source event ID, sequence, event digest, verification state, and redaction state. A projection is never hashed as though it were the canonical event. The response ceiling is 4,194,304 bytes and may reduce the page below the requested limit.

The authority applies query filters only after authenticating the reader and enforcing its classification allowlist. Events whose classification is not granted are omitted rather than partially redacted into misleading records. `qxctl` implements `stav status`, `stav verify`, `stav query`, and the client-side `stav doctor` composition through this read-only interface; it never gains append or direct-file authority.

## Local Frame Mechanics

The ratified local IPC frame is a four-byte unsigned big-endian payload length followed by exact canonical JSON bytes. Zero length is invalid. Requests are limited to 65,536 bytes and responses to 4,194,304 bytes; the length is validated before allocation. One request receives one response. Request operations are `append`, `status`, `query`, and `verify`, with exact payload unions defined by the canonical local-envelope schemas. `doctor` is intentionally not a server operation.

The listener accepts only authenticated local Unix-socket peers on supported Darwin and Linux TOPS nodes. The listener resolves the peer grant before dispatch, enforces a bounded per-request deadline, and returns only registered safe reason codes. STAV append/query IPC is not HTTP and is outside SACV/OpenAPI.

## Mutation Availability Policy

Security, credential, provider, policy, and configuration mutations using the ordinary audited path MUST fail closed when the append authority cannot accept the required audit event. A denied request remains denied when audit is unavailable; v1 MUST NOT create an ungoverned producer-side spool or secondary writer as a fallback.

No audit-deferred mutation path is implemented in STAV v1. A future target-host-administrator recovery protocol MUST be explicit rather than automatic, permission-backed without evaluating caller type, expected-state bound, and protocol-integrity preserving. It MUST durably record a local recovery journal before completing the operation, mark the result audit-deferred, and reconcile forward through the append authority when STAV returns. It MUST NOT edit the ledger, impersonate the original event time, hide the interruption, or permit a permanent unreconciled record.

Audit-deferred recovery and later reconciliation are administrative cold/freezing-path operations. They MUST NOT run inline with hot or warm execution, acquire locks shared with hot/warm paths, make hot/warm progress synchronously depend on STAV availability, or otherwise introduce blocking, jitter, or latency there. No vector-engine event class or producer grant is added by this rule; each requires separate STAV contract review.

## Operational v1 Boundary and Deferred Gates

Canonical candidate/event/receipt/query/query-page/verification content, serialization, SHA-256 domains, genesis construction, bounded query grammar, local stream framing, configuration/status/local-envelope content, peer authentication, producer and reader grants, authorization, listener activation, storage framing, fsync-before-receipt, crash recovery, idempotency reconstruction, preserve-all retention, disabled rotation, runtime projection policy, and native supervision are implemented and verified in the operational Go v1 foundation.

Signed checkpoints, remote export, non-repudiation, automatic retention or rotation, and general ledger repair remain deferred.

## Go Toolchain Migration

Production code and the root workspace remain pinned to Go 1.26.5. Go 1.27 is the intended target after general availability. Adoption requires final release-note review, byte-identical valid/invalid/JCS/digest corpus results, race and framing tests, and supported-TOPS cross-builds. New JSON or UUID standard-library facilities may replace only private kernel internals and only when they preserve every accepted value, rejection, canonical byte, digest, public API, and authority boundary. A toolchain update never changes the STAV wire protocol.
