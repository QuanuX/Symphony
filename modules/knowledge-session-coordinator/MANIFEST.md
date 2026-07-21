# Knowledge Session Coordinator Manifest

## Identity

- module ID: `knowledge-session-coordinator`
- source path: `modules/knowledge-session-coordinator/`
- executable: `symphony-knowledge-session`
- language: C++26
- development version: `0.1.0-dev`
- thermal placement: administrative freezing path

## Protocols

- process: `symphony.knowledge.engine-process.v1`
- descriptor: `symphony.knowledge.engine-descriptor.v1`
- install receipt: `symphony.knowledge.install-receipt.v1`

## Implemented Operations

| Operation | State | Canonical mutation |
|---|---|---|
| `inspect` | implemented | no |
| `check` | implemented | no |
| `begin`, `status`, `checkpoint`, `close`, `recover` | reserved | no |
| `apply` | disabled | prohibited |

The implemented scope is user-process invocation. System/TOPS session provisioning is not yet claimed.

## Installability

The executable installs beneath a module-and-version-specific `libexec` path, with contracts, AGPL and third-party licenses, and a deterministic receipt. Installation leaves the module `installed_undocked`, creates no global executable alias, changes no active binding, and does not contact Maestro. Uninstall removes only receipt-owned files.

## Dependencies

The coordinator statically links `knowledge-vector-engine-cpp` and has no runtime shared-library, network, Python, Go, cgo, or provider dependency.

## Boundaries

There is no network listener, daemon, credential input, secret field, canonical write, worktree journal mutation, hook, watcher, qxctl command, SSIAG decision, STAV append, or Maestro dock in this version.
