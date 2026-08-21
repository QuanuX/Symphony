# Accordare STAV Producer Installation

## Sequence

1. Install and enroll SSIAG and STAV for the target TOPS.
2. Build the producer with `CGO_ENABLED=0`.
3. Run `symphony-accordare-stav-producer install --prefix ABSOLUTE --version 0.1.0-dev`.
4. Enroll the per-TOPS producer with the exact STAV configuration path, producer UID/GID, and qxctl SSIAG subject ID/kind. User scope defaults UID/GID to the invoking process; system scope requires explicit values.
5. Stop the STAV append authority.
6. Observe the exact STAV configuration SHA-256 and invoke `qxctl stav accordare-grant install` with that expected digest and a stable operation ID.
7. Restart STAV, then run `qxctl stav accordare supervisor-install --prefix ABSOLUTE --version 0.1.0-dev --tops-id UUID --scope user|system`. `--no-start` writes the deterministic descriptor for an owner-provided equivalent supervisor.
8. Run `qxctl stav accordare status`; use `reconcile` until `append_pending` is zero. Any `intent_pending` item requires retry of its exact original Named Version command.

Uninstall in reverse: remove producer supervision, reconcile append work and resolve every intent, stop STAV, remove the qxctl grant, restart STAV if desired, unenroll the producer, then remove the receipt-owned package. `unenroll --purge` refuses non-empty intent or outbox state.

The module generates independent per-TOPS launchd/systemd liveness descriptors for macOS and Linux. qxctl verifies the exact receipt-v2 package before invoking the installed executable's supervisor operation. Units pin the exact versioned binary, use bounded restart/shutdown, and contain no SSIAG/STAV startup dependency. Native Windows service engines are intentionally unsupported; Windows clients use WSL or a remote Linux TOPS node.
