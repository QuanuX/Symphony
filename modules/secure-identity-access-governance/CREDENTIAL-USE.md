# Internal credential-use guard

## Implementation status

SQV-14 adds an internal, process-local lifecycle primitive in
`internal/credential/use.go`. SQV-15 composes it in the internal
[dispatch admission coordinator](CREDENTIAL-DISPATCH.md). It is not connected to
a service route, operational provider, lease issuer or secret channel. Existing provider v1 control and
synthetic channel schemas retain their disabled operational flags. This document
describes the implemented primitive and the outstanding integration work; it does
not admit a new public protocol or a production Keychain provider.

The SSIAG foundation remains Go 1.26.5 with cgo disabled. The separately installed
macOS provider retains its Swift platform boundary; SQV consumers remain C++26.

## Implemented behavior

`NewOneShotUse` copies a validated value binding. It includes the canonical TOPS
UUID; subject ID, kind and authority; exact consumer, provider-binding, policy,
configuration, authorization and committed-audit evidence digests; lease, request
and correlation IDs; provider, credential name, explicit generation and type;
operation, resource, audience and scope; byte ceiling; and UTC issue/deadline.
An empty generation is invalid. No implicit current/latest generation is resolved.

The current internal profile admits 1–4,096 bytes and a positive lifetime up to
30 seconds. These are implementation ceilings for this primitive, not a universal
credential size or policy limit. Any later transport must enforce its own byte
count and deadline and can impose stricter limits. Metadata tokens follow the
existing authorization-token grammar: 1–256 ASCII letters, digits or `._:-`.
Digests use lowercase `sha256:` plus 64 hexadecimal digits and refer only to
nonsecret evidence. Syntax checks do not detect secrets or prove authority.

`Consume` atomically spends the guard before validating the current binding.
Concurrent attempts permit at most one successful result. Every field must still
match; cancellation, invalid context/clock, expiry or drift also spend the guard.
The deadline is exclusive. A clock earlier than the issue time fails closed.
Timestamps are UTC; monotonic components are removed for value comparison. The
future dispatcher must supply trusted time and bound I/O using a monotonic timeout;
this primitive alone does not protect against host clock changes.

The guard has no secret buffer, exported serializable state, persistence or
restoration method. Its pointer must remain private to its owning dispatcher and
must not be copied after construction. A new guard requires independently renewed
authority; recreating it from caller claims is not a retry mechanism. Restart or
ambiguous delivery must never imply that a previous attempt was unused.

## Required integration before credential release

1. An enrolled TOPS and kernel-authenticated consumer establish the subject and
   connection. The dispatcher independently verifies executable/provider evidence,
   exact current policy authorization and the committed safe audit receipt. A
   digest-shaped string or caller-created `UseBinding` grants no permission.
2. The dispatcher maps the authorized resource to the exact provider, credential
   generation, namespace and consumer. It must keep that authority snapshot stable
   through dispatch, or invalidate admission when policy/provider/generation changes.
3. A separately reviewed provider operation opens a protected one-shot channel to
   that same authenticated recipient, enforces byte/time bounds, terminates on
   cancellation and closes every descriptor. Control JSON, qxctl, logs, STAV and
   knowledge files never carry credential bytes or credential-derived digests.
4. Durable intent/outcome handling distinguishes refusal, cancellation, committed
   delivery and indeterminate delivery. The in-memory guard is not a crash journal.
5. The macOS provider needs real item creation/import, exact-generation lookup,
   replacement/revocation and user-interaction handling, with verified signing,
   private access group, data-protection Keychain selection and per-TOPS namespace.
   The current readiness observer performs none of these operations.

These are implementation dependencies, not requirements to create new qxctl
surfaces. The Databento acquisition mapping remains owned by SQAV with SSIAG
credential authority; source provenance stores safe references only.

## Focused verification

`internal/credential/use_test.go` covers exact binding, independent caller-value
ownership, valid drift of every field and each reference member, malformed input,
deadline boundaries, rollback before issue, cancellation, replay, zero/nil guards,
and 128 concurrent attempts. A race-enabled test run complements the cgo-free
test run. These tests verify this local primitive; they do not verify provider
authentication, Keychain item access, secure delivery or crash recovery.
