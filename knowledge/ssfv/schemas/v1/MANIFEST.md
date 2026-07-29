# SSFV v1 Schema Manifest

This directory owns the machine-readable JSON Schema contracts for the future Symphony Semantic Feature Vector engine.

| Schema | Purpose |
|---|---|
| `feature-record.schema.json` | Normalized canonical feature record |
| `namespace-entry.schema.json` | Normalized stable-ID namespace allocation |
| `registry-entry.schema.json` | Normalized canonical feature-routing entry |
| `check-result.schema.json` | Read-only structural and freshness result |
| `diff-input.schema.json` | Caller-declared bounded comparison input |
| `diff-result.schema.json` | Deterministic feature-change evidence |
| `proposal-input.schema.json` | Caller-declared bounded multi-file proposal input |
| `graph-projection.schema.json` | Disposable portable feature-graph projection |

These schemas govern future JSON inputs, results, and normalized records. They do not implement the engine, create a feature record, authorize canonical apply, or make a generated graph authoritative.
