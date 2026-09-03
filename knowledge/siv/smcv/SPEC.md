# Symphony Markdown Conversion Vector Specification

## Translation Boundary

SMCV sits above an enabled IPC connection. It may accept a bounded precise Markdown representation, map it to the exact JSON or other payload required by the receiving owner contract, and map the exact result back to bounded Markdown for the model.

The owner format remains authoritative. SMCV must preserve:

- protocol and schema identity;
- field presence, type, ordering semantics where meaningful, nullability, and bounds;
- exact values, versions, identifiers, digests, timestamps, and error distinctions;
- unknown critical-state rejection;
- lossless round-trip requirements established by the eventual mapping contract;
- the difference between transport success, parse success, semantic validity, authorization, and execution.

The desired fixed space or line discipline for model context remains a future grammar decision.

## qxctl Boundary

A future qxctl surface may enable, disable, and configure SMCV. qxctl does not perform domain translation by implication and cannot choose a mapping without the exact endpoint contract and version.

## Thermal Boundary

SMCV is an agentic or administrative adapter. It must not become a synchronous dependency of trading hot-path execution merely because another component uses IPC.

## Non-Authorization Statement

SMCV cannot fill missing fields, guess mappings, change intent, reinterpret an incompatible version, expose secrets, or cause a converted payload to execute.
