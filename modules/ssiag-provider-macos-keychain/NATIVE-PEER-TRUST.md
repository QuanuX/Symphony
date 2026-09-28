# Internal native peer-code observation

## Status

SQV-18 adds `NativePeerTrust.swift` as an internal, unwired prerequisite for
operational native admission. It uses real macOS kernel socket identity and Apple
Security code validation. No provider-v1 operation calls it; no new protocol,
endpoint, command, Keychain operation or operational flag is enabled.

It is a time-specific observation, not a complete credential-delivery pin. It
cannot prevent a peer from execing or transferring descriptors after a check.
The Go `AdmissionRegistry` still requires an operational native backend. This
component must not be used as that backend merely because validation succeeded.

## Trusted input and identity

The native owner supplies an existing connected Unix stream and a policy that
binds expected effective UID/GID and a native code requirement. The owner must
establish policy provenance independently; compiling requirement text proves
syntax, not authority or policy strength. Peer messages, JSON declarations and
caller-shaped digests cannot supply operational signing policy.

The current requirement input is bounded to 4,096 UTF-8 bytes with no NUL.
`SecRequirementCreateWithString` compiles it. There is no PID, process path, raw
audit token, signing-label shortcut, lookup by latest executable or permissive
fallback input. Production policy must still meet selected deployment signing,
receipt, hardening and identity requirements. Test requirements are explicitly
isolated ad-hoc fixture requirements and confer no production eligibility.

## Implemented observation

1. Duplicate the supplied descriptor with `F_DUPFD_CLOEXEC`. The caller must keep
   its descriptor stable during duplication. The observation owns its duplicate;
   closing or reusing the caller's original number cannot redirect it.
2. Require a connected `AF_UNIX`/`SOCK_STREAM` socket. Refuse regular files,
   unconnected sockets, datagrams and observed socket errors/hangup. No socket
   payload is read or written by this component.
3. Obtain the complete audit token from `LOCAL_PEERTOKEN`. Use Apple's `libbsm`
   accessors for effective UID/GID/PID. Require the exact configured UID/GID and a
   non-self process. A socket pair created before launching another executable
   does not by itself attest that later executable as its peer.
4. Resolve running code using `SecCodeCopyGuestWithAttributes` with
   `kSecGuestAttributeAudit`. Validate that code with `SecCodeCheckValidity` and
   the owner-selected requirement. Native errors are replaced with closed
   internal error cases; raw token and requirement remain private.
5. Keep the owned socket, original token and compiled requirement in the session.
   Each `revalidate` checks the socket and exact token again and resolves a fresh
   code object from the original token. It never refreshes authority from a PID
   or silently replaces the original token after exec.
6. A failed revalidation closes the session permanently. Close is idempotent;
   normal teardown also closes the descriptor. A private lock serializes close
   and validation across threads. No descriptor or serializable token is exposed.

The internal identity result contains UID/GID/PID only. PID is observation
metadata, not a stable subject identifier or a reusable authorization token.
There are no credential bytes, secret-derived hashes, logs or audit events.

## Limits required for later integration

The selected socket's audit token attests the identity associated with that
connection; it does not prove who will receive every later byte. Code validation
and subsequent I/O are not atomic. A peer may close, exec or transfer a descriptor
immediately after success. Socket poll is only an observation and does not prove
future readiness or detect every half-close condition. No process freeze or
continuous validity guarantee is claimed.

The operational transport must authenticate the actual request/message and
receiving operation, handle any descriptor transfer, enforce deadlines and byte
limits, and independently terminate/reap children. Native Security calls here are
synchronous and not forcibly cancellable in-process. This session implements no
SSIAG policy, TOPS mapping, lease, request binding, durable outcome or delivery.
Those remain separate owners/gates.

## Verification

Tests use actual accepted Unix connections to a separately compiled C++26 helper
with an ad-hoc fixture signature. They verify matching identity and native policy,
wrong UID/GID/requirement, invalid requirements/descriptors, self-socket refusal,
owned descriptor lifetime, queued-data preservation, successful/failed cleanup,
process exit, socket closure while the process survives, and concurrent close.
The exec test re-executes the SAME signed binary under the SAME PID while retaining
its socket: the old audit-token generation must no longer validate.

The helper lives under `Tests/Fixtures`, has no production target and handles only
fixed test markers. Swift tests compile and sign a disposable copy locally; they
do not use a production signing identity or Keychain item. Apple Security and
`libbsm` remain in the separate Swift provider, outside the cgo-free Go foundation.
