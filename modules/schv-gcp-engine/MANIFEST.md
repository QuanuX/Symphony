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
- version: `0.1.0-dev`
- language: C++26
- thermal path: freezing
- semantic owner: `knowledge/scv/schv/gcp/SPEC.md`

## Package Boundary

The module owns its exact versioned executable, receipt, installed contract documents and licenses. It shares implementation source with `modules/scv-engine/`; source reuse does not merge package identity, select an active version, or require its parent/provider siblings at runtime. The immutable receipt declares the actual owned files and entry point. No active alias, service, listener, login hook or Maestro presence is implied.

## Domain State

Source configurations, captured artifacts, accepted interpretations and selected graph revisions are distinct private installation data. The process receives bounded explicit inputs and emits bounded evidence or transitions. It cannot rewrite canonical repository knowledge, grant permissions or commit remote provider changes.
