# Maestro Skill

## Purpose

Inspect or administer exact Maestro receptor presence through qxctl.

## Procedure

1. Resolve an exact receipt-validated Maestro installation.
2. Use inspect or status for one receptor, or `qxctl maestro inventory` for a complete derived TOPS-wide read-only view.
3. Perform dock and undock only through `qxctl knowledge lifecycle apply` after the coordinator durably prepares the exact dependency-ready action.
4. Reuse operation identifiers and exact expected registry digests during retry.
5. Use explicit recovery only for one unambiguous digest-linked forward state.

## Boundaries

Never edit presence slots or heads, infer a preferred version, bypass SSIAG, treat installation as docking, invoke an engine through this slice, or use Maestro presence or derived inventory as canonical vector truth. An inventory failure is not an empty inventory; repair the named receptor stream first.
