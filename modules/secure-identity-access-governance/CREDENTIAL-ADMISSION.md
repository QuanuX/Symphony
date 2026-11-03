# Internal credential admission snapshots

## Status and boundary

SQV-17 implements `AdmissionRegistry` in `internal/credential/admission.go`.
It implements the dispatcher's `ProviderAdmission` interface by composing a
current owner-verified resource/lease snapshot with a required native admission
backend. The process-local constructor starts empty; `OpenAdmissionRegistry` restores
a protected exact-owner snapshot. Both refuse use without a current matching
grant supplied by trusted SSIAG owner code. It reads no configuration, issues no
lease, creates no endpoint and enables no provider operation.

The native backend remains unimplemented for operational delivery. Test backends
use safe fixture metadata and no credential bytes. A valid digest or this internal
snapshot is not proof of signed executable identity, connection lifetime, Keychain
access or audit commitment. The dispatcher still owns policy and STAV admission;
the native backend must independently verify and retain the actual provider and
recipient/channel before returning a pin.

## Snapshot contents and replacement

One immutable TOPS owns the registry. Each value binds the canonical subject and
kernel UID/GID, exact operation/resource/audience/scope, full provider/name/type/
generation reference, nonsecret consumer and provider-binding digests, a lease ID,
UTC not-before/expiry, and byte ceiling. PID is observed per connection, never
stored as durable subject authority. Strings and value fields are copied into
private storage; retaining or changing the caller's input cannot mutate it.

The current profile permits one active reference for each subject/target tuple
and one entry per lease ID. Ambiguous entries reject the entire replacement.
Tokens/digests use the existing internal grammar; byte bounds remain 1–4,096.
The internal profile permits at most 64 active entries and a 1 MiB serialized
snapshot. Overflow rejects the whole replacement without changing durable bytes.
Reserved UID/GID ownership sentinels are refused. These limits do not change
the repository's separate SKVI JSON capacity.

`Replace` is a trusted owner-only complete-snapshot CAS. Revision
starts at 1 and advances on each successful replacement, including identical
values. Stale revisions fail; the counter never wraps. Empty state removes all
admission. Expired or malformed entries reject the whole candidate. Future leases
may be published but cannot admit use before their not-before time.

Replacement never waits for an active reader: `busy` means no change committed.
An existing delivery and native cleanup finish before replacement can succeed.
A busy response must not be described as completed revocation or rotation. It
neither cancels an admitted delivery nor promises writer fairness. A trusted
lifecycle owner must arrange retry/drain and audit publication before reporting
its own operation complete. No such administration route is enabled here.

`NewAdmissionRegistry` retains process-local revision semantics. The optional
`OpenAdmissionRegistry(stateDir, topsID, authorityDigest, native)` constructor
holds an exclusive lifetime lock and restores the same revision and exact grants.
The authority digest must come from current protected SSIAG owner configuration,
not a request. Mismatched authority refuses rather than resetting or reissuing.
No implicit replay, lease reissuance, generation fallback or newest selection exists.

## Protected persistence

Internal state is `credential-admission/<tops-id>/active.json` with a persistent
`active.lock`, below the existing private SSIAG state directory. This is nonsecret
admission metadata, not a credential vault, public schema, bearer grant or audit
ledger. The Go owner uses the existing descriptor-relative, no-follow filesystem
primitives with owner-only directories/files, one cooperating writer, strict
canonical encoding, a domain-separated digest and TOPS/authority/revision binding.
The request journal retains its separate namespace and 8 KiB record limit.

Replacement writes and syncs a private temporary file, atomically renames it, and
syncs the directory before activating the in-memory snapshot. Any persistence
error fences all future pin/update attempts in that registry: the old grants may
no longer match disk. Close and independently reopen/reconcile the actual durable
state; an error never proves that a requested revocation committed. Missing state
beside an existing lock, missing lock beside state, malformed/unknown/duplicate
fields, altered bytes, unsafe permissions and symlinks refuse recovery.

`Close` refuses while native pins are held. It releases the persistent lock only
after native cleanup, does not delete state, and is idempotent. Expired recovered
grants remain retained but cannot reach the native backend. Every actual dispatch
still requires current policy/STAV authority and independent native recipient and
provider validation. The snapshot alone issues no lease and proves no credential
operation. Cooperative private storage does not provide rollback resistance
against the host owner replacing an entire old snapshot and lock; no such
anti-rollback or power-loss qualification is claimed.

## Admission and use

1. Require the same mapped kernel peer from the private context; refuse fabricated
   peer fields or unsupported subject attributes. Validate the complete intent.
2. Acquire a non-queuing snapshot read pin. Match subject, target, UID/GID, exact
   reference, byte ceiling and current lease time before invoking native code.
3. Bound native admission by the caller context, request deadline and lease expiry.
   Native admission is metadata-only. It must retain the actual signed provider,
   authenticated recipient and channel resources without reading key material.
4. Compare every native evidence field to the owner snapshot and observed peer.
   Native byte and expiry restrictions may be stricter but cannot be larger or
   later than the snapshot. Failed native pins are released even when returned
   alongside an error. Native errors never escape the wrapper.
5. Retain the state pin through execution and native release. Before the single
   execution, recheck the exact request/correlation, subject/target/reference,
   lease, digests, byte count, times and unchanged native evidence. Policy, config,
   authorization and audit evidence remain the dispatcher's responsibility.
6. Context cancellation from either admission or execution reaches native execution;
   the effective deadline cannot exceed any bound. Cancellation after entry or an
   unknown native outcome becomes `indeterminate`. Repeated execution is spent.
7. Release is idempotent and serialized with execution; it waits for an executing
   callback, releases native resources, cancels its context, then drops the state
   pin. It cannot forcibly interrupt arbitrary in-process code. Native callbacks
   must honor contexts; the future child/channel owner must enforce OS termination,
   descriptor closure and reaping independently.

Lock order remains policy, request journal, admission snapshot, native pin,
execution. Reverse-order or reentrant callbacks are unsupported. Owner mutation
must not wait for policy authority while holding an admission write lock.

## Verification and remaining integration

Focused tests exercise dispatcher composition, retained snapshot ownership,
rotation and revocation while pinned, stale CAS and overflow, real kernel-peer
context, native evidence mismatches, expiry/cancellation, native error cleanup,
concurrent one-shot execution, concurrent replacement and cleanup ordering.

Restart, crash/lock release, durable revocation, authority drift, capacity, corruption
and uncertain-write regressions cover the persistence primitive. Operational
deployment still requires permission-backed lease issuance/revocation and audit wiring,
authenticated native recipient and provider pin backend, separately versioned
protected channel, qualification of provider-outcome STAV integration and signed Keychain item
lifecycle. Existing provider v1 and synthetic-channel flags remain disabled.
