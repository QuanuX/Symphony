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
- reconciliation command: `symphony.knowledge.reconciliation-command.v1`
- reconciliation journal: `symphony.knowledge.reconciliation-journal.v1`
- reconciliation head: `symphony.knowledge.reconciliation-head.v1`
- reconciliation result: `symphony.knowledge.reconciliation-result.v1`

## Implemented Operations

| Operation | State | Canonical mutation |
|---|---|---|
| `inspect` | implemented | no |
| `check` | implemented | no |
| `compatibility` | implemented | no |
| `begin`, `status`, `checkpoint`, `close`, `recover` | implemented; noncanonical local state only | no |
| `apply` | disabled | prohibited |

The implemented scope is user-process invocation and user-scope reconciliation state. Authenticated-session and system/TOPS provisioning are not claimed.

## Installability

The executable installs beneath a module-and-version-specific `libexec` path, with contracts, AGPL and third-party licenses, and a deterministic receipt. Installation leaves the module `installed_undocked`, creates no global executable alias, changes no binding, and does not contact Maestro. qxctl may select the exact receipt in its separate protected user-default binding registry without changing receipt state. Uninstall removes only receipt-owned files and preserves reconciliation evidence.

## Dependencies

The coordinator statically links `knowledge-vector-engine-cpp` and has no runtime shared-library, network, Python, Go, cgo, or provider dependency.

## Boundaries

There is no network listener, daemon, credential input, secret field, canonical write, hook, watcher, SSIAG decision, STAV append, vector-engine invocation, or Maestro dock in this version. qxctl invokes reconciliation synchronously through the exact selected receipt. Absolute repository/state paths remain only in protected local state and process payloads.
