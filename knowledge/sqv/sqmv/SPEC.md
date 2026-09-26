# Symphony Quantitative Metadata Vector Specification

## Descriptive Ownership

SQMV owns the research-data description contract: dataset and snapshot identity, schema and representation references, units, supported time roles and precision, source revisions, coverage and known gaps, provenance, transformation lineage, and source-supplied rights or access classifications. Each assertion must remain attributable to its producer and exact evidence scope. The contract distinguishes a source-native identifier from a Symphony dataset revision and from a consumer-selected view.

SQMV can index or project those records for discovery. An index or graph is a derived view; it must not silently rewrite retained observations, promote an interpretation into a source fact, or confer redistribution permission. No universal provider schema or canonical storage format is selected here.

## Time, Version, and Completeness

Source event, publication, revision or effective, provider receipt, Symphony acquisition, preparation, and retention-commit times are distinct roles when supported by evidence. A missing source timestamp must not be filled with a local timestamp under the same role. Precision, calendar, clock origin, and uncertainty remain part of the interpretation.

Provider version, source schema version, representation layout, dataset revision, transformation release, and module version remain separate dimensions. An explicit compatibility decision is required before a reader interprets a changed schema or layout as equivalent. Coverage describes admitted intervals, exclusions, gaps, and unresolved unknowns; a valid file or complete response need not mean a complete dataset.

## Runtime Relationship

SQMV owns the meaning of metadata carried or referenced by a prepared batch; SQFV owns the batch's transfer, buffer lifetime, and cursor mechanics. A compact reference is usable only after its exact version and interpretation have been resolved for the receiving scope. Metadata needed per batch should be immutable for that batch's lifetime. No resident catalogue or synchronous lookup is required on a data path by this specification. The current `sqfv-batch-cpp` binding compares caller-supplied reference, dataset revision, schema version, layout version, and access scope as opaque exact bytes. It does not define their SQMV grammar, resolve a reference, or authorize a reader.

## Deferred Technical Contract

The exact descriptor schema, dataset identifier grammar, version-negotiation rules, metadata persistence, privacy enforcement mechanism, and public ABI remain to be admitted through a later reviewed contract. The offline SQV prototype does not settle them. This Quad creates no installed capability or namespace family.

## Non-Authorization Statement

SQMV does not assert source authenticity from a digest alone, grant access, supply provider data, move bulk bytes, commit durability, reinterpret historical snapshots as latest data, or define a user's research method or live strategy.
