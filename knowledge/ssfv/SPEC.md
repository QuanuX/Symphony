# Symphony Semantic Feature Vector Specification

## Status and Normative Terms

Architect-ratified engine implementation and partial-catalog contract. MUST, MUST NOT, SHOULD, SHOULD NOT, and MAY are normative. The canonical registry contains exactly eighty-six experimental records; the bounded SSFV engine and qxctl client remain without canonical apply, naming, or semantic-decision authority.

## Purpose

Define canonical feature identity, semantics, hierarchy, lifecycle, placement, routing, validation, proposal, freshness, and graph-projection boundaries.

## Layer 0 Canonical Surfaces

The canonical vector surfaces are:

- `knowledge/ssfv/INTENT.md`;
- `knowledge/ssfv/MANIFEST.md`;
- `knowledge/ssfv/SKILL.md`;
- `knowledge/ssfv/SPEC.md`;
- `knowledge/ssfv/COVERAGE.md`;
- `knowledge/ssfv/NAMESPACES.md`;
- `knowledge/ssfv/REGISTRY.md`;
- `knowledge/ssfv/FEATURE-FILE-FORMAT.md`;
- `knowledge/ssfv/schemas/v1/`;
- `knowledge/ssfv/schemas/v2/`.

Future distributed `FEATURES.md` files become canonical only when explicitly registered by SSFV and indexed by SKVI. Generated projections and external graph stores remain noncanonical.

## Stable Identity

A feature identifier has the exact form:

```text
ssfv:<namespace>:<stable.dotted-key>
```

It MUST match:

```text
^ssfv:[a-z][a-z0-9-]{0,62}:[a-z][a-z0-9]*(?:[.-][a-z0-9]+)*$
```

The identifier is an SSFV internal stable ID, not a URI scheme. The first-party namespace is `ssfv:symphony:`. Renaming a title, moving code, or changing a registry path MUST NOT silently change the stable ID.

Namespaces MUST be registered in `NAMESPACES.md` before use. Namespace allocation MUST NOT imply trademark ownership, package ownership, external identity, or runtime authority.

## Feature Kinds

Exactly four kinds exist:

- `capability`: a root-level application or platform reason for existing;
- `feature`: a distinct system behavior within a capability;
- `subfeature`: a bounded subsystem behavior within a feature;
- `microfeature`: a small but materially capability-bearing behavior at the narrowest useful boundary.

Folder depth does not select kind. Kind expresses semantic containment.

## Feature-Worthiness Gate

All five criteria in `INTENT.md` MUST pass before a canonical feature record is created. Tools MAY report candidates, evidence gaps, or likely parent relationships. They MUST NOT autonomously pass the semantic gate.

Language choice, line count, directory existence, public visibility, and caller type are never sufficient evidence by themselves.

## Lifecycle

Canonical lifecycle states are:

- `experimental`: implemented behavior whose interface or semantics may still change;
- `implemented`: available behavior supported by the cited implementation evidence;
- `deprecated`: implemented behavior retained temporarily with a replacement or terminal reason;
- `retired`: behavior no longer available, retained as semantic lineage.

`planned` is not a canonical lifecycle state. A proposal may describe prospective capability without registering it as present application truth.

## Hierarchy and Relationships

Every non-root record MUST have exactly one primary parent. A root `capability` MUST use a null parent. A `feature` has a `capability` parent, a `subfeature` has a `feature` parent, and a `microfeature` has a `subfeature` parent. Primary-parent cycles, missing targets, self-links, and duplicate links are prohibited.

Typed crosslinks are:

- `depends_on`;
- `enables`;
- `composes_with`;
- `extends`;
- `alternative_to`;
- `supersedes`;
- `distinguished_from`.

Each `distinguished_from` relationship MUST include a factual distinction. Crosslinks never create a second primary parent.

## Required Semantics

Every feature record MUST answer:

- `who`: systems, roles, callers, or integrations that use or receive the capability;
- `what`: the observable capability;
- `how`: the bounded mechanism without duplicating implementation documentation;
- `when`: lifecycle, trigger, session, or operating conditions;
- `where`: source scope, runtime boundary, and deployment context;
- `why`: the durable reason the capability exists.

Descriptions MUST be caller-neutral. They may describe a caller category as usage context, but MUST NOT derive authority from that category.

Each record also contains exact implementation paths, implementation languages with their roles, distinctions, typed relationships, applicable cross-vector references, evidence, and explicit non-claims.

## Implementation Evidence

