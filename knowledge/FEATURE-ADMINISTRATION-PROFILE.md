# Symphony Feature Administration Profile

## Status

Canonical bootstrap policy for `symphony.knowledge.feature-administration-profile.v1`.

## Current Baseline

- profile ID: `symphony.registered-features.administration.v1`
- SSFV source: `knowledge/ssfv/REGISTRY.md`
- catalog scope: `registered_partial_catalog`
- catalog complete: `false`
- registered feature count: `91`
- reviewed interaction expectations: `179`
- forward gate: `enforce_new_records`

All ninety-one currently registered feature IDs appear exactly once in the normalized machine profile. Their 179 explicit expectations comprise 169 required interactions, nine prohibited interactions, and one not-applicable interaction. Evidence-backed runtime-only and system-orchestrated requirements remain explicit where a direct qxctl leaf would be meaningless or unsafe. SAV and SEV retain exact headless read-only/proposal bindings, while the Accordare producer adds two-phase mutation-audit invocation, separate intent/append status and recovery, exact native supervision, and direct grant administration. Existing lifecycle, invariant, provider-trust/readiness/binding, root-summary, governed-validation, and policy-reset boundaries remain unchanged.

All reviewed qxctl-mapped administration surfaces now carry their exact backend feature and interaction bindings. The foundational exception publishes twenty stable commands and twenty separately registered module operations in addition to the established mappings. No operation identity is invented before its dispatch or service contract publishes it. The independently installed SSFV engine evaluates these cross-layer facts from the expected qxctl registry and supplied engine descriptors.

The checked-in machine-evaluable profile is `knowledge/FEATURE-ADMINISTRATION-PROFILE.json`. Its registry digest binds the exact current registry bytes; its profile digest follows the omit-self canonical JSON rule in `knowledge/FEATURE-ADMINISTRATION.md`.

## Bootstrap Close Evidence

The current profile declares 179 reviewed surfaces. Runtime satisfaction counts and the result digest remain evaluation-specific because they also bind supplied engine descriptors and any observed qxctl registry. The canonical profile digest is `sha256:96e4060392cf8464381386473f3ec359221a8cce3cce2cc73398266bffe8a16e`, the bound SSFV registry digest is `sha256:1ab6fb5708a5046c8a073bea4060657181a411566142ae160716d3f8f774c2de`, and the expected 198-leaf qxctl registry digest is `sha256:05080715181c783d893fc221c0db93e80cfb25dfb7ffea952f0479f3b683f732`.

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
