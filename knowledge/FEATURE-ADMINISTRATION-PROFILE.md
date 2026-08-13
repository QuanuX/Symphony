# Symphony Feature Administration Profile

## Status

Canonical bootstrap policy for `symphony.knowledge.feature-administration-profile.v1`.

## Current Baseline

- profile ID: `symphony.registered-features.administration.v1`
- SSFV source: `knowledge/ssfv/REGISTRY.md`
- catalog scope: `registered_partial_catalog`
- catalog complete: `false`
- registered feature count: `70`
- reviewed interaction expectations: `131`
- forward gate: `enforce_new_records`

All seventy currently registered feature IDs appear exactly once in the normalized machine profile. Their 131 explicit expectations comprise 121 required interactions, nine prohibited interactions, and one not-applicable interaction. Eight runtime-only and four system-orchestrated requirements record evidence-backed boundaries where a direct qxctl leaf would be meaningless or unsafe. The four previously uncovered administrator-facing interactions—SSIAG supervision, SSIAG TOPS enrollment, STAV supervision, and STAV TOPS enrollment—now each bind five exact qxctl leaves to five distinct module operations. Root-summary assurance binds one exact read-only qxctl leaf and prohibits apply. Governed validation binds its profile/baseline circuit plus the complete caller-neutral warning lifecycle query and CAS mutation surface. SSIAG policy reset is not a missing configure leaf: it is an explicit mode of the existing proposal-plus-apply circuit.

All reviewed qxctl-mapped administration surfaces now carry their exact backend feature and interaction bindings. The foundational exception publishes twenty stable commands and twenty separately registered module operations in addition to the established mappings. No operation identity is invented before its dispatch or service contract publishes it. The independently installed SSFV engine evaluates these cross-layer facts from the expected qxctl registry and supplied engine descriptors.

The checked-in machine-evaluable profile is `knowledge/FEATURE-ADMINISTRATION-PROFILE.json`. Its registry digest binds the exact current registry bytes; its profile digest follows the omit-self canonical JSON rule in `knowledge/FEATURE-ADMINISTRATION.md`.

## Bootstrap Close Evidence

Against semantic snapshot digest `sha256:aac97b3cb7521d55aa85f35ea8cbfb50b41ccbb329bacb7a9e0f84dd77ac9da1`, SSFV registry digest `sha256:5c1dfad607ee5df102942ea2c04b2ed3f54bef90217f33cca2875f89b687a2c5`, profile digest `sha256:b04e003ddf26185a794c2aee72ea5e277006477f9142800996d2b670070be9ff`, and expected qxctl registry digest `sha256:560b023fb5656a2619815e84ee770b829ce076f8ce15931baaa63c56f7f87e3c`, the engine-first full-profile assessment reports 131 surfaces: 109 satisfied, zero uncovered, 13 exempt, nine prohibited, zero stale, and zero unresolved.

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
