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

Before removal, reconcile append work and resolve every intent while the producer is still available. Then remove producer supervision, stop STAV, remove the qxctl grant, restart STAV if desired, unenroll the producer, and remove the receipt-owned package using an exact retained copy of the native owner outside that installation. An executing controller cannot delete itself. Both forms of `unenroll` refuse non-empty intent or outbox state before deleting the recovery configuration. Unenrollment takes exclusive package-use ownership, so a live service or concurrent startup also blocks it.

The module generates independent per-TOPS launchd/systemd liveness descriptors for macOS and Linux. qxctl verifies the exact receipt-v2 package before invoking the installed executable's supervisor operation. Units pin the exact versioned binary, use bounded restart/shutdown, and contain no SSIAG/STAV startup dependency. Native Windows service engines are intentionally unsupported; Windows clients use WSL or a remote Linux TOPS node.

## Native retirement boundary

Fresh native installs publish only the executing installer's exact bytes and a receipt with `symphony.package-retirement.v1`. Existing unguarded packages cannot be adopted or removed through this contract. Install publication and final unlink use descriptors pinned to the selected installation root. Admission remains closed until both executable and receipt are durable. `serve`, enrollment, and supervisor installation/start require the exact installed executable and retain package-use ownership for their work. Installed stop/status/recovery controllers also retain use ownership during retiring without reopening startup admission. A service retains ownership until startup reconciliation, requests, and socket cleanup finish.

Removal first refuses known pending audit work. It then durably closes package admission and requires exclusive use ownership; a live process leaves removal blocked and new starts refused. After all participants drain, it rechecks enrollment and orphan intent/outbox state, verifies the exact receipt-owned bytes, removes the executable and receipt in that order, and retains the retirement tombstone. Exact retries can finish an interrupted receipt-last removal. The same retired identity cannot be reopened by install or a force flag.

The `.symphony-package-retirement/<owner>/<version>` state and `.symphony-lifecycle.lock` are local to the installation, bind root and lock identities, and must never enter a portable package, CI staging payload, source/SDK bundle, or installer image. Each recipient's native install creates its own state. Preserve these files during local recovery; copying or deleting them is not migration.

Qualification is limited to the current user's fresh, guarded native installation and participating native entry points. System/root reclamation, complete cross-user reference discovery, arbitrary loaders or legacy consumers, and qxctl-administered shared-root removal remain blocked. Updated receipt validation also requires source-bound Go client releases; published older clients are not claimed to support this contract. Raw development executables can inspect or control an existing service but cannot enroll, serve, or install its supervisor as a qualified package.

If pending work appears after admission closes and the existing producer subsequently dies, this increment retains configuration, receipt, intent, and outbox evidence but does not provide a qualified restart/offline intent recovery path. Do not delete or reopen its guard to bypass that state. Recovery/migration of that retained work remains a blocker before general distribution removal can be enabled. Native stop, read-only status, and reconciliation of an already-running producer remain available while retiring; no host service activation is implied by these source tests.
