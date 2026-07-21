# Knowledge Session Coordinator Specification

## Status

Implemented read-only development slice, version `0.1.0-dev`. Not a published release and not an operational authenticated-session manager.

## Process Contract

The executable implements the exact common request/response envelope under `knowledge/schemas/v1/`. It reads one request from standard input and emits one compact response on standard output. `--help`, `--version`, and `--descriptor` are direct diagnostic modes and do not accept process input.

Stable exit statuses are:

| Status | Meaning |
|---:|---|
| `0` | successful operation |
| `2` | malformed, excessive, or invalid request/argument |
| `3` | protocol, target, or deadline mismatch |
| `4` | unsupported operation or invalid operation payload |
| `5` | bounded engine/path/internal failure |

Every process-mode error still attempts one safe protocol response. If response serialization itself fails, the executable exits `5` without emitting unbounded fallback text.

## `inspect`

Payload: exact empty object.

The result returns the descriptor, `read_only_foundation` readiness, and explicit false values for canonical apply, session mutation, and Maestro docking.

## `check`

Payload fields:

- `paths`: one to 1,024 unique safe relative regular-file paths;
- `expected_snapshot_digest`: `null` or an exact `sha256:` digest.

The operation roots access at the current working directory, rejects symlinks and special files, reads at most 4 MiB per file, sorts paths, and returns only path, size, content digest, aggregate snapshot digest, and an expected-state comparison. It never returns file content or writes state.

The direct process checks its deadline before and between file-read chunks. A future qxctl client must separately terminate the child at the same deadline; this slice does not claim hard cancellation of an operating-system call blocked below the process.

## Descriptor Truth

Only `inspect` and `check` are implemented. Session lifecycle operations are `reserved`; `apply` is `disabled`. The descriptor declares user-scope process invocation, C++26, freezing-path placement, `installed_undocked`, no default receptor, and no network listener.

## Install and Uninstall

Installation uses module-and-version-specific paths and creates no active alias. The receipt uses `prefix_mode: installation_prefix`, lists all owned relative files, and carries no host-specific secret or timestamp. The generated uninstall script removes those files only and refuses directory removal.

## Non-Authorization

This implementation does not authenticate a caller, establish or recover a session, create a mutable journal, take a writer lock, run a watcher, invoke a vector engine, call qxctl/SSIAG/STAV, mutate a repository, activate a version, or dock with Maestro.
