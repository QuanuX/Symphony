# Symphony Cloud Hyperscalers Vector — Azure Engine Manifest

## Canonical Surfaces

- `modules/schv-azure-engine/FEATURES.md`
- `modules/schv-azure-engine/INSTALL.md`
- `modules/schv-azure-engine/INTENT.md`
- `modules/schv-azure-engine/MANIFEST.md`
- `modules/schv-azure-engine/SKILL.md`
- `modules/schv-azure-engine/SPEC.md`

## Identity

- module: `schv-azure-engine`
- engine: `symphony-schv-azure`
- vector: `schv-azure`
- version: `0.2.0-dev`
- language: C++26
- thermal path: freezing
- semantic owner: `knowledge/scv/schv/azure/SPEC.md`

## Package Boundary

The module owns its exact versioned executable, receipt, installed contract documents and licenses. It shares implementation source with `modules/scv-engine/`; source reuse does not merge package identity, select an active version, or require its parent/provider siblings at runtime. The immutable receipt declares the actual owned files and entry point. No active alias, service, listener, login hook or Maestro presence is implied.

## Domain State

Source configurations, captured artifacts, accepted interpretations and selected graph revisions are distinct private installation data. The process receives bounded explicit inputs and emits bounded evidence or transitions. It cannot rewrite canonical repository knowledge, grant permissions or commit remote provider changes.

The `0.2.0-dev` descriptor has seventeen operations, including the four corpus operations owned by `knowledge/scv/CORPUS.md`. Immutable corpus bytes and job bookkeeping are qxctl adapter state, not this process's package files or a selected corpus head. Existing exact `0.1.0-dev` installations remain independently invocable.
