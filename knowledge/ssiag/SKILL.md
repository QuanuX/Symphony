# Symphony Secure Identity and Access Governance Skill

## Purpose

Guide every authorized caller in safely reading, reviewing, configuring, and implementing SSIAG contracts.

## Required Reading Order

1. `knowledge/ssiag/INTENT.md`
2. `knowledge/ssiag/MANIFEST.md`
3. `knowledge/ssiag/SPEC.md`
4. `knowledge/ssiag/PROVIDER-LIFECYCLE.md` before changing provider inventory or binding behavior
5. `knowledge/ssiag/PROVIDER-READINESS.md` before changing signing, bundle, session, or operational-eligibility behavior
6. `modules/secure-identity-access-governance/REQUIREMENTS.md`
7. the selected provider module contracts
8. `knowledge/stav/SPEC.md` before changing audit output

## Caller Authority

Caller type is not an authorization input. A caller may inspect canonical contracts, query safe SSIAG metadata, propose administrative changes, or use a future apply operation only to the extent permitted by the target host and its caller-neutral safeguards. No supported operation may request, print, persist, or infer excluded credential values; bypass effective permission; edit operational STAV ledgers; invent fallback providers; or promote a draft relationship or schema without permission-backed ratification.

## Change Procedure

1. Identify the affected canonical relationship and requirement IDs.
2. Keep immutable IDs separate from display names.
3. update canonical contracts before or atomically with implementations;
4. keep foundation changes Go-only and cgo-free;
5. keep native platform code in an independent adapter;
6. verify safe-output and fail-closed tests;
7. update SKVI relationships;
8. create SCLV evidence only after the real review and merge facts exist.

## Ratified Implementation Sequence

1. Maintain the implemented build-tagged local peer-credential authentication, exact UID/GID mapping, and endpoint trust.
2. Maintain the implemented foundational bootstrap supervision without granting the supervisor policy authority.
3. Maintain the implemented dedicated per-TOPS STAV append authority integration.
4. Maintain the implemented exact-grant deny-by-default authorization decision and non-transferable capability surface.
5. Maintain qxctl/coordinator authenticated-session use without converting decision evidence into bearer or canonical apply authority.
6. Preserve the implemented local policy proposal/apply/recovery circuit: kernel-derived authority, CAS, idempotent STAV-before-commit, protected attempt/state files, and noncanonical result binding.
7. Maintain the implemented exact provider mutual-executable-trust and metadata-control runtime without widening its v1 surface or enabling the synthetic secret channel.
8. Preserve the exact provider installation/binding lifecycle: bounded inventory, no newest-version selection, distinct command/backend identities, exact plan and state digests, durable initiating audit identity, state-before-committed ordering, `prepared -> candidate_verified -> audited -> committed` recovery, and committed distinct STAV evidence.
9. Preserve the implemented Phase 10B complete-bundle receipt/staging and three-layer readiness circuit. Native structural success and policy match remain non-operational evidence; qxctl must validate the closed result without accepting signing or path authority.
10. Enable per-user macOS Keychain operations only after the separate Phase 10C gate, beginning with non-exportable capability where suitable.

## Stop Conditions

Stop and obtain permission-backed owner approval before choosing an unrecorded namespace, enabling remote access, enabling canonical apply, weakening peer authentication, exporting a non-exportable credential, adding a provider fallback, changing the provider IPC major version, auto-selecting an installed provider version, widening provider binding into provider execution, publishing an API, or weakening safe metadata exclusions. Do not mark a ratified capability operational until its exact contract and tests pass.
