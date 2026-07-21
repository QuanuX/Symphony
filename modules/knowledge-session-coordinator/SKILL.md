# Knowledge Session Coordinator Skill

## Purpose

Guide safe direct use and development of the read-only coordinator foundation.

## Direct Diagnostics

```bash
symphony-knowledge-session --help
symphony-knowledge-session --version
symphony-knowledge-session --descriptor
```

Without arguments, send exactly one bounded `symphony.knowledge.engine-process.v1` request on standard input. `inspect` accepts an empty payload. `check` accepts exactly `paths` and `expected_snapshot_digest`, runs relative to the process working directory, follows no symlinks, and returns digests rather than file contents.

## Safety Rules

- Use a deadline no more than five minutes ahead.
- Treat stdout as one protocol response; do not mix diagnostic text into it.
- Pass no secrets, credentials, arbitrary commands, provider payloads, or absolute portable paths.
- Treat `installed_undocked`, `reserved`, and `disabled` descriptor states literally.
- Do not infer session authentication from a successful `inspect` or `check`.

## Stop Conditions

Stop for a new reviewed increment before enabling session mutation, persistent journals, locks, authentication, qxctl grammar, SSIAG/STAV calls, apply, system/TOPS installation claims, an unversioned active alias, or Maestro docking.
