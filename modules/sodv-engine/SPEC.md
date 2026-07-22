# SODV Engine Specification

## Status

Implemented read/proposal development slice, version `0.1.0-dev`. It is source-installable and not a published module release.

## Process Contract

The engine uses `symphony.knowledge.engine-process.v1`. Direct diagnostics are `--help`, `--version`, and `--descriptor`. Exit statuses are 0 completed, 2 malformed input, 3 process protocol mismatch, 4 invalid operation semantics, and 5 bounded repository/internal failure.

## Operations

- `inspect`: exact empty payload; reports the ledger and disabled authority surfaces.
- `check`: exact nullable expected-ledger digest; validates the append-only legacy-v1 and provider-neutral-v2 record relationships.
- `verify`: exact `observed-state.schema.json`; compares raw caller-supplied observations with canonical authorization/correction/completion truth.
- `propose`: exact `proposal-input.schema.json`; proposes one forward v2 authorization, correction, completion, or failure record targeting only `knowledge/sodv/RELEASES.md`.
- `recover`: exact `recovery-input.schema.json`; reconciles a digest-bound noncanonical journal with observed state without editing or deleting it.
- `project`: exact `format: json`; emits a noncanonical, rebuildable release-transaction inventory.

## Provider and Evidence Boundary

The v1 engine accepts only `go_module` publication units because those are the only completed public artifacts currently recorded. Coordinates, repository providers, and source references remain provider-neutral strings. The caller gathers Git and public-package evidence; the engine validates shapes, digests, relationships, and mismatches but performs no network request and does not trust a warm cache as completion evidence.

## Bounds

The common request, response, JSON, path, file, count, and deadline bounds apply. SODV permits 4,096 records, 128 units per record/observation, 2,048 findings, and eight subject records per v2 record. Canonical files are bounded no-follow regular files.

## Non-Authorization

No operation authenticates, grants release permission, writes a file, creates or moves a tag, uploads an artifact, contacts a package proxy or checksum database, appends the release ledger, publishes documentation, configures Mintlify, automates NotebookLM, starts a listener, activates a version, or docks with Maestro.
