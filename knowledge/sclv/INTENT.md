# Symphony Change Log Vector Intent

### Purpose

SCLV preserves permission-backed, ratified change truth for architectural, contractual, compatibility, and publication-significant changes to Symphony.

### Scope

SCLV records completed canonical changes and their evidence. It is intentionally selective: implementation-only pull requests, closure carriers, and unrelated repository activity need no SCLV record unless they change a governed contract or boundary.

SCLV is append-only at the record level. Git and pull-request history remain supporting evidence; neither replaces SCLV, and SCLV does not attempt to mirror every GitHub event.

## Operational Intent

SCLV supports concurrent work without treating GitHub pull-request numbers as a contiguous ledger sequence. A pull-request number identifies its matching record when one exists; it is not an SCLV sequence number.

Work in progress belongs in an ephemeral session journal outside the canonical knowledge tree. Only completed, ratified truth enters `CHANGELOG.md`. Interrupted sessions are reconciled on the next session through an ordinary post-merge record or an explicitly explained late-recovery record, never by rewriting prior records.

## Relationships

- SKVI indexes SCLV canonical surfaces.
- SODV governs release and documentation publication truth informed by SCLV.
- symphony-validator checks SCLV shape, identity, sparse-namespace, and temporal invariants.
- the independently installed SCLV engine may inspect, check, propose, recover ephemeral closure state, and build derived projections under `knowledge/SPEC.md`.
- qxctl administers those proposal-only operations but has no implemented canonical mutation authority.
- NotebookLM may align corpus context, and Mintlify may publish derived documentation; neither is canonical authority.

### Non-scope

SCLV is not Git history, PR history, a runtime audit ledger, a release artifact registry, or an agent-owned decision log. STAV owns runtime audit-event truth. SODV owns publication authorization and completion records.

### Non-authorization Statement

This surface authorizes a proposal-only C++ SCLV engine at `modules/sclv-engine/` after the common contract transition merges. It does not grant ratification permission merely because an engine or caller generated a proposal or validation result. A caller holding the required repository/transition-owner permission ratifies; authority-free tools detect, check, reconcile evidence, and propose bounded forward changes. No caller may use SCLV to edit prior records, manufacture merge evidence, publish releases, mutate STAV ledgers, or expose public documentation.
