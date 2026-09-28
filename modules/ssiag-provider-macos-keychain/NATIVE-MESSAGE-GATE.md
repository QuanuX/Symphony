# Internal native XPC message gate

## Status and transport selection

SQV-19 selects macOS XPC for this bounded, one-way **nonsecret metadata** increment.
`NativeMessageGate.swift` consumes one inactive XPC connection supplied by trusted
native owner code. It creates no listener or service registration and is unwired
from provider-v1 and the Go dispatcher. The separate Swift process owns native
APIs; the cgo-free Go foundation and C++26 SQV consumers retain their boundaries.

This is an internal transport/profile selection, not a universal Symphony
transport rule, public operational protocol, credential release permission or
completed `ProviderAdmission` backend. Existing provider-v1 and synthetic-channel
flags remain disabled. Test app-bundled XPC services are disposable fixtures.

## Trusted inputs

- A validated `NativeMessageBinding`: canonical lowercase TOPS and request UUIDs,
  a canonical challenge UUID, and a lowercase SHA-256 digest of nonsecret admitted
  use-binding metadata. The native owner must derive these from current SSIAG
  authorization/journal state; incoming fields cannot establish their own authority.
- A `NativeMessagePolicy`: expected connect-time UID/GID and an independently
  selected native code requirement. Requirement text is bounded to 4,096 UTF-8
  bytes, contains no NUL, and must compile with Apple Security. Compilation does
  not establish policy provenance or strength.
- A finite positive lifetime no longer than 30 seconds, measured with a monotonic
  dispatch deadline, and a trusted nonblocking completion callback.
- Exclusive ownership of an inactive accepted connection with no prior target
  queue, event handler or peer-requirement installation. Reconfiguring an already
  activated/shared connection violates XPC's owner contract and is not supported.

Invalid value/lifetime/type inputs refuse before ownership is transferred.
After setup begins the gate owns connection cancellation, including setup failure.
The owner retains the gate until completion or explicit cancellation. Releasing
it early cancels resources without promising a completion callback.

## Implemented checks

Before activation, the gate installs the native peer code-signing requirement
using `xpc_connection_set_peer_code_signing_requirement`. XPC checks received
messages against that requirement. For each delivered dictionary, the gate also
obtains the actual message sender's code with `SecCodeCreateWithXPCMessage` and
validates the owner-selected requirement. A cached PID or an earlier socket
observation is not substituted for message identity.

The internal dictionary has exactly five string members:

| Member | Exact expected value |
| --- | --- |
| `format` | `ssiag-internal-native-message-1` |
| `tops_id` | owner-selected TOPS UUID |
| `request_id` | owner-selected request UUID |
| `binding_digest` | owner-derived nonsecret binding digest |
| `challenge` | owner-selected request challenge UUID |

Every member must have the exact XPC string type, byte length and value. Extra or
missing fields, binary data, typed-number substitutions, oversized strings and
any binding mismatch refuse. No generic JSON decoding, payload copy, FD transfer,
endpoint transfer or credential value is admitted. Framework/kernel allocations
before handler admission are outside this small-field validation bound.

The private serial queue orders message handling, monotonic expiry and explicit
cancellation. The first handled message spends the gate even when rejected. One
terminal transition cancels the timer and connection and calls completion once.
Outcomes are `matched`, `rejected`, `expired`, `cancelled` or `disconnected`.
Queued duplicates cannot re-open it. Security checks are followed by another
exclusive-deadline check; expiry cannot become a successful late match.

A wrong signature can be dropped by XPC before the handler; expiry/disconnection
therefore remains a valid refusal outcome. Native error descriptions, arbitrary
message fields, tokens, signatures and requirement text never enter completion.
`matched` reports only local metadata admission. This one-way gate sends no
receipt and does not prove acknowledgment or credential delivery.

## Principal and delivery boundaries still open

XPC's public connection UID/GID accessors report credentials **at connection
creation**. The implementation checks them but does not claim current per-message
UID/GID or continuity of one OS principal after connection transfer or credential
changes. Code identity is checked from each message's audit token; this alone is
not the complete SSIAG subject/connection-lifetime binding required for release.

The next integration must bind the actual sending principal and recipient,
current SSIAG subject, TOPS, lease, use binding, durable request reservation and
safe outcome audit. A challenge is correlation material, not a transferable
bearer grant. Creating another gate with the same values is not restart recovery;
the owner must enforce the existing durable journal and no-replay lifecycle.

No secret-bearing direction, exported descriptor, bounded byte-delivery framing,
receiver acknowledgment, Keychain operation, production signing policy or
provider/consumer launcher is implemented. Such additions need their own exact
contracts and tests. Native Security calls and owner callbacks are synchronous;
the enclosing process owner must independently enforce cancellation/termination
for a blocked callback. A dispatch timer is not an OS-level kill guarantee.

## Focused evidence

Tests build a temporary ad-hoc-signed app with a real separate Swift XPC service
and C++26 client. No launchd plist, Mach service registration, host install or
persistent daemon is created. Fourteen cross-process scenarios cover valid and
duplicate messages, wrong code/UID, every binding member, unknown/missing/type/
data/oversized fields and expiry. Fixtures retain the completed gate beyond its
original deadline to detect repeated completion. Separate tests cover malformed
owner input, invalid lifetimes and 64 concurrent cancellation requests. Fixture
signatures and constants do not establish production identity.
