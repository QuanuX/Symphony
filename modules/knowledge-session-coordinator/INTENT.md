# Knowledge Session Coordinator Intent

## Purpose

Provide the domain-neutral, independently installable process that coordinates authenticated knowledge sessions, separately locked worktree reconciliation contexts, bounded report-only cross-vector lifecycle planning, and protected noncanonical lifecycle boot journals without absorbing vector semantics.

## Implemented Scope

Development version `0.1.0-dev` implements user-scope reconciliation and authenticated-session vertical slices:

- `inspect` reports the exact engine descriptor and disabled capabilities;
- `check` computes a bounded deterministic digest over an explicit relative-path set in the process working directory and optionally compares expected state;
- `compatibility` negotiates exact process, journal-format, and capability overlap with qxctl;
- `begin`, `status`, `checkpoint`, `close`, and `recover` manage a protected noncanonical worktree context through a dual-slot journal, atomic head, expected-state generations, and content-addressed snapshots;
- `session_begin`, `session_status`, `session_checkpoint`, `session_close`, and `session_recover` validate exact SSIAG decision/capability evidence and manage a separate protected noncanonical authority epoch through the same evidence-preserving durability model;
- `lifecycle_plan` validates exact desired-state and caller-supplied observation evidence, negotiates receipt/protocol capabilities, and emits a deterministic dependency-ready-set plan with forward/inverse identities, localized blockers, exact receptor targets, and disabled apply;
- `lifecycle_boot`, `lifecycle_boot_status`, and `lifecycle_boot_recover` independently validate exact SSIAG evidence, recompute the stable inventory binding, and administer protected per-TOPS/profile dual-slot report-only journals with exact compare-and-swap, idempotent operation replay, timestamp-stable no-op detection, linked plan revisions, and unambiguous forward recovery;
- strict `symphony.knowledge.engine-process.v1` standard-input/output handling;
- versioned install receipt, isolated install paths, and receipt-owned uninstall proof.

## Deferred Scope

qxctl validates, binds, and invokes this exact inactive-undocked coordinator for reconciliation, authenticated-session, and report-only lifecycle operations. qxctl obtains each authorization decision from the kernel-authenticated SSIAG Unix socket; the coordinator independently validates authorization evidence for stateful session and lifecycle-journal operations. qxctl lifecycle profile administration and fixed-layout observation are implemented outside this module. Lifecycle action execution, applied-state persistence, vector-engine invocation, observers/hooks, proposal serialization, canonical apply, direct coordinator-to-STAV coordination, system/TOPS engine-binding profiles, format migration beyond the current v1 compatibility window, host boot integration, and live Maestro docking remain unimplemented.

qxctl may explicitly compose session primitives into an idempotent login, refresh, or logout transition. That composition is a qxctl responsibility; it adds no coordinator operation, watcher, boot service, or implicit recovery behavior. The report-only lifecycle planner and durable boot operation accept fully supplied, digest-bound desired and observed state; they do not scan configured roots, read protected profiles, or apply an action. The boot operation persists only its bounded noncanonical journal and current plan identity. Its journal binds stable validated inventory rather than document collection time, so a timestamp-only refresh does not create a generation or plan revision.

## Authority

The coordinator never decides vector meaning or caller authority. SSIAG makes the authorization decision; the coordinator validates the supplied bounded evidence and mutates only protected noncanonical reconciliation, authenticated-session, or lifecycle-journal state. It cannot mutate canonical files. Self-healing uses verifiable slot/head, linked-epoch, and digest-linked lifecycle evidence and never guesses across incompatible state or manufactures authority.
