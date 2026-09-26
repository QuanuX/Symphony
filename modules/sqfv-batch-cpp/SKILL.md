# SQFV Batch C++ Skill

## Use

1. Read [INTENT.md](INTENT.md), [SPEC.md](SPEC.md), and the public [C header](include/symphony/sqfv/batch.h) before integrating the library.
2. Supply finite context, frame, descriptor, port, and byte budgets. Keep the exact dataset/view revision, representation, scope, partition, generation, and sequence outside any implicit latest-version lookup.
3. Copy prepared bytes into an immutable batch before offering it. Hold every returned read lease for as long as its view is used, then release that exact lease. Releasing a lease is memory and credit accounting; it does not attest processing or persistence.
4. Treat a blocked offer, duplicate, gap, stale position, changed-byte conflict, and scope mismatch as distinct outcomes. Handle a stalled optional port without borrowing another port's credit.
5. Use the frame codec only for its documented bounded local serialization purpose. Authenticate and authorize externally when a later selected transport requires it.
6. Build against one exact installed CMake package. Verify its receipt and test a source-checkout-free consumer when changing the ABI or package layout.

## Stop Conditions

Do not infer a production SQMV identity, credential grant, private-data isolation, durable commit, cross-process lifetime, network security, exactly-once processing, or throughput guarantee from this trusted same-process module.
