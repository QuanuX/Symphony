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
- authenticated-session command: `symphony.knowledge.session-command.v1`
- authenticated-session journal: `symphony.knowledge.session-journal.v1`
- authenticated-session head: `symphony.knowledge.session-head.v1`
- authenticated-session result: `symphony.knowledge.session-result.v1`
- report-only lifecycle command: `symphony.knowledge.lifecycle-plan-command.v1`
- desired/observation/plan evidence: `symphony.knowledge.lifecycle-desired-state.v1`, `symphony.knowledge.lifecycle-observation.v1`, and `symphony.knowledge.lifecycle-plan.v1`
- SSIAG authorization evidence: `symphony.ssiag.authorization-decision.v1` and `symphony.ssiag.capability.v1`

## Implemented Operations

| Operation | State | Canonical mutation |
|---|---|---|
| `inspect` | implemented | no |
| `check` | implemented | no |
| `compatibility` | implemented | no |
| `begin`, `status`, `checkpoint`, `close`, `recover` | implemented; noncanonical local state only | no |
| `session_begin`, `session_status`, `session_checkpoint`, `session_close`, `session_recover` | implemented; SSIAG-authorized noncanonical authority-epoch state only | no |
| `lifecycle_plan` | implemented; deterministic report-only result only | no |
| `apply` | disabled | prohibited |

The implemented scope is user-process invocation, user-scope reconciliation, authenticated-session state, and caller-supplied report-only lifecycle planning. System/TOPS binding profiles, configured-root collection, desired-profile administration, lifecycle persistence, and provisioning are not claimed.

## Installability

The executable installs beneath a module-and-version-specific `libexec` path, with contracts, AGPL and third-party licenses, and a deterministic receipt. Installation leaves the module `installed_undocked`, creates no global executable alias, changes no binding, and does not contact Maestro. qxctl may select the exact receipt in its separate protected user-default binding registry without changing receipt state. Uninstall removes only receipt-owned files and preserves reconciliation and authenticated-session evidence.

## Dependencies

The coordinator statically links `knowledge-vector-engine-cpp` and has no runtime shared-library, network, Python, Go, cgo, or provider dependency.

## Boundaries

There is no network listener, daemon, credential input, secret field, canonical write, hook, watcher, direct SSIAG/STAV call, vector-engine invocation, lifecycle filesystem discovery, action execution, or Maestro dock in this version. qxctl obtains an SSIAG decision and invokes reconciliation or session coordination synchronously through the exact selected receipt. The lifecycle operation is a direct report-only process surface until qxctl integration is separately implemented. Absolute repository/state paths remain only in protected local state and process payloads.
