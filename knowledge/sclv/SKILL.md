# Symphony Change Log Vector Skill

## Purpose

Use this guidance when reviewing, recording, validating, or recovering a canonical Symphony change.

## Normal Recording Procedure

1. Determine whether the merged change alters canonical architecture, contracts, compatibility, publication boundaries, or another SCLV-governed surface. Do not create a record merely because a PR number exists.
2. Verify the final PR URL, merge timestamp, 40-character merge commit, and affected canonical paths.
3. Append a version-2 record after the change has merged and been ratified.
4. Set `recording_disposition` to `post_merge` when recording occurred in the expected closure sequence.
5. Validate the repository and commit the closure record. The closure-carrier PR does not recursively require its own SCLV record unless it makes an independently significant architectural change.

## Interrupted-Session Recovery

1. Inspect the ephemeral marker under `.git/symphony/sclv/pending/` and compare it with GitHub and Git history.
2. If the source PR remains open, resume or explicitly abandon the pending work.
3. If it closed without merge, record the local journal outcome and delete the marker; do not add a canonical SCLV error record.
4. If it merged and has no canonical record, append a version-2 record with `recording_disposition: late_recovery` and a factual `recovery_reason`.
5. Delete the marker only after the canonical recovery record is committed, or after an unmerged operation is confirmed abandoned.

## Review Rules

- Treat PR numbers as sparse identifiers, never a contiguous sequence.
- Treat physical file order as immutable append order.
- Require strict UTC timestamps and `change_started_at <= change_completed_at <= recorded_at`.
- Require version-2 `recorded_at` values to be nondecreasing in physical file order.
- Reject `pending`, `unresolved`, or equivalent states in the canonical ledger.
- Never synthesize a tag, hash, merge commit, or timestamp to satisfy a checker.
- Never edit an existing record to make later history appear orderly.

## Non-Authorization Statement

symphony-validator detects and reports. It never edits. An agent may draft a recovery record, but the Architect or designated human reviewer ratifies it. qxctl may later expose read-only derived evidence only.
