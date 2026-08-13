# Symphony Feature Administration Profile

## Status

Canonical bootstrap policy for `symphony.knowledge.feature-administration-profile.v1`.

## Current Baseline

- profile ID: `symphony.registered-features.administration.v1`
- SSFV source: `knowledge/ssfv/REGISTRY.md`
- catalog scope: `registered_partial_catalog`
- catalog complete: `false`
- registered feature count: `69`
- reviewed interaction expectations: `128`
- forward gate: `enforce_new_records`

All sixty-nine currently registered feature IDs appear exactly once in the normalized machine profile. Their 128 explicit expectations comprise 119 required interactions, eight prohibited interactions, and one not-applicable interaction. Eight runtime-only and four system-orchestrated requirements record evidence-backed boundaries where a direct qxctl leaf would be meaningless or unsafe. Four required administrator-facing interactions deliberately have no command mapping yet: SSIAG supervision, SSIAG TOPS enrollment, STAV supervision, and STAV TOPS enrollment. They are uncovered work, not exemptions. SSIAG policy reset is not a missing configure leaf: it is an explicit mode of the existing proposal-plus-apply circuit.

The completed adjudication does not claim that every expected command already carries the exact backend feature binding or stable engine-operation identity. The independently installed SSFV engine evaluates those cross-layer facts from the expected qxctl registry and supplied engine descriptors and reports mismatches as uncovered evidence. Advancing the profile gate prevents new semantic records from recreating unreviewed debt; it does not convert an expected route into implemented coverage.

The checked-in machine-evaluable profile is `knowledge/FEATURE-ADMINISTRATION-PROFILE.json`. Its registry digest binds the exact current registry bytes; its profile digest follows the omit-self canonical JSON rule in `knowledge/FEATURE-ADMINISTRATION.md`.

## Bootstrap Close Evidence

Against SSFV registry digest `sha256:93af8380103b30f323e379349ef25aba5d45e340878d0ed887fab803550e3277`, profile digest `sha256:84610f5696f8818035c667362d3d2a9c74fbe10ab53efc2e685b7469ec4cc0d3`, and expected qxctl registry digest `sha256:ab314800fe998ee7f6d59aee68ef23e7cd9d3cb6ff48b30fd8999eb46966647c`, the engine-first full-profile assessment reports 128 surfaces: 67 satisfied, 40 uncovered, 13 exempt, eight prohibited, zero stale, and zero unresolved.

The forty uncovered surfaces divide cleanly. Four administrator-facing lifecycle requirements have no expected qxctl command: SSIAG supervision, SSIAG TOPS enrollment, STAV supervision, and STAV TOPS enrollment. The other thirty-six have reviewed expected qxctl commands, but those command records currently bind only the qxctl wrapper feature rather than the corresponding backend feature and interaction. This is exact feature-binding debt, not missing grammar and not permission failure. Stable `engop` adoption beyond the SSFV engine remains a separately reviewed implementation gate; the profile does not invent operation identities before their owning dispatch or service contracts publish them.

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
