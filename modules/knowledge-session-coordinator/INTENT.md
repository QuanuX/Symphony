# Knowledge Session Coordinator Intent

## Purpose

Provide the domain-neutral, independently installable process that coordinates authenticated knowledge sessions, persistent SSFV maintenance evidence, durable SAV Named Version selection evidence, separately locked worktree reconciliation contexts, bounded cross-vector lifecycle planning, protected report-only boot journals, and separately authorized apply-capable lifecycle journals without absorbing vector semantics or executing host actions itself.

## Implemented Scope

Development version `0.1.0-dev` implements user-scope reconciliation and authenticated-session vertical slices:

- `inspect` reports the exact engine descriptor and disabled capabilities;
- `check` computes a bounded deterministic digest over an explicit relative-path set in the process working directory and optionally compares expected state;
- `compatibility` negotiates exact process, journal-format, and capability overlap with qxctl;
- `begin`, `status`, `checkpoint`, `close`, and `recover` manage a protected noncanonical worktree context through a dual-slot journal, atomic head, expected-state generations, and content-addressed snapshots;
- `session_begin`, `session_status`, `session_checkpoint`, `session_close`, and `session_recover` validate exact SSIAG decision/capability evidence and manage a separate protected noncanonical authority epoch through the same evidence-preserving durability model;
- `ssfv_maintenance_begin`, `ssfv_maintenance_status`, `ssfv_maintenance_checkpoint`, `ssfv_maintenance_close`, and `ssfv_maintenance_recover` preserve a content-addressed semantic baseline and exact read-only SSFV, binding-registry, authority-session, and optional Maestro inventory lineage in a separate per-TOPS/subject/repository stream;
- `named_version_prepare`, `named_version_seal`, `named_version_alias`, `named_version_lookup`, `named_version_status`, and `named_version_recover` preserve SAV-validated immutable composition objects and noncanonical selectors in a protected per-TOPS dual-slot registry with exact SSIAG evidence, compare-and-swap, replay protection, and unique forward recovery;
- `lifecycle_plan` validates exact desired-state and caller-supplied observation evidence, negotiates receipt/protocol capabilities, and emits a deterministic dependency-ready-set plan with forward/inverse identities, localized blockers, exact receptor targets, and disabled apply;
- `lifecycle_boot`, `lifecycle_boot_status`, and `lifecycle_boot_recover` independently validate exact SSIAG evidence, recompute the stable inventory binding, and administer protected per-TOPS/profile dual-slot report-only journals with exact compare-and-swap, idempotent operation replay, timestamp-stable no-op detection, linked plan revisions, and unambiguous forward recovery;
- `lifecycle_apply_prepare`, `lifecycle_apply_finalize`, `lifecycle_apply_close`, `lifecycle_apply_status`, and `lifecycle_apply_recover` validate fresh exact SSIAG evidence and administer a separate apply-capable v2 stream that binds one exact report journal, serializes attempts before external mutation, verifies post-action observations, and selects content-addressed applied-state evidence only through a durable head commit;
- strict `symphony.knowledge.engine-process.v1` standard-input/output handling;
- versioned install receipt, isolated install paths, and receipt-owned uninstall proof.

## Deferred Scope

qxctl also binds the exact SAV and coordinator installations for the implemented Named Version persistence circuit. A closed safe-metadata STAV vocabulary is registered for that circuit, but its runtime producer and producer grant remain separately gated and unimplemented.

qxctl validates, binds, and invokes this exact inactive-undocked coordinator for reconciliation, authenticated-session, report-only lifecycle operations, and the explicit apply-compatible circuit. qxctl obtains each authorization decision from the kernel-authenticated SSIAG Unix socket; the coordinator independently validates authorization evidence for stateful session and lifecycle-journal operations. qxctl lifecycle profile administration, fixed-layout observation, and reviewed external action adapters—including exact established-role binding and Maestro presence mutation—remain outside this module. The coordinator never changes package files, qxctl runtime state, binding state, or Maestro presence; it records the action before qxctl acts, verifies complete after-evidence, and commits only protected noncanonical journals/applied evidence. A qxctl-managed coordinator handoff is out of place: the candidate must first prove it can read the prepared journal, and the current coordinator remains responsible for uninterrupted finalization. Receipt-v1 mutation, arbitrary entry-point execution, live service activation, vector-engine invocation, observers/hooks, proposal serialization, canonical apply, direct coordinator-to-STAV coordination, system/TOPS engine-binding profiles, automatic in-place format migration, host boot integration, coordinator-owned self-handoff, and in-place replacement remain unimplemented.

qxctl may explicitly compose session primitives into an idempotent login, refresh, or logout transition. That composition is a qxctl responsibility; it adds no coordinator operation, watcher, boot service, or implicit recovery behavior. Lifecycle planner, boot, and apply operations accept fully supplied, digest-bound desired and observed state; they do not scan configured roots or read protected profiles. Boot persists only report evidence. Apply requires the exact boot journal as source authority, persists a prepared attempt before qxctl executes an adapter, and accepts finalization only after a complete observation proves the transition. Both journal families bind stable validated inventory rather than document collection time, so a timestamp-only refresh does not create semantic churn.

## Authority

The coordinator never decides vector meaning or caller authority. SSIAG makes the authorization decision; the coordinator validates the supplied bounded evidence and mutates only protected noncanonical reconciliation, authenticated-session, SSFV-maintenance, lifecycle-journal, or applied-state files. An SSFV maintenance review state is evidence about change, never a decision that a feature is worthy, correct, or ratified. It cannot mutate canonical files or host package/runtime state. Self-healing uses verifiable slot/head, linked-epoch, and digest-linked lifecycle evidence and never guesses across incompatible state, invents successful execution, or manufactures authority.