Implementation paths MUST be repository-relative, normalized, explicit, non-directory-traversing paths. Implicit globs are prohibited in v1. A source file, contract, or build surface may support multiple features, and a feature may span multiple exact paths.

Language is role evidence, not an automatic feature. Each language entry explains what that language implements. C++ is the default runtime language where suitable; Go remains the qxctl administration language; platform adapters may use a required native language. Thermal-path doctrine is owned elsewhere and MUST NOT be invented by SSFV.

## Distributed FEATURES.md Contract

A `FEATURES.md` file is permitted only at a source scope owning one or more ratified records. It MUST:

- identify the exact repository-relative owner scope;
- contain complete records rather than links alone;
- use stable IDs registered in `REGISTRY.md`;
- avoid copying records owned by child scopes;
- link parent and child records through stable identities;
- remain sparse when no feature-worthy behavior exists.

The current partial catalog is ratified and contains exactly one root owner file and fourteen nested owner files. Any additional `FEATURES.md` or feature record remains separately gated by the complete feature-worthiness and reviewed-change procedure.

Every registered feature file uses the exact managed-region and embedded JSON-envelope grammar in `FEATURE-FILE-FORMAT.md`. The exact literal `.` represents repository-root source scope and owns root `FEATURES.md`; any other normalized directory scope owns `<source_scope>/FEATURES.md`.

The embedded envelope contains one or more complete v2 feature records. A feature ID is globally unique. Several records may share one owner file and the same `feature_file`, `owner_contract`, and `source_scope` routing tuple. One source scope maps to one routing tuple, and one feature identity has one canonical owner file.

## Registry Contract

`REGISTRY.md` maps each stable feature ID to exactly one canonical `FEATURES.md`, owner contract, source scope, lifecycle state, primary parent, and record digest. An explicitly empty registry is valid.

Registry entries MUST use the grammar in `REGISTRY.md`, be unique by ID, use a consistent owner routing tuple, refer to regular no-follow repository files and directory scopes, and have SKVI coverage. Registry routing never replaces the distributed record.

## Cross-Vector Boundaries

- SSFV owns capability semantics.
- SKVI owns canonical location and relationship routing.
- SCLV owns reviewed change records.
- SACV owns OpenAPI and HTTP contract semantics.
- SODV owns release and publication authorization.
- STAV owns safe audit-event truth.
- SSIAG owns identity, authentication, access, and governance.
- Maestro owns persisted installation and docking state.

A feature record references those surfaces when applicable and uses explicit not-applicable reasoning when a schema or operation requires a relationship that does not apply. It MUST NOT restate their canonical content.

## Deterministic and Semantic Inputs

Deterministic facts such as paths, digests, declared languages, registry coverage, and relationship integrity MAY be generated or checked. Semantic claims such as feature-worthiness, purpose, distinctions, and lifecycle acceptance remain caller-declared and permission-backed.

Tool output MUST distinguish deterministic findings from unratified semantic proposals.

Feature IDs supplied to administration assurance are actual stable SSFV identifiers, not labels generated by the engine. A new module descriptor MAY initially declare no feature IDs or an `unreviewed` administration disposition so the independently installed engine can deterministically report `semantic_registration_required` or `administration_unintegrated` from supplied evidence. Repository admission separately discovers bounded implemented module roots that lack an owner `FEATURES.md`, registry route, or administration-profile entry. These checks expose omission without requiring qxctl or AI, but neither invents a name, ratifies feature semantics, infers an exemption from silence, scans an installed host without supplied inventory, or proves installed-package completeness.

## SSFV Engine Operations

The implemented operation set is:

- `inspect`: report contract, namespace, registry, and installed-engine state;
- `check`: validate structural integrity and produce semantic-freshness evidence;
- `diff`: compare bounded content-addressed feature snapshots;
- `propose`: create an immutable bounded multi-file change proposal without canonical writes;
- `graph`: emit a disposable portable JSON graph projection.

The executable is `symphony-ssfv`, module ID is `ssfv-engine`, and qxctl namespace is `qxctl ssfv`. The engine uses the common `symphony.knowledge.engine-process.v1` process envelope and installs independently as inactive `installed_undocked`.

`inspect` accepts exactly `{}`. `graph` accepts exactly `{"format":"json"}`. Check, diff, and proposal inputs conform to the executable v2 schemas. Results bind the exact engine and supported contract versions.

## Session and Freshness Contract

