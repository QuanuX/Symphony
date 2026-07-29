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

No distributed `FEATURES.md` record exists at contract-transition time. The explicit empty registry is canonical and valid.

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
- Maestro may later persist inactive or docked engine state; it does not own feature semantics.
- qxctl is the implemented inspect/check/diff/propose/graph interface and eventual lifecycle administrator; it does not own SSFV truth.

## Installability

The independently installable C++ engine lives at `modules/ssfv-engine/` with executable `symphony-ssfv` and module identifier `ssfv-engine`. It installs under an exact versioned prefix as inactive `installed_undocked`, including alongside other compatible engine versions. Its operation vocabulary is `inspect`, `check`, `diff`, `propose`, and `graph`.

The Go qxctl client validates the exact inactive-undocked receipt and invokes the selected version out of process. Generic qxctl installation, activation, receptor selection, docking, and canonical apply remain deferred.

## Non-Authorization Statement

This manifest authorizes the canonical SSFV contract and bounded engine/client implementation. It does not authorize feature bootstrap, application `FEATURES.md` creation, canonical apply, repository mutation, graph-database persistence, Maestro docking, public documentation, or marketing claims.

## Status

Architect-ratified engine implementation. Namespace `symphony` is allocated, the registry is intentionally empty, and no application feature record exists.
