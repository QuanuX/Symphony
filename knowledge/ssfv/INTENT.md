# Symphony Semantic Feature Vector Intent

## Purpose

SSFV defines Symphony's canonical application-feature truth. It explains why a capability exists, what it does, who or what uses it, how it works, when and where it applies, and how it differs from nearby capabilities.

The vector makes small but important behavior discoverable without treating code size, language, marketing prominence, or caller type as a measure of significance.

## Source-Truth Boundary

`knowledge/ssfv/` owns feature identity, vocabulary, lifecycle, hierarchy, relationship rules, the deterministic feature-file format, and the registry that routes feature identities to distributed `FEATURES.md` owner records. A distributed feature record owns the feature semantics for its declared source scope.

SSFV does not replace:

- SKVI location and contract routing;
- SCLV reviewed change history;
- SACV API-contract truth;
- SODV publication and release truth;
- STAV operational audit truth;
- SSIAG identity, authentication, access, and governance decisions;
- Maestro deployment, docking, or persisted runtime state;
- source code as implementation evidence.

Generated inventories, graph projections, summaries, encyclopedia entries, and marketing materials are derived consumers. They never become canonical feature truth.

## Scope

SSFV governs:

- stable feature identifiers and namespaces;
- capability, feature, subfeature, and microfeature records;
- explicit hierarchy and typed cross-feature relationships;
- complete 5W1H semantics;
- implementation-path and implementation-language-role evidence;
- lifecycle and distinction from closely related features;
- sparse distributed `FEATURES.md` placement;
- deterministic validation, comparison, proposal, and graph-projection contracts.

## Feature-Worthiness Intent

A behavior is feature-worthy only when all five statements are supported by evidence:

1. it exposes a distinct user-, operator-, system-, or integration-visible capability;
2. it has a stable purpose that can be described independently;
3. it has an identifiable implementation or contract boundary;
4. losing or changing it would materially alter capability;
5. it can be distinguished from its parent and adjacent features.

Tooling may identify candidates and missing evidence. It MUST NOT make the semantic decision that a behavior is feature-worthy.

## Caller-Neutral Intent

Any caller operating within effective host permissions may inspect feature truth, produce evidence, or propose changes. Authority derives from ownership, authenticated identity, granted permissions, target-host policy, and operation context—not from whether the caller is described as human, AI, agentic, automated, a service, a workload, or an organization.

Owner-configured safeguards may constrain operations by permission, scope, risk, session, environment, or operation. Safeguards MUST remain caller-neutral.

## Implementation Boundary

The independently installable C++ `ssfv-engine` and Go qxctl client implement bounded inspect, check, diff, proposal, and disposable graph operations. They provide application-owned mechanics without deciding feature-worthiness or mutating canonical truth. The separate C++ knowledge-session coordinator and qxctl session surface persist content-addressed SSFV baselines and review evidence across an authenticated session; this maintenance stream remains noncanonical and cannot apply its findings.

The current partial semantic catalog contains seventy-four Architect-ratified experimental records across the repository root and fourteen implemented owner scopes. Fifty-nine nested subfeatures are now recorded across thirteen owner scopes: qxctl, the knowledge-session coordinator, Maestro, the SSIAG foundation, the STAV append authority, the STAV Go protocol kernel, the shared C++ foundation, the Symphony Validator, and the SKVI, SCLV, SACV, SODV, and SSFV engines. The latest record captures SSIAG's exact provider-installation and protected binding lifecycle after its mutually verified metadata-only provider trust boundary; the preceding records cover qxctl invariant assurance and validator-owned invariant-ownership and implemented-module-admission assurance. `COVERAGE.md` records that exact progress, the reviewed macOS adapter no-child disposition, explicit non-feature boundaries, the top-level source universe, three proposal-only exclusions, freshness rule, and the remaining conditions reserved for a future `complete` state.

## Non-Authorization Statement

This vector and its engine do not autonomously create application `FEATURES.md` records, generate feature names or IDs, decide feature-worthiness, enable canonical mutation, administer Maestro, publish documentation, or authorize a graph database. Session maintenance may consume a complete derived Maestro receptor inventory only as lineage evidence; neither SSFV nor the coordinator owns Maestro state. The current records and coverage inventory authorize no unreviewed record, complete-catalog claim, complete legacy-invariant claim, or installed-host completeness claim.
