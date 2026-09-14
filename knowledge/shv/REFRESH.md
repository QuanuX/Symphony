# SHV source-bound refresh contract v1

New CLI composition, unchanged independently installed C++ engines. Commands `shv refresh build|verify|schema|template`, explicit source/kernel prefixes and versions, state-root, TOPS and source identity. Build/verify additionally select source-root and input. No automatic network access or catalogue publication.

Build input exactly `{expected_source_digest,captures,mapping,profile,subject_ids,requirements}`. Each capture is the existing capture_import input without source_root/source; those are supplied by the selected protected source. One to eight captures; mappings must use their manifest IDs and subjects declared by that source revision. This scope constraint applies to this explicitly source-bound tool, not arbitrary user catalogues.

Under the protected source lock, require expected digest == current committed revision. Capture imports, catalogue build, coverage, evaluation and separate source/catalogue graph projections all execute through existing C++ owners and independent consumer checks. Coverage inventory derives from the exact materialized catalogue, not a second unbound caller inventory. Coverage does not silently filter catalogue/evaluation. Requirements and explicit subject selection remain caller-authored.

The sealed bundle retains the complete request, exact source and installation identities, TOPS/source identity and all stage artifacts. source_root is deliberately external so retained byte-identical files can be relocated. Reinspect installations before releasing the lock. Reject aggregate output beyond the one-MiB replay admission bound. Failed stages produce no bundle or catalogue publication; rerun the original input after correcting external availability. No durable checkpoint or partial result is claimed.

Verify takes an exact bundle, requires its source revision in the selected protected committed history and exact original installations, reexecutes every native stage against actual bytes, and compares the entire canonical bundle. It reports whether the historical source is still current; it never rewinds or republishes it. Canonical seals identify content, not publisher truth. A validly resealed forged catalogue, missing bytes or substituted installation must reject.

This is one-source bounded materialization and historical replay. Cross-source aggregation, comparing successive materializations, durable catalogue heads, resumable partitions and automated acquisition remain subsequent contracts. Request/profile/requirements identities remain individually present, allowing later change classification without attributing a mapping change to a vendor.

The CLI-owned embedded schema exposes Request, Bundle, Verification, Schema and Template definitions. Native schemas are copied with scoped references; independently selected source 0.1.0-dev and kernel 0.1.0-dev/0.2.0-dev enforce their exact semantics. No native version substitution occurs.

## Verified comparison

`REFRESH-COMPARISON.md` adds exact structural comparison after native replay of both bundles. This supersedes the earlier deferred comparison status above. Partitioning, durable publication and causal inference remain unimplemented.
