# Knowledge Session Coordinator Intent

## Purpose

Provide the domain-neutral, independently installable process that will coordinate authenticated knowledge sessions and separately locked worktree reconciliation contexts without absorbing vector semantics.

## Implemented Scope

Development version `0.1.0-dev` implements user-scope reconciliation and authenticated-session vertical slices:

- `inspect` reports the exact engine descriptor and disabled capabilities;
- `check` computes a bounded deterministic digest over an explicit relative-path set in the process working directory and optionally compares expected state;
- `compatibility` negotiates exact process, journal-format, and capability overlap with qxctl;
- `begin`, `status`, `checkpoint`, `close`, and `recover` manage a protected noncanonical worktree context through a dual-slot journal, atomic head, expected-state generations, and content-addressed snapshots;
- `session_begin`, `session_status`, `session_checkpoint`, `session_close`, and `session_recover` validate exact SSIAG decision/capability evidence and manage a separate protected noncanonical authority epoch through the same evidence-preserving durability model;
- strict `symphony.knowledge.engine-process.v1` standard-input/output handling;
- versioned install receipt, isolated install paths, and receipt-owned uninstall proof.

## Deferred Scope

qxctl validates, binds, and invokes this exact inactive-undocked coordinator for reconciliation and authenticated-session operations. qxctl obtains every session-operation decision from the kernel-authenticated SSIAG Unix socket; the coordinator then independently checks the evidence's exact subject, TOPS, operation, resource, audience, scope, policy/configuration digests, binding digest, non-transferability, non-apply status, and expiry. Vector-engine invocation, observers/hooks, proposal serialization, canonical apply, direct coordinator-to-STAV coordination, system/TOPS binding profiles, format migration beyond the current v1 compatibility window, and live Maestro docking remain unimplemented.

## Authority

The coordinator never decides vector meaning or caller authority. SSIAG makes the authorization decision; the coordinator validates the supplied bounded evidence and mutates only protected noncanonical reconciliation or authenticated-session state. It cannot mutate canonical files. Self-healing uses verifiable slot/head and linked-epoch evidence and never guesses across incompatible state or manufactures authority.
