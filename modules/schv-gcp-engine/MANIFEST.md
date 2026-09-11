# Symphony Cloud Hyperscalers Vector — Google Cloud Engine Manifest

## Canonical Surfaces

- `modules/schv-gcp-engine/FEATURES.md`
- `modules/schv-gcp-engine/INSTALL.md`
- `modules/schv-gcp-engine/INTENT.md`
- `modules/schv-gcp-engine/MANIFEST.md`
- `modules/schv-gcp-engine/SKILL.md`
- `modules/schv-gcp-engine/SPEC.md`

## Identity

- module: `schv-gcp-engine`
- engine: `symphony-schv-gcp`
- vector: `schv-gcp`
- version: `0.5.0-dev`
- language: C++26
- thermal path: freezing
- semantic owner: `knowledge/scv/schv/gcp/SPEC.md`

## Package Boundary

The module owns its exact versioned executable, receipt, installed contract documents and licenses. It shares implementation source with `modules/scv-engine/`; source reuse does not merge package identity, select an active version, or require its parent/provider siblings at runtime. The immutable receipt declares the actual owned files and entry point. No active alias, service, listener, login hook or Maestro presence is implied.

## Domain State

Source configurations, captured artifacts, accepted interpretations and selected graph revisions are distinct private installation data. The process receives bounded explicit inputs and emits bounded evidence or transitions. It cannot rewrite canonical repository knowledge, grant permissions or commit remote provider changes.

The `0.3.0-dev` descriptor has twenty operations, including the four corpus operations owned by `knowledge/scv/CORPUS.md` and three profile/connection operations owned by `knowledge/scv/INTERPRETATION.md`. Immutable corpus bytes and job bookkeeping are qxctl adapter state, not this process's package files or a selected corpus head. Existing exact `0.1.0-dev` and `0.2.0-dev` installations remain independently invocable; interpretation wrappers retain authored mappings and their exact evidence without altering prior data shapes.

The additive `0.4.0-dev` package exposes 21 operations, including native profile preparation, and owns its exact schema catalog, schemas and templates. `knowledge/scv/AGENT-WORKFLOWS.md` defines the new engine/CLI boundary. Retained workflow evidence is outside immutable package files; `.3` and earlier installations remain separately invocable.

## Maintained Provider Coverage

The additive `0.5.0-dev` package has 22 operations. `knowledge/scv/COVERAGE.md` owns native accounting of declared sources, exact corpus selection and independently replayed interpretations. Missing, unlisted, unselected, partial and stale evidence remains explicit. The operation does not rank providers or establish runtime compatibility. Earlier exact `.4` and prior installations remain preserved.
