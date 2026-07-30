# Knowledge Session Coordinator Intent

## Purpose

Provide the domain-neutral, independently installable process that will coordinate authenticated knowledge sessions and separately locked worktree reconciliation contexts without absorbing vector semantics.

## Implemented Scope

Development version `0.1.0-dev` implements the user-scope reconciliation vertical slice:

- `inspect` reports the exact engine descriptor and disabled capabilities;
- `check` computes a bounded deterministic digest over an explicit relative-path set in the process working directory and optionally compares expected state;
- `compatibility` negotiates exact process, journal-format, and capability overlap with qxctl;
- `begin`, `status`, `checkpoint`, `close`, and `recover` manage a protected noncanonical worktree context through a dual-slot journal, atomic head, expected-state generations, and content-addressed snapshots;
- strict `symphony.knowledge.engine-process.v1` standard-input/output handling;
- versioned install receipt, isolated install paths, and receipt-owned uninstall proof.

## Deferred Scope

Authenticated-session lifecycle remains reserved. qxctl validates, binds, and invokes this exact inactive-undocked coordinator for reconciliation only. SSIAG authority-epoch binding, vector-engine invocation, observers/hooks, proposal serialization, canonical apply, STAV coordination, system/TOPS profiles, format migration beyond the current v1 compatibility window, and live Maestro docking remain unimplemented.

## Authority

The coordinator never decides vector meaning or caller authority. It mutates only protected noncanonical reconciliation state and cannot mutate canonical files or establish an authenticated session. Self-healing uses verifiable slot/head evidence and never guesses across incompatible state.
