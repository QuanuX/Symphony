# SKVI Engine Specification

## Status

Implemented read-only/proposal development slice, version `0.1.0-dev`. It is not a published module release.

## Process and Exit Contract

The engine uses the exact common bounded process envelope. Direct diagnostics are `--help`, `--version`, and `--descriptor`. Process exit statuses are `0` for a completed operation, `2` for malformed input, `3` for protocol/target/deadline mismatch, `4` for invalid or unsupported operation semantics, and `5` for bounded repository or internal failure.

## `inspect`

Payload: exact empty object. The result reports the descriptor, canonical index path, implemented operation set, read-only state, and disabled canonical apply.

## `check`

Payload: exact `expected_index_digest`, which is `null` or a tagged SHA-256 digest. The operation parses `knowledge/skvi/INDEX.md`, checks required field presence, canonical status, unique safe paths, no-follow regular-file existence, required SKVI/umbrella coverage, and indexed relationship targets. Findings are evidence; an invalid index produces a completed check result with `state: invalid` rather than a canonical mutation or repair.

The exact result is governed by `knowledge/skvi/schemas/v1/check-result.schema.json`.

## `propose`

Payload is governed by `knowledge/skvi/schemas/v1/operation-payload.schema.json`. Repository identity and revision are provider-neutral caller input. Created/expiry timestamps are caller-declared strict UTC values and therefore remain deterministic input rather than engine wall-clock output.

The caller explicitly selects `add_entry`, `replace_entry`, or `remove_entry`. Add requires an existing unindexed regular file. Replace/remove require the exact current entry digest. The engine checks the current index before proposing and rejects ambiguous or stale operations.

The result conforms to `knowledge/schemas/v1/proposal.schema.json`, binds the index and SKVI contract read set, records a prospective write set targeting only `knowledge/skvi/INDEX.md`, and states `engine_decided_membership: false`. It does not include a patch or write canonical content.

## `project`

Payload: exact `format: "json"`. A clean check is required. The result conforms to `knowledge/skvi/schemas/v1/projection.schema.json` and contains normalized entries plus input, contract-snapshot, engine, and projection digests. It is returned in-process, never written by the engine, and is noncanonical and rebuildable.

## Bounds

The common request, response, JSON, path, file, count, and deadline limits apply. SKVI additionally permits at most 512 entries, 64 KiB per normalized field, 1,024 exception-evidence items, and one proposal operation. Successful subchecks are retained as deterministic aggregate counts rather than repeated evidence objects so a healthy maximum-size projection stays within the common JSON value envelope. Projection format `json` is the only implemented format in this version.

## Non-Authorization

The engine has no authentication, permission, ratification, session, apply, filesystem-write, hook, watcher, network, provider, release, STAV, SSIAG, qxctl-lifecycle, or Maestro authority. A valid proposal or projection is evidence only.

## Development Packaging Constraint

Version `0.1.0-dev` requires the exact `libexec` executable and `share` receipt/document/license layout consumed by the qxctl exact-version resolver. CMake rejects customized install-directory names rather than producing an installation that appears valid but cannot complete the documented invocation circuit.
