# Symphony Change Log Vector Manifest

## Canonical Surfaces

- `knowledge/sclv/CHANGELOG.md`
- `knowledge/sclv/INTENT.md`
- `knowledge/sclv/MANIFEST.md`
- `knowledge/sclv/RECOVERY.md`
- `knowledge/sclv/SKILL.md`
- `knowledge/sclv/SPEC.md`

## Identity

SCLV is the Symphony Change Log Vector, a canonical peer vector within SKV.

## Declared Contract Truth Role

- `INTENT.md` declares purpose and boundaries.
- `SPEC.md` owns record and recovery semantics.
- `CHANGELOG.md` is the append-only canonical change ledger.
- `RECOVERY.md` is the operator recovery runbook and incident record.
- `SKILL.md` guides safe use by humans and tools.

## Installability Considerations

SCLV has an independently installable C++ proposal engine at `modules/sclv-engine/` with executable `symphony-sclv`. Its initial operations are inspect, check, propose, ephemeral-session recovery, and derived projection. The repository validator remains an independent bounded checker. qxctl invokes the engine but does not own or directly append the ledger.

## Core Invariants

- Canonical records describe completed, permission-backed ratified change truth.
- Historical records are never reordered, deleted, or rewritten to conceal an interruption.
- GitHub PR identifiers form a sparse namespace.
- Record file order is recording order, not PR-number order.
- Legacy version-1 records remain valid without temporal fields.
- Existing version-2 records remain valid and immutable.
- Provider-neutral version 3 is the prospective writable record format after the engine/validator activation increment merges. Version-1 and version-2 records remain immutable and valid.
- `affected_surfaces` retain safe historical path provenance without imposing current file-presence or SKVI-membership obligations.
- New `skvi_references` identify surviving current evidence and are regular-file and current-SKVI obligations at proposal/admission; later retirement is non-failing historical drift, not retroactive record invalidation.
- A late recovery carries a permanent explanation but no permanently active error state.
- Pending work exists only outside the canonical knowledge tree.

## Non-Authorization Statement

The Architect and any caller granted the applicable review/ratification permission may ratify canonical truth. Any caller or tool may assist with evidence collection and draft changes within its effective permissions. `symphony-sclv` and symphony-validator are authority-free; neither may ratify, append, reorder, or publish. qxctl has no implemented SCLV apply surface.

## Relationships

SKVI indexes source truth. SCLV records change truth. SODV governs publication truth. SSCG interprets compatibility. STAV records operational audit events.

## Explicit Non-Authorizations

This manifest authorizes the repository-maintained ledger, deterministic validator checks, bounded proposal/projection engine, and separately discoverable local-Git and air-gapped declaration evidence processes. It does not let an authority-free process manufacture ratification, write generated canonical records, publish publicly, or mutate runtime audit state.
