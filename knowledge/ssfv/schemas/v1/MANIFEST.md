# SSFV v1 Schema Manifest

## Canonical Surfaces

- `knowledge/ssfv/schemas/v1/MANIFEST.md`
- `knowledge/ssfv/schemas/v1/check-result.schema.json`
- `knowledge/ssfv/schemas/v1/diff-input.schema.json`
- `knowledge/ssfv/schemas/v1/diff-result.schema.json`
- `knowledge/ssfv/schemas/v1/feature-file.schema.json`
- `knowledge/ssfv/schemas/v1/feature-record.schema.json`
- `knowledge/ssfv/schemas/v1/graph-input.schema.json`
- `knowledge/ssfv/schemas/v1/graph-projection.schema.json`
- `knowledge/ssfv/schemas/v1/namespace-entry.schema.json`
- `knowledge/ssfv/schemas/v1/proposal-input.schema.json`
- `knowledge/ssfv/schemas/v1/registry-entry.schema.json`
- `knowledge/ssfv/schemas/v1/semantic-snapshot.schema.json`

This directory owns the first machine-readable JSON Schema contracts for the Symphony Semantic Feature Vector engine.

| Schema | Purpose |
|---|---|
| `feature-record.schema.json` | Normalized canonical feature record |
| `feature-file.schema.json` | Embedded canonical `FEATURES.md` envelope |
| `namespace-entry.schema.json` | Normalized stable-ID namespace allocation |
| `registry-entry.schema.json` | Normalized canonical feature-routing entry |
| `check-result.schema.json` | Read-only structural and freshness result |
| `diff-input.schema.json` | Caller-declared bounded comparison input |
| `diff-result.schema.json` | Deterministic feature-change evidence |
| `proposal-input.schema.json` | Caller-declared bounded multi-file proposal input |
| `semantic-snapshot.schema.json` | Content-addressed feature and implementation evidence |
| `graph-input.schema.json` | Exact portable graph request |
| `graph-projection.schema.json` | Disposable portable feature-graph projection |

These schemas govern initial JSON inputs, results, containers, snapshots, and normalized records. Executable v2 operation contracts live in `knowledge/ssfv/schemas/v2/`. They do not create a feature record, authorize canonical apply, or make a generated graph authoritative.
