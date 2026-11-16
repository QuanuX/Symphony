# Symphony Secure Identity and Access Governance Manifest

## Canonical Surfaces

- `modules/secure-identity-access-governance/ARCHITECTURE.md`
- `modules/secure-identity-access-governance/CREDENTIAL-USE.md`
- `modules/secure-identity-access-governance/CREDENTIAL-DISPATCH.md`
- `modules/secure-identity-access-governance/CREDENTIAL-JOURNAL.md`
- `modules/secure-identity-access-governance/FEATURES.md`
- `modules/secure-identity-access-governance/IMPLEMENTATION.md`
- `modules/secure-identity-access-governance/INSTALL.md`
- `modules/secure-identity-access-governance/INTENT.md`
- `modules/secure-identity-access-governance/MANIFEST.md`
- `modules/secure-identity-access-governance/README.md`
- `modules/secure-identity-access-governance/REQUIREMENTS.md`
- `modules/secure-identity-access-governance/SKILL.md`
- `modules/secure-identity-access-governance/SPEC.md`
- `modules/secure-identity-access-governance/THREAT-MODEL.md`

## Module Identity

- **name**: `secure-identity-access-governance`
- **path**: `modules/secure-identity-access-governance/`
- **binary**: `symphony-ssiag`
- **schema prefix**: `symphony.ssiag.*`
- **environment prefix**: `SYMPHONY_SSIAG_*`
- **language/runtime**: Go 1.26.5 with pinned `golang.org/x/sys` for cgo-free kernel peer credentials
- **cgo**: prohibited
- **status**: DRAFT foundation; supervised metadata, audited authorization, protected policy administration, exact provider binding, and Phase 10B provider readiness with operations disabled

## Canonical Authority

`knowledge/ssiag/` owns vocabulary, relationships, configuration extensions, provider protocol, compatibility, and authority boundaries. This module implements that truth. qxctl administers and queries it but owns neither schema nor runtime state.

## Contract Files

- `INTENT.md`, `MANIFEST.md`, `INSTALL.md`, `SKILL.md`, `SPEC.md`
- `ARCHITECTURE.md`, `REQUIREMENTS.md`, `IMPLEMENTATION.md`, `THREAT-MODEL.md`

## Implemented Surfaces

- `package install|uninstall --prefix --version`: immutable compiled-version entry point and receipt-last `symphony.knowledge.install-receipt.v2`, with owner-local admission and retirement fences for freshly installed user-owned packages;
- legacy `install` / `uninstall`: fixed-path compatibility layout, outside qualified live reclamation;
- `enroll` / `unenroll`: isolated per-TOPS configuration and state;
- `foundation-lifecycle`: bounded stdin/stdout `describe`, `observe`, `plan`, `apply`, `apply_status`, and `recover` for enrollment and supervisor surfaces;
- `serve`: one protected Unix-socket API for one TOPS;
- `supervisor install` / `supervisor uninstall`: per-TOPS launchd or systemd liveness profile with conservative state preservation;
- Darwin/Linux kernel peer authentication on every accepted API connection;
- exact per-TOPS UID/GID-to-canonical-subject resolution for future subject-gated operations;
- stable per-TOPS service identity, pre-listen process verification, and client-side exact endpoint verification;
- owner-provisioned system identity validation, distinct service-owned state/runtime children, bounded restart/shutdown, and serialized stale-socket recovery;
- `status` / `providers`: safe local inspection;
- `POST /v1/authorization/decisions`: kernel-subject-derived, exact-grant, deny-by-default authorization with fail-closed STAV audit and non-transferable capability evidence;
- `GET /v1/policy/status` plus `POST /v1/policy/proposals|apply|recover`: protected per-TOPS operational policy administration with exact host authority, CAS, atomic state, STAV-before-commit, and explicit recovery;
- `GET /v1/provider-installations/<provider>` plus provider-binding `status|plans|apply|attempts|recover`: bounded exact-pair inventory and protected per-TOPS binding lifecycle with independent candidate verification and STAV-before-commit;
- `POST /v1/provider-readiness/<provider>/observations`: permission-backed, read-only structural-signature, protected-policy, and security-session evidence with all operational flags false;
- `provider-binding-recover`: receipt-v2-bound native recovery that requires target-host ownership, entry into the enrolled service identity, exclusive service socket-lifecycle ownership, and an absent socket;
- `qxctl ssiag status|providers|doctor|policy ...|provider installations|binding ...`: provider-neutral headless administration.

