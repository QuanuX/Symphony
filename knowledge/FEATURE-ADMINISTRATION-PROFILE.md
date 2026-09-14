# Symphony Feature Administration Profile

## Status

Canonical reviewed policy for `symphony.knowledge.feature-administration-profile.v1`, retaining the `enforce_new_records` gate and explicitly partial catalog.

## Current Baseline

- profile ID: `symphony.registered-features.administration.v1`
- SSFV source: `knowledge/ssfv/REGISTRY.md`
- catalog scope: `registered_partial_catalog`
- catalog complete: `false`
- registered feature count: `107`
- reviewed interaction expectations: `281`
- forward gate: `enforce_new_records`

The 107 registered feature IDs appear exactly once. Their 281 reviewed expectations comprise 271 required, 9 prohibited and 1 not-applicable interactions. Runtime-only and system-orchestrated exceptions retain their owner evidence. The SCV increment adds explicit domain-engine operation mappings and a separately owned qxctl adapter for immutable corpus retention, profile/connection evidence and protected source/graph administration. A source or graph result is not permission, durable selection or a provider action.

## Exact Machine Evidence

The profile digest is `sha256:9e7f40fa2c2f9bacd5d8c338a0da17380643e74c9891b921c308a9e9da4f2460`. Its bound SSFV registry digest is `sha256:20fa43a1434dd401450fc28c63c45c754f8b81b7676f0252cf0ee56bac857ab4`. The expected qxctl registry has 339 leaves with digest `sha256:094b8139c9d37eaf38df99095d6e36eeb59b3b01626f2f9d5ca6aef9dabeaa91`. These are source-level mappings; actual installed acceptance also requires the selected engine descriptors, exact receipts and exercised authority/storage boundaries. Catalog completeness remains false.

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
