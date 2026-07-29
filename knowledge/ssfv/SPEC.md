# Symphony Semantic Feature Vector Specification

## Status and Normative Terms

Architect-ratified contract transition. MUST, MUST NOT, SHOULD, SHOULD NOT, and MAY are normative. The canonical registry is empty, and no SSFV runtime is implemented by this specification.

## Purpose

Define canonical feature identity, semantics, hierarchy, lifecycle, placement, routing, validation, proposal, freshness, and graph-projection boundaries.

## Layer 0 Canonical Surfaces

The canonical vector surfaces are:

- `knowledge/ssfv/INTENT.md`;
- `knowledge/ssfv/MANIFEST.md`;
- `knowledge/ssfv/SKILL.md`;
- `knowledge/ssfv/SPEC.md`;
- `knowledge/ssfv/NAMESPACES.md`;
- `knowledge/ssfv/REGISTRY.md`;
- `knowledge/ssfv/schemas/v1/`.

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

Every non-root record MUST have exactly one primary parent. A root `capability` MUST use a null parent. Cycles are prohibited.

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

The first bootstrap is separately gated. This specification does not create a root or nested `FEATURES.md` file.

## Registry Contract

`REGISTRY.md` maps each stable feature ID to exactly one canonical `FEATURES.md`, owner contract, source scope, lifecycle state, primary parent, and record digest. An explicitly empty registry is valid.

Registry entries MUST use the grammar in `REGISTRY.md`, be unique by ID and owner location, refer to regular no-follow repository files, and have SKVI coverage. Registry routing never replaces the distributed record.

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

## Future SSFV Engine Operations

The reserved initial operation set is:

- `inspect`: report contract, namespace, registry, and installed-engine state;
- `check`: validate structural integrity and produce semantic-freshness evidence;
- `diff`: compare bounded content-addressed feature snapshots;
- `propose`: create an immutable bounded multi-file change proposal without canonical writes;
- `graph`: emit a disposable portable JSON graph projection.

The reserved executable is `symphony-ssfv`, module ID is `ssfv-engine`, and qxctl namespace is `qxctl ssfv`. The future engine uses the common `symphony.knowledge.engine-process.v1` process envelope. None of these surfaces is implemented by this contract transition.

## Session and Freshness Contract

Every result MUST bind to content-addressed canonical-contract, namespace, registry, distributed-record, and relevant source snapshots. A result from stale inputs MUST NOT be represented as current.

Structural integrity checking is mandatory. Semantic freshness is an administrator-controlled safeguard that MAY fail a session-close or apply gate when enabled. The default safeguard policy and its qxctl controls require separate implementation review. Session boundaries are configurable; the default spans authentication through logout or mandatory reauthentication.

No unresolved structural error may be silently carried into a later session as canonical truth.

## Proposal and Mutation Boundary

A future proposal may coordinate a bounded update to a distributed `FEATURES.md`, `REGISTRY.md`, and SKVI. It MUST include exact expected digests, paths, operation intent, expiry, and affected feature IDs.

`propose` never mutates canonical files. Canonical apply, recovery, rollback, locking, permission evaluation, session-close behavior, and audit emission require a separately ratified qxctl mutation design.

## Graph Projection

The v1 graph projection is portable JSON. It contains feature nodes, primary-parent edges, typed crosslinks, source digests, and a projection digest. It is noncanonical and rebuildable.

No graph database, daemon, network listener, or persistent graph store is authorized. A future provider may consume the projection only after its own install, permission, consistency, and source-lineage contract is ratified.

## Resource Bounds

The schemas bound individual strings, arrays, records, and snapshots. A future implementation MUST additionally bound file size, record count, traversal depth, total input bytes, runtime, memory, and output size, and MUST fail closed on unreadable files, symlinks, traversal, duplicate identities, cycles, digest mismatch, or incomplete records.

## Non-Authorization Statement

This specification authorizes canonical SSFV governance and machine-readable payload contracts only. It does not authorize an engine implementation, distributed feature bootstrap, canonical apply, qxctl command, Maestro docking, persistent graph store, remote interface, public documentation, or capability claim.