## Install and Enrollment Separation

Package retirement never deletes per-TOPS state. For fresh guarded user-owned installations, it durably closes start admission before checking held process use, retained supervisors, live endpoints and unresolved lifecycle attempts, and preserves bytes on refusal. It requires an exact retained owner executable outside the target installation. System-owned, legacy, uncontrolled-consumer and qxctl-administered shared-root reclamation remain unqualified; these are not inferred safe from an empty process or socket scan. `unenroll` preserves state by default; native-only `unenroll --purge` targets exactly one validated TOPS UUID after descriptor, live-socket, and lifecycle-lock proof. Protected digest-linked attempts reside outside the purge subtree. Display names are configuration metadata and never path components.

Ordinary foundational lifecycle apply is implemented fail-closed pending a real STAV receipt path. Explicit `audit_deferred` apply is durable and reports reconciliation required. The typed committed-receipt binding hook is not a completed reconciliation endpoint, and qxctl lifecycle v1 exposes no purge intent.

## Provider Boundary

No operational credential provider or canonical knowledge apply route is enabled. Provider-binding apply changes only the exact metadata adapter/foundation pair selected for one TOPS; it does not enable adapter operations. The Phase 9 configuration declaration remains immutable legacy/bootstrap evidence. Lifecycle-capable state is service-owned under the per-TOPS state root. Native dependencies remain independently installed adapters.

## Contamination Boundary

Secret values, proofs, assertions, raw tokens, provider payloads, and native errors must not cross into qxctl, SKV, SKVI, SCLV, SODV, STAV, manifests, inventories, logs, or status responses. A bounded local policy file and proposal may cross qxctl only for the explicit policy-administration operation; status and STAV expose digests and safe references, never the policy body.

## Internal Credential Admission

`CREDENTIAL-ADMISSION.md` owns the internal resource/lease snapshot and native
admission composition contract. `internal/credential/admission.go` implements
private value ownership, revision CAS and pins through native execution/cleanup;
`admission_test.go` verifies its affected boundaries. Native authentication,
protected delivery and operational Keychain access remain disabled.

`CREDENTIAL-JOURNAL.md` and `CREDENTIAL-DISPATCH.md` also cover the separate
provider-outcome STAV receipt. `internal/credential/outcome.go` owns candidate
replay and verified receipt persistence; `outcome_test.go` and
`outcome_process_test.go` cover migration, audit failure, lost acknowledgement
and actual append-authority restart. `CREDENTIAL-QUALIFICATION.md` records the
remaining native bridge, deployment and live-provider acceptance work.

### Package retirement qualification boundary

New native receipts declare `symphony.package-retirement.v1`. Installation creates
nonportable host-local identity under `.symphony-package-retirement/` within the
prefix: root and stable lock device/inode identities, an installation epoch, and
exact receipt/executable evidence. This state is never a shipped payload and must
not be deleted or copied to adopt an existing installation. Closed admission
survives controller crashes; repeated removal reconciles the same identity and
retains a tombstone. A retired prefix/version is never implicitly reopened.

The participating domain covers this owner's immutable-package serve entry point,
enrollment creation and native supervisor install/start. The serve lease outlives
owned state and server shutdown. Existing stop and recovery paths remain available;
there is no new service manager, arbitrary process discovery, or process killing.
Old direct launches, copied SDK consumers and fixed-path installations remain
outside this proof. Real service-manager/platform acceptance remains a separate
release gate.

The source receipt change is not a published module-version update. qxctl clients
using older published owner validators require the S05 source/version closure
before this guarded lane is available through distribution dispatch. Published
tags and checksums have not been changed.
