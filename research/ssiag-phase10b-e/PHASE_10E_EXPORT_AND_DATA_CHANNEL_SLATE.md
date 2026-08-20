# Phase 10E — Export and Protected Data-Channel Design Slate

> **Noncanonical research:** This slate is archived design provenance, not an
> authorization or implementation claim. Current canonical SSIAG contracts
> take precedence, and Phase 10E remains unratified unless recorded there.

## Status

Design last; implementation prohibited. Provider v1's inherited descriptor
remains synthetic (`-1`, zero bytes, nonoperational), and all secret-channel
flags remain false. This phase begins only after signed-bundle trust, item
lifecycle, non-exportable operation, authorization, crash, and audit gates are
operationally proven.

## First Question: Is Export Actually Required?

Every use case must first answer whether the provider can perform the operation
internally. If sign, decrypt, assert, or exchange can remain inside the
provider, no secret export channel should exist. Export is a distinct
capability and permission, never implied by storage, lookup, or provider
readiness.

## Separate Channel Classes

Do not overload one generic pipe:

| Class | Direction | Content | Status |
|---|---|---|---|
| control | bidirectional request/response | safe metadata only | existing v1 |
| public result | provider to authenticated consumer | bounded signature/public key | Phase 10D decision |
| secret import | authorized source to provider | secret bytes | Phase 10E |
| secret export | provider to authorized sink | secret bytes | Phase 10E |

One channel has one direction, one request, one producer, one consumer, one
maximum, one deadline, and one close. Bidirectional secret negotiation is
prohibited in the first design.

## Candidate Operational Descriptor

The operational descriptor must use a new exact schema/protocol rather than
changing v1 constants. Candidate members:

```text
protocol
channel_id
request_id
correlation_id
tops_id
provider_name
adapter_identifier
provider_binding_digest
operation_id
authorization_binding_digest
direction
descriptor_number
maximum_bytes
exact_bytes_or_null
created_at
deadline_at
single_use: true
replay_permitted: false
retry_permitted: false
close_after_delivery: true
secret_bytes_in_control: false
descriptor_digest
```

The descriptor number is process-local metadata, not proof. The inherited
descriptor and already authenticated process relationship provide the local
transport binding. A path-based temporary file, named socket, environment
value, command argument, shell pipeline, qxctl stdin/stdout, or OpenAPI request
is not an acceptable substitute.

## Candidate Transport and Framing

Recommendation for later ratification:

1. Go creates an anonymous local socket pair before process creation.
2. Only the exact endpoint required by direction is inherited by the adapter;
   all unrelated descriptors are close-on-exec.
3. Control JSON carries the digest-bound descriptor metadata, never bytes.
4. The sender writes one fixed header containing protocol version, channel ID,
   and unsigned length, followed by exactly that many bytes, then half-closes.
5. The receiver rejects zero, oversized, short, long, duplicate, trailing, or
   late frames and closes immediately.
6. No secret-derived checksum is logged, persisted, audited, or returned. If an
   in-memory integrity check is later needed, its value dies with the channel.
7. Cancellation closes both endpoints, terminates the adapter process group,
   waits for cleanup, and returns an indeterminate category when commit state
   cannot be proven.
8. Neither endpoint is reused, pooled, reconnected, or retried.