Every result MUST bind to content-addressed canonical-contract, namespace, registry, distributed-record, and relevant source snapshots. A result from stale inputs MUST NOT be represented as current.

Structural integrity checking is mandatory. The implemented per-invocation freshness modes are `disabled`, `report`, and `require`. `report` produces caller-neutral semantic-review candidates; `require` makes unresolved stale semantics unsuccessful. Persistent safeguard profiles, automatic background execution, and canonical apply remain deferred. Session boundaries remain configurable direction; the implemented default spans authentication through logout or mandatory reauthentication.

The separately installed knowledge-session coordinator owns a protected noncanonical SSFV maintenance stream keyed by TOPS, authenticated subject, and repository. `qxctl knowledge session features begin` stores the initial complete semantic snapshot and its exact engine identity. `checkpoint` and `close` obtain a current snapshot, compare it to that immutable baseline through the selected SSFV engine, and persist the result as review evidence. `status` is read-only. `recover` is explicit, compare-and-swap bound, and may select only one uniquely valid forward state. The baseline engine and current engine are recorded separately so compatible upgrade or rollback order cannot reinterpret the original evidence.

Every mutating maintenance operation requires an open authenticated-session journal digest, the exact qxctl binding-registry digest, a fresh operation/resource-bound SSIAG decision, and either a complete read-only Maestro inventory observation or an explicit `not_configured` evidence object. The derived Maestro inventory binds current docking lineage without becoming feature truth. A finding can change maintenance state to `review_required`; no maintenance operation edits a `FEATURES.md`, registry, namespace, SKVI entry, or other canonical surface.

No unresolved structural error may be silently carried into a later session as canonical truth.

`coverage_state` reports semantic-catalog coverage, not merely registry closure. `empty` means no feature record is registered. `partial` means one or more records exist in an incrementally bootstrapped catalog, including when every registered relationship is structurally valid. `COVERAGE.md` now defines the explicit source universe, exclusions, evidence, freshness, and completion rule; its current disposition remains `partial` because nested feature, subfeature, and microfeature review is incomplete. `complete` MUST NOT be emitted until every condition in that contract is owner-ratified and mechanically satisfied. Successful structural validation alone never establishes repository-wide feature completeness.

## Proposal and Mutation Boundary

An engine proposal may coordinate a bounded update to a distributed `FEATURES.md`, `REGISTRY.md`, `NAMESPACES.md`, and SKVI. It includes exact expected digests, paths, operation intent, expiry, affected feature IDs, and one caller-declared desired namespace or feature record.

`propose` never mutates canonical files. The implemented coordinator recovery and locking mechanisms apply only to protected noncanonical maintenance evidence. Canonical apply, canonical rollback, semantic permission to edit source truth, automatic session-close hooks, and audit emission require separately ratified designs.

A prospective new path binds a typed path-specific absence digest plus `target_must_be_absent`; absence is not represented as an empty-file digest.

## Graph Projection

The v1 graph projection is portable JSON. It contains feature nodes, primary-parent edges, typed crosslinks, source digests, and a projection digest. It is noncanonical and rebuildable.

No graph database, daemon, network listener, or persistent graph store is authorized. A future provider may consume the projection only after its own install, permission, consistency, and source-lineage contract is ratified.

## Resource Bounds

The schemas bound individual strings, arrays, records, and snapshots. The engine additionally bounds requests to 1 MiB, responses to 4 MiB, a file to 4 MiB, total SSFV evidence reads to 64 MiB, feature files to 1,024, feature records to 8,192, graph edges to 32,768 within the response ceiling, and deadlines to the common 300-second maximum. Maintenance commands embed their complete semantic evidence and therefore share the coordinator's 1 MiB request ceiling; qxctl and the coordinator fail before journal mutation when that bound is exceeded. It fails closed on unreadable files, symlinks, traversal, special files, duplicate identities, invalid parent progression, cycles, digest mismatch, ambiguous markers, incomplete records, or excessive output.

## Non-Authorization Statement

This specification authorizes canonical SSFV governance, the bounded independently installed engine and qxctl client, the exact eighty-six-record partial catalog, the explicit owner-scope coverage inventory, ratified nested-review progress, invariant-, provider-trust-, and provider-binding-assurance reporting, and the protected noncanonical maintenance composition above. It does not authorize an unratified distributed feature record, a repository- or installed-host-completeness claim, complete legacy-invariant coverage, canonical apply, Maestro state mutation through SSFV, persistent graph storage, a remote interface, public documentation, or an application capability claim outside those records.
