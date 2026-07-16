# STAV Append Authority Specification

## Status

Owner-ratified namespace scaffold. Operational append behavior is not enabled.

## Namespace

- executable: `symphony-stav-append-authority`;
- environment prefix reservation: `SYMPHONY_STAV_`;
- user install target: `${HOME}/.local/bin/symphony-stav-append-authority`;
- system install target: `/usr/local/bin/symphony-stav-append-authority`.

For canonical lowercase TOPS ID `<tops-id>`:

| Surface | User scope | System scope |
|---|---|---|
| configuration | `${XDG_CONFIG_HOME:-${HOME}/.config}/symphony/<tops-id>/stav/append-authority.json` | `/etc/symphony/<tops-id>/stav/append-authority.json` |
| state | `${XDG_STATE_HOME:-${HOME}/.local/state}/symphony/<tops-id>/stav/` | `/var/lib/symphony/<tops-id>/stav/` |
| socket | `${XDG_RUNTIME_DIR}/symphony/<tops-id>/stav/append.sock` | `/run/symphony/<tops-id>/stav/append.sock` |
| socket fallback | `${XDG_STATE_HOME:-${HOME}/.local/state}/symphony/<tops-id>/stav/run/append.sock` | not applicable |

Path resolution is pure in this increment: it creates none of these per-TOPS surfaces.

## Binary Lifecycle

Install copies the invoking regular executable atomically to the selected target with executable permissions. It rejects non-regular targets and differing binaries unless `--force` is explicit. Uninstall removes only the selected regular executable, is idempotent when absent, and rejects a digest mismatch unless `--force` is explicit.

No installation manifest is emitted because no such schema has been ratified. No directory is removed during uninstall.

## Protocol Dependency

Canonical candidate, event, receipt, query, query-page, and verification schemas are owned by `knowledge/stav/`. Their pure-Go mechanics are implemented by `libraries/stav-protocol-go`. This module currently imports only shared TOPS-ID validation and does not decode candidates or emit events or receipts.

`symphony.stav.append-authority.config.v1`, `symphony.stav.append-authority.status.v1`, `symphony.stav.local.request.v1`, and `symphony.stav.local.response.v1` remain name reservations only.

## Fail-Closed Command Surface

The executable accepts lifecycle/help/version commands only. Any operational command is rejected. It has no raw append surface.
