# SACV v1 Schema Manifest

This directory owns the machine-readable JSON Schema contracts for the initial Symphony API Contract Vector engine.

| Schema | Purpose |
|---|---|
| `registry-entry.schema.json` | Normalized canonical registry entry |
| `check-result.schema.json` | Read-only registry and owner-contract validation result |
| `diff-input.schema.json` | Caller-declared bounded comparison input |
| `diff-result.schema.json` | Deterministic compatibility evidence |
| `proposal-input.schema.json` | Caller-declared register or replace proposal input |
| `projection.schema.json` | Disposable registry inventory projection |

These schemas govern JSON engine payloads and results. They do not create an HTTP endpoint, authorize canonical apply, or make a generated projection authoritative.
