# Symphony Semantic Feature Vector Manifest

## Canonical Target

`knowledge/ssfv/`

## Identity

SSFV is the Symphony Semantic Feature Vector.

## Classification

- autonomous Symphony Knowledge Vector contract surface;
- canonical feature vocabulary, namespace, and routing authority;
- distributed semantic feature-truth coordinator;
- freezing-path administrative capability;
- not source code, change history, deployment state, public documentation, or a graph database.

## Declared Contract Truth Role

SSFV owns:

- stable feature-ID grammar and namespace allocation;
- the capability, feature, subfeature, and microfeature kinds;
- the experimental, implemented, deprecated, and retired lifecycle states;
- primary-parent and typed crosslink rules;
- the feature-worthiness test;
- 5W1H and distinction requirements;
- distributed `FEATURES.md` placement and record grammar;
- feature registry routing;
- deterministic engine payload and result schemas;
- derived graph-projection boundaries.

The source scope that owns a feature owns its canonical distributed record. SSFV MUST NOT centralize semantic copies merely for tooling convenience.

## Canonical Surface

```text
knowledge/ssfv/
  INTENT.md
  MANIFEST.md
  SKILL.md
  SPEC.md
  NAMESPACES.md
  REGISTRY.md
  FEATURE-FILE-FORMAT.md
  schemas/v1/
  schemas/v2/
```

The current partial catalog distributes exactly four canonical records across `FEATURES.md`, `libraries/knowledge-vector-engine-cpp/FEATURES.md`, `modules/knowledge-session-coordinator/FEATURES.md`, and `modules/maestro/FEATURES.md`. `REGISTRY.md` routes those records, and SKVI indexes all four owner files. No other feature record or complete-catalog claim is authorized.

## Record Model

Every record has:

- one stable `ssfv:<namespace>:<stable.dotted-key>` identity;
- one kind and lifecycle state;
- zero or one primary parent, with no parent only for a root capability;
- one owner contract and one bounded source scope;
- complete who, what, how, when, where, and why semantics;
- implementation paths and language-role evidence;
- typed relationships, distinctions, cross-vector references, evidence, and non-claims.

## Relationship to Other Truth Surfaces

- SKVI indexes each canonical SSFV surface and distributed feature file.
- SCLV records reviewed changes after actual merge evidence exists.
- SACV owns API description contracts referenced by features.
- SODV governs feature-derived publication.
- STAV records safe operational audit metadata where separately authorized.
- SSIAG controls permission-backed feature administration.
- Maestro persists exact authenticated docking presence and derives complete read-only receptor inventory evidence; it does not own feature semantics or execute recorded engines.
- qxctl is the implemented inspect/check/diff/propose/graph interface and session-maintenance administrator; it does not own SSFV truth.

## Installability

The independently installable C++ engine lives at `modules/ssfv-engine/` with executable `symphony-ssfv` and module identifier `ssfv-engine`. It installs under an exact versioned prefix as inactive `installed_undocked`, including alongside other compatible engine versions. Its operation vocabulary is `inspect`, `check`, `diff`, `propose`, and `graph`.

The Go qxctl client validates the exact inactive-undocked receipt and invokes the selected version out of process. `qxctl knowledge session features begin|status|checkpoint|close|recover` composes exact SSFV engine evidence, an open authenticated-session digest, optional complete Maestro inventory evidence, and the separately installed knowledge-session coordinator. Generic engine installation, activation, receptor selection, and canonical apply remain deferred; lifecycle-administered installation, activation, and docking are governed separately by `knowledge/LIFECYCLE.md`.

## Non-Authorization Statement

This manifest authorizes the canonical SSFV contract, bounded engine/client implementation, exact four-record partial catalog, and protected noncanonical session-maintenance evidence. It does not authorize an additional application `FEATURES.md`, another feature record, repository-wide completeness, canonical apply, repository mutation by SSFV tooling, graph-database persistence, Maestro state mutation by SSFV, public documentation, or marketing claims.

## Status

Architect-ratified engine implementation and partial catalog. Namespace `symphony` is allocated, exactly four experimental application-feature records exist, and coverage remains explicitly partial.
