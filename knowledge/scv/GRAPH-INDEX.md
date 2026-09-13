# SCV Graph Index Connector Contract

Contract v1, September 13, 2026. The supplied implementation is the independently installed C++26 `scv-graph-duckdb-connector` integration adapter, version `0.1.0-dev`. Duncan selected DuckDB as Symphony's default SQL database installation. This connector provides a relational index of existing SCV graphs; that choice does not adopt a universal graph database or constrain another caller-selected backend. Existing SCV `0.10.0-dev` engine packages and schemas remain unchanged.

## Ownership and identities

SCV owns the meaning of sources, captures, claims, supports, dependencies, qualifications and composition findings. The connector owns a bounded mechanical mapping of a complete, immutable native graph into three row collections. qxctl is the human and agent operating interface and independently checks the mapping before presenting results. It invokes the exact selected SCV owner for semantic validation. A database row match is not a new inference, source attestation, freshness renewal, runtime certificate or permission decision.

Keep these identities separate:

| Identity | Meaning |
| --- | --- |
| `graph.digest` | Existing sealed native `symphony.scv.graph.v1` value, including its evidence and policy. |
| `owner` | Exact SCV installation selected to validate the graph; not proof that this installation originally produced it. |
| `connector` | Exact receipt-owned adapter installation that performs the projection. |
| `snapshot.digest` | Graph, both installations, backend, mapping version, TOPS ID and namespace together. |
| `intent.digest` | One operation ID, exact snapshot and explicit validation query time. |
| `projection_digest` | Canonical complete expected claim/node/edge row inventory. |
| Database bytes | Backend-specific physical representation, including recovery state; not the native graph or snapshot identity. |

Installations retain all ten existing case-sensitive Go fields: `Role`, `ModuleID`, `EngineID`, `Version`, `Prefix`, `ReceiptPath`, `ReceiptDigest`, `ReceiptProtocol`, `ExecutablePath`, `ExecutableDigest`. Receipt protocol is `symphony.knowledge.install-receipt.v2`. Exact paths, receipt bytes, executable bytes, module/engine/role/version and vector must match at the operating boundary. There is no fallback owner, implicit upgrade or relabeling of a child domain. Graph domain equals the selected validating owner's role.

The local index root, TOPS identity and namespace are caller choices. The backend defaults to `duckdb` but is explicit in every snapshot and native data result. Namespace is ASCII `[A-Za-z0-9][A-Za-z0-9._:-]{0,127}`; the opaque operation ID uses the existing bounded ASCII token contract, at most 128 bytes. TOPS ID is a canonical UUID admitted by the existing STAV TOPS validator. These identifiers are data, never path components or SQL source.

## Exact artifacts and mapping

The module's receipt-owned `graph-index.schema.json` supplies closed named definitions. Its embedded SCV graph and query definitions preserve the selected existing shapes; they do not establish a new semantic owner. Schema validation alone cannot prove canonical digests, path correspondence, exact byte limits, native graph meaning or stored inventory integrity.

```text
Snapshot = {protocol:"symphony.scv.graph-index-snapshot.v1", backend:"duckdb",
  mapping_version:"1", tops_id, namespace, graph, owner, connector, digest}
Intent = {protocol:"symphony.scv.graph-index-intent.v1", operation_id,
  snapshot, validation_query_time, digest}
Projection = {claims:[{key,value}], nodes:[{key,value}], edges:[{key,value}]}
```

Seals follow existing canonical JSON and tagged SHA256 over the entire object except its `digest` member. A projection digest hashes the complete projection object, which has no seal member. Claims use their `claim_id`, nodes their `node_id`, and edges the tagged SHA256 of their complete canonical JSON as row keys. Preserve each complete native row value, including decimal strings, evidence and dependencies. Reject duplicate keys or malformed collection types. Sort each collection by unsigned UTF-8 byte order of keys. Every database key includes its TOPS/namespace/snapshot scope. No row is a globally unique claim simply because it has a familiar local ID.

A snapshot preserves the full graph so a consumer can independently regenerate every expected row. Queries and export verify the entire stored inventory, not merely the requested page. Missing, extra or changed rows fail; they are never silently repaired or converted into partial success. Rebuilding requires the retained graph and the caller's selected owner, not trust in a private database copy.

