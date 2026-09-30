# SNV Spec

## Admitted release

`symphony-snv` / `snv-engine` / `0.1.0-dev` is a C++26 composition engine and static SDK. It statically embeds SNIV, SNRV, SCIV and SCNV at exactly `0.1.0-dev`, using SnvCommon `0.1.0` and KnowledgeVectorEngine `0.2.0`. Installed child executables are independent entry points; parent replay uses its compiled reducers. Installation receipts bind owned bytes, not publisher authenticity or physical truth.

`OWNER-INTERFACE.json` and the exact `schemas/v1/` resources declare all v1 operations. Generated native/Go/CMake projections must agree with that declaration. Unsupported owner, schema or writer versions reject. No newest-version selection or cross-version semantic substitution occurs.

## Source evidence and composition

A bundle contains its caller-selected ID, at most one artifact per child owner and explicit typed relations. Every artifact carries its exact owner version, operation, original UTF-8 input and SHA-256 of those original bytes. Replay parses those bytes with the strict parser and invokes the exact child reducer. The child's canonical input digest and artifact byte digest have different meanings. Caller-supplied result assertions and extra fields reject.

References distinguish physical Node, incarnation, local resource, remote attachment, cluster, system, bus, membership, connection and name-association identity. Illegal owner/kind combinations reject. Missing companions or subjects remain explicit findings and partial coverage. A relationship name resolves only against an independently supplied bundle relation or SCIV membership/connection; a naming record cannot establish its subject.

SNRV identity rebinds resolve to the exact SNIV predecessor/successor record. A current rebind names the classified proposed SNIV record, rather than any historical record with similar subjects. Hardware-change classification binds exact before/after inventory references and the caller's versioned materiality claims. Explicit whole-Node replacement, established provider-resource replacement and retirement use their own categorical SNIV boundaries. A retained rebind does not demand a new transition for later availability-only changes. Mismatched subjects or evidence references reject; unavailable classification remains insufficient. Successful verification does not alone make an otherwise complete bundle partial.

Typed references include retained owner records and accepted proposals. A supplied but suppressed identity proposal cannot establish a referenced subject. `partial` describes missing owner/reference coverage; the unchanged child views separately retain their semantic conflict, insufficient evidence and connectivity outcomes. Selecting such recorded evidence does not establish physical truth, active participation or a successful deployment. No universal cross-owner as-of clock is inferred from absent or differently interpreted timestamps.

Original attributed records preserve each owner's declared temporal and causal rules. Parent history exposes those records and proposed records with owner/kind/source binding. It does not reinterpret their effective times or create a universal clock order. Diff compares exact owner semantic projections against an explicit baseline. Neither projection repairs evidence, establishes deployment readiness, assigns names, provisions resources or joins a bus.

## Native operations

| Operation | Effect |
| --- | --- |
| `snv_inspect` | Pure replay, projection, diff, bounded history, export manifest or export chunk. |
| `snv_evidence_plan` | Pure validation of a supplied bundle or complete portable export; sealed retention proposal. |
| `snv_state_plan` | Pure proposal for a caller-named view, exact prior head and retained candidate or explicit unselect. |
| `snv_state_reduce` | Independently regenerates and compares the complete plan/input, then returns a sealed `proposed_only` transition. |

Native plans require the complete input, expected state, operation ID and exact causal head. A head binds TOPS/view, generation, previous digest, bundle digest, tombstone, writer and operation. Explicit unselection produces a new tombstone preserving lineage. It does not delete retained data or retire a name. A digest binds representation and correspondence; it grants no authority.

## qxctl retention and selection

The Go adapter owns private filesystem transactions, with domain interpretation delegated to C++. Evidence `prepare` durably retains an exact private intent; `commit` retains the independently replayed bundle, and its sealed result acknowledges that local outcome. No selected-view effect occurs. An exact retry reconciles the same intent; reuse of an operation ID with changed input, plan or installation rejects.

The evidence journal and TOPS/view journals are separate namespaces beneath an explicitly selected private state root. They use owned no-follow directories, regular single-link files, serialization, bounded reads, same-directory atomic replacement and file/directory fsync. A successful result requires the local durability operations to succeed. This profile claims tested process-interruption recovery, not hardware power-loss certification. Status and selected inspection observe without creating, repairing or selecting journal state. No separate mutable index is authoritative; a missing or corrupt required journal fails explicitly.

`state plan` proves candidate retention and captures the exact current head without writing it. Its public capsule contains the native input/plan and exact installation binding. `apply` accepts that capsule or an exact retained operation ID. Before authorization it retains the complete input, plan, transition, installation and durable correlation. Recovery reuses that lineage, replays it under the exact installation and obtains fresh authority. Competing/stale plans fail compare-and-swap. Retrying a committed older operation reports its recorded disposition without rewinding a later head.

Selection and unselection require SSIAG action `symphony.snv.view.select` on the SHA-256 resource derived from `{tops_id, view_id, owner_engine_id: "symphony-snv"}`. qxctl rechecks exact installed bytes, retained candidate, replay correspondence, capability expiry and current policy/config/subject/grant immediately before rename. This is a check-to-effect boundary with a remaining race; it is not an atomic authorization-and-filesystem transaction.

SSIAG's actual policy decision is audited through STAV. The local outcome journal remains distinct. qxctl reports `authorization_audit: "ssiag_policy_decision_only"` and a null `head_write_stav_receipt`; it never fabricates a STAV receipt for the head write. Audit unavailability prevents a new protected selection. Stored historic decisions may be expired and still readable; they do not authorize recovery.

## Portable representation and removal

Export is observational. The sealed manifest binds writer/version, embedded reducers, canonical bundle digest, length and ordered raw-byte chunk digests. Hex chunks preserve UTF-8 boundaries and exact original artifact strings. Import requires every ordered chunk with matching manifest/revision/length/digest; missing, duplicate, reordered, tampered or mixed-revision data rejects. A summary is not an export. Commands do not secretly write export files; callers save returned JSON.

Only the admitted initial writer is readable in this release. Explicit migration metadata binds source and target but cannot make an incompatible writer admissible. Future migrations require a separately admitted reader/reducer. Package removal preserves private evidence and view journals. Data erasure, external deletion and automatic garbage collection are unsupported.

## Independent bounds

- Process request 1 MiB, response 4 MiB, depth 64, JSON values 262144.
- Canonical bundle 256 KiB, each original owner artifact 64 KiB, at most four artifacts and 2048 explicit relations.
- Direct child record profiles select 512, 1024 or 2048; these do not enlarge process or composition byte bounds.
- History page limit 2048 with snapshot-bound offset; export chunks 16384 raw bytes.
- Private journal 16 MiB and 512 operations/bundles. Capacity refusal preserves the prior valid journal.

Exact external identifiers and quantities follow their owner schemas, including decimal strings where required. No float coercion, hidden truncation or Unicode normalization is introduced by composition. Protocol errors and deadlines yield bounded structured failures. A timed-out mutation may have committed; status/recovery reconciles the retained operation rather than asserting no effect.
