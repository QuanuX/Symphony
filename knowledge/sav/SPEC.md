# Symphony Accordare Vector Specification

## Status and Normative Terms

Architect-ratified v1 contract. MUST, MUST NOT, SHOULD, SHOULD NOT, and MAY are normative. Runtime implementation claims remain subordinate to tested source evidence.

## Purpose

SAV deterministically resolves one exact Symphony composition and evaluates declared relationships without centralizing or rewriting their owner truth.

## Stable Identities

SAV v1 uses:

```text
savref:<namespace>:<stable.dotted-key>
savrel:<namespace>:<stable.dotted-key>
savtrait:<namespace>:<stable.dotted-key>
savver:<namespace>:<stable.dotted-key>
```

Every identity MUST match:

```text
^(savref|savrel|savtrait|savver):[a-z][a-z0-9-]{0,62}:[a-z][a-z0-9]*(?:[.-][a-z0-9]+)*$
```

The first-party namespace is `symphony`. IDs are semantic stable identifiers, not URI schemes, paths, titles, database keys, or permission grants. An incompatible meaning requires a new ID. Deprecated and retired IDs remain tombstoned.

## Accord Reference

`symphony.sav.accord-reference.v1` is the immutable canonical declaration of one expected relationship. It contains:

- `reference_id` and `relationship_id`;
- owner vector and owner contract;
- exact subject and object IDs;
- one relationship type from `RELATIONSHIPS.md`;
- applicability state and one closed applicability rule;
- required evidence protocols;
- one closed deterministic evaluation rule and version;
- definite-violation severity;
- exception eligibility and bounds;
- thermal restriction;
- source and record digests.

An Accord Reference points to owner truth. It MUST NOT embed a mutable semantic copy merely for evaluation convenience.

### Closed rule algebra

SAV v1 executes no prose, expression language, script, plugin, or model output. Both `applicability_rule` and `evaluation_rule` use the same closed rule object with explicit presence for `kind`, sorted unique `source_ids`, `pointer`, `expected_json`, and `expected_digest`:

- `always` and `never` consume no source;
- `source_available` succeeds only when every named source is present, available, and not invalid;
- `source_content_digest_equals` compares one named source with one exact tagged SHA-256 digest;
- `source_payload_pointer_equals` resolves one RFC 6901 JSON Pointer and compares the selected value with parsed `expected_json`.

Fields unused by a rule remain explicitly `null` or an empty array. `expected_json` is JSON text rather than an untyped sentinel, so JSON `null` remains a valid expected value. Invalid JSON, an invalid pointer, an absent source, stale evidence where freshness is required, or a mismatched evidence protocol produces a typed unknown/failure result according to the operation; it never falls back to textual interpretation. Expanding this algebra requires a new evaluation version and compatible schema change.

## Typed Source Projection

Every source projection supplied to the engine declares:

- `source_id`, owner vector, owner contract, and protocol;
- source authority role;
- collection state;
- stable content digest;
- optional observation digest;
- optional STSC whole-second `observed_at`;
- freshness state;
- bounded JSON payload.

Sources are sorted by `source_id`. Duplicate source IDs, mismatched self-digests, conflicting owners, unsupported critical protocols, unbounded payloads, or timestamps inside a claimed stable identity fail closed.

## CURRENT Snapshot

The machine protocol is `symphony.sav.current-snapshot.v1`. `CURRENT` is a conceptual label only.

A snapshot contains:

- exact TOPS ID, request ID, correlation ID, and causal operation identity;
- `snapshot_started_at` and `snapshot_completed_at` in STSC whole-second UTC;
- optional Named Version ID and digest;
- declared scope and `scope_digest`;
- `coverage_state`: `complete`, `partial`, or `unknown`;
- sorted source projections;
- sorted unresolved sources with typed reasons;
- stable `snapshot_digest`, omitting collection-only timestamps;
- timestamped `observation_digest`;
- `canonical: false`, `derived: true`, and `read_only: true`.

The engine MUST NOT claim `complete` when an applicable required source is absent, unresolved, stale under a required-freshness policy, or incompatible. The caller supplies source evidence; the engine does not perform ambient discovery.

## Result Axes

SAV reports three independent axes:

- reference resolution: `resolved`, `reference_unresolved`, `stale`, or `invalid`;
- composition accord: `in_accord`, `cacophonous`, or `indeterminate`;
- transition readiness: `ready`, `attunement_required`, `blocked`, or `not_evaluated`.

`cacophonous` requires at least one definite applicable `out_of_accord` finding. Missing evidence yields `unknown`, `reference_unresolved`, or `indeterminate`, never fabricated contradiction or success.

## Evaluation

`evaluate` consumes one exact CURRENT snapshot plus a bounded set of Accord References. It produces one finding per reference:

- reference and relationship IDs;
- subject and object;
- applicability;
- `in_accord`, `out_of_accord`, `unknown`, or `not_applicable`;
- typed reason code and detail;
- exact evidence source IDs and digests;
- owner contract;
- optional SEV eligibility;
- finding digest.

