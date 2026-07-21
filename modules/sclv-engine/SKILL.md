# SCLV Engine Skill

## Purpose

Use the installed SCLV processes to inspect/check the canonical ledger, assemble a caller-declared v3 proposal, reconcile an ephemeral journal, or build a disposable projection.

## Safe Procedure

1. Invoke the exact installed version through qxctl or direct bounded process input.
2. Collect revision and ratification evidence through an applicable bounded adapter.
3. Treat adapter evidence as evidence, not permission.
4. Keep proposals and projections noncanonical until a separate permission-backed apply path exists.
5. Preserve all v1/v2 records and physical record order.

## Boundaries

Never provide secrets, credentials, proofs, raw assertions, provider payloads, shell commands, environment dumps, absolute portable paths, or fabricated revision/ratification facts. `recover` never deletes the caller's journal. Apply, commit, lifecycle activation, docking, and publication remain unavailable.
