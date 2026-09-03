# SCLV Engine Specification

## Status

Implemented proposal-only development slice, version `0.1.0-dev`. It is source-installable and not a published module release.

## Process Contract

All three executables use `symphony.knowledge.engine-process.v1`, strict bounded JSON, one standard-output response, deterministic response digests, and exit statuses 0/2/3/4/5. Direct diagnostics are `--help`, `--version`, and `--descriptor`.

## Engine Operations

- `inspect`: exact empty payload.
- `check`: exact nullable `expected_ledger_digest`.
- `propose`: exact `knowledge/sclv/schemas/v3/proposal-input.schema.json` input.
- `recover`: exact `knowledge/sclv/schemas/v3/recovery-input.schema.json` input; reconciliation only.
- `project`: exact `format: json`.

`propose` requires a clean current ledger, one unique v3 record, exact revision/tree evidence, asserted ratification evidence matching the record, safe bounded affected paths, current no-follow regular SKVI-reference paths, nondecreasing recording time, and caller-declared proposal expiry. Affected paths are historical provenance and may be absent after the recorded change removes them. Only surviving `skvi_references` enter the proposal read set and must remain current-SKVI indexed. It returns one common immutable proposal containing deterministic Markdown but performs no append.

`check` applies the temporal split without rewriting a legacy or v3 record. All recorded paths retain safe bounded syntax. A historical SKVI reference absent from the current-only index produces non-failing warning evidence; it does not retroactively invalidate the record bound to its historical revision and tree. A reference still claimed by current SKVI must resolve as a current no-follow regular file, or the current index contradiction is a violation. A stable-identity retirement record may cite the root namespace tombstone; ordinary removed prose requires no tombstone. These rules add no record field and preserve the existing v3 wire and Markdown shapes.

`recover` returns `symphony.sclv.recovery-result.v1` with an explicit `resume`, `abandon`, `no_op`, or `propose_late_recovery` action. Every result states `journal_mutated: false` and `canonical_apply_enabled: false`; only late recovery contains a nested proposal.

## Adapter Operations

Each adapter supports `normalize`. Local Git accepts a full SHA-1/SHA-256 commit, runs fixed `/usr/bin/git` commands without a shell, bounds output, and hashes the recursive tree listing. Air-gap normalization binds declared revision, change-request, and ratification metadata to a normalized evidence digest. Well-formed evidence is not proof of permission.

## Bounds

The common 1 MiB request, 4 MiB response, JSON depth/value/string, path, snapshot, and deadline bounds apply. SCLV additionally permits 4,096 records, 1,024 affected/index references per record, 1,024 exception findings, and eight provider-evidence envelopes per proposal.

## Non-Authorization

No operation authenticates, grants permission, ratifies, applies, mutates a ledger or journal, edits Git/provider state, uses a network, invokes SSIAG/STAV, activates a version, docks with Maestro, or participates in hot/warm paths.
