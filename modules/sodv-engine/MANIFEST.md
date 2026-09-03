# SODV Engine Manifest

## Canonical Surfaces

- `modules/sodv-engine/FEATURES.md`
- `modules/sodv-engine/INSTALL.md`
- `modules/sodv-engine/INTENT.md`
- `modules/sodv-engine/MANIFEST.md`
- `modules/sodv-engine/SKILL.md`
- `modules/sodv-engine/SPEC.md`

## Identity

- Module: `sodv-engine`
- Executable: `symphony-sodv`
- Version: `0.1.0-dev`
- Language: C++26
- Thermal path: freezing
- State after installation: inactive and undocked

## Contract

The engine implements `knowledge/sodv/` and the common knowledge-engine process/proposal contracts. Canonical truth remains in `knowledge/sodv/RELEASES.md` and its Contract Quad.

## Operations

`inspect`, `check`, `verify`, `propose`, `recover`, and `project` are implemented. `apply`, tag creation, publication, upload, proxy access, and listener operations are disabled.

## Installation

The exact `0.1.0-dev` receipt owns one versioned executable, one receipt, five contract documents, and two license files. No unversioned alias, service, socket, active receptor, or default docking state is installed.

## Boundaries

The engine receives observed external state from the caller and never contacts Git hosting, a package proxy, a checksum database, Mintlify, NotebookLM, or another network service. It cannot grant release permission, create or move tags, append records, declare completion independently, or publish artifacts.
