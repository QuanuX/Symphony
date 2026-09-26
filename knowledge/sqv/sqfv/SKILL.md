# Symphony Quantitative Flow Vector Skill

## Purpose

Keep research-data movement bounded, independently composable, and honest about ownership, release, and failure.

## Reading Order

1. `knowledge/ARCHITECTURE.md` and the SQV Contract Quad
2. this SQFV Quad and the exact SQMV descriptor meaning it carries
3. `modules/sqfv-batch-cpp/SPEC.md` and `include/symphony/sqfv/batch.h` when using the first trusted same-process library
4. selected acquisition, transformation, persistence, delivery, access, Habitat, and transport contracts

## Procedure

1. Identify the exact producer, consumer, representation, scope, locality, and failure policy for each edge.
2. Separate source-native position, internal cursor, producer generation, and any retained position.
3. State byte and allocation budgets, queue limits, credit units, oversize behavior, and reserved control capacity before claiming bounded flow.
4. Prove that mutable preparation ends before publication and that every real reader or device releases or is fenced before memory reuse.
5. Test stalled readers, independent branches, cancellation, oversize input, scope rejection, stale generations, and bounded exhaustion at each implemented boundary.
6. Keep a transport acknowledgement separate from consumer processing, persistent commit, and memory release.
7. Review exact ABI, frame, receipt, feature, and administration consequences when an installable implementation is proposed.

## Stop Conditions

Stop before claiming a cross-process handle, network transport, durable recovery, access grant, provider semantics, or performance threshold from the first library or offline prototype. Claim an installed package only against its exact receipt and tested extent. Do not require a central bus, cache, or qxctl relay for user-selected paths.
