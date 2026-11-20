# Credential-use integration and operational qualification

This is the L14 integration handoff. Test profiles and an authorized TWS SDK
were unavailable when this increment was prepared. It is not operational
credential qualification or permission to retrieve a real secret.

Scope clarification: this handoff retains the general L14/provider-v1 integration
gaps and its recorded test boundary. The later Cloudflare I4 route implements
separate native use through `/v1/cloudflare/use` and the opt-in
[`ssiag-provider-private-store`](../ssiag-provider-private-store/SPEC.md), with
local native/SSIAG fixture qualification. It does not complete the general
dispatcher, Apple Keychain, Linux Secret Service or native source/Databento
consumers listed below, and it does not establish live Cloudflare acceptance.

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

1. Protected exact-owner admission persistence now restores revisions and grants
   after restart, fences uncertain writes and retains committed empty revocation
   snapshots; see `CREDENTIAL-ADMISSION.md`. The SSIAG lifecycle owner must still
   wire permission-backed issuance/revocation and safe audit commitment into this
   internal primitive. The process-local constructor remains available for bounded
   composition. A busy or unavailable replacement is not completed revocation.
2. Implement and qualify the native admission backend's independent provider
   identity, recipient executable/connection lifetime and bounded protected
   one-shot delivery. A PID, digest-shaped assertion or fixture pin is not proof.
   Enforce actual byte ceilings, deadline cancellation, descriptor closure and
   native child termination/reaping; Go contexts alone cannot guarantee these.
3. On Apple Silicon, qualify the [development Keychain exercise](../ssiag-provider-macos-keychain/Development/README.md)
   in the selected user's login session. It exercises a generated generic password
   in the signed app's private Keychain access group. The production adapter still needs its own
   signed identity, item operations, access-control policy and SSIAG integration.
4. On Ubuntu, integrate the [Linux Secret Service backend](platform/linux-secret-service/)
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

## General lease lifecycle on macOS and Linux

The two native store implementations exercise item access. SSIAG must also
establish who may authorize a specific executable to use one item generation.
The following work belongs to the shared credential owner.

1. Define the lease proposal, apply and recovery permissions under the
   [SSIAG administration contract](../../knowledge/ssiag/SPEC.md). Bind the
   kernel-authenticated manager and intended beneficiary to the same TOPS,
   approved use operation, resource, audience, scope, byte ceiling and expiry.
   Specify who may revoke a lease. A credential-use grant alone cannot grant
   authority to manage someone else's lease.
2. Resolve the resource through protected owner configuration to its exact
   provider installation, item generation and recipient executable. Bind the
   resulting plan to its expected admission revision. Apply must reauthenticate
   the manager and recheck current permission, expiry and that mapping. Exercise
   identity remapping, stale proposals, wrong generations and replacement of a
   provider or consumer between proposal and apply.
3. Persist the attempt and its audit request identities before mutation. Use a
   committed STAV policy-decision receipt and the existing lease lifecycle event
   types. Define when an issued grant can admit use and when revocation is
   complete relative to the outcome receipt and durable registry revision. Test
   active pins, a crash at each transition, a lost append acknowledgement and
   an uncertain snapshot write. On restart, reconcile the original candidate
   and revision before admitting affected use. Recovery must preserve expiry
   and cannot mint another lease or repeat delivery. Serialize operations on the
   same item generation through the credential owner. Secret Service clients in
   the session can change item attributes between lookup and native use.
4. Hold the authenticated provider and recipient connection through the actual
   native use. Bind the operating-system peer, executable, subject and lease to
   that connection. Exercise wrong recipients, connection replacement,
   cancellation and cleanup on each platform with generated test bytes. A
   completed store lookup alone cannot establish delivery or consumer use.

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
