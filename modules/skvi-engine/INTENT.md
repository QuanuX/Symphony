# SKVI Engine Intent

## Purpose

Implement the canonical SKVI contract as an independently installable, authority-free C++ process that checks repository-maintained index truth, prepares caller-declared immutable proposals, and emits disposable structural projections.

## Implemented Scope

Development version `0.1.0-dev` implements bounded `inspect`, `check`, `propose`, and `project` operations through `symphony.knowledge.engine-process.v1`. It reads `knowledge/skvi/INDEX.md`, the SKVI Contract Quad, and indexed repository files through no-follow regular-file access.

## Authority Boundary

The engine never decides whether a surface is feature-worthy or belongs in SKVI. A proposal operation is supplied explicitly by the caller and remains noncanonical. The engine validates shape, paths, expected state, and deterministic evidence; it does not ratify or apply the operation.

## Projection Boundary

Projection output is returned in the bounded process response. It is digest-bound, disposable, and rebuildable and is never written into the repository by the engine.
