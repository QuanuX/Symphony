# Symphony Cloud Edge Vector — Cloudflare Engine Manifest

## Canonical Surfaces

- `modules/scev-cf-engine/FEATURES.md`
- `modules/scev-cf-engine/INSTALL.md`
- `modules/scev-cf-engine/INTENT.md`
- `modules/scev-cf-engine/MANIFEST.md`
- `modules/scev-cf-engine/SKILL.md`
- `modules/scev-cf-engine/SPEC.md`

## Identity

- module: `scev-cf-engine`
- engine: `symphony-scev-cf`
- vector: `scev-cf`
- version: `0.1.0-dev`
- language: C++26
- thermal path: freezing
- semantic owner: `knowledge/scv/scev/cf/SPEC.md`

## Package Boundary

The module owns its exact versioned executable, receipt, installed contract documents and licenses. It shares implementation source with `modules/scv-engine/`; source reuse does not merge package identity, select an active version, or require its parent/provider siblings at runtime. The immutable receipt declares the actual owned files and entry point. No active alias, service, listener, login hook or Maestro presence is implied.

## Domain State

Source configurations, captured artifacts, accepted interpretations and selected graph revisions are distinct private installation data. The process receives bounded explicit inputs and emits bounded evidence or transitions. It cannot rewrite canonical repository knowledge, grant permissions or commit remote provider changes.
