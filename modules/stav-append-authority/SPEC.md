# STAV Append Authority Specification

## Status

Architect-ratified operational v1 implementation of `knowledge/stav/SPEC.md`.

## Namespace

For canonical TOPS ID `<tops-id>`:

| Surface | User scope | System scope |
|---|---|---|
| configuration | `${XDG_CONFIG_HOME:-${HOME}/.config}/symphony/<tops-id>/stav/append-authority.json` | `/etc/symphony/<tops-id>/stav/append-authority.json` |
| state | `${XDG_STATE_HOME:-${HOME}/.local/state}/symphony/<tops-id>/stav/` | `/var/lib/symphony/<tops-id>/stav/` |
| ledger | `<state>/ledger-v1.stavlog` | `<state>/ledger-v1.stavlog` |
| recovery evidence | `<state>/recovery/` | `<state>/recovery/` |
| socket | `${XDG_RUNTIME_DIR}/symphony/<tops-id>/stav/append.sock` | Linux: `/run/symphony/<tops-id>/stav/append.sock`; macOS: `/var/run/symphony/<tops-id>/stav/append.sock` |
| socket fallback | `<state>/run/append.sock` | not applicable |

## Lifecycle

`install` and `uninstall` own only the host executable. `enroll` writes a strict per-TOPS configuration and enrollment marker with an explicit authority identity and empty producer/reader arrays. `unenroll` removes only the marker by default. `unenroll --purge` refuses an active listener and removes only the selected TOPS configuration and state.

`supervisor install` and `supervisor uninstall` own only the selected per-TOPS launchd/systemd descriptor and its liveness registration. They preserve configuration, ledgers, and recovery evidence. System identities are pre-provisioned by the owner and must match the authority UID/GID in configuration; supervisor installation creates no principal and grants no STAV permission.

## Authentication and Authorization

The authority verifies its configured effective UID/GID before listening. Every accepted Darwin/Linux Unix connection is authenticated from kernel peer credentials. Append requires an exact producer grant and permission tuple; status, verify, and query require an exact reader grant. Clients verify the connected authority UID/GID before sending. Socket ownership and modes remain defense in depth.

After identity verification, the authority exclusively locks the persistent adjacent `append.sock.lock` before inspecting, removing, or binding `append.sock`. It rejects live/foreign endpoints, accepts only the defined stale failure cases, drains bounded accepted work on SIGTERM, removes the socket, and releases the lock last. Supervisor socket activation is prohibited.

## Durability

The authority holds a non-blocking exclusive file lock, verifies the entire ledger before readiness, writes one `length || canonical event || checksum` frame, calls file sync, then returns a committed receipt. It reconstructs request-ID idempotency from events. Only an incomplete final frame is preserved as evidence and truncated automatically; complete corruption prevents startup.

## Command Surface

- `install`, `uninstall`;
- `supervisor install`, `supervisor uninstall`;
- `enroll`, `unenroll`;
- `serve`;
- `help`, `--version`.

There is no raw append, repair, truncate, rotate, export, HTTP, or remote command. Producers use the authenticated typed client; readers use qxctl.
