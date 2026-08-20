# SAV v1 Schema Manifest

This directory owns the machine-readable v1 contracts for Accord References, typed source projections, CURRENT resolution and snapshots, evaluation, graph projection, and Named Versions.

| Schema | Purpose |
|---|---|
| `accord-reference.schema.json` | Immutable expected relationship declaration |
| `source-projection.schema.json` | Typed owner evidence supplied to SAV |
| `current-resolution-input.schema.json` | Bounded CURRENT construction input |
| `current-snapshot.schema.json` | Immutable derived CURRENT result |
| `evaluation-input.schema.json` | Exact snapshot and reference evaluation input |
| `evaluation-result.schema.json` | Three-axis accord evidence |
| `graph-projection.schema.json` | Disposable deterministic composition graph |
| `named-version.schema.json` | Immutable composition envelope |

These schemas do not transfer source ownership, authorize apply, create an installed snapshot, seal a Named Version, or make a projection canonical.
