# Symphony Secure Identity and Access Governance Manifest

## Module Identity

- **name**: `secure-identity-access-governance`
- **path**: `modules/secure-identity-access-governance/`
- **binary**: `symphony-ssiag`
- **schema prefix**: `symphony.ssiag.*`
- **environment prefix**: `SYMPHONY_SSIAG_*`
- **language/runtime**: Go 1.26.5 with pinned `golang.org/x/sys` for cgo-free kernel peer credentials
- **cgo**: prohibited
- **status**: DRAFT foundation; supervised metadata, audited authorization, and protected local policy administration

## Canonical Authority

`knowledge/ssiag/` owns vocabulary, relationships, configuration extensions, provider protocol, compatibility, and authority boundaries. This module implements that truth. qxctl administers and queries it but owns neither schema nor runtime state.

## Contract Files

- `INTENT.md`, `MANIFEST.md`, `INSTALL.md`, `SKILL.md`, `SPEC.md`
- `ARCHITECTURE.md`, `REQUIREMENTS.md`, `IMPLEMENTATION.md`, `THREAT-MODEL.md`

## Implemented Surfaces

- `package install|uninstall --prefix --version`: immutable compiled-version entry point and receipt-last `symphony.knowledge.install-receipt.v2`, with reference-safe removal;
- legacy `install` / `uninstall`: verified fixed-path dual-read migration layout;
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
- `qxctl ssiag status|providers|doctor|policy ...`: provider-neutral query and local policy administration interface.

## Install and Enrollment Separation

Host uninstall never deletes per-TOPS state and refuses retained supervisor, live endpoint, or unresolved lifecycle-attempt references. `unenroll` preserves state by default; native-only `unenroll --purge` targets exactly one validated TOPS UUID after descriptor, live-socket, and lifecycle-lock proof. Protected digest-linked attempts reside outside the purge subtree. Display names are configuration metadata and never path components.

Ordinary foundational lifecycle apply is implemented fail-closed pending a real STAV receipt path. Explicit `audit_deferred` apply is durable and reports reconciliation required. The typed committed-receipt binding hook is not a completed reconciliation endpoint, and qxctl lifecycle v1 exposes no purge intent.

## Provider Boundary

No operational credential provider or canonical knowledge apply route is enabled. Local policy apply affects only SSIAG's protected operational overlay. Native dependencies remain in independently installed adapters. The first adapter scaffold is `modules/ssiag-provider-macos-keychain/`, a separate Swift executable whose current capability is metadata only.

## Contamination Boundary

Secret values, proofs, assertions, raw tokens, provider payloads, and native errors must not cross into qxctl, SKV, SKVI, SCLV, SODV, STAV, manifests, inventories, logs, or status responses. A bounded local policy file and proposal may cross qxctl only for the explicit policy-administration operation; status and STAV expose digests and safe references, never the policy body.
