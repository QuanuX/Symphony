# SAV Engine Manifest

## Identity

- module: `sav-engine`
- engine: `symphony-sav`
- vector: `sav`
- version: `0.1.0-dev`
- language: C++26
- thermal path: freezing

## Runtime

The executable implements `inspect`, `reference_check`, `current_resolve`, `evaluate`, `diff`, `explain`, `project_graph`, and `compatibility`. All operations are deterministic, bounded, non-mutating, caller-neutral, and available through `symphony.knowledge.engine-process.v1`.

## Packaging

Receipt v2 installs the exact binary, documentation, and licenses side by side as `installed_undocked`. Uninstall removes only receipt-owned paths.
