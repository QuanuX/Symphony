# Knowledge Session Coordinator Skill

## Purpose

Guide safe direct use and development of the reconciliation coordinator foundation.

## Direct Diagnostics

```bash
symphony-knowledge-session --help
symphony-knowledge-session --version
symphony-knowledge-session --descriptor
```

Without arguments, send exactly one bounded `symphony.knowledge.engine-process.v1` request on standard input. `inspect` accepts an empty payload. `check` accepts exactly `paths` and `expected_snapshot_digest`, runs relative to the process working directory, follows no symlinks, and returns digests rather than file contents. Reconciliation, authenticated-session, and report-only `lifecycle_plan` operations accept their exact common command schemas and return their exact common results.

## Safety Rules

- Use a deadline no more than five minutes ahead.
- Treat stdout as one protocol response; do not mix diagnostic text into it.
- Pass no secrets, credentials, arbitrary commands, provider payloads, or absolute portable paths.
- Treat `installed_undocked`, implemented session mutation, and disabled canonical apply states literally.
- Do not infer session authentication from a successful `inspect` or `check`.
- Require a compatibility mode of `full` before mutation.
- Use a stable operation ID and exact journal digest for every mutation.
- Use discovery recovery only after normal status fails and preserve its repair evidence.
- Never delete slots, bypass a critical extension, or rewrite an incompatible journal to make it appear current.
- For every session operation, require a fresh exact SSIAG allow decision and recompute its non-transferable capability binding; never treat the JSON object as bearer authority.
- Keep session and reconciliation journals separate. Context references may attach to an authority epoch but never become identity or permission evidence.
- Treat lifecycle desired state and observation as caller-supplied, digest-bound evidence. Require full explicit protocol/capability overlap, preserve blocked components, and use the returned dependency-ready set rather than array order or version recency.
- Treat every lifecycle action, receptor identity, blocker, and target-state digest as a report. `apply_authorized: false` is absolute in this slice.

## Stop Conditions

qxctl may select this exact inactive-undocked package in its protected user-default binding registry and invoke its reconciliation and authenticated-session operations. Selection and invocation do not activate the package or grant authority; SSIAG remains the decision authority.

qxctl may compose the existing authenticated-session operations into an explicit login, refresh, or logout transition. Treat the host event ID as the stable retry identity and preserve every underlying operation's separate SSIAG decision and journal evidence. Do not add a coordinator-side watcher, login hook, or implicit transition operation.

Stop for a new reviewed increment before enabling direct coordinator-to-SSIAG/STAV calls, vector-engine invocation, configured-root lifecycle observation, desired-profile administration, lifecycle persistence or apply, automatic format migration beyond the declared compatibility window, observers/hooks, canonical apply, system/TOPS binding or installation claims, an unversioned active alias, or live Maestro docking.
