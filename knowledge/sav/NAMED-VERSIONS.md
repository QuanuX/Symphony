# SAV Named Versions

## Purpose

A Named Version is an immutable, digest-backed composition envelope. It identifies an exact reusable Symphony composition without replacing package versions, engine versions, Git identities, receipts, or SODV publication truth.

## Lifecycle

1. A caller supplies a complete candidate envelope and exact authority evidence.
2. SAV validates identity, ordering, predecessor lineage, content bindings, platform bounds, traits, and Accord References without persisting it.
3. qxctl requests a fresh caller-neutral SSIAG decision for the exact candidate digest.
4. The knowledge-session coordinator durably prepares the immutable write, then finalizes only the SAV-validated bytes under expected-state compare-and-swap.
5. qxctl re-reads and revalidates the stored artifact before closure and submits safe STAV metadata.

An alias is a convenience selector, never identity. One alias cannot select two digests in the same profile. A changed composition creates a successor whose `predecessor_digest` names the prior immutable envelope. Rollback selects an earlier digest; it never rewrites lineage. SODV linkage is null until an actual publication exists.

## Bounds and Non-Authority

V1 uses the bounds in `SPEC.md`, runs only on the freezing path, and preserves exact side-by-side versions. SAV does not authorize or persist a seal. qxctl does not define Named Version semantics. The coordinator does not validate composition truth. A database, graph, head file, or alias index is disposable selection evidence and never the immutable artifact itself.
