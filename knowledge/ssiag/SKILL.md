# Symphony Secure Identity and Access Governance Skill

## Purpose

Guide humans and agents in safely reading, reviewing, configuring, and implementing SSIAG contracts.

## Required Reading Order

1. `knowledge/ssiag/INTENT.md`
2. `knowledge/ssiag/MANIFEST.md`
3. `knowledge/ssiag/SPEC.md`
4. `modules/secure-identity-access-governance/REQUIREMENTS.md`
5. the selected provider module contracts
6. `knowledge/stav/SPEC.md` before changing audit output

## Agent Authority

Agents may inspect canonical contracts, query safe SSIAG metadata through qxctl, and propose administrative changes. They must not request, print, persist, or infer credential values; bypass policy; edit operational STAV ledgers; invent fallback providers; or promote a draft relationship or schema to canonical status without owner review.

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
4. Implement local proposal/apply mutation with replay, idempotency, expected-state, and audit gates.
5. Implement provider mutual executable trust and separate control/secret channels.
6. Enable per-user macOS Keychain operations beginning with non-exportable capability where suitable.

## Stop Conditions

Stop and obtain owner approval before choosing an unrecorded namespace, enabling remote access, allowing agents to apply, weakening peer authentication, exporting a non-exportable credential, adding a provider fallback, changing the provider IPC major version, publishing an API, or weakening safe metadata exclusions. Do not mark a ratified capability operational until its exact contract and tests pass.
