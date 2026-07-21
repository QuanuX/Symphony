# Knowledge Session Coordinator Intent

## Purpose

Provide the domain-neutral, independently installable process that will coordinate authenticated knowledge sessions and separately locked worktree reconciliation contexts without absorbing vector semantics.

## Implemented Scope

Development version `0.1.0-dev` implements the first read-only vertical slice:

- `inspect` reports the exact engine descriptor and disabled capabilities;
- `check` computes a bounded deterministic digest over an explicit relative-path set in the process working directory and optionally compares expected state;
- strict `symphony.knowledge.engine-process.v1` standard-input/output handling;
- versioned install receipt, isolated install paths, and receipt-owned uninstall proof.

## Deferred Scope

Session `begin`, `status`, `checkpoint`, `close`, and `recover` are descriptor-visible but reserved. Authenticated authority binding, mutable worktree journals, locks, observers, qxctl integration, SSIAG/STAV coordination, proposal serialization, apply, and live Maestro docking remain unimplemented.

## Authority

The coordinator never decides vector meaning or caller authority. Its current operations are read-only for every caller and cannot mutate canonical files or establish a session.
