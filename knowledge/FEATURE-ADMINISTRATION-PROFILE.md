# Symphony Feature Administration Profile

## Status

Canonical bootstrap policy for `symphony.knowledge.feature-administration-profile.v1`.

## Current Baseline

- profile ID: `symphony.registered-features.administration.v1`
- SSFV source: `knowledge/ssfv/REGISTRY.md`
- catalog scope: `registered_partial_catalog`
- catalog complete: `false`
- registered feature count: `69`
- forward gate: `report_only`

All sixty-nine currently registered feature IDs MUST appear exactly once in the normalized machine profile before the gate advances. During report-only bootstrap, an empty `expectations` array means feature-level review has not yet identified any administration interaction; it is visible unreviewed debt and does not invent an interaction or requirement. When an interaction is known but its mapping remains unadjudicated, it is represented explicitly with `delivery: unreviewed`, empty command and engine-operation arrays, a null inheritance source, rationale, and no manufactured evidence. Absence of a feature record is invalid, and an empty expectations array is never not-applicable.

The checked-in machine-evaluable profile is `knowledge/FEATURE-ADMINISTRATION-PROFILE.json`. Its registry digest binds the exact current registry bytes; its profile digest follows the omit-self canonical JSON rule in `knowledge/FEATURE-ADMINISTRATION.md`.

## Advancement

The gate advances to `enforce_new_records` only after every current record has a reviewed expectation, evidence-backed exception, or finite acyclic inherited expectation. Thereafter every new SSFV record must add its administration disposition and evidence in the same ratified change. Empty expectation arrays and `delivery: unreviewed` fail both enforcement gates; JSON Schema provides structural closure, while the engine and validator enforce that cross-record policy. `enforce_all_records` is permitted only when no current record remains unreviewed. This profile never changes SSFV's partial-catalog status.

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