## Six bounded native operations

All requests use `symphony.knowledge.engine-process.v1`. `inspect` accepts `{}` and returns the standard sealed `symphony.knowledge.engine-descriptor.v2`; it never opens an index. The descriptor has its standard fields, with no invented backend metadata extension. Receipt-owned dependency provenance and this contract describe the selected storage profile.

| Operation | Exact payload | Result |
| --- | --- | --- |
| `prepare` | `{tops_id,namespace,operation_id,graph,owner,connector,query_time}` | Status with durable `prepared` intent, or identical existing state. |
| `commit` | `{tops_id,namespace,operation_id,expected_intent_digest}` | Status after atomic complete publication, or identical committed state. |
| `status` | `{tops_id,namespace,operation_id}` | Durable intent and verified stored state; missing operation is an error. |
| `query` | `{tops_id,namespace,snapshot_digest,kind,filters,cursor,limit}` | Exact snapshot, complete projection metadata and bounded matching page. |
| `export` | `{tops_id,namespace,snapshot_digest}` | Exact snapshot and complete projection metadata. |

Status is sealed `{protocol:"symphony.scv.graph-index-status.v1",backend,intent,state,snapshot_digest,projection_digest,counts,index_verified,digest}`. `counts` is exactly `{claims,nodes,edges}`. Prepared counts and projection digest describe expected rows; `index_verified` is false. Committed status returns true only after verifying the snapshot and complete row inventory.

Query is sealed `{protocol:"symphony.scv.graph-index-query.v1",backend,input,snapshot,projection_digest,counts,rows,matched_count,next_cursor,digest}`. `input` preserves the exact native payload. Export is sealed `{protocol:"symphony.scv.graph-index-export.v1",backend,snapshot,projection_digest,counts,digest}`. Success never truncates either result.

## Query and time semantics

`kind` is `claims`, `nodes` or `edges`. Filters are optional, conjunctive exact equality: claims admit `claim_id`, `subject`, `predicate`, `scope`; nodes admit `node_id`, `kind`, `capture_digest`; edges admit `from`, `relation`, `to`. Scope compares the entire supplied scope object, not a subset. Non-scope filter values are nonempty strings. Unsupported filter names fail. Structural edge lookup does not evaluate support paths.

`limit` is 1–128. Cursor is null or `{query_digest,after_key}`. The query digest hashes exactly `{tops_id,namespace,snapshot_digest,kind,filters}`; limit is intentionally excluded. A supplied cursor must bind this query and its key must occur among all matching rows. Return rows strictly after that key in byte order. `matched_count` counts every match before pagination; emit a next cursor only when more matches remain. An absent cursor does not select a newest snapshot.

Native indexed retrieval has no evaluation time and makes no freshness claim. qxctl query/export separately require explicit `query_time`, inspect the retained exact owner and invoke its existing `graph_query` on the full graph. Complete graph semantics preserve qualifications, conflicts, dependency propagation and expiry even when only one indexed row is requested. The returned connector result and owner evaluation remain separate evidence. Different query times can change a finding without changing the graph, snapshot or projection digest.

## Durable publication and recovery

`prepare` first validates the mechanical snapshot and stores its exact intent in a durable transaction. No snapshot or index rows become visible through this preparation alone. Repeating the same operation and intent is idempotent; reusing an operation for a different graph, owner, connector, scope or query time fails. Different operations may reference the same snapshot.

`commit` requires the exact expected intent digest. In one DuckDB transaction it publishes the immutable snapshot and all projection rows, and marks that intent committed. An interrupted or failed transaction must not expose a partly published graph. A committed retry returns the same logical state. Native commit does not claim SSIAG authorization or semantic policy approval.

qxctl `import` validates the selected owner and graph at the supplied time, prepares, independently verifies the returned prepared evidence, then commits that exact intent. `recover` obtains status, inspects the retained owner and replays the full graph at the retained validation query time, then commits or confirms the exact completed intent. Losing an original validating owner blocks semantic recovery/query/export; connector status can still report retained durable evidence without owner execution. Recovery does not reinterpret historical validation as evidence fresh at the present time.

