# STAV Append Authority

This independently buildable Go module is Symphony's durable, single-writer STAV authority for one immutable TOPS identity per process.

The current public source release is `github.com/QuanuX/Symphony/modules/stav-append-authority@v0.2.1`, which includes operational UID/GID validation and ownership verification after changes. Version `v0.2.0` remains immutable historical evidence for the supervised boundary; version `v0.1.0` remains immutable historical evidence for the pre-supervision boundary. These are public Go source releases, not GitHub binary releases.

It installs immutable receipt-v2 executable packages, enrolls isolated per-TOPS instances, installs bounded per-TOPS launchd/systemd liveness profiles, mutually authenticates Darwin/Linux Unix-socket peers from kernel credentials, authorizes exact producer tuples and reader classifications, appends fsync-backed canonical events, reconstructs idempotency on restart, verifies the digest chain, preserves incomplete-tail evidence, and serves the read-only qxctl STAV interface. The installed executable also owns the bounded `foundation-lifecycle` JSON adapter for offline enrollment and supervisor observation, planning, apply status, mutation, and recovery.

Foundational mutation uses digest-linked attempts outside the purged STAV subtree. Ordinary apply commits an exact intent through the closed SSIAG audit bridge before native mutation. It records the observed outcome separately; stopping either foundation may leave that outcome `audit_deferred` until explicit restart and recovery obtain its own real receipt. Bootstrap requires administrator-authored exact SSIAG producer grants. See `knowledge/ssiag/SPEC.md` for the two event pairs and recovery contract.

`knowledge/stav/` remains protocol truth. No producer, qxctl client, supervisor, or other caller edits the ledger through a supported interface. Start with `INTENT.md`, `MANIFEST.md`, `INSTALL.md`, and `SKILL.md`.
