# SCV Maintained Corpus Contract v1

## Scope and Version

The `0.2.0-dev` SCV, SCHV, SCEV and five provider engines add four pure corpus operations to the original thirteen operations. The source, capture, interpretation and graph v1 shapes remain unchanged. The common process envelope, exact receipt-v2 identity and independent installation boundaries remain in force. Corpus snapshots are private retained evidence, not official Symphony knowledge or a complete inventory of a provider.

This companion owns `capture-index.schema.json`, `corpus.schema.json` and the named definitions in `corpus-operation.schema.json` under `schemas/v1/`. Structural schemas do not replace UTF-8 byte limits, strict UTC time, digest and cross-field checks. An input metadata protocol selects its named request definition; it does not add a protocol field to that payload.

| Operation | Request definition | Result protocol | Interaction |
| --- | --- | --- | --- |
| `capture_index` | `captureIndexInput` | `symphony.scv.capture-index.v1` | invoke, evidence only |
| `corpus_build` | `corpusBuildInput` | `symphony.scv.corpus.v1` | invoke, evidence only |
| `corpus_query` | `corpusQueryInput` | `symphony.scv.corpus-query.v1` | query, read only |
| `corpus_diff` | `corpusDiffInput` | `symphony.scv.corpus-diff.v1` | validate, evidence only |

## Capture Index and Exact Bytes

`capture_index` validates a complete supplied Capture v1 object, including its source and byte/content seals. The resulting index records the invoked domain; provider, family, source and locator identities; exact source digest and generation; requested URI; capture and body digests; byte size; observation time; upstream revision; representation; disposition and issues. Here validating a complete object does not mean its acquisition disposition must be `complete`: partial and failed attempts are indexed honestly.

An index contains no body. Its seal proves internal identity, not correspondence with a missing capture. A consumer claiming reconstructability must load the referenced capture, validate it and regenerate the exact index for comparison. Byte storage may deduplicate identical artifacts; this does not merge independent sources or establish independent corroboration. Index scope may be subordinate to the consuming parent engine; provider/family specialization remains explicit. Corpus artifacts themselves identify exactly the invoked domain.

## Building Immutable Snapshots

`corpus_build` receives `{corpus_id,previous,snapshot_time,attempts}`. `previous` is null or the exact predecessor corpus. Each attempt supplies a `member_id` and a capture index. The request declares the full member selection, including an explicitly empty set; it is not an append-only delta or automatic discovery instruction. Corpus and member IDs are bounded opaque identifiers, not names inferred from titles or hashes.

At most 128 unique members are admitted, with one member per `{family_id,provider_id,source_id,locator_id}` tuple. A member retained from the immediate predecessor cannot be rebound to another tuple. Owner/source identity collisions fail. This checks the supplied predecessor boundary; it does not claim an unbounded ancestry lookup. Snapshot generation advances by one, starting at one with no parent. IDs and domains must match the supplied predecessor. Member sorting makes input enumeration order irrelevant. Snapshot time cannot precede the predecessor or its latest attempts.

Each output member contains `latest_attempt` and `last_complete`. A complete latest attempt becomes `last_complete`. A partial or failed attempt remains the latest attempt and may retain the same member's earlier complete capture separately. That retained capture keeps its original source revision, timestamp and digest. A user may deliberately select historical source revisions; corpus refresh does not mutate the source head or impose a monotonic source-generation selection rule.

An omitted predecessor member leaves this snapshot's selection. Omission, a timeout, a 404 or a changed feed window does not establish vendor retirement. The prior immutable snapshot remains evidence. Coverage counts requested members, complete/partial/failed latest attempts and members retaining prior complete evidence after a partial or failed latest attempt. Completing the finite request never implies complete vendor coverage.

## Explicit Query and Difference

`corpus_query` receives the exact corpus, query time, explicit member IDs, selection mode and optional-age value represented explicitly as null or an integer. Empty member IDs select all members in this corpus; unknown or duplicate IDs fail. Selection is exactly `latest_attempt` or `last_complete`. Results retain latest-attempt identity alongside the chosen index, its acquisition status, freshness, source-revision agreement and reasons. A missing selected capture is `unavailable` and `not_selected`. A future observation is not current; retained old complete evidence does not become fresh because refresh failed.

Query coverage describes latest attempts for the requested members. It does not replace the selected capture's separate status/freshness. The query is evidence selection, not admission of a claim into a graph. A later `knowledge_interpret` request still supplies its own explicit interpretation and evidence policy.

