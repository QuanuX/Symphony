# SCNV engine v1 specification

Version `0.1.0-dev`, C++26, public API `symphony::snv::scnv::operations()` and `handle(operation, payload, deadline_unix_ms)`. Pure reducers do not retain evidence or select a view. The independently installed bounded executable uses `symphony.knowledge.engine-process.v1` and advertises its exact descriptor.

## Records and comparison

Every association supplies `record_id`, stable `association_id`, `kind`, nullable `previous_record_id`, nullable `origin_record_id`, `subject`, `name`, `name_kind`, `namespace`, `source`, `scope`, `interval`, `observed_unix_ms` and `recorded_unix_ms`. Unknown fields and unknown protocols are refused. IDs are opaque source-qualified locators; string equality does not establish physical identity or operating authority.

Subjects are Nodes, incarnations, separately named resources, clusters or their relationships. Each subject carries an explicit kind and ID. Assigning sources carry kind and ID. Name kind preserves provider-resource identifiers, provider-offering names, infrastructure names, Node identities, cluster names, user nicknames, short names, resource/incarnation/relationship names and other external names. These are distinct associations even when they resolve to the same subject.

Scope is an explicit `(kind,id)` pair, with kind `tops`, `trog` or `cluster`. A cluster scope requires the exact identified cluster ID; the resolver never discovers a cluster from a nickname or parses slash/`::` shorthand. No scope inheritance or global namespace is inferred.

Comparison is exact valid UTF-8 bytes. Case, Unicode normalization forms and punctuation are preserved; no folding, normalization, fuzzy matching or name generation occurs. C0/C1/DEL control characters, invalid UTF-8 and empty strings are refused. Names allow at most 512 UTF-8 bytes; IDs/namespaces allow 256, reasons 1024. Schema string limits express a conservative character boundary; native byte validation is authoritative.

An interval is `[from_unix_ms,until_unix_ms)`, with explicit nullable unbounded end. Both effective times and observed/recorded timestamps are nonnegative exact integers through `9007199254740991`. Distinct effective, observed and recorded time fields are preserved; no wall-clock newest-writer rule chooses between sources.

## Immutable revisions and transitions

An `assignment` starts a supplied association with both lineage pointers null. `correction` and `retirement` reference the exact predecessor through `previous_record_id`, retaining the same association ID. Each supplied association has one complete linear revision chain; missing predecessors, cycles, forks, duplicate IDs and competing roots refuse. The terminal revision determines the association's current interpretation across its effective interval. Earlier raw records remain part of the snapshot. A correction can explicitly rectify supplied association facts; retirement keeps subject/name/source/namespace/scope/start fixed and only bounds its prior interval.

`reuse` and `restoration` start a new association and explicitly reference a terminal retired origin through `origin_record_id`. Their interval starts at or after the origin's finite end, their exact name/scope are preserved, and restoration also preserves subject. Reuse permits a separately supplied subject. A restored association has its own later interval, preserving the gap after retirement. These records preserve historical evidence without extending an old interval across a period of absence.

`names_validate`, protocol `scnv.names.v1`, has `mode:records` or `mode:transition`. Both carry `records`, `coverage:complete|partial` and `limits:{max_records:512|1024|2048}`. Transition adds exact naming `policy` with ID/version and supported constants `utf8_bytes`, `exact_scope`, `report`, `half_open`, and `intent:{action,expected_evidence_digest,new_record,reason}`. Actions are `assign`, `correct`, `retire`, `reuse`, `restore`. The engine validates the predecessor snapshot and supplied new record, reports resulting collisions, and returns only the proposed record plus exact predecessor, intent and policy digests. No IDs or names are fabricated. A capacity or stale-evidence conflict refuses rather than truncating or overwriting.

Collision findings summarize simultaneous different-subject claims to the same unqualified name in the same exact scope. Namespace/kind qualification can still make a specific query unique, but unqualified overlap remains visible. Multiple matching associations for the same typed subject do not create ambiguity. Temporal overlap is tested on terminal revisions; adjoining half-open intervals do not collide. Findings are bounded by grouping, and no first record wins.

## Resolution and bounded candidates

`names_resolve`, protocol `scnv.resolve.v1`, consumes explicit `records`, `coverage`, `limits` and `query`. Query supplies `mode:exact|scoped_candidates`, nullable `name`, exact `scope`, nullable `namespace` and `name_kind`, explicit `as_of_unix_ms`, `offset`, `limit` and nullable `expected_evidence_digest`.

Exact mode requires a name and zero offset. It returns `unique`, `absent`, `ambiguous`, `insufficient` or `unsupported_scope`; it never chooses a first match. More than one typed subject is ambiguous. Partial coverage makes zero/one-subject resolution insufficient. Complete means complete within the caller's declared supplied snapshot, never exhaustive external knowledge. Query scope kinds outside TOPS/TROG/cluster return `unsupported_scope`.

Scoped candidates enumerate exact supplied associations within the scope/time, optionally filtered by exact name, namespace and kind. Pages contain 1–128 records, sorted by record ID, with explicit total and next offset. Every continuation requires the original `evidence_digest`; changed evidence rejects. No candidate is silently dropped or fuzzy matched.

Unique resolution includes immutable subject, record IDs, scope, time, owner version and evidence digest in a sealed `resolution_binding`. A later alias reassignment produces a different binding and cannot alter a previously prepared exact subject. An operating consumer must independently revalidate its own target and authority requirements.

## Results, reproducibility and limits

All results carry `owner`, `owner_version`, `source_digest` and `subject_ids`. `source_digest` is SHA-256 of the exact canonical JSON whole input (`nlohmann::json::dump`: UTF-8 preserved, sorted object keys, compact encoding, array order preserved). `evidence_digest` hashes `{protocol:scnv.evidence.v1,records,coverage,limits}` using the same representation. It binds pagination independently of query changes. Digests prove representation agreement, not the truth of a subject or a claim.

The initial record profiles follow 512/1024/2048, each explicitly selected. No profile automatically doubles capacity. Shared process byte/depth/value/response bounds and a request deadline remain independent. Pure SDK operations enforce cardinality, field/byte/numeric bounds and check their deadline during scans. Capacity failure does not return a partial successful result.

Canonical registration, feature administration and cross-owner SNV replay are integrated separately by their owners. SOV operations, name issuance, provider/hardware discovery, universal namespace allocation and secret delivery receive no authority from these results.
