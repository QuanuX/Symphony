# SHV durable graph store v1

The independently installed C++26 `shv-graph-duckdb-connector` 0.1.0-dev persists the existing `symphony.graph.exchange.v1` value losslessly using exact DuckDB 1.5.5. It owns structural persistence only. SHV kernel/source/PDF owners retain meaning and replay. It accepts structurally conforming caller-defined owner artifacts; it does not grant them SHV authority. DuckDB is the SQL default; no dedicated graph database is selected.

## Identity and storage

Snapshot = sealed `{protocol:symphony.shv.graph-store-snapshot.v1, backend:duckdb, mapping_version:1, tops_id, namespace, graph, connector}` (mapping_version is the string `1`). Connector is the exact receipt-v2 Installation with all ten case-sensitive fields. Intent = sealed `{protocol:symphony.shv.graph-store-intent.v1, operation_id, snapshot}`. Database bytes are not a snapshot identity. Graph properties and owner_artifact remain complete canonical JSON; no hardware field census is imposed.

Projection is `{nodes:[{key:id,value:complete-node}],edges:[{key:id,value:complete-edge}]}` in graph's required sorted order. Every row is scoped by TOPS, namespace and snapshot digest. Before committed status, export or query succeeds, all rows and indexed columns must match the complete retained graph. An absent row, extra row or changed field fails closed; no automatic repair. A retained source remains necessary for semantic owner replay.

## Operations and qxctl authority

One command family owns this surface: `qxctl shv graph store`. It does not duplicate in-memory `shv graph adapter` operations or semantic `shv graph validate`.

- `inspect`: descriptor, no database access.
- `prepare`: input `{tops_id,namespace,operation_id,graph,connector}`; durable immutable intent. qxctl supplies connector from exact installation; callers supply the other fields.
- `commit`: `{tops_id,namespace,operation_id,expected_intent_digest}`; atomic snapshot/rows/intent transition. Repeating the same commit is the recovery operation; no second recovery command.
- `status`: `{tops_id,namespace,operation_id}`; prepared or committed evidence. Operation ID reuse with different content is rejected.
- `export`: `{tops_id,namespace,snapshot_digest}`; complete stored graph inside its snapshot.
- `query`: `{tops_id,namespace,snapshot_digest,kind,filters,cursor,limit}`; kind nodes|edges, node filters id, edge filters id|from|to|label. Empty filters match all. Limit 1..128. Cursor `{query_digest,after_key}` binds snapshot/scope/kind/filters and an actual matching row; page size may change. Results disclose total matches and explicit continuation. No global hardware conclusion follows from an empty page.
- `schema` and `template`: exact receipt-owned discovery; templates remain unanswered examples.

All leaves select `--connector-prefix`, `--connector-version` and `--backend` (default duckdb). Data leaves require an existing private `--store-root`; all except inspect/schema/template require `--input`. Scope is in that input only, preventing competing flag/payload scope authorities. All support JSON. qxctl independently verifies seals, projection, result correspondence and exact selected installation. It does not claim semantic replay merely because a store result passes.

## Durability profile

Adapted private storage and transaction mechanics from SCV, with a separate database identity and row mapping. No SCV database is accepted or migrated. Existing owner-only 0700 root, private 0600 regular files, no symlinks, bounded interprocess lock, one DuckDB writer invocation at a time. Reads do not create a missing database; physical WAL recovery may write during logical reads. Fixed SQL and bound parameters only. External access, extension loading/install, spill and persistent secrets are disabled. Exact linked version is checked. One DuckDB thread, 256MB configured memory budget (not a hard process RSS guarantee), 512MiB database/WAL input bounds, 128 intents and snapshots, existing process request 1MiB/response 4MiB and deadlines. Generic graph limits remain 1024 nodes/2048 edges. Caller owns all storage and coverage choices. This pinned package recipe targets macOS x86_64; expanding platform recipes requires exact dependency evidence.

Prepare persists before commit. A killed transaction cannot expose partial rows; committed retry verifies complete inventory. Tests distinguish deterministic SIGKILL recovery from graceful close/reopen. No cross-store atomicity or hardware durability guarantee is implied.

## Publication boundary

Storage commit records immutable evidence. It does not select a canonical catalogue, mutate SSIAG policy or update a protected head. The protected authority transaction is implemented separately under `knowledge/shv/PUBLICATION.md`; `knowledge/shv/CATALOGUE-PUBLICATION-PLAN.md` retains the earlier plan. Preserve separate identities for manifest, materialized partitions, graph snapshot and selected catalogue head. Never use a newest SQL row as a canonical head.

## Inventory release 0.2.0-dev (SHV-19)

`qxctl shv graph store inventory` is a logical read operation under the existing
storage owner, using explicit connector prefix/version, store root and JSON input.
It takes tops_id, namespace, expected_revision (null or exact manifest digest),
cursor (null or revision/after_operation_id), and limit 1..16. The complete scoped
manifest contains sorted operation summaries and snapshot reference counts;
records contains the selected page of full verified status objects. Prepared
intents remain distinct from committed snapshots. Shared snapshots retain every
operation reference. Empty scope means no retained operations, not no hardware.

The owner verifies every retained intent, committed snapshot and projection,
including other scopes, and rejects orphan snapshots and extra projection rows.
Global counts and an opaque global_revision expose capacity changes without
returning other scopes' records. A change anywhere in the store invalidates the
manifest revision and its cursors. No implicit rebase or newest-row selection.
Physical database/WAL sizes are observations, not content identity or usable-space
promises. Existing 128-intent/128-snapshot bounds are unchanged.

The independent Go verifier checks exact input, seals, ordering, reference
accounting, bounds, selected records and continuation. Global database completeness
and off-page evidence are checked by the native owner, not inferred from a digest
alone by qxctl. Unknown graph properties and retired fields remain intact.

Inventory 0.2 can inspect retained 0.1 and 0.2 writer records in the unchanged v1
database schema. Historical receipt identities are preserved; a reader does not
replace a writer. Other data commands still require the exact selected writer;
0.2 refuses committing a 0.1 intent before changing state. Keep the original 0.1
installation for its writes, exports and SHV-18 publication bindings. Publication
engine 0.1 does not silently admit a 0.2 writer. Existing installed packages remain
unchanged. This release adds no transfer execution, deletion, pruning, retention
policy, catalogue selection or database migration. These require later contracts.
