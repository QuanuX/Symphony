# Accordare STAV Producer Installation

## Sequence

1. Install and enroll SSIAG and STAV for the target TOPS.
2. Build the producer with `CGO_ENABLED=0`.
3. Run `symphony-accordare-stav-producer install --prefix ABSOLUTE --version 0.1.0-dev`.
4. Enroll the per-TOPS producer with the exact STAV configuration path, producer UID/GID, and qxctl SSIAG subject ID/kind. User scope defaults UID/GID to the invoking process; system scope requires explicit values.
5. Stop the STAV append authority.
6. Observe the exact STAV configuration SHA-256 and invoke `qxctl stav accordare-grant install` with that expected digest and a stable operation ID.
7. Restart STAV, then start the producer with `serve --supervised` for system scope or the equivalent owner-managed user supervisor.
8. Run producer `status`; use `reconcile` until pending is zero.

Uninstall in reverse: stop the producer, reconcile to zero, stop STAV, remove the qxctl grant, restart STAV if desired, unenroll the producer, then remove the receipt-owned package. `unenroll --purge` refuses a non-empty outbox.

Native descriptor generation is not claimed by this initial producer package. An owner-controlled supervisor may invoke the immutable receipt-v2 executable; qxctl lifecycle integration must bind that exact installed identity before Symphony claims managed liveness.
