# Symphony Feature Administration Profile

## Status

Canonical bootstrap policy for `symphony.knowledge.feature-administration-profile.v1`.

## Current Baseline

- profile ID: `symphony.registered-features.administration.v1`
- SSFV source: `knowledge/ssfv/REGISTRY.md`
- catalog scope: `registered_partial_catalog`
- catalog complete: `false`
- registered feature count: `73`
- reviewed interaction expectations: `139`
- forward gate: `enforce_new_records`

All seventy-three currently registered feature IDs appear exactly once in the normalized machine profile. Their 139 explicit expectations comprise 128 required interactions, ten prohibited interactions, and one not-applicable interaction. Eight runtime-only and four system-orchestrated requirements record evidence-backed boundaries where a direct qxctl leaf would be meaningless or unsafe. The four foundational lifecycle features each bind five exact qxctl leaves to five distinct module operations. Invariant assurance adds stable status, list, show, and exact-validator check routes, while its validator-owned check remains explicitly read-only. Provider-trust assurance binds exact `show` inspection and permission-backed fresh `verify` routes without claiming operational Keychain or secret delivery. Root-summary assurance binds one exact read-only qxctl leaf and prohibits apply. Governed validation binds its profile/baseline circuit plus the complete caller-neutral warning lifecycle query and CAS mutation surface. SSIAG policy reset is not a missing configure leaf: it is an explicit mode of the existing proposal-plus-apply circuit.

All reviewed qxctl-mapped administration surfaces now carry their exact backend feature and interaction bindings. The foundational exception publishes twenty stable commands and twenty separately registered module operations in addition to the established mappings. No operation identity is invented before its dispatch or service contract publishes it. The independently installed SSFV engine evaluates these cross-layer facts from the expected qxctl registry and supplied engine descriptors.

The checked-in machine-evaluable profile is `knowledge/FEATURE-ADMINISTRATION-PROFILE.json`. Its registry digest binds the exact current registry bytes; its profile digest follows the omit-self canonical JSON rule in `knowledge/FEATURE-ADMINISTRATION.md`.

## Bootstrap Close Evidence

The exact engine-first close evidence reports 139 surfaces: 116 satisfied, 13 exempt, ten prohibited, and zero uncovered, stale, or unresolved. The result digest is evaluation-specific because it also binds supplied engine descriptors and any observed qxctl registry. The canonical profile digest is `sha256:586510a7a5fe0323ee6bb53749178580a446ae75a40e106ee754ecbb21d667fa`, the bound SSFV registry digest is `sha256:fdda20a7d802df55b0a5eb7774ab08e405568a473c675dbc6ef0e39f16933eec`, and the expected 150-leaf qxctl registry digest is `sha256:c5ce07ee01b27209b131557000670c2af641b1e6fd47bb9946228aa52d5ec37a`.

The four closed surfaces are exact administrator-facing lifecycle routes: SSIAG supervision, SSIAG TOPS enrollment, STAV supervision, and STAV TOPS enrollment. Each exposes status, plan, apply, apply-status, and recover through qxctl while the Go module owns transaction behavior. The expected registry retains the qxctl-owned wrapper binding beside every backend binding so command-layer behavior and administered capability remain distinct and independently auditable.

## Advancement

The gate is now `enforce_new_records`. Every new SSFV record MUST add a reviewed administration expectation, evidence-backed exception, or finite acyclic inherited expectation in the same ratified change. Empty expectation arrays and `delivery: unreviewed` fail both enforcement gates; JSON Schema provides structural closure, while the engine and validator enforce that cross-record policy. `enforce_all_records` remains separately gated because this profile adjudicates only the registered partial catalog and does not prove repository-wide feature completeness. This profile never changes SSFV's partial-catalog status.

Normalized profile JSON is governed by `knowledge/schemas/v1/feature-administration-profile.schema.json`; it is derived from this policy, the exact SSFV registry, and reviewed mappings. It is not an independently editable source of feature semantics.

## Required Checks

- profile feature IDs are unique and equal the exact registered set for the bound registry digest;
- interaction keys are unique within each feature;
- command and engine-operation IDs resolve in their bound expected registries/descriptors;
- inherited feature and interaction references resolve, preserve requirement meaning, and form no cycle;
- prohibited and not-applicable expectations carry no executable mapping;
- unreviewed expectations remain visible debt;
- expected command coverage remains evaluable with no observed qxctl manifest;
- observed qxctl and installed-engine compatibility affect only live state;
- authorization evidence affects only authorization state.

## Non-Authorization

This bootstrap profile does not declare all Symphony features known, grant an exemption by omission, authorize a command or operation, change module installation state, or permit canonical apply.
