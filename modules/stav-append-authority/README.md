# STAV Append Authority

This independently buildable Go module is Symphony's durable, single-writer STAV authority for one immutable TOPS identity per process.

The current public source release is `github.com/QuanuX/Symphony/modules/stav-append-authority@v0.2.0`. Version `v0.1.0` remains immutable historical evidence for the pre-supervision boundary. Neither version is a GitHub binary release.

It installs immutable receipt-v2 executable packages, enrolls isolated per-TOPS instances, installs bounded per-TOPS launchd/systemd liveness profiles, mutually authenticates Darwin/Linux Unix-socket peers from kernel credentials, authorizes exact producer tuples and reader classifications, appends fsync-backed canonical events, reconstructs idempotency on restart, verifies the digest chain, preserves incomplete-tail evidence, and serves the read-only qxctl STAV interface. The installed executable also owns the bounded `foundation-lifecycle` JSON adapter for offline enrollment and supervisor observation, planning, apply status, mutation, and recovery.

Foundational mutation uses digest-linked attempts outside the purged STAV subtree. Ordinary audited apply currently fails closed because the closed SSIAG audit-producer composition is not yet reachable from this module adapter. Explicit `audit_deferred` apply retains reconciliation-required evidence; the module never marks that evidence reconciled without a later exact STAV receipt.

`knowledge/stav/` remains protocol truth. No producer, qxctl client, supervisor, or other caller edits the ledger through a supported interface. Start with `INTENT.md`, `MANIFEST.md`, `INSTALL.md`, and `SKILL.md`.
