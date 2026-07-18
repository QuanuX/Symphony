# Symphony Change Log Vector Specification

## Status and Authority

This specification is the canonical SCLV behavioral contract. `knowledge/sclv/CHANGELOG.md` is the human-authored canonical record surface. `tools/symphony-validator/` implements deterministic, read-only checks of this contract.

## Purpose

Define which completed changes belong in SCLV, how their immutable records are shaped and ordered, and how interrupted closure sessions recover without rewriting canonical history.

## Record Selection

An SCLV record is required for a completed, human-ratified change that materially alters canonical architecture, contracts, doctrine, compatibility, namespaces, publication boundaries, governed tooling behavior, or another indexed knowledge surface.

SCLV is intentionally sparse. A GitHub pull request may be implementation-only, a closure carrier, or unrelated to SCLV scope. The absence of an SCLV record for such a PR is correct. A record named `SCLV-PR-NNN` must align with PR `NNN` when present, but adjacent records need not have adjacent PR numbers.

## Append-Only Semantics

Records are appended after their source change is complete. Physical file order is canonical recording order. It does not claim PR creation order, merge order, or runtime-event order. Existing records must not be reordered, deleted, or rewritten to disguise concurrency or late recording.

Corrections are new records that reference the earlier evidence. Derived views may sort by `change_completed_at`, but a projection must never rewrite canonical file order.

## Version 1 Compatibility

Records without `record_version` are legacy version-1 records. Their historical shape remains valid. Temporal metadata must not be partially added to a version-1 record.

## Layer 0 Canonical Record Shape

Every new record uses `record_version: 2` and retains the established SCLV fields:

- `record_id`, `title`, `status`, `date`, `change_type`
- `related_pr`, `merge_commit`
- `affected_surfaces`, `skvi_references`
- `change_summary`, `relationship_changes`, `doctrine_changes`
- `compatibility_consequences`, `publication_consequences`, `projection_consequences`
- `evidence`, `non_authorizations`, `notes`

Version 2 additionally requires:

- `change_started_at`: strict UTC time when the source change operation began, normally PR creation.
- `change_completed_at`: strict UTC time when the source change became complete, normally merge time.
- `recorded_at`: strict UTC time when the closure or recovery record was authored.
- `recording_disposition`: `post_merge` or `late_recovery`.
- `recovery_reason`: required and non-empty only for `late_recovery`.

Strict UTC time has shape `YYYY-MM-DDTHH:MM:SSZ`. Leap years and calendar ranges must be valid. Each record must satisfy:

`change_started_at <= change_completed_at <= recorded_at`

Version-2 `recorded_at` values must be nondecreasing in physical file order.

## Canonical State Rule

The only canonical record status is `canonical`. Pending, interrupted, unresolved, abandoned, or failed work is session state and must not be appended to `CHANGELOG.md`. A recovered canonical record explains the interruption through `late_recovery`; it does not preserve a permanently active error flag.

## Ephemeral Session Journal

Tools that coordinate change closure may maintain repository-local state under:

`.git/symphony/sclv/pending/<session-id>.json`

The journal is explicitly noncanonical and uncommitted. A marker should contain its format version, session ID, source operation, base commit, intended surfaces, `started_at`, known PR URL and head commit when available, and current local state. Secrets, credentials, provider payloads, and runtime STAV events are prohibited.

On clean closure, delete the marker after the canonical record is committed. On a later session, reconcile a stale marker against GitHub and Git:

- still open: resume or explicitly abandon;
- closed without merge: confirm abandonment and delete the marker;
- merged and already recorded: verify alignment and delete the marker;
- merged but unrecorded: append a `late_recovery` record, validate, commit it, then delete the marker;
- indeterminate evidence: fail closed and request human review.

This is recovery by forward correction. It is never permission to edit history or fabricate evidence.

## Non-Authorization Statement

SKVI indexes SCLV surfaces. SODV consumes SCLV change truth when governing release or documentation publication. SSCG interprets compatibility. STAV owns per-installation runtime audit truth. Git and GitHub provide evidence only.

Agents may query and propose. Humans ratify. symphony-validator may parse and check, but never mutate or self-heal canonical files. Automated self-healing is limited to reconciling ephemeral state and preparing a forward-only recovery proposal.

## Explicit Non-Authorizations

SCLV does not authorize autonomous ratification, canonical pending records, rewriting or reordering existing records, fabricated evidence, tag publication, runtime audit mutation, generated canonical projections, or qxctl mutation authority.
