# SCV DuckDB Graph Connector Specification

The semantic owner companion is `knowledge/scv/GRAPH-INDEX.md`, installed beside this file as `GRAPH-INDEX.md`. This module owns its mechanical projection and persistence contract. SCV retains graph/evidence meaning, and qxctl retains operating orchestration and independent result validation. The separately installed adapter does not change an existing SCV engine release.

## Identity and package

- Module: `scv-graph-duckdb-connector`.
- Engine: `symphony-scv-graph-duckdb-connector`.
- Vector: `scv`; version: `0.1.0-dev`; language: C++26; thermal path: freezing.
- Receipt kind: `adapter`; receipt protocol: `symphony.knowledge.install-receipt.v2`.
- Process: `symphony.knowledge.engine-process.v1`; descriptor: `symphony.knowledge.engine-descriptor.v2`.
- Feature: `ssfv:symphony:scv-graph-duckdb-connector`.
- Dependency: actual C++ DuckDB engine 1.5.5 through its stable C ABI, with receipt-owned library, license and `DUCKDB-PROVENANCE.json`.

The current adapter supports DuckDB only. Duncan's default SQL selection is not a restriction on future independently published adapters, a mandatory runtime dependency of every user composition, or adoption of a universal graph database.

## Published operations

| Native operation | Administration | Input schema definition | Output schema definition |
| --- | --- | --- | --- |
| `inspect` | Inspect exact installation without opening a database. | `InspectInput` | `Descriptor` |
| `prepare` | Durably retain an exact import intent; publish no snapshot. | `PrepareInput` | `StatusResult` |
| `commit` | Publish all rows and snapshot atomically for the expected intent. | `CommitInput` | `StatusResult` |
| `status` | Inspect retained intent and verify committed projection. | `StatusInput` | `StatusResult` |
| `query` | Retrieve one exact immutable snapshot's bounded matching page. | `QueryInput` | `QueryResult` |
| `export` | Return a complete verified snapshot. | `ExportInput` | `ExportResult` |

Definitions are in the single self-contained receipt-owned `schemas/v1/graph-index.schema.json`, flattened to `graph-index.schema.json` at installation. Exact installation/path correspondence, seals, byte limits, cross-field identities, full projection regeneration, query ordering and semantic replay require runtime validation in addition to structural schema checks. No arbitrary SQL is an operation.

The standard descriptor has exactly its v2 fields and six operation records `engop:symphony:scv.graph-index.<operation>`. Backend capabilities are expressed by this versioned contract and dependency provenance; no unregistered descriptor field is added.

## Mapping and durable state

The snapshot preserves the entire existing sealed graph, exact validating SCV owner, exact connector installation, caller TOPS ID and namespace, backend and mapping version. Snapshot digest, native graph digest, projected row inventory digest and physical database bytes have different meanings. Row keys derive from claim IDs, structural node IDs and exact edge-content hashes. Full native JSON values remain intact. Scope, source generation, timestamps, support sets and numeric representation are never reconstructed from lossy SQL columns.

Prepare and commit are distinct durable transactions. An exact operation retry is idempotent; conflicting reuse fails. Commit publishes snapshot, complete rows and committed intent state in one transaction. A committed read verifies the retained graph seal and every expected row, rejecting corruption without repair. Namespace and snapshot scope bind all keys and cursors. No selected graph head, canonical file or SSIAG decision is mutated.

The native connector performs mechanical checks. qxctl import/recover/query/export inspect and invoke the exact retained SCV owner against the full graph; a validating owner is not asserted to be the original producer. Missing owner blocks semantic routes. qxctl status can inspect durable state without execution of that owner. Structural index results and time-qualified native owner evaluation are separate outputs.

## Storage and bounded process behavior

The process cwd is an existing private owned root; storage names are fixed. The connector serializes invocations with a bounded interprocess lock and rejects unsafe paths, symlinks, ownership and unrelated database schema. Reads do not create an absent database. Backend WAL/checkpoint recovery may write physical state during logical reads; logical publication still requires commit. There is no promise of concurrent unrelated DuckDB writers or a general database server.

All six operations retain shared 1 MiB request, 4 MiB response and canonical JSON depth/value/integer bounds. Graph and global inventory maxima are those in the owner companion. Response overflow is failure, not truncation. Fixed parameterized SQL and disabled external/extension behavior keep the published interface finite. Resource settings and linked dependency are explicit in source and installed provenance. No SQLite transaction settings apply.

## Verification and exclusions

Focused new native process, Go consumer and qxctl tests cover exact row projection/filtering, sealed identity, prepared/committed recovery, corruption, namespace isolation, unsafe roots, missing or changed owners, and original SCV semantics at an explicit time. Test execution is recorded separately from this specification. Crash recovery, graceful restart and full power-loss guarantees must not be conflated. The separate noninstalled `scv-graph-duckdb-commit-fault-test` target compiles this connector implementation with deterministic prepare/commit barriers; `tests/interrupted_commit.py` kills that writer and recovers through the exact installed production connector and qxctl. Fault controls are absent from the installed executable. See `tests/README.md` for stage coverage and limits; actual results belong to change evidence.

This release excludes arbitrary query languages, automatic backend migration, deletion/garbage collection, background ingestion, generalized graph algorithms, provider authority selection and SHV runtime. A future connector may implement compatible logical operations under its own exact package contract; this package does not silently accept it.
