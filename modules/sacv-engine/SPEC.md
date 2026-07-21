# SACV Engine Specification

## Status

Implemented read-only/proposal development slice, version `0.1.0-dev`. It is not a published module release.

## Process Contract

The engine uses `symphony.knowledge.engine-process.v1`. Direct diagnostics are `--help`, `--version`, and `--descriptor`. Exit statuses follow the common knowledge-engine contract: 0 completed, 2 malformed input, 3 protocol/target/deadline mismatch, 4 invalid operation semantics, and 5 bounded repository/internal failure.

## Parser Compatibility Boundary

OpenAPI remains pinned to 3.2.0. JSON entry documents receive the complete v1 structural, security, reference, example-safety, and compatibility checks described here. YAML is canonical when owner-justified, but this engine version emits `sacv.document.parser_unavailable` for `.yaml` documents. It never downgrades 3.2.0, shells out, reads a network reference, or claims YAML conformance without an independently ratified parser gate.

## Operations

- `inspect`: exact empty payload; reports the descriptor, parser formats, empty-registry validity, and disabled authority surfaces.
- `check`: exact nullable `expected_registry_digest`; validates registry grammar, uniqueness, ownership placement, SKVI coverage, no-follow files, and registered JSON OpenAPI documents.
- `diff`: exact baseline/candidate relative paths and tagged digests; returns deterministic `identical`, `compatible_additive`, `breaking`, or `review_required` evidence without making an acceptance decision.
- `propose`: common repository/session/context/timestamp envelope plus one explicit `register_contract` or `replace_contract` operation. The candidate owner document must already exist and validate. Output targets only `knowledge/sacv/REGISTRY.md` and contains no patch or write route.
- `project`: exact `format: json`; emits the normalized registry inventory and conformance metadata. It never bundles raw source documents or generates code.

## Bounds and Security

The common request, response, JSON, path, file, count, and deadline bounds apply. SACV permits 256 registry entries, 2,048 diff changes, and 1,024 evidence findings. Files are relative, bounded, no-follow regular files. Only repository-local `#` references are implemented in v1; remote and external file references fail closed. Protected registered operations require effective OpenAPI security. Secret-shaped examples, credentials, production-looking servers in non-public documents, duplicate operation IDs, missing response classes, and registry/document identity drift are violations.

## Non-Authorization

No operation authenticates a caller, grants permission, writes a file, ratifies ownership, creates an endpoint, publishes documentation, invokes Mintlify, generates an SDK or runtime binding, exposes MCP, starts a listener, or docks with Maestro.
