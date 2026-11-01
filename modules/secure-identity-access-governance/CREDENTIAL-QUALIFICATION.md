# Credential-use integration and operational qualification

This is the L14 integration handoff. Test profiles and an authorized TWS SDK
were unavailable when this increment was prepared. It is not operational
credential qualification or permission to retrieve a real secret.

## Implemented owner sequence

The cgo-free dispatcher authenticates the kernel peer, pins exact current policy,
reserves the original request durably and commits its policy decision to STAV.
`AdmissionRegistry` then requires an owner-published resource/lease snapshot and
a native backend that pins the actual recipient and provider. Before execution,
the journal stores the complete nonsecret one-shot binding and a distinct
provider-outcome audit request ID. Execution is bounded by caller, policy, lease
and native expiry. Provider resources are released on every path.

After execution, the observed result is durable before its separate STAV event
is submitted. A genuine committed receipt must match TOPS, request and candidate.
Missing acknowledgement or receipt persistence leaves reconciliation required;
freshly authorized recovery replays that same safe candidate without accessing
the provider. Audit commitment does not prove the recipient successfully used a
credential, and an indeterminate delivery remains indeterminate.

`CREDENTIAL-JOURNAL.md` records format-1 compatibility, format-2 evidence, crash
boundaries and conservative recovery. No HTTP credential route or operational
provider-v1 flag is enabled by this increment.

## Native and deployment work still required

1. The SSIAG owner must persist exact resource/lease generation state, issue and
   revoke it under current authority, and republish only recovered current state
   after restart. The existing process-local registry starts empty. A busy
   replacement is not completed revocation.
2. Implement and qualify the native admission backend's independent provider
   identity, recipient executable/connection lifetime and bounded protected
   one-shot delivery. A PID, digest-shaped assertion or fixture pin is not proof.
   Enforce actual byte ceilings, deadline cancellation, descriptor closure and
   native child termination/reaping; Go contexts alone cannot guarantee these.
3. On Apple Silicon, complete the recorded Phase10C data-protection Keychain
   item/operation/access-control policy, deployment signing identity and actual
   user-login context. Metadata/signature readiness is not operational access.
4. On Ubuntu, implement the separately identified Linux Secret Service provider
   in the canonical provider order. A headless workload/hardware identity is a
   separate explicit profile. Do not fall back to plaintext, `pass`, an invented
   vault, another user's session or a different provider. Ubuntu ARM64 execution
   suffices for this sprint; physical DGX Spark/GPU claims remain unqualified.
5. Bridge the native consumer to `source::CredentialUse` and Databento's
   `SsiagHistoricalUse` with the exact request reference, lease, recipient and
   generation. Keep borrowed bytes within the native callback and overwrite
   locally owned transient buffers. Do not promise erasure of framework copies.

These are remaining implementation and deployment gates, not merely missing test
credentials. IBKR's already-authenticated external TWS/Gateway sessions have
their own owner boundary; a Session object does not supply login authority.

## Local regressions

Use the repository's pinned Go1.26.5 toolchain and source workspace, with private
caches and `CGO_ENABLED=0` for production builds. Run the credential package with
race detection and no cached results. The tests include actual kernel peers,
filesystem locks, crash recovery, audit/refusal/expiry checks, format migration,
candidate substitution and receipt persistence failures. The process regression
builds the workspace-selected or independently pinned STAV authority, serves only a private temporary Unix socket,
loses an acknowledged outcome, restarts the authority and checks that the
original receipt is returned without a second delivery or outcome append.

Provider callbacks in these regressions remain synthetic. Hosted Apple Silicon,
Ubuntu x86_64 and Ubuntu ARM64 runs must be reviewed against the exact source.

## Later authorized operational run

Select an existing deployment/profile and exact immutable provider/consumer
installations; record nonsecret receipt, executable, source and platform evidence.
Select a specific test account, exact operation, time window, entitlement and
request/spend ceiling. Configure exact STAV producer grants for policy decisions
and provider operations; audit configuration is never implied by a binary install.
Use the approved native import/provisioning path after that path is implemented.
Never supply credential values through qxctl, command arguments, environment
variables, logs, manifests, test examples or this handoff.

For every selected target and provider, record both one successful bounded use
and refusal for wrong recipient/reference/generation, expired or revoked lease,
deadline/cancellation, unavailable audit, interrupted delivery and lost outcome
acknowledgement. Verify cleanup and restart; recovery must preserve uncertainty
and must not automatically retry acquisition. Compare real provider responses
and entitlements independently from internal audit/transport success. Retain
safe evidence only, including which checks were not performed. Credentials and
provider payloads never belong in STAV or repository qualification artifacts.