qxctl publishes a sealed `symphony.qxctl.scv-graph-index-result.v1` object with exact fields `{protocol,operation,connector_result,owner_evaluation,digest}`. Operation is the user route name `inspect|import|status|recover|query|export`; owner evaluation is null for inspect/status and the separately validated native `graph_query` result for other successful routes. The complete envelope remains subject to the existing response bound.

Import input is `{operation_id,graph,query_time}`; query input is `{snapshot_digest,kind,filters,cursor,limit,query_time}`; export input is `{snapshot_digest,query_time}`. Data routes require `--index-root`, `--tops-id`, `--namespace`, `--connector-prefix` and `--connector-version`; import also selects `--domain`, `--prefix` and `--version` for its validating SCV owner. `--backend` defaults to duckdb. Status/recover use `--operation-id`; query/export derive the validating owner from the exact stored snapshot. Inspect requires the selected connector installation only.

No source head, corpus selection, protected graph head or canonical knowledge is changed. The existing selected-graph SSIAG/CAS circuit remains separate. There is no claim of an atomic transaction spanning this database and another authority store.

## DuckDB profile and limits

The exact package pins DuckDB 1.5.5 and its provenance. The adapter is C++26 and calls the stable C API of the actual C++ DuckDB engine, avoiding an assumption of compatibility with DuckDB's explicitly unstable C++ client API. This is a language-binding choice within the selected C++ implementation. [DuckDB C API](https://duckdb.org/docs/current/clients/c/overview), [C++ API stability](https://duckdb.org/docs/current/clients/cpp).

Each invocation uses one existing owned private `0700` index root and fixed `index.duckdb` plus its connector lock and backend recovery files. Reject symlinks, unrelated schemas and unsafe ownership. Inspect touches no database; reads do not create one. A bounded exclusive interprocess connector lock serializes this adapter's invocations. Unrelated writers are outside this profile. DuckDB's native in-process write mode is a single-process ownership model; this connector is not a shared database server. [DuckDB concurrency](https://duckdb.org/docs/current/connect/concurrency).

DuckDB transaction/WAL/checkpoint recovery can perform physical writes during an otherwise logically read-only status/query/export. Such writes cannot publish new logical snapshots. Transactions provide the backend commit/rollback boundary; actual interruption evidence must be reported separately from graceful close/reopen evidence. Filesystem/hardware guarantees are not inferred from a successful API call. [DuckDB transactions](https://duckdb.org/docs/current/sql/statements/transactions).

The selected implementation requests one DuckDB thread and a 256 MB DuckDB memory budget, disables temporary-directory spill, external access, automatic extension installation/loading, community/unsigned extensions, persistent secrets and allocator background threads, then locks configuration. Opening storage rejects a database or WAL above 512 MiB; these finite local limits are not a database capacity or performance guarantee. The configured DuckDB budget is not a verified hard ceiling on total process RSS. A request-deadline watcher interrupts database work.

Use fixed SQL and bound parameters; no arbitrary SQL, database URL, remote source, extension autoload, replacement scan or user code is admitted. Resource and cancellation limits belong to the exact installed profile. Shared framing stays 1 MiB request, 4 MiB response, existing depth/value/integer bounds. Graph maxima are 16 captures, 128 claims, 512 nodes and 1,024 edges; the database admits at most 128 intents and 128 snapshots globally. A large full snapshot plus semantic evaluation can exceed a response bound; reject it rather than silently truncate or weaken old limits. Backend migration, automatic deletion, general graph traversal and a performance claim are outside this increment.

Official links were inspected September 13, 2026; live documentation can change. Exact selected release provenance and runtime evidence take precedence over assumptions based on an unpinned current page.

## Transfer and acceptance

Reusable seams include exact installation identity, content-addressed snapshots, deterministic projections, private storage lifecycle, independent consumers and qxctl recovery. SHV can evaluate these mechanisms when its own domain contracts are ready; no hardware ontology, capability meaning or ownership is imported from SCV by this contract.

Focused acceptance covers new native producer/storage operations, independent Go projection and result verification, exact owner replay, namespaces, filters/cursors, corruption, prepare/reopen/recover, idempotence/conflicts, transaction interruption, missing owners, unchanged graph with expired evidence, and boundary rejection. Named checks are coverage definitions; actual pass/fail results and measured performance belong to the change closure. Installing a connector is not evidence that every optional backend or a generalized graph engine exists.
