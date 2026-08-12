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
- install receipt: `symphony.knowledge.install-receipt.v2`
- reconciliation command: `symphony.knowledge.reconciliation-command.v1`
- reconciliation journal: `symphony.knowledge.reconciliation-journal.v1`
- reconciliation head: `symphony.knowledge.reconciliation-head.v1`
- reconciliation result: `symphony.knowledge.reconciliation-result.v1`
- authenticated-session command: `symphony.knowledge.session-command.v1`
- authenticated-session journal: `symphony.knowledge.session-journal.v1`
- authenticated-session head: `symphony.knowledge.session-head.v1`
- authenticated-session result: `symphony.knowledge.session-result.v1`
- SSFV maintenance command/result: `symphony.knowledge.ssfv-maintenance-command.v1` and `symphony.knowledge.ssfv-maintenance-result.v1`
- SSFV maintenance journal/head: `symphony.knowledge.ssfv-maintenance-journal.v1` and `symphony.knowledge.ssfv-maintenance-head.v1`
- report-only lifecycle command: `symphony.knowledge.lifecycle-plan-command.v1`
- durable lifecycle command/result: `symphony.knowledge.lifecycle-boot-command.v1` and `symphony.knowledge.lifecycle-boot-result.v1`
- durable lifecycle journal/head: `symphony.knowledge.lifecycle-boot-journal.v1` and `symphony.knowledge.lifecycle-boot-head.v1`
- apply lifecycle command/result: `symphony.knowledge.lifecycle-apply-command.v1` and `symphony.knowledge.lifecycle-apply-result.v1`
- apply lifecycle journal/head: `symphony.knowledge.lifecycle-boot-journal.v2` and `symphony.knowledge.lifecycle-boot-head.v2`
- applied evidence: `symphony.knowledge.lifecycle-applied-state.v1`
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
| `ssfv_maintenance_begin`, `ssfv_maintenance_status`, `ssfv_maintenance_checkpoint`, `ssfv_maintenance_close`, `ssfv_maintenance_recover` | implemented; SSIAG-authorized noncanonical semantic-baseline and review state only | no |
| `lifecycle_plan` | implemented; deterministic report-only result only | no |
| `lifecycle_boot`, `lifecycle_boot_status`, `lifecycle_boot_recover` | implemented; SSIAG-authorized protected report-only journal state | no |
| `lifecycle_apply_prepare`, `lifecycle_apply_finalize`, `lifecycle_apply_close` | implemented; SSIAG-authorized protected attempt/applied-state coordination only | no |
| `lifecycle_apply_status`, `lifecycle_apply_recover` | implemented; read-only inspection or evidence-bounded v2 journal repair | no |
| canonical `apply` | disabled | prohibited |

The implemented module scope is user-process invocation, user-scope reconciliation, authenticated-session state, persistent SSFV maintenance evidence, caller-supplied lifecycle planning, protected report/apply journal persistence/recovery, attempt serialization, verified closure, and content-addressed applied-state commitment. qxctl performs desired-profile, configured-root observation, read-only SSFV/Maestro collection, and external package/runtime action responsibilities; those are not coordinator-owned filesystem operations. System/TOPS engine-binding profiles, host provisioning, arbitrary executable invocation, and canonical apply are not claimed.

## Installability

The executable installs beneath a module-and-version-specific `libexec` path, with contracts, AGPL and third-party licenses, and a deterministic receipt. Installation is initially observed as `installed_undocked`, creates no global executable alias, changes no binding, and does not contact Maestro. qxctl may select the exact receipt in its separate protected user-default binding registry without writing mutable lifecycle state into the receipt. Uninstall removes only receipt-owned files and preserves reconciliation, authenticated-session, SSFV-maintenance, lifecycle-journal, and desired-profile evidence.

## Dependencies

The coordinator statically links `knowledge-vector-engine-cpp` and has no runtime shared-library, network, Python, Go, cgo, or provider dependency.

## Boundaries

There is no network listener, daemon, credential input, secret field, canonical write, hook, watcher, direct SSIAG/STAV call, vector-engine invocation, lifecycle receipt discovery, host action execution, or Maestro dock in this version. qxctl obtains SSIAG decisions, invokes SSFV and optional Maestro separately, and supplies their validated read-only evidence when it invokes the coordinator. The coordinator never treats that evidence as canonical feature truth. Applied-state writes are protected noncanonical evidence and occur only after exact after-observation verification. Absolute repository/state paths remain only in protected local state and process payloads.
