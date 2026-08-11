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

The independently installable C++ `ssfv-engine` and Go qxctl client implement bounded inspect, check, diff, proposal, and disposable graph operations. They provide application-owned mechanics without deciding feature-worthiness or mutating canonical truth.

The current partial semantic catalog contains exactly four Architect-ratified records: the repository-root platform capability, the shared knowledge-vector engine foundation, the knowledge-session coordinator foundation with durable reconciliation, and authenticated durable Maestro docking presence. It proves sparse root and nested ownership without claiming repository-wide feature coverage.

## Non-Authorization Statement

This vector and its engine do not autonomously create application `FEATURES.md` records, decide feature-worthiness, enable canonical mutation, administer Maestro, publish documentation, or authorize a graph database. The four current records authorize no additional record or complete-catalog claim.
