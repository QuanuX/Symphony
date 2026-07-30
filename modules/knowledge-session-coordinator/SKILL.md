# Knowledge Session Coordinator Skill

## Purpose

Guide safe direct use and development of the reconciliation coordinator foundation.

## Direct Diagnostics

```bash
symphony-knowledge-session --help
symphony-knowledge-session --version
symphony-knowledge-session --descriptor
```

Without arguments, send exactly one bounded `symphony.knowledge.engine-process.v1` request on standard input. `inspect` accepts an empty payload. `check` accepts exactly `paths` and `expected_snapshot_digest`, runs relative to the process working directory, follows no symlinks, and returns digests rather than file contents. Reconciliation operations accept the exact common command schema and return the exact common result schema.

## Safety Rules

- Use a deadline no more than five minutes ahead.
- Treat stdout as one protocol response; do not mix diagnostic text into it.
- Pass no secrets, credentials, arbitrary commands, provider payloads, or absolute portable paths.
- Treat `installed_undocked`, `reserved`, and `disabled` descriptor states literally.
- Do not infer session authentication from a successful `inspect` or `check`.
- Require a compatibility mode of `full` before mutation.
- Use a stable operation ID and exact journal digest for every mutation.
- Use discovery recovery only after normal status fails and preserve its repair evidence.
- Never delete slots, bypass a critical extension, or rewrite an incompatible journal to make it appear current.

## Stop Conditions

qxctl may select this exact inactive-undocked package in its protected user-default binding registry and invoke its reconciliation operations. Selection and invocation do not activate the package or establish authority.

Stop for a new reviewed increment before enabling authenticated-session mutation, SSIAG/STAV calls, vector-engine invocation, automatic format migration beyond the declared compatibility window, observers/hooks, apply, system/TOPS installation claims, an unversioned active alias, or Maestro docking.
