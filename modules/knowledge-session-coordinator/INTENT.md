# Knowledge Session Coordinator Intent

## Purpose

Provide the domain-neutral, independently installable process that coordinates authenticated knowledge sessions, separately locked worktree reconciliation contexts, and bounded report-only cross-vector lifecycle planning without absorbing vector semantics.

## Implemented Scope

Development version `0.1.0-dev` implements user-scope reconciliation and authenticated-session vertical slices:

- `inspect` reports the exact engine descriptor and disabled capabilities;
- `check` computes a bounded deterministic digest over an explicit relative-path set in the process working directory and optionally compares expected state;
- `compatibility` negotiates exact process, journal-format, and capability overlap with qxctl;
- `begin`, `status`, `checkpoint`, `close`, and `recover` manage a protected noncanonical worktree context through a dual-slot journal, atomic head, expected-state generations, and content-addressed snapshots;
- `session_begin`, `session_status`, `session_checkpoint`, `session_close`, and `session_recover` validate exact SSIAG decision/capability evidence and manage a separate protected noncanonical authority epoch through the same evidence-preserving durability model;
- `lifecycle_plan` validates exact desired-state and caller-supplied observation evidence, negotiates receipt/protocol capabilities, and emits a deterministic dependency-ready-set plan with forward/inverse identities, localized blockers, exact receptor targets, and disabled apply;
- strict `symphony.knowledge.engine-process.v1` standard-input/output handling;
- versioned install receipt, isolated install paths, and receipt-owned uninstall proof.

## Deferred Scope

qxctl validates, binds, and invokes this exact inactive-undocked coordinator for reconciliation, authenticated-session, and report-only lifecycle operations. qxctl obtains each authorization decision from the kernel-authenticated SSIAG Unix socket; the coordinator independently validates authorization evidence for stateful session operations, while qxctl validates and binds lifecycle authorization before invoking the authority-free report operation. qxctl lifecycle profile administration and fixed-layout observation are implemented outside this module. Lifecycle journaling, action execution, vector-engine invocation, observers/hooks, proposal serialization, canonical apply, direct coordinator-to-STAV coordination, system/TOPS engine-binding profiles, format migration beyond the current v1 compatibility window, and live Maestro docking remain unimplemented.

qxctl may explicitly compose session primitives into an idempotent login, refresh, or logout transition. That composition is a qxctl responsibility; it adds no coordinator operation, watcher, boot service, or implicit recovery behavior. The report-only lifecycle planner accepts fully supplied, digest-bound desired and observed state; it does not scan configured roots, read protected profiles, persist a plan, or apply an action. Its observation key uses stable validated inventory rather than document collection time, so a timestamp-only refresh preserves the transaction and semantic action identities.

## Authority

The coordinator never decides vector meaning or caller authority. SSIAG makes the authorization decision; the coordinator validates the supplied bounded evidence and mutates only protected noncanonical reconciliation or authenticated-session state. It cannot mutate canonical files. Self-healing uses verifiable slot/head and linked-epoch evidence and never guesses across incompatible state or manufactures authority.
