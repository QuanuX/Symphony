# Symphony Change Log Vector Manifest

## Identity

SCLV is the Symphony Change Log Vector, a canonical peer vector within SKV.

## Declared Contract Truth Role

- `INTENT.md` declares purpose and boundaries.
- `SPEC.md` owns record and recovery semantics.
- `CHANGELOG.md` is the append-only canonical change ledger.
- `RECOVERY.md` is the operator recovery runbook and incident record.
- `SKILL.md` guides safe use by humans and tools.

## Installability Considerations

SCLV has no independently installed runtime. The repository validator is its bounded executable checker. Any future qxctl view is a derived, read-only projection.

## Core Invariants

- Canonical records describe completed, human-ratified change truth.
- Historical records are never reordered, deleted, or rewritten to conceal an interruption.
- GitHub PR identifiers form a sparse namespace.
- Record file order is recording order, not PR-number order.
- Legacy version-1 records remain valid without temporal fields.
- New records use version 2 and carry start, completion, and recording timestamps.
- A late recovery carries a permanent explanation but no permanently active error state.
- Pending work exists only outside the canonical knowledge tree.

## Non-Authorization Statement

The Architect and designated human reviewers ratify canonical truth. Agents may assist with evidence collection and draft changes. symphony-validator is read-only and cannot repair, ratify, append, reorder, or publish. qxctl has no SCLV mutation authority.

## Relationships

SKVI indexes source truth. SCLV records change truth. SODV governs publication truth. SSCG interprets compatibility. STAV records operational audit events.

## Explicit Non-Authorizations

This manifest authorizes the current human-authored ledger and deterministic validator checks only. It does not authorize autonomous ratification, generated canonical records, public publication, a database projection, or runtime audit mutation.
