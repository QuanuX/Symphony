# Knowledge Session Coordinator Skill

## Purpose

Guide safe direct use and development of the reconciliation coordinator foundation.

## Direct Diagnostics

```bash
symphony-knowledge-session --help
symphony-knowledge-session --version
symphony-knowledge-session --descriptor
```

Without arguments, send exactly one bounded `symphony.knowledge.engine-process.v1` request on standard input. `inspect` accepts an empty payload. `check` accepts exactly `paths` and `expected_snapshot_digest`, runs relative to the process working directory, follows no symlinks, and returns digests rather than file contents. Reconciliation, authenticated-session, persistent `ssfv_maintenance_begin|status|checkpoint|close|recover`, report-only `lifecycle_plan`, durable report-only `lifecycle_boot|lifecycle_boot_status|lifecycle_boot_recover`, and apply-capable `lifecycle_apply_prepare|lifecycle_apply_finalize|lifecycle_apply_close|lifecycle_apply_status|lifecycle_apply_recover` operations accept their exact common command schemas and return their exact common results.

## Safety Rules

- Use a deadline no more than five minutes ahead.
- Treat stdout as one protocol response; do not mix diagnostic text into it.
- Pass no secrets, credentials, arbitrary commands, provider payloads, or absolute portable paths.
- Treat `installed_undocked`, implemented protected lifecycle apply coordination, and disabled canonical apply states literally.
- Do not infer session authentication from a successful `inspect` or `check`.
- Require a compatibility mode of `full` before mutation.
- Use a stable operation ID and exact journal digest for every mutation.
- Use discovery recovery only after normal status fails and preserve its repair evidence.
- Never delete slots, bypass a critical extension, or rewrite an incompatible journal to make it appear current.
- For every session operation, require a fresh exact SSIAG allow decision and recompute its non-transferable capability binding; never treat the JSON object as bearer authority.
- Keep session and reconciliation journals separate. Context references may attach to an authority epoch but never become identity or permission evidence.
- Keep the SSFV maintenance journal separate from both. Preserve its original baseline snapshot and baseline engine identity across compatible engine upgrades; checkpoint only with the exact current journal digest, live session digest, binding digest, current SSFV evidence, and explicit observed/not-configured Maestro evidence.
- Treat `review_required` as noncanonical evidence for caller review. Never convert it into feature-worthiness, ratification, `FEATURES.md` generation, or apply authority.
- Treat lifecycle desired state and observation as caller-supplied, digest-bound evidence. Require full explicit protocol/capability overlap, preserve blocked components, and use the returned dependency-ready set rather than array order or version recency.
- Treat observation collection time as document evidence only. Verify timestamp-only refresh preserves the stable inventory key, transaction, and semantic action identities.
- For lifecycle boot mutation, require a fresh exact SSIAG decision, a stable operation ID, and `absent` or the exact current journal digest. Use discovery recovery only when the local head cannot be trusted and the slot chain is unique.
- Preserve noncritical lifecycle extensions exactly. Unknown critical/newer state, divergent equal generations, and unlinked generation jumps are stop conditions.
- Treat every report-journal action, receptor identity, blocker, and target-state digest as non-executable evidence. Only the separately authorized v2 prepare/finalize/close circuit may coordinate a qxctl external action, and it never authorizes canonical mutation.
- Require the exact report-journal digest, apply-journal digest, applied-state digest, desired/profile evidence, stable inventory, and artifact set before every apply mutation. Preserve the prepared action across interruption and finalize only from a newly verified complete observation.
- Keep v1 and v2 lifecycle journals side by side. Do not rewrite a report stream into an apply stream, select an orphan applied-state file, or recover across a divergent chain.

## Stop Conditions

qxctl may select this exact inactive-undocked package in its protected user-default binding registry and invoke its reconciliation and authenticated-session operations. Selection and invocation do not activate the package or grant authority; SSIAG remains the decision authority.

qxctl may compose the existing authenticated-session operations into an explicit login, refresh, or logout transition. Treat the host event ID as the stable retry identity and preserve every underlying operation's separate SSIAG decision and journal evidence. Do not add a coordinator-side watcher, login hook, or implicit transition operation.

Stop for a new reviewed increment before enabling direct coordinator-to-SSIAG/STAV calls, vector-engine invocation, configured-root lifecycle observation, desired-profile administration, coordinator-owned host action execution, direct Maestro state writes, receipt-v1 mutation, arbitrary entry-point execution, live service/process activation, automatic format migration beyond the declared compatibility window, observers/hooks, canonical apply, system/TOPS binding claims, coordinator-owned or in-place self-handoff, or an unversioned active alias. qxctl's candidate-verified out-of-place handoff is implemented; the coordinator may prepare and verify externally executed binding and Maestro actions without owning those mutations.
