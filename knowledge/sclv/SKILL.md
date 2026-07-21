# Symphony Change Log Vector Skill

## Purpose

Use this guidance when reviewing, recording, validating, or recovering a canonical Symphony change.

## Normal Recording Procedure

1. Determine whether the merged change alters canonical architecture, contracts, compatibility, publication boundaries, or another SCLV-governed surface. Do not create a record merely because a PR number exists.
2. Verify the applicable provider-neutral change-request state, exact revision scheme/value, tree/content digest, completion time, ratification evidence, and affected canonical paths. A GitHub PR and 40-character Git SHA-1 are one supported evidence combination, not universal requirements.
3. Append a version-3 record after the change has merged and been ratified. Use `schemas/v3/record.schema.json` and `templates/v3/record.md`; never rewrite a v1/v2 record.
4. Set `recording_disposition` to `post_merge` when recording occurred in the expected closure sequence.
5. Validate the repository and commit the closure record. The closure-carrier PR does not recursively require its own SCLV record unless it makes an independently significant architectural change.

## Interrupted-Session Recovery

1. Inspect the ephemeral marker under `.git/symphony/sclv/pending/` and compare it with GitHub and Git history.
2. If the source PR remains open, resume or explicitly abandon the pending work.
3. If it closed without merge, record the local journal outcome and delete the marker; do not add a canonical SCLV error record.
4. If it merged and has no canonical record, prepare a version-3 record with `recording_disposition: late_recovery` and a factual `recovery_reason`.
5. Delete the marker only after the canonical recovery record is committed, or after an unmerged operation is confirmed abandoned.

An installed `symphony-sclv` engine may perform the evidence reconciliation in steps 1-4 and emit a bounded proposal. It may not append, ratify, or commit the proposal. If recovery crosses logout, expiry, revocation, or required re-authentication, establish a fresh authenticated session before an authorized operation continues.

## Review Rules

- Treat PR numbers as sparse identifiers, never a contiguous sequence.
- Treat physical file order as immutable append order.
- Require strict UTC timestamps and `change_started_at <= change_completed_at <= recorded_at`.
- Require version-2 and version-3 `recorded_at` values to be nondecreasing in physical file order.
- Reject `pending`, `unresolved`, or equivalent states in the canonical ledger.
- Never synthesize a tag, hash, merge commit, or timestamp to satisfy a checker.
- Never edit an existing record to make later history appear orderly.
- Emit a canonical version-3 record only after the exact schema/template and validator activation increment has merged.
- Keep GitHub, GitLab, local, air-gapped, and proprietary provider evidence behind the same typed v3 semantics.

## Non-Authorization Statement

symphony-validator detects and reports. It never edits. The SCLV engine may inspect, check, reconcile ephemeral state, propose, and project; it never ratifies or directly appends. Any caller may draft a recovery record within its effective permissions, but only the Architect or another caller holding the applicable ratification permission may ratify it. Caller type is not evaluated. qxctl may invoke proposal/read operations; apply remains disabled.
