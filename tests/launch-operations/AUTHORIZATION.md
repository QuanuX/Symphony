# Account and authorization-to-effect qualification

## Qualified and unqualified profiles

The MVP procedure uses a local Unix account with caller-owned private state, exact receipt-bound executables, authenticated Unix-socket peers and explicit SSIAG grants. Same-UID processes share OS-level ownership: a process acting as that UID can modify that user's files. Digests, private modes and cooperating locks do not isolate hostile code running as the same owner. Do not advertise this profile as tenant isolation or protection from the owner/root.

A distinct service account is a separate deployment profile. Do not make it work by sharing private roots, relaxing modes or overriding a peer UID. Exact configured service identity, socket access, parent traversal, STAV owner, SSIAG subject/grants and every retained-state owner must be independently qualified. This increment does not claim a successful multi-account deployment. It tests rejection of an actual differently owned directory and authenticated peer mismatch; those refusals are not a complete cross-account qualification.

## Evidence matrix

| Boundary | Executed regression surface | What it proves |
| --- | --- | --- |
| Real kernel owner mismatch | `snvstate.TestSNVActualDifferentDirectoryOwner` | Non-root caller refuses a root-owned directory without altering system files |
| Socket identity and exact TOPS/config binding | `ssiagclient.TestPeerVerificationFailsBeforeHTTPBytes`, `TestSocketOverrideRetainsConfiguredPeerIdentity`, `TestAuthenticatedClientReadsStatusOverUnixSocket`, `TestConfigTOPSAndScopeBinding` | Peer/config checks and actual local authenticated transport; fixture-verifier cases remain labeled in their tests |
| Denial, expiry, wrong resource/correlation/capability | `snvstate.TestSNVSelectionGuardAndRecovery`, `TestSNVAuthorityBinding`; SHV source/publication `CASReplayAndHistory`, `AuthorizationBinding`, `InterruptedPublicationAndExpiry` | Store admission and exact decision binding under structural test decisions |
| Revoked/changed final guard | SNV selection guard; SCV source interrupted authorization/expiry; SHV source/publication interrupted tests | Refusal before effect and retained recovery lineage when the supplied final guard fails |
| SSIAG server policy and kernel peer | `TestAuthorizationOverKernelAuthenticatedSocketIsAudited`, `TestHostOwnerCanProposeApplyAndActivateLocalPolicy`, `TestHandlerRejectsRequestWithoutKernelPeerContext` | Actual local server/peer and policy transitions; this server unit suite records audit calls through a test audit adapter |
| Actual SSIAG/STAV decision and replay | `TestInstalledSHVBoundedCatalogueWorkflow`, through the required SHV installed runner | Private same-UID authority processes, kernel-authenticated policy event and no duplicate audit on replay; not full publication apply |
| Restore after interruption | `*QuiescentRestore*` tests | Historical bytes and pending intent survive; restore itself grants no authority |

The structural decision/guard tests are not live revocation-service proof. For an actual deployment, retain denied, expired, revoked, wrong-resource and interrupted-operation evidence with the exact service/config/profile identities before claiming that deployment qualified. A live protected-publication interruption campaign must separately record its exact owner builds, process barriers and policy transitions; test-only fault hooks do not belong in production builds.

## Decision, effect and remaining race

SCV/SHV/SNV native planners describe transitions; qxctl owns the filesystem mutation. A protected operation retains its immutable intent/correlation before requesting authority, validates the exact resource/action/audience/TOPS and installation, and rechecks freshness/current authority immediately before replacement. Compare-and-swap prevents a stale plan from silently overwriting another head.

The policy check and head rename are not one atomic transaction across SSIAG and filesystem state. Revocation or evidence change after the final check can race the rename. This is an explicit MVP boundary, not a guarantee of atomic revocation. Strengthening it requires an agreed security contract and a separately reviewed protocol; this increment does not add locks across owners or invent that guarantee.

STAV records the SSIAG policy decision. The retained local outcome journal records the mutation disposition. `authorization_audit: ssiag_policy_decision_only` and `head_write_stav_receipt: null` must remain truthful. An allow receipt is not proof that the head was written; a timeout is not proof that nothing changed. Reconcile status, retained operation ID and current head before retrying, with fresh authorization when effect remains necessary. Never fabricate a head-write audit receipt or erase an uncertain attempt.

Keep policy identities, grant/config digests, request/correlation IDs and before/after head digests in private qualification evidence. Redact secrets and private source bytes from shared reports. The reporting destination is the one selected in `SECURITY.md`; this document does not infer a response SLA or attest the ticket system's privacy configuration.
