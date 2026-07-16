# Symphony TOPS Audit Vector Specification

## Status

Owner-ratified v1 canonical content, read-message content, framing mechanics, append-authority architecture, and implementation namespaces. No runtime listener or writer is enabled until its remaining implementation gates pass.

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

`schemas/v1/common.schema.json`, `registries/v1/base.md`, and `fixtures/v1/` are also canonical. Canonical artifacts under `knowledge/stav/` outrank Go types, generated output, qxctl, or module code.

The following names remain reservations only and have no v1 content schema: `symphony.stav.append-authority.config.v1`, `symphony.stav.append-authority.status.v1`, `symphony.stav.local.request.v1`, and `symphony.stav.local.response.v1`. Their absence is intentional. Nothing may infer their fields or activate a listener from their names.

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

A receipt binds the request ID to the candidate digest. A rejected receipt carries only a safe registered reason and an explicit `not_committed` result. The schema describes the future committed form, but operational code MUST NOT emit `committed` until durability, acknowledgement, recovery, and idempotency retention are ratified and implemented.

## Append Protocol

Producers submit candidate events without a trusted ledger event ID, timestamp, sequence, or chain digest. One dedicated Go append-authority process per TOPS serialization domain authenticates the local producer, validates authorization, schema, presence, redaction, TOPS scope, reason codes, and size; assigns the trusted ledger identity and ordering values; writes atomically and durably; then returns a safe receipt. Concurrent submissions MUST serialize deterministically. Failure MUST NOT leave a valid-looking partial event.

The producer-to-authority transport is authenticated local IPC and is not HTTP or OpenAPI. Kernel-attested peer identity MUST map to an authorized producer subject. Socket permissions are defense in depth. The append authority's supervisor owns liveness only and MUST NOT gain producer or ledger authority.

Direct file mutation is prohibited for qxctl, producers, and agents. Recovery and repair require a separately ratified administrative procedure and must preserve evidence of the original failure.

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

qxctl implements the ratified grammar `qxctl stav query --tops-id UUID [--scope user|system] [bounded filters] [--json]` but opens no socket until reader authentication/authorization and local envelope content are ratified. `doctor` remains a client-side composition rather than a distinct server operation.

## Local Frame Mechanics

The ratified local IPC frame is a four-byte unsigned big-endian payload length followed by exact canonical JSON bytes. Zero length is invalid. Requests are limited to 65,536 bytes and responses to 4,194,304 bytes; the length is validated before allocation. One request receives one response.

This ratifies reusable framing mechanics only. It does not authorize a listener, define the local request/response envelope, select peer authentication, or choose ledger file framing. STAV append/query IPC is not HTTP and is outside SACV/OpenAPI.

## Mutation Availability Policy

Security, credential, provider, policy, and configuration mutations MUST fail closed when the append authority cannot accept the required audit event. A denied request remains denied when audit is unavailable; v1 MUST NOT create an ungoverned producer-side spool or secondary writer as a fallback.

## Remaining Implementation Gates

Canonical candidate/event/receipt/query/query-page/verification content, serialization, SHA-256 domains, genesis construction, bounded query grammar, and local stream framing are ratified and implemented in the authority-free protocol kernel.

Configuration/status/local-envelope content, peer authentication, producer and reader subjects, authorization, listener activation, storage/ledger framing, fsync and acknowledgement semantics, crash recovery, idempotency retention, retention, rotation, evidence-preserving repair, and runtime projection policy remain gated before enablement.

Signed checkpoints, remote export, and non-repudiation remain deferred.

## Go Toolchain Migration

Production code and the root workspace remain pinned to Go 1.26.5. Go 1.27 is the intended target after general availability. Adoption requires final release-note review, byte-identical valid/invalid/JCS/digest corpus results, race and framing tests, and supported-TOPS cross-builds. New JSON or UUID standard-library facilities may replace only private kernel internals and only when they preserve every accepted value, rejection, canonical byte, digest, public API, and authority boundary. A toolchain update never changes the STAV wire protocol.
