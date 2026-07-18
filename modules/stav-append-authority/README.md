# STAV Append Authority

This independently buildable Go module is Symphony's durable, single-writer STAV authority for one immutable TOPS identity per process.

It installs one host executable, enrolls isolated per-TOPS instances, installs bounded per-TOPS launchd/systemd liveness profiles, mutually authenticates Darwin/Linux Unix-socket peers from kernel credentials, authorizes exact producer tuples and reader classifications, appends fsync-backed canonical events, reconstructs idempotency on restart, verifies the digest chain, preserves incomplete-tail evidence, and serves the read-only qxctl STAV interface.

`knowledge/stav/` remains protocol truth. Producers, qxctl, agents, and supervisors never edit the ledger. Start with `INTENT.md`, `MANIFEST.md`, `INSTALL.md`, and `SKILL.md`.
