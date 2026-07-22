# SODV Engine Skill

## Purpose

Use `symphony-sodv` through qxctl to inspect and validate the canonical release ledger, compare caller-supplied immutable publication evidence, create noncanonical forward proposals, reconcile noncanonical journals, and build disposable inventories.

## Procedure

1. Read `knowledge/sodv/` and identify the exact installed version and prefix.
2. Run `qxctl sodv check` before preparing or resuming a transaction; bind automation to the returned ledger digest.
3. Supply raw observed tag and package state only through the exact bounded JSON schema. Use `verify` to distinguish absence, propagation delay, completion readiness, canonical completion, and mismatch.
4. Use `propose` only for one explicit forward v2 record. Inspect the proposal, its read/write sets, and `ratified: false`; it is not applied.
5. Use `recover` with the digest-bound local journal and the same authorization lineage. Follow `delete_recommended` only after the result reports an already canonical completion.
6. Use `project` only for a disposable, rebuildable inventory.

## Fail-Closed Conditions

Stop for review on stale ledger digests, unknown fields, unsafe paths/tags, duplicate IDs or units, time reversal, changed authorization coordinates/versions/tags/revisions, external-state mismatch, symlinked inputs, deadline/resource failures, or a recovery proposal for another lineage.

## Boundaries

Treat every engine result as evidence. Never treat it as caller authentication, release permission, canonical apply, tag or artifact publication, public-proxy observation, Mintlify authorization, or a release-completion claim.