`corpus_diff` requires matching corpus identity/domain and compares two exact snapshots. It reports added and removed member selections, plus body, source, observation, coverage and retained-complete changes for common members. It does not retire a service, refresh a graph, infer compatibility or execute remediation.

## qxctl Retention and Recovery

`qxctl scv corpus acquire|import|recover|inspect|export|diff` administers immutable local evidence retention. New commands default exactly to `0.2.0-dev`; existing SCV commands retain the `0.1.0-dev` default. Explicit version selection uses that version's exact receipt, descriptor and operation set. The old descriptor remains thirteen operations; the new descriptor has seventeen. Unsupported combinations fail without selecting a newer package.

Acquisition/import jobs bind the exact operation, corpus identity, predecessor digest, member inputs and installation before processing. Acquire validates sources before public HTTPS retrieval. Import records supplied captures without inventing a live fetch. Completed members are checkpointed; recovery uses the original intent and installation and processes only unfinished members. Replaying a completed operation returns its recorded snapshot without new retrieval. A refresh is a new operation with an explicit predecessor digest. Independent concurrent jobs coexist; no last-writer-wins current alias exists.

The qxctl adapter rejects known future-dated imported captures or a future predecessor before recording intent, with an explicit clock/order error. It derives snapshot time from its actual clock when member work completes and checks that time against all observations and the predecessor before freezing the snapshot. A clock rollback during work leaves finalization unfinished and recoverable; it does not freeze an invalid timestamp or fabricate a later clock reading. This adapter check does not change pure C++ caller-supplied times or the query distinction between future and current evidence.

The store keeps exact captures, regenerated indexes and snapshots by content digest in a private no-follow directory. Duplicate writes verify existing bytes. Immutable publication uses an atomic no-replace operation after flushing file content; objects are never overwritten under an existing digest. A snapshot becomes addressable only after referenced objects are durable. Interrupted work may leave reusable unreferenced objects, but cannot expose a falsely complete snapshot. Job journals record work and recovery; they do not select a source, corpus or graph authority head.

The qxctl store admits captures through acquire/import and produces their indexes through the exact selected engine domain. It has no raw index or raw snapshot import surface. Parent-scope retention therefore explicitly imports subordinate captures and creates parent-domain indexes; it does not silently rewrite an external index or require a child installation. The pure C++ contract separately permits in-scope subordinate-produced indexes.

Deep store verification is bounded to 128 snapshots in the reconstruction chain, including the requested snapshot and 4,096 unique capture/index pairs. A successor reserves its additional ancestry level and verifies new evidence pairs before snapshot publication. These adapter bounds are independent of the pure 128-member snapshot bound. Exceeding them fails explicitly; no automatic pruning, ancestry omission, retention deletion or migration is implied. Older bytes remain necessary for any declared full reconstruction.

Inspect and diff require explicit snapshot identities and validate referenced objects. Export additionally selects member IDs, selection mode, query time and evidence age. It invokes `corpus_query`, verifies exact selected capture/index correspondence and materializes at most sixteen captures within the existing byte/value bounds. Missing or altered bytes fail with their exact identity; a surviving digest cannot reconstruct unavailable content. Exported captures remain ordinary inputs to `knowledge_interpret`.

## Bounds, Authority and Evidence

The 128-member catalog limit does not enlarge the common 1 MiB request, 4 MiB response, value/depth or deadline limits. Each original capture body remains bounded to 65,536 UTF-8 bytes. qxctl admits at most 128 members under a 120-second member scheduling/network budget plus one bounded capture/index/checkpoint tail; unfinished members remain recoverable. Preflight, deep reconstruction and finalization have their separate ancestry/object caps and per-owner five-second process bound. The member budget is not a 120-second command wall-clock guarantee. Per-request bounds and checkpoint behavior remain independent adapter limits. There is no crawler, scheduler, source-selected execution or permission expansion.

No selected corpus head is introduced. Immutable cache publication and local job bookkeeping do not create a source/graph selection or require fabrication of an SSIAG/STAV write receipt. Actual protected source and graph selection continue through their existing authorization, intent and audit contracts. The caller owns corpus membership, historical evidence choices and subsequent interpretation/composition policy.

Producer, consumer, interrupted-job, retention-integrity and exact installed-version tests provide evidence only for the exercised scope. Full provider expertise, arbitrary inference, automatic architecture synthesis, provider accounts, deployment, general artifact transport and unimplemented retention deletion/migration remain separate work. A successful source fetch, corpus digest or native parse is not semantic verification.
