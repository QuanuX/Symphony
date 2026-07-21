# SCLV Engine Intent

## Purpose

Implement the canonical SCLV contract as one independently installable C++26, freezing-path, proposal-only process module.

## Implemented Operations

- `inspect` reports exact engine, v3, adapter, and disabled-authority state.
- `check` validates the append-only v1/v2/v3 ledger without modifying it.
- `propose` validates caller-declared v3 content and provider evidence and returns one immutable append proposal.
- `recover` reconciles caller-supplied ephemeral journal evidence into resume, abandonment, no-op, or late-recovery proposal output without journal mutation.
- `project` returns a disposable provider-neutral JSON view.

Two separately discoverable package executables normalize local-Git revision evidence and air-gapped declarations. They supply bounded evidence only.

## Non-Authorization

The engine and adapters do not authenticate, grant permission, establish legal capacity, ratify, append, commit, edit history, delete journals, publish, call STAV/SSIAG, listen on a network endpoint, activate a version, or dock with Maestro.
