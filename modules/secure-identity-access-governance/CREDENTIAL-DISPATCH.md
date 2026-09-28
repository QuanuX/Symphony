# Internal credential dispatch admission

## Status and ownership

SQV-15 implements the internal coordinator in `internal/credential/dispatch.go`.
It composes existing kernel peer authentication, the real policy engine and the
real SSIAG STAV producer with the [single-use guard](CREDENTIAL-USE.md). There is
no credential HTTP route, qxctl surface, production provider admission backend or
enabled Keychain operation. The provider boundary is exercised by test doubles;
kernel identity is exercised using real accepted Unix connections.

The coordinator is Go 1.26.5, cgo-free. The separate native provider and C++26 SQV
consumer retain their existing language and ownership boundaries.

## Implemented sequence

1. `Prepare` copies a bounded intent containing an authorization request, an exact
   credential reference and byte ceiling. There is no subject, executable digest,
   lease, receipt or authority field for the caller to supply. Preparation creates
   process-local state and grants no permission.
2. `Run` atomically spends the attempt before any action. It obtains the subject
   from SSIAG's private kernel-peer context and refuses missing or unmapped peers.
   It arms a context deadline bounded by the caller's deadline and 30-second
   internal profile. An already-cancelled or expired attempt cannot dispatch.
3. `policy.Engine.WithDecision` tries to acquire a policy read pin without
   queuing, validates request freshness, and evaluates the current exact policy.
   The pin remains held through the callback. A pending policy writer prevents
   new admission. An already-admitted operation releases its pin before a policy
   replacement commits; replacement cannot retroactively undo delivered bytes.
4. The concrete STAV producer submits the existing closed policy-decision event,
   verifies its committed receipt and exact candidate digest. The coordinator
   additionally verifies receipt TOPS and request identity. Denial or audit failure
   returns before invoking the provider admission backend. No provider/lease
   operation success is asserted by this policy audit.
5. The trusted provider admission backend pins the authenticated recipient and
   exact current resource-to-reference mapping, lease generation, provider binding
   and executable evidence. Its `Pin` operation is metadata-only. The coordinator
   compares TOPS, subject, kernel credentials, target and reference, and restricts
   the byte ceiling and deadline to the authorization and provider evidence.
6. It constructs and consumes one `OneShotUse` using the derived fields and the
   committed nonsecret receipt digest, then calls the pinned execution boundary
   with the bounded context. Every acquired pin is released, including a pin
   returned alongside an error. A provider error never appears in the result.
7. Cancellation after execution begins, or any unknown execution result, returns
   `indeterminate`. A fresh object, restored process or ambiguous result is not
   permission to repeat delivery. The result vocabulary carries no raw errors or
   credential bytes.

The initial policy snapshot now owns its own grant storage. Caller mutation of
the original config cannot alter active grants or limits without a digest-bound
replacement. This also applies to the existing authorization evaluator.

## Trusted backend obligations

`ProviderAdmission` and `PinnedUse` are first-party owner interfaces, not adapter
messages, plugins supplied by a request, or evidence accepted by possession.
They must verify the resource-to-reference relationship, current lease and
generation, signed consumer/provider identity and the actual connection lifetime.
Matching a PID or a digest-shaped string is insufficient.

The lock order is policy pin, provider/lease/recipient pin, execution, release
provider pin, release policy pin. Backend and audit callbacks must honor context
cancellation and must never reenter the policy engine or acquire locks in the
reverse order. Contexts provide cooperative cancellation; this coordinator cannot
forcibly terminate arbitrary in-process code. The native launcher/channel must
enforce termination, reaping, descriptor closure and byte bounds independently.

No production implementation of these provider interfaces ships in this
increment. The Go interfaces contain no secret buffer and cannot alone establish
protected delivery. Capability or receipt digests are always nonsecret evidence;
credential bytes and credential-derived hashes remain excluded.

## Gates still required for operational access

- Durable intent/outcome journal with authenticated request identity, duplicate
  suppression across newly created objects and restarts, and explicit recovery
  for indeterminate delivery. The atomic attempt only prevents reuse of itself.
- Provider/lease/recipient pin implementation coordinated with revocation,
  rotation and exact namespace selection; no implicit generation fallback.
- Authenticated protected channel, bounded native child termination and consumer
  handling; byte capacity is currently passed as a contract, not enforced by an
  implemented channel.
- Signed macOS Keychain import, lookup, rotation/revocation and user-session
  behavior under the selected deployment's TOPS and access-group identity.
- Durable provider-outcome auditing and full delivery/recovery conformance before
  admitting a versioned operational endpoint. Existing metadata protocols and
  their disabled flags remain unchanged.

## Focused evidence

Tests cover real kernel identity mapping, independently derived authority,
committed audit-before-provider ordering, receipt candidate/TOPS/request mismatch,
target/generation/peer drift, malformed provider evidence, cancellation before
and after dispatch, exact deadline propagation, cleanup, and 64 simultaneous
attempts with exactly one execution. Policy tests cover snapshot pinning,
non-queuing busy refusal, replacement after release and caller-config ownership.
Existing server authorization and policy-administration tests cover the affected
shared engine boundaries. These tests do not prove operational Keychain access.
