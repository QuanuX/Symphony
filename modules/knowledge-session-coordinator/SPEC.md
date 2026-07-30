# Knowledge Session Coordinator Specification

## Status

Implemented reconciliation development slice, version `0.1.0-dev`. Not a published release and not an operational authenticated-session manager.

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

The result returns the descriptor, `reconciliation_foundation` readiness, and explicit false values for canonical apply, authenticated-session mutation, and Maestro docking.

## `check`

Payload fields:

- `paths`: one to 1,024 unique safe relative regular-file paths;
- `expected_snapshot_digest`: `null` or an exact `sha256:` digest.

The operation roots access at the current working directory, rejects symlinks and special files, reads at most 4 MiB per file, sorts paths, and returns only path, size, content digest, aggregate snapshot digest, and an expected-state comparison. It never returns file content or writes state.

The direct process checks its deadline before and between file-read chunks. qxctl independently terminates the child at the same deadline plus a bounded process-reaping allowance.

## Reconciliation Operations

`compatibility`, `begin`, `status`, `checkpoint`, `close`, and `recover` use the exact command and result schemas under `knowledge/schemas/v1/`. qxctl supplies the protected absolute user state root, its own protocol/version/capability declaration, and operation-specific expected state. The process roots repository reads at its canonical current working directory.

`begin` requires a stable caller operation ID, one to 1,024 unique safe relative regular-file paths, and either `absent` or the exact digest of a closed prior journal. It creates a new open context and initial checkpoint. `checkpoint` and `close` require the exact current journal digest and stable operation ID. Replaying the same completed operation ID is an idempotent no-op; reusing it for different state fails closed. `status` performs no repair. `recover` requires an exact digest during ordinary recovery or an explicit `discover` state when the head cannot be trusted.

Each canonical worktree path maps to one protected context directory under:

```text
<state-root>/symphony/knowledge-session-coordinator/reconciliation/v1/contexts/<worktree-key>/
```

The directory contains `journal.lock`, `journal.0.json`, `journal.1.json`, and `head.json`. Reads and mutations serialize through a nonblocking no-follow lock. State roots and files must be owned by the effective user, managed directories are mode `0700`, managed files are mode `0600`, and group/other-writable or non-regular objects fail closed.

A write synchronizes the inactive journal slot, atomically replaces the head, then synchronizes the containing directory. Recovery validates all available digests and continuity, removes only private coordinator-named atomic-head temporaries left by interrupted commits, and reports every repair action. A unique valid slot one generation ahead of the head with a matching previous digest may be adopted. A damaged or stale head may be rebuilt from a valid slot. Otherwise recovery rolls forward from the highest unique valid slot and records the abandoned head digest. Divergent equally ranked slots, unknown critical extensions, stale expected state, unsupported versions, and ambiguous evidence require administrator review and remain unmodified.

## Compatibility Contract

The coordinator advertises process protocol v1, journal read/write version 1, and named capabilities for dual-slot durability, expected-state compare-and-swap, content snapshots, idempotent operation replay, extension preservation, and recovery. qxctl and the coordinator intersect declared versions and capabilities before reconciliation.

Unknown noncritical extension values and their declared payload digests are preserved exactly as parsed by every write. An implementation that does not recognize a critical extension or stored protocol version preserves the files and blocks both mutation and automated recovery; it cannot fall back to an older slot and call that repair. Newer implementations must continue writing an existing supported format until an explicit stepwise migration commits. No implementation may infer compatibility from semantic-version ordering, silently downgrade, drop unknown state, or perform a lossy conversion.

| Observed combination | Operational behavior |
|---|---|
| older/newer qxctl and coordinator share process v1, journal v1, and every required capability | full read/write operation; executable version order is irrelevant |
| journal v1 is readable but qxctl lacks its write version or any required capability | status/compatibility remain read-only; mutations fail before state changes |
| an engine binding is added, removed, upgraded, or rolled back while a context is open | the next exact-state checkpoint remains valid and records the new binding-registry and engine-inventory digests |
| a synchronized inactive slot is exactly one linked generation ahead of the head | ordinary or discovery recovery adopts it and commits a new recovery checkpoint |
| the head or one slot is damaged while another unique valid slot remains | explicit discovery recovery rebuilds the head and records its disposition |
| an inactive slot is equally ranked, unexpectedly ahead, newer-format, or contains an unknown critical extension | normal writes and automated downgrade stop; all evidence remains in place |
| a future implementation needs a new journal format | it first dual-reads the prior format, preserves that write format while old participants remain bound, then performs a separately contracted idempotent migration |

## Descriptor Truth

`inspect`, `check`, `compatibility`, `begin`, `status`, `checkpoint`, `close`, and `recover` are implemented. Authenticated-session lifecycle remains reserved and `apply` is disabled. The descriptor declares user-scope process invocation, C++26, freezing-path placement, `installed_undocked`, no default receptor, and no network listener.

## Install and Uninstall

Installation uses module-and-version-specific paths and creates no active alias. The receipt uses `prefix_mode: installation_prefix`, lists all owned relative files, and carries no host-specific secret or timestamp. qxctl may bind the exact validated receipt and executable digests in its separate user-default registry; that does not alter this receipt's inactive-undocked state. The generated uninstall script removes those files only and refuses directory removal.

## Non-Authorization

This implementation does not authenticate a caller, establish or recover an authority epoch, mutate canonical repository content, run a watcher, install a hook, invoke a vector engine, call SSIAG/STAV, activate an install receipt, or dock with Maestro. Reconciliation journal mutation is noncanonical local coordination only.
