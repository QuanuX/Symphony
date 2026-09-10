# Symphony Cloud Vector Engine Manifest

## Canonical Surfaces

- `modules/scv-engine/FEATURES.md`
- `modules/scv-engine/INSTALL.md`
- `modules/scv-engine/INTENT.md`
- `modules/scv-engine/MANIFEST.md`
- `modules/scv-engine/SKILL.md`
- `modules/scv-engine/SPEC.md`

## Identity

- module: `scv-engine`
- engine: `symphony-scv`
- vector: `scv`
- version: `0.1.0-dev`
- language: C++26
- thermal path: freezing
- semantic owner: `knowledge/scv/SPEC.md`

## Package Boundary

The module owns its exact versioned executable, receipt, installed contract documents and licenses. It shares implementation source with `modules/scv-engine/`; source reuse does not merge package identity, select an active version, or require its parent/provider siblings at runtime. The immutable receipt declares the actual owned files and entry point. No active alias, service, listener, login hook or Maestro presence is implied.

## Domain State

Source configurations, captured artifacts, accepted interpretations and selected graph revisions are distinct private installation data. The process receives bounded explicit inputs and emits bounded evidence or transitions. It cannot rewrite canonical repository knowledge, grant permissions or commit remote provider changes.