The eventual schema must choose a maximum per capability rather than one large
universal ceiling. Apple describes Keychain Services as storage for small
chunks of data; large payload storage is not an intended provider use.
[Keychain Services](https://developer.apple.com/documentation/security/keychain-services)

## Secret Import Ownership Gap

qxctl must not accept a secret flag, environment variable, config field,
clipboard value, JSON value, or normal stdin stream. Therefore import requires
an independently authenticated local secret source that can supply a protected
descriptor directly to SSIAG. That source, its authority, peer authentication,
process trust, consent, and recovery semantics are not designed yet.

Until such a source exists, general secret import remains unavailable even if
the adapter could receive a descriptor. An administrator may provision through
the provider's native, separately documented facility, but qxctl must not
pretend that out-of-band provisioning was a Symphony operation.

## Secret Export Sink Gap

The Go foundation must not read secret bytes merely to return them through its
HTTP response. A future export requires an authenticated sink descriptor bound
to the exact authorized operation. The preferred topology forwards provider
bytes directly to that sink while SSIAG controls lifecycle and verifies safe
completion metadata.

The sink must prove:

- kernel identity and current SSIAG subject mapping;
- exact operation/audience/scope and nontransferable authorization;
- protected executable/binding identity where required;
- willingness and ability to receive within the deadline and maximum;
- no qxctl, OpenAPI, STAV, log, or persistent-file hop;
- explicit handling of partial/ambiguous delivery.

No export API should ship until this receiving boundary exists.

## Memory and Crash Policy

Swift `Data`, Core Foundation, Security framework, Go slices, and kernel socket
buffers may create copies. Perfect erasure cannot be claimed. The exact policy
must document and test:

- bounded allocation before reading;
- no conversion to `String`;
- minimum copies and lifetime;
- explicit zeroization of Symphony-owned mutable buffers where the compiler and
  platform guarantee can be demonstrated;
- locked memory only if supported, bounded, and failure behavior is ratified;
- production core dumps disabled for every process that may handle bytes;
- no diagnostic attachment, tracing, panic dump, crash reporter field, or
  telemetry capture;
- close/zeroize on success, failure, cancellation, timeout, and broken pipe;
- the residual exposure that framework and kernel copies cannot be proven
  erased immediately.

The existing threat model correctly says perfect erasure cannot be guaranteed;
Phase 10E must preserve that honesty.

## Authorization, Audit, and Retry

- obtain a fresh exact SSIAG decision before creating the descriptor;
- bind authorization to direction, reference, generation, byte maximum,
  purpose, consumer/source identity, and deadline;
- require committed safe STAV precondition under the ratified provider/lease
  event ordering;
- audit only safe references and outcome categories;
- never audit byte length if it becomes sensitive for the use case without an
  explicit classification decision;
- never audit a content digest;
- import mutation follows the complete Phase 10C journal and reconciliation
  circuit;
- export delivery is never retried automatically;
- timeout, process death, short write/read, or lost completion after bytes
  moved produces an explicit indeterminate result requiring a new authorized
  operation, not replay.

## Safe Error Categories

The consumer sees only stable values such as `not_exportable`,
`source_unavailable`, `sink_unavailable`, `channel_invalid`, `channel_timeout`,
`partial_delivery`, `interaction_cancelled`, `item_locked`,
`authorization_expired`, `audit_unavailable`, or `internal_failure`. Native
error text and secret-shaped details remain inside the adapter and are not
logged.

## Negative Test Gate

- descriptor injection, reuse, wrong number, wrong direction, and inherited-FD
  confusion;
- request/channel/TOPS/provider/operation mismatch;
- zero, maximum, maximum-plus-one, short, long, trailing, and multi-frame input;
- EOF before/after header and cancellation at every byte boundary;
- adapter/foundation/source/sink crash at every lifecycle state;
- expired authorization, changed policy/config/provider binding, and logout;
- cross-user, cross-session, cross-TOPS, and fast-user-switch attempts;
- accidental stdout/stderr/HTTP/qxctl/STAV/journal/environment/argument/file
  exposure;
- core/crash/log artifact marker scans;
- compiler-optimized zeroization review;
- concurrency exhaustion and descriptor leaks;
- import commit with lost response and export partial-delivery recovery;
- operational v2 rejection by v1 binaries and exact side-by-side version
  selection.

## Architect Ratification Slate

- **10E-A:** concrete use cases that truly require import or export;
- **10E-B:** authenticated secret source and sink ownership;
- **10E-C:** new operational protocol major and exact descriptor schema;
- **10E-D:** anonymous transport, framing, per-capability byte maxima, and
  deadline;
- **10E-E:** authorization tuple and whether a lease is mandatory;
- **10E-F:** qxctl exclusion and any separate headless consumer interface;
- **10E-G:** memory, zeroization, locked-memory, core-dump, and crash-report
  policy;
- **10E-H:** STAV precondition/result ordering and safe length classification;
- **10E-I:** partial/indeterminate delivery and no-retry semantics;
- **10E-J:** side-by-side protocol/version compatibility and rollback.

Until every applicable decision passes implementation and leakage tests,
provider v1's synthetic descriptor remains the only descriptor and secret
delivery remains impossible.
