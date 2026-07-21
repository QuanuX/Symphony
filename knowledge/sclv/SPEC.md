# Symphony Change Log Vector Specification

## Status and Authority

This specification is the canonical SCLV behavioral contract. `knowledge/sclv/CHANGELOG.md` is the repository-maintained canonical record surface. `tools/symphony-validator/` implements deterministic, read-only checks of this contract.

## Purpose

Define which completed changes belong in SCLV, how their immutable records are shaped and ordered, and how interrupted closure sessions recover without rewriting canonical history.

## Record Selection

An SCLV record is required for a completed, permission-backed ratified change that materially alters canonical architecture, contracts, doctrine, compatibility, namespaces, publication boundaries, governed tooling behavior, or another indexed knowledge surface.

SCLV is intentionally sparse. A GitHub pull request may be implementation-only, a closure carrier, or unrelated to SCLV scope. The absence of an SCLV record for such a PR is correct. A record named `SCLV-PR-NNN` must align with PR `NNN` when present, but adjacent records need not have adjacent PR numbers.

## Append-Only Semantics

Records are appended after their source change is complete. Physical file order is canonical recording order. It does not claim PR creation order, merge order, or runtime-event order. Existing records must not be reordered, deleted, or rewritten to disguise concurrency or late recording.

Corrections are new records that reference the earlier evidence. Derived views may sort by `change_completed_at`, but a projection must never rewrite canonical file order.

## Version 1 Compatibility

Records without `record_version` are legacy version-1 records. Their historical shape remains valid. Temporal metadata must not be partially added to a version-1 record.

## Version 2 Compatibility

Version-2 records and the current validator contract remain valid and immutable. Version 2 is the writable format until the version-3 Markdown schema/template and validator conformance increment are reviewed and merged. Activation of version 3 never rewrites a version-1 or version-2 record.

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

## Provider-Neutral Version 3 Direction

Version 3 removes GitHub-shaped evidence from the universal record contract. Its exact Markdown schema must be separately checked before the first version-3 record, but every conforming version-3 record must represent these typed facts:

- stable SCLV record identity independent of a pull-request number;
- provider-neutral change-request reference with provider namespace, opaque identifier, and an explicit absent/not-applicable state when no forge exists;
- revision scheme and opaque revision value, including Git SHA-1, Git SHA-256, or a registered provider-defined scheme;
- exact tree/content digest independent of the review provider;
- permission-backed ratification evidence naming the accountable subject, effective permission, method, and evidence digest/reference;
- start, completion, and recording timestamps plus post-merge or late-recovery disposition;
- affected surfaces, relationship/doctrine/compatibility/publication/projection consequences, safe evidence, and non-authorizations.

GitHub, GitLab, Gerrit, local Git, air-gapped host-authority approval, and proprietary review systems are adapters to this model. No provider URL, 40-character SHA-1, or forge object is universally required. Provider payloads, credentials, raw assertions, and unbounded review content remain excluded.

The v3 activation increment must update symphony-validator and its fixtures before any v3 record is accepted. Until that increment merges, an engine may emit a noncanonical v3 migration preview but not claim the preview is ledger-valid.

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
- indeterminate evidence: fail closed and request review by a caller holding the applicable permission.

This is recovery by forward correction. It is never permission to edit history or fabricate evidence.

## SCLV Engine Operations

The initial `symphony-sclv` operations are:

- `inspect`: report ledger, contract, and adapter compatibility;
- `check`: produce deterministic shape, continuity, temporal, and provider-evidence diagnostics;
- `propose`: create an immutable forward-only record proposal without appending it;
- `recover`: reconcile an ephemeral closure journal with observed repository/provider state and emit a no-op, abandonment result, or late-recovery proposal;
- `project`: build a disposable provider-neutral view.

The engine uses `symphony.knowledge.engine-process.v1`; qxctl exposes vector-specific operations under `qxctl sclv ...`. It may update or delete safe ephemeral journal state as defined here, but programmatic canonical append remains disabled until the common apply gate passes.

## Non-Authorization Statement

SKVI indexes SCLV surfaces. SODV consumes SCLV change truth when governing release or documentation publication. SSCG interprets compatibility. STAV owns per-installation runtime audit truth. Git and GitHub provide evidence only.

Callers may query or propose within their effective permissions. A caller holding the required repository/transition-owner permission ratifies, regardless of caller type. symphony-validator may parse and check, but never mutate or self-heal canonical files. Automated self-healing is limited to reconciling ephemeral state and preparing a forward-only recovery proposal; generation does not manufacture authority.

## Explicit Non-Authorizations

SCLV does not grant ratification to an authority-free process, authorize canonical pending records, permit rewriting or reordering existing records, accept fabricated evidence, publish tags, mutate runtime audit data, create canonical projections, or enable qxctl apply authority before the common gate passes.
