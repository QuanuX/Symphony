# SQMV Metadata C++ Use

## Purpose

Use the exact installed C++26 module to create or resolve an immutable SQM1
manifest and derive the existing SQFV binding. Read the installed SPEC for the
wire contract, bounds, ownership, and precise verification extent.

## Operation

1. Select explicit positive `Limits` and a complete `Description`.
2. Supply exactly one producer-attributed schema, layout, and access evidence
   reference. Add source, time, coverage, lineage, or units evidence only when
   the caller has such evidence. Omission retains an unknown, not a default fact.
3. Use `Manifest::create` to freeze caller-owned values. Use `Manifest::resolve`
   with the expected exact reference when accepting retained or transferred
   bytes. Treat returned statuses explicitly and retain the manifest as long
   as its borrowed views are needed.
4. Use `binding` for the five SQFV fields and `verify_binding` when receiving
   a batch under an already resolved manifest. Keep source acquisition IDs and
   per-batch transfer state under their respective owner contracts.

## Scope

The library checks structure, exact digest, exact expected reference, and exact
binding bytes. The caller resolves and interprets opaque evidence according to
the relevant source or schema contract and obtains actual access authority from
its owner. A digest proves byte identity, not producer authenticity or rights.
