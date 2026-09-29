# Symphony Feature Administration Profile

## Status

Canonical reviewed policy for `symphony.knowledge.feature-administration-profile.v1`, retaining the `enforce_new_records` gate and explicitly partial catalog.

## Current Baseline

- profile ID: `symphony.registered-features.administration.v1`
- SSFV source: `knowledge/ssfv/REGISTRY.md`
- catalog scope: `registered_partial_catalog`
- catalog complete: `false`
- registered feature count: `131`
- reviewed interaction expectations: `339`
- forward gate: `enforce_new_records`

The 131 registered feature IDs appear exactly once. Their 339 reviewed expectations comprise 329 required, 9 prohibited and 1 not-applicable interactions. Runtime-only and system-orchestrated exceptions retain their owner evidence. The SCV increment adds explicit domain-engine operation mappings and a separately owned qxctl adapter for immutable corpus retention, profile/connection evidence and protected source/graph administration. A source or graph result is not permission, durable selection or a provider action.

## Exact Machine Evidence

The profile digest is `sha256:c77420c2fffeae39b37166873ed795f6b6c9c1bb92ae0054efb05f0ddd512250`. Its bound SSFV registry digest is `sha256:bb9dacd8a11ea68a23506f4efa396820f54676a3e1f7236432c95aa28472d353`. The expected qxctl registry has 383 leaves with digest `sha256:52026cfe39276aa24b33d004a1820dd055640304ac3e047b21801356a19646bc`. These are source-level mappings; installed acceptance also requires exact owner descriptors, receipts and exercised process/storage boundaries. Catalog completeness remains false.

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

The four SCLV historical-warning commands extend the existing governed-validation configure, discover and inspect expectations. They select exact historical-reference subjects and reuse local acknowledgement and reopening. They add no independent feature or expectation and do not supply delivery acceptance state.

The SQV increment adds runtime-only invocation expectations for exact C++26 SQAV, SQFV, SQMV, SQTV, SQPV and SQDV library contracts; no SQV command or process is inferred from those records.

The SQAV Databento DBNv1/v3 MBO file library has its own runtime-only invocation expectation; it supplies no network or credential command.

The non-live SQV administration pass adds six exact native owner adapters and seven canonical leaves. Metadata projections, flow resource trials, integer selection and retained-state inspection share schema/template and discovery paths. Read-only observation does not recover storage or activate providers.
