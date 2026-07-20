# STAV Append Authority

This independently buildable Go module is Symphony's durable, single-writer STAV authority for one immutable TOPS identity per process.

The current public source release is `github.com/QuanuX/Symphony/modules/stav-append-authority@v0.2.0`. Version `v0.1.0` remains immutable historical evidence for the pre-supervision boundary. Neither version is a GitHub binary release.

It installs one host executable, enrolls isolated per-TOPS instances, installs bounded per-TOPS launchd/systemd liveness profiles, mutually authenticates Darwin/Linux Unix-socket peers from kernel credentials, authorizes exact producer tuples and reader classifications, appends fsync-backed canonical events, reconstructs idempotency on restart, verifies the digest chain, preserves incomplete-tail evidence, and serves the read-only qxctl STAV interface.

`knowledge/stav/` remains protocol truth. No producer, qxctl client, supervisor, or other caller edits the ledger through a supported interface. Start with `INTENT.md`, `MANIFEST.md`, `INSTALL.md`, and `SKILL.md`.
