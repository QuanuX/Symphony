# Internal credential admission snapshots

## Status and boundary

SQV-17 implements `AdmissionRegistry` in `internal/credential/admission.go`.
It implements the dispatcher's `ProviderAdmission` interface by composing a
current owner-verified resource/lease snapshot with a required native admission
backend. The registry starts empty and refuses every use until trusted SSIAG
owner code publishes a complete snapshot. It reads no configuration, issues no
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
The registry has no serialized collection or fixed entry-count ceiling. This does
not change the repository's separate SKVI JSON capacity.

`Replace` is a trusted owner-only, process-local complete-snapshot CAS. Revision
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

The revision is tied to this registry object's lifetime. It is not a persistent
revision or a cross-process authority token. Restart creates empty state; a future
loader must recover current durable authority before publishing. No implicit
replay, lease reissuance, generation fallback or discovery of newest state exists.

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

Operational deployment still requires the persistent authority/lease owner,
authenticated native recipient and provider pin backend, separately versioned
protected channel, provider-outcome STAV integration and signed Keychain item
lifecycle. Existing provider v1 and synthetic-channel flags remain disabled.
