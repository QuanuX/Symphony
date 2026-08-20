# Symphony Accordare Vector Skill

## Purpose

Guide authorized callers in resolving, evaluating, comparing, and explaining one bounded Symphony composition without displacing the sources that own each fact.

## Required Reading Order

1. `knowledge/sav/INTENT.md`
2. `knowledge/sav/MANIFEST.md`
3. `knowledge/sav/SPEC.md`
4. `knowledge/sav/RELATIONSHIPS.md`
5. `knowledge/sav/TRAITS.md`
6. `knowledge/TIME.md`
7. `knowledge/LIFECYCLE.md`
8. `knowledge/FEATURE-ADMINISTRATION.md`
9. the owner contracts for every supplied projection
10. `knowledge/sev/SPEC.md` before proposing evolution

## Engine Procedure

1. Resolve one exact installed SAV engine by immutable receipt and protected binding; never choose by recency.
2. Inspect its descriptor and verify read-only, freezing-path, no-listener, and disabled canonical-apply boundaries.
3. Obtain fresh SSIAG decisions for every protected source operation.
4. Invoke each source owner separately and validate its closed result.
5. Supply a bounded, sorted source-projection envelope to `current_resolve`.
6. Preserve `coverage_state`, unresolved sources, stable snapshot digest, and timestamped observation digest.
7. Use `reference_check` before `evaluate` when Accord References are caller supplied.
8. Treat `unknown`, `reference_unresolved`, `indeterminate`, and `partial` as explicit evidence, never success.
9. Use `diff`, `explain`, and `project_graph` only for noncanonical evidence.
10. Send findings to SEV only when evolution has been separately requested.

## Caller Authority

Authority derives from effective host ownership or exact SSIAG permission, not caller class. Engine output is evidence, never permission.

## Stop Conditions

Stop when evidence owners conflict, a critical schema is unknown, a projection digest is stale, a source claims authority outside its namespace, coverage is asserted without evidence, an overlay would weaken a closed invariant, or a requested action requires mutation not covered by a separately authorized lifecycle operation.

## Non-Authorization Statement

This skill does not authorize source discovery by ambient filesystem scan, canonical writes, Named Version sealing, lifecycle apply, arbitrary graph persistence, automatic compatibility inference, or public claims.