Evaluation is deterministic for identical semantic inputs. Every result binds its input snapshot and reference-set digest.

## Precedence and Overlays

1. Closed safety invariants and explicit prohibitions cannot be weakened locally.
2. The registered owner vector controls its namespace.
3. A Named Version selects exact identities; it does not rewrite them.
4. A local overlay may constrain behavior or fill a declared extension point.
5. An overlay cannot contradict a closed invariant or redefine source truth.
6. An exact SSIAG-authorized exception may suppress activation only at a declared exception point and within its scope, expiry, evidence, and recovery contract.
7. Missing ownership, conflicting ownership, ambiguous precedence, or unknown critical input is `reference_unresolved`.

## Named Version

`symphony.sav.named-version.v1` is an immutable composition envelope. It contains a stable `savver:` ID, display alias, exact component and contract requirements, Accord References, traits, extension points, platform and thermal bounds, predecessor, seal evidence, composition-authority reference, optional SODV publication reference, and content digest.

The alias is not identity. A seal does not publish a package, grant permission, install a component, or activate a version. Modification creates a successor; it never rewrites a sealed envelope.

## Engine Process

The exact engine ID is `symphony-sav`, module ID `sav-engine`, and vector ID `sav`. It uses `symphony.knowledge.engine-process.v1`, common strict JSON parsing, one bounded response, exact request/correlation propagation, deterministic response digests, and standard exit classes.

### v1 Operations

- `inspect`: exact empty payload;
- `current_resolve`: typed source projections and declared scope;
- `reference_check`: Accord Reference array and optional expected digest;
- `evaluate`: exact current snapshot and references;
- `diff`: two exact current snapshots;
- `explain`: one evaluation result plus exact relationship ID;
- `project_graph`: one snapshot plus optional evaluation result, JSON only;
- `compatibility`: caller-supported protocol/profile lists.

Operation identities are:

- `engop:symphony:sav.inspect`;
- `engop:symphony:sav.current.resolve`;
- `engop:symphony:sav.reference.check`;
- `engop:symphony:sav.accord.evaluate`;
- `engop:symphony:sav.current.diff`;
- `engop:symphony:sav.finding.explain`;
- `engop:symphony:sav.graph.project`;
- `engop:symphony:sav.compatibility`.

The C++ dispatch table is the single source for observed descriptor operations.

## Bounds

V1 admits at most:

- 1 MiB request;
- 4 MiB response;
- 256 source projections;
- 1,024 Accord References and findings;
- 4,096 graph nodes and 8,192 edges;
- 256 unresolved sources;
- common JSON depth, value, string, path, and deadline limits.

Measured ordinary payloads reaching half a hard bound require capacity review. Bounds MUST NOT be raised silently.

## Digests

Self-digests are tagged SHA-256 over compact UTF-8 JSON after recursive lexical key sorting with the self-digest field omitted. Arrays whose contract declares semantic sets are sorted by stable identity before hashing. Collection-only timestamps are excluded from `snapshot_digest` and included in `observation_digest`.

Digests provide deterministic evidence binding, not authentication, permission, native platform validation, or non-repudiation.

## Persistence

The engine is stateless. A caller MAY persist immutable snapshots under a per-TOPS Accordare state directory. Any selected head uses the common no-follow, dual-slot, atomic-head, synchronization, linked-generation, and unique-recovery contract. Ambient directory order never selects truth. A graph or database projection is disposable and rebuildable.

## Time

STSC governs every instant, duration, freshness check, and causal distinction. The target TOPS host owns durable commit time. Wall-clock time is never identity or sole causal order.

## Installation and Maestro

Exact versions install side by side from receipt v2 as inactive `installed_undocked`. No install creates an active alias. qxctl owns protected explicit selection. The recommended receptor identity is `receptor:symphony:knowledge.sav`. Maestro docking records presence only and does not execute SAV or grant authority.

## Security and Audit

Protected observation and administration require fresh exact SSIAG decisions for the resource and operation. Caller class is not an input. Safe STAV events may record outcome, counts, scope digest, snapshot digest, and correlation identity. They MUST NOT contain source payloads, local overlay content, credentials, tokens, proofs, or secrets. qxctl never gains raw append authority.

## Upgrade and Recovery

Schemas are immutable. New writers emit the current protocol. Explicit bounded legacy readers MAY preserve compatible prior evidence. Unknown critical fields or versions are preserved and rejected for mutation. Multiple exact versions coexist; forward and reverse selection use the same protected binding circuit. No newest, timestamp, path, or directory-order inference is permitted.

## Non-Authorization Statement

SAV cannot authenticate, authorize, ratify, write canonical knowledge, fabricate completeness, resolve ambiguous ownership, choose an installation, mutate Maestro, execute SEV plans, seal a Named Version, persist a canonical database, contact a network, or enter a hot/warm path.
