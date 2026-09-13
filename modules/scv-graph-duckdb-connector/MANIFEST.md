# SCV DuckDB Graph Connector Manifest

## Identity

- module: `scv-graph-duckdb-connector`
- engine: `symphony-scv-graph-duckdb-connector`
- vector: `scv`
- version: `0.2.0-dev`
- language: C++26
- thermal path: freezing
- semantic owner: `knowledge/scv/GRAPH-INDEX.md`

## Canonical Surfaces

- `modules/scv-graph-duckdb-connector/INTENT.md`
- `modules/scv-graph-duckdb-connector/MANIFEST.md`
- `modules/scv-graph-duckdb-connector/SKILL.md`
- `modules/scv-graph-duckdb-connector/SPEC.md`
- `modules/scv-graph-duckdb-connector/INSTALL.md`
- `modules/scv-graph-duckdb-connector/FEATURES.md`
- `modules/scv-graph-duckdb-connector/schemas/v1/graph-index.schema.json`
- `modules/scv-graph-duckdb-connector/schemas/v2/graph-index.schema.json`
- `knowledge/scv/GRAPH-INDEX.md`
- `knowledge/scv/INDEX-MAINTENANCE.md`

## Package and State Boundary

The immutable receipt identifies the exact connector executable, owned contracts, schema and supporting files. The caller's database, DuckDB journal files and import state are outside those package files. Removing the installed connector must not delete a caller's database or change another installed version. There is no active alias, resident service, network listener or mandatory backend dependency for the existing SCV engines.

The connector release is independent of the existing SCV domain-engine releases. It neither changes their graph v1 semantics nor substitutes its installation identity for the selected validating graph owner. Logical graph identity, validating-owner evidence, connector identity, physical format and caller namespace remain distinct.

Database projections are disposable when their declared inputs remain available. A database copy alone does not prove that the original source process or selected semantic owner can be replayed. Installation or index publication grants no authority over a protected source/graph selection, provider account, Node or canonical repository.
