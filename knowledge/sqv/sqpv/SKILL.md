# Symphony Quantitative Persistence Vector Skill

## Purpose

Keep research retention exact about what was stored, what survived, and what can be read or reclaimed.

## Reading Order

1. `knowledge/ARCHITECTURE.md` and the SQV Contract Quad
2. this SQPV Quad and the SQMV dataset meaning being retained
3. the selected SQFV lifetime, SQTV lineage, SQDV resume, storage, access, and platform contracts

## Procedure

1. Identify the selected dataset revision, representation, partition, source or derived lineage, writer generation, backend, access scope, and finite retention budget.
2. Declare the exact commit and read guarantee, including synchronization and failure behavior on the target platform.
3. Bind each retained result to artifact identity, integrity evidence, and exact covered ranges; expose gaps rather than inferring contiguity.
4. Test interruption before commit, after commit but before acknowledgement, stale-writer fencing, corruption, exhausted capacity, and read protection during retention.
5. Reconcile retained catch-up with SQFV and SQDV positions without treating storage commit as recipient processing.
6. Review schema, ABI, installation, feature, and namespace consequences only for a concrete adapter.

## Stop Conditions

Stop before treating a buffered write or digest as durable commit, an old writer as current, a high position as gap-free, a storage bridge as a compulsory database, or this architecture Quad as proof of an installed backend.
